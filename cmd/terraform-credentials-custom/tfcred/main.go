package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/AlfredoBall/terraform-credentials-custom/internal/store"
	"github.com/AlfredoBall/terraform-credentials-custom/internal/tfcontext"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

var supportedDomains = []string{"app.terraform.io", "app.eu.terraform.io"}

var version = "dev"

func shouldShowHelp(args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func versionString(v string) string {
	if strings.TrimSpace(v) == "" {
		return "dev"
	}
	return strings.TrimSpace(v)
}

func main() {
	if len(os.Args) < 2 || shouldShowHelp(os.Args[1:]) {
		printHelp()
		if len(os.Args) < 2 {
			fmt.Printf("%s[tfcred][error] missing command%s\n", colorRed, colorReset)
			os.Exit(1)
		}
		return
	}

	switch os.Args[1] {
	case "version":
		fmt.Println(versionString(version))

	case "init":
		domain := parseDomainFlag("", os.Args[2:])
		if domain == "" {
			domain = promptDefaultDomain()
		}
		store.Init(domain)
		fmt.Printf("%s[tfcred] initialized with default domain %s%s\n", colorGreen, domain, colorReset)

	case "config":
		configCmd := flag.NewFlagSet("config", flag.ExitOnError)
		defaultDomain := configCmd.String("default-domain", "", "set the default Terraform domain")
		show := configCmd.Bool("show", false, "show current configuration")
		_ = configCmd.Parse(os.Args[2:])

		if *show || *defaultDomain == "" {
			f := store.Load()
			fmt.Printf("default_domain=%s\n", f.DefaultDomain)
			return
		}

		if !isSupportedDomain(*defaultDomain) {
			fmt.Printf("%s[tfcred][error] unsupported domain: %s%s\n", colorRed, *defaultDomain, colorReset)
			os.Exit(1)
		}
		store.SetDefaultDomain(*defaultDomain)
		fmt.Printf("%s[tfcred] default domain set to %s%s\n", colorGreen, *defaultDomain, colorReset)

	case "add":
		addCmd := flag.NewFlagSet("add", flag.ExitOnError)
		ctx := addCmd.String("context", "", "context name")
		org := addCmd.String("org", "", "organization")
		tokenType := addCmd.String("token-type", "user", "user|team|org|default")
		domain := addCmd.String("domain", "", "Terraform domain to use for this context")
		token := addCmd.String("token", "", "optional token")
		_ = addCmd.Parse(os.Args[2:])

		if *ctx == "" {
			fmt.Printf("%s[tfcred][error] --context is required%s\n", colorRed, colorReset)
			os.Exit(1)
		}
		if *tokenType != "default" && *org == "" {
			fmt.Printf("%s[tfcred][error] --org is required for non-default contexts%s\n", colorRed, colorReset)
			os.Exit(1)
		}
		if *domain != "" && !isSupportedDomain(*domain) {
			fmt.Printf("%s[tfcred][error] unsupported domain: %s%s\n", colorRed, *domain, colorReset)
			os.Exit(1)
		}
		config := store.Load()
		if *domain == "" {
			if config.DefaultDomain == "" {
				fmt.Printf("%s[tfcred][error] no default domain configured; pass --domain%s\n", colorRed, colorReset)
				os.Exit(1)
			}
			*domain = config.DefaultDomain
		}
		if *token != "" && !isValidTokenFormat(*token) {
			fmt.Printf("%s[tfcred][error] invalid token format; expected [a-zA-Z0-9]{14}\\.atlasv1\\.[a-zA-Z0-9]{30,70}%s\n", colorRed, colorReset)
			os.Exit(1)
		}
		if existing, exists := config.Contexts[*ctx]; exists {
			if existing.Org == *org && existing.TokenType == *tokenType && existing.Domain == *domain && existing.TokenType != "default" {
				fmt.Printf("%s[tfcred][warning] context '%s' already exists%s\n", colorYellow, *ctx, colorReset)
				fmt.Print("Overwrite it? [y/N]: ")
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if !strings.EqualFold(confirm, "y") && !strings.EqualFold(confirm, "yes") {
					fmt.Println("[tfcred] Aborted.")
					return
				}
			}
		}
		store.Add(*ctx, *org, *tokenType, *domain, *token)

	case "remove":
		removeCmd := flag.NewFlagSet("remove", flag.ExitOnError)
		_ = removeCmd.Parse(os.Args[2:])

		removeArgs := removeCmd.Args()
		if len(removeArgs) < 1 {
			fmt.Printf("%s[tfcred][error] usage: tfcred remove <context>%s\n", colorRed, colorReset)
			os.Exit(1)
		}

		ctxName := removeArgs[0]
		if ctxName == "default" {
			fmt.Printf("%s[tfcred][error] the 'default' context is a core system fallback and cannot be removed%s\n", colorRed, colorReset)
			os.Exit(1)
		}

		_, found := store.Remove(ctxName)
		if !found {
			fmt.Printf("%s[tfcred][error] unknown context: %s%s\n", colorRed, ctxName, colorReset)
			os.Exit(1)
		}

		fmt.Printf("%s[tfcred]%s successfully removed context '%s' from metadata.%s\n", colorGreen, colorReset, ctxName, colorReset)
		fmt.Println("[note] The associated token remains stored in Windows registry and current session. It is no longer managed by tfcred.")
		fmt.Println("       Use 'tfcred add <context>' with the same context name to re-associate the token with a new context.")

	case "purge":
		purgeCmd := flag.NewFlagSet("purge", flag.ExitOnError)
		domain := purgeCmd.String("domain", "", "purge all tokens for a specific domain (e.g. app.terraform.io)")
		all := purgeCmd.Bool("all", false, "purge all tokens across all domains")
		_ = purgeCmd.Parse(os.Args[2:])

		purgeArgs := purgeCmd.Args()

		if *all {
			removed := store.PurgeAll()
			purgeEnvVars(removed)
			fmt.Printf("%s[tfcred] purged all contexts and env vars%s\n", colorGreen, colorReset)
			return
		}

		if *domain != "" {
			if !isSupportedDomain(*domain) {
				fmt.Printf("%s[tfcred][error] unsupported domain: %s%s\n", colorRed, *domain, colorReset)
				os.Exit(1)
			}
			removed := store.PurgeDomain(*domain)
			purgeEnvVars(removed)
			fmt.Printf("%s[tfcred] purged domain %s entries%s\n", colorGreen, *domain, colorReset)
			return
		}

		if len(purgeArgs) < 1 {
			fmt.Printf("%s[tfcred][error] usage: tfcred purge <context> [--domain <domain>] [--all]%s\n", colorRed, colorReset)
			os.Exit(1)
		}

		ctxName := purgeArgs[0]
		if ctxName == "default" {
			fmt.Printf("%s[tfcred][error] cannot purge the 'default' context%s\n", colorRed, colorReset)
			os.Exit(1)
		}

		entry, found := store.Remove(ctxName)
		if !found {
			fmt.Printf("%s[tfcred][error] unknown context: %s%s\n", colorRed, ctxName, colorReset)
			os.Exit(1)
		}

		removed := store.EntryEnvNames(entry)
		purgeEnvVars(removed)
		fmt.Printf("%s[tfcred] purged context %s and its token%s\n", colorGreen, ctxName, colorReset)

	case "list":
		store.List()

	case "switch":
		if len(os.Args) < 3 {
			fmt.Printf("%susage: tfcred switch <context>%s\n", colorRed, colorReset)
			os.Exit(1)
		}
		ctxName := os.Args[2]
		f := store.Load()
		entry, ok := f.Contexts[ctxName]
		if !ok {
			fmt.Printf("%s[tfcred][error] unknown context: %s%s\n", colorRed, ctxName, colorReset)
			os.Exit(1)
		}
		var tfContextStr string
		if entry.TokenType == "default" {
			tfContextStr = "default"
		} else {
			tfContextStr = fmt.Sprintf("%s:%s", entry.TokenType, entry.Org)
		}
		fmt.Println(tfContextStr)

	case "status":
		ctx := os.Getenv("TF_CONTEXT")
		p, _ := tfcontext.Parse(ctx)
		if p.Type == "default" {
			fmt.Println("[tfcred] context=default")
			return
		}
		fmt.Printf("[tfcred] type=%s org=%s\n", p.Type, p.Org)

	case "current":
		ctx := os.Getenv("TF_CONTEXT")
		if ctx == "" {
			ctx = "default"
		}
		fmt.Println(ctx)

	case "env":
		verbose := false
		showSecret := false
		all := false
		for _, arg := range os.Args[2:] {
			if arg == "--json" {
				verbose = true
			}
			if arg == "--show-secret" {
				showSecret = true
			}
			if arg == "--all" {
				all = true
			}
		}
		ctx := os.Getenv("TF_CONTEXT")
		p, err := tfcontext.Parse(ctx)
		if ctx == "" {
			reports := buildContextEnvReports(showSecret, store.Load().Contexts)
			if verbose {
				out := map[string]any{"TF_CONTEXT": ctx, "contexts": reports}
				enc, _ := json.MarshalIndent(out, "", " ")
				fmt.Println(string(enc))
			} else {
				for _, report := range reports {
					printOrderedReport(report)
					fmt.Println()
				}
			}
			return
		}

		result := map[string]string{
			"TF_CONTEXT": ctx,
		}
		if err != nil {
			result["error"] = err.Error()
		}
		if p.Type == "default" {
			result["TF_TOKEN_env"] = ""
			result["TF_TOKEN_value"] = ""
			result["TF_TOKEN_value_present"] = fmt.Sprintf("%v", false)
		} else {
			_, entry, ok := store.FindByTypeOrg(p.Type, p.Org)
			if !ok {
				fmt.Printf("%s[tfcred][error] unknown context in current TF_CONTEXT%s\n", colorRed, colorReset)
				os.Exit(1)
			}
			envs := store.EntryEnvNames(entry)
			if all {
				if verbose {
					result["TF_TOKEN_envs"] = strings.Join(envs, ",")
					result["TF_TOKEN_values"] = strings.Join(renderMaskedValues(envs, showSecret), ",")
					result["TF_TOKEN_value_present"] = fmt.Sprintf("%v", anyTokenPresent(envs))
				} else {
					for _, env := range envs {
						value := os.Getenv(env)
						fmt.Printf("%s=%s\n", env, redactSecret(value, showSecret))
					}
					fmt.Printf("TF_CONTEXT=%s\n", ctx)
					fmt.Printf("TF_TOKEN_value_present=%v\n", anyTokenPresent(envs))
				}
			} else {
				if verbose {
					result["TF_TOKEN_env"] = envs[0]
					result["TF_TOKEN_value"] = redactSecret(os.Getenv(envs[0]), showSecret)
					result["TF_TOKEN_value_present"] = fmt.Sprintf("%v", os.Getenv(envs[0]) != "")
				} else {
					fmt.Printf("TF_TOKEN_env=%s\n", envs[0])
					fmt.Printf("TF_TOKEN_value=%s\n", redactSecret(os.Getenv(envs[0]), showSecret))
					fmt.Printf("TF_TOKEN_value_present=%v\n", os.Getenv(envs[0]) != "")
				}
			}
			return
		}
		if verbose {
			enc, _ := json.MarshalIndent(result, "", " ")
			fmt.Println(string(enc))
		} else {
			printOrderedFields(result)
		}

	case "doctor":
		fmt.Println("[tfcred doctor] Beginning system diagnostics...")

		ctxRaw := os.Getenv("TF_CONTEXT")
		_, err := tfcontext.Parse(ctxRaw)
		if err != nil {
			fmt.Printf("   %s[✕] Error: Invalid TF_CONTEXT formatting pattern: %v%s\n", colorRed, err, colorReset)
		} else {
			fmt.Printf("   %s[✓]%s Context Structure: OK\n", colorGreen, colorReset)
		}

		f := store.Load()
		if f.Contexts == nil {
			fmt.Printf("   %s[✕] Error: contexts.json file missing or unreadable.%s\n", colorRed, colorReset)
		} else {
			fmt.Printf("   %s[✓]%s Configuration Storage: OK\n", colorGreen, colorReset)
		}

		for name, entry := range f.Contexts {
			if name == "default" {
				continue
			}
			envs := store.EntryEnvNames(entry)
			for _, env := range envs {
				value := os.Getenv(env)
				if value != "" && !isValidTokenFormat(value) {
					fmt.Printf("   %s[✕] Warning: %s contains an invalid token value; run 'tfcred add --context %s --org %s --token-type %s' to refresh it.%s\n", colorYellow, env, name, entry.Org, entry.TokenType, colorReset)
				}
			}
		}

		hasBin := true
		helperCheck := exec.Command("powershell", "-Command", "Get-Command terraform-credentials-custom.exe -ErrorAction SilentlyContinue")
		if err := helperCheck.Run(); err != nil {
			fmt.Printf("   %s[✕] PATH Warning: 'terraform-credentials-custom.exe' is not visible on your system PATH.%s\n", colorRed, colorReset)
			hasBin = false
		} else {
			fmt.Printf("   %s[✓]%s Binary Path Registration: OK\n", colorGreen, colorReset)
		}

		hasRcHook := false
		appData := os.Getenv("APPDATA")
		if appData != "" {
			rcPath := filepath.Join(appData, "terraform.rc")
			if rcBytes, err := os.ReadFile(rcPath); err == nil {
				if strings.Contains(string(rcBytes), "credentials_helper") && strings.Contains(string(rcBytes), "\"custom\"") {
					hasRcHook = true
				}
			}
		}

		if hasRcHook {
			fmt.Printf("   %s[✓]%s Terraform Profile Configuration: OK (Hook found in terraform.rc)\n", colorGreen, colorReset)
		} else {
			fmt.Printf("   %s[✕] Config Warning: 'credentials_helper \"custom\"' block is missing from your terraform.rc configuration.%s\n", colorRed, colorReset)
		}

		if !hasBin || !hasRcHook {
			fmt.Printf("\n%s[!] DIAGNOSTICS ALERT: Terraform is currently NOT routing credentials through tfcred.%s\n", colorRed, colorReset)
			fmt.Println("    To re-register your context switcher and fix these issues, execute your project installer:")
			fmt.Println("    ↳ Run: .\\scripts\\install.ps1")
		} else {
			fmt.Printf("\n%s[✓] SUCCESS: tfcred is actively mapped and ready for native Terraform execution loops.%s\n", colorGreen, colorReset)
		}

	case "whoami":
		ctx := os.Getenv("TF_CONTEXT")
		p, _ := tfcontext.Parse(ctx)
		if p.Type == "default" {
			fmt.Println("context=default")
			return
		}
		_, entry, ok := store.FindByTypeOrg(p.Type, p.Org)
		if !ok {
			fmt.Printf("%s[tfcred][error] unknown context for current TF_CONTEXT%s\n", colorRed, colorReset)
			os.Exit(1)
		}
		fmt.Printf("context=%s\norg=%s\ntype=%s\ndomain=%s\n", ctx, entry.Org, entry.TokenType, entry.Domain)

	case "explain":
		verbose := false
		trace := false
		for _, arg := range os.Args[2:] {
			if arg == "--json" {
				verbose = true
			}
			if arg == "--trace" {
				trace = true
			}
		}
		ctxRaw := os.Getenv("TF_CONTEXT")
		p, err := tfcontext.Parse(ctxRaw)
		out := map[string]any{
			"TF_CONTEXT": ctxRaw,
		}

		if trace {
			out["trace"] = map[string]any{
				"step_1_context_read": ctxRaw,
				"step_2_parse_error":  err != nil,
			}
			out["terraform_hostname"] = "app.terraform.io"
			out["credential_helper"] = "custom"
		}

		if err != nil {
			out["error"] = err.Error()
		}
		if p.Type == "default" {
			out["mode"] = "default"
			out["resolved_env"] = ""
			out["token_present"] = false
		} else {
			_, entry, ok := store.FindByTypeOrg(p.Type, p.Org)
			if ok {
				env := store.EntryEnvNames(entry)[0]
				token := os.Getenv(env)
				out["mode"] = "scoped"
				out["type"] = p.Type
				out["org"] = p.Org
				out["domain"] = entry.Domain
				out["resolved_env"] = env
				out["token_present"] = token != ""
			} else {
				out["mode"] = "scoped"
				out["type"] = p.Type
				out["org"] = p.Org
				out["resolved_env"] = ""
				out["token_present"] = false
			}
		}

		if verbose {
			b, _ := json.MarshalIndent(out, "", " ")
			fmt.Println(string(b))
		} else {
			for k, v := range out {
				fmt.Printf("%s=%v\n", k, v)
			}
		}

	default:
		fmt.Printf("%s[tfcred][error] unknown command: %s%s\n", colorRed, os.Args[1], colorReset)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`tfcred commands:
  init [--domain <domain>]       Initialize storage and choose the default domain
  config --default-domain <d>    Set the default Terraform domain
  config --show                  Show the configured default domain
  add --context <name> --org <org> --token-type <type> [--domain <domain>] [--token <token>]
                                 Store a context-scoped token mapping
  list                           List configured contexts
  remove <context>               Remove a stored context entry
  purge <context>                Remove a context and its env var
  purge --domain <domain>        Remove all contexts for a domain
  purge --all                    Remove all contexts and env vars
  switch <context>               Set TF_CONTEXT to an existing context
  current                        Print the current TF_CONTEXT value
  status                         Show current context resolution status
  env [--json] [--show-secret] [--all]
                                 Show the current context's token env vars
  doctor                         Show environment and configuration diagnostics
  whoami                         Show the current context identity
  explain [--json] [--trace]     Explain how tfcred would resolve the current context

Use 'tfcred <command> --help' for command-specific usage.`)
}

func promptDefaultDomain() string {
	fmt.Println("Select a default Terraform domain:")
	fmt.Println("1) app.terraform.io")
	fmt.Println("2) app.eu.terraform.io")
	fmt.Print("Choice [1]: ")
	var choice string
	_, _ = fmt.Scanln(&choice)
	choice = strings.TrimSpace(choice)
	if choice == "2" {
		return "app.eu.terraform.io"
	}
	return "app.terraform.io"
}

func parseDomainFlag(defaultValue string, args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--domain" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return defaultValue
}

func isSupportedDomain(domain string) bool {
	domain = strings.TrimSpace(domain)
	for _, candidate := range supportedDomains {
		if domain == candidate {
			return true
		}
	}
	return false
}

var tokenFormatPattern = regexp.MustCompile(`^[a-zA-Z0-9]{14}\.atlasv1\.[a-zA-Z0-9]{30,70}$`)

func isValidTokenFormat(token string) bool {
	return tokenFormatPattern.MatchString(strings.TrimSpace(token))
}

func printOrderedReport(report map[string]string) {
	keys := []string{"context", "TF_CONTEXT", "TF_TOKEN_envs", "TF_TOKEN_values", "TF_TOKEN_value_present"}
	for _, key := range keys {
		if value, ok := report[key]; ok {
			fmt.Printf("%s=%s\n", key, value)
		}
	}
}

func printOrderedFields(fields map[string]string) {
	keys := []string{"TF_CONTEXT", "TF_TOKEN_env", "TF_TOKEN_value", "TF_TOKEN_value_present", "error"}
	for _, key := range keys {
		if value, ok := fields[key]; ok {
			fmt.Printf("%s=%s\n", key, value)
		}
	}
}

func buildContextEnvReports(showSecret bool, contexts map[string]store.Entry) []map[string]string {
	names := make([]string, 0, len(contexts))
	for name := range contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	reports := make([]map[string]string, 0, len(names))
	for _, name := range names {
		entry := contexts[name]
		envs := store.EntryEnvNames(entry)
		masked := renderMaskedValues(envs, showSecret)
		present := anyTokenPresent(envs)
		reports = append(reports, map[string]string{
			"context":                name,
			"TF_CONTEXT":             tfContextString(entry),
			"TF_TOKEN_envs":          strings.Join(envs, ","),
			"TF_TOKEN_values":        strings.Join(masked, ","),
			"TF_TOKEN_value_present": fmt.Sprintf("%v", present),
		})
	}
	return reports
}

func renderMaskedValues(envs []string, showSecret bool) []string {
	masked := make([]string, 0, len(envs))
	for _, env := range envs {
		masked = append(masked, redactSecret(os.Getenv(env), showSecret))
	}
	return masked
}

func anyTokenPresent(envs []string) bool {
	for _, env := range envs {
		if os.Getenv(env) != "" {
			return true
		}
	}
	return false
}

func tfContextString(entry store.Entry) string {
	if entry.TokenType == "default" {
		return "default"
	}
	return fmt.Sprintf("%s:%s", entry.TokenType, entry.Org)
}

func redactSecret(value string, showSecret bool) string {
	if value == "" {
		return ""
	}
	if showSecret {
		return value
	}
	return "*********"
}

func purgeEnvVars(names []string) {
	for _, envName := range names {
		if envName == "" {
			continue
		}
		registryCmd := fmt.Sprintf("[Environment]::SetEnvironmentVariable('%s', $null, 'User')", envName)
		cmd := exec.Command("powershell", "-Command", registryCmd)
		_ = cmd.Run()
		if os.Getenv(envName) != "" {
			fmt.Printf("%s[tfcred]%s Registry key deleted for %s.%s\n", colorGreen, colorReset, envName, colorReset)
			fmt.Printf("%s[tfcred][info] Token may still exist in your PowerShell session.%s\n", colorGreen, colorReset)
		}
	}
}
