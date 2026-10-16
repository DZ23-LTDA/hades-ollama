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
# E-561 - `-First 10` NUM COMANDO NATIVO E' UMA FALSA APROVACAO.
#
# *O DEFEITO:* `Select-Object -First N` fecha o pipeline apos N objectos e TERMINA o comando a montante.
# Medido nesta sessao: `& npx ... | Select-Object -First 30` matou o `npx` (exit -1). Se o `go build`
# produzir mais de 10 linhas, ele e' morto a meio e `$LASTEXITCODE` pode nao reflectir o build inteiro
# => o passo 2 diria OK sobre um build que NAO terminou. A linha do passo 4 ja' usava `-Last 12`; so' esta
# nao. Corrigido para `-Last 10`: le' tudo e mostra as ultimas 10 (a saida de um build e' curta).
& go build ./... 2>&1 | Select-Object -Last 10 | ForEach-Object { "    $_" }
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
  # `-skip`: **nao** e' para esconder falha - e' para o gate rodar o que e' GATE, nao o que e' MEDICAO.
  # A historia deste numero, toda medida (2026-10-15):
  #   1. a suite cheia morria no limite padrao de 10 min do `go test`;
  #   2. a causa era REAL e foi corrigida (E-671): `TestMatrizRealComModelosLocais` chamava um modelo que NAO gera
  #      (`llama3.2:3b`) e a sonda de disponibilidade dizia "disponivel" ⇒ 3 min por chamada;
  #   3. com o travamento corrigido, o gate passou uma vez (**549,506 s**) e falhou noutra (**1507,556 s** = o limite
  #      de 25 min), com quadro de `net/http` preso ⇒ *a duracao nao e' estavel porque depende de INFERENCIA REAL*;
  #   4. a causa: os testes opt-in de modelo local fazem trabalho real de LLM (benchmark 2 tentativas, repeticao
  #      **3x2 geracoes**, fixer 2 reparos, cada tentativa com orcamento de 4 min) ⇒ **6-10 geracoes** = minutos.
  #      *Isso e' MEDICAO, nao verificacao.* No CI (sem Ollama) eles pulam na hora - e' por isso que o CI e' verde.
  #   5. ⇒ o gate EXCLUI os testes de medicao real e **DECLARA** que os excluiu. Eles continuam executaveis sob
  #      demanda (ver `scripts/hades_medicao_real.ps1`), e sao eles que produzem os artefatos T2.
  # *Licao (tres hipoteses minhas cairam por medicao: vazamento de processos, travamento no filho, "so lentidao"):
  # um gate com escopo errado inventa falhas; um gate que nao roda teste nao substitui o CI (E-650).*
  $testesDeMedicaoReal = 'TestMatrizRealComModelosLocais|TestT2RepeticaoEstatistica|TestBenchmarkComModeloLocalOllama|TestFixerComModeloLocalConsertaNoLacoReal'
  '--- 4. go test ./internal/agent ./server -count=1 -timeout 25m -skip <medicao real com modelo local> (o passo que mais reprova no CI)'
  '    (excluidos por MEDICAO REAL de LLM, cada um declarando NAO_MEDIDO quando nao da para medir:)'
  '    TestMatrizRealComModelosLocais, TestT2RepeticaoEstatistica, TestBenchmarkComModeloLocalOllama, TestFixerComModeloLocalConsertaNoLacoReal'
  $saidaTeste = & go test ./internal/agent ./server -count=1 -timeout 25m -skip $testesDeMedicaoReal 2>&1
  $saidaTeste | Select-Object -Last 12 | ForEach-Object { "    $_" }
  if ($LASTEXITCODE -ne 0) { $falhas.Add("go test ./internal/agent ./server (exit $LASTEXITCODE)") } else { '    OK' }
}

