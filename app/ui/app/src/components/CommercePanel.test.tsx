// Testes do painel de comércio.
//
// O caso que mais importa é o DINHEIRO: `125000` cêntimos são `R$ 1.250,00`. Um
// teste que só verificasse que o total aparece passaria com o valor na unidade
// errada -- e um valor de dinheiro na unidade errada é uma mentira com duas casas
// decimais.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { CommercePanel } from "@/components/CommercePanel";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<CommercePanel />);
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

const PRODUTOS = [
  { id: "p1", organization_id: "o1", sku: "CAM-01", name: "Camisola", price_cents: 125000, currency: "BRL", active: true, created_at: "2026-10-19T00:00:00Z" },
  { id: "p2", organization_id: "o1", sku: "CAN-02", name: "Caneca", price_cents: 2590, currency: "BRL", active: false, created_at: "2026-10-19T00:00:00Z" },
];
const PEDIDOS = [
  { id: "o1", organization_id: "o1", number: "PED-1", status: "pending", items: [], total_cents: 125000, currency: "BRL", idempotency_key: "k1", created_at: "2026-10-19T00:00:00Z", updated_at: "2026-10-19T00:00:00Z" },
  { id: "o2", organization_id: "o1", number: "PED-2", status: "paid", items: [], total_cents: 2590, currency: "BRL", idempotency_key: "k2", created_at: "2026-10-19T00:00:00Z", updated_at: "2026-10-19T00:00:00Z" },
  { id: "o3", organization_id: "o1", number: "PED-2", status: "paid", items: [], total_cents: 2590, currency: "BRL", idempotency_key: "k2", created_at: "2026-10-19T00:00:00Z", updated_at: "2026-10-19T00:00:00Z", duplicated: true },
  { id: "o4", organization_id: "o1", number: "PED-3", status: "estado-novo", items: [], total_cents: 0, currency: "BRL", idempotency_key: "k3", created_at: "2026-10-19T00:00:00Z", updated_at: "2026-10-19T00:00:00Z" },
];
const RESUMO = {
  organization_id: "o1", products: 2, orders: 4, pending: 1, paid: 1,
  fulfilled: 0, cancelled: 0, revenue_cents: 125000, low_stock_skus: ["CAM-01"],
};

function responder() {
  mockAgentFetch
    .mockResolvedValueOnce({ products: PRODUTOS, count: PRODUTOS.length })
    .mockResolvedValueOnce({ orders: PEDIDOS, count: PEDIDOS.length })
    .mockResolvedValueOnce(RESUMO);
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("CommercePanel", () => {
  it("o 503 aparece como LOJA AUSENTE, não como lista vazia", async () => {
    const erro = new Error("ErrCommerceInvalid");
    mockAgentFetch.mockRejectedValueOnce(erro).mockRejectedValueOnce(erro).mockRejectedValueOnce(erro);

    const renderer = await montar();

    const semLoja = texto(renderer, "commerce-sem-loja");
    expect(semLoja).toContain("não está ligada");
    expect(semLoja).toContain("não é «não há produtos»");
    expect(porTestId(renderer, "commerce-erro")).toHaveLength(0);
    expect(porTestId(renderer, "commerce-sem-produtos")).toHaveLength(0);
  });

  it("CENTAVOS VIRAM REAIS: 125000 -> R$ 1.250,00", async () => {
    responder();

    const renderer = await montar();

    const produtos = texto(renderer, "commerce-produtos");
    expect(produtos).toContain("R$ 1.250,00");
    expect(produtos).toContain("R$ 25,90");
    // O numero cru de centavos NAO pode aparecer como se fosse dinheiro.
    expect(produtos).not.toContain("125000");
    expect(produtos).not.toContain("R$ 1250,00");
  });

  it("a receita do resumo também é convertida", async () => {
    responder();

    const renderer = await montar();

    const resumo = texto(renderer, "commerce-resumo");
    expect(resumo).toContain("R$ 1.250,00");
    expect(resumo).toContain("2 produto(s)");
    expect(resumo).toContain("4 pedido(s)");
  });

  it("NÃO conta a devolução idempotente como pedido novo", async () => {
    responder();

    const renderer = await montar();

    const contagem = texto(renderer, "commerce-contagem-pedidos");
    expect(contagem).toContain("3 novo(s)");
    expect(contagem).toContain("1 devolução(ões) idempotente(s)");
    // A tabela marca a devolução.
    expect(texto(renderer, "commerce-pedidos")).toContain("devolução idempotente");
  });

  it("mostra os totais dos pedidos em reais", async () => {
    responder();

    const renderer = await montar();

    const pedidos = texto(renderer, "commerce-pedidos");
    expect(pedidos).toContain("R$ 1.250,00");
    expect(pedidos).toContain("R$ 25,90");
  });

  it("estado desconhecido não ganha rótulo inventado", async () => {
    responder();

    const renderer = await montar();

    expect(texto(renderer, "commerce-pedidos")).toContain("estado desconhecido (estado-novo)");
  });

  it("mostra os SKUs sem disponibilidade que o servidor reporta", async () => {
    responder();

    const renderer = await montar();

    expect(texto(renderer, "commerce-stock-baixo")).toContain("CAM-01");
  });

  it("loja ligada com zero produtos diz isso, e é diferente de loja ausente", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ products: [], count: 0 })
      .mockResolvedValueOnce({ orders: [], count: 0 })
      .mockResolvedValueOnce({ ...RESUMO, products: 0, orders: 0, revenue_cents: 0, low_stock_skus: [] });

    const renderer = await montar();

    expect(texto(renderer, "commerce-sem-produtos")).toContain("loja está ligada");
    expect(porTestId(renderer, "commerce-sem-loja")).toHaveLength(0);
  });

  it("um erro que não é 503 continua a ser erro", async () => {
    const erro = new Error("organizacao invalida");
    mockAgentFetch.mockRejectedValueOnce(erro).mockRejectedValueOnce(erro).mockRejectedValueOnce(erro);

    const renderer = await montar();

    expect(texto(renderer, "commerce-erro")).toContain("organizacao invalida");
    expect(porTestId(renderer, "commerce-sem-loja")).toHaveLength(0);
  });

  it("as três chamadas usam o prefixo explícito e são GET", async () => {
    responder();

    await montar();

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string).sort();
    expect(urls).toEqual([
      "/api/agent/v1/commerce/orders",
      "/api/agent/v1/commerce/products",
      "/api/agent/v1/commerce/summary",
    ]);
    for (const chamada of mockAgentFetch.mock.calls) {
      const init = chamada[1] as RequestInit | undefined;
      expect(init?.method ?? "GET").toBe("GET");
    }
  });

  it("a fronteira diz que só lê", async () => {
    responder();

    const renderer = await montar();

    const fronteira = texto(renderer, "commerce-fronteira");
    expect(fronteira).toContain("só lê");
    expect(fronteira).toContain("centavos");
  });
});
