# Estágio 21 — `fsync` em lote: a alternativa conservadora (E-169)

## 1. O que faltava da recomendação

E-168 entregou `durable` (padrão) e `relaxed`. A recomendação original de E-166/E-167 também previa *"agrupar o
`fsync`"* — o meio-termo que ficou registrado como **não feito**. Esta rodada implementa o modo **`batched`**.

| Modo | Sincroniza | Perda possível numa queda abrupta |
| --- | --- | --- |
| `durable` (PADRÃO) | toda escrita (arquivo + diretório) | nenhuma escrita já confirmada |
| **`batched`** | no máximo **1× por janela de 2 s** | **limitada à janela** (o que passou de 2 s já é durável) |
| `relaxed` | nunca | **tudo desde o início do processo** |

A diferença que importa entre `batched` e `relaxed` **não é velocidade** (medida abaixo): é o **LIMITE** da
perda. Com `relaxed`, a exposição cresce sem teto; com `batched`, ela é sempre ≤ 2 s.

## 2. Medição dos três modos (go1.26.9 nos dois lados, 200 memórias)

| Modo | Contêiner | Windows |
| --- | --- | --- |
| `durable` (padrão) | **3,22–3,29 ms** | 2,09–2,14 ms |
| **`batched`** | **0,43–0,45 ms** | 1,79–2,07 ms |
| `relaxed` | 0,43–0,47 ms | 2,09–2,26 ms |

No contêiner, `batched` ≈ `relaxed` no tempo — e isso é **esperado**: em escrita densa quase tudo cai dentro da
janela e só a primeira escrita sincroniza. No Windows os três ficam dentro do ruído (o `fsync` já era barato ali,
como E-167 mostrou). **O ganho do `batched` é o limite da perda, não a velocidade.**

## 3. Visibilidade e fechamento da janela

- `/health` → `subsystems.escrita` agora traz `status`, `detail` **e `pendentes`**: a janela em aberto é
  **medida**, não prometida.
- `FlushDurableWrites()` fecha a janela sob demanda (encerramento ordenado). Dizer que existe um flush e não
  documentar quem chama seria meio-caminho: **o flush está disponível e testado**; o que **não** foi feito é
  ligá-lo a um hook de encerramento do servidor (limite declarado).
- `PendingDurableWrites()` devolve quantas escritas estão gravadas e ainda não sincronizadas.

## 4. Defeito real de plataforma encontrado pelo teste

A primeira implementação sincronizava o arquivo **reabrindo-o para leitura** (`os.Open` + `File.Sync()`). No
Windows isso falha com **"Acesso negado"** — `FlushFileBuffers` exige acesso de escrita. O teste do modo em lote
pegou na primeira execução. Corrigido em dois lugares: a escrita sincroniza o **próprio handle de escrita** antes
do `Close`, e o flush posterior abre com `os.O_RDWR`.

**Sem o teste, o modo em lote seria entregue quebrado no Windows** — e quebrado justamente no caminho que existe
para proteger dados.

## 5. Erro meu repetido (registrado)

Usei *here-string* do PowerShell para editar Go com **crases** no comentário: a crase é caractere de escape no
PowerShell e o comentário saiu corrompido (`atched grava`, `elaxed`), quebrando a compilação. É a **segunda vez**
nesta sequência (mesma classe da rodada 122, com backticks em `python -c`). Regra prática: **edição de código com
crases vai por arquivo de script ou pela ferramenta de edição — nunca por here-string.**

## 6. Testes

| Teste | O que trava |
| --- | --- |
| `…BatchedKeepsAtomicWriteAndReadableFile` | troca atômica, arquivo legível e **sem temporário** no modo em lote |
| `…BatchedSyncsFirstWriteAndCountsPending` | a **primeira** escrita sincroniza; a seguinte fica **pendente e contabilizada**; o flush zera |
| `…BatchedIsNotTheDefault` | o modo em lote é **escolha**: o padrão segue `durable`, e fora dele não há pendência |
| `…BatchedRejectsUnknownValue` | valor desconhecido recusa e cai para o seguro |

## 7. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run 'TestWriteDurability\|TestRuntimeDurability\|TestWriteJSONAtomic' -count=2` | **PASS** |
| `go test ./server/ -run TestHealthReportsWriteDurabilityPolicy` | **PASS** |
| `go test ./internal/agent/ -run AAA -bench BenchmarkMemoryFileWrite -benchtime=50x -count=2` | **OK** nos dois ambientes (números acima) |
| `go test ./internal/agent/ ./server/ -run '...'` | **PASS** (bateria da rodada) |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `gofmt -l internal/agent server` | **sem saída** |

## 8. Limites declarados

- A janela é **de tempo (2 s)**, não configurável por enquanto, e é **por processo**.
- **Não** há sincronização em segundo plano: o lote só fecha quando **outra escrita** acontece depois da janela
  ou quando `FlushDurableWrites()` é chamado. Um processo que escreve uma vez e fica ocioso mantém aquela escrita
  pendente **até o flush** — é um limite real do desenho atual.
- **Não** há hook de encerramento ligado ao flush (o servidor não chama `FlushDurableWrites` no shutdown).
- `pendentes` conta **caminhos**, não bytes nem registros.
- Medições em **uma** máquina e **um** contêiner; `benchtime=50x`/`-count=2`, sem p95/p99.
- **Não** se testou perda real sob queda de energia (seria destrutivo).
