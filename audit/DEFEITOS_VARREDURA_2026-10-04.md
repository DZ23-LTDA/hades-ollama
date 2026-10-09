# Varredura honesta de defeitos — Hades (2026-10-04)

Varredura de QA por código + comportamento da API, feita por 5 auditores em paralelo.
**73 defeitos** catalogados com evidência `arquivo:linha`. Nada foi corrigido aqui — é o
inventário honesto para priorização.

> **Nota de honestidade sobre os dois servidores:** o app tem DOIS servidores — o Ollama
> (`:11434`, com *stubs* no-op para conectores/provedores) e o da webview (`app/ui/ui.go`,
> com os handlers reais). No **app instalado**, a UI fala com o servidor da webview (handlers
> reais + token), então os defeitos marcados **[DEV/PROXY]** ou **[REDE-EXPOSTA]** afetam o
> fluxo de QA/dev ou o modo exposto à rede — não necessariamente o uso local instalado. Os
> demais afetam o app instalado normalmente.

---

## GRAVIDADE ALTA

### Computadores / controle remoto (afeta app instalado)
- **EndpointPage.tsx:222-230** — Botão "Controlar pelo seu telefone (não configurado)" é morto (`disabled`, sem `onClick`, sem rota de pareamento). É a reclamação direta do usuário: não há caminho para controlar pelo celular.

### Saúde
- **HealthCenter.tsx:34** — Botão "Corrigir" chama o MESMO handler de "Detalhes" (`setDetails(id)`); não corrige nada, só mostra texto. Enganoso.

### Console de Missões (Agentic)
- **AgenticConsole.tsx:294-297 / AgenticSplitShell.tsx:158-165** — Campo "Envie uma instrução…" do rodapé não funciona: `createMission` lê `objective` por closure desatualizada (setObjective é assíncrono) → cai no early-return e nada acontece (ou cria missão com o texto errado).

### Automações
- **AutomationsPage.tsx:55-60,334 (context.go:915)** — Um "Gatilho/Webhook" também dispara sozinho num timer: `handleCreate` sempre envia `interval_seconds` (padrão 24h) mesmo para `webhook`. Automação criada como "evento" roda automaticamente a cada 24h — oposto do prometido.
- **AutomationsPage.tsx (fluxo webhook)** — Webhook é beco sem saída: a UI nunca mostra a URL do endpoint (`/api/agent/v1/webhooks/:id`), nem o header `X-Ollama-Agent-Secret`, nem o `Idempotency-Key`. O usuário configura e não tem como acionar.

### Studio
- **StudioCanvasPage.tsx:337-343,440-447,872-884** — "Publicar" é placeholder: `handleDeployAttempt` só faz `setDeployStatus("BLOCKED_EXTERNAL")` e NUNCA chama `deployBuilder` (existe em agenticClient.ts:492, nem importado). Deploy não funciona nem com credenciais; e os 4 provedores (Vercel/Cloudflare/Netlify/SSH) dão a mesma mensagem citando só tokens de Vercel/Cloudflare/Netlify (incoerente para SSH).

### Conectores
- **ConnectorsPage.tsx:174-184** — Abas de categoria quebradas: filtram por palavras em inglês contra categorias em português. "Criatividade", "Viagem" e "Saúde" ficam **sempre vazias**; "Finanças" **omite todos os ~13 de Pagamentos** (Stripe, Mercado Pago, PayPal, Asaas, Pagar.me…).
- **connector_catalog.go + ConnectorQuickConnect.tsx** — 28 conectores `auth:"oauth"` marcados como "available" mostram "Conectar" mas caem em "login de um clique indisponível" (sem app OAuth registrado): Google Workspace, Gmail, Drive, Agenda, Slack, Instagram, Meta/TikTok Ads, Canva, Salesforce, Zoom, YouTube, LinkedIn, X, Dropbox…
- **[DEV/PROXY] desktop_local_routes.go:108-113** — No `:11434`, `PUT/DELETE /api/v1/connectors/:id/key` é no-op (`204` sem salvar). No fluxo de dev/proxy o usuário cola a chave, clica Conectar, recebe sucesso falso e o card continua "Conectar". (No app instalado o handler real de `ui.go` é usado.)
- **[DEV/PROXY] desktop_local_routes.go:93-107** — No `:11434`, `/api/v1/providers` é no-op → ProvidersPage mostra "0 de 0 provedores" e "Salvar chave" não persiste. (Handler real em `ui.go`.)

