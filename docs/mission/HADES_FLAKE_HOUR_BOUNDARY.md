# O flake da virada de hora — CI vermelho que era defeito MEU no teste (E-142)

## 1. O que aconteceu

O PR #214 (**DLQ persistente**) abriu com **CI vermelho**. Localmente tudo passava. O log real do job
`Go agentic and server gates` mostrou a linha exata:

```
--- FAIL: TestBusinessSnapshotIsIdempotentPerWindow (0.02s)
    business_snapshots_test.go:91: registrar na mesma hora não pode criar outra amostra
FAIL	github.com/ollama/ollama/internal/agent	19.277s
```

E o carimbo de tempo do log entregou o resto: **`13:59:57 UTC`** — a **três segundos** da virada de hora.

## 2. A causa

O teste usava o relógio de parede do fixture (`now := time.Now().UTC()`) e chamava:

```go
runtime.BusinessSnapshot(organization, now)
runtime.BusinessSnapshot(organization, now.Add(time.Minute))   // <- pode CRUZAR a hora
```

A janela da série é **de hora** (`2006-01-02T15`). Se `now` está em `XX:59:40`, então
`now.Add(time.Minute)` cai em `XX+1:00:40` — **outra janela** — e a segunda chamada cria uma amostra
legitimamente. A asserção "não pode criar outra amostra" falhava **sozinha**, sem que houvesse nada
errado no produto: um teste que só passa em 59 dos 60 minutos.

**O agravante, registrado por honestidade:** na rodada E-134 eu havia tratado exatamente esse risco no
teste de **rota** (*"o handler usa o relógio REAL, então a segunda chamada pode cair na hora seguinte…
em vez de escrever uma asserção que falha às vezes"*) — e deixei este teste de unidade refém do relógio
real. **A lição foi aplicada em um lugar e não no vizinho**, e o CI cobrou a diferença.

## 3. A correção

Relógio **fixo** (`base = 2026-10-12T10:00:00Z`) e, mais importante, a **virada de hora passou a ser
exercitada de propósito**:

| Chamada | Janela | Esperado |
| --- | --- | --- |
| `base` | `…T10` | **cria** |
| `base + 1 min` | `…T10` | reaproveita (`created=false`, mesma amostra) |
| `10:59:30` | `…T10` | reaproveita — **10:59 pertence à hora 10** |
| `11:00:30` | `…T11` | **cria** — a janela virou |
| `11:45:00` | `…T11` | reaproveita |

O que era acidente de calendário virou **asserção explícita de semântica**.

## 4. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run TestBusinessSnapshotIsIdempotentPerWindow -count=50` | **ok 8,742s** (50 execuções) |
| `go test ./internal/agent/ -run 'TestBusinessSnapshot\|TestBusinessTrend' -count=5` | **ok 6,701s** |
| `go test ./internal/agent/ -count=1` (suíte completa) | **ok 61,813s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |

Uma execução com relógio fixo não pode depender de onde o ponteiro estava: o determinismo é
**estrutural**, e as 50 repetições são confirmação, não a garantia.

## 5. Limites declarados

- **Outros testes do arquivo usam `time.Now()`** (pelo fixture) sem depender da hora: os deslocamentos
  são de horas inteiras (`+1h`, `+2h`), onde a janela muda por intenção. A varredura deste arquivo
  confirmou **um único** ponto com deslocamento pequeno — o corrigido.
- **O fixture continua usando o relógio real**: torná-lo determinístico é mudança maior, que toca
  vários testes; fica declarado em vez de feito às pressas.
