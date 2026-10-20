# Relatório de baseline — estágio 2 (gerado, não editar à mão)

> Gerado por `python scripts/hades_baseline_report.py`. O `--check` no CI falha se este arquivo divergir do
> gerador, para o baseline não envelhecer em silêncio.

## 1. Ambiente das medições

- commit de referência (`main`): `4b3e21854`
- commit **medido** (`measured_commit`): `7895b33e8a2f3a5c1f09ff77c75c7b1bbbb585df`
- medido em (`measured_at`): 2026-10-20T11:11:26-03:00
- SO das medições locais: Windows (runner do agente); o CI roda Linux
- toolchain: **go1.26.9**, fixado por `go-version-file: go.mod` em todos os workflows (E-167)
- plataforma Linux: container `golang:1.26.9`; execução de plugin: container `node:22-bookworm`

## 2. Suítes medidas

| Suíte | Comando reproduzível | Resultado observado |
| --- | --- | --- |
| agent (Windows) | `go test ./internal/agent/ -count=1` | ok — 70,2 s a 74,6 s |
| server (Windows) | `go test ./server/ -count=1` | ok — 22,2 s a 24,0 s |
| agent (Linux, container golang:1.26.9) | `docker run --rm -v <repo>:/src -w /src golang:1.26.9 sh -c 'go vet ./internal/agent/ ./server/ && go test ./internal/agent/ ./server/'` | ok — agent ≈20 s a 27 s · server ≈6 s a 8 s |
| race detector (Linux, CGO_ENABLED=1) | `scripts/verify-linux-parity.ps1 (etapa de -race) em golang:1.26.9` | sem DATA RACE — agent 80,5 s · server 11,0 s |
| lint | `golangci-lint run --allow-parallel-runners ./internal/agent/... ./server/...` | 0 issues |
| contrato de superfícies Class A+ | `node scripts/verify-contracts.mjs` | PASSED — chamadas do frontend batem com método e rota |
| web E2E — primeira execução | `npx playwright test --project=first-run --reporter=list (app/ui/app)` | 2 passed (26,0 s) — instalação, onboarding, persistência, reinício, atualização e rollback |
| plugin (terceiro + workspace + diagnóstico) | `go test ./internal/agent/ -run 'ThirdParty|PreparePluginWorkspace|ExecutionFailure' -count=1` | instalação/verificação/extracção PASS; execução SKIP declarada (sandbox Linux-only e sem privilégio de montagem) |

## 3. O que este baseline prova — e o que não prova

- **Prova**: as suítes acima foram executadas com estes comandos e estes resultados, nesta máquina e no CI (os
  workflows rodam as mesmas suítes; o CI é a autoridade sobre o verde).
- **Não prova**: desempenho em hardware diferente, nem que a suíte cobre todos os caminhos — cobertura não é
  medida aqui. Os tempos são de **uma máquina** e variam.
- **Não prova**: homologação em macOS nem em pacote assinado — esses requisitos seguem dependência externa.

## 4. Regressão

Comparar com este arquivo antes de integrar: mudança de tempo em ordem de grandeza, suíte que passa a falhar ou
SKIP novo precisam ser explicados no PR. Foi assim que a portabilidade Windows pegou o defeito de fim de linha
do fixture de terceiro (E-183) e o `govet` pegou o `return` inalcançável (E-183).
