# HADES — MATRIZ DE CAPACIDADES

> **Taxonomia canônica (7 valores):** `IMPLEMENTADO_E_TESTADO`,
> `IMPLEMENTADO_NAO_HOMOLOGADO`, `PARCIAL`, `APENAS_ADAPTER`,
> `APENAS_CATALOGO`, `AUSENTE`, `BLOQUEADO_EXTERNAMENTE`.
> `PENDENTE_DE_AUDITORIA` **não** é valor canônico: marca superfície não
> inspecionada e **não** promove nada.
> **Risco:** `CRITICO` / `ALTO` / `MEDIO` / `BAIXO`.

## 1. Jornada do usuário e onboarding

| Capacidade | Superfície | Classificação | Risco | Evidência | Lacuna |
| --- | --- | --- | --- | --- | --- |
| Gate de onboarding lido do servidor | `routes/index.tsx:8-38` | IMPLEMENTADO_E_TESTADO | CRITICO | E-011 | erro de `getSettings` é engolido (P0-4) |
| Tela de onboarding (intro/welcome/run) | `components/Onboarding.tsx` | IMPLEMENTADO_E_TESTADO | ALTO | E-011 | — |
| Persistência de `OnboardingVersion` na API de settings | `app/ui/ui.go:1649-1684`; `ui_test.go:855` | IMPLEMENTADO_E_TESTADO | CRITICO | E-011 | — |
| Download do primeiro modelo com gate de UI | `api.ts:589-621` | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-011 | CI usa stub rotulado; download real fora de escopo |
| E2E de primeira execução com backend real | `e2e/firstRun.spec.ts`, `e2e/first-run-backend.mjs` | IMPLEMENTADO_E_TESTADO | CRITICO | E-011, E-012 | 7 cenários no total do conjunto |
| E2E gateado no CI | workflow passo 15 + `pr-gate.yaml:59` | IMPLEMENTADO_E_TESTADO | CRITICO | E-011, E-032 | — |

## 2. Studio / Builder visual

| Capacidade | Superfície | Classificação | Risco | Evidência | Lacuna |
| --- | --- | --- | --- | --- | --- |
| Canvas de construção visual | `StudioCanvasPage.tsx` | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-008 | E2E cobre o reorder; preview/export e undo/redo ainda não |
| Reordenação por arraste | `StudioCanvasPage.tsx:975-988` | IMPLEMENTADO_E_TESTADO | MEDIO | E-008, E-034 | premissa anterior ("ausente") estava errada |
| Reordenação por botão/teclado | `StudioCanvasPage.tsx:1012,1024` | IMPLEMENTADO_E_TESTADO | MEDIO | E-008, E-034 | paridade provada em E2E |
| Persistência da ordem no servidor | `:337-340` → `updateBuilderVisual` | IMPLEMENTADO_E_TESTADO | ALTO | E-008, E-009, E-034 | ordem lida do servidor após reinício em aba nova |
| Helper puro de reordenação | `lib/studioReorder.ts` | IMPLEMENTADO_E_TESTADO | BAIXO | E-009 | unitário apenas |
| Auxiliares de IA/HTML do Studio | `studioAI.test.ts`, `studioHtml.test.ts` | IMPLEMENTADO_E_TESTADO | MEDIO | E-028 | — |

## 3. Agentes, missões e orquestração

