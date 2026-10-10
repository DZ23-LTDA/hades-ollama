# HADES — RELATÓRIO POR ESTÁGIO (1–38)

> Rodada: checkpoint V2.1 · 2026-10-10 · base `main@889750abe`
> Convenções em `HADES_MASTER_GOAL.md §5`. **Título do estágio é rótulo
> operacional; a numeração é a autoridade.**

## 1. Sumário

| Estado | Estágios | Qtd |
| --- | --- | --- |
| `COMPLETED_VERIFIED` | — | 0 |
| `TESTING` | — | 0 |
| `IN_PROGRESS` | 1,2,3,4,6,7,9,10,11,12,13,14,16,17,18,19,20,21,22,33,34,35,36,37 | 24 |
| `NOT_STARTED` | — | 0 |
| `FAILED` | — | 0 |
| `BLOCKED_BY_EXTERNAL_DEPENDENCY` | 5,8,15,23,24,25,26,27,28,29,30,31,32,38 | 14 |

**Nenhum estágio está `COMPLETED_VERIFIED`.** Isso é consequência da regra de
evidência: não há, nesta rodada, prova reproduzível de ponta a ponta de nenhum
estágio, e vários estágios (5, 8, 15) sequer foram inspecionados
(`auditoria: PENDENTE`). Os 14 estágios marcados como bloqueados dependem de
terceiros (token, conta, telefonia, certificado) ou de dados reais de estágios
anteriores; o bloqueio **não** interrompe os independentes.

## 2. Tabela completa

| # | Estágio | Estado | Auditoria | Capacidade | Evidência | Bloqueio |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Auditoria total e inventário real | IN_PROGRESS | parcial | PARCIAL | E-001, E-002, E-021, E-023 | — |
| 2 | Estabilização e consolidação da base | IN_PROGRESS | parcial | PARCIAL | E-011, E-013, E-014, E-015, E-016, E-017 | B-09 |
| 3 | Arquitetura canônica dos componentes | IN_PROGRESS | PENDENTE | PENDENTE_DE_AUDITORIA | — | — |
| 4 | Model Gateway (rotação, fallback, orçamento) | IN_PROGRESS | PENDENTE | PENDENTE_DE_AUDITORIA | E-024 | B-07 |
| 5 | Context Bootstrap / contexto de projeto | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-07 |
| 6 | Harness Runtime governado | IN_PROGRESS | parcial | PARCIAL | E-025 | — |
| 7 | MCP Manager | IN_PROGRESS | parcial | APENAS_CATALOGO | E-018, E-019, E-026 | B-10 |
| 8 | Plugins e Skills | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-07 |
| 9 | Agentes e subagentes | IN_PROGRESS | PENDENTE | PENDENTE_DE_AUDITORIA | — | — |
| 10 | Mission Control multiagente | IN_PROGRESS | parcial | PARCIAL | E-020, E-027 | — |
| 11 | Paridade Codex/Claude | IN_PROGRESS | parcial | PARCIAL | E-025 | B-02 |
| 12 | Studio / Builder visual | IN_PROGRESS | **AUDITADO** | IMPLEMENTADO_NAO_HOMOLOGADO | E-008, E-009, E-028 | — |
| 13 | Workflow Engine | IN_PROGRESS | parcial | IMPLEMENTADO_NAO_HOMOLOGADO | E-020, E-029 | — |
| 14 | Browser Operator / computer-use | IN_PROGRESS | parcial | PARCIAL | E-015 | B-09 |
| 15 | Memória e RAG local | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-07 |
| 16 | Canais de IM | IN_PROGRESS | parcial | APENAS_ADAPTER | E-030 | B-03 |
| 17 | Workspace do agente | IN_PROGRESS | parcial | PARCIAL | E-027 | — |
| 18 | Segurança, permissões e multitenancy | IN_PROGRESS | parcial | PARCIAL | E-019, E-031 | — |
| 19 | UX/onboarding e primeira execução | IN_PROGRESS | **AUDITADO** | IMPLEMENTADO_E_TESTADO | E-011, E-012, E-032 | — |
| 20 | Distribuição e instaladores | IN_PROGRESS | parcial | PARCIAL | E-022 | B-06 |
| 21 | Benchmarks e comparativo | IN_PROGRESS | PENDENTE | PARCIAL | E-021 | — |
| 22 | Gate de engenharia intermediário | IN_PROGRESS | parcial | PARCIAL | E-005, E-006, E-007, E-013, E-014, E-017 | — |
| 23 | Company Factory | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-07 |
| 24 | CRM e pipeline de vendas | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-05 |
| 25 | Commerce / loja e pedidos | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-05 |
| 26 | Marketplace multi-vendedor | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-05 |
| 27 | Pagamentos e faturamento | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-05 |
| 28 | Atendimento e suporte | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-03 |
| 29 | Telefonia e call center | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-04 |
| 30 | Marketing e campanhas | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-05 |
| 31 | Social e publicação | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-05 |
| 32 | BI / business status | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-07 |
| 33 | Canal Telegram | IN_PROGRESS | parcial | APENAS_ADAPTER | E-030 | B-03 |
| 34 | Canal WhatsApp | IN_PROGRESS | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-03 |
| 35 | Inbox unificado omnicanal | IN_PROGRESS | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-03 |
| 36 | Voz e transcrição | IN_PROGRESS | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-04 |
| 37 | Chamadas e follow-up | IN_PROGRESS | PENDENTE | PENDENTE_DE_AUDITORIA | — | B-04 |
| 38 | Gate final de release verificado | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | AUSENTE | E-033 | B-06 |

