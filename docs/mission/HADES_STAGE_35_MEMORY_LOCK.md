# Estágio 35 — trava entre processos no arquivo de memórias (E-163)

## 1. O limite declarado, e o defeito que ele escondia

E-162 declarou: *"não há trava entre PROCESSOS no mesmo arquivo — último-escritor continua valendo"*. O que
isso significa na prática: cada instância do Hades mantém a **própria cópia** das memórias do projeto. Se a
instância A recalcula vetores enquanto a instância B grava uma memória nova, quem gravar por último impõe a sua
cópia — e o trabalho do outro **desaparece em silêncio**.

A infraestrutura de trava **já existia** no repositório (`file_lock_unix.go` com `flock`,
`file_lock_windows.go` com `LockFileEx`, usada por filas e push). O caminho do arquivo de memórias era o que
estava fora dela. **Nada de implementação nova de lock**: só uso onde faltava (lição de duplicação de E-157,
aplicada antes de escrever código).

## 2. O que mudou

| Peça | Comportamento |
| --- | --- |
| `memoryLockPath` / `memoryFilePath` | lock e arquivo do projeto, num só lugar |
| `updateMemoriesFileLocked` | **ler de novo → alterar → gravar**, tudo dentro da trava |
| `ErrStoreLocked` | recusa **declarada** quando outro dono segura a trava |
| `MemoryFileLockTimeout = 10s` | espera limitada: um dono travado não pendura quem só quer gravar |

O re-embed **síncrono** e o **job por lote** passaram a usar a atualização travada por ID: a nossa cópia em
memória deixa de ser a fonte da verdade na hora de gravar.

## 3. O teste que encontrou um defeito real na trilha que eu NÃO tinha corrigido

Escrevi dois testes — um para o job, outro para o síncrono —, cada um simulando outro processo gravando no
arquivo antes do re-embed:

- **job por lote: PASS** (eu havia corrigido esse caminho);
- **síncrono: FAIL** — *"o re-embed apagou o trabalho de outro processo"*.

Ou seja: o mesmo defeito de perda silenciosa continuava vivo no `ReembedMemories`. Corrigido com a mesma
atualização travada por ID. **Se eu tivesse testado só o caminho que acabei de escrever, o defeito iria para a
`main` com teste verde** — é a repetição exata da lição de E-158 no mesmo dia seguinte.

## 4. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run '...\|TestReembedDoesNotErase\|TestReembedJobDoesNotErase\|TestMemoryWriteLock'` | **106/106 PASS** |
| `go test ./internal/agent/ -count=1` | **ok 73,600s** |
| `go test ./server/ -count=1` | **ok 23,077s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agente 24,636s · servidor 7,178s) |
| `gofmt -l internal/agent server` | **sem saída** |

Os testes provam: o re-embed **preserva** o que outro processo gravou (síncrono **e** job), as memórias do
próprio projeto continuam todas, o espaço novo é carimbado, e um lock disputado resulta em `ErrStoreLocked`
declarado (com `t.Skip` honesto se o sistema de arquivos não bloquear).

## 5. Melhorias de higiene no mesmo passo

- **`unused: 1`**: eu havia escrito `writeMemoriesFileLocked` e não usei — removida em vez de silenciada.
- **`nolintlint: 2`**: as diretivas `//nolint:misspell` em `company.go` e `desktop.go` ficaram **sem uso**
  porque o `ignore-rules` do `misspell` cobriu as palavras portuguesas. Removidas: **duas supressões a menos**,
  nenhuma nova. E `processos` entrou no `ignore-rules` (idioma, não permissão).

## 6. Limites declarados

- **Cobre o arquivo de memórias** (leitura-modificação-escrita do re-embed). Outros stores do Hades têm as suas
  próprias travas onde precisam — e onde não têm, o limite continua sendo o de antes.
- **A fusão é por ID, campo a campo de vetor**: se outro processo alterar o **texto** da mesma memória entre a
  nossa leitura e a gravação, prevalece o que está no arquivo para os campos que não são vetor.
- **A espera é limitada a 10 s** e a falha é declarada; **não há roubo de trava** (um dono morto que deixou o
  arquivo travado bloqueia escritas até o processo morrer ou o SO liberar o lock).
- **Não há fila de escritas**: sob disputa, quem perde recebe erro e decide o que fazer.
- **A trava é advisory**: um processo que escreva no arquivo **sem** passar pelo store não é impedido.
