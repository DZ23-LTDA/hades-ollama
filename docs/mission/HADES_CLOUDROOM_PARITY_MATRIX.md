# HADES × CLOUDROOM — matriz de paridade (lado Hades, medido)

Complemento: **MASTER V2.2 — Hades Session Teleport & Distributed Agent Runtime**.
Método: classificar **só o que foi medido**, por nomes que o código usa, com `ficheiro:linha`.
A lição do E-619 está aplicada aqui: uma capacidade declarada «em falta» sem busca pelo nome certo
é uma afirmação falsa à espera de acontecer.

## 1. Matriz (20 capacidades)

| # | Capacidade (Cloudroom) | Hades | Evidência medida |
| --- | --- | --- | --- |
| 1 | Teleport local ↔ cloud | `BLOCKED_EXTERNAL` | 0 ocorrências de `Teleport`/`transferir sessão`. Depende de **executor remoto**, que não existe (item 4) |
| 2 | Persistência de sessões | `IMPLEMENTED_VERIFIED` | missões persistidas (store JSON) + snapshots + retomada |
| 3 | Retomada nativa dos harnesses | `IMPLEMENTED_PARTIAL` | `cli_harness.go` (Codex/Gemini/Copilot) existe; **retomada nativa** por harness não foi medida |
| 4 | Execução em máquinas remotas | `BLOCKED_EXTERNAL` | 0 ocorrências de `RemoteExecutor`/`executor remoto`. Sem infraestrutura de execução remota no repositório |
| 5 | Isolamento por sessão | `IMPLEMENTED_VERIFIED` | workspace isolado + snapshot + sandbox + isolamento multitenant testado |
| 6 | Branches Git independentes | `IMPLEMENTED_VERIFIED` | workspace isolado/worktree + `gitMergeOriginTool` |
| 7 | Filas duráveis de mensagens | `IMPLEMENTED_VERIFIED` | fila distribuída (Redis) com DLQ, lease e *claim* — testes de reclaim e de ordem |
| 8 | Idempotência por `request_id` | `IMPLEMENTED_VERIFIED` | 339 ocorrências (`IdempotencyKey`, `pedido_idempotente`); **e dedupe persistida** no omnichannel (`omnichannel.go:633`, `omnichannel_dedupe.go`) com 7 testes, incluindo sobreviver a reinício e **declarar** quando não há store |
| 9 | Sleep e wake de sessões | `MISSING` | 0 ocorrências de *sleep/wake* de sessão; as 2 existentes são **suspensão de aprovação** (`whatsapp_gateway.go:36`), não sessão |
| 10 | Recuperação após reinicialização | `IMPLEMENTED_VERIFIED` | retomada de missão (`MissionRecovering`), DLQ persistida, dedupe restaurada, benchmark `retomada_apos_reinicio` |
| 11 | Cancelamento e fechamento | `IMPLEMENTED_VERIFIED` | rota de cancelamento de missão + cancelamento de job |
| 12 | Propagação de eventos | `IMPLEMENTED_VERIFIED` | eventos de missão + stream SSE (`/missions/:id/events/stream`) |
| 13 | Comunicação pais ↔ filhos | `IMPLEMENTED_VERIFIED` | delegação (`Delegation`) + aprovações por passo |
| 14 | Computer-use remoto | `IMPLEMENTED_PARTIAL` | `browserOperatorTool` + `desktopCompanionTool` existem **localmente**; remoto não medido |
| 15 | Compartilhamento de arquivos | `IMPLEMENTED_VERIFIED` | artefatos + orçamento de anexos |
| 16 | Contas e credenciais | `IMPLEMENTED_VERIFIED` | secrets + credenciais com rotação |
| 17 | Estado desconhecido após interrupção | `IMPLEMENTED_PARTIAL` | a **protecção** do §5 existe (dedupe persistida impede repetir o efeito externo, e declara quando não pode garantir); falta o **nome de estado** `UNKNOWN_OUTCOME` e um consumidor de reconciliação |
| 18 | Segurança de execução | `IMPLEMENTED_VERIFIED` | política de capacidades, aprovações, sandbox, DLP, privilégio mínimo |
| 19 | Observabilidade e diagnósticos | `IMPLEMENTED_VERIFIED` | traces, auditoria encadeada, diagnósticos, OTLP |
| 20 | Compatibilidade entre SOs | `IMPLEMENTED_VERIFIED` | CI em ubuntu/windows/macos + ficheiros por plataforma |

**Resumo**: 14 `IMPLEMENTED_VERIFIED` · 3 `IMPLEMENTED_PARTIAL` (3, 14, 17) · 1 `MISSING` (9) ·
2 `BLOCKED_EXTERNAL` (1, 4).

## 2. As duas reclassificações, e por que mudaram

1. **`UNKNOWN_OUTCOME`: de `MISSING` para `PARTIAL`.** O §5 exige não repetir efeitos externos
   após reconexão e registar o desfecho incerto. Medido: `RetryDeadLetter`
   (`omnichannel.go:985`) reescreve a entrada passando o `ExternalMessageID`, e o **dedupe**
   (`omnichannel.go:633`) garante que uma entrega duplicada é aceite **uma vez** — com testes de
   reinício, tecto, expiração e **degradação declarada** quando não há store. *A protecção
   existe; falta o nome.* Implementar só a constante, sem consumidor, seria código não ligado.
2. **`Teleport` e `executor remoto`: `BLOCKED_EXTERNAL`.** Não é uma lacuna de código por fazer
   — é uma capacidade que exige **infraestrutura de execução remota** que este repositório não
   tem. Classificá-la como «por implementar» faria alguém tentar, e o resultado seria uma
   simulação. O complemento é explícito: *testes simulados devem ser rotulados como simulados*.

## 3. O que esta matriz NÃO afirma

- **O lado de referência não foi estudado** (Cloudroom GUI/Core são repositórios externos). Cada
  linha diz «o Hades tem isto», **não** «isto é equivalente ao Cloudroom».
- `BLOCKED_EXTERNAL` aqui significa *«sem o ambiente, não há como medir»* — não é uma promessa de
  que, com o ambiente, funcionaria.
- Nada aqui autoriza declarar `RELEASE_READY`: o §18 do complemento exige execução comprovada em
  **ambientes distintos**, e isso não existe neste repositório.

## 4. Evidência

Medições feitas a 2026-10-14: contagem de ocorrências por nome (`Select-String`), leitura de
`RetryDeadLetter` (`omnichannel.go:985-1010`) e do dedupe (`omnichannel.go:633`), listagem de
ficheiros com tamanho, e o run de CI `5bba1132e` (PR #438) com **23 checks e zero vermelhos** na
mesma janela. Nenhum código de produção foi alterado para produzir esta matriz.
