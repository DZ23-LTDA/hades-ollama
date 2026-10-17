# Estágio 21 — benchmark do Hades com metodologia registrada (E-160)

> **Artefactos MEDIDOS deste estagio**: `docs/parity/HADES_T2_MATRIX.md` (matriz RED->GREEN com modelos reais -- 3/8 verde, 4 `HARNESS_TRAFICOU`, 1 vermelho -- `E-1047`) e `docs/parity/HADES_T2_REPETICAO.md` (repeticao estatistica -- `E-1049`). *Os comandos reproduziveis estao em `HADES_STAGE_21_BENCHMARK.md`, secao 6.*

## 1. O que estava em aberto

O estado do projeto declarava `ExternalComparison = NAO_EXECUTADA` desde o início da missão. A diretriz V2.1 é
explícita: *"Melhor que todos é objetivo competitivo a medir com benchmarks; jamais reivindicar supremacia sem
metodologia independente"*. Esta rodada entrega o que é **honestamente mensurável** e declara o que não é.

## 2. Metodologia registrada

| Item | Definição |
| --- | --- |
| **O que é medido** | caminhos determinísticos do próprio Hades: turno de atendimento com plano por regras, busca literal, busca semântica e re-embed |
| **Como** | `go test -bench` (`internal/agent/platform_benchmark_test.go`), dados fixos criados por iteração, **sem rede** |
| **Ambiente** | Windows 11 · `go1.26.9` · `GOARCH=amd64` · Intel Core i9-13900HX (13ª ger.) · `benchtime=50x/100x`, `-count=2..3` |
| **Ambiente Linux (E-161)** | mesma máquina, dentro de `golang:1.26.9` via `scripts/verify-linux-benchmark.ps1` — compara **runtime**, não hardware |
| **Comando** | `go test ./internal/agent/ -run '^$' -bench 'BenchmarkServiceAgentTurn\|BenchmarkMemorySearchLiteral\|BenchmarkMemorySearchSemantic\|BenchmarkMemoryReembed' -benchtime=100x -count=3` |
| **Isolamento** | cada iteração usa diretório temporário próprio; a criação do cenário fica **fora** do cronômetro onde indicado |
| **Verdade do dado** | o benchmark **falha** se a busca não encontrar a memória plantada ou se o re-embed não recalcular todas — número sem verificação não entra |

## 3. Resultado medido

| Benchmark | Medido (3 execuções) | Nota |
| --- | --- | --- |
| `BenchmarkServiceAgentTurn` | **1,69–1,90 ms** por turno · ~180 allocs | inclui persistência de trilha/estado em disco |
| `BenchmarkMemorySearchLiteral` | **0,27–0,52 ms** | 1.000 memórias, alvo no meio da varredura |
| `BenchmarkMemorySearchSemantic` | **0,68–1,36 ms** | mesmo volume, caminho vetorial (`cosineSimilarity`) |
| `BenchmarkMemoryReembed` | **1,95–2,07 ms** para 200 memórias | ≈ **10 µs por memória** |

**Diagnóstico do turno no Linux (E-164)**: o controle separa computação de I/O — planejador puro **≈0,4 µs nas duas plataformas** e busca em memória **igual**, enquanto a **escrita atômica** de 200 memórias vai de **1,66–1,76 ms (25 allocs)** no Windows para **2,51–2,70 ms (224–226 allocs)** no contêiner. As três operações mais lentas são **exatamente** as que escrevem arquivo ⇒ a diferença está no **caminho de escrita**, não na computação; o mecanismo específico **não** foi isolado e o contêiner usa sistema de arquivos em camadas (não é "Linux mais lento").

**Linux (E-161, mesma máquina em contêiner)**: turno **2,62–2,70 ms** · literal **0,29 ms** · semântica **0,62–0,65 ms** · re-embed **2,81–2,90 ms**. A busca é equivalente entre plataformas; turno e re-embed ficam ~40% mais lentos no Linux e com **mais alocações** (turno 206 vs 180, re-embed 429 vs 231). **Causa não medida** — hipótese de diferença de I/O/caminho, registrada como pergunta aberta, não como conclusão.

