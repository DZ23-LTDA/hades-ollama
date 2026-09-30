# Auditoria final de hardening — Ollama Classe A+

**Data:** 2026-09-22
**Escopo:** runtime agentic, shell desktop, Company OS/Growth OS, MCP/connectors, autenticação, mobile e gates de release.

## Conclusão

A rodada corrigiu uma camada adicional de riscos reproduzíveis em connectors, MCP, lifecycle administrativo e Browser Operator, mas a auditoria independente de 2026-09-22 encontrou **P0s internos ainda abertos** em isolamento multi-tenant, approvals, sandbox/process isolation, SSRF/DNS rebinding, DLP, Builder e release. O repositório possui uma base local-first funcional, com contratos, persistência, UI, smoke tests e guardrails parciais. O resultado correto é **preview/local RC em hardening**, não produto final nem produção-ready.

## Achados resolvidos

| Área | Correção verificada | Evidência |
|---|---|---|
| Autenticação | Bind não-loopback exige auth por padrão; `auth=false` não desliga essa proteção | `server/agent_auth_policy_test.go` |
| MCP stdio | Allowlist vazia é rejeitada e resposta com ID divergente falha | `internal/agent/mcp.go`, `internal/agent/mcp_test.go` |
| Remote MCP | HTTPS fora de loopback, allowlist, timeout, bloqueio de IP privado, redirect sem troca de origem e correlation ID | `internal/agent/mcp_remote.go`, `internal/agent/mcp_remote_test.go` |
| Configuração | Manifests MCP usam `DisallowUnknownFields` e rejeitam JSON residual | `server/agent_config_loader_test.go` |
| Filesystem | Componentes symlink e escapes de workspace são rejeitados; StepID é validado antes do sandbox | `internal/agent/tools.go`, `internal/agent/runtime_test.go` |
| Approvals | Approval registra tenant, policy, nonce, actor e expiração; decisão exige motivo e organização compatível | `internal/agent/runtime.go`, `internal/agent/runtime_test.go` |
| Growth OS | Mutações validam empresa/tenant; SKU é único; pedido suporta `Idempotency-Key`; fulfillment repetido é idempotente | `server/company_growth_routes.go`, `internal/agent/company_growth.go`, `company_growth_test.go` |
| Skills | Manifesto não pode marcar a própria skill como confiável sem atestado assinado futuro | `internal/agent/context.go` |
| UI | Settings e Control Center normalizam respostas nulas e preservam estados offline | 20 arquivos Vitest, 199 testes |
| Mobile | Outbox não persiste bearer; usa idempotency key, backoff, limite de tentativas e conflitos permanentes | `apps/mobile-agentic/App.tsx`, `npm run typecheck` |
| CLI/runtime | Teste antigo que rejeitava `ollama agent` foi corrigido; flags legados continuam rejeitados | `cmd/cmd_test.go` |
| Vet | Runtime não copia `sync.Mutex`; contexts de scheduler têm cancelamento em todos os retornos | `internal/agent/runtime.go`, `server/routes.go`, `go vet ./...` |

## Gates executados

Os seguintes comandos passaram no ambiente local:

```text
scripts/check-class-a-plus-integrity.sh
CGO_ENABLED=1 go test ./... -count=1
CGO_ENABLED=1 go vet ./...
CGO_ENABLED=0 go test ./internal/agent -count=1
CGO_ENABLED=1 go test ./server ./cmd/launch ./internal/multillm -count=1
CGO_ENABLED=1 go build -o /tmp/ollama-classe-a-plus-hardening .
app/ui/app: npm test -- --run       # 20 arquivos, 199 testes
app/ui/app: npm run build
apps/mobile-agentic: npm run typecheck
```

`CGO_ENABLED=0 go test ./...` não é um gate global aplicável a este fork porque os pacotes upstream de SQLite/MLX exigem CGO. O gate sem CGO continua válido para `internal/agent`, que é o núcleo independente de runtime agentic.

## Riscos residuais internos

A proteção de filesystem é uma contenção robusta contra symlinks presentes no momento da validação e deve evoluir para `openat`/handles de diretório por plataforma se o produto for exposto a concorrência hostil no mesmo filesystem. A política de skills ainda precisa de um formato de atestado assinado para promover uma skill a `trusted`; até lá, manifests carregados permanecem em revisão. Os testes de distributed RLS, Redis, OTLP, SAML/OIDC contra IdP real e companheiros físicos não podem ser simulados por testes locais.

## Blockers externos

Ainda faltam credenciais e ambientes autorizados para Composio, xAI, Desktop Commander Remote, Meta/Instagram, YouTube, WhatsApp, TikTok Shop, Shopify, CRM, anúncios e logística. Também faltam sandbox regional e app review de marketplaces; PostgreSQL/RLS, Redis e OTLP distribuídos; runners Windows/macOS/Linux; GPU e modelos multimídia locais; assinatura de instaladores; distribuição Android/iOS; e deploy real em contas do operador. Nenhum desses itens foi declarado conectado ou validado neste commit.

## Decisão de release

**Classificação:** `FIXING` / preview-local em hardening. A branch ainda não satisfaz os gates de conclusão porque existem P0 internos reproduzíveis, além dos blockers externos listados abaixo.
**Não classificar como produção universal:** os blockers internos e externos permanecem.
**Próxima ação segura:** publicar somente a camada verificada de hardening na PR pública, depois corrigir os P0 em slices verticais com testes negativos de tenant, approval, egress, DLP e Builder antes de qualquer classificação de conclusão.


## Achados da auditoria original que permanecem abertos

A correção desta rodada não encerra todos os 15 achados críticos originais. Permanecem abertos, por motivos técnicos ou externos, a prova de isolamento forte do sandbox com seccomp/cgroups/quotas; o ciclo completo Streamable HTTP/OAuth do MCP remoto, com sessão, refresh, resumption e pairing; uma capability policy assinada por skill com atestado de origem; auditoria e enforcement completos de egress para todos os connectors; storage seguro/bridge para tokens do desktop renderer; RLS e locking distribuído de produção; SBOM, provenance e assinatura verificável de artefatos; e smoke E2E com providers e dispositivos reais.

