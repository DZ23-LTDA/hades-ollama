# Correção — fim das "falhas ambientais": um defeito real e o limite de caminho do Windows (E-116)

## 1. Ponto de partida

Eu vinha declarando **3 falhas ambientais** de teste e as tratava como "específicas do host,
passam no CI". Na rodada anterior uma delas se revelou **defeito real** (E-115). Decidi
investigar as restantes em vez de continuar repetindo o rótulo — e o rótulo **não se
sustentou**.

## 2. O que a investigação encontrou

### 2.1 DEFEITO REAL: a inicialização do repositório importado NÃO era idempotente

Com um diagnóstico que roda a sequência de `git` passo a passo, a causa apareceu:

| Chamada | Resultado |
| --- | --- |
| **Primeira** `initializeImportedRepository` | **sucesso** (cria repositório e commita o snapshot) |
| **Segunda** chamada, mesma árvore | **falha**: `On branch master / nothing to commit, working tree clean` |

O passo exigia `exit 0` do `git commit`, e o git retorna **1** quando **não há mudanças para
commitar**. Consequência prática: **reimportar o mesmo projeto, ou retomar uma importação
interrompida, QUEBRAVA** — exatamente o caminho de **retomada** que os estágios 10/11 exigem.
É a família recorrente deste projeto: uma ação que **não** fecha o que abriu.

### 2.2 LIMITE DE CAMINHO DO WINDOWS: `Filename too long` dentro do `.git`

O erro tipado que eu tinha acabado de adicionar **revelou** a causa real no outro teste:

```
error: unable to write file .git/objects/62/9ada…: Filename too long
error: attachments/01-notes.txt: failed to insert into database
fatal: adding files failed
```

O caminho da raiz tinha **211 caracteres** e o caminho **interno do git**
(`<raiz>/.agent-imports/<id>/origin/.git/objects/xx/<38 caracteres>`) ultrapassa o
**MAX_PATH = 260**. O `t.TempDir()` do Go ainda acrescenta o **nome do teste**, e este
repositório é fundo: com `TMP` dentro do próprio repositório, o limite é atingido sempre.

## 3. O que foi corrigido

| Correção | Onde | Natureza |
| --- | --- | --- |
| **Idempotência**: só commita quando há mudanças; valida `HEAD` ao final | `internal/agent/project_import.go` | **DEFEITO REAL** (retomada/reimportação) |
| **Erro tipado e acionável** `ErrImportPathTooLong` com o limite, o caminho e o que fazer (`diretório mais curto` ou `LongPathsEnabled`) | `internal/agent/project_import.go` | usabilidade e diagnóstico: antes o usuário via apenas `exit status 128` |
| **Raiz curta nos testes** (sem o nome do teste), devolvendo o orçamento de MAX_PATH | `short_root_test.go` (agente e servidor) | teste passa a medir **importação**, não comprimento de caminho |

Nada foi desativado, nada virou `t.Skip` para "ficar verde", e nenhum `//nolint` foi usado.

## 4. Verificação (esta máquina, Windows)

```
go test ./internal/agent/ -count=1 -timeout 30m   → ok   (54,9 s)
go test ./server/        -count=1 -timeout 30m   → ok   (20,6 s)
golangci-lint run ./internal/agent/... ./server/... → 0 issues
```

**Pela primeira vez neste workspace as DUAS suítes passam inteiras, sem nenhuma falha
ambiental.** Os testes novos são as guardas:

- `TestInitializeImportedRepositoryIsIdempotent`: duas chamadas seguidas passam; **sem**
  mudanças o commit **não** muda; **com** mudança real gera novo commit; a árvore termina
  **limpa**.
- `TestProjectImportReportsLongPathClearly`: caminho acima de 260 devolve
  `ErrImportPathTooLong` com mensagem **acionável**.
- `TestLargeUploadProjectImportStreamsZipAndCreatesIsolatedWorktree` e
  `TestImportMissionAttachmentsCreatesIndexedOrgProject`: **PASS** (antes falhavam).

## 5. Estado corrigido do bloqueador B-09

O B-09 dizia "4 testes falham por ambiente, provados pré-existentes". Hoje:

| Teste | Antes declarado | Realidade medida |
| --- | --- | --- |
| kill de process group | "`sh` ausente" | **teste frágil** (limite de 2s + dependência de `sh`) — corrigido (E-115) |
| import de upload/worktree | "ambiente, exit 128" | **defeito real de idempotência** + MAX_PATH — corrigido (E-116) |
| `TestImportMissionAttachmentsCreatesIndexedOrgProject` | "`Filename too long` ambiental" | **MAX_PATH** no `.git` — tratado com erro acionável e raiz curta nos testes (E-116) |
| browser operator | Python ausente | permanece: depende de Python instalado (declarado) |

**Resultado:** de 4 falhas declaradas para **0** falhas nas suítes locais. O rótulo
"ambiental" escondia **um defeito de produto**; foi por isso que valeu investigar em vez de
repetir a explicação.
