#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Installs PamElevationAgent service, directories, policy, and binaries.
#>
param(
    [string]$SourceDir = "",
    [string]$InstallDir = "${env:ProgramFiles}\PamAgent"
)

$ErrorActionPreference = "Stop"
$ServiceName = "PamElevationAgent"

if (-not $SourceDir) {
    $SourceDir = Join-Path $PSScriptRoot "..\build"
}
$SourceDir = Resolve-Path -Path $SourceDir

$agentExe = Join-Path $SourceDir "pam-agent.exe"
$requestExe = Join-Path $SourceDir "pam-request.exe"
if (-not (Test-Path $agentExe)) {
    Write-Error "Missing $agentExe — build the agent first (see agent\README.md)."
}
if (-not (Test-Path $requestExe)) {
    Write-Error "Missing $requestExe — build the agent first (see agent\README.md)."
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$policyDir = Join-Path $InstallDir "policy"
$logsDir = Join-Path $InstallDir "logs"
New-Item -ItemType Directory -Force -Path $policyDir | Out-Null
New-Item -ItemType Directory -Force -Path $logsDir | Out-Null

Copy-Item $agentExe (Join-Path $InstallDir "pam-agent.exe") -Force
Copy-Item $requestExe (Join-Path $InstallDir "pam-request.exe") -Force

$policySrc = Join-Path $PSScriptRoot "..\policy\policy.json"
if (Test-Path $policySrc) {
    Copy-Item $policySrc (Join-Path $policyDir "policy.json") -Force
} else {
    Write-Warning "policy\policy.json not found at repo root; create $policyDir\policy.json before using the agent."
}

$existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existing) {
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    $null = & sc.exe delete $ServiceName
    Start-Sleep -Seconds 2
}

$bin = Join-Path $InstallDir "pam-agent.exe"
$binForSvc = '"' + $bin + '"'
New-Service -Name $ServiceName `
    -BinaryPathName $binForSvc `
    -StartupType Automatic `
    -DisplayName "PAM Elevation Agent" | Out-Null
$null = & sc.exe description $ServiceName "PAM local elevation broker (Phase 1 — named pipe + local policy)"
Start-Service -Name $ServiceName

Write-Host "Installed to $InstallDir"
Write-Host "Service $ServiceName started (LocalSystem, Automatic)."
