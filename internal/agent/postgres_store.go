package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStore struct {
	db             *sql.DB
	timeout        time.Duration
	organizationID string
	systemAccess   bool
}

var ErrPostgresTenantRequiresNonSuperuser = errors.New("tenant-scoped postgres store requires a non-superuser role")
var ErrPostgresTenantIsolationUnavailable = errors.New("PostgreSQL agent storage is disabled until tenant context is non-forgeable and background work is tenant-scoped")

func OpenPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("postgres DSN is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	store := &PostgresStore{db: db, timeout: 10 * time.Second, systemAccess: true}
	pingCtx, cancel := context.WithTimeout(ctx, store.timeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}
	// PostgreSQL bypasses Row Level Security for superusers and BYPASSRLS roles,
	// even with FORCE ROW LEVEL SECURITY. This store is multi-tenant; there is no
	// process-wide override that can safely turn that isolation off.
	var privileged bool
	if err := db.QueryRowContext(pingCtx, `SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&privileged); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("verify database role privileges: %w", err)
	}
	if privileged {
		_ = db.Close()
		return nil, errors.New("agent database role must not be a superuser or have BYPASSRLS; use a NOSUPERUSER NOBYPASSRLS role")
	}
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

// WithOrganization returns a view that sets the PostgreSQL tenant context for every transaction.
// It shares the connection pool but never mutates the parent store.
func (s *PostgresStore) WithOrganization(organizationID string) *PostgresStore {
	if s == nil {
		return nil
	}
	return &PostgresStore{db: s.db, timeout: s.timeout, organizationID: strings.TrimSpace(organizationID), systemAccess: false}
}

func (s *PostgresStore) Migrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("postgres store is not initialized")
	}
	if ctx == nil {
		return errors.New("postgres migration context is required")
	}
	if s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	const migrationLock int64 = 0x4f4c4c414d414655
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLock); err != nil {
		return err
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS agent_missions (id TEXT PRIMARY KEY, version BIGINT NOT NULL, objective TEXT NOT NULL, provider TEXT NOT NULL DEFAULT '', model TEXT NOT NULL DEFAULT '', workspace TEXT NOT NULL DEFAULT '', project_id TEXT NOT NULL DEFAULT '', organization_id TEXT NOT NULL DEFAULT '', capabilities JSONB NOT NULL DEFAULT '[]'::jsonb, auto_run BOOLEAN NOT NULL DEFAULT FALSE, state TEXT NOT NULL, plan JSONB NOT NULL, approvals JSONB NOT NULL, artifacts JSONB NOT NULL, last_error TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, completed_at TIMESTAMPTZ NULL)`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS capabilities JSONB NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS organization_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS workspace_isolated BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS workspace_snapshot_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE agent_missions ADD COLUMN IF NOT EXISTS workspace_snapshot_sha256 TEXT NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS agent_events (id TEXT PRIMARY KEY, mission_id TEXT NOT NULL, organization_id TEXT NOT NULL DEFAULT '', type TEXT NOT NULL, step_id TEXT NOT NULL DEFAULT '', payload JSONB NULL, created_at TIMESTAMPTZ NOT NULL)`,
		`ALTER TABLE agent_events ADD COLUMN IF NOT EXISTS organization_id TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS agent_events_mission_created_idx ON agent_events (mission_id, created_at, id)`,
		`CREATE INDEX IF NOT EXISTS agent_missions_organization_updated_idx ON agent_missions (organization_id, updated_at, id)`,
		`ALTER TABLE agent_missions FORCE ROW LEVEL SECURITY`,
		`ALTER TABLE agent_events FORCE ROW LEVEL SECURITY`,
		`DROP POLICY IF EXISTS agent_missions_tenant_policy ON agent_missions`,
		`DROP POLICY IF EXISTS agent_events_tenant_policy ON agent_events`,
		`ALTER TABLE agent_missions ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE agent_events ENABLE ROW LEVEL SECURITY`,
		// BLOCKED security boundary: custom GUCs are caller-settable by the
		// current database role. A non-spoofable policy needs a separately
		// provisioned role/function or signed-context secret contract; do not
		// treat renaming these GUCs or SET LOCAL as protection.
		`CREATE POLICY agent_missions_tenant_policy ON agent_missions USING (current_setting('app.system_access', true) = '1' OR (organization_id <> '' AND organization_id = current_setting('app.current_organization_id', true))) WITH CHECK (current_setting('app.system_access', true) = '1' OR (organization_id <> '' AND organization_id = current_setting('app.current_organization_id', true)))`,
		`CREATE POLICY agent_events_tenant_policy ON agent_events USING (current_setting('app.system_access', true) = '1' OR (organization_id <> '' AND organization_id = current_setting('app.current_organization_id', true))) WITH CHECK (current_setting('app.system_access', true) = '1' OR (organization_id <> '' AND organization_id = current_setting('app.current_organization_id', true)))`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *PostgresStore) begin(ctx context.Context) (*sql.Tx, context.CancelFunc, error) {
	if s == nil || s.db == nil {
		return nil, nil, errors.New("postgres store is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if !s.systemAccess && strings.TrimSpace(s.organizationID) == "" {
		return nil, nil, errors.New("organization-scoped postgres store requires an organization id")
	}
	timeout := s.timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	operationCtx, cancel := context.WithTimeout(ctx, timeout)
	tx, err := s.db.BeginTx(operationCtx, nil)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	if !s.systemAccess {
		var privileged bool
		if err := tx.QueryRowContext(operationCtx, `SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&privileged); err != nil {
			_ = tx.Rollback()
			cancel()
			return nil, nil, err
		}
		if privileged {
			_ = tx.Rollback()
			cancel()
			return nil, nil, ErrPostgresTenantRequiresNonSuperuser
		}
	}
	org := s.organizationID
	if _, err := tx.ExecContext(operationCtx, `SELECT set_config('app.current_organization_id', $1, true), set_config('app.system_access', $2, true)`, org, boolString(s.systemAccess)); err != nil {
		_ = tx.Rollback()
		cancel()
		return nil, nil, err
	}
	return tx, cancel, nil
}

