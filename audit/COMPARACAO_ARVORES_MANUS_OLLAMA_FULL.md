# Comparação das árvores Manus e Ollama Full

## Conclusão executiva

Esta comparação indica **paridade parcial em todos os dez domínios analisados**, sem domínio marcado como falho no mapa fornecido e sem base para declarar paridade funcional ampla. Há blocos relevantes do Ollama Full validados localmente — especialmente runtime agentic, chat local, automação, pesquisa, grants e alguns controles de artifacts —, mas muitas jornadas dependem de adapters, credenciais, serviços externos, staging distribuído, hardware ou testes E2E ainda não executados. A superfície visível descrita para o Manus é ampla, porém a sua presença na árvore não certifica que cada fluxo funcione ponta a ponta.

O ponto mais importante é metodológico: **o lado Manus foi tratado como uma árvore observável de funcionalidades e jornadas de produto, não como uma arquitetura interna reverse-engineered**. Menus, nomes de telas, rotas e itens de navegação são evidência de superfície documentada; não são prova independente de backend, isolamento, confiabilidade, acessibilidade ou execução de cada recurso. Do mesmo modo, um adapter, um contrato de API ou um teste local no Ollama não equivale a uma integração externa ou a uma jornada real validada.

O alvo do Ollama Full é, em vários aspectos, **mais amplo que a prova atual**: explicita local-first, controles de segurança por tenant e recurso, workflows duráveis, observabilidade operacional, mobile, CI/CD, backup/restore, connectors e A2A. Isso demonstra direção de produto, não superioridade já comprovada. A comparação correta, portanto, é entre a superfície Manus documentada e o estado de evidência Ollama disponível — não entre o alvo aspiracional do Ollama e uma suposta implementação interna do Manus.

As prioridades mais urgentes são:

1. **P0 — segurança e controle antes de efeitos externos:** autorização por tenant/recurso, deny-by-default, approvals vinculadas à ação, secrets/redaction, sandbox/egress e modo local-only.
2. **P0 — fechar a jornada vertical principal:** Nova tarefa → missão → timeline → approval → artifact, com estados reais, falhas, retry, cancelamento e recuperação.
3. **P0 — fechar o shell utilizável:** navegação de workspace/projeto, conta/settings, composer e estados responsivos/acessíveis, sem tratar rotas ou menus como prova.
4. **P0/P1 — integrações e providers reais sob controle:** OAuth, MCP externo, picker de modelos, fallback e credenciais, sempre com isolamento e auditoria.
5. **P1 — durabilidade operacional:** workers persistentes, execução distribuída, checkpoints, replay, idempotência, recuperação e smoke E2E.
6. **P1 — jornadas de browser/pesquisa e repo-first/builder:** proveniência verificável, runner/repair loop, Studio contra backend real, preview/export e publicação controlada.

Não é calculada uma porcentagem geral: os domínios têm naturezas, evidências e riscos diferentes, e uma média ocultaria bloqueios críticos como segurança, durabilidade e jornada vertical.

## Escopo, fontes e metodologia

### O que foi comparado

O relatório sintetiza os dez resultados estruturados fornecidos para:

- Workspace, identidade e shell;
- Nova tarefa, chat e modelos;
- Agentes, missões e workflows;
- Engenharia de software e Git;
- Browser, computer use, pesquisa e conhecimento;
- Skills, plugins, MCP e integrações;
- Agendado, tarefas e automação;
- Biblioteca, projetos, builders e artifacts;
- Segurança, privacidade e controle;
- Company OS, mobile e operação.

As referências mencionadas nos resultados são, principalmente, `PARITY_MATRIX.md`, `PRODUCT_TREE.md` e `OLLAMA_FULL_PRODUCT_VISION.md`, além das classificações de evidência e lacunas associadas a cada domínio.

### Como a evidência foi lida

A matriz fornecida distingue níveis que não devem ser misturados:

- **VALIDADA LOCALMENTE:** há implementação e teste/execução reproduzível no ambiente local descrito, mas isso não prova operação distribuída, integração externa, produção ou jornada E2E com dados reais.
- **ADAPTER IMPLEMENTADO:** existe uma camada de integração ou contrato preparado, mas ainda faltam credenciais, provider, IdP, serviço, hardware ou smoke real. Adapter não é integração comprovada.
- **IMPLEMENTADA CONDICIONALMENTE:** a capacidade existe no runtime sob dependências e permissões explícitas; faltam validações de produção, modalidade, provider, custo, hardware ou ambiente.
- **PARCIAL:** há superfície, rota, fatia de fluxo ou base de implementação, mas faltam estados, telas, integração e prova da jornada completa.
- **PROTOTIPADO / NÃO INTEGRADO:** existe um protótipo ou teste isolado, mas ele não está ligado ao runtime ou ao ciclo operacional que seria necessário para declarar capacidade pronta.
- **PENDENTE / GATE FALTANTE:** a documentação identifica uma prova, dependência ou etapa ainda não concluída.

