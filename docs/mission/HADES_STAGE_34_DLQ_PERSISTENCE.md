# Estágio 34 — a fila de mortos deixou de morrer no reinício (E-141)

## 1. O limite que esta rodada fecha

O estado registrava, com estas palavras: *"DLQ em memória (persistência é trabalho futuro)"*.

Fila de mortos em memória tem um defeito específico e **silencioso**: o evento que falhou **desaparece
no reinício**. E o que se perde não é um número — é a mensagem de um cliente que ninguém vai reprocessar,
porque ninguém sabe que ela existiu.

## 2. O que foi entregue

| Peça | O que faz |
| --- | --- |
| `OmnichannelDLQStore` | Persiste **um arquivo por organização** (`<dataRoot>/.agent-omnichannel-dlq/<org>.json`) com escrita atômica |
| Ligação no `Runtime` | `omnichannelRuntime.SetDLQStore(NewOmnichannelDLQStore(omnichannelDLQRoot(dataRoot)))` — sem isso seria código morto |
| Teto **por organização** | `trimDeadLettersLocked` aplica `OmnichannelMaxDLQ` a cada tenant |
| `DiscardDeadLetter` | Remoção **explícita** (decisão registrada) em vez de deixar a entrada apodrecer até o teto expulsá-la |
| Tentativa e motivo persistidos | `RetryDeadLetter` (falha) e `bumpDeadLetter` gravam o histórico do incidente |
| `OmnichannelDLQStatus` | Estado real: `Persisted`, `Storage`, `Entries`, `PerOrganization`, `Error` |
| Rota de DLQ | Passa a devolver `storage` junto da lista, para quem opera ver a durabilidade |

## 3. Um defeito de isolamento que estava escondido no teto

O teto era **global**: `if len(r.deadLetters) > OmnichannelMaxDLQ { … }`. Consequência: o volume de um
tenant **expulsava** as entradas de outro — ou seja, **um tenant conseguia apagar evidência alheia
apenas publicando muito**. Agora o teto é por organização, e há teste que enche `org_a` com
`OmnichannelMaxDLQ + 20` entradas e exige que a única entrada de `org_b` continue lá.

## 4. Quatro regras de honestidade (todas travadas por teste)

| Regra | Por quê |
| --- | --- |
| **Falha de persistência NÃO apaga da memória** | perder em silêncio é exatamente o defeito que esta rodada corrige; o erro aparece em `status.Error` |
| **Sem store, `Persisted=false`** | durabilidade não se presume: uma fila "em memória" que se apresenta como durável é pior que a ausência dela |
| **Arquivo corrompido é REPORTADO, não tratado como fila vazia** | ver §5 — foi o linter que pegou isso no meu próprio código |
| **Isolamento vale no disco** | o arquivo de uma organização não contém entrada de outra (verificado lendo o arquivo, não só a API) |

## 5. O erro que o `nilerr` pegou em mim

Minha primeira versão de `Load` fazia:

```go
if err := readJSON(path, &file); err != nil {
    return nil, nil   // "arquivo ausente é normal"
}
```

Isso trata **qualquer** erro — arquivo corrompido, permissão, disco com problema — como **fila vazia**:
uma fila de mortos corrompida passaria a parecer limpa, apagando a evidência de que existiam mensagens
não processadas. O linter `nilerr` apontou; a correção distingue `errors.Is(err, os.ErrNotExist)` (estado
normal) de todo o resto (**reportado**), e o teste
`TestOmnichannelDeadLetterCorruptedFileIsReportedNotHidden` escreve JSON inválido no arquivo e exige que
o problema apareça no estado em vez de virar silêncio.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run TestOmnichannelDeadLetter` | **8/8 PASS** |
| `go test ./internal/agent/ -count=1` (suíte completa) | **ok 61,813s** |
| `go test ./server/ -count=1` (suíte completa) | **ok 21,294s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY: OK com golang:1.26.9** (agent 17,545s · server 5,464s · exit 0) |

Os oito testes: sobrevive ao reinício · tentativas e motivo persistem · descarte persiste a remoção ·
teto por organização · isolamento no disco · falha de persistência reportada · sem store declara
memória · arquivo corrompido reportado.

## 7. Limites declarados

- **Rate limit continua em memória** (não distribuído entre processos).
- **Retry continua explícito**: não há agendador que reprocesse a fila sozinho — quem opera decide.
- **Sem download/armazenamento de mídia** e **sem rotação de segredo** do conector.
- **Entrega de saída ao provedor** segue dependente de credencial
  (`BLOCKED_BY_EXTERNAL_DEPENDENCY`): a DLQ guarda o que falhou **na entrada**; publicar para fora não é
  o que esta rodada entrega.
