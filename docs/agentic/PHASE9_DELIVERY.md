# Relatório de Entrega da Fase 9: Paridade Total com Shell Agentic Manus

**Data:** 29 de Setembro de 2026  
**Status:** VALIDADA LOCALMENTE COM EVIDÊNCIAS DE TESTE E BUILD  
**Autor:** DZ23 Automação e Engenharia  

---

## 1. Escopo e Objetivos da Entrega

A Fase 9 fecha o gap crítico de paridade observável com o assistente agentic desktop (Manus), unificando as seguintes frentes técnicas:

1. **Frente A — Shell Split-Screen na UI (`app/ui/app`):**
   - Layout de tela dividida em 2 colunas responsivas:
     - **Esquerda (Conversa e Raciocínio):** Cabeçalho da missão com indicador de status em tempo real, árvore de passos interativa com badges de estado (Executando com spinner, Concluído com checkmark, Falho com erro, Aguardando Aprovação com relógio pulsante), thought stream consumido via Server-Sent Events (`/api/agent/v1/missions/:id/events/stream`) sem necessidade de polling, e cards de aprovação in-line na timeline.
     - **Direita (Canvas Ativo):** Abas para Navegador ao Vivo (Live Browser), Visualizador de Artefatos com abas e Terminal/Código.
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
   - Atualizados `docs/agentic/PARITY_MATRIX.md` e `docs/agentic/PRODUCT_TREE.md` marcando os componentes implementados como `VALIDADA LOCALMENTE`.

---

## 2. Evidências de Validação Automatizada

### Testes de Backend (Go)
```text
=== RUN   TestRuntimeEmitsBrowserFrameEvent
--- PASS: TestRuntimeEmitsBrowserFrameEvent (0.01s)
PASS
ok      github.com/ollama/ollama/internal/agent 0.020s
```

### Testes de Frontend (React / Vitest)
```text
 ✓ src/components/AgenticConsole.approval.test.tsx (3 tests) 64ms
   ✓ AgenticConsole approval decisions > blocks an empty decision reason and sends the explicit reason 34ms
   ✓ AgenticConsole approval decisions > requests an isolated mission snapshot only after an explicit opt-in with a project 22ms
   ✓ AgenticConsole approval decisions > announces runtime load failures as an accessible alert 6ms
 ✓ src/components/AgenticSplitShell.test.tsx (2 tests) 85ms
   ✓ AgenticSplitShell component > renders split-screen layout with objective, status and tabs 68ms
   ✓ AgenticSplitShell component > switches to artifacts tab and displays artifact details 17ms

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
✓ built in 14.55s
```

---

## 3. Arquivos Criados e Modificados

- `app/ui/app/src/components/AgenticSplitShell.tsx`: Container mestre do shell split-screen com thought stream e canvas ativo.
- `app/ui/app/src/components/ArtifactsViewer.tsx`: Visualizador de artefatos com abas de preview, markdown, código e proveniência SHA-256.
- `app/ui/app/src/hooks/useMissionEvents.ts`: Hook de consumo de Server-Sent Events (`EventSource`) em tempo real.
- `app/ui/app/src/components/AgenticSplitShell.test.tsx`: Testes unitários do layout split-screen e troca de abas.
- `app/ui/app/src/components/AgenticConsole.tsx`: Integração do modo Split-Screen e alternância com o console detalhado.
- `app/ui/app/src/components/HomePage.tsx`: Suporte a autorun imediato na criação de missões a partir da tela inicial.
- `internal/agent/browser_helper.py`: Captura de frames/screenshots Playwright em formato base64.
- `internal/agent/runtime.go`: Emissão do evento `browser.frame` no fluxo de eventos da missão.
- `internal/agent/runtime_test.go`: Teste automatizado `TestRuntimeEmitsBrowserFrameEvent`.
- `docs/agentic/PARITY_MATRIX.md`: Atualização da matriz de paridade com o status `VALIDADA LOCALMENTE`.
- `docs/agentic/PRODUCT_TREE.md`: Atualização da árvore de produto.
