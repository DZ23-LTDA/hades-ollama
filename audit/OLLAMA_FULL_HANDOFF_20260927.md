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