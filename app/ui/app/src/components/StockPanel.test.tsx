// Testes do stock por SKU.
//
// Discriminantes: on_hand != reserved != available; o saldo e' DERIVADO (a soma e' conferida);
// um saldo que nao bate com os movimentos e' mostrado como NAO conforme.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { StockPanel } from "@/components/StockPanel";
import {
  conferirDerivacao,
  conferirDisponibilidade,
  descreverIndisponivel,
  descreverMovimento,
  descreverSaldo,
} from "@/lib/stockClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<StockPanel />);
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

const SALDO = { sku: "SKU-1", on_hand: 10, reserved: 3, available: 7, movement_count: 2 };
const MOVS = [
  { id: "m1", sku: "SKU-1", delta: 12, kind: "entrada", created_at: "2026-10-19T00:00:00Z" },
  { id: "m2", sku: "SKU-1", delta: -2, kind: "saida", created_at: "2026-10-19T01:00:00Z" },
];

function responder(r: Record<string, unknown> = { balance: SALDO, movements: MOVS }) {
  mockAgentFetch.mockResolvedValueOnce(r as never);
}

async function consultar(renderer: ReturnType<typeof create>, valor: string) {
  const campos = porTestId(renderer, "stock-sku");
  await act(async () => {
    campos[0].props.onChange({ target: { value: valor } });
  });
  const botoes = porTestId(renderer, "stock-consultar");
  await act(async () => {
    botoes[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de stock", () => {
  it("os TRES numeros aparecem, nao so' um", () => {
    const t = descreverSaldo(SALDO as never);
    expect(t).toContain("7 disponível");
    expect(t).toContain("10 em armazém");
    expect(t).toContain("3 reservado");
  });

  it("o que ja' tem dono e' dito com todas as letras", () => {
    const t = descreverIndisponivel(SALDO as never);
    expect(t).toContain("já têm dono");
    expect(t).toContain("não estão disponíveis");
  });

  it("sem reservas, nao inventa aviso", () => {
    expect(descreverIndisponivel({ on_hand: 5, reserved: 0, available: 5 } as never)).toBeNull();
  });

  it("o sinal do movimento e' dito por palavras", () => {
    expect(descreverMovimento({ delta: 5 } as never)).toContain("entrada");
    expect(descreverMovimento({ delta: -3 } as never)).toContain("saída");
    expect(descreverMovimento({ delta: 0 } as never)).toContain("sem alteração");
  });

  it("o saldo e' conferido contra a soma dos movimentos", () => {
    const t = conferirDerivacao({ balance: SALDO, movements: MOVS } as never);
    expect(t).toContain("saldo conferido");
    expect(t).toContain("2 movimento(s)");
  });

  it("um saldo que NAO bate e' dito como nao conforme", () => {
    const t = conferirDerivacao({
      balance: { ...SALDO, on_hand: 99 },
      movements: MOVS,
    } as never);
    expect(t).toContain("saldo NÃO confere");
  });

  it("sem movimentos nao ha' prova -- devolve null, nao 'conferido'", () => {
    expect(conferirDerivacao({ balance: SALDO, movements: [] } as never)).toBeNull();
  });

  it("available tem de ser on_hand - reserved", () => {
    expect(conferirDisponibilidade(SALDO as never)).toBeNull();
    const t = conferirDisponibilidade({ on_hand: 10, reserved: 3, available: 9 } as never);
    expect(t).toContain("não confere");
  });
});

describe("StockPanel", () => {
  it("sem SKU, pede o SKU em vez de mostrar zero", async () => {
    const renderer = await montar();

    expect(texto(renderer, "stock-ausente")).toContain("Indique um SKU");
    expect(porTestId(renderer, "stock-saldo")).toHaveLength(0);
  });

  it("mostra os tres numeros do saldo", async () => {
    responder();
    const renderer = await montar();
    await consultar(renderer, "SKU-1");

    const t = texto(renderer, "stock-saldo");
    expect(t).toContain("7 disponível");
    expect(t).toContain("10 em armazém");
    expect(t).toContain("3 reservado");
  });

  it("o que nao esta' disponivel e' marcado", async () => {
    responder();
    const renderer = await montar();
    await consultar(renderer, "SKU-1");

    expect(texto(renderer, "stock-indisponivel")).toContain("já têm dono");
  });

  it("a derivacao do saldo e' conferida e mostrada", async () => {
    responder();
    const renderer = await montar();
    await consultar(renderer, "SKU-1");

    expect(texto(renderer, "stock-derivacao")).toContain("saldo conferido");
  });

  it("um saldo incoerente e' mostrado como erro, nao escondido", async () => {
    responder({ balance: { ...SALDO, available: 99 }, movements: MOVS });
    const renderer = await montar();
    await consultar(renderer, "SKU-1");

    expect(texto(renderer, "stock-disponibilidade")).toContain("não confere");
  });

  it("os movimentos mostram o sinal", async () => {
    responder();
    const renderer = await montar();
    await consultar(renderer, "SKU-1");

    expect(texto(renderer, "stock-mov-m1")).toContain("+12 (entrada)");
    expect(texto(renderer, "stock-mov-m2")).toContain("-2 (saída)");
  });

  it("o 503 aparece como store ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("commerce nao esta' ligado"));
    const renderer = await montar();
    await consultar(renderer, "SKU-1");

    expect(texto(renderer, "stock-ausente")).toContain("não está ligado");
    expect(porTestId(renderer, "stock-erro")).toHaveLength(0);
  });

  it("o SKU e' codificado no caminho", async () => {
    responder();
    const renderer = await montar();
    await consultar(renderer, "a/b");

    expect(mockAgentFetch.mock.calls[0][0]).toBe(
      "/api/agent/v1/commerce/stock/a%2Fb",
    );
  });

  it("a fronteira diz que o saldo e' derivado", async () => {
    const renderer = await montar();

    const t = texto(renderer, "stock-fronteira");
    expect(t).toContain("derivado");
    expect(t).toContain("não é");
  });
});
