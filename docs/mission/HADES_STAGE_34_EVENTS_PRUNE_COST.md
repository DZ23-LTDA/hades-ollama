# Estágio 34 — custo da poda do cache de eventos: medido, com controle (E-175)

## 1. O limite que isto fecha

Em E-174 ficou **declarado como não medido**: *"a poda percorre os eventos da organização a cada mensagem aceita e
a latência dessa varredura não foi medida sob carga"*. Este documento **mede** — e a medição **contrariou** a
explicação mais confortável, que era minha.

## 2. O que foi medido

Ambiente: **Windows**, Go **1.26.9** (o mesmo do CI), `i9-13900HX`, `-benchtime 2000x -count=3`, teto de 2000
eventos. Números em **µs por ingestão aceita** (três rodadas por caso):

| Caso | Rodada 1 | Rodada 2 | Rodada 3 | O que isola |
| --- | --- | --- | --- | --- |
| Cache **abaixo** do teto | 110,6 | 114,3 | 163,8 | ingestão normal (mapa pequeno) |
| Cache **no teto**, poda roda | 384,4 | 424,3 | 551,0 | ingestão + poda, mapa grande |
| **Controle**: mapa grande de OUTRA organização, sem poda | 399,2 | 403,3 | 415,6 | ingestão, mapa grande, **poda nunca roda** |

## 3. A conclusão que a medição sustenta

O **controle** — mesmo tamanho de mapa, **poda desligada** — custa praticamente o mesmo que o caso com poda
(≈400 µs contra ≈384–551 µs). Ou seja:

> **A poda não é o custo dominante.** O custo é o **tamanho do mapa vivo** (~100 µs → ~400 µs), e é exatamente
> isso que o **teto** limita.

## 4. Duas hipóteses minhas, refutadas pelos números

| Tentativa | Resultado medido | Veredito |
| --- | --- | --- |
| Trocar `sort` por **varredura de mínimo** (O(n), sem alocação) | 483–523 µs (antes: 447–491 µs) | **refutada** — dentro do ruído |
| **Poda amortizada** (contador O(1) por organização + folga de 64) | 484–633 µs | **refutada** — não melhorou |

A variante amortizada ainda introduziu um **bug de seleção invertida** (mantinha os mais antigos em vez de
descartá-los), pego pelo teste `TestOmnichannelEventsCacheRespectsTheCap`. Isso reforça a regra: **testar e medir
antes de acreditar na própria explicação**.

## 5. Decisão de engenharia

Voltou a implementação **mais simples que está correta e medida** — sem complexidade que **não comprou
desempenho**. O código carrega a nota de medição para que a tentativa não seja repetida às cegas.

O benchmark fica no repositório como **instrumento permanente** (`internal/agent/omnichannel_events_prune_benchmark_test.go`),
incluindo o caso de controle: sem ele, a conclusão acima seria opinião.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run 'TestOmnichannelEventsCache|TestOmnichannelDedupe' -count=1` | **PASS** |
| `go test ./internal/agent/ ./server/ -run '...'` | **PASS** (bateria da rodada) |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `go test ./internal/agent/ -run '^$' -bench BenchmarkOmnichannelIngest -benchtime 2000x -count=3` | números da tabela da seção 2 |

## 7. Limites declarados

- Medido **por ingestão isolada**, num laço de benchmark: **não** é medição de carga concorrente, nem de latência
  sob contenção do mutex, nem de comportamento de GC em processo longo.
- O número é de **Windows/go1.26.9**; o container Linux mede diferente (outras evidências já mostraram isso).
- A conclusão vale para o **teto de 2000**; tetos muito maiores não foram medidos, e o custo cresce com o tamanho
  do mapa vivo.
- **Não** foi medido **RSS** real por organização — o "tamanho do mapa vivo" é inferido do custo de ingestão, não
  de um perfil de memória.
- A poda continua sendo **O(n log n) por mensagem aceita com o cache cheio**; ela só **não é** o gargalo medido.
  Reduzi-la sem medir antes seria otimização de fé — e foi exatamente o que a medição impediu.
