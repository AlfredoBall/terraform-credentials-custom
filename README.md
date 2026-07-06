# tfcred — Terraform Credential Context Manager

`tfcred` is a local development tool that manages **Terraform Cloud authentication contexts** by controlling which `TF_TOKEN_*` environment variable Terraform resolves at runtime.

It works alongside a Terraform CLI credential helper to provide **multi-organization, multi-token switching without re-login or manual environment juggling**.

---

## Why this exists

Terraform Cloud authentication is simple in theory:

- Terraform reads a credential helper
- The helper returns a token for `app.terraform.io`

In practice, things get messy when you have:

- multiple Terraform Cloud organizations
- multiple token types (user, team, org)
- multiple environments per machine
- CI vs local differences

`tfcred` provides a deterministic way to manage that complexity.

---

## Architecture overview


Terraform CLI
↓
credentials helper (this project)
↓
TF_CONTEXT
↓
tfcred resolution engine
↓
TF_TOKEN_app_terraform_io_* env vars


---

## Key concepts

### TF_CONTEXT

A runtime string that defines the active credential scope:


default
team:networking
user:platform
org:control-plane


Parsed as:


Type: team | user | org | default
Org: Terraform Cloud organization


---

### Token resolution

Terraform tokens are resolved via environment variables:


TF_TOKEN_app_terraform_io
TF_TOKEN_app_terraform_io_<type>_<org>


Example:


TF_TOKEN_app_terraform_io_team_networking


---

## Installation

### 1. Build

```bash
go build ./cmd/tfcred
2. Install credential helper
.\scripts\install.ps1

This configures:

credentials_helper "custom" {}

in your Terraform CLI config.

Usage
Initialize context store
tfcred init
Add a context
tfcred add --context control-plane --org networking --token-type team

Optionally set token:

tfcred add --context control-plane --org networking --token-type team --token <token>
Switch context
tfcred switch control-plane

Sets:

TF_CONTEXT=team:networking
Current context
tfcred current
Who am I (resolved identity)
tfcred whoami

Outputs:

context
org
token type mapping
Environment inspection
tfcred env

JSON mode:

tfcred env --json

Shows:

TF_CONTEXT
resolved token environment variable
token presence
Doctor (diagnostics)
tfcred doctor

Verbose:

tfcred doctor --verbose

Checks:

TF_CONTEXT format validity
contexts.json integrity
missing environment variables
resolution state
Explain (resolution trace)
tfcred explain

JSON:

tfcred explain --json

Trace mode:

tfcred explain --trace

Shows:

parsed context
resolved TF token environment variable
credential helper mode (default vs scoped)
env state relevant to Terraform resolution
whether token is present
Credential helper behavior

When Terraform runs:

Terraform requests credentials for app.terraform.io
The custom credential helper is invoked
The helper reads TF_CONTEXT
The helper resolves:
Default mode
TF_TOKEN_app_terraform_io
Scoped mode
TF_TOKEN_app_terraform_io_<type>_<org>
Example workflow
tfcred init

tfcred add --context control-plane --org networking --token-type team
tfcred switch control-plane

terraform plan
Context file

Stored locally:

contexts.json

This file is only metadata:

does NOT store tokens
does NOT affect Terraform directly
used only by tfcred CLI
Limitations
Tokens are resolved from environment variables only
No automatic secret storage
No fallback chain between token types
No remote sync of contexts

This is intentional for predictability and security.

Debugging tips

If authentication fails:

tfcred explain --trace
tfcred doctor --verbose
tfcred env --json

License

MIT