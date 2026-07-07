package main

import (
	"testing"

	"github.com/AlfredoBall/terraform-credentials-custom/internal/store"
)

func TestRedactSecretMasksValueByDefault(t *testing.T) {
	got := redactSecret("super-secret", false)
	if got != "*********" {
		t.Fatalf("expected masked secret, got %q", got)
	}
}

func TestRedactSecretRevealsValueWhenRequested(t *testing.T) {
	got := redactSecret("super-secret", true)
	if got != "super-secret" {
		t.Fatalf("expected full secret when requested, got %q", got)
	}
}

func TestRedactSecretReturnsEmptyStringForEmptyValue(t *testing.T) {
	got := redactSecret("", false)
	if got != "" {
		t.Fatalf("expected empty output for empty secret, got %q", got)
	}
}

func TestBuildContextEnvReportsIncludesStoredContexts(t *testing.T) {
	contexts := map[string]store.Entry{
		"platform": {
			Org:       "acme",
			TokenType: "team",
			Domain:    "app.terraform.io",
		},
	}

	reports := buildContextEnvReports(false, contexts)
	if len(reports) != 1 {
		t.Fatalf("expected one report, got %d", len(reports))
	}
	if reports[0]["context"] != "platform" {
		t.Fatalf("expected platform context report, got %#v", reports[0])
	}
	if reports[0]["TF_TOKEN_envs"] != "TF_TOKEN_app_terraform_io_team_acme" {
		t.Fatalf("expected scoped env name, got %q", reports[0]["TF_TOKEN_envs"])
	}
}

func TestIsValidTokenFormat(t *testing.T) {
	if !isValidTokenFormat("abcdefghijklmn.atlasv1.abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatal("expected valid token format to pass")
	}
	if isValidTokenFormat("not-a-valid-token") {
		t.Fatal("expected invalid token format to fail")
	}
}

func TestShouldShowHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args", args: nil, want: true},
		{name: "help flag", args: []string{"--help"}, want: true},
		{name: "short help flag", args: []string{"-h"}, want: true},
		{name: "command", args: []string{"init"}, want: false},
		{name: "unknown command", args: []string{"bogus"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldShowHelp(tt.args); got != tt.want {
				t.Fatalf("shouldShowHelp(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestVersionStringUsesProvidedVersion(t *testing.T) {
	if got := versionString("1.2.3"); got != "1.2.3" {
		t.Fatalf("expected 1.2.3, got %q", got)
	}
}

func TestVersionStringFallsBackToDev(t *testing.T) {
	if got := versionString(""); got != "dev" {
		t.Fatalf("expected dev fallback, got %q", got)
	}
}
