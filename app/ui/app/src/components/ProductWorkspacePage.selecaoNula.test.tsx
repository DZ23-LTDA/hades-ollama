import { describe, expect, it } from "vitest";

import { skillActivationsFromSelection } from "./ProductWorkspacePage";
import type { AgentSkillSelection } from "@/lib/skillCatalog";

// Regressao do defeito observado em 2026-10-21 na pagina /skills do produto INSTALADO:
// a app abria, a pagina caia no ecra de erro com
//   TypeError: Cannot read properties of null (reading 'map')
// porque o servidor devolve `selected: null` quando o contexto nao tem skills associadas.
function selecao(parcial: Partial<AgentSkillSelection>): AgentSkillSelection {
  return {
    selected: [],
    skipped: [],
    used_tokens: 0,
    token_budget: 0,
    budget_defined: false,
    ...parcial,
  } as AgentSkillSelection;
}

describe("skillActivationsFromSelection", () => {
  it("nao rebenta quando o servidor devolve selected e skipped a null", () => {
    const vazio = selecao({ selected: null, skipped: null });
    expect(() => skillActivationsFromSelection(vazio)).not.toThrow();
    expect(skillActivationsFromSelection(vazio)).toEqual([]);
  });

  it("devolve lista vazia quando a selecao e' null", () => {
    expect(skillActivationsFromSelection(null)).toEqual([]);
  });

  it("junta selecionadas e ignoradas, sem duplicar ids", () => {
    const s = selecao({
      selected: [{ id: "a" }, { id: "b" }] as AgentSkillSelection["selected"],
      skipped: [{ id: "b" }, { id: "c" }] as AgentSkillSelection["skipped"],
    });
    const ids = skillActivationsFromSelection(s).map((a) => a.skill_id);
    expect(ids).toEqual(["a", "b", "c"]);
  });

  it("sem skipped continua a devolver as selecionadas", () => {
    const s = selecao({ selected: [{ id: "x" }] as AgentSkillSelection["selected"], skipped: undefined });
    expect(skillActivationsFromSelection(s).map((a) => a.skill_id)).toEqual(["x"]);
  });
});
