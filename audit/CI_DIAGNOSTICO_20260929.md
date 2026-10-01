# Diagnóstico CI recovery — 2026-09-29 20:10 -03

Branch: recovery/ollama-full-snapshot. SHAs: 7ed0a385 (gatilhos CI), 16e71858 (fixes CI/Windows).
Workflows agora rodam em push da recovery (dz23-agentic-quality e class-a-plus-integrity).
class-a-plus-integrity = SUCCESS em ambos os SHAs.

## Job PostgreSQL RLS (run 36642520602, SHA 16e71858) — FALHA restante
Erro real: `psql: error: connection to server at "127.0.0.1", port 5432 failed: FATAL: permission denied for database "ollama_agent" / DETAIL: User does not have CONNECT privilege.`
Causa: o fixture cria `ollama_agent_reverse_member` e imediatamente faz o probe de login
(`reverse_member_login="$(... psql -U ollama_agent_reverse_member -d ollama_agent ...)"`) ANTES
de qualquer GRANT CONNECT. O bootstrap fresh revoga PUBLIC CONNECT, então o probe falha.
Correção: conceder `GRANT CONNECT ON DATABASE ollama_agent TO ollama_agent_reverse_member;`
dentro do bloco SQL do fixture, logo após criar a role (antes do probe).
(Os GRANTs adicionais nas fases 1/2 são idempotentes e podem permanecer.)

## Job Windows portability (mesmo run) — 31 falhas -> 2 falhas
Resolvido: guards de plataforma (workspace_isolation_platform_test.go) para testes de isolamento
descriptor-bound (Linux-only) e para sandbox.exec (Linux-only).
Restantes:
1. `git_repo_test.go:22: trusted system Git could not be resolved: Git executable was not found in trusted system directories`
   -> safeToolPath() no Windows lista `C:\Program Files\Git\cmd;C:\Program Files\Git\bin;C:\Windows\System32;C:\Windows`.
   No runner, git pode estar em outro diretório (ex.: mingw64\bin / usr\bin) ou o diretório falha no
   trustedWindowsSecurity. Ação: ampliar a lista de diretórios confiáveis do Git e/ou revisar ACEs.
2. `snapshot_permissions_windows_test.go:90: snapshot ACL does not inherit to child directories and files: flags=0x0`
   -> o ACE criado por windows.ACLFromEntries não está preservando SUB_CONTAINERS_AND_OBJECTS_INHERIT.
   Ação: construir/atribuir o ACE com flags de herança explícitas (AddAce ou ACL manual) e reler para validar.

## Observações de segurança mantidas
- HBA de produção: runtime/migrator só em ollama_agent; rejects depois dos allows.
- Exceção `ollama_agent_reverse_member` existe somente no HBA temporário gerado no CI ($RUNNER_TEMP).
- Gitleaks no diff staged: sem leaks em todos os commits desta rodada.

## Próxima tarefa do usuário (novo escopo)
Live streaming SSE + frames do browser + split-screen UI. Verificações já feitas:
- SSE real: `GET /api/agent/v1/missions/:id/events/stream` (server/agent_routes.go) envia
  `event: mission` com array de eventos (poll de 750ms), NÃO eventos nomeados step_start/browser_screenshot.
- Tipos reais de evento: step.started, step.succeeded, step.failed, step.retry_scheduled,
  step.awaiting_approval, mission.created/planned/running/completed/failed/cancelled, browser.approval.
- Browser: internal/agent/browser.go (108 linhas) chama browser_helper.py, que já suporta action
  "screenshot" (page.screenshot). Não existe arquivo internal/agent/browser.go com captura automática
  nem evento browser_frame — precisa ser criado.
- UI: Vite + TanStack Router/Query + Tailwind + Headless UI + framer-motion (sem Radix).
