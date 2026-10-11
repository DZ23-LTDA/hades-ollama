# Estágio 35 — rota de operador para o re-embed (E-159)

## 1. A lacuna declarada, fechada

E-158 deixou explícito: *"o re-embed não tem rota de operador ainda (só chamada programática)"*. Sem gatilho,
a detecção que **declara** que parte do conhecimento ficou fora do alcance semântico seria uma promessa sem
remédio acionável.

```
POST /api/agent/projects/:id/memories/reembed?limit=N
{ "project_id": "...", "reembedded": 2, "failed": 0, "skipped": 1, "remaining": 0, "limit": 200 }
```

| Decisão | Por quê |
| --- | --- |
| Escopo pelo **projeto** (`projectForRequest`) | mesma checagem das outras rotas de memória — outra organização não opera projeto alheio |
| `limit` com teto de **200** por chamada | manutenção não pode virar requisição que ninguém consegue terminar |
| `remaining` na resposta | operação limitada que não diz o resto **esconde trabalho pela metade** de quem operou |
| `limit` vigente na resposta | sem ele, `remaining` seria um número sem referência |
| Sem embedder → **falha declarada** | não há espaço de destino; fingir sucesso seria pior que recusar |

## 2. A semântica do contador (uma decisão que o teste forçou)

`CountMemoriesOutsideVectorSpace` contava só vetor com **carimbo diferente**. Ao testar o limite, percebi que
memória **sem vetor nenhum** (gravada antes de existir embedder) também está fora do alcance **semântico** — e
é justamente o que o re-embed resolve. O contador passou a incluir os dois casos, que é o significado
**acionável** de "fora do espaço".

## 3. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run 'TestServiceAgent\|TestRuntimeServiceAgents\|TestAgentMemory'` | **94/94 PASS** (4 novos de rota) |
| `go test ./internal/agent/ -count=1` | **ok 64,729s** |
| `go test ./server/ -count=1` | **ok 19,468s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agente 20,464s · servidor 6,350s) |
| `gofmt -l internal/agent server` | **sem saída** |

Testes da rota: recálculo real com contadores; **limite respeitado** com `remaining` (e o resto continua fora
do espaço); **outra organização recusada sem vazar conteúdo**; e **sem embedder a falha é declarada**.

## 4. Erros meus nesta rodada (registrados)

1. **Apaguei o corpo de `searchMemories`.** Fiz um `edit` que substituía oito linhas por uma só, achando que
   estava inserindo antes da função. Reparei em seguida, mas a lição é de método: ao inserir código perto de
   outra função, ancorar na **linha de registro/única**, nunca num bloco que se pretende manter.
2. **Inventei API que não existe** (`ginParamLike`, `.toGin()`) em vez de olhar como os testes vizinhos
   montam parâmetros (`gin.Params{{Key: "id", ...}}`). O compilador pegou — mas olhar primeiro seria mais
   barato.
3. **Ordem de fixture invertida outra vez**: gravei as memórias **depois** de ligar o embedder, então todas já
   estavam no espaço atual e o teste viu `skipped`. É a **segunda vez** nesta sequência (E-157 teve a mesma
   armadilha) — está registrado como risco recorrente de quem escreve o teste, não do código.
4. **`gofmt` esquecido** depois de editar por script → lint acusou `gofmt: 1`.
5. **Bloco de ambiente incompleto** no lint: faltando `TMP`/`TEMP`, o golangci-lint falhou com
   `context loading failed: no go files to analyze`. É **erro de ambiente, não de código** — a lição antiga
   (APPDATA/LOCALAPPDATA) ganhou um terceiro campo obrigatório.

## 5. Limites declarados

- **Síncrono**: a requisição espera todos os embeddings daquela chamada; não há job em segundo plano nem
  streaming de progresso.
- **Limitado a 200 por chamada** e **por projeto**: não existe varredura de organização inteira.
- **Chamadas concorrentes no mesmo projeto** não são coordenadas além do mutex do store — a última gravação
  vence no arquivo (o conteúdo não se perde, mas o resultado relatado pode ser de uma execução anterior).
- **Sem estimativa prévia**: o operador não sabe quantas memórias serão recalculadas antes de chamar; descobre
  pelo `remaining`.
