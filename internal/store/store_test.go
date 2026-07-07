package store

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestInitDoesNotCreateSyntheticDefaultContext(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	Init("app.terraform.io")

	f := Load()
	if f.DefaultDomain != "app.terraform.io" {
		t.Fatalf("expected default domain to be set, got %q", f.DefaultDomain)
	}
	if _, ok := f.Contexts["default"]; ok {
		t.Fatalf("expected no synthetic default context entry, got one")
	}
}

func TestListPrintsStoredContexts(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	Add("team-a", "acme", "team", "app.terraform.io", "")
	Add("user-b", "acme", "user", "app.terraform.io", "")

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	List()

	_ = w.Close()
	os.Stdout = oldStdout

	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	text := string(output)
	if !strings.Contains(text, "team-a") || !strings.Contains(text, "user-b") {
		t.Fatalf("expected list output to include stored contexts, got %q", text)
	}
}
