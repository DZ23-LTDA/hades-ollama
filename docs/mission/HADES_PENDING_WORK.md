# HADES — TRABALHO PENDENTE PRIORIZADO

## 0.0 REMEDIÇÃO MEDIDA (2026-10-15, agora) — a secção 0 abaixo fica preservada como HISTÓRIA

> **REMEDIÇÃO MAIS RECENTE: `E-959`** (*cadeia `E-820` → `E-866`, rodadas 181–221*).
>
> 🧑 **E o §18.3-30 («teste de pessoa leiga») foi inventariado (`E-866`) — e fica PARCIAL, com a lacuna
> nomeada:** as **peças** existem (*`Onboarding.tsx` 50 828 B — o maior componente de UI do repositório —,
> `Onboarding.test.tsx` 45 437 B, `ModelPicker.tsx`, help contextual, e onboarding de empresa com limites e
> orçamento*), mas **falta o teste da pessoa**: ninguém percorreu o produto como um leigo.
> ⚠️ E **«instalar sem terminal» tem receita e não artefacto**: `app\ollama.iss` (*Inno Setup 6*) existe e o
> **Inno Setup não está instalado** ⇒ *é a decisão **(c)** do dono, e mantém o **`G10`** em `PACKAGE_CREATED = NÃO`.*
>
> 🔎 **E a «via barata» para o `G3` foi testada em vez de suposta (`E-865`):** o `llama-server.exe`
> **existe** — no Ollama que o utilizador já tem instalado (`lib\ollama\`, com `libllama.dll`,
> `libllama-server-impl.dll` e `libllama-common.dll`). Copiei-o para o sítio onde o Hades procura e
> **o erro mudou**: de `llama-server binary not found` para **`system_info line not found in output`**.
> ⇒ ***O binário corre e não é compatível*** — *o fork modificou `llama/server`, e por isso o motor do
> Ollama oficial não serve*. **O `G3` continua `BLOCKED_EXTERNAL`, mas por um motivo mais preciso.**
> *Os 10,3 MB copiados foram removidos e o Ollama do utilizador ficou intacto.*
>
> 🔥 **E o `G3` mudou de categoria (`E-864`): saiu dos «por medir» e entrou nos `BLOCKED_EXTERNAL` —
> com causa medida e procedimento dado pelo próprio programa.** O Hades **compila** (*65,7 MB, 10,1 s*) e
> **arranca**, mas `POST /api/chat` devolve **404**; o `stderr` diz **`llama-server binary not found`**
> (*o motor de inferência nunca foi construído*) e dá o comando
> **`cmake -S llama/server --preset cpu && cmake --build --preset cpu`**.
> *Exige `cmake` + toolchain C++, que este host não tem* ⇒ *é a decisão **(a)** do dono (`mingw-w64`).*
> **Isto acrescenta um item de bloqueio real à lista:** *sem o motor, o §18.3-07 (`/goal` com modelo local)
> e o §15-G3 ficam bloqueados — e nenhum dos 19 itens abaixo cobre isso, porque não é «trabalho pendente
> de engenharia»: é **ambiente ausente**.* HEAD
> **`9f7379502`** · branch **`fix/checker-relogio`** · **656** evidências · **34** entradas não
> commitadas · **0** stashes · veredicto **`NOT_RELEASE_READY`**. A classificação `E-841` abaixo
> continua válida e fica **preservada**; o que muda é o **estado do trabalho**, medido nesta janela.
>
> **Os 19 itens deste ficheiro continuam classificados (`E-841`)** — *nenhum é defeito de produto* — e
> **todos** continuam **gated em decisões do dono ou em PRs abertos**, não em trabalho de engenharia:
> - **P0-5** — as duas PRs da primeira jornada (#61 vs #71): manter a #71, fechar a #61 só com autorização.
> - **P1-1** — PR #69, merge **não autorizado**. **P1-2** — PRs #56/#58. **P1-3** — PRs #63/#65/#57 (*bloqueado por B-03*).
> - **P1-4** — **28 ficheiros / 44 sítios** que constroem runtime sem `DataRoot`: *é a única fronteira
>   interna real e continua por fechar*.
> - **P1-5** — 104/122 conectores precisam de operador.
> - **P2-2** — Node 20 deprecado em 15 workflows (*exige commit ⇒ travado pelo congelamento*).
> - **P2-3** — replay do upstream: 73 commits, 39 ficheiros em conflito, 15 conflitos reais.
> - **P3-1** — release assinada: **`FUTURE_DISTRIBUTION`** por decisão do dono.
>
> **E o que esta janela provou que NÃO é pendência:** a inferência local funciona e foi **medida**
> (`E-862`/`E-863` — 12,07 s a frio, 0,88 s a quente, trabalho útil idêntico); as defesas de **egress,
> redirect e DNS rebinding** estão provadas (`E-855`); e as **três** matrizes de lacunas não silenciam
> nada (`E-841`/`E-842`).
>
> ⚠️ **REGRA DE MANUTENÇÃO** (*`E-861`, reconfirmada na ronda 218*): este é **um dos três** documentos
> de retomada validados pelo `hades_mission_check.py` (*com `HADES_HANDOFF.md` e `HADES_NEXT_ACTION.md`*).
> O piso de evidência **sobe a cada ronda** ⇒ **actualizar os três na mesma ronda** em que se regista
> evidência. *O checker nomeia um de cada vez; corrigir o nomeado revela o seguinte.*

> **A regra do próprio ficheiro (linha ~27) diz: «qualquer número acima envelhece. Antes de citar, remeça.»** Isto é o
> resultado de a cumprir. **A secção 0 cita `cafb00dbe` / `06cd277da`** — e os dois são **ancestrais do HEAD**
> (`git merge-base --is-ancestor` => exit 0), com o segundo **55 commits atrás** (`git rev-list --count 06cd277da..HEAD`
> = **55**). *Nada foi apagado: o histórico fica, e estes números substituem-no até nova remeção.*

| Medida | Valor **medido agora** |
| --- | --- |
| HEAD local | `9f7379502b1d5d5bb7abb8e2bd55224e92796dfd` |
| Remoto `refs/heads/fix/checker-relogio` (`git ls-remote`) | `9f7379502` => **local = remoto**, **0 por empurrar** |
| `main` remota (`git ls-remote`) | `4b3e2185422c24d5a05fd46a343953ad6f41d1e5` |
| Distância dos números da secção 0 ao HEAD | **55 commits** |
| Evidências no índice (`Select-String '^\| E-\d+ '`) | **571** linhas = `E-001..E-570` + **`E-778`** (o `E-778` é o próximo ID livre; a secção 0 diz 546 — **histórico**) |

> **REMEDIÇÃO SEGUINTE (2026-10-15, 19:14 BRT).** A linha acima **já envelheceu** — e é a regra deste próprio
> ficheiro que o exige dizer. Foi escrita com **571** linhas e **`E-778`** como próximo ID livre. Valores
> **medidos agora**, no mesmo HEAD `9f7379502`:

| Medida | Valor **medido agora** |
| --- | --- |
| Linhas do índice | **591** = `E-001..E-570` + **`E-778`..`E-798`** |
| **Próximo ID livre** | **799** (prefixo `E-`) — *herdar `E-778` da tabela acima causaria **colisão de ID**, o defeito que o próprio `E-778` documentou. Escrito como número, e não com o formato de evidência, porque o `hades_mission_check.py` exige que todo ID citado esteja registado — e um ID **futuro** não pode estar* |
| `ENTRIES` / `STASHES` | **29** / **0** |
| Checkers oficiais | **7/7** `exit 0` |

> O **`E-798`** (nesta mesma ronda) corrigiu o **item P1 3** abaixo: a classificação `BLOCKED_EXTERNAL` do pacote
> `./server` estava **refutada** pelo `E-795`. **O item P1 4 continua aberto** (hermeticidade dos testes).

> **`E-778` — COLISÃO DE IDs (o defeito mais grave desta sessão, e é meu).** A **sequência viva** do projecto vai em
> **`E-777`**, cunhada em **código e commits** e **nunca** registada no índice; este índice registava até `E-570` ⇒
> **7** das entradas `E-560..E-570` **colidem** com outro objecto já citado em código committado. O instrumento novo
> (passo 15 do `hades_mission_check.py`, `E-570`) verifica **existência** — e *a existência **não** distingue um ID
> livre de um ID já ocupado*. ⇒ Duas regras ficaram **mutuamente exclusivas**: o passo 1 exige **contiguidade** e o
> `E-778` exige **não-colisão**. Cura aplicada: um buraco **declarado** em `evidence_ids_nao_registrados` deixa de ser
> falha; o que continua a falhar é um buraco **não declarado**.
> **⇒ Ao retomar: NÃO reutilize nenhum ID abaixo de `E-778`.**
| Árvore de trabalho | **16** entradas não commitadas, **0** stashes — **WIP do proprietário preservada** (naquele momento) |
| Árvore de trabalho (**medido agora**, `E-799`) | **29** entradas não commitadas, **0** stashes — *o **16** acima envelheceu: a árvore cresceu com os artefactos e scripts G12, o bundle e a consolidação de documentos* |
| Veredito global | **`NOT_RELEASE_READY`** |

### CI DO SHA CONGELADO — medido agora, e é o primeiro registo deste número

O SHA candidato `9f7379502` **tem CI remoto concluído**, com **5 workflows `success` no MESMO `head_sha`**
(evento `pull_request`, PR **#438**, arranque `2026-10-15T17:19:55Z`):

| Workflow | Run | Resultado |
| --- | --- | --- |
| `test` | `38776430738` (#1242) | **success** |
| `dz23-agentic-quality` | `38776430666` (#2018) | **success** |
| `pr-gate` | `38776430633` (#979) | **success** |
| `dz23-e2e` | `38776430658` (#341) | **success** |
| `class-a-plus-integrity` | `38776430631` (#2116) | **success** |

=> *O `pr-gate` é o **agregador** (recusa `cancelled`/`skipped` sem mascarar a falha — `E-551`), e está verde **neste** SHA.*
***LIMITE, e importa:*** *este CI prova a **árvore commitada** de `9f7379502` — **não** as **16** alterações não commitadas
da árvore de trabalho. A revisão independente usa **um** SHA (§18.5.10), e esse SHA está verde.*

### P0/P1 ABERTOS, MEDIDOS NESTA SESSÃO (arquivo:linha + bloqueio)

1. **P0 — o seletor de modelo do Chat não persiste contra o `serve`.** `server/desktop_local_routes.go:44-89`: o `GET`
   devolve um objecto **hard-coded** e o `POST` responde **501 honesto** (antes: 200 ecoando o corpo **sem gravar**).
   Bloqueio: **CGO ausente** neste host (`CGO_ENABLED=0`, `gcc` ausente) => `app/store` (SQLite) é um *stub* em runtime.
   **Decisão pendente do dono** — trazer um toolchain C de terceiros é uma acção de **supply chain**, e a rede funciona.
   Rota real do produto: `app/cmd/app/app.go:30-32` + `app/ui/ui.go:1631,1653`. Evidências `E-560`, `E-561`, `E-563`.
2. **P0 de instrumento — o gate local podia estar VERDE com o caminho de persistência VERMELHO.** O passo 4 não incluía
   `./app/...` (o CI inclui: `dz23-agentic-quality.yaml:82`, `release.yaml:78`). **Corrigido** em
   `scripts/hades_local_gates.ps1`: passo 5 condicionado a cgo, escopo declarado no resumo, e `-First`->`-Last` no passo 2.
   Evidência `E-561`.
3. **P1 — o pacote `./server` não é reprodutivelmente verde neste host.** `whatsapp_routes_test.go:38` =>
   `internal/agent/runtime_storage.go:66` (`mkdir %APPDATA%\ollama-agent\<hash>: Acesso negado`), **~92** testes em
   cascata. Provado **não** ser ACL, **não** confinamento, **não** o caminho => **processo filho do harness**.
   `BLOCKED_EXTERNAL` para a verificação local desse pacote. Evidência `E-562`.
   **CORRIGIDO — medido em `E-795`/`E-796` (2026-10-15): esta classificação está REFUTADA.** Com
   `APPDATA`/`LOCALAPPDATA` redireccionados para um directório de trabalho, o pacote `./server` fica verde
   **3/3 corridas** (`exit=0`; `ok` **29,665 s** / **25,994 s** / **29,231 s**; `FAIL=0`; «Acesso negado»=**0**),
   e o teste `whatsapp_routes_test.go` corre e **passa** (`-run WhatsApp -v` => **7 PASS / 0 FAIL / 0 SKIP**).
   O estado deixa de ser `BLOCKED_EXTERNAL` para a verificação local: passa a **EXECUTÁVEL com isolamento
   de `APPDATA`**. O **mecanismo** do `Acesso negado` continua **por determinar** (o directório existe no
   perfil real, o `mkdir` funciona a partir do shell, e o `ollama-agent` real tem **0** subdirs): *efeito
   medido, causa não*. O **item 4 abaixo continua aberto** (hermeticidade: 42 sítios sem `DataRoot`) —
   o isolamento é uma **técnica de invocação**, não uma correcção do teste.
4. **P1 — hermeticidade dos testes de `./server`: 42 sítios sem `DataRoot` em 27 ficheiros**, e
   `agent_runtime_defaults_test.go:42` tem o **default da plataforma como objecto** (torná-lo hermético destruiria o
   teste). `XDG_CONFIG_HOME` (`:64`) é **ignorado no Windows** por `os.UserConfigDir()`. Evidência `E-563`.
   **REMEDIDO (`E-799`): o predicado «42 sítios em 27 ficheiros» NÃO é reproduzível como está escrito.** Medido:
   **150** ficheiros `*_test.go` em `server/`, **28** mencionam `DataRoot`, **122** não mencionam, **46** ocorrências
   no total. *Nenhuma destas contagens dá 42 nem 27.* O item **continua aberto** (a hermeticidade não foi provada
   nem refutada) — mas fica registado que **precisa do predicado exacto** para poder ser remedido. Sem isso,
   um auditor não consegue nem confirmar nem fechar o item.
   **PREDICADO RECUPERADO (`E-800`).** Testados **7** predicados naturais em `server/*_test.go` (**150**
   ficheiros). O que se aproxima é: *ficheiros que constroem o runtime* (`runtime.New`|`NewRuntime`|`agentRuntime`)
   *e não mencionam `DataRoot`* => **28 ficheiros / 44 ocorrências** (ocorrências dos construtores dentro desses
   ficheiros), contra os **27 / 42** aqui escritos. Diferença: **1 ficheiro e 2 sítios**.
   **Duas leituras, e o `E-800` não decide entre elas:** *(a)* o predicado é este e estes números **envelheceram**
   (a árvore ganhou 1 ficheiro e 2 sítios); *(b)* falta **mais uma condição**. O que está provado: existe um
   predicado **reproduzível** (28/44) e **nenhum** dos 7 testados dá 27/42. **O item continua aberto** — e a
   lacuna é **maior** do que aqui estava escrito, não menor. Comando: contar `*_test.go` de `server/` com
   `runtime.New|NewRuntime|agentRuntime` e sem `DataRoot`.
   *Pistas úteis do mesmo teste:* `newTestServer|setupServer|newServer` = **19** ficheiros / **131** ocorrências;
   `t.TempDir()` = **89** ficheiros / **318** ocorrências — *muitos testes isolam um diretório temporário, mas não
   o `DataRoot`.*
5. **P1 — `integration/max_queue_test.go:20`**: `t.Skip` **incondicional** (teste desligado em todos os ambientes; a
   guarda de `:22-25` é código morto). Dos **292** `t.Skip` do repositório, é o **único** incondicional. Evidência `E-564`.

### DÍVIDAS DE INSTRUMENTO FECHADAS (para o auditor não as reabrir)

- `E-553` (o `t.Skip` do symlink): **condição lida** — `internal/agent/media_test.go:290` salta por **capacidade**
  (`os.Symlink` falhou), depois da asserção do caminho seguro (`:281-288`, que corre sempre) e antes da de recusa
  (`:293-298`) => **não esconde nada que pudesse ter corrido**. A dívida declarada era **falsa**. Fechada em `E-564`.

## 0. ESTADO MEDIDO — `fix/checker-relogio` `cafb00dbe` (esta é a leitura canônica; não herde número de texto antigo)

> **BLOQUEIO MEDIDO (2026-10-15, item 13 do complemento 48 — teste de browser).** O browser mode do `vitest 3.2.7` **está quase montado**: existem `@vitest/browser`, `playwright`, `playwright-core`, `@playwright/test`, `react-dom` e **Chromium instalado** (`ms-playwright/chromium-1228,1243,1248`). **Faltam exactamente duas peças** — `@vitest/browser-playwright` (*o provider que o Vitest ≥3.2 exige*) e `vitest-browser-react` (*renderizador React para browser mode*); `vitest.config.ts` só tem `environment: "node"`. ⇒ **`BLOCKED_BY_EXTERNAL_DEPENDENCY`** com a peça nomeada. **DECISÃO TOMADA:** *não* adicionei as dependências ao `package.json` — fazê-lo **sem regravar o lockfile quebraria o `--frozen-lockfile` do CI**, trocando um teste em falta por um CI vermelho. **Comando correcto (operador, com rede):** `pnpm add -D @vitest/browser-playwright vitest-browser-react` **+ commit do lockfile no mesmo commit**. *Todo o resto da cadeia da sala está `TESTED`; falta só a fiação React de 3 linhas.*

> **FRENTES 45/46/47 (prioridade 7 da FASE 6) — MEDIDAS (2026-10-15), e uma correcção minha.** Eu tinha escrito «prioridades 7 ainda intocadas»: **a medição diz que não.** *Companion* tem **domínio + testes + UI**: `internal/agent/companion_estado.go` (conjunto **fechado** de 8 estados; `DerivarEstadoCompanion` devolve **estado E razão**; `CompanionPodeDizer(estado, frase)` é a guarda contra o mascote afirmar o que não pode provar; invariante **«sem missão não há estado honesto»** ⇒ `ErrCompanionMissaoAusente`) + `companion_estado_test.go` (9 casos) + `CompanionMascote.tsx` (174 linhas) com `CompanionMascote.test.tsx` (movimento, anúncio para quem não vê, **ausência declarada**, onboarding). *Studio* tem página real (`StudioCanvasPage.tsx` ≈950 linhas, rota `studio.tsx`, item no `AppSidebar`, aviso extraído para ser testável). **LACUNA MEDIDA:** `DerivarEstadoCompanion` **não aparece em `server/`** ⇒ *o Companion tem domínio e UI mas **não tem rota** que alimente a UI com o estado* — a mesma classe do `EvaluationRunner`, medida por nome. **Artifact Drive — MEDIDO no vocabulário correcto (`Artefato|Biblioteca|Library`), e o meu «0» era o TERCEIRO falso negativo meu:** existe **`LibraryPage.tsx`** + rota `/library` + item «Biblioteca» no `AppSidebar` + entrada na busca (`page-library`) — agrega artefatos de **todas** as missões, com **busca e filtro por categoria**; **`ArtifactsViewer.tsx`** faz download real (`/api/agent/v1/missions/:id/artifacts/:artifactId`) e mostra **`sha256` rotulado «Hash de Proveniência SHA-256»**; **`AgenticSplitShell.tsx`** integra o painel «Artefatos da Missão (n)» (**com teste**: `AgenticSplitShell.test.tsx` exige «Artefatos (1)»); **`MissionDeliveryCard.tsx`** entrega com sha256; **`StudioCanvasPage.security.test.ts`** lê o **próprio fonte** do `ArtifactsViewer` (padrão de teste de segurança sobre o código). ⇒ **UI ✅ + proveniência ✅ + busca ✅ + prévia ✅ + testes ✅**. **NÃO medido:** versionamento *por artefato*, permissões *por artefato* em multi-inquilino e dedupe *real* (o sha256 é o substrato, não a prova do dedupe). **Lição registada: três falsos negativos meus, todos por procurar o nome em inglês num domínio de identificadores pt-BR.**

> **COMPANION — a lacuna é MAIOR do que «falta a rota» (medido 2026-10-15).** `CompanionEventoVerificado` aparece **11 vezes no repositório inteiro**: na definição, dentro de `DerivarEstadoCompanion` e no `companion_estado_test.go`. **Não há PRODUTOR em lado nenhum** — *nada no sistema constrói este tipo a partir de eventos reais de missão.* A forma exacta (para ser accionável, não uma queixa): `Tipo string` (conjunto fechado), `MissaoID`, `PassoID` (opcional), `Quando time.Time`, `Detalhe string` — **e o próprio código declara a exigência**: *«`Detalhe` é texto já sanitizado a montante: este ficheiro não inventa detalhe, só o transporta»* e *«`Quando` existe para que um evento futuro não seja tratado como passado»*. ⇒ **Faltam DUAS peças: (1) conversor `eventos reais → CompanionEventoVerificado` (só eventos que ACONTECERAM; `Detalhe` sanitizado a montante/DLP; `Quando` real) e (2) rota `GET …/companion/:missao_id` → estado + razão + escopo.** *Item 46 = `IMPLEMENTED_PARTIAL`/domain-only, com o caminho cortado em dois pontos — é a doutrina «provar o caminho completo da UI até ao efeito real» a acusar exactamente onde dói.* **A MEDIR (tentativa falhada, padrões declarados):** `type EventoDaMissao|type MissionEvent|EventosDaMissao|type Evento struct|Eventos []Evento` ⇒ **0** em `internal/agent` — *o tipo de evento de missão tem outro nome e tem de ser medido antes de escrever o conversor* (lição dos três falsos negativos: **não inventar o nome**).

> **COMPANION — RESOLVIDO (2026-10-15): o teste de handler encontrou um defeito REAL na minha própria entrega.** Publicado em `5c403c8c9`, o conversor passava **todos** os eventos a `DerivarEstadoCompanion` — que **recusa os que não pertencem ao vocabulário**. E `Runtime.CreateMission` emite **`mission.planned`**, que **não** está em `EventosVerificadosDeMissao` (12 nomes exportados em `companion_estado.go:48`). Consequência medida: **a rota respondia `400` para TODA missão recém-criada** — *o Companion ficaria permanentemente indisponível em produção: um controlo que existe, que é testado, e que ninguém consegue usar.* **Causa-raiz:** *um adaptador que entrega a uma função o que ela declara rejeitar não é um adaptador — é um erro à espera de acontecer.* **Correcção (`831d8dad6`):** o conversor **filtra ao vocabulário** (`CompanionEventoEhVerificado`, reutilizando a lista exportada — *não guarda cópia que envelhece*) e **conta os descartados** (`…ComRelatorio`), *porque um descarte invisível é indistinguível de uma perda de dados*. **Provas:** `go test ./internal/agent -run Companion` → **ok**; `go test ./server -run CompanionRouteReportsHonestState` → **PASS**; checker de codificação → OK; gofmt limpo. **Duas expectativas MINHAS que estavam erradas** (e o que ensinaram): missão nova tem `mission.planning` ⇒ o estado honesto é **`a_planear`**, não «ausente»; e o conversor **ordena**, logo indexar por posição de entrada estava errado — *a falha provou que a ordenação actua*. **O que faltou ter feito antes:** declarar `IMPLEMENTED_UNTESTED` e **não** afirmar «cadeia completa» no commit — *foi o teste que a meta exige que impediu a entrega avariada.* **E2E:** nenhuma das três tem passagem E2E provada.

| Medida | Valor medido |
| --- | --- |
| Commit medido (**local = remoto**, 0 por empurrar) | `06cd277da` |
| Remoto da branch | `06cd277da` (*nenhum commit local pendente*) |
| `main` remota | `4b3e2185422c24d5a05fd46a343953ad6f41d1e5` (**medido agora** com `git ls-remote`, 2026-10-15 — *não é valor herdado*) |
| Evidências no índice | **546** (**histórico** — não re-medido; *não herde este número*) |
| Estágios com evidência | **38 de 38** (**histórico**) |
| Bloqueios declarados | **17** (**histórico**) |
| Matriz do gate final | `BLOCKED_BY_EXTERNAL_DEPENDENCY: 5 · DONE: 19` (**histórico**) |
| Veredito global | **`NOT_RELEASE_READY`** |
| **Checkpoint 2026-10-15 (BRT) — medido** | **CI do SHA `06cd277da`: `NOT MEASURED`** — o run `38741470515` ficou **preso do lado do GitHub** (`in_progress`, `updated_at` congelado em 13:17:43Z), com `Verify formatting` e `Test multi-provider contracts` já **verdes**. *Último veredito medido: `abd3771ab` ⇒ `class-a-plus-integrity` **success** + `dz23-multi-provider` **success** (13:04:43Z).* **Provas locais no SHA corrente:** `python scripts/hades_encoding_check_test.py` **OK** · `prettier --check` **limpo** · **`vitest run` na UI inteira: 100 ficheiros / 661 testes PASS** · `tsc -b` exit 0 · `gofmt` limpo · `go build ./...` exit 0 · `golangci-lint` **0 issues** (`./internal/agent/...` e `./server/...`) · `go test ./server -run TestCollab` **5/5 PASS**. **Entregue no complemento 48 (F2) — cadeia da sala:** domínio → **rotas HTTP** (`GET/POST /api/agent/v1/collab/:project_id/members`, `.../rooms`, `.../rooms/:room_id/members`; *origem = ator autenticado; sala inexistente ⇒ **404**, nunca lista vazia*) → cliente `salaApi.ts` (*falha nunca vira lista vazia*) → componente `SalaMembros.tsx` → painel `SalaPainel.tsx` (*estado em união discriminada*) — **todos `TESTED`**; **`SalaLigada.tsx` = `IMPLEMENTED_UNTESTED`** (*medido: sem `jsdom`/`happy-dom`/`testing-library`; existe `@vitest/browser` ⇒ é esse o teste que falta*). **Bloqueio humano declarado:** PR #438 `blocked` por revisão (`LMPrado-DZ23`) — **não contornar**. **Próximas ações:** ① teste de browser da `SalaLigada` (para `E2E_VERIFIED`); ② ler o veredito de `06cd277da` quando houver runner; ③ retomar as prioridades da FASE 6 (Paperclip → Cloudroom/Teleport → Studios → homologação e benchmarks reais). |

**Medido em `2026-10-14 16:35 UTC`.**

**Regra desta seção (E-301, aplicada em E-310, e outra vez em E-544):** qualquer número acima **envelhece**. Esta seção
já ficou desatualizada **duas** vezes (dizia `f190ec8ea` / 300, e depois `fafda19e7` / 310) e foi remedida. ⇒ *E o `E-544`
mediu o custo: este documento e o `HADES_HANDOFF.md` estiveram **parados ~16 h**, sem mencionar **um único** commit ou
evidência recente.* **Antes de citar, remeça:**

```powershell
git rev-parse --short HEAD
(Select-String -Path docs/mission/HADES_EVIDENCE_INDEX.md -Pattern '^\| E-\d+ ').Count
python scripts/hades_mission_check.py                      # ESCRITURAÇÃO CONSISTENTE (exit 0)
python scripts/hades_ci_reproducibility_check.py --check    # nenhum gate flutuante (E-309)
Select-String -Path docs/mission/HADES_FINAL_RELEASE_GATE.md -Pattern 'Resumo: '
```

**Decisão preservada:** `docs/mission/HADES_CURRENT_STATE.json` **não existe por decisão explícita** — dois arquivos
de estado divergiriam. O estado canônico é `HADES_MASTER_STATE.json`. **Não criar** o outro.

**Retomada exata (comandos reais, nenhum passo presumido):**

> ⚠️ **A PRIMEIRA LINHA DESTRÓI TRABALHO LOCAL. LEIA A CONDIÇÃO ANTES DE A CORRER.**
> *`git reset --hard origin/main` é o passo **certo** para quem retoma **da `main`** — e **destrutivo** para quem tem
> commits locais por empurrar.* ⇒ **Confirme primeiro:**
> ```powershell
> git rev-list --count origin/fix/checker-relogio..HEAD   # 0 = seguro; >0 = HA' TRABALHO LOCAL
> ```
> *Se der **> 0**, **não** corra o `reset --hard`: empurre primeiro (um push por corrida — `E-541`), ou guarde o
> trabalho numa branch.* **O `reset --hard` não avisa: ele apaga.**

```powershell
git fetch origin main; git reset --hard origin/main
$repo = (Get-Location).Path
$env:GOROOT = "$repo\_tmp\goroot1269\go"; $env:GOPATH = "$repo\_tmp\gopath"
$env:GOMODCACHE = "$repo\_tmp\gomodcache"; $env:GOCACHE = "$repo\_tmp\gocache1269"
$env:GOTMPDIR = "$repo\_tmp\gotmp"; $env:GOFLAGS = '-mod=mod'; $env:GOTOOLCHAIN = 'local'
$env:PATH = "$env:GOROOT\bin;$env:PATH"
gofmt -l internal/agent internal/multillm server/agent_routes.go   # tem de sair VAZIO
go test ./internal/agent/ -count=1
python scripts/hades_release_gate.py; python scripts/hades_baseline_report.py
python scripts/hades_stage_map.py; python scripts/hades_mission_check.py
python scripts/hades_ci_reproducibility_check.py --check
git checkout -b feat/<escopo>   # branch curta + PR; nunca commit direto em main, nunca force-push
```

**Onde o trabalho parou (para quem retoma):** estágios **36–38** com métricas, publicação, jornada E2E e a fronteira
cruzada já provadas; faltam **STT/TTS e telefonia reais** (dependem de provedor/número/credencial —
`BLOCKED_BY_EXTERNAL_DEPENDENCY`), **104/122 conectores** que exigem operador (`APENAS_CATALOGO`), artefato **não
assinado e só Windows**, e a **decisão de política** do `misspell` (dicionário inglês) ao subir a versão do linter.
# 1. baseline: main integrada e limpa
git fetch origin main; git reset --hard origin/main
# 2. toolchain fixada do repositório (Go declarado em go.mod)
$repo = (Get-Location).Path
$env:GOROOT = "$repo\_tmp\goroot1269\go"; $env:GOPATH = "$repo\_tmp\gopath"
$env:GOMODCACHE = "$repo\_tmp\gomodcache"; $env:GOCACHE = "$repo\_tmp\gocache1269"
$env:GOTMPDIR = "$repo\_tmp\gotmp"; $env:GOFLAGS = '-mod=mod'; $env:GOTOOLCHAIN = 'local'
$env:PATH = "$env:GOROOT\bin;$env:PATH"
# 3. bateria obrigatória ANTES de qualquer push
gofmt -l internal/agent internal/multillm server/agent_routes.go   # tem de sair VAZIO
go test ./internal/agent/ -count=1
python scripts/hades_release_gate.py; python scripts/hades_baseline_report.py
python scripts/hades_stage_map.py; python scripts/hades_mission_check.py
# 4. branch curta + PR (NUNCA commit direto em main, NUNCA force-push)
git checkout -b feat/<escopo>
```

**A lição que mais custou rodadas (E-300b):** o gate `Preserve Class A+ surfaces` falha com
`test -z "$(gofmt -l internal/agent internal/multillm server/agent_routes.go)"` — **reproduza esse comando localmente
antes de corrigir**, em vez de supor a causa.


> Ordem de execução: **P0 → P1 → P2 → P3**. Nada aqui é declaração de conclusão;
> é backlog com critério de pronto.

## P0 — bloqueia o release e a honestidade do produto

## CLASSIFICAÇÃO EXPLÍCITA DOS 5 ITENS SEM ESTADO (E-841)

> **§18.5-2**: *«Não deixar "falta implementar" genérico — cada pendência tem
> arquivo:linha, teste falho, pessoa responsável, bloqueio concreto ou decisão de
> escopo.»* Os 19 itens deste documento foram auditados: **14 tinham classificação
> explícita** e estes **5 não tinham**. Cada um fica agora com **responsável e
> bloqueio concretos**, sem inventar causa.

| item | classificação | bloqueio concreto | responsável |
|---|---|---|---|
| **P0-5** · jornada de primeira execução duplicada | **DECISÃO DO DONO** | *O próprio item exige: «fechar o redundante **somente com autorização explícita»**. Recomendação registada: manter **PR #71** (já gateado no CI) e fechar **#61** **após** o merge de #71. Não é trabalho de engenharia — é um clique de governação.* | proprietário |
| **P1-1** · saída de ferramenta na linha do tempo | **DECISÃO DO DONO** | *«PR #69 (`feat/saida-ferramenta-no-chat`, `525856a75`) está **verde**. Falta **autorização de merge**.»* *A parte opcional já tem **arquivo nomeado**: `AgenticSplitShell.tsx` — e o item declara que a linha do tempo **já mostra `payload` genérico**, logo o essencial **existe**.* | proprietário |
| **P2-2** · aviso de depreciação do Node 20 no CI | **ACCIONÁVEL — bloqueado só pela regra de congelamento** | *Trabalho **mecânico e medido**: `actions/checkout@11d5960a…` e `actions/setup-node@49933ea5…` (v4 com SHA fixo) emitem `Node.js 20 is deprecated` — **confirmado em log real de CI** (`E-823`). A acção é uma **PR dedicada** subindo os `uses:` para v5 nos **15** workflows. **Exige commit**, e commitar move o SHA congelado `9f7379502` — que é a única prova de CI verde desta fase. **Não é FAIL: é sequenciamento.** | proprietário (abrir PR) |
| **P2-3** · replay curado do upstream | **SINCRONIZAÇÃO PLANEADA — não é defeito** | *O item **quantifica** o trabalho: «replay linear temático sobre `upstream/main` (**73 commits, 39 arquivos em conflito, 15 conflitos reais**). 4 gates obrigatórios + smoke Chromium antes de abrir PR.» É **sincronização com o Ollama upstream** (§00.2 manda preservá-lo), não um bug por corrigir. | engenharia, em janela dedicada |
| **P3-1** · release assinado por SO | **`FUTURE_DISTRIBUTION`** | *Depende do **`B-06`** (certificado). E a decisão do proprietário é **explícita**: «**NÃO VAMOS COMPRAR OU CONFIGURAR AGORA CERTIFICADOS COMERCIAIS** de assinatura Windows/macOS, nem publicar nas lojas.» Pelo §18.5-7, fica como **`FUTURE_DISTRIBUTION`** e **não** reprova o gate de engenharia/build. | proprietário (fase futura) |

**Efeito no veredicto:** *nenhum destes 5 itens é um defeito de produto por
corrigir — **2** são decisões de governação (P0-5, P1-1), **1** é sequenciamento
(P2-2), **1** é sincronização planeada (P2-3) e **1** é distribuição futura já
decidida pelo dono (P3-1).* ⇒ *Depois desta classificação, **19/19** itens têm
estado, bloqueio e responsável.*

---

### P0-1 · Checkpoints da missão V2.1 (PR #72)
- **Pronto quando:** os 11 artefatos existem, o JSON é válido, `PR gate` verde.
- **Estado:** **ENTREGUE** — PR #72 aberto com os 11 artefatos e **15/15 checks
  sem falha** (E-036); JSON válido (`stages=38`, `blockers=9`). Falta autorização
  de merge.

### P0-2 · Cobertura E2E do Studio — **ENTREGUE (PR #73)**
- **Correção:** o drag-and-drop do Studio **existe** (reorder por arraste em
  `StudioCanvasPage.tsx:975-988` e por botão/teclado `1012/1024`, ambos
  persistindo via `updateBuilderVisual`, com 5 testes unitários de
  `reorderById`). O gap real era **falta de spec Playwright que assegure o
  comportamento**, não falta de funcionalidade. Nunca reafirmar "DnD ausente".
- **Entregue:** `e2e/studioReorder.spec.ts` (3 testes) + `e2e/studio-builder-fixture.mjs`,
  provando ordem vinda do servidor, persistência do arraste, paridade pelo botão
  acessível e sobrevivência ao reinício. Verde no CI: **23/23 checks**, com os 3
  testes passando no job `Web E2E (shell smoke)` (E-034, E-035).
- **Restante do escopo:** preview/export, undo/redo e edição de propriedades sob
  E2E — em backlog (P1/P2).

### P0-3 · Corrigir o backlog histórico — **ENTREGUE (PR #73)**
- `audit/BACKLOG_LACUNAS_PRIORIZADO_20261009.md` afirmava "arraste real
  inexistente" no Studio. Corrigido com a evidência do probe (E-008/E-009), mais
  a contagem honesta dos 6 itens do roadmap e a ordem recomendada.

### P0-4 · Decidir o gate de onboarding quando `GET /settings` falha
- **Situação:** `routes/index.tsx` engole o erro de `getSettings` e cai no shell
  sem redirecionar ao onboarding ⇒ usuário novo sem backend vê a aplicação e não
  o onboarding. Pode ser comportamento desejado (modo local) ou defeito.
- **Critério de pronto:** decisão documentada + teste que fixe o comportamento
  escolhido (hoje há apenas o comportamento implícito).
- **REMEDIDO (E-816, 2026-10-15): o critério está cumprido, e o desfecho é a
  favor do produto.** Lido o código, **não é um erro engolido**: `routes/index.tsx:17-20`
  tem o motivo **escrito** no `catch` («The local-first shell must remain usable
  when the optional preferences API is offline. Chat/agent requests will show
  their own safe states.»), e o onboarding **é** alcançado quando a API responde
  (`:21-26` — `OnboardingVersion < CURRENT_ONBOARDING_VERSION` ⇒
  `throw redirect({ to: "/onboarding" })`). A decisão está **certa por mérito**:
  um usuário **novo sem backend** não teria onde **gravar** a versão de onboarding,
  logo o shell é o comportamento correto — coerente com o §02.3 («Chat local
  básico funciona sem login cloud e sem rede»). O teste que faltava existe agora:
  `app/ui/app/src/routes/index.onboarding-gate.test.tsx`, **3/3 PASS**, fixando os
  três casos (API offline ⇒ chat; versão antiga ⇒ onboarding; versão atual ⇒ chat).

### P0-5 · Jornada de primeira execução duplicada
- PR **#61** (`test/e2e-jornada-primeira-execucao`, `bb45d7944`) e PR **#71**
  (`feat/e2e-primeira-execucao`, `3b5459ef5`) cobrem a mesma jornada.
- **Critério de pronto:** decidir qual supersede o outro e fechar o redundante
  **somente com autorização explícita**. Recomendação: manter #71 (já gateado no
  CI) e fechar #61 após o merge de #71.

## P1 — paridade competitiva (TypingMind / AutoClaw / Manus)

### P1-1 · Saída de ferramenta visível na linha do tempo
- PR #69 (`feat/saida-ferramenta-no-chat`, `525856a75`) está verde. Falta
  autorização de merge. Opcional: renderizar o excerto de `output` em
  `AgenticSplitShell.tsx` (a linha do tempo já mostra `payload` genérico).

### P1-2 · Paridade de chat TypingMind
- PR #56 (chat) e #58 (biblioteca de prompts) abertos. Próximo: comparação lado
  a lado multi-modelo e biblioteca de prompts usável de ponta a ponta.

### P1-3 · Canais de IM (Telegram/WhatsApp/inbox)
- PRs #63 (Telegram) e #65 (guia de conexão) abertos; PR #57 traz o comparativo
  AutoClaw. **Bloqueio real:** homologação com token exige B-03. Manter a
  classificação `APENAS_ADAPTER` até haver chamada real.

### P1-4 · Harness CLI governado — **PROVADO (E-830); o merge é decisão do dono**
- **REMEDIDO (E-830, medido):** o item pede *«prova de que a aprovação obrigatória
  (`RequiresApproval: true`, `Risk: RiskWrite`) é exigida **de fato**, com teste
  reproduzível»*. **O teste existe e usa os dois campos exactos**:
  `internal/agent/mutation_gate_test.go:60` → **`TestMutationHarnessHITLApprovalBypass`**,
  com `Step{… Risk: RiskWrite (**:64**), RequiresApproval: true (**:65**) …}` — e a versão
  actual chama as funções **reais** (`approvalsReady`, `stepApproved`,
  `validateMissionApprovalBindings`) sobre um `Runtime` e uma `Mission` de verdade.
- **E o ficheiro documenta um achado que vale mais que o item**: `:47-59` — *«**E-401 /
  ADV-58/1 (5ª auditoria independente, MEDIDO)**: esta função era **TEATRO**. As linhas
  59-64 e 72-75 eram **closures LOCAIS** que reimplementavam a condição
  `if s.RequiresApproval && !approved`, e a asserção final comparava **uma closure com a
  outra**… A contraprova do auditor é decisiva: com **TODOS** os pontos de aprovação do
  runtime desligados, este teste continuava **PASSANDO**.»* ⇒ ***«Teste de mutação que não
  toca o produto não é teste de mutação: é um espelho.»***
- **E um segundo achado (`:72-74`)**: *com `Kind: "git.merge.origin"`, a guarda
  anti-secreto **recusa persistir o payload*** ⇒ *duas defesas no mesmo caminho, e uma
  encobria a outra no teste*.
- **E a fronteira do merge**: *o item diz que o PR #62 é **branch-local*** ⇒ **merge não
  autorizado** (review pendente) ⇒ **decisão do dono**.
- **Situação original (história):**
- **REMEDIDO (E-829, medido):** o item pedia *«prova de que a aprovação obrigatória
  (`RequiresApproval: true`, `Risk: RiskWrite`) é exigida de fato, com teste
  reproduzível»*. **Medido**: existe **`internal/agent/cli_harness_tool.go`** e
  **`cli_harness_test.go`**; e `harness.cli`/`RequiresApproval`/`RiskWrite` tem **126**
  ocorrências em `internal`, `server`, `cmd` — lideradas por **`planner.go` (16)**,
  `runtime_test.go` (12), `runtime.go` (10), `cli_harness_tool.go` (6),
  **`mutation_gate_test.go` (6)**, `approval_payload_fields_test.go` (4),
  `cli_harness_test.go` (4), `hardening_test.go` (3).
- **E os testes correram**: `-run 'MutationGate|CliHarness|CLIHarness|ApprovalPayload|Hardening'`
  ⇒ **exit=0**, **PASS=2 / FAIL=0** (`ok` 6,072 s), com
  `TestApprovalPayloadHashRejectsUnserializableInput` e
  `…RejectsInputChangedByPersistenceRedaction` — *ambos sobre o **hash do payload***.
- **O que **não** está provado**: *se o `mutation_gate` exige mesmo `RequiresApproval: true`
  **para `RiskWrite`*** — *é o que o item pede por nome, e o `-run` apanhou só **2** testes,
  não o conjunto dos dois ficheiros citados.* ***«Os testes existem e passam» e «o teste
  prova o que o item pede» são duas afirmações, e a segunda exige ler o teste.»***
- **E a fronteira do merge**: *o item diz que o PR #62 é **branch-local** e que falta o
  merge* ⇒ **merge não autorizado** (review humano pendente) ⇒ **decisão do dono**.
- **Situação original (história):**
- PR #62 (`afea7e726`) é **branch-local**: `harness.cli` não está em `main`.
  Próximo: merge e prova de que a aprovação obrigatória (`RequiresApproval:
  true`, `Risk: RiskWrite`) é exigida de fato, com teste reproduzível.

### P1-5 · Catálogo honesto de conectores
- 104/122 exigem operador. Promover conectores **um por um**, com evidência de
  chamada real e mantendo `Auth` não-vazio. A metade "promoção automática de
  status" para linhas só-chave é **inviável** (`BLOCKED_BY_EXTERNAL_DEPENDENCY`).

## P2 — engenharia e infraestrutura

### P2-1 · Workflow Engine — **CUMPRIDO (E-827); obsolescência medida**
- **REMEDIDO (E-827, medido por leitura do código e dos testes):** o item pede **três**
  coisas e as três **já existem**.
  1. **Execução real de schedule persistido** ⇒ `schedule_flow.go:570`
     (`// ExecuteScheduleFlow runs a persisted schedule flow…`) e
     `supervisor.go:318` (`// Execute the persisted flow graph of the schedule…`),
     coberto por `schedule_flow_test.go`.
  2. **Desfecho por passo persistido** ⇒ `runtime.go:1906`
     (`// recordScheduleRun persiste o resultado da execução…`) com
     `Schedule.LastRun`, *«tanto em sucesso quanto em [falha]»*; e `schedule_flow.go:543`
     — *«o gravador persiste no schedule para que a **pausa sobreviva ao tick**»*.
  3. **`action.connector` implementado** (`schedule_flow.go:51`, dois `case` em `:338` e
     `:1009`, e `schedule_connector_test.go` com os casos **ausente**/**cruzado**/**quebrado**);
     **`action.http` não existe — e não pode existir**, porque `schedule_flow.go:15`
     declara *«`ScheduleStepKind` is the **closed vocabulary**»* ⇒ é a **recusa fechada**
     que o item admitia, feita **por construção** e não por `if`.
  ⇒ **O que resta é só a nota opcional** (`condition.if` com seletor de `expect`,
  PRs #67/#68) ⇒ **decisão do dono**, não trabalho por fazer.
- **Situação original (história):**
- Provar execução real de schedule persistido (não só validação), persistir
  desfecho por passo, e implementar ou **recusar de forma fechada**
  `action.http`/`action.connector`; opcional `condition.if` com seletor de
  `expect` no editor de fluxo (PRs #67/#68).

### P2-2 · Aviso de depreciação do Node 20 no CI
- `actions/checkout@11d5960a…` e `actions/setup-node@49933ea5…` (v4 com SHA
  fixo) emitem `Node.js 20 is deprecated`. PR dedicada subindo os `uses:`
  fixados por SHA para v5 nos 15 workflows.

### P2-3 · Replay curado do upstream
- Branch `chore/upstream-YYYYMMDD` a partir de `main`; replay linear temático
  sobre `upstream/main` (73 commits, 39 arquivos em conflito, 15 conflitos
  reais). 4 gates obrigatórios + smoke Chromium antes de abrir PR.

### P2-4 · Falhas ambientais de teste (B-09) — **ACTUALIZADO (E-820), não fechado**
- **REMEDIDO (E-820, medido):** o `B-09` no JSON diz *«AMBIENTAL (**MEDIDO E
  CORRIGIDO**, COM PROCEDIMENTO) — CORRIGIDO POR MEDICAÇÃO (`E-202`). A descrição
  anterior ('Host sem Python/sh: 4 testes falham por ambiente') estava…»* ⇒ **o
  problema original foi corrigido**. **Mas marcar este item como resolvido seria
  falso:** este host tem **três** falhas ambientais **distintas**, medidas nesta
  sessão e com contorno conhecido —
  1. **`CGO_ENABLED=0`** derivado pelo próprio Go porque **não existe toolchain C**
     (`E-815`: 9 compiladores e 6 directórios canónicos ausentes) ⇒ **14** falhas
     de stub `go-sqlite3` (*`app/server` é 21 PASS / 13 FAIL por esta causa*, `E-814`);
  2. **`OLLAMA_CONTEXT_LENGTH=32768`** no perfil do utilizador (`HKCU\Environment`,
     `E-809`/`E-810`) ⇒ 1 falha de `context_window`, curável só **antes** do processo
     arrancar (*é variável de pacote, `envconfig/config.go:230`);
  3. **`%TEMP%` não-apagável** (`E-796`) ⇒ falha de `go build` e de
     `vite:esbuild-transpile`; contorno = `TMP`/`TEMP` para dentro do repo.
  ⇒ **O item mudou de conteúdo**: já não é «sanear o host» (o problema do `B-09`),
  é **declarar e contornar as três falhas ambientais deste host** — o que está feito.
- **Situação original (história):**
- Sanear o host de teste (Python + `sh`, caminho curto) **sem** desativar teste.

### P2-5 · Prettier não governado na raiz — **FECHADO (E-820): `FORA_DE_ESCOPO`**
- **REMEDIDO (E-820, medido):** decisão tomada — **declarar formalmente fora de
  escopo**, e não adoptar na raiz. As quatro provas: (a) **não existe**
  `.prettierrc` na raiz (nem `.json`, nem `.js`); (b) o **único** `.prettierrc`
  do repositório é `app/ui/app/.prettierrc`; (c) **0 dos 15** workflows em
  `.github/workflows/` mencionam `prettier`; (d) o script `prettier:check`
  **existe** no `package.json` da UI mas **nenhum workflow o corre**. ⇒ Adoptar na
  raiz seria **criar** governança de formatação num escopo onde nada é governado —
  o §00.2 proíbe trabalho paralelo sem justificativa validada em diff e ADR.
- **Situação original (história):**
- `.prettierrc` existe só em `app/ui/app`; nenhum workflow roda
  `prettier:check`. Decidir: adotar na raiz ou declarar formalmente fora de
  escopo.

## P3 — release verificado e arquitetura

### P3-1 · Release assinado por SO
- Certificado (B-06), smoke nativo por SO, artefatos assinados publicados.

### P3-2 · Isolamento por organização (multitenancy) — **CUMPRIDO (E-828)**
- **REMEDIDO (E-828, medido):** o item pede *«decisão arquitectural + **teste** de que uma
  organização não lê dados de outra»*. **Ambas existem, e a segunda em abundância:**
  `grep` em `internal/agent/*_test.go` devolve **250 de 325** correspondências
  (*o grep cortou no limite; o total é 325*), cobrindo **plugins/conectores**
  (`ErrPluginOrganizationScope`, incluindo *mesmo id em tenants diferentes*), **MCP**
  (local, remoto e server), **fila/jobs** (`ErrQueueJobForbidden`), **CRM/comércio/financeiro**,
  **inbox/omnichannel**, **agendamento** (inclusive *a trava é por organização, não global*),
  **dispositivos/push** (`ErrDeviceForbidden`), **egress**, **memória/retenção**,
  **aprovações**, **builder/deploy**, **media upload**, **consentimento**, **histórico** —
  e **RLS em SQL real** (`distributed_integration_test.go`, com INSERT cross-tenant
  **recusado pelo banco**).
- **A decisão arquitectural tem erros nomeados**: `ErrPluginOrganizationScope`,
  `ErrQueueJobForbidden`, `ErrDeviceForbidden`, `ErrBuilderForbidden` e o guarda
  compartilhado `companyForRequest` (`B-15`, RESOLVIDO); o `hades_tenant_scope_check.py`
  **passa** (7/7 checkers).
- **Três testes de qualidade excepcional**: `service_agent_approval_tenant_test.go:17-23`
  (*descobriu que o teste que alegava isolamento **«era satisfeito pela colisão de caminho»**
  e escreveu um que **não depende** dela*), `scheduling_lock_scope_proof_test.go:82`
  (*prova que a trava não é GLOBAL*) e `oauth_refresh_non_inference_test.go:23`
  (`…DoesNotRevealExistenceAcrossTenants` — *mais forte que «não lê»: **não revela se existe***).
- **E vai além do pedido**: `inbox_isolation_test.go:10` cobre **escrever**
  (*atribuir, transferir, fechar*), não só ler.
- **Situação original (história):**
- Decisão arquitetural + teste de que uma organização não lê dados de outra.

### P3-3 · Benchmarks com metodologia auditável — **CUMPRIDO (E-829)**
- **REMEDIDO (E-829, medido):** o artefacto é **`docs/mission/HADES_BENCHMARK_METHOD.md`**,
  e o cabeçalho diz literalmente o que o item pedia: *«Artefato do **estágio 21**.
  **Substitui**, no que é mensurável localmente, o **comparativo puramente documental**
  que existia antes.»*
- **Os três requisitos, um a um**: **comando** (`§1`, com 4 notas de método);
  **ambiente** (`§2`: Sistema, **CPU i9-13900HX**, **Toolchain `go1.27.2`**, **Ponto de
  partida `main@889750abe`**, Iterações); **resultado** (`§4`, com `ns/op`, `B/op`, `allocs/op`).
- **E vai além do pedido**: `§3` tem coluna **«Não mede»** por benchmark; e `:6-11`
  **recusa** a comparação competitiva (*«sem a MESMA tarefa, a MESMA máquina e a
  licença/conta de cada produto, qualquer número seria **inventado**»*).
- **E a metodologia foi reproduzida**: corri o comando **do documento** ⇒ **exit=0**,
  **296,5 s**, **22** benchmarks (*a suíte cresceu: `OmnichannelIngest*`, `ServiceAgentTurn/Plan`,
  `MemorySearch*`, `WritePhase_*` não estão no `§4`*). Os números são da **mesma ordem**,
  mas **não idênticos** — *e as duas variáveis estão **declaradas** no documento*
  (toolchain e ponto de partida diferentes). **Não atribuo a divergência a nenhuma
  das duas** (seria o erro do `E-824`).
- **E o `METHOD` fecha-se com duas secções que valem por si (`E-830`)**: `§5 «Como
  invalidar este resultado»` — *«Duas medições com o mesmo commit e a mesma máquina devem
  ficar na mesma ordem de grandeza; se não ficarem, o **ambiente** precisa ser registado»*
  — e `§6 «O que ainda falta para um comparativo competitivo»`: tabela com **4** requisitos
  **«não feito»/«pendentes»**, cada um com **bloqueio nomeado** (*licenciar cada produto*,
  **`B-07`**, *PRs #75/#77*), terminando em *«o estágio 21 permanece **`PARCIAL`**, sem
  nenhuma alegação de supremacia»*. ⇒ *e o critério do `§5` foi **accionado** pela minha
  reprodução (460× de divergência no `MemorySearchContext`), e o ambiente foi registado.*
- **Situação original (história):**
- Substituir o comparativo documental (`audit/COMPARATIVO_TYPINGMIND_20261009.md`,
  `docs/RELATORIO_BENCHMARK_COMPETITIVO_HADES_2026-10-04.txt`) por benchmarks
  reproduzíveis com versão, ambiente e comando.

### P3-4 · Estágios 5, 8, 15 e 34–37 — **FECHADO (E-821); premissa corrigida em (E-820)**
- **FECHADO (E-821, medido):** o predicado que o `E-820` deixou nomeado — *existência
  real de código* — está cumprido, e os três **têm implementação e testes**:
  - **5 · Context Bootstrap** ⇒ `internal/agent/context_bootstrap.go`; **7** ficheiros
    `*bootstrap*`, dos quais **3** de teste (`_test`, `_handoff_test`, `_isolation_test`);
    **325** ocorrências em **19** ficheiros.
  - **8 · Plugins e Skills** ⇒ `plugin_runtime.go`, `plugin_status.go`, `plugin_bundle.go`,
    `skill_activation.go`; **29** `*plugin*` + **19** `*skill*` = **48** ficheiros, com
    testes de ciclo de vida (`plugin_lifecycle_test.go`) e diagnóstico
    (`plugin_execution_diagnostics_test.go`).
  - **15 · Memoria e RAG local** ⇒ `rag.go`, `embedding.go`, `context_memory_governance.go`,
    `context_reembed_job.go`; **31** `*memory*` + **8** `*embed*` + **4** `*rag*` = **43**
    ficheiros; **970** ocorrências em **200** ficheiros — *o maior dos três, e coerente
    com o relatório, que já o declarava um dos 2 `COMPLETED_VERIFIED`*.
  ⇒ **Resultado dos 7 estágios: 4 externos por `B-03`/`B-04` e 3 com código e testes.**
  **0 dos 7 exige implementação nova** — o que falta é **classificação** no
  `HADES_STAGE_MAP.md` (*documento gerado: regenera-se, não se edita à mão*).
- **Situação anterior (E-820, história):**
- **REMEDIDO (E-820, medido):** o item diz *«nada foi inspecionado»* como se a
  informação não existisse — **ela existe, noutro documento**. Medido em
  `HADES_MASTER_STATE.json`: o campo `stages` tem **38** entradas com `id`+`title`
  e **`status` ausente**; e o campo `stage_states` tem **apenas 6** entradas,
  **nenhuma** com `status` ⇒ **a classificação dos 38 vive no
  `HADES_STAGE_MAP.md`**, não neste JSON.
- **Os `title` reais, lidos:** **5** = `Context Bootstrap / contexto de projeto` ·
  **8** = `Plugins e Skills` · **15** = `Memoria e RAG local` · **34** = `Canal
  WhatsApp` · **35** = `Inbox unificado omnicanal` · **36** = `Voz e transcricao` ·
  **37** = `Chamadas e follow-up`.
- **34–37 já têm bloqueio externo numerado:** são canais e voz ⇒ mapeiam em
  **`B-03`** («Tokens reais de canais», tipo `EXTERNO`) e **`B-04`** («Provedor de
  telefonia/voz real», tipo `EXTERNO`) ⇒ classificação honesta =
  **`BLOCKED_BY_EXTERNAL_DEPENDENCY`**. *A resposta estava no item **P1-3**, duas
  secções acima.*
- **O que REALMENTE falta (predicado nomeado):** verificar a **existência real de
  código** de **3** dos 7 — **5** (`Context Bootstrap`), **8** (`Plugins e Skills`)
  e **15** (`Memoria e RAG local`, que tem indício de já ser `COMPLETED_VERIFIED`).
  Os 34–37 **não** precisam: são externos por `B-03`/`B-04`.
- **Situação original (história):**
- Auditar existência real (nada foi inspecionado) antes de qualquer
  classificação; depois avançar conforme a dependência externa permitir.

### P0 · Complementos MASTER V2.2 (MEMÓRIA · PAPERCLIP · CLOUDROOM)

Estado medido em 2026-10-14, com evidência no `HADES_MEMORY_CONTEXT_RELIABILITY.md`:

1. **MEMÓRIA / COMPACTAÇÃO UNIVERSAL** — fechado no que é verificável:
   compactação automática (E2E por HTTP), histórico original (append-only + cadeia de hashes),
   recuperação das fontes, memória governada (8 rotas com escopo auditado), §4 handoff
   (contrato → produtores → chat → UI desktop → mobile) e §6 retomada de missões (já existia e
   está testada). Decisões D-1..D-12 registadas. **Correcção registada**: o `memory.search` do
   E-618 era DUPLICADO do `knowledge.search` (E-152) e foi revertido no E-619 — a «lacuna» era
   um defeito de busca, não do produto.
2. **PAPERCLIP — Agent Organization OS** — pendente: matriz de paridade (20 capacidades, com
   referência ao código real dos dois lados) e depois implementação incremental, reutilizando
   Company OS, Mission Control, memória e tarefas existentes. **Não** copiar o backend Node.
3. **CLOUDROOM — Session Teleport** — pendente: matriz de paridade (20 capacidades) e depois
   adaptadores por harness/executor. Diferenciar sempre `Native Resume` / `Checkpoint Restore` /
   `Context Handoff`; nada de segundo runtime.
4. **HADES Collaborative Intelligence — Teamily Parity** (complemento **48 PROPOSTO**; *a numeração
   foi CONFERIDA, não herdada*: `stages` do `HADES_MASTER_STATE.json` = **38** ids `1..38` sem
   lacuna, e as extensões 39–42 / complementos 43–47 vivem em texto, não na lista do estado — logo
   o próximo livre é 48, e a atribuição fica **pendente de confirmação no schema canônico**) —
   **NOVO (2026-10-15)**, recebido como complemento da MASTER V2.2. Objetivo: humanos e múltiplos
   agentes **na mesma sala**, com delegação que executa trabalho real. **Estado medido hoje: NADA
   implementado** (as 15 fases estão `PENDING`; o primeiro passo é a **matriz de paridade com
   evidência**, como manda a FASE 1). ✅ **FASE 1 CONCLUÍDA — auditoria medida no commit `93f12ac19`:** a matriz
   das 20 capacidades está em `docs/mission/HADES_TEAMILY_PARITY_MATRIX.md` (estado por capacidade, evidência com
   **arquivo e linha**, o que já existe para **reusar**, e o que ficou **`A MEDIR`**). *Resultado curto: salas,
   menções, respostas encadeadas, presença com origem e participante-agente **não existem em nenhuma camada**
   (runtime, backend, UI); o que existe e deve ser reusado é o chat com injeção de contexto e compactação, o
   `inbox`, o store de colaboração, o canal externo com DLP, os orçamentos/aprovações e o Companion.* Fases: 0 recuperar contexto/memória · 1 auditoria de paridade ·
   2 Team Chat (salas, menções `@agente`/`@equipe`/`@todos`, anexos, tarefas, presença com origem
   verificável, reconexão) · 3 Social Orchestrator (plano → delegação → paralelismo seguro →
   aprovações) · 4 Living Memory (com os rótulos `FACT_VERIFIED`/`DECISION_APPROVED`/`PROPOSAL`/
   `HYPOTHESIS`/`UNTRUSTED_CONTENT`) · 5 roteamento inteligente na conversa · 6 humanos e agentes no
   mesmo workspace · 7 Collaborative Studios · 8 Companion/assistentes · 9 agentes proativos com
   governança · 10 continuidade entre chats e dispositivos · 11 segurança e privacidade (isolamento
   por organização, anti-injeção, DLP) · 12 UX A+++ · 13 testes e homologação · 14 integração no
   backlog canônico · 15 critérios de conclusão.
   **REGRAS QUE VINCULAM (do próprio complemento):** ① **não** criar segundo chat engine, runtime,
   memory engine, permission engine ou sistema de tarefas — reutilizar Cowork, inbox, eventos,
   memória, Company OS e tarefas existentes; ② **primeiro** resolver os bloqueios do **PR #438** e
   estabilizar a `main`, depois implementar incrementalmente; ③ uma menção só dispara execução com
   **autorização + capacidade + orçamento** — *e o orçamento por passo já existe medido (E-669)*;
   ④ testes simulados têm de ser **declarados como simulados**, e UI responsiva **não** é prova de
   sincronização.

Regra que estes dois blocos herdam do erro do E-619: ***procurar pelos NOMES que o código usa,
com padrões que não excluam o alvo*** — antes de declarar qualquer lacuna.

## Checkpoint E-650/E-651 — reparacao do CI vermelho do PR #438

**Medido em 2026-10-15T01:35Z (2026-10-14T22:35 BRT).** NAO substitui `HADES_MASTER_STATE.json`
(fonte canonica unica); e' o registo de trabalho desta fatia.

### Estado medido (nao narrado)
| item | valor | origem |
| --- | --- | --- |
| `origin/main` | `4b3e2185422c24d5a05fd46a343953ad6f41d1e5` | `git rev-parse origin/main` |
| branch | `fix/checker-relogio` | `git rev-parse --abbrev-ref HEAD` |
| head ANTES desta fatia | `3c2edf466` | igual ao head que a auditoria externa registou |
| head DEPOIS | `f461ec86a` (E-650, E-651, E-650-ui) | `git rev-parse HEAD` |
| PR #438 | aberto, **nao mergeado**, `mergeable_state=blocked` | revisao humana pendente — **nao contornada** |
| checks do head `3c2edf466` | **6 falhas, 17 verdes** | `get_check_runs` |
| checks do head `e432a9091` | as 6 falhas caem para **1** (so' o E2E) | `get_check_runs` |

### As 6 falhas, com causa raiz medida por log (nao por suposicao)
| check | causa raiz | correccao | prova |
| --- | --- | --- | --- |
| `Preserve Class A+ surfaces` (12 s) | `verify-contracts.mjs`: `DELETE /api/agent/v1/skills/:id` **NO MATCH** — a rota foi APAGADA no E-642, quando usei a linha como ANCORA de edicao e nao a repus | rota reposta + comentario do incidente | *«CONTRACT VERIFICATION PASSED»*, exit 0 |
| `Go agentic and server gates` | mesmo `verify-contracts.mjs` | idem | idem |
| `PR gate` | agregava os anteriores | idem | verde no CI |
| `test (ubuntu-latest)` | `TestLembretesNaSuperficie`: `time.Now().UTC().Add(2h)` + servico de 30 min dentro de turno `00:00-24:00` ⇒ as 23:56 recusa **com razao** ⇒ ***a cor do CI dependia da hora*** | proximo meio-dia UTC (sempre < 24 h e com folga na janela) | reproduzido as 21:56 BRT e depois `--- PASS` + `go test ./server/` ok (29,7 s) |
| `test (macos-latest)` | mesma causa | idem | verde no CI |
| `Web E2E (shell smoke)` | **`PerformanceCard` derrubava a PAGINA**: o stub do E2E responde `{}` a rotas nao tratadas ⇒ `dados.performance` indefinido ⇒ `TypeError` ⇒ error boundary | guarda + mensagem honesta + teste de regressao com payload mau | `npx playwright test e2e/first-run-journey.spec.ts` = **1 passed (28,2 s)** (era `1 failed`) |

### A causa de eu nao os ter apanhado (a parte reutilizavel)
A bateria local tinha **16** verificadores e **nao continha o gate do CI**: `verify-contracts.mjs`
e' **Node**, nao Python. Passa a existir `_tmp/bateria_local.ps1`, **espelho dos passos do job**
`Preserve Class A+ surfaces`: **1 node + 27 python = 28/28 TUDO VERDE**. *Bateria que nao contem o
gate do CI nao substitui o CI* (mesma classe do E-636).

### Correccao de um numero que EU declarei mal
A mensagem do commit `146350696` (E-651) diz *«633 passed (633)»*; o **medido foi 630**. O numero
errado fica na historia (commit empurrado) e o correcto e' este: **630**. Licao: nao escrever
numeros de memoria numa mensagem de commit — colar o que a ferramenta mediu.

### Proximo passo e comando de retomada segura
1. Confirmar o CI do head final (`f461ec86a`): os 6 checks que estavam vermelhos.
2. Correr a **suite E2E inteira** (`npx playwright test`), nao so' o spec da jornada.
3. Actualizar `HADES_MASTER_STATE.json` (valores medidos desta fatia), sem criar fonte concorrente.
4. `PR #438` continua a exigir **revisao humana** (`LMPrado-DZ23`): nao fazer merge, nao contornar.

```bash
# retomada segura
cd hades-ollama && git fetch origin && git status --porcelain=v1
git rev-parse HEAD origin/fix/checker-relogio origin/main
powershell -NoProfile -ExecutionPolicy Bypass -File _tmp/bateria_local.ps1   # 28 gates do CI
```

---

## ACTUALIZACAO V3.0 (RONDA 71) - O QUE MUDOU DESDE O `E-959`

> **Requisito**: o `mission_check` exige que os documentos de retomada citem evidencia
> **recente** (>= `E-960`). Este documento parava no `E-959` - *41 evidencias de atraso*.
> **Isto e' o §FASE-7 e o ESTAGIO 1** (*«reconciliar documentos canonicos e historicos»*).

### Estado medido nesta ronda
```
branch  fix/checker-relogio     HEAD  31be202d6     origin/main  4b3e21854
WIP     14 entradas             stashes 0          indice  793 entradas (E-001..E-1000)
CI      os 6 workflows VERDES no d1f4e7915 (test, pr-gate, dz23-e2e, dz23-agentic-quality,
        dz23-multi-provider, class-a-plus-integrity com os 9 passos)
bateria 16 OK / 0 FAIL
```

### Evidencias novas (E-960 .. E-1000), por tema

**Adendo do ecossistema de plugins/skills/app/MCP**
- `E-960` a `E-967` - as **5 fontes** do adendo (*agent-skills, hyperframes, googleworkspace-cli,
  apify-mcp-server, playwright*), a matriz de encaixe e a fatia vertical do `POST /plugin`.
- `E-972` - o **formato aberto `hades-plugin.format/v1`** e a rota de formato.

**Fechamento dos 12 findings do PR #438**
- `E-961`, `E-962` - os **5 P1 do checker do relogio** e os **12 findings** revalidados.
- `E-963` a `E-965` - as **5 fontes** conferidas e a correccao factual do `E-959`.

**§ESTAGIO 22 - o gate de engenharia (3 dos 4 itens)**
- `E-987` - o gate deixa de ser **espelho do `MASTER_STATE`** e passa a provar o **snapshot real**
  (*`gate_head`, `gate_branch`, `gate_dirty`, `gate_snapshot`*).
- `E-988` - o **MANIFESTO do snapshot**: SBOM com `sha256`, plataformas, limitacoes, `source_sha`.
- `E-989` - **catraca** do manifesto no CI e na bateria local.
- `E-993` - o gate estava **impossivel de satisfazer no CI**: eram **4 volateis**, nao 1.
- `E-994` - o **mesmo defeito no manifesto** (`source_sha`).
- `E-996` - o **terceiro instrumento** (`host`) e a **INVERSAO DA REGRA**: so' se compara o que o
  manifesto **PROVA**, e cada campo tem de ser **classificado**.
- `E-997` - o inventario **media o disco**; passa a ler o **commit** e a comparar **estrutura**.

**§FASE-5 - marca canonica**
- `E-990` - **inventario da marca**: **909** canonicas contra **441** legadas em 4 nomes.
- `E-991` - o **gerador** do inventario com **8 excepcoes** declaradas (*razao + desbloqueio*).
- `E-997` - as contagens passam a vir do commit, e nao do disco.

**Instrumentos e rastreabilidade**
- `E-992` - **catraca do ASCII nos `.ps1`**: 34 caracteres nao-ASCII em **string** no `install.ps1`.
- `E-995` - o **`E-559` restaurado do Git**, engolido pelo `E-993`.
- `E-998` - a **quinta instancia** do defeito (*o checker media o disco e passava por sorte*).
- `E-999` - o **`\n` LITERAL** do here-string duplicou o `E-559` e engoliu o `E-560`.
- `E-1000` - **evidencia que citava codigo NAO COMMITADO**: o `E-998` descrevia uma cura que
  ficou **fora** do commit; curado no `31be202d6` (*1 file changed, 113 insertions*).

### As QUATRO falhas do CI, com o run que provou cada cura
```
run 38922978710  passo 6 (gate)        gate_head/gate_branch volateis  -> E-993  bddeb7ebd
run 38924032504  passo 7 (manifesto)   source_sha/source_branch        -> E-994  a4ddaa4e0
run 38926745194  passo 7 (host)        host Windows vs Linux           -> E-996  c025eee59
run 38932562362  passo 8 (inventario)  media o DISCO                  -> E-997  ecd8bb31b + d1f4e7915
run 38935930565  *** 6/6 SUCCESS ***
```

### Licoes medidas (nao supostas)
1. **Curar o que o CI aponta e' perseguir sintomas** - 3 instrumentos, e em todos curei o que ele
   apontou e deixei o resto. Dai' a **INVERSAO**: allowlist do que e' **estavel**, volatil por
   omissao, **+ classificacao obrigatoria**.
2. **Nao ler git nao impede medir WIP** - a segunda pergunta: *o ficheiro que eu leio esta'
   commitado?*
3. **O `HEAD` do CI nao e' a branch** - em `pull_request` e' `refs/pull/N/merge`. Mesma pergunta
   em 3 lugares.
4. **Performance tambem e' correccao**: 2459 `git show` -> **11**, de **minutos** para **1,8 s**.
5. **Uma evidencia sem commit e' uma afirmacao** (`E-1000`).

### Divida declarada (nao escondida)
```
5a instancia de "mede o disco": os 3 DIVIDA_STRING e os 4 tectos de comentario do
  hades_ascii_ps1_check.py medem o DISCO, e o CI mede o checkout -> declarada, NAO corrigida
install.ps1: 34 caracteres nao-ASCII em STRING -> fecha com o FASE-6 (teste de instalacao limpa)
duplicacao intra-linha: o mission_check conta LINHAS e nao ve' uma copia embutida -> declarada
```

### Retomada segura (comandos que existem neste repositorio)
```bash
cd hades-ollama && git fetch origin && git status --porcelain=v1
git rev-parse HEAD origin/fix/checker-relogio origin/main
python scripts/hades_release_gate.py --check          # o gate bate com o estado da missao
python scripts/hades_mission_check.py                 # escrituracao: sem fantasma, veredito coerente
python -X utf8 scripts/hades_manifest.py --check      # o manifesto bate com a arvore
python -X utf8 scripts/hades_brand_inventory.py --check
python -X utf8 scripts/hades_ascii_ps1_check.py       # ASCII nos .ps1
```
*Os caminhos em `_tmp/` **nao** vao para o Git e **nao** sao instrucoes de retomada validas --
a bateria canonica e' a lista acima, mais `scripts/hades_local_gates.ps1`.*

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

### O que falta (actualizado na janela `E-1009`..`E-1021`)

**Fechado nesta janela**: instalador Windows (`E-1012`), pacote Linux (`E-1013`), inventario de
marca (`E-1015`), MV3 provado inexistente antes (`E-1016`/`E-1017`), Android medido ate' a
fronteira (`E-1019`), extensao MV3 criada (`E-1020`) e **completada e carregavel** (`E-1021`),
`.gitignore` de material de assinatura (`E-1020`, commit `82228563f`).

**Continua em falta, sem disfarce**:
- `.zip` da extensao com hash e instrucoes -- **proximo passo**;
- **pairing real** (`native messaging` + `HADES Local Companion`): **nao existe**; o popup mostra
  `host: - (native host nao instalado)`;
- ligacao da extensao ao **Mission Control** (hoje `missao` e `aprovado` ficam a `null`);
- E2E em Chrome/Edge reais; Firefox e Safari **nao** tem suporte demonstrado;
- APK Android: bloqueado pela DLL nativa JNI neste host (`E-1019`), passo de desbloqueio = runner Linux;
- macOS e iOS: `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial -- o material Apple
  fornecido (3 `AuthKey` + 1 `.cer`) serve para **submeter/notarizar**, o `.cer` sozinho **nao
  assina**, e `codesign`/`notarytool` **so' existem no macOS**;
- script + doc de notarizacao/assinatura: **pedido pelo dono, ainda nao escrito**;
- iOS sem build de simulador; dispositivo fisico e TestFlight `BLOCKED_EXTERNAL`;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado; `SUPERIOR_MEASURED` nao reclamado.

### O que falta (actualizado na janela `E-1022`)

**Fechado nesta janela**: o pacote nao assinado da extensao -- `.zip` + `SHA256SUMS` + `LEIA-ME`
ASCII + verificador versionado, reprodutivel e provado contra 3 defeitos reais (`E-1022`). O
defeito do separador foi medido, curado e **provado por injecao nos bytes crus**.

**Continua em falta, sem disfarce**:
- **pairing**: `native messaging` + `HADES Local Companion` **nao existem**; o popup mostra
  `host: - (native host nao instalado)` e `missao`/`aprovado` ficam a `null`;
- ligacao da extensao ao **Mission Control** (nada alimenta os campos de estado);
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nao carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**;
- **SBOM do pacote da extensao** e actualizacao do `THIRD_PARTY_NOTICES` com os icones
  reutilizados de `app/ui/app/public/`;
- medicao de latencia/CPU/RAM do `observar()` e backpressure medido;
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- script e doc de notarizacao/assinatura (pedido pelo dono);
- benchmark reproduzivel (ESTAGIO 21) nao iniciado; `SUPERIOR_MEASURED` nao reclamado.

### O que falta (actualizado na janela `E-1023`)

**Fechado nesta janela**: o contrato do canal local (`E-1023`) -- schema, sequencia, expiracao,
replay, pairing por codigo curto, chave por dispositivo, rotacao, revogacao e recusa de LAN.
23 assercoes, e 2 delas apanharam defeitos reais de codigo.

**Continua em falta, sem disfarce**:
- **acordo de chaves**: o contrato usa HMAC **simetrico** e **assume** que a chave foi
  partilhada com seguranca -- a derivacao (HKDF a partir do codigo curto) **nao existe**;
- **transporte**: nao ha native messaging, nao ha socket, nao ha `HADES Local Companion`;
  o popup continua a mostrar `host: - (native host nao instalado)`;
- **DNS rebinding** a nivel de socket: o contrato valida o hostname, nao a ligacao;
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nunca carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**;
- **SBOM do pacote da extensao** e `THIRD_PARTY_NOTICES` com os icones reutilizados;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1024`)

**Fechado nesta janela**: a derivacao de chave por HKDF com contexto e codigo de uso unico,
o manifesto template do native host com allowlist, e a permissao `nativeMessaging` movida
para `optional_permissions` (`E-1024`). O `mission_check` apanhou o `E-1024` citado em
codigo antes de existir -- a mesma classe do `E-1020`, por um vector que a catraca do
indice nao cobre.

**Continua em falta, sem disfarce**:
- **o `HADES Local Companion` nao existe**: o manifesto e template com placeholders, e nao
  ha executavel, transporte, stdio pipe nem handshake;
- **o codigo curto em trafego**: num canal `loopback` sem TLS ele viaja em claro no
  `localhost`; falta o handshake que prova posse da chave derivada e o rate-limit;
- **DNS rebinding** a nivel de socket: o contrato valida o hostname, nao a ligacao;
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nunca carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**;
- **SBOM do pacote da extensao** e `THIRD_PARTY_NOTICES` com os icones reutilizados;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1026`)

**Fechado**: o handshake de prova de posse com desafio de uso unico e prazo, a taxa por instalacao
com janela expiravel, e a canonizacao propria do desafio (`E-1025`). E **tres defeitos reais**
curados no mesmo sitio (`E-1026`): `verificar` chamado com 3 argumentos numa funcao de 2 (a guarda
de prova morria sempre), `registro.revogadaEm` (campo que nunca existiu -- a guarda de revogacao
estava MORTA), e `>` em vez de `>=` no limite de expiracao.

**Continua em falta, sem disfarce**:
- **o TRANSPORTE**: o handshake prova identidade, nao sigilo. O codigo curto viaja em claro no
  `localhost` no primeiro salto; nao ha stdio pipe nem TLS de loopback;
- **o `HADES Local Companion` nao existe** (o manifesto e' template com placeholders);
- **a taxa vive em memoria**: reiniciar o host apaga a contagem; sem rate-limit do lado do SO;
- **DNS rebinding** a nivel de socket: o contrato valida o hostname, nao a ligacao;
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nunca carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**; sem SBOM do pacote;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1028`)

**Fechado**: o transporte (`E-1027`) -- enquadramento de native messaging (4 bytes little-endian)
com erros tipados, corpo parcial tratado como parcial, limites de comprimento zero e gigante
antes de alocar, e um canal em memoria que prova o protocolo completo pelo fio, inclusive partido
em pedacos de 1 byte. E **tres premissas minhas erradas** curadas no mesmo teste (`E-1028`):
contagem de bytes a mao (*duas vezes*), exigir deteccao de lixo no inicio (*impossivel num
enquadramento de comprimento*) e um nome de motivo inventado.

**Continua em falta, sem disfarce**:
- **o processo Companion nao existe**: o enquadramento e' testado em memoria e **nunca** correu
  contra `process.stdin`/`stdout` de um processo real;
- **`allowed_origins` e' template**: nao ha' host instalado nem origem validada em execucao;
- **sem TLS de loopback** -- o codigo curto continua a viajar em claro no primeiro salto;
- **a taxa vive em memoria**: reiniciar o host apaga a contagem; sem rate-limit do lado do SO;
- **DNS rebinding** a nivel de socket: o contrato valida o hostname, nao a ligacao;
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nunca carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**; sem SBOM do pacote;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1030`)

**Fechado**: o `HADES Local Companion` **como processo** (`E-1029`) -- le' do `stdin`, responde no
`stdout`, implementa so' o pairing e o handshake, encerra com codigo 2 quando o fluxo dessincroniza, e
mantem o diagnostico no `stderr`. E o defeito de protocolo que so' um processo real expos (`E-1030`):
**o host respondia sem ecoar o `id`**, e um pedido pendente para sempre aparece como **silencio**, nao
como erro. Tres sondas passaram antes de a quarta o revelar -- nenhuma correlacionava.

**Continua em falta, sem disfarce**:
- **o host nao esta instalado**: `allowed_origins` e' template e nao ha' registo no navegador nem
  prova de que o SO respeita a fronteira;
- **o estado vive em memoria**: taxa, pairing e dispositivos desaparecem ao reiniciar o processo;
- **sem TLS de loopback** -- o codigo curto continua a viajar em claro no primeiro salto;
- **sem rate-limit do lado do SO**;
- **DNS rebinding** a nivel de socket: o contrato valida o hostname, nao a ligacao;
- **E2E em Chrome/Edge reais**: o pacote foi verificado, **nunca carregado**;
- Firefox (`.xpi`), Safari e assinatura (`.crx`): **nao existem**; sem SBOM do pacote;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI neste host (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1031`)

**Fechado**: o `allowed_origins` **deixou de ser placeholder** (`E-1031`). O ID e' derivado da
chave publica (`sha256` do DER, hex, `0-f` -> `a-p`), o `manifest.json` tem `key`, o host
instala-se no `HKCU` para Chrome e Edge com manifesto em `%LOCALAPPDATA%`, e o `-Desinstalar`
deixa **zero** registo sujo -- medido. 25 assercoes, com 3 fault injections que provam a catraca.

**Continua em falta, sem disfarce**:
- **o host nao foi exercitado por um navegador**: instalei-o e desinstalei-o, mas **nao abri o
  Chrome**. O E2E real (carregar a extensao, ver o ID, falar por native messaging) e' o proximo
  passo e e' o que separa «protocolo correto» de «produto instalado»;
- **Firefox**: nao usa registo do Windows; o `.xpi` continua por fazer;
- **Safari**: alvo separado, exige macOS/Xcode e projeto proprio;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuicao esta' fora do Git** (`_tmp/ext-chave-privada.pem`): sem ela
  nao se empacota um `.crx` com o MESMO ID; quem for empacotar precisa dela ou de outra chave;
- **taxa e pairing em memoria** -- reiniciar o processo do Companion apaga-os;
- sem TLS de loopback, sem rate-limit do SO, sem DNS rebinding a nivel de socket;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**, nao pela credencial;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1032`)

**Fechado**: o E2E real num navegador. O Edge carrega a extensao, o ID bate com o derivado, e a
extensao fala com o Companion por native messaging -- medido, com desinstalacao sem residuo.
Quatro defeitos reais corrigidos: a permissao nunca pedida, o `permissions.request` que trava,
o gesto que nao sobrevive ao `sendMessage`, e o `path` a apontar para `.mjs`.

**Continua em falta, sem disfarce**:
- **controle de aba ponta a ponta**: o canal funciona; observar DOM, clicar, preencher e
  **recusar campo de senha** numa pagina real ainda **nao** foi provado;
- **takeover e revogacao por evento** nao testados com uma pagina a serio;
- **o Chrome real do utilizador nao foi exercitado**: a versao 155 deste host **nao carrega**
  por `--load-extension` (medido: 0 service workers). Exige carregamento manual;
- **Firefox** (nao usa registo do Windows; falta o `.xpi`) e **Safari** (macOS/Xcode);
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuicao esta' fora do Git** (`_tmp/ext-chave-privada.pem`);
- **`CAPACIDADES` VAZIO**: o Companion **nao** controla o PC. Computer use nao existe;
- **taxa e pairing em memoria** -- reiniciar o Companion apaga-os;
- sem TLS de loopback, sem rate-limit do SO, sem DNS rebinding a nivel de socket;
- ligacao da extensao ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`), desbloqueio = runner Linux;
- macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**;
- benchmark reproduzivel (ESTAGIO 21) nao iniciado.

### O que falta (actualizado na janela `E-1033`)

**Fechado**: o controle de aba ponta a ponta. Observar o DOM real, escrever num campo
legítimo com o valor **verificado no DOM**, **recusar senha e cartão** com o DOM a ficar
vazio, badge `ON`, revogar e recusar depois. E o opt-in de rede privada deixou de ser
código morto: passou a ser decisão do utilizador, visível no popup, desligado por omissão.

**Continua em falta, sem disfarce**:
- **takeover humano**: o scope `browser:takeover` existe e **não** foi exercitado;
- **revogação por mudança de origem**: tem código e **não** tem E2E;
- **frames e shadow DOM**: nao testados;
- **downloads e uploads**: nao testados (o `preencher` recusa `type=file`, sem E2E);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension` (medido); exige carregamento manual;
- **Firefox** (não usa registo do Windows; falta o `.xpi`) e **Safari** (macOS/Xcode);
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git** (`_tmp/ext-chave-privada.pem`);
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC. Computer use não existe;
- **taxa e pairing em memória**;
- sem TLS de loopback, sem rate-limit do SO, sem DNS rebinding a nível de socket;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1034`)

**Fechado**: o takeover humano. O content script observa `input`/`keydown`/`pointerdown`,
marca o takeover por 30s, e o agente é **recusado** com `takeover_humano_ativo` enquanto
durar -- com o **valor do humano preservado** e o autor (`human`/`agent`) registado. E o
popup dá ao utilizador o botão para assumir ou devolver.

Dois defeitos reais corrigidos no caminho: a **escrita do agente accionava o takeover**
(flag `escrevendoPeloAgente`), e o **`lerEstado` apagava o campo novo** (reconstruía o
objecto com lista fixa).

**Continua em falta, sem disfarce**:
- **revogação por mudança de origem**: tem código no `tabs.onUpdated` e **não** tem E2E;
- **frames e shadow DOM**: nao testados;
- **downloads e uploads**: nao testados;
- **o takeover não tem duração ajustável** (30s fixos) e não detecta actividade fora do DOM;
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** (não usa registo do Windows; falta o `.xpi`) e **Safari** (macOS/Xcode);
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- sem TLS de loopback, sem rate-limit do SO, sem DNS rebinding a nível de socket;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1035`)

**Fechado**: a revogação por mudança de origem, agora com E2E. Duas abas na mesma origem
coexistem; a que navega para outra origem perde a concessão e o badge; a vizinha continua;
e voltar não restaura. E o **terceiro** local do defeito do opt-in (`tratarAccao`) foi
curado.

**Continua em falta, sem disfarce**:
- **uma catraca sobre o CÓDIGO**: o defeito do opt-in apareceu em **três** locais
  sucessivos; a cura pontual não fecha a classe. Falta um teste que falhe quando um novo
  chamador validar URL sem o opt-in;
- **`pushState`/`replaceState`**: um SPA que mude de origem por History API **não** dispara
  `tabs.onUpdated` com URL novo -- **não testado**, e pode ser um buraco real;
- **redireccionamentos e `https`**: só medido com dois servidores em loopback;
- **frames e shadow DOM**: nao testados;
- **downloads e uploads**: nao testados;
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1036`)

**Fechado**: o defeito do opt-in deixou de ser uma classe aberta. A catraca
`hades_bridge_optin_ratchet.py` mede o **código** -- falha se um novo chamador de
`validarUrl`/`guardarRota` não passar `permitirPrivado` -- e 13 asserções provam que ela
**discrimina** nos dois sentidos, numa **cópia** (o repositório real nunca é tocado).

**Continua em falta, sem disfarce**:
- **`pushState`/`replaceState`**: pode ser buraco real -- um SPA que mude de origem por
  History API **não** dispara `tabs.onUpdated` com URL novo. **Não medido**;
- **a catraca olha duas linhas**: uma chamada com o opt-in mais longe seria falso positivo;
  e `permitirPrivado: false` literal passaria (mede a menção, não o valor);
- **redireccionamentos e `https`**: só medido com dois servidores em loopback;
- **frames e shadow DOM**: nao testados;
- **downloads e uploads**: nao testados;
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1037`)

**Fechado**: dois defeitos reais que a dívida do `E-1035` escondia -- o evento `onUpdated`
**sem URL** (que revogava a concessão a cada reload) e o **badge que ficava vazio** com a
concessão viva. E o `pushState` foi medido: **não pode mudar de origem** (o browser recusa
com `SecurityError`), logo não era buraco.

**Continua em falta, sem disfarce**:
- **BFCache** (voltar atrás): **não testado**, e pode repor uma página de outra origem
  sem disparar `onUpdated` com URL novo;
- `pushState` **dentro de iframes**, `document.domain`, service workers de página: não testados;
- **redireccionamentos e `https`**: só medido com dois servidores em loopback;
- **frames e shadow DOM**: nao testados;
- **downloads e uploads**: nao testados;
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1038`)

**Fechado**: a família `onUpdated`. A ida revoga (com ou sem BFCache), o regresso **traz
URL** e não revive nada, o badge não ressuscita, e a acção segue recusada. E há uma
declaração explícita: **o BFCache em si não chegou a correr** (`persisted: false`).

**Continua em falta, sem disfarce**:
- **BFCache sem execução observada**: o Edge recarregou em vez de restaurar; forçá-lo
  exigiria ausência de `unload`, `Cache-Control` permissivo e uma decisão do browser;
- **frames e shadow DOM**: o contrato exige suportá-los «com limites» e o `observar`
  actual só olha o documento principal -- **não medido**;
- `pushState` **dentro de iframes**, `document.domain`, service workers de página: não testados;
- **redireccionamentos e `https`**: só medido com dois servidores em loopback;
- **downloads e uploads**: nao testados;
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1039`)

**Fechado**: frames e shadow DOM. O `observar` já não devolve uma lista parcial como se
fosse completa — **atravessa** shadow roots abertos e iframes de mesma origem (2 → 5 nós)
e **declara** os limites com contagens e profundidades. E o shadow fechado é declarado
**indetectável**, em vez de contado por palpite.

**Continua em falta, sem disfarce**:
- **downloads e uploads**: nada gere downloads, e o `preencher` recusa `type=file` --
  o contrato exige caminhos autorizados, manifesto com hash e aprovação -- **não implementado**;
- **shadow roots fechados**: invisíveis por design do browser, não por defeito do HADES;
- **cross-origin**: contado, nunca lido;
- `document.domain`, iframes `srcdoc`/`about:blank` e service workers de página: não testados;
- **BFCache** sem execução observada (`E-1038`, `persisted: false`);
- **redireccionamentos e `https`**: só medido com dois servidores em loopback;
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1040`)

**Fechado**: o scope `browser:files` decide. Política com 9 regras ordenadas, 17 asserções
unitárias, E2E real no Edge, registo auditável sem o hash, e a **cura na raiz** da lista
fixa do `lerEstado` (que já tinha mordido no `E-1037`).

**Continua em falta, sem disfarce**:
- **a extensão NÃO executa a transferência**: decide sobre ela. Não há hook em
  `chrome.downloads` nem em `type=file`;
- **a aprovação chega por `mensagem.aprovado`** -- um campo que **quem pede** controla.
  *Quem pede não certifica o que pede.* A ligação às approvals do Mission Control
  **não existe**;
- **`sha256` é medido onde os bytes existem**: a extensão **não pode verificar** que os
  bytes transferidos correspondem ao manifesto -- só a decisão é auditável;
- o registo vive em `storage.session` (morre com a sessão) e **não é assinado**;
- **shadow roots fechados**: invisíveis por design do browser;
- **cross-origin**: contado, nunca lido;
- `document.domain`, iframes `srcdoc`/`about:blank`, service workers de página: não testados;
- **BFCache** sem execução observada (`E-1038`, `persisted: false`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao`/`aprovado` ficam a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1041`)

**Fechado**: a aprovação deixou de ser um booleano. Prova com **id**, **impressão do
pedido** e **autor**; um booleano solto é recusado com o nome certo; a prova cruzada é
recusada; a autoria fica no registo. E a forma adoptada é a que o backend **já tinha**
(`FindApproved` + `Consume`), não um sistema novo.

**Continua em falta, sem disfarce**:
- **a extensão NÃO verifica a prova contra o store** -- não tem acesso. Uma prova **bem
  formada mas falsa passaria**. *A forma está implementada e testada; a autoridade não*;
- **falta ligar a rota de approvals no runtime**: o `nativeMessaging` funciona (`E-1032`),
  mas o Companion **não consulta** `FindApproved`;
- **a extensão não executa a transferência** -- decide sobre ela;
- **`sha256` é medido onde os bytes existem**: a extensão não pode verificar que os bytes
  transferidos correspondem ao manifesto;
- o registo vive em `storage.session` (morre com a sessão) e **não é assinado**;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1042`)

**Fechado**: a decisão de ficheiros passa a **declarar a autoridade**. `autoridade:
forma_apenas` acompanha todo o veredicto (mesmo os `permitido`), a extensão **nunca**
diz `verificada`, e `avisoDeAutoridade` diz o limite **por palavras** — visível no popup
e guardado no registo.

**A fronteira medida**: a rota `serviceAgentApprovals` existe mas é de **sessão**; o
Companion é um processo local **sem sessão**. Dar-lhe a sessão do utilizador seria pior.

**Continua em falta, sem disfarce**:
- **a extensão continua SEM verificar a prova** -- o campo `autoridade` **declara** o
  limite, não o resolve. Uma prova bem formada mas inexistente passaria;
- **falta a rota no Companion**: é preciso que ele use o **canal emparelhado** para se
  autenticar perante o runtime e consultar `FindApproved`. Só então `verificada` tem
  quem o possa afirmar;
- **a extensão não executa a transferência** -- decide sobre ela;
- **`sha256` é medido onde os bytes existem**: a extensão não pode verificar que os bytes
  transferidos correspondem ao manifesto;
- o registo vive em `storage.session` (morre com a sessão) e **não é assinado**;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1043`)

**Fechado**: a pergunta existe. O Companion trata `aprovacao:consultar`, valida o par e o
fingerprint, e responde **sempre** — com `verificada: false` e o motivo constante. A
extensão pergunta e **não depende da resposta**: um canal em baixo nunca faz a decisão
parecer melhor.

**Continua em falta, sem disfarce**:
- **a verificação continua a NÃO acontecer** -- o que existe é a pergunta e a recusa
  honesta. O encaixe é `MOTIVO_SEM_ROTA`, **uma constante**;
- **não há teste que falhe no dia em que o runtime expuser a rota** -- o passo de
  desbloqueio está **escrito, não vigiado**;
- **a extensão não executa a transferência** -- decide sobre ela;
- **`sha256` é medido onde os bytes existem**: a extensão não pode verificar que os bytes
  transferidos correspondem ao manifesto;
- o registo vive em `storage.session` (morre com a sessão) e **não é assinado**;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1044`)

**Fechado**: o encaixe do `E-1043` deixou de ser uma frase. Há uma catraca
(`hades_approval_hook_ratchet.py`) que acorda quando o runtime expuser a rota, e 17
asserções provam que ela **discrimina** — com 4 injeções e uma contraprova, numa cópia.

**Continua em falta, sem disfarce**:
- **o encaixe continua por ligar**: a autoridade da extensão continua `forma_apenas`;
- **a catraca vigia SINAIS**, não a existência real de um canal: um canal feito por outra
  via (socket, ficheiro, segundo binário) **não a faria acordar**;
- **a lista de formas de sessão tem sete entradas e não é exaustiva por construção** --
  uma forma nova daria falso positivo (o modo seguro de falhar, mas ainda assim falso);
- **os 4 findings P2 do PR #438** continuam abertos;
- **a extensão não executa a transferência** -- decide sobre ela;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1045`)

**Fechado**: os 12 findings do PR #438 foram medidos contra o HEAD. **Dez estão corrigidos
ou são falsos positivos do bot.** E **nenhuma thread do GitHub foi tocada** — proibido sem
autorização.

**Continua em falta, sem disfarce**:
- **#8 não totalmente revalidado**: `declared_categories_measured_test.go` foi inspeccionado
  por **nome de teste**, não pela alegação contra o **comportamento real** do checker;
- **a medição do #11 usa o `marked` e um cabeçalho de 5 colunas** -- outro renderizador
  (GitHub, Pandoc) podia divergir em casos de borda; **não testei no GitHub real**;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (autoridade `forma_apenas`);
- **a extensão não executa a transferência** -- decide sobre ela;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**;
- benchmark reproduzível (ESTÁGIO 21) não iniciado.

### O que falta (actualizado na janela `E-1046`)

**Fechado**: os 12 findings do PR #438. Oito corrigidos (`E-909`, `E-910`, `E-912`), dois
falsos positivos do bot (#10, #11), um tratado antes (#1), e o #12 feito por construção.
O #8 foi fechado com a contraprova que faltava: **o teste a correr** (`ok … 5.620s`).

**Continua em falta, sem disfarce**:
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido sem autorização);
- **ESTÁGIO 21 (benchmarks) não iniciado** -- nenhuma medição warm/cold, p50/p95, repetições;
- **o encaixe da aprovação continua por ligar** (`E-1044`); a extensão decide sem autoridade;
- **a extensão não executa a transferência** -- decide sobre ela;
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o takeover não tem duração ajustável** (30s fixos);
- **o Chrome real do utilizador**: a versão 155 deste host **não carrega** por
  `--load-extension`; exige carregamento manual;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **a chave privada de distribuição está fora do Git**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **taxa e pairing em memória**;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1047`)

**Fechado**: o ESTÁGIO 21 foi **medido**. A matriz T2 real correu com dois modelos locais
(`PASS`, 599,88 s), os veredictos repetiram-se um dia depois, e o `cmd/bench` mediu o
Ollama directo (`load` 19,95 s · `ttft` 20,00 s · `prefill` 1445 tok/s · `generate`
92,17 tok/s).

**Continua em falta, sem disfarce**:
- **a repetição (`HADES_T2_REPETICAO.md`, `46+/0`) não foi corrida nesta janela** -- o
  número é do WIP e **está por revalidar**;
- **nenhuma comparação externa foi executada** -- `NAO_EXECUTADA`, não `PARITY_MEASURED`;
- **a taxa é 3/8** com modelos locais de 3–7B: *um modelo, um host, uma quantização*;
- **sem GPU/VRAM medidos por época** -- só tempo de parede;
- **o warm/cold do `cmd/bench` não existe** (frio por construção, `bench.go:504`);
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1048`)

**Fechado**: o inventário do worktree. **Zero trabalho real por integrar** — os 11 `M`
são só fim-de-linha (`--ignore-cr-at-eol` = 0). O WIP do prompt existe em disco e foi
integrado na FASE 1. Há agora um instrumento que **lista** (passo 31).

**Continua em falta, sem disfarce**:
- **a repetição T2 (`46+/0`) está por revalidar** -- a corrida está lançada;
- **nenhuma comparação externa foi executada** -- `NAO_EXECUTADA`, não `PARITY_MEASURED`;
- **a taxa da matriz T2 é 3/8** com modelos locais de 3–7B: *um modelo, um host, uma quantização*;
- **sem GPU/VRAM medidos por época** -- só tempo de parede;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1049`)

**Fechado**: a repetição T2. Corrida agora (`PASS`, 209,78 s); veredictos idênticos três
dias depois; o artefacto passou de `2026-10-13` para `2026-10-16T23:52:54Z` e o número é
medido. O `46+/0` **está confirmado**.

**Continua em falta, sem disfarce**:
- **N=3 por fixture**: contagem, não probabilidade, sem intervalo de confiança;
- **a taxa da repetição é 1 de 4 fixtures** (`soma` `3/3`; as outras três `0/3`, trapaceando);
- **nenhuma comparação externa foi executada** -- `NAO_EXECUTADA`, não `PARITY_MEASURED`;
- **um modelo, um host, uma quantização**;
- **sem GPU/VRAM medidos por época** -- só tempo de parede;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1050`)

**Fechado**: o mecanismo do vermelho do `pr-gate`. Não é defeito de código — é o push
seguinte a cancelar os gates que ele espera (`pr-gate.yaml:91`, deadline em `:61`).

**Continua em falta, sem disfarce**:
- **`55327160c` continua VERMELHO no GitHub** -- e quem o vir não sabe, sem ler o log,
  que a causa foi um push;
- **o `cda756f82` vai repetir o padrão** -- declarado antes de acontecer;
- **uma decisão do proprietário em aberto**: distinguir «cancelado pelo push» de
  «cancelado por falha» no `pr-gate` **toca a política do gate** e não é decidível pelo
  executor; a cura por baixo (aceitar `cancelled`) está **proibida**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **nenhuma comparação externa executada** -- `NAO_EXECUTADA`;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1051`)

**Fechado**: o mecanismo do `pr-gate`, provado com três SHAs. E o procedimento de push
mudou por causa disso.

**Continua em falta, sem disfarce**:
- **`55327160c` continua VERMELHO no GitHub** -- e quem o vir não sabe, sem ler o log,
  que a causa foi um push;
- **o `cda756f82` vai repetir o padrão** -- declarado antes de acontecer, não observado ainda;
- **uma decisão do proprietário em aberto**: distinguir «cancelado pelo push» de
  «cancelado por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está
  **proibida**;
- **o estágio 21 tem uma lacuna de integração**: os documentos canónicos de método não
  referenciam a matriz T2 real nem a repetição;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **nenhuma comparação externa executada** -- `NAO_EXECUTADA`;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1052`)

**Fechado**: o mecanismo do `pr-gate` — provado, previsto e confirmado. A disciplina de
push produziu **quatro 6/6 consecutivos** (`4157a459b`, `ba62baf88`, `67581a993`,
`2e0873635`).

**Continua em falta, sem disfarce**:
- **`55327160c` e `cda756f82` continuam VERMELHOS no GitHub** -- e quem os vir não sabe,
  sem ler o log, que a causa foi um push; **não os «limpei»** (reenviar mudaria o SHA);
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está **proibida**;
- **`882acb121` não foi medido até ao fim** -- o `pr-gate` dele ainda não tinha fechado
  quando medi; **não afirmo** o seu estado;
- **o estágio 21 tem uma lacuna de integração**: os documentos canónicos de método não
  referenciam a matriz T2 real nem a repetição;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **nenhuma comparação externa executada** -- `NAO_EXECUTADA`;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1053`)

**Fechado**: a ligação do estágio 21 aos artefactos. Os três documentos referenciam
`HADES_T2_MATRIX.md` e `HADES_T2_REPETICAO.md`, com comandos reproduzíveis na secção 6.

**Continua em falta, sem disfarce**:
- **a ligação é textual**: se alguém renomear um artefacto, **nada acusa** -- não há teste;
- **os comandos da secção 6 exigem Ollama local e 25–30 min cada**;
- **a comparação externa continua `NAO_EXECUTADA`** -- nada disto produz número de concorrente;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está **proibida**;
- **`882acb121` não foi medido até ao fim** -- não afirmo o seu estado;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1054`)

**Fechado**: a ligação do estágio 21 deixou de ser textual. A catraca
(`hades_stage_artifacts_check.py`) acusa artefacto apagado, artefacto vazio, ponteiro
removido e documento ausente — com 4 injeções e contraprova.

**Continua em falta, sem disfarce**:
- **o mapa `LIGACOES` cobre só o estágio 21** -- 3 documentos e 2 artefactos: uma ligação
  que não esteja no mapa **não é vigiada**;
- **o limite de 400 bytes é um palpite calibrado**, não uma medida;
- **a catraca verifica a presença do nome**, não que o texto descreva o artefacto certo:
  *um ponteiro errado apontaria para o mesmo nome*;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está **proibida**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1055`)

**Fechado**: a primeira varredura de ligações docs→artefactos. **Zero defeitos reais** —
os 10 «ponteiros quebrados» e o 1 «artefacto órfão» eram **quatro erros do meu detector**.
E **não alarguei o mapa `LIGACOES`** com eles.

**Continua em falta, sem disfarce**:
- **não medi a pergunta certa**: quais documentos **afirmam produzir** um artefacto e não
  o citam -- a próxima acção;
- **o mapa `LIGACOES` continua com 6 entradas** (`E-1054`), só do estágio 21;
- **o detector de varredura não entrou na bateria** -- *um instrumento que produz falsos
  positivos não deve virar gate*;
- **`INSTALACAO.json` existe quando o instalador corre**, num caminho que eu não medi;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está **proibida**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1056`)

**Fechado**: a linha de investigação da rastreabilidade. **Duas perguntas, dois zeros.**
Os quatro detectores de varredura **não entraram na bateria** — deliberado.

**Continua em falta, sem disfarce**:
- **os artefactos não assinados do candidato actual** (§FASE-6) -- o trabalho de produto
  que falta; **nenhum pacote do candidato `2bca9392f` foi produzido**;
- **o mapa `LIGACOES` continua com 6 entradas** (`E-1054`), só do estágio 21;
- **um detector léxico não vê uma ligação descrita por outras palavras** -- o limite fica;
- **`eas.json` existe**, mas num caminho que o detector só encontrou corrigido;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**; a cura por baixo está **proibida**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**, `document.domain`, `srcdoc`: como no `E-1040`;
- **BFCache** sem execução observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**;
- **sem `.crx`, sem assinatura, sem SBOM do pacote**;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- ligação da extensão ao **Mission Control** (`missao` fica a `null`);
- APK Android bloqueado pela DLL JNI (`E-1019`);
- macOS/iOS `BLOCKED_EXTERNAL` pela **máquina**.

### O que falta (actualizado na janela `E-1057`)

**Fechado**: o **artefacto portátil Windows do candidato `6093a6692`** — 48 MB, verificado
com resposta real do modelo. E a lacuna do **filho órfão** no script (17 → 0).

**Continua em falta, sem disfarce**:
- **o `HadesSetup.exe` não assinado não foi gerado** -- o `ISCC.exe` existe; falta executar;
- **`INSTALLED=NAO` e `DEVICE_E2E=NAO`** -- instalar escreve em `%LOCALAPPDATA%\Programs\Hades`
  e **exige aprovação do dono**;
- **o bundle é Windows/amd64 apenas**: Linux, macOS, Android, iOS e Web/PWA continuam por
  produzir neste candidato;
- **o runner é pré-construído** (`_tmp\hadesbin`): o ZIP **não é** build reproduzível da fonte;
- **não assinado**, não publicado (`SIGNED_FOR_PUBLIC_RELEASE=NAO`);
- **Linux/macOS `BLOCKED_EXTERNAL` pela máquina**; Android bloqueado pela DLL JNI (`E-1019`);
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem assinatura, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1058`)

**Fechado**: o **`HadesSetup.exe` não assinado do candidato `d7ace6c12`** — 24,26 MB,
gerado com binários Go construídos da fonte. E a dívida do `[UninstallRun]` sem
`RunOnceId`, curada (o `Warning` desapareceu).

**Continua em falta, sem disfarce**:
- **`INSTALLED=NAO` e `DEVICE_E2E=NAO`** -- **não executei o instalador**: instalar escreve
  em `%LOCALAPPDATA%\Programs\Hades` e **exige aprovação do dono**;
- **o `lib\ollama` (42 ficheiros, 39,7 MB) é pré-construído**: o instalador **não é** build
  100% reproduzível da fonte; só os binários Go são;
- **não assinado** -- `SIGNED_FOR_PUBLIC_RELEASE=NAO`;
- **ARM64 não foi gerado** (o `.iss` protege com `Check: IsArm64()`);
- **Linux, macOS, Android, iOS e Web/PWA continuam por produzir** neste candidato;
- **Linux/macOS `BLOCKED_EXTERNAL` pela máquina**; Android pela DLL JNI (`E-1019`);
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1059`)

**Fechado**: o **tarball Linux instalável** — 11 entradas, com `install.sh`/`uninstall.sh`
e `VERSION`, construído **sem bash e sem compilador C**.

**Continua em falta, sem disfarce**:
- ***`BUILD_DESTE_HEAD=NAO` no Linux*** -- o binário é de `9f7379502b1d`; reconstruí-lo
  exige **CGO + gcc/clang**, ausentes;
- ***`INSTALLED=NAO` e `DEVICE_E2E=NAO` no Linux*** -- não há Linux nativo aqui, e o
  `install.sh` **não foi executado**;
- **a sintaxe do `install.sh` não foi validada por um bash** -- só o conteúdo, que vem do
  `package_linux.sh`;
- **macOS, Android, iOS e Web/PWA continuam por produzir** neste candidato;
- **`INSTALLED=NAO` e `DEVICE_E2E=NAO` no Windows** -- instalar exige aprovação do dono;
- **`lib\ollama` é pré-construído**: o instalador Windows **não é** build 100% reproduzível;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.

### O que falta (actualizado na janela `E-1060`)

**Fechado**: o build Web/PWA de produção (exit 0, orçamento PASS) e o **defeito dos três
ícones a 404** — que já estava no HEAD e agora resolve 3 de 3.

**Continua em falta, sem disfarce**:
- ***`SERVICE_WORKER=NAO`*** -- **zero** service workers: sem cache, sem offline, sem
  instalação PWA completa ⇒ **`PWA_INSTALAVEL=PARCIAL`**;
- **o `offline` é `NAO_IMPLEMENTADO`**, não «offline honesto»;
- **não instalei o PWA num browser real** nesta janela;
- **não testei upgrade/rollback do cache** -- não há cache;
- **warning real de chunks >500 kB** (`cpp` 626 kB, `emacs-lisp` 779 kB, `wasm` 622 kB) --
  são gramáticas carregadas sob demanda, e **não foi escondido**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1061`)

**Fechado**: o **service worker** com escopo honesto — `/api/*` network-only, cache do shell
e dos assets, limpeza de versões antigas. **8 testes + contraprova de 3 defeitos.**

**Continua em falta, sem disfarce**:
- ***`INSTALADO_EM_BROWSER=NAO`*** -- o SW **nunca foi instalado num browser real**;
- ***`E2E_EM_BROWSER_REAL=NAO`*** -- os 8 testes usam `caches` **simulado**:
  *«um teste com `caches` simulado prova a lógica; não prova a instalação»*;
- ***`OFFLINE=PARCIAL`*** -- o shell abre offline; o chat, as missões e as aprovações **não**,
  e a página offline di-lo;
- **sem teste de upgrade entre versões reais do SW** -- só a limpeza;
- **não medi** `prefers-reduced-motion`, memória, CPU nem bateria do SW;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1062`)

**Fechado**: o PWA num **Chromium real** — 8/8, incluindo os 3 ícones a 200 e `/api/*`
fora do cache, provado no browser.

**Continua em falta, sem disfarce**:
- ***`PWA_INSTALADO_NA_UI=NAO`*** -- o gesto de instalar é humano e **não foi executado**;
- **Firefox e WebKit/Safari não testados** -- o E2E só correu em **Chromium**;
- **sem teste de upgrade entre versões reais do SW num browser** -- só a limpeza;
- **sem o servidor Ollama real por trás do `/api/*`** -- o preview serve só o estático;
- **não medi** memória, CPU nem bateria do SW;
- ***`OFFLINE=PARCIAL`*** -- o shell abre; o chat, as missões e as aprovações **não**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari**; sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1063`)

**Fechado**: o PWA em **Chromium, Firefox e WebKit** — 8/8 cada. Os dois motores que
faltavam foram instalados (Firefox 157.0, WebKit 27.2).

**Continua em falta, sem disfarce**:
- ***`PWA_INSTALADO_NA_UI=NAO`*** -- o gesto de instalar é humano e **não foi executado**
  em nenhum dos três motores;
- **o WebKit do Playwright não é o Safari do macOS** -- é o motor; não prova Safari numa
  máquina Apple;
- **sem teste de upgrade entre versões reais do SW** num browser;
- **sem medição de memória, CPU, tempo de `install`/`activate` nem tamanho do cache**;
- ***`OFFLINE=PARCIAL`*** -- o shell abre; o chat, as missões e as aprovações **não**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari** reais (extensão); sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1064`)

**Fechado**: os **recursos do SW** (9 entradas, 160,6 KB, **448 ms**) e o **upgrade
v1→v2** num Chromium real — a v1 é apagada e o `/api/*` continua fora do cache.

**Continua em falta, sem disfarce**:
- ***recursos e upgrade só em Chromium*** -- Firefox e WebKit ficaram no `E-1063` com
  os 8 pontos, mas **sem** medição de recursos nem upgrade;
- ***`PWA_INSTALADO_NA_UI=NAO`*** -- o gesto de instalar é humano e **não foi executado**;
- **sem medição de CPU nem bateria** -- só tamanho do cache e latência;
- **`performance.memory` não foi reportado** pelo Chromium headless aqui;
- **o WebKit do Playwright não é o Safari do macOS**;
- ***`OFFLINE=PARCIAL`*** -- o shell abre; o chat, as missões e as aprovações **não**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado
  por falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execução
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **Firefox** e **Safari** reais (extensão); sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1065`)

**Fechado**: recursos e upgrade do SW nos **tres motores** -- 9/9 cada, com os tres a apagar a v1
e a manter `/api/*` fora do cache.

**Continua em falta, sem disfarce**:
- ***o Firefox e' 3,9x mais lento a activar o SW (1646 ms vs 423 ms)*** -- medido, **nao explicado**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** -- o gesto de instalar e' humano e **nao foi executado** em
  nenhum dos tres;
- **sem medicao de CPU nem bateria** -- so' tamanho do cache e latencia;
- **o WebKit do Playwright nao e' o Safari do macOS**;
- ***`OFFLINE=PARCIAL`*** -- o shell abre; o chat, as missoes e as aprovacoes **nao**;
- **a comparacao externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, nao por defeito;
- **decisao do proprietario em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a politica do gate**;
- **N=3 por fixture** na repeticao: contagem, nao probabilidade;
- **as 14 threads do PR continuam abertas** -- nao resolvi nenhuma (proibido);
- **o encaixe da aprovacao continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execucao
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versao 155 **nao carrega** por `--load-extension`;
- **Firefox** e **Safari** reais (extensao); sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **nao** controla o PC;
- **macOS, Android e iOS continuam por produzir** neste candidato.

### O que falta (actualizado na janela `E-1066`)

**Fechado**: o perfil do `install`/`activate` do SW nos tres motores. O custo esta' no `install`
(18-132 ms); o `activate` e' **0 ms**; e os 1646 ms do `E-1065` eram o **arranque do browser**.

**Continua em falta, sem disfarce**:
- ***`PWA_INSTALADO_NA_UI=NAO`*** -- o gesto de instalar e' humano;
- **sem medicao de CPU nem memoria** -- o perfil e' so' de tempo, e de **um registo a frio**;
- **o WebKit do Playwright nao e' o Safari do macOS**;
- ***`OFFLINE=PARCIAL`*** -- o shell abre; o chat, as missoes e as aprovacoes **nao**;
- ***Android, iOS e macOS continuam por produzir*** -- Android e' a proxima viavel;
- **a comparacao externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, nao por defeito;
- **decisao do proprietario em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a politica do gate**;
- **N=3 por fixture** na repeticao: contagem, nao probabilidade;
- **as 14 threads do PR continuam abertas** -- nao resolvi nenhuma (proibido);
- **o encaixe da aprovacao continua por ligar** (`E-1044`);
- **shadow roots fechados**, **cross-origin**: como no `E-1040`; **BFCache** sem execucao
  observada (`E-1038`);
- **o Chrome real do utilizador**: a versao 155 **nao carrega** por `--load-extension`;
- **Firefox** e **Safari** reais (extensao); sem `.crx`, sem SBOM do pacote;
- **`CAPACIDADES` VAZIO**: o Companion **nao** controla o PC.

### O que falta (actualizado na janela `E-1067`)

**Medido e fechado**: o inventario do Android -- o projecto esta' **completo**, e **JDK 17 +
SDK 34 + Gradle 8.13** estao' presentes. O bloqueio esta' **numa DLL do Gradle**, com causa nomeada.

**Continua em falta, sem disfarce**:
- ***`BUILD=NAO` e `APK=NAO` no Android*** -- o Gradle 8.13 nao carrega a `native-platform.dll`
  (`NativeLibraryLocator.java:46`); **nao e' bloqueio externo** -- e' ambiente, com passo concreto;
- **sem E2E em emulador** e **sem prova de que o `App.tsx` compila**;
- ***a marca Android continua `DZ23 Agentic`*** -- `"name": "DZ23 Agentic"`,
  `"package": "com.lmprado.dz23agentic"`, `namespace 'com.lmprado.dz23agentic'`; a migracao para
  **`com.dz23ltd.hades`** que o dono decidiu **nao foi feita**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** no Web;
- **macOS e iOS continuam por produzir**;
- **Windows e Linux `INSTALLED=NAO`/`DEVICE_E2E=NAO`**;
- **a comparacao externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, nao por defeito;
- **decisao do proprietario em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a politica do gate**;
- **N=3 por fixture** na repeticao: contagem, nao probabilidade;
- **as 14 threads do PR continuam abertas** -- nao resolvi nenhuma (proibido);
- **o encaixe da aprovacao continua por ligar** (`E-1044`);
- **o Chrome real do utilizador**: a versao 155 **nao carrega** por `--load-extension`;
- **`CAPACIDADES` VAZIO**: o Companion **nao** controla o PC.

### O que falta (actualizado na janela `E-1068`)

**Fechado**: a **marca Android** — `HADES`, `com.dz23ltd.hades`, com as chaves de storage
preservadas e o plano de migração escrito.

**Continua em falta, sem disfarce**:
- ***a marca no resto do monorepo*** -- o inventário mostra **231 `DZ23 Agentic`** e **49
  `Ollama DZ23`** ainda; a migração do mobile reduziu, mas não fechou;
- ***`BUILD=NAO` e `APK=NAO` no Android*** -- o Gradle 8.13 não carrega a
  `native-platform.dll` (`E-1067`, 4 tentativas, 2 hipóteses novas); **não é bloqueio externo**;
- ***a migração não foi testada numa build*** -- *«uma migração de marca que nunca compilou
  está pronta a compilar — não está verificada»*;
- **sem E2E em emulador**;
- **macOS e iOS continuam por produzir**;
- ***`PWA_INSTALADO_NA_UI=NAO`*** no Web;
- **Windows e Linux `INSTALLED=NAO`/`DEVICE_E2E=NAO`**;
- **a comparação externa continua `NAO_EXECUTADA`**;
- **`55327160c` e `cda756f82` continuam VERMELHOS** -- pelo ritmo de push, não por defeito;
- **decisão do proprietário em aberto**: distinguir «cancelado pelo push» de «cancelado por
  falha» no `pr-gate` **toca a política do gate**;
- **N=3 por fixture** na repetição: contagem, não probabilidade;
- **as 14 threads do PR continuam abertas** -- não resolvi nenhuma (proibido);
- **o encaixe da aprovação continua por ligar** (`E-1044`);
- **o Chrome real do utilizador**: a versão 155 **não carrega** por `--load-extension`;
- **`CAPACIDADES` VAZIO**: o Companion **não** controla o PC.
