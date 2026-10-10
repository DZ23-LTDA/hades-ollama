# HADES — HANDOFF (como retomar a missão do zero)

## 1. Ordem de leitura obrigatória

1. `docs/mission/HADES_MASTER_GOAL.md` — objetivo, limites de autonomia, estados.
2. `docs/mission/HADES_MASTER_STATE.json` — **estado canônico de máquina**.
3. `docs/mission/HADES_STAGE_REPORT.md` — o que está feito/pendente por estágio.
4. `docs/mission/HADES_BLOCKERS.md` — o que depende de terceiros.
5. `docs/mission/HADES_EVIDENCE_INDEX.md` — o que é prova e o que não é.
6. `docs/mission/HADES_PENDING_WORK.md` — próximos passos priorizados.
7. `docs/mission/HADES_NEXT_GOAL_READINESS.md` — critérios de aceite.
8. `audit/HADES_MASTER_AUDIT.md`, `audit/HADES_CAPABILITY_MATRIX.md`,
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
- Precedente de PR docs-only: #57 (passou sem esperar gates profundos).

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
