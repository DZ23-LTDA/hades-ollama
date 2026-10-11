import { act, create, type ReactTestInstance, type ReactTestRenderer } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { BusinessStatusPanel } from "./BusinessStatusPanel";
import type { BusinessStatusReport } from "@/lib/agenticClient";

vi.mock("@/lib/agenticClient", () => ({
  getBusinessStatus: vi.fn(),
  getBusinessTrend: vi.fn(),
  recordBusinessSnapshot: vi.fn(),
  downloadBusinessExport: vi.fn(),
}));

// O helper de download manipula DOM (indisponivel no ambiente de teste): aqui ele e substituido, e o
// comportamento dele e testado em src/lib/download.test.ts.
vi.mock("@/lib/download", () => ({ triggerDownload: vi.fn() }));

import { downloadBusinessExport, getBusinessStatus, getBusinessTrend, recordBusinessSnapshot } from "@/lib/agenticClient";
import { triggerDownload } from "@/lib/download";

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
    // Padrão: série com uma amostra (sem base). Cada teste sobrescreve quando precisa de tendência.
    vi.mocked(getBusinessTrend).mockResolvedValue({
      organization_id: "local", snapshots: 1, sem_base: true, motivo: "apenas uma amostra",
      history: [], amostras_descartadas: 0, retencao_maxima: 96,
    } as never);
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

  it("sem duas amostras, mostra o MOTIVO em vez de uma variação de zero", async () => {
    vi.mocked(getBusinessTrend).mockResolvedValue({
      organization_id: "local", snapshots: 1, sem_base: true,
      motivo: "apenas uma amostra no período: sem duas medições não há tendência",
      history: [], amostras_descartadas: 0, retencao_maxima: 96,
    } as never);
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("Sem tendência: apenas uma amostra no período");
    expect(rendered).toContain("retenção máxima 96");
    // "não mudou" e "não há com o que comparar" são coisas diferentes: nenhum delta é exibido.
    expect(rendered).not.toContain("Intervalo comparado");
  });

  it("com duas amostras, mostra o intervalo e os deltas com sinal", async () => {
    vi.mocked(getBusinessTrend).mockResolvedValue({
      organization_id: "local", snapshots: 2, intervalo: "2026-10-13T12:00:00Z → 2026-10-13T14:00:00Z",
      sem_base: false,
      deltas: [
        { metric: "pedidos", from: 3, to: 4, delta: 1 },
        { metric: "liquido_cents", from: 1500, to: 500, delta: -1000 },
        { metric: "violacoes_sla", from: 1, to: 0, delta: -1 },
      ],
      history: [], amostras_descartadas: 2, retencao_maxima: 96,
    } as never);
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("Intervalo comparado: 2026-10-13T12:00:00Z → 2026-10-13T14:00:00Z");
    // Sinal explícito: alta e queda são distinguíveis de relance.
    expect(rendered).toContain("pedidos: +1 (3 → 4)");
    expect(rendered).toContain("liquido_cents: -1000 (1500 → 500)");
    expect(rendered).toContain("violacoes_sla: -1 (1 → 0)");
    // E o que ficou fora do recorte é declarado.
    expect(rendered).toContain("2 fora do recorte");
  });

  it("registrar amostra diz a VERDADE quando a janela já tinha amostra", async () => {
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });

    // Primeiro: amostra realmente nova.
    vi.mocked(recordBusinessSnapshot).mockResolvedValue({
      created: true,
      snapshot: { id: "bsn_1", organization_id: "local", taken_at: "2026-10-13T12:00:00Z", window: "2026-10-13T12", report: reportWith() },
      trend: { organization_id: "local", snapshots: 1, sem_base: true, motivo: "apenas uma amostra", history: [], amostras_descartadas: 0, retencao_maxima: 96 },
    } as never);
    await act(async () => {
      buttonWithText(root, "Registrar amostra agora")?.props.onClick();
    });
    expect(textContent(root.root)).toContain("Amostra registrada agora.");

    // Depois: a MESMA hora já tinha amostra — o servidor é idempotente por janela e a tela diz isso,
    // em vez de comemorar um dado novo que não existe.
    vi.mocked(recordBusinessSnapshot).mockResolvedValue({
      created: false,
      snapshot: { id: "bsn_1", organization_id: "local", taken_at: "2026-10-13T12:00:00Z", window: "2026-10-13T12", report: reportWith() },
      trend: { organization_id: "local", snapshots: 1, sem_base: true, motivo: "apenas uma amostra", history: [], amostras_descartadas: 0, retencao_maxima: 96 },
    } as never);
    await act(async () => {
      buttonWithText(root, "Registrar amostra agora")?.props.onClick();
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("havia amostra nesta hora");
    expect(rendered).not.toContain("Amostra registrada agora.");
  });

  it("falha ao registrar amostra aparece como falha", async () => {
    vi.mocked(recordBusinessSnapshot).mockRejectedValue(new Error("série indisponível: falha simulada"));
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });
    await act(async () => {
      buttonWithText(root, "Registrar amostra agora")?.props.onClick();
    });
    expect(textContent(root.root)).toContain("série indisponível: falha simulada");
  });

  it("exporta CSV declarando que o arquivo leva as lacunas e a origem", async () => {
    const blob = new Blob(["secao,chave,valor,observacao\n"], { type: "text/csv" });
    vi.mocked(downloadBusinessExport).mockResolvedValue(blob as never);
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });
    // A nota ligada ao botão (aria-describedby) diz o que o arquivo carrega.
    const rendered = textContent(root.root);
    expect(rendered).toContain("A exportação é CSV com esquema único");
    expect(rendered).toContain("leva as lacunas");
    expect(rendered).toContain("origem de cada número");

    await act(async () => {
      buttonWithText(root, "Exportar CSV")?.props.onClick();
    });
    expect(downloadBusinessExport).toHaveBeenCalledWith("cmp_1");
    expect(vi.mocked(triggerDownload)).toHaveBeenCalledWith(blob, "hades-business-status.csv");
    expect(textContent(root.root)).toContain("hades-business-status.csv gerado");
    expect(textContent(root.root)).toContain("inclui o que NÃO é medido");
  });

  it("falha na exportação aparece como falha, sem dizer que exportou", async () => {
    vi.mocked(downloadBusinessExport).mockRejectedValue(new Error("exportação indisponível: falha simulada"));
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });
    await act(async () => {
      buttonWithText(root, "Exportar CSV")?.props.onClick();
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("exportação indisponível: falha simulada");
    expect(vi.mocked(triggerDownload)).not.toHaveBeenCalled();
    expect(rendered).not.toContain("gerado");
  });

  it("com amostras, mostra o gráfico com tabela equivalente e permite trocar a métrica", async () => {
    vi.mocked(getBusinessTrend).mockResolvedValue({
      organization_id: "local",
      snapshots: 2,
      intervalo: "2026-10-13T10 → 2026-10-13T11",
      sem_base: false,
      deltas: [{ metric: "pedidos", from: 1, to: 3, delta: 2 }],
      history: [
        { id: "bsn_a", organization_id: "local", taken_at: "2026-10-13T10:00:00Z", window: "2026-10-13T10", report: reportWith() },
        { id: "bsn_b", organization_id: "local", taken_at: "2026-10-13T11:00:00Z", window: "2026-10-13T11", report: reportWith() },
      ],
      amostras_descartadas: 0,
      retencao_maxima: 96,
    } as never);
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("Gráfico da série");
    expect(rendered).toContain("Métrica do gráfico");
    // A curva tem nome acessível e a tabela equivalente está visível.
    expect(rendered).toContain("Receita (pedidos): 2 amostra(s)");
    expect(rendered).toContain("Janela");
    expect(rendered).not.toContain("NaN");

    // Trocar a métrica muda o NOME acessível — o gráfico não fica "mudo" ao mudar de série.
    const select = root.root.findAll((node) => node.type === "select")[0];
    await act(async () => {
      select.props.onChange({ target: { value: "pedidos" } });
    });
    const depois = textContent(root.root);
    expect(depois).toContain("Pedidos: 2 amostra(s)");
    expect(depois).not.toContain("Receita (pedidos): 2 amostra(s)");
  });

  it("sem amostras, NÃO inventa gráfico", async () => {
    vi.mocked(getBusinessTrend).mockResolvedValue({
      organization_id: "local",
      snapshots: 0,
      sem_base: true,
      motivo: "nenhuma amostra registrada",
      history: [],
      amostras_descartadas: 0,
      retencao_maxima: 96,
    } as never);
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<BusinessStatusPanel companyId="cmp_1" />);
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("Sem tendência: nenhuma amostra registrada");
    expect(rendered).not.toContain("Gráfico da série");
  });
});
