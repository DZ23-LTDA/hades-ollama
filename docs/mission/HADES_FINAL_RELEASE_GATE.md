# HADES — GATE FINAL DE RELEASE

> Exigido pela seção **11.8**. Este é o **gate GLOBAL**, posterior aos estágios 1–38 — e **não** substitui o gate
> intermediário de engenharia (estágio 22, em [`HADES_ENGINEERING_GATE.md`](HADES_ENGINEERING_GATE.md)).
> **Gerado por `scripts/hades_release_gate.py`** — rode com `--check` para conferir.

## Procedência

🚨 **E-552 + E-554: *o commit medido tem de estar ESCRITO, ou o documento não se pode reproduzir nem contestar.***

| Campo | Valor |
| --- | --- |
| **Commit medido** (`measured_commit`) | `3b11e7a334f07835d2ef561cecf74f6342feeca5` |
| **Commit da `main`** (`git.main_sha`) | `4b3e21854` |
| **Gerado em** (`measured_at`) | 2026-10-19T09:47:53-03:00 |
| **Snapshot real** (`gate_head`) | `3b11e7a334f07835d2ef561cecf74f6342feeca5` |
| **Ramo do snapshot** (`gate_branch`) | `fix/checker-relogio` |
| **Entradas nao commitadas** (`gate_dirty`) | 20 |
| **O gate prova o snapshot candidato?** (`gate_snapshot`) | **MESMO** |

*Dois campos, e não um: o `E-552` mediu que este derivado **não** citava commit nenhum, e que o
`HADES_BASELINE_REPORT.md` citava **só** a `main` — *o que não está errado (*ele mede o baseline, e o baseline é da
`main`*), mas era **ambíguo**: «commit de referência» não dizia **se** era o medido ou o comparado* ⇒ *agora a
ambiguidade desaparece **por construção**.*

## Veredito

**`NOT_RELEASE_READY`** — `RELEASE_CANDIDATE` e `COMPLETED_VERIFIED` **não** podem ser declarados
enquanto houver requisito obrigatório não comprovado. O veredito é consequência da matriz abaixo, não avaliação
subjetiva.

## Matriz de aceite (seção 9 das diretrizes)

Resumo: BLOCKED_BY_EXTERNAL_DEPENDENCY: 5 · DONE: 19