O **Growth OS** continua sendo um sandbox local reversível. Ele não é um agente comercial autorizado a publicar, comprar, cobrar, contratar, anunciar ou enviar pedidos. O **Desktop Commander** continua sendo um adapter com handshake e testes locais; nenhuma conta, máquina pareada ou credencial remota foi inventada. A matriz pública e o roadmap continuam sendo a fonte de verdade para esses gates posteriores.


## Incremento posterior à revisão

Depois da revisão inicial, foi adicionada uma policy de capabilities por missão. O default concede apenas escopos de workspace, enquanto browser, desktop, terminal, sandbox, MCP, connectors e deploy exigem escopos explícitos. Também foi removido o bypass genérico por sufixo `/connect`; somente o FullPath exato do handshake de dispositivo recebe tratamento especial, e o próprio handshake ainda exige transporte seguro, token de dispositivo e validações de sessão.

Os gates completos foram repetidos após esse incremento: suíte Go com CGO, `go vet`, build, Vitest, build UI, smokes funcionais e captura Chromium permaneceram verdes.


## Incremento operacional posterior — 2026-09-22

A rodada seguinte transformou adapters em superfícies operacionais locais. Grok Live ganhou cliente Responses com streaming, retry e circuito; Evaluation OS e provider router ganharam testes determinísticos; Company OS ganhou agentes departamentais e Social OS sandbox; plugins, MCP e skills ganharam lifecycle server-side; e o deploy Builder passou a checar approval antes de provider ausente.

As evidências adicionais foram `scripts/smoke-company-growth.sh`, `scripts/smoke-builder.sh`, `app/ui/app/scripts/smoke-shell.mjs`, captura Chromium das dez rotas do shell, `CGO_ENABLED=1 go test ./... -count=1`, `CGO_ENABLED=1 go vet ./...`, build Go, build UI, 20 arquivos/199 testes Vitest, typecheck mobile e integrity guard. O gate distribuído PostgreSQL/Redis/OTLP ficou `N/A` neste sandbox porque Docker não está instalado; não foi convertido em sucesso.

A classificação permanece `CANDIDATE_COMPLETED` para o release preview local. Os blockers externos continuam: contas e quotas xAI/Composio/Desktop Commander, OAuth e app review de redes sociais/marketplaces, staging distribuído, IdP, GPU, dispositivos físicos, assinatura, lojas e deploy do operador.


## Correções posteriores — 2026-09-22

A iteração final corrigiu riscos internos adicionais: `ConnectorManager.Call` não permite mais chamada sem organização; o caminho de operação usa matching por segmento; o dialer rejeita redes privadas resolvidas por DNS, liberando somente loopback explícito; Remote MCP e MCP stdio validam nomes de ambiente e Remote MCP rejeita headers de transporte; lifecycle global de plugins exige owner/admin quando autenticado; e o Browser Operator possui fallback de Chromium para caminhos configurados ausentes.

Provas: testes focados de connectors/MCP/server passaram; integrity guard passou; `CGO_ENABLED=1 go test ./... -count=1`, `CGO_ENABLED=1 go vet ./...`, build Go, Vitest 20/199, build UI, typecheck mobile e Browser Operator com caminho inválido passaram localmente. O CI remoto desta nova revisão ainda depende do workflow disparado após o commit.

## Estado após a auditoria independente ampliada — 2026-09-22

A auditoria ampliada foi considerada como evidência de risco, não como conclusão. Permanecem P0 internos que impedem declarar o produto final: isolamento por organização ainda não é demonstrado de forma uniforme para orchestration, traces, devices/pairing, Builder, plugins/MCP/skills e todas as operações CRUD; approvals ainda precisam de separação forte entre `approve` e `execute`, CAS/nonce e política de aprovador; o sandbox stdio/local continua sendo contenção best-effort sem seccomp/cgroups/rlimits/PID limits comprovados; Remote MCP/connectors/media ainda precisam de verificação do IP efetivamente conectado, redirect-chain e DLP uniforme; Company/Growth ainda aceitam invariantes server-managed e gasto/estado privilegiado que exigem correção atômica; Builder/artifacts precisam de ownership, entry validation e proteção contra symlink/XSS; release/CI ainda não prova o pacote de supply chain completo.

O próximo incremento deve começar por testes negativos cross-tenant e por DTOs allowlisted/ownership no servidor. Enquanto esses testes não passarem, `CANDIDATE_COMPLETED` não é uma classificação válida para o conjunto do produto. O Growth OS permanece sandbox local reversível e adapters externos permanecem não conectados sem credenciais, conta, dispositivo, sandbox ou evidência autorizada.


## Slice P0 validado — Builder e Company — 2026-09-22

O primeiro slice pós-auditoria ampliada adicionou `OrganizationID` aos projetos Builder e aplicou ownership server-side a todas as operações de projeto expostas pelos handlers, incluindo deploy e preview de arquivo. Também adicionou entry validation, recusa de symlink no preview/export e escaping de nomes em templates. No Company OS, o create usa DTO allowlisted e reseta campos server-managed; gasto de agente pausado ou acima do budget falha sem mutação do valor gasto.

A prova inclui `server/builder_scope_test.go`, que cria um projeto em `org-a` e demonstra listagem vazia e `403` sem mutação em `org-b`, além dos testes de entry/XSS/symlink e Company create/budget. Os gates completos do slice passaram. Isso encerra somente esta fronteira; orchestration, traces, devices/pairing, approvals fortes, sandbox/MCP isolation, egress/DLP e release supply chain continuam abertos.


## Slice P0 validada — orchestration, traces e devices — 2026-09-22

