# Estágio 34 — conectores persistidos: o canal sobrevive ao reinício (E-171)

## 1. O limite que um teste encontrou sozinho

Na rodada anterior, o teste de "reinício" da janela de rate limit falhou com *"connector is not configured"*. A
causa não era o rate limit: os **conectores** vivem em `r.connectors` (memória). A consequência é pior que perder
um número:

- **`Ingest` procura o conector por organização e canal** ⇒ depois de um reinício ele não existe mais e **todo
  webhook legítimo do canal é recusado** com "not configured": **o canal para sozinho**;
- um `RegisterConnector` novo gera **ID NOVO** ⇒ assinaturas referentes ao ID antigo deixam de valer e quem já
  estava integrado **quebra sem entender por quê**.

## 2. O que foi feito (mesmo padrão da DLQ e da janela de rate limit)

| Peça | Comportamento |
| --- | --- |
| `OmnichannelConnectorStore` | **um arquivo por organização** (`<dataRoot>/.agent-omnichannel-connectors/<org>.json`) |
| `writeJSONAtomic` | escrita atômica, herdando a política de durabilidade de E-168/E-169 |
| `restoreConnectorsLocked` | carga **uma vez** por organização, usada pelo `connectorFor` (org+canal) |
| `restoreAllConnectorsLocked` / `LoadAll` | carga completa **uma vez**, para as buscas **por ID** (`SignWebhook`), que não sabem a organização antes de carregar |
| `RegisterConnector` | persiste o registro ao cadastrar (sem isso o canal viveria só até o próximo reinício) |
| estado | `ConnectorPersisted()` / `LastConnectorError()` / `ConnectorRoot()`; erro de **leitura e gravação separados** |

Ligado no **runtime real** (`NewRuntime`).

## 3. O segredo NUNCA vai para o disco — e há teste

O arquivo guarda o **NOME da variável de ambiente** (`secret_env`), nunca o valor. `TestOmnichannelConnectorFileNeverStoresTheSecretValue`
define um valor distintivo, cadastra o conector, lê o arquivo e verifica que ele contém o **nome** e **não** contém
o **valor**.

## 4. Um segundo defeito pego pelo teste

A primeira versão restaurava o conector só no `connectorFor` (busca por org+canal). O teste de reinício falhou de
novo: **`SignWebhook` procura por ID** e não passa por ali. Corrigido com `LoadAll` + `restoreAllConnectorsLocked`
(carga completa uma vez), em vez de inventar um índice ID→organização.

## 5. Testes

| Teste | O que trava |
| --- | --- |
| `TestOmnichannelConnectorSurvivesRestart` | runtime novo, **mesmo store**, sem recadastrar: `SignWebhook` e `Ingest` continuam funcionando |
| `TestOmnichannelConnectorFileNeverStoresTheSecretValue` | disco tem o **nome** do segredo e **nunca** o valor |
| `TestOmnichannelConnectorWithoutStoreSaysSoInsteadOfPretending` | sem store: `ConnectorPersisted()==false` e o reinício perde o canal (**declarado**) |
| `TestOmnichannelConnectorReportsUnreadableRegistry` | registro ilegível é **reportado** e o canal **falha fechado** |
| `TestRuntimeWiresConnectorPersistence` | o runtime real usa a raiz de conectores |

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run 'TestOmnichannelConnector|TestRuntimeWiresConnector' -count=1` | **5/5 PASS** |
| `go test ./internal/agent/ ./server/ -run '...'` | **PASS** (bateria da rodada) |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `gofmt -l internal/agent server` | **sem saída** |

## 7. Limites declarados

- **A dedupe do omnichannel** (`eventsByKey`, por `ExternalMessageID`) **continua em memória**: depois de um
  reinício, uma entrega repetida do mesmo `external_message_id` pode ser processada de novo. A proteção de
  *replay* de webhook (`WebhookReplayStore`) é outro caminho e **é** durável — mas não cobre este.
- A **carga completa** (`LoadAll`) lê todos os arquivos de organização uma vez por processo; com muitas
  organizações o custo aparece na primeira busca por ID.
- A persistência é **por cadastro**, não por alteração em massa; não há compactação/rotação do arquivo.
- **Sem store** (dataRoot vazio) o comportamento é o de antes, declarado por `ConnectorPersisted()==false`.
- `Enabled=false` continua sendo respeitado na busca por org+canal (conector desativado não ingere), mas a busca
  **por ID** (`SignWebhook`) não filtra `Enabled` — comportamento anterior, mantido e **declarado**.
