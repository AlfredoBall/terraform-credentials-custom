package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/zalando/go-keyring"
)

type Entry struct {
	Org       string `json:"org"`
	TokenType string `json:"tokenType"`
	Domain    string `json:"domain"`
}

type File struct {
	Contexts      map[string]Entry  `json:"contexts"`
	DefaultDomain string            `json:"defaultDomain"`
	Directories   map[string]string `json:"directories"` // Stores "C:\repos\infra" -> "net-prod"
}

func getStoragePath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "contexts.json"
	}
	targetDir := filepath.Join(appData, "tfcred")
	_ = os.MkdirAll(targetDir, 0o755)
	return filepath.Join(targetDir, "contexts.json")
}

// Binds a directory path to a specific context key persistently
func BindDirectory(dir string, contextKey string) error {
	f := load()
	if f.Directories == nil {
		f.Directories = make(map[string]string)
	}

	cleanDir := filepath.Clean(dir)
	f.Directories[cleanDir] = contextKey

	write(f)
	return nil
}

// Climbs up the folder tree to resolve which context owns the current folder
func ResolveContextByDir(startDir string) (string, bool) {
	f := load()
	if f.Directories == nil {
		return "", false
	}

	// Clean the path to ensure trailing slashes and casing are standardized
	current := filepath.Clean(startDir)

	// Exact match look up only—no climbing up to parent folders!
	if contextKey, exists := f.Directories[current]; exists {
		return contextKey, true
	}

	return "", false
}

