# Estágio 38 + 31 — o supervisor entra no laço da certificação

## 1. O que mudou

A certificação E2E deixou de terminar no financeiro e passou a **exercer o supervisor
dentro da própria jornada**: um segundo pedido é pago, tem receita lançada e é cancelado,
o que gera **estorno devido** — e a política exige **decisão humana** para mexer em
dinheiro. A jornada agora tem **21 etapas**, e as três novas são:

| Etapa | O que prova |
| --- | --- |
| `supervisor_ciclo` | o ciclo observa os domínios, o estorno **aguarda humano** e a pendência é consultável |
| `aprovacao_humana` | a decisão de `gerente-certificacao` **executa o efeito real** no livro e fica registrada |
| `supervisor_isolamento` | a pendência de decisão **não vaza** para outra organização |

O relatório ganhou um bloco `supervisor` com sinais, pendências, bloqueios por orçamento,
quem decidiu e **qual foi o efeito no livro**.

## 2. O defeito real que isso expôs (e que eu corrigi)

Ao estender a jornada, os testes acusaram dois pedidos em vez de um — esperado. Mas ao
olhar o financeiro apareceu um **defeito de verdade**: depois de o estorno ser **aprovado**,
o relatório **continuava marcando o pedido como estorno devido**, porque ele só considerava
a receita lançada e **não reconhecia o estorno já registrado**.

O efeito seria grave e silencioso: o supervisor proporia **a mesma decisão em todo ciclo,
para sempre**, mesmo depois de uma pessoa ter aprovado. Corrigi em `FinanceOps.Report`: o
relatório agora coleta os lançamentos com idempotência `refund:<pedido>` e trata o pedido
cancelado como **LIQUIDADO** — e a jornada verifica, num ciclo seguinte, que
`AwaitingApproval` volta a **zero** e a fila de pendências fica **vazia**.

É o defeito clássico que o próprio estágio aponta: **a ação que não fecha o que abriu**.

## 3. Um erro meu no teste

Eu também errei a aritmética da asserção: escrevi `preço × quantidade × 2`, mas o pedido
estornado tem **uma** unidade (não `quantidade`). A receita correta é a do pedido atendido
mais **uma** unidade — e foi a **asserção** que estava errada, não o motor. Corrigi a
asserção para a expressão correta.

## 4. Verificação

- **Jornada E2E com 21 etapas**, todas as etapas essenciais `OK` e **com evidência**;
  financeiro final com **duas** vendas registradas (uma atendida, uma estornada), **nenhum
  estorno devido**, **nenhuma divergência** e **exatamente um** lançamento de estorno.
- **Ciclo seguinte não repropõe a decisão**: `AwaitingApproval = 0` e a fila vazia.
- **Reinício real** (novo runtime, mesmo diretório de dados): os **dois** pedidos, a
  receita, a conciliação, o **estorno aprovado** e a **ausência de pendências** sobrevivem.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** · suítes apenas
  com as **falhas ambientais conhecidas**.

## 5. Limites declarados (inalterados)

Marketplace e redes sociais (conta e scopes) · telefonia e voz (número e provedor) ·
pagamento real (**sem PSP**) · fiscal (NF-e/impostos) · canais externos (credencial). O
estorno aprovado é **registro contábil**: nenhum dinheiro é transferido. O veredito segue
`sandbox_aprovado` — **não** release.
