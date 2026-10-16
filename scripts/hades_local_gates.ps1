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
param([switch]$SemLint, [switch]$SemTestes, [switch]$ComE2E)

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
  #      (`llama3.2:3b`) e a sonda de disponibilidade dizia "disponivel" -> 3 min por chamada;
  #   3. com o travamento corrigido, o gate passou uma vez (**549,506 s**) e falhou noutra (**1507,556 s** = o limite
  #      de 25 min), com quadro de `net/http` preso -> *a duracao nao e' estavel porque depende de INFERENCIA REAL*;
  #   4. a causa: os testes opt-in de modelo local fazem trabalho real de LLM (benchmark 2 tentativas, repeticao
  #      **3x2 geracoes**, fixer 2 reparos, cada tentativa com orcamento de 4 min) -> **6-10 geracoes** = minutos.
  #      *Isso e' MEDICAO, nao verificacao.* No CI (sem Ollama) eles pulam na hora - e' por isso que o CI e' verde.
  #   5. -> o gate EXCLUI os testes de medicao real e **DECLARA** que os excluiu. Eles continuam executaveis sob
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

# *POR QUE ISTO EXISTE:* o `E-1008`. O `mission_check` exige "alguma evidencia >= E-960", e um
# documento de retomada parado no E-1000 satisfaz PARA SEMPRE. Esta catraca mede a JANELA: o
# maior ID citado tem de estar a menos de 12 evidencias da cabeca do indice.
'--- 10. JANELA de citacao dos documentos de retomada'
$saidaJanela = & python -X utf8 scripts/hades_handoff_window_check_test.py 2>&1
$saidaJanela | Select-Object -Last 2 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_handoff_window_check_test.py (exit $LASTEXITCODE)") }
$saidaJanela2 = & python -X utf8 scripts/hades_handoff_window_check.py --check 2>&1
$saidaJanela2 | Select-Object -First 6 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_handoff_window_check.py --check (exit $LASTEXITCODE)") } else { '    OK' }

# *POR QUE ISTO EXISTE:* o `E-1010`. **18 instrumentos em `scripts/` nao corriam em nenhum gate.**
# Entre eles, **dois testes** cujas proprias docstrings dizem porque existem -- e o
# `hades_scope_classifier_autoteste.py` **descobriu** uma divergencia entre a afirmacao do `E-352`
# e o codigo, **reportou-a**, e saiu **0 desde entao**. => *Um instrumento que mede e nao atua e'
# decorativo.* => *Estes passam a correr aqui.*
'--- 11. instrumentos de auditoria que estavam ORFAOS'
$orfaos = @(
  @('scripts/hades_orphan_instruments_check.py', '--check', 'a catraca dos orfaos'),
  @('scripts/hades_scope_classifier_autoteste.py', '', 'o autoteste do classificador (E-352)'),
  @('scripts/hades_mission_check_invariantes_test.py', '', 'invariantes do mission_check (E-444/E-446)'),
  @('scripts/hades_timeout_class_check_test.py', '', 'a catraca da classe de orcamento (E-362)'),
  @('scripts/hades_checkers_autoteste.py', '', 'o teste do teste (ADV-54)'),
  @('scripts/hades_fix_mojibake.py', '--check', 'a dupla codificacao'),
  @('scripts/audit_tenant_guards.py', '', 'a guarda de tenant nos handlers')
)
foreach ($o in $orfaos) {
  $saida = & python -X utf8 $o[0] $o[1] 2>&1
  $saida | Select-Object -Last 1 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(96, $_.ToString().Length)) }
  if ($LASTEXITCODE -ne 0) { $falhas.Add("$($o[0]) $($o[1]) (exit $LASTEXITCODE) - $($o[2])") }
}
'    OK'

