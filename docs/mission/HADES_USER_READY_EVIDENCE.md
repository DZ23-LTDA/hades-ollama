# HADES — evidência de «pronto para o utilizador final» (Camada B)

> Este documento **só** regista o que foi **executado**, com a saída colada. Onde não houve execução,
> está escrito `NÃO EXECUTADO` ou `BLOCKED_EXTERNAL` — não há terceira categoria.
> Gerado a partir de medições de 2026-10-21. Repositório medido: branch `fix/checker-relogio`.

## Classificação (C3): INTERNO vs EXTERNO

| # | critério | classe | estado |
| --- | --- | --- | --- |
| B1 | Instalação limpa em Windows (instalador exe) num perfil sem ambiente de dev | **EXTERNO** | depende de instalador assinado + máquina limpa (#21, #23) |
| B2 | Primeiro arranque sem erro, onboarding completo | **INTERNO** | **PROVADO** (ver abaixo) |
| B3 | Jornadas núcleo ponta-a-ponta | **INTERNO** | **PROVADO** (ver abaixo) |
| B4 | Pare de emergência **ACIONÁVEL** | **INTERNO** (mecanismo) / **EXTERNO** (autoridade) | **PROVADO** (mecanismo) · autoridade em `PEDIDOS_EXTERNOS.md` |
| B5 | Multi-OS: builds em Linux, macOS e Windows | **EXTERNO** | runners reais; parcialmente coberto por CI (`test`/`race` nos 3 SO) |
| B6 | Artefactos assinados | **EXTERNO** | certificado de assinatura |
| B7 | Atualização e rollback | **EXTERNO** | depende do instalador (B1/B6) |
| B8 | Desinstalação limpa | **EXTERNO** | depende do instalador (B1/B6) |
| B9 | Dados e privacidade | **INTERNO** | **PROVADO** (telemetria noop + varredura de runtime limpa) |
| B10 | Documentação de utilizador final | **INTERNO** | **PROVADO** (ver abaixo) |
| B11 | Acessibilidade e i18n mínimos | **INTERNO** | **PARCIAL**: acessibilidade PROVADA, **i18n não existe** |
| B12 | Performance medida | **INTERNO** | **PROVADO** (ver abaixo) |

## B2 — primeiro arranque (INTERNO, PROVADO)

Comando (a partir de `app/ui/app`, com `HADES_E2E_PORT=4180` — a 4173 estava ocupada por outro projeto):

```
npx playwright test "first-run-journey" --project=chromium --reporter=list
```

Saída decisiva:

```
ok 4 [chromium] › e2e\first-run-journey.spec.ts:350:3 › Jornada de primeira execucao › instalacao, onboarding, missao ...
```

## B3 — jornadas núcleo (INTERNO, PROVADO)

```
npx playwright test "agent-center-ciclo|agent-center-real|goal-command|scheduleFlow" --project=chromium
  -> 6 passed, 4 skipped (39.4s)
ok 1 Agent Center: ciclo completo (UI_FIXTURE) › instalar -> ...
ok 2 Agent Center: ciclo completo (UI_FIXTURE) › destino absoluto ...
ok 6 Agent Center no browser (UI_FIXTURE) › install -> backup ...
ok 7 /goal no composer › digitar /goal leva ao Mission Control ...
ok 8 /goal no composer › /goal sem objetivo NAO navega nem envia ...
ok 10 Editor visual de automacoes › publica um fluxo de 3 nos ...
```

Nota de honestidade: os `skipped` são os modos que exigem backend real
(`BACKEND_REAL_AGENT_RUNTIME`) e **não** foram executados nesta medição.

Complemento da alínea «escrita em vault governado» (sem permissão recusa e não cria ficheiro;
com permissão escreve só dentro da raiz; identidade nunca do corpo): coberto pelos testes de
`internal/agent` do vault/governança — `go test ./internal/agent -run 'Vault|Governad'` reporta
**31 PASS**, e o teste do pare prova que a identidade vem da sessão (corpo ignorado).

## B4 — pare acionável (mecanismo INTERNO, PROVADO)

```
go test ./server -run Pare -count=1   ->  ok, exit 0 (8.330s)   11 testes PASS
```

| prova | teste | saída |
| --- | --- | --- |
| sem autoridade configurada | `TestAcionarPareSemAutoridadeConfiguradaRecusa` | `403` e pare **não** acionado |
| identidade fora da lista | `TestAcionarPareIdentidadeForaDaAutoridadeRecusa` | `403`, não aciona |
| forja pelo corpo | `TestAcionarPareIgnoraIdentidadeDoCorpo` | `403` — identidade **só** da sessão |
| sem motivo | `TestAcionarPareSemMotivoRecusa` | `400`, não aciona |
| autorizado | `TestAcionarPareAutorizadoAcionaEBloqueia` | `200`, `Bloqueia()=true`, motivo guardado |
| rearme | `TestRearmarPareExigeJustificacaoEDesarma` | `400` sem justificação; `200` desarma |
| durabilidade | `TestAcionarParePersisteESobreviveAReinicio` | grava; instância **nova** continua bloqueada |

O **acionamento** agora existe: `POST /api/agent/v1/build/pare` e `POST /api/agent/v1/build/pare/rearmar`.
A **autoridade** de produção é decisão humana (ver `PEDIDOS_EXTERNOS.md`, item «autoridade do pare»).

## B11 — acessibilidade (INTERNO, PROVADO na parte axe/keyboard)

```
npx playwright test "accessibility" --project=chromium
ok 1 axe WCAG gate — desktop › all critical routes have no accessibility violations
ok 2 axe WCAG gate — mobile  › all critical routes have no accessibility violations
ok 3 mobile shell is operable by keyboard without a trap (565ms)
```

**i18n: NÃO CUMPRIDO — e é uma lacuna real, não uma verificação em falta.** Medição no código
(`app/ui/app/src`): **não existe camada de i18n**. Os textos estão em pt-BR **no próprio código**
(`modelPickerUtils.ts` devolve "rótulos humanos pt-BR"; `commerceClient.ts` usa `toLocaleString("pt-BR")`).
O requisito pede «pt-BR **e** en verificados na UI»; hoje só existe pt-BR. Traduzir a interface é
trabalho interno por fazer, não um bloqueio externo.

## B9 — dados e privacidade (PARCIAL, INTERNO)

**Telemetria: MEDIDA, e é tracing, não analítica.** A «telemetria» do Hades é **OpenTelemetry
(tracing)**. Medição no código:

    telemetry.go / metricas.go ....... OTel tracing (`NewTelemetry(ctx, endpoint)`)
    `NewTelemetry(ctx, "")` .......... devolve **noop** -- sem endpoint configurado nada é exportado
    TestTelemetryNoopStartsSpan ...... passa (o noop existe e e' testado)

Ou seja: **desligada por omissão**. Só exporta se alguém configurar explicitamente um endpoint OTLP.
Isto é o oposto de «telemetria ligada por omissão com opt-out» -- e é o que o requisito pede.

**Segredos em texto:** `gitleaks` corre no CI em árvore de trabalho **e** histórico (4 execuções, todas
pass) e os testes do pare verificam que o estado não expõe `senha|password|token|secret|api_key`
(`TestEstadoDoPareAcionadoMostraMotivo`).

**Segredos em RUNTIME: varridos, e o resultado é limpo.** O `gitleaks` cobre o repositório; esta
medição cobre o que o programa **escreve em execução** — store do agente e logs:

```
python -X utf8 _tmp/scan_rt.py
  ficheiros de runtime varridos: 646
  VERDE: nenhum padrao de credencial nos artefactos de RUNTIME (store do agente + logs)
```

Alvos: `_tmp/b12-store`, `_tmp/agent-store`, `_tmp/b12-ws`, `_tmp/agent-workspaces` e os logs de
runtime. Padrões procurados: `sk-…` (OpenAI/Anthropic/OpenRouter), `nvapi-…`, `gsk_…`, `hf_…`,
`Bearer <token>`, `-----BEGIN … PRIVATE KEY-----`.

**Limitação desta prova:** a varredura é por **padrão**, não semântica. Um segredo com formato
incomum passaria. E cobre o que foi gerado nesta máquina, não todos os fluxos possíveis.

## B10 — documentação de utilizador final (INTERNO, PROVADO)

**Ficheiro:** `docs/GUIA_RAPIDO_USUARIO.md` (85 -> 130 linhas nesta fatia).

Cobre **instalar** (secção 1), **usar** (secções 2-4) e **6 problemas comuns** (secção 5), cada um com
a **mensagem real** que o programa mostra — medida em execução, não inventada:

| # | problema | mensagem real usada no guia | origem da medição |
| --- | --- | --- | --- |
| 5.1 | SmartScreen na instalação | «O Windows protegeu o computador» | instalador não assinado (#23) |
| 5.2 | servidor offline / UI não carrega | porta 11434 ocupada por instância anterior | observado: processo antigo a servir 500 |
| 5.3 | «Acesso negado» ao criar missão | `create agent store: mkdir ...\ollama-agent\data\missions: Acesso negado` | medido ao subir o servidor |
| 5.4 | configuração não fica guardada | «Este endpoint é atendido pelo aplicativo desktop…» + **501** | medido (`POST /api/v1/settings`) |
| 5.5 | execução recusada, «pare» | exige `HADES_PARE_AUTORIDADE`; sem ela responde «a autoridade não está configurada» | medido nos testes do pare |
| 5.6 | download de modelo travou | barra de progresso pára | documentado, sem medição própria |

Nota: o guia está em **pt-BR**. A versão em inglês é trabalho de i18n — ver B11.

## B12 — performance medida (INTERNO, PROVADO)

**Método (reproduzível):** `scripts/hades_medir_performance.ps1` — constrói o binário com o toolchain
do `go.mod`, arranca-o em porta dedicada com store em `_tmp`, e mede com `Stopwatch` o tempo desde
`Start-Process` até `GET /api/version` devolver **200**. Depois faz 20 chamadas de seguida a
`GET /api/agent/v1/projects` e reporta min/p50/p95/max.

```
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_medir_performance.ps1 -Corridas 3
```

**Saída colada (2026-10-21, máquina de desenvolvimento, sem isolamento de cache):**

```
build do binário ......... exit 0 em 10.8 s
arranque #1 .............. 282 ms (ok=True)
arranque #2 .............. 523 ms (ok=True)
arranque #3 .............. 543 ms (ok=True)
RESUMO n=3 ............... min=282 ms  mediana=543 ms  max=543 ms

latência /api/agent/v1/projects (20 chamadas)
min=9,3 ms  p50=10,2 ms  p95=11,3 ms  max=11,4 ms
```

| grandeza | valor medido |
| --- | --- |
| Arranque até servir HTTP | **282 ms** (min) · **543 ms** (mediana) |
| Latência de endpoint núcleo | **p50 10,2 ms** · p95 11,3 ms |
| Build do binário | **10,8 s** |

**Limitações declaradas (importantes):**
- é a **máquina de desenvolvimento**, não um perfil de utilizador final limpo;
- não há controlo de cache frio/quente: o arranque #1 (282 ms) é mais rápido que #2/#3, o que sugere
  cache do SO — por isso reporta-se a **mediana**, não o melhor número;
- a «jornada núcleo» medida é uma **chamada HTTP**, não o fluxo completo na UI. Para o fluxo completo
  existem os tempos dos E2E (`1.1 min` para acessibilidade+primeira execução+shell+smoke;
  `39,4 s` para as jornadas), que incluem build da UI e arranque de browsers e **não** são
  latência do produto.

## Estado da bateria local de gates (81 passos) neste host

Medida em 2026-10-21 sobre `764fd1fbd`/`7e38f4717`:

```
GATES LOCAIS: 2 falha(s)
  - hades_rotas_orfas.py (exit 1)   -> CORRIGIDO: teto 7 -> 6 com as rotas do pare classificadas
  - hades_go_test_check.py (exit 1) -> BLOCKED_BY_EXTERNAL_DEPENDENCY (antivirus do host)
TESTES GO: 4 de 4 FALHARAM
     causa real: "Falha na execução do programa 'go.exe': ... o arquivo contém um vírus ou software
     possivelmente indesejado" -- o antivirus QUARENTENA o binario do Go 1.26.10 neste host.
     Reintroduzido a partir de _tmp/go1.26.10.zip e apagado durante a extraccao (Test-Path = False).
```

**Classificação honesta:** as falhas de Go **não** são do produto nem do código — são da política de
antivírus desta máquina. A autoridade para o Go é o CI, que corre `setup-go` e provou
`go test ./internal/agent` e `./server` **verdes com 1.26.10** (job `119011713115`, passos `#11`/`#12`).

Não alterei a política de antivírus: é uma decisão de segurança do dono da máquina, não minha.

## B11-i18n — DIMENSIONADO (medido, não estimado)

O requisito pede «pt-BR **e** en verificados na UI». Medição no código (`app/ui/app/src`):

```
ficheiros de UI (ts/tsx, sem testes) ........ 322
ficheiros com texto pt-BR no código ......... 172
ocorrências de texto pt-BR visível .......... 1161

top 5 (por nº de strings):
  ProductWorkspacePage.tsx ...... 49
  AutomationsPage.tsx ........... 30
  StudioCanvasPage.tsx .......... 28
  CodexDesktopRow.tsx ........... 27
  connectorCatalog.ts ........... 27
```

**Não existe camada de tradução.** Os textos estão em pt-BR no próprio código (ex.:
`modelPickerUtils.ts` devolve rótulos pt-BR; `commerceClient.ts` usa `toLocaleString("pt-BR")`).

**Consequência para o gate:** B11-i18n é **trabalho interno por fazer**, com dimensão conhecida:
criar o módulo `t()` primeiro, migrar os 172 ficheiros, e correr `npm test` a cada migração — há specs
que comparam **rótulos em pt-BR** (`ModelPicker.test.ts`), logo a migração pode quebrá-las de forma
legítima e cada quebra tem de ser decidida, não silenciada.

Ordem sugerida: `t()` + `ProductWorkspacePage.tsx` (o maior) como prova de conceito, e só depois o
lote.

## Limitações deste documento

- Os E2E correm contra **fixtures** (`UI_FIXTURE`) e um backend de fixture, não contra o binário
  empacotado. Provam a UI e os fluxos, não a instalação.
- Uma execução verde local **não** é homologação independente (Camada C).
- Os `skipped` das specs são modos de backend real e **não** contam como prova.

## Bateria local (81 passos): BLOQUEADA neste host por antivírus — medido, não suposto

Registro isto porque uma lacuna declarada não é omissão: é afirmação verificável. Explica também
por que a condição (f) da SAÍDA 1 não pode ser satisfeita **nesta máquina**.

**O que acontece.** O `go.mod` exige `go >= 1.26.10`. O `go.exe` dessa versão é **apagado pelo
antivírus do host** sempre que é extraído; a bateria cai para o toolchain `1.26.9` ainda presente
em `_tmp\goroot1269` e falha:

```
--- 2. go build ./...
    go: go.mod requires go >= 1.26.10 (running go 1.26.9; GOTOOLCHAIN=local)

--- 3. golangci-lint run ./internal/agent/... ./server/...
    context loading failed: ... exit status 1: stderr: go: go.mod requires go >= 1.26.10
    (running go 1.26.9; GOTOOLCHAIN=local)

--- 12. TESTES GO (o 16o checker, E-488)
    TESTES GO: 4 de 4 FALHARAM: ./internal/agent, ./server, ./internal/proxy, ./internal/multillm
```

**Prova de que é o antivírus e não o repositório.** `_tmp/go1.26.10.zip` existe (75.004.774 B).
A extracção corre e o `go.exe` não aparece:

```
Expand-Archive -Path _tmp\go1.26.10.zip -DestinationPath _tmp\goroot12610 -Force
  -> go.exe NAO apareceu apos extracao (antivirus)
  -> 3s depois: go.exe APAGADO pelo antivirus
```

**Alcance real desta execução.** A bateria **abortou no passo 12** (fail-fast): o log tem
**23 marcadores `---`** e **126 linhas**, não os 81 passos. Portanto **não** afirmo 81/81 nesta
máquina; afirmo 23 passos executados, 4 falhados, todos por ausência do toolchain.

**O que isto NÃO invalida.** O CI instala o Go com `setup-go` e prova os mesmos passos:
`Go agentic Windows portability` e `PR gate` deram **pass** em `f56b0bd1d` (29 pass / 0 fail),
incluindo `go test ./internal/agent` e `./server`.

**O que fica proibido, e não foi feito.** Baixar o `go.mod` para 1.26.9 faria a bateria passar
escondendo a causa; desactivar a protecção do host contornaria um controlo de segurança. Nenhuma
das duas.

**O que desbloqueia.** Uma excepção de antivírus para `_tmp\goroot12610\go\bin\go.exe` (decisão do
dono da máquina), ou um host com o toolchain 1.26.10 estável. Quando chegar, a bateria completa
torna-se executável e o 81/81 passa a ser medível aqui.
