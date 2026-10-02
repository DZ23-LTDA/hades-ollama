#!/usr/bin/env pwsh
# Hades/Ollama Full local-first quickstart (Windows / PowerShell 5.1+ or 7+).
# Mirrors scripts/hades-quickstart.sh: builds the checked-out source, starts the
# backend + web UI, and optionally offers a recommended local model. It never
# downloads binaries or credentials silently.
[CmdletBinding()]
param(
  [switch]$NoModel,
  [switch]$Help
)

$ErrorActionPreference = "Stop"

$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$InstallDir = if ($env:OLLAMA_INSTALL_DIR) { $env:OLLAMA_INSTALL_DIR } else { Join-Path $Root "bin" }
$Binary = if ($env:OLLAMA_BINARY) { $env:OLLAMA_BINARY } else { Join-Path $InstallDir "ollama-full.exe" }
$HostAddr = if ($env:OLLAMA_HOST) { $env:OLLAMA_HOST } else { "127.0.0.1:11434" }
$UiPort = if ($env:HADES_UI_PORT) { $env:HADES_UI_PORT } else { "5173" }
$UiHost = if ($env:HADES_UI_HOST) { $env:HADES_UI_HOST } else { "127.0.0.1" }
$Model = if ($env:HADES_RECOMMENDED_MODEL) { $env:HADES_RECOMMENDED_MODEL } else { "qwen2.5:0.5b" }

function Show-Usage {
  @"
Uso: hades-quickstart.ps1 [-NoModel]

Builda e inicia Hades localmente:
  backend: http://$HostAddr
  UI:      http://${UiHost}:$UiPort

Variáveis: OLLAMA_INSTALL_DIR, OLLAMA_BINARY, OLLAMA_HOST,
HADES_UI_PORT, HADES_UI_HOST, HADES_UI_ALLOW_LAN, HADES_UI_AUTH_REQUIRED,
HADES_RECOMMENDED_MODEL.
"@
}

if ($Help) { Show-Usage; exit 0 }

function Assert-Command {
  param([string]$Name, [string]$Hint)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    Write-Error "$Name nao encontrado. $Hint"
    exit 1
  }
}

Assert-Command -Name go -Hint "Instale Go conforme o guia do projeto."
Assert-Command -Name node -Hint "Instale Node.js conforme o guia do projeto."
Assert-Command -Name npm -Hint "Instale Node.js conforme o guia do projeto."

if (-not (Test-Path (Join-Path $Root "go.mod")) -or
    -not (Test-Path (Join-Path $Root "app/ui/app/package.json"))) {
  Write-Error "Execute este script na arvore clonada do Hades."
  exit 1
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

Write-Host "[1/4] Buildando backend..."
Push-Location $Root
try { & go build -trimpath -o $Binary . } finally { Pop-Location }

Write-Host "[2/4] Instalando dependencias e buildando UI..."
Push-Location (Join-Path $Root "app/ui/app")
try {
  & npm ci --no-audit --no-fund
  & npm run build
} finally { Pop-Location }

function Test-Health {
  try {
    Invoke-WebRequest -UseBasicParsing -Uri "http://$HostAddr/api/version" -TimeoutSec 2 | Out-Null
    return $true
  } catch {
    return $false
  }
}

$backend = $null
try {
  Write-Host "[3/4] Iniciando backend em $HostAddr..."
  $env:OLLAMA_HOST = $HostAddr
  $backend = Start-Process -FilePath $Binary -ArgumentList "serve" -PassThru -NoNewWindow

  Write-Host "Aguardando o backend responder..."
  $healthy = $false
  for ($i = 0; $i -lt 40; $i++) {
    if (Test-Health) { $healthy = $true; break }
    Start-Sleep -Milliseconds 250
  }
  if (-not $healthy) {
    Write-Error "Backend nao ficou saudavel."
    exit 1
  }
  Write-Host "Backend saudavel em http://$HostAddr"

  if (-not $NoModel) {
    $hasModels = $false
    try {
      $tags = Invoke-RestMethod -UseBasicParsing -Uri "http://$HostAddr/api/tags" -TimeoutSec 5
      if ($tags.models) { $hasModels = @($tags.models).Count -gt 0 }
    } catch {
      $hasModels = $false
    }

    if (-not $hasModels) {
      # Only prompt on an interactive console; never download in a pipeline.
      if ([Environment]::UserInteractive -and -not [Console]::IsInputRedirected) {
        $answer = Read-Host "Nenhum modelo local encontrado. Baixar $Model agora? [s/N]"
        if ($answer -match '^[sS]$') {
          Write-Host "Baixando modelo recomendado: $Model"
          & $Binary pull $Model
        } else {
          Write-Host "Sem modelo: a UI abrira em estado acionavel; instale depois com: $Binary pull $Model"
        }
      } else {
        Write-Host "Nenhum modelo local encontrado; nao baixando em modo nao interativo. Use: $Binary pull $Model"
      }
    }
  }

  Write-Host "[4/4] Iniciando UI em http://${UiHost}:$UiPort ..."
  $env:HADES_UI_HOST = $UiHost
  if (-not $env:HADES_UI_ALLOW_LAN) { $env:HADES_UI_ALLOW_LAN = "false" }
  if (-not $env:HADES_UI_AUTH_REQUIRED) { $env:HADES_UI_AUTH_REQUIRED = "false" }

  Write-Host ""
  Write-Host "Hades esta disponivel em http://${UiHost}:$UiPort"
  Write-Host "Backend saudavel: http://$HostAddr"
  Write-Host "Se a tela informar que nao ha modelo, abra Configuracoes ou execute: $Binary pull $Model"
  Write-Host "Pressione Ctrl+C para encerrar backend e UI."

  Push-Location (Join-Path $Root "app/ui/app")
  try {
    # Run the UI in the foreground. --strictPort so the advertised URL is the
    # URL that actually serves the UI (no silent port fallback).
    & npm run preview -- --host $UiHost --port $UiPort --strictPort
  } finally {
    Pop-Location
  }
}
finally {
  if ($backend -and -not $backend.HasExited) {
    Stop-Process -Id $backend.Id -Force -ErrorAction SilentlyContinue
  }
}
