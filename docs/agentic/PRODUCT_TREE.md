# Árvore de produto — Manus Desktop observável e Ollama Full

## 1. Como interpretar esta árvore

O objetivo do Ollama Full é oferecer as mesmas **jornadas observáveis** de um assistente desktop agentic completo e combinar essas jornadas com padrões úteis de Claude Code, Codex, OmniRoute e outros harnesses. A meta não é copiar código, interface proprietária ou internals de qualquer produto. A implementação deve permanecer compatível com as licenças usadas, manter as atribuições upstream e provar cada capacidade por testes e execução real.

A imagem fornecida mostra uma superfície desktop do Manus com navegação lateral, criação de tarefa, agente, habilidades, plugins, agendamento, biblioteca, projetos, tarefas, conta, créditos e configurações. A árvore abaixo descreve essa superfície observável e amplia o produto somente com recursos que fazem parte do escopo declarado para o Ollama Full ou que podem ser implementados por contratos públicos.

Os marcadores têm este significado:

- **[ATUAL]** — existe implementação ou fluxo verificável no repositório.
- **[ADAPTER]** — o contrato e o adapter existem, mas a execução depende de serviço, credencial, hardware ou ambiente externo.
- **[PARCIAL]** — existe uma parte funcional, mas a jornada completa ainda não foi fechada.
- **[ALVO]** — estrutura planejada para completar a jornada.
- **[DEPENDÊNCIA]** — não pode ser declarado concluído sem uma validação externa específica.

## 2. Árvore observável do Manus Desktop

Esta é uma árvore funcional baseada na superfície visual fornecida e nas jornadas de produto que ela representa. Ela não afirma conhecimento de componentes internos do Manus.

```text
Manus Desktop
├── Workspace e identidade
│   ├── Seletor de workspace/projeto
│   ├── Conta e perfil do operador
│   ├── Plano, créditos, uso e limites
│   ├── Personalização e preferências
│   ├── Ajuda e documentação
│   └── Sair
├── Navegação global
│   ├── Pesquisa
│   ├── Alternância de layout/painel
│   ├── Nova tarefa
│   ├── Agente
│   ├── Habilidades
│   ├── Plugins
│   ├── Agendado
│   ├── Biblioteca
│   ├── Projetos
│   ├── Tarefas
│   └── Empresa / Company OS
├── Nova tarefa
│   ├── Composer de linguagem natural
│   │   ├── Texto e instruções
│   │   ├── Arquivos e imagens
│   │   ├── Referências e fontes
│   │   ├── Voz e transcrição
│   │   ├── Seleção de modelo/agente
│   │   └── Envio, cancelamento e retry
│   ├── Ações rápidas
│   │   ├── Criar slides
│   │   ├── Criar site
│   │   ├── Design
│   │   ├── Criar jogo
│   │   └── Mais ferramentas
│   ├── Recomendações
│   │   ├── Edição de imagens
│   │   ├── Análise visual
│   │   └── Pesquisa de mercado/ecossistema
│   └── Rascunhos e histórico de prompts
├── Agente
│   ├── Missões ativas
│   ├── Missões em background
│   ├── Plano e subtarefas
│   ├── Timeline de eventos
│   ├── Browser/computer use
│   ├── Terminal e código
│   ├── Pesquisa e citações
│   ├── Aprovações humanas
│   ├── Correção e recuperação
│   ├── Artefatos gerados
│   └── Compartilhamento e colaboração
├── Habilidades
│   ├── Catálogo
│   ├── Habilidades instaladas
│   ├── Habilidades por projeto
│   ├── Manifesto, versão e permissões
│   ├── Teste, habilitação e desabilitação
│   └── Criar habilidade customizada
├── Plugins e integrações
│   ├── Catálogo de plugins
│   ├── Conectar/desconectar
│   ├── OAuth e renovação
│   ├── Scopes e consentimento
│   ├── Secrets server-side
│   ├── GitHub
│   ├── Google Workspace
│   ├── E-mail
│   ├── Slack e Discord
│   ├── WhatsApp
│   ├── Jira, Notion e Microsoft 365
│   ├── MCP/A2A
│   └── Desktop Commander local/Remote MCP
├── Agendado
│   ├── Tarefa única
│   ├── Recorrência
│   ├── Timezone
│   ├── Webhook/evento
│   ├── Retry e DLQ
│   ├── Pausar/retomar
│   └── Histórico de execuções
├── Biblioteca
│   ├── Documentos
│   ├── Slides
│   ├── Planilhas
│   ├── Imagens, áudio e vídeo
│   ├── Sites, apps e jogos
│   ├── Dashboards e gráficos
│   ├── Busca e filtros
│   ├── Preview
│   ├── Versionamento e diff
│   ├── Download/exportação
│   └── Compartilhar e permissões
├── Projetos
│   ├── Novo projeto
│   ├── Contexto persistente
│   ├── Arquivos e fontes
│   ├── Memória e instruções
│   ├── Membros e papéis
│   ├── Tarefas do projeto
│   ├── Artifacts do projeto
│   ├── Builder do projeto
│   └── Deploy do projeto
├── Tarefas
│   ├── Inbox
│   ├── Em execução
│   ├── Aguardando aprovação
│   ├── Concluídas
│   ├── Falhas e retries
│   ├── Filtros e busca
│   └── Reabrir, duplicar e arquivar
└── Superfície de execução
    ├── Chat/timeline
    ├── Estado do plano
    ├── Logs e terminal
    ├── Browser preview
    ├── Canvas/editor
    ├── Artefato preview
    ├── Exportar
    ├── Aprovar/rejeitar
    ├── Parar/retomar
    └── Feedback e auditoria
```

