# Estágio 34 — detector de corrida na suíte inteira (E-177)

## 1. O que E-176 deixou em aberto

E-176 trouxe o `-race` para o CI, mas com um escopo **estreito e declarado**: *"o `-race` cobre **apenas**
`TestOmnichannel*`"*. A justificativa era custo. O problema dessa escolha é simples: **corrida não aparece só onde
se procura** — ela aparece onde há concorrência, e o runtime de agentes tem várias (memória, store, re-embed,
supervisor, health).

## 2. O que foi medido

Execução do detector na **suíte inteira** (`./internal/agent/ ./server/`), container `golang:1.26.9` com
`CGO_ENABLED=1`:

| Pacote | Tempo sob `-race` | Resultado |
| --- | --- | --- |
| `internal/agent` | **80,5 s** | `ok` |
| `server` | **11,0 s** | `ok` |
| `DATA RACE` | — | **nenhuma** |

Ou seja: a suíte **inteira** é limpa sob o detector, ao custo de ~91 s. O escopo estreito de E-176 não se
justificava por preço.

## 3. O que mudou

`scripts/verify-linux-parity.ps1`:

| Antes (E-176) | Agora (E-177) |
| --- | --- |
| `RacePackages = ./internal/agent/` | `RacePackages = ./internal/agent/ ./server/` |
| `RaceFilter = TestOmnichannel` | `RaceFilter = ''` ⇒ **suíte inteira** |
| ~5 s de detector | ~91 s de detector |

O parâmetro `-SkipRace` continua existindo (para quando o motivo for declarado), e `-RaceFilter` permite estreitar
o escopo quando alguém quiser investigar um teste específico.

## 4. Verificação

| Comando | Resultado |
| --- | --- |
| `docker run … CGO_ENABLED=1 golang:1.26.9 go test -race ./internal/agent/ ./server/ -count=1` | **ok** (agent 80,5 s · server 11,0 s) · **sem DATA RACE** |
| `powershell -File scripts/verify-linux-parity.ps1` (gate completo, com o novo passo) | **LINUX PARITY OK** — ver evidência |
| `golangci-lint` / contrato / bateria da rodada | ver evidência |

## 5. Limites declarados

- `-race` **não** é prova de ausência de corrida: é detecção **dinâmica**, só encontra o que o teste executado
  realmente exercita. Corrida em caminho não coberto por teste continua invisível.
- O detector **não** foi executado em `-count` alto: uma corrida rara pode passar numa única execução. `-count=1`
  é o que o gate faz.
- O custo real do gate subiu: o passo de paridade Linux agora leva ~91 s a mais (medido no container, host
  Windows). Um gate mais lento e mais honesto é a troca feita aqui — e ela fica **dita**.
- Os pacotes cobertos são `./internal/agent/` e `./server/`; o restante do repositório (fora do escopo do gate
  atual) continua sem detector.
