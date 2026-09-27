<#
.SYNOPSIS
    Build and install Ollama Full from a checked-out source tree.

.DESCRIPTION
    This script deliberately does not download Ollama upstream installers,
    binaries, models, or Docker images. It builds the current fork with Go.
    Signed release artifacts and Windows store distribution are not provided by
    this repository yet.

    Usage from the repository root:

        powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1

    Optional variables:

        $env:OLLAMA_INSTALL_DIR = "$env:LOCALAPPDATA\Programs\OllamaFull"
        $env:OLLAMA_BINARY_NAME = "ollama-full.exe"
#>

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$installDir = if ($env:OLLAMA_INSTALL_DIR) { $env:OLLAMA_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\OllamaFull" }
$binaryName = if ($env:OLLAMA_BINARY_NAME) { $env:OLLAMA_BINARY_NAME } else { "ollama-full.exe" }
$output = Join-Path $installDir $binaryName

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go is required to build Ollama Full from source."
}
if (-not (Test-Path (Join-Path $repoRoot "go.mod"))) {
    throw "Run this script from a checked-out Ollama Full repository."
}

New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Write-Host "Building Ollama Full from $repoRoot"
Push-Location $repoRoot
try {
    & go build -trimpath -o $output .
    if ($LASTEXITCODE -ne 0) { throw "Go build failed with exit code $LASTEXITCODE." }
} finally {
    Pop-Location
}

Write-Host "Installed source-built executable at $output"
Write-Host "Run it with: `"$output`" serve"
Write-Host "No signed installer, release binary, model download, or upstream service was used."
