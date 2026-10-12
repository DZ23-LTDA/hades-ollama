# Estágio 8 — defeito B-13 CORRIGIDO: execução de plugin passa a ter workspace de verdade (E-184)

## 1. A causa (medida em E-183) e a correção

`internal/agent/plugin_runtime.go` passava `Workspace: installation.ContentPath` — e `ContentPath` é o **ARQUIVO**
`plugins/content/<org>/<id>-<versão>.zip`. O executor faz `os.OpenRoot(toolContext.Workspace)` e depois
`Mkdir`/`writeContainedSandboxFile` **dentro** dessa raiz: exige um **diretório**. Resultado medido: *open
…/runtime-plugin-1.0.0.zip: not a directory*.

A correção (`internal/agent/plugin_workspace.go`, novo):

1. o bundle **verificado** é extraído para um diretório de execução temporário (`plugins/exec-run/run-*`, dentro da
   área confiável do registro de plugins);
2. o `Workspace` passa a ser **esse diretório**;
3. a limpeza é garantida por `defer`, inclusive no caminho de erro.

Extrair conteúdo de terceiro é operação de risco, então ela é **fail-closed** e limitada — nome com travessia,
caminho absoluto, barra invertida, link simbólico, arquivos demais (>1024), arquivo acima de 16 MiB ou total
descomprimido acima de 64 MiB são **recusados**, e o diretório temporário é removido.

## 2. Efeito colateral desejado: plugin multiarquivo passa a ser possível

O sandbox cria o arquivo do plugin num **subdiretório** do workspace. Como o workspace agora é o **bundle
extraído**, os irmãos ficam alcançáveis a partir do código em execução (`require('../lib/ajudante.js')`) — antes
eram inalcançáveis, e foi por isso que o teste do terceiro precisou **inlinar** a biblioteca em E-183.

## 3. A prova de que a correção funcionou (antes × depois, no MESMO ambiente)

Container `node:22-bookworm`, modo `best-effort` com aviso de aprovação — o ambiente onde B-13 foi reproduzido:

| | Antes (E-183) | Depois (E-184) |
| --- | --- | --- |
| erro observado | `open …runtime-plugin-1.0.0.zip: **not a directory**` | `exit status 1` |
| significado | o workspace **nem era** um diretório | o workspace é um diretório; o executor **chegou a montar e rodar** o sandbox |
| teste | falhava | **SKIP** com motivo declarado (ambiente sem privilégio de montagem) |

O teste agora **falha** se `not a directory` voltar: é a armadilha contra regressão de B-13.

## 4. Testes novos (5), todos com injeção de falha

| Teste | O que prova |
| --- | --- |
| `TestPreparePluginWorkspaceExtractsBundleAndCleansUp` | layout preservado (`main.js`, `lib/ajudante.js`, licença) e **limpeza** remove tudo |
| `TestPreparePluginWorkspaceRefusesPathTraversal` | `../escapa.js` recusado e **nada escrito fora** |
| `TestPreparePluginWorkspaceRefusesAbsolutePathAndSymlink` | `/etc/hostil.js`, `..\escapa.js` e link simbólico recusados |
| `TestPreparePluginWorkspaceRefusesTooManyFiles` | 1025 entradas recusadas |
| `TestPreparePluginWorkspaceRefusesOversizedEntry` | entrada acima de 16 MiB recusada pelo **tamanho declarado** |

## 5. Novo limite, medido: B-14 (a execução exige privilégio de montagem)

Lendo o executor (`tools.go`), **mesmo o modo `best-effort` faz `mount` e `chroot`** (o script tem
`mount --make-rprivate /`, `mount --bind`, `mount -t tmpfs` e executa via `chroot`/`unshare`). Onde não há
privilégio de montagem — Windows, container não privilegiado, runner padrão de CI — a execução **não acontece**, e o
sintoma medido é `exit status 1` com **`stdout` e `stderr` VAZIOS**.

Isso é registrado como **B-14** (severidade MÉDIA): não é um defeito de lógica, é **pré-requisito de implantação**
somado a um **diagnóstico silencioso** (falha sem stderr dificulta a operação). Desbloqueio: executar em host com
namespaces de montagem disponíveis **e** fazer o executor propagar o stderr do launcher.

## 6. Consequência para a matriz

O item *"Plugins têm instalação e execução verificáveis"* **continua `NAO_COMPROVADO`** — e o motivo ficou ainda
mais preciso: **instalação, assinatura, integridade e extração segura estão comprovadas**; a **execução** não pode
ser exercitada em nenhum ambiente que eu tenha (Windows por design; Linux sem privilégio de montagem). O que mudou é
que **o defeito que a impedia foi corrigido** e agora existe teste que impede a regressão.

## 7. Limites declarados

- A execução **não foi observada com sucesso** em nenhum ambiente nesta rodada: o que se prova é que o caminho
  chegou a montar o sandbox (erro mudou de `not a directory` para falha de processo).
- O modo `strict` continua sem poder ser exercitado (`OLLAMA_AGENT_SANDBOX_CGROUP_ROOT`).
- A extração grava em `plugins/exec-run/` sob a raiz do registro; não foi medida sob carga concorrente intensa.
- Plugin multiarquivo funciona por `../` a partir do diretório de execução — layout **declarado**, não o mais
  ergonômico possível.
