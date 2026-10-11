# Estágio 30 — Finance & Operations: razão financeiro e conciliação

## 1. O que já existia (e por que não dupliquei)

| Já existia | O que cobre |
| --- | --- |
| `CompanyBudget` (limite mensal, limiar de aprovação, moeda) | política de gasto |
| `RecordSpendRequestWithIdempotency` | gasto sob política, com chave de idempotência |
| `DecideApproval` | aprovação humana do gasto |

O que faltava é o **outro lado** da operação financeira: **registrar o fato
financeiro** (receita/despesa), **conciliar** contra referência externa e **medir o
que não conciliou**. Sem isso não existe fluxo de caixa nem conciliação — dois itens
explícitos do estágio.

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| **Idempotência por chave, por empresa** | retry de webhook ou reenvio de requisição devolve o lançamento **original** em vez de cobrar duas vezes. A checagem e a inserção acontecem **sob o mesmo lock**, então vale para requisições **concorrentes** |
| **O razão NÃO mexe no orçamento** | `SpentCents` é **política de gasto**, já incrementada por `RecordSpendRequest`; somar de novo aqui **duplicaria** o consumo do limite. São eixos diferentes, e o código diz isso |
| **Conciliar é idempotente com a mesma referência e CONFLITO com outra** | repetir não é erro; afirmar que o mesmo lançamento foi conciliado com dois extratos diferentes precisa ser **recusado** |
| **Valor em centavos e positivo** | o sinal vem do tipo (`income`/`expense`). Valor zero ou negativo é erro de entrada, não "despesa negativa" |
| **Falha de gravação desfaz a inserção** | sem gravação não há lançamento: melhor errar do que mentir em memória |

## 3. Um bug real que o teste de concorrência encontrou

O primeiro teste (12 goroutines com a **mesma** chave) **derrubou o processo**:

```
panic: assignment to entry in nil map
```

O mapa do razão ficava `nil` quando a empresa ainda **não tinha arquivo**, e a
primeira gravação escrevia num mapa inexistente. Corrigido inicializando o mapa
**antes** de qualquer leitura. Este é o valor do teste: o caminho "primeira escrita
concorrente" não aparecia em uso sequencial.

## 4. Alcançável pelo produto

| Rota | Comportamento |
| --- | --- |
| `POST /companies/:id/ledger` | **201** na criação; **200 com o lançamento original** no reenvio idempotente |
| `GET /companies/:id/ledger` | lançamentos + resumo de fluxo de caixa, com recorte `from`/`to` |
| `GET /companies/:id/ledger/summary` | indicadores + **fila do que não foi conciliado** (`older_than_days`) |
| `POST /companies/:id/ledger/:entry_id/reconcile` | concilia; mesma referência é no-op (`changed=false`), outra é conflito |

Todas passam por `companyForRequest`, que verifica a **organização**: lançamento de
uma empresa não é lido nem conciliado por outra.

## 5. Verificação

- **6 testes de motor**: idempotência sob **12 goroutines concorrentes** (exatamente
  um lançamento) · entradas inválidas recusadas (sem empresa, tipo desconhecido,
  valor zero/negativo, empresa inexistente) · conciliação idempotente/conflito/
  referência obrigatória/lançamento inexistente · resumo de fluxo de caixa com
  cobertura de conciliação e fila atrasada · **sobrevive ao reinício** e **não
  duplica o orçamento** · razão separado por empresa (mesma chave em empresas
  diferentes é operação diferente).
- **2 testes de rota**: idempotência pelo HTTP (201 depois 200 com o **mesmo ID**),
  400 para entrada inválida e data inválida, listagem com resumo, conciliação
  (muda → repete sem mudar → conflito → 400 sem referência) e **isolamento por
  organização**; o recorte de "hoje" é **inclusivo** (limite superior no fim do dia).
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** · suítes
  apenas com as **falhas ambientais conhecidas**.

## 6. Limites declarados (o estágio NÃO está completo)

1. **Sem integração bancária real**: a conciliação recebe a **referência externa**
   informada; importar extrato (OFX/CNAB/API do banco) é trabalho futuro e depende
   de credencial/contrato.
2. **Sem fiscal**: notas, impostos, retenções e integrações fiscais **não** existem.
3. **Sem cobrança**: geração de boleto/Pix/link de pagamento **não** existe.
4. **Sem pagamento**: nenhuma transferência é executada — e, por decisão de
   segurança, **não** deve ser sem autorização explícita.
5. **Sem rateio por centro de custo** nem conciliação automática por valor/data.

Por isso o estágio 30 permanece **`IN_PROGRESS`** com capacidade **`PARCIAL`**: a
conciliação e o fluxo de caixa passaram a existir, o resto não.
