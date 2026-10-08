# Auditoria ponta a ponta — Hades (DZ23-LTDA/hades-ollama)

**Data:** 2026-10-01
**Branch auditada:** `recovery/ollama-full-snapshot` (HEAD `ef9c21d`, código mais recente)
**Método:** inspeção direta do código/CI/docs + verificação dos runs no GitHub Actions (`gh run list`, `gh api`) + 4 auditorias paralelas (usuário/docs, engenharia/CI, design/UX/a11y, arquitetura).
**Veredito geral:** produto **real, extenso e honesto**, em estado de **RC local em hardening** — não production-ready, e o próprio projeto declara isso. Não encontramos "100% de paridade" falsa; ao contrário, o time é exemplarmente transparente (`NOT_CONFIGURED`/`BLOCKED_EXTERNAL`). O que existe de **grave** são falhas pontuais de governança e segurança que precisam de correção, não um problema estrutural.

---

## 1. Achados críticos (corrigir primeiro)

### C1. `release-gate` trava a publicação para sempre — ALTA
`.github/workflows/release.yaml:822-835` exige os checks `"PR gate"`, `"test (ubuntu|windows|macos-latest)"`, `"race (ubuntu|macos-latest)"` e `"go_mod_tidy"`. Todos esses **só são criados em `pull_request`** (`test.yaml`, `pr-gate.yaml`), e o `release.yaml` é `workflow_dispatch` (sem trigger de tag). Logo, no SHA de release esses check-runs **não existem** → o loop (linhas 840-847) marca `FAIL` → `exit 1` → o job `release` (linha 858) nunca roda.
Contradição interna: o próprio `release-readiness.yaml:56-74` **documenta e corrige** isso, excluindo esses checks da lista. O `release-gate` deveria espelhar a lista do `release-readiness`.
*Para leigos:* o porteiro da release pede carimbos que ninguém emite naquele commit; na primeira publicação real, trava para sempre.

### C2. Regra de proteção da `main` real está MAIS FRACA que o arquivo no repo — ALTA (governança)
O arquivo `.github/rulesets/main-protection.json` diz `required_approving_review_count: 1`, `require_last_push_approval: true`, `required_review_thread_resolution: true`. Mas o ruleset **realmente aplicado** no GitHub (id `23940518`) tem:
- `required_approving_review_count: 0` ← **nenhuma aprovação humana é exigida para merge na main**
- `require_last_push_approval: false`
- `required_review_thread_resolution: false`
- `require_code_owner_review: false` (CODEOWNERS é decorativo)

O que **está** ativo: `deletion`/`non_fast_forward` bloqueados + `required_status_checks` ("Preserve Class A+ surfaces" + "PR gate", strict). Ou seja: **CI é obrigatória, mas revisão humana não é.** O JSON do repo está dessincronizado do ruleset vivo.
*Para leigos:* o "manual" diz que precisa de 1 aprovação, mas na prática qualquer merge com CI verde passa sem ninguém revisar.

