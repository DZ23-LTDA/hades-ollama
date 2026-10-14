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

## 5. Coluna de REFERÊNCIA — README do Paperclip (`master`, MIT), lido a 2026-10-14

O Paperclip descreve-se como «Node.js server e React UI que orquestra uma equipa de agentes para
tocar um negócio» — **quatro pilares**: (1) *Agentic Task Manager*, (2) *Org Chart for Agents*,
(3) *Agent Employee Training* (Skill Studio, evals, revisões de desempenho, templates de equipa),
(4) *Agentic OS* (runtime multi-provedor, sandboxing e MCP, SSO/GRC/RBAC e custos, privacidade e
tracing, conexões, histórico). O próprio `ROADMAP.md` do repositório marca o estado de cada
capacidade — e é isso que permite julgar **paridade** em vez de julgar **existência**.

### 5.1 O que o roadmap do Paperclip marca como FEITO (✅) e que o Hades não tem

| Capacidade | Paperclip | Hades | Efeito na matriz |
| --- | --- | --- | --- |
| **Agent evals & feedback** | ✅ feito | `PARTIAL` (métricas de negócio; sem avaliação por agente) | **lacuna real de paridade** (item 9) |
| **Skills Manager, Skill Studio & Skills Store** | ✅ feito | `PARTIAL` (`skill_activation.go`; sem studio/versões/loja) | **lacuna real** (item 8) |
| **Skill Version History & Restore** | ✅ feito | não encontrado | **lacuna real** |
| **Cloud / Sandbox agents** (e2b, Cloudflare, Daytona, Modal, Novita, K8s) | ✅ feito | `MISSING` | **confirma o `BLOCKED_EXTERNAL` do Cloudroom** (itens 1 e 4): outros já o fazem, nós não |
| **Heartbeat Execution** (fila de despertar em BD, com *coalescing*, orçamento, injecção de segredo, carregamento de skills) | ✅ feito | `PARTIAL` (ciclo do supervisor, não heartbeat por agente) | **lacuna real** (item 4) |
| **MCP Tool Gateway & Apps** (acesso governado a ferramentas) | ✅ feito | não medido nesta matriz | a medir |
| **Deep Planning** (planos versionados e aprovados) | ✅ feito | planos existem; versionamento de plano não medido especificamente | a medir |

### 5.2 O que muda a classificação do Hades — para MELHOR

- **Item 19 (Automatic Organizational Learning): `MISSING` no Hades, mas ⚪ no roadmap do
  Paperclip também** ⇒ *não é lacuna de paridade*. Reclassificado: **`NOT_APPLICABLE` para
  efeitos de paridade** (ambos ausentes). O Hades não está atrás aqui; ninguém está à frente.
- **Memory / Knowledge: 🟡 (parcial) no Paperclip; `IMPLEMENTED_VERIFIED` no Hades** (Estágio 15
  governado + `knowledge.search` + 8 rotas auditadas) ⇒ **nesta capacidade o Hades está à
  frente**, e isso fica dito porque é medido — não como propaganda.
- **Atomic task checkout: ✅ nos dois** (o Paperclip diz «execution locks»; o Hades tem *lease* e
  *claim* idempotente com testes de *reclaim*) ⇒ paridade confirmada dos dois lados.

### 5.3 Honestidade que o próprio Paperclip pratica, e que vale registar

O README declara limites que muitos produtos escondem: «*enforcement uses recorded spend; usage
reporting and in-flight work can delay a stop*» (o travão de orçamento pode chegar tarde); «*plain
environment values and local paths can remain*» em pacotes exportados (a limpeza de segredos não
é total); «sandbox providers extend execution beyond the local host; **availability depends on the
configured environment and adapter**». *É a mesma disciplina de declarar janelas residuais que
esta missão usa — e é a razão de eu não tratar «o Paperclip faz» como «nós conseguimos fazer
igual».*

## 6. Auditoria da lacuna «Skill Version History & Restore» — com o ponto de inserção

**Medido** (`internal/agent/context.go`):

| facto | linha |
| --- | --- |
| `skills map[string]SkillManifest` — **uma versão por ID** | `:27` |
| `registerSkill` escreve por cima da anterior | `:816`, `:841` |
| o manifesto que SAI está em `previous, existed := s.skills[manifest.ID]` | `:839` |
| persistência por ID: `<root>/skills/<ID>.json` | `:832-834` |
| `writeJSONAtomic` com rollback em falha | `:844`, `:845-854` |
| `Trusted = false` imposto na entrada | `:817` |
| promoção de confiança **governada por política**: `PromoteSkillTrustedForOrganization` | `:941` |
| manifesto já tem `ID` + `Version` validados | `:764`, `:788-796` |

