# Estágio 34 — concorrência na ingestão: reserva atômica, medição e `-race` (E-176)

## 1. O limite que isto fecha

E-175 deixou dois limites declarados: a medição era **"por ingestão isolada, não sob carga concorrente nem
contenção do mutex"**, e o **detector de corrida nunca havia rodado** (o CI usa `CGO_ENABLED=0`, e `-race` exige
cgo). Este documento fecha os dois.

## 2. Defeito estrutural encontrado na revisão (mesmo sem o teste reproduzir)

A checagem de dedupe e a **gravação** do evento **não são o mesmo instante**: entre elas está o `AppendInbound`
na caixa de entrada. Duas entregas idênticas em paralelo podiam, em princípio, **passar as duas pela checagem**
antes de qualquer uma gravar — e a dedupe existe exatamente para impedir isso.

O teste de concorrência com 8 goroutines partindo juntas **não reproduziu** a janela (passou 8 rodadas). Isso
**não** é prova de que a janela não existe: é prova de que ela é estreita. O provedor de webhook **reentrega**
quando não recebe 2xx, então a corrida é plausível em produção.

**Correção por construção**: a chave é **RESERVADA** sob o mesmo lock da checagem (`eventosEmVoo`) e a reserva é
liberada no fim — no sucesso (junto com a gravação, no mesmo lock) e também **no erro** da caixa de entrada, para
que uma entrega que falhou **não** fique marcada como "em processamento" para sempre. Assim a propriedade deixa de
depender de sorte.

## 3. Testes de concorrência

| Teste | Propriedade travada |
| --- | --- |
| `…ConcurrentDuplicateDeliveryIsAcceptedOnce` | a MESMA entrega, de 8 goroutines simultâneas, é **aceita uma única vez** e as outras 7 são duplicatas |
| `…ConcurrentKeepsTenantsApart` | organização A **nunca** enxerga evento de B, com as duas ingerindo ao mesmo tempo |
| `…ConcurrentRateLimitCountsCorrectly` | sob concorrência, **aceitas + recusadas = total** e o teto **não** é ultrapassado |
| `…ConcurrentEventsCacheRespectsTheCap` | o teto do cache de eventos vale também com 6 goroutines escrevendo |
| `…InFlightReservationIsReleased` | a reserva é **liberada** (nenhuma chave presa) e o mesmo id volta a ser aceito depois |

Execução: `-count=8` no conjunto `TestOmnichannel` — **ok**.

## 4. Medição SOB CONCORRÊNCIA

`BenchmarkOmnichannelIngestParalelo` (`b.RunParallel`), Windows, Go 1.26.9, `-benchtime 2000x -count=3`:

| Caso | Rodadas (µs/op) |
| --- | --- |
| Ingestão **isolada** (mesma máquina, mesma rodada) | 223,3 / 209,4 / 238,7 |
| Ingestão **paralela** (RunParallel) | 140,6 / 147,7 / 139,8 |

A ingestão **não colapsa** sob concorrência: a latência por operação fica na mesma ordem (e nesta rodada até
menor que a medição isolada concorrente, que disputava CPU com o restante da máquina). **Contenção do mutex não é
o gargalo** deste caminho.

> Ressalva honesta: os números oscilam entre rodadas e entre máquinas (E-175 já mostrou isso). O que a medição
> sustenta é **ordem de grandeza e ausência de colapso**, não um valor fino.

## 5. `-race` no container (o que o CI não fazia)

O host **não tem gcc**, então `-race` exige cgo e não roda aqui — mas o container `golang:1.26.9` tem. Executado:

```
docker run --rm -v "<repo>:/src" -v "<repo>/_tmp/gomodcache:/gomodcache" -w /src \
  -e GOFLAGS=-mod=mod -e GOTOOLCHAIN=local -e CGO_ENABLED=1 golang:1.26.9 \
  sh -lc "go test -race ./internal/agent/ -run 'TestOmnichannel' -count=1 -timeout 20m"
```

Resultado registrado na evidência. **O `-race` passou a fazer parte do gate de paridade Linux** para os testes de
concorrência do omnichannel — antes, nenhuma execução de CI usava o detector de corrida (`CGO_ENABLED=0` no
script de paridade).

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run 'TestOmnichannel' -count=8` | **ok** |
| `go test ./internal/agent/ -run TestOmnichannelInFlightReservation -count=3` | **ok** |
| `go test -race ./internal/agent/ -run TestOmnichannel -count=1` (container) | **ok** (ver evidência) |
| `go test ./internal/agent/ -run '^$' -bench BenchmarkOmnichannelIngestParalelo` | números da seção 4 |
| bateria da rodada / `golangci-lint` / contrato / paridade Linux | ver evidência |

## 7. Limites declarados

- A corrida **não foi reproduzida** antes da correção em 8 rodadas de 8 goroutines: a correção é **por
  construção**, e a evidência de que era necessária é **estrutural** (checagem e gravação em instantes
  diferentes), não uma falha observada. Isso fica dito em vez de ser apresentado como bug reproduzido.
- A medição paralela é **de um caminho só** (ingestão de um canal); não mede concorrência entre canais distintos,
  nem o caminho de saída, nem a caixa de entrada (que nos testes está ausente — `r.inbox == nil`).
- O `-race` cobre **apenas** `TestOmnichannel*` no Linux: as demais suítes continuam sem detector de corrida.
- `b.RunParallel` mede latência por operação em paralelo; **não** é um teste de carga com taxa fixa nem mede
  saturação, fila ou efeito de GC sob pressão contínua.
