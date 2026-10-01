# Composio no Ollama Full

O Ollama Full possui um adapter para registrar o **Composio Connect** como Remote MCP. A integração usa o protocolo público documentado pelo Composio e não incorpora código proprietário no runtime.

## O que foi implementado

O arquivo [`examples/dz23-composio-connect.json`](../../examples/dz23-composio-connect.json) configura:

- endpoint HTTPS `https://connect.composio.dev/mcp`;
- header `x-consumer-api-key` injetado apenas no processo do servidor;
- allowlist JSON-RPC para `initialize`, `notifications/initialized`, `tools/list` e `tools/call`;
- timeout de 45 segundos;
- approval obrigatório pela tool `mcp.remote.call` já existente no runtime.

O adapter aceita headers associados a variáveis de ambiente por `headers_env`. Os nomes das variáveis podem aparecer no catálogo sanitizado, mas os valores nunca são enviados à UI, à memória, aos artifacts ou aos logs.

## Configuração local

```bash
export OLLAMA_AGENT_REMOTE_MCP="$PWD/examples/dz23-composio-connect.json"
export COMPOSIO_CONSUMER_API_KEY='valor-fora-do-repositorio'
```

Reinicie o servidor depois de alterar as variáveis. O catálogo pode ser consultado em `GET /api/agent/v1/mcp`; a presença do servidor no catálogo significa apenas que o preset foi carregado.

## Autorização de apps

O Composio Connect expõe meta-tools para procurar tools, obter schemas, gerenciar conexões e executar ações. Na primeira utilização de um app, o operador deve completar o OAuth no navegador e revisar a conta e os scopes solicitados. O Ollama Full deve registrar a organização, o usuário, o tool slug, o approval e o resultado, mas nunca o token.

Para uma futura operação multiusuário, a integração deve trocar o preset global por sessões Composio por organização/usuário, com:

1. `organization_id` e usuário autenticado vinculados à sessão;
2. secrets cifrados no credential store;
3. scopes mínimos por departamento e tool;
4. revogação e reconexão explícitas;
5. idempotência, rate limit, DLP e auditoria;
6. aprovação para qualquer efeito externo.

## Estado honesto

A integração Composio foi implementada como transporte Remote MCP e coberta por teste local de bearer/header server-side e allowlist. Não há conta Composio, API key ou connected account configurada no repositório público. A disponibilidade de uma ação depende do toolkit, da autorização da conta, da região, das políticas do upstream e dos scopes aprovados.

Fontes oficiais: [Composio documentation](https://docs.composio.dev/docs), [Composio Connect](https://docs.composio.dev/docs/composio-connect), [toolkits](https://composio.dev/toolkits).
