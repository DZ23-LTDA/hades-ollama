# Estágio 19 — onboarding validado: jornada real de primeira execução (E-182)

## 1. O item estava NAO_COMPROVADO — e a lacuna era de CONSOLIDAÇÃO, não de capacidade

A matriz dizia: *"O onboarding está validado — **NAO_COMPROVADO**: não há teste de jornada de primeiro uso com
usuário"*. Fui verificar antes de "construir" qualquer coisa, e o que existia era bastante:

- `app/ui/app/e2e/firstRun.spec.ts` — jornada de **7 etapas** de UI;
- `app/ui/app/e2e/first-run-backend.mjs` — **backend HTTP real** que serve o contrato do portão de onboarding
  (`GET/POST /api/v1/settings`), com `POST /api/me` → 401 e `/api/version` para health check;
- `playwright.config.ts` — sobe o **build real** (`npm run build` + `vite preview`) e o backend real, com proxy
  de `/api` do preview para o backend: **sem `page.route`**, ou seja, requisição HTTP de verdade;
- e o **CI já executa isso**: passo `Web E2E — primeira execução` em
  `.github/workflows/dz23-agentic-quality.yaml` (`npx playwright test --project=first-run`).

O teste narrado pela própria spec: **instalação, onboarding, persistência, reinício, atualização e rollback**.

## 2. Execução local (navegador real)

```
npx playwright test --project=first-run --reporter=list
  ok 1 [first-run] › e2e\firstRun.spec.ts › primeira execução — gate de onboarding servido pelo backend (…)
  ok 2 [first-run] › e2e\firstRun.spec.ts:100:3 › primeira execução — gate de onboarding servido pelo backend ›
       instalação, onboarding, persistência, reinício, atualização e rollback (1.9s)
  2 passed (26.0s)
```

O backend real registrou o tráfego esperado: `GET /api/v1/settings → 200`, `POST /api/v1/settings → 200`,
`POST /api/me → 401`, `GET /api/version → 200`.

## 3. Um pré-requisito de ambiente que custou uma execução (e fica registrado)

A **primeira** tentativa falhou **no build, não no teste**:

```
[vite:esbuild-transpile] remove C:\Users\<user>\AppData\Local\Temp\esbuild-…: Acesso negado.
```

A causa é o `TEMP` do host negar remoção ao esbuild. Com `TMP`/`TEMP`/`TMPDIR` apontados para um diretório do
próprio workspace, a execução passou (**2 passed**). É o **mesmo tipo de obstáculo** já conhecido do lado Go
(onde `TMP`/`TEMP` e `APPDATA`/`LOCALAPPDATA` precisam estar no workspace para o `golangci-lint` funcionar) — e
agora está **dito**, não descoberto de novo na próxima sessão.

## 4. O que muda na matriz

| Item | Antes | Depois |
| --- | --- | --- |
| 3 — "O onboarding está validado" | **NAO_COMPROVADO** | **DONE** (E-182) |

Resumo da matriz: **17 DONE · 1 IN_PROGRESS · 3 BLOCKED_BY_EXTERNAL_DEPENDENCY · 3 NAO_COMPROVADO**.
Veredito global **inalterado**: `NOT_RELEASE_READY`.

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `npx playwright test --project=first-run --reporter=list` | **2 passed (26,0 s)** |
| `python scripts/hades_release_gate.py --check` | acusa divergência antes de regenerar; **consistente** depois |
| `python scripts/hades_mission_check.py` | exit 0 |

## 6. Limites declarados

- A jornada roda contra um **backend de contrato mínimo** (`first-run-backend.mjs`), **não** contra o binário
  completo do Hades servindo modelos: ela prova o **portão de onboarding, a persistência, o reinício, a
  atualização e o rollback**, não a instalação de um pacote assinado nem o uso com modelo real.
- O "usuário" é automatizado (Playwright + axe onde aplicável): **não** houve teste com pessoa real nem sessão de
  usabilidade observada.
- A jornada valida a **UI do shell**; fluxos de onboarding de **outras** superfícies (CLI, desktop empacotado)
  continuam sem essa prova.
- O pré-requisito de `TEMP` gravável é do **ambiente desta máquina**; em CI (Linux) ele não aparece.
