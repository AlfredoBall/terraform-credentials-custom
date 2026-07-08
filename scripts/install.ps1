$ErrorActionPreference = "Stop"

Write-Host "[tfcred] Installing Terraform credential helper..." -ForegroundColor Cyan

# ==============================================================================
# 1. ESTABLISH NATIVE PLUGINS LOCATION (Strict HashiCorp Location Rule)
# ==============================================================================
# The background helper MUST live in this explicit folder layout, or the CLI ignores it.
$terraformPluginDir = "$env:APPDATA\terraform.d\plugins"
if (!(Test-Path $terraformPluginDir)) {
    New-Item -ItemType Directory -Path $terraformPluginDir -Force | Out-Null
}

# Source pathing assumptions (maps out of your GoReleaser / Local Source structure)
$srcHelperBin = ".\dist\terraform-credentials-amiasea.exe"
$srcCliBin    = ".\dist\tfcred.exe"

# Destination for the user-facing CLI manager utility (Standard Tools Pathing)
$userToolsDir = "$env:USERPROFILE\bin"
if (!(Test-Path $userToolsDir)) {
    New-Item -ItemType Directory -Path $userToolsDir -Force | Out-Null
}

# Execute explicit file movement transactions safely if local source artifacts exist
if (Test-Path $srcHelperBin) {
    Copy-Item -Path $srcHelperBin -Destination "$terraformPluginDir\" -Force
    Write-Host "[tfcred] Registered 'terraform-credentials-amiasea.exe' in native plugins search path." -ForegroundColor Green
} else {
    Write-Host "[tfcred][warn] Compiled binary '$srcHelperBin' not found in dist. Skipping copy." -ForegroundColor Yellow
}

if (Test-Path $srcCliBin) {
    Copy-Item -Path $srcCliBin -Destination "$userToolsDir\" -Force
    Write-Host "[tfcred] Dropped interactive manager 'tfcred.exe' into user tools workspace: $userToolsDir" -ForegroundColor Green
    
    # Safely verify user path registration contains our custom tools directory boundary
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -notlike "*$userToolsDir*") {
        [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$userToolsDir", "User")
        $env:PATH = "$env:PATH;$userToolsDir"
        Write-Host "[tfcred] Appended $userToolsDir to User environment variables PATH." -ForegroundColor Green
    }
}

# ==============================================================================
# 2. DETECT CONFIGURATION MODE
# ==============================================================================
$useCustomConfig = $null -ne $env:TF_CLI_CONFIG_FILE

if ($useCustomConfig) {
    $configPath = $env:TF_CLI_CONFIG_FILE
    if (-not ($configPath -match "\.tfrc$")) {
        Write-Host "[tfcred][warn] TF_CLI_CONFIG_FILE should use .tfrc extension" -ForegroundColor Yellow
    }
    Write-Host "[tfcred] Using custom configuration hook file: $configPath" -ForegroundColor Green
} else {
    # Default Windows home directory configuration schema file
    $configPath = "$env:USERPROFILE\terraform.tfrc"
    Write-Host "[tfcred] Using default configuration file path: $configPath" -ForegroundColor Green
}

# Ensure destination parent tree folder layers are established
$configDir = Split-Path $configPath
if (!(Test-Path $configDir)) {
    New-Item -ItemType Directory -Path $configDir -Force | Out-Null
}

# ==============================================================================
# 3. WRITE TERRAFORM CLI CONFIGURATION SAFELY (Non-destructive Appending)
# ==============================================================================
# Updated configuration text payload mapping directly to your branded namespace identity
$configContent = @"

# --- Added by Amiasea tfcred Installer ---
credentials_helper "amiasea" {
  args = ["init"]
}
"@

if (Test-Path $configPath) {
    $existing = Get-Content $configPath -Raw
    
    # 1. Check if our exact tool is already there
    if ($existing -match 'credentials_helper\s+"amiasea"') {
        Write-Host "[tfcred] 'credentials_helper ""amiasea""' block is already configured inside $configPath" -ForegroundColor Yellow
    } 
    # 2. Prompt the user for an interactive choice if a conflicting helper exists
    elseif ($existing -match "credentials_helper") {
        Write-Host "[tfcred][warn] Conflict detected! Another credentials_helper block already exists in: $configPath" -ForegroundColor Yellow
        Write-Host "[tfcred][warn] Terraform only supports one global credentials_helper at a time." -ForegroundColor Yellow
        
        $choice = Read-Host "Would you like to comment out the old helper and register amiasea? [y/N]"
        if ($choice -match "^[yY](es)?$") {
            # Safely comment out the old credentials_helper block by prepending '#' to those lines
            $updatedContent = $existing -replace '(?m)^(.*credentials_helper.*$)', '# $1'
            $updatedContent = $updatedContent + "`n$configContent"
            
            Set-Content -Path $configPath -Value $updatedContent
            Write-Host "[tfcred] Commented out old helper and registered amiasea configuration successfully." -ForegroundColor Green
        } else {
            Write-Host "[tfcred][error] Installation aborted by user to preserve existing configuration." -ForegroundColor Red
            Exit 1
        }
    } 
    # 3. Safe to append if the file exists but has no helper blocks
    else {
        Add-Content -Path $configPath -Value "`n$configContent"
        Write-Host "[tfcred] Appended amiasea validation helper configuration successfully." -ForegroundColor Green
    }
} else {
    Set-Content -Path $configPath -Value $configContent
    Write-Host "[tfcred] Initialized fresh configuration mapping file." -ForegroundColor Green
}

Write-Host "[tfcred] Installation sequence finalized successfully." -ForegroundColor Green