## 3. Árvore atual do Ollama Full

A árvore abaixo representa o que já existe ou está documentado no fork público. Ela não deve ser confundida com a árvore-alvo.

```text
Ollama Full
├── Runtime Ollama upstream [ATUAL]
│   ├── Inferência local
│   ├── Modelos e catálogo
│   ├── API Ollama
│   ├── API OpenAI-compatible
│   ├── API Anthropic-compatible
│   └── CLI e launch integrations
├── Runtime agentic em internal/agent [ATUAL/PARCIAL]
│   ├── Mission runtime
│   ├── Planner e validação de plano
│   ├── Tools e capability grants
│   ├── Sandbox e workspace containment
│   ├── Approvals
│   ├── Event history, traces e metrics
│   ├── Recovery, retry e cancelamento
│   ├── Artifacts, hashes e versões
│   ├── AgentOrchestrator e swarm de papéis
│   ├── ResearchEngine com citações e SSRF guard
│   ├── Memory, projects e ingestion
│   ├── Skills e MCP stdio/Remote HTTP
│   ├── Desktop Commander local/Remote MCP [ADAPTER]
│   ├── Scheduler, webhooks e jobs
│   ├── Browser Operator Playwright [ADAPTER]
│   ├── Desktop companion Linux/macOS/Windows [ADAPTER]
│   ├── Connectors allowlisted [ADAPTER]
│   ├── OAuth/OIDC/SAML/MFA/RBAC [ADAPTER]
│   ├── PostgreSQL/RLS, Redis/DLQ e OTLP [ADAPTER]
│   ├── Media, OCR e vision [ADAPTER]
│   ├── Builder/canvas/undo/redo/exporters [ADAPTER]
│   ├── Deploy Vercel/Netlify/generic [ADAPTER]
│   ├── Company OS: identidade, departamentos, roadmap, KPIs, backlog, ciclos e budget [ATUAL/PARCIAL]
│   └── Device pairing, push e colaboração [PARCIAL]
├── Multi-provider em internal/multillm [ATUAL/ADAPTER]
│   ├── Provider OpenAI-compatible
│   ├── Provider Anthropic
│   ├── Provider CLI com allow_execution
│   ├── Namespacing e catálogo de modelos
│   ├── Roteamento por capacidade/prioridade
│   ├── Auto aliases e local/private
│   ├── Tradução de formatos Ollama/OpenAI/Anthropic
│   ├── Auth server-side e redaction
│   ├── HTTPS, limites, redirects e SSRF controls
│   └── OmniRoute via endpoint genérico local [ADAPTER NOVO]
├── API server/agent_routes.go [ATUAL]
│   ├── Missões e eventos
│   ├── Auth, organizações e colaboração
│   ├── Jobs, traces e métricas
│   ├── Research, ingestion e devices
│   ├── Media e OCR
│   ├── Builders, preview, export e deploy
│   └── Policies e approvals
├── UI Web/Desktop [PARCIAL]
│   ├── Chat e histórico
│   ├── Onboarding
│   ├── Connect/Claude/Codex settings
│   ├── Agentic Console
│   ├── Mission timeline inicial
│   ├── Metrics, orchestration e research
│   └── Settings completa ainda em evolução
├── Companion Desktop [ADAPTER]
│   ├── Linux
│   ├── macOS
│   ├── Windows
│   ├── Pairing
│   ├── Capability report
│   ├── Heartbeat
│   └── Revogação
├── Mobile Expo [PARCIAL]
│   ├── Missões
│   ├── Approvals
│   ├── Auth
│   ├── Cache/outbox offline
│   └── Base para push/EAS
├── Infraestrutura [ADAPTER]
│   ├── PostgreSQL e RLS
│   ├── Redis workers/DLQ/replay
│   ├── OpenTelemetry Collector
│   ├── WebSocket/mTLS
│   ├── Docker Compose de desenvolvimento
│   └── CI/SBOM
└── Documentação pública [ATUAL]
    ├── Manual Ollama Full
    ├── Architecture/API/Integrations
    ├── Roadmap e fases de entrega
    ├── Matriz de paridade
    ├── Screenshots reais
    ├── Mockups marcados como conceito
    ├── Contributing e Security
    └── Changelog e release preview
```

