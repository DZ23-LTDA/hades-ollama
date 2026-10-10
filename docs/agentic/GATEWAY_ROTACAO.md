# Gateway de inferência e rotação automática

O Ollama Full pode atuar como o roteador central de inferência do seu ambiente,
no mesmo papel que um OmniRoute: ele recebe as chamadas no formato nativo do
cliente (Claude, Codex, OpenAI), decide qual provedor atende e, quando o
provedor escolhido recusa a requisição, tenta a próxima alternativa elegível
sem devolver erro ao cliente.

## Três protocolos, um roteador

| Cliente | Rota que ele usa | Middleware |
| --- | --- | --- |
| Claude Code / SDK Anthropic | `POST /v1/messages` | `AnthropicMessagesMiddleware` |
| Codex | `POST /v1/responses`, `POST /v1/responses/compact` | `responsesCompactionMiddleware` |
| SDK OpenAI e apps Web | `POST /v1/chat/completions`, `GET /v1/models` | `ChatMiddleware` |

As três rotas terminam no mesmo `ChatHandler`, então qualquer melhoria de
roteamento vale para os três clientes ao mesmo tempo.

## Quando a rotação acontece

A rotação só é considerada quando o provedor já escolhido falha **antes de
qualquer byte chegar ao cliente**. São rotacionáveis:

- falha de conexão, DNS ou timeout na chamada ao provedor;
- respostas `429`, `500`, `502`, `503` e `504` do provedor.

Não há rotação depois que a resposta começou. Se o provedor já emitiu o status
e parte do corpo, o gateway repassa o que veio e encerra: repetir entregaria ao
cliente duas respostas parciais diferentes.

Provedores do tipo `cli` nunca participam da rotação, nem como origem nem como
destino. A execução de um processo externo tem efeito colateral e não pode
acontecer duas vezes por causa de uma tentativa repetida.

## Configuração

```json
{
  "gateway_api_key_env": "OLLAMA_DZ23_GATEWAY_KEY",
  "rotation": {
    "enabled": true,
    "max_attempts": 3,
    "cross_provider": false
  },
  "providers": []
}
```

- `rotation.enabled` — ausente equivale a `true`. Defina `false` para voltar a
  tentativa única.
- `rotation.max_attempts` — total de tentativas, incluindo a primeira.
  Ausente equivale a `3`, e o valor é limitado a `5` para não multiplicar o
  custo de uma requisição que falha em série.
- `rotation.cross_provider` — ausente equivale a `false`. Um modelo fixado
  explicitamente pelo cliente (por exemplo `openrouter/coder`) só aceita
  rotação para **outro** provedor quando o operador autoriza. Sem essa
  autorização, o gateway respeita a escolha e devolve o erro do provedor.

Um exemplo completo, com API de terceiros e um provedor de reserva no mesmo
caminho, está em `examples/dz23-hades-gateway.json`. O teste
`TestShippedGatewayExampleLoadsAndEnablesRotation` carrega esse arquivo a cada
execução da suíte, então ele não sai de sincronia com o schema.

### Provedor de assinatura por CLI

Um provedor `"type": "cli"` roda um binário já autenticado na sua máquina, o que
serve para reaproveitar assinatura em vez de chave de API:

```json
{
  "name": "subscription",
  "type": "cli",
  "executable": "/usr/local/bin/claude",
  "allow_execution": true,
  "timeout_seconds": 600,
  "models": [{ "id": "claude-code", "harness_id": "claude-code" }]
}
```

`executable` precisa ser um caminho **absoluto** e `allow_execution` precisa ser
`true`; o registro recusa o carregamento caso contrário. Como o caminho varia por
sistema operacional, o exemplo publicado não traz provedor `cli` — copie o bloco
acima e ajuste o caminho.

O processo filho não herda o ambiente do servidor. Ele recebe apenas uma lista
fixa de variáveis básicas (`PATH`, `HOME`, `SystemRoot`, `USERPROFILE` e
equivalentes) mais a `api_key_env` declarada e o que você listar em
`passthrough_env`. É por esse motivo que a chave de um provedor nunca vaza para
o binário de outro.

Um provedor `cli` também não pode declarar `/v1/messages` nem
`/v1/embeddings` — o registro recusa o carregamento. Para atender o Claude pelo
protocolo nativo use um provedor do tipo `anthropic`.

## Como escolher os candidatos

A primeira tentativa continua sendo a do roteamento normal: para requisições
com alias (`auto`, `auto/coding`), o `Resolve` do registro escolhe por
prioridade. Depois disso, o gateway monta a lista de alternativas a partir do
mesmo motor de pontuação usado pelo restante do produto, com três filtros:

1. o candidato precisa declarar a mesma `allow_private` do provedor principal,
   para que uma requisição privada nunca saia para a internet;
2. o candidato precisa declarar a mesma capacidade exata do modelo principal,
   sem resposta degradada;
3. o candidato não pode ser um modelo `cli` nem o mesmo modelo já tentado.

Esgotadas as tentativas, o gateway repassa ao cliente a última falha do
provedor — status, cabeçalho e corpo originais — em vez de inventar um `502`.

## Modelo local

Modelos servidos pelo próprio Ollama não precisam de entrada em `providers`.
Quando o nome pedido não pertence ao registro, o middleware não intervém e o
Ollama responde normalmente pela rota nativa. Não declare um provedor apontando
para `http://127.0.0.1:11434`, isso cria um laço do gateway para si mesmo.

## Segurança

- `Resolve` nunca cai para um provedor remoto quando a política é `LocalOnly`.
- O provedor remoto exige `Authorization` válido quando
  `gateway_api_key_env` está configurado; sem chave configurada, só chamadas
  locais (loopback) são aceitas.
- Cada tentativa descarta os cabeçalhos do provedor anterior antes de
  repassar os do novo, para que nenhum `Retry-After`, `Set-Cookie` ou
  cabeçalho de limitação de um provedor derrotado vaze para o cliente.
