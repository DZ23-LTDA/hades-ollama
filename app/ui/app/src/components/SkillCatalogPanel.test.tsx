import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

import { SkillCatalogPanel, type SkillCatalogPanelProps } from "./SkillCatalogPanel";
import type { AgentSkill } from "@/lib/agenticClient";
import { emptySkillContext, type AgentSkillSelection } from "@/lib/skillCatalog";

function skill(overrides: Partial<AgentSkill> = {}): AgentSkill {
  return {
    id: "skill-runtime",
    version: "1.0.0",
    description: "skill assinada",
    scopes: ["workspace:read"],
    tools: [],
    trusted: true,
    enabled: true,
    category: "dados",
    token_budget: 40,
    ...overrides,
  } as AgentSkill;
}

const selection: AgentSkillSelection = {
  selected: [
    { id: "skill-runtime", version: "1.0.0", category: "dados", token_budget: 40, activation: { skill_id: "skill-runtime" } },
  ],
  skipped: [{ id: "skill-grande", reason: "orçamento insuficiente (10 tokens restantes, custo 40)" }],
  used_tokens: 40,
  token_budget: 100,
  budget_defined: true,
};

async function render(props: Partial<SkillCatalogPanelProps> = {}): Promise<ReactTestRenderer> {
  let renderer: ReactTestRenderer | null = null;
  await act(async () => {
    renderer = create(
      <SkillCatalogPanel
        skills={[]}
        activations={[]}
        selection={null}
        context={emptySkillContext}
        onContextChange={() => {}}
        onResolve={() => {}}
        onActivate={() => {}}
        onDeactivate={() => {}}
        {...props}
      />,
    );
  });
  return renderer as unknown as ReactTestRenderer;
}

function textOf(renderer: ReactTestRenderer): string {
  return JSON.stringify(renderer.toJSON());
}

describe("SkillCatalogPanel", () => {
  let renderer: ReactTestRenderer | null = null;

  afterEach(() => {
    renderer?.unmount();
    renderer = null;
  });

  it("mostra categoria, custo e escopos reais da skill", async () => {
    renderer = await render({ skills: [skill()] });
    const text = textOf(renderer);
    expect(text).toContain("dados");
    expect(text).toContain("40 tokens");
    expect(text).toContain("workspace:read");
  });

  it("bloqueia ativação de skill não confiável e explica o porquê", async () => {
    renderer = await render({ skills: [skill({ trusted: false })] });
    const text = textOf(renderer);
    expect(text).toContain("não confiável");
    expect(text).toContain("recusa ativar skill não confiável");

    const button = renderer.root.findAllByType("button").find((node) => JSON.stringify(node.props.children).includes("Ativar"));
    expect(button?.props.disabled).toBe(true);
  });

  it("mostra a seleção do servidor com orçamento e o motivo das exclusões", async () => {
    renderer = await render({ skills: [skill()], selection });
    const text = textOf(renderer);
    expect(text).toContain("40 de 100 tokens usados");
    expect(text).toContain("Seleção do servidor");
    expect(text).toContain("orçamento insuficiente");
    // A skill excluída aparece como excluída — nunca como pronta.
    expect(text).not.toContain("Pronto para usar");
  });

  it("reflete o estado de ativação do contexto e alterna a ação", async () => {
    renderer = await render({
      skills: [skill()],
      activations: [{ skill_id: "skill-runtime", project_id: "proj-1" }],
      context: { ...emptySkillContext, projectId: "proj-1" },
    });
    expect(textOf(renderer)).toContain("Desativar");

    renderer.unmount();
    renderer = await render({
      skills: [skill()],
      activations: [{ skill_id: "skill-runtime", project_id: "proj-1" }],
      context: { ...emptySkillContext, projectId: "proj-2" },
    });
    expect(textOf(renderer)).toContain("Ativar");
  });

  it("recusa orçamento inválido antes de chamar o servidor", async () => {
    const onResolve = vi.fn();
    renderer = await render({ skills: [skill()], context: { ...emptySkillContext, tokenBudget: "abc" }, onResolve });
    const text = textOf(renderer);
    expect(text).toContain("Use um número inteiro de tokens");
    const resolve = renderer.root.findAllByType("button").find((node) => JSON.stringify(node.props.children).includes("Resolver catálogo"));
    expect(resolve?.props.disabled).toBe(true);
    expect(onResolve).not.toHaveBeenCalled();
  });

  it("trata lista vazia e erro de leitura com honestidade", async () => {
    renderer = await render();
    expect(textOf(renderer)).toContain("Nenhuma skill registrada");
    renderer.unmount();
    renderer = await render({ error: "organização não autenticada" });
    expect(textOf(renderer)).toContain("organização não autenticada");
  });
});
