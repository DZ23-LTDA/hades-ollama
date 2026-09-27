package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

type Store interface {
	GetMission(id string) (Mission, error)
	ListMissions() ([]Mission, error)
	CreateMission(mission Mission) error
	PutMission(mission Mission) error
	PutMissionIfVersion(mission Mission, expectedVersion int64) error
	AppendEvent(event Event) error
	ListEvents(missionID string) ([]Event, error)
}

var ErrMissionVersionConflict = errors.New("mission version conflict")
var ErrMissionAlreadyExists = errors.New("mission already exists")

const (
	maxJSONStoreEventPayloadBytes = 1 << 20
	maxJSONStoreEventCount        = 10_000
	maxJSONStoreEventFileBytes    = 20 << 20
	maxJSONStoreEventFiles        = 10_000
	maxJSONStoreTotalEvents       = 100_000
	maxJSONStoreTotalEventBytes   = 256 << 20
	maxMissionRecordBytes         = 8 << 20
	maxMissionRecordNodes         = 100_000
	maxMissionRecordDepth         = 64
	maxMissionPlanSteps           = 32
	maxMissionApprovals           = 32
	maxMissionArtifacts           = 512
	maxJSONStoreMissions          = 10_000
	maxJSONStoreTotalMissionBytes = 512 << 20
)

type JSONStore struct {
	mu         sync.RWMutex
	root       string
	missions   map[string]Mission
	events     map[string][]Event
	persistent bool
}

