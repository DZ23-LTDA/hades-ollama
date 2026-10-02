## Checkpoint vigente — Release candidate e auditoria final — 2026-10-02 01:28 UTC
```yaml
state: RELEASE_CANDIDATE_LOCAL_GATES_PASS_EXTERNAL_CI_BLOCKED
branch: recovery/ollama-full-snapshot
sha: 2444e6d2
proofs_local:
  - go build ./...: PASS (após remover node_modules gerado)
  - CGO_ENABLED=0 go build ./...: PASS
  - go test ./internal/agent ./server: PASS
  - go vet ./...: PASS
  - frontend tsc/lint/vitest/build: PASS
  - bundle budget: PASS (1,823,863 bytes bruto / 530,772 gzip)
  - contracts/integrity/gofmt/diff-check: PASS
  - shell E2E 10 telas desktop+mobile: PASS (0 HTTP, 0 console)
external_status:
  - class-a-plus-integrity: BLOCKED_EXTERNAL (run 36949729206, job não iniciou)
  - dz23-agentic-quality: BLOCKED_EXTERNAL (run 36949729166, job não iniciou)
  - reason: GitHub reportou "account is locked due to a billing issue"; rerun attempt 2 confirmou o mesmo bloqueio
known_limits:
  - Ollama local sem modelos instalados neste ambiente
  - credenciais/IdP/WhatsApp/MCP/deploy externos continuam BLOCKED_EXTERNAL ou NOT_CONFIGURED
next_action: proprietário deve resolver o bloqueio de billing do GitHub; então rerun class-a-plus-integrity, dz23-agentic-quality e test.yaml no SHA 2444e6d2
```




## Checkpoint — S1 egress provider fechado — 2026-10-01 20:56 -03
```yaml
state: TESTING
current_task: S2 — isolamento forte do MCP local
completed_tasks:
  - S1: internal/egresspolicy agora é a política única consumida por agent e multillm
  - S1: provider bloqueia loopback, privado, metadata, CGNAT, RFC2544 198.18/15, ranges de documentação e IPv6 documentation
proofs:
  - go test ./internal/egresspolicy ./internal/multillm ./internal/agent -run Egress|SSRF|DNSRebind|Provider -count=1: PASS
  - git diff --check: PASS
remaining: S2,S3,S4,S5,S6,S7,R1-R4,F1-F2,U1-U11,E1-E8,D1-D6,A1-A10
next_action: inspecionar o sandbox forte existente e o processo MCP local; implementar bloqueio honesto ou isolamento real
```

## Checkpoint vigente — Hardening consolidado P0/P1 — 2026-10-01 20:54 -03
```yaml
state: EXECUTING
branch: recovery/ollama-full-snapshot
head: ef9c21d2
objective: fechar os P0/P1 do adendo de 8 auditorias com testes negativos e execução real, sem declarar production/public pronto
current_task: S1 — alinhar egress multillm ao classificador zero-trust do agent
completed_tasks:
  - auditoria consolidada criada em audit/AUDIT_PONTA_A_PONTA_2026-10-01.md
  - inventário P0/P1 recebido em pasted_content_21.txt
acceptance:
  - provider bloqueia loopback, privado, metadata, CGNAT, 198.18/documentation e DNS rebinding
  - testes negativos reproduzem bloqueios
  - nenhum gate ou assertion será removido/enfraquecido
next_action: implementar S1, testar, revisar diff e atualizar checkpoint
blockers:
  - credenciais externas, IdP, WhatsApp, deploy e assinatura de release permanecem dependências externas
```

## LINT-DEBT — gates locais concluídos — 2026-10-01 12:16 UTC
```yaml
state: LOCAL_GATES_PASS_CI_PENDING
branch: recovery/ollama-full-snapshot
implemented:
  - golangci-lint v2: 0 issues, sem desabilitar linters
  - testes Go completos e race dos pacotes críticos
  - builds nativo, CGO_ENABLED=0 e GOOS=windows
  - gates frontend, contratos, integrity e go vet
  - correções portáveis em fixtures, contexto, HTTP bodies e loops bounded
pending:
  - commit/push
  - test.yaml completo, incluindo macOS race
next_action: publicar e acompanhar todos os jobs no SHA final; corrigir apenas falhas reproduzíveis
```
# Estado da missão autônoma — Ollama DZ23 Agentic Platform

## Checkpoint vigente — Auditoria ampla e hardening local — 2026-09-30 19:41 -03
```yaml
state: LOCAL_FIXES_APPLIED_GATES_PENDING
branch: recovery/ollama-full-snapshot
scope: auditoria ampla de segurança, qualidade, UX e release
implemented:
  - reject imported .git control plane and validate contained Git worktree
  - reject self-approval for deployment ledger while preserving replay conflict
  - harden media/download redirects and special-use IP handling
  - bound/redact WhatsApp error responses
  - atomic builder export with regular-file-only policy and artifact descriptor close
  - accessible ModelPicker/import dialog; Home attachment errors visible
  - authenticated artifact preview/download and WhatsApp admin RBAC
  - reproducible Linux metadata and trusted release signing key requirement
proofs_local:
  - go test ./internal/agent ./server: PASS
  - frontend tsc/lint/vitest/build: PASS
pending:
  - contracts/integrity and full platform gates
  - final static scan and diff review
  - commit/push and remote CI on final SHA
  - real browser re-audit for changed Studio/Import flows
next_action: run remaining gates; fix only reproducible failures; do not claim production or 100% parity
```

## Checkpoint vigente — UI-2 launcher central Home/Nova tarefa — 2026-09-30 19:00 -03
```yaml
state: COMPLETED_CI_GREEN
branch: recovery/ollama-full-snapshot
base_commit: 3dd15dec
objective: tornar Home/Nova tarefa o launcher central com composer, anexos, importação de projeto e ações rápidas
implemented:
  - ImportProjectDialog reutilizável extraído de ProductWorkspacePage
  - Home oferece Anexar arquivo e Importar projeto junto dos modos Automático/Manual e ações rápidas
  - Projetos reutiliza o mesmo componente; chamadas GitHub/ZIP e segurança permanecem reais
proofs:
  - npx tsc -b: PASS
  - npm run lint: PASS
  - npx vitest run: 37 arquivos/257 testes PASS
  - npm run build: PASS
  - node scripts/verify-contracts.mjs: PASS
  - bash scripts/check-class-a-plus-integrity.sh: PASS
  - Playwright desktop/mobile: quatro screenshots UI-2 e auditoria com zero erros
ci:
  class_a_plus_integrity: {id: 36782993795, conclusion: success}
  dz23_agentic_quality: {id: 36782993723, conclusion: success}
  sha: 849faa97751d43f96e6f9d4e05dd1137d4d9e356
next_action: nenhum dentro do UI-2; manter integrações privadas e ZIP grande como dependências externas honestas
rollback: preservar 3dd15dec; sem reset/clean/force-push
```


## Checkpoint vigente — WIN-1 build Windows/sem-CGO — 2026-09-30 18:10 -03
```yaml
state: COMPLETED_CI_GREEN
branch: recovery/ollama-full-snapshot
base_commit: ebd51c69
objective: corrigir o build completo sem CGO e o cross-build Windows, substituir os.DevNull no Git e adicionar gate CI permanente
facts:
  - CGO_ENABLED=0 go build ./... passa com fallback MLX/xgrammar/webview e app desktop isolado por plataforma
  - GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./... passa
  - internal/agent/git_repo.go e workspace_snapshot.go usam GIT_CONFIG_SYSTEM/GLOBAL vazios, sem os.DevNull
acceptance:
  - CGO_ENABLED=0 go build ./... PASS
  - GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./... PASS
  - go build ./..., go test ./internal/agent ./server e frontend gates PASS
  - CI executa os dois builds e reprova regressões
rollback: preservar ebd51c69; sem reset/clean/force-push
ci:
  class_a_plus_integrity: {id: 36777180701, conclusion: success}
  dz23_agentic_quality: {id: 36777180620, conclusion: success}
  sha: b703f2a505f2daa1b0980c11580ca594265775dc
next_action: nenhum dentro do WIN-1; smoke nativo Windows/DACL/MLX/WebView continuam limites externos documentados
```

## Checkpoint vigente — FASE 11 Import Full Project — 2026-09-30 17:45 -03
```yaml
state: COMPLETED
branch: recovery/ollama-full-snapshot
base_commit: 2b6ff7d7
implemented:
  - GitHub HTTPS import with Zero-Trust transport, approved redirects and server-side optional auth
  - resumable ZIP upload connected to direct-on-disk bounded extraction and incremental ingestion
  - isolated git worktree/branch, tenant checks, traversal/symlink/size/file limits, no auto-execution
  - authenticated project import routes and accessible Projects dialog with GitHub/ZIP tabs
proofs:
  - go test -run 'Import|GitHubImport|LargeUpload|ZipStream|Clone|Ingest' ./internal/agent ./server: PASS
  - go test ./internal/agent ./server and go build ./...: PASS
  - npx tsc -b, npm run lint, npx vitest run (37 files/257 tests), npm run build: PASS
  - node scripts/verify-contracts.mjs and bash scripts/check-class-a-plus-integrity.sh: PASS
  - Playwright live public GitHub import: PASS; desktop/mobile evidence and browser-console-audit-f11.json have zero console/http errors
ci:
  class_a_plus_integrity: {id: 36774562486, conclusion: success}
  dz23_agentic_quality: {id: 36774562246, conclusion: success}
next_action: none for Fase 11; private GitHub remains NOT_CONFIGURED without operator credentials
rollback: preserve commit 2b6ff7d7; no reset/clean/force-push
```

```yaml
mission_id: dz23-agentic-platform-2026-09-21
objective: Evoluir o Ollama DZ23 para uma plataforma agentic local-first com execução segura, ferramentas, memória, artefatos, automações, integrações e superfícies Desktop/Mobile.
state: RECOVERING
iteration: 1
started_at: 2026-09-21
last_progress_at: 2026-09-21
branch: feat/dz23-claude-codex-desktop
base_commit: 4a3dd89ed9224f2d6cd87579434ca6798334fa4a

acceptance:
  - O núcleo de missão possui estados explícitos, persistência, eventos e recuperação.
  - Toda ferramenta possui contrato, limites, autorização e resultado auditável.
  - Terminal e execução de código são isolados por política; não há shell arbitrário por padrão.
  - Memória, projetos, skills, MCP, artefatos e jobs têm fronteiras versionadas.
  - Browser/computer-use, integrações externas, mídia, Web/Desktop/Mobile têm adapters testáveis.
  - Fluxos críticos têm testes unitários, contrato, integração, segurança e smoke real.
  - Nenhuma capacidade é marcada como completa sem prova observável.

completed:
  - Auditoria do commit 4a3dd89e e confirmação da base multi-provider.
  - Ponte Anthropic Messages para providers OpenAI-compatible.
  - Inventário externo para Claude Desktop e Codex Desktop.
  - Testes focados de multillm, proxy e launch.

current_task: Arquitetura e núcleo vertical de missões.
pending:
  - Runtime de missão com planner, executor, observer e recovery.
  - Store durável, eventos e idempotência.
  - Tool registry, approvals, sandbox e artifacts.
  - Memória/projetos/skills/MCP.
  - Browser/computer-use e integrações.
  - Mídia, builder, jogos, slides, dados e mobile.
  - Auditorias independentes, build e publicação.

blockers:
  - GitHub push bloqueado por 403 para a identidade dz23trading-collab.
  - Build nativo completo depende das toolchains de cada plataforma.
  - Integrações externas reais dependem de credenciais e autorização do usuário.

risks:
  - Execução de shell, browser, desktop e conectores podem produzir efeitos externos; exigir aprovação e allowlists.
  - Não implementar browser/computer-use falso baseado apenas em respostas do modelo.
  - Não persistir segredos em missão, memória, logs ou artefatos.

next_action: Criar contratos agentic e implementar uma missão vertical persistente com ferramenta segura de filesystem.
```

## Regra de retomada

Antes de continuar, conferir este arquivo contra `git status`, o commit atual, os testes e os artefatos. Retomar pela primeira tarefa não concluída; não repetir a ponte Claude/Codex já validada.


## Adendo — fase agentic multimodal, builders, auth e colaboração — 2026-09-21

```yaml
state: VERIFIED_LOCAL_PHASE
completed:
  - auth: organizations, memberships, RBAC, revocable tokens, OAuth PKCE state, AES-GCM credential storage and refresh contract
  - jobs: persistent queue, retries, dead-letter queue, replay, trace spans and SSE events
  - media: HTTPS image/video/speech/transcription adapters plus deterministic WAV smoke fixture
  - builders: website/app/game/slides/dashboard templates, preview containment, ZIP export and local versioned publish
  - desktop: Linux implementation preserved, Darwin and Windows adapters compile cross-platform
  - collaboration: persistent comments, presence, snapshots and SSE stream
  - mobile: Expo SecureStore session, EAS profiles, Android/iOS identifiers and typecheck
proofs:
  - CGO_ENABLED=0 go test ./internal/agent -count=1: PASS
  - GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build ./internal/agent: PASS
  - GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./internal/agent: PASS
  - apps/mobile-agentic npm ci && npm run typecheck: PASS
  - server gate: BLOCKED by existing upstream MLX symbols/toolchain, not by internal/agent tests
next_action: Resolve MLX build environment, then commit/review/push the verified phase.
```


## Adendo — fase 4 multiagente, pesquisa, devices e ingestão — 2026-09-21

```yaml
state: TESTING
implemented:
  - multiagent_orchestrator: specialist_roles, concurrency_budget, retries, cancellation, persistence and synthesis conflict detection
  - deep_research: multi-source HTTPS fetch, HTML extraction, cache, citations, hashes, robots policy and SSRF guard
  - device_pairing: one-time pairing, capability report, heartbeat, online/offline state and revocation
  - document_ingestion: txt/md/html/json/csv/pdf/docx/xlsx readers, chunking and provenance-backed memories
proofs:
  - CGO_ENABLED=0 go test ./internal/agent -count=1 after phase 4: PASS
  - focused swarm/research/device tests: PASS
remaining:
  - resolve full server build with CGO/MLX toolchain and run server tests
  - integrate production mTLS/WebSocket companion transport, OAuth provider adapters and distributed stores
next_action: run full server gate after build-essential installation, then commit phase 4 and package artifacts.
```


## Fechamento da fase 4 — 2026-09-21

```yaml
state: CANDIDATE_COMPLETED
proofs:
  - CGO_ENABLED=0 go test ./internal/agent -count=1: PASS
  - CGO_ENABLED=1 go test ./server -count=1: PASS
  - CGO_ENABLED=1 go build -o ollama-dz23-agentic-server .: PASS
  - frontend ./node_modules/.bin/tsc --noEmit: PASS
  - git diff --check: PASS
features:
  - multiagent orchestration with seven roles, budgets, retries, cancellation and synthesis
  - deep research with citations, hashes, cache, HTML extraction, robots and SSRF policy
  - device pairing, capability report, heartbeat, offline state and revocation
  - safe PDF/DOCX/XLSX/document ingestion with chunks and provenance
  - Web Agentic Console controls for missions, orchestration and research
blockers:
  - production mTLS/WebSocket companion transport, OCR/vision providers, distributed stores, public hosting deploy, signed desktop/mobile releases and physical device validation remain
next_action: run final diff review, commit phase 4 and build verified ZIP; do not publish main.
```


## Adendo — fase 5 infraestrutura distribuída, companion seguro e mobile offline — 2026-09-21

```yaml
state: TESTING
implemented:
  - postgres_store: idempotent migrations, mission upsert and append-only event persistence
  - redis_queue: enqueue, claim, retry backoff, dead-letter, replay and worker integration
  - opentelemetry: optional OTLP HTTP exporter with HTTPS-by-default and local noop fallback
  - companion_transport: WebSocket handshake, device token, capabilities, heartbeat, TLS/mTLS policy and origin allowlist
  - mobile_offline: cached mission, queued actions and synchronization retry
  - ci_sbom: Go/web/mobile gates and CycloneDX artifact workflow
  - local_stack: PostgreSQL, Redis and OTEL Collector compose files
proofs:
  - CGO_ENABLED=0 go test ./internal/agent: PASS
  - CGO_ENABLED=1 go test ./server: PASS
  - apps/mobile-agentic npm ci && npm run typecheck: PASS
pending_production:
  - integration tests against real PostgreSQL/Redis/OTLP endpoints
  - certificate rotation, RLS and tenant isolation review
  - remote push notifications, mobile conflict resolution and physical device tests
next_action: verify root build, diff, commit phase 5 and package ZIP; preserve unresolved external credential/hardware gates.
```


## Fechamento da fase 5 — 2026-09-21

```yaml
state: CANDIDATE_COMPLETED
commit: 7b0609d8e43414c006150bc4de9f5fbe9571b57a
proofs:
  - CGO_ENABLED=0 go test ./internal/agent -count=1: PASS
  - CGO_ENABLED=1 go test ./server -count=1: PASS
  - apps/mobile-agentic npm ci && npm run typecheck: PASS
  - CGO_ENABLED=1 go build -o ollama-dz23-agentic-phase5 .: PASS
  - zipinfo -t phase5 archive: PASS
features:
  - PostgreSQL store and Redis queue adapters with local fallback
  - OTLP OpenTelemetry exporter with HTTPS-by-default and noop fallback
  - TLS/mTLS policy and WebSocket companion handshake/heartbeat
  - mobile cached mission and offline action queue
  - CI quality workflow, CycloneDX SBOM and local infra compose stack
external:
  - push remains blocked by GitHub HTTP 403 for dz23trading-collab
remaining:
  - real PostgreSQL/Redis/OTLP integration tests, certificate rotation, RLS, remote push, physical devices, public deploy adapters, OCR/model providers and signed releases
next_action: proceed to the production-adapter phase only after external credentials, certificates and test infrastructure are available.
```