| # | Requisito | Status | Evidência / limite | O que destrava |
| --- | --- | --- | --- | --- |
| 1 | O Hades inicia corretamente | **DONE** | Suítes de runtime e inicialização verdes; nenhuma falha de startup pendente | — |
| 2 | A interface funciona | **DONE** | Contrato de frontend verificado por `scripts/verify-contracts.mjs` (job obrigatório do CI) | — |
| 3 | O onboarding está validado | **DONE** | E-182: jornada de primeira execução (instalação → onboarding → persistência → reinício → atualização → rollback) executada em **navegador real** sobre o SPA real com backend HTTP real (`npx playwright test --project=first-run` → **2 passed**), e o **CI já a roda** no passo `Web E2E — primeira execução` | — |
| 4 | Modelos locais são descobertos e usados | **DONE** | Descoberta e uso local com evidências no índice (estágio 4) | — |
| 5 | APIs externas configuradas são testadas e usadas | **BLOCKED_BY_EXTERNAL_DEPENDENCY** | B-07: chaves de provedor | Credencial de provedor (estágio 4) |
| 6 | A troca de modelo preserva contexto autorizado | **DONE** | Bootstrap de contexto reconstruído por modelo, com teste (estágio 5) | — |
| 7 | O sistema de permissões é aplicado pelo servidor | **DONE** | Autorização server-side com testes de escopo (estágios 5 e 18) | — |
| 8 | Plugins têm instalação e execução verificáveis | **BLOCKED_BY_EXTERNAL_DEPENDENCY** | **Provado (30 testes PASS)**: instalação com verificação de assinatura, rejeição de conteúdo adulterado, rollback que restaura a versão anterior UMA vez, isolamento por organização, sete caminhos de recusa e **delegação ao sandbox governado** com trilha de auditoria. **E-293**: o desfecho da execução passou a ser **afirmado, não tolerado** (o teste anterior aceitava sucesso OU falha). Medido neste host: `sandbox.exec requires the Linux namespace executor; host-process fallback is disabled` — o Hades **recusa cair para processo de host** (fail-closed, correto por segurança), logo a execução real é `BLOCKED_BY_EXTERNAL_DEPENDENCY` | Executar em **Linux/WSL2 com o executor de namespace** (mantendo o fallback de host desabilitado) |
| 9 | MCPs têm instalação e execução verificáveis | **DONE** | MCP local/stdio exercitado com testes de revogação e auditoria (estágio 7) | — |
| 10 | Skills têm ativação e execução verificáveis | **DONE** | Ativação por projeto/agente/modelo com testes (estágio 8) | — |
| 11 | Agentes e subagentes executam missões reais | **DONE** | Orquestração com cotas e artefatos verificáveis (estágio 9) | — |
| 12 | O runtime suporta recuperação e retomada | **DONE** | Checkpoints, cancelamento e retomada testados (estágios 10 e 13). **E-291**: a auditoria encontrou e corrigiu um **DEFEITO REAL** desta linha — o claim de agendamento decidia pela cópia em memória, e dois processos sobre a mesma pasta de dados **disparavam o mesmo efeito externo duas vezes**; o defeito foi **provado pelo teste antes da correção** e o claim passou a decidir sobre estado fresco. Limite residual: o frescor é por **releitura do arquivo** (sem trava entre processos no claim) | — |
| 13 | Ferramentas funcionam dentro das permissões | **DONE** | Execução governada com recusa fora de escopo (estágios 7 e 18) | — |
| 14 | O Builder entrega aplicações utilizáveis | **DONE** | E-178: app do Studio criado, **publicado**, **servido por HTTP (200)** e **aberto em navegador real**, que renderizou 3 componentes no canvas com **0 erros de console** (`docs/mission/evidence/`). **LIMITE**: app de **demonstração** (heading/table/button) — **não** é full-stack com banco/auth/CRUD | App full-stack com banco, autenticação e CRUD navegado ponta a ponta (estágio 12) |
| 15 | O agente de código entrega correções testadas | **DONE** | Fluxo issue → teste → correção → PR comprovado (estágio 11) | — |
| 16 | Workflows executam os nós realmente suportados | **DONE** | Motor determinístico com retry/DLQ e log por nó (estágio 13) | — |
| 17 | Browser/computer-use funciona nos ambientes homologados | **BLOCKED_BY_EXTERNAL_DEPENDENCY** | B-09 residual: Python ausente para o E2E de browser | Ambiente com navegador/Python para o E2E (estágio 14) |
| 18 | A memória respeita os escopos | **DONE** | Isolamento entre organizações com prova em disco (estágio 15). **E-289/E-290**: a escrita passou a **recusar** sobrescrever trabalho de outro escritor — **intenção declarada por caminho** (não-destrutivos guardados, destrutivos por intenção) e detecção também por **revisão**. Limites: os 3 caminhos destrutivos seguem **sem guarda por intenção**; a trava do arquivo é **ADVISORY**; sem **fila de escritas** | — |
| 19 | A segurança multitenant está validada | **DONE** | **E-294 — lastro medido**: o job de CI *PostgreSQL RLS, Redis DLQ and OTLP integration* sobe um **PostgreSQL 16 real** com **três papéis distintos** (admin/runtime/migrator, senhas geradas por execução), restringe a autenticação **por papel** via `pg_hba`, roda as migrações e executa com `-tags integration`: `TestDistributedPostgresRLSAndEvents`, `...RuntimeRejectsRLSDrift` (**guarda**: recusa quando o RLS desvia), `...RuntimeCannotConnectOutsideApplicationDatabase` (**sonda negativa** de permissão), `...TenantKeyRotation` e a suíte `TestDistributed` completa (**32 testes**). Limite: roda no runner Linux do CI, não no host de desenvolvimento, e **não** mede inferência por timing | — |
| 20 | A suíte obrigatória está aprovada | **DONE** | Lint, contrato, suíte Go e paridade Linux (com detector de corrida) verdes na rodada mais recente | — |
| 21 | A instalação limpa está homologada | **BLOCKED_BY_EXTERNAL_DEPENDENCY** | B-06: certificado de assinatura e runners de SO | Certificado + runners reais (estágio 20) |
| 22 | A documentação corresponde ao produto | **DONE** | Relatórios de estágio, índice de evidências e **limites declarados por módulo** — inclusive os mais recentes (artefato gerado, memória, claim de agendamento). **Limites declarados não são omissões: são afirmações medidas**, e o índice registra também os **resultados negativos** (E-288) | — |
| 23 | Os artefatos de distribuição estão disponíveis | **BLOCKED_BY_EXTERNAL_DEPENDENCY** | **E-295 — lastro medido**: existe release publicada (**`v0.2.0-rc.2`**, `HadesSetup.exe` com **SHA-256 documentado**), **porém não assinada** e **somente Windows**. O status honesto é bloqueio externo, não *não comprovado*: **existe artefato publicado com hash**; o que falta é **certificado** (B-06) e **runners de SO** | Certificado de assinatura + runners Linux/macOS |
| 24 | Não existem bloqueios críticos ou altos sem mitigação aceita | **DONE** | **E-295 — lastro medido**: checagem **online** do repositório devolve **zero PRs abertos** ⇒ **B-01** (*“#56–#71 abertos”*) está **RESOLVIDO** e o documento estava desatualizado. **Nove bloqueios já resolvidos** (B-09, B-11–B-13, B-15–B-18). Os remanescentes são **externos** (B-02–B-07, cada um com procedimento de desbloqueio), **cobertura declarada** (B-10: 104/122 conectores exigem operador, classificado `APENAS_CATALOGO` honestamente) e **ambiente** (B-14: execução de plugin exige privilégio de montagem, **fail-closed**). **Não há defeito CRÍTICO ou ALTO sem mitigação** | — |

### Como ler os status

- **DONE**: há execução verificada com evidência no índice.
- **IN_PROGRESS**: parte comprovada, parte **não** — o limite está declarado na coluna de evidência.
- **BLOCKED_BY_EXTERNAL_DEPENDENCY**: depende de credencial, conta, certificado, hardware ou serviço
  indisponível. O bloqueio **não** interrompe trabalho independente e o procedimento está em
  [`HADES_BLOCKERS.md`](HADES_BLOCKERS.md).
- **NAO_COMPROVADO**: não há evidência de execução. Não é "quase pronto": é **não comprovado**.

## Bloqueios externos em aberto

| ID | Descrição | Detalhe |
| --- | --- | --- |
| B-02 | MEDIDO (E-204/E-205), nao suposto. O que o B-02 dizia ('conta real logada em Codex/Claude') esta parcialmente vencido: o CODEX esta INSTALADO e LOGADO (metodo c | ver HADES_BLOCKERS.md |
| B-03 | Tokens reais de canais (Telegram bot token, WhatsApp) | ver HADES_BLOCKERS.md |
| B-04 | Provedor de telefonia/voz real | ver HADES_BLOCKERS.md |
| B-05 | Contas reais de marketplace/social/pagamento | ver HADES_BLOCKERS.md |
| B-06 | Identidade de assinatura de release (certificado). ESTADO MEDIDO (E-201): a autenticacao de NOTARIZACAO foi PROVADA com a chave de API do App Store Connect (JWT | ver HADES_BLOCKERS.md |
| B-07 | MEDIDO (E-209), nao suposto. O B-07 dizia 'chaves de provedor de modelo para homologacao de chamada real'. Medido: 19 provedores com credencial PRESENTE e habil | ver HADES_BLOCKERS.md |
| B-19 | (1) Chrome real exige a Chrome Web Store (conta + revisao) -- o `--load-extension` foi removido e o service_worker nao acorda; o Chromium 156 e o Edge 155 passa | ver HADES_BLOCKERS.md |

## O que faria o veredito mudar

1. **Fechar os itens `NAO_COMPROVADO`** com execução real e evidência anexada.
2. **Resolver ou aceitar formalmente** cada bloqueio externo.
3. **Concluir os estágios `IN_PROGRESS`** com os limites que hoje estão declarados.

Nenhuma dessas condições pode ser satisfeita por documentação: todas exigem **execução com evidência**.

## Honestidade de estado

- O release global está `NOT_RELEASE_READY` e permanece assim até a matriz acima fechar.
- `execution` por estágio é **derivado por regra** (ver `execution_rule` no `HADES_MASTER_STATE.json`).
- Evidências sem comando reproduzível ou sem ambiente declarado **não** contam para homologação.
- Este arquivo é **gerado**: alterar à mão cria divergência que o `--check` acusa.
