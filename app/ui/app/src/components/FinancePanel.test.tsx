// Testes do painel financeiro.
//
// Discriminantes centrais:
//   1. o liquido aparece SEMPRE com a cobertura de reconciliacao ao lado;
//   2. com 0% de cobertura, o liquido e' declarado PROVISORIO;
//   3. «a receber» nunca e' apresentado como receita;
//   4. as divergencias nao sao escondidas;
//   5. 503 nao e' empresa sem numeros.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { FinancePanel } from "@/components/FinancePanel";
import {
  descreverDivergencia,
  descreverNaoContabilizado,
  descreverQualidadeDoLiquido,
  descreverRecebivel,
  descreverReembolso,
  formatarPercentagem,
} from "@/lib/financeClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<FinancePanel />);
  });
  return renderer;
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
    if (Array.isArray(nodo)) { nodo.forEach(visitar); return; }
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

const LIVRO = {
  company_id: "c1", currency: "BRL",
  income_cents: 125000, expense_cents: 25000, net_cents: 100000,
  entry_count: 4, reconciled_count: 4, unreconciled_count: 0,
  unreconciled_income_cents: 0, unreconciled_expense_cents: 0,
  reconciliation_coverage_pct: 100,
};

const RECEBIVEL = {
  order_id: "o1", order_number: "PED-1", total_cents: 50000,
  currency: "BRL", created_at: "2026-10-19T00:00:00Z",
};

const DIVERGENCIA = {
  order_id: "o2", order_number: "PED-2", kind: "valor_divergente",
  amount_cents: 1250, detail: "extrato diz 98750, livro diz 100000",
};

function relatorio(extra: Record<string, unknown> = {}) {
  return {
    company_id: "c1",
    ledger: LIVRO,
    receivable_cents: 50000,
    receivables: [RECEBIVEL],
    unposted_revenue_cents: 0,
    refund_due_cents: 0,
    divergences: [],
    orders_paid: 3, orders_fulfilled: 2, orders_pending: 1, orders_cancelled: 0,
    ...extra,
  };
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de financeiro", () => {
  it("0% de reconciliacao torna o liquido PROVISORIO", () => {
    const texto0 = descreverQualidadeDoLiquido({ ...LIVRO, reconciliation_coverage_pct: 0 });
    expect(texto0).toContain("PROVISORIO");
    expect(texto0).toContain("soma dos lancamentos");
  });

  it("cobertura parcial e' declarada parcial", () => {
    expect(
      descreverQualidadeDoLiquido({ ...LIVRO, reconciliation_coverage_pct: 62.5 }),
    ).toContain("62,5% dos lancamentos conferidos");
  });

  it("cobertura total diz conciliado", () => {
    expect(descreverQualidadeDoLiquido(LIVRO)).toContain("conciliado");
  });

  it("«a receber» e' dito como ainda nao entrado", () => {
    const t = descreverRecebivel(50000, 1);
    expect(t).toContain("a receber");
    expect(t).toContain("ainda nao entrou");
    expect(descreverRecebivel(0, 0)).toBe("nada a receber");
  });

  it("nao contabilizado e reembolso distinguem zero de valor", () => {
    expect(descreverNaoContabilizado(0)).toContain("nenhuma receita por contabilizar");
    expect(descreverNaoContabilizado(100)).toContain("ainda nao contabilizada");
    expect(descreverReembolso(0)).toBe("nenhum reembolso devido");
    expect(descreverReembolso(500)).toContain("a devolver");
  });

  it("divergencia traz tipo, valor e detalhe", () => {
    const t = descreverDivergencia(DIVERGENCIA);
    expect(t).toContain("PED-2");
    expect(t).toContain("valor_divergente");
    expect(t).toContain("extrato diz");
  });

  it("percentagem usa virgula pt-BR", () => {
    expect(formatarPercentagem(62.5)).toBe("62,5%");
    expect(formatarPercentagem(100)).toBe("100%");
  });
});

