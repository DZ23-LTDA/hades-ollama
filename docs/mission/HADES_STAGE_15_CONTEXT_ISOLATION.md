# Estágio 15 — contexto do modelo, troca de modelo e o que ainda falta

## 1. O terceiro irmão da mesma família (ALTO, ativo)

Na rodada anterior corrigi o **export** e o **delete** de memória, que não eram
escopados por ator. Faltava o caminho que mais longe leva o dado: o **contexto
injetado no modelo**.

| Caminho | Antes | Efeito |
| --- | --- | --- |
| Busca | `SearchMemoriesForActor` | protegido |
| Export | `ExportMemories` | **corrigido** em E-088 |
| Delete | `RemoveMemory` | **corrigido** em E-088 |
| **Contexto do modelo** | `SearchMemoriesContext` + filtro só por **projeto** | **vazava memória `private` de outro usuário para o preâmbulo de sistema** |

O preâmbulo de sistema **vai para o modelo**. Com um modelo externo, a anotação
privada de um colega sairia da máquina. Medido no código: a rota montava o
contexto com a busca **sem ator** e `BuildContextBootstrap` filtrava apenas por
`ProjectID`, ignorando `Visibility` e `ActorID`.

## 2. A correção (fail-closed, em duas camadas)

1. `ContextBootstrapInput` ganhou **`ActorID`** e o builder **recusa** memória que
   o ator não pode ler — nem um chamador que entregue a lista crua do projeto
   consegue vazar.
2. A rota passou a usar `SearchMemoriesForActor` **e** a informar o ator: não se
   entrega ao builder o que não se deve ler.

## 3. Critério do estágio, provado AO VIVO com dois modelos reais

O critério pede **troca de modelo, reinício, retomada e isolamento**. Não simulei:
o Ollama local desta máquina está ativo com 8 modelos, e o teste fez a troca
**real** entre `qwen2.5:7b` e `llama3.2:3b` (9,91s).

O que ficou provado:

- os **dois modelos** chegaram ao provedor (a seleção por missão vence o padrão);
- o contexto injetado ficou **byte-idêntico** na troca — mensagem de sistema e
  conjunto completo de mensagens;
- a memória privada de **outro** usuário **não** foi enviada ao modelo, e o
  conhecimento do time **foi**;
- reinício do store sobre o mesmo diretório devolve o **mesmo** contexto (retomada).

Sem Ollama local o teste é **pulado com causa declarada**, nunca aprovado por
omissão.

## 4. Por que o estágio NÃO foi promovido

Medi as atividades restantes em vez de supô-las:

| Atividade do estágio | Medição |
| --- | --- |
| Isolamento por organização/projeto/usuário | coberto |
| Busca semântica com embeddings | **existe** (`embedding.go`, `RetrieveRelevant`) |
| Exportação e exclusão | **coberto** (E-088) |
| Proveniência | **existe** (`Source`, `Kind`, fontes não confiáveis derivadas) |
| Retenção | **coberto** (E-015) |
| **Deduplicação** | **AUSENTE** na memória (os `dedup` do repositório são de fila/webhook/workspace) |
| **Versionamento da memória** | **AUSENTE** |
| **Resumos** | **AUSENTE** |
| **Criptografia em repouso** | **AUSENTE** para memória (só `auth.go`) |
| **Escopos missão/agente/equipe** no tipo `Memory` | **AUSENTE** (só `ProjectID` + `ActorID`/`Visibility`) |

O **critério** está cumprido e a **evidência obrigatória** também, mas **quatro
atividades** seguem ausentes. Pelo mesmo padrão aplicado ao estágio 13 — promover
só com a lista de atividades coberta — o estágio permanece **`IN_PROGRESS`**. A
diferença é que agora o que falta está **nomeado e medido**, não presumido.
