# Relatório de prontidão — Ollama Classe A+ (histórico de 2026-09-22)

> **Registro histórico, não runbook atual.** Este documento descreve o estado e configuração observados em 2026-09-22 e não deve ser usado para provisionar PostgreSQL. Para o desenho candidato atual, siga `docs/agentic/INTEGRATIONS.md` e `SECURITY.md`; PostgreSQL/RLS continua sem aprovação de produção.

**Data:** 2026-09-22
**Repositório:** `DZ23-LTDA/ollama-classe-a-plus`
**Branch auditada:** `feat/manus-parity-omniroute`
**Base local:** commit `dfe07d5665b11f025fb36c6382298e8c053733f5`

## Veredito executivo

O projeto está **pronto como release candidate local-first para instalação, desenvolvimento e validação controlada**. Ele não está pronto para ser classificado como produto final de produção universal, porque ainda faltam validações com contas externas, infraestrutura distribuída real, dispositivos físicos, modelos multimídia locais, assinatura de instaladores, lojas e operações comerciais.

A distinção é importante: o runtime, a UI, os contratos, os guardrails e as jornadas sandbox podem ser executados e testados. Já uma integração com uma conta real de xAI, Composio, TikTok Shop, Meta, Shopify ou Desktop Commander não pode ser considerada concluída apenas porque existe um adapter ou uma tela.

## Teste das APIs fornecidas

As chaves do arquivo fornecido foram usadas somente em chamadas curtas de leitura/diagnóstico e em uma inferência gratuita do OpenRouter. Nenhuma chave foi copiada para o repositório, para logs públicos, screenshots ou este relatório. O arquivo fornecido continha credenciais em texto aberto; todas devem ser consideradas expostas e rotacionadas pelo operador.

| Serviço | Verificação realizada | Resultado | Interpretação |
|---|---|---:|---|
| OpenRouter | `GET /api/v1/models` e inferência gratuita | HTTP 200 | Chave respondeu; gateway Classe A+ também encaminhou uma chamada gratuita ponta a ponta |
| Groq | `GET /openai/v1/models` | HTTP 200 | Chave respondeu ao catálogo |
| DeepSeek | `GET /models` | HTTP 200 | Chave respondeu ao catálogo |
| Fireworks | `GET /inference/v1/models` | HTTP 200 | Chave respondeu ao catálogo |
| Cerebras | `GET /v1/models` | HTTP 200 | Chave respondeu ao catálogo |
| Mistral | `GET /v1/models` | HTTP 200 | Chave respondeu ao catálogo |
| NVIDIA | `GET /v1/models` | HTTP 200 | Chave respondeu ao catálogo |
| Novita | `GET /openai/v1/models` | HTTP 200 | Chave respondeu ao catálogo |
| Cohere | `GET /v1/models` | HTTP 200 | Chave respondeu ao catálogo |
| Google Gemini | `GET /v1beta/models` com `x-goog-api-key` | HTTP 200 | Chave respondeu ao catálogo |
| Hugging Face | `GET /api/whoami-v2` | HTTP 200 | Token respondeu à identificação |
| GitHub | `GET /user` | HTTP 200 | Token respondeu à identificação; revisar escopos e rotacionar |
| Cloudflare | `GET /user/tokens/verify` | HTTP 200 | Token respondeu à verificação |
| Together | `GET /v1/models` | HTTP 401 | Chave inválida, expirada ou sem permissão para esse endpoint |
| xAI | `GET /v1/models` | HTTP 403 | A chave foi recusada; não classificar xAI como conectado |
| Hyperbolic | `GET /v1/models` | HTTP 401 | Chave inválida, expirada ou sem permissão |
| Alibaba/DashScope | `GET /compatible-mode/v1/models` | HTTP 401 | Chave inválida, expirada ou endpoint/escopo incompatível |
| Voyage | endpoint de catálogo tentado | HTTP 404 | Endpoint escolhido não prova validade ou invalidade da chave; requer health check específico |

O smoke escolheu o modelo gratuito `inclusionai/ling-3.0-flash-vl:free` no OpenRouter. A chamada direta retornou HTTP 200. Em seguida, o servidor Ollama Classe A+ foi iniciado em uma porta isolada, publicou o modelo remoto em `/v1/models` e devolveu HTTP 200 através de `/v1/chat/completions`. Esse é o principal teste externo ponta a ponta desta rodada.

## Gates locais

Os gates locais abaixo passaram antes ou durante esta rodada: `scripts/check-class-a-plus-integrity.sh`, `CGO_ENABLED=1 go test ./... -count=1`, `CGO_ENABLED=1 go vet ./...`, build Go, Vitest da UI com 20 arquivos e 199 testes, build Vite, typecheck Expo, smoke Chromium do shell, smoke Growth OS e smoke Builder com approval. O teste específico `TestBrowserOperatorNavigateAndSnapshot` também passou localmente após a correção do CI.

O gate distribuído com PostgreSQL, Redis e OTLP foi **N/A neste sandbox**, porque Docker não está instalado. O workflow público possui job para essa validação; a conclusão deve ser verificada no GitHub após a execução do novo commit.

## Correção interna desta rodada

O workflow `.github/workflows/dz23-agentic-quality.yaml` agora instala Playwright e Chromium antes de executar os testes do Browser Operator e exporta o executável descoberto para o processo Go. O adaptador `internal/agent/browser.go` também passou a incluir stderr sanitizado no erro, tornando falhas de dependência diagnosticáveis sem imprimir segredos.

## O que falta para o produto final de produção

| Prioridade | Pendência | Por que ainda não está pronta |
|---|---|---|
| P0 | Rotação das chaves fornecidas | Foram entregues em texto aberto; devem ser revogadas e recriadas antes de qualquer uso contínuo |
| P0 | CI remoto verde no novo commit | O commit anterior falhou no Browser Operator por dependência ausente; a correção precisa ser executada no GitHub |
| P0 | Testes de autorização e isolamento em staging distribuído | O compose, RLS, Redis e OTLP existem, mas precisam de PostgreSQL/Redis/Collector reais e execução repetível |
| P1 | Composio e Desktop Commander | Adapter/preset existem; OAuth, sessão, pairing, dispositivo e uma tool real ainda precisam de conta de teste autorizada |
| P1 | xAI/Grok | O adapter existe; a chave fornecida recebeu 403, portanto Responses, tools e quota xAI não foram validados |
| P1 | Social commerce | Growth OS é sandbox; TikTok Shop, Meta, Instagram, Shopify, YouTube e WhatsApp exigem apps, escopos, webhooks, região e aprovação |
| P1 | Desktop e mobile físicos | Linux/macOS/Windows e Android/iOS precisam de execução em dispositivos/runners reais, push, offline e rollback |
| P1 | Mídia local | OCR, visão, imagem, vídeo, TTS e STT precisam de modelos instalados, hardware e testes de qualidade |
| P1 | Builder e deploy externo | Preview/export/deploy local estão testados; Vercel, Netlify, AWS, Cloudflare e DNS exigem contas e smoke reversível |
| P2 | Release de distribuição | Faltam instaladores assinados, SBOM/provenance verificável, auto-update/rollback e publicação nas lojas |
| P2 | Hardening adicional | Ainda são desejáveis `openat`/handles contra TOCTOU, atestados assinados de skills, egress enforcement completo e storage seguro de credenciais do renderer |

## Conclusão operacional

O projeto pode ser entregue agora para **desenvolvimento local, avaliação do runtime, testes de providers com credenciais rotacionadas e construção de jornadas agentic**. Ele não deve ser vendido ou documentado como uma plataforma já validada em produção, como uma cópia interna do Manus/Grok/Claude/Codex, nem como uma operação comercial autônoma sem os gates externos acima.

A próxima barreira objetiva é publicar a correção do Browser Operator, aguardar o CI e, depois, executar staging distribuído e integrações externas uma por uma, sempre com escopos mínimos, approval, idempotência, rollback e nenhuma chave no Git.


## Hardening posterior ao smoke de APIs — 2026-09-22

A rodada posterior fechou quatro riscos internos que ainda eram ajustáveis sem credenciais externas. Chamadas de connector agora exigem `organization_id` no caminho tenant-aware; o matching de `path_prefixes` respeita limites de segmento; e o transporte padrão rejeita destinos privados resolvidos por DNS, mantendo loopback somente quando o próprio `BaseURL` é loopback. O Remote MCP passou a validar nomes de ambiente e bloquear headers de transporte que poderiam interferir no framing HTTP. O MCP stdio aplica a mesma validação às variáveis herdadas.

Alterações globais de lifecycle de connectors, MCP, Remote MCP e skills agora exigem papel `owner` ou `admin` quando o runtime está autenticado. O modo local-first sem autenticação continua disponível somente no bind loopback já protegido pelo middleware. O Browser Operator passou a escolher um fallback conhecido de Chromium quando um caminho configurado não existe; o teste específico passou localmente inclusive com caminho configurado inválido.

Os gates desta iteração passaram localmente: integrity guard, `CGO_ENABLED=1 go test ./... -count=1`, `CGO_ENABLED=1 go vet ./...`, build Go, testes UI (20 arquivos/199 testes), build UI, typecheck Expo mobile e teste Browser Operator isolado. A nova revisão pública ainda precisa concluir no GitHub; o resultado remoto deve ser associado ao SHA desta iteração, não ao workflow histórico.

Essas correções não convertem adapters em integrações externas conectadas. Continuam pendentes, por dependerem de autoridade, contas, ambientes ou dispositivos reais, os testes distribuídos PostgreSQL/RLS, Redis e OTLP, OAuth/SSO contra IdP, Composio, xAI, Desktop Commander Remote, canais sociais e marketplaces, GPU/modelos multimídia, runners físicos, assinatura/lojas e deploy externo.


## Slice P0 de ownership e invariantes Company — 2026-09-22

O primeiro slice P0 após a auditoria ampliada fechou uma fronteira específica, não a auditoria inteira. Projetos Builder agora carregam `organization_id`; listagem, lookup, mutação visual, undo/redo, preview, export, publish, deploy e preview de arquivo passam por lookup scoped no servidor. O entry precisa existir, nomes usados em templates HTML são escapados e preview/export recusam symlinks que resolvam fora do projeto. O Company OS passou a criar empresas por DTO allowlisted, resetando ID, status, coleções, gasto, risco e timestamps server-managed; gasto de agente pausado ou acima do budget falha sem incrementar o valor gasto.

A evidência reproduzível desta slice é o teste HTTP `server/builder_scope_test.go` com duas organizações, que confirma listagem vazia e `403` sem mutação para o tenant errado; testes de Builder para entry ausente, nome com XSS, symlink e organização; e testes Company para create privilegiado e gasto atômico. Os gates completos passaram: integrity guard, `CGO_ENABLED=1 go test ./... -count=1`, `go vet`, build Go, Vitest (20 arquivos/199 testes), build UI e typecheck mobile.

Ainda não é produção-ready. Permanecem P0 em orchestration/traces/devices/pairing e demais objetos, approvals com separação forte de aprovador, sandbox/MCP process isolation, SSRF/DNS rebinding e DLP, lifecycle pausado de todas as mutações Growth/Social, bearer/redirect/session handling, CI/release e validações distribuídas/externas. A classificação continua **preview/local RC em hardening**.


## Slice P0 de orchestration, traces e devices — 2026-09-22

A segunda slice pós-auditoria adicionou `organization_id` aos orchestration jobs e propagou o tenant aos spans de missão e ferramenta. As rotas de criar, ler, executar e cancelar orchestration, bem como listagem global de traces, agora usam o tenant da requisição. Devices e pairing passaram a filtrar listagem, heartbeat e revoke por organização; o código de pairing fica vinculado à organização que o criou e não pode ser sobrescrito por outro tenant. Regressões negativas confirmam `403` e ausência de mutação para reads, cancelamento, revoke e heartbeat cross-tenant.

A evidência inclui `server/p0_scope_test.go`, testes de Orchestrator, TraceStore e DeviceStore, além dos gates completos: integrity guard, `CGO_ENABLED=1 go test ./... -count=1`, `go vet`, build Go, Vitest 20/199, build UI e typecheck mobile. O handshake WebSocket continua autenticado pelo device token; mTLS/TLS, pairing físico e testes em companions reais permanecem dependências externas.

Esta slice não encerra a auditoria. Continuam P0 em plugins/MCP/skills/artifacts, approvals com separação forte de aprovador, sandbox/process isolation, SSRF/DNS rebinding e DLP, lifecycle pausado de todas as mutações Growth/Social, sessão segura no renderer e release supply chain. A classificação permanece **preview/local RC em hardening**.


## Slice P0 de approvals de missão — 2026-09-22

A terceira slice pós-auditoria endureceu approvals de missão. Quando autenticação está ativa, somente `owner` ou `admin` da organização pode decidir; a decisão passa por `DecideApprovalForActorCAS`, exige a versão corrente da missão e consome o nonce emitido para aquela aprovação. Replays, nonce incorreto, organização errada, razão vazia e decisão já consumida são rejeitados. O Agentic Console passou a enviar o nonce retornado pela API.

