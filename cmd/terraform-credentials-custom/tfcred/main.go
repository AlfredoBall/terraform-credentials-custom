package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/AlfredoBall/terraform-credentials-custom/internal/store"
	"github.com/AlfredoBall/terraform-credentials-custom/internal/tfcontext"
)

func main() {

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {

	case "init":
		store.Init()
		fmt.Println("[tfcred] initialized")

	case "add":
		addCmd := flag.NewFlagSet("add", flag.ExitOnError)

		ctx := addCmd.String("context", "", "context name")
		org := addCmd.String("org", "", "organization")
		tokenType := addCmd.String("token-type", "user", "user|team|org|default")
		token := addCmd.String("token", "", "optional token")

		_ = addCmd.Parse(os.Args[2:])

		if *ctx == "" {
			fmt.Println("[tfcred][error] --context is required")
			os.Exit(1)
		}

		store.Add(*ctx, *org, *tokenType, *token)

	case "list":
		store.List()

	case "switch":

		if len(os.Args) < 3 {
			fmt.Println("usage: tfcred switch <context>")
			os.Exit(1)
		}

		ctxName := os.Args[2]
		f := store.Load()

		entry, ok := f.Contexts[ctxName]
		if !ok {
			fmt.Printf("[tfcred][error] unknown context: %s\n", ctxName)
			os.Exit(1)
		}

		var tfContextStr string

		if entry.TokenType == "default" {
			tfContextStr = "default"
		} else {
			tfContextStr = fmt.Sprintf("%s:%s", entry.TokenType, entry.Org)
		}

		os.Setenv("TF_CONTEXT", tfContextStr)

		fmt.Printf("\033[32m[tfcred]\033[0m switched to %s (%s)\n",
			ctxName, tfContextStr)

	case "status":
		ctx := os.Getenv("TF_CONTEXT")

		p, _ := tfcontext.Parse(ctx)

		if p.Type == "default" {
			fmt.Println("[tfcred] context=default")
			return
		}

		fmt.Printf("[tfcred] type=%s org=%s\n", p.Type, p.Org)

	case "current":
		fmt.Println(os.Getenv("TF_CONTEXT"))

	// ----------------------------
	// ENV --JSON
	// ----------------------------
	case "env":

		verbose := false
		if len(os.Args) > 2 && os.Args[2] == "--json" {
			verbose = true
		}

		ctx := os.Getenv("TF_CONTEXT")
		p, err := tfcontext.Parse(ctx)

		result := map[string]string{
			"TF_CONTEXT": ctx,
		}

		if err != nil {
			result["error"] = err.Error()
		}

		if p.Type == "default" {
			result["TF_TOKEN_app_terraform_io"] = os.Getenv("TF_TOKEN_app_terraform_io")
		} else {
			env := fmt.Sprintf("TF_TOKEN_app_terraform_io_%s_%s", p.Type, p.Org)
			result["TF_TOKEN_env"] = env
			result["TF_TOKEN_value_present"] = fmt.Sprintf("%v", os.Getenv(env) != "")
		}

		if verbose {
			enc, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(enc))
		} else {
			for k, v := range result {
				fmt.Printf("%s=%s\n", k, v)
			}
		}

	// ----------------------------
	// DOCTOR --VERBOSE
	// ----------------------------
	case "doctor":

		verbose := len(os.Args) > 2 && os.Args[2] == "--verbose"

		fmt.Println("[tfcred doctor]")

		ctxRaw := os.Getenv("TF_CONTEXT")

		p, err := tfcontext.Parse(ctxRaw)

		if verbose {
			fmt.Println("raw_context:", ctxRaw)
		}

		if err != nil {
			fmt.Println("[error] invalid TF_CONTEXT format:", err)
		} else {
			fmt.Println("TF_CONTEXT OK")
		}

		f := store.Load()

		if f.Contexts == nil {
			fmt.Println("[error] contexts.json missing or invalid")
		} else {
			fmt.Println("contexts.json OK")
		}

		if p.Type != "default" {
			env := fmt.Sprintf("TF_TOKEN_app_terraform_io_%s_%s", p.Type, p.Org)

			val := os.Getenv(env)

			if val == "" {
				fmt.Println("[warning] missing env var:", env)
			} else if verbose {
				fmt.Println("resolved_token_env:", env)
				fmt.Println("token_present: true")
			}
		}

	case "whoami":

		f := store.Load()
		ctx := os.Getenv("TF_CONTEXT")

		p, _ := tfcontext.Parse(ctx)

		if p.Type == "default" {
			fmt.Println("context=default")
			return
		}

		entry := f.Contexts[p.Org]

		fmt.Printf("context=%s\norg=%s\ntype=%s\n",
			ctx, entry.Org, entry.TokenType)

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

		// ----------------------------
		// TRACE MODE (extra verbosity)
		// ----------------------------
		if trace {

			out["trace"] = map[string]any{
				"step_1_context_read": ctxRaw,
				"step_2_parse_error":  err != nil,
			}

			// Terraform CLI always resolves credentials per hostname
			out["terraform_hostname"] = "app.terraform.io"
			out["credential_helper"] = "custom"

			// Show relevant env vars ONLY (not full env dump)
			relevant := []string{
				"TF_TOKEN_app_terraform_io",
			}

			if p.Type != "default" {
				relevant = append(relevant,
					fmt.Sprintf("TF_TOKEN_app_terraform_io_%s_%s", p.Type, p.Org),
				)
			}

			envState := map[string]bool{}
			for _, k := range relevant {
				envState[k] = os.Getenv(k) != ""
			}

			out["env_state"] = envState
		}

		// ----------------------------
		// NORMAL RESOLUTION OUTPUT
		// ----------------------------
		if err != nil {
			out["error"] = err.Error()
		}

		if p.Type == "default" {

			env := "TF_TOKEN_app_terraform_io"
			token := os.Getenv(env)

			out["mode"] = "default"
			out["resolved_env"] = env
			out["token_present"] = token != ""

		} else {

			env := fmt.Sprintf(
				"TF_TOKEN_app_terraform_io_%s_%s",
				p.Type,
				p.Org,
			)

			token := os.Getenv(env)

			out["mode"] = "scoped"
			out["type"] = p.Type
			out["org"] = p.Org
			out["resolved_env"] = env
			out["token_present"] = token != ""
		}

		// ----------------------------
		// OUTPUT FORMAT
		// ----------------------------
		if verbose {
			b, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(b))
		} else {
			for k, v := range out {
				fmt.Printf("%s=%v\n", k, v)
			}
		}

	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println(`tfcred commands:
  init
  add
  list
  switch
  current
  status
  env [--json]
  doctor [--verbose]
  whoami`)
}
