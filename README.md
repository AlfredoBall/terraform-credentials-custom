# tfcred — Terraform credential context manager
tfcred is a lightweight Terraform credential helper that makes it easy to switch between multiple Terraform Cloud / Terraform Enterprise tokens using named contexts.

Instead of relying on a single global token, you can maintain several organization or team-scoped tokens and activate them on demand via the TF_CONTEXT environment variable.

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
winget install AlfredoBall.tfcred
```

After installation, initialize the Terraform credentials helper:

```powershell
tfcred init
```

The init command registers the custom credential helper in your Terraform CLI configuration.

### From source

Build the binaries:

```powershell
go build -o ./dist/tfcred.exe ./cmd/terraform-credentials-custom/tfcred
go build -o ./dist/terraform-credentials-custom.exe ./cmd/terraform-credentials-custom
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
| `config --show` | Show current configuration |
| `add --context <name> --org <org> [--token-type <type>] [--domain <domain>] [--token <token>]` | Add or update a context |
| `list` | List all configured contexts |
| `switch <context>` | Switch active context (sets `TF_CONTEXT`) |
| `remove <context>` | Remove context metadata |
| `purge <context>` | Remove context and delete its token(s) |
| `purge --domain <domain>` | Purge all contexts for a specific domain |
| `purge --all` | Purge everything |

## Inspection & Debugging

| Command | Description |
|---------|-------------|
| `current` | Print current `TF_CONTEXT` value |
| `status` | Show current context resolution status |
| `whoami` | Show detailed information about current context |
| `env [--json] [--show-secret] [--all]` | Display token environment variables |
| `explain [--json] [--trace]` | Explain how the current context is resolved |
| `doctor` | Run full system diagnostics |

## Supported Values

Token types: user (default), team, org

Domains: app.terraform.io, app.eu.terraform.io

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