A evidência inclui teste de CAS/nonce no Runtime, teste de role policy no servidor e os gates completos: integrity guard, `CGO_ENABLED=1 go test ./... -count=1`, `go vet`, build Go, Vitest 20/199, build UI e typecheck mobile. O modo local sem autenticação continua disponível para instalação loopback e não deve ser confundido com autorização multiusuário.

Esta slice cobre approvals de missão, não todas as decisões do Company/Growth/Social OS. Endpoints que ainda aceitam campos booleanos de aprovação para campanhas, pedidos, drafts ou gasto permanecem pendência P0/P1 e não devem ser tratados como autoridade auditável para efeitos externos.


## Slice P0 de approvals auditáveis do Company OS — 2026-09-22

Campanhas, programas de afiliados, pedidos de dropshipping e drafts sociais agora criam uma decisão server-side `CompanyApproval` com organização, recurso, policy, nonce, expiração, actor, razão e status. Os endpoints de approve localizam o approval pendente no servidor, exigem owner/admin quando auth está ativa, validam nonce e versão da Company e só então projetam `approved=true`/estado do recurso. Regressões cobrem nonce incorreto, replay, actor/org e HTTP.

Os gates completos passaram: integrity guard, `CGO_ENABLED=1 go test ./... -count=1`, `go vet`, build Go, Vitest 20/199, build UI e typecheck mobile. O campo booleano `approved` continua no schema apenas como projeção legada; o cliente não é mais a autoridade para aprovar. `RecordSpend`/gasto da Company ainda aceita uma rota distinta com booleano e permanece P0 aberto para migrar ao mesmo approval ledger.


## Slice P0 final de approvals de gasto — 2026-09-22

A rota de gasto foi migrada para `RecordSpendRequest`: quando a política exige aprovação, a solicitação cria um approval `spend` pendente e retorna `202 Accepted` sem debitar o budget. A fila `CompanyApprovalQueue` permite owner/admin decidir com nonce e CAS por endpoint genérico; somente após decisão aprovada o valor é contabilizado. Rejeição, replay, organização incorreta, versão conflitante e excesso de budget não produzem débito parcial.

Os gates completos após a correção de contrato TypeScript passaram: integrity guard, `CGO_ENABLED=1 go test ./... -count=1`, `go vet`, build Go, Vitest 20/199, build UI e typecheck mobile. O método interno legado `RecordSpend(..., approved bool)` permanece apenas para compatibilidade de domínio/testes; a rota HTTP e a UI não o utilizam mais nem aceitam essa autoridade do cliente.


## Slice P0 de DLP em resultados e observabilidade — 2026-09-22

`RedactValue` passou a redigir recursivamente strings, mapas e listas, além de valores sob chaves sensíveis como `token`, `secret`, `password`, `api_key` e `private_key`. Foram adicionados padrões para PEM, GitHub, OpenAI, OpenRouter, xAI, AWS, Slack, Bearer e assignments de credenciais. O runtime aplica a redação a `Step.Result`, erros de step/mission e payloads de eventos; TraceStore protege atributos, nomes e erros; JSONStore e PostgresStore protegem serializações persistidas.

Testes negativos injectam tokens em resultado de tool, evento, trace e missão persistida e verificam que nenhum token cru reaparece. Os gates completos passaram: integrity guard, Go tests/vet/build, Vitest 20/199, UI build e mobile typecheck. Payloads de saída para connectors/MCP ainda exigem uma política separada para distinguir segredo operacional autorizado de dado sensível do usuário; essa lacuna não foi declarada resolvida.


## Slice P0 de egress Remote MCP — 2026-09-22

O transporte Remote MCP agora desabilita proxy ambiental, mantém redirects somente no mesmo origin permitido e verifica o endereço IP efetivamente conectado pelo socket TCP. Um hostname que resolver ou rebinding para loopback, privado, link-local, multicast ou unspecified é rejeitado no dialer externo; loopback explícito continua disponível apenas para servidores locais. Testes cobrem redirect same-origin e conexão privada real, além dos allowlists existentes.

Os gates completos passaram: integrity guard, Go tests/vet/build, Vitest 20/199, UI build e mobile typecheck. A prova é local; ainda são necessários testes de rede distribuída, TLS/certificados, DNS controlado e auditoria do egress de Media/Connectors para classificar a plataforma como production-ready.


## Slice P0 de MCP stdio lifecycle — 2026-09-22

MCP stdio agora exige caminho absoluto para um executável regular não-symlink, limita quantidade/tamanho de argumentos, usa ambiente mínimo com apenas variáveis explicitamente allowlisted e sempre inicia em diretório de trabalho explícito privado. Quando o diretório não é informado, um workspace temporário é criado e removido no stop; após timeout/cancelamento ele é recriado para permitir restart limpo. Requests e responses têm limite de 4 MiB, stderr fica limitado a 64 KiB e é redigido antes de aparecer em erros.

Em Unix, o processo é iniciado em grupo próprio e o stop tenta encerrar apenas esse grupo, com fallback seguro ao processo individual; Windows usa o kill nativo. Testes cobrem comando relativo, symlink, workspace cleanup, payload limit, cancelamento e restart. Os gates completos passaram: integrity guard, Go tests/vet/build, Vitest 20/199, UI build e mobile typecheck.

A classificação permanece **contenção best-effort**, não sandbox forte. Ainda faltam seccomp, cgroups, rlimits, limites de PID/memória/CPU e validação física multi-plataforma para afirmar isolamento forte.


## Slice P0 de Media egress/download — 2026-09-22

O MediaManager passou a usar transport sem proxy ambiental, dialer com verificação do IP efetivamente conectado e redirects desabilitados. Downloads de provider aceitam HTTPS ou HTTP explícito em loopback, rejeitam userinfo/fragments, fazem leitura bounded com detecção de overflow e validam `Content-Type` contra a extensão e magic bytes de PNG, JPEG, WebP, MP4 e WAV antes de materializar o artefato. Respostas JSON e áudio também têm limites explícitos.

Regressões locais cobrem redirect, MIME mismatch, magic inválido, payload acima do limite e socket privado. Os gates completos passaram: integrity guard, Go tests/vet/build, Vitest 20/199, UI build e mobile typecheck. O upload de transcrição ainda é limitado por tamanho de entrada, e testes de rede distribuída/TLS real continuam necessários.


## Slice P0 de Connector egress parity — 2026-09-22

Connectors agora aplicam transport sem proxy ambiental também quando o client é substituído por um transport HTTP customizado, preservam o dialer que verifica o IP efetivamente conectado, bloqueiam redirects em cada request e limitam request body a 1 MiB e response body a 2 MiB com detecção de overflow. Regressões cobrem redirect, resposta oversized e request oversized, além dos testes de OAuth tenant-scoped existentes.

Os gates completos passaram: integrity guard, Go tests/vet/build, Vitest 20/199, UI build e mobile typecheck. O contrato de egress não redige credenciais operacionais autorizadas; ainda é necessário separar classificação de dados de usuário, injeção de segredo e payload externo, além de validar uploads e redes distribuídas reais.


## Slice P0 de OAuth redirect URI hardening — 2026-09-22

OAuth providers agora carregam `OLLAMA_AGENT_OAUTH_<PROVIDER>_REDIRECT_URIS` como allowlist separada por vírgula, ponto e vírgula ou linha, e só aceitam no start/callback uma URI canônica exatamente presente nessa lista. Redirects exigem HTTPS; HTTP só pode ser usado para loopback quando `..._ALLOW_LOOPBACK_REDIRECT=true` e a URI loopback também está na allowlist. Userinfo, fragmentos e URIs opacas são rejeitados. O AuthStore aplica sintaxe segura e grava a forma canônica no estado PKCE, reduzindo mismatch e evitando confiança em um valor arbitrário do request.

Testes cobrem host não allowlisted, HTTP externo, fragmento, userinfo, loopback explícito e estado inseguro. Os gates completos passaram: integrity guard, Go tests/vet/build, Vitest 20/199, UI build e mobile typecheck. A validação de issuer/audience/nonce OIDC já existente permanece separada; endpoints OAuth ainda precisam ser alinhados ao mesmo egress/IP/redirect policy em uma slice posterior.


## Slice P0 de session handling web/mobile — 2026-09-22

O cliente web agentic deixou de ler bearer e organização de `localStorage`: a sessão é mantida somente em memória por `setAgentSession`, pode ser removida por `clearAgentSession` e limpa automaticamente em respostas 401/403, emitindo evento local para a UI reagir. Testes verificam que o header é formado sem acessar localStorage, que token vazio é rejeitado e que sessão expirada é apagada.

No mobile, 401/403 removem o token do Expo SecureStore sem enfileirar mutações não autorizadas; approvals enviam nonce, ficam desabilitados enquanto uma decisão está em andamento e receberam labels de acessibilidade. Logout exige confirmação quando há missão/cache/outbox e, após confirmação, remove sessão, missão, eventos, push marker e ações offline. A validação de armazenamento seguro nativo depende de build/dispositivo real; a UI web ainda requer uma jornada de login que chame `setAgentSession` em vez de persistir credenciais.


## Slice P0 de OAuth endpoint egress — 2026-09-22

Discovery OIDC, JWKS, userinfo e token exchange passaram a usar um client seguro que remove proxy ambiental, bloqueia redirects e verifica o IP efetivamente conectado, aceitando loopback apenas quando o endpoint configurado é explicitamente loopback. URLs de endpoints rejeitam credentials/fragments e exigem HTTPS; discovery valida issuer e também os endpoints de autorização, token, JWKS e userinfo retornados antes de usá-los.

Regressões cobrem redirect OAuth e socket privado real, além de validação de endpoint existente. Os gates completos passaram: integrity guard, Go tests/vet/build, Vitest 20/199, UI build e mobile typecheck. Um IdP real/staging ainda é necessário para validar discovery, assinatura, audience, nonce, rotação e refresh ponta a ponta.


## Slice P0 de contenção best-effort de tools — 2026-09-22

`terminal.exec` e `sandbox.exec` agora iniciam processo em grupo próprio e encerram o grupo ao cancelar ou atingir timeout, evitando deixar filhos órfãos no caminho comum. Saídas stdout/stderr são limitadas e passam por `RedactDLP` antes de retornar; o resultado informa a postura `best-effort-process-group` ou `best-effort-unshare`. O sandbox `unshare` aplica, quando suportado pelo shell/kernel, limites best-effort de CPU, memória virtual, processos, descriptors e tamanho de arquivo via `ulimit`, além de continuar com user/mount/PID/network namespaces e ambiente mínimo.

Testes cobrem redaction de stderr, status explícito, cancelamento rápido e encerramento do grupo; a suíte `internal/agent` e os gates completos passaram. Isso não é sandbox forte: seccomp, cgroups, enforcement robusto de PID/memória/CPU e validação real em Windows/macOS/Linux continuam pendentes. O produto deve mostrar essa limitação ao operador em vez de alegar isolamento equivalente a E2B.


## Slice P0 de ownership de plugins/MCP/skills — 2026-09-22

Connector, MCP stdio, Remote MCP e Skill manifests agora possuem `organization_id` quando são tenant-owned. Catálogos autenticados mostram recursos tenant-owned da organização ativa e preservam recursos sem owner como configuração global read-only; operações enable/disable/remove autenticadas exigem ownership exato e retornam `403` para outro tenant ou para recurso global, sem mutação. O modo local sem autenticação continua usando lifecycle compatível no bind loopback.

Regressões criam recursos em `org-a` e `org-b`, confirmam filtragem de lista e rejeitam mutation cross-tenant para Connector, MCP, Remote MCP e Skill. Os gates completos passaram: integrity guard, Go tests/vet/build, UI Vitest/build e mobile typecheck. Ainda faltam endpoints server-owned para registrar novos recursos por organização, assinatura/attestation de skills, isolamento de execução MCP por tenant e prova distribuída com Postgres/RLS.


## Slice P0 de CI/release quality gates — 2026-09-22

O workflow `dz23-agentic-quality` foi corrigido para YAML válido, removeu `npm ci` sem lockfile do mobile, e passou a executar typecheck, Vitest e build de produção da UI além de typecheck mobile, integrity, suíte Go completa, vet e build. O Browser Operator mantém instalação e fallback explícito do Chromium. O workflow de release ganhou um job `quality` independente com esses gates e todos os builds/publicação dependem dele; isso bloqueia publicação de tag quando a qualidade falha.

A validação local confirmou parser YAML, integrity guard e `git diff --check`. Esta sandbox não executa GitHub Actions, Docker distribuído, builds físicos macOS/Windows nem signing. SBOM/provenance/attestation continuam dependentes do runner e das credenciais/configurações de release; não foram declarados como concluídos localmente.


## Slice P1 de Grok/provider contract — 2026-09-22

O cliente Grok agora rejeita modelos fora de uma allowlist server-configured (`OLLAMA_AGENT_GROK_MODELS`, com o modelo padrão como fallback), e `Probe` só marca o provider como saudável quando o catálogo remoto contém pelo menos um modelo configurado. A rota HTTP `/api/agent/v1/grok/responses` rejeita `stream:true` com `501 Not Implemented` antes de chamar o upstream, evitando declarar streaming de produção quando o handler não expõe SSE; a API interna `StreamResponses` permanece testada para uma futura rota dedicada.

