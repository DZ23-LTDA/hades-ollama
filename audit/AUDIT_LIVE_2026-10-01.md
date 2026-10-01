# AUDIT-LIVE — Ollama Full

**Data:** 2026-10-01 02:25–02:30 UTC (2026-09-30 23:25–23:30 -03)
**Branch auditada:** `recovery/ollama-full-snapshot`
**HEAD observado:** `a457e151`
**Escopo:** auditoria E2E ao vivo, sem correção de código. Foram alterados apenas este relatório e evidências em `docs/evidencias/`.

## Resumo executivo

1. A casca visual abre em Desktop (1440×900) e Mobile (390×844), e 34 capturas foram geradas para 17 rotas.
2. O backend inicia, mas a saúde agentic exige Bearer e retorna `401` para o navegador sem sessão/token; isso bloqueia a maior parte dos fluxos reais.
3. Foram observadas **56 respostas HTTP 401** em endpoints de missões, MCP, conectores, projetos, schedules, skills, companies, builders e configurações.
4. A rota `/connect` quebra com `TypeError: M.find is not a function` em Desktop e Mobile, exibindo a tela genérica de erro.
5. O Ollama local respondeu `{"models":[]}`; portanto não havia modelo local para validar chat/execução, e o relatório não declara paridade completa.

## Três piores problemas

### P1 — Auth agentic impede todos os fluxos de dados

- **Tela/fluxo:** Agentic, Tarefas, Automações, Biblioteca, Conectores, Habilidades, Empresa, Projetos, Studio, Plugins e Configurações.
- **Evidência:** `docs/evidencias/browser-console-audit-live.json`; 56 respostas `401 Unauthorized`.
- **Exemplo:** `GET /api/agent/v1/missions`, `/mcp`, `/connector-catalog`, `/projects`, `/schedules`, `/companies`, `/supervisor/status`, `/builders` e `/config/safe`.
- **Impacto:** a UI renderiza a casca, mas não consegue provar carregamento, criação, aprovação, importação, exportação ou persistência contra o backend real.
- **Observação:** o endpoint direto `/api/agent/v1/health` também respondeu `401 {"error":"bearer token is required"}`. O servidor foi iniciado em `OLLAMA_HOST=0.0.0.0:11434`; o comando originalmente tentado com `serve :11434` é inválido.

### P1 — Rota Harnesses/Connect quebra em runtime

- **Tela/fluxo:** `/connect`, Desktop e Mobile.
- **Evidência:** `audit-live-connect-desktop.png`, `audit-live-connect-mobile.png`; console registra `TypeError: M.find is not a function` no bundle `index-CB-R3-KY.js`.
- **Impacto:** a tela não chega ao conteúdo de Harnesses/Codex; mostra apenas “Esta página encontrou um erro”.
- **Reprodução:** abrir `/connect` em viewport 1440×900 ou 390×844.

### P1 — Nenhum modelo local disponível para provar Chat/IA Manual

- **Tela/fluxo:** Home, Chat novo, Agentic e seleção Manual.
- **Evidência:** `curl http://127.0.0.1:11434/api/tags` retornou `{"models":[]}`; a Home em Manual mostra “Escolher modelo PASS”, sem modelo disponível para selecionar.
- **Impacto:** não foi possível executar uma missão/chat real, testar roteamento para modelo local, gerar artefato ou validar ciclo completo.
- **Limite de ambiente:** o log do servidor também informa que o `llama-server`/biblioteca de inferência não está construído no workspace. Isso foi registrado, não mascarado.

## Matriz por tela

