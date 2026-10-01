# API agentic v1

A primeira API agentic roda no mesmo listener do Ollama e é local-first. Ela cria uma missão, gera um plano, aplica a policy de tools, persiste eventos e permite acompanhar o resultado.

## Configuração

Defina `OLLAMA_AGENT_ROOT` para o diretório que pode ser usado pelas missões. Por padrão, o runtime usa um diretório temporário do sistema. Defina `OLLAMA_AGENT_STORE` para o diretório persistente de missões, eventos e empresas. Se `OLLAMA_AGENT_MODEL` estiver definido, o runtime usa esse modelo para gerar o plano JSON; se o provider/modelo solicitado não tiver planner configurado ou a chamada falhar, a missão falha fechado em vez de usar um plano determinístico silencioso. `OLLAMA_AGENT_EMBED_MODEL` ativa memória semântica sobre a API `/api/embed`; `OLLAMA_AGENT_CONNECTORS`, `OLLAMA_AGENT_MCP` e `OLLAMA_AGENT_REMOTE_MCP` apontam para os manifestos de integração descritos em [INTEGRATIONS.md](INTEGRATIONS.md). `OLLAMA_AGENT_AUTH_STORE` habilita o store de identidade; `OLLAMA_AGENT_AUTH_REQUIRED=true` exige Bearer token; `OLLAMA_AGENT_CREDENTIAL_KEY` é obrigatório para persistir credenciais OAuth cifradas; cada provider OAuth pode declarar `OLLAMA_AGENT_OAUTH_<PROVIDER>_REVOCATION_URL`; `OLLAMA_AGENT_MEDIA_BASE_URL` e `OLLAMA_AGENT_MEDIA_API_KEY` ativam o adapter multimídia HTTPS; `OLLAMA_AGENT_REDIS_URL=redis://...` ativa a fila Redis com claim Lua e lease local.

`rediss://` é rejeitado fail-closed neste adapter enquanto não houver configuração verificada de CA/server name/mTLS; o runtime não faz downgrade para TCP plaintext. A validação distribuída exige Redis real e fixture TLS autorizada.

```bash
export OLLAMA_AGENT_ROOT=/home/usuario/dz23-workspaces
export OLLAMA_AGENT_STORE=/home/usuario/dz23-data/agent-store
export OLLAMA_AGENT_MODEL=qwen3-coder:latest
export OLLAMA_AGENT_EMBED_MODEL=nomic-embed-text
export OLLAMA_AGENT_CONNECTORS=/etc/ollama-dz23/agent-connectors.json
export OLLAMA_AGENT_MCP=/etc/ollama-dz23/agent-mcp.json
export OLLAMA_AGENT_REMOTE_MCP=/etc/ollama-dz23/agent-remote-mcp.json
export OLLAMA_AGENT_SANDBOX_MODE=best-effort
# Em Linux multiusuário, aponte para um subtree cgroup v2 delegado ao processo:
# export OLLAMA_AGENT_SANDBOX_MODE=strict
# export OLLAMA_AGENT_SANDBOX_CGROUP_ROOT=/sys/fs/cgroup/ollama-agent
ollama serve
```

O modelo não recebe permissão implícita. Ele somente sugere passos; o runtime valida cada passo contra o registry e a policy.

### CLI autenticada

Os comandos `ollama agent` enviam `Authorization: Bearer ...` quando `OLLAMA_AGENT_TOKEN` está definido. Forneça o token por secret manager ou pelo ambiente seguro do processo; não passe token como flag/argumento nem o grave em arquivo versionado. As respostas dos callbacks OAuth/OIDC e SAML incluem um token local de sessão sujeito à expiração. O CLI ainda não implementa login interativo nem renovação automática; obtenção/refresh devem ocorrer por um fluxo de identidade já configurado. O endpoint `auth/dev/token` é exclusivamente bootstrap local com a proteção de desenvolvimento e loopback; nunca o habilite como fluxo de produção.

## MFA e proteção contra tentativa repetida

Quando `auth_required=true` e o usuário tem MFA habilitado, o middleware aceita `X-Ollama-MFA-Code` ou um `X-Ollama-MFA-Recovery-Code` de uso único. Falhas são contabilizadas por usuário e peer remoto, com limite de cinco falhas em cinco minutos; a quinta falha abre lockout de 15 minutos. Durante o lockout, a API responde `429 Too Many Requests` com `Retry-After` em segundos e não executa a ação protegida. O estado fica em `OLLAMA_AGENT_AUTH_STORE/mfa-attempts.json` e sobrevive a restart do `AuthStore`; um código TOTP válido após o lockout limpa a janela.

O ledger local é atômico em relação ao arquivo do `AuthStore`, mas não substitui um rate limiter coordenado para múltiplas instâncias que compartilhem armazenamento sem locking distribuído. Essa homologação, além de política de proxy para identificação de IP real, permanece `BLOCKED_BY_EXTERNAL_DEPENDENCY` até existir infraestrutura e teste distribuídos autorizados.

### Sandbox de execução

`sandbox.exec` usa `best-effort` por padrão para preservar compatibilidade local. Em Linux, tenta user/mount/PID/network namespaces quando o host permite, além de `ulimit`, timeout, grupo de processo e saída limitada. Em macOS e Windows, executa pelo interpretador disponível como processo best-effort com timeout e saída limitada; o resultado informa `network_isolation=not-enforced`. Nenhum desses modos é uma fronteira forte de segurança.

Em Linux, `OLLAMA_AGENT_SANDBOX_MODE=strict` só executa quando `OLLAMA_AGENT_SANDBOX_CGROUP_ROOT` é um subtree cgroup v2 real e delegado com `cpu`, `memory` e `pids`. O runtime cria um cgroup efêmero por passo, aplica CPU, memória, PIDs e swap, coloca o processo nele por `CgroupFD`, usa namespaces sem rede, `setpriv --no-new-privs` e instala um filtro seccomp antes do interpreter. Se qualquer pré-requisito faltar — inclusive em macOS/Windows — a execução é recusada; nunca há downgrade silencioso para best-effort. O operador deve provisionar o subtree, validar a política do host e manter AppArmor/SELinux ou um executor/container dedicado quando a ameaça exigir uma fronteira mais forte.

