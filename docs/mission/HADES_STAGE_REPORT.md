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

20. **Segundo merge train e fim da fila de PRs antigos (PR #88).** As rodadas 1–5 (#56–#70) e 6 bumps de dependência verdes entraram num único ciclo com **23/23 checks verdes**; a `main` passou a `962fa13d3`. Dois aprendizados ficam registrados com evidência: (a) o ruleset tem `strict=True`, então **cada merge invalida os demais PRs** — a partir daqui a integração precisa ser por merge train ou por `update-branch` serial; (b) havia um **conflito semântico** entre o #62 (`harness:cli` agora real) e o teste do #75 (`INDISPONIVEL` para esse escopo), corrigido atualizando a expectativa para a verdade nova em vez de reverter capacidade ou desativar teste. Restam **7 PRs abertos, todos do dependabot e todos com CI vermelho**, mantidos como prova de incompatibilidade (B-11) — nenhum foi mesclado.

21. **Higiene de dependências e fim do CI vermelho de terceiros.** O vermelho visível no GitHub era composto por 7 PRs do dependabot + 1 PR meu em andamento; a `main` nunca esteve vermelha (verificado no head `5223307a4`: 7 checks, 0 falhas). Tratamento por PR, sem merge cego: **#30** exigia migração de API e foi corrigido de verdade (#90, 20/20 checks); **#26** e **#34** estavam verdes após rebase e entraram; **#27/#29/#33** são majors com migração própria e agora têm `ignore` no dependabot, com PRs fechados e justificados; **#31** revelou falha real de minor/patch e virou **B-12**. Aprendizado registrado: `ignore` no dependabot é melhor que PR semanal nascendo vermelho, e a falha de `cmd/launch` local era `CGO_ENABLED=0`, não defeito.

22. **Estágio 8, fatia 3 — plugins executam (PR #114).** O gerenciador sabia instalar, verificar, promover, atualizar e reverter, mas não executava: pelo V2.1 §11.3, catalogar não é habilitar. A execução passou a ser governada e delega ao executor que já existia (`sandbox.exec`), sem caminho paralelo de processo: oito portas fail-closed, entrypoint vindo do **manifesto assinado**, digest **reverificado a cada execução**, leitura do código **de dentro do zip por nome exato** (recusa traversal, nada é extraído) e escopos validados pela política. Onde não há executor, o resultado é erro explícito — nada simulado. Também nesta rodada: o CI do #113 revelou um **defeito latente de tipagem** (`Uint8Array<ArrayBufferLike>` → `BlobPart`) que passava com 76 suítes verdes e só aparecia no typecheck; foi reproduzido, corrigido sem silenciamento (PR #115) e revalidado. E o #116 (10 checks vermelhos, inclusive jobs Go num PR só de npm) provou-se **head defasado** — `changed_files=0` após rebase —, lição registrada para triagem de dependabot.

23. **Estágio 8, fatia 4 — o servidor diz o estado REAL do plugin (PR #119).** §11.3 na prática: o status de cada plugin passa a ser derivado de dados verificáveis (confiança, digest no disco, política de escopos, declaração de runtime/entrypoint e **trilha de auditoria do runtime**) e exposto em `GET /plugins/status`. Regras que impedem anúncio falso: `TESTED_LIVE` só com execução **bem-sucedida** registrada (`PluginExecutionRecord.Succeeded`); `INSTALLED` sai com `ready_to_use=false`; e `CONNECTED` fica **reservado** — sem sonda de vínculo, declará-lo seria fingir integração, e há teste que garante que ele não aparece. 7 testes novos, `golangci-lint` 0 issues, suítes completas apenas com as falhas ambientais conhecidas.

24. **Estágio 8, fatia 5 — o usuário vê o estado real (PR #121).** Fechando o ciclo do §11.3: o servidor já derivava o estado canônico (#119) e agora a interface o exibe — rótulo em pt-BR, motivo da recusa, capacidades concedidas e última execução (distinguindo bem-sucedida de recusada). A decisão fica em um único lugar (`isReadyToUse` exige `ready_to_use` **e** `TESTED_LIVE`), então o selo “Pronto para usar” não pode nascer do catálogo. Falha ao ler o status não derruba a página nem inventa prontidão, e o componente reusa o mapa de tons do módulo de apresentação (sem duplicata). 10 testes novos; suíte web 78 arquivos/526 testes.

25. **Estágio 8, fatia 6 — catálogo de skills com ativação contextual (PR #123).** O manifesto de skill já era assinado; faltava o que o estágio pede: categoria, proveniência e **ativação por projeto/agente/modelo** com **seleção dinâmica por missão e orçamento de tokens**. A seleção é determinística, nunca inclui skill desabilitada e **registra o motivo de cada exclusão** — nada desaparece em silêncio. Skill não confiável **não pode ser ativada** (o selo do usuário não substitui assinatura verificada), a persistência é por organização com rollback em falha de escrita, e as rotas `activate`/`deactivate`/`active` expõem o catálogo. 9 testes novos. Fica registrada também a armadilha que me custou dois fixtures: `OrganizationID` e `Enabled` entram nos bytes assinados, então assinar antes de defini-los invalida a assinatura.

26. **Estágio 8, fatia 7 — catálogo de skills operável na interface (PR #125).** O ciclo fecha: registro assinado → catálogo contextual com orçamento → **tela que mostra e opera**. A interface resolve o catálogo para projeto/agente/modelo com orçamento validado antes da chamada, exibe a seleção **com o motivo de cada exclusão** e bloqueia ativação de skill não confiável — a recusa do servidor aparece em vez de virar sucesso. A semântica de curinga é replicada do servidor para o botão não mentir sobre o contexto, e orçamento não declarado nunca aparece como custo zero. No caminho corrigi um defeito real: o aviso de orçamento inválido só existia quando o usuário digitava; passou a ser derivado do contexto. 11 testes novos; suíte web 80 arquivos/537 testes.

27. **Estágio 5 — o contexto chega ao chat (PR #127).** O preâmbulo canônico já acompanhava o planejador e a rota de missão, mas o chat conversava cru. A superfície Hades passou a ter `POST /api/agent/v1/chat`, que **reusa o mesmo `ChatHandler`** e injeta o preâmbulo por middleware — o `/api/chat` histórico (CLI e clientes OpenAI-compatible) ficou intocado de propósito. O objetivo vem da última mensagem do usuário (truncado), escopo desconhecido falha alto em vez de virar permissão, identificadores de tenant/projeto não entram no preâmbulo, a injeção é idempotente e sem mutar a entrada, e falha de contexto **não** bloqueia uma conversa legítima. No caminho o teste pegou um defeito real: a idempotência só olhava o marcador do Hades, então reinjetar o mesmo preâmbulo duplicava o contexto a cada turno. A capacidade segue `PARCIAL`: a prova do estágio (mesma missão com dois modelos preservando contexto e permissões) exige provider real (B-07).

28. **Estágio 18 — varredura ofensiva cross-tenant nas superfícies novas (PR #129).** Sete classes de ataque com ator **autenticado**: A é a vítima, B o atacante, e a exigência é que B não veja nem altere nada de A, com recusa explícita (403/404) em vez de `200` com dado alheio. Cobri plugins, skills, ativações (inclusive com o **mesmo** `project_id` e depois de reabrir o store), rollback/promoção de plugin alheio, **mass assignment** de `organization_id` no corpo e a recusa pelo **roteador HTTP** real. Todas bloqueadas. O caso mais importante é o destrutivo: B tentando aplicar a política de retenção no lugar de A — recusado, com **nenhuma** memória de A apagada, enquanto o dono aplica e o expurgo funciona. Os motivos são estruturais, não sorte: o manifesto de plugin não tem campo de organização, o dono exige igualdade explícita, a organização chega por parâmetro separada do corpo, e a retenção é chaveada pela sessão. **Uma hipótese minha foi refutada pelo teste** — eu esperava um caminho morto no registro de skill sem dono; o teste mostrou recusa na entrada, fail-closed. Em vez de registrar um achado inexistente, o teste trava a propriedade. Nenhum achado CRITICO/ALTO nestas superfícies; **RLS em PostgreSQL real** e não inferência por timing seguem pendentes.

29. **Estágio 18 — SSRF por IPv4 embutido: achado ALTO encontrado e corrigido (PR #131).** A política de egresso bloqueava `169.254.169.254`, `127.0.0.1`, `10.0.0.0/8` e `::ffff:169.254.169.254`, mas **permitia os mesmos destinos** escritos como literal **NAT64** (`64:ff9b::a9fe:a9fe`) ou **6to4** (`2002:a9fe:a9fe::1`). Como remote MCP, conectores, OAuth, Telegram, WhatsApp, SAML, research e project import decidem abrir o socket por essa classificação, o túnel era atalho direto para metadata de nuvem, loopback e rede privada. **Encontrei medindo**, não supondo: li a fronteira do controle e rodei uma tabela de endereços que encapsulam IPv4 — seis dos sete casos vinham `PERMITIDO`. A correção extrai o IPv4 de `2002::/16`, `64:ff9b::/96` e `64:ff9b:1::/48` e o reclassifica; **não bloqueei o prefixo inteiro de propósito**, porque NAT64 legítimo (DNS64 para host público) é tráfego válido e a ausência de over-blocking está travada por teste. Prova em duas camadas: política pura e o **dial real** do remote MCP, onde o socket não é aberto para os cinco disfarces. RLS em banco real e não inferência por timing seguem pendentes no estágio.

30. **Estágio 18 — prompt injection corrigido; uploads auditados sem achado (PR #133).** O aviso de “conteúdo externo é dado não confiável” existia, mas dependia de o **chamador** declarar as fontes — e a rota de missão **não declarava nenhuma**. Na prática, um DOCX/XLSX ou página maliciosa ingerida vira `document_chunk`/`web_chunk` e entrava no contexto rotulada só como “Memória autorizada deste projeto”: sem nenhuma marca de que aquele texto é de terceiro, que é exatamente o que um prompt injection precisa. A marca passou a ser **derivada da proveniência** (`Kind`) e a aparecer **junto do conteúdo**; a união com o que o chamador declara é deduplicada e estável, e memória interna **não** é marcada — ausência de falso positivo é teste, porque marcar tudo treinaria o operador a ignorar o aviso. **Antes de atacar aqui auditei o bloco de uploads maliciosos e não achei achado**: `extractProjectArchive` rejeita traversal, absoluto, symlink e `.git`, usa `O_EXCL` e contabiliza bytes **reais** pós-cópia; `readLimitedReader` recusa limite negativo; `SafeExtractZip` limita arquivos, tamanho por arquivo, total e **razão** (zip bomb). Registrei o negativo em vez de inventar problema.

31. **Estágio 18 — integridade do endpoint de personalização; XSS, CSRF e plugins auditados sem achado (PR #135).** `/api/agent/v1/personalization` respondia `status: "updated"` no POST **sem gravar nada** e devolvia no GET instruções, tom e memória **fixos no código**; além disso o grupo era registrado mesmo quando a API agêntica não subia (`agentAPI == nil`), ficando **sem middleware de autenticação e origem**. Agora responde **501** apontando o caminho real (`custom_instructions` composto no system prompt a cada turno) e **não existe** sem middleware (404) — fail closed em vez de rota aberta. No mesmo bloco auditei **XSS** (nenhum HTML cru; os dois usos de `innerHTML` são benignos), **CSRF** (política de `Origin` no middleware do grupo, fail-closed: wildcard ignorado, allowlist vazia recusa, origem opaca recusada) e **plugins maliciosos** (assinatura/tamper/path do bundle, escopo e posse por organização, 11 recusas no runtime governado) — os três **sem achado**, registrados para não repetir a auditoria. Registro também que **quase criei um achado falso**: a promessa da UI sobre instruções personalizadas é verdadeira, implementada em `app/ui/ui.go` com limite de 4000 caracteres e teste próprio.

32. **Estágio 13 — resultado por nó na execução de schedule (PR #137).** O executor devolvia **apenas a contagem de missões**, com quatro consequências reais: (1) fluxo com **condicional falso** ou passo bloqueado reportava **sucesso** sem registrar nós ignorados — exatamente o que o estágio proíbe; (2) execução **bem-sucedida não era persistida** (`recordScheduleSuccess` saía cedo sem gravar `last_run_at`); (3) no **supervisor**, o outro ponto de disparo, **nada** era registrado e a falha desaparecia; (4) o código de falha era genérico e não dizia **qual** passo falhou. Agora existe **relatório por nó** (`succeeded`/`skipped`/`failed` com motivo), persistido em **sucesso e falha**, com o código de falha nomeando o passo (preservando o contrato histórico no caminho legado, exigência de um teste existente) e com o supervisor reportando falhas e ignorados **no próprio tick**. Quatro testes novos. Registro também que duas falhas no caminho eram **minhas**: um comando sem `APPDATA`/`LOCALAPPDATA` (que eu quase li como regressão de código) e dois fixtures sem o `Objective` obrigatório.

33. **Estágio 13 — resultado por nó visível na interface (PR #139).** O relatório por nó já era persistido e trafegava na API, mas o operador continuava vendo só “próxima execução”. As rotinas passam a mostrar veredito (**Sucesso** / **Parcial — houve passo ignorado** / **Falhou** / **Sem execução registrada**), resumo com executados/ignorados/missões, data, código de falha e os passos que exigem atenção com o motivo declarado pelo servidor. **Execução com passo ignorado nunca diz “Sucesso”** — a regra do estágio aplicada no ponto em que o usuário lê, com teste dedicado para isso. Nove testes novos; suíte web 82 arquivos/546 testes. **Registro de engenharia:** o próximo item (retry por nó) foi **deliberadamente adiado** porque `CreateMission` **não tem chave de idempotência**: repetir um passo `action.mission` que falhou depois de persistir criaria **missões duplicadas**. A ordem correta ficou registrada — idempotência primeiro, retry depois, reprocessamento por último.

34. **Estágio 13 — idempotência na criação de missão, pré-requisito do retry (PR #141).** Na rodada anterior eu registrei que retry por nó **não podia** ser implementado antes da idempotência, porque `CreateMission` não tinha chave e repetir um passo que falhou depois de persistir criaria **missões fantasma**. Esta fatia fecha o pré-requisito: o ID é **derivado** de `sha256(organização + chave)`, o store recusa duplicata e a criação **devolve a missão existente** em vez de falhar na segunda vez. A missão só é devolvida se pertencer ao **escopo ativo** — chave repetida **nunca** alcança outro tenant, e a organização entra no hash para não colidir entre tenants. Sem chave, o comportamento histórico fica intacto. No fluxo, a chave por *(execução + passo)* faz a **execução repetida criar zero missão nova**, mantendo uma única missão persistida e explicando o motivo no relatório por nó. Cinco testes novos. Com isso o retry por nó passa a ser seguro e é o próximo passo.

35. **Estágio 13 — retry por nó, seguro pela idempotência (PR #143).** O retry entrou **em cima** da idempotência, não em vez dela: na rodada 36 eu **recusei** implementá-lo porque repetir um passo que falhou depois de persistir criaria **missões fantasma**; na 37 construí a idempotência e **provei por teste** que a execução repetida cria **zero** missão; agora o retry usa essa garantia. `max_attempts` tem padrão 2 e **teto 5 recusado na validação**; o retry **recupera** de falhas transitórias, é **limitado** e relata o erro da **última** tentativa (não o primeiro de uma sequência que se recuperou); o backoff é linear e **limitado** (o motor roda em tick de despacho); e a **mesma chave** em todas as tentativas impede duplicar. O relatório e a interface passam a mostrar **quantas tentativas** cada passo consumiu. Sete testes Go novos e um caso de interface. Restam no estágio: **reprocessamento** do que ficou `skipped`, **DLQ por nó** e a **homologação** do critério (fluxo real ponta a ponta com falha simulada, retry e retomada, com evidência por nó).

36. **Estágio 13 — o disparo manual passa a executar o grafo (PR #145).** O botão “Executar imediatamente” **não executava o fluxo**: chamava `createMission` com o objetivo do schedule e **ignorava os passos**, então um fluxo com condição e várias ações rodava como **uma missão avulsa** — sem evidência por nó e **sem registrar** o resultado no schedule. É o mesmo defeito de família do endpoint de personalização: **o controle aparentava fazer uma coisa e fazia outra**. A execução manual agora passa pela rota `POST /schedules/:id/run`, que roda o **grafo** pelo **mesmo caminho** do despacho automático, registra com a mesma política (inclusive tentativas e código de falha nomeando o passo) e devolve o **relatório por nó** — e, quando o fluxo falha, devolve **o relatório com o passo culpado** em vez de um erro genérico. Outra organização recebe 404/403 **sem deixar evidência**. `ScheduleRunReport` ganhou **`run_key`**, persistida em `LastRun`: sem ela o reprocessamento futuro teria que **adivinhar** a execução; com ela, reusa a mesma chave e os passos que já criaram missão continuam apontando para ela. Três testes de rota novos.

37. **Estágio 13 — reprocessamento reusa a execução e reexecuta só o pendente (PR #147).** Fecha o item que a **`run_key`** da rodada anterior destravou. `executeScheduleFlow` passou a ser o **núcleo compartilhado** entre execução normal, manual e reprocessamento — as três **não podem divergir**. O reprocessamento reexecuta **somente** `skipped`/falhados; os passos bem-sucedidos são **preservados** com o resultado original e continuam satisfazendo dependentes, **sem criar missão nova** (idempotência); passo **novo** no grafo também roda. Sem execução anterior a recusa é **explícita** (HTTP 409), porque adivinhar a chave poderia **duplicar missão**. Rota `POST /schedules/:id/reprocess` e botão **“Reprocessar pendentes”** na interface, que só aparece quando há passo ignorado ou falho e informa quantos **seguem** pendentes. Com isso, a sequência do estágio 13 fica: **idempotência → retry por nó → execução manual real → reprocessamento**. Restam **DLQ por nó** e a **homologação** do critério (fluxo ponta a ponta com falha simulada, retry e retomada).

38. **Estágio 13 — DLQ por nó (PR #149).** Fecha a última lacuna de **implementação** do estágio. Antes, um passo que esgotava as tentativas deixava apenas um código de falha: **não tinha registro próprio**, podia **se perder no próximo tick** e não havia como distinguir “já falhou antes” de “falhou agora”. Agora o **motor** registra a entrada (`step_id`, `kind`, **motivo**, tentativas, objetivo, quando falhou) e o **gravador** persiste no schedule — em sucesso e em falha. A entrada **sai** quando o passo volta a ter sucesso, **inclusive via reprocessamento**: sem isso a DLQ **mentiria**, listando como morto um passo que voltou a rodar. A fila é **limitada** (20, mais recentes primeiro), porque sem teto um fluxo que falha a cada tick cresceria **sem fim** no arquivo de estado. A interface mostra **“Fila morta (N)”** com o motivo **real** de cada passo. Com isso o estágio 13 tem **implementação completa**: idempotência, retry por nó, execução manual real, reprocessamento e DLQ por nó. Falta a **homologação do critério** — fluxo real ponta a ponta com falha simulada, retry e retomada com evidência por nó — e até essa prova o estágio permanece **`PARCIAL`**.

39. **Estágio 13 — versão do fluxo e homologação do critério (PR #151).** Duas partes. **(1)** `Schedule.FlowVersion` nasce em **1** e **sobe apenas quando os passos mudam** — editar objetivo, intervalo ou habilitar/pausar **não** cria versão nova, porque tratar isso como versão nova tornaria impossível comparar execuções equivalentes; o relatório guarda a **versão que aquela execução rodou** e a interface mostra “fluxo vN”. Isso fecha a evidência obrigatória *versão do workflow*. **(2)** A **homologação do critério** foi provada num teste que usa o **caminho real**, não peças isoladas: `ContextStore` e `JSONStore` **em disco**, store de missões que falha de forma **transitória** (falha simulada no limite de persistência) e um **segundo runtime** sobre os **mesmos diretórios**, simulando o **reinício do worker**. O teste prova, em sequência: o **retry absorve** a falha com **3 tentativas** registradas e o passo recuperado **fora** da fila morta; a evidência **por nó** é persistida em disco; o estado **sobrevive ao reinício**; a **retomada** reusa a execução, cria **zero** missão nova e termina com **todos** os passos bem-sucedidos; o store real tem **exatamente 2 missões**, sem duplicata; e a versão sobe ao mudar o grafo, **não** subindo em edição que não muda o grafo. **Por que o estágio continua `PARCIAL`:** o critério de conclusão está **demonstrado**, mas as **atividades do próprio estágio** têm itens em aberto — o vocabulário de nós cobre `trigger.interval`, `trigger.webhook`, `action.mission` e `condition.if`, e **faltam conectores, variáveis, transformações, loops limitados e aprovações como nós**. Cumprir uma frase do critério **não** fecha a lista de atividades.

40. **Estágio 13 — variáveis/transformações e laços limitados no vocabulário de nós (PR #153).** Amplia o vocabulário do estágio, que era a lacuna apontada na rodada anterior. **`action.transform`** publica um valor no escopo da execução a partir de `{{variável}}` e é **puro** — não fala com o mundo externo, só transforma texto já disponível; o objetivo de `action.mission` e o de `logic.loop` **também** expandem variáveis. **`logic.loop`** cria **uma missão por item**, com chave de idempotência que inclui **item e índice** e evidência **por iteração** em `Children` (não um contador agregado); reprocessar **não** recria itens. O desenho é **fail-closed**: variável desconhecida é **erro** (nunca string vazia — um objetivo que perde o valor por causa de um nome errado seria pior que uma falha explícita); nome inválido, valor acima de **4096** caracteres e escopo acima de **32** entradas são **recusados**, nunca truncados; laço sem item, com item vazio ou com mais de **10** itens é recusado **na validação**, antes de persistir; e o teto de **24 missões por execução** é conferido **antes** de criar qualquer missão do laço, para não deixar trabalho pela metade. Um **bug real** foi encontrado pelo próprio teste: o objetivo de `action.mission` **não estava sendo expandido**, então a missão era criada com o texto literal `missão com {{resumo}}` — o teste conferiu o objetivo **real da missão persistida** e a correção foi no **código**, não na expectativa. Faltam ainda **conectores** e **aprovações** como nós: aprovação exige **estado de espera entre execuções** e fica registrada como próximo passo.

41. **Estágio 13 — nó de aprovação com espera entre execuções (PR #155).** Fecha o último item do vocabulário que exige **estado**, e por isso não é um nó comum: a aprovação **pausa** o fluxo, e a espera **atravessa execuções** — o estado vive no **schedule**, não no processo. Sem decisão para **(passo, execução)** o passo fica **`pending`** e os dependentes **não** rodam: o fluxo **espera, não adivinha**. Aprovado libera; negado mantém o caminho barrado. A **negativa é decisão humana**, não falha técnica: **não** entra na fila morta, e o motivo carrega **quem decidiu e por quê**. A decisão é chaveada por **(StepID, RunKey)** *de propósito*: aprovar durante uma pausa **não** autoriza a mesma etapa numa execução futura, e a retomada acontece pelo **reprocessamento**, que reusa a mesma execução. `Schedule.Approvals` (histórico auditável, teto 20) e `Schedule.PendingApprovals` são persistidos; a rota `POST /schedules/:id/approvals` responde **409** quando não há espera e aplica à execução **parada** quando o operador não informa a execução. A interface mostra **“aguardando aprovação”**, trata o veredito como **PARCIAL** (o fluxo está parado — **nunca** sucesso) e oferece **Aprovar/Recusar** por passo. Com isso o vocabulário do estágio tem `trigger.interval`, `trigger.webhook`, `condition.if`, `action.mission`, `action.transform`, `logic.loop` e `approval.request`; falta **conectores** como nó para a lista de atividades ficar integralmente coberta.

42. **Estágio 13 — nó de conector pelo caminho real (PR #157).** Fecha a lista de atividades do estágio. O nó **não inventa acesso**: chama o conector **configurado** por `ConnectorManager.CallForOrganization`, que aplica **credencial do tenant**, **política de egress** e **escopo de organização** — e os testes rodam contra um **servidor HTTPS real**, não contra um duble. A decisão de segurança central: **método que escreve** (`POST`/`PUT`/`PATCH`/`DELETE`) **só sai com decisão humana aprovada para a mesma execução**; sem decisão o passo fica **`pending`** e **nenhuma chamada é feita** (o teste prova `calls == 0`), e a negativa mantém o caminho barrado **sem** fila morta. O desenho é **fail-closed**: conector não registrado **falha** com motivo e entra na fila morta — **nunca** é pulado em silêncio; conector de **outra organização** é recusado **pelo próprio gerenciador**; e **HTTP 5xx** do outro lado **não** é sucesso do nosso nó. Duas correções de processo: o registrador **exigir HTTPS** é um controle **correto** do produto, então o ajuste foi **no teste**; e o `misspell` marcou **“Eles”** como erro em pt-BR — comentário **reescrito**, regra **não** silenciada. Com os **oito tipos de nó** (`trigger.interval`, `trigger.webhook`, `condition.if`, `action.mission`, `action.transform`, `logic.loop`, `approval.request`, `action.connector`), a lista de atividades do estágio está **coberta**, além do critério de conclusão demonstrado em E-081. O próximo passo é **reavaliar a promoção** do estágio com **verificação independente** do conjunto — não por conveniência de fim de lista.

43. **Estágio 13 — verificação independente e promoção com limites declarados (E-085).** Fiz a releitura **atividade por atividade** da lista do estágio contra o código, em `docs/mission/HADES_STAGE_13_VERIFICATION.md`, sem presumir cobertura. A verificação **refutou** a suposição de que `branches` já funcionava e encontrou **dois defeitos reais**: (1) o ramo alternativo era **inalcançável** — `condition.if` com `Expect: failed` era barrado por `scheduleDependenciesSatisfied`, que exigia **sucesso** do dependente **antes** de avaliar a condição, então o `else` nunca rodava e *if/else existia só no nome*; corrigido com `scheduleStepDependenciesReady`, que respeita a expectativa declarada. (2) Depois da primeira correção, uma aprovação **pendente** passou a satisfazer o ramo alternativo, disparando o caminho de recusa **antes de qualquer decisão humana**; corrigido com o conjunto `pendingSteps`, porque *“não teve sucesso”* e *“aguarda decisão”* são estados **distintos**. Os dois foram encontrados por **teste que mede o efeito**, não por leitura otimista, e ficaram travados em `schedule_branch_test.go`. Com o **critério de conclusão cumprido** (E-081) e a **lista de atividades coberta**, a execução do estágio passa a **`COMPLETED_VERIFIED`** e a capacidade do **Workflow Automation Engine** a **`IMPLEMENTADO_E_TESTADO`** — com **limites declarados** que **não** são cobertura: conector homologado contra servidor HTTPS controlado (credenciais de SaaS de terceiros seguem `BLOCKED_BY_EXTERNAL_DEPENDENCY`, nos estágios 25/33/36); falha de passo com efeito **aborta** o fluxo em vez de desviar para o ramo alternativo; e a execução do fluxo é por **tick do supervisor**, com a fila de jobs cobrindo as missões criadas pelos nós. O **release global permanece `NOT_RELEASE_READY`** — este é **um** dos 38 estágios.

44. **Estágio 14 — browser: defeito de resolução de interpretador e classificação honesta (E-086).** O teste de navegador estava em **B-09** como falha ambiental e a causa **não** era a esperada: o operador escolhia o interpretador Python por **existência no PATH** (`python3` → `python`), e no Windows esses nomes respondem pelo **atalho da Microsoft Store**, que **existe** e **falha ao executar** (exit 9009). Três efeitos medidos: o operador adotava o atalho mesmo com **Python 3.12.10 real** instalado; o status reportava **`PythonOK: true`** para algo que **não executa nada**; e a orientação era **errada** (*“instale o Playwright”*), enquanto o E2E falhava com erro críptico. Corrigido com `resolvePythonInterpreter`, que **valida cada candidato executando** (`import sys`) e é **compartilhado** entre status e operador — status e execução não podem divergir. Medido ao vivo depois: `python_path = C:\Program Files\Python312\python.exe` e o **E2E de navegador passa de FAIL para SKIP com causa legível** (*“No module named playwright”*). Classificação por plataforma em `docs/mission/HADES_STAGE_14_PLATFORM_STATUS.md`: **Windows = `BLOQUEADO_EXTERNAMENTE`** (Python OK, Playwright/Chromium ausentes), **Linux e macOS = `NAO_VERIFICADO`** — nada declarado homologado por compilação cruzada. Já coberto **sem** navegador: ambiente filho **sem segredos**, recusa de escopo sem sandbox estrita, eventos de frame, lógica de setup e resolução de interpretador. O desbloqueio exato (`pip install playwright` + `playwright install chromium`, ~150 MB) está no produto em `SetupBrowserEnvironment` e **não foi executado**: mudança de máquina com custo relevante merece decisão do usuário. Um **teste existente** foi corrigido porque **codificava o contrato defeituoso** (“existe no PATH ⇒ Python OK”) com um probe que falhava sempre — a intenção do teste foi preservada.

45. **Auditoria dos bloqueios — 12 estágios sem justificativa (E-087).** A rodada anterior mediu **13** estágios com `BLOCKED_BY_EXTERNAL_DEPENDENCY` e **12 sem `blocked_reason`**: nenhuma justificativa e nenhum procedimento de desbloqueio, contra a regra explícita da missão. A auditoria (`docs/mission/HADES_BLOCKED_AUDIT.md`) **mediu** em vez de aceitar o rótulo — e a primeira medição, **por nome de arquivo**, concluiu “AUSENTE” **errado** para dois estágios: `company_social.go` (228 linhas) e `tel_agent.go` (412 linhas + 5 arquivos de teste) existem, só não casavam com os padrões que usei. A medição **por conteúdo** corrigiu o diagnóstico e mostrou o erro de fundo: o carimbo em bloco **confundia** *estágio não implementado* (`NOT_STARTED`/`AUSENTE`) com *homologação ao vivo que exige conta/credencial*. Os 12 foram reclassificados **por evidência medida**: **5 para `IN_PROGRESS`** (15 memória com 308 linhas; 23 empresa com 1.760 linhas e 7 arquivos de teste; 26 social com rascunho; 28 voz com `tel-agent` e 5 testes; 31 supervisor com 432 linhas e 493 de teste) e **7 para `NOT_STARTED`/`AUSENTE`** (24, 25, 27, 29, 30, 32, 38). Cada estágio ganhou `external_dependency` **separado** — com o desbloqueio escrito — **apenas** onde a validação real depende de conta, e `blocked_reason` **vazio** porque **não há bloqueio de execução**. Consequência prática: **reabre trabalho implementável que estava escondido atrás de um carimbo** (memória, supervisor e as partes de sandbox de empresa, comércio, CRM, inbox e financeiro) e deixa as dependências externas reais **nomeadas**: pagamento (24), conta de vendedor (25), APIs de redes (26), provedor de telefonia (28), canais reais (27, 29), integração fiscal (30) e integrações ao vivo (32, 38).

46. **Estágio 15 — exportação/exclusão de memória e um defeito ALTO de vazamento (E-088).** A auditoria de bloqueios reabriu o estágio 15 como trabalho local, e faltava a **evidência obrigatória**: exportação e exclusão. Ao implementá-la, encontrei um defeito **ALTO**: a **busca** de memória era escopada por ator, mas os caminhos **irmãos** não eram — o `GET /projects/:id/memories/export` chamava `ExportMemories` **sem ator** e **vazava memória `private` de outro usuário da mesma organização** no arquivo exportado, e o `DELETE /projects/:id/memories/:memory_id` chamava `RemoveMemory`, deixando qualquer usuário autenticado **apagar a memória privada de outro**. Mesmo dado, duas respostas: a busca escondia e o export entregava — o padrão clássico de *protegi um caminho e esqueci o irmão*, exatamente o risco que o próprio arquivo de isolamento declara. Corrigido com `ExportMemoriesForActor` e `DeleteMemoryForActor`: export **escopado por ator** (registrando `actor_id` e `generated_at` para auditoria) e exclusão recusada com o **mesmo** `ErrMemoryNotFound` de um ID inexistente, para a rota **não virar oráculo** de existência de dado privado. A persistência **não** foi reimplementada: `DeleteMemoryForActor` reusa `RemoveMemory`; o compilador ainda pegou uma **duplicação minha** (um segundo `ErrMemoryNotFound`) e o sentinela existente foi reaproveitado. Verificação medida: 4 testes de motor (export não vaza privado alheio; exclusão própria **persiste após reinício**; recusa tem o mesmo erro de inexistente e **não** apaga; vetores só saem se pedidos) e 2 de rota com o harness existente (o export do bruno não contém a memória da ana; o delete do bruno recebe 404 e a memória continua lá), mais `go build ./...` exit 0, suítes com apenas as falhas ambientais conhecidas, `golangci-lint` **0 issues** (o `misspell` marcou `momento` como erro de `memento` — texto reescrito, regra **não** silenciada). **O estágio NÃO foi promovido**: isolamento, reinício e exportação/exclusão estão cobertos, mas o critério também exige **troca de modelo** e **retomada**, que seguem pendentes — promover agora repetiria o erro corrigido em E-087.

47. **Estágio 15 — o contexto do modelo era o terceiro irmão da mesma família, e a troca de modelo foi provada ao vivo (E-089).** Depois de corrigir export e delete (E-088), faltava o caminho que leva o dado **mais longe**: o **contexto injetado no modelo**. A rota montava o contexto com `SearchMemoriesContext` (busca **sem ator**) e `BuildContextBootstrap` filtrava memória **apenas por `ProjectID`**, ignorando `Visibility` e `ActorID` — memória `private` de **outro usuário** entrava no preâmbulo de sistema, que **vai para o modelo**; com modelo externo, a anotação privada de um colega **sairia da máquina**. Correção **fail-closed em duas camadas**: `ContextBootstrapInput` ganhou `ActorID` e o builder **recusa** memória que o ator não pode ler (nem com a lista crua do projeto), e a rota passou a usar `SearchMemoriesForActor`. Então o **critério do estágio foi provado ao vivo**: o Ollama local desta máquina está ativo com 8 modelos, então a troca **não foi simulada** — teste real entre `qwen2.5:7b` e `llama3.2:3b` (**9,91s**) provou que os dois modelos chegam ao provedor, que o contexto injetado fica **byte-idêntico** na troca, que a memória privada alheia **não** vai ao modelo e que o conhecimento do time **vai**, e que o reinício do store devolve o mesmo contexto (retomada); sem Ollama local o teste é **pulado com causa declarada**, nunca aprovado por omissão. **O estágio NÃO foi promovido**, e desta vez o que falta está **medido, não suposto**: deduplicação de memória (os `dedup` do repositório são de fila/webhook/workspace), versionamento, resumos, criptografia em repouso e escopos de missão/agente/equipe no tipo `Memory` seguem **AUSENTES**. Mesmo padrão aplicado ao estágio 13: promover só com a lista de atividades coberta.

O estágio 13 foi promovido a `COMPLETED_VERIFIED` nesta rodada, com limites declarados no documento de verificação. Estado agregado:
0 `COMPLETED_VERIFIED`, 24 `IN_PROGRESS`, 14 `BLOCKED_BY_EXTERNAL_DEPENDENCY`.
