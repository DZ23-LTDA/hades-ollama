# Estágio 15 — versionamento de memória

## 1. O que faltava

Antes desta rodada a memória era só **append + delete**: corrigir uma anotação
exigia apagar e recriar, e o que foi corrigido **se perdia**. Sem histórico não há
como auditar quem reescreveu o conhecimento do projeto nem desfazer uma alteração
errada — exatamente o risco *"memória sem proveniência"* do estágio.

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| O histórico guarda o **conteúdo anterior** com quem alterou, data e motivo | auditabilidade e desfazer |
| O histórico é **limitado** (`MaxMemoryVersions = 20`, descarta o mais antigo) | sem teto, uma memória muito editada viraria arquivo sem fim no store e no export |
| Alteração passa pela **mesma checagem de visibilidade** da exclusão | memória privada alheia é recusada com o erro de **inexistente**, para a rota não ser oráculo |
| O vetor de embedding do texto antigo é **invalidado** | manter vetor de texto que não existe mais faria a busca semântica devolver a memória errada |
| Reedição com o **mesmo conteúdo** não cria revisão | histórico é de alterações, não de tentativas |
| Persistência com **rollback** no mesmo padrão do add | erro de escrita não pode deixar estado divergente |

## 3. Alcançável pelo produto

A capacidade não ficou só na API Go:

- `PATCH /projects/:id/memories/:memory_id` — altera e versiona;
- `GET /projects/:id/memories/:memory_id/history` — histórico da memória.

As duas são **escopadas por ator** (o quarto irmão foi exatamente um caminho que
esqueceu o ator) e devolvem **404 uniforme** para memória inexistente e para memória
privada alheia.

## 4. Verificação

- **4 testes de motor**: histórico com conteúdo anterior, quem alterou, motivo e
  data; recusa de memória privada alheia e do histórico alheio com a memória
  intacta; **teto** do histórico após 25 edições e **sobrevivência ao reinício**;
  vetor invalidado na alteração.
- **2 testes de rota**: alteração pela dona versiona e o terceiro recebe 404 sem ver
  conteúdo; histórico legível pela dona e recusado a terceiro; conteúdo vazio recusado
  com 400; a busca encontra a revisão **atual**; e as duas rotas estão **registradas
  no roteador** (não basta a função existir).

`go build ./...` exit 0 · suítes apenas com as **falhas ambientais conhecidas** ·
`golangci-lint` **0 issues** (o `misspell` marcou `autor` como erro de `author` em três
pontos — texto reescrito, regra **não** silenciada) · `gofmt` limpo.

## 5. Estado do estágio 15

| Atividade | Situação |
| --- | --- |
| Isolamento · exportação/exclusão · contexto do modelo · troca de modelo · reinício · retomada | cobertos |
| Deduplicação | coberta (E-092) |
| **Versionamento** | **coberto nesta rodada** |
| Resumos · criptografia em repouso · escopos missão/agente/equipe | **ainda ausentes** |

Continua **`IN_PROGRESS`**: ainda faltam três atividades.
