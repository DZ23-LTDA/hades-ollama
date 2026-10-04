#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

required_files=(
  "UPSTREAM_BASE_COMMIT"
	  "docs/agentic/PRODUCT_TREE.md"
	  "docs/agentic/PARITY_MATRIX.md"
	  "audit/HARNESS_CAPABILITY_MATRIX.md"
		  "docs/agentic/COMPANY_OS.md"
	  "docs/agentic/DESKTOP_COMMANDER_REMOTE.md"
	  "docs/agentic/COMPOSIO.md"
	  "docs/agentic/XAI_GROK.md"
	  "docs/agentic/EVALUATION.md"
	  "examples/dz23-composio-connect.json"
	  "examples/dz23-xai.json"
	  "internal/agent/runtime.go"
	  "internal/agent/capability_policy.go"
	  "internal/agent/capability_policy_test.go"
	  "internal/agent/company.go"
	  "internal/agent/company_growth.go"
	  "internal/agent/company_social.go"
	  "internal/agent/company_agents.go"
	  "internal/agent/company_test.go"
	  "internal/agent/builder.go"
	  "internal/agent/builder_test.go"
	  "internal/agent/swarm.go"
	  "internal/agent/traces.go"
	  "internal/agent/devices.go"
	  "internal/agent/traces_test.go"
	  "internal/agent/evaluation.go"
	  "internal/agent/browser_helper.py"
	  "internal/agent/connectors.go"
	  "internal/agent/connectors_test.go"
	  "internal/agent/mcp_remote.go"
	  "internal/agent/mcp_remote_test.go"
	  "internal/agent/mcp.go"
	  "internal/agent/sandbox_linux.go"
	  "internal/agent/sandbox_other.go"
	  "internal/agent/sandbox_seccomp_linux.go"
	  "internal/agent/sandbox_seccomp_other.go"
	  "internal/agent/plugin_lifecycle_test.go"
	  "internal/grok/client.go"
	  "internal/grok/live.go"
	  "internal/multillm/router.go"
	  "internal/multillm/registry.go"
	  "server/agent_routes.go"
	  "server/grok_routes.go"
	  "server/plugin_routes.go"
	  "server/plugin_routes_test.go"
	  "server/agent_mcp_bootstrap_test.go"
	  "server/builder_scope_test.go"
	  "server/p0_scope_test.go"
	  "server/company_routes.go"
	  "server/company_growth_routes.go"
	  "apps/mobile-agentic/App.tsx"
	  "apps/mobile-agentic/package.json"
	  ".github/workflows/release.yaml"
  "app/ui/app/src/components/AppSidebar.tsx"
  "app/ui/app/src/components/AgenticControlCenter.tsx"
  "app/ui/app/src/routes/agentic.tsx"
  "app/ui/app/src/routes/projects.tsx"
  "app/ui/app/src/routes/library.tsx"
  "app/ui/app/src/routes/scheduled.tsx"
  "app/ui/app/src/routes/skills.tsx"
  "app/ui/app/src/routes/plugins.tsx"
  "app/ui/app/src/routes/tasks.tsx"
  "app/ui/app/src/routes/company.tsx"
	  "app/ui/app/src/components/CompanyGrowthPanel.tsx"
	  "app/ui/app/src/components/CompanyOperationsPanel.tsx"
  "scripts/smoke-company-growth.sh"
  "scripts/smoke-builder.sh"
)

for file in "${required_files[@]}"; do
  test -s "$file" || { echo "missing required Ollama Full surface: $file" >&2; exit 1; }
done

