# Changelog — Hades

Este arquivo registra as entregas públicas da distribuição `DZ23-LTDA/hades-ollama`. O projeto mantém a atribuição e a licença do Ollama upstream; os recursos agentic específicos estão descritos com seus limites no [guia Hades](docs/CLASS_A_PLUS_GUIDE.md).

Formato baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/). As entradas acumuladas para a próxima versão ficam sob **[Unreleased]**; as notas datadas abaixo de [0.1.0] são o registro cronológico de desenvolvimento, incluindo trabalho posterior à tag v0.1.0 ainda não promovido a um novo release.

## [Unreleased]

### presets guiados de provedor de modelo — 2026-10-10

- Catálogo de presets em `internal/multillm/presets.go` para o cadastro guiado do estágio 4: Ollama, vLLM, llama.cpp (llama-server) e LM Studio como servidores locais; OpenAI, Anthropic, Gemini, DeepSeek, Groq, Mistral, OpenRouter e xAI como APIs; e um preset genérico para qualquer servidor compatível com OpenAI.
- Cada preset entrega `base_url`, nome da **variável de ambiente** da credencial, estilo de autenticação e caminhos de protocolo suportados — nunca o valor da credencial. Servidores locais precisam optar explicitamente por HTTP em loopback (`allow_private` + `allow_insecure_loopback`); os remotos exigem HTTPS.
- Todo preset passa pela MESMA validação usada para a configuração vinda do disco (`validateProvider`), então um cadastro guiado não contorna as regras de HTTPS, endereço privado e caminho suportado.
- `GET /api/v1/providers` passa a devolver `presets` junto de `providers`, alimentando o formulário "adicionar provedor" da interface sem rota nova e sem expor segredo.
- Cobertura: 9 testes em `internal/multillm/presets_test.go` (validade/unicidade, famílias exigidas, loopback sem credencial, HTTPS e nome de variável nos remotos, auth `Bearer` e `X-API-Key`+`Anthropic-Version`, placeholder do preset genérico, tipo inválido) e 1 teste de endpoint em `app/ui/providers_test.go`.
- Limite declarado: um preset não declara modelo nem capacidade — isso continua vindo de descoberta/sonda real; e a homologação com credencial de verdade segue bloqueada por B-07.

### rodada de paridade observável

- Árvore de produto completa em [`docs/agentic/PRODUCT_TREE.md`](docs/agentic/PRODUCT_TREE.md), separando superfície observável, estado atual e alvo unificado.
- Matriz de paridade em [`docs/agentic/PARITY_MATRIX.md`](docs/agentic/PARITY_MATRIX.md), com gates para Manus, Claude, Codex, OmniRoute e demais famílias de harness.
- Shell desktop com rotas reais de Projetos, Biblioteca, Agendado, Habilidades, Plugins e Tarefas, além de navegação lateral Hades.
- Agentic Control Center na Settings com catálogo de modelos e configuração sanitizada; o estado sem backend permanece explícito.
- Endpoint `GET /api/agent/v1/config/safe` sem tokens, caminhos privados ou valores sensíveis.
- Preset OmniRoute local, opt-in HTTP loopback protegido e smoke tests de forwarding/bearer server-side.
- Screenshots reais das novas rotas capturadas com Chromium e notas de proveniência atualizadas.
- Menu lateral Hades aberto por padrão, com smoke test Chromium de rotas, links, ações primárias e estados vazios.
- `UPSTREAM_BASE_COMMIT`, `UPSTREAM_POLICY.md`, guardrail de integridade e workflow CI para impedir perda silenciosa das superfícies agentic.

### fluxo vertical funcional e HarnessRouter — 2026-09-22

- CRUD real tenant-aware de Projetos e Agendado, listagem de missões e artifacts na Biblioteca, com ações server-side e confirmação para exclusão.
- Agentic Console com seleção de Ollama local, Claude, Codex, OmniRoute e projeto persistente; Nova tarefa cria missão real e mantém timeline/approvals.
- Home do shell com composer, recomendações e atalhos funcionais para slides, site, design, jogos e missões; fallback local-first não bloqueia a aplicação quando Settings está offline.
- Catalogação funcional de connectors, MCP stdio, skills e CLIs com normalização de respostas nulas e nenhuma exposição de secret.
- Endpoint JSON `GET /api/agent/v1/metrics` registrado para o Console e smoke E2E live cobrindo home, CRUD, Plugins, Skills e criação de missão.
- Adapter HarnessRouter por OpenAI Responses-compatible com `harness_id` server-side, teste de preservação de metadata e preset [`examples/dz23-harnessrouter.json`](examples/dz23-harnessrouter.json); a execução real continua dependente de instância, chave e harness instalados.
- Manual, matriz de paridade, roadmap, checkpoint de missão e screenshots atualizados.

### Company OS e Desktop Commander Remote MCP — 2026-09-22

- Company OS persistente tenant-aware em `internal/agent/company.go`, com identidade, posicionamento, modelo de negócio, departamentos virtuais, roadmap, metas/KPIs, backlog priorizado, ciclos diários/semanais, relatório operacional, budget, approvals, anomalias e pausa automática.
- API `/api/agent/v1/companies` e rota `/company` com criação, seleção, atualização, roadmap, metas, backlog, ciclos ligados ao scheduler, pausa/retomada e registro de gasto.
- Worker de schedules passou a respeitar `company://<id>` e não cria novas missões para uma empresa pausada.
- Adapter `RemoteMCPManager` para Streamable HTTP com HTTPS fora de loopback, timeout, allowlist, bearer opcional por variável server-side e tool `mcp.remote.call` com approval.
- Presets sem segredos para Desktop Commander local stdio e Remote MCP oficial em `examples/dz23-desktop-commander-mcp.json` e `examples/dz23-desktop-commander-remote.json`.
- Documentação pública de Company OS, Desktop Commander, integração, API, matriz de paridade e estado honesto atualizada.
- Verificações desta rodada: testes `internal/agent`, `server` e `internal/multillm`, testes JSON dos presets e build Vite/TypeScript aprovados. OAuth PKCE, pareamento, conta, agentes físicos, connectors de social/afiliados/dropshipping e operação empresarial real continuam dependências externas.