## 4. Árvore-alvo: Ollama Full unificado

Esta é a árvore que deve orientar a implementação. Ela combina a navegação observável do Manus com recursos de coding agents, roteamento de providers, builders, pesquisa, operação local e segurança empresarial.

```text
Ollama Full — Unified Agentic Desktop
├── 0. Workspace, identidade e controle
│   ├── Workspace switcher
│   ├── Organizações, equipes e projetos
│   ├── Conta, perfil e personalização
│   ├── Plano, quotas, custo e uso por provider
│   ├── Local-first mode e privacy mode
│   ├── Região, timezone e idioma
│   ├── RBAC/ABAC, grupos e policy packs
│   ├── SSO OIDC/SAML, MFA e recovery codes
│   ├── Sessions, devices e revogação
│   ├── Secrets manager e DLP
│   └── Auditoria, retenção e exportação de dados
├── 1. Shell de desktop
│   ├── Sidebar recolhível
│   ├── Nova tarefa
│   ├── Agente
│   ├── Habilidades
│   ├── Plugins
│   ├── Agendado
│   ├── Biblioteca
│   ├── Projetos
│   ├── Tarefas
│   ├── Empresa / Company OS
│   ├── Configurações
│   ├── Busca global
│   ├── Command palette
│   ├── Notifications/inbox
│   ├── Split panes
│   ├── Tema claro/escuro
│   ├── Teclado, foco, screen reader e reduced motion
│   └── Responsive desktop/tablet/web
├── 2. Nova tarefa e entrada multimodal
│   ├── Chat composer
│   ├── Texto, arquivo, imagem, áudio e vídeo
│   ├── Drag-and-drop e clipboard
│   ├── Voice input e transcrição
│   ├── Web/source selection
│   ├── Model/provider picker
│   │   ├── Local Ollama
│   │   ├── Claude/Anthropic
│   │   ├── Codex/OpenAI
│   │   ├── OmniRoute/auto
│   │   ├── OpenAI-compatible catalog
│   │   └── CLI agents explicitamente autorizados
│   ├── Mode picker
│   │   ├── Chat
│   │   ├── Coding
│   │   ├── Research
│   │   ├── Browser
│   │   ├── Computer use
│   │   ├── Builder
│   │   ├── Media
│   │   └── Local-only
│   ├── Templates de missão
│   ├── Drafts e histórico
│   └── Approval preview antes de executar
├── 3. Orquestrador autônomo
│   ├── Intent parser
│   ├── Planner
│   ├── Decomposer/map-reduce
│   ├── Specialist agents
│   │   ├── Researcher
│   │   ├── Coder
│   │   ├── Browser operator
│   │   ├── Data analyst
│   │   ├── Designer
│   │   ├── Writer
│   │   ├── QA/tester
│   │   ├── Security reviewer
│   │   └── Release manager
│   ├── Provider/model routing
│   ├── Tool selection
│   ├── Budget, timeout e rate limits
│   ├── Plan approval e action approval
│   ├── Observation, reflection e correction
│   ├── Checkpoints e resumability
│   ├── Retry, compensation e rollback
│   ├── Dead-letter queue e replay
│   ├── Human takeover
│   └── Final audit antes de conclusão
├── 4. Coding agent de alto nível
│   ├── Repository/worktree manager
│   ├── File tree e search
│   ├── Terminal sandbox
│   ├── Patch/diff viewer
│   ├── Test runner
│   ├── Lint/typecheck/build
│   ├── Browser preview
│   ├── Issue-to-branch-to-PR
│   ├── Coding plans
│   ├── Context compaction e summaries
│   ├── Claude Code adapter
│   ├── Codex adapter
│   ├── Cline/Roo/Aider-like approval loop
│   ├── OpenHands/Devin-like long-running task loop
│   ├── SWE-agent-like issue workflow
│   └── Local models com fallback controlado
├── 5. Browser e computer use
│   ├── Browser profiles isolados
│   ├── Navigate, click, type, select e scroll
│   ├── Upload/download
│   ├── Screenshots e visual grounding
│   ├── Browser network/console diagnostics
│   ├── Takeover humano
│   ├── Desktop screen capture
│   ├── Mouse, keyboard e clipboard
│   ├── Process control com grants
│   ├── Device pairing
│   ├── TLS 1.3/mTLS
│   └── Approval obrigatório para efeitos externos/sensíveis
├── 6. Pesquisa e conhecimento
│   ├── Pesquisa multiagente
│   ├── Search/fetch com citations
│   ├── Browser research
│   ├── PDFs, DOCX, XLSX e imagens
│   ├── OCR e vision
│   ├── Deduplicação, cache e hash
│   ├── Robots/SSRF policy
│   ├── Source confidence e provenance
│   ├── Embeddings e semantic memory
│   ├── Project knowledge base
│   ├── Notebook/data analysis
│   └── Relatório, referências e exportação
├── 7. Skills, plugins, MCP e A2A
│   ├── Skill registry
│   ├── Manifests, versões e assinatura
│   ├── Enable/disable por projeto
│   ├── Scopes, allowlists e capability grants
│   ├── MCP stdio/HTTP
│   ├── Desktop Commander Remote MCP
│   ├── Lifecycle, health e logs
│   ├── A2A agent cards e tasks
│   ├── Sandboxing de plugins
│   ├── Consentimento e approval
│   └── Rollback de versão
├── 8. Automação e integrações
│   ├── Scheduler
│   ├── Cron/interval/timezone
│   ├── Webhooks verificados
│   ├── Idempotency keys
│   ├── Retries/backoff/DLQ/replay
│   ├── GitHub
│   ├── Google Workspace
│   ├── E-mail
│   ├── Slack/Discord
│   ├── WhatsApp
│   ├── Jira/Notion/Microsoft 365
│   ├── OAuth, refresh e revogação
│   ├── Tenant isolation
│   └── Connector health, quotas e audit
├── 8.5 Company OS e operação empresarial
│   ├── Empresa como organização/tenant
│   ├── Identidade, posicionamento, oferta e modelo de negócio
│   ├── Departamentos CEO, produto, engenharia, marketing, vendas, suporte e operações
│   ├── Roadmap estratégico, metas e KPIs
│   ├── Backlog priorizado e ciclos diários/semanais
│   ├── CRM, leads e pipeline
│   ├── Pesquisa, prospecção e campanhas de marketing
│   ├── Redes sociais, afiliados, ecommerce e dropshipping por connectors allowlisted
│   ├── E-mail, anúncios, domínio, deploy, analytics e conversões
│   ├── Budget, approval para gasto/anúncio/contrato/mensagem e auditoria
│   ├── Relatórios financeiros/operacionais e avaliação contínua de agentes
│   └── Pausa automática por erro, gasto excessivo, fraude ou anomalia
├── 9. OmniRoute e roteamento de modelos
│   ├── Gateway OpenAI-compatible
│   ├── Endpoint local configurável
│   ├── Model discovery
│   ├── Auto route e named profiles
│   ├── Fallback e circuit breaker
│   ├── Provider/model lockout
│   ├── Quotas, custo e saúde
│   ├── Compression/context budgets
│   ├── API key/OAuth status sem expor segredo
│   ├── Local-only nunca faz fallback remoto
│   ├── Health check e smoke test
│   └── Logs de routing sem prompt sensível
├── 10. Builders e criação
│   ├── Website builder
│   │   ├── Component tree
│   │   ├── Rich components
│   │   ├── Bindings/events
│   │   ├── Responsive preview
│   │   ├── Code export
│   │   └── Deploy
│   ├── App builder
│   │   ├── Screens/navigation
│   │   ├── Data/auth
│   │   ├── API integration
│   │   └── Mobile/web preview
│   ├── Game builder
│   │   ├── Scene/entity system
│   │   ├── Assets/audio
│   │   ├── Physics/input
│   │   ├── Play preview
│   │   └── Multiplayer adapter
│   ├── Slides builder
│   ├── Document builder
│   ├── Dashboard/chart builder
│   ├── Canvas visual
│   ├── Drag-and-drop
│   ├── Undo/redo/history
│   ├── Multiplayer/CRDT
│   ├── Comments/presence
│   ├── Preview/sandbox
│   ├── Export PDF/DOCX/PPTX
│   └── Publish/deploy com approval
├── 11. Artefatos e biblioteca
│   ├── Artifact manifest
│   ├── Content hash
│   ├── Versions e branches
│   ├── Diff
│   ├── Preview por tipo
│   ├── Download
│   ├── Exportadores profissionais
│   ├── Permissions/sharing
│   ├── Retention e garbage collection
│   └── Provenance e dependency list
├── 12. Projetos e colaboração
│   ├── Contexto persistente
│   ├── Memory scope
│   ├── Files/sources
│   ├── Tasks/missions
│   ├── Members/roles
│   ├── Comments/mentions
│   ├── Presence
│   ├── Realtime snapshots
│   ├── Conflict resolution
│   ├── Activity/audit feed
│   └── Export/transfer/archive
├── 13. Mobile companion
│   ├── Login/SSO/MFA
│   ├── Mission inbox
│   ├── Approvals
│   ├── Notifications/push
│   ├── Offline cache/outbox
│   ├── Conflict resolution
│   ├── Artifacts/preview/download
│   ├── Device pairing
│   ├── Project/task control
│   └── Android/iOS distribution
├── 14. Observabilidade e operações
│   ├── Structured logs
│   ├── Correlation IDs
│   ├── Mission/event traces
│   ├── Prometheus metrics
│   ├── OTLP exporter/collector
│   ├── Queue depth/latency/failures
│   ├── Provider health/quotas/cost
│   ├── SLOs and alerts
│   ├── Incident/runbook
│   ├── Replay and recovery
│   ├── SBOM/dependency scan
│   └── CI/CD, signed release e rollback
└── 15. Segurança de produto
    ├── Approval gates
    ├── Sandbox e path containment
    ├── SSRF e egress allowlist
    ├── Secret redaction
    ├── AuthN/AuthZ server-side
    ├── Tenant/RLS isolation
    ├── Rate limits e resource budgets
    ├── Upload validation
    ├── CSP/CORS/CSRF/session security
    ├── Plugin/MCP isolation
    ├── Audit log e tamper evidence
    ├── Privacy/local-only mode
    └── Security review antes de release
```