grep -q '^policy=manual-review-only$' UPSTREAM_BASE_COMMIT
grep -q 'allow_insecure_loopback' internal/multillm/registry.go
grep -q 'safeConfig' server/agent_routes.go
grep -q 'Nova tarefa' app/ui/app/src/components/AppSidebar.tsx
grep -q 'Agentic Control Center' app/ui/app/src/components/AgenticControlCenter.tsx
grep -q 'RemoteMCP' internal/agent/runtime.go
grep -q 'mcp.remote.call' internal/agent/mcp_remote.go
grep -q 'HeadersEnv' internal/agent/mcp_remote.go
grep -q 'api.x.ai/v1' examples/dz23-xai.json
grep -q 'Growth OS' app/ui/app/src/components/CompanyGrowthPanel.tsx
grep -q 'addCompanyCampaign' server/company_growth_routes.go
grep -q 'grokResponses' server/grok_routes.go
grep -q 'SetConnectorEnabled' internal/agent/runtime.go
grep -q 'connectorPathMatches' internal/agent/connectors.go
grep -q 'validRemoteMCPHeaderName' internal/agent/mcp_remote.go
grep -q 'GetForOrganization' internal/agent/builder.go
grep -q 'CreateRequest' internal/agent/company.go
grep -q 'ErrCompanyAgentBudgetExceeded' internal/agent/company_agents.go
grep -q 'PlanForOrganization' internal/agent/swarm.go
grep -q 'ListForOrganization' internal/agent/traces.go
grep -q 'HeartbeatForOrganization' internal/agent/devices.go
grep -q 'DecideApprovalForActorCAS' internal/agent/runtime.go
grep -q 'requireApprovalApprover' server/agent_routes.go
grep -q 'nonce' app/ui/app/src/components/AgenticConsole.tsx
grep -q 'type CompanyApproval struct' internal/agent/company.go
grep -q 'DecideApproval' server/company_approval_routes.go
grep -q 'PendingApproval' internal/agent/company.go
grep -q 'CompanyCampaignApprovalHTTPUsesNonceAndOrganization' server/company_approval_test.go
grep -q 'RecordSpendRequest' internal/agent/company.go
grep -q 'decideCompanyApprovalByID' server/company_approval_routes.go
grep -q 'CompanyApprovalQueue' app/ui/app/src/components/CompanyApprovalQueue.tsx
grep -q 'func RedactValue' internal/agent/secrets.go
grep -q 'TestRuntimeRedactsStepResultsEventsTracesAndPersistence' internal/agent/runtime_test.go
grep -q 'redactMissionForPersistence' internal/agent/store.go
grep -q 'remote MCP destination resolves to a private address' internal/agent/mcp_remote.go
grep -q 'TestRemoteMCPDialRejectsPrivateActualAddress' internal/agent/mcp_remote_test.go
grep -q 'MCP command must be an absolute executable path' internal/agent/mcp.go
	grep -q 'TestMCPPayloadLimitAndCancellationRestart' internal/agent/mcp_test.go
	grep -q 'TestMCPNotificationsDoNotBreakResponseCorrelation' internal/agent/mcp_test.go
	grep -q 'TestLoadAgentMCPBootstrapsStrictManifest' server/agent_mcp_bootstrap_test.go
	grep -q 'TestLoadAgentRemoteMCPRejectsEmptyAllowlist' server/agent_mcp_bootstrap_test.go
	grep -q 'configureMCPProcess' internal/agent/mcp_process_unix.go
	grep -q 'browserPythonExecutable' internal/agent/browser.go
	grep -q "playwright==1.63.0" .github/workflows/test.yaml
		grep -q 'group: \${{ github.workflow }}-\${{ github.event.pull_request.number || github.run_id }}' .github/workflows/test.yaml
	grep -q 'workflow_dispatch:' .github/workflows/test.yaml
	grep -q 'run_native_matrix:' .github/workflows/test.yaml
	grep -q 'github.event_name == '\''workflow_dispatch'\'' && inputs.run_native_matrix == true' .github/workflows/test.yaml
	grep -q 'media destination resolves to a private address' internal/agent/media.go
