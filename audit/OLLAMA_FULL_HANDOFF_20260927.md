# Handoff de continuidade — Ollama Full

**Atualizado:** 2026-09-27 07:24 (-03)

**Propósito:** permitir retomar o trabalho depois de encerrar/formatar este PC ou trocar de sessão, sem depender do histórico da conversa.

## Estado remoto confirmado

- **Projeto:** [DZ23-LTDA/ollama-classe-a-plus](https://github.com/DZ23-LTDA/ollama-classe-a-plus)
- **Branch de continuidade:** [`recovery/ollama-full-snapshot`](https://github.com/DZ23-LTDA/ollama-classe-a-plus/tree/recovery/ollama-full-snapshot)
- **Ponta verificada no GitHub:** `2ac42cc24f64546c87d7296f919b1519de0c67b5`
- **Commits mais recentes:**
  - `8c2e3604a086abeab401e2c80e96fe6ad4473566` — `security: harden tenant isolation and workspace boundaries` (snapshot de 134 arquivos; 22.536 inserções, 1.582 remoções)
  - `2ac42cc24f64546c87d7296f919b1519de0c67b5` — `docs: record verified GitHub backup`
- **`main`:** continua em `8635e30dc9e95a1f5b29700169783abc24093ceb`; não foi alterada. Não houve force-push nem PR.
- O checkout local estava limpo após o push. A branch de recuperação acompanha `origin/recovery/ollama-full-snapshot`.

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

1. `audit/OLLAMA_FULL_HANDOFF_20260927.md` — este handoff.
2. `audit/OLLAMA_FULL_MISSION_STATE.md` — histórico durável detalhado e evidências por rodada.
3. `audit/FINAL_SECURITY_REAUDIT_POSTFIX_20260927.md` — achados da auditoria independente, distinção entre findings remediados e o bloqueador remanescente.
4. `docs/agentic/PARITY_MATRIX.md` — paridade por domínio, com estados e gates faltantes; não tratar adapters como recursos completos.
5. `docs/OLLAMA_FULL_PRODUCT_VISION.md` e `audit/COMPARACAO_ARVORES_MANUS_OLLAMA_FULL.md` — visão de produto e comparação de escopo.
6. `SECURITY.md` e `docs/RELEASE_READINESS.md` — regras de segurança e critérios de liberação.

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

As policies usam GUCs como `app.current_organization_id` e `app.system_access` que podem ser configuráveis pela role da aplicação. A API/runtime mantêm PostgreSQL **desabilitado e fail-closed** como contenção. Não habilitar Postgres, não dizer que o RLS está corrigido e não marcar isolamento enterprise até:

1. projetar contexto/roles não forjáveis pela role runtime (separar role/DSN de migração e runtime; remover caminhos de bypass);
2. implementar migração/backfill compatível e revisar grants/policies;
3. obter PostgreSQL de teste com roles não-superuser/não-BYPASSRLS e provar por testes adversariais que SQL executado pela role runtime não consegue forjar tenant nem system access;
4. executar integração real e rever independentemente a arquitetura.

A documentação preserva evidência histórica de smoke PostgreSQL descartável em uma retomada anterior, mas a reauditoria mais recente não teve `OLLAMA_AGENT_TEST_POSTGRES_URL` nem `OLLAMA_AGENT_TEST_REDIS_URL` disponíveis. Nenhum desses smokes remove o problema de GUC forjável.

### P1 — Reauditoria dos findings remediados

O relatório `FINAL_SECURITY_REAUDIT_POSTFIX_20260927.md` preserva findings originais e uma atualização pós-remediação: `terminal.exec`/PATH de Git, catálogo MCP ownerless, URLs de provider com query assinada, aliases de `signature` no DLP, quotas/trabalho do PushOutbox, owner divergente na listagem de jobs e respostas de deploy truncadas foram marcados como corrigidos no checkout, com testes. **As três revisões independentes ainda precisam validar o estado pós-fix em checkout limpo.** Faça uma reauditoria focada nesses fluxos; execute todos os testes afetados; atualize o relatório com evidências e qualquer finding novo antes de dizer que estão encerrados.

### P2 — Paridade funcional e plataforma

A matriz continua classificando vários domínios como `PARCIAL` ou `ADAPTER IMPLEMENTADO`. Entre os itens explícitos: jornada E2E longa de missões; estados reais, acessibilidade e responsividade de cada tela; smoke autorizado de providers externos/OAuth; ferramentas reais de browser/desktop; ciclo coding com worktree/branch, execução de testes segura, repair loop e aplicação de mudanças; colaboração CRDT; mobile em dispositivos; deploy com contas reais e rollback; multimídia com modelos/GPU/Tesseract; staging de queues/observabilidade. Priorize uma jornada completa end-to-end por vez e atualize a matriz com implementação, teste automático e execução observável.

### P3 — Integridade/limites restantes

Revisar o relatório e a matriz quanto a falhas de crash/failover, semântica at-least-once, Redis Cluster não suportado, fallbacks não-Linux que devem permanecer fail-closed, boundedness em todos os formatos/callsites, logs/metrics, e execução nativa Windows/macOS. Não transformar teste cross-compile em alegação de teste nativo.

## Gates comprovados neste snapshot

Executados sobre a árvore congelada anterior ao commit documental `2ac42cc` (que só alterou texto do checkpoint):

- `gofmt` check dos Go alterados/novos — PASS;
- `go test -p=2 ./... -count=1` — PASS;
- `go vet -p=2 ./...` — PASS;
- `go test -race -p=2 ./internal/agent ./server -count=1` — PASS;
- `CGO_ENABLED=1 go build -p=2 ./...` — PASS;
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -p=2 -o /tmp/ollama-agent-windows.test.exe ./internal/agent` — PASS (cross-compile do pacote agent; não é execução nativa do Windows);
- `bash scripts/check-class-a-plus-integrity.sh` — PASS;
- `git diff --check` — PASS;
- gitleaks redacted no conjunto de arquivos modificados/novos — 18 detecções em `_test.go` contendo fixtures sintéticas; zero detecções em arquivos não-test. Valores não foram impressos.

Após qualquer alteração de código, repetir testes relevantes; antes de novo release/merge, repetir todos os gates em checkout limpo. O commit documental posterior só mudou três linhas do checkpoint, mas deve passar `git diff --check` quando houver nova mudança.

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

**Estado na criação deste handoff:** remoto verificado em `2ac42cc24f64546c87d7296f919b1519de0c67b5`; `main` em `8635e30dc9e95a1f5b29700169783abc24093ceb`; nenhum serviço local de screenshot ativo.
