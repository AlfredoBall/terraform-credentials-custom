# ==============================================================================
# tfcred Enhanced Behavior Shell Integration Function
# ==============================================================================
function tfcred {
    <#
    .SYNOPSIS
        Enhanced shell wrapper for tfcred to handle parent process environment synchronization.
    .DESCRIPTION
        Intercepts key commands (switch, add, remove, purge) to keep the current
        PowerShell session in sync with registry changes.
    #>

    if ($args.Count -eq 0) {
        & (Get-Command tfcred.exe -Syntax)
        return
    }

    $command = $args[0]

    switch ($command) {
        "switch" {
            if ($args.Length -lt 2) {
                Write-Host "[tfcred][error] usage: tfcred switch <context>" -ForegroundColor Red
                return
            }

            $tfContextStr = & (Get-Command tfcred.exe -Syntax) switch $args[1]
            if ($LASTEXITCODE -ne 0) { return }

            $tfContextStr = $tfContextStr.Trim()
            $env:TF_CONTEXT = $tfContextStr

            # Update .vscode/settings.json for this workspace
            $SettingsDir = ".\.vscode"
            $SettingsPath = ".\.vscode\settings.json"
            $TargetKey = "terminal.integrated.env.windows"

            if (!(Test-Path $SettingsDir)) { New-Item -ItemType Directory -Path $SettingsDir | Out-Null }

            if (Test-Path $SettingsPath) {
                try {
                    $Settings = Get-Content $SettingsPath -Raw | ConvertFrom-Json -ErrorAction Stop
                } catch {
                    $Settings = [PSCustomObject]@{}
                }
            } else {
                $Settings = [PSCustomObject]@{}
            }

            if ($null -eq $Settings.$TargetKey) {
                $Settings | Add-Member -MemberType NoteProperty -Name $TargetKey -Value ([PSCustomObject]@{})
            }

            $Settings.$TargetKey | Add-Member -MemberType NoteProperty -Name "TF_CONTEXT" -Value $tfContextStr -Force
            $Settings | ConvertTo-Json -Depth 10 | Out-File $SettingsPath -Encoding UTF8

            Write-Host "[tfcred] Switched context to: $tfContextStr" -ForegroundColor Green
            Write-Host "[tfcred] Updated workspace settings: $SettingsPath" -ForegroundColor Cyan
        }

        "add" {
            & (Get-Command tfcred.exe -Syntax) $args
            if ($LASTEXITCODE -eq 0) {
                # Refresh all TF_TOKEN_* variables into current session
                [Environment]::GetEnvironmentVariables("User").GetEnumerator() | 
                    Where-Object { $_.Name -like "TF_TOKEN_*" } | 
                    ForEach-Object { 
                        Set-Item "Env:\$($_.Name)" $_.Value 
                    }
                Write-Host "[tfcred] Token environment variables refreshed in current session." -ForegroundColor Green
            }
        }

        "remove" {
            & (Get-Command tfcred.exe -Syntax) $args
            # No auto-purge logic here anymore - remove only affects metadata
        }

        "purge" {
            & (Get-Command tfcred.exe -Syntax) $args

            if ($LASTEXITCODE -eq 0) {
                # Aggressively clean all TF_TOKEN_* from current session
                $cleaned = 0
                Get-ChildItem Env:\TF_TOKEN* | ForEach-Object {
                    Remove-Item "Env:\$($_.Name)" -ErrorAction SilentlyContinue
                    Write-Host "[tfcred] Cleared from session: $($_.Name)" -ForegroundColor Green
                    $cleaned++
                }
                if ($cleaned -eq 0) {
                    Write-Host "[tfcred] No TF_TOKEN_* variables found in current session." -ForegroundColor Cyan
                }
            }
        }

        default {
            # All other commands (list, status, doctor, init, etc.) pass through
            & (Get-Command tfcred.exe -Syntax) $args
        }
    }
}