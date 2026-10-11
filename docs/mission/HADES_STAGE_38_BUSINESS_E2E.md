# Estágio 38 — homologação de uma empresa autônoma em sandbox

## 1. O que foi entregue

`Runtime.CertifyBusinessJourney` percorre a jornada completa de uma empresa de sandbox e
devolve **evidência por etapa**, **prova de isolamento** e a **lista do que não foi feito**.
Exposto como superfície real: `POST /business/certify`.

## 2. As 18 etapas da jornada

| # | Etapa | Evidência produzida |
| --- | --- | --- |
| 1 | empresa criada | `company=<id>` |
| 2 | produto e estoque | `product=<id>` e unidades disponíveis |
| 3 | contato no CRM | `contact=<id>` |
| 4 | consentimento registrado | canal do opt-in |
| 5 | mensagem recebida | `conversation=<id>` |
| 6 | agente respondeu | `message=<id>` e **procedência do texto** |
| 7 | oportunidade criada | `opportunity=<id>` |
| 8 | pedido criado | `order=<id>` e total |
| 9 | estoque reservado | disponível, físico e reservado |
| 10 | pagamento sandbox | método por **referência** |
| 11 | receita lançada | `ledger=<id>`, valor e moeda |
| 12 | conciliação | referência do extrato |
| 13 | pedido atendido | físico após a saída |
| 14 | SLA medido | volume de entrada/saída e violações |
| 15 | financeiro consistente | receita, líquido e conciliados |
| 16 | recuperação idempotente | **mesmo** pedido, sem nova receita |
| 17 | isolamento provado | verificações cruzadas |
| 18 | externo pendente declarado | itens dependentes de conta/provedor |

## 3. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| **CADA ETAPA CARREGA A EVIDÊNCIA** | o relatório não diz "passou": diz **qual identificador** foi produzido e conferido. Um "OK" sem identificador seria **afirmação sem lastro** |
| **A PROVA DE ISOLAMENTO É PARTE DO RELATÓRIO** | cada recurso da organização é **lido como a outra organização** e a tentativa precisa falhar; o veredito só aprova se **nenhum** vazamento aparecer |
| **O QUE NÃO FOI FEITO É DECLARADO** | marketplace, redes sociais, telefonia, PSP real, fiscal e canais externos aparecem como `SKIPPED_EXTERNAL` com motivo. **Declarar é diferente de omitir**, e simulação **nunca** é chamada de produção |
| **ETAPA QUE FALHA NÃO É ESCONDIDA** | erro vira etapa `FALHOU` com o motivo e o veredito é `FALHOU`; a rota devolve **422 com o relatório**, para o operador ver **qual** etapa quebrou |
| **A JORNADA É IDEMPOTENTE** | refazer checkout e lançamento devolve os **mesmos** identificadores — sem isso, certificar duas vezes **cobraria duas vezes** |

## 4. Verificação

- **2 testes de motor**: a jornada completa aprovada com **18 etapas**, todas as etapas
  essenciais `OK`, **toda** etapa aprovada com evidência não vazia, dependência externa
  declarada (≥5 itens), isolamento provado (≥3 verificações, **nenhuma** visível para a
  outra organização), idempotência provada, um pedido atendido e financeiro sem
  divergência · **REINÍCIO REAL**: novo runtime sobre o **mesmo diretório de dados**
  recupera pedido atendido, estoque, receita lançada **uma vez**, conciliação, idempotência
  do checkout e **consentimento** — e repostar a receita **não** cria um segundo lançamento.
- **1 teste de rota**: certificação `200` com ≥18 etapas, isolamento e recuperação provados
  e lista de pendências externas; corpo inválido `400`.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** · suítes apenas
  com as **falhas ambientais conhecidas**.

## 5. Limites declarados (os mesmos do relatório)

1. **Marketplace e redes sociais**: exigem conta autorizada e scopes válidos.
2. **Telefonia e voz**: exigem número e provedor real.
3. **Pagamento real**: não há PSP nem transferência de dinheiro.
4. **Fiscal (NF-e, impostos)** e integração contábil.
5. **Canais externos de mensagem**: exigem credencial do provedor.
6. **A jornada é sandbox**: nada aqui autoriza chamar o resultado de produção. O veredito é
   `SANDBOX_APROVADO`, não release.
