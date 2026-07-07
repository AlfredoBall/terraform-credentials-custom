package resolve

import (
	"fmt"
	"os"

	"github.com/AlfredoBall/terraform-credentials-custom/internal/store"
	"github.com/AlfredoBall/terraform-credentials-custom/internal/tfcontext"
)

func Resolve(ctx tfcontext.Context, domain string) (string, error) {
	if domain == "" {
		return "", fmt.Errorf("domain is not configured")
	}

	if ctx.Type == "default" {
		return "", fmt.Errorf("default context does not use a fallback token; configure an explicit context token first")
	}

	env := store.TokenEnvName(domain, ctx.Type, ctx.Org)
	token := os.Getenv(env)
	if token == "" {
		return "", fmt.Errorf("missing environment token mapping vector variable: %s", env)
	}
	return token, nil
}
