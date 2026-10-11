# Estágio 21 — fases da escrita isoladas: onde o tempo realmente está (E-166)

## 1. A pergunta que E-165 deixou aberta

E-165 eliminou duas hipóteses de otimização e registrou, com honestidade: *"o custo interno (`Sync`, `Rename`,
`MkdirAll`, serialização) **não foi isolado** — sabe-se que **não** é o buffer do `MarshalIndent`, mas não se sabe
quanto cada etapa pesa"*. Esta rodada isola cada fase.

Método: benchmarks que **replicam a sequência real** de `writeJSONAtomicBounded` (mesma carga de 200 memórias,
mesmas chamadas, mesma ordem) — não a função, porque desligar o `Sync` exigiria alterar produção só para medir.
O benchmark da **função real** (`BenchmarkMemoryFileWrite`) serve de **controle**: se a soma das fases não
fechasse com ele, o método estaria errado. **A soma fecha.**

## 2. Resultado (200 memórias, `benchtime=50x`, `-count=3`)

| Fase | Windows (nativo) | Linux (contêiner) | Observação |
| --- | --- | --- | --- |
| **Função REAL** (controle) | 2,09–2,23 ms · 24 allocs | **3,48–3,64 ms · 224–226 allocs** | o que se quer explicar |
| `Marshal` (serialização, **sem I/O**) | 0,20–0,25 ms · **4 allocs** · 71 KB | 0,33–0,36 ms · **204 allocs** · 98 KB | **6–9% do tempo** |
| `MkdirAll` (diretório existente) | 0,02–0,03 ms | **0,001 ms** | irrelevante |
| `CreateTemp` + `Chmod` + remoção | 0,32–0,43 ms | **0,017–0,020 ms** | Windows paga caro aqui |
| **`Sync`** (criar + escrever + **fsync**) | **1,20–1,30 ms** | **1,79–1,93 ms** | **~41% (Win) e ~50% (Linux)** |
| `Rename` (criar + escrever + renomear, **sem** fsync) | 1,08–1,61 ms | **0,16–0,17 ms** | troca atômica |

## 3. Conclusões que os dados sustentam

1. **O tempo é dominado pelo sistema de arquivos, não pela serialização.** Serializar 200 memórias custa
   **0,20–0,36 ms**; `fsync` sozinho custa **1,2–1,9 ms**. A serialização é **6–9%** do total.
   Isso **explica o resultado negativo de E-165**: as duas tentativas mexeram na serialização — os **~9%** —, então
   nenhuma podia mudar o quadro. A medição não só eliminou hipóteses como **mostra por que** elas eram as erradas.
2. **`Sync` (durabilidade) é o maior item isolado**: ~41% no Windows e ~50% no Linux. É uma **escolha
   deliberada** (sobreviver a queda de energia/processo), então o alvo de otimização é a **política de
   durabilidade** — uma decisão de produto sobre risco de perder escritas —, **não** um ajuste silencioso de
   código.
3. **A diferença de ALOCAÇÕES entre plataformas está no `json.MarshalIndent`**, e não no I/O: 204 allocs na
   serialização no contêiner contra 4 no Windows, enquanto as fases de arquivo alocam **7 nos dois**. E o
   **tamanho** do JSON difere: **98 KB no contêiner contra 71 KB** no Windows, para a mesma struct.
4. **Isto corrige a atribuição de E-164.** E-164 disse que a diferença estava "no caminho de escrita". Com as
   fases separadas, a precisão é: **o TEMPO** se explica por `fsync` + `rename` (que é ~7× mais rápido no
   contêiner que no Windows: 0,17 vs 1,18 ms); **as ALOCAÇÕES** se explicam **inteiramente pela codificação
   JSON**, dentro do contêiner.

## 4. O que NÃO está explicado (declarado)

- **Por que `json.MarshalIndent` aloca 204 vezes e produz 98 KB no contêiner contra 4 allocs e 71 KB no Windows
  para a MESMA struct e o MESMO comando?** Causa **não determinada**. Não é o instrumento (foi o mesmo comando,
  a mesma versão de Go 1.26.9 e o mesmo tamanho de carga) e não é o I/O (as fases de arquivo alocam igual).
  Hipóteses a testar em rodada própria: diferença de implementação de `encoding/json` habilitada por ambiente,
  efeito de `sync.Pool`/GC sob o cgroup do contêiner, ou codificação de texto do arquivo-fonte. **Fica como
  pergunta aberta com o número medido**, não como conclusão.
- **Por que o `Rename` e o `CreateTemp` são muito mais baratos no contêiner** também **não** foi isolado
  (sistema de arquivos em camadas contra NTFS é a hipótese, não a medição).
- Tudo medido em **uma** máquina física, Windows nativo × Linux em contêiner.

## 5. Recomendação acionável (não executada)

O ganho real, se houver, está na **política de durabilidade**: agrupar `fsync` (uma sincronização para um lote de
escritas), torná-lo opcional por configuração, ou separar "gravação durável" de "gravação rápida" com declaração
explícita de risco. Isso muda a garantia contra queda de energia — **decisão de produto que precisa ser
explícita** — e por isso **não** foi feita nesta rodada. A hipótese alternativa (ganho por menos alocações) foi
refutada em E-165 e agora **explicada** aqui.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run '^$' -bench 'BenchmarkWritePhase|BenchmarkMemoryFileWrite' -benchtime=50x -count=3` | **OK** em Windows e no contêiner (números acima) |
| `go test ./internal/agent/ ./server/ -run '...'` | **PASS** (bateria da rodada) |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `gofmt -l internal/agent server` | **sem saída** |

## 7. Limites declarados

- As fases são uma **réplica** da sequência (a função real não foi alterada para medir); a credibilidade vem de a
  soma fechar com o controle — **2,2 ms** somados contra **2,09–2,23 ms** medidos.
- `benchtime=50x`, `-count=3`: sem p95/p99 nem concorrência.
- Uma máquina física; Linux em contêiner, não em SO nativo.
- **Duas causas permanecem abertas** (alocações/tamanho do JSON no contêiner; custo de `rename`/`createtemp` por
  plataforma) e a **recomendação de política de durabilidade não foi executada**.
