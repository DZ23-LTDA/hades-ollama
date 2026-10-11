# Estágio 35 — a aprovação ganhou PRAZO e PAPEL (E-149)

## 1. Os dois limites que esta rodada fecha

O estado declarava, sobre a aprovação humana das escritas: *"não tem prazo"* e *"não há checagem de papel de
quem aprova"*. Os dois estão fechados — e o que **não** foi fechado continua declarado.

## 2. Prazo: aprovação vence, e o vencimento é EVIDÊNCIA

| Peça | Comportamento |
| --- | --- |
| `ApprovalTTL` | Prazo de vigência depois da decisão; padrão **30 minutos** |
| `ExpiresAt` | O instante do vencimento acompanha a aprovação (a resposta da rota traz `expires_at`) |
| Estado `expired` | A vencida é **MARCADA**, não apagada: quem audita vê que houve aprovação e que ela venceu |
| Busca e consumo | O prazo é conferido nos **dois** caminhos — `FindApproved` para de oferecê-la e `Consume` **recusa**; um caminho que consome sem conferir anularia o prazo |
| `ApprovalTTL = 0` | Desliga o prazo de forma **explícita**, e a aprovação então não declara `ExpiresAt` |

Resultado prático: autorização esquecida **não continua valendo**; o turno volta a exigir decisão nova, e o
pedido anterior fica registrado como vencido.

## 3. Papel: quem aprova, e a ausência de verificação como DADO

- `PUT /service-agents/approval-approvers` define a **lista de aprovadores** da organização (exige
  `actor_id`; sem ele, **400** — mudar quem autoriza é decisão de risco).
- Com lista configurada, ator fora dela recebe **403** na rota e `ErrServiceAgentApprovalNotAllowed` no
  store. O pedido **continua pendente**: recusa de papel não decide nem consome.
- **Sem lista**, a decisão é aceita e gravada com `RoleChecked = false`. A escolha é deliberada: recusar
  toda decisão quando ninguém configurou a lista travaria o produto; e a alternativa — aceitar em silêncio —
  esconderia que não houve verificação. `RoleChecked` transforma a ausência de checagem em **dado visível**.
- A lista é **isolada por organização** (não vaza para outra) e **persiste** ao reinício: uma checagem de
  papel que desaparece em silêncio é pior que não existir.

## 4. Relógio explícito (lição de E-143)

`FindApproved` e `ListApprovals` passaram a receber `now` em vez de ler o relógio da parede lá dentro. Sem
isso, o prazo só seria testável com espera real — e teste que depende do relógio da máquina é o que produziu
o defeito de E-142 (o caso que falhava 3 s antes da virada da hora). Com o instante como parâmetro, o teste
decide no passado e prova o vencimento **de forma determinística**.

## 5. Erros meus nesta rodada (registrados)

1. **`404` que parecia defeito de rota era defeito do meu teste**: o helper `serviceAgentRequest` do pacote
   sempre usa `POST`, e a rota nova é `PUT`. Cheguei a verificar as rotas registradas (estavam lá) antes de
   olhar o próprio auxiliar. Lição: quando o teste discorda do sistema, **confira o instrumento primeiro**.
2. **`print` do script não é o arquivo**: meu Python imprimiu `erro novo: True` lendo a **string em memória**,
   enquanto a declaração não estava no arquivo — e só o **compilador** pegou. É a mesma família de "medir o
   artefato, não o resumo do instrumento": depois de escrever por script, **releia o disco ou compile**.
3. **Outros 2 `misspell`** por falso positivo do dicionário pt-BR (`validade` → *validated*), reescritos para
   **vigência** — nunca `//nolint`, conforme a regra da missão.
4. **Here-string do PowerShell comeu aspas de novo**; resolvido usando a ferramenta de edição para texto com
   aspas, como em rodadas anteriores.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run TestServiceAgent` | **51/51 PASS** (7 novos: 6 de política + 1 de rota) |
| `go test ./internal/agent/ -run TestServiceAgentApproval -count=10` | **ok 11,838s** (sem flake) |
| `go test ./internal/agent/ -count=1` | **ok 58,769s** |
| `go test ./server/ -count=1` | **ok 20,582s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agent 20,649s · server 6,171s · exit 0) |
| `gofmt -l internal/agent server` | **sem saída** |

Os testes novos: vencida **não autoriza** e ainda impede escrever no CRM; a vencida fica **marcada** e o
pedido seguinte é **outro**; dentro do prazo executa; **papel fora da lista é recusado** sem decidir o
pedido; **sem lista** a decisão sai com `RoleChecked = false`; `ApprovalTTL = 0` desliga o prazo e a
aprovação continua valendo; `Consume` **recusa vencida mesmo aprovada**; e a lista de aprovadores é
**isolada e durável**, preservada quando aprovações são gravadas.

## 7. Limites declarados

- **Sem quórum**: não existe dupla aprovação para valores altos; uma decisão de um ator autorizado basta.
- **Mudar a lista de aprovadores NÃO entra na trilha de auditoria**: a alteração exige ator, mas não gera
  registro na trilha durável das ferramentas. É uma lacuna real e está aqui declarada.
- **O prazo é do store, não por ferramenta**: não há TTL diferente para escrita de maior risco.
- **Sem notificação**: o pedido pendente aparece na listagem, mas nada avisa um humano (sem push/e-mail).
- **A impressão digital cobre apenas os campos listados**; parâmetros fora dela não participam da decisão.
