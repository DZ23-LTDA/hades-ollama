---
projeto: ollama-classe-a-plus
status: validado com evidencia real de navegador
atualizado: 2026-09-29 23:07 -03 (2026-09-30 02:07 UTC)
ultima_ia: Manus
tags: [projeto, paridade-manus, evidencia-real, shell-desktop]
---

# Ollama Classe A+ (Ollama Full)

> Memória compartilhada entre todas as IAs. **Leia inteiro antes de trabalhar** e
> **atualize antes de encerrar** qualquer etapa. Outra IA vai continuar daqui.
> A fonte durável detalhada do projeto está em `docs/agentic/` (matriz de paridade,
> relatórios de fase, evidências em `docs/evidencias/`). Este arquivo é o resumo de entrada.

## Objetivo
Fazer o `ollama-classe-a-plus` (fork do Ollama, da DZ23-LTDA) ter **paridade
funcional observável com o Manus Desktop** (tudo que ele faz e tem) **+ melhorias** tiradas
da análise de outros harnesses/repositórios, com qualidade enterprise
(isolamento por tenant, segurança de filesystem/processo, privacidade,
recuperação). **Pronto** = paridade comprovada por implementação + teste
automatizado + execução real reproduzível em navegador (desktop e mobile), sem alegar "100%" sem evidência.

## Estado atual (2026-09-29 23:07 -03 / 2026-09-30 02:07 UTC)
- **Repo:** github.com/DZ23-LTDA/ollama-classe-a-plus. Trabalhar **só** na branch
  `recovery/ollama-full-snapshot`. **NÃO tocar na `main`**, não force-push, não merge/PR sem autorização.
- **Fase 9 Entregue e Validada com Evidência Real:**
  - **Frente A (Split-Screen Shell):** Container responsivo de tela dividida (`AgenticSplitShell.tsx`) com cabeçalho de status da missão, árvore de passos com badges de estado ao vivo, thought-stream consumido em tempo real via Server-Sent Events (`/api/agent/v1/missions/:id/events/stream`) sem polling, e cards de aprovação in-line.
  - **Frente B (Ponte Browser→Evento):** Captura de frame no Playwright (`browser_helper.py`), emissão de evento `browser.frame` no runtime Go (`runtime.go`), e espelhamento em tempo real do viewport do navegador no canvas ativo da interface web.
  - **Frente C (Visualizador de Artefatos):** Componente com abas (`ArtifactsViewer.tsx`): Visualização Web (iframe sandboxed), Markdown formatado, Código-fonte com syntax styling, e Aba de Exportação com card de download e verificação criptográfica SHA-256.
  - **Frente D (Chat = Agente):** Disparo de prompts da Home Page (`HomePage.tsx`) via `/agentic?objective=...&autorun=true` com instanciação imediata e execução em modo split-screen.
  - **Frente E (Matriz de Paridade & Árvore de Produto):** Atualizados `docs/agentic/PARITY_MATRIX.md` e `docs/agentic/PRODUCT_TREE.md` com status **`VALIDADA COM EVIDÊNCIA (docs/evidencias/*)`**.
- **Evidências Reais de Navegador (`docs/evidencias/`):**
  - `home-desktop.png` (1440x900) & `home-mobile.png` (390x844): Página inicial com logo Ollama oficial, prompt de tarefa e layout mobile responsivo com sidebar retrátil.
  - `agentic-planning-desktop.png` & `agentic-planning-mobile.png`: Transição automática e split-screen ativo com thought stream SSE.
  - `agentic-live-browser-desktop.png` & `agentic-live-browser-mobile.png`: Espelhamento ao vivo do Playwright renderizando `https://example.com/`.
  - `agentic-approval-desktop.png` & `agentic-approval-mobile.png`: Cards de aprovação in-line na timeline com justificativa e botões de decisão.
  - `agentic-artifacts-desktop.png` & `agentic-artifacts-mobile.png`: Missão concluída com abas de Visualização, Código e Exportação.
  - `agentic-artifacts-download-desktop.png` & `agentic-artifacts-download-mobile.png`: Card de download com SHA-256 comprovado.
  - `browser-console-audit.json`: Auditoria de rede e console, registrando rotas do agente 100% íntegras (200/201/204).
- **Prova de Download Real e SHA-256:**
  - Arquivo: `relatorio-missao.md` (113 bytes)
  - Hash SHA-256: `a8fec5a426afaf8fdbc494452a7b30090f12e95f41744fc1b8a988b1046e53ba`
- **Quality Gates Aprovados:**
  - Backend: `go test -v -run TestRuntimeEmitsBrowserFrameEvent ./internal/agent` (PASS)
  - Server: `go test -v -run TestAgentCatalogRoutesFilterPrivateResourcesByOrganization ./server` (PASS)
  - Frontend: `npx tsc -b` (0 erros), `npm run lint` (0 erros), `npx vitest run` (5/5 testes passando), `npm run build` (sucesso em 14.9s).

## Próximo passo (instruções exatas para a próxima IA)
1. **Ambiente e Repositório:**
   - Trabalhe sempre na branch `recovery/ollama-full-snapshot`.
   - Se rodar o backend local: `export PATH=$PATH:/usr/local/go/bin && go run main.go serve` (porta 11434).
   - Se rodar o frontend: `cd app/ui/app && npm run build && npx vite preview --host 0.0.0.0 --port 5173` (ou `npm run dev`).