Testes cobrem modelo arbitrário, stream rejeitado, catálogo compatível/incompatível, allowlist por ambiente e não chamada ao upstream. Gates completos Go/integrity/build/UI/mobile passaram. Nenhuma credencial xAI foi usada ou declarada conectada; health real contra xAI continua dependente de credencial válida e ambiente autorizado.


## Slice P0 de Compose/infrastructure defaults — 2026-09-22

A composição agentic passou a exigir `OLLAMA_AGENT_POSTGRES_PASSWORD` e `OLLAMA_AGENT_REDIS_PASSWORD`, usar role PostgreSQL não-superuser, Redis com `requirepass` e portas PostgreSQL/Redis/OTLP ligadas a `127.0.0.1` por padrão. O init SQL não contém mais credencial fixa. O workflow distribuído cria passwords efêmeras por execução e injeta URLs autenticadas nos testes.

O integrity guard e `git diff --check` passaram. Docker/Compose não está instalado nesta sandbox; `docker compose config` e o teste real PostgreSQL/Redis/OTLP ficaram `NOT_RUN_DOCKER_UNAVAILABLE`, não sendo tratados como prova local. O GitHub Actions continua sendo a prova autoritativa dessa integração.


## Slice P1 de provider selection contract — 2026-09-22

A seleção de motor na Nova tarefa agora envia `provider` explicitamente e o runtime persiste essa escolha. Como somente o planner Ollama local está implementado nesta árvore, providers Claude/Codex/OmniRoute/automático aparecem como não conectados na UI e são rejeitados pelo servidor com erro de configuração, em vez de serem enviados silenciosamente ao Ollama como se fossem adapters reais.

O teste negativo server-side e os gates completos Go/integrity/UI passaram. A implementação de adapters externos, renovação OAuth e execução real dos CLIs continuam pendentes e não são declaradas conectadas.


## Slice P1 de release artifact integrity — 2026-09-22

O job final de release passou a rejeitar artefatos ausentes ou vazios antes de gerar checksums. A attestation de provenance pode ser executada somente quando a variável de ambiente GitHub `OLLAMA_ENABLE_ATTESTATIONS=true` estiver configurada, usando a permissão OIDC/attestations; caso contrário, permanece explicitamente não executada. A sandbox não publicou release, assinou binários ou executou Actions, portanto essas provas seguem externas.


## Slice P0 de tenant isolation em jobs/replay — 2026-09-22

Listagem e replay da fila agora são filtrados pela organização proprietária da missão. Um tenant não recebe jobs de outra organização e o replay cross-tenant retorna `403` sem alterar o estado da fila. A regressão HTTP foi adicionada ao teste combinado de escopo.

Gates completos Go/integrity/build/UI/mobile passaram. O isolamento ainda depende de prova distribuída com Redis/PostgreSQL reais e de revisão de outras superfícies de execução, mas jobs/replay não permanecem mais como leitura/mutação global no handler.


## Slice P0 de artifact manifest path safety — 2026-09-22

`BuildArtifactManifest` agora rejeita componentes symlink e resoluções fora do workspace antes de calcular hash, tamanho ou MIME. Foram adicionados testes para symlink externo e arquivo regular.

Gates completos Go/integrity/build/UI/mobile passaram. Isso protege a geração do manifest; a prova de publicação/exportação em todos os runtimes e adapters distribuídos continua separada.


## Remediação do workflow distribuído — 2026-09-22

O workflow `dz23-agentic-quality` executado no commit `2933f6e2` encontrou uma falha real no teste `TestDistributedPostgresRLSAndEvents`: a URL do smoke usava a role `ollama_agent` criada pelo Compose, que é a role bootstrap/superusuária, e o adapter tenant-scoped recusou corretamente operar com ela. O mesmo passo havia colocado passwords efêmeras no `GITHUB_ENV`, fazendo com que os valores aparecessem no bloco de ambiente do log do passo; esses valores eram efêmeros do CI, não credenciais de terceiros, e não foram adicionados ao repositório.

A correção publicada nesta iteração mantém as passwords somente em variáveis locais de um único passo, cria/ajusta uma role separada `ollama_agent_test` sem privilégio de superusuário e executa o smoke RLS usando essa role. O integrity guard também impede o retorno de `GITHUB_ENV` no job distribuído. A prova local desta correção é estática (parser YAML e integrity) e os gates completos locais passaram; a confirmação autoritativa depende de uma nova execução do GitHub Actions, porque Docker não está disponível nesta sandbox.

A classificação permanece **preview/local RC em hardening**. O CI distribuído continua sendo uma dependência de validação remota, não uma prova inventada localmente.


## Slice de workflow distribuído corrigida — 2026-09-22

- Falha observada no run `35735628693`: somente o teste PostgreSQL RLS falhou por uso de superusuário; Redis DLQ, OTLP e qualidade web/mobile passaram naquele run.
- Correção: role não-superusuária dedicada para o teste tenant-scoped e passwords confinadas ao passo de execução.
- Provas locais: integrity guard, YAML parser, `CGO_ENABLED=1 go test ./... -count=1`, `CGO_ENABLED=1 go vet ./...`, build Go, Vitest/build UI e typecheck mobile: PASS.
- Prova ainda pendente: novo run do GitHub Actions no commit desta correção.
- Não foram repetidas nem publicadas credenciais de terceiros; qualquer chave fornecida anteriormente continua considerada exposta e deve ser rotacionada pelo operador.


## Slice de workflow distribuído corrigida — 2026-09-22

O job PostgreSQL RLS deixou de conectar o teste tenant-scoped com a role bootstrap superusuária do Compose. O passo agora cria uma role de teste separada, executa toda a validação com ela e mantém passwords efêmeras em escopo local, sem `GITHUB_ENV`. O guard estático bloqueia regressão dessa prática.

Evidências locais desta correção: `scripts/check-class-a-plus-integrity.sh`, parser dos workflows, `git diff --check`, suíte Go completa com CGO, `go vet`, build Go, 20 arquivos/199 testes Vitest, build Vite e `npm run typecheck` mobile — todos PASS. Docker e a nova execução remota permanecem necessários para fechar o gate distribuído.

A classificação não muda: **preview/local RC em hardening**, sem merge automático em `main` e sem declarar produção-ready.


## Follow-up do CI distribuído — 2026-09-22

A execução `35737050056` confirmou que o job Go, SBOM e Web/Mobile passaram, mas o job distribuído falhou antes do teste RLS: o bloco `DO $$` misturava sintaxe de variável do `psql` com PL/pgSQL e produziu erro de sintaxe. A limpeza também não tinha acesso às variáveis locais do passo de start, porque cada passo do GitHub Actions possui ambiente separado.

O workflow foi corrigido novamente para consultar a existência da role com `psql -c`, executar `CREATE ROLE` ou `ALTER ROLE` com password hexagonal efêmera e aplicar os grants em uma chamada separada. A limpeza agora injeta somente placeholders não secretos, suficientes para o Compose interpolar a configuração e executar `down -v`, sem recuperar nem persistir os secrets do passo anterior.

Esta segunda correção ainda aguarda nova execução remota. A validação local deve cobrir YAML, integrity, diff, Go, UI e mobile; Docker/PostgreSQL/Redis/OTLP permanecem impossíveis de executar nesta sandbox.


## Validação remota do CI distribuído — 2026-09-22

O commit `15e8ae60` foi validado pelo GitHub Actions no run `35737772235`. Todos os jobs passaram: Go agentic/server com Browser Operator, SBOM, Web/Mobile e integração PostgreSQL RLS + Redis DLQ + OTLP. O teste RLS executou com role não-superusuária dedicada; a limpeza Docker terminou sem depender dos secrets do passo de start.

Esta evidência fecha o gate distribuído desta slice, mas não a classificação global do produto. Permanecem P0/P1 internos e dependências externas descritos na matriz e na auditoria; a classificação segue **preview/local RC em hardening**.


## Slice P1 de CapabilityPolicy central — 2026-09-22

A execução de missões passou a usar uma `CapabilityPolicy` central, deny-by-default para descritores sem scopes, grants desconhecidos e scopes não declarados. O runtime valida a policy na inicialização, no planejamento e imediatamente antes da execução de cada tool; approvals agora registram os scopes efetivos junto do risco. O grant padrão foi reduzido a `workspace:read`; a UI oferece `Permitir escrita` como opt-in explícito, mantendo a approval obrigatória para efeitos de escrita.

Evidências locais: `scripts/check-class-a-plus-integrity.sh`, `CGO_ENABLED=1 go test ./... -count=1`, `CGO_ENABLED=1 go vet ./...`, build Go, Vitest completo, build Vite e typecheck mobile — todos PASS. Isso reduz capability overgrant no runtime, mas não substitui sandbox forte, autorização externa por endpoint nem testes físicos/distribuídos.

A classificação permanece **preview/local RC em hardening**; a slice não fecha os P0/P1 restantes nem valida credenciais ou adapters externos.


## Remediação dos gates upstream Go/race — 2026-09-22

A reprodução local confirmou que o `go test` normal passava e que o `go test -race` encontrava duas corridas reais: o contador compartilhado do fixture de pesquisa e o buffer stderr do MCP, cujo `ReadFrom` promovido por `bytes.Buffer` contornava a proteção existente. O teste de pesquisa agora usa `atomic.Int64`; o buffer MCP tem mutex, `String` protegido e `ReadFrom` próprio limitado.

