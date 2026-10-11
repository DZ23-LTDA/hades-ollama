# Estágio 27 — consentimento, opt-out e limites de contato

## 1. O buraco que existia

| Caminho de envio | Controles que existiam | Consentimento |
| --- | --- | --- |
| **WhatsApp** | allowlist por número + **aprovação humana por mensagem** + DLP | **AUSENTE** |
| **Telegram** | **só DLP** — nenhuma checagem de autorização de contato | **AUSENTE** |

Allowlist autoriza o **número**; aprovação autoriza a **mensagem**. Nenhuma das duas
registra **opt-out** nem **prazo**. E o Telegram não tinha nem isso. O estágio 27 exige
*"consentimento, opt-out, limites e proteção de dados"*, e a diretriz do projeto proíbe
contato sem consentimento.

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| **FALHA FECHADA**: contato desconhecido **não** é contatável | o padrão do sistema é "não"; assumir consentimento por omissão é exatamente a violação que se quer evitar |
| **Registro ausente NEGA o envio** | caminho não instrumentado **não pode virar caminho livre** |
| **Opt-out é decisão final** e exige **motivo** | descadastro sem justificativa não é auditável, e não pode ser revertido por fluxo automático |
| **Prazo de validade**: vencido volta a ser ausência de consentimento | consentimento velho não vale para sempre |
| **Trilha**: quem concedeu, quem revogou, quando, por quê | sem isso não há como provar conformidade nem responder a um titular |
| **Escopo por organização E por canal** | consentir no WhatsApp **não** autoriza Telegram nem outro tenant |
| **Sentinela PURO** (sem embrulho) | os caminhos de canal comparam erros por identidade (`==`); erro embrulhado faria o bloqueio parecer "outro erro" |

## 3. Os DOIS caminhos, não um

O controle é aplicado no **Telegram** e no **WhatsApp**, pela **mesma** função de
decisão (`authorizedContact`). Um controle que vale num caminho só é o defeito de
**irmão esquecido** que já apareceu quatro vezes neste projeto — por isso há teste de
comportamento cobrindo os dois.

## 4. Dois bugs meus, achados pelos próprios testes

1. **Escopo vazio tornava o WhatsApp inutilizável.** O gateway checava o consentimento
   sob `runtime.organizationScope`, que é **vazio** no modo local; o registro era
   gravado sob a mesma chave vazia em outro ponto e as chaves **nunca casariam** — ou
   pior, casariam por acidente. Corrigido com `consentScope`, que normaliza vazio para
   a **organização local**, a mesma convenção do HTTP e do supervisor.
2. **Sentinela embrulhado quebrava a comparação por identidade.** `AuthorizeContact`
   devolvia `fmt.Errorf("%w: ...")`; o código dos canais compara com `==`, então o
   bloqueio apareceria como erro desconhecido. Passou a devolver o sentinela puro.

E uma **expectativa minha que estava errada**: eu esperava que conceder consentimento
com organização vazia fosse **inválido**. Com a normalização (correta), vazio é a
organização local. Corrigi o **teste**, não o código.

## 5. Verificação

- **6 testes de motor do registro**: falha fechada (contato desconhecido e registro
  `nil`) · opt-out bloqueia, exige motivo, guarda responsável e **preserva a trilha do
  consentimento original** · prazo vencido bloqueia e renovação libera · escopo por
  organização e por canal (e listagem escopada) · sobrevive ao reinício · normalização
  de escopo vazio com canal/contato ainda obrigatórios.
- **3 testes de enforcement**: no **Telegram**, contato sem consentimento **não gera
  nenhuma requisição** (servidor de teste conta chamadas), consentimento libera, opt-out
  volta a bloquear e consentimento de **outra organização** não libera · no **WhatsApp**,
  sem consentimento o **adaptador não é chamado** (contador), consentimento libera e
  opt-out bloqueia · um teste de comportamento garante que os **dois** caminhos usam a
  mesma decisão e que consentimento de um canal **não** vale no outro.
- **1 teste existente reforçado**: `TestWhatsAppOutboundRequiresAllowlistAndApproval`
  manteve todas as asserções (allowlist e aprovação) e ganhou o **terceiro controle** —
  allowlist + aprovação **sem** consentimento é negado, e opt-out bloqueia mesmo com as
  duas válidas.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** · suítes
  apenas com as **falhas ambientais conhecidas** (que passam no CI).

## 6. Limites declarados (o estágio NÃO está completo)

1. **Sem UI de consentimento**: as rotas HTTP de concessão/revogação **não** foram
   expostas nesta rodada; o registro é usado pelos caminhos de envio e pelo motor.
2. **Sem captura automática de opt-out**: uma mensagem recebida com "PARE" **não**
   revoga sozinha; a revogação é uma decisão registrada (evita que texto de terceiro
   altere consentimento — dado externo **não** é instrução privilegiada).
3. **Sem CRM nem histórico de conversa** (estágio 27 pede integração a CRM comum e
   continuidade humano/IA).
4. **Sem agentes comerciais** (consultor, qualificador, propostas, pós-venda).
5. **Sem retenção/anonimização de dado pessoal** ligada ao consentimento.

Por isso o estágio 27 permanece **`IN_PROGRESS`** com capacidade **`PARCIAL`**.
