# Estágio 35 — a trilha de execução virou auditoria durável (E-146)

## 1. O limite que esta rodada fecha

O estado declarava: *"a trilha é POR TURNO — volta na resposta, mas não é auditoria durável"*.

Trilha que desaparece no reinício não serve para auditoria: a pergunta que se faz **depois** de um
incidente é *"o que aquele agente executou ontem?"* — e a resposta não pode depender de alguém ter guardado
a resposta HTTP.

## 2. O que foi entregue

| Peça | O que faz |
| --- | --- |
| `ServiceAgentAuditStore` | Persiste **um arquivo por organização** (`<dataRoot>/.agent-service-audit/<org>.json`) com escrita atômica |
| Ligação no `Runtime` | `serviceAgentRuntime.SetAuditStore(NewServiceAgentAuditStore(serviceAgentAuditRoot(dataRoot)))` |
| Gravação em **ponto único** | Logo após as ferramentas rodarem: **todo** caminho de saída do turno (escalada, resposta, falha) passa pelo registro |
| `AuditTrail(org, limit)` | Leitura do **disco** a cada chamada, mais recentes primeiro, escopada por organização |
| `AuditStatus(org)` | `Persisted`, `Storage`, `Entries`, `PerOrganization`, `Error` |
| Rota `GET /service-agents/audit` | Torna a auditoria **alcançável**; devolve as entradas **com o estado junto** |

## 3. Seis regras de honestidade (todas travadas por teste)

| Regra | Por quê |
| --- | --- |
| **Sobrevive ao reinício** | auditoria que desaparece não é auditoria |
| **A TENTATIVA também é evidência** | bloqueado, indisponível e falho entram na trilha: saber que uma ferramenta foi **bloqueada** é informação de auditoria |
| **Teto POR ORGANIZAÇÃO** | um tenant com volume não pode expulsar a evidência de outro (defeito real corrigido na DLQ, em E-141) |
| **Isolamento conferido no ARQUIVO** | o arquivo de uma organização não contém registro de outra |
| **Falha de gravação é REPORTADA** | `turn.AuditError` e `status.Error`; perder registro em silêncio é o pior desfecho de uma auditoria |
| **O TEXTO DO CONTATO NÃO ENTRA** | guardamos ferramenta, efeito, estado, resumo do store e motivo — nunca o que o cliente escreveu |

A última tem teste próprio: o turno recebe um texto contendo um número de cartão e a trilha em disco é
lida para provar que **nem o texto nem o número** aparecem nela.

## 4. O achado que a paridade Linux pegou (antes do CI)

A primeira execução de `scripts/verify-linux-parity.ps1` **falhou**:

```
--- FAIL: TestServiceAgentAuditReportsPersistenceFailure
service_agent_audit_test.go:219: open .../not-a-directory/audit/org-a.json: not a directory
```

A causa é a **mesma lição institucionalizada em E-115/E-127**: a injeção de falha ("um arquivo no lugar do
diretório") manifesta-se **diferente por sistema** — no Linux a leitura devolve **ENOTDIR** (erro); no
Windows o caminho aparece como inexistente, e o que sobra é o erro de **gravação** guardado no estado. Meu
teste afirmava o **mecanismo** de um sistema só. Corrigido para afirmar a **propriedade** que importa — *a
falha é reportada, por um caminho ou pelo outro* — e a paridade passou.

**Registro:** foi o gate de paridade que detectou, **antes** de o CI ficar vermelho. Rodar o gate no SO onde
o gate roda continua sendo o que separa "passa aqui" de "passa".

Também corrigido nesta rodada: o linter `nilerr` apontou que `AuditStatus` engolia o erro de leitura
devolvendo `nil`. Agora **arquivo ilegível sobe como erro** — uma trilha corrompida apresentada como vazia
apagaria a evidência de que existiam execuções (mesmo raciocínio que a DLQ já usava).

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ -run TestServiceAgent` | **24/24 PASS** (7 novos de auditoria) |
| `go test ./server/ -run TestServiceAgent` | **3/3 PASS** (1 novo: a rota da trilha) |
| `go test ./internal/agent/ -count=1` | **ok 60,916s** |
| `go test ./server/ -count=1` | **ok 19,447s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agent 18,650s · server 5,683s · exit 0) |
| `gofmt -l internal/agent server` | **sem saída** |

## 6. Limites declarados

- **A trilha não é assinada nem encadeada criptograficamente**: um operador com acesso ao disco pode
  editá-la. Integridade por cadeia de hashes é a evolução natural e **não** está feita.
- **Sem exportação** da trilha (CSV/JSON por período) e **sem retenção por tempo** — o teto é por
  quantidade.
- **Gravação é síncrona no turno**: preferi a escrita no caminho do turno à assincronia que poderia perder
  registro em queda abrupta; o custo é latência, e isso fica declarado.
- **Permanecem indisponíveis** `knowledge.search`, `social.draft` e `voice.transfer`; e a escrita segue
  **sem aprovação humana**.