## Fechamento da fase 7 — 2026-09-21

```yaml
state: CANDIDATE_COMPLETED
features:
  - saml_sp: crewjam metadata validation, signed AuthnRequest, one-time RelayState, ACS claims and tenant provisioning
  - oauth_connectors: tenant-aware encrypted credential resolution for connector.http
  - builder_history: rich component fields, validation, persistent undo/redo and API endpoints
  - postgres_rls: FORCE ROW LEVEL SECURITY and tenant policies excluding blank organization records
  - companion_tls: TLS 1.3/mTLS listener configuration with per-handshake certificate reload
proofs:
  - CGO_ENABLED=0 go test ./internal/agent -count=1: PASS
  - CGO_ENABLED=1 go test ./server ./cmd/launch ./internal/multillm -count=1: PASS
  - CGO_ENABLED=1 go build -o ollama-dz23-phase7-bin .: PASS
  - apps/mobile-agentic npm ci && npm run typecheck: PASS
  - JSON manifests and git diff --check: PASS
remaining_external:
  - SAML end-to-end IdP and certificate fixtures
  - real PostgreSQL/Redis/OTLP execution outside CI and production RLS migration review
  - physical companion/mobile tests, push credentials, signed installers, app-store distribution and public cloud deploy credentials
  - full editorial exporters, CRDT collaboration, local generative media models and complete hosting adapters
next_action: commit phase 7, create reproducible archive, then continue with deploy adapters and physical/infrastructure gates without publishing main.
```


## Fechamento da fase 8 — 2026-09-21

```yaml
state: CANDIDATE_COMPLETED
features:
  - deployments: Vercel, Netlify and generic hosting adapters
  - publish_security: workspace containment, symlink rejection, file/size limits, no redirects, HTTPS outside loopback
  - publish_approval: external deployment endpoint requires approved=true
proofs:
  - CGO_ENABLED=0 go test ./internal/agent -count=1: PASS
  - CGO_ENABLED=1 go test ./server ./cmd/launch ./internal/multillm -count=1: PASS
  - CGO_ENABLED=1 go build -o ollama-dz23-phase8-bin .: PASS
  - generic HTTPS deployment smoke and external HTTP rejection: PASS
remaining_external:
  - real Vercel/Netlify/AWS/Cloudflare accounts, project IDs and permissions
  - signed installers, physical devices, store distribution and full local media models
next_action: commit phase 8, package a reproducible archive and continue provider-specific production smoke tests only with operator credentials.
```

## Rodada 2026-09-22 — árvore de produto, shell desktop e OmniRoute
```yaml
state: CANDIDATE_COMPLETED
branch: feat/manus-parity-omniroute
features:
  - product_tree: Manus observable surface, current Classe A+ tree and unified target tree
  - parity_matrix: evidence states and acceptance journeys
  - desktop_shell: real routes for projects, library, scheduled, skills, plugins and tasks
  - settings_control_center: sanitized runtime/provider/approval/integration status
  - safe_config_endpoint: GET /api/agent/v1/config/safe without secrets or private paths
  - omniroute_loopback: explicit local HTTP opt-in plus preset and proxy smoke tests
  - screenshots: Chromium captures for shell routes and Settings
proofs:
  - CGO_ENABLED=0 go test ./internal/agent -count=1: PASS
  - CGO_ENABLED=1 go test ./server ./cmd/launch ./internal/multillm -count=1: PASS
  - UI npm run build: PASS
  - Chromium route capture: PASS for projects, library, scheduled, skills, plugins, tasks, agentic and settings
  - git diff --check: PASS
remaining_external:
  - OmniRoute instance, credentials, provider discovery and fallback smoke outside the fixture
  - CRUD persistence for new shell surfaces and complete provider settings editor
  - real browser/desktop/mobile devices, distributed staging, IdP, deploy accounts and signed releases
next_action: review diff, commit the feature branch, publish branch for review and continue the vertical flow Nova tarefa -> missão -> approval -> artifact.
```


## Checkpoint atual — fluxo vertical e HarnessRouter — 2026-09-22

```yaml
state: CANDIDATE_COMPLETED
branch: feat/manus-parity-omniroute
completed:
  - shell_home: composer, recommendations, shortcuts and local-first offline fallback
  - project_schedule_crud: tenant-aware create/list/update/delete with live API
  - mission_console: provider selector for Ollama, Claude, Codex, OmniRoute and project association
  - catalogs: connectors, MCP, skills and CLI status with null-safe frontend normalization
  - observability: JSON metrics route registered for Agentic Console
  - harnessrouter_adapter: OpenAI Responses-compatible provider with server-side harness_id metadata injection
  - public_docs: README, manual, parity matrix, roadmap, changelog and screenshots updated
proofs:
  - go test ./internal/agent ./server -count=1: PASS
  - UI npm run build: PASS
  - live CRUD smoke for projects and schedules against 127.0.0.1:3001: PASS
  - live mission/list/events smoke: PASS
  - Chromium capture for home, projects, library, scheduled, skills, plugins, tasks, agentic and settings: PASS
  - node app/ui/app/scripts/smoke-shell.mjs against Vite and local server: PASS
  - HarnessRouter metadata proxy regression test: pending final Go gate
remaining_external:
  - HarnessRouter instance, provider key and installed harnesses for streaming/follow-up/cancel/artifact validation
  - builder drag-and-drop and CRDT collaboration
  - distributed PostgreSQL/Redis/OTLP, IdP, real deploy accounts and physical desktop/mobile tests
next_action: run focused multillm/server gates, review diff, commit and publish the feature branch without changing main.
```


## Checkpoint atual — Company OS e Desktop Commander Remote MCP — 2026-09-22

```yaml
state: CANDIDATE_COMPLETED
branch: feat/manus-parity-omniroute
completed:
  - company_os_store: tenant-aware identity, departments, roadmap, goals, backlog, cycles, budget and risk pause
  - company_os_api: CRUD identity, report, roadmap, goals, backlog, cycles, pause/resume, anomaly and spend endpoints
  - company_os_ui: /company functional workspace with creation, KPI/backlog/cycle forms and guardrails
  - company_scheduler_guard: company:// workspaces skip mission creation while paused
  - remote_mcp_adapter: Streamable HTTP, HTTPS policy, timeout, allowlist, optional server-side bearer and approval tool
  - desktop_commander_presets: local stdio and official remote endpoint examples without secrets
  - public_docs: Company OS, Remote MCP, API, integrations, parity matrix, roadmap and changelog
proofs:
  - go test ./internal/agent ./server ./internal/multillm -count=1: PASS
  - remote MCP httptest JSON/SSE, bearer and URL policy tests: PASS
  - CompanyStore lifecycle, persistence, budget pause and anomaly tests: PASS
  - JSON preset validation: PASS
  - UI npm run build: PASS
remaining_external:
  - Desktop Commander OAuth PKCE, account, device pairing, physical agent and revocation flow
  - real Remote MCP call against an authorized paired test device
  - CRM, social, affiliate, ecommerce, advertising, logistics and analytics connectors
  - distributed Company OS storage/RLS, IdP, deploy accounts and physical desktop/mobile tests
next_action: review diff, run integrity and full gates, update screenshots if needed, commit and publish the feature branch without changing main
```


## Verificação final da rodada — 2026-09-22

```yaml
state: CANDIDATE_COMPLETED
proofs:
  - Class A+ integrity guard: PASS
  - CGO_ENABLED=0 go test ./internal/agent -count=1: PASS
  - CGO_ENABLED=1 go test ./server ./cmd/launch ./internal/multillm -count=1: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI npm run build: PASS
  - Company OS live API smoke: PASS
  - Chromium shell smoke including /company and MCP catalog: PASS
  - Company OS screenshot capture and visual inspection: PASS
  - git diff --check: PASS
external_blockers_unchanged:
  - OAuth PKCE, device pairing and physical Desktop Commander agent
  - real social, CRM, affiliate, ecommerce, ads, logistics and analytics providers
  - distributed Company OS RLS, IdP, deployment accounts and signed devices
next_action: commit and push the verified feature branch, then update the public PR; do not merge into main automatically
```


## Verificação adicional — jornadas verticais Growth/Builder/providers — 2026-09-22

```yaml
state: CANDIDATE_COMPLETED
branch: feat/manus-parity-omniroute
completed:
  - growth_os_sandbox: campaigns, approvals, launch/pause, affiliate programs/links/conversions, products, orders, inventory and fulfillment
  - company_growth_ui: Growth OS panel embedded in /company with local metrics and explicit sandbox labels
  - builder_vertical: create, visual update, preview/hash artifact, export, local publish and external deploy approval gate
  - provider_sessions: OpenAI-compatible streaming translation with gateway auth and server-side provider bearer
  - remote_mcp_evidence: official setup source recorded; adapter tests remain green
proofs:
  - go test ./internal/agent ./server -count=1: PASS
  - focused multi-provider streaming/OmniRoute/HarnessRouter tests: PASS
  - focused Remote MCP tests: PASS
  - scripts/smoke-company-growth.sh against 127.0.0.1:3001: PASS
  - scripts/smoke-builder.sh against 127.0.0.1:3001: PASS
  - UI npm run build: PASS
  - Chromium capture /company with Growth OS: PASS
external_blockers:
  - Desktop Commander OAuth PKCE, account, device pairing, physical agent, remote call and revocation; no test token/device is provisioned
  - real social/CRM/affiliate/ecommerce/ads/logistics/analytics connectors and compliance staging
  - real provider sessions for Claude/Codex/OmniRoute/HarnessRouter with operator credentials
  - distributed PostgreSQL/Redis/OTLP, IdP, deployment accounts and physical desktop/mobile tests
next_action: run full integrity/backend/UI gates, stage the complete diff, commit and update the public PR
```


## Verificação adicional — Composio, xAI/Grok e social commerce — 2026-09-22

```yaml
state: CANDIDATE_COMPLETED
branch: feat/manus-parity-omniroute
completed:
  - composio_remote_mcp_preset: headers_env server-side, JSON-RPC allowlist, approval and local regression test
  - xai_responses_preset: HTTPS OpenAI-compatible provider, bearer server-side and local passthrough test
  - commercial_source_audit: official Composio, xAI, TikTok Shop, Instagram and Shopify sources recorded
  - parity_docs: README, integrations, parity matrix, changelog and roadmap updated
proofs:
  - gofmt and focused Remote MCP tests: PENDING_FINAL_GATE
  - focused multi-provider xAI Responses test: PENDING_FINAL_GATE
  - JSON preset validation: PENDING_FINAL_GATE
  - integrity/backend/UI gates: PENDING_FINAL_GATE
external_blockers:
  - Composio API key, OAuth connected accounts and per-tenant session provisioning
  - xAI API key/quota and provider tool validation; Grok Bot cloud product is not embedded
  - TikTok Shop Partner Center app, seller/creator/partner authorization, scopes and region-specific sandbox
  - Meta/Instagram, YouTube, WhatsApp, Shopify and other marketplace credentials, app review and webhooks
  - compliance, idempotency, DLP, approvals, returns/refunds, logistics, payments and reconciliation tests
next_action: run final gates, commit and push the public PR; do not claim external accounts are connected
```


## Auditoria independente consolidada — 2026-09-22

```yaml
mission_id: class-a-plus-final-hardening-2026-09-22
state: EXECUTING
iteration: 1
objective: fechar blockers internos de segurança, wiring, tenant, MCP, QA, UI/mobile e release antes de nova declaração pública
source_report: audit/FINAL_HARNESS_AUDIT_2026-09-22.md
findings:
  critical: 15
  high: 32
  medium: 33
  low: 6
completed_before_this_phase:
  - adapters Composio/xAI e documentação pública
  - Growth OS sandbox, Builder smoke e provider session fixtures
  - branch pública feat/manus-parity-omniroute em PR #1
current_priority:
  - fail-closed auth e WebSocket /connect
  - containment resistente a symlink/TOCTOU e StepID
  - MCP loader, allowlist obrigatória, SSRF/redirect e correlation ID
  - Company/Growth tenant checks e approvals
  - baseline go test ./..., UI tests e mobile typecheck
  - CI/release gates e supply-chain pinning
external_blockers_preserved:
  - credenciais e OAuth de providers, Composio e Desktop Commander
  - sandbox/app review de TikTok Shop, Meta, Shopify e demais canais
  - PostgreSQL/RLS, Redis, OTLP, runners Windows/macOS, GPU e dispositivos móveis
  - assinatura/provenance de releases com identidade do publisher
strategy: corrigir somente controles internos reproduzíveis; marcar dependências externas como BLOCKED_BY_EXTERNAL_DEPENDENCY
next_action: inspecionar contratos P0 e implementar patches pequenos com testes de regressão
```


## Verificação final de hardening — 2026-09-22

```yaml
mission_id: class-a-plus-final-hardening-2026-09-22
state: CANDIDATE_COMPLETED
iteration: 2
objective: fechar blockers internos de segurança, wiring, tenant, MCP, QA, UI/mobile e release antes da publicação
completed:
  - fail_closed_agent_auth_outside_loopback
  - strict_mcp_and_remote_mcp_config_loading
  - remote_mcp_ssrf_redirect_timeout_and_correlation_guards
  - workspace_symlink_and_step_id_containment
  - auditable_approval_metadata_and_actor_tenant_binding
  - tenant_safe_growth_mutations_and_idempotent_fulfillment
  - resilient_settings_control_center_and_mobile_outbox
  - stale_cli_test_corrected_for_agent_command
  - vet_mutex_copy_and_context_cancel_fixes
proofs:
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=0 go test ./internal/agent -count=1: PASS
  - CGO_ENABLED=1 go test ./server ./cmd/launch ./internal/multillm -count=1: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest: 20 files, 199 tests, PASS
  - UI npm run build: PASS
  - mobile npm run typecheck: PASS
  - class-a-plus-integrity guard: PASS
  - final review: audit/FINAL_THREE_AGENT_REVIEW.md
external_blockers:
  - Composio, xAI, Desktop Commander, social, marketplace and logistics credentials/sandboxes
  - PostgreSQL/RLS, Redis, OTLP and IdP staging
  - GPU/local media models and physical desktop/mobile runners
  - signed installers, app stores and operator deployment accounts
next_action: commit verified hardening, push feature branch, update PR metadata; do not merge main automatically
```


## Incremento final pós-auditoria — 2026-09-22

```yaml
state: CANDIDATE_COMPLETED
completed:
  - mission_capability_policy_with_local_default_and_explicit_external_scopes
  - exact_fullpath_companion_handshake_bypass_only
  - capability_and_auth_regression_tests
proofs:
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - UI 20 files / 199 tests: PASS
  - UI build: PASS
  - functional smokes Growth/Builder/Shell: PASS
  - final Chromium capture: PASS
next_action: stage, commit and push verified branch; preserve external blockers in release notes
```


## Incremento operacional e lifecycle — 2026-09-22

```yaml
mission_id: class-a-plus-final-hardening-2026-09-22
state: CANDIDATE_COMPLETED
iteration: 3
objective: transformar adapters restantes em jornadas locais verificáveis e publicar a evolução com estado honesto
completed:
  - grok_live_responses_streaming_retry_circuit_and_sanitized_status
  - evaluation_os_deterministic_cases_and_evidence_based_provider_router
  - company_department_agents_with_budget_sla_pause_resume_and_supervisor
  - social_os_sandbox_with_oauth_pending_drafts_approval_and_metrics
  - server_side_lifecycle_for_connectors_mcp_remote_mcp_and_skills
  - builder_deploy_approval_checked_before_provider_configuration
  - public_docs_readme_roadmap_api_matrix_changelog_and_company_os_updated
proofs:
  - backend focused tests internal/agent internal/grok internal/multillm server: PASS
  - class-a-plus-integrity guard: PASS
  - UI npm run build: PASS
  - UI Vitest: 20 files / 199 tests: PASS
  - shell Chromium E2E: PASS
  - Growth OS live smoke: PASS
  - Builder live smoke including approval gate: PASS
  - JSON/provider presets and diff check: PASS
external_blockers:
  - xAI key/quota and live tools/web search/Voice/Imagine validation
  - Composio connected accounts and OAuth per tenant
  - Desktop Commander account, PKCE, pairing and physical agent
  - TikTok Shop/Meta/Instagram/YouTube/WhatsApp/Shopify apps, scopes, sandboxes and webhooks
  - PostgreSQL/RLS, Redis, OTLP and IdP staging
  - GPU/local media models and physical desktop/mobile runners
  - signed installers, app stores and operator deployment accounts
next_action: run complete release gates, capture final screens, commit and push feature branch; do not merge main automatically
```


## Publicação da rodada operacional — 2026-09-22

```yaml
commit: d0809136fdda6871a7371bd632750972cbb65305
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
local_tree: clean_after_commit
public_ci: queued_at_publish
next_action: external credentialed journeys and distributed staging; do not claim universal production readiness
```


## Smoke de APIs e correção de CI — 2026-09-22

