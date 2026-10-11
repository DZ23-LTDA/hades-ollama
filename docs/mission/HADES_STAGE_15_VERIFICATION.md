# Verificação do Estágio 15 — Memória Universal e Contexto Persistente

**Método:** releitura da **lista de atividades** do estágio contra o código e as
evidências, atividade por atividade, sem presumir cobertura. Mesmo método usado no
estágio 13 (E-085) — e que lá **refutou** a suposição de cobertura.

## 1. Atividade por atividade

| Atividade do estágio | Cobertura | Onde está | Evidência |
| --- | --- | --- | --- |
| Escopo de **usuário** | COBERTO | `memoryVisibleTo`, `AddMemoryForActor` | E-015, E-088 |
| Escopo de **organização** | COBERTO | raiz do store por organização + política de retenção por organização | E-015 |
| Escopo de **projeto** | COBERTO | chave `memories/<projeto>.json` | E-015 |
| Escopo de **workspace** | DERIVADO | o projeto é criado dentro do workspace; a memória herda por projeto | E-095 (limite declarado) |
| Escopo de **missão** | COBERTO | `Memory.MissionID` + `memoryInScope` | E-095 |
| Escopo de **agente** | COBERTO | `Memory.AgentID` + `memoryInScope` | E-095 |
| Escopo de **equipe** | COBERTO | `Memory.TeamID` + `memoryInScope` | E-095 |
| Indexação | COBERTO | `ContextStore.memories` + embeddings | E-015 |
| Busca semântica | COBERTO | `RetrieveRelevant`, `embedding.go` | E-015 |
| **Resumos** | COBERTO | `memory_summary.go`, rota `/memories/summary` | **E-096** |
| **Versionamento** | COBERTO | `memory_version.go`, rota `PATCH` + `/history` | E-093 |
| Proveniência | COBERTO | `Source`/`Kind` + derivação de fonte não confiável | E-040, E-096 |
| **Deduplicação** | COBERTO | `memory_dedup.go` | E-092 |
| Retenção | COBERTO | `memory_retention.go` | E-015 |
| Atualização | COBERTO | `UpdateMemoryForActor` | E-093 |
| Exclusão | COBERTO | `DeleteMemoryForActor` | E-088 |
| Exportação | COBERTO | `ExportMemoriesInScope` | E-088, E-095 |
| Controle de acesso | COBERTO | `memoryInScope` + rotas escopadas por ator | E-088, E-089, E-090, E-095 |
| **Criptografia** | COBERTO | `memory_encryption.go` (AES-256-GCM do primitivo existente) | E-094 |
| Privacidade | COBERTO | isolamento por ator/organização + DLP + retenção | E-015, E-088 |
| Agentes recuperam conhecimento autorizado **sem misturar organizações** | COBERTO | raiz por organização + testes de tenant | E-015, E-090 |
| Integração externa via contrato | **AUSENTE** | — | o **próprio estágio** a marca como **opcional** |

## 2. Critério de conclusão

"Testar troca de modelo, reinício, retomada e isolamento entre usuários":

| Item | Situação |
| --- | --- |
| Troca de modelo | **provada AO VIVO** entre `qwen2.5:7b` e `llama3.2:3b`, contexto byte-idêntico (E-089) |
| Reinício | **provado** — export, exclusão, dedup, versionamento e histórico sobrevivem (E-088, E-092, E-093) |
| Retomada | **provada** — store novo sobre o mesmo diretório devolve o mesmo contexto (E-089) |
| Isolamento entre usuários | **provado** em motor e rotas (E-015, E-088, E-089, E-090, E-095) |

## 3. Evidência obrigatória

| Evidência exigida | Situação |
| --- | --- |
| Testes de isolamento | **COBERTA** |
| Proveniência | **COBERTA** |
| Exportação/exclusão | **COBERTA** (E-088) |

## 4. Defeitos reais corrigidos neste estágio (todos ALTO)

1. **Exportação** vazava memória privada de outro usuário (E-088);
2. **Exclusão** deixava apagar memória privada alheia (E-088);
3. **Contexto do modelo** levava memória privada alheia ao preâmbulo de sistema, que
   **vai para o modelo** (E-089);
4. **Comando `/memory`** devolvia memória privada alheia no corpo da resposta (E-090).

Família de **caminhos irmãos**: o mesmo dado servido por cinco caminhos, cada um
protegendo um pedaço. Fechada, e o invariante (**zero leitura sem ator**) ficou
verificável por comando (E-090, E-091).

## 5. LIMITES DECLARADOS (não são cobertura)

1. **Resumo padrão é digest determinístico**, não texto de modelo. O contrato aceita
   um resumidor injetado e o resultado **declara qual foi usado**; não há resumo por
   modelo homologado.
2. **Escopo de workspace é derivado do projeto**, não um campo próprio da memória.
3. **Integração externa opcional ausente** — o próprio estágio a marca como
   *opcional*.
4. **Criptografia é opt-in** e a **gestão da chave é do operador**: chave perdida
   significa memória cifrada **irrecuperável**, e o erro diz isso.
5. **Busca semântica depende do embedder configurado**; sem ele a comparação é por
   texto, e o sistema não finge semântica.

## 6. Veredito

- **Critério de conclusão:** CUMPRIDO, com a troca de modelo **provada ao vivo**.
- **Lista de atividades:** COBERTA; o único item ausente é **opcional pelo próprio
  estágio** e fica declarado como limite.
- **Evidência obrigatória:** COBERTA.
- **Execução do estágio:** `COMPLETED_VERIFIED`.
- **Capacidade (Memory and Context Engine):** `IMPLEMENTADO_E_TESTADO`.
- **Release global:** permanece `NOT_RELEASE_READY` — este é **um** dos 38 estágios.
