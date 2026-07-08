package resolve

import (
	"fmt"

	"github.com/amiasea/terraform-credentials-amiasea/internal/store"
	"github.com/amiasea/terraform-credentials-amiasea/internal/tfcontext"
)

func Resolve(ctx tfcontext.Context, domain string) (string, error) {
	if ctx.Key == "" {
		return "", fmt.Errorf("cannot resolve credential: target context key is empty")
	}

	token, err := store.GetToken(ctx.Key)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve token for context '%s': %w", ctx.Key, err)
	}

	if token == "" {
		return "", fmt.Errorf("retrieved token for context '%s' is empty", ctx.Key)
	}

	return token, nil
}
