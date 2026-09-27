# Company OS — empresas agentic no Ollama Full

O **Company OS** transforma uma organização/tenant em uma unidade empresarial operável pelo runtime agentic. Ele guarda identidade, missão, posicionamento, modelo de negócio, oferta, público-alvo, departamentos virtuais, roadmap, metas, backlog, ciclos de execução, orçamento, riscos e relatórios. A referência de categoria é a proposta pública da Polsia de uma IA que planeja, constrói e opera uma empresa [1]; o Ollama Full implementa uma arquitetura local-first, com providers substituíveis, approvals server-side e isolamento por organização.

## Fluxo implementado

A rota `/company` permite criar uma empresa como tenant local, selecionar empresas existentes, editar identidade e modelo de negócio, visualizar os sete departamentos padrão e criar marcos de roadmap. Também é possível registrar metas/KPIs, adicionar itens ao backlog com prioridade, configurar ciclos diários ou semanais e consultar um relatório operacional.

Cada ciclo cria um schedule persistente com workspace `company://<id>`. O worker agentic transforma o schedule em missão recorrente. Se a empresa estiver pausada, o worker não cria novas missões para aquele ciclo. O usuário pode pausar e retomar a empresa pela interface ou pela API.

## Departamentos padrão

A empresa começa com CEO/Estratégia, Produto, Engenharia, Marketing, Vendas, Suporte e Operações. Eles são estruturas de coordenação, não agentes fictícios: para executar trabalho, cada departamento precisa de uma missão, provider, skill e tool permitidos. Os efeitos externos permanecem sujeitos a approval.

## Orçamento e anomalias

O budget suporta moeda, limite mensal, gasto acumulado e limiar de approval. Gastos em anúncios e contratos exigem approval explícito. O sistema pausa automaticamente a empresa quando o limite é excedido. Anomalias de severidade alta/crítica também pausam o tenant; três anomalias registradas igualmente acionam a pausa. O motivo fica no estado operacional e no relatório.

O guardrail impede que uma interface contorne a política. O endpoint server-side valida o tenant, o valor, a categoria e o approval. A implementação não movimenta dinheiro nem cria contas de anúncios, bancos, marketplaces ou gateways de pagamento.

## Marketing, redes sociais, afiliados e dropshipping

A arquitetura está preparada para acrescentar departamentos e connectors de marketing. O caminho seguro é:

1. pesquisar nicho, concorrentes e termos públicos;
2. criar posicionamento, oferta, calendário editorial e backlog de campanhas;
3. gerar rascunhos de posts, anúncios, páginas, e-mails e criativos;
4. submeter cada publicação, anúncio pago, abordagem comercial ou alteração de catálogo a uma policy de approval;
5. registrar campanha, canal, custo, conversão e attribution;
6. pausar automaticamente quando custo, erro, reclamação, fraude ou anomalia superar o limite.

Afiliados e dropshipping podem ser implementados por connectors allowlisted para plataformas, catálogos, CRM, e-mail, redes sociais, analytics e logística. Nesta revisão eles não são declarados como integrações concluídas: faltam contratos específicos, credenciais, escopos, testes de sandbox, políticas de marca, compliance, devoluções, impostos e validação de pedidos reais.

## Growth OS sandbox

O Company OS agora inclui um **Growth OS local** para testar o ciclo de campanhas, afiliados e dropshipping sem chamar plataformas externas. Uma campanha nasce como `draft`, um programa de afiliados nasce como `pending` e um pedido nasce como `pending_approval`. Aprovar é uma ação separada de iniciar campanha, criar link ou fulfillar pedido.

O módulo registra campanhas, canais, orçamento diário, aprovação, impressões, cliques, conversões e gasto. Programas de afiliados possuem rede e comissão. Links exigem destino HTTPS e programa aprovado. Produtos têm SKU, fornecedor, custo, preço e estoque. Pedidos são associados a cliente e produto; fulfillment sandbox exige aprovação, código de rastreio e estoque suficiente. O relatório `/growth/report` consolida campanhas ativas, conversões, produtos, pedidos e receita registrada.

Esse fluxo é **sandbox local**. Ele não publica anúncios, envia posts, cria contas, acessa marketplaces, compra estoque, cobra clientes, movimenta dinheiro, chama transportadoras ou envia pedidos reais. Connectors reais só podem ser adicionados depois de escopos mínimos, secrets server-side, sandbox do provedor, approval, DLP, idempotência, compliance e testes autorizados.

## API principal

Todas as rotas ficam sob `/api/agent/v1` e exigem o mesmo escopo de organização do runtime:

| Operação | Endpoint |
|---|---|
| Criar/listar empresa | `POST /companies`, `GET /companies` |
| Ler/editar identidade | `GET /companies/:id`, `PATCH /companies/:id` |
| Relatório operacional | `GET /companies/:id/report` |
| Roadmap, KPI e backlog | `POST /companies/:id/roadmap`, `/goals`, `/backlog` |
| Ciclo diário/semanal | `POST /companies/:id/cycles` |
| Pausar/retomar | `POST /companies/:id/pause`, `/resume` |
| Registrar anomalia | `POST /companies/:id/anomalies` |
| Registrar gasto | `POST /companies/:id/spend` |
| Growth report | `GET /companies/:id/growth/report` |
| Campanha e approval | `POST /companies/:id/campaigns`, `/campaigns/:campaign_id/approve`, `/launch`, `/pause` |
| Afiliados | `POST /companies/:id/affiliate-programs`, `/affiliate-links`, `/conversion` |
| Catálogo e pedidos | `POST /companies/:id/products`, `/orders`, `/orders/:order_id/approve`, `/fulfill` |

Os dados locais são persistidos em `OLLAMA_AGENT_STORE/companies`. A integração PostgreSQL existente continua sendo o caminho para missões/eventos distribuídos; a persistência empresarial distribuída e RLS específico de Company OS são próximos gates.

## Estado honesto

O Company OS desta rodada é uma fundação funcional de planejamento e operação controlada. Ele não prova que a empresa gera receita, encontra clientes, fecha contratos, administra anúncios, atende clientes ou envia pedidos sem configuração externa. Para uma operação real, o mantenedor deve fornecer connectors, credenciais, escopos mínimos, ambientes de sandbox, políticas de aprovação e testes de integração. Nenhuma chave deve entrar no Git, na UI ou nos logs.

## Referências

[1]: https://polsia.com/ "Polsia — AI That Runs Your Company While You Sleep"


## Agentes departamentais e Social OS

Cada departamento pode manter um agente persistente com supervisor, escopo de capabilities, orçamento, SLA, memória, estado de pausa e registro de última execução. O supervisor pode pausar ou retomar o agente e o limite de orçamento bloqueia novas ações quando atingido. Esse mecanismo organiza a execução contínua, mas não transforma um agente em pessoa jurídica nem autoriza decisões financeiras sem approval.

O Social OS sandbox mantém contas em estado `oauth_pending` até que o operador complete autorização. Ele suporta drafts, aprovação, publicação simulada e métricas locais. Nenhum token, post, anúncio, pedido ou mensagem externa é criado sem connector configurado, escopo mínimo, approval e teste do canal. Integrações com TikTok Shop, Meta/Instagram, YouTube, WhatsApp, Shopify e marketplaces permanecem adapters dependentes de contas, revisão de app, região, webhooks e compliance.
