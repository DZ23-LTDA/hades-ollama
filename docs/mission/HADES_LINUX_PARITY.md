# Paridade Linux local: validar onde o GATE roda (E-129)

## 1. A lacuna que este script fecha

Na rodada 86 o CI recusou um PR com uma falha que **passava no Windows**:

```
store_atomic_test.go:54: failed new mission remained in memory:
lstat /tmp/.../not-a-directory/store: not a directory
```

Ler um caminho sob um **arquivo** devolve **ENOTDIR** no Linux e "não encontrado" no Windows. A
suíte inteira havia passado na máquina de desenvolvimento e falhou no **gate**. A lição registrada
foi: *injeção de falha trocada precisa ser validada no sistema onde o gate roda* — mas essa lição só
tem valor se houver um **meio de fazer isso localmente**. Este é o meio.

## 2. O que foi entregue

`scripts/verify-linux-parity.ps1` roda `go vet` + `go test` dos pacotes do gate **dentro de um
contêiner Linux**, montando:

| Montagem | Para quê |
| --- | --- |
| repositório em `/src` | o código real, sem cópia |
| `_tmp/gomodcache` em `/gomodcache` | reaproveitar as dependências já baixadas, sem rebaixar |
| `GOCACHE=/tmp/gocache` e `TMPDIR=/tmp` | cache e temporários dentro do contêiner |

Comportamento deliberado:

- **recusa rodar sem daemon**: um contêiner que falha por Docker ausente seria um falso "a suíte
  passou" — o script falha alto com instrução;
- **tenta imagens em ordem** (`golang:1.26.9`, `golang:1.26`, `golang:latest`) e distingue
  **imagem indisponível** de **falha de teste**: no primeiro caso tenta a próxima, no segundo
  devolve o código de saída do Linux, dizendo explicitamente que as mensagens são do Linux;
- **nunca publica** nada e não contata serviço externo.

## 3. Resultado medido (nesta máquina)

```
Linux parity: imagens=golang:1.26.9, golang:1.26, golang:latest
Linux parity: comando=go vet ./internal/agent/ ./server/ && go test ./internal/agent/ ./server/ -count=1 -timeout 30m

ok  github.com/ollama/ollama/internal/agent  18.071s
ok  github.com/ollama/ollama/server           5.963s

LINUX PARITY: OK com golang:1.26.9 (exit 0)
```

Imagem `golang:1.26.9`, digest `sha256:d7722066f0b60ceccb6c0643cbed1f5f9e15506ac237146c95333504d7805d89`.

Dois ganhos além da paridade:

1. **Cobertura maior no Linux**: testes que **pulam** no Windows por falta de symlink (por exemplo
   `TestBuildArtifactManifestRejectsSymlinkOutsideWorkspace` e `TestBuilderPreviewRejectsSymlinkEntry`)
   **executam** no contêiner. Rodar só no Windows deixa esses caminhos sem verificação;
2. **Ciclo mais rápido**: 24 s no Linux contra ~78 s no Windows para os mesmos dois pacotes, o que
   torna viável rodar a validação a cada mudança de risco.

## 4. Como usar

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-linux-parity.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-linux-parity.ps1 -TestFilter TestJSONStoreRollsBack
```

## 5. Limites declarados

- **Não substitui o CI**: o contêiner é Go puro; o job do CI instala navegador Playwright, serviços
  de apoio e roda passos próprios. O script cobre o que causou a regressão da rodada 86, não o
  pipeline inteiro;
- **Não cobre macOS**, que continua sem medição neste host;
- **Requer Docker Desktop ativo**; sem daemon o script falha alto e não finge sucesso;
- **Não roda a suíte de integração** (que exige PostgreSQL/Redis/OTLP): para isso existe o caminho
  do compose já registrado nas evidências E-122 e E-123.

## 6. Erros meus, corrigidos durante a construção

O script só passou a funcionar depois de três defeitos **meus**, todos encontrados por executá-lo:

1. `"$repositoryRoot:"` foi interpretado como **unidade de variável** (`$var:`) — corrigido com
   `${repositoryRoot}`;
2. acentos em um `.ps1` quebram o parser no Windows PowerShell 5.1, que lê o arquivo como ANSI —
   o script passou a ser **sem acentos**, com o motivo registrado no cabeçalho;
3. `$ErrorActionPreference = 'Stop'` transformava a mensagem normal do docker em STDERR
   (`Unable to find image ...`) em erro **terminante**, abortando antes de tentar a próxima imagem —
   corrigido para `Continue`, com os erros tratados por `throw`/`exit` explícitos.
