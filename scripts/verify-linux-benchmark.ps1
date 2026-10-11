<#
.SYNOPSIS
    Coleta os benchmarks do estagio 21 dentro do conteiner Linux.

.DESCRIPTION
    O relatorio de benchmark (docs/mission/HADES_BENCHMARK_21.md) declara uma limitacao honesta: os numeros
    foram medidos em UM host (Windows). Este script coleta os MESMOS benchmarks em Linux, para que a comparacao
    entre plataformas exista com dado em vez de suposicao.

    Ele nao e um gate de aprovacao: benchmark varia com a maquina, e transformar variacao de tempo em falha de
    CI seria repetir o defeito de E-155 (teste que mede a velocidade do host). Aqui o script MEDE e grava.

    O arquivo evita acentos de proposito: o Windows PowerShell 5.1 le .ps1 como ANSI e os caracteres
    multibyte quebravam o parser (mesma licao do verify-linux-parity.ps1).

.PARAMETER Image
    Imagem Go a usar (padrao golang:1.26.9).

.PARAMETER BenchTime
    Iteracoes por benchmark (padrao 50x).

.PARAMETER Count
    Repeticoes de cada benchmark (padrao 2).

.EXAMPLE
    powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-linux-benchmark.ps1
#>
[CmdletBinding()]
param(
    [string]$Image = 'golang:1.26.9',
    [string]$BenchTime = '50x',
    [int]$Count = 2,
    [int]$TimeoutMinutes = 20
)

$ErrorActionPreference = 'Continue'

$docker = (Get-Command docker -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source -ErrorAction SilentlyContinue)
if (-not $docker) { $docker = 'C:\Users\DZ23\AppData\Local\Programs\DockerDesktop\resources\bin\docker.exe' }
if (-not (Test-Path $docker)) { throw 'docker nao encontrado: instale o Docker ou ajuste o caminho neste script.' }

$repositoryRoot = Split-Path -Parent $PSScriptRoot
if (-not (Test-Path (Join-Path $repositoryRoot 'go.mod'))) {
    throw "go.mod nao encontrado em ${repositoryRoot}: rode o script de dentro do repositorio."
}

$serverVersion = & $docker info --format '{{.ServerVersion}}' 2>&1
if ($LASTEXITCODE -ne 0) {
    throw "daemon do Docker nao respondeu (${serverVersion}): inicie o Docker Desktop e rode de novo (nada foi executado)."
}

$moduleCache = Join-Path $repositoryRoot '_tmp\gomodcache'
if (-not (Test-Path $moduleCache)) { New-Item -ItemType Directory -Path $moduleCache -Force | Out-Null }

# Os benchmarks do estagio 21 + os CONTROLES do experimento de E-164 (planejador puro e escrita de arquivo).
# `-run '^$'` garante que NENHUM teste rode aqui.
$filtro = 'BenchmarkServiceAgentTurn|BenchmarkMemorySearchLiteral|BenchmarkMemorySearchSemantic|BenchmarkMemoryReembed|BenchmarkServiceAgentPlan|BenchmarkMemoryFileWrite'
$shellCommand = "go test ./internal/agent/ -run '^$' -bench '$filtro' -benchtime=$BenchTime -count=$Count -timeout ${TimeoutMinutes}m"

$logDir = Join-Path $repositoryRoot '_tmp'
if (-not (Test-Path $logDir)) { New-Item -ItemType Directory -Path $logDir -Force | Out-Null }
$logAtual = Join-Path $logDir 'linux-benchmark-last.log'

Write-Host "Linux benchmark: imagem=$Image"
Write-Host "Linux benchmark: comando=${shellCommand}"

$output = & $docker run --rm `
    --volume "${repositoryRoot}:/src" `
    --volume "${moduleCache}:/gomodcache" `
    --workdir /src `
    --env GOFLAGS=-mod=mod `
    --env GOTOOLCHAIN=local `
    --env GOPATH=/gopath `
    --env GOMODCACHE=/gomodcache `
    --env GOCACHE=/tmp/gocache `
    --env TMPDIR=/tmp `
    --env CGO_ENABLED=0 `
    $Image sh -c $shellCommand 2>&1
$exit = $LASTEXITCODE
$output | ForEach-Object { $_ }
$texto = ($output | Out-String)
$texto | Set-Content -Path $logAtual -Encoding UTF8

Write-Host ""
if ($exit -ne 0) {
    Write-Host "LINUX BENCHMARK: FALHOU (exit $exit) - as mensagens acima sao do Linux, nao do host."
    Write-Host "LINUX BENCHMARK: saida completa em ${logAtual}"
    exit $exit
}

# A tabela de resultados vai para a tela: numero sem comando e sem ambiente nao entra no relatorio.
Write-Host "LINUX BENCHMARK: resultados"
foreach ($linha in ($texto -split "`n")) {
    if ($linha -match '^(goos|goarch|cpu|Benchmark|ok)') {
        Write-Host ("  " + $linha.Trim())
    }
}
Write-Host "LINUX BENCHMARK: OK (exit 0) - saida completa em ${logAtual}"
exit 0
