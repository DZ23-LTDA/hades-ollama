# Auditoria ponta a ponta — Hades / Ollama Full

**Data:** 2026-10-01
**Branch auditada:** `recovery/ollama-full-snapshot`
**SHA observado pelos auditores:** `ef9c21d2`
**Escopo:** usuário leigo, frontend/design, acessibilidade WCAG 2.2 AA, backend Go, arquitetura, segurança, QA, CI/CD, packaging e release.
**Modo:** somente leitura; nenhum arquivo-fonte foi alterado durante a auditoria.

> **Veredito executivo: BLOCKED para publicação pública e para a afirmação de “100% pronto”.** O projeto tem boas bases — isolamento de tenant em várias rotas, política de egress, approvals com CAS/nonce, importação ZIP endurecida, estados vazios honestos e CI significativo — mas ainda há riscos P0/P1 que precisam de correção e validação real.

## 1. Resumo para o Claude

A auditoria foi executada por seis frentes independentes e uma consolidação técnica. Foram analisados código Go/React, workflows, scripts, documentação e evidências E2E já versionadas.

**O que está bom:**

- O produto é majoritariamente local-first e não finge integrações externas sem credencial.
- Há controles relevantes de tenant, egress, DLP, approvals, importação de ZIP e recuperação de rotas.
- O CI possui gates de Go, frontend, integridade, cross-platform e race; o workflow Windows unsigned foi validado de forma real no SHA anterior.
- O shell visual tem identidade consistente e várias telas mobile funcionam bem.
- A documentação registra limitações, unsigned e dependências externas em diversos pontos.

**O que impede a liberação:**

1. Fronteiras de confiança incompletas para MCP local, secrets globais por tenant e importação GitHub.
2. Possível traversal por ID de schedule e contratos incompletos de download por chunks.
3. Release gate incoerente e incompleto: exige contexto de PR em release e não torna E2E funcional, install/upgrade/rollback, packaging por artefato ou provenance/assinatura uniformes pré-condições.
4. Primeiro uso pode mandar o leigo para instruções upstream, mostrar o executável errado ou falhar sem modelo/servidor com pouca explicação.
5. Barreiras relevantes de teclado, modais, labels, tabs ARIA e responsive mobile.

## 2. Limitações e nível de confiança

Esta auditoria **não** executou build, `go test -race`, carga Redis, servidor, Lighthouse/axe, accessibility tree, leitor de tela, dispositivo físico, credenciais reais ou integração externa. Portanto:

- Achados de código/contrato/documentação são fatos observados.
- Impactos de SSRF, corrupção, data race, retenção de goroutine e cruzamento de segredo são hipóteses técnicas sustentadas pelo caminho de código; devem ser reproduzidos com testes negativos antes de serem tratados como exploração confirmada.
- Contraste e tamanho de alvo precisam de medição renderizada em light/dark.
- Não declarar WCAG 2.2 AA, produção-ready ou 100% de paridade enquanto os gates abaixo não forem executados.

Também foi observada divergência entre evidências versionadas (`Ollama Full`/Studio v17) e preview live (`Hades`/Studio v24). Toda nova evidência deve carregar SHA/build ID.

## 3. Achados prioritários

### P0 — segurança: MCP local fora do sandbox forte

**Área:** sandbox, plugins, MCP
**Evidência:** `internal/agent/mcp.go:93-107` aceita executável absoluto e `:426-443` usa `exec.CommandContext`; `internal/agent/mcp_process_unix.go:10-12` apenas configura `Setpgid`. Não foram observados namespace, chroot, seccomp, `no_new_privs`, cgroup, limite de rede, filesystem mínimo ou UID dedicado neste caminho. O sandbox forte existente está em outro caminho (`sandbox.exec`).

**Impacto:** um MCP malicioso/comprometido pode executar como o usuário do agente e alcançar filesystem, rede e credenciais do host. Em multi-tenant, isso rompe a fronteira de capacidade.

**Correção recomendada:** executar MCP em helper dedicado com isolamento equivalente ao sandbox forte: UID separado, filesystem mínimo e preferencialmente read-only, rede negada por padrão, seccomp/`no_new_privs`, limites de CPU/memória/PIDs/saída e auditoria. Sem isolamento, o estado deve ser `BLOCKED_EXTERNAL`/`NOT_CONFIGURED` em produção multi-tenant.

**Gate de aceite:** teste com plugin comprometido comprova que não alcança host, credenciais ou outro tenant; o modo sem isolamento é recusado.