## Criar missão

```bash
curl -sS http://localhost:11434/api/agent/v1/missions \
  -H 'Content-Type: application/json' \
	-d '{"objective":"inspecionar o workspace e listar os arquivos","capabilities":["workspace:read"],"auto_run":true}'
```

A resposta contém `mission_id`, estado, plano, approvals e timestamps. Uma missão de leitura pode entrar em execução automaticamente quando `auto_run` é verdadeiro.

`capabilities` é a lista de escopos concedidos à missão. Quando omitida, o runtime concede apenas `workspace:read` e `workspace:write` para preservar o modo local-first. Browser, desktop, terminal, sandbox, MCP, connectors e deploy exigem os escopos correspondentes de forma explícita. A aprovação continua necessária quando o descriptor da tool exigir.

## Consultar missão e eventos

```bash
curl -sS http://localhost:11434/api/agent/v1/missions/MISSION_ID
curl -sS http://localhost:11434/api/agent/v1/missions/MISSION_ID/events
curl -N http://localhost:11434/api/agent/v1/missions/MISSION_ID/events/stream
curl -sS http://localhost:11434/api/agent/v1/missions/MISSION_ID/traces
```

Eventos são append-only no store e servem como trilha de planejamento, início, retry, sucesso, bloqueio, aprovação e conclusão.

## Artefatos

Quando uma tool produz um arquivo, a missão registra nome, caminho relativo, MIME type, tamanho e SHA-256. O download exige o identificador da missão e do artifact; o runtime revalida que o arquivo ainda está dentro do workspace.

```bash
curl -OJ http://localhost:11434/api/agent/v1/missions/MISSION_ID/artifacts/ARTIFACT_ID
```

## Aprovar ação de escrita

Passos de escrita ou terminal são marcados com `requires_approval`. A missão não executa a tool antes da aprovação.

```bash
curl -sS -X POST \
  http://localhost:11434/api/agent/v1/missions/MISSION_ID/approvals/APPROVAL_ID \
  -H 'Content-Type: application/json' \
  -d '{"approved":true,"reason":"revisado pelo operador"}'

curl -sS -X POST http://localhost:11434/api/agent/v1/missions/MISSION_ID/run
```

A rejeição torna a missão falha de forma explícita. A aprovação não é reutilizada entre missões.

## Projetos e memória

```bash
curl -sS -X POST http://localhost:11434/api/agent/v1/projects \
  -H 'Content-Type: application/json' \
  -d '{"name":"Meu projeto","root":"/workspace"}'

curl -sS http://localhost:11434/api/agent/v1/projects
curl -sS -X PATCH http://localhost:11434/api/agent/v1/projects/PROJECT_ID \
  -H 'Content-Type: application/json' \
  -d '{"name":"Meu projeto atualizado","root":"/workspace"}'
curl -sS -X DELETE http://localhost:11434/api/agent/v1/projects/PROJECT_ID

curl -sS -X POST http://localhost:11434/api/agent/v1/projects/PROJECT_ID/memories \
  -H 'Content-Type: application/json' \
  -d '{"kind":"decision","content":"usar testes de contrato","confidence":1,"source":"operator"}'

curl -sS 'http://localhost:11434/api/agent/v1/projects/PROJECT_ID/memories?q=contrato'
```

Memórias são persistidas por projeto, têm fonte, confiança e, quando `OLLAMA_AGENT_EMBED_MODEL` está configurado, vetor embedding. A busca usa similaridade coseno e mantém fallback lexical quando o embedder não está disponível.

## Capabilities e observabilidade

```bash
curl -sS http://localhost:11434/api/agent/v1/tools
curl -sS http://localhost:11434/api/agent/v1/connectors
curl -sS http://localhost:11434/api/agent/v1/mcp
curl -sS http://localhost:11434/api/agent/v1/metrics
curl -sS http://localhost:11434/api/agent/v1/metrics/prometheus
```

O endpoint Prometheus expõe counters de missões, passos, retries, approvals e chamadas de tool.

Notificações de eventos de missão para mobile não são enviadas em uma goroutine descartável. Quando push está configurado, o runtime grava primeiro em `OLLAMA_AGENT_STORE/.agent-push-outbox/outbox.json`; um worker local reivindica cada item com lease, tenta a entrega, remove no sucesso e aplica backoff persistente no erro. Restart recupera itens não concluídos, e `ollama_agent_event_persist_failures`, `ollama_agent_push_outbox_failures` e `ollama_agent_push_delivery_failures` ficam disponíveis no snapshot/Prometheus. Isso comprova durabilidade local e observabilidade, não entrega remota, APNs/FCM, push em dispositivo ou operação distribuída.

`GET /api/agent/v1/connectors` retorna o catálogo tenant-aware e o estado seguro de cada connector. O campo `credential_configured` é apenas booleano; tokens, nomes de variáveis e ciphertext nunca fazem parte da resposta. Catálogo, adapter e credencial configurada não significam conta OAuth ativa: a conexão real ainda exige autorização do operador, teste reversível e evidência do upstream.

## Agendamentos e webhooks

Um schedule cria missões recorrentes. O intervalo é limitado a 31 dias e o claim é idempotente no store. `GET /schedules` lista os schedules do tenant ativo; `PATCH /schedules/SCHEDULE_ID` edita o objetivo, intervalo e estado; `DELETE /schedules/SCHEDULE_ID` remove a automação.

```bash
curl -sS -X POST http://localhost:11434/api/agent/v1/schedules \
  -H 'Content-Type: application/json' \
  -d '{"objective":"verificar o projeto","interval_seconds":3600,"webhook_secret_env":"DZ23_WEBHOOK_SECRET"}'

curl -sS http://localhost:11434/api/agent/v1/schedules
curl -sS -X PATCH http://localhost:11434/api/agent/v1/schedules/SCHEDULE_ID \
  -H 'Content-Type: application/json' \
  -d '{"objective":"verificar novamente","interval_seconds":7200,"enabled":true}'
curl -sS -X DELETE http://localhost:11434/api/agent/v1/schedules/SCHEDULE_ID
```

