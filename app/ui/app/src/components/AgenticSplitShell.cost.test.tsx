import { act, create, type ReactTestInstance } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  AgenticSplitShell,
  formatMissionCost,
  type MissionData,
} from "./AgenticSplitShell";

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: vi.fn().mockResolvedValue({ events: [] }),
  agentFetchBlob: vi
    .fn()
    .mockResolvedValue(new Blob(["artifact"], { type: "text/plain" })),
}));

/**
 * Consumo e ORIGEM na linha do tempo (limite declarado em E-244). A regra que não pode ser quebrada: custo
 * DESCONHECIDO nunca vira "R$ 0,00" — isso afirmaria gratuidade que ninguém mediu.
 */

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

const missaoBase: MissionData = {
  id: "mis_consumo",
  objective: "medir consumo e origem",
  state: "COMPLETED",
  plan: [
    {
      id: "step_1",
      title: "Inspecionar",
      kind: "workspace.list",
      state: "SUCCEEDED",
      requires_approval: false,
    },
  ],
  approvals: [],
  artifacts: [],
};

async function renderizar(mission: MissionData) {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(
      <AgenticSplitShell mission={mission} onRefreshMission={vi.fn().mockResolvedValue(undefined)} />,
    );
    await Promise.resolve();
  });
  return textContent(renderer.root);
}

describe("AgenticSplitShell consumo", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    vi.stubGlobal(
      "EventSource",
      class {
        addEventListener = vi.fn();
        removeEventListener = vi.fn();
        close = vi.fn();
      },
    );
  });

  it("mostra consumo e origem quando a missao tem uso registrado", async () => {
    const texto = await renderizar({
      ...missaoBase,
      usage: { calls: 1, prompt_tokens: 1200, completion_tokens: 300, cost_cents: 12, cost_known: true },
      usage_origin: "planning",
    });
    expect(texto).toContain("Consumo:");
    expect(texto).toContain("1 chamada");
    expect(texto).toContain("1500 tokens");
    expect(texto).toContain("origem: planning");
  });

  it("nao cria linha de consumo quando nao ha uso", async () => {
    const texto = await renderizar(missaoBase);
    expect(texto).not.toContain("Consumo:");
  });

  it("declara custo DESCONHECIDO em vez de afirmar R$ 0,00", () => {
    const texto = formatMissionCost({ calls: 2, cost_known: false });
    expect(texto).toContain("2 chamadas");
    expect(texto).toContain("custo desconhecido");
    expect(texto).not.toContain("R$");
  });

  it("mostra custo conhecido em pt-BR e usa plural correto", () => {
    const texto = formatMissionCost({ calls: 3, cost_cents: 1250, cost_known: true });
    expect(texto).toContain("3 chamadas");
    expect(texto).toContain("12,50");
  });

  it("nao rende nada sem uso", () => {
    expect(formatMissionCost(null)).toBe("");
    expect(formatMissionCost(undefined)).toBe("");
  });
});
