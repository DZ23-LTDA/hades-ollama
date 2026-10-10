# HADES — AUDITORIA MESTRA

**Rodada:** 2026-10-10 (rodadas 1 e 2) · **Base:** `main@889750abe4006985b3bf9037bca2323d1e40cef0`
**Autor:** agente DSH (deepseek-v4.1-flash) · **Escopo:** estágio 1 (auditoria
total) + reconferência dos estágios tocados nesta rodada.

---

## 1. Método

1. **Estado vivo primeiro.** Nada foi assumido do histórico: SHA de `main`, lista
   de PRs abertos e estado de merge foram consultados na API do GitHub no
   momento da coleta (E-001, E-002, E-003).
2. **Probe antes de afirmar.** Cada classificação de capacidade exigiu leitura do
   arquivo e comando reproduzível; onde não houve probe, o estado ficou
   `PENDENTE_DE_AUDITORIA` em vez de "ausente".
3. **Separação de eixos.** Capacidade (o que existe), execução (o que roda),
   release (o que pode ser entregue) são avaliados separadamente — um produto
   pode ter capacidade implementada e não estar pronto para release.
4. **Distinção prova × pista.** Os ~30 documentos datados de `audit/` são
   tratados como pistas; nenhum sustenta promoção de capacidade sem
   reconfirmação por comando.

## 2. O que foi confirmado

- `main` = `889750abe…`, runs recentes de `main` verdes (E-001, E-004).
- **29 PRs abertos**; **somente #54 e #55 mergeados** (E-002, E-003).
- `docs/mission/` **não existia** e foi criado nesta rodada; `audit/` tem 30
  documentos datados (E-021).
- **15 workflows** de CI (E-022); o classificador de superfícies de CI está
  coerente consigo mesmo (13/13 testes, E-017) e decide corretamente o caminho
  docs-only (E-005, E-006).
- **Gates locais verdes** no head de trabalho: Go (`gofmt` limpo, gofumpt,
  golangci-lint `0 issues.`, vet, build, E-014) e Web (lint, 65 arquivos/396
  testes, build de produção, E-013).
- **Jornada de primeira execução** provada por E2E com backend fixture real
  dentro do CI, virando o passo `15 | Web E2E — primeira execução | success` de
  um job cujo nome literal é exigido pelo gate agregador (E-011, E-032).
- **Catálogo de conectores honesto:** 122 linhas, 17 `available`, 104 exigindo
  operador, `SEM_SCOPES=0`, `Auth` sempre não-vazio (E-018, E-019).
- **Política de capacidades** com 16 escopos e recusa por padrão de escopo
  elevado em auto-run (E-026).
- **Rodada 2:** 18 estágios antes `PENDENTE_DE_AUDITORIA` receberam
  classificação a partir de probe real, incluindo a confirmação de que o
  Harness Runtime está ausente em `main` (E-041) e de que o Model Gateway não
  cobre 5 dos provedores exigidos (E-038).

## 3. Correções de premissa (auditoria que se corrige)

### 3.1 Studio: "drag-and-drop ausente" era **falso**

O backlog anterior classificava o Studio como sem reordenação real. O probe
mostra o contrário:

| Evidência | Local |
| --- | --- |
| `draggable` / `onDragStart` / `onDragOver` / `onDrop` / `onDragEnd` | `StudioCanvasPage.tsx:975, 976, 980, 984, 988` |
| Botões de acessibilidade `ArrowUpIcon` / `ArrowDownIcon` | `StudioCanvasPage.tsx:1012, 1024` |
| Persistência `syncComponents` → `updateBuilderVisual` | `StudioCanvasPage.tsx:337-340` |
| Helper puro de reordenação | `lib/studioReorder.ts` |
| 5 testes unitários do helper | `lib/studioReorder.test.ts` |

⇒ A classificação correta é **`IMPLEMENTADO_NAO_HOMOLOGADO`**: existe e tem
teste unitário, mas **não tinha asserção E2E**. O gap passou a ser "falta de
spec Playwright", não "falta de funcionalidade". Esta correção é a prova de que
a regra "probe antes de afirmar" está sendo aplicada de fato.

