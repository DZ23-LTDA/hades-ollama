# HADES — ÍNDICE DE EVIDÊNCIAS

> Regra: **relatório não é prova; comando reproduzível é.** Toda classificação de
> capacidade precisa de uma entrada aqui. Nesta rodada **nenhuma capacidade foi
> promovida a `COMPLETED_VERIFIED`**.

Ambiente de coleta: host Windows, PowerShell 5.1, repositório
`C:\Users\DZ23\Documents\deepseek-harness\default-workspace\hades-ollama`,
Go em `_tmp\goroot\go\bin\go.exe` (go1.27.2), Node v24.19.0, `gh 2.102.0`,
Python 3.12. Datas relativas a `2026-10-10T08:17-03:00`.

| ID | Tipo | Comando reproduzível | Resultado observado | Severidade residual |
| --- | --- | --- | --- | --- |
| E-001 | git/estado | `gh api repos/DZ23-LTDA/hades-ollama/commits/main` | `main = 889750abe4006985b3bf9037bca2323d1e40cef0`, commit em `2026-10-09T10:24:45Z` | — |
| E-002 | pr/estado | `gh api 'repos/DZ23-LTDA/hades-ollama/pulls?state=open&per_page=100'` | **29 PRs abertos** (dependabot #21–#34/#64 + feature #56–#71) | nenhum dependabot fechado |
| E-003 | pr/estado | `gh api repos/DZ23-LTDA/hades-ollama/pulls/{54,55,63}` | #54 `merged=True @2026-10-09T02:46:26Z`; #55 `merged=True @2026-10-09T10:24:46Z`; #63 `open` | só #54/#55 mergeados |
| E-004 | ci/estado | `gh api .../actions/runs?head_sha=889750abe...` | runs recentes de `main` = `success` | — |
| E-005 | ci/classificador | `python scripts/classify-ci-surfaces.py` sobre a lista planejada de arquivos | `categories=DOCS_ONLY`, `docs_only=True`, `require_agentic_gates=False`, `EXIT=0` | — |
| E-006 | ci/precedente | check-runs do PR #57 (head `6c0142955`) | 15 checks, 0 falhas; `PR gate` = success; `linux`/`go_license`/`windows` = skipped | — |
| E-007 | ci/workflow | leitura de `.github/workflows/pr-gate.yaml` (107 linhas) | roda em todo `pull_request` sem path filter; pula poll profundo quando classificador retorna falso | — |
| E-008 | código/probe | `Select-String StudioCanvasPage.tsx -Pattern 'onDrag|draggable|onDrop|dataTransfer|reorderById|ArrowUpIcon|ArrowDownIcon'` | DnD real `975 draggable`/`976 onDragStart`/`980 onDragOver`/`984 onDrop`/`988 onDragEnd`; `445 reorderById`; botões `1012`/`1024` | **corrige premissa anterior** |
| E-009 | teste unitário | `Get-Content app/ui/app/src/lib/studioReorder.ts` + `studioReorder.test.ts` | helper puro + **5 testes** (frente, trás, no-op, imutabilidade, ids desconhecidos) | cobertura só unitária |
| E-010 | métrica de testes | contagem em `internal/agent` e `server` | `internal/agent` = 102 `_test.go` / 583 `func Test*`; `server` = 83 / 434 | métrica bruta, não cobertura |
| E-011 | ci/PR #71 | `gh api .../actions/jobs/114195639233`; log salvo em `_tmp/ci-job-web-mobile.log` | 23/23 checks `success`; passo `15 \| Web E2E — primeira execução \| success`; `2 passed (28.6s)` | Node 20 deprecation warning |
| E-012 | e2e/local | `npx playwright test` (sem filtro) com `workers: 1` | `Running 7 tests using 1 worker` · `7 passed (40.7s)` | — |
| E-013 | web/gates | `npm run lint` · `npx vitest run` · `npm run build` | lint exit 0 (1 warning pré-existente `react-refresh/only-export-components` em `confirmDialog.tsx:31`); 65 arquivos / 396 testes verdes; build ok `index-KIG9eL1f.js` 1.374,24 kB (gzip 379,82 kB) | warning de chunk > 500 kB |
| E-014 | go/gates | `gofmt -l` · gofumpt · `golangci-lint run` · `go vet ./internal/agent/` · `go build ./...` | `gofmt -l` vazio; gofumpt exit 0; lint `0 issues.`; vet exit 0; build exit 0 | — |
| E-015 | go/testes ambientais | `go test ./internal/agent/` | exatamente **3 falhas ambientais pré-existentes** (provadas em `main` limpo): import upload/worktree exit 128; browser operator exit 9009 (Python ausente); kill de process group (`sh` ausente) | **não são regressões** |
| E-016 | go/testes ambientais | `go test ./server/` | `TestImportMissionAttachmentsCreatesIndexedOrgProject` falha com `Filename too long` **nos dois branches** | ambiente Windows |
| E-017 | ci/teste | `python scripts/classify_ci_surfaces_test.py` | `Ran 13 tests … OK`, exit 0 | — |
| E-018 | catálogo | medição do catálogo de conectores (PR #70) | **122 linhas**: `available=17`, `operator_setup_required=104`, `provider_selection_required=1`; `Kind` app=13 / custom_api=105 / mcp=4; `SEM_SCOPES=0`; `Auth` sempre não-vazio; `quickConnects` 69 chaves | 85% exigem operador |
| E-019 | teste | `go test ./internal/agent/ -run 'TestOAuthClientStore'` + testes de catálogo (PR #70) | 6 testes OAuth + 3 de catálogo; CI verde; promoção não pode deixar `Auth` vazio | OAuth live exige `OLLAMA_TEST_OAUTH=1` |
| E-020 | teste/correção | PR #67 e PR #68 | `misspell` corrigido por `settings.misspell.ignore-rules` (sem `//nolint`); `flowGraph` 31/31 verde | — |
| E-021 | inventário | `Get-ChildItem audit/*.md`; `Test-Path docs/mission` | `audit/` com 30 documentos datados; `docs/mission/` **não existia** (criado nesta rodada) | docs datados são pistas, não prova |
| E-022 | inventário | `Get-ChildItem .github/workflows` | **15 workflows** (inclui `pr-gate`, `dz23-agentic-quality`, `dz23-e2e`, instaladores por SO, `release*`) | — |
| E-023 | inventário | contagem em `app/ui/app/e2e` | 3 specs `.ts` + 14 scripts `.mjs` (evidência não-assertiva) | — |
| E-024 | pr/não verificado | PR #60 `feat/rotacao-automatica-gateway` (`99ba19fbb`) | aberto; **diff não inspecionado nesta rodada** | sem evidência anexada |
| E-025 | código/branch-local | PR #62 `internal/agent/cli_harness_tool.go` | `ScopeHarnessCLI="harness:cli"`, tool `harness.cli.exec`, `RiskWrite`, `RequiresApproval: true` — **só no branch** | não está em `main` |
| E-026 | política | `internal/agent/capability_policy.go` + `server/agent_routes.go` | 16 escopos canônicos; `validateAutoRunCapabilities` recusa tudo exceto vazio/`workspace:read` | — |
| E-027 | runtime | `internal/agent/runtime.go` | `observeEvent` → `event()` com payload redigido; `Events(id)`; `store.AppendEvent`/`ListEvents`; sem helper genérico de emissão | PR #69 expõe saída no chat |
| E-028 | teste/segurança | `StudioCanvasPage.security.test.ts` | teste de segurança do canvas existe e passa | sem E2E de asserção |
| E-029 | código | `internal/agent/context.go` | `CreateSchedule`, `ValidateScheduleSteps`, `updateScheduleForOrganization`; execução via `ExecuteScheduleFlow` | desfecho por passo não persistido |
| E-030 | pr/adapter | PRs #63 (Telegram), #65 (guia de canais), #57 (comparativo AutoClaw) | adaptador + guia + comparativo abertos | homologação real exige token (B-03) |
| E-031 | pr/ui | PR #66 `feat/concessao-capacidades-ui` (`360b81c45`) | UI de concessão de capacidades aberta | — |
| E-032 | ci/cadeia de gate | `pr-gate.yaml:48-79` + `:59`; `release-readiness.yaml:72`; `release.yaml:836` | o nome literal `Web and mobile quality` é exigido; inserir passo **dentro** do job inalterado gateia o E2E sem editar branch protection | — |
| E-033 | release | `release.yaml`, `release-readiness.yaml`, `dz23-windows-installer.yaml`, `dz23-macos-build.yaml`, `dz23-linux-package.yaml` | pipelines existem; **nenhum artefato assinado publicado**; sem certificado de assinatura | B-06 |
| E-034 | ci/PR #73 | `gh api .../commits/6465e88f7.../check-runs` + log do job `Web E2E (shell smoke)` (run `38050086918`, job `114207287767`) | **23/23 checks `success`**, 0 pendentes, `mergeable_state=clean`; `Running 10 tests using 1 worker` · `10 passed (53.5s)`, com os 3 testes de `e2e/studioReorder.spec.ts` verdes (`✓ 6`, `✓ 7`, `✓ 8`) | — |
| E-035 | e2e/local | `npx playwright test` · `npx vitest run` · `npm run lint` · `npm run build` · `prettier --check` | 10 passed (7 anteriores + 3 novos); 65 arquivos / 396 testes; lint 0 erros (1 warning pré-existente); build ok; prettier ok | o **estado** do backend é fixture rotulada; o caminho exercitado é real |
| E-036 | ci/PR #72 | `gh api .../commits/d64a47ef3.../check-runs` | **15/15 checks completed, 0 falhas, 0 pendentes** em PR docs-only; `linux`/`go_license`/`windows` `skipped` por desenho | — |

## Limitações declaradas deste índice

0. E-034/E-035/E-036 foram coletadas **depois** da primeira versão deste
   checkpoint, quando os PRs #72 (checkpoints) e #73 (E2E do Studio) ficaram
   verdes; são as evidências mais recentes do conjunto.
1. E-011/E-012/E-013/E-014/E-015/E-016/E-017/E-018/E-019/E-020 vêm da sessão de
   trabalho já registrada (histórico desta missão), com os comandos e saídas
   citados; não foram re-executados após a criação deste checkpoint.
2. Nenhuma evidência aqui cobre: chamada real a provedor de modelo, conta real
   logada, token real de canal, telefonia, pagamento, assinatura de release.
3. `PENDENTE_DE_AUDITORIA` em `HADES_MASTER_STATE.json` marca exatamente as
   superfícies sem entrada neste índice.
