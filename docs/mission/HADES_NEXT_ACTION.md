## JANELA E-1320

A cabeça do índice é **E-1320**.

- **E-1320** — **a primeira das 93 lacunas passou a ser produto.** `/retention` (GET/PUT) e
  `/retention/apply` (POST) eram três das 93 medidas no E-1319 — e as mais irónicas, porque o E-1316,
  E-1317 e E-1318 foram todos sobre retenção que existia e não acontecia.
  *Uma rota que existe e não é consumida é uma promessa do backend, não uma funcionalidade do produto.*
  Entregues `retentionClient.ts` + `RetentionPanel.tsx`, montado no `CapabilitiesPage`. O painel lê a
  política, grava-a e pede a aplicação, mostrando os **números reais** do que foi removido.
  **As decisões que o painel não pode errar:**
  **(1) `max_memories_per_project = 0` significa SEM TECTO** — o servidor aceita 0 como legítimo
  (0..10000). Dizer «0 memórias por projecto» diria o oposto do que a política faz.
  *Zero não é sempre um limite: às vezes é a ausência dele, e uma interface que confunde os dois mente
  sobre o que vai apagar.*
  **(2) «Não configurada» é um ESTADO, não um erro.** **(3) Gravar não é aplicar** — o aviso diz
  «Nada foi removido ainda». **(4) Zero removido diz-se em palavras.**
  **Dois defeitos meus, apanhados pelos gates:** `TS2367` numa condição que não filtrava nada; e o
  teste do `act` — no mesmo `act` o `onChange` e o clique não deixam re-renderizar, o clique apanhava
  o estado antigo e o PUT saía. *Um teste que mede o estado anterior aprova o defeito que devia travar.*
  **O TETO DESCEU: 93 → 90**, e o `TETO_ORFAS` acompanhou.
  *Baixar o teto é progresso; deixar o teto em 93 depois de fechar 3 rotas seria guardar crédito por
  trabalho já feito.*
- **E-1319** — auditoria de rotas órfãs: 93 lacunas, com gate e teto.
- **E-1318** — `PodarDisco` ligado ao arranque.
- **E-1317** — retenção nos dois sítios; `misspell` resolvido por `ignore-rules`.

## A distância, agora

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **106** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **90** (era 93) |