A análise separa três perguntas:

1. **O que é descrito como visível no Manus?** A árvore de produto é lida como inventário de superfícies e jornadas documentadas.
2. **O que existe como evidência atual no Ollama?** São considerados os níveis acima, sem elevar validação local a E2E.
3. **Qual é o alvo do Ollama e qual é a lacuna?** O alvo é registrado como intenção/escopo, nunca como prova retroativa de implementação.

### Limitações e honestidade da comparação

Esta é uma **comparação baseada nos documentos e resultados fornecidos**. **Não é uma auditoria live recém-executada do Manus, nem um teste de runtime do Manus ou do Ollama realizado para este relatório.** Não foram inferidos estados não registrados, nem foram transformados itens de menu em capacidades funcionais.

Em particular:

- Não foi feita inspeção do código ou dos serviços internos do Manus.
- Não se afirma que uma função ausente da árvore não exista no Manus; apenas que ela não é comprovada pelos materiais observados.
- Não se afirma que uma função presente na árvore funcione em todos os casos.
- Não se afirma que testes locais representem staging, produção, hardware real ou serviços externos.
- Não se atribui um número único de paridade.
- Os resultados fornecidos não registram domínios falhos; por isso, `failed_domains` é zero, sem significar que não existam bloqueios ou riscos.

## Tabela comparativa

