# Auditoria dos bloqueios do HADES_MASTER_STATE.json

**Motivo:** a verificação da rodada anterior mediu **13 estágios** com estado
`BLOCKED_BY_EXTERNAL_DEPENDENCY` e **12 deles sem `blocked_reason`** — nenhuma
justificativa, nenhum procedimento de desbloqueio. A regra da missão é explícita:
bloqueio exige **evidência** e **procedimento exato de desbloqueio**.

## 1. O que o estado dizia (medido)

| Estágio | Capacidade declarada | `done` | `blocked_reason` | Evidências |
| --- | --- | --- | --- | --- |
| 15 Memória | IMPLEMENTADO_NAO_HOMOLOGADO | vazio | **vazio** | 4 |
| 23 Company Factory | PARCIAL | vazio | **vazio** | 1 |
| 24 Commerce | AUSENTE | vazio | **vazio** | 1 |
| 25 Marketplace | AUSENTE | vazio | **vazio** | 1 |
| 26 Social | AUSENTE | vazio | **vazio** | 1 |
| 27 Sales | AUSENTE | vazio | **vazio** | 1 |
| 28 Voice | AUSENTE | vazio | **vazio** | 1 |
| 29 Inbox | AUSENTE | vazio | **vazio** | 1 |
| 30 Finance | PARCIAL | vazio | **vazio** | 1 |
| 31 Supervisor | IMPLEMENTADO_NAO_HOMOLOGADO | vazio | **vazio** | 1 |
| 32 E2E negócio | PARCIAL | vazio | **vazio** | 1 |
| 38 Empresa autônoma | AUSENTE | vazio | **vazio** | 2 |

## 2. O que EXISTE de verdade (medido por conteúdo, não por nome de arquivo)

Uma primeira medição por **nome de arquivo** concluiu "AUSENTE" em vários domínios —
e estava **errada**: `company_social.go` não casa com `social*`, `whatsapp_gateway.go`
não casa com os padrões que usei. A medição por **conteúdo** corrigiu o diagnóstico:

| Estágio | Evidência medida | Tamanho |
| --- | --- | --- |
| 15 Memória | `memory_isolation.go` + `memory_retention.go` + 2 testes | 308 + 390 linhas |
| 23 Company Factory | `company.go`, `company_agents.go`, `company_growth.go`, `company_social.go`, **7 arquivos de teste** | 1.760 + testes |
| 26 Social | `company_social.go` (`CreateSocialDraft`) — rascunho, **sem publicação real** | 228 linhas |
| 28 Voice | `tel_agent.go`, `tel_agent_session.go` + **5 arquivos de teste** + rotas em `company_routes.go`/`agent_routes.go` | 412 + testes |
| 31 Supervisor | `supervisor.go` + `supervisor_test.go` | 432 + 493 linhas |
| 24 Commerce | **sem** checkout, carrinho ou pagamento | — |
| 25 Marketplace | **sem** mercado livre/amazon/shopee | — |
| 27 Sales | **sem** CRM, lead ou oportunidade | — |
| 29 Inbox | apenas **comandos** no registry; sem runtime de atendimento | — |
| 30 Finance | **sem** invoice, recebível, pagável ou fluxo de caixa | — |
| 32 E2E negócio | depende de 23–31 | — |
| 38 Empresa autônoma | depende de 23–37 | — |

## 3. O erro de classificação

`BLOCKED_BY_EXTERNAL_DEPENDENCY` foi aplicado em bloco, **confundindo duas coisas**:

1. **o estágio não está implementado** (é `NOT_STARTED`/`AUSENTE`); e
2. **a homologação ao vivo de uma parte exige conta/credencial** (dependência real).

A missão separa os eixos justamente para isso: **estado de execução** e **capacidade**
são um eixo; **bloqueio por requisito** é outro campo. Marcar o estágio inteiro como
bloqueado **escondia trabalho implementável**: memória, supervisor, empresa em sandbox,
CRM/inbox/financeiro/checkout em sandbox **não** dependem de conta externa alguma.

## 4. Reclassificação aplicada (com a evidência acima)

| Estágio | Estado novo | Capacidade nova | Dependência externa real |
| --- | --- | --- | --- |
| 15 Memória | IN_PROGRESS | IMPLEMENTADO_NAO_HOMOLOGADO | **nenhuma** (critério testável localmente) |
| 23 Company Factory | IN_PROGRESS | PARCIAL | registro legal real exige autoridade do usuário |
| 24 Commerce | NOT_STARTED | AUSENTE | meios de pagamento reais (sandbox é local) |
| 25 Marketplace | NOT_STARTED | AUSENTE | conta de vendedor + API oficial |
| 26 Social | IN_PROGRESS | APENAS_ADAPTER | contas/APIs oficiais para publicar |
| 27 Sales | NOT_STARTED | AUSENTE | canais reais (CRM em sandbox é local) |
| 28 Voice | IN_PROGRESS | IMPLEMENTADO_NAO_HOMOLOGADO | provedor de telefonia/número para chamada real |
| 29 Inbox | NOT_STARTED | AUSENTE | canais reais (inbox em sandbox é local) |
| 30 Finance | NOT_STARTED | AUSENTE | integração fiscal autorizada (razão em sandbox é local) |
| 31 Supervisor | IN_PROGRESS | IMPLEMENTADO_NAO_HOMOLOGADO | **nenhuma** |
| 32 E2E negócio | NOT_STARTED | AUSENTE | integrações ao vivo são parte do E2E |
| 38 Empresa autônoma | NOT_STARTED | AUSENTE | integrações ao vivo; sandbox é local |

Cada estágio passou a ter `blocked_reason` **vazio** (não é bloqueio de execução) e um
campo `external_dependency` **separado**, com o desbloqueio escrito — exatamente o que a
missão pede para o caso em que a validação real depende de conta/credencial.

## 5. Consequência para o plano

Isto **reabre trabalho implementável** que estava escondido atrás de um carimbo:
memória (15), supervisor (31) e as partes de sandbox de empresa, comércio, vendas,
inbox e financeiro. A ordem natural continua sendo a das ondas, começando pelo que não
depende de conta externa.
