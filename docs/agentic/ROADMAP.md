# Roadmap executável — Ollama DZ23 Agentic Platform

## Fase 0 — Base, contratos e gates — CONCLUÍDA LOCALMENTE

Consolidar os contratos versionados, checkpoint, build, testes, threat model, política de secrets e observabilidade. Esta fase inclui a ponte Claude/Codex já implementada e o primeiro pacote de documentação agentic.

## Fase 1 — Núcleo vertical de missões — CONCLUÍDA LOCALMENTE

Implementar o store, estados, eventos, planner, executor, observer, recovery, aprovação e artifact manifest. O fluxo comprovado será: criar missão, gerar plano, executar leitura autorizada, persistir resultado, produzir manifesto e consultar eventos.

## Fase 2 — Ferramentas seguras — CONCLUÍDA LOCALMENTE

Adicionar filesystem por workspace, terminal com executável e argumentos separados, execução de código em sandbox, limites de CPU/memória/tempo/rede, logs redacted e kill seguro. A execução arbitrária permanece bloqueada até a policy fornecer scope e aprovação.

## Fase 3 — Projetos, memória, skills e MCP — CONCLUÍDA LOCALMENTE

Criar projetos com permissões e contexto, memória episódica e semântica com retenção configurável, importação de arquivos, skill manifests assinados ou confiáveis e lifecycle de MCP servers. Cada fonte de contexto terá origem e nível de confiança.

## Fase 4 — Jobs e automações — CONCLUÍDA LOCALMENTE

Adicionar fila persistente, scheduler, retries, idempotency keys, webhooks verificados, dead-letter queue e replay. Integrações externas serão adapters com secrets server-side, scopes mínimos e confirmação para efeitos sensíveis.

## Fase 5 — Browser e computer use — ADAPTERS IMPLEMENTADOS; PROVA NATIVA PENDENTE

Integrar browser isolado com perfis, downloads, uploads, navegação, screenshots, ações e takeover humano. Integrar Desktop companion para tela, mouse, teclado, clipboard e processos usando capability grants. Nenhuma dessas capacidades será simulada por texto.

## Fase 6 — Artefatos multimídia e builders — ADAPTERS IMPLEMENTADOS; PROVIDERS/DEPLOY PENDENTES

Adicionar documentos, slides, planilhas, gráficos, dashboards, imagens, áudio, voz, transcrição, vídeo, sites, aplicativos e jogos. Cada domínio deve possuir renderer, preview, export, manifest, checksum e smoke test.

## Fase 7 — Superfícies de produto — PARCIAL

Construir API pública versionada, CLI agentic, Web/Desktop com timeline de missão, diff, approvals, logs, artifacts e terminal controlado. Construir aplicativo Mobile para chat, inbox de missões, approvals, notifications, artifacts e controle de projetos.

## Fase 8 — Integrações e colaboração — PARCIAL

Adicionar GitHub, Google Workspace, e-mail, Slack, Discord, WhatsApp e outros connectors por adapters. Implementar organizações, usuários, papéis, RBAC/ABAC, auditoria, compartilhamento, comentários, presença e colaboração em tempo real.

## Gates por fase

Cada fase exige testes unitários, contratos, autorização negativa, integração real quando o adapter existir, segurança, observabilidade e documentação. Uma feature fica `PARTIAL` enquanto seu adapter não tiver execução real e smoke reproduzível.

## Ordem de investimento

A prioridade é segurança e recuperação, depois execução real, depois conectores e superfícies. A interface não será usada para mascarar lacunas de runtime. O próximo incremento executável é substituir os adapters locais por stores/filas distribuídos, configurar providers OAuth/media/deploy, executar smoke tests em Windows/macOS/Android/iOS e fechar o fluxo de publicação no GitHub.


## Incremento 2026-09-21 — Multiagente, pesquisa e dispositivos

Foi implementado um AgentOrchestrator com sete papéis especializados, concorrência limitada, retries, orçamento, cancelamento, persistência e reducer com conflitos. O ResearchEngine agora executa pesquisa multi-fonte com cache, citações, hash, extração HTML, robots policy e SSRF guard. O DeviceStore adiciona pairing one-time, capability report, heartbeat, listagem e revogação. O próximo incremento deve conectar o reducer a um modelo de síntese validado, adicionar fontes PDF/OCR, WebSocket/mTLS e testes físicos dos companions.


## Incremento 2026-09-21 — Infraestrutura distribuída e transporte seguro

A fase adicionou adapters opcionais de PostgreSQL, Redis e OpenTelemetry, WebSocket de companion com TLS/mTLS policy, stack Docker Compose de desenvolvimento, workflow de qualidade/SBOM e fila offline no mobile. A próxima etapa de produção deve validar Redis/PostgreSQL reais em CI, configurar RLS/tenant isolation, rotação mTLS, push remoto, resolução de conflitos mobile, exporters persistentes e assinatura/rollback das releases.


## Incremento 2026-09-21 — SSO enterprise, credenciais por tenant e histórico visual

A fase adicionou adapter SAML baseado em `crewjam/saml`, incluindo metadata, AuthnRequest assinado, RelayState one-time, ACS e provisionamento de claims; conectores com resolução de access token OAuth cifrado por organização; RLS PostgreSQL forçado para impedir bypass pelo dono da tabela; TLS 1.3/mTLS opcional com recarga de certificado por handshake; e canvas visual com bindings, eventos, histórico persistente, undo e redo. Permanecem pendentes os testes ponta a ponta com IdP, serviços distribuídos reais fora do CI, providers multimídia/deploy, testes físicos de companions e distribuição assinada.


## Incremento 2026-09-21 — Publicação real de builders

