# Estágio 32/19 — o painel de BI na interface (E-133)

## 1. O limite que esta rodada fecha

Na rodada anterior (E-132) entreguei o BI no backend e registrei no próprio documento: *"sem tela
dedicada: a superfície entregue é a HTTP"*. Um painel que só existe como JSON não é um painel.

## 2. O que foi entregue

`BusinessStatusPanel`, montado no workspace da empresa, consumindo
`GET /companies/:id/business-status`:

- **17 indicadores** em cartões: empresas, pedidos, pagos, receita, receita/despesa/líquido/conciliado
  financeiro, campanhas, agendadas, impressões, cliques, propostas, follow-ups vencidos, conversas,
  violações de SLA e canais incompletos;
- **"O que este painel NÃO mede"**: a lista `nao_medido` renderizada **inteira**, com o motivo;
- **"Origem de cada número"**: a proveniência bloco a bloco vinda do servidor;
- faixa de **publicação externa** declarada pelo próprio relatório.

## 3. Três regras que guiaram a tela

| Regra | Por quê |
| --- | --- |
| **Sem dado, a tela DIZ que não há dado** | exibir `0` com aparência de resposta é o erro mais fácil de cometer num dashboard. Com tudo zerado, aparece *"Sem dados ainda nesta empresa: os indicadores aparecem quando houver pedido, lançamento, campanha, proposta ou conversa registrados"* |
| **As lacunas ficam NA TELA** | esconder a lista de "não medido" faria o painel parecer completo. Nenhum selo verde é exibido |
| **Falha apaga o número antigo** | quando a consulta falha, o relatório anterior **não** permanece na tela: dado velho apresentado como atual é pior que nenhum. O teste força a falha e exige que `35.00` (o valor anterior) **desapareça** |

E o único controle — **"Atualizar painel"** — realmente refaz a consulta: o teste confirma que a
chamada passa de 1 para 2. Não há botão decorativo.

## 4. Verificação (4 testes + cadeia completa)

| Teste | O que prova |
| --- | --- |
| indicadores e proveniência | receita `35.00` (de `3500` centavos), follow-ups vencidos, violações de SLA, origem `CommerceStore (…)` e a lacuna **na tela** |
| sem dados | aparece "Sem dados ainda nesta empresa" e as lacunas continuam declaradas; nenhum "tudo em ordem" |
| falha | a mensagem real aparece e o número anterior **desaparece** |
| atualizar | o controle refaz a consulta de verdade (1 → 2 chamadas) |

Cadeia: `npx vitest run` → **83 arquivos, 562 testes PASS**; `npx tsc -b` exit **0**;
`npm run lint` **0 erros**; `node scripts/verify-contracts.mjs` →
`[PASS] GET /api/agent/v1/companies/${encodeURIComponent(id)}/business-status -> GET /api/agent/v1/companies/:id/business-status (server/agent_routes.go)` e
**CONTRACT VERIFICATION PASSED**.

## 5. Limites declarados

- **Continua sendo uma fotografia**: sem série temporal, sem tendência e sem comparação entre
  períodos — o painel mostra o agora.
- **Sem exportação** (CSV/PDF) e sem agendamento de relatório.
- **Sem gráfico**: são cartões numéricos; visualização de série depende da série temporal, que ainda
  não existe.
- **Sem e2e Playwright** da jornada nesta rodada: a verificação é unitária + contrato.
