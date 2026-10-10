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

**Cobertura de auditoria:** 36 de 38 estágios com diagnóstico (`auditoria != PENDENTE`); restam 21, 38.

## 2. Tabela completa

| # | Estágio | Estado | Auditoria | Capacidade | Evidência | Bloqueio |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Auditoria total e inventario real | IN_PROGRESS | parcial | PARCIAL | E-001, E-002, E-021, E-023, E-036 | — |
| 2 | Estabilizacao e consolidacao da base | IN_PROGRESS | parcial | PARCIAL | E-011, E-013, E-014, E-015, E-016, E-017 | B-09 |
| 3 | Arquitetura canonica dos componentes | IN_PROGRESS | parcial | PARCIAL | E-037, E-040, E-043, E-044, E-045 | — |
| 4 | Model Gateway (roteamento, rotacao, fallback, orcamento) | IN_PROGRESS | **AUDITADO** | IMPLEMENTADO_NAO_HOMOLOGADO | E-024, E-037, E-038 | B-07 |
| 5 | Context Bootstrap / contexto de projeto | BLOCKED_BY_EXTERNAL_DEPENDENCY | parcial | PARCIAL | E-040 | B-07 |
| 6 | Harness Runtime (execucao governada) | IN_PROGRESS | **AUDITADO** | AUSENTE | E-041, E-025 | B-01 |
| 7 | MCP Manager (servidores, ferramentas, escopos) | IN_PROGRESS | parcial | APENAS_CATALOGO | E-018, E-019, E-026 | B-10 |
| 8 | Plugins e Skills | BLOCKED_BY_EXTERNAL_DEPENDENCY | **AUDITADO** | PARCIAL | E-042 | — |
| 9 | Agentes e subagentes (delegacao) | IN_PROGRESS | **AUDITADO** | IMPLEMENTADO_NAO_HOMOLOGADO | E-043 | — |
| 10 | Mission Control multiagente (missoes, eventos, SSE, artefatos) | IN_PROGRESS | parcial | PARCIAL | E-020, E-027 | — |
| 11 | Paridade Codex/Claude (harnesses de codigo) | IN_PROGRESS | parcial | PARCIAL | E-025, E-046 | B-02 |
| 12 | Studio / Builder visual | IN_PROGRESS | **AUDITADO** | IMPLEMENTADO_NAO_HOMOLOGADO | E-008, E-009, E-028, E-034, E-035, E-039 | — |
| 13 | Workflow Engine (fluxos, schedules, condicoes) | IN_PROGRESS | parcial | IMPLEMENTADO_NAO_HOMOLOGADO | E-020, E-029 | — |
| 14 | Browser Operator / computer-use | IN_PROGRESS | parcial | PARCIAL | E-015 | B-09 |
| 15 | Memoria e RAG local | BLOCKED_BY_EXTERNAL_DEPENDENCY | **AUDITADO** | IMPLEMENTADO_NAO_HOMOLOGADO | E-040 | B-07 |
| 16 | Canais de IM (WhatsApp/Telegram/inbox unificado) | IN_PROGRESS | parcial | APENAS_ADAPTER | E-030 | B-03 |
| 17 | Workspace do agente (threads, artefatos, preview) | IN_PROGRESS | parcial | PARCIAL | E-027 | — |
| 18 | Seguranca, permissoes e multitenancy | IN_PROGRESS | parcial | PARCIAL | E-019, E-031 | — |
| 19 | UX/onboarding e jornada de primeira execucao | IN_PROGRESS | **AUDITADO** | IMPLEMENTADO_E_TESTADO | E-011, E-012, E-032 | — |
| 20 | Distribuicao e instaladores (Windows/macOS/Linux) | IN_PROGRESS | parcial | PARCIAL | E-022 | B-06 |
| 21 | Benchmarks e comparativo competitivo | IN_PROGRESS | PENDENTE | PARCIAL | E-021 | — |
| 22 | Gate de engenharia intermediario | IN_PROGRESS | parcial | PARCIAL | E-005, E-006, E-007, E-013, E-014, E-017, E-034, E-036, E-048 | — |
| 23 | Company Factory (empresa orquestrada) | BLOCKED_BY_EXTERNAL_DEPENDENCY | **AUDITADO** | PARCIAL | E-044 | B-07 |
| 24 | CRM e pipeline de vendas | BLOCKED_BY_EXTERNAL_DEPENDENCY | **AUDITADO** | AUSENTE | E-047 | B-05 |
| 25 | Commerce / loja e pedidos | BLOCKED_BY_EXTERNAL_DEPENDENCY | **AUDITADO** | AUSENTE | E-047 | B-05 |
| 26 | Marketplace multi-vendedor | BLOCKED_BY_EXTERNAL_DEPENDENCY | **AUDITADO** | AUSENTE | E-047 | B-05 |
| 27 | Pagamentos e faturamento | BLOCKED_BY_EXTERNAL_DEPENDENCY | **AUDITADO** | AUSENTE | E-047 | B-05 |
| 28 | Atendimento e suporte (helpdesk) | BLOCKED_BY_EXTERNAL_DEPENDENCY | **AUDITADO** | AUSENTE | E-047 | B-03 |
| 29 | Telefonia e call center | BLOCKED_BY_EXTERNAL_DEPENDENCY | **AUDITADO** | AUSENTE | E-047 | B-04 |
| 30 | Marketing e campanhas | BLOCKED_BY_EXTERNAL_DEPENDENCY | parcial | PARCIAL | E-044 | B-05 |
| 31 | Social e publicacao | BLOCKED_BY_EXTERNAL_DEPENDENCY | **AUDITADO** | IMPLEMENTADO_NAO_HOMOLOGADO | E-044 | B-05 |
| 32 | BI / business status e metricas | BLOCKED_BY_EXTERNAL_DEPENDENCY | parcial | PARCIAL | E-044 | B-07 |
| 33 | Canal Telegram | IN_PROGRESS | parcial | APENAS_ADAPTER | E-030, E-045 | B-03 |
| 34 | Canal WhatsApp | IN_PROGRESS | **AUDITADO** | PARCIAL | E-045 | B-03 |
| 35 | Inbox unificado omnicanal | IN_PROGRESS | **AUDITADO** | AUSENTE | E-047 | B-03 |
| 36 | Voz e transcricao | IN_PROGRESS | **AUDITADO** | AUSENTE | E-047 | B-04 |
| 37 | Chamadas e follow-up | IN_PROGRESS | **AUDITADO** | AUSENTE | E-047 | B-04 |
| 38 | Gate final de release verificado | BLOCKED_BY_EXTERNAL_DEPENDENCY | PENDENTE | AUSENTE | E-033, E-048 | B-06 |

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
- **Gap real (fechado depois desta primeira redação):** faltava **cobertura E2E de
  asserção** (spec Playwright) do canvas; os 14 scripts `.mjs` de `e2e/` são
  captura de evidência, não asserção. **Entregue no PR #73**
  (`e2e/studioReorder.spec.ts` + `e2e/studio-builder-fixture.mjs`), verde no CI
  (E-034): ordem inicial vinda do servidor, arraste persistindo via
  `POST .../visual`, paridade pelo botão acessível e ordem sobrevivendo ao
  reinício em aba nova.
