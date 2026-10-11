# Estágio 30 — Finance & Operations ligado ao comércio (+ persistência do comércio)

## 1. O que foi entregue

Duas coisas que se completam:

1. **PERSISTÊNCIA DO COMÉRCIO** — fecha o limite mais frágil declarado na rodada
   anterior: um arquivo por organização (catálogo, livro de estoque, pedidos, chaves de
   idempotência e a sequência de numeração).
2. **FINANCE & OPERATIONS** — deriva lançamentos do comércio no **livro financeiro que
   já existia** (`company_ledger.go`), com conciliação cruzada e divergências medidas.

**Não criei um segundo razão.** Duplicar o livro criaria duas verdades sobre o dinheiro.

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| **PEDIDO NÃO PAGO NÃO É RECEITA** | lançar receita de pedido aguardando pagamento criaria **receita fictícia**. Há teste que confere razão **zerado** com pedido pendente |
| **IDEMPOTÊNCIA POR PEDIDO** | a chave é o próprio pedido: postar duas vezes **não** duplica receita |
| **LACUNA É MEDIDA, NÃO ESCONDIDA** | pedido pago sem lançamento aparece em `unposted_revenue_cents` — é o detector de receita que "sumiu" entre comércio e financeiro |
| **DIVERGÊNCIA É DECLARADA** | pedido cancelado que já tem receita vira **estorno devido**; o lançamento **não é apagado**, porque dinheiro que entrou e foi devolvido tem história |
| **CONCILIAÇÃO CRUZADA** | a referência do extrato é gravada no **próprio lançamento**, ligando comércio e banco pelo mesmo documento; referência divergente é **409** |
| **RAZÃO É POR EMPRESA; COMÉRCIO É POR ORGANIZAÇÃO** | a resolução organização → empresa é explícita; sem empresa criada o lançamento é **recusado** em vez de ir para o vazio |
| **NENHUM DINHEIRO SE MOVE** | não há pagamento, transferência nem estorno automático — só registro e sinalização. Mover dinheiro exige autorização |
| **PERSISTÊNCIA POR ORGANIZAÇÃO** | o isolamento fica no **armazenamento**, não só na consulta: uma organização não escreve no arquivo de outra |

## 3. Alcançável pelo produto

`POST /finance/orders/:id/revenue` (**201** cria, **200** idempotente, **409** se não pago)
· `POST /finance/orders/:id/cost` · `POST /finance/orders/:id/reconcile` (**409** se
divergente) · `GET /finance/report` (razão + contas a receber + lacunas + divergências).

## 4. Dois erros meus, corrigidos pelos testes

1. **O `company_id` do recibo carregava a ORGANIZAÇÃO** — a conciliação então procurava
   uma empresa com o ID da organização e falhava com *"company not found: org-a"*. O
   campo significa **empresa**; corrigi o valor em vez de afrouxar a busca.
2. **A empresa recebe ID próprio na criação**, e eu havia fixado `"emp-1"` nos testes.
   Passei a usar o **ID devolvido** pela criação, em vez de presumir.
3. **Conflito de conciliação respondia 400**; é **409** — ajustei o mapeamento de status.

## 5. Verificação

- **4 testes financeiros**: receita recusada para pedido não pago (com razão zerado) e
  aceita após o pagamento, batendo com o total e a moeda · **receita única por pedido**
  (repetição devolve o mesmo lançamento; custo idempotente; saldo conferido) ·
  **lacuna medida** (pago sem lançamento ⇒ `unposted_revenue_cents`, depois falta de
  conciliação, depois zero) + **estorno devido** com o lançamento **preservado** ·
  contas a receber e escopo por organização (com recusa quando não há empresa).
- **1 teste de persistência**: reinício recupera pedido (com pagamento), estoque físico e
  reservado, catálogo **e a chave de idempotência** — reenviar o mesmo checkout após o
  reinício devolve o **mesmo** pedido, sem criar um segundo.
- **1 teste de rota**: 409 para pedido não pago, 201/200 na idempotência, custo, 409 na
  conciliação divergente, 400 sem referência, 404 pedido inexistente e relatório final com
  receita, custo, saldo e **zero divergências**.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** · suítes apenas
  com as **falhas ambientais conhecidas**.

## 6. Limites declarados

1. **Sem integração bancária** (OFX/CNAB/API) e sem importação de extrato: a referência é
   registrada, não conciliada automaticamente contra um arquivo real.
2. **Sem fiscal** (NF-e, impostos) e sem integração contábil.
3. **Sem cobrança**: não há emissão de Pix/boleto nem PSP; nada é transferido.
4. **Sem rateio por centro de custo** e sem orçamento por área.
5. **Sem estorno automático**: o sistema **sinaliza** o estorno devido; executá-lo exige
   autorização humana.