func NewJSONStore(root string) (*JSONStore, error) {
	if strings.TrimSpace(root) == "" {
		return NewMemoryStore(), nil
	}
	if err := os.MkdirAll(filepath.Join(root, "missions"), 0o700); err != nil {
		return nil, fmt.Errorf("create agent store: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "events"), 0o700); err != nil {
		return nil, fmt.Errorf("create agent event store: %w", err)
	}
	store := &JSONStore{root: root, missions: make(map[string]Mission), events: make(map[string][]Event), persistent: true}
	entries, err := readBoundedStoreDirEntries(filepath.Join(root, "missions"), maxJSONStoreMissions)
	if err != nil {
		return nil, fmt.Errorf("read agent missions: %w", err)
	}
	if err := checkJSONStoreMissionFileEntries(entries, ""); err != nil {
		return nil, fmt.Errorf("validate agent mission file totals: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		var mission Mission
		path := filepath.Join(root, "missions", entry.Name())
		if err := withMissionStoreLock(root, func() error {
			if err := readMissionJSON(path, &mission); err != nil {
				return err
			}
			previous := mission
			safe := redactMissionForPersistence(mission)
			mission = safe
			if !samePersistedMission(previous, safe) {
				return writeMissionJSONAtomic(path, safe)
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("redact stored mission %s: %w", id, err)
		}
		store.missions[id] = cloneMission(mission)
	}
	eventEntries, err := readBoundedStoreDirEntries(filepath.Join(root, "events"), maxJSONStoreEventFiles)
	if err != nil {
		return nil, fmt.Errorf("read agent events: %w", err)
	}
	for _, entry := range eventEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		var events []Event
		path := filepath.Join(root, "events", entry.Name())
		if err := withMissionStoreLock(root, func() error {
			if err := readJSONBounded(path, &events, maxJSONStoreEventFileBytes); err != nil {
				return err
			}
			if err := validateJSONStoreEvents(events); err != nil {
				return err
			}
			for index := range events {
				if events[index].MissionID != id {
					return fmt.Errorf("event %s belongs to mission %s", events[index].ID, events[index].MissionID)
				}
				events[index].Payload = RedactValue(events[index].Payload)
			}
			return writeJSONAtomicBounded(path, events, maxJSONStoreEventFileBytes, "JSONStore event file")
		}); err != nil {
			return nil, fmt.Errorf("redact stored mission events %s: %w", id, err)
		}
		store.events[id] = cloneEvents(events)
	}
	if err := withMissionStoreLock(root, func() error {
		return validateJSONStoreAggregateMissionQuotas(filepath.Join(root, "missions"), "", Mission{}, false)
	}); err != nil {
		return nil, fmt.Errorf("validate aggregate agent mission store: %w", err)
	}
	if err := withMissionStoreLock(root, func() error {
		return validateJSONStoreAggregateEventQuotas(filepath.Join(root, "events"), "", nil)
	}); err != nil {
		return nil, fmt.Errorf("validate aggregate agent event store: %w", err)
	}
	return store, nil
}

func NewMemoryStore() *JSONStore {
	return &JSONStore{missions: make(map[string]Mission), events: make(map[string][]Event)}
}

func (s *JSONStore) GetMission(id string) (Mission, error) {
	if !validSnapshotID(id) {
		return Mission{}, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.persistent {
		mission, ok := s.missions[id]
		if !ok {
			return Mission{}, os.ErrNotExist
		}
		return cloneMission(mission), nil
	}
	var result Mission
	err := withMissionStoreLock(s.root, func() error {
		path := filepath.Join(s.root, "missions", id+".json")
		var mission Mission
		if err := readMissionJSON(path, &mission); err != nil {
			delete(s.missions, id)
			return err
		}
		if mission.ID != id {
			delete(s.missions, id)
			return errors.New("stored mission id does not match its filename")
		}
		safe := redactMissionForPersistence(mission)
		if !samePersistedMission(mission, safe) {
			if err := writeMissionJSONAtomic(path, safe); err != nil {
				return err
			}
		}
		s.missions[id] = cloneMission(safe)
		result = cloneMission(safe)
		return nil
	})
	if err != nil {
		return Mission{}, err
	}
	return result, nil
}

func (s *JSONStore) ListMissions() ([]Mission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.persistent {
		refreshed := make(map[string]Mission)
		err := withMissionStoreLock(s.root, func() error {
			entries, err := readBoundedStoreDirEntries(filepath.Join(s.root, "missions"), maxJSONStoreMissions)
			if err != nil {
				return err
			}
			if err := checkJSONStoreMissionFileEntries(entries, ""); err != nil {
				return err
			}
			for _, entry := range entries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
					continue
				}
				id := strings.TrimSuffix(entry.Name(), ".json")
				if !validSnapshotID(id) {
					return errors.New("stored mission filename contains an invalid id")
				}
				path := filepath.Join(s.root, "missions", entry.Name())
				var mission Mission
				if err := readMissionJSON(path, &mission); err != nil {
					return err
				}
				if mission.ID != id {
					return errors.New("stored mission id does not match its filename")
				}
				safe := redactMissionForPersistence(mission)
				if !samePersistedMission(mission, safe) {
					if err := writeMissionJSONAtomic(path, safe); err != nil {
						return err
					}
				}
				refreshed[id] = cloneMission(safe)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		s.missions = refreshed
	}
	missions := make([]Mission, 0, len(s.missions))
	for _, mission := range s.missions {
		missions = append(missions, cloneMission(mission))
	}
	sortMissions(missions)
	return missions, nil
}

func (s *JSONStore) PutMission(mission Mission) error {
	if !validSnapshotID(mission.ID) {
		return errors.New("valid mission id is required")
	}
	stored, err := normalizeMissionForPersistence(mission)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.persistent {
		if current, exists := s.missions[mission.ID]; exists {
			current = redactMissionForPersistence(current)
			if current.OrganizationID != "" && current.OrganizationID != stored.OrganizationID {
				s.missions[mission.ID] = cloneMission(current)
				return os.ErrPermission
			}
			if !validMissionVersionTransition(current.Version, stored.Version) {
				s.missions[mission.ID] = cloneMission(current)
				return ErrMissionVersionConflict
			}
			if current.Version == stored.Version && !samePersistedMission(current, stored) {
				s.missions[mission.ID] = cloneMission(current)
				return ErrMissionVersionConflict
			}
		}
		if err := validateInMemoryJSONStoreMissionQuotas(s.missions, mission.ID, stored); err != nil {
			return err
		}
		s.missions[mission.ID] = cloneMission(stored)
		return nil
	}
	return withMissionStoreLock(s.root, func() error {
		path := filepath.Join(s.root, "missions", mission.ID+".json")
		var diskCurrent Mission
		if err := readMissionJSON(path, &diskCurrent); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if err := writeMissionJSONAtomic(path, stored); err != nil {
				return err
			}
			s.missions[mission.ID] = cloneMission(stored)
			return nil
		}

		diskStored := redactMissionForPersistence(diskCurrent)
		if diskCurrent.OrganizationID != "" && diskCurrent.OrganizationID != stored.OrganizationID {
			s.missions[mission.ID] = cloneMission(diskStored)
			return os.ErrPermission
		}
		if !validMissionVersionTransition(diskCurrent.Version, stored.Version) {
			s.missions[mission.ID] = cloneMission(diskStored)
			return ErrMissionVersionConflict
		}
		if diskCurrent.Version == stored.Version {
			if !samePersistedMission(diskStored, stored) {
				s.missions[mission.ID] = cloneMission(diskStored)
				return ErrMissionVersionConflict
			}
			if !samePersistedMission(diskCurrent, diskStored) {
				if err := writeMissionJSONAtomic(path, diskStored); err != nil {
					return err
				}
			}
			s.missions[mission.ID] = cloneMission(diskStored)
			return nil
		}
		if err := writeMissionJSONAtomic(path, stored); err != nil {
			return err
		}
		s.missions[mission.ID] = cloneMission(stored)
		return nil
	})
}

func (s *JSONStore) CreateMission(mission Mission) error {
	if !validSnapshotID(mission.ID) {
		return errors.New("valid mission id is required")
	}
	stored, err := normalizeMissionForPersistence(mission)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.persistent {
		if _, exists := s.missions[mission.ID]; exists {
			return ErrMissionAlreadyExists
		}
		if err := validateInMemoryJSONStoreMissionQuotas(s.missions, mission.ID, stored); err != nil {
			return err
		}
		s.missions[mission.ID] = cloneMission(stored)
		return nil
	}
	return withMissionStoreLock(s.root, func() error {
		path := filepath.Join(s.root, "missions", mission.ID+".json")
		var existing Mission
		if err := readMissionJSON(path, &existing); err == nil {
			return ErrMissionAlreadyExists
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := writeMissionJSONAtomic(path, stored); err != nil {
			return err
		}
		s.missions[mission.ID] = cloneMission(stored)
		return nil
	})
}

func (s *JSONStore) PutMissionIfVersion(mission Mission, expectedVersion int64) error {
	if !validSnapshotID(mission.ID) {
		return errors.New("valid mission id is required")
	}
	if !validMissionVersionAdvance(expectedVersion, mission.Version) {
		return fmt.Errorf("mission version must advance exactly once from %d", expectedVersion)
	}
	stored, err := normalizeMissionForPersistence(mission)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.persistent {
		current, ok := s.missions[mission.ID]
		if !ok || current.Version != expectedVersion {
			return ErrMissionVersionConflict
		}
		if err := validateInMemoryJSONStoreMissionQuotas(s.missions, mission.ID, stored); err != nil {
			return err
		}
		s.missions[mission.ID] = cloneMission(stored)
		return nil
	}
	return withMissionStoreLock(s.root, func() error {
		path := filepath.Join(s.root, "missions", mission.ID+".json")
		var diskCurrent Mission
		if err := readMissionJSON(path, &diskCurrent); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				delete(s.missions, mission.ID)
				return ErrMissionVersionConflict
			}
			return err
		}
		diskStored := redactMissionForPersistence(diskCurrent)
		if diskCurrent.OrganizationID != "" && diskCurrent.OrganizationID != stored.OrganizationID {
			s.missions[mission.ID] = cloneMission(diskStored)
			return os.ErrPermission
		}
		if diskCurrent.Version != expectedVersion {
			s.missions[mission.ID] = cloneMission(diskStored)
			return ErrMissionVersionConflict
		}
		previous := cloneMission(diskStored)
		if err := writeMissionJSONAtomic(path, stored); err != nil {
			s.missions[mission.ID] = previous
			return err
		}
		s.missions[mission.ID] = cloneMission(stored)
		return nil
	})
}

func validMissionVersionAdvance(expectedVersion, nextVersion int64) bool {
	return expectedVersion >= 0 && expectedVersion < 9223372036854775807 && nextVersion == expectedVersion+1
}

func validMissionVersionTransition(currentVersion, nextVersion int64) bool {
	return nextVersion == currentVersion || validMissionVersionAdvance(currentVersion, nextVersion)
}

func (s *JSONStore) AppendEvent(event Event) error {
	if !validSnapshotID(event.MissionID) || !validSnapshotID(event.ID) {
		return errors.New("valid event and mission ids are required")
	}
	if err := validateJSONStoreEventPayload(event.Payload); err != nil {
		return err
	}
	event.Payload = RedactValue(event.Payload)
	if err := validateJSONStoreEventPayload(event.Payload); err != nil {
		return err
	}
	event.Payload = cloneJSONValue(event.Payload)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.persistent {
		mission, ok := s.missions[event.MissionID]
		if !ok {
			return os.ErrNotExist
		}
		if mission.OrganizationID != event.OrganizationID {
			return os.ErrPermission
		}
		for _, existing := range s.events[event.MissionID] {
			if existing.ID == event.ID {
				if samePersistedEvent(existing, event) {
					return nil
				}
				return fmt.Errorf("event id %s already exists with different content", event.ID)
			}
		}
		if len(s.events[event.MissionID]) >= maxJSONStoreEventCount {
			return errors.New("event count exceeds persistence limit")
		}
		candidate := append(cloneEvents(s.events[event.MissionID]), event)
		if err := validateInMemoryJSONStoreEventQuotas(s.events, event.MissionID, candidate); err != nil {
			return err
		}
		s.events[event.MissionID] = append(s.events[event.MissionID], cloneEvent(event))
		return nil
	}
	path := filepath.Join(s.root, "events", event.MissionID+".json")
	var events []Event
	if err := withMissionStoreLock(s.root, func() error {
		var mission Mission
		if err := readMissionJSON(filepath.Join(s.root, "missions", event.MissionID+".json"), &mission); err != nil {
			return err
		}
		if mission.ID != event.MissionID {
			return errors.New("stored mission id does not match its filename")
		}
		s.missions[event.MissionID] = cloneMission(mission)
		if mission.OrganizationID != event.OrganizationID {
			return os.ErrPermission
		}
		if err := readJSONBounded(path, &events, maxJSONStoreEventFileBytes); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				events = []Event{}
			} else {
				return err
			}
		}
		if err := validateJSONStoreEvents(events); err != nil {
			return err
		}
		for index := range events {
			events[index] = cloneEvent(events[index])
			events[index].Payload = RedactValue(events[index].Payload)
		}
		for _, existing := range events {
			if existing.ID == event.ID {
				if samePersistedEvent(existing, event) {
					return nil
				}
				return fmt.Errorf("event id %s already exists with different content", event.ID)
			}
		}
		if len(events) >= maxJSONStoreEventCount {
			return errors.New("event count exceeds persistence limit")
		}
		events = append(events, event)
		if err := validateJSONStoreAggregateEventQuotas(filepath.Join(s.root, "events"), event.MissionID, events); err != nil {
			return err
		}
		return writeJSONAtomicBounded(path, events, maxJSONStoreEventFileBytes, "JSONStore event file")
	}); err != nil {
		return err
	}
	s.events[event.MissionID] = cloneEvents(events)
	return nil
}

func samePersistedEvent(left, right Event) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func redactMissionForPersistence(mission Mission) Mission {
	safe := cloneMission(mission)
	safe.Objective = RedactDLP(safe.Objective)
	safe.LastError = RedactDLP(safe.LastError)
	for index := range safe.Plan {
		safe.Plan[index].Title = RedactDLP(safe.Plan[index].Title)
		if safe.Plan[index].Input != nil {
			safe.Plan[index].Input, _ = RedactValue(safe.Plan[index].Input).(map[string]any)
		}
		safe.Plan[index].Result = RedactValue(safe.Plan[index].Result)
		safe.Plan[index].Error = RedactDLP(safe.Plan[index].Error)
	}
	for index := range safe.Approvals {
		safe.Approvals[index].Reason = RedactDLP(safe.Approvals[index].Reason)
	}
	for index := range safe.Artifacts {
		safe.Artifacts[index].Name = RedactDLP(safe.Artifacts[index].Name)
		safe.Artifacts[index].Path = RedactDLP(safe.Artifacts[index].Path)
		safe.Artifacts[index].MediaType = RedactDLP(safe.Artifacts[index].MediaType)
	}
	return safe
}

func normalizeMissionForPersistence(mission Mission) (Mission, error) {
	if err := validateMissionPersistenceBounds(mission); err != nil {
		return Mission{}, err
	}
	encoded, err := json.Marshal(mission)
	if err != nil {
		return Mission{}, fmt.Errorf("mission is not JSON-serializable: %w", err)
	}
	var normalized Mission
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		return Mission{}, fmt.Errorf("normalize mission JSON: %w", err)
	}
	normalized = redactMissionForPersistence(normalized)
	if err := validateMissionPersistenceBounds(normalized); err != nil {
		return Mission{}, err
	}
	return normalized, nil
}