- **Próxima ação:** cobrir sob E2E o restante do Studio (preview/export,
  undo/redo, edição de propriedades) antes de promover a capacidade.

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
  #57 e #72 ficaram `PR gate | success` com os gates profundos `skipped` quando
  devidos. Os heads de #71 e #73 têm **23/23 checks verdes** (E-011, E-034); o
  head de #72 (docs-only) tem 15/15 sem falhas (E-036).
- **Limitação:** é gate **intermediário**; não substitui o gate final (38).

## 4. Estágios sem auditoria nesta rodada

3, 4, 5, 8, 9, 15, 20(parcial), 21, 23–32, 34–37. Nenhum deles recebeu
classificação de capacidade canônica; todos estão marcados
`PENDENTE_DE_AUDITORIA`. **Próximo passo obrigatório do Estágio 1:** inspecionar
cada um com comando reproduzível antes de qualquer promoção.

## 5. Rodada 2 — lacunas de auditoria fechadas (2026-10-10)

Probes desta rodada (E-037 a E-047), todos reproduzíveis a partir do
repositório, fecharam a maior parte das superfícies que estavam
`PENDENTE_DE_AUDITORIA`. Achados que mudam decisão:

1. **Harness Runtime (estágio 6) é `AUSENTE` em `main`.** `grep harness` em
   `server/*.go` = 0 e em `internal/agent/*.go` = 1 (arquivo de teste). A
   implementação governada está no PR #62, ainda não mergeado ⇒ depende de B-01.
2. **Model Gateway (estágio 4) é real e é guiado por TIPO de protocolo, não
   por fornecedor.** `internal/multillm` tem 26 arquivos Go (15 de teste) com
   registry, router automático, descoberta, probe, orçamento (`spend.go`),
   credenciais por SO e egress zero-trust. `registry.go:49-51` define três
   tipos — `openai-compatible`, `anthropic`, `cli` — e os provedores vêm de
   config JSON (`base_url`, HTTPS obrigatório fora de loopback, `allow_private`
   explícito, validação de IP privado). Portanto vLLM, LM Studio, llama.cpp e
   Lemonade (servidores OpenAI-compatible) são **configuráveis**; o que falta é
   **preset guiado** para eles e homologação com credencial real (B-07).
   Correção registrada: uma primeira leitura por contagem de palavras sugeria
   "5 provedores ausentes" — era imprecisa.