A segunda slice pós-auditoria adicionou `organization_id` aos orchestration jobs e aos spans de missão/ferramenta; os handlers de plan/get/run/cancel e traces globais passaram a filtrar pelo tenant. Devices e pairing aplicam ownership em listagem, heartbeat e revoke, e pairing codes não aceitam override de organização. O teste HTTP `server/p0_scope_test.go` cobre duas organizações e verifica `403` sem mutação para orchestration, traces e devices; testes de domínio cobrem pairing cross-tenant.

Os gates completos desta slice passaram. Isso não encerra os P0 da auditoria ampliada: plugins/MCP/skills/artifacts, approvals fortes, sandbox/process isolation, egress/DLP, Company Growth pausado, renderer session e release supply chain continuam abertos.


## Slice P0 validada — mission approvals — 2026-09-22

Approvals de missão agora separam a ação de aprovar do executor: em auth mode, owner/admin é obrigatório; a decisão valida organização, razão, nonce de uso único e versão corrente da missão. O Agentic Console envia o nonce retornado pela API. Testes cobrem role policy, nonce errado, replay e CAS; os gates completos passaram.

A correção não cobre ainda os booleans `approved` de Company/Growth/Social, que continuam uma lacuna explícita para decisão auditável, actor/policy/nonce/expiração e proteção contra auto-approval. Sandbox forte, egress/DLP e outros P0 também permanecem abertos.


## Slice P0 validada — CompanyApproval ledger — 2026-09-22

Campanhas, programas de afiliados, pedidos e drafts sociais passaram a usar approvals server-side com `resource_type/resource_id`, organization, policy, nonce, expiração, actor, razão e CAS de Company version. Os endpoints de aprovação não confiam mais no booleano ou no estado enviado pelo cliente; o campo `approved` é apenas uma projeção após decisão válida. A UI passa a enviar o nonce da decisão pendente.

A rota de gasto ainda possui `approved` booleano e não foi incluída nesta slice; orçamento, anúncios, contratos e mensagens externas exigem a continuação do approval ledger. A auditoria geral continua aberta.


## Slice P0 validada — spend approval e budget atômico — 2026-09-22

A rota HTTP de gasto não aceita mais `approved` como autoridade. Solicitações sujeitas à política criam uma decisão `spend` pendente e retornam `202` sem mutar `SpentCents`; a fila Company OS usa o endpoint genérico de decisão com owner/admin, nonce e CAS. A aprovação aplicada verifica pausa e limite antes do débito, e o teste HTTP/domínio cobre ausência de mutação antes da decisão e débito único depois dela.

O método interno legado com booleano continua somente para compatibilidade de domínio/testes e não é utilizado pela rota pública. A auditoria maior continua aberta para sandbox/MCP, egress/DLP, plugins e demais superfícies.


## Slice P0 validada — DLP de resultados e observabilidade — 2026-09-22

A redação recursiva agora é aplicada antes de persistir/emitir `Step.Result`, erros, eventos e atributos de traces, incluindo JSONStore e PostgresStore. Testes de token injection confirmam que credenciais em mapas, listas, eventos, traces e missões reabertas não aparecem em claro. A política de payload egress de connectors/MCP permanece aberta para não confundir redaction de dados com remoção indevida de credenciais operacionais autorizadas.


## Slice P0 validada — Remote MCP egress — 2026-09-22

O transporte Remote MCP desabilita proxy ambiental, limita redirects ao mesmo origin e confere o IP do socket efetivamente conectado após o dial. A regressão local confirma rejeição de endereço privado conectado e redirect same-origin permitido. Isso reduz DNS rebinding nessa superfície, mas não fecha Media/Connectors, TLS distribuído ou egress global.


## Slice P0 validada — MCP stdio lifecycle — 2026-09-22

MCP stdio agora aplica allowlist de executável absoluto não-symlink, cwd privado, ambiente mínimo, limites de payload e stderr, redaction e encerramento seguro de grupo com restart após cancelamento. Os testes negativos cobrem symlink, comando relativo, cleanup, timeout e payload excedente. A auditoria não considera isso sandbox forte: seccomp, cgroups, rlimits, PID/memória/CPU e validação em plataformas reais permanecem abertos.


## Slice P0 validada — Media egress/download — 2026-09-22

Downloads remotos de mídia passaram a bloquear redirects, proxy ambiental e destinos privados efetivamente conectados; respostas são bounded e só viram artefato após MIME/magic validation. Regressões cobrem redirect, MIME incompatível, magic inválido, overflow e IP privado. A paridade de egress para upload/Connectors, assim como testes distribuídos de DNS/TLS, ainda permanece aberta.


## Slice P0 validada — Connector egress parity — 2026-09-22

Connectors agora removem proxy ambiental, bloqueiam redirects, verificam o IP efetivamente conectado e limitam request/response. Testes negativos reproduzem redirect e payloads oversized em TLS local. A classificação de dados do usuário versus credenciais operacionais, uploads e validação distribuída permanecem abertas.


## Slice P0 validada — OAuth redirect URI allowlist — 2026-09-22

OAuth start/callback não aceitam mais redirect arbitrário: cada provider precisa de URI allowlisted e a forma canônica é usada no PKCE state. HTTPS é obrigatório, com loopback HTTP somente em opt-in explícito. A auditoria de egress dos endpoints OAuth e validação real contra um IdP/staging continuam abertas.


## Slice P0 validada — session handling web/mobile — 2026-09-22

O bearer web não é mais lido de localStorage: a sessão fica em memória e é limpa em 401/403. O mobile limpa SecureStore em sessão expirada, aplica nonce/busy/accessibility em approvals e pede confirmação antes de remover cache/outbox no logout. A jornada de login web e a validação física de secure storage nativo ainda não estão fechadas.


## Slice P0 validada — OAuth endpoint egress — 2026-09-22

OIDC discovery/JWKS/userinfo/token exchange agora usam egress sem proxy, redirects bloqueados e verificação de IP conectado; endpoints retornados por discovery são validados antes do uso. A prova contra IdP real, refresh/revogação e operação SSO de produção permanecem externas ao sandbox.


