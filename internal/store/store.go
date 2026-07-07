package store

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Entry struct {
	Org       string `json:"org"`
	TokenType string `json:"tokenType"`
	Domain    string `json:"domain"`
}

type File struct {
	Contexts      map[string]Entry `json:"contexts"`
	DefaultDomain string           `json:"defaultDomain"`
}

func getStoragePath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "contexts.json"
	}
	targetDir := filepath.Join(appData, "tfcred")
	_ = os.MkdirAll(targetDir, 0755)
	return filepath.Join(targetDir, "contexts.json")
}

func Init(defaultDomain string) {
	storageFile := getStoragePath()
	f := File{
		DefaultDomain: defaultDomain,
		Contexts:      map[string]Entry{},
	}

	if _, err := os.Stat(storageFile); err == nil {
		current := load()
		if current.DefaultDomain == "" {
			current.DefaultDomain = defaultDomain
		}
		if current.Contexts == nil {
			current.Contexts = map[string]Entry{}
		}
		delete(current.Contexts, "default")
		write(current)
		fmt.Println("[tfcred] already initialized globally at:", storageFile)
		return
	}

	write(f)
}

func Add(ctx, org, tokenType, domain, token string) {
	f := load()
	if f.Contexts == nil {
		f.Contexts = map[string]Entry{}
	}
	if domain == "" {
		domain = f.DefaultDomain
	}
	f.Contexts[ctx] = Entry{
		Org:       org,
		TokenType: tokenType,
		Domain:    domain,
	}

	if token != "" {
		envName := TokenEnvName(domain, tokenType, org)
		registryCmd := fmt.Sprintf("[Environment]::SetEnvironmentVariable('%s', '%s', 'User')", envName, token)
		cmd := exec.Command("powershell", "-Command", registryCmd)
		_ = cmd.Run()

		fmt.Println("[tfcred] Successfully registered persistent variable target:", envName)
	}
	write(f)
}

func Remove(ctx string) (Entry, bool) {
	f := load()
	entry, exists := f.Contexts[ctx]
	if !exists {
		return Entry{}, false
	}

	delete(f.Contexts, ctx)
	write(f)

	return entry, true
}

func FindByTypeOrg(tokenType, org string) (string, Entry, bool) {
	f := load()
	for name, entry := range f.Contexts {
		if entry.TokenType == tokenType && entry.Org == org {
			return name, entry, true
		}
	}
	return "", Entry{}, false
}

func SetDefaultDomain(domain string) {
	f := load()
	f.DefaultDomain = domain
	if f.Contexts == nil {
		f.Contexts = map[string]Entry{}
	}
	if entry, ok := f.Contexts["default"]; ok {
		entry.Domain = domain
		f.Contexts["default"] = entry
	}
	write(f)
}

func PurgeDomain(domain string) []string {
	f := load()
	removed := []string{}
	if f.Contexts == nil {
		f.Contexts = map[string]Entry{}
	}

	for name, entry := range f.Contexts {
		if name == "default" {
			continue
		}
		if entry.Domain == domain {
			removed = append(removed, EntryEnvNames(entry)...)
			delete(f.Contexts, name)
		}
	}

	if f.DefaultDomain == domain {
		removed = append(removed, TokenEnvDefault(domain))
	}

	write(f)
	return uniqueStrings(removed)
}

func PurgeAll() []string {
	f := load()
	removed := []string{}
	if f.Contexts == nil {
		f.Contexts = map[string]Entry{}
	}

	for name, entry := range f.Contexts {
		if name == "default" {
			continue
		}
		removed = append(removed, EntryEnvNames(entry)...)
		delete(f.Contexts, name)
	}

	if f.DefaultDomain != "" {
		removed = append(removed, TokenEnvDefault(f.DefaultDomain))
	}

	write(f)
	return uniqueStrings(removed)
}

func Load() File {
	return load()
}

func List() File {
	f := load()
	if f.Contexts == nil || len(f.Contexts) == 0 {
		fmt.Println("[tfcred] no contexts configured")
		return f
	}

	fmt.Println("[tfcred] configured contexts:")
	for name, entry := range f.Contexts {
		if name == "default" {
			continue
		}
		if entry.TokenType == "default" {
			fmt.Printf("  - %s (default, domain=%s)\n", name, entry.Domain)
			continue
		}
		fmt.Printf("  - %s (type=%s org=%s domain=%s)\n", name, entry.TokenType, entry.Org, entry.Domain)
	}
	return f
}

func load() File {
	storageFile := getStoragePath()
	b, err := os.ReadFile(storageFile)
	if err != nil {
		return File{Contexts: map[string]Entry{}}
	}
	var f File
	_ = json.Unmarshal(b, &f)
	return f
}

func write(f File) {
	storageFile := getStoragePath()
	b, _ := json.MarshalIndent(f, "", " ")
	_ = os.WriteFile(storageFile, b, 0644)
}

func DeleteEnvVar(name string) error {
	registryCmd := fmt.Sprintf("[Environment]::SetEnvironmentVariable('%s', $null, 'User')", name)
	cmd := exec.Command("powershell", "-Command", registryCmd)
	return cmd.Run()
}

func TokenEnvBase(domain string) string {
	return fmt.Sprintf("TF_TOKEN_%s", sanitizeDomain(domain))
}

func TokenEnvDefault(domain string) string {
	return TokenEnvBase(domain)
}

func TokenEnvName(domain, tokenType, org string) string {
	base := TokenEnvBase(domain)
	if tokenType == "" || tokenType == "default" {
		return base
	}
	return fmt.Sprintf("%s_%s_%s", base, sanitizeTokenComponent(tokenType), sanitizeTokenComponent(org))
}

func EntryEnvNames(entry Entry) []string {
	if entry.TokenType == "default" {
		return []string{TokenEnvDefault(entry.Domain)}
	}
	return []string{TokenEnvName(entry.Domain, entry.TokenType, entry.Org)}
}

func sanitizeDomain(domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	domain = strings.ReplaceAll(domain, ".", "_")
	domain = strings.ReplaceAll(domain, "-", "_")
	return domain
}

func sanitizeTokenComponent(input string) string {
	component := strings.TrimSpace(strings.ToLower(input))
	component = strings.ReplaceAll(component, "-", "_")
	return component
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	list := []string{}
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		list = append(list, item)
	}
	return list
}