### jornadas verticais operacionais — 2026-09-22

- Growth OS local no Company OS: campanhas com approval/launch/pause, programas e links de afiliados com destino HTTPS, conversões, catálogo de produtos, pedidos, approval, fulfillment sandbox, estoque e relatório agregado.
- Painel visual funcional na rota `/company` para criar e operar campanhas, programas, produtos e pedidos sem transformar estados externos em sucesso falso.
- Smoke live `scripts/smoke-company-growth.sh` com criação, bloqueio por ausência de approval, aprovação, execução sandbox, conversão, fulfillment e validação de métricas.
- Smoke live `scripts/smoke-builder.sh` com criação, editor visual, preview com hash, export, publicação local e bloqueio de deploy externo sem aprovação explícita.
- Teste de provider session streaming OpenAI-compatible com autenticação do gateway, bearer server-side e tradução de eventos para o protocolo nativo.
- Teste do adapter Desktop Commander Remote MCP e registro da evidência oficial de setup; OAuth, conta, pareamento, agente físico e revogação continuam pendentes por falta de credenciais/dispositivo autorizados.

A rodada não publica campanhas, não movimenta dinheiro, não envia pedidos, não conecta contas sociais/CRM/marketplaces e não declara providers externos como conectados. Os fluxos comprovados são locais e reversíveis.

### Composio, xAI/Grok e social commerce — 2026-09-22

- Remote MCP agora aceita `headers_env` server-side com validação de nomes, permitindo o preset oficial Composio Connect sem expor `x-consumer-api-key` na UI, memória ou logs.
- Preset [`examples/dz23-composio-connect.json`](examples/dz23-composio-connect.json) com endpoint HTTPS, métodos JSON-RPC allowlisted e approval via `mcp.remote.call`.
- Preset [`examples/dz23-xai.json`](examples/dz23-xai.json) para xAI Responses/chat com bearer server-side, e teste de passthrough de `input`/`tools` para `/v1/responses`.
- Runbooks [`COMPOSIO.md`](docs/agentic/COMPOSIO.md) e [`XAI_GROK.md`](docs/agentic/XAI_GROK.md), além de fontes oficiais registradas em [`audit/COMMERCIAL_INTEGRATIONS_SOURCES_2026-09-22.md`](audit/COMMERCIAL_INTEGRATIONS_SOURCES_2026-09-22.md).
- Matriz de paridade atualizada com Composio, xAI/Grok API e Social Commerce/TikTok Shop, separando adapter de conta, scopes, sandbox e operação real.

A rodada não conecta contas externas nem publica posts, anúncios, produtos ou pedidos. Composio, xAI, TikTok Shop, Instagram, Shopify e demais canais exigem credenciais, aprovação de app, scopes mínimos, testes de sandbox, compliance e validação por região.

### hardening interno e baseline de release — 2026-09-22

- Autenticação agentic agora exige bearer por padrão fora de loopback e não aceita `OLLAMA_AGENT_AUTH_REQUIRED=false` em bind externo.
- MCP stdio e Remote MCP exigem métodos allowlisted; Remote MCP usa timeout, bloqueio de destino privado, redirects sem mudança de origem e correlation ID JSON-RPC.
- Contenção de workspace rejeita componentes symlink e StepID fora do formato seguro antes de executar sandbox ou manipular arquivos.
- Approvals carregam organização, actor, policy, nonce e expiração; mutações de Growth validam tenant; pedidos usam `Idempotency-Key`; fulfillment repetido não reduz estoque novamente.
- Control Center e Settings normalizam catálogos nulos; outbox mobile remove bearer persistido, usa idempotency key e backoff com limite de tentativas.
- Teste obsoleto que rejeitava a CLI agentic foi corrigido para exigir sua disponibilidade, mantendo rejeição dos flags antigos.
- Gates aprovados: `CGO_ENABLED=1 go test ./... -count=1`, `CGO_ENABLED=1 go vet ./...`, integrity guard, build Go, build UI, Vitest 20/199 e typecheck mobile.

O hardening não transforma adapters externos em contas conectadas. Composio, xAI, Desktop Commander, redes sociais, marketplaces, PostgreSQL/RLS, Redis, OTLP, dispositivos físicos, instaladores assinados e lojas continuam dependências externas a validar.


### Incremento final de policy — 2026-09-22

Missões agora persistem `capabilities` e recebem somente `workspace:read`/`workspace:write` por default. Tools com scopes de browser, desktop, terminal, sandbox, MCP, connectors ou deploy são negadas até que o escopo seja concedido explicitamente; approvals e allowlists continuam independentes. O bypass de autenticação foi limitado ao FullPath exato do handshake `/devices/:id/connect`, em vez de liberar qualquer caminho terminado em `/connect`.

### capacidades operacionais e lifecycle — 2026-09-22

- Grok Live local-first em `internal/grok`: Responses API, streaming SSE, retries limitados, circuito de falha, status sanitizado e distinção entre fontes ao vivo e memória.
- Evaluation OS determinístico para coding, browser, tools, segurança, memória, planejamento e recuperação; provider router por capacidades, saúde, latência, custo, privacidade e qualidade histórica.
- Company Operations com agentes departamentais persistentes, supervisor, orçamento, SLA, memória, pausa/retomada e Social OS sandbox para contas OAuth pendentes, drafts, approvals, publicação simulada e métricas.
- Lifecycle server-side para connectors, MCP stdio, Remote MCP e skills: habilitar, desabilitar e remover com bloqueio de execução e catálogo sanitizado.
- Correção do deploy Builder: `approved:false` retorna `428 Precondition Required` antes de consultar providers ou configuração de deploy ausente.
- Smoke real aprovado para Growth OS, Builder e shell Chromium; build UI, Vitest, testes backend, integrity guard e contratos locais permanecem verdes.

