package agent

import (
	"fmt"
	"sync/atomic"
)

type RuntimeMetrics struct {
	missionsCreated         atomic.Int64
	missionsCompleted       atomic.Int64
	missionsFailed          atomic.Int64
	stepsStarted            atomic.Int64
	stepsSucceeded          atomic.Int64
	stepsFailed             atomic.Int64
	retries                 atomic.Int64
	approvals               atomic.Int64
	toolCalls               atomic.Int64
	eventPersistFailures    atomic.Int64
	redisQueueFailures      atomic.Int64
	redisQueueDroppedErrors atomic.Int64
	pushOutboxFailures      atomic.Int64
	pushDeliveryFailures    atomic.Int64
}

type MetricsSnapshot struct {
	MissionsCreated         int64 `json:"missions_created"`
	MissionsCompleted       int64 `json:"missions_completed"`
	MissionsFailed          int64 `json:"missions_failed"`
	StepsStarted            int64 `json:"steps_started"`
	StepsSucceeded          int64 `json:"steps_succeeded"`
	StepsFailed             int64 `json:"steps_failed"`
	Retries                 int64 `json:"retries"`
	Approvals               int64 `json:"approvals"`
	ToolCalls               int64 `json:"tool_calls"`
	EventPersistFailures    int64 `json:"event_persist_failures"`
	RedisQueueFailures      int64 `json:"redis_queue_failures"`
	RedisQueueDroppedErrors int64 `json:"redis_queue_dropped_errors"`
	PushOutboxFailures      int64 `json:"push_outbox_failures"`
	PushDeliveryFailures    int64 `json:"push_delivery_failures"`
}

func (m *RuntimeMetrics) Snapshot() MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{}
	}
	return MetricsSnapshot{
		MissionsCreated: m.missionsCreated.Load(), MissionsCompleted: m.missionsCompleted.Load(), MissionsFailed: m.missionsFailed.Load(),
		StepsStarted: m.stepsStarted.Load(), StepsSucceeded: m.stepsSucceeded.Load(), StepsFailed: m.stepsFailed.Load(),
		Retries: m.retries.Load(), Approvals: m.approvals.Load(), ToolCalls: m.toolCalls.Load(),
		EventPersistFailures: m.eventPersistFailures.Load(), RedisQueueFailures: m.redisQueueFailures.Load(), RedisQueueDroppedErrors: m.redisQueueDroppedErrors.Load(),
		PushOutboxFailures: m.pushOutboxFailures.Load(), PushDeliveryFailures: m.pushDeliveryFailures.Load(),
	}
}

func (m MetricsSnapshot) Prometheus() string {
	return fmt.Sprintf("# TYPE ollama_agent_missions_created counter\nollama_agent_missions_created %d\n# TYPE ollama_agent_missions_completed counter\nollama_agent_missions_completed %d\n# TYPE ollama_agent_missions_failed counter\nollama_agent_missions_failed %d\n# TYPE ollama_agent_steps_started counter\nollama_agent_steps_started %d\n# TYPE ollama_agent_steps_succeeded counter\nollama_agent_steps_succeeded %d\n# TYPE ollama_agent_steps_failed counter\nollama_agent_steps_failed %d\n# TYPE ollama_agent_retries counter\nollama_agent_retries %d\n# TYPE ollama_agent_approvals counter\nollama_agent_approvals %d\n# TYPE ollama_agent_tool_calls counter\nollama_agent_tool_calls %d\n# TYPE ollama_agent_event_persist_failures counter\nollama_agent_event_persist_failures %d\n# TYPE ollama_agent_redis_queue_failures counter\nollama_agent_redis_queue_failures %d\n# TYPE ollama_agent_redis_queue_dropped_errors counter\nollama_agent_redis_queue_dropped_errors %d\n# TYPE ollama_agent_push_outbox_failures counter\nollama_agent_push_outbox_failures %d\n# TYPE ollama_agent_push_delivery_failures counter\nollama_agent_push_delivery_failures %d\n", m.MissionsCreated, m.MissionsCompleted, m.MissionsFailed, m.StepsStarted, m.StepsSucceeded, m.StepsFailed, m.Retries, m.Approvals, m.ToolCalls, m.EventPersistFailures, m.RedisQueueFailures, m.RedisQueueDroppedErrors, m.PushOutboxFailures, m.PushDeliveryFailures)
}