Leitura honesta dos números: a busca **semântica** custa ~2× a literal no mesmo volume — coerente com comparar
cosseno em vez de substring; o turno é dominado por I/O (arquivos de auditoria/aprovação), não por cálculo.
**Nada disso é comparação com outro produto** — é a linha de base do Hades.

## 4. `ExternalComparison: NAO_EXECUTADA` (declarado, com desbloqueio)

Comparar com Manus, Claude Code, Codex, Devin, OpenHands, Cursor, Lovable, Bolt, v0, Base44, n8n, Dify,
Perplexity, CrewAI ou LangGraph exige **instalar, autenticar e pagar** por eles, com as mesmas tarefas e
métrica. Nenhum desses passos foi feito — então **não há número comparativo**, e inventar um seria a pior
saída possível.

Procedimento exato de desbloqueio: (1) instalar a ferramenta de referência; (2) configurar a conta/credencial
correspondente; (3) executar **a mesma** lista de tarefas com critério de sucesso objetivo; (4) medir com este
mesmo comando e ambiente; (5) registrar no índice com a data. Sem (1)–(3), `ExternalComparison` permanece
`NAO_EXECUTADA`.

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `go test -bench ... -benchtime=100x -count=3` | **4 benchmarks, 3 execuções, sem falha** |
| `go test ./internal/agent/ ./server/ -run 'TestServiceAgent\|TestRuntimeServiceAgents\|TestAgentMemory'` | **94/94 PASS** |
| `go test ./internal/agent/ -count=1` | **ok 66,806s** |
| `go test ./server/ -count=1` | **ok 22,427s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agente 20,424s · servidor 6,513s) |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |

## 6. Erros meus nesta rodada (registrados)

1. **Ordem de fixture, terceira vez**: o benchmark de re-embed usou a mesma store entre iterações com o
   embedder já ligado — as memórias nasciam carimbadas e o benchmark mediu **`Skipped: 200`** em vez de
   recálculo. A saída foi criar store nova por iteração (o `StopTimer`/`StartTimer` agora cobre só o cenário).
   É a **terceira** ocorrência da mesma armadilha nesta sequência (E-157 e E-159 antes) — registrada como
   risco recorrente de quem escreve o teste/benchmark.
2. **Runtime sem conversa**: o primeiro `BenchmarkServiceAgentTurn` falhou com `inbox conversation not found`.
   Eu havia montado o runtime por `NewRuntime` e inventado um `ConversationID`. A correção foi **reutilizar o
   fixture real** — e, em vez de copiá-lo, generalizá-lo para `testing.TB` (serve teste e benchmark). Duplicar
   teria sido a mesma dívida que o embedder duplicado do E-157.
3. **`intrange` (3 achados)**: o repositório exige `for range b.N`; dois índices ficaram **sem uso** depois da
   troca, e o `vet` pegou (`declared and not used`). Corrigido para `for range b.N` sem variável.

## 7. Limites declarados

- **Sem comparação externa**: nenhum concorrente foi executado (`NAO_EXECUTADA`), com procedimento de
  desbloqueio explícito.
- **Duas plataformas, UMA máquina**: os números Linux vêm de contêiner na MESMA máquina física — mede
  runtime/sistema de arquivos, **não** hardware distinto, e a causa da diferença **não** foi investigada.
- **`benchtime` fixo em 50–100 iterações**: serve para comparação interna entre caminhos, não para
  percentis de cauda (p95/p99) nem para carga concorrente.
- **Cenários sintéticos**: 1.000 memórias e 200 memórias por re-embed não representam a maior base real.
- **O turno inclui I/O**: o número mede o caminho completo (plano + ferramenta + persistência), não só o
  planejador.
