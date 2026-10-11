# Estágio 35 — `knowledge.search` ligado: nenhuma ferramenta indisponível (E-152)

## 1. O que esta rodada fecha

Era a **última** ferramenta sem executor. Ela agora busca na **memória governada** (estágio 15) e o registro
de executores fica **completo**: um teste novo monta o `Runtime` de verdade e exige que
`UnavailableTools()` esteja **vazio**. Se algum dia alguém declarar capacidade sem executor, esse teste
quebra — que é exatamente o defeito que esta série de rodadas vem corrigindo.

## 2. O escopo é a decisão crítica — e é fail-closed

| Regra | Por quê |
| --- | --- |
| Só projetos **cujo dono é a organização da sessão** | a memória é indexada por projeto, e o projeto carrega `OrganizationID` |
| Projeto **SEM organização fica FORA** | projeto sem dono seria pesquisável por **todos** os tenants |
| Sem projeto da organização → "nenhum projeto de conhecimento desta organização" | resposta honesta, não busca vazia fingindo ausência de conteúdo |
| Sem consulta → **falha** explicando o que falta | busca sem texto não é busca |

O teste que importa: com uma memória em `org-a` e um "segredo" em `org-b`, a busca de `org-a` traz o
conhecimento dela e **nenhum** trecho da outra. Um segundo teste prova que memória em projeto **sem**
organização **não** aparece.

## 3. O achado que só a execução revelou (sondar, não supor)

A primeira execução dos testes voltou **vazia** — com a memória presente e a consulta contendo palavras dela.
Investiguei o motor em vez de supor: `SearchMemoriesContext` faz **busca semântica quando há embedder** e
**substring quando não há**. Sem embedder configurado, a frase inteira de um atendimento
("qual é a política de troca?") **não é substring** de nada — então a resposta era "nenhuma memória" mesmo
havendo memória.

Correção: a ferramenta tenta **a frase inteira primeiro** (é o insumo certo quando há embedder) e depois até
**cinco termos** com quatro letras ou mais (é o que funciona no modo lexical). O teto de termos é
deliberado: sem ele, uma mensagem longa de terceiro viraria varredura da base inteira.

Sem esse passo, a ferramenta pareceria pronta e devolveria vazio em produção — o pior desfecho possível.

## 4. Procedência declarada no resultado

O resumo diz **"dado, não instrução"** e de qual projeto veio. Conteúdo de base de conhecimento é
**informação**, nunca ordem: um texto envenenado na base não pode se passar por instrução do sistema. Cada
trecho é limitado a 140 caracteres e o total a três trechos.

## 5. Erros meus nesta rodada (registrados)

1. **Supus que a busca casaria a frase** — o próprio teste refutou a suposição, e foi a investigação do motor
   (não um palpite novo) que explicou o vazio. É a lição já institucionalizada: hipótese plausível precisa ser
   **sondada**.
2. **`NewContextStore` devolve dois valores** (store e erro) — o compilador pegou.
3. **Faltou o import de `path/filepath`** no arquivo de teste — o compilador pegou.
4. **Mais 2 falsos positivos de `misspell`** (`correspondente` → *correspondence*, `confidencial` →
   *confidential*), reescritos — nunca `//nolint`.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run 'TestServiceAgent|TestRuntimeServiceAgents'` | **67/67 PASS** (6 novos) |
| `go test ./internal/agent/ -count=1` | **ok 64,279s** |
| `go test ./server/ -count=1` | **ok 21,395s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agent 22,421s · server 5,799s · exit 0) |
| `gofmt -l internal/agent server` | **sem saída** |

Os testes novos: busca **escopada por organização** (sem vazamento); projeto **sem organização** ignorado;
**honestidade** quando não há resultado; **falha** sem consulta; disponibilidade **segue o registro** (sem
store, indisponível); e **completude** do registro no `Runtime` real.

## 7. Limites declarados

- **A busca é lexical por padrão**: sem embedder configurado o motor casa substring, então a qualidade depende
  do vocabulário. Busca **semântica** exige embedder — **não** está configurada, e não a declaramos ativa.
- **O resultado entra na trilha de auditoria da própria organização**: o conteúdo da base é da organização,
  então não há vazamento entre tenants, mas quem audita vê o trecho consultado.
- **Sem ranqueamento entre projetos**: a ordem dos projetos decide quais trechos entram (limite de 3).
- **A consulta vem do texto do contato** — dado de terceiro usado como consulta (nunca como instrução), com
  teto de termos para não virar varredura.
- **A base é a memória do Hades**: não há indexação de arquivos externos nem fonte corporativa além dela.
