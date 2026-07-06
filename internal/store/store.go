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

	for k, v := range f.Contexts {
		fmt.Printf("%s → org=%s type=%s\n", k, v.Org, v.TokenType)
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
	b, _ := json.MarshalIndent(f, "", "  ")
	_ = os.WriteFile(path, b, 0644)
}
