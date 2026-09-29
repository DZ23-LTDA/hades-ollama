# Integrações externas e MCP

## Conectores HTTP

O runtime aceita um arquivo JSON apontado por `OLLAMA_AGENT_CONNECTORS`. O arquivo contém apenas metadados, URLs HTTPS, operações permitidas e o nome da variável de ambiente que contém o token. Para tenants autenticados, uma operação pode declarar `oauth_provider` e o runtime resolve o access token cifrado da organização; nesse modo o token não é aceito pelo input da tool. O valor do token nunca entra no repositório, missão, memória, evento ou resposta de capabilities.

```bash
export OLLAMA_AGENT_CONNECTORS=/etc/ollama-dz23/agent-connectors.json
export DZ23_GITHUB_TOKEN='...'
export DZ23_GOOGLE_TOKEN='...'
export DZ23_SLACK_TOKEN='...'
export DZ23_DISCORD_TOKEN='...'
export DZ23_WHATSAPP_TOKEN='...'
```

O exemplo [agent-connectors.json](../../examples/agent-connectors.json) cobre GitHub, Google Workspace, Slack, Discord e WhatsApp Cloud. Cada chamada exige uma operação declarada, método permitido e prefixo de caminho permitido. Redirects são desativados, o timeout é limitado e a resposta é limitada a 2 MiB. Operações de escrita são tools de efeito externo e exigem approval do runtime.

O fluxo de autenticação OAuth usa PKCE, state one-time, armazenamento AES-GCM, refresh server-side com rotação/CAS e revogação local com endpoint remoto opcional. Configure os endpoints do provider, incluindo `OLLAMA_AGENT_OAUTH_<PROVIDER>_REVOCATION_URL` quando suportado, e `OLLAMA_AGENT_CREDENTIAL_KEY` fora do repositório. Tokens pessoais por `token_env` continuam disponíveis apenas para desenvolvimento ou conectores explicitamente não multiusuário. A implementação possui testes com provider TLS fixture; a homologação contra cada IdP, revocation semantics, quotas e rotação real continua dependente de conta de teste do operador.

O mesmo `OLLAMA_AGENT_CREDENTIAL_KEY` cifra tokens de push quando subscriptions são persistidas. Configure e mantenha essa chave estável antes de iniciar o serviço com registros existentes; arquivos legados de subscriptions com token em texto claro são regravados em formato cifrado no startup. Se a chave estiver ausente ou não puder decifrar um registro existente, a inicialização falha fechado — não remova a chave nem apague o arquivo para contornar a falha.

O endpoint `GET /api/agent/v1/connectors` também expõe um catálogo de integrações com categoria, capabilities, estado de habilitação e `credential_configured`. Esse último campo é somente booleano e não revela token, nome de variável ou ciphertext. O catálogo inclui contratos para GitHub, Google Workspace, Slack, Discord, WhatsApp, Composio, deploy, social commerce, Woovi/OpenPix e fiscal/NF-e; ele não declara qualquer conta externa como conectada. Uma conexão real exige credencial provisionada pelo operador, escopos mínimos, approval e smoke reversível com auditoria.

## SAML enterprise

O SP SAML é configurado por `OLLAMA_AGENT_SAML_<PROVIDER>_IDP_METADATA_URL`, `_METADATA_URL`, `_ACS_URL`, `_ENTITY_ID`, `_SP_PRIVATE_KEY_FILE`, `_SP_CERTIFICATE_FILE` e `_DEFAULT_REDIRECT_URI`. O metadata do IdP é validado pelo pacote `crewjam/saml`; o runtime assina AuthnRequests, usa RelayState one-time e valida o ACS antes de provisionar usuário e organização. O fluxo de primeiro login só fica público com `OLLAMA_AGENT_AUTH_SSO_PUBLIC=true`. Certificados, chaves e URLs devem ser provisionados pelo operador, e o teste ponta a ponta depende de um IdP real.

## MCP stdio

O runtime aceita um arquivo JSON apontado por `OLLAMA_AGENT_MCP`:

```json
[
  {
    "id": "filesystem-readonly",
    "command": "/usr/local/bin/my-mcp-server",
    "args": ["--readonly"],
    "allowed_methods": ["initialize", "tools/list", "tools/call"],
    "environment_vars": ["MCP_HOME"],
    "timeout_seconds": 30
  }
]
```

O processo é iniciado apenas quando uma chamada é feita. O manager usa JSON-RPC por stdin/stdout, limita methods, repassa somente variáveis explicitamente declaradas, desabilita shell e encerra o processo em timeout/cancelamento. O `mcp.call` é uma tool de efeito externo e exige approval.

