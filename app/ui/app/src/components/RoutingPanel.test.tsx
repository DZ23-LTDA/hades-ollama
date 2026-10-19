// Testes do painel de roteamento.
//
// O que cada caso trava, e porque falharia com o defeito:
//
//   - o 422 tem de aparecer como RECUSA com as restrições, não como avaria.
//     Se aparecesse como erro genérico, o utilizador procuraria um bug do
//     produto onde o roteador tomou uma decisão.
//   - os factores têm de aparecer: um score sem os componentes que o somam
//     obriga o utilizador a confiar em quem o escreveu.
//   - a fronteira de Pareto tem de ser rotulada como ALTERNATIVAS, não ranking.
//   - a query só leva os parâmetros com valor: `capabilities=` vazio não pode
//     virar `capabilities=`.
//   - `local_only` só entra quando ligado; um `local_only=false` explícito não
//     muda o resultado do servidor, mas suja a URL.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: vi.fn(),
}));

import { agentFetch } from "@/lib/agenticClient";
import { RoutingPanel } from "@/components/RoutingPanel";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<RoutingPanel />);
  });
  return renderer;
}

async function agir(accao: () => unknown) {
  await act(async () => {
    await accao();
  });
}

function juntarTexto(nodo: unknown): string {
  if (typeof nodo === "string") return nodo;
  if (typeof nodo === "number") return String(nodo);
  if (Array.isArray(nodo)) return nodo.map(juntarTexto).join("");
  if (nodo === null || typeof nodo !== "object") return "";
  return juntarTexto((nodo as { children?: unknown }).children);
}

function texto(renderer: ReturnType<typeof create>, id: string): string {
  const arvore = renderer.toJSON();
  const encontrados: string[] = [];
  const visitar = (nodo: unknown) => {
    if (Array.isArray(nodo)) {
      nodo.forEach(visitar);
      return;
    }
    if (nodo === null || typeof nodo !== "object") return;
    const props = (nodo as { props?: Record<string, unknown> }).props;
    if (props && props["data-testid"] === id) encontrados.push(juntarTexto(nodo));
    visitar((nodo as { children?: unknown }).children);
  };
  visitar(arvore);
  return encontrados.join(" ");
}

function porTestId(renderer: ReturnType<typeof create>, id: string) {
  return renderer.root.findAll((nodo) => nodo.props["data-testid"] === id);
}