grep -q 'validateMediaMagic' internal/agent/media.go
grep -q 'TestMediaMaterializeRejectsRedirectAndInvalidMagic' internal/agent/media_test.go
grep -q 'connector response payload exceeds limit' internal/agent/connectors.go
grep -q 'TestConnectorEgressBlocksRedirectsAndBoundsPayloads' internal/agent/connectors_test.go
grep -q 'ValidateRedirectURI' internal/agent/auth.go
grep -q 'RedirectURIs' internal/agent/auth.go
grep -q 'TestOAuthProviderRedirectAllowlist' internal/agent/auth_test.go
grep -q 'setAgentSession' app/ui/app/src/lib/agenticClient.ts
grep -q 'agenticClient.security.test' app/ui/app/src/lib/agenticClient.security.test.ts || grep -q 'keeps the bearer only in memory' app/ui/app/src/lib/agenticClient.security.test.ts
grep -q 'Apagar sessão local' apps/mobile-agentic/App.tsx
grep -q 'oauth destination connected to a private address' internal/agent/auth.go
grep -q 'validateOAuthEndpointURL' internal/agent/auth.go
grep -q 'TestOAuthClientBlocksRedirectAndPrivateActualAddress' internal/agent/auth_test.go
grep -q 'runToolCommand' internal/agent/tools.go
grep -q 'ulimit -t 55' internal/agent/tools.go
grep -q 'TestRunToolCommandKillsProcessGroupOnCancellation' internal/agent/tools_test.go
grep -q 'best-effort-unshare' internal/agent/tools.go
grep -q 'ErrPluginOrganizationScope' internal/agent/plugin_scope.go
grep -q 'SetEnabledForOrganization' internal/agent/connectors.go
grep -q 'ListForOrganization' internal/agent/mcp.go
grep -q 'SkillsForOrganization' internal/agent/context.go
grep -q 'TestPluginManagersEnforceOrganizationOwnership' internal/agent/plugin_scope_test.go
grep -q 'pluginCatalog' server/plugin_routes.go
grep -q 'npm test -- --run' .github/workflows/dz23-agentic-quality.yaml
grep -q 'npm run build' .github/workflows/dz23-agentic-quality.yaml
grep -q 'CGO_ENABLED=1 go vet ./...' .github/workflows/dz23-agentic-quality.yaml
grep -q 'needs: \[release-gate, darwin-build, windows-app, docker-merge-push, quality\]' .github/workflows/release.yaml
grep -q 'ErrModelNotAllowed' internal/grok/client.go
grep -q 'validateCatalogModel' internal/grok/client.go
grep -q 'Grok streaming is not exposed' server/grok_routes.go
grep -q 'TestGrokResponsesRejectsStreamBeforeUpstream' server/grok_routes_test.go
grep -q '127.0.0.1:' deploy/docker-compose.agentic.yml
grep -q 'OLLAMA_AGENT_POSTGRES_ADMIN_PASSWORD' deploy/docker-compose.agentic.yml
grep -q 'OLLAMA_AGENT_RUNTIME_PASSWORD' deploy/docker-compose.agentic.yml
grep -q 'OLLAMA_AGENT_MIGRATOR_PASSWORD' deploy/docker-compose.agentic.yml
grep -q 'OLLAMA_AGENT_REDIS_PASSWORD' deploy/docker-compose.agentic.yml
grep -q 'OLLAMA_AGENT_TEST_POSTGRES_RUNTIME_URL' .github/workflows/dz23-agentic-quality.yaml
grep -q 'OLLAMA_AGENT_TEST_POSTGRES_MIGRATOR_URL' .github/workflows/dz23-agentic-quality.yaml
grep -q 'OLLAMA_AGENT_TEST_POSTGRES_TENANT_KEY' .github/workflows/dz23-agentic-quality.yaml
grep -q 'active legacy ollama_agent login' internal/agent/postgres_security.go
grep -q 'unexpected columns' internal/agent/postgres_security.go
grep -q 'legacy tenant key table has unsupported columns' internal/agent/postgres_security.go
grep -q 'legacy_key_rows<>1' internal/agent/postgres_security.go
grep -q 'dedicated PostgreSQL cluster' deploy/postgres/migrate-existing-roles.sql
grep -q 'public schema contains unrelated non-extension objects' deploy/postgres/migrate-existing-roles.sql
grep -q 'legacy role owns large objects outside the supported Ollama schema' deploy/postgres/migrate-existing-roles.sql
grep -q 'agent_tenant_context_matches' deploy/postgres/migrate-existing-roles.sql
grep -q 'ALTER ROLE ollama_agent NOLOGIN NOSUPERUSER' deploy/postgres/migrate-existing-roles.sql
grep -q 'WITH RECURSIVE legacy_members(member_oid)' deploy/postgres/migrate-existing-roles.sql
grep -q "SET LOCAL lock_timeout = '5s'" deploy/postgres/migrate-existing-roles.sql
grep -q 'drain_post_transfer_legacy_sessions' deploy/postgres/migrate-existing-roles.sql
grep -q 'WITH RECURSIVE legacy_members(member_oid)' deploy/postgres/retire-legacy-role.sql
grep -q "SET LOCAL lock_timeout = '5s'" deploy/postgres/retire-legacy-role.sql
grep -q 'ollama_cutover_unrelated_admin_probe' .github/workflows/dz23-agentic-quality.yaml
grep -q 'pg_catalog.pg_proc p JOIN pg_catalog.pg_roles r ON r.oid=p.proowner' .github/workflows/dz23-agentic-quality.yaml
grep -q 'TestDistributedPostgresRuntimeCannotConnectOutsideApplicationDatabase' internal/agent/distributed_integration_test.go
grep -q 'OLLAMA_AGENT_TEST_POSTGRES_ADMIN_OTHER_DB_URL' internal/agent/distributed_integration_test.go
grep -q "d.classid='pg_catalog.pg_database'::pg_catalog.regclass" deploy/postgres/migrate-existing-roles.sql
grep -q 'hba_file=/etc/postgresql/pg_hba.conf' deploy/docker-compose.agentic.yml
grep -q '\${OLLAMA_AGENT_POSTGRES_HBA_FILE:-./postgres/pg_hba-runtime.conf}:/etc/postgresql/pg_hba.conf:ro' deploy/docker-compose.agentic.yml
grep -qE '^host[[:space:]]+all[[:space:]]+ollama_agent_runtime.*reject' deploy/postgres/pg_hba-runtime.conf
grep -qE '^local[[:space:]]+all[[:space:]]+ollama_agent_runtime.*reject' deploy/postgres/pg_hba-runtime.conf
grep -qE '^host[[:space:]]+ollama_agent[[:space:]]+ollama_agent_runtime.*scram-sha-256' deploy/postgres/pg_hba-runtime.conf
if grep -qE '^host[[:space:]]+all[[:space:]]+ollama_agent_runtime.*scram-sha-256' deploy/postgres/pg_hba-runtime.conf; then
  echo 'runtime HBA allow must not precede and shadow database-isolation rejects' >&2
  exit 1