Para disparar por evento, configure a variável secreta no ambiente do processo e envie o header `X-Ollama-Agent-Secret`. O valor nunca é gravado no schedule.

```bash
export DZ23_WEBHOOK_SECRET='valor-fora-do-repositorio'
curl -sS -X POST http://localhost:11434/api/agent/v1/webhooks/SCHEDULE_ID \
  -H "X-Ollama-Agent-Secret: $DZ23_WEBHOOK_SECRET" \
  -H 'Content-Type: application/json' \
  -d '{"event":"push","ref":"main"}'
```

## Company OS

Uma empresa é criada no tenant ativo e persistida em `OLLAMA_AGENT_STORE/companies`. A criação inicial gera CEO/Estratégia, Produto, Engenharia, Marketing, Vendas, Suporte e Operações. Roadmap, metas, backlog e ciclos são dados da empresa, não fixtures da UI.

```bash
curl -sS -X POST http://localhost:11434/api/agent/v1/companies \
  -H 'Content-Type: application/json' \
  -d '{"name":"Minha empresa","mission":"Resolver um problema real","business_model":"SaaS","budget":{"currency":"BRL","monthly_limit_cents":100000,"approval_threshold_cents":10000}}'

curl -sS http://localhost:11434/api/agent/v1/companies
curl -sS http://localhost:11434/api/agent/v1/companies/COMPANY_ID/report
curl -sS -X POST http://localhost:11434/api/agent/v1/companies/COMPANY_ID/backlog \
  -H 'Content-Type: application/json' \
  -d '{"title":"Validar oferta com clientes","priority":10,"owner_department":"sales"}'
curl -sS -X POST http://localhost:11434/api/agent/v1/companies/COMPANY_ID/cycles \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ciclo diário","objective":"Revisar métricas e executar o próximo item","frequency":"daily","interval_seconds":86400}'
```

Criar um ciclo também cria um schedule persistente com workspace `company://COMPANY_ID`. `POST /companies/COMPANY_ID/pause` impede que o worker crie novas missões para os ciclos dessa empresa. Gasto em anúncios/contratos ou acima do limiar exige `approved:true`; limite excedido e anomalias de alta severidade pausam a empresa.

O Growth OS local usa approval explícito antes de qualquer efeito externo:

```bash
curl -sS -X POST http://localhost:11434/api/agent/v1/companies/COMPANY_ID/campaigns \
  -H 'Content-Type: application/json' \
  -d '{"name":"Campanha de validação","channel":"social","objective":"Gerar leads","daily_budget_cents":0}'
curl -sS -X POST http://localhost:11434/api/agent/v1/companies/COMPANY_ID/campaigns/CAMPAIGN_ID/approve
curl -sS -X POST http://localhost:11434/api/agent/v1/companies/COMPANY_ID/campaigns/CAMPAIGN_ID/launch

curl -sS -X POST http://localhost:11434/api/agent/v1/companies/COMPANY_ID/products \
  -H 'Content-Type: application/json' \
  -d '{"sku":"SKU-001","name":"Produto sandbox","supplier":"Fornecedor sandbox","cost_cents":500,"price_cents":1200,"inventory":10}'
curl -sS -X POST http://localhost:11434/api/agent/v1/companies/COMPANY_ID/orders \
  -H 'Content-Type: application/json' \
  -d '{"product_id":"PRODUCT_ID","customer_ref":"customer-test","quantity":1}'
curl -sS -X POST http://localhost:11434/api/agent/v1/companies/COMPANY_ID/orders/ORDER_ID/approve
curl -sS -X POST http://localhost:11434/api/agent/v1/companies/COMPANY_ID/orders/ORDER_ID/fulfill \
  -H 'Content-Type: application/json' -d '{"tracking_code":"SANDBOX-001"}'
```

O endpoint `GET /companies/COMPANY_ID/growth/report` resume campanhas ativas, conversões de afiliados, catálogo, pedidos e receita registrada. A implementação não chama redes sociais, redes de afiliados, marketplaces, fornecedores, gateways de pagamento ou transportadoras.

## Tools da primeira fatia

`workspace.list` lê entradas do workspace autorizado e limita a quantidade retornada. `workspace.read` lê no máximo 1 MiB e rejeita traversal. `workspace.write` exige approval, limita o payload, escreve com permissões restritas e cria um manifesto com tamanho e SHA-256. `terminal.exec` existe em modo conservador e só permite `pwd`, `ls` e `git status` com argumentos separados; shell interpolation e outros executáveis são bloqueados. `sandbox.exec` executa Python ou Node. Em Linux best-effort tenta namespaces; em macOS/Windows usa processo best-effort e declara que a rede não está isolada. O modo strict exige Linux, cgroup v2 delegado e approval. `browser.operator` usa Chromium/Playwright com perfil persistente por sessão, bloqueio SSRF, snapshot, click, fill, press, upload, download, screenshot e estado `approval_required` para takeover humano. `desktop.companion` possui adapters Linux, macOS e Windows para screenshot, mouse, teclado, clipboard e processos através de executáveis separados e sem shell concatenado. `mcp.call` usa JSON-RPC stdio com lifecycle, timeout, env allowlist e methods allowlisted. `mcp.remote.call` usa Remote MCP Streamable HTTP, exige HTTPS fora de loopback, aceita bearer apenas por variável de ambiente e mantém allowlist de métodos; OAuth PKCE e pareamento do provedor são externos. `connector.http` chama GitHub, Google, Slack, Discord ou WhatsApp somente por operação HTTPS declarada.

### Sandbox strict

