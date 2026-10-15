# HADES — TRABALHO PENDENTE PRIORIZADO

## 0. ESTADO MEDIDO — `fix/checker-relogio` `cafb00dbe` (esta é a leitura canônica; não herde número de texto antigo)

> **BLOQUEIO MEDIDO (2026-10-15, item 13 do complemento 48 — teste de browser).** O browser mode do `vitest 3.2.7` **está quase montado**: existem `@vitest/browser`, `playwright`, `playwright-core`, `@playwright/test`, `react-dom` e **Chromium instalado** (`ms-playwright/chromium-1228,1243,1248`). **Faltam exactamente duas peças** — `@vitest/browser-playwright` (*o provider que o Vitest ≥3.2 exige*) e `vitest-browser-react` (*renderizador React para browser mode*); `vitest.config.ts` só tem `environment: "node"`. ⇒ **`BLOCKED_BY_EXTERNAL_DEPENDENCY`** com a peça nomeada. **DECISÃO TOMADA:** *não* adicionei as dependências ao `package.json` — fazê-lo **sem regravar o lockfile quebraria o `--frozen-lockfile` do CI**, trocando um teste em falta por um CI vermelho. **Comando correcto (operador, com rede):** `pnpm add -D @vitest/browser-playwright vitest-browser-react` **+ commit do lockfile no mesmo commit**. *Todo o resto da cadeia da sala está `TESTED`; falta só a fiação React de 3 linhas.*

> **FRENTES 45/46/47 (prioridade 7 da FASE 6) — MEDIDAS (2026-10-15), e uma correcção minha.** Eu tinha escrito «prioridades 7 ainda intocadas»: **a medição diz que não.** *Companion* tem **domínio + testes + UI**: `internal/agent/companion_estado.go` (conjunto **fechado** de 8 estados; `DerivarEstadoCompanion` devolve **estado E razão**; `CompanionPodeDizer(estado, frase)` é a guarda contra o mascote afirmar o que não pode provar; invariante **«sem missão não há estado honesto»** ⇒ `ErrCompanionMissaoAusente`) + `companion_estado_test.go` (9 casos) + `CompanionMascote.tsx` (174 linhas) com `CompanionMascote.test.tsx` (movimento, anúncio para quem não vê, **ausência declarada**, onboarding). *Studio* tem página real (`StudioCanvasPage.tsx` ≈950 linhas, rota `studio.tsx`, item no `AppSidebar`, aviso extraído para ser testável). **LACUNA MEDIDA:** `DerivarEstadoCompanion` **não aparece em `server/`** ⇒ *o Companion tem domínio e UI mas **não tem rota** que alimente a UI com o estado* — a mesma classe do `EvaluationRunner`, medida por nome. **Artifact Drive — MEDIDO no vocabulário correcto (`Artefato|Biblioteca|Library`), e o meu «0» era o TERCEIRO falso negativo meu:** existe **`LibraryPage.tsx`** + rota `/library` + item «Biblioteca» no `AppSidebar` + entrada na busca (`page-library`) — agrega artefatos de **todas** as missões, com **busca e filtro por categoria**; **`ArtifactsViewer.tsx`** faz download real (`/api/agent/v1/missions/:id/artifacts/:artifactId`) e mostra **`sha256` rotulado «Hash de Proveniência SHA-256»**; **`AgenticSplitShell.tsx`** integra o painel «Artefatos da Missão (n)» (**com teste**: `AgenticSplitShell.test.tsx` exige «Artefatos (1)»); **`MissionDeliveryCard.tsx`** entrega com sha256; **`StudioCanvasPage.security.test.ts`** lê o **próprio fonte** do `ArtifactsViewer` (padrão de teste de segurança sobre o código). ⇒ **UI ✅ + proveniência ✅ + busca ✅ + prévia ✅ + testes ✅**. **NÃO medido:** versionamento *por artefato*, permissões *por artefato* em multi-inquilino e dedupe *real* (o sha256 é o substrato, não a prova do dedupe). **Lição registada: três falsos negativos meus, todos por procurar o nome em inglês num domínio de identificadores pt-BR.** **E2E:** nenhuma das três tem passagem E2E provada.

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

### P1-4 · Harness CLI governado
- PR #62 (`afea7e726`) é **branch-local**: `harness.cli` não está em `main`.
  Próximo: merge e prova de que a aprovação obrigatória (`RequiresApproval:
  true`, `Risk: RiskWrite`) é exigida de fato, com teste reproduzível.

### P1-5 · Catálogo honesto de conectores
- 104/122 exigem operador. Promover conectores **um por um**, com evidência de
  chamada real e mantendo `Auth` não-vazio. A metade "promoção automática de
  status" para linhas só-chave é **inviável** (`BLOCKED_BY_EXTERNAL_DEPENDENCY`).

## P2 — engenharia e infraestrutura

### P2-1 · Workflow Engine
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

### P2-4 · Falhas ambientais de teste (B-09)
- Sanear o host de teste (Python + `sh`, caminho curto) **sem** desativar teste.

### P2-5 · Prettier não governado na raiz
- `.prettierrc` existe só em `app/ui/app`; nenhum workflow roda
  `prettier:check`. Decidir: adotar na raiz ou declarar formalmente fora de
  escopo.

## P3 — release verificado e arquitetura

### P3-1 · Release assinado por SO
- Certificado (B-06), smoke nativo por SO, artefatos assinados publicados.

### P3-2 · Isolamento por organização (multitenancy)
- Decisão arquitetural + teste de que uma organização não lê dados de outra.

### P3-3 · Benchmarks com metodologia auditável
- Substituir o comparativo documental (`audit/COMPARATIVO_TYPINGMIND_20261009.md`,
  `docs/RELATORIO_BENCHMARK_COMPETITIVO_HADES_2026-10-04.txt`) por benchmarks
  reproduzíveis com versão, ambiente e comando.

### P3-4 · Estágios 5, 8, 15 e 34–37
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
