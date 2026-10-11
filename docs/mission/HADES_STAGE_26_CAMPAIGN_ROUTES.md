# Estágio 26 — superfície HTTP de campanha

## 1. O limite que eu havia declarado

Na rodada anterior eu registrei, no próprio documento do estágio: *"**Rotas HTTP ainda não
expostas** para agendamento/métricas: a capacidade está no runtime e testada; a superfície fica
para a próxima rodada (declarado, não escondido)"*. Esta rodada fecha exatamente esse item.

## 2. Rotas entregues

| Rota | Status |
| --- | --- |
| `POST /companies/:id/campaigns/:campaign_id/schedule` | **200** · **400** data inválida · **409** sem aprovação ou estado |
| `POST /companies/:id/campaigns/:campaign_id/metrics` | **200** · **400** contador negativo · **409** sem execução ou orçamento estourado |
| `GET /companies/:id/campaigns/report` | **200** · **404** empresa inexistente |

O agendamento aceita `start_at` em **RFC 3339** ou `start_in_hours`; as duas formas passam pelo
**mesmo domínio**, que exige aprovação e início no futuro — a rota não cria caminho alternativo
para executar campanha sem aprovação.

## 3. Verificação (1 teste de rota, cobrindo cada status)

`TestCampaignRoutesScheduleMetricsAndReport` **PASS**: criação **201**; agendar sem aprovação
**409**; `start_at` inválido **400**; métrica antes de executar **409**; agendamento aprovado
**200** com estado `scheduled` e data registrada; orçamento estourado **409**; contador negativo
**400**; métrica válida **200** com `10.00%` e `20.00%` e **sem** publicação externa; painel **200**
declarando **sandbox**, publicação externa **bloqueada** e motivo, com gasto total `700`; empresa
inexistente **404**.

## 4. Limites declarados

- **Sem publicação externa**: `BLOCKED_BY_EXTERNAL_DEPENDENCY` — exige conta autorizada e scopes
  válidos da plataforma.
- **Sem ingestão automática de métrica**: o registro é explícito (operador/importação).
- **Sem segmentação, criativo ou leilão**: o escopo é aprovação → agendamento → medição.
- **Sem interface visual de campanha**: a superfície entregue é a HTTP; a tela fica para a frente
  de UX (estágio 19).

## 5. Verificação final

- `go test ./server/ -run TestCampaignRoutes` → **PASS**.
- `go test ./internal/agent/` → **ok** · `go test ./server/` → **ok**.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` → **0 issues**.
