# Hades × concorrentes — mapa honesto de capacidades (2026-10-04)

Objetivo do projeto: ser **mais completo e melhor** que as ferramentas da lista
(agentes de código, app builders, automação, roteamento, inferência local).
Este documento é **honesto**: marca onde o Hades já lidera, onde empata e onde
ainda falta — para guiar a execução, não para alegar superioridade sem prova.

Legenda: ✅ já funciona e verificado · 🟡 existe, falta prova/ampliar · ⛔ falta construir.

## O diferencial estrutural do Hades (poucos concorrentes têm os três juntos)
1. **Local-first de verdade** — roda no PC do usuário (base Ollama), sem nuvem obrigatória.
   A maioria (Devin, v0, Bolt, Lovable, Replit, Manus, Cursor-cloud) é SaaS.
2. **Agentic + builder + automação + conectores + segurança por tenant NUM app só** —
   os concorrentes fazem UMA categoria bem; o Hades junta várias.
3. **Segurança levada a sério** — grants deny-by-default, aprovação humana vinculada ao
   diff, SSRF/DNS-pinning, isolamento por organização, OAuth por `(provider, subject)`.

## Matriz por categoria

| Categoria | Concorrentes-chave | Estado no Hades | O que falta p/ LIDERAR |
|---|---|---|---|
| **Chat local + cloud** | Ollama, LM Studio, ChatGPT | ✅ chat local/cloud, pt-BR, instruções personalizadas, web_search | 🟡 multimodal (imagem) só em alguns modelos |
| **Agente de código** | Devin, OpenHands, Aider, SWE-agent, Cursor | 🟡 missões agênticas, git worktree, merge com approval-por-diff, runner de testes | ⛔ issue→PR fim-a-fim, review por hunk, E2E longo provado |
| **App builder** | v0, Bolt, Lovable, Replit, Builder.io, FlutterFlow | 🟢 Studio: **16 componentes**, drag-and-drop, preview instantâneo + export HTML offline, **geração por IA LOCAL** ("descreva → componentes", verificado: qwen2.5-coder gerou 9) ✅ e **edição inline no canvas** (duplo-clique) ✅ — o prompt→app do v0/Bolt, mas local-first | ⛔ qualidade de geração dos modelos de fronteira do v0, templates ricos prontos, componentes com estilo/foto por IA |
| **Automação/workflow** | n8n, dify, Make | 🟢 agendamentos, webhooks (com URL+segredo), DLQ, retry; **editor visual de fluxo por nós** ✅ — grafo trigger/action/condition, conectar-por-clique, anti-ciclo, validação pt-BR; **publicar cria automação real** e **fluxo multi-nó vira plano ordenado executado pelo agente** (missão+HTTP+conector+condição, verificado end-to-end) | ⛔ engine determinístico de grafo (hoje é agente-mediado, não passo-a-passo garantido), amplitude de integrações/gatilhos do n8n |
| **Roteamento LLM** | litellm, omniroute, HarnessRouter, 9router | ✅ roteador multi-provedor (DZ23 Router), harness metadata | 🟡 harness Claude/Codex só macOS (falta ponte Windows) |
| **Conectores/integrações** | n8n (~400), Zapier, dify | 🟡 catálogo amplo; **~67 quick-connect por chave de API** (agora +OpenAI/OpenRouter/Groq/Together/Mistral/Cohere/ClickUp/Vultr/Linode, cada um com verificação read-only validada por teste) ✅; **OAuth real p/ 19 provedores** ✅ | ⛔ amplitude ~400 do n8n, teste live E2E (1 clique), mais serviços de nicho |
| **Browser/computer use** | OpenHands, Devin, Manus | 🟡 Browser Operator sob guards; **prontidão plug-and-play** ✅ — painel em Configurações detecta Python/Playwright/Chromium, guia pt-BR com o comando exato e botão "Preparar automaticamente" (instala o que falta); núcleo + endpoints + UI com testes, painel verificado no navegador | ⛔ computer-use nos 3 SOs, ativação do endpoint exige rebuild do backend |
| **Deep research** | Perplexity | 🟡 pesquisa + citações | ⛔ síntese multi-fonte com proveniência exportável |
| **Canvas/flow** | tldraw, xyflow | 🟡 Studio canvas (pilha) | ⛔ canvas livre com arraste/zoom (estilo tldraw) |
| **Memória/projetos** | — | ✅ projetos, memória local, import ZIP/GitHub com retomada | 🟡 busca semântica entre projetos |
| **Company OS/multiagente** | AutoGen, CrewAI, MetaGPT | 🟡 agentes departamentais, supervisor, orquestração fan-out | ⛔ delegação/handoff visível, workers persistentes distribuídos |
| **Inferência local** | Ollama, vLLM | ✅ base Ollama | 🟡 destravar pull de modelos cloud novos (rebase) |
| **Segurança/privacidade** | poucos fazem | ✅ grants, approval-por-diff, SSRF, tenant, OAuth seguro | 🟡 sandbox forte por SO, DLP |
| **Excluir/gerenciar dados** | — | ✅ excluir chat/projeto/conector/skill/automação/**tarefa** | — |

## Onde o Hades JÁ está à frente da maioria (verificado nesta sessão)
- **Honestidade da UI**: 73 defeitos de "promessa falsa" corrigidos — promete = cumpre.
- **Excluir tudo que cria** (inclui tarefas/missões) — vários concorrentes não deixam.
- **OAuth local real com 13 provedores** (campo de credencial no app + fluxo de login)
  num app desktop local-first — raro; a maioria é SaaS com app OAuth pré-registrado deles.
  13/13 geram a URL de login correta (verificado) + teste de regressão.
- **Studio preview/export instantâneo offline** (Ver agora + Baixar HTML) — v0/Bolt fazem,
  mas Hades faz local e sem servidor; núcleo com testes.
- **Segurança**: 13 findings SEC corrigidos + approval vinculado ao diff; auditoria
  Codex fechada (SEC-01..13, Q-11, UX-01..06/09..13, CI-01..05, Q-02 já unificado,
  Q-03 gating por campo estruturado). SSRF/egress com política única em
  `internal/egresspolicy`.

## Roadmap priorizado para LIDERAR (próximas capacidades, por impacto)
1. **Fechar OAuth E2E** (teste live Google) + ampliar provedores (Notion, HubSpot, Zoom,
   Salesforce, Linear, Asana, Discord, Spotify) → mais conectores que a maioria local.
2. **Harness Claude/Codex/Copilot no Windows** (ponte nativa p/ os CLIs) → usar as
   melhores IAs pagas de dentro do Hades em qualquer SO.
3. **Browser/computer-use plug-and-play** (setup automático do navegador) → paridade
   com OpenHands/Devin/Manus em automação de tela.
4. **Studio: preview ao vivo + arraste real** → paridade com v0/Bolt/Lovable.
5. **Editor visual de automações (nós)** → paridade com n8n/dify.
6. **Rebase do Ollama** → destrava modelos cloud novos + recursos recentes.

> Nenhum item é marcado "pronto/melhor" sem verificação. O avanço é capacidade por
> capacidade, cada uma testada. Este mapa é atualizado conforme cada uma fecha.