## Slice P0 validada — tool process containment — 2026-09-22

Terminal/sandbox tools ganharam grupo de processo encerrável, timeout, output bound, stderr redaction, ambiente mínimo e `ulimit` best-effort no sandbox. O resultado é explicitamente `best-effort`; seccomp/cgroups, quotas fortes e testes físicos por plataforma continuam abertos, portanto não equivale a sandbox forte.


## Slice P0 validada — plugin/MCP/skill ownership — 2026-09-22

Lifecycle autenticado de Connector, MCP stdio, Remote MCP e Skill agora exige ownership exato; catálogos preservam recursos sem owner como globais read-only. Testes negativos cobrem duas organizações e ausência de mutação cross-tenant. Registro tenant-owned server-side, attestation e isolamento forte de processos ainda permanecem abertos.


## Slice P0 validada — CI/release quality gates — 2026-09-22

Quality CI corrigiu indentação do Browser Operator e adicionou Go full/vet/build, UI Vitest/build e mobile typecheck. Release ganhou job de quality obrigatório para builds e publicação. Ações GitHub, Docker distribuído, runners físicos, signing e attestation ainda precisam de execução real; a alteração não os finge concluídos.


## Slice P1 validada — Grok/provider contract — 2026-09-22

Modelos Grok agora são allowlisted e o health probe valida catálogo. O endpoint HTTP rejeita stream antes do upstream com `501`, mantendo a capacidade interna de streaming separada de uma promessa pública. Credenciais xAI e streaming real continuam não comprovados externamente.


## Slice P0 validada — Compose/infrastructure defaults — 2026-09-22

Compose agora usa loopback-only, passwords de ambiente e Redis autenticado; CI injeta secrets efêmeros. A composição foi validada estaticamente/integrity, mas Docker/Compose e os serviços distribuídos não executaram nesta sandbox.


## Slice P1 validada — provider selection contract — 2026-09-22

A UI transporta provider e o runtime rejeita seleções sem adapter implementado. Claude/Codex/OmniRoute não são apresentados como conectados; execução externa, credenciais por tenant e streaming ainda permanecem pendentes.


## Slice P1 validada — release artifact integrity — 2026-09-22

O release bloqueia artefatos vazios/ausentes e só ativa provenance attestation com flag e permissões explícitas. Signing, SBOM final e publicação real permanecem não comprovados nesta sandbox.


## Slice P0 validada — jobs/replay tenant scope — 2026-09-22

Queue list/replay usa ownership da missão e o teste HTTP confirma `403` sem mutação cross-tenant. Redis distribuído e restart recovery ainda não foram executados nesta sandbox.


## Slice P0 validada — artifact manifest path safety — 2026-09-22

`BuildArtifactManifest` rejeita symlink e resolução externa antes da leitura. A correção foi coberta por regressões de arquivo regular e symlink.


## Remediação verificada do workflow distribuído — 2026-09-22

O primeiro run do workflow `dz23-agentic-quality` no commit `2933f6e2` forneceu uma evidência nova: o teste PostgreSQL RLS falhou porque a URL do teste usava a role bootstrap superusuária do Compose, enquanto o adapter tenant-scoped exige uma role não-superusuária. Redis DLQ, OTLP e o job web/mobile passaram nesse run. Também foi identificado que passwords efêmeras escritas em `GITHUB_ENV` apareciam no bloco de ambiente do log do passo; não eram credenciais de terceiros, mas a prática foi removida.

A correção agora cria/ajusta uma role `ollama_agent_test` dedicada, roda o teste com essa role e conserva as passwords somente em variáveis locais no mesmo passo. O integrity guard bloqueia o retorno de `GITHUB_ENV` no job distribuído. Os gates locais completos passaram, incluindo integrity, YAML, Go test/vet/build, Vitest/build e typecheck mobile. A confirmação de Docker/PostgreSQL/Redis/OTLP depende da nova execução remota.

Este achado não fecha os P0 restantes: sandbox forte, capability enforcement, egress/DLP completo, storage/IdP distribuído, adapters externos, dispositivos e supply chain continuam abertos. A decisão permanece **FIXING / preview-local em hardening**.


## Follow-up do workflow distribuído — 2026-09-22

O run `35737050056` manteve Go, SBOM e Web/Mobile verdes, mas falhou no bootstrap PostgreSQL. A causa foi objetiva: `:'app_password'` foi colocado dentro de `DO $$`, onde o servidor recebeu sintaxe inválida; adicionalmente, a etapa `down -v` não herda variáveis shell do passo anterior. O workflow foi corrigido para usar `psql -c` condicional com password gerada em hexadecimal e para executar cleanup com placeholders neutros, sem depender de secrets entre steps.

A nova execução remota é obrigatória antes de marcar o gate distribuído como verde. O produto continua em **FIXING / preview-local em hardening**.


## Gate distribuído confirmado — 2026-09-22

Após duas correções incrementais, o run GitHub Actions `35737772235` no commit `15e8ae60` passou integralmente. PostgreSQL RLS, Redis DLQ e OTLP foram executados no runner real; Go/server, Browser Operator, SBOM e Web/Mobile também passaram. O workflow não depende mais de variáveis shell entre steps para cleanup e o teste usa role tenant-scoped não-superusuária.

O gate distribuído está fechado para esta slice. Isso não elimina os demais achados P0/P1 da auditoria nem autoriza declarar produção-ready.


## Slice P1 validada — central CapabilityPolicy — 2026-09-22

A policy de capabilities foi centralizada no Runtime. Tools precisam declarar scopes conhecidos; grants desconhecidos ou descritores sem scopes falham; a verificação ocorre na inicialização, durante o planejamento e imediatamente antes da execução. O approval registra scopes e risco, enquanto o default de missão é somente leitura e a UI torna escrita um opt-in explícito.

A regressão foi coberta por testes de grant desconhecido, descriptor sem scopes, grant parcial, policy determinística, policy de approval e execução completa do runtime. Esta correção reduz overgrant, mas não converte o sandbox best-effort em isolamento forte e não fecha os blockers externos ou os demais P0/P1.