A rodada continua sem conectar contas externas nem publicar posts, anúncios, produtos, pedidos ou deploys públicos. xAI, Composio, Desktop Commander, social commerce, marketplaces, PostgreSQL/RLS, Redis, OTLP, IdP, GPU, dispositivos físicos, assinatura e lojas exigem ambientes e credenciais autorizados.

### smoke de APIs e correção do Browser Operator — 2026-09-22

- Smoke seguro de catálogos provider com as chaves fornecidas, sem persistir ou imprimir valores secretos.
- Inferência curta em modelo gratuito do OpenRouter aprovada diretamente e pelo gateway local Hades (`/v1/chat/completions`).
- Relatório público de prontidão em [`docs/agentic/READINESS_2026-09-22.md`](docs/agentic/READINESS_2026-09-22.md), distinguindo adapters, validações locais e blockers externos.
- Workflow agentic-quality atualizado para instalar Playwright/Chromium antes do teste real do Browser Operator.
- Erros do Browser Operator agora incluem stderr sanitizado para diagnóstico de dependência, sem expor credenciais.

As credenciais fornecidas no anexo foram tratadas como expostas por terem sido compartilhadas em texto aberto; devem ser revogadas e recriadas pelo operador. Nenhum provider externo é promovido a integração de produção apenas por responder a um health check.

### hardening de connectors, plugins e Browser Operator — 2026-09-22

- Connectors passaram a exigir escopo de organização em chamadas operacionais, respeitar fronteiras de segmento nos `path_prefixes` e rejeitar destinos privados resolvidos por DNS, com exceção explícita apenas para loopback configurado.
- Remote MCP e MCP stdio passaram a validar nomes de ambiente; Remote MCP também bloqueia headers HTTP de transporte e framing.
- Lifecycle global de connectors, MCP, Remote MCP e skills exige `owner`/`admin` no modo autenticado; o modo local permanece restrito ao listener loopback.
- Browser Operator agora usa fallback de Chromium quando um caminho configurado está ausente, preservando diagnóstico e validação de URL.
- Gates locais aprovados: integrity guard, suíte Go completa com CGO, `go vet`, build Go, Vitest 20/199, build UI, typecheck mobile e teste Browser Operator com caminho inválido.

A classificação permanece release candidate local-first. Integrações externas, ambientes distribuídos, dispositivos físicos, assinaturas, lojas e deploys reais continuam condicionados a credenciais, aprovação, sandbox e validação do operador.

### pós-V5, macOS portability e connector lifecycle — 2026-09-22

- Corrigida a validação de workspace em macOS para não confundir o alias `/var` → `/private/var` com um symlink no componente final; a proteção de componentes descendentes e containment foi preservada.
- Corrigido o teste de defaults duráveis para consultar `os.UserConfigDir()` efetivo da plataforma.
- Confirmado o upstream `test` no head `0da6be80`: run `35808941810` SUCCESS, incluindo normal macOS `107015911723`, race macOS `107015880499`, normal Ubuntu `107015911678`, normal Windows `107015911689` e race Ubuntu `107015880534`; integrity `35808941807`, agentic quality `35808941824` e multi-provider `35808941746` também passaram.
- Publicado `411335ba6247b16a431c7f10b5daf8a9fcc0e8f4` com manifest durável de connectors em `OLLAMA_AGENT_STORE/connectors.json`, escrita `0600`, rename após arquivo temporário, rollback em falha e redaction de referências sensíveis na resposta.
- Adicionado `POST /api/agent/v1/connectors` com owner/admin, binding server-side do tenant, validação HTTPS/allowlist e rejeição de campos desconhecidos, `credential_configured` caller-controlled e valores de segredo.
- A tela Plugins passou a registrar connectors com endpoint, operações, nome de env ou ID OAuth, sem campo para token. O valor da credencial continua responsabilidade do ambiente seguro do operador ou do fluxo OAuth oficial.

Os checks remotos do novo SHA estavam pendentes no momento desta publicação documental. A entrega permanece **preview/local RC em hardening**. Nenhum connector externo, conta OAuth, Composio, Woovi/OpenPix, fiscal/NF-e, marketplace, rede social ou provider é declarado conectado ou validado por estes adapters.

### MCP/Remote MCP/skills lifecycle — 2026-09-22

O commit `96fd7edd` adiciona registro durável tenant-aware para MCP stdio, Remote MCP e skills. O runtime padrão usa `mcp.json`, `remote-mcp.json` e `context/skills/*.json` abaixo de `OLLAMA_AGENT_STORE`; manifestos são escritos atomicamente com permissões restritas. O bootstrap via `OLLAMA_AGENT_MCP`, `OLLAMA_AGENT_REMOTE_MCP` ou `OLLAMA_AGENT_CONNECTORS` segue como modo estático explícito e não promete que mutations da UI editem esses arquivos.

Foram publicadas as rotas `POST /api/agent/v1/mcp`, `POST /api/agent/v1/remote-mcp` e `POST /api/agent/v1/skills`, com owner/admin, organização derivada da sessão, rejeição de colisão cross-tenant e JSON estrito. MCP aceita somente executável absoluto regular, métodos allowlisted e nomes de ambiente válidos. Remote MCP mantém HTTPS/SSRF/DNS pinning, redirects same-origin e referências de headers por env. Skills são sempre persistidas `trusted=false`; `trusted` e `enabled` não são autoridade do cliente.

A UI Plugins recebeu chamadas e formulários operacionais correspondentes, sem campo para tokens ou passwords. Os testes cobrem restart, `0600`, rollback, tenant isolation, trust fail-closed, auth, unknown fields, normal e race. O runner local completo passou em integrity, YAML, Go test, vet, build, UI, mobile, audit e diff.

O resultado continua preview/local RC em hardening. Nenhum MCP remoto, conta OAuth, provider externo, social commerce, marketplace, Woovi/OpenPix, fiscal/NF-e ou ação em conta de terceiros é declarado conectado ou validado.

### lifecycle UI completion — 2026-09-22

