#!/usr/bin/env pwsh
<#
.SYNOPSIS
  hades_checkpoint_note.ps1 - grava um checkpoint DURAVEL na memoria compartilhada.

.DESCRIPTION
  Existe porque "resumir so' no fim do chat" falha: se o chat morre, o resumo nunca sai.
  Este script e' chamado automaticamente pelo hook `.git/hooks/post-commit` a cada commit,
  e pode ser chamado a mao a qualquer momento. Ele MEDE o repositorio e anexa uma entrada
  ao log de checkpoints da memoria compartilhada.

  Idempotente: se a ultima entrada do log ja' tem o mesmo commit e o mesmo motivo, nao duplica.

  ASCII PURO pelo mesmo motivo do hades_resume.ps1: o Windows PowerShell 5.1 le .ps1 SEM BOM
  como Windows-1252, e um acento quebraria o parser. A gravacao usa UTF-8 SEM BOM de proposito
  (nada de `Add-Content`, que no PS 5.1 insere BOM e quebraria leitores estritos).

.EXAMPLE
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts/hades_checkpoint_note.ps1 -Motivo "pos-commit"
#>
[CmdletBinding()]
param(
  [string]$Memoria = 'C:\Users\DZ23\Documents\Obsidian Vault\Memoria-Compartilhada-IA',
  [string]$Motivo = 'post-commit',
  [switch]$Silencioso
)

$ErrorActionPreference = 'Continue'

function Dizer($t) { if (-not $Silencioso) { $t } }

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $repo
$log = Join-Path $Memoria 'HADES-CHECKPOINT-LOG.md'

if (-not (Test-Path $Memoria)) { Dizer "MEMORIA AUSENTE: $Memoria"; exit 1 }

$sha = (git rev-parse HEAD 2>$null)
if (-not $sha) { Dizer 'SEM COMMIT (repositorio sem HEAD)'; exit 1 }
$curto = $sha.Substring(0, [Math]::Min(9, $sha.Length))
$branch = (git rev-parse --abbrev-ref HEAD 2>$null)
$sujo = @(git status --porcelain=v1).Count
$contagem = git rev-list --left-right --count origin/main...HEAD 2>$null
$atras = 'n/d'; $frente = 'n/d'
if ($contagem) {
  $p = @(($contagem -split '\s+') | Where-Object { $_ -ne '' })
  if ($p.Count -ge 2) { $atras = $p[0]; $frente = $p[1] }
}

$bom = 'n/d'; $release = 'n/d'
$estado = Join-Path $repo 'docs/mission/HADES_MASTER_STATE.json'
if (Test-Path $estado) {
  $b = [System.IO.File]::ReadAllBytes($estado)
  $bom = ($b.Length -ge 3 -and $b[0] -eq 0xEF -and $b[1] -eq 0xBB -and $b[2] -eq 0xBF)
  try {
    $d = (Get-Content -LiteralPath $estado -Raw -Encoding UTF8) | ConvertFrom-Json
    $release = $d.release_state
  } catch { $release = 'ILEGIVEL' }
}

$quando = (Get-Date).ToString('yyyy-MM-ddTHH:mm:sszzz')
$marcador = "commit=$curto motivo=$Motivo"

if (Test-Path $log) {
  $ultimas = Get-Content -LiteralPath $log -Tail 8 -Encoding UTF8
  if ($ultimas -match [regex]::Escape($marcador)) { Dizer "CHECKPOINT JA REGISTADO ($marcador)"; exit 0 }
}

$bloco = @()
if (-not (Test-Path $log)) {
  $bloco += '# HADES - LOG DE CHECKPOINTS AUTOMATICOS'
  $bloco += ''
  $bloco += '> Gerado por `scripts/hades_checkpoint_note.ps1` (hook `post-commit`). Cada linha e MEDICAO do Git, nao narracao.'
  $bloco += ''
}
$bloco += "## $quando - $branch @ $curto"
$bloco += "- motivo: $Motivo"
$bloco += "- commit: $sha"
$bloco += "- arquivos modificados nao commitados: $sujo"
$bloco += "- vs origin/main: atras=$atras a frente=$frente"
$bloco += "- estado canonico: BOM_UTF8=$bom release_state=$release"
$bloco += ''

$utf8SemBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::AppendAllText($log, (($bloco -join [Environment]::NewLine) + [Environment]::NewLine), $utf8SemBom)
Dizer "CHECKPOINT GRAVADO: $log -> $marcador"
