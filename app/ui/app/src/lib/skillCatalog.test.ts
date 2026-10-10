import { describe, expect, it } from "vitest";

import {
  activatedSkillIDs,
  activationMatches,
  emptySkillContext,
  parseTokenBudget,
  skillSkipLabel,
  skillUsageLabel,
  type AgentSkillSelection,
} from "./skillCatalog";

function selection(overrides: Partial<AgentSkillSelection> = {}): AgentSkillSelection {
  return {
    selected: [
      {
        id: "skill-a",
        version: "1.0.0",
        category: "dados",
        token_budget: 40,
        activation: { skill_id: "skill-a", project_id: "proj-1" },
      },
    ],
    skipped: [],
    used_tokens: 40,
    token_budget: 100,
    budget_defined: true,
    ...overrides,
  };
}

describe("catálogo de skills na interface", () => {
  it("resume o orçamento sem inventar consumo quando não há orçamento", () => {
    expect(skillUsageLabel(null)).toContain("Nenhuma seleção");
    expect(skillUsageLabel(selection())).toBe("40 de 100 tokens usados · 1 skill(s)");
    expect(
      skillUsageLabel(selection({ budget_defined: false, used_tokens: 0, token_budget: 0 })),
    ).toBe("1 skill(s) selecionada(s) sem orçamento declarado.");
  });

  it("mostra o motivo da exclusão em vez de esconder a skill", () => {
    expect(skillSkipLabel({ id: "skill-b", reason: "orçamento insuficiente (10 tokens restantes, custo 40)" })).toBe(
      "skill-b: orçamento insuficiente (10 tokens restantes, custo 40)",
    );
  });

  it("aplica a mesma semântica de curinga do servidor", () => {
    const global = { skill_id: "skill-global" };
    expect(activationMatches(global, emptySkillContext)).toBe(true);
    expect(activationMatches(global, { projectId: "proj-1", agentId: "", model: "", tokenBudget: "" })).toBe(true);

    const escopo = { skill_id: "skill-proj", project_id: "proj-1" };
    expect(activationMatches(escopo, { projectId: "proj-1", agentId: "", model: "", tokenBudget: "" })).toBe(true);
    expect(activationMatches(escopo, { projectId: "proj-2", agentId: "", model: "", tokenBudget: "" })).toBe(false);

    const porModelo = { skill_id: "skill-modelo", model: "Qwen2.5:7b" };
    expect(activationMatches(porModelo, { projectId: "", agentId: "", model: "qwen2.5:7b", tokenBudget: "" })).toBe(true);
    expect(activationMatches(porModelo, { projectId: "", agentId: "", model: "llama3.1:8b", tokenBudget: "" })).toBe(false);

    const porAgente = { skill_id: "skill-agente", agent_id: "revisor" };
    expect(activationMatches(porAgente, { projectId: "", agentId: "revisor", model: "", tokenBudget: "" })).toBe(true);
    expect(activationMatches(porAgente, { projectId: "", agentId: "qa", model: "", tokenBudget: "" })).toBe(false);
  });

  it("lista os ids ativados no contexto, de forma estável", () => {
    const activations = [
      { skill_id: "skill-b", project_id: "proj-1" },
      { skill_id: "skill-a" },
      { skill_id: "skill-c", project_id: "proj-2" },
    ];
    expect(activatedSkillIDs(activations, { projectId: "proj-1", agentId: "", model: "", tokenBudget: "" })).toEqual([
      "skill-a",
      "skill-b",
    ]);
  });

  it("valida o orçamento digitado antes de chamar o servidor", () => {
    expect(parseTokenBudget("")).toEqual({ value: null, error: null });
    expect(parseTokenBudget(" 250 ")).toEqual({ value: 250, error: null });
    expect(parseTokenBudget("0")).toEqual({ value: 0, error: null });
    expect(parseTokenBudget("abc").error).not.toBeNull();
    expect(parseTokenBudget("-10").error).not.toBeNull();
  });
});