## Remediação verificada — upstream Go race — 2026-09-22

A falha upstream no head anterior foi reproduzida localmente. O `go test -race ./...` apontou o contador não atômico de `TestResearchEngineFetchesSourcesWithCitationsAndCache` e o uso concorrente do `bytes.Buffer` promovido por `mcpStderrBuffer.ReadFrom`. O contador foi tornado atômico e o buffer MCP recebeu sincronização explícita e `ReadFrom` próprio.

Após a correção, os gates locais de teste normal, race, vet, build e integrity passaram. A confirmação remota do workflow upstream continua necessária; portanto o PR não é tratado como globalmente verde até o novo run terminar.


## Quality workflow remoto confirmado — 2026-09-22

O run `35749351291` no SHA `428a99bd` passou Go/server, PostgreSQL RLS + Redis DLQ + OTLP, Web/Mobile e SBOM. O resultado é válido para o workflow agentic do fork. A matriz upstream `test` ainda deve ser observada no head corrente antes de classificar o PR como totalmente validado.


## Bootstrap MCP/Remote MCP testado — 2026-09-22

Os loaders de configuração do servidor agora possuem testes positivos e negativos no caminho real de bootstrap. O resultado fecha a lacuna de cobertura do loader, mas não eleva a integração a `connected` ou `upstream-ready`; endpoint, credencial, sessão OAuth, revogação e transporte externo continuam dependências do ambiente.


## Correlação MCP com notificações — 2026-09-22

A regressão do transporte stdio demonstra que uma notificação JSON-RPC sem `id` não é confundida com a resposta do request. O sistema continua falhando fechado para ID divergente. A correção não encerra os achados de multiplexação, auditoria, sessão Remote MCP ou sandbox forte.


## Remediação do Browser Operator upstream — 2026-09-22

A auditoria do run `35750983274` encontrou `ModuleNotFoundError` de Playwright no teste Browser Operator, enquanto o workflow agentic do fork passou. A correção adiciona Playwright `1.63.0`/Chromium aos jobs upstream, remove o caminho Python Linux-only do launcher Go e torna a matriz nativa Linux/Windows manual e opt-in para não aguardar runners ausentes no PR. A reprodução local passou teste normal/race, mas a confirmação remota e a matriz de hardware continuam pendentes. Nenhum claim de produção-ready é alterado.


## Follow-up verificado — workflow upstream sem fila infinita — 2026-09-22

O run `35755046119` não era um teste único travado: sua matriz nativa aguardava runners customizados indisponíveis, enquanto `test` e `race` Ubuntu falharam por uma versão Playwright inexistente no índice. O patch substitui a dependência por `1.63.0`, corrige a concorrência e torna a matriz nativa manual e opt-in. Isso melhora a determinismo do PR, mas não equivale a executar GPU, Windows, macOS ou dispositivos físicos; a classificação continua preview/local RC em hardening.


## Validação independente — upstream CI multiplataforma — 2026-09-22

A correção de CI convergiu por diagnóstico observável, sem relaxar assertions ou remover jobs. O head `de0e86772e96372789c10d924eb5738f8808821b` passou os quatro workflows relevantes: integrity `35769597628`, agentic quality `35769597363`, multi-provider `35769597578` e upstream `test` `35769597404`. O upstream passou testes em Linux, macOS e Windows, race em Linux e macOS, patches e `go_mod_tidy`; o workflow agentic passou Go/server, PostgreSQL RLS + Redis DLQ + OTLP, Web/Mobile e SBOM.

O achado é **mitigado para o caminho normal de CI**. A matriz GPU/nativa não foi executada e permanece manual/opt-in com runners do operador. O resultado não altera os achados P0/P1 de sandbox forte, autorização distribuída, OAuth revocation, adapters externos, validação física de desktop/mobile, signing/provenance e homologação. A decisão permanece **FIXING / preview-local em hardening**, sem merge automático em `main`.


## Validação independente — hardening pós-CI — 2026-09-22

A revisão desta rodada encontrou e corrigiu cinco lacunas internas verificáveis. O cliente UI agora conserva a sessão em `403` e invalida apenas em `401`; o planner implementa o contrato do tipo `api.ChatResponseFunc`; Company Growth e Social usam um ledger por digest para impedir replay de spend, conversão e métrica; o terminal allowlisted rejeita flags e paths fora do workspace; e o logout agentic revoga o token no `AuthStore` antes da limpeza local. Cada slice recebeu regressões negativas e foi publicada em commits separados/coerentes.

O head `935fb273` passou integrity (`35784211426`), agentic quality (`35784211429`), multi-provider (`35784211481`) e upstream `test` (`35784211483`). O upstream executou testes Linux/macOS/Windows e race Linux/macOS; o workflow agentic executou Go/server, PostgreSQL RLS + Redis DLQ + OTLP, Web/Mobile e SBOM. Isso é mitigação observável do caminho normal, não homologação de GPU, sandbox forte, integrações externas, dispositivos físicos ou release assinado.

A decisão permanece **FIXING / preview-local em hardening**. Não há base para declarar produção-ready ou fazer merge automático em `main`.


## Validação independente — sandbox, CSRF, mobile e release — 2026-09-22

A rodada fechou quatro lacunas internas verificáveis. O sandbox strict Linux agora falha fechado sem cgroup v2 delegado, aplica namespaces/no-new-privs/seccomp e mata o cgroup no timeout; o default best-effort permanece explicitamente classificado. O middleware rejeita mutations com Origin não permitido. O mobile não reutiliza cache/outbox/push entre organizações e não enfileira ação autenticada antes de confirmar `organization_id` via `auth/session`. O release workflow gera SBOM, metadata, checksums e valida o manifesto antes do upload.