O commit `eb97e6b8` completou a UI de lifecycle: connectors, MCP/Remote MCP e skills podem ser removidos com confirmação, além de habilitados/desabilitados. A microcopy distingue remoção local de revogação no upstream. Build TypeScript/Vite e 204 testes Vitest passaram; o tree foi publicado limpo. A entrega permanece preview/local RC e não representa revogação de contas externas.

### strict MCP manifest parsing — 2026-09-22

O commit `89b4203e` faz os loaders persistentes de MCP e Remote MCP exigirem EOF após o primeiro documento JSON. Payloads como `[] {}` agora falham no bootstrap em vez de deixar conteúdo trailing sem validação. Regressões normal/race e os gates Go completos passaram: integrity, test, vet, build e diff.

### persistent manifest parser consistency — 2026-09-22

O commit `5816c020` aplica a exigência de EOF ao manifest persistente de connectors, alinhando-o ao hardening já feito para MCP e Remote MCP. Os três loaders agora rejeitam JSON trailing, com regressão dedicada e gates Go completos aprovados.

### strict static connector bootstrap — 2026-09-22

O loader de `OLLAMA_AGENT_CONNECTORS` agora usa o decoder estrito compartilhado, rejeitando campos desconhecidos e JSON trailing. Testes normal/race e gates Go completos passaram. O modo continua estático e não implica conta externa conectada.

### strict skill manifest parsing — 2026-09-22

O loader de skills agora rejeita campos desconhecidos e JSON trailing com regressões dedicadas. A família de plugins passa a ter parsing estrito nos loaders de connectors, MCP, Remote MCP e skills. Os gates Go completos passaram.

### local auth logout hardening — 2026-09-22

`POST /api/agent/v1/auth/logout` agora retorna `204` sem exigir bearer quando `auth_required=false`, evitando dependência indevida do AuthStore. Em modo autenticado, a revogação de bearer permanece estrita. Testes normal/race e gates Go completos passaram.

### local Origin policy hardening — 2026-09-22

O middleware agentic agora aplica a política de `Origin` também sem auth bearer. Mutações cross-site recebem `403`; origens loopback padrão continuam permitidas. Testes normal/race e gates Go completos passaram.

### strict Origin wildcard matching — 2026-09-23

A allowlist `scheme://host:*` agora aceita somente portas variáveis no host exato; configurações não são mais interpretadas como prefixo arbitrário de domínio. Testes normal/race e gates Go completos passaram.

### portable best-effort sandbox — 2026-09-23

O `sandbox.exec` agora resolve Python/Node por plataforma. macOS e Windows usam processo best-effort com timeout, output limitado e cancelamento; a resposta informa `network_isolation=not-enforced`. Strict permanece Linux-only e fail-closed. Gates Go completos e compilações Darwin/Windows do pacote agentic passaram.

### strict agentic request JSON — 2026-09-23

O helper `decodeJSON` agora rejeita trailing JSON depois do primeiro documento, além de campos desconhecidos. A proteção vale para todos os handlers que o reutilizam. Testes normal/race e gates Go completos passaram.

### bounded agentic JSON bodies — 2026-09-23

O parser HTTP agentic agora limita bodies JSON a 4 MiB antes do decode, além de exigir documento único e rejeitar campos desconhecidos. Testes normal/race e gates Go completos passaram.

### strict development token input — 2026-09-23

`/auth/dev/token` agora usa o parser JSON central com limite, campos conhecidos e EOF; a flag explícita e o loopback continuam obrigatórios. Testes normal/race e gates Go completos passaram.

### deployment adapter hardening — 2026-09-23

O adapter de deploy agora rejeita root symlink, bloqueia redirects e valida o IP conectado para evitar SSRF por DNS. HTTP é aceito somente em loopback; deploys externos continuam não validados sem credenciais e aprovação do operador. Testes normal/race e gates Go completos passaram.

### media workspace input hardening — 2026-09-23

O MediaManager agora rejeita traversal e symlink em inputs de áudio/imagem, valida arquivos regulares dentro do workspace e recusa áudio acima de 100 MiB antes da leitura. Testes normal/race e gates Go completos passaram.

### Company Growth sandbox mode — 2026-09-23

Campaigns, affiliate programs/links e orders agora expõem `mode: sandbox`; o relatório informa `sandbox_only`, e modos externos falham sem adapter upstream validado. Testes normal/race e gates Go completos passaram.

### durable configuration reporting — 2026-09-23

`safeConfig` agora reconhece connectors, MCP, mídia e deployments registrados no Runtime durável, sem vazar valores. O estado continua significando apenas configuração, não conexão upstream.

### media/deploy audit and observable captures — 2026-09-23

- Corrigida a escrita de outputs de mídia e OCR contra symlink intermediário e TOCTOU externo com `os.Root`, paths relativos, quotas e validação do destino antes de provider/processo externo.
- Corrigida a leitura de transcrição e visão para usar descritor seguro, limite antes da alocação integral, leitura cancelável e detecção de crescimento.
- Corrigido o pacote de deploy para excluir arquivos privados, credenciais, metadados internos, backups, logs e arquivos não regulares antes da leitura ou transmissão.
- Corrigido o egress de deploy para resolver e validar endereços antes do TCP, com loopback somente por contexto explícito e Host/SNI preservados.
- O capturador de paridade agora é portátil ao checkout, exige estado estável e interação segura, falha em diagnósticos inesperados e grava manifesto de SHA/build/viewport/limitações. A Home do README continua sendo tratada no PR documental separado baseado na main.

### UI runtime dependency hardening — 2026-09-23

O commit `eff054c1` atualizou `streamdown` para `2.6.0`, removeu o `@tanstack/react-router-devtools` não utilizado de `dependencies`, declarou `shiki` diretamente e fixou `mdast-util-to-hast` em `13.2.1`. A mudança removeu do grafo runtime os caminhos vulneráveis de Mermaid, DOMPurify, lodash-es, seroval e uuid. A UI passou 204 testes, build e `npm audit --omit=dev` sem vulnerabilidades; o mobile também passou typecheck e audit de produção.

### explicit approval reasons — 2026-09-23

