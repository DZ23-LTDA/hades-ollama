# Handoff de continuidade — Ollama Full

**Atualizado:** 2026-09-29 02:38 (-03)

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
