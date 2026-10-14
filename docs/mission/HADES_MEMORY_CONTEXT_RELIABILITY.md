# HADES — Confiabilidade de Contexto e Memória (complemento V2.2)

**Escopo:** complemento transversal da missão MASTER V2.2 (§9 do
`HADES_V2_2_COMPLEMENTO_MEMORIA_COMPACTACAO_UNIVERSAL.txt`). **Não** reclassifica os
estágios 1–47 nem substitui o Estágio 15, que continua `COMPLETED_VERIFIED`.

**Estado do complemento:** `IN_PROGRESS` — **não homologável** (§10). A peça central
existe e está ligada; **o efeito sobre as mensagens não existe ainda**.

**Commit medido:** a medição mais recente deste documento refere-se a `d2d580367`
(branch `fix/checker-relogio`, PR #438). Medições anteriores são históricas.

---

## 1. O que existe, e o que está **ligado**

| Capacidade | Estado | Evidência |
| --- | --- | --- |
| Compactação no protocolo **Responses/Codex** | `IMPLEMENTADO_E_TESTADO` (35 testes) | `server/responses_compact.go` (248 linhas), `openai/responses_compact.go` |
| Medição de tokens **rotulada** (`Exata=false`) | `TESTED` **e ligada** | `internal/agent/memory_summary_envelope.go`, chamada no funil |
| Decisão por **orçamento** com motivo | `TESTED` **e ligada** | `server/chat_context_budget.go` + `chat_context.go:146` |
| Envelope versionado (versão, hash, fontes, validação) | `TESTED` **e DESLIGADO** | `memory_summary_envelope.go`; só usado no próprio teste |
| Gravar **histórico bruto antes** do resumo | **AUSENTE** | nenhum snapshot de conversa por faixa de mensagens |
| Substituir mensagens pelo resumo | **AUSENTE** | o funil decide e regista; **não altera** `request.Messages` |
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
