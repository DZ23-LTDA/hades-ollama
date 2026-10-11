# Estágio 35 — comércio e empresa ligados: 4 ferramentas saem de "indisponível" (E-145)

## 1. O que esta rodada fecha

Na rodada anterior (E-144) eu troquei o booleano escrito à mão por **disponibilidade derivada do registro de
executores** — e, como consequência honesta, **7 ferramentas** passaram a aparecer como **INDISPONÍVEIS**
por não terem executor. Quatro delas dependiam apenas de ligar stores que **já existem** no runtime:

`catalog.read` · `commerce.stock.read` · `commerce.order.read` · `company.report.read`

## 2. Como foi feito (sem afrouxar a regra)

`SetCommerce(store)` e `SetCompanies(store)` **registram executores** — a disponibilidade continua sendo
**consequência** do registro, não de uma lista. A prova é um teste que roda o mesmo agente **sem** a
ligação e exige `unavailable`: o caminho desligado não vira "disponível" só porque o código existe.

No `Runtime` a ligação acontece **depois** da criação dos stores (comércio e empresas nascem depois dos
agentes de atendimento); por isso a escolha por **setter**, em vez de reordenar a construção.

## 3. Cinco regras de honestidade nos novos executores

| Regra | Por quê |
| --- | --- |
| **Sem store ligado = indisponível** | disponibilidade é consequência: nada de anunciar o que não executa |
| **Catálogo vazio DIZ que está vazio** | `catalog.read` sem produto responde "nenhum produto cadastrado", em vez de inventar oferta |
| **Zero pedido é dito como zero** | o resumo de `commerce.order.read` mostra `0 pedido(s)` — número honesto, não omitido |
| **Estoque baixo aponta o SKU REAL** | o teste exige o SKU `BON-002` (o que ficou sem saldo), vindo do store |
| **Isolamento por organização** | o catálogo de `org-a` não aparece para `org-b` |

## 4. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run TestServiceAgent` | **17/17 PASS** (3 novos nesta rodada) |
| `go test ./internal/agent/ -count=1` | **ok 58,423s** |
| `go test ./server/ -count=1` | **ok 20,697s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agent 21,189s · server 6,434s · exit 0) |
| `gofmt -l internal/agent server` | **sem saída** |

Os três testes novos: **indisponível sem ligação**, **executa com dado real** (produto com preço, SKU de
estoque baixo, contagem de pedidos, empresa pelo nome) e **isolamento por organização**.

## 5. Erro meu nesta rodada (registrado)

A substituição que inseriu `SetCommerce`/`SetCompanies` deixou a **chave de fechamento** de `SetCompanies`
fora, e o arquivo passou a ter uma função sem fim — **erro de sintaxe pego pelo compilador**
(`expected operand, found ']'`). Registro porque é o terceiro caso da mesma família nesta missão: **edição
por substituição de texto precisa de verificação depois**, e o compilador/formatador é que fecha a conta.

## 6. Limites declarados

- **Continuam INDISPONÍVEIS, com motivo**: `knowledge.search`, `social.draft`, `voice.transfer` — dependem
  de módulos (base de conhecimento, social, voz) que **não** estão ligados a este runtime.
- **A trilha de ferramentas segue por turno**, não é auditoria durável.
- **Não há seleção de ferramenta por modelo**: quem pede é o chamador.
- **Escrita sem aprovação humana** (`crm.opportunity.write` executa com o pedido do turno).