### P0 — segurança: secrets globais atravessam tenant/destino

**Área:** secrets, auth/tenant, egress/DLP, connectors/MCP
**Evidência:** `internal/agent/connectors.go:176-200` valida apenas o nome de `TokenEnv`, e `:581-596` lê/envia o valor; `internal/agent/mcp_remote.go:602-609,650-655` faz o equivalente para `TokenEnv`/`HeadersEnv`. As rotas permitem registro por organização sem namespace técnico de segredo (`server/plugin_routes.go:68-97,186-207`).

**Impacto:** um administrador de organização pode referenciar variável global com token do operador/outro tenant e enviar o segredo a um endpoint escolhido. A exploração depende das permissões do processo, mas a ausência de binding técnico é observável.

**Correção recomendada:** trocar nomes arbitrários de env vars por IDs de um secret broker com ACL por organização, destino e propósito; negar `TokenEnv`/`HeadersEnv` configurados pelo tenant em modo compartilhado; permitir somente secrets pré-provisionados pelo operador; registrar rotação e uso.

**Gate de aceite:** teste multi-tenant prova que nenhum tenant consegue referenciar ou enviar variável global.

### P0 — segurança: importação GitHub usa token global

**Área:** importação, tenant, credenciais
**Evidência:** `server/agent_routes.go:21-37` liga o token do importer a `OLLAMA_GITHUB_TOKEN` global; `internal/agent/project_import.go:75-93,166-175,251-260` aceita qualquer `github.com/owner/repository` válido e envia o token.

**Impacto:** tenant autenticado pode solicitar importação de repositório privado que o token global consegue ler, sem binding comprovado entre principal, organização, repositório e credencial.

**Correção recomendada:** GitHub App/installation token ou OAuth por organização/repositório; allowlist explícita de owner/repository; negar token global em modo compartilhado; auditar organização, principal e escopo.

**Gate de aceite:** importação privada sem credencial do tenant retorna `NOT_CONFIGURED`; token de outro tenant não funciona; repositório fora da política é rejeitado.

### P0 — release: gate incoerente e cobertura insuficiente

**Área:** release engineering, QA, supply chain
**Evidência:** `.github/workflows/release.yaml:3-19` é `workflow_dispatch` e `:822-845` exige PR gate no `GITHUB_SHA`; `.github/workflows/pr-gate.yaml:16-18` dispara somente em `pull_request`; `docs/RELEASE_READINESS.md:48-57` reconhece que não há PR gate em main/tag. Além disso, `release-readiness.yaml` não exige explicitamente E2E funcional, `test-install`, package smoke, installer smoke e upgrade/rollback no mesmo SHA. `dz23-e2e.yaml` cobre apenas shell offline Chromium.

**Impacto:** release pode bloquear por contexto inexistente; se o gate for contornado, um SHA pode passar sem testar o produto real, instalação/upgrade/rollback, cada artefato publicado ou cada plataforma.

**Correção recomendada:** criar uma matriz canônica de contexts por SHA candidato; usar workflow reutilizável para checks de release; exigir E2E funcional, package/install/upgrade/rollback e installer smoke por OS/arquitetura; fixar actions e downloads; exigir SBOM/provenance/assinatura por artefato.

**Gate de aceite:** um SHA candidato produz e executa os checks necessários no mesmo commit, e a publicação falha quando qualquer artefato não é homologado.

### P0 — persistência: ID de schedule permite traversal

**Área:** AppSec e persistência
**Evidência:** `server/agent_routes.go:1360-1371` decodifica o schedule e sobrescreve apenas `OrganizationID`; `internal/agent/context.go:669-696` aceita `schedule.ID` e usa `filepath.Join(s.root, "schedules", schedule.ID+".json")` sem allowlist nem verificação final de contenção.

**Impacto:** hipótese de sobrescrita/corrupção de arquivo de outro namespace se o ID apontar para `../` ou separadores de plataforma.

**Correção recomendada:** gerar ID no servidor ou aplicar `validSnapshotID`; rejeitar `../`, separadores, IDs absolutos e caracteres fora da allowlist; verificar contenção da rota final.

**Gate de aceite:** testes com traversal Unix, Windows e symlinks comprovam que nenhum arquivo fora de `root/schedules` é criado, sobrescrito ou removido.

### P1 — integridade: downloader não valida contrato HTTP por chunk

