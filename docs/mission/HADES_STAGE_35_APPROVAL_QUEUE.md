# Estágio 35 — pedido vencido e fila com idade (E-154)

## 1. Os dois limites que esta rodada fecha

O estado declarava: *"aprovação PARCIAL não expira por si (o prazo só começa quando o quórum é cumprido)"* e
*"SEM NOTIFICAÇÃO ao aprovador"*. O primeiro era o mais grave dos dois: uma primeira assinatura esquecida
podia ser somada, semanas depois, a um segundo olhar sobre um contexto que ninguém revisou junto. **Quórum
com assinatura velha não é quórum: é carimbo.**

| Peça | Comportamento |
| --- | --- |
| `PendingTTL` | Prazo do pedido **pendente** (mesmo com assinatura parcial). **Padrão 24h**; `0` desliga |
| Expiração | O pendente vencido vira **`expired`** — marcado, com decisões preservadas como evidência |
| `Decide` | Recusa pedido **vencido** (`ErrServiceAgentApprovalExpired`): aceitar reabriria contexto não revisado a tempo |
| Fila | `ApprovalQueue` com **o que falta**, **idade** e **atraso** (`OverduePendingThreshold = 4h`) |

## 2. A "notificação" honesta é uma fila que não esconde o tempo

Não existe push, e-mail nem webhook — e **não** vou chamar de notificação o que não é. O que passou a existir
é uma fila **alcançável pela rota** (`GET /service-agents/approvals` → campo `queue`) que diz:

- quantos pedidos estão pendentes e **quantos passaram do limiar** de atenção;
- **idade** de cada pedido e o mais antigo da fila;
- **quantas aprovações faltam** em cada um (o quórum fica visível);
- o **prazo vigente** (`pending_ttl_seconds`), para quem opera saber quando o pedido vence.

Sinónimo de "pendente" e "esquecido" deixa de existir na tela: o atraso é um campo.

## 3. O flake de paridade: o que eu vi e o que eu não sei

Na primeira execução da bateria, `scripts/verify-linux-parity.ps1` **falhou** no pacote do agente — com o
Windows verde no mesmo commit. **Não capturei o nome do teste que falhou** (sobrescrevi a saída ao rodar de
novo): erro meu de processo, e a lição é a mesma de sempre — **capture a saída da falha antes de reexecutar**.

O que fiz com o dado que sobrou, sem inventar causa:

1. rodei a paridade **com filtro** nos agentes de atendimento → **OK** (`ok 2,048s`);
2. rodei os testes novos de aprovação/quórum/fila no container → **OK** (`ok 0,722s`);
3. rodei a paridade **completa três vezes seguidas** → **OK nas três**, sem falhas.

Conclusão honesta: **a falha não se reproduziu em 4 execuções** e não tenho evidência para nomear a causa.
Não afirmo que era "só um hiccup": registro como **risco residual aberto**, com o número de execuções limpas,
e a pendência concreta de investigar se reaparecer com a saída capturada.

## 4. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run 'TestServiceAgent\|TestRuntimeServiceAgents'` | **78/78 PASS** (5 novos + 1 de rota) |
| `go test ./internal/agent/ -count=1` | **ok 67,303s** |
| `go test ./server/ -count=1` | **ok 20,658s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `scripts/verify-linux-parity.ps1` | **OK em 4 execuções** (1 com filtro + 3 completas); 1 falha anterior **não capturada e não reproduzida** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `gofmt -l internal/agent server` | **sem saída** |

Os testes novos: pedido pendente **vence** e exige pedido novo; **decidir o vencido é recusado**; a
**assinatura parcial não conta** para o pedido seguinte (o quórum recomeça do zero); a **fila** mostra idade,
atraso por item e o mais antigo primeiro; a fila **conta os vencidos**; `PendingTTL = 0` **desliga** o prazo
de forma explícita; e a **rota** entrega a fila com `age_seconds`, `approvals_needed`, `pending_ttl_seconds` e
`overdue`.

## 5. Limites declarados

- **Não há push**: a fila é visível, mas ninguém é avisado ativamente — o aprovador precisa olhar.
- **O prazo é do store**, não por ferramenta: 24h para todo pedido pendente.
- **O limiar de atraso (4h) é fixo**, não configurável por organização.
- **A expiração é avaliada nas leituras** (`FindApproved`, `ListApprovals`, `ApprovalQueue`): um pedido vencido
  que ninguém consulta fica com o estado antigo no arquivo até a próxima leitura — o efeito prático é o
  mesmo (nunca autoriza), mas o arquivo pode estar "atrasado" em relação ao relógio.
- **A falha de paridade não reproduzida** segue como risco residual aberto (acima).