Os gates locais de Go agent/server, vet/build/integrity, mobile typecheck e Expo web export passaram. Integrity push `35793674459` e PR `35793679458` passaram no head `a7bc82f5`; agentic quality, multi-provider e upstream test ainda não estavam concluídos no momento desta revisão. `npm audit` mobile reportou 18 vulnerabilidades de produção (11 moderate, 7 high), sem aplicar correção forçada.

O resultado é mitigação observável, não homologação de cgroup/AppArmor/SELinux, IdP/OAuth/providers externos, push remoto, dispositivos físicos, assinatura/provenance efetiva, rollback ou lojas. A decisão continua **FIXING / preview-local em hardening**, sem merge automático em `main`.


## Follow-up independente — audit mobile — 2026-09-22

A triagem inicialmente reportada foi concluída. Como o fix automático exigia Expo 57/React Native 0.87 major, a remediação adotou overrides transitivos mínimos: `image-size@2.0.4`, `postcss@8.5.28` e `uuid@11.1.1`. O lock, typecheck, Expo web export e `npm audit --omit=dev` passaram com zero vulnerabilidades de produção. A decisão é manter a migração major fora desta slice até haver testes físicos Android/iOS.


## Validação independente — CI normal final no head e6e0632b — 2026-09-22

A revisão confirmou o head `e6e0632ba5ff0495ed4b061221c90696478b3d67` nos quatro workflows do PR: upstream `test` `35794443450`, integrity `35794443307`, multi-provider `35794443300` e agentic quality `35794443501`, todos PASS. O conjunto ficou em 21 checks successful, 3 skipped, 0 failing e 0 pending. O upstream executou as plataformas públicas disponíveis e o workflow agentic cobriu Go/server, Browser Operator, serviços distribuídos, Web/Mobile e SBOM.

A troca para `npm ci` torna os gates web/mobile reproduzíveis; o audit de produção mobile permanece zero. Os skips de licença e matriz nativa não contam como aprovação. O resultado mitiga o caminho normal de CI, mas não fecha sandbox de host, IdP/OAuth, providers externos, deploys, devices, signing, stores ou homologação. A decisão continua **FIXING / preview-local em hardening**, sem merge automático em `main`.


## Validação independente — V5 provider/connectors e CI — 2026-09-22

A revisão confirmou cinco melhorias internas verificáveis. O planner deixou de aceitar fallback silencioso e exige provider/modelo executável. O runtime separa dados duráveis do workspace. O Remote MCP fixa o destino aprovado depois de DNS. O catálogo e a UI de connectors expõem somente readiness seguro e o seletor de missão não apresenta providers ausentes como conectados. O integrity guard passou a proteger esse contrato dinâmico.

O upstream falhou primeiro por um helper `OllamaPlanner.fallback` não utilizado; a correção foi removê-lo. A execução seguinte expôs uma colisão `EEXIST/ENOENT` no cache npm compartilhado do runner Windows, com cancelamento dos irmãos por `fail-fast`. O workflow foi corrigido para cache por runner e `fail-fast: false`. O head `0f95b6a1969462fb309e00e29813e8136d15f757` passou os quatro workflows: `test` `35804229207`, integrity `35804229189`, multi-provider `35804229204` e agentic quality `35804229301`.

O resultado é mitigação observável do caminho normal, não homologação externa. Não há base para declarar Composio, OAuth, Woovi/OpenPix, fiscal, marketplaces, deploy, mídia, dispositivos ou signing conectados/concluídos. A decisão permanece **FIXING / preview-local em hardening**, com PR aberto e sem merge automático em `main`.


## Addendum de evidências pós-V5 — 2026-09-22

O head `0da6be80` foi revalidado pela matriz pública. O run upstream `35808941810` terminou `SUCCESS`; normal macOS `107015911723`, race macOS `107015880499`, normal Ubuntu `107015911678`, normal Windows `107015911689` e race Ubuntu `107015880534` terminaram com sucesso. Os workflows do fork correspondentes a integrity `35808941807`, agentic quality `35808941824` e multi-provider `35808941746` também passaram. A falha macOS anterior foi diagnosticada como uma diferença de alias de filesystem (`/var` e `/private/var`) e corrigida sem remover a rejeição de symlinks descendentes.

O commit `411335ba` fechou uma parte antes aberta do registro tenant-owned de connectors. O runtime agora usa manifest durável em `OLLAMA_AGENT_STORE/connectors.json` quando não há manifest estático explícito. Escritas usam modo `0600`, arquivo temporário, sync, rename e rollback. `POST /api/agent/v1/connectors` exige owner/admin em auth mode, força o tenant server-side, recusa campos desconhecidos e valores de segredo e bloqueia colisões de ID cross-tenant. O cliente web possui formulário para endpoint HTTPS, operações, nome de env e ID OAuth, mas nenhum campo de token.

A slice foi coberta por testes de persistência, redaction, rollback, admin/member, cross-tenant, raw-secret rejection e build UI. O novo SHA ainda exige a conclusão dos checks remotos próprios; a evidência do head anterior não é reutilizada como se cobrisse o novo código. Persistência/lifecycle de MCP, Remote MCP e skills, attestation de skills, isolamento de processo e auditoria distribuída continuam fora desta slice.

O veredito não muda: **preview/local RC em hardening**, não final e não production-ready. Cadastro de connector não equivale a OAuth consentido, conta conectada, upstream saudável ou ação externa validada.


## Addendum de lifecycle MCP/Remote MCP/skills — 2026-09-22

O commit `96fd7edd3440876ed67ae7051cc475ca953d2847` estendeu o lifecycle durável além de connectors. Managers padrão persistem MCP stdio em `mcp.json`, Remote MCP em `remote-mcp.json` e skills em `context/skills/*.json` abaixo do DataRoot. Os writers são atômicos, os arquivos são privados e reload tests demonstram round-trip após restart. Configuração por manifest externo continua sendo bootstrap estático explícito.