```yaml
mission_id: class-a-plus-readiness-api-smoke-2026-09-22
state: CANDIDATE_COMPLETED
branch: feat/manus-parity-omniroute
base_before_commit: dfe07d5665b11f025fb36c6382298e8c053733f5
proofs:
  - provider catalog checks executed without printing secrets
  - OpenRouter free inference HTTP 200
  - local Ollama Classe A+ gateway model listing HTTP 200
  - local Ollama Classe A+ gateway chat HTTP 200
  - Browser Operator focused test passes locally
  - integrity guard passes locally
changes:
  - install Python Playwright/Chromium in dz23-agentic-quality CI job
  - improve Browser Operator stderr diagnostics
  - add public readiness report
external_blockers:
  - rotate all credentials supplied in plaintext
  - wait for new GitHub CI result
  - distributed PostgreSQL/Redis/OTLP staging
  - OAuth, app reviews, devices, GPU, signed installers, stores and deploy accounts
next_action: commit and push readiness report plus Browser Operator CI fix; do not claim universal production readiness
```


## Retomada após auditoria independente ampliada — 2026-09-22

```yaml
mission_id: class-a-plus-final-hardening-2026-09-22
state: FIXING
iteration: 4
objective: corrigir P0 internos reproduzíveis por slices verticais, sem declarar produção-ready enquanto isolamento, approvals, sandbox, egress/DLP, Company/Builder e release permanecerem incompletos
branch: feat/manus-parity-omniroute
head_before_publish: 9b2d2e87
working_tree_at_checkpoint: hardening local não commitado; revisar antes de publicar

completed_this_iteration:
  - connectors: organização obrigatória, matching de path por segmento e bloqueio de destinos DNS privados
  - remote_mcp: validação de TokenEnv/HeadersEnv e headers de transporte
  - mcp_stdio: validação estrita de nomes de variáveis de ambiente
  - plugin_lifecycle: owner/admin obrigatório quando auth está ativa
  - browser_operator: fallback para Chromium conhecido quando caminho configurado não existe
  - plugin_routes_test: matriz owner/admin/operator/viewer/local mode
  - ci: instalação/verificação mais robusta de Browser Operator
  - docs: readiness, roadmap, changelog e revisão ajustados para não afirmar conclusão universal

proofs_local:
  - scripts/check-class-a-plus-integrity.sh: PASS
  - focused connector/mcp/plugin/browser/server tests: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest: 20 arquivos / 199 testes PASS
  - UI npm run build: PASS, warning conhecido de chunks >500KB
  - mobile npm run typecheck: PASS
  - Browser Operator com executável inválido e fallback local: PASS

audited_open_p0:
  - tenant isolation/object authorization para orchestration, traces, devices/pairing, Builder, plugins/MCP/skills e CRUD completo
  - approval policy separada de execute, aprovador autorizado, CAS/nonce/expiração e anti-auto-approval
  - sandbox/MCP stdio sem isolamento forte comprovado; faltam seccomp/cgroups/rlimits/PID/memory/process-group limits
  - SSRF/DNS rebinding/redirect-chain e IP efetivamente conectado para Remote MCP/connectors/media
  - DLP/redaction antes de Step.Result, events, traces e payloads externos
  - CompanyCreateRequest allowlisted, empresa/agente pausados e gasto atômico dentro do budget
  - Builder OrganizationID, ownership, entry/symlink/XSS e manifest de artifacts
  - dev-token, OAuth redirect allowlist, bearer desktop e claims de provider/UI
  - CI/release: quality dependency, SBOM/provenance/signing e smoke pós-build

external_blockers:
  - credenciais e OAuth de providers, Composio, xAI e Desktop Commander
  - app review/sandboxes de TikTok Shop, Meta, Shopify e canais de commerce
  - PostgreSQL/RLS, Redis, OTLP, IdP, Docker/staging distribuído
  - runners Windows/macOS, GPU, dispositivos móveis, assinatura e lojas

current_task: publicar somente a camada de hardening local já implementada; depois iniciar tenant isolation por testes negativos
next_actions:
  - revisar diff e atualizar este checkpoint com o SHA publicado
  - commit/push sem merge para main e confirmar PR #1
  - criar duas organizações em teste HTTP e provar 403/no mutation em Builder, Company/Growth, orchestration, traces e devices
  - atualizar novamente a auditoria apenas com evidência reproduzível
classification: preview/local RC em hardening; NÃO final, NÃO production-ready
rollback: reverter o commit deste slice na branch feature; não force-push e não alterar main
```


## Publicação do slice de hardening — 2026-09-22

```yaml
state: FIXING
iteration: 4
commit: 3f7e4065
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
published: true
working_tree: clean
changes_published:
  - connector organization scope, path-segment matching and DNS private-destination denial
  - strict MCP stdio/Remote MCP environment/header validation
  - owner/admin guard for authenticated global plugin lifecycle
  - Browser Operator Chromium fallback and CI diagnostics
  - dev-token restricted to actual TCP loopback plus explicit development flag
  - negative plugin and auth policy regressions
proofs:
  - scripts/check-class-a-plus-integrity.sh: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest: 20 files / 199 tests PASS
  - UI npm run build: PASS; known >500KB chunk warning remains
  - mobile npm ci && npm run typecheck: PASS
  - Browser Operator invalid configured path fallback: PASS
warnings:
  - npm ci reports 18 audit findings in the existing mobile dependency graph (11 moderate, 7 high); not silently treated as resolved
  - distributed Docker/PostgreSQL/Redis/OTLP gates remain N/A because Docker/staging are unavailable here
open_p0_next:
  - implement Builder OrganizationID and object authorization first, with cross-tenant 403/no-mutation tests
  - then CompanyCreateRequest allowlist and atomic paused/budget spend tests
next_action: implement Builder tenant slice on top of 3f7e4065; do not merge main or claim final readiness
```


## Slice P0 Builder/Company validado — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 5
base_commit: 3f7e4065
working_tree: alterações do slice P0 ainda não commitadas
implemented:
  - BuilderProject.OrganizationID e métodos GetForOrganization/ListForOrganization
  - ownership server-side em list/create/visual/undo/redo/preview/export/publish/deploy/preview-file
  - entry obrigatório, template HTML escaped e preview/export sem symlink externo
  - server.Builder HTTP cross-tenant 403/no-mutation regression
  - CompanyCreateRequest allowlisted e reset de server-managed fields
  - RecordAgentSpend nega empresa/agente pausados e over-budget sem incremento
proofs:
  - focused Builder/Company/AgentAuth tests: PASS
  - HTTP cross-tenant Builder test: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
remaining_p0:
  - orchestration/traces/devices/pairing and plugin/MCP/skills object scope
  - approval policy/role separation/CAS/nonce and external-effect gates
  - sandbox process isolation, Remote MCP/connectors/media DNS rebinding and DLP
  - paused state coverage for all Company/Growth/Social mutations
next_action: review secret diff, commit and push slice; then continue with orchestration/traces/devices tenant tests
```


## Publicação do slice P0 Builder/Company — 2026-09-22

```yaml
state: FIXING
iteration: 5
commit: 4344b24b
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: clean
published: true
proofs: integrity, go test ./..., go vet, go build, UI Vitest/build, mobile typecheck and HTTP cross-tenant Builder regression all PASS
classification: preview/local RC em hardening; P0s remain open
next_slice: organization-scoped orchestration/traces/devices/pairing with negative tests and no-mutation assertions
```


## Slice P0 orchestration/traces/devices validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 6
base_commit: 4344b24b
working_tree: alterações da slice ainda não commitadas
implemented:
  - organization_id em OrchestrationJob e Plan/Get/Run/Cancel scoped
  - organization_id em TraceSpan e criação/listagem scoped
  - device List/Get/Heartbeat/Revoke scoped
  - pairing code vinculado à organização de origem e override cross-tenant rejeitado
  - HTTP negative tests para orchestration, traces, devices e no-mutation
proofs:
  - focused store/handler tests: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC em hardening; P0s restantes não resolvidos
next_action: commit/push slice e depois plugins/MCP/skills/artifacts ou approval policy, sem mergear main
```


## Publicação da slice P0 orchestration/traces/devices — 2026-09-22

```yaml
state: FIXING
iteration: 6
commit: 07c56edd
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: clean
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and HTTP cross-tenant P0 regression all PASS
classification: preview/local RC em hardening
next_slice: plugins/MCP/skills/artifacts ownership or approval policy; do not merge main or claim final readiness
```


## Slice P0 mission approvals validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 7
base_commit: 066b8334
working_tree: alterações de approval ainda não commitadas
implemented:
  - owner/admin policy para decisão autenticada
  - DecideApprovalForActorCAS com expected mission version
  - nonce obrigatório e single-use no endpoint
  - UI/tipos transportam nonce e policy
  - negative tests para viewer/operator, wrong nonce, replay e CAS
proofs:
  - focused approval tests: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
open_p0:
  - Company/Growth/Social booleans still are not audit-grade approval authorities
  - plugin/MCP/skills/artifacts ownership and sandbox/egress/DLP remain
classification: preview/local RC em hardening
next_action: commit/push, then migrate Company/Growth/Social approvals or harden MCP process isolation
```


## Publicação da slice P0 mission approvals — 2026-09-22

```yaml
state: FIXING
iteration: 7
commit: 45cd4f42
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and approval negative tests all PASS
classification: preview/local RC em hardening
next_slice: Company/Growth/Social audit-grade approvals or MCP process isolation; no main merge
```


## Slice P0 CompanyApproval ledger validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 8
base_commit: 8b5d9752
working_tree: alterações CompanyApproval ainda não commitadas
implemented:
  - CompanyApproval server-side para campaign, affiliate_program, order e social_draft
  - policy, nonce, expiry, actor, organization, reason e status auditáveis
  - endpoints approve usam pending lookup, owner/admin policy e Company version CAS
  - UI Growth/Social envia nonce da decisão pendente
proofs:
  - focused Company/domain/HTTP tests: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
open_p0:
  - RecordSpend still accepts approved boolean
  - sandbox/MCP/egress/DLP and plugin ownership remain
classification: preview/local RC em hardening
next_action: commit/push, then migrate spend approvals or harden MCP stdio process isolation
```


## Publicação da slice P0 CompanyApproval ledger — 2026-09-22

```yaml
state: FIXING
iteration: 8
commit: 45b6e019
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and Company HTTP/domain approval regressions all PASS
classification: preview/local RC em hardening
next_slice: migrate RecordSpend approved boolean or sandbox/MCP process isolation; no main merge
```


## Slice P0 spend approvals validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 9
base_commit: f87f95fa
working_tree: spend approval/domain/UI/docs ainda não commitados
implemented:
  - HTTP spend route no longer accepts approved as caller authority
  - pending spend approval returns 202 without budget debit
  - generic approval decision endpoint with owner/admin, nonce and CAS
  - atomic budget debit only after approved decision
  - CompanyApprovalQueue UI for pending spend actions
proofs:
  - focused Go/domain/HTTP tests: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC em hardening
open_risks: strong sandbox/MCP isolation, SSRF/DNS rebinding/DLP, plugin ownership and external credentials/deploys
next_action: commit/push this slice, then select sandbox/MCP process isolation or egress/DLP as next P0
```


## Publicação da slice P0 spend approvals — 2026-09-22

```yaml
state: FIXING
iteration: 9
commit: 99ea0b01
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and spend HTTP/domain regressions all PASS
classification: preview/local RC em hardening
next_slice: sandbox/MCP process isolation or egress/DLP; no main merge
```


## Slice P0 DLP validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 10
base_commit: 018e08dc
working_tree: DLP changes and tests still uncommitted
implemented:
  - recursive RedactValue with structured sensitive-key handling
  - credential patterns for PEM, GitHub, OpenAI, OpenRouter, xAI, AWS, Slack and Bearer
  - Step.Result, error, event, trace and JSON/Postgres persistence redaction
proofs:
  - focused DLP/runtime tests: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
open_risks: connector/MCP outbound payload classification, SSRF/DNS rebinding, redirects, strong sandbox/process isolation
classification: preview/local RC em hardening
next_action: commit/push, then address egress classification or MCP stdio isolation
```


## Publicação da slice P0 DLP — 2026-09-22

```yaml
state: FIXING
iteration: 10
commit: 157e7012
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and DLP token-injection regressions all PASS
classification: preview/local RC em hardening
next_slice: egress classification/SSRF or MCP stdio process isolation; no main merge
```


## Slice P0 Remote MCP egress validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 11
base_commit: e663bd01
working_tree: Remote MCP egress changes/tests/docs still uncommitted
implemented:
  - proxy disabled for Remote MCP transport
  - same-origin allowed redirects only
  - actual connected socket IP private-range rejection
  - local regressions for redirect and private address
proofs:
  - focused Remote MCP tests: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
open_risks: Media/Connector egress parity, TLS/DNS distributed tests, strong sandbox/MCP process isolation
classification: preview/local RC em hardening
next_action: commit/push, then unify Media/Connector egress or harden MCP stdio lifecycle
```


## Publicação da slice P0 Remote MCP egress — 2026-09-22

```yaml
state: FIXING
iteration: 11
commit: b74c25ea
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and Remote MCP egress regressions all PASS
classification: preview/local RC em hardening
next_slice: Media/Connector egress parity or MCP stdio lifecycle; no main merge
```


## Slice P0 MCP stdio lifecycle validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 12
base_commit: ed12f252
working_tree: MCP stdio changes/tests/docs still uncommitted
implemented:
  - absolute non-symlink executable and bounded args
  - explicit private cwd, temporary by default, cleanup and restart recreation
  - minimal environment with explicit variable allowlist
  - request/response/stderr limits and stderr DLP
  - Unix process group termination with parent-safe fallback
proofs:
  - focused MCP tests: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC; MCP stdio containment best-effort, not strong sandbox
open_risks: seccomp/cgroups/rlimits/PID-memory-CPU controls, physical platform tests, Media/Connector egress parity
next_action: commit/push, then continue Media/Connector egress or platform sandbox primitives
```


## Publicação da slice P0 MCP stdio lifecycle — 2026-09-22

```yaml
state: FIXING
iteration: 12
commit: 86302569
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and MCP stdio negative lifecycle tests all PASS
classification: preview/local RC; stdio containment best-effort, not strong sandbox
next_slice: Media/Connector egress parity or platform sandbox primitives; no main merge
```


## Slice P0 Media egress/download validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 13
base_commit: dbc325b5
working_tree: Media egress changes/tests/docs still uncommitted
implemented:
  - media proxy disabled and redirects rejected
  - actual connected IP private-range rejection
  - bounded response reads with overflow detection
  - MIME and magic validation for PNG/JPEG/WebP/MP4/WAV
  - loopback HTTP allowed only for explicit local providers
proofs:
  - focused Media tests: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC em hardening
open_risks: Connector/upload egress parity, distributed DNS/TLS tests, strong sandbox/process isolation
next_action: commit/push, then continue Connector egress or platform isolation
```


## Publicação da slice P0 Media egress/download — 2026-09-22

```yaml
state: FIXING
iteration: 13
commit: b84ac97c
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and Media egress negative tests all PASS
classification: preview/local RC em hardening
next_slice: Connector/upload egress parity or platform sandbox primitives; no main merge
```


## Slice P0 Connector egress parity validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 14
base_commit: b1a098bf
working_tree: Connector egress changes/tests/docs still uncommitted
implemented:
  - proxy disabled on default and custom HTTP transports
  - redirects blocked on every request
  - connected IP validation retained
  - request 1 MiB and response 2 MiB bounded limits
proofs:
  - focused Connector tests: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC em hardening
open_risks: external payload classification/credential injection, upload egress, distributed network tests, strong sandbox
next_action: commit/push, then close a focused product/security contract or platform sandbox primitive
```


## Publicação da slice P0 Connector egress parity — 2026-09-22

```yaml
state: FIXING
iteration: 14
commit: d6506a1a
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and Connector egress negative tests all PASS
classification: preview/local RC em hardening
next_slice: external payload policy or platform sandbox primitives; no main merge
```


## Slice P0 OAuth redirect URI allowlist validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 15
base_commit: 21f49f69
working_tree: OAuth redirect changes/tests/docs still uncommitted
implemented:
  - provider-scoped exact redirect allowlist from environment
  - canonical redirect URI used in PKCE state
  - HTTPS requirement and explicit loopback HTTP opt-in
  - fragment/userinfo/opaque URI rejection
proofs:
  - focused Auth/OAuth tests: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC em hardening
open_risks: OAuth endpoint egress parity, real IdP/staging, session storage/UI, strong sandbox
next_action: commit/push, then address OAuth endpoint egress or secure browser token storage
```


## Publicação da slice P0 OAuth redirect URI allowlist — 2026-09-22

```yaml
state: FIXING
iteration: 15
commit: 150273ee
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and OAuth redirect negative tests all PASS
classification: preview/local RC em hardening
next_slice: OAuth endpoint egress parity or secure browser token storage; no main merge
```


## Slice P0 session handling web/mobile validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 16
base_commit: 74f722f9
working_tree: session handling changes/tests/docs still uncommitted
implemented:
  - web agent token held in memory, no localStorage reads
  - web 401/403 clears session and emits local event
  - mobile SecureStore cleared on auth failure
  - mobile approval nonce, busy lock and accessibility labels
  - logout confirmation and cache/outbox cleanup
proofs:
  - web security Vitest tests: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
  - integrity/Go full gates pending in this slice
classification: preview/local RC em hardening
open_risks: web login wiring, native secure-storage physical builds, backend session cookie/CSRF strategy, strong sandbox
next_action: run full gates, commit/push, then continue backend OAuth egress or sandbox controls
```