const RESPOSTA = {
  winner: { model: "qwen2.5:7b", provider: "ollama", score: 0.82, reason: "melhor qualidade local" },
  candidates: [
    {
      rank: 1,
      model: "qwen2.5:7b",
      provider: "ollama",
      score: 0.82,
      cost_cents_per_1k: 0,
      quality_score: 0.78,
      reason: "melhor qualidade local",
      factors: { capacidade: 0.9, custo: 1, latencia: 0.6 },
    },
    {
      rank: 2,
      model: "llama3.1:8b",
      provider: "ollama",
      score: 0.7,
      cost_cents_per_1k: 0,
      quality_score: 0.7,
      reason: "alternativa",
      factors: { capacidade: 0.8, custo: 1, latencia: 0.5 },
    },
  ],
  frontier: [
    { model: "qwen2.5:7b", score: 0.82, quality: 0.78, cost: 1, latency: 0.6, privacy: 1 },
    { model: "llama3.1:8b", score: 0.7, quality: 0.7, cost: 1, latency: 0.9, privacy: 1 },
  ],
};

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("RoutingPanel", () => {
  it("mostra o modelo escolhido e o motivo", async () => {
    mockAgentFetch.mockResolvedValueOnce(RESPOSTA);

    const renderer = await montar();

    expect(texto(renderer, "routing-vencedor")).toContain("qwen2.5:7b");
    expect(texto(renderer, "routing-vencedor")).toContain("0.82");
    expect(texto(renderer, "routing-motivo")).toContain("melhor qualidade local");
  });

  it("mostra os factores, não só o score", async () => {
    mockAgentFetch.mockResolvedValueOnce(RESPOSTA);

    const renderer = await montar();

    const tabela = texto(renderer, "routing-candidatos");
    expect(tabela).toContain("capacidade=0.90");
    expect(tabela).toContain("custo=1.00");
    expect(tabela).toContain("latencia=0.60");
  });

  it("mostra a fronteira de Pareto como alternativas", async () => {
    mockAgentFetch.mockResolvedValueOnce(RESPOSTA);

    const renderer = await montar();

    const fronteira = texto(renderer, "routing-fronteira");
    expect(fronteira).toContain("qwen2.5:7b");
    expect(fronteira).toContain("llama3.1:8b");
    const nota = texto(renderer, "routing-fronteira-nota");
    expect(nota).toContain("não os recalcula");
  });

  it("o 422 é recusa com as restrições, não avaria", async () => {
    mockAgentFetch.mockRejectedValueOnce(
      new Error(
        "nenhum modelo satisfaz as restricoes (capabilities=[vision] local_only=true max_cost_cents=0 path=\"/api/chat\")",
      ),
    );

    const renderer = await montar();

    const semRota = texto(renderer, "routing-sem-rota");
    expect(semRota).toContain("Nenhum modelo satisfaz");
    expect(semRota).toContain("capabilities=[vision]");
    expect(semRota).toContain("não uma avaria");
    expect(porTestId(renderer, "routing-erro")).toHaveLength(0);
  });

  it("um erro que não é 422 continua a ser erro", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("registry indisponivel"));

    const renderer = await montar();

    expect(texto(renderer, "routing-erro")).toContain("registry indisponivel");
    expect(porTestId(renderer, "routing-sem-rota")).toHaveLength(0);
  });

  it("a consulta inicial não leva parâmetros vazios", async () => {
    mockAgentFetch.mockResolvedValueOnce(RESPOSTA);

    await montar();

    expect(mockAgentFetch.mock.calls[0][0]).toBe("/api/agent/v1/routing?preference=auto");
  });

  it("leva as capacidades, o custo e o local_only quando preenchidos", async () => {
    mockAgentFetch
      .mockResolvedValueOnce(RESPOSTA)
      .mockResolvedValueOnce(RESPOSTA);

    const renderer = await montar();
    await agir(() =>
      porTestId(renderer, "routing-capacidades")[0].props.onChange({
        target: { value: "coding, vision" },
      }),
    );
    await agir(() =>
      porTestId(renderer, "routing-custo")[0].props.onChange({ target: { value: "500" } }),
    );
    await agir(() => porTestId(renderer, "routing-local")[0].props.onChange({ target: { checked: true } }));
    await agir(() => porTestId(renderer, "routing-consultar")[0].props.onClick());

    const url = mockAgentFetch.mock.calls[1][0] as string;
    expect(url).toContain("capabilities=coding%2C%20vision");
    expect(url).toContain("max_cost_cents=500");
    expect(url).toContain("local_only=true");
    expect(url).toContain("preference=auto");
  });

  it("o custo diz-se em reais, e zero diz-se grátis", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      ...RESPOSTA,
      candidates: [
        { ...RESPOSTA.candidates[0], cost_cents_per_1k: 0 },
        { ...RESPOSTA.candidates[1], cost_cents_per_1k: 250 },
      ],
    });

    const renderer = await montar();

    const tabela = texto(renderer, "routing-candidatos");
    expect(tabela).toContain("grátis");
    expect(tabela).toContain("R$ 2,50/1k");
  });

  it("só lê: nenhuma chamada de escrita", async () => {
    mockAgentFetch.mockResolvedValueOnce(RESPOSTA);

    await montar();

    for (const chamada of mockAgentFetch.mock.calls) {
      const init = chamada[1] as RequestInit | undefined;
      expect(init?.method ?? "GET").toBe("GET");
    }
  });
});