| Domínio | Capacidade Manus visível | Status atual do Ollama | Paridade / lacunas | Prioridade |
|---|---|---|---|---|
| **Workspace, identidade e shell** | Seletor de workspace/projeto, conta/perfil, plano/uso, preferências, ajuda, sair; navegação global, Nova tarefa, Agentic Console, Biblioteca, Projetos, Agendado, Tarefas e Company OS. Composer descrito com texto, arquivos/imagens, fontes, voz/transcrição e modelo/agente. É superfície observável, não prova de funcionamento. | **Parcial.** Agentic Console e rotas-base existem; auth/RBAC/SSO é adapter; memória/projetos e chat têm validação local; shell, estados de tela, responsividade, acessibilidade e E2E ainda faltam. | Não comprovados: jornada real de workspace/projeto e conta/settings, quotas/custos visíveis, pesquisa global, command palette, atalhos, notificações, layout/painéis, composer completo e WCAG. | **P0** — é a porta de entrada e o gate da jornada vertical; segurança, conta, uso e acessibilidade devem ser critérios de conclusão. |
| **Nova tarefa, chat e modelos** | Nova tarefa com instruções, arquivos/imagens, referências/fontes, voz/transcrição, escolha de modelo/agente, envio/cancelamento/retry; ações rápidas, recomendações, drafts/histórico e chat/timeline. A árvore não certifica cada ação. | **Parcial.** Chat local e APIs/modelos locais validados; Claude/Codex, OmniRoute e HarnessRouter são adapters; multimídia condicional; pesquisa/routing locais validados, mas faltam dados reais e jornada E2E. | Falta fechar Nova tarefa → missão/timeline/approval/artifact; picker conectado a credenciais/capacidades, modos/templates/drafts, anexos/fontes/voz, falhas/retry, fallback local-only e smokes externos. | **P0** para a jornada vertical; **P1** para providers, routing, multimodal e fontes reais. |
| **Agentes, missões e workflows** | Missões ativas/background, plano/subtarefas, timeline de eventos, approvals, recuperação, artifacts, compartilhamento, parar/retomar, retry e colaboração descrita. A árvore não explicita checkpoints, durabilidade ou especialistas. | **Parcial.** Runtime, planner, ferramentas, eventos, recovery, artifacts e approvals validados localmente; automação e agentes departamentais locais; UI parcial; colaboração é adapter; operação distribuída e E2E longa faltam. | Persistência/execução distribuída, restart/checkpoint/replay sem duplicação, timeline causal real, handoff/takeover, delegação/fan-out/fan-in, workers persistentes e CRDT ainda não comprovados. | **P1** — durabilidade e recuperação são o gate de confiança para missões longas; colaboração depois. |
| **Engenharia de software e Git** | Terminal/código, plano, timeline, correção/recuperação, approvals, artifacts, versionamento/diff na Biblioteca. A árvore não declara explicitamente worktree, runner, issue-to-PR, review ou commit. | **Parcial.** `git.repo.inspect` read-only e escrita/patch com diff e approval hash-bound validados localmente; snapshot Git é protótipo standalone, não integrado; ciclo repo-first, runner, repair loop e PR faltam. | Não provados: snapshot/worktree integrado, CAS cross-process, crash atomicity, testes no sandbox, recuperação após crash, review/rollback por hunk, issue→branch→PR, CI vinculado e E2E real. | **P1** — completar repo-first seguro antes de habilitar efeitos Git externos. |
| **Browser, computer use, pesquisa e conhecimento** | Browser/computer use e pesquisa/citações no Agente; composer com arquivos/fontes; Biblioteca, busca, Projetos com fontes/memória e browser preview. A árvore não comprova browser, ingestão ou proveniência em execução. | **Parcial.** Browser Operator e pesquisa profunda validados localmente sob guards; upload/download, takeover, desktop físico e fontes externas pendentes; mídia condicional; memória local validada sem benchmark distribuído. | Falta E2E de search/fetch/citação/proveniência/exportação, Chromium e jornadas externas controladas, bypass não-TCP, computer use nos três SOs, ingestão PDF/DOCX/XLSX/imagem e qualidade de recuperação. | **P1** — fechar pesquisa/browser verificável; hardware e integrações dependem do ambiente. |
| **Skills, plugins, MCP e integrações** | Catálogo/instaladas/por projeto, manifestos, versões, permissões, teste, enable/disable, customizadas; plugins, OAuth, scopes, secrets, conectores, MCP/A2A e Desktop Commander são superfícies descritas. Isso não prova lifecycle nem isolamento. | **Parcial.** Núcleo de manifestos, scopes, allowlists, MCP stdio/Remote MCP, enable/disable/remove e approval validado localmente; Composio, Desktop Commander e OAuth/connectors são adapters; tenant/contas reais faltam. | Prioridade em isolamento por tenant/recurso, deny-by-default, testes negativos, OAuth/refresh/revogação, secrets server-side, lifecycle, health/logs/rollback, MCP externo, PKCE/pairing e A2A. | **P0** para autorização, secrets e isolamento; **P1** para OAuth/MCP/lifecycle; **P2** para A2A e amplitude; **P3** para catálogo/polish. |
| **Agendado, tarefas e automação** | Agendamentos únicos/recorrentes, timezone, webhooks/eventos, retry/DLQ, pause/resume, histórico; Inbox, estados, falhas, reabrir/duplicar/arquivar; runtime com eventos/recovery. A árvore não prova semântica operacional. | **Parcial.** Scheduler, webhooks, retries, DLQ e replay validados localmente; Redis workers/DLQ é adapter; workers persistentes, smoke externo e UI completa faltam. | Falta execução distribuída durável, restart, idempotência, delivery real de webhook, backoff/DLQ/replay, timezone e UI de Agendado/Tarefas/Inbox com timeline causal. | **P1** — workers/recovery e E2E antes de equivalência; depois UI e timezone. |
| **Biblioteca, projetos, builders e artifacts** | Biblioteca por tipos, busca/filtros, preview, versionamento/diff, exportação, compartilhamento; Projetos com contexto, arquivos/fontes, memória, membros, tarefas, artifacts; builders, canvas, drag-and-drop, undo/redo, preview, colaboração e deploy. Inventário não prova execução. | **Parcial.** Manifest/hashes/versões/download e backend local de canvas/undo/redo/export/publicação; Studio tem Playwright com API mockada; deploy é adapter; memória local e UI geral parciais; realtime/CRDT faltam. | Falta E2E Studio/backend real, editor visual integrado, biblioteca navegável com permissões/arquivos grandes, projeto com recuperação medida, deploy real com health/rollback e colaboração CRDT. | **P1** — provar builder/artifact completo; publicação só com approval/health/rollback. |
| **Segurança, privacidade e controle** | Workspace/projeto, perfil/uso, approvals, membros/papéis, scopes/consentimento e permissões de compartilhamento. A árvore não comprova auth, sandbox, DLP, retenção, auditoria ou local-only. | **Escopo-alvo supera a observabilidade Manus, mas a prova é parcial.** Grants deny-by-default, approvals, Browser e SSRF guards têm validações locais; auth/RBAC/SSO é adapter; sandbox forte, DLP, secrets manager, auditoria completa, RLS staging e local-only E2E faltam. | É pré-condição, não apenas feature: IdP/RLS real, cross-tenant/IDOR, replay de approval, sandbox por SO, egress/SSRF não-TCP, uploads, redaction/DLP, retenção/export/delete e ausência de fallback remoto precisam ser provados. | **P0** — antes de autonomia, conectores e efeitos externos. |
| **Company OS, mobile e operação** | Empresa/Company OS, membros/papéis, tarefas, agendamento, approvals, auditoria/feedback e execução são superfícies descritas. A árvore não declara mobile, SLOs, traces, backup/restore ou CI/CD. | **Parcial.** Company OS e agentes departamentais validados localmente; mobile é parcial; observabilidade é adapter; operação distribuída, backup/restore e CI/CD/distribuição não têm prova suficiente. | Falta staging distribuído, connectors/compliance externos, push/offline/reconciliação em dispositivos, Collector/Redis/Postgres, SLOs/alertas, recuperação, backup/restore, SBOM e releases assinados/rollback. | **P1** para operação integrada, recuperação e backup/restore; **P2** para lojas/instaladores e polish multi-OS. |

