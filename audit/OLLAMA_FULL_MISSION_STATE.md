# Estado da missão — Ollama Full

> **Leitura do checkpoint:** os blocos YAML abaixo são snapshots históricos append-only de cada fase, não um estado vivo consolidado. Campos `state`, `head`, `uncommitted_changes` e `files_changed` dos blocos antigos não descrevem o worktree atual; consulte sempre o último bloco `Estado atual autoritativo` no fim deste arquivo e confirme `git status` antes de atuar.

```yaml
mission_id: 20260926-ollama-full-foundation
objective: Evoluir o fork local-first Ollama Classe A+ para o produto Ollama Full, aproveitando princípios comprovados de Manus Desktop, coding harnesses e repositórios comparados, sem copiar código/interface proprietária nem quebrar compatibilidade Ollama.
scope:
  in:
    - Nome de produto Ollama Full na identidade visível do app e documentação nova.
    - Uso da logo fornecida em interfaces adequadas a claro/escuro, preservando o arquivo original.
    - Roadmap técnico por fases, priorizando segurança e a jornada repo-first.
    - Iterações de implementação pequenas, reversíveis e verificadas no branch feat/ollama-full-foundation.
    - Manter APIs, CLI, nomes de binários, protocolo, paths de dados e aviso de fork até migração compatível aprovada.
  out:
    - Renomear o repositório GitHub, organização, domínio ou publicar releases sem confirmação específica.
    - Prometer paridade total ou declarar recursos sem implementação e evidência E2E.
    - Copiar código, assets, UI ou internals proprietários de Manus ou outros produtos.
    - Fazer push/merge/publicação ou ações que alterem o repositório remoto.
acceptance_criteria:
  - A nova visão e os critérios de aceitação estão versionados no repo.
  - A primeira fatia de marca exibe Ollama Full em superfícies web acordadas e utiliza a logo fornecida sem danificar original/transparência.
  - Compatibilidade de binários/API/armazenamento e atribuição ao Ollama upstream permanecem explícitas.
  - Testes aplicáveis e verificações de diff passam; alterações são revisadas.
  - Roadmap distingue baseline observado, feature proposta, implementação e validação.
gates:
  required: [git_state, docs_consistency, ui_lint_or_typecheck, ui_tests, ui_build, diff_check, logo_asset_integrity, final_product_security_review]
  not_applicable:
    - item: full_native_build
      reason: A fatia inicial é documentação e interface web; não altera o daemon Go/C++.
    - item: external_provider_smoke
      reason: Não haverá conexão nem publicação externa nesta fatia.
delivery_destination: branch local feat/ollama-full-foundation; remote/push não autorizado nesta etapa.
approvals_required:
  - Renomeação pública do repositório/slug GitHub, domínio, organização ou publicação.
  - Alterar identidade de protocolo/binário/path/instalação que possa afetar usuários existentes.
budget:
  max_equivalent_attempts: 3
  max_attempts_without_progress: 5
  max_parallel_agents: 3
  task_timeout: 900s
  no_progress_timeout: 15m
  api_or_cost_limit: Apenas ferramentas e testes locais já disponíveis; sem compras, cloud provisioning ou providers pagos.
rollback_plan: Reverter somente commits próprios do branch local; não descartar alterações não commitadas do usuário. Asset derivado pode ser removido mantendo attachment original fora do repo; cada tela pode reverter isoladamente.

state: EXECUTING
iteration: 1
started_at: 2026-09-26T09:43:00-03:00
heartbeat_at: 2026-09-26T09:43:00-03:00
last_progress_at: 2026-09-26T09:43:00-03:00
repository:
  path: /home/ubuntu/ollama-classe-a-plus
  branch: feat/ollama-full-foundation
  upstream: origin/main
  remotes: [origin]
  head: 8635e30dc9e95a1f5b29700169783abc24093ceb
  uncommitted_changes: false
current_task: Especificar identidade e roadmap; implementar primeira fatia visual local.
current_failure: ""
current_strategy: Consolidar auditoria existente e fazer uma fatia vertical mínima de marca/documentação antes de ampliar funcionalidades.
plan:
  - Recuperar instruções/checkpoints e estado git atual.
  - Registrar visão do produto e roadmap com baseline e fases verificáveis.
  - Integrar logo em assets claro/escuro e atualizar marca visual sem renomear APIs/paths.
  - Executar testes, build, revisar diff e fazer auditorias de produto/segurança.
  - Solicitar confirmação apenas para rename/push remoto e continuar fatias locais seguras.
completed_tasks: [recovery_git_clean_main_sha_8635e30]
pending_tasks: [vision_roadmap, branding_slice, quality_gates, final_audits]
dependencies: []
blockers: []
approvals_pending: []
hypotheses:
  - "A logo PNG é artwork preto com transparência; gerar variante branca para fundos escuros preservando alpha e pixels não visíveis."
  - "Ollama Full é nome de produto, não rename imediato do slug do repositório nem substituição da marca upstream Ollama."
decisions:
  - "Criada branch local feat/ollama-full-foundation a partir de main; nenhum remote alterado."
  - "Identidade: Ollama Full by DZ23-LTDA, claramente descrito como fork/distribuição baseada no Ollama."
  - "Manter nomes upstream de binários/API/protocolo/pastas na fase 1 para evitar breaking changes; usar brand name nas superfícies visíveis."
  - "Usar a logo fornecida e variante invertida para dark theme; sem inventar paleta além de monocromática enquanto não houver guideline adicional."
strategies_tried: []
discarded_hypotheses: []
attempt_count: 0
same_failure_count: 0
tests_passed_delta: 0
tests_failed_delta: 0
completed_tasks_delta: 0
files_changed: []
commands_and_tests:
  - "git status --short --branch: clean em main antes da branch"
  - "git switch -c feat/ollama-full-foundation: sucesso"
  - "Logo inspect Pillow: PNG RGBA 1388x1831, arte preta/transparente; anexo não modificado"
evidence:
  - claim: A branch de trabalho é isolada de main e a cópia inicial estava limpa.
    command_or_observation: git status --short --branch; git switch -c feat/ollama-full-foundation
    result: clean origin/main no SHA 8635e30; branch local criada.
    timestamp: 2026-09-26T09:43:29-03:00
    artifact_or_log: git worktree em /home/ubuntu/ollama-classe-a-plus
artifacts: []
commits: []
delegated_agents: []
audits: []
risks:
  - "O nome contém Ollama e deve continuar identificando o projeto como fork independente, sem sugerir endosso oficial."
  - "Renomear o slug remoto pode quebrar consumidores/URLs; está fora do escopo local até confirmação específica."
  - "O projeto é grande; uma única sessão não pode honestamente entregar paridade completa. Trabalhar por fatias verificáveis e manter checkpoint."
context_summary: "Repositório auditado: Go/C++/React/TypeScript/Expo; missão nova em branch local sobre SHA 8635e30. Baseline inclui runtime agentic extenso, porém auditoria estática mostrou gaps de E2E e ausência aparente de fluxo Git nativo repo→PR. O usuário quer renomear produto para Ollama Full e trouxe logo PNG preta transparente. Próximo passo é criar docs/OLLAMA_FULL_PRODUCT_VISION.md e assets, atualizar branding web com compatibilidade preservada; não push/rename remoto."
next_action: Criar documento de visão/roadmap e assets de logo; depois atualizar cabeçalho/sidebar.
resume_instructions: Ler este checkpoint e comparar com git status/HEAD antes de continuar. Não repetir a pesquisa comparativa. Não dar push nem renomear slug remoto. Continuar primeiro slice de marca, testar, corrigir, checkpoint.
```


## Checkpoint de progresso — primeira fatia de marca (2026-09-26)

```yaml
state: TESTING
iteration: 2
last_progress_at: 2026-09-26T09:46:30-03:00
repository:
  branch: feat/ollama-full-foundation
  head: 8635e30dc9e95a1f5b29700169783abc24093ceb
  uncommitted_changes: true
completed_tasks:
  - recuperação do snapshot e branch isolada
  - docs/OLLAMA_FULL_PRODUCT_VISION.md criado com árvores de capacidades, gates e roadmap
  - logo original copiada intacta e variante branca gerada preservando alpha; favicons 256x256 derivados
  - branding aplicado no sidebar, home, title, favicon, README; smoke E2E atualizado
  - npm ci executado a partir do package-lock.json, sem editar dependências declaradas
  - guardrail scripts/check-class-a-plus-integrity.sh: PASS
  - frontend unit: 22 arquivos, 206 testes PASS
  - frontend eslint: PASS
  - frontend build TypeScript/Vite: PASS; warning existente de chunk JS ~1.50 MB permanece
  - asset logo: dimensões/transparência/alpha preservation PASS
pending_tasks: [playwright_e2e, visual_review, three_independent_audits, complete_implementation_slices]
commands_and_tests:
  - "(cd app/ui/app && npm ci --no-audit --no-fund): PASS, 722 packages pelo lockfile"
  - "(cd app/ui/app && npm test -- --run): PASS, 22 files / 206 tests"
  - "(cd app/ui/app && npm run lint): PASS"
  - "(cd app/ui/app && npm run build): PASS; chunk warning registrado"
  - "scripts/check-class-a-plus-integrity.sh: PASS"
  - "Playwright test:e2e: em execução"
evidence:
  - claim: variante dark mantém exatamente os canais alpha da imagem fornecida.
    command_or_observation: Pillow comparou canal alpha de light/dark.
    result: PASS; PNG RGBA 1388x1831 original preservado.
    timestamp: 2026-09-26T09:45:30-03:00
    artifact_or_log: app/ui/app/public/ollama-full-logo*.png
  - claim: suite e build UI passam depois da mudança local.
    command_or_observation: npm test -- --run; npm run lint; npm run build; scripts/check-class-a-plus-integrity.sh.
    result: 206 tests PASS, lint PASS, build PASS, integrity PASS.
    timestamp: 2026-09-26T09:46:27-03:00
    artifact_or_log: terminal sessions ollama-full-tests, ollama-full-lint, ollama-full-build, ollama-full-integrity
files_changed:
  - docs/OLLAMA_FULL_PRODUCT_VISION.md
  - README.md
  - app/ui/app/index.html
  - app/ui/app/src/components/AppSidebar.tsx
  - app/ui/app/src/components/Chat.tsx
  - app/ui/app/e2e/shell.spec.ts
  - app/ui/app/public/ollama-full-logo.png
  - app/ui/app/public/ollama-full-logo-dark.png
  - app/ui/app/public/ollama-full-icon.png
  - app/ui/app/public/ollama-full-icon-dark.png
  - audit/OLLAMA_FULL_MISSION_STATE.md
risks:
  - external rename/push/release remain unauthorized and not performed
  - logo is monochrome black artwork with transparency; dark variant is a direct white recolor; verify actual placement via browser before accepting
  - no Go toolchain present in shell; this UI/doc slice does not change Go backend
next_action: Await Playwright result; inspect browser appearance; run three independent read-only audits; fix findings; proceed to next safe vertical feature rather than claim full product done.
```


## Checkpoint de progresso — E2E e inspeção visual (2026-09-26)

```yaml
state: INSPECTING
iteration: 3
last_progress_at: 2026-09-26T09:49:30-03:00
completed_tasks:
  - Playwright Chromium E2E: 1/1 PASS para home, nome e presença da logo
  - Inspeção visual de screenshot light/dark: logo legível e sem distorção em ambos
pending_tasks: [three_independent_audits, next_vertical_feature]
commands_and_tests:
  - "npm run test:e2e -- --project=chromium: PASS, 1 test"
  - "Chromium Vite preview: title Ollama Full, body brand visible, logo assets naturalWidth 1388, variants corretas em light/dark"
  - "Console mostrou GET /api/user 404 pois backend Go não estava iniciado; UI mostrou estado de carregamento do modelo. Isso não foi tratado como erro da marca nem como funcionalidade completa."
evidence:
  - claim: UI carrega e exibe a marca em ambos os temas.
    command_or_observation: Playwright + Chromium preview, scheme light/dark.
    result: Ollama Full visível; light usa PNG original e dark usa PNG branco; assets carregam; screenshot capturado.
    timestamp: 2026-09-26T09:49:25-03:00
    artifact_or_log: /tmp/ollama-full-home-light.png; /tmp/ollama-full-home-dark.png
  - claim: O E2E inicial falhou somente porque o Chromium headless não estava instalado; após instalação fixada a suíte passou.
    command_or_observation: npx playwright install chromium; npm run test:e2e -- --project=chromium
    result: PASS em 919 ms de teste após browser setup.
    timestamp: 2026-09-26T09:48:29-03:00
    artifact_or_log: terminal session ollama-full-e2e
risks:
  - backend não iniciou no sandbox: shell reportou Go CLI indisponível; smoke visual requer backend para validar os demais estados operacionais
  - continuar não significa concluir produto; implementação de coding repo-first, mission graph e demais áreas ainda está pendente
next_action: Iniciar três auditorias read-only (engenharia/runtime, segurança/capability policy, produto/UX da marca e roadmap); usar achados para definir próxima fatia.
```


## Estado atual autoritativo — 2026-09-26

```yaml
state: IMPLEMENTED_AND_VALIDATED_LOCAL_BRANCH
branch: feat/ollama-full-foundation
head: 8635e30dc9e95a1f5b29700169783abc24093ceb
worktree: dirty_intentional_changes_only
remote_push_or_release: false
product_scope_delivered:
  - identidade web Ollama Full; assets claro/escuro e favicons; screenshots antigas rotuladas como baseline histórico
  - correção de logo duplicada no composer/onboarding e foco inicial para navegação por teclado
  - saúde local com backoff limitado, estado offline explicado e retry/recovery
  - skip link, atribuição About e controles de repo:read com consentimento explícito
  - Git inspector read-only limitado, sem comandos fornecidos pelo modelo, sem rede lazy fetch e com teto real de memória/output
  - bloqueio de egress privado do browser via proxy SOCKS5, resolução uma vez e pinning de IP público
  - recuperação de journal JSON após restart e verificação de conteúdo de artifacts no download
  - documentação Ollama Full, matriz/árvore alinhadas e manifesto de proveniência da logo
checks:
  go_test_all: PASS
  go_vet_all: PASS
  go_build_all: PASS
  ui_unit_tests: PASS_207
  ui_lint: PASS
  ui_build: PASS
  chromium_e2e: PASS_3
  class_a_plus_integrity_guard: PASS
  git_diff_check: PASS
  png_integrity_metadata_sha256: PASS
limitations:
  - sem commit, push, merge, rename do repositório, binário, protocolo, diretório de dados, domínio ou release
  - smoke real com runtime/modelo local e testes de instalação/API por sistema operacional continuam pendentes
  - sandbox estrito por plataforma, approval vinculado a payload e recuperação distribuída de execução permanecem no roadmap
  - adapters externos não foram autenticados nem conectados nesta missão
  - build Vite emite aviso preexistente de chunks acima de 500 kB; bundle splitting permanece melhoria futura
next_gate: review humana do patch/identidade antes de qualquer publicação ou ampliação a escrita/worktrees/commit
```

### Arquivos e evidência

- Mudanças locais de implementação, testes, documentação e quatro PNGs em `feat/ollama-full-foundation`; a lista exata está no `git status --short --untracked-files=all` e no patch binário exportado para esta entrega.
- O nome de produto foi aplicado à UI web e à documentação nova. Binário, APIs, protocolo, armazenamento, slug GitHub e instalador continuam sem rename deliberadamente; os testes não provam compatibilidade multi-OS nem um runtime local com modelo carregado.
- A logo anexada foi autorizada para uso no projeto pelo pedido do usuário. Licença/titularidade para redistribuição pública não foram confirmadas; verificar antes de release.
- Pesquisa comparativa detalhada preservada em `ollama-classe-a-plus-comparativo-estrategico-2026-09-26.md`; direcionamento priorizado no `docs/OLLAMA_FULL_PRODUCT_VISION.md`.


## Checkpoint de paridade — incremento Studio + multimídia (2026-09-26)

**Estado:** `IN_PROGRESS`; esta atualização substitui apenas o status de trabalho mais recente, não os históricos acima. **Não** declara paridade total com Manus nem conclusão do Ollama Full.

### Implementado nesta etapa

- Studio visual web em `/builders`, ligado aos endpoints Builder existentes: listar/criar projeto, editar árvore JSON, undo/redo, preparar preview isolado, publicação local e export ZIP.
- Preview servido como stream sob containment de `os.Root`, com MIME explícito, `nosniff`, `no-store` e CSP; export/publish/preview validados por organização.
- Tool `media.process` integrada ao runtime quando há provider configurado, cobrindo imagem, vídeo, fala, transcrição, visão, OCR e tom. Exige `media:execute` opcional e approval por step; rotas HTTP diretas legadas foram removidas para evitar bypass.
- Agentic Console consulta tools disponíveis, desabilita mídia sem provider, apresenta payload da tool na tela de approval, concede o scope somente com checkbox explícito e o reseta após criação.
- Regressão cross-tenant em connectors: organização não pode usar connector registrado por outra organização; a tentativa não emite request externa.
- E2E comprovou criação de projeto e carregamento de preview em iframe `sandbox="allow-scripts"`; E2E anterior cobre shell offline, retry do health e logo única nos temas claro/escuro.

### Evidência desta revisão

- `go test ./... -count=1 -timeout=15m`: PASS.
- `go vet ./...`: PASS.
- `go build ./...`: PASS.
- Frontend: 23 arquivos Vitest passaram na suíte; teste direcionado mais recente do Agentic Console: 4/4 PASS.
- `npm run lint`: PASS.
- `npm run build`: PASS; permanece aviso conhecido de chunks grandes (>500 kB).
- Playwright Chromium: 4/4 PASS, incluindo Builder create + preview sandbox.
- Regressão de runtime multimídia: provider recebe zero requests antes de approval e exatamente uma após approval; artifact é anexado ao mission/step: PASS.
- `scripts/check-class-a-plus-integrity.sh` e `git diff --check`: PASS.

### Limites ainda abertos

- Paridade completa com Manus **não alcançada**. O E2E Builder ainda não cobre salvar/editar, undo/redo, publicação local ou export ZIP; a operação é um studio estruturado JSON, não um editor visual drag-and-drop nem um gerador frontend aberto.
- Execução repo-aware ainda está no nível de inspeção read-only: faltam patch editável revisável, gerenciamento isolado de worktrees, execução de testes iniciada pelo agente, repair loop, restore/recovery e fluxo Git seguro de commit/PR.
- Ainda faltam testes reais com modelo Ollama instalado/carregado, providers multimídia de produção, organização/SSO de ponta a ponta, integrações autenticadas e smoke/install por sistema operacional.
- Tarefas, biblioteca, skills, settings, projetos e conectores têm áreas implementadas, parciais ou adapters; a árvore e matriz de paridade, não a promessa de nome, definem cada estado.
- Branch local `feat/ollama-full-foundation`; sem commit, push, merge ou release desta continuação.

### Próximo gate

Completar jornada coding repo-aware segura e o ciclo Builder (salvar → desfazer/refazer → publicar local → exportar) com E2E, depois reavaliar lacunas por domínio e repetir auditorias independentes antes de qualquer declaração de release.


### Delta de engenharia repo-aware — 2026-09-26

`workspace.read` agora retorna SHA-256; `workspace.write` continua atrás de capability e approval, cria arquivos novos ou sobrescreve apenas após receber o hash observado, valida compare-and-swap, grava atomicamente por `os.Root` e preserva o conteúdo anterior como backup artifact rastreado por mission/step. O planner recebe instruções explícitas para não sobrescrever sem leitura/hash. Testes direcionados de escrita, stale hash, backup, symlink e approval passaram. Isso **não** fecha a fase repo-aware: `git.repo.inspect` segue read-only e ainda faltam patch reviewável multi-arquivo, worktree, runner de testes acionado pelo agente e repair/recovery E2E.


## Checkpoint — approval de escrita revisável e Builder E2E ampliado (2026-09-26)

**Estado:** `IN_PROGRESS`; não significa paridade Manus concluída nem release.

### Novo comportamento implementado

- `workspace.write` só prepara a operação quando `workspace:write` foi explicitamente concedida. O prompt do planner recebe os grants e instrui a não propor tools para capabilities ausentes.
- Antes da approval, o plano passa a incluir unified diff do arquivo existente/novo e SHA-256 autoritativo da versão-base; arquivo binário, symlink, caminho fora do workspace e pasta reservada de backup são recusados.
- O SHA-256 canônico da operação (tool/risk/input, incluindo conteúdo e diff de aprovação) é guardado na Approval e verificado ao aprovar e novamente antes do dispatch. A rejeição continua permitida mesmo se o payload ficou obsoleto.
- A UI inclui o diff dentro do payload exato da etapa. A gravação permanece atômica e com compare-and-swap; overwrite cria backup artifact. Diff limitado a 32 KiB e escrita por arquivo limitada a 1 MiB.
- Builder Playwright com APIs mockadas cobre agora create, save, undo, redo, preview isolado, publicação local e export ZIP; não confundir esse mock E2E com o backend validado em integração.

### Evidência confirmada nesta etapa

- `go test ./... -count=1 -timeout=15m`: PASS.
- `go test -race ./internal/agent -count=1 -timeout=15m`: PASS.
- `go vet ./...`: PASS; `go build ./...`: PASS.
- Testes Go focados de unified diff, hash-base, path reservado, persistência do payload hash, CAS e approvals: PASS.
- Agentic Console diff antes da approval: 5/5 testes direcionados PASS.
- Builder Playwright: 2/2 casos PASS, incluindo save/undo/redo/publish/export via mocks.
- Suíte web completa (Vitest/lint/build/Playwright integrado): em execução no momento deste checkpoint.
- Guardrail de integridade do fork e `git diff --check`: PASS antes do último delta de preview; repetir no gate final.

### Limites ainda abertos

- Não há ainda patch multi-arquivo transacional ou branch/worktree isolado, execução de testes iniciada por missão nem repair loop. A escrita é unitária, um step/approval por arquivo.
- O Builder segue editor JSON estruturado, não paridade de drag-and-drop; ainda falta E2E de backend real e deploy público.
- Integrações externas, providers multimídia de produção, desktop companion real e suporte de instalação por OS não foram validados por esta bateria local.
- Paridade total com Manus e “projeto completamente finalizado” continuam critérios abertos; a matriz de paridade é a fonte do status por capacidade.

### Próximo gate

Concluir gates web, rerodar guardrail/diff-check; depois implementar worktree isolado + patch multi-file revisável e um E2E coding loop com teste executado e repair/recovery. Permanecer sem commit/push/publicação nesta missão.


## Checkpoint — gates web finais e recuperação da missão (2026-09-26)

**Estado:** `EXECUTING`; branch `feat/ollama-full-foundation`, `HEAD=8635e30d`; worktree contém alterações não commitadas intencionais. Única worktree Git ativa. Nenhum push, merge, commit ou publicação foi feito.

### Gates da execução web completa

| Gate | Resultado | Evidência |
|---|---|---|
| Vitest | PASS | `npm test -- --run`: 23 arquivos passaram; processo exit 0 |
| ESLint | PASS | `npm run lint`, exit 0 |
| Vite build | PASS com aviso | `npm run build`, exit 0; aviso de chunks minificados >500 kB, bundle principal ~1.5 MB |
| Playwright Chromium | PASS | `npm run test:e2e -- --project=chromium`: 5/5 testes, incluindo shell e Builder |
| Go | PASS | `go test ./... -count=1 -timeout=15m`, `go test -race ./internal/agent -count=1 -timeout=15m`, `go vet ./...`, `go build ./...` em etapa anterior desta sessão |
| Segurança/consistência | PASS | `scripts/check-class-a-plus-integrity.sh`, `git diff --check`, `go list -mod=readonly ./internal/agent` |

A saída do build contém warning de bundle, não erro. Os E2E Builder continuam com API mockada; isso não constitui integração backend real.

### Estado recuperado e próxima decisão

A implementação existente cobre `workspace.write` unitário com preview diff + SHA-256 base + CAS + rename atômico + backup, approvals hash-bound e inspector Git somente leitura. `git worktree list` mostra apenas o checkout atual e o worktree está dirty; por isso, não se cria worktree de tarefa derivada do `HEAD` sem primeiro estabelecer snapshot fiel dessas alterações locais. Em vez de ocultar esse risco, a próxima fatia é estudar e implementar um `workspace.patch` limitado (vários arquivos, diff por arquivo, comparação de hashes, aprovação exata e rollback compensatório em falha observada), documentando expressamente que não é crash-atomic sem journal. Somente após isso projetar worktree isolado/snapshot e runner estrito; não expor comando arbitrário de shell nem contornar `sandbox:execute`.

**Limites registrados:** ainda não existe runner de testes do agente com integração end-to-end; `sandbox.exec` atual é um executor de trechos Python/Node, não runner do repo. Não existe criação/limpeza de branch/worktree no runtime. O patch multi-file planejado não será denominado plenamente transacional até haver recuperação pós-crash demonstrada.


## Checkpoint — workspace.patch revisável (2026-09-26)

**Estado:** `TESTING`; branch local continua `feat/ollama-full-foundation`; sem commit/push/publicação. Worktree contém alterações anteriores e desta missão; nenhuma é descartada.

### Implementação nesta fatia

`workspace.patch` adiciona aplicação de até 16 arquivos (1 MiB por arquivo, 4 MiB agregados) num único step e approval. Reaproveita a capability existente `workspace:write`, sem criar permissão paralela: cada arquivo recebe unified diff + hash base no servidor; a approval SHA-256 vincula tool, risco e input inteiro; as bases são lidas com limite e revalidadas imediatamente antes da substituição, com writers do mesmo processo serializados. Isso não é CAS cross-process. Overwrites mantêm backup artifacts. O planner foi informado do novo tool kind e prefere `workspace.write` quando só há um arquivo. O Console já apresenta o input aprovado integral, e o teste React confirma os diffs de todos os arquivos.

**Sem exagerar a garantia:** preflight evita começar com base obsoleta; se uma falha de execução é observada, a rotina tenta compensar writes anteriores em ordem reversa. Uma alteração concorrente que não corresponda ao conteúdo escrito não é sobrescrita pela compensação; o erro exige inspeção manual. Não há journal/recovery após crash, então `crash_atomic=false`, uma interrupção abrupta pode deixar estado parcial, e o recurso é documentado como parcialmente validado, não como transação crash-safe. Um alvo que retorna erro também não é removido especulativamente, pois o erro não prova propriedade da alteração.

**Correção de terminologia nos registros anteriores:** alguns checkpoints históricos usaram “CAS/compare-and-swap” para descrever hash-check seguido de rename. Uma auditoria provou que essa sequência não é atomicamente compare-and-swap frente a processos externos: ainda há janela entre verificação e rename. O processo Ollama Full serializa os próprios writers e revalida a base imediatamente antes da substituição atômica; editores/processos externos não participam desse lock. Portanto a claim válida é revalidação pré-rename com race cross-process residual, não CAS universal. Os documentos canônicos foram corrigidos; frases históricas acima devem ser interpretadas com esta ressalva.

### Auditoria e ajustes adicionais em curso

Revisão independente identificou: (1) leituras ilimitadas de bases antes do limite; corrigidas para `LimitReader(limit+1)` em preview/write, `workspace.patch`, backup e `workspace.read`; `workspace.list` e varredura de backups agora também leem no máximo limite+1 entradas; (2) rename/hash-check não é CAS cross-process; writers desta instância são serializados, mas a estreita janela externa permanece explicitamente documentada; (3) manifests retornados junto de erros eram descartados; runtime agora os preserva/deduplica na missão e patch propaga os do writer com falha; (4) retry genérico podia duplicar efeitos externos; retries internos só repetem `RiskRead`, jobs associados a missões não-read entram em `failed` sem Nack retryável, e steps não-read persistidos como `RUNNING` falham para inspeção ao reiniciar. Cobertura de fila/runtimes está sendo adicionada. Uma auditoria de engenharia também apontou compatibilidade quando `capabilities` é omitido; o default read-only está documentado e clientes legados precisam conceder `workspace:write` explicitamente para mutações.

### Testes focados confirmados

