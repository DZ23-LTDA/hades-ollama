# Persistência: o defeito que se repetiu DUAS vezes foi corrigido na RAIZ (E-127)

## 1. O sintoma, duas vezes

Na força de vendas (E-119) e de novo no marketplace (E-126), a **primeira** gravação falhou com
*"caminho não encontrado"*: `writeJSONAtomic` exigia um diretório que **ninguém criava**. A primeira
vez eu corrigi no store; a segunda vez ficou claro que o lugar certo era a **raiz**.

## 2. A correção

`writeJSONAtomicBounded` agora executa `os.MkdirAll(filepath.Dir(path), 0o700)` antes de criar o
arquivo temporário. Com isso, **todo** store — presente e futuro — deixa de depender de alguém
lembrar de criar o diretório. As duas correções locais redundantes foram **removidas**, para haver
um único mecanismo.

## 3. Guarda de regressão

`TestWriteJSONAtomicCreatesParentDirectory` grava em `a/b/c/dados.json` (aninhado, inexistente),
exige sucesso, confere que o arquivo existe, que o conteúdo volta como JSON e que o modo restrito
vale **onde há semântica POSIX** — no Windows o `Chmod` de arquivo reporta `0666` independentemente
do pedido, então afirmar isso ali mediria o sistema operacional, não a escrita.

## 4. O que a correção quebrou — e por que isso importa

A mudança **quebrou quatro testes de rollback** (`CompanyStore`, `ContextStore`, `AgentOrchestrator`
e `CollaborationStore`), porque eles injetavam falha de persistência apontando o store para um
**diretório inexistente** — e a escrita agora **cria** o diretório, então a falha desaparecia e o
rollback deixava de ser exercitado.

A saída **não** foi reverter a correção nem apagar os testes: a injeção passou a ser um **arquivo**
no lugar da raiz (`unwritableStoreRoot` e o bloqueio equivalente em `swarm_test.go` e
`collaboration_test.go`), o que faz `MkdirAll` falhar de forma **portátil** em Linux, macOS e
Windows. Os quatro testes voltaram a exercitar o rollback com a asserção original intacta.

**Lição registrada**: corrigir uma causa raiz pode **desarmar silenciosamente** testes que usavam o
defeito como mecanismo de injeção. Procurar por "quem depende deste erro" faz parte da correção —
e aqui a busca só apareceu porque a suíte inteira foi rodada, não apenas os testes novos.

## 5. Erro meu no processo (registrado)

Ao reescrever as injeções por script, perdi o `store.root = ` em uma das substituições, e os quatro
testes passaram a **não falhar por não injetar nada** — situação que se parece com sucesso. O
sintoma (`company update unexpectedly succeeded`) foi o que denunciou. Também deixei aspas
escapadas (`\"`) em três arquivos, que o compilador recusou. Nenhum dos dois foi escondido: as
linhas foram reescritas de forma canônica e verificadas uma a uma.

## 6. A injeção por ARQUIVO não é equivalente entre sistemas — e o CI provou isso

O PR foi **recusado pelo CI** em `TestJSONStoreRollsBackMissionMutationWhenPersistenceFails`:

```
store_atomic_test.go:54: failed new mission remained in memory:
lstat /tmp/.../not-a-directory/store: not a directory
```

**Causa medida, não suposta**: com a raiz bloqueada por um **arquivo**, ler um caminho sob ele
devolve **ENOTDIR** no Linux — e o teste esperava `os.ErrNotExist`. No Windows o mesmo acesso mapeia
para "não encontrado", então o teste passava **aqui** e falhava **lá**. A injeção não era
equivalente entre sistemas, e o teste passou a medir o **sistema de arquivos** em vez de medir a
ausência da missão.

**Correção**: a raiz volta a ser válida **antes** da leitura, o que verifica as **duas** coisas de
forma portátil — nada em memória **e** nada no disco. A asserção original foi preservada, nenhum
teste foi removido e a correção de raiz (`writeJSONAtomic` criando o diretório) permanece.

**Lição**: uma injeção de falha trocada precisa ser validada **no sistema onde o gate roda**, não
apenas onde se desenvolve. Rodar a suíte inteira localmente **não bastou**, porque a divergência era
do sistema de arquivos, não do código.