## 5. O melhor padrão de cada família de harness

| Família | Padrão absorvido no Ollama Full | Limite de integração |
|---|---|---|
| Manus | Missões de alto nível, projetos, skills, plugins, tarefas agendadas, biblioteca, artifacts, builders e superfície desktop | A paridade é observável; internals, UI proprietária e serviços fechados não são copiados |
| Claude Code | Coding agent orientado a ferramentas, contexto de repositório, terminal, edição e workflow iterativo | Requer adapter/credencial ou modelo local compatível |
| Codex | Execução de coding task com patch, testes, sandbox e aprovação | Capacidades dependem da API/sessão e não são inventadas localmente |
| OmniRoute | Gateway único, auto routing, fallback, quotas, perfis, health e compressão | Integração é externa via endpoint OpenAI-compatible; upstreams exigem contas próprias |
| Cursor, Windsurf, Zed e Void | Experiência de editor, contexto inline, diff, comandos e feedback rápido | A UI será implementada no projeto, sem copiar assets proprietários |
| Cline, Roo Code, Aider e Continue | Loop plan-act-observe, aprovação de comandos, edição incremental e suporte local | Cada tool exige capability e teste de autorização |
| OpenHands, Devin e SWE-agent | Missões longas, issue-to-branch, sandbox, testes e recuperação | Execução persistente exige worker, limites e observabilidade reais |
| Perplexity e pesquisa agentic | Pesquisa com fontes, síntese e citações | Robots, SSRF, disponibilidade e termos das fontes precisam ser respeitados |
| AutoGen, CrewAI, MetaGPT, ChatDev e OpenSquad | Papéis especializados, swarm, reducer, síntese e coordenação | Concorrência, orçamento, conflitos e cancelamento precisam ser server-side |
| Builder.io, FlutterFlow, v0, Lovable, Bolt, Replit Agent, Base44 e afins | Builder visual, componentes, dados, preview, exportação e deploy | Preview não é deploy; publicação exige approval, credencial e smoke real |
| Databutton, Marblism, Create.xyz e Polsia | Geração de produto a partir de especificação e integração de backend | Secrets, dados e persistência devem permanecer no ambiente autorizado |
| E2B | Sandbox de execução e isolamento por missão | Sandbox local/container e políticas do operador continuam necessários |
| LiteLLM e Dify | Abstração de providers, workflow, catálogo e observabilidade | Não incorporar internals; reutilizar somente padrões/compatibilidades permitidas |

