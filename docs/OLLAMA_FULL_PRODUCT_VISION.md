# Ollama Full — visão de produto e roadmap

> **Ollama Full** é a evolução local-first do fork mantido por DZ23-LTDA: modelos Ollama, agentes, ferramentas, builders e fluxos de trabalho num ambiente controlável, extensível e verificável.

Ollama Full deve ser ambicioso sem alegar que já é “o melhor” ou que tem paridade com qualquer referência antes de demonstrá-la. A promessa do produto é **poder local com execução governada e resultados verificáveis**. A referência observável de cada entrega é seu código, teste e evidência de execução, não uma lista de menus.

Este é o **documento canônico de produto e prioridade**. `docs/agentic/ROADMAP.md`, `PRODUCT_TREE.md` e `PARITY_MATRIX.md` continuam como registros técnicos de execução e evidência; títulos antigos com “Ollama Full” ou “DZ23” identificam linhagem/documentação histórica, não um segundo nome de produto. Em conflito de prioridade ou claim, esta visão e a evidência de código/testes prevalecem.

## Posicionamento e limites de marca

- **Nome de produto:** Ollama Full.
- **Atribuição:** mantido por DZ23-LTDA e baseado no projeto Ollama; não é o serviço hospedado nem uma distribuição oficial/endossada pelo Ollama upstream.
- **Prioridade:** local-first por padrão; rede, providers remotos, integrações, execução externa e publicação são capabilities explícitas, configuráveis e auditáveis.
- **Compatibilidade:** até existir uma migração versionada, manter CLI, API, env vars, protocolos, diretórios de dados e formatos de modelo existentes. A marca visível não deve causar migração destrutiva ou instalações paralelas silenciosas.
- **Proveniência:** estudar comportamento e princípios dos produtos comparados; não copiar código, interface, marca ou material proprietário. Revisar licença e dependências antes de qualquer reutilização de código.
- **Estados honestos:** cada feature é rotulada como `planejada`, `implementada`, `testada localmente`, `validada E2E` ou `dependente de serviço/dispositivo externo`.

## Experiência que queremos oferecer

Uma pessoa deve conseguir escolher um modelo local, conversar, pesquisar arquivos próprios, delegar uma missão, revisar o plano, autorizar capacidades específicas, observar o progresso, interromper ou retomar, inspecionar mudanças e exportar um resultado. Para código, o caminho deve partir de um repositório e terminar num diff testado e revisável. Para builders, deve terminar em preview executável, artifact versionado e publicação separada e autorizada.

A mesma missão e seus eventos precisam ser entendidos na interface Web, CLI e clientes desktop/mobile. Cada superfície mostra o que o agente sabe, o que executou, o que falta, quais dados saíram da máquina e qual ação precisa de aprovação. Respostas factuais devem preservar fontes; geração de código deve preservar diff/testes; ações externas devem preservar consentimento, aprovação e trilha de auditoria.

## Árvore-alvo de capacidades

```text
Ollama Full
├── 1. Modelos e inferência local
│   ├── descobrir hardware, backends e capacidades reais
│   ├── baixar/gerir modelos com licença e origem visíveis
│   ├── chat, streaming, vision, audio, embeddings e tool calls por capacidade
│   ├── profiles de desempenho e privacidade; fallback sem egress implícito
│   └── registro de provider/modelo com versão, limites, saúde e benchmark
├── 2. Assistente e pesquisa
│   ├── chat multimodal com contexto de arquivos autorizado
│   ├── pesquisa com fontes, datas e proveniência por afirmação
│   ├── memória factual separada de skills procedurais, com escopo/ACL
│   └── contexto local indexado, busca exata + semântica e limites de contexto
├── 3. Missões e workflows
│   ├── plano editável, orçamento, critérios de aceite e policy preview
│   ├── grafo versionado/tipado: entradas, saídas, condições, approval e retry
│   ├── execução durável, idempotente, cancelável, checkpoint e replay
│   ├── tools/plugins/MCP por capability e escopo mínimo
│   └── eventos causais, artifacts e handoff/takeover humano
├── 4. Engenharia de software
│   ├── explorar repo antes de editar; map/tree-sitter e proveniência de contexto
│   ├── branch/worktree isolado e sessão reentrante
│   ├── patch incremental, diff por arquivo/hunk, aceitar/rejeitar e rollback
│   ├── terminal/testes/lint/build em sandbox declarado por plataforma
│   ├── evidence bundle e checklist de segurança
│   └── commit/PR somente após approval, com CI e resultado vinculados
├── 5. Builders e artefatos
│   ├── web/app/game/slides/dashboard/document e formatos suportados
│   ├── design tokens/component registry + árvore visual tipada
│   ├── preview executável, browser smoke, accessibility e diff visual
│   ├── versões imutáveis, comentários/revisão e exportação portátil
│   └── publish autorizado com health check, rollback e proveniência
├── 6. Agentes e equipes
│   ├── papéis/SOPs versionados com I/O e limites declarados
│   ├── agentes specialists dentro de workflows duráveis
│   ├── delegação, fan-out/fan-in e takeover na mesma timeline
│   └── Company OS opt-in: ciclos, KPIs, budgets, approval e pausa segura
├── 7. Conectividade e dispositivos
│   ├── connectors/APIs/MCP opt-in, escopados e revogáveis
│   ├── navegador/computer-use com sessão explícita e aprovação contextual
│   ├── desktop companion e cliente mobile, estados offline/reconciliação
│   └── adapters de serviços externos sem dependência obrigatória
├── 8. Segurança, privacidade e controle
│   ├── auth, tenant/resource authorization e policy deny-by-default
│   ├── sandbox/egress/segredos com garantia explícita por SO
│   ├── prompt injection, DLP, retenção, redaction e export/delete governado
│   └── supply-chain, threat model e auditorias reproduzíveis
└── 9. Operação, qualidade e comunidade
    ├── Mission Observatory: timeline, status, quotas, custos, stale/unknown
    ├── evaluation/benchmarks para modelos, harnesses, builders e workflows
    ├── logs/metrics/traces, health, backup/restore e runbooks
    ├── docs, CLI, API, mobile/desktop e distribuição versionada
    └── upstream sync manual, atribuição e guardrails de compatibilidade
```

