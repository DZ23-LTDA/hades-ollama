// Testes dos lançamentos financeiros.
//
// Discriminantes: NÃO lança ao montar; a RECEITA de pedido não pago é recusada pelo servidor (receita
// fictícia); as três operações NÃO devolvem o mesmo status; o montante é em CENTAVOS com o factor 100
// visível; o id vai no caminho codificado.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { FinanceOrderPanel } from "@/components/FinanceOrderPanel";
import {
  descreverFronteiraDaReceita,
  descreverOperacao,
  formatarCentavos,
  validarCusto,
  validarMontante,
} from "@/lib/financeOrderClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<FinanceOrderPanel />);
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
  const encontrados: string[] = [];
  const visitar = (nodo: unknown) => {
    if (Array.isArray(nodo)) { nodo.forEach(visitar); return; }
    if (nodo === null || typeof nodo !== "object") return;
    const props = (nodo as { props?: Record<string, unknown> }).props;
    if (props && props["data-testid"] === id) encontrados.push(juntarTexto(nodo));
    visitar((nodo as { children?: unknown }).children);
  };
  visitar(renderer.toJSON());
  return encontrados.join(" ");
}

function porTestId(renderer: ReturnType<typeof create>, id: string) {
  return renderer.root.findAll((nodo) => nodo.props["data-testid"] === id);
}

async function preencher(renderer: ReturnType<typeof create>, campos: Record<string, string>) {
  for (const [id, valor] of Object.entries(campos)) {
    const el = porTestId(renderer, `financeiro-${id}`);
    await act(async () => {
      el[0].props.onChange({ target: { value: valor } });
    });
  }
}

async function confirmar(renderer: ReturnType<typeof create>) {
  const cs = porTestId(renderer, "financeiro-confirmar");
  await act(async () => {
    cs[0].props.onChange({ target: { checked: true } });
  });
}

