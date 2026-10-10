# Reauditoria independente pós-correção — 2026-10-09

Reauditoria exigida pelo item P1 de `audit/CLAUDE_CODEX_RESUME_PROMPT_20260927.md`:
reconstruir, no código atual, cada achado bloqueante de
`audit/FINAL_SECURITY_REAUDIT_POSTFIX_20260927.md` — sem aceitar o relatório
anterior como prova — e registrar veredito por achado com evidência reproduzível.

Revisão de código somente leitura + execução de testes. Nenhum arquivo de
produção foi alterado nesta reauditoria.

## 1. Base verificada

| Item | Valor |
| --- | --- |
| Commit de `main` auditado | `91256ac` (merge do PR #54) |
| Árvore local | `hades-ollama`, branch `fix/ci-verde-main-20261008` (já mesclada), sem alterações pendentes além de `_tmp/` |
| CI em `main` (`push`) | `class-a-plus-integrity` run `37876151049` = success; `dz23-agentic-quality` run `37876151048` = success; ambos sobre `91256ac` |
| Suíte focal de segurança | `go test ./internal/agent -run '<filtro>' -count=1` → `ok github.com/ollama/ollama/internal/agent` (exit 0) |
| Config. lint | `misspell` habilitado por inteiro; nenhum `ignore-words` novo, nenhum `//nolint` novo, nenhum bloco `issues:` adicionado |

Nota de ambiente (Windows): testes que criam diretórios em `%TEMP%`/`%APPDATA%`
falham nesta máquina com `Acesso negado` (ACL do host), o que **não** é defeito de
código. Com `TMP`/`TEMP`/`APPDATA`/`LOCALAPPDATA` redirecionados para `_tmp/`, os
mesmos testes passam. Os vereditos abaixo usam essa execução.

## 2. Veredito por achado

### Achado 1 — `terminal.exec` aceitava `git` do PATH ambiente — **FECHADO**

- `internal/agent/tools.go:40` — o registro de produção allowlista apenas
  `pwd` e `ls`; `git` não é registrado.
- `internal/agent/tools.go:244-250` — o executável é reduzido a `filepath.Base`,
  validado contra allowlist e, para `git`, rejeitado explicitamente com erro que
  aponta para o caminho endurecido (`git.repo.inspect`).
- `internal/agent/tools.go:266,277,279` — a resolução do binário ocorre **antes**
  de montar o comando, via `trustedToolExecutable`; o processo filho ainda recebe
  `PATH=/usr/bin:/bin` fixo.
- `internal/agent/git_repo.go:579-614` — `trustedToolExecutable`/`safeToolPath`
  só aceitam diretórios de sistema conhecidos (`/usr/local/bin:/usr/bin:/bin`,
  e no Windows `C:\Program Files\Git\...`, `System32`, `Windows`), exigem arquivo
  regular, `trustedSystemExecutableFile` e bit de execução (fora do Windows).
  O PATH ambiente herdado pelo serviço nunca é consultado.
- Regressão: `internal/agent/tools_test.go:45-64`
  `TestTerminalGitIgnoresAmbientPATHReplacement` injeta um `git` falso no início do
  `PATH` e comprova que o processo não é executado (marcador ausente) e que o erro
  cita `git.repo.inspect`. **PASS**.
- Cobertura complementar: `TestTerminalUsesPinnedWorkspaceAfterPathReplacement`
  (skip apenas fora de Linux, por design).

### Achado 2 — listagem autenticada expunha configs MCP sem dono — **FECHADO**

- `internal/agent/plugin_scope.go:27-35` — `pluginAccessibleByOrganization` é
  tenant-only: para organização real exige dono explícito e igual; somente
  `LocalOrganizationID` enxerga registros globais (sem dono), e isso está
  documentado no próprio código como reservado a APIs locais confiáveis.
- `internal/agent/mcp.go:220-235` — `ListForOrganization` filtra por esse escopo e
  remove `Args` e `EnvironmentVars` antes de devolver.
- `internal/agent/mcp_remote.go:477-494` — mesmo filtro + projeção sanitizada:
  `URL = providerCatalogOrigin(...)`, `TokenEnv = ""`, `HeadersEnv = nil`,
  `AllowedMethods` copiado.
- `internal/agent/connectors.go:355-371` — mesmo filtro + `BaseURL` reduzido à
  origem e `TokenEnv` limpo.
- Não há rota autenticada consumindo a listagem global sem escopo: buscas por
  `ListGlobal` em `server/` e `app/` não retornam nenhuma ocorrência; as rotas
  usam `ListForOrganization(organizationID do chamador)` (`server/agent_routes.go:2591,2666`).
- Testes: `TestMCPManagerCallRejectsOwnerlessServerForTenant` (**PASS**) e
  `TestRemoteMCPRequiresExplicitGlobalOrOrganizationScope` (**PASS**).

### Achado 3 — URLs de provedor (inclusive query assinada) devolvidas inteiras — **FECHADO**

- `internal/agent/egress_policy.go:22-28` — `providerCatalogOrigin` devolve somente
  `esquema://host`; retorna `[REDACTED]` se a URL não parseia, não tem esquema/host,
  tem userinfo ou contém material sensível. Caminho e query são descartados.
- `internal/agent/egress_policy.go:35-45,47-70` — `validateConfiguredEndpointURL`
  (fail-closed, teto de 16 KiB) e `endpointURLHasSensitiveMaterial` rejeitam
  credenciais e query ambígua, normalizando cada chave de query com
  `normalizeDLPKey` + decodificação completa do percent-encoding.
- Aplicado nas três superfícies de listagem (`deploy.go:252`,
  `mcp_remote.go:487`, `connectors.go:365`).
- `internal/agent/deploy.go:642-658` — `sanitizeProviderURL` protege também a URL
  devolvida em deploy bem-sucedido (rejeita userinfo, fragmento, esquema inválido e
  chaves `token/secret/credential/apikey/accesskey/signature/sig`), com `RedactDLP`
  por cima.
- Testes **PASS**: `TestDeploymentTenantCatalogSanitizesProviderURL`,
  `TestDeploymentRejectsCredentialBearingEndpointURL`,
  `TestDeploymentRejectsSecretBearingEndpointBeforePersistence`,
  `TestDeploymentRedactsCredentialBearingSuccessfulURL`,
  `TestRemoteMCPRejectsSecretBearingEndpointBeforePersistence`,
  `TestConfiguredEndpointURLRejectsEncodedSignatureJSON`.

### Achado 4 — chaves de assinatura escapavam do DLP — **FECHADO**

- `internal/agent/secrets.go:496-512` — `sensitiveDLPKey` cobre exatamente
  `signature` e `sig`, além dos sufixos `_signature` e `_sig` (entre outros).
- `internal/agent/secrets.go:514-546` — `normalizeDLPKey` converte camelCase em
  snake_case, troca qualquer execução de caracteres não alfanuméricos por `_`
  (colapsando repetições) e faz `Trim("_")`: `x.sig` → `x_sig`,
  `X-Signature` → `x_signature`, `providerSignature` → `provider_signature`,
  `authSignature` → `auth_signature`. Todas essas formas casam com o switch ou com
  a lista de sufixos.
- Aplicação também nas chaves de query das URLs
  (`egress_policy.go:63-67`).
- Testes **PASS**: `TestDLPRedactsSignatureAliasesRecursively`,
  `TestDLPRedactsKnownCredentialShapes`,
  `TestDLPRedactsCredentialQueryURLsRecursively`,
  `TestDLPRedactsAmbiguousEncodedURLCandidates`,
  `TestDLPRedactsCredentialAndTokenAssignmentsInMetadata`,
  `TestDLPRedactsEmbeddedSensitiveJSONFromStrings`,
  `TestRedactValueRecursesThroughToolPayloads`,
  `TestValidateOutboundPayloadRejectsSensitiveKeysAndValues`.

### Achado 5 — `PushOutbox` sem cota nem limite de trabalho — **FECHADO**

- `internal/agent/push_outbox.go:38-40` — tetos explícitos:
  `maxPushOutboxAttempts = 8`, `maxPushOutboxItems = 256`,
  `maxPushOutboxItemsPerTenant = 64`.
- `internal/agent/push_outbox.go:127-172` — `Enqueue` valida entrada, passa por
  varredura DLP e por `validatePushOutboxItemSize`; sob o lock de arquivo poda
  terminais (`:159`) e aplica cota global (`:160-162`) e por tenant
  (`:163-171`), devolvendo `errPushOutboxQuota`.
- `internal/agent/push_outbox.go:184-199` — `ClaimDueContext` também poda
  terminais e ignora itens com lease ativo, `NextAttemptAt` futuro ou tentativas
  esgotadas (sem starvation).
- `internal/agent/runtime.go:1297-1331` — o flush é limitado por
  `maxPushOutboxBatch` e por `maxPushOutboxFlushDuration`, verifica
  `ctx.Err()` antes e depois do claim, e registra falha do item quando o DLP
  rejeita o payload antes da entrega.
- Testes **PASS**: `TestPushOutboxEnforcesStorageQuotas`,
  `TestPushOutboxExhaustedRetryDoesNotStarveDueItems`,
  `TestPushOutboxTerminalPruningRespectsActiveLease`,
  `TestPushOutboxConcurrentInstancesPreserveEnqueuesAndClaimsOnce`,
  `TestPushOutboxReturnedDataIsDeepCopied`,
  `TestPushOutboxFencesStaleWorkerCompletionAndFailure`,
  `TestPushOutboxExpiredLeaseFencesCompleteAndFail`,
  `TestPushOutboxFailureCleanupSurvivesCanceledDeliveryContext`,
  `TestPushOutboxCompletionSurvivesCanceledDeliveryContext`,
  `TestPushOutboxCanceledCleanupIsNotBlockedByLocalFileLockWaiter`,
  `TestRuntimeFlushPushOutboxIsBoundedByBatchSize`,
  `TestRuntimeFlushPushOutboxDeliversAndRemovesItem`,
  `TestRuntimeFlushPushOutboxRetainsFailedDelivery`.

### Achado 6 — listagem de jobs não exigia coincidência de dono — **FECHADO**

- `internal/agent/runtime.go:1532-1547` —
  `QueueJobsWithError` só devolve job cujo `mission.OrganizationID` é igual ao
  escopo do runtime **e** cujo `job.OrganizationID` é igual ao da missão; falha ao
  ler a missão exclui o item (fail-closed).
- `internal/agent/runtime.go:1549-1554` — o caminho Redis reusa o mesmo
  pós-filtro (`queueJobsRaw`), então a garantia não depende do backend.
- `internal/agent/runtime.go:1587-1613` — `QueueJobsForOrganization` aplica a
  mesma igualdade nos dois ramos, incluindo organização vazia
  (`ErrQueueJobForbidden`).
- Testes **PASS**: `TestRuntimeQueueJobsOrganizationScope`,
  `TestRuntimeQueueWorkerRejectsOwnerlessAndForeignJobs`.

### Achado 7 — RLS PostgreSQL condicional — **NÃO RESOLVIDO (bloqueio arquitetural mantido)**

- Mantido desabilitado e fail-closed no produto público; nenhuma alegação de
  isolamento corporativo é feita aqui.
- Prerequisito documentado (inalterado): papéis/DSNs distintos para migrador e
  runtime, contexto de tenant não forjável pelo papel de aplicação, grants,
  políticas, backfill, prova com PostgreSQL real sob papel sem superuser/BYPASSRLS
  e revisão independente.

### Recomendação adicional (não bloqueante) — resposta HTTP `200` > 4 MiB truncada — **FECHADO**

- `internal/agent/deploy.go:491-498` — lê `maxDeploymentResponseBytes+1` (4 MiB) e
  devolve erro estável `"deployment provider response exceeded the size limit"`
  em vez de truncar em silêncio.
- `internal/agent/deploy.go:505-511` — JSON malformado → erro; JSON válido que não
  seja objeto → erro.
- `internal/agent/deploy.go:523-525` — resposta sem identidade de deploy
  (`DeploymentID` e `URL` vazios) → erro, nunca sucesso vazio.
- Teste **PASS**: `TestDeploymentRequestRejectsMalformedAndOversizedSuccessfulResponses`
  (subtestes `truncated-json`, `non-object-json`, `oversized`).

## 3. Resultado

- 6 achados bloqueantes + 1 recomendação adicional: **corrigidos e cobertos por
  teste na árvore auditada**.
- 1 bloqueio arquitetural (RLS PostgreSQL) permanece aberto e explicitamente
  desabilitado.
- Nenhum achado novo acionável foi encontrado no caminho dos fluxos reauditados
  (execução de terminal, escopo de listagens MCP/conectores/provedores, DLP,
  outbox, listagem de jobs, resposta HTTP de deploy).
- Esta reauditoria **não** é uma auditoria completa do produto: limita-se aos
  fluxos citados. Persistem as pendências conhecidas e não cobertas aqui
  (jornada E2E UI-first de primeira execução, empacotamento assinado, smoke nativo
  por SO, backlog de paridade da matriz).

## 4. Reprodução

```powershell
$repo='C:\Users\DZ23\Documents\deepseek-harness\default-workspace\hades-ollama'
$env:TMP="$repo\_tmp\testtmp"; $env:TEMP="$repo\_tmp\testtmp"
$env:APPDATA="$repo\_tmp\appdata"; $env:LOCALAPPDATA="$repo\_tmp\localappdata"
Set-Location $repo
go test ./internal/agent -run 'TestTerminalGitIgnoresAmbientPATHReplacement|TestMCPManager|TestRemoteMCP|TestPushOutbox|TestRuntimeQueueJobsOrganizationScope|TestRuntimeQueueWorkerRejectsOwnerlessAndForeignJobs|TestDeployment|TestDLPRedacts|TestRuntimeFlush' -count=1
```
