

## AUD-FIX-3 / T2 — Endpoint e ações remotas — 2026-10-01 03:15 UTC

- **Achado:** a tela Computadores/Endpoint apresentava ações sem backend configurado (criar nuvem, solicitar acesso remoto, conectar computador e controlar pelo telefone), com alertas locais que sugeriam autorização/execução.
- **Correção:** as quatro ações agora são controles `disabled`, rotulados explicitamente como **não configurado**, com `title` explicando a dependência e mensagem `NOT_CONFIGURED`; a mutação `remoteRequested` e os `alert()` foram removidos. O host local continua sendo exibido apenas com dados reais de `/api/v1/host`, `/api/version` e `/api/tags`.
- **Evidência:** `docs/evidencias/screen-night-t2-endpoint-desktop.png` (1440x900), `screen-night-t2-endpoint-mobile.png` (390x844) e `browser-console-audit-night-t2.json`.
- **Resultado Playwright:** desktop e mobile renderizaram sem página de erro; console e HTTP errors vazios; quatro controles remotos desabilitados; nenhum texto `Acesso Remoto Autorizado`.
- **Estado honesto:** Endpoint permanece `PARCIAL`: host local validado; recursos remotos continuam `NOT_CONFIGURED` até existir endpoint, pareamento, credencial e aprovação explícitos.


### T3–T8 — fechamento/reclassificação — 2026-10-01 06:34 UTC
- Criações/deploy: reclassificado `NOT_EXECUTED`; não há claim de publicação, rollback ou health sem adapter/credencial real.
- Notificações: alerta fictício removido; estado vazio honesto, backend persistente ainda ausente.
- i18n: controles de configuração Claude/ChatGPT traduzidos para pt-BR; nomes próprios e integrações permanecem quando semanticamente necessários.
- Automações: UI distingue intervalo de webhook e referencia segredo; entrega externa permanece condicionada à configuração.
- Código morto: kinds inacessíveis e mensagem fixa de reinício removidos.
- T8: contratos, integrity, vet, Go nativo/CGO0 e gates frontend passaram. Os 401 em rotas protegidas sem Bearer permanecem registrados, não mascarados.