# E-561 - O GATE LOCAL NAO VIA O CAMINHO DE PERSISTENCIA (`./app/...`).
#
# *POR QUE ISTO EXISTE:* o passo 4 cobre `./internal/agent` e `./server`. Mas o CI tambem roda
# `CGO_ENABLED=1 go test ./...` (dz23-agentic-quality.yaml, release.yaml), e `./...` inclui `./app/...`,
# onde vive `app/store` (SQLite via `github.com/mattn/go-sqlite3`, que exige CGO). O gate local NAO
# incluia `./app/...`: medido em 2026-10-15, o gate local estava VERDE enquanto `go test ./app/store`
# era 100% VERMELHO com `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work.
# This is a stub`, porque este host tem `CGO_ENABLED=0` e `gcc` AUSENTE. *Um gate que nao mede o
# caminho de persistencia nao pode aprova-lo* - e o `E-560` (seletor de modelo) vive nesse caminho.
#
# DECISAO: rodar `./app/...` QUANDO houver cgo; quando NAO houver, PULAR e DIZER que pulou, com o passo
# exacto de desbloqueio. Ausencia de cgo NAO se converte em sucesso: seria o silenciamento proibido.
$appRodou = $false
if ($SemTestes) {
  '--- 5. go test ./app/... PULADO por -SemTestes (isto NAO e aprovacao de app/store)'
} else {
  $cgo = (& go env CGO_ENABLED).Trim()
  $temCC = $null -ne (Get-Command gcc -ErrorAction SilentlyContinue)
  '--- 5. go test ./app/... -count=1 (persistencia: app/store usa SQLite via CGO)'
  if ($cgo -eq '1' -and $temCC) {
    $saidaApp = & go test ./app/... -count=1 -timeout 25m 2>&1
    $saidaApp | Select-Object -Last 12 | ForEach-Object { "    $_" }
    if ($LASTEXITCODE -ne 0) { $falhas.Add("go test ./app/... (exit $LASTEXITCODE)") } else { '    OK' }
    $appRodou = $true
  } else {
    "    PULADO: CGO_ENABLED=$cgo e gcc presente=$temCC - isto NAO e aprovacao de app/store"
    '    DESBLOQUEIO: instalar um compilador C (mingw-w64/w64devkit) e rodar com CGO_ENABLED=1.'
    '    NAO MEDIDO NESTE HOST: app/store (settings, chats e memoria em SQLite) nao pode ser verificado.'
  }
}

# E-989 - O MANIFESTO DO SNAPSHOT NAO TINHA CATRACA.
#
# *POR QUE ISTO EXISTE:* o ESTAGIO 22 item 3 mandou emitir manifestos, e o `E-988` criou o
# `scripts/hades_manifest.py`. Mas um ficheiro GERADO sem `--check` num gate **envelhece em
# silencio** - e' o mesmo defeito que o `E-987` curou no `hades_release_gate.py`, onde o
# relatorio citava um commit de ontem e ninguem notava.
#
# Aqui ele passa a ser conferido com o MESMO escopo do CI (`class-a-plus-integrity.yaml`), para
# que o defeito apareca localmente antes de aparecer no CI.
'--- 6. manifesto do snapshot (ESTAGIO 22 item 3) - drift entre o manifesto e a arvore'
$saidaManifesto = & python -X utf8 scripts/hades_manifest.py --check 2>&1
$saidaManifesto | Select-Object -Last 4 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_manifest.py --check (exit $LASTEXITCODE)") } else { '    OK' }

# E-991 - O INVENTARIO DE MARCA TAMBEM TEM GERADOR, E UM GERADOR SEM CATRACA ENVELHECE.
#
# *POR QUE ISTO EXISTE:* o `E-991` criou o `scripts/hades_brand_inventory.py` (ESTAGIO 22 /
# e 3). E a primeira versao desse gerador **media o proprio output**: ele escrevia um documento
# que contem as strings da marca e depois contava-as, o que dava 15 linhas diferentes entre
# duas geracoes. *Foi o `--check` do E-989 que o apanhou* - e e' por isso que ele tem de
# continuar a correr: um gerador sem catraca envelhece, e foi assim que o defeito nasceu.
'--- 7. inventario de marca (FASE-5) - drift entre o inventario e o repositorio'
$saidaMarca = & python -X utf8 scripts/hades_brand_inventory.py --check 2>&1
$saidaMarca | Select-Object -Last 3 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_brand_inventory.py --check (exit $LASTEXITCODE)") } else { '    OK' }