## Leitura detalhada por domínio

### 1. Workspace, identidade e shell

A árvore observável do Manus descreve uma experiência de entrada relativamente abrangente: seletor de workspace/projeto, conta/perfil, plano, créditos, uso, limites, preferências, ajuda e navegação para Agente, Habilidades, Plugins, Agendado, Biblioteca, Projetos, Tarefas e Company OS. Também descreve a Nova tarefa como um composer com texto, arquivos/imagens, referências/fontes, voz/transcrição e seleção de modelo/agente.

No Ollama, a base de shell e o Agentic Console são parciais, e há adapters de organizações, RBAC, OAuth/OIDC, SAML, MFA e RLS. Memória/projetos e chat foram validados localmente em backend/APIs, mas isso não prova uma UX completa de projeto ou uma jornada visual com dados reais. A matriz não apresenta prova de quotas, créditos/custos por provider visíveis e confiáveis, pesquisa global, command palette, atalhos, notificações, sidebar recolhível, layouts ou preferências.

O fechamento recomendado é uma sequência vertical que comece pela Nova tarefa e atravesse missão, timeline, approval e artifact. A conclusão do shell deve incluir estados reais, falhas, responsividade desktop/tablet/web e acessibilidade — teclado, foco, leitor de tela e reduced motion —, em vez de considerar uma rota HTTP ou um item de menu suficiente.

### 2. Nova tarefa, chat e modelos

O Manus é descrito com uma entrada multimodal e ações rápidas, mas a fonte disponível não permite declarar que cada modo, template, draft, fonte ou integração funcione. O Ollama tem chat local e compatibilidade de APIs/modelos locais validados; adapters para Claude, Codex, OmniRoute e HarnessRouter ainda requerem credenciais, instâncias ou smoke autorizado. A multimídia está implementada condicionalmente com approval por etapa e zero requests antes da aprovação, mas providers, hardware, Tesseract, disponibilidade e custo/quota continuam dependências.

A lacuna decisiva não é apenas adicionar controles ao composer. É demonstrar a jornada completa: selecionar entrada, fonte, provider/modelo e modo; iniciar missão; observar streaming/timeline; lidar com erro, cancelamento, retry e recuperação; aprovar; obter artifact. Também é necessário exibir políticas de fallback, especialmente a garantia de `Local-only`, e mostrar provider/modelo, latência, falha e limites sem expor prompts ou segredos.

### 3. Agentes, missões e workflows

A superfície Manus observável inclui missões ativas e em background, planos, subtarefas, timeline, approvals, recuperação, artifacts e colaboração. Isso permite comparar jornadas, mas não permite atribuir checkpoints, execução durável ou agentes especialistas como internals do Manus, porque não estão explicitados na árvore.

O runtime Ollama tem validações locais de planner, ferramentas, eventos, recovery, artifacts e approvals. A automação e agentes departamentais também têm validação local. Porém, a execução distribuída persistente, trabalhadores em staging, jornada longa E2E, restart/checkpoint/retomada/replay e garantias contra efeitos duplicados continuam gates. Comentários, presença e snapshots são adapter; realtime/CRDT e autorização por tenant ainda não foram demonstrados.

O próximo gate deve ser uma missão durável em que um worker falha e retoma sem duplicar efeitos, preservando approvals, eventos causais, artifacts e estado visível. Só depois faz sentido declarar equivalência para workflows longos ou priorizar colaboração avançada.

### 4. Engenharia de software e Git