## Publicação da slice P0 session handling web/mobile — 2026-09-22

```yaml
state: FIXING
iteration: 16
commit: 772b07a1
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI security Vitest/build, mobile typecheck all PASS
classification: preview/local RC em hardening
next_slice: wire web login to memory session, OAuth endpoint egress, or platform sandbox; no main merge
```


## Slice P0 OAuth endpoint egress validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 17
base_commit: f1d93445
working_tree: OAuth endpoint egress changes/tests/docs still uncommitted
implemented:
  - safe client for discovery/JWKS/userinfo/token exchange
  - proxy disabled and redirects blocked
  - connected private IP rejection
  - discovery endpoint HTTPS/issuer validation
proofs:
  - focused OAuth tests: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199: PASS
  - UI build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC em hardening
open_risks: real IdP/staging and refresh/revocation, web login wiring, strong sandbox
next_action: commit/push, then continue secure session/CSRF contract or sandbox resource controls
```


## Publicação da slice P0 OAuth endpoint egress — 2026-09-22

```yaml
state: FIXING
iteration: 17
commit: 7e72d9d9
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and OAuth egress negative tests all PASS
classification: preview/local RC em hardening
next_slice: secure session/CSRF contract or sandbox resource controls; no main merge
```


## Slice P0 tool process containment validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 18
base_commit: 16310fa2
working_tree: tool containment changes/tests/docs still uncommitted
implemented:
  - process-group lifecycle and cancellation kill
  - bounded/redacted stdout and stderr
  - explicit best-effort isolation/resource status
  - sandbox ulimit CPU/memory/PID/fd/filesize best effort
proofs:
  - focused tools tests: PASS
  - internal/agent suite: PASS
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest 20/199 and build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC em hardening; containment partial, not strong sandbox
open_risks: seccomp/cgroups/physical platform isolation, web login wiring, real IdP/staging
next_action: commit/push, then continue plugin/MCP tenant ownership or release/CI hardening
```


## Publicação da slice P0 tool process containment — 2026-09-22

```yaml
state: FIXING
iteration: 18
commit: 3d19f111
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, internal/agent suite, UI Vitest/build, mobile typecheck and process cancellation/redaction tests all PASS
classification: preview/local RC em hardening; containment partial, not strong sandbox
next_slice: plugin/MCP tenant ownership or release/CI hardening; no main merge
```


## Slice P0 plugin/MCP/skill ownership validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 19
base_commit: 88a6e7e3
working_tree: plugin ownership changes/tests/docs still uncommitted
implemented:
  - organization_id on Connector/MCP/Remote MCP/Skill metadata
  - organization-filtered catalog with global read-only compatibility
  - scoped enable/disable/remove lifecycle
  - cross-tenant negative tests
proofs:
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest/build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC em hardening
open_risks: server-owned tenant registration, skill attestation, strong MCP isolation, distributed/RLS proof
next_action: commit/push, then continue release/CI or plugin registration; no main merge
```


## Publicação da slice P0 plugin/MCP/skill ownership — 2026-09-22

```yaml
state: FIXING
iteration: 19
commit: 30b1ecb8
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and cross-tenant plugin tests all PASS
classification: preview/local RC em hardening
next_slice: server-owned tenant registration or release/CI hardening; no main merge
```


## Slice P0 CI/release quality gates validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 20
base_commit: 4e531827
working_tree: CI/release YAML and docs still uncommitted
implemented:
  - valid Browser Operator YAML indentation and fallback check
  - Go full test/vet/build gates in agentic quality workflow
  - UI Vitest/build and mobile install/typecheck in CI
  - release quality job required by build and publish jobs
proofs:
  - YAML parser: PASS
  - integrity guard: PASS
  - git diff --check: PASS
external_not_proven: GitHub Actions, Docker distributed integration, physical runners, signing, SBOM/provenance attestation
classification: preview/local RC em hardening
next_action: commit/push, then continue remaining P0s; no main merge
```


## Publicação da slice P0 CI/release quality gates — 2026-09-22

```yaml
state: FIXING
iteration: 20
commit: 66258da7
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: YAML parser, integrity guard and diff check PASS; CI now declares full Go/UI/mobile gates
classification: preview/local RC em hardening
next_slice: continue remaining P0s, especially runtime/provider contracts and release evidence; no main merge
```


## Slice P1 Grok/provider contract validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 21
base_commit: 111a6328
working_tree: Grok contract changes/tests/docs still uncommitted
implemented:
  - configured model allowlist
  - provider catalog validation in health probe
  - HTTP stream:true explicit 501 until SSE route exists
  - negative tests for arbitrary model, stream and upstream avoidance
proofs:
  - integrity guard: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest/build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC em hardening
open_risks: xAI credential/real provider validation, production SSE, remaining provider UI contracts
next_action: commit/push, then continue remaining P0/P1s; no main merge
```


## Publicação da slice P1 Grok/provider contract — 2026-09-22

```yaml
state: FIXING
iteration: 21
commit: 0739dec7
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity, Go tests/vet/build, UI Vitest/build, mobile typecheck and Grok negative tests all PASS
classification: preview/local RC em hardening
next_slice: continue remaining P0/P1s, especially provider/runtime UI contracts and release evidence; no main merge
```


## Slice P0 Compose/infrastructure defaults validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 22
base_commit: 6cb644f2
working_tree: Compose/init/docs/CI changes still uncommitted
implemented:
  - loopback-only Postgres/Redis/OTLP ports
  - env-required Postgres/Redis secrets
  - Redis requirepass and authenticated CI URL
  - removed fixed init SQL password
proofs:
  - integrity guard: PASS
  - git diff --check: PASS
  - docker compose config: NOT_RUN_DOCKER_UNAVAILABLE
  - distributed service integration: NOT_RUN_DOCKER_UNAVAILABLE
classification: preview/local RC em hardening
open_risks: distributed RLS/DLQ/OTLP proof, production TLS/secrets manager
next_action: commit/push, then continue remaining P0/P1s; no main merge
```


## Publicação da slice P0 Compose/infrastructure defaults — 2026-09-22

```yaml
state: FIXING
iteration: 22
commit: a07d1a68
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity/static Compose checks and diff check PASS; Docker integration remains NOT_RUN_DOCKER_UNAVAILABLE
classification: preview/local RC em hardening
next_slice: continue remaining P0/P1s, especially provider/runtime UI contracts and release evidence; no main merge
```


## Slice P1 provider selection contract validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 23
base_commit: e762761d
working_tree: provider runtime/UI/docs changes still uncommitted
implemented:
  - explicit provider field in mission request/result
  - server rejects Claude/Codex/OmniRoute/automatic until real adapter exists
  - UI disables unconnected providers and sends provider for local runtime
proofs:
  - integrity guard: PASS
  - focused/runtime Go tests: PASS
  - UI Vitest/build: PASS; known >500KB warning
classification: preview/local RC em hardening
open_risks: external provider adapters, credentials, OAuth refresh, streaming
next_action: commit/push, then continue remaining P0/P1s; no main merge
```


## Publicação da slice P1 provider selection contract — 2026-09-22

```yaml
state: FIXING
iteration: 23
commit: dcd3a19f
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: provider negative test, integrity, Go focused, UI Vitest/build PASS
classification: preview/local RC em hardening
next_slice: continue remaining P0/P1s, especially external adapters and release evidence; no main merge
```


## Slice P1 release artifact integrity validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 24
base_commit: f315d339
working_tree: release/docs changes still uncommitted
implemented:
  - reject missing/empty release payloads
  - optional provenance attestation gated by OLLAMA_ENABLE_ATTESTATIONS=true
  - explicit permissions for OIDC/attestations
proofs:
  - integrity guard: PASS
  - YAML parser: PASS
  - git diff --check: PASS
  - release execution/signing: NOT_RUN in sandbox
classification: preview/local RC em hardening
open_risks: real GitHub release, SBOM final artifact binding, signing credentials
next_action: commit/push, then continue remaining P0/P1s; no main merge
```


## Publicação da slice P1 release artifact integrity — 2026-09-22

```yaml
state: FIXING
iteration: 24
commit: 6d35ad20
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: integrity/YAML/diff checks PASS; release execution/signing NOT_RUN in sandbox
classification: preview/local RC em hardening
next_slice: continue remaining P0/P1s and external evidence; no main merge
```


## Slice P0 jobs/replay tenant scope validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 25
base_commit: 17325891
working_tree: jobs/runtime/server/tests/docs changes still uncommitted
implemented:
  - organization-filtered queue listing
  - organization-checked replay
  - HTTP cross-tenant 403/no-mutation regression
proofs:
  - focused Go tests: PASS
  - full Go test/vet/build: PASS
  - integrity: PASS
  - UI Vitest/build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC em hardening
open_risks: Redis distributed/restart proof and remaining execution surfaces
next_action: commit/push, then continue remaining P0/P1s; no main merge
```


## Publicação da slice P0 jobs/replay tenant scope — 2026-09-22

```yaml
state: FIXING
iteration: 25
commit: 287484e7
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: focused/full Go, vet/build, integrity, UI and mobile gates PASS; distributed Redis not run locally
classification: preview/local RC em hardening
next_slice: continue remaining P0/P1s and external evidence; no main merge
```


## Slice P0 artifact manifest path safety validada — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 26
base_commit: f10aa8e9
working_tree: artifact/docs changes still uncommitted
implemented:
  - reject symlink components before artifact hashing
  - reject paths resolving outside workspace
  - regression tests for external symlink and regular file
proofs:
  - focused/full Go tests: PASS
  - vet/build: PASS
  - integrity: PASS
  - UI Vitest/build: PASS; known >500KB warning
  - mobile typecheck: PASS
classification: preview/local RC em hardening
open_risks: distributed artifact stores/export signing and other external proofs
next_action: commit/push, then continue remaining P0/P1s; no main merge
```


## Publicação da slice P0 artifact manifest path safety — 2026-09-22

```yaml
state: FIXING
iteration: 26
commit: 4bf40774
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
proofs: focused/full Go, vet/build, integrity, UI and mobile gates PASS
classification: preview/local RC em hardening
next_slice: continue remaining P0/P1s and external evidence; no main merge
```


## Publicação do addendum da auditoria consolidada — 2026-09-22

```yaml
state: FIXING
iteration: 27
audit_commit: 146d203c
branch: feat/manus-parity-omniroute
remote: https://github.com/DZ23-LTDA/hades-ollama.git
pull_request: https://github.com/DZ23-LTDA/hades-ollama/pull/1
working_tree: checkpoint pending commit
published: true
classification: preview/local RC em hardening
note: FINAL_HARNESS_AUDIT now distinguishes historical findings from published partial remediations; it does not claim production readiness
next_action: continue remaining P0/P1s and external evidence; no main merge
```


## Remediação do CI distribuído — aguardando publicação — 2026-09-22

```yaml
mission_id: class-a-plus-final-hardening-2026-09-22
state: RELEASING
iteration: 28
base_commit: 2933f6e2a758aafaf47c9c96dcb3e1741d3e8f69
branch: feat/manus-parity-omniroute
working_tree: correção de workflow/guard e documentação ainda não commitada
trigger:
  - GitHub run 35735628693 falhou em TestDistributedPostgresRLSAndEvents porque o DSN usava a role Compose superusuária
  - passwords efêmeras no GITHUB_ENV apareceram no bloco de ambiente do log do passo
implemented:
  - workflow cria role não-superusuária ollama_agent_test para o smoke RLS
  - passwords ficam somente em variáveis locais do passo; GITHUB_ENV removido do job distribuído
  - integrity guard bloqueia regressão de GITHUB_ENV e exige o contrato tenant_password
proofs_local:
  - integrity guard: PASS
  - workflow YAML parser: PASS
  - git diff --check: PASS
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./...: PASS
  - CGO_ENABLED=1 go build: PASS
  - UI Vitest/build: PASS
  - mobile typecheck: PASS
external_validation_pending:
  - nova execução GitHub Actions com Docker/PostgreSQL/Redis/OTLP
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: revisar diff/segredos, commitar e fazer push; então verificar o novo run do PR sem repetir valores sensíveis
rollback: reverter o commit desta slice na branch feature; não force-push e não alterar main
```


## Follow-up do CI distribuído — nova correção aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 29
base_commit: 9f1afeb3dc147632816ce8eeab583b7cf40fb08e
remote_run: 35737050056
remote_result:
  - Go agentic/server: PASS
  - SBOM: PASS
  - web/mobile: PASS
  - distributed: FAIL antes do teste RLS
root_causes:
  - sintaxe psql `:'app_password'` inválida dentro de `DO $$`
  - cleanup em step separado não herda variáveis shell locais
implemented:
  - criação/alteração de role via psql -c com password hexagonal efêmera
  - grants em chamada psql separada
  - cleanup usa placeholders neutros apenas para interpolação do Compose
proof_pending:
  - nova execução GitHub Actions do job PostgreSQL/RLS/Redis/OTLP
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: executar gates locais, commit/push e acompanhar novo run; não repetir valores sensíveis do CI
```


## Publicação e validação remota do CI distribuído — 2026-09-22

```yaml
state: FIXING
iteration: 29
commit: 15e8ae6021c5dee3a65917025a1a36aa1c0031d0
branch: feat/manus-parity-omniroute
remote_run: 35737772235
remote_result: PASS
jobs:
  - Go agentic/server + Browser Operator: PASS
  - PostgreSQL RLS + Redis DLQ + OTLP: PASS
  - Web/mobile quality: PASS
  - SBOM: PASS
local_result: PASS
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: selecionar o próximo P0 interno; manter PR aberto e sem merge automático em main
```


## Slice P1 central CapabilityPolicy — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 30
base_commit: 0f4409e090aa27d39ff9a5af58c7483c0fdccaef
scope: deny-by-default capability enforcement
files:
  - internal/agent/capability_policy.go
  - internal/agent/capability_policy_test.go
  - internal/agent/runtime.go
  - internal/agent/tools.go
  - internal/agent/runtime_test.go
  - app/ui/app/src/lib/agenticClient.ts
  - app/ui/app/src/components/AgenticConsole.tsx
  - scripts/check-class-a-plus-integrity.sh
evidence:
  integrity: PASS
  go_test_all: PASS
  go_vet: PASS
  go_build: PASS
  web_vitest: PASS
  web_build: PASS
  mobile_typecheck: PASS
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: commit/push da slice e confirmação do quality workflow no novo SHA
```


## Publicação da slice P1 central CapabilityPolicy — 2026-09-22

```yaml
state: FIXING
iteration: 30
commit: 705074d5
branch: feat/manus-parity-omniroute
remote: class-a-plus/feat/manus-parity-omniroute
local_evidence: integrity, go test ./..., go vet ./..., go build, web vitest/build, mobile typecheck = PASS
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_slice: observar quality workflow remoto e continuar P0/P1s restantes; sem merge automático em main
```


## Complemento Master V3 incorporado — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 31
input: /home/ubuntu/upload/pasted_content_6.txt
artifact: audit/HARNESS_CAPABILITY_MATRIX.md
references_cataloged: R01-R44 plus G01-G13
mode: documentary triage and architecture decision; no third-party repo executed
invariants: Ollama local-first, existing Runtime, canonical mission contract, deny-by-default capabilities, LOCAL_ONLY egress boundary
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: validate document links/diff, commit/push matrix, then continue internal P0/P1 remediation
```


## Publicação do complemento Master V3 — 2026-09-22

```yaml
state: FIXING
iteration: 31
commit: 4776840d
branch: feat/manus-parity-omniroute
remote: class-a-plus/feat/manus-parity-omniroute
artifact: audit/HARNESS_CAPABILITY_MATRIX.md
coverage: 44 product references + 13 explicitly provided repositories
validation: integrity PASS; document/link existence PASS; no third-party repository executed
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: concluir diagnóstico dos gates upstream Go/race e continuar P0/P1 internal hardening; sem merge automático em main
```


## Correção dos upstream Go races — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 32
root_cause:
  - research test shared request counter
  - MCP stderr bytes.Buffer promoted ReadFrom bypassed synchronization
files:
  - internal/agent/research_test.go
  - internal/agent/mcp.go
local_evidence:
  focused_race: PASS
  go_test_all: PASS
  go_race_all: PASS
  go_vet: PASS
  go_build: PASS
  integrity: PASS
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: commit/push race fix and verify upstream workflow test on new SHA
```


## Publicação da correção dos upstream Go races — 2026-09-22

```yaml
state: VALIDATING_RELEASE
iteration: 32
commit: 11f6c940
branch: feat/manus-parity-omniroute
remote: class-a-plus/feat/manus-parity-omniroute
local_evidence: go test, go test -race, go vet, go build, integrity = PASS
previous_failure: upstream test/race Ubuntu on fd3abc60
root_causes_fixed: research fixture counter; MCP stderr promoted ReadFrom race
next_action: confirm upstream test/race and agentic quality workflows on this SHA
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
```


