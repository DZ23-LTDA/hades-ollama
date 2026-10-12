# Mapa de estágios: estado da missão × prompt master (gerado, não editar à mão)

> Gerado por `python scripts/hades_stage_map.py`. O `--check` no CI falha se este arquivo divergir da tabela.

## 1. Por que este mapa existe (E-187)

A seção 5 do prompt master define os 38 estágios. O estado da missão foi construído com **outra numeração**: **37 dos 38** títulos divergem e, **do 24 em diante, o escopo está deslocado**. Sem esta tradução, *"estágio 24 concluído"* tem **dois significados** — `CRM e pipeline de vendas` no estado e `COMMERCE ENGINE` no master — e nenhuma tabela de cobertura pode ser lida como cobertura do documento da missão.

## 2. Situação do pareamento

| Situação | Quantos |
| --- | --- |
| EQUIVALENTE | 1 |
| TITULO_DIFERENTE | 21 |
| ESCOPO_DIVERGENTE | 16 |
| SEM_PAR_NO_MESTRE | 0 |

**Dívida de governança declarada:** 16 estágios com escopo divergente e 0 sem par no master.

## 3. O mapa

| Estado | Título no estado | Mestre | Título no master | Situação | Observação |
| --- | --- | --- | --- | --- | --- |
| 1 | Auditoria total e inventario real | 1 | AUDITORIA TOTAL DO REPOSITÓRIO | **TITULO_DIFERENTE** | mesmo escopo: auditoria do repositório |
| 2 | Estabilizacao e consolidacao da base | 2 | ESTABILIZAÇÃO E CONSOLIDAÇÃO | **TITULO_DIFERENTE** | mesmo escopo: baseline e falhas |
| 3 | Arquitetura canonica dos componentes | 3 | ARQUITETURA UNIVERSAL DO HADES | **TITULO_DIFERENTE** | mesmo escopo: 16 componentes |
| 4 | Model Gateway (roteamento, rotacao, fallback) | 4 | MODEL GATEWAY UNIVERSAL | **TITULO_DIFERENTE** | mesmo escopo |
| 5 | Context Bootstrap / contexto de projeto | 5 | CONTEXTO AUTOMÁTICO PARA QUALQUER IA | **TITULO_DIFERENTE** | mesmo escopo |
| 6 | Harness Runtime (execucao governada) | 6 | HARNESS RUNTIME UNIVERSAL | **TITULO_DIFERENTE** | mesmo escopo |
| 7 | MCP Manager (servidores, ferramentas, escopo) | 7 | MCP MANAGER COMPLETO | **TITULO_DIFERENTE** | mesmo escopo |
| 8 | Plugins e Skills | 8 | PLUGINS E SKILLS | **EQUIVALENTE** | título e escopo coincidem |
| 9 | Agentes e subagentes (delegacao) | 9 | AGENTES E SUBAGENTES ESPECIALIZADOS | **TITULO_DIFERENTE** | mesmo escopo |
| 10 | Mission Control multiagente (missoes, eventos) | 10 | PARIDADE OPERACIONAL COM MANUS | **ESCOPO_DIVERGENTE** | DECISAO_A (COBERTURA PARCIAL COM EVIDENCIA E-214): o ciclo de missao existe e e exercitado de ponta a ponta - 42 testes PASS no subconjunto de missao/timeline/cancelamento com ambiente gravavel, incluindo ciclo REAL de missao com git worktree e MERGE COMMIT (log: 'Mission git worktree cycle completed successfully: merge commit eac8ee9d...'), cancelamento dominando entre instancias, idempotencia de criacao, falha de fila fail-closed, falha de provedor persistindo estado e notificacoes a partir de eventos persistidos. LACUNAS INTERNAS DECLARADAS: a linha do tempo por etapa (agente, ferramenta, evidencia, retentativa e CUSTO) e uma missao autonoma com modelo real NAO estao evidenciadas. Historico: o estado descreve o COMPONENTE Mission Control; o master pede PARIDADE OPERACIONAL COM MANUS (missão autônoma ponta a ponta) |
| 11 | Paridade Codex/Claude (harnesses de codigo) | 11 | PARIDADE COM CODEX, CLAUDE CODE E DEVIN | **TITULO_DIFERENTE** | mesmo escopo |
| 12 | Studio / Builder visual | 12 | PARIDADE COM LOVABLE, BOLT, V0, BASE44 E FLUTTERFLOW | **TITULO_DIFERENTE** | mesmo escopo |
| 13 | Workflow Engine (fluxos, schedules, condicoes) | 13 | MOTOR DE AUTOMAÇÃO ESTILO N8N | **TITULO_DIFERENTE** | mesmo escopo |
| 14 | Browser Operator / computer-use | 14 | BROWSER, DESKTOP E COMPUTER-USE | **TITULO_DIFERENTE** | mesmo escopo |
| 15 | Memoria e RAG local | 15 | MEMÓRIA UNIVERSAL E CONTEXTO PERSISTENTE | **TITULO_DIFERENTE** | mesmo escopo |
| 16 | Canais de IM (WhatsApp/Telegram/inbox unificado) | 16 | CANAIS, VOZ E TEL-AGENT | **ESCOPO_DIVERGENTE** | DECISAO_A (COBERTURA POR UNIAO): o master 16 junta IM + voz/Tel-Agent; o estado cobre IM aqui e voz/telefonia no estagio 29 - a uniao 16+29 cobre o escopo. Historico: o master inclui VOZ e TEL-AGENT neste estágio; no estado, voz/telefonia aparecem adiante |
| 17 | Workspace do agente (threads, artefatos, presenca) | 17 | FRIGG-LIKE AGENT WORKSPACE E COLABORAÇÃO | **TITULO_DIFERENTE** | mesmo escopo (frigg-like) |
| 18 | Seguranca, permissoes e multitenancy | 18 | SEGURANÇA ENTERPRISE E MULTITENANCY | **TITULO_DIFERENTE** | mesmo escopo |
| 19 | UX/onboarding e jornada de primeira execucao | 19 | EXPERIÊNCIA PROFISSIONAL PARA USUÁRIOS LEIGOS | **TITULO_DIFERENTE** | mesmo escopo |
| 20 | Distribuicao e instaladores (Windows/macOS/Linux) | 20 | DEVOPS, INSTALADORES E RELEASE | **TITULO_DIFERENTE** | mesmo escopo |
| 21 | Benchmarks e comparativo competitivo | 21 | BENCHMARK COMPETITIVO E HOMOLOGAÇÃO FINAL | **TITULO_DIFERENTE** | mesmo escopo |
| 22 | Gate de engenharia intermediario | 22 | RELEASE READINESS INTERMEDIÁRIO (GATE DE ENGENHARIA) | **TITULO_DIFERENTE** | mesmo escopo |
| 23 | Company Factory (empresa orquestrada) | 23 | COMPANY FACTORY | **TITULO_DIFERENTE** | mesmo escopo |
| 24 | CRM e pipeline de vendas | 27 | AI SALES WORKFORCE | **ESCOPO_DIVERGENTE** | DECISAO_A (DESLOCAMENTO): CRM/pipeline corresponde ao master 27 (AI SALES WORKFORCE) e a evidencia do estado sustenta esse escopo. Historico: DESLOCAMENTO: o master chama de 24 o COMMERCE ENGINE; CRM/pipeline corresponde ao master 27 (AI SALES WORKFORCE) |
| 25 | Commerce / loja e pedidos | 24 | COMMERCE ENGINE | **ESCOPO_DIVERGENTE** | DECISAO_A (DESLOCAMENTO): e o COMMERCE ENGINE do master 24, com evidencia propria. Historico: DESLOCAMENTO: é o COMMERCE ENGINE do master (24) |
| 26 | Marketplace multi-vendedor | 25 | MARKETPLACE HUB | **ESCOPO_DIVERGENTE** | DECISAO_A (DESLOCAMENTO): e o MARKETPLACE HUB do master 25. Historico: DESLOCAMENTO: é o MARKETPLACE HUB do master (25) |
| 27 | Pagamentos e faturamento | 30 | FINANCE & OPERATIONS | **ESCOPO_DIVERGENTE** | DECISAO_A (COBERTURA PARCIAL COM EVIDENCIA E-213): o escopo FINANCE & OPERATIONS (master 30) tem modulo proprio - internal/agent/finance_ops.go (FinanceReceivable, FinanceDivergence, FinanceReport, PostOrderRevenue, PostOrderCost, ReconcileOrderWithStatement, Report) e company_ledger.go (LedgerEntry, LedgerSummary, RecordLedgerEntry, ListLedgerEntries, ReconcileLedgerEntry, UnreconciledLedgerEntries), com rotas /companies/:id/ledger, /ledger/summary, /ledger/:entry_id/reconcile, /finance/orders/:id/revenue e /cost, e 11 funcoes de teste nos arquivos. LACUNAS INTERNAS DECLARADAS: cobrancas, orcamentos e integracoes fiscais NAO evidenciados. Historico: o master trata pagamentos dentro de COMMERCE ENGINE (24) e FINANCE & OPERATIONS (30); no estado estão isolados |
| 28 | Atendimento e suporte (helpdesk) | 29 | OMNICHANNEL INBOX | **ESCOPO_DIVERGENTE** | DECISAO_A (DESLOCAMENTO): e o OMNICHANNEL INBOX do master 29. Historico: é o OMNICHANNEL INBOX do master (29); o master 28 é VOICE & CALL CENTER, que o estado trata no 29 |
| 29 | Telefonia e call center | 28 | VOICE & CALL CENTER | **ESCOPO_DIVERGENTE** | DECISAO_B (LACUNA DECLARADA): a evidencia ENTREGUE aqui e a CAIXA DE ENTRADA UNIFICADA (master 29), nao o VOICE & CALL CENTER do master 28 - o conteudo nao corresponde ao par declarado, e telefonia real depende de provedor (B-04). Escopo do master 28 sem dono declarado. Historico: é o VOICE & CALL CENTER do master (28) |
| 30 | Marketing e campanhas | 26 | SOCIAL MEDIA OS | **TITULO_DIFERENTE** | é o SOCIAL MEDIA OS do master (26): campanhas, calendário e conteúdo sob aprovação |
| 31 | Social e publicacao | 36 | SOCIAL MEDIA E MARKETPLACE OPERACIONAL | **ESCOPO_DIVERGENTE** | DECISAO_A (COBERTURA PARCIAL COM EVIDENCIA E-213): a operacao de redes do master 36 tem modulo proprio - internal/agent/company_social.go (CompanySocialAccount, CompanySocialDraft, CompanySocialMetric, CompanySocialReport; AddSocialAccount, CreateSocialDraft, ApproveSocialDraft, PublishSocialDraft, RecordSocialMetricWithIdempotency, SocialReport) e company_marketplace.go (ScheduleCampaign, RecordCampaignMetrics, CampaignReport) e marketplace_sync.go (MarketplaceSyncStore, Sync, RemoteStock), com rotas /companies/:id/social/* e /companies/:id/campaigns/*. PUBLICACAO EXIGE APROVACAO. LACUNAS INTERNAS DECLARADAS: respostas por IA em redes e alertas NAO evidenciados. Historico: operação de redes com publicação autorizada e métricas corresponde ao master 36 (SOCIAL MEDIA E MARKETPLACE OPERACIONAL); a parte de MARKETPLACE do master 36 é coberta pelo estado 26 |
| 32 | BI / business status e metricas | 31 | AUTONOMOUS BUSINESS SUPERVISOR | **ESCOPO_DIVERGENTE** | DECISAO_A (COBERTURA PARCIAL DECLARADA): dashboards e indicadores atendem ao monitoramento do master 31. Historico: dashboards/indicadores correspondem ao AUTONOMOUS BUSINESS SUPERVISOR do master (31) |
| 33 | Canal Telegram | 33 | CONSOLIDAR WHATSAPP E TELEGRAM EXISTENTES | **ESCOPO_DIVERGENTE** | DECISAO_A (DIVISAO DECLARADA): o master 33 junta WhatsApp e Telegram; o estado divide em 33 e 34 e a uniao cobre. Historico: o master junta WhatsApp E Telegram em UM estágio (33); o estado separa em 33 e 34 |
| 34 | Canal WhatsApp | 34 | HADES OMNICHANNEL RUNTIME | **ESCOPO_DIVERGENTE** | DECISAO_A (CORRIGIDA COM EVIDENCIA E-188): este estagio carrega o RUNTIME OMNICHANNEL do master 34. Historico: CORRIGIDO COM EVIDENCIA (E-188): este estágio carrega o RUNTIME OMNICHANNEL do master 34 - E-105 aponta `docs/mission/HADES_STAGE_34_OMNICHANNEL.md` e E-141/E-170..E-177 medem dead letter, rate limit, conectores, cache de eventos, reserva em voo e -race. O canal do master 33 é coberto pelo estado 33 |
| 35 | Inbox unificado omnicanal | 35 | AGENTES DE ATENDIMENTO E VENDAS | **ESCOPO_DIVERGENTE** | DECISAO_A (COBERTURA COM EVIDENCIA E-212): o escopo AGENTES DE ATENDIMENTO E VENDAS (master 35) esta implementado em internal/agent/service_agent*.go - ServiceAgentCatalog, NewServiceAgentRuntime, RunTurn, aprovacoes, trilha de auditoria, conhecimento e ferramentas de comercio/empresa, com 85 testes PASS. A lacuna era de ATRIBUICAO no estado, nao de implementacao. Historico: CORRIGIDO COM EVIDENCIA (E-188): este estágio carrega os AGENTES DE ATENDIMENTO E VENDAS do master 35 - E-106 aponta `docs/mission/HADES_STAGE_35_SERVICE_AGENT...` e E-144..E-156 medem `TestServiceAgent` (política, auditoria, aprovação). O OMNICHANNEL INBOX do master 29 é coberto pelo estado 28 |
| 36 | Voz e transcricao | 37 | VOZ E TELEFONIA COM TEL-AGENT | **ESCOPO_DIVERGENTE** | DECISAO_A (DIVISAO DECLARADA): e o master 37, atendido por DOIS estagios do estado (36 e 37); telefonia real segue dependente de provedor (B-04). Historico: VOZ E TELEFONIA COM TEL-AGENT do master (37) |
| 37 | Chamadas e follow-up | 37 | VOZ E TELEFONIA COM TEL-AGENT | **ESCOPO_DIVERGENTE** | DECISAO_A (DIVISAO DECLARADA): SEGUNDO estagio do estado para o MESMO master 37 - divisao declarada, nao colisao. Historico: segundo estágio do estado para o mesmo do master (37) |
| 38 | Gate final de release verificado | 32, 38 | BUSINESS E2E CERTIFICATION + HOMOLOGAÇÃO DE UMA EMPRESA AUTÔNOMA | **ESCOPO_DIVERGENTE** | DECISAO_A (COBERTURA POR UNIAO): o gate final cobre o master 32 e o master 38, como a propria linha declara. Historico: CORRIGIDO COM EVIDENCIA (E-188): este estágio carrega a HOMOLOGAÇÃO DE EMPRESA do master 32 - E-109 e E-112 medem `TestBusinessE2E` (2/2, 21 etapas, com reinício real sobre o mesmo diretório) e `TestBusinessCertification`. O GATE de release global NÃO é um estágio do master: é o requisito transversal da seção 11.8, e por isso não aparece como par |

## 4. Cobertura dos estágios do master

- estágios do master **cobertos** pelo estado: 38 de 38
- estágios do master **sem nenhum** estágio do estado apontando: nenhum
- estágios do master apontados por **mais de um** estágio do estado: {37: 2}

## 5. O que este mapa NÃO faz

- **Não** renumera o estado: renumerar 38 estágios muda o significado de todas as evidências acumuladas, o que não é decisão reversível de baixo risco.
- **Não** afirma que os pareamentos estão certos: cada linha é **julgamento de escopo** por leitura dos dois documentos, e onde equivalência não se sustenta sem inventar, a linha diz `ESCOPO_DIVERGENTE` ou `SEM_PAR_NO_MESTRE` em vez de forçar um número.
- **Não** substitui a matriz do gate: para afirmar cobertura do master é preciso citar esta tradução.

## 6. Procedimento de reconciliação

1. congelar a numeração do estado (que carrega o histórico de evidências);
2. tratar este mapa como a **única** tradução permitida entre os dois;
3. por estágio `ESCOPO_DIVERGENTE`, decidir: (a) manter o escopo do estado e registrar o master como coberto, ou (b) abrir estágio para o escopo do master que não existe;
4. só então usar a matriz do gate para afirmar cobertura do master, citando o mapa.
