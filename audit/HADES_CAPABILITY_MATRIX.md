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
| Orquestrador de agentes e subagentes | `swarm.go` (20 func / 10 type) | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-043 | papéis, orçamento, plano, run e cancel testados; sem missão real com modelo |
| Harness CLI governado | `internal/agent/cli_harness_tool.go` | PARCIAL | CRITICO | E-025 | **branch-local** (PR #62), não em `main` |

| Context Bootstrap no chat da superfície Hades | `server/chat_context.go`, rota `/api/agent/v1/chat` (PR #127) | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-069 | injeção verificada e idempotente; prova com dois modelos reais depende de provider (B-07) |

## 4. Fluxos, schedules e automação

| Capacidade | Superfície | Classificação | Risco | Evidência | Lacuna |
| --- | --- | --- | --- | --- | --- |
| Criação/validação de schedule | `context.go:739,835` | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-029 | desfecho por passo não persistido |
| Execução de schedule | `supervisor.go:291-314`; `runtime.go:1376-1392` | PARCIAL | ALTO | E-029 | execução real não provada |
| Grafo de fluxo e editor persistido | PRs #67/#68 | IMPLEMENTADO_NAO_HOMOLOGADO | MEDIO | E-020 | `action.http`/`connector` em aberto |
| Condições (`condition.if` com `expect`) | — | AUSENTE | MEDIO | — | opcional no backlog |

| Segurança: varredura cross-tenant nas superfícies novas | `server/tenant_sweep_test.go` (PR #129) | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-070 | 7 classes de ataque bloqueadas com 403/404; RLS em banco real e não inferência por timing pendentes |

## 5. Segurança, permissões e tenancy

| Capacidade | Superfície | Classificação | Risco | Evidência | Lacuna |
| --- | --- | --- | --- | --- | --- |
| Política de capacidades (16 escopos) | `capability_policy.go:27` | IMPLEMENTADO_E_TESTADO | CRITICO | E-026 | — |
| Recusa de escopo elevado em auto-run | `agent_routes.go:1819` | IMPLEMENTADO_E_TESTADO | CRITICO | E-026 | — |
| Catálogo de escopos exposto em `/tools` | `agent_routes.go:1258`; `types.go:156-163` | IMPLEMENTADO_E_TESTADO | MEDIO | E-026 | — |
| Concessão de capacidades na UI | PR #66 | IMPLEMENTADO_NAO_HOMOLOGADO | CRITICO | E-031 | PR aberto |
| Store de clientes OAuth | `oauth_clients_test.go` | IMPLEMENTADO_E_TESTADO | ALTO | E-019 | — |
| OAuth contra provedor real | `oauth_live_test.go` | BLOQUEADO_EXTERNAMENTE | ALTO | E-019 | exige `OLLAMA_TEST_OAUTH=1` (B-02/B-07) |
| Isolamento por organização (camada de store) | `store.go`, `plugin_scope.go`, `swarm.go` | PARCIAL | CRITICO | E-042, E-043, E-044 | escopo aplicado em store/plugin/orquestrador e tel-agent; RLS PostgreSQL segue desabilitado por decisão arquitetural |

| Egresso: bloqueio de IPv4 embutido em prefixos de transição | `internal/egresspolicy/policy.go`, `EmbeddedIPv4` (PR #131) | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-071 | NAT64/6to4 para metadata, loopback e rede privada agora recusados, com destino público embutido ainda permitido; catálogo de prefixos de transição precisa de revisão contínua |

| Prompt injection: conteúdo externo marcado pela proveniência | `internal/agent/context_bootstrap.go` (`untrustedMemoryKindSource`, `canonicalUntrustedSources`) (PR #133) | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-072 | documento/web/canal ingeridos entram marcados como dado, não instrução; defesa é de contexto, não impede o modelo de ser persuadido por conteúdo malicioso |

| Integridade da interface: nenhum controle finge funcionar | `server/desktop_local_routes.go` (501 + rota ausente sem middleware) (PR #135) | IMPLEMENTADO_NAO_HOMOLOGADO | MÉDIO | E-073 | endpoint de personalização deixou de simular persistência; falta varredura semelhante nas demais telas |

| Workflow: grafo validado + resultado por nó persistido | `internal/agent/schedule_flow.go`, `Schedule.LastRun`, `SupervisorTickResult.ScheduleRunErrors` (PR #137) | IMPLEMENTADO_NAO_HOMOLOGADO | MÉDIO | E-074 | nós ignorados e falhas deixam de ser invisíveis; retry/DLQ por nó ainda não existem |

| Workflow: resultado por nó visível na interface | `app/ui/app/src/lib/scheduleRun.ts`, `ScheduleRunReportCard.tsx`, `AutomationsPage.tsx` (PR #139) | IMPLEMENTADO_NAO_HOMOLOGADO | MÉDIO | E-075 | veredito parcial distinto de sucesso; retry depende de idempotência na criação de missão |

| Missão: criação idempotente por chave | `internal/agent/runtime.go` (`idempotentMissionID`, `missionScopeMatches`), `schedule_flow.go` (PR #141) | IMPLEMENTADO_E_TESTADO | MÉDIO | E-076 | repetir criação não duplica e não cruza tenant; habilita retry por nó com segurança |

| Workflow: retry por nó com tentativas registradas | `internal/agent/schedule_flow.go` (`runWithRetry`, `max_attempts`), `ScheduleStepOutcome.Attempts` (PR #143) | IMPLEMENTADO_NAO_HOMOLOGADO | MÉDIO | E-077 | recupera de falha transitória, limitado e com erro da última tentativa; mesma chave de idempotência impede duplicar |

| Workflow: execução manual do grafo persistido | `server/agent_routes.go` (`runScheduleNow`), `ScheduleRunReport.RunKey` (PR #145) | IMPLEMENTADO_NAO_HOMOLOGADO | MÉDIO | E-078 | disparo manual executa o fluxo real e registra evidência; `run_key` habilita reprocessamento |

| Workflow: reprocessamento reusando a execução | `internal/agent/schedule_flow.go` (`ReprocessScheduleFlow`, `executeScheduleFlow`), `POST /schedules/:id/reprocess` (PR #147) | IMPLEMENTADO_NAO_HOMOLOGADO | MÉDIO | E-079 | reexecuta só pendentes sem duplicar missão; recusa explícita sem execução anterior |

| Workflow: DLQ por nó | `internal/agent/schedule_flow.go` (`ScheduleStepDeadLetter`, `mergeScheduleDeadLetters`), `Schedule.DeadLetters`, `ScheduleRunReportCard` (PR #149) | IMPLEMENTADO_NAO_HOMOLOGADO | MÉDIO | E-080 | falha ganha registro próprio com motivo, limitado e limpo quando o passo se recupera |

## 6. Integrações, MCP e conectores

| Capacidade | Superfície | Classificação | Risco | Evidência | Lacuna |
| --- | --- | --- | --- | --- | --- |
| Catálogo de conectores (122 linhas) | `connector_catalog.go` | IMPLEMENTADO_E_TESTADO | MEDIO | E-018 | 104 exigem operador |
| Conectores realmente disponíveis | 17 de 122 | APENAS_CATALOGO | ALTO | E-018 | B-10 |
| Conectores MCP catalogados | `Kind=mcp` (4) | APENAS_CATALOGO | ALTO | E-018 | chamada real não homologada |
| Escopos de MCP (`mcp:call`, `mcp:remote:call`) | `capability_policy.go` | PARCIAL | ALTO | E-026 | enforcement de rota remota sem prova |
| Canal Telegram | PR #63 | APENAS_ADAPTER | ALTO | E-030 | token real (B-03) |
| Canal WhatsApp | `whatsapp_adapter.go` (18 func), `whatsapp_gateway.go` (31 func), `whatsapp_media.go`, 4 testes | PARCIAL | ALTO | E-045 | adapter + gateway + mídia reais; homologação exige token (B-03) |
| Inbox unificado omnicanal | — | AUSENTE | ALTO | E-047 | sem motor em `internal/agent`; deve nascer sobre o runtime compartilhado (sem duplicar adaptador) |
| Voz e transcrição | — | AUSENTE | MEDIO | E-047 | sem STT/TTS implementados; provedor externo (B-04) |
| Chamadas e follow-up | — | AUSENTE | MEDIO | E-047 | sem motor de telefonia; Tel-Agent exige auditoria de licença e provedor (B-04) |
| Browser operator / computer-use | testes de `internal/agent` | PARCIAL | ALTO | E-015 | falha ambiental (Python) |
| Escopos de desktop/terminal/sandbox | `capability_policy.go` | PARCIAL | CRITICO | E-026 | declarados e governados; execução real não provada ponta a ponta |
| Plugins: manifesto, assinatura, atualização e rollback | `plugin_registry.go`, `plugin_bundle.go` | IMPLEMENTADO_E_TESTADO | ALTO | E-053, E-055 | ciclo de vida com assinatura, digest e rollback testados; sem publisher externo homologado |
| Plugins: **execução governada** | `plugin_runtime.go` (PR #114) | PARCIAL | ALTO | E-063 | oito portas fail-closed e delegação ao `sandbox.exec`; host não-Linux recusa explicitamente; faltam publisher real e UI de estado |
| Rotação/fallback de modelos | `internal/multillm` (`registry.go`, `router.go`, `spend.go`, `egress_zero_trust.go`) | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-037, E-038 | router automático, descoberta, probe, orçamento e egress zero-trust reais; faltam presets guiados e chamada real; PR #60 (rotação) segue não inspecionado |

## 7. Plataforma, dados e distribuição

| Capacidade | Superfície | Classificação | Risco | Evidência | Lacuna |
| --- | --- | --- | --- | --- | --- |
| Memória e RAG local | `ContextStore` (`context.go`), `embedding.go` | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-040 | `AddMemory`, `SearchMemories`, `RetrieveRelevant` com score mínimo e escopo por projeto; retenção/proveniência/exportação não evidenciadas |
| Injeção de contexto em qualquer modelo | `ContextStore` + `capability_policy.go` | PARCIAL | ALTO | E-040 | nenhuma função de bootstrap/injeção localizada; a montagem do contexto não está evidenciada ponta a ponta |
| Skills: catálogo **operável na interface** | `skillCatalog.ts`, `SkillCatalogPanel.tsx`, `ProductWorkspacePage.tsx` (PR #125) | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-068 | resolver por contexto com orçamento, motivos de exclusão visíveis e ativação bloqueada para skill não confiável; falta skill de terceiro real |
| Skills: catálogo com ativação por projeto/agente/modelo | `skill_activation.go`, rotas `/skills/:id/activate`, `/skills/active` (PR #123) | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-067 | seleção determinística com orçamento de tokens e motivo de exclusão; skill não confiável não ativa; falta expor na interface e homologar com skill de terceiro |
| Skills (manifesto, assinatura, não-confiança) | `types.go:207`, `context.go:512`, `supply_chain.go:26-62` | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-042 | ed25519 + testes; ativação por projeto/agente/modelo não evidenciada |
| Workspace do agente (threads/artefatos) | PR #69 | IMPLEMENTADO_NAO_HOMOLOGADO | MEDIO | E-027 | PR aberto |
| Saída de ferramenta visível no chat | PR #69 | IMPLEMENTADO_NAO_HOMOLOGADO | MEDIO | E-027 | render opcional do excerto |
| Instaladores por SO | workflows `windows/macos/linux` | PARCIAL | ALTO | E-022 | smoke nativo real ausente |
| Release assinado | `release.yaml` | AUSENTE | CRITICO | E-033 | B-06 |
| Benchmarks competitivos | `audit/COMPARATIVO_TYPINGMIND_20261009.md` | PARCIAL | MEDIO | E-021 | documental, sem metodologia auditável |
| Company OS (Factory, campanhas, social, supervisor) | `company.go` (36 func), `company_growth.go`, `company_social.go`, `company_agents.go`, `supervisor.go` | IMPLEMENTADO_NAO_HOMOLOGADO | ALTO | E-044 | 11 arquivos de teste; aprovações, idempotência e isolamento cobertos |
| CRM / Commerce / Marketplace / Pagamentos / Inbox / Telefonia | — | AUSENTE | CRITICO | E-047 | sem motor em `internal/agent`; apenas entradas de catálogo de conector |

## 8. Leitura da matriz

- **Atualização da rodada 2 (2026-10-10):** as linhas antes marcadas
  `PENDENTE_DE_AUDITORIA` receberam classificação canônica com base nos probes
  E-037 a E-047. Restam sem auditoria apenas os estágios 21 (benchmark) e 38
  (gate final), fora desta matriz.
- **Classificações canônicas atribuídas:** `IMPLEMENTADO_E_TESTADO` em 12 linhas,
  `IMPLEMENTADO_NAO_HOMOLOGADO` em 8, `PARCIAL` em 10, `APENAS_ADAPTER` em 1,
  `APENAS_CATALOGO` em 3, `AUSENTE` em 2, `BLOQUEADO_EXTERNAMENTE` em 1.
  (Atualizado em 2026-10-10 após o PR #73: reorder por arraste, reorder por
  botão/teclado e persistência da ordem passaram a `IMPLEMENTADO_E_TESTADO` com
  E-034.)
- **Sem auditoria:** 0 linhas nesta matriz após a rodada 2 (antes eram 12).
- **Novos achados da rodada 2:** o Model Gateway é guiado por **tipo de
  protocolo** (`openai-compatible`, `anthropic`, `cli`) com provedores vindos de
  configuração — logo servidores OpenAI-compatible são suportados; falta preset
  guiado e chamada real (E-038). O Harness Runtime está `AUSENTE` em `main`
  (E-041).
- Nenhuma linha foi promovida a `COMPLETED_VERIFIED` (esse valor pertence ao
  eixo de estágio/release, não a esta matriz).
- Um conector no catálogo **não** é um conector funcional: `APENAS_CATALOGO`
  existe exatamente para impedir essa confusão.