A árvore Manus permite observar terminal/código, plano, timeline, correção/recuperação, approvals, artifacts e versionamento/diff na Biblioteca. Não há evidência documental suficiente para atribuir ao Manus worktree/branch, runner de testes, issue-to-PR, review ou commit como capacidades específicas.

No Ollama, o read-only de repositório e operações de escrita/patch com diff e approval hash-bound foram validados localmente. O snapshot físico Git, entretanto, é protótipo standalone não integrado ao Runtime/ToolContext e sem lifecycle operacional. Permanecem CAS cross-process, atomicidade contra crash/processos externos, runner dentro do sandbox, repair loop, ciclo branch/worktree, review/commit/PR opt-in, CI/resultados vinculados e E2E repo-first com backend real.

A recomendação é preservar o princípio de aprovação antes de qualquer commit/PR e provar o ciclo completo em um repositório real, incluindo estado dirty, falha de teste, reparo, cancelamento, retomada e cleanup.

### 5. Browser, computer use, pesquisa e conhecimento

A árvore Manus apresenta browser/computer use, pesquisa/citações, arquivos/fontes, Biblioteca, Projetos, memória e browser preview como superfícies relacionadas. Ela não comprova, por si só, que upload/download, citações ou ingestão de cada formato funcionem.

O Ollama tem Browser Operator localmente validado com proxy de egress, IP pinning e bloqueios descritos, e pesquisa profunda localmente validada com cache, citações, robots e SSRF guard. Ainda faltam jornadas externas controladas, Chromium instalado e validado, upload/download, takeover, bypass de protocolos não-TCP, computer use físico em Linux/macOS/Windows e fontes reais. A mídia é condicional; não há prova documental de ingestão completa de PDF, DOCX, XLSX e imagens, nem de uma base de conhecimento E2E.

A primeira entrega de alto valor é pesquisa com `search/fetch`, citação ligada à afirmação, proveniência, síntese, exportação, limites e retomada. Depois devem vir a validação de hardware e a medição de recuperação semântica/exata por projeto/organização, com ACL e storage distribuído.

### 6. Skills, plugins, MCP e integrações

A árvore Manus lista catálogo de habilidades, manifests, versões, permissões, enable/disable, customização, plugins, OAuth, scopes, secrets, conectores, MCP/A2A e Desktop Commander. Como nos outros domínios, isso documenta uma superfície, não a execução de cada serviço ou a eficácia das políticas.

O Ollama tem evidência local para manifests não confiáveis por padrão, scopes, allowlists, MCP stdio/Remote MCP, enable/disable/remove e approval por ferramenta. Composio, Desktop Commander, OAuth real, contas conectadas e integrações por tenant são adapters ou gates. A prioridade é validar autorização e isolamento antes de ampliar catálogo: cross-tenant/IDOR, scopes mínimos, deny-by-default, revogação, secrets server-side e auditoria por chamada.

Só depois devem ser habilitados OAuth/refresh/revogação reais, MCP remoto e lifecycle de instalação/atualização/rollback. A2A, assinatura/health/rollback e amplitude dos conectores são etapas posteriores, não capacidades já demonstradas.

### 7. Agendado, tarefas e automação

A árvore Manus descreve agendamentos únicos/recorrentes, timezone, webhooks, retries, DLQ, pause/resume, histórico e Inbox. Esses termos não permitem concluir semântica de timezone, backoff, entrega confiável ou replay no Manus.

No Ollama, scheduler, webhooks, retries, DLQ e replay foram validados localmente, mas workers persistentes e smoke com serviços externos continuam pendentes. Redis workers/DLQ é adapter. Também falta UI completa para Agendado e Tarefas, com estados, erros, retry, pause/resume, replay, Inbox e timeline causal.

O gate deve ser uma execução agendada que sobreviva a restart, entregue webhook de forma idempotente, registre falha e backoff, entre em DLQ e seja reprocessada sem duplicação. A semântica de timezone deve ser testada explicitamente, inclusive em mudanças de horário.

### 8. Biblioteca, projetos, builders e artifacts

O Manus é descrito com Biblioteca por tipo, busca/filtros, preview, versionamento/diff, exportação e permissões; Projetos com contexto persistente, arquivos/fontes, memória, membros, tarefas e artifacts; e builders com canvas, drag-and-drop, undo/redo, preview, colaboração e deploy. O inventário não comprova branches, rollback ou diff visual quando não os explicita.

O Ollama já tem validação local de manifests, hashes, versões e download. Há backend local de canvas/undo/redo/export/publicação e Studio web, mas o Playwright usa API mockada. Deploy para Vercel, Netlify e genérico é adapter, aguardando contas e smoke autorizado. Memória/projetos são locais, e colaboração é adapter.