### Chat (mensagens do backend em inglês)
- **ui.go:646** — Limite de uso (402): "You've reached your usage limit, please upgrade to continue" em inglês, exibido cru.
- **ui.go:658,652** — "Connection lost" e "Unable to download model…" em inglês no chat.
- **ui.go:664** — Fallback `err.Error()` manda a string de erro do Go (inglês/técnico) direto pra tela.

### Biblioteca / Criações (afeta modo rede-exposta; em local funciona)
- **[REDE-EXPOSTA] LibraryPage.tsx:42** — `fetch` cru sem auth; com `authRequired` ligado retorna 401 e a Biblioteca fica permanentemente "Nenhum arquivo encontrado". (Resto do app usa `agentFetch`.)
- **[REDE-EXPOSTA] CreationsPage.tsx:45** — `fetch` cru sem auth; em 401 exibe "publicação e rollback ainda não estão disponíveis" — mensagem enganosa (o problema é auth).
- **[REDE-EXPOSTA] CreationsPage.tsx:178-183** — Preview em `<iframe>` para URL protegida por Bearer → com auth mostra 401 em vez do preview.

---

## GRAVIDADE MÉDIA

### Agentic
- **AgenticSplitShell.tsx:434,158-165** — Placeholder promete "pergunte ou crie missão", mas só cria missão nova; não dá pra mandar instrução à missão em execução.
- **AgenticSplitShell.tsx:481-492,537-551** — Aba "Terminal / Canvas" não é terminal nem editor: só despeja `JSON.stringify(payload)` num `<pre>`.
- **AgenticSplitShell.tsx:515-528** — "Navegador ao Vivo" promete espelhamento em tempo real, mas só funciona se o objetivo contiver "navegar/browser" E houver Python/Playwright no PATH; senão fica vazio sem explicar o pré-requisito.
- **AgenticConsole.tsx:309,331,344 / SplitShell:205,300,409** — Estados crus em inglês vazam: `mission.state` (COMPLETED/RUNNING/FAILED/AWAITING_APPROVAL), tipos de evento (step.succeeded…), `step.kind`, selo "approval".

### Automações
- **AutomationsPage.tsx:47-48,174,181,266** — Atalhos do Card 1/3 não passam `type`; se `triggerType` ficou "webhook" (de um clique anterior) e `webhookSecretEnv` vazio, `handleCreate` retorna sem feedback: botão silenciosamente não faz nada.
- **AutomationsPage.tsx:249-251** — "Automação avançada (Beta)" promete descrever QUANDO agir em linguagem natural, mas ignora o "quando" e fixa 24h.
- **AutomationsPage.tsx:33-41** — `loadData` com `fetch` cru engole erros (401/403/500) e mostra estado vazio em vez de reportar.

### Conectores / Harness
- **ProductWorkspacePage.tsx:284** — Botão "Configurar/Definir API/Escolher provedor" de qualquer conector não configurado leva para `/providers` (provedores de IA), não para um fluxo do conector. Ação enganosa.
- **connector_catalog.go:280-288** — 24 serviços (asaas, dropbox, hubspot, linear, mercado-pago, netlify, shopify, railway…) têm base/header de quick-connect mas sem operações → aparecem como OAuth/avançado; feature pela metade.
- **WhatsAppGatewayPanel.tsx:200-229** — Selos "Command Bridge / DLQ ativa / Zero loops" exibidos mesmo com gateway `NOT_CONFIGURED`; cards de backend são `div onClick` sem acessibilidade; mostra o enum cru "NOT_CONFIGURED".
- **ClaudeDesktopModelsSettings.tsx:453,486 / CodexDesktopModelsSettings.tsx:321,402** — Harness Claude/ChatGPT "disponíveis no aplicativo Ollama para macOS"; no Windows do usuário somem ou dão beco "só no macOS". Harness anunciado sem capacidade na plataforma atual.
- **ConnectorsPage.tsx:122-155** — "Criar Conector" registra operações GET/POST/PUT/DELETE em path "/" (full-access r/w), contradizendo "quick-connect é só leitura".

