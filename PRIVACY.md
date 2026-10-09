# Política de privacidade — Hades

**Última atualização: 2026-10-01**

O Hades é um aplicativo **local-first** mantido por DZ23-LTDA. Esta política descreve o comportamento do código distribuído neste repositório; ela não transforma integrações opcionais em serviços hospedados nem promete que um provedor externo tratará dados de forma específica.

## Dados processados

- Conversas, missões, artefatos, projetos, preferências e credenciais são armazenados localmente por padrão, no diretório de dados configurado pelo operador.
- O runtime pode armazenar eventos, auditoria e metadados necessários para recuperação, isolamento de tenant e approvals.
- Segredos de provedores são aceitos apenas por configuração explícita e não devem ser colocados em issues, commits, prompts públicos ou logs.
- O projeto não inclui telemetria obrigatória, rastreador de publicidade ou analytics hospedado por padrão.

## Saídas externas

Uma requisição externa somente ocorre quando o operador configura e habilita um provider, connector, MCP, WhatsApp, deployment ou outro adapter. O transporte passa pela política de egress do runtime; credenciais são encaminhadas apenas ao host aprovado e dados sensíveis são redigidos antes de logs, artifacts ou memória.

Integrações sem credencial, conta ou smoke autorizado permanecem `NOT_CONFIGURED`, `BLOCKED_EXTERNAL` ou `NOT_EXECUTED`; o aplicativo não finge conexão, publicação, pagamento, entrega ou homologação.

## Controle do operador

O operador pode desligar providers, remover credenciais, revogar sessões e apagar os dados locais pelo sistema operacional. Ações sensíveis — gasto, publicação, mensagens externas e alterações de acesso — exigem approval server-side quando aplicável.

Importar um projeto não executa o código importado. Testes/builds de terceiros exigem sandbox e approval conforme a configuração do runtime.

## Responsabilidade por serviços de terceiros

Quando o operador usa OpenAI, Anthropic, GitHub, WhatsApp, SignPath, Composio ou qualquer outro serviço, o processamento também fica sujeito aos termos e à política de privacidade desse serviço. O Hades não é o controlador desses serviços nem promete retenção, residência ou exclusão que o provedor não ofereça.

## Segurança e contato

Relate vulnerabilidades sem incluir dados pessoais ou segredos usando o canal descrito em [`SECURITY.md`](SECURITY.md). Para dúvidas sobre o projeto, abra uma issue no [repositório Hades](https://github.com/DZ23-LTDA/hades-ollama) sem anexar credenciais.

## Histórico

Mudanças relevantes desta política devem ser registradas no histórico do repositório e revisadas pelos mantenedores antes de uma release pública.
