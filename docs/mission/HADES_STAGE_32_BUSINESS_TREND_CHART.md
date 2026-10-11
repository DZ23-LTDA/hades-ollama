# Estágios 19/32 — gráfico acessível da série, sem dependência nova (E-139)

## 1. O limite que esta rodada fecha

Eu vinha declarando: *"**sem gráfico**: a série aparece como lista de deltas, não como curva; um gráfico
exige decisão de biblioteca e de acessibilidade (equivalente textual), que ficam para a frente de UX"*.

A decisão tomada foi **não** adicionar biblioteca: onze pontos cabem em um `<polyline>` de SVG. Isso
elimina a decisão de licença, o peso do pacote e o risco de um gráfico que só existe no navegador.

## 2. O que foi entregue

`BusinessTrendChart` com:

- **curva SVG** (`role="img"` + `aria-label` descritivo, tipo *"Receita (pedidos): 2 amostra(s)"*);
- **tabela equivalente SEMPRE visível** (janela, valor, variação), porque um gráfico sozinho exclui quem
  usa leitor de tela e quem precisa do número exato — e esconder a tabela atrás de um botão a torna
  inútil para quem mais precisa dela;
- **seletor de métrica** com 11 métricas em pt-BR, no painel (`BusinessStatusPanel`);
- valores em dinheiro formatados (centavos → reais) e primeira linha **sem** variação (`—`), porque
  "+0" inventaria um marco que não existe.

## 3. Quatro riscos clássicos, cada um travado por teste

| Risco | O que o teste exige |
| --- | --- |
| **`NaN` na curva** quando todos os valores são iguais (divisão por zero) | coordenadas finitas e linha plana no meio; um `polyline` com `NaN` **não desenha e não avisa** |
| **Curva fora de ordem** (parece oscilação real do negócio) | `trendPoints` ordena por janela |
| **Gráfico sem alternativa acessível** | o nome acessível e a tabela com valores e variações estão no render |
| **"Tendência" com uma amostra** | menos de 2 amostras → diz que faltam amostras, mostra a amostra única e **não** desenha curva |

Também há teste para **valores negativos** (o financeiro líquido pode ser negativo) e para série vazia.

## 4. O bug latente que o `tsc` pegou — e por que os testes não pegaram (registrado)

`TREND_METRICS` é uma **lista de dados**: cada item tem um `pick` que lê um campo do relatório. A métrica
*"Lançamentos em aberto"* lia `finance.unreconciled`, e o campo real é **`unreconciled_entries`**.

O `npx tsc -b` acusou na hora (`TS2339: Property 'unreconciled' does not exist`), mas **todos os testes
passavam** — porque o `pick` é uma *lambda*: só executa quando aquela métrica é **escolhida**, e nenhum
teste a escolhia. Em produção, selecionar essa opção mostraria `undefined` na tabela e coordenadas
inválidas.

A correção veio em **duas partes**: (a) o campo certo; (b) um teste que **percorre a lista inteira**,
exigindo valor finito e formatação sem `undefined`/`NaN` para **cada** métrica. Fica registrado o
princípio: **typecheck não é redundante com teste** — e lista de opções é dado, então cada item precisa
ser verificado por varredura, não por lembrança.

Também registrado um erro menor meu: o teste novo falhou primeiro por **import faltando**
(`formatTrendValue`), pego na primeira execução.

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `npx vitest run src/components/BusinessStatusPanel.test.tsx src/components/BusinessTrendChart.test.tsx` | **20/20 PASS** |
| `npx vitest run` (suíte web) | **580 testes PASS** (569 + 11 novos — **contagem conferida**, prática de E-137) |
| `npx tsc -b` | exit **0** |
| `npm run lint` | **0 erros** (12 avisos pré-existentes) |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |

## 6. Limites declarados

- **Sem zoom, tooltip nem seleção de intervalo**: a curva mostra o recorte retido inteiro.
- **Uma métrica por vez**: comparar duas séries no mesmo eixo é decisão de produto (e de escala), não
  improviso — fica declarado em vez de escondido.
- **Eixo Y sem marcações numéricas**: os valores exatos estão na tabela equivalente, que é a fonte
  acessível; rótulos no eixo ficariam ilegíveis em 96 px de altura.
