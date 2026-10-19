// Testes do ciclo de supervisão.
//
// Discriminantes: NÃO dispara ao montar (é uma escrita); os QUATRO desfechos não são somados;
// `errors` vem COM resultado e não o substitui; o número do tick não pode recuar.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { BusinessTickPanel } from "@/components/BusinessTickPanel";
import {
  descreverDesfechos,
  descreverErros,
  descreverOrcamento,
  descreverTick,
  precisaDeAlguem,
  resumirAcoes,
} from "@/lib/businessTickClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<BusinessTickPanel />);
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

const TICK = {
  tick_number: 7,
  executed_at: "2026-10-19T12:00:00Z",
  signals: [],
  actions: [
    { id: "a1", kind: "restock", subject: "SKU-1", reason: "baixo", risk: "low", cost_cents: 500, status: "executed", created_at: "2026-10-19T12:00:00Z" },
    { id: "a2", kind: "refund", subject: "ord-1", reason: "pedido", risk: "high", cost_cents: 900, status: "awaiting_approval", created_at: "2026-10-19T12:00:00Z" },
  ],
  budget_spent_cents: 500,
  budget_left_cents: 1500,
  autonomous_executed: 1,
  awaiting_approval: 1,
  blocked_by_budget: 2,
  incidents_recorded: 0,
};

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes do tick", () => {
  it("os QUATRO desfechos aparecem separados, nunca somados", () => {
    const t = descreverDesfechos(TICK as never);
    expect(t).toContain("1 executada(s) sem pessoa");
    expect(t).toContain("1 à espera de uma pessoa");
    expect(t).toContain("2 bloqueada(s) por orçamento");
    expect(t).toContain("0 incidente(s)");
  });

  it("pessoa e dinheiro sao somados SO' no que precisa de alguem", () => {
    expect(precisaDeAlguem(TICK as never)).toBe(3);
  });

  it("o tick avancou quando o numero e' maior", () => {
    expect(descreverTick(TICK as never, 6)).toContain("anterior: #6");
    expect(descreverTick(TICK as never, 6)).not.toContain("NÃO avançou");
  });

  it("um tick que NAO avancou e' dito como nao corrido", () => {
    expect(descreverTick(TICK as never, 7)).toContain("NÃO avançou");
    expect(descreverTick(TICK as never, 9)).toContain("NÃO avançou");
  });

  it("errors vazio nao vira aviso", () => {
    expect(descreverErros(TICK as never)).toBeNull();
  });

  it("errors com resultado NAO e' ausencia de resultado", () => {
    const t = descreverErros({ ...TICK, errors: ["e1", "e2"] } as never);
    expect(t).toContain("2 erro(s)");
    expect(t).toContain("apesar deles");
  });

  it("o orcamento mostra os DOIS numeros", () => {
    const t = descreverOrcamento(TICK as never, null);
    expect(t).toContain("gasto: 500");
    expect(t).toContain("restante: 1500");
  });

  it("a soma do orcamento e' conferida quando o total e' dado", () => {
    expect(descreverOrcamento(TICK as never, 2000)).toContain("soma confere");
    expect(descreverOrcamento(TICK as never, 9999)).toContain("NÃO confere");
  });

  it("as acoes sao agrupadas por ESTADO, nao por total", () => {
    const t = resumirAcoes(TICK as never);
    expect(t).toContain("executed");
    expect(t).toContain("awaiting_approval");
  });

  it("sem acoes, diz que o ciclo nao produziu", () => {
    expect(resumirAcoes({ actions: [] } as never)).toContain("não produziu");
  });
});

describe("BusinessTickPanel", () => {
  it("NAO dispara um tick ao montar -- e' uma escrita", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "ciclo-pronto")).toContain("escrita");
  });

  it("sem confirmacao, recusa correr", async () => {
    const r = await montar();
    const bs = porTestId(r, "ciclo-correr");
    await act(async () => {
      bs[0].props.onClick();
    });

    expect(texto(r, "ciclo-erro")).toContain("Confirme antes");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("com confirmacao, corre e mostra os quatro desfechos", async () => {
    mockAgentFetch.mockResolvedValueOnce(TICK as never);
    const r = await montar();

    const cs = porTestId(r, "ciclo-confirmar");
    await act(async () => {
      cs[0].props.onChange({ target: { checked: true } });
    });
    const bs = porTestId(r, "ciclo-correr");
    await act(async () => {
      bs[0].props.onClick();
    });

    expect(texto(r, "ciclo-desfechos")).toContain("bloqueada(s) por orçamento");
    expect(texto(r, "ciclo-desfechos")).toContain("à espera de uma pessoa");
  });

  it("diz quantas precisam de alguem, separando pessoa de dinheiro", async () => {
    mockAgentFetch.mockResolvedValueOnce(TICK as never);
    const r = await montar();
    const cs = porTestId(r, "ciclo-confirmar");
    await act(async () => {
      cs[0].props.onChange({ target: { checked: true } });
    });
    await act(async () => {
      porTestId(r, "ciclo-correr")[0].props.onClick();
    });

    const t = texto(r, "ciclo-precisa");
    expect(t).toContain("de uma pessoa");
    expect(t).toContain("de dinheiro");
  });

  it("o pedido e' POST para /business/tick", async () => {
    mockAgentFetch.mockResolvedValueOnce(TICK as never);
    const r = await montar();
    const cs = porTestId(r, "ciclo-confirmar");
    await act(async () => {
      cs[0].props.onChange({ target: { checked: true } });
    });
    await act(async () => {
      porTestId(r, "ciclo-correr")[0].props.onClick();
    });

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/business/tick");
    expect((mockAgentFetch.mock.calls[0][1] as { method?: string }).method).toBe("POST");
  });

  it("errors aparecem mas o resultado tambem", async () => {
    mockAgentFetch.mockResolvedValueOnce({ ...TICK, errors: ["falha parcial"] } as never);
    const r = await montar();
    const cs = porTestId(r, "ciclo-confirmar");
    await act(async () => {
      cs[0].props.onChange({ target: { checked: true } });
    });
    await act(async () => {
      porTestId(r, "ciclo-correr")[0].props.onClick();
    });

    expect(texto(r, "ciclo-erros")).toContain("apesar deles");
    expect(texto(r, "ciclo-desfechos")).toContain("executada(s)");
  });

  it("o 503 aparece como supervisor ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("supervisor nao esta' ligado"));
    const r = await montar();
    const cs = porTestId(r, "ciclo-confirmar");
    await act(async () => {
      cs[0].props.onChange({ target: { checked: true } });
    });
    await act(async () => {
      porTestId(r, "ciclo-correr")[0].props.onClick();
    });

    expect(texto(r, "ciclo-ausente")).toContain("não está ligado");
  });

  it("a fronteira diz que sao quatro desfechos, nao um", async () => {
    const r = await montar();

    const t = texto(r, "ciclo-fronteira");
    expect(t).toContain("quatro desfechos, não um");
    expect(t).toContain("não são a mesma pendência");
  });
});