**Área:** download, blobs, persistência
**Evidência:** `server/download.go:339-375` envia `Range`, chama `client.Do` e copia `resp.Body` com `io.CopyN` sem validar `StatusCode`, `Content-Range` ou `Content-Length`; `server/images.go:1077-1093` só verifica digest em etapa posterior.

**Correção:** exigir `206` para ranges, validar `Content-Range`/comprimento, rejeitar status não-2xx antes da cópia e só publicar cache após digest atômico.

**Gate:** respostas 200 ignorando Range, 206 incorreto, 404 com corpo grande e 416 nunca produzem blob válido.

### P1 — concorrência: streaming/cancelamento pode reter trabalho

**Área:** HTTP, workers, cancelamento
**Evidência:** `server/routes.go:1171-1187,1221-1243` envia resultados por canal sem selecionar `Request.Context().Done()`; geração em `:692-792` segue padrão semelhante. `server/agent_routes.go:1987-1992,2015-2019` usa `context.Background()`. `internal/agent/swarm.go:366-386` não cancela jobs em execução, e retries em `server/download.go:303-307`/`upload.go:163-166,199-207,262-273` usam `time.Sleep` sem contexto.

**Correção:** centralizar `sendContext`, ligar produtor ao ciclo de vida da requisição, guardar `CancelFunc`, usar timers canceláveis e medir cancelamentos.

**Gate:** disconnect durante pull/push/geração, cancelamento concorrente e shutdown terminam job, goroutine, conexão e I/O; executar `-race`/goleak.

### P1 — approvals: self-approval de missão não é bloqueado

**Área:** HITL, IAM, compliance
**Evidência:** o approval contém missão, step, organização, policy, hash, nonce e expiração, mas não requester (`internal/agent/runtime.go:951-960`). A decisão valida actor (`:1837-1894`) e a rota exige owner/admin (`server/agent_routes.go:2862-2873`), porém não compara actor com solicitante.

**Correção:** persistir requester autenticado e exigir actor distinto para write/external side effect/destructive; aplicar role separation/quorum quando a política exigir.

**Gate:** self-approval, cross-tenant approval, replay, restart e concorrência retornam bloqueio sem mutação.

### P1 — primeiro uso pode direcionar ao upstream ou falhar silenciosamente

**Área:** onboarding, documentação, modelos, offline
**Evidência:** `docs/quickstart.mdx`, `docs/linux.mdx` e `docs/windows.mdx` mantêm instruções para `ollama.com`; `app/ui/app/src/components/onboardingUtils.ts:1-4` fixa `FIRST_MODEL_COMMAND` como `ollama`, enquanto README/scripts usam `bin/ollama-full` e `CLASS_A_PLUS_GUIDE.md` usa outro nome. Sem modelo, `ModelPicker.tsx:337-340` não oferece download; `useChats.ts:302-304` lança `No model selected`, mas o erro é apenas logado. Offline, `ChatForm.tsx:776-782` usa overlay sem diagnóstico.

**Correção:** uma sequência fork-specific única, comando canônico, banner upstream-versus-Hades, estado Baixar/Usar, CTA de modelo, erro inline com retry que preserva rascunho e banner offline com endereço/Verificar novamente.

**Gate:** checkout limpo leva ao Hades; comando existe; `/api/tags=[]` produz CTA de download; envio bloqueado explica o motivo; retry preserva texto.

### P1 — acessibilidade: teclado, modais, labels e tabs

**Área:** WCAG 2.2 AA
**Evidência:** `ChatForm.tsx:437-457,373-391,632-653` intercepta Tab e faz wrap circular; `AppSidebar.tsx:517-568` cria overlays sem `role=dialog`, `aria-modal`, foco, trap, retorno ou Escape; `SearchDialog.tsx:51-61` e `HelpDialog.tsx:25-45` não têm ciclo de foco completo; `ChatForm.tsx:911-920` tem textarea somente com placeholder e `:1013-1047` botão SVG sem nome dinâmico. Tabs em `ImportProjectDialog.tsx:123-126` e `ProductWorkspacePage.tsx:280` não têm `tabpanel`, `aria-controls` ou teclas APG. `index.html:2` declara `lang="en"` embora a UI seja majoritariamente PT-BR.

**Correção:** remover trap global do composer; usar `<dialog>` ou modal APG completo; adicionar labels/nome dinâmico “Enviar mensagem”/“Parar geração”; implementar tabs APG ou links; `lang=pt-BR`; um único `main` e skip link.