## 3. Detalhe dos estágios auditados

### Estágio 1 — Auditoria total e inventário real

- **Feito:** SHA de `main` reconferido ao vivo; 29 PRs abertos enumerados com
  branch e head; `docs/` e `audit/` inventariados; `docs/mission/` inexistente
  antes desta rodada (criado aqui).
- **Pendente:** os 30 documentos datados de `audit/` são pistas, não prova —
  precisam ser reconfirmados por comando antes de sustentar qualquer
  classificação de capacidade.
- **Próxima ação:** reconfirmar, um por um, os documentos que sustentam
  capacidades promovidas.

### Estágio 2 — Estabilização e consolidação

- **Feito:** gates locais verdes (`gofmt -l` vazio, gofumpt exit 0,
  golangci-lint `0 issues.`, `go vet` exit 0, `go build ./...` exit 0, lint web
  exit 0, 65 arquivos/396 testes de vitest verdes, build de produção ok);
  classificador de CI com 13/13 testes.
- **Limitação real:** 4 falhas de teste são **ambientais/pré-existentes**
  (provadas em `main` limpo), não regressões: import de upload/worktree (exit
  128), browser operator (Python ausente, 9009), kill de process group (`sh`
  ausente) e `Filename too long` em `./server/`.
- **Próxima ação:** sanear o ambiente de teste ou marcar explicitamente como
  ambiente-dependente sem desativar teste.

### Estágio 4 — Model Gateway

- **Feito:** PR #60 aberto implementa rotação automática de gateway.
- **Limitação:** **não verificado nesta sessão** — não inspecionei o diff nem
  rodei chamada real a provedor. Não há evidência anexada.
- **Bloqueio:** B-07 (chaves de provedor para homologação de chamada real).

### Estágio 6 / 11 — Harness Runtime governado e paridade Codex/Claude

- **Feito:** `harness.cli` existe **apenas** no branch de PR #62
  (`feat/harness-cli-governado`), com `ScopeHarnessCLI = "harness:cli"`, tool
  `harness.cli.exec`, `Risk: RiskWrite`, `RequiresApproval: true`.
- **Limitação:** branch-local ⇒ **não está em `main`**; homologação com conta
  real logada é B-02.

### Estágio 7 / 18 — MCP Manager e segurança de permissões

- **Feito:** política de capacidades com 16 escopos canônicos
  (`workspace:read|workspace:write|terminal:allowlisted|sandbox:execute|...|repo:read`);
  `validateAutoRunCapabilities` **recusa** tudo que não seja vazio ou
  `workspace:read`; catálogo de conectores medido em 122 linhas com `Auth`
  sempre não-vazio (`SEM_SCOPES=0`) e 6 testes de OAuth + 3 de catálogo verdes
  no PR #70.
