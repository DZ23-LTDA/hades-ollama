# Estágio 35 — ferramentas dos agentes deixaram de ser promessa (E-144)

## 1. O limite que esta rodada fecha

O estado declarava: *"ferramentas apenas DECLARADAS (a execução real depende dos módulos de comércio, voz e
conhecimento)"*.

Havia algo pior que a declaração. `DefaultServiceAgentTools()` listava **11 ferramentas como disponíveis**,
e disponibilidade ali era um **booleano escrito à mão**. Três delas — `commerce.order.read`,
`commerce.stock.read`, `company.report.read` — **não tinham executor nenhum**. Ferramenta "disponível" sem
implementação é a forma mais fácil de apresentar capacidade que não existe.

## 2. O que foi entregue

**A disponibilidade passou a ser DERIVADA do registro de executores** (`availableTools()` =
declarado ∩ executável). Um mapa digitado não pode mais afirmar capacidade que ninguém implementou.

| Peça | O que faz |
| --- | --- |
| `ServiceAgentTool` / `serviceAgentTools` | **7 executores reais** ligados aos stores existentes: `crm.contact.read`, `crm.opportunity.write`, `inbox.conversation.read`, `inbox.reply`, `inbox.handoff`, `inbox.assign`, `inbox.metrics.read` |
| `ServiceAgentToolCall` | Trilha por turno: nome, **efeito**, status, resumo e motivo |
| `RunTool` | Ordem de política: declaração do agente → executor → consentimento (para saída) → execução |
| `ToolExecutors()` / `UnavailableTools()` | O que existe de verdade e o que é declarado **sem** implementação |
| Turno + rota | `requested_tools` e `tool_arguments` no corpo do turno; a trilha volta na resposta |

## 3. Cinco regras de honestidade (todas travadas por teste)

| Regra | Por quê |
| --- | --- |
| **Pedir não é poder** | quem declara a ferramenta é a **especificação do agente**, não o pedido; pedir `inbox.assign` no recepcionista é **bloqueado**, e executado no supervisor que a declara |
| **Sem executor = INDISPONÍVEL, com motivo** | `commerce.stock.read` deixa de ser "disponível" e passa a ser indisponível — e aparece em `UnavailableTools()` |
| **Sucesso traz dado do STORE** | o resumo de `crm.contact.read` contém o contato real do CRM, não texto genérico |
| **Falha é falha, com o erro real** | `crm.opportunity.write` sem título falha com o motivo verdadeiro e **sem** resumo de sucesso |
| **Escopo vem da sessão** | `ToolArguments.OrganizationID` é **ignorado**; a organização alheia continua vazia depois do turno |

Mais: **saída exige consentimento**; **`inbox.reply` é recusado como ferramenta** (a resposta já sai pelo
caminho único — permitir criaria uma **segunda** mensagem no mesmo turno); e **pedido repetido não duplica
execução**, porque efeito duas vezes por engano é pior que ignorar a repetição.

## 4. Erros MEUS nesta rodada (registrados)

1. **Campos inventados**: escrevi `CRMContact.Channel` e `InboxMetrics.Conversations/Open/SLAViolations` —
   os nomes reais são outros (`Phone`, `TotalConversations`, `OpenConversations`, `SLABreached`). **O
   compilador pegou**; foi leitura que faltou, não decisão.
2. **`RequestedTools` passado como "declarado"**: isso tornaria o **pedido a própria autorização** — pedir
   qualquer ferramenta passaria na verificação de privilégio. Percebi ao escrever o teste do bloqueio;
   agora `declared` é `nil` e a especificação do agente decide.
3. **Contato não passado ao executor**: as ferramentas falhavam com `contact_id` vazio. Ao corrigir,
   apareceu o detalhe real do domínio: **a conversa carrega o TELEFONE** (identidade), não o ID interno do
   CRM — a resolução agora repete a mesma regra do turno, para contexto e ferramenta não discordarem.
4. **Dois edits meus colaram uma linha `func` dentro de um comentário** (erro de sintaxe) e um deles
   deixou um `if` que não fazia nada. Corrigidos; o `gofmt`/compilador foram o detector.
5. **4 falsos positivos de `misspell`** em português (`eles`, `colateral`) — **reescritos**, nunca
   silenciados; e havia um "Efefeito" que era erro de digitação meu.

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run TestServiceAgent` | **14/14 PASS** (7 novos) |
| `go test ./internal/agent/ -count=1` | **ok 60,520s** |
| `go test ./server/ -count=1` | **ok 20,454s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agent 20,372s · server 6,479s · exit 0) |

## 6. Limites declarados

- **Continuam INDISPONÍVEIS, com motivo visível**: `commerce.order.read`, `commerce.stock.read`,
  `company.report.read`, `catalog.read`, `knowledge.search`, `social.draft`, `voice.transfer` — dependem de
  módulos (comércio, empresa, conhecimento, social, voz) que este runtime não recebe. Estão declaradas **de
  propósito**, para aparecerem como indisponíveis em vez de desaparecerem.
- **A trilha é POR TURNO**: ela volta na resposta, mas **não** é persistida como auditoria durável.
- **Não há seleção de ferramenta por modelo**: quem pede é o chamador; autonomia de escolha não foi
  inventada.
- **A ferramenta de escrita não passa por aprovação humana**: `crm.opportunity.write` executa com o pedido do
  turno. Aprovação para efeitos de maior risco (pagamento, publicação externa) segue fora deste caminho.
