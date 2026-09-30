# Relatório de Entrega da Fase 9: Paridade Total com Shell Agentic Manus

**Data:** 29 de Setembro de 2026  
**Status:** VALIDADA COM EVIDÊNCIA DE NAVEGADOR (DESKTOP & MOBILE)  
**Autor:** DZ23 Automação e Engenharia  

---

## 1. Escopo e Objetivos da Entrega

A Fase 9 fecha o gap crítico de paridade observável com o assistente agentic desktop (Manus), unificando as seguintes frentes técnicas:

1. **Frente A — Shell Split-Screen na UI (`app/ui/app`):**
   - Layout de tela dividida em 2 colunas responsivas:
     - **Esquerda (Conversa e Raciocínio):** Cabeçalho da missão com indicador de status em tempo real, árvore de passos interativa com badges de estado (Executando com spinner, Concluído com checkmark, Falho com erro, Aguardando Aprovação com relógio pulsante), thought stream consumido via Server-Sent Events (`/api/agent/v1/missions/:id/events/stream`) sem necessidade de polling, e cards de aprovação in-line na timeline.
     - **Direita (Canvas Ativo):** Abas para Navegador ao Vivo (*Live Browser*), Visualizador de Artefatos com abas e Terminal/Código.
   - Cards de aprovação in-line na própria timeline quando a missão entra em `AWAITING_APPROVAL`, contendo campo de justificativa obrigatório e botões de decisão (Aprovar e Prosseguir / Rejeitar) integrados diretamente ao contrato `POST /api/agent/v1/missions/:id/approvals/:approval_id`.

2. **Frente B — Ponte Browser→Evento:**
   - Adicionada captura automática de frame JPEG base64 no helper Python do Playwright (`internal/agent/browser_helper.py`).
   - Implementada no runtime Go (`internal/agent/runtime.go`) a emissão do evento `browser.frame` contendo URL, título e screenshot após execução de ferramenta do browser.
   - Integrado ao fluxo de streaming SSE `events/stream`, permitindo que o painel direito da interface espelhe em tempo real o que o agente visualiza durante a navegação.
   - Teste unitário automatizado comprovado em Go: `TestRuntimeEmitsBrowserFrameEvent`.

3. **Frente C — Painel de Artefatos e Entrega (`ArtifactsViewer.tsx`):**
   - Componente especializado com seletor de múltiplos artefatos e abas:
     - **Visualização Web:** iframe com sandbox restrito (`allow-scripts`) para renderização segura de HTML e protótipos web.
     - **Markdown:** visualização estilizada de documentos e relatórios gerados.
     - **Código-Fonte:** visualização monospace com quebras e rolagem para arquivos de código.
     - **Exportar e Integridade:** card de download com nome, tamanho em KB, badge de hash de proveniência SHA-256 e botão de download direto para `/api/agent/v1/missions/:id/artifacts/:artifact_id`.

4. **Frente D — Chat = Agente Executor:**
   - Unificação do fluxo de entrada: prompts submetidos na Home Page (`HomePage.tsx`) são direcionados com `&autorun=true` para o `AgenticConsole`.
   - O console agentic detecta o parâmetro `autorun`, cria a missão com capacidades completas (`workspace:read`, `workspace:write`, `browser:navigate`, `browser:files`, `browser:takeover`) e inicia a execução imediata em modo split-screen.

5. **Frente E — Varredura de Paridade:**
   - Atualizados `docs/agentic/PARITY_MATRIX.md` e `docs/agentic/PRODUCT_TREE.md` marcando os componentes implementados como `VALIDADA COM EVIDÊNCIA`.

---

## 2. Evidências Reais de Navegador (Playwright E2E)

O fluxo completo foi executado em ambiente real com backend Ollama (`go run main.go serve`) na porta `11434` e frontend de produção na porta `5173`. Todas as capturas foram salvas no diretório `docs/evidencias/` em resoluções Desktop (1440x900) e Mobile (390x844):

