/**
 * Catálogo de skills na interface (V2.1 §11.3) — lógica de apresentação.
 *
 * Regra central: a tela mostra o que o SERVIDOR selecionou para o contexto, com
 * o orçamento consumido e o motivo de cada exclusão. Nada é inferido do catálogo
 * e uma skill não confiável não pode ser ativada.
 */

export type AgentSkillActivation = {
  skill_id: string;
  organization_id?: string;
  project_id?: string;
  agent_id?: string;
  model?: string;
  activated_at?: string;
};

export type AgentActiveSkill = {
  id: string;
  version: string;
  description?: string;
  category?: string;
  license?: string;
  homepage?: string;
  scopes?: string[];
  tools?: string[];
  token_budget?: number;
  activation: AgentSkillActivation;
};

export type AgentSkillSkip = {
  id: string;
  reason: string;
};

export type AgentSkillSelection = {
  selected: AgentActiveSkill[];
  skipped?: AgentSkillSkip[];
  used_tokens: number;
  token_budget: number;
  budget_defined: boolean;
};

export type SkillContextInput = {
  projectId: string;
  agentId: string;
  model: string;
  tokenBudget: string;
};

export const emptySkillContext: SkillContextInput = {
  projectId: "",
  agentId: "",
  model: "",
  tokenBudget: "",
};

/** Resumo honesto do orçamento: sem orçamento declarado, nada é "usado". */
export function skillUsageLabel(selection: AgentSkillSelection | null): string {
  if (!selection) return "Nenhuma seleção resolvida ainda.";
  if (!selection.budget_defined) {
    return `${selection.selected.length} skill(s) selecionada(s) sem orçamento declarado.`;
  }
  return `${selection.used_tokens} de ${selection.token_budget} tokens usados · ${selection.selected.length} skill(s)`;
}

export function skillSkipLabel(skip: AgentSkillSkip): string {
  return `${skip.id}: ${skip.reason}`;
}

/**
 * Uma ativação cobre o contexto quando todos os campos declarados batem (campo
 * vazio é curinga) — mesma semântica do servidor, para o botão refletir a
 * verdade e não uma suposição da interface.
 */
export function activationMatches(activation: AgentSkillActivation, context: SkillContextInput): boolean {
  const project = (activation.project_id ?? "").trim();
  if (project && project !== context.projectId.trim()) return false;
  const agent = (activation.agent_id ?? "").trim();
  if (agent && agent !== context.agentId.trim()) return false;
  const model = (activation.model ?? "").trim();
  if (model && model.toLowerCase() !== context.model.trim().toLowerCase()) return false;
  return true;
}

export function activatedSkillIDs(activations: AgentSkillActivation[], context: SkillContextInput): string[] {
  const ids = new Set<string>();
  for (const activation of activations) {
    if (activationMatches(activation, context)) ids.add(activation.skill_id);
  }
  return [...ids].sort();
}

/** Orçamento só é enviado quando o usuário digitou um inteiro não negativo. */
export function parseTokenBudget(raw: string): { value: number | null; error: string | null } {
  const trimmed = raw.trim();
  if (trimmed === "") return { value: null, error: null };
  if (!/^\d+$/.test(trimmed)) return { value: null, error: "Use um número inteiro de tokens (ou deixe vazio)." };
  return { value: Number(trimmed), error: null };
}