# *POR QUE ISTO EXISTE:* o `E-1010`. O `hades_go_test_check.py` (E-488) foi criado porque "a
# bateria de 15 checkers nao corria `go test`" -- e ele nao estava em gate NENHUM. *Um checker
# que nunca corre nao mede nada.* Medido com a receita completa: 4 pacotes, exit=0, ~450s.
# => *Ele exige o `LOCALAPPDATA` sob `_tmp/` e recusa (exit=2) se o valor HERDADO vier errado --
#    e' um anti-armadilha contra teste que passa por acidente, e o valor fica aqui, nao no script.*
# *POR QUE ISTO EXISTE:* o `E-1012`. O `dist\HadesSetup.exe` existia desde 10-16 05:11 com
# `produto='Hades'`, e **nenhuma** leitura do repositorio o mostrava -- porque o `dist\` esta' no
# `.gitignore`. => *Antes de declarar um artefacto ausente, mede-se o DISCO, e nao so' o Git.*
# => *O script confere os 3 alvos do `.iss` ANTES de o chamar, porque o `.iss` e' DEFENSIVO
#    (`#if FileExists(...)`) e um alvo em falta sairia como instalador INCOMPLETO, com exit 0.*
'--- 13. INSTALADOR Windows (Inno Setup), NAO assinado'
$saidaInst = & powershell -NoProfile -ExecutionPolicy Bypass -File scripts/hades_installer_windows.ps1 2>&1
$saidaInst | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(104, $_.ToString().Length)) }
if ($LASTEXITCODE -eq 3) {
  '    BLOCKED_BY_EXTERNAL_DEPENDENCY: sem ISCC.exe -- declarado, nao e falha do produto'
} elseif ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_installer_windows.ps1 (exit $LASTEXITCODE)")
} else { '    OK' }

'--- 12. TESTES GO (o 16o checker, E-488)'
$env:LOCALAPPDATA = (Join-Path $PWD '_tmp\localappdata')
New-Item -ItemType Directory -Force -Path $env:LOCALAPPDATA | Out-Null
$saidaGo = & python -X utf8 scripts/hades_go_test_check.py 2>&1
$saidaGo | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -eq 3) {
  '    BLOCKED_BY_EXTERNAL_DEPENDENCY: sem `go` -- declarado, nao e falha do produto'
} elseif ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_go_test_check.py (exit $LASTEXITCODE)")
} else { '    OK' }



# *POR QUE ISTO EXISTE:* o `E-1022`. O pacote da extensao (`dist\browser-bridge\*.zip`) e' um
# artefacto local, invisivel ao Git. *Um instrumento que nunca corre nao mede nada* -- a catraca
# `hades_orphan_instruments_check.py` apontou-o como ORFAO precisamente por isso.
# => *O script CONSTROI e VERIFICA: se o pacote nao existir, ele e' refeito; se existir, e' medido.*
# => *E ele reprova um defeito real medido nesta sessao: o `.NET` escreve as entradas do ZIP com
#    barra invertida (8 de 9), a spec exige barra normal, e o Chrome recusa carregar o manifest.
'--- 14. PACOTE da extensao HADES Browser Bridge (zip nao assinado)'
$saidaPkg = & python -X utf8 scripts/hades_browser_bridge_package.py 2>&1
$saidaPkg | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -eq 3) {
  '    BLOCKED_BY_EXTERNAL_DEPENDENCY: sem `python` -- declarado, nao e falha do produto'
} elseif ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_browser_bridge_package.py (exit $LASTEXITCODE)")
} else { '    OK' }

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
# E-1022: o pacote da extensao entra no resumo. *Um passo que corre e nao aparece no resumo e'
# um passo que ninguem sabe que correu* (o E-989, aplicado outra vez).
$passos += 'pacote-extensao'

# E-1031: o ID da extensao e o manifesto do native host. *O `allowed_origins` era um placeholder
# (`REPLACE_WITH_EXTENSION_ID`) porque o ID so' existia depois de carregar a extensao no navegador.*
# Com a chave publica versionada, o ID e' CALCULAVEL e a instalacao do host deixa de ser adivinhacao.
'--- 15. ID deterministico da extensao e manifesto do native host'
$saidaNh = & python -X utf8 scripts/hades_native_host.py --check-repo 2>&1
$saidaNh | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -eq 3) {
  '    BLOCKED_BY_EXTERNAL_DEPENDENCY: sem `python` -- declarado, nao e falha do produto'
} elseif ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_native_host.py --check-repo (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }

''
'--- 16. testes do ID deterministico (com fault injection)'
$saidaNhT = & python -X utf8 scripts/hades_native_host_test.py 2>&1
$saidaNhT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -eq 3) {
  '    BLOCKED_BY_EXTERNAL_DEPENDENCY: sem `python`'
} elseif ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_native_host_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }

