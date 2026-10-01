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
tip. Continue on `recovery/ollama-full-snapshot`; never force-push or change
`main` without explicit authorization.
