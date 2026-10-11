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