A Agentic Console agora exige um motivo escrito antes de aprovar ou rejeitar uma ação, limita a entrada a 512 caracteres, desabilita decisões vazias e envia o texto real no ledger via API. A cobertura adicionada confirma o payload com nonce, decisão e justificativa.

### server-side approval reason bound — 2026-09-23

O runtime agora rejeita motivos de approval acima de 2048 bytes antes da decisão CAS, complementando o limite de 512 caracteres da UI e a regressão contra clientes que contornem o formulário.

### Tel-Agent textual no Company OS — 2026-09-23

O Company OS ganhou o canal `tel-agent.text`, com histórico persistente tenant-aware, retorno estruturado e operações locais allowlisted para leitura de relatório, criação de backlog e rascunho de campanha. Mutações exigem role operacional quando auth está ativa; entradas são limitadas e redigidas antes da persistência. O rascunho de campanha permanece sandbox e gera approval pendente. Telefonia, WhatsApp, SMS e mensagens externas continuam explicitamente não configurados e não homologados.

### idempotência do Tel-Agent — 2026-09-23

O commit `d67e7a79` adicionou `Idempotency-Key` ao Tel-Agent: replay retorna o mesmo resultado, payload divergente retorna `409`, o vínculo sobrevive a restart e a UI conserva a chave durante retry. O head passou os gates locais completos; a CI pública ainda estava em execução no registro.

### Company cycles transacionais — 2026-09-23

O commit `46d6b34c` impede ciclos Company órfãos quando a persistência do schedule falha, adiciona replay por `Idempotency-Key` e evita schedules duplicados em retries. Testes normais/race e o runner local completo passaram; a CI remota ainda estava em execução.

### bounded schedule worker retries — 2026-09-23

O commit `86597b82` registra falhas de criação de missão, aplica backoff bounded e pausa automaticamente schedules após três falhas consecutivas. Claims não persistidos são revertidos em memória. Gates locais completos passaram; worker distribuído e DLQ real permanecem pendentes.

### transactional local queue mutations — 2026-09-23

O commit `717a7e4f` impede que falhas de persistência deixem jobs parcialmente aplicados em memória durante claim, ack, nack ou replay. Gates locais completos passaram; lease distribuído e Redis real continuam pendentes.

### queue running-state guards — 2026-09-23

O commit `9d28cbc2` impede `Ack` e `Nack` fora de jobs `running` em queues local e Redis. A validação Go completa passou; integração Redis real e lease distribuído continuam pendentes.

### Redis queue failure compensation — 2026-09-23

O commit `883a17e2` adicionou compensações para Enqueue, Claim, retry e Replay do RedisQueue quando comandos subsequentes falham. Gates Go passaram; o smoke Redis real continua não executado por falta de endpoint configurado.

### Redis delayed-job requeue compensation — 2026-09-23

O commit `b1aaebfd` restaura jobs delayed quando `moveDue` não consegue completar o `LPUSH` para pending após remover o item do sorted set.

### Settings screenshot capture guard — 2026-09-23

O alias histórico `docs/images/screens/settings.png` foi atualizado para a captura funcional de Settings. O capturador agora rejeita páginas com conteúdo insuficiente e screenshots PNG anormalmente pequenos, evitando evidência visual branca ou incompleta.

## [0.1.0] — Preview público

### Incluído

- Runtime agentic em Go com missões persistentes, planejamento, execução, observação, recovery, artifacts e approvals.
- Ferramentas com sandbox, contenção de workspace, políticas de SSRF/path e trilhas de auditoria.
- Browser Operator via Playwright, MCP stdio, memória, projetos, skills e scheduler/webhooks.
- UI Agentic Console para missões, métricas, multiagente, pesquisa e approvals.
- Base de companion desktop para Linux, macOS e Windows e cliente Expo mobile com cache/outbox offline.
- Conectores allowlisted, providers Claude/Codex, autenticação, RBAC, OAuth/OIDC/SAML/MFA, colaboração e device pairing.
- Filas, DLQ/replay, PostgreSQL/RLS, Redis, OTLP, WebSocket/mTLS e exportadores/builders.
- DeploymentManager com providers Vercel, Netlify e generic, approval obrigatório e controles de segurança de publicação.
- Documentação pública em português, arquitetura, API, integrações, roadmap, política de segurança e screenshots reais.

### Estado honesto do preview

Adapters, contratos, testes e documentação não significam que contas externas, credenciais, instaladores assinados, publicação em lojas, modelos multimídia ou testes físicos estejam disponíveis neste ambiente. Deploy real, SSO contra um IdP real, GPU/modelos locais e distribuição de companions permanecem validações dependentes de ambiente. Os mockups em `docs/images/mockups/` são conceitos; as limitações das screenshots reais estão em `docs/images/screens/SCREEN_CAPTURE_NOTES.md`.

### Verificação

Os gates comprovados desta revisão incluem testes focados do runtime/server, build Go, build/typecheck da UI e typecheck mobile em suas fases correspondentes. O workflow público [`dz23-agentic-quality`](.github/workflows/dz23-agentic-quality.yaml) executa os gates de qualidade e a integração distribuída no GitHub Actions.

[0.1.0]: https://github.com/DZ23-LTDA/hades-ollama/releases/tag/v0.1.0

## Registro de desenvolvimento

### 2026-09-22 — P0 ownership e Company invariants

- Builder projects receberam `OrganizationID` e todas as operações HTTP de projeto passaram a usar lookup/listagem scoped, com teste cross-tenant `403` e ausência de mutação.
- Builder agora exige entry presente, rejeita symlink no preview/export e escapa nomes inseridos nos templates HTML.
- Company create passou a usar DTO allowlisted e a resetar campos server-managed; gasto de agente pausado ou acima do budget falha atomicamente.
- Gates locais do slice passaram; isso não encerra os P0 restantes nem transforma adapters externos em integrações conectadas.

### 2026-09-22 — P0 tenant isolation em orchestration, traces e devices

