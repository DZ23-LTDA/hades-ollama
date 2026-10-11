# Estágio 35 — agentes de atendimento e vendas

## 1. O que foi entregue

Sete agentes especializados reutilizáveis — **Recepcionista, Vendas, Suporte, Redes
Sociais, Voz, Comércio e Supervisor** — que **usam** o que já existe: CRM, caixa de
entrada, registro de consentimento/DLP e o catálogo de ferramentas.

| Agente | Objetivo | Ferramentas declaradas |
| --- | --- | --- |
| Recepcionista IA | receber, identificar a necessidade e encaminhar | CRM, conversa, resposta, handoff |
| Agente de Vendas IA | qualificar, apresentar oferta e registrar o próximo passo | CRM, oportunidade, catálogo, resposta, handoff |
| Agente de Suporte IA | resolver dúvidas e falhas de primeiro nível com histórico | CRM, conversa, resposta, handoff, base de conhecimento |
| Agente de Redes Sociais IA | responder interações públicas com aprovação | CRM, rascunho social, resposta |
| Agente de Voz IA | atender chamadas com transferência para humano | CRM, resposta, handoff, transferência de voz |
| Agente de Comércio IA | acompanhar pedido, estoque e entrega | CRM, pedido, estoque, resposta, handoff |
| Supervisor de Atendimento IA | acompanhar filas, SLA e qualidade | métricas, atribuição, handoff, relatório da empresa |

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| **A PROCEDÊNCIA do texto é declarada** (`reply_source`) | sem modelo conectado o agente usa um **modelo determinístico** e **diz isso**. Apresentar template como "resposta da IA" seria **mentir para o usuário** |
| **Política antes do texto, em qualquer modelo** | trocar de modelo **não** pode remover consentimento nem DLP. Há teste em três modelos diferentes, incluindo um externo |
| **Sem consentimento o agente ESCALA** | a decisão correta não é "manda mesmo assim": é passar para uma pessoa, com o **motivo** registrado |
| **Pedido de humano escala imediatamente** | detectado por marcadores explícitos no texto do contato |
| **Ferramenta que o runtime não tem aparece BLOQUEADA** | em vez de ignorada em silêncio: o contexto do turno mostra `allowed_tools` e `blocked_tools` |
| **Contexto explícito e verificável** | contato do CRM, oportunidades, **histórico da conversa**, estado de consentimento, ferramentas e políticas ficam no turno — o que o modelo receberia está à vista |
| **A resposta entra pela MESMA caixa de entrada** | a política não depende do caminho: a inbox revalida o consentimento |
| **A pergunta de DLP é booleana** | "este texto pode sair?" é decisão de política, não falha de servidor — por isso virou `outboundTextAllowed` em vez de propagar um erro que ninguém consumiria |

## 3. Verificação

- **7 testes de motor**: catálogo com os **sete** agentes (ordem, objetivo, canais,
  ferramentas e regras de escalada) · turno usa **CRM real** (contato e oportunidade),
  **histórico** da conversa, consentimento declarado e procedência declarada ·
  **escala sem consentimento** e não deixa **nenhuma** mensagem sair · escala ao pedido
  de humano · **troca de modelo mantém política** (três modelos; depois do opt-out, escala
  em **qualquer** modelo) · ferramenta não registrada aparece **bloqueada** · escopo por
  organização e agente inexistente falha.
- **2 testes de rota**: catálogo de sete agentes · turno com contexto real, procedência e
  políticas declaradas, agente inexistente e corpo inválido recusados · escalada
  respondendo **202 com motivo** e **sem** mensagem.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** · suítes
  apenas com as **falhas ambientais conhecidas**.

## 4. Um ajuste de forma feito por causa do linter

O `nilerr` apontou o retorno `nil` no bloqueio de DLP. Em vez de silenciar a regra (ou de
propagar um erro que ninguém consumiria), **mudei a forma**: a política responde
**sim/não** (`outboundTextAllowed`), porque bloquear não é falha do servidor — é decisão
que vira escalada. O código ficou mais claro **e** o linter ficou limpo.

## 5. Limites declarados

1. **Sem geração por modelo**: o texto do turno é determinístico e **declarado como tal**.
   Ligar um modelo real é o próximo passo, e a procedência existe justamente para isso.
2. **Ferramentas são declaradas, não executadas**: o turno mostra quais o agente pode e
   quais estão bloqueadas; a execução real de cada ferramenta depende dos módulos de
   comércio, voz e conhecimento.
3. **Base de conhecimento ausente**: `knowledge.search` aparece bloqueada.
4. **Sem filas de distribuição por especialidade**: o agente é escolhido pelo chamador.
5. **Sem métricas de qualidade por agente** (satisfação, resolução na primeira resposta).
