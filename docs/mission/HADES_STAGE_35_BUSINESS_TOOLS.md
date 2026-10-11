# Estágio 35 — social e voz deixaram de ser "indisponível" (E-151)

## 1. O que esta rodada liga

Restavam **três** ferramentas sem executor. Duas ganharam ligação a módulos que **já existiam e já tinham
testes** — nenhum módulo novo foi criado:

| Ferramenta | Módulo que a executa | Efeito |
| --- | --- | --- |
| `social.draft` | `CompanyStore.CreateSocialDraft` (estágio 26) | escrita (exige aprovação) |
| `voice.transfer` | `CompanyStore.RequestTelAgentHandoff` / Tel-Agent (estágio 28) | escrita (exige aprovação) |
| `knowledge.search` | **continua indisponível** — e com o motivo declarado | — |

## 2. Três regras aplicadas nas novas ferramentas

1. **Rascunho social nasce em SANDBOX e NÃO é publicado.** O módulo já marca `ApprovalRequired = true`,
   `Approved = false` e não preenche `PublishedAt`; o teste confere os três. O resumo que o agente vê diz
   *"sem publicação"*, para que ninguém leia o sucesso como algo que foi ao ar.
2. **Nada de adivinhar a empresa.** Com `company_id` explícito a ferramenta usa essa empresa — e **recusa**
   empresa de outra organização; sem `company_id`, usa a empresa quando existe **exatamente uma**; com duas
   ou mais, **falha pedindo `company_id`**, porque escolher por conta própria seria decidir pelo humano.
3. **`voice.transfer` exige `session_id`** e transfere uma sessão **real** do Tel-Agent: sem o identificador,
   falha explicando o que falta (não encaminha "a sessão mais recente"); com ele, a sessão termina em
   `WAITING_HUMAN` no store.

## 3. Por que `knowledge.search` continua indisponível

A busca de conhecimento existe (`ContextStore.SearchMemories`), mas é indexada por **projeto** — e **não há
mapeamento organização→projeto** no runtime. Ligar a ferramenta exigiria **inventar** um escopo de busca: a
busca de um tenant poderia alcançar memória de outro, que é exatamente o que a missão proíbe.

Então a ferramenta segue **INDISPONÍVEL, com motivo**, e um teste trava isso: ela aparece como
`unavailable` com razão e **não** consta no registro de executáveis. Declarar a lacuna é melhor que
preencher com um chute.

## 4. A aprovação continua valendo — e agora cobre os novos parâmetros

`ToolArguments` ganhou `company_id`, `session_id`, `provider` e `body`, e a **impressão digital da aprovação
passou a incluí-los**. Um teste prova o caso que importa: **aprovar a transferência de uma sessão de voz não
autoriza outra** — a segunda exige decisão nova.

## 5. Erros meus nesta rodada (registrados)

1. **O MESMO erro de ancoragem da rodada anterior, repetido**: ancorei a declaração do novo erro em
   `ErrServiceAgentApprovalInvalid` — que aparece primeiro **em uso**, dentro de `Decide` — e a declaração caiu
   **dentro da função**; o compilador pegou (`declared and not used`). Na rodada anterior eu já havia
   registrado a lição e ela **não foi aplicada de imediato**: a correção é mecânica — ancorar em
   `^var ` (a **declaração**), nunca no primeiro texto que casa.
2. **Nomes inventados de novo**: escrevi `TelAgentSessionHumanHandoff` para o estado da sessão; o real é
   `TelSessionWaitingHuman`, e o campo é `State` (não `Status`). O compilador pegou os dois. A regra que
   funciona nesta base continua sendo: **ler o tipo antes de usar o campo**.
3. **Mais 2 falsos positivos de `misspell`** (`transfere` → *transferred*), reescritos para *encaminha*.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run TestServiceAgent` | **61/61 PASS** (5 novos) |
| `go test ./internal/agent/ -run TestServiceAgent -count=5` | **ok 16,098s** (sem flake) |
| `go test ./internal/agent/ -count=1` | **ok 57,765s** |
| `go test ./server/ -count=1` | **ok 22,407s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agent 22,146s · server 5,857s · exit 0) |
| `gofmt -l internal/agent server` | **sem saída** |

Os cinco testes novos: rascunho social **exige aprovação e não cria nada antes dela**, e nasce em sandbox não
publicado; **com duas empresas o pedido falha pedindo `company_id`** em vez de escolher; `voice.transfer`
**falha sem `session_id`** e **executa com a sessão real** deixando-a em `WAITING_HUMAN`; a aprovação é
**amarrada à sessão**; e `knowledge.search` segue **indisponível com motivo**.

## 7. Limites declarados

- **Publicação real em rede social não existe**: o rascunho é sandbox e a publicação exige conta autorizada,
  scopes válidos e fluxo live — `BLOCKED_BY_EXTERNAL_DEPENDENCY`.
- **Duas aprovações coexistem** para o mesmo rascunho: a do agente (store de aprovações) e a da empresa
  (`queueCompanyApproval` do módulo social). São portões diferentes com propósitos diferentes, e isso fica
  declarado em vez de parecer duplicação acidental.
- **`voice.transfer` não cria sessão**: a sessão precisa existir (criada pelo Tel-Agent); o agente encaminha,
  não inicia chamada — telefonia real segue `BLOCKED_BY_EXTERNAL_DEPENDENCY`.
- **`knowledge.search` segue indisponível** enquanto não houver mapeamento organização→projeto confiável.