- Go: `go test ./internal/agent -run 'WorkspacePatch|MultiFilePatch|Approval|AcceptsMultiFilePatch' -count=1 -timeout=5m`: PASS.
- Cobertura inclui diff/hash-base para arquivo existente e novo, preflight de todos os arquivos antes do primeiro write, rollback após falha, proteção contra clobber de alteração concorrente, não remoção de alvo ambíguo, limite/paths duplicados, approval hash-bound, capability `workspace:write` e fluxo runtime de approval seguida de escrita.
- Frontend: `npm test -- --run src/components/AgenticConsole.approval.test.tsx`: 6/6 PASS; inclui renderização do diff para cada arquivo no payload de approval.
- `git diff --check`: PASS após os patches e correção da indentação da árvore.
- Web suite após a primeira implementação da UI: Vitest (suite completa), lint, build, Playwright Chromium 5/5: PASS; não houve mudanças no frontend após esse gate.
- Go suite completa, race em `internal/agent`, vet e build: PASS após as correções da primeira auditoria.
- Segunda reauditoria localizou retry automático da fila após erro de missão contendo write/efeito externo, mais leituras sem teto real em `workspace.list`/quota/backup e perda de artifacts em erros parciais. Alterações correspondentes foram implementadas; testes de integração da fila e limites bounded estão adicionados, e exigem gates completos reexecutados antes de confirmar PASS.

### Arquivos/estado e trabalho aberto

Novos arquivos de runtime/teste: `internal/agent/workspace_patch.go`, `internal/agent/workspace_patch_test.go` e `internal/agent/workspace_patch_runtime_test.go`. Alterados: tool registry, planner, runtime, React approval test e documentação API/árvore/matriz/visão.

Próxima ação após gates: revisar feedback real das suítes, tratar qualquer regressão e fazer auditoria de segurança deste patch. Somente depois iniciar lifecycle de worktree isolado com estratégia explícita para snapshot do checkout dirty; runner de testes e repair loop permanecem pendentes. Não habilitar shell arbitrário, não descartar/commitar alterações e não usar remote.


## Checkpoint — fencing de queue/runtime e hardening de artifacts (2026-09-26)

**Estado:** `TESTING`; branch `feat/ollama-full-foundation`; checkout permanece dirty por mudanças anteriores e atuais. Não houve commit, push, reset, publicação nem alteração de remotes.

**Delta em revisão:** (1) Queue claim usa lease token UUID e Ack/Nack local e Redis exigem o claim exato; Nack Redis e Replay Redis são Lua atômicos. (2) JSONStore `PutMissionIfVersion` usa file lock cross-process (Flock POSIX; LockFileEx Windows) e compara a versão lida do disco; runtime migrou updates de estado/approval/steps para CAS. (3) retry automático só ocorre para read; classificação usa o descriptor registrado como risco mínimo mesmo se o campo legado for vazio; step non-read em `RUNNING` é terminal para inspeção mesmo com `Attempts=0`; enqueue/replay recusam estados non-runnable/failed. (4) ausência de contexto organizacional não expõe/lista/reexecuta job tenant. (5) sandbox usa subdiretório por UUID de execução e cgroup único; strict remonta workspace read-only. (6) processamento de artifacts é bounded em 1 GiB; manifesto de target comprovadamente revertido deixa de ser retornado, mantendo backup artifacts e artefatos do alvo cujo resultado falhou/é ambíguo.

**Hipóteses/gates:** `go test ./internal/agent -count=1 -timeout=10m` foi executado e encontrou uma falha apenas na nova asserção (o teste deixou o segundo legacy write como RUNNING); asserção corrigida e a suíte está rerodando. Seguir com Go completo, race, vet, build, cross-compile Windows do lock, integrity/diff check. Redis real não está instalado/configurado no sandbox; integração Redis pode permanecer sem execução, e isso será identificado como limitação — scripts Lua não serão apresentados como testados em Redis real se não houver servidor.

**Próxima ação:** concluir suíte focada; corrigir qualquer falha sem reduzir assertions; reexecutar gates e solicitar revisão independente do delta de filas/CAS/sandbox. Depois atualizar API/parity/product docs e checkpoint com resultados exatos. Nenhuma mudança de worktree/runner é iniciada antes desses gates.


### Finding adicional durante inspeção final (12:44-03)

O backend cria `.agent-queue` persistente por `DataRoot`, mas o `JobQueue` sincronizava apenas com mutex in-process e o startup reabria todo job `RUNNING` como `PENDING`. Duas instâncias apontando ao mesmo DataRoot podiam sobrescrever o JSON com mapas desatualizados ou disparar retry enquanto um worker ainda executava. Isso é independente do lease token lógico. **Correção:** file lock + reload do mapa em cada transição/listagem, com regressão de duas instâncias. Jobs `RUNNING` não são roubados/invalidados por outra instância na inicialização. **Limitação deliberada:** após crash, um job persistido como `RUNNING` não é reexecutado nem convertido automaticamente em `failed`, pois não existe heartbeat/ownership protocol que diferencie crash de processo ainda ativo; a missão requer inspeção operacional. A fila Redis continua usando lease expirável e fencing próprio. Teste cobre que uma segunda instância observa o claim e não toma seu lease enquanto o primeiro worker conclui.


### Retomada — Redis real, expiry fencing e compatibilidade legacy (2026-09-26 13:11 -03)

**Estado:** `TESTING`; ainda sem commit/push/reset/alteração de remotes. Branch continua `feat/ollama-full-foundation`.

**Correções novas nesta retomada:** integração Redis real encontrou `WRONGTYPE` em `redisReplayScript`: dead-letter é uma Redis List, não Sorted Set. Substituído `ZREM` por `LREM`; `TestDistributedRedisRetriesDeadLetterReplay` passou em Redis local. Auditoria subsequente identificou que token/worker validavam Ack/Nack/heartbeat mesmo após expirar, se o reclaim ainda não havia rodado. Ack/Nack/heartbeat agora também exigem score de lease ainda vigente; `JobQueue` local aplica a mesma regra. Leases Redis/local que excedem `max_attempts` vão a dead-letter no reclaim. Jobs locais legados `RUNNING` sem `lease_until` permanecem inspect-only — não se infere crash a partir de `LockedAt` e não se rouba worker antigo sem protocolo de heartbeat.

**Cobertura/evidência observada:** `go test ./internal/agent` + `go test -race ./internal/agent` passaram na rodada focada anterior; três testes de integração Redis real (retry/dead-letter/replay, claim/reclaim, heartbeat/atomic delayed move) passaram após correção de `LREM`. Em uma rodada integral iniciada antes do último patch, `go test ./...` passou, porém o gate foi interrompido durante race por alteração adicional concorrente; não contar essa rodada como gate completo. Um teste de worker da fila falhou uma vez sob race por assertion imediatamente após handler; assertion foi ajustada para aguardar o Ack persistido e a suite race voltou a passar. Os novos casos de expiry, CAS overflow e event-ID collision e a preservação legacy estão sendo validados pela rodada atual.

**Gate atual:** `gofmt`, `go test ./internal/agent`, race e Redis real estão rodando. A auditoria independente final do delta Redis/local queue, store e sandbox também está pendente. Reexecutar Go full, vet, build, cross-compile Windows, integrity e diff check apenas depois de incorporar findings e congelar alterações; frontend não mudou nesta retomada, e seus gates completos anteriores permanecem documentados acima.


### Revisão independente e fix loop — P1/P2 de leases/artifacts (2026-09-26 13:26 -03)

**Estado:** `FIXING` → `RETESTING`; sem release/commit/push. Auditoria independente não encontrou P0 e validou JSON/Postgres mission-CAS/tenant fencing e transições Redis Lua já revisadas. Findings relevantes: (P1) handler que ignora cancelamento podia continuar side effect após Nack/requeue; (P1) Redis comparava score com relógio do cliente, permitindo clock-skew reclaim prematuro; (P2) `QueueJob.LeaseUntil` não refletia deadline Redis; (P2) artifact helper aceitava raiz symlink em uso direto. Também solicitou Ack/Nack local recusar lease ausente.

**Correções aplicadas:** `drainQueueHandler` cancela e aguarda até 30 s antes de liberar a execução; enquanto drena, tenta heartbeat, e retorna `ErrQueueNonRetryable` para impedir retry automático após handler parar (ou timeout). Integrações local/Redis verificam que job não é liberado durante handler não cooperativo; Redis terminaliza Nack após dreno. Lua Redis usa `TIME` para lease claim/heartbeat/expiry, reclaim e delayed-move; backoff score é relativo à hora do servidor. `QueueJob` agora expõe deadline epoch-ms coerente com `LeaseUntil` em Redis e local. Ack/Nack local rejeitam deadline zero. Artifact manifest/snapshot rejeitam root symlink e comparam o diretório `Lstat` com o handle `os.Root` aberto.

**Gates desta revisão:** rodada anterior completa (pré-findings) passou `go test ./...`, `go test -race ./internal/agent`, `go vet ./...`, `go build ./...`, cross-compile Windows, Redis real, integrity guard e diff check. Essa evidência não cobre os fixes desta seção. Gate focado atual: gofmt, `go test ./internal/agent`, race e integrações Redis reais após os novos patches. Em seguida reexecutar gates completos e atualizar documentação/matriz; clock monotônico multi-host para fila JSON permanece fora de escopo (fila local exige processos no mesmo host; distribuição deve usar Redis), e exactly-once de provider não é prometido.


### Reauditoria independente — drain/cancelamento/timestamps (2026-09-26 13:39 -03)

**Findings independentes:** (P1) drain não renovava lease imediatamente, deixando janela de expiry se o cancelamento ocorria perto do prazo; (P1) `select` poderia aceitar `done` apesar de contexto cancelado e Ack/retry erroneamente; (P2) timestamps legíveis Redis ainda refletiam caller clock apesar dos sorted sets usarem `TIME`; (P3) testes Lua textuais são frágeis fora da integração real. **Política QueueFailed:** o auditor observou divergência no replay low-level. A decisão mantida é fail-closed: `QueueFailed` após efeito não-read ambíguo permanece inspect-only; Runtime e endpoint replayam somente dead-letter após validar missão `READY/RECOVERING` e step seguro. Isso é uma restrição deliberada contra reexecução de efeitos, não um fluxo de recuperação; falta ainda operação de inspeção/recuperação operacional apropriada.

**Correções aplicadas:** `drainQueueHandler` tenta heartbeat imediatamente antes de aguardar; `queueHandlerResult` faz cancelamento dominar conclusão simultânea no worker local e Redis; ambos os `Start` classificam resultado com o contexto atual, e o worker local checa cancelamento antes de novo claim. Redis passa a persistir epoch-ms autoritativos (`available_at_ms`, `locked_at_ms`, `updated_at_ms`) derivados de `TIME` em enqueue, claim, heartbeat, Ack/Nack, reclaim e replay; decoder converte esses campos para os timestamps Go e preserva compatibilidade com registros legados sem epoch-ms. Foram acrescentados testes locais e integrações Redis reais para cancellation/handler nil, drain próximo do expiry, timestamps sob relógio do caller futuro e QueueFailed não-replayable.

**Estado dos gates:** primeira tentativa pós-fix encontrou apenas erro de compilação nas assinaturas de retorno de dois asserts do novo teste local; o teste foi corrigido. `go test ./...`, `go test -race ./internal/agent`, `go vet ./...`, `go build ./...`, cross-compile Windows, integração Redis real, integrity guard e diff-check foram iniciados novamente; resultados ainda pendentes neste checkpoint. Sem commit/push/publicação.


### Fix loop 2 — correção dos achados de reauditoria (2026-09-26 13:57 -03)

A revisão independente pós-fix encontrou (1) expectativa FIFO errada na nova integração Redis (o código usa LPUSH/RPOP corretamente), (2) timestamps RFC3339 de client ainda persistidos contraditoriamente ao lado dos epoch-ms server-time e (3) falta de teste pelo caminho completo `RedisQueue.Start` para cancellation-dominant.

**Ações:** o teste foi reordenado para claimar o job FIFO antes de enfileirar o segundo; agora inspeciona JSON Redis bruto e exige que os campos textuais de data sejam null e os `*_ms` sejam positivos. O Redis Lua grava `created_at_ms`, `available_at_ms`, `locked_at_ms`, `updated_at_ms` com `TIME`; `QueueJob` converte os campos ms para datas legíveis na API Go. Adicionado teste real de `RedisQueue.Start` com handler que retorna `nil` após cancelamento; deve terminar em `QueueFailed`, nunca `succeeded`/retry.

**Evidência focada pós-correção:** `go test ./internal/agent -count=1` PASS; `go test -race ./internal/agent -run 'TestJobQueue(Drain|StartCancellation)' -count=1` PASS; sete testes `TestDistributedRedis*` contra Redis local PASS, incluindo Redis Start e validação de JSON bruto sob clock skew. Full gates e nova revisão independente destes últimos ajustes seguem pendentes. O requisito de jobs QueueFailed inspect-only foi mantido e documentado como decisão fail-closed; falta feature operacional de inspeção/recuperação, portanto não se declara workflow de reparo completo.


**Progresso dos gates (13:58 -03):** `go test ./... -count=1 -timeout=15m` passou em todos os pacotes no checkout pós-fix. `go test -race ./internal/agent`, `go vet`, `go build`, cross-compile Windows, integrações Redis reais, guardrail de integridade e `git diff --check` ainda estão na mesma rodada e não foram antecipados como aprovados.


### Retomada — preflight Redis, FIFO empatado e avaliação Harness (2026-09-26 14:24 -03)

**Estado:** `RETESTING`; sem commit/push/publicação. A implementação Redis agora usa helper Lua comum para verificar tipos esperados de chaves e validar JSON/campos de jobs antes das mutações conhecidas em Claim, move-due, heartbeat, Ack/Nack, reclaim, Enqueue e Replay. O move-due prevalida todos os candidatos antes de gravar, adiciona à lista pending antes de remover do sorted set e ordena scores empatados pelo sequenciador monotônico `available_seq` (com fallback estável por data/ID). Nack/Reclaim/Replay usam chave `sequence` com checagem de tipo; chamadas EVAL atualizadas.

**Evidência focada:** `go test ./internal/agent -count=1 -timeout=5m` PASS; integração real em Redis local (`127.0.0.1:6399`) para wrongtype/falha sem mutação parcial observada nos cenários cobertos, retry/dead-letter/replay, lease/reclaim e FIFO delayed de scores iguais PASS. Foram adicionadas regressões em `internal/agent/distributed_integration_test.go` e contrato estrutural em `redis_queue_test.go`. Isso não estabelece rollback universal para OOM, crash do servidor ou failover; a API foi documentada com esse limite.

**Revisão externa em paralelo ao escopo:** investigado `harness/harness` (GitHub, Apache-2.0) e salvo relatório em `audit/HARNESS_REPO_REVIEW_2026-09-26.md`. Conclusão: serve como referência arquitetural de scheduler tipado/cancelável, audit service por ator/recurso/ação, policy de egress e secret interface; é plataforma SCM/CI/CD/Gitspaces/registry, não coding agent e distinta do HarnessRouter. Matriz/API registram os aprendizados sem declarar código/capacidades importados.

**Próximo gate:** execução integral em andamento: `go test ./...`, `go test -race ./internal/agent`, todas as integrações Redis/agent, `go vet ./...`, `go build ./...`, cross-compile Windows, guardrail de integridade e `git diff --check`. Corrigir qualquer falha antes de concluir.

### Reauditoria Redis — validação de consistência dos índices (2026-09-26 14:39 -03)

A revisão do delta Lua identificou um gap que os testes WRONGTYPE anteriores não cobriam: um membro órfão (sem `job:<id>`), payload cujo `id` não correspondesse à key/index, ou status incompatível podia ser aceito/consumido em alguns fluxos. Corrigido com validação de ID/status antes de `RPOP`, remoções ou transições; Claim, move-due, reclaim, heartbeat/Ack/Nack/Replay e leitura auxiliar agora conferem a identidade esperada conforme o fluxo. Não limpar automaticamente registros inconsistentes; falhar fechado e manter índices para inspeção.

**Regressões Redis reais:** novos testes para pending órfão, ID embutido divergente, delayed órfão, e status inconsistente nos índices pending/delayed/lease. `go test -tags=integration ./internal/agent -run 'TestDistributedRedis(RejectsOrphanAndMismatchedJobRecords|RejectsQueueIndexStatusMismatch|WrongTypesDoNotPartiallyMutateQueue|DelayedJobsWithEqualScoreRetainRetryOrder)$' -count=1 -timeout=3m` PASS em Redis local (`127.0.0.1:6399`).

**Limite confirmado:** as keys do queue namespace são `prefix:suffix` sem hash tag Redis Cluster e o cliente RESP2 não lida com MOVED/ASK; Redis Cluster fica explicitamente não suportado/não validado. Documentação atualizada. Os gates integrais anteriores passaram antes deste fix; estão sendo repetidos agora no checkout atualizado.


### Fix loop — preflight Redis completo e regressões por findings (2026-09-26 15:07 -03)

**Estado:** `FINAL_AUDIT` para a fatia Redis; sem commit/push/publicação. Reauditoria independente anterior apontou: validação incompleta de ARGV, `attempts >= max_attempts` aceito em jobs pending/delayed, Replay sem validar membership da dead-letter list, delayed→pending sem rejeitar ID já pending, índice/payload de missão divergente no Enqueue, formato Redis legacy sem política, Redis Cluster não suportado e documentação que chamava a instância de teste de efêmera sem o teste gerenciá-la.

**Correções implementadas:**
- Scripts Lua validam aridade exata e chaves Redis distintas; Claim/Heartbeat exigem duração positiva e o wrapper ceil-rounds milissegundos com piso de 3 ms e teto de 24 h; Nack valida modo/backoff antes de mutar sequence.
- Decoder comum valida mission ID, ID do job, tipos/limites de attempts e max_attempts, stamps epoch-ms, sequence pending e lease running; pending/delayed no limite de tentativas e registros sem stamps/lease falham fechados para inspeção.
- Replay exige uma única origem na dead-letter list e ausência nos outros índices; delayed move verifica duplicate/conflicting membership antes de LPUSH/ZREM.
- Enqueue exige `payload.mission_id == ARGV mission_id` e rejeita índice de missão apontando para job de outra missão.
- API deixa claro que as integrações consomem `OLLAMA_AGENT_TEST_REDIS_URL`, não inicializam/paralisam o daemon e não provam efemeridade; Redis Cluster segue explicitamente não suportado.
- Adicionadas regressões reais: argumentos inválidos/aridade, lease mínima, tentativas no limite, Replay sem índice de origem, mission index/payload collision, índices duplicados e registros legacy pending/running.

**Evidência:** `go test -tags=integration ./internal/agent -count=1 -timeout=5m` PASS contra Redis local configurado em `127.0.0.1:6399`; `git diff --check` PASS antes das últimas adições. Ainda pendem reauditoria independente do fix loop, gates integrais pós-última alteração (Go full, race, vet, build, Windows cross-compile, Redis temporário controlado e integrity/diff check). Redis Cluster, failover/OOM/crash injection, Redis gerenciado e exactly-once externo continuam fora do que foi comprovado; não declarar paridade total do produto.


### Resultado dos gates abrangentes pós-fix (2026-09-26 15:08 -03)

**Resultado:** PASS no checkout `feat/ollama-full-foundation`, sem commit/push. O comando criou Redis local standalone temporário (bind 127.0.0.1, sem persistência), configurou `OLLAMA_AGENT_TEST_REDIS_URL` e encerrou o daemon temporário via trap no fim. Passaram, em sequência: `gofmt` check; `go test ./... -count=1 -timeout=15m`; `go test -race ./internal/agent -count=1 -timeout=10m`; `go test -tags=integration ./internal/agent -count=1 -timeout=5m` contra Redis temporário; `go vet ./...`; `go build ./...`; cross-compile Windows do pacote agent com `GOOS=windows GOARCH=amd64`; `scripts/check-class-a-plus-integrity.sh`; `git diff --check`. A primeira compilação das regressões teve um erro de assinatura `queue.do`; foi corrigido no próprio teste, e o pacote completo + gate abrangente subsequentes passaram. Revisão independente final dos scripts atualizados continua pendente. Isso fecha apenas a fatia de confiabilidade de fila/runtime; não declara o produto concluído nem paridade total com Manus.


### Fix loop — mission index stale e validação pré-housekeeping (2026-09-26 15:15 -03)

**Estado:** `RETESTING`; branch `feat/ollama-full-foundation`, checkout dirty preexistente + alterações locais desta missão; sem commit/push/publicação. A reauditoria independente anterior apontou um finding Medium: Enqueue podia retornar idempotência para pending/running de mesma missão sem membership real na fila/lease. Também apontou finding Low: Claim fazia moveDue/reclaimExpired antes de validar duração >24 h.

**Correções:** `redisEnqueueScript` recebe agora sete KEYS (mission, job-prefix, pending, delayed, leases, dead, sequence), preflighta tipos e valida antes de retornar que pending tem exatamente uma membership pending/delayed e zero lease/dead, ou running tem lease score igual a `lease_until_ms` e zero pending/delayed/dead. `Claim` valida a duração da lease antes de qualquer housekeeping. Os testes de integração novos cobrem stale index para pending sem membership, running sem lease e duração inválida que não pode mover job delayed.

**Evidência já concluída:** `go test ./...` PASS no código de produção após o fix de Claim e antes apenas da adição do novo caso de teste Lease-preflight; integração real `go test -tags integration ./internal/agent -run '^TestDistributedRedis' -count=1` PASS em Redis standalone temporário local (bind loopback, sem persistência, encerrado via trap), incluindo ambos os testes stale e teste de pré-housekeeping (1.441 s); `gofmt` e `git diff --check` PASS nessa rodada. O toolchain Go 1.26.8 estava instalado em `/usr/local/go/bin`, fora do PATH inicial do shell.