describe("FinancePanel", () => {
  it("mostra o liquido E a cobertura no mesmo ecra", async () => {
    mockAgentFetch.mockResolvedValueOnce(relatorio());

    const renderer = await montar();

    expect(texto(renderer, "financeiro-liquido")).toContain("1.000,00");
    expect(texto(renderer, "financeiro-cobertura")).toContain("100%");
    expect(texto(renderer, "financeiro-qualidade")).toContain("conciliado");
  });

  it("com 0% de cobertura, o liquido e' declarado PROVISORIO", async () => {
    mockAgentFetch.mockResolvedValueOnce(
      relatorio({ ledger: { ...LIVRO, reconciliation_coverage_pct: 0 } }),
    );

    const renderer = await montar();

    expect(texto(renderer, "financeiro-qualidade")).toContain("PROVISORIO");
    expect(texto(renderer, "financeiro-liquido")).toContain("1.000,00");
  });

  it("«a receber» nao e' somado como receita: aparece em bloco proprio", async () => {
    mockAgentFetch.mockResolvedValueOnce(relatorio());

    const renderer = await montar();

    const recebivel = texto(renderer, "financeiro-recebivel");
    expect(recebivel).toContain("a receber");
    expect(recebivel).toContain("ainda nao entrou");
    // O liquido continua a ser o do livro, sem os 50000.
    expect(texto(renderer, "financeiro-liquido")).toContain("1.000,00");
    expect(texto(renderer, "financeiro-liquido")).not.toContain("1.500,00");
  });

  it("lista os recebiveis com numero e valor", async () => {
    mockAgentFetch.mockResolvedValueOnce(relatorio());

    const renderer = await montar();

    expect(texto(renderer, "financeiro-recebiveis")).toContain("PED-1");
  });

  it("as divergencias sao mostradas, com tipo e detalhe", async () => {
    mockAgentFetch.mockResolvedValueOnce(relatorio({ divergences: [DIVERGENCIA] }));

    const renderer = await montar();

    const d = texto(renderer, "financeiro-divergencias");
    expect(d).toContain("PED-2");
    expect(d).toContain("valor_divergente");
    expect(d).toContain("extrato diz");
  });

  it("sem divergencias mas com lancamentos, nao diz que esta' tudo bem", async () => {
    mockAgentFetch.mockResolvedValueOnce(relatorio());

    const renderer = await montar();

    expect(texto(renderer, "financeiro-sem-divergencias")).toContain(
      "nenhuma divergência registada",
    );
  });

  it("livro vazio diz que nao ha lancamentos", async () => {
    mockAgentFetch.mockResolvedValueOnce(
      relatorio({
        ledger: { ...LIVRO, entry_count: 0, reconciled_count: 0, net_cents: 0,
          income_cents: 0, expense_cents: 0, reconciliation_coverage_pct: 0 },
        receivable_cents: 0, receivables: [], divergences: [],
      }),
    );

    const renderer = await montar();

    expect(texto(renderer, "financeiro-sem-divergencias")).toContain("Não há lançamentos");
  });

  it("mostra o que falta contabilizar e o reembolso devido", async () => {
    mockAgentFetch.mockResolvedValueOnce(
      relatorio({ unposted_revenue_cents: 25000, refund_due_cents: 10000 }),
    );

    const renderer = await montar();

    expect(texto(renderer, "financeiro-nao-contabilizado")).toContain("ainda nao contabilizada");
    expect(texto(renderer, "financeiro-reembolso")).toContain("a devolver");
  });

  it("o 503 aparece como financeiro AUSENTE, nao como empresa sem numeros", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("company financeira nao configurada"));

    const renderer = await montar();

    const sem = texto(renderer, "financeiro-sem-financeiro");
    expect(sem).toContain("não está configurado");
    expect(sem).toContain("não é «a empresa não tem números»");
    expect(porTestId(renderer, "financeiro-erro")).toHaveLength(0);
  });

  it("a chamada usa o prefixo explicito e e' GET", async () => {
    mockAgentFetch.mockResolvedValueOnce(relatorio());

    await montar();

    expect(mockAgentFetch.mock.calls).toHaveLength(1);
    expect(mockAgentFetch.mock.calls[0][0]).toBe("/api/agent/v1/finance/report");
    const init = mockAgentFetch.mock.calls[0][1] as RequestInit | undefined;
    expect(init?.method ?? "GET").toBe("GET");
  });
});
