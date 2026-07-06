package resolve

import (
	"fmt"
	"os"

	"github.com/AlfredoBall/terraform-credentials-custom/internal/tfcontext"
)

func Resolve(ctx tfcontext.Context) (string, error) {

	if ctx.Type == "default" {
		return os.Getenv("TF_TOKEN_app_terraform_io"), nil
	}

	env := fmt.Sprintf(
		"TF_TOKEN_app_terraform_io_%s_%s",
		ctx.Type,
		ctx.Org,
	)

	token := os.Getenv(env)

	if token == "" {
		return "", fmt.Errorf("missing env var: %s", env)
	}

	return token, nil
}