Foi adicionado o DeploymentManager com adapters Vercel, Netlify e generic, coleta segura de arquivos, limites, redirects bloqueados, token server-side e approval explícito. O próximo gate é executar smoke contra contas reais e completar adapters de AWS/Cloudflare conforme credenciais e requisitos de cada ambiente; a publicação externa nunca é simulada como concluída apenas por existir um preview local.

## Incremento 2026-09-22 — Árvore de produto, shell desktop e OmniRoute

Foi criada a árvore pública [`PRODUCT_TREE.md`](PRODUCT_TREE.md), que separa a superfície observável de um desktop agentic, a árvore atual do Ollama Full e a árvore-alvo unificada com Claude, Codex, OmniRoute, coding agents, pesquisa, builders, mobile, colaboração e operações. A matriz [`PARITY_MATRIX.md`](PARITY_MATRIX.md) passa a ser o contrato de honestidade para diferenciar `VALIDADA LOCALMENTE`, `ADAPTER IMPLEMENTADO`, `PARCIAL` e `PENDENTE`.

O shell web ganhou rotas reais para Biblioteca, Projetos, Agendado, Habilidades, Plugins e Tarefas, além de uma navegação lateral com Nova tarefa, Agente, workspace e conta. A Settings ganhou o Agentic Control Center com catálogo de modelos, status sanitizado de runtime, approvals, isolamento, conectores, MCP, mídia, deploy e OTLP. O endpoint `/api/agent/v1/config/safe` nunca retorna tokens, caminhos privados ou valores de configuração.

O multi-provider agora aceita HTTP somente para um serviço numérico de loopback quando `allow_private` e `allow_insecure_loopback` estão explicitamente definidos. Isso permite o preset [`examples/dz23-omniroute.json`](../../examples/dz23-omniroute.json) para uma instância OmniRoute local, mantendo HTTPS obrigatório para hosts externos. Foram adicionados testes de carregamento, rejeição de HTTP externo e encaminhamento com bearer server-side. A integração continua dependente de uma instância OmniRoute, credencial e smoke test do operador.

As telas novas têm estados vazios honestos e ações de entrada. CRUD persistido, colaboração, editor drag-and-drop, provider discovery completo e deploy real continuam como próximos gates; a interface não os apresenta como concluídos.


## Incremento 2026-09-22 — fluxo vertical, home e HarnessRouter

A rodada substituiu estados estáticos do shell por contratos reais: Projetos e Agendado possuem criação, listagem, edição/exclusão com escopo de organização; Tarefas e Biblioteca consultam missões/artifacts; Plugins e Habilidades consultam catalogs server-side; e o Agentic Console cria missões com provider (Ollama local, Claude, Codex, OmniRoute) e projeto selecionáveis. A home agora possui composer, recomendações e atalhos funcionais, com fallback offline local-first.

