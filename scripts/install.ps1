<#
.SYNOPSIS
    Build and install Ollama Full from a checked-out source tree, with upgrade,
    rollback and uninstall.

.DESCRIPTION
    This script deliberately does not download Ollama upstream installers,
    binaries, models, or Docker images. It builds the current fork with Go.
    Signed release artifacts and Windows store distribution are not provided by
    this repository yet.

    Usage from the repository root:

        powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1
        powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1 -Rollback
        powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1 -Uninstall
        powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1 -SkipBuild

    Optional variables:

        $env:OLLAMA_INSTALL_DIR = "$env:LOCALAPPDATA\Programs\OllamaFull"
        $env:OLLAMA_BINARY_NAME = "ollama-full.exe"

    UPGRADE E ROLLBACK
    Toda instalação sobre um binário existente é um UPGRADE: o binário anterior é
    preservado em "<nome>.previous" antes de ser substituído, junto com um manifesto
    que registra commit, data e hash SHA-256 do que foi instalado. O rollback restaura
    exatamente esse binário anterior — sem ele, "atualizar" seria uma operação sem
    volta. A desinstalação remove o binário e o backup e **preserva os dados do
    usuário** (modelos), porque remover modelos junto seria perda de dados.
#>
[CmdletBinding()]
param(
    [switch]$Rollback,
    [switch]$Uninstall,
    [switch]$SkipBuild,
    [string]$Ldflags = ""
)

$ErrorActionPreference = "Stop"
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$installDir = if ($env:OLLAMA_INSTALL_DIR) { $env:OLLAMA_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\OllamaFull" }
$binaryName = if ($env:OLLAMA_BINARY_NAME) { $env:OLLAMA_BINARY_NAME } else { "ollama-full.exe" }
$output = Join-Path $installDir $binaryName
$backup = Join-Path $installDir "$binaryName.previous"
$manifest = Join-Path $installDir "install-manifest.json"

function Get-FileHashOrNull([string]$path) {
    if (-not (Test-Path $path)) { return $null }
    return (Get-FileHash -Path $path -Algorithm SHA256).Hash
}

# DESINSTALAÇÃO: remove o binário e o backup; NÃO toca nos dados do usuário.
if ($Uninstall) {
    if (-not (Test-Path $installDir)) {
        Write-Host "Nada a desinstalar: $installDir não existe."
        exit 0
    }
    foreach ($target in @($output, $backup, $manifest)) {
        if (Test-Path $target) { Remove-Item -Path $target -Force }
    }
    Write-Host "Desinstalado de $installDir (binário, backup e manifesto removidos)."
    Write-Host "Os dados do usuário (modelos) foram preservados."
    exit 0
}

# ROLLBACK: restaura o binário anterior preservado pelo último upgrade.
if ($Rollback) {
    if (-not (Test-Path $backup)) {
        throw "Rollback indisponível: não existe binário anterior em $backup. Faça uma atualização antes."
    }
    if (Test-Path $output) {
        # O binário atual NÃO é descartado: ele vira o anterior, então o rollback é
        # reversível por natureza (dá para voltar e depois avançar de novo).
        Move-Item -Path $output -Destination "$output.rolledback" -Force
    }
    Move-Item -Path $backup -Destination $output -Force
    $restored = Get-FileHashOrNull $output
    Write-Host "Rollback concluído: binário anterior restaurado em $output"
    Write-Host "SHA-256 do binário restaurado: $restored"
    Write-Host "O binário que estava instalado foi preservado em $output.rolledback"
    exit 0
}

if (-not $SkipBuild) {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "Go is required to build Ollama Full from source."
    }
    if (-not (Test-Path (Join-Path $repoRoot "go.mod"))) {
        throw "Run this script from a checked-out Ollama Full repository."
    }
}

New-Item -ItemType Directory -Force -Path $installDir | Out-Null

$action = "INSTALL"
if (Test-Path $output) {
    # UPGRADE: preserva o binário atual ANTES de substituir. É isto que torna a
    # atualização reversível.
    $action = "UPGRADE"
    Copy-Item -Path $output -Destination $backup -Force
    Write-Host "Atualização detectada: binário anterior preservado em $backup"
}

if (-not $SkipBuild) {
    Write-Host "Building Ollama Full from $repoRoot"
    Push-Location $repoRoot
    try {
        if ($Ldflags) {
            & go build -trimpath -ldflags $Ldflags -o $output .
        } else {
            & go build -trimpath -o $output .
        }
        if ($LASTEXITCODE -ne 0) { throw "Go build failed with exit code $LASTEXITCODE." }
    } finally {
        Pop-Location
    }
} elseif (-not (Test-Path $output)) {
    throw "-SkipBuild foi usado, mas não existe binário instalado em $output."
}

$commit = "desconhecido"
Push-Location $repoRoot
try {
    $commit = (& git rev-parse --short HEAD 2>$null)
} finally {
    Pop-Location
}
$record = [ordered]@{
    action       = $action
    binary       = $binaryName
    path         = $output
    commit       = "$commit"
    installed_at = (Get-Date).ToUniversalTime().ToString("o")
    sha256       = (Get-FileHashOrNull $output)
    previous     = (Get-FileHashOrNull $backup)
}
$record | ConvertTo-Json | Set-Content -Path $manifest -Encoding UTF8

Write-Host "$action concluído em $output"
Write-Host "SHA-256 instalado: $($record.sha256)"
Write-Host "Manifesto: $manifest"
Write-Host "Run it with: `"$output`" serve"
Write-Host "No signed installer, release binary, model download, or upstream service was used."