**Gate:** Tab/Shift+Tab/Escape/retorno de foco em todos dialogs; setas/Home/End em tabs; accessibility tree e leitor de tela em light/dark; sem declaração AA antes da revalidação.

### P1 — mobile: Studio e Criações cortam ações

**Área:** design responsivo e touch
**Evidência:** `docs/evidencias/screen-f12-studio-mobile.png` mostra toolbar cortada em 390x844; `StudioCanvasPage.tsx:352-377,377-447` mantém header/ações sem wrap/menu. `screen-f12-creations-mobile.png` recorta “Construir agora” e categorias; `CreationsPage.tsx:93-119` não usa wrap. `AgenticSplitShell.tsx:171-231` comprime ID/status/objetivo; `SettingsTabs.tsx:11,19` depende de scroll sem affordance.

**Correção:** em `<=640px`, empilhar ou tornar toolbars explicitamente roláveis com affordance, preservar labels e garantir alvos touch maiores; colocar status/ações em segunda linha.

**Gate:** asserts E2E em 375/390/768/1280/1920 para bounding boxes, overflow esperado e foco.

### P2 — upload, filas e persistência têm riscos de sincronização/limites

**Área:** backend, Redis, durabilidade
**Evidência:** `server/upload.go:129-216,319-341` acessa `done/err` sem mutex/atomic/canal; `:281-289` muta token em retries. `internal/agent/context.go:240-260` pode remover projeto e falhar na segunda remoção sem rollback. `internal/agent/redis_queue.go:36-39` declara quotas não usadas e materializa todos os atrasados/leasing em `:197-230,408-467`; `queue.go:479-495,521-547` engole falhas de Claim/List. `store.go:906-928` não sincroniza diretório após Rename.

**Correção:** canal `done`/mutex/atomic, serializar refresh, deleção transacional/tombstone, quotas e lotes/cursor Redis, health/backoff e fsync do diretório ou journal/WAL.

**Gate:** `go test -race`, carga Redis, falha na segunda remoção, recuperação após interrupção e upload concurrente com 401.

### P2 — evidência E2E e estados 401 não sustentam os claims

**Área:** QA, observabilidade, erro
**Evidência:** `browser-console-audit-night-t3-t7.json:30-54` registra 401 em `/scheduled` e `/settings`, mas não há estado inline equivalente; `AppSidebar.tsx:134-147` faz catch silencioso de `fetchUser`. `test_all_sidebar_screens.mjs:27-108` coleta screenshots/contagens, mas não assertiva overflow, foco, teclado ou hit area. Evidências antigas mostram “Ollama Full”/Studio v17, preview atual mostrou Hades/Studio v24.

**Correção:** 401/403 inline com contexto/retry/autenticação; asserts DOM/keyboard/focus/target; estados loading/empty/error; evidência com commit/build ID.

### P2 — i18n, reduced motion, contraste e targets

**Área:** design system e acessibilidade visual
**Evidência:** onboarding/model picker/chat/error misturam inglês e português (`Onboarding.tsx:194-365`, `ModelPicker.tsx:188,211,215,337-340`, `ChatForm.tsx:833-915`, `ErrorMessage.tsx:63`). `MessageList.tsx:126-149` mantém typing infinito, não coberto por `prefers-reduced-motion` em `index.css:131-145`. `AppSidebar.tsx:73,232` usa tokens potencialmente abaixo de contraste AA; alguns botões de ícone podem ficar abaixo de 24px, a confirmar com medição renderizada.

**Correção:** locale PT-BR extraído, reduced motion global, medição de contraste light/dark, alvos `>=24px` (preferencialmente 44px).

## 4. Melhorias imediatas para leigos

1. Banner nas páginas upstream explicando “Esta é documentação do Ollama upstream; para Hades use este quickstart”.
2. Um único caminho de instalação Hades para Unix e Windows, com nome de binário consistente.
3. Onboarding com **Verificar servidor**, endereço/porta, modelo instalado e ação clara de download.
4. Estado sem modelo com **Instalar modelo**, espaço/memória/tamanho e progresso; nunca deixar o composer parecer apenas quebrado.
5. Estado offline com **Servidor desconectado**, **Verificar novamente** e instrução de iniciar/reiniciar.
6. Erros próximos ao composer com **Tentar novamente**, **Escolher modelo** ou **Configurar provedor**, preservando o rascunho.
7. Toolbar mobile sem cortes; ações primárias sempre visíveis.
8. Jornada principal inteiramente em PT-BR, preservando apenas nomes próprios de integrações.

