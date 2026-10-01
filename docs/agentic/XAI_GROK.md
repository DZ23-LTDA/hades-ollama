# xAI / Grok no Ollama Full

O Ollama Full pode usar a **API xAI** como provider OpenAI-compatible. Isso é diferente de incorporar o produto hospedado Grok Bot: o primeiro é uma API configurável pelo operador; o segundo é um serviço com computadores cloud, browser, filesystem, terminal, bots persistentes e rotinas próprios da xAI/Cursor.

## Preset

Use [`examples/dz23-xai.json`](../../examples/dz23-xai.json):

```bash
export OLLAMA_DZ23_CONFIG="$PWD/examples/dz23-xai.json"
export OLLAMA_DZ23_GATEWAY_KEY='chave-do-gateway-fora-do-repositorio'
export XAI_API_KEY='chave-xai-fora-do-repositorio'
```

O modelo lógico é `xai/grok-4.7`. O preset permite `/v1/responses` e `/v1/chat/completions`. A chave xAI é aplicada server-side, e o cliente precisa autenticar no gateway quando `gateway_api_key_env` estiver configurado.

## Capacidades e limites

A documentação pública xAI descreve Responses API, tool calling, web search, structured outputs, modelos de texto/código, Voice API e Imagine API. O proxy atual encaminha o corpo de Responses sem reimplementar os tools xAI; assim, os campos e tool types aceitos dependem da versão da API xAI configurada. A auditoria local comprova path, substituição do model id e bearer server-side em um upstream de teste, mas não comprova conta, quota, billing, modelo ou tool xAI reais.

Para aproximar a jornada do Grok Bot com o runtime Ollama Full, a composição recomendada é:

- xAI Responses como provider de raciocínio/código;
- Browser Operator do Ollama Full para browser e takeover;
- sandbox/desktop companion para filesystem, terminal e computer use;
- memória, skills, scheduler, approvals e artifacts próprios;
- Company OS para departamentos, metas, campanhas e orçamentos;
- Composio ou connectors dedicados para apps externos.

Não se deve afirmar que essa composição é o Grok Bot, nem que possui o computador cloud, sessões ou políticas internas do serviço xAI.

Fontes oficiais: [xAI API overview](https://docs.x.ai/overview), [Grok Bot overview](https://docs.x.ai/grok-bot/overview), [xAI tools](https://docs.x.ai/developers/tools/overview).


## Grok Live no runtime

Além do preset multi-provider, o runtime expõe `GET /api/agent/v1/grok/status` e `POST /api/agent/v1/grok/responses`. O primeiro nunca retorna a chave e informa somente estado sanitizado. O segundo usa `XAI_API_KEY` no processo do servidor, aceita `input`, `tools` e streaming conforme o adapter, e falha fechado quando o modelo ou a credencial não está configurado.

O cliente local implementa retries limitados, circuito de falha e registro de latência. Isso fornece a superfície operacional para usar a API xAI em missões, mas não incorpora o Grok Bot hospedado nem prova disponibilidade de web search, Voice, Imagine ou quota sem um smoke autorizado.
