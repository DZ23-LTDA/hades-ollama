# Estágio 27 — propostas comerciais e pós-venda

## 1. O que faltava

O AI Sales Workforce tinha agentes, CRM e catálogo, mas **não tinha proposta comercial** — o
documento com itens e preço que o vendedor envia — nem **pós-venda**, o acompanhamento depois da
entrega. Um agente de vendas sem proposta não vende; um sem pós-venda não descobre que o cliente
ficou insatisfeito.

## 2. Decisões deliberadas

| Decisão | Por quê |
| --- | --- |
| **PREÇO VEM DO CATÁLOGO, NUNCA DO CHAMADOR** | a proposta recebe apenas **SKU e quantidade** e o total é **recalculado no servidor**. Aceitar preço enviado por quem chama a API permitiria proposta com valor inventado e desconto forjado. **SKU fora do catálogo é erro**, não linha com preço zero |
| **ENVIAR EXIGE CONSENTIMENTO** | proposta é mensagem para terceiro: o envio passa pelo `ConsentRegistry`, então quem pediu **opt-out não recebe proposta** — a mesma política do resto do sistema, sem caminho alternativo |
| **VIGÊNCIA É RESPEITADA** | aceitar proposta vencida é recusado (`ErrProposalExpired`), em vez de aceitar em silêncio um preço que já não vale |
| **DECISÃO É TERMINAL** | aceita/recusada não volta a rascunho: reabrir apagaria o resultado (mesma disciplina do pipeline do CRM) |
| **PÓS-VENDA É MEDIDO** | o acompanhamento tem prazo, é **idempotente por pedido** e o **atraso é contável** — como o SLA da caixa de entrada. Pós-venda que ninguém mede não é operação, é intenção |
| **ESCRITA COM DESFAZIMENTO** | se a persistência falha, o estado em memória é **revertido**, para não relatar sucesso que não existe no disco |

## 3. Verificação (4 testes, nesta máquina)

| Teste | O que prova |
| --- | --- |
| `TestProposalPricesComeFromCatalogAndSendingRequiresConsent` | total **recalculado** (2×9900 + 1×19900 = **39700**), SKU desconhecido e quantidade zero **recusados**, envio **sem** consentimento recusado, envio **com** consentimento aceito e, **depois do opt-out**, uma nova proposta **não** é enviada |
| `TestProposalExpiryAndTerminalDecision` | decidir **rascunho** é recusado; **fora da vigência** é recusado; dentro da vigência aceita com autor e data; decisão **terminal** não se repete; decisão desconhecida é erro declarado |
| `TestFollowUpIsIdempotentPerOrderAndOverdueIsMeasurable` | prazo padrão (72 h), **idempotente por pedido**, motivo obrigatório, escopo obrigatório, **atraso medido** (0 antes, 1 depois), conclusão registrada e **não** mais contada como atraso |
| `TestSalesWorkforceIsScopedByOrganization` | propostas e pós-venda de outra organização **inacessíveis**; a dona vê |

**Achado durante a implementação:** a primeira gravação falhava com *"caminho não encontrado"*
porque `writeJSONAtomic` **não cria o diretório** — a proposta nunca seria persistida. Corrigido
com `MkdirAll` antes da escrita, o que é o que os outros stores fazem.

## 4. Limites declarados

- A proposta **não** é enviada por um provedor externo: o envio registra o estado e a política;
  a entrega real depende de canal configurado (estágio 33, com credencial).
- **Sem desconto negociado**: o preço é o do catálogo, sempre. Desconto exigiria política de
  aprovação própria, e não foi inventado.
- **Sem imposto, frete ou parcelamento**: nada disso é calculado aqui.
- Pós-venda é um **acompanhamento com prazo**, não uma pesquisa de satisfação com NPS.

## 5. Verificação final

- `go test ./internal/agent/ -run 'TestProposal|TestFollowUp|TestSalesWorkforce'` → **4/4 PASS**.
- `go test ./internal/agent/` → **ok** · `go test ./server/` → **ok**.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` → **0 issues** (falsos positivos do
  `misspell` em "validade" foram **reescritos**, não silenciados).
