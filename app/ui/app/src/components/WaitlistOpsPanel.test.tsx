// Testes da fila de espera e do no-show.
//
// Discriminantes: NÃO corre ao montar; FILA VAZIA é `offered:false` com 200 e NÃO é erro;
// «faltou» não é «cancelou»; `now` NÃO vai no corpo; o painel não calcula o prazo da oferta.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { WaitlistOpsPanel } from "@/components/WaitlistOpsPanel";
import {
  descreverNoShow,
  descreverOferta,
  descreverPrazoDaOferta,
  validarSlot,
  validarVaga,
} from "@/lib/waitlistOpsClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<WaitlistOpsPanel />);
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
    const el = porTestId(renderer, `fila-${id}`);
    await act(async () => {
      el[0].props.onChange({ target: { value: valor } });
    });
  }
}

async function confirmar(renderer: ReturnType<typeof create>) {
  const cs = porTestId(renderer, "fila-confirmar");
  await act(async () => {
    cs[0].props.onChange({ target: { checked: true } });
  });
}

async function clicar(renderer: ReturnType<typeof create>, botao: string) {
  const bs = porTestId(renderer, `fila-${botao}`);
  await act(async () => {
    bs[0].props.onClick();
  });
}

const AG = {
  id: "ag-9",
  resource_id: "r1",
  service_id: "s1",
  starts_at: "2026-11-06T14:00:00Z",
  ends_at: "2026-11-06T14:30:00Z",
  busy_until: "2026-11-06T14:45:00Z",
  status: "no_show",
  created_at: "2026-10-19T12:00:00Z",
};

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes da fila", () => {
  it("slot vazio e' recusado com o formato", () => {
    const t = validarSlot("");
    expect(t).toContain("obrigatório");
    expect(t).toContain("RFC 3339");
  });

  it("slot sem fuso e' recusado", () => {
    expect(validarSlot("2026-11-06T14:00:00")).toContain("RFC 3339");
    expect(validarSlot("2026-11-06T14:00:00Z")).toBeNull();
  });

  it("recurso em falta e' dito por nome", () => {
    expect(validarVaga("", "2026-11-06T14:00:00Z")).toContain("resource_id");
  });

  it("FILA VAZIA e' dita como ausencia, nao como falha", () => {
    const t = descreverOferta({ offered: false });
    expect(t).toContain("fila está vazia");
    expect(t).toContain("não é falha");
    expect(t).toContain("ausência de oferta");
  });

  it("uma oferta real mostra entrada, horario e expiracao", () => {
    const t = descreverOferta({
      offered: true,
      offer: { entry_id: "e-1", slot: "2026-11-06T14:00:00Z", expires_at: "2026-11-06T12:00:00Z" },
    });
    expect(t).toContain("e-1");
    expect(t).toContain("2026-11-06T14:00:00Z");
    expect(t).toContain("expira em");
  });

  it("a oferta sem detalhes NAO inventa detalhes", () => {
    const t = descreverOferta({ offered: true });
    expect(t).toContain("não devolveu os detalhes");
  });

  it("o prazo da oferta e' do SERVIDOR, e o painel di-lo", () => {
    expect(descreverPrazoDaOferta(undefined)).toContain("prazo é do servidor");
    const t = descreverPrazoDaOferta("2026-11-06T12:00:00Z");
    expect(t).toContain("recusado pelo motor");
  });

  it("FALTOU nao e' cancelou -- e o texto di-lo", () => {
    const t = descreverNoShow({ appointment: AG } as never);
    expect(t).toContain("FALTOU");
    expect(t).toContain("não é um cancelamento");
    expect(t).toContain("no_show");
  });
});