3. **Memória/contexto (estágios 5 e 15) tem motor real** (`ContextStore` com
   `AddMemory`, `SearchMemories`, `RetrieveRelevant` com score mínimo,
   `OllamaEmbedder`, escopo por organização testado), mas **nenhuma função de
   bootstrap/injeção de contexto** foi localizada — a injeção em si não está
   evidenciada.
4. **Subagentes (estágio 9) têm orquestrador completo** (`swarm.go`: papéis,
   orçamento, plano, execução, cancelamento, escopo por organização,
   `SubagentRunner`) — o que falta é missão multiagente real com artefatos.
5. **Plugins/Skills (estágio 8) são `PARCIAL`**: skills têm manifesto
   versionado, assinatura ed25519 e testes de não-confiança; plugins só têm
   escopo por organização — o gerenciador (instalar/atualizar/rollback) não foi
   evidenciado.
6. **Negócio (estágios 24–29 e 35) não tem motor em `main`**: `marketplace`,
   `inbox` e `telephony` = 0 ocorrências; `crm`/`billing` só aparecem no
   catálogo de conectores. Já **Company OS (23/30/31/32)** e **WhatsApp (34)**
   têm código e testes substanciais (11 e 4 arquivos de teste).
7. **Paridade Codex/Claude (estágio 11) é ponte reversa**, não execução:
   `internal/proxy/claude_desktop*` e `codex_desktop*` expõem modelos locais aos
   apps, e `cli_subscription.go` reconhece modelo de assinatura e custo — não há
   execução do CLI oficial.

8. **Lacuna do estágio 4 parcialmente fechada nesta rodada.** Os *presets guiados* que faltavam foram entregues no PR #74: catálogo canônico em `internal/multillm/presets.go` (4 servidores locais + 8 APIs + genérico), exposto em `GET /api/v1/providers` e consumido pela tela de Provedores, com 14 testes novos e 23/23 checks verdes (E-049). O que continua aberto é a homologação com credencial real, bloqueada por B-07.

9. **Lacuna do estágio 5 parcialmente fechada nesta rodada.** Não existia injeção de contexto em nenhum ponto de chamada de modelo. O PR #75 entregou o **Context Bootstrap** canônico (`internal/agent/context_bootstrap.go`): permissões declaradas e nunca concedidas (escopo desconhecido vira `INDISPONIVEL`), identificadores de tenant/projeto/workspace fora do prompt, memória apenas do projeto com `RedactDLP`, limite de 8 KiB e 8 memórias com truncamento explícito, e injeção como mensagem de sistema no planejador de missão — 12 testes novos e 20/20 checks verdes (E-050). Continua aberto: injetar nos demais pontos de chamada e provar a mesma missão com dois modelos distintos, bloqueado por B-07.