## 6. Estrutura técnica correspondente

A árvore de produto exige uma árvore de código coerente. A parte marcada como **existente** deve ser preservada; a parte marcada como **alvo** orienta os próximos incrementos.

```text
ollama-classe-a-plus/
├── cmd/                                  # CLI e launchers existentes
├── internal/
│   ├── agent/                            # existente: runtime agentic e subsistemas
│   │   ├── runtime.go, planner.go        # missão, plano e execução
│   │   ├── tools.go, sandbox             # ferramentas e isolamento
│   │   ├── browser.go, desktop*.go       # browser/computer adapters
│   │   ├── research.go, ingestion.go     # pesquisa e conhecimento
│   │   ├── memory/context.go             # memória e contexto
│   │   ├── mcp.go, mcp_remote.go         # MCP stdio e Remote MCP
│   │   ├── connectors.go, company.go     # integrações e Company OS
│   │   ├── queue.go, redis_queue.go      # workers, retries, DLQ e replay
│   │   ├── auth.go, saml.go, secrets.go   # identidade e segredos
│   │   ├── builder.go, exporters.go      # builders e artifacts
│   │   ├── deploy.go                     # publicação com approval
│   │   └── traces.go, telemetry.go       # observabilidade
│   ├── multillm/                         # existente: registry e proxy de providers
│   │   ├── registry.go                   # catálogo, policies e segurança
│   │   ├── proxy.go                      # forwarding/traduções
│   │   ├── anthropic*.go                 # compatibilidade Anthropic
│   │   └── catalog.go                    # catálogo de CLIs
│   └── ...
├── server/
│   ├── routes.go                         # servidor e multi-provider
│   ├── agent_routes.go                   # API agentic
│   ├── agent_tls.go                      # TLS/mTLS
│   └── ...
├── app/ui/app/src/
│   ├── routes/
│   │   ├── index.tsx                     # alvo: home/Nova tarefa
│   │   ├── agentic.tsx                   # atual: Agentic Console
│   │   ├── settings.tsx                  # alvo: Settings vertical slice
│   │   ├── library.tsx                   # alvo: Biblioteca
│   │   ├── projects.tsx                  # alvo: Projetos
│   │   ├── scheduled.tsx                 # alvo: Agendado
│   │   ├── skills.tsx                    # alvo: Habilidades
│   │   ├── plugins.tsx                   # alvo: Plugins/conectores
│   │   ├── company.tsx                   # atual: Company OS
│   ├── components/
│   │   ├── AppSidebar.tsx                # atual/base: shell
│   │   ├── AgenticConsole.tsx            # atual: missão e métricas
│   │   ├── MissionTimeline.tsx           # alvo: timeline completa
│   │   ├── ApprovalCenter.tsx             # alvo: approvals
│   │   ├── ArtifactPanel.tsx              # alvo: artifacts/preview/diff
│   │   ├── ProviderPicker.tsx             # alvo: local/Claude/Codex/OmniRoute
│   │   ├── SettingsWorkspace.tsx          # alvo: config sanitizada
│   │   ├── BuilderCanvas.tsx              # alvo: drag-and-drop
│   │   └── ...
│   └── lib/                              # clientes, status, permissões e testes
├── apps/mobile-agentic/                  # existente/parcial: Expo mobile
├── examples/
│   ├── dz23-providers.json               # providers gerais
│   ├── dz23-omniroute.json               # preset OmniRoute local
│   └── ...
├── deploy/                               # compose, workers e observabilidade
├── docs/agentic/
│   ├── PRODUCT_TREE.md                   # este documento
│   ├── PARITY_MATRIX.md                  # matriz de evidências
│   ├── ARCHITECTURE.md
│   ├── API.md
│   ├── INTEGRATIONS.md
│   └── ROADMAP.md
└── audit/                                # checkpoints e auditorias
```