A prioridade é substituir o mock por uma jornada real: criar/salvar, editar, desfazer/refazer, gerar preview executável, exportar, persistir, verificar permissões e publicar sob approval com health check, proveniência e rollback. CRDT, expansão de formatos e colaboração podem seguir após o fluxo principal ser comprovado.

### 9. Segurança, privacidade e controle

A árvore Manus permite observar workspace/projeto, membros/papéis, approvals, scopes/consentimento e permissões de compartilhamento. Ela não fornece base para afirmar sandbox, DLP, secrets manager, retenção, auditoria resistente a adulteração, política de egress ou local-only.

No Ollama, há enforcement local de `CapabilityPolicy` deny-by-default, scopes conhecidos, grants e escrita com opt-in/approval; Browser Operator e SSRF guard têm proteções locais descritas. Auth/RBAC/SSO é adapter, a validação de IdP real e RLS em staging faltam, e a garantia geral de sandbox ainda não está comprovada. Também faltam DLP/secrets manager enterprise, auditoria completa, retenção/export/delete governados e prova E2E de que `Local-only` não faça fallback remoto implícito.

Este domínio é uma condição de segurança para todos os demais, não um item de polish. Os testes prioritários são cross-tenant/IDOR, path traversal, shell, egress, uploads, SSRF, protocolos não-TCP, replay de approvals, vazamento em prompts/logs/artifacts e revogação de sessão/segredo.

### 10. Company OS, mobile e operação

A árvore Manus documenta Empresa/Company OS, Projetos, membros/papéis, tarefas, agendamento, approvals, auditoria/feedback e controle de execução. Ela não explicita app mobile, traces, dashboards/SLOs, backup/restore ou CI/CD, portanto esses itens não são atribuídos ao Manus nesta comparação.

O Company OS e agentes departamentais do Ollama foram validados localmente, mas operação contínua distribuída, OAuth/scopes reais, connectors por plataforma, compliance, fluxos externos e staging ainda faltam. Mobile é parcial: há cliente Expo, cache/outbox offline e base de approvals, mas faltam push real, dispositivos, conflitos e distribuição. Observabilidade é adapter; Collector/Redis/Postgres em staging e falhas de recuperação não foram comprovados. Backup/restore e distribuição geral estão no alvo, não no estado validado.

Para operação com dados persistentes, backup/restore deve ser tratado como P1, com consistência, retenção e runbook. Depois entram releases assinados, SBOM, CI/CD, update/rollback e testes por SO; app stores e polish multi-OS podem ficar em P2.

## Forças compartilhadas e convergências

As duas árvores/visões convergem em várias superfícies importantes:

- **Experiência agentic orientada a missão:** plano, subtarefas, execução, timeline, approvals e artifacts aparecem como conceitos centrais.
- **Composer e chat como porta de entrada:** texto e contexto de arquivos/imagens são fundamentais; o Ollama acrescenta explicitamente áudio/vídeo, voz, fontes e seleção de provider/modelo como alvo.
- **Projects/workspaces e contexto persistente:** ambos organizam tarefas, arquivos, fontes e memória em torno de um espaço de trabalho.
- **Artifacts e builders:** documentos, slides, sites/apps, dashboards, mídia e outros resultados são tratados como objetos produzidos e exportáveis.
- **Pesquisa, browser e ferramentas externas:** a árvore Manus expõe essas jornadas; o Ollama tem guards, adapters e uma visão explícita de proveniência/consentimento.
- **Aprovação humana e controle de efeitos:** approvals, permissões, scopes e ações externas aparecem como componentes relevantes, embora a prova de enforcement varie.
- **Automação e recuperação:** agendamento, retries, histórico e recuperação são objetivos comuns; no Ollama, boa parte da prova ainda é local.
- **Segurança e governança como preocupação de produto:** identidade, RBAC, escopos, isolamento e approvals aparecem em graus diferentes. Nenhum item deve ser considerado garantido apenas por aparecer na árvore.

Essas convergências mostram que a comparação é útil como mapa de jornadas, mas não autorizam a conclusão de que os produtos têm a mesma implementação, confiabilidade ou cobertura operacional.

## Gaps de maior impacto, priorizados

### P0 — Segurança e autorização antes de ampliar autonomia