# E-992 - A CATRACA DO ASCII NOS .ps1. *Duas vezes escrevi um byte > 127 num comentario deste
# proprio ficheiro* - e as duas vezes porque confiei na disciplina em vez de num teste. O
# checker separa COMENTARIO (tecto declarado) de STRING (a que e' impressa ao utilizador, e
# que o powershell.exe 5.1 corrompe por ler o ficheiro como Windows-1252).
'--- 8. ASCII nos .ps1 - comentario com tecto, string com divida declarada'
$saidaAscii = & python -X utf8 scripts/hades_ascii_ps1_check.py 2>&1
$saidaAscii | Select-Object -Last 3 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_ascii_ps1_check.py (exit $LASTEXITCODE)") } else { '    OK' }

# *POR QUE ISTO EXISTE:* o `E-1007`. Quatro anomalias INTRA-REGISTO passaram todos os checkers
# nesta janela (E-995, E-999, E-1002, e o proprio E-1003) porque **todos contam linhas que
# comecam por `| E-`** -- e uma entrada **partida** nao comeca. Esta catraca mede a FORMA:
# entrada = uma linha, sem linhas orfas dentro da tabela, IDs unicos, sem newline LITERAL e
# sem ID citado sem entrada fora da faixa historica declarada (E-571..E-777).
'--- 9. FORMA do indice de evidencias - entrada = uma linha, sem ID fantasma'
$saidaForma = & python -X utf8 scripts/hades_index_shape_check_test.py 2>&1
$saidaForma | Select-Object -Last 2 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_index_shape_check_test.py (exit $LASTEXITCODE)") }
$saidaForma2 = & python -X utf8 scripts/hades_index_shape_check.py --check 2>&1
$saidaForma2 | Select-Object -First 6 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_index_shape_check.py --check (exit $LASTEXITCODE)") } else { '    OK' }

# *POR QUE ISTO EXISTE:* o `E-1008`. O `mission_check` exige «alguma evidencia >= E-960», e um
# documento de retomada parado no E-1000 satisfaz PARA SEMPRE. Esta catraca mede a JANELA: o
# maior ID citado tem de estar a menos de 12 evidencias da cabeca do indice.
'--- 10. JANELA de citacao dos documentos de retomada'
$saidaJanela = & python -X utf8 scripts/hades_handoff_window_check_test.py 2>&1
$saidaJanela | Select-Object -Last 2 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_handoff_window_check_test.py (exit $LASTEXITCODE)") }
$saidaJanela2 = & python -X utf8 scripts/hades_handoff_window_check.py --check 2>&1
$saidaJanela2 | Select-Object -First 6 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_handoff_window_check.py --check (exit $LASTEXITCODE)") } else { '    OK' }


''
if ($falhas.Count -gt 0) {
  "GATES LOCAIS: $($falhas.Count) falha(s)"
  $falhas | ForEach-Object { "  - $_" }
  exit 1
}
$passos = @('gofmt', 'build')
if (-not $SemLint) { $passos += 'lint' }
if (-not $SemTestes) { $passos += 'testes' }
# E-561: o escopo de `./app/...` entra no resumo SEMPRE - medido ou PULADO. Um resumo que omitisse o
# passo pulado faria um gate de escopo menor parecer um gate de escopo maior.
if ($appRodou) { $passos += 'app' } else { $passos += 'app-PULADO' }
# E-989: o manifesto entra no resumo. *Um passo que corre e nao aparece no resumo e' um passo que
# ninguem sabe que correu.*
$passos += 'manifesto'
$passos += 'marca'
$passos += 'ascii'
"GATES LOCAIS: $($passos -join ' + ') VERDES"
