# Backlog priorizado de lacunas de capacidade — 2026-10-09

Recorte escolhido para a Fase (c): **auditoria de lacunas priorizada** (opção 1) —
consolidar, em um único backlog acionável, o que já existe, o que falta e **como
provar** que cada lacuna fechou. Nenhuma funcionalidade nova é implementada aqui.

Base: `audit/COMPARATIVO_HADES_vs_CONCORRENTES_2026-10-04.md` (roadmap de 6 itens),
`audit/ANALISE_COMPETITIVA_COMPLETA_2026-10-04.md` e
`audit/HARNESS_CAPABILITY_MATRIX.md`. Cada item abaixo foi **reverificado no
código de `main` = `91256ac`** antes de ser classificado; o roadmap anterior foi
escrito há 5 dias e parte dele já foi implementada.

## 1. Estado verificado hoje

| # | Item do roadmap (2026-10-04) | Estado em 2026-10-09 | Evidência |
| --- | --- | --- | --- |
| 1 | OAuth E2E live + ampliar provedores | **Parcial**: 122 conectores no catálogo, mas só 17 `available`; 104 `operator_setup_required` e 1 `provider_selection_required`. Não existe teste live de OAuth no repo | contagem em `internal/agent/connector_catalog.go` (`{ID: "` = 122); busca por `OLLAMA_TEST_OAUTH`/`oauth…live` em `internal/agent/*_test.go` = 0 ocorrências; `internal/agent/oauth_clients.go` sem arquivo de teste |
| 2 | Harness Claude/Codex/Copilot no Windows | **Parcial e em outra direção**: existe ponte **reversa** (expõe modelos locais para Claude Desktop e Codex Desktop). Não existe ponte para **executar** os CLIs de dentro do Hades; Copilot não aparece em lugar nenhum | `internal/proxy/claude_desktop.go` (976 linhas, importa `anthropic`), `internal/proxy/codex_desktop.go` (692 linhas); `glob **/*{claude,codex,copilot,harness}*` em `internal/` não retorna nenhum `copilot`, e não há diretório de harness |
| 3 | Browser/computer-use plug-and-play | **Implementado no backend**: status do ambiente + instalação automática (Playwright + Chromium) | `internal/agent/browser_environment.go:40,86,95,128` (`checkBrowserEnvironment`, `BrowserEnvironment`, `SetupBrowserEnvironment`, passos `pip install --user playwright` e `playwright install chromium`); testes `browser_environment_test.go:44,56,78,103,122,143`; CI já roda `browser_helper_test.py` em Linux e Windows |
| 4 | Studio: preview ao vivo + arraste real | **Corrigido em 2026-10-10**: a premissa anterior ("arraste real inexistente") era **falsa**. O arraste existe e foi implementado com DnD nativo do HTML5, sem dependência de biblioteca | `app/ui/app/src/components/StudioCanvasPage.tsx:975` (`draggable`), `:976-988` (`onDragStart`/`onDragOver`/`onDrop`/`onDragEnd`), `:1012-1024` (botões de acessibilidade); `lib/studioReorder.ts` + 5 testes em `lib/studioReorder.test.ts`; persistência em `:337-340` (`updateBuilderVisual`). O que faltava era **asserção E2E**, suprida por `app/ui/app/e2e/studioReorder.spec.ts` + `e2e/studio-builder-fixture.mjs` |
| 5 | Editor visual de automações (nós) | **Ausente**: nenhuma dependência ou componente de canvas de nós | busca por `xyflow|reactflow|workflow-canvas|node-editor|WorkflowCanvas` em `app/ui/app/src` = 0 ocorrências; `package.json` sem `@xyflow/react` |
| 6 | Rebase do Ollama | **Pendente e desassistido**: não há remote `upstream` configurado; HEAD local em `v0.2.0-rc.2-23-g08d1d812` | `git remote -v` = somente `origin` |

