package agent

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	postgresTenantContextGUC                  = "app.tenant_context"
	postgresTenantContextTTL                  = 5 * time.Minute
	postgresRuntimeRole                       = "ollama_agent_runtime"
	postgresTenantSecurityMigrationLock int64 = 0x4f4c4c414d414655
)

var (
	postgresOrganizationIDPattern  = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)
	ErrPostgresTenantSecurityShape = errors.New("PostgreSQL tenant security shape is invalid")
	ErrPostgresLegacyRoleActive    = errors.New("active legacy ollama_agent login, privilege, membership, or session detected")
)

const postgresTenantContextFunctionSource = `DECLARE
  context_value TEXT;
  context_parts TEXT[];
  context_org TEXT;
  context_expiry TEXT;
  context_signature TEXT;
  context_secret BYTEA;
  expiry_epoch BIGINT;
  expected_signature TEXT;
BEGIN
  context_value := current_setting('app.tenant_context', TRUE);
  IF context_value IS NULL OR length(context_value) > 512 THEN RETURN FALSE; END IF;
  context_parts := string_to_array(context_value, '|');
  IF array_length(context_parts, 1) IS DISTINCT FROM 3 THEN RETURN FALSE; END IF;
  context_org := context_parts[1];
  context_expiry := context_parts[2];
  context_signature := context_parts[3];
  IF context_org IS NULL OR context_expiry IS NULL OR context_signature IS NULL THEN RETURN FALSE; END IF;
  IF requested_org IS NULL OR requested_org = '' OR context_org <> requested_org OR context_org !~ '^[A-Za-z0-9_.-]{1,128}$' THEN RETURN FALSE; END IF;
  IF context_expiry !~ '^[0-9]{1,12}$' OR context_signature !~ '^[0-9a-f]{64}$' THEN RETURN FALSE; END IF;
  expiry_epoch := context_expiry::BIGINT;
  IF expiry_epoch < floor(extract(epoch FROM statement_timestamp()))::BIGINT OR expiry_epoch > floor(extract(epoch FROM statement_timestamp()))::BIGINT + 300 THEN RETURN FALSE; END IF;
  SELECT secret INTO context_secret FROM public.agent_tenant_context_key WHERE key_id = TRUE;
  IF context_secret IS NULL THEN RETURN FALSE; END IF;
  expected_signature := encode(public.hmac(convert_to(context_org || E'\n' || context_expiry, 'UTF8'), context_secret, 'sha256'), 'hex');
  RETURN expected_signature = context_signature;
EXCEPTION WHEN OTHERS THEN
  RETURN FALSE;
END;`

var postgresExpectedTenantPolicies = map[string][4]string{
	"agent_missions": {
		"agent_missions_tenant_policy", "ALL",
		"((organization_id<>''::text)ANDagent_tenant_context_matches(organization_id))",
		"((organization_id<>''::text)ANDagent_tenant_context_matches(organization_id))",
	},
	"agent_events": {
		"agent_events_tenant_policy", "ALL",
		"((organization_id<>''::text)ANDagent_tenant_context_matches(organization_id)AND(EXISTS(SELECT1FROMagent_missionsmWHERE((m.id=agent_events.mission_id)AND(m.organization_id=agent_events.organization_id)))))",
		"((organization_id<>''::text)ANDagent_tenant_context_matches(organization_id)AND(EXISTS(SELECT1FROMagent_missionsmWHERE((m.id=agent_events.mission_id)AND(m.organization_id=agent_events.organization_id)))))",
	},
}

