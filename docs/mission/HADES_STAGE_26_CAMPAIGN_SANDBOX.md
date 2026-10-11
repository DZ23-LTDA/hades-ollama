# Estágio 26 — campanha em SANDBOX: aprovação, agendamento e métricas

## 1. O que faltava

A campanha já nascia com `Mode = "sandbox"`, exigia aprovação e **recusava** lançamento fora do
sandbox. Os campos de métrica (impressões, cliques, conversões, gasto) já existiam — mas **não
havia agendamento nem forma de registrar métrica**: o painel mostrava zeros para sempre e
"métricas" era só intenção. Era exatamente o `APENAS_ADAPTER` do estágio.

## 2. Decisões deliberadas

| Decisão | Por quê |
| --- | --- |
| **AGENDAR EXIGE APROVAÇÃO** | sem isso, o agendamento viraria um **caminho paralelo** para colocar campanha em execução sem passar pela aprovação humana |
| **INÍCIO NO FUTURO** | agendar no passado significaria "já rodou", e nada rodou |
| **MÉTRICA SÓ EM CAMPANHA QUE EXECUTOU** | medir `draft` ou apenas aprovada seria **inventar resultado** de algo que não executou |
| **MÉTRICA SÓ SOMA** | contador negativo é recusado: permitir subtração **apagaria histórico de desempenho**, que é o dado a preservar |
| **ORÇAMENTO LIMITA** | o incremento de gasto não pode passar do orçamento diário; orçamento que não limita não é orçamento |
| **NUNCA PUBLICA** | nada aqui chama API de plataforma: o resultado traz `PublicadoExternamente: false` e o relatório declara `BLOCKED_BY_EXTERNAL_DEPENDENCY` com o motivo — ninguém confunde simulação com campanha no ar |
| **CTR SEM DIVISÃO POR ZERO** | com zero impressões a taxa é `0.00%`, não `NaN` |

## 3. Verificação (3 testes, nesta máquina)

| Teste | O que prova |
| --- | --- |
| `TestScheduleCampaignRequiresApprovalAndFutureStart` | agendar **sem** aprovação é recusado; início no **passado** é recusado; após aprovar, o estado vira `scheduled` com a **data registrada**; campanha inexistente é erro declarado |
| `TestCampaignMetricsOnlyAfterExecutionAndOnlyAdds` | medir **rascunho** e **apenas aprovada** é recusado; contador **negativo** é recusado; **estourar o orçamento** é recusado; o registro válido acumula `1000/50/5/3000` e calcula `5.00%` e `10.00%`; o segundo registro **SOMA** (`1500/75/5000`); o resultado **não** declara publicação externa |
| `TestCampaignReportDeclaresSandboxAndNoExternalPublication` | painel sem campanha tem CTR `0.00%`; o relatório declara `sandbox`, publicação externa **bloqueada** e o motivo; com uma campanha medida, os totais do painel fecham (`200/20` e conversão `20.00%`) |

## 4. Limites declarados

- **PUBLICAÇÃO EXTERNA**: `BLOCKED_BY_EXTERNAL_DEPENDENCY` — exige conta autorizada e scopes
  válidos da plataforma; sem credencial, não há campanha no ar e nada aqui finge que há.
- **Sem ingestão automática de métrica**: o registro é explícito (operador/importação), porque
  buscar desempenho real exigiria a API da plataforma.
- **Sem segmentação, criativo ou leilão**: o escopo entregue é aprovação → agendamento → medição.
- **Rotas HTTP ainda não expostas** para agendamento/métricas: a capacidade está no runtime e
  testada; a superfície fica para a próxima rodada (declarado, não escondido).

## 5. Verificação final

- `go test ./internal/agent/ -run 'TestScheduleCampaign|TestCampaignMetrics|TestCampaignReport'` →
  **3/3 PASS**.
- `go test ./internal/agent/` → **ok** · `go test ./server/` → **ok**.
- `go build ./...` exit 0 · `golangci-lint` → **0 issues** (falso positivo do `misspell` em
  "decorativo"/"denominador" foi **reescrito**, não silenciado).
