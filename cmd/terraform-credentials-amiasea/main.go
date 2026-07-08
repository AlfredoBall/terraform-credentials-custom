package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/amiasea/terraform-credentials-amiasea/internal/log"
	"github.com/amiasea/terraform-credentials-amiasea/internal/resolve"
	"github.com/amiasea/terraform-credentials-amiasea/internal/store"
	"github.com/amiasea/terraform-credentials-amiasea/internal/tfcontext"
)

type creds struct {
	Token string `json:"token"`
}

func main() {
	if len(os.Args) < 2 {
		log.Err("missing command")
		os.Exit(1)
	}

	// 1. CRITICAL FIX: Evaluate the command name string at index 1 instead of the raw slice object
	switch os.Args[1] {
	case "get":
		// 1. Prime Strategy: Resolve context dynamically based strictly on active working directory
		cwd, err := os.Getwd()
		if err != nil {
			log.Err(fmt.Sprintf("failed to get current working directory: %v", err))
			os.Exit(1)
		}

		contextKey, found := store.ResolveContextByDir(cwd)
		if !found {
			log.Err(fmt.Sprintf("this directory '%s' is not bound to a tfcred context. Run 'tfcred switch <context>' here first.", cwd))
			os.Exit(1)
		}
		resolutionMode := "directory_scoped"

		// 2. Load Metadata Config and Fetch the Context Profile
		storeFile := store.Load()
		entry, ok := storeFile.Contexts[contextKey]
		if !ok {
			log.Err(fmt.Sprintf("unknown context key '%s' found during mapping lookup", contextKey))
			os.Exit(1)
		}

		// 3. Resolve Domain cleanly from the explicit context entry metadata
		domain := entry.Domain
		if domain == "" {
			domain = storeFile.DefaultDomain
			if domain == "" {
				domain = "app.terraform.io"
			}
		}

		// 4. Pass the simplified tfcontext.Context to your internal resolver
		activeCtx := tfcontext.Context{Key: contextKey}
		token, err := resolve.Resolve(activeCtx, domain)
		if err != nil {
			log.Err(err.Error())
			os.Exit(1)
		}

		// 5. Generate Environment Log Analytics tracking variables safely
		env := store.TokenVaultKey(domain, entry.TokenType, entry.Org)
		log.Info("resolution_mode=" + resolutionMode)
		log.Info("env=" + env)
		log.Info(fmt.Sprintf("context=%s type=%s org=%s domain=%s", contextKey, entry.TokenType, entry.Org, domain))

		_ = json.NewEncoder(os.Stdout).Encode(creds{Token: token})

	case "store", "forget":
		os.Exit(0)

	default:
		log.Err("unknown command")
		os.Exit(1)
	}
}
