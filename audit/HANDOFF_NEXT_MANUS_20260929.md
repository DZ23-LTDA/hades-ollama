# Ollama Full — handoff para o próximo Manus (2026-09-29 20:15 -03)

## 1. Onde você está (leia antes de qualquer coisa)

- Repositório local: `/home/ubuntu/ollama-full-recovery`
- Branch autorizada: **`recovery/ollama-full-snapshot`** (NUNCA `main`, NUNCA force-push)
- Remoto: `https://github.com/DZ23-LTDA/ollama-classe-a-plus`
- HEAD local == remoto: **`2aa59a20505a627d936856a350eb784eb81b63f4`**
- Worktree: limpo. `git pull` não é necessário; apenas `git fetch origin`.

Branches que NÃO devem ser tocadas: `main`, `fix/audit-security-deps-2026-09-25` (PR #38),
`feat/ui-shell-parity`, `feat/ui-additive-parity`.

## 2. O que já está pronto e verificado (não refaça)

1. **Consolidação do PR #38 na recovery** (merge --no-ff, commit `8be25f6c`).
   Connectors/Providers/quick-connect/ícones/uploads/Windows entraram pela versão do PR;
   PostgreSQL/RLS/HBA/HMAC/cutover permaneceram da recovery. Sem duplicação do marketplace
   de conectores da linha de UI.
2. **Segurança de quick-connect** (endurecida nesta linha):
   - Bearer/organização do agente são repassados às rotas de conector (`app/ui/app/src/lib/connectorConnect.ts`).
   - Preflight de autenticação **antes** de gravar qualquer segredo.
   - Allowlist de operações GET somente-leitura por provider (`internal/agent/connector_catalog.go`);
     providers sem política explícita **não** são quick-connectable (fail closed).
   - Rollback do segredo anterior em qualquer falha pós-gravação, recusando sobrescrever
     credencial existente ilegível.
   - Rejeição de path traversal codificado em operações de conector.
3. **Gates locais verdes** nesta linha: `go test ./...`, `go vet ./...`, `go test -race ./internal/agent`,
   build nativo, cross-compile Windows do agent, `gofmt`, `git diff --check`,
   validador YAML/Bash do workflow, validador do Compose, integrity guard, Gitleaks (diff staged).
4. **CI remoto**: workflows `dz23-agentic-quality` e `class-a-plus-integrity` agora rodam em push
   da recovery. `class-a-plus-integrity` = SUCCESS.

## 3. O que falta (fila real, em ordem)

### 3.1 Job PostgreSQL RLS — correção recém-publicada, aguardando confirmação
- Falha observada: `FATAL: permission denied for database "ollama_agent" / User does not have CONNECT privilege`.
- Causa: o fixture criava `ollama_agent_reverse_member` e imediatamente testava o login, antes de qualquer GRANT.
- Correção publicada em `2aa59a20` (GRANT CONNECT dentro do bloco SQL do fixture, antes do probe).
- **Ação**: confirmar o run de `2aa59a20` em
  `gh run list --repo DZ23-LTDA/ollama-classe-a-plus --branch recovery/ollama-full-snapshot`.
  Se ainda falhar, ler `--log-failed` e filtrar apenas as linhas de saída real (o log é dominado
  pelo script ecoado com escape `[36;1m`).

### 3.2 Job Windows portability — 31 falhas → 2 falhas restantes
Resolvido: testes de isolamento descriptor-bound e `sandbox.exec` agora são guardados como Linux-only
(`internal/agent/workspace_isolation_platform_test.go`).

Restantes (ambas são lacunas reais de portabilidade Windows, não do merge):
1. `git_repo_test.go:22: trusted system Git could not be resolved: Git executable was not found in trusted system directories`
   - `safeToolPath()` no Windows usa `C:\Program Files\Git\cmd;C:\Program Files\Git\bin;C:\Windows\System32;C:\Windows`.
   - O bash do runner é `C:\Program Files\Git\bin\bash.EXE`, então Git existe; provavelmente um diretório
     falha em `trustedWindowsSecurity` (ACE de escrita de um principal não confiável, ou ACE
     inherit-only — este último já foi corrigido em `trusted_exec_windows.go`).
   - **Ação**: no runner, imprimir `icacls "C:\Program Files\Git\cmd"` e `C:\Program Files\Git\bin`
     para ver qual ACE reprova, e ampliar/ajustar a allowlist com base em evidência.
2. `snapshot_permissions_windows_test.go:90: snapshot ACL does not inherit to child directories and files: flags=0x0`
   - `windows.ACLFromEntries` está perdendo `SUB_CONTAINERS_AND_OBJECTS_INHERIT` (chega `0x0`).
   - **Ação**: construir o DACL com flags de herança explícitas. Opções: usar `AddAccessAllowedAceEx`
     via `windows.NewLazySystemDLL("advapi32.dll")` (não exportado pelo x/sys v0.42), ou montar o
     `EXPLICIT_ACCESS`/`ACL` manualmente. Depois reler o ACE e confirmar as flags.
   - Não "conserte" enfraquecendo a asserção: herança é requisito de segurança real.

### 3.3 Nova tarefa do usuário (escopo já validado contra o código)
Objetivo: live streaming SSE + frames do browser + painel split-screen.

**Correções importantes ao briefing recebido** (o briefing estava impreciso):
- O SSE real é `GET /api/agent/v1/missions/:id/events/stream` e envia `event: mission` com um
  **array** de eventos (poll interno de 750 ms) — NÃO eventos nomeados `step_start`,
  `browser_screenshot`, etc.
- Tipos reais de evento: `step.started`, `step.succeeded`, `step.failed`, `step.retry_scheduled`,
  `step.awaiting_approval`, `mission.created|planned|running|completed|failed|cancelled`, `browser.approval`.
- Não existe `browser_frame` nem captura automática: `internal/agent/browser.go` (108 linhas) apenas
  invoca `browser_helper.py`, que **já suporta** `action: "screenshot"` (`page.screenshot`).
- Não existe `useMissionStream.ts`. A UI é Vite + TanStack Router/Query + Tailwind + Headless UI +
  framer-motion (sem Radix).
- Portanto a Frente 2 exige código novo: publicar o frame como evento estruturado (ex.: `browser.frame`)
  no canal de eventos da missão, com throttle por keyframe.

### 3.4 Itens de produção ainda em aberto (não declarar pronto)
- Auditoria contínua do RLS/HBA em ambiente real de staging.
- Revisão externa de staging/produção.
- `main` só recebe a recovery em um release deliberado e separado.

## 4. Regras de continuidade (obrigatórias)

Após **cada** rodada verificada (~20-30 min de trabalho), nesta ordem:
1. `git add -A` + commit com mensagem clara.
2. `git push origin recovery/ollama-full-snapshot` (sem force-push).
3. Atualizar `audit/OLLAMA_FULL_HANDOFF_20260927.md` e `audit/OLLAMA_FULL_MISSION_STATE.md`
   com estado, último SHA, o que está pela metade e o PRÓXIMO passo concreto; commitar e pushar.

Nunca versionar credenciais. Sempre rodar Gitleaks no diff staged antes de commitar.
Nunca declarar paridade total com o Manus nem produção pronta sem evidência.

## 5. Evidência e arquivos de apoio

- Diagnóstico detalhado desta rodada: `audit/CI_DIAGNOSTICO_20260929.md`.
- Log do CI que falhou (para consulta): `/tmp/ci-failed-2.log` (pode não sobreviver a um novo sandbox).
- Harness adversarial PostgreSQL 16 + Redis: `/tmp/verify_ollama_full_security_round.sh`
  (temporário; recriar a partir do workflow se ausente).
