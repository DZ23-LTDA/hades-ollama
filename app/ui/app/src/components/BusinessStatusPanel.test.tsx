import { act, create, type ReactTestInstance, type ReactTestRenderer } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { BusinessStatusPanel } from "./BusinessStatusPanel";
import type { BusinessStatusReport } from "@/lib/agenticClient";

vi.mock("@/lib/agenticClient", () => ({
  getBusinessStatus: vi.fn(),
}));

import { getBusinessStatus } from "@/lib/agenticClient";

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

function buttonWithText(root: ReactTestRenderer, text: string): ReactTestInstance | undefined {
  return root.root.findAll((node) => node.type === "button").find((node) => textContent(node).includes(text));
}

function reportWith(overrides: Partial<BusinessStatusReport> = {}): BusinessStatusReport {
  return {
    organization_id: "local",
    generated_at: "2026-10-13T12:00:00Z",
    companies: 1,
    commerce: { products: 2, orders: 3, paid: 2, fulfilled: 0, cancelled: 1, revenue_cents: 3500, low_stock_skus: ["BI-LOW"] },
    finance: { entries: 2, income_cents: 2500, expense_cents: 1000, net_cents: 1500, reconciled_cents: 2500, unreconciled_entries: 1 },
    campaigns: { total: 1, draft: 0, approved: 0, scheduled: 1, active: 0, impressions: 300, clicks: 30, conversions: 3, spend_cents: 1500, publicacao_externa: "BLOCKED_BY_EXTERNAL_DEPENDENCY" },
    sales: { proposals: 1, follow_ups: 1, follow_ups_overdue: 1 },
    inbox: { conversations: 1, open: 1, closed: 0, sla_violations: 1 },
    marketplace_channels_incompletos: 3,
    sources: [
      { metric: "commerce", source: "CommerceStore (pedidos, produtos e estoque)" },
      { metric: "finance", source: "CompanyStore.ListLedgerEntries (livro financeiro por empresa)" },
    ],
    nao_medido: [
      "faturamento fiscal e obrigações tributárias: exigem integração fiscal autorizada",
      "custo de tokens e de modelo: exigem provedor externo com preço declarado",
    ],
    sandbox: true,
    ...overrides,
  };
}

const emptyReport = reportWith({
  commerce: { products: 0, orders: 0, paid: 0, fulfilled: 0, cancelled: 0, revenue_cents: 0 },
  finance: { entries: 0, income_cents: 0, expense_cents: 0, net_cents: 0, reconciled_cents: 0, unreconciled_entries: 0 },
  campaigns: { total: 0, draft: 0, approved: 0, scheduled: 0, active: 0, impressions: 0, clicks: 0, conversions: 0, spend_cents: 0, publicacao_externa: "BLOCKED_BY_EXTERNAL_DEPENDENCY" },
  sales: { proposals: 0, follow_ups: 0, follow_ups_overdue: 0 },
  inbox: { conversations: 0, open: 0, closed: 0, sla_violations: 0 },
});

describe("BusinessStatusPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
  });

  it("mostra os indicadores reais, a origem de cada número e a lacuna declarada", async () => {
    vi.mocked(getBusinessStatus).mockResolvedValue(reportWith() as never);
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });
    const rendered = textContent(root.root);
    expect(getBusinessStatus).toHaveBeenCalledWith("cmp_1");
    // Indicadores reais, formatados em reais a partir de centavos.
    expect(rendered).toContain("Receita");
    expect(rendered).toContain("35.00");
    expect(rendered).toContain("Follow-ups vencidos");
    expect(rendered).toContain("Violações de SLA");
    // PROVENIÊNCIA visível: quem desconfia do número precisa poder conferir de onde ele veio.
    expect(rendered).toContain("CommerceStore (pedidos, produtos e estoque)");
    // E a LACUNA na tela, com motivo — não escondida.
    expect(rendered).toContain("O que este painel NÃO mede");
    expect(rendered).toContain("faturamento fiscal e obrigações tributárias");
    expect(rendered).toContain("custo de tokens e de modelo");
    expect(rendered).toContain("BLOCKED_BY_EXTERNAL_DEPENDENCY");
  });

  it("sem dados, DIZ que não há dados em vez de exibir zero como resposta", async () => {
    vi.mocked(getBusinessStatus).mockResolvedValue(emptyReport as never);
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("Sem dados ainda nesta empresa");
    // As lacunas continuam declaradas mesmo sem dado nenhum.
    expect(rendered).toContain("O que este painel NÃO mede");
    // Nenhum selo verde de "tudo certo" quando não há o que medir.
    expect(rendered.toLowerCase()).not.toContain("tudo em ordem");
  });

  it("falha do servidor aparece como falha e não mantém número antigo como se fosse atual", async () => {
    vi.mocked(getBusinessStatus).mockResolvedValue(reportWith() as never);
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });
    expect(textContent(root.root)).toContain("35.00");

    vi.mocked(getBusinessStatus).mockRejectedValue(new Error("BI indisponível: falha simulada"));
    await act(async () => {
      buttonWithText(root, "Atualizar painel")?.props.onClick();
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("BI indisponível: falha simulada");
    // O número anterior NÃO fica na tela: dado velho apresentado como atual é pior que nenhum.
    expect(rendered).not.toContain("35.00");
  });

  it("o controle de atualizar realmente refaz a consulta", async () => {
    vi.mocked(getBusinessStatus).mockResolvedValue(reportWith() as never);
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });
    expect(vi.mocked(getBusinessStatus).mock.calls.length).toBe(1);
    await act(async () => {
      buttonWithText(root, "Atualizar painel")?.props.onClick();
    });
    expect(vi.mocked(getBusinessStatus).mock.calls.length).toBe(2);
  });
});