**Ainda em execução:** gate abrangente sequencial com `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `go test -race ./...` e cross-build `GOOS=windows GOARCH=amd64 go build ./...`; reauditoria independente read-only do delta em curso. Não antecipar resultado desses gates. Redis Cluster/failover/crash recovery/effects exactly-once não foram validados.

**Próxima ação:** receber gates e reauditoria, corrigir findings se surgirem, registrar resultado final; só então iniciar especificação/implementação de snapshot/worktree isolado + runner de testes do projeto + repair loop. A matriz continua estimando ~39% dos domínios como localmente validados (12/31); isso não é % ponderado de feature completeness nem paridade Manus.


### Fix loop 2 — corrigir teste falso positivo e fechar cobertura de membership (2026-09-26 15:25 -03)

A auditoria independente pós-fix não encontrou Critical/High no código de produção, mas localizou finding Medium de cobertura: `TestDistributedRedisEnqueueRejectsMismatchedPayloadMission` ainda passava `numkeys=4` depois que `redisEnqueueScript` passou a exigir 7 KEYS, portanto seu `err != nil` podia validar apenas a aridade errada. **Correção:** ajustar chamada de teste para os sete KEYS na mesma ordem da produção. A revisão também indicou lacuna Low de cobertura no caminho Enqueue idempotente; adicionados subcasos para pending-only-in-delayed como sucesso, pending+delayed, pending+dead, e running com ZSCORE divergente como rejeições, preservando índices em erro. Cada subcaso usa prefixo Redis isolado.

A rodada prévia `go test ./... -count=1`, `go vet ./...`, `go build ./...` e `go test -race ./...` passou; a sequência falhou somente no `GOOS=windows GOARCH=amd64 go build ./...`, pois `app/cmd/app` importa `app/webview` e todos os arquivos desse pacote são excluídos pelas build constraints para esse alvo. Não tratar como falha do delta Redis nem alterar tags para forçar verde. Gate de substituição em andamento: `GOOS=windows GOARCH=amd64 go test -c ./internal/agent` para validar o pacote afetado. Também estão em execução nova suíte Go aplicável, teste Redis standalone com casos ampliados e revisão independente pós-fix.


### Resultado pós-correção do teste e gates (2026-09-26 15:26 -03)

**PASS:** `go test ./... -count=1`; `go vet ./...`; `go build ./...`; `go test -race ./internal/agent -count=1`; `GOOS=windows GOARCH=amd64 go test -c -o /tmp/ollama-full-agent.test.exe ./internal/agent`; `git diff --check`; `scripts/check-class-a-plus-integrity.sh`; integração Redis real `go test -tags integration ./internal/agent -run '^TestDistributedRedis' -count=1` contra Redis standalone loopback sem persistência, iniciado/encerrado pelo runner; testes novos de stale index, delayed-only permitido, pending+delayed, pending+dead, lease-score divergente, Claim fail-fast e payload-mission com aridade 7 passaram. Os Redis temporários das portas 16379/16380 foram encerrados após confirmar PID/porta; serviços preexistentes nas outras portas não foram tocados.

**Limite Windows:** uma tentativa de `GOOS=windows GOARCH=amd64 go build ./...` falha por `app/cmd/app` importar `app/webview`, cujo código é excluído pelas build constraints para esse target. O pacote Go afetado `internal/agent` compila como teste Windows com `go test -c`; não mascaramos o problema nem alteramos build tags. O smoke de instalação/runtime Windows não foi executado neste Linux sandbox.

**Revisão independente final:** está sendo feita sobre o teste corrigido e seus subcasos; não concluir o fix loop até recebê-la e tratar findings se houver. Percentual permanece em aproximadamente 39% dos 31 domínios da matriz com estado `VALIDADA LOCALMENTE` (12/31). Esse proxy por domínio não pondera subfeatures nem significa 39% de uma promessa executável fechada; estimativa global honesta continua aproximadamente 40%, com muitas áreas parciais/adapters e validações externas pendentes.


### Fechamento da rodada Redis e transição repo-aware (2026-09-26 15:47 -03)

A revisão independente final dos asserts apontou lacunas que foram fechadas: snapshot de estado agora compara conteúdo completo de pending/dead, conteúdo e scores integrais dos ZSETs, registro JSON do job, mission index e sequence; payload-mission usa 7 KEYS e exige a causa de identidade esperada; todos os 21 callsites de filas na suíte `distributed_integration_test.go` passam por `openRedisTestQueue`, com prefix `ollama:` validado, UUID terminal, cleanup SCAN delimitado ao próprio namespace e contexto novo com timeout. A revisão estática final do helper encontrou **sem findings** nos usos atuais.

**Gates finais da suíte Redis (loopback, sem persistência):** `go vet -tags integration ./internal/agent`, `go test -tags integration ./internal/agent -count=1` e `go test -race -tags integration ./internal/agent -count=1` passaram. Após as duas execuções, `DBSIZE=0`, provando que os testes não deixaram chaves no Redis; daemon foi encerrado. Também passaram as verificações de `gofmt` e `git diff --check`. Os gates gerais anteriormente registrados (`go test ./...`, vet, build, race de `internal/agent` e cross-build Windows do pacote) permanecem verdes; cross-build Windows de `./...` continua limitado pelo pacote `app/webview`, conforme explicado acima.

### Próxima fatia aprovada para implementação local

Não implementar como `git worktree add` sobre o checkout dirty. O levantamento encontrou 48 tracked modificados e 35 untracked no momento da análise, e vários arquivos atuais ainda não existem em HEAD; preservá-los integralmente. Criar snapshot físico isolado sob `Runtime.DataRoot` com manifesto (HEAD/branch/index hash/status, tracked staged+unstaged e untracked não ignorados), sem copiar `.git`, sem tocar `.git/index`, sem seguir symlinks e sem expor conteúdo ignorado/sensível. ToolContext de coding deve apontar só para `SnapshotRoot`; `coding.test` usa perfil de comandos allowlisted em sandbox sem egress e limites; apply à origem é ferramenta separada, approval hash-bound ao fingerprint/patch, com recheck CAS e conflito fail-closed. Snapshot/cleanup deve ser tenant+mission-scoped, preservar origem e suportar inspeção/recovery após falhas. Windows nativo (junction/reparse/process-tree) exige CI real, não apenas cross-compile. Sequência: contratos/persistência → manager + testes unitários → lifecycle/runtime → runner sandbox allowlist → apply/approval/CAS → rotas/E2E → UI. Até esses gates existirem, a matriz mantém coding repo-aware como `PARCIAL`.


### Snapshot core — protótipo isolado (2026-09-26 16:42 -03)

**Escopo entregue nesta fatia:** criada `CreateWorkspaceSnapshot` como primitiva standalone (ainda não chamada pelo Runtime). Ela valida Mission/tenant/project IDs, paths e limites; exige DataRoot separado e valida o Git antes de criá-lo; captura HEAD/branch/status/index hash e arquivos stage-index/untracked não ignorados sem executar ações remotas; não copia `.git`; preserva conteúdo staged/unstaged do working tree; não copia ignored, symlinks, submodules, nomes sensíveis ou conteúdo detectado por `ScanDLP`; limita volume/profundidade/tamanho, registra manifesto hash-bound por arquivo, grava arquivos 0600 (0700 se executáveis), limpa snapshot parcial em erro e rejeita paths não portáveis entre SOs.

**Loops/falhas corrigidos durante testes:** contenção `sameOrWithin` estava invertida; repositório Git sem commit inicial era recusado; `Lstat` de symlink precisava usar `os.Root` sem convertê-lo em erro; chmod era chamado depois de fechar o descritor; segredo chegava a existir em tempfile antes da triagem (agora a leitura é limitada e a triagem ocorre em memória, antes de persistir); DataRoot aninhado podia criar diretórios dentro da origem antes de ser rejeitado (preflight passou a ocorrer antes de qualquer `mkdir`). A primeira auditoria de testes apontou checks que podiam dar falso verde para arquivos omitidos, manifest/status e cleanup pós-falha; a suíte foi ampliada para comparação exata de árvore, conteúdo/hash/status/HEAD/branch/index, preservação de todos os fixtures, arquivo executável, token DLP, symlink de arquivo e pai, submodule, names portáveis, DataRoot inválido e falha depois da primeira cópia.

**Evidência restrita:** os 8 testes top-level de snapshot (incluindo subtestes de path) passaram no Linux, sob `-race`, e compilaram para Windows amd64. Esses testes foram executados em harness que compilou os arquivos reais `workspace_snapshot.go`, `workspace_snapshot_test.go`, `secrets.go` e `project_paths.go`, mais shims temporários removidos ao fim para helpers compartilhados ausentes nesta cópia; portanto isto **não é** `go test ./...` nem valida integração do pacote inteiro. `git diff --check` e `scripts/check-class-a-plus-integrity.sh` passaram. Um probe confirmou que o conjunto de comandos Git local configurado sem pager, hooks/fetch/remotes e sem config global não executou um filtro clean configurado no repositório fixture.

**Bloqueio do checkout recuperado:** `CGO_ENABLED=0 go test ./internal/agent -run '^$'` falha no compile porque o working tree carregado aqui não contém definições que os arquivos preservados referenciam (`withFileLock`, `mediaProcessTool`, `prepareWorkspaceWriteApproval`, `approvalPayloadSHA256`, `ErrApprovalPayloadChanged`; o compilador parou em “too many errors”). Não inventar green nem tratar isso como falha funcional do snapshot: recuperar/reconstruir e testar os helper files é pré-requisito para gates nativos e integração. Nenhuma alteração foi commitada ou publicada.

**Não concluído:** não há integração a `Runtime`, `ToolContext`, persistência/recuperação após restart, lifecycle de retenção/cleanup por missão, runner de testes allowlisted sem egress, aplicação aprovada/CAS do patch ao source ou UI. O snapshot core é protótipo local e não fecha a feature repo-aware nem a paridade Manus. Próxima sequência: recuperar dependências ausentes; fazer o pacote compilar nativamente; criar manager/lifecycle tenant+mission e teste de recuperação/cleanup; executar sandbox runner sem egress; implementar apply separado, approval com SHA-256 e CAS; depois backend E2E e UI.


### Retomada após recuperação do checkout e comparação das árvores (2026-09-26 17:14 -03)

**Estado observado:** checkout em `/home/ubuntu/ollama-full-recovery`, branch local `recovery/ollama-full-snapshot`, HEAD `8635e30dc9e95a1f5b29700169783abc24093ceb`; alterações da missão permanecem locais e não commitadas. A cópia original parcialmente transferida foi preservada; não mudar para `main` nem sobrescrever as alterações sem recuperação completa.

**Comparação criada:** `audit/COMPARACAO_ARVORES_MANUS_OLLAMA_FULL.md` compara dez domínios, sem atribuir percentual agregado e distingue superfície observável do Manus de evidência local/adapters/alvo do Ollama. A conclusão é paridade parcial; o documento não é uma nova auditoria live do Manus.

**Recuperação nativa em curso:** o compile inicial encontrou definições ausentes nesta cópia do working tree. Foram restauradas localmente as primitivas de lock Unix/Windows, escrita atomic/hash-bound de `workspace.write`, ferramenta `media.process` condicionada por approval, leitura limitada e lock do JSON store; ajustados os testes de fila para o atual contrato `QueueJob` com fencing. Adicionados testes para payload binding, CAS de arquivo, backups, DLP, compensação de patch e zero chamadas ao provider antes de approval. Primeira compilação `CGO_ENABLED=0 go test ./internal/agent -run '^$'` passou após restaurar esses helpers; gates de pacote normal/race/vet/Windows estão sendo executados agora e ainda não podem ser declarados verdes.

**Riscos/limites que continuam abertos:** essas definições foram reconstruídas a partir das chamadas, tipos e testes disponíveis, não copiadas de uma fonte original; necessitam revisão independente e integração em `go test ./...`. O lock só coordena escritores que obedecem ao lock; não torna CAS atômico contra um editor externo concorrente. A primitive `workspace_snapshot.go` continua standalone e não integrada ao Runtime; coding.test sem egress, lifecycle por missão, apply/CAS ao source, UI E2E, verificações multi-OS em execução real e os demais gaps da matriz permanecem pendentes. Nada foi commitado nem publicado.


### Hardening pós-review dos helpers (2026-09-26 17:28 -03)

**Correções nesta rodada:** `workspace.write` não aceita mais `suppress_backup` vindo do planner; teste tenta forjar a opção e exige backup no overwrite. `media.process` vincula análise/transcrição aos bytes exatos (SHA-256 + tamanho) lidos com limite antes da aprovação e relidos após; mudanças de conteúdo/tamanho falham com `ErrMediaInputChanged`; DLP roda tanto no planejamento quanto na fronteira de egress do provider. Caminhos relativos de mídia são resolvidos dentro do workspace. `workspace.write`, leitura limitada e `workspace.patch` agora abrem root com verificação de identidade do diretório; patches rejeitam paths duplicados case-insensitive em todos os SOs. Lock Windows não permite compartilhamento de delete, preservando o handle de coordenação. O teste do worker passou a esperar Ack até deadline em vez de observar o estado imediatamente após o handler.

**Gates confirmados nesta revisão:** `CGO_ENABLED=0 go test ./internal/agent -count=1`; `CGO_ENABLED=1 go test -race ./internal/agent -count=1`; `go vet ./internal/agent`; cross-build de testes `GOOS=windows GOARCH=amd64 CGO_ENABLED=0` do pacote; e `go test -tags integration ./internal/agent -count=1` com Redis real isolado, `DBSIZE=0` e servidor temporário explicitamente encerrado. `git diff --check` passou após os edits.

**Gates pendentes:** a primeira tentativa do repositório completo com `CGO_ENABLED=0` falhou por configuração incompatível do runner (SQLite exige cgo; MLX/tree-sitter excluem arquivos nessa configuração), não por regressão do patch. A repetição correta com `CGO_ENABLED=1` e `HOME/TMPDIR` temporários está em execução. Também estão em execução os gates web (Vitest, ESLint, build) e mobile (TypeScript, offline policy tests); não declarar resultado até concluírem.

**Limites abertos:** file lock serializa somente escritores cooperantes; um editor/processo externo ainda pode disputar o último rename de workspace.write/patch. O snapshot Git continua standalone, não integrado ao Runtime nem a um runner sem egress; lifecycle/retention/recovery, apply separado com CAS ao source, E2E repo-first e gaps de paridade seguem pendentes. Não houve commit ou push; branch atual `recovery/ollama-full-snapshot`, base `8635e30d`, destino remoto `origin` no repositório DZ23-LTDA autorizado pelo pedido, mas publicação só após gates e avaliação transparente do escopo concluído.


**Mobile confirmado:** dependências instaladas do lockfile; `npm run typecheck` e `npm run test:policy` passaram (`offlinePolicy: PASS`). Permanecem apenas warnings upstream de módulo Node/experimental type stripping; sem erro.
**Web confirmado:** `npm test -- --run` passou 22 arquivos/206 testes; `npm run lint` e `npm run build` passaram. Observações não bloqueantes do build: bundle principal/minificados acima do limiar de 500 kB e `react-test-renderer` deprecated. E2E Playwright ainda não executado.


### Marca visible corrigida (2026-09-26 17:35 -03)

`AppSidebar` agora exibe `Ollama Full` e monograma `OF`; a página nova do chat também usa `Ollama Full`; o smoke E2E testa a presença da marca nova e ausência do nome antigo. Isso evita colocar um segundo arquivo de logo no sidebar, mas **não é o logo Swole**: não há ativo Swole nesta cópia nem no tree atual de `main` do GitHub; o componente `Logo.tsx` existente continua exibindo o mascote antigo. Não inventar logo de marca. Falta asset aprovado para concluir a troca integral do logo. Após a alteração, os 206 testes Vitest, ESLint e Vite build passaram de novo; build mantém aviso de chunk acima de 500 kB. Chromium foi instalado em diretório temporário; E2E está sendo reexecutado.


**E2E web confirmado:** após instalação temporária de Chromium fora do cache global, `npm run test:e2e` passou: 1 teste Playwright da home local-first em 948 ms (35,7 s total, incluindo build/servidor). A primeira tentativa falhou por browser ausente; ambiente corrigido e rerun passou.


### Gates de revalidação e renomeação ativa (2026-09-26 17:40 -03)

- **PASS (repositório Go completo):** `CGO_ENABLED=1 go test ./... -count=1`, `go vet ./...`, `CGO_ENABLED=1 go build ./...` e `scripts/check-class-a-plus-integrity.sh`, executados com `HOME` temporário e removido ao final. A suite completa concluiu com exit code 0.
- **PASS (frontend/mobile):** testes, lint e build web; `npm run typecheck` e `npm run test:policy` mobile; Playwright E2E da home (`1 passed`) com Chromium em diretório temporário. Aviso existente de chunks Vite >500 kB continua como melhoria de performance, não falha de build.
- **Branding:** rótulos ativos do shell, README/guia, árvore/matriz e scripts de instalação/empacotamento Windows/Linux foram migrados para “Ollama Full”; o E2E mantém uma assertion negativa para impedir regressão do nome anterior. A URL/slug GitHub e nomes de caminhos legados são mantidos por compatibilidade. O asset “Swole” não está presente no checkout, Downloads nem no tree remoto inspecionado, portanto não foi inventado/substituído; a adoção fiel desse logo segue pendente até o arquivo original estar disponível.
- **Limite de escopo:** gates verdes comprovam o incremento local, não paridade integral com Manus nem prontidão de release. Integrações que exigem credenciais, hardware, assinatura/lojas, ambientes distribuídos e jornadas E2E reais continuam fora da evidência local. Não executar/prometer publicação em branch protegida sem passar a auditoria final e confirmar o destino/estado remoto.


**Confirmação de gates após rename:** o gate completo terminou com exit code 0. A primeira execução do integrity guard pós-rename detectou corretamente uma asserção hardcoded para `local/ollama-classe-a-plus`; a asserção e a mensagem do próprio guard foram migradas para `local/ollama-full`, e `scripts/check-class-a-plus-integrity.sh` + `git diff --check` passaram em seguida. A smoke de empacotamento Linux está sendo repetida via `bash` porque o script no checkout não tem bit executável; não se alterou seu modo no repositório.


### Snapshot por missão e egress — incremento atual (2026-09-26 18:02 -03)

**Implementado nesta retomada:** `CreateMissionRequest.IsolateWorkspace`/`Mission` guardam o estado e metadata do snapshot; a criação opt-in exige projeto explicitamente pertencente ao tenant ativo, copia o estado de Git para diretório privado de snapshot e persiste snapshot ID/SHA na missão. A UI `AgenticConsole` oferece checkbox acessível, desabilitado sem projeto e desligado por padrão; regressão Vitest verifica o payload explícito. Uma falha da persistência da missão remove o snapshot ainda não referenciado, coberta por teste regressivo. O caminho de source e diretórios persistentes da missão ficam separados. `secrets.go`/`egress_policy.go` e gates nos managers de connector/MCP bloqueiam payloads reconhecidos como credenciais antes do egress; policy é heuristic e não uma garantia universal de no-egress.

**Evidência desta versão:** teste Vitest focado do console passou (3 testes). Suite web final, lint, build e Playwright E2E local passaram (1 E2E); o build ainda reporta bundle >500 kB. Uma suite Go iniciada enquanto fontes ainda mudavam terminou por encerramento do job/exit 137 durante downloads de dependências, antes de produzir resultado completo — isso não é PASS nem falha de teste. A rerun do `go test ./... -count=1` está em andamento, com `HOME` e `TMPDIR` temporários. Após essa suite, devem ser executados race, vet, build, cross-compile Windows, integração Redis real com verificação DB vazio e integrity guard, na ordem, sem alterar arquivos durante os gates.

**Revisão independente:** três revisores read-only estão em andamento para snapshot/runtime, DLP e media/write/queue; corrigir findings concretos e repetir os gates afetados antes de commit/push. O checkout continua em `recovery/ollama-full-snapshot` sobre `8635e30d`, com working tree extenso modificado e vários arquivos novos. Branch remota futura deve ser uma feature branch não existente (verificada a lista remota), nunca `main`; nenhuma publicação ocorreu.

**Limites atuais:** snapshot é cópia física opt-in, não `git worktree`/branch; não há merge/apply ao source, runner de testes do projeto, repair loop nem política operacional de cleanup/retention por lifecycle. O produto permanece longe de paridade integral e não deve ser declarado finalizado. O arquivo de logo Swole não foi encontrado e não foi substituído por arte inventada; essa parte de branding espera o asset original.


### Retomada de segurança e snapshot — 2026-09-26 18:25 -03

**Estado vivo confirmado:** checkout `/home/ubuntu/ollama-full-recovery`, branch local `recovery/ollama-full-snapshot`, HEAD ainda `8635e30d`; worktree contém alterações pré-existentes extensas e novas alterações da retomada. Remote `origin` aponta ao repositório Ollama classe A+; nenhum commit, push ou publicação foi feito. `OLLAMA_AGENT_TEST_POSTGRES_URL` e `OLLAMA_AGENT_TEST_REDIS_URL` não estão configurados neste sandbox.

**Correções implementadas nesta rodada:** Postgres migra e carrega/salva `workspace_isolated`, `workspace_snapshot_id` e `workspace_snapshot_sha256` em `GetMission`, `PutMission` e CAS; CAS exige versão `expected+1`; cada operação tenant-scoped verifica `rolsuper OR rolbypassrls`, mesmo se o override de bootstrap tiver sido usado. Snapshot resolve Git root real para suportar projeto em subdiretório, valida `os.OpenRoot` contra `Lstat` prévio e contra identidade atual do caminho, verifica manifesto/hash/tenant/project/workspace antes de enfileirar e executar, preserva snapshot após retorno ambíguo de persistência, e reconcilia órfãos com idade >24h no startup, sem deletar referências e respeitando escopo Postgres. Push outbox revalida DLP imediatamente antes da entrega. Hash de approval agora falha fechado se o input seria alterado pela redaction de persistência, evitando aprovação impossível após reload.

**Regressões adicionadas:** round-trip Postgres metadata condicional; raiz substituída após `OpenRoot`; projeto nested; tamper de manifest; sweeper por idade/ref/tenant; PutMission ambíguo preserva snapshot; payload contaminado do outbox nunca atinge o servidor; approval input sensível que seria redigido não recebe hash.

**Evidência parcial:** testes focados iniciais de snapshot/isolation e testes MCP/DLP/connector/push passaram antes das duas últimas alterações (Postgres privilege/CAS e approval hash); devem ser repetidos após congelação. Web: Vitest 22 arquivos/207 testes PASS; ESLint e Vite build concluídos sem erro (warning de bundle >500 kB); Playwright E2E inicialmente falhou porque buscou Chromium no cache padrão, depois passou (`1 passed`, Chromium em `/tmp/ollama-full-playwright-browsers`). O primeiro gate Go completo encontrou um mismatch de erro: o teste MCP de limite recebeu bloqueio DLP genérico; corrigida a semântica para distinguir limite de DLP, e o teste MCP + testes focados DLP/connector/remote MCP/push passou. O rerun correto fail-fast de `go test ./...`, vet, build, race, Windows test compile e integrity está em execução; não declarar PASS até receber resultado. Nenhuma validação Postgres real é possível sem DSN configurado.

**Achados/limites ainda abertos:** sweeper remove órfãos antigos, mas não aplica retenção a snapshots ainda referenciados por missões terminais; `workspace.write/patch` ainda só serializam writers cooperantes e o último rename pode concorrer com editor externo; não há runner de testes em sandbox, repair loop, apply/merge ao source nem E2E repo-first com backend real. DLP segue heurístico e credenciais em headers/env ainda precisam de política própria. Paridade total Manus não comprovada; não declarar produto finalizado. Logo Swole não existe neste checkout e depende do asset original.

**Próxima ação:** aguardar o gate Go fail-fast; se passar, rodar testes focados depois do último diff, validar `git diff --check`, atualizar estado/parity e planejar próxima rodada. Se falhar, corrigir causa e repetir gate afetado. Push permanece condicionado a gates verdes, auditoria final, revisão do diff completo e destino/branch autorizado; sem push nesta retomada.


### Retomada após auditoria e testes distribuídos — 2026-09-26 19:21 -03

**Escopo da rodada:** fechar achados concretos da revisão independente anterior e completar validação de snapshot/persistência; isto não significa paridade completa com Manus nem release final.

**Achados tratados nesta rodada:** (1) runtimes organization-scoped agora exigem `ProjectID` e validam propriedade tenant no `CreateMission`; (2) recovery passa tenant efetivo ao orphan sweep; (3) tools e artifacts de missões isoladas usam `os.Root` do snapshot validado; (4) evento Postgres só é aceito por `INSERT … SELECT` da missão da mesma organização, JSON inválido falha, e replay conflitante diferencia conteúdo divergente sem revelar eventos de outro tenant; JSONStore também valida ownership; (5) falhas de evento deixam de retornar sucesso aparente; (6) Postgres persiste provider/capabilities/snapshot, aplica timeout transacional, CAS exacto, migration serializada e rejeita roles superuser/BYPASSRLS; auth-disabled com store Postgres multi-tenant falha fechado. Teste de webhook foi ajustado para cenário correto com projeto tenant-owned (a proteção não foi relaxada).

**Correção adicional contra TOCTOU de projeto:** o root aprovado do projeto é capturado como `os.FileInfo`; snapshot valida que esse mesmo inode continua no caminho autorizado antes e depois da cópia. Regressão cobre substituição por diretório diferente entre autorização e snapshot.

**Retenção:** o sweeper preserva órfãos por 24 h e agora remove snapshot ainda referenciado somente se a missão for `COMPLETED`, `CANCELLED` ou `FAILED`, tiver `UpdatedAt` presente e estiver terminal há pelo menos 30 dias. `organizationScope` limita a limpeza a um tenant; missões ativas e terminais sem timestamp são preservadas. **Decisão assumida:** 30 dias é retenção local padrão conservadora para snapshots de missões terminais; sweeper ocorre no startup/recovery, não é job periódico.

**Evidência já obtida (mas não considerar gate final do estado congelado atual):** `npm run lint`, `npm test -- --run` (Vitest), `npm run build`, Playwright E2E (1 teste), `git diff --check` passaram. `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `go test -race ./internal/agent`, Windows test compile, integrity guard e `git diff --check` passaram antes das últimas alterações de retenção/identidade; rodada focused após essas alterações passou para regressões de source/project identity, sweeper/retention e runtime; teste do webhook passou. PostgreSQL 16 local com usuário `NOSUPERUSER/NOBYPASSRLS` e Redis real efêmero passaram `go test -tags integration ./internal/agent -run 'TestDistributed(Postgres|Redis)' -count=1`, incluindo RLS, CRUD/CAS, round-trip dos metadados e eventos; um primeiro run falhou na mensagem de replay conflitivo, causa corrigida e rerun passou.

**Auditoria:** duas das três revisões independentes concluíram e seus achados foram inspecionados; o terceiro subagente falhou por erro interno do serviço durante a geração da resposta e foi reiniciado com escopo DLP/media/write/queue em modo read-only. Reexecutar a auditoria independente final após receber seu resultado e corrigir qualquer finding concreto.

**Estado Git:** checkout `/home/ubuntu/ollama-full-recovery`, branch `recovery/ollama-full-snapshot`, HEAD `8635e30dc9e95a1f5b29700169783abc24093ceb`, `origin` confirmado como `DZ23-LTDA/hades-ollama`; working tree extensa continua não commitada; nenhum push. Redis/Postgres de teste foram encerrados e as portas locais de teste confirmadas fechadas.

**Próxima ação exata:** aguardar o reviewer DLP/media/write/queue; integrar findings, congelar fontes; executar novamente todos gates Go (incluindo vet/build/race/Windows/integrity), integração PostgreSQL+Redis descartável e gates frontend; então atualizar checkpoint/parity com resultados observados, revisar diff e secret scan, e só depois avaliar commit/push na feature branch remota (nunca `main`). Limites remanescentes incluem ausência de execução/repair loop completo e git worktree/apply, integração real de credenciais/providers, logo Swole original ausente, jornadas E2E backend/desktop e equivalência integral com Manus ainda não demonstrada.


### Retomada final de DLP, idempotência e binding de approvals — 2026-09-26

**Correções verificadas nesta retomada:** o scanner DLP agora examina chaves sensíveis em JSON embutido/escapado e aceita DTOs struct com limites de bytes/nós; planner remoto e webhook têm regressões que bloqueiam credenciais JSON antes de persistência/egress. Webhook lê um único JSON até 64 KiB, aplica DLP antes do claim e usa estado durável `pending/accepted`, file lock cross-process, lease de pending, retenção/capacidade limitadas e ID determinístico para recuperar aceitação ambígua; a regressão comprova replay estável e retry após falha anterior à persistência. Uploads têm expiry/cleanup, gramática exata de filename gerado e erros HTTP sem paths absolutos. Approval hash agora vincula descriptor/fingerprint do tool e configuração efetiva do connector; execução do connector compara o fingerprint imediatamente antes da chamada e regressão comprova zero requests ao destino alterado. Teste Gin full-router usa bearer tokens de duas organizações e compara respostas (404 indistinguível) em detalhe, eventos, SSE, traces, artifact, run, cancelamento e approval. Teste integration Postgres usa role `NOSUPERUSER NOBYPASSRLS` e raw SQL sem predicates para validar RLS, além de CRUD/CAS e eventos.

**Gates observados:** `go test ./...` passou; `go vet ./internal/agent ./server` passou; `go test -race ./internal/agent ./server` passou; build CGO do executável passou; cross-compile do test binary Windows para `internal/agent` passou; script de integridade Ollama Full passou; `git diff --check` passou. Frontend: Vitest 22 arquivos/207 testes, lint, build de produção e Playwright Chromium (1 E2E) passaram; build ainda emite aviso de chunk grande. Mobile: typecheck e offline policy passaram nesta retomada anterior. Integração real local: PostgreSQL 16 descartável e Redis descartável passaram `go test -tags integration ./internal/agent -run '^TestDistributed(Postgres|Redis)' -count=1`; role/database de teste, Redis e cluster temporário foram removidos e portas confirmadas fechadas. `gitleaks`, `trufflehog` e `detect-secrets` não estão instalados; portanto não declarar secret scan automatizado concluído.

**Revisões e riscos residuais:** revisões independentes precedentes confirmaram controles de snapshot, tenant, egress e fila, mas documentam limitações: RLS usa GUCs caller-settable e o DSN da aplicação executa migrations/pode ser dono das tabelas, logo DB credentials/SQL arbitrário fazem parte da trusted computing base (documentado em `SECURITY.md`); não há role runtime separada de migration/owner. DLP é heurístico, não detecta conteúdo semântico de áudio/imagem, e respostas/erros de Connector/MCP/provider ainda não são uniformemente redigidos antes de retornar ao runtime. Falta rate/cost limit por tenant em rotas multimídia; patch multi-file não é crash-atomic; worker recovery transacional e Redis Cluster não estão comprovados; smoke com credenciais/providers reais e E2E backend/desktop não rodaram.

**Estado de produto/release:** matriz atual contém 32 domínios: 12 `VALIDADA LOCALMENTE`, 13 `ADAPTER IMPLEMENTADO`, 6 `PARCIAL` e 1 `IMPLEMENTADA CONDICIONALMENTE`. Uma ponderação indicativa de status dá 66,4%, mas NÃO representa equivalência funcional ao Manus; muitas integrações são adapters e a paridade total não foi demonstrada. Logo, não declarar projeto completamente finalizado. Logo original Swole continua ausente nesta cópia. Checkout `recovery/ollama-full-snapshot`, ainda com working tree extensa não commitada; `origin` é o repositório esperado; nenhum commit/push foi feito. Não publicar como release final porque os requisitos de paridade do usuário permanecem incompletos.


### Correção final de redação DLP em respostas/erros — 2026-09-26 20:53 -03

**Correção adicionada:** `RedactDLP` agora também detecta propriedades sensíveis em JSON livre, escapado ou aninhado e oculta a string inteira quando não pode confiar na forma do valor. `RedactValue` aplica isso recursivamente aos resultados de ferramentas persistidos; `writeAgentError`, as respostas de erro de deployment parcial, e a saída do handler Grok redigem antes de serializar. Cobertura de regressão valida valor secreto comum (não token-shaped) dentro de JSON embutido em strings, erro HTTP, resultado de tool persistido e resposta Grok. O comportamento é heurístico por chave/token reconhecido, não uma classificação semântica de todo dado sensível; limites e esse caveat estão registrados em `SECURITY.md` e na matriz.

**Gates após as alterações de código:** `go test ./...`, `go vet ./internal/agent ./server`, `go test -race ./internal/agent ./server`, build CGO, cross-compile Windows do test binary, integrity guard e `git diff --check` passaram (job final exit 0). Focados de segredo/resultados/API/Grok também passaram. Gates web ainda aplicáveis, sem código web alterado: Vitest 22 arquivos/207 testes, ESLint, build e Playwright E2E (`1 passed`) passaram. Mobile typecheck/offline-policy passaram. Integração PostgreSQL 16 descartável com role `NOSUPERUSER NOBYPASSRLS` e Redis real passou na execução anterior desta retomada; os serviços e banco temporários foram removidos. Scanner automatizado de segredos não disponível (`gitleaks`, `trufflehog`, `detect-secrets` ausentes).

**Situação:** review independente focado desta última redação ainda está em andamento; integrar achados concretos e repetir gates atingidos. Nenhum commit/push ocorreu; checkout continua `recovery/ollama-full-snapshot` com working tree extensa. A matriz agora tem 33 domínios: 13 `VALIDADA LOCALMENTE`, 13 `ADAPTER IMPLEMENTADO`, 6 `PARCIAL` e 1 `IMPLEMENTADA CONDICIONALMENTE`; 13/33 = 39,4% estão plenamente validados localmente, métrica de status da matriz e não promessa de conclusão funcional. Paridade total com Manus, E2E backend/desktop de jornada longa, runner/repair loop de coding e logo Swole original continuam pendentes.


### Fechamento dos bypasses DLP encontrados na auditoria independente — 2026-09-26 21:11 -03

**Achados e correções:** a revisão identificou (a) JSON nested com chaves Unicode-escaped, (b) mais de uma camada de escape, (c) propriedades sensíveis depois de uma propriedade benigna e (d) nome de chave acima de 256 bytes. O scanner agora percorre todas as propriedades em cada camada, decodifica apenas escapes Unicode aplicáveis à camada (paridade de barras invertidas respeitada), limita inspeção a oito camadas e falha fechado acima do limite ou para chaves com mais de 256 bytes. `RedactDLP` oculta toda a string nesses casos; `ScanDLP` barra o payload na fronteira de egress. Os testes cobrem escapes Unicode da chave e delimitadores, JSON aninhado, chave custom camelCase após `status`, chave longa e nesting acima do limite.

