# Estágio 35 — busca semântica provada e reserva declarada (E-157)

## 1. O defeito real (mais grave que a ausência de recurso)

O limite declarado era *"a busca é lexical por padrão; a semântica exige embedder"*. Investigando a fiação,
o embedder **já era configurável** (`OLLAMA_AGENT_EMBED_MODEL`, ligado em `server/agent_routes.go:393`). O
defeito era outro: **degradação silenciosa**.

`knowledge.search` chamava `SearchMemories`, que **engole o erro** do embedder. Com o embedder configurado e
o serviço fora do ar, a ferramenta respondia *"nenhuma memória encontrada para esta organização"* — o agente
parecia ignorante e a configuração quebrada ficava invisível.

## 2. O que mudou

| Peça | Comportamento |
| --- | --- |
| `ErrEmbedderUnavailable` | classe de **transporte**: serviço fora, modelo não carregado |
| `ErrEmbedderInvalid` | classe de **defeito**: sem cliente/modelo, vetor vazio, resposta ilegível |
| `SearchMemoriesContext` + `errors.Is` | o chamador **distingue** as duas: reserva declarada x falha |
| `SearchMemoriesLexical` | caminho literal **explícito**, reaproveitando a MESMA pontuação |
| Resposta | com reserva: `[busca LITERAL: embedder indisponível]` — quem lê sabe que o caminho semântico não rodou |

**Sem duplicação**: eu comecei a escrever um embedder HTTP próprio e o compilador acusou
`OllamaEmbedder redeclared` — o projeto já tinha um, com o cliente oficial. Apaguei o meu e mantive só o que
faltava (as classes de erro). Reaproveitar era a resposta certa, e o compilador apontou antes de virar dívida.

## 3. Prova de que a busca é SEMÂNTICA (e não substring disfarçada)

O teste grava a memória *"aceitamos troca em até 30 dias"* e consulta *"prazo para devolver produto"* — **sem
nenhuma palavra em comum**. O embedder de teste mapeia os dois textos para o mesmo vetor. A resposta traz o
conteúdo: só o caminho vetorial explica isso. Em seguida:

| Cenário | Resultado exigido | Resultado |
| --- | --- | --- |
| Embedder funcionando | acha sem palavra em comum e **sem** marcador | PASS |
| Embedder caindo na consulta | acha por literal e **declara** `[busca LITERAL: embedder indisponível]` | PASS |
| Embedder com resposta inválida | ferramenta **FALHA** e diz a classe — nada de reserva silenciosa | PASS |
| Sem embedder | busca literal normal, **sem** marcador de reserva | PASS |

## 4. Medição honesta: 131 s que não eram regressão

A bateria mediu a suíte do agente em **131,400 s** — o dobro do patamar de ~64 s. Em vez de aceitar, medi por
teste: **62,586 s**, nenhum teste ≥20 s, soma dos PASS 58,6 s. A diferença era **carga do host** (o contêiner
da paridade rodando ao mesmo tempo), não o meu diff. Fica a lição operacional: **medição de tempo feita com o
gate de contêiner rodando em paralelo não é comparável** — e é a mesma classe de contenção que produziu a
falha não reproduzida de E-154 (hipótese, não causa provada).

## 5. Erros meus nesta rodada (registrados)

1. **Duplicação**: comecei a escrever `OllamaEmbedder` sem procurar o existente; o compilador pegou
   (`redeclared`).
2. **Aspas quebradas por script**: meu gerador trocou `"` por `'` e produziu Go inválido (`illegal rune
   literal`) — compilador pegou.
3. **Sombreamento**: o parâmetro do executor se chamava `context` e sombreava o pacote `context`; renomeado.
4. **Contexto inexistente**: `ServiceAgentToolRequest` **não tem** contexto de requisição; declarei um prazo
   próprio (`serviceAgentKnowledgeSearchTimeout = 20s`) em vez de inventar um campo.
5. **Script que "passou" sem aplicar**: meu patch em Python imprimiu sucesso e o arquivo continuou o
   antigo — só a leitura do artefato mostrou. É a repetição de uma lição já registrada: **nunca confiar no
   print do script; ler o arquivo**.

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run 'TestServiceAgent\|TestRuntimeServiceAgents'` | **86/86 PASS** (4 novos) |
| `go test ./internal/agent/ -count=1` | **ok 62,586s** (medido de novo; 131,400s era carga do host) |
| `go test ./server/ -count=1` | **ok 24,934s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agente 19,323s · servidor 6,174s) |
| `gofmt -l internal/agent server` | **sem saída** |

## 7. Limites declarados

- **A reserva só ajuda memória já gravada**: com o embedder configurado e fora do ar, **gravar** memória nova
  falha (`embed memory: ...`) — a falha é alta na escrita, e isso é correto: gravar sem vetor deixaria a
  memória invisível para a busca semântica sem ninguém saber.
- **Trocar o modelo de embedding não re-embeda o histórico**: as memórias antigas ficam no espaço vetorial
  antigo e a comparação por cosseno deixa de ser significativa. **Não há backfill** — é uma armadilha real e
  está declarada, não escondida.
- **O executor não tem contexto de requisição**: o prazo é fixo (20 s) e não cancelável pelo chamador.
- **A qualidade semântica é do modelo**: o teste usa um embedder determinístico; ele prova o CAMINHO, não a
  qualidade de um modelo real.
- **A busca segue escopada por projeto/organização** (E-152), com ou sem embedder.
