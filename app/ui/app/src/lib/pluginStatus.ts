/**
 * Status canônico de plugin (V2.1 §11.3) — lógica de apresentação.
 *
 * A regra que este módulo existe para garantir: "pronto para usar" NUNCA é
 * inferido do catálogo. Só é exibido quando o servidor devolve `ready_to_use`,
 * que por sua vez só é verdadeiro quando houve execução real bem-sucedida.
 */

export type AgentPluginStatus = {
  plugin_id: string;
  version: string;
  kind: string;
  state: string;
  enabled: boolean;
  trusted: boolean;
  content_verified: boolean;
  capabilities: string[];
  requested_scopes?: string[];
  reasons?: string[];
  ready_to_use: boolean;
  last_execution?: {
    step_id: string;
    executed_at: string;
    succeeded: boolean;
    sandbox_mode: string;
  } | null;
};

const stateLabels: Record<string, string> = {
  INSTALLED: "Instalado",
  CONNECTED: "Conectado",
  CONFIG_REQUIRED: "Configuração pendente",
  UNAUTHORIZED: "Não autorizado",
  UNHEALTHY: "Conteúdo divergente",
  UNSUPPORTED: "Não executável",
  TESTED_LIVE: "Execução comprovada",
};

const stateDescriptions: Record<string, string> = {
  INSTALLED: "Instalado e executável, mas ainda sem execução comprovada.",
  CONNECTED: "Vínculo verificado com o serviço externo.",
  CONFIG_REQUIRED: "Falta configuração: habilitação ou escopos concedidos.",
  UNAUTHORIZED: "O servidor não considera este plugin confiável.",
  UNHEALTHY: "O conteúdo no disco difere do que foi verificado.",
  UNSUPPORTED: "Sem runtime/entrypoint declarado ou conteúdo não verificado.",
  TESTED_LIVE: "Já executou com sucesso; pode ser usado.",
};

const stateTones: Record<string, string> = {
  TESTED_LIVE: "green",
  CONNECTED: "green",
  INSTALLED: "neutral",
  CONFIG_REQUIRED: "amber",
  UNAUTHORIZED: "red",
  UNHEALTHY: "red",
  UNSUPPORTED: "amber",
};

export function pluginStateLabel(state: string): string {
  const key = String(state || "").toUpperCase();
  return stateLabels[key] ?? "Estado desconhecido";
}

export function pluginStateDescription(state: string): string {
  const key = String(state || "").toUpperCase();
  return (
    stateDescriptions[key] ??
    "O servidor não reconhece este estado; trate como indisponível até haver verificação."
  );
}

export function pluginStateTone(state: string): string {
  const key = String(state || "").toUpperCase();
  return stateTones[key] ?? "neutral";
}

/** Só o servidor decide que algo está pronto — e só com execução comprovada. */
export function isReadyToUse(status: AgentPluginStatus | null | undefined): boolean {
  if (!status) return false;
  return status.ready_to_use === true && String(status.state).toUpperCase() === "TESTED_LIVE";
}

export function pluginExecutionSummary(status: AgentPluginStatus): string {
  const execution = status.last_execution;
  if (!execution) return "Nenhuma execução registrada.";
  const when = execution.executed_at ? new Date(execution.executed_at).toLocaleString() : "";
  if (execution.succeeded) {
    return `Última execução bem-sucedida em ${when} (${execution.sandbox_mode || "sandbox"}).`;
  }
  return `A última tentativa foi recusada em ${when}; o plugin continua não comprovado.`;
}