func boolString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func (s *PostgresStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *PostgresStore) GetMission(id string) (Mission, error) {
	if !validSnapshotID(id) {
		return Mission{}, os.ErrNotExist
	}
	tx, cancel, err := s.begin(context.Background())
	if err != nil {
		return Mission{}, err
	}
	defer cancel()
	defer tx.Rollback()
	mission, err := s.getMissionTx(tx, id)
	if err != nil {
		return Mission{}, err
	}
	if !s.systemAccess && mission.OrganizationID != s.organizationID {
		return Mission{}, os.ErrNotExist
	}
	mission, err = scrubMissionDLPInTransaction(tx, mission)
	if err != nil {
		return Mission{}, err
	}
	return mission, tx.Commit()
}

func (s *PostgresStore) getMissionTx(tx *sql.Tx, id string) (Mission, error) {
	var mission Mission
	var capabilities, plan, approvals, artifacts []byte
	var completed sql.NullTime
	err := tx.QueryRow(`SELECT id,version,objective,provider,model,workspace,project_id,organization_id,capabilities,workspace_isolated,workspace_snapshot_id,workspace_snapshot_sha256,auto_run,state,plan,approvals,artifacts,last_error,created_at,updated_at,completed_at FROM agent_missions WHERE id=$1 AND ($2 = '' OR organization_id=$2)`, id, s.organizationID).Scan(&mission.ID, &mission.Version, &mission.Objective, &mission.Provider, &mission.Model, &mission.Workspace, &mission.ProjectID, &mission.OrganizationID, &capabilities, &mission.WorkspaceIsolated, &mission.WorkspaceSnapshotID, &mission.WorkspaceSnapshotSHA256, &mission.AutoRun, &mission.State, &plan, &approvals, &artifacts, &mission.LastError, &mission.CreatedAt, &mission.UpdatedAt, &completed)
	if errors.Is(err, sql.ErrNoRows) {
		return Mission{}, os.ErrNotExist
	}
	if err != nil {
		return Mission{}, err
	}
	if err = json.Unmarshal(capabilities, &mission.Capabilities); err != nil {
		return Mission{}, err
	}
	if err = json.Unmarshal(plan, &mission.Plan); err != nil {
		return Mission{}, err
	}
	if err = json.Unmarshal(approvals, &mission.Approvals); err != nil {
		return Mission{}, err
	}
	if err = json.Unmarshal(artifacts, &mission.Artifacts); err != nil {
		return Mission{}, err
	}
	if completed.Valid {
		mission.CompletedAt = &completed.Time
	}
	return mission, nil
}

