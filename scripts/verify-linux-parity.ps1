<#
.SYNOPSIS
    Valida a suite Go no LINUX local, antes de empurrar para o CI.

.DESCRIPTION
    A licao que originou este script: uma injecao de falha "portatil" se comportou de forma
    diferente no Linux do CI (ENOTDIR ao ler um caminho sob um arquivo) e a suite passou no Windows
    e falhou la. Rodar a suite inteira localmente NAO bastou, porque a divergencia era do sistema de
    arquivos, nao do codigo.

    Este script roda os mesmos comandos do gate do CI dentro de um conteiner Linux, montando o cache
    de modulos local para nao rebaixar dependencias. Nada e publicado e nenhum servico externo e
    contatado.

    O arquivo evita acentos de proposito: o Windows PowerShell 5.1 le .ps1 como ANSI e os caracteres
    multibyte quebravam o parser.

.PARAMETER Image
    Imagem Go a usar. Se a primeira falhar, as alternativas sao tentadas na ordem.

.PARAMETER Packages
    Pacotes Go a testar. Padrao: ./internal/agent/ ./server/

.PARAMETER TestFilter
    Filtro opcional de teste (-run). Vazio roda a suite inteira.

.PARAMETER TimeoutMinutes
    Tempo maximo por execucao, em minutos.

.PARAMETER SkipRace
    Pula o detector de corrida. Use apenas quando o motivo for declarado.

.PARAMETER RacePackages
    Pacotes do detector de corrida. Padrao: ./internal/agent/ ./server/

.PARAMETER RaceFilter
    Filtro (-run) do detector de corrida. Vazio roda a suite INTEIRA (padrao): a corrida nao aparece so onde se
    procura, e a suite completa sob -race levou ~91s no container (agent ~80s + server ~11s).

.EXAMPLE
    powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-linux-parity.ps1
    powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-linux-parity.ps1 -TestFilter TestJSONStoreRollsBack
    powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-linux-parity.ps1 -SkipRace
#>
[CmdletBinding()]
param(
    [string[]]$Image = @('golang:1.26.9', 'golang:1.26', 'golang:latest'),
    [string[]]$Packages = @('./internal/agent/', './server/'),
    [string]$TestFilter = '',
    [int]$TimeoutMinutes = 30,
    [switch]$SkipRace,
    [string[]]$RacePackages = @('./internal/agent/', './server/'),
    [string]$RaceFilter = ''
)

# Erro explicito e sempre com throw/exit: a preferencia fica em Continue porque, no Windows
# PowerShell 5.1, qualquer mensagem que o docker escreve em STDERR ("Unable to find image ...")
# viraria erro TERMINANTE com Stop, e o script abortaria antes de conseguir tentar a proxima
# imagem ou interpretar o resultado do teste.
$ErrorActionPreference = 'Continue'

function Resolve-DockerExecutable {
    $candidates = @(
        (Get-Command docker -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source -ErrorAction SilentlyContinue),
        (Join-Path $env:LOCALAPPDATA 'Programs\DockerDesktop\resources\bin\docker.exe'),
        "$env:ProgramFiles\Docker\Docker\resources\bin\docker.exe"
    ) | Where-Object { $_ -and (Test-Path $_) }
    if (-not $candidates) {
        throw 'docker nao encontrado: instale o Docker ou ajuste o caminho neste script.'
    }
    return $candidates[0]
}

$docker = Resolve-DockerExecutable
$repositoryRoot = Split-Path -Parent $PSScriptRoot
if (-not (Test-Path (Join-Path $repositoryRoot 'go.mod'))) {
    throw "go.mod nao encontrado em ${repositoryRoot}: rode o script de dentro do repositorio."
}

# O daemon precisa responder: um conteiner que falha por daemon ausente seria um falso "a suite passou".
$serverVersion = & $docker info --format '{{.ServerVersion}}' 2>&1
if ($LASTEXITCODE -ne 0) {
    throw "daemon do Docker nao respondeu (${serverVersion}): inicie o Docker Desktop e rode de novo (nada foi executado)."
}

$moduleCache = Join-Path $repositoryRoot '_tmp\gomodcache'
if (-not (Test-Path $moduleCache)) {
    New-Item -ItemType Directory -Path $moduleCache -Force | Out-Null
}

$testArgs = ($Packages -join ' ')
$runClause = if ($TestFilter) { "-run '$TestFilter' " } else { '' }
$shellCommand = "go vet $testArgs && go test $testArgs -count=1 $runClause-timeout ${TimeoutMinutes}m"

Write-Host "Linux parity: imagens=$($Image -join ', ')"
Write-Host "Linux parity: raiz=${repositoryRoot}"
Write-Host "Linux parity: comando=${shellCommand}"

