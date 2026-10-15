#!/usr/bin/env pwsh
<#
.SYNOPSIS
  hades_resume.ps1 - imprime o estado MEDIDO do HADES para retomada em qualquer chat/sessao.

.DESCRIPTION
  Existe porque "trocar de chat" nao pode custar o contexto do trabalho. Este script nao
  pergunta nada a nenhuma conversa: ele le o GIT e os documentos canonicos versionados e
  imprime o ponto de retomada.

  Regra de honestidade: a saida e MEDICAO. Se o `measured_commit` do estado divergir do HEAD
  real, quem manda e o HEAD - o estado e um registro datado, nao a verdade corrente.

  POR QUE ESTE ARQUIVO E ASCII PURO: o Windows PowerShell 5.1 le arquivos .ps1 SEM BOM como
  Windows-1252. Um acento ou travessao escrito em UTF-8 sem BOM vira mojibake e QUEBRA o
  parser (medido: "HADES - RETOMADA" virou "HADES a-euro-quote RETOMADA" e a string terminou
  no meio). As saidas sao: (a) gravar o .ps1 COM BOM, ou (b) manter o fonte em ASCII, que e o
  que se faz aqui - assim o checker de codificacao do repositorio (que acusa BOM) continua
  valendo para todos os arquivos, sem abrir dispensa.

.EXAMPLE
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts/hades_resume.ps1
  pwsh -File scripts/hades_resume.ps1 -NoFetch
#>
[CmdletBinding()]
param(
  [switch]$NoFetch,
  [string]$Memoria = 'C:\Users\DZ23\Documents\Obsidian Vault\Memoria-Compartilhada-IA'
)

$ErrorActionPreference = 'Continue'
try { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8 } catch {}

$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $repo

function Linha([string]$chave, $valor) { "{0,-26} {1}" -f $chave, $valor }

"=== HADES - RETOMADA (medido em $(Get-Date -Format 'yyyy-MM-ddTHH:mm:sszzz')) ==="
Linha 'repositorio' $repo

if (-not $NoFetch) { git fetch origin --prune --quiet 2>$null }

Linha 'branch' (git rev-parse --abbrev-ref HEAD)
Linha 'HEAD' (git rev-parse HEAD)
$up = git rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>$null
Linha 'upstream' $up
$mainFull = git rev-parse origin/main 2>$null
Linha 'origin/main' $mainFull
$contagem = git rev-list --left-right --count origin/main...HEAD 2>$null
if ($contagem) {
  $partes = @(($contagem -split '\s+') | Where-Object { $_ -ne '' })
  Linha 'vs main (atras/a frente)' ("atras={0} a frente={1}" -f $partes[0], $partes[1])
}

$alterados = @(git status --porcelain=v1)
Linha 'arquivos modificados' $alterados.Count
if ($alterados.Count -gt 0) { $alterados | Select-Object -First 20 | ForEach-Object { "   $_" } }
Linha 'stashes' (@(git stash list).Count)
$worktrees = @(git worktree list)
if ($worktrees.Count -gt 1) { Linha 'worktrees' $worktrees.Count; $worktrees | ForEach-Object { "   $_" } }

# --- estado canonico: BOM quebra leitor estrito, entao isso e' MEDIDO, nao suposto
$estado = Join-Path $repo 'docs/mission/HADES_MASTER_STATE.json'
if (Test-Path $estado) {
  $bytes = [System.IO.File]::ReadAllBytes($estado)
  $temBom = ($bytes.Length -ge 3 -and $bytes[0] -eq 0xEF -and $bytes[1] -eq 0xBB -and $bytes[2] -eq 0xBF)
  Linha 'estado com BOM UTF-8' $temBom
  try {
    $d = (Get-Content -LiteralPath $estado -Raw -Encoding UTF8) | ConvertFrom-Json
    $measured = if ($d.measured_commit) { ([string]$d.measured_commit).Substring(0, [Math]::Min(9, ([string]$d.measured_commit).Length)) } else { 'n/d' }
    Linha 'release / stages / blocks' ("{0} | stages={1} | blockers={2}" -f $d.release_state, @($d.stages).Count, @($d.blockers).Count)
    Linha 'measured_commit (datado)' $measured
  } catch {
    Linha 'estado canonico' "NAO FOI POSSIVEL LER: $($_.Exception.Message)"
  }
} else {
  Linha 'estado canonico' 'AUSENTE'
}

$proxima = Join-Path $repo 'docs/mission/HADES_NEXT_ACTION.md'
if (Test-Path $proxima) {
  '--- proxima acao declarada (HADES_NEXT_ACTION.md) ---'
  Get-Content -LiteralPath $proxima -TotalCount 6 -Encoding UTF8 | ForEach-Object { "  $_" }
}

if (Test-Path $Memoria) {
  Linha 'memoria compartilhada' $Memoria
  Linha 'leia primeiro' 'MEMORY_INDEX.md e HADES-CURRENT-TASK.md'
  $nota = Join-Path $Memoria 'HADES-CURRENT-TASK.md'
  if (Test-Path $nota) { Linha 'nota viva atualizada em' ((Get-Item $nota).LastWriteTime.ToString('yyyy-MM-ddTHH:mm:sszzz')) }
} else {
  Linha 'memoria compartilhada' 'AUSENTE (caminho nao encontrado)'
}

'--- ferramentas locais (limite REAL de prova) ---'
foreach ($t in @('go', 'node', 'python', 'bash')) {
  $c = Get-Command $t -ErrorAction SilentlyContinue
  Linha $t $(if ($c) { $c.Source } else { 'AUSENTE - provavel so no runner do CI' })
}

'=== REGRA: confronte isto com o pedido atual. Memoria nao substitui medicao. ==='
