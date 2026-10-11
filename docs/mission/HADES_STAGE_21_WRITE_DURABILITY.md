# Estágio 21 — política de durabilidade explícita: o `fsync` virou escolha declarada (E-168)

## 1. A recomendação que E-166/E-167 deixaram registrada

A medição mostrou que o **`fsync` domina** o custo da escrita atômica (1,08–1,93 ms de 2,6–3,6 ms) e que a
recomendação era *"agrupar ou tornar opcional o `fsync` — decisão de produto explícita, não ajuste silencioso"*.
Esta rodada **implementa** a decisão, preservando o padrão seguro.

## 2. O que existe agora

| Modo | Comportamento | Garantia |
| --- | --- | --- |
| **`durable` (PADRÃO)** | grava, `fsync` do arquivo **e** do diretório, renomeia | sobrevive a queda abrupta |
| `relaxed` | grava, renomeia (**troca atômica mantida**), **sem** `fsync` | **o último lote pode ser perdido** numa queda abrupta |

Configuração: **`HADES_WRITE_DURABILITY=durable|relaxed`** (ausente ou vazio = `durable`).

Três decisões que valem mais que o código:

1. **O padrão não mudou** — ninguém ganha velocidade por acidente.
2. **Valor inválido INTERROMPE a criação do runtime** (*fail-closed*). A alternativa ("cai para o seguro e
   ninguém soube") foi descartada de propósito: uma garantia de durabilidade trocada em silêncio é pior que um
   erro de configuração visível.
3. **O modo efetivo é VISÍVEL em `/health`**, com a **consequência declarada** no texto
   (`subsystems.escrita.status` + `detail`), e há teste travando os dois rótulos.

## 3. Quanto custa a garantia (medido, go1.26.9 dos dois lados)

| Ambiente | `durable` (padrão) | `relaxed` | Ganho |
| --- | --- | --- | --- |
| **Contêiner Linux** (200 memórias) | **3,39–3,52 ms** | **0,39–0,44 ms** | **≈8×** |
| Windows nativo (200 memórias) | 1,74–2,01 ms | 1,24–1,84 ms | ≈1,35× (ruidoso) |

O ganho no contêiner é grande porque o modo seguro faz **dois** `fsync` (arquivo **e** diretório) e o sistema de
arquivos em camadas cobra caro por cada um. O ganho no Windows é pequeno: **o valor da política depende do
sistema de arquivos** — e é exatamente por isso que ela é uma **escolha do operador**, não um padrão novo.

## 4. Testes

| Teste | O que trava |
| --- | --- |
| `TestWriteDurabilityDefaultsToSafe` | padrão é `durable` |
| `TestWriteDurabilityUnknownValueFallsBackAndDeclares` | valor desconhecido **recusa e declara**, não vira `relaxed` |
| `TestWriteDurabilityRelaxedKeepsAtomicWriteAndReadableFile` | mesmo sem `fsync`: arquivo **legível**, **sem temporário** sobrando |
| `TestWriteDurabilityFromEnvIsApplied` | o ambiente configura e o vazio volta ao seguro |
| `TestRuntimeDurabilityDefaultsToSafe` / `…AppliesConfiguredMode` / `…InvalidValueFailsLoudly` | runtime aplica; inválido **não cria runtime** |
| `TestHealthReportsWriteDurabilityPolicy` | `/health` relata o modo **e** o risco, nos dois casos |

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run 'TestWriteDurability\|TestRuntimeDurability'` | **7/7 PASS** |
| `go test ./server/ -run 'TestHealthReportsWriteDurabilityPolicy'` | **PASS** |
| `go test ./internal/agent/ -run '^$' -bench 'BenchmarkMemoryFileWrite' -benchtime=50x -count=3` | **OK** (números da tabela) |
| `go test ./internal/agent/ ./server/ -run '...'` | **PASS** (bateria da rodada) |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `gofmt -l internal/agent server` | **sem saída** |

## 6. Limites declarados

- **`relaxed` pode perder o último lote** numa queda abrupta — é a consequência, dita no `detail` do `/health` e
  não escondida. Não há mitigação: a escolha é entre velocidade e durabilidade.
- A política é **por processo** (global), não por arquivo/projeto: não há como pedir durabilidade só para o
  arquivo de um tenant.
- **Não** há sincronização em lote ("agrupar `fsync`"), que era a alternativa mais conservadora da recomendação:
  ficou registrada como **não feita**.
- O ganho foi medido numa máquina e num contêiner; **não** se generaliza para outros discos/sistemas.
- `benchtime=50x`/`-count=3`, sem p95/p99 nem concorrência.
- **Não** se testou a perda real sob queda de energia (seria destrutivo): a garantia é declarada a partir do que
  `fsync` significa, mais o teste de que ele está presente/ausente no caminho.
