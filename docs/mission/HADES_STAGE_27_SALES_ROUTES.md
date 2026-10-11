# Estágio 27 — rotas HTTP de vendas e a separação entre erro de entrada e de estado

## 1. O que faltava

Propostas e pós-venda existiam **só no runtime**: capacidade real, mas **não alcançável** pela
API nem pela interface. Sem superfície, a força de vendas não é operável por quem usa o produto.

## 2. Rotas entregues

| Rota | Status |
| --- | --- |
| `POST /sales/proposals` | **201** (preço do catálogo, recalculado no servidor) |
| `GET /sales/proposals` | **200** com filtro `?status=` |
| `GET /sales/proposals/:id` | **200** / **404** |
| `POST /sales/proposals/:id/send` | **200** · **403** sem consentimento · **409** estado inválido |
| `POST /sales/proposals/:id/decide` | **200** · **400** decisão desconhecida · **409** vigência vencida ou estado |
| `POST /sales/follow-ups` | **201** · **400** motivo/escopo ausente |
| `GET /sales/follow-ups` | **200** com `?overdue=true` (o atraso é **consultável**) |
| `POST /sales/follow-ups/:id/complete` | **200** · **404** inexistente · **409** já concluído |

`POST /sales/proposals` **aceita** o campo `unit_price_cents` apenas para **ignorá-lo**: o preço
é sempre o do catálogo. Isso é testado — enviar `1` centavo não muda o total (`3 × 4200 = 12600`).

## 3. Defeito de API encontrado e corrigido

Na primeira versão, `ErrFollowUpInvalid` cobria **duas coisas diferentes**: motivo/escopo
ausentes (**erro de entrada**) e concluir algo já concluído (**conflito de estado**). Isso
devolvia **409** para um corpo inválido. Encontrei porque a asserção do meu próprio teste ficou
**frouxa** ("não pode criar em silêncio") — o que é um cheiro de que o mapeamento estava errado.
Separei em `ErrFollowUpInvalid` (**400**) e `ErrFollowUpStateInvalid` (**409**), e a asserção
passou a ser **exata** (`noReason.Code != 400` falha).

## 4. Verificação

- `go test ./server/ -run TestSalesRoutes` → **PASS**: ciclo completo pela rota (criação com preço
  do catálogo, SKU desconhecido **422**, envio **403** sem consentimento e **200** com consentimento,
  decisão desconhecida **400**, aceite **200**, decisão repetida **409**, proposta inexistente
  **404**, listagem filtrada, pós-venda **201**, motivo vazio **400**, atraso **0** antes do prazo,
  conclusão **200**, conclusão repetida **409**, pós-venda inexistente **404**).
- `go test ./internal/agent/ -run TestFollowUp` → **ok** (domínio ajustado para o erro de estado).
- `go test ./internal/agent/` → **ok** · `go test ./server/` → **ok**.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` → **0 issues**.

## 5. Limites declarados

As rotas **não** entregam a proposta a um provedor externo: registram o envio e aplicam a
política. A entrega real depende de canal configurado (estágio 33, com credencial). Seguem
valendo os limites do estágio: sem desconto negociado, sem imposto/frete e pós-venda é
acompanhamento com prazo, não pesquisa de satisfação.