# A saida e SEMPRE gravada: uma falha de paridade nao se reproduz necessariamente na proxima execucao, e
# perder o nome do teste que falhou (como aconteceu em E-154) deixa o defeito sem causa. O log fica em
# _tmp\linux-parity-last.log e a execucao corrente tambem e preservada com carimbo de tempo.
$logDir = Join-Path $repositoryRoot '_tmp'
if (-not (Test-Path $logDir)) {
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
}
$logAtual = Join-Path $logDir 'linux-parity-last.log'
$logHistorico = Join-Path $logDir ("linux-parity-" + (Get-Date -Format 'yyyyMMdd-HHmmss') + ".log")

$lastOutput = ''
foreach ($candidate in $Image) {
    Write-Host ""
    Write-Host "=== tentando imagem $candidate ==="
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
        $candidate sh -c $shellCommand 2>&1
    $exit = $LASTEXITCODE
    $output | ForEach-Object { $_ }
    $lastOutput = ($output | Out-String)

    # DETECTOR DE CORRIDA (E-176/E-177). O gate acima roda com CGO_ENABLED=0, e `-race` EXIGE cgo - por isso
    # nenhuma execucao de CI jamais usou deteccao de corrida. O host tambem nao tem gcc, mas a imagem golang TEM,
    # entao o detector roda aqui. A suite INTEIRA roda sob -race (nao so onde se procura a corrida): no container
    # isso custou ~91s (agent ~80s + server ~11s), sem DATA RACE.
    if ($exit -eq 0 -and -not $SkipRace) {
        $raceClause = if ($RaceFilter) { "-run '$RaceFilter' " } else { '' }
        $rotulo = if ($RaceFilter) { $RaceFilter } else { 'suite inteira' }
        Write-Host ""
        Write-Host "=== detector de corrida: $rotulo (CGO_ENABLED=1) ==="
        $raceOutput = & $docker run --rm `
            --volume "${repositoryRoot}:/src" `
            --volume "${moduleCache}:/gomodcache" `
            --workdir /src `
            --env GOFLAGS=-mod=mod `
            --env GOTOOLCHAIN=local `
            --env GOPATH=/gopath `
            --env GOMODCACHE=/gomodcache `
            --env GOCACHE=/tmp/gocache-race `
            --env TMPDIR=/tmp `
            --env CGO_ENABLED=1 `
            $candidate sh -c "/usr/local/go/bin/go test -race $($RacePackages -join ' ') -count=1 $raceClause-timeout ${TimeoutMinutes}m" 2>&1
        $raceExit = $LASTEXITCODE
        $raceOutput | ForEach-Object { $_ }
        $lastOutput = $lastOutput + ($raceOutput | Out-String)

        if ($raceExit -ne 0 -or ($raceOutput | Out-String) -match 'DATA RACE') {
            $lastOutput | Set-Content -Path $logAtual -Encoding UTF8
            $lastOutput | Set-Content -Path $logHistorico -Encoding UTF8
            Write-Host ""
            Write-Host "LINUX PARITY: FALHOU no detector de corrida (exit $raceExit)"
            if (($raceOutput | Out-String) -match 'DATA RACE') {
                Write-Host "LINUX PARITY: DATA RACE detectada - a saida acima traz o rastro."
            }
            Write-Host "LINUX PARITY: saida completa em ${logAtual} (copia: ${logHistorico})"
            exit 1
        }
        Write-Host "LINUX PARITY: detector de corrida OK (sem DATA RACE)"
    }

    $lastOutput | Set-Content -Path $logAtual -Encoding UTF8
    $lastOutput | Set-Content -Path $logHistorico -Encoding UTF8

    if ($exit -eq 0) {
        Write-Host ""
        Write-Host "LINUX PARITY: OK com $candidate (exit 0)"
        Write-Host "LINUX PARITY: saida completa em ${logAtual}"
        exit 0
    }
    # Falha de imagem inexistente nao e falha de teste: tenta a proxima.
    if ($lastOutput -match 'manifest unknown|not found|No such image|pull access denied') {
        Write-Host "imagem $candidate indisponivel; tentando a proxima"
        continue
    }
    Write-Host ""
    Write-Host "LINUX PARITY: FALHOU com $candidate (exit $exit) - as mensagens acima sao do Linux, nao do host."
    # Os NOMES dos testes que falharam vao para a tela: sem eles o defeito fica sem causa.
    $nomes = @($lastOutput -split "`n" | Where-Object { $_ -match '^\s*--- FAIL: (\S+)' } | ForEach-Object { $_.Trim() })
    if ($nomes.Count -gt 0) {
        Write-Host "LINUX PARITY: testes que falharam:"
        $nomes | ForEach-Object { Write-Host "  $_" }
    } else {
        Write-Host "LINUX PARITY: nenhum '--- FAIL:' na saida; a falha foi de compilacao, vet ou imagem."
    }
    Write-Host "LINUX PARITY: saida completa em ${logAtual} (copia: ${logHistorico})"
    exit $exit
}

Write-Host ""
Write-Host "LINUX PARITY: nenhuma imagem disponivel ($($Image -join ', '))"
exit 2
