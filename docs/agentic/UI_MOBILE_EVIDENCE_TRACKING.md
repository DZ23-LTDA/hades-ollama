# Rastreamento de evid├¬ncias ÔÇö UI e mobile

> Fonte ├║nica de status: [`docs/agentic/PARITY_MATRIX.md`](PARITY_MATRIX.md) e [`docs/agentic/PRODUCT_TREE.md`](PRODUCT_TREE.md).
> Este documento lista, por dom├¡nio de interface/mobile, o baseline real observado no reposit├│rio, as lacunas em rela├º├úo ├á matriz e a cadeia de evid├¬ncias exigida para promover qualquer linha. **Nenhum status ├® promovido sem implementa├º├úo + teste automatizado + execu├º├úo/print reproduz├¡vel.**

## 1. Como usar este documento

Cada dom├¡nio cont├®m:

- **Baseline real**: arquivos, rotas, componentes e testes que existem hoje no checkout `feat/docs-tests-parity`.
- **Gaps vs. matriz**: o que a matriz classifica como `PARCIAL`, `ADAPTER IMPLEMENTADO`, `IMPLEMENTADA CONDICIONALMENTE` ou `PENDENTE` para UI/mobile.
- **Evid├¬ncias existentes**: prints, E2E, manifestos e gates que j├í foram executados (com refer├¬ncia, n├úo com alega├º├úo de prontid├úo).
- **Pr├│xima proposta de avan├ºo**: mudan├ºa m├¡nima vi├ível sugerida, com vincula├º├úo a teste automatizado e captura.
- **Depend├¬ncias expl├¡citas**: credenciais, hardware, servi├ºos externos, dispositivos ou decis├Áes de produto que bloqueiam a valida├º├úo.

---

## 2. Shell desktop

### Baseline real

- Rotas: `app/ui/app/src/routes/index.tsx` (home/chat), `agentic.tsx`, `company.tsx`, `settings.tsx`, `library.tsx`, `projects.tsx`, `scheduled.tsx`, `skills.tsx`, `plugins.tsx`, `tasks.tsx`, `connect.tsx`, `onboarding.tsx`, `c.$chatId.tsx`.
- Sidebar: `app/ui/app/src/components/AppSidebar.tsx`.
- E2E offline: `app/ui/app/e2e/shell.spec.ts` ÔÇö valida home, marca `Ollama Full`, modo local-first e itens laterais `Agente`, `Tarefas`, `Empresa`.
- Capturas: `docs/images/screens/class-a-plus-*.png` (10 telas) + `ollama-full-current-ui-2026-09-27.png`; manifesto `docs/images/screens/class-a-plus-capture-manifest.json`.

### Gaps vs. matriz

A matriz classifica **Shell desktop** como `PARCIAL`:

> ÔÇ£Shell com sidebar, Nova tarefa, Agentes, Habilidades, Plugins, Agendado, Biblioteca, Projetos, Tarefas, Conta e Configura├º├ÁesÔÇØ ÔÇö **faltam estados reais de cada tela, atalhos, responsividade, acessibilidade e captura E2E**.

Faltam componentes alvo listados em `PRODUCT_TREE.md`:

- `MissionTimeline.tsx` ÔÇö n├úo existe.
- `ApprovalCenter.tsx` ÔÇö n├úo existe (apenas `CompanyApprovalQueue.tsx` e teste de approval do Agentic Console).
- `ArtifactPanel.tsx` ÔÇö n├úo existe.
- `ProviderPicker.tsx` ÔÇö n├úo existe como componente dedicado (sele├º├úo est├í dentro do Agentic Console).
- `BuilderCanvas.tsx` ÔÇö n├úo existe.
- `SettingsWorkspace.tsx` ÔÇö n├úo existe.
- Navega├º├úo por teclado, command palette, busca global, split panes, notifica├º├Áes/inbox, tema claro/escuro, reduced motion e responsive desktop/tablet n├úo t├¬m testes nem prints.

### Evid├¬ncias existentes

| Evid├¬ncia | Arquivo/Comando | Limita├º├úo |
|---|---|---|
| 10 screenshots desktop 1440├ù900 | `docs/images/screens/class-a-plus-*.png` | Apenas desktop; n├úo cobrem mobile/tablet, acessibilidade, estados vazios/erro, anima├º├Áes |
| Manifesto de captura | `class-a-plus-capture-manifest.json` | Prova proveni├¬ncia do arquivo, n├úo funcionalidade |
| E2E shell offline | `app/ui/app/e2e/shell.spec.ts` | S├│ home; n├úo navega para sub-rotas porque depende de backend |
| Notas de proveni├¬ncia | `docs/images/screens/SCREEN_CAPTURE_NOTES.md` | Declara que prints s├úo sandbox/local-only |

