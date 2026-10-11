# Estágio 35 — escrita passa a exigir aprovação humana (E-148)

## 1. O limite que esta rodada fecha

O estado declarava: *"a escrita segue SEM APROVAÇÃO HUMANA"*. Era o risco residual mais sério do que foi
construído: um agente alterava CRM, caixa de entrada e atribuição por conta de uma frase do cliente — e o
documento de missão diz o contrário (*"aprovação humana é necessária conforme risco"*, com
`REQUER_APROVACAO` entre os estados de permissão declarados).

## 2. O fluxo, em três tempos

1. **O turno pede a escrita.** O runtime verifica o pedido contra uma *impressão digital* de ferramenta +
   escopo + parâmetros. Sem aprovação válida, o estado é **`approval_required`** e **nada é executado** —
   não existe "executou e pediu desculpa".
2. **O humano decide.** `GET /service-agents/approvals?status=pending` mostra o pedido;
   `POST /service-agents/approvals/:id/decide` registra a decisão, com **ator identificado** e motivo.
3. **O turno se repete e executa** — uma única vez: a aprovação é **consumida** ao executar.

## 3. Sete decisões, cada uma com motivo

| Decisão | Por quê |
| --- | --- |
| **LEITURA não exige aprovação; EFEITO exige** | bloquear leitura tornaria o agente inútil sem ganho de segurança |
| **Aprovação amarrada aos PARÂMETROS** | aprovar "criar oportunidade de 100" não autoriza criar de 9.999.900 |
| **USO ÚNICO, consumido ANTES de executar** | aprovação reutilizável é autorização permanente disfarçada; consumir depois deixaria a janela em que duas execuções usam a mesma aprovação |
| **PERSISTE por organização** | aprovação que desaparece no reinício obriga o humano a decidir de novo — e quem não entende por quê aprova no automático |
| **Pedido pendente NÃO se duplica** | repetir o turno enquanto o humano decide não pode inundar a caixa: pedido duplicado esconde quantas decisões faltam |
| **NEGADO é decidido, não ignorado** | a tentativa seguinte fica **bloqueada**, nomeando quem recusou e por quê |
| **Sem store de aprovações, escrita é RECUSADA** | o padrão é negar: um runtime que não sabe pedir autorização não pode executar efeito |

Duas regras adicionais: **decidir exige ator** (aprovação anônima não é aprovação) e o **teto por
organização preserva as PENDENTES** — descartar um pedido pendente por volume silenciaria uma decisão que
alguém precisa tomar.

## 4. Testes que mudaram de comportamento (divulgação)

Quatro testes existentes codificavam o comportamento antigo ("escrita executa no turno") e foram
**atualizados para o fluxo correto** — não desativados:

| Teste | Antes | Agora |
| --- | --- | --- |
| `TestServiceAgentToolExecutesAgainstRealStore` | escrita executava | `approval_required` + **prova de que NADA foi gravado no CRM**, depois aprovação e execução real |
| `TestServiceAgentTurnBlockedToolNotDeclaredByAgent` | supervisor executava direto | pedido, aprovação e então execução |
| `TestServiceAgentToolFailureIsRecordedWithRealError` | falha direta | pendente → aprovado → **falha real** do executor |
| `TestServiceAgentAuditRecordsUnavailableAndFailed` | trilha com 3 tentativas | trilha com 5: `blocked`, `unavailable`, **`approval_required`** e `failed` — todos são evidência |

## 5. Erros meus nesta rodada (registrados)

1. **O guard do meu próprio script impediu a mudança**: usei `if 'ApprovalID' not in t` para inserir o campo
   no struct — e a string já existia (`call.ApprovalID`). O campo não foi inserido e o compilador acusou.
   Lição: guard de idempotência por substring precisa olhar o **alvo** (a declaração), não o arquivo inteiro.
2. **`misspell` 7 vezes** por falsos positivos do dicionário pt-BR (`argumentos` → *arguments*,
   `autor` → *author*, e a forma maiúscula). Reescrito para **parâmetros** e **quem recusou** — nunca
   `//nolint`, conforme a regra da missão.
3. **`gofumpt`**: três `var` seguidos precisavam de bloco agrupado.
4. **Alarme falso que quase virou conclusão errada**: rodei o `golangci-lint` sem `APPDATA`/`LOCALAPPDATA` e
   ele respondeu *"no go files to analyze"*. **Diagnostiquei** (`go list` carregava os dois pacotes) em vez
   de concluir que o linter estava quebrado — era env incompleto, a armadilha já conhecida desta missão.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run TestServiceAgent` | **43/43 PASS** (9 novos de aprovação, 1 novo de rota) |
| `go test ./internal/agent/ -count=1` | **ok 60,460s** |
| `go test ./server/ -count=1` | **ok 18,239s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agent 20,462s · server 6,005s · exit 0) |
| `gofmt -l internal/agent server` | **sem saída** |

Os testes novos cobrem: **nada é gravado sem aprovação**; **pedido não duplicado**; **uso único**;
**amarração aos parâmetros** (aprovar 100 não autoriza 9.999.900, e a escrita não aprovada **não** existe no
CRM); **negado bloqueia nomeando o ator**; **decisão exige ator humano**, e organização B não decide pedido
de A; **sobrevive ao reinício e é isolada por organização** (conferindo o arquivo em disco); **sem store a
escrita é recusada**; e o **teto preserva pendentes**.

## 7. Limites declarados

- **A aprovação NÃO TEM PRAZO**: uma aprovação dada e não consumida continua válida até ser usada. Expiração
  é o próximo passo natural.
- **Não há checagem de PAPEL**: o sistema exige ator identificado, mas **não** verifica se aquele ator tem
  permissão de aprovar. O controle de quem pode aprovar é da camada de autenticação, e isso está declarado.
- **Um aprovador, sem quórum**: não existe dupla aprovação para valores altos.
- **Sem notificação**: o pedido aparece na listagem, mas nada avisa um humano que ele existe (sem push/e-mail).
- **A impressão digital cobre os campos listados** (título, valor, dono, responsável, conversa, contato);
  parâmetros que não estão nessa lista não participam da decisão.
- **A aprovação do gateway de WhatsApp** (`pendingApprovals` do adaptador) é outro mecanismo, para saída de
  mensagem — este store não a substitui nem a duplica.