Consequência honesta: dos 6 itens do roadmap, **1 está fechado** (browser), **2
estão parciais** (OAuth, harness), **1 foi corrigido e fechado** (Studio — a
premissa de "arraste inexistente" era falsa; faltava asserção E2E) e **2 estão
ausentes** (nós, rebase). Segue **falso** afirmar que o Hades é "melhor e mais
completo que todos"; o que é verificável é o fechamento de lacuna por lacuna.

## 2. Backlog por prioridade

Cada item exige: branch curta a partir de `main`, PR, e os 2 contextos
obrigatórios verdes (`Preserve Class A+ surfaces`, `PR gate`). Nenhum item admite
desabilitar teste, `nolint` novo ou exclusão de lint para obter verde.

### P0-1 — E2E UI-first de primeira execução (Playwright)

- Lacuna: a jornada instalar → onboarding → missão → artefato → aprovação →
  cancelamento → reinício → persistência → atualização → rollback só é coberta
  parcialmente por `test-install.yaml`.
- Aceite: um spec único em `app/ui/app/e2e/` roda a jornada ponta a ponta contra o
  servidor real e falha se qualquer etapa regredir; coberto pelo gate
  `Web and mobile quality`.
- Prova: `npx playwright test <spec>` local + job verde em PR.
- Bloqueio: nenhum. **Começar por aqui** (não depende de credencial nem de
  hardware).
- **Entregue** (branch `feat/e2e-primeira-execucao`, 2026-10-10): spec
  `app/ui/app/e2e/firstRun.spec.ts` mais o backend real de teste
  `app/ui/app/e2e/first-run-backend.mjs`, rodando como passo novo
  `Web E2E — primeira execução` do job `Web and mobile quality`. Cobre
  instalação → onboarding → persistência → reinício → atualização → rollback.
  Escopo declarado: as etapas de missão/artefato/aprovação/cancelamento exigem
  runtime de agente com modelo servido e continuam cobertas por `internal/agent/`
  e `app/ui/app/e2e/scheduleFlow.spec.ts`, não por este E2E.

### P0-2 — Editor visual de automações (nós)

- Lacuna: ausência total; é a maior distância funcional frente a n8n/dify.
- Aceite: canvas com nós, conexões persistidas, execução real de um fluxo de 3
  nós (gatilho → ação → condição) com o mesmo modelo de permissão
  (approval/egress/DLP) já usado pelos fluxos atuais; testes unitários + 1 E2E.
- Prova: testes novos verdes + gate `Web and mobile quality` + `PR gate`.
- Bloqueio: decisão de produto sobre escopo mínimo (1 PR grande ou 3 pequenos).

### P0-3 — Studio: arraste real e preview ao vivo — **ENTREGUE em 2026-10-10**

- **Correção de premissa:** a lacuna "sem drag-and-drop" estava errada. O arraste
  já existia com DnD nativo (`StudioCanvasPage.tsx:975-988`) e a reordenação por
  botão/teclado também (`:1012-1024`), ambos persistindo no servidor por
  `updateBuilderVisual`. A lacuna real era **ausência de asserção E2E**.
- **Entregue:** `app/ui/app/e2e/studioReorder.spec.ts` (3 testes) contra o
  backend HTTP real (`e2e/studio-builder-fixture.mjs`, delegado por
  `e2e/first-run-backend.mjs`), provando: ordem inicial vinda do servidor,
  arraste persistindo `POST /api/agent/v1/builders/{id}/visual`, paridade pelo
  botão acessível, e a ordem sobrevivendo ao reinício em aba nova.
- Aceite cumprido: os componentes são reordenados e a nova ordem é lida do
  servidor após reinício — sem estado local de tela.
- Prova: `npx playwright test` (10 testes, 1 worker) + `npm run lint` +
  `npx vitest run` (65 arquivos / 396 testes) + `npm run build` +
  `prettier --check`, todos verdes nesta rodada.
- Bloqueio: nenhum (o DnD nativo evitou dependência nova, como o projeto prefere).
- Limite declarado: o estado servido é fixture rotulada (projeto com três
  componentes); o caminho exercitado é o real, incluindo o proxy `/api` do
  preview.

