# Estágio 35 — o ATO HUMANO entrou na trilha de auditoria (E-150)

## 1. O limite que esta rodada fecha

O estado declarava: *"mudar a lista de aprovadores NÃO entra na trilha de auditoria"*. E a diretriz da
missão (§11.1) pede mais que isso: *"`/approve` e `/deny` representam aprovação ou rejeição de ação de
risco, com autenticação e trilha de auditoria"*.

A trilha existente cobria o que o **agente** executou. Aprovar, negar e definir quem aprova são atos
**humanos** de risco — e ficavam registrados apenas no store de aprovações, que guarda **estado**, não a
história do ato.

## 2. O que passou a ser registrado

| Ato | Entrada na trilha | Estado |
| --- | --- | --- |
| Aprovar uma escrita | `approval.decide` | `executed` |
| **Negar** uma escrita | `approval.decide` | **`blocked`** — negar é decisão tomada, não ausência de decisão |
| Definir quem aprova | `policy.approvers.set` | `executed` |

Detalhes deliberados:

- **`AgentID: "policy"`** distingue ato humano de chamada de agente na mesma trilha: sem isso, "quem fez"
  ficaria ambíguo entre o agente e a pessoa.
- **O ator vai no `Reason`** (`por <ator>: <motivo>`), porque auditoria sem autoria não responde à pergunta
  que a motiva.
- **As entradas entram na MESMA cadeia de hashes** das ferramentas — teste prova que misturar os dois tipos
  não quebra a verificação.
- **A ausência de trilha é DECLARADA**: se o store de auditoria não estiver configurado, o ato continua
  acontecendo (a decisão já está no store de aprovações), mas a resposta traz
  `audit_recorded: false` e `audit_reason` com o motivo. Auditoria que falta **em silêncio** é pior que
  auditoria ausente, porque parece existir.

## 3. Erros meus nesta rodada (registrados)

1. **A declaração do erro caiu DENTRO de uma função**: usei `t.index("ErrServiceAgentAuditChainBroken")` para
   ancorar a inserção e o índice casou o **uso** (`fmt.Errorf(...)`), não a declaração — o compilador pegou
   (`declared and not used`). Lição: ao inserir por âncora, ancore na **declaração** (`^var X = `), não no
   primeiro texto que casa.
2. **Duas asserções minhas estavam erradas, não o código**: eu esperava **1** entrada na trilha depois de um
   turno + uma decisão, mas o turno **também** registra (`approval_required`) — são 2. O teste passou a
   procurar a entrada **pelo nome** em vez de contar posições, que é o que a pergunta realmente faz.
3. **Mais 2 falsos positivos de `misspell`** (`eles` → *eels*), a mesma família já vista nesta missão:
   reescritos para "esses atos" / "tais atos", nunca `//nolint`.

## 4. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run TestServiceAgent` | **56/56 PASS** (5 novos de auditoria de política) |
| `go test ./internal/agent/ -run TestServiceAgent -count=5` | **ok 15,233s** (sem flake) |
| `go test ./internal/agent/ -count=1` | **ok 63,631s** |
| `go test ./server/ -count=1` | **ok 21,828s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agent 20,815s · server 6,067s · exit 0) |
| `gofmt -l internal/agent server` | **sem saída** |

Os testes novos: a decisão de aprovar consta **com o ator** e como ato de **política**; a **negativa** consta
como **bloqueada** com o motivo; a mudança de aprovadores consta com o tamanho da lista; **sem trilha** o
erro é declarado (`ErrServiceAgentAuditUnavailable`) e a decisão **continua** registrada no store; e a cadeia
de hashes **permanece válida** com eventos de ferramenta e de política misturados. A rota verifica
`audit_recorded: true` e que `approval.decide` e `policy.approvers.set` aparecem na trilha com o nome de quem
decidiu.

## 5. Limites declarados

- **Sem quórum**: uma decisão de ator autorizado basta; não há dupla aprovação para valores altos.
- **A trilha registra o TAMANHO da lista, não o conteúdo**: a lista em si vive no store de aprovações, e o
  histórico de cada nome não é auditado.
- **A entrada da decisão não carrega os PARÂMETROS** da escrita aprovada (eles estão na impressão digital do
  store de aprovações) — quem audita precisa cruzar os dois registros.
- **Sem notificação** ao aprovador: o pedido aparece na listagem, mas nada avisa um humano.
- **O prazo é do store, não por ferramenta**, e a impressão digital cobre apenas os campos listados.
