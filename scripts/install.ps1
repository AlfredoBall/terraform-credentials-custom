$ErrorActionPreference = "Stop"

Write-Host "[tfcred] installing Terraform credential helper..." -ForegroundColor Cyan

# ---- 1. detect config mode ----
$useCustomConfig = $env:TF_CLI_CONFIG_FILE -ne $null

if ($useCustomConfig) {
    $configPath = $env:TF_CLI_CONFIG_FILE

    if (-not ($configPath -match "\.tfrc$")) {
        Write-Host "[tfcred][warn] TF_CLI_CONFIG_FILE should use .tfrc extension" -ForegroundColor Yellow
    }

    Write-Host "[tfcred] using custom config: $configPath" -ForegroundColor Green
}
else {
    $configPath = "$env:APPDATA\terraform.rc"
    Write-Host "[tfcred] using default config: $configPath" -ForegroundColor Green
}

# ---- 2. ensure directory exists ----
$configDir = Split-Path $configPath
if (!(Test-Path $configDir)) {
    New-Item -ItemType Directory -Path $configDir | Out-Null
}

# ---- 3. write terraform CLI config safely ----
$configContent = @'
credentials_helper "custom" {}
'@

if (Test-Path $configPath) {
    $existing = Get-Content $configPath -Raw
    if ($existing -match "credentials_helper") {
        Write-Host "[tfcred] credentials_helper already configured" -ForegroundColor Yellow
    } else {
        Add-Content -Path $configPath -Value "`n$configContent"
        Write-Host "[tfcred] updated config file" -ForegroundColor Green
    }
}
else {
    Set-Content -Path $configPath -Value $configContent
    Write-Host "[tfcred] created config file" -ForegroundColor Green
}

# ---- 4. install PowerShell wrapper helper ----
$wrapperDir = Join-Path $HOME "Documents\PowerShell"
$wrapperFileName = "tfcred-profile.ps1"
$wrapperPath = Join-Path $wrapperDir $wrapperFileName

if (!(Test-Path $wrapperDir)) {
    New-Item -ItemType Directory -Path $wrapperDir | Out-Null
}

$sourceWrapper = Join-Path (Split-Path -Parent $MyInvocation.MyCommand.Path) "profile.ps1"
Copy-Item -Path $sourceWrapper -Destination $wrapperPath -Force
Write-Host "[tfcred] installed PowerShell wrapper script to $wrapperPath" -ForegroundColor Green

$profilePath = $PROFILE.CurrentUserAllHosts
if (!(Test-Path $profilePath)) {
    New-Item -ItemType File -Path $profilePath -Force | Out-Null
}

$importLine = ". '$wrapperPath'"
$profileContent = Get-Content -Path $profilePath -Raw -ErrorAction SilentlyContinue
if ($profileContent -notmatch [regex]::Escape($importLine)) {
    Add-Content -Path $profilePath -Value "`n$importLine"
    Write-Host "[tfcred] added wrapper import to PowerShell profile: $profilePath" -ForegroundColor Green
} else {
    Write-Host "[tfcred] PowerShell profile already imports tfcred wrapper" -ForegroundColor Yellow
}

Write-Host "[tfcred] install complete" -ForegroundColor Green