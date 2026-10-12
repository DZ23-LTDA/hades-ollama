import { act, create, type ReactTestInstance } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  AgenticSplitShell,
  formatStepResult,
  STEP_RESULT_MAX_CHARS,
  type MissionData,
} from "./AgenticSplitShell";

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: vi.fn().mockResolvedValue({ events: [] }),
  agentFetchBlob: vi
    .fn()
    .mockResolvedValue(new Blob(["artifact"], { type: "text/plain" })),
}));

/**
 * Lacuna declarada do estagio 17: a linha do tempo mostrava o ERRO da etapa, mas nao a SAIDA DA FERRAMENTA —
 * uma etapa que produziu resultado parecia nao ter produzido nada.
 */

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

const missaoComSaida: MissionData = {
  id: "mis_saida_ferramenta",
  objective: "Inspecionar a saida de ferramenta por etapa",
  state: "COMPLETED",
  plan: [
    {
      id: "step_com_saida",
      title: "Ler o arquivo de notas",
      kind: "workspace.read",
      state: "COMPLETED",
      requires_approval: false,
      result: { arquivo: "notas.txt", linhas: 12 },
    },
    {
      id: "step_sem_saida",
      title: "Aguardar a proxima etapa",
      kind: "wait",
      state: "COMPLETED",
      requires_approval: false,
    },
  ],
  approvals: [],
  artifacts: [],
};

const missaoSemSaida: MissionData = {
  ...missaoComSaida,
  plan: [missaoComSaida.plan![1]],
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

describe("AgenticSplitShell saida de ferramenta", () => {
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

  it("mostra a saida da ferramenta na etapa que produziu resultado", async () => {
    const texto = await renderizar(missaoComSaida);
    expect(texto).toContain("Saída da ferramenta");
    expect(texto).toContain("notas.txt");
    expect(texto).toContain("12");
  });

  it("nao cria bloco de saida quando nenhuma etapa produziu resultado", async () => {
    const texto = await renderizar(missaoSemSaida);
    expect(texto).not.toContain("Saída da ferramenta");
  });

  it("formata string, numero, booleano e objeto", () => {
    expect(formatStepResult("  pronto  ")).toBe("pronto");
    expect(formatStepResult(7)).toBe("7");
    expect(formatStepResult(false)).toBe("false");
    expect(formatStepResult({ a: 1 })).toContain('"a": 1');
  });

  it("nao rende bloco para vazio, nulo ou indefinido", () => {
    expect(formatStepResult(null)).toBe("");
    expect(formatStepResult(undefined)).toBe("");
    expect(formatStepResult("   ")).toBe("");
  });

  it("declara o truncamento em vez de cortar em silencio", () => {
    const longo = "x".repeat(STEP_RESULT_MAX_CHARS + 25);
    const texto = formatStepResult(longo);
    expect(texto).toContain("truncada");
    expect(texto).toContain("25 caracteres não exibidos");
    expect(texto.length).toBeGreaterThan(STEP_RESULT_MAX_CHARS);
  });
});