### C3. `main` (pública) está 14 commits atrás da branch de desenvolvimento — ALTA
`origin/main` = `3719378` (merge do PR #40), `origin/recovery/ollama-full-snapshot` = `ef9c21d` (**14 commits à frente**). Esses 14 commits incluem correções de CI reais: `fix: avoid mtime race in Codex request counting` (flaky do `go test ./...`), `ci: use actual Windows installer output name` e `ci: prepare SignPath/SSL.com Windows signing` (o installer chegou a falhar no checksum). O default do GitHub é `main`, então **quem clona e builda pelo `main` pega código com esses bugs**.
*Para leigos:* a "vitrine" do GitHub mostra uma versão mais velha que a de verdade, com falhas que já foram arrumadas mas ainda não publicadas.

### C4. Duas políticas de egress zero-trust divergentes (uma delas default-allow) — ALTA (segurança)
`internal/agent/egress_zero_trust.go` usa `unsafeEgressIP` (`internal/agent/egress_policy.go:88`) que é **default-block** (bloqueia RFC 2544 198.18.0.0/15, ranges de documentação, IPv6 special-use). Já `internal/multillm/egress_zero_trust.go:24-66` (`ClassifyProviderEgressIP`) é uma cópia independente que **não** cobre esses ranges e termina em `return false, ""` (**default-allow**) na linha 65. O caminho multi-provider é menos restritivo que o caminho agent e não herda correções.
*Para leigos:* a "cerca" que decide o que a IA pode acessar foi feita duas vezes, com regras diferentes — uma delas deixa passar mais coisa.

### C5. Freio HITL do loop autônomo decide por "achar palavras" — ALTA (segurança)
`internal/agent/supervisor.go:270-349` decide se um ciclo do Company OS exige aprovação testando `strings.Contains` em objective+name: `"spend"|"budget"|"buy"|"comprar"|"pagar"…` (gasto) e `"publish"|"deploy"|"publicar"|"send_external"…` (publicação). Sinônimos (`"transfer"`, `"acquire"`, `"postar"`, `"lançar"`) não disparam o freio. É heurística de texto, não Policy Engine (o `RiskClass` já existe em `types.go:42-49` mas não é usado aqui). A marcação `BLOCKED_EXTERNAL` também é por lista fixa (`"tiktok_shop"|"meta_ads"|"shopify_sync"|"whatsapp_live"`, linhas 338-341).
*Para leigos:* o freio de "pedir permissão antes de gastar/publicar" procura palavras específicas; trocando a palavra, ele não dispara.

### C6. Nomenclatura/binário quebrados após 3 renomes — ALTA (usuário)
O produto teve 3 nomes ("Ollama Classe A+" → "Ollama Full" → "Hades"). Resultado no código/docs:
- `README.md#L53` builda `bin/ollama-full`, mas `README.md#L177` chama `./bin/ollama-classe-a-plus agent create …` (comando que não existe).
- `CLASS_A_PLUS_GUIDE.md#L50-51`: `git clone …/hades-ollama.git` seguido de `cd ollama-classe-a-plus` — pasta que **não existe** (o clone cria `hades-ollama`). Quebra o quickstart no passo 1.
- `docs/quickstart.mdx` é o quickstart do **Ollama upstream** (manda baixar de `ollama.com/download` e usar o binário `ollama`), não do Hades.
*Para leigos:* o "comece aqui" ensina a instalar outro programa, e os comandos apontam para nomes de pasta/arquivo que não existem.

---

## 2. Como USUÁRIO (docs, onboarding, uso)

### CORRIGIR
- **Binário/`cd`/clone inconsistentes** — ver C6 (ALTA).
- **`docs/quickstart.mdx` é do upstream** (ALTA): ensina o Ollama oficial, não o Hades.
- **Branch canônica ambígua** (ALTA): `MEMORIA.md#L110` diz `recovery/ollama-full-snapshot`; `CONTRIBUTING.md` e o guia mandam "branch a partir de `main`". Colaborador que clona cai em `main` (sem o trabalho do recovery).
- **Modelo recomendado `gemma4:e2b`/`gemma4:31b` não confirmado** (MÉDIA): `README.md#L78/L157` e `scripts/hades-quickstart.sh#L12` recomendam `gemma4` (~7.2 GB); se o nome não existir, o `pull` falha e o onboarding trava.
- **CONTRIBUTING.md é o do upstream** (MÉDIA): título "Contributing to Ollama", links de issue/security/Discord para `github.com/ollama/ollama` e `discord.gg/ollama`.
- **Templates de issue apontam para o Ollama upstream** (MÉDIA): `config.yml` e `10_bug_report.yml` pedem `ollama --version` e linkam o Discord do Ollama.
- **Caminhos Linux `/tmp` em comandos** (MÉDIA): `CONTRIBUTING.md#L97` e `CLASS_A_PLUS_GUIDE.md#L256` usam `/tmp/ollama-classe-a-plus` (não roda no Windows).
- **Links mortos** (BAIXA): `CHANGELOG.md#L39` → release `v0.1.0`; `CLASS_A_PLUS_GUIDE.md#L313` → branch `feat/manus-parity-omniroute` (não existe mais).
- **MEMORIA.md desatualizado** (BAIXA): `projeto: ollama-classe-a-plus`, caminhos `/home/ubuntu/…`, e **dois** blocos "## Histórico de sessões".

### MELHORAR
- Consolidar o nome do produto (convivem "Hades", "Ollama Full", "Ollama Classe A+", "Ollama DZ23 Agentic Platform").
- Padronizar idioma (README metade EN, SECURITY.md repete instruções).
- Criar uma seção "Pré-requisitos" com versões mínimas (Go/Node).
- Documentar o fluxo Windows em primeiro lugar (é o ambiente real); hoje o único "um comando" é bash.
- Templates `20_feature_request.md` e `30_model_request.md` estão vazios (só frontmatter).
- Criar `.github/PULL_REQUEST_TEMPLATE.md` (não existe; o `pr-gate.yaml` até o referencia em paths-ignore).

### AMPLIAR (sugestões para leigos)
- **"O que é o Hades e para quem"** em 4-6 linhas sem jargão (prioridade).
- **Guia passo a passo para leigo** (Windows primeiro), substituindo `quickstart.mdx`.
- **FAQ pt-BR** (o que é modelo local? quanto de RAM/VRAM? por que não é o Ollama oficial? o que é "agentic"/"local-first"?).
- **Glossário** de uma linha por sigla (MCP, RLS, DLQ, OTLP, IdP, PKCE, CAS, RBAC).
- **Vídeo narrado** de 2-3 min (instalar → rodar → primeira missão) com legenda.
- **Tabela "o que funciona hoje vs. o que é futuro"** em linguagem simples.

---

## 3. Como ENGENHEIRO (código, CI/CD, segurança, build)

### CORRIGIR
- **`release-gate` trava a release** — ver C1 (ALTA).
- **CODEOWNERS decorativo** (MÉDIA): `require_code_owner_review: false` no ruleset vivo.
- **Ações de terceiros pinadas por tag mutável** (MÉDIA): `al-cheb/configure-pagefile-action@v1.5` (release.yaml:241) e `signpath/github-action-submit-signing-request@v3` (windows-installer.yaml:81). Padrão correto já existe no repo (`anchore/sbom-action@<sha>`).
- **Windows installer falhou no checksum** (registro, já resolvido): run `36934704349` — corrigido por "use actual Windows installer output name" + "prepare SignPath signing".

### MELHORAR
- `release-gate` usa `tail -1` (release.yaml:841) — não trata tentativa `cancelled` por concurrency (o `pr-gate` e o `release-readiness` já fazem o certo).
- **Sem `timeout-minutes`** nos jobs de gate (default 360 min): um hang queima 6 h de runner.
- `.golangci.yaml:21` desabilita `errcheck` (e `govet` desabilita `unusedresult`) — num codebase de egress/segurança, erro não checado é exatamente o que se quer pegar.
- `release` (release.yaml:858) **não depende** de `docker-merge-push` (linha 767) — o merge do manifest multi-arch pode falhar enquanto artefatos já publicam.
- `.dockerignore` não ignora `apps/` nem `**/node_modules` — o `COPY . .` envia `node_modules` do mobile ao daemon.
- **Node inconsistente**: 20 vs 22 entre workflows.
- `test.yaml:488` usa `if: always()` no step de benchmark → falso negativo em cascata.
- `go.mod:3` pin `go 1.26.6` (patch exato) — confirmar que o toolchain existe no dl server (Dockerfile e test.yaml baixam exatamente esse patch).
- `verify-contracts.mjs` `methodNear` cai para `GET` quando não acha `method:` — pode mascarar mismatch POST→GET.

### AMPLIAR
- **`dz23-provider-smoke.yaml` injeta 18 chaves reais sem `environment:` protegido** — qualquer write pode disparar e gerar custo.
- Falta **`dependency-review-action`** nos PRs (bloquear CVE/malware na entrada; hoje só `govulncheck` + `npm audit`).
- Falta **varredura de segredos dedicada** (gitleaks/trufflehog) — o guard atual é grep regex que não pega `ghp_…`/`AKIA…` inline.
- Sem gate de **build reproduzível** Windows/macOS (só o tarball Linux usa `SOURCE_DATE_EPOCH`).

**Já está forte:** egress zero-trust com auditoria + pin de IP; `RedactValue`/`RedactDLP`; classificador fail-safe com teste de consistência; `pr-gate` anti-deadlock docs-only; teste de RLS/cutover do Postgres nível "banco de hardening".

---

## 4. Como DESIGNER (UI, UX, acessibilidade)

### CORRIGIR
- **`<html lang="en">`** (index.html:2) num app 100% pt-BR — leitor de tela anuncia em inglês; `theme-color` fixo em tom escuro (ALTA).
- **`select-none` no Chat e Settings** (Chat.tsx:241,245; Settings.tsx:430,451) impede copiar resposta/endereço (ALTA).
- **Dois design systems paralelos** (Catalyst/Headless `ui/` com tokens `zinc` vs páginas `neutral`/`violet`/`emerald` ad hoc); `tailwind.config.js` v3 morto (o app é Tailwind v4) (ALTA).
- **Botão de envio do chat sem nome acessível** (ChatForm.tsx:1013-1027) e anexo só com `title` (ALTA).
- **Modais da sidebar sem `role="dialog"`/focus trap/Escape** (AppSidebar.tsx:518,553) — inconsistente com SearchDialog/HelpDialog.
- **SearchDialog/HelpDialog sem focus trap** e sem retorno de foco (MÉDIA).
- **Mistura pt-BR/EN no Settings** (MÉDIA) — "Connect Ollama Account/Sign out/Cloud…" + frase quebrada "Tamanho do contexto determines…".
- **Microcopy do chat em inglês** ("Send a message", "Press ESC…", "This model does not support images") — o ponto de maior contato (MÉDIA).
- **Tab customizado no chat** intercepta a navegação nativa (ChatForm.tsx:437-458) — pode prender o foco (MÉDIA).
- **`window.confirm` nativo em inglês** ("Discard unapplied app model changes?") (MÉDIA).
- **Switch "Memória de Longo Prazo" é decorativo** (Settings.tsx:861-864) — `checked={true}` sem persistência (MÉDIA).
- Alternadores/abas da Biblioteca sem `aria-pressed`/`role="tablist"`; rótulos sem `htmlFor`; título `<h2>` onde deveria ser `<h1>`; toast "Salvo" sem `aria-live` (BAIXA).
- **App mobile sem modo escuro** e com cores hardcoded; título "Mission mobile" (EN) (MÉDIA).

### MELHORAR
- Centralizar tokens no `@theme` do Tailwind v4; remover o `tailwind.config.js` morto.
- Escolher UM sistema de componentes (adotar `ui/*` ou abandonar o kit Catalyst).
- Padronizar 100% pt-BR com glossário fixo.
- Estados de loading/erro consistentes (a Library tem o empty state exemplar).
- Normalizar ícones (20/solid vs 24/outline vs SVG à mão).
- Revisar microcopy da Home ("Automático · grátis-primeiro", "Escolher modelo PASS").

### AMPLIAR
- **Landmarks + skip-link** ("pular para conteúdo") — hoje inconsistente.
- **Combobox acessível** (SearchDialog mistura `role="listbox"` com `<button>` aninhado em `<option>`); usar `aria-activedescendant`.
- **Modo escuro no mobile** (o desktop já tem Claro/Escuro/Automático persistido).
- Onboarding/motion com `framer-motion` (já instalado), respeitando `prefers-reduced-motion`.

**Já está bom:** `aria-current="page"`, `role="status"/aria-live` em vários avisos, tema claro/escuro/auto persistido, `prefers-reduced-motion` tratado, empty state da Biblioteca, brand-kit completo, `HadesEmblem` com `role="img"`.

---

## 5. Como ARQUITETO (ponta a ponta)

### CORRIGIR
- **Duas políticas de egress divergentes** — ver C4 (ALTA).
- **HITL/BLOCKED_EXTERNAL por substring** — ver C5 (ALTA).
- **CAS do PostgresStore com org vazia** (MÉDIA): `postgres_store.go:392` usa `($23 = '' OR organization_id=$23)` — org vazia torna o predicado sempre true e delega o isolamento só à RLS. Mitigado, mas cheiro que pode mascarar bug de wiring.
- **Backoff exponencial sem jitter** (MÉDIA): `queue.go:426-429` — thundering herd em retries em massa.
- **SQLite prometido mas nunca implementado** (BAIXA): `ARCHITECTURE.md:53` cita "SQLite/PostgreSQL"; não há driver SQLite — o store local real é JSON por-arquivo.
- **"Store local transacional" impreciso** (BAIXA): há atomicidade por arquivo + CAS, mas **não há transação ACID missão↔eventos**; leituras serializam num `mu.Lock()` global.
- **ARCHITECTURE.md subdocumenta a API** (BAIXA): lista 8 endpoints; o real tem ~150 rotas.

### MELHORAR
- Consolidar egress numa política única (`multillm` consumir `agent.NewSafeEgressHTTPClient`).
- Quebrar arquivos-monstro: `agent_routes.go` (101 KB), `routes.go` (99 KB), `runtime.go` (80 KB), `workspace_snapshot.go` (71 KB), `postgres_security.go` (63 KB), `sched.go` (58 KB).
- Classificador de risco por enum/`RiskClass` em vez de substring.
- Centralizar configuração `OLLAMA_AGENT_*` (espalhada) num pacote tipado.
- Remover dead code (`egress_policy.go:138,160` marcado `nolint:unused` "para o futuro").
- Jitter no backoff (e no Redis).
- Documentar contrato real do store (JSON, não SQLite) e usar `RLock` em leituras.

### AMPLIAR
- **Postgres/Redis/OTLP → produção**: adapters existem e o CI já roda RLS+DLQ+OTLP reais; falta HA/lease distribuído e failover operacional.
- **Observabilidade com SLO/alerta**: há spans + `/metrics/prometheus`, mas faltam SLOs por missão, correlação por `correlation_id`, dashboards e alertas de DLQ.
- **`Idempotency-Key` uniforme** nos POSTs mutáveis.
- **Sandbox forte fora do Linux** (AppArmor/SELinux no Linux; Sandbox/AppContainer em macOS/Windows).
- **Migrações de schema versionadas** (hoje SQL solto + `migrate-existing-roles.sql`/`retire-legacy-role.sql` sem runner com checksum).
- **Auditoria por chamada de tool/MCP + DLP semântico** (a matriz admite que "heurística DLP não comprova ausência de segredo desconhecido").

**Já está forte:** egress zero-trust do agent; RLS Postgres com `FORCE ROW LEVEL SECURITY` + `NOBYPASSRLS` + `SECURITY DEFINER` + tenant context HMAC; sandbox Linux strict real (namespaces+cgroup v2+seccomp+rlimits); fila com lease/ack/nack/DLQ/replay e CAS versionado; honestidade de paridade genuína.

---

## 6. Seção para leigos — para o Claude auditar

> Pedido do usuário: ideias de melhoria em linguagem simples, para o Claude revisar.

1. **"Explique o que é o Hades em uma frase"** — hoje não existe. Sugestão: *"Hades é um assistente de IA que roda no seu próprio computador (sem depender da nuvem) para automatizar tarefas, pesquisar e construir coisas — com você no controle do que ele pode fazer."*
2. **Caminho feliz em 10 minutos** — uma página única: instalar Go e Node → rodar `install.ps1` (Windows) → abrir `http://127.0.0.1:5173` → rodar o primeiro modelo. Sem presumir que a pessoa sabe o que é terminal.
3. **Perguntas frequentes em linguagem simples** — "Quanto de memória preciso?", "Preciso de placa de vídeo?", "Isso envia meus dados para a nuvem?", "Por que aparece 'não configurado' em vários lugares?".
4. **Dicionário de siglas** — uma linha para cada: MCP, RLS, DLQ, OTLP, IdP, RBAC, PKCE, CAS.
5. **Tabela "o que dá para usar hoje vs. o que é promessa"** — o projeto já separa isso honestamente, mas em jargão de auditoria; traduzir para "dá para usar hoje / em breve / depende de conta externa".
6. **Vídeo de 2-3 minutos** narrado em português, do "instalar" até a "primeira missão".
7. **Fixar UM nome** — decidir "Hades" e tratar "Ollama Full"/"Classe A+" como nomes antigos, em todos os arquivos e comandos (hoje há 3 nomes e comandos quebrados).
8. **Corrigir a "porta de entrada"** — o quickstart hoje ensina a instalar o Ollama oficial, não o Hades; isso deve ser a primeira coisa a arrumar.
9. **Deixar o celular igual ao computador** — modo escuro e textos em português (hoje o app mobile é só claro e tem título em inglês).
10. **Copiar resposta com o mouse** — hoje não dá para selecionar/copiar a resposta do chat; usuário comum sente isso como "bug".

---

## 7. Priorização sugerida

| # | Ação | Esforço | Impacto |
|---|---|---|---|
| 1 | Corrigir `release-gate` (espelhar lista do release-readiness) | baixo | destrava publicação |
| 2 | Sincronizar ruleset real da `main` com o JSON (≥1 aprovação + CODEOWNERS) | baixo | governança |
| 3 | Merge `recovery` → `main` (publicar os 14 commits de correção) | baixo | usuário pega código corrigido |
| 4 | Unificar egress (remover default-allow do `multillm`) | médio | SSRF/segurança |
| 5 | Trocar HITL por `RiskClass`/policy em vez de substring | médio | segurança do loop autônomo |
| 6 | Corrigir nomes/comandos dos docs (C6) | baixo | onboarding |
| 7 | A11y crítica (lang, select-none, aria-label do botão de envio) | baixo | acessibilidade |
| 8 | `dependency-review-action` + varredura de segredos | médio | supply chain |
| 9 | Quebrar arquivos-monstro / consolidar design system / docs para leigos | alto | manutenção e adoção |

*Nota: este arquivo foi escrito no working tree local (não commitado). Nada foi alterado no código do projeto nem publicado no GitHub.*