type missionJSONBudget struct {
	nodes int
	bytes int
}

func validateMissionPersistenceBounds(mission Mission) error {
	if len(mission.Plan) > maxMissionPlanSteps || len(mission.Approvals) > maxMissionApprovals || len(mission.Artifacts) > maxMissionArtifacts || len(mission.Capabilities) > 64 {
		return errors.New("mission collection exceeds persistence limit")
	}
	budget := missionJSONBudget{}
	if err := validateBoundedJSONValue(reflect.ValueOf(mission), 0, &budget); err != nil {
		return fmt.Errorf("mission exceeds bounded JSON limits: %w", err)
	}
	encoded, err := json.MarshalIndent(mission, "", "  ")
	if err != nil {
		return fmt.Errorf("mission is not JSON-serializable: %w", err)
	}
	if len(encoded)+1 > maxMissionRecordBytes {
		return errors.New("mission record exceeds persistence limit")
	}
	return nil
}

func validateBoundedJSONValue(value reflect.Value, depth int, budget *missionJSONBudget) error {
	if !value.IsValid() {
		return nil
	}
	jsonMarshalerType := reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	timeType := reflect.TypeOf(time.Time{})
	if value.Type().Implements(jsonMarshalerType) && value.Type() != timeType && !(value.Kind() == reflect.Pointer && value.Type().Elem() == timeType) {
		return errors.New("custom JSON marshaler is not allowed in bounded persistence values")
	}
	if depth > maxMissionRecordDepth {
		return errors.New("JSON nesting depth exceeds persistence limit")
	}
	budget.nodes++
	if budget.nodes > maxMissionRecordNodes {
		return errors.New("JSON node count exceeds persistence limit")
	}
	addBytes := func(size int) error {
		if size < 0 || size > maxMissionRecordBytes-budget.bytes {
			return errors.New("JSON encoded size exceeds persistence limit")
		}
		budget.bytes += size
		return nil
	}
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		return validateBoundedJSONValue(value.Elem(), depth+1, budget)
	case reflect.String:
		return addBytes(value.Len())
	case reflect.Map:
		if value.IsNil() {
			return nil
		}
		if value.Len() > maxMissionRecordNodes-budget.nodes {
			return errors.New("JSON map exceeds persistence limit")
		}
		iter := value.MapRange()
		for iter.Next() {
			if err := validateBoundedJSONValue(iter.Key(), depth+1, budget); err != nil {
				return err
			}
			if err := validateBoundedJSONValue(iter.Value(), depth+1, budget); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice && value.IsNil() {
			return nil
		}
		if value.Len() > maxMissionRecordNodes-budget.nodes {
			return errors.New("JSON array exceeds persistence limit")
		}
		for index := 0; index < value.Len(); index++ {
			if err := validateBoundedJSONValue(value.Index(index), depth+1, budget); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			field := value.Type().Field(index)
			if field.PkgPath != "" || field.Tag.Get("json") == "-" {
				continue
			}
			if err := validateBoundedJSONValue(value.Field(index), depth+1, budget); err != nil {
				return err
			}
		}
	case reflect.Bool:
		return addBytes(1)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return addBytes(16)
	case reflect.Invalid:
		return nil
	default:
		return fmt.Errorf("unsupported JSON value kind %s", value.Kind())
	}
	return nil
}