O proxy multi-provider ganhou `harness_id` declarativo por modelo e injeção server-side de `metadata.harness_id`, permitindo o preset opcional [`examples/dz23-harnessrouter.json`](../../examples/dz23-harnessrouter.json) para Codex e Claude Code no [HarnessRouter](https://github.com/HarnessRouter/harnessrouter). Isso é adapter implementado; não é validação de uma instância ou credencial externa.

Provas desta rodada: `go test ./internal/agent ./server`, build Vite, smoke CRUD real contra servidor local, smoke de missão/eventos, captura Chromium das nove rotas e `node app/ui/app/scripts/smoke-shell.mjs` com home, Projects, Scheduled, Plugins, Skills e seleção Claude. O próximo gate é HarnessRouter real com streaming/follow-up/cancelamento/artifacts, seguido de editor drag-and-drop/CRDT e testes distribuídos/IdP/dispositivos.


## Incremento 2026-09-22 — Company OS e Desktop Commander Remote MCP

A base passou a incluir um Company OS persistente. Uma empresa é criada no tenant ativo e recebe identidade, posicionamento, modelo de negócio, departamentos virtuais, roadmap, metas, backlog, ciclos, budget, relatório e risco. Ciclos são ligados a schedules persistentes e o runtime interrompe a criação de novas missões quando a empresa está pausada por ação do operador, orçamento excedido ou anomalia grave.

Também foi implementado um adapter Remote MCP Streamable HTTP e o preset oficial do Desktop Commander. O runtime valida HTTPS fora de loopback, mantém allowlist de métodos, usa bearer apenas por variável server-side e expõe `mcp.remote.call` com approval. O transporte não é confundido com autenticação: OAuth PKCE, conta, device pairing, agente `npx ... remote`, revogação e testes físicos permanecem dependentes do operador e do serviço oficial.

Os próximos gates do Company OS são connectors reais com CRM, e-mail, redes sociais, anúncios, afiliados, ecommerce, logística e analytics, sempre com sandbox, scopes mínimos, DLP, approval e testes de compliance. Os próximos gates do Remote MCP são completar um fluxo OAuth autorizado, parear um dispositivo de teste, executar apenas jornadas reversíveis e validar revogação, logs e desligamento. Nenhum desses gates é simulado pela presente revisão.


## Incremento 2026-09-22 — jornadas verticais de Growth, Builder e provider sessions

O Company OS ganhou um **Growth OS local** com campanhas, aprovação, pausa, programas de afiliados, links HTTPS, conversões, catálogo de produtos, pedidos e fulfillment sandbox. O painel `/company` exibe métricas de campanhas ativas, programas, conversões, produtos e pedidos. O smoke `scripts/smoke-company-growth.sh` comprovou criação, bloqueio sem approval, aprovação, lançamento, conversão, estoque e relatório agregado sem chamar serviços externos.

O Builder foi exercitado por API com criação de site, atualização visual, preview com hash de artifact, exportação, publicação local e rejeição do booleano `approved:true` legado em `scripts/smoke-builder.sh`; o fluxo válido de deploy externo agora requer approval server-side persistente, nonce e consumo único. O gateway multi-provider ganhou cobertura de uma sessão OpenAI-compatible streaming, autenticação de entrada e bearer server-side, incluindo tradução dos chunks para o protocolo nativo.

Estas jornadas são operacionais no ambiente local. O Growth OS não publica anúncios, envia mensagens, compra produtos, cobra clientes ou chama marketplaces; o Builder não foi promovido a deploy externo validado; provider sessions externas continuam dependendo de credenciais, instâncias e harnesses instalados. O próximo passo é conectar cada adapter externo a um sandbox autorizado com idempotência, scopes mínimos, logs, DLP, approval e rollback.


## Incremento 2026-09-22 — Composio, xAI/Grok e social commerce

O runtime ganhou suporte a headers server-side no Remote MCP e um preset Composio Connect com allowlist de JSON-RPC e approval. Isso torna possível usar os toolkits Composio por MCP, mas a conexão de cada app continua dependente de OAuth, connected account, scopes e políticas do upstream. A API xAI também recebeu preset HTTPS para Responses/chat; o proxy comprovadamente preserva `input` e `tools` e aplica a chave no servidor, sem afirmar disponibilidade de quota, modelo ou Grok Bot hospedado.

O Growth OS agora tem o mapa técnico para sair do sandbox local rumo a social commerce: Instagram/Meta, X/Twitter, YouTube, WhatsApp, TikTok Shop, Shopify e outros marketplaces. O próximo trabalho por canal deve criar contrato de provider, OAuth por tenant, webhooks verificadas, idempotência, rate limits, DLP, approval e smoke em sandbox. No TikTok Shop, seller/creator/partner authorization, escopos por região e a disponibilidade das Affiliate APIs precisam ser tratados por mercado; a documentação consultada informa indisponibilidade atual para Reino Unido e União Europeia.

As integrações externas ainda não são declaradas conectadas. A conclusão desta fase exige contas de teste autorizadas, app reviews, sandbox dos provedores, credenciais provisionadas fora do Git e testes reversíveis de publicação, catálogo, pedido, fulfillment, returns/refunds e reconciliação.


## Incremento 2026-09-22 — hardening interno e baseline de release

A auditoria independente encontrou falhas internas em autenticação, filesystem, MCP, approvals, tenant scope, idempotência, UI, mobile e vet. A correção desta rodada tornou o agentic fail-closed por padrão quando o listener não é loopback; manteve o modo local-first apenas para bind local; exigiu allowlists não vazias em MCP stdio e Remote MCP; rejeitou redirects cross-origin e destinos privados; validou correlation IDs; reforçou a contenção de symlinks; validou StepID; adicionou metadata de tenant, policy, nonce, actor e expiração às approvals; aplicou tenant checks nas mutações de Growth; adicionou idempotência a pedidos e fulfillment; tornou o catálogo da Settings resiliente a respostas nulas; e retirou bearer tokens do outbox mobile.

Os gates locais desta etapa passaram com `CGO_ENABLED=1 go test ./... -count=1`, `CGO_ENABLED=1 go vet ./...`, `CGO_ENABLED=0 go test ./internal/agent -count=1`, testes do servidor/multi-provider, build Go, build da UI, 20 arquivos Vitest/199 testes e typecheck Expo mobile. O teste `CGO_ENABLED=0 go test ./...` continua `N/A` para os pacotes que exigem CGO, como SQLite/MLX; os gates oficiais do fork usam a combinação compatível com esses componentes.

Permanecem bloqueadores externos honestos: credenciais e sandbox de Composio, xAI, Desktop Commander, redes sociais, TikTok Shop e marketplaces; PostgreSQL/RLS, Redis e OTLP reais; IdP e app review; GPU/modelos multimídia; runners físicos; assinatura de instaladores; publicação nas lojas; e deploy externo autorizado. Esses itens não podem ser declarados concluídos sem ambiente, autoridade e testes correspondentes.


### Policy de capabilities e handshake — 2026-09-22

O Runtime passou a persistir escopos por missão, com default limitado ao workspace local. Tools sensíveis exigem capability explícita e continuam sujeitas a approval. O middleware também restringe o bypass de bearer ao FullPath único do handshake de companion; qualquer rota arbitrária terminada em `connect` permanece protegida.


## Incremento 2026-09-22 — capacidades operacionais e lifecycle

Esta rodada transformou mais adapters em jornadas verificáveis. O runtime ganhou Grok Live local-first com Responses API, streaming, retries limitados, circuito de falha e status sanitizado. A distinção entre fontes ao vivo e memória ficou explícita. O Evaluation OS agora oferece casos determinísticos para coding, browser, tools, segurança, memória, planejamento e recuperação. O provider router considera capacidades, saúde, latência, custo, privacidade e qualidade histórica.

O Company OS passou a expor agentes departamentais persistentes com supervisor, orçamento, SLA, memória, pausa e retomada. O Social OS ganhou contratos sandbox para contas, drafts, approvals, publicação controlada e métricas. A UI Company Operations conecta esses estados ao backend sem declarar contas externas como conectadas.

Plugins, MCP e skills agora possuem lifecycle explícito de habilitar, desabilitar e remover. O servidor mantém a decisão e o bloqueio de execução; a UI apenas solicita a mutação. O catálogo continua sanitizado e manifests sem atestado não recebem confiança executável.

O fluxo de deploy do Builder foi corrigido para verificar approval antes de consultar provider ou retornar ausência de configuração. Os smokes finais passaram para Growth OS, Builder e shell E2E, incluindo criação real de projetos, schedules e missões.

A promoção para produção continua condicionada a credenciais e ambientes externos. Permanecem pendentes os smokes autorizados de xAI/Composio/Desktop Commander, Social Commerce/TikTok Shop/Meta/Shopify, PostgreSQL/RLS/Redis/OTLP, IdP, GPU, runners físicos, assinatura de instaladores, lojas e deploy externo.


## Incremento 2026-09-22 — prontidão e smoke de APIs

Foi executado um smoke seguro com as chaves fornecidas pelo operador: catálogos OpenRouter, Groq, DeepSeek, Fireworks, Cerebras, Mistral, NVIDIA, Novita, Cohere, Gemini, Hugging Face, GitHub e Cloudflare responderam; Together, xAI, Hyperbolic e Alibaba foram recusados; o endpoint tentado de Voyage retornou 404 e não foi interpretado como prova de validade ou invalidade. Uma inferência curta no modelo gratuito do OpenRouter respondeu diretamente e também através do gateway Ollama Full local, comprovando o caminho provider → gateway → cliente.

A auditoria do CI encontrou falha no teste real do Browser Operator porque o job não instalava a dependência Python Playwright/Chromium. O workflow agora instala a dependência e exporta o executável descoberto; o teste específico passa localmente. O CI remoto do novo commit ainda precisa concluir para fechar essa pendência.

A classificação continua **release candidate local-first**, não produção universal. O relatório [`READINESS_2026-09-22.md`](READINESS_2026-09-22.md) é a fonte de verdade para os resultados, a rotação obrigatória das chaves fornecidas e os blockers externos restantes.


## Incremento 2026-09-22 — hardening de egress e lifecycle administrativo

A revisão de prontidão foi ampliada com escopo obrigatório de organização para connectors, matching seguro por segmento, bloqueio de destinos privados após resolução DNS, validação estrita de headers/ambiente no MCP e autorização administrativa para lifecycle global de plugins. O Browser Operator ganhou fallback de executável Chromium e teste de regressão com caminho ausente.

O resultado fecha riscos internos de baixo nível sem afirmar produção universal. Os próximos gates continuam sendo distribuídos (PostgreSQL/RLS, Redis, OTLP), IdP/OAuth, providers e toolkits com contas autorizadas, canais de social commerce, dispositivos físicos, GPU, assinatura, lojas e deploy externo.


## Incremento 2026-09-22 — primeiro slice P0 de ownership

A primeira correção pós-auditoria ampliada protegeu Builder por organização e fechou invariantes server-managed do Company create e do gasto de agentes. O slice inclui testes negativos para duas organizações, entry inexistente, XSS em template, symlink e excesso de budget. O próximo trabalho deve levar o mesmo padrão a orchestration, traces, devices/pairing, plugins/MCP/skills e artifacts, seguido da política de approvals separada de execute.

O resultado permanece **preview/local RC em hardening**. A existência de teste local não comprova RLS distribuído, sandbox forte, providers externos, dispositivos, assinatura ou deploy de produção.


## Incremento 2026-09-22 — orchestration, traces e devices scoped

A segunda slice pós-auditoria adicionou ownership de organização a orchestration jobs, spans de missão/ferramenta e devices/pairing. Os handlers de plan/get/run/cancel/list/heartbeat/revoke aplicam o tenant da requisição, e testes negativos confirmam `403`/no-mutation entre duas organizações. O próximo foco é fechar plugins/MCP/skills/artifacts e a policy de approvals, sem tratar os testes em memória como prova de RLS ou operação distribuída.

O resultado permanece **preview/local RC em hardening**; TLS/mTLS, companions físicos, Redis/PostgreSQL/OTLP reais e os P0 de sandbox/egress/DLP ainda dependem de ambientes apropriados.


## Incremento 2026-09-22 — approvals de missão com CAS

Approvals de missão agora têm policy owner/admin em auth mode, nonce de uso único, razão obrigatória e CAS da versão da missão. O próximo trabalho de approvals deve migrar Company/Growth/Social de booleans caller-controlled para decisões auditáveis com actor, policy, nonce, expiração, idempotência e proteção contra auto-approval, antes de qualquer efeito externo.

A classificação permanece **preview/local RC em hardening**; esta mudança não prova autorização empresarial distribuída nem aprovação dos adapters externos.


## Incremento 2026-09-22 — CompanyApproval ledger

O Company OS passou a usar uma decisão auditável para campanhas, afiliados, pedidos e social drafts. O próximo passo imediato é retirar o booleano caller-controlled da rota de gasto e aplicar o mesmo ledger a orçamento, anúncios, contratos e mensagens externas; em paralelo permanecem sandbox/MCP process isolation, egress/DLP e ownership completo de plugins.

A classificação continua **preview/local RC em hardening**. O workflow de afiliados/dropshipping/social permanece sandbox até connectors OAuth e contas externas serem realmente configurados e testados.


## Incremento 2026-09-22 — spend approval e budget atômico

A autorização de gasto deixou de ser um checkbox: a solicitação é criada sem débito, fica pendente na fila de approvals e só altera o budget após decisão auditável com nonce/CAS. A próxima prioridade de segurança é aplicar a mesma separação a integrações externas, MCP/egress, DLP de resultados e ownership de plugins; esses itens não são considerados concluídos por esta slice.


## Incremento 2026-09-22 — DLP de resultado/observabilidade

A primeira camada de DLP agora protege resultados de ferramentas, eventos, traces e persistência. O próximo slice de egress deve definir classificação de dados, injeção explícita de credenciais e redaction de payloads externos sem quebrar integrações autorizadas; SSRF/DNS rebinding, redirects e sandbox/MCP process isolation continuam pendentes.


## Incremento 2026-09-22 — Remote MCP egress

Remote MCP agora tem proxy nil, redirect same-origin e validação do IP conectado para reduzir DNS rebinding. O próximo trabalho de egress deve alinhar Media e Connectors ao mesmo contrato, incluindo redirects, MIME/magic, tamanho máximo e testes de rede; sandbox/process isolation forte segue pendente.


## Incremento 2026-09-22 — MCP stdio lifecycle

A contenção de processo MCP stdio ganhou cwd privado, ambiente mínimo, allowlist de executável, limites de payload, stderr redigido e lifecycle de grupo/restart. O próximo nível requer implementação por plataforma de seccomp/cgroups/rlimits/PID/memória/CPU e testes em runners reais; até lá, a documentação deve tratar stdio como best-effort.


## Incremento 2026-09-22 — Media egress/download

Downloads de mídia agora compartilham a postura de proxy nil, redirect bloqueado, IP conectado, limite bounded e MIME/magic validation. O próximo trabalho deve alinhar upload/Connector egress à mesma política sem redigir credenciais operacionais autorizadas, além de executar testes distribuídos com DNS/TLS controlados.


## Incremento 2026-09-22 — Connector egress parity

Connectors agora compartilham bloqueio de proxy/redirect, verificação do IP conectado e limites bounded de payload com Remote MCP e Media. A próxima etapa precisa definir classificação/consentimento para payload externo e injeção de credenciais, além de cobrir upload e testes distribuídos sem afirmar integração externa conectada.


## Incremento 2026-09-22 — OAuth redirect URI allowlist

OAuth redirect agora depende de allowlist provider-scoped e canonicalização segura, com loopback HTTP explicitamente opt-in. A próxima etapa é aplicar egress/IP/redirect policy aos endpoints de discovery, JWKS, token e userinfo sem afirmar SSO conectado sem IdP real.


## Incremento 2026-09-22 — session handling web/mobile

Bearer web deixou de ser persistido no localStorage e sessões inválidas são limpas; o mobile protege SecureStore, approvals e descarte de outbox. A próxima etapa é ligar a jornada de login web ao setter in-memory e validar secure storage/biometria em builds reais de Android/iOS/desktop, sem afirmar isso no sandbox.


## Incremento 2026-09-22 — OAuth endpoint egress

Os endpoints OAuth/OIDC agora compartilham política de egress sem proxy/redirect e com IP efetivo seguro. Continua necessário executar um IdP de staging para provar discovery, JWKS, assinatura, audience, nonce, refresh e revogação sem simular credenciais ou declarar SSO de produção.


## Incremento 2026-09-22 — tool process containment

Tools agora têm lifecycle de grupo, redaction e limites best-effort reportados. A próxima fronteira de isolamento é implementar/adotar seccomp, cgroups, quotas e políticas específicas por plataforma; até isso ser testado, o runtime continua classificado como contenção parcial.


## Incremento 2026-09-22 — plugin/MCP/skill ownership

Lifecycle autenticado agora é tenant-aware para Connector, MCP stdio, Remote MCP e Skill. Recursos globais são read-only nesse caminho; a próxima etapa é implementar registro server-owned por organização, attestation de skills e isolamento de execução MCP por tenant.


## Incremento 2026-09-22 — CI/release quality gates

CI agora cobre Go/UI/mobile de forma explícita e o release depende desse quality job. A próxima validação deve ocorrer no GitHub Actions com Docker/RLS/Redis/OTLP, runners físicos e credenciais de assinatura; o sandbox local não substitui essas provas.


## Incremento 2026-09-22 — Grok/provider contract

O contrato Grok agora rejeita modelo arbitrário, valida catálogo no health e não promete streaming HTTP ainda. A próxima etapa de paridade é expor uma rota SSE com backpressure/cancelamento e testes reais contra o provider, sem esconder falhas de credencial.


## Incremento 2026-09-22 — Compose secure defaults

A stack local agora exige secrets e loopback binding. Falta executar e inspecionar a integração PostgreSQL/RLS, Redis/DLQ e OTLP em runner Docker real; essa prova não pode ser substituída por YAML estático.


## Incremento 2026-09-22 — provider selection contract

A seleção de provider deixou de ser apenas cosmética: o runtime valida e persiste o provider real. A próxima etapa é implementar adapters externos isolados, com credenciais por organização, health/allowlist, streaming e testes; até lá a UI mantém esses motores desativados.


## Incremento 2026-09-22 — release artifact integrity

O release valida presença/tamanho dos artefatos e oferece attestation condicional. A assinatura de instaladores/binários, SBOM ligado aos artefatos finais e execução real do release ainda dependem de credenciais, policies e runners do operador.


## Incremento 2026-09-22 — jobs/replay scoped

Queue list/replay agora consultam a organização da missão antes de expor ou mutar jobs. A validação distribuída Redis e recuperação após restart continuam pendentes no runner Docker.


## Incremento 2026-09-22 — artifact manifest path safety

A geração de manifests reutiliza a contenção de symlink do workspace. Exportação assinada, armazenamento distribuído e validações físicas de artefatos continuam pendentes.


## Incremento 2026-09-22 — distributed CI smoke remediation

A primeira execução pública do workflow ampliado revelou um defeito útil no próprio gate: o teste PostgreSQL RLS conectava com a role bootstrap superusuária, e o armazenamento tenant-scoped recusava corretamente essa configuração. O workflow foi ajustado para criar uma role de teste não-superusuária e manter passwords efêmeras apenas no passo local, sem `GITHUB_ENV`. A prova local está verde; o próximo passo é confirmar o novo run remoto com Docker e serviços distribuídos. A classificação do produto continua preview/local RC em hardening.


## Follow-up 2026-09-22 — bootstrap do smoke distribuído

O segundo run público encontrou uma falha de sintaxe no bootstrap condicional da role PostgreSQL e confirmou que steps separados não compartilham variáveis shell. A correção usa `psql -c` com password efêmera hexagonal no passo de start e placeholders neutros na limpeza. O gate remoto ainda precisa ser repetido; nenhuma classificação de produção foi alterada.


## Validação remota 2026-09-22 — distributed quality workflow verde

O run `35737772235` no commit `15e8ae60` passou todos os jobs de qualidade, incluindo PostgreSQL RLS, Redis DLQ e OTLP reais no runner GitHub. O próximo trabalho volta aos P0 internos ainda abertos: capability enforcement completo, sandbox/process isolation forte, egress/DLP residual, storage/IdP e contratos externos.


## Incremento 2026-09-22 — central capability enforcement

A policy de capabilities agora é central e deny-by-default no Runtime, com grants desconhecidos rejeitados, scopes obrigatórios em tools e aprovação vinculada aos scopes. O próximo trabalho de segurança permanece: sandbox forte com isolamento por worker/container, auditoria por chamada MCP/tool, capability policy por skill/projeto/tenant e prova distribuída de recovery/concurrency.


## Complemento Master V3 — matriz de referências — 2026-09-22

O complemento recebido foi incorporado como `audit/HARNESS_CAPABILITY_MATRIX.md`. Ele cataloga as 44 referências de produto e os 13 repositórios fornecidos, agrupando capacidades em engenharia, edição/contexto, builders, pesquisa, Company OS, memória/automação, providers, colaboração/dispositivos e execução. A decisão é preservar Ollama e o Runtime como núcleo, usar implementação nativa quando suficiente e adotar integrações substituíveis somente quando houver contrato, licença, isolamento e teste de aceite.

A matriz não converte triagem documental em integração operacional. Os gates externos permanecem abertos para sessões reais de harness, contas OAuth, hardware, deploy, lojas, sandbox forte, colaboradores/dispositivos e homologação distribuída.


## Remediação 2026-09-22 — upstream Go race

O workflow upstream revelou duas corridas que não apareciam nos gates agentic específicos: contador do fixture HTTP de pesquisa e `ReadFrom` promovido no buffer stderr do MCP. Ambas foram corrigidas sem relaxar o teste ou ocultar o race detector. O próximo gate é a execução remota do workflow `test` no novo SHA.


## Validação remota 2026-09-22 — agentic quality verde

O SHA `428a99bd` passou o workflow agentic completo no run `35749351291`. A próxima prova específica é o workflow upstream `test` no head corrente do PR, incluindo a matriz de plataformas aplicável.


## Incremento 2026-09-22 — bootstrap MCP verificável

O loader de manifestos MCP passou a ter regressões de servidor que exercitam o caminho de configuração real. O próximo trabalho de transporte continua sendo correlação/multiplexação JSON-RPC, sessão/revogação Remote MCP, auditoria por chamada e sandbox forte.


## Incremento 2026-09-22 — MCP JSON-RPC correlation

O transporte stdio agora separa notificações sem `id` da resposta correlacionada. Permanecem como trabalho de hardening: reader dedicado com multiplexação segura, cancelamento de requests individuais, framing/streaming completo, auditoria por chamada e sessão/reconexão Remote MCP.


## Remediação 2026-09-22 — Browser Operator upstream CI

A matriz upstream identificou uma dependência de teste ausente, não uma falha do transporte MCP: `playwright` não era instalado antes de `go test`. O patch pinou a dependência nos jobs `test` e `race` e tornou a escolha do Python portável. Falta o novo run remoto confirmar Linux/macOS/Windows; isso não encerra os blockers de sandbox forte, Browser/desktop real ou release.


## Follow-up 2026-09-22 — upstream CI sem fila infinita

A matriz nativa herdada foi retirada do caminho automático de pull request porque usa labels `linux`/`windows` e hardware GPU que não existem no ambiente público deste fork. Ela continua disponível apenas como execução manual opt-in quando o operador configurar runners compatíveis. O caminho automático foi corrigido para Playwright `1.63.0`; permanece pendente a prova remota do novo head e a homologação física de cada plataforma.


## Validação remota 2026-09-22 — upstream normal multiplataforma verde

O head `de0e86772e96372789c10d924eb5738f8808821b` passou o caminho normal do PR: `test` (`35769597404`), `dz23-agentic-quality` (`35769597363`), `dz23-multi-provider` (`35769597578`) e `class-a-plus-integrity` (`35769597628`). Os jobs upstream de teste passaram em Linux, macOS e Windows, e os jobs race passaram em Linux e macOS. O Browser Operator foi validado no fluxo remoto após a instalação pinada do Playwright e a descoberta portável do executável.

A matriz GPU/nativa continua manual e opt-in (`workflow_dispatch` + `run_native_matrix=true`), porque depende de runners compatíveis do operador. O próximo trabalho permanece nos P0/P1 internos e externos: sandbox forte, autorização/session/CSRF/IdP, OAuth lifecycle, egress residual, adapters reais, device validation, signing/provenance e homologação. A classificação não muda: preview/local RC em hardening.


## Hardening 2026-09-22 — Company idempotente, sessão revogável e terminal restrito

As operações críticas de Company agora possuem ledger de idempotência por digest: spend imediato ou pendente, conversão de afiliado e métrica social não duplicam efeitos em retry e rejeitam reuso de chave com fingerprint diferente. O ciclo de sessão ganhou logout server-side com revogação do bearer; o cliente preserva tokens em memória e diferencia 401 de 403. `terminal.exec` continua deliberadamente allowlisted e agora valida flags, argumentos e paths do workspace antes de iniciar o processo.

Os workflows do head `935fb273` passaram: integrity `35784211426`/`35784205466`, agentic quality `35784211429`/`35784205382`, multi-provider `35784211481` e upstream `test` `35784211483`. O próximo trabalho não é declarar encerramento: sandbox forte, OAuth/IdP/CSRF distribuído, contratos externos reais, hardware/dispositivos, signing/provenance e homologação permanecem pendentes. A matriz GPU/nativa segue manual e opt-in.


## Hardening 2026-09-22 — sandbox strict, mobile por tenant e release metadata

O sandbox agora possui um caminho Linux strict opt-in com cgroup v2 delegado, namespaces, seccomp multi-arquitetura, no-new-privs, limites e kill do cgroup no timeout; ausência de enforcement retorna erro em vez de degradar silenciosamente. A política CSRF/Origin cobre mutations autenticadas. No mobile, auth/session confirma o tenant antes de outbox/push, e cache/fila/registro são isolados por servidor e organização. O pacote também passou typecheck e export web.

O release workflow agora produz SBOM CycloneDX, metadata rastreável por commit/ref, checksum manifest e verificação antes do upload; attestation permanece opcional e nenhuma assinatura/loja foi executada. A próxima etapa deve triagem das 18 vulnerabilidades de produção reportadas pelo npm audit mobile (11 moderate, 7 high), conclusão dos runs do head e, depois, P0/P1 externos: host sandbox real, IdP/OAuth/providers/deploy/media, dispositivos, push remoto e signing/provenance.

A classificação continua **preview/local RC em hardening**. A matriz GPU/nativa segue manual e opt-in.


## Follow-up 2026-09-22 — audit mobile sem force upgrade

Os 18 achados transitivos do primeiro audit mobile foram tratados com overrides mínimos para `image-size`, `postcss` e `uuid`, sem migrar Expo/React Native major no escuro. O lock foi resolvido e `npm audit --omit=dev` agora retorna zero vulnerabilidades de produção; typecheck e export web continuam verdes. Migração de SDK major, testes físicos Android/iOS e distribuição permanecem tarefas de validação separadas.


## Validação 2026-09-22 — caminho normal de CI e6e0632b

O head `e6e0632b` fechou o caminho normal do PR com upstream test, integrity, multi-provider e agentic quality verdes. A matriz comprovou Linux/macOS/Windows, race aplicável, Browser Operator, Go/server, RLS + Redis DLQ + OTLP, Web/Mobile e SBOM; os gates Node usam `npm ci` com lockfiles. Permanecem fora do resultado os jobs nativos/GPU manuais e os skips condicionais.

O roadmap agora prioriza validações que não podem ser simuladas honestamente nesta sandbox: host strict com cgroup/AppArmor/SELinux, IdP/OAuth/provider real, deploy/media/marketplace, push e dispositivos físicos, signing/provenance/rollback e stores/app review. A classificação permanece **preview/local RC em hardening**.


## V5 — provider efetivo, catálogo de connectors e CI verde — 2026-09-22

O caminho interno avançou em quatro frentes. O planner agora usa provider/modelo real e não substitui falhas por um plano determinístico silencioso. O runtime separa estado durável do workspace e o Remote MCP usa pinagem de IP após DNS. O catálogo de connectors e a UI de Plugins mostram estado seguro e tenant-aware, enquanto o Mission Console libera somente providers/modelos publicados pelo runtime.

O upstream teve duas falhas diagnosticadas e corrigidas sem relaxar gates: helper Go não utilizado e colisão do cache npm global no Windows. O workflow agora isola o cache por runner e deixa test/race concluírem todas as plataformas. O head `0f95b6a1` passou upstream, integrity, multi-provider e agentic quality, com Linux/macOS/Windows, race Linux/macOS, PostgreSQL RLS, Redis DLQ, OTLP, Web/Mobile e SBOM.

A próxima prioridade continua sendo a validação que não pode ser simulada honestamente nesta sandbox: host strict com cgroup/AppArmor/SELinux, IdP/OAuth e providers reais, deploy/media/marketplaces, Woovi/OpenPix e fiscal, push e dispositivos físicos, signing/provenance/rollback e stores/app review. O estado permanece **preview/local RC em hardening**.


## Incremento 2026-09-22 — macOS CI e lifecycle durável de connectors

O head `0da6be80` fechou a portabilidade do workspace e foi validado pela matriz pública normal/race de Linux, macOS e Windows. A correção foi específica para o alias de filesystem `/var` → `/private/var` no macOS e não relaxou a proteção contra symlinks descendentes. O run upstream `35808941810` terminou com sucesso, e os workflows do fork `35808941807`, `35808941824` e `35808941746` também passaram nesse head.

O commit `411335ba` adicionou o primeiro registro server-owned para connectors. Em auth mode, somente owner/admin pode cadastrar; o servidor ignora qualquer tentativa de escolher outro tenant e rejeita colisões de ID entre organizações. Sem um manifesto estático explícito, configurações registradas são persistidas em `OLLAMA_AGENT_STORE/connectors.json` com permissões `0600`, rename após arquivo temporário e rollback quando a persistência falha. O manifest guarda nomes de variáveis e referências OAuth, não valores de credenciais. A UI agora oferece o fluxo correspondente e mantém a distinção entre registrado, credencial presente e conta upstream validada.

O próximo incremento deve aplicar o mesmo padrão, separadamente e com testes negativos, ao registro/persistência de MCP stdio, Remote MCP e skills. Esse trabalho não deve copiar segredos, permitir comandos fora da allowlist ou transformar OAuth em status conectado. Depois disso permanecem os gates de isolamento físico do sandbox, staging distribuído, OAuth/IdP e contas reais, providers externos, social commerce, fiscal/NF-e, dispositivos físicos, GPU, signing, lojas, app review e deploy autorizado.


## Incremento 2026-09-22 — MCP, Remote MCP e skills

Depois do lifecycle de connectors, o commit `96fd7edd` fechou o cadastro persistente dos outros três tipos de extensibilidade. O runtime padrão grava manifestos de MCP e Remote MCP no DataRoot e manifestos de skills no diretório privado de contexto. As operações são atômicas, os arquivos têm modo `0600`, e as alterações de enable/remove passam pelo mesmo caminho de persistência. Os testes de restart demonstram que o estado volta após recriar os managers.

O endpoint MCP aplica a política de executável absoluto regular e workspace preparado; o endpoint Remote MCP conserva a allowlist de métodos, HTTPS, SSRF, resolução/pinning de endereços aprovados, redirects same-origin e env-only credentials. O endpoint de skills grava `Trusted=false` independentemente do input e exige revisão/approval para qualquer confiança futura. A UI Plugins expõe os três cadastros e mantém a diferença entre manifest registrado, credencial presente e upstream validado.

O próximo trabalho independente é endurecer o storage/authorization distribuído e a homologação física do sandbox, não adicionar botões que simulem integração externa. Permanecem abertos: OAuth e refresh/revoke contra contas reais; egress/providers/social commerce/fiscal; isolamento físico host/device; Postgres/Redis/OTLP distribuídos; Windows/macOS/Linux/Android/iOS; GPU/native; signing, installers, rollback, stores, app review e deploy autorizado.


## Incremento 2026-09-23 — sandbox best-effort portátil

O runtime passou a separar claramente portabilidade de isolamento forte. Em Linux, `sandbox.exec` mantém os caminhos best-effort e strict já existentes. Em macOS e Windows, o modo best-effort executa Python/Node pelo processo disponível, aplica timeout/output bounds e encerra o processo no cancelamento, mas retorna explicitamente que a rede não está isolada. O modo strict continua Linux-only e falha fechado fora do executor com cgroup v2 delegado.

Esse avanço remove uma quebra de execução multiplataforma sem transformar compilação cruzada em homologação física. Permanecem como trabalho externo a validação real em Windows/macOS, sandbox forte de host, AppArmor/SELinux, isolamento de dispositivos, assinaturas, installers e distribuição.


## Incremento 2026-09-23 — deploy adapter seguro

O adapter de deploy agora possui coleta de workspace sem root symlink, limites de arquivos/tamanho, redirects desativados e verificação do IP conectado após DNS. Esse caminho está pronto para smoke local determinístico e permanece separado de publicação externa.

O próximo gate externo continua exigindo credenciais do operador, contas de provedor, domínio ou projeto, aprovação, custos, smoke reversível, logs, rollback e confirmação de estado. Nenhum desses requisitos é considerado concluído pelo adapter local.


## Incremento 2026-09-23 — mídia confinada e limites explícitos

Inputs de áudio/imagem agora ficam confinados ao workspace, com rejeição de symlink/traversal, arquivo não regular e áudio acima de 100 MiB. Os outputs também validam o root antes de gravar. O próximo gate para mídia real continua sendo homologação por provider/modelo, credenciais, quota, GPU ou serviço externo, qualidade, moderação e recuperação.


## Incremento 2026-09-23 — Growth OS sandbox-only

O Company Growth OS agora identifica explicitamente campaigns, afiliados e orders como sandbox local. O relatório informa essa condição e qualquer modo externo é fail-closed. O próximo estágio exige adapters específicos por rede, marketplace, gateway, fiscal/NF-e ou fornecedor, com OAuth/API, webhooks assinados, idempotência, reconciliação, approval, limites de orçamento, rollback e homologação.


## Incremento 2026-09-23 — R01–R04 e captura observável

A reauditoria direcionada foi reproduzida no pacote real e corrigida em dois commits publicados. O próximo baseline inclui outputs de mídia/OCR ancorados em `os.Root`, leitura de visão limitada e cancelável, pacote de deploy público filtrado antes da leitura e egress que valida DNS antes de TCP. Os testes focados e suas variantes race cobrem os controles positivos e negativos; compilação Darwin/Windows foi verificada sem alegar teste físico.

O capturador de paridade agora produz screenshots somente após estado observável e interação segura, com manifesto de SHA/build/viewport e política explícita para falhas esperadas do bridge local. A captura local não é prova de provider externo, conta, deploy, device ou release.

Próxima frente independente: abrir e verificar o PR documental da Home a partir da main. O trabalho de produto continua aberto para CI do novo head, jornada Tel-Agent, integração externa homologada e demais requisitos V5. Dependências de conta, consentimento, hardware, certificado, loja, app review ou custo devem permanecer `BLOCKED_BY_EXTERNAL_DEPENDENCY` até existir autorização e evidência específica.


## Incremento 2026-09-23 — Tel-Agent textual acima do canvas

A prioridade Tel-Agent agora possui uma primeira jornada executável no Company OS: canal textual persistente, histórico por tenant, retorno observável e três operações allowlisted (`report.read`, `backlog.create`, `campaign.draft`). O caminho não cria um motor concorrente ao Ollama, não publica ações externas e mantém approvals e modo sandbox.

Próximos gates separados: adapter de telefonia/SIP/SMS ou WhatsApp, identidade da conta operadora, consentimento de escopos, gravação/retensão aprovada, testes reversíveis de destino e homologação externa. A ausência desses recursos não bloqueia o canal textual local, mas impede classificar Tel-Agent como telefonia completa ou produto final.


## Incremento 2026-09-23 — Company cycles transacionais

O Company OS agora trata a criação de ciclo e schedule com idempotência e compensação local, incluindo replay após restart e prevenção de agendas duplicadas. O próximo gate ainda aberto é a execução distribuída real com worker persistente, lease/claim em PostgreSQL/Redis e observabilidade de falhas; a implementação local não é apresentada como homologação desse ambiente.
