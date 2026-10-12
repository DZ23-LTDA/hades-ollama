import { act, create, type ReactTestInstance, type ReactTestRenderer } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { BusinessTrendChart, TREND_METRICS, filterTrendPoints, trendAxisLabels, trendPoints } from "./BusinessTrendChart";
import type { BusinessSnapshot } from "@/lib/agenticClient";

function snapshot(window: string, revenueCents: number): BusinessSnapshot {
  return {
    id: `bsn_${window}`,
    organization_id: "local",
    taken_at: `${window}:00:00Z`,
    window,
    report: {
      commerce: { revenue_cents: revenueCents, orders: 1 },
      finance: { net_cents: revenueCents, income_cents: revenueCents, expense_cents: 0, unreconciled: 0 },
      campaigns: { active: 0, impressions: 0 },
      sales: { proposals: 0, follow_ups_overdue: 0 },
      inbox: { sla_violations: 0 },
    },
  } as unknown as BusinessSnapshot;
}

function texto(node: ReactTestInstance): string {
  return node.children.map((child) => (typeof child === "string" ? child : texto(child))).join("");
}
function linhasDaTabela(root: ReactTestRenderer): number {
  return root.root.findAll((node) => node.type === "tr").length - 1; // menos o cabecalho
}

const receita = TREND_METRICS[0];
const serie = [1000, 3000, 5000, 7000, 9000, 11000].map((valor, indice) => snapshot(`2026-10-13T${10 + indice}`, valor));

// E-272: eixo numerico, tooltip por FOCO e selecao de intervalo. A propriedade central: o grafico e a tabela NAO
// podem discordar - se filtrassem listas diferentes, a fonte acessivel mentiria sobre o grafico.
describe("BusinessTrendChart — eixo, tooltip e intervalo", () => {
  beforeEach(() => {
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
  });

  it("rotula o eixo com maximo, meio e minimo (dinheiro dividido por 100)", () => {
    const pontos = trendPoints(serie, receita);
    const eixo = trendAxisLabels(pontos, receita);
    expect(eixo.max).toBe("110.00");
    expect(eixo.min).toBe("10.00");
    expect(eixo.mid).toBe("60.00");
  });

  it("sem amostras nao inventa rotulo de eixo", () => {
    expect(trendAxisLabels([], receita)).toEqual({ max: "", mid: "", min: "" });
  });

  it("filtra o intervalo pelos ULTIMOS pontos e mantem tabela e curva com a MESMA contagem", async () => {
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessTrendChart history={serie} metric={receita} rangeLimits={[3, 0]} />);
    });
    expect(linhasDaTabela(root)).toBe(6);
    const botao = root.root.findAll((node) => node.type === "button").find((node) => texto(node).includes("Últimas 3"));
    expect(botao).toBeTruthy();
    await act(async () => {
      botao!.props.onClick();
    });
    // A tabela e a curva saem da MESMA lista: 3 linhas e 3 circulos.
    expect(linhasDaTabela(root)).toBe(3);
    expect(root.root.findAll((node) => node.type === "circle")).toHaveLength(3);
  });

  it("mostra o valor exato no FOCO (teclado), nao apenas no mouse", async () => {
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessTrendChart history={serie} metric={receita} />);
    });
    const circulos = root.root.findAll((node) => node.type === "circle");
    expect(circulos.length).toBe(6);
    // A dica e o elemento role=status; o eixo TAMBEM mostra 110.00, entao a assercao precisa ser especifica.
    const dica = () =>
      root.root
        .findAll((node) => node.props?.role === "status")
        .map((node) => texto(node))
        .join(" | ");
    expect(dica()).not.toContain("2026-10-13T15");
    await act(async () => {
      circulos[5].props.onFocus();
    });
    expect(dica()).toContain("2026-10-13T15: 110.00");
    await act(async () => {
      circulos[5].props.onBlur();
    });
    expect(dica()).not.toContain("2026-10-13T15");
    // E o mouse mostra O MESMO que o teclado: duas verdades diferentes seria bug de acessibilidade.
    await act(async () => {
      circulos[5].props.onMouseEnter();
    });
    expect(dica()).toContain("2026-10-13T15: 110.00");
  });

  it("filterTrendPoints com limite maior que a serie devolve a serie inteira", () => {
    expect(filterTrendPoints(serie, receita, 90)).toHaveLength(6);
    expect(filterTrendPoints(serie, receita, 4)).toHaveLength(4);
  });
});
