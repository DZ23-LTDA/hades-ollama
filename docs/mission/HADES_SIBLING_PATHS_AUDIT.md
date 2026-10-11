# Auditoria transversal — a família de "caminhos irmãos"

## 1. O método (e por que ele existe)

Três rodadas seguidas encontraram o **mesmo tipo** de defeito: um dado com escopo
por ator cujo **filtro existia em um caminho e faltava no irmão**. Corrigir caso a
caso não fecha a família; então esta rodada varreu **todos os leitores** de cada
recurso escopado.

Regra aplicada: para cada recurso com escopo por ator, **enumerar todo caminho que
lê o mesmo dado** e conferir, um por um, se o filtro é aplicado. Função escopada
existir **não** prova que ela é usada.

## 2. A varredura, medida

Busca no servidor por chamadas às contrapartes **sem ator** das funções escopadas
(`SearchMemoriesContext`, `AddMemoryContext`, `ExportMemories`, `RemoveMemory`,
`RetrieveRelevant`) e pelas versões **com ator**:

| Chamada sem ator | Antes | Depois |
| --- | --- | --- |
| `SearchMemoriesContext` | **1** (`/memory`) | **0** |
| `AddMemoryContext` | 0 | 0 |
| `ExportMemories` | 1 (corrigido em E-088) | 0 |
| `RemoveMemory` | 1 (corrigido em E-088) | 0 |
| `RetrieveRelevant` | 0 | 0 |
| **Total sem ator** | **3** | **0** |

Chamadas **com ator** hoje: `SearchMemoriesForActor` 3, `AddMemoryForActor` 1,
`ExportMemoriesForActor` 1, `DeleteMemoryForActor` 1, `GroundedAnswerForActor` 1.

Observação de método: a única ocorrência restante de `SearchMemories(` é a
**definição** do handler `func (a *agentAPI) searchMemories`, verificada linha a
linha — não é chamada.

## 3. O quarto irmão (ALTO, ativo)

O comando **`/memory`** — da lista de comandos obrigatórios da seção 6 — chamava
`SearchMemoriesContext` e devolvia a memória `private` de **outro usuário** do
mesmo projeto **no corpo da resposta**. Ele validava organização e projeto, mas
não o ator. Corrigido para `SearchMemoriesForActor` com `agentActorID(c)`.

O mesmo dado era servido por **quatro** caminhos, cada um protegendo um pedaço
diferente:

| Caminho | Estado |
| --- | --- |
| Rota de busca | protegido desde o início |
| Exportação | corrigido em E-088 |
| Exclusão | corrigido em E-088 |
| Contexto do modelo | corrigido em E-089 |
| **Comando `/memory`** | **corrigido nesta rodada** |

## 4. Famílias verificadas e consideradas corretas

- **Aprovações de missão**: a rota usa `DecideApprovalForActorCAS` — escopado.
- **Aprovações de empresa**: passam `organizationID` e `agentActorID` ao store, e o
  caminho por ID exige `requireApprovalApprover`.
- **`/skills` e `/mcp`**: listam por **organização**; não existe conceito privado
  por usuário nesses recursos, então o escopo de organização é o correto.

## 5. Regra que fica para o projeto

> Recurso com escopo por ator só está protegido quando **todos** os caminhos que
> leem o dado aplicam o filtro. A existência de uma função escopada não é prova de
> uso — e o irmão esquecido é o caminho mais provável do vazamento.

O invariante agora é **verificável por comando**: zero chamadas sem ator no
servidor, e qualquer chamada nova aparece nessa contagem.
