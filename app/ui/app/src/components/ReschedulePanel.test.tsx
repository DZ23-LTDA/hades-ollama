// Testes dos horários para remarcar.
//
// Discriminantes: os DOIS campos são obrigatórios e a falta é dita por nome; step tem tecto;
// "0 horários" não é sucesso; um slot que se declara ocupado numa lista de "livres" é mostrado.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { ReschedulePanel } from "@/components/ReschedulePanel";
import {
  contarNaoLivres,
  descreverAlvo,
  descreverDisponibilidade,
  descreverDuracoes,
  validarPasso,
  validarPedido,
} from "@/lib/rescheduleClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<ReschedulePanel />);
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

async function consultar(
  renderer: ReturnType<typeof create>,
  id: string,
  dia: string,
  passo = "30",
) {
  const is = porTestId(renderer, "remarcar-id");
  await act(async () => {
    is[0].props.onChange({ target: { value: id } });
  });
  const ds = porTestId(renderer, "remarcar-data");
  await act(async () => {
    ds[0].props.onChange({ target: { value: dia } });
  });
  const ps = porTestId(renderer, "remarcar-passo");
  await act(async () => {
    ps[0].props.onChange({ target: { value: passo } });
  });
  const bs = porTestId(renderer, "remarcar-consultar");
  await act(async () => {
    bs[0].props.onClick();
  });
}

const SLOTS = {
  slots: [
    { starts_at: "2026-11-03T09:00:00Z", ends_at: "2026-11-03T09:30:00Z", free: true },
    { starts_at: "2026-11-03T09:30:00Z", ends_at: "2026-11-03T10:00:00Z", free: true },
  ],
  count: 2,
};

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de remarcacao", () => {
  it("os DOIS campos em falta sao ditos por nome", () => {
    const t = validarPedido("", "");
    expect(t).toContain("agendamento");
    expect(t).toContain("data");
    expect(t).toContain("os dois são obrigatórios");
  });

  it("so' a data em falta e' dita -- nao as duas", () => {
    const t = validarPedido("ag-1", "");
    expect(t).toContain("a data");
    expect(t).not.toContain("o agendamento (");
  });

  it("so' o agendamento em falta e' dito", () => {
    const t = validarPedido("", "2026-11-03");
    expect(t).toContain("o agendamento");
    expect(t).not.toContain("a data (");
  });

  it("data em formato errado e' recusada", () => {
    expect(validarPedido("ag-1", "03/11/2026")).toContain("AAAA-MM-DD");
  });

  it("o passo tem tecto de 240", () => {
    expect(validarPasso("241")).toContain("entre 1 e 240");
    expect(validarPasso("240")).toBeNull();
  });

  it("zero horarios NAO e' sucesso -- e' resposta negativa do motor", () => {
    const t = descreverDisponibilidade({ slots: [], count: 0 } as never);
    expect(t).toContain("o motor respondeu antes");
  });

  it("a promessa e' dita como ANTES", () => {
    const t = descreverDisponibilidade(SLOTS as never);
    expect(t).toContain("ANTES de o cliente escolher");
  });

  it("um slot ocupado numa lista de livres e' contradicao", () => {
    expect(
      contarNaoLivres({ slots: [{ starts_at: "x", ends_at: "y", free: false }], count: 1 } as never),
    ).toBe(1);
  });

  it("duracoes iguais sao ditas como uma so'", () => {
    expect(descreverDuracoes(SLOTS as never)).toContain("todos os horários duram 30 min");
  });

  it("o alvo incompleto e' dito incompleto", () => {
    expect(descreverAlvo("", "")).toContain("alvo incompleto");
    expect(descreverAlvo("ag-1", "2026-11-03")).toContain("remarcar ag-1");
  });
});

describe("ReschedulePanel", () => {
  it("sem agendamento nem data, diz o que falta e nao chama", async () => {
    const r = await montar();
    await consultar(r, "", "");

    const t = texto(r, "remarcar-invalido");
    expect(t).toContain("os dois são obrigatórios");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("so' a data em falta e' apontada", async () => {
    const r = await montar();
    await consultar(r, "ag-1", "");

    expect(texto(r, "remarcar-invalido")).toContain("a data");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("passo acima do tecto e' recusado antes de chamar", async () => {
    const r = await montar();
    await consultar(r, "ag-1", "2026-11-03", "241");

    expect(texto(r, "remarcar-invalido")).toContain("entre 1 e 240");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("mostra os horarios livres com o alvo", async () => {
    mockAgentFetch.mockResolvedValueOnce(SLOTS as never);
    const r = await montar();
    await consultar(r, "ag-1", "2026-11-03");

    expect(texto(r, "remarcar-alvo")).toContain("remarcar ag-1");
    expect(texto(r, "remarcar-resumo")).toContain("ANTES de o cliente escolher");
  });

  it("os dois campos e o passo vao no caminho", async () => {
    mockAgentFetch.mockResolvedValueOnce(SLOTS as never);
    const r = await montar();
    await consultar(r, "ag-1", "2026-11-03", "60");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toContain("/api/agent/v1/scheduling/reschedule/slots");
    expect(caminho).toContain("appointment_id=ag-1");
    expect(caminho).toContain("date=2026-11-03");
    expect(caminho).toContain("step=60");
  });

  it("a contradicao de free=false e' mostrada", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      slots: [{ starts_at: "2026-11-03T09:00:00Z", ends_at: "2026-11-03T09:30:00Z", free: false }],
      count: 1,
    } as never);
    const r = await montar();
    await consultar(r, "ag-1", "2026-11-03");

    expect(texto(r, "remarcar-contradicao")).toContain("declaram-se ocupados");
  });

  it("zero horarios aparece como vazio, nao como sucesso", async () => {
    mockAgentFetch.mockResolvedValueOnce({ slots: [], count: 0 } as never);
    const r = await montar();
    await consultar(r, "ag-1", "2026-11-03");

    expect(texto(r, "remarcar-vazio")).toContain("nenhum horário livre");
  });

  it("o 503 aparece como motor ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("motor de agendamento nao esta' ligado"));
    const r = await montar();
    await consultar(r, "ag-1", "2026-11-03");

    expect(texto(r, "remarcar-ausente")).toContain("não está ligado");
  });

  it("a fronteira diz que a resposta vem antes", async () => {
    const r = await montar();

    const t = texto(r, "remarcar-fronteira");
    expect(t).toContain("adivinhar");
    expect(t).toContain("antes");
  });
});
