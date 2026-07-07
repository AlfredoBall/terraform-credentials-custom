package resolve

import (
	"strings"
	"testing"

	"github.com/AlfredoBall/terraform-credentials-custom/internal/tfcontext"
)

func TestResolveRejectsDefaultContextWithoutExplicitContextToken(t *testing.T) {
	_, err := Resolve(tfcontext.Context{Raw: "default", Type: "default"}, "app.terraform.io")
	if err == nil {
		t.Fatal("expected default context resolution to fail")
	}
	if !strings.Contains(err.Error(), "default context") {
		t.Fatalf("expected default context error, got %v", err)
	}
}