## Quality agentic verde; upstream test pendente — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 33
quality_run: 35749351291
quality_sha: 428a99bd9d98dc4dad5b9310b8cc1689f21bac0f
quality_result: PASS
quality_jobs: Go/server; PostgreSQL RLS+Redis DLQ+OTLP; Web/Mobile; SBOM
upstream_test_current_head: NOT_RUN
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: publish this evidence-only commit, then verify test.yaml on the current PR head
```


## Publicação da evidência quality e bloqueio de sincronização upstream — 2026-09-22

```yaml
state: FIXING
iteration: 33
commit: c184c472
branch: feat/manus-parity-omniroute
remote: class-a-plus/feat/manus-parity-omniroute
remote_runs:
  - class-a-plus-integrity: 35749696520 PASS
  - dz23-agentic-quality: 35749696581 PASS
upstream_test_current_head: NOT_RUN
upstream_observation: test.yaml has pull_request-only trigger; GitHub PR API remained at 428a99bd while branch ref is c184c472; no current-head test run exists
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: continue next internal P0 slice; re-check upstream trigger later without treating historical queued runs as evidence
```


## Slice P0 bootstrap MCP/Remote MCP — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 34
files:
  - server/agent_mcp_bootstrap_test.go
  - scripts/check-class-a-plus-integrity.sh
scope: real server bootstrap tests for OLLAMA_AGENT_MCP and OLLAMA_AGENT_REMOTE_MCP
local_evidence: focused normal/race PASS; integrity PASS
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: run full Go/server gates, commit/push, then verify quality workflow
```


## Publicação do slice P0 bootstrap MCP/Remote MCP — 2026-09-22

```yaml
state: VALIDATING_RELEASE
iteration: 34
commit: 1db3911e
branch: feat/manus-parity-omniroute
remote: class-a-plus/feat/manus-parity-omniroute
local_evidence: focused normal/race; integrity; go test ./...; go vet ./...; go build = PASS
remote_evidence: pending for this SHA
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: confirm agentic quality/integrity on this SHA and continue transport/security P0s
```


## Validação remota do slice MCP bootstrap — 2026-09-22

```yaml
state: FIXING
iteration: 34
commit: 00f7c75c
remote_runs:
  - class-a-plus-integrity: 35750321534 PASS
  - dz23-agentic-quality: 35750321486 PASS
  - jobs: Go/server; PostgreSQL RLS+Redis DLQ+OTLP; Web/Mobile; SBOM
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: continue MCP transport audit/correlation and remaining P0/P1s; no automatic merge
```


## Slice P1 MCP correlation — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 35
files:
  - internal/agent/mcp.go
  - internal/agent/mcp_test.go
  - scripts/check-class-a-plus-integrity.sh
scope: ignore JSON-RPC notifications and enforce correlated response id
local_evidence: focused normal/race PASS
remote_evidence: pending
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: full gates, commit/push, remote verification, then continue MCP audit trail/session work
```


## Publicação da slice P1 MCP correlation — 2026-09-22

```yaml
state: VALIDATING_RELEASE
iteration: 35
commit: 9eb7a4c9
branch: feat/manus-parity-omniroute
remote: class-a-plus/feat/manus-parity-omniroute
local_evidence: focused normal/race; integrity; go test ./...; vet; build = PASS
remote_evidence: pending for this SHA
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: verify integrity/quality remote, then continue audit trail and Remote MCP session work
```


## Correção do Browser Operator upstream CI — aguardando publicação — 2026-09-22

```yaml
state: RELEASING
iteration: 36
base_commit: e8017591
root_cause: upstream test/race omitted Python Playwright; the initial pin was unpublished and failed with No matching distribution and ModuleNotFoundError
files:
  - .github/workflows/test.yaml
  - internal/agent/browser.go
  - scripts/check-class-a-plus-integrity.sh
local_evidence: integrity; YAML; focused Browser Operator normal/race; local go test and agent/server race = PASS
remote_evidence: pending; prior run 35755046119 cancelled after queued custom linux/windows matrix was confirmed unavailable
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: verify remote test/race with Playwright 1.63.0; native GPU matrix is manual opt-in and requires configured runners; continue remaining P0/P1 audit slices
```


## Validação remota do hardening de CI multiplataforma — 2026-09-22

```yaml
state: FIXING
iteration: 37
commits:
  - 5db7261e: upstream lint/test OAuth response cleanup
  - 86a2706b: portable Browser Operator executable discovery and full lint cleanup
  - 849781af: Darwin companion gofumpt cleanup
  - de0e8677: Windows platform lint/build cleanup
branch: feat/manus-parity-omniroute
remote: class-a-plus/feat/manus-parity-omniroute
head_sha: de0e86772e96372789c10d924eb5738f8808821b
local_evidence:
  - integrity guard, YAML parse, git diff --check: PASS
  - golangci-lint v2.13.2: 0 issues
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - CGO_ENABLED=1 go test -race ./... -count=1: PASS
  - CGO_ENABLED=1 go vet ./... and go build: PASS
  - GOOS=windows affected-package lint and test compilation: PASS
remote_evidence:
  - class-a-plus-integrity PR run 35769597628: PASS
  - dz23-agentic-quality PR run 35769597363: PASS (Go/server, PostgreSQL RLS + Redis DLQ + OTLP, Web/Mobile, SBOM)
  - dz23-multi-provider PR run 35769597578: PASS
  - upstream test PR run 35769597404: PASS (Linux, macOS and Windows test; Linux/macOS race; patches; go_mod_tidy)
  - push duplicate integrity/agentic runs 35769592895/35769592096: PASS
pr_checks: 21 successful, 3 skipped, 0 failing, 0 pending
warnings: non-blocking GitHub Actions Node 20 and ubuntu-latest migration notices
native_matrix: skipped on pull_request; available only via workflow_dispatch with run_native_matrix=true and compatible operator runners
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
blockers: strong sandbox/process isolation, auth/session/CSRF/IdP lifecycle, OAuth revocation, external provider/deploy/media contracts, physical device/mobile/desktop validation, signing/provenance and store/app review remain open
next_action: continue the next internal P0/P1 slice; do not merge main automatically
```


## Hardening P0/P1 pós-CI — 2026-09-22

```yaml
state: FIXING
iteration: 38
branch: feat/manus-parity-omniroute
remote: class-a-plus/feat/manus-parity-omniroute
head_sha: 935fb27348842fe40d2fc7aa2ec5b85323ee6d85
commits:
  - 8dc74980: UI mantém sessão em memória diante de 403; somente 401 invalida credencial
  - e9905206: contrato do planner usa api.ChatResponseFunc e compila com o cliente Ollama real
  - b5d2befe: ledger persistido por digest torna spend, conversão de afiliado e métrica social idempotentes
  - ed265a5f: terminal allowlisted rejeita flags/path escapes e mantém processo/grupo encerrável
  - 935fb273: logout agentic revoga bearer server-side e limpa sessão local
local_evidence:
  - internal/agent e server tests/vet: PASS
  - terminal/sandbox focused normal/race e Company/auth regressions: PASS
  - UI agent session Vitest (5 tests), tsc -b e Vite build: PASS
  - git diff --check: PASS
remote_evidence:
  - class-a-plus-integrity push 35784205466: PASS
  - dz23-agentic-quality push 35784205382: PASS
  - class-a-plus-integrity PR 35784211426: PASS
  - dz23-agentic-quality PR 35784211429: PASS
  - dz23-multi-provider PR 35784211481: PASS
  - upstream test PR 35784211483: PASS (Linux, macOS e Windows; race Linux/macOS; patches e go_mod_tidy)
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
native_matrix: não executada; permanece workflow_dispatch + run_native_matrix=true com runners compatíveis do operador
blockers: sandbox/process isolation forte, auth/session/CSRF/IdP distribuído, OAuth lifecycle/revocation externo, providers/deploy/media reais, devices físicos, signing/provenance, stores/app review e homologação externa
next_action: continuar slices P0/P1 independentes; manter PR aberto para revisão; não fazer merge automático em main
```


## Hardening sandbox/auth/mobile/release — 2026-09-22

```yaml
state: FIXING
iteration: 39
branch: feat/manus-parity-omniroute
remote: class-a-plus/feat/manus-parity-omniroute
head_sha: a7bc82f5df416b66a71df819a4969604962be0b8
commits:
  - 64b755c8: OAuth refresh/revocation lifecycle tenant-scoped, com CAS e endpoint externo opcional
  - 0459532a: sandbox strict Linux opt-in com cgroup v2 delegado, namespaces, no-new-privs e seccomp amd64/arm64
  - e931fcfc: Origin allowlist/CSRF guard para mutations agentic e safe config de sandbox
  - a12a15e4: cache de missão, outbox offline e push namespaced por servidor/organização; preview web mobile
  - d1058209: SBOM CycloneDX, release metadata e checksum verification no workflow de release
  - a7bc82f5: cgroup.kill no timeout do sandbox strict
local_evidence:
  - CGO_ENABLED=1 go test ./... -count=1: PASS
  - go vet, go build, integrity, YAML e diff check: PASS
  - agent/server auth, sandbox e Company regressions: PASS
  - mobile typecheck e Expo web export: PASS
  - checksum manifest smoke: PASS
remote_evidence:
  - class-a-plus-integrity push 35793674459: PASS
  - class-a-plus-integrity PR 35793679458: PASS
  - dz23-agentic-quality PR 35793679479: IN_PROGRESS no momento do checkpoint
  - dz23-multi-provider PR 35793679448: IN_PROGRESS no momento do checkpoint
  - upstream test PR 35793679567: QUEUED no momento do checkpoint
  - release workflow não foi executado: exige tag e ambiente/credenciais do operador
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
limits:
  - strict sandbox é opt-in e depende de cgroup v2 delegado no host; nenhum host externo foi homologado nesta rodada
  - default best-effort permanece explicitamente diferente de isolamento forte
  - npm audit mobile reportou 18 vulnerabilidades de produção (11 moderate, 7 high); não aplicar npm audit fix --force sem triagem
  - attestation GitHub é condicional a OLLAMA_ENABLE_ATTESTATIONS=true; signing, instaladores, lojas e rollback real continuam não executados
  - matriz GPU/nativa permanece workflow_dispatch + run_native_matrix=true e exige runners compatíveis
blockers: auth/IdP distribuído, OAuth com providers reais, egress/integrations/deploy/media externos, testes físicos, push remoto, signing/provenance efetiva, dependências mobile, stores/app review e homologação externa
next_action: concluir evidência CI do head, triage de vulnerabilidades mobile e continuar P0/P1 independentes; manter PR aberto e não fazer merge automático em main
```


## Follow-up mobile dependency audit — 2026-09-22

A triagem do `npm audit --omit=dev` mostrou que os 18 achados vinham de transitivos do Expo 53/React Native 0.79: `image-size`, `postcss` e `uuid`. Em vez de executar `npm audit fix --force` e migrar majors sem validação, foram adicionados overrides mínimos (`image-size@2.0.4`, `postcss@8.5.28`, `uuid@11.1.1`). Após resolver o lock, o typecheck, Expo web export e `npm audit --omit=dev` passaram; o audit agora reporta `0` vulnerabilidades de produção. A migração Expo/React Native major continua uma tarefa separada que exige testes Android/iOS físicos.


## CI normal verde após mobile audit e installs reproduzíveis — 2026-09-22

```yaml
state: FIXING
iteration: 40
branch: feat/manus-parity-omniroute
head_sha: e6e0632ba5ff0495ed4b061221c90696478b3d67
commits:
  - 482bdd39: overrides transitivos mobile e audit de produção zero
  - ddbe2fab: integrity guard para sandbox, mobile e release contracts
  - e6e0632b: npm ci nos gates agentic/release web e mobile
local_evidence:
  - npm ci mobile/web, mobile typecheck e web tsc: PASS
  - npm audit --omit=dev: 0 vulnerabilities
  - integrity, YAML e diff check: PASS
remote_evidence:
  - upstream test PR 35794443450: PASS; Linux, macOS, Windows, race Linux/macOS, patches e go_mod_tidy
  - class-a-plus-integrity PR 35794443307: PASS
  - dz23-multi-provider PR 35794443300: PASS
  - dz23-agentic-quality PR 35794443501: PASS; Go/server, PostgreSQL RLS + Redis DLQ + OTLP, Web/Mobile e SBOM
pr_checks: 21 successful, 3 skipped, 0 failing, 0 pending
native_matrix: não executada; workflow_dispatch + run_native_matrix=true, runners compatíveis do operador
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
blockers: host sandbox/AppArmor/SELinux e isolamento forte homologado, auth/IdP/OAuth externos, providers/deploy/media/marketplaces, devices físicos, push remoto, signing/provenance efetiva, rollback, stores/app review e homologação externa
next_action: continuar P0/P1 independentes e manter PR #1 aberto; não fazer merge automático em main
```


## V5 — provider/catalog/egress e CI multiplataforma verde — 2026-09-22

```yaml
state: FIXING
iteration: 41
branch: feat/manus-parity-omniroute
remote: class-a-plus/feat/manus-parity-omniroute
head_sha: 0f95b6a1969462fb309e00e29813e8136d15f757
commits:
  - 2237a802: planner efetivo por provider/modelo configurado, sem substituição silenciosa
  - 2439ea51: data root durável separado do workspace de execução
  - 1bc64191: pinagem do IP aprovado no Remote MCP após resolução DNS
  - bff02ef3: estados truthful de connector e credential_configured booleano sem secrets
  - b3c364a3: seletor de provider/modelo derivado do catálogo real
  - 6d8b8710: integrity guard alinhado ao seletor dinâmico
  - 93a0115f: remoção do helper OllamaPlanner.fallback não utilizado
  - 0f95b6a1: cache npm isolado por runner e fail-fast=false nas matrizes test/race
local_evidence:
  - Go completo em sequência desta rodada: CGO_ENABLED=1 go test ./... -count=1, go vet ./... e go build ./...: PASS
  - golangci-lint v2.13.2, testes agent/server e integrity após o cleanup do planner: PASS
  - UI Vitest, tsc -b e Vite build: PASS
  - mobile typecheck e npm audit --omit=dev: PASS; 0 vulnerabilidades
  - YAML de workflows, npm ci com cache isolado e git diff --check: PASS
remote_evidence:
  - class-a-plus-integrity PR run 35804229189: PASS
  - dz23-agentic-quality PR run 35804229301: PASS (Go/server, PostgreSQL RLS + Redis DLQ + OTLP, Web/Mobile e SBOM)
  - dz23-multi-provider PR run 35804229204: PASS
  - upstream test PR run 35804229207: PASS (Linux, macOS e Windows; race Linux/macOS; go_mod_tidy e patches)
  - PR #1: 21 checks successful, 3 skipped, 0 failing, 0 pending
ci_diagnosis:
  - head 6d8b8710 falhou no upstream somente por helper planner não utilizado; removido em 93a0115f
  - head 93a0115f teve falha transitória EEXIST/ENOENT do cache global npm no Windows e cancelamento fail-fast dos irmãos; 0f95b6a1 isolou cache por runner e desabilitou cancelamento em cascata
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
limits:
  - catálogo de connectors, MCPs, skills e providers não equivale a contas OAuth ativas ou plugins externos homologados
  - Woovi/OpenPix, fiscal/NF-e, Composio, marketplaces, social commerce, deploy, mídia e providers externos exigem credenciais, aprovação, smoke reversível e evidência do operador
  - sandbox strict depende de cgroup v2 delegado e host compatível; AppArmor/SELinux e homologação física continuam externos
  - matriz GPU/nativa permanece workflow_dispatch + run_native_matrix=true; não foi executada
  - auth/IdP distribuído, dispositivos físicos, push remoto, signing/provenance efetiva, rollback, stores e app review permanecem abertos
next_action: continuar P0/P1 internos independentes; manter PR #1 aberto para revisão e não fazer merge automático em main
```


## Verificação do checkpoint documental — 2026-09-22

```yaml
state: FIXING
iteration: 42
branch: feat/manus-parity-omniroute
head_sha: b6a814d596af495a304203496451cfb36883053d
functional_code_head: 0f95b6a1969462fb309e00e29813e8136d15f757
delta: documentation-only checkpoint for V5 evidence and connector/API contracts
remote_evidence:
  - upstream test PR 35804952089: PASS
  - class-a-plus-integrity PR 35804952127: PASS
  - dz23-agentic-quality PR 35804952072: PASS
  - dz23-multi-provider PR 35804952157: PASS
pr_checks: 21 successful, 3 skipped, 0 failing, 0 pending
classification: preview/local RC em hardening; NÃO final; NÃO production-ready
next_action: manter PR #1 aberto; continuar somente slices P0/P1 independentes e validações externas autorizadas
```


## Fechamento do head portátil pós-V5 — 2026-09-22

```yaml
state: CANDIDATE_COMPLETED
iteration: 4
head: 0da6be80
completed:
  - durable_runtime_data_root_and_legacy_state_migration
  - fail_closed_default_planner_and_provider_resolver_wiring
  - tenant_binding_for_missions_projects_and_local_catalog
  - ingestion_workspace_containment_cancelability_and_compressed_byte_budgets
  - macos_workspace_alias_portability_fix
macos_fix:
  observed_failure: os.EvalSymlinks treated macOS /var -> /private/var ancestor alias as a final workspace symlink
  correction: os.Lstat checks only the final workspace entry while descendant containment checks remain symlink-aware
  test_portability: server default runtime test now asserts effective os.UserConfigDir rather than Linux-only XDG semantics