`OLLAMA_AGENT_SANDBOX_MODE=strict` habilita a fronteira Linux com cgroup v2 delegado, namespaces, `no-new-privs`, seccomp e limites de CPU/memória/PIDs. Se `OLLAMA_AGENT_SANDBOX_CGROUP_ROOT` não apontar para um cgroup v2 delegado com os controllers necessários, a execução falha fechado. O modo vazio ou `best-effort` não deve ser tratado como isolamento forte; AppArmor/SELinux e homologação do host continuam responsabilidade do operador.

## OAuth e organizações

Com `OLLAMA_AGENT_AUTH_REQUIRED=true`, a API valida sessão, organização e RBAC antes de permitir ações. O fluxo `GET /auth/oauth/:provider/start` exige `redirect_uri` e `code_verifier` PKCE; o callback consome `state` uma única vez, troca o código no servidor e cifra access/refresh tokens com AES-GCM usando `OLLAMA_AGENT_CREDENTIAL_KEY`. Configure URLs e nomes das variáveis de segredo por provider sem colocar valores no repositório.

Mutations autenticadas com `Origin` usam a allowlist de `OLLAMA_ORIGINS`; origens cross-site são recusadas antes do Bearer/RBAC. `GET`, `HEAD`, `OPTIONS` e clientes nativos sem `Origin` não são bloqueados por esse guard, mas continuam sujeitos a autenticação e autorização. O bypass público de SSO fica limitado a start/callback/metadata/ACS e não alcança refresh, revoke, logout ou ações de tenant.

Depois do callback, o cliente pode renovar e revogar uma credencial somente dentro da organização autenticada:

```bash
curl -sS -X POST http://localhost:11434/api/agent/v1/auth/oauth/github/refresh \
  -H 'Authorization: Bearer TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"credential_id":"CREDENTIAL_ID"}'

curl -i -X POST http://localhost:11434/api/agent/v1/auth/oauth/github/revoke \
  -H 'Authorization: Bearer TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"credential_id":"CREDENTIAL_ID"}'
```

O refresh faz rotação quando o provider retorna um novo refresh token, usa CAS para impedir duas renovações concorrentes e não retorna ciphertext. A revogação chama o endpoint do provider quando `REVOCATION_URL` está configurado e sempre persiste a revogação local após uma resposta 2xx. O bypass público de SSO não se aplica a refresh ou revoke.

Para encerrar uma sessão local, envie o bearer atual a `POST /api/agent/v1/auth/logout`. O servidor revoga o token e responde `204`; o cliente deve limpar sua cópia em memória mesmo se a chamada falhar. `403` indica escopo/RBAC recusado e não deve ser tratado como expiração automática da sessão.

```bash
curl -i -X POST http://localhost:11434/api/agent/v1/auth/logout \
  -H "Authorization: Bearer $OLLAMA_AGENT_ACCESS_TOKEN"
```

### SAML enterprise

`GET /auth/saml/:provider/start` cria AuthnRequest assinado e retorna `authorization_url` e RelayState one-time. `GET /auth/saml/:provider/metadata` publica o metadata do SP. `POST /auth/saml/:provider/acs` valida assinatura, audiência, destination, condições e `InResponseTo` pelo adapter `crewjam/saml`, extrai claims e cria sessão local tenant-aware. O fluxo exige `OLLAMA_AGENT_AUTH_SSO_PUBLIC=true` quando usado como primeiro login público.

As variáveis são `OLLAMA_AGENT_SAML_<PROVIDER>_IDP_METADATA_URL`, `_METADATA_URL`, `_ACS_URL`, `_SP_PRIVATE_KEY_FILE`, `_SP_CERTIFICATE_FILE`, `_ENTITY_ID` e `_DEFAULT_REDIRECT_URI`. URLs de metadata, SP e ACS devem ser HTTPS; certificados e chaves devem ser montados fora do repositório.

### TLS, mTLS e RLS

`OLLAMA_AGENT_TLS_CERT_FILE` e `OLLAMA_AGENT_TLS_KEY_FILE` ativam TLS 1.3 para o listener. `OLLAMA_AGENT_REQUIRE_MTLS=1` exige também `OLLAMA_AGENT_TLS_CLIENT_CA_FILE` e valida certificado de cliente. O certificado do servidor é recarregado a cada handshake, permitindo rotação sem reiniciar o processo. O adapter PostgreSQL candidato aplica `FORCE ROW LEVEL SECURITY` e verifica `app.tenant_context` por HMAC em cada policy; a role runtime não possui o segredo, ownership, `CREATE`, `BYPASSRLS` nem acesso de migração. A integração adversarial real e a revisão de release continuam gates obrigatórios; consulte [`SECURITY.md`](../../SECURITY.md) antes de qualquer uso compartilhado.

## Mídia

Com um provider HTTPS compatível, use `POST /media/image`, `/media/video`, `/media/speech` e `/media/transcribe`, sempre com `mission_id`; os resultados são salvos em `.agent-media` e devolvem manifesto com hash. `POST /media/tone` gera um WAV determinístico local para smoke tests e não deve ser confundido com geração musical neural.

## Builders e preview

`POST /builders` cria `website`, `app`, `game`, `slides` ou `dashboard` a partir de arquivos controlados ou template. `POST /builders/:id/preview` valida a entrada e retorna manifesto; `GET /builders/:id/preview/*path` serve preview local com containment; `POST /builders/:id/export` cria ZIP; `POST /builders/:id/publish` publica uma cópia versionada localmente. Deploy público exige um adapter de hosting configurado e não é alegado por essa publicação local.

`POST /builders/:id/visual` atualiza componentes, bindings, estilos e eventos do canvas. `POST /builders/:id/undo` e `POST /builders/:id/redo` alteram o histórico persistido, incrementam a versão e regeneram o preview. O histórico é limitado às últimas 50 alterações para impedir crescimento sem limite.

