# Estágio 21 — o "mistério do JSON" era versão de Go: três evidências minhas corrigidas (E-167)

## 1. O que eu tinha medido, e o que estava errado

| Afirmação anterior | O que a medição de agora mostra |
| --- | --- |
| E-166: *"o JSON sai com **98 KB no contêiner contra 71 KB** no Windows"* | **ERRADO — eu li `B/op` (bytes ALOCADOS) como se fosse o tamanho do arquivo.** O JSON de fato tem **34.983 bytes nos dois lados**, byte a byte |
| E-166: *"o gap de alocações está no `json.MarshalIndent` … 204 no contêiner contra 4 no Windows"* | **Meia verdade perigosa**: o gap existe, mas **não é da plataforma** — é da **versão do Go** (ver abaixo) |
| E-161/E-164: *"o turno é ~40% mais lento no Linux"* | Comparação **CONFUNDIDA**: o Windows rodava **go1.27.2** e o contêiner **go1.26.9**. Eram **duas** variáveis (SO + toolchain), não uma |

## 2. A causa: o toolchain local havia derivado do fixado

```
_tmp/goroot/go/bin/go.exe version   ->  go version go1.27.2 windows/amd64
go.mod                              ->  go 1.26.9
.github/workflows/*                 ->  go-version-file: go.mod   (ou seja: go1.26.9)
contêiner golang:1.26.9             ->  go version go1.26.9 linux/amd64
```

**Consequência prática que importa mais que o benchmark**: minha validação local rodava em **1.27.2** enquanto o
CI e o gate Linux rodam **1.26.9** — toolchain **diferente do que aprova o repositório**. Medições e até
resultados de teste locais podiam divergir do gate, e as comparações que publiquei misturavam SO com versão.

## 3. Correção do instrumento antes de continuar

Baixei o toolchain **fixado** (`go1.26.9.windows-amd64.zip`, dentro do próprio workspace em
`_tmp/goroot1269`) e repeti as medições com **o mesmo Go dos dois lados**. É esse o dado válido:

| Fase | Windows · **go1.26.9** | Contêiner · **go1.26.9** | Leitura |
| --- | --- | --- | --- |
| `Marshal` (sem I/O) | 0,35–0,48 ms · **204 allocs** | 0,33–0,36 ms · **203–204 allocs** | **IGUAL** |
| `MkdirAll` | 0,016–0,041 ms | 0,0010–0,0011 ms | Windows ~20–30× mais lento |
| `CreateTemp` | 0,35–0,39 ms | 0,017–0,020 ms | Windows ~20× mais lento |
| `Sync` (fsync) | **1,08–1,35 ms** | **1,79–1,93 ms** | Linux ~1,5× mais lento |
| `Rename` | 0,86–0,92 ms | 0,16–0,17 ms | Windows ~5× mais lento |
| **`MemoryFileWrite` (real)** | **2,56–2,63 ms · 223–225 allocs** | 3,48–3,64 ms · 224–226 allocs | Linux ~1,4× mais lento |
| `ServiceAgentTurn` | 2,00–2,53 ms · 203–204 allocs | 3,22–3,37 ms · 206–208 allocs | Linux ~1,4× mais lento |

## 4. O que os dados corrigidos dizem

1. **Alocação igual entre plataformas** (204 × 203–204) com o mesmo Go ⇒ **não existe gap de alocação por SO**.
   A versão 1.27.2 aloca **4**; a 1.26.9 aloca **204** para a mesma saída de 34.983 bytes. Go 1.27 mudou o
   comportamento de `encoding/json` — e é isso, não o sistema de arquivos.
2. **Formato estável**: os bytes são **idênticos** nos dois lados (e determinísticos — agora há teste). A
   preocupação de que um Hades não lesse o arquivo do outro **não se confirmou**.
3. **A diferença de SO real é mista**: o Windows é **muito mais lento** em `CreateTemp` e `Rename` e **mais
   rápido** em `fsync`; no total do arquivo de 200 memórias o contêiner fica ~**1,4×** mais lento — e **não** os
   1,6–1,9× que eu havia publicado, porque lá a base Windows estava medida com o toolchain errado (1,66–1,76 ms
   com 1.27.2 contra 2,56–2,63 ms com 1.26.9).
4. **`fsync` continua dominando** (1,08–1,93 ms): a conclusão de E-166 sobre onde está o tempo **sobrevive** à
   correção — quem estava errado era a atribuição do gap de alocações.

## 5. Lição institucionalizada

**Versionar o instrumento junto com o artefato.** Uma comparação entre plataformas só vale se o **toolchain**
estiver fixado nos dois lados; sem isso, o resultado mede a soma de duas variáveis e a conclusão sai errada com
aparência de número. A partir desta rodada, a validação local usa `_tmp/goroot1269` (**o Go do CI**), e não o
toolchain mais novo que estava no `_PATH`.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `_tmp/goroot1269/go/bin/go.exe version` | **go1.26.9 windows/amd64** (o do CI) |
| `go test ./internal/agent/ -run TestDiagnoseJSONEnvironment -count=1` | **PASS** — bytes 34.983 iguais, 204 allocs, ida e volta e determinismo verificados |
| `go test ./internal/agent/ -run '^$' -bench 'BenchmarkWritePhase\|BenchmarkMemoryFileWrite\|BenchmarkServiceAgentTurn' -benchtime=50x -count=3` | **OK** (números acima) |
| `go test ./internal/agent/ ./server/ -run '...'` | **PASS** (bateria da rodada) |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `gofmt -l internal/agent server` | **sem saída** |

## 7. Limites declarados

- Windows com o toolchain **baixado dentro do workspace** (`_tmp/goroot1269`) — passou a ser o padrão desta
  sessão; o toolchain 1.27.2 continua instalado na máquina e **pode voltar** se o `PATH` mudar.
- Uma máquina física; Linux em contêiner (não SO nativo).
- `benchtime=50x`/`-count=3`, sem p95/p99 nem concorrência.
- **Não** investiguei *o que* em `encoding/json` mudou entre 1.26.9 e 1.27.2 (só o efeito medido).
- A conclusão de E-166 sobre `fsync` foi mantida, mas os **números** dela para Windows (1,66–1,76 ms) ficam
  **substituídos** pelos desta rodada (2,56–2,63 ms), medidos no toolchain correto.
