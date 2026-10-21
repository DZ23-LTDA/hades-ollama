# HADES -- prepara o ambiente para os testes Go neste host.
#
# PORQUE ISTO EXISTE (E-1097)
# ---------------------------
# Medido neste host: `go test ./server` dava **89 falhas**, das quais **87** com a mesma causa:
#
#     create runtime data root: mkdir C:\Users\DZ23\AppData\Roaming\ollama-agent\<hash>: Acesso negado.
#
# *O processo Go lancado deste harness e' CONFINADO e nao consegue criar directorias fora do workspace*
# (o `E-1087` diagnosticou-o). => *os testes que chamam `agent.NewRuntime` **sem `DataRoot`** caem no
# `%APPDATA%` real do utilizador e falham -- **mas passam no CI**, onde o `%APPDATA%` e' gravavel.*
#
# => ***o defeito nao e' do codigo dos testes: e' do AMBIENTE.*** *E a cura certa nao e' mudar 107 testes
# (a catraca dos testes hermeticos declara essa divida com tecto); e' **preparar o ambiente como o CI o
# tem**.*
#
# ***E O RESULTADO, MEDIDO, com este preparador:***
#     go test ./server          =>  0 falhas   (29.9 s)      [era 89]
#     go test ./internal/agent  =>  0 falhas   (327.8 s)
#
# Uso
# ---
#     . .\scripts\hades_go_test_env.ps1            # DOT-SOURCE: define as variaveis na sessao actual
#     go test ./server ./internal/agent -count=1
#
# *Tem de ser dot-sourced (`. .\scripts\...`), porque um script chamado com `&` corre num processo filho
# e as variaveis morrem com ele.*
#
# Este ficheiro e' ASCII puro de proposito: o `powershell.exe` 5.1 le `.ps1` sem BOM como Windows-1252.

$raiz = Split-Path -Parent $PSScriptRoot
$tmp = Join-Path $raiz '_tmp'

# *As tres raizes que o Go usa, e que o confinamento bloqueia fora do workspace.*
$appdata = Join-Path $tmp 'appdata'
$localappdata = Join-Path $tmp 'localappdata'
$webtemp = Join-Path $tmp 'webtemp'

foreach ($d in @($appdata, $localappdata, $webtemp)) {
    if (-not (Test-Path $d)) {
        New-Item -ItemType Directory -Path $d -Force | Out-Null
    }
}

# ***AS VARIAVEIS.*** *O `APPDATA` cobre o `os.UserConfigDir()` e o `resolveRuntimeDataRoot`; o
# `LOCALAPPDATA` cobre o `os.UserCacheDir()` (os locks do workspace); o `TMP`/`TEMP` cobrem o esbuild do
# Playwright e o GOTMPDIR.*
$env:APPDATA = $appdata
$env:LOCALAPPDATA = $localappdata
$env:TMP = $webtemp
$env:TEMP = $webtemp

# *E o GOTMPDIR, que o `go test` usa para os binarios temporarios.*
$gotmp = Join-Path $tmp 'gotmp'
if (-not (Test-Path $gotmp)) { New-Item -ItemType Directory -Path $gotmp -Force | Out-Null }
$env:GOTMPDIR = $gotmp

Write-Host "HADES GO TEST ENV: APPDATA=$appdata"
Write-Host "HADES GO TEST ENV: LOCALAPPDATA=$localappdata"
Write-Host "HADES GO TEST ENV: TMP=$webtemp"
Write-Host "HADES GO TEST ENV: GOTMPDIR=$gotmp"
Write-Host "HADES GO TEST ENV: pronto (use dot-source: . .\scripts\hades_go_test_env.ps1)"