`GET /deployments` lista somente os providers configurados sem tokens. O fluxo externo usa `POST /builders/:id/deploy/:provider/approval` para solicitar uma aprovação, `POST /builders/:id/deploy/:provider/approval/:approval_id` para a decisão owner/admin com nonce e `POST /builders/:id/deploy/:provider` para consumir a approval uma única vez. O deploy empacota o workspace contido do projeto e chama Vercel, Netlify ou um deployer genérico declarado em `OLLAMA_AGENT_DEPLOYMENTS`; `approved=true` enviado diretamente ao deploy é rejeitado como campo desconhecido. O token é lido exclusivamente de `token_env` no servidor. O adapter impõe limite de 2.000 arquivos, 50 MiB, redirects desabilitados e HTTPS fora de loopback.

## Colaboração

`GET /collab/:project_id`, `POST /collab/:project_id/comments`, `POST /collab/:project_id/presence` e `GET /collab/:project_id/stream` oferecem snapshot persistente, comentários, presença e SSE. Quando autenticação está ativa, o actor vem da sessão e o tenant é validado pelo middleware.

## Mobile

O cliente Expo em `apps/mobile-agentic` usa SecureStore para o Bearer token, polling compatível com Android/iOS e EAS profiles para development/preview/production. Assinatura, credenciais de loja, `ascAppId` e hosting da API continuam fora do código.


## Orquestração multiagente

`POST /api/agent/v1/orchestration/jobs` recebe `objective`, `roles` opcionais (`research`, `programming`, `testing`, `design`, `security`, `data`, `review`), `workspace`, `project_id` e `budget`. Se `auto_run` for verdadeiro, o job começa em background; caso contrário, a API devolve um job `PLANNED`.

```bash
curl -sS -X POST http://localhost:11434/api/agent/v1/orchestration/jobs \
  -H 'Content-Type: application/json' \
  -d '{"objective":"pesquisar e revisar a arquitetura","roles":["research","security","review"],"budget":{"max_agents":3,"max_seconds":600,"max_retries":1},"auto_run":true}'

curl -sS http://localhost:11434/api/agent/v1/orchestration/jobs/ORCH_ID
curl -sS -X POST http://localhost:11434/api/agent/v1/orchestration/jobs/ORCH_ID/run
curl -sS -X POST http://localhost:11434/api/agent/v1/orchestration/jobs/ORCH_ID/cancel
```

O orquestrador limita concorrência, tempo, tamanho de saída e retries por job. A síntese é determinística e conserva a saída de cada papel, citações e conflitos de evidência; um modelo posterior pode substituir o reducer sem remover a validação do runtime.

## Pesquisa profunda

`POST /api/agent/v1/research` baixa URLs HTTPS públicas em paralelo, limita bytes por fonte, aplica bloqueio SSRF, respeita `robots.txt` quando solicitado, usa cache por URL e retorna fontes, SHA-256, texto extraído, citações e erros individuais.

```bash
curl -sS -X POST http://localhost:11434/api/agent/v1/research \
  -H 'Content-Type: application/json' \
  -d '{"query":"arquitetura agentic","urls":["https://example.org"],"max_sources":8,"respect_robots":true}'
```

A pesquisa não acessa hosts privados, loopback ou link-local em produção. Login, CAPTCHA e fontes autenticadas continuam exigindo o fluxo Browser Operator com takeover humano; o ResearchEngine não tenta contornar autenticação.

## Dispositivos e companions

`POST /devices/pair/start` cria um código one-time com expiração curta. O companion envia esse código em `POST /devices/pair/complete` junto com plataforma, nome e capability report; o token retornado deve ser armazenado somente no dispositivo. `POST /devices/:id/heartbeat` aceita Bearer token ou `X-Device-Token` e atualiza capabilities. `GET /devices` lista somente metadados; `POST /devices/:id/revoke` revoga o token e impede novos heartbeats.

```bash
curl -sS -X POST http://localhost:11434/api/agent/v1/devices/pair/start
curl -sS -X POST http://localhost:11434/api/agent/v1/devices/pair/complete \
  -H 'Content-Type: application/json' \
  -d '{"pairing_code":"CODE","name":"Meu Desktop","platform":"darwin","capabilities":[{"name":"screen","scopes":["desktop:screen"]}]}'
```

O pairing local não substitui mTLS, assinatura de binário ou store distribuído de produção. Esses gates permanecem obrigatórios antes de expor companions em rede pública.


## Ingestão e memória documental

`POST /projects/:id/ingest` importa arquivos do root autorizado ou URLs públicas para chunks de memória. O ingestor suporta texto, Markdown, HTML, JSON, CSV, PDF via `pdftotext`, DOCX via `document.xml` e XLSX via worksheets; cada chunk conserva o caminho/URL de origem. Embeddings são produzidos pelo embedder configurado no ContextStore. Imagens exigem um adapter de visão/OCR configurado e não são tratadas como texto automaticamente.

```bash
curl -sS -X POST http://localhost:11434/api/agent/v1/projects/PROJECT_ID/ingest \
  -H 'Content-Type: application/json' \
  -d '{"paths":["docs/spec.pdf","README.md"],"chunk_size":1800,"chunk_overlap":200}'
```


## Grok Live, Evaluation OS e routing

`GET /api/agent/v1/grok/status` retorna o estado sanitizado do provider xAI/Grok. `POST /api/agent/v1/grok/responses` encaminha uma requisição Responses com `input`, `model`, `tools` e `stream` usando a chave configurada somente no servidor. Sem credencial, o endpoint não tenta uma chamada externa.

O Evaluation OS executa casos determinísticos de coding, browser, tools, segurança, memória, planejamento e recuperação. O provider router pode restringir uma decisão a modelos locais, exigir capacidades, impor orçamento e ordenar candidatos por saúde, latência, custo e qualidade histórica. Essas métricas não provam qualidade geral nem substituem avaliação com dados autorizados.

## Lifecycle de plugins

Os endpoints seguintes alteram somente recursos já registrados no servidor e são protegidos pelo mesmo middleware agentic:

