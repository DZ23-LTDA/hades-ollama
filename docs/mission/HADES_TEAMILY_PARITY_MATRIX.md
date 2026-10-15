# HADES × TEAMILY — MATRIZ DE PARIDADE (FASE 1 do complemento 48)

> **Documento derivado.** A **fonte canônica** é `docs/mission/HADES_MASTER_STATE.json`. Este arquivo **não** cria
> segunda fonte de verdade: ele mede, linha por linha, o que existe no código **neste commit**.
>
> - **medido em:** 2026-10-15 (horário de Brasília) · **commit:** `93f12ac19` · **branch:** `fix/checker-relogio` (PR #438)
> - **método:** leitura de código + **varredura por texto** (`grep`) nas três camadas — runtime (`internal/agent`),
>   backend (`server/`) e **UI** (`app/ui/app/src`) — pelos **nomes que o código usa**, *não* por nome de arquivo.
> - **estados:** `NOT_STARTED` · `ADAPTER_ONLY` · `IMPLEMENTED_UNTESTED` · `TESTED` · `E2E_VERIFIED` · `EXTERNALLY_BLOCKED`
> - **regra desta matriz:** `FACT_VERIFIED` = medido por mim neste commit, com arquivo e linha. `A MEDIR` = **não**
>   foi medido nesta passada — *e está dito, em vez de presumido*.

## 0. A REFERÊNCIA — o que dela está medido e o que NÃO está

| Fonte | O que foi observado | Limite |
| --- | --- | --- |
| [teamily.ai](https://teamily.ai/) | Título observável: *"The Human–Agent Platform Where You and Always-on Agents Get Work Done Together in One Unified and Context-Aware Space"* | O conteúdo é **renderizado por JS**: o `fetch` devolveu HTTP 200 e **apenas o título** ⇒ **detalhes de capacidade da referência = `NOT_MEASURED`** |
| [Forbes — "Teamily AI Brings Agent Teams To Human Teams"](https://www.forbes.com/sites/charliefink/2026/03/06/teamily-ai-brings-agent-teams-to-human-teams/) | Título (posicionamento: **times de agentes dentro de times humanos**) | Matéria **não lida** ⇒ sem coluna de detalhe |
| Listagem em loja ([App Store](https://apps.apple.com/cn/app/teamily-ai-human-ai-team/id6761445638)) | Existência do produto (Human+AI Team) | Conteúdo **não lido** |

⇒ **A coluna "referência" desta matriz é deliberadamente magra.** Comparar capacidade a capacidade exigiria acesso ao
produto; o que **não** foi medido está marcado como `NOT_MEASURED`, *jamais como paridade*.

## 1. AS 20 CAPACIDADES — MEDIDAS NESTE COMMIT

| # | Capacidade (fase do complemento) | Estado | Evidência medida | Próxima ação |
| --- | --- | --- | --- | --- |
| 1 | **Salas/canais de time** (F2) | `NOT_STARTED` | **0** ocorrências de `room`/`sala` como conceito de chat em `internal/agent/*.go` e `server/*.go`; em `*.tsx` do repo: **0** ocorrências de `Cowork`/`comentari`/`sala` | Definir a sala **reusando** `inbox` (conversa por contato+canal) e `CompanyChannel` (`company.go:35`) — **sem** segundo engine |
| 2 | **Menções `@agente`/`@equipe`/`@todos`** (F2) | `NOT_STARTED` | **0** ocorrências de `mention` (o único `mentions` é escopo do conector X/Twitter, `connector_catalog.go:226`) | Parser de menção + resolução de participante; **execução só com autorização + capacidade + orçamento** (regra do complemento) |
| 3 | **Respostas vinculadas** (F2) | `NOT_STARTED` | `Comment{ID, ProjectID, UserID, Body, CreatedAt}` **não tem** campo de pai — `collaboration.go:16-22` | Campo de encadeamento + teste de persistência |
| 4 | **Presença com origem verificável** (F2) | `ADAPTER_ONLY` | `Presence{ProjectID, UserID, Status, UpdatedAt}` com `Status` **texto livre, sem origem** — `collaboration.go:23-28` (*status forjado é indistinguível de real*) | Origem assinada/atribuída ao processo que publica + teste de forja |
| 5 | **Agentes como participantes da mesma sala** (F2/F6) | `NOT_STARTED` | `Presence` é chaveado por **ProjectID+UserID** — não há participante agente (`collaboration.go:23-28`) | Unificar participante humano/agente na mesma estrutura |
| 6 | **Comentário em ENTREGÁVEL** (F7) | `NOT_STARTED` | `Comment` é **escopo de projeto**, sem `ArtifactID` — `collaboration.go:16-22` | Vincular comentário a artefato/missão |
| 7 | **Chat com contexto unificado** (F1/F5) | `TESTED` | Rota **`/api/agent/v1/chat`** + middleware de contexto: `server/chat_context.go` (+ `chat_context_test.go`, `chat_context_missao_test.go`); `server` **ok 24,706 s** no gate de 2026-10-15 | Manter como **base única** do Team Chat (não criar segundo chat) |
| 8 | **Compactação + histórico preservado** (F4) | `TESTED` (pipeline) / `NOT_MEASURED` (E2E real) | `server/chat_context_compactacao*` e `server/chat_context_budget.go` (orçamento de contexto para quem consome depois) | E2E com **modelo real** e janela longa — *medir, não declarar continuidade* |
| 9 | **Memória viva com níveis** (`FACT_VERIFIED`/`DECISION_APPROVED`/`PROPOSAL`/`HYPOTHESIS`/`UNTRUSTED_CONTENT`) (F4) | `NOT_STARTED` | **0** ocorrências desse vocabulário como tipo de memória no código | Tipificar a origem da memória **no mesmo store** existente |
| 10 | **Roteamento por modalidade na conversa** (F5/F43) | `TESTED` | `step_routing.go` (`RouteModelForStep`, `trilhaDeCandidatos`), `capability_policy.go`, `ModelCapabilityRegistry`; CI **verde** em `fd3afc89e` (`Go agentic and server gates`) | Ampliar para modalidade (imagem/vídeo/voz) com **provider real** ou `EXTERNALLY_BLOCKED` |
| 11 | **Orçamento por ação/menção** (F2/F7) | `TESTED` | Orçamentos por agente (`company_agents.go:14`, imposição em `:100`), swarm (`swarm.go:72`), supervisor (`business_supervisor.go:88`), área (`finance_area_budget.go:24`), empresa (`company.go:83`), campanha (`company_growth.go:21`) + **teto por passo** (`MaxStepCostCents`, E-669, CI verde) | Ligar `Mission.AgentID` → saldo do agente (E-670 proposto) |
| 12 | **Aprovação para efeito externo** (F11) | `TESTED` | `inferenceTool` com `RequiresApproval: true` e escopo `model:infer` (`inference_tool.go`); `DefaultCapabilityPolicy()` + `approvalRequiredScopes` (`capability_policy.go`, `context_bootstrap.go`) | Reusar o **mesmo** mecanismo para publicar/mensagem externa |
| 13 | **Estado do agente visível na UI** (F2/F6) | `NOT_STARTED` | **Nenhum** `.tsx` consome presença/participantes (0 ocorrências de `Cowork`/`mention` na UI) | Superfície de sala **ligada à mesma engine** (F44) |
| 14 | **Companion (mascote) — a11y e estado verdadeiro** (F46) | `TESTED` | `app/ui/app/src/components/CompanionMascote.tsx` + `CompanionMascote.test.tsx` (E-538: 11/11 `vitest`; animação **derivada do estado**, `aria-live`, voz declarada indisponível) | Estender o estado ao trabalho real do time (não inventar atividade) |
| 15 | **Orquestrador social/marketing (F3)** | `ADAPTER_ONLY` | `connector_catalog.go` declara conectores (X/Twitter `:226`, LiveKit `rooms` `:304`) com `Status: "operator_setup_required"` | **CONFIG_REQUIRED**: sem credencial do operador, permanece indisponível e **declarado** |
| 16 | **NÃO criar segundo engine** (regra ①) | `DECISION` | Raiz única de composição de produção: `server/agent_routes.go` (`newDefaultAgentRuntime`, ~341) | Manter: toda superfície nova consome essa raiz |
| 17 | **Canal externo com DLP e allowlist** (F11) | `TESTED` | `im_telegram.go` (allowlist, DLP **antes** do envio, higiene de credencial; 9 testes) | Reusar no Team Chat para egresso |
| 18 | **Canais (email/WhatsApp/voz/telefonia)** (F10) | `CONFIG_REQUIRED` | Catálogo de conectores com `operator_setup_required` | Sem credencial ⇒ `EXTERNALLY_BLOCKED` (**não** simular) |
| 19 | **Testes simulados declarados como simulados** (regra ④) | `DECISION` parcial | Doutrina em uso: as sondas declaram `NAO_MEDIDO` (E-671) e o gate **declara** o que exclui (E-672) | Estender a regra às telas de time |
| 20 | **Handoff/continuidade entre agentes e modelos** (F5) | `A MEDIR` | **Não medido nesta passada** — *a existência de arquivos `*handoff*` não é prova de caminho completo UI→efeito* | Medir os call sites do handoff antes de declarar qualquer estado |

## 2. O QUE JÁ EXISTE E **DEVE SER REUSADO** (não recriar)

- **Chat + contexto**: `/api/agent/v1/chat`, `server/chat_context.go`, `chat_context_budget.go`, compactação.
- **Conversas**: `inbox.go` (conversa por contato+canal) e `CompanyChannel` (`company.go:35`).
- **Colaboração**: `collaboration.go` (comentários + presença, persistidos em `comments.json`/`presence.json`).
- **Egresso seguro**: `im_telegram.go` (allowlist + DLP + credencial).
- **Roteamento/capacidade/custo**: `step_routing.go`, `capability_policy.go`, `ModelCapabilityRegistry`, orçamentos.
- **UI**: `app/ui/app/src/components` (React + `vitest`), com o Companion já acessível.

## 3. REGRAS QUE VINCULAM (do complemento 48)

1. **Nenhum** segundo chat/runtime/memória/permissão/tarefas — reusar o que está em §2.
2. **PR #438 e estabilização da `main` primeiro** (o PR está `blocked` por **revisão humana** — não contornar).
3. Menção **só** dispara execução com **autorização + capacidade + orçamento**.
4. Teste simulado **declara-se** simulado; **UI responsiva não é prova de sincronização**.
5. Não atribuir número de estágio sem conferir a numeração (`stages` = 38 ids `1..38`; complemento = **48 PROPOSTO**).

## 4. LIMITES DESTA PASSADA (declarados)

- A **referência (Teamily)** não foi verificada em detalhe (`NOT_MEASURED`).
- As capacidades **10–12, 14, 17** foram medidas **por leitura + CI verde do commit `fd3afc89e`**, não por E2E em tela.
- **Handoff (20)** ficou `A MEDIR`.
- **Não** foi criada nenhuma funcionalidade nesta passada: esta matriz é **auditoria**, e a missão **não** termina por documentação.