func verifyPostgresTenantSecurityShape(ctx context.Context, tx *sql.Tx) error {
	// Supported migrations take this lock exclusively. Hold its shared form for
	// the entire tenant transaction so function/policy DDL cannot race attestation.
	if _, err := tx.ExecContext(ctx, `SELECT pg_catalog.pg_advisory_xact_lock_shared($1)`, postgresTenantSecurityMigrationLock); err != nil {
		return fmt.Errorf("lock PostgreSQL tenant security definition: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `LOCK TABLE public.agent_missions, public.agent_events IN ACCESS SHARE MODE`); err != nil {
		return fmt.Errorf("lock PostgreSQL tenant tables for security-shape verification: %w", err)
	}
	var tableCount int
	var tablesSecure, ownersSecure bool
	if err := tx.QueryRowContext(ctx, `
SELECT count(*), bool_and(c.relrowsecurity AND c.relforcerowsecurity),
       bool_and(owner.rolname = 'ollama_agent_migrator')
	  FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
	  JOIN pg_catalog.pg_roles owner ON owner.oid=c.relowner
 WHERE n.nspname='public' AND c.relname IN ('agent_missions','agent_events') AND c.relkind IN ('r','p')`).Scan(&tableCount, &tablesSecure, &ownersSecure); err != nil {
		return fmt.Errorf("verify PostgreSQL tenant table security flags: %w", err)
	}
	if tableCount != 2 || !tablesSecure || !ownersSecure {
		return errors.New("PostgreSQL tenant tables must exist, be owned by ollama_agent_migrator, and have ENABLE plus FORCE ROW LEVEL SECURITY")
	}
	for table, expected := range postgresExpectedTenantPolicies {
		var policyCount, exactCount int
		err := tx.QueryRowContext(ctx, `
SELECT count(*), count(*) FILTER (WHERE policyname=$2 AND cmd=$3 AND permissive='PERMISSIVE'
 AND roles=ARRAY['public']::name[]
 AND regexp_replace(coalesce(qual,''), '[[:space:]]+', '', 'g')=$4
 AND regexp_replace(coalesce(with_check,''), '[[:space:]]+', '', 'g')=$5)
	  FROM pg_catalog.pg_policies WHERE schemaname='public' AND tablename=$1`, table, expected[0], expected[1], expected[2], expected[3]).Scan(&policyCount, &exactCount)
		if err != nil {
			return fmt.Errorf("verify PostgreSQL %s row-level security policy: %w", table, err)
		}
		if policyCount != 1 || exactCount != 1 {
			return fmt.Errorf("PostgreSQL %s tenant RLS policy is absent or differs from the required fail-closed definition", table)
		}
	}
	var functionSecure bool
	err := tx.QueryRowContext(ctx, `
SELECT p.prosecdef AND p.provolatile='s' AND p.prorettype='boolean'::pg_catalog.regtype AND p.pronargs=1
   AND owner.rolname='ollama_agent_migrator'
   AND p.proconfig=ARRAY['search_path=pg_catalog, public']::text[]
   AND regexp_replace(p.prosrc, '[[:space:]]+', '', 'g')=regexp_replace($1, '[[:space:]]+', '', 'g')
	  FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_roles owner ON owner.oid=p.proowner
 WHERE p.oid='public.agent_tenant_context_matches(text)'::pg_catalog.regprocedure`, postgresTenantContextFunctionSource).Scan(&functionSecure)
	if err != nil {
		return fmt.Errorf("verify PostgreSQL signed-tenant verifier definition: %w", err)
	}
	if !functionSecure {
		return errors.New("PostgreSQL signed-tenant verifier must be the migrator-owned SECURITY DEFINER function with pinned search_path and the approved HMAC implementation")
	}
	var runtimePrivilegesSecure bool
	if err := tx.QueryRowContext(ctx, `
SELECT r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolbypassrls AND NOT r.rolcreatedb
       AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolinherit
       AND NOT pg_catalog.has_schema_privilege(r.rolname,'public','CREATE')
       AND NOT pg_catalog.has_database_privilege(r.rolname,pg_catalog.current_database(),'CREATE')
       AND NOT pg_catalog.has_database_privilege(r.rolname,pg_catalog.current_database(),'TEMP')
       AND NOT pg_catalog.has_table_privilege(r.rolname,'public.agent_tenant_context_key','SELECT')
       AND pg_catalog.has_table_privilege(r.rolname,'public.agent_missions','SELECT')
       AND pg_catalog.has_table_privilege(r.rolname,'public.agent_missions','INSERT')
       AND pg_catalog.has_table_privilege(r.rolname,'public.agent_missions','UPDATE')
       AND pg_catalog.has_table_privilege(r.rolname,'public.agent_missions','DELETE')
       AND pg_catalog.has_table_privilege(r.rolname,'public.agent_events','SELECT')
       AND pg_catalog.has_table_privilege(r.rolname,'public.agent_events','INSERT')
       AND pg_catalog.has_table_privilege(r.rolname,'public.agent_events','UPDATE')
       AND pg_catalog.has_table_privilege(r.rolname,'public.agent_events','DELETE')
       AND NOT pg_catalog.has_table_privilege(r.rolname,'public.agent_missions','TRUNCATE')
       AND NOT pg_catalog.has_table_privilege(r.rolname,'public.agent_missions','REFERENCES')
       AND NOT pg_catalog.has_table_privilege(r.rolname,'public.agent_missions','TRIGGER')
       AND NOT pg_catalog.has_table_privilege(r.rolname,'public.agent_events','TRUNCATE')
       AND NOT pg_catalog.has_table_privilege(r.rolname,'public.agent_events','REFERENCES')
       AND NOT pg_catalog.has_table_privilege(r.rolname,'public.agent_events','TRIGGER')
       AND pg_catalog.has_function_privilege(r.rolname,'public.agent_tenant_context_matches(text)','EXECUTE')
       AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid OR m.roleid=r.oid)
  FROM pg_catalog.pg_roles r WHERE r.rolname='ollama_agent_runtime'`).Scan(&runtimePrivilegesSecure); err != nil {
		return fmt.Errorf("verify PostgreSQL runtime privilege set: %w", err)
	}
	if !runtimePrivilegesSecure {
		return errors.New("PostgreSQL runtime role privileges differ from the required least-privilege tenant DML set")
	}
	return nil
}

// OpenPostgresRuntimeStore opens a tenant-only store. Its role must be a
// non-owner, NOSUPERUSER/NOBYPASSRLS role with DML privileges only. The key
// must match the secret provisioned by MigratePostgresAgentSchema.
func OpenPostgresRuntimeStore(ctx context.Context, dsn string, tenantContextKey []byte) (*PostgresStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("postgres runtime DSN is required")
	}
	if len(tenantContextKey) < 32 {
		return nil, errors.New("postgres tenant context key must be at least 32 bytes")
	}
	if ctx == nil {
		return nil, errors.New("postgres runtime context is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	store := &PostgresStore{db: db, timeout: 10 * time.Second, tenantContextKey: append([]byte(nil), tenantContextKey...)}
	pingCtx, cancel := context.WithTimeout(ctx, store.timeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect PostgreSQL runtime role: %w", err)
	}
	if err := store.verifyRuntimeRoleAndContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

// OpenPostgresStore is intentionally disabled: callers must choose the
// explicit runtime or migration entrypoint so startup cannot silently DDL.
func OpenPostgresStore(context.Context, string) (*PostgresStore, error) {
	return nil, ErrPostgresTenantIsolationUnavailable
}

func validatePostgresOrganizationID(organizationID string) error {
	if !postgresOrganizationIDPattern.MatchString(organizationID) {
		return errors.New("organization id must contain 1-128 letters, digits, dots, underscores, or hyphens")
	}
	return nil
}

func (s *PostgresStore) signedTenantContext(organizationID string, now time.Time) (string, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := validatePostgresOrganizationID(organizationID); err != nil {
		return "", err
	}
	if len(s.tenantContextKey) < 32 {
		return "", ErrPostgresTenantIsolationUnavailable
	}
	expires := now.UTC().Add(postgresTenantContextTTL).Unix()
	message := organizationID + "\n" + fmt.Sprint(expires)
	mac := hmac.New(sha256.New, s.tenantContextKey)
	_, _ = mac.Write([]byte(message))
	return organizationID + "|" + fmt.Sprint(expires) + "|" + hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *PostgresStore) tenantRuntimeReady() bool {
	return s != nil && s.db != nil && len(s.tenantContextKey) >= 32
}

func postgresTenantRuntimeReady(store Store) bool {
	postgres, ok := store.(*PostgresStore)
	return ok && postgres.tenantRuntimeReady() && postgres.organizationID == ""
}

func (s *PostgresStore) verifyRuntimeRoleAndContext(ctx context.Context) error {
	var privileged, isRuntimeRole, ownerOrMember, canCreateSchema, canCreateDatabase, canUseTemp, canReadKey, hasDML, canExecuteVerifier, isMigratorMember, legacyRoleActive bool
	err := s.db.QueryRowContext(ctx, `
SELECT NOT r.rolcanlogin OR r.rolinherit OR r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication
         OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles elevated WHERE (elevated.rolsuper OR elevated.rolbypassrls OR elevated.rolcreaterole OR elevated.rolcreatedb OR elevated.rolreplication) AND pg_catalog.pg_has_role(current_user, elevated.oid, 'MEMBER')),
       current_user = 'ollama_agent_runtime',
       pg_catalog.pg_has_role(current_user, m.relowner, 'MEMBER') OR m.relowner = r.oid,
       pg_catalog.has_schema_privilege(current_user, 'public', 'CREATE'),
       pg_catalog.has_database_privilege(current_user, pg_catalog.current_database(), 'CREATE'),
       pg_catalog.has_database_privilege(current_user, pg_catalog.current_database(), 'TEMP'),
       pg_catalog.has_table_privilege(current_user, 'public.agent_tenant_context_key', 'SELECT'),
	       pg_catalog.has_table_privilege(current_user, 'public.agent_missions', 'SELECT,INSERT,UPDATE,DELETE')
	         AND pg_catalog.has_table_privilege(current_user, 'public.agent_events', 'SELECT,INSERT,UPDATE,DELETE'),
	       pg_catalog.has_function_privilege(current_user, 'public.agent_tenant_context_matches(text)', 'EXECUTE'),
	       pg_catalog.pg_has_role(current_user, 'ollama_agent_migrator', 'MEMBER'),
	       EXISTS (SELECT 1 FROM pg_catalog.pg_roles legacy WHERE legacy.rolname='ollama_agent' AND (legacy.rolcanlogin OR legacy.rolsuper OR legacy.rolbypassrls OR legacy.rolcreatedb OR legacy.rolcreaterole OR legacy.rolreplication OR legacy.rolinherit OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members membership WHERE membership.member=legacy.oid OR membership.roleid=legacy.oid)))
	         OR EXISTS (SELECT 1 FROM pg_catalog.pg_stat_activity legacy_session WHERE legacy_session.usename='ollama_agent' AND legacy_session.pid<>pg_catalog.pg_backend_pid())
  FROM pg_catalog.pg_roles AS r
	  JOIN pg_catalog.pg_class AS m ON m.relname = 'agent_missions' AND m.relnamespace = 'public'::pg_catalog.regnamespace
 WHERE r.rolname = current_user`).Scan(&privileged, &isRuntimeRole, &ownerOrMember, &canCreateSchema, &canCreateDatabase, &canUseTemp, &canReadKey, &hasDML, &canExecuteVerifier, &isMigratorMember, &legacyRoleActive)
	if err != nil {
		return fmt.Errorf("verify PostgreSQL runtime role; run the explicit migration with the migrator DSN first: %w", err)
	}
	if legacyRoleActive {
		return ErrPostgresLegacyRoleActive
	}
	if privileged || !isRuntimeRole || ownerOrMember || canCreateSchema || canCreateDatabase || canUseTemp || canReadKey || !hasDML || !canExecuteVerifier || isMigratorMember {
		return fmt.Errorf("%w: runtime role must use ollama_agent_runtime with NOINHERIT, no elevated/schema/database/TEMP privileges, no secret read, no owner/migrator membership; only tenant DML and verifier EXECUTE are permitted", ErrPostgresTenantSecurityShape)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SET LOCAL search_path = pg_catalog, public`); err != nil {
		return fmt.Errorf("pin PostgreSQL runtime readiness search path: %w", err)
	}
	if err := verifyPostgresTenantSecurityShape(ctx, tx); err != nil {
		return fmt.Errorf("%w: %v", ErrPostgresTenantSecurityShape, err)
	}
	const probeOrganization = "ollama_full_context_probe"
	token, err := s.signedTenantContext(probeOrganization, time.Now())
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.tenant_context', $1, true)`, token); err != nil {
		return fmt.Errorf("set PostgreSQL tenant context: %w", err)
	}
	var valid bool
	if err := tx.QueryRowContext(ctx, `SELECT public.agent_tenant_context_matches($1)`, probeOrganization).Scan(&valid); err != nil {
		return fmt.Errorf("verify PostgreSQL tenant signing key and RLS function: %w", err)
	}
	if !valid {
		return errors.New("PostgreSQL tenant context key does not match the migrated database secret")
	}
	return tx.Rollback()
}

// MigratePostgresAgentSchema applies schema changes with a separately supplied
// migration DSN. It never runs as part of normal server startup.
func MigratePostgresAgentSchema(ctx context.Context, dsn string, tenantContextKey []byte) error {
	if strings.TrimSpace(dsn) == "" {
		return errors.New("postgres migrator DSN is required")
	}
	if len(tenantContextKey) < 32 {
		return errors.New("postgres tenant context key must be at least 32 bytes")
	}
	if ctx == nil {
		return errors.New("postgres migration context is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect PostgreSQL migrator role: %w", err)
	}
	var unsafeRole bool
	if err := db.QueryRowContext(ctx, `
SELECT NOT r.rolcanlogin OR r.rolinherit OR r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication
	         OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles elevated WHERE (elevated.rolsuper OR elevated.rolbypassrls OR elevated.rolcreaterole OR elevated.rolcreatedb OR elevated.rolreplication) AND pg_catalog.pg_has_role(current_user, elevated.oid, 'MEMBER'))
	         OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid OR m.roleid=r.oid)
	  FROM pg_catalog.pg_roles r WHERE r.rolname = current_user`).Scan(&unsafeRole); err != nil {
		return fmt.Errorf("verify PostgreSQL migrator role: %w", err)
	}
	if unsafeRole {
		return errors.New("PostgreSQL migrator must be a dedicated non-superuser, NOBYPASSRLS, NO CREATEROLE/CREATEDB/REPLICATION role with no membership in such roles")
	}
	var isMigrator bool
	if err := db.QueryRowContext(ctx, `SELECT current_user = 'ollama_agent_migrator'`).Scan(&isMigrator); err != nil {
		return fmt.Errorf("verify PostgreSQL migrator identity: %w", err)
	}
	if !isMigrator {
		return errors.New("PostgreSQL migrations require the dedicated ollama_agent_migrator role")
	}
	return migratePostgresAgentSchema(ctx, db, tenantContextKey)
}

func migratePostgresAgentSchema(ctx context.Context, db *sql.DB, tenantContextKey []byte) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_catalog.pg_advisory_xact_lock($1)`, postgresTenantSecurityMigrationLock); err != nil {
		return err
	}
	// Only the dedicated migrator may create objects in public. Runtime paths
	// pin pg_catalog first and fully qualify application tables.
	if _, err := tx.ExecContext(ctx, `SET LOCAL search_path = pg_catalog, public`); err != nil {
		return fmt.Errorf("pin PostgreSQL migration search path: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DO $privileges$ BEGIN
  EXECUTE format('REVOKE TEMP ON DATABASE %I FROM PUBLIC', pg_catalog.current_database());
  EXECUTE format('REVOKE TEMP ON DATABASE %I FROM ollama_agent_runtime', pg_catalog.current_database());
  EXECUTE format('REVOKE CREATE ON DATABASE %I FROM PUBLIC', pg_catalog.current_database());
  EXECUTE format('REVOKE CREATE ON DATABASE %I FROM ollama_agent_runtime', pg_catalog.current_database());
END $privileges$`); err != nil {
		return fmt.Errorf("revoke PostgreSQL runtime temporary/database-create privileges: %w", err)
	}
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public`,
		`CREATE TABLE IF NOT EXISTS public.agent_missions (id TEXT PRIMARY KEY, version BIGINT NOT NULL, objective TEXT NOT NULL, provider TEXT NOT NULL DEFAULT '', model TEXT NOT NULL DEFAULT '', workspace TEXT NOT NULL DEFAULT '', workspace_identity TEXT NOT NULL DEFAULT '', project_id TEXT NOT NULL DEFAULT '', organization_id TEXT NOT NULL DEFAULT '', capabilities JSONB NOT NULL DEFAULT '[]'::jsonb, auto_run BOOLEAN NOT NULL DEFAULT FALSE, state TEXT NOT NULL, plan JSONB NOT NULL, approvals JSONB NOT NULL, artifacts JSONB NOT NULL, last_error TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, completed_at TIMESTAMPTZ NULL)`,
		`CREATE TABLE IF NOT EXISTS public.agent_events (id TEXT PRIMARY KEY, mission_id TEXT NOT NULL, organization_id TEXT NOT NULL DEFAULT '', type TEXT NOT NULL, step_id TEXT NOT NULL DEFAULT '', payload JSONB NULL, created_at TIMESTAMPTZ NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS public.agent_tenant_context_key (key_id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (key_id), secret BYTEA NOT NULL CHECK (octet_length(secret) >= 32))`,
		`DO $ownership$ DECLARE invalid_owners BIGINT; BEGIN
SELECT count(*) INTO invalid_owners FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
WHERE n.nspname='public' AND c.relname IN ('agent_missions','agent_events','agent_tenant_context_key')
  AND c.relkind IN ('r','p') AND c.relowner <> (SELECT oid FROM pg_catalog.pg_roles WHERE rolname='ollama_agent_migrator');
IF invalid_owners <> 0 THEN RAISE EXCEPTION 'PostgreSQL agent tables must be owned by ollama_agent_migrator; run deploy/postgres/migrate-existing-roles.sql as administrator before migration'; END IF;
END $ownership$`,
		`ALTER TABLE public.agent_missions ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE public.agent_missions ADD COLUMN IF NOT EXISTS workspace_identity TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE public.agent_missions ADD COLUMN IF NOT EXISTS capabilities JSONB NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE public.agent_missions ADD COLUMN IF NOT EXISTS organization_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE public.agent_missions ADD COLUMN IF NOT EXISTS workspace_isolated BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE public.agent_missions ADD COLUMN IF NOT EXISTS workspace_snapshot_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE public.agent_missions ADD COLUMN IF NOT EXISTS workspace_snapshot_sha256 TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE public.agent_events ADD COLUMN IF NOT EXISTS organization_id TEXT NOT NULL DEFAULT ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS agent_missions_id_organization_uidx ON public.agent_missions (id, organization_id)`,
		`DO $constraint$ BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint WHERE conname='agent_events_mission_org_fk' AND conrelid='public.agent_events'::regclass) THEN
  ALTER TABLE public.agent_events ADD CONSTRAINT agent_events_mission_org_fk
    FOREIGN KEY (mission_id, organization_id) REFERENCES public.agent_missions (id, organization_id)
    ON UPDATE RESTRICT ON DELETE CASCADE NOT VALID;
END IF;
		END $constraint$`,
		`ALTER TABLE public.agent_events VALIDATE CONSTRAINT agent_events_mission_org_fk`,
		`CREATE INDEX IF NOT EXISTS agent_events_mission_created_idx ON public.agent_events (mission_id, created_at, id)`,
		`CREATE INDEX IF NOT EXISTS agent_missions_organization_updated_idx ON public.agent_missions (organization_id, updated_at, id)`,
		`LOCK TABLE public.agent_missions, public.agent_events IN ACCESS EXCLUSIVE MODE`,
		`ALTER TABLE public.agent_missions DISABLE ROW LEVEL SECURITY`,
		`ALTER TABLE public.agent_events DISABLE ROW LEVEL SECURITY`,
	}
	for index, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply PostgreSQL agent migration statement %d: %w", index+1, err)
		}
		if index == 12 {
			if err := validatePostgresLegacySchemaShape(ctx, tx, false); err != nil {
				return err
			}
		}
	}
	if err := validatePostgresLegacySchemaShape(ctx, tx, true); err != nil {
		return err
	}
	var invalidMissions, invalidEvents, mismatchedEvents, invalidWorkspaceBindings int64
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM public.agent_missions WHERE organization_id IS NULL OR organization_id !~ '^[A-Za-z0-9_.-]{1,128}$'`).Scan(&invalidMissions); err != nil {
		return fmt.Errorf("validate existing mission tenant ownership: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM public.agent_events WHERE organization_id IS NULL OR organization_id !~ '^[A-Za-z0-9_.-]{1,128}$'`).Scan(&invalidEvents); err != nil {
		return fmt.Errorf("validate existing event tenant ownership: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM public.agent_events e LEFT JOIN public.agent_missions m ON m.id=e.mission_id WHERE m.id IS NULL OR e.organization_id IS NULL OR m.organization_id IS NULL OR e.organization_id <> m.organization_id`).Scan(&mismatchedEvents); err != nil {
		return fmt.Errorf("validate event-to-mission ownership: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM public.agent_missions WHERE state IN ('READY','RUNNING','RECOVERING') AND (workspace_identity IS NULL OR btrim(workspace_identity) = '')`).Scan(&invalidWorkspaceBindings); err != nil {
		return fmt.Errorf("validate runnable mission workspace authorization: %w", err)
	}
	if invalidMissions != 0 || invalidEvents != 0 || mismatchedEvents != 0 {
		return fmt.Errorf("PostgreSQL migration refused unsafe tenant backfill (missions without valid owner=%d, events without valid owner=%d, orphaned or mismatched events=%d); assign and verify owners before retrying", invalidMissions, invalidEvents, mismatchedEvents)
	}
	if invalidWorkspaceBindings != 0 {
		return fmt.Errorf("PostgreSQL migration refused %d runnable legacy missions without a persisted workspace identity; reauthorize/recreate those missions before retrying", invalidWorkspaceBindings)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO public.agent_tenant_context_key (key_id, secret) VALUES (TRUE, $1) ON CONFLICT (key_id) DO NOTHING`, tenantContextKey); err != nil {
		return fmt.Errorf("provision protected tenant context key: %w", err)
	}
	var storedTenantKey []byte
	if err := tx.QueryRowContext(ctx, `SELECT secret FROM public.agent_tenant_context_key WHERE key_id = TRUE`).Scan(&storedTenantKey); err != nil {
		return fmt.Errorf("verify protected tenant context key: %w", err)
	}
	if len(storedTenantKey) != len(tenantContextKey) || subtle.ConstantTimeCompare(storedTenantKey, tenantContextKey) != 1 {
		return errors.New("tenant context key differs from the provisioned database secret; ordinary migrations do not rotate keys")
	}
	functionSQL := `CREATE OR REPLACE FUNCTION public.agent_tenant_context_matches(requested_org TEXT) RETURNS BOOLEAN
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $function$
DECLARE
  context_value TEXT;
  context_parts TEXT[];
  context_org TEXT;
  context_expiry TEXT;
  context_signature TEXT;
  context_secret BYTEA;
  expiry_epoch BIGINT;
  expected_signature TEXT;
BEGIN
  context_value := current_setting('app.tenant_context', TRUE);
  IF context_value IS NULL OR length(context_value) > 512 THEN RETURN FALSE; END IF;
  context_parts := string_to_array(context_value, '|');
	IF array_length(context_parts, 1) IS DISTINCT FROM 3 THEN RETURN FALSE; END IF;
  context_org := context_parts[1];
  context_expiry := context_parts[2];
  context_signature := context_parts[3];
	IF context_org IS NULL OR context_expiry IS NULL OR context_signature IS NULL THEN RETURN FALSE; END IF;
	IF requested_org IS NULL OR requested_org = '' OR context_org <> requested_org OR context_org !~ '^[A-Za-z0-9_.-]{1,128}$' THEN RETURN FALSE; END IF;
  IF context_expiry !~ '^[0-9]{1,12}$' OR context_signature !~ '^[0-9a-f]{64}$' THEN RETURN FALSE; END IF;
  expiry_epoch := context_expiry::BIGINT;
  IF expiry_epoch < floor(extract(epoch FROM statement_timestamp()))::BIGINT OR expiry_epoch > floor(extract(epoch FROM statement_timestamp()))::BIGINT + 300 THEN RETURN FALSE; END IF;
  SELECT secret INTO context_secret FROM public.agent_tenant_context_key WHERE key_id = TRUE;
  IF context_secret IS NULL THEN RETURN FALSE; END IF;
  expected_signature := encode(public.hmac(convert_to(context_org || E'\n' || context_expiry, 'UTF8'), context_secret, 'sha256'), 'hex');
  RETURN expected_signature = context_signature;
EXCEPTION WHEN OTHERS THEN
  RETURN FALSE;
END;
$function$`
	finalStatements := []string{
		functionSQL,
		`DROP POLICY IF EXISTS agent_missions_tenant_policy ON public.agent_missions`,
		`DROP POLICY IF EXISTS agent_events_tenant_policy ON public.agent_events`,
		`ALTER TABLE public.agent_missions ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE public.agent_missions FORCE ROW LEVEL SECURITY`,
		`ALTER TABLE public.agent_events ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE public.agent_events FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY agent_missions_tenant_policy ON public.agent_missions USING (organization_id <> '' AND public.agent_tenant_context_matches(organization_id)) WITH CHECK (organization_id <> '' AND public.agent_tenant_context_matches(organization_id))`,
		`CREATE POLICY agent_events_tenant_policy ON public.agent_events USING (organization_id <> '' AND public.agent_tenant_context_matches(organization_id) AND EXISTS (SELECT 1 FROM public.agent_missions m WHERE m.id = agent_events.mission_id AND m.organization_id = agent_events.organization_id)) WITH CHECK (organization_id <> '' AND public.agent_tenant_context_matches(organization_id) AND EXISTS (SELECT 1 FROM public.agent_missions m WHERE m.id = agent_events.mission_id AND m.organization_id = agent_events.organization_id))`,
		`REVOKE ALL ON TABLE public.agent_missions, public.agent_events, public.agent_tenant_context_key FROM PUBLIC`,
		`REVOKE ALL ON TABLE public.agent_missions, public.agent_events, public.agent_tenant_context_key FROM ollama_agent_runtime`,
		`REVOKE TRUNCATE, REFERENCES, TRIGGER ON TABLE public.agent_missions, public.agent_events FROM ollama_agent_runtime`,
		`GRANT USAGE ON SCHEMA public TO ollama_agent_runtime`,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE public.agent_missions, public.agent_events TO ollama_agent_runtime`,
		`REVOKE ALL ON FUNCTION public.agent_tenant_context_matches(TEXT) FROM PUBLIC`,
		`GRANT EXECUTE ON FUNCTION public.agent_tenant_context_matches(TEXT) TO ollama_agent_runtime`,
		`REVOKE CREATE ON SCHEMA public FROM PUBLIC`,
		`GRANT CREATE ON SCHEMA public TO ollama_agent_migrator`,
	}
	for _, statement := range finalStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("finalize PostgreSQL tenant RLS migration: %w", err)
		}
	}
	return tx.Commit()
}

func validatePostgresLegacySchemaShape(ctx context.Context, tx *sql.Tx, requireOwnershipConstraints bool) error {
	type columnShape struct {
		typ        string
		notNull    bool
		defaultSQL string
	}
	required := map[string]map[string]columnShape{
		"agent_missions": {
			"id": {"text", true, ""}, "version": {"bigint", true, ""}, "objective": {"text", true, ""},
			"provider": {"text", true, "''::text"}, "model": {"text", true, "''::text"}, "workspace": {"text", true, "''::text"},
			"workspace_identity": {"text", true, "''::text"}, "project_id": {"text", true, "''::text"}, "organization_id": {"text", true, "''::text"},
			"capabilities": {"jsonb", true, "'[]'::jsonb"}, "auto_run": {"boolean", true, "false"}, "workspace_isolated": {"boolean", true, "false"},
			"workspace_snapshot_id": {"text", true, "''::text"}, "workspace_snapshot_sha256": {"text", true, "''::text"}, "state": {"text", true, ""},
			"plan": {"jsonb", true, ""}, "approvals": {"jsonb", true, ""}, "artifacts": {"jsonb", true, ""},
			"last_error": {"text", true, "''::text"}, "created_at": {"timestamp with time zone", true, ""},
			"updated_at": {"timestamp with time zone", true, ""}, "completed_at": {"timestamp with time zone", false, ""},
		},
		"agent_events": {
			"id": {"text", true, ""}, "mission_id": {"text", true, ""}, "organization_id": {"text", true, "''::text"},
			"type": {"text", true, ""}, "step_id": {"text", true, "''::text"}, "payload": {"jsonb", false, ""},
			"created_at": {"timestamp with time zone", true, ""},
		},
	}
	for table, requiredColumns := range required {
		rows, err := tx.QueryContext(ctx, `SELECT a.attname, pg_catalog.format_type(a.atttypid,a.atttypmod), a.attnotnull, COALESCE(pg_catalog.pg_get_expr(d.adbin,d.adrelid),'') FROM pg_catalog.pg_attribute a LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum WHERE a.attrelid=pg_catalog.to_regclass($1) AND a.attnum>0 AND NOT a.attisdropped`, "public."+table)
		if err != nil {
			return fmt.Errorf("inspect PostgreSQL agent schema shape: %w", err)
		}
		present := make(map[string]columnShape, len(requiredColumns))
		for rows.Next() {
			var column string
			var shape columnShape
			if err := rows.Scan(&column, &shape.typ, &shape.notNull, &shape.defaultSQL); err != nil {
				_ = rows.Close()
				return fmt.Errorf("inspect PostgreSQL agent schema shape: %w", err)
			}
			present[column] = shape
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("inspect PostgreSQL agent schema shape: %w", err)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("inspect PostgreSQL agent schema shape: %w", err)
		}
		var missing, incompatible []string
		for column, expected := range requiredColumns {
			actual, ok := present[column]
			if !ok {
				missing = append(missing, column)
				continue
			}
			if actual.typ != expected.typ || actual.notNull != expected.notNull || actual.defaultSQL != expected.defaultSQL {
				incompatible = append(incompatible, fmt.Sprintf("%s (got type=%s not-null=%t default=%q, want type=%s not-null=%t default=%q)", column, actual.typ, actual.notNull, actual.defaultSQL, expected.typ, expected.notNull, expected.defaultSQL))
			}
		}
		if len(missing) != 0 {
			return fmt.Errorf("PostgreSQL agent schema is older than the supported baseline: public.%s is missing required columns %s; create a verified backup and perform an explicit versioned schema upgrade before retrying", table, strings.Join(missing, ", "))
		}
		if len(incompatible) != 0 {
			return fmt.Errorf("PostgreSQL agent schema has incompatible type/nullability for public.%s: %s; create a verified backup and perform an explicit versioned schema upgrade before retrying", table, strings.Join(incompatible, "; "))
		}
	}
	if !requireOwnershipConstraints {
		return nil
	}
	for table := range required {
		var primaryKeyValid bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (
  SELECT 1 FROM pg_catalog.pg_constraint c
  WHERE c.conrelid=pg_catalog.to_regclass($1) AND c.contype='p'
    AND c.conkey=ARRAY[(SELECT attnum FROM pg_catalog.pg_attribute WHERE attrelid=pg_catalog.to_regclass($1) AND attname='id' AND NOT attisdropped)]::smallint[]
)`, "public."+table).Scan(&primaryKeyValid); err != nil {
			return fmt.Errorf("inspect PostgreSQL %s primary key: %w", table, err)
		}
		if !primaryKeyValid {
			return fmt.Errorf("PostgreSQL supported baseline requires public.%s primary key(id)", table)
		}
	}
	var tenantUniqueIndex, eventOwnershipForeignKey bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (
  SELECT 1 FROM pg_catalog.pg_index i
  WHERE i.indrelid='public.agent_missions'::pg_catalog.regclass
	    AND i.indisunique AND i.indisvalid AND i.indisready AND i.indnkeyatts=2 AND i.indnatts=2
	    AND i.indkey[0]=(SELECT attnum FROM pg_catalog.pg_attribute WHERE attrelid='public.agent_missions'::pg_catalog.regclass AND attname='id' AND NOT attisdropped)
	    AND i.indkey[1]=(SELECT attnum FROM pg_catalog.pg_attribute WHERE attrelid='public.agent_missions'::pg_catalog.regclass AND attname='organization_id' AND NOT attisdropped)
)`).Scan(&tenantUniqueIndex); err != nil {
		return fmt.Errorf("inspect PostgreSQL tenant ownership unique index: %w", err)
	}
	if !tenantUniqueIndex {
		return errors.New("PostgreSQL supported baseline requires a valid unique index on public.agent_missions(id, organization_id)")
	}
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (
  SELECT 1 FROM pg_catalog.pg_constraint c
  WHERE c.conrelid='public.agent_events'::pg_catalog.regclass
    AND c.confrelid='public.agent_missions'::pg_catalog.regclass
    AND c.contype='f' AND c.convalidated
    AND c.conkey=ARRAY[
      (SELECT attnum FROM pg_catalog.pg_attribute WHERE attrelid='public.agent_events'::pg_catalog.regclass AND attname='mission_id' AND NOT attisdropped),
      (SELECT attnum FROM pg_catalog.pg_attribute WHERE attrelid='public.agent_events'::pg_catalog.regclass AND attname='organization_id' AND NOT attisdropped)
    ]::smallint[]
    AND c.confkey=ARRAY[
      (SELECT attnum FROM pg_catalog.pg_attribute WHERE attrelid='public.agent_missions'::pg_catalog.regclass AND attname='id' AND NOT attisdropped),
      (SELECT attnum FROM pg_catalog.pg_attribute WHERE attrelid='public.agent_missions'::pg_catalog.regclass AND attname='organization_id' AND NOT attisdropped)
    ]::smallint[]
    AND c.confdeltype='c' AND c.confupdtype='r'
)`).Scan(&eventOwnershipForeignKey); err != nil {
		return fmt.Errorf("inspect PostgreSQL event tenant ownership foreign key: %w", err)
	}
	if !eventOwnershipForeignKey {
		return errors.New("PostgreSQL supported baseline requires a validated event(mission_id, organization_id) foreign key with ON DELETE CASCADE and ON UPDATE RESTRICT")
	}
	return nil
}
