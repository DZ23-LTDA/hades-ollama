# Estágio 32 — BI e status de negócio, com proveniência e lacunas declaradas (E-132)

## 1. O vazio que esta rodada fecha

O estágio 32 estava `NOT_STARTED` / `AUSENTE`: **"BI / business status e métricas"**. É o capstone
natural sobre a cadeia empresarial construída nas rodadas anteriores (empresa, comércio, campanha,
vendas, atendimento, marketplace) — e, importante, é implementável **sem credencial externa**,
porque agrega o que os módulos internos já registram.

## 2. O que foi entregue

`Runtime.BusinessStatus(organizationID, now)` e a rota `GET /companies/:id/business-status`:

| Bloco | Origem (proveniência declarada na resposta) |
| --- | --- |
| `commerce` | `CommerceStore` (produtos, pedidos, receita, estoque baixo) |
| `finance` | `CompanyStore.ListLedgerEntries` (receita, despesa, líquido, conciliado) |
| `campaigns` | `CompanyStore.CampaignReport` (estados, impressões, cliques, conversões, gasto) |
| `sales` | `SalesWorkforceStore` (propostas por estado, follow-ups, vencidos) |
| `inbox` | `InboxStore.ListConversations` (conversas, abertas, violações de SLA) |
| `marketplace` | `MarketplaceChannels` (quantos canais **não** estão conectados) |

## 3. Três decisões deliberadas

1. **CADA NÚMERO TEM PROVENIÊNCIA.** A resposta traz `sources` ligando cada bloco ao store de
   origem. Um painel sem origem não pode ser auditado — e um número sem origem, num produto
   enterprise, é indistinguível de um número inventado.
2. **O QUE NÃO É MEDIDO É DECLARADO.** `nao_medido` lista, **com motivo**: faturamento fiscal,
   saldo/conciliação bancária, custo de tokens/modelo, publicação em marketplace e redes sociais,
   telefonia e voz, e preço negociado. A razão é direta: **um painel que mostra zero onde não há
   medição é pior que um painel vazio, porque zero parece resposta.**
3. **ESCOPO VEM DA SESSÃO.** A rota usa `companyForRequest`; o `organization_id` **nunca** vem do
   corpo ou da query. Um painel é exatamente o lugar onde um parâmetro de organização vindo do
   cliente se tornaria vazamento entre organizações.

## 4. Verificação (5 testes, todos PASS)

| Teste | O que prova |
| --- | --- |
| `TestBusinessStatusAggregatesRealStores` | os números vêm dos stores reais: `2` produtos, `3` pedidos, `2` pagos, `1` cancelado, receita `3500`; livro com `2500` de receita, `1000` de despesa, líquido `1500`, `2500` conciliado e `1` lançamento em aberto; campanha `scheduled` com `300/30/3` e gasto `1500`; `1` proposta e `1` follow-up **vencido**; `1` conversa com `1` violação de SLA; **proveniência** de 5+ blocos |
| `TestBusinessStatusDeclaresWhatIsNotMeasured` | as lacunas citam **fiscal, bancária, tokens, marketplace e telefonia**, e cada item traz **motivo** (contém `:`) |
| `TestBusinessStatusIsScopedByOrganization` | a organização `org_b` recebe **zeros** em comércio, financeiro, campanhas, vendas e atendimento, e os números da organização dona **não mudam** |
| `TestBusinessStatusOnEmptyOrganizationStillDeclaresGaps` | mesmo vazio, o painel mantém origem e lacunas declaradas |
| `TestBusinessStatusRouteReportsIndicatorsAndGaps` | a rota devolve **200** com escopo vindo da sessão, proveniência e lacunas; empresa inexistente é **404** |

Um detalhe de honestidade do próprio teste: eu havia **assumido** que 9 unidades de estoque
produziriam "estoque baixo" — o teste falhou e o código estava certo (o limiar é `disponível <= 0`).
A fixture passou a **produzir de fato** o SKU zerado (`BI-LOW`) e as asserções foram corrigidas para
o valor real, em vez de afrouxar a verificação.

## 5. Verificação da cadeia

- `go test ./internal/agent/ -run TestBusinessStatus` → **4/4 PASS**.
- `go test ./server/ -run TestBusinessStatusRoute` → **PASS**.
- `golangci-lint` → **0 issues**.
- `scripts/verify-linux-parity.ps1` → ver resultado registrado na evidência (gate em Linux).

## 6. Limites declarados

- **Não mede o que depende de fora**: fiscal, bancário, custo de tokens, publicação externa e
  telefonia seguem em `nao_medido` — `BLOCKED_BY_EXTERNAL_DEPENDENCY` onde aplicável.
- **É uma fotografia, não série temporal**: não há histórico, tendência nem comparação entre
  períodos; cada chamada calcula o estado atual.
- **Sem tela dedicada**: a superfície entregue é a HTTP. Ligar o painel na interface é trabalho da
  frente de UX.
- **Sem exportação** (CSV/PDF) e sem agendamento de relatório.