fi
grep -qE '^host[[:space:]]+ollama_agent[[:space:]]+ollama_agent_migrator.*scram-sha-256' deploy/postgres/pg_hba-runtime.conf
if grep -qE '^host[[:space:]]+all[[:space:]]+ollama_agent_migrator.*scram-sha-256' deploy/postgres/pg_hba-runtime.conf; then
  echo 'migrator HBA allow must not precede and shadow database-isolation rejects' >&2
  exit 1
fi
test "$(grep -cE '^host[[:space:]]+ollama_agent[[:space:]]+ollama_agent_runtime[[:space:]]+(0\.0\.0\.0/0|::/0)[[:space:]]+scram-sha-256$' deploy/postgres/pg_hba-runtime.conf)" -eq 2
test "$(grep -cE '^host[[:space:]]+ollama_agent[[:space:]]+ollama_agent_migrator[[:space:]]+(0\.0\.0\.0/0|::/0)[[:space:]]+scram-sha-256$' deploy/postgres/pg_hba-runtime.conf)" -eq 2
runtime_allow_line="$(grep -nE '^host[[:space:]]+ollama_agent[[:space:]]+ollama_agent_runtime[[:space:]]+0\.0\.0\.0/0[[:space:]]+scram-sha-256$' deploy/postgres/pg_hba-runtime.conf | cut -d: -f1)"
runtime_reject_line="$(grep -nE '^host[[:space:]]+all[[:space:]]+ollama_agent_runtime[[:space:]]+0\.0\.0\.0/0[[:space:]]+reject$' deploy/postgres/pg_hba-runtime.conf | cut -d: -f1)"
migrator_allow_line="$(grep -nE '^host[[:space:]]+ollama_agent[[:space:]]+ollama_agent_migrator[[:space:]]+0\.0\.0\.0/0[[:space:]]+scram-sha-256$' deploy/postgres/pg_hba-runtime.conf | cut -d: -f1)"
migrator_reject_line="$(grep -nE '^host[[:space:]]+all[[:space:]]+ollama_agent_migrator[[:space:]]+0\.0\.0\.0/0[[:space:]]+reject$' deploy/postgres/pg_hba-runtime.conf | cut -d: -f1)"
test "$runtime_allow_line" -lt "$runtime_reject_line"
test "$migrator_allow_line" -lt "$migrator_reject_line"
grep -q 'ollama_agent_reverse_member.*scram-sha-256' .github/workflows/dz23-agentic-quality.yaml
grep -q 'reverse_member_login' .github/workflows/dz23-agentic-quality.yaml
grep -q 'REVOKE CONNECT ON DATABASE ollama_agent FROM PUBLIC' deploy/postgres/init/001-agent-role.sql
grep -q 'REVOKE USAGE, CREATE ON SCHEMA public FROM PUBLIC' deploy/postgres/init/001-agent-role.sql
grep -q 'aclexplode(col.attacl)' internal/agent/postgres_security.go
grep -q 'public-database-create-temp-granted' internal/agent/distributed_integration_test.go
grep -q 'keyring-secret-column-granted' internal/agent/distributed_integration_test.go
grep -q 'GRANT SELECT (objective) ON TABLE public.agent_missions TO PUBLIC' .github/workflows/dz23-agentic-quality.yaml
grep -q 'TestDistributedPostgresRuntimeReadinessForLegacyRole' internal/agent/distributed_integration_test.go
grep -q 'OLLAMA_AGENT_TEST_EXPECT_LEGACY_ROLE_ACTIVE' .github/workflows/dz23-agentic-quality.yaml
grep -q 'pg_restore --list' docs/agentic/INTEGRATIONS.md
grep -q 'SOURCE_PGSSLROOTCERT' docs/agentic/INTEGRATIONS.md
grep -q 'STAGING_PGSSLROOTCERT' docs/agentic/INTEGRATIONS.md
grep -q 'sslmode=verify-full' docs/agentic/INTEGRATIONS.md
grep -q 'PGSSLROOTCERT="$STAGING_PGSSLROOTCERT" pg_restore' docs/agentic/INTEGRATIONS.md
grep -q 'if \[ "$source_cluster_id" = "$staging_cluster_id" \]' docs/agentic/INTEGRATIONS.md
grep -q 'POSTGRES_HA_FAILOVER_REHEARSAL.md' docs/agentic/INTEGRATIONS.md
grep -q 'pg_basebackup' docs/agentic/POSTGRES_HA_FAILOVER_REHEARSAL.md
grep -q 'pg_promote' docs/agentic/POSTGRES_HA_FAILOVER_REHEARSAL.md
grep -qi 'split[- ]brain' docs/agentic/POSTGRES_HA_FAILOVER_REHEARSAL.md
grep -q 'restore_check' .github/workflows/dz23-agentic-quality.yaml
grep -q 'cleanup-placeholder' .github/workflows/dz23-agentic-quality.yaml
if sed -n '65,210p' .github/workflows/dz23-agentic-quality.yaml | grep -q 'GITHUB_ENV'; then
  echo 'distributed integration secrets must remain step-local' >&2
  exit 1
