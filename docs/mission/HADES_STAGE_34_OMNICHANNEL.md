# Estágio 34 — Omnichannel Runtime: contrato canônico de mensagens

## 1. O que faltava

Havia canais (WhatsApp e Telegram), caixa de entrada e CRM — mas **nenhuma porta de
entrada para eventos externos**. O estágio pede contrato canônico com *validação de
webhooks, deduplicação, idempotência, replay protection, DLQ, rate limiting, logs
sanitizados* e a regra dura: **nunca cruzar empresas**.

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| **O TENANT VEM DO CONECTOR AUTENTICADO, nunca do payload** | um campo `tenant_id` no corpo é **dado de terceiro** e não pode escolher onde a mensagem entra. É o que impede cruzamento entre empresas |
| **HMAC-SHA256 com comparação em tempo constante** (`hmac.Equal`) | comparar assinatura com `==` vaza informação pelo tempo de resposta |
| **Sem segredo configurado, o webhook é RECUSADO** | aceitar não assinado "porque não há chave" transformaria o endpoint em **porta aberta** |
| **Anti-replay por janela de tempo** (±5 min) | a assinatura de um evento antigo continuaria válida para sempre sem isso |
| **Deduplicação por `ExternalMessageID`** | uma proteção **não** cobre a outra: replay é sobre **quando**, dedupe é sobre **o quê**. As duas existem |
| **DLQ com motivo e tentativas** | evento que falha vai para a fila de mortos **com o motivo**; o reprocessamento é explícito — nada é descartado em silêncio |
| **Rate limit por organização e canal** | limita abuso de um conector sem penalizar os outros |
| **A mensagem entra pela MESMA caixa de entrada** | um segundo caminho de mensagem poderia divergir do primeiro — o defeito de irmão esquecido |
| **Log sanitizado** | o evento guarda **prévia curta**; o **segredo e a assinatura nunca** são guardados nem devolvidos |

## 3. Contrato e códigos de resposta

`POST /omnichannel/webhooks/:channel` responde:

| Situação | Código |
| --- | --- |
| Aceito | **202** |
| Reentrega deduplicada | **200** |
| Assinatura ausente/inválida | **401** |
| Carimbo fora da janela (replay) | **409** |
| Limite excedido | **429** |
| Canal sem conector | **404** |
| Evento inválido | **400** |

**Por que 401 e 409 são separados:** a assinatura inválida pede ao provedor *corrigir a
assinatura*; o carimbo fora da janela pede *rever relógio/retentativa*. Juntar os dois
mandaria o provedor **depurar a coisa errada** — foi o que meu próprio teste pegou.

Rotas de apoio: `POST/GET /omnichannel/connectors` (só o **nome** da variável de
ambiente é exposto), `GET /omnichannel/events`, `GET /omnichannel/dlq`,
`POST /omnichannel/dlq/:id/retry`.

## 4. Verificação

- **6 testes de motor**: assinatura ausente, inválida e **de outro corpo** recusadas, e
  **falha fechada sem segredo** · anti-replay (evento antigo e carimbo futuro) + dedupe
  (reentrega devolve o mesmo evento e a conversa continua com **uma** mensagem) ·
  **tenant vem do conector** (payload dizendo outra organização é ignorado; a outra
  organização não vê o evento; canal sem conector é recusado) · rate limit (aceita 3,
  recusa o 4º) e DLQ inexistente para evento recusado na validação ·
  **reprocessamento explícito** da DLQ que esvazia a fila, mantém histórico, falha com
  id inexistente e é escopado por organização · **segredo e assinatura nunca
  guardados** nem no evento nem no conector exposto.
- **1 teste de rota**: registro do conector (resposta **sem** o valor do segredo),
  401 sem assinatura, 202 com assinatura válida, 200 na reentrega deduplicada, **409**
  fora da janela, canal sem conector recusado, listagens escopadas e **404** no retry
  de DLQ inexistente.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** · suítes
  apenas com as **falhas ambientais conhecidas**.

## 5. Um erro meu, corrigido pelo teste

Eu tinha mapeado **replay e assinatura para o mesmo 401**. O teste exigia **409** para
replay, e ele estava certo: os dois pedem ações diferentes do provedor. Separei os
códigos em vez de ajustar o teste.

## 6. Limites declarados

1. **Sem entrega de saída ao provedor**: o runtime **recebe** eventos; enviar por
   conector externo depende de credencial (dependência externa).
2. **Rate limit em memória**: não é distribuído entre instâncias; um limite global
   exigiria armazenamento compartilhado (Redis), que existe no projeto mas não foi
   ligado aqui.
3. **DLQ em memória**: sobrevive enquanto o processo vive; persistência da fila de
   mortos é trabalho futuro.
4. **Sem mídia**: eventos carregam referência de anexo, mas o download/armazenamento
   de mídia não foi implementado.
5. **Sem rotação de segredo**: o segredo vem do ambiente; rotação exige reinício.