10. **Estágio 5, segunda fatia entregue (PR #76).** Além da montagem e da injeção no planejador (PR #75), a rota `GET /api/agent/v1/missions/:id/context` devolve o documento para qualquer cliente autorizado, herdando o isolamento por organização e mantendo o caráter somente leitura — 3 testes, 20/20 checks verdes (E-051). Continua aberto: injetar nos demais pontos de chamada (chat e agentes) e provar dois modelos distintos (B-07).

11. **Lacuna do estágio 15 parcialmente fechada nesta rodada (PR #77).** O motor de memória já existia, mas sem ciclo de vida. Foram entregues exclusão com rollback, exportação com proveniência (embeddings opt-in) e retenção por idade preservando memória sem data, expostas em três rotas com isolamento por organização — 8 testes novos e 20/20 checks verdes (E-052). Continua aberto: expurgo automático por política de organização e prova de isolamento entre usuários.

12. **Lacuna do estágio 8 parcialmente fechada nesta rodada (PR #78).** Plugins passaram a ter manifesto versionado, escopo recusado quando desconhecido, confiança decidida por assinatura ed25519 de chave autorizada (falha fechada sem raiz configurada), atualização otimista com histórico, rollback, isolamento por organização e persistência que detecta estado adulterado — 10 testes novos e 20/20 checks verdes (E-053). Continua aberto: baixar e verificar o CONTEÚDO do pacote e expor o registro na interface.

13. **Estágio 21 saiu do comparativo apenas documental (PR #79).** Passou a existir medição reproduzível dos motores próprios — busca de memória, ciclo de vida de plugin e montagem de contexto fundamentado — com metodologia publicada em `docs/mission/HADES_BENCHMARK_METHOD.md`, leitura honesta dos números (busca linear; benchmarks de plugin dominados por I/O) e a lista explícita do que falta para comparar com concorrentes. **Nenhuma alegação de superioridade** é feita; o comparativo real continua bloqueado por licença/conta e inferência real (B-05/B-07).

14. **Conteúdo de plugin verificado (PR #80), segunda fatia do estágio 8.** O pacote passa a ser assinado (ed25519 sobre o SHA-256 do arquivo), só entra de diretório permitido pelo operador (falha fechada sem allowlist), é copiado com digest registrado — com rollback da instalação se a cópia falhar — e o digest é recomputado depois para detectar adulteração. Instalar pacote **não** concede escopo: `promote` continua obrigatório. 6 testes novos e 20/20 checks verdes (E-055). Continua aberto: executar código de plugin em processo isolado (sandbox) e expor o registro na UI.

15. **Retenção deixou de ser indefinida (PR #81).** Passou a existir política declarada por organização (idade máxima e teto por projeto), persistida com rollback e recusando política inválida em disco, com aplicação **explícita e idempotente** que varre apenas os projetos da própria organização, preserva memória sem data e devolve contagens reais; sem política, nada é removido. A decisão de não fingir automação contínua está registrada no código e no CHANGELOG. 6 testes novos e 20/20 checks verdes (E-056). Continua aberto: prova de isolamento entre **usuários** e ligação a um agendador real quando existir executor persistente.

16. **Isolamento de memória por USUÁRIO (PR #82).** Duas pessoas da mesma organização e do mesmo projeto deixaram de conseguir ler a memória privada uma da outra: a visibilidade passou a ser `organization` (padrão, sem migração) ou `private`, a autoria vem da sessão (o `actor_id` do corpo é ignorado) e a filtragem acontece na busca, na recuperação e no RAG **antes** das citações. O modo local usa o ator sintético `local` — explicitamente **não** é autenticação e nada aqui é apresentado como se fosse. 7 testes novos e 20/20 checks verdes (E-057). Continua aberto: prova com usuário autenticado de verdade (token) e ligação da retenção a um agendador real.

17. **Prompt injection deixou de ser só recomendação (PR #83).** Documentos, páginas web, e-mails e respostas de MCP que entram no contexto fundamentado agora aparecem cercados e rotulados como DADOS, com padrões clássicos de instrução e tokens de controle neutralizados de forma visível; a cerca não pode ser quebrada pelo próprio conteúdo. O sanitizador **não** redige rótulos comuns de documento em pt-BR, para não apagar conteúdo real — há teste que falha se isso acontecer. Declarado como mitigação determinística, **não** prova de imunidade. 6 testes novos e 20/20 checks verdes (E-058). Continua aberto: prova de não inferência por timing e RLS com banco real.

18. **§11.1 cumprida no que é verificável: registro de comandos com backend conferido (PR #84).** Dos 32 comandos da seção 6, apenas `/goal` era reconhecido antes. Agora há registro de 35 comandos (incluindo `/cancel`, `/approve` e `/deny`), com **disponibilidade decidida pela rota real** — os seis sem backend (`/research`, `/support`, `/telegram`, `/inbox`, `/crm`, `/handoff`) aparecem como indisponíveis com motivo específico e a ajuda diz que não executam nada. Um teste no pacote `server` compara cada comando disponível com o roteador registrado: **rota inventada faz o teste falhar**. 5 testes novos e 20/20 checks verdes (E-059).

19. **Autorização de merge concedida e primeira integração real na `main` (PR #86).** `main` saiu de `889750abe` para `8b3eda830`: #71 e #73 mesclados individualmente e os outros 13 PRs das rodadas 6–13 integrados por merge train, com o conflito de âncora do `CHANGELOG` resolvido por script e `server/agent_routes.go` resolvido por merge de 3 vias **por declaração**. Verificação local completa e CI **23/23 verde** (incluindo o `Web E2E` que estava vermelho na `main` e foi corrigido pelo #71). Conteúdo de cada PR confirmado em `main` por arquivo e por rota. Os PRs individuais foram fechados com comentário apontando para o trem (E-060). Restam 28 PRs abertos: dependabot e as rodadas 1–5 (#56–#70), que têm o mesmo conflito e serão tratados num segundo trem.

Nada foi promovido a `COMPLETED_VERIFIED` nesta rodada. Estado agregado:
0 `COMPLETED_VERIFIED`, 24 `IN_PROGRESS`, 14 `BLOCKED_BY_EXTERNAL_DEPENDENCY`.