### Pr├│xima proposta de avan├ºo

1. Criar `MissionTimeline` vinculado ├á API `/api/agent/v1/missions/:id/events`.
2. Criar `ApprovalCenter` que liste approvals pendentes do tenant.
3. Criar `ArtifactPanel` com preview/diff/download para artifacts da miss├úo.
4. Cobrir navega├º├úo lateral por teclado (tab/enter) e adicionar skip-link.
5. Adicionar testes E2E para `/agentic`, `/tasks`, `/settings` e `/company` com backend mockado.

### Depend├¬ncias

- Backend real para estados din├ómicos (requer `go run . serve` ou mock).
- Playwright/Chromium instalado.
- Decis├úo de design system/tokens para responsividade.

---

## 3. Chat local / Nova tarefa

### Baseline real

- `app/ui/app/src/routes/index.tsx`, `Chat.tsx`, `ChatForm.tsx`, `ChatSidebar.tsx`, `Message.tsx`, `MessageList.tsx`, `ThinkButton.tsx`.
- Hooks: `useChats.ts`, `useSelectedModel.ts`, `useModels.ts`.
- E2E verifica ÔÇ£O que posso fazer por voc├¬?ÔÇØ e marca.

### Gaps vs. matriz

A matriz classifica **Chat local** como `VALIDADA LOCALMENTE`, mas com gate faltante:

> ÔÇ£Jornada E2E com dados reais, falhas de modelo e recupera├º├úoÔÇØ.

Al├®m disso, faltam na UI:

- Entrada multimodal (arquivo, imagem, ├íudio, voz, drag-and-drop, clipboard).
- Provider/model picker dedicado mostrando estado real de cada provider (Claude, Codex, OmniRoute, local).
- Mode picker (Chat/Coding/Research/Browser/Computer use/Builder/Media/Local-only).
- Templates de miss├úo, drafts e hist├│rico de prompts.
- Approval preview antes de executar.

### Evid├¬ncias existentes

- Smoke E2E home passa offline.
- Prints mostram composer central.

### Pr├│xima proposta de avan├ºo

1. Exibir badges de estado por provider (`credential_configured`, n├úo conectado, indispon├¡vel).
2. Adicionar upload de arquivo/├¡cone de anexo no `ChatForm`.
3. Implementar mode picker no composer.
4. Adicionar E2E com backend mockado para cria├º├úo de miss├úo e erro de modelo.

### Depend├¬ncias

- Backend mock ou servidor Go.
- Defini├º├úo de quais providers devem aparecer como ÔÇ£conect├íveisÔÇØ vs. desabilitados.

---

## 4. Agentic Console / Miss├Áes

### Baseline real

- `app/ui/app/src/components/AgenticConsole.tsx`.
- Teste: `AgenticConsole.approval.test.tsx`.
- Rota: `app/ui/app/src/routes/agentic.tsx`.

### Gaps vs. matriz

- Falta timeline de eventos integrada.
- Falta visualiza├º├úo de plano/subtarefas.
- Falta Browser/computer use, terminal e logs na mesma tela.
- Falta aprova├º├Áes humanas centralizadas.
- Falta artefatos gerados e compartilhamento.

### Pr├│xima proposta de avan├ºo

1. Adicionar `MissionTimeline` ao Agentic Console.
2. Adicionar lista de approvals pendentes/aceitos.
3. Exibir artifacts da miss├úo com hash/vers├úo.

### Depend├¬ncias

- API `/api/agent/v1/missions/:id/events`, `/artifacts`, `/approvals`.

---

## 5. Biblioteca

### Baseline real

- Rota: `app/ui/app/src/routes/library.tsx`.
- Print: `class-a-plus-library.png`.

### Gaps vs. matriz

A matriz classifica **Artefatos** como `VALIDADA LOCALMENTE` e **Biblioteca** (dentro de Shell desktop) como `PARCIAL`.

Faltam na UI:

- Filtros de tipo (documentos, slides, planilhas, imagens, ├íudio, v├¡deo, sites, apps, dashboards).
- Preview por tipo.
- Versionamento/diff.
- Download/exporta├º├úo.
- Compartilhar e permiss├Áes.
- Busca e filtros.

### Pr├│xima proposta de avan├ºo

1. Implementar grid/lista com filtros por `MediaType`.
2. Implementar preview sandbox para sites/apps (j├í existe backend de preview no Builder).
3. Adicionar download com checksum.

### Depend├¬ncias

- API de artifacts e m├¡dia.
- Sandbox de preview seguro.

---

## 6. Projetos

### Baseline real

- Rota: `app/ui/app/src/routes/projects.tsx`.
- Componente: `ProductWorkspacePage.tsx`.
- Print: `class-a-plus-projects.png`.

