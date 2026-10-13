# HADES — PRONTIDÃO PARA O PRÓXIMO GOAL / RELEASE

## 0. ESTADO MEDIDO — `main` `fafda19e7` (esta é a leitura canônica; não herde número de texto antigo)

| Medida | Valor medido |
| --- | --- |
| Commit verificado | `fafda19e7` |
| Evidências no índice | **310** |
| Estágios com evidência | **38 de 38** |
| Bloqueios externos declarados | **9** |
| Matriz do gate final | `BLOCKED_BY_EXTERNAL_DEPENDENCY: 5 · DONE: 19` |
| Veredito global | **`NOT_RELEASE_READY`** |

**Regra desta seção (E-301, aplicada em E-310):** qualquer número acima **envelhece**. Esta seção já ficou desatualizada
uma vez (dizia `f190ec8ea` / 300) e foi remedida. **Antes de citar, remeça:**

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

**Veredito global: `NOT_RELEASE_READY`.**

Motivo: nenhum critério de aceite tem, hoje, prova reproduzível de ponta a ponta
que o satisfaça integralmente. Vários dependem de terceiros (B-02…B-07) e **esta lista de superfícies não auditadas envelheceu**: medida em `f190ec8ea`, são **38 de 38** estágios com evidência no índice (ver §0 acima). Declarar "100% pronto"
aqui seria falso.

| # | Critério | Estado | Base |
| --- | --- | --- | --- |
| 1 | `main` estável com CI verde | **DONE** | E-001, E-004 |
| 2 | Fluxo branch curta + PR respeitado (sem commit em `main`) | **DONE** | AGENTS.md + histórico |
| 3 | Checkpoints V2.1 no repositório | **DONE** | este PR (`docs/mission/*`, `audit/HADES_*`) |
| 4 | Auditoria real do estado (sem assumir histórico) | **DONE** | E-001, E-002, E-003, E-021 |
| 5 | Inventário completo das áreas canônicas de capacidade | **DONE** | 36 de 38 estágios auditados (E-037…E-047); restam apenas 21 (benchmark) e 38 (gate final) |
| 6 | Nenhuma capacidade promovida sem evidência | **DONE** | `HADES_EVIDENCE_INDEX.md` — 0 promoções |
| 7 | Jornada de primeira execução provada por E2E com backend real | **DONE** | E-011, E-012 |
| 8 | E2E do Studio (reorder + persistência) | **DONE** | E-008, E-009, E-034 (PR #73: arraste e botão acessível persistindo, ordem sobrevivendo ao reinício) |
| 9 | Workflow Engine com schedule provado ponta a ponta | **NOT_DONE** | E-029 (validação provada; execução real não) |
| 10 | Harness CLI governado em `main` com aprovação obrigatória | **NOT_DONE** | E-025 (branch-local, PR #62) |
| 11 | Mission Control com orçamento/pausa/cancelamento testados | **NOT_DONE** | E-020, E-027 (parcial) |
| 12 | Canais de IM homologados com credencial real | **BLOCKED** | B-03 |
| 13 | Inbox unificado omnicanal operacional | **BLOCKED** | B-03 (estágio 35 sem auditoria) |
| 14 | Voz e telefonia operacionais | **BLOCKED** | B-04 |
| 15 | Nenhum conector exibido como funcional sem sê-lo | **DONE** | E-018 (catálogo honesto: `available=17` de 122) |
| 16 | Capacidades negadas por padrão (falha fechada) | **DONE** | E-026 |
| 17 | Isolamento entre organizações provado por teste | **NOT_DONE** | sem evidência anexada |
| 18 | Nenhum segredo em Git, log ou resposta | **NOT_DONE** | não verificado nesta rodada (não afirmar sem varredura) |
| 19 | Instaladores por SO com smoke nativo real | **NOT_DONE** | E-022, E-033 |
| 20 | Release assinado e publicado por SO | **BLOCKED** | B-06 |
| 21 | Benchmarks com metodologia auditável | **NOT_DONE** | E-021 (documental) |
| 22 | UX de onboarding utilizável de ponta a ponta | **DONE** | E-011, E-012 (com limitação declarada do stub de download) |
| 23 | Gates reproduzíveis documentados e executáveis | **DONE** | E-013, E-014, E-017 + `HADES_HANDOFF.md §4` |
| 24 | Backlog priorizado e correto | **DONE** | `HADES_PENDING_WORK.md` (P0-2 corrigido; falta corrigir o backlog datado em P0-3) |
| 25 | Gate final dos 38 estágios verificado | **NOT_DONE** | estágio 38 `BLOCKED_BY_EXTERNAL_DEPENDENCY` |

## Contagem

- `DONE` = **13** (critérios 1, 2, 3, 4, 5, 6, 7, 8, 15, 16, 22, 23, 24)
- `NOT_DONE` = **8** (9, 10, 11, 17, 18, 19, 21, 25)
- `BLOCKED` = **4** (12, 13, 14, 20)
- Total = 25 ✓

Atualização de 2026-10-10 (pós-PR #73): o critério 8 saiu de `NOT_DONE` para
`DONE` com a cobertura E2E do reorder do Studio (E-034). O critério 24 também
melhorou: o backlog datado foi corrigido (P0-3 entregue).

Nota de honestidade: vários `DONE` são **parciais por escopo** (ex.: critério 7
prova o gate de UI, não o download de modelo real). Eles valem como "satisfeito
no escopo declarado", não como release pronto.

## O que falta para virar `RELEASE_CANDIDATE`

1. Fechar P0 (E2E do Studio, decisão do gate de falha de settings, correção do
   backlog datado, triagem da jornada duplicada).
2. Auditar os estágios `PENDENTE_DE_AUDITORIA` com comando reproduzível.
3. Obter de terceiros: contas/credenciais (B-02…B-07) para homologar o que é
   `APENAS_ADAPTER`/`APENAS_CATALOGO`.
4. Merge autorizado dos PRs verdes (#69, #70, #71 e os demais aprovados).
5. Gate final (estágio 38) com evidência por SO e release assinado.

## Regra de declaração

O estado só muda para `RELEASE_CANDIDATE` quando **todo** critério obrigatório
tiver evidência no índice; só muda para `COMPLETED_VERIFIED` após o gate final
reproduzido em ambiente limpo. Enquanto isso: `NOT_RELEASE_READY`.