## 5. Roadmap recomendado

### Fase 0 — bloquear release pública

- Corrigir o contexto PR/release e criar matriz de checks por SHA candidato.
- Exigir E2E funcional, package/install/upgrade/rollback, smoke de instalador, SBOM por artefato, provenance e assinatura verificável.
- Rotular artefatos unsigned como preview; não chamar isso de release assinada.

### Fase 1 — fechar fronteiras de confiança

- Sandbox forte para MCP local.
- Secret broker com ACL por organização/destino/purpose.
- GitHub App/OAuth por tenant para importação privada.
- Validar IDs/caminhos de schedule.
- Impedir self-approval em ações sensíveis.
- Unificar OAuth/Remote MCP no transporte e audit trail central de egress.

### Fase 2 — runtime cancelável e durável

- `sendContext`, cancel func por job e backoff cancelável.
- Validar 206/Content-Range/digest.
- Corrigir estado concorrente de upload.
- Quotas/lotes/health no Redis.
- Deleções transacionais e durabilidade do diretório após Rename.

### Fase 3 — jornada de primeiro uso

- Sequência canônica Hades para instalação.
- Estados explícitos para sem modelo, offline, provider ausente e erro de envio.
- Download consentido com dados de tamanho/memória/espaço/progresso.

### Fase 4 — WCAG e mobile

- Modal completo, tabs APG, skip link, `lang=pt-BR`, labels e foco.
- Reduced motion global, contraste medido e targets adequados.
- Studio/Criações/Agentic responsivos em 390px.

### Fase 5 — evidência contínua

- E2E por estado e fluxo, não só screenshots.
- Viewports 375/390/768/1280/1920.
- Testes nativos dos OS suportados e leitor de tela.
- Evidência com SHA/build ID e relatório de console/network.

## 6. Checklist de aceite para o Claude

- [ ] Nenhum processo MCP local roda fora de sandbox forte em modo protegido.
- [ ] Nenhum tenant referencia env var/secret global de outro escopo.
- [ ] Importação GitHub privada usa credencial vinculada à organização/repositório.
- [ ] Schedule ID não permite traversal em Unix, Windows ou symlink.
- [ ] Downloader rejeita 200/206/416 inválidos e só publica blob após digest.
- [ ] Disconnect/cancel/shutdown termina produtores, goroutines, retries e jobs.
- [ ] Self-approval de write/external/destructive é rejeitado.
- [ ] Release gate executa no mesmo SHA todos os checks exigidos.
- [ ] E2E funcional cobre missão, aprovação, cancelamento, persistência, artifact e download.
- [ ] Install/upgrade/rollback é testado usando os artefatos reais.
- [ ] SBOM/provenance/assinatura são por artefato e verificáveis fora do job.
- [ ] Primeiro uso Hades não aponta para upstream nem usa nome de binário divergente.
- [ ] Sem modelo/offline/erro têm CTA e recuperação visíveis.
- [ ] Composer, dialogs e tabs passam teclado, foco, screen reader e labels.
- [ ] Studio e Criações passam 390x844 sem corte de ações.
- [ ] E2E registra commit/build ID e asserts overflow/foco/targets/estados.

## 7. Veredito final

**Estado atual: CANDIDATO DE RELEASE INTERNO, não publicação pública/produção.**

O projeto avançou muito além de um protótipo visual, mas ainda não é honesto classificá-lo como 100% funcional ou pronto para qualquer usuário final. A prioridade deve ser fechar P0/P1 com implementação + teste negativo + execução real, e só então reavaliar a matriz de paridade e o release gate.

**Próxima ação recomendada:** criar uma sequência de correções por P0, começando por sandbox/secrets/tenant e pelo release gate; não iniciar novas features de superfície antes desses bloqueios.


## Follow-up — CI e dependências mobile

**Achado reproduzido:** o job Go rodava depois do build web e descobria uma árvore Go acidental dentro de `app/ui/app/node_modules`, causando falhas multiplataforma por diretório transitivo incompleto.

**Correção aplicada:** o `test.yaml` remove a árvore de dependências web antes de `go test` e `go test -race`; o teste Go continua sendo exatamente `./...`.