**Revisão independente posterior à correção:** confirmou os quatro casos reportados bloqueados/redigidos e não encontrou bypass concreto nos padrões/paridade examinados; verificou 128 combinações de chaves `api_key` literal/Unicode em até 12 níveis, sem falso seguro, e nomes não sensíveis equivalentes permaneceram permitidos. Limites restantes: detector heurístico por tokens/chaves, sem classificação semântica de PII/dados comerciais; policy específica de credenciais em headers/env ainda pendente.

**Gates de código após a última alteração:** `go test ./...`, `go vet ./...`, `go test -race ./internal/agent ./server`, build CGO, cross-compile Windows do test binary, integrity guard e `git diff --check`: todos PASS (job `job_3celQonv`, exit 0; probes/testes focados também passaram). Depois da alteração apenas documental em `SECURITY.md`, integrity e `git diff --check` foram repetidos e passaram. Sem portas/serviços temporários em execução. Evidência web e mobile da retomada: 22 arquivos/207 testes Vitest, lint, build, 1 E2E Playwright; mobile typecheck/offline policy — PASS. PostgreSQL/Redis descartáveis passaram os integration tests na rodada anterior e foram limpos.

**Estado e decisão de release:** checkout continua na branch `recovery/ollama-full-snapshot` com cerca de 75 arquivos e alterações não commitadas; `origin` continua `DZ23-LTDA/hades-ollama`; nenhum commit ou push realizado. A matriz tem 33 domínios (13 validados localmente, 13 adapters, 6 parciais, 1 condicional); somente 39,4% estão no status “VALIDADA LOCALMENTE”, que é contagem de domínios e não percentual global de conclusão. Paridade total com Manus não foi demonstrada; faltam jornadas E2E backend/desktop, runner/repair loop de coding, integrações reais e logo Swole original. Não declarar projeto finalizado nem publicar como release final nesta etapa.


### Ancoragem por descritor do sandbox.exec em Linux — 2026-09-26 21:17 -03

Ao mapear o próximo gap de coding, foi encontrado um boundary adicional: `sandbox.exec` reabria o workspace por pathname, apesar de o Runtime manter o snapshot em um `os.Root` ancorado. Corrigido: execução reaproveita `ToolContext.WorkspaceRoot` para diretório temporário, código, launcher e limpeza; em Linux, abre `WorkspaceRoot.Open(".")`, herda esse diretório como fd 3 e faz bind mount de `/proc/self/fd/3` dentro do namespace. Assim, trocar o nome do diretório após abrir o snapshot não redireciona o código à árvore substituta. O teste `TestSandboxExecUsesAnchoredWorkspaceAfterPathReplacement` passou e observou somente o arquivo da árvore original. Revisão independente read-only confirmou o fluxo, incluindo fd herdado, mount e cleanup, e não encontrou achado concreto; quatro regressões relacionadas passaram.

**Gates após a alteração Go:** `go test ./...`, `go vet ./...`, `go test -race ./internal/agent ./server`, build CGO, compilação Windows do test binary, integrity e diff-check: PASS (job `job_snmXL2qD`, exit 0). Atualizadas `SECURITY.md` e `docs/agentic/PARITY_MATRIX.md`; ressalva explicitada: fallback não-Linux continua best-effort/path-based, sem a mesma garantia de inode por descritor; strict Linux ainda depende de namespace/seccomp/cgroup delegado e falha fechado se indisponível.

Este reparo não implementa o ciclo de coding equivalente a Manus: worktree/branch isolado, runner de testes do projeto dentro do sandbox, repair loop, revisão/commit/merge e E2E repo-first continuam gaps. Estado de release permanece igual: branch `recovery/ollama-full-snapshot`, alterações amplas não commitadas, nenhum push; não declarar paridade ou conclusão global.


### Retomada: inspeção Git, snapshot ancorado, DLP de armazenamento e isolamento — 2026-09-26 22:40 -03

**Escopo desta retomada:** continuar a hardening review no branch existente sem descartar trabalho anterior. A feature `git.repo.inspect` foi mantida opt-in/read-only e de escopo limitado a artefatos persistidos; planner e resposta reportam essa limitação sem afirmar que observam o workspace inteiro.

**Achados independentes e disposição:** (1) snapshot destination TOCTOU foi confirmado: APIs de path podiam criar conteúdo fora do `DataRoot` após troca concorrente do path. Corrigido com abertura componente a componente e handles `os.Root` para data/tenant/mission/snapshot/tree, criação de subdiretórios, temp files, chmod pelo descritor aberto, rename, manifesto e cleanup. Regressão `TestCopyWorkspaceSnapshotFileRemainsAnchoredAfterDestinationSwap` troca a árvore já aberta por symlink e comprova que o conteúdo fica no diretório inode originalmente aberto. (2) auditor DLP confirmou cobertura nos boundaries revistos e **não** confirmou o alegado bypass de deploy: nome/destino/conteúdo dos arquivos já eram varridos antes de chamar provider; `TestDeploymentBlocksSecretContentBeforeProviderSideEffects` passou e exige zero requests. (3) leaks de cache de missão foram fechados: JSON/memory Store guardam cópias redigidas; `Runtime.CreateMission` e helper de falha devolvem valores redigidos; arquivos legados de missão/evento são regravados sob lock. (4) tokens push persistidos agora usam `encryptCredential`/AES-GCM sob `OLLAMA_AGENT_CREDENTIAL_KEY`; registros plaintext legados migram no startup; falha ao cifrar/decifrar impede serviço de carregar. Testes cobrem at-rest e restart/migração. (5) `Runtime.WithOrganization` aplica `organizationScopedStore` a Postgres, JSON e memory; métricas agregadas/Prometheus ficam 404 quando auth multi-organização está ativo. A RLS PostgreSQL ainda usa GUCs que a própria role do DSN pode definir: não protege uma role de DB comprometida ou SQL arbitrário. Esse limite está documentado em `SECURITY.md`; não há exploração HTTP direta confirmada nesta revisão, e uma separação real de roles runtime/migration é trabalho operacional/arquitetural pendente.

**Documentação atualizada:** `SECURITY.md`, `docs/agentic/PARITY_MATRIX.md` e `docs/agentic/INTEGRATIONS.md` descrevem ancoragem, escopo de resultados Git, redaction, tenant fallback, métricas, chave de criptografia de push e limitações RLS/DLP. Deployment foi registrado como controle já existente e validado, sem reimplementar código duplicado.

**Gates desta edição (focados, não gates finais):** passaram `go test ./internal/agent -run 'Test(DeploymentBlocksSecretContentBeforeProviderSideEffects|DeploymentManagerGenericProvider|JSONStoreRedactsMissionValuesInMemoryAndDisk|OrganizationScopedStoreEnforcesMemoryStoreOwnership|PushServiceEncryptsTokensAtRestAndReloads|PushServiceMigratesLegacyPlaintextTokenFile|PushServiceRegisterRollsBackOnPersistenceFailure|PushServiceBlocksSensitiveDirectCallBeforeEgress|RuntimeFlushPushOutboxDeliversAndRemovesItem|RuntimeFlushPushOutboxRetainsFailedDelivery|RuntimeCreateMissionReturnsRedactedMissionValues)$' -count=1`; `go test ./server -run 'Test(GlobalMetricsHiddenWhenOrganizationAuthenticationIsEnabled|MissionRoutesHideForeignMissionIDs|RegisteredMissionRoutesBindBearerTokenToOrganization)$' -count=1`; e `go test ./internal/agent -run 'WorkspaceSnapshot|Snapshot|VerifyWorkspaceSnapshot' -count=1` (incluindo o teste de replacement); `gofmt` e `git diff --check` também passaram. Ainda falta o gate completo pós-freeze para `go test ./...`, vet, build, race, test compile Windows, integrity, PostgreSQL/Redis disposable integration se disponíveis, e web/mobile conforme mudanças aplicáveis; revisões anteriores e gates em outro estado não substituem essa execução.

**Estado atual:** branch `recovery/ollama-full-snapshot`, working tree extensa sem commit; nenhum push nesta retomada. Secret scanner `gitleaks` foi instalado no sandbox e usado em temp tree em rodada anterior; repetir contra o estado final sem imprimir achados/valores. O conjunto completo ainda não é release final: paridade integral com Manus, ciclo repo-first com worktree/runner/repair, integrações com credenciais reais e staging/E2E longo seguem pendentes. `OLLAMA_AGENT_CREDENTIAL_KEY` exige provisionamento externo e estável em deploy com push persistido. Não declarar projeto completamente finalizado nem fazer push sem gates completos e autorização de destino atual confirmada.

**Próxima ação concreta:** terminar regressões de migração/locks se houver falha, congelar fontes, rodar os gates completos em sequência com paralelismo Go limitado, integrar qualquer finding concreto, secret scan silencioso, diff review, e gravar evidência final. Destino permitido continua a branch de recuperação, mas esta sessão não deve executar publicação remota sem revisar autorização atual e estado do destino.


### Retomada após compactação: handoff snapshot e consistência de persistência — 2026-09-26 23:24 -03

**Alterações desta continuação:** (1) Runtime agora recebe/retém handles ancorados para DataRoot, snapshot e árvore; o handoff valida identidade dos diretórios, cria o workspace relativo ao tree root e lê o manifesto sob o snapshot root; a limpeza de rollback fecha handles e remove sob roots ancorados. (2) orphan sweeper percorre e remove diretórios com `os.Root`/operações relativas, rejeitando substituição de storage parent; regressão simula symlink no parent. (3) permissões de snapshot: Unix chmod via descritor aberto; Windows aplica DACL protegido owner-only e valida identidade do path/handle. (4) approval reason é redigido antes de persistir, emitir evento e retornar; nova regressão cobre payload JSON com token. (5) JSONStore agora faz deep-copy de mission/event graphs no ingresso/egresso, redige e migra arquivos legados sob lock, e serializa `PutMission` persistente com leitura/validação de owner/version/write no mesmo lock; regressões cobrem mutação aninhada de caller/resultado e stores persistentes stale/foreign. (6) push subscriptions/outbox usam stable file lock, reload e read-modify-write serializado, chaves de ID vinculadas a tenant/user/token, tokens cifrados com migração fail-closed, validação DLP de registros legados, deep-copy e regressões concorrentes.

**Evidência desta retomada:** passou `/usr/local/go/bin/go test ./internal/agent -run 'Test(CreateWorkspaceSnapshot|PrepareWorkspaceSnapshot|SweepOrphanedWorkspaceSnapshots|CopyWorkspaceSnapshotFile|ApprovalReasonDLP|Runtime.*(Snapshot|Mission))' -count=1`; depois passou `/usr/local/go/bin/go test ./internal/agent -run 'Test(JSONStore|OrganizationScopedStore|Push|RuntimeFlushPushOutbox|ApprovalReasonDLP|CreateWorkspaceSnapshot|PrepareWorkspaceSnapshot|SweepOrphanedWorkspaceSnapshots|CopyWorkspaceSnapshotFile|Runtime.*(Snapshot|Mission))' -count=1`; cross-compile de test binary Windows `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c ./internal/agent` passou. `gofmt` executado nos arquivos alterados e `git diff --check` anterior passou; repetir no gate final.

**Agentes independentes:** revisores de JSONStore e push/outbox terminaram e suas alterações/testes foram inspecionadas. Suas regressões focused individuais passaram (JSONStore do agente; push/outbox integrado ao teste focused acima). Nenhum dos agentes fez commit/push. Ainda não houve auditoria final read-only de todo o diff pós-correções.

**Próxima ação concreta:** revisar os deltas restantes e contratos, rodar suite completa Go/vet/build/race, cross-compile Windows depois do freeze, integrity guard, frontend/mobile e integração Postgres/Redis descartável quando possível; realizar auditorias independentes pós-fix, secret scan sem imprimir valores e atualizar SECURITY/parity com fatos. Release/paridade permanece não comprovada; nenhum commit/push nesta retomada.


### Gates e regressão de payload legado — 2026-09-26 23:28 -03

A primeira passagem completa `go test ./... -count=1` falhou em um teste: `TestRuntimePushOutboxRechecksDLPBeforeNetworkDelivery` tentava gravar intencionalmente um registro sensível pelo método `persistLocked`, mas a nova validação correta do método impediu o fixture de ser criado. Corrigido o teste para simular arquivo legado corrompido escrevendo o JSON de fixture diretamente; agora verifica zero chamadas de rede, falha da reivindicação e que bytes persistidos não são alterados. O foco `go test ./internal/agent -run 'Test(RuntimePushOutboxRechecksDLPBeforeNetworkDelivery|PushOutbox|PushService|RuntimeFlushPushOutbox)' -count=1` passou. A nova execução completa Go está em andamento; seu resultado ainda não foi confirmado.

Gates web/mobile pós-hardening concluídos com exit 0: Vitest, ESLint, build de produção, Playwright E2E (1 teste), mobile typecheck e policy offline. Secret scan gitleaks na cópia de arquivos alterados/untracked reportou 13 candidatos em 6 arquivos, todos arquivos `*_test.go` com fixtures sintéticas de formato credencial; nenhum candidato em fonte de produção/documentação. Nenhum valor foi impresso. O scanner, portanto, reporta os fixtures de teste, não é um resultado “zero findings”.

Auditoria independente final pós-correção está rodando em quatro frentes read-only (snapshot, JSONStore/tenant, push/DLP, approval/rotas). Não publicar nem concluir até integrar achados e obter o resultado da nova gate Go completa.


### Correções após auditoria independente de push — 2026-09-26 23:32 -03

**Achados confirmados pelo revisor read-only:** (a) stale worker completava/falhava item depois que outro claim recuperava o lease; (b) `NotifyOrganization` copiava até 500 bytes de resposta do provider para o erro, expondo dados arbitrários a callers/logs; (c) endpoint HTTP usava `HasPrefix`, aceitando hosts `localhost.evil`/`127.0.0.1.evil` e userinfo. Não havia finding crítico relatado; havia três findings major.

**Remediação aplicada:** cada `ClaimDue` gera UUID `LeaseToken`; `Complete`/`Fail` exigem token atual e recusam token obsoleto; runtime encaminha o token do claim. URL é parseada como absoluta sem userinfo; HTTP só permite `localhost` exato/IP loopback; HTTPS aceita outros hosts; cliente não segue redirects. Respostas e erros de transporte não incluem body/URL não confiáveis (status HTTP é preservado). Docs SECURITY e parity registram limite at-least-once: fencing protege estado, mas provider ainda pode receber duplicata se worker pausado já enviou antes do lease expirar.

**Regressões focused passaram:** `go test ./internal/agent -run 'Test(PushServiceRejectsNonLoopbackHTTPAndEndpointUserinfo|PushServiceDoesNotEchoProviderErrorBodyOrFollowRedirects|PushOutboxFencesStaleWorkerCompletionAndFailure|PushOutbox|RuntimePushOutboxRechecksDLPBeforeNetworkDelivery|RuntimeFlushPushOutbox)' -count=1`. A execução completa Go corrente foi cancelada porque avaliava código antigo; iniciar nova somente após coletar e integrar demais revisões independentes.


### Retomada final: tenant deployment, provider errors e gate — 2026-09-26 23:59 -03

**Alterações pós-auditoria:** providers de deployment agora normalizam `organization_id` na configuração e, no modo autenticado, catálogo, pedido de approval e execução usam somente a lista scoped; o exemplo `examples/agent-deployments.json` marca IDs de organização como placeholders. Mensagens não-2xx dos providers Media, Deployment, Connector, MCP remoto e Push não ecoam bodies/URLs arbitrários; docs `SECURITY.md`, `docs/agentic/INTEGRATIONS.md` e `docs/agentic/PARITY_MATRIX.md` foram alinhados ao limite. O helper Windows de snapshot abre o diretório explicitamente com `WRITE_DAC`/`READ_CONTROL`, e o novo teste windows-native verifica DACL protegido, ACE apenas do SID do serviço, direitos e herança a filhos; o test binary Windows compilou, mas não foi executado nativamente.

**Teste/falha e correção:** os testes focados do agente e rotas server passaram; cross-compile Windows AMD64 passou. A primeira repetição da suíte `go test ./... -count=1 -p=2` falhou em `TestRegisterConnectorRejectsCrossTenantAndRawSecretFields`: o handler de registro de connector havia sido ligado ao mapper de lifecycle (404), alterando indevidamente a semântica de uma tentativa de criação cross-tenant que deve continuar 403. Corrigido para usar `writePluginRegistrationError`; os testes direcionados de registro cross-tenant e lifecycle 404 passaram. A suíte completa pós-correção ainda precisa ser repetida.

**Secret scan:** gitleaks verificou 104 arquivos de texto alterados/untracked; encontrou 14 padrões somente em fixtures `*_test.go`, nenhum em fontes/documentação fora de testes. Nenhum valor foi emitido. Uma tentativa anterior com `--redact=100` era opção inválida na versão instalada; a execução válida usou `--redact`, escreveu relatório e confirmou as 14 ocorrências em testes.

**Próxima ação concreta:** aguardar os três auditores independentes pós-correção (snapshot, Store/Postgres, push/provider egress), aplicar e testar findings concretos; congelar árvore; repetir `go test ./...`, `go vet ./...`, race, build/CGO, Windows compile, integrity, frontend/mobile aplicável e PostgreSQL/Redis descartáveis; revisar diff/scan e então commit/push apenas para `origin`/`recovery/ollama-full-snapshot` já autorizado no pedido histórico. Não declarar paridade integral ou entrega global concluída: a matriz ainda documenta gaps de coding repo-first, E2E longa, integrações reais e teste nativo Windows.

STATUS: CONTINUE


### Bloqueio encontrado pela auditoria independente pós-fix — 2026-09-27 00:07 -03

**Gates:** a repetição da suíte completa pós-correção ainda estava em andamento, mas foi encerrada antes de validar o gate inteiro; não contar como PASS. Uma execução anterior da suíte falhou devido ao mapper de erro errado no registro cross-tenant; correção aplicada, regressões direcionadas `TestRegisterConnectorRejectsCrossTenantAndRawSecretFields` e `TestConnectorLifecycleHidesForeignIDsAsNotFound` passaram. A sequência completa Go posterior foi encerrada quando os revisores reportaram findings adicionais.

**Achados confirmados por três revisões read-only (não corrigidos ainda, portanto release bloqueado):**
- PostgreSQL RLS concede acesso privilegiado com GUCs customizáveis pelo próprio DB role (`app.system_access`, `app.current_organization_id`); `NOSUPERUSER/NOBYPASSRLS` não protege contra alguém com o login runtime ou SQL arbitrário. A resposta robusta talvez exija role runtime sem capacidade de forjar contexto (ou context HMAC com segredo não acessível via a mesma query role e função/roles de owner), separação de migrator/admin DSN e alteração de bootstrap/contrato operacional; não aceitar renomear GUC como correção. Um agente read-only está avaliando o desenho.
- Redaction DLP de missões legadas omite metadados de artefato (`Name`, `Path`, `MediaType`); JSON e Postgres Get/List podem retornar e manter esses valores sem redaction.
- `PutMission` permite avanço de versão maior que 1; `PostgresStore.GetMission` não aplica o mesmo ID exato/validator do JSONStore e trims alias de whitespace.
- Media provider aceita userinfo nas URLs de configuração; downloads retornam erros de transporte incluindo URL/query do provider; Deploy retorna transporte cru contendo host/path. Connector SSRF só rejeita IP privado após conectar TCP. Remote MCP propaga `JSON-RPC Error.Message` arbitrário.
- Snapshot: leituras Git por pathname podem observar repo substituto entre leituras enquanto cópia usa source handle antigo; identidade final pode não detectar troca/restauração. Windows DACL não valida ownership do DataRoot pré-existente. Falha criando leaf pode deixar diretórios pai vazios que sweeper não poda. Nota checkpoints/tests descreve teste Windows como verificação apesar de apenas cross-compile disponível (sem execução nativa).

**Estado:** a revisão confirma que hardening de push/outbox previamente trabalhado não tem finding adicional nestas frentes, mas os achados acima são independentes e bloqueiam a afirmação de isolamento/egress total. Correções distribuídas a três workstreams com escopo de arquivo separado; RLS sem solução segura aprovada/configurável será explicitamente mantido como blocker, nunca declarado corrigido por placebo. Reexecutar gates só depois da integração dos findings e freeze.

STATUS: CONTINUE


### Continuation: snapshot Git pinning, metadata DLP, and RLS decision — 2026-09-27 00:30 -03

**Hardening concluded in this slice:** completed the Linux descriptor-bound Git inventory plumbing, anchored `.git/index` hashing to the already-open source root, added a deterministic source-path replacement regression, and added rollback cleanup of only the snapshot parent directories created by the current attempt (and only while still empty). Added regression coverage for a leaf-creation failure. The DLP key/value detector now recognizes `token=`/`credential=` assignment forms used in artifact metadata, and metadata redaction regressions pass in the full `internal/agent` suite. Snapshot-focused tests passed; Windows AMD64 agent test-binary cross-compilation passed (not native Windows execution).

**Full-suite finding and correction:** the combined `go test ./internal/agent ./server -count=1` run passed `internal/agent` completely. The server package had one failure in `TestDeploymentApprovalRequiresAdminAndNonceBeforeProviderCall`: its authenticated fixture used a now-disallowed global deployment provider. The fixture now explicitly binds `organization_id=local`; targeted deployment approval/catalog/foreign-ID and plugin lifecycle tests passed. A new authoritative full release-gate sequence has started; its outcome is not yet known.

**RLS disposition remains BLOCKED:** the read-only architecture review confirms the present `app.system_access` and `app.current_organization_id` GUC policies are spoofable by the database login, the sole DSN auto-migrates and may own policy objects, the root Runtime/background worker performs unscoped operations, and Compose bootstraps a role that is normally superuser. No safe drop-in RLS change is implemented. A credible fix needs distinct runtime/migrator identities and a non-forgeable tenant context (or tenant-specific DB identity), system worker separation, schema/backfill migration, and adversarial tests; renaming GUCs or adding `FORCE RLS` is insufficient. `SECURITY.md` and the parity matrix now state this as a production-isolation blocker. **Do not publish this branch as enterprise-grade PostgreSQL tenant isolation or mark the overall project complete while this remains open.**

**Current verification:** `go test ./internal/agent` passed during the combined run; targeted server regressions passed after fixture correction; targeted snapshot suite and Windows cross-compile passed; `gofmt` and `git diff --check` passed. Full Go test/vet/race/build/Windows gate sequence is running. Prior web, Playwright and mobile gates passed before these backend-only changes; frontend sources are unchanged in this slice. No commit or GitHub push has been made.

STATUS: CONTINUE — release remains blocked pending completed gates and PostgreSQL architecture resolution.


### Documentação e gate congelado pós-correções — 2026-09-27 00:32 -03

**Correção de fixture:** investigação da tentativa anterior confirmou `internal/agent` PASS integralmente e um único failure de `server`: o provider `self` do teste autenticado não tinha `organization_id`. O fixture agora declara `organization_id=local`; a seleção direcionada de aprovação, catálogo scoped, IDs estrangeiros, registro cross-tenant e lifecycle 404 passou.

**Documentação:** `SECURITY.md` agora declara sem ambiguidade que a RLS PostgreSQL atual é bloqueador para isolamento enterprise, lista a política GUC caller-settable e as pré-condições de correção. O guia também delimita Git descriptor-bound no Linux versus fallback path-based noutros sistemas, cleanup de pais vazios e os controles de DNS-pinning/provider. `docs/agentic/PARITY_MATRIX.md` atualiza evidência de snapshot/DLP e reclassifica Auth/RBAC/RLS para `PARCIAL; PostgreSQL RLS BLOQUEADO`; a linha de egress continua `PARCIAL` e enumera limites restantes.

**Secret scan:** gitleaks foi executado num corpus isolado de 107 arquivos de texto alterados/untracked, com redaction ligada e sem exibir nenhum valor. Foram reportadas 15 correspondências em 8 arquivos, todas em `*_test.go`; `non_test_findings=0`. Saída/relatório ficaram em `/tmp` e não no repositório.

**Em execução:** revisão independente pós-fix em três áreas (snapshot; Store/Postgres/RLS; egress/providers) e sequência backend autoritativa (`go test ./...`, vet, race do package agent, build CGO, Windows test-binary cross-compile e diff-check). Resultados ainda não disponíveis neste checkpoint; não inferir PASS.

STATUS: CONTINUE — não fazer commit/push enquanto testes/auditoria convergem; a limitação estrutural de PostgreSQL RLS permanece aberta.


### Gates autoritativos e integração real — 2026-09-27 00:37 -03
**Go:** sequência congelada em `/tmp/ollama-full-release-gates-postfix-20260927.log` concluiu `ALL_PASS`: `go test -p=2 ./... -count=1`, `go vet -p=2 ./...`, `go test -race -p=2 ./internal/agent -count=1`, `CGO_ENABLED=1 go build -p=2 ./...`, cross-compile Windows AMD64 de `internal/agent` e `git diff --check`. A suite reportou 83 linhas de pacote Go (`ok`/`?`).
**Integração:** `go test -tags integration ./internal/agent -run '^TestDistributed' -count=1` passou contra Postgres 16 local com role não-superuser/NOBYPASSRLS e Redis autenticado descartável; inclui persistência/CAS/DLP/RLS/filas. Limitação importante: os testes exercitam contexto RLS comum, mas não tentam forjar `app.system_access=1`; portanto não provam resistência a spoofing do GUC e não alteram o blocker de produção já registrado.
**Limpeza:** o job não encerrou os processos temporários apesar de reportar PASS; foram identificados por nomes/PID exclusivos desta execução e limpos manualmente. Verificação posterior: database e role retornaram contagem zero, Redis porta 16386 fechado/sem arquivos temporários e cluster PostgreSQL parado.
**Em aberto:** três revisões independentes pós-fix ainda estão executando. O isolamento PostgreSQL permanece explicitamente bloqueado até desenhar/provisionar identidades runtime/migrator/system separadas, remover GUC de bypass, representar workers sem acesso root no processo de API, backfill/migrar com segurança e passar testes adversariais. Não publicar como isolado em nível enterprise, nem declarar missão concluída.
STATUS: CONTINUE — gates Go e integração corrente passaram; aguardar relatórios independentes e prosseguir no desenho/implementação de RLS não-forjável.


### Remediação independente de segurança — 2026-09-27 01:16 -03

**Escopo corrigido neste ciclo:** (1) `Runtime.Run` passa a exigir fingerprint planejado válido e igualdade do descriptor para cada ferramenta, inclusive `RiskRead`; regressão de substituição do `git.repo.inspect` confirma bloqueio antes da execução. (2) Captura de arquivos do snapshot relê e compara conteúdo pelo mesmo descritor, além das validações existentes; regressão determinística altera um byte sem alterar tamanho/mtime e confirma rejeição. (3) rota HTTP de artefato remove snapshot temporário verificado ao encerrar a entrega; teste de servidor confirma HTTP 200 e ausência de novos `.ollama-artifact-*.tmp`. (4) execução Git no Windows usa somente diretórios allowlisted, sem herdar PATH ambiental. (5) Remote MCP exige `CallGlobal` explicitamente privilegiado para servidores ownerless; `Call` sem escopo falha, e requests tenant-scoped só acessam configuração daquele tenant. (6) Connector e Remote MCP sanitizam saídas 2xx recursivamente antes de devolver resultados; deploy remove URL com query de credencial; regressões de sucesso verificam não exposição. (7) Push Outbox agora rejeita tentativas inválidas e aplica fencing por expiração do lease em `Complete`/`Fail`; regressões cobrem lease expirado e attempts negativos. Notificação push segura o lock de estado compartilhado desde a checagem final até iniciar/concluir a requisição; revogação que vence antes do envio impede a requisição (uma requisição já iniciada não pode ser retraída). (8) CAS do JSONStore consulta versão durável sob lock interprocesso antes de comparar e atualizar cache; SecretStore recarrega, aplica read-modify-write e persiste sob lock interprocesso, sem alterar cache em caso de falha.

**Evidência nova:** passaram os testes focados de todos os defeitos acima em `go test ./internal/agent -run '<regressões>' -count=1` (`ok`, 0.129s) e teste de rota `go test ./server -run '^TestArtifactRouteRemovesVerifiedTemporarySnapshotAfterDelivery$' -count=1` (`ok`, 0.034s). `gofmt` e `git diff --check` passaram. Um run inicial mais amplo falhou em fixtures, os fixtures foram corrigidos sem afrouxar assertions e os testes direcionados passaram; a suíte completa ainda não foi repetida.