## 7. Jornadas de aceitação da árvore-alvo

A árvore só será considerada concluída quando as jornadas abaixo forem executadas com evidência. Não basta renderizar menus ou retornar HTTP 200.

1. O operador abre **Nova tarefa**, anexa arquivos, escolhe local, Claude, Codex ou OmniRoute e vê o status real da credencial sem que o segredo apareça.
2. O operador cria uma missão. O planner apresenta etapas, o runtime executa ferramentas em sandbox, a timeline mostra eventos e a missão pode ser pausada e retomada.
3. Uma missão de coding cria um worktree, edita arquivos, executa testes, mostra diff e solicita approval antes de publicar ou alterar um sistema externo.
4. Uma missão de pesquisa navega por fontes permitidas, coleta citações, salva memória contextual e exporta um relatório verificável.
5. Uma missão usa Browser Operator ou companion com capability grant. O takeover humano funciona quando a política exige intervenção.
6. Um builder cria um artefato, permite preview, undo/redo, exportação e deploy somente depois de approval. O sistema registra hash, versão e logs.
7. Um usuário no mobile recebe uma approval, responde offline, sincroniza sem duplicar ação e observa o resultado no desktop.
8. Um usuário de outra organização não consegue ler missão, memória, artifact, connector, trace ou segredo da primeira organização.
9. Uma falha de provider ou worker produz retry controlado, DLQ, replay e trace correlacionado sem vazar prompt, token ou dados privados.
10. O modo `local-only` nunca encaminha uma missão para Claude, Codex, OmniRoute ou outro provider remoto.