## Roadmap por gates

### Fase 0 — Fundação, naming e evidência (esta branch)

- Registrar a visão e diferenciar baseline de promessa futura.
- Atualizar o nome de produto nas superfícies web selecionadas e usar a logo fornecida para temas claro/escuro.
- Preservar upstream attribution, slug remoto, CLI, API, protocolo e diretórios de dados.
- Criar uma suíte de evidência que classifica implementação, testes locais e E2E.

**Saída:** brand slice aprovado por build/testes; documentação sem falsas alegações; nenhuma quebra de compatibilidade.

### Fase 1 — Segurança antes de autonomia

- Mapear capability → tool/endpoint/adapter e exigir autorização no servidor por tenant, projeto, sessão e recurso.
- Negação global prioritária, allowlists, approval vinculada ao payload e proteção contra replay.
- Distinguir sandbox estrito de best-effort por OS; negar execução não confiável quando o isolamento requerido não estiver disponível.
- Fazer threat model para filesystem, shell, browser, MCP, plugins, segredos, SSRF, uploads e prompt injection.

**Saída:** testes negativos reproduzíveis (cross-tenant, IDOR, path traversal, shell, egress, segredo) e matriz clara de garantias por plataforma.

### Fase 2 — Engenharia repo-first de ponta a ponta

- Descoberta limitada/auditável do repo; contexto citado por arquivo/símbolo.
- Spec → plano por arquivo → branch/worktree isolado → patch/diff → teste/lint/build → review → approval → commit/PR opt-in.
- Cancelamento real, retomada e correção de falhas sem duplicar efeitos.
- Evidence bundle imutável com SHA, modelo/provider, tools, comandos, logs redigidos, testes e artifacts.

**Saída:** benchmark sintético E2E com casos de sucesso, falha de teste, cancelamento, reinício, prompt injection e credencial ausente; nenhum efeito Git externo sem approval.

### Fase 3 — Workflow Studio e Mission Observatory

- Um schema versionado serve planner, executor, UI, API, CLI e MCP.
- Nós tipados, condições e limites, idempotência, budgets, retry, approvals e replay.
- Grafo UI read-only primeiro; ações de edição tornam-se comandos validados pelo servidor.
- Timeline causal com estado, eventos, errors, costs, source/timestamp e unknown/stale explícitos.

**Saída:** missão controlada com trigger, steps, approval, falha/retry, restart e replay; persistência e policy verificadas.

### Fase 4 — Builder que entrega e prova

- Índice local de componentes/tokens/estilo e política de dependências.
- Modelo visual tipado conectado ao código real; alterações reversíveis.
- Preview real, smoke browser, acessibilidade e teste de API/backend antes de marcar Ready.
- Exportação e publicação independentes, com aprovação, health check, versionamento e rollback.

**Saída:** app de referência gerado e editado por código e visual, sem tokens inventados não explicados; testes e export reproduzíveis.

### Fase 5 — Model registry, conectores e distribuição

