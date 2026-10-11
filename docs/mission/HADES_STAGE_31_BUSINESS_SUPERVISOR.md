# Estágio 31 — supervisor autônomo de negócio

## 1. O que foi entregue

O supervisor de **missões** já existia (tick, journal, ocorrências, aprovação de risco).
Este é o supervisor de **negócio**: ele observa o que os outros módulos já produzem —
**atendimento** (SLA e fila humana), **comércio** (estoque e pedidos) e **financeiro**
(receita não lançada, estorno devido, divergências) — e age sob **política declarada**.

**Ele não duplica o journal**: achados altos são registrados nos **mesmos** registros de
ocorrência que o operador já consulta, e a recuperação os resolve.

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| **AÇÃO IRREVERSÍVEL NÃO EXECUTA SOZINHA** | estorno (mexe em dinheiro) e cancelamento de pedido (desfaz uma venda) ficam **aguardando aprovação humana**, com motivo e custo à vista |
| **AÇÃO REVERSÍVEL/CONTÁBIL É AUTÔNOMA** | lançar receita de pedido **já pago** é registro **idempotente** que **não move dinheiro**; repor estoque é **ajuste** com movimento no livro. Esses podem ser autônomos, e o efeito real é registrado |
| **ORÇAMENTO É LIMITE, NÃO SUGESTÃO** | custo acima do orçamento restante ⇒ **não executa**, vira `BLOQUEADA_ORCAMENTO` com motivo, e **não consome** orçamento. Estourar em silêncio é o defeito que o estágio aponta |
| **APROVAR EXECUTA, E SÓ UMA VEZ** | a decisão é idempotente: aprovar de novo devolve a mesma ação **sem repetir o efeito externo** |
| **A DECISÃO PRECISA DE ATOR** | aprovação sem ator identificado é recusada — sem ator não há trilha de auditoria |
| **O QUE NÃO SEI EXECUTAR É RECUSADO** | tipo desconhecido falha explícito, em vez de "concluir" sem efeito |
| **FALHA DE UM DOMÍNIO NÃO DERRUBA O CICLO** | organização sem empresa lida ⇒ o domínio financeiro é declarado ilegível em `errors` e a supervisão dos outros domínios **continua** |
| **O LIMITE É O DA POLÍTICA** | o supervisor compara com `MinAvailableUnits` **da própria política**, não com o limiar do resumo do comércio |

## 3. Alcançável pelo produto

`POST /business/tick` (roda o ciclo e devolve sinais, ações, orçamento, bloqueios e erros
declarados) · `GET /business/actions` (pendências + política vigente) ·
`POST /business/actions/:id/decide` (`APROVAR` executa, `RECUSAR` não executa).

Códigos: **404** ação inexistente · **400** decisão inválida/sem ator · **422** tipo não
executável · **409** conflito.

## 4. Dois defeitos meus, encontrados pelos testes

1. **Eu comparava o estoque com o limiar do `Summary` do comércio** (que só marca saldo
   **zerado**), então a política `MinAvailableUnits = 2` **nunca disparava**. Passei a
   comparar com o limite **da própria política** — o supervisor é quem manda no seu
   limiar, e depender do limiar do outro módulo faria a política ser decorativa.
2. **Organização sem empresa derrubava o ciclo inteiro** com erro no domínio financeiro.
   Agora a falha é **declarada** em `errors` e a supervisão dos outros domínios
   **continua**: derrubar tudo esconderia os achados que já existiam.

## 5. Verificação

- **6 testes de motor**: observação dos domínios com ocorrência registrada no journal
  existente e receita lançada por ação autônoma · **reposição autônoma** com efeito real
  (estoque 1 → 11), custo **35000** no orçamento, restante **65000** e movimento auditável
  no livro · **bloqueio por orçamento** (1000 < 35000) sem executar e sem consumir ·
  **estorno exige aprovação** (nada lançado antes), aprovação executa e registra o ator,
  e **aprovar duas vezes não estorna duas vezes** · **recusa não executa**, exige ator e
  recusa decisão inválida, com ação inexistente falhando explícito · **recuperação
  observada** resolve a ocorrência e o escopo por organização é respeitado, com tipo
  desconhecido recusado.
- **1 teste de rota**: ciclo `200` com sinais e aprovação pendente, listagem com política
  vigente, **nenhum estorno antes da decisão**, recusa registrada, `400` para decisão
  inválida, `404` para ação inexistente e `400` para corpo inválido.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** (falsos
  positivos do `misspell` **reescritos** em vez de silenciados) · suítes apenas com as
  **falhas ambientais conhecidas**.

## 6. Limites declarados

1. **Aprovações e histórico do ciclo ficam em memória**: as **ocorrências** e o journal do
   supervisor de missões são persistentes, mas as pendências de decisão não sobrevivem a
   um reinício nesta rodada.
2. **O ciclo não roda sozinho**: ele é acionado por rota (ou pelo agendador existente); não
   há loop próprio com lease.
3. **Estorno é registro contábil**: a despesa entra no livro, mas **nenhum dinheiro é
   transferido** — executar a devolução real exige o provedor de pagamento.
4. **Sem precificação dinâmica nem promoções**: as ações são de estoque, fila, lançamento
   e estorno.
5. **Sem coordenar marketplaces e redes sociais**: dependem de conta autorizada.