**Conclusão**: o versionamento **não** falta (o manifesto tem id+versão validados); o que falta é
**guardar** as versões em vez de as substituir. A lacuna é estreita e concreta.

**Desenho aprovado para implementar:**
1. em `:839`, se `existed && previous.Version != manifest.Version`, arquivar `previous` em
   `<root>/skills/<ID>.versions/<versão>.json` com o mesmo `writeJSONAtomic`;
2. **falha ao arquivar REPROVA o registo** (fail-closed, coerente com o rollback que já existe em
   `:845-854`): *se não se consegue preservar a versão que sai, não se substitui* — perder
   funcionalidade é preferível a perder histórico em silêncio;
3. `SkillVersionsForOrganization(org, id)` lista as versões arquivadas (versão, ficheiro, tamanho),
   rejeitando ficheiros inválidos com `validateSkillManifest`;
4. `RestoreSkillVersionForOrganization(org, id, versão)` lê o arquivado e **volta a passar por
   `registerSkill`** — o que impõe `Trusted=false` de graça e faz a confiança continuar a exigir
   `PromoteSkillTrustedForOrganization` com política;
5. testes: (a) registar a v2 **preserva** a v1 e a lista mostra as duas; (b) restaurar devolve o
   manifesto antigo ***com `Trusted=false`**, mesmo que o arquivado tenha sido promovido; (c)
   restaurar versão inexistente falha.

**O que este desenho NÃO faz**: não promove confiança, não apaga versões, e não mexe no que a
activação (`skill_activation.go`) já faz.

## 7. Auditoria da lacuna «Heartbeat Execution» — a lacuna é de COMPOSIÇÃO

**Medido** (busca pelos nomes que o código usa):

| termo | resultado |
| --- | --- |
| `WakeQueue` / `fila de despertar` | **0 ocorrências** |
| `Coalesce` / `coalesc` / `Wakeup` / `acordar` | 19 ocorrências, **todas** `COALESCE(` de SQL (rotação de chaves, segurança) — nada de fila |
| `Heartbeat` | 51 ocorrências; as primeiras são `DeviceStore.Heartbeat(deviceID, token, capabilities)` — heartbeat de **DISPOSITIVO**, não de agente |

**O que o Hades JÁ tem**: fila durável (`OrchestrationJob` com *lease*, *claim* idempotente e DLQ no
Redis) ✅ · agendador do supervisor (`business_supervisor_scheduler.go`, ciclo no intervalo
configurado) ✅ · heartbeat de dispositivo ✅.

**O que NÃO tem**: uma fila de despertar **por agente** com **coalescing** — «vários sinais
pendentes para o mesmo agente ⇒ UM despertar».

**Conclusão que muda o trabalho**: a lacuna é de **composição**, não de infraestrutura. As peças
existem; falta a semântica. E isso implica a decisão de desenho já tomada: **não se cria uma fila
nova ao lado da existente** — seria o «segundo sistema de tarefas» que os complementos proíbem. O
despertar coalescido compõe-se sobre a fila durável que já tem lease e idempotência.

**Primeiro passo testável, antes de ligar à fila**: a regra de coalescência como função pura
(`DespertaresCoalescidos(sinais) []Despertar`), com teste de que N sinais do mesmo agente e dentro
da janela produzem **um** despertar que carrega **todas** as razões — e que sinais fora da janela,
ou de agentes diferentes, **não** se fundem. *A garantia vai para onde é testável; a ligação à
fila vem depois, e é a parte que pode falhar por razões que o teste puro não vê.*

### 7.1 Auditoria do WIRING — e ela qualifica o E-633

Antes de ligar o despertar coalescido à fila, medi o que a fila **já faz**:
- `queue.go:216` — `JobQueue.Enqueue(missionID, maxAttempts)`;
- `queue_test.go:287` — **`TestJobQueueEnqueueIsIdempotentByMission`**: enfileirar a mesma missão
  duas vezes **não duplica** o job;
- `runtime.go:1947` — `Runtime.EnqueueMission(missionID)` é onde um despertar vira job.

**Conclusão que qualifica o E-633**: a fila durável **já coalesce por MISSÃO**. O que ela **não**
faz é fundir **razões diferentes** (tarefa + mensagem + rotina) num só despertar, nem é escopada
por **agente** (o `QueueJob` carrega `missionID`, não um agente).

**Desencontro de modelos, dito antes de alguém o descobrir a meio**: a unidade da fila é a
**missão**; o coalescimento do E-633 é por **agente**. Ligá-los exige um mapeamento
agente→missões, que **não medi**.

**Consequência honesta**: o ganho de **carga** do E-633 é **menor** do que eu supus (a fila já
deduplica por missão); o que ele acrescenta é juntar as **razões** numa só notificação — ganho de
**observabilidade e experiência**, não de volume. Fica escrito para não se vender o contrário.
