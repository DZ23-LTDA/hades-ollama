# HADES — GATE FINAL DE RELEASE

> Exigido pela seção **11.8**. Este é o **gate GLOBAL**, posterior aos estágios 1–38 — e **não** substitui o
> gate intermediário de engenharia (estágio 22, em [`HADES_ENGINEERING_GATE.md`](HADES_ENGINEERING_GATE.md)).

## Veredito

**`NOT_RELEASE_READY`** — `RELEASE_CANDIDATE` e `COMPLETED_VERIFIED` **não** podem ser declarados
enquanto houver requisito obrigatório não comprovado. O veredito é uma consequência da matriz abaixo, não uma
avaliação subjetiva.

## Matriz de aceite (seção 9 das diretrizes)

Resumo: BLOCKED_BY_EXTERNAL_DEPENDENCY: 3 · DONE: 15 · IN_PROGRESS: 1 · NAO_COMPROVADO: 5

| # | Requisito | Status | Evidência / limite | O que destrava |
| --- | --- | --- | --- | --- |
| 1 | O Hades inicia corretamente | **DONE** | Suítes de runtime e inicialização verdes (ver índice); nenhuma falha de startup pendente | — |
| 2 | A interface funciona | **DONE** | Contrato de frontend verificado por `scripts/verify-contracts.mjs` (job obrigatório do CI) | — |
| 3 | O onboarding está validado | **NAO_COMPROVADO** | Não há teste de jornada de primeiro uso com usuário | Executar jornada de onboarding com evidência (estágio 19) |
| 4 | Modelos locais são descobertos e usados | **DONE** | Descoberta e uso local com evidências no índice (estágio 4) | — |
| 5 | APIs externas configuradas são testadas e usadas | **BLOCKED_BY_EXTERNAL_DEPENDENCY** | B-07: chaves de provedor | Credencial de provedor (estágio 4) |
| 6 | A troca de modelo preserva contexto autorizado | **DONE** | Bootstrap de contexto reconstruído por modelo, com teste (estágio 5) | — |
| 7 | O sistema de permissões é aplicado pelo servidor | **DONE** | Autorização server-side com testes de escopo (estágios 5 e 18) | — |
| 8 | Plugins têm instalação e execução verificáveis | **NAO_COMPROVADO** | Há contrato e testes herméticos; falta execução real de um plugin de terceiro | Instalar e executar plugin real (estágio 8) |
| 9 | MCPs têm instalação e execução verificáveis | **DONE** | MCP local/stdio exercitado com testes de revogação e auditoria (estágio 7) | — |
| 10 | Skills têm ativação e execução verificáveis | **DONE** | Ativação por projeto/agente/modelo com testes (estágio 8) | — |
| 11 | Agentes e subagentes executam missões reais | **DONE** | Orquestração com cotas e artefatos verificáveis (estágio 9) | — |
| 12 | O runtime suporta recuperação e retomada | **DONE** | Checkpoints, cancelamento e retomada testados (estágios 10 e 13) | — |
| 13 | Ferramentas funcionam dentro das permissões | **DONE** | Execução governada com recusa fora de escopo (estágio 7/18) | — |
| 14 | O Builder entrega aplicações utilizáveis | **NAO_COMPROVADO** | App de estoque gerado e testado em sandbox; falta navegação real ponta a ponta | E2E de navegador do app gerado (estágio 12) |
| 15 | O agente de código entrega correções testadas | **DONE** | Fluxo issue→teste→correção→PR comprovado (estágio 11) | — |
| 16 | Workflows executam os nós realmente suportados | **DONE** | Motor determinístico com retry/DLQ e log por nó (estágio 13) | — |
| 17 | Browser/computer-use funciona nos ambientes homologados | **BLOCKED_BY_EXTERNAL_DEPENDENCY** | B-09 residual: Python ausente para o E2E de browser | Ambiente com navegador/Python para o E2E (estágio 14) |
| 18 | A memória respeita os escopos | **DONE** | Isolamento entre organizações com prova em disco (estágio 15) | — |
| 19 | A segurança multitenant está validada | **IN_PROGRESS** | RLS e isolamento com testes; falta prova em banco real (estágio 18) | Banco PostgreSQL real para provar RLS (estágio 18) |
| 20 | A suíte obrigatória está aprovada | **DONE** | Lint, contrato, suíte Go e paridade Linux verdes na rodada mais recente | — |
| 21 | A instalação limpa está homologada | **BLOCKED_BY_EXTERNAL_DEPENDENCY** | B-06: certificado de assinatura e runners de SO | Certificado + runners reais (estágio 20) |
| 22 | A documentação corresponde ao produto | **DONE** | Relatórios de estágio, índice de evidências e limites declarados por módulo | — |
| 23 | Os artefatos de distribuição estão disponíveis | **NAO_COMPROVADO** | Empacotamento existe; falta release assinado publicado | Assinatura e publicação de release (estágio 20) |
| 24 | Não existem bloqueios críticos ou altos sem mitigação aceita | **NAO_COMPROVADO** | 6 bloqueios externos em aberto e limites de severidade MÉDIO declarados nas evidências mais recentes | Resolver ou aceitar formalmente cada bloqueio |

### Como ler os status

- **DONE**: há execução verificada com evidência no índice.
- **IN_PROGRESS**: parte comprovada, parte **não** — o limite está declarado na coluna de evidência.
- **BLOCKED_BY_EXTERNAL_DEPENDENCY**: depende de credencial, conta, certificado, hardware ou serviço
  indisponível. O bloqueio **não** interrompe trabalho independente e o procedimento de desbloqueio está em
  [`HADES_BLOCKERS.md`](HADES_BLOCKERS.md).
- **NAO_COMPROVADO**: não há evidência de execução. Não é "quase pronto": é **não comprovado**.

## Bloqueios externos em aberto

| ID | Descrição | Estágios afetados |
| --- | --- | --- |
| B-02 | Conta real logada em Codex/Claude para homologar harnesses | — |
| B-03 | Tokens reais de canais (Telegram bot token, WhatsApp) | — |
| B-04 | Provedor de telefonia/voz real | — |
| B-05 | Contas reais de marketplace/social/pagamento | — |
| B-06 | Identidade de assinatura de release (certificado) | — |
| B-07 | Chaves de provedor de modelo para homologacao de chamada real | — |

## O que faria o veredito mudar

1. **Fechar os itens `NAO_COMPROVADO`** com execução real e evidência anexada (onboarding, plugin de terceiro,
   E2E do Builder, artefatos de distribuição publicados).
2. **Resolver ou aceitar formalmente** cada bloqueio externo (credenciais de modelo, certificado de assinatura,
   banco real para RLS, ambiente de browser).
3. **Concluir os estágios `IN_PROGRESS`** com os limites que hoje estão declarados.

Nenhuma dessas condições pode ser satisfeita por documentação: todas exigem **execução com evidência**.

## Honestidade de estado

- O release global está `NOT_RELEASE_READY` e permanece assim até a matriz acima fechar.
- `execution` por estágio é **derivado por regra** (ver `execution_rule` no `HADES_MASTER_STATE.json`).
- Evidências sem comando reproduzível ou sem ambiente declarado **não** contam para homologação.