func scrubMissionDLPInTransaction(tx *sql.Tx, mission Mission) (Mission, error) {
	safe := redactMissionForPersistence(mission)
	if samePersistedMission(mission, safe) {
		return safe, nil
	}
	plan, err := json.Marshal(safe.Plan)
	if err != nil {
		return Mission{}, err
	}
	approvals, err := json.Marshal(safe.Approvals)
	if err != nil {
		return Mission{}, err
	}
	artifacts, err := json.Marshal(safe.Artifacts)
	if err != nil {
		return Mission{}, err
	}
	result, err := tx.Exec(`UPDATE agent_missions SET objective=$1,last_error=$2,plan=$3,approvals=$4,artifacts=$5 WHERE id=$6 AND version=$7 AND organization_id=$8`, safe.Objective, safe.LastError, plan, approvals, artifacts, mission.ID, mission.Version, mission.OrganizationID)
	if err != nil {
		return Mission{}, err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return Mission{}, err
	} else if rows != 1 {
		return Mission{}, ErrMissionVersionConflict
	}
	return safe, nil
}

func (s *PostgresStore) ListMissions() ([]Mission, error) {
	tx, cancel, err := s.begin(context.Background())
	if err != nil {
		return nil, err
	}
	defer cancel()
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id FROM agent_missions WHERE ($1 = '' OR organization_id = $1) ORDER BY updated_at ASC,id ASC`, s.organizationID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	var missions []Mission
	for _, id := range ids {
		mission, err := s.getMissionTx(tx, id)
		if err != nil {
			return nil, err
		}
		mission, err = scrubMissionDLPInTransaction(tx, mission)
		if err != nil {
			return nil, err
		}
		missions = append(missions, mission)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	sort.Slice(missions, func(i, j int) bool { return missions[i].UpdatedAt.Before(missions[j].UpdatedAt) })
	return missions, nil
}

func (s *PostgresStore) PutMission(mission Mission) error {
	if !validSnapshotID(mission.ID) {
		return errors.New("valid mission id is required")
	}
	if !s.systemAccess && strings.TrimSpace(mission.OrganizationID) != s.organizationID {
		return os.ErrPermission
	}
	mission, err := normalizeMissionForPersistence(mission)
	if err != nil {
		return err
	}
	capabilities, err := json.Marshal(mission.Capabilities)
	if err != nil {
		return err
	}
	plan, err := json.Marshal(mission.Plan)
	if err != nil {
		return err
	}
	approvals, err := json.Marshal(mission.Approvals)
	if err != nil {
		return err
	}
	artifacts, err := json.Marshal(mission.Artifacts)
	if err != nil {
		return err
	}
	tx, cancel, err := s.begin(context.Background())
	if err != nil {
		return err
	}
	defer cancel()
	defer tx.Rollback()
	var currentVersion int64
	var currentOrganization string
	err = tx.QueryRow(`SELECT version,organization_id FROM agent_missions WHERE id=$1 FOR UPDATE`, mission.ID).Scan(&currentVersion, &currentOrganization)
	if errors.Is(err, sql.ErrNoRows) {
		result, insertErr := tx.Exec(`INSERT INTO agent_missions (id,version,objective,provider,model,workspace,project_id,organization_id,capabilities,workspace_isolated,workspace_snapshot_id,workspace_snapshot_sha256,auto_run,state,plan,approvals,artifacts,last_error,created_at,updated_at,completed_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21) ON CONFLICT (id) DO NOTHING`, mission.ID, mission.Version, mission.Objective, mission.Provider, mission.Model, mission.Workspace, mission.ProjectID, mission.OrganizationID, capabilities, mission.WorkspaceIsolated, mission.WorkspaceSnapshotID, mission.WorkspaceSnapshotSHA256, mission.AutoRun, mission.State, plan, approvals, artifacts, mission.LastError, mission.CreatedAt, mission.UpdatedAt, mission.CompletedAt)
		if insertErr != nil {
			return insertErr
		}
		if rows, rowsErr := result.RowsAffected(); rowsErr != nil {
			return rowsErr
		} else if rows != 1 {
			return ErrMissionVersionConflict
		}
		return tx.Commit()
	}
	if err != nil {
		return err
	}
	if currentOrganization != "" && currentOrganization != mission.OrganizationID {
		return os.ErrPermission
	}
	if !validMissionVersionTransition(currentVersion, mission.Version) {
		return ErrMissionVersionConflict
	}
	if mission.Version == currentVersion {
		current, readErr := s.getMissionTx(tx, mission.ID)
		if readErr != nil {
			return readErr
		}
		if !samePersistedMission(redactMissionForPersistence(current), mission) {
			return ErrMissionVersionConflict
		}
	}
	result, err := tx.Exec(`UPDATE agent_missions SET version=$1,objective=$2,provider=$3,model=$4,workspace=$5,project_id=$6,organization_id=$7,capabilities=$8,workspace_isolated=$9,workspace_snapshot_id=$10,workspace_snapshot_sha256=$11,auto_run=$12,state=$13,plan=$14,approvals=$15,artifacts=$16,last_error=$17,updated_at=$18,completed_at=$19 WHERE id=$20 AND version=$21 AND organization_id=$22`, mission.Version, mission.Objective, mission.Provider, mission.Model, mission.Workspace, mission.ProjectID, mission.OrganizationID, capabilities, mission.WorkspaceIsolated, mission.WorkspaceSnapshotID, mission.WorkspaceSnapshotSHA256, mission.AutoRun, mission.State, plan, approvals, artifacts, mission.LastError, mission.UpdatedAt, mission.CompletedAt, mission.ID, currentVersion, currentOrganization)
	if err != nil {
		return err
	}
	if rows, rowsErr := result.RowsAffected(); rowsErr != nil {
		return rowsErr
	} else if rows != 1 {
		return ErrMissionVersionConflict
	}
	return tx.Commit()
}

func (s *PostgresStore) CreateMission(mission Mission) error {
	if !validSnapshotID(mission.ID) {
		return errors.New("valid mission id is required")
	}
	if !s.systemAccess && strings.TrimSpace(mission.OrganizationID) != s.organizationID {
		return os.ErrPermission
	}
	mission, err := normalizeMissionForPersistence(mission)
	if err != nil {
		return err
	}
	capabilities, err := json.Marshal(mission.Capabilities)
	if err != nil {
		return err
	}
	plan, err := json.Marshal(mission.Plan)
	if err != nil {
		return err
	}
	approvals, err := json.Marshal(mission.Approvals)
	if err != nil {
		return err
	}
	artifacts, err := json.Marshal(mission.Artifacts)
	if err != nil {
		return err
	}
	tx, cancel, err := s.begin(context.Background())
	if err != nil {
		return err
	}
	defer cancel()
	defer tx.Rollback()
	result, err := tx.Exec(`INSERT INTO agent_missions (id,version,objective,provider,model,workspace,project_id,organization_id,capabilities,workspace_isolated,workspace_snapshot_id,workspace_snapshot_sha256,auto_run,state,plan,approvals,artifacts,last_error,created_at,updated_at,completed_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21) ON CONFLICT (id) DO NOTHING`, mission.ID, mission.Version, mission.Objective, mission.Provider, mission.Model, mission.Workspace, mission.ProjectID, mission.OrganizationID, capabilities, mission.WorkspaceIsolated, mission.WorkspaceSnapshotID, mission.WorkspaceSnapshotSHA256, mission.AutoRun, mission.State, plan, approvals, artifacts, mission.LastError, mission.CreatedAt, mission.UpdatedAt, mission.CompletedAt)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrMissionAlreadyExists
	}
	return tx.Commit()
}

func (s *PostgresStore) PutMissionIfVersion(mission Mission, expectedVersion int64) error {
	if !validSnapshotID(mission.ID) {
		return errors.New("valid mission id is required")
	}
	if !validMissionVersionAdvance(expectedVersion, mission.Version) {
		return fmt.Errorf("mission version must advance exactly once from %d", expectedVersion)
	}
	if !s.systemAccess && strings.TrimSpace(mission.OrganizationID) != s.organizationID {
		return os.ErrPermission
	}
	mission, err := normalizeMissionForPersistence(mission)
	if err != nil {
		return err
	}
	capabilities, err := json.Marshal(mission.Capabilities)
	if err != nil {
		return err
	}
	plan, err := json.Marshal(mission.Plan)
	if err != nil {
		return err
	}
	approvals, err := json.Marshal(mission.Approvals)
	if err != nil {
		return err
	}
	artifacts, err := json.Marshal(mission.Artifacts)
	if err != nil {
		return err
	}
	tx, cancel, err := s.begin(context.Background())
	if err != nil {
		return err
	}
	defer cancel()
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE agent_missions SET version=$1,objective=$2,provider=$3,model=$4,workspace=$5,project_id=$6,organization_id=$7,capabilities=$8,workspace_isolated=$9,workspace_snapshot_id=$10,workspace_snapshot_sha256=$11,auto_run=$12,state=$13,plan=$14,approvals=$15,artifacts=$16,last_error=$17,updated_at=$18,completed_at=$19 WHERE id=$20 AND version=$21 AND ($22 = '' OR organization_id=$22)`, mission.Version, mission.Objective, mission.Provider, mission.Model, mission.Workspace, mission.ProjectID, mission.OrganizationID, capabilities, mission.WorkspaceIsolated, mission.WorkspaceSnapshotID, mission.WorkspaceSnapshotSHA256, mission.AutoRun, mission.State, plan, approvals, artifacts, mission.LastError, mission.UpdatedAt, mission.CompletedAt, mission.ID, expectedVersion, s.organizationID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrMissionVersionConflict
	}
	return tx.Commit()
}