- **Limitação:** dos 122 conectores, **104 exigem configuração do operador**
  (`available=17`, `provider_selection_required=1`) ⇒ classificação honesta
  `APENAS_CATALOGO`, sem promover conector a "funcional".

### Estágio 12 — Studio / Builder visual (**correção de auditoria**)

- **Correção importante:** o backlog anterior afirmava que o Studio **não tinha
  drag-and-drop real**. **Isso é falso** e foi verificado nesta rodada:
  `StudioCanvasPage.tsx` possui `draggable` (975), `onDragStart` (976),
  `onDragOver` (980), `onDrop` (984), `onDragEnd` (988), além dos botões de
  acessibilidade `ArrowUpIcon`/`ArrowDownIcon` (1012/1024); ambos os caminhos
  chamam `syncComponents` ⇒ `updateBuilderVisual` (persistência). O arraste usa
  o helper puro `reorderById`, com **5 testes unitários** próprios.
- **Gap real:** falta **cobertura E2E de asserção** (spec Playwright) do canvas;
  os 14 scripts `.mjs` de `e2e/` são captura de evidência, não asserção.
- **Próxima ação:** escrever spec de reorder (arraste + botão + persistência).

### Estágio 13 — Workflow Engine

- **Feito:** fluxo agendado persistido (PRs #67/#68) com `CreateSchedule`,
  `ValidateScheduleSteps` e execução via `ExecuteScheduleFlow`; correção de
  `misspell` por `settings.misspell.ignore-rules` (sem `//nolint`); testes de
  `flowGraph` 31/31 verdes.
- **Limitação:** `action.http`/`action.connector` e o status por passo não estão
  comprovados; persistência de desfecho por passo é pergunta aberta.
- **Próxima ação:** provar execução real de schedule persistido ponta a ponta.

### Estágio 19 — UX/onboarding e primeira execução (o item mais forte desta rodada)

- **Feito (PR #71, verde):** spec Playwright `e2e/firstRun.spec.ts` + backend
  fixture HTTP real sem dependências (`e2e/first-run-backend.mjs`, porta 43117)
  provando a jornada completa: gate de onboarding lido do **servidor**
  (`GET /api/v1/settings`), persistência de `OnboardingVersion`, reinício,
  atualização e rollback. `workers: 1` elimina corrida entre projetos.
- **Evidência de gate:** o passo virou `15 | Web E2E — primeira execução |
  success` dentro do job cujo nome literal `Web and mobile quality` é exigido
  por `pr-gate.yaml:48-79`, `release-readiness.yaml:72` e `release.yaml:836` ⇒
  o E2E é gateado **sem** editar lista de checks obrigatórios.
- **Limitação declarada:** o download de um modelo de vários GB está fora do
  escopo de CI; o fixture **é rotulado como stub**. A asserção provada é o gate
  de UI (sem `Continuar` antes do fim do stream; com `Continuar` depois).
- **Bug de produto em aberto:** se `GET /settings` falhar, o `beforeLoad` de
  `routes/index.tsx` engole o erro e cai no shell, sem redirecionar ao
  onboarding. Decidir se é defeito de produto (documentar) e não apenas de teste.

### Estágio 22 — Gate de engenharia intermediário

- **Feito:** `pr-gate.yaml` roda em **todo** PR e pula o poll profundo quando o
  classificador retorna `DOCS_ONLY`/`require_agentic_gates=false`; PR docs-only
  #57 ficou `PR gate | success` com os gates profundos `skipped`. O head de #71
  tem **23/23 checks verdes**.
- **Limitação:** é gate **intermediário**; não substitui o gate final (38).

## 4. Estágios sem auditoria nesta rodada

3, 4, 5, 8, 9, 15, 20(parcial), 21, 23–32, 34–37. Nenhum deles recebeu
classificação de capacidade canônica; todos estão marcados
`PENDENTE_DE_AUDITORIA`. **Próximo passo obrigatório do Estágio 1:** inspecionar
cada um com comando reproduzível antes de qualquer promoção.
