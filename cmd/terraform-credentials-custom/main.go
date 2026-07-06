package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AlfredoBall/terraform-credentials-custom/internal/log"
	"github.com/AlfredoBall/terraform-credentials-custom/internal/resolve"
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

		token, err := resolve.Resolve(ctx)
		if err != nil {
			log.Err(err.Error())
			os.Exit(1)
		}

		// ----------------------------
		// IMPROVED LOGGING (NEW)
		// ----------------------------

		if ctx.Type == "default" {
			log.Info("resolution_mode=default")
			log.Info("env=TF_TOKEN_app_terraform_io")
		} else {
			env := fmt.Sprintf("TF_TOKEN_app_terraform_io_%s_%s", ctx.Type, ctx.Org)
			log.Info("resolution_mode=scoped")
			log.Info("env=" + env)
		}

		log.Info(fmt.Sprintf("context=%s type=%s org=%s", ctx.Raw, ctx.Type, ctx.Org))

		_ = json.NewEncoder(os.Stdout).Encode(creds{Token: token})

	case "store", "forget":
		os.Exit(0)

	default:
		log.Err("unknown command")
		os.Exit(1)
	}
}
