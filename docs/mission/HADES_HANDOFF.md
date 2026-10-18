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

# HADES — HANDOFF (como retomar a missão do zero)
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
## JANELA E-1218 -- CONTEXTO MINIMO PARA OUTRA IA RETOMAR

> Substitui a janela `E-1205`. Este bloco e' o handoff: se voce' e' outra IA a retomar, leia
> isto e o indice, nao o historico inteiro.

### ONDE ESTA' O TRABALHO

Checkout: `C:/Users/DZ23/Documents/deepseek-harness/default-workspace/hades-ollama`.
Branch `fix/checker-relogio`, PR #438. Veredito global **`NOT_RELEASE_READY`**.

### O QUE FOI CONSTRUIDO NAS RONDAS 33-38

Duas capacidades canonicas, cada uma em quatro camadas com testes reais em cada camada:

| fase | camada | ficheiro | testes |
|---|---|---|---|
| 28.6 | motor | `internal/agent/toolchain_plan.go` | 14 |
| 28.7 | CLI | `cmd/agent_toolchain.go` | 8 |
| 28.7 | API | `server/agent_toolchain_routes.go` | 6 |
| 28.8 | UI | `components/CapabilitiesPage.tsx` | 6 |
| 28.5 | cofre | `internal/multillm/signing_identity.go` | 16 |
| 28.5 | ficheiro | `internal/multillm/signing_identity_file.go` | 13 |
| 28.5 | API | `server/agent_signing_routes.go` | 4 |
| 28.5 | UI | `components/SigningIdentitiesPage.tsx` | 7 |

### AS DECISOES QUE NAO PODEM SER DESFEITAS

- Tres estados, tres cores. `incompleto` e `desconhecida` em ambar, **nunca** vermelho.
- Trabalho independente aparece sempre que ha' bloqueio.
- **Zero botoes mortos** nas superficies de diagnostico.
- Producao exige aprovacao; a regra vive no servidor **e** no cliente.
- `DisallowUnknownFields` no ficheiro de identidades: campo desconhecido recusa o ficheiro inteiro.
- Registo parcial nunca e' devolvido: volta vazio.
- Estado calculado na leitura, nunca gravado.
- Rotas de diagnostico e de identidades sao **so' leitura**.

### O VERMELHO, EM UMA FRASE

`gitleaks (historico do PR)`: o fixture do proprio teste que prova que `private_key` e' recusado
usava um valor com entropia de chave. Corrigido na arvore; o commit antigo nao pode ser reescrito.
**Precisa de decisao do proprietario, nao de codigo.**

### ORDEM DE RETOMADA

1. Ler `HADES_EVIDENCE_INDEX.md` a partir de `E-1214`.
2. **Nao** repetir o diagnostico do gitleaks.
3. E2E Playwright das duas paginas novas, ou a matriz de assinatura por plataforma (28.4).

## 0.0 REMEDIÇÃO MEDIDA (2026-10-15, agora) — a secção 0 abaixo fica preservada como HISTÓRIA

