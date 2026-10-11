# Estágio 25/19 — a sincronização passou a enviar o que o operador monta (E-131)

## 1. O limite que esta rodada fecha

Na rodada anterior (E-130) o painel tinha o catálogo e o botão de sincronizar, mas registrei no
próprio documento: *"sincronizar catálogo envia **zero itens** nesta tela: o backend recusa com 400
(nada a sincronizar) e a interface mostra o motivo"*. Botão que só produz erro é controle pela
metade. Esta rodada entrega o **seletor de itens**.

## 2. O que foi entregue

No painel de marketplaces:

- **seletor de SKU** alimentado pelo catálogo real da empresa (`company.products`);
- **quantidade de estoque** por item;
- **lista dos itens montados**, com a versão de cada um;
- botão que informa quantos itens serão enviados: *"Sincronizar 2 item(ns) (sandbox)"*;
- relatório da execução com aplicados, ignorados, conflitos e uso do teto de operações.

## 3. Três decisões deliberadas

| Decisão | Por quê |
| --- | --- |
| **A versão incrementa a cada item adicionado** | o servidor é **idempotente por versão**: reenviar a mesma versão é (corretamente) **ignorado**. Se a tela não incrementasse, o operador veria "0 aplicados, 2 ignorados" e concluiria que a sincronização está quebrada |
| **Botão desabilitado sem item, com o motivo escrito** | o servidor recusa com 400; botão habilitado que sempre falha é o **controle decorativo** proibido pelo estágio 19. A tela diz: *"Adicione ao menos um item: sincronizar sem item é recusado pelo servidor"* |
| **Aplicados saem da fila; conflitos e ignorados PERMANECEM** | item aplicado já foi; item em **conflito** ou **ignorado** ainda precisa de decisão humana. Limpar a fila daria a impressão de sincronização completa — exatamente o "status simulado" que o estágio 10 proíbe |

## 4. Verificação (7 testes, todos PASS)

| Teste | O que prova |
| --- | --- |
| declaração de bloqueio | `BLOCKED_BY_EXTERNAL_DEPENDENCY` **com motivo**; a palavra "no ar" **não** aparece |
| estado real do canal | `CONFIG_REQUIRED`, `Testado ao vivo: não` e procedimento de desbloqueio |
| agendamento | botão **só** para campanha aprovada; envia as horas (`scheduleCompanyCampaign("cmp_1", "cmp_aprovada", 2)`) |
| métricas | botão **só** para campanha que executou; envia números |
| falha | erro do servidor aparece como erro, **sem** a mensagem de sucesso |
| **itens (novo)** | sem item o botão está **desabilitado**; dois itens geram **versões 1 e 2**; a chamada envia **exatamente** `[{estoque, SKU-1, v1, qtd 1}, {estoque, SKU-1, v2, qtd 1}]`; aplicados saem da fila |
| **conflito (novo)** | `conflito: SKU-1 v2` com o detalhe *"divergência não é sobrescrita"* e `ignorado: SKU-1 v1` ficam **visíveis**, e o item **continua na fila** |

Cadeia completa: `npx vitest run` → **83 arquivos, 558 testes PASS**; `npx tsc -b` exit **0**;
`npm run lint` **0 erros**; `node scripts/verify-contracts.mjs` → **CONTRACT VERIFICATION PASSED**.

## 5. Limites declarados

- **Sem publicação real**: `BLOCKED_BY_EXTERNAL_DEPENDENCY` — exige conta autorizada e scopes válidos.
- **Conflito não é resolvido automaticamente**: a tela mostra e mantém na fila; decidir entre manter o
  remoto ou forçar o local é decisão humana, e uma "resolução" automática esconderia divergência de
  estoque — o defeito que o próprio desenho do servidor evita.
- **Sem pull de pedidos** (depende da API real) e **sem e2e Playwright** da jornada nesta rodada.
- **Sem envio automático de catálogo inteiro**: o operador escolhe item por item; enviar tudo sozinho
  seria o oposto do teto de operações que protege contra violação de rate limit.
