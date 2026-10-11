# Estágio 32/19 — a série temporal chegou à interface (E-135)

## 1. O limite que esta rodada fecha

Na rodada anterior (E-134) entreguei a série no backend e registrei: *"sem gráfico; a série existe mas
não é exibida"*. Série que ninguém vê não responde "está melhorando?".

## 2. O que foi entregue

Seção **"Série temporal"** no painel de BI (`BusinessStatusPanel`):

- contagem de amostras, **retenção máxima** e **quantas ficaram fora do recorte**;
- quando há duas amostras: **intervalo comparado** e os **deltas com sinal** (`+1`, `-1000`), no formato
  `métrica: delta (de → para)`;
- quando **não** há duas amostras: o **motivo** do servidor, sem nenhum delta;
- controle **"Registrar amostra agora"**, que realmente chama o `POST` e informa o que aconteceu.

## 3. Quatro regras de honestidade (todas travadas por teste)

| Regra | Por quê |
| --- | --- |
| **Sem duas amostras, o MOTIVO aparece — nunca delta zero** | "não mudou" e "não há com o que comparar" são coisas diferentes; exibir `0` seria inventar uma medição |
| **O sinal é explícito** (`+1`, `-1000`) | alta e queda precisam ser distinguíveis de relance, sem depender de cor |
| **`created=false` é dito como é** | o servidor é idempotente por janela de hora; a tela diz *"Já havia amostra nesta hora…"* em vez de comemorar um dado novo que não existe |
| **O descarte é declarado** | `2 fora do recorte` aparece quando a retenção deixou amostras para trás — esconder isso faria a série parecer completa |

Falha ao registrar amostra também aparece como falha (teste força `série indisponível: falha simulada`).

## 4. Verificação (8 testes no componente)

| Teste | O que prova |
| --- | --- |
| indicadores e proveniência | (existente) receita `35.00`, origem dos blocos e lacunas na tela |
| sem dados | (existente) "Sem dados ainda nesta empresa" |
| falha | (existente) erro real e o número anterior **desaparece** |
| atualizar | (existente) refaz a consulta (1 → 2 chamadas) |
| **sem base (novo)** | mostra "Sem tendência: apenas uma amostra…", a retenção, e **não** exibe "Intervalo comparado" |
| **deltas (novo)** | intervalo exato, `pedidos: +1 (3 → 4)`, `liquido_cents: -1000 (1500 → 500)`, `violacoes_sla: -1 (1 → 0)` e `2 fora do recorte` |
| **janela (novo)** | `created=true` → "Amostra registrada agora."; `created=false` → "havia amostra nesta hora" e **sem** a mensagem de novidade |
| **falha ao registrar (novo)** | a mensagem de erro aparece |

Cadeia: `npx vitest run` → **83 arquivos, 566 testes PASS**; `npx tsc -b` exit **0**;
`npm run lint` **0 erros**; `node scripts/verify-contracts.mjs` →
`[PASS] POST …/business-snapshots` e **CONTRACT VERIFICATION PASSED**.

## 5. Um erro meu, pego pelo próprio teste (registrado)

A primeira versão da asserção procurava `"já havia amostra nesta hora"` enquanto a mensagem começa com
**"Já"** — `toContain` diferencia caixa, e o teste falhou. A correção foi no **teste** (comparar sem a
primeira letra), não no texto do produto: a mensagem está correta e capitalizada.

## 6. Limites declarados

- **Sem gráfico**: a série aparece como lista de deltas, não como curva. Um gráfico exige decisão de
  biblioteca e de acessibilidade (equivalente textual), que ficam para a frente de UX.
- **Sem agregação por período** (diária/semanal) e **sem exportação** dos dados.
- **Sem agendador**: registrar amostra continua sendo ação explícita de quem opera.
- **A tendência compara extremos** do período, não médias — está dito no próprio nome dos campos.
