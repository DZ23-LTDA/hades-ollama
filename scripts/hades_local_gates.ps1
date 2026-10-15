#!/usr/bin/env pwsh
<#
.SYNOPSIS
  hades_local_gates.ps1 - roda LOCALMENTE os gates do CI que mais reprovam, com o MESMO binario.

.DESCRIPTION
  POR QUE ISTO EXISTE (E-663): o commit `fc7b71ccd` ficou VERMELHO no CI por **5 apontamentos do
  `golangci-lint`** em arquivos meus (gofumpt, intrange, staticcheck SA9003 e um campo orfao). A bateria local
  que eu vinha rodando (`_tmp/bateria_local.ps1`) espelha o job `Preserve Class A+ surfaces` - **python + node** -
  e **nao continha o `golangci-lint`**, que roda no job `test (ubuntu-latest)`.

  *Bateria que nao contem o gate do CI nao substitui o CI* - a licao ja' tinha custado o E-650, e custou de novo.
  Este script fecha a lacuna de forma VERSIONADA (nao em `_tmp`, que e' ignorado pelo Git).

  O QUE ELE RODA:
    1. `gofmt -l` nos alvos protegidos (tem de sair VAZIO);
    2. `go build ./...` (o gate de compilacao);
    3. `golangci-lint run ./internal/agent/...` com a **mesma versao do CI** (v2.13.0).

  LIMITE DECLARADO: o `golangci-lint` e' escopado a `./internal/agent/...` de proposito - no repositorio inteiro
  ele esbarra no typecheck do pacote `mlx`, que exige a toolchain C (medido em `_tmp/golangci-local.txt`). *Um
  gate escopado e' melhor que gate nenhum, desde que o escopo esteja ESCRITO.* Se o binario nao estiver presente,
  o script **nao declara aprovacao**: diz que o gate nao foi medido.

.EXAMPLE
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_local_gates.ps1
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_local_gates.ps1 -SemLint
#>
[CmdletBinding()]
param([switch]$SemLint, [switch]$SemTestes)

$ErrorActionPreference = 'Continue'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $repo

# Toolchain fixada DENTRO do repositorio (o `go` nao esta no PATH deste host).
$goroot = Join-Path $repo '_tmp\goroot1269\go'
if (Test-Path (Join-Path $goroot 'bin\go.exe')) { $env:GOROOT = $goroot; $env:PATH = "$goroot\bin;$env:PATH" }
$env:GOTOOLCHAIN = 'local'
$env:GOFLAGS = '-mod=mod'
$env:GOPATH = Join-Path $repo '_tmp\gopath'
$env:GOMODCACHE = Join-Path $repo '_tmp\gomodcache'
$env:GOCACHE = Join-Path $repo '_tmp\gocache1269'
$env:GOTMPDIR = Join-Path $repo '_tmp\gotmp'
$env:TMP = $env:GOTMPDIR
$env:TEMP = $env:GOTMPDIR
$env:APPDATA = Join-Path $repo '_tmp\appdata\Roaming'
$env:LOCALAPPDATA = Join-Path $repo '_tmp\appdata\Local'

$falhas = New-Object System.Collections.Generic.List[string]

'--- 1. gofmt -l internal/agent internal/multillm server/agent_routes.go'
$saidaFmt = & gofmt -l internal/agent internal/multillm server/agent_routes.go 2>&1
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("gofmt terminou com erro de parse (exit $LASTEXITCODE)")
  ($saidaFmt | Select-Object -First 10) | ForEach-Object { "    $_" }
} elseif ($saidaFmt) {
  $falhas.Add('arquivos fora do gofmt')
  ($saidaFmt | Select-Object -First 10) | ForEach-Object { "    $_" }
} else { '    (vazio) OK' }

'--- 2. go build ./...'
& go build ./... 2>&1 | Select-Object -First 10 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add('go build ./...') } else { '    OK' }

if ($SemLint) {
  '--- 3. golangci-lint PULADO por -SemLint (isto NAO e aprovacao do gate de lint)'
} else {
  '--- 3. golangci-lint run ./internal/agent/... ./server/... (mesma versao do CI)'
  $gl = Join-Path $repo '_tmp\golangci\v2130\golangci-lint.exe'
  if (-not (Test-Path $gl)) {
    $achado = Get-Command golangci-lint -ErrorAction SilentlyContinue
    if ($achado) { $gl = $achado.Source } else { $gl = $null }
  }
  if ($gl) {
    $env:GOLANGCI_LINT_CACHE = Join-Path $repo '_tmp\golangci-cache'
    & $gl run ./internal/agent/... ./server/... 2>&1 | Select-Object -Last 20 | ForEach-Object { "    $_" }
    if ($LASTEXITCODE -ne 0) { $falhas.Add("golangci-lint (exit $LASTEXITCODE)") } else { '    OK' }
  } else {
    $falhas.Add('golangci-lint AUSENTE: o gate de lint do CI NAO foi medido (v2.13.0 em _tmp/golangci/v2130)')
  }
}

if ($SemTestes) {
  '--- 4. go test PULADO por -SemTestes (isto NAO e aprovacao dos testes)'
} else {
  # E-670 — OS TESTES ENTRAM NO GATE LOCAL.
  #
  # *POR QUE ISTO EXISTE:* este script rodava `gofmt` + `go build` + `golangci-lint` e **nao rodava teste nenhum** -
  # e o passo que MAIS reprova no CI e' `go test`. *Uma bateria que nao contem o gate do CI nao substitui o CI*: foi
  # essa lacuna que deixou o lint passar (E-663/E-665) e que aqui fica fechada tambem para os testes.
  #
  # ESCOPO DECLARADO: os dois pacotes que o job decisivo do CI (`Go agentic and server gates`) cobre de forma mais
  # densa - `go test ./internal/agent` e `go test ./server`. *Um gate escopado e' melhor que gate nenhum, desde que
  # o escopo esteja ESCRITO.* Se o tempo incomodar, `-SemTestes` pula - e o script DIZ que pulou.
  #
  # `-timeout 150s`: **nao** e' para esconder falha - e' para a falha ser RAPIDA e ALTA. Medido (2026-10-15):
  # `TestMatrizComCorrecaoDeReferenciaDaVerdeEmTodas` (`harness_benchmark_matrix_test.go:46`) **trava** neste host
  # (conexao HTTP presa, `net/http.persistConn.writeLoop`) e so' morre no limite padrao de 10 min do `go test`.
  # Com 150s ele falha em 2m30s DIZENDO qual teste travou, em vez de consumir 10 minutos em silencio.
  '--- 4. go test ./internal/agent ./server -count=1 -timeout 150s (o passo que mais reprova no CI)'
  $saidaTeste = & go test ./internal/agent ./server -count=1 -timeout 150s 2>&1
  $saidaTeste | Select-Object -Last 12 | ForEach-Object { "    $_" }
  if ($LASTEXITCODE -ne 0) { $falhas.Add("go test ./internal/agent ./server (exit $LASTEXITCODE)") } else { '    OK' }
}

''
if ($falhas.Count -gt 0) {
  "GATES LOCAIS: $($falhas.Count) falha(s)"
  $falhas | ForEach-Object { "  - $_" }
  exit 1
}
$passos = @('gofmt', 'build')
if (-not $SemLint) { $passos += 'lint' }
if (-not $SemTestes) { $passos += 'testes' }
"GATES LOCAIS: $($passos -join ' + ') VERDES"
