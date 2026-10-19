// Testes da lista de espera.
//
// Discriminantes: a janela PEDIDA nao e' o slot OFERECIDO; uma oferta sem prazo nao e' valida;
// uma oferta expirada nao se mostra como aceitavel; `appointment_id` significa ja' marcado.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { WaitlistPanel } from "@/components/WaitlistPanel";
import {
  descreverAcao,
  descreverEstado,
  descreverJanelaPedida,
  ofertaValida,
} from "@/lib/waitlistClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<WaitlistPanel />);
  });
  await act(async () => {});
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

const FUTURO = "2027-01-01T00:00:00Z";
const PASSADO = "2020-01-01T00:00:00Z";

function entrada(extra: Record<string, unknown> = {}) {
  return {
    id: "w1",
    resource_id: "r1",
    service_id: "s1",
    from: "2026-11-01T09:00:00Z",
    to: "2026-11-01T18:00:00Z",
    created_at: "2026-10-19T00:00:00Z",
    status: "waiting",
    ...extra,
  };
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes da fila", () => {
  it("uma oferta SEM prazo nao e' valida", () => {
    expect(ofertaValida(entrada({ offered_slot: FUTURO }) as never)).toBe(false);
  });

  it("uma oferta com prazo futuro e' valida", () => {
    expect(ofertaValida(entrada({ offered_slot: FUTURO, offer_expires_at: FUTURO }) as never)).toBe(true);
  });

  it("uma oferta expirada NAO e' valida", () => {
    expect(ofertaValida(entrada({ offered_slot: PASSADO, offer_expires_at: PASSADO }) as never)).toBe(false);
  });

  it("a oferta sem prazo e' dita invalida, nao omissa", () => {
    const t = descreverEstado(entrada({ offered_slot: FUTURO }) as never);
    expect(t).toContain("sem prazo declarado");
    expect(t).toContain("não é válida");
  });

  it("a oferta expirada e' dita expirada com a data", () => {
    const t = descreverEstado(entrada({ offered_slot: PASSADO, offer_expires_at: PASSADO }) as never);
    expect(t).toContain("expirada");
  });

  it("a janela pedida e' dita como PEDIDA, nao como marcada", () => {
    const t = descreverJanelaPedida(entrada() as never);
    expect(t).toContain("pediu de");
    expect(t).not.toContain("marcad");
  });

  it("um agendamento ja' convertido e' dito marcado", () => {
    const t = descreverEstado(entrada({ appointment_id: "a1" }) as never);
    expect(t).toContain("já convertida em agendamento");
  });

  it("a acao diz o que se pode fazer agora, sem inventar", () => {
    expect(descreverAcao(entrada() as never)).toContain("aguardar");
    expect(descreverAcao(entrada({ offered_slot: FUTURO, offer_expires_at: FUTURO }) as never)).toContain("pode aceitar");
    expect(descreverAcao(entrada({ offered_slot: PASSADO, offer_expires_at: PASSADO }) as never)).toContain("expirou");
    expect(descreverAcao(entrada({ appointment_id: "a1" }) as never)).toContain("nada a fazer");
  });
});

describe("WaitlistPanel", () => {
  it("mostra a contagem e as ofertas validas", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      waitlist: [
        entrada({ id: "w1", offered_slot: FUTURO, offer_expires_at: FUTURO }),
        entrada({ id: "w2" }),
      ],
      count: 2,
    } as never);
    const r = await montar();

    expect(texto(r, "fila-total")).toContain("2 entrada(s)");
    expect(texto(r, "fila-total")).toContain("1 com oferta válida");
  });

  it("a janela pedida aparece, nao o slot", async () => {
    mockAgentFetch.mockResolvedValueOnce({ waitlist: [entrada()], count: 1 } as never);
    const r = await montar();

    expect(texto(r, "fila-w1")).toContain("pediu de");
  });

  it("uma oferta sem prazo e' mostrada como invalida", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      waitlist: [entrada({ offered_slot: FUTURO })],
      count: 1,
    } as never);
    const r = await montar();

    expect(texto(r, "fila-estado-w1")).toContain("não é válida");
  });

  it("a fila vazia e' dita vazia, nao erro", async () => {
    mockAgentFetch.mockResolvedValueOnce({ waitlist: [], count: 0 } as never);
    const r = await montar();

    expect(texto(r, "fila-vazia")).toContain("vazia");
    expect(porTestId(r, "fila-erro")).toHaveLength(0);
  });

  it("o 503 aparece como motor ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("motor de agendamento nao esta' ligado"));
    const r = await montar();

    expect(texto(r, "fila-ausente")).toContain("não está ligado");
    expect(porTestId(r, "fila-erro")).toHaveLength(0);
  });

  it("a fronteira distingue pedido de marcado", async () => {
    mockAgentFetch.mockResolvedValueOnce({ waitlist: [], count: 0 } as never);
    const r = await montar();

    const t = texto(r, "fila-fronteira");
    expect(t).toContain("pedida");
    expect(t).toContain("não o horário marcado");
  });
});
