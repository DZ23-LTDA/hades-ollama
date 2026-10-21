// Testes da remarcação de agendamento.
//
// Discriminantes: NÃO remarca ao montar; a recusa vem do MOTOR e é mostrada como recusa (não como erro
// do painel); o painel NÃO valida turno/sobreposição; remarcar não é cancelar+criar.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { RescheduleMutationPanel } from "@/components/RescheduleMutationPanel";
import {
  descreverFronteiraDaRemarcacao,
  descreverMovimento,
  descreverRemarcacao,
} from "@/lib/rescheduleMutationClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<RescheduleMutationPanel />);
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
    const el = porTestId(renderer, `remarcar-${id}`);
    await act(async () => {
      el[0].props.onChange({ target: { value: valor } });
    });
  }
}

async function confirmar(renderer: ReturnType<typeof create>) {
  const cs = porTestId(renderer, "remarcar-confirmar");
  await act(async () => {
    cs[0].props.onChange({ target: { checked: true } });
  });
}

async function clicar(renderer: ReturnType<typeof create>) {
  const bs = porTestId(renderer, "remarcar-enviar");
  await act(async () => {
    bs[0].props.onClick();
  });
}

const REMARCADO = {
  id: "ag-9",
  resource_id: "r1",
  service_id: "s1",
  starts_at: "2026-11-06T14:00:00Z",
  ends_at: "2026-11-06T14:30:00Z",
  busy_until: "2026-11-06T14:45:00Z",
  status: "confirmed",
  created_at: "2026-10-19T12:00:00Z",
};

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de remarcacao", () => {
  it("a fronteira diz que o MOTOR decide o que cabe", () => {
    const t = descreverFronteiraDaRemarcacao();
    expect(t).toContain("é o motor que decide");
    expect(t).toContain("não o painel");
  });

  it("a fronteira lista os tres criterios do motor", () => {
    const t = descreverFronteiraDaRemarcacao();
    expect(t).toContain("sobreposição");
    expect(t).toContain("fechamento");
    expect(t).toContain("turno");
  });

  it("o movimento mostra os dois instantes", () => {
    const t = descreverMovimento("2026-11-05T10:00:00Z", "2026-11-06T14:00:00Z");
    expect(t).toContain("de ");
    expect(t).toContain(" para ");
  });

  it("mover para o MESMO instante e' dito como manutencao", () => {
    const t = descreverMovimento("2026-11-05T10:00:00Z", "2026-11-05T10:00:00Z");
    expect(t).toContain("mantém");
    expect(t).toContain("mesmo");
  });

  it("sem instantes, nada a comparar", () => {
    expect(descreverMovimento("", "")).toContain("nada a comparar");
  });

  it("a remarcacao diz o novo instante e o estado", () => {
    const t = descreverRemarcacao({ appointment: REMARCADO } as never);
    expect(t).toContain("ag-9");
    expect(t).toContain("2026-11-06T14:00:00Z");
    expect(t).toContain("confirmed");
  });

  it("um estado nao declarado NAO vira confirmado", () => {
    const semEstado = { ...REMARCADO, status: "" };
    expect(descreverRemarcacao({ appointment: semEstado } as never)).toContain("não declarado");
  });
});

describe("RescheduleMutationPanel", () => {
  it("NAO remarca ao montar -- e' uma escrita", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "remarcar-pronto")).toContain("Nada remarcado");
  });

  it("sem confirmacao, recusa antes de chamar", async () => {
    const r = await montar();
    await preencher(r, { id: "ag-9", inicio: "2026-11-06T14:00:00Z" });
    await clicar(r);

    expect(texto(r, "remarcar-erro")).toContain("Confirme antes");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("com confirmacao mas sem id, recusa com o nome do campo", async () => {
    const r = await montar();
    await preencher(r, { inicio: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r);

    expect(texto(r, "remarcar-recusado")).toContain("appointment_id");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("inicio sem fuso e' recusado antes de chamar", async () => {
    const r = await montar();
    await preencher(r, { id: "ag-9", inicio: "2026-11-06T14:00:00" });
    await confirmar(r);
    await clicar(r);

    expect(texto(r, "remarcar-recusado")).toContain("RFC 3339");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("com dados validos, remarca e mostra o resultado", async () => {
    mockAgentFetch.mockResolvedValueOnce({ appointment: REMARCADO } as never);
    const r = await montar();
    await preencher(r, { id: "ag-9", inicio: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r);

    expect(texto(r, "remarcar-resultado")).toContain("ag-9");
    expect(texto(r, "remarcar-resultado")).toContain("2026-11-06T14:00:00Z");
  });

  it("o pedido e' POST para /scheduling/reschedule com os dois campos", async () => {
    mockAgentFetch.mockResolvedValueOnce({ appointment: REMARCADO } as never);
    const r = await montar();
    await preencher(r, { id: "ag-9", inicio: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r);

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(caminho).toBe("/api/agent/v1/scheduling/reschedule");
    expect(init.method).toBe("POST");

    const corpo = JSON.parse(init.body ?? "{}");
    expect(corpo.appointment_id).toBe("ag-9");
    expect(corpo.starts_at).toBe("2026-11-06T14:00:00Z");
    // Mover nao e' cancelar+criar: nao ha' campo de cancelamento nem de recurso/servico.
    expect(corpo.resource_id).toBeUndefined();
    expect(corpo.reason).toBeUndefined();
  });

  it("a RECUSA do motor aparece como recusa, nao como erro do painel", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("400: horario sobreposto com outro agendamento"));
    const r = await montar();
    await preencher(r, { id: "ag-9", inicio: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r);

    const t = texto(r, "remarcar-recusado");
    expect(t).toContain("O motor recusou");
    expect(t).toContain("sobreposto");
    expect(porTestId(r, "remarcar-erro")).toHaveLength(0);
  });

  it("o 503 aparece como motor ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("motor de agendamento nao esta' ligado"));
    const r = await montar();
    await preencher(r, { id: "ag-9", inicio: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r);

    expect(texto(r, "remarcar-ausente")).toContain("não está ligado");
  });

  it("o resultado diz que e' o MESMO agendamento, nao um novo", async () => {
    mockAgentFetch.mockResolvedValueOnce({ appointment: REMARCADO } as never);
    const r = await montar();
    await preencher(r, { id: "ag-9", inicio: "2026-11-06T14:00:00Z" });
    await confirmar(r);
    await clicar(r);

    const t = texto(r, "remarcar-identidade");
    expect(t).toContain("MESMO agendamento");
    expect(t).toContain("não um cancelamento");
  });

  it("a fronteira diz que a rota nao decide o que cabe", async () => {
    const r = await montar();

    const t = texto(r, "remarcar-fronteira");
    expect(t).toContain("é o motor que decide");
    expect(t).toContain("duplicaria o motor");
  });
});