func Init(defaultDomain string) {
	storageFile := getStoragePath()
	f := File{
		DefaultDomain: defaultDomain,
		Contexts:      map[string]Entry{},
		Directories:   map[string]string{},
	}
	if _, err := os.Stat(storageFile); err == nil {
		current := load()
		if current.DefaultDomain == "" {
			current.DefaultDomain = defaultDomain
		}
		if current.Contexts == nil {
			current.Contexts = map[string]Entry{}
		}
		if current.Directories == nil {
			current.Directories = map[string]string{}
		}
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

	f.Contexts[ctx] = Entry{
		Org:       org,
		TokenType: tokenType,
		Domain:    domain,
	}

	// Securely store token in Windows Credential Manager under a isolated target label
	if token != "" {
		secretTargetKey := fmt.Sprintf("tfcred:context:%s", ctx)
		err := keyring.Set("tfcred", secretTargetKey, token)
		if err != nil {
			fmt.Printf("[tfcred][error] Failed to store token securely in Windows Credential Manager: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[tfcred] Token securely vaulted in Windows Credential Manager.")
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

	// Clean up any stale directory maps pointing to this deleted context profile
	if f.Directories != nil {
		for dir, boundKey := range f.Directories {
			if boundKey == ctx {
				delete(f.Directories, dir)
			}
		}
	}

	// Purge the accompanying token from Windows Credential Manager
	secretTargetKey := fmt.Sprintf("tfcred:context:%s", ctx)
	_ = keyring.Delete("tfcred", secretTargetKey)

	write(f)
	return entry, true
}

func SetDefaultDomain(domain string) {
	f := load()
	f.DefaultDomain = domain
	if f.Contexts == nil {
		f.Contexts = map[string]Entry{}
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
		if entry.Domain == domain {
			secretTargetKey := fmt.Sprintf("tfcred:context:%s", name)
			_ = keyring.Delete("tfcred", secretTargetKey)

			removed = append(removed, name)
			delete(f.Contexts, name)
		}
	}

	// Clear directory bindings pointing to purged contexts
	if f.Directories != nil {
		for dir, boundKey := range f.Directories {
			if _, exists := f.Contexts[boundKey]; !exists {
				delete(f.Directories, dir)
			}
		}
	}

	write(f)
	return removed
}

func PurgeAll() []string {
	f := load()
	removed := []string{}
	if f.Contexts == nil {
		f.Contexts = map[string]Entry{}
	}
	for name := range f.Contexts {
		secretTargetKey := fmt.Sprintf("tfcred:context:%s", name)
		_ = keyring.Delete("tfcred", secretTargetKey)

		removed = append(removed, name)
		delete(f.Contexts, name)
	}

	f.Directories = map[string]string{}
	write(f)
	return removed
}

func Load() File {
	return load()
}

func List() File {
	f := load()
	if len(f.Contexts) == 0 {
		fmt.Println("[tfcred] no contexts configured")
		return f
	}

	fmt.Println("[tfcred] configured contexts:")
	// Use tabwriter to create a beautifully aligned terminal table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "  CONTEXT\tTYPE\tORGANIZATION\tDOMAIN")
	fmt.Fprintln(w, "  -------\t----\t------------\t------")

	for name, entry := range f.Contexts {
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", name, entry.TokenType, entry.Org, entry.Domain)
	}
	w.Flush()
	return f
}

func load() File {
	storageFile := getStoragePath()
	b, err := os.ReadFile(storageFile)
	if err != nil {
		return File{Contexts: map[string]Entry{}, Directories: map[string]string{}}
	}
	var f File
	_ = json.Unmarshal(b, &f)
	if f.Directories == nil {
		f.Directories = map[string]string{}
	}
	return f
}

func write(f File) {
	storageFile := getStoragePath()
	b, _ := json.MarshalIndent(f, "", " ")
	_ = os.WriteFile(storageFile, b, 0o644)
}

func sanitizeDomain(domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	domain = strings.ReplaceAll(domain, ".", "_")
	domain = strings.ReplaceAll(domain, "-", "_")
	return domain
}

// GetToken securely fetches the credential token from the Windows vault for a given context
func GetToken(ctx string) (string, error) {
	secretTargetKey := fmt.Sprintf("tfcred:context:%s", ctx)
	token, err := keyring.Get("tfcred", secretTargetKey)
	if err != nil {
		return "", err
	}
	return token, nil
}

// TokenVaultBase converts a domain name into a standardized secure vault namespace base string.
func TokenVaultBase(domain string) string {
	return fmt.Sprintf("tfcred:domain:%s", sanitizeDomain(domain))
}

// TokenVaultKey builds the unique, isolated storage tracking key identifier used by the Windows Vault.
func TokenVaultKey(domain, tokenType, org string) string {
	base := TokenVaultBase(domain)
	if tokenType == "" {
		return base
	}
	return fmt.Sprintf("%s:%s:%s", base, sanitizeTokenComponent(tokenType), sanitizeTokenComponent(org))
}

// EntryVaultKeys converts an explicit context entry into its formatted secure storage lookup tracking keys.
func EntryVaultKeys(entry Entry) []string {
	return []string{TokenVaultKey(entry.Domain, entry.TokenType, entry.Org)}
}

// sanitizeTokenComponent cleans formatting parameters (spaces, casing, dashes) for logging alignment
func sanitizeTokenComponent(input string) string {
	component := strings.TrimSpace(strings.ToLower(input))
	component = strings.ReplaceAll(component, "-", "_")
	return component
}

// CleanOrphanedDirectories scans the mapping table, checks the host filesystem,
// removes non-existent folders or dead context keys, and returns what it cleared.
func CleanOrphanedDirectories() ([]string, []string) {
	f := load()
	// Idiomatic Go: len() on a nil map evaluates to 0 safely, omitting the nil check
	if len(f.Directories) == 0 {
		return nil, nil
	}

	deadPaths := []string{}
	deadContexts := []string{}

	// Track keys to delete after loop completion to prevent concurrent map mutation crashes
	var pathsToDelete []string

	if f.Contexts == nil {
		f.Contexts = map[string]Entry{}
	}

	for dir, contextKey := range f.Directories {
		// Check 1: Does the folder still physically exist on the hard drive?
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			deadPaths = append(deadPaths, dir)
			pathsToDelete = append(pathsToDelete, dir)
			continue
		}

		// Check 2: Does the context profile it points to still exist in our configuration?
		if _, exists := f.Contexts[contextKey]; !exists {
			deadContexts = append(deadContexts, fmt.Sprintf("%s -> [%s]", dir, contextKey))
			pathsToDelete = append(pathsToDelete, dir)
		}
	}

	// Safely purge identified stale elements out of the active loop scope
	if len(pathsToDelete) > 0 {
		for _, dir := range pathsToDelete {
			delete(f.Directories, dir)
		}
		write(f)
	}

	return deadPaths, deadContexts
}

// CheckOrphanedDirectories runs a non-destructive filesystem scan over all
// registered bindings and returns any stale or unmapped directory paths.
func CheckOrphanedDirectories() ([]string, []string) {
	f := load()
	// Idiomatic Go: len() on a nil map evaluates to 0 safely, omitting the nil check
	if len(f.Directories) == 0 {
		return nil, nil
	}

	missingPaths := []string{}
	missingContexts := []string{}

	if f.Contexts == nil {
		f.Contexts = map[string]Entry{}
	}

	for dir, contextKey := range f.Directories {
		// 1. Check if the directory path still exists on the machine
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			missingPaths = append(missingPaths, dir)
			continue
		}

		// 2. Check if the target profile key has been deleted from contexts
		if _, exists := f.Contexts[contextKey]; !exists {
			missingContexts = append(missingContexts, fmt.Sprintf("%s -> [%s]", dir, contextKey))
		}
	}

	return missingPaths, missingContexts
}

// NukeStorage completely overwrites contexts.json with empty default maps
func NukeStorage() {
	emptyFile := File{
		Contexts:      map[string]Entry{},
		Directories:   map[string]string{},
		DefaultDomain: "app.terraform.io",
	}
	write(emptyFile)
}
