# Estágio 34 — dedupe durável e com teto (E-172)

## 1. Dois defeitos no mesmo lugar

O mapa `eventsByKey` (dedupe por `ExternalMessageID`) tinha **dois** problemas, e o segundo foi descoberto ao
inspecioná-lo para corrigir o primeiro:

| Defeito | Consequência |
| --- | --- |
| **Vivia só em memória** | um reinício **apagava a memória de deduplicação**: a mesma entrega voltava a ser processada — exatamente o que a dedupe existe para impedir |
| **Sem teto e sem poda** | num processo de longa duração o mapa **crescia sem limite**. Vazamento lento é mais difícil de perceber que uma falha, e por isso é pior |

## 2. O que foi feito (mesmo padrão da DLQ, E-170 e E-171)

| Peça | Comportamento |
| --- | --- |
| `OmnichannelDedupeStore` | **um arquivo por organização** (`<dataRoot>/.agent-omnichannel-dedupe/<org>.json`) |
| `OmnichannelDedupeEntry` | registro **mínimo** (id externo, canal, contato, tipo, quando entrou) — não o payload |
| **Retenção** | `OmnichannelDedupeRetention = 7 dias`: a janela de replay de webhook é muito menor, então 7 dias dão margem para entregas atrasadas sem crescer para sempre |
| **Teto** | `OmnichannelMaxDedupePerOrg = 5000`: ao passar do teto, os **mais antigos** saem primeiro |
| `podarDedupe` | aplica retenção **e** teto — em memória e em disco, **na gravação e na carga** |
| estado | `DedupePersisted()` / `DedupeCount(org)` / `LastDedupeError()` / `DedupeRoot()`; sem store, o runtime **declara** que não há persistência |

Ligado no **runtime real** (`NewRuntime`).

## 3. Um defeito pego pelo próprio teste (e a correção)

A primeira versão podava apenas ao **registrar** uma mensagem nova. O teste
`TestOmnichannelDedupeDoesNotHonourExpiredEntries` mostrou a consequência: uma entrada **vencida que já estava no
disco** continuava bloqueando uma mensagem nova, porque era carregada sem poda. Corrigido aplicando `podarDedupe`
**também na carga**.

## 4. Um teste lento que media o caminho errado

O teste do teto fazia **5025 ingestões**, cada uma reescrevendo o arquivo inteiro — **51,9 s** de execução para
provar uma propriedade que se prova melhor **semeando** um índice acima do teto e verificando a poda na carga.
Reescrito: **0,08 s** (≈650× mais rápido), testando exatamente o caminho que importa (retenção + teto, mais
antigos primeiro).

## 5. Testes

| Teste | O que trava |
| --- | --- |
| `…DedupeSurvivesRestart` | entrega repetida é reconhecida **depois do reinício** (e uma mensagem nova continua entrando) |
| `…DedupeRespectsTheCap` | índice semeado acima do teto é **podado**, e os **mais recentes** sobrevivem |
| `…DedupeDoesNotHonourExpiredEntries` | entrada vencida no disco **não** bloqueia mensagem nova |
| `…DedupeWithoutStoreSaysSoInsteadOfPretending` | sem store: `DedupePersisted()==false` |
| `…DedupeReportsUnreadableIndex` | índice ilegível é **reportado** e o evento continua sendo processado |
| `…DedupeFileHasTheKeys` | o arquivo em disco é JSON válido e contém a **chave da mensagem** |

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run TestOmnichannelDedupe -count=1` | **6/6 PASS** |
| `go test ./internal/agent/ ./server/ -run '...'` | **PASS** (bateria da rodada) |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `gofmt -l internal/agent server` | **sem saída** |

## 7. Limites declarados

- Depois de um reinício, a resposta a uma duplicata é o **registro persistido** (id, canal, contato, tipo e
  horário) com `Deduplicated=true` e `DeliveryState="duplicate"` — **não** o evento completo original: o índice é
  de propósito **menor** que o evento, e isso é a troca feita para não reescrever payloads.
- O **teto é por organização e global por processo** (5000 chaves); com muitas organizações o arquivo de cada uma
  é reescrito **a cada mensagem aceita** (custo de escrita por evento, como na janela de rate limit).
- A **retenção é de 7 dias**; entregas repetidas depois disso **não** são reconhecidas como duplicata.
- `eventsByKey` (evento completo, em memória) **não** é limitado: quem serve respostas ricas a duplicatas dentro do
  mesmo processo continua crescendo — o índice persistido é que tem teto. **Limite declarado**, não escondido.
- A poda por teto descarta os **mais antigos**: sob volume muito alto, uma duplicata recente é protegida, uma
  antiga pode cair antes da retenção de 7 dias.