func readMissionJSON(path string, mission *Mission) error {
	if err := readJSONBounded(path, mission, maxMissionRecordBytes); err != nil {
		return err
	}
	return validateMissionPersistenceBounds(*mission)
}

func writeMissionJSONAtomic(path string, mission Mission) error {
	if err := validateMissionPersistenceBounds(mission); err != nil {
		return err
	}
	if err := validateJSONStoreAggregateMissionQuotas(filepath.Dir(path), mission.ID, mission, true); err != nil {
		return err
	}
	return writeJSONAtomicBounded(path, mission, maxMissionRecordBytes, "mission record")
}

func checkJSONStoreMissionFileEntries(entries []os.DirEntry, replacementID string) error {
	files := 0
	var totalBytes int64
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" || strings.TrimSuffix(entry.Name(), ".json") == replacementID {
			continue
		}
		files++
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() < 0 || info.Size() > maxMissionRecordBytes {
			return errors.New("mission file exceeds persistence limit")
		}
		totalBytes += info.Size()
		if files > maxJSONStoreMissions || totalBytes > maxJSONStoreTotalMissionBytes {
			return errors.New("aggregate mission store exceeds persistence limit")
		}
	}
	return nil
}

func readBoundedStoreDirEntries(path string, maxEntries int) ([]os.DirEntry, error) {
	if maxEntries <= 0 {
		return nil, errors.New("store directory entry limit must be positive")
	}
	directory, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	entries, err := directory.ReadDir(maxEntries + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > maxEntries {
		return nil, errors.New("store directory entry count exceeds persistence limit")
	}
	return entries, nil
}

func validateJSONStoreAggregateMissionQuotas(missionsDir, replacementID string, replacement Mission, hasReplacement bool) error {
	entries, err := readBoundedStoreDirEntries(missionsDir, maxJSONStoreMissions)
	if err != nil {
		return err
	}
	if err := checkJSONStoreMissionFileEntries(entries, replacementID); err != nil {
		return err
	}
	files := 0
	var totalBytes int64
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" || strings.TrimSuffix(entry.Name(), ".json") == replacementID {
			continue
		}
		files++
		info, err := entry.Info()
		if err != nil {
			return err
		}
		totalBytes += info.Size()
	}
	if hasReplacement {
		encoded, err := json.MarshalIndent(replacement, "", "  ")
		if err != nil {
			return err
		}
		files++
		totalBytes += int64(len(encoded) + 1)
	}
	if files > maxJSONStoreMissions || totalBytes > maxJSONStoreTotalMissionBytes {
		return errors.New("aggregate mission store exceeds persistence limit")
	}
	return nil
}

