// Human, pt-BR labels for the raw backend enums (mission states, event types,
// step kinds). The UI must never show the raw UPPER_SNAKE / dotted identifiers
// to the user. Unknown values fall back to a humanized form instead of leaking
// the raw token.

const MISSION_STATES: Record<string, string> = {
  PENDING: "Pendente",
  QUEUED: "Na fila",
  PLANNING: "Planejando",
  RUNNING: "Em execução",
  OBSERVING: "Observando",
  RECOVERING: "Recuperando",
  AWAITING_APPROVAL: "Aguardando aprovação",
  PAUSED: "Pausada",
  BLOCKED: "Bloqueada",
  BLOCKED_EXTERNAL: "Bloqueada (dependência externa)",
  COMPLETED: "Concluída",
  FAILED: "Falhou",
  CANCELLED: "Cancelada",
  CANCELED: "Cancelada",
  NOT_EXECUTED: "Não executada",
  PASS: "OK",
};

const EVENT_TYPES: Record<string, string> = {
  "mission.created": "Missão criada",
  "mission.planned": "Plano gerado",
  "mission.run": "Missão iniciada",
  "mission.running": "Em execução",
  "mission.completed": "Missão concluída",
  "mission.failed": "Missão falhou",
  "mission.cancelled": "Missão cancelada",
  "mission.repair_attempt": "Tentando corrigir",
  "mission.repair_succeeded": "Correção aplicada",
  "mission.repair_failed_test": "Teste ainda falhando",
  "mission.repair_error": "Erro ao corrigir",
  "mission.repair_exhausted": "Tentativas de correção esgotadas",
  "mission.queue_failed": "Falha ao enfileirar",
  "step.started": "Passo iniciado",
  "step.succeeded": "Passo concluído",
  "step.failed": "Passo falhou",
  "step.awaiting_approval": "Passo aguardando aprovação",
  "step.retry_scheduled": "Nova tentativa agendada",
  "browser.operator": "Navegador",
  "browser.frame": "Quadro do navegador",
  "browser.approval": "Aprovação do navegador",
  "git.merge.succeeded": "Merge concluído",
  "git.merge.rejected": "Merge recusado",
};

const STEP_KINDS: Record<string, string> = {
  "workspace.read": "Ler arquivo",
  "workspace.write": "Escrever arquivo",
  "workspace.patch": "Alterar arquivo",
  "workspace.list": "Listar arquivos",
  "browser.operator": "Operar navegador",
  "project.test.run": "Rodar testes",
};

// humanize turns an UNKNOWN raw token ("some.raw_state") into "Some raw state".
function humanize(raw: string): string {
  const text = raw.replace(/[._]+/g, " ").trim();
  if (!text) return raw;
  return text.charAt(0).toUpperCase() + text.slice(1).toLowerCase();
}

export function missionStateLabel(state: string | undefined | null): string {
  if (!state) return "";
  return MISSION_STATES[state] ?? humanize(state);
}

export function eventTypeLabel(type: string | undefined | null): string {
  if (!type) return "";
  return EVENT_TYPES[type] ?? humanize(type);
}

export function stepKindLabel(kind: string | undefined | null): string {
  if (!kind) return "";
  return STEP_KINDS[kind] ?? humanize(kind);
}