describe("WaitlistOpsPanel", () => {
  it("NAO corre ao montar -- sao tres escritas", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "fila-pronto")).toContain("três escritas");
  });

  it("sem confirmacao, recusa correr", async () => {
    const r = await montar();
    await preencher(r, { recurso: "r1", slot: "2026-11-06T14:00:00Z" });
    await clicar(r, "oferecer");

    expect(texto(r, "fila-erro")).toContain("Confirme antes");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("oferecer sem recurso e' recusado antes de chamar", async () => {
    const r = await montar();
    await preencher(r, { slot: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r, "oferecer");

    expect(texto(r, "fila-recusado")).toContain("resource_id");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("slot invalido e' recusado antes de chamar", async () => {
    const r = await montar();
    await preencher(r, { recurso: "r1", slot: "amanha" });
    await confirmar(r);
    await clicar(r, "oferecer");

    expect(texto(r, "fila-recusado")).toContain("RFC 3339");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("FILA VAZIA aparece como ausencia, NAO como erro", async () => {
    mockAgentFetch.mockResolvedValueOnce({ offered: false } as never);
    const r = await montar();
    await preencher(r, { recurso: "r1", slot: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r, "oferecer");

    const t = texto(r, "fila-vazia");
    expect(t).toContain("fila está vazia");
    expect(t).toContain("não é falha");
    // Nao pode cair no estado de erro nem no de recusa.
    expect(porTestId(r, "fila-erro")).toHaveLength(0);
    expect(porTestId(r, "fila-recusado")).toHaveLength(0);
  });

  it("uma oferta real aparece com os detalhes", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      offered: true,
      offer: { entry_id: "e-1", slot: "2026-11-06T14:00:00Z" },
    } as never);
    const r = await montar();
    await preencher(r, { recurso: "r1", slot: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r, "oferecer");

    expect(texto(r, "fila-oferta")).toContain("e-1");
  });

  it("o pedido de oferta NAO leva `now` -- um corpo que escolhe «agora» escolhe o prazo", async () => {
    mockAgentFetch.mockResolvedValueOnce({ offered: false } as never);
    const r = await montar();
    await preencher(r, { recurso: "r1", slot: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r, "oferecer");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(caminho).toBe("/api/agent/v1/scheduling/waitlist/offer");
    expect(init.method).toBe("POST");
    const corpo = JSON.parse(init.body ?? "{}");
    expect(corpo.resource_id).toBe("r1");
    expect(corpo.slot).toBe("2026-11-06T14:00:00Z");
    expect(corpo.now).toBeUndefined();
  });

  it("uma oferta EXPIRADA aparece como recusa do motor", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("400: oferta expirada"));
    const r = await montar();
    await preencher(r, { recurso: "r1", slot: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r, "oferecer");

    expect(texto(r, "fila-recusado")).toContain("expirada");
  });

  it("confirmar vai para /confirm e leva SO' o entry_id", async () => {
    mockAgentFetch.mockResolvedValueOnce({ appointment: AG } as never);
    const r = await montar();
    await preencher(r, { entrada: "e-1" });
    await confirmar(r);
    await clicar(r, "confirmar-oferta");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/scheduling/waitlist/confirm");
    const corpo = JSON.parse((mockAgentFetch.mock.calls[0][1] as { body?: string }).body ?? "{}");
    expect(corpo.entry_id).toBe("e-1");
    expect(corpo.now).toBeUndefined();
  });

  it("marcar falta vai para /no-show e NAO diz cancelamento", async () => {
    mockAgentFetch.mockResolvedValueOnce({ appointment: AG } as never);
    const r = await montar();
    await preencher(r, { ordem: "ag-9" });
    await confirmar(r);
    await clicar(r, "faltou");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/scheduling/no-show");
    const corpo = JSON.parse((mockAgentFetch.mock.calls[0][1] as { body?: string }).body ?? "{}");
    expect(corpo.appointment_id).toBe("ag-9");

    const t = texto(r, "fila-falta");
    expect(t).toContain("FALTOU");
    expect(t).toContain("não é um cancelamento");
  });

  it("o 503 aparece como agenda ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("motor de agendamento nao esta' ligado"));
    const r = await montar();
    await preencher(r, { recurso: "r1", slot: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r, "oferecer");

    expect(texto(r, "fila-ausente")).toContain("não está ligado");
  });

  it("a fronteira separa faltou de cancelou e fila vazia de falha", async () => {
    const r = await montar();

    const t = texto(r, "fila-fronteira");
    expect(t).toContain("não é quem cancelou");
    expect(t).toContain("não é falha");
    expect(t).toContain("ausência de oferta");
  });
});