func validateInMemoryJSONStoreMissionQuotas(existing map[string]Mission, replacementID string, replacement Mission) error {
	files := 1
	var totalBytes int64
	for missionID, mission := range existing {
		if missionID == replacementID {
			continue
		}
		encoded, err := json.MarshalIndent(mission, "", "  ")
		if err != nil {
			return err
		}
		files++
		totalBytes += int64(len(encoded) + 1)
		if files > maxJSONStoreMissions || totalBytes > maxJSONStoreTotalMissionBytes {
			return errors.New("aggregate mission store exceeds persistence limit")
		}
	}
	encoded, err := json.MarshalIndent(replacement, "", "  ")
	if err != nil {
		return err
	}
	totalBytes += int64(len(encoded) + 1)
	if files > maxJSONStoreMissions || totalBytes > maxJSONStoreTotalMissionBytes {
		return errors.New("aggregate mission store exceeds persistence limit")
	}
	return nil
}

type organizationScopedStore struct {
	store          Store
	organizationID string
}

func (s organizationScopedStore) GetMission(id string) (Mission, error) {
	mission, err := s.store.GetMission(id)
	if err != nil {
		return Mission{}, err
	}
	if mission.OrganizationID != s.organizationID {
		return Mission{}, os.ErrNotExist
	}
	return mission, nil
}

