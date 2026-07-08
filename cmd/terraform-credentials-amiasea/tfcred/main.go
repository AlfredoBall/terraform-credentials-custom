package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/amiasea/terraform-credentials-amiasea/internal/store"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
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
		shouldSwitch := addCmd.Bool("switch", false, "automatically switch to this context after adding")
		_ = addCmd.Parse(os.Args[2:])

		if *ctx == "" {
			fmt.Printf("%s[tfcred][error] --context is required%s\n", colorRed, colorReset)
			os.Exit(1)
		}

		// Rule: User tokens do not carry or require a specific organization map scope
		if *tokenType == "user" && *org != "" {
			fmt.Printf("%s[tfcred][error] --org should not be specified for 'user' token types%s\n", colorRed, colorReset)
			os.Exit(1)
		}

		if *tokenType != "user" && *tokenType != "default" && *org == "" {
			fmt.Printf("%s[tfcred][error] --org is required for non-user contexts%s\n", colorRed, colorReset)
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
			fmt.Printf("%s[tfcred][error] invalid token format%s\n", colorRed, colorReset)
			os.Exit(1)
		}

		// Unique Triple Validation Rule: prevents overlapping background configurations
		for name, existingCtx := range config.Contexts {
			if name == *ctx {
				continue
			}

			// Enforce strict uniqueness for "org" level tokens (only 1 org token type per org/domain)
			if *tokenType == "org" && existingCtx.TokenType == "org" && existingCtx.Org == *org && existingCtx.Domain == *domain {
				fmt.Printf("%s[tfcred][error] Duplicate mapping rejected. An 'org' type token already exists for organization '%s' on domain '%s' under context: '%s'%s\n", colorRed, *org, *domain, name, colorReset)
				os.Exit(1)
			}

			// Traditional exact duplicate check (skips if token-type is "team" to allow multiple team tokens)
			if *tokenType != "team" && existingCtx.Org == *org && existingCtx.TokenType == *tokenType && existingCtx.Domain == *domain {
				fmt.Printf("%s[tfcred][error] Duplicate mapping rejected. This configuration already uniquely maps to context key: '%s'%s\n", colorRed, name, colorReset)
				os.Exit(1)
			}
		}

		// Overwrite Confirmation
		if _, exists := config.Contexts[*ctx]; exists {
			fmt.Printf("%s[tfcred][warning] context '%s' already exists.%s\n", colorYellow, *ctx, colorReset)
			fmt.Print("Overwrite it? [y/N]: ")
			var confirm string
			_, _ = fmt.Scanln(&confirm)
			if !strings.EqualFold(confirm, "y") && !strings.EqualFold(confirm, "yes") {
				fmt.Println("[tfcred] Aborted.")
				return
			}
		}

		// Pushes credentials securely to Windows Vault, and metadata to JSON store.
		store.Add(*ctx, *org, *tokenType, *domain, *token)
		fmt.Printf("%s[tfcred] Context '%s' configured successfully.%s\n", colorGreen, *ctx, colorReset)

		// Automated directory binding if requested via the flag
		if *shouldSwitch {
			cwd, err := os.Getwd()
			if err == nil {
				_ = store.BindDirectory(cwd, *ctx)
				fmt.Printf("%s[tfcred] Automatically bound current directory to context '%s'%s\n", colorGreen, *ctx, colorReset)
			}
		}

	case "remove":
		if len(os.Args) < 3 {
			fmt.Printf("%susage: tfcred remove <context>%s\n", colorRed, colorReset)
			os.Exit(1)
		}
		ctxName := os.Args[2]
		f := store.Load()
		if _, exists := f.Contexts[ctxName]; !exists {
			fmt.Printf("%s[tfcred][error] unknown context: %s%s\n", colorRed, ctxName, colorReset)
			os.Exit(1)
		}

		// Fix: Safe key collection to avoid concurrent map mutation crashes
		var dirsToRemove []string
		if f.Directories != nil {
			for dir, boundKey := range f.Directories {
				if boundKey == ctxName {
					dirsToRemove = append(dirsToRemove, dir)
				}
			}
			for _, dir := range dirsToRemove {
				delete(f.Directories, dir)
			}
		}

		// Perform standard file and secure vault deletions via your store package
		store.Remove(ctxName)
		fmt.Printf("%s[tfcred] Context '%s' and all associated directory bindings removed successfully.%s\n", colorGreen, ctxName, colorReset)

	case "purge":
		purgeCmd := flag.NewFlagSet("purge", flag.ExitOnError)
		force := purgeCmd.Bool("force", false, "skip the confirmation prompt")
		_ = purgeCmd.Parse(os.Args[2:])

		if !*force {
			fmt.Printf("%s[tfcred][warning] This will wipe ALL contexts, directory bindings, and vaulted tokens!%s\n", colorYellow, colorReset)
			fmt.Print("Are you absolutely sure you want to proceed? [y/N]: ")
			var confirm string
			_, _ = fmt.Scanln(&confirm)
			if !strings.EqualFold(confirm, "y") && !strings.EqualFold(confirm, "yes") {
				fmt.Println("[tfcred] Purge sequence aborted.")
				return
			}
		}

		fmt.Println("[tfcred] Wiping all secure Windows Credential Vault tokens and metadata...")
		f := store.Load()
		for name := range f.Contexts {
			// Reuse your existing, robust individual context cleanup logic!
			store.Remove(name)
		}

		// Clear out any leftover directory keys and write the fresh empty baseline
		store.NukeStorage()

		fmt.Printf("%s[tfcred] SUCCESS: System entirely purged. Use 'tfcred init' to start fresh.%s\n", colorGreen, colorReset)

	case "list":
		store.List()

	case "switch":
		if len(os.Args) < 3 {
			fmt.Printf("%susage: tfcred switch <context>%s\n", colorRed, colorReset)
			os.Exit(1)
		}
		ctxName := os.Args[2]

		f := store.Load()
		if _, ok := f.Contexts[ctxName]; !ok {
			fmt.Printf("%s[tfcred][error] unknown context: %s%s\n", colorRed, ctxName, colorReset)
			os.Exit(1)
		}

		// Grab the active working directory of the terminal invoking the tool
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("%s[tfcred][error] failed to determine current directory: %v%s\n", colorRed, err, colorReset)
			os.Exit(1)
		}

		// Write the persistent link to storage (e.g., store.BindDirectory)
		err = store.BindDirectory(cwd, ctxName)
		if err != nil {
			fmt.Printf("%s[tfcred][error] failed to save directory mapping: %v%s\n", colorRed, err, colorReset)
			os.Exit(1)
		}

		fmt.Printf("%s[tfcred] Context '%s' is now persistently bound to directory: %s%s\n", colorGreen, ctxName, cwd, colorReset)

	case "status":
		// 1. Resolve context cleanly using the directory-based system
		cwd, _ := os.Getwd()
		contextKey, found := store.ResolveContextByDir(cwd)
		if !found {
			fmt.Printf("[tfcred] No active context bound to this directory. Run 'tfcred switch <context>' here to map it.\n")
			return
		}

		// 2. Fetch the metadata configuration for this context key
		storeFile := store.Load()
		entry, ok := storeFile.Contexts[contextKey]
		if !ok {
			fmt.Printf("%s[tfcred][error] unknown context key '%s' bound to this directory%s\n", colorRed, contextKey, colorReset)
			os.Exit(1)
		}

		// 3. Print out the accurate profile parameters cleanly
		fmt.Printf("[tfcred] context=%s type=%s org=%s domain=%s\n", contextKey, entry.TokenType, entry.Org, entry.Domain)

	case "current":
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("%s[tfcred][error] failed to read current directory: %v%s\n", colorRed, err, colorReset)
			os.Exit(1)
		}

		// Track the active binding by climbing up the tree from our current directory
		contextKey, found := store.ResolveContextByDir(cwd)
		if !found {
			fmt.Printf("[tfcred] Current directory tree is not bound to a context profile. (Run 'tfcred switch <context>')\n")
			return
		}

		fmt.Printf("Active Directory: %s\n", cwd)
		fmt.Printf("active_context=%s\n", contextKey)

	case "context":
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

		// 1. Resolve context cleanly using the directory-based system
		cwd, _ := os.Getwd()
		contextKey, found := store.ResolveContextByDir(cwd)
		storeFile := store.Load()

		// 2. If no context is bound to this directory, render the global summary report
		if !found || contextKey == "" {
			reports := buildContextReports(showSecret, storeFile.Contexts)
			if verbose {
				out := map[string]any{"working_directory": cwd, "contexts": reports}
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

		// 3. Context found! Perform a direct map key lookup
		entry, ok := storeFile.Contexts[contextKey]
		if !ok {
			fmt.Printf("%s[tfcred][error] unknown context key '%s' bound to this directory%s\n", colorRed, contextKey, colorReset)
			os.Exit(1)
		}

		// Fetch the actual decrypted token value safely from the Windows Vault
		vaultToken, _ := store.GetToken(contextKey)
		hasToken := vaultToken != ""
		displayToken := "[MASKED]"
		if showSecret && hasToken {
			displayToken = vaultToken
		}

		// Generate the target environment variable label Terraform expects behind the scenes
		expectedEnvName := store.TokenVaultKey(entry.Domain, entry.TokenType, entry.Org)

		// Fix: Define separate data objects to satisfy both JSON (any) and flat text (string) output types
		if verbose {
			resultJSON := map[string]any{
				"bound_directory": cwd,
				"context":         contextKey,
			}

			if all {
				var allEnvs []string
				var hasTokens []string
				for name, ctxEntry := range storeFile.Contexts {
					allEnvs = append(allEnvs, store.TokenVaultKey(ctxEntry.Domain, ctxEntry.TokenType, ctxEntry.Org))
					t, _ := store.GetToken(name)
					hasTokens = append(hasTokens, fmt.Sprintf("%s:%t", name, t != ""))
				}
				resultJSON["expected_terraform_envs"] = strings.Join(allEnvs, ",")
				resultJSON["vault_token_statuses"] = strings.Join(hasTokens, ",")
				resultJSON["has_active_token"] = hasToken
			} else {
				resultJSON["target_env_name"] = expectedEnvName
				resultJSON["token_value"] = displayToken
				resultJSON["has_vaulted_token"] = hasToken
			}

			enc, _ := json.MarshalIndent(resultJSON, "", " ")
			fmt.Println(string(enc))
		} else {
			if all {
				for name, ctxEntry := range storeFile.Contexts {
					t, _ := store.GetToken(name)
					eName := store.TokenVaultKey(ctxEntry.Domain, ctxEntry.TokenType, ctxEntry.Org)
					tDisplay := "[MASKED]"
					if showSecret && t != "" {
						tDisplay = t
					} else if t == "" {
						tDisplay = "[NOT_SET]"
					}
					fmt.Printf("context=%s target_env=%s token=%s\n", name, eName, tDisplay)
				}
				fmt.Printf("active_context=%s\n", contextKey)
				fmt.Printf("has_active_token=%t\n", hasToken)
			} else {
				// Fix: Use strict map[string]string for the text field printer utility block
				resultText := map[string]string{
					"target_env_name":   expectedEnvName,
					"token_value":       displayToken,
					"has_vaulted_token": fmt.Sprintf("%t", hasToken), // String-formatted boolean
				}
				printOrderedFields(resultText)
			}
		}

	case "doctor":
		fmt.Println("[tfcred doctor] Beginning system diagnostics...")

		// 1. Context Resolution Strategy (Directory Scoped Only)
		cwd, _ := os.Getwd()
		contextKey, found := store.ResolveContextByDir(cwd)
		storeFile := store.Load()

		if found {
			if _, exists := storeFile.Contexts[contextKey]; exists {
				fmt.Printf("  %s[✓]%s Active Context: Key '%s' is registered and OK for this directory\n", colorGreen, colorReset, contextKey)
			} else {
				fmt.Printf("  %s[✕] Error: Active directory context key '%s' does not exist in your store metadata.%s\n", colorRed, contextKey, colorReset)
			}
		} else {
			fmt.Printf("  %s[-]%s Active Context: This directory is unbound. Run 'tfcred switch <context>' here to activate it.\n", colorYellow, colorReset)
		}

		// 2. Validate structural mapping storage file
		if storeFile.Contexts == nil {
			fmt.Printf("  %s[✕] Error: contexts.json file missing or unreadable.%s\n", colorRed, colorReset)
		} else {
			fmt.Printf("  %s[✓]%s Configuration Storage: OK\n", colorGreen, colorReset)
		}

		// 3. Verify target executable binary path discovery (Strict HashiCorp Location Rule)
		hasBin := false
		appData := os.Getenv("APPDATA")
		expectedHelperPath := filepath.Join(appData, "terraform.d", "plugins", "terraform-credentials-amiasea.exe")

		if _, err := os.Stat(expectedHelperPath); err == nil {
			fmt.Printf("  %s[✓]%s Binary Path Registration: OK (Discovered natively by Terraform)\n", colorGreen, colorReset)
			hasBin = true
		} else {
			fmt.Printf("  %s[✕] Path Error: 'terraform-credentials-amiasea.exe' is missing from your native plugin directory: %s%s\n", colorRed, expectedHelperPath, colorReset)
		}

		// 4. Trace application configurations inside native user windows profiles (.rc or .tfrc formats)
		hasRcHook := false
		userProfile := os.Getenv("USERPROFILE")
		var verifiedConfigPath string

		// Scan both standard fallback deployment naming schemas on Windows
		configFiles := []string{
			filepath.Join(appData, "terraform.rc"),
			filepath.Join(userProfile, "terraform.tfrc"),
		}

		for _, path := range configFiles {
			if rcBytes, err := os.ReadFile(path); err == nil {
				// Updated rule checking strings to locate your unique brand identifier block
				if strings.Contains(string(rcBytes), "credentials_helper") && strings.Contains(string(rcBytes), "\"amiasea\"") {
					hasRcHook = true
					verifiedConfigPath = path
					break
				}
			}
		}

		if hasRcHook {
			fmt.Printf("  %s[✓]%s Terraform Profile Configuration: OK (Hook found in %s)\n", colorGreen, colorReset, filepath.Base(verifiedConfigPath))
		} else {
			fmt.Printf("  %s[✕] Config Warning: 'credentials_helper \"amiasea\"' block is missing from your active terraform configuration profiles.%s\n", colorRed, colorReset)
		}

		// 5. Final Diagnostic Evaluation Panel
		if !hasBin || !hasRcHook {
			fmt.Printf("\n%s[!] DIAGNOSTICS ALERT: Terraform is currently NOT routing credentials through tfcred.%s\n", colorRed, colorReset)
			fmt.Println("  Please verify that 'terraform-credentials-amiasea.exe' is added to your System PATH variables,")
			fmt.Println("  and your terraform configuration file contains the mandatory credentials_helper configuration block.")
		} else {
			fmt.Printf("\n%s[✓] SUCCESS: tfcred is actively mapped and ready for native Terraform execution loops.%s\n", colorGreen, colorReset)
		}

	case "whoami":
		// 1. Resolve context cleanly using the directory-based system
		cwd, _ := os.Getwd()
		contextKey, found := store.ResolveContextByDir(cwd)
		if !found {
			fmt.Printf("%s[tfcred][error] No active context bound to this directory. Run 'tfcred switch <context>' here first.%s\n", colorRed, colorReset)
			os.Exit(1)
		}

		// 2. Fetch the metadata configuration for this context key
		storeFile := store.Load()
		entry, ok := storeFile.Contexts[contextKey]
		if !ok {
			fmt.Printf("%s[tfcred][error] unknown context key '%s' bound to this directory%s\n", colorRed, contextKey, colorReset)
			os.Exit(1)
		}

		// 3. Print out the accurate profile parameters cleanly
		fmt.Printf("context=%s\norg=%s\ntype=%s\ndomain=%s\n", contextKey, entry.Org, entry.TokenType, entry.Domain)

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

		// 1. Context Resolution Strategy (Directory Scoped Only)
		var contextKey string
		var resolutionStrategy string

		cwd, err := os.Getwd()
		if err == nil {
			resolvedKey, found := store.ResolveContextByDir(cwd)
			if found {
				contextKey = resolvedKey
				resolutionStrategy = "directory_scoped"
			}
		}

		out := map[string]any{
			"working_directory": cwd,
		}

		if trace {
			out["trace"] = map[string]any{
				"step_1_working_dir":     cwd,
				"step_2_resolved_via":    resolutionStrategy,
				"step_3_context_matched": contextKey != "",
			}
			out["terraform_hostname"] = "" // Initialized as empty until verified
			out["credential_helper"] = "amiasea"
		}

		// 2. Metadata Profile Mapping Verification
		if contextKey == "" {
			out["error"] = "No active context bound to this directory."
			out["mode"] = "unresolved"
		} else {
			storeFile := store.Load()
			entry, ok := storeFile.Contexts[contextKey]
			if ok {
				token, _ := store.GetToken(contextKey)
				vaultKey := store.TokenVaultKey(entry.Domain, entry.TokenType, entry.Org)

				out["mode"] = resolutionStrategy
				out["context"] = contextKey
				out["type"] = entry.TokenType
				out["org"] = entry.Org
				out["domain"] = entry.Domain
				out["vault_key"] = vaultKey    // Fixed: Replaced 'resolved_env' with 'vault_key'
				out["has_token"] = token != "" // Fixed: Replaced 'token_present' with 'has_token'

				if trace {
					out["terraform_hostname"] = entry.Domain
				}
			} else {
				out["error"] = fmt.Sprintf("unknown context key '%s' found in config store", contextKey)
				out["mode"] = "failed_lookup"
				out["context"] = contextKey
				out["vault_key"] = ""
				out["has_token"] = false
			}
		}

		// 3. Format telemetry variables cleanly for terminal display
		if verbose {
			b, _ := json.MarshalIndent(out, "", " ")
			fmt.Println(string(b))
		} else {
			for k, v := range out {
				if k == "trace" {
					if traceMap, ok := v.(map[string]any); ok {
						fmt.Println("[Trace Analytics]:")
						for tk, tv := range traceMap {
							fmt.Printf(" ↳ %s=%v\n", tk, tv)
						}
						continue
					}
				}
				fmt.Printf("%s=%v\n", k, v)
			}
		}

	case "orphaned":
		fmt.Println("[tfcred] Inspecting registered folder mappings for stale data...")

		missingPaths, missingContexts := store.CheckOrphanedDirectories()

		if len(missingPaths) == 0 && len(missingContexts) == 0 {
			fmt.Printf("%s[✓] No orphaned links found. All directory bindings are valid and mapped.%s\n", colorGreen, colorReset)
			return
		}

		if len(missingPaths) > 0 {
			fmt.Printf("\n%s[!] Paths that no longer physically exist on disk (%d):%s\n", colorRed, len(missingPaths), colorReset)
			for _, path := range missingPaths {
				fmt.Printf("  • %s\n", path)
			}
		}

		if len(missingContexts) > 0 {
			fmt.Printf("\n%s[!] Paths mapped to context profiles that have been deleted (%d):%s\n", colorRed, len(missingContexts), colorReset)
			for _, binding := range missingContexts {
				fmt.Printf("  • %s\n", binding)
			}
		}

		fmt.Printf("\n%s[i] To safely clear these stale bindings out of your registry, run: tfcred clean-dirs%s\n", colorCyan, colorReset)

	case "clean-dirs":
		fmt.Println("[tfcred] Purging stale directory bindings from storage...")

		// Execute the destructive self-healing sweep we wrote earlier
		deadPaths, deadContexts := store.CleanOrphanedDirectories()

		if len(deadPaths) == 0 && len(deadContexts) == 0 {
			fmt.Printf("%s[✓] Nothing to clean! Your directory registry is perfectly optimized.%s\n", colorGreen, colorReset)
			return
		}

		if len(deadPaths) > 0 {
			fmt.Printf("  %s[✓]%s Removed %d paths missing from the filesystem.\n", colorGreen, colorReset, len(deadPaths))
		}
		if len(deadContexts) > 0 {
			fmt.Printf("  %s[✓]%s Flushed %d paths linked to missing context keys.\n", colorGreen, colorReset, len(deadContexts))
		}
		fmt.Printf("%s[✓] Registry cleanup sequence complete.%s\n", colorGreen, colorReset)

	default:
		fmt.Printf("%s[tfcred][error] unknown command: %s%s\n", colorRed, os.Args[1], colorReset)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`tfcred — Terraform credential context manager (Branded for amiasea)

Usage:
  tfcred <command> [arguments]

Core Commands:
  init [--domain <domain>]                                    Initialize storage and choose the default domain
  config --default-domain <d>                                 Set the default Terraform domain
  config                                                      Show the configured default domain
  add --context <name> --org <org> --token-type <type> 
      [--domain <domain>] --token <token> [--switch]          Store a context-scoped secure vault profile mapping
  list                                                        List configured contexts in a structured table layout
  switch <context>                                            Switch the active context for your current working directory
  remove <context>                                            Remove a specific context entry and its local directory bindings
  purge [--force]                                             Nuclear option: Completely wipe all metadata and vaulted tokens cleanly

Inspection & Diagnostics:
  current                                                     Print the active context key string bound to this directory
  status                                                      Show current folder resolution status and metadata mapping parameters
  env [--json] [--show-secret] [--all]                        Display active or global vault lookup keys and token parameters
  whoami                                                      Show detailed platform identity structures about the active context
  explain [--json] [--trace]                                  Trace and explain how tfcred resolves the active directory path
  orphaned                                                    Inspect registered folder mappings for stale or missing directory paths
  clean-dirs                                                  Purge stale directory bindings safely from your local config store

Use 'tfcred <command> --help' for command-specific argument details.`)
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
	keys := []string{"context", "vault_key", "token", "has_token"}
	for _, key := range keys {
		if value, ok := report[key]; ok {
			fmt.Printf("%s=%s\n", key, value)
		}
	}
}

func printOrderedFields(fields map[string]string) {
	keys := []string{"context", "vault_key", "token", "has_token", "error"}
	for _, key := range keys {
		if value, ok := fields[key]; ok {
			fmt.Printf("%s=%s\n", key, value)
		}
	}
}

func buildContextReports(showSecret bool, contexts map[string]store.Entry) []map[string]string {
	names := make([]string, 0, len(contexts))
	for name := range contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	reports := make([]map[string]string, 0, len(names))
	for _, name := range names {
		entry := contexts[name]

		expectedKey := store.TokenVaultKey(entry.Domain, entry.TokenType, entry.Org)

		vaultToken, _ := store.GetToken(name)
		hasToken := vaultToken != ""
		displayToken := "[MASKED]"
		if showSecret && hasToken {
			displayToken = vaultToken
		} else if !hasToken {
			displayToken = "[NOT_SET]"
		}

		reports = append(reports, map[string]string{
			"context":   name,
			"vault_key": expectedKey, // Fixed: Clean, self-documenting key name
			"token":     displayToken,
			"has_token": fmt.Sprintf("%t", hasToken),
		})
	}
	return reports
}
