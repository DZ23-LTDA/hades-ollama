# Estágio 35 — re-embed assíncrono com progresso e estimativa (E-162)

## 1. A pendência que estava aberta

E-159 entregou a rota de operador e declarou os limites: *"re-embed SÍNCRONO, 200 por chamada e por projeto,
sem estimativa prévia do volume"*. Operar uma base grande assim significava chamar várias vezes e adivinhar o
progresso. Agora a operação é um **job observável**.

```
GET  /api/agent/projects/:id/memories/reembed/estimate
POST /api/agent/projects/:id/memories/reembed/jobs
GET  /api/agent/projects/:id/memories/reembed/jobs/:job_id
POST /api/agent/projects/:id/memories/reembed/jobs/:job_id/cancel
```

| Decisão | Por quê |
| --- | --- |
| **Estimativa antes** (`pendentes` + `modelo_atual`) | "faltam 300" sem dizer **para qual modelo** é número solto |
| **Um job por projeto** (409 no segundo) | dois jobs competiriam pelo mesmo arquivo: o último a gravar apagaria o trabalho do outro |
| **Progresso gravado por LOTE** (25) | processo que morre no meio não perde o que já foi feito |
| **Job vazio nasce concluído** | não há "rodando por um instante" fingindo trabalho |
| **`tentadas` inclusive as falhas** | memória cujo embedder recusou seria reprocessada **para sempre**: laço infinito |
| **Cancelamento é PEDIDO** | o job verifica entre lotes e termina como `cancelled` com o que fez contabilizado |
| **Job é do projeto** | consultar por outro projeto é recusado — 403 na checagem de projeto, 404 na de dono |

## 2. Verificação

| Comando | Resultado |
| --- | --- |
| `go test ./internal/agent/ ./server/ -run 'TestServiceAgent\|...\|TestReembedJob\|TestAgentReembedJob'` | **103/103 PASS** (6 novos de job + 3 de rota) |
| `go test ./internal/agent/ -run TestReembedJob -count=3` | **ok 8,219s** |
| `go test ./internal/agent/ -run TestReembedJob -race` (contêiner Linux, `CGO_ENABLED=1`) | **ok 3,163s — sem corrida de dados** |
| `go test ./internal/agent/ -count=1` | **ok 71,049s** |
| `go test ./server/ -count=1` | **ok 23,153s** |
| `golangci-lint run ./internal/agent/... ./server/...` | **0 issues** |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |
| `scripts/verify-linux-parity.ps1` | **LINUX PARITY OK** |
| `gofmt -l internal/agent server` | **sem saída** |

Testes: estimativa antes (e zero sem espaço de destino); job roda até o fim com contadores; **segundo job
recusado**; job vazio nasce concluído; **falha não vira laço infinito** e o que falhou continua contado;
**cancelamento preserva o que foi gravado**; job desconhecido é declarado; e pela rota: 202 ao iniciar,
progresso até `done`, **409** no segundo job e **recusa sem vazar conteúdo** em outro projeto.

## 3. Dois achados que a verificação pegou (e que valem mais que o código novo)

1. **`vet` do Linux pegou `lostcancel`**: em `StartReembedJob`, o caminho do job vazio retornava **sem chamar
   `cancel`** — vazamento de contexto. O `go vet` do Windows **não** reportou; o do contêiner **sim**. Corrigido
   com `cancel()` explícito naquele caminho. **A lição operacional de E-104/E-127 se repetiu uma quarta vez**:
   o gate Linux vê o que o host não vê.
2. **O log obrigatório da paridade (E-155) pagou no mesmo dia**: a falha veio com a linha *"nenhum `--- FAIL:`
   na saida; a falha foi de compilacao, vet ou imagem"* e o caminho do log — em vez de me fazer caçar um teste
   que não falhou.

## 4. Erros meus nesta rodada (registrados)

1. **Deduplicação do motor**: montei o teste com conteúdos `"memória a".."memória z"` (26 únicos) esperando 60
   memórias — `AddMemory` **deduplica**, então o job relatou 26. O motor estava certo; o teste é que supunha.
2. **Recusa acontece antes do que eu esperava**: esperei **404** (dono do job) e veio **403** (escopo do
   projeto). É defesa em profundidade, não defeito — o teste passou a exigir "não entrega e não vaza".
3. **Quarta repetição da lição do gate**: `lostcancel` só apareceu no `vet` do Linux.

## 5. Limites declarados

- **Estado do job em memória**: reiniciar o processo perde o registro (o que já foi **gravado** permanece).
  Persistir o job é outra feature, e não está feita.
- **Um job por projeto**: não há fila de jobs por organização nem paralelismo entre projetos no mesmo processo.
- **Progresso é por lote (25)**: `percent` pode ficar até 25 memórias atrás do trabalho real.
- **`Total` é a estimativa do início**: memória escrita DURANTE o job entra na conta de `Remaining` depois,
  não no `Total` original.
- **Cancelamento entre lotes**: um lote em andamento termina antes de o cancelamento valer.
- **Concorrência entre processos**: dois processos Hades no mesmo arquivo continuam resolvendo por
  último-escritor — o job não introduz trava entre processos.