- `POST /api/agent/v1/connectors/:id/enable` e `POST /api/agent/v1/connectors/:id/disable`;
- `POST /api/agent/v1/mcp/:id/enable` e `POST /api/agent/v1/mcp/:id/disable`;
- `POST /api/agent/v1/remote-mcp/:id/enable` e `POST /api/agent/v1/remote-mcp/:id/disable`;
- `POST /api/agent/v1/skills/:id/enable` e `POST /api/agent/v1/skills/:id/disable`;
- `DELETE` nos recursos correspondentes para remoção explícita.

A UI não concede scopes por conta própria. O servidor valida o identificador, o estado e as allowlists antes de alterar o lifecycle. Tokens continuam fora das respostas e manifests sem atestado permanecem sem confiança executável.


## Lifecycle de connectors

`GET /api/agent/v1/connectors` lista somente connectors acessíveis à organização ativa e redige `token_env`; `credential_configured` é calculado no servidor como booleano. Quando `OLLAMA_AGENT_CONNECTORS` não está definido, o runtime usa o manifest durável `OLLAMA_AGENT_STORE/connectors.json`. O arquivo é criado com diretório privado, modo `0600` e atualização por arquivo temporário seguido de rename. Ele contém referências de configuração, nunca valores de token, ciphertext ou headers secretos.

Em modo autenticado, `POST /api/agent/v1/connectors` exige membership `owner` ou `admin`. O servidor força `organization_id` a partir do contexto autenticado. Um `organization_id` conflitante, um ID já pertencente a outro tenant, campos JSON desconhecidos e valores de segredo como `token` ou `api_key` são rejeitados. O payload usa apenas nomes de variáveis de ambiente e/ou identificadores de provider OAuth:

```bash
curl -sS -X POST http://localhost:11434/api/agent/v1/connectors \
  -H 'Authorization: Bearer TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"id":"github-prod","provider":"GitHub","base_url":"https://api.github.com","token_env":"DZ23_GITHUB_TOKEN","oauth_provider":"github","operations":[{"name":"profile","methods":["GET"],"path_prefixes":["/user"]}]}'
```

O endpoint valida HTTPS sem userinfo, nomes de ambiente, operações, métodos e prefixos de caminho. Requests e responses do connector permanecem limitados pelo manager e o egress mantém bloqueio de proxy/redirect e verificação do endereço conectado. A configuração `OLLAMA_AGENT_CONNECTORS=/caminho/manifest.json` continua disponível como modo estático explícito; nesse modo o manifest é carregado na inicialização e o lifecycle cadastrado pela UI não é apresentado como persistente nesse arquivo. O operador deve escolher conscientemente entre o manifest estático e o store durável.

A resposta do catálogo nunca devolve o nome da variável de token. Registrar um endpoint não prova que a conta do provider existe, que OAuth foi consentido, que o upstream está saudável ou que uma ação externa foi executada. Esses estados exigem credencial provisionada fora do repositório, escopos mínimos, approval e smoke reversível autorizado.


## Registro de MCP, Remote MCP e skills

Os endpoints de registro seguem o mesmo contrato de `POST /connectors`: autenticação owner/admin, binding server-side da organização, `DisallowUnknownFields` e resposta de catálogo redigida. O MCP stdio aceita `id`, `command` absoluto, `args`, `working_directory`, `allowed_methods`, `environment_vars` e `timeout_seconds`. O servidor exige arquivo executável regular, rejeita symlink como executable, limita args e valida nomes de env. Exemplo:

```json
{"id":"desktop-commander","command":"/opt/agent/bin/mcp-server","args":["--stdio"],"allowed_methods":["tools/list","tools/call"],"environment_vars":["DESKTOP_COMMANDER_TOKEN"]}
```

Remote MCP aceita `id`, `url`, `token_env`, `headers_env`, `allowed_methods` e `timeout_seconds`. A URL deve ser HTTPS fora de loopback, não pode conter userinfo/fragment/private address, e redirects precisam conservar origin permitido. Os valores de token/header nunca entram no JSON; somente nomes de env são armazenados. O DNS é resolvido e os endereços aprovados são pinados para a conexão.

Skills aceitam `id`, `version`, `description`, `scopes` e `tools`. `trusted` e `enabled` são estados derivados do servidor; o request com qualquer um desses estados como `true` é rejeitado. O runtime persiste o manifest em `OLLAMA_AGENT_STORE/context/skills/<id>.json`, sempre com `trusted=false`, para permitir revisão/approval explícita.

Sem `OLLAMA_AGENT_MCP` ou `OLLAMA_AGENT_REMOTE_MCP`, os managers padrão usam `OLLAMA_AGENT_STORE/mcp.json` e `remote-mcp.json`. Quando o operador define um arquivo estático por variável de ambiente, ele é carregado como bootstrap e mutations posteriores ficam apenas no manager da execução; a API não afirma ter editado o arquivo externo. Registrar é uma operação de configuração local: não equivale a OAuth consentido, conta conectada, smoke de upstream ou ação externa executada.


## Logout e escopo de sessão

`POST /api/agent/v1/auth/logout` tem dois comportamentos explícitos. Quando `auth_required=false`, é uma operação local idempotente e responde `204 No Content` mesmo sem bearer; não há sessão remota para revogar. Quando `auth_required=true`, o middleware exige bearer válido e o handler marca o hash do token como revogado, respondendo `204`. Tokens ausentes, inválidos ou expirados continuam retornando `401` no modo autenticado.

Esse endpoint revoga somente o token do runtime agentic. Não revoga automaticamente tokens OAuth de connectors, sessões de IdP ou contas de terceiros; cada credential/provider possui seu fluxo separado de refresh/revoke e requer homologação externa.


## Origin policy

Mutations agentic com `Origin` presente são aceitas somente quando a origem está em `OLLAMA_ORIGINS` ou nos defaults loopback (`localhost`, `127.0.0.1`, `0.0.0.0`, HTTP/HTTPS e portas). Essa política é aplicada com ou sem bearer. GET/HEAD, OPTIONS e clientes nativos sem header `Origin` preservam o caminho compatível. A regra é uma proteção de navegador e não substitui autenticação ou mTLS.


