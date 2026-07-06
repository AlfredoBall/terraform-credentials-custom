package store

import (
	"encoding/json"
	"fmt"
	"os"
)

type Entry struct {
	Org       string `json:"org"`
	TokenType string `json:"tokenType"`
}

type File struct {
	Contexts map[string]Entry `json:"contexts"`
}

const path = "contexts.json"

func Init() {
	if _, err := os.Stat(path); err == nil {
		fmt.Println("[tfcred] already initialized")
		return
	}
	f := File{
		Contexts: map[string]Entry{
			"default": {
				Org:       "",
				TokenType: "default",
			},
		},
	}
	write(f)
}

func Add(ctx, org, tokenType, token string) {
	f := load()
	f.Contexts[ctx] = Entry{
		Org:       org,
		TokenType: tokenType,
	}
	if token != "" {
		env := fmt.Sprintf("TF_TOKEN_app_terraform_io_%s_%s", tokenType, org)
		_ = os.Setenv(env, token)
		fmt.Println("[tfcred] set env var:", env)
	}
	write(f)
}

func List() {
	f := load()

	// Capture active context state from shell environment
	currentCtx := os.Getenv("TF_CONTEXT")
	if currentCtx == "" {
		currentCtx = "default"
	}

	// Catch completely empty configuration state files cleanly
	if len(f.Contexts) == 0 {
		fmt.Println("Configured Terraform Contexts:")
		activeMarker := "  "
		if currentCtx == "default" {
			activeMarker = "* "
		}
		fmt.Printf("%sdefault (Native/Global Fallback Context)\n", activeMarker)
		fmt.Println("\n[tfcred] No custom contexts configured yet. Use 'tfcred add' to create one.")
		return
	}

	fmt.Println("Configured Terraform Contexts:")
	for k, v := range f.Contexts {
		activeMarker := "  "
		if k == currentCtx {
			activeMarker = "* "
		}

		if k == "default" {
			fmt.Printf("%s%s (Native/Global Fallback Context)\n", activeMarker, k)
		} else {
			fmt.Printf("%s%s → org=%s type=%s\n", activeMarker, k, v.Org, v.TokenType)
		}
	}

	// Print an optional tip for beginners if they only have the default entry listed
	if len(f.Contexts) == 1 && f.Contexts["default"].TokenType == "default" {
		fmt.Println("\n[tfcred] No custom contexts configured yet. Use 'tfcred add' to create one.")
	}
}

func Load() File {
	return load()
}

func load() File {
	b, err := os.ReadFile(path)
	if err != nil {
		return File{Contexts: map[string]Entry{}}
	}
	var f File
	_ = json.Unmarshal(b, &f)
	return f
}

func write(f File) {
	b, _ := json.MarshalIndent(f, "", " ")
	_ = os.WriteFile(path, b, 0644)
}