### P0-4 / P0-5 — Paridade de chat com o TypingMind (ver `COMPARATIVO_TYPINGMIND_20261009.md`)

- P0-4: respostas **multi-modelo lado a lado** (backend já existe em
  `internal/multillm/`). Aceite: E2E com 1 mensagem e 2 respostas distintas.
- P0-5: **biblioteca de prompts** com variáveis, pastas e reuso; aceite: E2E de
  criar/salvar/reusar com persistência local.
- Ambos atacam a maior desvantagem medida do Hades frente ao TypingMind: tempo
  até o primeiro valor. Nenhum depende de credencial externa.

### P1-4 — Harness de CLIs (Claude/Codex/Copilot) dentro do Hades

- Lacuna: a ponte atual é reversa (Claude/Codex Desktop consumindo modelos do
  Hades). Falta o inverso: iniciar o CLI oficial a partir do Hades, com captura de
  saída, aprovação e DLP, e funcionar no Windows.
- Aceite: comando no Hades dispara o CLI instalado, transmite a saída para o chat
  e aplica approval/DLP; teste de paridade Windows + Linux; sem copiar credenciais
  do CLI.
- Prova: testes novos + `Go agentic Windows portability` verde.
- Bloqueio: `BLOCKED_BY_EXTERNAL_DEPENDENCY` para a parte de **validação com
  conta real** (login do CLI); a implementação e os testes com CLI falso não são
  bloqueados.

### P1-5 — OAuth: E2E live + ampliação de provedores

- Lacuna: 104 dos 122 conectores exigem configuração do operador; não há teste
  live de OAuth.
- Aceite: teste live opt-in (pulado sem credencial, jamais falhando por ausência
  dela) para um provedor real, mais ao menos 3 provedores promovidos de
  `operator_setup_required` para `available` com fluxo verificado.
- Prova: teste com `OLLAMA_TEST_OAUTH=1` quando a credencial existir + regressão
  do catálogo.
- Bloqueio: `BLOCKED_BY_EXTERNAL_DEPENDENCY` — requer app OAuth registrado e
  credenciais do usuário; sem isso, só a parte de catálogo/regressão avança.

### P2-6 — Rebase do Ollama

- Lacuna: sem remote `upstream`, HEAD em `v0.2.0-rc.2-23`.
- Aceite: `upstream` configurado, rebase aplicado em branch, todos os gates
  verdes, sem perder os patches de proxy/harness.
- Prova: histórico linear na branch + 5 contextos obrigatórios de release.
- Bloqueio: nenhum técnico; exige janela de trabalho e revisão de conflitos.

### P3-7 — Plataforma e confiabilidade

- Smoke nativo por SO (cross-compile não é teste nativo) — hoje fora de Linux é
  fail-closed.
- Empacotamento assinado (Authenticode/notarização Apple): `BLOCKED_BY_EXTERNAL_DEPENDENCY`
  (exige certificado).
- RLS PostgreSQL: **continua desabilitado por projeto**; ver
  `audit/CLAUDE_CODEX_RESUME_PROMPT_20260927.md` (prerequisito de papéis/DSNs
  separados e contexto de tenant não forjável). Não é item de backlog executável
  sem essa decisão arquitetural.

## 3. Ordem recomendada

1. **P0-1** (E2E UI-first) — ENTREGUE (PR #71); destrava confiança em todos os
   demais; sem dependência externa.
2. **P0-3** (asserção E2E do arraste no Studio) — ENTREGUE (esta rodada, PR
   empilhado em #71); **P0-2** (nós) continua sendo a maior lacuna de produto
   percebida pelo usuário final.
3. **P1-4** e **P1-5** — paridade de ecossistema, com bloqueios externos
   explícitos nas partes que dependem de credencial.
4. **P2-6** e **P3-7** — manutenção de plataforma e release assinado.

## 4. Critério de "pronto" deste backlog

Um item só sai da lista quando existir: commit em `main`, teste automatizado que
falha se a capacidade regredir, e gate verde correspondente. Relatório não é
prova; teste reproduzível é.
