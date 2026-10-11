# Estágio 35 — carimbo de espaço vetorial e re-embed (E-158)

## 1. O risco concreto

O limite declarado era *"trocar o modelo de embedding não re-embeda o histórico, sem backfill"*. O risco não é
a ausência de recurso: é a busca comparar vetores de **espaços diferentes** e devolver um resultado que
**parece bom**. Similaridade de cosseno entre espaços distintos é ruído com cara de resposta.

| Peça | O que faz |
| --- | --- |
| `Memory.EmbeddingModel` | carimba QUAL modelo gerou aquele vetor |
| `comparableVector` | decide se a comparação semântica é legítima |
| `CountMemoriesOutsideVectorSpace` | conta o que ficou fora — e vira **declaração** na resposta |
| `ReembedMemories` | recalcula os vetores no espaço atual (a remediação) |

## 2. A regra de comparabilidade (e o erro que a suíte pegou)

| modelo atual identificado | carimbo do vetor | compara? | por quê |
| --- | --- | --- | --- |
| sim | igual | **SIM** | mesmo espaço |
| sim | **diferente** | **NÃO** | espaços distintos: cosseno vira ruído plausível |
| sim | ausente | SIM | dado anterior ao carimbo, gravado pelo MESMO embedder configurado |
| não | qualquer | SIM | sem identidade não há como distinguir espaços — declarado |

Minha primeira versão recusava **carimbo ausente**. Os testes novos passaram, a suíte completa **não**:
`TestSemanticMemorySearchRanksByCosineSimilarity` quebrou — eu tinha acabado de quebrar, em silêncio, uma
busca semântica que já funcionava. Sem rodar a suíte inteira, isso teria ido para a `main` com 4 testes novos
verdes. A regra passou a recusar **só** o caso perigoso (os dois lados identificados e diferentes).

## 3. Remediação

`ReembedMemories(ctx, projectID)` recalcula os vetores fora do espaço atual e regrava o arquivo; falha de
embedding em **uma** memória conta em `Failed` e **não** interrompe as demais — um texto problemático não pode
impedir a recuperação dos outros. Se a gravação falhar, o estado anterior é restaurado em memória.

Quando existem vetores fora do espaço, a resposta da ferramenta declara:
`[N memória(s) fora do espaço vetorial atual: rode o re-embed]`.

## 4. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run 'TestServiceAgent\|TestRuntimeServiceAgents'` | **90/90 PASS** (4 novos) |
| `go test ./internal/agent/ -count=1` | **ok 63,965s** |
| `go test ./server/ -count=1` | **ok 21,381s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** (agente 20,235s · servidor 6,449s) |
| `gofmt -l internal/agent server` | **sem saída** |

Os testes novos: memória de outro modelo **não** é comparada e a resposta **declara**; `ReembedMemories`
traz o conhecimento de volta e a busca semântica volta a achá-lo (sem declaração de espaço); embedder **sem
identidade** continua comparável (o caso que a suíte pegou); e **uma falha não impede** o re-embed das
outras.

## 5. Erros meus nesta rodada (registrados)

1. **A regra estrita quebrou capacidade existente** — descoberto só porque rodei a **suíte completa**.
   Lição: *teste novo verde não é ausência de regressão*.
2. **Sombreamento de `context` outra vez** (variável local × pacote) no arquivo de teste — renomeada para
   `store`. É a segunda vez nesta sequência de rodadas.
3. **Import `time` sem uso** no arquivo novo — o compilador pegou.
4. **`misspell`**: `essencial` é palavra portuguesa correta; entrou na lista de `ignore-rules` do próprio
   repositório (idioma, não permissão).

## 6. Limites declarados

- **O re-embed não tem rota de operador ainda**: existe como operação do store (chamada programática). É a
  lacuna honesta desta rodada — a detecção e a declaração estão prontas, o gatilho pela interface não.
- **Sem identidade não há detecção**: se o embedder não se identifica, não há como saber que o espaço mudou —
  a comparação segue e isso está declarado acima (não é escondido).
- **Re-embed síncrono e por projeto**: um projeto grande embeda tudo dentro da chamada; não há lote, fila nem
  limite de tempo próprio.
- **Sem métrica de tamanho**: os vetores ficam no arquivo de memórias do projeto; o crescimento não foi
  medido nesta rodada.