## Allowlist de Origin com portas variáveis

Para permitir portas variáveis em um host, use `https://host.example:*`. Esse formato compara scheme e hostname exatos e exige uma porta no header `Origin`; ele não permite subdomínios parecidos. Wildcards de esquema como `app://*` seguem sendo explícitos e separados. Userinfo, path, query e fragment não são aceitos no formato de porta variável.


## Bodies JSON

Os handlers agentic que usam `decodeJSON` aceitam exatamente um documento JSON. Campos desconhecidos são rejeitados e qualquer segundo objeto, array ou conteúdo inválido após o primeiro produz `400`. Whitespace final é permitido. Essa regra é complementar à autorização, approval e validação específica do endpoint.


O limite de transporte do `decodeJSON` é 4 MiB por request. Bodies maiores são rejeitados antes de concluir o parse; endpoints com payloads específicos podem impor limites menores. O orçamento não substitui limites de arquivos, documentos comprimidos ou respostas de providers.


`POST /api/agent/v1/auth/dev/token` é apenas uma rota de desenvolvimento: requer `OLLAMA_AGENT_AUTH_DEV=true` e peer loopback. Seu body também respeita o limite de 4 MiB, campos conhecidos e documento JSON único; a rota não deve ser exposta como onboarding de produção.


## Deployments

O adapter de deployment aceita HTTPS para serviços externos e HTTP somente em loopback. O root do workspace não pode ser symlink; redirects são bloqueados; a conexão padrão verifica o endereço IP real após o DNS e rejeita destinos privados não-loopback. O smoke local usa um servidor fixture. Configurar Vercel, Netlify, AWS, Cloudflare ou outro serviço não significa conta validada, domínio publicado, billing autorizado ou rollback testado.

Deploy externo não confia em `{"approved":true}` enviado pelo cliente. O fluxo é persistente e vinculado à organização, ao builder, ao provider e ao target:

1. `POST /api/agent/v1/builders/:id/deploy/:provider/approval` coleta um manifesto determinístico do workspace e solicita uma aprovação pendente; o corpo aceita somente `target`. A resposta retorna `approval.manifest_sha256` e `manifest`, com listas separadas de arquivos incluídos (`path`, `size`, `sha256`) e excluídos (`path`, `reason`). Arquivos privados nunca são enviados ao provider.
2. `POST /api/agent/v1/builders/:id/deploy/:provider/approval/:approval_id` decide a aprovação com `approved`, `reason` e `nonce`. Em auth-required, somente owner/admin podem decidir; o servidor compara tenant, builder, provider, estado pendente, expiração e nonce.
3. `POST /api/agent/v1/builders/:id/deploy/:provider` exige `approval_id`, `nonce` e o mesmo `target`. O servidor coleta novamente o manifesto, compara seu SHA-256 com a aprovação e só então consome a aprovação com uma transição CAS persistente antes de chamar o provider; replay, tenant mismatch, nonce incorreto, alteração do workspace ou a tentativa de usar o booleano cliente sem aprovação não chegam ao egress.

O ledger local fica em `OLLAMA_AGENT_STORE/.agent-deployment-approvals` e sobrevive a restart. Se o provider falhar depois do consumo, o adapter preserva `deployment.status` como `partial` quando conhece um site/deployment e como `unknown` quando não consegue determinar o estado. A rota responde `502` com `project`, `approval`, `deployment` e erro sanitizado para permitir reconciliação manual; não há retry automático nem rollback universal de Vercel/Netlify/generic. Rollback e health-check provider-specific continuam dependentes de contas, contratos e homologação externa.


## Mídia

`Transcribe` e `AnalyzeImage` exigem arquivos regulares dentro do workspace informado. O workspace final e os componentes do caminho não podem ser symlinks; traversal e input externo são rejeitados. Transcrição acima de 100 MiB falha antes da leitura. Geração e speech gravam somente em um workspace validado. Provider fixture local valida contrato, não conta, quota, GPU ou upstream externo.


## Company Growth OS

Campaigns, affiliate programs/links e orders retornam `mode: "sandbox"`; o relatório de crescimento retorna `sandbox_only: true`. O runtime pode planejar, aprovar, registrar métricas, manter inventário e simular fulfillment local. Um modo diferente de sandbox retorna `external growth action is not configured` até existir adapter, connector, credencial e smoke autorizado. Esses endpoints não cobram, publicam, movimentam dinheiro, fazem pedido externo ou emitem NF-e.


## Configuração segura

`GET /api/agent/v1/config` considera tanto arquivos/env de bootstrap quanto manifests registrados nos managers duráveis. `connectors_configured`, `mcp_configured`, `media_configured` e `deployments_configured` indicam configuração aceita pelo Runtime. Esses booleanos não afirmam credencial presente, conta OAuth conectada, resposta upstream, publicação ou homologação.


## Contratos de mídia e deploy após a reauditoria

Outputs de mídia e OCR são escritos somente por paths relativos ao workspace autorizado. A implementação usa `os.Root` e rejeita raiz, diretório ou componente symlink; uma troca concorrente para um alvo fora do root falha fechada. A validação do diretório de output ocorre antes de chamar provider ou processo externo. A leitura de áudio e imagem abre o descritor dentro do root e aplica orçamento cancelável. `Transcribe` rejeita entradas acima de 100 MiB. `AnalyzeImage` rejeita entradas acima de 25 MiB antes da leitura integral e também falha se o arquivo crescer durante a leitura.

O coletor de deployment aceita somente arquivos regulares de conteúdo público. `.git`, `.hg`, `.svn`, `.agent`, `.ollama`, `.secrets`, `node_modules`, `.env`, chaves, certificados, backups, dumps, bancos, logs e nomes explicitamente marcados como secret/credential/password/apikey ou `token` delimitado são excluídos antes da leitura. Assets legítimos como `logo.svg` e `tokenizer.json` não são excluídos por substring acidental. O manifesto expõe o motivo (`private-directory`, `private-extension` ou `private-name-marker`) e o hash dos arquivos incluídos. Isso é uma política conservadora de pacote; um build que necessite de conteúdo adicional deve produzir um diretório público aprovado, não depender de filtro posterior.

