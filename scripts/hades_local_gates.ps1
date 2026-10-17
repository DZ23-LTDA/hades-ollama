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
  # E-670 -- OS TESTES ENTRAM NO GATE LOCAL.
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

# *POR QUE ISTO EXISTE:* o `E-1070` e o `E-1071`. O `E-1068` mudou o `apps/mobile-agentic/app.json`
# -- um ficheiro que **alimenta um instrumento** -- e **nao correu o instrumento**. O
# `HADES_BRAND_INVENTORY.md` ficou desactualizado e o CI reprovou com `INVENTARIO DIVERGENTE`.
# *A causa-raiz nao era o `app.json`: era **nao haver nada que obrigasse**.*
# => este passo corre os **tres** geradores em `--check` e diz **qual regenerar**.
'--- 7b. derivados batem com a fonte (E-1071) - o pre-commit tambem corre isto'
$saidaDeriv = & python -X utf8 scripts/hades_derivados_check.py --quiet 2>&1
$saidaDeriv | Select-Object -Last 8 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_derivados_check.py (exit $LASTEXITCODE)") } else { '    OK   derivados consistentes (3 geradores)' }

# *POR QUE ISTO EXISTE:* o `E-1055` e o `E-1056` ja' o disseram -- **copiar um scanner para
# `scripts/` sem um gate e' um INSTRUMENTO ORFAO**. E este teste tem o seu proprio valor: *um
# teste que nao corre e' um teste que nao existe.*
'--- 7c. o teste do pre-commit dos derivados (E-1071)'
$saidaDerivT = & python -X utf8 scripts/hades_derivados_check_test.py 2>&1
$saidaDerivT | Select-Object -Last 4 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_derivados_check_test.py (exit $LASTEXITCODE)") } else { '    OK   13/13 verificacoes do pre-commit' }

# *POR QUE ISTO EXISTE:* o `E-1076` / FASE-7 item 8. O pacote de auditoria independente
# (`README_AUDITOR.md`, `PARIDADE_EVIDENCIAS.csv`, `GATES.md`, `LIMITACOES.md`,
# `ARTEFATOS.sha256`) e' **gerado** a partir do indice, do estado, dos gates declarados e dos
# artefactos que existem. *Um pacote escrito a mao envelhece no dia seguinte* -- e' o `E-1071`.
'--- 7d. PACOTE de auditoria independente (FASE-7 item 8) - bate com o repositorio?'
$saidaAudit = & python -X utf8 scripts/hades_audit_package.py --check 2>&1
$saidaAudit | Select-Object -Last 3 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_audit_package.py --check (exit $LASTEXITCODE)") } else { '    OK   pacote de auditoria consistente (5 artefactos)' }

# *24 verificacoes, 4 defeitos injetados (artefacto apagado, linha a mais, indice que muda,
# e o `E-83` no proprio gerador: o `tipo` manda e nao o texto).*
'--- 7e. teste do pacote de auditoria (4 defeitos + contraprova)'
$saidaAuditT = & python -X utf8 scripts/hades_audit_package_test.py 2>&1
$saidaAuditT | Select-Object -Last 4 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_audit_package_test.py (exit $LASTEXITCODE)") } else { '    OK   24/24 verificacoes do pacote de auditoria' }

# *POR QUE ISTO EXISTE:* o `E-1077` / FASE-7 item 8, ultima parte: *"a lista de findings
# abertos/fechados"*. O `FINDINGS.json` foi **capturado da API do GitHub** (com data e SHA) e o
# `FINDINGS.md` e' gerado **dele** -- logo o `--check` **nao precisa de rede**.
'--- 7f. FINDINGS do PR #438 - o MD bate com o JSON capturado?'
$saidaFind = & python -X utf8 scripts/hades_findings_report.py --check 2>&1
$saidaFind | Select-Object -Last 3 | ForEach-Object { "    $_" }
if ($LASTEXITCODE -ne 0) { $falhas.Add("hades_findings_report.py --check (exit $LASTEXITCODE)") } else { '    OK   findings do PR consistentes (MD bate com o JSON)' }

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
$passos += 'derivados'
$passos += 'auditoria'
$passos += 'findings'
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