| Tela/rota | Desktop | Mobile | Console/rede | Resultado live |
|---|---:|---:|---|---|
| Início/Nova tarefa `/` | PASS visual | PASS visual | limpo no carregamento | Composer, Automático/Manual, anexo e Importar projeto visíveis. Import dialog abre. Manual mostra picker sem opções confirmadas por catálogo vazio. |
| Agentic `/agentic` | PASS shell | PASS shell | 401 ×4 por viewport | Criar missão, iniciar orquestração e pesquisa ficam bloqueados pelo backend sem Bearer. `/plan` foi preenchido; missão não foi comprovada. |
| Tarefas `/tasks` | PASS shell | PASS shell | 401 `/missions` | Lista real não carrega; “Tentar novamente” não resolve sem autenticação. |
| Automações `/scheduled` | PASS shell/formulário | PASS shell/formulário | 401 `/schedules` | Formulário “Nova automação” abriu; não foi possível provar POST/listagem/pausa/execução persistida. |
| Biblioteca `/library` | PASS shell | PASS shell | 401 `/missions` | Filtros e busca aparecem, mas não há artefatos reais carregados. |
| Criações `/creations` | PASS shell | PASS shell | sem erro adicional observado | Empty state aparece; não havia criação real para validar cards/ações. |
| Computadores/Endpoint `/endpoint` | PASS | PASS | sem erro adicional observado | Host/endpoint e comandos de integração renderizam; ações externas/cloud não foram executadas sem credenciais. |
| Conectores `/connectors` | PASS shell | PASS shell | 401 em catálogo/MCP/conectores/WhatsApp | Categorias e aba WhatsApp abrem. WhatsApp mostra honestamente `NOT_CONFIGURED`; catálogo real não foi carregado. |
| Habilidades `/skills` | PASS shell | PASS shell | 401 `/skills` | Empty/error state com “Tentar novamente”; nenhum catálogo real comprovado. |
| Empresa `/company` | PASS shell | PASS shell | 401 `/companies` e supervisor | Campos de criação e controles aparecem; criação/tick/supervisor não foram persistidos. |
| Projetos `/projects` | PASS shell/formulário | PASS shell/formulário | 401 `/projects` | Novo projeto e diálogo Importar projeto abrem; importação pública/ZIP não pôde concluir por auth. |
| Studio `/studio` | PASS shell | PASS shell | 401 `/builders` | Editor vazio, componentes, Undo/Redo, Preview, Exportar ZIP e Deploy aparecem. O banner informa que o projeto não carregou por login exigido; edição/export real não comprovados. |
| Configurações `/settings` | PASS parcial | PASS parcial | 401 ×8 por viewport | Agentic Control Center fica sem dados. Foram vistos controles em inglês (“Browse”, “Start ChatGPT”, “Reset to defaults”) em uma tela predominantemente pt-BR. |
| Provedores `/providers` | PASS shell | PASS shell | sem erro adicional observado | Empty state sem provedor remoto; não foi possível validar conexão/modelos. |
| Harnesses `/connect` | **FAIL** | **FAIL** | `TypeError: M.find is not a function` | Tela genérica de erro; conteúdo não acessível. |
| Plugins `/plugins` | PASS shell | PASS shell | 401 em catálogo/configuração | Registro avançado aparece, mas dados e configuração real ficam bloqueados. |
| Chat novo `/c/new` | PASS shell | PASS shell | nenhuma chamada até envio | Composer e ações rápidas renderizam; não enviado porque não havia modelo local nem sessão autenticada. |

## Interações realizadas

- **Home:** abriu Manual; abriu Importar projeto; confirmou visualmente o picker Manual e o diálogo; anexos não foram enviados porque não havia arquivo de teste solicitado e o foco era auditoria sem alterar estado.
- **Agentic:** preencheu `/plan auditoria live segura` e tentou iniciar; o backend respondeu 401.
- **Automações:** abriu o formulário de nova automação; não submeteu por causa do bloqueio de autenticação e para não criar dado sem sessão.
- **Conectores:** abriu a categoria WhatsApp; confirmou status `NOT_CONFIGURED` e allowlist vazia.
- **Projetos:** abriu o diálogo de importação; não concluiu GitHub/ZIP porque a API real estava 401.
- **Studio:** clicou em Card, Undo, Redo, Preview e Exportar ZIP; o projeto não carregou e o backend de builders estava 401. Não houve export falso declarado como sucesso.
- **Configurações:** navegou entre Geral, Provedores, Computadores & API e Conectores/Plugins.
- **Todas as rotas:** foram abertas em ambos os viewports; inventário de botões, links, inputs, interações e erros está no JSON de auditoria.

## Botões/ações sem validação de sucesso ou bloqueados

- Agentic: `Criar missão`, `Iniciar orquestração`, `Pesquisar` e submissão de `/plan` — bloqueados por 401.
- Tarefas: `Tentar novamente` — não resolve sem token.
- Automações: `Criar agendamento`, `Criar gatilho`, `Começar` — não submetidos/persistidos por 401.
- Conectores: `Gerenciar Conectores`, `Criar Conector`, categorias e MCP — shell funciona, catálogo real bloqueado por 401.
- Empresa: `Atualizar`, `Ativar`, `Ciclo Manual (Tick)`, `Criar empresa e tenant` — não persistidos por 401.
- Projetos: `Criar`, `Importar projeto`, GitHub e ZIP — diálogo funciona; importação real bloqueada por 401.
- Studio: adicionar componentes, mover/editar, Undo/Redo, Preview, `Exportar ZIP` e `Deploy` — não comprovados com projeto carregado; Deploy também é ação externa e requer alvo/approval.
- Endpoint: criação de computador/cloud e conexões externas — não executadas sem credencial/alvo; não foram tratadas como sucesso.
- WhatsApp: configuração/allowlist — UI mostra `NOT_CONFIGURED`; não foi inventada conexão.
- `/connect`: todos os controles ficam indisponíveis porque a rota quebra antes de renderizar o conteúdo.