### Gaps vs. matriz

Faltam na UI:

- Membros e pap├®is.
- Contexto persistente por projeto.
- Mem├│ria/instru├º├Áes do projeto.
- Tarefas do projeto.
- Builder do projeto.
- Deploy do projeto.

### Pr├│xima proposta de avan├ºo

1. CRUD de projetos com `organization_id` (backend j├í existe).
2. Tela de detalhe do projeto com abas: contexto, tarefas, artifacts, builder, deploy.

### Depend├¬ncias

- API `/api/agent/v1/projects` e relacionamentos.

---

## 7. Agendado / Tarefas

### Baseline real

- Rotas: `scheduled.tsx`, `tasks.tsx`.
- Prints: `class-a-plus-scheduled.png`, `class-a-plus-tasks.png`.

### Gaps vs. matriz

- Faltam estados reais de jobs: inbox, em execu├º├úo, aguardando aprova├º├úo, conclu├¡das, falhas, filtros e busca.
- Faltam retry/DLQ/replay na UI.
- Faltam pausar/retomar e hist├│rico de execu├º├Áes.

### Pr├│xima proposta de avan├ºo

1. Listar `QueueJob`s por organiza├º├úo/miss├úo.
2. Adicionar filtros por estado e organiza├º├úo.
3. Adicionar a├º├Áes de pausar/retomar/replay com approval.

### Depend├¬ncias

- API `/api/agent/v1/jobs`, `/api/agent/v1/schedules`.

---

## 8. Habilidades / Plugins

### Baseline real

- Rotas: `skills.tsx`, `plugins.tsx`.
- Componentes: cat├ílogo de connectors, lifecycle de MCP/Remote MCP/skills.
- Prints: `class-a-plus-skills.png`, `class-a-plus-plugins.png`.

### Gaps vs. matriz

- Faltam testes E2E de habilitar/desabilitar/remover.
- Faltam detalhe de manifesto, vers├úo, permiss├Áes.
- Faltam skill customizada.

### Pr├│xima proposta de avan├ºo

1. E2E de lifecycle: registrar, desabilitar, remover.
2. Tela de detalhe do skill/plugin mostrando scopes e approval.

### Depend├¬ncias

- Backend mock ou servidor real.

---

## 9. Settings / Conta

### Baseline real

- Rota: `settings.tsx`.
- Componentes: `AgenticControlCenter.tsx`, `Settings.tsx`, `ClaudeDesktopModelsSettings.tsx`, `CodexDesktopModelsSettings.tsx`.
- Print: `class-a-plus-settings.png`.

### Gaps vs. matriz

- Faltam configura├º├Áes de workspace, organiza├º├Áes, quotas/custo, regi├úo/idioma, RBAC, SSO, sessions/devices, secrets manager, auditoria/retent├º├úo.
- Faltam tema claro/escuro explicitamente testado.
- Faltam atalhos de teclado e acessibilidade.

### Pr├│xima proposta de avan├ºo

1. Adicionar se├º├úo de organiza├º├Áes e membros.
2. Adicionar toggle de tema com teste de snapshot dark.
3. Adicionar se├º├úo de sess├Áes/dispositivos.

### Depend├¬ncias

- APIs `/api/agent/v1/organizations`, `/auth/session`, `/devices`.

---

## 10. Builder / Canvas

### Baseline real

- Componente: `ProductWorkspacePage.tsx`.
- Backend: Builder CRUD com undo/redo, preview, export ZIP, publica├º├úo local.
- Playwright E2E mockado (citado em `OLLAMA_FULL_MISSION_STATE.md`).

### Gaps vs. matriz

A matriz classifica **Builders** como `PARCIAL`:

> ÔÇ£Studio visual realmente interativo para sites, apps, jogos, slides e dashboards, com export e deploy rastre├íveis. Faltam E2E contra backend real; editor drag-and-drop; colabora├º├úo/CRDT e deploy externo autorizado com health check/rollbackÔÇØ.

Faltam na UI:

- Canvas visual drag-and-drop.
- Preview interativo.
- Undo/redo na UI.
- Export/PDF/DOCX/PPTX.

### Pr├│xima proposta de avan├ºo

1. Adicionar E2E real contra backend (n├úo mockado) para create/save/undo/redo/preview/publish/export.
2. Adicionar teste de responsividade do preview.

### Depend├¬ncias

- Backend Builder rodando.
- Decis├úo de usar canvas existente ou criar novo.

---

## 11. Mobile

### Baseline real

- `apps/mobile-agentic/App.tsx`, `offlinePolicy.ts`, `offlinePolicy.test.ts`.
- README com instru├º├Áes de desenvolvimento e EAS.

### Gaps vs. matriz

