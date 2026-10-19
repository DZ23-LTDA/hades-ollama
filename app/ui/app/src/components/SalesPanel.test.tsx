// Testes do painel de vendas.
//
// Dois casos centrais, ambos com um teste que falharia se o defeito estivesse presente:
//   1. o frete NAO declarado nao pode aparecer como R$ 0,00;
//   2. a conta tem de aparecer ABERTA -- nao basta o total.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { SalesPanel } from "@/components/SalesPanel";
import { abrirAConta, descreverFrete, formatarPontosBase } from "@/lib/salesClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<SalesPanel />);
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

const PROPOSTA = {
  id: "p1", organization_id: "o1", contact_id: "c1", currency: "BRL",
  items: [
    { sku: "SKU-1", name: "Consultoria", quantity: 2, unit_price_cents: 50000, total_cents: 100000 },
  ],
  subtotal_cents: 100000,
  discount_basis_points: 500,
  discount_cents: 5000,
  tax_basis_points: 2300,
  tax_cents: 23000,
  shipping_cents: 2500,
  shipping_declared: true,
  total_cents: 120500,
  status: "sent",
  valid_until: "2026-11-19T00:00:00Z",
  created_at: "2026-10-19T00:00:00Z",
  updated_at: "2026-10-19T00:00:00Z",
  sent_at: "2026-10-19T12:00:00Z",
};

const PROPOSTA_SEM_FRETE = {
  ...PROPOSTA,
  id: "p2",
  shipping_cents: 0,
  shipping_declared: false,
  total_cents: 118000,
};

const ACOMPANHAMENTOS = [
  { id: "a1", organization_id: "o1", contact_id: "c1", reason: "confirmar entrega",
    status: "pending", due_at: "2026-10-15T00:00:00Z",
    created_at: "2026-10-10T00:00:00Z" },
];

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de vendas", () => {
  it("frete nao declarado nao e' R$ 0,00", () => {
    expect(descreverFrete(PROPOSTA_SEM_FRETE)).toBe("frete nao declarado");
    expect(descreverFrete(PROPOSTA)).toBe("R$ 25,00");
  });

  it("a conta abre em cinco linhas, com o total no fim", () => {
    const linhas = abrirAConta(PROPOSTA);
    expect(linhas.map((l) => l.etiqueta)).toEqual([
      "subtotal", "desconto (5%)", "imposto (23%)", "frete", "total",
    ]);
    expect(linhas[0].valor).toBe("R$ 1.000,00");
    expect(linhas[1].valor).toBe("- R$ 50,00");
    expect(linhas[4].valor).toBe("R$ 1.205,00");
  });

  it("pontos-base viram percentagem legivel", () => {
    expect(formatarPontosBase(500)).toBe("5%");
    expect(formatarPontosBase(1250)).toBe("12,5%");
    expect(formatarPontosBase(0)).toBe("0%");
  });
});