**Hardening adicional:** o `node-forge` transitivo do Expo é resolvido por um commit upstream que contém a correção ASN.1, através de tarball HTTPS fixado no lockfile. Instalação limpa, typecheck e `npm audit --omit=dev --audit-level=high` passaram.

**Estado:** validado localmente; confirmação final depende do runner remoto macOS/Windows e do `go_mod_tidy` no SHA publicado.


## Follow-up — 2026-10-02 — hardening de CI/DevEx

### Correções aplicadas

- Todas as referências `uses:` nos workflows versionados foram fixadas em SHAs imutáveis, mantendo a tag apenas como comentário documental.
- `dz23-e2e` e `dz23-provider-smoke` agora têm execução semanal e, quando uma execução agendada falha, tentam abrir uma issue com link para o run; a falha do workflow permanece preservada.
- O frontend passou a ter orçamento verificável para o entry JavaScript: máximo de 2.000.000 bytes bruto e 600.000 bytes gzip. O build real mediu **1.823.863 bytes bruto / 530.772 bytes gzip** e passou.
- A tentativa de `manualChunks` foi revertida após evidência de regressão: ela agrupou Shiki em um chunk de 9,5 MB. O Vite já gera chunks de linguagens sob demanda; a otimização regressiva não foi mantida.

### CI remoto — estado honesto

Nos SHAs `8816da06`, `01d94ae2` e `81b05741`, os workflows `class-a-plus-integrity` e `dz23-agentic-quality` terminaram como `failure` em aproximadamente 2–3 segundos, com **zero steps executados e nenhum runner materializado** em todos os jobs. O rerun do SHA `8816da06` repetiu o mesmo padrão. O status público do GitHub indicou “All Systems Operational”, portanto a causa exata do startup failure não foi comprovada pelo código nem pelos logs disponíveis. Isso é um bloqueio de validação remota, não deve ser reportado como CI verde.

### Gates locais desta etapa

- `npm run build`: passou após reinstalação limpa.
- `node scripts/check-bundle-budget.mjs`: passou.
- `npm run lint`: passou.
- `npx vitest run`: **39 arquivos / 265 testes passaram**.
- `node scripts/verify-contracts.mjs`: passou com aviso honesto de 135 rotas backend sem chamador frontend estático; essas rotas incluem endpoints de auth, uploads, media, traces e health e precisam de triagem separada, não devem ser apagadas automaticamente.
- `bash scripts/check-class-a-plus-integrity.sh`: passou.

**Reclassificação:** o produto continua como candidato de release interno. O bundle budget e o hardening de supply chain estão implementados, mas a liberação pública permanece bloqueada até o GitHub Actions executar os jobs em runners reais e todos os checks obrigatórios concluírem no mesmo SHA.

## Follow-up — 2026-10-02 01:28 UTC — fechamento técnico e CI externo

- **Gates locais:** PASS após remover `node_modules` gerado pelo smoke web antes da descoberta Go. Passaram build nativo, build `CGO_ENABLED=0`, testes dos pacotes agent/server, `go vet`, contratos, integrity, frontend completo e bundle budget (entry 1.823.863 bytes bruto / 530.772 gzip).
- **CI remoto:** NÃO aprovado. Os workflows `class-a-plus-integrity` (`36949729206`) e `dz23-agentic-quality` (`36949729166`) foram rerunados no SHA `2444e6d2`, mas nenhum job iniciou. A annotation oficial do GitHub é: **“The job was not started because your account is locked due to a billing issue.”**
- **Classificação:** `BLOCKED_EXTERNAL`, não falha de implementação. O bloqueio precisa ser resolvido pelo proprietário da conta/organização no GitHub; não deve ser mascarado por skip, alteração de required checks ou declaração de CI verde.
- **Ação pendente:** após desbloqueio, rerodar os dois workflows e o `test.yaml` completo, confirmando o mesmo SHA.


## Follow-up F1/F2 — 2026-10-02

- **F1 resolvido:** o controle de memória de longo prazo deixou de ser um switch decorativo (`checked={true}` sem persistência) e passou a declarar “Ainda não configurada”.
- **F2 resolvido:** o Endpoint deriva o estado do host de `/api/tags`, com estados `Verificando…`, `Online` e `Offline`; não há mais claim de Online quando o backend está indisponível.
- **Evidência local:** `npx tsc -b`, `npm run lint`, `npx vitest run` (266 testes), `npm run build`, `go test ./internal/agent ./server`, `verify-contracts` e `check-class-a-plus-integrity` passaram.
