# Estágio 31 (recuperação) — as decisões do supervisor sobrevivem ao reinício

## 1. O que eu estava devendo

Na rodada anterior eu declarei, no meu próprio relatório, este limite: *"aprovações e
histórico do ciclo ficam em memória"*. Isso significava que, após um reinício, uma
**decisão pendente** desapareceria — e uma ação irreversível que esperava aprovação humana
**sumiria do radar do operador** (ou voltaria a ser proposta como se nunca tivesse sido
vista). Este é o defeito que eu fui corrigir.

## 2. O que foi feito

Persistência do supervisor de negócio num arquivo único
(`.agent-business-supervisor/business-supervisor.json`) com:

| O que | Por quê |
| --- | --- |
| **Ações e suas decisões** | uma pendência que espera humano **não pode desaparecer**; e uma ação recusada **não pode voltar a pendente** |
| **Política vigente** | limites não se perdem no reinício (orçamento, mínimo de estoque, profundidade de DLQ) |
| **Orçamento consumido** | sem isso o supervisor **gastaria duas vezes** o mesmo orçamento |
| **Contagem de ciclos** | a numeração dos ciclos continua, em vez de reiniciar em 1 |

A gravação acontece **nos pontos que mudam estado**: ao registrar a pendência, ao
bloquear por orçamento, ao executar ação autônoma, ao decidir e ao trocar a política. A
pendência é gravada **antes de sair** do ciclo.

## 3. Verificação

- **3 testes novos de reinício REAL** (novo runtime sobre o **mesmo diretório de dados**):
  1. **pendência sobrevive**: a ação de estorno continua lá, é a **mesma** (ID e tipo), a
     política vigente permanece, e a decisão pode ser tomada **depois** do reinício com
     efeito real (estorno registrado **uma** vez e ator gravado);
  2. **decisão não volta a pendente**: recusa registrada antes do reinício continua
     recusada depois; decidir de novo é **idempotente** e a decisão **original prevalece**
     (aprovar depois de recusar **não** executa);
  3. **orçamento e contagem sobrevivem**: o segundo ciclo é o **número 2** e o orçamento
     gasto (35000) e o restante (65000) permanecem.
- Os **6 testes anteriores** continuam verdes (observação dos domínios, reposição com
  efeito real, bloqueio por orçamento, aprovação humana, recusa e recuperação).
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** · suítes apenas
  com as **falhas ambientais conhecidas**.

## 4. Limite que continua declarado

O **histórico dos ciclos** (`history`) segue **em memória** — ele é a trilha analítica do
supervisor, não a decisão. Pendência, decisão, política e orçamento, que são o que
**muda o mundo**, agora são persistentes. O ciclo continua **acionado por rota** (ou pelo
agendador existente), sem loop próprio com lease.
