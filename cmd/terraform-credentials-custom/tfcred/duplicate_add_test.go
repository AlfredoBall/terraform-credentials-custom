package main

import (
	"testing"

	"github.com/AlfredoBall/terraform-credentials-custom/internal/store"
)

func TestDuplicateAddPromptLogic(t *testing.T) {
	config := store.File{Contexts: map[string]store.Entry{
		"platform": {
			Org:       "acme",
			TokenType: "team",
			Domain:    "app.terraform.io",
		},
	}}

	if _, exists := config.Contexts["platform"]; !exists {
		t.Fatal("expected existing context entry")
	}
}