### Chat / Modelos
- **HomePage.tsx:206** — "PASS" (status interno de health-check) vaza no rótulo "Escolher modelo PASS".
- **modelPickerUtils.ts:31 / ModelPicker.tsx:417 / HomePage.tsx:206** — Rótulos de custo crípticos "0-assinatura"/"0-local" (deveriam ser "Grátis"/"Incluído na assinatura").
- **ChatForm.tsx:720-765,1091-1098** — Botão de anexar (+) depende de `window.webview?.selectMultipleFiles()`; sem isso o clique é ignorado em silêncio, e o fallback `<input type=file>` tem `handleFileInputChange` sem `onload` (código morto) → nenhum anexo é adicionado.
- **ChatForm.tsx:533-544 (slashCommands.ts:82)** — `/goal`, `/test`, `/review` sem texto são enviados como mensagem literal ao LLM em vez de abrir missão.

### Biblioteca / Studio / Projetos
- **LibraryPage.tsx:264-271,313-320** — Botão "Baixar" usa `<a download>` sem `Authorization`; com auth retorna 401/HTML em vez do arquivo. (Studio resolve com `agentFetchBlob`.)
- **CreationsPage.tsx:215-232** — "Abrir"/"Baixar código" usam a mesma URL protegida; com auth salvam HTML de erro, não o código.
- **CreationsPage.tsx:157-159 (agent_routes.go:1978)** — `creations` sempre retorna `PASS`; com zero criações e sem filtro, mostra "Nenhum artefato corresponde aos filtros atuais" (falso, não há filtro).
- **ProductWorkspacePage.tsx:273** — `mission.state` cru (RUNNING/OBSERVING/RECOVERING) na lista de Tarefas, sem tradução.
- **StudioCanvasPage.tsx:373** — Subtítulo "arrastando componentes", mas não há drag-and-drop algum; entram por clique e o canvas ignora x/y (pilha vertical).
- **StudioCanvasPage.tsx:636-648** — Só existe "Mover para cima"; o ramo "down" é caminho morto; não dá pra descer componente.
- **StudioCanvasPage.tsx:889,865** — "BLOCKED_EXTERNAL" cru e "Deploy Adapter" (inglês) ao usuário.

### Configurações
- **Settings.tsx:748-749** — Descrição do "Tamanho do contexto" é bilíngue quebrada: "…considerar local LLMs can remember and use to generate responses."
- **Settings.tsx:107** — `window.confirm("Discard unapplied app model changes?")` em inglês.
- **Settings.tsx:933-940** — "Banco de Dados: SQLite local-first + PostgreSQL RLS isolado" e "Contenção de Workspace: Ativa" são afirmações hardcoded, não vêm do backend.

### Empresa
- **CompanyOperationsPanel.tsx:57** — "Grok Live" fabrica estado plausível (`grok?.model ?? "grok-4"`, `?? "cataloged"`) mesmo quando a consulta falha.

---

## GRAVIDADE BAIXA