async function clicar(renderer: ReturnType<typeof create>, botao: string) {
  const bs = porTestId(renderer, `financeiro-${botao}`);
  await act(async () => {
    bs[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes financeiras", () => {
  it("o montante em reais (nao inteiro) e' recusado", () => {
    const t = validarMontante(12.5);
    expect(t).toContain("CENTAVOS");
    expect(t).toContain("não em reais");
  });

  it("montante zero e' recusado", () => {
    expect(validarMontante(0)).toContain("positivo");
    expect(validarMontante(1250)).toBeNull();
  });

  it("montante em falta e' dito com a unidade", () => {
    expect(validarMontante(undefined)).toContain("centavos");
  });

  it("categoria e referencia sao ditas por nome", () => {
    const t = validarCusto("", "");
    expect(t).toContain("a categoria");
    expect(t).toContain("a referência");
  });

  it("o factor 100 fica visivel: 1250 centavos = R$ 12,50", () => {
    expect(formatarCentavos(1250)).toBe("R$ 12,50");
    expect(formatarCentavos(125000)).toBe("R$ 1.250,00");
  });

  it("os centavos com zero a esquerda aparecem", () => {
    expect(formatarCentavos(5)).toBe("R$ 0,05");
    expect(formatarCentavos(100)).toBe("R$ 1,00");
  });

  it("montante invalido vira travessao, nao zero", () => {
    expect(formatarCentavos(undefined)).toBe("—");
  });

  it("a fronteira da receita diz que o painel NAO sabe se foi pago", () => {
    const t = descreverFronteiraDaReceita();
    expect(t).toContain("não sabe se o pedido foi pago");
    expect(t).toContain("receita fictícia");
  });

  it("cada operacao declara o seu status -- e as tres NAO sao iguais", () => {
    expect(descreverOperacao("revenue")).toContain("201");
    expect(descreverOperacao("revenue")).toContain("200");
    expect(descreverOperacao("cost")).toContain("201");
    expect(descreverOperacao("reconcile")).toContain("200");
    expect(descreverOperacao("reconcile")).toContain("não cria nada");
  });
});

describe("FinanceOrderPanel", () => {
  it("NAO lanca ao montar -- sao tres escritas", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "financeiro-pronto")).toContain("três escritas");
  });

  it("sem confirmacao, recusa lancar", async () => {
    const r = await montar();
    await preencher(r, { ordem: "ord-1" });
    await clicar(r, "receita");

    expect(texto(r, "financeiro-erro")).toContain("Confirme antes");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("custo sem montante e' recusado antes de chamar", async () => {
    const r = await montar();
    await preencher(r, { ordem: "ord-1", categoria: "frete", referencia: "nf-1" });
    await confirmar(r);
    await clicar(r, "custo");

    expect(texto(r, "financeiro-recusado")).toContain("montante");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("custo com montante NAO inteiro e' recusado -- factor 100", async () => {
    const r = await montar();
    await preencher(r, { ordem: "ord-1", centavos: "12.5", categoria: "frete", referencia: "nf-1" });
    await confirmar(r);
    await clicar(r, "custo");

    expect(texto(r, "financeiro-recusado")).toContain("CENTAVOS");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("receita vai para /finance/orders/:id/revenue com POST e sem corpo", async () => {
    mockAgentFetch.mockResolvedValueOnce({ id: "l-1", kind: "revenue" } as never);
    const r = await montar();
    await preencher(r, { ordem: "ord-1" });
    await confirmar(r);
    await clicar(r, "receita");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(caminho).toBe("/api/agent/v1/finance/orders/ord-1/revenue");
    expect(init.method).toBe("POST");
    // Revenue nao leva corpo: o valor vem do pedido pago.
    expect(JSON.parse(init.body ?? "{}")).toEqual({});
  });

  it("custo vai para /cost e leva o montante em centavos", async () => {
    mockAgentFetch.mockResolvedValueOnce({ id: "l-2", kind: "cost" } as never);
    const r = await montar();
    await preencher(r, { ordem: "ord-1", centavos: "1250", categoria: "frete", referencia: "nf-1" });
    await confirmar(r);
    await clicar(r, "custo");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/finance/orders/ord-1/cost");
    const corpo = JSON.parse((mockAgentFetch.mock.calls[0][1] as { body?: string }).body ?? "{}");
    expect(corpo.amount_cents).toBe(1250);
    expect(corpo.category).toBe("frete");
    expect(corpo.reference).toBe("nf-1");
  });

  it("reconciliar vai para /reconcile e leva SO' a referencia", async () => {
    mockAgentFetch.mockResolvedValueOnce({ id: "l-3" } as never);
    const r = await montar();
    await preencher(r, { ordem: "ord-1", referencia: "extrato-77" });
    await confirmar(r);
    await clicar(r, "reconciliar");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/finance/orders/ord-1/reconcile");
    const corpo = JSON.parse((mockAgentFetch.mock.calls[0][1] as { body?: string }).body ?? "{}");
    expect(corpo.reference).toBe("extrato-77");
    // Reconciliar liga a um lancamento que ja' existe: nao leva montante.
    expect(corpo.amount_cents).toBeUndefined();
  });

  it("o id vai CODIFICADO no caminho", async () => {
    mockAgentFetch.mockResolvedValueOnce({ id: "l-4" } as never);
    const r = await montar();
    await preencher(r, { ordem: "ord/1?x=2" });
    await confirmar(r);
    await clicar(r, "receita");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    // Um id cru reescreveria o caminho.
    expect(caminho).toBe("/api/agent/v1/finance/orders/ord%2F1%3Fx%3D2/revenue");
  });

  it("a RECUSA de receita de pedido nao pago e' dita como recusa do servidor", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("409: pedido aguardando pagamento; receita seria ficticia"));
    const r = await montar();
    await preencher(r, { ordem: "ord-1" });
    await confirmar(r);
    await clicar(r, "receita");

    const t = texto(r, "financeiro-recusado");
    expect(t).toContain("O servidor recusou");
    expect(t).toContain("ficticia");
    expect(porTestId(r, "financeiro-erro")).toHaveLength(0);
  });

  it("o 503 aparece como financeiro ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("financeiro nao esta' ligado"));
    const r = await montar();
    await preencher(r, { ordem: "ord-1" });
    await confirmar(r);
    await clicar(r, "receita");

    expect(texto(r, "financeiro-ausente")).toContain("não está ligado");
  });

  it("o montante aparece em reais ao lado dos centavos ANTES do erro", async () => {
    const r = await montar();
    await preencher(r, { centavos: "1250" });

    const t = texto(r, "financeiro-montante");
    expect(t).toContain("1250 centavos");
    expect(t).toContain("R$ 12,50");
  });

  it("a fronteira diz que as tres nao devolvem o mesmo status", async () => {
    const r = await montar();

    const t = texto(r, "financeiro-fronteira");
    expect(t).toContain("receita fictícia");
    expect(t).toContain("não cria nada");
  });
});
