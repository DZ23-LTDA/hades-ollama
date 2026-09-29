package agent

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// RotatePostgresAgentTenantKey disables new runtime logins, requires all
// existing runtime sessions to be gone, then atomically activates a newer key.
// Runtime processes must be restarted with the new version after it succeeds.
func RotatePostgresAgentTenantKey(ctx context.Context, adminDSN, migratorDSN string, currentVersion int, currentKey []byte, nextVersion int, nextKey []byte) error {
	if strings.TrimSpace(adminDSN) == "" || strings.TrimSpace(migratorDSN) == "" {
		return errors.New("postgres admin and migrator DSNs are required for key rotation")
	}
	if ctx == nil {
		return errors.New("postgres key rotation context is required")
	}
	if currentVersion < 1 || currentVersion > 999999999 || nextVersion <= currentVersion || nextVersion > 999999999 {
		return errors.New("tenant key rotation requires a valid positive version and a strictly greater next version")
	}
	if len(currentKey) < 32 || len(nextKey) < 32 {
		return errors.New("current and next PostgreSQL tenant context keys must each be at least 32 bytes")
	}
	if subtle.ConstantTimeCompare(currentKey, nextKey) == 1 {
		return errors.New("next PostgreSQL tenant context key must differ from the current key")
	}

	adminDB, err := sql.Open("pgx", adminDSN)
	if err != nil {
		return err
	}
	defer adminDB.Close()
	if err := adminDB.PingContext(ctx); err != nil {
		return fmt.Errorf("connect PostgreSQL admin role for key rotation: %w", err)
	}
	adminConn, err := adminDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve PostgreSQL admin session for key rotation: %w", err)
	}
	defer adminConn.Close()
	var adminReady bool
	if err := adminConn.QueryRowContext(ctx, `SELECT current_user='ollama_agent_admin' AND r.rolsuper FROM pg_catalog.pg_roles r WHERE r.rolname=current_user`).Scan(&adminReady); err != nil {
		return fmt.Errorf("verify PostgreSQL admin identity for key rotation: %w", err)
	}
	if !adminReady {
		return errors.New("PostgreSQL key rotation requires the dedicated ollama_agent_admin superuser for runtime login fencing")
	}
	var runtimeMemberships bool
	if err := adminConn.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles r ON r.oid=m.roleid OR r.oid=m.member WHERE r.rolname='ollama_agent_runtime')`).Scan(&runtimeMemberships); err != nil {
		return fmt.Errorf("verify runtime role-membership fence: %w", err)
	}
	if runtimeMemberships {
		return errors.New("PostgreSQL runtime role has memberships; remove all forward/reverse memberships before key rotation")
	}
	var adminSystemID, adminDatabase string
	if err := adminConn.QueryRowContext(ctx, `SELECT system_identifier::text FROM pg_catalog.pg_control_system()`).Scan(&adminSystemID); err != nil {
		return fmt.Errorf("read admin PostgreSQL system identity: %w", err)
	}
	if err := adminConn.QueryRowContext(ctx, `SELECT current_database()`).Scan(&adminDatabase); err != nil {
		return fmt.Errorf("read admin PostgreSQL database identity: %w", err)
	}
	db, err := sql.Open("pgx", migratorDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect PostgreSQL migrator role for key rotation: %w", err)
	}
	var unsafeRole, isMigrator bool
	if err := db.QueryRowContext(ctx, `SELECT NOT r.rolcanlogin OR r.rolinherit OR r.rolsuper OR r.rolbypassrls OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid OR m.roleid=r.oid) FROM pg_catalog.pg_roles r WHERE r.rolname=current_user`).Scan(&unsafeRole); err != nil {
		return fmt.Errorf("verify PostgreSQL migrator role for key rotation: %w", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT current_user='ollama_agent_migrator'`).Scan(&isMigrator); err != nil {
		return fmt.Errorf("verify PostgreSQL migrator identity for key rotation: %w", err)
	}
	if unsafeRole || !isMigrator {
		return errors.New("PostgreSQL key rotation requires the dedicated non-superuser ollama_agent_migrator role")
	}
	var migratorSystemID, migratorDatabase string
	if err := db.QueryRowContext(ctx, `SELECT system_identifier::text FROM pg_catalog.pg_control_system()`).Scan(&migratorSystemID); err != nil {
		return fmt.Errorf("read migrator PostgreSQL system identity: %w", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&migratorDatabase); err != nil {
		return fmt.Errorf("read migrator PostgreSQL database identity: %w", err)
	}
	if adminSystemID != migratorSystemID || adminDatabase != migratorDatabase {
		return errors.New("PostgreSQL admin and migrator DSNs must target the same database on the same cluster")
	}
	var rotationLockAcquired bool
	if err := adminConn.QueryRowContext(ctx, `SELECT pg_catalog.pg_try_advisory_lock($1)`, postgresTenantSecurityMigrationLock+1).Scan(&rotationLockAcquired); err != nil {
		return fmt.Errorf("acquire PostgreSQL key-rotation coordinator lock: %w", err)
	}
	if !rotationLockAcquired {
		return errors.New("another PostgreSQL tenant-key rotation is already running")
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var unlocked bool
		_ = adminConn.QueryRowContext(unlockCtx, `SELECT pg_catalog.pg_advisory_unlock($1)`, postgresTenantSecurityMigrationLock+1).Scan(&unlocked)
	}()
	if _, err := adminConn.ExecContext(ctx, `ALTER ROLE ollama_agent_runtime NOLOGIN`); err != nil {
		return fmt.Errorf("fence new PostgreSQL runtime logins: %w", err)
	}
	runtimeLoginDisabled := true
	defer func() {
		if runtimeLoginDisabled {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = adminConn.ExecContext(cleanupCtx, `ALTER ROLE ollama_agent_runtime LOGIN`)
		}
	}()
	var runtimeSessions int
	if err := adminConn.QueryRowContext(ctx, `SELECT count(*) FROM pg_catalog.pg_stat_activity WHERE usename='ollama_agent_runtime' AND pid<>pg_catalog.pg_backend_pid()`).Scan(&runtimeSessions); err != nil {
		return fmt.Errorf("verify runtime quiescence for PostgreSQL key rotation: %w", err)
	}
	if runtimeSessions != 0 {
		return fmt.Errorf("PostgreSQL runtime is not quiescent: %d runtime session(s) remain; stop every runtime/worker and retry", runtimeSessions)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SET LOCAL lock_timeout='2min'`); err != nil {
		return fmt.Errorf("set bounded PostgreSQL key rotation lock wait: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT pg_catalog.pg_advisory_xact_lock($1)`, postgresTenantSecurityMigrationLock); err != nil {
		return fmt.Errorf("drain PostgreSQL tenant transactions for key rotation: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `SET LOCAL search_path=pg_catalog,public`); err != nil {
		return fmt.Errorf("pin PostgreSQL key rotation search path: %w", err)
	}
	var legacyKeyTable bool
	if err := tx.QueryRowContext(ctx, `SELECT pg_catalog.to_regclass('public.agent_tenant_context_key') IS NOT NULL`).Scan(&legacyKeyTable); err != nil {
		return fmt.Errorf("check legacy PostgreSQL key migration state: %w", err)
	}
	if legacyKeyTable {
		return errors.New("PostgreSQL tenant keyring has not been migrated from its legacy singleton table")
	}
	var keyringShapeSecure bool
	if err := tx.QueryRowContext(ctx, `SELECT
	EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
	        WHERE n.nspname='public' AND c.relname='agent_tenant_context_keys' AND c.relkind='r'
	          AND c.relowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname=current_user))
	AND EXISTS (SELECT 1 FROM pg_catalog.pg_index i JOIN pg_catalog.pg_class idx ON idx.oid=i.indexrelid
	        WHERE i.indrelid='public.agent_tenant_context_keys'::pg_catalog.regclass
	          AND idx.relname='agent_tenant_context_keys_one_active_uidx'
	          AND i.indisunique AND i.indisvalid AND i.indisready AND i.indpred IS NOT NULL
	          AND i.indnatts=1 AND i.indnkeyatts=1
	          AND pg_catalog.pg_get_expr(i.indpred,i.indrelid)='is_active')`).Scan(&keyringShapeSecure); err != nil {
		return fmt.Errorf("verify PostgreSQL keyring ownership and uniqueness before rotation: %w", err)
	}
	if !keyringShapeSecure {
		return errors.New("PostgreSQL tenant keyring ownership or unique-active-index contract has drifted; repair with explicit migration first")
	}
	if _, err := tx.ExecContext(ctx, `LOCK TABLE public.agent_tenant_context_keys IN EXCLUSIVE MODE`); err != nil {
		return fmt.Errorf("lock PostgreSQL tenant keyring for rotation: %w", err)
	}
	var activeCount int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM public.agent_tenant_context_keys WHERE is_active`).Scan(&activeCount); err != nil {
		return fmt.Errorf("read active PostgreSQL tenant key version: %w", err)
	}
	if activeCount != 1 {
		return fmt.Errorf("PostgreSQL tenant keyring has %d active versions; exactly one is required", activeCount)
	}
	var storedVersion int
	var storedKey []byte
	if err := tx.QueryRowContext(ctx, `SELECT key_version,secret FROM public.agent_tenant_context_keys WHERE is_active FOR UPDATE`).Scan(&storedVersion, &storedKey); err != nil {
		return fmt.Errorf("lock active PostgreSQL tenant key: %w", err)
	}
	if storedVersion != currentVersion || len(storedKey) != len(currentKey) || subtle.ConstantTimeCompare(storedKey, currentKey) != 1 {
		return errors.New("supplied current PostgreSQL tenant key/version does not match the active database key")
	}
	var highestRecordedVersion int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(max(key_version),0) FROM public.agent_tenant_context_keys`).Scan(&highestRecordedVersion); err != nil {
		return fmt.Errorf("read PostgreSQL tenant key version high-water mark: %w", err)
	}
	if nextVersion <= highestRecordedVersion {
		return fmt.Errorf("next PostgreSQL tenant key version must exceed the highest recorded version %d; versions are never reused or decremented", highestRecordedVersion)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO public.agent_tenant_context_keys (key_version,secret,is_active) VALUES ($1,$2,FALSE)`, nextVersion, nextKey); err != nil {
		return fmt.Errorf("stage next PostgreSQL tenant key: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE public.agent_tenant_context_keys SET is_active=FALSE WHERE key_version=$1 AND is_active`, currentVersion)
	if err != nil {
		return fmt.Errorf("deactivate current PostgreSQL tenant key: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return fmt.Errorf("deactivate current PostgreSQL tenant key affected %d rows; expected exactly one", rows)
	}
	result, err = tx.ExecContext(ctx, `UPDATE public.agent_tenant_context_keys SET is_active=TRUE WHERE key_version=$1 AND NOT is_active`, nextVersion)
	if err != nil {
		return fmt.Errorf("activate next PostgreSQL tenant key: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return fmt.Errorf("activate next PostgreSQL tenant key affected %d rows; expected exactly one", rows)
	}
	var finalCount, finalVersion int
	if err := tx.QueryRowContext(ctx, `SELECT count(*), COALESCE(max(key_version),0) FROM public.agent_tenant_context_keys WHERE is_active`).Scan(&finalCount, &finalVersion); err != nil {
		return fmt.Errorf("verify rotated PostgreSQL tenant key: %w", err)
	}
	if finalCount != 1 || finalVersion != nextVersion {
		return fmt.Errorf("rotated PostgreSQL tenant key invariant failed: active=%d version=%d", finalCount, finalVersion)
	}
	var activatedKey []byte
	if err := tx.QueryRowContext(ctx, `SELECT secret FROM public.agent_tenant_context_keys WHERE is_active AND key_version=$1`, nextVersion).Scan(&activatedKey); err != nil {
		return fmt.Errorf("read activated PostgreSQL tenant key for verification: %w", err)
	}
	if len(activatedKey) != len(nextKey) || subtle.ConstantTimeCompare(activatedKey, nextKey) != 1 {
		return errors.New("activated PostgreSQL tenant key did not match the supplied next secret")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit PostgreSQL tenant key rotation: %w", err)
	}
	if _, err := adminConn.ExecContext(ctx, `ALTER ROLE ollama_agent_runtime LOGIN`); err != nil {
		return fmt.Errorf("key rotation committed, but runtime login re-enable failed; restore LOGIN as admin after verifying the active key: %w", err)
	}
	runtimeLoginDisabled = false
	return nil
}
