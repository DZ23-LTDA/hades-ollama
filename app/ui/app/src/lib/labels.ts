// Human labels for the raw backend enums (mission states, event types, step
// kinds). The UI must never show the raw UPPER_SNAKE / dotted identifiers to
// the user. Unknown values fall back to a humanized form instead of leaking
// the raw token.
//
// As tabelas guardam CHAVES de i18n, não texto final: `t()` é chamado dentro
// das funções (e não no carregamento do módulo), para que a troca de idioma em
// tempo de execução repinte os rótulos.

import { t } from "@/i18n";

const MISSION_STATES: Record<string, string> = {
  PENDING: "labels.missionState.pending",
  QUEUED: "labels.missionState.queued",
  PLANNING: "labels.missionState.planning",
  RUNNING: "labels.missionState.running",
  OBSERVING: "labels.missionState.observing",
  RECOVERING: "labels.missionState.recovering",
  AWAITING_APPROVAL: "labels.missionState.awaitingApproval",
  PAUSED: "labels.missionState.paused",
  BLOCKED: "labels.missionState.blocked",
  BLOCKED_EXTERNAL: "labels.missionState.blockedExternal",
  COMPLETED: "labels.missionState.completed",
  FAILED: "labels.missionState.failed",
  CANCELLED: "labels.missionState.cancelled",
  CANCELED: "labels.missionState.cancelled",
  NOT_EXECUTED: "labels.missionState.notExecuted",
  PASS: "labels.missionState.pass",
};

const EVENT_TYPES: Record<string, string> = {
  "mission.created": "labels.eventType.missionCreated",
  "mission.planned": "labels.eventType.missionPlanned",
  "mission.run": "labels.eventType.missionRun",
  "mission.running": "labels.eventType.missionRunning",
  "mission.completed": "labels.eventType.missionCompleted",
  "mission.failed": "labels.eventType.missionFailed",
  "mission.cancelled": "labels.eventType.missionCancelled",
  "mission.repair_attempt": "labels.eventType.missionRepairAttempt",
  "mission.repair_succeeded": "labels.eventType.missionRepairSucceeded",
  "mission.repair_failed_test": "labels.eventType.missionRepairFailedTest",
  "mission.repair_error": "labels.eventType.missionRepairError",
  "mission.repair_exhausted": "labels.eventType.missionRepairExhausted",
  "mission.queue_failed": "labels.eventType.missionQueueFailed",
  "step.started": "labels.eventType.stepStarted",
  "step.succeeded": "labels.eventType.stepSucceeded",
  "step.failed": "labels.eventType.stepFailed",
  "step.awaiting_approval": "labels.eventType.stepAwaitingApproval",
  "step.retry_scheduled": "labels.eventType.stepRetryScheduled",
  "browser.operator": "labels.eventType.browserOperator",
  "browser.frame": "labels.eventType.browserFrame",
  "browser.approval": "labels.eventType.browserApproval",
  "git.merge.succeeded": "labels.eventType.gitMergeSucceeded",
  "git.merge.rejected": "labels.eventType.gitMergeRejected",
};

const STEP_KINDS: Record<string, string> = {
  "workspace.read": "labels.stepKind.workspaceRead",
  "workspace.write": "labels.stepKind.workspaceWrite",
  "workspace.patch": "labels.stepKind.workspacePatch",
  "workspace.list": "labels.stepKind.workspaceList",
  "browser.operator": "labels.stepKind.browserOperator",
  "project.test.run": "labels.stepKind.projectTestRun",
};

// humanize turns an UNKNOWN raw token ("some.raw_state") into "Some raw state".
function humanize(raw: string): string {
  const text = raw.replace(/[._]+/g, " ").trim();
  if (!text) return raw;
  return text.charAt(0).toUpperCase() + text.slice(1).toLowerCase();
}

function lookup(table: Record<string, string>, raw: string): string {
  const key = table[raw];
  return typeof key === "string" ? t(key) : humanize(raw);
}

export function missionStateLabel(state: string | undefined | null): string {
  if (!state) return "";
  return lookup(MISSION_STATES, state);
}

export function eventTypeLabel(type: string | undefined | null): string {
  if (!type) return "";
  return lookup(EVENT_TYPES, type);
}

export function stepKindLabel(kind: string | undefined | null): string {
  if (!kind) return "";
  return lookup(STEP_KINDS, kind);
}