func (s *PostgresStore) AppendEvent(event Event) error {
	if !validSnapshotID(event.ID) || !validSnapshotID(event.MissionID) {
		return errors.New("valid event and mission ids are required")
	}
	if !s.systemAccess && strings.TrimSpace(event.OrganizationID) != s.organizationID {
		return os.ErrPermission
	}
	event.Payload = RedactValue(event.Payload)
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}
	tx, cancel, err := s.begin(context.Background())
	if err != nil {
		return err
	}
	defer cancel()
	defer tx.Rollback()
	result, err := tx.Exec(`INSERT INTO agent_events (id,mission_id,organization_id,type,step_id,payload,created_at)
		SELECT $1,m.id,m.organization_id,$4,$5,$6,$7 FROM agent_missions AS m WHERE m.id=$2 AND m.organization_id=$3
		ON CONFLICT (id) DO UPDATE SET id=agent_events.id
		WHERE agent_events.mission_id=EXCLUDED.mission_id
		  AND agent_events.organization_id=EXCLUDED.organization_id
		  AND agent_events.type=EXCLUDED.type
		  AND agent_events.step_id=EXCLUDED.step_id
		  AND agent_events.payload IS NOT DISTINCT FROM EXCLUDED.payload
		  AND agent_events.created_at=EXCLUDED.created_at`, event.ID, event.MissionID, event.OrganizationID, event.Type, event.StepID, payload, event.CreatedAt)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		var existingID string
		existingErr := tx.QueryRow(`SELECT id FROM agent_events WHERE id=$1`, event.ID).Scan(&existingID)
		if existingErr == nil {
			return fmt.Errorf("event %s already exists with different content", event.ID)
		}
		if !errors.Is(existingErr, sql.ErrNoRows) {
			return existingErr
		}
		return fmt.Errorf("event %s was not inserted: mission is missing/outside the active organization or event ID conflicts", event.ID)
	}
	return tx.Commit()
}