| Capacidade | Superfície | Classificação | Risco | Evidência | Lacuna |
| --- | --- | --- | --- | --- | --- |
| Criação de missão e plano | `server/agent_routes.go:527` | PARCIAL | ALTO | E-027 | orçamento/pausa/cancelamento sem prova |
| Merge de missão | `server/agent_routes.go:542` | PARCIAL | ALTO | E-027 | sem prova de conflito/rollback |
| SSE de eventos de missão | `runtime.go:2356`; `store.go:459,909` | PARCIAL | ALTO | E-027 | sem helper genérico de emissão |
| Aprovações de missão | limites `maxMissionApprovals` | PARCIAL | CRITICO | E-026 | fluxo de aprovação sem prova E2E |
| Delegação a subagentes | — | PENDENTE_DE_AUDITORIA | ALTO | — | não auditado |
| Harness CLI governado | `internal/agent/cli_harness_tool.go` | PARCIAL | CRITICO | E-025 | **branch-local** (PR #62), não em `main` |

## 4. Fluxos, schedules e automação

| Capacidade | Superfície | Classificação | Risco | Evidência | Lacuna |
| --- | --- | --- | --- | --- | --- |
| Criação/validação de schedule | `context.go:739,835` | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-029 | desfecho por passo não persistido |
| Execução de schedule | `supervisor.go:291-314`; `runtime.go:1376-1392` | PARCIAL | ALTO | E-029 | execução real não provada |
| Grafo de fluxo e editor persistido | PRs #67/#68 | IMPLEMENTADO_NAO_HOMOLOGADO | MEDIO | E-020 | `action.http`/`connector` em aberto |
| Condições (`condition.if` com `expect`) | — | AUSENTE | MEDIO | — | opcional no backlog |

## 5. Segurança, permissões e tenancy

| Capacidade | Superfície | Classificação | Risco | Evidência | Lacuna |
| --- | --- | --- | --- | --- | --- |
| Política de capacidades (16 escopos) | `capability_policy.go:27` | IMPLEMENTADO_E_TESTADO | CRITICO | E-026 | — |
| Recusa de escopo elevado em auto-run | `agent_routes.go:1819` | IMPLEMENTADO_E_TESTADO | CRITICO | E-026 | — |
| Catálogo de escopos exposto em `/tools` | `agent_routes.go:1258`; `types.go:156-163` | IMPLEMENTADO_E_TESTADO | MEDIO | E-026 | — |
| Concessão de capacidades na UI | PR #66 | IMPLEMENTADO_NAO_HOMOLOGADO | CRITICO | E-031 | PR aberto |
| Store de clientes OAuth | `oauth_clients_test.go` | IMPLEMENTADO_E_TESTADO | ALTO | E-019 | — |
| OAuth contra provedor real | `oauth_live_test.go` | BLOQUEADO_EXTERNAMENTE | ALTO | E-019 | exige `OLLAMA_TEST_OAUTH=1` (B-02/B-07) |
| Isolamento por organização | — | PENDENTE_DE_AUDITORIA | CRITICO | — | sem evidência anexada |

## 6. Integrações, MCP e conectores

| Capacidade | Superfície | Classificação | Risco | Evidência | Lacuna |
| --- | --- | --- | --- | --- | --- |
| Catálogo de conectores (122 linhas) | `connector_catalog.go` | IMPLEMENTADO_E_TESTADO | MEDIO | E-018 | 104 exigem operador |
| Conectores realmente disponíveis | 17 de 122 | APENAS_CATALOGO | ALTO | E-018 | B-10 |
| Conectores MCP catalogados | `Kind=mcp` (4) | APENAS_CATALOGO | ALTO | E-018 | chamada real não homologada |
| Escopos de MCP (`mcp:call`, `mcp:remote:call`) | `capability_policy.go` | PARCIAL | ALTO | E-026 | enforcement de rota remota sem prova |
| Canal Telegram | PR #63 | APENAS_ADAPTER | ALTO | E-030 | token real (B-03) |
| Canal WhatsApp | — | PENDENTE_DE_AUDITORIA | ALTO | — | não auditado |
| Inbox unificado omnicanal | — | PENDENTE_DE_AUDITORIA | ALTO | — | compartilha Omnichannel Runtime |
| Voz e transcrição | — | PENDENTE_DE_AUDITORIA | MEDIO | — | B-04 |
| Chamadas e follow-up | — | PENDENTE_DE_AUDITORIA | MEDIO | — | B-04 |
| Browser operator / computer-use | testes de `internal/agent` | PARCIAL | ALTO | E-015 | falha ambiental (Python) |
| Escopos de desktop/terminal/sandbox | `capability_policy.go` | PARCIAL | CRITICO | E-026 | declarados e governados; execução real não provada ponta a ponta |
| Gateway de modelo com rotação | PR #60 | PENDENTE_DE_AUDITORIA | ALTO | E-024 | diff não inspecionado nesta rodada |

## 7. Plataforma, dados e distribuição

| Capacidade | Superfície | Classificação | Risco | Evidência | Lacuna |
| --- | --- | --- | --- | --- | --- |
| Memória e RAG local | — | PENDENTE_DE_AUDITORIA | ALTO | — | não auditado |
| Context Bootstrap de projeto | — | PENDENTE_DE_AUDITORIA | MEDIO | — | não auditado |
| Plugins e Skills | — | PENDENTE_DE_AUDITORIA | MEDIO | — | não auditado |
| Workspace do agente (threads/artefatos) | PR #69 | IMPLEMENTADO_NAO_HOMOLOGADO | MEDIO | E-027 | PR aberto |
| Saída de ferramenta visível no chat | PR #69 | IMPLEMENTADO_NAO_HOMOLOGADO | MEDIO | E-027 | render opcional do excerto |
| Instaladores por SO | workflows `windows/macos/linux` | PARCIAL | ALTO | E-022 | smoke nativo real ausente |
| Release assinado | `release.yaml` | AUSENTE | CRITICO | E-033 | B-06 |
| Benchmarks competitivos | `audit/COMPARATIVO_TYPINGMIND_20261009.md` | PARCIAL | MEDIO | E-021 | documental, sem metodologia auditável |
| Company Factory / CRM / commerce / marketplace / pagamentos / marketing / social / BI | — | PENDENTE_DE_AUDITORIA | ALTO | — | estágios 23–32 não auditados |

## 8. Leitura da matriz

- **Classificações canônicas atribuídas:** `IMPLEMENTADO_E_TESTADO` em 12 linhas,
  `IMPLEMENTADO_NAO_HOMOLOGADO` em 8, `PARCIAL` em 10, `APENAS_ADAPTER` em 1,
  `APENAS_CATALOGO` em 3, `AUSENTE` em 2, `BLOQUEADO_EXTERNAMENTE` em 1.
  (Atualizado em 2026-10-10 após o PR #73: reorder por arraste, reorder por
  botão/teclado e persistência da ordem passaram a `IMPLEMENTADO_E_TESTADO` com
  E-034.)
- **Sem auditoria:** 12 linhas `PENDENTE_DE_AUDITORIA` — **não contam** como
  capacidade entregue nem como ausente.
- Nenhuma linha foi promovida a `COMPLETED_VERIFIED` (esse valor pertence ao
  eixo de estágio/release, não a esta matriz).
- Um conector no catálogo **não** é um conector funcional: `APENAS_CATALOGO`
  existe exatamente para impedir essa confusão.
