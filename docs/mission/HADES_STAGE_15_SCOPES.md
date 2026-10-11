# Estágio 15 — escopos de missão, agente e equipe

## 1. O que faltava

O estágio pede escopos de **usuário, organização, projeto, workspace, missão, agente
e equipe**. Organização e projeto sempre estiveram na chave do store; o escopo de
usuário veio com o isolamento por ator. Faltavam **missão, agente e equipe**.

## 2. A distinção que evita o erro clássico

| Eixo | Regra | Por quê |
| --- | --- | --- |
| **PROPRIEDADE** (alterar/apagar) | regra por **ator** | o dono continua corrigindo e removendo a própria memória mesmo presa a uma missão antiga; negar isso criaria memória **órfã e imortal** — o oposto da exclusão e da retenção que o estágio pede |
| **DIVULGAÇÃO** (ver) | propriedade **mais** escopo | memória presa à missão X não aparece em busca de outro contexto |

Escopo **vazio** significa "vale para o projeto", e é o que preserva todo o
comportamento anterior: só memória com escopo explícito fica restrita.

## 3. Um só lugar decide (de novo)

Havia **nove** pontos decidindo visibilidade. Em vez de acrescentar uma segunda
regra ao lado, os **leitores** passaram a usar `MemoryScope`:

- `SearchMemoriesForActor` → delega para `SearchMemoriesInScope` com escopo de ator;
- `RetrieveRelevantForActor` → delega para `RetrieveRelevantInScope`;
- `GroundedAnswerForActor` → delega para `GroundedAnswerInScope` (**não** chama mais
  `RetrieveRelevantForActor`, para não pular a camada de escopo);
- `ExportMemoriesForActor` → delega para `ExportMemoriesInScope`.

`filterMemoriesForActor` ficou **sem uso** e foi **removida** — a prova de que nenhum
caminho ficou na regra antiga. Regra duplicada é convite a divergir.

## 4. Alcançável pelo produto

As rotas de **busca** e **export** aceitam `?mission_id=`, `?agent_id=` e
`?team_id=`; o ator continua vindo **da sessão**. Um valor de escopo errado
**esconde** dado — nunca revela: o escopo só restringe.

## 5. Verificação

- **3 testes de motor**: memória da missão um **não aparece** em busca sem missão,
  **aparece** na missão dona e **não aparece** na missão dois; o export segue a mesma
  regra; escopo de agente e de equipe restringem a divulgação; a **dona** ainda
  corrige e apaga a própria memória presa a agente, e um **terceiro não**; escopo
  vazio preserva o comportamento anterior, inclusive no RAG fundamentado.
- **1 teste de rota**: busca por HTTP sem missão **não** entrega memória de missão,
  com `mission_id` entrega, o export também respeita, e o escopo de agente restringe.
- `go build ./...` exit 0 · `golangci-lint` **0 issues** · suítes apenas com as
  **falhas ambientais conhecidas**.

## 6. Um erro meu, corrigido pelo teste

Meu primeiro teste esperava que um **terceiro** não pudesse alterar a memória presa ao
agente — mas ela era `organization` (conhecimento **compartilhado**, alterável por
ator autorizado, comportamento já travado em E-088). O **teste estava errado, não o
código**: corrigi o caso para **privada + com escopo**, que é onde a combinação das
duas regras realmente importa.

## 7. Estado do estágio 15

| Atividade | Situação |
| --- | --- |
| Isolamento · exportação/exclusão · contexto do modelo · troca de modelo · reinício · retomada | cobertos |
| Deduplicação · versionamento · criptografia em repouso | cobertos (E-092, E-093, E-094) |
| **Escopos de missão/agente/equipe** | **coberto nesta rodada** |
| **Resumos** | **ainda ausente** |

Falta **uma** atividade: resumos.
