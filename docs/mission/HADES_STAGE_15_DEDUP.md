# Estágio 15 — deduplicação de memória

## 1. A atividade que faltava

A verificação do estágio (E-089) mediu **quatro atividades ausentes**; esta rodada
fecha a primeira: **deduplicação**. O risco que ela mitiga é declarado no próprio
estágio — *contexto inflado*: sem dedup, a mesma decisão registrada duas vezes ocupa
duas linhas do preâmbulo do modelo e consome o orçamento de contexto que outras
memórias usariam.

## 2. A regra (conservadora de propósito)

Só é a **mesma** memória quando coincidem **conteúdo, tipo, visibilidade/ator e
origem**. Qualquer diferença é conhecimento diferente:

| Diferença | Por que NÃO funde |
| --- | --- |
| `kind` | fato e decisão não são a mesma coisa |
| visibilidade/ator | fronteira de privacidade |
| `source` | proveniência diferente (documento, web, canal) |

Fundir o que não é idêntico **perderia informação** — e perder memória é pior que
guardar uma linha repetida. A gravação duplicada devolve a memória **original**,
para quem grava saber que ela já existia.

A checagem acontece **antes** do embedding (não gasta vetor em duplicata) e é
repetida **sob o lock**, para duas gravações concorrentes não passarem juntas.

## 3. O efeito nas fixtures de retenção (e o erro que eu cometi)

A dedup quebrou **três testes de retenção**. Diagnóstico, não suposição: as fixtures
inseriam **N memórias com conteúdo idêntico** em laço para testar retenção **por
contagem**. Não era expectativa de produto — era artefato da fixture. Corrigi as
fixtures dando **origem própria** a cada memória, com o motivo escrito no código.

**E errei na primeira tentativa**: usei `Source: "seed-" + índice`, que **colide
entre duas chamadas de seed** com o mesmo conteúdo — então a dedup fundiu as
memórias "novas" com as "antigas" e o teste acusou `removed by count = 0, want 2`.
O conserto foi incluir o **prefixo** na origem (`prefixo-índice`). O erro foi
encontrado porque o teste **mediu o efeito**, não porque eu reli o código.

## 4. Verificação

- 3 testes novos: duplicata devolve a memória original e o contexto **não infla**
  (contagem 1); quatro diferenças legítimas continuam separadas (tipo, origem,
  privada do mesmo ator, privada de outro); dedup **sobrevive ao reinício** —
  inclusive gravando de novo depois dele.
- Suítes completas de `internal/agent` e `server` de volta ao estado anterior:
  **apenas as falhas ambientais conhecidas**.
- `go build ./...` exit 0 · `golangci-lint` **0 issues** · `gofmt` limpo.

## 5. Estado do estágio 15

| Atividade | Situação |
| --- | --- |
| Isolamento · exportação/exclusão · contexto do modelo | cobertos (E-015, E-088, E-089, E-090) |
| Troca de modelo · reinício · retomada | **provados ao vivo** (E-089) |
| **Deduplicação** | **coberta nesta rodada** |
| Versionamento · resumos · criptografia em repouso · escopos missão/agente/equipe | **ainda ausentes** |

Continua **`IN_PROGRESS`**: a lista de atividades ainda não está completa.
