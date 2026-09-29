# PostgreSQL tenant-HMAC key rotation

**Status:** implemented candidate; not production-ready until full release gates, independent review closure, and isolated operational rehearsal pass.

## Security and availability contract

Tenant contexts are signed as `key_version|organization_id|expires_unix|hmac_hex`; HMAC-SHA256 covers the UTF-8 transcript `key_version\norganization_id\nexpires_unix`. PostgreSQL accepts a signature only when its version equals the single active key version, and the active key is stored in a migrator-owned table that the runtime role cannot read. An old or unconfigured process fails closed after activation; there is no dual-active grace period.

A runtime transaction takes the shared tenant-security advisory lock before attestation and holds it through commit/rollback. Rotation additionally connects as the dedicated admin, commits `NOLOGIN` for `ollama_agent_runtime`, and refuses to proceed unless a cluster-wide `pg_stat_activity` check finds zero runtime sessions. This fences new runtime connections and rejects idle pools, raw SQL clients and non-cooperating clients; the exclusive transaction advisory lock also drains any remaining participant. Only after the key transaction commits does the command restore `LOGIN`. Operators should still stop instances/traffic so they can be restarted coherently with the new version. If the process crashes after fencing, `NOLOGIN` remains and an admin must verify the active key and restore the role deliberately.

## Storage and invariants

The migrator-owned `public.agent_tenant_context_keys` stores `key_version INT PRIMARY KEY`, `secret BYTEA` (at least 32 bytes), and `is_active BOOLEAN`. A partial unique index permits at most one active row; migration/rotation/readiness also require exactly one. Version numbers are positive and strictly increase. The legacy singleton key table is read only during migration from an unversioned installation, imported as version 1 after constant-time comparison, then removed in the same schema transaction; import rejects any schema other than the supported two-column shape and requires exactly one valid row. A fresh, genuinely empty keyring may bootstrap exactly once at version 1. After successful rotation, older inactive keys remain stored as rollback/audit history only: runtime accepts only the single active version, and an inactive version is never a valid fallback. A keyring with zero active rows, or a recorded high-water version above the active version, fails closed. No routine migration may replace or downgrade a key.

The runtime has only DML on mission/event tables and EXECUTE on the verifier; it has no table privilege on the versioned key table. The SECURITY DEFINER verifier pins `search_path`, parses and bounds all four fields, and selects the secret by both version and `is_active`.

## Rotation transaction

`ollama agent rotate-postgres-key` reads `OLLAMA_AGENT_POSTGRES_ADMIN_DATABASE_URL`, the migrator DSN, current version/key, and next version/key only from environment/secret injection; values are never command-line arguments or logs. It verifies both roles, disables runtime LOGIN, requires no runtime sessions, locks and validates the keyring and high-water mark, verifies the supplied current secret with constant-time comparison, inserts the next inactive row, flips old inactive/new active, verifies exactly one active row, commits, and restores runtime LOGIN. Any pre-commit error rolls back keyring state and attempts to restore LOGIN; if the process dies or login restoration fails, the role remains safely fenced and requires operator recovery. An exact retry after success is rejected as stale-current configuration.

Normal migration takes the configured current version/key and verifies an exact active-row match; it never rotates. Runtime startup takes the same version/key and proves a signed probe through the live verifier. Changing the secret without matching the database is rejected.

## Operator sequence and rollback

1. Back up both old and new secrets in the approved secret manager; record active version; test access to the new secret.
2. Scale application/runtime instances to zero, stop queue workers, and reject/drain traffic. The CLI independently fences `ollama_agent_runtime` with `NOLOGIN` and aborts if any database session remains.
3. Inject the admin DSN as well as migrator DSN, and run the rotation CLI once with current and next secrets/versions. A bounded timeout aborts without changing the active key if the advisory-lock drain does not complete; confirm runtime LOGIN was safely restored before retrying.
4. Deploy the runtime with the new key and version, run a tenant-scoped smoke test, then restore traffic/workers.
5. If the new deployment fails, quiesce again and run a new monotonic rotation to the prior secret under a **higher** version (for example, active 2 -> restore old secret as version 3). Never decrement/reuse a retired version; the source secret remains in the secret manager.
6. Retire/remove inactive database key rows only through a separately reviewed maintenance change after rollback window and incident-retention requirements are satisfied. Never delete the active row. This initial implementation does not automatically prune historical rows.

If runtime sessions remain, the exclusive-lock wait times out, active version/secret does not exactly match the supplied current values, keyring history is invalid, or the next version is not above the recorded high-water mark, abort and preserve current keyring state. No silent secret replacement, runtime dual-key acceptance, guessed rollback, or online fleet rotation is permitted. For new databases, install `pgcrypto` as the bootstrap/superuser before running the migrator; `deploy/postgres/init/001-agent-role.sql` does this on fresh Compose volumes. Legacy cutovers transfer legacy ownership via `REASSIGN OWNED` before retirement.

## Required evidence

Live PostgreSQL integration must prove: legacy singleton import as version 1; active-version signing/readiness; zero-active keyring fails closed; ordinary-migration mismatch refusal; rotation blocks when a runtime session remains or an advisory-lock holder blocks; successful atomic activation; old runtime fail-closed/new runtime success; concurrent rotations serialize and stale current credentials fail; high-water non-reuse; active-row uniqueness and runtime secret-read denial; every failed rotation leaves the active version unchanged; old-secret rollback uses a higher version. Full release gates and independent review remain mandatory.
