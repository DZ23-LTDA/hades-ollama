# Estágio 25 — marketplace: contrato de sincronização contra DUBLE de sandbox (E-126)

## 1. O que o estágio pedia

*"Conectores oficiais operando em sandbox com evidências"* — e a homologação live declarada
bloqueada por fornecedor. O que foi entregue é **exatamente isso, e nada além**: um **contrato** de
sincronização (produtos, preços, estoque, pedidos) exercitado contra um **duble de sandbox** em
processo. **Nenhuma** chamada a Mercado Livre, Amazon, Shopee ou qualquer plataforma acontece.

## 2. Decisões deliberadas

| Decisão | Por quê |
| --- | --- |
| **CATALOGAR NÃO É HABILITAR** (§11.3) | canal sem credencial é `CONFIG_REQUIRED` com procedimento; com credencial mas sem handshake é `UNAUTHORIZED` e `LiveTested=false`. `CONNECTED` só vale para o **duble** |
| **Teto de operações por execução** | um conector que "sincroniza tudo" sem teto é a forma mais rápida de violar rate limit; a violação é recusada **antes** de aplicar qualquer efeito |
| **Idempotência por versão** | reexecutar a mesma sincronização não reaplica; o item aparece como **ignorado**, não como sucesso novo |
| **Conflito não é sobrescrita silenciosa** | quando o estoque remoto diverge do local, o item vira **conflito** e **não** é aplicado — sobrescrever esconderia divergência, que é o erro que produz venda sem produto |
| **Sem publicação externa** | o relatório traz `homologacao_live: BLOCKED_BY_EXTERNAL_DEPENDENCY` com motivo e procedimento |

## 3. Verificação (4 testes, nesta máquina)

| Teste | O que prova |
| --- | --- |
| `TestMarketplaceChannelCatalogDeclaresRealState` | todo canal sem credencial é `CONFIG_REQUIRED` com procedimento e **sem** teste live; credencial presente vira `UNAUTHORIZED`, **não** `CONNECTED` |
| `TestMarketplaceSyncAppliesIsIdempotentAndDeclaresSandbox` | aplica 2 itens; o relatório declara **sandbox** e homologação **bloqueada**; o duble recebe o estoque; reexecutar a **mesma versão** só ignora (`0` aplicados, `2` ignorados); **versão nova** aplica |
| `TestMarketplaceSyncRefusesRateLimitAndInvalidInput` | acima do teto é **recusado antes de qualquer efeito** (nada foi aplicado, comprovado); canal desconhecido, pedido vazio, direção inválida, item sem SKU e escopo ausente são erros declarados |
| `TestMarketplaceSyncReportsConflictInsteadOfOverwriting` | com remoto `10` e desejado `5`, o item vira **conflito** com detalhe, `0` aplicados, e o remoto **não é sobrescrito** |

## 4. Limites declarados

- **Homologação live**: `BLOCKED_BY_EXTERNAL_DEPENDENCY` — exige conta autorizada, aplicação
  registrada e scopes válidos; **procedimento**: definir a variável de credencial do canal e
  executar teste de conexão real com evidência de handshake.
- **Sincronização de pedidos é declarada, não exercitada**: o duble cobre produtos, preços e
  estoque; `pedidos` está no catálogo de capacidades e no contrato, mas o pull de pedidos depende da
  API real.
- **Sem rate limit negociado por canal**: o teto é global por execução, não o limite real de cada
  plataforma.
- **Sem interface visual e sem rotas HTTP** para o marketplace: a capacidade está no runtime e
  testada.
