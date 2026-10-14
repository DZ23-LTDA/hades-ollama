# AGENTS.md

## Building

For a full build from the repository root:

```sh
cmake -B build .
cmake --build build --parallel 8
./ollama serve
```

For quick Go-only iteration against an existing native payload:

```sh
go build .
go run . serve
```

See `docs/development.md` for prerequisites, platform notes, GPU backends, and
the full development workflow.

## Ongoing Hades mission

Before continuing agentic-security or Manus-parity work, read
`audit/CLAUDE_CODEX_RESUME_PROMPT_20260927.md` and
`audit/OLLAMA_FULL_HANDOFF_20260927.md`; verify the live Git branch and remote
tip. `main` is the integration branch: cut a short-lived branch from `main`,
open a PR against it, and let the required checks run. The historical
`recovery/ollama-full-snapshot` snapshot has already been merged into `main`.
Never force-push or change `main` directly without explicit authorization.

## Windows: redirecione TMP/TEMP quando o teste "não roda"

Nesta máquina o **temp do usuário nega escrita**, e o sintoma é sempre *"o teste não
roda"* — **nunca** *"o teste falha"*. Isso já causou **duas** sessões perdidas
(E-405 e a 5ª auditoria independente), em **dois** toolchains diferentes:

| instrumento | erro | correção |
| --- | --- | --- |
| Go (runtime/`go test`) | falha ao criar `%AppData%\ollama-agent` e `os.UserCacheDir()` | `GOPATH`, `GOMODCACHE`, `GOCACHE`, `GOTMPDIR`, `APPDATA`, `LOCALAPPDATA` **todos** sob `_tmp/` |
| Node/esbuild/Playwright | `[vite:esbuild-transpile] remove C:\…\AppData\Local\Temp\esbuild-…: Acesso negado` e o `webServer` não sobe | `TMP`, `TEMP` e `TMPDIR` sob `_tmp/` |

Sem o redirecionamento do Node, o `npx playwright test` **falha ao iniciar** (*exit 1*,
`Build failed`) — e não por defeito do produto. Com ele, a suíte E2E passa
(*9 specs, 18 testes, 105 `expect`s, 0 skips*). Preserve `LOCALAPPDATA` real ao rodar
Playwright: o Chromium baixado vive em `%LOCALAPPDATA%\ms-playwright`.

Recapitulando por que isso importa: *instrumento que não roda é indistinguível de
instrumento que não mede nada*, e as duas vezes em que isso aconteceu nesta missão
produziram um **falso resultado** antes de alguém medir a causa.
