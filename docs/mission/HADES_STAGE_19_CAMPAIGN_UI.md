# Estágio 19 — a capacidade agora é visível e operável na interface (E-130)

## 1. O limite que esta rodada fecha

Nas rodadas 84–87 eu entreguei, no backend, o **agendamento** e as **métricas** de campanha (E-124/E-125)
e a **superfície HTTP do marketplace** (E-128) — sempre com o mesmo limite declarado: *"sem interface
visual"*. Capacidade sem superfície é capacidade que o usuário não tem. Este é o fechamento.

## 2. O que foi entregue

`CompanyCampaignSchedulingPanel.tsx`, ligado ao `CompanyGrowthPanel` (Growth OS), com:

| Controle | Regra aplicada na interface |
| --- | --- |
| **Agendar** | aparece **só** para campanha **aprovada e ainda não agendada** — exatamente a regra do servidor |
| **Registrar métricas** | aparece **só** para campanha em `scheduled`/`active`/`paused` — medir rascunho seria **inventar resultado** |
| **Painel de totais** | impressões, cliques, CTR, conversão e gasto vêm do relatório do servidor |
| **Catálogo de marketplaces** | mostra o **estado real** (`CONFIG_REQUIRED`), `Testado ao vivo: não` e o **procedimento de desbloqueio** |
| **Faixa de publicação externa** | exibe literalmente `BLOCKED_BY_EXTERNAL_DEPENDENCY` **com o motivo** vindo do servidor |

## 3. Duas regras de honestidade que guiaram a implementação

1. **NADA é exibido como publicado.** A faixa de topo mostra o estado declarado pelo servidor, com o
   motivo. Um selo de "campanha no ar" seria mentira — e o teste verifica que a palavra "no ar" **não**
   aparece.
2. **Controle só aparece quando a AÇÃO é possível.** Botão que sempre falha é exatamente o
   "controle decorativo" que o estágio 19 proíbe. Por isso o agendamento não aparece para rascunho, e
   a métrica não aparece para campanha que não executou.

E uma terceira, sobre erro: **falha do servidor aparece como falha**. O teste força um erro de
agendamento e exige que a mensagem real apareça e que a mensagem de sucesso **não**.

## 4. Contrato frontend ↔ servidor

O repositório tem um verificador (`node scripts/verify-contracts.mjs`, usado no job
`Preserve Class A+ surfaces`) que confere se **toda** chamada do frontend casa em método e forma com
uma rota do servidor. Rodado depois das mudanças:

```
CONTRACT VERIFICATION PASSED: all frontend calls match method and route shape.
```

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `npx vitest run src/components/CompanyCampaignSchedulingPanel.test.tsx` | **5/5 PASS** |
| `npx vitest run` (suíte completa) | **83 arquivos, 556 testes, todos PASS** |
| `npx tsc -b` | exit **0** |
| `npm run lint` | **0 erros** (8 avisos pré-existentes de `react-refresh`) |
| `node scripts/verify-contracts.mjs` | **CONTRACT VERIFICATION PASSED** |

Os 5 testes cobrem: declaração de bloqueio com motivo e ausência de selo de "no ar"; estado real do
canal com procedimento de desbloqueio; `Agendar` presente **só** para aprovada e envio das horas
(`scheduleCompanyCampaign("cmp_1", "cmp_aprovada", 2)`); `Registrar métricas` presente **só** para
campanha executada, com números (`impressions: 100, clicks: 5`); e falha exibida como falha.

## 6. Limites declarados

- **Sem publicação real**: `BLOCKED_BY_EXTERNAL_DEPENDENCY` — exige conta autorizada e scopes válidos;
  desbloqueio: definir a variável de credencial do canal e executar teste de conexão real.
- **Sincronizar catálogo envia zero itens** nesta tela: o backend **recusa** com 400 (nada a
  sincronizar) e a interface mostra o motivo. Uma tela que montasse itens sozinha seria inventar
  catálogo; o seletor de itens fica para a próxima rodada, junto com o formulário de SKU/estoque.
- **Sem e2e Playwright novo** para esta tela nesta rodada: a verificação é unitária + contrato; a
  jornada e2e do painel fica declarada.
- **Sem edição de criativo/segmentação**: o escopo é aprovação → agendamento → medição.