fi
if grep -Rqi 'change-me-local-only' deploy; then echo 'fixed development credential found'; exit 1; fi
grep -Eq 'Provider[[:space:]]+string' internal/agent/types.go
grep -q 'provider != "ollama-local"' internal/agent/runtime.go
	grep -q 'const providerChoices = useMemo' app/ui/app/src/components/AgenticConsole.tsx
	grep -q 'getModels("")' app/ui/app/src/components/AgenticConsole.tsx
	if grep -q 'Claude / Anthropic (não conectado)' app/ui/app/src/components/AgenticConsole.tsx; then
		echo "stale hardcoded provider alias detected" >&2
		exit 1
	fi
grep -q 'TestRuntimeRejectsUnconfiguredMissionProvider' internal/agent/runtime_test.go
		grep -q 'Empty release artifact' .github/workflows/release.yaml
	grep -q '#define MyAppName "Hades"' app/ollama.iss
	if grep -q 'Ollama Classe A+' app/ollama.iss; then
		echo 'legacy installer branding must not return' >&2
		exit 1
	fi
			grep -q 'actions/attest-build-provenance@96b4a1ef7235a096b17240c259729fdd70c83d45' .github/workflows/release.yaml
	grep -q 'OLLAMA_ENABLE_ATTESTATIONS' .github/workflows/release.yaml
	grep -q 'ollama-classe-a-plus-sbom.cdx.json' .github/workflows/release.yaml
	grep -q 'sha256sum -c sha256sum.txt' .github/workflows/release.yaml
