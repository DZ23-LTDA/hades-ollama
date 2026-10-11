# Estágio 34 — janela de rate limit persistida (E-170)

## 1. Antes de "corrigir", eu conferi — e minha nota estava errada pela metade

A nota de continuidade dizia *"rate-limit/replay em memória"*. Fui verificar e as duas coisas **não** estavam no
mesmo estado:

| Proteção | Estado encontrado |
| --- | --- |
| **Replay de webhook** | **JÁ DURÁVEL** — `WebhookReplayStore` com trava de disco, aluguel de 5 min, retenção de 30 dias e teto de 100 000 claims |
| **Rate limit por organização/canal** | **só em memória** (`r.buckets`) |

Registrar isso importa: eu quase "corrigi" algo que já estava certo. O trabalho real é o **rate limit**.

## 2. O defeito concreto

`r.buckets` vive no processo. **Reiniciar ZERA a cota**: quem estourou o limite e reinicia o serviço — ou quem
explora um reinício — volta a ter a janela cheia. A proteção desaparece **exatamente quando é atacada**.

## 3. O que foi feito (seguindo o padrão que a DLQ já usa)

| Peça | Comportamento |
| --- | --- |
| `OmnichannelRateLimitStore` | **um arquivo por organização** (`<dataRoot>/.agent-omnichannel-rate/<org>.json`): o isolamento entre tenants vale **no disco**, não só no filtro de leitura |
| `writeJSONAtomic` | escrita atômica — e, por ser o caminho único, **respeita a política de durabilidade de E-168/E-169** |
| `restoreRateLimitLocked` | carrega a janela **uma vez** por organização e **descarta janela vencida** (não se herda cota velha) |
| `SetRateLimitStore` | sem store, a janela segue em memória e `RateLimitPersisted()` **diz isso** — não finge durabilidade |
| erro de persistência | **leitura e gravação são relatadas separadamente** e nunca engolidas |

Ligado no **runtime real** (`NewRuntime` → `internal/agent/runtime.go`), não apenas disponível como API.

## 4. Dois defeitos que os próprios testes pegaram

1. **Eu apagava o erro de leitura numa gravação bem-sucedida.** O teste do arquivo corrompido mostrou
   `LastRateLimitError()==""`. Corrigido separando `rateLimitLoadError` e `rateLimitSaveError`: um `save` que dá
   certo **não** apaga a notícia de que a janela anterior não pôde ser lida.
2. **Os CONECTORES também vivem em memória.** O teste de "reinício" falhou com *"connector is not configured"*:
   o segundo runtime não conhece o conector do primeiro. Isso é um **limite declarado NOVO**, encontrado por
   acidente — não foi corrigido nesta rodada e fica registrado em vez de escondido.

## 5. Testes

| Teste | O que trava |
| --- | --- |
| `TestOmnichannelRateLimitWindowSurvivesRestart` | depois de consumir a cota, um **novo runtime** sobre o mesmo store **continua limitando** |
| `TestOmnichannelRateLimitWithoutStoreSaysSoInsteadOfPretending` | sem store: `RateLimitPersisted()==false` (declarado) |
| `TestOmnichannelRateLimitReportsUnreadableWindow` | janela ilegível é **reportada**, e o evento **continua** sendo processado |
| `TestOmnichannelRateLimitDoesNotRestoreExpiredWindow` | janela vencida no disco **não** bloqueia evento novo |
| `TestRuntimeWiresRateLimitPersistence` | o runtime **real** liga a persistência (não é API órfã) |

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run 'TestOmnichannel|TestRuntimeWires' -count=1` | **PASS** (5 novos) |
| `go test ./internal/agent/ ./server/ -run '...'` | **PASS** (bateria da rodada) |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `gofmt -l internal/agent server` | **sem saída** |

## 7. Limites declarados

- **CONECTORES continuam em memória**: um reinício perde a configuração de conector (organização, canal, nome da
  variável de segredo) — a janela de rate limit sobrevive, o **conector não**. Encontrado por teste nesta rodada,
  **não corrigido**.
- **Dedupe do omnichannel** (`eventsByKey`) também está em memória: a deduplicação por `ExternalMessageID` não
  sobrevive a reinício (a proteção de *replay* de webhook é outro caminho e **é** durável).
- A janela é persistida **a cada evento aceito** (custo de escrita por evento; pequena, mas real).
- **Falha de gravação não bloqueia a ingestão**: a janela pode ser perdida no reinício e o erro fica visível em
  `LastRateLimitError()`. Escolha declarada — bloquear todo o tráfego por um disco com problema seria pior que
  relatar a degradação; é um risco residual de proteção contra abuso, **não** uma garantia.
- Sem store (dataRoot vazio), o comportamento é o de antes, **declarado por `RateLimitPersisted()==false`**.