**Build MLX:** o log anterior registrava `CGO_ENABLED=0 go build` e símbolos MLX/xgrammar/tree-sitter ausentes por constraints; o checkpoint anterior documenta gate completo aprovado com build CGO habilitado. Portanto isso não está classificado como defeito de código até repetir o build suportado (`CGO_ENABLED=1`).

**Bloqueador de release preservado:** RLS PostgreSQL continua spoofable via GUC caller-settable e o Runtime worker base tem autoridade sobre todas as organizações. Também faltam desenho/provisionamento de roles Runtime/migrator/dispatcher/executor segregados, contexto tenant não-forjável e testes adversariais. Não declarar paridade completa, isolamento PostgreSQL enterprise ou missão concluída e não publicar branch até resolução. Limites de equivalência Windows/macOS para snapshot continuam documentados; esta rodada reduz risco de PATH Windows, mas não é smoke nativo.

**Próxima ação concreta:** executar gates completos atuais (`go test -p=2 ./... -count=1`, `go vet -p=2 ./...`, `go test -race -p=2 ./internal/agent -count=1`, `CGO_ENABLED=1 go build -p=2 ./...`, cross-compile Windows AMD64, secret scan, diff review); fazer auditoria final independente; atualizar decisão de release sem ocultar o blocker RLS. Nenhum commit/push feito.

STATUS: CONTINUE — correções focadas passaram; gates completos e revisão final pendentes, RLS ainda bloqueia release.


**Atualização de verificação — 2026-09-27 01:18 -03:** `gitleaks detect --no-git --redact` executado em cópia isolada dos arquivos de texto staged/untracked/modified: 18 achados em 9 arquivos `*_test.go`, zero achados em arquivos não-test; nenhum valor foi impresso. Os focados do ciclo, inclusive revogação entre listagem/envio e rota de artefato, estão verdes. Gates completos (job `job_Y2nV94Hu`) e workflow com três revisores independentes iniciados; resultados ainda pendentes. Nenhum commit/push.


**Gates completos após todas as correções — 2026-09-27 01:19 -03:** job `job_Y2nV94Hu` encerrou `ALL_GATES_PASS`. Evidência no log `/tmp/ollama-full-release-gates-post-remediation-20260927.log`: `go test -p=2 ./... -count=1` PASS; `go vet -p=2 ./...` PASS; `go test -race -p=2 ./internal/agent -count=1` PASS; `CGO_ENABLED=1 go build -p=2 ./...` PASS; cross-compile `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c ... ./internal/agent` PASS; `git diff --check` PASS. Isso confirma que os símbolos MLX ausentes anteriores foram efeito do build `CGO_ENABLED=0` aplicado à árvore inteira, não falha do build nativo suportado. Cross-compile valida compilabilidade do pacote agent no Windows, não execução nativa nem paridade do snapshot.

**Scanner:** gitleaks com redaction em corpus isolado de 18 hits/9 arquivos de teste; zero em arquivos não-test. Revisão independente em três áreas via workflow continua em execução; aguardar seus resultados antes da auditoria final/conclusão. RLS caller-settable e dispatcher/worker com privilégio all-tenant seguem bloqueadores do release PostgreSQL/enterprise; nenhuma publicação/commit foi realizada.

**Fechamento adicional — 2026-09-27 01:23 -03:** `go test -race -p=2 ./server -count=1` PASS (11.496s), cobrindo inclusive rota de artefatos. O integrity guard tinha uma string de diagnóstico obsoleta (`connected to a private address`) após sanitização do erro SSRF; o contrato correto atual é `resolves to a private address`. Atualizei somente essa assertion, sem enfraquecer o controle, e `bash scripts/check-class-a-plus-integrity.sh` passou. `git diff --check` passou depois da correção. Repetindo gitleaks no estado atual e aguardando a revisão independente. Ainda sem commit/push; PostgreSQL RLS continua bloqueado por desenho.


### Retomada pós-compactação — owner de jobs e contenção PostgreSQL — 2026-09-27 01:46 -03

**Correções adicionais após a auditoria independente anterior:** (1) `QueueJob` persiste `OrganizationID`; `Runtime.EnqueueMission` usa a organização lida do mission persistido; workers rejeitam job ownerless/foreign e executam missões somente via Runtime/store scoped. (2) Enqueue local e Redis migram job ativo legado sem owner apenas a partir do enqueue confiável, preservam idempotência para o mesmo tenant e rejeitam dedupe cross-tenant; Redis valida membership antes de persistir o owner. (3) Replay local/Redis recebe a organização derivada do mission persistido, vincula ownerless dead letters antigos e rejeita owner mismatch antes de reativar; o worker volta a verificar owner e mission. (4) `newDefaultAgentRuntime` agora recusa `OLLAMA_AGENT_DATABASE_URL` e `newAgentAPI` recusa PostgresStore até arquitetura de tenant não-forjável, independentemente de auth; `RequiresOrganizationAuthentication` identifica Postgres também sob `organizationScopedStore`. Isso é contenção operacional, **não** resolução do spoofing RLS nem dos papéis migrator/runtime.

**Regressões:** passaram testes locais de enqueue, owner binding, replay e worker (`go test ./internal/agent -run 'Test(JobQueueEnqueueBindsOrganizationOwner|JobQueueReplayBindsAndEnforcesOrganizationOwner|RuntimeQueueJobsOrganizationScope|RuntimeQueueWorkerRejectsOwnerlessAndForeignJobs)' -count=1`); passaram testes server de refusal para Postgres base e wrapped/authenticated (`go test ./server -run ...`). Novo teste Redis real, executado contra processo loopback descartável criado e limpo nesta sessão, passou: `go test -tags integration ./internal/agent -run '^TestDistributedRedisQueueOrganizationBindingAndReplay$' -count=1`. A primeira execução desse teste falhou somente porque a fixture ainda tinha o job de legacy binding pendente; o fixture foi corrigido para claim/ack antes do segundo caso, que então passou. Redis/Postgres integration URLs não estão configuradas no ambiente atual; o teste Redis usou processo descartável local. `gofmt` e `git diff --check` passaram.

**Gates:** a sequência completa anterior (`job_UqlZIAdv`) passou `go test -p=2 ./...`, `go vet`, race em agent+server, build nativo CGO, cross-compile Windows agent, integrity guard e diff-check, porém terminou antes da última regressão local de replay. Nova sequência congelada está em execução com secret scan redacted: `/tmp/ollama-full-release-gates-post-tenant-replay-20260927.log`; não inferir resultado antes do término. Revisão independente pós-fix em três áreas também está em execução (`57489b3574e0`).

**RLS e release:** a implementação PostgreSQL permanece arquitetura insegura para produção: GUCs caller-settable, DSN/migrações sem separação de papéis e store direto seguem no pacote; o servidor público agora falha fechado como mitigação e os workers locais/Redis têm owner explícito, mas não existe ainda credencial runtime/migrator segregada, contexto tenant criptograficamente/nativamente não-forjável, migração/backfill ou teste adversarial de spoofing. Sem Postgres integration DSN nesta sessão não é possível provar a nova arquitetura. Manter PostgreSQL bloqueado, não declarar paridade plena/isolamento enterprise, não criar commit/push/publicar este branch enquanto o gate e a revisão final não fecharem. Frontend/mobile não foram alterados neste ciclo; evidências anteriores permanecem referenciadas nos checkpoints preexistentes.

STATUS: CONTINUE — gates pós-replay e auditoria independente pendentes; PostgreSQL continua fail-closed e bloqueado para produção.


**Tightening final — runtime-level refusal — 2026-09-27 01:49 -03:** a review of embedding paths found that guarding only server construction was insufficient for callers that create `Runtime` directly. `NewRuntime` now rejects direct `PostgresStore` and the built-in `organizationScopedStore` wrapper before any worker can start; `newAgentAPI` remains a defensive guard and default server startup refuses `OLLAMA_AGENT_DATABASE_URL`. Tests cover direct and wrapped stores even when HTTP auth is set. Updated `SECURITY.md`, `docs/CLASS_A_PLUS_GUIDE.md`, `docs/agentic/INTEGRATIONS.md`, `docs/agentic/PHASE5_DELIVERY.md`, and `docs/agentic/PARITY_MATRIX.md` so no setup path recommends the unsafe public mode. Latest targeted agent/server tests passed. Final full gate/secret-scan job `job_VII1ASko` is running against this tree; final independent reviewers `57489b3574e0_a0..a2` are still running. Report results only after their terminal notifications.


### Reauditoria residual — correções implementadas — 2026-09-27 02:08 -03

**Contexto:** o relatório independente `audit/FINAL_SECURITY_REAUDIT_POSTFIX_20260927.md` reproduziu seis falhas de segurança/robustez, além de recomendar endurecer respostas de deployment. As correções deste ciclo são:

1. `terminal.exec` resolve executáveis por caminho absoluto dentro de diretórios fixos confiáveis antes de `exec.Command`; teste cria um `git` falso no PATH herdado e confirma ausência de execução do marcador.
2. Catálogos MCP autenticados exigem ownership exato; conectores/MCP/deployments devolvem só origem (sem path/query), omitem token/header env mappings onde aplicável. Ownerless/global não aparece em listagens de organização. Teste de rota valida isolamento e ausência de valores sensíveis.
3. DLP recursivo agora trata `signature`, `sig`, `_signature`, `_sig` e aliases normalizados como `x.sig`/`providerSignature`; teste confirma bloqueio outbound, redaction de mapas e JSON incorporado; teste Remote MCP valida o caminho real de resposta 2xx.
4. Outbox push: no máximo 8 tentativas; 256 registros totais/64 por organização; payload integral por item ≤64 KiB; snapshot persistido ≤20 MiB com checagem antes de ler/escrever; terminal retries são compactados após 7 dias; flush limita 32 entregas e 30 s, respeitando cancelamento. Testes cobrem quotas, tamanho, compactação e batches. A persistência continua snapshot JSON atômico, mas agora com limites rígidos de custo/tamanho.
5. `QueueJobs`/`QueueJobsForOrganization` só incluem registros em que `QueueJob.OrganizationID == mission.OrganizationID`; teste de adulteração de metadata comprova que a listagem não vaza job inconsistente.
6. Respostas de deployment usam leitura `max+1`, rejeitam resposta >4 MiB, JSON malformado e payload não-objeto; deployment genérico rejeita sucesso sem ID/URL. Regressões cobrem 2xx truncado, array, excesso de tamanho e identidade ausente.

**Evidências já aprovadas neste ciclo:** gofmt; testes focalizados agent e server (ambos PASS); teste `TestDistributedRedisQueueOrganizationBindingAndReplay` com Redis loopback descartável real (PASS; servidor/dados temporários limpos). O repositório segue na branch `recovery/ollama-full-snapshot`, HEAD `8635e30d`, 113 paths modificados, remote origin autorizado presente; nenhum commit/push foi feito.

**Gates finais:** job `job_wYVcAWwc` em execução para `go test ./...`, `go vet ./...`, race agent/server, build nativo, cross-build Windows, integrity guard, `git diff --check` e secret scan redacted. Registrar resultados somente após o job terminar. PostgreSQL runtime público continua recusado pelo `NewRuntime`/API como contenção: GUC RLS spoofável não foi corrigido; PostgreSQL integration DSN e PostgreSQL local não disponíveis. Nenhuma alegação de paridade completa com Manus Desktop: o escopo frontend/mobile e várias capacidades funcionais ainda não foram verificados/concluídos neste ciclo. Não publicar/push enquanto gates ou bloqueador de isolamento enterprise permanecerem.

STATUS: CONTINUE — aguardar evidência final do gate atual; revisar auditoria independente pós-fix; RLS PostgreSQL e paridade global continuam bloqueadores.


### Terceira rodada de refinamento de segurança — 2026-09-27 02:43 -03

**Correções nesta rodada:** missões não isoladas agora persistem identidade de diretório e reabrem workspace por `os.Root`, rejeitando identidade ausente (legado) ou troca do pathname. O inspetor Git aceita tanto raízes normais descriptor-bound como snapshots isolados com manifesto, valida `.git` sem symlinks/arquivos especiais e rejeita `alternates`, `http-alternates` e `commondir` que poderiam redirecionar leitura de objetos para fora do repo. `terminal.exec` continua bloqueando `git` e exige raiz pinada/Linux. Store local de filas agora verifica snapshot <=20 MiB, <=100k registros, retry metadata/status/lease antes de carregar e ao persistir. Push outbox valida global/per-tenant quotas na leitura e escrita e não remove retry terminal durante lease ativo. Remote MCP SSE mede total de bytes, incluindo eventos não correlacionados, com cap acumulado de 4 MiB.

**Regressões focadas PASS:** replays ownerless negados sem mutação; retry metadata inválida negada no startup; identidade de workspace trocada/ausente negada; root normal descriptor-bound continua permitindo Git; Git rejeita symlink de object store e alternates externos; lease ativo preservado pela compactação; stream SSE >4 MiB antes da resposta correlacionada rejeitado. `go test -p=2 ./internal/agent ./server -count=1` passou antes do último teste de alternates; após isso os testes adversariais focados passaram.

**Gate completo, resultado parcial (não final):** a primeira tentativa falhou num teste antigo que ainda esperava rejeição de raiz normal pinada; o teste foi corrigido para refletir a nova distinção. A segunda execução passou `go test -p=2 ./...`, `go vet -p=2 ./internal/agent ./server`, `go test -race -p=2 ./internal/agent ./server -count=1` e `CGO_ENABLED=1 go build -p=2 ./...`. O passo de cross-build de `./internal/agent ./server` no Windows falhou porque a árvore upstream `mlx`/`mlxrunner/xgrammar` exclui todos os arquivos sob GOOS=windows; os diagnósticos foram `build constraints exclude all Go files` e símbolos MLX ausentes, não erro introduzido por estes fixes. O escopo cross-platform suportado anteriormente foi cross-compile de `./internal/agent` isoladamente; ele será repetido junto aos gates finais. Como a última mudança de Git metadata entrou durante aquela execução, os gates devem ser repetidos sobre a árvore congelada.

**Estado externo:** reviewers independentes da terceira rodada (workflow `bdf4d02c12b2`, áreas workspace/Git, queue/outbox e DLP/providers) ainda executam. O blocker PostgreSQL RLS GUC-spoofable continua fail-closed, sem solução arquitetural nem teste adversarial com PostgreSQL disponível. Branch `recovery/ollama-full-snapshot`, HEAD `8635e30d`, remote `origin` configurado; nenhum commit ou push. Não declarar paridade total nem publicar enquanto auditoria/gates estiverem pendentes e o blocker de isolamento persistir.

STATUS: CONTINUE — repetir gates em árvore congelada e aguardar reauditoria independente; PostgreSQL enterprise continua bloqueado.


### Retomada final — egress e entrega push — 2026-09-27 03:55 -03

A sequência pós-workflow encontrou dois defeitos de integração nos testes: a política comum de IP bloqueava IPv4 público recebido como `net.IP` de 16 bytes (corrigido por canonicalização IPv4/mapped antes da classificação; ranges especiais continuam bloqueados), e o teste da rota tentava registrar endpoints com query de credencial, agora corretamente rejeitados. As suítes completas `go test -p=2 ./internal/agent -count=1` e `go test -p=2 ./server -count=1` passaram após a correção; teste focused também PASS para endpoint com assinatura rejeitado, lock push liberado durante HTTP bloqueado, evento de missão com falha de outbox reportada por métrica/log sem inverter transição persistida, recuperação de panic de handler e fencing de revogação.

`PushService.NotifyOrganization` agora faz leitura/validação e prepara o payload sob o lock de estado, libera locks em-processo e cross-processo e só então chama o provider. Uma revogação depois da validação final ocorre durante entrega já autorizada/em andamento; ela não bloqueia atrás do request de rede. Falha de enqueue no push outbox continua sem desfazer transição de missão já persistida e agora gera log explícito, além de `PushOutboxFailures` (regressão cobre ambos). Queue handler panic é convertido em `ErrQueueNonRetryable` e não derruba o processo.

O gate definitivo da árvore congelada é `job_8LuKAPGl`, log `/tmp/ollama-full-final-gates-post-lock-20260927.log`, cobrindo `go test ./...`, vet, race de agent/server, build nativo, cross-compile Windows do pacote agent, integrity guard, diff e gitleaks em arquivos modificados com saída redacted/agregada. Não afirmar resultado antes da conclusão. Os agentes de correção `0597198596e1` reportaram trabalho parcial; as mudanças foram inspecionadas e os defeitos de escopo MCP/DLP/IP completados localmente, mas Redis quotas/índice persistente e reconciliation de criação de missão ainda exigem análise adicional. PostgreSQL RLS baseada em GUC segue fail-closed e sem arquitetura não-forjável; nenhum commit/push/publicação foi feito.

STATUS: CONTINUE — aguardar `job_8LuKAPGl`, depois anotar gates comprovados e distinguir claramente escopo de segurança validado do produto/paridade ainda incompletos.


### Quarta rodada — achados reabertos pela auditoria final — 2026-09-27 04:16 -03

O gate congelado anterior ao novo workflow concluiu com exit 0 em `go test -p=2 ./... -count=1`, `go vet -p=2 ./...`, race de `internal/agent`+`server`, build nativo `./...`, cross-compile Windows do pacote agent, integrity guard, `git diff --check` e gitleaks redacted em 121 arquivos texto alterados. A auditoria final read-only, porém, reproduziu blockers novos; verde de testes não equivale a release seguro. Três frentes independentes do workflow estão ativas, chamadas efetivas 3 de 5, compartilhando checkout sem permissão para commit/push:

1. **Tenant/schedules/local mode:** worker `resumePending` pode consumir e atualizar schedules de outra organização; local mode sem auth pode acessar/mutar registros tenant-bound e aceitar organization_id escolhido pelo cliente. Correção exigida: claim/update owner-aware atômico; namespace local reservado e isolamento de leitura/mutação.
2. **Workspace/Git/processo/media/artifacts:** launchers `unshare`/`setpriv` e OCR usam PATH ambient; Git usa fallback path-based em não-Linux e permite include config externo; `ls` e media têm possíveis check/use por pathname; artifacts são reabertos por path após hash e stale temps podem sobreviver a crash. Correção exige handle binding ou fail-closed (sem alegar cobertura multiplataforma inexistente).
3. **Egress/credenciais/bounds:** push usa classificador IP mais fraco; `connector requestPath` permite queries com credenciais; deploy/Remote MCP prosseguem sem envs de credencial configuradas; push endpoint aceita query/fragment potencialmente secretos; push subscriptions, eventos JSONStore e traces não têm quotas completas em records/bytes.

Os achados estão preservados nos resultados do workflow de auditoria final `6ba3dfb9e7e1`; as recomendações e evidências reproduzíveis foram entregues no contexto. PostgreSQL RLS/GUC continua bloqueio crítico de produção e fail-closed; paridade Manus Desktop continua parcial segundo `docs/agentic/PARITY_MATRIX.md`. Nenhum commit/push foi feito. A próxima ação é integrar/verificar os três resultados, corrigir testes/falhas, rodar gates de novo e repetir revisão independente. STATUS: CONTINUE.


**Retomada após correções parciais — 2026-09-27 04:46 -03:** o segundo workflow também foi interrompido antes de fechar suas unidades. Corrigi a regressão objetiva `runtime.go:974: no new variables on left side of :=` (a variável `organizationScope` já existia); `gofmt` foi executado em todos os Go modificados/novos e compilação sem testes passou: `go test -p=2 ./internal/agent ./server -run '^$'`. As mudanças ainda não estão completas nem validadas semanticamente. O auditor tenant não adicionou testes nem fechou lifecycle/catalog/local route; o auditor filesystem não integrou media/OCR/artifact descriptor handoff integral; o auditor egress não fechou/validou todos os testes e foi observada a fixture legacy de push com formato incorreto. Há mudanças no checkout compartilhado em 125 caminhos, branch `recovery/ollama-full-snapshot`, HEAD `8635e30d`, remote `origin`, sem commit/push. `go test -p=2 ./internal/agent ./server -count=1` foi iniciado em `/tmp/ollama-full-agent-server-post-fourth-wave.log`; STATUS: CONTINUE.


### Quinta rodada — media pinning, bounded persistence e sandbox approval — 2026-09-27 05:08 -03

**Remediações integradas nesta rodada:**

- Media process exige `ToolContext.WorkspaceRoot` e encaminha o descriptor a leituras aprovadas, geração/materialização de arquivos, OCR e operações HTTP relacionadas; resultados usam caminhos relativos ao workspace fixado. Regressões adversariais cobrem substituição de pathname do workspace antes de leitura/escrita.
- JSONStore mantém limites existentes por arquivo/missão (10.000 eventos, 20 MiB, payload até 1 MiB) e agora limita também agregado (10.000 event files, 100.000 eventos, 256 MiB), verificando reload e append sob lock compartilhado; store em memória aplica as mesmas quotas agregadas.
- TraceStore limita 10.000 spans, 64 KiB por span e 20 MiB serializados, rejeita crescimento acima dos limites e devolve cópias profundas de atributos.
- Push subscriptions mantêm cap global existente e passam a impor até 1.000 por organização e 2 MiB de representação por organização, além de limite por registro/arquivo; ids de usuário/tenant também são limitados.
- `sandbox.exec` agora grava o modo autoritativo (`best-effort`/`strict`) no payload aprovado, ignora tentativa do planner de escolher o modo, exige correspondência com a configuração no instante da execução e falha com `ErrApprovalPayloadChanged` se houver drift. Tool descriptor version foi elevada para 4 para invalidar compatibilidade implícita de approvals antigos.
- Inspetor Git rejeita configurações locais `include`/`includeIf`; ambiente de processo já desativa config global/sistema e Git não-Linux falha fechado. Regressões cobrem includes diretos/condicionais e comentário inofensivo.

**Evidência:** focused regressions PASS; `go test -p=2 ./internal/agent ./server -count=1` PASS (agent 11.795s; server 3.272s). Gates completos iniciados no job `job_97qBxyy5`, log `/tmp/ollama-full-third-wave-release-gates-20260927.log`: `go test ./...`, vet, race agent/server, build CGO nativo, cross-compile Windows agent, integrity guard e `git diff --check`; ainda sem resultado final. Dois reviewers independentes read-only estão ativos (`job_EuQBOuY1` para quotas; `job_elvSPYar` para Git/sandbox).

**Release/authority:** branch local `recovery/ollama-full-snapshot`, remote `origin` é o repo autorizado de projeto; checkout contém numerosas mudanças anteriores não commitadas. Nenhum commit/push foi feito. PostgreSQL RLS caller-settable por GUC continua bloqueador real de produção e fail-closed; não declarar release production-ready, isolamento PostgreSQL resolvido ou paridade completa Manus Desktop. Aguardar jobs, corrigir achados e repetir gates após alterações.

STATUS: CONTINUE — aguardar gates e reviews, corrigir eventuais findings e manter PostgreSQL bloqueado.


### Sexta rodada — findings críticos da revisão independente — 2026-09-27 05:39 -03

A revisão independente revelou que a quarta/quinta rodada ainda não era suficiente; findings adicionais foram corrigidos no checkout:

- **Sandbox:** fecha FD 3 imediatamente após o bind mount, executa o processo dentro de chroot montado com `/usr`, bibliotecas e configuração somente leitura, workspace fixado em `/workspace`, `/tmp` privado e sem procfs; qualquer helper/execução host no Windows/macOS foi removido em favor de rejeição fail-closed. O teste adversarial confirma que tanto `/proc/self/fd/3/../...` como caminho absoluto de arquivo irmão não revelam segredo e que path substitution do workspace ainda usa o descriptor original.
- **Approval:** validação exige exatamente um approval por cada step que requer approval; IDs únicos; mission/organization/step corretos; nonce/hash não vazios; hash atual igual ao payload aprovado. Missões ativas corrompidas falham antes de execução na entrada de `Run`, na decisão e na recuperação.
- **Git:** config parser agora lida com comentário trailing em section header; rejeita `include`/`includeIf` e seções `filter` antes de qualquer comando Git; o mesmo guard é aplicado ao snapshot isolado. Teste inclui clean-filter com marker de execução para provar que subprocesso não dispara.
- **Persistência:** JSONStore limita missão individual a 8 MiB, plan 32 steps, 32 approvals, 512 artifacts, profundidade 64 e 100.000 nós; coleção persistida limita 10.000 missões/512 MiB e diretórios são enumerados com teto. Eventos evitam criar arquivo em leitura ausente, refrescam mission owner sob lock, preservam idempotência no limite e preflight payload antes de redaction/clone. TraceStore serializa read-modify-write por lock e mescla spans por ID; preflight rejeita estrutura cíclica/deep. Push quota mede mapa JSON indentado efetivamente persistido.

**Validação após estas mudanças:** suite focal de 12 famílias PASS (`go test ./internal/agent -run ...`, 1.610s); teste Linux real de isolamento/chroot também PASS isoladamente. Um gate completo ainda está rodando (`job_QUlaFbfp`), mas começou antes de todas as alterações finais e precisa ser considerado evidência provisória, não substitui novo gate congelado. Dois reviewers independentes estão reavaliando o estado atual (`job_4OG5KRii`, `job_FFwlZNDL`). Após fixar possíveis findings/revisões, executar novamente testes totais, vet, race, build CGO, cross-compile Windows, integrity guard e `git diff --check`.

O bloqueador de produção PostgreSQL RLS/GUC permanece sem solução e continua impedindo declarar release production-ready ou paridade total; ainda não houve commit nem push.


### Backup e continuação — 2026-09-27 07:10 -03

O usuário pediu explicitamente preservar no GitHub o projeto modificado, pois vai formatar o PC. Checkout em `/home/ubuntu/ollama-full-recovery`, branch `recovery/ollama-full-snapshot`, base `8635e30dc9e95a1f5b29700169783abc24093ceb` (também `origin/main`). O remoto é o repositório público `DZ23-LTDA/hades-ollama`. O commit de snapshot aprovado `8c2e3604a086abeab401e2c80e96fe6ad4473566` contém 134 arquivos (22.536 inserções, 1.582 remoções) e foi publicado na branch; `main` permanece no hash base.

Nesta retomada, foi fechado o race reportado entre validação Git e execução de subprocessos: comandos Git de inspector/snapshot recebem um `GIT_DIR` temporário privado, descriptor-bound, com cópia limitada de HEAD/index/objects/refs e sem `config`, `config.worktree`, hooks ou filtros. O `GIT_WORK_TREE` usa o descriptor do workspace. A configuração e árvore `.git` original são revalidadas após cada subprocesso e troca insegura faz falhar fechado. Regressões injetam configuração clean-filter antes de `diff` e substituem `.git/HEAD` entre comandos.

**Evidência local desta mudança:** `go test -p=1 ./internal/agent -run 'TestGitRepo|TestGitReaders|TestCreateWorkspaceSnapshot(GitUsesPinnedSourceAfterPathReplacement|RejectsGitMetadataReplacementBetweenCommands|CopiesSafeWorkingTreeWithoutMutatingSource)' -count=1` PASS; `go test -p=2 ./internal/agent -run 'Test(GitRepo|GitReaders|.*WorkspaceSnapshot)' -count=1` PASS. `git diff --check` PASS. Gitleaks redacted em todos os arquivos modificados/novos encontrou 18 alertas, todos em arquivos `_test.go` com fixtures de segredo sintético; nenhum alerta fora de testes. Os detalhes/valores não foram impressos.

Captura real da interface atual foi salva em `docs/images/screens/ollama-full-current-ui-2026-09-27.png` e em `/home/ubuntu/ollama-full-current-ui.png`. Browser limitado a localhost; backend foi executado com `OLLAMA_NO_CLOUD=1`, sem chamadas externas na captura final. Serviços temporários foram desligados. O início inicial, antes de o modo sem nuvem ser aplicado, tentou hidratar catálogo remoto; foi encerrado imediatamente e não houve upload de dados do projeto.

