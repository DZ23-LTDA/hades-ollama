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
	postgresTenantContextGUC = "app.tenant_context"
	postgresTenantContextTTL = 5 * time.Minute
	postgresRuntimeRole      = "ollama_agent_runtime"
)

var postgresOrganizationIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)

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
	var privileged, isRuntimeRole, ownerOrMember, canCreateSchema, canReadKey, hasDML, isMigratorMember, legacyRoleActive bool
	err := s.db.QueryRowContext(ctx, `
SELECT r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication
         OR EXISTS (SELECT 1 FROM pg_roles elevated WHERE (elevated.rolsuper OR elevated.rolbypassrls OR elevated.rolcreaterole OR elevated.rolcreatedb OR elevated.rolreplication) AND pg_has_role(current_user, elevated.oid, 'MEMBER')),
       current_user = 'ollama_agent_runtime',
       pg_has_role(current_user, m.relowner, 'MEMBER') OR m.relowner = r.oid,
       has_schema_privilege(current_user, 'public', 'CREATE'),
       has_table_privilege(current_user, 'public.agent_tenant_context_key', 'SELECT'),
	       has_table_privilege(current_user, 'public.agent_missions', 'SELECT,INSERT,UPDATE,DELETE')
	         AND has_table_privilege(current_user, 'public.agent_events', 'SELECT,INSERT,UPDATE,DELETE'),
	       pg_has_role(current_user, 'ollama_agent_migrator', 'MEMBER'),
	       EXISTS (SELECT 1 FROM pg_roles legacy WHERE legacy.rolname='ollama_agent' AND (legacy.rolcanlogin OR legacy.rolsuper OR legacy.rolbypassrls))
	         OR EXISTS (SELECT 1 FROM pg_stat_activity legacy_session WHERE legacy_session.usename='ollama_agent' AND legacy_session.pid<>pg_backend_pid())
  FROM pg_roles AS r
  JOIN pg_class AS m ON m.relname = 'agent_missions' AND m.relnamespace = 'public'::regnamespace
 WHERE r.rolname = current_user`).Scan(&privileged, &isRuntimeRole, &ownerOrMember, &canCreateSchema, &canReadKey, &hasDML, &isMigratorMember, &legacyRoleActive)
	if err != nil {
		return fmt.Errorf("verify PostgreSQL runtime role; run the explicit migration with the migrator DSN first: %w", err)
	}
	if privileged || !isRuntimeRole || ownerOrMember || canCreateSchema || canReadKey || !hasDML || isMigratorMember || legacyRoleActive {
		return errors.New("PostgreSQL runtime must use ollama_agent_runtime: no superuser, BYPASSRLS, CREATEROLE, CREATEDB, owner/migrator membership, schema CREATE, tenant-secret SELECT, or active legacy ollama_agent login; DML on missions/events is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
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
SELECT r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication
         OR EXISTS (SELECT 1 FROM pg_roles elevated WHERE (elevated.rolsuper OR elevated.rolbypassrls OR elevated.rolcreaterole OR elevated.rolcreatedb OR elevated.rolreplication) AND pg_has_role(current_user, elevated.oid, 'MEMBER'))
  FROM pg_roles r WHERE r.rolname = current_user`).Scan(&unsafeRole); err != nil {
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
	const migrationLock int64 = 0x4f4c4c414d414655
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLock); err != nil {
		return err
	}
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public`,
		`CREATE TABLE IF NOT EXISTS agent_missions (id TEXT PRIMARY KEY, version BIGINT NOT NULL, objective TEXT NOT NULL, provider TEXT NOT NULL DEFAULT '', model TEXT NOT NULL DEFAULT '', workspace TEXT NOT NULL DEFAULT '', workspace_identity TEXT NOT NULL DEFAULT '', project_id TEXT NOT NULL DEFAULT '', organization_id TEXT NOT NULL DEFAULT '', capabilities JSONB NOT NULL DEFAULT '[]'::jsonb, auto_run BOOLEAN NOT NULL DEFAULT FALSE, state TEXT NOT NULL, plan JSONB NOT NULL, approvals JSONB NOT NULL, artifacts JSONB NOT NULL, last_error TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, completed_at TIMESTAMPTZ NULL)`,
		`CREATE TABLE IF NOT EXISTS agent_events (id TEXT PRIMARY KEY, mission_id TEXT NOT NULL, organization_id TEXT NOT NULL DEFAULT '', type TEXT NOT NULL, step_id TEXT NOT NULL DEFAULT '', payload JSONB NULL, created_at TIMESTAMPTZ NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS agent_tenant_context_key (key_id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (key_id), secret BYTEA NOT NULL CHECK (octet_length(secret) >= 32))`,
		`DO $ownership$ DECLARE invalid_owners BIGINT; BEGIN
SELECT count(*) INTO invalid_owners FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
WHERE n.nspname='public' AND c.relname IN ('agent_missions','agent_events','agent_tenant_context_key')
  AND c.relkind IN ('r','p') AND c.relowner <> (SELECT oid FROM pg_roles WHERE rolname='ollama_agent_migrator');
IF invalid_owners <> 0 THEN RAISE EXCEPTION 'PostgreSQL agent tables must be owned by ollama_agent_migrator; run deploy/postgres/migrate-existing-roles.sql as administrator before migration'; END IF;
END $ownership$`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS workspace_identity TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS capabilities JSONB NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS organization_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS workspace_isolated BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS workspace_snapshot_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS workspace_snapshot_sha256 TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_events ADD COLUMN IF NOT EXISTS organization_id TEXT NOT NULL DEFAULT ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS agent_missions_id_organization_uidx ON agent_missions (id, organization_id)`,
		`DO $constraint$ BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='agent_events_mission_org_fk' AND conrelid='public.agent_events'::regclass) THEN
  ALTER TABLE public.agent_events ADD CONSTRAINT agent_events_mission_org_fk
    FOREIGN KEY (mission_id, organization_id) REFERENCES public.agent_missions (id, organization_id)
    ON UPDATE RESTRICT ON DELETE CASCADE NOT VALID;
END IF;
		END $constraint$`,
		`ALTER TABLE agent_events VALIDATE CONSTRAINT agent_events_mission_org_fk`,
		`CREATE INDEX IF NOT EXISTS agent_events_mission_created_idx ON agent_events (mission_id, created_at, id)`,
		`CREATE INDEX IF NOT EXISTS agent_missions_organization_updated_idx ON agent_missions (organization_id, updated_at, id)`,
		`LOCK TABLE agent_missions, agent_events IN ACCESS EXCLUSIVE MODE`,
		`ALTER TABLE agent_missions DISABLE ROW LEVEL SECURITY`,
		`ALTER TABLE agent_events DISABLE ROW LEVEL SECURITY`,
	}
	for index, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply PostgreSQL agent migration: %w", err)
		}
		if index == 3 {
			if err := validatePostgresLegacySchemaShape(ctx, tx); err != nil {
				return err
			}
		}
	}
	var invalidMissions, invalidEvents, mismatchedEvents, invalidWorkspaceBindings int64
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM agent_missions WHERE organization_id !~ '^[A-Za-z0-9_.-]{1,128}$'`).Scan(&invalidMissions); err != nil {
		return fmt.Errorf("validate existing mission tenant ownership: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM agent_events WHERE organization_id !~ '^[A-Za-z0-9_.-]{1,128}$'`).Scan(&invalidEvents); err != nil {
		return fmt.Errorf("validate existing event tenant ownership: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM agent_events e LEFT JOIN agent_missions m ON m.id=e.mission_id WHERE m.id IS NULL OR e.organization_id <> m.organization_id`).Scan(&mismatchedEvents); err != nil {
		return fmt.Errorf("validate event-to-mission ownership: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM agent_missions WHERE state IN ('READY','RUNNING','RECOVERING') AND btrim(workspace_identity) = ''`).Scan(&invalidWorkspaceBindings); err != nil {
		return fmt.Errorf("validate runnable mission workspace authorization: %w", err)
	}
	if invalidMissions != 0 || invalidEvents != 0 || mismatchedEvents != 0 {
		return fmt.Errorf("PostgreSQL migration refused unsafe tenant backfill (missions without valid owner=%d, events without valid owner=%d, orphaned or mismatched events=%d); assign and verify owners before retrying", invalidMissions, invalidEvents, mismatchedEvents)
	}
	if invalidWorkspaceBindings != 0 {
		return fmt.Errorf("PostgreSQL migration refused %d runnable legacy missions without a persisted workspace identity; reauthorize/recreate those missions before retrying", invalidWorkspaceBindings)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO agent_tenant_context_key (key_id, secret) VALUES (TRUE, $1) ON CONFLICT (key_id) DO NOTHING`, tenantContextKey); err != nil {
		return fmt.Errorf("provision protected tenant context key: %w", err)
	}
	var storedTenantKey []byte
	if err := tx.QueryRowContext(ctx, `SELECT secret FROM agent_tenant_context_key WHERE key_id = TRUE`).Scan(&storedTenantKey); err != nil {
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
		`DROP POLICY IF EXISTS agent_missions_tenant_policy ON agent_missions`,
		`DROP POLICY IF EXISTS agent_events_tenant_policy ON agent_events`,
		`ALTER TABLE agent_missions ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE agent_missions FORCE ROW LEVEL SECURITY`,
		`ALTER TABLE agent_events ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE agent_events FORCE ROW LEVEL SECURITY`,
		`CREATE POLICY agent_missions_tenant_policy ON agent_missions USING (organization_id <> '' AND public.agent_tenant_context_matches(organization_id)) WITH CHECK (organization_id <> '' AND public.agent_tenant_context_matches(organization_id))`,
		`CREATE POLICY agent_events_tenant_policy ON agent_events USING (organization_id <> '' AND public.agent_tenant_context_matches(organization_id) AND EXISTS (SELECT 1 FROM public.agent_missions m WHERE m.id = agent_events.mission_id AND m.organization_id = agent_events.organization_id)) WITH CHECK (organization_id <> '' AND public.agent_tenant_context_matches(organization_id) AND EXISTS (SELECT 1 FROM public.agent_missions m WHERE m.id = agent_events.mission_id AND m.organization_id = agent_events.organization_id))`,
		`REVOKE ALL ON TABLE agent_missions, agent_events, agent_tenant_context_key FROM PUBLIC`,
		`REVOKE ALL ON TABLE agent_missions, agent_events, agent_tenant_context_key FROM ollama_agent_runtime`,
		`GRANT USAGE ON SCHEMA public TO ollama_agent_runtime`,
		`GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE agent_missions, agent_events TO ollama_agent_runtime`,
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

func validatePostgresLegacySchemaShape(ctx context.Context, tx *sql.Tx) error {
	required := map[string][]string{
		"agent_missions": {"id", "version", "objective", "model", "workspace", "project_id", "auto_run", "state", "plan", "approvals", "artifacts", "last_error", "created_at", "updated_at", "completed_at"},
		"agent_events":   {"id", "mission_id", "type", "step_id", "payload", "created_at"},
	}
	for table, requiredColumns := range required {
		rows, err := tx.QueryContext(ctx, `SELECT attname FROM pg_attribute WHERE attrelid=to_regclass($1) AND attnum>0 AND NOT attisdropped`, "public."+table)
		if err != nil {
			return fmt.Errorf("inspect PostgreSQL agent schema shape: %w", err)
		}
		present := make(map[string]struct{}, len(requiredColumns))
		for rows.Next() {
			var column string
			if err := rows.Scan(&column); err != nil {
				_ = rows.Close()
				return fmt.Errorf("inspect PostgreSQL agent schema shape: %w", err)
			}
			present[column] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("inspect PostgreSQL agent schema shape: %w", err)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("inspect PostgreSQL agent schema shape: %w", err)
		}
		var missing []string
		for _, column := range requiredColumns {
			if _, ok := present[column]; !ok {
				missing = append(missing, column)
			}
		}
		if len(missing) != 0 {
			return fmt.Errorf("PostgreSQL agent schema is older than the supported baseline: public.%s is missing required columns %s; create a verified backup and perform an explicit versioned schema upgrade before retrying", table, strings.Join(missing, ", "))
		}
	}
	return nil
}