O cliente de deployment resolve A/AAAA antes do TCP. Todos os endereços retornados precisam ser não privados antes que o socket seja criado. O dial usa o IP validado, enquanto a URL mantém hostname para Host/SNI. HTTP é aceito somente para loopback em contexto local explícito; endpoints externos exigem HTTPS e redirects continuam desabilitados.


## Tel-Agent textual do Company OS

`POST /api/agent/v1/companies/:id/tel-agent` executa uma operação textual dentro do tenant da empresa. O corpo aceita `message` e uma operação explicitamente allowlisted: `report.read`, `backlog.create` ou `campaign.draft`. Para mutações, `title`, `description`, `priority` e `daily_budget_cents` são validados conforme a operação. O organization ID não é aceito do cliente: é derivado do contexto de autenticação. Em auth-required, apenas owner, admin ou operator podem executar mutações; leitura de relatório permanece disponível a membros autorizados.

A resposta contém `company`, `exchange` e, para leitura, `report`. Cada exchange inclui `channel: tel-agent.text`, actor, organization, operação, status, retorno, recurso criado quando aplicável e indicação de approval. `campaign.draft` sempre permanece sandbox e gera approval pendente; não publica campanha nem chama provedor externo.

`GET /api/agent/v1/companies/:id/tel-agent/history` retorna as últimas 100 trocas persistidas para o tenant e informa `telephony: not_configured`. Mensagens são limitadas a 2048 bytes e passam por redaction DLP antes da persistência. O canal textual não comprova OAuth, telefonia, WhatsApp, SMS, SIP ou qualquer conta externa ativa.


### Idempotência do Tel-Agent

`POST /api/agent/v1/companies/:id/tel-agent` aceita opcionalmente `Idempotency-Key` no header, com no máximo 128 bytes. O corpo não pode declarar a chave. Para `report.read`, `backlog.create` e `campaign.draft`, o servidor grava apenas um digest do input e associa a chave ao resultado. Repetir a mesma chave com o mesmo payload retorna a exchange já criada sem duplicar backlog, campanha ou approval; reutilizar a chave com payload diferente responde `409 Conflict`. O ledger sobrevive ao restart do store. A UI gera e conserva a chave durante retries, mas não a exibe.


## Ciclos autônomos do Company OS

`POST /api/agent/v1/companies/:id/cycles` aceita opcionalmente `Idempotency-Key` no header, com limite de 128 bytes. A criação grava o ciclo e o schedule como uma jornada compensatória: se o schedule não puder ser persistido ou vinculado, o ciclo é removido; o `ContextStore` também desfaz a entrada em memória quando sua escrita falha. Repetir a mesma chave e payload retorna `200` com o Company já agendado, sem criar outro schedule; reutilizar a chave com payload diferente retorna `409 Conflict`. O organization ID continua derivado da sessão e o schedule recebe o mesmo tenant.


Schedules também expõem `failure_count` e `last_failure_code` de forma sanitizada. O worker local registra `mission_creation_failed`, tenta novamente com backoff de 5 e 10 segundos e desabilita o schedule após três falhas consecutivas; uma execução bem-sucedida zera esses campos. O código interno do provider não é persistido. Essa garantia é local ao `ContextStore`; claim/lease distribuído e DLQ de schedule exigem a camada distribuída correspondente e testes reais.

### Webhooks de schedules

`POST /api/agent/v1/webhooks/:schedule_id` exige o segredo configurado em `WebhookSecretEnv`, compara a organização do schedule com a organização ativa e exige `Idempotency-Key` (ou o alias `X-Ollama-Webhook-ID`) de até 200 bytes. A chave é persistida por schedule antes da criação da missão; repetir a chave retorna `409 Conflict` sem criar outra missão. O ledger fica em `OLLAMA_AGENT_STORE/.agent-webhook-replay` e sobrevive a restart. Falha ao persistir a chave falha fechada; o endpoint não é uma autorização bearer cross-tenant nem um webhook de provider externo já homologado.


### Atomicidade do queue local

As mutações de job (`Claim`, `Ack`, `Nack` e `Replay`) só permanecem no estado em memória quando a gravação atômica de `jobs.json` é confirmada. Em erro de filesystem, a fila restaura o snapshot anterior. `Enqueue` é idempotente enquanto já existe job `pending` ou `running` para a mesma missão, evitando duplicatas de `resumePending` na instância local.

Quando `auto_run=true` não consegue persistir o enqueue, a missão é marcada `FAILED`, recebe `last_error` sanitizado e um evento `mission.queue_failed`; o erro é devolvido ao caller. Ela não fica silenciosamente `READY` para ser duplicada por um restart.


### Estado permitido para Ack/Nack

Um worker só pode concluir (`Ack`) ou devolver para retry/DLQ (`Nack`) um job que esteja `running`. Jobs em `pending`, `succeeded` ou `dead_letter` são rejeitados. A guarda existe no queue local e no adapter Redis; ela não substitui lease, fencing token ou recuperação de worker distribuídos.


### Compensação do adapter Redis

O adapter Redis usa scripts Lua para tornar `Enqueue` idempotente por missão e para fazer o `Claim` retirar/verificar/marcar o job atomicamente. Cada claim registra lease de visibilidade de 15 minutos; antes de um novo claim, leases expirados são recuperados para `pending`. Erros de claim/Ack/Nack e indexação dead-letter são publicados no canal bounded `RedisQueue.Errors()` em vez de serem descartados.


O movimento de jobs delayed para pending no adapter Redis ainda usa compensação entre comandos (`ZREM`/`LPUSH`); falha do `LPUSH` recoloca o ID no sorted set. O fixture Redis local comprovou retry/dead-letter, enqueue idempotente e recuperação de lease. Fencing token, Redis TLS `rediss://`, perda de conexão e múltiplas instâncias continuam exigindo homologação distribuída autorizada.
