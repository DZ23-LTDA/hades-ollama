# Estágio 15 — exportação e exclusão de memória: defeito ALTO corrigido

**Origem:** a auditoria de bloqueios (E-087) reabriu o estágio 15 como trabalho
implementável e local. Faltava a **evidência obrigatória** do estágio:
exportação e exclusão de memória.

## 1. O defeito encontrado (ALTO)

A **busca** de memória já era escopada por ator (`SearchMemoriesForActor`), mas os
caminhos **irmãos** chamavam as funções **sem ator**:

| Rota | Antes (`origin/main`) | Efeito |
| --- | --- | --- |
| `GET /projects/:id/memories/export` | `a.context.ExportMemories(...)` | **VAZAVA** memória `private` de outro usuário da mesma organização no arquivo exportado |
| `DELETE /projects/:id/memories/:memory_id` | `a.context.RemoveMemory(...)` | Qualquer usuário autenticado **apagava** a memória privada de outro |

Mesmo dado, duas respostas: a busca escondia e o export entregava. É o padrão
clássico de *protegi um caminho e esqueci o irmão* — e o risco que o próprio
arquivo de isolamento diz existir para evitar (*“um workspace compartilhado vaza
anotação pessoal entre colegas — risco ALTO de privacidade”*).

## 2. A correção

| Rota | Depois |
| --- | --- |
| export | `ExportMemoriesForActor(projectID, agentActorID(c), includeEmbeddings)` |
| delete | `DeleteMemoryForActor(projectID, memoryID, agentActorID(c))` |

Duas decisões deliberadas:

1. **Exportar é escopado por ator.** Exportar “o projeto inteiro” é o vetor do
   vazamento; o arquivo exportado passa a conter apenas o que o ator pode ler, e
   registra `actor_id` e `generated_at` para auditoria.
2. **A exclusão recusada é indistinguível de memória inexistente.** Apagar memória
   privada alheia devolve o **mesmo** `ErrMemoryNotFound` (404) de um ID que não
   existe: distinguir os casos transformaria a rota em oráculo de existência de
   dado privado.

A persistência **não** foi reimplementada: `DeleteMemoryForActor` reusa
`RemoveMemory`, que já faz escrita atômica com rollback. O compilador ainda pegou
uma duplicação minha — eu havia declarado um segundo `ErrMemoryNotFound` — e o
sentinela existente foi reaproveitado.

## 3. Verificação (medida, não presumida)

- `export` antes/depois, lido do próprio Git: `origin/main` chamava `ExportMemories`,
  a árvore de trabalho chama `ExportMemoriesForActor`.
- 4 testes de motor: export não vaza privado alheio; exclusão própria persiste
  **após reinício**; recusa tem o mesmo erro de inexistente e **não** apaga; export
  sobrevive ao reinício com o mesmo conteúdo; vetores só saem se pedidos.
- 2 testes de rota com o harness existente: o export do bruno **não** contém a
  memória privada da ana (e o da própria ana contém); o delete do bruno recebe 404
  e a memória continua lá.
- `go build ./...` exit 0 · suítes com apenas as falhas ambientais conhecidas ·
  `golangci-lint` **0 issues** · `gofmt` limpo.

## 4. Estado do estágio (sem promoção)

| Item do critério | Situação |
| --- | --- |
| Isolamento entre usuários | **coberto** |
| Reinício | **coberto** (export e exclusão sobrevivem) |
| Exportação e exclusão (evidência obrigatória) | **coberto** nesta rodada |
| Troca de modelo preservando contexto | **pendente** |
| Retomada | **pendente** |

O estágio **permanece `IN_PROGRESS`** com capacidade `IMPLEMENTADO_NAO_HOMOLOGADO`.
Promover agora repetiria exatamente o erro corrigido em E-087: fechar o estágio com
parte do critério sem prova.
