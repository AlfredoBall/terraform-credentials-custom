# tfcred — Terraform Credential Context Manager

`tfcred` is a lightweight, zero-dependency development tool designed to manage multiple **Terraform Cloud/Enterprise authentication contexts** on a single machine. By driving a custom Terraform CLI credential helper, `tfcred` allows developers to cleanly hot-swap between multiple organizations, teams, and token scopes instantly without continuous authentications, environment juggling, or profile pollution.

---

## Why This Exists

By default, the native Terraform CLI configuration dictates a rigid, single-token mapping for `app.terraform.io`. Juggling multi-tenant environments usually requires constant terminal exports, custom script wrappers, or manual logins when switching across:
* **Multiple Organizations** (e.g., Core Platform vs Network Engineering)
* **Varying Privilege Tiers** (User Tokens vs Team Tokens vs Organization Tokens)
* **Local Workspace Paradigms** (Local development states vs CI execution profiles)

`tfcred` shifts authentication scopes into standard, isolated terminal process environment vectors seamlessly.

---

## Architecture Overview

```text
       Terraform CLI Execution Event
                    ↓
   Native Credentials Helper Call Interface
                    ↓
        Reads: \$env:TF_CONTEXT 
                    ↓
     tfcred Deterministic Resolution Engine
                    ↓
  Maps Target Token Environment Target Keys:
  - Default: TF_TOKEN_app_terraform_io
  - Scoped:  TF_TOKEN_app_terraform_io_<type>_<org>
```

---

## Key Concepts

### 1. TF_CONTEXT
A runtime environment string assigned to the active shell session specifying the current scope. Valid structures evaluate to:
* `default`
* `team:networking`
* `user:platform`
* `org:control-plane`

**Taxonomy Mechanics:** `[Type]:[Organization Name]`

### 2. Token Resolution Mapping
Tokens are read strictly from environment keys matching the configuration metadata schema:
* **Global Default:** `TF_TOKEN_app_terraform_io`
* **Context Scoped:** `TF_TOKEN_app_terraform_io_<type>_<org>`
* *Example Target Example:* `TF_TOKEN_app_terraform_io_team_networking`

---

## Installation

### Method A: Via Windows Package Manager (WinGet)
Once published upstream, install the complete context management package directly via the official community repository:
```powershell
winget install AlfredoBall.tfcred
```

### Method B: Manual Go Compilation From Source
If you are building or testing the tool matrix locally from your source repository root:
```powershell
# 1. Compile both the CLI application and core credential helper binaries
go build -o ./dist/tfcred.exe ./cmd/terraform-credentials-custom/tfcred
go build -o ./dist/terraform-credentials-custom.exe ./cmd/terraform-credentials-custom

# 2. Register the helper execution configuration hook inside your global Terraform profile
.\scripts\install.ps1
```
*Note: This automatically configures a secure `credentials_helper "custom" {}` payload vector inside your primary `%APPDATA%\terraform.rc` configuration file.*

---

## CLI Usage Reference

### Context Initialization
Prepare your workspace runtime configuration state:
```powershell
tfcred init
```

### Profile Registry Management
Add a brand new tracking schema context boundary:
```powershell
tfcred add --context control-plane --org networking --token-type team
```
Alternatively, pass your active token payload vector directly during creation to set up runtime variables:
```powershell
tfcred add --context control-plane --org networking --token-type team --token <your_tfc_token_here>
```

### Active Shell Swapping
Switch the active shell environment target cleanly:
```powershell
tfcred switch control-plane
```
*(This sets your system process state machine pointer directly to `$env:TF_CONTEXT="team:networking"`)*

### System Diagnostics & Diagnostics State Machine

| Command | Action Scope | Output Details |
| :--- | :--- | :--- |
| `tfcred current` | Context Verification | Prints the active `$env:TF_CONTEXT` string value. |
| `tfcred whoami` | Account Introspection | Outputs active organizational mapping, token types, and context tracking metadata. |
| `tfcred env` | Configuration Inspection | Dumps variable alignments. Append `--json` for clean programmatic scripting arrays. |
| `tfcred doctor` | Workspace Sanity Check | Verifies file integrity, path formats, and highlights missing token variable keys. |
| `tfcred explain` | Engine Execution Trace | Traces resolution logic. Use `--trace` or `--json` to inspect helper execution logic. |

---

## Complete Usage Workflow Example

```powershell
# Initialize empty contexts storage configuration registry
tfcred init

# Configure specialized scoped automation tracking target entries
tfcred add --context network-dev --org enterprise-net --token-type team --token secret_tfc_abc123

# Activate your newly configured execution context 
tfcred switch network-dev

# Execute native workflow safely — the custom helper resolves the target token transparently
terraform plan
```

---

## Storage & Security Boundary Mechanics

* **`contexts.json` Structure:** Contexts are logged locally within a configuration file containing strictly structural profile organization pointers and token type definitions. 
* **Zero Secret Storage:** `contexts.json` **never** records, caches, or writes cryptographic tokens or sensitive credential fields directly to disk. 
* **Process Lifetime Rule:** Real credential strings reside exclusively within your environment variable block.

---

## Troubleshooting & Debugging

If your active Terraform execution throws unexpected authentication errors, run these validation routines to catch missing variables instantly:
```powershell
tfcred explain --trace
tfcred doctor --verbose
tfcred env --json
```

---

## License

Distributed under the terms of the official **MIT License**. Check the `LICENSE` file for additional policy disclosures.
