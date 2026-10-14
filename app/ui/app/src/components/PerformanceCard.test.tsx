import { act, create, type ReactTestInstance } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../lib/agenticClient", () => ({ getPerformance: vi.fn() }));

import { PerformanceCard } from "./PerformanceCard";
import { getPerformance } from "../lib/agenticClient";

function texto(node: ReactTestInstance): string {
  return node.children.map((child) => (typeof child === "string" ? child : texto(child))).join("");
}

const RELATORIO = {
  performance: {
    missions: 4,
    completed: 2,
    failed: 1,
    cancelled: 0,
    running: 1,
    recovering: 0,
    completion_rate: 0.5,
    steps: 9,
    steps_with_rework: 2,
    extra_attempts: 3,
    rework_rate: 2 / 9,
    pending_approvals: 1,
    prompt_tokens: 1200,
    completion_tokens: 300,
    model_calls: 5,
    known_cost_cents: 250,
    missions_with_known_cost: 3,
    missions_with_unknown_cost: 1,
    not_measured: ["qualidade do resultado", "satisfacao do utilizador"],
  },
  scope: "missoes visiveis para o chamador (organizacao da sessao)",
};

async function renderizar(): Promise<string> {
  let arvore!: ReturnType<typeof create>;
  await act(async () => {
    arvore = create(<PerformanceCard />);
  });
  await act(async () => {});
  return texto(arvore.root);
}

describe("PerformanceCard", () => {
  beforeEach(() => {
    vi.mocked(getPerformance).mockReset();
  });

  it("mostra as métricas e DECLARA o custo desconhecido", async () => {
    vi.mocked(getPerformance).mockResolvedValue(RELATORIO as never);
    const conteudo = await renderizar();
    expect(conteudo).toContain("2/4");
    expect(conteudo).toContain("50%");
    expect(conteudo).toContain("R$ 2.50");
    // (1) O CUSTO DESCONHECIDO É DITO — não se deixa ler o total como se fosse tudo.
    expect(conteudo).toContain("1 missão com custo desconhecido");
    // (2) E O QUE NÃO SE MEDE É RENDERIZADO.
    expect(conteudo).toContain("O que este painel NÃO mede");
    expect(conteudo).toContain("qualidade do resultado");
    expect(conteudo).toContain("satisfacao do utilizador");
    // (3) O ESCOPO É DECLARADO: um agregado sem escopo é lido como global.
    expect(conteudo).toContain("missoes visiveis para o chamador");
  });

  it("quando não há custo desconhecido, não inventa o aviso", async () => {
    vi.mocked(getPerformance).mockResolvedValue({
      ...RELATORIO,
      performance: { ...RELATORIO.performance, missions_with_unknown_cost: 0 },
    } as never);
    const conteudo = await renderizar();
    expect(conteudo).not.toContain("custo desconhecido");
  });

  it("ao falhar, diz o que falhou e NÃO afirma métrica nenhuma", async () => {
    vi.mocked(getPerformance).mockRejectedValue(new Error("sem permissão"));
    const conteudo = await renderizar();
    expect(conteudo).toContain("sem permissão");
    // E nada de números: uma falha não pode deixar um painel a mostrar 0 como se fosse medido.
    expect(conteudo).not.toContain("Taxa de conclusão");
    expect(conteudo).not.toContain("R$");
  });
});
