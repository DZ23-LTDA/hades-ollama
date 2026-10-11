# Estágio 21 — o turno ~40% mais lento no Linux: onde está a diferença (E-164)

## 1. A pergunta aberta que E-161 deixou

E-161 mediu o turno com **2,62–2,70 ms** no contêiner contra **1,69–1,90 ms** no Windows e escreveu, com
honestidade: *"a CAUSA da diferença de turno/re-embed NÃO foi instrumentada — fica como pergunta aberta (a
hipótese mais provável é I/O e manipulação de caminho, mas hipótese não é medição)"*.

Esta rodada **instrumenta** a pergunta, com um **controle**: dois benchmarks novos que separam a **computação
pura** (planejador, sem I/O) da **escrita de arquivo** (persistência atômica de 200 memórias).

## 2. O experimento

| Benchmark | O que isola | Windows | Linux (contêiner) | Razão |
| --- | --- | --- | --- | --- |
| `BenchmarkServiceAgentPlan` | **só computação**, sem I/O | 0,37–0,47 µs | 0,38–0,57 µs | **≈1×** |
| `BenchmarkMemorySearchLiteral` | memória, sem I/O | 0,27–0,52 ms | 0,27–0,28 ms | ≈1× |
| `BenchmarkMemorySearchSemantic` | memória, sem I/O | 0,68–1,36 ms | 0,62–0,68 ms | ≈1× |
| `BenchmarkMemoryFileWrite` (200) | **escrita atômica** | **1,66–1,76 ms** · **25 allocs** | **2,51–2,70 ms** · **224–226 allocs** | **≈1,6×** |
| `BenchmarkServiceAgentTurn` | turno completo (persiste) | 1,79–1,90 ms · 178–180 allocs | 3,22–3,37 ms · 206–208 allocs | **≈1,8×** |
| `BenchmarkMemoryReembed` (200) | recalcula + grava por lote | 1,95–2,07 ms | 3,66–4,10 ms | **≈1,9×** |

Mesma máquina física (`i9-13900HX`) nos dois lados; mesmos comandos, `benchtime=50x`, `-count=2`.

## 3. Conclusão que os dados sustentam

1. **Computação pura é idêntica** (planejador ≈ 0,4 µs nos dois) → a diferença **não é CPU, GC ou runtime**.
2. **Caminhos em memória são idênticos** (busca literal e semântica) → **não** é diferença geral de memória.
3. As **três** operações mais lentas no Linux são **exatamente** as que **escrevem arquivo**
   (turno, re-embed e a escrita atômica isolada) — e a escrita isolada sozinha responde por **~1,7 ms dos
   ~3,3 ms** do turno.
4. A **contagem de alocações da escrita** salta de **25** (Windows) para **~225** (Linux) para o mesmo trabalho:
   o custo extra está **no caminho de escrita**, não no conteúdo.

Isso **fecha a pergunta como localização**: a diferença vive no **caminho de escrita de arquivo**.

## 4. O que os dados NÃO provam

- **Não** provam que "Linux é mais lento": o contêiner usa o **sistema de arquivos em camadas do contêiner**
  sobre a máquina Windows. O que se mediu é *"neste contêiner, neste host, a escrita é ~1,6–1,9× mais lenta"*.
- **Não** isolam o mecanismo dentro do caminho de escrita (`fsync`, `rename` atômico, permissões, resolução de
  caminho, overlayfs). A correlação está medida; **o mecanismo específico continua não provado**.
- **Não** medem carga concorrente nem p95/p99: são 50 iterações por benchmark.

## 5. Achado acionável para a próxima rodada

A escrita alocar **~225 vezes** contra **25** é alvo claro de otimização que **beneficia as duas plataformas**:
serializar duas vezes (uma para o arquivo, outra para o `rename`), criar o caminho e o arquivo temporário a cada
chamada são candidatos. Fica registrado como **trabalho derivado desta medição**, não como feito.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `scripts/verify-linux-benchmark.ps1` (6 benchmarks agora) | **OK** — log em `_tmp/linux-benchmark-last.log` |
| `go test ./internal/agent/ ./server/ -run '...'` | **106/106 PASS** |
| `go test ./internal/agent/ -count=1` | **ok 70,298s** |
| `go test ./server/ -count=1` | **ok 23,241s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `gofmt -l internal/agent server` | **sem saída** |

## 7. Limites declarados

- Duas plataformas, **uma** máquina física; Linux em **contêiner** (`golang:1.26.9`), não em SO nativo.
- `benchtime=50`, `-count=2`: serve à comparação, **não** a p95/p99.
- O **mecanismo** dentro do caminho de escrita **não** foi isolado (sem `strace`/`perf`).
- A hipótese "reduzir alocações melhora os dois lados" **não foi testada** — é próximo passo.
