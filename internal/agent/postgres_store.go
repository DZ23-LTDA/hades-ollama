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
	db               *sql.DB
	timeout          time.Duration
	organizationID   string
	tenantContextKey []byte
	requestContext   context.Context
}

var ErrPostgresTenantRequiresNonSuperuser = errors.New("tenant-scoped postgres store requires a non-superuser role")
var ErrPostgresTenantIsolationUnavailable = errors.New("PostgreSQL agent storage requires the signed-tenant runtime and separate migrator configuration")

// WithOrganization returns a tenant-only view. The view shares the runtime
// connection pool but carries an immutable organization scope.
func (s *PostgresStore) WithOrganization(organizationID string) *PostgresStore {
	return s.WithOrganizationContext(context.Background(), organizationID)
}

// WithOrganizationContext binds database operations to the lifetime of one
// request or background operation while retaining the immutable tenant scope.
func (s *PostgresStore) WithOrganizationContext(ctx context.Context, organizationID string) *PostgresStore {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &PostgresStore{db: s.db, timeout: s.timeout, organizationID: strings.TrimSpace(organizationID), tenantContextKey: s.tenantContextKey, requestContext: ctx}
}

func (s *PostgresStore) begin(ctx context.Context) (*sql.Tx, context.Context, context.CancelFunc, error) {
	if s == nil || s.db == nil {
		return nil, nil, nil, errors.New("postgres store is not initialized")
	}
	if ctx == nil {
		ctx = s.requestContext
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(s.organizationID) == "" {
		return nil, nil, nil, errors.New("organization-scoped postgres store requires an organization id")
	}
	if err := validatePostgresOrganizationID(s.organizationID); err != nil {
		return nil, nil, nil, err
	}
	timeout := s.timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	operationCtx, cancel := context.WithTimeout(ctx, timeout)
	tx, err := s.db.BeginTx(operationCtx, nil)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	tenantContext, err := s.signedTenantContext(s.organizationID, time.Now())
	if err == nil {
		_, err = tx.ExecContext(operationCtx, `SELECT set_config('app.tenant_context', $1, true)`, tenantContext)
	}
	if err != nil {
		_ = tx.Rollback()
		cancel()
		return nil, nil, nil, err
	}
	return tx, operationCtx, cancel, nil
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
	tx, operationCtx, cancel, err := s.begin(nil)
	if err != nil {
		return Mission{}, err
	}
	defer cancel()
	defer tx.Rollback()
	mission, err := s.getMissionTx(operationCtx, tx, id)
	if err != nil {
		return Mission{}, err
	}
	if mission.OrganizationID != s.organizationID {
		return Mission{}, os.ErrNotExist
	}
	mission, err = scrubMissionDLPInTransaction(operationCtx, tx, mission)
	if err != nil {
		return Mission{}, err
	}
	return mission, tx.Commit()
}

func (s *PostgresStore) getMissionTx(ctx context.Context, tx *sql.Tx, id string) (Mission, error) {
	var mission Mission
	var capabilities, plan, approvals, artifacts []byte
	var completed sql.NullTime
	err := tx.QueryRowContext(ctx, `SELECT id,version,objective,provider,model,workspace,workspace_identity,project_id,organization_id,capabilities,workspace_isolated,workspace_snapshot_id,workspace_snapshot_sha256,auto_run,state,plan,approvals,artifacts,last_error,created_at,updated_at,completed_at FROM agent_missions WHERE id=$1 AND ($2 = '' OR organization_id=$2)`, id, s.organizationID).Scan(&mission.ID, &mission.Version, &mission.Objective, &mission.Provider, &mission.Model, &mission.Workspace, &mission.WorkspaceIdentity, &mission.ProjectID, &mission.OrganizationID, &capabilities, &mission.WorkspaceIsolated, &mission.WorkspaceSnapshotID, &mission.WorkspaceSnapshotSHA256, &mission.AutoRun, &mission.State, &plan, &approvals, &artifacts, &mission.LastError, &mission.CreatedAt, &mission.UpdatedAt, &completed)
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

func scrubMissionDLPInTransaction(ctx context.Context, tx *sql.Tx, mission Mission) (Mission, error) {
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
	result, err := tx.ExecContext(ctx, `UPDATE agent_missions SET objective=$1,last_error=$2,plan=$3,approvals=$4,artifacts=$5 WHERE id=$6 AND version=$7 AND organization_id=$8`, safe.Objective, safe.LastError, plan, approvals, artifacts, mission.ID, mission.Version, mission.OrganizationID)
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
	tx, operationCtx, cancel, err := s.begin(nil)
	if err != nil {
		return nil, err
	}
	defer cancel()
	defer tx.Rollback()
	rows, err := tx.QueryContext(operationCtx, `SELECT id FROM agent_missions WHERE ($1 = '' OR organization_id = $1) ORDER BY updated_at ASC,id ASC`, s.organizationID)
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
		mission, err := s.getMissionTx(operationCtx, tx, id)
		if err != nil {
			return nil, err
		}
		mission, err = scrubMissionDLPInTransaction(operationCtx, tx, mission)
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
	if strings.TrimSpace(mission.OrganizationID) != s.organizationID {
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
	tx, operationCtx, cancel, err := s.begin(nil)
	if err != nil {
		return err
	}
	defer cancel()
	defer tx.Rollback()
	var currentVersion int64
	var currentOrganization string
	err = tx.QueryRowContext(operationCtx, `SELECT version,organization_id FROM agent_missions WHERE id=$1 FOR UPDATE`, mission.ID).Scan(&currentVersion, &currentOrganization)
	if errors.Is(err, sql.ErrNoRows) {
		result, insertErr := tx.ExecContext(operationCtx, `INSERT INTO agent_missions (id,version,objective,provider,model,workspace,workspace_identity,project_id,organization_id,capabilities,workspace_isolated,workspace_snapshot_id,workspace_snapshot_sha256,auto_run,state,plan,approvals,artifacts,last_error,created_at,updated_at,completed_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22) ON CONFLICT (id) DO NOTHING`, mission.ID, mission.Version, mission.Objective, mission.Provider, mission.Model, mission.Workspace, mission.WorkspaceIdentity, mission.ProjectID, mission.OrganizationID, capabilities, mission.WorkspaceIsolated, mission.WorkspaceSnapshotID, mission.WorkspaceSnapshotSHA256, mission.AutoRun, mission.State, plan, approvals, artifacts, mission.LastError, mission.CreatedAt, mission.UpdatedAt, mission.CompletedAt)
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
		current, readErr := s.getMissionTx(operationCtx, tx, mission.ID)
		if readErr != nil {
			return readErr
		}
		if !samePersistedMission(redactMissionForPersistence(current), mission) {
			return ErrMissionVersionConflict
		}
	}
	result, err := tx.ExecContext(operationCtx, `UPDATE agent_missions SET version=$1,objective=$2,provider=$3,model=$4,workspace=$5,workspace_identity=$6,project_id=$7,organization_id=$8,capabilities=$9,workspace_isolated=$10,workspace_snapshot_id=$11,workspace_snapshot_sha256=$12,auto_run=$13,state=$14,plan=$15,approvals=$16,artifacts=$17,last_error=$18,updated_at=$19,completed_at=$20 WHERE id=$21 AND version=$22 AND organization_id=$23`, mission.Version, mission.Objective, mission.Provider, mission.Model, mission.Workspace, mission.WorkspaceIdentity, mission.ProjectID, mission.OrganizationID, capabilities, mission.WorkspaceIsolated, mission.WorkspaceSnapshotID, mission.WorkspaceSnapshotSHA256, mission.AutoRun, mission.State, plan, approvals, artifacts, mission.LastError, mission.UpdatedAt, mission.CompletedAt, mission.ID, currentVersion, currentOrganization)
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
	if strings.TrimSpace(mission.OrganizationID) != s.organizationID {
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
	tx, operationCtx, cancel, err := s.begin(nil)
	if err != nil {
		return err
	}
	defer cancel()
	defer tx.Rollback()
	result, err := tx.ExecContext(operationCtx, `INSERT INTO agent_missions (id,version,objective,provider,model,workspace,workspace_identity,project_id,organization_id,capabilities,workspace_isolated,workspace_snapshot_id,workspace_snapshot_sha256,auto_run,state,plan,approvals,artifacts,last_error,created_at,updated_at,completed_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22) ON CONFLICT (id) DO NOTHING`, mission.ID, mission.Version, mission.Objective, mission.Provider, mission.Model, mission.Workspace, mission.WorkspaceIdentity, mission.ProjectID, mission.OrganizationID, capabilities, mission.WorkspaceIsolated, mission.WorkspaceSnapshotID, mission.WorkspaceSnapshotSHA256, mission.AutoRun, mission.State, plan, approvals, artifacts, mission.LastError, mission.CreatedAt, mission.UpdatedAt, mission.CompletedAt)
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
	if strings.TrimSpace(mission.OrganizationID) != s.organizationID {
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
	tx, operationCtx, cancel, err := s.begin(nil)
	if err != nil {
		return err
	}
	defer cancel()
	defer tx.Rollback()
	result, err := tx.ExecContext(operationCtx, `UPDATE agent_missions SET version=$1,objective=$2,provider=$3,model=$4,workspace=$5,workspace_identity=$6,project_id=$7,organization_id=$8,capabilities=$9,workspace_isolated=$10,workspace_snapshot_id=$11,workspace_snapshot_sha256=$12,auto_run=$13,state=$14,plan=$15,approvals=$16,artifacts=$17,last_error=$18,updated_at=$19,completed_at=$20 WHERE id=$21 AND version=$22 AND ($23 = '' OR organization_id=$23)`, mission.Version, mission.Objective, mission.Provider, mission.Model, mission.Workspace, mission.WorkspaceIdentity, mission.ProjectID, mission.OrganizationID, capabilities, mission.WorkspaceIsolated, mission.WorkspaceSnapshotID, mission.WorkspaceSnapshotSHA256, mission.AutoRun, mission.State, plan, approvals, artifacts, mission.LastError, mission.UpdatedAt, mission.CompletedAt, mission.ID, expectedVersion, s.organizationID)
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
	if strings.TrimSpace(event.OrganizationID) != s.organizationID {
		return os.ErrPermission
	}
	event.Payload = RedactValue(event.Payload)
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}
	tx, operationCtx, cancel, err := s.begin(nil)
	if err != nil {
		return err
	}
	defer cancel()
	defer tx.Rollback()
	result, err := tx.ExecContext(operationCtx, `INSERT INTO agent_events (id,mission_id,organization_id,type,step_id,payload,created_at)
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
		existingErr := tx.QueryRowContext(operationCtx, `SELECT id FROM agent_events WHERE id=$1`, event.ID).Scan(&existingID)
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
	tx, operationCtx, cancel, err := s.begin(nil)
	if err != nil {
		return nil, err
	}
	defer cancel()
	defer tx.Rollback()
	rows, err := tx.QueryContext(operationCtx, `SELECT id,mission_id,organization_id,type,step_id,payload,created_at FROM agent_events WHERE mission_id=$1 AND ($2 = '' OR organization_id=$2) ORDER BY created_at ASC,id ASC`, missionID, s.organizationID)
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
			result, err := tx.ExecContext(operationCtx, `UPDATE agent_events SET payload=$1 WHERE id=$2 AND mission_id=$3 AND organization_id=$4`, encoded, events[index].ID, events[index].MissionID, events[index].OrganizationID)
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
	return fmt.Sprintf("PostgresStore(timeout=%s, organization=%q, tenant_only=true)", s.timeout, s.organizationID)
}
