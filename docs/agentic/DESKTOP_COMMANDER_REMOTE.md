# Desktop Commander Remote MCP

O [Desktop Commander](https://desktopcommander.app/) oferece acesso de IA a arquivos, terminal, processos e máquinas reais. O repositório oficial [Remote Desktop Commander](https://github.com/desktop-commander/remote-desktop-commander) documenta um servidor MCP hospedado em `https://mcp.desktopcommander.app/mcp`, usando Streamable HTTP e OAuth 2.0 com PKCE. Um agente local é iniciado no computador controlado com `npx @wonderwhy-er/desktop-commander@0.2.51 remote`; revise conscientemente a versão antes de atualizá-la. A máquina fica acessível somente enquanto esse processo está em execução.

A documentação oficial lista leitura e busca de arquivos, edição, processos, sessões persistentes, informações de dispositivos, pareamento e revogação. O serviço remoto está em beta e sua implementação hospedada é proprietária; o servidor local `DesktopCommanderMCP` é o projeto open source relacionado. O agente executa com as permissões do usuário da máquina, portanto o Ollama Full deve manter allowlist, escopos, approvals para escrita, execução, processos e ações externas, além de revogação e auditoria.

Este arquivo registra a integração como um preset configurável. Não realiza login, pareamento ou conexão de dispositivo automaticamente. Credenciais OAuth, conta, máquina pareada e processo `npx` continuam responsabilidade do operador.

## Referências

[1]: https://desktopcommander.app/ "Desktop Commander — Use your computer from chat"
[2]: https://github.com/desktop-commander/remote-desktop-commander "Remote Desktop Commander — official remote MCP server"
[3]: https://github.com/desktop-commander/remote-desktop-commander/blob/main/docs/SETUP.md "Remote Desktop Commander setup guide"

## Presets no Ollama Full

Para usar o servidor local stdio, instale Node.js 18 ou superior e configure `OLLAMA_AGENT_MCP` apontando para [`examples/dz23-desktop-commander-mcp.json`](../../examples/dz23-desktop-commander-mcp.json). O exemplo fixa `@wonderwhy-er/desktop-commander@0.2.51`; revise conscientemente essa versão antes de atualizá-la. O runtime ainda aplica timeout, métodos allowlisted e approval na tool `mcp.call`.

```bash
export OLLAMA_AGENT_MCP="$PWD/examples/dz23-desktop-commander-mcp.json"
```

Para o serviço Remote MCP, configure `OLLAMA_AGENT_REMOTE_MCP` apontando para [`examples/dz23-desktop-commander-remote.json`](../../examples/dz23-desktop-commander-remote.json). O endpoint oficial é HTTPS e usa Streamable HTTP. O adapter aceita bearer token somente se o operador provisionar `DESKTOP_COMMANDER_ACCESS_TOKEN` fora do repositório:

```bash
export OLLAMA_AGENT_REMOTE_MCP="$PWD/examples/dz23-desktop-commander-remote.json"
export DESKTOP_COMMANDER_ACCESS_TOKEN='valor-fora-do-repositorio'
```

O serviço oficial documenta OAuth 2.0 com PKCE e pareamento por device flow. Esta rodada adiciona o transporte Remote MCP e o preset, mas não automatiza login OAuth, criação de conta, pareamento ou revogação no dashboard. Sem uma sessão OAuth/bearer válida, a chamada remota deve falhar de forma explícita e não deve ser classificada como conexão ativa.