A matriz classifica **Mobile** como `PARCIAL`:

> ÔÇ£App Android/iOS com inbox, push, offline sync e device control. Faltam testes f├¡sicos, push remoto, resolu├º├úo de conflitos e distribui├º├úo em lojasÔÇØ.

Faltam na UI:

- Telas de miss├Áes, approvals, login/SSO.
- Push notifications.
- Offline sync de fato testado em dispositivo.
- Preview/download de artifacts.

### Evid├¬ncias existentes

- `offlinePolicy.test.ts` ÔÇö pol├¡tica offline passa.
- `npm run typecheck` e `npm run test:policy` passaram (registrado em `OLLAMA_FULL_MISSION_STATE.md`).

### Pr├│xima proposta de avan├ºo

1. Adicionar testes unit├írios de telas mobile.
2. Documentar mock de push local.
3. Criar EAS preview build com registro de limita├º├úo.

### Depend├¬ncias

- Dispositivo/emulador Android/iOS.
- Conta Expo/EAS e credenciais de assinatura.
- Push provider autorizado.

---

## 12. Acessibilidade e responsividade

### Baseline real

- Nenhum teste a11y autom├ítico encontrado.
- Nenhum print mobile/tablet.
- `AppSidebar.tsx` marca `Ollama Full` e monograma `OF`.

### Gaps

- Sem testes de contraste/foco/screen reader.
- Sem capturas 390├ù844.
- Sem reduced motion.

### Pr├│xima proposta de avan├ºo

1. Rodar `axe-core` nos testes E2E.
2. Adicionar prints mobile via `playwright_browser_resize` 390├ù844.
3. Verificar navega├º├úo por teclado na sidebar.

### Depend├¬ncias

- Playwright instalado.
- Defini├º├úo de padr├úo WCAG 2.2 AA.

---

## 13. Registro de promo├º├úo de status

S├│ promover uma linha da matriz quando a seguinte evid├¬ncia estiver presente:

| Requisito | Onde registrar |
|---|---|
| Implementa├º├úo | PR/commit com arquivos alterados |
| Teste automatizado | Teste Vitest/Playwright/Go no repo |
| Execu├º├úo reproduz├¡vel | Comando + resultado + ambiente |
| Print/captura | `docs/images/screens/<tela>-desktop.png` e `<tela>-mobile.png` |
| Console limpo | Registro de erros reais corrigidos; favicon 404 pode ser ignorado |
| Limita├º├úo expl├¡cita | Nota no print e neste documento |

---

## 14. Depend├¬ncias cruzadas que afetam UI/mobile

| Depend├¬ncia | Impacto |
|---|---|
| PostgreSQL RLS P0 | Multi-tenant real na UI depende de isolamento enterprise resolvido |
| Providers externos (Claude, Codex, OmniRoute, xAI, Composio) | UI mostra adapters; smoke real depende de credenciais |
| GPU/Tesseract/modelos multim├¡dia | Entrada multimodal e preview de m├¡dia |
| Browser Operator Playwright | Operador visual na UI |
| Companion desktop/mobile | Device control e pairing |
| Builders externos/deploy | Deploy rastre├ível e rollback |

---

## 15. Checklist da tarefa 1 (DOCS)

- [x] Branch `feat/docs-tests-parity` confirmada; worktree limpo; upstream aponta para `origin/recovery/ollama-full-snapshot`.
- [x] `AGENTS.md`, `CLAUDE.md`, `audit/OLLAMA_FULL_HANDOFF_20260927.md`, `audit/OLLAMA_FULL_MISSION_STATE.md` e `docs/agentic/PARITY_MATRIX.md` lidos.
- [x] Invent├írio verific├ível dos gaps de UI/mobile criado (este documento).
- [x] Cadeia de evid├¬ncias proposta vinculando implementa├º├úo ÔåÆ teste ÔåÆ execu├º├úo/print.
- [ ] `CHANGELOG.md` atualizado com baseline real, sem promover status de paridade.
- [ ] `audit/OLLAMA_FULL_MISSION_STATE.md` atualizado com baseline real.
- [ ] `git diff --check` e demais checks documentais aplic├íveis.
- [ ] Commit e push apenas em `feat/docs-tests-parity`.

---

## Refer├¬ncias

- `docs/agentic/PARITY_MATRIX.md`
- `docs/agentic/PRODUCT_TREE.md`
- `docs/images/screens/SCREEN_CAPTURE_NOTES.md`
- `docs/images/screens/class-a-plus-capture-manifest.json`
- `app/ui/app/e2e/shell.spec.ts`
- `apps/mobile-agentic/README.md`
- `audit/OLLAMA_FULL_MISSION_STATE.md`
- `audit/OLLAMA_FULL_HANDOFF_20260927.md`
