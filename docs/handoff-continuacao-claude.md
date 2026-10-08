# Handoff — continuação (Claude Opus 4.8)

> Sessão de continuação porque o Manus ficou sem créditos. Branch de trabalho:
> `work/continue-hades` → **PR #42** para `recovery/ollama-full-snapshot`.
> Tudo com teste; `go build ./...` e `go vet` limpos. Um data race (pego pelo CI
> `-race`, que não roda local por falta de gcc) foi corrigido no `swarm Cancel`.

## O que foi entregue (backend, testado)

### Resiliência
- **E1** `server/routes.go` — o servidor core (generate/chat/tags/version) sobe mesmo se o agent falhar na init (degrada; todos os deref de `s.agentRuntime` são nil-guard).
- **E3** `internal/agent/runtime.go` — recovery re-enfileira missão em `MissionObserving` (transiciona Running/Observing → Recovering antes de enfileirar).
- **E4** `server/agent_routes.go` `health()` — profundidade real da fila (pending/running/dead_letter) + status degradado com dead-letter.
- **E5** `internal/agent/swarm.go` — `Cancel` interrompe orquestração **em execução** (registra cancel func por job; só sinaliza, não persiste — evita race).
- **E6** `server/download.go` — valida 206/200 antes de copiar o blob.

### Hardening / dados
- **A4** `internal/agent/project_import.go` — cap de ZIP por bytes reais escritos.
- **A5** `internal/agent/queue.go` — jitter (equal jitter) no backoff de retry.
- **A6** `internal/agent/backup.go` — `BackupData`/`RestoreData`: tar.gz do data root com checksum SHA-256, **exclui** `auth/` e `push/` (secrets).
- **A7** `server/diagnostics.go` — `buildDiagnosticReport()`: snapshot de suporte (versão/OS/integrações como booleanos), sem vazar valores (DLP).

### Diferenciais (cores)
- **G1 (RAG)** `internal/agent/context.go` + `rag.go`:
  - `RetrieveRelevant(ctx, projectID, query, limit, minScore)` → `[]ScoredMemory` (gate de relevância).
  - `BuildGroundedContext(query, sources)` → prompt citado `[n]` + `[]Citation`; sem fonte → instrui "não sei".
  - `(*ContextStore).GroundedAnswerContext(...)` → junta os dois (ponto de entrada).
- **G2** `internal/agent/rag.go` — `BuildWebGroundedContext(query, []ResearchSource)`: cita só páginas realmente lidas (título+URL+data).
- **G3** `internal/multillm/spend.go` — `Model.CostCents(in,out)` + `SpendLedger` (teto diário/mensal, `ErrSpendCapExceeded`, persiste JSON). Sem fallback pago silencioso.
- **G9** `internal/multillm/router.go` — `RoutePreference` (fastest/quality/cheapest) retrocompatível; ligado via `OLLAMA_AGENT_ROUTE_PREFERENCE` (aliases pt: rápido/qualidade/barato) em `server/agent_planner_resolver.go`.

### Portabilidade de teste (E9)
- `server/sched_test.go` `TestMain` limpa `OLLAMA_*` de tuning; fixtures/host/CONTEXT_LENGTH/limites deterministas; fixture MCP `.exe` no Windows.

## Próxima fase — wiring na UI (pro usuário USAR)
Os cores existem e passam em teste, mas **não estão ligados à UI/fluxo**. Pontos de integração:

1. **G1 RAG "Adicionar fontes":** endpoint que recebe arquivos/URLs → `DocumentIngestor.Ingest` (já existe) → no chat, chamar `GroundedAnswerContext` e injetar o contexto citado no prompt do planner; renderizar as `Citation` na resposta.
2. **G3 teto de gasto:** instanciar `SpendLedger` no gateway (`internal/multillm/proxy.go`), `Authorize` no início do request e `Record` após a resposta (parsing de `usage.prompt_tokens/completion_tokens`; cuidado com streaming — fazer best-effort). Expor limites em Settings.
3. **G9 preferência:** seletor no Model Picker que persiste a preferência (hoje é via env `OLLAMA_AGENT_ROUTE_PREFERENCE`); idealmente um setting → passado ao `RouteRequest.Preference`.
4. **A6 backup/restore:** botões em Settings chamando `BackupData`/`RestoreData` com download/upload.
5. **A7 diagnóstico:** botão "Exportar diagnóstico" → `buildDiagnosticReport` como download JSON.

Verificar tudo no navegador (desktop 1440x900 + mobile 390x844), console limpo, prints em `docs/evidencias/`.

## Ainda pendente (grande / externo)
- **Frontend inteiro:** G4 (app builder), G5 (modo código), G7 (canvas).
- **Refactors:** E2 (SSE pub/sub), A2 (quebrar god-objects).
- **Depende do dono/infra:** R5 (flaky), R7 (14 Dependabot), R9 (sync recovery→main + rc.2), S9 (rotação chave RLS/Postgres), S2 (sandbox MCP forte), assinatura SignPath.
