# Gate 22 — instalação, atualização, rollback e desinstalação executados

## 1. O que estava pendente

O gate intermediário registrava **"instalação/upgrade/rollback de pacote executados de
verdade: NÃO MEDIDO"**. O script `scripts/install.ps1` **compilava e copiava** o binário,
mas não tinha **backup**, **rollback** nem **desinstalação** — ou seja, "atualizar" era uma
operação **sem volta**. Isso é exatamente o que o estágio proíbe.

## 2. O que foi implementado

`scripts/install.ps1` agora entrega os quatro caminhos, com manifesto auditável:

| Caminho | Comportamento |
| --- | --- |
| **Instalação** | compila do código (`go build -trimpath`, com `-Ldflags` opcional para release) e grava `install-manifest.json` com **ação, commit, data e SHA-256** |
| **Atualização** | detecta binário existente, **preserva o anterior** em `<binário>.previous` e registra o hash anterior no manifesto |
| **Rollback** | restaura o binário anterior e **preserva o que estava instalado** em `<binário>.rolledback` — então o rollback é **reversível** |
| **Desinstalação** | remove binário, backup e manifesto e **preserva os dados do usuário** (modelos), porque apagar modelos seria **perda de dados**, não limpeza |

## 3. O ciclo, executado de verdade (transcrito)

Diretório descartável (`_tmp/install-cycle`, fora do Git), binário construído do código:

| Passo | Observado |
| --- | --- |
| **1. Instalação limpa** | `INSTALL concluído`; SHA-256 `ADCA65F8…ACFAD6C`; manifesto `action=INSTALL`, `commit=e443e9b19`; backup **não** existe |
| **2. Execução do binário instalado** | `ollama version is 0.40.1` — o binário **roda**, não é só um arquivo |
| **3. Atualização** (com `-Ldflags -X main.Version=9.9.9-upgrade`) | `Atualização detectada: binário anterior preservado`; SHA-256 **`D27C2249…5E1052`**, **diferente** do instalado; manifesto `action=UPGRADE` com `previous` = hash anterior |
| **4. Rollback** | `Rollback concluído`; SHA-256 restaurado `ADCA65F8…ACFAD6C`; **restaurado == instalação original: True**; o binário da atualização ficou preservado em `.rolledback` |
| **5. Desinstalação** | binário, backup e manifesto **removidos**; dados do usuário **preservados** |

**Nota de honestidade:** o `-Ldflags` mudou o **hash** do binário (a prova de que a
atualização entregou um artefato **diferente**), mas a saída de `--version` continuou
mostrando `client version is 0.0.0` nas duas builds — a variável injetada **não** é a que
essa mensagem imprime. A prova de conteúdo diferente é o **hash**, e é assim que está
registrado; não afirmo que a string de versão mudou.

## 4. O que continua NÃO medido (e por quê)

| Pendência | Situação |
| --- | --- |
| Instaladores **de pacote** (`.deb`/`.rpm`/`.dmg`/`.msi`) e **assinatura digital** | **NÃO MEDIDO**: este ambiente não produz pacote assinado, e anunciar homologação sem artefato seria invenção |
| Instalação em **macOS** e **Linux** reais | **NÃO MEDIDO** neste host (Windows); o CI cobre *build*, não instalação |
| **Desktop/computer-use nativo** | **AUSENTE** |
| **APIs externas com credenciais reais** | **BLOCKED_BY_EXTERNAL_DEPENDENCY** |

O gate intermediário continua **NAO_APROVADO**, agora por **um** motivo a menos.

## 5. Verificação

- **Guarda de contrato**: `go test . -run TestInstallScript` **PASS** — impede que uma
  edição futura remova rollback, backup, manifesto, hash ou a declaração de que a
  desinstalação **preserva** os dados do usuário.
- O ciclo acima foi **executado** neste host (Windows), com o binário rodando
  (`ollama version is 0.40.1`).
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** nos pacotes
  (`./internal/agent/...`, `./server/...` e o pacote raiz).
- Observação: `_tmp/` contém rascunhos `.go` em UTF-16 **de outra sessão** (ignorados pelo
  Git); foram **preservados** de propósito, e por isso a verificação de lint é feita nos
  pacotes do produto.