## 8. Ordem de implementação

A ordem recomendada é vertical, não apenas visual. Primeiro deve ser fechado o shell com **Nova tarefa → missão → timeline → approval → artifact**. Em seguida, o provider picker deve conectar local, Claude, Codex e OmniRoute aos contratos já existentes. Depois devem ser concluídos Settings, Biblioteca, Projetos e Agendado. Por último, a expansão de builders, colaboração, mobile de produção, instaladores assinados e deploys externos deve avançar com seus ambientes reais.

A cada rodada, atualizar esta árvore, a matriz de paridade, o roadmap, o changelog, os testes e as screenshots. Recursos externos permanecem marcados como dependências até que exista uma execução autorizada e reproduzível.

## Referências

[1]: https://github.com/diegosouzapw/OmniRoute "OmniRoute — gateway AI OpenAI-compatible"

[2]: https://github.com/ollama/ollama "Ollama — repositório upstream"

[3]: https://manus.im/ "Manus — página pública do produto"


## Atualização técnica desta rodada

```text
Ollama Full Runtime
├── Provider Intelligence
│   ├── Provider Router: capabilities, health, latency, cost, privacy, quality
│   ├── Grok Live: Responses, streaming, retry, circuit, sources vs memory
│   └── Evaluation OS: coding, browser, tools, security, memory, planning, recovery
├── Company Operations
│   ├── Department Agents: supervisor, budget, SLA, memory, pause/resume
│   └── Social OS sandbox: accounts, drafts, approval, simulated publish, metrics
└── Plugin Lifecycle
    ├── Connector enable/disable/remove
    ├── MCP stdio enable/disable/remove
    ├── Remote MCP enable/disable/remove
    └── Skill enable/disable/remove with trust fail-closed
```

Esses nós possuem contratos, implementação e testes locais. A árvore não implica que providers externos, contas sociais, marketplaces ou dispositivos físicos estejam conectados.