ci:
  integrity_pr: 35808941807 PASS
  agentic_quality_pr: 35808941824 PASS
  multi_provider_pr: 35808941746 PASS
  upstream_test: 35808941810 SUCCESS
  upstream_normal_macos_job: 107015911723 SUCCESS
  upstream_race_macos_job: 107015880499 SUCCESS
  upstream_normal_ubuntu_job: 107015911678 SUCCESS
  upstream_normal_windows_job: 107015911689 SUCCESS
  upstream_race_ubuntu_job: 107015880534 SUCCESS
proofs_local:
  - focused ingestion normal and race tests: PASS
  - focused portable runtime default normal and race tests: PASS
  - complete local gates before portable tweak: PASS
external_blockers_unchanged:
  - strong host-enforced sandbox and multi-platform process/device isolation
  - real OAuth/IdP/provider/account smoke and external connector authorization
  - physical Windows/macOS/Linux/Android/iOS and GPU/native validation
  - signed release, installer/store/app-review and operator deployment credentials
classification: preview/local RC em hardening; NOT final; NOT production-ready
next_action: publish connector lifecycle slice and document current CI separately from its pending checks
```

## Slice P1 de lifecycle durável de connectors — 2026-09-22

```yaml
state: TESTING
iteration: 5
base_commit: 0da6be80
commit: 411335ba6247b16a431c7f10b5daf8a9fcc0e8f4
branch: feat/manus-parity-omniroute
completed:
  - persistent_connector_manifest_under_runtime_data_root
  - atomic_0600_manifest_writes_without_credential_values
  - register_connector_http_endpoint
  - owner_admin_guard_and_server_side_organization_binding
  - strict_unknown_field_and_raw_secret_rejection
  - ID_collision_cross_tenant_regression
  - plugins_ui_form_using_env_names_or_oauth_provider_ids_only
proofs_local:
  - connector persistence, redaction, rollback and tenant collision tests: PASS
  - HTTP admin/member, cross-tenant and response redaction tests: PASS
  - UI Vitest and Vite build: PASS
  - integrity guard and git diff --check: PASS
public_ci_at_checkpoint:
  - SHA 411335ba: 18 PR checks pending at initial query; no final conclusion claimed
remaining:
  - MCP/Remote MCP/skill registration persistence and process isolation are separate slices
  - raw secret values are intentionally unsupported by the API; operator must provision env/OAuth securely
  - external accounts, OAuth consent and provider smoke remain unvalidated
classification: preview/local RC em hardening; NOT final; NOT production-ready
next_action: verify SHA 411335ba CI when available, then audit MCP/Remote MCP registration lifecycle
rollback: revert the connector lifecycle commit on the feature branch; do not force-push or merge main
```


## Slice P1 de lifecycle durável MCP/Remote MCP/skills — 2026-09-22

```yaml
state: TESTING
iteration: 6
base_commit: 137fe6ab
commit: 96fd7edd3440876ed67ae7051cc475ca953d2847
branch: feat/manus-parity-omniroute
completed:
  - persistent_mcp_manifest_at_ollama_agent_store_mcp_json
  - persistent_remote_mcp_manifest_at_ollama_agent_store_remote_mcp_json
  - persistent_skill_manifests_under_context_skills
  - owner_admin_registration_endpoints_for_mcp_remote_mcp_and_skills
  - server_side_organization_binding_and_cross_tenant_collision_rejection
  - strict_json_unknown_field_rejection
  - local_mcp_absolute_regular_executable_validation
  - remote_mcp_https_ssrf_dns_pinning_and_env_only_headers
  - skill_trust_fail_closed_and_server_derived_enabled_state
  - ui_forms_and_types_for_three_registration_flows
proofs_local:
  - go test ./internal/agent ./server: PASS
  - focused normal and race persistence tests: PASS
  - focused normal and race HTTP registration tests: PASS
  - complete local runner: PASS
  - integrity, YAML, go test, vet, build, UI Vitest/typecheck/build, mobile typecheck/audit, diff: PASS
public_ci:
  - 96fd7edd PR checks were newly queued at publication; no remote final conclusion claimed here
semantics:
  - OLLAMA_AGENT_MCP, OLLAMA_AGENT_REMOTE_MCP and OLLAMA_AGENT_CONNECTORS remain explicit static bootstrap modes
  - UI mutations are durable only when the corresponding default DataRoot manager is used
  - static bootstrap managers are intentionally runtime-only for UI mutations
external_blockers_unchanged:
  - real OAuth/account consent, external provider smoke and social/commerce/fiscal homologation
  - physical host/device isolation, Windows/macOS/Linux/Android/iOS and GPU/native validation
  - signing, attestation, installers, stores, app review, deploy and operator credentials
classification: preview/local RC em hardening; NOT final; NOT production-ready
next_action: verify current PR checks once, then review distributed/session/sandbox P1 gaps
```


## Complemento de lifecycle na UI — 2026-09-22

O commit `eb97e6b808f2130071eaf12a85d98c0488d6b83a` completou a superfície de lifecycle no frontend. Connectors, MCP/Remote MCP e skills agora possuem remoção explícita com confirmação, além de habilitar/desabilitar; a mensagem deixa claro que remover o manifest local não revoga a credencial no upstream. O build TypeScript/Vite e os 21 arquivos de teste UI, totalizando 204 testes, passaram. A branch foi publicada e permaneceu limpa após o push.

A classificação não muda: preview/local RC em hardening. CI remoto do head mais recente continua aguardando conclusão; não é declarado verde por inferência do gate local.


## Hardening de parsing estrito de manifestos — 2026-09-22

A auditoria P1 encontrou que os novos loaders persistentes de MCP e Remote MCP aceitavam um primeiro JSON válido mesmo quando havia um segundo valor após ele. O commit `89b4203e9b6815ce9be535edf077fe2231b737e6` agora exige `io.EOF` após o primeiro documento; conteúdo trailing produz erro de bootstrap e não é parcialmente aplicado. Foram adicionadas regressões para `[] {}` nos dois loaders.

Evidências: testes normais e race dos managers persistentes passaram; integrity passou; o gate completo Go passou com `CGO_ENABLED=1 go test ./... -count=1 -timeout=900s`, `go vet ./...`, `go build ./...` e `git diff --check`. Nenhum fallback foi relaxado. O SHA foi publicado na branch de revisão; CI pública do head permanece separada da evidência local até conclusão.


## Harmonização de parsing strict dos manifests persistentes — 2026-09-22

A auditoria cruzada encontrou o mesmo padrão no `NewPersistentConnectorManager`: o decoder aceitava o primeiro array e ignorava conteúdo JSON posterior. O commit `5816c020b83993007d91adee420c5bac4e1d7d3c` adicionou a exigência de EOF ao loader de connectors e uma regressão para `[] {}`. Agora connectors, MCP e Remote MCP usam parsing estrito e falham fechado quando há documento trailing.

Os testes normal e race do connector manager passaram. O gate completo pós-harmonização passou em integrity, `CGO_ENABLED=1 go test ./...`, vet, build e diff. O tree segue publicado na branch pública; a CI remota do head mais recente permanece sujeita ao workflow do GitHub.


## Hardening do bootstrap estático de connectors — 2026-09-22

O commit `f78fa0d6c49dc956f8db71452f70f19390c46fee` substituiu o `json.Unmarshal` do loader `OLLAMA_AGENT_CONNECTORS` por `decodeAgentConfigJSON`. O bootstrap estático agora rejeita campos desconhecidos e JSON trailing, alinhando-se aos manifests persistentes. Testes específicos cobrem os dois casos em normal e race.

O gate completo após a mudança passou em integrity, `CGO_ENABLED=1 go test ./... -count=1 -timeout=900s`, vet, build e diff. A configuração por arquivo de ambiente continua explicitamente estática; nenhuma mutation da UI é apresentada como edição desse arquivo. A CI pública do head mais recente continua separada da evidência local.


## Fechamento do strict decode da família de plugins — 2026-09-22

O commit `8630090062c437c1de1fc3f8e779c3963f57c58a` endureceu `ContextStore.LoadSkillsForOrganization`. Manifestos de skills agora usam `DisallowUnknownFields` e exigem EOF após o documento principal; `trusted` e `enabled` continuam derivados pelo servidor. Regressões cobrem campo desconhecido e documento trailing, além dos testes de persistência e trust fail-closed existentes.

O gate completo pós-mudança passou em integrity, `CGO_ENABLED=1 go test ./... -count=1 -timeout=900s`, vet, build e diff. Com isso, connectors, MCP, Remote MCP e skills têm parsing estrito tanto no bootstrap/persistência aplicável quanto nas regressões. O head de código foi publicado; a documentação deste checkpoint será o próximo commit.


## Hardening P1 de logout local — 2026-09-22

A auditoria de auth encontrou que `POST /api/agent/v1/auth/logout` exigia um Bearer e acessava o AuthStore mesmo quando `auth_required=false`. O commit `0ea1039aba7d50014d7422cd40b69355e8451ed7` torna o endpoint idempotente no modo local: retorna `204 No Content` sem sessão e sem AuthStore. No modo autenticado, a revogação Bearer continua obrigatória e não muda de escopo.

Foi adicionada regressão local, além do teste existente que prova revogação do bearer. Testes normal/race específicos passaram e os gates Go completos passaram em integrity, todos os pacotes, vet, build e diff. O contrato não declara que logout local revoga conta externa; ele apenas encerra a ausência de sessão do runtime local.


## Hardening P1 de Origin/CSRF no modo local — 2026-09-22

A auditoria de auth identificou que `auth_required=false` pulava `agentOriginAllowed` e deixava mutações locais sem a política de origem, embora o serviço pudesse ser acessado por um navegador. O commit `79a1197e2e9e857090c32f22c290c1c457a35207` aplica a mesma verificação no ramo local. Origens loopback padrão (`localhost`, `127.0.0.1` e `0.0.0.0`, HTTP/HTTPS e portas) continuam permitidas; uma origem cross-site recebe `403` em métodos de mutação. Leituras, preflight e requests sem Origin preservam o comportamento compatível.

Foi adicionada regressão de middleware local cross-site, além da matriz de `agentOriginAllowed`. Testes normal/race específicos e gates Go completos passaram em integrity, todos os pacotes, vet, build e diff. O modo local continua sem bearer por desenho, mas agora não trata ausência de autenticação como ausência de política de navegador.


## Hardening P1 de wildcard de Origin — 2026-09-23

A revisão da política de origem encontrou que o matcher anterior tratava qualquer configuração terminada em `*` como prefixo livre. Uma allowlist malformada como `https://trusted.example*` poderia aceitar `https://trusted.example.evil`. O commit `4200607483bcd51f3d77e6a96ba46c5159839360` restringe `scheme://host:*` a porta variável no hostname exato, rejeita userinfo/path/query/fragment e preserva somente os wildcards de esquema explícitos como `app://*` e `file://*`.

Regressões cobrem porta válida, hostname parecido e ausência de porta. Testes normal/race e gates Go completos passaram em integrity, todos os pacotes, vet, build e diff. A política local continua aplicando Origin a mutações sem bearer.


## Portabilidade P1 do sandbox best-effort — 2026-09-23

O commit `b4c4c243c43939b1fbbf81155226acee636be2ba` corrigiu uma lacuna multiplataforma: o caminho comum sempre tentava `unshare`, embora strict fosse Linux-only. O sandbox agora resolve Python/Node conforme o sistema e, em macOS/Windows, usa processo best-effort com timeout, limite de stdout/stderr e kill no cancelamento. O resultado declara `execution_isolation=best-effort-platform-process`, `resource_limits=context-timeout-output-bounded` e `network_isolation=not-enforced`. Strict continua recusado fora de Linux e não há downgrade silencioso.

Testes normais/race do sandbox passaram. O gate completo Go passou em integrity, todos os pacotes, vet, build e diff. Também foram compilados os testes de `internal/agent` para `GOOS=darwin` e `GOOS=windows` com `CGO_ENABLED=0`. Isso é prova de compilação multiplataforma, não homologação física de sistemas, dispositivos ou isolamento host.


## Hardening P1 do decoder JSON HTTP — 2026-09-23

A auditoria encontrou que `decodeJSON` já rejeitava campos desconhecidos, mas aceitava um segundo documento após o primeiro. O commit `f06891fc02397aca33ae4eaa04397911120f7d9c` exige EOF depois do único documento decodificado. Como dezenas de handlers de agentic reutilizam esse helper, a correção cobre connectors, plugins, Company OS, schedules, OAuth actions, missions e demais mutações que passam pelo parser comum.

Foram adicionados testes para segundo objeto, campo desconhecido e whitespace válido. Testes normal/race e gates Go completos passaram em integrity, todos os pacotes, vet, build e diff. A mudança é de parsing de request; autorização, approval e dependências externas continuam sendo verificadas separadamente.


## Limite P1 de bodies JSON agentic — 2026-09-23

O commit `09eb26eb33ec12ffd2b2558db3f4949c45e2e243` adicionou `http.MaxBytesReader` ao helper `decodeJSON`, limitando cada body JSON agentic a 4 MiB antes do parsing. A regra vale transversalmente aos handlers que reutilizam o helper e evita leitura ilimitada em endpoints que antes já tinham campos/EOF estritos.

A regressão cobre body acima do limite, além dos casos de campo desconhecido, documento trailing e whitespace válido. Testes normal/race e gates Go completos passaram em integrity, todos os pacotes, vet, build e diff.


## Fechamento da exceção dev token — 2026-09-23

A revisão transversal do parser descobriu que `POST /api/agent/v1/auth/dev/token` usava `ShouldBindJSON` diretamente. O commit `6d00127af3341c32b3246c963be324db89f0e3e5` migrou o handler para `decodeJSON`, portanto o endpoint também recebe o limite de 4 MiB, rejeita campos desconhecidos e exige EOF. O gate de segurança do endpoint permanece: flag `OLLAMA_AGENT_AUTH_DEV=true` e peer loopback real.

Regressão normal/race confirma rejeição de campo inesperado antes de criar usuário/organização. Gates Go completos passaram em integrity, todos os pacotes, vet, build e diff.


## Hardening P1 do adapter de deploy — 2026-09-23

O commit `58cc1f40e478dba5b1b87dc452e172e86b9b2f42` endureceu o adapter de deployments. O workspace raiz agora rejeita symlink; redirects continuam bloqueados; o client padrão remove proxy, marca explicitamente endpoints loopback e rejeita o IP privado real após a conexão para hosts não-loopback. A configuração HTTP local aceita `localhost`, `127.0.0.1` e `::1`; endpoints externos exigem HTTPS.

Regressões cobrem deploy genérico fixture, root symlink, localhost HTTP e dial para endereço privado. Testes normal/race e gates Go completos passaram em integrity, todos os pacotes, vet, build e diff. Nenhum deploy Vercel, Netlify, AWS, Cloudflare ou outro foi executado; esses estados continuam `configured/available`, não `upstream validated`.


## Hardening P1 de inputs multimídia — 2026-09-23

O commit `f45aae49866c16f5b5be60ccb083715663af2896` endureceu `MediaManager`. `Transcribe` e `AnalyzeImage` agora aceitam somente arquivos regulares dentro do workspace, recusam workspace/input symlink e rejeitam traversal. A transcrição verifica o tamanho antes de abrir e falha acima de 100 MiB, em vez de truncar silenciosamente com `io.CopyN`. Outputs de imagem, vídeo e speech também validam o root antes de gravar.

Regressões cobrem provider TLS fixture, containment, symlink, arquivo grande, redirects, MIME/magic e limites. Testes normal/race e gates Go completos passaram em integrity, todos os pacotes, vet, build e diff. O fixture local não comprova conta, quota ou provider externo conectado.


## Company Growth OS explicitamente sandbox-only — 2026-09-23

O commit `a286c15a8c26eda227a695e42889f2ccde34ecea` adicionou `mode: sandbox` a campaigns, affiliate programs/links e orders. `CompanyGrowthReport` agora expõe `sandbox_only=true`; registros legados são normalizados ao carregar; qualquer modo diferente de sandbox falha com `ErrCompanyGrowthExternalUnavailable` até existir adapter upstream validado.

O fluxo local preserva approval, orçamento, idempotência, inventário e métricas, mas não afirma campanha lançada, venda cobrada, fulfillment contratado ou conversão externa. Testes normal/race e gates Go completos passaram em integrity, todos os pacotes, vet, build e diff.


## Correção P1 de safeConfig para manifests duráveis — 2026-09-23

O commit `fda916253cf408336ce25024ccdb24983d2fe626` corrigiu o endpoint de configuração segura. Antes, `connectors_configured`, `mcp_configured`, `media_configured` e `deployments_configured` dependiam somente das variáveis de bootstrap e podiam reportar `false` depois de um registro persistente pelo lifecycle.

Agora os campos combinam bootstrap estático com managers duráveis do Runtime. O teste registra um connector fixture, verifica `configured=true` e confirma que o endpoint não revela URL. A semântica continua restrita a configuração local; não é prova de credential presence, OAuth account ou upstream smoke. Testes normal/race e gates Go completos passaram.


## Reauditoria README/mídia/deploy — correções R01–R04 e captura observável — 2026-09-23

O snapshot `aee66ee988c99d2ed6da2e8e9fc1009e3c90592f` foi reconciliado com o HEAD antes da alteração. A reprodução temporária no pacote real confirmou R01, R02 e R03: `.agent-media` symlink permitia escrita externa antes de `BuildArtifactManifest`, `.env`/`.git/config` entravam no pacote de deploy e o dialer estabelecia TCP privado antes da recusa. R04 foi confirmado por arquivo esparso de `25 MiB + 1`: `AnalyzeImage` recusava depois de `os.ReadFile`.