func (s *PostgresStore) ListEvents(missionID string) ([]Event, error) {
	if !validSnapshotID(missionID) {
		return nil, os.ErrNotExist
	}
	tx, cancel, err := s.begin(context.Background())
	if err != nil {
		return nil, err
	}
	defer cancel()
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id,mission_id,organization_id,type,step_id,payload,created_at FROM agent_events WHERE mission_id=$1 AND ($2 = '' OR organization_id=$2) ORDER BY created_at ASC,id ASC`, missionID, s.organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var event Event
		var payload []byte
		if err := rows.Scan(&event.ID, &event.MissionID, &event.OrganizationID, &event.Type, &event.StepID, &payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		if len(payload) > 0 && string(payload) != "null" {
			if err := json.Unmarshal(payload, &event.Payload); err != nil {
				return nil, err
			}
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for index := range events {
		original := events[index].Payload
		safePayload := RedactValue(original)
		if !reflect.DeepEqual(original, safePayload) {
			encoded, err := json.Marshal(safePayload)
			if err != nil {
				return nil, err
			}
			result, err := tx.Exec(`UPDATE agent_events SET payload=$1 WHERE id=$2 AND mission_id=$3 AND organization_id=$4`, encoded, events[index].ID, events[index].MissionID, events[index].OrganizationID)
			if err != nil {
				return nil, err
			}
			if rows, err := result.RowsAffected(); err != nil {
				return nil, err
			} else if rows != 1 {
				return nil, errors.New("event changed while applying legacy DLP redaction")
			}
		}
		events[index].Payload = safePayload
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *PostgresStore) SetTimeout(timeout time.Duration) {
	if timeout > 0 {
		s.timeout = timeout
	}
}

func (s *PostgresStore) String() string {
	return fmt.Sprintf("PostgresStore(timeout=%s, organization=%q, system=%t)", s.timeout, s.organizationID, s.systemAccess)
}