1. Validar IdP real, descoberta/JWKS, issuer/audience/nonce, rotação, sessão, recuperação e revogação.
2. Executar testes negativos cross-tenant, IDOR, path traversal, shell, upload, egress e secrets.
3. Aplicar e comprovar grants mínimos, deny-by-default, approvals vinculadas ao payload/hash e proteção contra replay em tools, MCP e conectores.
4. Definir garantias de sandbox e contenção por sistema operacional, incluindo workers, paths, rede, redirects, DNS/IP pinning e protocolos não-TCP.
5. Implementar/validar secrets manager, redaction e DLP; demonstrar que segredos e dados sensíveis não aparecem em prompts, logs ou artifacts.
6. Provar auditoria por chamada, retenção, exportação/apagamento governados e ausência de egress remoto implícito em modo local-only.

**Por que é P0:** um conector funcional sem isolamento, um fallback remoto não controlado ou um approval não vinculado à ação pode transformar uma capacidade aparentemente pronta em risco sistêmico.

### P0 — Jornada vertical Nova tarefa → artifact

1. Entregar o composer web real e estados de entrada, seleção de modelo/provider, modo, fontes, anexos e voz.
2. Iniciar missão real com streaming/timeline, plano e eventos causais.
3. Testar cancelamento, erro de provider, retry, recuperação e aprovação.
4. Gerar artifact, exibir preview e verificar persistência/exportação.
5. Capturar E2E com dados reais, incluindo falhas e estados vazios, sem usar apenas HTTP 200 ou mocks.

**Por que é P0:** sem essa fatia, shell, adapters, planner e módulos não constituem experiência de produto demonstrada.

### P0 — Shell, conta, uso e acessibilidade

Fechar workspace/projeto, perfil, settings e ajuda navegáveis; tornar quotas, custo e limites observáveis com origem confiável; implementar pesquisa global, command palette, atalhos, sidebar/painéis e notificações; e testar teclado, foco, leitor de tela, reduced motion, responsividade e critérios WCAG 2.2 AA nos fluxos principais.

### P1 — Durabilidade, workers e recuperação

Provar execução distribuída persistente, checkpoint, restart, retomada, replay, idempotência, DLQ e ausência de efeitos duplicados. Integrar workers, Redis/Postgres e observabilidade em staging. Usar missões longas com approvals, falhas e artifacts como testes de aceitação.

### P1 — Providers, integrações e pesquisa externa

Executar smokes autorizados para Claude/Codex, OmniRoute, HarnessRouter, MCP remoto, OAuth e connected accounts. Medir capacidade, saúde, limites, custo e fallback. Fechar pesquisa externa com citação por afirmação, proveniência, robots/SSRF, exportação e limites.

### P1 — Repo-first e builder/artifact real

Integrar snapshot/worktree ao runtime, runner sandboxed, repair loop, review/rollback e branch/PR opt-in. Substituir APIs mockadas do Studio por backend real, validar preview/export e publicação com approval, health check e rollback.

### P1/P2 — Operação e distribuição

Em P1 para qualquer deployment persistente: backup/restore, runbook, Collector/Redis/Postgres em staging, SLOs, alertas e recuperação. Em P2: mobile push/distribuição em lojas, installers assinados, SBOM, CI/CD multi-OS e polish de catálogo.

## Alvo mais amplo versus paridade comprovada

É necessário manter duas colunas mentais separadas:

### O que o alvo do Ollama amplia

Os documentos do Ollama explicitam intenções que ultrapassam a superfície Manus observada ou que a árvore Manus não permite confirmar, incluindo:

- local-first e privacidade sem fallback remoto implícito;
- autorização por tenant, projeto e recurso, RLS, grants mínimos, DLP e secrets manager;
- workflows versionados, tipados, duráveis, idempotentes, com checkpoints, replay, compensação e takeover;
- coding repo-first com worktree, runner, repair loop, review, CI e PR opt-in;
- pesquisa com proveniência por afirmação, ingestão de PDF/DOCX/XLSX, OCR/vision e memória com ACL;
- registry de providers com capacidade, saúde, limites, privacidade, benchmark e fallback;
- Skills/plugins/MCP/A2A, lifecycle, assinatura, rollback, OAuth e isolamento por tenant;
- Mission Observatory, métricas, traces, SLOs, alertas, replay, backup/restore, SBOM e CI/CD;
- Company OS e mobile companion com operação offline e reconciliação.

Esses itens são **escopo-alvo**. A documentação não os converte automaticamente em funcionalidade entregue.

### O que já tem alguma prova no Ollama

Há evidência local, em graus diferentes, para chat e modelos locais; runtime/planner/ferramentas/eventos/recovery/artifacts/approvals; scheduler/webhooks/retries/DLQ/replay; pesquisa com guards; grants deny-by-default; browser e SSRF protections locais; manifestos/scopes/allowlists/MCP; memória/projetos; Company OS local; e partes de artifacts/builders. Também há adapters para diversos providers, OAuth/SSO, deploy, observabilidade e conectores.