O commit `06990ef573129da2e4a272b49435857b9e7f46c4` corrigiu a causa raiz. Outputs de mídia e OCR usam `os.Root`, paths relativos, rejeição de componentes symlink e criação limitada. Inputs de transcrição e visão usam descritor aberto pelo root, orçamento de leitura com cancelamento e detecção de crescimento. O diretório de output é validado antes de provider/ processo externo. O coletor de deploy exclui `.git`, metadados internos, `.env`, chaves, backups, logs e arquivos não regulares antes da leitura; o provider fixture confirmou que conteúdo privado sintético não chegou ao payload. O dialer resolve A/AAAA, rejeita qualquer endereço privado antes do TCP e disca o IP aprovado, preservando hostname da URL para Host/SNI; loopback só é permitido pelo contexto explícito.

As regressões permanentes passaram em normal e `-race`, com testes positivos de output legítimo, filtro de pacote, cancelamento, limite e ausência de TCP proibido. O pacote `internal/agent` também compilou para Darwin e Windows com `go test -c`; a tentativa inicial de `go test` cross-compile foi descartada como erro operacional de execução de binário estrangeiro, não como falha de compilação.

O commit `ec7f52c048c238510ca6dc08212cd2c10535bfe1` tornou `capture-parity-screens.mjs` relativo ao checkout, respeitou `SCREEN_OUTPUT` explícito, rejeitou base URL inválida, aguardou estado observável, executou interação segura por rota, coletou page/console/request/HTTP diagnostics e gravou manifesto com SHA/build/viewport/limitações. O preview distribuível foi servido por bridge local conectado ao backend Ollama real; dez rotas foram capturadas com interação, zero page errors, zero console errors inesperados, zero request failures inesperados e zero HTTP failures inesperados. As respostas 401/404 conhecidas de ausência de conta ou endpoints base sem implementação foram classificadas no manifesto, não escondidas.

A Home fornecida pelo mantenedor ainda segue o caminho documental separado a partir da main. O PR do produto não é base para trocar a imagem da página padrão; `README_MAIN_STATUS=PENDING_DOCS_MERGE` até aprovação e merge do PR documental independente.

Estado: `FIXING`; implementação e publicação dos commits R01–R04 e do capturador concluídas; homologação externa, contas, deploy real, dispositivos, lojas e CI do novo head continuam pendentes ou `BLOCKED_BY_EXTERNAL_DEPENDENCY` conforme a matriz do readiness.


## Hardening da cadeia de dependências runtime da UI — 2026-09-23

O gate local completo iniciado no head `2f5674cf` passou por integrity, YAML, Go test, vet, build e testes/build da UI, mas parou em `app/ui/app/npm audit --omit=dev` com 12 vulnerabilidades runtime. O mobile não chegou a executar nesse runner abortado; a auditoria independente do mobile depois retornou zero vulnerabilidades.

A análise do grafo mostrou que `streamdown@1.4.0` trazia Mermaid 11.12, DOMPurify, lodash-es e uuid em runtime; `@tanstack/react-router-devtools` estava em `dependencies` apesar de não ser importado pela aplicação e trazia `solid-js/seroval`. O commit `eff054c14c38d824654fc44c095b910038bc9726` atualizou Streamdown para `2.6.0`, removeu o devtools não utilizado, declarou Shiki usado diretamente e fixou `mdast-util-to-hast` em `13.2.1` via override. O grafo runtime resultante não contém Mermaid, DOMPurify, lodash-es, seroval ou o devtools removido.

Evidência local pós-correção: Vitest 21 arquivos/204 testes, `npm run build`, `npm ci`/dry-run do lockfile, `npm audit --omit=dev` com zero vulnerabilidades, `apps/mobile-agentic/npm run typecheck` e `npm audit --omit=dev` com zero vulnerabilidades. O CI público do novo SHA ainda precisa concluir; a falha race macOS do head anterior não é reutilizada nem declarada resolvida.


## P1 de approval com motivo explícito na UI — 2026-09-23

A Agentic Console deixou de enviar o texto genérico `Aprovado no Agentic Console` ou `Rejeitado no Agentic Console`. O commit `720bbd9245ab4589a508e2627b976c5f7bbf577b` adicionou um campo por decisão, limita o motivo a 512 caracteres, bloqueia Aprovar/Rejeitar quando o texto está vazio e envia o motivo efetivamente escrito junto do nonce e da decisão. O campo é limpo depois de uma resposta aceita.

A regressão `AgenticConsole.approval.test.tsx` confirma que uma decisão sem motivo permanece bloqueada e que o POST contém `approved`, `nonce` e o texto explícito. No head anterior `f9ae5dd8`, o runner absoluto fechou `FULL_LOCAL_GATES=PASS`: integrity, YAML, `CGO_ENABLED=1 go test ./...`, vet, build Go, Vitest 22 arquivos/205 testes, build UI, audit UI/mobile, typecheck mobile e diff. O novo head `720bbd92` foi publicado e seu CI remoto ainda está em execução.


## Fechamento P1 do limite server-side de approval — 2026-09-23

O commit `92c7bdd220a39e444e824f2022d9399e756306d5` fechou a fronteira de confiança da melhoria de approval: além do `maxLength=512` da UI, `Runtime.DecideApprovalForActorCAS` agora normaliza e rejeita motivos acima de 2048 bytes com erro de input, antes de alterar o approval. A regressão cobre o bypass por cliente direto e a variante race do conjunto de approvals passou.


## Jornada P1 Tel-Agent textual acima do canvas — 2026-09-23

O commit `45393d8ca08599f008ea81598ef1f8d3f75f66bc` adiciona o primeiro vertical funcional da prioridade Tel-Agent definida na continuidade pós-V5. O Company OS agora possui o canal `tel-agent.text`, persistido por tenant nas últimas 100 trocas, com redaction DLP antes da gravação, actor e organization derivados do contexto autenticado e histórico recuperável por `GET /api/agent/v1/companies/:id/tel-agent/history`.

A operação `POST /api/agent/v1/companies/:id/tel-agent` aceita somente `report.read`, `backlog.create` e `campaign.draft`. Leituras são permitidas a membros autenticados do tenant; mutações exigem owner, admin ou operator. `backlog.create` cria uma tarefa local priorizada e `campaign.draft` cria somente rascunho sandbox com approval pendente. Nenhuma operação envia mensagem, publica anúncio, movimenta dinheiro, chama telefonia ou afirma conta externa conectada. A UI do Company OS coloca o canal antes dos demais painéis, com retorno estruturado, histórico e aviso explícito de `telephony: not_configured`.

Evidência local: `go test ./internal/agent -run 'TestCompanyTelAgent'`, `go test ./server -run 'TestCompanyTelAgent'`, variantes `-race`, Vitest completo, build TypeScript/Vite e `Class A+ integrity guard` passaram. O SHA remoto confirmado é `45393d8ca08599f008ea81598ef1f8d3f75f66bc`; no primeiro check público, os workflows ainda estavam `queued`/`in_progress`, portanto a CI desse candidato permanece pendente.

Classificação: **IMPLEMENTADO, TESTADO, PUBLICADO; não homologado externamente**. Telefonia, SIP, SMS, WhatsApp e provedores de voz continuam `BLOCKED_BY_EXTERNAL_DEPENDENCY` até existir conector, consentimento, conta de operador, destino de teste e smoke test reversível autorizado.


## Evidência final local do head Tel-Agent — 2026-09-23

Após a publicação documental, o runner absoluto `FULL_LOCAL_GATES=PASS` foi concluído no head `55dfa325d1f270284169f4d99beb4f539f38bd54`: integrity, YAML, `CGO_ENABLED=1 go test ./... -count=1 -timeout=900s`, `go vet`, build Go, `npm ci`/Vitest/build/audit de produção da UI, `npm ci`/typecheck/audit de produção mobile e `git diff --check` passaram. Avisos de chunks Vite grandes e pacotes npm deprecated não produziram falha nem foram tratados como vulnerabilidades.

A conferência pública do mesmo SHA encontrou SBOM, serviços distribuídos, Web/Mobile, integrity, mudanças, patches e go_mod_tidy concluídos com sucesso; jobs Go e a matriz upstream ainda estavam `in_progress`, sem falha observada naquele instante. Portanto, a prova local é completa, mas a CI remota permanece pendente até seus jobs terminarem.


## P1 de idempotência do Tel-Agent — 2026-09-23

O commit `d67e7a7910d850aa69953ed3be3b6ba40ed57bfa` fecha retries duplicados no canal Tel-Agent. O endpoint aceita `Idempotency-Key` somente pelo header, limita a chave a 128 bytes, deriva fingerprint sem persistir a mensagem original e usa o ledger Company existente. Um replay retorna a mesma exchange/recurso sem nova mutação; o mesmo header com payload diferente falha com conflito `409`; o vínculo de resultado permanece disponível após restart do `CompanyStore`. A UI gera uma chave opaca por operação e a mantém enquanto uma falha pode ser repetida, limpando-a somente após sucesso.

Regressões normais e race de `internal/agent` e `server` passaram, incluindo replay HTTP `200`, conflito HTTP `409` e restart. Depois do commit, o runner absoluto completo também terminou `FULL_LOCAL_GATES=PASS`: integrity, YAML, Go test/vet/build, UI Vitest/build/audit, mobile typecheck/audit e diff. A CI pública do novo SHA iniciou sem falha observada, mas ainda estava em execução; não é considerada totalmente verde.


## P1 de consistência transacional dos ciclos Company — 2026-09-23

O commit `46d6b34c1ace1f5e9065559427701cbd3bcc0258` corrige uma janela em `POST /api/agent/v1/companies/:id/cycles`: antes, o ciclo podia ser persistido como `enabled` antes de o schedule ser criado, deixando uma unidade órfã se a escrita do schedule falhasse; retries também podiam duplicar ciclos e schedules. A criação agora aceita `Idempotency-Key`, vincula o resultado ao ciclo, retorna replay `200` sem nova agenda e responde `409` quando a mesma chave recebe payload divergente. Falhas na criação ou no binding do schedule removem o schedule/ciclo compensatoriamente. `ContextStore.CreateSchedule` também desfaz a entrada em memória quando a escrita atômica falha.

Testes normais e race cobrem replay após restart, conflito de payload, rollback do CompanyStore, rollback HTTP e ausência de schedule duplicado. O runner local absoluto completo terminou `FULL_LOCAL_GATES=PASS` no head funcional. A CI pública do SHA iniciou com jobs queued/in progress e sem falha observada na consulta; ainda não foi chamada de totalmente verde.


## P1 de retry bounded do worker de schedules — 2026-09-23

O commit `86597b827fb247dcc60bbcb0a7c31ee1a0ddc4be` fecha uma perda de tentativa em `Runtime.resumePending`: depois do claim, falhas de `CreateMission` agora ficam registradas no próprio schedule com `failure_count` e código sanitizado `mission_creation_failed`. O worker aplica backoff local de 5 e 10 segundos e desabilita o schedule após três falhas consecutivas; uma execução posterior bem-sucedida limpa o estado. O claim também só avança `NextRunAt` depois da persistência e restaura a cópia anterior se a escrita falhar.

Testes normais e race cobrem planner indisponível, backoff, auto-pause após três falhas, reset após sucesso e rollback de claim em storage quebrado. O runner local absoluto completo passou `FULL_LOCAL_GATES=PASS`. A CI pública do SHA iniciou com jobs queued/in progress e sem falha observada na consulta. Este é um worker local bounded; lease/claim distribuído, múltiplas réplicas e DLQ de schedule continuam dependentes de infraestrutura PostgreSQL/Redis homologada.


## P1 de rollback transacional da fila local — 2026-09-23

O commit `717a7e4f9444c4d77d76422f0b1e70ccfe08a944` corrige uma janela no `JobQueue`: `Claim`, `Ack`, `Nack` e `Replay` alteravam o mapa em memória antes de confirmar `jobs.json`. Se a escrita falhasse, um job podia ficar `running` apenas em memória e desaparecer do fluxo até restart. O helper agora restaura o estado anterior ou remove a inserção quando a persistência atômica falha.

A regressão normal/race cobre falha de persistência durante claim e confirma que o job permanece `pending` com zero tentativas. Os gates locais completos passaram `FULL_LOCAL_GATES=PASS`; a CI pública do novo SHA iniciou com jobs em andamento e ainda não é considerada verde. O queue local continua single-process; lease distribuído, fencing e homologação Redis continuam externos a esta slice.


## P1 de transições de queue condicionadas a running — 2026-09-23

O commit `9d28cbc2b947160ad18e7b26c87a81dc740806f1` restringe `Ack` e `Nack` da fila local e do adapter Redis a jobs no estado `running`. Um job `pending`, `succeeded` ou `dead_letter` não pode ser finalizado ou reenviado por uma chamada fora da execução que o possui. A regressão local cobre as transições inválidas; o adapter Redis foi compilado e permanece dependente de integração Redis para o smoke distribuído.

Integrity, YAML, `CGO_ENABLED=1 go test ./...`, vet, build e race do queue passaram localmente. A CI remota do head ainda está em execução. O worker distribuído continua sem claim lease/fencing homologado em múltiplos processos.


## P1 de compensação multi-comando do RedisQueue — 2026-09-23

O commit `883a17e29b8aa3bd1b1d08de3d75518974711c6e` reduz estados órfãos no adapter Redis. Enqueue remove a chave do job se o `LPUSH` falhar; claim devolve o ID ao pending quando não consegue ler ou persistir a transição; reencaminhamento restaura o snapshot `running` quando o `ZADD` falha; replay restaura o estado anterior quando o `LPUSH` falha. As compensações não são apresentadas como transação Redis atômica.

Integrity, YAML, `CGO_ENABLED=1 go test ./...`, vet, build, race do queue e o teste com build tag de integração passaram. O teste distribuído foi executado sem `OLLAMA_AGENT_TEST_REDIS_URL` e, por isso, não abriu conexão real. A CI pública do SHA iniciou em jobs queued. Permanecem bloqueados por dependência externa o smoke Redis real, lease/fencing e recuperação multi-worker.


## Complemento de moveDue Redis — 2026-09-23

O commit `b1aaebfdb4ad7ad753591402ad47a220fcd67fdd` fecha o caso restante de `moveDue`: se o `ZREM` for confirmado mas o `LPUSH pending` falhar, o ID é recolocado no sorted set delayed. Testes normais/race focados e integrity passaram; o smoke Redis real segue bloqueado por ausência de endpoint configurado.


## Correção de screenshot Settings e captura fail-closed — 2026-09-23

A auditoria visual encontrou que `docs/images/screens/settings.png` era um arquivo histórico quase vazio de 8.2 KB, embora a implementação de Settings e a captura atual `class-a-plus-settings.png` estivessem funcionais. O alias foi substituído pela captura funcional, com checksum `89420370b9d3c7317a2c5d7c51bb9f8d4eba914a6d7b7bf829c855c909902e72`. O capturador agora exige conteúdo mínimo em `body`/`main` e rejeita PNG menor que 16 KiB, evitando publicar uma tela branca silenciosamente. Testes de `Settings`, build UI, Node syntax, integrity e YAML passaram. Commit publicado: `0b0710cfe9d646c859aed8c624fb19fe1ac3cd43`; CI remota iniciou com checks pendentes.


## Retomada autônoma — completar Ollama Full sem sobrescrever trabalho herdado — 2026-09-26 21:56 -03

