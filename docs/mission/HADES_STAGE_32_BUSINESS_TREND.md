# Estágio 32 — série temporal do BI: amostras datadas com regras explícitas (E-134)

## 1. O limite que esta rodada fecha

Registrei nas duas rodadas anteriores que o BI era uma **fotografia**: cada chamada calculava o agora,
sem memória. Sem série não há tendência — e sem tendência o painel não responde à pergunta que
importa: *está melhorando?*

## 2. O que foi entregue

`BusinessSnapshotStore` + `Runtime.BusinessSnapshot/BusinessTrend` e as rotas:

| Rota | Efeito |
| --- | --- |
| `POST /companies/:id/business-snapshots` | registra a amostra do agora e devolve a série; **201** quando é amostra nova, **200** quando a janela já tinha uma |
| `GET /companies/:id/business-snapshots` | lê a série e a tendência, com recorte opcional `from` (RFC 3339) |

A tendência traz **11 deltas** (pedidos, receita de comércio, receita/despesa/líquido financeiro,
campanhas ativas, impressões, propostas, follow-ups vencidos, conversas, violações de SLA) e o
**intervalo exato** comparado.

## 3. Quatro decisões deliberadas

| Decisão | Por quê |
| --- | --- |
| **Uma amostra por janela (hora)** | registrar em laço criaria uma "tendência" feita de repetições do mesmo instante — ruído com aparência de série. A segunda chamada na mesma hora devolve a amostra existente e `created=false` |
| **Sem duas amostras, a tendência DECLARA ausência de base** | `delta zero` significaria "não mudou"; a ausência de base significa "não há com o que comparar". São coisas diferentes, e confundi-las é o erro clássico de dashboard |
| **Retenção com teto declarado** | o histórico tem limite (`Retention`), e o que fica fora do recorte é **relatado** (`Discarded`). Histórico que finge ser infinito é promessa que o disco não cumpre |
| **O intervalo é explícito** | `Interval` diz de quando até quando comparou, para o número não ser lido como se fosse de outro período |

## 4. Verificação (4 testes no store + 1 na rota)

| Teste | O que prova |
| --- | --- |
| `TestBusinessTrendRequiresTwoSamplesAndComputesRealDeltas` | com **1** amostra: `SemBase=true`, motivo presente e **zero deltas**; depois de mover o negócio (1 pedido pago + `5000` de receita no livro) e registrar a segunda: `pedidos=+1`, `receita_financeira_cents=+5000`, receita de comércio positiva, intervalo declarado |
| `TestBusinessSnapshotIsIdempotentPerWindow` | mesma hora **não** cria outra amostra, devolve a mesma (`ID` igual) e a série continua com `1`; a hora seguinte cria |
| `TestBusinessSnapshotRetentionLimitsAndReportsDiscards` | com teto `3` e `5` registros, a série fica com `3` e a mais antiga é a **terceira** janela; o recorte por período relata `Discarded=2` e, com uma só amostra, declara ausência de base |
| `TestBusinessSnapshotIsScopedByOrganization` | outra organização **não** herda a série; a amostra guarda o relatório da organização dona; escopo ausente é erro |
| `TestBusinessSnapshotRoutesRecordListAndValidate` | **201** na primeira amostra com `created=true`; código coerente com `created` na segunda; `GET` reflete o que o `POST` respondeu; `Retention` declarada; `from` inválido **400**; empresa inexistente **404** |

**Uma decisão de teste que vale registrar**: o handler usa o relógio **real**, então a segunda chamada
pode cair na hora seguinte se o teste cruzar a virada de hora — e aí `created=true` é
**legitimamente** correto. Em vez de escrever uma asserção que falha de vez em quando (flake), o
teste de rota afirma o que é **sempre** verdade (código coerente com `created` e série refletindo a
resposta), e a semântica exata de janela é verificada no **store**, com o instante injetado.

## 5. Verificação da cadeia

- `go test ./internal/agent/ -run 'TestBusinessTrend|TestBusinessSnapshot'` → **4/4 PASS**.
- `go test ./server/ -run TestBusinessSnapshotRoutes` → **PASS**.
- `golangci-lint` → **0 issues** (dois achados corrigidos: `gofumpt` e `intrange`).
- `scripts/verify-linux-parity.ps1` → resultado registrado na evidência.

## 6. Limites declarados

- **Sem agendador**: nada registra amostra sozinho; quem chama o `POST` é o operador, a interface ou
  um agendador externo. Prometer coleta automática sem executor persistente seria mentira.
- **Sem agregação por período** (diária/semanal) e **sem gráfico**: a série é uma lista de amostras
  com deltas entre a primeira e a última.
- **A tendência compara extremos** do período, não médias: com poucas amostras, um pico na ponta
  domina a leitura — está declarado no nome do campo (`from`/`to`).
- **Sem exportação** da série.