- **AgenticConsole.tsx:351-353** — Visão Clássica não tem botão parar/cancelar missão (só na Visão Dividida).
- **AgenticConsole.tsx:24,256** — `output`/`evidence` dos especialistas (orquestração) são descartados na tela.
- **AutomationsPage.tsx:349-354** — "Ver histórico de execuções →" aponta para `/agentic` (não é histórico).
- **AutomationsPage.tsx:76-91** — "Executar agora" só mostra toast, não leva a acompanhar a missão.
- **ArtifactsViewer.tsx:162-165** — Preview de `.md` não renderiza markdown (texto cru em `<pre>` com classe `prose`).
- **ArtifactsViewer.tsx:33** — Aba `"markdown"` no tipo, mas sem botão que a selecione (estado morto).
- **Onboarding.tsx:392-401** — Mostra "ollama run qwen2.5:0.5b" com botão copiar, contradizendo "sem terminal" e vazando a marca Ollama no fork Hades.
- **FirstModelCard.tsx:69 / ModelsPanel.tsx:102** — `value={modelName || recommended}`: ao apagar tudo, o recomendado reaparece como valor (campo parece não aceitar ser limpo).
- **Message.tsx:367,379-388** — "Resultado bruto da ferramenta" mostra `JSON.stringify` cru ao expandir.
- **ProductWorkspacePage.tsx:284** — Rótulos "Configurar/Definir API/Escolher provedor" diferentes, mas todos vão para `/providers`.
- **ProductWorkspacePage.tsx:160-164** — `window.location.assign` (reload completo) em vez de navegação SPA.
- **ImportProjectDialog.tsx:95-146** — ZIP cria o projeto ANTES do upload; se falhar, fica projeto órfão e cada tentativa cria outro duplicado.
- **StudioCanvasPage.tsx:519-568** — "Modelos de Projeto" sem `.catch` e substituem o projeto atual por um vazio sem confirmação.
- **StudioCanvasPage.tsx:774-844** — Inspetor incompleto: `navbar.links`, `metric.change`, `button.action` não editáveis.
- **StudioCanvasPage.tsx:725** — "URL do Preview" mostra URL protegida por Bearer (dá 401 se copiada).
- **EndpointPage.tsx:110-118,169-176,214-221** — Botões "Criar computador na nuvem", "Acesso remoto", "Conectar meu computador" mortos (`disabled`, sem handler).
- **EndpointPage.tsx:152-164** — Detalhes do host hardcoded ("Porta 11434", "Isolamento: Sandbox local e RLS", "Permissão: Total com aprovação").
- **EndpointPage.tsx:268,280,143 / Settings.tsx:636-639** — Strings em inglês ("OpenAI-Compatible", "Local-First", "Show apps in menu").
- **Settings.tsx:879-890** — Badge "Salvo" pisca a cada tecla no textarea de instruções.
- **Settings.tsx:893-908** — "Memória de Longo Prazo" com badge "Em desenvolvimento" ocupando espaço.
- **CompanySupervisorPanel.tsx:188** — `toLocaleTimeString()` sem locale "pt-BR".
- **CodexDesktopRow.tsx:504 / CodexDesktopModelsSettings.tsx:444 / ClaudeDesktopModelsSettings.tsx:566,154** — Strings em inglês ("Retry", "ChatGPT is taking longer…", "Let Claude decide…").
- **connectorConnect.ts:47-96** — KEY_HELP com URLs para serviços não quick-connect (link nunca renderiza).
- **connector_catalog.go (calendly)** — `auth:"oauth"` + quick-connect mostra "Chave de API", inconsistente.
- **ConnectorsManagePanel.tsx:158 / ConnectorQuickConnect.tsx:47** — "registro avançado →" leva a `/plugins` → botões que desviam para `/providers` (beco).
- **ProductWorkspacePage.tsx:259-267 / skills** — Cadastro de skill gera manifesto inerte (sem comportamento de runtime); `/skills` sempre vazio.

---

## Padrões (causas-raiz que resolvem muitos de uma vez)
1. **Dois servidores com rotas duplicadas** (stubs no `:11434` vs handlers reais em `ui.go`) — unificar/encaminhar resolve os no-ops de conector/provedor no dev.
2. **`fetch` cru sem `agentFetch`** (Library, Creations, Automations) — trocar por `agentFetch`/blob autenticado resolve vazio/401/download no modo exposto.
3. **Estados/erros crus em inglês** (ui.go + `mission.state`/`step.kind` na UI) — um dicionário pt-BR + humanização resolve ~15 itens de idioma.
4. **Botões "não configurado"/placeholder** (Computadores, Studio deploy, Health "Corrigir") — ou implementar, ou remover/explicar honestamente.
5. **Filtro de categorias por id em inglês** (ConnectorsPage) — mapear categorias pt-BR corrige todas as abas de uma vez.
</content>
