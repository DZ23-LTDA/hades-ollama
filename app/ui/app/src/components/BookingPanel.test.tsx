// Testes da marcação de agendamento.
//
// Discriminantes: NÃO marca ao montar; sem confirmação nada acontece; 409 é REENTREGA e não falha;
// `busy_until` (buffer) não é `ends_at`; a organização NÃO vai no corpo.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { BookingPanel } from "@/components/BookingPanel";
import {
  bufferEmMinutos,
  descreverIdempotencia,
  descreverMarcacao,
  descreverOcupacao,
  validarAlvo,
  validarInicio,
} from "@/lib/bookingClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<BookingPanel />);
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
    const el = porTestId(renderer, `marcar-${id}`);
    await act(async () => {
      el[0].props.onChange({ target: { value: valor } });
    });
  }
}

async function confirmar(renderer: ReturnType<typeof create>) {
  const cs = porTestId(renderer, "marcar-confirmar");
  await act(async () => {
    cs[0].props.onChange({ target: { checked: true } });
  });
}

async function clicar(renderer: ReturnType<typeof create>) {
  const bs = porTestId(renderer, "marcar-enviar");
  await act(async () => {
    bs[0].props.onClick();
  });
}

const AG = {
  id: "ag-9",
  resource_id: "r1",
  service_id: "s1",
  starts_at: "2026-11-05T10:00:00Z",
  ends_at: "2026-11-05T10:30:00Z",
  busy_until: "2026-11-05T10:45:00Z",
  status: "confirmed",
  created_at: "2026-10-19T12:00:00Z",
};

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de marcacao", () => {
  it("inicio vazio e' recusado com o formato", () => {
    const t = validarInicio("");
    expect(t).toContain("obrigatório");
    expect(t).toContain("RFC 3339");
  });

  it("inicio sem fuso e' recusado", () => {
    expect(validarInicio("2026-11-05T10:00:00")).toContain("RFC 3339");
    expect(validarInicio("2026-11-05T10:00:00Z")).toBeNull();
    expect(validarInicio("2026-11-05T10:00:00-03:00")).toBeNull();
  });

  it("recurso e servico em falta sao ditos por nome", () => {
    const t = validarAlvo("", "");
    expect(t).toContain("o recurso");
    expect(t).toContain("o serviço");
  });

  it("sem chave de idempotencia, o risco e' dito", () => {
    expect(descreverIdempotencia("")).toContain("SEGUNDO agendamento");
  });

  it("com chave, a promessa e' o MESMO recurso", () => {
    const t = descreverIdempotencia("k-1");
    expect(t).toContain("MESMO agendamento");
    expect(t).toContain("não uma cópia");
  });

  it("o buffer e' medido de ends_at a busy_until", () => {
    expect(bufferEmMinutos(AG as never)).toBe(15);
  });

  it("a ocupacao diz o buffer, nao esconde", () => {
    const t = descreverOcupacao(AG as never);
    expect(t).toContain("buffer de 15 min");
    expect(t).toContain("busy_until".replace("busy_until", "10:45"));
  });

  it("sem buffer, diz sem buffer", () => {
    const semBuffer = { ...AG, busy_until: AG.ends_at };
    expect(descreverOcupacao(semBuffer as never)).toContain("sem buffer");
  });

  it("a marcacao criada diz o estado real", () => {
    const t = descreverMarcacao({ appointment: AG } as never);
    expect(t).toContain("ag-9");
    expect(t).toContain("criado");
    expect(t).toContain("confirmed");
  });

  it("uma reentrega e' dita como reentrega", () => {
    const t = descreverMarcacao({ appointment: AG, idempotent_replay: true } as never);
    expect(t).toContain("reentrega");
    expect(t).toContain("já existia");
  });
});

describe("BookingPanel", () => {
  it("NAO marca ao montar -- e' uma escrita", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "marcar-pronto")).toContain("escrita");
  });

  it("sem recurso/servico, recusa antes de chamar", async () => {
    const r = await montar();
    await clicar(r);

    expect(texto(r, "marcar-invalido")).toContain("o recurso");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("inicio invalido e' recusado antes de chamar", async () => {
    const r = await montar();
    await preencher(r, { recurso: "r1", servico: "s1", inicio: "amanha" });
    await clicar(r);

    expect(texto(r, "marcar-invalido")).toContain("RFC 3339");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("sem confirmacao, recusa marcar", async () => {
    const r = await montar();
    await preencher(r, { recurso: "r1", servico: "s1", inicio: "2026-11-05T10:00:00Z" });
    await clicar(r);

    expect(texto(r, "marcar-erro")).toContain("Confirme antes");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("com confirmacao, marca e mostra a ocupacao com buffer", async () => {
    mockAgentFetch.mockResolvedValueOnce({ appointment: AG } as never);
    const r = await montar();
    await preencher(r, { recurso: "r1", servico: "s1", inicio: "2026-11-05T10:00:00Z" });
    await confirmar(r);
    await clicar(r);

    expect(texto(r, "marcar-resultado")).toContain("ag-9");
    expect(texto(r, "marcar-ocupacao")).toContain("buffer de 15 min");
    expect(texto(r, "marcar-buffer")).toContain("10:45");
  });

  it("o pedido vai para /scheduling/book sem a organizacao", async () => {
    mockAgentFetch.mockResolvedValueOnce({ appointment: AG } as never);
    const r = await montar();
    await preencher(r, { recurso: "r1", servico: "s1", inicio: "2026-11-05T10:00:00Z", chave: "k-1" });
    await confirmar(r);
    await clicar(r);

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(caminho).toBe("/api/agent/v1/scheduling/book");
    expect(init.method).toBe("POST");

    const corpo = JSON.parse(init.body ?? "{}");
    expect(corpo.resource_id).toBe("r1");
    expect(corpo.starts_at).toBe("2026-11-05T10:00:00Z");
    expect(corpo.idempotency_key).toBe("k-1");
    // O escopo vem do middleware; um corpo que escolhe o tenant escolhe o tenant.
    expect(corpo.organization_id).toBeUndefined();
  });

  it("409 e' REENTREGA, nao falha -- e nao inventa o id", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("409 conflict: idempotency key already used"));
    const r = await montar();
    await preencher(r, { recurso: "r1", servico: "s1", inicio: "2026-11-05T10:00:00Z" });
    await confirmar(r);
    await clicar(r);

    const t = texto(r, "marcar-reentrega");
    expect(t).toContain("Reentrega detectada");
    expect(t).toContain("NÃO foi duplicado");
    expect(porTestId(r, "marcar-erro")).toHaveLength(0);
  });

  it("o 503 aparece como motor ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("motor de agendamento nao esta' ligado"));
    const r = await montar();
    await preencher(r, { recurso: "r1", servico: "s1", inicio: "2026-11-05T10:00:00Z" });
    await confirmar(r);
    await clicar(r);

    expect(texto(r, "marcar-ausente")).toContain("não está ligado");
  });

  it("a fronteira diz que o contrato e' separado e que o buffer existe", async () => {
    const r = await montar();

    const t = texto(r, "marcar-fronteira");
    expect(t).toContain("separado");
    expect(t).toContain("buffer");
  });
});
