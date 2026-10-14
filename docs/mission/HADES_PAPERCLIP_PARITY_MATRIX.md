# HADES × PAPERCLIP — matriz de paridade (lado Hades, medido)

Complemento: **MASTER V2.2 — Hades Agent Organization OS (Paperclip Parity)**.
Regra de trabalho desta matriz: **classificar só o que foi medido no código**, com
`ficheiro:linha` como evidência. *«Não encontrei pelo nome» não é «não existe»* — a diferença
custou o E-619 (uma ferramenta declarada em falta que já existia como `knowledge.search`).

## 1. Matriz (20 capacidades)

| # | Capacidade (Paperclip) | Hades | Evidência medida |
| --- | --- | --- | --- |
| 1 | Agentic Task Manager | `IMPLEMENTED_VERIFIED` | missões + passos + runtime de execução |
| 2 | Organograma e hierarquia | `IMPLEMENTED_VERIFIED` | `internal/agent/company_agents.go` — 7 agentes com departamento, supervisor, SLA, `Autonomy`, `ApprovalRequired`, `MemoryScope` |
| 3 | Goal Alignment | `IMPLEMENTED_VERIFIED` | `internal/agent/company.go:56` — `type CompanyGoal struct` |
| 4 | Heartbeats e rotinas | `IMPLEMENTED_PARTIAL` | `business_supervisor_scheduler.go:33` — `SupervisorScheduler` dispara o ciclo supervisionado **no intervalo configurado**. É ciclo de supervisor, **não** heartbeat durável por agente |
| 5 | Atomic Task Checkout | `IMPLEMENTED_VERIFIED` | fila distribuída com *lease* — testes de *claim* idempotente e de *reclaim* expirado |
| 6 | Governança e aprovações | `IMPLEMENTED_VERIFIED` | `service_agent_approvals.go` (37,4 KB) + quórum + cadeia de auditoria |
| 7 | Controle de orçamento | `IMPLEMENTED_VERIFIED` | `company_ledger.go` (12,6 KB) |
| 8 | Skills Manager | `IMPLEMENTED_PARTIAL` | `skill_activation.go` (11,2 KB) — ativação existe; **«Skill Studio»** (criar/importar/versionar/comparar/restaurar/distribuir) não medido |
| 9 | Avaliações e feedback | `IMPLEMENTED_PARTIAL` | `business_status.go` — `BusinessMetricSource` + relatórios de Finance/Commerce/Campaign/Sales/Inbox; componente **declarado** `observability_evaluation_engine`. **Avaliação por agente** (qualidade, retrabalho, custo por agente) não encontrada |
| 10 | Equipes reutilizáveis | `IMPLEMENTED_VERIFIED` | `company_factory.go` (15,7 KB) — templates e procedimento de desbloqueio |
| 11 | Conexões pessoais e compartilhadas | `IMPLEMENTED_VERIFIED` | `connectors*.go` + catálogo de conectores |
| 12 | Gerenciamento de credenciais | `IMPLEMENTED_VERIFIED` | secrets + credenciais com rotação (`multillm`) |
| 13 | Multi-organização | `IMPLEMENTED_VERIFIED` | isolamento multitenant com testes de vazamento entre organizações |
| 14 | Histórico de execução e artefatos | `IMPLEMENTED_VERIFIED` | artefatos + `business_snapshots.go` |
| 15 | Recuperação de trabalho interrompido | `IMPLEMENTED_VERIFIED` | `HADES_STAGE_31_RECOVERY.md` + supervisor |
| 16 | Auditoria de ações | `IMPLEMENTED_VERIFIED` | `service_agent_audit.go` + cadeia de auditoria |
| 17 | Planejamento com versões | `IMPLEMENTED_VERIFIED` | planos + contratos versionados (`component_contracts*`) |
| 18 | Supervisão visual | `IMPLEMENTED_VERIFIED` | `app/ui/app/src` (Cowork/Agentic Console) |
| 19 | Aprendizagem organizacional | `MISSING` (candidato) | nenhuma ocorrência de `learning`/`aprendizagem` em código de produção; `company_growth.go` é **crescimento comercial** (campanhas, afiliados, produtos, encomendas). Sem ciclo proposta → avaliação → aprovação → regra. **Classificado como candidato, a confirmar contra a referência** |
| 20 | Integração de diferentes harnesses | `IMPLEMENTED_VERIFIED` | `cli_harness.go` + `capability_policy.go:40 ScopeHarnessCLI` |

**Resumo**: 16 `IMPLEMENTED_VERIFIED`, 3 `IMPLEMENTED_PARTIAL` (4, 8, 9), 1 candidato a `MISSING` (19).

## 2. O que esta matriz NÃO afirma

- **O lado de referência não foi estudado.** O complemento manda comparar com
  `github.com/paperclipai/paperclip`; esta matriz classifica **só o lado Hades**. Sem o lado de
  referência, cada linha diz «o Hades tem isto», **não** «isto é equivalente ao Paperclip».
- **Nada aqui é `ADAPTER_ONLY` nem `BLOCKED_EXTERNAL`** — não avaliei adaptadores de terceiros
  nem dependências externas para estas 20 capacidades.
- Os três `PARTIAL` são parciais **pelo que foi procurado**, e a busca foi por nomes do código.
  Alargar a busca pode mover 8 e 9 para `VERIFIED`; o item 19 é o único em que a ausência foi
  procurada em comentários, nomes de tipos e nomes de funções.

## 3. Próximo passo desta frente, em ordem

1. **Estudar a referência** (Paperclip) e completar a matriz com a coluna do outro lado.
2. Verificar 8 e 9 por leitura dirigida (`skill_activation.go`, `business_status.go`), antes de
   qualquer implementação — *a lição do E-619 é que procurar é mais barato do que implementar*.
3. Só então implementar, e apenas as lacunas que sobreviverem ao passo 2.
4. Não copiar o backend Node do Paperclip como segundo backend; reutilizar Company OS, Mission
   Control, memória e tarefas existentes.

## 4. Evidência

As classificações desta matriz vêm de medições feitas no repositório a 2026-10-14 (listagem de
ficheiros com tamanho, `Select-String` por nomes de tipo/função e por comentários de desenho, e
`grep` por nomes de ferramentas). O run de CI `5bba1132e` (PR #438) fechou com **23 checks e zero
vermelhos** na mesma janela. Nenhum código de produção foi alterado para produzir esta matriz.