- Orchestration jobs receberam `organization_id`; plan/get/run/cancel server-side passaram a usar métodos scoped.
- Spans de missão e ferramenta carregam organização e a API de traces filtra por tenant.
- Devices e pairing passaram a aplicar ownership em listagem, heartbeat, revoke e consumo de código de pairing, com regressões HTTP cross-tenant `403` e sem mutação.
- Gates completos da slice passaram; companion físico, mTLS e os demais P0 continuam sem prova de produção.

### 2026-09-22 — mission approvals com policy, nonce e CAS

- Decisão de approval autenticada exige membership owner/admin; operator/viewer não podem aprovar.
- O endpoint usa nonce de uso único e versão corrente da missão, rejeitando replay e conflito de concorrência.
- Agentic Console e tipos de API foram atualizados para transportar nonce/policy.
- Company/Growth/Social approvals booleanas continuam explicitamente fora desta garantia até migrarem para decisões auditáveis.

### 2026-09-22 — Company OS approvals auditáveis

Campanhas, afiliados, pedidos e drafts sociais passaram a emitir `CompanyApproval` server-side com policy, nonce, expiração, actor, organização, razão e versão CAS. Os endpoints de aprovação não aceitam mais `{}` como autorização: localizam a decisão pendente e só projetam o recurso aprovado após validação. A rota de gasto com `approved` booleano permanece aberta para a próxima slice.

### 2026-09-22 — spend approvals e budget atômico

A rota HTTP de gasto deixou de aceitar `approved` do cliente. Solicitações que atravessam a política de budget viram approvals pendentes com nonce, expiração, actor, organização e CAS; a nova fila do Company OS decide e só então contabiliza o valor. O limite mensal continua pausando a Company sem débito parcial.

### 2026-09-22 — DLP de resultados e observabilidade

Resultados de tools, erros, eventos, traces e serializações JSON/Postgres passaram a usar redação recursiva de credenciais. A cobertura inclui campos estruturados sensíveis e tokens GitHub/OpenAI/OpenRouter/xAI/AWS/Slack/Bearer/PEM, com regressões de token injection. Egress de connectors/MCP permanece uma superfície distinta, pois credenciais operacionais não podem ser redigidas cegamente.

### 2026-09-22 — Remote MCP egress hardening

O cliente Remote MCP deixou de usar proxy ambiental, limita redirects ao mesmo origin e verifica o IP efetivamente conectado para rejeitar destinos privados e DNS rebinding. Foram adicionadas regressões para redirect same-origin e conexão TCP privada local.

### 2026-09-22 — MCP stdio lifecycle hardening

MCP stdio passou a usar executável absoluto não-symlink, cwd privado, ambiente mínimo, limites de argumentos/request/response/stderr, redaction de stderr, grupo de processo e restart/cleanup verificáveis. A implementação é explicitamente best-effort; seccomp/cgroups/rlimits e limites fortes de recursos continuam fora desta slice.

### 2026-09-22 — Media egress/download hardening

Downloads de mídia agora bloqueiam redirects e proxy ambiental, verificam o IP conectado, aplicam limites bounded e validam MIME/magic antes de materializar PNG/JPEG/WebP/MP4/WAV. Testes negativos cobrem redirects, MIME incompatível, conteúdo falso, overflow e destinos privados.

### 2026-09-22 — Connector egress parity

Connectors passaram a bloquear proxy ambiental e redirects mesmo com client HTTP substituído, manter verificação de IP conectado e aplicar limites bounded de request/response. Testes negativos cobrem redirects e payloads oversized sem depender de uma conta externa.

### 2026-09-22 — OAuth redirect URI allowlist

OAuth start/callback agora exigem allowlist exata por provider, canonicalização, HTTPS e rejeição de fragmentos/userinfo. Loopback HTTP só é permitido com flag explícita e URI allowlisted; o estado PKCE usa a URI canônica.

### 2026-09-22 — session handling web/mobile

O cliente web agentic removeu bearer persistido em localStorage e passou a usar sessão em memória com limpeza 401/403. O mobile limpa SecureStore ao expirar, envia nonce de approval, impede dupla ação e exige confirmação antes de apagar cache/outbox no logout.

### 2026-09-22 — OAuth endpoint egress hardening

OIDC discovery, JWKS, userinfo e token exchange passaram a bloquear proxy/redirects, validar HTTPS e rejeitar IP privado efetivamente conectado. Discovery também valida os endpoints retornados pelo IdP antes de usá-los.

### 2026-09-22 — tool process containment

Terminal e sandbox tools passaram a usar grupo de processos encerrável, limites de timeout/saída, stderr redacted e status explícito de isolamento. O sandbox usa `ulimit` best-effort quando disponível; seccomp/cgroups e sandbox forte permanecem fora do claim.

### 2026-09-22 — plugin/MCP/skill ownership

Connectors, MCP stdio, Remote MCP e skills passaram a carregar ownership opcional por organização; lifecycle autenticado usa métodos scoped e nega cross-tenant/global mutation, enquanto catálogo global permanece read-only.

### 2026-09-22 — CI/release quality gates

O workflow agentic passou a verificar suíte Go completa, vet, build, UI tests/build e mobile typecheck. O release workflow ganhou quality gate obrigatório antes dos builds e da publicação; signing, SBOM e provenance continuam condicionais ao ambiente GitHub/credenciais reais.

### 2026-09-22 — Grok provider contract

Grok passou a validar modelo contra allowlist e catálogo no health probe; a rota HTTP rejeita `stream:true` explicitamente até existir endpoint SSE de produção.

### 2026-09-22 — Compose secure defaults

PostgreSQL/Redis/OTLP foram limitados a loopback por padrão; PostgreSQL/Redis exigem secrets de ambiente e CI usa credenciais efêmeras. O sandbox local não possui Docker, então a integração distribuída permanece dependente do Actions.

### 2026-09-22 — provider selection contract

A Nova tarefa envia e persiste `provider`; apenas `ollama-local` é aceito até adapters Claude/Codex/OmniRoute reais existirem. A UI marca os demais como não conectados.

### 2026-09-22 — release artifact integrity

Release agora falha para artefatos ausentes/vazios e pode gerar attestation de provenance somente com configuração explícita `OLLAMA_ENABLE_ATTESTATIONS=true` no ambiente GitHub.