```yaml
mission_id: viqyzCzZsu3R88Zh33feNk-ollama-full-completion
objective: Evoluir o fork em direção às jornadas verificáveis do Ollama Full e fechar o maior próximo gap local sem alegar paridade/produção sem evidência.
state: RECOVERING
rigor: L3 (plataforma com autenticação, isolamento multi-tenant, filesystem, shell e integrações externas)
repository:
  path: /home/ubuntu/ollama-full-recovery
  branch: recovery/ollama-full-snapshot
  head: 8635e30d (origin/main; commit Windows checksum)
  remote: origin -> https://github.com/DZ23-LTDA/hades-ollama.git
  uncommitted_changes: true
  observed_modified_and_untracked_paths: dozens; backend, server, web, docs, packaging and untracked security/snapshot modules
scope_in:
  - preservar integralmente as alterações herdadas e reconciliar os checkpoints com o estado de disco;
  - priorizar jornadas locais de alto valor da árvore de produto, com implementação + teste + execução real;
  - executar gates oficiais/aplicáveis e corrigir falhas pela causa raiz;
  - documentar limitações, dependências externas e prova observada.
scope_out:
  - declarar paridade total com Manus/60+ ferramentas sem matriz e benchmarks observáveis;
  - usar/adivinhar ativo Swole que não esteja presente no checkout;
  - conectar contas/credenciais externas, publicar/deploy produção, assinar artefatos ou testar dispositivos sem dependências/autoridade apropriadas;
  - force-push, reescrever história, limpar/resetar worktree ou mesclar main.
acceptance_criteria:
  - checkpoint corresponde ao branch/HEAD/worktree e distingue histórico de estado atual;
  - pelo menos uma próxima jornada crítica selecionada do roadmap tem fluxo/API, testes e execução real no ambiente local;
  - gates aplicáveis passam no mesmo estado de fonte, com failures e N/A justificados;
  - alterações herdadas não são descartadas nem commitadas sem revisão/secret scan/gates;
  - matriz/readiness não afirma capabilities externas como homologadas;
  - blockers finais listam logo original, serviços/credenciais, CI/assinatura/dispositivos quando ainda faltarem.
gates:
  required: [integrity, gofmt, go_test_all, go_vet, go_build, postgres_redis_integration_if_available, web_typecheck_lint_unit_build, mobile_typecheck_policy, e2e_smoke, secret_scan, final_security_review]
  not_applicable: [external_provider_smoke_without_operator_credentials, signed_release_without_signing_keys, native_physical_device_test_without_devices]
delivery_destination: branch recovery/ollama-full-snapshot; push permitido pelo pedido anterior somente após gates e auditoria; nunca main/merge/release sem novo escopo explícito.
approvals_required: [production deploy/publish, external accounts or paid services, signing credentials, publication beyond authorized feature branch]
rollback_plan: preservar HEAD 8635e30d e nunca executar reset/clean; reverter somente hunks introduzidos neste ciclo com patch revisado; manter logs/teste temporários fora do repo.
current_task: Reconciliar alterações herdadas e executar a primeira auditoria de gates desta retomada.
current_failure: Checkpoint autônomo histórico está em 2026-09-23 e não representa o checkout observado de 2026-09-26; não há processos de teste ativos.
current_strategy: Preservar worktree; evidência mais nova primeiro; escolher fatia repo-first do produto, sem paralelismo sobre os mesmos arquivos.
plan:
  - id: recover-and-freeze
    status: in_progress
    action: inventory full diff and checkpoint current state; no destructive git operations
  - id: verify-inherited-backend
    status: pending
    action: integrity, gofmt, focused tests, then full Go tests/vet/build
  - id: verify-product-surfaces
    status: pending
    action: web/mobile E2E and accessibility/security-relevant gates after backend fixes
  - id: advance-repo-first
    status: pending
    action: implement the smallest missing project-test-runner/worktree flow with safe isolation and regression proof
  - id: final-review-and-release
    status: pending
    action: independent review, secret scan, same-tree gates, then branch-only publication if still authorized and safe
blockers:
  - item: original Swole brand asset
    status: BLOCKED_BY_EXTERNAL_DEPENDENCY
    evidence: no Swole-specific logo file found in checkout's bounded asset search
    needed: exact source image/vector from maintainer
    impact: brand/logo replacement cannot be truthfully marked complete
  - item: external service and device validation
    status: BLOCKED_BY_EXTERNAL_DEPENDENCY
    evidence: provider accounts, credentials, signing keys and physical devices are not present in the project context
    needed: operator-configured authorized test accounts/devices/signing materials
    impact: no claim of upstream OAuth/provider/deploy/store/physical smoke
hypotheses:
  - The highest-value local gap remains repo-first coding completion: isolated branch/worktree, project test runner, repair loop and E2E.
decisions:
  - decision: rigor L3, but explicitly not regulated-finance L4
    reason: multi-tenant agent platform executes filesystem/shell actions and holds integrations; high security/privacy bar is warranted without claiming a regulated deployment.
  - decision: preserve all pre-existing and untracked work; do not rebase, reset, clean, or squash it
    reason: checkout contains dozens of user-requested changes from the earlier task, and ownership/status of each file cannot be safely inferred by destructive cleanup.
  - decision: do not push until a single frozen tree passes release gates and secret review
    reason: branch-only publication is previously authorized, but current tree is broad/uncommitted and not yet verified end-to-end.
last_progress_at: 2026-09-26 21:56 -03
next_action: Run integrity and focused Go regressions on the current checkout; reconcile any failure before full gates.
resume_instructions: Read this section and the latest security checkpoint; verify HEAD/worktree again; preserve every inherited edit; continue from first pending plan item and update this file after significant progress.
```


## Fase 12 — auditoria E2E adversarial — retomada 2026-09-30 16:14 -03
state: AUDITING
head: 9b188b92
branch: recovery/ollama-full-snapshot
scope: somente auditoria, relatório, evidências e reclassificação documental; nenhuma alteração de comportamento
known_observations:
  - backend/frontend locais ativos; frontend / retorna 200; endpoint /api/agent/v1/status retorna 404 (endpoint não é health check válido)
  - scan encontrou clientes HTTP crus em vários caminhos server/agent; approved bool ainda existe em structs de domínio company_growth/company_social
next_action: executar gates completos, adversarial H1-H5 e E2E real; consolidar findings sem corrigir código


## Fase 12 — fechamento da auditoria — 2026-09-30 16:22 -03
state: AUDIT_COMPLETE
head: 9b188b92
report: audit/E2E_AUDIT_2026-09-30.md
result: local gates PASS; P1 Studio visual desktop HTTP 400; P1 server egress/DLP coverage incomplete; routes /whatsapp and /models were invalid script paths and were retested through /connectors and /c/new
next_action: documentation-only commit, push branch, verify both GitHub workflows on resulting docs commit

## FASE 11 — retomada no sandbox atual (2026-09-30 17:26 -03)
```yaml
state: IMPLEMENTING
mission: Importar projeto inteiro por GitHub ou ZIP grande, com isolamento, ingestão e UI real
branch: recovery/ollama-full-snapshot
head_verified: 2b6ff7d7
facts:
  - UploadManager já é chunked/resumable e org-scoped, ContextStore/DocumentIngestor persistem projetos e memórias
  - CreateGitWorktree já cria branch/worktree isolado para reuso
  - importador seguro foi implementado no runtime, com egress, limites ZIP e sem execução do código importado
next_action: adicionar testes focados, executar gates disponíveis e corrigir contratos
rollback: preservar commit 2b6ff7d7; não usar reset/clean/force-push
```


## Checkpoint vigente — Auditoria ampla e correções — 2026-09-30 19:16 -03
```yaml
state: AUDITING
mission: auditar e corrigir de ponta a ponta o projeto sem push remoto
branch: recovery/ollama-full-snapshot
head: 943339931bfc9fac995b12c77313264ce5b4e08b
worktree: clean_before_audit
remote: origin/recovery/ollama-full-snapshot
scope:
  - backend Go/runtime/API e fronteiras de tenant, approvals, egress, uploads, execução e integrações
  - frontend React/TypeScript, acessibilidade, responsividade e fluxos reais Playwright
  - native/CMake, mobile, Docker, CI, scripts, documentação e coerência de claims
constraints:
  - preservar trabalho preexistente; sem reset/clean/rebase/merge/exclusões destrutivas
  - sem push, PR, publicação, deploy, serviços pagos ou credenciais reais
acceptance:
  - reproduzir achados confirmados, corrigir causa raiz quando possível e adicionar regressões
  - executar gates reais e registrar antes/depois, evidências e bloqueios externos
  - criar docs/auditoria.md e atualizar MEMORIA.md antes de encerrar
baseline_observed:
  - UI preview e backend local estavam ativos; serão encerrados ao final se iniciados/reutilizados pela auditoria
  - último commit é documentação UI-2; não há alterações não commitadas
next_action: executar linha de base de testes/build/vet/integridade e mapear falhas reais
rollback: preservar HEAD 94333993; não descartar alterações de terceiros
```

## Retomada após CI — 2026-09-30 22:02 -03
state: RETESTING
commit: edff7c44
finding: dz23-agentic-quality falhou apenas no Windows em TestLargeUploadProjectImportStreamsZipAndCreatesIsolatedWorktree; o erro vinha da comparação literal do caminho retornado por `git rev-parse --show-toplevel` com o caminho do fixture.
fix: project_import.go agora normaliza separadores, absolutiza e compara caminhos sem diferenciar caixa no Windows; a rejeição de `.git` permanece ativa.
evidence: go test -run 'Import|GitHubImport|LargeUpload|ZipStream|Clone|Ingest' ./internal/agent ./server PASS; CGO_ENABLED=0 go build ./... PASS; GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./... PASS.
next: commit, push e acompanhar novamente class-a-plus-integrity e dz23-agentic-quality no mesmo SHA.

## Segunda correção do CI Windows — 2026-09-30 22:06 -03
state: RETESTING
finding: a comparação de caminho ainda falhava no runner Windows apesar da normalização, pois o formato emitido por `git rev-parse --show-toplevel` varia no shell Git.
fix: validação agora usa `rev-parse --is-inside-work-tree` e `--show-prefix` vazio, sem depender de drive letters/separadores; o diretório verificado continua sendo o root criado pelo importador.
evidence: go test ./internal/agent ./server PASS; CGO_ENABLED=0 go build ./... PASS; GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./... PASS.
next: commit, push e acompanhar os dois workflows novamente.

## CI final verde — 2026-09-30 22:15 -03
state: COMPLETED
commit: da49fd89
ci: class-a-plus-integrity=success (run 36799588232); dz23-agentic-quality=success (run 36799588701), ambos no mesmo SHA.
final_evidence: testes Go críticos, build nativo, CGO0 e GOOS=windows passaram localmente; o job Windows remoto passou após a validação portátil do topo Git.
next: nenhuma ação pendente desta auditoria ampla; novas mudanças devem iniciar novo checkpoint.


## AUD-FIX-1 — retomada 2026-09-30 23:37 -03
```yaml
state: FIXING
branch: recovery/ollama-full-snapshot
base_commit: c393b0bc
scope: logout real, perfil POST /api/me, download Library e verificador de contratos
acceptance:
  - POST /api/signout antes de fechar modal e limpar sessão
  - sidebar/Settings usam POST /api/me via fetchUser
  - Library usa GET /api/agent/v1/missions/:id/artifacts/:artifact_id
  - verify-contracts considera método, parâmetros por posição, href/wrappers e avisa rotas órfãs
  - testes frontend, build Go, contratos, integrity e Playwright desktop/mobile
next_action: aplicar correções e executar gates; não alterar escopo
rollback: preservar c393b0bc; sem reset/clean/force-push
```


## AUD-FIX-1 — checkpoint local
state: COMPLETED
commit_base: c393b0bc
ci_commit: edd226e2
changes: AppSidebar/Settings profile+logout, Library artifact route, contract scanner, frontend regression tests
gates: frontend, Go, contracts and integrity PASS
next_step: nenhum dentro do escopo AUD-FIX-1; repetir evidência live autenticada quando houver sessão Bearer e artifact real


## AUD-FIX-2 — 2026-10-01 03:02 UTC
state: IMPLEMENTED_LOCAL_CI_PENDING
branch: recovery/ollama-full-snapshot
base: 331759eb
finding: `/connect` quebrava com TypeError ao aplicar `.find()`/`.filter()` sobre o envelope de integrações.
fix: normalização em `api.ts`, guarda de runtime em `Onboarding.tsx` e regressão de payload inválido em `Onboarding.test.tsx`.
evidence: Playwright desktop/mobile sem console ou HTTP errors e sem tela de crash; Vitest 38 arquivos/260 testes; tsc/lint/build/contracts/integrity/Go PASS.
next_action: commit, push e acompanhar `class-a-plus-integrity` e `dz23-agentic-quality` no mesmo SHA.

## Retomada — missão noturna T2 — 2026-10-01 03:13 UTC
```yaml
state: COMPLETED_CI_GREEN
mission: fechar achados de auditoria em ordem, sem avançar antes de CI verde
branch: recovery/ollama-full-snapshot
base_commit: 1214a90b671beaca8cab6e309e21d033002a41f0
completed:
  - T1 /connect: correção de normalização de integrações e evidência Playwright desktop/mobile
  - T1 CI: class-a-plus-integrity e dz23-agentic-quality success no mesmo SHA
current:
  - T2 Endpoint: remover ações falsas e autorização local simulada; recursos remotos honestamente NOT_CONFIGURED
acceptance_t2:
  - sem alert() nem mutação local ao clicar ações remotas
  - controles remotos disabled com estado NOT_CONFIGURED acessível
  - matriz de paridade Endpoint PARCIAL, sem PASS indevido
  - gates locais e Playwright desktop/mobile
next_action: T2 encerrada; iniciar T3 somente após decisão/escopo explícito
ci:
  class_a_plus_integrity: {id: 36809795958, conclusion: success}
  dz23_agentic_quality: {id: 36809796036, conclusion: success}
  sha: 2b13a2ce42ef09b0a48908b3b8b02af3eb6c24a2
rollback: preservar 1214a90b; sem reset/clean/force-push
```


## Missão noturna T3–T7 — 2026-10-01 06:30 UTC
state: IMPLEMENTED_LOCAL_CI_PENDING
completed:
  - T3 Criações: status NOT_EXECUTED real, sem PASS falso
  - T4 Notificações: painel vazio honesto, sem alert fictício
  - T5 i18n: controles de configuração traduzidos para pt-BR
  - T6 Automações: formulário intervalo/webhook com segredo referenciado
  - T7 Código morto: kinds duplicados e mensagem fixa removidos
evidence: docs/evidencias/browser-console-audit-night-t3-t7.json; screen-night-t3-t7-*.png
gates: Go build/test + CGO0 PASS; tsc/lint/Vitest/build PASS; contracts/integrity PASS
limitation: Playwright sem Bearer registrou 401 em scheduled/settings; não declarar console limpo nessas telas
next_action: T8 varredura final honesta, gates completos, commit/push e CI


## Missão noturna T3–T8 — checkpoint T8 — 2026-10-01 06:34 UTC
state: COMPLETED_LOCAL_CI_PENDING
branch: recovery/ollama-full-snapshot
completed: [T3, T4, T5, T6, T7, T8]
evidence: docs/evidencias/screen-night-t3-t7-*.png, docs/evidencias/browser-console-audit-night-t3-t7.json
gates: go build, CGO_ENABLED=0 build, go test, go vet, tsc, lint, Vitest, UI build, contracts, integrity PASS
limitations: protected scheduled/settings requests without Bearer recorded 401; external credentials, persistent notifications, deploy/live creations, physical mobile, signed installer remain outside proof
next_action: commit/push and wait for class-a-plus-integrity plus dz23-agentic-quality on same SHA
rollback: preserve ea6d8fd2; no force-push

## LINT-DEBT — retomada 2026-10-01 08:51 -03
state: PLANNING
mission: tornar .github/workflows/test.yaml verde na matriz Ubuntu/macOS/Windows e race, corrigindo a dívida golangci-lint sem alterar comportamento
branch: recovery/ollama-full-snapshot
base_commit: b573bf251d1526b86ce03763101bafd5b4492e53
scope:
  - .golangci.yaml
  - arquivos Go apontados por golangci-lint/gofmt/gofumpt
  - MEMORIA.md e este checkpoint
out_of_scope:
  - mudanças de lógica de produto não exigidas pelos linters
  - desativar linters, reduzir assertions ou alterar workflow para esconder falhas
acceptance:
  - golangci-lint run ./... sem findings
  - gofmt -l . e gofumpt -l . vazios
  - go build ./...
  - CGO_ENABLED=0 go build ./...
  - go test ./...
  - go test -race ./internal/agent ./server
  - workflow test.yaml verde em Ubuntu/macOS/Windows e race no SHA final
next_step: instalar a mesma versão do golangci-lint da action e capturar a linha de base

## Checkpoint — S1/S2 e E2E de shell validados — 2026-10-01 21:02 UTC
```yaml
state: LOCAL_HARDENING_VALIDATED_CI_PENDING
completed:
  - S1: política egress compartilhada entre agent e multillm, com testes negativos
  - S2: MCP local tenant-scoped bloqueado sem sandbox forte; sem fallback inseguro
  - E2E shell: 10 telas desktop/mobile, 0 HTTP errors, 0 console errors
remaining:
  - gates completos locais e CI remoto
  - sandbox MCP forte ainda não disponível/configurado
  - credenciais externas, IdP, deploy, assinatura e PostgreSQL production cutover permanecem externos/bloqueados
next_action: executar gates completos, atualizar auditoria e publicar se tudo estiver verde
```


## Fechamento de CI e dependências — 2026-10-02 UTC

- **Causa do vermelho em `test.yaml`:** depois do build da UI, `go test ./...` e `go test -race ./...` atravessavam `app/ui/app/node_modules`; o pacote transitivo `flatted/golang/pkg/flatted` podia aparecer como árvore Go incompleta (`stat .../node_modules/flatted/golang/pkg/flatted: directory not found`). O workflow agora remove `app/ui/app/node_modules` antes de cada descoberta/teste Go, preservando o build da UI e mantendo o comando Go integral, sem excluir pacotes via filtro.
- **Dependência mobile:** `node-forge` transitivo do Expo estava no release vulnerável. O lockfile agora fixa o commit upstream `529a5b4f004595c8f8a75697eca958efe1a2f70d` via tarball HTTPS do codeload; `npm audit --omit=dev --audit-level=high` retorna 0 vulnerabilidades após `npm ci` limpo.
- **Evidências locais:** `go test -race -p=2 ./... -count=1`, `go test -count=1 -bench=. -benchtime=1x ./...`, `go generate ./...`, `go build ./...`, `CGO_ENABLED=0 go build ./...`, typecheck/lint/Vitest/build web e typecheck/policy/audit mobile passaram.
- **Próximo passo:** executar o `test.yaml` no SHA publicado e confirmar Ubuntu/macOS/Windows, race e `go_mod_tidy` verdes; se o runner macOS ainda falhar, usar o log do job para corrigir somente a causa reproduzível.
