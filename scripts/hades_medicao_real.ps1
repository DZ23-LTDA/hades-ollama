# HADES - MEDICAO REAL COM MODELO LOCAL (sob demanda)  *** NAO E' GATE ***
#
# POR QUE ESTE SCRIPT EXISTE (medido 2026-10-15):
#   Os testes abaixo fazem **inferencia REAL** com um modelo local (Ollama):
#     - TestMatrizRealComModelosLocais        (matriz T2: 4 fixtures x 2 tentativas)
#     - TestT2RepeticaoEstatistica            (3 repeticoes x 2 tentativas = 6 geracoes)
#     - TestBenchmarkComModeloLocalOllama     (2 tentativas)
#     - TestFixerComModeloLocalConsertaNoLacoReal (2 reparos)
#   Cada tentativa tem orcamento proprio (4 min por padrao, `HADES_T2_ATTEMPT_TIMEOUT`).
#   => sao **6-10 geracoes de LLM**: minutos, e a duracao VARIa com a maquina e com o modelo.
#   Medido no mesmo host: a suite inteira passou em **549,506 s** numa rodada e estourou o limite de
#   25 min (**1507,556 s**) noutra. *Isso e' MEDICAO, nao verificacao* - e um gate que depende de quanto
#   o modelo local demora **inventa falhas**. Por isso o gate (`hades_local_gates.ps1`) os EXCLUI com
#   `-skip`, de forma declarada, e eles ficam aqui, executaveis sob demanda.
#
# USO (host sem PowerShell 7):
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_medicao_real.ps1
#
# EFEITOS: escreve os artefatos de medicao no repositorio (`docs/parity/HADES_T2_*.md`) - eles sao
# GERADOS pelos proprios testes. Nada e' commitado por este script.
#
# NAO IMPRIME NEM GRAVA SEGREDOS. ASCII puro (o PowerShell 5.1 le .ps1 sem BOM como Windows-1252).

$ErrorActionPreference = 'Continue'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $repo

$goroot = Join-Path $repo '_tmp\goroot1269\go'
if (-not (Test-Path (Join-Path $goroot 'bin\go.exe'))) {
  Write-Host "GOROOT pinado nao encontrado em $goroot - nao vou usar um Go de fora do repositorio." -ForegroundColor Red
  exit 2
}
$env:GOROOT = $goroot
$env:PATH = (Join-Path $goroot 'bin') + ';' + $env:PATH
$env:GOTOOLCHAIN = 'local'
$env:GOFLAGS = '-mod=mod'
$env:GOMODCACHE = Join-Path $repo '_tmp\gomodcache'
$env:GOCACHE = Join-Path $repo '_tmp\gocache1269'
$env:GOPATH = Join-Path $repo '_tmp\gopath'
$env:GOTMPDIR = Join-Path $repo '_tmp\gotmp'
$env:TMP = Join-Path $repo '_tmp\gotmp'
$env:TEMP = Join-Path $repo '_tmp\gotmp'
$env:APPDATA = Join-Path $repo '_tmp\appdata\Roaming'
$env:LOCALAPPDATA = Join-Path $repo '_tmp\appdata\Local'
# Orcamento curto para a SONDA de capacidade: *a sonda existe para nao gastar minutos descobrindo que nao
# da' para medir*. Sem isto, um modelo que nao gera custa 3 min por chamada (era o defeito do E-671).
if (-not $env:HADES_T2_PROBE_TIMEOUT) { $env:HADES_T2_PROBE_TIMEOUT = '10' }

$testes = 'TestMatrizRealComModelosLocais|TestT2RepeticaoEstatistica|TestBenchmarkComModeloLocalOllama|TestFixerComModeloLocalConsertaNoLacoReal'

Write-Host '=== MEDICAO REAL COM MODELO LOCAL (nao e gate) ==='
Write-Host ("inicio: " + (Get-Date -Format 'yyyy-MM-ddTHH:mm:sszzz'))
Write-Host 'Sonda de capacidade: um modelo que NAO gera e DECLARADO como NAO_MEDIDO (nunca "ok" por ausencia).'
Write-Host ''
$inicio = Get-Date
& go test ./internal/agent -run $testes -count=1 -v -timeout 60m 2>&1 | Tee-Object -FilePath (Join-Path $repo '_tmp\medicao_real.log')
$codigo = $LASTEXITCODE
$decorrido = (Get-Date) - $inicio
Write-Host ''
Write-Host ("fim: " + (Get-Date -Format 'yyyy-MM-ddTHH:mm:sszzz') + " - duracao " + $decorrido.ToString('hh\:mm\:ss') + " - exit " + $codigo)
Write-Host 'Log completo: _tmp\medicao_real.log'
Write-Host 'Os testes NAO falham por resultado ruim: a matriz e MEDICAO. Falham apenas se a medicao for impossivel.'
exit $codigo