grep -q 'Sign release checksum manifest' .github/workflows/release.yaml
grep -q 'release-signing-public.pem' .github/workflows/release.yaml
grep -q 'verify-release-artifact.sh' .github/workflows/release.yaml
	grep -q 'release-metadata.json' .github/workflows/release.yaml
	grep -q 'workflow_dispatch:' .github/workflows/latest.yaml
	grep -q "vars.OLLAMA_ENABLE_LATEST == 'true'" .github/workflows/latest.yaml
	if grep -q '^  release:' .github/workflows/latest.yaml; then
		echo "latest Docker publication must not be release-triggered in the fork" >&2
		exit 1
	fi
	grep -q 'local/ollama-full' scripts/env.sh
	grep -q 'refusing to publish an upstream or local placeholder image' scripts/tag_latest.sh
	grep -q 'operator-owned registry' scripts/build_docker.sh
		grep -q 'strictSandboxLauncher' internal/agent/sandbox_seccomp_linux.go
	grep -q 'OLLAMA_AGENT_SANDBOX_CGROUP_ROOT' internal/agent/sandbox_linux.go
	grep -q 'killSandboxControl' internal/agent/tools.go
	grep -q 'agentOriginAllowed' server/agent_routes.go
	grep -q 'sandbox_strict_cgroup_configured' server/agent_routes.go
	grep -q 'queueStorageKey' apps/mobile-agentic/App.tsx
	grep -q 'auth/session' apps/mobile-agentic/App.tsx
	grep -q '"overrides"' apps/mobile-agentic/package.json
	grep -q '"image-size": "2.0.4"' apps/mobile-agentic/package.json
	grep -q '"postcss": "8.5.28"' apps/mobile-agentic/package.json
	grep -q '"uuid": "11.1.1"' apps/mobile-agentic/package.json
grep -q 'QueueJobsForOrganization' internal/agent/runtime.go
grep -q 'ReplayJobForOrganization' internal/agent/runtime.go
grep -q 'TestRuntimeQueueJobsOrganizationScope' internal/agent/runtime_test.go
grep -q 'cross-tenant job replay' server/p0_scope_test.go
grep -q 'rejectSymlinkComponents(root, candidate)' internal/agent/artifacts.go
grep -q 'TestBuildArtifactManifestRejectsSymlinkOutsideWorkspace' internal/agent/artifacts_test.go
	grep -q 'Capabilities' internal/agent/runtime.go
	grep -q 'ErrUnknownCapability' internal/agent/capability_policy.go
	grep -q 'ValidateToolDescriptor' internal/agent/capability_policy.go
	grep -q 'capabilities = \[\]string{"workspace:read"\}' internal/agent/runtime.go
	grep -q 'Permitir escrita' app/ui/app/src/components/AgenticConsole.tsx
	grep -q 'R44' audit/HARNESS_CAPABILITY_MATRIX.md
	grep -q 'G13' audit/HARNESS_CAPABILITY_MATRIX.md
	grep -q 'Evaluation' internal/agent/evaluation.go
for route in projects library scheduled skills plugins tasks company; do
  grep -q "routes/${route}" app/ui/app/src/routeTree.gen.ts || {
    echo "route tree is missing /${route}" >&2
    exit 1
  }
done

if git ls-files | grep -E '(^|/)(\.env|.*\.key|.*\.pem|node_modules/)' >/dev/null; then
  echo "tracked secret or generated dependency path detected" >&2
  exit 1
fi

git diff --check
node scripts/verify-design-system.mjs
echo "Verifying frontend-backend contract matrix..."
node scripts/verify-contracts.mjs

printf '%s\n' "Ollama Full integrity guard: PASS"