''
# E-1044: a CATRACA DO ENCAIXE. *Um passo de desbloqueio escrito nao e' um passo de
# desbloqueio vigiado.* Falha se aparecer uma rota HTTP que chame `FindApproved` ou um
# handler de approvals sem sessao -- o sinal de que o Companion ja' pode verificar.
'--- 28. CATRACA do encaixe: acorda se o runtime expuser a rota de aprovacao'
$saidaEnc = & python -X utf8 scripts/hades_approval_hook_ratchet.py 2>&1
$saidaEnc | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -eq 3) {
  '    BLOCKED_BY_EXTERNAL_DEPENDENCY: sem `python`'
} elseif ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_approval_hook_ratchet.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }

''
'--- 29. testes da catraca do encaixe (4 injecoes + contraprova, numa COPIA)'
$saidaEncT = & python -X utf8 scripts/hades_approval_hook_ratchet_test.py 2>&1
$saidaEncT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -eq 3) {
  '    BLOCKED_BY_EXTERNAL_DEPENDENCY: sem `python`'
} elseif ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_approval_hook_ratchet_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'catraca-encaixe'

''
# E-1045: o RENDERIZADOR do indice. *Nao se adivinha um renderizador: usa-se um.*
# Renderiza as entradas com o `marked` e exige 5 campos em todas -- as contagens de `|` deram
# 767, 303 e 768, e todas estavam erradas.
'--- 30. RENDER do indice: todas as entradas com 5 campos'
$saidaRen = & node scripts/hades_index_render_check.mjs 2>&1
$saidaRen | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_index_render_check.mjs (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'render-indice'

''
# E-1048: o INVENTARIO do worktree. *Uma contagem nao e' um inventario.* Separa `M` de conteudo
# de `M` de fim-de-linha -- reportei "14 entradas" durante dezenas de rondas e havia ZERO de
# trabalho real.
'--- 31. INVENTARIO do worktree: `M` de conteudo vs `M` de fim-de-linha'
$saidaInv = & python -X utf8 scripts/hades_worktree_inventory.py --check 2>&1
$saidaInv | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_worktree_inventory.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'inventario-worktree'

''
# E-1054: a LIGACAO docs -> artefactos. *Um resultado sem ponteiro existe e nao se encontra.*
# No E-1053 a ligacao do estagio 21 foi feita com TEXTO -- e o texto nao vigia nada.
'--- 32. CATRACA da ligacao: os docs do estagio citam os artefactos?'
$saidaLig = & python -X utf8 scripts/hades_stage_artifacts_check.py --check 2>&1
$saidaLig | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_stage_artifacts_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }

''
'--- 33. testes da ligacao (4 injecoes + contraprova, numa COPIA)'
$saidaLigT = & python -X utf8 scripts/hades_stage_artifacts_check_test.py 2>&1
$saidaLigT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_stage_artifacts_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'catraca-ligacao'

''
# E-1090: a DOCUMENTACAO do Companion. *Um documento que existe mas nao cobre os estados e' pior do
# que nenhum: parece documentacao e nao o e'.* => *e o mesmo defeito que o E-1084 apanhou no
# inventario -- um detector lexical ve termos, nao relacoes -- por isso cada verificacao e' especifica:
# o estado tem de estar na tabela E no mapa, a chave tem de ser a MESMA do codigo, e as limitacoes
# tem de estar declaradas em vez de escondidas.*
'--- 34. DOCUMENTACAO do Companion: os 8 estados, o mapa para os 14, e as limitacoes'
$saidaDocC = & python -X utf8 scripts/hades_companion_doc_check.py 2>&1
$saidaDocC | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_companion_doc_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'doc-companion'

''
# E-1090: o fault injection da catraca da documentacao. *Um checker que so' passa nao vale nada -- o
# E-1082 ensinou que um teste que passa com o defeito nao discrimina.* => *aqui injectam-se 4 defeitos
# no DOCUMENTO real, numa copia descartavel, e exige-se que o checker os apanhe.*
'--- 35. fault injection da catraca da documentacao (4 injecoes + contraprova)'
$saidaDocCT = & python -X utf8 scripts/hades_companion_doc_check_test.py 2>&1
$saidaDocCT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_companion_doc_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'doc-companion-testes'