func (s organizationScopedStore) ListMissions() ([]Mission, error) {
	missions, err := s.store.ListMissions()
	if err != nil {
		return nil, err
	}
	filtered := make([]Mission, 0, len(missions))
	for _, mission := range missions {
		if mission.OrganizationID == s.organizationID {
			filtered = append(filtered, mission)
		}
	}
	return filtered, nil
}

func (s organizationScopedStore) CreateMission(mission Mission) error {
	if mission.OrganizationID != s.organizationID {
		return os.ErrPermission
	}
	return s.store.CreateMission(mission)
}

func (s organizationScopedStore) PutMission(mission Mission) error {
	if _, err := s.GetMission(mission.ID); err != nil {
		return err
	}
	if mission.OrganizationID != s.organizationID {
		return os.ErrPermission
	}
	return s.store.PutMission(mission)
}

func (s organizationScopedStore) PutMissionIfVersion(mission Mission, expectedVersion int64) error {
	if _, err := s.GetMission(mission.ID); err != nil {
		return err
	}
	if mission.OrganizationID != s.organizationID {
		return os.ErrPermission
	}
	return s.store.PutMissionIfVersion(mission, expectedVersion)
}

func (s organizationScopedStore) AppendEvent(event Event) error {
	mission, err := s.GetMission(event.MissionID)
	if err != nil {
		return err
	}
	if event.OrganizationID != s.organizationID || mission.OrganizationID != event.OrganizationID {
		return os.ErrPermission
	}
	return s.store.AppendEvent(event)
}