O que continua aberto: **90 rotas** ainda sem superfície no UI; as 93 não foram reclassificadas uma a
uma; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel de
estado do build **não se actualiza sozinho**; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`;
**FASE 30 não iniciada**.

---
## JANELA E-1319

A cabeça do índice é **E-1319**.

- **E-1319** — **duas rondas seguidas com o mesmo padrão, e eu a procurar um de cada vez.**
  O E-1316 foi `persistir()` escrito e nunca invocado; o E-1318 foi `PodarDisco()` escrito e nunca
  invocado. Achei os dois por sorte, dentro da fatia em que estava a trabalhar. Se o padrão aconteceu
  duas vezes em dois subsistemas que eu conhecia, o número real tinha de ser **medido**.
  *Um defeito que se encontra por sorte não foi encontrado: foi tropeçado.*
  **MEDIDO:** 348 rotas no backend · **109 não referenciadas** no frontend nem no E2E · dessas
  **16 são orfandade CORRECTA** (webhooks, `/metrics`, `/auth/saml/*`, `/mcp/rpc`, pareamento) e
  **93 são lacuna de produto**. Maiores: **scheduling 14, commerce 12, crm 8, inbox 8, sales 8,
  service-agents 6, omnichannel 5** — 21 domínios.
  *Uma rota que existe e não é consumida é uma promessa do backend, não uma funcionalidade do produto.*
  **Entregue** `scripts/hades_rotas_orfas.py` + teste, com **`TETO_ORFAS = 93`**: falha se subir, baixa
  quando descer, e **exige razão escrita para cada prefixo de consumidor-externo** — para que
  «orfandade correcta» não vire a gaveta onde se esconde o que não se quer ver.
  *Um teto não é uma aprovação: é o número de hoje, congelado para que a próxima ronda não o aumente
  sem que alguém escreva porquê.*
  **E o gate está LIGADO ao `hades_local_gates.ps1`** — sem isso seria o terceiro instrumento escrito e
  nunca invocado na mesma sessão. *Um gate que existe e não é invocado é um gate que não existe.*
  **O que o instrumento NÃO prova:** não prova que as 93 sejam todas necessárias no UI — algumas podem
  ser legitimamente só de CLI e estar mal classificadas. Reclassificar é a próxima fatia.
- **E-1318** — `PodarDisco` ligado ao arranque.
- **E-1317** — retenção nos dois sítios; `misspell` resolvido por `ignore-rules`.
- **E-1316** — persistência do store ligada ao `DataRoot`.

## A distância medida

| categoria | rotas |
|---|---|
| no backend | **348** |
| órfãs no frontend | **109** |
| consumidor externo (correcto) | **16** |
| **lacuna de produto** | **93** |

O que continua aberto: as 93 não foram reclassificadas uma a uma; **nenhuma** foi ligada ao UI nesta
ronda; o build **não é assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel de
estado **não se actualiza sozinho**; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não
iniciada**.

---
## JANELA E-1318

A cabeça do índice é **E-1318**.

- **E-1318** — **a função de limpeza existia e nunca corria.** Entreguei `PodarDisco()` no E-1317 com
  quatro testes verdes — e **nenhum chamador no produto**. O `storeDeBuilds()` chamava
  `CarregarDoDisco()` e devolvia. Um servidor que arranca com 500 ficheiros e fica parado nunca volta a
  chamar `podar` (que só corre a cada build novo), portanto o disco nunca descia.
  *Uma função de limpeza sem chamador é a mesma família do store que persistia e nunca escrevia: código
  que existe e não acontece.*
  **É a SEGUNDA VEZ no mesmo subsistema.** No E-1316 o defeito foi `persistir()` escrito e nunca
  invocado; agora `PodarDisco()` escrito e nunca invocado. Os dois passaram nos testes porque o teste
  chamava a função diretamente.
  *Um teste que chama a função que o produto não chama prova que a função funciona, não que o produto a
  usa.*
  Entregue `PodarDisco()` no arranque, depois do `CarregarDoDisco()`; `MaxBuildsGuardadosParaTeste()`
  expõe o teto; teste `TestArranqueDoStoreAplicaRetencao` com 212 ficheiros.
  **Um `nolintlint` FALSO:** com o typecheck do teste quebrado, o `golangci-lint` reportou dois
  `//nolint:nilerr` como «unused» **noutro ficheiro**. Corrigidos os imports, 0 issues.
  *Um linter que reporta noutro ficheiro quando um terceiro não compila ensina a procurar o defeito onde
  ele não está.*
- **E-1317** — retenção nos dois sítios; `misspell` resolvido por `ignore-rules`.
- **E-1316** — persistência do store ligada ao `DataRoot`.
- **E-1315** — botão de iniciar build.

## O que continua aberto

Build **não assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o painel de estado **não se
atualiza sozinho**; **a poda no arranque corre uma vez** — um servidor com uptime longo volta a crescer
até ao próximo build, e não há poda periódica; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30
não iniciada**.

---
## JANELA E-1317

A cabeça do índice é **E-1317**.

- **E-1317** — **dois defeitos: um de retenção, um de ferramenta.**
  **(1) A retenção só valia para metade dos sítios.** O `podar()` descartava os registos terminados do
  mapa em memória e **deixava os ficheiros em disco a crescer**. *Uma política de retenção que só conta
  os registos que caberam em memória não é uma política de retenção: é uma política de memória.*
  Corrigido: `podar()` apaga o `.json` com o mesmo critério, e há `PodarDisco()` para o caso que o
  `podar` não cobria — um processo que arranca, carrega 200 registos e fica parado nunca volta a chamar
  `podar`. `PodarDisco` **nunca remove um build a correr neste processo**.
  **(2) A oitava ocorrência do `misspell` — e a causa era outra.** O `.golangci.yaml` **já tinha
  `ignore-rules`** para o `misspell` com cinco palavras portuguesas legítimas. `discriminante` não
  estava lá. E o `misspell` **só flagra o SINGULAR** — o plural `discriminantes` já passava, como
  `toolchain_plan_test.go:3` provava sem que eu reparasse. As outras ocorrências estão em `.ts`/`.tsx`,
  que o `golangci-lint` não analisa.
  **Passei oito rondas a reescrever comentários quando a solução era uma linha que o repo já tinha.**
  *Reescrever o comentário oito vezes é tratar o sintoma de um corretor ortográfico com a ferramenta
  errada.* — e o comentário reescrito ficava **pior**, porque perdia o termo técnico exacto.
  **Prova da ignore-rule:** repus a palavra de propósito num comentário e `golangci-lint` deu **0
  issues** (antes: `misspell: 1`). A regra foi testada com o caso que falhava, não aceite por confiança.
  **Medido:** 4 testes de retenção PASS · `./server` ok 44.664 s · `./internal/agent` ok 347.894 s.
- **E-1316** — persistência do store ligada ao `DataRoot`; builds abandonados marcados `FALHOU`.
- **E-1315** — botão de iniciar build; alvo bloqueado não ganha botão.
- **E-1314** — painel de estado; o 409 de `cancel` é resultado, não falha.

## `PodarDisco` — o que ainda falta

`PodarDisco` existe e está testado, mas **não está ligado a nenhum agendador**: só corre quando alguém o
chama. A retenção em disco funciona no caminho do `podar()` (a cada build novo), mas um servidor parado
com disco cheio não se limpa sozinho.

O que continua aberto: build **não assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o
painel de estado **não se actualiza sozinho**; `PodarDisco` sem agendador; `gitleaks`; instalação limpa
`BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1316

A cabeça do índice é **E-1316**.

- **E-1316** — **o store de builds do E-1313 tinha persistência MORTA.** `NovoBuildStoreComDirectorio` e
  `persistir()` existiam, e o único chamador da fábrica com diretório era **o próprio teste**. O
  `storeDeBuilds()` usava `NovoBuildStore()`, sem diretório ⇒ `persistir()` nunca escrevia nada.
  *Um store que existe e não persiste é um store que perde o trabalho quando o processo reinicia — e um
  reinício não é um caso raro: é a próxima vez que o utilizador abrir o produto.*
  **E o pior: declarei a limitação em três evidências como se fosse desenho.** Não era — era código
  escrito e não ligado. *Declarar não é corrigir.*
  Entregues `CarregarDoDisco()` e `Directorio()`; o `storeDeBuilds()` passou a usar
  `Runtime.DataRoot()/builds` (o getter já existia — não inventei caminho novo).
  **A decisão que o carregamento não pode errar: um registo `A_EXECUTAR` em disco NÃO está a correr.**
  O processo morreu. Passam a `FALHOU` com motivo explícito «interrompido».
  *Um build que o processo abandonou não está a correr: está interrompido, e o estado tem de o dizer.*
  **Ficheiro corrompido não derruba o arranque**, mas também não conta como build válido.
  **Medido:** `go build ./...` 0 · `./server` **ok 54 s** · `./internal/agent` **ok 382 s** · lint 0 ·
  4 testes novos PASS. **Sexta ocorrência do `misspell` a recusar «discriminante».**
- **E-1315** — botão de iniciar build; alvo bloqueado não ganha botão. 131 ficheiros / 944 testes.
- **E-1314** — painel de estado de build; o 409 de `cancel` é resultado, não falha.
- **E-1313** — `build.status`/`logs`/`artifacts`/`cancel` + `BuildStore`.

## Ciclo de build — estado real

| passo | backend | UI | persiste |
|---|---|---|---|
| planear | **existe** | **existe** | — |
| iniciar | **existe** | **existe (E-1315)** | **sim (E-1316)** |
| acompanhar | **existe** | **existe (E-1314)** | **sim (E-1316)** |
| logs / artefactos | **existe** | **existe (E-1314)** | **sim (E-1316)** |
| cancelar | **existe** | **existe (E-1314)** | **sim (E-1316)** |
| assinar | **AUSENTE** | — | — |
| publicar | **AUSENTE** | — | — |

O que continua aberto: build **não assinado nem instalado**; `sign.start`/`publish.*` inexistentes; o
painel de estado **não se actualiza sozinho**; **os ficheiros em disco não são podados** (o tecto de
200 é só em memória); `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1315

A cabeça do índice é **E-1315**.

- **E-1315** — **o `build.start` não tinha botão.** O E-1313 deu-lhe modo assíncrono e o E-1314 deu
  painel ao estado; entre os dois faltava o gesto que cria o build. *Um backend completo e um produto
  incompleto não são a mesma coisa, e a distância entre os dois é a interface.*
  Entregues `iniciarBuild()` em `buildStatusClient.ts` (`POST /build/start`) e, no `BuildPlanPanel`,
  uma coluna **Ações** com o botão por alvo pronto.
  **A decisão que um painel apressado erraria: alvo bloqueado NÃO ganha botão, nem desactivado.**
  *Um botão que nunca funciona ensina a ignorar botões* — e um desactivado sem explicação ensina o
  mesmo. Para um alvo bloqueado o painel mostra `—`, e a razão já está na coluna «O que falta».
  **O modo assíncrono é o padrão do botão.** *Um `build.start` síncrono é um pedido que só responde
  quando já não há nada a cancelar* — e o `cancel` só tem alvo enquanto o build corre.
  **O 412 é um resultado:** o painel diz **«O plano de «X» está incompleto»**, não «erro de rede».
  Mesma família da decisão do E-1314 sobre o 409.
  **Medido:** `tsc -b` 0 · `vitest` **131 ficheiros / 944 testes PASS** (era 130/939) · `eslint` 0
  errors · `vite build` 0 · bundle **PASS** (435 057 ≤ 600 000) · 5 testes novos (os 17 pré-existentes
  do painel continuam PASS).
- **E-1314** — painel de estado de build; 409 de `cancel` é resultado, não falha. 130/939 testes.
- **E-1313** — `build.status`/`logs`/`artifacts`/`cancel` + `BuildStore`.
- **E-1312** — CI verde (`d75b2ff16`). Instalação limpa `BLOCKED_EXTERNAL` com prova.
- **E-1311** — `ResumoDeBlockers` mentia; **uma correção de ~20 linhas fechou as 7 falhas de CI.**

## Ciclo de build no UI — fechado

| passo | backend | UI |
|---|---|---|
| planear | `GET /build/plan` | **existe** |
| **iniciar** | `POST /build/start` | **existe (E-1315)** |
| acompanhar | `GET /build/status/:id` | **existe (E-1314)** |
| logs / artefactos | `GET /build/logs`, `/artifacts` | **existe (E-1314)** |
| cancelar | `POST /build/cancel/:id` | **existe (E-1314)** |
| assinar | `sign.start` | **AUSENTE** |
| publicar | `publish.submit` | **AUSENTE** |

O que continua aberto: build iniciado **não é assinado nem instalado**; store **não sobrevive a
reinício**; `sign.start`/`publish.*` mutantes inexistentes; **o painel de estado não se actualiza
sozinho**; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1314

A cabeça do índice é **E-1314**.

- **E-1314** — **as cinco rotas do E-1313 não tinham cliente.** O servidor sabia responder
  `build.status`/`logs`/`artifacts`/`cancel`; o produto não perguntava. *Uma rota que existe e não é
  consumida é uma promessa do backend, não uma funcionalidade do produto.*
  Entregues `app/ui/app/src/lib/buildStatusClient.ts` e
  `app/ui/app/src/components/BuildStatusPanel.tsx`, montado em `CapabilitiesPage`.
  **A decisão que um painel ingénuo erraria:** o **409** de `cancel` é um **resultado**, não uma
  falha. Cancelar um build que já acabou devolve 409; mostrar «falhou ao cancelar» seria contar a
  história errada sobre um servidor que funcionou. O painel diz **«já tinha terminado — cancelar não
  tem efeito»**. Um erro de rede real continua a aparecer como erro.
  **O painel não calcula nada:** `nada_assinado`/`nada_publicado`/`nada_instalado` e `cancelavel` vêm
  do servidor; `EstadoDeBuild` é texto e não é remapeado.
  **Medido:** `tsc -b` 0 · `vitest` **130 ficheiros / 939 testes PASS** (era 129/931) · `eslint` 0
  errors (31 warnings pré-existentes) · `vite build` 0 · bundle **PASS** (434 737 ≤ 600 000) ·
  8 testes novos do painel.
- **E-1313** — `build.status`/`logs`/`artifacts`/`cancel` + `BuildStore`. `cancel` de terminado → 409;
  build cancelado **não** é des-cancelado. `./server` ok 49 s, `./internal/agent` ok 378 s.
- **E-1312** — CI verde (`d75b2ff16`: 21 success, só `gitleaks`). Instalação limpa
  `BLOCKED_EXTERNAL` com prova.
- **E-1311** — `ResumoDeBlockers` mentia: `alvos_bloqueados=3` com `blockers=[]`. **Uma correção de
  ~20 linhas fechou as 7 falhas de CI.**

## Cobertura do Release Hub

| superfície | backend | cliente (UI) |
|---|---|---|
| `toolchain.*`, `project.detect` | **existe** | **existe** |
| `build.plan` | **existe** | **existe** |
| `build.start` | **existe** (síncrono + assíncrono) | **AUSENTE** |
| **`build.status`/`logs`/`artifacts`/`cancel`** | **existe** | **existe (E-1314)** |
| `build.verify`, `build.smoke`, `artifact.*`, `sbom.generate` | **existe** | **existe** |
| `sign.plan`, `credentials.status`, `signing.*` | **existe** | **existe** |
| `sign.start`/`sign.status`/`sign.verify` | **AUSENTE** | — |
| `publish.preflight` | **existe** | **existe** |
| `publish.request_approval`/`submit`/`status`/`rollback` | **AUSENTE** | — |
| `provenance.attest`, `policy.evaluate` | **AUSENTE** | — |

O que continua aberto: **`build.start` não tem botão no UI** (o assíncrono muito menos); o store não
sobrevive a reinício; nada assinado/publicado/instalado; `sign.start`/`publish.submit`/`rollback`
inexistentes; `gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1313

A cabeça do índice é **E-1313**.

- **E-1313** — **as quatro rotas que faltavam, e o store que nenhuma delas pode dispensar.** O
  doc-comment do `build.verify` dizia *«não existe registo de builds»*. Entregues
  `internal/agent/build_store.go` (`BuildStore`, `RegistoDeBuild`, 5 estados, cancelamento por
  contexto, tectos de saída e de registos) e **5 rotas**: `GET /build/status/:id`,
  `GET /build/logs/:id`, `GET /build/artifacts/:id`, `GET /build/registos`,
  `POST /build/cancel/:id`. O `build.start` passou a **registar** e a devolver `build_id`, com modo
  `assincrono` (**202**, corre em segundo plano). As duas promessas travadas: `cancel` sobre build
  terminado → **409**; e um build cancelado **não** é des-cancelado quando o processo morre depois.
  *Um cancelamento que o resultado desfaz não é um cancelamento: é uma sugestão.*
  **O modo assíncrono não é extra:** se o `start` só responde no fim, o `cancel` só encontra builds
  terminados e o 409 recusa todos.
  **Medido:** `go build ./...` 0 · `./server` **ok 49 s** · `./internal/agent` **ok 378 s** · lint 0 ·
  17+10+2 testes PASS. Ciclo real: `start`→200 com `build_id`; `status`→200 `CONCLUIDO`;
  assíncrono→202 `A_EXECUTAR` e termina; `cancel` de terminado→409.
  **Duas ocorrências aqui:** `itoa` colidiu com o `itoa` de `omnichannel_dedupe_test.go`; e o
  **`misspell` recusou «discriminante»** — **quinta** vez nesta sessão.
- **E-1312** — CI verde (`d75b2ff16`: 21 success, só `gitleaks` vermelho). Instalação limpa
  `BLOCKED_EXTERNAL` com prova: `.iss` usa `{autopf}` → exige UAC; ficou pendurado 600 s sem log;
  **nada foi instalado** (sem directório, sem registo, zero processos).
- **E-1311** — `ResumoDeBlockers` mentia: `alvos_bloqueados=3` com `blockers=[]`. Segunda ocorrência
  do mesmo defeito. **Uma correção de ~20 linhas fechou as 7 falhas de CI.**
- **E-1310** — `ProviderDeInstaladorWindows`; `HadesSetup-teste.exe` 3 363 167 B, **`NotSigned`**.

## Cobertura do Release Hub

| superfície | estado |
|---|---|
| `toolchain.*` | **existe** |
| `project.detect`, `build.plan`, `build.start` | **existe** |
| **`build.status`/`logs`/`artifacts`/`cancel`** | **existe (E-1313)** |
| `build.verify`, `build.smoke`, `artifact.*`, `sbom.generate` | **existe** |
| `sign.plan`, `credentials.status`, `signing.*` | **existe** |
| `sign.start`/`sign.status`/`sign.verify` | **AUSENTE** |
| `publish.preflight` | **existe** |
| `publish.request_approval`/`submit`/`status`/`rollback` | **AUSENTE** (por desenho: o provider só tem `Preflight`) |
| `provenance.attest`, `policy.evaluate` | **AUSENTE** |

O que continua aberto: o store **não sobrevive a reinício**; nenhum artefacto assinado nem instalado;
`sign.start`/`publish.submit` inexistentes; **nenhum painel do UI consome as rotas novas**;
`gitleaks`; instalação limpa `BLOCKED_EXTERNAL`; **FASE 30 não iniciada**.

---
## JANELA E-1312

A cabeça do índice é **E-1312**.

- **E-1312** — **o CI do E-1311 está VERDE.** `d75b2ff16` fechou com **21 success e 2 failure**, e as
  duas são o `gitleaks (historico do PR)` — a dívida conhecida do proprietário. Os três gates que
  estavam vermelhos passaram: **`Go agentic and server gates`**, **`PR gate`** e
  **`Go agentic Windows portability`**. **A comparação mede a correção:** no commit anterior
  (`f86bcce8e`) eram **7 falhas**, incluindo `test (macos-latest)`, `test (ubuntu-latest)` e
  `race (ubuntu-latest)` — o `TestRotaPlanoBlockersEMotivosNoTopo` partia o `./server` nas **três
  plataformas**, e o `PR gate` reprovava por consequência. Uma correção de ~20 linhas no
  `ResumoDeBlockers` fechou sete.
- **A INSTALAÇÃO LIMPA ficou `BLOCKED_EXTERNAL`, com prova.** O `.iss` de teste usa
  `DefaultDirName={autopf}` → `C:\Program Files`, que exige elevação. O instalador com `/VERYSILENT`
  ficou **pendurado 600 s**, sem log e sem exit code: à espera de um UAC que não aparece em sessão
  não-interativa. **O que isso NÃO deixou:** nada — `C:\Program Files\HADESTeste` não existe,
  `%LOCALAPPDATA%\HADESTeste` não existe, registo `*HADES*` vazio, zero processos. A tentativa foi
  limpa. **Desbloqueio exato:** VM Windows descartável + `/VERYSILENT /LOG=...`, depois desinstalar e
  verificar resíduos — ação do proprietário, não do agente.
- **E-1311** — `ResumoDeBlockers` mentia: `alvos_bloqueados=3` com `blockers=[]`. Segunda ocorrência
  do mesmo defeito (o E-1307 corrigiu-o no provider e deixou-o no resumo). Corrigido com
  `MotivosDeNaoPronto()` + prefixo do alvo. `internal/agent` ok 344 s, `server` ok 44 s.
- **E-1310** — `ProviderDeInstaladorWindows`: `windows-installer` → 200, `HadesSetup-teste.exe`
  3 363 167 B, `'Inno Setup'` presente, **`Status = NotSigned`**.
- **E-1309** — o artefacto **corre** no alvo: `HADES-ARTEFACTO-OK so=windows arch=amd64`; ELF e
  Mach-O **não** produziram a marca.

## Estado por alvo

| alvo | build | formato | assinatura | instalação |
|---|---|---|---|---|
| `windows` | **200** | PE `.exe` | — | **EXECUTADO** |
| `linux` | **200** | ELF | — | **NÃO EXECUTADO** (sem host Linux) |
| `macos` | **200** | Mach-O | — | **NÃO EXECUTADO** (sem host macOS) |
| `windows-installer` | **200** | Inno Setup `.exe` | **NotSigned** (por desenho) | **`BLOCKED_EXTERNAL`** |

O que continua aberto: a instalação limpa (`BLOCKED_EXTERNAL`), nenhum artefacto assinado (por
desenho), `build.status`/`logs`/`cancel`/`artifacts` inexistentes, `gitleaks (historico do PR)`,
build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a **FASE 30 não
iniciada**.

---
## JANELA E-1311

A cabeça do índice é **E-1311**.

- **E-1311** — **o CI do E-1309 estava VERMELHO e o defeito era meu.** `0666cca86` fechou com
  `Go agentic and server gates = failure` e `PR gate = failure` (o `PR gate` falha *porque* o gate
  profundo falhou). Log: `--- FAIL: TestRotaPlanoBlockersEMotivosNoTopo` /
  `plano tem 3 alvo(s) bloqueado(s) mas 'blockers' esta' vazio`. **Causa:** o E-1308 tirou o
  `inno-setup` do alvo `windows`, os três alvos ficaram `PRESENT_COMPATIBLE_UNVERIFIED`, e
  `ResumoDeBlockers()` lia `.Blockers` (que só tem ausências) enquanto `AlvosBloqueados` conta
  `Pronto()==false`. Medido: `alvos_bloqueados=3`, `blockers=[]` — **o painel mostrava «3 bloqueados»
  e nenhum blocker**.
  **É a SEGUNDA ocorrência do mesmo defeito** — o E-1307 corrigiu-o no `ProviderDeBuildGo` e deixou-o
  vivo no resumo do projecto. *Uma correção num sítio não é uma correção no sistema.*
  Corrigido: `ResumoDeBlockers()` usa `MotivosDeNaoPronto()` e **prefixa o alvo** (o dedupe global
  colapsava o `go` de `windows` e `linux` numa linha, apagando a contagem).
  **`go test ./internal/agent` ok (344 s)**, **`go test ./server` ok (44 s)**, lint 0.
  **O processo falhou, não só o código:** depois do E-1308 corri `-run TestInicioDeBuild` em vez da
  suíte completa do `server`. *Um `-run` que escolhe o teste que eu escrevi é um teste que confirma o
  que eu já sabia.*
- **E-1310** — `ProviderDeInstaladorWindows`: `windows-installer` → **200**,
  `HadesSetup-teste.exe` 3 363 167 B, `'Inno Setup'` no binário, **`Status = NotSigned`**.
- **E-1309** — o artefacto **corre** no alvo: `HADES-ARTEFACTO-OK so=windows arch=amd64`; ELF e
  Mach-O **não** produziram a marca.
- **E-1308** — o alvo `windows` prometia um instalador que nenhum provider produzia.
- **E-1307** — o `build.start` passou a construir. Quatro defeitos.

O que continua aberto: nenhum instalador foi **instalado nem executado**; `build.status`/`logs`/
`cancel`/`artifacts` não existem; **nenhum artefacto assinado** (por desenho); `gitleaks (historico do
PR)`; build Android e arranque do app (`BLOCKED_EXTERNAL`); `INSTALLED=NAO` (E-1252); **FASE 30 não
iniciada**.

---
## JANELA E-1310

A cabeça do índice é **E-1310**.

- **E-1310** — o quarto alvo deixou de recusar: existe o **`ProviderDeInstaladorWindows`**. O E-1308
  separou `windows` de `windows-installer` porque o provider de binários nunca invocaria o Inno Setup;
  o alvo ficou a responder **412**. Agora o `build.start` escolhe o provider **pelo alvo**.
  **Medido:** `windows-installer` → **200**, **`HadesSetup-teste.exe` 3 363 167 B**, magic `4D 5A`,
  string `Inno Setup` dentro do binário, e `Get-AuthenticodeSignature` →
  **`Status = NotSigned`**, `SignerCert = (nenhum)` — **não assinado, que é o requisito da RC**.
  **8 testes PASS**, lint 0. O provider **não adivinha o nome** do artefacto (lista o destino antes e
  depois e mede o que apareceu), **não apaga o `dist`**, e **recusa** com dois `.iss` em vez de
  escolher por nós.
- **E-1309** — o artefacto **corre** no alvo: binário Windows executado devolveu
  `HADES-ARTEFACTO-OK so=windows arch=amd64`; ELF e Mach-O **não** produziram a marca (negativo
  discriminante). O discriminante é a **marca na saída**, não o exit code.
- **E-1308** — o alvo `windows` prometia um instalador que nenhum provider produzia.
- **E-1307** — o `build.start` passou a construir. Quatro defeitos: mensagem `0 blocker(s)`, espera
  circular, `GOOS` do host, catálogo sem `macos`.
- **E-1306** — `POST /build/start`, a primeira rota do Release Hub que executa.

## Estado por alvo

| alvo | build | formato | execução no alvo |
|---|---|---|---|
| `windows` | **200** | PE `.exe` | **EXECUTADO** — `so=windows arch=amd64` |
| `linux` | **200** | ELF | **NÃO EXECUTADO** (sem host Linux) |
| `macos` | **200** | Mach-O | **NÃO EXECUTADO** (sem host macOS) |
| `windows-installer` | **200** | Inno Setup `.exe` | **NÃO INSTALADO** — não assinado, como a RC exige |

O que continua aberto: **nenhum instalador foi instalado nem executado**; o `.iss` usado é de
**teste** (`HADES Teste`), não o `app/ollama.iss` canónico; `build.status`/`logs`/`cancel`/`artifacts`
não existem; **nenhum artefacto assinado** (por desenho); `gitleaks (historico do PR)`; build Android e
arranque do app (`BLOCKED_EXTERNAL`); `INSTALLED=NAO` (E-1252); **FASE 30 não iniciada**.

---
## JANELA E-1309

A cabeça do índice é **E-1309**.

- **E-1309** — o elo que os três builds verdes ainda escondiam: **o artefacto corre no sistema alvo?**
  Provado para o único alvo que este host pode executar. O binário **Windows** produzido pelo
  `build.start` foi executado e respondeu **`HADES-ARTEFACTO-OK so=windows arch=amd64`** (exit 0) —
  identifica-se como Windows/amd64 **ao correr**, o que é mais forte do que o `MZ` no cabeçalho.
  **Negativo discriminante:** os binários `linux` (ELF) e `macos` (Mach-O) foram tentados no mesmo
  host e **não produziram a marca**. O discriminante é a **marca na saída**, não o exit code — o
  `exit=0` das tentativas falhadas é do PowerShell, e contá-lo teria dado **três verdes falsos**.
- **E-1308** — o alvo `windows` prometia um instalador que nenhum provider produz. Separado em
  `windows` (binário) e `windows-installer` (recusa honesta).
- **E-1307** — o `build.start` passou a construir. Quatro defeitos: mensagem `0 blocker(s)`, espera
  circular, `GOOS` do host, catálogo sem `macos`.
- **E-1306** — `POST /build/start`, a primeira rota do Release Hub que executa. 8 testes PASS.
- **E-1305** — o `misspell` leu *Resolvemos* como erro de inglês. **Verde confirmado em `aea7076be`.**

## Estado por alvo — a distinção que o adendo exige

| alvo | build | formato | execução no alvo |
|---|---|---|---|
| `windows` | **200** | PE `.exe` | **EXECUTADO** — `so=windows arch=amd64` |
| `linux` | **200** | ELF | **NÃO EXECUTADO** (sem host Linux) |
| `macos` | **200** | Mach-O | **NÃO EXECUTADO** (sem host macOS) |
| `windows-installer` | **412** | — | sem provider |

O que continua aberto: o instalador Windows não existe, `build.status`/`logs`/`cancel`/`artifacts` não
existem, **nenhum artefacto foi assinado**, `gitleaks (historico do PR)` (decisão do proprietário),
build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a **FASE 30 não
iniciada**.

---
## JANELA E-1308

A cabeça do índice é **E-1308**.

- **E-1308** — o alvo `windows` prometia um **instalador** que nenhum provider produz. O catálogo dizia
  *«Instalador .exe não assinado (Inno Setup)»* e exigia `inno-setup` + `powershell`; o único provider
  corre `go build`. **Medido** com `ISCC.exe` (1 456 272 B) e PowerShell no PATH: **200** e
  `build-alvo-windows.exe` (2 458 624 B, `MZ`) — e **instaladores: 0**. Separado em `windows` (binário,
  só `go`) e `windows-installer` (instalador declarado, `SEM PROVIDER: o build.start recusa`).
  **Os três alvos de binário constroem com a plataforma certa:** `linux`→ELF, `macos`→Mach-O,
  `windows`→PE. `go test ./internal/agent` **ok (361 s)**, lint 0.
- **E-1307** — o `build.start` passou a **construir**. Quatro defeitos: mensagem dizia `0 blocker(s)`;
  **espera circular** (`Pronto()` exigia o health que o próprio `go build` produz); `GOOS` vinha do
  host (artefacto `-linux` com bytes `MZ`); catálogo **sem o alvo `macos`** que o provider prometia.
- **E-1306** — `POST /build/start`: a primeira rota do Release Hub que executa. 8 testes PASS.
- **E-1305** — o `misspell` leu *Resolvemos* como erro de inglês. **Verde em `aea7076be`**:
  `test (windows-latest)` success, `test (ubuntu-latest)` success.
- **E-1304** — o E2E do `build.plan` nunca tinha corrido contra o backend real (faltava `HADES_E2E_REPO`).

O que continua aberto: **o instalador Windows não existe** (`windows-installer` sem provider e o
`ISCC.exe` fora do PATH do servidor), `build.status`/`logs`/`cancel`/`artifacts` não existem,
**nenhum artefacto foi assinado, instalado ou executado no sistema alvo**, `gitleaks (historico do PR)`
(decisão do proprietário), build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO`
(E-1252) e a **FASE 30 não iniciada**.

---
## JANELA E-1307

A cabeça do índice é **E-1307**.

- **E-1307** — o `build.start` do E-1306 passou a **construir de verdade**. Quatro defeitos encontrados
  ao executá-lo contra o backend real: (1) a mensagem dizia `0 blocker(s): ` quando o plano não estava
  pronto por uma ferramenta `PRESENT_COMPATIBLE_UNVERIFIED`; (2) **espera circular** — o `Construir`
  exigia `plano.Pronto()`, cujo health funcional para o `go` **é o próprio `go build`**; (3) `GOOS`
  vinha do host, não do alvo: pedir `linux` produzia um ficheiro chamado `build-alvo-linux` com bytes
  `4D 5A 90 00` (Windows PE); (4) o catálogo **não tinha o alvo `macos`** que o provider prometia.
  Medido: `linux` → **ELF** (`7F 45 4C 46`), `macos` → **Mach-O** (`CF FA ED FE`), `windows` → 412
  honesto (`inno-setup` ABSENT). `golangci-lint` 0 issues, testes **ok**.
- **E-1306** — `POST /build/start`: a primeira rota do Release Hub que **executa**. O `ProviderDeBuildGo`
  existia desde o E-1261 e nenhuma rota o expunha. 8 testes PASS.
- **E-1305** — o E-1303 corrigiu o macOS e partiu windows/ubuntu: `misspell` leu *Resolvemos* como erro.
  **Verde confirmado em `aea7076be`:** `test (windows-latest)` success, `test (ubuntu-latest)` success.
- **E-1304** — o E2E do `build.plan` nunca tinha corrido contra o backend real (faltava `HADES_E2E_REPO`).
- **E-1303** — `/var`→`/private/var`: `filepath.Clean` não resolve symlinks.

O que continua aberto: **o alvo `windows` não constrói** (`inno-setup` ABSENT — o alvo produz
instalador), `build.status`/`logs`/`cancel`/`artifacts` não existem, **nenhum artefacto foi assinado,
instalado ou executado no sistema alvo**, `gitleaks (historico do PR)` (decisão do proprietário),
build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a **FASE 30 não
iniciada**.

---
## JANELA E-1306

A cabeça do índice é **E-1306**.

- **E-1306** — `POST /build/start`: a primeira rota do Release Hub que **executa** alguma coisa. O
  `ProviderDeBuildGo` existia desde o E-1261 e **nenhuma rota o expunha**. As defesas já estavam no
  provider (plano pronto, alvo no catálogo, raiz absoluta com `go.mod`, timeout 10 min, saída truncada,
  `dry_run`). A rota acrescenta validação do corpo, plano vindo do `ResolverDeToolchain` e aviso de
  fronteira no corpo. **8 testes PASS**, `golangci-lint ./server` 0 issues. Contra o binário real:
  412 (raiz relativa/vazia), 400 (timeout acima do tecto / negativo / corpo inválido), 200 (`dry_run`).
  **O caminho de sucesso de uma build real NÃO foi executado.**
- **E-1305** — o E-1303 corrigiu o macOS e partiu windows/ubuntu: o `misspell` leu *Resolvemos* como
  erro de inglês. Corrigido; `golangci-lint` 0 issues nos três pacotes.
- **E-1304** — o E2E do `build.plan` nunca tinha corrido contra o backend real: sem `HADES_E2E_REPO`
  3 passed / 4 failed; com ela 7 passed. O spec passou a falhar alto sem a variável.
- **E-1303** — `/var`→`/private/var`: `filepath.Clean` não resolve symlinks.
- **E-1302** — `publish.preflight` contra o binário Go real: 9 passed no modo real, 9 no fixture.

O que continua aberto: **caminho de sucesso de `build.start` não provado**,
`build.status`/`logs`/`cancel`/`artifacts` não existem (falta store), `gitleaks (historico do PR)`
(decisão do proprietário), build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO`
(E-1252) e a **FASE 30 não iniciada**.

---
## JANELA E-1305

A cabeça do índice é **E-1305**.

- **E-1305** — o E-1303 corrigiu o macOS mas partiu o **Windows e o Ubuntu**: o `misspell` do
  `golangci-lint` leu a palavra portuguesa *Resolvemos* num comentário como erro de inglês. Medido:
  `f5fd1af7a` windows/ubuntu **success**; `63ea97085` **failure** em cada. Corrigido removendo a
  palavra; `golangci-lint run ./server/` **0 issues**, `./internal/agent/` e `./internal/multillm/`
  **0 issues**. **O gate local existia e não foi invocado.**
- **E-1304** — o E2E do `build.plan` nunca tinha corrido contra o binário Go real: sem
  `HADES_E2E_REPO` dava 3 passed / **4 failed**; com ela **7 passed**. O spec passou a falhar alto sem
  a variável.
- **E-1303** — `/var`→`/private/var` no macOS: `filepath.Clean` não resolve symlinks. Corrigido com
  `EvalSymlinks` nos dois lados.
- **E-1302** — E2E do `publish.preflight` contra o backend Go real: 9 passed no modo real, 9 no
  fixture, suíte 136/11.
- **E-1301/E-1300** — `Preserve Class A+ surfaces` verde; a causa era o checker de `.ps1` a exigir
  `powershell.exe` no Ubuntu.

O que continua aberto: `build.start`/`status`/`logs`/`cancel`/`artifacts` não existem,
`gitleaks (historico do PR)` (decisão do proprietário), build Android e arranque do app
(`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a **FASE 30 não iniciada**.

---
## JANELA E-1304

A cabeça do índice é **E-1304**.

- **E-1304** — o E2E do `build.plan` **tinha** modo `BACKEND_REAL_AGENT_RUNTIME` mas nunca foi
  executado contra o binário Go real. Executado: **3 passed / 4 failed / 2 skipped**. A causa não era o
  produto: faltava `HADES_E2E_REPO`, e o spec caía em silêncio para `/repo-fixture`. Com a variável:
  **7 passed / 2 skipped**. O spec passa a **falhar alto** sem ela.
- **E-1303** — `test`/`race` do macOS vermelhos por `filepath.Clean` não resolver symlinks
  (`/var`→`/private/var`); corrigido com `EvalSymlinks` nos dois lados.
- **E-1302** — E2E do `publish.preflight` contra o binário Go real: seis canais a 200, 400/412 nos
  negativos; **9 passed** no modo real e **9** no fixture.
- **E-1301/E-1300** — `Preserve Class A+ surfaces` **verde**; a causa era o checker de `.ps1` a exigir
  `powershell.exe` no Ubuntu.
- **E-1299/E-1298/E-1289** — cadeia `publish.preflight` completa.

O que continua aberto: **`build.start`/`status`/`logs`/`cancel`/`artifacts` não existem** (o adendo
pede-os), `gitleaks (historico do PR)` (decisão do proprietário), build Android e arranque do app
(`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a **FASE 30 não iniciada**.

---
## JANELA E-1303

A cabeça do índice é **E-1303**.

- **E-1303** — `test (macos-latest)` e `race (macos-latest)` ficaram **vermelhos**: o meu teste do
  E-1286 (`TestDeteccaoNaoUsaARaizDoPedido`) comparava o caminho com `filepath.Clean`, que **não
  resolve symlinks**. No macOS `/var` é symlink de `/private/var`, e o teste chamava diferentes a dois
  nomes do mesmo sítio. Corrigido com `filepath.EvalSymlinks` nos **dois** lados.
- **E-1302** — E2E do `publish.preflight` contra o **binário Go real**: seis canais a 200, canal
  inválido a 400, artefacto ausente a 412; Playwright **9 passed** em `BACKEND_REAL_AGENT_RUNTIME` e
  **9 passed** em `UI_FIXTURE`; suíte completa **136 passed / 11 skipped**.
- **E-1301** — `Preserve Class A+ surfaces` **verde** (era falha em cinco commits).
- **E-1300** — a causa era o checker de `.ps1` a exigir `powershell.exe` no Ubuntu; corrigido com
  `exit 3` declarado + `skipUnless` com motivo + três ramos explícitos, **sem** `|| true`.
- **E-1299/E-1298/E-1289** — cadeia `publish.preflight` completa (painel, cliente, rota).

O que continua aberto: `gitleaks (historico do PR)` (decisão do proprietário), `BuildProvider` sem
cliente/E2E, build Android e arranque do app (`BLOCKED_EXTERNAL`), `INSTALLED=NAO` (E-1252) e a
**FASE 30 não iniciada**.

---
## JANELA E-1302

Estado em `026ca7bbd` + este commit. A cabeça do índice é **E-1302**.

- **E-1302** — o E2E do painel de `publish.preflight` passou a correr contra o **binário Go real**
  (`MODO=BACKEND_REAL_AGENT_RUNTIME`): os seis canais devolvem 200, `LOCAL_ONLY` fica `pronto=true`
  e `publico=false`, os cinco canais públicos ficam `pronto=false`/`requer_aprovacao=true`, e em todos
  `somente_leitura=true`/`nada_publicado=true`/`publicado=false`. O E2E revelou **três defeitos nos
  meus próprios testes** (condição `aprovacao_humana` inexistente, contagem 2 em vez de 3, caixa fixa
  nas frases) — corrigidos por modo, sem baixar a exigência.
- **E-1301** — o check-run `Preserve Class A+ surfaces` ficou **VERDE** em `f7b5e5eea`; era `failure`
  em cinco commits. As falhas do SHA passaram de 3 para 1 (só `gitleaks (historico do PR)`).
- **E-1300** — a causa: `hades_ps1_syntax_check_test.py` exigia `powershell.exe`, que **não existe no
  runner Ubuntu** (8 erros `FileNotFoundError`). Corrigido com `exit 3`
  (`BLOCKED_BY_EXTERNAL_DEPENDENCY`) + `skipUnless` com motivo escrito + workflow com três ramos
  explícitos — **sem** `|| true` e **sem** `continue-on-error`.
- **E-1299/E-1298/E-1289** — a cadeia do `publish.preflight`: painel, cliente e rota. FASE 28.8 fechada
  ponta a ponta.
- **E-1290** — o build de RELEASE Android deixou de usar `signingConfigs.debug`.

Próxima ação: `BuildProvider` tem painel (`BuildPlanPanel`) mas **não tem cliente nem E2E** — é a
mesma lacuna que o E-1302 acabou de fechar para `publish.preflight`.

---
## JANELA E-1289

**Ronda 107. O `PublishingProvider` ganhou porta — e o inventário foi corrigido.**

- **E-1289** — `server/agent_publish_preflight_route.go` + `_test.go`: `POST /publish/preflight/:canal`,
  **13 PASS** (8 testes + 5 subtestes).
- **CORREÇÃO DE INVENTÁRIO:** o resumo do projeto dizia que `CapabilityResolver` e
  `PublishingProvider` estavam **ausentes**. **Falso nos dois.** O `CapabilityResolver` **existe como
  `ResolverDeToolchain`** (`toolchain_plan.go`, 17 143 B) com `CapabilityPlan`, 6 `ToolState`, 10
  alvos, `ArgsDeVersao` e **3 rotas**; a UI **já o consome** (`toolchainClient.ts`, 20+ exportações,
  `CapabilitiesPage`, `InstallProfilesPanel`, `install-profiles.spec.ts` com 21 referências). E
  `PlanoDeInstalação` existe com 2 rotas.
  *Um inventário que procura nomes inventados declara ausente o que existe com outro nome* — 13.ª
  ocorrência do defeito, e a primeira em que o erro era **a favor** do projeto e mesmo assim foi
  corrigido.
- **A lacuna real:** o `PublishingProvider` tinha 6 canais e 9 testes e **ZERO rotas**. A rota é
  `publish.preflight`, **não** `publish.submit`: o provider não tem `Publicar()`.
- **Caminhos vêm do ambiente:** `HADES_ARTEFACTO_A_PUBLICAR` e `HADES_CREDENCIAL_DE_PUBLICACAO` —
  variáveis próprias, distintas das outras quatro fronteiras. O pedido só escolhe o CANAL.
- **Garantias medidas:** `publicado` **sempre** false e `nada_publicado` **sempre** true, em campos
  separados; **canal público NUNCA fica `pronto`** — os 5 testados; canal desconhecido recusado com
  **400 antes de tocar no filesystem**; **canário de vazamento** `CANARIO-QUE-NAO-PODE-VAZAR-12345`
  ausente do detalhe e do corpo serializado ("presente (N bytes); conteúdo NAO lido").

Medido: **13 PASS** · `go test ./server/` ok · `go build ./...` 0 · `go vet` 0 ·
`golangci-lint ./server` 0 · `gofumpt` limpo.

**Lacuna:** falta cliente + painel + E2E do preflight. `publish.submit` é deliberadamente inexistente.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1288

**Ronda 106. O E2E do `ProjectDetectPanel`: o nono elo fechado no browser real.**

- **E-1288** — `e2e/project-detect-panel.spec.ts` + rota `project/detect` no fixture:
  **10 passed / 1 skipped**.
- **O teste que só o browser faz aqui:** o fixture devolve as stacks `go`/`node` e os gestores
  `go-modules`/`npm` — os valores REAIS do backend — e o spec exige os rótulos traduzidos **"Go",
  "Node", "Go modules", "npm"**. Se o `switch` tivesse inventado `golang`/`nodejs`, o teste unitário
  passaria (usaria o mesmo valor inventado) e **o E2E falharia**.
- **Não tem campo de raiz, provado no browser:** `input` `toHaveCount(0)` e
  `Object.keys(postDataJSON()) === []`, com ausência de `raiz`/`caminho`/`path`/`directorio`/`root`.
- **Não existe botão de instalar, construir, assinar, publicar nem build:** um único botão, cinco
  rótulos proibidos. **A prova não promete toolchain:** exige "NÃO prova que a toolchain esteja
  instalada" e `toolchain.doctor`.
- **No modo de backend real a asserção é acionável:** sem `HADES_RAIZ_PROJECTO` o servidor recusa com
  412 e o spec exige o erro **nomeando a variável**, sem relatório. Não fabrica resultado.

Medido: **10 passed / 1 skipped** · `npx playwright test` **127 passed / 11 skipped** (era 117/10) ·
`npx vitest run` 128 ficheiros / 921 testes · `tsc -b` 0 · `eslint .` 0 erros · `vite build` 0 ·
`BUNDLE_BUDGET=PASS`.

**Lacuna:** falta E2E com raiz REAL. O detector **não alimenta** o `BuildProvider`.
**2 providers sem rota nem painel** (`CapabilityResolver`, `PublishingProvider`).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1287

**Ronda 105. O oitavo provider fechado nos três elos: `ProjectDetectPanel` consome `project.detect`.**

- **E-1287** — `projectDetectClient.ts` + `ProjectDetectPanel.tsx` + **12 testes**, montado em
  `CapabilitiesPage`. A E-1286 deu rota ao `ProjectDetector`, mas **nenhum consumidor na UI**: a rota
  era inalcançável a partir do produto.
- **Os valores foram lidos antes de escrito, 12.ª vez:** stacks `go`/`node`/`expo`/`python`/`rust`/
  `dotnet`/`flutter` e gestores `go-modules`/`npm`/`pnpm`/`yarn`/`bun`/`poetry`/`uv`/`cargo`/`nuget`/
  `pub` — **todos extraídos de `project_detector.go`**, nenhum inventado.
- **Não tem campo de raiz:** `input` `toHaveLength(0)` e corpo do pedido `{}`. Um único botão, com
  instalar/construir/assinar/publicar/build proibidos.
- **A prova não promete toolchain:** o teste exige "NÃO prova que a toolchain esteja instalada" e
  `toolchain.doctor`. E quando há alvos mas **nenhum gestor**, o painel diz "NENHUM gestor de pacotes
  reconhecido" em vez de mostrar uma lista que parece boa.
- **Os avisos do detector são mostrados:** `avisos` tem `omitempty`; o painel mostra-os quando existem
  e não inventa um estado quando não.

Medido: **12 testes PASS** · `npx vitest run` **128 ficheiros / 921 testes** (era 127/909) ·
`tsc -b` 0 · `eslint .` 0 erros · `vite build` 0 · `BUNDLE_BUDGET=PASS` ·
`npx playwright test` **117 passed / 10 skipped**.

**Lacuna:** sem E2E do painel. O detector **não alimenta** o `BuildProvider`.
**2 providers sem rota nem painel** (`CapabilityResolver`, `PublishingProvider`).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1286

**Ronda 104. O `ProjectDetector` ganhou porta: `POST /project/detect`.**

- **E-1286** — `server/agent_project_detect_route.go` + **8 testes**. O detector existe desde a E-1245
  com 12 KB e teste próprio, mas **nenhuma rota**: só o programador o podia invocar.
- **A fronteira, e o teste que a prova:** a raiz vem de `HADES_RAIZ_PROJECTO`, **nunca do pedido**.
  `TestDeteccaoNaoUsaARaizDoPedido` manda no corpo `/caminho/inventado/pelo/cliente` e exige **200 com
  a raiz do ambiente** — falha se a rota tiver obedecido.
- **Uma suposição minha desmentida pelo teste:** escrevi o teste a esperar **200 com listas vazias**; o
  detector devolve **409** com *"nenhuma stack reconhecida"*. **O contrato real é melhor que a minha
  suposição:** *uma lista vazia e um erro não são a mesma coisa: quem lê um relatório vazio acredita
  que o projeto não tem stack; quem lê "nenhuma stack reconhecida" sabe que o detector não soube
  classificar.*
- **Segundo sintoma mal lido:** o teste seguinte exigia o caminho completo no erro; o caminho **estava**
  lá, mas com barras escapadas (`\\`). `filepath.Base` resolveu. *O sintoma parecia falta de informação
  e era o escape do JSON.*
- **O aviso declara que os alvos são SUGESTÃO:** o teste exige `SUGESTAO` e `toolchain.doctor`. *Um
  alvo derivado de um ficheiro não prova que a toolchain exista.*

Medido: **8 testes PASS** · `go build ./...` 0 · `go vet` 0 · `golangci-lint` **0** · `gofumpt` limpo ·
`npx vitest run` 127 ficheiros / 909 testes · `tsc -b` 0.

**Lacuna:** sem cliente, sem painel e sem E2E — o `ProjectDetector` continua a não ser consumido pelo
produto. **2 providers sem rota nem painel** (`CapabilityResolver`, `PublishingProvider`).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1285

**Ronda 103. O E2E do `VerificationPanel`: o oitavo elo fechado no browser real.**

- **E-1285** — `e2e/verification-panel.spec.ts` + rota `build/verify` no fixture: **11 passed / 1 skipped**.
- **O teste que só o browser consegue fazer:** o fixture devolve `sem_assinatura: 1` e
  `nao_verificadas: 1` — **o mesmo valor nos dois**. Se o painel os somasse, um campo leria **2**. O E2E
  lê cada `data-testid` e exige **1 e 1**. Um teste unitário que constrói o próprio objecto não
  apanharia isto; **o browser apanha porque lê o DOM que o painel realmente pintou**.
- **Não existe botão de assinar, construir, instalar ou publicar:** um único botão, quatro rótulos
  proibidos, `input` `toHaveCount(0)`, `Object.keys(corpo) === ["artefactos"]`, `sha256` casado com
  `/^[a-f0-9]{64}$/`.
- **O resumo não pode dizer "assinado":** o E2E exige `not.toContainText(/assinad/i)` e a frase "NAO
  foi/foram inspeccionado".
- **No modo de backend real a asserção é disjunta** (`erro` OU `relatorio`) — os caminhos de
  demonstração não existem no router de produção, e o teste **não fabrica** um resultado para fingir
  que passou.

Medido: **11 passed / 1 skipped** · `npx playwright test` **117 passed / 10 skipped** (era 106/9) ·
`npx vitest run` 127 ficheiros / 909 testes · `tsc -b` 0 · `eslint .` 0 erros · `vite build` 0 ·
`BUNDLE_BUDGET=PASS`.

**Lacuna:** falta ligação ao `BuildProvider` para verificar artefactos REAIS. `BuildPlanPanel` sem E2E.
**3 providers sem rota nem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1284

**Ronda 102. O sétimo provider fechado: `VerificationPanel` consome `build.verify`.**

- **E-1284** — `verificationClient.ts` + `VerificationPanel.tsx` + **12 testes**, montado em
  `CapabilitiesPage`. A rota existia desde E-1256 mas **nenhum consumidor na UI**.
- **A distinção que o painel não pode perder, e há teste que a trava:** `sem_assinatura` (o adaptador
  olhou, não havia) e `nao_verificadas` (ninguém olhou) em `data-testid` **separados**, proibido somar.
  Se fossem somados, o painel diria "N com assinatura presente" para artefactos que ninguém inspecionou.
- **Inventado-vs-actual, apanhado antes de testar, 11.ª vez:** escrevi `v.integridade`; o campo é
  `integros`. Corrigido lendo `build_provider.go`. E o `eslint` apanhou `SHA_A` declarada e nunca usada
  — **o gate apanhou o meu próprio teste**.
- **`aprovada: true` não é prova de assinatura:** o teste exige "nada foi assinado" e "não é prova de
  assinatura".
- **Não tem campo nem botão de assinar/construir/instalar:** `input` `toHaveLength(0)`, um botão,
  `Object.keys(corpo) === ["artefactos"]`. Caminhos de demonstração declarados como tal.

Medido: **12 testes PASS** · `npx vitest run` **127 ficheiros / 909 testes** (era 126/897) ·
`tsc -b` 0 · `eslint .` **0 erros** · `vite build` 0 · `BUNDLE_BUDGET=PASS` ·
`npx playwright test` **106 passed / 9 skipped**.

**Lacuna:** sem E2E do painel; caminhos são de demonstração (falta ligação ao `BuildProvider`).
**3 providers sem rota nem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1283

**Ronda 101. O E2E do `SbomPanel`: o sétimo elo fechado no browser real.**

- **E-1283** — `e2e/sbom-panel.spec.ts` + a rota `sbom/generate` no fixture.
- **Os tipos são os do BACKEND, e o spec verifica:** o fixture emite `MODULO_GO` e `PACOTE_NPM`, e o
  spec exige que o painel mostre "modulo Go" e "pacote npm". **É o antídoto do defeito
  inventado-vs-actual**: se o painel inventasse `GO_MODULE`, o E2E falhava — não o teste unitário, que
  usava o mesmo valor inventado.
- **Não tem campo de raiz, provado no browser:** `input` com `toHaveCount(0)` e
  `Object.keys(postDataJSON()) === []`, com ausência de `raiz`/`caminho`/`path`/`directorio`/`root`.
- **Não existe botão de assinar nem de atestar:** um único botão; `assinado` e `provenance` leem
  `nao`; o aviso contém "SBOM MINIMO", "sempre false" e "nunca do pedido".

**ACHADO DE CI — `cancelled` NÃO É FALHA:** em `5718d7f25`, 17 success + **6 `cancelled`** + 2
`failure` (gitleaks). Os 6 `cancelled` são **concurrency cancel-in-progress** — o push seguinte
cancelou-os (`started_at` 22:52, commit seguinte no mesmo minuto). **Ler `cancelled` como falha
inflacionaria o vermelho de 2 para 8.**

Medido: **9 passed / 1 skipped** · `npx playwright test` **106 passed / 9 skipped** (era 97/8) ·
`npx vitest run` 126 ficheiros / 897 testes · `tsc -b` 0 · `eslint .` 0 erros · `vite build` 0 ·
`BUNDLE_BUDGET=PASS`.

**Lacuna:** no modo de backend real só a recusa 412 corre. **4 providers sem rota nem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1282

**Ronda 100. O sexto provider fechado nos três elos: `SbomPanel` consome `sbom.generate`.**

- **E-1282** — `sbomClient.ts` + `SbomPanel.tsx` + **11 testes**, montado em `CapabilitiesPage`.
  **Décimo sétimo reabrir do defeito, sétimo fecho.**
- **Defeito do "inventado vs actual", apanhado ANTES de correr:** escrevi
  `GO_MODULE | NPM_PACKAGE | DESCONHECIDO`; os valores reais são **`MODULO_GO` / `PACOTE_NPM` /
  `BINARIO`**. Corrigido lendo `compliance_provider.go` antes de testar. *Um teste que inventa o valor
  que o backend devolve nunca falha por causa do backend.* **Décimo reabrir deste defeito desde
  E-1245.**
- **Não tem campo de raiz, e há teste que o prova:** `input` com `toHaveLength(0)` e o corpo do
  pedido é `{}` — `Object.keys(corpo) === []`.
- **Não existe botão de assinar nem de atestar:** um único botão, com "assinar"/"atestar" proibidos.
  **`nao_produzido` é mostrado por extenso.**
- **Um inventário de zero componentes não é erro:** **zero** `role="alert"`, `total=0` e a frase
  "nenhuma fonte de dependência encontrada". **`assinado`/`provenance` verdadeiros GRITAM.**

Medido: **11 testes PASS** · `npx vitest run` **126 ficheiros / 897 testes** · `tsc -b` 0 ·
`eslint .` 0 erros · `vite build` 0 · `BUNDLE_BUDGET=PASS`.

**Lacuna:** sem E2E Playwright. O SBOM não cobre transitivas nem attestation. **4 providers sem rota
nem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1281

**Ronda 99. O sexto provider ligado ao produto: `sbom.generate` consome o `ComplianceProvider`.**

- **E-1281** — `server/agent_sbom_route.go` + **6 testes**. **Décimo sexto reabrir do defeito, sexto
  fecho.**
- **A raiz vem do ambiente e é uma variável própria:** `HADES_RAIZ_SBOM`, não `HADES_ARTEFACTOS`.
  *`HADES_ARTEFACTOS` é a fronteira dos binários; `HADES_RAIZ_SBOM` é a fronteira do código. São cofres
  diferentes, logo variáveis diferentes — e nenhuma das duas é escolhida pelo cliente.* Há teste que
  envia um `raiz` no corpo a apontar para um diretório vazio e exige que o ambiente ganhe.
- **Recusas:** sem a variável **412**; raiz que não é directório **412**; falha do provider **409** —
  *um 500 aqui confundiria "o servidor falhou" com "o projecto não tem fontes"*.
- **`assinado` e `provenance` são SEMPRE false**, e o aviso declara "MINIMO", "NAO inclui", "sempre
  false" e "nunca do pedido". *Um SBOM que inclui o que o programador supõe é pior do que nenhum.*
- **Um directório vazio não é erro:** **200** com `total: 0` e `nao_produzido` **não vazio** — um SBOM
  vazio e válido seria uma mentira silenciosa.

Medido: **6 PASS** · `./server` ok 51,3 s · `go build ./...` 0 · `go vet` 0 · lint 0 issues.

**CI medido:** `6ee5ebb64` → **17 success · 2 failure (só os gitleaks históricos) · 3 skipped**.
Nenhum vermelho novo.

**Lacuna:** a rota **não tem painel**. O SBOM não cobre transitivas, notices, provenance nem
attestation. **4 providers sem rota nem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1280

**Ronda 98. O E2E do `SigningPanel`: o sexto elo fechado no browser real.**

- **E-1280** — `e2e/signing-panel.spec.ts` + a rota `sign/plan` no fixture.
- **Sem campo de senha, provado no browser:** `input` com `toHaveCount(2)`,
  `input[type="password"]` com contagem **zero**, e o texto sem "senha" nem "chave privada:".
- **Sem botão de assinar, provado no browser:** `getByRole("button")` tem de ser **exactamente um**,
  com rótulo a conter "avaliar" e não "assinar".
- **O corpo do pedido é inspeccionado:** `postDataJSON()` exige `[identidade, tipo]` e, dentro da
  identidade, `[ambiente, caminho_do_certificado, rotulo]`; o JSON serializado não pode conter
  `senha`, `password`, `chave_privada`, `private_key`, `pfx` nem `p12`.
- **Um certificado sem EKU não pode ser tratado como apto:** `apta=nao`, frase do EKU, OID
  `1.3.6.1.5.5.7.3.3` e `prova=nao assinavel`.

Medido: **9 passed / 1 skipped** · `npx playwright test` **97 passed / 8 skipped** (era 88/7) ·
`npx vitest run` 125 ficheiros / 886 testes · `tsc -b` 0 · `eslint .` 0 erros · `vite build` 0 ·
`BUNDLE_BUDGET=PASS`.

**CI no momento desta ronda:** 17 check-runs **pendentes, 0 vermelhos** em `6ee5ebb64`.

**Lacuna:** sem E2E com certificado REAL do kit. **5 providers sem painel.** Nenhuma assinatura real
aplicada.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1279

**Ronda 97. O quinto provider ligado ao produto: `SigningPanel` consome `sign/plan`.**

- **E-1279** — `signingPlanClient.ts` + `SigningPanel.tsx` + **11 testes**, montado em
  `CapabilitiesPage`. **Décimo quinto reabrir do defeito, quinto fecho.**
- **Defeito real, causado por mim, apanhado pelo `tsc`:** escrevi o cliente **por cima** de
  `signingClient.ts`, que **já existia** e serve o Signing Identity Center (FASE 28.5). Partiu
  `SigningIdentitiesPage.tsx` com **8 erros**. *Um nome que parece livre não prova que o ficheiro não
  existe: a pasta não foi lida.* Corrigido: original restaurado do `HEAD`, novo cliente em
  `signingPlanClient.ts`, tipo renomeado para `IdentidadeParaAvaliar`.
- **Sem campo para segredo, e há teste que o prova:** os `input` têm de ser **exactamente**
  `[assinatura-certificado, assinatura-rotulo]`, nenhum `type="password"`, e o corpo do pedido tem de
  ter `Object.keys(identidade) === [ambiente, caminho_do_certificado, rotulo]`.
- **Cada negação tem a sua frase, e o teste discrimina:** `SEM_EKU` mostra o OID; **`EXPIRADA` mostra
  "expirada" e o teste exige que a frase de EKU NÃO apareça** — *dizer "falta o EKU" a um certificado
  expirado manda quem lê corrigir a coisa errada.* Valor desconhecido **falha alto** no `default` do
  `switch` com `never`.

Medido: **11 testes PASS** · `npx vitest run` **125 ficheiros / 886 testes** · `tsc -b` 0 ·
`eslint .` 0 erros · `vite build` 0 · `BUNDLE_BUDGET=PASS`.

**Lacuna:** sem E2E Playwright. **5 dos 10 providers sem painel.** O provider **não valida cadeia
criptográfica nem revogação** — só o nome da CA emissora.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1278

**Ronda 96. O E2E do `RollbackPanel`: o quinto elo fechado no browser real.**

- **E-1278** — `e2e/rollback-panel.spec.ts` + a rota `publish/rollback` no fixture.
- **Duas classificações:** `UI_FIXTURE` (plano com candidato, ou **409** do Play sem service account)
  e `BACKEND_REAL_AGENT_RUNTIME` (atravessa o proxy; **`HADES_ARTEFACTOS` não existe em CI** ⇒ **412**,
  e o spec exige **erro accionável nomeando a variável**, nunca um plano).
- **Nenhum botão de restaurar, provado no browser:** `toHaveCount(1)` nos botões do painel e proibição
  dos rótulos `restaurar`, `aplicar`, `executar rollback`.
- **O corpo do pedido é inspeccionado no browser:** `postDataJSON()` exige `Object.keys === ["tipo"]`
  e a ausência de `directorio`.
- **O 409 do Play não é alerta genérico:** mostra *"service account JSON"* em `role="alert"` e
  **nenhum plano** aparece.

Medido: **7 passed / 1 skipped** · `npx playwright test` **88 passed / 7 skipped** (era 81/6) ·
`npx vitest run` 124 ficheiros / 875 testes · `tsc -b` 0 · `eslint .` 0 erros.

**A lição da E-1275 foi aplicada antes de correr:** o `readBody` devolve string, logo `JSON.parse`
desde o início — não custou uma falha.

**Lacuna:** no modo de backend real só a recusa 412 corre. **6 dos 10 providers sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1277

**Ronda 95. O quarto provider ligado ao produto: `RollbackPanel` consome `publish.rollback`.**

- **E-1277** — `rollbackClient.ts` + `RollbackPanel.tsx` + 8 testes, montado em `CapabilitiesPage`.
  **Décimo quarto reabrir do defeito, quarto fecho.**
- **Não existe botão de restaurar, e há teste que o prova:** conta os botões e exige **exactamente um**
  ("Planear rollback"), proibindo os rótulos "restaurar", "aplicar" e "executar rollback".
- **O pedido leva só o `tipo`, nunca um `directorio`:** o teste faz `JSON.parse` do corpo e exige
  `Object.keys(corpo) === ["tipo"]`.
- **O painel grita se o servidor declarar `executado: true`** — mostra *"ATENCAO: o servidor declarou
  que EXECUTOU o rollback"* em vez de o esconder. *Um estado que só o programador pode nomear inválido
  não é um estado.*
- **Canal não suportado não é alerta:** o 409 traz a explicação e o painel mostra-a como informação.

Medido: **8 testes PASS** · `npx vitest run` **124 ficheiros / 875 testes** · `tsc -b` 0 ·
`eslint .` 0 erros · `vite build` 0 · `BUNDLE_BUDGET=PASS`.

**Lacuna:** sem E2E Playwright. **6 dos 10 providers sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1276

**Ronda 94. O terceiro provider ligado ao produto: `publish.rollback` consome o `RollbackProvider`.**

- **E-1276** — `server/agent_rollback_route.go` + 6 testes. A E-1268 provou o provider com 7 testes,
  **mas só os testes o invocavam**. **Décimo terceiro reabrir do defeito, terceiro fecho.**
- **O directório vem do AMBIENTE, não do pedido:** lê `HADES_ARTEFACTOS` — a **mesma** variável que
  `build.smoke`. *Duas rotas que leem o mesmo cofre não podem ter duas fronteiras diferentes.* Há teste
  que envia um `directorio` no corpo e exige que seja **ignorado**.
- **`tipo` vazio é recusado (400):** *o canal de rollback não tem valor por omissão* — escolher um
  canal em nome de quem pede seria inventar a intenção.
- **Canal sem credencial devolve 409, não 500:** *um 500 aqui confundiria "o servidor falhou" com "o
  canal não existe"*.
- **`executado` é SEMPRE false** e o `aviso_de_fronteira` garante que nenhum ficheiro foi movido.

**Oito rotas do Release Hub:** `build/plan` · `build/artifact/checksum` · `build/artifact/verify` ·
`build/verify` · `build/smoke` · `publish/rollback` · `sign/plan` · `credentials/status`.

Medido: 6 PASS; `./server` ok 58,9 s; build 0; vet 0; lint 0.

**Lacuna:** a rota **não tem painel**. **7 dos 10 providers sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1275

**Ronda 93. O E2E do `SmokePanel`: o segundo elo fechado no browser real.**

- **E-1275** — `e2e/smoke-panel.spec.ts` + a rota `build/smoke` no fixture.
- **Duas classificações:** `UI_FIXTURE` (smoke aprovado com aviso) e `BACKEND_REAL_AGENT_RUNTIME`
  (atravessa o proxy; **`HADES_ARTEFACTOS` não existe em CI** ⇒ **412** — e o spec exige que o painel
  mostre isso como **erro accionável nomeando a variável**, nunca como resultado).
- **Quatro garantias nos dois modos:** diz que executa antes; botão **desactivado** sem caminho;
  **nenhum resultado sem pedido**; e **o pedido HTTP sai de verdade**.
- **No modo fixture:** saída `0.40.1`, duração `42 ms`, **aviso de sandbox** com *"NAO ha' sandbox"*,
  e o painel **não pode conter "binário seguro"**.

**Defeito real apanhado na execução:** a fixture devolvia 400 a dizer que faltava o `caminho`. A causa
não era o painel — era eu: **`readBody` devolve uma string** e eu testava `typeof corpo === "object"`.
**O sintoma parecia defeito do cliente e era do fixture.** Corrigido com `JSON.parse`.

Medido: **7 passed / 1 skipped**; `npx playwright test` **81 passed / 6 skipped**;
`npx vitest run` 123 ficheiros / 867 testes; `tsc -b` 0; `eslint .` 0 erros.

**Lacuna:** no modo de backend real só a recusa 412 corre — falta E2E com raiz REAL e binário REAL.
**8 dos 10 providers sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1274

**Ronda 92. O segundo provider ligado ao produto: `SmokePanel` consome `build.smoke`.**

- **E-1274** — `smokeClient.ts` + `SmokePanel.tsx` + 11 testes, montado em `CapabilitiesPage`. A
  E-1273 provou a rota com 6 testes, mas **nenhum painel a consumia**. **Décimo segundo reabrir do
  defeito, segundo fecho** — agora `CredentialProvider` e `TestProvider` têm rota + cliente + painel.
- **Três obrigações de um painel que executa código, todas com teste:**
  1. **dizer que executa antes** do botão;
  2. **mostrar o aviso de sandbox do backend por inteiro** — *um aviso que fica na documentação não
     chega a quem correu o binário*;
  3. **não chamar "funciona" ao que só arrancou** — o painel escreve *"provado que arranca e responde;
     NÃO provado que faz o que promete"*, e um teste proíbe a frase "binário seguro".
- **Não há campo para a raiz:** o painel não escolhe onde estão os binários. *Um cliente que escolhe a
  raiz dos binários é um cliente que decide o que corre.*
- **`separarArgumentos` não imita o shell:** respeita aspas mas **não expande `$VAR`** — teste com
  `$HOME/x` a exigir que fique literal.
- **Botão desactivado sem caminho**; **timeout mostrado como processo que não terminou**, nunca como
  aprovado.

Medido: **11 testes PASS** · `npx vitest run` **123 ficheiros / 867 testes** · `tsc -b` 0 ·
`eslint .` 0 erros · `vite build` 0 · `BUNDLE_BUDGET=PASS`.

**Lacuna:** sem E2E Playwright para o `SmokePanel`. **8 dos 10 providers sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1273

**Ronda 91. A primeira rota que EXECUTA um binário — e a fronteira que a torna segura.**

- **E-1273** — `server/agent_smoke_route.go` + 6 testes. `build/plan`, `artifact/*` e `sign/plan` são
  GET e só leem; esta é **POST e executa**. Sem fronteira seria um serviço de execução remota:
  *uma rota que executa o que lhe mandarem não é um TestProvider: é um shell com JSON*.
- **A fronteira é uma RAIZ, não uma lista de extensões:** o caminho tem de estar dentro de
  `HADES_ARTEFACTOS`. Sem a variável, a rota recusa com **412**.
- **Três verificações, nesta ordem:** normalizar; **resolver symlinks nos dois lados**; conter com
  `filepath.Rel`. A do meio é a que importa — *uma verificação de prefixo sem resolver symlinks é uma
  verificação que aprova o link, não o alvo*. Há teste com symlink a escapar a exigir 403.
- **Não há shell:** `exec.Command` com argumentos como lista. *Um argumento que passa por um shell é
  um argumento que o shell pode reescrever.*
- **Recusa sem argumentos**, recusa >32 argumentos (413) e **corta o timeout para o máximo** em vez de
  o aceitar por escrito. *Um timeout escolhido pelo cliente é uma pausa que o servidor aceita por
  escrito.*
- **A fronteira vai no resultado:** `sem_sandbox: true` + `aviso_de_sandbox`.

**Sete rotas do Release Hub:** `build/plan` · `build/artifact/checksum` · `build/artifact/verify` ·
`build/verify` · `build/smoke` · `sign/plan` · `credentials/status`.

Medido: 5 PASS / 1 SKIP (symlink no Windows); `./server` ok 54,2 s; build 0; lint 0.

**CI dos três commits anteriores: 0 red novos** — só os 2 gitleaks históricos.

**Lacuna:** a rota **não tem painel**.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1272

**Ronda 90. O E2E que faltava: o painel de credenciais no browser real.**

- **E-1272** — `e2e/credentials-panel.spec.ts` + a rota `credentials/status` no fixture. A E-1271
  provou o painel com transporte **mockado**; a E-1270 provou a rota com `httptest`; **os dois nunca
  falaram um com o outro**.
- **Duas classificações, e o spec diz qual corre:** `UI_FIXTURE` (kit com certificado e segredo) e
  `BACKEND_REAL_AGENT_RUNTIME` (atravessa o proxy até ao router de produção, onde o kit não está
  configurado em CI — e o spec exige a mensagem própria, **não um alerta**).
- **Cinco garantias nos dois modos:** painel com nome acessível; **o pedido HTTP sai de verdade**
  (`page.waitForRequest`); **"não configurado" não é erro**; aviso de fronteira no DOM; **nenhum botão
  existe** no painel.
- **No modo fixture, três asserções discriminantes:** certificado com sujeito e validade; segredo com
  **"NAO lido"**; e o painel **não pode conter "verificado com sucesso"**.

**Dois defeitos apanhados na execução:**
1. **`LOCALAPPDATA` redireccionado para `_tmp`** fez o Playwright procurar os browsers no sítio errado
   — ele guarda-os em `%LOCALAPPDATA%\ms-playwright`, e esse redireccionamento não serve para ele.
2. A fixture usava **`json(200, ...)`** mas o helper é **`sendJSON(response, status, payload)`** — o
   `ReferenceError` derrubava o servidor inteiro (daí `ECONNRESET` em todas as rotas).

Medido: **7 passed / 1 skipped** no spec; `npx playwright test` **74 passed / 5 skipped**;
`npx vitest run` **122 ficheiros / 856 testes**; `tsc -b` 0; `eslint .` 0 erros.

**Lacuna:** no modo de backend real só a garantia de "não configurado" corre — falta o E2E com o kit
REAL configurado no servidor.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1271

**Ronda 88. O primeiro painel que consome um dos dez providers: `CredentialsPanel`.**

- **E-1271** — `credentialsClient.ts` + `CredentialsPanel.tsx` + 7 testes, montado em `CapabilitiesPage`.
  A E-1270 provou a rota com 7 testes, mas **nenhum painel a consumia**: *um painel provado contra um
  mock e um backend provado contra `httptest` não provam que um fala com o outro*.
- **Décimo primeiro reabrir do defeito recorrente, primeiro fecho** — o elo rota→cliente→painel
  existe agora inteiro.
- **Não há campo para o valor de um segredo:** o tipo `CredencialPublica` não o tem, o backend não o
  envia e o painel não tem onde o guardar. *Um painel que tem um campo para o segredo acaba por o
  receber.* `eSegredo()` é uma lista **positiva** dos que não são segredo — um tipo novo cai em
  "é segredo".
- **Três estados que não se confundem, todos com teste:** não configurado (mensagem própria, não
  alerta), erro (`role="alert"`) e kit vazio.
- **"Não lido" é informação, não falha:** um teste exige que o painel **não** contenha "verificado com
  sucesso".
- **O aviso de fronteira vem do backend e é mostrado por inteiro** — o painel não o resume.

Medido: **7 testes PASS** · `npx vitest run` **122 ficheiros / 856 testes** · `tsc -b` 0 ·
`eslint .` 0 erros (31 warnings preexistentes) · `vite build` 0 · `BUNDLE_BUDGET=PASS`.

**Lacuna:** sem E2E Playwright — o elo está provado por teste de componente com `agentFetch` mockado.
**9 dos 10 providers continuam sem painel.**

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1270

**Ronda 87. O primeiro provider ligado ao produto: `credentials.status`.**

- **E-1270** — `server/agent_credentials_route.go` + 7 testes. A E-1265 media o kit real mas **só os
  testes a invocavam**: *uma comparação que só o programador consegue invocar não é uma
  funcionalidade: é uma função*.
- **O caminho vem do AMBIENTE (`HADES_KIT_CREDENCIAIS`), não do pedido.** Há teste que passa um
  `?raiz=` para outro directório e exige que seja ignorado. *Uma rota que aceita qualquer caminho é
  uma rota que lê qualquer caminho.*
- **Redacção:** devolve só caminhos relativos e **não devolve a raiz** — teste serializa a resposta e
  exige que a string da raiz não apareça nela. *A rota que devolve o mapa do cofre não é a rota que
  mostra o estado do cofre.*
- **Segredos continuam não lidos:** `conteudo_lido: false`, `fingerprint` vazio, frase "NAO lido" no
  `.p12` — tudo exigido por teste.
- **Não configurado não é erro:** responde **200** com `configurado: false` e diz qual variável
  definir. Um 404 faria um cliente concluir que o servidor está quebrado.

**Seis rotas do Release Hub:** `build/plan` · `build/artifact/checksum` · `build/artifact/verify` ·
`build/verify` · `sign/plan` · `credentials/status`.

Medido: 7 PASS.

**Lacuna:** a rota **não tem painel** no frontend. O defeito recorrente deu um passo, não fechou.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1268 e E-1269 -- OS DEZ PROVIDERS DO ADENDO 28.1 COMPLETOS

**Ronda 86. Fecham os dois últimos: `RollbackProvider` e `PublishingProvider`.**

### E-1268 -- `RollbackProvider`: o plano que NUNCA executa

- `Planear` monta o plano e verifica a integridade do alvo; **não restaura**. Restaurar um binário de
  produção é exactamente a acção que o adendo 28.5 põe atrás de *"confirmação humana no momento da
  acção"*. Um `Executar()` que só devolvesse "não autorizado" seria *um botão morto*.
- **`Executado` é SEMPRE false** e o aviso garante que *nenhum ficheiro foi movido, copiado ou
  apagado* — há teste que exige as duas frases.
- **Cada candidato tem SHA-256 medido em fluxo** e o teste recalcula por fora e compara.
- **Canal sem credencial é declarado, não fingido:** Play (`CONFIG_REQUIRED`), App Store (sem macOS),
  EAS Update (sem canal), OCI (sem registry) — 4 canais com `ErrRollbackNaoSuportado` e motivo.

### E-1269 -- `PublishingProvider`: e ele NÃO PUBLICA, por desenho

- A autorização proíbe publicar, por isso o provider implementa **`publish.preflight`** (o nome do
  adendo 28.7) e **não** `publish.submit`. A diferença é a diferença entre verificar e agir.
- **`Publicado` é SEMPRE false** nos 6 canais.
- **`LOCAL_ONLY` NÃO é deploy público:** a explicação diz que o artefacto *está no disco, NÃO que foi
  publicado* — e há teste que exige a frase.
- **Canal público nunca é declarado disponível:** a condição fica insatisfeita com o detalhe a nomear
  a proibição; `RequerAprovacao` é verdadeiro neles e falso no `LOCAL_ONLY`.
- **A credencial é verificada por existência, sem abrir:** há teste com **sentinela** de texto a
  provar que ela não vazou para o resultado nem para a explicação.

### Os dez providers

| # | Provider | Ficheiro |
|---|---|---|
| 1 | `ProjectDetector` | `project_detector.go` |
| 2 | `CapabilityResolver` | (via `PlanoParaProjeto`) |
| 3 | `BuildProvider` | `build_provider.go` |
| 4 | `SigningProvider` | `signing_provider.go` |
| 5 | `VerificationProvider` | `verification_provider.go` |
| 6 | `PublishingProvider` | `publishing_provider.go` |
| 7 | `CredentialProvider` | `credential_provider.go` |
| 8 | `ComplianceProvider` | `compliance_provider.go` |
| 9 | `TestProvider` | `test_provider.go` |
| 10 | `RollbackProvider` | `rollback_provider.go` |

**Nota honesta:** o item 2 (`CapabilityResolver`) tem apenas **1 ocorrência** em `internal/agent` —
é o nome no comentário do contrato, não uma abstracção própria. O plano de capability existe em
`PlanoParaProjeto`, mas **não há interface `CapabilityResolver`**. Declarado como lacuna.

Medido: `./internal/agent` **ok 340,1 s**; build 0; vet 0; lint 0.

**Lacuna transversal:** **nenhum dos dez tem rota HTTP própria nem painel** — só `sign.plan`,
`build/*` e `artifact/*` têm rotas. O defeito recorrente continua.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1267

**Ronda 85. O oitavo dos dez providers: `TestProvider` — o primeiro teste que EXECUTA o artefacto.**

- **E-1267** — `test_provider.go` + `process_tree.go` + 8 testes.
- A E-1261 prova integridade do ficheiro; **não prova que o binário corre**. Este provider fecha o
  degrau `build → verificar checksum → EXECUTAR e observar`.
  *Um ficheiro com bytes não é um artefacto utilizável.*
- **Três fronteiras no resultado:** `sem_sandbox: true` com aviso (sem Docker nesta máquina), não
  instala (E-1252), e **não aceita comando arbitrário** — só o próprio artefacto com os argumentos
  declarados. *Um evento sem allowlist não é um evento: é código com um nome diferente.*
- **Recusa executar sem argumentos** — *executar um binário sem argumentos é executá-lo no modo que o
  programador não escolheu*.
- **DEFEITO REAL encontrado pelo próprio teste:** `exec.CommandContext` matava o `cmd.exe` e deixava
  o `ping` filho vivo a segurar o pipe — **59,5 s para um timeout de 400 ms**. *Um timeout que mata o
  pai e deixa o filho não é um timeout: é uma promessa.* Corrigido com `cmd.Cancel` →
  `taskkill /T /F /PID` + `WaitDelay`. **Depois: 0,61 s.**
- **Alvos declarados com honestidade:** `AlvosTestaveis` = `BINARIO_DO_HOST`/`EXE_WINDOWS`;
  `AlvosNaoTestaveis` declara `EMULADOR_ANDROID`, `SIMULADOR_IOS`, `VM_EFEMERA`,
  `CONTENTOR_DOCKER`, `DISPOSITIVO_FISICO` — todos `BLOCKED_EXTERNAL` com motivo.

**Medido contra o artefacto REAL da E-1251:** `hades-windows-amd64.exe --version` →
`ollama version is 0.40.1`, **exit 0**, **2960 ms**.

Medido: 8 PASS; `./internal/agent` **ok 367,7 s**; build 0; vet 0; lint 0.

**8 dos 10 providers.** Faltam 2: `Publishing`, `Rollback`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1266

**Ronda 84. O sétimo dos dez providers: `ComplianceProvider` — o SBOM que recusa produzir o que não consegue.**

- **E-1266** — `compliance_provider.go` + 9 testes. Lê `go.mod` (bloco `require`, linha única,
  `// indirect`) e `package.json` (separando `dependencies` de `devDependencies`).
- **As três coisas que o adendo 28.9 pede e este provider NÃO faz, ditas no corpo:**
  `THIRD_PARTY_NOTICES` (exige o TEXTO da licença), árvore transitiva (exigiria subprocessos),
  `provenance` (exige builder e artefacto). *Um SBOM que inclui o que o programador supõe é pior do
  que nenhum.*
- **`Licenca` vazio NÃO é "sem licença":** significa **NÃO DECLARADA** — factos diferentes, e há
  teste que exige a frase no `aviso`.
- **`package.json` inválido não inventa componentes:** entra como `INVALIDO` em `FicheirosLidos`.
- **`node_modules` não é percorrido** — teste com dependência inventada a provar.
- **Assinatura estável:** *um SBOM que muda de ordem a cada leitura não permite dizer se mudou.*

**Medido contra o próprio repositório HADES:** **113 módulos Go**, 113 sem licença declarada,
**0 ficheiros npm** — e o limite foi encontrado: os `package.json` estão **dois níveis abaixo** e o
provider só olha a raiz e subdirectórios imediatos. **Declarado, não escondido.**

**Defeito meu apanhado:** `TipoDeComponente` somado com `string` sem conversão. E o `misspell` acusou
`directos` → "directors" (a grafia correcta é `diretos`).

Medido: 8 PASS; PASS contra o repo real; `./internal/agent` **ok 354,1 s**; build 0; vet 0; lint 0.

**7 dos 10 providers.** Faltam 3: `Publishing`, `Test`, `Rollback`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1265

**Ronda 83. O sexto dos dez providers: `CredentialProvider` — e a regra que o define é NÃO LER.**

- **E-1265** — `credential_provider.go` + 9 testes. Abre **certificados públicos**; faz **apenas
  `os.Stat`** em `.p12`, `.jks`, `.p8`, `.dpapi`, `.mobileprovision`.
  *Não leia o segredo para provar que ele existe: o tamanho e a data já provam que o ficheiro está
  lá, e a leitura é o risco.*
- **A prova de que não abre não é o comentário:** um teste tira a permissão de leitura a um `.jks`
  (`chmod 000`) e verifica que ele **continua** no inventário com o tamanho certo. Um `os.Open`
  teria falhado.
- **`DESCONHECIDA` também não é lida:** *na dúvida, não se abre.*
- **Três defeitos apanhados na própria ronda:**
  1. `Certificados` contava o TIPO e não os válidos — corrigido para exigir fingerprint.
  2. **`misspell` acusou `inventario` → "inventor"** — mesma armadilha da E-1255, resolvida com
     `ignore-rules`. *O `misspell` segue LIGADO.*
  3. **`nilerr` acusou dois `return nil` que engoliam erros** — e tinha razão: agora vão para
     `Avisos` + `NaoLegiveis`. *Um erro que não aborta não é um erro que não aconteceu.*

**Medido contra o kit real da DZ23 LTDA:** **29 ficheiros, 15 lidos, 14 NÃO lidos, 6 certificados,
0 expirados**. Certificados lidos de verdade: `hades-upload-certificate.pem` ("HADES", 2054), 3 DER
do Google Play (2056), 2 Apple Distribution ("leandro prado (K8G36CUP3S)", 2027-10-17). **Os 14
segredos não foram abertos.**

Medido: 8 PASS; PASS contra o kit real; `./internal/agent` **ok 352,1 s**; build 0; vet 0; lint 0.

**6 dos 10 providers** implementados. Faltam 4: `Publishing`, `Compliance`, `Test`, `Rollback`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1264

**Ronda 82. `sign.plan`: o `SigningProvider` deixou de ser função que só os testes invocam.**

- **E-1264** -- `POST /api/agent/v1/sign/plan` + 7 testes. **5 rotas do Release Hub** no router.
- **Não há `sign.start`** por decisão: a autorização proíbe assinar, e uma rota que só devolvesse 403
  seria *um botão morto — e um botão morto é pior do que um botão ausente: o ausente não mente*.
- **Dois defeitos reais apanhados pelos testes:**
  1. **`ErrCertificadoIlegivel` devolvia 500** (não mapeado) — um 500 diz "o servidor falhou" quando
     o que falhou foi o ficheiro indicado. Corrigido para **400**.
  2. O teste do certificado Apple **reescrevia o `Issuer` em memória** — mas o PEM vinha do DER
     original. *Um teste que reescreve o campo em memória e verifica o ficheiro no disco testa a
     memória.* Corrigido gerando a folha a partir de uma CA fictícia; o Apple passou a ser recusado
     com `EMISSOR_INESPERADO`.
- **O aviso de fronteira ensina a ler `apta`:** não significa que o artefacto foi assinado, e a
  cadeia criptográfica e a revogação **NÃO** são validadas — há teste que exige as duas frases.

Medido: 7 PASS na rota; **23 PASS** no conjunto; `./server` **ok 46,9 s**; build 0; vet 0; lint 0.

**Ausente:** nenhum painel nem E2E chama `sign.plan`. A1 da DZ23 não avaliado (store do Windows).
Faltam **5 dos 10** providers.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1263

**Ronda 81. O quinto dos dez providers: `SigningProvider` — e um certificado REAL expôs um defeito meu.**

- **E-1263** -- `signing_provider.go` + 12 testes. **Nao** e' um `Assinar()` que devolve "nao
  autorizado": a missao proibe assinar, e *um botao morto e' pior do que um botao ausente*.
- `AvaliarCertificado` verifica **validade → emissor → EKU**, com 8 valores de aptidao e o motivo.
- **O defeito que o certificado real expôs:** `IdentidadeDeOutraCA` estava declarado e **nunca era
  produzido** -- codigo morto a fingir de verificacao. Medindo o `distribution.cer` **real** do kit,
  descobri que ele **TEM** o EKU de Code Signing `1.3.6.1.5.5.7.3.3`; sem `emissorCompativel` o
  provider declararia **um certificado Apple apto a assinar um `.exe` do Windows**.
- **A frase do adendo provada contra o ficheiro real:** *"Apple Distribution nao substitui Developer
  ID"* -- `APPLE_DEVELOPER_ID` → **`SEM_EKU_DE_ASSINATURA`**; `AUTHENTICODE` →
  **`EMISSOR_INESPERADO`**.
- **Validade antes do EKU:** *dizer "falta o EKU" a um expirado manda corrigir a coisa errada.*
- **Formato sem regra nao e' aprovado:** Android/OpenPGP → `FORMATO_DESCONHECIDO`, sem afirmar nada.
- **Limite declarado no codigo:** verifica o NOME da CA, nao a cadeia nem a revogacao.

Medido: 12 PASS; `./internal/agent` **ok 499,1 s**; build 0; vet 0; lint 0.

**Ausente:** sem rota HTTP nem painel. A1 da DZ23 **nao avaliado** (vive no store do Windows).
Faltam **5 dos 10** providers: `Publishing`, `Credential`, `Compliance`, `Test`, `Rollback`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1262

**Ronda 80. `build.verify`: a verificacao deixou de ser funcao que so' os testes invocam.**

- **E-1262** -- `POST /api/agent/v1/build/verify` + 8 testes. 4 rotas `build/*` no router.
- **POST e nao GET:** a lista de artefactos e checksums nao cabe num URL -- *uma rota que se le' mas
  exige POST obriga quem a usa a saber mais do que precisa*.
- **Recusas:** lote vazio 400 · sem `sha256` 400 · sem `caminho` 400 · **>64 artefactos 413** ·
  ficheiro apagado **404 e nao 500**, sem veredito.
- **Nao inventa registo de builds:** nao procura por ID -- *um ID que resolve para um ficheiro que
  ninguem registou e' um apelido, nao uma identidade*.
- **O aviso de fronteira ENSINA A LER o resultado:** `aprovada` nao significa que exista assinatura;
  o teste exige essa frase.
- **Adulterado devolve 200** com `aprovada=false` -- *um 4xx confundiria "pedido invalido" com
  "artefacto adulterado"*.

Medido: 8 PASS; `./server` **ok 70,2 s**; build 0; vet 0; lint 0.

**Ausente:** nenhum painel nem E2E chama `build.verify` -- o defeito recorrente volta a abrir-se
nesta rota. Sem adaptador de assinatura.

Proximo passo: o painel a chamar `build.verify` e um E2E que o atravesse.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1261

**Ronda 79. A ligacao que faltava: construir -> VERIFICAR.**

- **E-1261** -- `VerificarConstrucao` + 5 testes. O SHA medido no build e' a REFERENCIA; o
  verificador mede o ficheiro AGORA.
- **Tres defeitos meus apanhados pelos proprios testes:**
  1. `errado` colidia com uma **funcao** do pacote;
  2. `artefacto` (com "c") e' **funcao** do pacote; a variavel era `artefato` -- o nome errado
     **resolveu para a funcao**. *Um identificador local que coincide com um do pacote nao
     sombreia.*
  3. **o importante:** a explicacao contava `NAO_VERIFICADA` como **"assinatura presente"**.
     *"Nao consegui olhar" e "olhei e nao ha'" sao factos diferentes.* Corrigido com `NaoVerificadas`
     separado; o teste agora **proibe a frase** quando ninguem inspeccionou.
- **Build vazio nao e' aprovacao:** `ErrSemArtefactosParaVerificar`.
- **Erro de infra nao vira veredito:** ficheiro apagado aborta.

Medido: 5 PASS; `./internal/agent` **ok 446,2 s**; build 0; vet 0; lint 0.
**Artefacto real da E-1251 verificado:** 69 206 016 B, `fcef9d9d...` aprovado; o MESMO ficheiro
reprovado com referencia errada.

**Nao medido / ausente:** `VerificarConstrucao` **nao tem chamador fora dos testes** -- o mesmo
defeito que esta fatia corrigiu para o `ConstruirAlvos`. Sem verificacao de assinatura real.

Proximo passo: rota `build.verify` a chamar `VerificarConstrucao`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1260

**Ronda 78. O E2E encontrou um P1 que nenhum teste de componente podia encontrar.**

- **E-1260** -- `e2e/artifact-verify.spec.ts` (9 testes) + fixture com negacoes + **correcao do
  `agentFetch`**.
- **O defeito:** `agenticClient.ts:677` lia `body?.error` (ingles); o backend responde
  `{"erro": ...}`. Todo o painel que mostre um erro de rota dizia **`Bad Request`** em vez da razao.
  Corrigido lendo os dois campos.
- **Porque ninguem apanhara antes:** o teste de componente mocka exactamente essa funcao --
  *um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
  com o outro*; e *um teste que substitui o cliente testa o mock, nao o produto*.
- **O fixture aprendeu a negar:** relativo -> 400, `inexistente` -> 404, referencia de 16 chars ->
  400. *Um fixture que so' serve o caminho feliz nao prova que o painel distingue os outros.*
- A regra de caminho absoluto foi repetida no fixture **de proposito** -- se o fixture fosse mais
  permissivo que o servidor, o spec passaria num comportamento que a producao recusa.
- O spec carrega o modo no nome do describe; o teste de ficheiro inexistente faz `skip` declarado.

Medido: 9 E2E PASS; **E2E completo 67 passed / 4 skipped / 0 failed** (eram 58); 121/849 vitest;
tsc 0; eslint 0.

**Nao medido:** o modo `BACKEND_REAL_AGENT_RUNTIME` nao foi executado nesta ronda.

Proximo passo: o `BuildProvider` a chamar o verificador no fluxo de build.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1259

**Ronda 77. As rotas da E-1258 deixaram de estar so' no servidor: o painel chama-as.**

- **E-1259** -- `artifactClient.ts` + seccao **dentro** do `BuildPlanPanel` (nao painel paralelo) +
  7 testes. *Tres pecas provadas em separado nao sao uma integracao provada.*
- **Dois nomes que nao se confundem:** `obterChecksum` mede, `verificarArtefacto` compara -- *uma
  funcao que so' sabe verificar obriga quem so' quer o hash a inventar uma referencia.*
- **O tipo obriga a cobrir os quatro estados:** `const inesperado: never = estado` -- *se o backend
  ganhar um estado novo, o TypeScript falha AQUI e nao no ecra.*
- **O aviso de fronteira vem do backend** e e' mostrado tal e qual.
- **"Verificar" desactivado sem referencia**, com guarda tambem no handler.
- **Dois defeitos meus:** `vi.importActual`+spread voltou a dar `__spreadValues` (mesma armadilha da
  E-1248 -- corrigido mockando so' o `agentFetch`); e **9 dos 10 testes antigos falharam** com
  `Expected 1 but found 3 instances with node type: "input"` -- a seleccao passou a ser pelo `id`,
  **mais precisa**.
- **O bug `$T`/`$t` do PowerShell apareceu pela TERCEIRA vez.**

Medido: 7 + 10 PASS; **121 ficheiros / 849 testes**; tsc 0; eslint 0; build 0; budget PASS.

Proximo passo: E2E Playwright da seccao; depois o `BuildProvider` a chamar o verificador.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1258

**Ronda 76. O verificador deixou de ser uma funcao Go: `artifact.checksum` e `artifact.verify`.**

- **E-1258** -- `server/agent_artifact_verify_route.go` + 8 testes PASS. Adendo 28.7.
- **Duas rotas porque sao duas operacoes:** `checksum` mede; `verify` compara com referencia.
  *Junta-las obrigaria quem so' quer o hash a inventar uma referencia -- e uma referencia inventada
  "passa" sempre.*
- Recusas com significado: sem `caminho` 400; caminho **relativo** 400; inexistente 404; sem
  `sha256` 400; referencia **curta** 400.
- **Adulterado devolve 200 com `adulterado=true`, NAO 4xx** -- *um 4xx confundiria "pedido
  invalido" com "artefacto adulterado"*.
- A fronteira vai no CORPO (`somente_leitura`, `aviso_de_fronteira`), nao so' na documentacao.
- **Nao procuram por ID:** um registo de artefactos nao existe. *Um ID que resolve para um ficheiro
  que ninguem registou e' um apelido, nao uma identidade.*
- **Bug meu:** `$R` (caminho) e `$r` (conteudo) sao a MESMA variavel em PowerShell. O ficheiro ficou
  intacto (verificado) e as rotas foram registadas pela ferramenta de edicao.
  => *duas variaveis que diferem so' na caixa sao a mesma variavel.*

**CI do `993804798` (correcao do `darwin`): 0 red** no momento da medicao -- terceira causa fechada.

Medido: 8 testes PASS; `go test ./server` ok 55,4 s; lint 0; `go build ./...` 0.

Proximo passo: um painel ou o fluxo de build a chamar estas rotas.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1257

**Ronda 75. O terceiro red era o mesmo `darwin`!=`macos`, no sitio que a correcao anterior nao tocou.**

- **E-1257** -- `--- FAIL: TestBuildProviderConstruirAlvosSaltaBloqueados`, `artefactos = 0; esperado
  1`. O teste tinha `{Alvo: runtime.GOOS, Plano: plannerPronto(alvoDoHost())}`: a E-1253 substituiu
  o segundo mas **deixou o campo `Alvo`**. No macOS `darwin` != `macos` -> alvo **pulado**.
- => *Uma correccao aplicada por substituicao de texto corrige as ocorrencias que a STRING descreve,
  nao as que o DEFEITO tem.* **Quarto avatar** do mesmo defeito -- e **dentro do proprio teste que eu
  escrevi para o corrigir**.
- **Corrigido** (`Alvo: alvoDoHost()`) **e guardado**: duas assercoes **independentes de
  plataforma** -- `cobreAlvo(alvoDoHost())` true e `alvoDoHost()` != `darwin`. Sem elas o teste so'
  falhava no macOS: *um teste cuja falha depende do sistema onde corre e' um teste que so' se
  descobre na CI mais lenta.*
- A mensagem de falha passou a incluir `NaoConstruidos`, para nao ser preciso ir ao log.

Medido: 13 testes PASS; vet 0; lint **0 issues**; gofmt limpo.

Proximo passo: confirmar o commit verde; depois INSTALLED (E-1252).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1256

**Ronda 74. O item 5 de dez: `VerificationProvider` -- e o teste que me obrigou a decidir.**

- **E-1256** -- `internal/agent/verification_provider.go` + teste, **14 testes PASS**. O
  `BuildProvider` calculava o hash e devolvia-o: o unico verificador era quem produzia.
  => *um artefacto cuja unica verificacao e' o hash calculado por quem o produziu nao e'
  verificado: e' assinado por si mesmo.*
- Recusa verificar **sem referencia** (`ErrSemReferencia`) e **recusa um PREFIXO** de checksum
  (16 caracteres do checksum real sao recusados): *uma referencia curta nao e' uma referencia: e'
  uma esperanca com hexadecimal*.
- **O teste que importa:** adultera-se UM byte -> `Adulterado=true`, `Aprovado=false`.
- Assinatura com **quatro** estados, incluindo `NAO_VERIFICADA`: *"nao consegui olhar" e "olhei e
  nao ha'" sao factos diferentes, e um campo que os junta mente sobre um dos dois.*
- **O teste obrigou-me a decidir:** eu esperava `Aprovado=false` quando o adaptador falha; o teste
  estava errado, nao o codigo. Reprovar ali tornaria o verificador inutil fora do Windows.
  `Aprovado` passou a significar "passou os gates que consegui aplicar".
- **Validacao cruzada:** corrido contra o artefacto real da E-1251, mediu
  `fcef9d9d4303a75b0b9c658f155c9ae6a7ccb84912b4face2268dc51f8161df6` -- o mesmo que a E-1251
  declarou.

Providers fechados: **5 de 10**. Proximo passo: rota HTTP para o verificador; depois
`SigningProvider`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1255

**Ronda 73. O segundo red do CI era `misspell` -- e a palavra era portugues correcto.**

- **E-1253 VALIDADA:** com a correccao `darwin`->`macos`, `test (macos-latest)` passou a **success**
  no GitHub. Quatro dos cinco jobs de `go test` ficaram verdes.
- **O que restava:** `test (windows-latest)` RED, causa dada pelas anotacoes do check-run --
  `server/agent_e2e_harness_test.go:175/179`: **`projecto` -> `projection` (misspell)**.
- **Reproduzido antes de corrigir:** `golangci-lint run ./server` **exit 1**, 3 ocorrencias.
  *O texto que explica porque o fixture tem de ser um projecto verdadeiro foi o texto que o linter
  recusou.*
- **Correccao:** `projecto`/`projectos` no `ignore-rules`, pelo criterio que o proprio ficheiro ja'
  documenta. **O linter segue ligado.**
- **Medido:** `./server` exit 0 (era 1); `./internal/agent` exit 0 (config valida, nao exit 3);
  `./...` **0 misspell**; os 8 `unused` restantes sao **preexistentes**.
- **Dado novo:** `internal/agent` tem 35 `projecto` e nunca foi acusado, o `server` tinha 3.
  => *o dicionario casa contexto, nao so' a palavra; assumir "esta' ou nao esta' na lista" erra
  por omissao nos dois sentidos.*

Proximo passo declarado: confirmar o commit verde no GitHub; depois INSTALLED (E-1252).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1254

**Ronda 72. A proveniencia mentirosa do instalador foi corrigida -- e o gate falha de proposito.**

- **E-1254** -- `scripts\hades_installer_windows.ps1` passou a **medir cada alvo** (SHA-256, caminho,
  data) e a comparar a data com a do commit HEAD. Medido no `5cef8330f`: **3 de 3 alvos anteriores
  ao HEAD** -> declara `COMPOSICAO` e grava o aviso no `PROVENIENCIA.txt`.
- **TESTE NEGATIVO:** `-ExigirAlvosDoHead` sai com **exit 4** e **nao gera um unico `.exe`** -- o
  gate foi verificado pela **ausencia do artefacto**.
- **A HEURISTICA E'' DECLARADA:** *um timestamp nao e'' uma proveniencia: e'' um indicio de
  proveniencia.*
- **INSTALACAO:** quarto sintoma -- `& HadesSetup.exe /VERYSILENT /LOG=` devolve **exit 0 em 0 s**
  sem criar ficheiro nem log. O binario foi verificado como Inno Setup legitimo (`MZ`+`PE`, string
  `Inno Setup`, `VersionInfo` produto `Hades`). **4 tentativas, sintomas diferentes, parar.**

Proximo passo declarado: ligar `-ExigirAlvosDoHead` ao CI e reconstruir o app desktop no HEAD; a
instalacao exige diagnostico proprio.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1253

**Ronda 71. O CI ficou vermelho por minha causa -- e a causa era um nome.**

- **E-1253** -- `d547d7986` passou de 2 para **6 red**: os quatro jobs que correm `go test`
  (`windows`, `ubuntu`, `macos`, `race macos`). **Causa:** os testes do `BuildProvider` passavam
  `runtime.GOOS` como alvo; no macOS isso e' `darwin`, o catalogo diz `macos`.
  => *um teste que passa na maquina onde foi escrito e falha nas outras tres nao testa o produto:
  testa o sistema de quem o escreveu* -- **a mesma familia do defeito de caminhos** que ja' apareceu
  em tres avatares, agora num nome de sistema.
- **Corrigido em dois eixos:** `alvoDoHostPara(goos)` com tabela que cobre os tres sistemas; e
  `ambienteDeBuild()` que sobrepoe `CGO_ENABLED=0`/`GOOS`/`GOARCH` em vez de herdar o `GOOS=windows`
  que o CI exporta.

Proximo passo declarado: confirmar os 4 jobs verdes no GitHub; depois atacar a instalacao (E-1252).

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1252

**Ronda 70. O instalador Windows existe -- e a proveniencia dele mente.**

- **E-1252** -- `HadesSetup-d547d7986.exe`: **39,21 MB**, SHA-256
  `E8A733B8A45682A8E3B27D1F7D1BBEBC49CB3D65911C99805E62D99EF67CC50D`, **`NotSigned`** (o requisito
  da RC), produto **`Hades`**, versao `0.2.0.9012`. Compilado por ISCC 6.7.3 em 41,3 s.
- **O ACHADO:** o `PROVENIENCIA.txt` declara **um** SHA, mas o instalador empacota **tres** origens:
  motor reconstruido do HEAD (`b7f8ee93`), app desktop do commit `1d2ce88c9`, runner de 16/10.
  => *um artefacto composto que declara um SHA para partes de tres origens mente -- e mente mais
  alto do que um sem proveniencia, porque quem le' acredita.*
- **INSTALACAO NAO PROVADA:** tres execucoes nao instalaram um ficheiro nem escreveram log. Protocolo
  antiloop aplicado: 3 tentativas, mesmo sintoma, **parar**. Causa nao determinada; lacuna interna,
  nao `BLOCKED_EXTERNAL`.

Proximo passo declarado: corrigir a proveniencia composta (o script tem de listar o SHA de cada
alvo) e determinar porque o instalador nao escreve. Depois `SigningProvider`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1251

**Ronda 69. C1 CONSEGUIDO: um build real do HADES produzido, medido e reproduzido.**

- **E-1251** -- `internal/agent/build_provider.go` (item 3 do adendo 28.1) + 11 testes. **Artefacto:**
  `hades-windows-amd64.exe`, **66 MB**, SHA-256 `fcef9d9d4303a75b0b9c658f155c9ae6a7ccb84912b4face2268dc51f8161df6`,
  **`NotSigned`**, `--version` responde. **Segundo build = mesmo SHA-256: REPRODUZIVEL.**
- O provider recusa plano incompleto antes de criar `dist`, recusa alvo sem provider, dry-run nao
  executa, checksum recalculado no teste, binario **executado** e saida verificada, e nao tenta
  alvos bloqueados.
- **Erro meu corrigido na leitura:** o `go version -m` imprime o caminho do **ficheiro analisado**;
  eu quase declarei que o binario continha o caminho do build.

Proximo passo declarado: **instalar** o artefacto num Windows limpo (28.10 item 14) e gerar o
instalador Inno Setup. Depois `SigningProvider`.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1250

**Ronda 68.** O modo `BACKEND_REAL_AGENT_RUNTIME` foi exercitado -- e apanhou um bug que o fixture
nunca apanharia.

- **E-1250** -- harness a servir o **router de producao** em `127.0.0.1:43219`; `GET /build/plan`
  devolveu `200` com `stacks=go alvos=[linux windows] blockers=inno-setup (ABSENT)` sobre git real.
  E2E modo real **6/2/0**; modo fixture **9/0/0**.
- **O ACHADO:** o painel comparava `saudavel`/`ausente` -- **nomes que eu inventei**. Os `ToolState`
  reais sao `PRESENT_COMPATIBLE_HEALTHY`/`PRESENT_BROKEN`/`ABSENT`/`UNKNOWN`. **O fixture tambem os
  tinha inventado**, por isso passava nos dois lados: *um teste que inventa o valor que o backend
  devolve nunca falha por causa do backend* -- a mesma licao da E-1238. Corrigido no painel, no
  fixture e no mock do teste.

Proximo passo declarado: **C1** -- um build real produzido e instalado.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1249

**Ronda 67.** O painel e a rota passaram a falar um com o outro no browser real.

- **E-1249** -- `e2e/build-plan-panel.spec.ts`, duas classificacoes declaradas
  (`BACKEND_REAL_AGENT_RUNTIME` / `UI_FIXTURE`), rotas novas no fixture com **dois** planos (um com
  alvo pronto e um bloqueado; outro `sem_alvos`). **9 testes PASS**; suite E2E **58 passed / 4
  skipped / 0 failed** (antes 49/4/0). Dois testes sao **skipped no modo real** com a razao escrita
  -- *um teste que finge que o fixture e'' o backend prova o fixture, nao o backend*.

Proximo passo declarado: **exercitar o modo `BACKEND_REAL_AGENT_RUNTIME`** com o harness. Depois
**C1**: um build real instalado.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1248

**Ronda 66.** O plano de build passou a ter painel: o utilizador ve'' o que o projeto pede e o que
falta, sem abrir Go.

- **E-1248** -- `BuildPlanPanel.tsx` + `buildPlanClient.ts`, montado na pagina de Capacidades
  **depois de `WorktreesPanel`**, nao num painel paralelo. **10 testes**. O painel diz **no corpo**
  que apenas detecta e sonda; projeto sem alvos tem mensagem propria, nao erro; estados traduzidos
  para texto humano; blockers e motivos no topo. **Frontend inteiro verde:** `tsc` 0, `eslint .` 0,
  **120 ficheiros / 842 testes**, `vite build` 0, budget PASS.

**Quatro defeitos meus apanhados** -- `vi.importActual` com spread quebrava a transpilacao; campo
no sitio errado (`plano.gestores` vs `plano.projeto.gestores`); assumi que `agentFetch` devolvia
`Response` quando devolve o objecto tipado; e o mock devolvia um `Response` que o produto nao usa.

Proximo passo declarado: **E2E Playwright** do painel contra backend real. Depois **C1**: um build
real instalado.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1247

**Ronda 65.** O plano de release deixou de ser uma funcao que so'' o programador invoca: tem rota.

- **E-1247** -- `GET /api/agent/v1/build/plan?repo=`. **Nao aceita `alvo`** (ha'' teste que pede
  `&alvo=android` num repo Go e exige que seja ignorado) e declara a fronteira **no corpo**
  (`somente_leitura`, `nada_instalado`, `nada_construido`, `aviso_de_fronteira`). Codigos
  distinguidos por significado: **422** quando o caminho existe mas nao e'' um projecto. 10 testes.
  Auto-teste pela rota no repo HADES real: `alvos=[linux web windows]`.

Proximo passo declarado: **painel do frontend** que consome a rota -- e so'' depois **C1**: um build
real instalado.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1246

**Rondas 63-64.** O Build Hub deixou de receber o alvo escrito a mao: detecta o projecto e plana
contra o que ele PEDE.

- **E-1245** -- `ProjectDetector`: 7 stacks, gestor por lockfile, alvos filtrados pelo catalogo
  real, pastas nativas, manifests de prova. 15 testes. **Dois defeitos meus apanhados:** a regra
  de precedencia de lockfile apagava o npm quando era o npm o vencedor; e o `nilerr` acusou dois
  ramos onde eu engolia um erro de leitura em silencio.
- **E-1246** -- o ELO. `PlanoParaProjeto(ctx, raiz)` **nao aceita alvo**: o alvo vem do projecto.
  Cada alvo detectado tem plano ou erro explicito e nunca e'' omitido; `SemAlvos` e'' estado
  proprio, nao sucesso mudo. 9 testes.

Proximo passo declarado: **rota HTTP** para o Mission Control consumir o plano. Depois **C1**
propriamente dito -- um build real instalado.

Veredito global: `NOT_RELEASE_READY`.

## JANELA E-1244

**Rondas 60-62.** A matriz de lacunas existe, o validador do Studio esta'' ligado ao pipeline, e o
cenario E2E minimo da FASE 3 foi cumprido ponta a ponta.

- **E-1242** -- matriz Orca x HADES e OpenJarvis x HADES. FASE 0 com HEADs auditados
  (`1938bd05…` MIT e `2817faa5…` Apache-2.0), 26 capacidades em 3 grupos, 3 SUPERAR / 4 ADOTAR /
  19 ADAPTAR. **O achado que vale mais do que a matriz:** a primeira passagem do inventario
  procurou ficheiros com nomes inventados e devolveu 8 "AUSENTE"; a segunda procurou conteudo e
  encontrou `pty` 987, `SSH` 88, `Capability` 291, `VRAM` 27, `openai` 56, `inspec` 162.
  *Pela quinta vez o inventario errou na mesma direccao.*
- **E-1243** -- C3 da matriz: os validadores da E-1241 passaram a ser um GATE. `VisualComponent`
  tem `Props`/`Style`/`Bindings`/`Events` que o renderer executa; o validador existente so' via
  estrutura e **2 dos 3** caminhos de `renderVisualHTML` nem isso. 12 eventos e 11 accoes
  allowlisted. `TestCriacaoDeProjectoVisualRecusaComponenteInseguro` falha se a ligacao for
  removida.
- **E-1244** -- o cenario E2E minimo *"comparar duas solucoes e exigir aprovacao antes de
  integrar"*: repositorio git real, **duas solucoes em worktrees separados**, medicao por
  `git diff --stat`, rota de producao, e **prova de nao-integracao** (`main` no mesmo commit,
  `branch --merged` limpo, `extra.go` ausente, branches vivas).

Proximo passo declarado: **C1** -- matriz por target do Build Hub com pelo menos um build real
instalado. Depois **B3** -- os 12 E2E do Obsidian.

Veredito global: `NOT_RELEASE_READY`.

# HADES — PRÓXIMA AÇÃO (V2.2 + adendo de paridade)
## JANELA E-1231 -- FASE 2: WORKTREES COM PROTECCAO, ESTADOS E DESCOBERTA (rondas 45-49)

> Substitui a janela `E-1218`. *A catraca de janela acusou atras=13: o trabalho das rondas 45-49
> nao estava reflectido aqui.* **A protecao de trabalho nao integrado, a maquina de estados e o
> inventario de worktrees existem e estao provados com git real.** O que falta e' a superficie.

### O que ficou feito

| evidencia | o que prova |
|---|---|
| **E-1226** | `RemoveGitWorktree` apagava com `--force` + `os.RemoveAll` SEM verificar nada. Corrigido: diagnostico, erros tipados, guarda de alvo, branch preservado |
| **E-1227** | 9 estados fechados com transicoes declaradas; `InventariarWorktrees` descobre worktrees externos |
| **E-1228** | maquina LIGADA ao codigo: `CreateGitWorktree` -> `preparado`; suspender/retomar/arquivar/marcar-integrado; migracao `active`->`em_uso`; `PodeRemoverAgora` exige as duas condicoes |
| **E-1229** | CI macOS vermelho por minha causa: `/var` e' symlink de `/private/var`. Corrigido com `caminhoReal` |
| **E-1230** | `test (macos-latest)` **success** em `54df8a219`; restam so' os 2 gitleaks historicos |
| **E-1231** | mesmo defeito no runner Windows (`RUNNER~1` -> `runneradmin`); os testes passam por `caminhoReal` |

**37 testes** na suite de worktree, todos com git de verdade.

### As tres licoes de metodo que este ciclo produziu

1. *Apagar trabalho que ninguem integrou nao e' limpeza: e' perda de dados com nome tecnico.*
2. *Um caminho absoluto nao e' o mesmo que um caminho real.* (macOS `/var`)
3. *A correccao existia e o teste que a devia exercitar continuava a comparar strings.* (Windows)

A licao 3 e' a mais util: **a funcao `caminhoReal` foi criada para resolver o macOS e os testes
continuaram a usar `filepath.Clean` — corrigiram o codigo e nao o teste.** O mesmo defeito
apareceu em duas plataformas diferentes porque a comparacao era de strings, nao de alvos.

### O que continua AUSENTE (FASE 2)

- **nenhuma rota HTTP** expoe worktrees, estados, inventario ou diagnosticos;
- **nenhum painel** no Mission Control;
- suspensao/retomada **nao persiste entre processos** (o estado vive na sessao em memoria);
- o aviso de trabalho nao integrado na criacao nao foi exercitado num fluxo de missao ponta a ponta.

Estado da fase: `PARTIAL_IMPLEMENTED_NOT_E2E_VERIFIED`. Veredito global: `NOT_RELEASE_READY`.

### Proxima acao

Expor `GET /api/agent/v1/worktrees` (inventario + diagnostico por sessao) e o painel
correspondente; persistir o estado das sessoes para que suspender sobreviva a reinicio.
## JANELA E-1218 -- DUAS CAPACIDADES FECHADAS EM QUATRO CAMADAS; UM VERMELHO DE HISTORICO

> Substitui a janela `E-1205`.

### ESTADO MEDIDO

- `HEAD = 7b08e3e1e`, arvore limpa, 625 commits sobre `main`.
- `vitest run`: **117 ficheiros / 812 testes PASS**.
- `tsc -b` exit 0, `eslint` 0 erros, `BUNDLE_BUDGET=PASS`.
- `go test ./server/ ./internal/multillm/`: ambos `ok`; `golangci-lint` 0 issues.
- 41 `scripts/*_test.py` + 10 geradores + 10 checkers: 0 falhas.
- **CI: 2 red** -- apenas `gitleaks (historico do PR)`, com causa provada.

### A PROXIMA ACAO CONCRETA

**Nao repetir o diagnostico do gitleaks.** Esta' feito e medido em `E-1217` e `E-1218`; repetir
seria o mesmo comando pela terceira vez sem hipotese nova, o que o protocolo antiloop proibe.

O proximo valor de engenharia e' o **E2E Playwright** das duas paginas novas. As duas tem testes
com `react-test-renderer` e a camada HTTP mockada; *um E2E que intercepta toda a API e nunca toca
o backend real nao prova integracao.*

### O QUE NAO FAZER

- Force-push, rebase destrutivo, merge, tag, release, publicacao.
- Tocar no PR #438 como revisor (nem resolver threads).
- Usar allowlist no `.gitleaks.toml` para esconder o finding historico.
- Declarar `COMPLETO_VERIFICADO` em qualquer das duas capacidades: falta o E2E real.

## 📌 Checkpoint MEDIDO (rodada 101 — nada aqui é estimativa)

| Medida | Valor medido | Como foi medido |
| --- | --- | --- |
| `main` (remoto) | `4b3e21854` | `git rev-parse --short origin/main` |
| Branch do PR #438 | `eaad692e0` | `git rev-parse --short origin/fix/checker-relogio` |
| Evidências no índice | **547** | contagem de linhas `\| E-` |
| Estágios | **38** (*44–47 entregues*) | `HADES_MASTER_STATE.json` → `stages` |
| Bloqueios declarados | **17** (*17 itens / 17 ids únicos — o `B-15` duplicado foi removido no `E-545`*) | `hades_mission_check.py` |
| Conflitos com a `main` | **nenhum** | `git merge-tree --write-tree` → exit 0 |
| Credenciais expostas | **nenhuma** (*20 ocorrências de padrão `ghp_`, **20/20** em ficheiros de teste, **0** em produção*) | varrimento + classificação (`E-547`-era) |
| Veredito de release | **`NOT_RELEASE_READY`** | checker de missão |

> **Regra de honestidade sobre o SHA**: ele **envelhece a cada merge**. Este checkpoint vale para `eaad692e0`, medido em
> **2026-10-14**; qualquer trabalho posterior precisa **remedir** antes de citar.

---

## Fila imediata — REORDENADA pelas rodadas 40–101

### ✅ Já não está na fila (*entregue e testado*)
| O que era pendente | Onde ficou |
| --- | --- |
| **Estágio 44** — Cowork & Spark | `E-533`: `/spark`, `/cowork`, `/reject` na infra existente (*mesma rota de `/goal`*) |
| **Estágio 45** — Multimodal Studio | `E-529` matriz modalidade×engine · `E-535` a **recusa antes de disco/rede** · documentos PDF/DOCX/XLSX/PPTX já existiam |
| **Estágio 46** — Companion | `E-536` estados derivados de **eventos verificados** · `E-538` a11y ligada ao `prefers-reduced-motion` **existente** |
| **Estágio 47** — Artifact Drive | `E-537` busca + CAS · `E-539` comparar/restaurar **sem apagar história** · `E-542` dedup **dentro do tenant** |
| **FASE 1** — `race (macos)` | **provada**: 11/11 etapas, macOS 10 min 28 s vs ubuntu 6 min 32 s |

### ⏭️ O que resta, por ordem
| # | Ação | Critério de aceite |
| --- | --- | --- |
| 1 | **Revisão humana do PR #438** (*`LMPrado-DZ23`*) — **não** se contorna | aprovação registada; só depois o merge |
| 2 | **Contrato duplicado**: o `server` **repete** o número do `internal/agent` (*35→38 foi o sintoma*) | o `server` **lê** o valor em vez de o repetir — *dívida registada no `E-547`* |
| 3 | **P0-B / T2** harnesses externos (*Claude Code, Codex, Manus*) | `BLOCKED_BY_EXTERNAL_DEPENDENCY` até haver login ⇒ `NAO_MEDIDO` |
| 4 | **P0-A / T1** paridade Manus: pesquisa com fontes, planilha/documento, navegador autorizado | artefatos reais; navegador depende de ambiente autorizado |
| 5 | **P0-C / T3** app full-stack com auth, DB real, preview, E2E, rollback | **staging real** é dependência externa até haver alvo |
| 6 | **P1-E / T8** Creator Studio: MP4 **decodificável**, áudio, legendas, thumbnail | **MP4 real exige FFmpeg** ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` |
| 7 | **P1-F / T9** REA via MCP/skill governada sobre artefato próprio/autorizado | escopo, timeout e evidência; nunca burlar DRM |
| 8 | **P1-H** descoberta: README PT/EN, docs versionadas, sitemap, JSON-LD, acessibilidade | **sem** prometer ranking ou recomendação por IAs |

## Bloqueios que **não** se resolvem por código

`BLOCKED_BY_EXTERNAL_DEPENDENCY` com roteiro em `HADES_V22_IMPLEMENTATION_PLAN.md` §3: Ollama Cloud · CLIs
(Codex/Claude/Gemini) · canais Meta/Telegram · telefonia/SIP · contas de negócio · Apple/EAS · YouTube OAuth ·
assinatura Windows · licenças de ferramentas comerciais de REA · **FFmpeg** (produção real de MP4) · **TikTok**
(conector oficial) · **harnesses de paridade** (Claude Code/Codex/Manus exigem credencial).

## Válvulas de honestidade (lembrete permanente)

- Resultado ausente = **`NAO_MEDIDO`**; provedor indisponível **não** é teste aprovado.
- `existe módulo` ≠ capacidade homologada; estágio com evidência ≠ `COMPLETED_VERIFIED`.
- Nenhuma superioridade sobre Manus/Claude/Codex/Lovable/etc. é alegada **sem métrica independente**.
- O SHA medido envelhece a cada merge: registra-se **quando** e **qual revisão** foi medida.
- Chaves coladas em chat são **COMPROMETIDAS** até revogação comprovada pelo proprietário.

## Lições de processo acumuladas (para não repetir)

1. **Gerar derivados → validar → commitar** (nesta ordem).
2. **Nunca cortar branch de uma `main` sem o PR anterior mesclado** — numeração e contagens quebram.
3. **`git push -u`** em branch nova; **confirmar com `ls-remote`**; **nunca reusar nome de branch já mesclado**.
4. **Contrato testado do repositório ganha da preferência do agente** (ADV-19): revertar e **declarar**.
5. **Patcher genérico sobre código desconhecido é arma carregada**: whitelist + assinatura de uma linha + **zero certo
   por tipo**; o **compilador é o árbitro** das arities.
6. **Nunca `git reset --hard` em branch de trabalho**; reconciliar por `cherry-pick` (**sem force-push**).
7. **Hipótese não é diagnóstico; sonda é** (ADV-24): separar hipóteses por medição, não por plausibilidade.
8. **Instrumento que reconhece um único padrão transforma código certo em dívida** (ADV-21) — e o **desconhecido não
   pode passar em silêncio** (E-352).
9. **Limite que não está escrito não é limite: é surpresa** (E-353).
10. **Não declarar verificado o que não se observou**; e **não relatar artefato que não existe** (o PR precisa ser
    **criado** e a criação **confirmada**).

---

## PROXIMA ACCAO - V3.0 (RONDA 71)

> **Requisito**: o `mission_check` exige que este documento cite evidencia **recente** (>= `E-960`).
> Ele parava no `E-959`. **Isto e' o ESTAGIO 1** (*«actualizar inventario e registar divergencias»*).

### A accao unica, agora
**Fechar a QUINTA instancia do defeito «mede o disco»**: os `DIVIDA_STRING` e os tectos de
comentario do `scripts/hades_ascii_ps1_check.py` medem o **disco**, e o CI mede o **checkout**
(*`refs/pull/N/merge`*). *A cura e' a mesma do `E-997` - ler o commit - e ela ja' esta' escrita no
`rastreados_ps1()`, mas os dois dicionarios de tecto ainda sao comparados contra o resultado.*

*Alternativas, em ordem de custo:* **§ESTAGIO 21** (benchmarks reproduziveis), **§ESTAGIO 10**
(Mission Control unico), **§FASE-6** (instalador/`HadesSetup.exe`, que tambem fecha a divida dos
34 caracteres do `install.ps1`).

### O que acabou de fechar (E-960 .. E-1000)
- **§ESTAGIO 22**: `E-987` (snapshot real), `E-988` (manifesto), `E-989` (catraca), `E-993`/`E-994`/
  `E-996` (as 4 volateis e a inversao da regra), `E-997` (o inventario media o disco).
- **§FASE-5**: `E-990` (inventario da marca: 909 canonicas / 441 legadas) e `E-991` (o gerador com
  8 excecoes declaradas).
- **Instrumentos**: `E-992` (ASCII nos `.ps1`), `E-995` (`E-559` restaurado do Git),
  `E-998` (quinta instancia), `E-999` (`\n` literal), `E-1000` (evidencia sem commit).

### Estado medido
```
HEAD 31be202d6 · branch fix/checker-relogio · main 4b3e21854 · WIP 14 · stashes 0
indice 793 entradas (E-001..E-1000) · bateria 16 OK / 0 FAIL
CI: 6/6 verdes no d1f4e7915 (incluindo `test`, que era o que mais reprovava)
PR #438: OPEN · mergeState=BLOCKED · reviewDecision VAZIO (FALTA REVISAO HUMANA)
```

### Bloqueios externos reais (nao mascara-los como internos)
```
race / go test -race   CGO_ENABLED=0 e mingw-w64 ausente -> w64devkit + CGO_ENABLED=1
PACKAGE / INSTALL      nenhum artefacto do candidato    -> FASE-6
revisao humana         reviewDecision vazio no #438     -> o dono designa revisor (nao LMPrado-DZ23)
```

### Divergencias factuais registadas (nao reescrever historia)
1. O indice parava em `E-905` no `ROUND 256` e esta' em `E-1000`; a faixa `E-571..E-777` (**207**
   IDs) continua **ausente e declarada** como `EVIDENCE_LOST_HISTORICAL`.
2. **`E-998` citava codigo nao commitado** - corrigido no `31be202d6` (`E-1000`).
3. O `_tmp/bateria_local.ps1` citado neste documento **nao existe no repositorio** (foi removido):
   a bateria canonica e' `scripts/hades_local_gates.ps1` mais os 4 `--check` do `E-987`..`E-997`.

---

## ACTUALIZACAO V3.0 (RONDA 86) - O QUE MUDOU DESDE O `E-1000`

> **Requisito**: a catraca `hades_handoff_window_check.py` (`E-1008`) mede a **janela** de
> citacao -- *o maior ID citado tem de estar a menos de **12** evidencias da cabeca do indice*.
> Este documento parava no `E-1000`, e a cabeca estava em `E-1008`: **8 de atraso**, medidos.
> *A correccao fecha o `atras` para **0**.*

### As 8 evidencias novas, e o que cada uma fechou
``````
E-1001  os 3 documentos de retomada paravam no E-959 (42 de atraso)   aec62f0eb
E-1002  a divida declarada A MAIS (a 5a instancia ja' estava curada)  aec62f0eb
E-1003  o E-1002 PARTIDO em 2 linhas - e NENHUM checker viu          2e3de0dc2
E-1004  o normalizar so' apanhava METADE das tabelas de contagem      55d69ea5d
E-1005  as 5 strings do FASE-5 + o marcador que mutilava `DZ23`       a10e52cdc
E-1006  o E-1005 partiu o passo 8 -> comparar CONJUNTOS, nao listas   b1f1f0d86
E-1007  a catraca da FORMA do indice (5 regras, fault injection)      dbadd69b6
E-1008  a JANELA de citacao dos documentos de retomada (janela 12)    db1ee1fae
``````

### Estado medido nesta ronda
``````
HEAD db1ee1fae · branch fix/checker-relogio · main 4b3e21854 · WIP 13
indice 801 entradas (E-001..E-1008) · bateria 19 OK / 0 FAIL
CI: b1f1f0d86 e dbadd69b6 sem failure · db1ee1fae a correr com os 3 passos novos
    (inventario, FORMA do indice, JANELA de citacao)
PR #438: OPEN · mergeState=BLOCKED · reviewDecision VAZIO (FALTA REVISAO HUMANA)
``````

### As 6 causas de CI vermelho, todas curadas e provadas
``````
run 38922978710  passo 6  gate_head/gate_branch volateis      -> E-993
run 38924032504  passo 7  source_sha/source_branch            -> E-994
run 38926745194  passo 7  host Windows vs Linux               -> E-996
run 38932562362  passo 8  inventario media o DISCO            -> E-997
run 38939790452  passo 8  normalizar so' via 2 colunas        -> E-1004
run 38941538186  passo 8  o E-1005 partiu-o (tabela aberta)   -> E-1006
``````

### Licoes medidas nesta janela
1. Um checker que conta LINHAS nao ve' uma linha que NAO E' LINHA (`E-1007`).
2. Um risco DECLARADO e' uma medicao a fazer -- e as 2 declaracoes cumpriram-se (`E-1006`).
3. Uma cura pode ter o MESMO defeito que cura (`E-1005`).
4. Sobrestimar tambem e' defeito (`E-1002`).
5. Um limiar que mede «SE HA» nao mede «QUAO RECENTE» (`E-1008`).
6. Um detector de texto nao distingue uma MENCAO de uma OCORRENCIA (`E-1007`, `E-1008`).
7. Uma catraca sem teste e' uma crenca -- o teste do `E-1007` deu 3 defeitos da propria.

### Bloqueios externos reais
``````
race / go test -race   CGO_ENABLED=0, mingw-w64 ausente
pacotes / instalador   FASE-6 nao iniciada
revisao humana         reviewDecision VAZIO no PR #438
``````

### A proxima accao unica (actualizada)

**Empacotar a extensao `.zip` nao publicado, com SHA-256 e instrucoes de instalacao/remocao** --
depois de a extensao ficar carregavel (`E-1021`, 24 testes verdes). O `.zip` e' trabalho interno;
**publicar** na Chrome Web Store / Edge Add-ons exige conta e autorizacao do proprietario.

Ordem para os passos seguintes, sem inventar atalhos:
1. `.zip` da extensao + `SHA256SUMS` + `LEIA-ME` de instalacao e remocao;
2. pairing real (native messaging + `HADES Local Companion`) -- ainda **nao existe**;
3. E2E em Chrome/Edge reais (este host consegue, com uma sessao de browser controlada);
4. `E-1021` fechou o `Browser Bridge`; o `Computer Use` continua `PARTIAL`.

### A proxima accao unica (janela `E-1022`)

**O pacote da extensao esta fechado** (`E-1022`): `.zip` nao assinado de 84,3 KB, SHA-256
`6125f72adc9812ae...`, reproduzivel byte-a-byte, com `SHA256SUMS` e `LEIA-ME` ASCII, e um
verificador (`scripts/hades_browser_bridge_package.py`) que **reprova tres defeitos reais pela
regra propria** -- incluindo o separador invertido que o .NET escreve e o Chrome recusa.

**A proxima accao unica**: **provar o handshake sem navegador** -- escrever o contrato do canal
local (schema, sequencia, expiracao, replay) e um teste de contrato que o valide, porque **o
pairing nao existe**: nao ha native messaging nem `HADES Local Companion`, e o popup mostra
`host: - (native host nao instalado)`. Isto e trabalho interno, sem credencial.

Depois, pela ordem: (2) SBOM do pacote da extensao; (3) E2E em Chrome real (este host consegue,
com sessao de browser controlada); (4) Firefox (`.xpi`) ou `ABSENT` documentado; (5) script e doc
de notarizacao/assinatura (pedido pelo dono, ainda nao escrito).

### A proxima accao unica (janela `E-1023`)

**O contrato do canal local esta fechado e testado** (`E-1023`): 23 assercoes em
`apps/hades-browser-bridge/test/canal_local.test.js` provam que o pairing futuro nao pode
nascer com token estatico, replay, ligacao LAN, instalacao desconhecida, chave expirada ou
mensagem fora de ordem. Dois desses testes apanharam defeitos reais meus: o loopback IPv6
estava recusado (o `URL` do Node devolve `[::1]` com brackets) e o detector de `Math.random`
acusava uma **string de erro** que menciona a funcao para a proibir.

**A proxima accao unica**: **acordo de chaves e transporte** -- o contrato assume que a chave
simetrica ja foi partilhada com seguranca, e isso **nao esta feito**. Escrever a derivacao
(por exemplo HKDF a partir do codigo curto, com o codigo a morrer depois de usado) e o
manifesto do native host com a allowlist de `allowed_origins` restrita a extensao HADES.
Continua a ser trabalho interno, sem credencial.

Depois, pela ordem: (2) DNS rebinding a nivel de socket; (3) SBOM do pacote da extensao;
(4) E2E em Chrome real; (5) Firefox (`.xpi`) ou `ABSENT` documentado.

### A proxima accao unica (janela `E-1024`)

**A chave do canal ja deriva do codigo curto** (`E-1024`): HKDF-SHA256 com `salt` e
`info` do contexto, codigo de uso unico, native host com allowlist restrita e
`nativeMessaging` em `optional_permissions`. 56 assercoes nos 3 ficheiros de teste, e a
discriminacao provou que o teste do contexto reprova o defeito do `E-1023`.

**A proxima accao unica**: **proteger o codigo em TRAFEGO** -- o HKDF assume que o codigo
curto atravessou um canal ja autenticado, e num canal `loopback` sem TLS o codigo de 6
digitos viaja em claro no `localhost`. Definir e testar: (a) o host recusa arrancar sem um
handshake que prova posse da chave derivada, (b) o codigo so vive em memoria no host,
(c) rate-limit por pedido e por instalacao. Tudo interno, sem credencial.

Depois: (2) o executavel do `HADES Local Companion` (stdio, com o manifesto ja escrito);
(3) DNS rebinding a nivel de socket; (4) SBOM do pacote; (5) E2E em Chrome real.

### A proxima accao unica (janela `E-1026`)

**O handshake existe e discrimina** (`E-1025`): desafio de 32 bytes com prazo de 30 s, prova de
posse, taxa de 20 tentativas por instalacao numa janela de 60 s, e o nonce morre so' depois de a
prova conferir. 67 assercoes nos 3 ficheiros de teste.

**A proxima accao unica**: **o TRANSPORTE** -- o handshake prova **identidade**, nao **sigilo**.
O codigo curto continua a atravessar o `localhost` em claro no primeiro salto. Falta escolher e
implementar: (a) native messaging por `stdio` (o manifesto ja' existe como template), com o
Companion a ler/escrever mensagens de 4 bytes de comprimento + JSON; ou (b) `loopback` com TLS e
certificado efemero. Tudo interno, sem credencial.

Antes disso, dois defeitos de higiene que o `E-1026` expos: a taxa vive **em memoria** (reiniciar
o host apaga a contagem) e nao ha rate-limit do lado do SO.

### A proxima accao unica (janela `E-1028`)

**O transporte existe e o protocolo atravessa-o** (`E-1027`): enquadramento de native messaging
com 4 bytes little-endian, erros tipados, corpo parcial que nao e' erro, e um canal em memoria
que leva pairing -> handshake -> envelope -> replay recusado, com o fluxo partido em pedacos de
1 byte. 84 assercoes nos 4 ficheiros.

**A proxima accao unica**: **o processo `HADES Local Companion`** -- o enquadramento esta' pronto e
testado, mas **nunca correu contra `process.stdin`/`stdout` de um processo real**. Fazer o
executavel minimo (Node ou Go) que: le' do `stdin` com `desenquadrar`, responde no `stdout` com
`enquadrar`, usa o `estadoInicial()` do contrato, e implementa **so'** o handshake. Depois um teste
que o **arranca como subprocesso** e fala com ele pelo fio -- sem socket, sem mock. Tudo interno,
sem credencial.

Antes disso, o que o `E-1028` expos: a taxa vive em memoria, e nao ha' rate-limit do lado do SO.

### A proxima accao unica (janela `E-1030`)

**O Companion existe como processo** (`E-1029`): o transporte e o handshake atravessam `stdin`/`stdout`
reais, com o fio partido em bytes e a correlacao por `id` a funcionar. 93 assercoes nos 5 ficheiros.

**A proxima accao unica**: **instalar o host de verdade** -- gerar o manifesto de native messaging com o
ID real da extensao e um instalador/desinstalador reversivel, e provar que o navegador arranca o
Companion. Sem isso, `allowed_origins` continua **template** e nao ha' prova de que o SO/navegador
respeita a fronteira. Tudo interno; a instalacao e' local e reversivel.

Antes disso, o que o `E-1029` expos: a taxa e o estado do pairing vivem **em memoria** -- reiniciar o
processo apaga-os. Persistir com escopo e apagar ao revogar e' o passo seguinte.

### A proxima accao unica (janela `E-1031`)

**O ID da extensao deixou de ser adivinhacao** (`E-1031`): e' derivado da chave publica
versionada (`lbndhojeaplfofjmhdmlbocabojakddb`), o `manifest.json` tem `key`, e o native host
**instala-se e desinstala-se de verdade** no registo do Windows -- medido: `registo SUJO: 0`.

**A proxima accao unica**: **exercitar o host POR UM NAVEGADOR**. Instalei-o, mas **nunca abri
o Chrome**. A prova que falta e' o E2E real: carregar a extensao em `chrome://extensions`,
confirmar que o ID mostrado e' o esperado, e falar com o Companion por native messaging a serio.
Sem isso, o que existe e' um protocolo correcto testado por subprocesso -- e nao um produto
instalado num navegador.

Antes disso: persistir a taxa e o estado do pairing (hoje vivem **em memoria**, `E-1029`).

### A proxima accao unica (janela `E-1032`)

**O protocolo atravessou um navegador a serio** (`E-1032`): o Edge real carrega a extensao,
deriva o **mesmo ID**, e a extensao falou com o Companion por native messaging -- resposta de
ida e volta, com o `id` ecoado. Exit 0.

Pelo caminho cairam **4 defeitos que nenhum teste anterior podia ver**:
- `nativeMessaging` estava declarada mas **ninguem a pedia** (e sem ela `sendNativeMessage`
  **nao e' uma funcao**);
- `chrome.permissions.request` **trava para sempre** dentro do handler (curado com prazo de 8s);
- o gesto **nao sobrevive** ao `sendMessage` -- o pedido tem de viver no popup;
- o Windows **nao executa `.mjs`** -- o `path` tem de apontar para um lancador `.cmd`.

**A proxima accao unica**: **o controle de aba a funcionar ponta a ponta**. Hoje o E2E prova o
**canal**; nao prova **observar o DOM de uma pagina real, clicar, e recusar um campo de senha**.
E' um teste com uma pagina local e a extensao a fazer `browser:navigate` -> `browser:files`
com a concessao por aba, o badge, o takeover e a revogacao. Tudo interno.

### A proxima accao unica (janela `E-1033`)

**O controle de aba funciona ponta a ponta** (`E-1033`): observar o DOM real, escrever
num campo legítimo (**o valor chega ao DOM**), **recusar senha e cartão**, badge `ON`,
revogar. Exit 0.

E caiu uma **contradição interna** que nenhum teste de unidade via: o protocolo tinha o
opt-in de loopback -- e **ninguém o chamava**; e o `content_scripts.matches` cobria
**só** `127.0.0.1`/`localhost`, as origens que o protocolo **sempre** recusava. *A única
origem onde o content script era injectado era a única que o HADES não podia usar.*

**A proxima accao unica**: **o takeover humano**. O `browser:takeover` existe no
protocolo e é o scope que falta exercitar: enquanto o utilizador digita, o agente **não**
disputa o campo; pausar; devolver a sessão; e a revogação por **mudança de origem**
(que tem código e não tem E2E). Tudo interno, na mesma página local.

### A proxima accao unica (janela `E-1034`)

**O takeover humano existe e funciona** (`E-1034`): quando o utilizador digita ou clica, o
agente para -- e **o valor do humano não é sobrescrito**. O scope `browser:takeover` estava
declarado e não tinha uma linha de implementação.

Pelo caminho caiu um achado que só o E2E podia ver: **a automação accionava o próprio
travão** -- o `preencher` do agente despacha `input`, e o listener de takeover contava isso
como gesto humano. O agente escrevia **uma vez** e perdia o controlo por sua própria acção.

**A proxima accao unica**: **a revogação por mudança de origem**. Tem código no
`tabs.onUpdated` e **não tem E2E**: navegar de `127.0.0.1` para outra origem deve revogar a
concessão, limpar o badge e recusar a acção seguinte -- e o mesmo para outra **aba** com a
mesma origem (que deve continuar concedida). Tudo interno, na mesma página local.

### A proxima accao unica (janela `E-1035`)

**A revogação por mudança de origem funciona** (`E-1035`): a aba que navega para outra
origem perde a concessão, o badge limpa e a acção é recusada -- e a **aba vizinha, na
mesma origem, continua concedida**. Voltar não restaura.

E o E2E expôs o **terceiro** local do mesmo defeito: o `tratarAccao` validava a URL do
alvo **sem o opt-in** -- `curar dois de três sítios deixa o terceiro a recusar o que os
outros aceitam`. O sintoma era incoerência: a mesma origem concedia e depois recusava.

**A proxima accao unica**: **procurar os chamadores, não o sintoma**. Fazer um teste que
**falhe se um novo local validar URL sem o opt-in** -- isto é, uma catraca sobre o
**código**, e não sobre o resultado. O defeito apareceu três vezes; a cura pontual não o
fecha. Tudo interno.

### A proxima accao unica (janela `E-1036`)

**A catraca sobre o código existe** (`E-1036`): todo o chamador de `validarUrl`/
`guardarRota` tem de passar `permitirPrivado`, e ela **discrimina** -- verde com a cura,
vermelha sem ela. O defeito do opt-in apareceu em **três** locais em três rondas, cada vez
com a cura pontual do local que falhou.

⇒ ***uma cura pontual não fecha a classe do defeito.** O gate certo não mede o sintoma:
**mede o código que o produziria.***

**A proxima accao unica**: **o `pushState`**. Ficou registado como dívida no `E-1035` e
**pode ser um buraco real**: um SPA que mude de origem por History API **não** dispara
`tabs.onUpdated` com URL novo, logo a concessão **não** seria revogada. Medir primeiro:
um teste que navegue por `pushState` e verifique se a concessão sobrevive indevidamente.
Tudo interno, na mesma página local.

### A proxima accao unica (janela `E-1037`)

**A dívida declarada no `E-1035` era um não-problema** -- `pushState` **não pode** mudar de
origem: o browser lança `SecurityError`. Mas **ao medi-la encontrei dois defeitos reais**:

1. **`onUpdated` sem URL revogava a concessão.** O browser emite `changeInfo` com **só
   `status`** (sem `url`) a cada `pushState`, `replaceState`, **reload** e até mudança de
   título. O código validava a **string vazia**, falhava, e concluía «nova origem».
   ***A navegação mais comum que existe -- recarregar -- matava a sessão.***
2. **O badge mentia.** Com a concessão viva, o Chrome limpava o badge ao navegar e ele
   ficava vazio. ***Um indicador que não reflecte o estado é pior do que indicador nenhum.***

⇒ ***O valor de medir uma dúvida não está na resposta a essa dúvida: está no que aparece
pelo caminho.***

**A proxima accao unica**: **o BFCache**. Ficou sem cobertura e **pode repor uma página
de outra origem** ao voltar atrás -- se o browser restaurar do cache sem emitir `onUpdated`
com URL, a concessão pode viver sobre conteúdo de outra origem. Medir com dois servidores
e uma navegação real seguida de `goBack`. Tudo interno.

### A proxima accao unica (janela `E-1038`)

**O BFCache não revive a concessão** (`E-1038`), e agora sei **porquê**: o regresso traz
**URL** no `onUpdated` -- `{chaves:["status","url"], ...}` -- logo o handler **avalia-o**
contra a origem concedida. E não há restauração automática: 4 passos de ida-e-volta, a
concessão cai em todos e não revive em nenhum.

⇒ ***Mas o limite é tão importante como o resultado: as marcas da página deram `pageshow`
com `persisted: false` -- o browser **recarregou**, não restaurou do cache. Esta medição
não exercitou o BFCache. Um resultado favorável não é uma cobertura.***

**A proxima accao unica**: **a familia `onUpdated` está fechada; mudar de superficie.**
O que falta e é interno e executável: **os frames e o shadow DOM** -- o contrato exige
suportá-los «com limites», e o `observar` actual só olha o documento principal. Medir
primeiro o que acontece hoje com um `iframe` de mesma origem e com um componente de
shadow DOM: o `observar` vê-os, ignora-os, ou mente sobre eles? Tudo interno.

### A proxima accao unica (janela `E-1039`)

**O `observar` devolvia uma lista parcial como se fosse completa** (`E-1039`). Medido:
uma página com **3** controlos reais devolvia **2 nós** e `total: 2`.
***Um observador que devolve parcial como completo faz o modelo agir sobre uma página
que não é a que existe.*** Não era «não atravessar frames» — era **não dizer**.

Agora **atravessa** shadow roots abertos e iframes de mesma origem (2 → **5** nós) e
**declara** `limites` com contagens, profundidades e o que ficou de fora.

E a minha primeira cura **falhou de forma instrutiva**: contava shadow roots fechados
procurando custom elements com filhos — e devolveu `0` para uma página que **tinha** um.
***Um palpite que falha é pior do que uma declaração de ignorância: dá uma contagem que
parece medida.*** Agora diz `"nao_detectaveis_por_design"`.

**A proxima accao unica**: **downloads e uploads**. É a última capacidade do escopo da
FASE-3-B sem cobertura: o contrato exige «caminhos autorizados e manifestos com
tamanho/hash», e aprovação para download executável. Medir primeiro o que existe hoje:
o `preencher` recusa `type=file`, e **nada** gere downloads. Tudo interno.

### A proxima accao unica (janela `E-1040`)

**O scope `browser:files` não tinha uma linha de implementação** — e a mensagem do
`preencher` prometia um `caminho_autorizado` que não existia. Agora **decide**: um
executável **nunca** fica «permitido» (nem com aprovação), e o **mesmo** `a.txt` inerte
é `permitido` em download e `exige_aprovacao` em upload. ***Sair do disco é sempre um acto.***

**Duas lições que valem mais do que o código:**
1. **O canal não transporta binário.** `sendMessage` serializa em JSON — um `Uint8Array`
   atravessa sem `byteLength`. O conteúdo de um ficheiro **não passa por aqui**.
2. **A lista fixa do `lerEstado` ia morder outra vez** — terceira ocorrência. Curada na
   raiz: agora preserva o que não conhece. *Curar o campo que falhou não cura a lista.*

**A proxima accao unica**: **ligar `podesTransferir` às approvals reais do Mission
Control**. Hoje a aprovação chega por `mensagem.aprovado` — um campo que **quem pede**
controla. *Quem pede não certifica o que pede.* Medir a rota de approvals existente
(`internal/agent/service_agent_approvals.go`) e ligá-la, ou declarar honestamente que a
extensão decide sem autoridade de aprovação.

### A proxima accao unica (janela `E-1041`)

**A aprovação da extensão era um campo que quem pede controla** (`aprovado: true`).
Agora é uma **prova estruturada** — `{aprovacaoId, fingerprint, aprovadoPor}` — e o
`fingerprint` tem de **casar com o pedido**. ***Um campo de permissão que quem pede
controla não é uma permissão.***

E não inventei um sistema: o backend **já tinha** `FindApproved(org, fingerprint, now)`
e `Consume` (uso único). A extensão exige a **mesma forma de prova**.

⇒ ***Mas a fronteira é honesta: a extensão **não verifica** a prova contra o store — não
tem acesso. Uma prova bem formada mas falsa passaria. A forma está implementada e
testada; a autoridade não.***

**A proxima accao unica**: **o canal autenticado extensão ↔ runtime**. É o que falta para
a autoridade existir: o `nativeMessaging` já funciona (`E-1032`) e o Companion já fala o
protocolo. Medir se o Companion pode consultar `FindApproved` e devolver a prova — ou
declarar que sem a rota no runtime a extensão decide **sem autoridade** e assumi-lo no
contrato. Tudo interno.

### A proxima accao unica (janela `E-1042`)

**O canal não existe — e agora a decisão diz o que vale.** A rota de approvals existe
(`serviceAgentApprovals`), mas é **autenticada por sessão** — e o Companion é um processo
local **sem sessão**. ***Dar-lhe a sessão do utilizador seria pior: um processo local com
a sessão do utilizador é exactamente o que não queremos.***

Em vez de inventar a integração, **pus a fronteira dentro do contrato**:
`autoridade: "forma_apenas"` acompanha **todo** o veredicto, e `avisoDeAutoridade` diz
por palavras: *«Aprovação com FORMA válida, mas NÃO verificada … uma prova bem formada
mas inexistente passaria aqui.»*

⇒ ***Um limite que não se lê não é um limite. Um campo que diz `verificada` sem ter sido
verificado é pior do que não ter campo.***

**A proxima accao unica**: **a rota no Companion**. É o passo de desbloqueio exacto: uma
rota **no Companion** (não na extensão) que use o **canal emparelhado**
(`pairing:iniciar` / `handshake:*`, que já existem do `E-1031`) para se autenticar
perante o runtime e consultar `FindApproved`. Só então `verificada` tem quem o possa
afirmar. Tudo interno.

### A proxima accao unica (janela `E-1043`)

**A rota de aprovação existe ponta a ponta — e responde que não pode verificar.**
`aprovacao:consultar` no Companion, `arquivo:consultarAutoridade` na extensão: o caminho
completo extensão → native messaging → Companion. ***O que faltava não era a verificação:
era a pergunta.***

O Companion responde `{ok: true, verificada: false, motivo:
"sem_rota_autenticavel_no_runtime", desbloqueio: "expor FindApproved ..."}`.
⇒ ***Uma rota que responde «não posso» é melhor do que uma rota que não existe: quem
pergunta fica a saber que perguntou ao sítio certo.***

E **oito asserções** provam que ele **nunca** diz `verificada: true` — incluindo quatro
tentativas de o forçar.

**A proxima accao unica**: **a catraca do encaixe**. Hoje o passo de desbloqueio é a
constante `MOTIVO_SEM_ROTA`, **escrita mas não vigiada** — se o runtime expuser a rota,
nada falha. Fazer um teste que **falhe quando `FindApproved` passar a estar acessível**
(por exemplo, quando existir um endpoint loopback de approvals), para que o encaixe não
fique esquecido. Tudo interno.

### A proxima accao unica (janela `E-1044`)

**A catraca do encaixe existe.** No `E-1043` deixei o encaixe (`MOTIVO_SEM_ROTA`) e uma
frase num documento — ***e uma frase num documento não acorda ninguém***. Agora há um
checker que **acorda** se aparecer uma rota HTTP que chame `FindApproved` ou um handler
de approvals sem sessão — o sinal de que o Companion já pode verificar.

E falha **alto, com instrução**: *«troque `MOTIVO_SEM_ROTA` pela consulta real e eleve a
autoridade de `forma_apenas` para `verificada` — com teste que discrimine»*.

⇒ ***Uma catraca não é um obstáculo: é um lembrete que não se esquece.***

**A proxima accao unica**: **a família de extensão está fechada e vigiada; mudar de eixo.**
O que falta e é interno e executável: **os quatro achados P2 de integridade do PR #438**
(`scheduling_library_declared_test.go`, `declared_categories_measured_test.go`,
comentários obsoletos em `types.go`/`runtime.go`/`step_routing.go`, e o
`HADES_ADAPTIVE_COWORK_GAP_MATRIX.md`) — são findings de revisão válidos, não expandem
escopo, e fecham dívida de rastreabilidade. Tudo interno.

### A proxima accao unica (janela `E-1045`)

**Os 12 findings do PR #438 foram medidos um a um contra o HEAD.** Dez estão corrigidos
(#2–#6 no `E-909`, #7 no `E-912`, #9 no `E-910`) ou são **falsos positivos do bot**
(#10 e #11).

⇒ ***#11 — «pipe literal numa célula Markdown» — **não se reproduz**: renderizei as 837
linhas com o `marked` e **todas produzem 5 campos**. E a minha medição dele errou **três
vezes** (767, 303, 768) antes de eu usar um renderizador. ***Não se adivinha um
renderizador: usa-se um.*** Eu ia corrigir 768 linhas correctas.***

**A proxima accao unica**: **o #8 é o único não totalmente revalidado.**
`declared_categories_measured_test.go` foi inspeccionado **por nome de teste**, não pela
alegação contra o **comportamento real** do checker. Medir o que o teste afirma, correr o
checker e confrontar os dois — e corrigir a alegação se divergir. Tudo interno.

### A proxima accao unica (janela `E-1046`)

**O #8 está fechado por medição — e os 12 findings do PR #438 estão RESOLVIDOS.**
O teste afirmava que a catraca acusa «dispensa órfã **e obsoleta**»; o comentário deliberado
do próprio checker (`hades_tenant_scope_check.py:216-219`) diz que a premissa de **obsoleta**
estava **errada** e que a acusação foi **removida de propósito**. Já estava corrigido
(`E-912`) — e a contraprova que faltava foi correr o teste: **`ok … 5.620s`**.

**Balanço**: #1 tratado antes · #2–#9 **corrigidos** · #10 e #11 **falsos positivos do bot**
· #12 **feito por construção**. E **as 14 threads do GitHub continuam abertas** — proibido
resolvê-las sem autorização.

**A proxima accao unica**: **mudar de eixo para o ESTÁGIO 21 (benchmarks)**. É o único dos 10
estágios livres que nunca foi iniciado, e o prompt exige cenários reproduzíveis com hardware,
versão, modelo, contexto, warm/cold, repetições e erro. E há um ponto de partida honesto: o
Ollama local está alinhado em 16k de contexto e há modelos medidos neste host. Definir o
protocolo e correr **três repetições** com o que existe, sem inventar benchmark de concorrente.

### A proxima accao unica (janela `E-1047`)

**O ESTÁGIO 21 não estava por iniciar — estava por medir.** A minha própria nota dizia
«não iniciado»; tinha **13+ instrumentos**, método documentado e artefacto.

**Medido**: matriz T2 real com 2 modelos locais ⇒ ***`PASS` em 599,88 s***. Resultado:
**3/8 verde · 4 trapacearam · 1 vermelho**. E os **veredictos repetiram-se um dia depois**
(o artefacto anterior era de `2026-10-15T22:07Z`; o novo, de `2026-10-16T23:41Z`) — só os
tempos mudaram. ***Reprodutibilidade não é o tempo ser igual: é a conclusão ser igual
quando o tempo não é.***

E o `HARNESS_TRAFICOU` (`harness_benchmark.go:286`) apanhou **quatro** modelos a tentar
alterar o **teste** em vez do **código**.

**A proxima accao unica**: **revalidar a repetição** -- `HADES_T2_REPETICAO.md` (`46+/0`)
é do WIP e traz números que **eu nunca corri**. O `RepetirT2Matrix` existe
(`harness_benchmark_repetition.go`); correr com `HADES_T2_MEDIR_MODELO=1` e comparar o
`46+/0` com o que sai agora. Se divergir, o artefacto está desactualizado e deve ser
regenerado; se coincidir, a repetição ganha a mesma reprodutibilidade que a matriz.

### A proxima accao unica (janela `E-1048`)

**ZERO WIP real por integrar.** As «14 entradas» que reportei durante dezenas de rondas
eram **11 `M` de fim-de-linha** + 2 não rastreados + os 2 T2 que commitei.
`git diff --stat` = **0** e `--ignore-cr-at-eol` = **0**: ***num `git status` um `M` não é
necessariamente uma mudança de conteúdo.***

⇒ ***«Reportar o número certo da coisa errada é uma forma de mentir sem dizer uma palavra
falsa.»*** *Uma contagem não é um inventário.*

**Curado**: `scripts/hades_worktree_inventory.py` classifica em três categorias e **imprime
cada caminho** — passo 31 da bateria.

**A proxima accao unica**: **revalidar a repetição T2** (`46+/0`), cuja corrida está
lançada em background com `qwen2.5:7b` e 3 repetições — o mesmo que o artefacto antigo
(`2026-10-13T13:41:29Z`), para comparação directa. Enquanto corre, **não duplicar** o
trabalho que ela já faz.

### A proxima accao unica (janela `E-1049`)

**A repetição T2 foi revalidada: o `46+/0` está confirmado.** Corrida: `PASS` em
209,78 s. Os veredictos são **idênticos três dias depois** (soma `3/3`; comparação,
caso-limite e fatiamento `0/3` com trapaça nas três) — só as durações mudaram.

⇒ *O artefacto estava desactualizado (`2026-10-13`) e eu tinha-o commitado assim, com a
dívida declarada. **Agora a data é `2026-10-16T23:52:54Z` e o número é meu.***
⇒ ***«Declarar a dívida no commit não a paga; correr o teste paga.»***

**A proxima accao unica**: **o estágio 21 tem uma lacuna que não é medição — é
integração.** O `HADES_BENCHMARK_METHOD.md` e o `HADES_STAGE_21_BENCHMARK.md` descrevem
medições **sem referenciar** a matriz T2 real e a repetição (que agora são medidas do
mesmo snapshot). Verificar se os documentos canónicos do estágio apontam para os
artefactos reais e corrigir a fonte onde divergirem. Tudo interno.

### A proxima accao unica (janela `E-1050`)

**O vermelho do `pr-gate` está explicado, e não é defeito de código.** No `55327160c`, os
gates `test` e `dz23-agentic-quality` foram **cancelados** (pelo push seguinte) e o
`pr-gate.yaml:91` ficou a **esperar uma substituta** até ao deadline de **30 min**
(`:61`) — ***«Timeout aguardando os gates profundos deste SHA»***.

**A prova são dois SHAs, os mesmos gates:** `55327160c` (23:19:46Z) ⇒ `test` e
`dz23-agentic-quality` **cancelled** ⇒ `pr-gate` **failure**. `4157a459b` (23:28:43Z) ⇒
os mesmos dois **success** ⇒ ***6 runs, 6 success***.

⇒ ***«Um push rápido garante um vermelho que não é um defeito.»*** E o `cda756f82` já
tem os dois cancelados — **vai repetir o padrão**.

**O que NÃO fiz**: aceitar `cancelled` como aprovação no `pr-gate`. O owner proibiu-o por
palavras, e **a gate está certa em reprovar**: não pode certificar um SHA cujos gates
não correram.

**A proxima accao unica**: **não fazer push enquanto as runs do SHA corrente não
terminarem** — e verificar se `ba62baf88` (HEAD, 6 runs em curso) fecha com
`failure=0`, agora que sei o que o mata. Se fechar 6/6, o mecanismo fica provado por
**três** SHAs em vez de dois.

### A proxima accao unica (janela `E-1051`)

**O mecanismo do `pr-gate` está provado com TRÊS SHAs.** `55327160c` (gates cancelados)
⇒ `pr-gate` **failure**; `4157a459b` e `ba62baf88` (gates terminados) ⇒ **6/6 success**.
A previsão do `E-1050` foi escrita **antes** do terceiro SHA fechar — e cumpriu-se.

E o comportamento mudou por causa dela: o commit `67581a993` foi feito **localmente** e só
enviado **depois** do `ba62baf88` fechar 6/6. ***A disciplina de push deixou de ser uma
frase no handoff e passou a ser um procedimento seguido.***

**A proxima accao unica**: **fechar o estágio 21 com os documentos canónicos.** O
`HADES_BENCHMARK_METHOD.md` e o `HADES_STAGE_21_BENCHMARK.md` descrevem medições **sem
referenciar** a matriz T2 real e a repetição — que agora são medições do mesmo snapshot,
verificadas. Verificar se os documentos canónicos apontam para os artefactos reais e
corrigir a fonte onde divergirem. E, antes de qualquer push, **esperar que as runs
terminem**.

### A proxima accao unica (janela `E-1052`)

**A previsão cumpriu-se.** O `E-1051` escreveu que o `cda756f82` reprovaria pelo mesmo
mecanismo; reprovou (`pr-gate` failure, `test` e `dz23-agentic-quality` cancelled).
E a disciplina de push deu **quatro SHAs 6/6 consecutivos** — sem tocar numa linha da gate.

⇒ ***«Uma previsão escrita antes é a única prova de que a explicação serve.»***
⇒ ***«O problema tinha solução procedural, e a solução foi executada.»***

**A proxima accao unica**: **fechar o estágio 21 com os documentos canónicos.** O
`HADES_BENCHMARK_METHOD.md` e o `HADES_STAGE_21_BENCHMARK.md` descrevem medições **sem
referenciar** a matriz T2 real (`E-1047`) nem a repetição (`E-1049`) — que agora são
medições verificadas do mesmo snapshot. Verificar se os documentos canónicos apontam para
os artefactos reais e corrigir a fonte onde divergirem. **E, antes de qualquer push,
esperar que as runs do SHA remoto terminem com `failure=0`.**

### A proxima accao unica (janela `E-1053`)

**O estágio 21 está ligado aos seus artefactos.** Os três documentos canónicos
descreviam as medições e **nenhum chegava ao resultado** — um auditor que lesse o estágio
não chegava ao `3/8 verde`.

⇒ ***«Um método que não aponta para o artefacto é um método que o auditor não segue.»***
⇒ ***«O trabalho estava feito; a prova não estava ligada.»***

**Curado**: `HADES_STAGE_21_BENCHMARK.md` ganhou a **secção 6** (tabela `artefacto →
comando → evidência`), e os outros dois ganharam a ligação **dentro do bloco de
citação** — a primeira tentativa criou dois blocos partidos em Markdown, e foi corrigida.

**A proxima accao unica**: **fechar a lacuna de rastreabilidade com um teste**, não com
texto. Hoje a ligação é textual: **se alguém renomear `HADES_T2_MATRIX.md`, nada acusa.**
Um checker que verifique que os documentos do estágio 21 citam os artefactos (e que os
artefactos existem) fecha isso — e é o mesmo padrão da catraca do encaixe (`E-1044`).
Tudo interno, e **esperar que as runs terminem antes de qualquer push**.

### A proxima accao unica (janela `E-1054`)

**A ligação deixou de ser texto.** No `E-1053` liguei os documentos do estágio 21 aos
artefactos — com texto — e escrevi a dívida: *«se alguém renomear `HADES_T2_MATRIX.md`,
nada acusa»*. Agora acusa: `hades_stage_artifacts_check.py` (**passos 32 e 33**),
**18 asserções**, **4 injeções** e uma contraprova, numa cópia.

⇒ ***«Ligar com texto não é ligar: é descrever a ligação.»***

**E antes de escrever, medi**: existe `hades_discoverability_check.py` — li-o e corri-o;
protege a proibição de **prometer ranking**, não a ligação. ***Reutilizar primeiro não é
uma regra de estilo: é a diferença entre acrescentar e duplicar.***

**A proxima accao unica**: **estender o mapa `LIGACOES`** — hoje vigia só o estágio 21
(3 documentos, 2 artefactos). A mesma pergunta aplica-se a **outros** estágios que
produzem artefactos: verificar quais documentos canónicos citam artefactos reais e quais
não citam, e declarar cada ligação que falta. Medir primeiro; só depois escrever.

### A proxima accao unica (janela `E-1055`)

**Fui procurar ponteiros quebrados em 131 artefactos e 143 documentos: deu ZERO defeitos
reais e QUATRO erros meus.** Comparava `basename` contra caminhos, o `fullmatch` falhava
em nomes longos, e tratava ficheiros gerados em runtime e itens «A CRIAR» como quebrados.

⇒ ***«Quando uma medição produz muitos achados de uma vez, a primeira hipótese é a
medição.»*** — terceira vez nesta janela (`E-1044`/`#11`).
⇒ ***«Um mapa com falsos positivos é pior do que nenhum mapa, porque ensina a ignorá-lo.»***

**A decisão foi NÃO alargar a lista** com os 10 falsos: o `E-1054` continua com as
mesmas 6 entradas verificadas.

**A proxima accao unica**: **medir a pergunta certa** — *quais documentos **afirmam
produzir** um artefacto e **não o citam?*** Não «quais citam zero» (113 dos 143 são
narrativos e não têm de citar). A pergunta é sobre **afirmação**, não sobre ausência:
procurar nos documentos frases que nomeiam um artefacto como produzido por eles, e
verificar se o ponteiro está lá. Tudo interno, e **esperar as runs antes de qualquer push**.

### A proxima accao unica (janela `E-1056`)

**Fiz a pergunta certa — *quais documentos **afirmam produzir** um artefacto e não o
citam?* — e deu SETE falsos positivos, todos do mesmo erro: o regex lê «está em» e «a
partir de» como se fossem «gera».**

Verificados um a um: `eas.json` **existe**; `_tmp/rel_a.md` é **gitignored**;
`HADES_MASTER_STATE.json` é a **entrada** do gerador, não a saída; e os outros quatro são
**citações**, não produções.

⇒ ***«Um detector léxico não vê a relação: vê os termos.»***
⇒ ***«O resultado de procurar e não encontrar é «não encontrei» — e não «não existe».»***

**As duas perguntas úteis foram feitas e respondidas (zero). A linha de investigação
encerra aqui.** — ***«A diligência sem hipótese vira ruído, e ruído num gate é pior do que
um gate a menos.»***

**A proxima accao unica**: **voltar a construir em vez de auditar.** A janela 142 provou
que o estágio 21 mede (3/8 verde, veredictos reproduzem) e que a ligação vigia. O trabalho
que falta é **de produto**, não de rastreabilidade: os artefactos não assinados do
candidato actual (§FASE-6). O Windows é o único que o toolchain deste host consegue
produzir ponta a ponta — e há `scripts/hades_bundle_windows.ps1` no WIP integrado.
Medir primeiro o que ele faz; depois correr. **E esperar as runs antes de qualquer push.**

### A proxima accao unica (janela `E-1057`)

**O artefacto do candidato existe.** `Hades-6093a6692-windows-amd64-portable-completo.zip`
— **48 234 478 bytes**, verificado pelo próprio script: `/api/tags` 200, `/api/chat` 200
com resposta **real** (`[Tudo certo!]`), `/api/v1/model/upstream` **501** (contrato honesto).

⇒ ***«Um artefacto que declara os seus limites vale mais do que um que os omite.»***

**E o caminho até ele revelou dois achados**: o `TMP` negado (curado **na chamada**, não no
script) e **17 processos `llama-server` órfãos** de dez verificações anteriores — o script
matava o `hades.exe` mas **não o filho**, e o filho segurava os DLLs.
⇒ ***«Um teste que arranca um filho e mata só o pai deixa o filho a segurar o diretório.»***
**Curado e provado**: `runner parado 1 proceso(s)`, e **0 órfãos** depois.

**A proxima accao unica**: **o `HadesSetup.exe` não assinado** — o `build_windows.ps1`
(1369 linhas) e o `app/ollama.iss` (223 linhas) existem, e o `ISCC.exe` **existe e
funciona**. O `LEIA-ME` do bundle já o diz: *«o que falta não é a ferramenta: é EXECUTAR o
instalador»*. **Gerar** o instalador é interno; **instalá-lo** escreve em
`%LOCALAPPDATA%\Programs\Hades` e exige decisão do dono. Medir o que o
`build_windows.ps1` faz antes de o correr. **E esperar as runs antes de qualquer push.**

### A proxima accao unica (janela `E-1058`)

**O `HadesSetup.exe` não assinado do candidato existe** — 24,26 MB, gerado pelo `ISCC.exe`
a partir de binários que **EU** construí da fonte:
`ollama.exe` 46,65 MB e `windows-ollama-app-amd64.exe` 42,66 MB, ambos `exit 0`.

⇒ ***«O bloqueio não é de permissão: é de toolchain.»*** — o `build_windows.ps1` compila o
motor C++ (`cmake`/`ninja`/`cl` **ausentes**), mas **o caminho que exigia a ferramenta não
era o único caminho**: os binários Go constroem-se, e o C++ já está pronto.

**E o compilador denunciou uma dívida**: `[UninstallRun]` sem `RunOnceId` — **curada**
(2 linhas), e o `Warning` **desapareceu** na recompilação. O hash mudou: é a prova de que
a cura entrou no artefacto.
⇒ ***«Um aviso do compilador é uma medição, não um ruído.»***

**A proxima accao unica**: **medir o artefacto gerado sem o instalar.** `INSTALLED=NAO`
continua verdadeiro e **instalar exige decisão do dono** (escreve em
`%LOCALAPPDATA%\Programs\Hades`). O que é interno e se pode fazer: **verificar o
instalador** com o `/SILENT`+`/DIR` num diretório descartável? Não — isso **é instalar**.
O que resta sem instalar: **extrair o conteúdo do Setup** (o Inno tem `/EXTRACT` em
`innounp`-like tools, mas não há ferramenta) **ou** documentar os hashes e entregar. Medir
primeiro o que é possível sem tocar em `%LOCALAPPDATA%`.

### A proxima accao unica (janela `E-1059`)

**O tarball Linux instalável existe.** O que havia era **um tarball, não um tarball
instalável**: 9 entradas, **sem `install.sh`, sem `uninstall.sh`, sem `VERSION`**.

⇒ ***«A diferença entre os dois estava num ficheiro de 400 bytes.»***

**E o binário Linux é de outro candidato** — medido no próprio ELF: `0.2.0-rc.2.0.
20261015171943-9f7379502b1d+dirty` ⇒ commit **`9f7379502b1d`**, **não** o HEAD. E o
`VERSION` do pacote **diz isso**: ***`build_reproduzivel=NAO`.***

**Sem compilador C e sem bash real** (`gcc`/`clang` ausentes; o `bash` do sistema é o do
WSL e responde `execvpe(/bin/bash) failed`), o `package_linux.sh` **não corre** — e no
entanto o pacote foi produzido. ⇒ ***«O caminho que exigia a ferramenta não era o único
caminho — outra vez.»***

**A proxima accao unica**: **macOS e Web/PWA são os próximos tecnicamente possíveis.**
macOS exige macOS/Xcode (`BLOCKED_EXTERNAL` pela máquina), mas o **Web/PWA** é build
local: `app/ui/app` tem `vite build` e o prompt pede «build de produção + manifest +
ícones + PWA». Verificar se existe script de build de produção e correr. **E esperar as
runs antes de qualquer push.**

### A proxima accao unica (janela `E-1060`)

**O build Web/PWA passa o orçamento**: `✓ built in 15,68s`, entry `index-CWgSFuKK.js`
**raw 1 509 050 / 2 000 000** e **gzip 412 527 / 600 000** ⇒ **`BUNDLE_BUDGET=PASS`**.

**E encontrei um defeito real que já estava no HEAD**: os **três** ícones do manifest
apontavam para `/icons/…` e **o diretório `icons/` nunca existiu** — os PNGs estão na
**raiz** de `public/`. `git show HEAD:…` confirmou: **já era assim no HEAD**.

⇒ ***«Um caminho escrito à mão é uma promessa; um caminho medido é um facto.»***
**Curado**: 0 de 3 ícones resolviam → **3 de 3**.

**E o que falta no PWA é medido, não suposto**: **zero service workers**
(`ocorrencias em src/: 0`; `dist/sw.js` e `dist/service-worker.js` ausentes) ⇒ **sem cache,
sem offline, sem instalação completa**. ⇒ **`PWA_INSTALAVEL=PARCIAL`**.

**A proxima accao unica**: **o service worker** é o que falta para o PWA ser instalável e
offline. Mas **offline honesto** exige decidir o que cachear (e o que **não** cachear: o
chat e as aprovações não podem vir de cache). Desenhar o escopo mínimo — shell + assets
estáticos, **nunca** respostas de API — e implementar. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1061`)

**O service worker existe** — e a **API nunca entra no cache**, por desenho e por teste.

⇒ ***«Um cache que responde por uma aprovação é um aprovador a mais.»***

`/api/*` é **network-only, sem excepção**: servir `/api/chat` do cache devolveria uma
resposta de outro momento como se fosse agora; uma aprovação do cache poderia autorizar
uma acção que já não está pendente.

**8 testes** e **contraprova com 3 defeitos injetados**: remover a guarda de `/api/` ⇒
**vermelho**; remover a guarda de método ⇒ **vermelho**; remover o `caches.delete` ⇒
**vermelho**; restaurar ⇒ **verde**.
⇒ ***«A contraprova é o que separa um teste de uma decoração.»***

**Suite completa**: **111 ficheiros, 738 testes, 0 falhas.**

**A proxima accao unica**: **E2E num browser real** — o service worker existe mas
**nunca foi instalado num browser**. Isso exige um servidor de produção (`vite preview`)
e um browser real; o Playwright está instalado. E **esperar as runs antes do push.**

### A proxima accao unica (janela `E-1062`)

**O PWA foi instalado e testado num Chromium real: 8/8.** O `E-1061` testou o `sw.js`
com `caches` **simulado** — *«prova a lógica; não prova a instalação»*. Este fecha isso.

| verificação | resultado |
|---|---|
| app abre | **200** |
| **service worker ACTIVO** | `script=…/sw.js`, `scope=/` |
| manifest | `name=Hades`, `display=standalone` |
| **3 ícones** | **200 · 200 · 200** |
| **`/api/*` fora do cache** | **404 basic** — chegou à rede |
| cache do shell | `hades-sw-v1-shell`, 2 caches |
| **offline** | **504 honesto** — não inventou conteúdo |
| pedidos falhados | 0 inesperados |

**E duas falhas iniciais eram do meu teste** (6/8): o SW **estava** registado e eu li o
`state` cedo demais; e contei como falha um pedido que **o próprio teste** provocou.
⇒ ***«Medir cedo demais é uma forma de inventar um defeito.»***

**A proxima accao unica**: **Firefox e WebKit** — o E2E só correu em Chromium. O Playwright
tem `webkit-2311` instalado; **Firefox não**. Instalar o Firefox do Playwright e correr os
mesmos 8 pontos. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1063`)

**O PWA passa 8/8 nos tres motores**: `chromium` **8/8**, `firefox` **8/8**, `webkit` **8/8**.
O `E-1062` só tinha corrido em Chromium.

**Faltavam dois dos três motores**: o `playwright-core` 1.64.0 espera `firefox-1555` e
`webkit-2370`; só existia `webkit-2311` (de versão anterior). `npx playwright install
firefox webkit` resolveu em **27,7 s**.
⇒ ***«Duas ferramentas instaláveis não são dois bloqueios externos.»***

**E a diferença offline prova que o teste mede o motor**: Chromium devolve **504**;
**Firefox lança `TypeError: NetworkError`**; WebKit devolve **504**. E o teste aceita
ambos porque **ambos são honestos** — nenhum inventa conteúdo.
⇒ ***«Se o teste medisse a si mesmo, os três dariam o mesmo resultado.»***

**A proxima accao unica**: **`PWA_INSTALADO_NA_UI` continua `NAO`** — o gesto de instalar
é humano. O que é interno e resta: **medir recursos do service worker** (memória, CPU,
tempo de `install`/`activate`, tamanho do cache) num browser real, e **testar upgrade
entre versões reais do SW** (publicar v2, activar, confirmar que a v1 é apagada).
**E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1064`)

**Os recursos do service worker medidos e o upgrade v1→v2 provado num browser real: 9/9.**

| medição | orçamento (declarado **antes**) | medido |
|---|---|---|
| entradas no cache | ≤ 400 | **9** |
| bytes no cache | ≤ 64 MB | **0,16 MB** |
| activação | < 20 s | **448 ms** |

⇒ ***«Um limite que se ajusta ao resultado não é um limite.»***

**E o upgrade**: com o servidor a servir `hades-sw-v2`, o browser **aceitou a v2**, o
`activate` **apagou a v1** («nenhum cache `hades-sw-v1-*`»), e o **`/api/*` continuou a ir
à rede** depois do upgrade.
⇒ ***«Um service worker que nunca é substituído congela o produto no dia em que é
instalado.»***

**A proxima accao unica**: **Firefox e WebKit não foram medidos** — o `E-1063` correu os
8 pontos nos três motores, mas **recursos e upgrade só em Chromium**. Correr o mesmo
`pwa-recursos-e-upgrade.mjs` nos três motores (o script já abre um servidor próprio na
porta 5180; basta parametrizar o motor como no `E-1063`). **E esperar as runs antes do
push.**

### A proxima accao unica (janela `E-1065`)

**Os recursos e o upgrade do SW passam 9/9 nos tres motores** -- e a latencia de activacao difere:

| motor | activacao | resultado |
|---|---|---|
| Chromium | **423 ms** | 9/9 |
| WebKit | **1041 ms** | 9/9 |
| Firefox | **1646 ms** | 9/9 |

Mesmo comando, mesma maquina, mesmo `dist/`, mesmo orcamento de 20 s.
=> ***«Se o instrumento se medisse a si mesmo, os tres dariam o mesmo numero.»***

E os tres, em todos os pontos: SW `activated`; `hades-sw-v1-shell`(3) + `hades-sw-v1-assets`(6);
**9 entradas, 160,6 KB, 0,16 MB** (identico ao byte); **upgrade v1->v2 aceite**; **v1 APAGADA**;
e **`/api/*` continua a ir a rede** depois do upgrade.

=> ***«Um numero nao e' uma explicacao.»*** -- o Firefox e' **3,9x** mais lento a activar que o
Chromium, e **eu nao medi por que**.

**A proxima accao unica**: **investigar o +1,2 s do Firefox na activacao do SW** -- e' um numero
medido e nao explicado. Instrumentar os marcos (`install`, `waiting`, `activate`) com
`performance.now()` no proprio `sw.js` e comparar. Se for estrutural do motor, registar como
limite; se for do nosso `addAll`, corrigir. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1066`)

**O tempo da activacao do SW esta' no `install` -- e o numero de 1646 ms do `E-1065` NAO era o SW.**

| motor | install | activate | fora | pagina (ready->activated) |
|---|---|---|---|---|
| Chromium | **18,2 ms** | **0,1 ms** | 1,0 ms | **73,2 ms** |
| Firefox | **132,0 ms** | **0,0 ms** | 3,0 ms | **262,0 ms** |
| WebKit | **72,0 ms** | **0,0 ms** | 12,0 ms | **308,0 ms** |

**H1 ACEITE** nos tres: o custo e' o `addAll` (shell + 6 icones). O `activate` custa **0 ms**.

**E a correcao da minha propria medicao**: o `E-1065` media desde **abrir a pagina**; este mede
desde `navigator.serviceWorker.ready`. O numero grande era **o browser a arrancar**, nao o SW.
=> ***«O cronometro estava certo; a etiqueta no cronometro e' que estava errada.»***

**E o veredito sobre optimizar**: `install` do Firefox **132 ms** contra orcamento de **20 s** =
**~150x de folga**. Nao se optimiza. E trocar o `addAll` atomico para ganhar 100 ms poria em risco
a garantia do `E-1061`: *«preferimos um SW que nao instala a um SW que instala meio shell»*.
=> ***«Nao se optimiza o que ja' esta' 150x dentro do orcamento.»***

**A proxima accao unica**: o perfil do SW esta' fechado. **Web/PWA tem agora 8 pontos (3 motores),
9 pontos de recursos/upgrade (3 motores) e o perfil do install/activate.** O que resta no Web e'
**`PWA_INSTALADO_NA_UI`**, que e' humano. *Avancar para uma plataforma por produzir* -- e a
**Android** e' a proxima tecnicamente possivel (Gradle local, APK nao assinado). Medir primeiro o
que o `apps/mobile-agentic` exige. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1067`)

**O APK Android: o projecto esta' completo, os tres requisitos estao' presentes, e o Gradle 8.13
nao carrega a `native-platform.dll` neste host.**

**Medido**: `apps/mobile-agentic/android/` **nativo completo** (`gradlew`, `gradlew.bat`,
`build.gradle`, `settings.gradle`, `local.properties`, `gradle-wrapper`); **`_tmp\android-sdk`
com 394,2 MB** (`android-34`, `build-tools 34.0.0`, `adb`); **`_tmp\jdk17` = openjdk 17.0.20.1**;
**Gradle 8.13** ja' extraido. Os tres requisitos estavam la'.

**3 tentativas, 3 hipoteses**: wrapper (`FileNotFoundException` no zip apagado) -> gradle directo
(`Failed to load native library`) -> `-Djava.io.tmpdir` explicito (mesmo erro). O `--stacktrace`
**nomeou a causa**: `NativeLibraryLocator.java:46` -> `createFileExclusively` ->
***«O sistema nao pode encontrar o caminho especificado»***.
=> ***«O protocolo antiloop manda parar a' terceira e registar -- e' o que faco.»***

**E isto NAO e' `BLOCKED_EXTERNAL`**: nao falta credencial, conta nem dispositivo. E' uma **falha
de ambiente reproduzivel**, com causa nomeada e passo de desbloqueio concreto: (1) confirmar o
`java.io.tmpdir` que o processo java ve; (2) criar esse directorio exactamente; (3) se continuar,
extrair a `native-platform.dll` do jar a' mao e passar `-Dorg.gradle.native.dir`.

**A proxima accao unica**: **fazer o passo (1)** -- `java -XshowSettings:properties -version` com o
JDK 17 e o `TMP` que o Gradle herda, para ver **que caminho** o `NativeLibraryLocator` tenta criar.
E' uma medicao de 1 comando, com hipotese nova -- nao e' a quarta tentativa cega. **E esperar as
runs antes do push.**

### A proxima accao unica (janela `E-1068`)

**A marca Android migrada em três categorias, e elas não são a mesma coisa**:

| categoria | acção |
|---|---|
| **UI visível** | muda — `DZ23 AGENTIC` → **`HADES`** |
| **identificadores de package** | mudam — `com.lmprado.dz23agentic` → **`com.dz23ltd.hades`** |
| **chaves de storage** (`dz23.agent.*`) | **FICAM** (4 + 2 ocorrências preservadas) |

⇒ ***«Uma migração de marca que apaga a sessão de quem já a usava não é uma migração: é uma
perda.»***

**E o `android/` não se commitou**: é **saída do `expo prebuild`** (35 ficheiros não
rastreados), e o `app.json` é a fonte — o `build.gradle` deriva do `app.json.android.package`.
A migração **correcta** é no `app.json`, e está feita.

**O plano está escrito onde se lê**: `apps/mobile-agentic/MIGRACAO_MARCA.md` — com a tabela
antes/depois, o que **não** mudou e por quê, e o que acontece a quem tem a versão antiga (no
Android o `applicationId` **é a identidade** — a versão nova é **outra app**; e a mitigação
honesta é que **não existe APK com o id antigo**, logo não há ninguém a quebrar).

**E quatro ficheiros que eu não toquei apareceram como `M`**: medido com
`git diff --ignore-cr-at-eol --numstat` → **vazio** ⇒ **só fim-de-linha**.

**A proxima accao unica**: **o inventário de marca mostra 231 `DZ23 Agentic` e 49
`Ollama DZ23` ainda no monorepo** — a migração do mobile reduziu, mas falta o resto. **A
proxima fatia é classificar as 231 ocorrências** por categoria (UI visível / nome técnico /
excepção) usando o próprio `HADES_BRAND_INVENTORY.md`, que já as classifica. **E esperar as
runs antes do push.**

### A proxima accao unica (janela `E-1069`)

**As 231 ocorrências de `DZ23 Agentic` classificadas NO SÍTIO delas: ~250 são nome técnico
e 19 são histórico — não há marca visível por migrar.**

**Como foi medido**: carreguei o **próprio gerador** por `importlib`, chamei `inventariar()`, e
para **cada** ocorrência olhei **50 caracteres de contexto** em cada lado, classificando contra
**10 padrões técnicos declarados antes**.

**As 19 «marca», uma a uma**: **3** em `MIGRACAO_MARCA.md` (o documento **que eu criei**, e que
**cita os nomes antigos de propósito** — é uma tabela antes/depois) · **14** no
`HADES_EVIDENCE_INDEX.md` (**histórico**, e o §FASE-7 proíbe reescrever evidência) · **1+1** no
`HADES_HANDOFF.md` e `HADES_PENDING_WORK.md` (documentos que eu escrevi, a citar o passado).
⇒ ***«Nenhuma é texto de interface de utilizador.»***

**E as ~250 técnicas**: `DZ23-LTDA` é o **owner do GitHub** · `dz23-agentic-quality` é o **nome
do workflow de CI** · `com.lmprado.dz23agentic` é **bundle id** · `dz23.agent.token` é **chave
de storage** · `Ollama Full`/`Ollama Classe A+`/`Ollama DZ23` são **nomes históricos de
produto** em documentos datados.

**A correção da minha hipótese**: eu esperava um erro de **contagem**; o erro estava na
**granularidade da categoria**. ⇒ ***«Um ficheiro migrável não faz das suas ocorrências marca
visível.»*** ⇒ ***«Quem procura um erro no número pode não ver o erro na pergunta.»***

**A proxima accao unica**: o **§FASE-5 item 2** está cumprido (mobile migrado) e **não tem mais
nada para migrar**. O item 3 («manter, e dizer por que») resta — e o inventário já o diz para
as **8 exceções**. ⇒ **fechar o §FASE-5** e **voltar ao bloqueio do Android** com a última
hipótese não testada: extrair a `native-platform.dll` do jar **à mão** e passar
`-Dorg.gradle.native.dir`. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1072`)

**O Gradle arranca com a DLL extraída à mão — mas o daemon não a herda.**

**O ganho, medido**: extraí a `native-platform.dll` (**141 312 B**) do
`native-platform-windows-amd64-0.22-milestone-28.jar`, e
`gradle --version -Dorg.gradle.native.dir=…` deu **exit 0** — `Gradle 8.13`, `Launcher JVM:
17.0.20.1`. **Antes disto, o Gradle nem `--version` fazia.**

**O bloqueio**: `assembleDebug` arranca um **daemon**, e o daemon falha com
***`NativeServices.initializeOnDaemon` → `createFileExclusively` → «O sistema não pode
encontrar o caminho especificado»***. **6 hipóteses, 10 tentativas**, todas falharam no daemon.

**E a pista que o log do daemon deu — e que diz que já funcionou**: em **2026-10-16T06:42:43**
o daemon **executou uma build** e falhou com ***`No Java compiler found`*** — **não** com a DLL.
⇒ ***«Um log antigo que ninguém lê é uma pista que ninguém usa.»***

**A próxima accao unica**: a **6ª hipótese falhou** (o `gradle.properties`), e o `android/` é
**gerado** — testar mais variantes ali é **trabalho descartável**. O caminho real é **o
`JAVA_HOME` do daemon**: o log diz que o daemon já correu, e o erro de então era `No Java
compiler found` — **a JVM do daemon não está a ver o JDK 17**. Verificar o `org.gradle.java.home`
e o `daemon JVM` que o `--version` reporta. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1073`)

**O daemon do Gradle: 12 tentativas, 8 hipoteses, `BLOCKED` — e mudanca de trabalho.**

**As 8 hipoteses, todas medidas**: (1) `native.dir` no launcher → `--version` exit 0,
`assembleDebug` exit 1 · (2) `native=false` → o daemon **precisa** das native services ·
(3) `java.io.tmpdir` curto → **medi** que o efectivo é `AppData\Local\Temp` (o Java ignora o
`TMP` que eu defini) · (4) `GRADLE_OPTS`/`JAVA_OPTS` → não chegam ao daemon · (5) layout
alternativo → mesmo erro · (6) `gradle.properties` → mesmo erro · (7) **criar
`~/.gradle/native/`** (que **não existia**) → falhou igual · (8) `--info` → a única linha é
`Failed to load native library`, sem caminho.

**E a prova de que o daemon já carregou**: `daemon-3984.out.log` (10-16 06:42:43) tem
`[DEBUG] [NativeServices] Native-plat...` seguida de `Executing build` e de
`GeneratedClassCompilationException: No Java compiler found` ⇒ **o `NativeServices`
inicializou**; a falha de então era `javac` fora do `PATH`.

**Pelo protocolo antiloop, item 5**: *«se a mesma causa bloquear três tentativas, registe
BLOCKED com evidência e avance para trabalho independente»* — 12 tentativas é **quatro vezes**
o limite. ⇒ ***«Repetir com variação não é investigar — é insistir com disfarce.»***

**A proxima accao unica**: **mudar de trabalho** — não insistir no Gradle. O caminho interno
sem bloqueio é o **§FASE-7 item 3**: classificar os **73 IDs órfãos** por requisito/call site.
O desbloqueio do Android é **decisão do dono** (outro Gradle 8.14+/9.x, ou outro host/runner).
**E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1074`)

**Os 74 IDs orfaos classificados: todos dentro da faixa declarada, zero fora.**

**Como foi medido**: varri o repositorio por `E-(\d{3,})` e comparei com as **866** entradas
do indice. Resultado: **77 IDs citados sem entrada** — e desses, **3 são falsos positivos**
com prova: o `E-000` (sentinela), o **ano** de um aviso CVE, e a **cabeça de teste** do
proprio `hades_index_shape_check.py` ⇒ **74 orfaos reais**.

**O split exacto**: a faixa `E-571..E-777` tem **207** IDs ⇒ **74 citados + 133 não
citados = 207**. O prompt dizia **73 + 134** ⇒ ***a diferença é 1, e eu medi o número, não o
copiei.***

**E os 74 estão TODOS dentro da faixa; `FORA da faixa: 0`** ⇒ a catraca (`E-778`) **já os
permite** porque a faixa é **dívida declarada**. O §FASE-7 item 6 está cumprido.

**Onde os 74 vivem**: **63 em código** e **11 só em documentos**, em **88 ficheiros de
código**. Os que mais citam: `server/agent_routes.go` (18), `agenticClient.ts` (9),
`hades_local_gates.ps1` (7). A lista completa está em `_tmp/orfaos_tabela.md`.

**E a ironia que isto produziu**: ao **citar** os dois falsos positivos no texto do `E-1074`,
criei **dois novos** — porque o `CITACAO` é `\bE-(\d{3,})\b` e o texto tinha a forma exacta
de uma citação. ⇒ ***«O instrumento que mede o defeito participa do defeito.»***

**A proxima accao unica**: o **pacote de auditoria independente** (§FASE-7 item 8) —
`README_AUDITOR.md`, `PARIDADE_EVIDENCIAS.csv`, `GATES.md`, `LIMITACOES.md`,
`ARTEFATOS.sha256` — usando a classificação dos 74 como **alcance declarado**. **E esperar as
runs antes do push.**

### A cura do `E-1075` (a mesma janela)

**O `E-1074` expôs um defeito de paridade**: o `hades_handoff_window_check.py` **não** tinha o
limite inferior que o `hades_index_shape_check.py` já tinha ⇒ o `E-000` (que vira `0` no `int()`)
entrava como *«citado sem entrada»* em **qualquer** documento que o mencionasse.

**Cura**: a **mesma** regra (`_abaixo = min(registados)` e `c >= _abaixo`). ⇒ ***«Dois checkers, a
mesma regra, e um deles ficou para trás.»***

**Resultado**: `JANELA DE CITACAO - fault injection: 20 OK, 0 FAIL` (era 18/2), e a bateria voltou a
**33 OK / 0 FAIL**. ⇒ ***«O documento que documenta o falso positivo é o teste que o checker não
esperava.»***

### A proxima accao unica (janela `E-1076`)

**O pacote de auditoria independente (§FASE-7 item 8) existe — e é gerado, não escrito.**

**5 artefactos**, cada um função de uma fonte medível: `README_AUDITOR.md` (as secções, os
hashes, e **o que o auditor NÃO vai encontrar**) · `PARIDADE_EVIDENCIAS.csv` (as 869 entradas
do índice) · `GATES.md` (os **36** gates declarados, um por linha) · `LIMITACOES.md` (os **17**
blockers, a faixa `571..777`, os artefactos ausentes, a matriz por plataforma) ·
`ARTEFATOS.sha256` (SHA-256 dos artefactos que **existem**).

**Por que é gerado**: um pacote escrito à mão **envelhece no dia seguinte** — é o `E-1071`
outra vez. `--check` verifica que bate com o repositório.

**E o gerador tinha dois defeitos, ambos medidos**: (1) listava um zip que **já não existe** no
`dist/`; (2) separava os blockers resolvidos dos abertos por `"RESOLVIDO" in descricao` — e
**mediu mal**: dava 5 abertos e 12 resolvidos, quando há **8 abertos e 9 resolvidos**.
⇒ ***«É o `E-83` no meu próprio gerador: um detector lexical vê termos, não relações.»*** A
cura foi separar pelo **campo `tipo`**.

**24 verificações, 4 defeitos injetados**: artefacto apagado → exit 1; linha a mais → exit 1;
índice que muda → exit 1; e o **`E-83`**: um blocker com `tipo=RESOLVIDO` e a palavra «ABERTO»
na descrição fica em **Resolvidos**, e um com `tipo=EXTERNO` e a palavra «RESOLVIDO» fica em
**Abertos** — ambos verificados.

**A proxima accao unica**: o **`E-1076` fecha os 5 artefactos nomeados**; falta o **estado do
CI do mesmo snapshot** e a **lista de findings abertos/fechados** que o item 8 também pede.
E o **Android** continua `BLOCKED` (decisão do dono). **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1077`)

**A lista de findings do PR #438: 14 capturados da API, e o MD gerado do JSON — o `--check`
não precisa de rede.**

**O que foi medido**: `gh api …/pulls/438/comments` → **14 review comments**, todos de
`coderabbitai[bot]` → **5 Major** e **9 Minor**. O `FINDINGS.json` versionado guarda id, autor,
path, linha, data, severidade, categoria, título e trecho, mais o **SHA do checkout**
(`093390737`) e a **data da captura**.

**E os findings batem com o que o prompt descreve** — o que valida a captura: o #2
«Classifique cada uso de relógio separadamente» (item 2), o #4 «Continue a busca após o
primeiro `.Add(`» (item 3), o #5 «Vincule a triagem ao conteúdo» (item 4), o **#6 Major
«Protect every per-organization approval write»** (o **P0**, item 1), o #14 «Do not classify
partially closed findings as closed» (item 8).

**A decisão de desenho, por causa da rede**: dois passos. A captura (`--capturar`) escreve o
**JSON versionado** com data e SHA; a geração lê **só** o JSON. ⇒ ***«Um pacote de auditoria
que só se verifica com internet não se verifica num ambiente isolado.»***

**E os três defeitos que o gerador tinha, todos medidos**: (1) o `Total: 14 — .` saiu com a
lista vazia porque o regex procurava `"**Major**"` e o corpo tem `"**🟠 Major**"` — **o emoji
fica DENTRO do negrito**; (2) o **título** saiu como `🟠 Major` porque apanhava `negritos[1]`,
que é a severidade — o título é o **quarto**; (3) o ID injectado do DEFEITO 3 foi construído em runtime, e não escrito por extenso.
⇒ ***«Um índice por posição fixa quebra quando a decoração muda.»***

**O `FINDINGS.md` diz o que NÃO afirma**: não afirma que algum finding foi corrigido (nenhum
foi marcado no GitHub — proibido), não afirma que o bot está certo, não afirma que a lista é
completa.

**A proxima accao unica**: **os 14 findings ainda não foram validados contra o código**. O
`E-1070` mostrou o método (reproduzir num worktree limpo). O **#6 (P0)** e o **#2/#4/#5** são
os primeiros. **E esperar as runs antes do push.**