''
# E-1091: a CADEIA DE RASTREABILIDADE (FASE-7 item 4). *Para cada requisito activo:
# `REQ-ID -> arquivo:linha -> teste -> comando -> resultado -> plataforma -> commit -> artefacto ->
# decisao`.* => *o gerador verifica CADA campo no codigo; um requisito cuja prova nao existe fica
# `AUSENTE` e o `--check` FALHA.*
'--- 36. CADEIA de requisitos: REQ-ID -> arquivo:linha -> teste -> comando -> resultado'
$saidaReq = & python -X utf8 scripts/hades_requisitos.py --check 2>&1
$saidaReq | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_requisitos.py --check (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'requisitos'

''
'--- 37. fault injection da cadeia de requisitos (2 injecoes + contraprova)'
$saidaReqT = & python -X utf8 scripts/hades_requisitos_test.py 2>&1
$saidaReqT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_requisitos_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'requisitos-testes'

''
# E-1093: o AMBIENTE da medicao T2 (ESTAGIO 21). *O prompt exige "hardware, versao, modelo, contexto,
# warm/cold, repeticoes e erro" -- e as duas matrizes do T2 tinham os cenarios, os modelos e as repeticoes,
# e NAO tinham o ambiente.* => *sem ambiente, os numeros nao sao reproduziveis.*
'--- 38. AMBIENTE do T2: hardware, SO, Ollama, modelos, lacunas e proveniencia'
$saidaT2 = & python -X utf8 scripts/hades_t2_contexto.py 2>&1
$saidaT2 | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
$saidaT2c = & python -X utf8 scripts/hades_t2_contexto.py --check 2>&1
$saidaT2c | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_t2_contexto.py --check (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 't2-ambiente'

''
'--- 39. fault injection do ambiente T2 (3 injecoes + contraprova)'
$saidaT2T = & python -X utf8 scripts/hades_t2_contexto_test.py 2>&1
$saidaT2T | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_t2_contexto_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 't2-ambiente-testes'

''
# E-1094: a CATRACA dos testes NAO-HERMETICOS. *A mesma classe apareceu 3 vezes (E-1087 no companion,
# E-1092 no Studio, E-1094 no runtime e no workspace): `NewRuntime` sem `DataRoot` => escreve no
# `%APPDATA%` do utilizador => falha num processo confinado.* => *a catraca declara a divida (107) e
# FALHA se o numero subir.*
'--- 40. TESTES HERMETICOS: NewRuntime em testes sempre com DataRoot (divida declarada: 107)'
$saidaH = & python -X utf8 scripts/hades_testes_hermeticos_check.py 2>&1
$saidaH | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_testes_hermeticos_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'testes-hermeticos'

''
'--- 41. fault injection da catraca dos testes hermeticos (injecao + 3 contraprovas)'
$saidaHT = & python -X utf8 scripts/hades_testes_hermeticos_check_test.py 2>&1
$saidaHT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_testes_hermeticos_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'testes-hermeticos-testes'

''
# E-1095: o ESTAGIO 19 (UX/onboarding). *`/goal` no MESMO campo do chat, onboarding com as 3 pecas,
# acessibilidade e teclado, menu unificado (<= 24 rotas) e a marca com as excecoes declaradas.*
'--- 42. UX/ONBOARDING: /goal no composer, onboarding real, ARIA e teclado, menu unificado'
$saidaUX = & python -X utf8 scripts/hades_ux_onboarding_check.py 2>&1
$saidaUX | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_ux_onboarding_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'ux-onboarding'

''
'--- 43. fault injection do UX/onboarding (3 injecoes + contraprova)'
$saidaUXT = & python -X utf8 scripts/hades_ux_onboarding_check_test.py 2>&1
$saidaUXT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_ux_onboarding_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'ux-onboarding-testes'

''
# E-1096: o ESTAGIO 22 item 1 -- *"transformar o gate em prova do mesmo snapshot candidato"*.
# => *o `measured_commit` do estado tem de ser o HEAD **antes** de os derivados serem gerados; senao o
# `gate_snapshot` declara DIVERGENTE e o gate cita um commit de ontem (o defeito do E-987).*
# => *e corre como **GERADOR**, nao como check: exigir `MESMO` no `--check` faria o CI falhar sempre,
# porque em `pull_request` o `HEAD` e' o **merge commit sintetico** (o E-993, ja' escrito no ficheiro).*
'--- 44. SNAPSHOT: o `measured_commit` do estado e' o HEAD antes de gerar os derivados'
$saidaSync = & python -X utf8 scripts/hades_sync_state_metadata.py 2>&1
$saidaSync | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_sync_state_metadata.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'snapshot'

''
'--- 45. o gate DECLARA o snapshot (MESMO ou a distancia), sem esconder a diferenca'
$saidaSnap = & python -X utf8 scripts/hades_release_gate.py 2>&1
$linhaSnap = Select-String -Path 'docs/mission/HADES_RELEASE_REPORT.md' -Pattern 'gate_snapshot' | Select-Object -First 1
if ($linhaSnap) {
  $txt = $linhaSnap.Line.Trim()
  "    " + $txt.Substring(0, [Math]::Min(110, $txt.Length))
  $passos += 'snapshot-declarado'
} else {
  $falhas.Add("o gate nao declara o gate_snapshot")
  '    FALHA'
}

''
# E-1098: o ESTAGIO ESTAGIO 6 (Harness Runtime). *As 7 capacidades -- cancelamento, timeout, retry limitado,
# idempotencia, retomada, artefactos e governacao -- nomeadas por TESTE; o cancelamento entre instancias; o
# Ollama no caminho local; e o CLI externo com bloqueio declarado (t.Skip com MOTIVO) e sonda OPT-IN.*
'--- 46. HARNESS: as 7 capacidades, cancelamento entre instancias, Ollama, e o CLI com sonda opt-in'
$saidaH = & python -X utf8 scripts/hades_harness_check.py 2>&1
$saidaH | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_harness_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'harness'

''
'--- 47. fault injection do harness (3 injecoes + contraprova)'
$saidaHT = & python -X utf8 scripts/hades_harness_check_test.py 2>&1
$saidaHT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_harness_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'harness-testes'


''
# E-1099: o ESTAGIO ESTAGIO 9 (Agentes e subagentes). *O swarm com o `MaxAgents` APLICADO (semaforo), o orcamento
# de 4 limites, o custo por passo ligado ao motor, o isolamento de worktree, os artefactos reais, e o HITL
# com recusa por omissao.*
'--- 48. AGENTES: swarm com limite aplicado, orcamento, isolamento, artefactos e HITL'
$saidaAg = & python -X utf8 scripts/hades_agents_check.py 2>&1
$saidaAg | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_agents_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'agentes'

''
'--- 49. fault injection dos agentes (3 injecoes + contraprova)'
$saidaAgT = & python -X utf8 scripts/hades_agents_check_test.py 2>&1
$saidaAgT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_agents_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'agentes-testes'


''
# E-1100: o ESTAGIO ESTAGIO 1 (Auditoria total e inventario real). *A matriz de plataformas (6 plataformas x 7
# eixos), os 15 workflows citados, e ZERO instrucoes destrutivas sem o aviso de HISTORICO e PROIBIDO com WIP.*
'--- 50. AUDITORIA: matriz de plataformas, workflows citados, e o aviso das instrucoes destrutivas'
$saidaM = & python -X utf8 scripts/hades_platform_matrix.py 2>&1
$saidaM | Select-Object -Last 1 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
$saidaInv = & python -X utf8 scripts/hades_audit_inventory_check.py 2>&1
$saidaInv | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_audit_inventory_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'auditoria-inventario'

''
'--- 51. fault injection da auditoria (3 injecoes + contraprova)'
$saidaInvT = & python -X utf8 scripts/hades_audit_inventory_check_test.py 2>&1
$saidaInvT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_audit_inventory_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'auditoria-inventario-testes'


''
# E-1102: o ESTAGIO ESTAGIO 3 (Arquitetura canonica). *As fontes de verdade SINGULARES (1 Runtime, 1 Mission, 0
# MissionControl -- o MC e' superficie, nao motor), o lock partilhado da aprovacao, os testes de CONTRATO
# (nao busca de simbolo), e a arquitectura e a API documentadas.*
'--- 52. ARQUITETURA: fontes de verdade singulares, testes de contrato, e a documentacao'
$saidaArq = & python -X utf8 scripts/hades_architecture.py 2>&1
$saidaArq | Select-Object -Last 1 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
$saidaArqC = & python -X utf8 scripts/hades_architecture_check.py 2>&1
$saidaArqC | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_architecture_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'arquitetura'

''
'--- 53. fault injection da arquitectura (3 injecoes + contraprova)'
$saidaArqT = & python -X utf8 scripts/hades_architecture_check_test.py 2>&1
$saidaArqT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_architecture_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'arquitetura-testes'


''
# E-1104: os 3 instrumentos de SEGURANCA que corriam so' na bateria manual -- agora DECLARADOS no gates.
# *O ESTAGIO ESTAGIO 18 exige "RBAC/ABAC, tenant isolation, non-inference, approvals de uso unico e audit log"*
# => *e um instrumento que corre numa bateria nao-commitada nao esta' ligado a nada.*
'--- 54. SEGURANCA: tenant scope, non-inference, e o relogio de teste'
$saidaTen = & python -X utf8 scripts/hades_tenant_scope_check.py --check 2>&1
$saidaTen | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_tenant_scope_check.py --check (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'tenant-scope'

$saidaNon = & python -X utf8 scripts/hades_non_inference_scan.py 2>&1
$saidaNon | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_non_inference_scan.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'non-inference'

$saidaClock = & python -X utf8 scripts/hades_test_clock_check.py 2>&1
$saidaClock | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_test_clock_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'relogio-de-teste'

''
'--- 55. o CHECADOR da seguranca (9 eixos do Dominio K)'
$saidaSec = & python -X utf8 scripts/hades_security_check.py 2>&1
$saidaSec | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_security_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'seguranca'

''
'--- 56. fault injection da seguranca (3 injecoes + contraprova)'
$saidaSecT = & python -X utf8 scripts/hades_security_check_test.py 2>&1
$saidaSecT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_security_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'seguranca-testes'


''
# E-1105: o ESTAGIO ESTAGIO 14 / ESTAGIO FASE-3-B (HADES Browser Bridge). *MV3, incognito fora, permissoes minimas, CSP
# estrita, sem eval nem codigo remoto, native host restrito, e os 120 testes unitarios da extensao.*
'--- 57. EXTENSAO: o manifesto, as permissoes, a CSP e o native host'
$saidaExt = & python -X utf8 scripts/hades_extension_check.py 2>&1
$saidaExt | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_extension_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'extensao'

''
'--- 58. os 120 TESTES da extensao (node --test)'
$saidaExtT = & node --test apps/hades-browser-bridge/test/arquivos.test.js apps/hades-browser-bridge/test/canal_local.test.js apps/hades-browser-bridge/test/carregavel.test.js apps/hades-browser-bridge/test/companion.test.js apps/hades-browser-bridge/test/protocol.test.js apps/hades-browser-bridge/test/transporte.test.js 2>&1
$saidaExtT | Select-String -Pattern '^# (pass|fail)' | ForEach-Object { "    " + $_.Line.InnerText }
$saidaExtT | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("os testes da extensao (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'extensao-testes'

''
'--- 59. fault injection da extensao (3 injecoes + contraprova)'
$saidaExtIj = & python -X utf8 scripts/hades_extension_check_test.py 2>&1
$saidaExtIj | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_extension_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'extensao-injecoes'


''
# E-1106: o ESTAGIO 7 (MCP Manager). *O estado dizia `APENAS_CATALOGO`; o codigo tem cliente real --
# `exec.Command` no `mcp.go`, transporte HTTP no `mcp_remote.go`, SSRF no `unsafeEgressIP`, escopo de
# workspace, politica de comando, e o proprio HADES como servidor MCP.*
'--- 60. MCP: o catalogo, o spawn, o transporte, o SSRF e a politica'
$saidaMcp = & python -X utf8 scripts/hades_mcp_check.py 2>&1
$saidaMcp | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_mcp_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'mcp'

''
'--- 61. fault injection do MCP (3 injecoes + contraprova)'
$saidaMcpT = & python -X utf8 scripts/hades_mcp_check_test.py 2>&1
$saidaMcpT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_mcp_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'mcp-injecoes'


''
# E-1107: o ESTAGIO 8 (Plugins e Skills). *A METADE QUE NAO PRECISA DE PRIVILEGIO DE MONTAGEM: o manifesto
# tipado e versionado, o bundle com hash e limite (64 MiB), o registry, os 7 estados canonicos, o escopo de
# tenant, o status e o workspace.* => *a EXECUCAO (o sandbox mount/chroot) e' o `B-14`, um limite de HOST.*
'--- 62. PLUGINS: o manifesto, o bundle, o registry, os estados e o escopo'
$saidaPlug = & python -X utf8 scripts/hades_plugins_check.py 2>&1
$saidaPlug | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_plugins_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'plugins'

''
'--- 63. fault injection dos plugins (3 injecoes + contraprova)'
$saidaPlugT = & python -X utf8 scripts/hades_plugins_check_test.py 2>&1
$saidaPlugT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_plugins_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'plugins-injecoes'


''
# E-1108: o ESTAGIO 2 (Estabilizacao e consolidacao da base). *O flake do macOS esta' `NAO_REPRODUZIDO`
# (6 SHAs limpos depois do `E-197`), a base e' hermetica com divida declarada (107), a codificacao e' limpa,
# as catracas tem teste, e o `./server` passa com o comando EXACTO do CI (`-bench=. -benchtime=1x`).*
'--- 64. ESTABILIDADE: o flake, a hermeticidade, a codificacao e o comando do CI'
$saidaStab = & python -X utf8 scripts/hades_stability_check.py 2>&1
$saidaStab | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_stability_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'estabilidade'

''
'--- 65. fault injection da estabilidade (3 injecoes + contraprova)'
$saidaStabT = & python -X utf8 scripts/hades_stability_check_test.py 2>&1
$saidaStabT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_stability_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'estabilidade-injecoes'


''
# E-1109: a MATRIZ COMPETITIVA. *27/27 repositorios e 49/49 referencias de produto cobertos; nenhum
# `SUPERIOR_MEASURED` sem benchmark; e as correccoes factuais do prompt aplicadas.*
'--- 66. MATRIZ COMPETITIVA: cobertura obrigatoria e nenhuma vitoria sem prova'
$saidaComp = & python -X utf8 scripts/hades_competitive_check.py 2>&1
$saidaComp | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_competitive_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'matriz-competitiva'

''
'--- 67. fault injection da matriz competitiva (3 injecoes + contraprova)'
$saidaCompT = & python -X utf8 scripts/hades_competitive_check_test.py 2>&1
$saidaCompT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_competitive_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'matriz-injecoes'


''
# E-1110: a FASE 7 (rastreabilidade sem inventar historia). *A faixa `E-571..E-777` declarada, os 12 IDs
# citados marcados `EVIDENCE_LOST_HISTORICAL` com alcance e impacto, os placeholders separados, nenhum
# requisito ativo dependente de ID perdido, e o pacote de auditoria completo.*
'--- 68. RASTREABILIDADE: a evidencia perdida marcada, e nenhum requisito dependente dela'
$saidaTrace = & python -X utf8 scripts/hades_traceability_check.py 2>&1
$saidaTrace | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_traceability_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'rastreabilidade'

''
'--- 69. fault injection da rastreabilidade (3 injecoes + contraprova)'
$saidaTraceT = & python -X utf8 scripts/hades_traceability_check_test.py 2>&1
$saidaTraceT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_traceability_check_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'rastreabilidade-injecoes'


''
# E-1111: a REVALIDACAO dos 14 findings do PR #438. *O finding e' HIPOTESE: o veredicto vem do CODIGO ATUAL,
# nao do comentario do bot. 13 `CONFIRMADO_CORRIGIDO` e 1 `CONFIRMADO_PARCIAL` (o pipe no indice -- a classe
# persiste em 831 linhas, e corrigi-la exigiria tocar na evidencia, que o prompt proibe).*
'--- 70. FINDINGS do PR: a revalidacao contra o codigo atual'
$saidaFind = & python -X utf8 scripts/hades_findings_revalidation.py --check 2>&1
$saidaFind | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_findings_revalidation.py --check (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'findings-revalidados'

''
'--- 71. fault injection dos findings (3 injecoes + contraprova)'
$saidaFindT = & python -X utf8 scripts/hades_findings_revalidation_test.py 2>&1
$saidaFindT | Select-Object -Last 2 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_findings_revalidation_test.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'findings-injecoes'


''
# E-1112: o PACOTE LINUX. *Cross-compile GOOS=linux CGO=0 => binario ELF de 45,6 MB => tarball de 16,2 MB com
# VERSION, install/uninstall, PROVENIENCIA e SHA256SUMS que confere.* => *e a matriz de plataformas cita-o.*
# NAO foi instalado (nao houve Linux), NAO e' byte-a-byte o do workflow (que usa CGO=1), e NAO e' assinado.
'--- 72. PACOTE LINUX: o tarball, o binario ELF e o hash'
$saidaLinux = & python -X utf8 scripts/hades_linux_package.py --check 2>&1
$saidaLinux | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_linux_package.py --check (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'pacote-linux'


''
# E-1115: o E2E da extensao com DOM REAL. *A aba em 127.0.0.1, a observacao do DOM via chrome.scripting, o
# clique, o preenchimento de um campo nao-sensivel, a navegacao para outra origem, o NEGATIVO (o valor do
# password NAO e' lido), e a prova de que a extensao NAO injecta fora das origens autorizadas.*
'--- 73. E2E DA EXTENSAO COM DOM REAL (nove verificacoes)'
Push-Location app/ui/app
$saidaDom = & node e2e-extension-dom.mjs 2>&1
$saidaDom | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
$exitDom = $LASTEXITCODE
Pop-Location
if ($exitDom -ne 0) {
  $falhas.Add("e2e-extension-dom.mjs (exit $exitDom)")
  '    FALHA'
} else { '    OK' }
$passos += 'extensao-dom'


''
# E-1116: o PACOTE FIREFOX. *O manifest derivado (gecko.id + data_collection_permissions + background.scripts),
# CSP estrita preservada, e o `web-ext lint` do PROPRIO Mozilla em 0 erros / 0 avisos / 0 notices.*
'--- 74. PACOTE FIREFOX: o manifest derivado e o lint do Mozilla'
$saidaFf = & python -X utf8 scripts/hades_firefox_package.py --check 2>&1
$saidaFf | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_firefox_package.py --check (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'pacote-firefox'


''
# E-1117: o E2E do TAKEOVER e da REVOGACAO ao vivo. *A concessao criada e revogada pelo POPUP (a porta real),
# a accao que devolve PEDIDO e nao aprovacao, o scope nao-canonico recusado, o gesto humano que marca o takeover,
# a escrita do agente RECUSADA durante ele, e a devolucao que a faz voltar a funcionar.*
'--- 75. E2E DO TAKEOVER HUMANO E DA REVOGACAO AO VIVO (24 verificacoes)'
Push-Location app/ui/app
$saidaTk = & node e2e-extension-takeover.mjs 2>&1
$saidaTk | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
$exitTk = $LASTEXITCODE
Pop-Location
if ($exitTk -ne 0) {
  $falhas.Add("e2e-extension-takeover.mjs (exit $exitTk)")
  '    FALHA'
} else { '    OK' }
$passos += 'extensao-takeover'


''
# E-1118: o E2E do PAREAMENTO REAL com o Companion (processo real, pipe do SO). *O `host:ola` honesto, o
# codigo de 6 digitos com expiracao, a chave de 32 bytes da EXTENSAO, o handshake com HMAC-SHA256, o replay e a
# prova errada recusados, a rota de aprovacao a dizer `verificada: false`, e a REVOGACAO que o handshake recusa.*
'--- 76. E2E DO PAREAMENTO REAL COM O COMPANION (17 verificacoes)'
Push-Location apps/hades-browser-bridge
$saidaComp = & node test/e2e-companion-real.mjs 2>&1
$saidaComp | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
$exitComp = $LASTEXITCODE
Pop-Location
if ($exitComp -ne 0) {
  $falhas.Add("e2e-companion-real.mjs (exit $exitComp)")
  '    FALHA'
} else { '    OK' }
$passos += 'companion-pareamento'


''
# E-1119: o NATIVE HOST instalado de forma REVERSIVEL. *O roundtrip corre numa SANDBOX (o `LOCALAPPDATA`
# redireccionado e `-Navegador ''`), logo nao toca no registo real: instala, o `INSTALACAO.json` lista tudo o que
# criou, e o `-Desinstalar` remove exactamente isso -- inclusive a pasta que fica vazia.*
'--- 77. NATIVE HOST: instalacao REVERSIVEL (roundtrip na sandbox)'
$saidaNh = & python -X utf8 scripts/hades_native_host_roundtrip.py 2>&1
$saidaNh | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_native_host_roundtrip.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'native-host-reversivel'


''
# E-1120: o TIMEOUT do CI. *Todo `go test` que corre o `internal/agent` (ou o `./...` da raiz) declara um
# `-timeout` >= 20 min, e ele cabe no `timeout-minutes` do job. E' DECLARAR o tempo medido (353,4 s com 1722
# testes, e o maior teste so'zinho = 66,06 s a encher 5 000 produtos) -- NAO e' esconder deadlock.
'--- 78. TIMEOUT DO CI: o `-timeout` declarado nos `go test` do agent'
$saidaTimeout = & python -X utf8 scripts/hades_ci_timeout_check.py 2>&1
$saidaTimeout | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_ci_timeout_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'ci-timeout'


''
# E-1121: a extensao nos MOTORES REAIS. *Os 2 E2E (DOM e takeover) passam no Chromium do Playwright E no EDGE
# REAL -- o `E-1105` tinha dado o Edge por perdido, e ele funciona. O Chrome real continua sem o `--load-extension`
# (removido no 137+), e isso e' REPORTADO, nao exigido: PARTILHAR O MOTOR NAO E' PARTILHAR A POLITICA.
'--- 79. EXTENSAO nos MOTORES REAIS: Chromium e Edge'
$saidaMotores = & python -X utf8 scripts/hades_extension_engines_check.py 2>&1
$saidaMotores | Select-Object -Last 4 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_extension_engines_check.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'extensao-motores'


''
# E-1122: o FIREFOX -- a extensao INSTALA e ARRANCA num Firefox REAL. *O `web-ext` (a ferramenta do PROPRIO
# Mozilla) diz `Installed` e o Firefox 157.0 do Playwright arranca com 7 processos. O `operar` (DOM, clique) fica
# declarado como nao provado: o Playwright nao controla o Firefox que o `web-ext` lanca.
'--- 80. FIREFOX: a extensao INSTALA e ARRANCA num Firefox real'
$saidaFfE2E = & python -X utf8 scripts/hades_firefox_e2e.py 2>&1
$saidaFfE2E | Select-Object -Last 3 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
if ($LASTEXITCODE -ne 0) {
  $falhas.Add("hades_firefox_e2e.py (exit $LASTEXITCODE)")
  '    FALHA'
} else { '    OK' }
$passos += 'firefox-instala'


''
# E-1131: o GOFUMPT no gate local. *A licao do E-1130: o CI ficou vermelho com `gofumpt: 3 issues` e o gate local,
# que corre `gofmt`, PASSOU -- e eram 24 ficheiros, nao 3. Um gate local mais fraco do que o CI e' um atraso: ele
# diz verde e o vermelho so' aparece no runner, depois do push. E o `--diff` e' obrigatorio: um gate que ALTERA o
# codigo que mede nunca pode falhar -- e nunca pode avisar.
'--- 81. GOFUMPT: o CI usa-o, e o gate local tem de o usar tambem'
$fumpt = $null
foreach ($c in @("$Raiz\_tmp\gopath\bin\golangci-lint.exe", (Get-Command golangci-lint -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source -ErrorAction SilentlyContinue))) {
    if ($c -and (Test-Path $c)) { $fumpt = $c; break }
}
if (-not $fumpt) {
    $falhas.Add("golangci-lint nao encontrado (o CI usa gofumpt via golangci-lint)")
    '    FALHA: golangci-lint nao encontrado'
} else {
    $saidaFumpt = & $fumpt fmt --enable gofumpt --diff ./internal/agent/ ./server/ ./cmd/ 2>&1
    if ($LASTEXITCODE -ne 0) {
        $saidaFumpt | Select-Object -First 6 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
        $falhas.Add("golangci-lint fmt --enable gofumpt (exit $LASTEXITCODE)")
        '    FALHA'
    } elseif ($saidaFumpt) {
        $saidaFumpt | Select-Object -First 6 | ForEach-Object { "    " + $_.ToString().Substring(0, [Math]::Min(110, $_.ToString().Length)) }
        $falhas.Add("gofumpt: ha' ficheiros fora do formato (o CI vai falhar)")
        '    FALHA'
    } else {
        '    OK'
    }
}
$passos += 'gofumpt'

"GATES LOCAIS: $($passos -join ' + ') VERDES"
