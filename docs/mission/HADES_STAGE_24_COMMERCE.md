# Estágio 24 — Commerce Engine

## 1. O que foi entregue

Catálogo, **livro de estoque**, pedidos com checkout idempotente, quitação por
**referência** e conciliação — em sandbox, sem processar cartão.

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| **ESTOQUE É UM LIVRO DE MOVIMENTOS, não um contador** | o saldo é **derivado** dos lançamentos. Contador mutável perde atualização sob concorrência e não permite auditar quem mexeu; o movimento permite as duas coisas |
| **Reserva não reduz o FÍSICO** | `disponível = físico − reservado`. Somar a reserva no físico faria o saldo **mentir sobre o que existe no depósito** |
| **IDEMPOTÊNCIA DE CHECKOUT sob o mesmo lock** | reenviar o mesmo checkout **não** cria dois pedidos. Em comércio isso é a diferença entre **cobrar uma vez e cobrar duas** |
| **Sem chave de idempotência, o pedido é RECUSADO** | sem chave não há como garantir unicidade, e um retry viraria venda duplicada |
| **NÃO SE VENDE O QUE NÃO EXISTE** | o pedido **reserva** na criação e falha se não houver disponível |
| **O preço vem do CATÁLOGO, nunca do pedido** | o cliente **não escolhe quanto paga**; o motor ignora o preço enviado |
| **ZERO DADO DE CARTÃO** | métodos são **referências** (sandbox, Pix, boleto, manual). Referência com aparência de número de cartão é **recusada** — o estágio proíbe processar cartão sem conformidade, e o sistema não guarda PAN nem por acidente |
| **Total e valor conferidos** | pagamento com valor diferente do pedido é erro, não arredondamento silencioso |
| **Conciliação idempotente** | mesma referência é no-op; outra é conflito |
| **Atendimento consome o físico; cancelamento devolve** | pedido atendido **não** pode ser cancelado |

## 3. Um erro meu, encontrado antes do teste

Ao derivar o saldo, eu havia somado o delta da **reserva** no estoque **físico** — o que
faria o físico cair no momento da reserva e **mentir sobre o depósito** (a mercadoria
continua na prateleira). Corrigi a modelagem para `disponível = físico − reservado` e o
teste passou a travar exatamente isso: reservar 4 de 4 deixa `físico = 4`, `reservado = 4`
e `disponível = 0`.

## 4. Alcançável pelo produto

`POST/GET /commerce/products` · `POST /commerce/stock` · `GET /commerce/stock/:sku`
(com o livro de movimentos) · `POST /commerce/orders` (**exige `Idempotency-Key`**) ·
`GET /commerce/orders` · `GET /commerce/orders/:id` · `POST .../payment` (referência) ·
`POST .../reconcile` · `POST .../fulfill` · `POST .../cancel` · `GET /commerce/summary`.

Códigos: **409** estoque insuficiente, conflito de estado e conciliação divergente ·
**422** dado de cartão · **404** produto/pedido inexistente.

## 5. Efeito nos agentes (estágio 35)

`commerce.order.read` e `commerce.stock.read` **deixaram de ser bloqueadas** para o agente
de comércio, porque agora existem de verdade — e há teste verificando que o catálogo não
lista mais essas ferramentas como bloqueadas.

## 6. Verificação

- **7 testes de motor**: idempotência do checkout (mesmo pedido na reentrega, um só pedido
  criado, uma só reserva; sem chave recusa; chave diferente é pedido novo) · **não vender o
  que não existe** (5 de 4 falha; reservar tudo zera o **disponível** sem zerar o físico;
  movimento manual além do disponível falha) · livro de estoque separando físico e
  reservado (cancelamento devolve, atendimento reduz o físico, **todos** os tipos de
  movimento registrados, pedido atendido não cancela) · **recusa de dado de cartão** (três
  formatos de PAN) e de método desconhecido, com valor divergente recusado · **preço do
  catálogo** ignorando o preço enviado + conciliação idempotente com conflito · isolamento
  por organização · **idempotência sob concorrência** (dez checkouts simultâneos ⇒ **um**
  pedido e **uma** unidade reservada).
- **2 testes de rota**: catálogo, estoque, checkout idempotente, 400 sem chave, 409 sem
  estoque, 404 produto inexistente, pagamento por referência, **422** dado de cartão,
  conciliação, atendimento, 409 ao cancelar pedido atendido e resumo · ferramentas de
  comércio não bloqueadas.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** (aplicado
  `gofumpt` do próprio linter) · suítes apenas com as **falhas ambientais conhecidas**.

## 7. Limites declarados

1. **Sem emissão fiscal** (NF-e) e sem integração contábil.
2. **Sem cobrança real**: Pix/boleto são **referências** registradas; não há integração com
   PSP, nem transferência de dinheiro.
3. **Sem carrinho persistente** nem recuperação de carrinho abandonado.
4. **Sem frete, entrega e devolução** implementados.
5. **Sem sincronização com marketplace/ERP** (estágios 25 e 36).
6. **Persistência em memória** nesta rodada (o padrão de arquivo por organização dos outros
   módulos é o caminho para a próxima).