Evidências após a correção: `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, build Go e integrity guard — PASS. A nova execução upstream no head publicado ainda é necessária; o CI agentic anterior havia passado, mas o teste upstream falhou antes desta correção.

A classificação permanece **preview/local RC em hardening**. Esta slice corrige races de teste/lifecycle, mas não fecha sandbox forte, integração externa, device testing ou os demais P0/P1.


## Quality workflow remoto verde — 2026-09-22

O workflow `dz23-agentic-quality` passou no SHA `428a99bd` no run `35749351291`. Os jobs de Go/server, PostgreSQL RLS + Redis DLQ + OTLP, Web/Mobile e SBOM passaram. Esta evidência cobre o workflow agentic do fork; o workflow upstream `test` ainda precisa concluir no head corrente do PR.


## Slice P0 de bootstrap MCP/Remote MCP — 2026-09-22

Os loaders `OLLAMA_AGENT_MCP` e `OLLAMA_AGENT_REMOTE_MCP` agora têm prova de bootstrap real no pacote `server`. Os testes cobrem registro de manifestos válidos, JSON com campo desconhecido, trailing JSON e allowlist remota vazia. A configuração continua opt-in por arquivo e nenhuma conexão externa é declarada.

Evidências: testes focused normal e race, integrity guard e gates Go anteriores — PASS. A matriz upstream `test.yaml` permanece pendente por não ter sido acionada no head atual do PR.


## Slice P1 de correlação/notificações MCP — 2026-09-22

O transporte MCP stdio agora descarta notificações JSON-RPC sem `id` enquanto aguarda a resposta correlacionada ao request. Respostas com `id` ausente, nulo ou diferente continuam sendo tratadas como falha de correlação, evitando aceitar uma resposta de outra operação. A regressão normal e race passou com fixture local.

Isso não implementa multiplexação concorrente, framing completo, sessão Remote MCP ou auditoria persistida por chamada.


## Remediação do Browser Operator no workflow upstream — 2026-09-22

O run upstream `test` no head `e8017591` deixou `test (ubuntu-latest)` e `race (ubuntu-latest)` vermelhos porque o teste `TestBrowserOperatorNavigateAndSnapshot` encontrou `ModuleNotFoundError: No module named 'playwright'`. O workflow `dz23-agentic-quality` do fork permaneceu verde, pois já instala a dependência.

A correção local adiciona Playwright `1.63.0` pinado aos jobs upstream `test` e `race`, instala Chromium com dependências no Linux e sem essa flag em macOS/Windows, e torna o launcher Go portável ao escolher `python3` ou `python` no `PATH`. A matriz nativa Linux/Windows com GPUs foi restringida a `workflow_dispatch` com `run_native_matrix=true`, pois depende de runners compatíveis não disponíveis no PR público. Integrity, YAML, Browser Operator normal/race e gates Go locais passaram. A confirmação remota desta correção ainda está pendente.


## Follow-up do workflow upstream preso — 2026-09-22

O run upstream `35755046119` foi cancelado de forma controlada depois de permanecer com uma matriz nativa Linux/Windows dependente de runners customizados indisponíveis no PR público. Os dois jobs Ubuntu também falharam por causa verificável: a pinagem inicial `playwright==1.53.2` não existia no índice do runner, causando `No matching distribution found` e, em seguida, `ModuleNotFoundError` no Browser Operator.

A correção agora usa Playwright `1.63.0`, corrige o grupo de concorrência e deixa a matriz nativa com GPU disponível somente por `workflow_dispatch` e `run_native_matrix=true`. No caminho normal de pull request permanecem os runners públicos e os gates CPU/test/race aplicáveis. Validação local desta mudança: parser YAML dos três workflows, integrity guard e `git diff --check` passaram. A nova confirmação remota será registrada somente após um run no head publicado.


## Validação remota final do caminho normal de CI — 2026-09-22

A sequência de correções publicada nos commits `5db7261e`, `86a2706b`, `849781af` e `de0e8677` fechou os problemas de lint multiplataforma e de descoberta do Chromium gerenciado pelo Playwright. A prova local inclui integrity, parser YAML, `golangci-lint v2.13.2` com zero issues, suíte Go normal e race, vet, build e compilação cruzada dos pacotes Windows afetados.

No head `de0e86772e96372789c10d924eb5738f8808821b`, o GitHub Actions confirmou `class-a-plus-integrity` no run `35769597628`, `dz23-agentic-quality` no run `35769597363`, `dz23-multi-provider` no run `35769597578` e o workflow upstream `test` no run `35769597404`; todos passaram. O upstream executou testes em Linux, macOS e Windows, além de race em Linux e macOS. O PR consolidou 21 checks bem-sucedidos, 3 skipped e nenhum pending ou failing.

Os skips correspondem a jobs condicionais do workflow upstream e não são tratados como aprovação de hardware. A matriz GPU/nativa continua fora do caminho automático e exige `workflow_dispatch` com `run_native_matrix=true` e runners compatíveis configurados pelo operador. Avisos de migração de Node 20 e `ubuntu-latest` são não bloqueantes.

Esta evidência fecha o caminho normal de CI desta slice, mas não altera a classificação global. O produto permanece **preview/local RC em hardening**, e não production-ready. Ainda faltam sandbox forte e isolamento de processos, autenticação/session/CSRF/IdP end-to-end, revogação OAuth, contratos externos reais de providers/deploy/media, validação física de desktop/mobile, signing/provenance e homologação de lojas ou app review.


## Hardening pós-CI: sessão, Company e terminal — 2026-09-22

Foram publicadas cinco slices coesas no head `935fb273`: o cliente agentic agora diferencia `401 Unauthorized` de `403 Forbidden`, preservando uma sessão válida quando a organização ou a ação é recusada; o contrato do planner foi alinhado ao callback nomeado do cliente Ollama; as mutações críticas de Company passaram a aceitar `Idempotency-Key` com digest sem persistir a chave bruta e rejeitam replays/conflicts; `terminal.exec` restringe argumentos e caminhos ao workspace; e `POST /api/agent/v1/auth/logout` revoga o bearer no `AuthStore` antes de a UI limpar a sessão local.

As regressões locais de agent/server, Company, autenticação, terminal/sandbox e UI passaram. O workflow `class-a-plus-integrity` passou no push `35784205466` e no PR `35784211426`; `dz23-agentic-quality` passou no push `35784205382` e no PR `35784211429`, incluindo Go/server, PostgreSQL RLS + Redis DLQ + OTLP, Web/Mobile e SBOM; `dz23-multi-provider` passou no PR `35784211481`; e o upstream `test` passou no PR `35784211483`, com testes em Linux, macOS e Windows, race em Linux/macOS, patches e `go_mod_tidy`.

Essa evidência fecha os workflows normais observados para o head, não a matriz nativa/GPU. A matriz nativa permanece manual e opt-in, requer runners compatíveis do operador e não foi executada. A classificação continua **preview/local RC em hardening**, não production-ready. Permanecem sandbox forte/seccomp/cgroups enforceable, auth/session/CSRF/IdP distribuído, OAuth lifecycle/revocation externo, providers/deploy/media reais, validação física desktop/mobile, signing/provenance, stores/app review e homologação externa.


## Hardening adicional: sandbox, CSRF, mobile e release — 2026-09-22

Foram publicadas as slices `0459532a`, `e931fcfc`, `a12a15e4`, `d1058209` e `a7bc82f5`. O `sandbox.exec` agora possui um caminho Linux strict opt-in que falha fechado sem cgroup v2 delegado e combina user/mount/pid/network namespaces, `no-new-privs`, filtro seccomp auditado para amd64/arm64, limites cgroup e `cgroup.kill` no timeout. O default continua best-effort; a implementação não equivale à homologação de um host externo nem a AppArmor/SELinux configurado.

Mutations agentic autenticadas com `Origin` fora da allowlist `OLLAMA_ORIGINS` são recusadas antes do Bearer/RBAC. O mobile consulta `auth/session` antes de habilitar ações offline ou push autenticadas e namespacifica cache de missão, outbox e registro de push por servidor/organização, com `Idempotency-Key`, `If-Match`, backoff e conflitos explícitos. Typecheck e Expo web export passaram. O workflow de release agora gera SBOM CycloneDX, metadata do commit/ref, manifesto SHA-256 verificável e attestation condicional; nenhum release tag/assinatura foi executado nesta sandbox.

No head `a7bc82f5`, integrity passou no push `35793674459` e no PR `35793679458`; `dz23-agentic-quality` (`35793679479`) e `dz23-multi-provider` (`35793679448`) ainda estavam em execução, e upstream `test` (`35793679567`) estava queued no momento do registro. Essa evidência é parcial até os runs concluírem.

O pacote mobile reportou 18 vulnerabilidades de produção no `npm audit` (11 moderate e 7 high), ainda sem triagem/remediação de versão. Permanecem dependências externas e não executadas: cgroup real no host do operador, AppArmor/SELinux, IdP/OAuth real, providers/deploy/media, push remoto, dispositivos físicos, instaladores assinados, provenance efetiva, rollback, stores e app review. A classificação permanece **preview/local RC em hardening**, não production-ready.


## Follow-up de dependências mobile — 2026-09-22

A triagem do audit mobile foi concluída sem force upgrade. Os achados transitivos de `image-size`, `postcss` e `uuid` receberam overrides mínimos compatíveis no package lock (`2.0.4`, `8.5.28` e `11.1.1`); `npm run typecheck`, Expo web export e `npm audit --omit=dev` passaram, com `0` vulnerabilidades de produção no estado atual. Isso não substitui uma migração futura de Expo/React Native major nem testes físicos Android/iOS.


## Validação final do caminho normal de CI — head e6e0632b — 2026-09-22

O head `e6e0632ba5ff0495ed4b061221c90696478b3d67` passou todos os workflows normais do PR: upstream `test` `35794443450`, `class-a-plus-integrity` `35794443307`, `dz23-multi-provider` `35794443300` e `dz23-agentic-quality` `35794443501`. O upstream confirmou Linux, macOS e Windows, race em Linux/macOS, patches e `go_mod_tidy`; o workflow agentic confirmou Go/server, Browser Operator, PostgreSQL RLS + Redis DLQ + OTLP, Web/Mobile e SBOM. O PR reporta 21 checks bem-sucedidos, 3 skipped, 0 failing e 0 pending.

O patch também trocou instalações Node não determinísticas por `npm ci` nos gates web/mobile, mantendo a cadeia reproduzível pelos lockfiles. O audit de produção mobile permanece em zero após os overrides transitivos. Os três jobs `test/linux`, `test/windows` e `test/go_license` skipped não são aprovação de hardware nem licença; a matriz GPU/nativa continua manual/opt-in.

Esta evidência fecha o caminho normal de CI, não a prontidão de produção. Continuam abertos host sandbox realmente homologado, AppArmor/SELinux, auth/IdP/OAuth e providers externos, deploy/media/marketplaces, push remoto, dispositivos físicos, signing/provenance, rollback, stores/app review e homologação externa. A classificação permanece **preview/local RC em hardening**, não production-ready.


## V5 — catálogo de connectors, provider efetivo e CI multiplataforma — 2026-09-22

A rodada V5 fechou lacunas internas observáveis no caminho agentic. O planner agora usa o provider/modelo efetivamente solicitado e falha fechado quando não há executor configurado; o fallback determinístico silencioso foi removido. O estado durável do runtime fica separado do workspace de execução. O Remote MCP fixa os IPs aprovados após a resolução DNS. O catálogo de connectors expõe estados tenant-aware e somente o booleano `credential_configured`; a UI de Plugins pesquisa o catálogo e o seletor do Mission Console deriva providers/modelos do catálogo real, sem aliases que pareçam conectados.

Também foi corrigido o próprio guard de integridade para proteger o contrato dinâmico. O primeiro upstream após essa mudança apontou um helper Go morto, corrigido em `93a0115f`. Uma segunda execução revelou `EEXIST/ENOENT` no cache npm global do runner Windows; `0f95b6a1` passou a usar cache por runner e `fail-fast: false` nas matrizes normal/race. A nova execução confirmou Linux, macOS e Windows, race em Linux/macOS, `go_mod_tidy` e patches.

No head `0f95b6a1969462fb309e00e29813e8136d15f757`, passaram `class-a-plus-integrity` (`35804229189`), `dz23-agentic-quality` (`35804229301`), `dz23-multi-provider` (`35804229204`) e upstream `test` (`35804229207`). O PR consolidou 21 checks successful, 3 skipped, 0 failing e 0 pending. Os gates locais desta sequência passaram Go test/vet/build, lint, UI, mobile typecheck, `npm audit` com zero vulnerabilidades, YAML e integrity.

Essa evidência não transforma catálogo em integração externa conectada. Composio, Google Workspace, GitHub, Woovi/OpenPix, fiscal/NF-e, social commerce, marketplaces, deploy, mídia e providers externos continuam dependentes de contas, credenciais, escopos, aprovação e smoke autorizado. A matriz GPU/nativa permanece manual e opt-in. A classificação continua **preview/local RC em hardening**, não production-ready; permanecem abertos host sandbox homologado, IdP/OAuth distribuído, dispositivos físicos, push remoto, signing/provenance, rollback, stores e app review.


## Addendum pós-V5 e lifecycle de connectors — 2026-09-22

O head `0da6be80` concluiu o job normal `test (macos-latest)` `107015911723` com sucesso. A causa do erro anterior não era uma corrida: em macOS, `/var` pode ser um alias de sistema para `/private/var`, e a validação tratava esse ancestral como se o diretório final do workspace fosse um symlink. A correção passou a usar `os.Lstat` no componente final, preservando a rejeição de componentes descendentes inseguros. O teste de defaults do Runtime também passou a consultar o diretório de configuração efetivo da plataforma, em vez de assumir a semântica XDG do Linux.

A evidência pública do head portátil é: integrity PR run `35808941807` PASS; `dz23-agentic-quality` run `35808941824` PASS; multi-provider run `35808941746` PASS; upstream `test` run `35808941810` SUCCESS, incluindo race macOS `107015880499`, race Ubuntu `107015880534`, normal Ubuntu `107015911678`, normal Windows `107015911689` e normal macOS `107015911723`. Isso fecha a matriz pública referente a `0da6be80`, não a validação externa de providers, dispositivos ou produção.

O commit seguinte `411335ba6247b16a431c7f10b5daf8a9fcc0e8f4` adiciona lifecycle de connectors por organização. Quando nenhum manifesto estático `OLLAMA_AGENT_CONNECTORS` é definido, o runtime carrega e persiste `OLLAMA_AGENT_STORE/connectors.json`. O arquivo é escrito com modo `0600`, atualização temporária seguida de rename e somente metadados seguros: endpoint HTTPS, operações allowlisted, nome de variável de ambiente e identificador OAuth. O valor de token nunca é aceito pelo endpoint, persistido pelo manifest ou devolvido pela API. `POST /api/agent/v1/connectors` exige owner/admin em auth mode, força o `organization_id` do contexto autenticado e rejeita campos desconhecidos, segredos crus e colisões cross-tenant. A UI agora expõe o cadastro usando apenas referências de ambiente/OAuth.

Os testes locais da slice passaram para persistência, rollback, redaction, admin/member, cross-tenant e build UI. No momento do checkpoint, os checks da PR do novo SHA ainda estavam pendentes; portanto não são tratados como verdes. A configuração de `OLLAMA_AGENT_CONNECTORS` continua sendo um modo de manifesto estático explícito e não deve ser interpretada como lifecycle durável da UI.

A classificação continua **preview/local RC em hardening**, não final e não production-ready. Ainda faltam persistência/lifecycle equivalente para MCP e skills, isolamento físico forte do sandbox, testes distribuídos reais, OAuth e contas externas, runners/dispositivos físicos, GPU, assinatura, lojas, app review e deploy autorizado.


## Addendum de lifecycle MCP/Remote MCP/skills — 2026-09-22

O commit `96fd7edd3440876ed67ae7051cc475ca953d2847` adicionou o segundo lifecycle durável da plataforma. O runtime padrão agora carrega e atualiza `OLLAMA_AGENT_STORE/mcp.json` para MCP stdio, `OLLAMA_AGENT_STORE/remote-mcp.json` para Remote MCP e `OLLAMA_AGENT_STORE/context/skills/*.json` para manifestos de skills. As escritas usam o writer atômico existente, diretórios privados e arquivos `0600`. O bootstrap por `OLLAMA_AGENT_MCP`, `OLLAMA_AGENT_REMOTE_MCP` ou `OLLAMA_AGENT_CONNECTORS` continua deliberadamente estático; esses managers não apresentam mutations da UI como edição persistente do arquivo de ambiente.

As rotas `POST /api/agent/v1/mcp`, `POST /api/agent/v1/remote-mcp` e `POST /api/agent/v1/skills` exigem owner/admin no modo autenticado, vinculam a organização no servidor e rejeitam conflito de tenant. MCP exige executável absoluto, arquivo regular executável, allowlist de métodos e nomes de env válidos. Remote MCP mantém HTTPS fora de loopback, bloqueio de private/link-local, pinning DNS por request, redirects no mesmo origin e headers referenciados por env. Skill registration não aceita `trusted` nem `enabled` como autoridade: o servidor grava `trusted=false` e inicia habilitado para revisão/approval. O frontend possui formulários correspondentes e não coleta tokens.

Os testes incluem round-trip após restart, modo `0600`, colisão cross-tenant, remoção/rollback, trust fail-closed, authorization owner/admin, raw-secret/unknown-field rejection, normal e race. O runner local completo passou: integrity, YAML, Go test, vet, build, Vitest, typecheck, Vite build, mobile typecheck, npm audit e diff. Os checks remotos do novo commit estavam recém-enfileirados no instante desta nota e não são tratados como finais.

A classificação permanece **preview/local RC em hardening**, não final e não production-ready. Registrar um MCP, Remote MCP ou skill não executa ação externa, não consente OAuth, não prova uma conta ou upstream e não fornece isolamento físico além das políticas efetivamente provisionadas no host.


## Addendum de lifecycle UI — 2026-09-22

No commit `eb97e6b8`, a tela Plugins passou a cobrir também remoção de connectors, MCP/Remote MCP e skills. O botão exige confirmação local e a mensagem informa que a operação remove o manifest do runtime, mas não revoga uma credencial ou conta no serviço upstream. A validação local concluiu build TypeScript/Vite e 204 testes Vitest.

Esse complemento melhora a operação local, mas não muda os blockers de OAuth, upstream smoke, sandbox físico, devices, signing, stores, app review ou deploy. O produto segue **preview/local RC em hardening**, não final e não production-ready; os checks remotos do head mais recente ainda devem ser consultados quando concluírem.


## Addendum de parsing estrito de manifestos — 2026-09-22

O commit `89b4203e` corrigiu um hardening P1 nos carregadores duráveis de MCP e Remote MCP. Após decodificar o array principal, o loader agora exige fim real do documento JSON. Um manifest como `[] {}` é rejeitado em vez de ser aceito parcialmente. Isso reduz risco de configuração ambígua ou de dados anexados após um documento válido.

Os testes normal e race de persistência passaram. O gate completo Go também passou: integrity, todos os pacotes em `go test`, `go vet`, `go build` e `git diff --check`. A mudança não altera o modo de bootstrap estático nem converte falhas externas em sucesso. O produto continua preview/local RC em hardening, não final e não production-ready.


## Addendum de harmonização dos loaders — 2026-09-22

A mesma proteção de EOF foi aplicada ao manifest persistente de connectors no commit `5816c020`. A plataforma agora rejeita documento JSON trailing nos três loaders duráveis: connectors, MCP e Remote MCP. O caso `[] {}` é coberto por regressões e não é tratado como configuração parcial válida.

Após a mudança, passaram os testes normal/race do connector manager e os gates Go completos: integrity, todos os testes, vet, build e diff. O estado continua preview/local RC em hardening; não há mudança nos blockers de OAuth, upstream, sandbox físico, devices ou produção.


## Addendum de bootstrap estático strict — 2026-09-22

O commit `f78fa0d6` alinhou o loader de `OLLAMA_AGENT_CONNECTORS` ao contrato estrito dos demais manifests. Campos desconhecidos e conteúdo JSON após o array principal agora são rejeitados antes do registro de qualquer connector. Os testes normal/race do loader e os gates completos Go passaram em integrity, test, vet, build e diff.

Esse hardening melhora parsing e previsibilidade local. Ele não transforma o manifest em conexão OAuth, não valida conta upstream e não altera os blockers físicos/distribuídos. O estado permanece **preview/local RC em hardening**, não final e não production-ready.


## Addendum de strict decode de skills — 2026-09-22

O commit `86300900` completa a política de parsing seguro para skills. O loader de manifestos rejeita campos desconhecidos e JSON trailing antes de aplicar qualquer entrada. A confiança continua fail-closed: o payload não pode promover `trusted`, e o servidor deriva `enabled`.

Com essa mudança, a família de plugins — connectors, MCP, Remote MCP e skills — possui regressões para campos desconhecidos/trailing nos caminhos relevantes. Os gates Go completos passaram em integrity, testes, vet, build e diff. O produto segue preview/local RC em hardening e não afirma upstream, OAuth, hardware, sandbox físico ou produção.


## Addendum de logout local e revogação autenticada — 2026-09-22

O commit `0ea1039a` corrigiu o contrato de `POST /api/agent/v1/auth/logout`. Com `auth_required=false`, o endpoint agora é idempotente e retorna `204` sem exigir bearer ou AuthStore. Com auth requerida, o middleware autentica primeiro e o handler revoga o bearer apresentado; o teste existente continua cobrindo que o token revogado não autentica novamente.

A correção evita um falso requisito de sessão no modo local e um possível acesso a store ausente. Testes normal/race e gates Go completos passaram. Isso cobre somente a sessão do runtime local; não é logout ou revogação de contas OAuth externas. A classificação continua preview/local RC em hardening.


## Addendum de Origin/CSRF no modo local — 2026-09-22

O commit `79a1197e` aplica `agentOriginAllowed` também quando `auth_required=false`. Isso mantém o uso local sem bearer, mas bloqueia mutações iniciadas por uma origem cross-site. Os defaults de loopback permanecem permitidos, e GET/HEAD, OPTIONS e requests sem Origin não são transformados em falhas de navegador.

A regressão de middleware e os gates Go completos passaram. A proteção é uma barreira de navegador do runtime local; não substitui autenticação distribuída, IdP, mTLS ou homologação de host. O estado permanece preview/local RC em hardening.


## Addendum de wildcard de Origin — 2026-09-23

O commit `42006074` corrigiu o matcher de allowlist. Wildcards terminados em `:*` agora significam somente qualquer porta do hostname exato; não há prefix match de domínio. Userinfo, path, query e fragment são rejeitados nesse formato. Os wildcards explícitos de esquema (`app://*`, `file://*` e equivalentes já suportados) permanecem limitados ao esquema configurado.

A regressão cobre `trusted.example:8443`, `trusted.example.evil:8443` e ausência de porta. Os gates Go completos passaram. O estado permanece preview/local RC em hardening, sem claim de autenticação distribuída ou homologação de browser externo.


## Addendum de portabilidade do sandbox — 2026-09-23

O commit `b4c4c243` tornou `sandbox.exec` utilizável fora do Linux sem vender isolamento inexistente. macOS e Windows resolvem o interpretador disponível e executam em processo best-effort, com timeout, limite de saída e encerramento no cancelamento. A resposta marca `network_isolation=not-enforced`. No Linux, o caminho best-effort conserva a tentativa de namespaces; `strict` continua exigindo Linux, cgroup v2 delegado, namespaces, no-new-privs e seccomp, falhando fechado fora desse ambiente.

Os testes normal/race, gates Go completos e compilações de `internal/agent` para Darwin e Windows passaram. Isso não substitui testes físicos, AppArmor/SELinux, homologação de host ou isolamento forte do operador. O produto continua preview/local RC em hardening.


## Addendum de request JSON estrito — 2026-09-23

O commit `f06891fc` tornou `decodeJSON` estrito até o fim do body. Além de `DisallowUnknownFields`, o helper agora rejeita um segundo documento JSON ou dados inválidos depois do primeiro. Isso se aplica transversalmente aos handlers que reutilizam o parser agentic.

As regressões normal/race e os gates Go completos passaram. Esse hardening reduz ambiguidade de requests, mas não substitui auth, approval, DLP, SSRF ou validações específicas de cada recurso. O estado continua preview/local RC em hardening.


## Addendum de limite de body JSON — 2026-09-23

O commit `09eb26eb` adicionou limite central de 4 MiB ao `decodeJSON`. Requests agentic agora precisam conter um único documento JSON, sem campos desconhecidos, sem trailing e dentro do orçamento de bytes. A regressão de body oversized e os gates Go completos passaram.

Esse limite é um guardrail de transporte; endpoints continuam sujeitos a validações de schema, autorização, approval e limites específicos de payload. O produto permanece preview/local RC em hardening.


## Addendum de dev token estrito — 2026-09-23

O endpoint de desenvolvimento `POST /api/agent/v1/auth/dev/token` foi alinhado ao `decodeJSON` no commit `6d00127a`. Ele agora compartilha limite de 4 MiB, rejeição de campos desconhecidos e EOF. A emissão continua exposta somente com flag de desenvolvimento explícita e peer loopback.

A regressão normal/race e os gates Go completos passaram. Esse endpoint continua sendo uma ferramenta de desenvolvimento local, não onboarding de produção ou autenticação externa.


## Addendum de deploy seguro e não validado externamente — 2026-09-23

O commit `58cc1f40` adicionou proteção ao adapter de deploy. Roots symlink são rejeitados, redirects são desabilitados e a conexão padrão bloqueia endereços privados depois do DNS para endpoints externos. HTTP permanece permitido somente para loopback; serviços remotos precisam de HTTPS.

O adapter possui smoke determinístico com servidor fixture e limites de arquivo, mas não houve deploy real. Vercel, Netlify, AWS, Cloudflare e outros continuam dependentes de credenciais, contas, permissões, custos, aprovação e rollback do operador. O produto permanece preview/local RC em hardening.


## Addendum de segurança multimídia — 2026-09-23

O commit `f45aae49` adicionou containment e proteção de symlink para inputs de áudio/imagem e para outputs no workspace. Arquivos de transcrição acima de 100 MiB são rejeitados antes da leitura, sem truncamento silencioso. Testes normal/race e gates Go completos passaram.

Isso valida o comportamento do adapter e de um provider fixture local. Não valida modelos hospedados, GPU, quota, billing, conta externa, moderação ou qualidade de geração; esses requisitos continuam externos e não production-ready.


## Addendum de Company Growth sandbox-only — 2026-09-23

O commit `a286c15a` tornou explícito o campo `mode: sandbox` em campaigns, afiliados e pedidos. O relatório expõe `sandbox_only=true`, e modos externos são recusados sem adapter validado. Approval, budget, inventory, idempotência e métricas continuam testáveis localmente.

Isso não representa publicação em rede social, checkout, gateway de pagamento, pedido em marketplace, fulfillment, afiliado com comissão real ou emissão fiscal. Esses fluxos continuam bloqueados por credenciais, contas, escopos, homologação e smoke externo.


## Addendum de safeConfig durável — 2026-09-23

O commit `fda91625` faz o endpoint de configuração segura reconhecer connectors, MCP, mídia e deployments registrados nos managers persistentes, mesmo quando não existe variável de bootstrap. O teste confirma o estado configurado sem expor endpoint.

`configured` continua significando apenas manifest/manager aceito. Não significa credencial presente, OAuth consentido, conta conectada, provider respondendo ou deploy publicado.


## Addendum de reauditoria README, mídia, deploy e captura — 2026-09-23

O HEAD `06990ef5` corrigiu quatro achados de segurança reproduzidos no pacote real. A mídia não grava mais por `.agent-media` symlink intermediário: outputs usam `os.Root`, paths relativos e rejeição de componentes symlink. A escrita é limitada e o destino é validado antes de provider ou processo externo. Transcrição e visão abrem o descritor dentro do root e leem com orçamento cancelável; visão rejeita arquivos acima de 25 MiB antes de alocar o conteúdo inteiro. OCR e geração de tone também usam a mesma primitiva de output.

O coletor de deploy agora exclui antes da leitura arquivos privados sintéticos, `.git`, `.env`, chaves, backups, logs, metadados internos e objetos não regulares. O teste do provider fixture confirma que somente o conteúdo público aprovado é enviado. O dialer de deploy resolve e valida endereços antes de estabelecer TCP; todos os IPs retornados precisam ser permitidos. O hostname original permanece na URL para Host/SNI, enquanto o socket usa o IP validado. Nenhum deploy externo foi executado.

O HEAD `ec7f52c0` tornou o capturador de paridade portátil e observável. A execução bridged contra o bundle produzido e o backend Ollama local capturou dez rotas em viewport `1440x900`, com interação segura em cada rota e manifesto vinculado ao SHA `ec7f52c0`. Falhas de assets, JavaScript, requests ou endpoints agentic permanecem fatais; somente ausência de conta (`/api/me`) e endpoints base conhecidos sem implementação são classificados como esperados no manifesto. Essa captura representa o estado local observado e não prova providers externos, contas, dispositivos, deploy ou release.

| Dimensão | Estado |
|---|---|
| Implementado | R01–R04 e capturador observável no código publicado |
| Testado | Testes normais/race focados, vet, integrity, UI Vitest/build e compilação Darwin/Windows |
| Publicado | Commits `06990ef5` e `ec7f52c0` na branch do PR #1 |
| Homologado externamente | Não homologado; nenhum provider, conta, deploy, dispositivo ou loja foi usado |
| Pendente | CI do novo head, PR documental da Home, revisão visual e aprovação do merge documental |
| BLOCKED_BY_EXTERNAL_DEPENDENCY | OAuth/contas reais, smoke externo, hardware físico, assinatura, app review, lojas e homologação de operador |

`README_MAIN_STATUS=PENDING_DOCS_MERGE`: a Home escolhida será publicada por PR independente baseado na main, sem integrar o PR #1 do produto.


## Addendum de hardening de dependências runtime — 2026-09-23

O gate local do head documental `2f5674cf` revelou uma falha objetiva no audit de produção da UI: 12 vulnerabilidades transitivas, incluindo `seroval` crítico, Mermaid/DOMPurify/lodash-es/uuid e `mdast-util-to-hast`. A causa foi tratada no commit `eff054c1`: Streamdown passou de `1.4.0` para `2.6.0`, o devtools TanStack não utilizado saiu de `dependencies`, Shiki foi declarado diretamente e `mdast-util-to-hast` foi fixado em `13.2.1`.

Depois da correção, UI Vitest passou com 204 testes, o build TypeScript/Vite passou, `npm ci` foi reproduzível e `npm audit --omit=dev` retornou zero vulnerabilidades. O mobile também passou typecheck e audit de produção com zero vulnerabilidades. O CI remoto do novo head ainda está pendente; a execução anterior teve race macOS falho e normal macOS ainda em andamento, portanto não há claim de CI totalmente verde.

Estado: dependência interna **implementada/testada/publicada**; homologação externa e o novo CI permanecem pendentes. O veredito continua `FIXING / preview-local RC em hardening`, não final ou production-ready.


## Addendum de motivo explícito em approvals — 2026-09-23

A Agentic Console agora exige que o operador registre a justificativa da decisão antes de aprovar ou rejeitar uma ação. O servidor continua sendo a autoridade para nonce, actor, policy, organização e CAS; a UI não substitui essas verificações. O commit `720bbd92` adicionou o campo limitado a 512 caracteres e regressão de payload.

Os gates locais completos foram fechados no head anterior `f9ae5dd8` com caminhos absolutos: `FULL_LOCAL_GATES=PASS`. O novo head funcional foi publicado e o CI remoto ainda precisa concluir. O estado permanece `FIXING / preview-local RC em hardening`, sem promoção a final ou production-ready.


## Addendum de limite server-side de approval — 2026-09-23

O limite de motivo não depende mais somente da UI: o domínio rejeita razões acima de 2048 bytes antes da mutação CAS. Isso complementa o campo de 512 caracteres da Agentic Console e mantém input inválido em status de cliente, sem criar uma falsa garantia de homologação externa.


## Addendum — Tel-Agent textual operacional — 2026-09-23

O commit `45393d8c` entrega um canal Tel-Agent textual funcional no Company OS, acima dos painéis de construção: o operador escolhe uma operação allowlisted, envia uma mensagem bounded, recebe retorno estruturado e pode consultar o histórico persistente do tenant. As operações disponíveis são `report.read`, `backlog.create` e `campaign.draft`; mutações exigem owner/admin/operator quando a autenticação está ativa. Mensagens são redigidas por DLP antes da persistência e o histórico é limitado a 100 trocas.

O caminho é deliberadamente local-first. `campaign.draft` não publica campanha: grava um rascunho sandbox e cria approval pendente. O canal não simula telefonia, WhatsApp, SMS, voz ou integração externa; a resposta de histórico expõe `telephony: not_configured`. Assim, a jornada textual é **implementada, testada e publicada**, enquanto telefonia homologada, contas externas e mensagens reais continuam pendentes ou bloqueadas por dependência externa.

No primeiro check do SHA `45393d8c`, a CI pública estava parcialmente `queued`/`in_progress`; não há base para chamar o candidato totalmente verde ainda. O estado global continua **preview/local RC em hardening — não finalizado e não production-ready**.


## Addendum de gates completos do head Tel-Agent — 2026-09-23

O head documental `55dfa325` foi validado pelo runner absoluto com `FULL_LOCAL_GATES=PASS`. Passaram integrity, YAML, suíte Go completa com CGO, vet, build Go, Vitest, build TypeScript/Vite, audit de produção da UI, typecheck mobile, audit de produção mobile e diff limpo. Isso é evidência local reprodutível do candidato, não homologação de contas, hardware, lojas ou providers externos.

Na última consulta pública, SBOM, RLS/Redis/OTLP, Web/Mobile, integrity, mudanças, patches e `go_mod_tidy` estavam concluídos com sucesso; jobs Go e test/race da matriz upstream ainda estavam em execução. Assim, a CI remota do head não deve ser chamada totalmente verde até todos os jobs terminarem.


## Addendum de idempotência Tel-Agent — 2026-09-23

O commit `d67e7a79` tornou retries do Tel-Agent seguros contra duplicação local. `Idempotency-Key` é recebido por header, o payload é fingerprintado por digest, replay retorna a mesma exchange e uma reutilização com dados diferentes retorna conflito `409`. A UI mantém a mesma chave durante uma tentativa que falhou, sem exibir ou persistir seu valor.

O novo head passou o runner local absoluto completo (`FULL_LOCAL_GATES=PASS`), incluindo Go, UI, mobile, audits, integrity, YAML e diff. A CI remota havia iniciado e ainda estava pendente na última consulta; o veredito continua **preview/local RC em hardening**, não final ou production-ready.


## Addendum de consistência dos ciclos Company — 2026-09-23

O commit `46d6b34c` tornou a criação de ciclos Company idempotente e compensatória. O ciclo só permanece após schedule criado e vinculado; falhas de persistência não deixam ciclo enabled órfão. Replays com a mesma chave retornam `200` sem duplicar agenda, enquanto payload divergente retorna `409`.

A cobertura normal/race e o runner local completo passaram. A CI remota do novo head ainda estava queued/in progress na última consulta. O produto continua **preview/local RC em hardening**, não finalizado nem production-ready.


## Addendum de retry bounded do worker — 2026-09-23

O commit `86597b82` faz o worker local registrar falhas de criação de missão, aplicar backoff de 5/10 segundos e desabilitar o schedule após três falhas consecutivas. Uma execução válida limpa o estado. O claim não avança em memória quando a persistência falha.

Os testes normal/race e o runner local completo passaram. Isso não prova lease distribuído, múltiplos workers, DLQ de schedule ou recuperação em Postgres/Redis reais. A CI remota ainda estava em execução e o produto continua **preview/local RC em hardening**.


## Addendum de rollback da fila local — 2026-09-23

O commit `717a7e4f` tornou as transições persistentes da fila local transacionais em relação ao `jobs.json`: falha de escrita não deixa `Claim`, `Ack`, `Nack` ou `Replay` parcialmente aplicados em memória. Testes normais/race e o runner local completo passaram. Isso não equivale a lease multi-processo ou homologação de Redis; o produto permanece **preview/local RC em hardening**.


## Addendum de transições Ack/Nack — 2026-09-23

O commit `9d28cbc2` faz `Ack` e `Nack` aceitarem somente jobs `running` nas implementações local e Redis. A cobertura local e os gates Go completos passaram; o smoke Redis real e o lease distribuído continuam bloqueados por infraestrutura de homologação. O produto permanece **preview/local RC em hardening**.


## Addendum de compensações Redis — 2026-09-23

O commit `883a17e2` adicionou compensações para evitar chaves e IDs órfãos quando comandos Redis subsequentes falham. Os gates Go completos passaram e o teste de integração compilou, mas não houve conexão Redis real porque `OLLAMA_AGENT_TEST_REDIS_URL` não está configurado. O produto continua **preview/local RC em hardening**.


## Addendum de moveDue Redis — 2026-09-23

O commit `b1aaebfd` também restaura jobs no sorted set quando a promoção para pending falha após o `ZREM`. A correção passou testes focados normais/race; não houve homologação contra Redis real nesta sessão.


## Addendum de screenshot Settings e captura observável — 2026-09-23

A captura histórica `docs/images/screens/settings.png` foi corrigida para usar a mesma tela funcional de Settings registrada em `class-a-plus-settings.png`. O processo de captura também passou a falhar fechado quando `body`/`main` têm conteúdo insuficiente ou quando o arquivo PNG fica abaixo do orçamento mínimo de 16 KiB. O estado continua preview/local RC; essa correção não comprova providers externos, contas, dispositivos, deploy ou homologação.


## Addendum de containment, approval de deploy e webhook — 2026-09-23

O head `28d72528` corrige três achados de segurança reproduzíveis. Primeiro, `ContextStore` agora associa project roots a uma raiz de workspace canônica, rejeita escapes e symlinks em create/update e o ingestor valida novamente a raiz persistida antes de qualquer leitura. Um teste com projeto legado recarregado de disco confirmou rejeição antes de `os.Open` quando o root está fora do workspace do Runtime.

Segundo, deploy externo não aceita mais `approved=true` como autorização do cliente. O fluxo agora cria um approval persistente tenant-bound por builder/provider/target, exige nonce e CAS, limita decisão a owner/admin quando auth está ativa, consome a aprovação uma única vez e bloqueia replay. Teste HTTP com provider `httptest` confirmou zero chamadas externas para o booleano cliente e para operador sem papel administrativo; somente a decisão admin seguida do nonce correto alcançou o fixture.

Terceiro, webhook de schedule compara a organização ativa com a organização do schedule, exige `Idempotency-Key` ou `X-Ollama-Webhook-ID`, persiste a chave por schedule e rejeita replay com `409` antes de criar missão. O ledger local sobrevive a restart; não representa webhook externo assinado ou integração homologada.

Evidência local deste head: `CGO_ENABLED=0 go test ./internal/agent -count=1 -timeout=300s` passou; os testes HTTP novos de deploy/webhook passaram com `CGO_ENABLED=1`, incluindo `-race` para os três fluxos de segurança; `CGO_ENABLED=1 go vet ./internal/agent ./server` passou; `git diff --check` e scan de segredos dos arquivos alterados passaram. O guardrail de integridade e os gates UI/mobile/full-release ainda não foram executados neste novo head.

Na primeira consulta após o push, a CI pública do PR #1 estava `queued`/`in_progress`. O PR permanece aberto, sem merge automático. O estado continua **FIXING / preview-local RC em hardening — não finalizado e não production-ready**. Deploy real, rollback provider-specific, OAuth/IdP, Redis/Postgres/OTLP distribuídos, hardware, instaladores assinados, lojas e homologação externa continuam pendentes ou `BLOCKED_BY_EXTERNAL_DEPENDENCY`.


## Addendum de throttling MFA — 2026-09-23

O head `93e091e2` adiciona proteção server-side para tentativas repetidas de MFA. TOTP e recovery code passam por uma janela persistente por usuário e peer remoto: cinco falhas em cinco minutos abrem lockout de 15 minutos, respostas protegidas durante o lockout retornam `429 Too Many Requests` com `Retry-After`, e o estado é carregado de `mfa-attempts.json` após restart. Um TOTP válido após o período limpa a janela; recovery codes continuam one-time.

Evidência local: testes unitários de AuthStore e recovery passaram; o teste HTTP do middleware confirmou quatro `401`, quinto `429` com `Retry-After` e lockout preservado após reload; o cenário passou também com `-race`; `CGO_ENABLED=0 go test ./internal/agent`, `CGO_ENABLED=1 go vet ./internal/agent ./server` e `git diff --check` passaram. O rate limiter é durável no store local, mas não é ainda um coordenador distribuído com locking/lease entre múltiplas instâncias ou uma política de IP real atrás de proxy.

A CI pública precisa concluir para este novo head. O produto continua **FIXING / preview-local RC em hardening — não finalizado e não production-ready**; MFA/SSO/IdP distribuídos, Redis/Postgres/OTLP, providers, hardware, instaladores e homologações externas permanecem pendentes ou `BLOCKED_BY_EXTERNAL_DEPENDENCY`.


## Addendum de claim/lease Redis e rediss fail-closed — 2026-09-23

O head `785cbf3b` remove o downgrade perigoso de `rediss://`: enquanto não existe adapter TLS verificado com CA/server name/mTLS, a abertura é rejeitada antes de qualquer conexão. Para `redis://`, `Claim` agora usa script Lua para retirar, verificar e marcar o job como `running` atomicamente, incrementa attempts e cria lease de visibilidade de 15 minutos. Um reclaimer Lua devolve jobs `running` expirados ao pending antes de novos claims. O worker não descarta mais erros de claim, Ack, Nack e indexação dead-letter; eles vão para um canal bounded de observabilidade.

Evidência local: suíte completa `CGO_ENABLED=0 go test ./internal/agent -count=1 -timeout=300s`, race `CGO_ENABLED=1 go test -race ./internal/agent -count=1 -timeout=300s`, `CGO_ENABLED=1 go vet ./internal/agent ./server`, testes focados rediss/scripts e `git diff --check` passaram. O teste distribuído real `go test -tags integration ./internal/agent -run TestDistributedRedisRetriesDeadLetterReplay` foi executado e ficou `SKIP` porque `OLLAMA_AGENT_TEST_REDIS_URL` não está configurado; portanto Redis real, TLS Redis, múltiplos workers, fencing e perda/reconexão continuam `BLOCKED_BY_EXTERNAL_DEPENDENCY`.

A CI pública do novo head ainda precisa concluir. O produto continua **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de rollback dos stores persistentes — 2026-09-23

O head `44628055` torna mutations persistentes fail-safe em três camadas locais. `JSONStore` restaura mission/event em memória quando `writeJSONAtomic` falha; `ContextStore` restaura project, memory e schedules em updates/deletes com falha; `CompanyStore` usa snapshot profundo e restaura Update/mutate, inclusive quando a pausa por budget não consegue ser persistida, sem descartar o erro de filesystem.

Evidência local: regressões de fault injection para JSONStore, ContextStore e CompanyStore passaram; a suíte completa `CGO_ENABLED=0 go test ./internal/agent -count=1 -timeout=300s`, race `CGO_ENABLED=1 go test -race ./internal/agent -count=1 -timeout=300s`, `CGO_ENABLED=1 go vet ./internal/agent ./server` e `git diff --check` passaram. A garantia cobre uma instância e o writer atômico local; locking distribuído, NFS/FS remoto e falhas de energia ainda exigem ambiente de homologação e permanecem `BLOCKED_BY_EXTERNAL_DEPENDENCY`.

A CI pública do head atual continua pendente até concluir. O produto segue **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de enqueue idempotente e AutoRun fail-closed — 2026-09-23

O head `d0a7ae6e` evita duplicação de jobs por missão: o queue local retorna o job `pending/running` existente e o Redis usa índice Lua atômico por mission ID. `CreateMission` com `auto_run=true` não descarta erro de enqueue: persiste `FAILED`, `last_error` sanitizado e `mission.queue_failed`, e devolve o erro ao caller. `resumePending` passa a ser seguro contra reenqueue na mesma instância; a jornada não inventa uma fila durável se a persistência estiver indisponível.

Evidência local: suíte completa agent, race, vet e diff check passaram; Redis 7 local foi iniciado sem credenciais externas e os testes `TestDistributedRedisRetriesDeadLetterReplay` e `TestDistributedRedisClaimIsIdempotentAndReclaimsExpiredLease` passaram, cobrindo retry/DLQ, enqueue idempotente e recuperação de lease expirado entre workers. Não houve uso de contas ou serviços externos. Fencing token, Redis TLS `rediss://`, múltiplas instâncias com locking distribuído e perda de conexão continuam `BLOCKED_BY_EXTERNAL_DEPENDENCY` para homologação real.

A CI pública do novo head ainda precisa concluir. O produto segue **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de connector credential fail-closed — 2026-09-23

O head `55c805a0` corrige o caminho de egress dos connectors. Quando `TokenEnv` ou `OAuthProvider` é declarado e não resolve uma credencial, `CallForOrganization` retorna `connector credential is unavailable` antes de construir/enviar a requisição. A chamada não cai mais silenciosamente em HTTP sem `Authorization`.

Evidência local: testes de catálogo, OAuth tenant-aware, lifecycle, redirect/payload bounds e a regressão `TestConnectorFailsClosedBeforeEgressWhenTokenEnvIsMissing` passaram; essa regressão usa `httptest` e confirmou zero requests ao server. A suíte completa agent, race focado, vet e diff check passaram. Credenciais reais, OAuth upstream, scopes e homologação de cada connector continuam pendentes ou `BLOCKED_BY_EXTERNAL_DEPENDENCY`.

A CI pública do novo head precisa concluir. O produto permanece **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de qualidade web e mobile — 2026-09-23

O head `610de35e` limpa o lint mantido da UI sem desabilitar regras: usos manuais de `any` foram tipados com `unknown`/contratos específicos, refs imperativos receberam handle explícito, hooks passaram a declarar dependências completas e exports utilitários foram separados dos componentes para preservar Fast Refresh. A CI agora executa `npm run lint` no job Web and mobile quality.

Evidência local do head `610de35e`: `npm run lint` passou com zero errors e zero warnings; `./node_modules/.bin/tsc --noEmit` passou; `npm run test -- --run` passou com 22 arquivos/205 testes; `npm run build` passou; YAML, guardrail de integridade, diff check e scan de segredos passaram. O build ainda emite somente o aviso informativo de chunks grandes do Vite.

O head `b1f557d6` endurece o mobile em três pontos. Respostas HTTP do servidor deixam de ser tratadas como falha de transporte e não entram no retry offline; somente falhas sem status HTTP podem ser enfileiradas. O outbox preserva conflitos e respostas HTTP inesperadas para revisão, sem repetir uma mutação rejeitada pelo servidor. Approvals agora exigem motivo explícito digitado pelo operador, e o formulário usa `KeyboardAvoidingView`/persistência de toque para operação em teclado móvel.

Evidência local do head `b1f557d6`: `npm run typecheck` mobile passou; `npm run test:policy` passou com cenários de rede, 409, 422 e 503; YAML, guardrail de integridade, diff check e scan de segredos passaram. A CI pública foi disparada e estava `queued`/`in_progress` na primeira consulta; isso não é evidência de CI verde. Testes físicos Android/iOS, push remoto e resolução de conflitos em dispositivos reais continuam `BLOCKED_BY_EXTERNAL_DEPENDENCY`.

O produto segue **FIXING / preview-local RC em hardening — não finalizado e não production-ready**. PR #1 permanece aberto e sem merge automático.


## Addendum de deploy parcial e estado desconhecido — 2026-09-23

O head `190af0f3` deixa falhas de deploy observáveis. `DeploymentManager` agora preserva provider, deployment/site ID, URL, status e quantidade conhecida de arquivos quando Netlify cria efeitos remotos e falha na criação/upload; o erro tipado `DeploymentError` impede que o caller trate o caso como uma falha limpa. Falhas sem estado determinável retornam `unknown`. O endpoint de Builder responde `502 Bad Gateway` com `project`, `approval`, `deployment` e erro sanitizado para reconciliação manual, sem retry ou rollback universal.

Evidência local: testes de deploy generic/Netlify, exclusão de arquivos privados, symlink, SSRF e estado parcial passaram; o novo cenário parcial passou também com `-race`; testes server de `Deployment|Builder`, `go vet ./internal/agent ./server`, diff check e scan de segredos passaram. O rollback e health-check específicos de Vercel/Netlify/generic continuam `BLOCKED_BY_EXTERNAL_DEPENDENCY` até existirem contas, contratos, credenciais autorizadas e homologação externa.

A CI pública deste head foi disparada e precisa concluir. O produto segue **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de outbox persistente de push e observabilidade — 2026-09-23

O head `5a94f524` remove o envio de push em goroutine com erro descartado. Eventos de missão elegíveis entram em `.agent-push-outbox/outbox.json`, com organização, payload sanitizado, attempts, lease de um minuto, próximo retry, backoff e último erro. O worker iniciado pelo Runtime entrega por `PushService`, remove itens confirmados e mantém itens falhos para retry/restart. Falhas de AppendEvent, persistência do outbox e entrega push incrementam counters no snapshot/Prometheus.

Evidência local: regressões de persistência/restart, lease/backoff, entrega a fixture HTTPS local e retenção após resposta provider `502` passaram; a suíte completa `CGO_ENABLED=1 go test ./internal/agent -count=1 -timeout=300s`, race do pacote agent, `go vet ./internal/agent ./server`, guardrail de integridade, diff check e scan de segredos passaram. Isso comprova outbox e entrega local, não APNs/FCM, push em dispositivo, OAuth/conta externa ou worker distribuído; esses itens continuam `BLOCKED_BY_EXTERNAL_DEPENDENCY`.

A CI pública do head precisa concluir. O produto permanece **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de distribuição do fork, proveniência visual e release guard — 2026-09-23

O head `6724c6f0` torna a identidade de distribuição do fork explícita. `README.md`, `CLASS_A_PLUS_GUIDE.md`, `scripts/install.sh` e `scripts/install.ps1` agora orientam clone/build local do repositório DZ23-LTDA e não baixam `ollama.com`, instaladores upstream, modelos ou imagens Docker. Os scripts foram reduzidos a builders locais e não são instaladores assinados.

O capturador passou a registrar bytes e SHA-256; `docs/images/screens/class-a-plus-capture-manifest.json` foi versionado e `npm run screens:verify` falha quando um PNG diverge do manifesto. A CI web/mobile executa esse gate. O SBOM action foi fixado no commit `006b7ce8314066bdf1765b4500370d40fa6917a3` (v0.24.2). O workflow de release herdado deixou de disparar por tags `v*`: agora é manual, exige input explícito `enable_release=true`, variável de repositório `OLLAMA_ENABLE_RELEASE=true` e versão semver; isso não habilita release por si só e não prova assinatura, provenance, rollback, distribuição ou store review.

Evidência local: `go build -trimpath`, `sh -n scripts/install.sh`, `npm run screens:verify`, `npm run lint`, `npm test -- --run` (22 arquivos/205 testes), `npm run build`, YAML, integrity, diff check e scan básico de segredos passaram. Parsing PowerShell não foi executado neste Linux porque `pwsh` não está instalado: `BLOCKED_BY_EXTERNAL_DEPENDENCY`. Os checks públicos do head foram iniciados e estavam `in_progress`/`queued` na última consulta; não são declarados verdes.

O produto continua **FIXING / preview-local RC em hardening — não finalizado e não production-ready**. A PR principal permanece aberta, e a PR documental #5 permanece aberta aguardando autorização específica de merge.


## Addendum de estados de interação da UI — 2026-09-23

O head `79a0b06e` torna falhas do AgenticConsole observáveis para tecnologias assistivas: o erro recebe `role="alert"`, `aria-live="assertive"`, `aria-atomic` e foco programático. O Product Workspace agora diferencia carregamento de erro, oferece retry acessível e aplica pending por recurso nas mutações de projetos, schedules, plugins, connectors, MCPs e skills. Botões destrutivos receberam labels descritivos, tooltip e estado disabled durante a operação.

Evidência local: lint web sem erros ou warnings; typecheck; suíte Vitest com 22 arquivos e 206 testes; build Vite; teste focal do alerta acessível; guardrail Classe A+, YAML, diff check e scan básico de segredos passaram. O build continua emitindo apenas o aviso informativo de chunks grandes. A CI pública do head foi disparada e estava `queued`/`in_progress` na consulta inicial; isso não é evidência de CI verde.


## Addendum de reprodutibilidade de CI e presets — 2026-09-23

O head `f81722e5` fixa `tscriptify` em `v0.2.0`, o preset próprio do Desktop Commander em `0.2.51` e o action de attestation por SHA imutável `96b4a1ef7235a096b17240c259729fdd70c83d45` (v2). O guard de integridade foi atualizado para exigir esse SHA. Nenhum workflow contém mais `@latest`, `@main` ou `@master`; referências `@latest` que permanecem em páginas de integração herdadas não são usadas por workflows nem pelos presets do fork.

Evidência local: instalação de `tscriptify@v0.2.0`, parsing JSON do preset, YAML, guardrail Classe A+ e diff check passaram. A CI pública do head foi consultada uma vez e estava `queued`; a PR #1 continua aberta, com `mergeStateStatus=UNSTABLE`, sem merge automático.


## Addendum de dependências Go e auditoria de vulnerabilidades — 2026-09-23

O head `677c3dbc` atualiza o módulo para Go `1.26.6` e corrige as dependências alcançáveis apontadas pelo `govulncheck`, incluindo gRPC, `x/image`, `x/text`, `x/crypto`, pgx, goxmldsig e a família OpenTelemetry. O `go.mod` agora orienta a mesma versão patch-level corrigida que os workflows `setup-go` obtêm por `go-version-file`.

Evidência local: `GOTOOLCHAIN=go1.26.6 /tmp/govulncheck ./...` terminou com `No vulnerabilities found` e zero vulnerabilidades alcançáveis; o relatório ainda informa sete vulnerabilidades em pacotes importados e cinco em módulos exigidos que não são alcançáveis pelos caminhos analisados, portanto isso não equivale a uma declaração de risco zero em todas as dependências. `go mod verify`, `CGO_ENABLED=1 go test ./... -count=1 -timeout=900s`, `go vet ./...` e `go build -trimpath` passaram. `npm audit --omit=dev` também retornou zero vulnerabilidades para web e mobile.

O produto segue **FIXING / preview-local RC em hardening — não finalizado e não production-ready**. A CI pública do novo head ainda precisa concluir, e homologações de múltiplos sistemas, providers, IdP/OAuth, sandbox host, dispositivos, releases assinados e serviços externos continuam pendentes ou `BLOCKED_BY_EXTERNAL_DEPENDENCY`.


## Addendum de falhas de persistência e eventos observáveis — 2026-09-23

O head `fc7b77c6` remove descartes silenciosos no caminho de execução do Runtime. Writes de estado em awaiting approval e retry agora retornam imediatamente erros de persistência. O recovery de schedules e missões acumula falhas de reenqueue e de atualização de schedule, devolve um erro agregado e o worker registra a falha. Eventos de missão, passo e approval passam por `observeEvent`; falhas de persistência incrementam a métrica existente e são registradas com IDs/tipos não sensíveis.

Evidência local: testes focados de Runtime, schedules, queue e CreateMission passaram; `CGO_ENABLED=1 go test -race ./internal/agent -count=1 -timeout=600s`, `CGO_ENABLED=1 go vet ./internal/agent`, diff check e scan básico de segredos passaram. A CI pública do head foi disparada e estava `queued`/`in_progress` na consulta inicial; isso não é evidência de CI verde. O produto segue **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de falhas residuais de filas e orchestrator — 2026-09-23

O head `35be21c6` fecha descartes adicionais de erros nas superfícies agentic. O worker local registra falhas de ACK/NACK. A promoção de jobs delayed no Redis preserva e reporta falha de rollback quando o LPUSH não consegue concluir. O CollaborationStore rejeita ledgers JSON corrompidos em vez de inicializar silenciosamente estado vazio. O AgentOrchestrator restaura o snapshot em memória quando Plan, Run ou Cancel não conseguem persistir. As rotas HTTP de autorun e execução assíncrona registram falhas de RunForOrganization com identificador do job e organização.

Evidência local: regressões de colaboração, Plan/Run/Cancel, queue e runtime passaram; `CGO_ENABLED=1 go test -race ./internal/agent -count=1 -timeout=600s`, `CGO_ENABLED=1 go vet ./internal/agent ./server`, testes server de orchestration/agent, guardrail de integridade e diff check passaram. A CI pública do head foi consultada uma vez e estava `queued`. O produto segue **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de rollback de colaboração — 2026-09-23

O head `c43e02e6` completa a transação local do CollaborationStore. `AddComment` e `SetPresence` restauram o snapshot anterior quando qualquer write atômico falha, e o carregamento continua rejeitando ledgers corrompidos. O AgentOrchestrator agora retorna o snapshot efetivamente persistido quando a persistência final falha, em vez de devolver ao caller um estado que não está no ledger.

Evidência local: testes focados de colaboração e orchestrator, race, vet, guardrail de integridade e diff check passaram. A CI pública do head foi consultada uma vez e estava `queued`. O produto continua **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de persistência de traces — 2026-09-23

O head `a25e1a37` torna o encerramento de spans local transacional. Quando `spans.json` não pode ser escrito, `SpanHandle.End` restaura o span em memória para o estado anterior e registra a falha com IDs não sensíveis; o trace não é apresentado como encerrado quando o ledger não confirma a gravação.

Evidência local: testes focados e race do TraceStore, CollaborationStore, AgentOrchestrator e JobQueue passaram; `go vet ./internal/agent ./server`, guardrail de integridade e diff check passaram. A CI pública do head foi consultada uma vez e estava `queued`. O produto segue **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de subscriptions push transacionais — 2026-09-23

O head `708b4696` corrige a última mutação local observada no PushService: `Register` restaura a subscription anterior ou remove a nova entrada quando `subscriptions.json` não pode ser persistido. O caller recebe erro e não há divergência entre a lista em memória e o ledger local.

Evidência local: teste de rollback do PushService, testes do push outbox, race, vet, guardrail de integridade e diff check passaram. A CI pública foi consultada uma vez e estava `queued`. Isso cobre o adapter local e uma fixture de endpoint, não APNs/FCM nem push em dispositivos reais; esses itens continuam `BLOCKED_BY_EXTERNAL_DEPENDENCY`. O produto segue **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de gates backend completos — 2026-09-23

No head `3c0a5ec0`, a validação local final do backend passou com `GOTOOLCHAIN=go1.26.6 CGO_ENABLED=1 go test ./... -count=1 -timeout=900s`, cobrindo os pacotes `internal/agent` e `server` junto ao restante do módulo. Também passaram `go build -trimpath`, `go vet ./...` e `git diff --check`.

Essa evidência é local e não substitui os runners públicos. Na última consulta, a CI do head estava `queued` nos gates Go, web/mobile, SBOM, superfícies Classe A+ e integrações PostgreSQL/Redis/OTLP. O produto permanece **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de mídia, manifesto de deployment e supply chain — 2026-09-23

O head `5280db9d` torna o dialer de mídia fail-closed antes do TCP: a resolução DNS é feita previamente, todos os endereços A/AAAA retornados são validados contra destinos privados e o socket só é criado para um endereço aprovado. A regressão usa resolvedor/listener controlados e não chama serviço externo. O head `6b1207c5` fecha a fronteira de deployment: o pedido de approval coleta um manifesto determinístico com arquivos incluídos, hashes SHA-256 e exclusões justificadas; o hash é persistido no approval e é comparado novamente no deploy. Alteração do workspace, replay, tenant/provider/target mismatch ou o booleano legado `approved=true` não chegam ao provider. O adapter conserva estados `partial`/`unknown` quando um provider já produziu efeito e falha depois.

Evidência local do head `6b1207c5`: `CGO_ENABLED=1 go test ./internal/agent -count=1 -timeout=420s`, race dos testes de deployment/approval, testes HTTP de Builder/approval, `go vet ./internal/agent ./server`, YAML, guardrail de integridade e `git diff --check` passaram. Não houve provider real nem conta externa; rollback/health-check específicos continuam `BLOCKED_BY_EXTERNAL_DEPENDENCY`.

## Addendum de auditoria de dependências web/mobile — 2026-09-23

O head `75c6a1ae` atualiza a árvore web com `npm audit fix` **sem `--force`**, fixa um override explícito de `picomatch` transitive e adiciona à CI auditoria `npm audit --omit=dev --audit-level=high` para web e mobile. No candidato atual, a auditoria de produção encontrou `0 vulnerabilities` tanto na UI web quanto no app mobile; typecheck, política offline mobile, lint, typecheck, testes, build e verificação do manifesto de screenshots web passaram localmente.

A auditoria completa da UI ainda reporta 7 advisories moderados somente no toolchain de desenvolvimento Vitest/Storybook (`@vitest/*`, `vitest` e `storybook`). A correção automática remanescente exige Vitest `5.0.1` como upgrade major, enquanto o `@storybook/addon-vitest` disponível declara peers Vitest 3/4; por isso nenhum `--force` foi aplicado e a migração major permanece pendente de uma slice própria com atualização coordenada e testes de Storybook. Isso não deve ser descrito como supply chain totalmente zerada.

A CI pública foi consultada uma vez no head `75c6a1ae` e permanecia com jobs `queued`/`in_progress`, sem conclusão verde ou falha final. O produto segue **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.

## Addendum de documentação visual Home — 2026-09-23

A branch documental do PR #5 foi atualizada no head `605cd959` com a imagem Home fornecida pelo mantenedor (`docs/images/screens/class-a-plus-home.png`, SHA-256 `0291a918e1af8ea282721ea4b4f126c7d9ceb95c983299da964d1e9461b235bf`). A legenda identifica expressamente a imagem como captura da branch de desenvolvimento `feat/manus-parity-omniroute` e declara que ela não prova integração na `main`. O PR #5 continua aberto e sem checks; sua integração requer autorização específica do mantenedor. O PR principal #1 também continua aberto e sem merge automático.

Os bloqueios externos permanecem: instaladores assinados, imagens Docker publicadas, auto-update/rollback verificável, homologação física Windows/macOS/Linux/Android/iOS, contas OAuth/IdP/providers, Redis/PostgreSQL/OTLP operacionais e providers de deploy reais. Esses itens continuam `BLOCKED_BY_EXTERNAL_DEPENDENCY` com requisitos explicitados, não como funcionalidades homologadas.

Evidência de referência: `https://github.com/DZ23-LTDA/ollama-classe-a-plus/pull/1`, `https://github.com/DZ23-LTDA/ollama-classe-a-plus/pull/5`.


## Addendum de publicação Docker fail-closed — 2026-09-23

O head `18059ba1` corrige um risco residual de identidade/distribuição: o workflow herdado `latest.yaml` não é mais acionado por release; ele só pode ser disparado manualmente com `enable_latest=true`, `OLLAMA_ENABLE_LATEST=true` e `DOCKER_REPO` explicitamente configurado no ambiente de release. Os scripts locais deixaram de assumir `ollama/ollama`: builds sem push usam o tag neutro `local/ollama-classe-a-plus`, pushes com esse placeholder falham antes do Docker e `tag_latest.sh` recusa tanto o upstream quanto o placeholder sem opt-in e registry do operador. O guard de integridade mantém essas invariantes.

Evidência local: YAML, guardrail de integridade, `sh -n`, cenários negativos de `tag_latest.sh` e do push Docker sem registry explícito passaram. Nenhuma imagem foi construída, publicada ou autenticada em registry. A CI pública do head `18059ba1` foi consultada uma vez e estava `queued`/`in_progress`; não é CI verde.

O produto continua **FIXING / preview-local RC em hardening — não finalizado e não production-ready**. A habilitação de registry, assinatura, SBOM/provenance operacional, smoke de pull e rollback ainda exige configuração e homologação externa pelo mantenedor (`BLOCKED_BY_EXTERNAL_DEPENDENCY`).


## Addendum de gates locais finais — 2026-09-23

No head `ae15710e`, o roteiro reprodutível de gates locais terminou com `FINAL_GATES=PASS`. Passaram `python3 /tmp/check-dz23-yaml.py`, `bash scripts/check-class-a-plus-integrity.sh`, `git diff --check`, `GOTOOLCHAIN=go1.26.6 CGO_ENABLED=1 go test ./... -count=1 -timeout=900s`, `go vet ./...`, `go build -trimpath`, lint/typecheck/test/build da UI web, `npm audit --omit=dev --audit-level=high` web, `npm run screens:verify`, typecheck e política offline mobile e auditoria de produção mobile. A UI informou apenas o warning não bloqueante de chunks maiores que 500 kB; o teste mobile em Node 22 informou warning experimental de strip-types, sem falha.

O primeiro roteiro havia parado antes do mobile por ausência local de `tsc`, causada pela remoção deliberada de `apps/mobile-agentic/node_modules` ao limpar artefatos da PR documental. Após `npm ci --no-audit --no-fund` somente no ambiente, os gates mobile passaram e `node_modules`/`dist` foram removidos novamente. Nenhuma dependência gerada foi commitada.

A auditoria completa de desenvolvimento ainda mantém 7 advisories moderados no toolchain Vitest/Storybook, conforme registrado acima; as auditorias de produção web e mobile estão em zero. A CI pública do head posterior será consultada separadamente e continua não substituindo homologação física, contas externas, signing, registry, providers, OAuth/IdP, Redis/PostgreSQL/OTLP operacionais ou deploy real. O produto permanece **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.


## Addendum de CI duplicada e compatibilidade macOS — 2026-09-23

O head `ee8e5907c56f7c90e20771d44ddbffb7ed655ba2` corrige duas causas reproduzidas nas execuções públicas anteriores. Primeiro, o workflow `test` usava `github.pull_request.number` fora do contexto correto; a chave de concorrência podia cair em `run_id`, impedindo o cancelamento de execuções obsoletas do mesmo PR. O workflow agora usa `github.event.pull_request.number` e mantém `cancel-in-progress: true`. Os workflows próprios `class-a-plus-integrity` e `dz23-agentic-quality` deixaram de executar novamente em cada `push` de branch de feature, mantendo validação por `pull_request` e por `push` na `main`.

Segundo, os jobs macOS falhavam porque `/var` e `/private/var` são aliases do mesmo filesystem, mas o Runtime comparava caminhos não canonicalizados. A raiz do Runtime agora é canonicalizada após a criação do diretório, e `resolveWorkspace` usa a mesma fronteira segura para paths existentes e futuros. As asserções dos testes passaram a comparar a raiz persistida canonicalizada. A regressão foi reproduzida no log público e passou localmente em execução normal e com race detector.

O lint público anterior também foi reproduzido: `gofumpt` e `whitespace` no teste de webhook, um helper de deployment não utilizado e três falsos positivos `intrange` em código Go gerado dentro de `app/ui/app/node_modules`. O teste foi formatado, o helper removido e `.golangci.yaml` passou a excluir somente dependências/artefatos gerados (`node_modules/` e `app/ui/app/dist/`), sem excluir o código mantido. A validação local com golangci-lint v2.13.2 retornou `0 issues`.

Evidências locais do head: `GOTOOLCHAIN=go1.26.6 CGO_ENABLED=1 go test ./... -count=1 -timeout=900s`, `go vet ./internal/agent ./server`, `go build -trimpath`, golangci-lint v2.13.2, YAML, guardrail de integridade e `git diff --check` passaram. Nenhum segredo ou dependência gerada foi commitado. A publicação remota foi confirmada com o mesmo SHA.

Após o push, a API pública do GitHub mostrou 15 check runs do head `ee8e5907`, todos `queued` no momento da consulta; portanto a CI pública ainda não pode ser chamada de verde. O PR principal #1 permanece aberto e sem merge automático. As execuções históricas falhas exibidas na tela não são apagadas por essa correção; elas continuam como histórico do GitHub. O produto segue **FIXING / preview-local RC em hardening — não finalizado e não production-ready**.