''
# E-1032: o E2E REAL num navegador. *Um protocolo que nunca atravessou uma extensao a
# serio nao esta' provado.* Mas abre uma janela do Edge -- logo entra por opt-in: um gate
# que abre um navegador sem avisar nao serve para correr a toda a hora.
'--- 17. E2E REAL: Edge -> extensao -> native messaging -> Companion'
if ($ComE2E) {
  $saidaE2E = & node scripts/hades_browser_bridge_e2e.mjs 2>&1
  $saidaE2E | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
  if ($LASTEXITCODE -ne 0) {
    $falhas.Add("hades_browser_bridge_e2e.mjs (exit $LASTEXITCODE)")
    '    FALHA'
  } else { '    OK' }
} else {
  '    PULADO (opt-in: use -ComE2E para abrir o Edge e falar com o Companion)'
}

''
# E-1033: o controle de ABA. *O canal provado nao prova que o HADES consegue observar e
# interagir com uma pagina -- nem que ele RECUSA o que nao deve.* Servidor local, campo de
# senha, badge e revogacao. Opt-in pelo mesmo motivo do passo 17.
'--- 18. E2E DE ABA: concessao, DOM, campo sensivel, badge, revogacao'
if ($ComE2E) {
  $saidaAba = & node scripts/hades_browser_bridge_e2e_aba.mjs 2>&1
  $saidaAba | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
  if ($LASTEXITCODE -ne 0) {
    $falhas.Add("hades_browser_bridge_e2e_aba.mjs (exit $LASTEXITCODE)")
    '    FALHA'
  } else { '    OK' }
} else {
  '    PULADO (opt-in: use -ComE2E)'
}

''
# E-1034: o takeover humano. *O scope `browser:takeover` estava declarado e nao tinha uma
# linha de implementacao -- e ninguem via quem tinha o controlo.*
'--- 19. E2E DE TAKEOVER: o humano assume, o agente e recusado, a sessao volta'
if ($ComE2E) {
  $saidaTk = & node scripts/hades_browser_bridge_e2e_takeover.mjs 2>&1
  $saidaTk | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
  if ($LASTEXITCODE -ne 0) {
    $falhas.Add("hades_browser_bridge_e2e_takeover.mjs (exit $LASTEXITCODE)")
    '    FALHA'
  } else { '    OK' }
} else {
  '    PULADO (opt-in: use -ComE2E)'
}

''
# E-1035: a revogacao por MUDANCA DE ORIGEM. *Codigo de seguranca sem teste e' uma promessa.*
'--- 20. E2E DE ORIGEM: navegar para outra origem revoga, a outra aba continua'
if ($ComE2E) {
  $saidaOg = & node scripts/hades_browser_bridge_e2e_origem.mjs 2>&1
  $saidaOg | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
  if ($LASTEXITCODE -ne 0) {
    $falhas.Add("hades_browser_bridge_e2e_origem.mjs (exit $LASTEXITCODE)")
    '    FALHA'
  } else { '    OK' }
} else {
  '    PULADO (opt-in: use -ComE2E)'
}

''
# E-1036: a catraca sobre o CODIGO. *O defeito do opt-in apareceu em TRES locais sucessivos;
# a cura pontual nao fecha a classe.* Esta mede o codigo, nao o resultado: falha se um
# chamador novo de `validarUrl`/`guardarRota` nao passar `permitirPrivado`.
'--- 21. CATRACA do opt-in: todo o chamador de validarUrl/guardarRota passa o opt-in'
$saidaRat = & python -X utf8 scripts/hades_bridge_optin_ratchet.py 2>&1
$saidaRat | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -eq 3) {
  '    BLOCKED_BY_EXTERNAL_DEPENDENCY: sem `python`'
} elseif ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_bridge_optin_ratchet.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }

''
'--- 22. testes da catraca (fault injection, numa COPIA)'
$saidaRatT = & python -X utf8 scripts/hades_bridge_optin_ratchet_test.py 2>&1
$saidaRatT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -eq 3) {
  '    BLOCKED_BY_EXTERNAL_DEPENDENCY: sem `python`'
} elseif ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_bridge_optin_ratchet_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }

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
# E-1022: o pacote da extensao entra no resumo. *Um passo que corre e nao aparece no resumo e'
# um passo que ninguem sabe que correu* (o E-989, aplicado outra vez).
$passos += 'pacote-extensao'
# E-1031: *o ID e o native host.* Um passo que corre e nao aparece no resumo e' um passo
# que ninguem sabe que correu -- a mesma licao do E-989, aplicada pela terceira vez.
$passos += 'native-host-id'
$passos += 'native-host-testes'
$passos += 'e2e-aba'
$passos += 'e2e-takeover'
$passos += 'e2e-origem'
$passos += 'catraca-optin'

