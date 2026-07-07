# tfcred — Terraform credential context manager

`tfcred` is a lightweight helper for switching Terraform Cloud and Terraform Enterprise credentials by context. It is designed for workflows where you want to keep multiple organization- or team-scoped tokens available and select them explicitly through `TF_CONTEXT` instead of relying on a single global token.

## What this tool does

`tfcred` works with the Terraform credential helper flow by resolving a token from an environment variable that matches the active context. The tool does not use a fallback default token, and it does not read `credentials.tf.json`.

Instead, the model is:

- `tfcred init` sets the default Terraform domain once.
- `tfcred add` stores a context entry for a specific scope such as `team:acme` or `user:platform`.
- `tfcred switch <context>` sets `TF_CONTEXT` for the current shell/session.
- Terraform then resolves the matching token from the environment variable that `tfcred` registered.

## Core concepts

### Default domain
The default domain is only the Terraform hostname used for context entries when no explicit domain is provided. It is chosen during initialization and stored as configuration state.

Examples:
- `app.terraform.io`
- `app.eu.terraform.io`

### Contexts
A context is a named entry that maps to a token scope. The current context format is:

- `team:acme`
- `user:platform`
- `org:engineering`

The context name is stored locally in `contexts.json`, while the actual token remains in the environment and registry-backed variable storage.

## Installation

### From source
```powershell
go build -o ./dist/tfcred.exe ./cmd/terraform-credentials-custom/tfcred
go build -o ./dist/terraform-credentials-custom.exe ./cmd/terraform-credentials-custom
.\scripts\install.ps1
```

This installs the helper configuration into your Terraform profile so Terraform can route credentials through the custom helper.

## CLI reference

Run `tfcred` or `tfcred --help` to view the top-level command list.

### Initialize
```powershell
tfcred init --domain app.terraform.io
```

If you omit `--domain`, the CLI prompts you to choose one.

### Configure the default domain
```powershell
tfcred config --default-domain app.terraform.io
```

To show the current configured default domain:
```powershell
tfcred config --show
```

### Add a context
```powershell
tfcred add --context platform --org acme --token-type team --domain app.terraform.io --token <token>
```

The `--org` flag is required for non-default contexts. The `--token` flag is optional; if omitted, the tool stores the context metadata and expects the token to be set through the environment or registry workflow you use.

### List contexts
```powershell
tfcred list
```

### Switch context
```powershell
tfcred switch platform
```

### Remove or purge contexts
```powershell
tfcred remove platform
tfcred purge platform
tfcred purge --domain app.terraform.io
tfcred purge --all
```

### Inspect the current state
```powershell
tfcred current
tfcred status
tfcred whoami
tfcred env --json
tfcred env --show-secret
tfcred explain --trace
tfcred doctor
```

### Help
```powershell
tfcred --help
tfcred env --help
```

## Example workflow

```powershell
# Initialize with the default Terraform domain
tfcred init --domain app.terraform.io

# Add a context for a team token
tfcred add --context network-dev --org networking --token-type team --token <token>

# Activate it for the current shell/session
tfcred switch network-dev

# Run Terraform
terraform plan
```

## Storage and security

- `contexts.json` stores the named context metadata only.
- It does not store the actual token values.
- The real token is expected to be present in the environment/registry-backed variable used by the credential helper.

## License

Distributed under the MIT License. See [LICENSE](LICENSE) for details.