Os endpoints POST exigem owner/admin e derivam a organização da sessão. A colisão de IDs entre tenants é rejeitada. MCP mantém executável absoluto regular, allowlist de métodos e env names; Remote MCP mantém HTTPS, SSRF/private-address rejection, DNS pinning, same-origin redirects e env-only token/header references; skills rejeitam confiança derivada do cliente e gravam `trusted=false`. A UI recebeu contratos e formulários para os três casos sem campos de segredo.

Foram aprovados testes normais e race de persistence e HTTP, além do runner local completo em Go, vet, build, UI, mobile, integrity, YAML e diff. Os checks remotos do novo SHA ainda precisavam concluir quando esta nota foi criada. O resultado não cobre contas externas, OAuth consentido, upstream smoke, execução remota, sandbox físico ou trust attestation.

O veredito permanece **preview/local RC em hardening**, não final nem production-ready. A existência de uma rota e de um manifest válido não é evidência de conexão real, autorização de terceiro ou capacidade de operar um computador/conta fora do ambiente provisionado.


## Addendum de reauditoria mídia/deploy/captura — 2026-09-23

A reauditoria reproduziu quatro achados no pacote real: output de mídia via `.agent-media` symlink, inclusão de `.env` e `.git/config` no pacote, conexão TCP a endereço privado antes da recusa e leitura integral de imagem acima do limite. O commit `06990ef573129da2e4a272b49435857b9e7f46c4` corrigiu essas causas com `os.Root`, paths relativos, rejeição de symlink, leitura limitada/cancelável, validação antecipada de output, filtro de conteúdo público antes da leitura e resolução/validação de todos os IPs antes do dial. Testes normais/race e vet focados passaram; Darwin/Windows foram compilados com `go test -c`, sem alegar execução física.

O commit `ec7f52c048c238510ca6dc08212cd2c10535bfe1` tornou a captura relativa ao checkout, observável e fail-closed. Contra o bundle bridged ao backend Ollama local, dez rotas foram capturadas com interação segura; o manifesto vincula SHA, build e viewport e registra falhas esperadas de ausência de conta/endpoints base sem mascarar assets, JavaScript, request failures ou Agentic API failures. A captura local não é homologação externa.

Os três agentes mantêm o mesmo veredito: **FIXING / preview-local em hardening**, não final e não production-ready. Permanecem dependências externas para contas/OAuth, providers, social commerce/fiscal, deploy, dispositivos, hardware, assinatura, lojas, app review e homologação de operador.


## Addendum independente — Tel-Agent textual acima do canvas — 2026-09-23

A revisão desta rodada confirmou o primeiro vertical Tel-Agent sem criar um segundo motor: `45393d8c` adiciona `tel-agent.text` ao Company OS, histórico persistente tenant-aware e três operações allowlisted. A leitura de relatório é separada das mutações; owner/admin/operator são exigidos para escrita autenticada; DLP, limite de mensagem e retenção limitada protegem o histórico. A UI expõe o canal antes dos painéis de construção e informa que telefonia não está configurada.

Os testes Go normais/race, Vitest/build e integrity passaram localmente. O primeiro check remoto do SHA ainda estava parcialmente `queued`/`in_progress`, portanto não foi usado como evidência de CI final. Telefonia/SIP/SMS/WhatsApp, contas externas, OAuth e homologação continuam bloqueados por dependências do operador. A decisão permanece **FIXING / preview-local em hardening**, sem merge automático em `main`.


## Addendum independente — Company cycles transacionais — 2026-09-23

A revisão confirmou no commit `46d6b34c` que a criação de ciclo Company não deixa mais um ciclo `enabled` sem schedule quando a persistência falha. `Idempotency-Key` cobre replay após restart e conflito de payload; o handler compensa ciclo e schedule e o `ContextStore` desfaz a entrada em memória em erro de escrita. Testes normais/race, HTTP e o runner local completo passaram. O próximo risco é distribuído: lease/claim de worker, Postgres/Redis reais e execução multi-processo ainda não foram homologados.


## Addendum independente — retry bounded de schedules — 2026-09-23

A revisão confirmou no commit `86597b82` que falhas de criação de missão não são mais descartadas após o claim local. O schedule registra somente código sanitizado, tenta novamente com backoff bounded e desabilita após três falhas; claims cuja escrita falha retornam ao estado anterior. A cobertura normal/race e os gates completos passaram. Isso não encerra o risco de múltiplos workers sem lease distribuído nem substitui homologação de PostgreSQL, Redis e DLQ.


## Addendum independente — rollback transacional do queue local — 2026-09-23

A revisão confirmou no commit `717a7e4f` que uma falha de `jobs.json` não deixa mais claim ou outra transição parcialmente aplicada no mapa em memória. A regressão normal/race e os gates completos passaram. O resultado reduz o risco local de jobs presos, mas não fecha lease distribuído, fencing, múltiplos workers ou recuperação Redis.


## Addendum independente — Ack/Nack condicionados a running — 2026-09-23

A revisão confirmou no commit `9d28cbc2` que jobs fora de `running` não podem mais ser confirmados ou reenviados por Ack/Nack em local e Redis. Os gates Go completos e race passaram. O risco de lease distribuído, fencing e recuperação de worker permanece aberto e requer infraestrutura Redis/Postgres real.


## Addendum independente — compensações do RedisQueue — 2026-09-23

A revisão confirmou no commit `883a17e2` compensações para falhas entre comandos Redis em Enqueue, Claim, retry e Replay. Gates Go completos passaram. O teste distribuído foi compilado/executado sem endpoint, portanto não houve evidência de conexão, lease ou recuperação multi-worker reais. O risco residual exige Redis de homologação e, para atomicidade forte, scripts/transações apropriados.


## Addendum independente — moveDue Redis — 2026-09-23

A revisão confirmou no commit `b1aaebfd` a compensação do caminho `ZREM`/`LPUSH` em `moveDue`, evitando perda best-effort de retries delayed. Testes locais focados passaram; Redis real, lease e fencing permanecem não homologados.


## Auditoria independente da retomada — 2026-09-30

### Escopo e evidências