> **REMEDIÇÃO MAIS RECENTE: `E-959`** (*cadeia `E-820` → `E-867`, rodadas 181–222*).
>
> ⚡ **E o §18.3-25 completou-se nas duas partes que faltavam (`E-867`):** **API concorrente medida** —
> *1 pedido **370 ms**; 4 concorrentes **187/178/168/163 ms**, **4 de 4 `ok`**, parede **730 ms**,
> soma **696 ms*** ⇒ **`wall/serial = 1,05×`**: *o Ollama serve 4 clientes com **5%** de overhead*.
> E a **bateria** ficou `NÃO_MEDIDO` **com causa**: o WMI devolve `71582788` min = **`0x7FFFFFFF`**
> (*outro tecto de 32 bits, como o `4095 MB` do `E-860`*) e `status=2` = **ligado à corrente**.
> *A 1.ª tentativa deu `0,05×` — impossível — e foi o próprio número que denunciou o erro:
> `Start-Job` não herda `$body`.*
>
> 🧑 **E o §18.3-30 (pessoa leiga), inventariado (`E-866`):** o **onboarding é o maior componente de UI do
> repositório** (*`Onboarding.tsx` 50 828 B; `Onboarding.test.tsx` 45 437 B*) e há **onboarding de empresa**
> também. A **escolha do modelo** existe (`ModelPicker.tsx`) com erro em pt-BR que sugere a acção; **3 testes
> de onboarding passam**. ⚠️ **Mas «instalar sem terminal» tem receita e não artefacto**: `app\ollama.iss`
> (*Inno Setup 6*) existe e o Inno Setup **não está instalado** — *decisão **(c)** do dono*. **Item PARCIAL.**
>
> 🔎 **A «via barata» do `G3` foi tentada (`E-865`)**: o `llama-server.exe` **existe** no Ollama instalado
> (`%LOCALAPPDATA%\Programs\Ollama\lib\ollama\` — 4 DLLs, 43 ficheiros) e o Hades procura em **9 caminhos,
> todos dentro do repositório**. Copiei-o ⇒ **o erro mudou** de `binary not found` para
> **`system_info line not found in output`** ⇒ *o binário corre e **não é compatível***, porque o fork
> **modificou** `llama/server`. **`G3` continua `BLOCKED_EXTERNAL`** — agora por **incompatibilidade de
> protocolo**, não por ficheiro ausente. *10,3 MB copiados foram removidos; `git status` = **34**.*
>
> 🔥 **O `G3` MUDOU DE CATEGORIA (`E-864`):** o Hades **compila** (*65,7 MB, 10,1 s*) e **arranca**, mas
> `POST /api/chat` devolve **404** — e o `stderr` do próprio servidor diz por quê: **`llama-server binary
> not found`** (*o motor de inferência nunca foi construído*) e dá o comando
> **`cmake -S llama/server --preset cpu && cmake --build --preset cpu`**. *Exige `cmake` + C++, ausentes
> neste host* ⇒ **`G3` = `BLOCKED_EXTERNAL` com procedimento**, não «por medir».
> *E a GPU fica invisível ao Hades (`total_vram="0 B"`) enquanto o motor não existir.* HEAD
> **`9f7379502`** · branch **`fix/checker-relogio`** · PR **#438** aberto, **NÃO mesclado** ·
> **656** evidências no índice · **34** entradas não commitadas · **0** stashes · veredicto
> **`NOT_RELEASE_READY`**. Os blocos `E-819` e `E-778` abaixo ficam **preservados como HISTÓRIA**.
>
> **O que mudou nesta janela (medido, não narrado):**
> - **A inferência local foi MEDIDA pela primeira vez** (`E-862`/`E-863`): **12,07 s a frio** contra
>   **0,88 s a quente**; **93,7%** do tempo a frio é `load_duration` (*11 295,5 ms*) e o **trabalho útil
>   é idêntico** nos dois regimes (*738,7 vs 783,7 ms*). **TTFT 22,36 ms/token**, decode ~20 tokens/s.
>   Há **9** modelos locais. **VRAM real 8188 MiB** (`nvidia-smi`), o dobro dos **4095** que o WMI reporta.
> - **O §18.3 está em 13 medidos / 7 parciais / 6 externos / 6 por medir**; os que faltam são `30`
>   (pessoa leiga), `25` (API concorrente + bateria) e `22` (responsividade + sessão longa).
> - **`main` continua VERMELHA** (`4b3e21854`) ⇒ **`G0` FALHADO**; o SHA candidato não é ancestral de
>   `main`. **`G12` FALHADO** (*NOTICE de 105 566 B e SBOM fora do Git*). **`G2-race` = `NÃO_SUPORTADO`**
>   (*`go: -race requires cgo`*).
> - **6 alterações ao repositório**, todas **não commitadas**: 5 de código + os documentos de retomada.
>
> ⚠️ **REGRA DE MANUTENÇÃO** (*`E-861`, reconfirmada na ronda 218*): este ficheiro, o
> `HADES_PENDING_WORK.md` e o `HADES_NEXT_ACTION.md` são os **três** documentos de retomada que o
> `hades_mission_check.py` valida. O piso de evidência **sobe a cada ronda** — logo **toda a ronda que
> registe evidência nova tem de actualizar os três NA MESMA RONDA**. E o checker nomeia **um de cada
> vez**: corrigir o nomeado revela o seguinte. *Um documento de retomada desactualiza-se por omissão.*

> **REMEDIÇÃO MAIS RECENTE: `E-819`** (*cadeia `E-816` → `E-819`, rodadas 169–180*). O bloco `E-778` abaixo fica
> **preservado como HISTÓRIA** — a regra de numeração dele continua válida (*evidência nova segue o **maior ID
> cunhado***, e **não** se reutiliza nada abaixo de `E-778`), mas os números dele envelheceram.
>
> | Medida | Valor **medido agora** |
> | --- | --- |
> | Commit | `9f7379502` (**inalterado** desde a rodada 169) |
> | Branch / PR | `fix/checker-relogio` / **PR #438 ABERTO, NÃO mesclado** (*review humano pendente: `LMPrado-DZ23`*) |
> | `main` remota | `4b3e2185422c24d5a05fd46a343953ad6f41d1e5` |
> | Evidências no índice | **612** (`Select-String '^\| E-\d+ ' docs/mission/HADES_EVIDENCE_INDEX.md`) — *a **fonte** é o índice: **não** copie este literal* |
> | Entradas não commitadas | **33** · **Stashes: 0** |
> | Checkers | **7/7** `exit=0` · `GEN=0` · `--check=0` · `go build ./...` **exit 0** |
> | **Camada Web** | **719 PASS / 2 FAIL** em 110 ficheiros (`npx vitest run`, 13,55 s) — *medida pela **primeira vez** no `E-816`* |
> | Veredito global | **`NOT_RELEASE_READY`** |
>
> **O que uma IA que retoma precisa de saber (rondas 169–180):**
> 1. **`E-816`** — a camada **Web** nunca tinha sido contada: *o `go test ./...` não corre vitest*. São **719 PASS / 2 FAIL**, e as 2 falhas estão **num só ficheiro** (`OnboardingRoute.test.tsx`).
> 2. **`E-816`** — **P0-4 FECHADO**: o gate de onboarding é **decisão documentada** (`routes/index.tsx:17-20`), não erro engolido; o teste que faltava existe (`index.onboarding-gate.test.tsx`, **3/3 PASS**).
> 3. **`E-818`/`E-819`** — **DEFEITO DE PRODUTO, registado e NÃO corrigido**: o «Tentar de novo» do onboarding **apaga o erro e não grava nada**. A guarda anti-repetição (`useSettings.ts:150-152`) bloqueia o retry porque o payload é idêntico, e a guarda compara **dados**, não **origem** (`:61-68`). *Não tocar sem contrato novo: este ponto tem **dois** incidentes de carga medidos — **2 387 req/3 s** e **42 651 POST**.* A alternativa segura é **não limpar o erro antes de gravar** (`routes/onboarding.tsx:70-71`).
> 4. **`E-811`/`E-812`** — §18.2-A **em uso real**: model store em **`D:\AI\Ollama`** (49 blobs, 7 manifestos), configurável pela UI **com validação e fallback**.
> 5. **`E-815`** — **este host não tem toolchain C**: `CGO_ENABLED=0` é derivado pelo Go, e isso explica ~14 das falhas de teste (*stub `go-sqlite3`*). *Via recomendada: **mingw-w64 portátil**, que não toca em código.*
> 6. **`E-807`** — defeito **registado e não corrigido**: host de registry com `:` no `manifest.WriteManifest` (*válido em Linux, reservado em Windows*). *Corrigir mudaria o **formato do model store** ⇒ exige **ADR**.*
>
> **REGRA DE ISOLAMENTO (aprendida à força, 3 variáveis):** ao correr testes neste host, isolar `APPDATA`,
> `LOCALAPPDATA` **e** `OLLAMA_CONTEXT_LENGTH` — *esta última é lida no `init` (`envconfig/config.go:230`), logo
> **só** se neutraliza antes do processo arrancar; e `TMP`/`TEMP` devem apontar para dentro do repo, senão o build
> falha por não poder apagar em `%LOCALAPPDATA%\Temp`.*
>
> 7. **`E-822`** — 🔴 **A `main` ESTÁ VERMELHA**, confirmado por API: run **`38499987288`** (`run_number` 1784), evento **`push`**, `head_sha` **`4b3e21854`**, **`conclusion: failure`** (*14/10 02:14Z*). O run **anterior** (`#436`, `2bd4dd0eb`) foi **`success`** ⇒ ***a regressão entrou no PR #437***. ⇒ **`G0` está FALHADO** (*«main/PR reconciliados e aprovados»*), e é **por isso** que a branch `fix/checker-relogio` existe. **Cuidado ao ler o CI:** os 4 runs mais recentes na `main` são do **`dependabot`** com `success` — *não são os gates*. E `test.yaml` na `main` = **0 runs** (*corre só em PR*). **O token desta sessão é de `LMPrado-DZ23`, o revisor pedido no PR #438 ⇒ NÃO aprovar/mergear.**
>
> **PRÓXIMO PASSO (não mudou):** **revisão humana do PR #438** — *não se contorna*. Depois: `HADES_PENDING_WORK.md`
> (*P0-5, P1-1..5, P2-1..5, P3-1..4 ainda por remedir item a item*).

> **EVIDÊNCIA MAIS RECENTE: `E-778` — COLISÃO DE IDs.** Medido em `2026-10-15`: a **sequência viva** do projecto vai
> em **`E-777`** (cunhada em **código e commits**, não no índice), enquanto este índice registava até `E-570` ⇒
> **7** das entradas `E-560..E-570` **colidem** com trabalho committado de **outro** objecto (`E-560` = «409 do job»
> em `server/agent_reembed_job_routes_test.go:98`; `E-561` = «envelope/orçamento» em `internal/agent/historico_bruto.go:183`;
> `E-563` = «catraca do pré-filtro» em `secrets_prefixos_test.go:8`; `E-564` = «F4 da auditoria de segurança»; `E-565` =
> «F5, ordem disco-primeiro» em `internal/agent/context.go:346`; `E-566` = «injecção da compactação»; `E-568` =
> «ligação do P8-1 de memória» em `server/chat_context_wiring_test.go:14`). **REGRA CORRIGIDA:** *a evidência nova
> **não** continua do índice — continua do **maior ID cunhado em qualquer ficheiro***. É por isso que este registo é
> `E-778` e não `E-571`. ⇒ ***Ao retomar: NÃO reutilize nenhum ID abaixo de `E-778`.***

> **A regra do próprio documento (linha ~21) diz: «qualquer número acima envelhece. Antes de citar, remeça.»** Isto é o
> resultado de a cumprir — *e é a **terceira** remeção: a secção 0 já disse `f190ec8ea`/300, depois `fafda19e7`/310, e
> agora cita `cafb00dbe` / `48beffc9d` / **546**.* *Nada foi apagado: o histórico fica, e estes números substituem-no.*

| Medida | Valor **medido agora** |
| --- | --- |
| Commit medido (**local**) | `9f7379502b1d5d5bb7abb8e2bd55224e92796dfd` |
| Remoto da branch (`git ls-remote`) | `9f7379502` => **local = remoto**, **0 commits por empurrar** (a secção 0 diz «5 por empurrar») |
| `main` remota (`git ls-remote`) | `4b3e2185422c24d5a05fd46a343953ad6f41d1e5` (*igual ao que a secção 0 diz — este campo continua correcto*) |
| Distância da secção 0 ao HEAD | **227 commits** (`git rev-list --count 48beffc9d..HEAD`; e `48beffc9d` é **ancestral**: `merge-base --is-ancestor` => exit 0) |
| Evidências no índice | **566** no instante em que `E-566` entrou — ***e este literal NÃO deve ser copiado nem mantido aqui***: a **fonte** é o índice, e lê-se com `Select-String '^\| E-\d+ ' docs/mission/HADES_EVIDENCE_INDEX.md`. *O `E-550` mediu que **duas cópias de um número** são «uma coincidência à espera de divergir» — e eu criei **duas** (aqui e no `HADES_PENDING_WORK.md`) ao escrever este número literal.* |
| Entradas não commitadas | **17** (`git status --porcelain`, medido **antes** desta edição) — *esta edição acrescenta **1** => **18*** |
| Stashes | **0** |
| Veredito global | **`NOT_RELEASE_READY`** |

### CI DO SHA CONGELADO (medido; detalhe em `E-565`)

`9f7379502` tem **5 workflows `success` no MESMO `head_sha`** (evento `pull_request`, PR **#438**, `2026-10-15T17:19:55Z`):
`test` `38776430738` · `dz23-agentic-quality` `38776430666` · `pr-gate` `38776430633` · `dz23-e2e` `38776430658` ·
`class-a-plus-integrity` `38776430631`. *O `pr-gate` é o **agregador** que recusa `cancelled`/`skipped` sem mascarar (`E-551`).*
**LIMITE:** prova a **árvore commitada** — **não** as entradas não commitadas.

### ANTES DE RETOMAR: existe trabalho que NÃO está no CI nem no remoto

As entradas não commitadas incluem: o fix do `POST /api/v1/settings` (`server/desktop_local_routes.go`), a **catraca**
nova (`server/desktop_local_settings_ratchet_test.go`), o fix do **gate local** (`scripts/hades_local_gates.ps1`), o E2E
de backend real (`app/ui/app/e2e/chat-*-real.spec.ts`), as fixtures de sala/companion, e as **evidências `E-561`–`E-566`**
mais a reme­dição deste documento e do `HADES_PENDING_WORK.md`. ⇒ *A revisão independente usa **um** SHA (§18.5.10), e esse
SHA é `9f7379502`, que **não** contém isto.*

## 0. ESTADO MEDIDO — `fix/checker-relogio` `cafb00dbe` (esta é a leitura canônica; não herde número de texto antigo)

| Medida | Valor medido |
| --- | --- |
| Commit medido (**local**, à frente do remoto) | `cafb00dbe` |
| Remoto da branch | `48beffc9d` (*5 commits locais por empurrar*) |
| `main` remota | `4b3e21854` |
| Evidências no índice | **546** |
| Estágios com evidência | **38 de 38** |
| Bloqueios declarados | **17** (estado: 17 itens / **17 ids únicos**) |
| Matriz do gate final | `BLOCKED_BY_EXTERNAL_DEPENDENCY: 5 · DONE: 19` |
| Conflitos com a `main` | **nenhum** (`merge-tree` exit 0 · 111 commits à frente, 0 atrás) |
| Veredito global | **`NOT_RELEASE_READY`** |

**Medido em `2026-10-14 16:34 UTC`.** *Os dois campos «local» e «remoto» existem porque, durante a FASE 2, há commits
locais **segurados de propósito** até a corrida do CI anterior ser lida — *regra do `E-541`: um push por corrida*.
Citar só um dos dois daria uma imagem falsa do que está onde.*

**Regra desta seção (E-301, aplicada em E-310, e outra vez em E-544):** qualquer número acima **envelhece**. Esta seção
já ficou desatualizada **duas** vezes (dizia `f190ec8ea` / 300, e depois `fafda19e7` / 310) e foi remedida. ⇒ *E o `E-544`
mediu o custo disso: o documento esteve **parado ~16 h** sem mencionar **um único** commit ou evidência recente.*
**Antes de citar, remeça:**

```powershell
git rev-parse --short HEAD
(Select-String -Path docs/mission/HADES_EVIDENCE_INDEX.md -Pattern '^\| E-\d+ ').Count
python scripts/hades_mission_check.py                      # ESCRITURAÇÃO CONSISTENTE (exit 0)
python scripts/hades_ci_reproducibility_check.py --check    # nenhum gate flutuante (E-309)
Select-String -Path docs/mission/HADES_FINAL_RELEASE_GATE.md -Pattern 'Resumo: '
```

**Decisão preservada:** `docs/mission/HADES_CURRENT_STATE.json` **não existe por decisão explícita** — dois arquivos
de estado divergiriam. O estado canônico é `HADES_MASTER_STATE.json`. **Não criar** o outro.

> ⚠️ **PERIGO, MEDIDO — os DOIS blocos abaixo começam por `git fetch origin main; git reset --hard origin/main`**
> (`:40` e `:61`, medidos em 2026-10-15 **antes** desta nota). *Essa linha **destrói** todo o trabalho não commitado.*
> Na árvore medida há **17 entradas não commitadas**, incluindo os **dois WIP do proprietário**
> (`docs/parity/HADES_T2_MATRIX.md`, `docs/parity/HADES_T2_REPETICAO.md`) que a missão manda **PROTEGER**.
> ⇒ **Não cole este bloco sem guardar o trabalho primeiro.** Caminho **não destrutivo**: `git status --porcelain` para ver
> o que existe, `git stash push -u -m "wip-antes-da-retomada"` (ou copiar os ficheiros para fora do repositório) e **só
> depois** decidir. *Esta nota existe porque o texto abaixo se apresenta como «nenhum passo presumido» — e um passo
> destrutivo apresentado sem guarda é exactamente a definição de passo presumido.*

**Retomada exata (comandos reais, nenhum passo presumido):**

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
# BUILD CRUZADO (E-317): sufixo _GOOS acidental passa no Windows e quebra no Linux.
$env:GOOS = 'linux'; $env:CGO_ENABLED = '0'; go build ./internal/agent/; $env:GOOS = ''
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

## 1. Ordem de leitura obrigatória

1. `docs/mission/HADES_MASTER_GOAL.md` — objetivo, limites de autonomia, estados.
2. `docs/mission/HADES_MASTER_STATE.json` — **estado canônico de máquina**.
3. `docs/mission/HADES_STAGE_REPORT.md` — o que está feito/pendente por estágio.
4. `docs/mission/HADES_BLOCKERS.md` — o que depende de terceiros.
5. `docs/mission/HADES_EVIDENCE_INDEX.md` — o que é prova e o que não é.
6. `docs/mission/HADES_PENDING_WORK.md` — próximos passos priorizados.
7. `docs/mission/HADES_NEXT_GOAL_READINESS.md` — critérios de aceite.
8. `docs/mission/HADES_ENGINEERING_GATE.md` — gate intermediário do
   estágio 22 (relatório de 20 itens + suíte executada).
9. `audit/HADES_MASTER_AUDIT.md`, `audit/HADES_CAPABILITY_MATRIX.md`,
   `audit/HADES_RELEASE_BLOCKERS.md`.
9. Documentos históricos do repositório (pistas, não prova):
   `audit/CLAUDE_CODEX_RESUME_PROMPT_20260927.md`,
   `audit/OLLAMA_FULL_HANDOFF_20260927.md`,
   `audit/OLLAMA_FULL_MISSION_STATE.md`.

**Fonte única de verdade:** `HADES_MASTER_STATE.json`. **Não** existe
`HADES_CURRENT_STATE.json` por decisão explícita — dois arquivos de estado
divergiriam. Qualquer estado novo é escrito no JSON canônico e refletido nos
relatórios.

## 2. Regras invioláveis do repositório

- `main` é **integração**: nunca commitar direto. Branch curta a partir de
  `main`, PR contra `main`, checks obrigatórios verdes.
- **Nunca** `git push --force`; nunca reescrever `main`.
- Nunca desativar teste/lint/gate; corrigir causa raiz.
- Nunca imprimir segredo (chave, token, senha) em log, arquivo ou resposta.
- `_tmp/` é scratch ignorado pelo Git: **nunca** entra em commit.
- Merge/publicação/custo externo exigem autorização humana explícita.

## 3. Ambiente (bloco obrigatório antes de qualquer build/teste)

```powershell
$repo = 'C:\Users\DZ23\Documents\deepseek-harness\default-workspace\hades-ollama'
$env:GOROOT = "$repo\_tmp\goroot\go"
$env:GOPATH = "$repo\_tmp\gopath"
$env:GOMODCACHE = "$repo\_tmp\gomodcache"
$env:GOCACHE = "$repo\_tmp\gocache"
$env:GOTMPDIR = "$repo\_tmp\gotmp"
$env:GOFLAGS = '-mod=mod'
$env:GOTOOLCHAIN = 'local'
$env:TMP = "$repo\_tmp\e2e-tmp"
$env:TEMP = "$repo\_tmp\e2e-tmp"
$env:PATH = "$env:GOROOT\bin;$env:PATH"
```

`TMP`/`TEMP` redirecionados são **obrigatórios** para Node/Vite/esbuild e para
`server.TestMain` (que exige `APPDATA`/`LOCALAPPDATA` válidos).

## 4. Comandos de verificação (gates reais)

```powershell
# Go
& "$env:GOROOT\bin\go.exe" vet ./internal/agent/
& "$env:GOROOT\bin\go.exe" build ./...
& "$env:GOROOT\bin\gofmt.exe" -l internal server app        # saída vazia = ok
& "$env:GOROOT\bin\go.exe" test ./internal/agent/
& "$env:GOROOT\bin\go.exe" test ./server/

# Web (workdir = $repo\app\ui\app)
npm run lint
npx vitest run
npm run build
npx playwright test --project=chromium
npx playwright test --project=first-run
npx prettier --check playwright.config.ts e2e/firstRun.spec.ts e2e/first-run-backend.mjs

# Classificador de superfícies de CI
python scripts/classify-ci-surfaces.py    # e classify_ci_surfaces_test.py
```

**Falhas ambientais conhecidas (não são regressões):** ver B-09. Nunca
"consertar" desativando o teste.

**Pegadinhas de ferramenta já mapeadas:** `git diff/fetch/push` podem retornar
exit 1 exibindo sucesso no stderr — não confie no exit code; `gh api` sem
`--jq` (single-quote no path) e `--allow-escape-sequences` para logs; `rg` não
existe no PATH (usar a ferramenta `grep` com `path` estreito); não executar
arquivos `.ps1` (política de assinatura) — inline o corpo; `golangci-lint`
exige `GOLANGCI_LINT_CACHE` dentro do repositório e `GOPROXY` **não** definido
para `go run modulo@versao`.

## 5. Estado de branches/PRs no momento deste handoff

- `main` = `889750abe4006985b3bf9037bca2323d1e40cef0`, CI verde.
- `docs/checkpoints-missao-v21` = este checkpoint (base `main`).
- PRs de missão abertos: #56–#63, #65–#71 (ver lista completa no JSON).
- Mergeados: **#54** e **#55**, somente.
- Dependabot abertos: #21–#34 e #64 (15 PRs).
- Precedente de PR docs-only: #57 e **#72** (passaram sem esperar gates profundos;
  #72 com 15/15 checks sem falha — E-036).
- **PRs abertos criados por esta missão:** #72 `docs/checkpoints-missao-v21`
  (`d64a47ef3`, checkpoints canônicos, docs-only) e #73 `feat/e2e-studio-reorder`
  (`6465e88f7`, E2E do reorder do Studio, **empilhado em #71**). Ambos verdes
  (E-034, E-036) e aguardando **autorização de merge**.
- **#73 é empilhado:** base = `feat/e2e-primeira-execucao` (#71); depois do merge
  de #71 ele deve ser retargetado para `main`.

## 6. Primeiro passo ao retomar

1. Reconferir `main` ao vivo (`gh api .../commits/main`) — **nunca** assumir o
   SHA deste documento.
2. Reexecutar o classificador de superfícies para a mudança pretendida.
3. Escolher o próximo item de `HADES_PENDING_WORK.md` (ordem P0 → P1 → P2 → P3).
4. Trabalhar em branch curta, com evidência anexada ao índice antes de fechar.

## 7. Progresso e recuperação

Ao final de cada rodada: reexecutar os gates do que mudou, anexar evidência
(comando + ambiente + resultado), atualizar `HADES_MASTER_STATE.json` e o
relatório por estágio, e só então declarar estado. Após interrupção, retomar do
último estado **verificado** — não do que foi tentado.

## 8. Estado do CI e dos complementos

2026-10-14 — CI e complementos, MEDIDO

Run `5bba1132e` (PR #438): 23 checks, ZERO vermelhos.
  race (ubuntu-latest)  SUCCESS  653 s
  race (macos-latest)   SUCCESS  867 s
  ⇒ ambos ACIMA do limite antigo de 600 s do `go test`: a causa dos `race` vermelhos era
    ORÇAMENTO DE TEMPO (não corrida de dados - o log não tinha `WARNING: DATA RACE`), e a
    correcção foi um só limite (`-timeout 20m`) com o motivo escrito no workflow.
  test ubuntu/macos/windows ✅ · PR gate ✅ · Web and mobile quality ✅ · Go agentic gates ✅ ·
  CGO-free/Windows build ✅ · RLS/Redis/OTLP ✅ · Web E2E ✅ · SBOM ✅

---

## ACTUALIZACAO V3.0 (RONDA 71) - ESTADO E AS INSTRUCOES DESTRUTIVAS

> **Requisito**: o `mission_check` exige que os documentos de retomada citem evidencia **recente**
> (>= `E-960`). Este documento parava no `E-959`. **E' o ESTAGIO 1.**

### ⚠️ As instrucoes destrutivas deste documento, localizadas AGORA (E-986 revalidado)
```
:163  o aviso «PERIGO, MEDIDO - os DOIS blocos abaixo ...»  (*a linha que proibe*)
:175  BLOCO 1 destrutivo  (*12 linhas apos o aviso*)
:196  BLOCO 2 destrutivo  (*33 linhas apos o aviso*)
```
*Medicao desta ronda: **as 3 linhas continuam intactas** e o aviso **precede** os dois blocos.*
=> *O requisito do ESTAGIO 1 («marcar inequivocamente como historicas e proibidas com WIP») esta'
**satisfeito**, e foi reconfirmado - **nao** declarado uma vez e esquecido.*

**E este documento NAO reproduz os comandos, de proposito.** *A primeira versao desta seccao
copiou-os para os citar* => *e o `assert` do gerador apanhou **quatro** ocorrencias em vez de
**duas*** => *porque **documentar uma instrucao destrutiva e' reintroduzi-la*** => *um `grep` por
`reset --hard` neste ficheiro passaria a acusar duas ordens **a mais**, e uma delas seria a que
existe para as proibir.*
=> *Por isso aqui ha' **numeros de linha**, e nao o texto dos comandos.* **Consulte-os em `:175` e
`:196` se precisar; nao os execute com WIP presente.**

**Regra viva:** com WIP de 14 entradas, **qualquer** um desses dois blocos destruiria trabalho
nao commitado. *Eles sao **historicos**: descrevem o que foi feito numa janela em que a arvore
estava limpa.*

### Evidencias novas (E-960 .. E-1000), por tema
- **Adendo do ecossistema**: `E-960`..`E-967` (as 5 fontes, a matriz de encaixe, a fatia vertical do
  `POST /plugin`) e `E-972` (o formato aberto `hades-plugin.format/v1`).
- **Findings do PR #438**: `E-961`, `E-962` (os 5 P1 do checker do relogio, os 12 findings),
  `E-963`..`E-965` (as 5 fontes conferidas e a correccao factual do `E-959`).
- **ESTAGIO 22**: `E-987` (snapshot real), `E-988` (manifesto), `E-989` (catraca), `E-993`
  (4 volateis), `E-994` (`source_sha`), `E-996` (o `host` e a inversao da regra), `E-997` (o
  inventario media o disco).
- **FASE-5**: `E-990` (909 canonicas / 441 legadas) e `E-991` (o gerador com 8 excecoes).
- **Instrumentos**: `E-992` (ASCII nos `.ps1`), `E-995` (`E-559` restaurado do Git), `E-998`
  (quinta instancia), `E-999` (`\n` literal), `E-1000` (evidencia sem commit).

### Estado medido nesta ronda
```
HEAD 31be202d6 · branch fix/checker-relogio · main 4b3e21854 · WIP 14 · stashes 0
indice 793 entradas (E-001..E-1000) · bateria 16 OK / 0 FAIL
CI: 6/6 verdes no d1f4e7915 (incluindo `test`, que era o que mais reprovava)
PR #438: OPEN · mergeState=BLOCKED · reviewDecision VAZIO (FALTA REVISAO HUMANA)
```

### Retomada segura (o que EXISTE neste repositorio)
```bash
cd hades-ollama && git fetch origin && git status --porcelain=v1
git rev-parse HEAD origin/fix/checker-relogio origin/main
python scripts/hades_release_gate.py --check
python scripts/hades_mission_check.py
python -X utf8 scripts/hades_manifest.py --check
python -X utf8 scripts/hades_brand_inventory.py --check
python -X utf8 scripts/hades_ascii_ps1_check.py
```
*O `_tmp/bateria_local.ps1` citado na versao anterior deste documento **nao existe** no
repositorio; a bateria canonica e' `scripts/hades_local_gates.ps1` mais os `--check` acima.*

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

### Instrucoes de retomada (actualizadas na janela `E-1009`..`E-1021`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1021`; PR #438 **aberto** e **nao
tocado** (decisao do dono); `reviewDecision` **vazio** => sem revisao humana independente.

**Onde estao as coisas novas**:
- extensao: `apps/hades-browser-bridge/` (11 ficheiros, 24 testes: `node --test apps/hades-browser-bridge/test/protocol.test.js apps/hades-browser-bridge/test/carregavel.test.js`);
- ferramentas baixadas a mao em `_tmp/`: `jdk17` (JDK 17.0.20.1), `android-sdk` (394,2 MB), `gradle-8.13-bin.zip`;
- `_tmp/` **nao** entra no Git (`E-1020`);
- material de assinatura Apple: em `C:\Users\DZ23\Downloads`, **fora** do repo, **0** rastreados;
  o `.gitignore` agora cobre 20 padroes (verificado 19/19 com `git check-ignore`).

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash, `merge`,
tag, release, publicacao externa, ou **resolver threads** do PR #438. Nada disso esta' autorizado.

**Como verificar tudo antes de commitar** (bateria de 20 gates, exit 0 obrigatorio):
`hades_release_gate.py` (write) -> `--check`; `hades_manifest.py`; `hades_brand_inventory.py`;
`hades_ascii_ps1_check.py`; `hades_mission_check.py`; `hades_state_chronology_test.py`;
`hades_index_shape_check.py --check` **e** `_test.py`; `hades_handoff_window_check.py --check`
**e** `_test.py`; `hades_orphan_instruments_check.py --check`; `hades_audit_instruments_test.py`;
`hades_stage_map.py --check`; `hades_baseline_report.py --check`;
`hades_ci_reproducibility_check.py --check`; `hades_tenant_scope_check.py --check`;
`hades_non_inference_scan.py`; `hades_encoding_check.py`; `hades_fix_mojibake.py --check`;
`git diff --check`.

**Doutrina em uma linha**: *medir antes de afirmar; um gate que me apanha esta' a funcionar; e
evidencia sem commit e' uma afirmacao.*

### Instrucoes de retomada (janela `E-1022`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1022`; PR #438 **aberto** e **nao
tocado** (decisao do dono); `reviewDecision` **vazio** => sem revisao humana independente.

**Artefactos novos desta janela**:
- pacote: `dist/browser-bridge/hades-browser-bridge-0.1.0-unsigned.zip` + `SHA256SUMS.txt` +
  `LEIA-ME.txt` -- **fora do Git** (`dist/` esta no `.gitignore`; `git ls-files dist` e vazio);
- script versionado: `scripts/hades_browser_bridge_package.py` (constroi e verifica; `--check`
  so verifica);
- discriminador: `_tmp/discrimina_pacote_v2.py` (3 defeitos; exige que a **regra propria**
  dispare, nao basta o exit code).

**A armadilha a nao repetir**: o `zipfile` do Python **normaliza o separador ao LER E ao
ESCREVER**. Logo: (1) uma regra escrita sobre o `namelist()` **nunca dispara**; (2) `writestr`
com o nome trocado **nao cria** um zip com separador invertido. Para testar o separador e preciso
mexer nos **bytes crus**, nos **dois** cabecalhos: local file header (PK 03 04, nome em offset+30)
e central directory (PK 01 02, nome em offset+46), trocando 0x2F por 0x5C.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash, `merge`,
tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Bateria de 20 gates antes de commitar** (exit 0 obrigatorio): `hades_release_gate.py` (write)
-> `--check`; `hades_manifest.py`; `hades_brand_inventory.py`; `hades_ascii_ps1_check.py`;
`hades_mission_check.py`; `hades_state_chronology_test.py`; `hades_index_shape_check.py --check`
**e** `_test.py`; `hades_handoff_window_check.py --check` **e** `_test.py`;
`hades_orphan_instruments_check.py --check`; `hades_audit_instruments_test.py`;
`hades_stage_map.py --check`; `hades_baseline_report.py --check`;
`hades_ci_reproducibility_check.py --check`; `hades_tenant_scope_check.py --check`;
`hades_non_inference_scan.py`; `hades_encoding_check.py`; `hades_fix_mojibake.py --check`;
`git diff --check`.

**Doutrina em uma linha**: medir antes de afirmar; um gate que me apanha esta a funcionar; e um
teste que usa a mesma biblioteca que causa o defeito nunca injecta o defeito.

### Instrucoes de retomada (janela `E-1023`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1023`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**A extensao tem 3 ficheiros de teste (47 assercoes)**:
  node --test apps/hades-browser-bridge/test/protocol.test.js
                apps/hades-browser-bridge/test/carregavel.test.js
                apps/hades-browser-bridge/test/canal_local.test.js

**Artefactos**:
- pacote: `dist/browser-bridge/hades-browser-bridge-0.1.0-unsigned.zip` + `SHA256SUMS.txt`
  + `LEIA-ME.txt` -- **fora do Git**;
- script versionado: `scripts/hades_browser_bridge_package.py` (passo 14 da bateria);
- contrato do canal: `apps/hades-browser-bridge/src/canal_local.js`.

**A armadilha a nao repetir -- QUARTA vez**: *um detector de texto nao distingue USAR de
FALAR SOBRE*. Casos medidos: `E-1008` (um literal de quebra de linha descrito);
`E-1016`/`E-1017` (o grep que encontrou a propria declaracao de ausencia); `E-1021`
(a palavra tokens num comentario); e `E-1023` -- **`Math.random` dentro de uma STRING DE
ERRO**, que nem comentario e. A cura e procurar o **uso** (a chamada, com parenteses),
nunca a mencao.

**Outra armadilha**: `new URL("http://[::1]:9").hostname` devolve `[::1]` **COM brackets**.
Uma lista de loopback sem essa forma **recusa o IPv6** -- medido e curado no `E-1023`.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Bateria de 21 gates** (exit 0 obrigatorio): os 20 anteriores mais o **pacote da extensao**.
O passo 14 chama `hades_browser_bridge_package.py`, que constroi e verifica por si -- e seguro
sobre um `dist/` vazio.

**Doutrina em uma linha**: medir antes de afirmar; um gate que me apanha esta a funcionar; um
detector que procura texto confunde usar com falar sobre; e um teste que usa a mesma
biblioteca que causa o defeito nunca injecta o defeito.

### Instrucoes de retomada (janela `E-1024`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1024`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**A extensao tem 3 ficheiros de teste (56 assercoes)**:
  node --test apps/hades-browser-bridge/test/protocol.test.js
                apps/hades-browser-bridge/test/carregavel.test.js
                apps/hades-browser-bridge/test/canal_local.test.js

**Artefactos**:
- pacote: `dist/browser-bridge/hades-browser-bridge-0.1.0-unsigned.zip` +
  `SHA256SUMS.txt` + `LEIA-ME.txt` -- fora do Git;
- script versionado: `scripts/hades_browser_bridge_package.py` (passo 14 da bateria);
- contrato do canal: `apps/hades-browser-bridge/src/canal_local.js` (HKDF + anti-replay);
- native host (template):
  `apps/hades-browser-bridge/native-host/com.dz23ltd.hades.bridge.json`.

**DUAS CATRACAS COMPLEMENTARES -- nao confundir**:
- `hades_index_shape_check.py` so varre o **indice**; apanhou um ID citado sem entrada no
  `E-1020`;
- `hades_mission_check.py` varre **todo o codigo**; apanhou o mesmo erro no `E-1024`, por
  citacao em `canal_local.js` e `canal_local.test.js` -- vector que o `index_shape`
  **nao ve**. E ele tem a regra certa: **um ID novo continua do MAIOR cunhado, nao da
  cabeca do indice** (numerar onde o indice acaba reutiliza numeros com dono -- a colisao
  do `E-778`).

**A armadilha do detector de texto -- QUARTA vez**: `E-1008`, `E-1016`/`E-1017`, `E-1021` e
`E-1023` (`Math.random` dentro de uma **string de erro**). Procurar o **uso** (a chamada),
nunca a mencao.

**Outra armadilha**: `new URL("http://[::1]:9").hostname` devolve `[::1]` **COM brackets**.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Bateria de 21 gates** (exit 0 obrigatorio), incluindo o **pacote da extensao** no passo 14.

**Doutrina em uma linha**: medir antes de afirmar; um gate que me apanha esta a funcionar; um
detector que procura texto confunde usar com falar sobre; e a cabeca de uma numeracao e o
MAIOR cunhado, nao o ultimo registado.

### Instrucoes de retomada (janela `E-1026`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1026`; PR #438 **aberto** e **nao
tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**A extensao tem 3 ficheiros de teste (67 assercoes)**:
  node --test apps/hades-browser-bridge/test/protocol.test.js \
                apps/hades-browser-bridge/test/carregavel.test.js \
                apps/hades-browser-bridge/test/canal_local.test.js

**Artefactos**:
- pacote: `dist/browser-bridge/hades-browser-bridge-0.1.0-unsigned.zip` + `SHA256SUMS.txt` +
  `LEIA-ME.txt` -- fora do Git;
- script versionado: `scripts/hades_browser_bridge_package.py` (passo 14 da bateria);
- contrato do canal: `apps/hades-browser-bridge/src/canal_local.js` -- HKDF, codigo de uso unico,
  handshake de prova de posse, taxa e anti-replay;
- native host (template):
  `apps/hades-browser-bridge/native-host/com.dz23ltd.hades.bridge.json`.

**TRES ARMADILHAS DA JANELA, todas da classe «contrato inventado de memoria»**:
- `verificar(chave, nonce, prova)` -- **3 argumentos numa funcao de 2**. O 3o era ignorado, a
  guarda devolvia `false` SEMPRE, e um teste dava verde pelo motivo errado;
- `registro.revogadaEm` -- **campo que nunca existiu**. `undefined` e' falsy, logo a guarda de
  revogacao estava MORTA, e so' o `DataError: Zero-length key` a denunciou;
- `assinar(chave, "string")` -- `canonico` le' campos de ENVELOPE: o MAC era **o mesmo para
  todos os desafios**. A cura foi dar ao desafio a **sua propria** canonizacao.
  Antes de chamar funcao existente: **ler a assinatura**. Um `undefined` numa guarda e' uma
  guarda que nao existe.

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo** -- e foi ele que apanhou o `E-1024` citado em
codigo antes de existir. **Um ID novo continua do MAIOR cunhado, nao da cabeca do indice.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash, `merge`,
tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Bateria de 21 gates** (exit 0 obrigatorio), incluindo o **pacote da extensao** no passo 14.

**Doutrina em uma linha**: medir antes de afirmar; um gate que me apanha esta a funcionar; um
argumento a mais em JavaScript nao e' erro, e' silencio; e um teste que passa pelo motivo errado
nao mede nada.

### Instrucoes de retomada (janela `E-1028`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1028`; PR #438 **aberto** e **nao
tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**A extensao tem 4 ficheiros de teste (84 assercoes)**:
  node --test apps/hades-browser-bridge/test/protocol.test.js \
                apps/hades-browser-bridge/test/carregavel.test.js \
                apps/hades-browser-bridge/test/canal_local.test.js \
                apps/hades-browser-bridge/test/transporte.test.js

**Artefactos**:
- pacote: `dist/browser-bridge/hades-browser-bridge-0.1.0-unsigned.zip` + `SHA256SUMS.txt` +
  `LEIA-ME.txt` -- fora do Git; **11 entradas**, incluindo `src/transporte.js`;
- script versionado: `scripts/hades_browser_bridge_package.py` (passo 14 da bateria);
- contrato do canal: `apps/hades-browser-bridge/src/canal_local.js` -- HKDF, codigo de uso unico,
  handshake de prova de posse, taxa e anti-replay;
- transporte: `apps/hades-browser-bridge/src/transporte.js` -- enquadramento + canal em memoria;
- native host (template):
  `apps/hades-browser-bridge/native-host/com.dz23ltd.hades.bridge.json`.

**NAO CONTES BYTES A MAO -- MEDE-OS** (`E-1028`): na janela do transporte errei **tres**
contagens/nomes de cabeca (*12 vs 7 bytes*, *306 vs 308*, *`replay` vs
`replay_ou_fora_de_ordem`*). E o mesmo vale para nomes de campos (`E-1026`: `registro.revogadaEm`)
e para premissas do que e' detectavel (*um enquadramento de comprimento **nao tem numero magico**:
lixo no inicio nao e' detectavel, e exigir isso seria exigir adivinhacao*).

**UM TESTE VERMELHO PODE MEDIR O TESTE, NAO O CODIGO**: das 3 falhas do transporte, **3 eram
minhas** e **0 do codigo**. Curar o codigo para satisfazer uma premissa errada teria produzido um
transporte pior e um teste verde.

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo** -- e foi ele que apanhou o `E-1024` citado em
codigo antes de existir. **Um ID novo continua do MAIOR cunhado, nao da cabeca do indice.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash, `merge`,
tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Bateria de 21 gates** (exit 0 obrigatorio), incluindo o **pacote da extensao** no passo 14.

**Doutrina em uma linha**: medir antes de afirmar; um registro de comprimento nao distingue lixo
de cabecalho; e um teste que passa pelo motivo errado nao mede nada.

### Instrucoes de retomada (janela `E-1030`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1030`; PR #438 **aberto** e **nao
tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**A extensao tem 5 ficheiros de teste (93 assercoes)**:
  node --test apps/hades-browser-bridge/test/protocol.test.js \
                apps/hades-browser-bridge/test/carregavel.test.js \
                apps/hades-browser-bridge/test/canal_local.test.js \
                apps/hades-browser-bridge/test/transporte.test.js \
                apps/hades-browser-bridge/test/companion.test.js

**O Companion arranca-se a mao** (fala native messaging pelo `stdin`/`stdout`):
  node apps/hades-browser-bridge/native-host/companion.mjs

**Artefactos**:
- pacote da extensao: `dist/browser-bridge/hades-browser-bridge-0.1.0-unsigned.zip` +
  `SHA256SUMS.txt` + `LEIA-ME.txt` -- fora do Git; **11 entradas** (inclui `src/transporte.js` e
  **NAO** inclui o Companion: sao produtos separados por fronteira de permissao);
- script versionado: `scripts/hades_browser_bridge_package.py` (passo 14 da bateria);
- contrato do canal: `apps/hades-browser-bridge/src/canal_local.js`;
- transporte: `apps/hades-browser-bridge/src/transporte.js`;
- processo: `apps/hades-browser-bridge/native-host/companion.mjs`;
- native host (template): `.../native-host/com.dz23ltd.hades.bridge.json`.

**TRES SONDAS VERDES NAO PROVARAM NADA** (`E-1030`): contaram bytes, imprimiram a mensagem e
importaram o modulo -- todas mediram o que **chegou**, nenhuma mediu a **relacao** entre pedido e
resposta. A 4a, com o helper exacto do teste, imprimiu `MSG id=undefined` e o defeito ficou visivel.
Uma sonda tem de medir a **hipotese**, nao a vizinhanca dela.

**Regra de ouro do Companion**: o `stdout` e' o FIO. Todo diagnostico vai para o `stderr`; um
`console.log` perdido dentro do protocolo corrompe o fluxo. E **todo** `id` tem de ser ecoado, mesmo
nas respostas de erro.

**NAO CONTES BYTES A MAO -- MEDE-OS** (`E-1028`). E o mesmo vale para nomes de campos (`E-1026`:
`registro.revogadaEm`) e para nomes de motivos (`replay_ou_fora_de_ordem`).

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash, `merge`,
tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Bateria de 21 gates** (exit 0 obrigatorio), incluindo o **pacote da extensao** no passo 14.

**Doutrina em uma linha**: medir antes de afirmar; o `stdout` e' o fio; uma resposta sem `id` e' um
pedido que nunca termina; e tres sondas verdes que medem a mesma coisa continuam a ser uma sonda.

### Instrucoes de retomada (janela `E-1031`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1031`; PR #438 **aberto** e **nao
tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**O native host agora INSTALA-SE** (e desinstala-se):
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_native_host_install.ps1 -Diagnostico
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_native_host_install.ps1
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_native_host_install.ps1 -Desinstalar

Escreve SO' no `HKCU` (Chrome + Edge), **nunca** pede elevacao, e grava `INSTALACAO.json`
listando o que criou -- o `-Desinstalar` remove exactamente isso.

**O ID da extensao e' DETERMINISTICO** (nao precisa de navegador para o saber):
  python -X utf8 scripts\hades_native_host.py --check
  # id=lbndhojeaplfofjmhdmlbocabojakddb  (derivado de native-host/chave-publica.b64)

Se um dia a `chave-publica.b64` mudar, o ID muda e o `allowed_origins` **tem** de ser regerado --
o `--check` acusa (`MANIFESTO DIVERGE`).

**NUNCA versione o manifesto REAL**: ele tem o caminho absoluto da maquina. O que esta' no Git e'
a chave publica, o gerador e o instalador. O template do repo tem placeholders de proposito.

**A chave PRIVADA de distribuicao esta' fora do Git** (`_tmp/ext-chave-privada.pem`). Sem ela nao
se empacota um `.crx` com o mesmo ID. **Nunca** a imprima nem a mova para o repo.

**O que ainda NAO foi provado**: o host a falar com um navegador a serio. Instalei-o e
desinstalei-o; **nao abri o Chrome**. O E2E real e' o proximo passo.

**Bateria**: agora com **23 passos** -- os passos 15 (`native-host-id`) e 16
(`native-host-testes`) entraram, e a catraca de orfaos fecha com `ORFAO 0`.

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash, `merge`,
tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um ID que se deriva nao se adivinha; um manifesto com caminho absoluto
nao se versiona; instalar nao e' publicar; e uma catraca so' vale depois de ter ficado vermelha.

### Instrucoes de retomada (janela `E-1032`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1032`; PR #438 **aberto** e **nao
tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 24 passos.** O E2E real entra por **opt-in** (abre uma janela do Edge):
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_local_gates.ps1 -ComE2E

**O E2E real, sozinho**:
  node scripts/hades_browser_bridge_e2e.mjs
  # instala o host -> Edge carrega a extensao -> id do navegador == id derivado
  # -> popup -> conceder -> recarregar -> sendNativeMessage -> resposta -> desinstalar

**QUATRO ARMADILHAS MEDIDAS, PARA NAO AS REPETIR** (`E-1032`):
1. `chrome.runtime.sendNativeMessage` **nao existe** enquanto `nativeMessaging` nao for
   concedida -- e nao da' erro de permissao, **nao e' uma funcao**;
2. `chrome.permissions.request` **trava para sempre** dentro do handler do clique -- tem
   prazo de 8s no popup, senao o botao fica morto sem erro;
3. o **gesto do utilizador nao sobrevive** ao `chrome.runtime.sendMessage` -- o pedido tem
   de viver no `popup.js`;
4. o **Windows nao executa `.mjs`** -- o `path` do manifesto aponta para um lancador `.cmd`,
   e o `--check` recusa `path` terminado em `.mjs`.

**E O `chrome.runtime` NAO SE RECONSTROI**: com a permissao ja' concedida,
`typeof chrome.runtime.sendNativeMessage` continua `undefined` ate' a extensao recarregar.

**O Chrome 155 deste host nao carrega por `--load-extension`** (medido: 0 service workers).
O Edge carrega. Nao suponha qual serve -- meca.

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash, `merge`,
tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um protocolo que nunca atravessou um navegador a serio nao esta'
provado; um `path` que o SO nao executa nao e' um host; uma promessa que trava e' pior do que
uma excecao; e o gesto do utilizador nao sobrevive a um `sendMessage`.

### Instrucoes de retomada (janela `E-1033`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1033`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 25 passos.** Dois E2E entram por **opt-in** (`-ComE2E`), porque abrem o Edge:
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_local_gates.ps1 -ComE2E

**Os dois E2E, sozinhos**:
  node scripts/hades_browser_bridge_e2e.mjs       # canal: extensao <-> Companion
  node scripts/hades_browser_bridge_e2e_aba.mjs   # aba: DOM, escrita, RECUSA, badge

**A CONTRADICAO QUE O E2E DE ABA EXPOS** (`E-1033`): o protocolo tinha
`loopbackAutorizado()` -- e **ninguem o chamava**; e o `content_scripts.matches` cobria
**so** `127.0.0.1`/`localhost`, exactamente as origens que o `validarUrl` **sempre**
recusa. *A unica origem onde o content script era injectado era a unica que o HADES nao
podia usar.* Cada metade tinha o seu teste verde; o defeito vivia entre elas.

**O opt-in de rede privada e' do UTILIZADOR**: `chrome.storage.local`, chave canonica
`OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE`, **desligado por omissao**, controlo visivel no
popup. Fail-closed: sem conseguir ler, continua recusado.

**E O `sendMessage` DO WORKER NAO FUNCIONA PARA SI PROPRIO** (`Receiving end does not
exist`) -- nos E2E, emita do **popup**, que e' um contexto de extensao valido.

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: uma protecao que ninguem consegue exercer nao protege -- esconde;
duas metades coerentes podem ser juntas impossiveis; e um campo de senha recusado com o DOM
vazio vale mais do que dez documentos a dizer que a recusa existe.

### Instrucoes de retomada (janela `E-1034`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1034`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 26 passos.** Tres E2E entram por **opt-in** (`-ComE2E`), porque abrem o Edge:
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_local_gates.ps1 -ComE2E

**Os tres E2E, sozinhos** (todos exit 0):
  node scripts/hades_browser_bridge_e2e.mjs            # canal: extensao <-> Companion
  node scripts/hades_browser_bridge_e2e_aba.mjs        # aba: DOM, escrita, RECUSA, badge
  node scripts/hades_browser_bridge_e2e_takeover.mjs   # takeover: o agente PARA

**DOIS DEFEITOS QUE SO O E2E VIA** (`E-1034`):
1. **a escrita do agente accionava o takeover** -- o `preencher` despacha `input`, e o
   listener contava isso como gesto humano. O agente escrevia uma vez e perdia o controlo
   por sua propria accao. Curado com a flag `escrevendoPeloAgente`.
2. **`lerEstado` apagava o campo novo** -- ele reconstroi o objecto com uma lista fixa de
   campos, logo qualquer campo que nao esteja na lista e' perdido a cada leitura.

**O TAKEOVER TEM DE SER LIMPO NOS DOIS LADOS**: devolver pelo popup limpa o service worker
**e** o content script. *Um controlo que so' existe num dos lados nao devolve nada.*

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um scope declarado sem implementacao e' pior do que um scope
ausente; a automacao pode accionar o travao que existe para a travar; e distinguir QUEM
escreve e' a diferenca entre um travao e um obstaculo.

### Instrucoes de retomada (janela `E-1035`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1035`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 27 passos.** QUATRO E2E entram por **opt-in** (`-ComE2E`), porque abrem o Edge:
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_local_gates.ps1 -ComE2E

**Os quatro E2E, sozinhos** (todos exit 0):
  node scripts/hades_browser_bridge_e2e.mjs            # canal: extensao <-> Companion
  node scripts/hades_browser_bridge_e2e_aba.mjs        # aba: DOM, escrita, RECUSA, badge
  node scripts/hades_browser_bridge_e2e_takeover.mjs   # takeover: o agente PARA
  node scripts/hades_browser_bridge_e2e_origem.mjs     # origem: navegar revoga

**TRES LOCAIS, UM DEFEITO** (`E-1033` -> `E-1035`): o opt-in de rede privada nao era
passado ao `validarUrl`. Curei o `conectarAbaAtiva` e o `tabs.onUpdated`; o `tratarAccao`
ficou. *Curar dois de tres sitios deixa o terceiro a recusar o que os outros aceitam.*
=> **procure TODOS os chamadores, nao o que falhou.**

**Limitacao que NAO foi testada e pode ser buraco real**: `history.pushState` nao dispara
`tabs.onUpdated` com URL novo, logo um SPA que mude de origem por History API **nao** seria
revogado. Registado como divida, nao como coberto.

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: codigo de seguranca sem teste e' uma promessa; curar o sintoma
que se mediu nao fecha a classe do defeito; e um SPA que muda de origem sem disparar
`onUpdated` e' uma pergunta em aberto, nao uma resposta.

### Instrucoes de retomada (janela `E-1036`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1036`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 29 passos.** QUATRO E2E por **opt-in** (`-ComE2E`, abrem o Edge) e **duas**
catracas novas (passos 21 e 22):
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_local_gates.ps1 -ComE2E
  python -X utf8 scripts\hades_bridge_optin_ratchet.py        # a catraca
  python -X utf8 scripts\hades_bridge_optin_ratchet_test.py   # 13 assercoes, fault injection

**A CATRACA DO OPT-IN** (`E-1036`) mede o **codigo**, nao o resultado: falha se um
chamador novo de `validarUrl`/`guardarRota` nao passar `permitirPrivado`. O defeito
apareceu em **tres** locais em tres rondas (`E-1033` -> `E-1035`); a cura pontual nao
fecha a classe.

**E O TESTE NAO MUTA PRODUCAO**: `--raiz CAMINHO` aponta a catraca a uma **copia**; a
injecao vai para la', e o teste guarda o hash do ficheiro real antes e compara no fim.
*Um teste que muta o codigo de producao e' um risco, nao uma prova.*

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: uma cura pontual nao fecha a classe do defeito; o gate certo
mede o codigo que o produziria, nao o sintoma; uma catraca que fica vermelha com tudo nao
discrimina nada; e um teste que muta producao e' um risco.

### Instrucoes de retomada (janela `E-1037`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1037`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 30 passos.** CINCO E2E por **opt-in** (`-ComE2E`, abrem o Edge):
  node scripts/hades_browser_bridge_e2e.mjs            # canal: extensao <-> Companion
  node scripts/hades_browser_bridge_e2e_aba.mjs        # aba: DOM, escrita, RECUSA, badge
  node scripts/hades_browser_bridge_e2e_takeover.mjs   # takeover: o agente PARA
  node scripts/hades_browser_bridge_e2e_origem.mjs     # origem: navegar revoga
  node scripts/hades_browser_bridge_e2e_pushstate.mjs  # reload, pushState, badge

**DOIS DEFEITOS QUE A DIVIDA ESCONDIA** (`E-1037`):
1. **`onUpdated` SEM URL revogava a concessao.** O browser emite `changeInfo` com **so**
   `status` a cada `pushState`, `replaceState`, **reload** e mudanca de titulo. O codigo
   fazia `validarUrl(info.url ?? "")` -- validava a **string vazia**, falhava, e concluia
   "nova origem". ***A navegacao mais comum que existe matava a sessao.***
2. **O badge mentia.** Com a concessao viva, o Chrome limpava o badge ao navegar e ele
   ficava `""`. ***Um indicador que nao reflecte o estado e' pior do que indicador nenhum.***
   E o badge tinha **dois donos** (inline em dois sitios) => extraido para `marcarBadge`.

**A LICAO**: *uma divida declarada e' uma medicao a fazer, nao uma ressalva* -- e a
medicao dela encontrou dois defeitos que ela escondia. *O valor de medir uma duvida nao
esta' na resposta a essa duvida: esta' no que aparece pelo caminho.*

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um teste que so' exercita o caminho feliz nao ve o caminho
vazio; decidir sobre um vazio inventado e' o defeito e nao decidir e' a cura; e um
indicador que nao reflecte o estado da' confianca a quem nao a devia ter.

### Instrucoes de retomada (janela `E-1038`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1038`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 31 passos.** SEIS E2E por **opt-in** (`-ComE2E`, abrem o Edge):
  node scripts/hades_browser_bridge_e2e.mjs            # canal: extensao <-> Companion
  node scripts/hades_browser_bridge_e2e_aba.mjs        # aba: DOM, escrita, RECUSA, badge
  node scripts/hades_browser_bridge_e2e_takeover.mjs   # takeover: o agente PARA
  node scripts/hades_browser_bridge_e2e_origem.mjs     # origem: navegar revoga
  node scripts/hades_browser_bridge_e2e_pushstate.mjs  # reload, pushState, badge
  node scripts/hades_browser_bridge_e2e_bfcache.mjs    # voltar atras nao revive

**A FAMILIA `onUpdated` ESTA' FECHADA** (`E-1035` -> `E-1038`):
- a **ida** para outra origem revoga -- com URL (`E-1035`);
- o evento **sem URL** NAO revoga mais (era o defeito do `E-1037`);
- o **reload** preserva a concessao e o badge (`E-1037`);
- o **regresso** traz URL e **nao revive** nada (`E-1038`).

**O LIMITE DECLARADO**: o BFCache **nao chegou a correr** -- `pageshow` deu
`persisted: false`, o browser recarregou. *Um resultado favoravel nao e' uma cobertura.*

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um resultado favoravel nao e' uma cobertura; declarar o limite
e' parte do resultado; e a honestidade do limite vale mais do que a alegria do verde.

### Instrucoes de retomada (janela `E-1039`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1039`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 32 passos.** SETE E2E por **opt-in** (`-ComE2E`, abrem o Edge):
  node scripts/hades_browser_bridge_e2e.mjs            # canal: extensao <-> Companion
  node scripts/hades_browser_bridge_e2e_aba.mjs        # aba: DOM, escrita, RECUSA, badge
  node scripts/hades_browser_bridge_e2e_takeover.mjs   # takeover: o agente PARA
  node scripts/hades_browser_bridge_e2e_origem.mjs     # origem: navegar revoga
  node scripts/hades_browser_bridge_e2e_pushstate.mjs  # reload, pushState, badge
  node scripts/hades_browser_bridge_e2e_bfcache.mjs    # voltar atras nao revive
  node scripts/hades_browser_bridge_e2e_frames.mjs     # shadow aberto, iframe, limites

**O `observar` MUDOU DE CONTRATO** (`E-1039`): devolve `limites` --
`{parcial, totalNoDocumentoPrincipal, shadowRoots:{...}, frames:{...}, truncadoPorLimiteDeNos,
limitacaoEstrutural}`. *Quem consumir `total` tem agora de ler `limites` para saber se a
observacao e' completa -- e a resposta honesta e' "pode nao ser".*

**DOIS ERROS MEUS, REGISTADOS**:
1. `parcial` incluiria a limitacao estrutural => seria `true` sempre => **um aviso
   permanente deixa de ser aviso: vira ruido**. Separados em dois campos.
2. O contador de shadow fechados era **um palpite** (`custom element` com filhos) e deu
   `0` para uma pagina que tinha um. *Um palpite que falha e' pior do que uma declaracao
   de ignorancia: da' uma contagem que parece medida.*

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um observador que devolve parcial como completo e' pior do que
um que se recusa; `zero` parece medido e `nao sei` nao parece nada -- e o segundo e' a
verdade; e nem tudo o que limita e' uma perda: parte e' a forma do mundo.

### Instrucoes de retomada (janela `E-1040`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1040`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 34 passos.** OITO E2E por **opt-in** (`-ComE2E`) e as **17 assercoes** da
politica de ficheiros (passo 27, sem browser):
  node scripts/hades_browser_bridge_e2e_arquivos.mjs     # browser:files decide
  node --test apps/hades-browser-bridge/test/arquivos.test.js

**A POLITICA DE FICHEIROS** (`E-1040`), na ordem em que e' aplicada -- *a ordem e' a
politica*: nome com caminho -> **proibido**; sem `sha256` -> **proibido**; executavel ->
**exige_aprovacao**; sensivel -> **exige_aprovacao**; upload (qualquer) ->
**exige_aprovacao**; desconhecido -> **exige_aprovacao**; inerte em download -> **permitido**.

**UM EXECUTAVEL NUNCA FICA «PERMITIDO»** -- nem com aprovacao: continua `exige_aprovacao`,
com o motivo a dizer que foi aprovado nesta transferencia.

**`ok` vs `podesTransferir`**: `ok` diz que a avaliacao correu; `podesTransferir` diz se
esta' autorizado. *A ambiguidade num campo de permissao e' um defeito: quem le' `ok` age.*

**`lerEstado` FOI CURADO NA RAIZ**: a lista fixa de campos apagava qualquer campo novo --
o `takeover` caiu no `E-1037` e o `arquivos` ia cair agora. Agora normaliza so' os campos
**com regra** e **preserva tudo o resto**. *Curar o campo que falhou nao cura a lista.*

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um scope declarado sem implementacao e' pior do que um ausente,
e uma mensagem que promete o que nao existe e' pior ainda; uma lista fixa de campos e' uma
armadilha que se arma a cada campo novo; e quem pede nao certifica o que pede.

### Instrucoes de retomada (janela `E-1041`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1041`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 34 passos.** OITO E2E por **opt-in** (`-ComE2E`) e as **23 assercoes** da
politica de ficheiros (passo 27, sem browser):
  node scripts/hades_browser_bridge_e2e_arquivos.mjs     # browser:files decide
  node --test apps/hades-browser-bridge/test/arquivos.test.js

**A PROVA DE APROVACAO** (`E-1041`): `{aprovacaoId, fingerprint, aprovadoPor}`. Os tres
sao obrigatorios -- *uma aprovacao sem autor nao tem responsavel*. Um **booleano solto**
(`aprovado: true`) e' recusado com `booleano_solto_nao_e_aprovacao`, **com nome**, e nao
ignorado em silencio. E o `fingerprint` tem de **casar com o pedido**.

**OS TRES VEREDICTOS**: `permitido` (inerte em download, sem prova) / `exige_aprovacao`
(falta ou nao casa) / `autorizado` (a prova casa com ESTE pedido).

**A FRONTEIRA, DITA SEM DESFARCE**: a extensao **nao verifica** a prova contra o store --
nao tem acesso. *A forma esta' implementada e testada; a autoridade nao.* Uma prova bem
formada mas falsa passaria. Adoptou-se a forma do que ja' existia
(`FindApproved` + `Consume`) em vez de inventar um segundo sistema de aprovacao.

**`import()` NAO EXISTE NO SERVICE WORKER** -- `TypeError: import() is disallowed on
ServiceWorkerGlobalScope`. O fingerprint calcula-se **fora**, com o mesmo modulo, e vai
medido para dentro.

**A CATRACA APANHOU ANTES DO REGISTO**: o `mission_check` ficou VERMELHO com
*«evidencia CITADA e NAO REGISTADA ... E-1041»* e *«CABECA DO INDICE ATRAS DO CODIGO»* --
*um ID citado sem entrada nao e' verificavel por quem audita*. O registo veio depois.

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um campo de permissao que quem pede controla nao e' uma
permissao; nao inventar um sistema quando a forma do existente serve; e ignorar um campo
que alguem envia deixa quem o envia sem saber porque nao funcionou.

### Instrucoes de retomada (janela `E-1042`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1042`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 34 passos.** OITO E2E por **opt-in** (`-ComE2E`) e as **25 assercoes** da
politica de ficheiros (passo 27, sem browser):
  node scripts/hades_browser_bridge_e2e_arquivos.mjs     # browser:files decide
  node --test apps/hades-browser-bridge/test/arquivos.test.js

**A AUTORIDADE ENTROU NO CONTRATO** (`E-1042`): `autoridade: forma_apenas` acompanha
**todo** o veredicto -- mesmo `permitido`. A extensao **nunca** diz `verificada`
(`verificada` so' com afirmacao explicita, e um valor desconhecido cai no honesto).
E `avisoDeAutoridade` diz o limite **por palavras**, visivel no popup e no registo.

**A FRONTEIRA MEDIDA**: `serviceAgentApprovals` existe mas e' de **SESSAO**
(`organizationID` vem da sessao). O Companion e' um processo local **sem sessao**.
*Dar-lhe a sessao do utilizador seria pior: um processo local com a sessao do utilizador
e' exactamente o que nao queremos.*

**PASSO DE DESBLOQUEIO EXACTO**: uma rota **no Companion** (nao na extensao) que use o
**canal emparelhado** (`pairing:iniciar` / `handshake:*`, ja' existentes do `E-1031`)
para se autenticar perante o runtime e consultar `FindApproved`. So' entao `verificada`
tem quem o possa afirmar.

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: a fronteira certa nao e' um silencio -- e' um campo; nao ligar
o que nao se pode autenticar; e dizer «nao da'» seria falso, dizer «da', mas nao assim»
e' a verdade.

### Instrucoes de retomada (janela `E-1043`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1043`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 34 passos.** OITO E2E por **opt-in** (`-ComE2E`) e as **25 assercoes** da
politica de ficheiros (passo 27, sem browser):
  node scripts/hades_browser_bridge_e2e_arquivos.mjs     # browser:files decide
  node --test apps/hades-browser-bridge/test/arquivos.test.js

**A ROTA DE APROVACAO** (`E-1043`): `aprovacao:consultar` (Companion) e
`arquivo:consultarAutoridade` (extensao). O Companion responde sempre -- e responde
`verificada: false`, `motivo: "sem_rota_autenticavel_no_runtime"`, com o
`desbloqueio` dito na propria resposta.

**O COMPANION NUNCA DIZ `verificada: true`**: oito assercoes no `companion.test.js`,
incluindo **quatro tentativas de o forcar** (`verificada: true`, `forcar: true`,
`aprovado: true` no pedido). Todas devolvem `false`.

**A EXTENSAO NAO DEPENDE DA RESPOSTA**: `verificada: true` sobe a autoridade; `false`
deixa `forma_apenas`; canal indisponivel deixa `forma_apenas` com `canal_indisponivel`.
*Nunca o contrario: um canal em baixo nao pode fazer uma decisao parecer melhor.*

**`chrome.runtime.reload()` FECHA O POPUP** -- medido: `Target page, context or browser
has been closed`. E nao era preciso: o caso **sem** `nativeMessaging` mede-se sem
recarregar, e o caso **com** ela ja' esta' provado no `E-1032`.

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: o que faltava nao era a verificacao -- era a pergunta; uma
rota que responde «nao posso» e' melhor do que uma rota que nao existe; e nao repetir um
teste que ja' existe noutro sitio, onde ele corre no sitio certo.

### Instrucoes de retomada (janela `E-1044`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1044`; PR #438 **aberto** e
**nao tocado**; `reviewDecision` **vazio** => sem revisao humana independente.

**Bateria: 36 passos.** OITO E2E por **opt-in** (`-ComE2E`) e **tres** catracas
(passos 21, 22, 28, 29):
  python -X utf8 scripts\hades_approval_hook_ratchet.py        # a catraca do encaixe
  python -X utf8 scripts\hades_approval_hook_ratchet_test.py   # 17 assercoes, 4 injecoes

**A CATRACA DO ENCAIXE** (`E-1044`) vigia: nenhuma rota HTTP chama `FindApproved`,
nenhum handler de approvals dispensa a sessao, o encaixe `MOTIVO_SEM_ROTA` existe e e'
usado. Se o mundo mudar, ela **acorda e diz o que fazer**. Aceita `--raiz CAMINHO` para
os testes trabalharem numa copia.

**A LICAO DOS QUATRO FALSOS POSITIVOS**: o detector conhecia duas formas de ler a sessao
e havia cinco. *Um detector que conhece duas formas de uma coisa e' um detector que nao a
conhece.* Agora as formas sao uma **lista declarada** (sete) e o relatorio **nomeia qual**
cada handler usou -- um falso positivo futuro e' visivel a olho.

**E UM ERRO MEU**: o teste copiava so' `agent_routes.go` e `companion.mjs`, e a catraca
acordou com «`FindApproved` DESAPARECEU» -- **um falso vermelho DO TESTE**. *Um teste que
nao monta o mundo que o checker mede acusa o checker de um defeito que e' seu.*

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um passo de desbloqueio escrito nao e' um passo de desbloqueio
vigiado; um detector que conhece duas formas de uma coisa nao a conhece; e uma catraca
nao e' um obstaculo -- e' um lembrete que nao se esquece.

### Instrucoes de retomada (janela `E-1045`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1045`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**Bateria: 36 passos, 27 OK, 0 FAIL.** OITO E2E por opt-in e **tres** catracas.

**OS 12 FINDINGS, MEDIDOS**:
- **#2-#6** -> `E-909` (`TRIAGADOS` por assinatura de conteudo, `sem_comentarios`, o
  `_test.py` a importar `checker.TRIAGADOS`);
- **#7** -> `E-912` (o SIMBOLO, nao a chamada);
- **#9** -> `E-910` (comentario corrigido por medicao, texto antigo preservado);
- **#10 e #11** -> **FALSOS POSITIVOS** do bot;
- **#1** -> tratado em rodada anterior;
- **#8** -> **nao totalmente revalidado** (inspeccionado por nome de teste, nao pela
  alegacao contra o comportamento do checker).

**#11 -- E EU ERREI TRES VEZES A MEDI-LO**: 767, depois 303, depois 768. A quarta medicao
usou o `marked` (ja' no `node_modules`) e deu **0**: as 837 linhas produzem 5 campos.
*NAO SE ADIVINHA UM RENDERIZADOR: USA-SE UM.* E: *antes de curar, contar a coisa certa* --
eu ia corrigir 768 linhas correctas.

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um finding de bot e' uma hipotese -- duas das doze eram falsas;
quando a pergunta e' «o que isto faz?», a resposta nao se deduz, executa-se; e a pressa de
corrigir e' o caminho mais curto para estragar.

### Instrucoes de retomada (janela `E-1046`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1046`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**Bateria: 27 passos, 28 OK, 0 FAIL.** OITO E2E por opt-in e **quatro** catracas.

**OS 12 FINDINGS DO PR #438 -- RESOLVIDOS**:
- **#1** tratado antes; **#2-#6** `E-909`; **#7 e #8** `E-912`; **#9** `E-910`;
- **#10** FALSO POSITIVO (tabelas diferentes podem ter colunas diferentes);
- **#11** NAO SE REPRODUZ (838 entradas renderizam com 5 campos via `marked`);
- **#12** feito por construcao (os doze medidos no HEAD).

**O #8, FECHADO COM A CONTRAPROVA**: o teste afirmava «e obsoleta» e o comentario deliberado
do checker (`:216-219`) diz que essa acusacao foi **removida de proposito**. Corrido agora:
  go test ./internal/agent/ -run 'TestDelegacaoDeEscopoRecusaOrganizacaoVazia|TestTarefasDeBenchmarkTemNomeUnicoENaoVazio|TestValidacaoDeNomeDeTarefaReprovaVazioEDuplicado' -count=1
  => ok github.com/ollama/ollama/internal/agent 5.620s

**O RENDER CHECK DO INDICE e' o passo 30** (`hades_index_render_check.mjs`): renderiza as
838 entradas com o `marked` e exige 5 campos. *Nao se adivinha um renderizador: usa-se um.*

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um bot que erra duas vezes em doze ensina a nao copiar; um arquivo
que existe para impedir afirmacao sem medicao pode ele proprio afirmar sem medicao; e o
trabalho e' tecnico -- a thread e' do GitHub e a decisao de a fechar e' do proprietario.

### Instrucoes de retomada (janela `E-1047`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1047`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**Bateria: 27 passos, 28 OK, 0 FAIL.** OITO E2E por opt-in e **quatro** catracas.

**O ESTAGIO 21, MEDIDO** (era o unico dos 10 estagios livres que a minha nota dizia
«nao iniciado» -- *e a nota estava errada*):
  $env:HADES_T2_MEDIR_MODELO='1'
  go test ./internal/agent/ -run 'TestMatrizRealComModelosLocais' -count=1 -timeout 30m
  => --- PASS: TestMatrizRealComModelosLocais (599.88s)
  => ok github.com/ollama/ollama/internal/agent 604.638s

**RESULTADO**: 3/8 verde, **4 trapacearam**, 1 vermelho. E os **veredictos repetiram-se um
dia depois** (artefacto de `2026-10-15T22:07Z` vs `2026-10-16T23:41Z`): so' os tempos
mudaram. *Reprodutibilidade nao e' o tempo ser igual: e' a conclusao ser igual quando o
tempo nao e'.*

**O ANTI-TRAPACA E' CODIGO, NAO DOCUMENTO**: `harness_benchmark.go:286` -- «O teste do
fixture NUNCA e' alvo: harness que "conserta" o teste esta' trapaceando». O modelo
recebe `HARNESS_TRAFICOU` e um aviso na tentativa seguinte (`:305`).

**O WARM/COLD NAO EXISTE NO `cmd/bench`**: `unloadModel` (`bench.go:504`) corre ANTES de
medir -- toda corrida e' fria por construcao (cold 19,95 s vs warm 21,31 s = 0,94x).
O par real esta' DENTRO da corrida: 92,2 / 89,7 / 91,3 tok/s -- warm/cold **0,98x**.

**O WIP ALHEIO FICOU INTACTO**: o teste reescreveu `HADES_T2_MATRIX.md` e o
`git diff --numstat` continua **`9+/9-`**; `HADES_T2_REPETICAO.md` (`46+/0`) nao foi tocado.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: uma nota que dizia «nao iniciado» e um estagio a espera de
alguem o correr; medir a taxa de sucesso sem medir a trapaca seria medir a mentira; e a
comparacao externa continua NAO_EXECUTADA -- nao ha' numero inventado.

### Instrucoes de retomada (janela `E-1048`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1048`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**Bateria: 28 passos.** OITO E2E por opt-in, **quatro** catracas, o render do indice e o
inventario do worktree.

**O WORKTREE, MEDIDO (passo 31)** -- *nao contado, listado*:
  python -X utf8 scripts\hades_worktree_inventory.py
  => A. CONTEUDO: 0   B. SO' FIM-DE-LINHA: 11   C. NAO RASTREADO: 2
  => git diff --stat = 0 ; git diff --ignore-cr-at-eol --numstat = 0

***Reportei «14 entradas» durante dezenas de rondas sem nunca listar. O numero era 14; o
trabalho real era ZERO.*** A licao: *uma contagem nao e' um inventario* -- e *reportar o
numero certo da coisa errada e' uma forma de mentir sem dizer uma palavra falsa.*

**O WIP DO PROMPT ESTA' INTEGRADO**: `app/secrets/secrets.go`, `parser/parser.go`,
`server/desktop_local_routes.go`, `app/ui/app/src/lib/salaApi.ts`,
`scripts/hades_bundle_windows.ps1`, `THIRD_PARTY_NOTICES.md` -- todos em disco, commitados
na FASE 1. O que resta e' residuo de `core.autocrlf`.

**O ESTAGIO 21, MEDIDO** (`E-1047`): `PASS` em 599,88 s -- 3/8 verde, 4 trapacearam, 1
vermelho; veredictos identicos um dia depois. O anti-trapaca e' codigo
(`harness_benchmark.go:286`).

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: uma contagem nao e' um inventario; um `M` num `git status` nao
e' necessariamente conteudo; e reportar o numero certo da coisa errada e' uma forma de
mentir sem dizer uma palavra falsa.

### Instrucoes de retomada (janela `E-1049`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1049`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**Bateria: 28 passos, 29 OK, 0 FAIL.** OITO E2E por opt-in, **quatro** catracas, o render
do indice e o inventario do worktree.

**A REPETICAO T2, REVALIDADA** (`E-1049`):
  $env:HADES_T2_MEDIR_MODELO='1'; $env:HADES_T2_MODELS='qwen2.5:7b'; $env:HADES_T2_REPEATS_N='3'
  go test ./internal/agent/ -run 'TestT2RepeticaoEstatistica' -count=1 -timeout 40m
  => --- PASS: TestT2RepeticaoEstatistica (209.78s)
  => ok github.com/ollama/ollama/internal/agent 215.740s

**O `46+/0` RESISTIU, FIXTURE A FIXTURE**: `soma` 3/3 (11,8 -> 15,1 s) - `comparacao` 0/3
(9,5 -> 11,1 s) - `caso-limite` 0/3 (10,7 -> 25,0 s) - `fatiamento` 0/3 (20,8 -> 18,7 s).
*Veredictos identicos tres dias depois; so' as duracoes mudaram.*

**E O ARTEFACTO PASSOU DE AFIRMACAO A MEDICAO**: a data era `2026-10-13T13:41:29Z` e eu
tinha-o commitado no `E-1047` sem o correr (com a divida declarada no commit). Agora e'
`2026-10-16T23:52:54Z`. *Declarar a divida no commit nao a paga; correr o teste paga.*

**DUAS CONFIRMACOES DE REPRODUTIBILIDADE NA MESMA JANELA**: a matriz T2 (8 fixtures,
veredictos identicos **um** dia depois) e a repeticao (4 fixtures x 3 execucoes,
identicos **tres** dias depois).

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um numero que resiste a tres dias e' um numero; declarar a
divida no commit nao a paga; e a repeticao mede estabilidade, nao qualidade -- `0/3`
estavel continua a ser `0/3`.

### Instrucoes de retomada (janela `E-1050`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1050`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**Bateria: 28 passos, 29 OK, 0 FAIL.** OITO E2E por opt-in, **quatro** catracas, o render
do indice e o inventario do worktree.

**LEIA ISTO ANTES DE FAZER PUSH -- O `pr-gate` E O RITMO**:
  - `pr-gate.yaml:56` lista os gates profundos exigidos (entre eles `test` e
    `dz23-agentic-quality`);
  - `:61` tem `deadline=$(($(date +%s) + 1800))` -- **30 min**;
  - `:91` espera por uma **substituta** quando um gate foi cancelado;
  - `:100-101` reprova com `Timeout aguardando os gates profundos deste SHA`.

**CADA PUSH CANCELA AS RUNS DO SHA ANTERIOR** -- e o `pr-gate` desse SHA fica a esperar
por gates que nunca vao correr. Medido: `55327160c` (`test` e `dz23-agentic-quality`
cancelled) -> `pr-gate` **failure**; `4157a459b` (os mesmos dois **success**) -> **6/6**.

***Nao faca push enquanto as runs do SHA corrente nao terminarem.*** E **nao** cure isto
aceitando `cancelled` no `pr-gate`: *a gate esta' certa em reprovar, e a cura por baixo
esta' proibida pelo owner por palavras.*

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um vermelho que nao e' um defeito continua a ser um vermelho;
quando a cura facil exige enfraquecer a medida, a resposta e' mudar o procedimento; e o
sinal estava a medir o intervalo entre pushes, nao a qualidade do commit.

### Instrucoes de retomada (janela `E-1051`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1051`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- LEIA ANTES DE ENVIAR QUALQUER COISA**:
  1. faca o commit localmente;
  2. ESPERE que as runs do SHA remoto corrente fechem com `failure=0`;
  3. so' entao faca `git push`.
  *Cada push cancela as runs do SHA anterior; o `pr-gate` espera por gates do MESMO SHA
  (`pr-gate.yaml:91`) com deadline de 30 min (`:61`) -- e reprova se eles nao terminarem.*

**A PROVA, TRES SHAs**: `55327160c` (gates **cancelled**) -> `pr-gate` **FAILURE**;
`4157a459b` e `ba62baf88` (gates **success**) -> **6/6**. *Um push rapido garante um
vermelho que nao e' um defeito.*

**E NAO CURE ISTO POR BAIXO**: aceitar `cancelled` como aprovacao no `pr-gate` esta'
**proibido pelo owner por palavras**. *A gate esta' certa em reprovar.* Mudar a gate
**toca a politica do gate** e e' decisao do proprietario.

**O ESTAGIO 21, MEDIDO NESTA JANELA**:
  - matriz T2 real: `PASS` 599,88 s -- 3/8 verde, 4 trapacearam, 1 vermelho; veredictos
    identicos **um dia** depois (`E-1047`);
  - repeticao T2: `PASS` 209,78 s -- o `46+/0` resiste **tres dias** (`E-1049`);
  - anti-trapaca e' **codigo** (`harness_benchmark.go:286`);
  - o warm/cold do `cmd/bench` **nao existe** (frio por construcao, `bench.go:504`).

**INVENTARIO DO WORKTREE (passo 31)**: *conteudo 0, so' fim-de-linha 11, nao rastreado 1*.
***Uma contagem nao e' um inventario.***

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um vermelho que nao e' um defeito continua a ser um vermelho;
uma previsao que se cumpre vale mais do que uma explicacao que se acomoda; e um
procedimento que ninguem escreveu e' um procedimento que cada um inventa.

### Instrucoes de retomada (janela `E-1052`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1052`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA**:
  1. faca o commit localmente;
  2. ESPERE que as runs do SHA remoto fechem com `failure=0`;
  3. so' entao faca `git push`.
  *Resultado medido de seguir isto: QUATRO SHAs 6/6 consecutivos -- `4157a459b`,
  `ba62baf88`, `67581a993`, `2e0873635`.*

**A PROVA DO MECANISMO, SEIS SHAs**: `55327160c` (gates cancelled) -> `pr-gate`
**failure**; `cda756f82` (gates cancelled) -> `pr-gate` **failure** -- *previsto ANTES no
`E-1051`, e cumpriu-se*; `4157a459b`, `ba62baf88`, `67581a993`, `2e0873635` (gates
terminados) -> **6/6**. *Um push rapido garante um vermelho que nao e' um defeito.*

**NAO CURE ISTO POR BAIXO**: aceitar `cancelled` como aprovacao no `pr-gate` esta'
**proibido pelo owner por palavras**. *A gate esta' certa em reprovar.* Mudar a gate
**toca a politica do gate** e e' decisao do proprietario.

**O ESTAGIO 21, MEDIDO NESTA JANELA**: matriz T2 real `PASS` 599,88 s (3/8 verde, 4
trapacearam, 1 vermelho; veredictos identicos um dia depois -- `E-1047`); repeticao T2
`PASS` 209,78 s (o `46+/0` resiste tres dias -- `E-1049`); anti-trapaca e' **codigo**
(`harness_benchmark.go:286`); o warm/cold do `cmd/bench` **nao existe** (frio por
construcao, `bench.go:504`).

**INVENTARIO DO WORKTREE (passo 31)**: *conteudo 0, so' fim-de-linha 11, nao rastreado 1*.
***Uma contagem nao e' um inventario.***

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: uma previsao escrita antes e' a unica prova de que a explicacao
serve; quando a cura facil exige enfraquecer a medida, a resposta e' mudar o procedimento;
e um procedimento que ninguem escreveu e' um procedimento que cada um inventa.

### Instrucoes de retomada (janela `E-1053`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1053`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (quatro 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O ESTAGIO 21, AGORA LIGADO AOS ARTEFACTOS** (`E-1053`):
  - `docs/parity/HADES_T2_MATRIX.md` -- matriz real, 3/8 verde, 4 `HARNESS_TRAFICOU` (`E-1047`);
  - `docs/parity/HADES_T2_REPETICAO.md` -- repeticao, `46+/0` resiste tres dias (`E-1049`);
  - a **seccao 6** de `HADES_STAGE_21_BENCHMARK.md` traz `artefacto -> comando -> evidencia`;
  - os outros dois documentos tem a ligacao **dentro do bloco de citacao**.

**A LIGACAO E' TEXTUAL**: *se alguem renomear `HADES_T2_MATRIX.md`, nada acusa.* Fechar
com um checker e' a proxima accao -- mesmo padrao da catraca do encaixe (`E-1044`).

**O QUE OS ARTEFACTOS NAO PROVAM**: comparacao externa `NAO_EXECUTADA`; taxa 3/8 com
modelos de 3-7B; `N=3` por fixture (contagem, nao probabilidade); o warm/cold do
`cmd/bench` nao existe (`bench.go:504` desarrega antes de medir).

**DUAS CATRACAS COMPLEMENTARES**: `hades_index_shape_check.py` varre o **indice**;
`hades_mission_check.py` varre **todo o codigo**. **Um ID novo continua do MAIOR cunhado.**

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um metodo que nao aponta para o artefacto e' um metodo que o
auditor nao segue; um resultado sem ponteiro existe e nao se encontra; e acrescentar
informacao pode degradar a forma -- que e' como a informacao se le.

### Instrucoes de retomada (janela `E-1054`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1054`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (cinco 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**A CATRACA DA LIGACAO docs->artefactos** (`E-1054`, passos 32 e 33):
  python -X utf8 scripts\hades_stage_artifacts_check.py
  python -X utf8 scripts\hades_stage_artifacts_check_test.py   # 18 assercoes, 4 injecoes
  *Mede: artefacto existe (com tamanho -- `SUSPEITO` abaixo de 400 bytes), documento cita
  o artefacto, e a fonte da verdade e' o mapa `LIGACOES` declarado no proprio checker.*

**E ANTES DE ESCREVER UM INSTRUMENTO, MEDIR O QUE JA' EXISTE**: havia um
`hades_discoverability_check.py` -- li-o e corri-o antes de escrever; protege a proibicao
de **prometer ranking**, nao a ligacao. *Reutilizar primeiro nao e' regra de estilo: e' a
diferenca entre acrescentar e duplicar.*

**O ESTAGIO 21, MEDIDO E LIGADO**: matriz T2 real `PASS` 599,88 s (3/8 verde, 4
trapacearam, 1 vermelho -- `E-1047`); repeticao `PASS` 209,78 s (`46+/0` resiste tres dias
-- `E-1049`); a **seccao 6** de `HADES_STAGE_21_BENCHMARK.md` liga-os com comandos.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: ligar com texto nao e' ligar; declarar a fraqueza de uma cura
nao e' cura-la, mas e' o que permite ir procura-la; e um ficheiro vazio nao e' um artefacto.

### Instrucoes de retomada (janela `E-1055`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1055`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (seis 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**A VARREDURA DE LIGACOES, E O QUE ELA ENSINOU** (`E-1055`): 131 artefactos, 143
documentos -> **10 «ponteiros quebrados»** e **1 «artefacto orfao»** -> verificados um a
um: **ZERO defeitos reais, QUATRO erros meus.**
  - comparar `basename` contra uma lista de **caminhos**;
  - `fullmatch` que falha em nomes longos e **exclui em silencio**;
  - tratar **ficheiros gerados em runtime** (`INSTALACAO.json`) como artefactos do repo;
  - tratar itens **«A CRIAR»** como ponteiros quebrados.

***NAO ALARGUEI O MAPA `LIGACOES`.*** *Um mapa com falsos positivos e' pior do que nenhum
mapa -- e a catraca do `E-1054` acusa com precisao porque as 6 entradas foram todas
verificadas.*

**E A PERGUNTA QUE EU NAO FIZ**: *quais documentos **afirmam produzir** um artefacto e nao
o citam?* Medir «quais citam zero» da' 113 de 143 -- e a maioria e' narrativa. *A pergunta
e' sobre afirmacao, nao sobre ausencia.*

**O ESTAGIO 21, MEDIDO E LIGADO**: matriz T2 real `PASS` 599,88 s (`E-1047`); repeticao
`PASS` 209,78 s (`E-1049`); a catraca da ligacao com 18 assercoes e 4 injecoes (`E-1054`).

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: quando uma medicao produz muitos achados de uma vez, a primeira
hipotese e' a medicao; um mapa com falsos positivos e' pior do que nenhum; e um instrumento
que produz falsos positivos nao deve virar gate.

### Instrucoes de retomada (janela `E-1056`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1056`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (sete 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**A LINHA DA RASTREABILIDADE, ENCERRADA** (`E-1055`, `E-1056`): duas perguntas foram
feitas -- *quais documentos citam zero artefactos* (113 de 143, narrativos) e *quais
afirmam produzir um artefacto e nao o citam* (7 achados) -- e as duas deram **ZERO
defeitos reais**. Os quatro detectores de varredura **nao entraram na bateria**.
  *O resultado de procurar e nao encontrar e' «nao encontrei» -- e nao «nao existe».*

**E O ERRO DOS SETE TEM UM NOME**: o regex aceita `e' a fonte`, `gera` **e** `esta' em`,
`a partir de`. *Um detector lexico nao ve a relacao: ve os termos.*

**QUATRO MEDICOES NA MESMA JANELA QUE APONTARAM PARA O INSTRUMENTO**: o `#11`
(767/303/768 antes do renderizador); o `E-1044` (o teste que nao montava o mundo); o
`E-1055` (10 ponteiros quebrados que eram 4 erros); o `E-1056` (7 afirmacoes que eram 7
leituras erradas). ***Quando uma medicao produz muitos achados de uma vez, a primeira
hipotese e' a medicao.***

**O QUE A JANELA 142 PROVOU**: o estagio 21 mede (matriz T2 real `PASS` 599,88 s -- 3/8
verde, 4 trapacearam, veredictos repetem um dia depois); a repeticao resiste tres dias; a
ligacao vigia (18 assercoes, 4 injecoes); e o vermelho do `pr-gate` e' o **ritmo de push**.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um detector lexico ve termos, nao relacoes; procurar e nao
encontrar e' «nao encontrei»; e a diligencia sem hipotese vira ruido -- que num gate e'
pior do que um gate a menos.

### Instrucoes de retomada (janela `E-1057`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1057`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (oito 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O BUNDLE WINDOWS DO CANDIDATO** (`E-1057`):
  $env:TMP=$raiz\_tmp\webtemp; $env:TEMP=$env:TMP; $env:GOTMPDIR=$raiz\_tmp\gotmp
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_bundle_windows.ps1 -OutDir dist\hades-r143
  => dist\hades-r143\Hades-6093a6692-windows-amd64-portable-completo.zip  (48 234 478 bytes)
  => tags 200 / chat 200 [resposta real] / upstream 501  =>  RESULTADO: VERIFICADO

**DUAS ARMADILHAS MEDIDAS NESTE CAMINHO**:
  1. **`TMP` negado** -> `go build` falha com «creating work dir ... Acesso negado». *Curado na
     CHAMADA, nao no script: o script esta' certo em nao adivinhar o ambiente.*
  2. **filho orfao**: o script matava o `hades.exe` mas **nao o `llama-server`**, que e' um
     **filho** -- e ficaram **17 processos** de dez directorios. *Curado: mata o filho primeiro,
     e so' os que vivem DENTRO do directorio de verificacao.* Provado: **17 -> 0**.

**O QUE O ARTEFACTO NAO E'**: nao assinado; nao e' instalador Inno/NSIS (e' o portatil);
o runner vem **pre-construido** (`_tmp\hadesbin`) -- **nao e' build 100% reproduzivel da
fonte**; `INSTALLED=NAO`, `DEVICE_E2E=NAO`, `SIGNED_FOR_PUBLIC_RELEASE=NAO`.

**O `HadesSetup.exe` NAO FOI GERADO**: `build_windows.ps1` (1369 linhas) e `app/ollama.iss`
(223 linhas) existem; `ISCC.exe` **existe e funciona** em
`%LOCALAPPDATA%\Programs\Inno Setup 6\`. *O que falta nao e' a ferramenta: e' executar* --
e **instalar** exige aprovacao do dono.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um teste que arranca um filho e mata so' o pai deixa o filho a
segurar o directorio; um script que adivinha o ambiente esconde o ambiente que o chamador
devia declarar; e um artefacto que declara os seus limites vale mais do que um que os omite.

### Instrucoes de retomada (janela `E-1058`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1058`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (oito 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O INSTALADOR DO CANDIDATO** (`E-1058`):
  go build -trimpath -ldflags "-s -w -X=.../version.Version=$V -X=.../server.mode=release" -o dist\windows-amd64\ollama.exe .
  go build -trimpath -ldflags "-s -w -H windowsgui -X=.../app/version.Version=$V" -o dist\windows-ollama-app-amd64.exe ./app/cmd/app/
  $env:PKG_VERSION="0.2.0"; & "$env:LOCALAPPDATA\Programs\Inno Setup 6\ISCC.exe" /DARCH=amd64 .\ollama.iss
  => dist\HadesSetup.exe  24,26 MB  SHA256 a70b51073d89b7f65463f4f46d5a7a893fbab966d762a8c5aa679d52c0894456

**O BLOQUEIO NAO E' DE PERMISSAO: E' DE TOOLCHAIN.** O `build_windows.ps1` (1370 linhas)
compila o motor C++ e exige `cmake`/`ninja`/`cl` -- **todos ausentes**. Mas o `.iss` e' um
script Inno Setup: os **binarios Go** constroem-se da fonte, e o **C++ ja' esta' pronto**
(`dist\windows-amd64\lib\ollama`, 42 ficheiros). *O caminho que exigia a ferramenta nao era
o unico caminho.*

**UM AVISO DO COMPILADOR E' UMA MEDICAO**: o `ISCC` avisou `[UninstallRun]` sem `RunOnceId`;
curado com `RunOnceId: "HadesPararProcessos"` e `"HadesEsperarSaida"`; o `Warning`
**desapareceu** na recompilacao e o hash mudou -- *prova de que a cura entrou no artefacto*.

**E UM ERRO MEU PELO CAMINHO**: a primeira cura falhou porque o `for` percorria **todo** o
ficheiro e apanhava a primeira `Filename: "powershell.exe"` -- que esta' no `[Run]`, nao no
`[UninstallRun]`. *Um seletor que nao esta' limitado ao sitio onde o defeito vive apanha o
primeiro parecido.*

**O QUE NAO FOI FEITO, E E' DELIBERADO**: **nao instalei o instalador**. Instalar escreve em
`%LOCALAPPDATA%\Programs\Hades` e **exige aprovacao do dono**. `INSTALLED=NAO`,
`DEVICE_E2E=NAO`, `SIGNED_FOR_PUBLIC_RELEASE=NAO` continuam verdadeiros.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: o bloqueio nao e' de permissao, e' de toolchain; um aviso do
compilador e' uma medicao; e um seletor que nao esta' limitado ao sitio onde o defeito vive
apanha o primeiro parecido.

### Instrucoes de retomada (janela `E-1059`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1059`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (nove 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O TARBALL LINUX INSTALAVEL** (`E-1059`):
  dist\linux-amd64\r144\hades-linux-amd64-unsigned.tar.gz
  16 970 476 bytes   SHA256 1d870504a759dba2eb12d9381ae21495e9c6e87cad82fa7ffa9df3b0f46fe2db
  11 entradas: bin/hades + install.sh + uninstall.sh + VERSION + LEIA-ME +
               LICENSE + UPSTREAM_POLICY + THIRD_PARTY_NOTICES + 3 SBOMs

**O QUE O `VERSION` DIZ, E E' O QUE IMPORTA**:
  commit_do_binario=9f7379502b1d      <-- o binario NAO e' deste HEAD
  commit_do_head_no_momento=63a1b47c062a
  build_reproduzivel=NAO
  *Reconstrui-lo exigiria CGO_ENABLED=1 e gcc/clang -- ausentes -- e um bash real (o do
  WSL responde `execvpe(/bin/bash) failed`).*

**E O PACOTE FOI MONTADO SEM BASH**: o `package_linux.sh` nao corre aqui; montei o tar com
o modulo `tarfile` do Python -- **ordem alfabetica**, `mtime=0`, `uid=gid=0`, e
`install.sh`/`uninstall.sh` a `0o755`. *O caminho que exigia a ferramenta nao era o unico
caminho -- outra vez.*

**A MATRIZ DO LINUX, DITA COMO E'**: `SOURCE_READY=SIM` | **`BUILD_DESTE_HEAD=NAO`** |
`PACKAGE_CREATED=SIM` | **`INSTALLED=NAO`** | **`DEVICE_E2E=NAO`** |
`SIGNED_FOR_PUBLIC_RELEASE=NAO` | `STORE_PUBLISHED=NAO`.

**O ESTADO DAS PLATAFORMAS, SEM CONFUSAO**: Windows/amd64 -> portatil **e** instalador
(`E-1057`/`E-1058`), **nenhum assinado**, **nenhum instalado**. Linux/amd64 -> tarball
**instalavel** (`E-1059`), binario de `9f7379502b1d`. macOS, Android, iOS e Web/PWA ->
**por produzir**; macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um tarball nao e' um tarball instalavel; o caminho que exigia a
ferramenta nao era o unico caminho; e um artefacto que declara os seus limites vale mais
do que um que os omite.

### Instrucoes de retomada (janela `E-1060`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1060`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (dez 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O BUILD WEB/PWA** (`E-1060`):
  cd app/ui/app
  npm run build:check     # tsc -b && vite build + check-bundle-budget.mjs
  => ✓ built in 15.68s
  => Bundle entry: index-CWgSFuKK.js raw=1509050 gzip=412527
  => Budgets: raw<=2000000 gzip<=600000  =>  BUNDLE_BUDGET=PASS   (exit 0 em 47.8 s)

**O DEFEITO QUE O BUILD NAO APANHOU -- E QUE JA' ESTAVA NO HEAD**:
  o `public/manifest.json` apontava para `/icons/hades-icon-192.png` (e 512, maskable)
  e **o directorio `icons/` NUNCA EXISTIU** -- os PNGs estao na RAIZ de `public/`.
  `git show HEAD:app/ui/app/public/manifest.json` confirma: ja' era assim no HEAD.
  => 0 de 3 icones resolviam. Curado para `/hades-icon-*.png` => **3 de 3**.
  *Um caminho escrito a mao e' uma promessa; um caminho medido e' um facto.*

**O QUE FALTA NO PWA, MEDIDO**: *`ocorrencias de serviceWorker em src/: **0`*** =>
`dist/sw.js` e `dist/service-worker.js` **ausentes** => ***sem service worker nao ha' cache,
nao ha' offline e nao ha' instalacao PWA completa*** => **`PWA_INSTALAVEL=PARCIAL`**.
O `offline` e' **`NAO_IMPLEMENTADO`** -- *nao e' «offline honesto»*.

**O ESTADO DAS PLATAFORMAS, SEM CONFUSAO**: Windows/amd64 -> portatil **e** instalador
(`E-1057`/`E-1058`), **nenhum assinado**, **nenhum instalado**. Linux/amd64 -> tarball
**instalavel** (`E-1059`), binario de `9f7379502b1d`. **Web/PWA** -> build **PASS**, icones
**corrigidos**, **sem service worker**. macOS/Android/iOS -> **por produzir**;
macOS/iOS `BLOCKED_EXTERNAL` pela **maquina**.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um caminho escrito a mao e' uma promessa; o `git show HEAD:`
separa defeito de regressao; e um build que passa o orcamento mede o peso, nao a
instalabilidade.

### Instrucoes de retomada (janela `E-1061`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1061`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (onze 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O SERVICE WORKER** (`E-1061`) -- 3 ficheiros novos:
  app/ui/app/public/sw.js                          7 585 bytes
  app/ui/app/src/lib/registarServiceWorker.ts      registo, SO' em PROD
  app/ui/app/src/lib/registarServiceWorker.test.ts 8 testes
  app/ui/app/src/main.tsx                          +import +chamada (depois do render)

**A REGRA QUE O `sw.js` IMPOE A SI PROPRIO**: `/api/*` e' **network-only, sem excepcao**.
  - `/api/chat` do cache = resposta de outro momento como se fosse agora;
  - `/api/approvals` do cache = **um aprovador que ninguem autorizou**;
  - POST/PUT/DELETE do cache = accao que **nao aconteceu** e o utilizador julga que sim.
  => *Offline, o HADES mostra o shell e diz que esta' offline -- nao inventa um servidor.*

**COMO TESTAR E PROVAR QUE O TESTE DISCRIMINA**:
  cd app/ui/app && npx vitest run src/lib/registarServiceWorker.test.ts   => 8 passed
  python -X utf8 _tmp/contraprova_sw.py                                   => CONTRAPROVA OK
  *A contraprova injeta 3 defeitos (guarda de /api/, guarda de metodo, caches.delete) e
  exige que o teste fique VERMELHO nos tres. Se ficasse verde, o teste era decoracao.*

**SUITE COMPLETA DO APP**: `npx vitest run` => **111 ficheiros, 738 testes, 0 falhas**
(7,13 s).

**O QUE FALTA, MEDIDO**: `INSTALADO_EM_BROWSER=NAO` e `E2E_EM_BROWSER_REAL=NAO` -- o SW
nunca correu num browser; `OFFLINE=PARCIAL` (shell sim, dados nao); sem teste de upgrade
entre versoes reais; sem medicao de memoria/CPU/bateria do SW.

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`),
binario de `9f7379502b1d`. **Web/PWA** -> build PASS (`E-1060`), icones 3/3 (`E-1060`),
**service worker** (`E-1061`), mas **sem E2E num browser**. macOS/Android/iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um cache que responde por uma aprovacao e' um aprovador a mais;
a contraprova e' o que separa um teste de uma decoracao; e offline e' um extra -- abrir e'
o requisito.

### Instrucoes de retomada (janela `E-1062`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1062`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (doze 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O E2E DO PWA NUM CHROMIUM REAL** (`E-1062`) -- 8/8:
  cd app/ui/app
  node node_modules/vite/bin/vite.js preview --port 5173 --host 127.0.0.1   (processo separado)
  node e2e/pwa-serviceworker-real.mjs
  => (1) app abre 200
  => (2) service worker ACTIVO  scope=http://127.0.0.1:5173/  script=.../sw.js
  => (3) manifest  name=Hades  display=standalone
  => (4) 3 icones -> 200 200 200
  => (5) /api/* NAO servido do cache  (status=404, tipo=basic -> chegou a' rede)
  => (6) cache do shell existe e contem /index.html  (hades-sw-v1-shell, 2 caches)
  => (7) offline: 504 HONESTO  ({"status":504,"tipo":"basic"}) -- nao inventa conteudo
  => (8) 0 pedidos falhados inesperados

**DUAS ARMADILHAS MEDIDAS NESTE CAMINHO**:
  1. **`Start-Job` morre com o `pwsh`**. O preview tem de ser **processo separado**
     (`[System.Diagnostics.Process]::Start`), senao a chamada seguinte recebe
     `ERR_CONNECTION_REFUSED`.
  2. **o import `playwright` resolve a partir do FICHEIRO**. Um script em `_tmp/` da'
     `ERR_MODULE_NOT_FOUND`. Tem de viver **dentro** de `app/ui/app/` -> `e2e/`.

**E DUAS FALHAS INICIAIS ERAM DO MEU TESTE** (6/8): o SW **estava** registado (o detalhe
dizia `scope=…` e `script=…`) e eu li o `state` antes de `activated`; e contei como falha
um pedido que **o proprio teste** provocou. Curado: espera pelo `activated` com limite, e
filtra os pedidos do teste. *Medir cedo demais e' uma forma de inventar um defeito.*

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build PASS (`E-1060`), icones 3/3 e SW (`E-1061`), **E2E Chromium 8/8**
(`E-1062`). macOS/Android/iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: medir cedo demais e' uma forma de inventar um defeito; um teste
que nao constroi o mundo que mede acusa o produto do seu proprio defeito; e um cache que
responde por uma aprovacao e' um aprovador a mais.

### Instrucoes de retomada (janela `E-1063`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1063`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (treze 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O E2E DO PWA NOS TRES MOTORES** (`E-1063`):
  npx playwright install firefox webkit     # Firefox 157.0 (v1555) + WebKit 27.2 (v2370)
  node node_modules/vite/bin/vite.js preview --port 5173 --host 127.0.0.1
  $env:HADES_E2E_BROWSER='chromium'; node e2e/pwa-serviceworker-real.mjs   # 8/8
  $env:HADES_E2E_BROWSER='firefox';  node e2e/pwa-serviceworker-real.mjs   # 8/8
  $env:HADES_E2E_BROWSER='webkit';   node e2e/pwa-serviceworker-real.mjs   # 8/8

**A DIFERENCA QUE PROVA QUE O TESTE MEDE O MOTOR**: offline, Chromium devolve **504
basic**; **Firefox lanca `TypeError: NetworkError when attempting to fetch resource`**;
WebKit devolve **504 basic**. O teste aceita os dois porque **ambos sao honestos**: nenhum
inventa conteudo. *Se medisse a si mesmo, os tres dariam o mesmo resultado.*

**O QUE MUDOU NO TESTE**: `serviceWorkers: "allow"` **so' existe no Chromium** -- nos
outros motores o default ja' permite, e a opcao **nao** e' passada. Um motor desconhecido
sai com **exit 2**, nao corre em silencio com o errado.

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build PASS (`E-1060`), icones 3/3 e SW (`E-1061`), **E2E 8/8 em Chromium,
Firefox e WebKit** (`E-1062`/`E-1063`), mas **`PWA_INSTALADO_NA_UI=NAO`**. macOS/Android/
iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: duas ferramentas instalaveis nao sao dois bloqueios externos;
se o teste medisse a si mesmo os tres motores dariam o mesmo resultado; e um motor nao e'
um dispositivo.

### Instrucoes de retomada (janela `E-1064`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1064`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (quatorze 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**OS RECURSOS E O UPGRADE DO SW** (`E-1064`) -- `app/ui/app/e2e/pwa-recursos-e-upgrade.mjs`:
  node e2e/pwa-recursos-e-upgrade.mjs      # abre servidor proprio na porta 5180
  => v1 activo; caches hades-sw-v1-shell(3) + hades-sw-v1-assets(6)
  => 9 entradas, 160.6 KB, 0.16 MB      (orcamento declarado ANTES: 400 / 64 MB)
  => activo em 448 ms                    (orcamento declarado ANTES: 20 s)
  => upgrade: browser aceitou a v2 e activou-a  (caches = hades-sw-v2-*)
  => upgrade: a v1 FOI APAGADA            (nenhum cache hades-sw-v1-*)
  => upgrade: /api/* continua a ir a rede  ({"status":404})
  ==== 9/9 ====

**COMO O UPGRADE E' TESTADO SEM INVENTAR NADA**: o script serve o **mesmo `dist/`** por um
servidor local que, **so' no caminho `/sw.js`**, reescreve `const VERSAO` para
`hades-sw-v2`. O corpo do SW e' o mesmo; so' a constante muda. O codigo de limpeza do
`activate` ja' existia -- o `E-1061` testou-o em simulacao; **aqui e' um browser.**

**O QUE FALTA, MEDIDO**: recursos e upgrade **so' em Chromium**; Firefox/WebKit ficaram
no `E-1063` com os 8 pontos mas sem esta medicao; sem CPU nem bateria;
`PWA_INSTALADO_NA_UI=NAO`.

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build PASS (`E-1060`), icones 3/3 e SW (`E-1061`), **E2E 8/8 em 3 motores**
(`E-1062`/`E-1063`), **recursos + upgrade 9/9 em Chromium** (`E-1064`), mas
**`PWA_INSTALADO_NA_UI=NAO`**. macOS/Android/iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um limite que se ajusta ao resultado nao e' um limite; um
service worker que nunca e' substituido congela o produto no dia em que e' instalado; e o
cache que sobrevive a' sua propria versao e' um produto que nao consegue corrigir-se.

### Instrucoes de retomada (janela `E-1065`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1065`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (quinze 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**RECURSOS E UPGRADE DO SW NOS TRES MOTORES** (`E-1065`):
  $env:HADES_E2E_BROWSER='chromium'; node e2e/pwa-recursos-e-upgrade.mjs   # 9/9  423 ms
  $env:HADES_E2E_BROWSER='firefox';  node e2e/pwa-recursos-e-upgrade.mjs   # 9/9 1646 ms
  $env:HADES_E2E_BROWSER='webkit';   node e2e/pwa-recursos-e-upgrade.mjs   # 9/9 1041 ms

**A LATENCIA DIFERE, E ISSO PROVA QUE O INSTRUMENTO MEDE**: 423 / 1041 / 1646 ms, mesmo
comando, mesma maquina, mesmo `dist/`, mesmo orcamento de 20 s. *Se o instrumento se medisse a
si mesmo, os tres dariam o mesmo numero.*

**E O FIREFOX E' 3,9x MAIS LENTO A ACTIVAR -- MEDIDO, NAO EXPLICADO**: o proximo passo e'
instrumentar os marcos (`install`, `waiting`, `activate`) com `performance.now()` no `sw.js` e
comparar os tres. *Registar a diferenca e' medir; explica-la sem medir seria inventar.*

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build PASS (`E-1060`), SW (`E-1061`), **8 pontos em 3 motores** (`E-1062`/
`E-1063`), **recursos + upgrade 9/9 nos 3 motores** (`E-1064`/`E-1065`), mas
**`PWA_INSTALADO_NA_UI=NAO`**. macOS/Android/iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: tres numeros diferentes sao a prova de que o cronometro nao esta' a
medir o cronometro; e um numero nao e' uma explicacao.

### Instrucoes de retomada (janela `E-1066`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1066`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (dezesseis 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O PERFIL DO install/activate DO SW** (`E-1066`) -- `app/ui/app/e2e/pwa-onde-esta-o-tempo.mjs`:
  $env:HADES_E2E_BROWSER='chromium'; node e2e/pwa-onde-esta-o-tempo.mjs
  => install=18.2 ms  activate=0.1 ms  fora=1.0 ms  pagina=73.2 ms
  $env:HADES_E2E_BROWSER='firefox';  node e2e/pwa-onde-esta-o-tempo.mjs
  => install=132.0 ms activate=0.0 ms  fora=3.0 ms  pagina=262.0 ms
  $env:HADES_E2E_BROWSER='webkit';   node e2e/pwa-onde-esta-o-tempo.mjs
  => install=72.0 ms  activate=0.0 ms  fora=12.0 ms pagina=308.0 ms
  => H1 ACEITE nos tres: o custo e' o addAll. O activate custa 0 ms.

**COMO FOI DISCRIMINADO SEM TOCAR NO PRODUTO**: o servidor de teste serve o `dist/` e **injeta**
uma versao instrumentada **so' no `/sw.js`** -- `performance.now()` no inicio/fim do `install` e
do `activate`, com `postMessage` para a pagina. ***O `sw.js` do produto continua sem
telemetria.*** *Um service worker de producao nao leva cronometros dentro.*

**E A CORRECCAO DA MEDICAO ANTERIOR**: o `E-1065` mediu Firefox **1646 ms**; este mede **262 ms**
no mesmo motor e maquina. A diferenca e' o que cada um cronometra: o `E-1065` desde **abrir a
pagina**, este desde `navigator.serviceWorker.ready`. ***O numero grande era o browser a arrancar,
nao o service worker.*** *O cronometro estava certo; a etiqueta no cronometro e' que estava
errada.*

**NAO OPTIMIZAR**: `install` 132 ms vs orcamento de 20 s = **~150x de folga**. Trocar o `addAll`
atomico para ganhar 100 ms poria em risco a garantia do `E-1061`.

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build (`E-1060`), SW (`E-1061`), **8 pontos em 3 motores** (`E-1062`/`E-1063`),
**recursos+upgrade 9/9 em 3 motores** (`E-1064`/`E-1065`), **perfil install/activate**
(`E-1066`), mas **`PWA_INSTALADO_NA_UI=NAO`**. macOS/Android/iOS -> por produzir; **Android e' a
proxima viavel**.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um numero nao e' uma explicacao; um numero certo com o nome errado e'
uma conclusao errada; e nao se optimiza o que ja' esta' 150x dentro do orcamento.

### Instrucoes de retomada (janela `E-1067`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1067`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (dezesete 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O ANDROID, MEDIDO** (`E-1067`) -- tudo esta' la', menos o Gradle a arrancar:
  apps/mobile-agentic/android/      nativo completo (gradlew, gradlew.bat, build.gradle, ...)
  _tmp/android-sdk                  394.2 MB  (android-34, build-tools 34.0.0, adb)
  _tmp/jdk17                        openjdk 17.0.20.1 (Temurin)
  local.properties                  sdk.dir=<raiz>/_tmp/android-sdk   (ja' aponta para o SDK)
  ~/.gradle/.../gradle-8.13         ja' extraido

**AS 3 TENTATIVAS, E A CAUSA NOMEADA**:
  1. gradlew.bat assembleDebug  -> FileNotFoundException no gradle-8.13-bin.zip (o zip foi apagado)
  2. gradle.bat assembleDebug   -> Failed to load native library 'native-platform.dll'
     (-Dorg.gradle.native=false e --no-daemon NAO ajudam)
  3. -Djava.io.tmpdir explicito -> mesmo erro; --stacktrace deu a causa:
     NativeLibraryLocator.java:46 -> File.createNewFile -> createFileExclusively
     -> 'O sistema nao pode encontrar o caminho especificado'

**ISTO NAO E' BLOCKED_EXTERNAL**: falta o Gradle a extrair uma DLL, nao uma credencial. O passo
de desbloqueio e' concreto: (1) ver o `java.io.tmpdir` que o processo java ve;
(2) criar esse directorio; (3) extrair a `native-platform.dll` do jar a' mao e passar
`-Dorg.gradle.native.dir`.

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build, SW, 3 motores, recursos+upgrade, perfil (`E-1060`..`E-1066`), mas
`PWA_INSTALADO_NA_UI=NAO`. **Android** -> projecto completo + toolchain presente, **Gradle
bloqueado na DLL** (`E-1067`). macOS/iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um bloqueio de ambiente nao se disfarca de bloqueio externo -- e
tambem nao se esconde atras de «quase».

### Instrucoes de retomada (janela `E-1068`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1068`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (dezoito 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**A MARCA ANDROID MIGRADA** (`E-1068`):
  app.json      name=HADES  slug=hades  scheme=hades
                android.package=com.dz23ltd.hades  ios.bundleIdentifier=com.dz23ltd.hades
  build.gradle  namespace + applicationId = com.dz23ltd.hades   (mas android/ e' GERADO)
  package.json  name=hades-mobile
  App.tsx       '<Text>DZ23 AGENTIC</Text>' -> '<Text>HADES</Text>'
                as chaves dz23.agent.* FICAM (4 + 2 ocorrencias)
  README.md     '# DZ23 Agentic Mobile' -> '# HADES Mobile'
  MIGRACAO_MARCA.md   2917 B  (o plano, com o que acontece a quem tem a versao antiga)

**O `android/` NAO SE COMMITA**: e' saida do `expo prebuild` (35 ficheiros nao rastreados), e
o `app.json` e' a **fonte** -- o `build.gradle` deriva do `app.json.android.package`.
A migracao correcta e' no `app.json`, e esta' feita.

**AS TRES CATEGORIAS, QUE NAO SAO A MESMA COISA**: UI visivel muda; package muda; **chaves de
storage ficam**. *Uma migracao de marca que apaga a sessao de quem ja' a usava nao e' uma
migracao: e' uma perda.*

**E A MITIGACAO DO `applicationId`**: no Android ele **e' a identidade da app**, logo a versao
nova e' **outra app**. A mitigacao honesta: **nao existe APK publicado com o id antigo**
(`E-1067`), logo **nao ha' utilizadores a quebrar**. Se um dia houver:
`MigrateAndroidApplicationId` (AGP 8+) ou `MY_PACKAGE_REPLACED`.

**QUATRO FICHEIROS QUE EU NAO TOQUEI APARECERAM COMO `M`**: `missionContext*` (4) =>
`git diff --ignore-cr-at-eol --numstat` deu **vazio** => **so' fim-de-linha**. O diff real da
migracao e' **4 ficheiros, 9 insercoes, 9 remocoes**.

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build, SW, 3 motores, recursos+upgrade, perfil (`E-1060`..`E-1066`), mas
`PWA_INSTALADO_NA_UI=NAO`. **Android** -> projecto completo, toolchain presente, marca
**migrada** (`E-1068`), mas **Gradle bloqueado na DLL** (`E-1067`) e **sem APK**. macOS/iOS
-> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: uma migracao de marca que apaga a sessao nao e' uma migracao; uma
marca muda-se num sitio e uma sessao perde-se em qualquer um; e uma migracao que nunca
compilou esta pronta a compilar, nao esta verificada.

### Instrucoes de retomada (janela `E-1069`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1069`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (dezenove 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**AS 231 OCORRENCIAS CLASSIFICADAS NO SITIO** (`E-1069`):
  importlib -> script/hades_brand_inventory.py -> inventariar()
  para cada ocorrencia: 50 chars de contexto em cada lado
  contra 10 padroes tecnicos declarados ANTES
  => MARCA (a migrar):  **19**  -- 3 MIGRACAO_MARCA.md, 14 HADES_EVIDENCE_INDEX.md, 1+1 handoff
  => TECNICO (excecao): **~250** -- DZ23-LTDA (owner), dz23-agentic-quality (workflow CI),
                             bundle id, chave de storage, nomes historicos de produto

**AS 19 «MARCA» SAO TODAS HISTORICO**: o `MIGRACAO_MARCA.md` cita os nomes antigos **de
proposito** (e' uma tabela antes/depois); o indice de evidencias cita **o que foi medido** (e o
§FASE-7 proibe reescreve-lo); e os outros dois sao documentos que eu escrevi a citar o passado.
**Nenhuma e' texto de interface de utilizador.**

**A CORRECCAO DA MINHA HIPOTESE**: eu esperava um erro de **contagem**; o erro estava na
**granularidade da categoria**. *Um ficheiro migravel nao faz das suas ocorrencias marca
visivel.* *Quem procura um erro no numero pode nao ver o erro na pergunta.*

**A MARCA, FECHADA**: o **§FASE-5 item 2** (migrar a superficie visivel) esta' cumprido no
mobile (`E-1068`) e **nao tem mais nada para migrar**; o item 3 (manter e dizer por que) e' o
que resta, e o inventario ja' o diz para as **8 excecoes**.

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build, SW, 3 motores, recursos+upgrade, perfil (`E-1060`..`E-1066`), mas
`PWA_INSTALADO_NA_UI=NAO`. **Android** -> projecto completo, toolchain presente, marca
migrada, **Gradle bloqueado na DLL** (`E-1067`), **sem APK**. macOS/iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um ficheiro migravel nao faz das suas ocorrencias marca visivel; e
quem procura um erro no numero pode nao ver o erro na pergunta.

### Instrucoes de retomada (janela `E-1072`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1072`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (vinte 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O GRADLE, MEDIDO** (`E-1072`):
  jar: native-platform-windows-amd64-0.22-milestone-28.jar (69.5 KB)
       -> net/rubygrapefruit/platform/windows-amd64/native-platform.dll (141 312 B)
  extraido para _tmp/gradle-native  (com o jar.exe do JDK 17)
  gradle --version --no-daemon -Dorg.gradle.native.dir=_tmp/gradle-native
  => exit 0: Gradle 8.13, Launcher JVM 17.0.20.1, OS Windows 11

**O DAEMON NAO HERDA -- 6 hipoteses testadas**:
  1. -Dorg.gradle.native.dir no launcher        -> --version SIM, daemon NAO
  2. -Dorg.gradle.native=false                  -> nao ajuda (o daemon PRECISA das native services)
  3. -Djava.io.tmpdir curto e existente (C:\gt) -> nao ajuda (e o tmpdir existe e e' gravavel)
  4. GRADLE_OPTS / JAVA_OPTS                    -> nao chegam ao daemon
  5. layout alternativo no native.dir           -> mesmo erro
  6. org.gradle.jvmargs no gradle.properties    -> mesmo erro

**A PISTA QUE O LOG DO DAEMON DEU**: ~/.gradle/daemon/8.13/daemon-3984.out.log diz que em
**2026-10-16T06:42:43** o daemon **executou uma build** e falhou com
***GeneratedClassCompilationException: No Java compiler found*** -- **nao** com a DLL.
=> *o daemon JA carregou a DLL nesse dia; o que mudou foi a JVM que ele ve.*
=> ***Um log antigo que ninguem le e' uma pista que ninguem usa.***

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build, SW, 3 motores, recursos+upgrade, perfil (`E-1060`..`E-1066`), mas
`PWA_INSTALADO_NA_UI=NAO`. **Android** -> projecto completo, toolchain presente, marca
migrada, **Gradle arranca mas o daemon nao carrega a DLL** (`E-1072`), **sem APK**.
macOS/iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.

**Doutrina em uma linha**: um `--version` que passa nao e' um build que passa -- o primeiro
passo nao e' o caminho, e' a prova de que o caminho existe; e um log antigo que ninguem le e'
uma pista que ninguem usa.

### Instrucoes de retomada (janela `E-1073`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1073`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (vinte e um 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O GRADLE, FECHADO COM `BLOCKED`** (`E-1073`):
  12 tentativas, 8 hipoteses, todas com a mesma linha:
    Failed to load native library 'native-platform.dll' for Windows 11 amd64
  O launcher carrega (com -Dorg.gradle.native.dir); o daemon nao.
  As duas causas candidatas foram ELIMINADAS por medicao:
    - java.io.tmpdir efectivo = C:\Users\DZ23\AppData\Local\Temp\ (o Java ignora TMP/TEMP)
    - ~/.gradle/native/ nao existia; criado e gravavel; falhou igual
  E a prova de que o daemon JA carregou: daemon-3984.out.log (10-16 06:42:43) tem
    [DEBUG] [NativeServices] Native-plat...   <- inicializou
  e a falha de entao era GeneratedClassCompilationException: No Java compiler found.
  => O desbloqueio e' DECISAO DO DONO: outro Gradle (8.14+/9.x) ou outro host/runner.

**PELO PROTOCOLO ANTILOOP, ITEM 5** -- *tres tentativas na mesma causa => BLOCKED e avancar*
  => 12 tentativas e' quatro vezes o limite => ***BLOCKED registado, e mudanca de trabalho.***
  => *Repetir com variacao nao e' investigar -- e' insistir com disfarce.*

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build, SW, 3 motores, recursos+upgrade, perfil (`E-1060`..`E-1066`), mas
`PWA_INSTALADO_NA_UI=NAO`. **Android** -> projecto completo, toolchain presente, marca
migrada, **`BLOCKED` no daemon do Gradle** (`E-1073`), **sem APK**. macOS/iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.
**E nao voltar ao Gradle sem hipotese nova**: 8 ja' foram testadas.

**Doutrina em uma linha**: repetir com variacao nao e' investigar -- e' insistir com disfarce;
e o que ja' funcionou uma vez neste host nao esta' provado impossivel -- esta' provado instavel.

### Instrucoes de retomada (janela `E-1074`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1074`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (vinte e dois 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**OS 74 ORFAOS, CLASSIFICADOS** (`E-1074`):
  medicao: varredura do repo por E-(\d{3,}) vs as 866 entradas do indice
  => 77 citados sem entrada => 3 falsos positivos => 74 orfaos reais
  faixa E-571..E-777 = 207 = 74 citados + 133 nao citados   (o prompt dizia 73+134)
  DENTRO da faixa: 74   |   FORA da faixa: 0
  em codigo: 63  (88 ficheiros)   |   so em documentos: 11
  lista completa: _tmp/orfaos_tabela.md (74 linhas)

**OS 3 FALSOS POSITIVOS, COM PROVA**:
  - E-000   -> sentinela/placeholder (o E-779 diz que E-999 e E-000 sao valores de exemplo)
  - o ANO de CVE-2026-85180 em x/transfer/redirect_test.go
  - a CABECA DE TESTE do proprio hades_index_shape_check.py

**A IRONIA DO E-1074**: ao CITAR os dois falsos positivos no texto, criei DOIS NOVOS -- porque
o CITACAO e' \bE-(\d{3,})\b e o texto tinha a forma exacta de uma citacao.
=> *O instrumento que mede o defeito participa do defeito.*
=> Curado com isencao MINIMA e provada, e com um teste que prova que ela e' minima:
   um ID vizinho (outro ano) e um ID abaixo (outra cabeca) CONTINUAM a acusar.
   hades_index_shape_check_test.py: 12 OK, 0 FAIL.

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build, SW, 3 motores, recursos+upgrade, perfil (`E-1060`..`E-1066`), mas
`PWA_INSTALADO_NA_UI=NAO`. **Android** -> projecto completo, toolchain presente, marca
migrada, **`BLOCKED` no daemon do Gradle** (`E-1073`), **sem APK**. macOS/iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.
**E nao voltar ao Gradle sem hipotese nova**: 8 ja' foram testadas.

**Doutrina em uma linha**: copiar um numero e' mais rapido do que medi-lo -- e e' assim que um
numero errado entra num relatorio; e o instrumento que mede o defeito participa do defeito.

### A cura do `E-1075` (a mesma janela)

**O `E-1074` expôs um defeito de paridade**: o `hades_handoff_window_check.py` **não** tinha o
limite inferior que o `hades_index_shape_check.py` já tinha ⇒ o `E-000` (que vira `0` no `int()`)
entrava como *«citado sem entrada»* em **qualquer** documento que o mencionasse.

**Cura**: a **mesma** regra (`_abaixo = min(registados)` e `c >= _abaixo`). ⇒ ***«Dois checkers, a
mesma regra, e um deles ficou para trás.»***

**Resultado**: `JANELA DE CITACAO - fault injection: 20 OK, 0 FAIL` (era 18/2), e a bateria voltou a
**33 OK / 0 FAIL**. ⇒ ***«O documento que documenta o falso positivo é o teste que o checker não
esperava.»***

### Instrucoes de retomada (janela `E-1076`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1076`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (vinte e tres 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**O PACOTE DE AUDITORIA** (`E-1076`, §FASE-7 item 8) -- `docs/audit/`:
  README_AUDITOR.md       <- seccoes, hashes, e «o que voce NAO vai encontrar»
  PARIDADE_EVIDENCIAS.csv <- as 869 entradas do indice
  GATES.md                <- os 36 gates declarados (LISTA, nao executa)
  LIMITACOES.md           <- 17 blockers (8 ABERTOS / 9 resolvidos pelo campo tipo),
                             faixa 571..777, artefactos ausentes, matriz por plataforma
  ARTEFATOS.sha256        <- SHA-256 dos artefactos que EXISTEM
  gerador: scripts/hades_audit_package.py  (com --check)
  teste:   scripts/hades_audit_package_test.py  (24 verificacoes, 4 defeitos injetados)

**COMO REGERAR**: python -X utf8 scripts/hades_audit_package.py  (e depois --check)

**O DEFEITO QUE O PROPRIO GERADOR TINHA**: a primeira versao separava resolvidos de abertos
por `"RESOLVIDO" in descricao.upper()` => e MEDIU MAL (5 abertos / 12 resolvidos quando sao
8 / 9) => *e' o E-83: um detector lexical ve termos, nao relacoes* => a cura foi usar o
**campo `tipo`**, e o LIMITACOES.md passou a mostrar o tipo em cada linha.

**O TESTE TAMBEM TEVE DE SER CURADO** (pela terceira vez): a copia descartavel nao tinha os
artefactos declarados, logo gerava um ARTEFATOS.sha256 diferente e falhava POR CONSTRUCAO
=> curado importando a lista ARTEFATOS_DECLARADOS do proprio gerador.
=> *E-1071 e E-1044/E-1054 outra vez: uma copia sem o mundo que o mede acusa o medidor.*

**E A CATRACA DE ORFAOS APANHOU-ME PELA QUARTA VEZ** (E-1055, E-1056, E-1071, agora):
criei os dois instrumentos e nao os pus na bateria => curado com os blocos 7d e 7e.

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build, SW, 3 motores, recursos+upgrade, perfil (`E-1060`..`E-1066`), mas
`PWA_INSTALADO_NA_UI=NAO`. **Android** -> projecto completo, toolchain presente, marca
migrada, **`BLOCKED` no daemon do Gradle** (`E-1073`), **sem APK**. macOS/iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.
**E nao voltar ao Gradle sem hipotese nova**: 8 ja' foram testadas.

**Doutrina em uma linha**: um pacote que so' lista o que existe e' uma brochura -- um que diz o
que falta e' um instrumento; e um detector lexical ve termos, nao relacoes.

### Instrucoes de retomada (janela `E-1077`)

**Estado**: branch `fix/checker-relogio`; cabeca do indice `E-1077`; PR #438 **aberto** e
**nao tocado** -- *as 14 threads continuam como estavam*; `reviewDecision` **vazio**.

**REGRA DE PUSH -- OBRIGATORIA, E JA' VALIDADA** (vinte e quatro 6/6 consecutivos):
  1. commit local -> 2. ESPERAR o SHA remoto fechar com `failure=0` -> 3. `git push`.

**OS FINDINGS DO PR** (`E-1077`, §FASE-7 item 8 ultima parte) -- `docs/audit/`:
  FINDINGS.json  <- capturado da API em 2026-10-17T06:03:40Z sobre o checkout 093390737
                    (id, autor, path, linha, data, severidade, categoria, titulo, trecho)
  FINDINGS.md    <- gerado DO JSON (o --check NAO precisa de rede)
  gerador: scripts/hades_findings_report.py
    --capturar  -> le' a API do GitHub (precisa de rede e de gh)
    (sem arg)   -> gera o MD do JSON
    --check     -> compara o MD com o JSON (SEM rede)

**O QUE OS 14 SAO**: 5 Major + 9 Minor, todos de coderabbitai[bot].
  Major: hades_test_clock_check.py x3, service_agent_approvals.go:879, router_pin_v2_test.go
  E BATEM COM O PROMPT: #2 (item 2), #4 (item 3), #5 (item 4), #6 (o P0, item 1),
  #14 (item 8), #13 (item 9).

**O FINDINGS.md DIZ O QUE NAO AFIRMA** (e isso e' o que importa):
  - nao afirma que algum finding foi corrigido (nenhum foi marcado no GitHub -- proibido);
  - nao afirma que o bot esta' certo (comentario de bot e' hipotese, nao facto);
  - nao afirma que a lista esta' completa (sao os review comments daquele instante);
  - e diz COMO validar: o E-1070 e o E-1074 sao o modelo (reproduzir num checkout limpo).

**OS TRES DEFEITOS DO GERADOR, MEDIDOS**: (1) `Total: 14 — .` com a lista vazia, porque o
regex procurava "**Major**" e o corpo tem "**🟠 Major**" -- o emoji fica DENTRO do negrito;
(2) o titulo saiu como "🟠 Major" porque apanhava negritos[1]; o titulo e' o QUARTO (depois do
esforco); (3) o ID injectado do DEFEITO 3 foi construido em runtime. => UM INDICE POR POSICAO FIXA
QUEBRA QUANDO A DECORACAO MUDA.

**E O TESTE DO PACOTE TEVE DE SER CURADO OUTRA VEZ**: o audit_package --check passou a exigir
FINDINGS.md e FINDINGS.json, e a copia do teste nao os tinha => 4 falhas por construcao.
Curado. E o E-1071/E-1044/E-1054 pela quarta vez: uma copia sem o mundo que o mede acusa o
medidor do seu proprio defeito.

**O ESTADO DAS PLATAFORMAS**: Windows/amd64 -> portatil **e** instalador (`E-1057`/
`E-1058`), nao assinados, nao instalados. Linux/amd64 -> tarball **instalavel** (`E-1059`).
**Web/PWA** -> build, SW, 3 motores, recursos+upgrade, perfil (`E-1060`..`E-1066`), mas
`PWA_INSTALADO_NA_UI=NAO`. **Android** -> projecto completo, toolchain presente, marca
migrada, **`BLOCKED` no daemon do Gradle** (`E-1073`), **sem APK**. macOS/iOS -> por produzir.

**O que NAO fazer**: `reset --hard`, `git clean`, rebase destrutivo, force-push, stash,
`merge`, tag, release, publicacao externa, ou **resolver threads** do PR #438.
**E nao voltar ao Gradle sem hipotese nova**: 8 ja' foram testadas.

**Doutrina em uma linha**: um `--check` que precisa de rede nao e' um `--check` -- o JSON e' a
prova do que a API disse naquele dia, e o MD e' a sua leitura; e um indice por posicao fixa
quebra quando a decoracao muda.