- Capability discovery com versão, modalidades, ferramentas, limites, privacidade, saúde, custo e quota.
- Conformance suite para Ollama, providers e harnesses; medir localmente antes de recomendar route/fallback.
- Adapters de terceiros opt-in, isolados, revogáveis, sem routing duplo e sem egress implícito.
- Melhorar installers/clientes somente com compatibilidade, assinatura, update/rollback e testes por OS.

**Saída:** matriz pública de suporte baseada em testes e decisão de routing auditável; nenhum claim de compatibilidade só por formato de API.

### Fase 6 — Colaboração e capacidades especializadas

- Colaboração realtime/CRDT depois do contrato de documento, autorização e conflito.
- Rotinas/skills promovidas após execução observada, lint, teste e approval.
- Voz/telefonia, fine-tuning e multi-cloud como opções especializadas com caso de uso, consentimento, custos e benchmark definidos.

**Saída:** cada extensão isolada e testável, sem comprometer a instalação básica local-first.

## Estado de execução da branch — 2026-09-26

A fundação visual Ollama Full está aplicada nas superfícies web selecionadas; o ativo Swole original não foi recuperado nesta sessão, então a troca de logo não está comprovada. A branch contém Git read-only opt-in, filtro de egress do Browser, ownership cross-tenant, artifacts verificados e journal restaurável, UI Studio para Builders, mídia agentic com approval hash-bound e escrita/patch de arquivo com diff, grants e revalidação. CreateMission agora pode criar snapshot físico opt-in de projeto Git, separado por tenant/mission, sem modificar a origem; a UI expõe essa opção e desabilita-a sem projeto. Também há DLP nos limites de connector/MCP, version-CAS no estado de missão/JSONStore, fencing da fila e Redis TLS com política loopback para plaintext. Writers de workspace ainda não têm CAS atômico contra processos externos. Os gates da alteração mais recente ainda estão em execução.

**Status honesto:** estamos em `Fases 0–1 + slices iniciais das fases 2 e 4`, não em paridade completa. Snapshot de projeto Git é opt-in e integrado à criação da missão, mas não equivale a worktree/branch: não há merge/apply de volta, cleanup/retention operacional, runner de testes do projeto nem repair loop. `workspace.patch` aplica conjuntos revisáveis de até 16 arquivos, com approval hash-bound e compensação best-effort; writers locais são serializados e bases revalidadas antes do rename, mas um processo externo ainda disputa a janela final e não é crash-atomic. Queue fencing e version-CAS não provam execução exatamente uma vez. O scanner DLP cobre boundaries específicas de Connector/MCP e é heurístico, não uma garantia global de zero egress. Próximo trabalho prioritário: lifecycle/retention de snapshots, runner de testes seguro dentro do sandbox, repair loop e E2E repo-first; em seguida, jornada Builder contra backend real. Adapters multimodais e externos continuam dependentes de provider, credenciais e ambiente habilitados.

## Definição global de “melhor”

Ollama Full só deve declarar vantagem quando, em uma tarefa e ambiente documentados, demonstrar um ou mais destes resultados contra alternativas comparáveis: maior taxa de conclusão correta; menor risco de alteração indevida ou vazamento; mais transparência e controle humano; menor latência/custo para a mesma qualidade; melhor recuperação após falha; portabilidade e qualidade de artefatos. Toda comparação registra versões, hardware, modelo, tarefa, limites, repetições e erro. “Tem tudo” não será medido por contagem de integrações.

## Métricas de release

- **Coding:** issue resolvida, tests pré-existentes preservados, regressões, arquivos tocados, intervenção, tempo, tokens e recuperação.
- **Workflow:** tarefas concluídas sem duplicação após restart/retry; replay determinístico; eventos perdidos/reordenados; aprovação respeitada.
- **Builder:** critérios de aceite aprovados, testes/browser/a11y, fidelidade ao design system, diff de dependências, export funcional e rollback.
- **Segurança:** deny coverage, cross-tenant tests, isolamento por OS, egress observado, segredos ausentes de logs/artifacts e taxa de injection bloqueada.
- **Usabilidade:** conclusão de tarefas com teclado, clareza de loading/empty/error/approval, acessibilidade WCAG 2.2 AA nos fluxos principais.
- **Modelo/provider:** capability real, TTFT/throughput, qualidade por benchmark, custo/quota, fallback e path local/cloud.

## Limites do primeiro incremento

Esta fatia não afirma que todas as capacidades acima já existem. Ela fixa o norte e cria o início da identidade **Ollama Full**. Features devem avançar por implementação vertical e permanecer rotuladas pelo estado de evidência. O rename do slug GitHub, domínio, publisher, binário e paths é uma decisão separada de migração e não está implícito na mudança do nome visível dentro da UI.