O isolamento de filesystem e rede do servidor MCP ainda deve ser reforçado com um executor sandbox dedicado em instalações multiusuário. Nunca registre um MCP com `environment_vars` que incluam credenciais sem uma policy de tenant e auditoria equivalente.

## Desktop Commander local e Remote MCP

O [Desktop Commander](https://desktopcommander.app/) pode ser usado localmente como servidor MCP stdio com o preset [`examples/dz23-desktop-commander-mcp.json`](../../examples/dz23-desktop-commander-mcp.json). Ele requer Node.js 18+ e inicia o pacote `@wonderwhy-er/desktop-commander` com os métodos allowlisted. Filesystem, terminal, processos e edição continuam sujeitos ao registry de tools, approval e permissões do usuário do processo.

O Remote Desktop Commander oficial documenta o endpoint `https://mcp.desktopcommander.app/mcp`, Streamable HTTP, OAuth 2.0 com PKCE e pareamento por device flow. O preset [`examples/dz23-desktop-commander-remote.json`](../../examples/dz23-desktop-commander-remote.json) é carregado por `OLLAMA_AGENT_REMOTE_MCP`; o adapter usa HTTPS, allowlist de métodos, timeout e um bearer opcional em `DESKTOP_COMMANDER_ACCESS_TOKEN`. O runtime não tenta obter credenciais, abrir sessão OAuth, parear máquinas nem revogar dispositivos. A conta, o agente `npx ... remote`, a aprovação do código e a revogação devem ser feitos pelo operador no dashboard oficial.

O catálogo `GET /api/agent/v1/mcp` retorna servidores stdio e remotos sem o valor do token. Chamadas remotas passam pela tool `mcp.remote.call`, que exige approval. O serviço remoto está documentado como beta e executa com as permissões do usuário da máquina; não conecte uma conta de produção sem revisar escopos, dispositivo, logs e política de desligamento.

## HarnessRouter e harnesses de coding

O [HarnessRouter Community Edition](https://github.com/HarnessRouter/harnessrouter) pode ser configurado como provider `openai-compatible` em [`examples/dz23-harnessrouter.json`](../../examples/dz23-harnessrouter.json). Cada `ModelConfig` pode declarar `harness_id`; o proxy Ollama Full preserva metadata existente e sobrescreve `metadata.harness_id` no servidor, permitindo selecionar `harnessrouter/codex` ou `harnessrouter/claude-code` sem aceitar esse controle do browser.

A integração é opt-in, usa `HARNESSROUTER_API_KEY` somente no processo do servidor e mantém uma chave de entrada do gateway Ollama Full separada. O endpoint local HTTP é permitido apenas com allowlist de loopback; hosts externos exigem HTTPS. O adapter não prova instalação, autenticação, licença ou disponibilidade de um CLI: streaming, sessões de follow-up, cancelamento, artifacts e recovery precisam ser testados contra uma instância real antes de classificar o provider como validado.


## Infraestrutura distribuída

O runtime permanece local-first por padrão. O adapter PostgreSQL agora separa um papel migrator de um papel runtime sem ownership, `CREATE`, `BYPASSRLS` ou superuser; migrations não executam no startup. As policies usam somente `app.tenant_context`, um HMAC com segredo não legível pela role runtime, expiração curta e validação SQL na própria policy; os GUCs antigos não concedem acesso. A integração local adversarial com PostgreSQL 16 e Redis 7 passou com runtime/migrator separados, inclusive evento→missão cross-tenant, adulteração GUC/HMAC, leitura do segredo, `SET ROLE`, DDL, `row_security=off`, ownerless backfill, role-upgrade de volume legado e recuperação por tenant. Essa arquitetura **ainda é candidata, não uma declaração de isolamento enterprise validado**: staging com TLS/backup-restore, suporte validado a rotação da chave, escala e auditoria final seguem necessários antes de produção. A rotação da chave HMAC não é suportada hoje: não altere `OLLAMA_AGENT_TENANT_CONTEXT_KEY` isoladamente nem use uma chave nova no migrator; isso não atualiza o segredo armazenado e pode interromper o acesso. Mantenha a chave imutável até existir um procedimento transacional, com quiescência, verificação e rollback, coberto por integração. A migration falha fechada se encontrar owner inválido, missão sem organização válida, evento órfão ou evento divergente do tenant da missão; não faz backfill automático de ownership.

Para desenvolvimento com `deploy/docker-compose.agentic.yml`, defina senhas distintas `OLLAMA_AGENT_POSTGRES_ADMIN_PASSWORD`, `OLLAMA_AGENT_MIGRATOR_PASSWORD`, `OLLAMA_AGENT_RUNTIME_PASSWORD` e `OLLAMA_AGENT_REDIS_PASSWORD`. Gere uma chave HMAC persistente com `openssl rand -hex 32`, armazene-a em um secret manager e forneça-a como `OLLAMA_AGENT_TENANT_CONTEXT_KEY` ao migrator e ao servidor. Use `OLLAMA_AGENT_MIGRATOR_DATABASE_URL` apenas no job administrativo e `OLLAMA_AGENT_DATABASE_URL` apenas no runtime; essas DSNs não devem aparecer em argumentos/logs nem conter senhas em arquivos versionados. Para banco novo, inicialize roles, rode `ollama agent migrate-postgres` explicitamente e só então suba o servidor com `OLLAMA_AGENT_AUTH_REQUIRED=true` e a fila Redis owner-bound. A migration revoga `CREATE` e `TEMP` de `PUBLIC` e do runtime no database dedicado. A chave HMAC não tem rotação suportada hoje: não a substitua isoladamente nem use outra chave no migrator.

### Cutover de volume legado PostgreSQL

**Use exatamente esta ordem; mantenha o serviço Ollama parado e o tráfego fechado até o smoke final.** O fluxo em várias fases não é uma transação única nem tem rollback automático:

1. **Backup e ensaio:** configure o destino como um cluster PostgreSQL de staging separado do cluster de origem. O script compara os `system_identifier` e recusa seguir se forem iguais. O dump fica fora do repositório, com diretório privado; o database temporário tem nome exclusivo e é removido por `trap` inclusive se `pg_restore` ou a contagem falhar. Não use o teste de drift adversarial contra produção; ele altera policies/RLS deliberadamente.

   ```bash
   umask 077
   install -d -m 700 "$HOME/ollama-full-backups"
   export BACKUP_FILE="$HOME/ollama-full-backups/ollama_agent-$(date -u +%Y%m%dT%H%M%SZ).dump"
   export DSN_ADMIN_LEGADO='host=127.0.0.1 port=5432 dbname=ollama_agent user=ollama_agent'
   export DSN_ADMIN_NOVO='host=127.0.0.1 port=5432 dbname=ollama_agent user=ollama_agent_admin'
   # Ambos os DSNs devem apontar para staging, em cluster DIFERENTE da origem.
   export STAGING_PGHOST='staging-db.example' STAGING_PGPORT=5432 STAGING_PGUSER='staging_restore_admin'
   export DSN_ADMIN_MAINTENANCE="host=$STAGING_PGHOST port=$STAGING_PGPORT dbname=postgres user=$STAGING_PGUSER sslmode=verify-full"
   export DSN_RESTORE_STAGING_BASE="$DSN_ADMIN_MAINTENANCE"
   read -rsp 'Senha do administrador legado: ' PGPASSWORD; echo; export PGPASSWORD
   read -rsp 'Senha do administrador de staging: ' STAGING_PGPASSWORD; echo; export STAGING_PGPASSWORD
   PGPASSWORD="$PGPASSWORD" psql "$DSN_ADMIN_LEGADO" -v ON_ERROR_STOP=1 -Atc 'SELECT current_user, current_database()'
   source_cluster_id="$(PGPASSWORD="$PGPASSWORD" psql "$DSN_ADMIN_LEGADO" -v ON_ERROR_STOP=1 -Atc 'SELECT system_identifier FROM pg_catalog.pg_control_system()')"
   staging_cluster_id="$(PGPASSWORD="$STAGING_PGPASSWORD" psql "$DSN_ADMIN_MAINTENANCE" -v ON_ERROR_STOP=1 -Atc 'SELECT system_identifier FROM pg_catalog.pg_control_system()')"
   test -n "$source_cluster_id" && test -n "$staging_cluster_id"
   if [ "$source_cluster_id" = "$staging_cluster_id" ]; then
     echo 'ERROR: restore rehearsal target is the source PostgreSQL cluster; use a separate staging cluster' >&2
     exit 1
   fi
   restore_db="ollama_agent_restore_$(date -u +%Y%m%d%H%M%S)_$$"
   case "$restore_db" in (*[!a-zA-Z0-9_]*) echo 'ERROR: invalid generated rehearsal database name' >&2; exit 1;; esac
   cleanup_restore_db() {
     status=$?
     trap - EXIT
     if [ "${restore_db_created:-0}" = 1 ]; then
       PGPASSWORD="$STAGING_PGPASSWORD" psql "$DSN_ADMIN_MAINTENANCE" -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS $restore_db WITH (FORCE)" >/dev/null || true
     fi
     exit "$status"
   }
   trap cleanup_restore_db EXIT
   PGPASSWORD="$PGPASSWORD" pg_dump --format=custom --no-owner --no-acl --file="$BACKUP_FILE" "$DSN_ADMIN_LEGADO"
   pg_restore --list "$BACKUP_FILE" >/dev/null
   PGPASSWORD="$STAGING_PGPASSWORD" psql "$DSN_ADMIN_MAINTENANCE" -v ON_ERROR_STOP=1 -c "CREATE DATABASE $restore_db"
   restore_db_created=1
   staging_restore_id="$(PGPASSWORD="$STAGING_PGPASSWORD" psql "$DSN_RESTORE_STAGING_BASE" --dbname="$restore_db" -v ON_ERROR_STOP=1 -Atc 'SELECT system_identifier FROM pg_catalog.pg_control_system()')"
   test "$staging_restore_id" = "$staging_cluster_id"
   PGPASSWORD="$STAGING_PGPASSWORD" PGHOST="$STAGING_PGHOST" PGPORT="$STAGING_PGPORT" PGUSER="$STAGING_PGUSER" PGSSLMODE=verify-full pg_restore --no-owner --no-acl --dbname="$restore_db" "$BACKUP_FILE"
   PGPASSWORD="$STAGING_PGPASSWORD" psql "$DSN_RESTORE_STAGING_BASE" --dbname="$restore_db" -v ON_ERROR_STOP=1 -Atc 'SELECT (SELECT count(*) FROM agent_missions), (SELECT count(*) FROM agent_events)'
   PGPASSWORD="$STAGING_PGPASSWORD" psql "$DSN_ADMIN_MAINTENANCE" -v ON_ERROR_STOP=1 -c "DROP DATABASE $restore_db WITH (FORCE)"
   restore_db_created=0
   trap - EXIT
   ```

2. **Prepare roles e ownership:** com o serviço parado e conectado como o antigo `ollama_agent` superuser, forneça interativamente ou pelo secret manager os três valores distintos exigidos pelo script. Eles precisam estar exportados com estes nomes exatos; o `psql \getenv` não lê aliases:

   ```bash
   read -rsp 'Nova senha admin: ' OLLAMA_AGENT_POSTGRES_ADMIN_PASSWORD; echo
   read -rsp 'Nova senha migrator: ' OLLAMA_AGENT_MIGRATOR_PASSWORD; echo
   read -rsp 'Nova senha runtime: ' OLLAMA_AGENT_RUNTIME_PASSWORD; echo
   export OLLAMA_AGENT_POSTGRES_ADMIN_PASSWORD OLLAMA_AGENT_MIGRATOR_PASSWORD OLLAMA_AGENT_RUNTIME_PASSWORD
   psql "$DSN_ADMIN_LEGADO" -v ON_ERROR_STOP=1 -f deploy/postgres/migrate-existing-roles.sql
   ```

   A preparação falha se detectar qualquer sessão legada no cluster; encerre os clientes Ollama/legacy e tente novamente. Ela cria as credenciais novas, transfere database/schema/tabelas para o migrator, revoga `CREATE`/`TEMP` do runtime e não desativa o login antigo.

3. **Verifique as novas identidades e migre o schema antes de aposentar o login legado:**

   ```bash
   read -rsp 'Senha do novo admin: ' PGPASSWORD; echo; export PGPASSWORD
   psql "$DSN_ADMIN_NOVO" -v ON_ERROR_STOP=1 -Atc 'SELECT current_user, current_database()'
   PGPASSWORD="$OLLAMA_AGENT_MIGRATOR_PASSWORD" psql 'host=127.0.0.1 port=5432 dbname=ollama_agent user=ollama_agent_migrator' -v ON_ERROR_STOP=1 -Atc 'SELECT current_user'
   PGPASSWORD="$OLLAMA_AGENT_RUNTIME_PASSWORD" psql 'host=127.0.0.1 port=5432 dbname=ollama_agent user=ollama_agent_runtime' -v ON_ERROR_STOP=1 -Atc 'SELECT current_user'
   export OLLAMA_AGENT_MIGRATOR_DATABASE_URL='postgres://ollama_agent_migrator@127.0.0.1:5432/ollama_agent?sslmode=verify-full'
   read -rsp 'Chave HMAC persistente (hex, 64 caracteres): ' OLLAMA_AGENT_TENANT_CONTEXT_KEY; echo; export OLLAMA_AGENT_TENANT_CONTEXT_KEY
   PGPASSWORD="$OLLAMA_AGENT_MIGRATOR_PASSWORD" ollama agent migrate-postgres
   ```

   Configure TLS/CA e DSNs de acordo com o cluster; `sslmode=verify-full` pressupõe certificado/hostname válidos. Execute a suíte adversarial e o smoke do runtime em **staging isolado**, usando as roles e a mesma chave persistente, e confira ownership/preservação dos dados antes de continuar.

4. **Aposente e drene a role antiga:** conecte pelo novo admin verificado, com sua senha fornecida por secret manager/prompt, e rode a segunda etapa. Ela confirma as flags/memberships das roles substitutas, bloqueia novos logins, revoga privilégios e termina/aguarda todas as sessões `ollama_agent` em qualquer database do cluster:

   ```bash
   psql "$DSN_ADMIN_NOVO" -v ON_ERROR_STOP=1 -f deploy/postgres/retire-legacy-role.sql
   ```

5. **Suba o runtime e reabra tráfego só após o smoke:** configure `OLLAMA_AGENT_DATABASE_URL` com `ollama_agent_runtime`, `OLLAMA_AGENT_TENANT_CONTEXT_KEY` com a chave imutável e `OLLAMA_AGENT_REDIS_URL`; inicie o serviço, confirme readiness e faça leitura/escrita de um tenant autorizado. Runtime recusa o início se a role antiga ou qualquer sessão legada ainda tiver privilégios ativos.

Se qualquer etapa falhar, mantenha o serviço parado e siga o plano de restauração de staging/backup previamente ensaiado; não reative a role antiga por improviso, não presuma que `ROLLBACK` desfaz etapas já commitadas e não execute o teste de drift em produção. A role antiga é desativada somente na fase 4. Dados sem `organization_id` válido e missões executáveis sem `workspace_identity` persistida exigem correção/reautorização manual antes da migration. A rotação da chave HMAC, TLS/HA/failover, scale test e avaliação operacional em staging continuam pendentes; esta implementação permanece candidata e não está liberada para produção.

Para traces distribuídos, defina `OLLAMA_AGENT_OTLP_ENDPOINT` com URL OTLP HTTP `https://`; `OLLAMA_AGENT_OTLP_ALLOW_INSECURE=1` é reservado para desenvolvimento local. A stack em `deploy/docker-compose.agentic.yml` fornece PostgreSQL, Redis e OpenTelemetry Collector.

## Companion WebSocket e mTLS

O endpoint `GET /api/agent/v1/devices/:id/connect` aceita WebSocket somente sobre TLS, salvo `OLLAMA_AGENT_ALLOW_INSECURE_COMPANION=1` para loopback local. `OLLAMA_AGENT_TLS_CERT_FILE` e `OLLAMA_AGENT_TLS_KEY_FILE` ativam TLS 1.3 no listener e recarregam o certificado por handshake; `OLLAMA_AGENT_REQUIRE_MTLS=1` exige também `OLLAMA_AGENT_TLS_CLIENT_CA_FILE`. O primeiro frame precisa ser `hello` com `device_id`, `token` e capabilities. `OLLAMA_AGENT_COMPANION_ORIGINS` limita Origins explícitas; a lista não deve ser `*` em produção.

O protocolo inicial é `dz23-companion.v1` e suporta `hello`, `heartbeat`, `ping` e respostas de rejeição para tipos ainda não habilitados. A camada de transporte não executa comandos arbitrários: ações de tela, teclado, processos e arquivos continuam sujeitas ao registry de tools, scopes e approvals do runtime.

## Publicação de builders

Configure `OLLAMA_AGENT_DEPLOYMENTS` apontando para um JSON como [`examples/agent-deployments.json`](../../examples/agent-deployments.json). Os adapters `vercel` e `netlify` usam as APIs oficiais; `generic` envia um payload de arquivos base64 para `/deploy`, permitindo integrar AWS, Cloudflare, um pipeline interno ou outro hosting sem colocar SDKs e credenciais no binário. Os tokens são lidos de `token_env`, e uma publicação exige approval no endpoint. O código cria a requisição de publicação e valida o workspace, mas credenciais de conta, domínio, projeto, DNS, billing e permissões de hosting continuam responsabilidade do operador.

Em modo autenticado multi-organização, cada provider precisa declarar `organization_id` igual ao ID da organização proprietária. `GET /api/agent/v1/deployments`, a solicitação de approval e a execução mostram/aceitam somente providers dessa organização; providers antigos sem proprietário continuam disponíveis apenas no modo local não autenticado. Não há cadastro HTTP global de provider. Um exemplo de entrada é `{"id":"vercel-org-a","organization_id":"org-a","provider":"vercel","base_url":"https://api.vercel.com","token_env":"ORG_A_VERCEL_TOKEN"}`. Erros HTTP não-2xx dos providers são convertidos em mensagens genéricas com status e não devolvem o corpo arbitrário upstream.


## Composio Connect e plugin Composio

O Ollama Full pode registrar o [Composio Connect](https://docs.composio.dev/docs/composio-connect) como Remote MCP através de [`examples/dz23-composio-connect.json`](../../examples/dz23-composio-connect.json). O preset usa `https://connect.composio.dev/mcp`, permite somente os métodos JSON-RPC necessários (`initialize`, `notifications/initialized`, `tools/list` e `tools/call`) e injeta `x-consumer-api-key` apenas no servidor por meio de `COMPOSIO_CONSUMER_API_KEY`. O valor nunca é retornado pela API de capabilities, browser ou logs.

```bash
export OLLAMA_AGENT_REMOTE_MCP="$PWD/examples/dz23-composio-connect.json"
export COMPOSIO_CONSUMER_API_KEY='valor-fora-do-repositorio'
```

O Composio Connect expõe meta-tools para descobrir tools, obter schemas, iniciar conexões OAuth e executar tools. Por isso, “ter o plugin” significa ter o adapter MCP e o fluxo de aprovação no Ollama Full; ainda é necessário autorizar cada conta upstream no navegador do operador. A integração não cria uma conta Composio, não completa OAuth automaticamente e não declara que Instagram, TikTok Shop, Shopify ou qualquer outro toolkit está conectado. Para multiusuário, a próxima evolução deve usar uma sessão Composio por `organization_id`/usuário, persistir somente referências cifradas e aplicar scopes mínimos por departamento.

## xAI / Grok por API

A API oficial da [xAI](https://docs.x.ai/overview) é OpenAI-compatible na Responses API. O preset [`examples/dz23-xai.json`](../../examples/dz23-xai.json) encaminha `/v1/responses` e `/v1/chat/completions` para `https://api.x.ai/v1`, usando `XAI_API_KEY` somente no processo do servidor:

```bash
export OLLAMA_DZ23_CONFIG="$PWD/examples/dz23-xai.json"
export OLLAMA_DZ23_GATEWAY_KEY='chave-do-cliente-fora-do-repositorio'
export XAI_API_KEY='chave-xai-fora-do-repositorio'
```

No cliente compatível com Responses API, use o modelo `xai/grok-4.7` e envie `input`, `tools` e as opções suportadas pela versão da API. O proxy Ollama Full preserva o corpo Responses, reescreve apenas o identificador lógico para o modelo upstream e aplica autenticação server-side. Isso integra a **API xAI**, não o produto hospedado Grok Bot. Browser, computador cloud persistente, bots coordenados, skills e rotinas continuam sendo implementados pelo runtime próprio do Ollama Full ou pelos adapters aprovados, sem copiar internals proprietários.

## Redes sociais, afiliados e marketplaces

A base atual tem Growth OS sandbox, conectores HTTP allowlisted, OAuth tenant-aware, MCP e approvals. Ela consegue planejar campanhas, criar rascunhos, manter catálogo/pedidos locais, registrar atribuição e simular fulfillment; não publica nem vende em uma plataforma externa sem um connector específico e credenciais autorizadas.

| Canal | Estado atual | Próximo adapter operacional |
|---|---|---|
| Instagram/Meta | Adapter genérico e catálogo Composio possível; publicação não validada no Ollama Full | Meta Login/OAuth, `instagram_business_content_publish`, mídia pública, webhooks, rate limit, approval e teste em conta profissional |
| X/Twitter | Toolkit Composio listado; não há conexão validada no projeto | OAuth, publicação/leitura permitida, rate limits, políticas de automação e approval |
| YouTube | Toolkit/API pública disponível; não há upload validado no projeto | OAuth Google, upload/resumable, metadata, quota, copyright e approval |
| WhatsApp | Connector HTTP/MCP possível; não há fluxo Cloud API validado | Meta Business, templates, opt-in, webhooks, proteção contra spam e approval |
| TikTok Shop | Growth sandbox; APIs oficiais cobrem catálogo, pedidos, fulfillment, promoções, finanças, webhooks e Affiliate Seller/Creator/Partner | App Partner Center, seller/creator authorization, scopes por região, sandbox, webhooks assinados, idempotência, returns/refunds, compliance e testes por mercado |
| Shopify | Pode ser conectado por Composio ou connector dedicado; não há OAuth/Admin GraphQL validado | App OAuth, scopes mínimos, produtos/pedidos, webhooks, rate limits e approval para mutações |
| Outros marketplaces | Connector genérico/MCP permite integração futura; nenhum marketplace é declarado conectado | Adapter específico por marketplace, catálogo, estoque, pedidos, logística, devoluções, pagamentos e reconciliação |

A documentação oficial consultada informa que as Affiliate APIs do TikTok Shop não estão disponíveis no Reino Unido e União Europeia e que o onboarding de creators não pode ser totalmente moderado por parceiros via API. Logo, a jornada “como TikTok Shop” é viável **arquiteturalmente** e pode ser implementada com autorização de seller/creator/partner, mas não pode ser marcada como operação real universal antes da aprovação do app, da região, dos escopos e da sandbox correspondente.

Nenhum connector deve publicar posts, iniciar anúncios, enviar mensagens, criar produtos, alterar preço/estoque, aprovar pedidos, solicitar fulfillment, cobrar ou movimentar dinheiro sem approval explícito, idempotency key, trilha de auditoria, limites de orçamento e política de pausa automática.


## Lifecycle operacional de plugins

Connectors, MCP stdio, Remote MCP e skills possuem lifecycle explícito no runtime. A UI pode solicitar habilitar, desabilitar ou remover um recurso, mas a decisão é server-side e revalida tenant, allowlist, estado e capabilities antes de alterar o registro. Desabilitar bloqueia a execução sem apagar credenciais; remover exige uma ação explícita e não remove secrets externos.

O Composio, xAI/Grok, Desktop Commander e canais de Social Commerce permanecem adapters opt-in. O Ollama Full fornece contratos, presets sem segredos, headers server-side, approvals e testes locais. Connected accounts, OAuth, quotas, app review, webhooks, device pairing e publicação real só podem ser promovidos após smoke autorizado e reversível.


## Cadastro durável e estados de conexão

O lifecycle agora distingue quatro estados que não devem ser colapsados na UI. **Registrado** significa que o runtime aceitou a configuração e a operação allowlisted. **Configurado** significa que o manifest durável contém endpoint e referências de autenticação. **Credencial presente** significa apenas que o nome de env resolve um valor no processo ou que há uma credencial OAuth local para a organização. **Upstream validado** exige uma chamada real, reversível e autorizada ao serviço, com evidência de conta, escopos, resposta e redaction. O catálogo e o booleano `credential_configured` nunca afirmam o quarto estado.

Na execução, `TokenEnv` e `OAuthProvider` são referências obrigatórias quando declaradas. Se a variável não resolve valor ou a credencial OAuth da organização não está disponível, `CallForOrganization` retorna `connector credential is unavailable` antes de construir ou enviar a requisição; um connector autenticado não faz egress silencioso.

Quando `OLLAMA_AGENT_CONNECTORS` está vazio, o `ConnectorManager` carrega e grava `OLLAMA_AGENT_STORE/connectors.json`. O manifest tem permissões restritas e persiste somente `token_env`, `oauth_provider`, URL HTTPS, métodos e prefixos. Quando `OLLAMA_AGENT_CONNECTORS` aponta para um arquivo, esse arquivo é tratado como fonte estática de bootstrap; alterações feitas pelo endpoint não são descritas como edição persistente desse manifest. O endpoint `POST /api/agent/v1/connectors` exige owner/admin em auth mode, força o tenant da sessão, usa `DisallowUnknownFields` e recusa token, api key, password, ciphertext ou qualquer outro segredo cru.

A tela Plugins segue a mesma regra: ela permite cadastrar a referência de env/OAuth, habilitar ou desabilitar e remover o registro, mas não coleta senha nem cola token no browser. Composio, Google Workspace, GitHub, Woovi/OpenPix, APIs fiscais/NF-e, redes sociais e marketplaces continuam disponíveis como adapters/configuráveis ou exigem ação do operador. Nenhum desses nomes deve ser renderizado como “conectado” sem smoke autorizado no upstream correspondente.


## MCP, Remote MCP e skills: registro não é conexão

O lifecycle persistente agora cobre três classes. MCP stdio é um manifest local com executável absoluto regular, workspace e métodos allowlisted. Remote MCP é um manifest HTTPS com SSRF/DNS pinning, origin de redirect e referências de env. Skill é um manifest de capacidade que inicia não confiável e exige revisão/approval para qualquer confiança posterior. Todos os três ficam escopados à organização quando cadastrados pelo endpoint autenticado; IDs já pertencentes a outro tenant não podem ser sobrescritos.

O estado **registrado** significa que a configuração passou pelas validações. O estado **persistido** significa que o runtime padrão escreveu o manifest no DataRoot. Isso não significa que um processo MCP foi iniciado com sucesso, que um endpoint remoto respondeu, que um token existe ou que uma skill tem código confiável. `credential_configured` e estado OAuth continuam separados de upstream smoke. O modo de bootstrap por `OLLAMA_AGENT_MCP`/`OLLAMA_AGENT_REMOTE_MCP` permanece estático e é documentado como tal.

A tela Plugins permite o cadastro sem receber tokens, passwords ou conteúdo de credenciais. Para Desktop Commander, Composio, Google Workspace, GitHub, Woovi/OpenPix, fiscal/NF-e, redes sociais e marketplaces, o produto fornece contratos e pontos de configuração, mas uma conta real exige provisionamento seguro, scopes mínimos, consentimento, approval e homologação do serviço. Nenhum adapter é reportado como conectado por ter sido apenas registrado.


## Deploy: adapter local versus publicação externa

O runtime possui adapter local para generic, Vercel e Netlify com coleta limitada do workspace, referências de token por nome de variável, HTTPS obrigatório para endpoints remotos, redirects bloqueados, root sem symlink e verificação do endereço conectado. O smoke automatizado usa servidor fixture e não representa publicação externa.

**Configured** significa que o manifest foi aceito e o token é referenciado por nome de ambiente. **Upstream validated** exigiria chamada real autorizada, projeto/conta válidos, resposta redigida, evidência de URL/status, aprovação de custo e caminho de rollback. Vercel, Netlify, AWS, Cloudflare e demais destinos continuam `available/configurable`; nenhuma conta ou publicação externa está conectada ou homologada.


## Mídia: contrato local versus provider externo

O `MediaManager` cobre imagem, vídeo, speech, transcrição, visão e tone local com limites, artifacts, MIME/magic e provider OpenAI-compatible/loopback. Inputs de áudio/imagem ficam confinados ao workspace e outputs não gravam em root symlink. Os testes usam fixtures TLS ou geração determinística local.

Isso significa **adapter testado**, não modelo externo conectado. Claude, Grok, providers de vídeo/imagem/TTS/STT, GPU, quota, moderação, billing e qualidade de produção exigem configuração do operador e smoke autorizado por provider; nenhum é promovido a `upstream validated` por esses testes.


## Company Growth OS: sandbox versus commerce real

Campaigns, affiliate programs/links e orders do Company OS carregam `mode: sandbox`, e o relatório expõe `sandbox_only`. O runtime simula planejamento, approval, budget, inventário, atribuição, métricas e fulfillment com tracking local. Isso não publica em Instagram/TikTok/WhatsApp, não cria checkout, não cobra via Woovi/OpenPix, não compra em marketplace, não solicita logística e não emite NF-e.

Qualquer modo não-sandbox é recusado até existir adapter específico, credencial tenant-aware, approval de ação, idempotency key, webhook/reconciliação, limites financeiros, rollback e smoke autorizado. Woovi/OpenPix, fiscal/NF-e, redes sociais, TikTok Shop, Shopify, Mercado Livre, Amazon Seller e redes de afiliados permanecem `available/configurable`, nunca `connected`.


## Reauditoria de mídia e deploy

Os adapters locais agora têm contratos de segurança mais fortes, mas continuam distintos de uma integração upstream validada. O `MediaManager` usa provider fixture local nos testes; nenhum provider externo ou conta foi conectado. O `DeploymentManager` monta e valida um pacote público, rejeita conteúdo privado antes da transmissão e protege o egress; nenhum deploy em Vercel, Netlify, AWS, Cloudflare ou outro destino foi executado.

A captura de dez rotas usa o backend Ollama local e um bridge local para o bundle distribuído. Ela comprova apenas o estado observado nessa execução. Não transforma catálogo, configuração, fixture, `configured` ou build em `connected`, `validated`, `published` ou `production-ready`.


## Tel-Agent: canal textual versus telefonia externa

O estado publicado em `45393d8c` cobre `tel-agent.text` dentro do Company OS. O canal aceita operações locais allowlisted, persiste histórico redigido por tenant e exige role operacional para mutações autenticadas. `report.read` retorna o estado da empresa; `backlog.create` cria uma tarefa local; `campaign.draft` cria um rascunho sandbox com approval pendente.

Isso não equivale a telefonia ou mensageria externa. SIP, PSTN, SMS, WhatsApp, gravação de voz e discagem permanecem `telephony: not_configured` e `BLOCKED_BY_EXTERNAL_DEPENDENCY` até contas, credenciais, consentimento, destino de teste e homologação autorizada. Nenhum catálogo ou tela de configuração deve ser interpretado como conta conectada.


### Projeções seguras dos catálogos e limites operacionais

Os catálogos tenant-facing de Remote MCP, Connector e Deployment omitem recursos ownerless/globais e mostram apenas a origem da URL configurada; paths, query strings, fragments e mapeamentos de env/headers que indicam credenciais não devem ser usados como configuração executável pelo cliente. O manager mantém a configuração completa somente no servidor. `terminal.exec` resolve binários allowlisted em caminhos absolutos confiáveis, sem confiar no PATH herdado.

O push outbox aceita até 256 registros globais (64 por organização), até 64 KiB por registro e 20 MiB de snapshot; cada flush processa no máximo 32 notificações durante até 30 segundos e respeita cancelamento. Há no máximo oito tentativas, com compactação de itens terminais após sete dias. Respostas de deployment são limitadas a 4 MiB e JSON 2xx inválido/sem identidade não é tratado como sucesso. Quotas excedidas e respostas inválidas exigem investigação operacional do backlog/provider; não copie corpos upstream para logs ou tickets.
