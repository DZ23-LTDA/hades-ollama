# Verificação do Estágio 13 — Motor de Automação (nós, retry, DLQ, idempotência)

**Método:** releitura da LISTA DE ATIVIDADES do estágio contra o código e as
evidências do índice, atividade por atividade, sem presumir cobertura. Cada linha
aponta o arquivo e a evidência que a sustenta. Divergência encontrada vira
correção de código, não nota de rodapé.

## 1. Atividade por atividade

| Atividade do estágio | Cobertura | Onde está | Evidência |
| --- | --- | --- | --- |
| Gatilhos | COBERTO | `trigger.interval`, `trigger.webhook` | E-074, E-077 |
| Condições | COBERTO | `condition.if` (`scheduleConditionMatches`) | E-074 |
| Ações | COBERTO | `action.mission` com retry e idempotência | E-074, E-076, E-077 |
| Conectores | COBERTO | `action.connector` via `CallForOrganization` | E-084 |
| Webhooks | COBERTO | `trigger.webhook`, replay de webhook na política de segurança | E-074, E-071 |
| Schedules | COBERTO | store + despacho por tick + supervisor | E-074, E-075 |
| Variáveis | COBERTO | builtins + publicação por `action.transform`/`logic.loop` | E-082 |
| Transformações | COBERTO | `action.transform` (`expandScheduleTemplate`) | E-082 |
| Branches | COBERTO (após correção) | `condition.if` irmãos com `Expect` opostos = if/else | E-085 |
| Loops limitados | COBERTO | `logic.loop` com teto de itens e de missões por execução | E-082 |
| Aprovações | COBERTO | `approval.request` com espera entre execuções | E-083 |
| Filas | COBERTO | fila de jobs para as missões criadas pelos nós | (pré-existente) |
| Retry | COBERTO | `runWithRetry` por nó | E-077 |
| DLQ | COBERTO | `ScheduleStepDeadLetter` por nó | E-080 |
| Idempotência | COBERTO | ID derivado de `sha256(organização + chave)` | E-076 |
| Reprocessamento | COBERTO | `ReprocessScheduleFlow` reusando a execução | E-079 |
| Persistência | COBERTO | store em disco + reinício de worker no teste | E-081 |
| Logs por nó | COBERTO | `Schedule.LastRun.Steps` persistido e visível na interface | E-074, E-075 |
| Versionamento | COBERTO | `Schedule.FlowVersion` + `RunKey` | E-081 |
| Testes | COBERTO | suíte do estágio (motor, rotas e interface) | E-074…E-085 |
| Execução manual | COBERTO | `POST /schedules/:id/run` executando o grafo | E-078 |
| Execução programada | COBERTO | tick do supervisor com relatório por nó | E-074 |

## 2. O que a releitura ENCONTROU (dois defeitos reais)

A verificação não confirmou o que eu esperava: ela **refutou** a alegação de que
"branches" já funcionava.

1. **Ramo alternativo inalcançável.** `condition.if` com `Expect: failed` era
   barrado por `scheduleDependenciesSatisfied`, que exigia **sucesso** do dependente
   antes de avaliar a condição. Resultado: o `else` nunca rodava — if/else existia
   só no nome. Corrigido com `scheduleStepDependenciesReady`, que trata a
   expectativa declarada.
2. **Pendente tratado como falha.** Depois da primeira correção, uma aprovação
   **pendente** passou a satisfazer o ramo alternativo, disparando o caminho de
   recusa **antes de qualquer decisão humana**. Corrigido com o conjunto
   `pendingSteps`: "não teve sucesso" e "aguarda decisão" são estados distintos.

Os dois defeitos foram encontrados por **teste que mede o efeito**, não por leitura
otimista. Os três testes que os travam estão em `schedule_branch_test.go`.

## 3. Critério de conclusão do estágio

"Workflow real ponta a ponta com falha simulada, retry, retomada e evidência por
nó" — **demonstrado** em E-081, com store em disco, falha transitória simulada,
retry absorvendo, reinício de worker e retomada criando **zero** missão nova.

## 4. LIMITES DECLARADOS (não são cobertura)

1. **Conector homologado contra servidor HTTPS controlado.** Credenciais de SaaS de
   terceiros **não** foram validadas ao vivo — isso é `BLOCKED_BY_EXTERNAL_DEPENDENCY`
   e pertence aos estágios 25/33/36, não a este.
2. **Falha de passo com efeito ABORTA o fluxo** em vez de desviar para o ramo
   alternativo. O ramo alternativo cobre "concluiu sem sucesso" (negativa de
   aprovação, condição não satisfeita, dependência barrada), **não** falha de
   execução. Comportamento travado por teste, não suposto.
3. **A execução do fluxo é por tick do supervisor**; a fila de jobs cobre as missões
   criadas pelos nós. Não há fila própria por fluxo.

## 5. Veredito

- **Critério de conclusão:** CUMPRIDO (E-081).
- **Lista de atividades:** COBERTA, com as correções desta rodada (E-085).
- **Execução do estágio:** `COMPLETED_VERIFIED`.
- **Capacidade (Workflow Automation Engine):** `IMPLEMENTADO_E_TESTADO`.
- **Release global:** permanece `NOT_RELEASE_READY` — este é **um** dos 38 estágios.