A árvore congelada passou: `go test -p=2 ./... -count=1`, `go vet -p=2 ./...`, `go test -race -p=2 ./internal/agent ./server -count=1`, `CGO_ENABLED=1 go build -p=2 ./...`, cross-compile Windows de `internal/agent`, guard de integridade, gofmt, diff check e gitleaks redacted (somente fixtures sintéticas nos arquivos de teste). O commit foi publicado sem force-push numa branch nova; verificação pela CLI e pela API GitHub confirmou seu hash e confirmou `main` intacta. PostgreSQL RLS caller-settable continua bloqueador de produção; não declarar release production-ready nem paridade Manus total.

STATUS: BACKUP REMOTO PRONTO — branch `recovery/ollama-full-snapshot`; `main` permanece intacta.


### Handoff de retomada — 2026-09-27 07:25 -03

Foi criado `audit/OLLAMA_FULL_HANDOFF_20260927.md` para continuidade após encerramento/formatação do PC. Ele reúne branch/remoto/commits, comandos de restauração, documentos-fonte, evidências de gates, remediações integradas, limites de paridade, findings remediados ainda aguardando validação independente, PostgreSQL fail-closed, próximos passos priorizados e regras de Git/publicação. Ao retomar, ler primeiro o handoff e depois comparar `git status`, `HEAD` e `origin` no GitHub; não confiar em hashes do texto sem revalidar, porque novos commits podem avançar a branch.

Estado verificado antes deste commit documental: `main`=`8635e30dc9e95a1f5b29700169783abc24093ceb`, recovery=`2ac42cc24f64546c87d7296f919b1519de0c67b5`, worktree limpo. O usuário pediu manter as informações necessárias para continuar e preservar tudo na branch pública `recovery/ollama-full-snapshot`; o conteúdo atual deste handoff e do checkpoint será enviado nessa mesma branch após `git diff --check`. Sem alteração/force-push de `main`.

STATUS: CONTINUE — retomar do handoff; P0 arquitetura PostgreSQL RLS; P1 reauditoria/gates pós-fix; seguir matriz para paridade funcional.


### Passagem de bastão Claude Code / Codex — 2026-09-27 07:29 -03

A pedido do usuário, criado `audit/CLAUDE_CODEX_RESUME_PROMPT_20260927.md`: prompt copiável que instrui agente novo a recuperar estado Git real, ler documentação de continuidade e instruções do repo, reauditar os blockers por prioridade, manter PostgreSQL fail-closed, executar regressões/gates, preservar alterações e limitar push à branch de recuperação. `AGENTS.md` (compartilhado com Codex) e `CLAUDE.md` agora apontam para esse prompt e para o handoff. `audit/OLLAMA_FULL_HANDOFF_20260927.md` lista o novo prompt e foi corrigido para distinguir o commit fixo do snapshot da ponta móvel da branch.

Estado antes desta publicação documental: branch `recovery/ollama-full-snapshot`, HEAD e remote `ed0c7ccf858379c18b9b4d02913ff52e7d6c6fb7`; `main` verificada em `8635e30dc9e95a1f5b29700169783abc24093ceb`; worktree estava limpo antes das alterações atuais. Esta rodada alterou somente Markdown/instruções de agente; nenhum código Go foi mudado, portanto os gates de código anteriores permanecem associados ao snapshot de código, sem alegar que foram executados novamente nesta rodada. Rodar `git diff --check` e integrity guard antes de commit/push; publicar somente na branch já autorizada.

STATUS: CONTINUE — ler prompt de retomada, validar branch/remoto e continuar P0 PostgreSQL RLS; P1 reauditoria pós-fix; depois paridade baseada na matriz.

### Retomada de engenharia — 2026-09-29 00:33 -03

**Estado verificado:** checkout `/home/ubuntu/ollama-full-recovery`, branch `recovery/ollama-full-snapshot`, HEAD e `origin/recovery/ollama-full-snapshot`=`d3a11965d57db28c5e11567d8d4a0a29f5aabd03`; worktree limpo no início da retomada. O PC reportado aparece como host Windows online, mas sem workspace exposto/selecionável; não se presume acesso aos discos dele. No sandbox, PostgreSQL 16 está instalado; foi criado apenas um cluster temporário local em `/tmp/ollama-full-pg-test-20260929`, loopback porta 55432, com DB/role descartáveis `ollama_agent_test`; nenhum dado de produção foi usado.

**Evidência inicial:** `OLLAMA_AGENT_TEST_POSTGRES_URL` do ambiente estava unset; a integração `go test -p=1 -tags integration ./internal/agent -run '^TestDistributedPostgresRLSAndEvents$' -count=1 -v` PASS com `ollama_agent_test` (`NOSUPERUSER NOBYPASSRLS`). Isto comprova compatibilidade básica do adapter, **não** isolamento contra GUC caller-settable: o desenho atual de policies ainda confia em `app.current_organization_id` e `app.system_access`, e `OpenPostgresStore` ainda mistura conexão e migração. `NewRuntime` e o startup público mantêm o bloqueio fail-closed.

**Plano P0 (em execução; hipótese a testar):** separar DSN/runtime de DSN/migrator e tirar migração do caminho de startup; provisionar owner/migrator sem login operacional da aplicação e role runtime não-super/BYPASSRLS, sem ownership/DDL; substituir GUCs como autoridade por contexto HMAC assinado pela aplicação e validado por função `SECURITY DEFINER` de search_path fixo cujo segredo esteja inacessível à role runtime; políticas RLS derivam organização e acesso sistêmico somente do contexto validado. O runtime só será reabilitado após teste PostgreSQL real que, sob role runtime, tente forjar tanto outro tenant quanto system-access via SQL e falhe, junto de grants/role audit e revisão independente. Threat model deve declarar que isto protege contra uso arbitrário das credenciais SQL isoladas, não contra comprometimento total do processo app/migrator.

**Reauditoria P1:** workflow `08c31477da9b` concluiu sete revisões somente leitura. Reabertos: executable trusted path/ancestry; catálogos locais usando listas globais; URL assinada de direct upload em erros/logs e sanitização parcial de deployment URL; JSON percentual com alias de assinatura na query Remote MCP; TOCTOU de leitura e lock sem contexto no outbox; URLs/IDs deployment sem formato estrito. Queue owner binding foi considerado sem achado explorável nos call sites atuais. Nenhum item reaberto está corrigido ainda; ver resultados workflow na tarefa para evidência completa.

**Recursos:** serviço PostgreSQL descartável `job_QRkOW9v5` está ativo nesta sessão para testes e deve ser encerrado/limpo ao finalizar; diretório é temporário. O host remoto não expõe workspace nesta sessão. Nenhum commit/push desta retomada ocorreu.

STATUS: CONTINUE — completar P0 e findings P1; manter Postgres público fail-closed.


### Continuação de segurança — 2026-09-29 00:58 -03

**P1 remediados nesta fatia:**
- Catálogos locais sem autenticação agora consultam projeções filtradas por `LocalOrganizationID`, em vez de serializar listas globais de conectores/MCP; teste confirma que recursos e segredos de tenants não vazam.
- Validação de endpoint/Remote MCP rejeita query JSON com chave de assinatura/token após percent-decoding, além de query malformada/ambígua e URLs fora do limite.
- Upload direto não propaga erros de transporte nem corpos de erro do CDN contendo URLs assinadas; deployment agora valida IDs e restringe/sanitiza URLs retornadas.
- Busca de Git/interpreters só aceita diretórios/arquivos de sistema com ancestry root-owned e não gravável por grupo/outros; links executáveis são aceitos apenas se o target resolvido também estiver em árvore confiável. O primeiro teste completo revelou que Python é symlink no sandbox; regra foi corrigida e os três testes do sandbox passaram depois.
- Leitura do snapshot do push outbox valida o mesmo inode antes/depois de abrir, rejeita symlink e limita a leitura pelo descritor; claim/fail/complete usam locks com contexto cancelável em Unix e Windows.

**Validação executada após correções:** `go test -p=2 ./internal/agent ./server ./x/transfer -count=1` PASS; Windows amd64 test-binary cross-compile de `internal/agent` PASS. A execução anterior dos mesmos pacotes falhou nos três testes de resolução de Python por symlink; corrigido e repetido com sucesso. Houve ainda log esperado de outbox rejeitando um snapshot inválido em teste. Executar gates de repositório completo (tests, vet, race, build, cross-compile, integrity, secret scan e diff) sobre a árvore congelada antes de commit/publicação.

**Atenção:** a tentativa incompleta de separar roles PostgreSQL foi revertida integralmente para o baseline fail-closed; nada em P0 foi implementado nesta rodada. O RLS continua confiando em GUCs caller-settable no adapter atual, então PostgreSQL deve continuar bloqueado para produção. O PostgreSQL temporário descartável para integração permanece apenas loopback em `/tmp/ollama-full-pg-test-20260929`, serviço `job_QRkOW9v5`, e deve ser parado/limpo ao concluir. Não declarar Ollama Full finalizado nem equivalente completo ao Manus: matriz de paridade continua sendo o plano para o trabalho funcional pendente.


### Continuação final desta rodada — 2026-09-29 01:38 -03

**Findings tardios da revisão independente corrigidos:**
- `requestDeploymentApproval` agora consulta providers apenas pela organização autenticada ou por `LocalOrganizationID` em modo local sem autenticação; o mesmo `organizationID` é usado ao criar a aprovação. Novo teste prova que provider de outro tenant retorna 404 e não cria aprovação local.
- `PushOutbox` agora adquire o lock cross-process antes do mutex em memória, para que um waiter de file lock não prenda o mutex necessário à limpeza. Falha e conclusão após entrega usam contexto de limpeza independente, limitado a 2 s. Regressões cobrem cancelamento, sucesso e a antiga ordem de deadlock.
- HEAD/init/direct/commit/PATCH/CDN/manifest upload erros não retornam transport errors com URL/body; erros de contexto `Canceled`/`DeadlineExceeded` continuam identificáveis. Novos testes cobrem URLs de HEAD/init com userinfo e cancelamento.

**Verificação congelada — PASS:** `gofmt`; `bash scripts/check-class-a-plus-integrity.sh`; `git diff --check`; `go test -p=2 ./... -count=1`; `go vet -p=2 ./...`; `go test -race -p=2 ./internal/agent ./server ./x/transfer -count=1`; `CGO_ENABLED=1 go build -p=2 ./...`; cross-compile do test binary Windows amd64 para `./internal/agent`. Log local: `/tmp/ollama-full-resumed-release-gates-final-r5-20260929.log`. Gitleaks redacted em fontes de produção alteradas: 0 findings; no conjunto incluindo testes houve uma detecção redacted no fixture sintético `internal/agent/deploy_test.go`, sem valor exibido.

**Revisão independente focada — sem findings acionáveis** sobre os quatro pontos acima (upload, outbox locks/cleanup, conclusão cancelada e approval local). Isto não equivale a uma nova auditoria integral de todas as superfícies nem remove o blocker PostgreSQL.

**Git:** checkout em `/home/ubuntu/ollama-full-recovery`, branch `recovery/ollama-full-snapshot`; o commit funcional `fdd9abfa30d583be9b69d76f6b3a133c492ab07d` foi enviado e verificado pela API GitHub e `git ls-remote`. Esta atualização final altera apenas documentação de retomada. A ref remota `main` continua em `8635e30dc9e95a1f5b29700169783abc24093ceb`; não alterada.

**Não finalizado:** P0 PostgreSQL RLS continua não implementado/forjável e public Postgres permanece fail-closed; falta ainda paridade funcional e validação E2E real conforme a matriz. Não afirmar produto 100% pronto ou equivalente integral ao Manus. O serviço PostgreSQL descartável usado no baseline foi encerrado e `/tmp/ollama-full-pg-test-20260929` removido. Próximo passo: retomar o projeto RLS com arquitetura migrator/runtime não-forjável e testes PostgreSQL adversariais, preservando os gates verdes desta rodada.


### Redesenho candidato do PostgreSQL RLS — 2026-09-29 02:40 -03

**Intenção:** remover o bloqueador P0 de RLS caller-settable no projeto Ollama Full, mantendo tenant isolation sob credenciais runtime, e preservar o produto fail-closed para quaisquer gaps de preparação/deploy.

**Arquitetura implementada no checkout local:**
- Contexto de tenant assinado com HMAC e validade curta (`app.tenant_context`); chave mantida em tabela sem SELECT pela role runtime, verificação no servidor em função `SECURITY DEFINER` com `search_path` fixo. GUCs antigos não são autoridade.
- Roles/DSNs separados: `ollama_agent_migrator` para migration DDL explicitamente via CLI; `ollama_agent_runtime` sem superuser, `BYPASSRLS`, ownership, role membership ou `CREATE`. `OpenPostgresStore`/startup não migram schema implicitamente.
- Bootstrap Compose cria as roles no volume novo. Para volumes antigos foi adicionado `deploy/postgres/migrate-existing-roles.sql`, com ownership transferido, senha/flags separados, grants legados revogados, sessões antigas terminadas e login antigo `ollama_agent` desativado. A documentação exige manutenção e backup validado.
- Policies/FK validam que evento pertence à mesma organização da missão. Migration rejeita missão/evento sem tenant válido, evento órfão/divergente, missão executável sem workspace identity e owners de schema/tabela incorretos; nenhuma atribuição/backfill é inferida.
- Recuperação Postgres percorre IDs de organização do AuthStore confiável e instancia runtime scoped sequencialmente; requer fila Redis owner-bound. `WorkspaceIdentity` agora é persistida em todas as rotas de insert/update/read do adapter.

**Falhas reais descobertas e corrigidas durante verificação:** relação evento→missão precisava verificar o tenant pai; recuperação Postgres não devia ser silenciosamente ignorada; volumes legados precisavam transferência explícita/aposentadoria do superuser antigo; adapter não persistia `WorkspaceIdentity`, quebrando recovery. Testes adversariais foram ampliados para todos esses casos.

**Evidência local (somente sandbox descartável, não produção):** PostgreSQL 16 e Redis 7 locais em loopback, credenciais runtime/migrator/admin sintéticas independentes. `go test -tags=integration ./internal/agent -run '^TestDistributed' -count=1` PASS; teste Postgres específico passou após upgrade de volume legado e após fixture fresh Compose. Cobertos cross-tenant read/write, evento→missão cruzado, GUC/HMAC inválidos, leitura da key, SET ROLE, DDL, `row_security=off`, key mismatch/imutabilidade, ownerless backfill, WorkspaceIdentity ausente, recovery por organização e role upgrade legado. Upgrade legado verificado: owners das tabelas = migrator, `ollama_agent` resultou NOLOGIN/não-superuser, zero sessões antigas ativas. Serviços PostgreSQL/Redis descartáveis foram parados após os testes.

**Gates no código final mais recente:** `go test -p=2 ./... -count=1`; `go vet -p=2 ./...`; `go test -race -p=2 ./internal/agent ./server -count=1`; `CGO_ENABLED=1 go build -p=2 ./...`; cross-compile Windows amd64; `bash scripts/check-class-a-plus-integrity.sh`; gofmt; `git diff --check`; parse Compose/CI YAML; gitleaks redacted em fontes de produção/deploy modificadas com zero achados — todos PASS. O Gitleaks não substitui controle de secret scanning do GitHub.

**Auditoria final:** três revisões read-only independentes estão ativas, com escopos separados (segurança, engenharia, operação/QA); seus vereditos ainda não foram recebidos. Não commitar nem publicar até reconciliar os achados e repetir os gates se houver qualquer patch.

**Git antes da publicação:** `/home/ubuntu/ollama-full-recovery`, branch `recovery/ollama-full-snapshot`; `HEAD` e upstream em `cdd91ceb22a3ca4637cc4e170e054c9750caa2d9` no início desta rodada. `main`=`8635e30dc9e95a1f5b29700169783abc24093ceb`. Mudanças desta rodada permanecem locais/não commitadas; autorização do usuário para manter o projeto nesta branch existe, sem merge/PR/main/force-push.

**Limites explícitos:** PostgreSQL continua candidato, não produção/isolamento enterprise validado, até staging TLS/HA, backup/restore comprovado, rotação coordenada de chave, crash/failover, avaliação de escala do sweep de recovery e auditorias concluídas. Paridade Manus Desktop e do conjunto de harnesses permanece incompleta conforme `docs/agentic/PARITY_MATRIX.md`; próximo objetivo depois desta liberação é fechar fluxos E2E prioritários, não declarar o produto 100% concluído.

STATUS: FINAL_AUDIT — aguardar três revisões independentes; corrigir/retestar e publicar somente em `recovery/ollama-full-snapshot` após verificação.


## Estado atual autoritativo — sincronização incremental 2026-09-29 07:51 -03

- **Branch/remoto:** `recovery/ollama-full-snapshot` / `origin`; ref confirmada via `git ls-remote` em `8cd2a71c8377cac46b8dd29642fe3ab38619ebff`. Não houve alteração em `main`, rebase, force-push ou PR.
- **Commits incrementais publicados nesta retomada:** `4bcc6ef2ff33c9e9c0b1ed09de1f7d9f1040d023` (`feat(postgres): harden tenant-only RLS and recovery gates`), `00314480e6fbb21f3b99755a1db19e143e33690e` (`test(postgres): reject stale privileged legacy sessions`) e `8cd2a71c8377cac46b8dd29642fe3ab38619ebff` (`chore: remove generated Redis test dump`). Um dump RDB de 4 KiB produzido localmente pelo Redis de teste entrou acidentalmente no segundo push, foi identificado, escaneado sem findings e removido no commit imediatamente seguinte; está fora do repositório agora. Não publicar dumps/logs/segredos.
- **P0 implementado como candidato, não liberado:** contexto tenant HMAC por transação, roles separadas e least-privilege, migration CLI explícita, upgrade legado em duas fases, auditorias locais de eventos/workspace/recovery, propagação de cancelamento e guardas de startup. Readiness também recusa role privilegiada antiga ativa **ou sessão antiga remanescente mesmo após demotion**.
- **Evidência nova reproduzível nesta rodada:** teste real `TestDistributedPostgresRuntimeReadinessForStaleLegacySession` passou em PostgreSQL 16 depois de abrir sessão pela role superuser antiga e só então demover/desabilitar o login. `go test -tags integration ./internal/agent -run '^TestDistributed' -count=1` passou depois do cutover de teste com PostgreSQL/Redis loopback. Testes afetados `go test -p=2 ./internal/agent ./server ./cmd -count=1`, integrity, gofmt, parse de Compose/CI YAML e gitleaks redacted de arquivos alterados passaram antes do último teste de integração; repetir os gates finais sobre o HEAD atual antes de declarar esta rodada encerrada.
- **Cobertura de reviews:** as três revisões anteriores apontaram, entre outros, exposição de recursos no modo local, fechamento de Runtime não pertencente à chamada, perda de diagnósticos Redis, lacunas de testes da CLI e cutover. Foram implementados filtros locais/owner checks, teste de ownership de Runtime, observabilidade/backoff, testes do CLI migrator, checks de active-session e ensaio de backup/restore. A tentativa de reauditoria independente do delta exato mais recente terminou como job failed/sem veredito; não relatar aprovação de auditoria final. Reconciliar os achados contra o HEAD atual e obter uma revisão read-only antes de promoção.
- **Validação de plataforma:** cross-compile Windows de `cmd` foi tentado e falha pelas build constraints/arquivos MLX (`Array` indefinido e `mlxrunner/xgrammar` sem arquivos para esse target). Isso não foi ocultado: o gate suportado continua sendo cross-compile do pacote `internal/agent`, não prova de CLI Windows nativo. O workflow Windows existente constrói instalador, mas ainda não fornece smoke nativo completo do migrator.
- **Limites abertos:** PostgreSQL/RLS permanece candidato e bloqueado para produção até staging TLS/HA, backup/restore/cutover com rollback realmente exercitado, rotação HMAC transacional e testada (hoje não suportada), crash/failover, escala do recovery sweep e auditoria independente final. A descoberta de tenants via AuthStore local não habilita operação multi-instância. Paridade Manus/harnesses segue incompleta; ver matriz.
- **O que está pela metade:** gates completos após o último delta ainda precisam ser repetidos; verificar resultado do CI para o SHA publicado; revisão independente do delta mais recente não concluída; operação de rotação e cutover multi-instância/staging pendentes.
- **PRÓXIMA AÇÃO CONCRETA:** executar `go test -p=2 ./... -count=1`, `go vet -p=2 ./...`, `go test -race -p=2 ./internal/agent ./server -count=1`, `CGO_ENABLED=1 go build -p=2 ./...`, cross-compile Windows suportado do pacote agent, integrity, YAML, `git diff --check` e secret scan no HEAD `8cd2a71c`; depois revisar/pushar este checkpoint e continuar somente com findings concretos, sem declarar produção ou paridade completa.

STATUS: P0_CANDIDATE_PUSHED — branch de recuperação sincronizada; produção e conclusão global continuam bloqueadas pelos gates acima.


## Estado atual autoritativo — rodada RLS/cutover publicada 2026-09-29 09:07 -03

- **Último commit enviado:** `43daf0e22fd81bb8977c79abf94ca9aa6a25519c` (`fix(postgres): close RLS drift and legacy cutover gaps`) em `origin/recovery/ollama-full-snapshot`; ref local/remota conferida igual após o push. Nenhuma alteração em `main`, sem force-push, rebase ou PR.
- **Achados independentes fechados nesta rodada:** serialização advisory-lock entre transações de runtime e migrations; qualificações `pg_catalog`/`public` contra shadowing; verificação de privilégios extras incluindo TRUNCATE/REFERENCES/TRIGGER/TEMP; RLS drift checks com sentinel específico e cleanup; retirement cluster-wide incluindo sessões em outro database; preflight/CI de role contract e reconexão antes/depois de `NOLOGIN`; seleção determinística do check-run mais recente; sanitização de todos os DSNs/senhas sintéticos da CI; runbook com uma única ordem, DSNs definidos, senha do migrator via ambiente, e ensaio pg_dump/pg_restore; ownerless skills visíveis somente no escopo local.
- **Evidência real após as mudanças:** `CGO_ENABLED=0 go test -tags integration ./internal/agent -run '^TestDistributed' -count=1` passou contra PostgreSQL 16 e Redis 7 descartáveis; testes separados para sessão privilegiada legada ativa e sessão stale passaram. Uma sessão legada foi mantida no database `postgres`; o script encerrou essa conexão ao aposentar a role, e o estado final verificado foi NOLOGIN/NOSUPERUSER/NOCREATEDB/NOCREATEROLE/NOREPLICATION/NOINHERIT, sem memberships ou sessões remanescentes. Integridade do repo, gofmt, diff whitespace, YAML de Compose/CI, e Gitleaks redacted nos arquivos alterados passaram (0 findings).
- **O que ainda está pela metade:** ainda não foram repetidos neste commit os gates completos (`go test ./...`, vet, race, build, Windows agent cross-compile); GitHub Actions para `43daf0e` ainda precisa ser observado; precisa haver uma nova revisão read-only independente do delta exato após remediação. PostgreSQL/RLS continua candidato bloqueado para produção: staging real TLS/HA, rotação segura da chave HMAC, failover/escala, operação multi-instância e rollout/restore com evidência continuam pendentes. Paridade funcional com Manus Desktop/harnesses está incompleta; ver matriz.
- **PRÓXIMO PASSO CONCRETO:** executar no HEAD `43daf0e` os gates completos; verificar os checks GitHub do mesmo SHA; solicitar/reconciliar revisão independente curta do delta. Corrigir achados concretos e, após cada rodada verificada, atualizar estes dois checkpoints e fazer commit+push somente na branch recovery.

STATUS: P0_RLS_CANDIDATE — delta de segurança publicado e integração real passou; release global/produção/paridade completa permanecem bloqueados.


## Estado atual autoritativo — rodada de hardening do cutover PostgreSQL 2026-09-29 10:07 -03

- **Último commit enviado:** `0aaa65548d4e51632111fe23d95ea2cb1970a6ac` (`fix(postgres): close schema and cutover review findings`), confirmado por `git ls-remote` em `origin/recovery/ollama-full-snapshot`. Nenhuma mudança em `main`; sem force-push, rebase ou PR.
- **SQL de cutover endurecido:** ambos os scripts agora fixam `search_path = pg_catalog, public` antes de lógica SQL, qualificam funções de sistema, revogam memberships legadas em ambas as direções e verificam estado least-privilege/ownership residual. A preparação recusa objetos não suportados e ownership em outros databases; transfere relações, funções/procedures, tipos compostos/domínios/enums e large objects ao migrator. A aposentadoria confere dependências de ownership via `pg_shdepend` e drena sessões no cluster.
- **Schema e NULL:** o preflight do migrator valida colunas requeridas, tipo, nullability e defaults antes de DDL que depende de `id`; depois valida PKs, índice único `(id, organization_id)` e FK validada do evento à missão. As queries de backfill/ownership tratam `NULL` explicitamente e falham fechado.
- **Fixtures/CI/runbook:** a suíte registra IDs em `t.Cleanup`, confirma ausência de missões/eventos, usa namespaces Redis únicos e remove recursos persistentes; o CI adiciona `search_path` hostil, membership reverso e objetos legacy dos cinco tipos transferidos. O ensaio de restore no runbook compara `system_identifier`, exige cluster staging distinto, separa senhas de origem/staging e limpa database exclusivo com `trap`.
- **Evidência real local — PostgreSQL 16 + Redis loopback descartáveis:** phase 1 e phase 2 passaram com funções-sombra no `search_path`; a fase 2 confirmou zero membership nos dois sentidos. Objetos legacy composite/domain/enum/function/large-object ficaram todos sob `ollama_agent_migrator` (5/5). `CGO_ENABLED=0 go test -tags integration ./internal/agent -run '^TestDistributed' -count=1` PASS (4,03 s); o grupo incluiu RLS adversarial, drift, readiness, Redis, e os testes de drift de tipo/null/default. O subconjunto que exige admin/legacy DSNs também passou. Após teardown, contagem de fixtures de missão/evento = 0 e Redis keys no namespace de recovery = 0. PostgreSQL e Redis descartáveis foram parados; nenhuma credencial/dump foi adicionada ao Git.
- **Outros gates PASS neste delta:** `go test ./internal/agent -count=1`; `go vet ./internal/agent`; `gofmt`; `git diff --check`; `bash -n` no snippet de restore. `actionlint` não está instalado; o parser YAML Python também não está instalado, então o workflow YAML não foi validado por parser local nesta sessão.
- **O que permanece incompleto/bloqueado:** rotação HMAC versionada e transacional continua não implementada/suportada; staging TLS/HA e ensaio operacional real não ocorreram. Não foram repetidos full `go test ./...`, vet total, race, build completo, cross-build, checks Actions do SHA nem re-review independente do delta atual. PostgreSQL permanece candidato, nunca declarar production-ready; paridade total com Manus/harnesses segue incompleta.
- **PRÓXIMO PASSO CONCRETO:** desenhar o protocolo de rotação HMAC com quiescência, versionamento e rollback; implementar migration/CLI transacional e testes PostgreSQL adversariais de crash/rollback/concorrência. Depois executar gates globais e solicitar revisão read-only independente; registrar/pushar novo checkpoint a cada rodada verificada em `recovery/ollama-full-snapshot`.

STATUS: P0_RLS_CANDIDATE — rodada incremental publicada e validada localmente em PostgreSQL 16/Redis; rotação HMAC, staging, auditoria final, produção e paridade global permanecem bloqueadas.


## Estado atual autoritativo — rotação HMAC versionada e cutover final 2026-09-29 12:01 -03

