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

Write-Host "[tfcred] install complete" -ForegroundColor Green