# Estágio 8 — plugin com código REAL de terceiro: verificação feita e **defeito encontrado na execução** (E-183)

## 1. O que eu fui fazer e o que encontrei

O item *'Plugins têm instalação e execução verificáveis'* estava `NAO_COMPROVADO` com a lacuna declarada: *'há
contrato e testes herméticos; falta executar um plugin REAL de terceiro'*. Fui executar um plugin cujo conteúdo é
**código publicado por terceiros de verdade** — o pacote `ms@2.1.3` (**MIT**), copiado **byte a byte** do
vendorizado (`internal/agent/testdata/thirdparty/ms/`, com a licença junto), com o sha256 do arquivo conferido
antes de executar (`e5f0b6a9…`).

Resultado: **a instalação, a assinatura, o hash de conteúdo e a recusa de adulteração passaram** — e **a execução
está quebrada**. Três achados reais, todos medidos:

| # | Achado | Evidência |
| --- | --- | --- |
| 1 | **A execução de plugin é Linux-only, por design** | `sandboxExecTool.Execute`: `runtime.GOOS != linux` → *sandbox.exec requires the Linux namespace executor; host-process fallback is disabled* |
| 2 | **O sandbox `strict` exige pré-requisito de operador** | no container com Node: *strict sandbox unavailable (NOT_CONFIGURED): strict sandbox requires `OLLAMA_AGENT_SANDBOX_CGROUP_ROOT`* |
| 3 | **DEFEITO: a execução falha no layout real de instalação** | modo `best-effort` explicitamente aprovado: *plugin sandbox execution failed: **open …/plugins/content/org-runtime/runtime-plugin-1.0.0.zip: not a directory*** |

## 2. A causa do defeito (localizada, não suposta)

`internal/agent/plugin_runtime.go`:

- a **linha 198** lê o entrypoint **do zip** corretamente (`readPluginEntrypoint` usa `zip.OpenReader` e valida
  caminhos suspeitos) — a leitura está certa;
- a **linha 215** passa `Workspace: installation.ContentPath`, e `ContentPath` é **o arquivo `.zip`**
  (`plugins/content/<org>/<id>-<versão>.zip`, por `plugin_bundle.go:113`). O executor trata `Workspace` como
  **diretório** e falha com `not a directory`.

Ou seja: **instalar funciona, verificar funciona, executar não funciona** — e o caminho de sucesso provavelmente
nunca foi exercitado, porque os testes existentes cobrem majoritariamente **recusas** (desabilitado, não confiável,
adulterado, escopo negado, entrypoint inválido). Isso é registrado como **B-13** em `HADES_BLOCKERS.md`.

## 3. Segundo limite, este CONFIRMADO por execução

O sandbox recebe **somente o conteúdo do entrypoint** (`readPluginEntrypoint` → o campo `code` do input). Um plugin
de **vários arquivos** instala (o bundle tem `lib/ajudante.js`) e **não consegue carregar o irmão**:
`TestPluginRuntimeShipsOnlyTheEntrypointToTheSandbox` **passou no Linux**, confirmando que o irmão presente no
bundle não é alcançável. Consequência prática: plugins multiarquivo não funcionam, e foi por isso que o teste do
terceiro **inlina** a biblioteca no entrypoint.

## 4. O que foi efetivamente verificado (e é evidência válida)

| Verificação | Onde | Resultado |
| --- | --- | --- |
| instalação por assinatura ed25519 + promoção a confiável | Windows | **PASS** |
| verificação de conteúdo por digest (`VerifyInstalledContent`) | Windows | **PASS** |
| o conteúdo instalado **começa pelo código de terceiro byte a byte** | Windows | **PASS** |
| licença (MIT) e homepage do terceiro no manifesto, e a licença dentro do bundle | Windows | **PASS** |
| adulteração do terceiro **depois** da instalação ⇒ recusada | Windows | **PASS** |
| execução de plugin | Windows | **SKIP** declarado (Linux-only) |
| execução de plugin | Linux + Node | **FALHA — defeito B-13** |
| plugin multiarquivo não alcança o irmão | Linux + Node | **PASS** (limite confirmado) |

## 5. Consequência para a matriz

O item **continua `NAO_COMPROVADO`** — e agora com motivo **melhor**: não é 'falta executar um plugin de
terceiro'; é **'a execução está quebrada no layout real (B-13) e o sandbox é Linux-only com pré-requisito de
operador'**. Promover este item a `DONE` seria exatamente o tipo de mentira que o gate existe para impedir.

## 6. Limites declarados

- O código de terceiro executado é **genuíno** (pacote publicado, byte a byte, licença preservada), mas está
  **inlinado no entrypoint** porque o runtime não entrega os irmãos do bundle — o teste **não** prova um plugin
  de terceiro com múltiplos arquivos.
- **Não** existe plugin publicado por terceiros com autor/canal próprios: não há marketplace no projeto. Essa
  parte continua dependência externa declarada.
- A execução foi tentada no modo **`best-effort`** (isolamento mais fraco, permitido apenas com aviso de aprovação
  explícito); o modo **`strict`** não pôde ser exercitado por falta de `OLLAMA_AGENT_SANDBOX_CGROUP_ROOT`.
- O defeito B-13 foi **identificado e localizado**, **não corrigido** nesta rodada.


## 7. Um terceiro achado: o teste quebrava no CI por FIM DE LINHA (e o job certo pegou)

O PR desta rodada **falhou no CI**, no job **"Go agentic Windows portability"** — e o motivo era meu, não do
produto: o checkout no Windows converte **LF → CRLF**, então o **sha256** que eu confiro contra o arquivo publicado
**deixava de bater**. O job existe exatamente para pegar isso, e pegou.

Correção em duas pontas: `.gitattributes` marca `internal/agent/testdata/thirdparty/**` como `-text` (preserva o
byte publicado) **e** o teste normaliza `\r\n` → `\n` antes de hashear — um teste que depende da configuração de
git de quem roda não prova o byte publicado, prova a configuração.


## 8. Terceiro pré-requisito ambiental (medido no CI, job `test (ubuntu-latest)`)

Com `node` presente no runner, o sandbox respondeu **`sandbox interpreter unavailable for node`** — ou seja, além de
ser Linux-only e de o `strict` exigir `CGROUP_ROOT`, o sandbox **só executa interpretadores de uma lista
permitida**, e o `node` do runner não está nela. Consequência para o teste: ele **pula** com o motivo em vez de
falhar, e só falha se a execução passar (armadilha que obriga a fechar B-13) ou se o erro for **inesperado**.