**Fechamento posterior (2026-10-10):** o gap de E2E foi fechado no PR #73 —
`e2e/studioReorder.spec.ts` prova a persistência do arraste e a paridade pelo
botão acessível, com a ordem relida do servidor após reinício em aba nova; verde
no CI com **23/23 checks** (E-034). O backlog datado foi corrigido (E-036
registra a saúde do PR dos checkpoints).

### 3.2 Duplicação de jornada de primeira execução

PR #61 e PR #71 entregam a mesma jornada. Isso é débito de processo, não
defeito de código; a decisão de qual supersede o outro exige autorização e está
registrada em `HADES_PENDING_WORK.md` (P0-5).

### 3.3 Falhas de teste não são regressões

Quatro falhas (`./internal/agent/`: upload/worktree exit 128, browser operator
Python ausente, kill de process group sem `sh`; `./server/`: `Filename too long`)
foram **provadas pré-existentes em `main` limpo** (E-015, E-016). Registradas
como bloqueio ambiental (B-09), **não** como defeito introduzido — e sem
desativar nenhum teste.

## 4. Cobertura de auditoria após a rodada 2

A rodada 1 deixou 12 superfícies sem probe. A rodada 2 (E-037…E-047) fechou
quase todas: **36 dos 38 estágios têm diagnóstico**; restam apenas o estágio 21
(benchmark, ainda documental) e o estágio 38 (gate final, que depende dos
demais).

Achados que mudam decisão de produto:

| Achado | Evidência | Consequência |
| --- | --- | --- |
| Harness Runtime **não existe em `main`** | `grep harness` = 0 em `server/`, 1 em `internal/agent` (teste) | estágio 6 é `AUSENTE`; rota é mergear o PR #62 (B-01) |
| Model Gateway cobre 7 famílias, não 12 | `internal/multillm` (26 arquivos) + contagem por provedor | faltam vLLM, llama.cpp, LM Studio, Lemonade e OpenRouter |
| Memória tem motor real, injeção de contexto não evidenciada | `ContextStore.RetrieveRelevant`, `OllamaEmbedder` | estágios 5 e 15 ficam `PARCIAL`/`IMPLEMENTADO_NAO_HOMOLOGADO` |
| Subagentes têm orquestrador completo | `swarm.go` (papéis, orçamento, plano, run, cancel) | falta apenas missão real com artefatos |
| Plugins/Skills são parciais | `plugin_scope.go` vs. manifesto assinado de skills | gerenciador de plugins não evidenciado |
| CRM, Commerce, Marketplace, Pagamentos, Inbox, Telefonia **não têm motor** | `marketplace=0`, `inbox=0`, `telephony=0`; `crm`/`billing` só no catálogo | estágios 24–29 e 35 são `AUSENTE`, não "parciais" |
| Company OS e WhatsApp são substanciais | 11 e 4 arquivos de teste, aprovações e idempotência | `IMPLEMENTADO_NAO_HOMOLOGADO`/`PARCIAL` com base real |

Regra que continua valendo: **ausência de evidência não é prova de ausência** —
mas, depois de probe negativo explícito com comando registrado, a classificação
honesta passa a ser `AUSENTE`, e não `PENDENTE_DE_AUDITORIA`.

## 5. Qualidade da base (métrica bruta)

- `internal/agent`: 102 arquivos `_test.go`, 583 `func Test*`.
- `server`: 83 arquivos `_test.go`, 434 `func Test*`.
- Web: 65 arquivos de teste, 396 casos; build de produção 1.374,24 kB
  (gzip 379,82 kB).

Métrica bruta **não** é cobertura; serve apenas para dimensionar a superfície
verificável existente (E-010).

## 6. Conclusão

O projeto está **estável e com CI verde**, com uma base de testes substancial,
um gate de CI que funciona e uma capacidade recente (jornada de primeira
execução) provada de ponta a ponta. Ao mesmo tempo, **não é release**: há 14
estágios dependentes de terceiros, 3 superfícies sem auditoria, um E2E de
Studio faltando e pendências de produto em aberto. Veredito:
**`NOT_RELEASE_READY`** (ver `docs/mission/HADES_NEXT_GOAL_READINESS.md`).

Nenhuma capacidade foi promovida a `COMPLETED_VERIFIED` nesta rodada, e isso é
deliberado: a regra de evidência vale mais que a aparência de progresso.
