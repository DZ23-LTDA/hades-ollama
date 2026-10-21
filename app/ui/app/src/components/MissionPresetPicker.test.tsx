import { act, create, type ReactTestInstance } from "react-test-renderer";
import { describe, expect, it, vi } from "vitest";

import MissionPresetPicker from "./MissionPresetPicker";
import {
  describeSchedule,
  parseMissionPresets,
  summarizePreset,
  type MissionPreset,
} from "@/lib/missionPresets";

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

function preset(over: Partial<MissionPreset> = {}): MissionPreset {
  return {
    id: "briefing-diario",
    name: "Briefing diário",
    description: "Resumo curto do dia",
    goal: "Produzir um briefing",
    permissions: ["workspace:read"],
    budget_seconds: 300,
    budget_tokens: 20000,
    completion_criteria: ["resumo entregue", "fontes citadas"],
    schedule: "daily",
    ...over,
  };
}

function render(
  props: Partial<React.ComponentProps<typeof MissionPresetPicker>> = {},
) {
  const onSelect = vi.fn();
  let renderer!: ReturnType<typeof create>;
  act(() => {
    renderer = create(
      <MissionPresetPicker
        presets={[preset()]}
        selected={null}
        onSelect={onSelect}
        {...props}
      />,
    );
  });
  const texto = textContent(renderer.root);
  return { renderer, texto, onSelect };
}

describe("MissionPresetPicker", () => {
  it("lista os presets e mostra o que cada um pede, custa e quando acaba", () => {
    const { texto } = render({
      presets: [
        preset({ id: "a", name: "Briefing diário" }),
        preset({
          id: "b",
          name: "Automação local",
          permissions: ["workspace:read", "terminal:allowlisted"],
          budget_seconds: 0,
          budget_tokens: 0,
          completion_criteria: ["script criado", "testes passam", "saída registada"],
          schedule: undefined,
        }),
      ],
    });
    expect(texto).toContain("Briefing diário");
    expect(texto).toContain("Automação local");
    // O que CUSTA: 300s => 5 min, 20000 tokens => 20k.
    expect(texto).toContain("5 min");
    expect(texto).toContain("20k tokens");
    // O que PEDE.
    expect(texto).toContain("1 permissão(ões) pedida(s)");
    expect(texto).toContain("2 permissão(ões) pedida(s)");
    // Quando ACABA -- o contrato da FASE 23.
    expect(texto).toContain("2 critério(s) de conclusão");
    expect(texto).toContain("3 critério(s) de conclusão");
    // E o agendamento.
    expect(texto).toContain("diário");
  });

  it("diz 'Sem orçamento declarado' em vez de mostrar zero", () => {
    // *Mostrar zero onde não há número é uma mentira de formatação.*
    const { texto } = render({
      presets: [preset({ budget_seconds: 0, budget_tokens: 0 })],
    });
    expect(texto).toContain("Sem orçamento declarado");
    expect(texto).not.toContain("0 min");
    expect(texto).not.toContain("0 tokens");
  });

  it("escolher um preset devolve o id e 'Sem preset' devolve null", () => {
    const { renderer, onSelect } = render();
    const radios = renderer.root.findAllByType("input");
    expect(radios.length).toBe(2);
    // radios[0] = "Sem preset", radios[1] = o primeiro preset.
    act(() => radios[1].props.onChange());
    expect(onSelect).toHaveBeenCalledWith("briefing-diario");
    act(() => radios[0].props.onChange());
    expect(onSelect).toHaveBeenCalledWith(null);
  });

  it("sem catálogo mostra aviso e NÃO aplica preset nenhum", () => {
    const { texto } = render({ presets: [], unavailable: true });
    expect(texto).toContain("Catálogo de presets indisponível");
    expect(texto).not.toContain("Briefing diário");
  });

  it("catálogo vazio diz que não há presets em vez de fingir um", () => {
    const { texto } = render({ presets: [] });
    expect(texto).toContain("não publicou nenhum preset");
  });
});

describe("parseMissionPresets", () => {
  it("descarta preset sem id", () => {
    const out = parseMissionPresets([
      { name: "sem id", completion_criteria: ["x"] },
      preset(),
    ]);
    expect(out.map((p) => p.id)).toEqual(["briefing-diario"]);
  });

  it("descarta preset SEM critério de conclusão", () => {
    // *Um preset sem critério de conclusão é um preset que nunca acaba.*
    const out = parseMissionPresets([
      preset({ id: "sem-criterio", completion_criteria: [] }),
      preset({ id: "bom" }),
    ]);
    expect(out.map((p) => p.id)).toEqual(["bom"]);
  });

  it("recusa payload que não é lista", () => {
    expect(parseMissionPresets(null)).toEqual([]);
    expect(parseMissionPresets({ presets: [] })).toEqual([]);
  });
});

describe("summarizePreset", () => {
  it("não inventa orçamento quando o preset não declara", () => {
    const s = summarizePreset(preset({ budget_seconds: 0, budget_tokens: 0 }));
    expect(s.budget).toBeNull();
    expect(s.permissionCount).toBe(1);
    expect(s.criteriaCount).toBe(2);
    expect(s.scheduled).toBe(true);
  });

  it("descreve o agendamento e devolve null quando não há", () => {
    expect(describeSchedule(preset({ schedule: "daily" }))).toBe("diário");
    expect(describeSchedule(preset({ schedule: "weekly" }))).toBe("semanal");
    expect(describeSchedule(preset({ schedule: "desconhecido" }))).toBe("desconhecido");
    expect(describeSchedule(preset({ schedule: undefined }))).toBeNull();
  });
});