- **Commit funcional enviado:** `1bd884144c173ba8d97b7cac36000db1e7fe767c` (`feat(postgres): implement versioned tenant key rotation`), `origin/recovery/ollama-full-snapshot`; `git ls-remote` confirmou a mesma SHA. Sem alteração em `main`, sem force-push/rebase/PR.
- **Key rotation implementada:** chave tenant com versão no transcript HMAC e keyring migrator-owned; bootstrap v1 fail-closed; import atômico da tabela singleton legada; recusa de keyring preexistente sem chave ativa; migration normal exige versão/segredo ativo exato. Nova CLI `ollama agent rotate-postgres-key` recebe DSNs/segredos por ambiente, verifica identidade admin/migrator/cluster/database, serializa rotações, coloca runtime `NOLOGIN`, aborta com sessões existentes, drena lock, atualiza versão monotônica em transação e só então reabilita LOGIN. Sem dual-key grace period; processos runtime devem reiniciar com a nova chave.
- **pgcrypto/fresh install:** bootstrap privilegiado instala `pgcrypto` antes da migration; cutover legacy transfere ownership via `REASSIGN OWNED` ao migrator.
- **Evidência real local:** cutover completo em cluster PostgreSQL 16 isolado, sob search_path externo hostil: fase 1, migration Go e fase 2 passaram. Resultado final observado: owner do `pgcrypto`=`ollama_agent_migrator`; role `ollama_agent`=`LOGIN false | SUPERUSER false | CREATEDB false | INHERIT false`; memberships residuais=0. A suíte adversarial PostgreSQL/Redis incluiu import/bootstrap/corrupção, timeout/fencing/sessão runtime, rotação concorrente, stale runtime, monotonicidade/high-water, índice ativo e cleanup; passou nas execuções locais reportadas antes deste gate global.
- **Gates globais concluídos:** `go test ./...` inicialmente expôs teste de servidor dependente do ambiente; o teste agora zera explicitamente `OLLAMA_AGENT_TENANT_CONTEXT_KEY` e o rerun integral PASS. `go vet ./...`, `gofmt`, `git diff --check` e `actionlint` no workflow alterado PASS. A checagem GitHub de check-runs para esta SHA retornou lista vazia; não há CI remoto observado nesta branch até o momento. Inspeção de diff não encontrou credenciais literais.
- **Revisão independente:** re-review read-only focada no delta de rotation/cutover continua em andamento; não alegar veredito limpo até recebê-lo e reconciliar findings.
- **Ainda incompleto/bloqueado:** operação real em staging TLS/HA, restore/rollback operacional, crash/failover, coordenação em fleet real, observabilidade/alertas e validação de segurança independente permanecem. Produto global e paridade Manus/harnesses não estão concluídos; PostgreSQL segue candidato, não produção pronta.
- **PRÓXIMO PASSO CONCRETO:** concluir e reconciliar o review independente; se houver finding, corrigir, repetir gates relevantes e publicar novo commit/checkpoint apenas em `recovery/ollama-full-snapshot`. Se limpo, registrar o veredito e buscar execução dos checks Actions desse SHA; manter explícito o bloqueio de produção até staging/restore/failover comprovados.

STATUS: HMAC_ROTATION_IMPLEMENTED_PUSHED — cutover real PG16 e gates Go locais PASS; review independente e evidência de staging ainda pendentes.


## Estado atual autoritativo — contenção de ACLs PUBLIC PostgreSQL — 2026-09-29 12:14 -03

- **Último commit funcional enviado:** `b7822ed569e0a32e3cabd1c7c5949512b70649bd` (`fix(postgres): close inherited public ACL privilege paths`) em `origin/recovery/ollama-full-snapshot`; local/remoto conferidos iguais. Commit de continuidade anterior: `88ebfc56adc4ec5017e66401e92d4f9ecd470bf6`. Nenhuma mudança em `main`; sem force-push/rebase.
- **Finding independente anterior fechado parcialmente nesta rodada:** grants de `PUBLIC` persistidos em tabelas/sequences/routines públicas (incluindo `SECURITY DEFINER`) podiam atravessar a fronteira esperada; também havia guia principal dizendo incorretamente que rotação HMAC não era suportada.
- **Correção:** cutover e init revogam ACLs `PUBLIC` e runtime de todas as relations/sequences/routines públicas, liberando somente `pgcrypto.hmac` para o migrator; migrations fixam defaults globais e de `public` para functions/tables/sequences. A readiness recusa privilégios efetivos em objetos públicos não allowlisted, grants `PUBLIC` nas duas tabelas autorizadas/verifier, schemas extras acessíveis e default function ACL inseguro. `docs/CLASS_A_PLUS_GUIDE.md` agora instrui o comando versionado `ollama agent rotate-postgres-key` e rollback apenas por versão superior.
- **Regressões incluídas:** integração PostgreSQL cria tabela PUBLIC-readable, sequence com grants PUBLIC e `SECURITY DEFINER` que expõe dado sentinela; runtime novo e pool existente recusam operar sob shape inseguro; migration remove os grants e ambos recuperam. Outro caso injeta `PUBLIC EXECUTE` em default privileges, exige recusa e reparo. CI de cutover agora semeia os mesmos objetos legacy e verifica que o runtime não possui `SELECT`/`USAGE`/`EXECUTE` depois da preparação. Teste CLI evita regressão textual no guia principal.
- **Evidência verificada:** `CGO_ENABLED=0 go test -tags integration ./internal/agent -run '^TestDistributed' -count=1 -v` PASS contra PostgreSQL 16 + Redis; os subtestes de ACL PUBLIC e defaults passaram. `go test ./... -count=1` PASS; `go vet ./...` PASS; gofmt, `git diff --check` e `bash -n` do bloco PostgreSQL CI PASS. Parser YAML Python e actionlint não estavam instalados; YAML remoto ainda deve ser validado por CI. Serviços locais de teste foram desligados.
- **Review independente:** re-review read-only da remediação ACL está em andamento; não afirmar aprovação limpa antes do resultado. O finding Major da review anterior motivou este fix e já não deve ser considerado fechado sem re-review.
- **O que segue incompleto/bloqueado:** ensaio de cutover legado end-to-end após mudanças de ACL (CI fixture contém o cenário, aguardando execução Actions) e re-review; checks GitHub do SHA `b7822ed`; staging com TLS/HA, backup/restore/rollback operacional, crash/failover e rollout coordenado. PostgreSQL continua candidato, não production-ready; paridade global/manus ainda incompleta.
- **PRÓXIMO PASSO CONCRETO:** receber/reconciliar o reviewer `job_YfyrxXaB`; executar em cluster PostgreSQL 16 descartável o script completo legacy phase 1 + migração Go + phase 2 com tabela/sequence/SECURITY DEFINER `PUBLIC`, e verificar cleanup/ownership/memberships. Depois rerodar CI/checks relevantes, atualizar ambos checkpoints após a rodada verificada e publicar somente em `recovery/ollama-full-snapshot`.

STATUS: P0_RLS_CANDIDATE — contenção ACL publicada e integração local real passou; re-review, cutover pós-fix/CI e gates de produção continuam pendentes.


## Estado atual autoritativo — revisão ACL concluída e cobertura ampliada — 2026-09-29 12:24 -03

- **Último commit funcional enviado:** `6cedbb1e84d035e9010cafeba3dd576d297dc526` (`test(postgres): cover all unsafe default ACL classes`) em `origin/recovery/ollama-full-snapshot`; SHA local/remota conferidas. O commit de correção ACL é `b7822ed569e0a32e3cabd1c7c5949512b70649bd`; checkpoint anterior `9af550297a11c3d29cec6e01e04f83e05535bf84`. Sem mudança em `main`, sem force-push/rebase.
- **Veredito independente:** a re-review read-only da implementação ACL/HMAC concluiu sem bloqueador crítico ou major e sem alteração ACL/HMAC solicitada antes de merge. O único apontamento era cobertura: testar defaults PUBLIC/runtime também para tabelas e sequences, além de functions; esse item foi implementado no commit `6cedbb1` e a consulta direta com `pg_default_acl`/`aclexplode` prova ausência de grants após a migration. O veredito não é uma aprovação de produção.
- **Cobertura final:** regressão PostgreSQL injeta default grants global e de schema para PUBLIC/runtime em funções, tabelas e sequences; readiness falha fechado; migration repara; consulta de catálogo deve retornar zero grants inseguros; runtime volta a abrir. CI também verifica ownership ao migrator dos três objetos PUBLIC legacy após cutover, além da negação de `SELECT`/`USAGE`/`EXECUTE` para runtime. Runbook explica preflight privilegiado de bancos externos.
- **Evidência de banco real:** em cluster PG16 descartável, com legacy-owned tabela com segredo sentinela/GRANT PUBLIC, sequence com grants PUBLIC e SECURITY DEFINER com EXECUTE PUBLIC e `search_path` hostil, phase 1 + migration Go + phase 2 passaram. Antes e depois, runtime efetivo não tem SELECT/USAGE/EXECUTE nesses objetos; owner dos três objetos e do pgcrypto foi migrator; legacy role ficou NOLOGIN sem memberships bilaterais. O teste `TestDistributedPostgresRuntimeRejectsRLSDrift`, com regressões ACL e defaults ampliados, passou nesse cluster.
- **Gates locais no estado final:** `go test ./... -count=1`, `go vet ./...`, pacotes focados, `CGO_ENABLED=0 go test -tags integration ./internal/agent -run '^TestDistributed' -count=1 -v` contra PostgreSQL 16 + Redis, YAML parser `gopkg.in/yaml.v3`, `bash -n` no job CI e `git diff --check` passaram. Clusters auxiliares `phaseoneprobe`, `reassignprobe` e `aclcut` foram removidos/parados; PostgreSQL main e Redis de teste ficaram parados.
- **CI remoto:** check-runs e `gh run list` retornaram vazios para a branch/SHA; o workflow atual roda em pull_request ou push para `main`, não push para branch recovery. Portanto, não alegar CI GitHub verde; nenhum PR foi criado.
- **Pendências/limites:** PostgreSQL/RLS continua candidato, não production-ready. Falta staging real com TLS/HA, backup/restore e rollback operacional, failover/crash, rollout coordenado e observabilidade; também faltam gates de release de Windows/artefatos e a conclusão global de parity Ollama Full/Manus. Não há evidência operacional desses itens nesta sessão.
- **PRÓXIMO PASSO CONCRETO:** preparar ensaio em staging separado e autorizado (cluster/DSNs, TLS/CA, backup e janela de manutenção), executar restore/rollback e failover com verificação independente; até essa evidência, manter a arquitetura bloqueada para produção. Para executar CI remoto, criar/usar PR somente conforme autorização do usuário; branch recovery já está sincronizada.

STATUS: RLS_ACL_REVIEW_NO_BLOCKERS — correção, recomendação menor de teste e gates locais concluídos; staging/release global permanecem pendentes.


## Estado atual autoritativo — backup/restore TLS ensaiado — 2026-09-29 12:31 -03

- **Último commit funcional enviado:** `1bf058b769395861e308fa986f7baf040e45bc46` (`docs(postgres): require verified TLS in restore rehearsal`) em `origin/recovery/ollama-full-snapshot`; SHA local/remota idênticas. Sem alterar `main`, sem force-push/rebase.
- **Runbook corrigido:** restore rehearsal agora exige DSNs source e staging com CA explícita e `sslmode=verify-full`, define `PGSSLROOTCERT` também para `pg_restore`, separa os admin DSNs de origem do maintenance DSN de staging, recusa system identifiers iguais e documenta `CREATEDB` + `pg_monitor` para o staging_restore_admin. Integrity guard testa esses invariantes e a CI duplicada de ownership foi limpa.
- **Evidência operacional local real:** two clusters PostgreSQL 16 efêmeros distintos, certificados self-signed com SAN IP/DNS, clientes `psql`, `pg_dump` e `pg_restore` usando verify-full. IDs observados foram distintos; restore custom format finalizou e preservou 1 mission + 1 event (`RESTORED_COUNTS=1|1`). Depois, fault injection imediatamente após CREATE DATABASE acionou o trap; os dois clusters, DB temporário, CA e dump foram removidos. Não se tocou em dados persistentes.
- **Gates finais desta rodada:** `go test ./... -count=1`, `go vet ./...`, `scripts/check-class-a-plus-integrity.sh`, YAML parse `gopkg.in/yaml.v3`, `bash -n` dos trechos CI e do runbook, `git diff --check` passaram. Como só runbook/CI guard mudaram após a rodada ACL, os testes reais PG16 de cutover/ACL permanecem evidência registrada no bloco anterior.
- **CI remoto:** check-runs seguem vazios para a branch recovery, pois Actions corre em `pull_request` ou push a `main`; nenhum PR foi criado nem se afirma CI remoto verde.
- **Pendência importante:** isto é um rehearsal local descartável e não substitui staging externo/operacional. Permanecem TLS/CA emitidos pelo operador, HA/replicação, crash/failover, coordenação real de maintenance, backup automatizado com retenção, RPO/RTO, restore de backup completo do ambiente com roles/extension, rollback operacional independente e observabilidade/alertas. Não declarar production-ready.
- **PRÓXIMO PASSO CONCRETO:** criar plano/runbook de staging e HA baseado nos DSNs/CA reais, coletar evidência de replicação/failover/rollback com revisão independente, e executar CI via mecanismo autorizado (PR se aprovado). Sem credenciais/ambiente staging disponibilizados, implementar somente harness local e manter status BLOCKED para validação operacional externa.

STATUS: POSTGRES_LOCAL_RESTORE_TLS_PASS — restore real + fault-cleanup demonstrados em clusters locais distintos; produção segue bloqueada por staging HA/failover/restore operacional.


## Estado atual autoritativo — failover PostgreSQL 16 ensaiado — 2026-09-29 12:42 -03

- **Último commit funcional enviado:** `9dc80fa16619704b49d30c2fcf0e2833b403b929` (`docs(postgres): record tested fenced failover rehearsal`) em `origin/recovery/ollama-full-snapshot`; branch local/remota sincronizadas. Não alterado `main`, sem force-push.
- **Runbook novo:** `docs/agentic/POSTGRES_HA_FAILOVER_REHEARSAL.md`, linkado pelo runbook de integrações. Registra limites explícitos: só mecânica manual local; split-brain, fencing/quorum, modo síncrono, RPO/RTO, HA automatizada, CA real e backup operacional continuam sem validação de produção.
- **Evidência PostgreSQL 16 real em clusters descartáveis:** streaming replication TLS `verify-full`, `pg_basebackup -X stream -R -C`, marcador replicado (2 linhas), primary parado/fenced antes da promoção, `pg_promote`, saída de recovery e novo write no promovido (`3/1`). A fault injection depois da promoção terminou com cleanup verificado; não restaram cluster, listeners nem diretório CA/data. Este ensaio não prova failover automático ou ausência de perda em rede particionada.
- **Gates da rodada:** `go test -json ./... -count=1` com `GO_TEST_EXIT=0`; `go vet ./...` passou; integrity guard, parse YAML, `bash -n` dos snippets de restore/HA, diff check, varredura redigida de padrões de secrets e estado de clusters/portas passaram.
- **CI GitHub remoto:** sem check associado ao recovery SHA (workflow de qualidade executa em PR/push main). Não criar PR nem alegar execução sem autorização/resultado.
- **Estado do produto:** PostgreSQL RLS/HMAC/cutover tem evidência adversarial local previamente registrada; restore TLS entre clusters distintos passou na rodada anterior; failover manual local passou nesta rodada. **PostgreSQL ainda não é production-ready**: staging real, quorum/fencing automatizado, replicação/HA, failover/rollback do antigo primary, backup completo/restoration, RPO/RTO, TLS/CA operacional, alertas e review operacional independente pendem.
- **PRÓXIMO PASSO CONCRETO:** obter ambiente staging real e decisão operacional sobre modo de replicação, quorum/fencing e metas RPO/RTO; executar os casos de perda de primary/standby/rede particionada e restore/rollback com evidência assinada por operadores. Até haver credenciais/topologia autorizadas, manter gate de produção bloqueado e seguir próximos itens locais independentes sem simular aprovação.

STATUS: LOCAL_PG16_TLS_RESTORE_AND_MANUAL_PROMOTION_PASS — produção bloqueada por HA/DR operacional externo.


## Atualização de referência técnica — PostgreSQL 16 oficial — 2026-09-29 12:43 -03

