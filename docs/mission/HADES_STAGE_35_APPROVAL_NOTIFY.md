# Estágio 35 — aviso opt-in do pedido pendente (E-156)

## 1. O limite que esta rodada fecha

O estado declarava: *"não há push; a fila é visível, mas ninguém é avisado ativamente"*. Agora existe um
**aviso por webhook** — opcional, desligado por padrão, e com quatro decisões deliberadas.

| Decisão | Por quê |
| --- | --- |
| **DESLIGADO por padrão** | sem destino configurado **nada sai da máquina**; avisar "por padrão" para um destino inventado seria mandar dado de organização para terceiro sem pedido |
| **Falha NÃO bloqueia** | o controle é a aprovação humana; um webhook fora do ar que travasse escrita trocaria um problema de operação por um incidente |
| **Payload mínimo** | identificador, ferramenta, valor e quantas decisões faltam — **sem** impressão digital dos parâmetros, sem contato, sem conteúdo de conversa: um webhook vaza o que carrega |
| **Segredo só no cabeçalho** | `X-Hades-Signature: sha256=<HMAC>` sobre o corpo; o erro descreve a falha **por classe** (status, tempo esgotado, destino não respondeu, destino inválido) e **nunca** a URL |

Configuração por ambiente, sem tocar código: `HADES_APPROVAL_WEBHOOK_URL` e `HADES_APPROVAL_WEBHOOK_SECRET`.

## 2. Um aviso por pedido — não um por turno

O aviso dispara na **criação** do pedido, e o store **reusa** o pedido pendente para a mesma impressão
digital. Repetir o turno enquanto o humano decide não gera segundo aviso: aviso duplicado é spam para quem
decide, e spam faz a pessoa parar de olhar a fila.

O aviso acontece **depois** de persistir: avisar sobre pedido que não ficou gravado seria alarme falso.

## 3. Visível para quem opera

A rota da fila passou a devolver também o estado do aviso:

```json
"notification": { "configurado": false, "entregues": 0, "ultimo_erro": "" }
```

Sem isso, um destino fora do ar deixaria a fila cheia e **ninguém** sabendo que ninguém foi avisado.

## 4. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run 'TestServiceAgent\|TestRuntimeServiceAgents'` | **82/82 PASS** (4 novos) |
| `go test ./internal/agent/ -count=1` | **ok 63,459s** |
| `go test ./server/ -count=1` | **ok 21,715s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agente 22,791s · servidor 6,496s) |
| `gofmt -l internal/agent server` | **sem saída** |

Os testes novos: **sem destino, zero avisos**; com destino, **exatamente um** aviso por pedido novo, com
corpo mínimo (**recusa explícita** de `fingerprint`, `conversation_id`, `contact`, `title` e do segredo) e
**assinatura conferida** contra HMAC; **repetir o turno não reenvia**; **destino com erro 500 não bloqueia**
— o pedido continua pendente, a decisão humana funciona e a escrita executa depois; e o notificador **não
vaza** destino nem segredo no erro.

## 5. Erro meu nesta rodada (registrado)

O primeiro `go build` falhou com `undefined: sync` e `undefined: context`: usei o mutex e o contexto antes de
acrescentar os imports. É a segunda vez nesta sequência de rodadas que o import fica atrás do código — e o
compilador pegou nos dois casos. A lição já institucionalizada continua valendo: **compilar é barato e
imediato; supor não**.

## 6. Limites declarados

- **Uma tentativa, sem retry nem backoff**: se o destino estiver fora no instante do aviso, ele não é
  reenviado. A fila continua mostrando o pedido.
- **Só na criação**: um pedido que **envelhece** não gera segundo aviso (o atraso aparece na fila, não no
  webhook).
- **Sem canal de e-mail/SMS**: o único canal é o webhook.
- **Configuração por variável de ambiente**, não pela interface; o segredo é lido do ambiente e **não** há
  rotação automática.
- **Entrega best-effort**: não há garantia de entrega, ordenação nem repetição — quem depende do aviso para
  cumprir prazo precisa olhar a fila.