Auditoria somente leitura do checkout `0d3aa7318367008aec96e21a77408f016f5936c9`, complementada por validações locais da correção de acessibilidade. Nenhum deploy, merge em `main` ou provider externo foi executado.

- Engenharia revisou runtime, Studio, testes e matriz; depois o Go focado foi executado com `/usr/local/go/bin/go`.
- Segurança revisou `SECURITY.md`, ADRs, rotas WhatsApp, auth/tenant, release e egress.
- Produto/QA/UX revisou matriz, árvore, evidências, rotas, E2E e acessibilidade.

### Achados desta rodada

#### HIGH

1. **RESOLVIDO — Webhook WhatsApp autenticado.** O GET delega o verify token ao adapter ativo; o POST valida HMAC-SHA256 do corpo cru em comparação constante, exige app secret, `object=whatsapp` e `phone_number_id` configurado. Assinatura inválida retorna 401 e segredo ausente retorna NOT_CONFIGURED/424.
2. **RESOLVIDO — Outbound WhatsApp fail-closed.** `SendMessage` exige destinatário allowlisted, adapter configurado e aprovação HITL aprovada e vinculada ao destinatário exato; sem qualquer requisito retorna bloqueio e nunca chama o adapter.
3. **RESOLVIDO — Boundary tenant fixo em auth desabilitado.** O middleware agora injeta sempre `LocalOrganizationID` e ignora `X-Ollama-Organization`/contexto externo no modo local; handlers usam o escopo fixo nos stores de missões, Company, connectors e schedules. Tentativas cross-tenant retornam 403/404 sem vazamento ou mutação.
4. **Preview/download do Studio não encaminham Bearer.** Navegação direta do iframe e download via link não carregam o header usado por `agentFetch`; o fluxo autenticado pode retornar 401.
5. **Preview do Studio sem `sandbox`.** A matriz declarava iframe sandboxed, mas o iframe do Studio não tinha o atributo; conteúdo de projeto é servido na mesma origem.

#### MEDIUM / IMPROVEMENT

- Atualizações visuais concorrentes podem perder edições/ordem de undo-redo porque o backend não usa CAS/versão esperada e a UI envia uma requisição por tecla.
- Export antigo continua associado ao projeto depois de edição, sem invalidar checksum/path ou bloquear download até novo export.
- Modais críticos de sidebar/deploy precisam semântica de diálogo, foco inicial/retorno e fechamento por Escape.
- Evidências e matriz não registram uniformemente viewport, commit e checksums; o script Studio tem asserções condicionais e não verifica bytes/checksum do download.
- Plugins possui duas superfícies (`/connectors` e `/plugins`) e a árvore ainda marca Builders como `[ADAPTER]` enquanto a matriz marca `PASS`.

### Correção executada nesta retomada

- `AgenticSplitShell.tsx`: campo e botão do composer receberam nomes acessíveis (`Instrução da missão`, `Enviar instrução`).
- `shell.spec.ts`: adicionada asserção Playwright em viewport `390x844`; expectativas obsoletas da Home foram alinhadas aos headings e selo reais, sem reduzir cobertura.
- Evidência: `npx playwright test e2e/shell.spec.ts` — **2 passed**; `npx vitest run` — **36 arquivos / 255 testes passed**; `go test ./internal/agent ./server -run 'WhatsApp|WA|Gateway' -count=1` — **PASS**.

### Veredicto

`REQUEST_CHANGES`. O produto tem implementação ampla e CI verde no commit auditado, mas **não** deve ser declarado production-ready/enterprise ou paridade final concluída enquanto os HIGHs de webhook/outbound/tenant/Studio autenticado e sandbox não tiverem regressões automatizadas e execução verificável.

### Próximo passo obrigatório

Próximo slice: corrigir Studio (Bearer, sandbox, invalidação de export e concorrência), atualizar matriz/árvore e repetir os três pareceres independentes. Dependências externas continuam `NOT_CONFIGURED`/`BLOCKED_EXTERNAL`.


## CA-1 — Correção dos achados WhatsApp 1 e 2 — 2026-09-30

- **Webhook:** `internal/agent/whatsapp_adapter.go` implementa HMAC-SHA256 sobre o corpo cru, comparação constante, bloqueio sem `app_secret`/`webhook_secret`, validação de `object` e `phone_number_id`; `server/whatsapp_routes.go` delega o handshake GET ao adapter e retorna 401/424 de forma honesta.
- **Outbound:** `internal/agent/whatsapp_gateway.go` exige allowlist e aprovação HITL aprovada, vinculada ao número exato; `RequestOutboundApproval`/`ApproveOutbound` registram o ciclo sem chamar provider antes da aprovação.
- **Regressões:** `TestWhatsAppCloudWebhookHMACAndIdentity`, `TestWhatsAppOutboundRequiresAllowlistAndApproval` e `TestWhatsAppRoutesWebhookRejectsInvalidSignatureAndIdentity`; focused Go e gates completos passaram.
- **Estado:** achados 1 e 2 encerrados. Permanecem abertos somente os achados 3–5 e melhorias listadas acima.


## CA-2 — Isolamento de tenant no modo local — 2026-09-30

- **Correção:** `server/agent_routes.go` fixa `agent.organization` em `agent.LocalOrganizationID` quando `authRequired=false`, marca o contexto como local e ignora headers/contexto de organização enviados pelo cliente.
- **Cobertura:** `TestAuthDisabledAlwaysUsesLocalOrganizationAcrossStores` verifica listagem sem vazamento e escrita forçada ao tenant local para missões, Company, connectors e schedules; também verifica leitura cross-tenant bloqueada.
- **Regressão:** `go test -run 'Tenant|Isolation|CrossTenant|Organization|AuthDisabled|LocalOrg' ./internal/agent ./server` passou.
- **Estado:** achado 3 encerrado para o modo local/dev. O isolamento enterprise PostgreSQL/RLS distribuído continua governado pelos gates próprios e não foi promovido artificialmente.