func (s organizationScopedStore) ListEvents(missionID string) ([]Event, error) {
	if _, err := s.GetMission(missionID); err != nil {
		return nil, err
	}
	return s.store.ListEvents(missionID)
}

func (s *JSONStore) ListEvents(missionID string) ([]Event, error) {
	if !validSnapshotID(missionID) {
		return nil, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.persistent {
		return cloneEvents(s.events[missionID]), nil
	}
	var events []Event
	path := filepath.Join(s.root, "events", missionID+".json")
	if err := withMissionStoreLock(s.root, func() error {
		var mission Mission
		if err := readMissionJSON(filepath.Join(s.root, "missions", missionID+".json"), &mission); err != nil {
			return err
		}
		if mission.ID != missionID {
			return errors.New("stored mission id does not match its filename")
		}
		s.missions[missionID] = cloneMission(mission)
		if err := readJSONBounded(path, &events, maxJSONStoreEventFileBytes); errors.Is(err, os.ErrNotExist) {
			events = []Event{}
			return nil
		} else if err != nil {
			return err
		}
		if err := validateJSONStoreEvents(events); err != nil {
			return err
		}
		for index := range events {
			events[index] = cloneEvent(events[index])
			events[index].Payload = RedactValue(events[index].Payload)
		}
		return writeJSONAtomicBounded(path, events, maxJSONStoreEventFileBytes, "JSONStore event file")
	}); err != nil {
		return nil, err
	}
	s.events[missionID] = cloneEvents(events)
	return cloneEvents(events), nil
}

func writeJSONAtomic(path string, value any) error {
	return writeJSONAtomicBounded(path, value, 0, "JSON file")
}

func writeJSONAtomicBounded(path string, value any, maxBytes int, label string) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if maxBytes > 0 && len(data) > maxBytes {
		return fmt.Errorf("%s exceeds persistence limit", label)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".agent-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func readJSONBounded(path string, target any, maxBytes int) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err != nil {
		return err
	}
	if len(data) > maxBytes {
		return errors.New("JSON file exceeds persistence limit")
	}
	return json.Unmarshal(data, target)
}

func validateJSONStoreEventPayload(payload any) error {
	budget := missionJSONBudget{}
	if err := validateBoundedJSONValue(reflect.ValueOf(payload), 0, &budget); err != nil {
		return fmt.Errorf("event payload exceeds bounded JSON limits: %w", err)
	}
	if budget.bytes > maxJSONStoreEventPayloadBytes {
		return errors.New("event payload exceeds persistence limit")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}
	if len(encoded) > maxJSONStoreEventPayloadBytes {
		return errors.New("event payload exceeds persistence limit")
	}
	return nil
}

func validateJSONStoreEvents(events []Event) error {
	if len(events) > maxJSONStoreEventCount {
		return errors.New("event count exceeds persistence limit")
	}
	for _, event := range events {
		if err := validateJSONStoreEventPayload(event.Payload); err != nil {
			return err
		}
	}
	return nil
}

func validateJSONStoreAggregateEventQuotas(eventsDir, replacementMissionID string, replacement []Event) error {
	entries, err := readBoundedStoreDirEntries(eventsDir, maxJSONStoreEventFiles)
	if err != nil {
		return err
	}
	var files, totalEvents int
	var totalBytes int64
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		missionID := strings.TrimSuffix(entry.Name(), ".json")
		if replacementMissionID != "" && missionID == replacementMissionID {
			continue
		}
		files++
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() < 0 || info.Size() > maxJSONStoreEventFileBytes {
			return errors.New("event file exceeds persistence limit")
		}
		totalBytes += info.Size()
		var stored []Event
		if err := readJSONBounded(filepath.Join(eventsDir, entry.Name()), &stored, maxJSONStoreEventFileBytes); err != nil {
			return err
		}
		if err := validateJSONStoreEvents(stored); err != nil {
			return err
		}
		totalEvents += len(stored)
	}
	if replacementMissionID != "" && len(replacement) > 0 {
		files++
		encoded, err := json.MarshalIndent(replacement, "", "  ")
		if err != nil {
			return err
		}
		totalBytes += int64(len(encoded) + 1)
		totalEvents += len(replacement)
	}
	return checkJSONStoreEventQuotaTotals(files, totalEvents, totalBytes)
}

