# Auditoria dos comandos nativos (§6) e a guarda do §11.1

## 1. O retrato medido

| Situação | Quantidade |
| --- | --- |
| Comandos no catálogo | **35** |
| Com rota declarada **e existente** | **32** |
| Indisponíveis com motivo declarado | **3** (`/support`, `/inbox`, `/handoff`) |

Antes desta rodada eram **30 disponíveis** e **5 indisponíveis**. Duas correções
mudaram o quadro — e as duas eram defeitos reais, em **direções opostas**.

## 2. O comando que escondia capacidade real (`/research`)

A rota `POST /api/agent/v1/research` **existia**, com `ResearchEngine` completo
(busca, fetch SSRF-safe, citações, conflitos). Mesmo assim o comando estava registrado
como **indisponível**, com o motivo *"pesquisa não implementada"*. Isso é o **inverso**
do problema do `/crm`: não era uma rota prometida e ausente, era **capacidade real
escondida do usuário**.

Corrigido: `/research` passou a apontar para `POST /api/agent/v1/research`, e o teste
de rota **executa a pesquisa de verdade** contra um servidor local, conferindo que o
conteúdo retornado é o da fonte.

## 3. O comando que faltava integrar (`/telegram`)

O canal Telegram **existia** (envio com DLP, verificação de identidade, allowlist,
redação de token) e as ferramentas `im.telegram.send`/`im.telegram.verify` também — mas
o comando dizia, com honestidade, *"sem rota registrada"*.

Em vez de escrever um cliente HTTP novo (o que **duplicaria** token, DLP e
consentimento), criei uma **porta exportada fina** sobre o MESMO canal:
`SendTelegramMessage` e `VerifyTelegramBot`. As rotas passaram a existir:

| Rota | Comportamento |
| --- | --- |
| `GET /telegram/status` | **honesto**: sem credencial devolve `configured=false`, `gate_status=NOT_CONFIGURED` e o **motivo real** — nunca "pronto para usar" |
| `POST /telegram/send` | reusa o canal com a ordem **validação → credencial → DLP → consentimento → egress** |

O teste de rota verifica que, sem credencial e sem consentimento, **não há sucesso** e a
resposta de erro **não** contém credencial nem `message_id` (não parece envio).

## 4. A guarda que institucionaliza o §11.1

Auditar à mão uma vez não impede a regressão. Criei
`TestEveryAvailableCommandMapsToARegisteredRoute`, que:

1. registra o roteador real;
2. para **cada** comando disponível, extrai método e caminho do `Backend` e confere
   contra as rotas **efetivamente registradas**, comparando pelo **formato** (nomes de
   parâmetro normalizados, então `:id` e `:approval_id` não criam falso negativo);
3. para **cada** comando indisponível, exige **motivo declarado** — um comando mudo e
   escondido é pior do que um comando ausente.

Este teste teria pego, sozinho, os dois defeitos desta rodada.

## 5. Um teste que codificava a verdade antiga

`TestSlashRegistryCoversSection6AndIsHonest` afirmava que `/research` e `/telegram`
**deviam** estar indisponíveis. A afirmação era correta quando o registro era o único
lugar a consultar; agora é **falsa**. Atualizei para exigir a **nova** verdade
(disponíveis **e** apontando para a rota real) em vez de remover a verificação.

## 6. Verificação

- **4 testes novos no servidor**: a guarda de contrato · a pesquisa executada de verdade
  contra um servidor local · status honesto do Telegram com envio que **não** devolve
  sucesso nem `message_id` · registro das três rotas.
- **Testes do registro atualizados** para a nova verdade.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** · suítes
  apenas com as **falhas ambientais conhecidas** (que passam no CI).

## 7. Limites declarados

1. **`/support`, `/inbox` e `/handoff` seguem indisponíveis** com motivo: dependem da
   caixa de entrada unificada (estágio 29), que não existe.
2. **Telegram real não homologado**: sem credencial de bot, a rota de envio existe mas o
   canal fica `NOT_CONFIGURED`. Homologação com bot real é dependência externa.
3. **A pesquisa depende de URLs explícitas** quando não há provedor de busca configurado:
   sem `SearchProvider`, um pedido sem URLs devolve erro honesto em vez de fingir
   resultado. Um provedor de busca real é integração externa.
4. **A guarda cobre rota, não semântica**: ela prova que a rota existe, não que o comando
   faz tudo o que o catálogo promete. Os testes de cada rota cobrem o resto.
