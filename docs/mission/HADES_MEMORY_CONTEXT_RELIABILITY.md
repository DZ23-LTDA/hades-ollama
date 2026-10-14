# HADES — Confiabilidade de Contexto e Memória (complemento V2.2)

**Escopo:** complemento transversal da missão MASTER V2.2 (§9 do
`HADES_V2_2_COMPLEMENTO_MEMORIA_COMPACTACAO_UNIVERSAL.txt`). **Não** reclassifica os
estágios 1–47 nem substitui o Estágio 15, que continua `COMPLETED_VERIFIED`.

**Estado do complemento:** `IN_PROGRESS` — **não homologável** (§10). A peça central
existe e está ligada; **o efeito sobre as mensagens não existe ainda**.

**Commit medido:** a medição mais recente deste documento refere-se a `f9ddaa75c`
(branch `fix/checker-relogio`, PR #438). Medições anteriores são históricas.

**Inventário de código construído e testado desde a primeira versão deste documento**
(cada um com testes próprios; **nenhum ligado ao funil**, ver §4):

| commit | peça | o que prova |
| --- | --- | --- |
| `68ab4b1a3` | histórico bruto append-only + cadeia de hashes | 5 testes: cadeia, retomada após reabrir, deteção de adulteração, linha corrompida recusada |
| `2f58f86ed` | plano (recentes intactas × faixa a resumir) | 4 testes: a instrução corrente nunca é resumida |
| `807db4bdc` | efeito (bruto ANTES do resumo) | 4 testes: §7-D — falha na gravação não substitui nada |
| `38a7d205b` | digest determinístico da faixa | 3 testes: fidelidade, corte declarado, corte nunca a meio de uma mensagem |
| `0ff52217c` | fluxo ponta a ponta | 3 testes: **recuperação provada** (fechar → reabrir → recuperar o material coberto) |
| `f9ddaa75c` | `ContextStore.Root()` | 3 testes: habilita o histórico dentro do escopo já governado |
| `0ca5cae62` | caminho do histórico com recusa de travessia | 3 testes: 12 casos de ataque recusados + identificadores legítimos com ponto aceites |
| `18e413433` | recuperação pelo **caminho derivado** | 2 testes: material recuperado no caminho derivado e isolamento entre projetos |
| `9ec100db3` | compactação no chat como unidade testável | 3 testes: 4 recusas devolvem as **originais**; sem efeito antes da decisão |
| `7cba79ee9` | **LIGAÇÃO** ao funil com a raiz do `ContextStore` | `vet` 0 · funil verde · 4 sítios (o 4.º só apareceu no compilador) |
| `856bfec6b` | **compactação PONTA A PONTA por HTTP** | 2 testes: corpo no handler com resumo + bruto recuperável em disco; **sem teto, nada muda** |

**A ligação EXISTE desde `7cba79ee9`** — esta secção substitui a afirmação anterior de que
era `AUSENTE`. O que continua a decidir o §10 é a **cobertura parcial**: a compactação e a
recuperação estão provadas ponta a ponta; o **handoff entre agentes e modelos (§4)
continua AUSENTE**, a retomada de **missões** após interrupção não está medida, e a memória
governada é a do Estágio 15 (que não tem `OrganizationID`/`WorkspaceID` no registo).

E a compactação é **opt-in**: sem `OLLAMA_AGENT_CHAT_CONTEXT_TOKENS` declarado a decisão é
«não declarado» e o corpo do pedido segue **igual** ao que o cliente enviou — medido em
`TestChatContextNaoCompactaSemTetoDeclarado`, que também exige que **nenhum histórico seja
criado sem compactação**.

**O que continua `AUSENTE` apesar destas peças:** a **ligação** ao funil do chat. O
`FluxoDaCompactacao` não é chamado por ninguém em produção, e o chat continua a enviar as
mensagens tal e qual. As duas linhas de decisão por orçamento (`4542ea468`) correm, mas
**por omissão não mudam nada** (sem teto declarado, a decisão é «não declarado»).

---

## 1. O que existe, e o que está **ligado**

| Capacidade | Estado | Evidência |
| --- | --- | --- |
| Compactação no protocolo **Responses/Codex** | `IMPLEMENTADO_E_TESTADO` (35 testes) | `server/responses_compact.go` (248 linhas), `openai/responses_compact.go` |
| Medição de tokens **rotulada** (`Exata=false`) | `TESTED` **e ligada** | `internal/agent/memory_summary_envelope.go`, chamada no funil |
| Decisão por **orçamento** com motivo | `TESTED` **e ligada** | `server/chat_context_budget.go` + `chat_context.go:146` |
| Envelope versionado (versão, hash, fontes, validação) | `TESTED` **e DESLIGADO** | `memory_summary_envelope.go`; só usado no próprio teste |
| Gravar **histórico bruto antes** do resumo | `TESTED` **e DESLIGADO** | `historico_bruto.go` (append-only + cadeia + retomada, 5 testes) e `compactacao_aplicar.go` (§7-D provado); nenhum chamador em produção |
| Substituir mensagens pelo resumo | `TESTED` **e DESLIGADO** | `compactacao_plano.go` + `compactacao_digest.go` + `compactacao_fluxo.go` (recuperação ponta a ponta provada); o funil decide e regista, mas **não altera** `request.Messages` |
| Ligação do fluxo ao funil do chat | `TESTED` **e LIGADA** (E-584) | `withAgentChatContext(raizDoHistorico)` + `CompactarMensagensDoChat`; raiz de `a.context.Root()`; provada ponta a ponta por HTTP em `856bfec6b` |
| Memória **governada por organização** (8 rotas) | **VERIFICADO — já coberto** | a governação é TRANSITIVA (`Memory` → `ProjectID` → `Project.OrganizationID`, `context.go:97/431` + `agent_routes.go:2145/2148`); escrita em `agent_object_scope_test.go:71` (projectB) e governação em `project_memories_test.go:85/129`, `memory_export_delete_routes_test.go:72`. **Nenhum teste novo era devido** — a "lacuna" era falsa |
| Handoff universal (§4) | **AUSENTE** | `BuildHandoff|MissionHandoff|HandoffContract` → 0 ocorrências |
| Camadas de memória A–E (§3) | **PARCIAL** | `Memory.Kind` é texto livre; sem camada, validade ou sensibilidade por registo |
| Busca híbrida / rerank / contradição (§3) | **AUSENTE** | `RetrieveRelevant` é **ou** cosseno **ou** literal, mutuamente exclusivos |
| UI de contexto e memória (§5) | **AUSENTE / declarada indisponível** | `Settings.tsx:893-900` mostra «Indisponível» |
| Sincronização multiplataforma (§6) | `NAO_MEDIDO` | exige dispositivos reais |

**Limites do Estágio 15, reverificados no código de hoje — todos continuam verdadeiros:**
resumo padrão é digest determinístico; embedder é opcional e o sistema não finge
semântica; criptografia é opt-in com chave do operador (chave perdida = memória
irrecuperável); integração externa opcional ausente; escopo de workspace é derivado do
projeto. **Nenhum foi reclassificado.**

---

## 2. Evidências desta fatia (commits em `fix/checker-relogio`)

| Commit | Entrega | Prova |
| --- | --- | --- |
| `6d1ff46a7` | envelope + gatilho por orçamento | 6 testes · Estágio 15 preservado (5/5) |
| `4c23e8f22` | orçamento no funil do chat | 4 testes · 6 testes do funil preservados |
| `18f25ec63` | catraca do pré-filtro DLP | 3 testes · 19 amostras + caixa alta + controlo |
| `7402ad345` | F4: `gho_/ghu_/ghs_/ghr_` | 30 testes DLP PASS |
| `8f7cf227b` | F5: `DeleteProject` disco-antes-da-memória | caminho normal PASS |
| `27a8be64d` | prova do caminho de falha do F5 | **controlo negativo executado** |
| `81cffdb47` | F1: orquestração valida projeto×organização | vet 0 · testes de orquestração PASS |
| `4542ea468` | **liga** o gatilho no funil (fecha P0-1) | teste do middleware executa as linhas novas |
| `d2d580367` | prova que a decisão chega ao handler | **controlo negativo executado** |

**Controlos negativos executados** (o padrão que este repositório exige: um teste que
passe nos dois sentidos não prova nada):
- E-566: reposta a ordem antiga do `DeleteProject` → `FAIL "o projeto desapareceu da memória: []"`;
- E-569: removida a linha de ligação do gatilho → `FAIL "o middleware NÃO registou a decisão"`.

---

## 3. Defeitos declarados (deste trabalho, não escondidos)

1. **`4c23e8f22` tem o assunto errado.** O título no histórico é o parágrafo da lição de
   instrumento, não `feat(memoria): orcamento de contexto no funil do chat (E-562)` — o
   texto acrescentado com `AppendAllText` acabou como primeira linha. Corrigir exigiria
   reescrever história (**force-push, proibido**); a discrepância fica registada aqui.
2. **E-561 e E-562 nasceram desligadas** e foram apontadas pela auditoria independente;
   o P0-1 só fechou no `4542ea468`. O comentário de `chat_context_budget.go` afirmava uma
   ligação que não existia.
3. **Não há teste do comportamento de recusa do guard F1** — o guard corre, mas não está
   medido que recusa um projeto de outra organização.
4. **Não há contagem exata de tokens** — a estimativa é 4 caracteres/token e está
   rotulada; o tokenizer real existe noutro caminho (`server/prompt.go`).
5. **`encrypted_content` do protocolo de compactação não é cifrado** — é JSON canónico;
   o nome sugere proteção que não existe.
6. **CADÊNCIA DE PUSH INVALIDA A VALIDAÇÃO (defeito operacional, meu).** Empurrei ~19
   commits em rápida sucessão e o `concurrency: cancel-in-progress` dos workflows
   **cancelou os runs intermédios**: `test` #1116 (E-575), `agentic-quality` #1892/#1895,
   `class-a-plus-integrity` #1990. Um run `cancelled` **não é verde nem vermelho — é
   `NAO_MEDIDO`**, e vários commits desta série nunca foram totalmente validados. Pior: o
   `pr-gate` recusa passar quando um gate profundo está `cancelled` (E-541), logo o PR
   pode estar bloqueado **pela cadência de push, não pelo código**. A regra já estava
   escrita nesta missão («um push por run; ler o run anterior antes de empurrar») e foi
   violada: ***a lição escrita não é a lição aplicada.*** Correcção em vigor: nenhum push
   sem ler o run anterior.

---

## 4. Aberto, por prioridade

| Prioridade | Item |
| --- | --- |
| **P0** | Compactação real: gravar histórico bruto imutável → resumo versionado (hash, faixa, orçamento) → substituir mensagens, com **teste de recuperação** |
| **P0** | `race (macos-latest)` excede 600 s — F2: data race em `swarm.go:313-372` × `:478-479` |
| **P1** | Teste do guard F1 · ligar o envelope a `/memories/summary` · histórico bruto antes do resumo |
| **P1** | Idempotência/transação da compactação; DLP e `LOCAL_ONLY` no caminho de compactação |
| **P2** | `Memory` sem `OrganizationID`/`WorkspaceID` · busca híbrida · §4 handoff · UI de memória · §6 dispositivos |

---

## 5. Regra de homologação (§10)

A funcionalidade **só** estará `COMPLETED_VERIFIED` quando compactação automática,
memória compartilhada governada, handoff, recuperação de fontes e retomada de missões
tiverem **provas ponta a ponta**, com persistência íntegra e autorização em todas as
rotas. **Falhas e dependências externas permanecem expressas, nunca disfarçadas.**
Enquanto o P0 da compactação real estiver aberto, este complemento é `IN_PROGRESS`.

## Decisões tomadas e ainda NÃO executadas

Registadas aqui de propósito: esta missão já perdeu decisões que existiam só na conversa
quando o contexto foi compactado. Uma decisão que não está em disco não é uma decisão.

### D-1. Envelope versionado no `/memories/summary` — forma ADITIVA, nunca breaking
`agent_routes.go:2322-2333` devolve `MemorySummary` **directo** (`c.JSON(200, summary)`) e o
envelope do E-561 (`ConstruirEnvelope`: versão, hash, fontes cobertas, orçamento, validação)
**existe e não é usado**. Escolha fixada:
- **(b) ADITIVA** — juntar os campos do envelope AO `MemorySummary`. Consumidores antigos
  continuam a ler o que liam. **É este o caminho.**
- (a) aninhar em `{"summary":…,"envelope":…}` está **RECUSADA**: muda a forma na rede sem
  necessidade, quebra `memory_summary_routes_test.go` e clientes — uma alteração breaking
  desnecessária não é progresso, é um incidente à espera de acontecer.
- Cuidado obrigatório: `MemorySummary` é **Estágio 15, a preservar**. *Preservar não é não
  tocar: é não lhe mudar o significado.* O teste tem de provar que os campos antigos saem
  **iguais** (nome, tipo e conteúdo) além dos novos.

### D-2. §4 handoff — o que falta é a INTEGRAÇÃO, não o contrato
`internal/agent/handoff.go` (E-588) tem contrato + `Validar` + `ParaContexto` provados, e
**nenhum chamador em produção**. Passo seguinte: ligar o `Handoff` a um caminho real de troca
entre agentes/modelos (Mission Control / Cowork) e prová-lo ponta a ponta, como se fez com a
compactação (E-585). Enquanto isso não existir, o §4 está **contrato-feito / integração-AUSENTE**
— o mesmo defeito «apenas adaptador» que esta missão persegue, aqui declarado em vez de escondido.

### D-3. `race (macos-latest)` — a medição que decide o F2
O E-589 fez `persistLocked` tomar `o.taskMu` (o mesmo lock das escritas em `job.Tasks`).
**Não está provado que fosse essa a causa do vermelho**: `Cancel` de um job a correr já
evitava persistir por causa dessa corrida (`swarm.go:454-458`), logo o autor conhecia-a e
fechara o caminho que conhecia. O log do próximo `race (macos)` decide:
- `WARNING: DATA RACE` ⇒ a corrida era real e o E-589 mirou certo;
- `panic: test timed out after 10m0s` ⇒ é **TEMPO**, o F2 não era a causa, e a frente passa a
  ser a duração do pacote `internal/agent` (fatia do CI medida localmente: 241,9–288,1 s;
  limite de 600 s por binário).

### D-4. Consumidor do handoff — desenho CORRIGIDO por leitura, não por suposição
Eu tinha proposto (D-2) injectar o `Handoff` no corpo do `POST /missions/:id/run`. **Está
errado**: `runMission` (`agent_routes.go:3863-3882`) **não aceita corpo** — resolve a missão
por `missionForRequest`, confere a versão, chama `EnqueueMission(id)` e devolve `202` com o
job. O plano/objetivo vivem no registo da missão no runtime, não no pedido.

Desenho corrigido, e é MENOR do que o proposto antes:
- `POST /missions/:id/handoff` — anexa um `Handoff` ao registo da missão, **validado com
  `Validar()` (E-588)**: um handoff inválido é RECUSADO com o motivo, não guardado.
- `GET /missions/:id/handoff` — exporta o handoff, construído com `HandoffDoJob` (E-593) a
  partir do job de orquestração.
- Ambos ADITIVOS, sob o guard de missão que já existe (`missionForRequest`) — nenhum caminho
  novo de autorização.

LIMITE DECLARADO ANTES DE ESCREVER: isto torna o handoff **transportável dentro do Hades**
(gravado + recuperável sob governação). **Injectá-lo no contexto do modelo durante a
execução continua a ser o passo seguinte** — e a diferença fica dita agora, não descoberta
depois de alguém ler «handoff implementado».

Superfícies adjacentes que NÃO são este conceito, para não serem confundidas: `resumeCompany`
e `resumeCompanyAgent` (`:762`/`:767`) são retoma de empresa/agente; `Last-Event-ID`
(`:2966`) é retoma de stream SSE.

### D-4 FECHADO (§4 completo) — o que fica dito para quem ler isto depois

A integração que o D-4 deixava em aberto está **feita e provada**, em quatro passos que se
podem seguir no histórico:

| commit | peça | prova |
| --- | --- | --- |
| `5d3633d20` (E-588) | contrato: `Validar` · `ParaContexto` · marcador de dado | 5 testes |
| `36db38979` (E-593) | produtor da **orquestração** (`HandoffDoJob`) | 5 testes |
| `3bf2fc3a5` (E-595) | produtor da **missão** (`HandoffDaMissao`), com autorização REAL e só a concedida | 5 testes |
| `30769e3a4` (E-596) | o bootstrap **aceita** o bloco (documento **e** preâmbulo) | 3 testes |
| `eaff74969` (E-597) | preâmbulo **REDIGIDO** (sem identificadores nem caminhos) | 3 testes |
| `c6171257d` (E-598) | **a ligação**: builder usa o redigido · `missionContext` deriva e passa | **prova conjunta**: `server` = 0 **e** `internal/agent` = 0 |

**E um defeito que só apareceu porque um teste já existia**: ligar o bloco completo ao
preâmbulo fez `mission_context_test.go:118` falhar com a linha exacta —
`preamble leaked "mis_…"`, `"prj_…"` e **o caminho do workspace**. *O teste estava certo e a
ligação estava errada.* Identificador interno e caminho local não são conteúdo de trabalho:
são topologia do servidor. Daí o `ParaPreambulo` por **lista branca** — *uma lista negra
esquece-se de um campo e vaza-o; uma lista branca esquece-se de um campo e omite-o.*

**O QUE CONTINUA FORA, e não se esconde atrás de «§4 feito»**: o contrato chega ao contexto
das **missões**; no **chat** ele só chegaria se o pedido identificasse a missão — e hoje o
chat identifica PROJETO, não missão (`chat_context.go:55-63`, sem `MissionID`). Fazer isso é
**decisão de contrato de API**, não detalhe de implementação.
