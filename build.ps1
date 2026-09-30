<#
.SYNOPSIS
    Lightmail 1-Step Build Script
.DESCRIPTION
    Builds the ultra-lightweight Svelte frontend with bun and compiles
    the self-contained Lightmail binary.
#>
param(
    [string]$TargetOS = "linux",
    [string]$TargetArch = "amd64"
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "   Lightmail Automated Build Pipeline" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# 1. Build Svelte Frontend
Write-Host "`n[1/3] Building Svelte Gmail-style UI with bun..." -ForegroundColor Yellow
Push-Location "$RepoRoot\fe-svelte"
try {
    bun run build
} finally {
    Pop-Location
}

# 2. Copy static dist to Go embed directory
Write-Host "`n[2/3] Embedding frontend into Go HTTP server assets..." -ForegroundColor Yellow
$DistDest = "$RepoRoot\server\listen\http_server\dist"
if (Test-Path $DistDest) {
    Remove-Item -Recurse -Force $DistDest
}
New-Item -ItemType Directory -Path $DistDest -Force | Out-Null
Copy-Item -Recurse -Force "$RepoRoot\fe-svelte\dist\*" $DistDest

# 3. Build Go Single Binary
$OutputBinary = "lightmail_${TargetOS}_${TargetArch}"
if ($TargetOS -eq "windows") {
    $OutputBinary += ".exe"
}

Write-Host "`n[3/3] Compiling Go static binary ($TargetOS/$TargetArch) -> server\$OutputBinary..." -ForegroundColor Yellow
Push-Location "$RepoRoot\server"
try {
    $env:CGO_ENABLED = "0"
    $env:GOOS = $TargetOS
    $env:GOARCH = $TargetArch
    go build -ldflags "-s -w" -o $OutputBinary main.go
    $binSize = (Get-Item $OutputBinary).Length / 1MB
    Write-Host "`nBuild Succeeded! Binary: server\$OutputBinary ($([math]::Round($binSize, 2)) MB)" -ForegroundColor Green
} finally {
    Pop-Location
}
