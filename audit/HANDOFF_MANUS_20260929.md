# Passagem de bastão do Ollama Full — 29/09/2026

## Objetivo do projeto

O proprietário está transformando o repositório `DZ23-LTDA/ollama-classe-a-plus` em **Ollama Full**: produto agentic local-first que pretende reunir e superar funcionalidades de Manus e de outros harnesses. A intenção é construir uma experiência integrada, não apenas telas: o chat deve executar missões e ferramentas, com autorização, histórico, arquivos/artifacts, terminal/browser/MCP e controles de segurança. **Paridade total não foi comprovada nem alcançada**; trate as análises anteriores como requisitos e direção, não como funcionalidades concluídas.

## Estado do snapshot e repositório

- Repositório: `https://github.com/DZ23-LTDA/ollama-classe-a-plus`.
- Branch do snapshot: `recovery/ollama-full-snapshot`.
- O commit-base da rodada de segurança é `30ae8f22394205115e1d9154dae9866fcee82a52` (`fix(security): close postgres isolation and cutover races`); `65cb08bec4880447369690668b2c35e024319049` é um checkpoint anterior. O SHA exato do snapshot Git incluído será o valor marcado `SOURCE_SNAPSHOT_SHA` em `README-START-HERE.md` na raiz deste pacote.
- O checkout estava limpo e alinhado a `origin/recovery/ollama-full-snapshot` quando foi empacotado; não há alteração sem commit conhecida.
- O `SOURCE_SNAPSHOT_SHA` registrado no README identifica o estado completo da branch recovery exportado para `source/`; os SHAs do bundle indicam separadamente o histórico das quatro refs. O ZIP contém snapshot Git-tracked da branch recovery, checkpoints, guia, Git bundle seletivo com histórico e referências visuais aprovadas anteriormente. Não contém `.git` como diretório, caches/builds locais, `.env`, credenciais, nem o ZIP antigo de backup da unidade P:.

## Segurança implementada e evidência disponível

O backend contém PostgreSQL RLS estrita, contexto tenant HMAC versionado, roles separadas (`ollama_agent_admin`, `ollama_agent_migrator`, `ollama_agent_runtime`), ACLs minimizadas e HBA com isolamento a nível de database. O cutover legado ocorre em duas fases, inicia com fence NOLOGIN, drena sessões de roles e membership transitivo, usa locks com timeout, recaptura membership durante transferência de ownership e verifica o drain final. A ownership da função pgcrypto `public.hmac(bytea,bytea,text)` é verificada por `pg_proc.proowner`.

O workflow CI gera um HBA **temporário e exclusivo do job** para permitir login do papel auxiliar `ollama_agent_reverse_member`; isso torna possível testar sessões autenticadas que executam `SET ROLE`. Esta regra não pertence ao HBA de produção. Runtime e migrator só recebem allow TCP para `ollama_agent` e têm rejects para outros databases; ACLs sozinhas não controlariam databases futuros.

A evidência local no snapshot veio de PostgreSQL 16 e Redis 7 descartáveis: fresh init, cutover legado (incluindo pgcrypto e key import), search_path hostil, drains phase 1/phase 2, preservação de admin independente, owner HMAC correto, runtime/migrator negados em database novo com `PUBLIC CONNECT`, drift/RLS e Redis. Ao final, o cluster e listeners temporários foram removidos.

Gates executados e aprovados nessa rodada: `go test ./...`, `go vet ./...`, `go test -race ./internal/agent`, `go build`, YAML do GitHub Actions e bash embutido, YAML do Compose, `bash -n`, `scripts/check-class-a-plus-integrity.sh`, `git diff --check` e Gitleaks staged. A última revisão independente aprovou o patch. **Não há confirmação de que GitHub Actions executou no SHA de segurança.** A aprovação não substitui validação real em staging.

Arquivos-chave para começar:

- `internal/agent/postgres_security.go` — assinatura/contexto HMAC, atestação de RLS/ACL/schema e readiness.
- `internal/agent/distributed_integration_test.go` — testes distribuídos adversariais.
- `deploy/postgres/init/001-agent-role.sql` — bootstrap e ACLs.
- `deploy/postgres/migrate-existing-roles.sql` — phase 1 do cutover.
- `deploy/postgres/retire-legacy-role.sql` — phase 2.
- `deploy/postgres/pg_hba-runtime.conf` — HBA de produção.
- `deploy/docker-compose.agentic.yml` — infraestrutura local/Compose e montagem HBA.
- `.github/workflows/dz23-agentic-quality.yaml` — CI real de PostgreSQL/Redis.
- `scripts/check-class-a-plus-integrity.sh` — guardas de integridade.
- `docs/agentic/INTEGRATIONS.md` e `docs/agentic/POSTGRES_HMAC_KEY_ROTATION.md` — operação.
- `audit/OLLAMA_FULL_MISSION_STATE.md` e `audit/OLLAMA_FULL_HANDOFF_20260927.md` — checkpoints acumulativos.

