import { describe, expect, it } from "vitest";
import {
  BusinessTrendChart,
  TREND_METRICS,
  formatTrendValue,
  trendCoordinates,
  trendPoints,
  type TrendMetric,
} from "./BusinessTrendChart";
import type { BusinessSnapshot } from "@/lib/agenticClient";

// O gráfico tem QUATRO riscos clássicos, e cada teste trava um:
//  1. coordenadas `NaN` quando todos os valores são iguais (divisão por zero) — o SVG some sem avisar;
//  2. curva desenhada fora de ordem — parece oscilação real do negócio;
//  3. gráfico sem alternativa acessível;
//  4. desenhar "tendência" com uma única amostra.
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

const receita = TREND_METRICS[0];

describe("trendCoordinates", () => {
  it("não produz NaN quando todos os valores são iguais", () => {
    const points = trendPoints([snapshot("2026-10-13T10", 500), snapshot("2026-10-13T11", 500)], receita);
    const coordinates = trendCoordinates(points);
    expect(coordinates).toHaveLength(2);
    for (const coordinate of coordinates) {
      expect(Number.isNaN(coordinate.x)).toBe(false);
      expect(Number.isNaN(coordinate.y)).toBe(false);
    }
    // Amplitude zero: linha plana no meio da área, que é a representação verdadeira.
    expect(coordinates[0].y).toBe(coordinates[1].y);
  });

  it("coloca o maior valor no topo (eixo Y do SVG cresce para baixo)", () => {
    const points = trendPoints([snapshot("2026-10-13T10", 100), snapshot("2026-10-13T11", 900)], receita);
    const coordinates = trendCoordinates(points);
    expect(coordinates[0].y).toBeGreaterThan(coordinates[1].y);
    expect(coordinates[0].x).toBe(0);
    expect(coordinates[1].x).toBe(320);
  });

  it("lida com valores negativos sem NaN", () => {
    const points = trendPoints([snapshot("2026-10-13T10", -350), snapshot("2026-10-13T11", 150)], receita);
    const coordinates = trendCoordinates(points);
    expect(coordinates.every((coordinate) => Number.isFinite(coordinate.y))).toBe(true);
  });

  it("devolve nada para série vazia", () => {
    expect(trendCoordinates([])).toEqual([]);
  });
});

describe("trendPoints", () => {
  it("ordena por janela, para a curva não parecer oscilação", () => {
    const fora = [snapshot("2026-10-13T12", 300), snapshot("2026-10-13T10", 100), snapshot("2026-10-13T11", 200)];
    expect(trendPoints(fora, receita).map((point) => point.window)).toEqual([
      "2026-10-13T10",
      "2026-10-13T11",
      "2026-10-13T12",
    ]);
  });

  it("TODAS as métricas da lista produzem números finitos", () => {
    // Este teste existe por um defeito real: a métrica "Lançamentos em aberto" lia um campo que não
    // existe no tipo (`unreconciled` em vez de `unreconciled_entries`). O typecheck apontou, mas os
    // testes passavam — porque o `pick` só é executado quando aquela métrica é escolhida. Percorrer a
    // LISTA inteira é o que fecha esse buraco sem depender de quem lembra de testar cada opção.
    const completo = (window: string, base: number) =>
      ({
        id: `bsn_${window}`,
        organization_id: "local",
        taken_at: `${window}:00:00Z`,
        window,
        report: {
          commerce: { revenue_cents: base, orders: base, paid: base, cancelled: 0, products: base },
          finance: {
            entries: base,
            income_cents: base,
            expense_cents: base,
            net_cents: base,
            reconciled_cents: base,
            unreconciled_entries: base,
          },
          campaigns: { total: base, scheduled: 0, active: base, impressions: base, clicks: base, conversions: base },
          sales: { proposals: base, follow_ups: base, follow_ups_overdue: base },
          inbox: { conversations: base, open: base, sla_violations: base },
        },
      }) as unknown as BusinessSnapshot;

    expect(TREND_METRICS.length).toBeGreaterThanOrEqual(11);
    for (const metric of TREND_METRICS) {
      const points = trendPoints([completo("2026-10-13T10", 7), completo("2026-10-13T11", 9)], metric);
      for (const point of points) {
        expect(Number.isFinite(point.value)).toBe(true);
        expect(formatTrendValue(point.value, metric)).not.toMatch(/undefined|NaN/);
      }
      for (const coordinate of trendCoordinates(points)) {
        expect(Number.isFinite(coordinate.x)).toBe(true);
        expect(Number.isFinite(coordinate.y)).toBe(true);
      }
    }
  });
});

describe("BusinessTrendChart", () => {
  const metric: TrendMetric = receita;

  it("com menos de duas amostras, DIZ que não há curva em vez de fingir tendência", async () => {
    const { create } = await import("react-test-renderer");
    const { act } = await import("react");
    let root!: import("react-test-renderer").ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessTrendChart history={[snapshot("2026-10-13T10", 1234)]} metric={metric} />);
    });
    const texto = JSON.stringify(root.toJSON());
    expect(texto).toContain("Sem curva");
    expect(texto).toContain("ao menos 2 amostras");
    expect(texto).toContain("12.34"); // a amostra única ainda é mostrada, sem virar tendência
  });

  it("com duas ou mais amostras, traz a curva acessível E a tabela equivalente", async () => {
    const { create } = await import("react-test-renderer");
    const { act } = await import("react");
    let root!: import("react-test-renderer").ReactTestRenderer;
    await act(async () => {
      root = create(
        <BusinessTrendChart history={[snapshot("2026-10-13T10", 1000), snapshot("2026-10-13T11", 2500)]} metric={metric} />,
      );
    });
    const texto = JSON.stringify(root.toJSON());
    // A curva existe com coordenadas válidas e NOME acessível.
    expect(texto).toContain("polyline");
    expect(texto).toContain("Receita (pedidos): 2 amostra(s)");
    expect(texto).not.toContain("NaN");
    // A tabela equivalente é a MESMA informação em forma acessível, e está sempre visível.
    expect(texto).toContain("Janela");
    expect(texto).toContain("Variação");
    expect(texto).toContain("2026-10-13T10");
    expect(texto).toContain("10.00");
    expect(texto).toContain("25.00");
    expect(texto).toContain("+15.00");
    // A primeira linha não tem variação: mostrar "+0" inventaria um marco que não existe.
    expect(texto).toContain("—");
  });

  it("declara linha plana quando todos os valores são iguais", async () => {
    const { create } = await import("react-test-renderer");
    const { act } = await import("react");
    let root!: import("react-test-renderer").ReactTestRenderer;
    await act(async () => {
      root = create(
        <BusinessTrendChart history={[snapshot("2026-10-13T10", 700), snapshot("2026-10-13T11", 700)]} metric={metric} />,
      );
    });
    const texto = JSON.stringify(root.toJSON());
    expect(texto).toContain("linha é plana");
    expect(texto).not.toContain("NaN");
  });
});
