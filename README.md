# tfcred — Terraform credential context manager
tfcred is a lightweight Terraform credential helper that makes it easy to switch between multiple Terraform Cloud / Terraform Enterprise tokens using named contexts.

See: https://developer.hashicorp.com/terraform/internals/credentials-helpers

Instead of relying on a single global token, you can maintain several organization or team-scoped tokens and activate them on demand via the tfcred switch <context> command.

## Features
Context-based credential switching

No fallback default token (explicit contexts only)

Secure token storage (Windows User environment variables / registry)

Metadata-only contexts.json (no tokens stored in files)

Full diagnostics and debugging commands

Works with both Terraform Cloud and Terraform Enterprise

## Installation

### Windows (WinGet)

Install `tfcred` using WinGet:

```powershell
winget install amiasea.tfcred
```

After installation, initialize the Terraform credentials helper:

```powershell
tfcred init
```

The init command registers the custom credential helper in your Terraform CLI configuration.

### From source

Build the binaries:

```powershell
go build -o ./dist/tfcred.exe ./cmd/terraform-credentials-amiasea/tfcred
go build -o ./dist/terraform-credentials-amiasea.exe ./cmd/terraform-credentials-amiasea
```

Run the installation script:

```powershell
.\scripts\install.ps1
```

## CLI Reference

Run tfcred or tfcred --help to see all commands.

## Core Commands

| Command | Description |
|---------|-------------|
| `version` | Show tfcred version |
| `init [--domain <domain>]` | Initialize storage and set default domain |
| `config --default-domain <d>` | Set the default Terraform domain |
| `config` | Show current configuration |
| `add --context <name> --org <org> [--token-type <type>] [--domain <domain>] --token <token> --switch` | Add or update a context |
| `list` | List all configured contexts |
| `switch <context>` | Switch active context |
| `remove <context>` | Remove context metadata |
| `purge <context>` | Remove context and delete its token(s) |
| `purge --domain <domain>` | Purge all contexts for a specific domain |
| `purge --all` | Purge everything |

## Inspection & Debugging

| Command | Description |
|---------|-------------|
| `current` | Print current context value |
| `status` | Show current context resolution status |
| `whoami` | Show detailed information about current context |
| `env [--json] [--show-secret] [--all]` | Display token environment variables |
| `explain [--json] [--trace]` | Explain how the current context is resolved |
| `doctor` | Run full system diagnostics |
| `orphaned` | Managed directory contexts not in sync with the file system |
| `clean-dirs` | Removes orphaned directories and associated context entries |

## Supported Values

Token types: user, team, org

Domains: app.terraform.io, app.eu.terraform.io, custom

## Example Workflow

### 1. Initialize

tfcred init --domain app.terraform.io

### 2. Add contexts

tfcred add --context platform --org acme --token-type team --token <your-team-token>

tfcred add --context personal --org myusername --token-type user --token <your-user-token>

### 3. Switch context

tfcred switch platform

### 4. Use Terraform normally

terraform plan

## Storage & Security

contexts.json stores only metadata (context name, org, type, domain).

Actual tokens are stored in the Windows registry (User scope) and environment variables.

Tokens are never written to disk in plain text.

## Troubleshooting

tfcred doctor                  # Run diagnostics

tfcred explain --trace         # Detailed resolution trace

tfcred env --show-secret       # Show current token (masked by default)

### If Terraform is not using the helper, re-run:

Run the [`install.ps1`](scripts/install.ps1) script:

```powershell
scripts\install.ps1 -Verbose
```

### License 

Distributed under the MIT License. See [LICENSE](LICENSE) for details.