### 2026-09-22 — tenant isolation de jobs/replay

Fila de jobs agora filtra listagem e replay pela organização da missão, negando cross-tenant sem mutação.

### 2026-09-22 — artifact manifest path safety

Manifests de artefatos agora rejeitam symlinks e resoluções fora do workspace antes de ler e hashear arquivos.

### 2026-09-22 — screenshots do estado atual

As dez telas públicas do shell (`class-a-plus-*.png`) foram recapturadas com Chromium contra o Vite dev e o servidor Ollama local em loopback. README, guia e notas de proveniência agora apontam para as imagens atuais de Home, Agentic Console, Settings, Company OS, Projetos, Biblioteca, Agendado, Skills, Plugins e Tarefas. As capturas mostram apenas estado local/sandbox e não afirmam providers, credenciais, deploys ou dispositivos externos configurados.

### 2026-09-22 — correção do smoke distribuído do CI

O job de integração do `dz23-agentic-quality` foi corrigido após a execução no commit `2933f6e2` revelar que o teste PostgreSQL RLS estava usando a role bootstrap superusuária. O workflow agora cria uma role não-superusuária dedicada ao teste tenant-scoped, mantém as passwords efêmeras em variáveis locais do passo e não as grava em `GITHUB_ENV`. O integrity guard verifica os dois contratos. Os gates locais completos passaram; a nova execução do GitHub Actions ainda é necessária para validar Docker/PostgreSQL/Redis/OTLP no runner.

### 2026-09-22 — follow-up do bootstrap PostgreSQL no CI

O run remoto `35737050056` revelou que a primeira correção do smoke ainda usava sintaxe inválida ao combinar variável `psql` com `DO $$`, e que a etapa de cleanup não herdava variáveis locais de outro passo. O workflow agora usa comandos condicionais `CREATE ROLE`/`ALTER ROLE` com password efêmera hexagonal e placeholders não secretos somente para o `docker compose down`. A nova confirmação distribuída permanece pendente.

### 2026-09-22 — quality workflow distribuído verde

O run GitHub Actions `35737772235`, no commit `15e8ae60`, passou integralmente: Go/server, Browser Operator, SBOM, Web/Mobile e PostgreSQL RLS + Redis DLQ + OTLP. O smoke agora possui prova remota real com role tenant-scoped não-superusuária e cleanup independente dos secrets do passo anterior.

### 2026-09-22 — CapabilityPolicy deny-by-default

Missões agora validam grants contra uma policy central e scopes declarados por tool na criação, no planejamento e antes do efeito. Descritores sem capability conhecida são rejeitados; o default passou a ser `workspace:read`. O Agentic Console expõe escrita como opt-in explícito, e approvals carregam os scopes efetivos da ação.

### 2026-09-22 — matriz de referências e capacidades

Foi adicionada `audit/HARNESS_CAPABILITY_MATRIX.md`, com decisões rastreáveis para as referências nomeadas e os 13 repositórios fornecidos. A matriz separa produto, coordenação, inferência, execução especializada e evidências; preserva Ollama como motor padrão; e mantém providers, harnesses e serviços externos substituíveis e opt-in.

### 2026-09-22 — correção de races em pesquisa e MCP

O fixture concorrente de pesquisa passou a usar contador atômico. O buffer stderr de MCP passou a proteger `Write`, `String` e `ReadFrom`, evitando que `io.Copy` contorne o limite/mutex durante cancelamento e `Stop`. `go test -race ./...` agora passa localmente.

### 2026-09-22 — quality workflow agentic verde no SHA 428a99bd

O run GitHub Actions `35749351291` passou os quatro jobs do workflow agentic: Go/server, PostgreSQL RLS + Redis DLQ + OTLP, Web/Mobile e SBOM. O teste upstream de matriz permanece separado e será confirmado no head corrente.

### 2026-09-22 — bootstrap MCP estrito

Foi adicionada cobertura de servidor para carregar manifestos MCP stdio e Remote MCP com JSON estrito, rejeição de campos desconhecidos, trailing data e allowlists vazias. O teste comprova registro local; não comprova endpoint, credencial ou sessão externa.

### 2026-09-22 — correlação JSON-RPC MCP

MCP stdio passou a ignorar notificações JSON-RPC sem identificador e aguardar o response com o `id` do request. A regressão inclui execução normal e race; multiplexação e transporte externo continuam fora da prova local.

### 2026-09-22 — Browser Operator no workflow upstream

O teste upstream revelou que os jobs `test`/`race` não instalavam o módulo Python Playwright, embora o Browser Operator o importe. O workflow agora instala Playwright `1.63.0` e Chromium nos jobs relevantes, com tratamento Linux/macOS/Windows; o launcher Go deixou de depender de `/usr/bin/python3` fixo. A matriz nativa Linux/Windows foi tornada manual e opt-in porque requer runners compatíveis. O CI remoto desta correção ainda precisa concluir.

### 2026-09-22 — workflow upstream preso e Playwright corrigido

O run upstream `35755046119` foi cancelado após confirmar jobs nativos `linux`/`windows` aguardando runners customizados inexistentes no PR. A pinagem anterior do Playwright também era inválida no índice do runner; os jobs `test`/`race` passam a usar `1.63.0`. A matriz nativa agora é manual e opt-in (`workflow_dispatch` + `run_native_matrix=true`), sem mascarar a ausência de hardware/runners como validação concluída.

### 2026-09-22 — upstream CI multiplataforma verde no head publicado

Os commits `5db7261e`, `86a2706b`, `849781af` e `de0e8677` corrigiram os achados determinísticos do golangci-lint, tornaram o Browser Operator capaz de localizar o Chromium instalado pelo Playwright e normalizaram os arquivos de companion Darwin/Windows. O head `de0e86772e96372789c10d924eb5738f8808821b` passou `class-a-plus-integrity` (`35769597628`), `dz23-agentic-quality` (`35769597363`), `dz23-multi-provider` (`35769597578`) e o upstream `test` (`35769597404`). O upstream executou test em Linux/macOS/Windows e race em Linux/macOS. Isso valida o caminho normal de CI, mas não executa a matriz GPU/nativa manual nem comprova sandbox forte, credenciais, deploys, dispositivos físicos, signing ou produção.

