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
| B9 | Dados e privacidade (sem segredos em texto, telemetria desligável) | **INTERNO** | parcial (ver nota B9) |
| B10 | Documentação de utilizador final | **INTERNO** | parcial (ver nota B10) |
| B11 | Acessibilidade e i18n mínimos | **INTERNO** | **PROVADO** (acessibilidade) |
| B12 | Performance medida | **INTERNO** | `NÃO EXECUTADO` |

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

**i18n**: `NÃO EXECUTADO` nesta medição (há UI em pt-BR e textos en no README; falta uma verificação
automática de cobertura das duas línguas).

## B9 — dados e privacidade (PARCIAL, INTERNO)

O que está provado: os testes do pare verificam que o estado **não** expõe
`senha|password|token|secret|api_key` (asserção em `TestEstadoDoPareAcionadoMostraMotivo`), e o
`gitleaks` corre no CI em árvore **e** histórico (pass). O que **não** está provado: «telemetria
desligável» verificado ponta-a-ponta nesta medição.

## B10 — documentação de utilizador final (PARCIAL, INTERNO)

Existem `README.md`, `README.en.md`, `docs/images/screens/SCREEN_CAPTURE_NOTES.md` e a secção de
estado verificável. O que **não** existe: um guia de utilizador com «os 5 erros mais comuns» e a sua
resolução. `NÃO EXECUTADO`.

## B12 — performance (NÃO EXECUTADO)

Nenhuma medição de tempo de arranque ou de uma jornada núcleo foi feita com método e número. Fica
explicitamente por fazer: **não** se declara «rápido» sem medição.

## Limitações deste documento

- Os E2E correm contra **fixtures** (`UI_FIXTURE`) e um backend de fixture, não contra o binário
  empacotado. Provam a UI e os fluxos, não a instalação.
- Uma execução verde local **não** é homologação independente (Camada C).
- Os `skipped` das specs são modos de backend real e **não** contam como prova.
