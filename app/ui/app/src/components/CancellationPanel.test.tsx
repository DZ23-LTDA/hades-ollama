// Testes do cancelamento de agendamento.
//
// Discriminantes: NÃO cancela ao montar; o MOTIVO é obrigatório (não é conveniência, é auditabilidade);
// o id vai no CORPO, não no caminho; a resposta é 200 e mostra o estado que o servidor devolveu.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { CancellationPanel } from "@/components/CancellationPanel";
import {
  descreverCancelamento,
  descreverOQueSePerde,
  validarIdAgendamento,
  validarMotivo,
} from "@/lib/cancellationClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<CancellationPanel />);
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
    const el = porTestId(renderer, `cancelar-${id}`);
    await act(async () => {
      el[0].props.onChange({ target: { value: valor } });
    });
  }
}

async function confirmar(renderer: ReturnType<typeof create>) {
  const cs = porTestId(renderer, "cancelar-confirmar");
  await act(async () => {
    cs[0].props.onChange({ target: { checked: true } });
  });
}

async function clicar(renderer: ReturnType<typeof create>) {
  const bs = porTestId(renderer, "cancelar-enviar");
  await act(async () => {
    bs[0].props.onClick();
  });
}

const CANCELADO = {
  id: "ag-9",
  resource_id: "r1",
  service_id: "s1",
  starts_at: "2026-11-05T10:00:00Z",
  ends_at: "2026-11-05T10:30:00Z",
  busy_until: "2026-11-05T10:45:00Z",
  status: "cancelled",
  created_at: "2026-10-19T12:00:00Z",
};

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de cancelamento", () => {
  it("id vazio e' recusado", () => {
    expect(validarIdAgendamento("")).toContain("appointment_id");
    expect(validarIdAgendamento("ag-1")).toBeNull();
  });

  it("motivo vazio e' recusado -- porque nao e' auditavel", () => {
    const t = validarMotivo("");
    expect(t).toContain("obrigatório");
    expect(t).toContain("auditável");
  });

  it("motivo so' com espacos tambem e' recusado", () => {
    expect(validarMotivo("   ")).toContain("auditável");
    expect(validarMotivo("cliente pediu")).toBeNull();
  });

  it("o estado mostrado e' o que o servidor devolveu", () => {
    const t = descreverCancelamento({ appointment: CANCELADO } as never);
    expect(t).toContain("ag-9");
    expect(t).toContain("cancelled");
  });

  it("um estado nao declarado NAO vira 'cancelado'", () => {
    const semEstado = { ...CANCELADO, status: "" };
    expect(descreverCancelamento({ appointment: semEstado } as never)).toContain("não declarado");
  });

  it("sem agendamento carregado, diz que so' actua pelo id", () => {
    expect(descreverOQueSePerde(null, "")).toContain("só pelo identificador");
  });

  it("com motivo, o registo diz o motivo", () => {
    const t = descreverOQueSePerde(CANCELADO as never, "cliente pediu");
    expect(t).toContain("liberta");
    expect(t).toContain("10:45");
    expect(t).toContain("motivo registado: cliente pediu");
  });

  it("sem motivo carregado, avisa que o registo nao vai dizer porque", () => {
    const t = descreverOQueSePerde(CANCELADO as never, "");
    expect(t).toContain("não vai dizer porque");
  });
});

describe("CancellationPanel", () => {
  it("NAO cancela ao montar -- e' uma escrita destrutiva", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "cancelar-pronto")).toContain("destrutiva");
  });

  it("sem id, recusa antes de chamar", async () => {
    const r = await montar();
    await preencher(r, { motivo: "cliente pediu" });
    await clicar(r);

    expect(texto(r, "cancelar-invalido")).toContain("appointment_id");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("SEM MOTIVO, recusa antes de chamar -- o motivo nao e' opcional", async () => {
    const r = await montar();
    await preencher(r, { id: "ag-9" });
    await clicar(r);

    expect(texto(r, "cancelar-invalido")).toContain("auditável");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("sem confirmacao, recusa cancelar", async () => {
    const r = await montar();
    await preencher(r, { id: "ag-9", motivo: "cliente pediu" });
    await clicar(r);

    expect(texto(r, "cancelar-erro")).toContain("Confirme antes");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("com id, motivo e confirmacao, cancela e mostra o estado", async () => {
    mockAgentFetch.mockResolvedValueOnce({ appointment: CANCELADO } as never);
    const r = await montar();
    await preencher(r, { id: "ag-9", motivo: "cliente pediu" });
    await confirmar(r);
    await clicar(r);

    expect(texto(r, "cancelar-resultado")).toContain("ag-9");
    expect(texto(r, "cancelar-resultado")).toContain("cancelled");
  });

  it("o id vai no CORPO, nao no caminho", async () => {
    mockAgentFetch.mockResolvedValueOnce({ appointment: CANCELADO } as never);
    const r = await montar();
    await preencher(r, { id: "ag-9", motivo: "cliente pediu" });
    await confirmar(r);
    await clicar(r);

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    // O caminho NAO contem o id: um id concatenado num caminho e' um id que o id pode reescrever.
    expect(caminho).toBe("/api/agent/v1/scheduling/cancel");
    expect(caminho).not.toContain("ag-9");
    expect(init.method).toBe("POST");

    const corpo = JSON.parse(init.body ?? "{}");
    expect(corpo.appointment_id).toBe("ag-9");
    expect(corpo.reason).toBe("cliente pediu");
  });

  it("o 503 aparece como motor ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("motor de agendamento nao esta' ligado"));
    const r = await montar();
    await preencher(r, { id: "ag-9", motivo: "cliente pediu" });
    await confirmar(r);
    await clicar(r);

    expect(texto(r, "cancelar-ausente")).toContain("não está ligado");
  });

  it("um 400 de motivo aparece como invalido, nao como falha generica", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("400: motivo obrigatorio"));
    const r = await montar();
    await preencher(r, { id: "ag-9", motivo: "cliente pediu" });
    await confirmar(r);
    await clicar(r);

    expect(texto(r, "cancelar-invalido")).toContain("motivo");
    expect(porTestId(r, "cancelar-erro")).toHaveLength(0);
  });

  it("a fronteira diz porque o motivo e' obrigatorio e onde vai o id", async () => {
    const r = await montar();

    const t = texto(r, "cancelar-fronteira");
    expect(t).toContain("não é auditável");
    expect(t).toContain("corpo");
  });
});