### 2026-09-22 — hardening de sessão, Company e terminal

O cliente agentic não encerra mais uma sessão válida ao receber `403`; somente `401` dispara invalidação local. O endpoint autenticado `POST /api/agent/v1/auth/logout` revoga o bearer no servidor, e a UI fornece o fluxo correspondente. Mutations críticas de spend, conversão de afiliado e métrica social aceitam `Idempotency-Key`, armazenam somente digest/fingerprint e tornam retries seguros, com conflito explícito para reutilização com payload diferente. O terminal allowlisted passou a rejeitar flags desconhecidas, argumentos de `pwd` e paths de `ls` fora do workspace. Também foi corrigido o contrato nomeado do callback do planner para o cliente Ollama real.

No head `935fb273`, `class-a-plus-integrity` passou nos runs `35784205466`/`35784211426`, `dz23-agentic-quality` nos runs `35784205382`/`35784211429`, `dz23-multi-provider` no run `35784211481` e o upstream `test` no run `35784211483`. O upstream passou Linux, macOS e Windows, race em Linux/macOS, patches e `go_mod_tidy`. A matriz GPU/nativa continua manual/opt-in e o produto permanece preview/local RC em hardening.

### 2026-09-22 — sandbox strict opt-in e cgroup kill

O `sandbox.exec` ganhou modo Linux `OLLAMA_AGENT_SANDBOX_MODE=strict`, que falha fechado sem cgroup v2 delegado e configura namespaces, `no-new-privs`, seccomp para amd64/arm64, limites de CPU/memória/PIDs/swap e `cgroup.kill` no timeout. O modo padrão permanece best-effort; não há claim de host externo homologado.

### 2026-09-22 — Origin policy e mobile tenant-aware

Mutations agentic com `Origin` cross-site são recusadas pela allowlist existente de `OLLAMA_ORIGINS`, sem transformar `403` em logout. O mobile consulta `auth/session` e somente depois habilita outbox/push autenticados; cache, fila offline e registro de push são namespaced por servidor e organização. Retries usam idempotency key, If-Match, backoff e conflitos explícitos. O typecheck e o Expo web export passaram; builds físicos, push remoto e stores continuam externos.

### 2026-09-22 — release SBOM e metadata verificável

O workflow de release passou a gerar SBOM CycloneDX, `release-metadata.json`, `sha256sum.txt` e verificação `sha256sum -c`; attestation GitHub permanece condicional a `OLLAMA_ENABLE_ATTESTATIONS=true`. Isso melhora a cadeia de evidência, mas não é assinatura efetiva de instaladores nem prova de um release tag executado.

No head `a7bc82f5`, integrity push `35793674459` e PR `35793679458` passaram; os workflows agentic/multi-provider/upstream ainda estavam em execução ou fila no momento do registro. O pacote mobile reportou 18 vulnerabilidades de produção no `npm audit` (11 moderate, 7 high), pendentes de triagem.

### 2026-09-22 — remediação transitiva do audit mobile

O audit inicial do pacote Expo encontrou 18 vulnerabilidades transitivas, cuja correção automática exigia Expo/React Native major. A solução publicada usa overrides mínimos para `image-size@2.0.4`, `postcss@8.5.28` e `uuid@11.1.1`; typecheck, Expo web export e `npm audit --omit=dev` passaram com zero vulnerabilidades de produção. A migração major e a validação física continuam separadas.

### 2026-09-22 — CI normal verde no head e6e0632b

O head `e6e0632ba5ff0495ed4b061221c90696478b3d67` passou upstream `test` (`35794443450`), integrity (`35794443307`), multi-provider (`35794443300`) e agentic quality (`35794443501`). O PR ficou com 21 checks successful, 3 skipped, 0 failing e 0 pending. O upstream executou Linux/macOS/Windows e race Linux/macOS; o workflow agentic passou Go/server, RLS + Redis DLQ + OTLP, Web/Mobile e SBOM.

Os gates Node agora usam `npm ci` com lockfile versionado. A matriz nativa/GPU continua manual e opt-in; os skips não são tratados como homologação física. O produto segue preview/local RC em hardening.

### 2026-09-22 — V5: provider efetivo, connectors truthful e CI multiplataforma

O runtime passou a selecionar planner pelo provider/modelo efetivo e a falhar fechado quando o executor não está configurado. O data root durável foi separado do workspace, o Remote MCP passou a fixar os IPs aprovados após DNS e o catálogo de connectors ganhou estados tenant-aware sem expor tokens. A UI passou a derivar o seletor de providers/modelos do catálogo real, eliminando aliases fictícios.

O upstream encontrou primeiro um helper planner não utilizado e depois uma falha `EEXIST/ENOENT` do cache npm global no Windows. O helper foi removido e o workflow passou a usar cache npm isolado por runner e `fail-fast: false` nas matrizes normal/race. No head `0f95b6a1`, passaram upstream `test` (`35804229207`), `class-a-plus-integrity` (`35804229189`), `dz23-multi-provider` (`35804229204`) e `dz23-agentic-quality` (`35804229301`). O caminho normal confirmou Linux, macOS e Windows, race aplicável, Go/server, serviços distribuídos, Web/Mobile e SBOM.

A matriz GPU/nativa continua manual e opt-in. Catálogo ou `credential_configured` não significa conta externa conectada; credenciais, OAuth, Woovi/OpenPix, fiscal, marketplaces, deploys, mídia, dispositivos, signing e stores continuam dependências do operador.

### 2026-09-23 — gates completos do candidato Tel-Agent

O head `55dfa325` passou o runner local absoluto completo: integrity, YAML, Go test/vet/build, Vitest/build/audit de produção da UI, typecheck/audit de produção mobile e diff. A CI pública ainda precisava concluir jobs Go e a matriz upstream no momento do registro; nenhum check falho foi observado nessa consulta.
