package agent

import (
	"sort"
	"strings"
	"time"
)

// Notification is a tenant-scoped, durable projection of a mission event.
// The source of truth remains the persisted mission event; this projection
// avoids inventing notifications when no event was recorded.
type Notification struct {
	ID        string    `json:"id"`
	MissionID string    `json:"mission_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ListNotifications returns recent mission events for the runtime's active
// organization. Because events are persisted with missions, the result also
// survives process restarts without a second, divergent notification store.
func (r *Runtime) ListNotifications(limit int) ([]Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	missions, err := r.ListMissions()
	if err != nil {
		return nil, err
	}
	result := make([]Notification, 0)
	for _, mission := range missions {
		events, eventErr := r.Events(mission.ID)
		if eventErr != nil {
			return nil, eventErr
		}
		for _, event := range events {
			result = append(result, Notification{
				ID:        event.ID,
				MissionID: event.MissionID,
				Type:      event.Type,
				Title:     notificationTitle(event.Type),
				Body:      mission.Objective,
				CreatedAt: event.CreatedAt,
			})
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func notificationTitle(eventType string) string {
	switch {
	case strings.HasSuffix(eventType, ".succeeded"), strings.HasSuffix(eventType, ".completed"):
		return "Missão concluída"
	case strings.HasSuffix(eventType, ".failed"), strings.HasSuffix(eventType, ".error"):
		return "Missão com erro"
	case strings.HasSuffix(eventType, ".approval_required"):
		return "Aprovação necessária"
	case strings.HasSuffix(eventType, ".cancelled"):
		return "Missão cancelada"
	default:
		return "Atualização da missão"
	}
}