''
# E-1037: o `pushState`, o RELOAD e o badge. *A divida declarada no E-1035 era um nao-problema
# -- `pushState` nao pode mudar de origem -- mas ao medi-la encontrei DOIS defeitos reais.*
'--- 23. E2E DO PUSHSTATE: a concessao sobrevive ao reload; a origem real revoga'
if ($ComE2E) {
  $saidaPs = & node scripts/hades_browser_bridge_e2e_pushstate.mjs 2>&1
  $saidaPs | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
  if ($LASTEXITCODE -ne 0) {
    $falhas.Add("hades_browser_bridge_e2e_pushstate.mjs (exit $LASTEXITCODE)")
    '    FALHA'
  } else { '    OK' }
} else {
  '    PULADO (opt-in: use -ComE2E)'
}
$passos += 'e2e-pushstate'

''
# E-1038: o BFCACHE. *A ultima divida da familia `onUpdated`: voltar atras reviveria a
# concessao sobre conteudo de outra origem?* Medido: nao -- e o regresso traz URL.
'--- 24. E2E DO BFCACHE: voltar atras NAO revive a concessao'
if ($ComE2E) {
  $saidaBf = & node scripts/hades_browser_bridge_e2e_bfcache.mjs 2>&1
  $saidaBf | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
  if ($LASTEXITCODE -ne 0) {
    $falhas.Add("hades_browser_bridge_e2e_bfcache.mjs (exit $LASTEXITCODE)")
    '    FALHA'
  } else { '    OK' }
} else {
  '    PULADO (opt-in: use -ComE2E)'
}
$passos += 'e2e-bfcache'

''
# E-1039: frames e shadow DOM. *O `observar` devolvia uma lista PARCIAL como se fosse
# completa: o `total` era o total do documento principal e parecia o total da pagina.*
'--- 25. E2E DE FRAMES: o observar ATRAVESSA shadow aberto e iframe, e DECLARA os limites'
if ($ComE2E) {
  $saidaFr = & node scripts/hades_browser_bridge_e2e_frames.mjs 2>&1
  $saidaFr | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
  if ($LASTEXITCODE -ne 0) {
    $falhas.Add("hades_browser_bridge_e2e_frames.mjs (exit $LASTEXITCODE)")
    '    FALHA'
  } else { '    OK' }
} else {
  '    PULADO (opt-in: use -ComE2E)'
}
$passos += 'e2e-frames'

''
# E-1040: o scope `browser:files`. *Estava nos 3 canonicos e nao tinha uma linha de
# implementacao -- e a mensagem do `preencher` prometia um caminho autorizado que nao existia.*
'--- 26. E2E DE ARQUIVOS: o browser:files DECIDE -- executavel nunca fica permitido'
if ($ComE2E) {
  $saidaAr = & node scripts/hades_browser_bridge_e2e_arquivos.mjs 2>&1
  $saidaAr | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
  if ($LASTEXITCODE -ne 0) {
    $falhas.Add("hades_browser_bridge_e2e_arquivos.mjs (exit $LASTEXITCODE)")
    '    FALHA'
  } else { '    OK' }
} else {
  '    PULADO (opt-in: use -ComE2E)'
}

''
# E-1040: os 17 testes da politica de ficheiros. *Nao precisam de browser: sao politica pura.*
'--- 27. TESTES da politica de ficheiros (17 assercoes, sem browser)'
$saidaArT = & node --test apps/hades-browser-bridge/test/arquivos.test.js 2>&1
$linhaAr = ($saidaArT | Select-String -Pattern 'pass ' | Select-Object -Last 1)
$linhaAr | ForEach-Object { "    " + $_.ToString().Trim() }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("test/arquivos.test.js (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'e2e-arquivos'
"GATES LOCAIS: $($passos -join ' + ') VERDES"
