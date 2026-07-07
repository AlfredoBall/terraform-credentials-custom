package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AlfredoBall/terraform-credentials-custom/internal/log"
	"github.com/AlfredoBall/terraform-credentials-custom/internal/resolve"
	"github.com/AlfredoBall/terraform-credentials-custom/internal/store"
	"github.com/AlfredoBall/terraform-credentials-custom/internal/tfcontext"
)

type creds struct {
	Token string `json:"token"`
}

func main() {
	if len(os.Args) < 2 {
		log.Err("missing command")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "get":
		ctx, err := tfcontext.Parse(os.Getenv("TF_CONTEXT"))
		if err != nil {
			log.Err(err.Error())
			os.Exit(1)
		}

		storeFile := store.Load()
		domain := storeFile.DefaultDomain
		if domain == "" {
			domain = "app.terraform.io"
		}

		if ctx.Type != "default" {
			_, entry, ok := store.FindByTypeOrg(ctx.Type, ctx.Org)
			if ok && entry.Domain != "" {
				domain = entry.Domain
			}
		}

		token, err := resolve.Resolve(ctx, domain)
		if err != nil {
			log.Err(err.Error())
			os.Exit(1)
		}

		var env string
		if ctx.Type == "default" {
			log.Info("resolution_mode=default")
			error := fmt.Errorf("default context does not use a fallback token; configure an explicit context token first")
			log.Err(error.Error())
			os.Exit(1)
		}

		env = store.TokenEnvName(domain, ctx.Type, ctx.Org)
		log.Info("resolution_mode=scoped")
		log.Info("env=" + env)
		log.Info(fmt.Sprintf("context=%s type=%s org=%s domain=%s", ctx.Raw, ctx.Type, ctx.Org, domain))

		_ = json.NewEncoder(os.Stdout).Encode(creds{Token: token})

	case "store", "forget":
		os.Exit(0)

	default:
		log.Err("unknown command")
		os.Exit(1)
	}
}