describe("SalesPanel", () => {
  it("a conta aparece ABERTA: subtotal, desconto, imposto, frete e total", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ proposals: [PROPOSTA], count: 1 })
      .mockResolvedValueOnce({ follow_ups: ACOMPANHAMENTOS, count: 1 })
      .mockResolvedValueOnce(PROPOSTA);

    const renderer = await montar();
    await agir(() => porTestId(renderer, "vendas-abrir-p1")[0].props.onClick());

    const conta = texto(renderer, "vendas-conta");
    expect(conta).toContain("subtotal");
    expect(conta).toContain("desconto (5%)");
    expect(conta).toContain("imposto (23%)");
    expect(conta).toContain("frete");
    expect(conta).toContain("total");
    // As parcelas, nao so' o total.
    expect(conta).toContain("R$ 1.000,00");
    expect(conta).toContain("R$ 1.205,00");
    // E nunca em centavos.
    expect(conta).not.toContain("120500");
    expect(conta).not.toContain("100000");
  });

  it("frete NAO declarado aparece como 'frete nao declarado', nunca R$ 0,00", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ proposals: [PROPOSTA_SEM_FRETE], count: 1 })
      .mockResolvedValueOnce({ follow_ups: [], count: 0 })
      .mockResolvedValueOnce(PROPOSTA_SEM_FRETE);

    const renderer = await montar();
    await agir(() => porTestId(renderer, "vendas-abrir-p2")[0].props.onClick());

    const conta = texto(renderer, "vendas-conta");
    expect(conta).toContain("frete nao declarado");
    // A linha do frete NAO pode ser R$ 0,00.
    const linhaFrete = conta.split("frete")[1] ?? "";
    expect(linhaFrete.slice(0, 30)).not.toContain("R$ 0,00");
  });

  it("a lista mostra a contagem de unidades e o total em reais", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ proposals: [PROPOSTA], count: 1 })
      .mockResolvedValueOnce({ follow_ups: [], count: 0 });

    const renderer = await montar();

    const lista = texto(renderer, "vendas-propostas");
    expect(lista).toContain("2 unidade(s)");
    expect(lista).toContain("R$ 1.205,00");
    expect(lista).not.toContain("120500");
  });

  it("mostra os itens com quantidade e SKU", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ proposals: [PROPOSTA], count: 1 })
      .mockResolvedValueOnce({ follow_ups: [], count: 0 })
      .mockResolvedValueOnce(PROPOSTA);

    const renderer = await montar();
    await agir(() => porTestId(renderer, "vendas-abrir-p1")[0].props.onClick());

    const itens = texto(renderer, "vendas-itens");
    expect(itens).toContain("2× Consultoria");
    expect(itens).toContain("SKU-1");
    expect(itens).toContain("R$ 1.000,00");
  });

  it("o 503 aparece como vendas AUSENTES, nao como lista vazia", async () => {
    const erro = new Error("força de vendas não está disponível no runtime");
    mockAgentFetch.mockRejectedValueOnce(erro).mockRejectedValueOnce(erro);

    const renderer = await montar();

    const semVendas = texto(renderer, "vendas-sem-vendas");
    expect(semVendas).toContain("não está ligada");
    expect(semVendas).toContain("não é «não há propostas»");
    expect(porTestId(renderer, "vendas-erro")).toHaveLength(0);
  });

  it("vendas ligadas com zero propostas e' diferente de ausente", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ proposals: [], count: 0 })
      .mockResolvedValueOnce({ follow_ups: [], count: 0 });

    const renderer = await montar();

    expect(texto(renderer, "vendas-sem-propostas")).toContain("não há propostas");
    expect(porTestId(renderer, "vendas-sem-vendas")).toHaveLength(0);
  });

  it("o filtro de estado vai na query", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ proposals: [], count: 0 })
      .mockResolvedValueOnce({ follow_ups: [], count: 0 })
      .mockResolvedValueOnce({ proposals: [], count: 0 })
      .mockResolvedValueOnce({ follow_ups: [], count: 0 });

    const renderer = await montar();
    await agir(() =>
      porTestId(renderer, "vendas-filtro")[0].props.onChange({ target: { value: "sent" } }),
    );
    await agir(() => porTestId(renderer, "vendas-filtrar")[0].props.onClick());

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string);
    expect(urls).toContain("/api/agent/v1/sales/proposals?status=sent");
  });

  it("'so atrasados' manda overdue=true e diz que nao ha atrasados e' diferente de nao haver", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ proposals: [], count: 0 })
      .mockResolvedValueOnce({ follow_ups: [], count: 0 })
      .mockResolvedValueOnce({ proposals: [], count: 0 })
      .mockResolvedValueOnce({ follow_ups: [], count: 0 });

    const renderer = await montar();
    await agir(() =>
      porTestId(renderer, "vendas-so-atrasados")[0].props.onChange({ target: { checked: true } }),
    );
    await agir(() => porTestId(renderer, "vendas-filtrar")[0].props.onClick());

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string);
    expect(urls).toContain("/api/agent/v1/sales/follow-ups?overdue=true");
    expect(texto(renderer, "vendas-sem-acompanhamentos")).toContain("diferente de");
  });

  it("mostra o acompanhamento com vencimento", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ proposals: [], count: 0 })
      .mockResolvedValueOnce({ follow_ups: ACOMPANHAMENTOS, count: 1 });

    const renderer = await montar();

    const linha = texto(renderer, "vendas-acompanhamento-a1");
    expect(linha).toContain("confirmar entrega");
    expect(linha).toContain("2026-10-15");
  });

  it("falha ao ler o detalhe mostra erro e nao esconde a lista", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ proposals: [PROPOSTA], count: 1 })
      .mockResolvedValueOnce({ follow_ups: [], count: 0 })
      .mockRejectedValueOnce(new Error("proposta nao encontrada"));

    const renderer = await montar();
    await agir(() => porTestId(renderer, "vendas-abrir-p1")[0].props.onClick());

    expect(texto(renderer, "vendas-detalhe-erro")).toContain("proposta nao encontrada");
    expect(texto(renderer, "vendas-propostas")).toContain("p1");
  });

  it("as chamadas usam o prefixo explicito e sao GET", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ proposals: [], count: 0 })
      .mockResolvedValueOnce({ follow_ups: [], count: 0 });

    await montar();

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string).sort();
    expect(urls).toEqual([
      "/api/agent/v1/sales/follow-ups",
      "/api/agent/v1/sales/proposals",
    ]);
    for (const chamada of mockAgentFetch.mock.calls) {
      const init = chamada[1] as RequestInit | undefined;
      expect(init?.method ?? "GET").toBe("GET");
    }
  });
});