A palavra-chave é **alguma**: validação local não cobre serviço externo, staging distribuído, hardware, credenciais, tenant real, UI, acessibilidade, falhas longas ou produção. O fato de uma função estar em um adapter mostra preparação de integração, não smoke bem-sucedido.

### O que não pode ser concluído

Não se pode concluir que o Ollama seja superior ao Manus porque seu alvo é mais detalhado, nem que o Manus seja superior porque sua árvore lista uma tela ou ação. Também não se pode declarar que todos os itens visíveis do Manus funcionam, nem que todos os itens alvo do Ollama estão implementados. A conclusão suportada pelos documentos é mais estreita: **o Ollama tem um núcleo técnico local relevante, mas sua paridade de produto e operação permanece parcial; o alvo é mais amplo que a prova atual**.

## Sequência recomendada de execução

### Fase 0 — evidência e contratos

1. Congelar a matriz de evidência por capacidade, distinguindo local, adapter, condicional, parcial e E2E.
2. Definir o pacote mínimo de evidências para cada gate: logs, captura E2E, estados de erro, dados de teste, ambiente, aprovação e resultado reproduzível.
3. Definir contratos de tenant, grants, approval, artifact, evento causal, provider, retry, DLQ e proveniência.
4. Não aceitar “menu presente”, “rota HTTP 200”, adapter ou teste mockado como prova de jornada.

### Fase 1 — segurança, identidade e shell mínimo

1. IdP/RLS e testes negativos em staging.
2. Deny-by-default, approvals hash-bound, secrets/redaction, sandbox/egress e local-only.
3. Shell acessível e responsivo: workspace, Nova tarefa, conta/settings, uso/quotas, busca/atalhos e estados.
4. Instrumentação de correlation IDs, erros, latência, provider/modelo e custo sem expor segredos.

### Fase 2 — fatia vertical principal

1. Nova tarefa com texto, arquivo/imagem e fonte autorizada.
2. Seleção de modelo/provider e modo, com limites e fallback explícitos.
3. Missão com plano, timeline, approval, cancelamento, retry e recovery.
4. Artifact persistido, preview/exportável e biblioteca mínima.
5. E2E com dados reais e falhas controladas.

### Fase 3 — durabilidade e automação

1. Workers persistentes, filas e staging distribuído.
2. Checkpoint, restart, replay, DLQ, backoff e idempotência.
3. Agendamentos, timezone e webhooks externos sob approval.
4. Observabilidade, alertas, SLOs e runbook de recuperação.

### Fase 4 — integrações e pesquisa verificável

1. OAuth real, connected accounts, scopes mínimos, refresh/revogação e MCP externo.
2. Browser/pesquisa com fontes externas, citações por afirmação, proveniência e exportação.
3. Ingestão por formato e memória com benchmark, ACL e storage distribuído.
4. Providers Claude/Codex/OmniRoute/HarnessRouter e fallback testados em ambiente autorizado.

### Fase 5 — repo-first, builders e artifacts

1. Snapshot/worktree integrado, runner sandboxed e repair loop.
2. Review, rollback, branch/PR opt-in e CI vinculado.
3. Studio contra backend real, preview executável, exportação e publicação com health/rollback.
4. Biblioteca com busca, filtros, diff, permissões, arquivos grandes e colaboração posterior.

### Fase 6 — escala de produto e operação

1. Company OS integrado a connectors reais com budget, compliance e pausa segura.
2. Mobile com push, offline idempotente, reconciliação e testes físicos.
3. Backup/restore, releases assinados, SBOM, CI/CD e distribuição multi-OS.
4. CRDT/realtime, A2A, amplitude de conectores e polish após os gates de segurança e durabilidade.

## Resultado agregado

- **Domínios comparados:** 10.
- **Domínios falhos no mapa fornecido:** 0.
- **Paridade plena demonstrada:** nenhum domínio foi sustentado como plenamente comprovado pelos resultados fornecidos.
- **Padrão predominante:** paridade parcial, com validações locais úteis, adapters e várias jornadas E2E/externas ainda pendentes.
- **Conclusão:** o Ollama Full possui uma base técnica e um escopo-alvo ambiciosos, mas deve fechar segurança, shell e jornada vertical antes de usar a amplitude do alvo como argumento de paridade. A árvore Manus é uma referência de superfície observável, não uma certificação de sua arquitetura ou operação interna.