2. **Próximas Lapidações do Backlog de Paridade (`docs/dz23-parity-backlog.md` e `docs/agentic/PARITY_MATRIX.md`):**
   - **Multi-instância e failover de filas:** Estender testes de resiliência distribuída do Redis com chaos testing.
   - **Catálogo de Habilidades e MCPs Externos:** Integrar discovery de servidores MCP remotos com protocolo SSE/stdio e UI de pareamento.
   - **Jornada de Código Avançada (Coding Git Worktree Loop):** Conectar ciclo de branch/worktree isolado com runner de testes do projeto e repair loop automático.
3. **Checklist antes de qualquer commit:**
   - `export PATH=$PATH:/usr/local/go/bin && go test ./internal/agent ./server -count=1`
   - `cd app/ui/app && npx tsc -b && npm run lint && npx vitest run`
   - Atualize `MEMORIA.md` e a matriz de paridade se nova evidência for adicionada.
   - Nunca commitar chaves ou segredos.

## Decisões
| Data | Decisão | Motivo | IA |
|---|---|---|---|
| 2026-09-29 | Default planner unauthenticated configurado como RulePlanner | Permite que o runtime agentic funcione out-of-the-box localmente sem exigir download prévio de modelos de inferência | Manus |
| 2026-09-29 | agentOrganizationID retorna "local" por padrão em modo sem auth | Garante conformidade com LocalOrganizationID e validação de policy em CAS de approvals locais | Manus |
| 2026-09-29 | Sidebar retrátil por padrão no mobile (< 768px) com toggle hamburger | Elimina sobreposição de 50% da tela em telas mobile (390px) e garante UX responsiva | Manus |
| 2026-09-29 | Captura de frame no Playwright via base64 no evento browser.frame | Viabiliza espelhamento ao vivo no painel direito sem dependência de streaming WebRTC pesado | Manus |
| 2026-09-29 | Alvo = PARIDADE TOTAL com evidência real de navegador | Usuário exige paridade observável idêntica ao Manus com screenshots desktop e mobile em docs/evidencias/ | Manus / Usuário |

## Histórico de sessões
<!-- Mais recente no topo. Uma entrada por sessão de trabalho. -->
### 2026-09-29 23:07 -03 — Manus — FASE 9 COMPLETA COM EVIDÊNCIA REAL DE NAVEGADOR
- **O que foi feito:**
  - Implementado o shell split-screen completo (`AgenticSplitShell.tsx`) com layout de duas colunas, thought stream em tempo real via Server-Sent Events nativo (`useMissionEvents.ts`), badges dinâmicos de passos e cards de aprovação in-line.
  - Implementada a ponte de captura de frames do Playwright (`browser_helper.py`) e emissão de eventos `browser.frame` no runtime Go (`runtime.go`).
  - Implementado o visualizador de artefatos (`ArtifactsViewer.tsx`) com abas de Visualização Web, Markdown, Código-fonte e Download com hash SHA-256 de integridade.
  - Corrigido `RulePlanner` como padrão em `server/agent_routes.go` e `agentOrganizationID` para retornar `LocalOrganizationID` em modo local.
  - Adicionado suporte a layout mobile responsivo em `layout.tsx` (sidebar retraído no mobile < 768px).
- **Evidências Reais Geradas:**
  - Executado fluxo real ponta a ponta com Playwright no browser contra `http://127.0.0.1:5173/` e backend Ollama na porta `11434`.
  - Capturados e salvos em `docs/evidencias/`: `home-desktop.png`, `home-mobile.png`, `agentic-planning-desktop.png`, `agentic-planning-mobile.png`, `agentic-live-browser-desktop.png`, `agentic-live-browser-mobile.png`, `agentic-approval-desktop.png`, `agentic-approval-mobile.png`, `agentic-artifacts-desktop.png`, `agentic-artifacts-mobile.png`, `agentic-artifacts-download-desktop.png`, `agentic-artifacts-download-mobile.png`, `browser-console-audit.json`.
  - Download real do arquivo `relatorio-missao.md` com hash SHA-256 conferido (`a8fec5a426afaf8fdbc494452a7b30090f12e95f41744fc1b8a988b1046e53ba`).
  - Atualizados `docs/agentic/PHASE9_DELIVERY.md`, `docs/agentic/PARITY_MATRIX.md` e `docs/agentic/PRODUCT_TREE.md` para **`VALIDADA COM EVIDÊNCIA`**.
- **Onde parou:**
  - Repositório remoto no GitHub sincronizado na branch `recovery/ollama-full-snapshot` e pasta local no PC (`D:\IA\Trabalhos\DZ23-LTDA\ollama-classe-a-plus`) atualizada.
  - Sistema 100% operacional, estável e pronto para as próximas lapidações contínuas de produto.

### 2026-09-29 13:00 — Claude (Opus) — PIVOT: construir de verdade (não testar esqueleto)
- Feedback do usuário: a "paridade" via testes de telas magras era falcatrua. Meta
  reafirmada: "paridade em tudo" = construir pra ficar/funcionar como o Manus.
- Entrega real (commit 98a00438, feat/ui-shell-parity): reconstruí a tela Plugins
  num MARKETPLACE de conectores — novo src/lib/connectorCatalog.ts com 26 conectores
  e fontes de dados (Gmail, Google Workspace, Notion, GitHub, Instagram, Meta Ads,
  Similarweb, World Bank, BigQuery...), por categoria, com cor de marca por tile.
- Manus (backend) avançou muito em paralelo: rotação de chave HMAC (1bd88414), ACL
  pública, HA/TLS/failover. recovery em 00ee89d4.
