param(
    [Parameter(Mandatory=$true)]
    [string]$Version
)
$ErrorActionPreference = "Stop"

# Configuration metadata mapping your system layout topology
$owner = "AlfredoBall"
$repo = "terraform-credentials-custom"
$packageId = "AlfredoBall.tfcred"

# Target GoReleaser zip archive layout structure location path
$archiveName = "terraform-credentials-custom_Windows_amd64.zip"
$distPath = ".\dist\$archiveName"

if (-not (Test-Path $distPath)) {
    throw "GoReleaser archive package not found at $distPath. Please execute your local goreleaser build cycle first."
}

$url = "https://github.com/$owner/$repo/releases/download/v$Version/$archiveName"
$hash = (Get-FileHash $distPath -Algorithm SHA256).Hash.ToLower()

# Setup nested output storage directory tree locations
$manifestDir = ".\.winget\manifests\a\$($owner)\tfcred\$($Version)"
if (Test-Path $manifestDir) { Remove-Item -Recurse -Force $manifestDir }
New-Item -ItemType Directory -Force -Path $manifestDir | Out-Null

# 1. Version file compilation build template
$versionManifest = @"
PackageIdentifier: $($packageId)
PackageVersion: $($Version)
ManifestType: version
ManifestVersion: 1.6.0
"@

# 2. Installer multi-binary layout setup configuration template
$installerManifest = @"
PackageIdentifier: $($packageId)
PackageVersion: $($Version)
InstallerLocale: en-US
Architecture: x64
InstallerType: zip
NestedInstallerType: portable
NestedInstallerFiles:
  - RelativeFilePath: tfcred.exe
    PortableCommandAlias: tfcred
  - RelativeFilePath: terraform-credentials-custom.exe
    PortableCommandAlias: terraform-credentials-custom
Installers:
  - Architecture: x64
    InstallerUrl: $($url)
    InstallerSha256: $($hash)
ManifestType: installer
ManifestVersion: 1.6.0
"@

# 3. Default localization metadata setup parameters
$localeManifest = @"
PackageIdentifier: $($packageId)
PackageVersion: $($Version)
PackageLocale: en-US
Publisher: $($owner)
PackageName: tfcred
License: MIT
ShortDescription: Terraform credential context manager
ManifestType: defaultLocale
ManifestVersion: 1.6.0
"@

# Save all validation manifest segments to their corresponding structural endpoints
$versionManifest | Out-File "$manifestDir\$($packageId).yaml" -Encoding UTF8 -NoNewline
$installerManifest | Out-File "$manifestDir\$($packageId).installer.yaml" -Encoding UTF8 -NoNewline
$localeManifest | Out-File "$manifestDir\$($packageId).locale.en-US.yaml" -Encoding UTF8 -NoNewline

Write-Host "[winget] Local multi-manifest configuration blocks created successfully at: $manifestDir" -ForegroundColor Green