## Relação atual das branches — verificada via GitHub em 29/09/2026 17:43 -03

| Ref | SHA atual observado | Estado |
|---|---|---|
| `main` | `8635e30dc9e95a1f5b29700169783abc24093ceb` | Base antiga do PR #38 |
| `recovery/ollama-full-snapshot` | `65cb08bec4880447369690668b2c35e024319049` | Snapshot de segurança deste pacote |
| `fix/audit-security-deps-2026-09-25` (head do PR #38) | `07a4f0bca2cbe327d5234abd9ebec69ff52154e6` | PR #38 aberto contra `main`, API marcou mergeable |
| `feat/ui-shell-parity` | `132e77155fed682c0b6244d256e6cdfe9b25c239` | Branch divergente de UI |

PR #38: https://github.com/DZ23-LTDA/ollama-classe-a-plus/pull/38 (estado `OPEN`). Comparação `recovery...PR38`: `diverged`; head do PR está 35 commits à frente de recovery e recovery está 29 commits à frente do PR; merge-base `8635e30dc9e95a1f5b29700169783abc24093ceb`, 121 arquivos no diff. PR #38 contém Connectors/Providers, quick-connect, rotas, catálogo/persistência Go, ícones e upload; por isso a tela de conectores duplicada em `feat/ui-shell-parity` não deve ser expandida sem reconciliar.

Comparação `recovery...feat/ui-shell-parity`: `diverged`, 17 commits à frente e 17 atrás, merge-base `add5881a260ff1f740b1340c6f394c26acc2d5d2`, 57 arquivos no diff. A branch de UI inclui elementos potencialmente aditivos como tema claro/escuro/automático, Home, `AgenticConsole`, jornadas/evidências E2E, rastreador de tarefas, status e `ProductWorkspacePage`. Há sobreposição direta em `ProductWorkspacePage`, `ConnectorLogo` e `connectorCatalog.ts`; preservar as páginas mais completas de Connectors/Providers do PR e adaptar a shell aditiva é hipótese de trabalho, ainda não validada.

Esses números são uma fotografia de 17:43 -03; PR/branches podem avançar. Antes de integração, atualizar todos os SHAs/comparações por `gh pr view 38` e `gh api repos/DZ23-LTDA/ollama-classe-a-plus/compare/<base>...<head>`. No Git bundle, recuperar branches remotas não cria automaticamente branches locais: use os comandos desta seção após verificar `git bundle list-heads`.

## Procedimento para abrir o Git bundle

O bundle desta entrega contém `main`, `recovery`, o head do PR e `feat/ui-shell-parity`. Depois de extrair o pacote:

```bash
git clone ollama-full-branches.bundle ollama-full-from-bundle
cd ollama-full-from-bundle
git bundle list-heads ../ollama-full-branches.bundle
```

Se o Git não criar branches locais automaticamente para as refs de tracking, crie-as a partir dos SHAs/refs listados, preservando recovery como está. Compare/inspecione primeiro; não use reset destrutivo nem force-push. As branches originais continuam no GitHub; bundle é uma cópia portátil, não um novo remote.

## Próxima fase recomendada

1. **Atualizar o terreno:** confirmar working tree, branches remotas, estado do PR #38 e comparar de novo usando os SHAs atuais. Ler este guia e os dois checkpoints antes de alterar.
2. **Produzir matriz de integração:** agrupar por arquivo e funcionalidade (security, backend/API, providers/connectors, shell/navegação, tema, Home, chat/console, testes/evidências, deps/Windows), indicar conflito direto, origem da melhor implementação, testes necessários e dono dos dados/migrations.
3. **Propor base canônica:** a hipótese mais segura é tratar recovery como fonte das correções PostgreSQL e integrar seletivamente código aprovado de PR #38 e UI parity numa branch candidata, sem substituir automaticamente a proteção de RLS/HBA. Não fazer merge do PR #38 ou de UI parity em recovery até revisar migrações, contracts, routes e conflitos.
4. **Reconciliar connectors:** priorizar a experiência madura `ConnectorsPage`/`ConnectorsManagePanel`/`ConnectorQuickConnect` e `ProvidersPage` do PR #38; reaproveitar da UI parity apenas shell/theme/workflows compatíveis. Decidir como `ProductWorkspacePage` acessa catálogo/API sem duplicar UX.
5. **Atacar o principal salto de produto:** chat principal como executor real de missão: tools, arquivos, terminal, browser/MCP, execução longa/background, artefatos, approval e observabilidade, conectando a Mission Console como detalhe e não como fluxo obrigatório paralelo. Fazer inventário antes de implementar; não chamar mock de paridade.
6. **Verificar cada lote:** frontend checks e testes de jornadas, Go tests/vet/race/build, integrações PG16/Redis, testes de authorization negativa/tenant, CI e independent reviews. Preservar checkpoints após cada lote verificado.
7. **Staging/produção é fase separada:** provisionar um cluster staging dedicado; validar CA/SAN e TLS `verify-full`; rehearsal de backup/restauração com `system_identifier` distinto; HA promotion/fencing/partições, RPO/RTO, alertas, smoke, rollback e aprovação operacional. Não direcionar testes adversariais para produção.
8. **Publicação:** só push normal para uma branch explicitamente autorizada; nunca force-push e nunca `main` por padrão. Atualizar ambos audit checkpoints com SHA, testes, riscos, partes incompletas e passo concreto seguinte.

## Restrições operacionais e itens não concluídos

- Produção permanece **não aprovada**: faltam exercícios em staging real para TLS/backup-restore, HA/fencing/partições, RPO/RTO, alertas e aceite do operador.
- Handoff não afirma paridade completa com Manus; há lacunas de produto e execução.
- Nunca expor credentials. Não usar mocks como evidência de infraestrutura real. Não usar o script local efêmero em sistemas que não sejam descartáveis; requer Linux, sudo, PostgreSQL 16, Redis, Go e as dependências, e possui paths/ports próprios.
- A confirmação do usuário neste contexto autorizou apenas o ZIP com o snapshot, o Git bundle das quatro linhas, o ZIP de imagens visuais existente e exclusão do antigo ZIP P:. Nenhuma branch foi mesclada ou modificada para esta entrega.

## Conteúdo deste pacote

O arquivo raiz `ollama-full-handoff-20260929.zip` deve incluir:

1. `source/` — snapshot git archive de recovery no SHA indicado.
2. `history/ollama-full-branches.bundle` — histórico das quatro refs selecionadas.
3. `handoff/HANDOFF_MANUS_20260929.md` — este plano e estado.
4. `handoff/` com checkpoints e arquivos de continuação existentes.
5. `references/ollama-full-reference-images.zip` — arquivo visual anterior, copiado sem extração.
6. `SHA256SUMS.txt` e `README-START-HERE.md` na raiz.


## Revalidação mais recente das branches — 2026-09-29 18:12 -03

Recovery estava em `6980786c484192ace6b735458bc8e51ab7ef7279` ao recalcular pela GitHub API: PR #38 continua OPEN, head `07a4f0bca2cbe327d5234abd9ebec69ff52154e6`, compare diverged com PR head +35 e recovery +31 commits, 121 paths, merge-base `8635e30dc9e95a1f5b29700169783abc24093ceb`. UI parity head `132e77155fed682c0b6244d256e6cdfe9b25c239`, compare diverged UI +17 / recovery +19, 57 paths, merge-base `add5881a260ff1f740b1340c6f394c26acc2d5d2`. Esse número mudou porque recovery recebeu commits de documentação após medições anteriores; as direções são “head da outra branch +N; recovery +N”. O pacote e o próximo Manus devem conferir de novo os SHAs remotos.

O snapshot de implementação principal continua sendo `30ae8f22394205115e1d9154dae9866fcee82a52`; commits depois dele são checkpoints/documentação. O ZIP final indicará snapshot/head incluído e SHAs no README/manifest.


## Snapshot exato usado antes do commit final do pacote — 2026-09-29 18:14 -03

Base recovery `3203fa1f732a013c35bd46467a2c15debdd27466`. PR #38 (`07a4f0bca2cbe327d5234abd9ebec69ff52154e6`) OPEN/mergeable: diverged, head PR +35 e recovery +32, 121 arquivos, merge-base `8635e30dc9e95a1f5b29700169783abc24093ceb`. UI parity (`132e77155fed682c0b6244d256e6cdfe9b25c239`): diverged, UI +17 e recovery +20, 57 arquivos, merge-base `add5881a260ff1f740b1340c6f394c26acc2d5d2`. Em seguida, recovery recebe apenas um commit de handoff/documentação. A snapshot de código e auditorias do ZIP deve identificar esse commit posterior exato; as outras branches permanecem imutáveis.


## Estado final do empacotamento

Os ZIPs intermediários anteriores são superseded. Entregar apenas o artefato final cujo `SOURCE_SNAPSHOT_SHA` no `README-START-HERE.md` corresponda ao head de recovery gravado no Git bundle e confirmado no origin. A auditoria final é commitada antes de gerar esse artefato; checksum/tamanho do ZIP são enviados no chat para evitar registrar num arquivo conteúdo auto-referente.
