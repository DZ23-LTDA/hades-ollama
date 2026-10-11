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

| Workflow: versão do fluxo executada | `Schedule.FlowVersion`, `ScheduleRunReport.FlowVersion`, `scheduleStepsEqual` (PR #151) | IMPLEMENTADO_E_TESTADO | BAIXO | E-081 | versão sobe só com mudança de grafo e o relatório guarda o que rodou |
| Workflow: homologação com falha simulada, retry e reinício | `internal/agent/schedule_homologation_test.go` (PR #151) | IMPLEMENTADO_E_TESTADO | MÉDIO | E-081 | critério do estágio provado no caminho real, com store em disco e reinício de worker sem duplicata |

| Workflow: variáveis e transformações | `internal/agent/schedule_flow.go` (`action.transform`, `expandScheduleTemplate`, `setScheduleVariable`) (PR #153) | IMPLEMENTADO_E_TESTADO | MÉDIO | E-082 | template puro com limites e variável desconhecida como erro; alimenta o objetivo da missão |
| Workflow: laço limitado | `logic.loop`, `maxScheduleLoopItems`, `maxScheduleMissionsPerRun`, `ScheduleStepOutcome.Children` (PR #153) | IMPLEMENTADO_E_TESTADO | MÉDIO | E-082 | uma missão por item com evidência por iteração e idempotência; teto conferido antes de criar |

| Workflow: aprovação com espera entre execuções | `approval.request`, `Schedule.Approvals`, `Schedule.PendingApprovals`, `DecideScheduleApproval`, `POST /schedules/:id/approvals` (PR #155) | IMPLEMENTADO_E_TESTADO | ALTO | E-083 | pausa o fluxo até decisão humana auditável; negativa não vira falha técnica; aprovação não vaza para execução futura |

| Workflow: nó de conector (caminho real) | `action.connector`, `ConnectorManager.CallForOrganization` (PR #157) | IMPLEMENTADO_E_TESTADO | ALTO | E-084 | escrita exige aprovação e nenhuma chamada sai sem ela; conector ausente ou de outro tenant falha em vez de ser pulado |

| Workflow Automation Engine: ramos if/else | `scheduleStepDependenciesReady`, `pendingSteps` em `schedule_flow.go` (E-085) | IMPLEMENTADO_E_TESTADO | ALTO | E-085 | `condition.if` irmãos com `Expect` opostos formam if/else; ramo alternativo inalcançável e pendente-como-falha foram corrigidos |

| Browser: resolução de interpretador e status honesto | `resolvePythonInterpreter`, `BrowserEnvironment`, `browserPythonExecutable` (E-086) | IMPLEMENTADO_E_TESTADO | MÉDIO | E-086 | candidato validado por execução (não por existência no PATH); status e operador compartilham a escolha |
| Browser: E2E de navegador real (Windows) | `internal/agent/browser_helper.py` + Playwright | BLOQUEADO_EXTERNAMENTE | ALTO | E-086 | Playwright/Chromium ausentes; desbloqueio em 2 comandos |

| Memória: exportação e exclusão escopadas por ator | `ExportMemoriesForActor`, `DeleteMemoryForActor` (E-088) | IMPLEMENTADO_E_TESTADO | ALTO (corrigido) | E-088 | o export **vazava** memória privada de outro usuário e o delete permitia apagar memória alheia; recusa agora indistinguível de inexistente |

| Memória: contexto do modelo escopado por ator | `ContextBootstrapInput.ActorID`, filtro em `BuildContextBootstrap`, `SearchMemoriesForActor` na rota (E-089) | IMPLEMENTADO_E_TESTADO | ALTO (corrigido) | E-089 | o preâmbulo de sistema levava memória privada de outro usuário ao modelo; filtro fail-closed |
| Memória: troca de modelo preservando contexto (ao vivo) | `TestLiveModelSwitchKeepsTheSameContext` — `qwen2.5:7b` ↔ `llama3.2:3b` (E-089) | IMPLEMENTADO_E_TESTADO | — | E-089 | dois modelos reais, contexto byte-idêntico na troca, reinício incluído |

| Comandos slash: `/memory` escopado por ator | `command_routes.go` + `SearchMemoriesForActor` (E-090) | IMPLEMENTADO_E_TESTADO | ALTO (corrigido) | E-090 | o comando devolvia memória privada de outro usuário; invariante de zero leitura de memória sem ator no servidor |

| Segurança: guarda de tenant em handlers HTTP | `scripts/audit_tenant_guards.py` (E-091) | IMPLEMENTADO_E_TESTADO | BAIXO | E-091 | 136 handlers com guarda; 14 exceções nomeadas e explicadas; zero defeito novo |

| Memória: deduplicação | `memory_dedup.go`, `AddMemoryContext` (E-092) | IMPLEMENTADO_E_TESTADO | BAIXO | E-092 | regra conservadora (conteúdo+tipo+visibilidade/ator+origem); duplicata devolve a original; sobrevive ao reinício |

| Memória: versionamento com histórico | `memory_version.go`, `PATCH /memories/:memory_id`, `GET .../history` (E-093) | IMPLEMENTADO_E_TESTADO | BAIXO | E-093 | histórico com conteúdo anterior, quem alterou, data e motivo; teto de 20 revisões; recusa uniforme; vetor antigo invalidado |

| Memória: criptografia em repouso | `memory_encryption.go`, `writeMemoriesFile`, `MemoryEncryption()` (E-094) | IMPLEMENTADO_E_TESTADO | MÉDIO | E-094 | AES-256-GCM do primitivo existente, opt-in, fail-closed sem chave; funil único de escrita; histórico também cifrado |

| Memória: escopos de missão, agente e equipe | `memory_scope.go`, `MemoryScope`, leitores `InScope` (E-095) | IMPLEMENTADO_E_TESTADO | BAIXO | E-095 | propriedade (ator) separada de divulgação (propriedade + escopo); escopo vazio = projeto; regra antiga removida |

| Memória: resumos com digest fiel | `memory_summary.go`, `DigestSummarizer`, rota `/memories/summary` (E-096) | IMPLEMENTADO_E_TESTADO | BAIXO | E-096 | digest determinístico copia trechos com referência; contrato para resumidor injetado; motor declarado |

| Supervisor: incidente, recuperação e logs | `supervisor_incidents.go`, `supervisor_journal.go`, rotas `/supervisor/incidents` e `/supervisor/ticks` (E-097) | IMPLEMENTADO_E_TESTADO | MÉDIO | E-097 | registro idempotente com estado e recuperação observada; diário de ticks persistido com erros; falha de retomada antes engolida agora vira incidente |

| Financeiro: razão e conciliação | `company_ledger.go`, rotas `/companies/:id/ledger` (E-100) | PARCIAL | MÉDIO | E-100 | idempotência por chave por empresa (inclusive concorrente), conciliação com referência externa, fluxo de caixa e fila de pendências; sem integração bancária, fiscal, cobrança ou pagamento |

| Contato: consentimento e opt-out | `consent_registry.go`, enforcement em `im_telegram.go` e `whatsapp_gateway.go` (E-101) | PARCIAL | MÉDIO | E-101 | falha fechada, opt-out com motivo, prazo, escopo por organização e canal, mesma decisão nos dois caminhos; sem UI, CRM ou agentes comerciais |

| CRM: contatos, oportunidades e funil | `crm.go`, rotas `/crm/*`, comando `/crm` disponível (E-102) | PARCIAL | MÉDIO | E-102 | dedupe por identidade com nono dígito, sem identidade recusa, fase terminal imutável, consentimento lido do registro; sem CRM externo nem histórico de conversa |

| Comandos nativos: contrato com rotas reais | guarda `TestEveryAvailableCommandMapsToARegisteredRoute`, porta do Telegram, `/research` corrigido (E-103) | IMPLEMENTADO_E_TESTADO | MÉDIO | E-103 | 32 de 35 comandos com rota existente; 3 indisponíveis **com motivo**; guarda impede regressão |

| Inbox unificado: conversas, SLA e handoff | `inbox.go`, rotas `/inbox/*`, comandos `/inbox`, `/handoff` e `/support` (E-104) | PARCIAL | MÉDIO | E-104 | uma conversa por contato/canal, dedupe de webhook por ExternalID (inclusive concorrente), SLA medido, handoff com motivo, resposta sujeita ao consentimento |

| Omnichannel: contrato, webhook assinado e DLQ | `omnichannel.go`, rotas `/omnichannel/*` (E-105) | PARCIAL | MÉDIO | E-105 | tenant vindo do conector, HMAC-SHA256 em tempo constante com falha fechada, anti-replay, dedupe, rate limit e DLQ com motivo; DLQ e limitador em memória |

| Agentes de atendimento e vendas | `service_agents.go`, rotas `/service-agents` (E-106) | PARCIAL | MÉDIO | E-106 | sete agentes, contexto real (CRM + histórico + consentimento), procedência do texto declarada, escalada sem consentimento e políticas preservadas em qualquer modelo; sem geração por modelo |

| Comércio: catálogo, estoque, pedidos e conciliação | `commerce.go`, rotas `/commerce/*` (E-107) | PARCIAL | MÉDIO | E-107 | estoque como livro de movimentos, checkout idempotente, não vende o que não existe, preço do catálogo, zero dado de cartão; sem fiscal, sem PSP e persistência em memória |

| Financeiro: receita do comércio e conciliação | `finance_ops.go`, rotas `/finance/*` (E-108) | PARCIAL | MÉDIO | E-108 | receita só de pedido pago, idempotente por pedido, lacuna medida, estorno devido sinalizado, conciliação cruzada; sem banco/fiscal/PSP e nenhum dinheiro movimentado |

| Certificação E2E de empresa em sandbox | `business_e2e.go`, rota `/business/certify` (E-109) | PARCIAL | MÉDIO | E-109 | 18 etapas com evidência por etapa, prova de isolamento, idempotência e reinício real, pendências externas declaradas; jornada em sandbox |

| Supervisor de negócio: limites, orçamento e aprovação | `business_supervisor.go`, rotas `/business/tick` e `/business/actions` (E-110) | PARCIAL | MÉDIO | E-110 | ação irreversível aguarda humano, ação reversível é autônoma com efeito real, orçamento bloqueia sem consumir, decisão idempotente com ator; pendências em memória |

| Supervisor de negócio: recuperação persistente | `business_supervisor.go` + `.agent-business-supervisor` (E-111) | IMPLEMENTADO_NAO_HOMOLOGADO | BAIXO | E-111 | pendências, decisões, política, orçamento e contagem de ciclos sobrevivem ao reinício real; histórico de ciclos em memória |

| Certificação E2E com supervisor no laço | `business_e2e.go` + `finance_ops.go` (E-112) | PARCIAL | BAIXO | E-112 | 21 etapas; estorno aguarda humano, aprovação executa efeito real e não vaza; financeiro reconhece estorno já registrado (a ação fecha o que abriu) |

| Instalação, atualização, rollback e desinstalação | `scripts/install.ps1`, guarda `TestInstallScript` (E-113) | PARCIAL | MÉDIO | E-113 | ciclo executado do código-fonte com manifesto e SHA-256, rollback reversível e desinstalação que preserva dados; sem pacote assinado nem macOS/Linux reais |

| Fábrica de empresa com artefatos | `company_factory.go`, rotas `/company/factory/*` (E-114) | PARCIAL | MÉDIO | E-114 | artefatos reais em disco, preço derivado de custo e margem, CRM não inventado, checklist com bloqueios e procedimento; nada legal emitido |

| Benchmark comparável (próprio produto) | `benchmark_suite.go`, executor `TestBenchmarkSuiteMeasuresHermeticTasks` (E-117) | PARCIAL | MÉDIO | E-117 | 5 tarefas herméticas medidas (5/5, 102 ms) com isolamento e retomada provados; **comparação externa NÃO executada** por falta de credenciais |

| Opt-out automático de consentimento | `consent_optout.go` (E-118) | PARCIAL | MÉDIO | E-118 | revoga por canal+contato com a frase como evidência, idempotente, dupla negação tratada; heurística não cobre todas as formas de dizer "pare" |

| Propostas comerciais e pós-venda | `sales_workforce.go` (E-119) | PARCIAL | MÉDIO | E-119 | preço do catálogo recalculado no servidor, envio exige consentimento, vigência e decisão terminal; pós-venda medido com atraso contável; sem entrega por provedor externo e sem desconto negociado |

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
