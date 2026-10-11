# Estágio 35/21 — últimas esperas por máquina e benchmark no Linux (E-161)

## 1. Fechando a classe de E-155

As duas últimas esperas que mediam a **velocidade do host** estavam no bloco Redis de
`internal/agent/distributed_integration_test.go` (5 s e 10 s). Elas passaram a usar `waitBudget(t)`:

```
-	case <-time.After(5 * time.Second):
+	case <-time.After(waitBudget(t)):
-		case <-time.After(10 * time.Second):
+		case <-time.After(waitBudget(t)):
```

Verificação de completude: **zero** ocorrências de `time.After(N * time.Second)` em arquivos `_test.go` do
pacote — **12** esperas de evento usam `waitBudget`. A única ocorrência restante está em
`internal/agent/supervisor.go`, que é **código de produção** (timeout real de supervisão, não teste).

## 2. Benchmark no Linux: a limitação "host único" com dado

`docs/mission/HADES_BENCHMARK_21.md` declarava: *"host único Windows; o gate Linux mede o mesmo código, mas com
números próprios que não foram coletados aqui"*. Agora existe `scripts/verify-linux-benchmark.ps1`, que roda os
**mesmos quatro benchmarks** no contêiner e grava `_tmp/linux-benchmark-last.log`.

O script **não é gate de aprovação** de propósito: transformar variação de tempo em falha de CI seria repetir o
defeito de E-155 (teste que mede a velocidade do host). Ele **mede e grava**.

| Benchmark | Windows (E-160) | Linux (contêiner, agora) |
| --- | --- | --- |
| `ServiceAgentTurn` | 1,69–1,90 ms | **2,62–2,70 ms** |
| `MemorySearchLiteral` (1.000) | 0,27–0,52 ms | **0,29 ms** |
| `MemorySearchSemantic` (1.000) | 0,68–1,36 ms | **0,62–0,65 ms** |
| `MemoryReembed` (200) | 1,95–2,07 ms | **2,81–2,90 ms** |

## 3. O que os números dizem — e o que NÃO dizem

- Busca **literal e semântica**: praticamente iguais entre as plataformas.
- **Turno** ~40% mais lento no Linux e **re-embed** ~40% mais lento, com **contagem de alocações maior**
  (turno 206 vs 180; re-embed 429 vs 231).
- **Causa NÃO medida**: a hipótese mais provável é diferença de I/O de arquivo e de manipulação de caminho
  entre os sistemas, mas isso não foi instrumentado — fica registrado como **pergunta aberta**, não como
  conclusão.
- **Não é comparação entre hardwares**: o contêiner roda **na mesma máquina física** (mesmo `i9-13900HX`
  reportado nos dois). O que se mediu é a diferença de **runtime/sistema de arquivos**, não de CPU.

## 4. Verificação

| Comando | Resultado |
| --- | --- |
| `scripts/verify-linux-benchmark.ps1` | **OK** — 4 benchmarks, 2 execuções, log gravado |
| `go test ./internal/agent/ ./server/ -run 'TestServiceAgent\|TestRuntimeServiceAgents\|TestAgentMemory'` | **94/94 PASS** |
| `go test ./internal/agent/ -count=1` | **ok 67,154s** |
| `go test ./server/ -count=1` | **ok 23,879s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agente 18,161s · servidor 5,825s) |
| `gofmt -l internal/agent server` | **sem saída** |

## 5. Erro meu nesta rodada (registrado)

Editei `distributed_integration_test.go` por script e **esqueci o `gofmt`** — o mesmo erro de E-159, agora pela
**terceira vez** na sequência (edições por script tendem a sair do formato). O `gofmt -l` pegou antes do
commit; a lição prática ficou explícita: **toda edição por script termina com `gofmt -w`**.

## 6. Limites declarados

- A comparação Windows×Linux é de **runtime na mesma máquina**, não de hardware distinto.
- **Uma única imagem de contêiner** (`golang:1.26.9`) e um único host: não há repetição em outra máquina.
- `benchtime` de 50 iterações: serve à comparação interna, **não** a p95/p99 nem a carga concorrente.
- O motivo da diferença de turno/re-embed **não foi investigado** (sem instrumentação de I/O).
- `ExternalComparison` com produtos de terceiros continua **`NAO_EXECUTADA`**.