| Tela / Estado | Desktop (1440x900) | Mobile (390x844) | Descrição do Estado |
|---|---|---|---|
| **Página Inicial (Home)** | `docs/evidencias/home-desktop.png` | `docs/evidencias/home-mobile.png` | Identidade Ollama Full oficial, prompt com atalho Ctrl+Enter e barra lateral responsiva |
| **Planejamento / SSE Stream** | `docs/evidencias/agentic-planning-desktop.png` | `docs/evidencias/agentic-planning-mobile.png` | Transição automática da Home para o Split-Screen Shell com eventos SSE ativos |
| **Navegador ao Vivo** | `docs/evidencias/agentic-live-browser-desktop.png` | `docs/evidencias/agentic-live-browser-mobile.png` | Espelhamento do Playwright em tempo real renderizando `https://example.com/` |
| **Card de Aprovação In-line** | `docs/evidencias/agentic-approval-desktop.png` | `docs/evidencias/agentic-approval-mobile.png` | Card in-line na timeline aguardando justificativa e decisão humana |
| **Visualizador de Artefatos** | `docs/evidencias/agentic-artifacts-desktop.png` | `docs/evidencias/agentic-artifacts-mobile.png` | Missão concluída com abas de Visualização, Código e Exportação |
| **Exportação e SHA-256** | `docs/evidencias/agentic-artifacts-download-desktop.png` | `docs/evidencias/agentic-artifacts-download-mobile.png` | Card com botão Baixar Arquivo e badge do hash de proveniência SHA-256 |

### Comprovação de Download Real e Integridade Criptográfica

Na missão executada de ponta a ponta (`mis_3f43616b-7ece-42c9-8a4d-35654b380121`), o artefato gerado pelo passo `step_2` (`workspace.write`) foi baixado via endpoint HTTP e verificado por checksum SHA-256:

- **ID do Artefato:** `art_1790733772658143028`
- **Nome do Arquivo:** `relatorio-missao.md`
- **Tamanho:** 113 bytes
- **MIME Type:** `text/markdown; charset=utf-8`
- **Hash SHA-256 Verificado:** `a8fec5a426afaf8fdbc494452a7b30090f12e95f41744fc1b8a988b1046e53ba`
- **Conteúdo Baixado:**
  ```markdown
  # Relatório de Execução da Missão

  Navegação realizada com sucesso e frame visual espelhado em tempo real.
  ```

### Auditoria de Erros do Console e Rede

Registrado em `docs/evidencias/browser-console-audit.json`:
- Rotas do runtime agentic (`/api/agent/v1/*`): **100% livres de erros** (respostas 200, 201, 204).
- Erros 404 pontuais registrados apenas em serviços auxiliares opcionais desconectados em modo local puro (`/api/v1/chats`, `/api/v1/settings`, `/api/v1/cloud`), sem qualquer impacto na jornada agentic.

---

## 3. Evidências de Validação Automatizada

### Testes de Backend (Go)
```text
=== RUN   TestRuntimeEmitsBrowserFrameEvent
--- PASS: TestRuntimeEmitsBrowserFrameEvent (0.01s)
=== RUN   TestAgentCatalogRoutesFilterPrivateResourcesByOrganization
--- PASS: TestAgentCatalogRoutesFilterPrivateResourcesByOrganization (0.01s)
PASS
ok      github.com/ollama/ollama/internal/agent 0.020s
ok      github.com/ollama/ollama/server 0.020s
```

### Testes de Frontend (React / Vitest)
```text
 ✓ src/components/AgenticConsole.approval.test.tsx (3 tests) 57ms
 ✓ src/components/AgenticSplitShell.test.tsx (2 tests) 89ms

 Test Files  2 passed (2)
      Tests  5 passed (5)
```

### Qualidade de Código (ESLint & TypeScript)
```text
$ npx tsc -b
(Zero erros de compilação TypeScript)

$ npm run lint
(Zero erros e zero warnings ESLint)

$ npm run build
✓ built in 14.93s
```