O guia HA foi conferido contra as páginas oficiais versionadas do PostgreSQL 16 sobre [failover/old primary e fencing](https://www.postgresql.org/docs/16/warm-standby-failover.html), [`pg_basebackup`/WAL streaming/slots](https://www.postgresql.org/docs/16/app-pgbasebackup.html), [TLS verify-full e CA](https://www.postgresql.org/docs/16/libpq-ssl.html) e [`pg_promote(wait, timeout)`](https://www.postgresql.org/docs/16/functions-admin.html). Os links/version pin estão no próprio runbook; a orientação de fencing corresponde ao alerta oficial de evitar ambos os primaries graváveis.

**Último commit funcional agora:** `31ce00bc0f8ac3e534170cb84b08abd43815c64c` (`docs(postgres): cite pg16 failover guarantees`), remoto e local iguais. Integridade, parser YAML, Bash syntax, diff check e inspeção de ausência de cluster/porta descartável passaram após a edição de referências. As evidências anteriores de teste Go completo (`GO_TEST_EXIT=0`), vet, restore TLS e promoção permanecem aplicáveis, sem alteração de código Go nesta pequena atualização documental.

Continua pendente e bloqueante para produção: HA/staging de operador real, quorum/fencing, rede/partições, backups/retention, recuperação do antigo primary, metas RPO/RTO, alertas e revisão independente operacional. Nenhuma autorização/credencial de staging fornecida; não fabricamos execução remota.


## Estado atual autoritativo — referências visuais + re-review PostgreSQL — 2026-09-29 13:13 -03

- **Branch/HEAD confirmado antes deste checkpoint:** `recovery/ollama-full-snapshot`, código `00ee89d406578a7ab3b3bca5889bd7d3e5e5ee9d`, remoto igual. Nenhuma alteração em `main`; sem force-push.
- **Entrega visual solicitada:** pacote `/home/ubuntu/ollama-full-reference-images.zip`, SHA-256 `fd988e8213643efc3a337b18e48878714c40d8a9befcc152ac240b703614c713`, 24.7 MB. Contém 13 imagens PNG 2560x1440 (uma para cada rota principal `/`, `/c/$chatId`, `/agentic`, `/company`, `/connect`, `/plugins`, `/skills`, `/library`, `/scheduled`, `/settings`, `/onboarding`, `/projects`, `/tasks`), folha de contato, README e manifest CSV. `unzip -t` sem erros; cada PNG aberto/verificado e dimensões conferidas. As telas são conceito futuro e dados ilustrativos, não capturas de produto pronto. O ZIP é artefato de entrega fora do Git para evitar inflar o repositório.
- **Novo review independente read-only do snapshot publicado:** job `job_NoiyZDTQ`, sem findings Critical, mas **Request changes para aprovação de produção** com três findings Major: (1) `validatePostgresLegacySchemaShape` aceita colunas extras apesar de grants DML em tabela inteira, permitindo dados inesperados sensíveis; rejeitar colunas não allowlisted (e avaliar generated/identity/trigger shape) e testar coluna-segredo. (2) readiness não audita `pg_attribute.attacl`/privilégios por coluna, deixando passar grants isolados a colunas sensíveis, inclusive na keyring; adicionar auditoria coluna a coluna e teste runtime SELECT de coluna apenas. (3) init/readiness não revogam/atestado `PUBLIC CONNECT` e `public` schema USAGE; definir baseline mínimo do PUBLIC para deployment fresh e verificar permissões de database/schema no startup. Reviewer acrescentou minor: corrigir redação de histórico de rotação no doc e validar forma/cardinalidade do legacy singleton table.
- **Veredito:** signed-context/RLS e rotação são substancialmente melhores e evidência local PG16 permanece válida, mas produção **não aprovada** até fechar os três Major e repetir adversarial. Mesmo após code fixes, staging operacional HA/DR/TLS e análise real ainda são gates externos.
- **Pergunta Manus Flex:** documentação pública oficial consultada descreve Manus como agente que planeja/executa tarefas em sandbox com filesystem persistente, podendo instalar software e criar ferramentas (`https://manus.im/docs/introduction/welcome`); a página pública consultada não define especificamente “Manus Flex”. Não inferir elegibilidade/plano/créditos. Manus pode ajudar como agente adicional se autorizado o acesso ao repo/branch, mas não substitui testes/review/validação externa nem garante conclusão automática.
- **PRÓXIMO PASSO CONCRETO:** corrigir os três findings Major (schema column allowlist, auditoria attacl/has_column_privilege, revogação/atestado PUBLIC CONNECT + schema ACL), acrescentar regressões reais PostgreSQL 16 e rodar cutover/readiness adversarial; em seguida atualizar este estado e handoff e fazer commit+push apenas para a branch recovery. Preservar o gate de produção como BLOCKED.

STATUS: UI_REFERENCE_ZIP_VERIFIED; POSTGRES_PRODUCTION_REQUEST_CHANGES — artefato visual entregue; RLS precisa fechar 3 findings Major e gates operacionais externos permanecem pendentes.


## Estado autoritativo — PostgreSQL isolation/cutover/HBA re-review — 2026-09-29 17:33 -03

- **Branch/código enviado:** `recovery/ollama-full-snapshot`, commit `30ae8f22394205115e1d9154dae9866fcee82a52` (`fix(security): close postgres isolation and cutover races`). Push normal confirmado; `origin/recovery/ollama-full-snapshot` aponta para o mesmo SHA. Sem force-push; nenhuma alteração em `main` ou nas branches de UI/PR.
- **Correções desta rodada:** capturar memberships legadas com CTE recursivo sobre as arestas `pg_auth_members` (sem `pg_has_role`/overmatch de superusers), usar lock curto e snapshot/revoke adicional durante transferência de ownership em phase 1, drenar e verificar novamente sessões após commit; phase 2 também drena os logins/membros capturados. Admins independentes não são alvos do drain. A propriedade de `public.hmac(bytea,bytea,text)` é verificada via `pg_proc`/`proowner`.
- **HBA corrigido e guardado:** regras TCP de `ollama_agent_runtime` e `ollama_agent_migrator` limitadas ao database `ollama_agent`, seguidas dos rejects `all`; admin continua com SCRAM TCP e trust admin limita-se ao socket do container. Integridade testa ausência de allows amplos, contagem e ordem das regras. Exceção de `ollama_agent_reverse_member` existe somente no arquivo temporário gerado pelo CI em `$RUNNER_TEMP`, com login TCP positivo provado; não aparece no HBA de produção.
- **Evidência local reproduzível desta rodada:** cluster PG16/Redis descartável com fresh init; cutover legado com `pgcrypto`/key import e search_path hostil; login direto e `SET ROLE` de membro encerrados em phase 1 e phase 2; sessão de admin independente sobrevive; owner do HMAC = migrator; runtime/migrator negados em DB recém-criado com `PUBLIC CONNECT`, runtime aceito no DB app; suíte Distributed PostgreSQL RLS drift/key rotation e Redis passou. Cleanup confirmou remoção do cluster/ports temporários.
- **Gates finais:** `go test ./...`, `go vet ./...`, `go test -race ./internal/agent`, `go build`, validação YAML Actions + bash embutido, Compose YAML, `bash -n`, integrity guard, `git diff --check` e Gitleaks staged passaram. Segunda revisão independente aprovou o diff sem blockers. CI remoto ainda não foi executado/confirmado para esse SHA.
- **Reconciliação somente leitura (2026-09-29):** PR [#38](https://github.com/DZ23-LTDA/hades-ollama/pull/38) segue OPEN, branch `fix/audit-security-deps-2026-09-25` SHA `07a4f0bca2cbe327d5234abd9ebec69ff52154e6`, base `main` SHA `8635e30dc9e95a1f5b29700169783abc24093ceb`, mergeable segundo API. `recovery...fix/audit-security-deps` está `diverged` (recovery +35, -27; merge-base `8635e30d`; 121 arquivos), e PR #38 inclui Connectors/Providers/quick-connect/ícones e código de integração. `recovery...feat/ui-shell-parity` também está `diverged` (+17/-15; merge-base `add5881a`; 57 arquivos). Não fiz merge, cherry-pick ou push em qualquer outra branch. Não continuar duplicando marketplace de conectores antes de reconciliar as árvores.
- **Limite operacional:** scripts pressupõem janela de cutover com outros DBAs/superusers e automações privilegiadas quiescidos; um superuser separado pode alterar membership depois da liberação de um lock. Mantenha runtime/tráfego parados até migration, phase 2, smoke e observabilidade.
- **Produção continua BLOCKED:** estes testes são locais, não provam staging real, HA/quorum/fencing, failover em partição, RPO/RTO, backup/restore operacional, CA/TLS real, alertas nem aceite de operador. Não declarar produção pronta.
- **PRÓXIMO PASSO CONCRETO:** fazer análise de consolidação (sem alterações ainda) de `recovery`, PR #38 e `feat/ui-shell-parity`; construir matriz por commit/arquivo para escolher a base canônica e preservar a implementação melhor de Connectors/Providers, a UI aditiva e as correções de segurança. Manter o status do PR #38 e as outras branches read-only até a decisão de integração; então executar integração controlada, gates completos e novo checkpoint em recovery.

STATUS: LOCAL_SECURITY_GATES_PASS; REVIEW_APPROVED; RECOVERY_PUSHED; PRODUCTION_BLOCKED; BRANCH_CONSOLIDATION_PENDING.


## Handoff portátil — branch audit e pacote — 2026-09-29 18:10 -03

A verificação read-only mais recente (17:43 -03) confirmou: PR #38 OPEN contra `main`; base `main`=`8635e30dc9e95a1f5b29700169783abc24093ceb`, head `fix/audit-security-deps-2026-09-25`=`07a4f0bca2cbe327d5234abd9ebec69ff52154e6`, PR marcado mergeable. Head de `feat/ui-shell-parity`=`132e77155fed682c0b6244d256e6cdfe9b25c239`. Recovery antes deste handoff=`65cb08bec4880447369690668b2c35e024319049`. Com recovery como base, PR diverged: head do PR +35 commits, recovery +29, 121 arquivos e merge-base `8635e30d`; UI parity diverged 17/17, 57 arquivos e merge-base `add5881a`. PR #38 possui páginas/rotas Connectors e Providers, quick-connect, catálogo Go, icons/upload; UI parity tem sobreposição em `ProductWorkspacePage`, `ConnectorLogo`, `connectorCatalog.ts`, além de tema/Home/console/jornadas. Não houve merge nem alterações nessas branches.

Foi criado `audit/HANDOFF_MANUS_20260929.md`. O ZIP acordado inclui snapshot tracked, Git bundle seletivo de `main`, recovery, PR #38 e UI parity, checkpoints e `/home/ubuntu/ollama-full-reference-images.zip` preservado sem extração; exclui o antigo arquivo backup de P: para evitar duplicação. O checkout tem cerca de 53 MB de objetos tracked (contra 1.2 GB de working directory local, que não deve ser arquivado). Próximo passo atual: concluir correção do checkpoint se necessário, gerar e validar o ZIP e incluir manifest/README com os SHA finais. Depois, a próxima missão prepara a matriz de consolidação sem mesclar automaticamente; manter staging/prod como BLOCKED pelas validações HA/TLS/restore/RPO-RTO/alertas pendentes.


## Pacote ZIP para próximo Manus — revalidação final — 2026-09-29 18:12 -03

Último SHA publicado antes deste apêndice: `6980786c484192ace6b735458bc8e51ab7ef7279` em recovery; branch limpa/alinhada. A comparação GitHub foi repetida contra esse SHA. PR #38 continua OPEN com head `07a4f0bca2cbe327d5234abd9ebec69ff52154e6`: PR head +35 e recovery +31 commits, 121 paths, merge-base `8635e30dc9e95a1f5b29700169783abc24093ceb`. UI parity head `132e77155fed682c0b6244d256e6cdfe9b25c239`: UI head +17 e recovery +19 commits, 57 paths, merge-base `add5881a260ff1f740b1340c6f394c26acc2d5d2`. Esses deltas incluem os commits de checkpoint que recovery recebeu desde consultas anteriores; interpretar sempre com base/HEAD explícitos.

O usuário confirmou gerar pacote com snapshot atual, bundle dos heads `main`, recovery, PR38 e UI, checkpoints/guia e o arquivo binário `ollama-full-reference-images.zip`; excluir `ollama-full-recovery-for-P-drive.zip`, backup antigo. Os binários não devem ser extraídos ou processados; somente verificar/copiar o ZIP visual preservando bytes. Snapshot source tracked estimado em ~53 MB; os 1.2 GB na pasta local incluem arquivos que não devem ser transportados. Próxima ação: concluir esta documentação por commit/push recovery; gerar `ollama-full-handoff-20260929.zip` com README, snapshot, bundle, guia de handoff, checkpoints e referências; manifest/checksums; testar o ZIP externo e bundle; entregar SHA-256, tamanho e link. Não incluir `.git` como pasta, caches, `.env`, credentials, banco/dumps nem o backup antigo P:. Nenhuma branch foi mesclada.


## Freeze dos refs para o pacote — comparação ancorada em 3203fa1 — 2026-09-29 18:14 -03

A comparação mais recente, usando recovery `3203fa1f732a013c35bd46467a2c15debdd27466` como base imediatamente antes do commit final de documentação, registrou PR #38 head `07a4f0bca2cbe327d5234abd9ebec69ff52154e6`: diverged, PR +35 e recovery +32, 121 arquivos, merge-base `8635e30dc9e95a1f5b29700169783abc24093ceb`; UI parity `132e77155fed682c0b6244d256e6cdfe9b25c239`: diverged, UI +17 e recovery +20, 57 arquivos, merge-base `add5881a260ff1f740b1340c6f394c26acc2d5d2`. O commit a seguir é só documentação, não modifica esses heads das outras branches; snapshot do ZIP será um commit de docs mais novo. PR #38 continuava OPEN/mergeable na consulta às 18:14.


## Correção SHA de snapshot no guia do pacote — 2026-09-29 18:17 -03

Review detectou que o guia chamava o SHA inicial da rodada de segurança `65cb08...` de snapshot incluído, embora a fonte já tenha avançado por commits documentais. Corrigido: segurança base=commit `30ae8f22...`; `65cb08...` é checkpoint anterior; `SOURCE_SNAPSHOT_SHA` do README é o SHA exato exportado. Esta mudança exige novo commit/push recovery e rebuild do ZIP/bundle. Último commit remoto antes da correção=`24b5eee05dd3184074f8433b2397d68d3a571c69`. Próximo passo: verificar diff, scan, commit/push; re-exportar `source/` e bundle do SHA resultante, atualizar README/manifest, retestar ZIP CRC e hashes.


## Verificação do candidato de handoff — 2026-09-29 18:19 -03

O candidato `ollama-full-handoff-20260929-v2.zip` foi criado e validado em CRC/manifest (1.871 entradas, 132.569.470 bytes); bundle de quatro refs validado e comparação byte a byte confirmou o ZIP visual copiado. SHA-256 do candidato `3ac44659eb923285ead1c04d3d4d0b8cf8fd9748f093fca33b40bbe539a46056`. Esse candidato foi gerado do source SHA `304f1224...`; antes da entrega, este próprio checkpoint será commitado/pushado e o pacote definitivo regenerado do novo SHA para conter a atualização também. Próximo passo: push normal desta nota e rebuild final, depois `unzip -t`, `sha256sum -c` e confirmar que source SHA, bundle recovery ref e origin head coincidem.


## Pacote de passagem de bastão — pronto para rebuild final — 2026-09-29 18:21 -03

O checkout recovery está em `0ba6b15f5dc55c0b8667a5561478a14bf81e3fd4`, limpo e alinhado com origin. O snapshot v3 foi exportado desse SHA, o Git bundle validou exatamente main/recovery/PR38/UI, o ZIP visual foi comparado byte a byte, o manifest SHA-256 passou e `zip -T`/`unzip -t` passaram; 1.871 entradas e ~132.6 MB. Antes da entrega, este checkpoint será publicado por commit normal; em seguida o pacote será recriado do SHA desse commit e só será entregue quando README SOURCE_SNAPSHOT_SHA, bundle recovery ref e origin coincidirem. Conteúdo aprovado conforme escopo confirmado: source tracked, históricos refs, checkpoints/guia, ZIP visual binário; sem backup P: velho, secrets, `.git` dir, cache ou DB dumps. Não foram mescladas outras branches. Próxima/última etapa: push deste checkpoint, rebuild final, validar checksums/CRC, entregar link e hash externo.


## Consolidação PR #38 em recovery — 2026-09-29 19:25 -03

- **Estado publicado:** integração `--no-ff` criada e enviada normalmente somente para `recovery/ollama-full-snapshot`. Merge commit **`8be25f6c144acf1fd11c4d3f55c24f24a8d9c8e5`** (`Merge PR #38 connector and provider features into recovery`), pais `5597da4e2e0f852b64805a23c8902fb7efa8c37b` (recovery pré-merge) e `07a4f0bca2cbe327d5234abd9ebec69ff52154e6` (head do PR #38). Push normal confirmado; sem force-push. Checkout limpo após o push.
- **Limites de branch respeitados:** `main` permanece `8635e30dc9e95a1f5b29700169783abc24093ceb`; head da branch do PR continua `07a4f0bca2cbe327d5234abd9ebec69ff52154e6`; `feat/ui-shell-parity` continua `132e77155fed682c0b6244d256e6cdfe9b25c239`. PR #38 ainda está OPEN/MERGEABLE, base `main`. Nenhuma dessas refs foi alterada; o PR não foi fechado nem mesclado em `main`. UI parity segue fora da recovery. Comparação atual UI parity versus recovery: diverged; UI +17, recovery +60, merge-base `add5881a260ff1f740b1340c6f394c26acc2d5d2`.
- **Conteúdo integrado:** melhor experiência de Connectors/Providers/quick-connect/catálogo/ícones e implementação associada de uploads, Windows e MCP do PR #38, combinadas com a shell/API/isolamento e segurança PostgreSQL/RLS/HMAC/HBA mantidas da recovery. Não foi copiado o marketplace duplicado de `feat/ui-shell-parity`.
- **Achados de segurança de quick-connect fechados antes do commit:** sessões Agent/organização seguem do browser same-origin para a ponte UI; autenticação é pré-validada antes de gravar segredo; falha de registro restaura/remove segredo e recusa overwrite se credencial anterior não puder ser lida; políticas de catálogo são GET-only e restritas por provider/path, sem política ampla; nested URL-encoding, backslash e traversal são rejeitados antes do matching; nested slices retornadas são cópias. Review independente read-only não encontrou blocker restante nesses pontos. Testes cobrem forwarding, rollback, policy e traversal.
- **Gates locais finais:** `go test ./...`, `go vet ./...`, `go test -race ./internal/agent`, `go build ./...`, cross-compile Windows de `internal/agent` e `app/ui`; frontend Vitest **34 arquivos/244 testes**, build e ESLint focado; workflow Actions YAML + 26 blocos Bash, Compose YAML (3 serviços), integrity guard, gofmt/diff check e Gitleaks sobre patch staged passaram. Build UI emite apenas warning conhecido de chunk >500 kB. Harness real descartável PG16/Redis nesta árvore confirmou fresh init, cutover legado fase 1/2, RLS, rotação e isolamento cross-database; os arquivos de segurança PostgreSQL da recovery permaneceram inalterados neste merge.
- **CI GitHub:** consulta pós-push não encontrou runs para a branch naquele instante (`gh run list` vazio); portanto CI remoto deste SHA **não está comprovado**. Verificar novamente antes de tratar a integração como validada em GitHub.
- **Estado do produto:** merge concluído; isto **não** prova paridade com Manus nem readiness de produção. Staging PostgreSQL, HA/quorum/fencing/partições, backup/restore, RPO/RTO, TLS/CA operacional, alertas/observabilidade e aceite do operador continuam bloqueadores externos.
- **PRÓXIMO PASSO CONCRETO:** no próximo turno, confirmar SHA/branch limpos e consultar Actions para `8be25f6c`; corrigir qualquer falha de CI somente em recovery, com novos gates e checkpoint. Depois, elaborar uma matriz do que é aditivo na UI parity sem importar seu marketplace duplicado e priorizar o chat como executor de missão (tools, filesystem, terminal, browser/MCP, artifacts e approvals). Não refazer merge do PR #38, não tocar `main`, `fix/audit-security-deps-2026-09-25` ou `feat/ui-shell-parity` e não declarar produção pronta.

STATUS: PR38_MERGED_INTO_RECOVERY_AND_PUSHED; ALL_LOCAL_GATES_PASS; REMOTE_CI_NOT_YET_OBSERVED; UI_PARITY_UNMERGED; PRODUCTION_BLOCKED.

## CI recovery: correções publicadas — 2026-09-29 20:15 -03

Último SHA publicado: `2aa59a20505a627d936856a350eb784eb81b63f4` (branch recovery/ollama-full-snapshot).

Publicado nesta rodada:
- `7ed0a385`: workflows de qualidade/integridade passam a rodar em push da recovery.
- `16e71858`: fixture de CI concede CONNECT ao membro reverso; testes de isolamento
  descriptor-bound e sandbox.exec guardados como Linux-only (Windows: 31 -> 2 falhas).
- `2aa59a20505a627d936856a350eb784eb81b63f4`: GRANT CONNECT movido para dentro do bloco SQL do fixture, antes do probe de login.

Estado dos gates: locais 100% verdes (test/vet/race/build/cross-compile Windows/gofmt/
integrity/YAML/Compose/Gitleaks). `class-a-plus-integrity` remoto = SUCCESS.
`dz23-agentic-quality` ainda falha nos jobs PostgreSQL e Windows — ver
`audit/CI_DIAGNOSTICO_20260929.md` para as duas causas restantes de Windows.

Pela metade: (1) confirmar o run de `2aa59a20505a627d936856a350eb784eb81b63f4`; (2) Git confiável no runner Windows;
(3) herança de DACL via ACLFromEntries no Windows.

PRÓXIMO passo concreto: ler o `--log-failed` do run de `2aa59a20505a627d936856a350eb784eb81b63f4` e, se o job PostgreSQL passou,
atacar as duas falhas de Windows com evidência real do runner (`icacls`) antes de qualquer
declaração de produção.

Guia portátil para retomada: `audit/HANDOFF_NEXT_MANUS_20260929.md`.


## Checkpoint — 2026-10-03: início da rodada de correções UX/segurança

- Solicitação explícita do usuário: “bora corrigir e melhorar”.
- Branch autorizada e ativa: `recovery/ollama-full-snapshot`; a branch `main` não será alterada. O tip remoto foi verificado antes desta rodada; não haverá commit, push nem publicação sem pedido.
- O relatório entregue em `outputs/auditoria_hades_ollama_2026-10-03.txt` e artefatos preexistentes em `.work/`/`.manus/` devem ser preservados.
- A árvore de código estava sem alterações rastreadas antes desta rodada. A rodada será incremental e coberta por testes: (1) anexos da Home persistidos por API, associados a projeto/organização e indexados; não transportar bytes em query string/sessionStorage nem afirmar sucesso antes de confirmação do servidor; (2) limitar privilégios do autorun e cobrir o comportamento no backend/UI; (3) corrigir onboarding/privacidade com copy condicional e tradução coerente; (4) tornar a busca documental útil mostrando trechos citados, sem apresentar instruções RAG como resposta gerada.
- Limite técnico reconhecido: o endpoint atual de documentos monta contexto para uma camada de resposta mas não chama inferência, e `agentAPI` não tem provider/modelo explícito; a UI desta rodada explicará a recuperação e apresentará snippets/citações. Não será adicionada geração implícita nem egress a provider externo.
- Validação prevista: testes focados, lint/build frontend e testes Go nos pacotes/rotas alterados. Registrar limitações reais do toolchain Windows/CGO, sem declarar gates não executados como aprovados.


## Checkpoint — 2026-10-03: primeira fatia de correções concluída localmente

- Branch ativa: `recovery/ollama-full-snapshot`, HEAD-base `553002b2915e716ba0720fa6a04edce2df08cbac`; antes do commit, a ref remota GitHub foi confirmada no mesmo SHA. `main` não foi alterada.
- Escopo entregue nesta fatia: upload multipart real de anexos da Home para projeto org-scoped/indexado (10 arquivos; 10 MiB por arquivo; 32 MiB agregado; allowlist do indexador; nomes/paths validados); conteúdo não vai em query string; provider remoto com anexos exige consentimento; escrita desliga autorun na Home e o Console só cria missão manual, enquanto o backend aceita autorun apenas com `workspace:read`.
- A busca documental agora é descrita como busca de trechos, renderiza snippets/fontes e não devolve ao cliente o prompt/contexto RAG interno.
- Onboarding: removida a alegação absoluta “never logged or trained on”; copy agora distingue modelos locais e remotos. A localização integral do onboarding em PT-BR permanece pendente para uma próxima fatia.
- Novos testes: importação e allowlist Go, endpoint multipart, validação de autorun, API documental sem `context`, Home upload/navegação/consentimento, snippets e copy de privacidade.
- Evidências locais: `go test ./internal/agent -run 'TestSupportedProjectDocumentFilename|TestProjectImport' -count=1`, testes selecionados de `server` (anexos, autorun e RAG) e `go vet ./internal/agent ./server` passaram; Vitest completo: 56 arquivos/317 testes passaram; `npm run build` passou (warning conhecido de chunk acima de 500 kB); `npm run lint` passou; `git diff --check` passou.
- Gate amplo `go test ./... -count=1` não ficou verde neste Windows: falhou por `CGO_ENABLED=0`/sqlite stub nos pacotes que usam `go-sqlite3` e por testes `mlx` com símbolos ausentes (`gelu`, `currentScope`, `gatedDeltaGraph`, etc.). Os testes Go focados dos pacotes alterados passaram; não declarar a suíte Go completa aprovada.
- Instrução mais recente do usuário (2026-10-03 10:15): corrigir melhorias e subir ao GitHub. Autorização abrange commit e push normal somente para `recovery/ollama-full-snapshot`; não abrange PR, `main`, force-push ou alterações em outras refs. Atualizar este checkpoint com o SHA após confirmar o push.


## Checkpoint de publicação — 2026-10-03

O commit `6473ed1c4d22763bad103c2d563bafd4b11b78f9` foi criado com exatamente um trailer `Co-authored-by: Manus <dev-agent@manus.ai>` e enviado por push normal para `recovery/ollama-full-snapshot`. A ref remota foi relida e coincide com esse SHA; `main` não foi escrita. Não foi aberto PR, pois a instrução foi subir as correções à branch autorizada. No momento do registro, `class-a-plus-integrity` (run 37125844818) e `dz23-agentic-quality` (run 37125844862) estavam `in_progress`; aguardar e substituir este estado provisório pelo resultado final.


## CI remoto — resultado final do commit 6473ed1c

Verificados os workflows associados ao SHA publicado: `class-a-plus-integrity` (run 37125844818) e `dz23-agentic-quality` (run 37125844862) concluíram com sucesso; `gh run watch` retornou exit code 0 para ambos. No integrity, o gate CGO-free/Windows build e a preservação Class A+ passaram. No quality, os jobs Go agentic/server, PostgreSQL RLS/Redis DLQ/OTLP, Windows portability, web/mobile, SBOM e classificação de superfícies concluíram com sucesso. O relatório registrou os avisos upstream do GitHub sobre Node.js 20 e a migração futura de `ubuntu-latest`; não foram falhas do commit.


## Checkpoint — auditoria independente pós-remediação — 2026-10-03 11:10 -03

- Branch: `recovery/ollama-full-snapshot`; HEAD e tip remoto confirmados em `6473ed1c4d22763bad103c2d563bafd4b11b78f9`. `main` não foi alterada. A árvore de código estava limpa; permanecem arquivos locais de auditoria/planejamento que devem ser preservados.
- GitHub Actions consultado pelo SHA exato: `class-a-plus-integrity` run `37125844818` e `dz23-agentic-quality` run `37125844862`, ambos `completed/success`, head_sha exatamente igual ao candidato. Isto não equivale a release-readiness/signing/provenance.
- Seis revisões read-only e consolidação concluídas no mesmo SHA. Findings confirmados: listener não-loopback aceita HTTP sem TLS; Browser Operator não intercepta redirects/subrecursos privados e roda sem isolamento de host; cadeia de release/provenance e downloads de toolchain sem digest verificado; quotas/GC Redis ausentes; cancelamento não se propaga entre instâncias; health pode mascarar erro; APIs/SSE sem paginação/cursor/heartbeat; resposta de execução declara RUNNING antes de claim; fsync de diretório ausente após rename; migration ledger ausente; falhas mobile (cloud opt-in payload, approval payload, outbox, troca de usuário); continuidade Home após import, acessibilidade/i18n; importação sem status por arquivo; search não descobre fontes; spend ledger desconectado; deploy sem health/rollback remoto; MCP matrix superestima a capacidade; SECURITY.md está desatualizado sobre a rotação HMAC.
- Correções anteriores confirmadas/não repetir: anexos Home são enviados antes de navegar e bytes não entram na URL; autorun e permissão de escrita foram restringidos; copy de privacidade já remove alegação absoluta; busca documental não devolve contexto bruto; alias DLP, manifestos locais, auth remota e cancelamento na mesma instância foram tratados/cobertos.
- Próxima execução autorizada pelo usuário: continuar corrigindo e fazer commit/push normal somente em `recovery/ollama-full-snapshot`; sem PR, sem `main`, sem force-push. Trabalhar em fatias testáveis; preservar `outputs/`, `.work/` e `.manus/`.
- Gates externos que permanecem bloqueados: SignPath/Authenticode/OIDC attestation e trust anchor; runners macOS/arm64 e downloads MLX; staging TLS/mTLS; Playwright instalado para Browser Operator; Redis/PostgreSQL/roles e fault-injection; providers/search/deploy reais; mobile físico/EAS/push; leitores de tela. Não converter ausência de ambiente em PASS.


## Checkpoint — continuação Claude (validação do WIP de remediação) — 2026-10-03

- Branch `recovery/ollama-full-snapshot`; tip local e remoto confirmados em `6473ed1c` antes do commit (push fast-forward). `main` não tocada; sem force-push; sem PR.
- Revisado o WIP não commitado deixado pela rodada anterior (TLS fail-closed para bind não-loopback; redução de ambiente herdado + bloqueio por tenant do Browser Operator; jornadas Home/Import/Onboarding+cloud opt-in; mobile `App.tsx`/`offlinePolicy.ts` com particionamento por org+usuário e modo offline conservador). Mudanças de segurança consideradas sólidas como defesa em profundidade; mantido o bloqueio fail-closed do Browser Operator por tenant até existir sandbox real de SO/rede.
- Corrigidos os testes que falhavam neste Windows **sem enfraquecer validação**:
  - `server/agent_object_scope_test.go`: corpos JSON agora serializados com `json.Marshal` (eliminado o `invalid escape sequence '\U'` ao concatenar caminhos Windows).
  - `server/agent_mcp_bootstrap_test.go` e `server/agent_catalog_scope_test.go`: fixtures passam a usar `os.Executable()` (caminho absoluto + executável regular real em qualquer SO) em vez de `/bin/echo` ou arquivo sem extensão; a validação estrita de executável permanece intacta.
  - `internal/agent/runtime_test.go`: `TestBrowserOperatorNavigateAndSnapshot` agora faz `t.Skip` quando falta a dependência Python `playwright`/intérprete, em vez de falhar; as asserções permanecem quando a dependência existe (CI instala e exercita o caminho completo).
- Gates locais executados e verdes nesta máquina (Windows):
  - Go: `go vet ./server ./internal/agent`; `go test ./server -count=1` **ok**; `go test ./internal/agent -count=1` **ok** (com 1 SKIP de Playwright).
  - Python: `py_compile` de `browser_helper.py` **ok**; `browser_helper_test.py` 5 testes **ok**.
  - Frontend: Vitest 57 arquivos/321 testes **ok**; `npm run lint` **ok**; `npm run build` **ok** (warning conhecido de chunk > 500 kB).
  - Mobile: `npm run typecheck` **ok**; `npm run test:policy` `offlinePolicy: PASS`.
  - `git diff --check` sem erros (apenas avisos de CRLF/line-ending repo-wide).
- `.gitignore` passou a ignorar `.manus/`, `.work/` e `outputs/` (scratch local de agentes, fora do código). Esses diretórios não foram commitados.
- Não executado / permanece bloqueado por ambiente externo (sem conversão em PASS): suíte Go completa `go test ./...` (sqlite CGO stub e símbolos MLX ausentes neste host); Playwright real para o Browser Operator; assinatura Authenticode/SignPath, attestation/provenance e trust anchor de release; staging TLS/mTLS; Redis/PostgreSQL/roles reais; mobile físico/EAS/push. Release NÃO declarado pronto.


## Checkpoint — rodada "finalizar para usuário final" (Claude) — 2026-10-03

Meta do usuário: deixar tudo pronto para o usuário final e o GitHub no estado
final. Branch `recovery/ollama-full-snapshot`; cada fatia abaixo foi commitada e
enviada por push fast-forward (sem tocar `main`, sem force). CI remoto do
primeiro commit da sessão anterior (`89e92367`) ficou verde
(`dz23-agentic-quality` + `class-a-plus-integrity`).

Fatias entregues (cada uma com teste e gates locais verdes):
- `0c3ab12a` Health não mascara falha da fila (RedisQueue.List/JobQueue.List
  devolvem erro; QueueJobsWithError; health reporta "unavailable"); SECURITY.md
  corrigido sobre a rotação de chave HMAC (comando `rotate-postgres-key` existe).
- `853deed4` `/missions/:id/run` reporta estado real (enfileirada) + queued,
  não RUNNING prematuro.
- `fb20e096` Import de projeto expõe manifesto por arquivo (indexado/ignorado +
  motivo) no backend e na UI (ImportProjectDialog/describeImport).
- `e1ef28a1` Pesquisa aceita SearchProvider opcional (descobre fontes da query)
  com fallback honesto; URLs descobertas passam pelo mesmo fetch SSRF-safe.
- `66b4e428` fsync do diretório pai após rename atômico (POSIX; no-op no Windows).
- `98189d65` Guia rápido para usuário final leigo (docs/GUIA_RAPIDO_USUARIO.md),
  linkado no README, com aviso honesto do SmartScreen (instalador unsigned).
- `bdf4a744` SSE com Last-Event-ID + heartbeat; eventos com ?limit e total
  (fatia segura/compatível de H-12).

Prontidão para usuário final confirmada: binário principal compila
(`go build .`), instalador `app/ollama.iss` coerente (HadesSetup), CI verde,
guia leigo publicado.

Findings que permanecem ABERTOS e NÃO foram feitos (com motivo honesto — não
declarar prontos):
- H-25 spend ledger desconectado do roteamento: exige propagar organization_id
  + possuir o SpendLedger + contabilizar tokens DEPOIS da chamada (dentro do
  planner). Authorize sozinho não enforce. Integração transversal no hot path;
  não validável sem providers reais. NÃO feito.
- H-08/H-09 quotas/GC do Redis: admissão por quota precisa entrar no script Lua
  de enqueue (hot path) e exige Redis real/fake para validar. NÃO feito.
- H-10/S6 cancelamento entre instâncias: marcador durável + checagem pelo worker
  em safe points; toca o loop de execução e precisa de prova multi-processo.
  NÃO feito (a fatia SSE não cobre isto).
- M-15 migration ledger PostgreSQL: precisa de Postgres real para validar. NÃO
  feito.
- M-30 matriz MCP: revisada; a matriz já carrega caveats honestos
  (NOT_CONFIGURED, coluna de gaps externos) e o código MCP é fail-closed — sem
  mudança necessária.
- Bloqueios externos inalterados: Authenticode/SignPath, attestation/provenance,
  staging TLS/mTLS, Playwright real, runners macOS/MLX, mobile físico/EAS.


## Checkpoint — findings de código FECHADOS (autorizado) — 2026-10-03

Com autorização explícita do usuário para assumir premissas de infra, os 4
findings de código que restavam foram implementados, testados e validados
(inclusive pelos jobs de integração do CI com Redis e Postgres reais). Tip
`e5bb698a`, ambos os workflows verdes.

- `81a5c0c0` H-10/S6 — cancelamento entre instâncias: watchDurableCancellation
  propaga o marcador durável para o runCtx local (abortando passo longo em voo
  em qualquer instância). Teste: TestWatchDurableCancellationPropagatesAcrossInstances.
- `8fd68e0d` H-25 — spend cap por organização no roteamento: resolver possui
  SpendLedger por env (OLLAMA_AGENT_SPEND_DAILY_CAP_CENTS/_MONTHLY_CAP_CENTS/
  _LEDGER_PATH); pré-autoriza custo nominal por roteamento e cai para local/free
  com motivo honesto ao estourar. organization_id threadado na interface.
  Teste: TestSpendLedgerEnforcesPaidModelCap. (Hold estimado, não contabilidade
  exata por token — documentado.)
- `0435f36a` H-08 — quota global atômica no enqueue Redis
  (OLLAMA_AGENT_REDIS_MAX_JOBS, default 100k). Teste de integração verde no CI
  com Redis real. (Sweeper de órfãos H-09 e quota por-tenant seguem como
  follow-up menor.)
- `e5bb698a` M-15 — ledger de migrations Postgres (agent_schema_migrations
  append-only, provenance + drift por checksum). Checksum testado puro; SQL
  exercido pelo job Postgres do CI.

Estado de código: os findings P1/P2 acionáveis desta auditoria estão fechados.
Permanecem APENAS os bloqueios externos (fora do código): assinatura
Authenticode/SignPath, attestation/provenance SLSA, staging TLS/mTLS real,
certificação RLS em produção, pentest externo, runners macOS/MLX e mobile
físico/EAS. Esses exigem credenciais/infra/decisão do usuário e não são
simuláveis.


## Checkpoint — backlog de código ENCERRADO (sem certificado) — 2026-10-03

Tip `57718598`, ambos os workflows verdes (inclusive o job de integração com
Redis e Postgres reais).

Além dos 4 findings já fechados, esta rodada adicionou:
- `afc94f87`→`57718598` H-09 (fila Redis, órfãos): a tentativa de auto-sanear
  órfãos no claim foi REVERTIDA porque violava uma invariante INTENCIONAL
  (TestDistributedRedisRejectsOrphanAndMismatchedJobRecords): um id pendente sem
  job é corrupção e deve ser REJEITADO+PRESERVADO (fail-closed), não consumido.
  Comportamento original restaurado. Isto NÃO é um gap — é o design correto.
- `ea68fceb` quota POR-TENANT na fila Redis
  (OLLAMA_AGENT_REDIS_MAX_JOBS_PER_TENANT, default 10k), com índice por tenant
  (SET) auto-saneável no enqueue (poda membros ausentes/terminais antes de
  contar) — só o script de enqueue muda. Teste de integração verde no CI.

Estado de código: **todos os findings P1/P2 acionáveis desta auditoria estão
fechados e validados no CI** (Redis/Postgres reais). Não há mais itens de código
pendentes do backlog.

Permanece APENAS o que é externo (e o usuário pediu para seguir SEM o
certificado): assinatura Authenticode/SignPath, attestation/provenance SLSA,
staging TLS/mTLS real, certificação RLS em produção e pentest externo. Esses
exigem credenciais/infra/decisão e um auditor independente — não são código e
não são simuláveis. O scaffolding de assinatura/attestation nos workflows já
existe e ativa quando os segredos forem fornecidos.


## Checkpoint — avanços SEM certificado (usabilidade + release) — 2026-10-03

A pedido do usuário ("continue sem o certificado"), fechei melhorias de CÓDIGO
que não dependem de assinatura. Tip `6fd58e65`, dz23-agentic-quality verde
(inclusive Web/mobile quality e integração Redis/Postgres).

Usabilidade leigo (pt-BR + a11y):
- `80844621` Onboarding inteiro em pt-BR + branding Hades (V-05), 39 testes.
- `ed94116c` ChatForm (placeholder + aria-label do anexar), ErrorMessage
  ("Erro"), CodexConnectedIntro, CodexDesktopRow (todo o fluxo ChatGPT),
  Codex/Claude DesktopModelsSettings ("Buscar modelo…"); testes atualizados.
- `1822e2a2` Mobile App.tsx em pt-BR (Missões/Linha do tempo/Aprovação) +
  accessibilityRole nos botões restantes.

Robustez de release SEM certificado:
- `6fd58e65` verify-release-artifact.sh ancora fingerprint da chave pública
  (OLLAMA_RELEASE_PUBKEY_SHA256) e a identidade repo/commit/ref via
  release-metadata.json coberto pelo manifesto assinado; novo teste de shell
  (5/5) e doc em RELEASE_READINESS.md. Resolve a confiança auto-referente sem
  precisar de Authenticode/attestation (que seguem como gate externo).

Resíduo conhecido (não feito): publicar SHA256SUMS + SBOM também em builds
unsigned ([13], mudança de workflow de médio risco) permanece como follow-up.
Assinatura/attestation/staging TLS/pentest continuam externos.
