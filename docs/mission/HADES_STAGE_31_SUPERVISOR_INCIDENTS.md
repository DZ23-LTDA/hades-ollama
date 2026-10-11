# Estágio 31 — incidentes, recuperação e logs de supervisão

## 1. O que faltava

O supervisor já **coordenava** (tick com schedules, retomada de missões e ciclos de
empresa), **limitava** (pausa por risco, aprovação obrigatória para risco não-leitura,
`CompanyBudget` no gasto) e **autorizava** (limiar de aprovação por valor). Faltavam
duas evidências obrigatórias do estágio:

| Evidência | Antes | Agora |
| --- | --- | --- |
| **Incidentes** | AUSENTE — havia contadores e uma lista de erros no resultado do tick | registro com **estado**, tentativas e recuperação |
| **Logs de supervisão** | contadores **só em memória**: reiniciar apagava "o que o supervisor fez" | **diário de ticks** limitado e **persistido**, com os erros de cada tick |

## 2. O defeito encontrado no caminho (falha engolida em silêncio)

No passo de retomada, o código fazia:

```
if err := resumePending(ctx); err == nil { ... }
else if retryErr := resumePending(ctx); retryErr == nil { ... }
```

Se **as duas** tentativas falhassem, **nada acontecia**: sem log, sem contador, sem
incidente. Missões pendentes paravam de ser retomadas e **o operador não tinha como
saber**. Corrigido: a falha persistente entra no resultado do tick **e** vira incidente
do tipo `recovery_failure`, e a recuperação (quando ocorre) é registrada.

## 3. Decisões deliberadas do registro de incidentes

| Decisão | Motivo |
| --- | --- |
| **Idempotente por (tipo, sujeito)** | a mesma falha repetida **soma tentativas**; um registro que cresce com a repetição vira ruído e esconde o que importa |
| **Recuperação OBSERVADA** | sucesso posterior do mesmo sujeito **resolve com data**; ninguém precisa "marcar como resolvido" para o registro dizer a verdade |
| **Reabre** | sujeito que volta a falhar depois de resolvido **reabre** — histórico que voltou a ser problema não fica escondido como resolvido |
| **Ciência ≠ resolução** | `acknowledge` exige **ator identificado** e **não** fecha; quem assume o incidente não é quem prova que ele acabou |
| **Limitado e persistido** | teto de 50 e gravação no diretório de dados: incidente que desaparece no reinício não serve para operação |

## 4. Alcançável pelo produto

- `GET /supervisor/incidents` — registro com estado e recuperação;
- `POST /supervisor/incidents/:id/acknowledge` — ciência humana (ator identificado);
- `POST /supervisor/incidents/:id/resolve` — resolução com motivo;
- `GET /supervisor/ticks` — diário dos últimos ticks.

Todas com a **mesma guarda** do supervisor (`requireSupervisorAdmin`, SEC-09): não
inventei autorização nova. Incidente inexistente devolve **404 explícito**, não 200 com
lista vazia.

## 5. Verificação

- **7 testes de motor**: idempotência por sujeito com soma de tentativas · recuperação
  observada resolve com data e não cria registro novo · resolvido reabre ao falhar de
  novo · ciência exige ator e não resolve · registro sobrevive ao reinício e respeita o
  teto · **tick registra a falha de retomada** que antes era engolida · diário grava
  cada tick **com os erros** e sobrevive ao reinício.
- **2 testes de rota**: listagem responde mesmo vazia; inexistente é 404 nas duas
  operações, e a resposta de erro **não** pode parecer resolvida; rotas registradas.
- `go build ./...` exit 0 · `golangci-lint` **0 issues** · suítes apenas com as
  **falhas ambientais conhecidas**.

## 6. Por que o estágio **NÃO** foi promovido

O critério ("supervisão operando em sandbox com limites e recuperação") está coberto, e
duas evidências obrigatórias foram fechadas nesta rodada. Mas a **lista de atividades**
ainda não está: coordenar marketing, vendas, atendimento, estoque, pedidos, finanças,
operações, conteúdo, tecnologia e qualidade depende dos **estágios 24–30**, que estão
**AUSENTES**. Promover agora repetiria exatamente o erro que a verificação do estágio 13
refutou. O estágio permanece **`IN_PROGRESS`**, com o que falta **nomeado**:

1. coordenação dos domínios de negócio (depende dos estágios 24–30);
2. orçamento de **tokens/custo de modelo** (o orçamento existente é o de gasto da
   empresa);
3. supervisor é componente **global** da implantação, com pino opcional por organização
   (`OLLAMA_SUPERVISOR_ORG`) — não há um supervisor por organização.