func validateInMemoryJSONStoreEventQuotas(existing map[string][]Event, replacementMissionID string, replacement []Event) error {
	var files, totalEvents int
	var totalBytes int64
	for missionID, events := range existing {
		if missionID == replacementMissionID || len(events) == 0 {
			continue
		}
		files++
		encoded, err := json.MarshalIndent(events, "", "  ")
		if err != nil {
			return err
		}
		totalEvents += len(events)
		totalBytes += int64(len(encoded) + 1)
	}
	if len(replacement) > 0 {
		files++
		encoded, err := json.MarshalIndent(replacement, "", "  ")
		if err != nil {
			return err
		}
		totalEvents += len(replacement)
		totalBytes += int64(len(encoded) + 1)
	}
	return checkJSONStoreEventQuotaTotals(files, totalEvents, totalBytes)
}

func checkJSONStoreEventQuotaTotals(files, events int, bytes int64) error {
	if files > maxJSONStoreEventFiles {
		return errors.New("event file count exceeds persistence limit")
	}
	if events > maxJSONStoreTotalEvents {
		return errors.New("aggregate event count exceeds persistence limit")
	}
	if bytes > maxJSONStoreTotalEventBytes {
		return errors.New("aggregate event bytes exceed persistence limit")
	}
	return nil
}

func cloneMission(mission Mission) Mission {
	copy := mission
	copy.Capabilities = append([]string(nil), mission.Capabilities...)
	copy.Plan = append([]Step(nil), mission.Plan...)
	for i := range copy.Plan {
		copy.Plan[i].Input = cloneMap(mission.Plan[i].Input)
		copy.Plan[i].Result = cloneJSONValue(mission.Plan[i].Result)
	}
	copy.Approvals = append([]Approval(nil), mission.Approvals...)
	for i := range copy.Approvals {
		if mission.Approvals[i].ExpiresAt != nil {
			expiresAt := *mission.Approvals[i].ExpiresAt
			copy.Approvals[i].ExpiresAt = &expiresAt
		}
	}
	copy.Artifacts = append([]ArtifactManifest(nil), mission.Artifacts...)
	if mission.CompletedAt != nil {
		completedAt := *mission.CompletedAt
		copy.CompletedAt = &completedAt
	}
	return copy
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = cloneJSONValue(value)
	}
	return output
}

func cloneEvent(event Event) Event {
	copy := event
	copy.Payload = cloneJSONValue(event.Payload)
	return copy
}

func cloneEvents(events []Event) []Event {
	if events == nil {
		return nil
	}
	copy := make([]Event, len(events))
	for index, event := range events {
		copy[index] = cloneEvent(event)
	}
	return copy
}

func cloneJSONValue(value any) any {
	if value == nil {
		return nil
	}
	return cloneJSONReflect(reflect.ValueOf(value)).Interface()
}

func cloneJSONReflect(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := cloneJSONReflect(value.Elem())
		result := reflect.New(value.Type()).Elem()
		result.Set(cloned)
		return result
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeMapWithSize(value.Type(), value.Len())
		iter := value.MapRange()
		for iter.Next() {
			result.SetMapIndex(cloneJSONReflect(iter.Key()), cloneJSONReflect(iter.Value()))
		}
		return result
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for index := 0; index < value.Len(); index++ {
			result.Index(index).Set(cloneJSONReflect(value.Index(index)))
		}
		return result
	case reflect.Array:
		result := reflect.New(value.Type()).Elem()
		for index := 0; index < value.Len(); index++ {
			result.Index(index).Set(cloneJSONReflect(value.Index(index)))
		}
		return result
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.New(value.Type().Elem())
		result.Elem().Set(cloneJSONReflect(value.Elem()))
		return result
	default:
		return value
	}
}

func samePersistedMission(left, right Mission) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func sortMissions(missions []Mission) {
	sort.Slice(missions, func(i, j int) bool { return missions[i].UpdatedAt.Before(missions[j].UpdatedAt) })
}
