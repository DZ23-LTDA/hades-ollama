# Handoff de continuidade — Ollama Full

**Atualizado:** 2026-09-29 07:51 (-03). Para o estado remoto mais recente, consulte a seção incremental no final deste documento; os hashes e estados anteriores abaixo são históricos.

**Propósito:** permitir retomar o trabalho depois de encerrar/formatar este PC ou trocar de sessão, sem depender do histórico da conversa.

## Estado remoto confirmado

- **Projeto:** [DZ23-LTDA/ollama-classe-a-plus](https://github.com/DZ23-LTDA/ollama-classe-a-plus)
- **Branch de continuidade:** [`recovery/ollama-full-snapshot`](https://github.com/DZ23-LTDA/ollama-classe-a-plus/tree/recovery/ollama-full-snapshot)
- **Commit fixo do snapshot de código:** `8c2e3604a086abeab401e2c80e96fe6ad4473566` — `security: harden tenant isolation and workspace boundaries` (134 arquivos; 22.536 inserções, 1.582 remoções).
- A branch contém commits documentais posteriores. **Sua ponta é móvel:** consulte-a no checkout/remote antes de retomar, em vez de tratar um hash escrito aqui como atual:
  ```bash
  git ls-remote --heads origin refs/heads/recovery/ollama-full-snapshot
  git ls-remote --heads origin refs/heads/main
  ```
- **`main` verificada antes desta atualização documental:** `8635e30dc9e95a1f5b29700169783abc24093ceb`; não foi alterada. Não houve force-push nem PR.
- O checkout estava limpo após cada publicação; confirme novamente que a branch local acompanha `origin/recovery/ollama-full-snapshot`.

### Restaurar em outro PC

```bash
git clone --branch recovery/ollama-full-snapshot \
  https://github.com/DZ23-LTDA/ollama-classe-a-plus.git
cd ollama-classe-a-plus
```

Para continuar num clone já existente:

```bash
git fetch origin
git switch --track origin/recovery/ollama-full-snapshot
git status --short --branch
```

## Documentos que devem ser lidos primeiro

1. `audit/CLAUDE_CODEX_RESUME_PROMPT_20260927.md` — prompt copiável e sequência inicial para Claude Code/Codex.
2. `audit/OLLAMA_FULL_HANDOFF_20260927.md` — este handoff com recuperação, branch e próximos passos.
3. `audit/OLLAMA_FULL_MISSION_STATE.md` — histórico durável detalhado e evidências por rodada.
4. `audit/FINAL_SECURITY_REAUDIT_POSTFIX_20260927.md` — achados independentes e status das remediações.
5. `docs/agentic/PARITY_MATRIX.md` — paridade por domínio, estados e gates faltantes; não tratar adapters como recursos completos.
6. `docs/OLLAMA_FULL_PRODUCT_VISION.md` e `audit/COMPARACAO_ARVORES_MANUS_OLLAMA_FULL.md` — visão de produto e comparação de escopo.
7. `SECURITY.md` e `docs/RELEASE_READINESS.md` — regras de segurança e critérios de liberação.

A captura do estado atual da UI está em `docs/images/screens/ollama-full-current-ui-2026-09-27.png` (1440 × 900). A captura foi feita contra localhost; na captura final o backend estava com `OLLAMA_NO_CLOUD=1` e o navegador bloqueou tráfego não local. Os serviços temporários foram desligados.

## Objetivo e critério de conclusão

Continuar evoluindo o produto para uma experiência Ollama Full inspirada em capacidades observáveis de Manus Desktop e outros harnesses, sem afirmar acesso a internals proprietários e sem sacrificar isolamento por tenant. **A missão de produto/paridade permanece incompleta.** Um gate verde prova apenas os comandos executados, não paridade funcional ou prontidão enterprise.

Critério mínimo antes de declarar pronto para produção: resolver a arquitetura PostgreSQL sem contexto tenant forjável, reauditar independentemente as correções da última rodada, verificar os fluxos end-to-end que hoje são adapters/parciais, executar os gates depois da última alteração e registrar limitações reais por plataforma/provider.

## Trabalho integrado no snapshot

O commit de segurança reuniu mudanças acumuladas de várias rodadas, incluindo:

- isolamento por organização em stores, jobs, schedules, plugins/Connectors/MCP e rotas autenticadas; lifecycle raw não pode alterar recursos tenant-owned em modo local/sem autenticação;
- autenticação/approval e fingerprints de ferramenta ligados à configuração efetiva; approvals inconsistentes falham fechado;
- sandbox Linux ancorado ao workspace por descritor, ambiente restrito, modo de isolamento hash-bound à aprovação, chroot com filesystem mínimo e sem `/proc`; fora das plataformas suportadas, certas execuções falham fechado em vez de usar caminho inseguro;
- Git de leitura usa diretório privado de metadados, construído de forma limitada a partir de handles; config, hooks e filtros do repositório original não são expostos ao subprocesso; configuração/árvore `.git` é revalidada ao redor das operações;
- snapshots, workspace writes/patches, artifacts, media/OCR e mídia aprovada usam handles `os.Root`/identidade/hash onde implementado, com regressões de substituição de path e metadados;
- egress/DNS/IP pinning, validação JSON e redação DLP em Connectors, providers e MCP; atenção especial a campos/URLs que podem conter assinaturas;
- quotas de contagem/bytes/profundidade e concorrência em JSONStore, traces, push subscriptions, queues/outbox e uploads; recuperação de handlers e fencing de leases;
- auditorias, visão, matriz de paridade e esta documentação de continuidade.

As descrições acima indicam o trabalho contido no snapshot; para a fonte autoritativa de limitações e evidências, use a matriz e o relatório de reauditoria.

## Bloqueadores e próximos passos prioritários

### P0 — PostgreSQL RLS / produção

A antiga arquitetura GUC-forjável foi substituída no checkout local por candidata HMAC tenant-context, role/DSN separados (migrator e runtime), migração explícita fora do startup, runtime `NOSUPERUSER`/`NOBYPASSRLS` sem ownership/DDL, key table ilegível pelo runtime, e políticas RLS que aceitam somente assinatura conferida no servidor PostgreSQL. Eventos têm ownership composto com a missão. O runtime exige Redis owner-bound e recupera trabalho enumerando organizações trusted no AuthStore; `WorkspaceIdentity` é persistida e missões executáveis legadas sem identidade são recusadas.

**Validação local real:** PostgreSQL 16 + Redis 7 disposable, com credenciais separadas, passou tentativas GUC/HMAC inválido, leitura da signing key, `SET ROLE`, DDL, `row_security=off`, cross-tenant read/write e ligação de evento a missão alheia; cobriu key mismatch/imutabilidade, refusal de ownerless rows e workspace identities ausentes, e recuperação por tenant. O procedimento `migrate-existing-roles.sql` também foi aplicado a um volume legado simulado, transferindo ownership, removendo privilégios e sessões residuais, e deixando a role antiga `NOLOGIN`/não-superuser; em seguida a integração adversarial passou.

**Ainda não é produção validada:** revisão independente final desta exata revisão está pendente; faltam staging real com TLS/HA, backup/restore, operação/rotação coordenada de chave, crash/failover e escala do sweep de recovery. Não declarar isolamento enterprise ou prontidão de produção até concluir esses gates.

### P1 — Reauditoria dos findings remediados

O relatório `FINAL_SECURITY_REAUDIT_POSTFIX_20260927.md` preserva findings originais e as atualizações pós-remediação. Uma revisão independente focada concluída em 2026-09-29 não encontrou findings acionáveis nos pontos tardios de URL/cancelamento de upload, lock/limpeza do PushOutbox e escopo local de aprovação de deployment. Isso não substitui uma reauditoria integral de todas as superfícies. Faça novas revisões adversariais conforme as áreas forem alteradas; execute os testes afetados; registre evidências e qualquer finding novo antes de encerrar.

### P2 — Paridade funcional e plataforma

A matriz continua classificando vários domínios como `PARCIAL` ou `ADAPTER IMPLEMENTADO`. Entre os itens explícitos: jornada E2E longa de missões; estados reais, acessibilidade e responsividade de cada tela; smoke autorizado de providers externos/OAuth; ferramentas reais de browser/desktop; ciclo coding com worktree/branch, execução de testes segura, repair loop e aplicação de mudanças; colaboração CRDT; mobile em dispositivos; deploy com contas reais e rollback; multimídia com modelos/GPU/Tesseract; staging de queues/observabilidade. Priorize uma jornada completa end-to-end por vez e atualize a matriz com implementação, teste automático e execução observável.

### P3 — Integridade/limites restantes

Revisar o relatório e a matriz quanto a falhas de crash/failover, semântica at-least-once, Redis Cluster não suportado, fallbacks não-Linux que devem permanecer fail-closed, boundedness em todos os formatos/callsites, logs/metrics, e execução nativa Windows/macOS. Não transformar teste cross-compile em alegação de teste nativo.

## Gates comprovados

Nesta retomada de 2026-09-29, após a alteração mais recente de migração e teste:

- `go test -tags=integration ./internal/agent -run '^TestDistributed' -count=1` — PASS com PostgreSQL 16/Redis 7 reais;
- `go test -p=2 ./... -count=1` — PASS;
- `go vet -p=2 ./...` — PASS;
- `go test -race -p=2 ./internal/agent ./server -count=1` — PASS;
- `CGO_ENABLED=1 go build -p=2 ./...` — PASS;
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -run '^$' -c -p=2 -o /tmp/ollama-agent-latest-windows.test.exe ./internal/agent` — PASS (cross-compile, não execução nativa);
- `bash scripts/check-class-a-plus-integrity.sh`, gofmt e `git diff --check` — PASS;
- Compose e CI YAML parse — PASS;
- gitleaks redacted nas fontes de produção/deploy alteradas — zero findings;
- upgrade de volume PostgreSQL legado simulado — PASS; verificação observou owners migrator, zero sessão legada ativa e role antiga sem login/superuser.

A auditoria independente final do delta está em andamento; não criar/publicar commit enquanto não registrar o veredito/fixes. Antes de merge/release, repetir os gates em checkout limpo e completar staging. O branch local continua `recovery/ollama-full-snapshot`; `HEAD` verificado no início desta rodada foi `cdd91ceb` e há alterações locais ainda não commitadas.

## Regras operacionais ao retomar

- Confirmar `git status`, branch, `HEAD`, `git remote -v` e referências antes de editar. Preservar as alterações; não usar `reset --hard`, `clean -fd` nem checkout destrutivo.
- Trabalhar em `recovery/ollama-full-snapshot`. Não enviar para `main`, não force-push e não reescrever histórico.
- O pedido do usuário é manter o trabalho do projeto no GitHub para recuperação; commits incrementais relevantes podem ser enviados **a essa branch** depois de validação, sem duplicar o snapshot. Merge/PR/release/main continuam fora do escopo sem autorização específica.
- Não configurar sincronização automática ou cron sem pedido explícito. “Manter atualizado” até aqui significa atualizar a branch conforme cada rodada verificada.
- Não publicar secrets, tokens ou dados locais; gitleaks e inspeção de `git status` antes do push.
- Diferenciar fato testado de hipótese, adapter, mock ou dependência externa; não alegar “paridade total” ou “100% finalizado”.

## Comandos úteis

```bash
# estado atual e documentação
pwd
git status --short --branch
git log --oneline -5
sed -n '1,240p' audit/OLLAMA_FULL_HANDOFF_20260927.md
sed -n '1,240p' audit/FINAL_SECURITY_REAUDIT_POSTFIX_20260927.md

# testes e gates (depois das alterações)
go test -p=2 ./... -count=1
go vet -p=2 ./...
go test -race -p=2 ./internal/agent ./server -count=1
CGO_ENABLED=1 go build -p=2 ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -p=2 -o /tmp/ollama-agent-windows.test.exe ./internal/agent
bash scripts/check-class-a-plus-integrity.sh
git diff --check
```

**Estado registrado em 2026-09-29 01:40 -03:** commit `fdd9abfa30d583be9b69d76f6b3a133c492ab07d` da branch `recovery/ollama-full-snapshot` foi enviado e verificado pela API GitHub e por `git ls-remote`. A ref `main` permanece `8635e30dc9e95a1f5b29700169783abc24093ceb`. O serviço PostgreSQL temporário foi encerrado e o diretório de dados descartável removido. Revalide refs e estado do checkout antes de continuar.


## Atualização obrigatória de continuidade — 2026-09-29 07:51 -03

**Fonte da verdade:** `origin/recovery/ollama-full-snapshot` foi confirmado em `8cd2a71c8377cac46b8dd29642fe3ab38619ebff`. Commits desta retomada: `4bcc6ef2ff33c9e9c0b1ed09de1f7d9f1040d023` (arquitetura RLS candidata e remediações), `00314480e6fbb21f3b99755a1db19e143e33690e` (regressão para sessão legacy privilegiada), `8cd2a71c8377cac46b8dd29642fe3ab38619ebff` (remoção do dump de teste). Somente a branch de recovery foi atualizada; sem force-push/main/PR. O RDB de teste foi removido do tree; não incluir arquivos locais em próximos commits.

**Resultado recente:** runtime PostgreSQL agora falha fechado tanto se a role antiga `ollama_agent` ainda for superuser/LOGIN quanto se houver sessão antiga aberta mesmo após a role ser demovida. O teste real `TestDistributedPostgresRuntimeReadinessForStaleLegacySession` e o grupo `go test -tags integration ./internal/agent -run '^TestDistributed' -count=1` passaram em fixture descartável PostgreSQL 16 + Redis 7. `go test -p=2 ./internal/agent ./server ./cmd -count=1`, integrity, formatação, YAML e gitleaks do delta anterior passaram; repetir full suite/vet/race/build/cross-build sobre o HEAD atual. O build do test binary Windows para `cmd` foi tentado, mas falha por constraints do MLX/upstream; isso não é evidência de execução nativa nem sucesso desse target.

**Estado e limitações:** PostgreSQL/RLS é uma implementação candidata, não aprovada para produção. HMAC key rotation ainda não existe; TLS/HA, staging real, rollback de cutover, crash/failover, escala, multi-instância e auditoria independente final estão pendentes. O último trio de auditores do delta mais recente encerrou como failed/sem veredito; os reviews anteriores tiveram achados e os fixes foram implementados, mas não afirmar “auditoria final aprovada”. Feature parity com Manus Desktop/harnesses também permanece incompleta, conforme `docs/agentic/PARITY_MATRIX.md`.

**Próximo passo exato:** no checkout `/home/ubuntu/ollama-full-recovery`, confirmar branch/ref/status; executar full Go tests, vet, race, build, cross-compile Windows agent, guard de integridade, parser YAML, `git diff --check` e Gitleaks para o estado atual; obter revisão read-only curta do delta; só então atualizar checkpoint incremental, commit e push **somente** em `recovery/ollama-full-snapshot`. A matriz e o runbook devem manter o gate de produção bloqueado até os itens acima terem evidência.


## Atualização obrigatória de continuidade — 2026-09-29 09:07 -03

**Último commit enviado:** `43daf0e22fd81bb8977c79abf94ca9aa6a25519c` (`fix(postgres): close RLS drift and legacy cutover gaps`), `origin/recovery/ollama-full-snapshot`. Hash local/remoto conferidos iguais; `main` não foi alterada.

**Concluído e evidenciado nesta rodada:** testes distribuídos PostgreSQL 16 + Redis 7 passaram; casos separados confirmaram recusa do login legado ativo, recusa de sessão superuser antiga ainda conectada depois de demotion e retirement cluster-wide com sessão held no database `postgres` terminada. RLS drift suite cobre policy/verifier, RLS flags, grants extras (TRUNCATE/REFERENCES/TRIGGER/TEMP), catálogo temporário e serialização migration/runtime; os testes restauram a forma aprovada e exigem o sentinel de segurança. Migração foi ajustada para manter `pg_catalog` primeiro e usar nomes `public.*` explícitos. Integridade, gofmt, `git diff --check`, YAML e Gitleaks redacted das mudanças passaram sem achados.

**Remediações de auditoria incluídas:** sessão em qualquer database do cluster entra no drain; contract das roles substitutas e role antiga é verificado; reconnect CI exige sucesso antes e falha pós-NOLOGIN; sanitizer inclui senha admin e todos os DSNs; runbook legacy tem fluxo único e comandos definidos, backup/restore rehearsal e senha do CLI migrator via `PGPASSWORD`; release-readiness seleciona check-run mais novo por ID; skills sem owner só aparecem no contexto local.

**Ainda pela metade / limitações:** falta rodar novamente no commit atual `go test -p=2 ./... -count=1`, `go vet -p=2 ./...`, race em agent/server, `CGO_ENABLED=1 go build -p=2 ./...`, Windows cross-compile suportado do pacote agent e revisar os checks do GitHub para o mesmo SHA. Falta auditoria read-only independente do delta exato pós-fix. Não alegar aprovação para produção: staging TLS/HA, rotação transacional da HMAC, failover, scale/multi-instância e evidência operacional de rollback/restore continuam abertas. O Windows agent cross-compile não atesta CLI Windows nativo; Manus/harness parity segue incompleta.

**Próximo passo exato:** executar os gates completos no HEAD `43daf0e22fd81bb8977c79abf94ca9aa6a25519c`, observar Actions desse mesmo commit e fazer review read-only independente do delta. Registrar resultados reais em ambos os checkpoints e publicar quaisquer correções apenas em `recovery/ollama-full-snapshot` (sem force-push/main).


## Continuidade — cutover/schema RLS corrigidos e enviados — 2026-09-29 10:07 -03

Commit funcional publicado: `0aaa65548d4e51632111fe23d95ea2cb1970a6ac` em `recovery/ollama-full-snapshot` (ref remota verificada). Não alterar `main`, não force-push/rebase.

O delta fecha hardening de `search_path`, memberships em ambas direções, inventário/transferência de objetos de role legado e recusa explícita de owners em outro database/objetos não suportados. A validação do schema agora cobre tipos, nullability, defaults, PK, índice tenant composto e FK validada; testes verificam drift de NULL/type/default. Test fixtures são apagadas e confirmadas ausentes por `t.Cleanup`, e a fila Redis de recovery usa namespace exclusivo. CI recebeu fixtures com funções-sombra, membership reverso e objetos legacy composite/domain/enum/function/large object. O runbook exige staging em cluster separado (system identifier), senhas distintas e cleanup do restore com `trap`.

Evidência contra instância descartável PostgreSQL 16 + Redis: ambos os scripts passaram com `search_path` hostil; memberships do legacy chegaram a zero; os cinco objetos legacy de categorias suportadas terminaram pertencendo ao migrator. `CGO_ENABLED=0 go test -tags integration ./internal/agent -run '^TestDistributed' -count=1` passou em 4,03 s; testes específicos de readiness legacy também passaram. Após o suite, objetivos/rows de teste rastreados = 0, Redis recovery keys = 0. Serviços foram parados e `dump.rdb` local removido. Também passaram `go test ./internal/agent -count=1`, `go vet ./internal/agent`, gofmt, `git diff --check` e `bash -n` do snippet de restore. `actionlint` e parser YAML local não estavam disponíveis; nenhum staging operacional ocorreu.

Ainda pendentes: protocolo e implementação de rotação HMAC versionada/transacional com rollback e testes de crash/concorrência; staging TLS/HA/restore; gates globais de Go, CI do SHA e review independente final. **Não declarar produção pronta nem paridade Manus concluída.**

Próxima ação exata: projetar e implementar key rotation com estados versionados (chave atual/próxima), quiescência, atualização atômica do segredo e verificação de todos os writers/readers, rollback fail-closed e teste adversarial contra PostgreSQL 16. Depois rodar full tests/vet/race/build/cross-compile suportado, conferir Actions do SHA e obter re-review read-only. A cada rodada verificada: `git add -A`, commit, push apenas em `origin/recovery/ollama-full-snapshot`, atualizar este arquivo e `audit/OLLAMA_FULL_MISSION_STATE.md` com os hashes exatos e push dos checkpoints também.


## Continuidade — rotação HMAC versionada e cutover PostgreSQL — 2026-09-29 12:01 -03

**Commit funcional enviado:** `1bd884144c173ba8d97b7cac36000db1e7fe767c` (`feat(postgres): implement versioned tenant key rotation`) na branch `recovery/ollama-full-snapshot`; hash local/remoto conferidos iguais. Não alterar `main`, não usar force-push/rebase e não abrir PR sem autorização.

Implementada rotação transacional da chave HMAC tenant por versão. Chave/version são incluídas no transcript; o keyring é restrito ao migrator. Migration importa a chave singleton como v1 somente durante transição legada e falha fechada para keyring corrompido/zero ativo/versão-segredo divergente. CLI `ollama agent rotate-postgres-key` exige admin + migrator DSNs por ambiente, valida papéis e mesmo cluster/database, verifica a chave ativa, serializa tentativas, coloca runtime em NOLOGIN, recusa sessões residuais, drena transações via lock, troca versão de forma atômica e reabilita LOGIN somente após commit. Não existe janela de dual-key: parar writers/workers e reiniciar runtime com a chave nova é obrigatório. O admin bootstrap instala pgcrypto; legacy cutover transfere ownership via REASSIGN OWNED.

**Evidência verificada:** num cluster PostgreSQL 16 descartável isolado, cutover phase 1 + migration Go + phase 2 passaram sob `search_path` hostil. Estado observado após cutover: pgcrypto owner=ollama_agent_migrator; legacy role com LOGIN/SUPERUSER/CREATEDB/INHERIT todos false; memberships=0. Suíte adversarial PostgreSQL/Redis descrita no histórico inclui rotação concorrente, fencing, stale runtime, zero-active corruption, bootstrap/import v1, high-water não reutilizável e drift do índice ativo. `go test ./...` inicialmente apontou que um teste esperava chave ausente sem limpar variável externa; teste corrigido com `t.Setenv(..., "")`, depois full suite PASS. `go vet ./...`, gofmt, `git diff --check` e `actionlint` PASS. API GitHub `check-runs` retornou vazio para a SHA; não assumir CI remoto verde. Não foram detectadas credenciais literais nos diffs.

**Review independente:** ainda em execução quando este checkpoint foi escrito; obter o resultado e corrigir qualquer finding antes de afirmar aprovação de segurança.

**Ainda pendente:** staging operacional com TLS/HA, backup e restore/rollback, crash/failover, rollout coordenado e revisão independente final. A implementação é candidata; não afirmar PostgreSQL production-ready, paridade total do Manus ou conclusão do produto.

**Próxima ação exata:** recuperar/concluir o job read-only de review da rotação/cutover; reconciliar findings e executar testes afetados. Atualizar ambos checkpoints com o veredito e novo hash; consultar GitHub Actions para `1bd884144c173ba8d97b7cac36000db1e7fe767c` (até agora sem check-runs). Depois planejar staging TLS/HA/restore/failover com ambientes e DSNs explicitamente separados. Toda nova rodada verificada: `git add -A`, commit, push apenas para `recovery/ollama-full-snapshot`, atualizar ambos checkpoints e publicar o SHA; nunca force-push.


## Continuidade — ACLs PUBLIC e guia de rotação corrigidos — 2026-09-29 12:14 -03

**Commit funcional enviado:** `b7822ed569e0a32e3cabd1c7c5949512b70649bd` (`fix(postgres): close inherited public ACL privilege paths`) na branch `recovery/ollama-full-snapshot`, ref local/remota conferida. Checkpoint docs anterior: `88ebfc56adc4ec5017e66401e92d4f9ecd470bf6`. Não alterar `main`, não fazer force-push/rebase.

Uma review independente da rotação/cutover apontou finding Major: ACLs `PUBLIC` em objetos públicos legados podiam manter visíveis tabelas, sequences ou rotinas `SECURITY DEFINER`, apesar de ownership transferido; também achou `docs/CLASS_A_PLUS_GUIDE.md` desatualizado sobre key rotation. O fix revoga grants de `PUBLIC` e runtime em todas as relations/sequences/routines públicas na fase de preparação e no baseline de migration; libera somente `pgcrypto.hmac` para o migrator; configura default ACLs globais e de `public`; e exige attestation negativa de todos os privilégios efetivos não allowlisted, incluindo objetos/schemas públicos extras e function defaults. O runtime continua com apenas DML nas tabelas tenant e EXECUTE no verifier SECURITY DEFINER. Guia principal removido de afirmações obsoletas e aponta para `ollama agent rotate-postgres-key`, versão monotônica e rollback por versão superior.

Regressões implementadas: objeto tabela com dado sentinela e `GRANT SELECT TO PUBLIC`, sequence com grants PUBLIC e função `SECURITY DEFINER` foram injetados; runtime e pool já aberto precisam recusar; migration limpa a exposição e runtime deve recuperar. Outro caso testa default function ACL inseguro. O fixture de CI da migração legacy também cria os mesmos objetos antes do cutover e confirma `has_table_privilege`, `has_sequence_privilege` e `has_function_privilege` falsos após migration. Teste CLI protege a consistência do guia.

**Testes PASS verificados:** `CGO_ENABLED=0 go test -tags integration ./internal/agent -run '^TestDistributed' -count=1 -v` contra PostgreSQL 16/Redis; subtestes PUBLIC table/sequence/function e default ACL passaram. `go test ./... -count=1`, `go vet ./...`, gofmt, `git diff --check` e `bash -n` do shell PostgreSQL CI passaram. Parser YAML Python/actionlint indisponíveis localmente; requer check remoto. PostgreSQL/Redis de teste desligados depois da suíte.

**Review independente da remediação:** job `job_YfyrxXaB` ainda estava em andamento quando este checkpoint foi gravado. A review anterior solicitou mudanças; não declarar auditoria limpa até obter e reconciliar o novo resultado.

**Próxima ação concreta:** recuperar o resultado do reviewer `job_YfyrxXaB`, corrigir findings concretos; ensaiar phase 1, migration Go e phase 2 em cluster PG16 disposable com ACLs PUBLIC legadas e verificar owner/role/membership/sem acesso; checar Actions para `b7822ed`. Se novo delta verificado ocorrer: atualizar Mission State + Handoff, commit e push nessa mesma branch. PostgreSQL/RLS permanece candidato, não production-ready; staging TLS/HA/restore/failover e rollout coordenado continuam fora da evidência local.


## Continuidade — re-review ACL sem blockers e defaults ampliados — 2026-09-29 12:24 -03

**Último commit de código enviado:** `6cedbb1e84d035e9010cafeba3dd576d297dc526` (`test(postgres): cover all unsafe default ACL classes`), `origin/recovery/ollama-full-snapshot`. Correção ACL principal: `b7822ed569e0a32e3cabd1c7c5949512b70649bd`; checkpoint anterior: `9af550297a11c3d29cec6e01e04f83e05535bf84`. Local/remoto sincronizados; não tocar `main`, sem force-push/rebase.

**Review independente da fatia ACL/HMAC:** veredito explícito sem bloqueadores críticos/major e sem alterações ACL/HMAC requeridas antes do merge. O reviewer recomendou ampliar teste de default ACL de functions para tabelas e sequences e assertar o catálogo. Implementado: teste agora injeta grants PUBLIC/runtime globais e de schema para functions/tables/sequences, confirma recusa do runtime, executa migration, exige `pg_default_acl`/`aclexplode` sem entradas inseguras e confirma recuperação runtime. CI confirma os owners legados de tabela, sequence e SECURITY DEFINER passam ao migrator e que runtime não herda SELECT/USAGE/EXECUTE. O guia principal não diz mais que rotação é impossível; aponta ao comando/versionamento. Isso valida a fatia ACL/HMAC; não é aprovação de produção.

**Evidência final:** em cluster PostgreSQL 16 descartável, com objeto legado `PUBLIC` (tabela com marcador secreto, sequence e SECURITY DEFINER) e search_path hostil, phase 1 do cutover + migration Go + phase 2 completaram. A query final confirmou role antiga desabilitada sem memberships em ambas direções, pgcrypto e os três objetos legados sob `ollama_agent_migrator`, runtime sem os três acessos efetivos. Teste integrado `TestDistributedPostgresRuntimeRejectsRLSDrift` passou com a cobertura de ACL/defaults. Também passaram `go test ./... -count=1`, `go vet ./...`, `CGO_ENABLED=0 go test -tags integration ./internal/agent -run '^TestDistributed' -count=1 -v` em PostgreSQL 16 + Redis, YAML `gopkg.in/yaml.v3`, `bash -n` do bloco CI e `git diff --check`. Serviços/clusters descartáveis foram desligados/removidos.

**Remote CI:** API de check-runs e `gh run list` não retornaram checks para `6cedbb1`. O workflow está configurado para pull_request e push apenas a main, não para push desta branch; não afirmar GitHub CI verde, e nenhum PR foi criado. A branch recovery está limpa e remota alinhada.

**Próxima ação concreta:** manter PostgreSQL/RLS como candidato. Para solicitar produção, preparar staging separado autorizado, TLS/CA, janela de manutenção e backup; executar restore/rollback, failover/crash e smoke multi-instância com evidência independente. Só promover após esses gates reais, CI GitHub associado ao PR autorizado e revisão do SHA final. A paridade Ollama Full/Manus também continua incompleta; esta rodada concluiu somente o hardening ACL/HMAC solicitado.


## Continuidade — restore verify-full em clusters PG16 distintos — 2026-09-29 12:31 -03

**Commit funcional enviado:** `1bf058b769395861e308fa986f7baf040e45bc46` (`docs(postgres): require verified TLS in restore rehearsal`), `origin/recovery/ollama-full-snapshot`; local/remoto confirmadas. Branch recovery apenas; nenhum push em `main` ou force-push.

O runbook source/staging agora usa `sslmode=verify-full` e CA distinta/explicitada em ambos DSNs e no `pg_restore`; deixa os DSNs do admin de origem separados do staging maintenance/restore, exige source/staging cluster IDs diferentes e informa pré-requisitos `CREATEDB` + `pg_monitor`. `scripts/check-class-a-plus-integrity.sh` impede regressão desses requisitos; removidos três comandos duplicados no fixture CI.

**Evidência local:** ensaio efêmero de PostgreSQL 16 com dois clusters independentes, dois certs TLS SAN IP/DNS, `psql`/`pg_dump`/`pg_restore` em verify-full; system identifiers diferentes; dump custom restaurado; contagens finais `1 mission / 1 event`. Segundo ensaio injetou falha logo após criar a database de staging: trap removeu database temporária, clusters, diretório CA e dump. Nenhum dado persistente afetado.

**Gates PASS:** `go test ./... -count=1`, `go vet ./...`, integrity guard, YAML parser Go yaml.v3, Bash syntax do bloco de integração CI e do snippet de restore, `git diff --check`. A evidência adversarial de cutover/ACL em PostgreSQL 16 e review sem blockers permanece no estado anterior. GitHub Actions não executou checks para branch recovery; workflow está limitado a pull_request/push main; nenhum PR foi aberto.

**Ainda não production-ready:** o rehearsal é local/descartável. Falta ambiente real staging, TLS/CA do operador, HA/replicação, failover/crash, backup/retention/RPO-RTO e restore/rollback operacional independente, rollout coordenado e alertas. A execução dessas fases externas depende de staging/DSNs, CA e agenda operacional autorizados. Enquanto faltarem, status do PostgreSQL é candidato e gate de produção continua fechado.

**Próximo passo:** preparar e executar staging HA com os valores reais, exercitar replicação/failover/restore/rollback e obter revisão independente. Em paralelo, manter branch recovery sincronizada e executar os checks GitHub por rota autorizada; não alegar CI remoto verde sem run/check ligado ao SHA.


## Continuidade — streaming TLS e promoção PostgreSQL 16 comprovados localmente — 2026-09-29 12:42 -03

**Commit funcional enviado:** `9dc80fa16619704b49d30c2fcf0e2833b403b929` (`docs(postgres): record tested fenced failover rehearsal`) em `origin/recovery/ollama-full-snapshot`, remote/local iguais.

Adicionado `docs/agentic/POSTGRES_HA_FAILOVER_REHEARSAL.md` e link no guia principal. A sequência local usa primary/standby PostgreSQL 16 descartáveis, TLS `verify-full`, `pg_basebackup`/WAL streaming, espera por `pg_stat_replication.state='streaming'`, marcador replayado, fencing (parada do primary) antes de promoção e write após promoção. Resultado observado: 2 linhas replayadas antes da promoção; `pg_is_in_recovery()=false`; total pós-promoção 3 com 1 write novo. Fault injection após promoção confirmou remoção do standby promovido, cluster primary de teste, portas e diretório temporário. O integrity guard protege a presença do guia e avisos split-brain/fencing.

Gates: `go test -json ./... -count=1` status 0; `go vet ./...`; integrity; parser yaml.v3; Bash syntax dos snippets restore/HA; `git diff --check`; secret pattern scan e verificação de ausência de cluster/porta local. A rodada anterior já ensaiou restore TLS/verify-full entre clusters diferentes e cleanup em falha.

A evidência é apenas de mecânica manual em sandbox; não representa HA gerenciada, produção, RPO/RTO, topologia/quorum, comportamento sob partição, failover automatizado, recuperação segura do old-primary/`pg_rewind`, certificados reais ou retenção de backup. PostgreSQL permanece bloqueado para produção até staging autorizado e review operacional. GitHub Actions não executou neste SHA; não há PR nem check-run a relatar.

**Próximo passo:** acordar e preparar staging real com CA, DSNs e topologia documentados; definir modo de replicação/fencing e metas RPO/RTO, então executar cenários de falha e restore/rollback com evidência independente. Não declarar produção aprovada só pelos ensaios descartáveis locais.


## Atualização — referências oficiais PostgreSQL 16 — 2026-09-29 12:43 -03

O guia `POSTGRES_HA_FAILOVER_REHEARSAL.md` agora cita as páginas oficiais PG16 para failover/fencing do old primary, `pg_basebackup` e replication slots, libpq TLS `verify-full`/CA e `pg_promote` com timeout. A orientação de prevenir split-brain está explicitamente alinhada ao manual do PostgreSQL, que exige impedir que o primary anterior retorne como escritor.

Último commit funcional: `31ce00bc0f8ac3e534170cb84b08abd43815c64c` (`docs(postgres): cite pg16 failover guarantees`), confirmado igual em `origin/recovery/ollama-full-snapshot`. Guard de integridade, YAML, `bash -n`, diff e limpeza de listeners/temporários passaram; os testes Go e vet passaram na rodada anterior a esta alteração exclusivamente documental.

Status mantém o gate de produção **BLOCKED**: sem HA operacional em staging autorizado, quorum/fencing, partição de rede, backup/restore operacional com RPO/RTO medidos, recuperação do antigo primary, alertas e review independente. Não afirmar aprovação de produção ou check GitHub remoto sem evidência associada ao SHA.


## Continuidade — imagens de referência e review production blockers — 2026-09-29 13:13 -03

Estado conferido na branch `recovery/ollama-full-snapshot`, HEAD/remote `00ee89d406578a7ab3b3bca5889bd7d3e5e5ee9d`. Entrega visual fora do Git: `/home/ubuntu/ollama-full-reference-images.zip`, 13 PNGs 2560×1440 cobrindo todas as rotas principais, contato, README, CSV; unzip e PNG integrity passaram. SHA-256 do zip: `fd988e8213643efc3a337b18e48878714c40d8a9befcc152ac240b703614c713`. São conceitos visuais futuros, não capturas nem prova de implementação.

Review read-only independente `job_NoiyZDTQ`: zero Critical; Request Changes para aprovação de produção devido a três Major: (1) schema validation aceita colunas extras embora runtime tenha DML de tabela inteira; (2) startup não audita privilégios por coluna (`pg_attribute.attacl`/`has_column_privilege`), inclusive para secrets/keyring; (3) fresh init/startup não define nem atesta completamente `PUBLIC CONNECT` no database e USAGE no schema `public`. Corrigir e adicionar regressões PG16 para coluna surpresa/secret, grant isolado em coluna e ACL PUBLIC de database/schema. Minor adicionais: doc de key history diz que qualquer history inativa é inválida embora rotação normal a crie; validar tipo/cardinalidade da tabela singleton legacy antes de import/drop.

Próximo passo exato: implementar esses três Major em migration/init/readiness com grants mínimos e controles fail-closed; testar em PostgreSQL 16 de verdade (role runtime/migrator/admin, grants por coluna, schema column drift, PUBLIC database/schema ACL) e rerodar integration, cutover scripts, `go test ./...`, `go vet ./...`, YAML/shell/diff; pedir re-review read-only do delta. Depois: `git add -A`, commit claro, `git push origin recovery/ollama-full-snapshot` sem force-push; atualizar Mission State + este handoff com hash funcional e commit+push de docs. PostgreSQL continua BLOQUEADO para produção sem staging real HA/DR/TLS/restore e aceite operacional independente.

Pergunta Manus Flex: a documentação oficial pública encontrada descreve sandbox persistente e execução de tarefas, mas não especifica exatamente o produto/plano denominado Manus Flex. Portanto não afirmar plano/créditos/eligibilidade; pode ser usado como colaborador se tiver acesso autorizado ao branch, sem substituir validação, revisão ou garantir término. Ver docs públicos: https://manus.im/docs/introduction/welcome e https://manus.im/docs/features/skills.


## Continuidade — PG16/HBA final review e branch reconciliation — 2026-09-29 17:33 -03

**Checkpoint funcional publicado**

- Branch: `recovery/ollama-full-snapshot`.
- Commit enviado sem force-push: `30ae8f22394205115e1d9154dae9866fcee82a52` — `fix(security): close postgres isolation and cutover races`.
- Confirmação: `git ls-remote origin refs/heads/recovery/ollama-full-snapshot` retornou o mesmo SHA. Nenhuma outra branch foi modificada.

**Estado implementado e evidência**

- PostgreSQL RLS/tenant-context HMAC, role split, ACLs, phase-one/phase-two cutover e HBA atualizado. Membership traversal segue apenas arestas explícitas de `pg_auth_members` com CTE recursivo; não usa `pg_has_role` contra a lista geral de roles. Phase 1 faz recaptura/revoke sob lock antes da transferência, depois drena e verifica de novo; phase 2 também drena membros autenticados `SET ROLE`. SQL tem `lock_timeout` curto e aborta em caso de contenção. Readiness/fixtures não matam sessão do administrador independente.
- HBA de produção concede runtime/migrator somente a `ollama_agent`, mantendo admins TCP via SCRAM; denies aparecem depois dos allows app-scoped. O CI constrói HBA temporário em `$RUNNER_TEMP` com entrada para `ollama_agent_reverse_member` só para o teste, confirma a linha gerada e realiza login TCP positivo antes do drain. Essa identidade não deve entrar no arquivo/deploy de produção.
- `public.hmac(bytea,bytea,text)` tem owner verificado corretamente por `pg_proc.proowner`.
- Harness real descartável PG16 + Redis: fresh init; cutover de legado incluindo extensão/key; search_path hostil; sessões do superuser legado e membro terminadas em ambas as fases; sessão admin não relacionada preservada; HMAC owner correto; runtime/migrator bloqueados por HBA em DB criado com `PUBLIC CONNECT`; suíte de drift RLS, rotação, Redis e cross-database passou. Cluster e listeners temporários limpos.
- Gates: workflow YAML/bash embutido, Compose YAML, integrity guard, `bash -n`, `go test ./...`, `go vet ./...`, `go test -race ./internal/agent`, `go build`, diff check e Gitleaks passaram. Review independente final: **Approve / no remaining blocker**. Ainda não há evidência de execução do GitHub Actions associada a este commit.
- Operação: mantenha janela de cutover sem outros DBAs/superusers/automação privilegiada; um superuser independente pode voltar a alterar memberships após um lock ser liberado. Serviço e tráfego só reabrem após phase 2 e smoke.

**Branches/PR observados via GitHub API (read-only, 2026-09-29)**

- PR [#38](https://github.com/DZ23-LTDA/ollama-classe-a-plus/pull/38) continua OPEN/mergeable, head `fix/audit-security-deps-2026-09-25` em `07a4f0bca2cbe327d5234abd9ebec69ff52154e6`, base `main` em `8635e30dc9e95a1f5b29700169783abc24093ceb`. Contém `ConnectorsPage`, quick-connect, Providers, icons e implementação relacionada; a interface de conectores dessa linha é mais completa que a duplicação parcial construída na linha de UI.
- Comparação `recovery...PR38`: diverged, recovery +35 / -27, merge-base `8635e30d`, 121 paths no diff.
- Comparação `recovery...feat/ui-shell-parity`: diverged, +17 / -15, merge-base `add5881a260ff1f740b1340c6f394c26acc2d5d2`, 57 paths. Nenhuma reconciliação/merge foi executada nesta etapa.
- O trabalho de UI aditivo citado no contexto inclui tema Claro/Escuro/Automático, Home/dashboard, status pills e rastreador de etapas. A proposta de produto registrada pelo usuário é convergir as linhas antes de construir mais telas e priorizar chat como agente executável (tools, arquivos, terminal, browser/MCP, artifacts e approvals), mas isso ainda precisa ser integrado/revalidado na base canônica.

**Próximo passo exato:** montar uma matriz de commits/paths/funcionalidades para recovery, PR #38 e `feat/ui-shell-parity`, definir qual branch vira base canônica e resolver Connectors duplicado preservando a melhor implementação; planejar integração que mantenha as correções PostgreSQL. Até lá, manter branches/PR read-only e não alegar paridade completa. Depois da decisão, integrar em branch autorizada, rodar gates reais, atualizar ambos checkpoints a cada rodada e fazer push normal apenas ao destino autorizado.

**Estado release:** backend security localmente validado e revisão fechada; produto não está declarado production-ready. Staging PostgreSQL real, HA/fencing/partições, backup/restore, RPO/RTO, TLS/CA operacional, monitoramento/alertas e aceite operacional continuam pendentes.


## Preparação da cópia para outro Manus — 2026-09-29 18:10 -03

Correção explícita de contagem (consulta GitHub em 17:43 -03, recovery como `base`): o branch head do PR #38 tem 35 commits não presentes em recovery; recovery tem 29 não presentes no PR; 121 arquivos no compare. UI parity tem 17 commits de cada lado, 57 arquivos. PR #38 está OPEN contra `main` e possui Connectors/Providers mais completos, quick-connect, ícones, catálogo/persistência e upload. UI parity tem sobreposição em `ProductWorkspacePage`, `ConnectorLogo` e `connectorCatalog.ts` e funcionalidades candidatas aditivas de shell, tema, Home, console e jornadas. Nenhuma branch foi mesclada ou alterada.

Guia novo: `audit/HANDOFF_MANUS_20260929.md`. Para o ZIP, incluir snapshot tracked recovery, Git bundle seletivo das quatro refs atuais, checkpoints/guias, pacote visual histórico preservado como bytes e manifest. O usuário confirmou excluir o antigo ZIP de backup P: e não extrair/interpretar os binários. O próximo agente deve confirmar estado e SHA atuais, ler o guia e construir primeiro uma matriz de funcionalidades/conflitos; não escolher nem integrar branches automaticamente. Produção continua não autorizada sem ensaios de staging e operações descritos nos checkpoints.


## Pacote para continuidade — estado recalculado em recovery 6980786c — 2026-09-29 18:12 -03

Reconsultados PR/refs no GitHub usando recovery `6980786c484192ace6b735458bc8e51ab7ef7279` como base: PR #38 permanece OPEN, head `07a4f0bca2cbe327d5234abd9ebec69ff52154e6`; compare diverged com PR +35 commits e recovery +31, 121 arquivos, merge-base main `8635e30dc9e95a1f5b29700169783abc24093ceb`. UI parity head `132e77155fed682c0b6244d256e6cdfe9b25c239`; compare diverged UI +17 e recovery +19, 57 arquivos, merge-base `add5881a260ff1f740b1340c6f394c26acc2d5d2`. Divergência, não merge; contadores variam com cada commit em recovery.

Escopo confirmado do handoff: snapshot Git-tracked atual + Git bundle de quatro refs (main/recovery/PR38/UI) + guia novo e checkpoints + cópia binária do ZIP visual já existente. Excluir o backup antigo de P:; não extrair ZIP visual. Gerar o arquivo ZIP de entrega fora do repositório, validar CRC, bundle refs e checksums, depois fornecer link de download e SHA-256. Nenhuma integração de branches foi autorizada nesta fase.


## Frozen compare para o handoff ZIP — 2026-09-29 18:14 -03

Base comparada: recovery `3203fa1f732a013c35bd46467a2c15debdd27466`, imediatamente antes do próximo commit documental. PR #38 head `07a4f0bca2cbe327d5234abd9ebec69ff52154e6`: estado OPEN/mergeable; diverged, PR head +35 e recovery +32 commits, 121 arquivos; merge-base `8635e30dc9e95a1f5b29700169783abc24093ceb`. UI parity `132e77155fed682c0b6244d256e6cdfe9b25c239`: diverged, UI +17 e recovery +20; 57 arquivos; merge-base `add5881a260ff1f740b1340c6f394c26acc2d5d2`. O commit subsequente adiciona somente documentação/checkpoint; o README do ZIP informa SHAs do snapshot e refs do bundle separadamente.


## Ajuste final de precisão SHA — 2026-09-29 18:17 -03

Corrigido no guia novo o rótulo enganoso que podia indicar `65cb08...` como snapshot entregue. Esse SHA é checkpoint antigo; a correção de segurança é `30ae8f22...`; a fonte que vai em `source/` usa `SOURCE_SNAPSHOT_SHA` exato no README do ZIP. O rebuild do pacote deve executar após a publicação do novo commit documental e recriar tanto export de source quanto Git bundle.


## Evidência preliminar do pacote ZIP — 2026-09-29 18:19 -03

Candidato v2 de 132.569.470 bytes (1.871 entradas) passou `zip -T`, `unzip -t`, SHA256SUMS dos membros e `git bundle verify` (4 refs); cópia do ZIP visual comparada byte a byte com o original. Checksum do candidato v2 `3ac44659eb923285ead1c04d3d4d0b8cf8fd9748f093fca33b40bbe539a46056`. Esse não é o arquivo final a ser anexado: foi gerado no source `304f1224...`; após publicar esta atualização de auditoria, regenerar pelo SHA final, confirmar `SOURCE_SNAPSHOT_SHA`=bundle recovery=origin, recalcular SHA do ZIP e entregar somente o final validado.


## Validação final para o pacote portable — 2026-09-29 18:21 -03

O candidato v3 foi gerado do source SHA `0ba6b15f5dc55c0b8667a5561478a14bf81e3fd4`, passou manifesto, `git bundle verify`, `zip -T` e `unzip -t`; continha 1.871 entradas. O source SHA e recovery ref do bundle eram iguais, e o ZIP visual foi comparado byte a byte. Antes da entrega, publicar este último checkpoint e regenerar o ZIP do novo source SHA; esse arquivo reconstruído será o único final. Nenhum merge ou modificação em `main`, PR #38 ou UI parity.
