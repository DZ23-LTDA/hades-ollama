# Estágio 21 — otimização da escrita: resultado NEGATIVO medido (E-165)

## 1. O trabalho derivado de E-164

E-164 mediu a escrita de 200 memórias em **1,66–1,76 ms / 25 allocs** (Windows) e **2,51–2,70 ms / 224–226
allocs** (contêiner) e registrou o achado acionável: *"a escrita alocar ~225 vezes contra 25 é alvo de otimização
que beneficia as duas plataformas"*. Esta rodada executou esse trabalho.

## 2. Duas tentativas, duas medições, nenhum ganho

| Variante | `BenchmarkMemoryFileWrite` (Windows) | Memória | Veredito |
| --- | --- | --- | --- |
| **Base** (`MarshalIndent` + um `Write`) | **1,66–1,76 ms** · 25 allocs | 102–106 KB | melhor medido |
| `json.Encoder` direto no arquivo (sem buffer) | **2,13–2,56 ms** · 24–25 allocs | 91–96 KB | **PIOR** |
| `json.Encoder` + `bufio` de 64 KB | **1,79–2,63 ms** · 26–27 allocs | **157–165 KB** | empatou e gastou mais |

**Por que a primeira tentativa piorou**: sem buffer, o encoder faz **centenas de escritas pequenas** (syscall por
linha); um `Write` único do documento inteiro custa menos que isso. O `bufio` devolveu a escrita em bloco, mas o
ganho foi nulo e o custo de memória por chamada subiu.

## 3. A hipótese foi REFUTADA pela medição

A premissa era *"menos alocações = mais rápido"*. As três variantes alocam **praticamente o mesmo** (24–27), o que
mostra que o buffer do `MarshalIndent` **não é** o gargalo: o custo está no `Sync`/`Rename` e na serialização em
si. A conclusão honesta é **desfazer a mudança** e registrar o resultado, não manter código mais complexo com
desempenho igual ou pior.

## 4. Um defeito real encontrado no caminho (pelo teste novo)

A variante com `bufio` introduziu um defeito grave: o erro do `Flush` era atribuído a um `err` **sombreado**
dentro do `if`, e a função seguia para `Sync`/`Rename` retornando **nil**. Efeito prático: o **limite de tamanho
deixava de valer** e um valor acima do limite seria gravado. O teste novo
(`TestWriteJSONAtomicBoundedRefusesOversizedValue`) pegou na hora.

**Sem o teste, o defeito entraria como "otimização"** — e o limite de persistência, que existe para impedir
gravação sem teto, ficaria desligado em silêncio.

## 5. O que ficou desta rodada

1. **Produção idêntica à `main`**: `internal/agent/store.go` foi **restaurado** do `origin/main` e recebeu apenas
   um comentário com o resultado medido, para ninguém repetir a tentativa sem nova medição.
2. **Dois testes de contrato novos** (`internal/agent/store_json_atomic_test.go`):
   - `TestWriteJSONAtomicKeepsTheExactFormat` — os bytes gravados continuam **idênticos** ao formato anterior
     (indentação de dois espaços + quebra final) e o arquivo continua JSON válido. Era um contrato **sem teste**:
     os arquivos do Hades são lidos por outras instâncias e por ferramentas externas.
   - `TestWriteJSONAtomicBoundedRefusesOversizedValue` — o limite **recusa**, a **mensagem** continua a mesma
     (`exceeds persistence limit`), **nada** é gravado no caminho final, e um valor dentro do limite continua
     sendo gravado.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run TestWriteJSONAtomic -count=2` | **6/6 PASS** |
| `go test ./internal/agent/ ./server/ -run '...'` | **PASS** (bateria da rodada) |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `git diff origin/main -- internal/agent/store.go` | apenas o comentário (restaurado do `main`) |

## 7. Limites declarados

- Medições em **uma** máquina (Windows nativo) para a decisão de reverter; o contêiner tem o mesmo código, mas
  **não** foi re-medido nas três variantes — o que **não** permite afirmar que o resultado vale igualmente lá.
- `benchtime=50x`, `-count=3`: sem p95/p99.
- **Não** foi isolado o custo interno (`Sync`, `Rename`, `MkdirAll`, serialização): sabe-se que **não** é o buffer
  do `MarshalIndent`, mas **não** se sabe quanto cada etapa pesa — próximo passo possível, com `strace`/`perf`.
- A otimização **continua pendente**: esta rodada eliminou duas hipóteses, não encontrou o ganho.