## Acessibilidade e responsividade observadas

- A Home expõe labels/aria-labels úteis para textarea, anexo e controles de IA; o foco visual e o layout foram capturados em 1440×900 e 390×844.
- O layout principal se adapta ao mobile, mas a auditoria não declara WCAG completa: ainda seria necessário executar uma bateria de teclado/AT com a autenticação funcionando.
- O erro de `/connect` ocorre nos dois viewports, portanto não é somente problema mobile.
- A presença de controles em inglês dentro de Configurações pt-BR é uma inconsistência de conteúdo/i18n P2.

## O que não foi possível verificar

1. Missão `/goal` ponta a ponta, SSE, aprovação HITL e artefato com checksum: backend 401 e nenhum modelo local.
2. Importação real de repositório público GitHub e ZIP grande: diálogo abre, mas endpoints de projeto exigem Bearer.
3. Persistência de schedule, Company OS/supervisor, conectores/MCP e allowlist WhatsApp: APIs 401 ou credenciais externas ausentes.
4. Studio real com edição CAS, preview sandboxed, export ZIP e deploy: builder 401; Deploy não seria executado sem alvo/approval.
5. Chat e roteamento por modelo: `/api/tags` retornou lista vazia.
6. Fluxos destrutivos ou externos (excluir, publicar, gastar, enviar WhatsApp, cloud) não foram acionados, pois exigem credencial/aprovação e não devem ser simulados.

## Evidências

- Auditoria estruturada, console e rede: [`browser-console-audit-live.json`](../docs/evidencias/browser-console-audit-live.json)
- Home Desktop: [`audit-live-home-desktop.png`](../docs/evidencias/audit-live-home-desktop.png)
- Home Manual Desktop: [`audit-live-home-manual-desktop.png`](../docs/evidencias/audit-live-home-manual-desktop.png)
- Importação Home Desktop: [`audit-live-home-import-desktop.png`](../docs/evidencias/audit-live-home-import-desktop.png)
- Studio Desktop: [`audit-live-studio-actions-desktop.png`](../docs/evidencias/audit-live-studio-actions-desktop.png)
- Conectores/WhatsApp Desktop: [`audit-live-connectors-whatsapp-desktop.png`](../docs/evidencias/audit-live-connectors-whatsapp-desktop.png)
- `/connect` Desktop com erro: [`audit-live-connect-desktop.png`](../docs/evidencias/audit-live-connect-desktop.png)
- Configurações Desktop: [`audit-live-settings-desktop.png`](../docs/evidencias/audit-live-settings-desktop.png)
- O diretório contém as capturas Desktop/Mobile das 17 rotas e estados-chave correspondentes.

## Conclusão honesta

A auditoria live **não aprova 100% de paridade**. A casca visual e vários estados honestos existem, mas o backend agentic não está utilizável pelo navegador sem um token Bearer, `/connect` tem crash determinístico e não há modelo local disponível. Esses bloqueios devem ser corrigidos e a auditoria E2E repetida antes de declarar os fluxos funcionais.


## AUD-FIX-1 — correções verificadas

- **Logout:** `AppSidebar` agora chama o `POST /api/signout` real e encerra a sessão antes de fechar o modal; regressão em `audfix1.contracts.test.ts`.
- **Perfil:** Sidebar e Settings usam `POST /api/me`, removendo o 405 causado pelo método incorreto.
- **Biblioteca:** os downloads usam `GET /api/agent/v1/missions/:id/artifacts/:artifact_id`; o sufixo inexistente `/content` foi removido.
- **Contratos:** `scripts/verify-contracts.mjs` agora considera método HTTP, templates condicionais/multilinha, parâmetros posicionais, wrappers e links de download. O gate terminou com `CONTRACT VERIFICATION PASSED`; avisos de rotas órfãs permanecem informativos.
- **Gates locais:** `npx tsc -b`, lint, Vitest (38 arquivos/259 testes), build Vite, `go build ./...`, `go test ./internal/agent ./server` e integrity passaram.

A evidência live da auditoria original continua válida como diagnóstico; uma nova captura autenticada de logout/perfil/download depende de uma sessão Bearer real e não foi inventada nesta rodada.
