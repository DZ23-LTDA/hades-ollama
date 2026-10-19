// Testes dos horários oferecíveis.
//
// Discriminantes: data vazia NAO e' "hoje"; step tem teto; "0 slots" nao e' "dia livre";
// um slot com free=false numa lista de abertos e' contradicao mostrada, nao escondida.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { SlotsPanel } from "@/components/SlotsPanel";
import {
  contarContradicoes,
  descreverDuracoes,
  descreverHorarios,
  duracaoEmMinutos,
  validarData,
  validarPasso,
} from "@/lib/slotsClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<SlotsPanel />);
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

async function consultar(renderer: ReturnType<typeof create>, dia: string, passo = "30") {
  const ds = porTestId(renderer, "horarios-data");
  await act(async () => {
    ds[0].props.onChange({ target: { value: dia } });
  });
  const ps = porTestId(renderer, "horarios-passo");
  await act(async () => {
    ps[0].props.onChange({ target: { value: passo } });
  });
  const bs = porTestId(renderer, "horarios-consultar");
  await act(async () => {
    bs[0].props.onClick();
  });
}

const SLOTS = {
  slots: [
    { starts_at: "2026-11-02T09:00:00Z", ends_at: "2026-11-02T09:30:00Z", free: true },
    { starts_at: "2026-11-02T09:30:00Z", ends_at: "2026-11-02T10:00:00Z", free: true },
  ],
  count: 2,
};

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de horarios", () => {
  it("data vazia NAO e' hoje -- e' recusa", () => {
    const t = validarData("");
    expect(t).toContain("obrigatória");
    expect(t).toContain("não significa hoje");
  });

  it("data em formato errado e' recusada", () => {
    expect(validarData("02/11/2026")).toContain("AAAA-MM-DD");
    expect(validarData("2026-11-02")).toBeNull();
  });

  it("o passo tem teto de 240", () => {
    expect(validarPasso("241")).toContain("entre 1 e 240");
    expect(validarPasso("0")).toContain("entre 1 e 240");
    expect(validarPasso("240")).toBeNull();
    expect(validarPasso("")).toBeNull();
  });

  it("a duracao e' calculada do inicio e fim", () => {
    expect(duracaoEmMinutos(SLOTS.slots[0] as never)).toBe(30);
  });

  it("inicio depois do fim nao da' duracao negativa", () => {
    expect(
      duracaoEmMinutos({ starts_at: "2026-11-02T10:00:00Z", ends_at: "2026-11-02T09:00:00Z" } as never),
    ).toBeNull();
  });

  it("lista de abertos com free=false e' contradicao contada", () => {
    expect(
      contarContradicoes({
        slots: [{ starts_at: "x", ends_at: "y", free: false }],
        count: 1,
      } as never),
    ).toBe(1);
  });

  it("zero horarios NAO e' dia livre", () => {
    const t = descreverHorarios({ slots: [], count: 0 } as never);
    expect(t).toContain("não é o mesmo que o dia estar livre");
  });

  it("a promessa e' dita com as tres condicoes", () => {
    const t = descreverHorarios(SLOTS as never);
    expect(t).toContain("dentro do turno");
    expect(t).toContain("fora de fechamento");
    expect(t).toContain("livre de ocupação");
  });

  it("duracoes iguais sao ditas como uma so'", () => {
    const t = descreverDuracoes(SLOTS as never, 30);
    expect(t).toContain("todos os horários duram 30 min");
  });
});

describe("SlotsPanel", () => {
  it("sem data, pede a data em vez de assumir hoje", async () => {
    const r = await montar();
    await consultar(r, "");

    expect(texto(r, "horarios-invalido")).toContain("não significa hoje");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("passo acima do teto e' recusado antes de chamar", async () => {
    const r = await montar();
    await consultar(r, "2026-11-02", "241");

    expect(texto(r, "horarios-invalido")).toContain("entre 1 e 240");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("mostra a promessa completa dos horarios ofereciveis", async () => {
    mockAgentFetch.mockResolvedValueOnce(SLOTS as never);
    const r = await montar();
    await consultar(r, "2026-11-02");

    const t = texto(r, "horarios-resumo");
    expect(t).toContain("REALMENTE oferecível");
    expect(t).toContain("dentro do turno");
  });

  it("a data e o passo vao no caminho", async () => {
    mockAgentFetch.mockResolvedValueOnce(SLOTS as never);
    const r = await montar();
    await consultar(r, "2026-11-02", "60");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toContain("/api/agent/v1/scheduling/slots");
    expect(caminho).toContain("date=2026-11-02");
    expect(caminho).toContain("step=60");
  });

  it("a contradicao de free=false e' mostrada", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      slots: [
        { starts_at: "2026-11-02T09:00:00Z", ends_at: "2026-11-02T09:30:00Z", free: true },
        { starts_at: "2026-11-02T09:30:00Z", ends_at: "2026-11-02T10:00:00Z", free: false },
      ],
      count: 2,
    } as never);
    const r = await montar();
    await consultar(r, "2026-11-02");

    expect(texto(r, "horarios-contradicao")).toContain("declaram-se ocupados");
  });

  it("zero horarios aparece como vazio, nao como dia livre", async () => {
    mockAgentFetch.mockResolvedValueOnce({ slots: [], count: 0 } as never);
    const r = await montar();
    await consultar(r, "2026-11-02");

    expect(texto(r, "horarios-vazio")).toContain("não ofereceu nenhum horário");
    expect(porTestId(r, "horarios-erro")).toHaveLength(0);
  });

  it("o 503 aparece como motor ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("motor de agendamento nao esta' ligado"));
    const r = await montar();
    await consultar(r, "2026-11-02");

    expect(texto(r, "horarios-ausente")).toContain("não está ligado");
  });

  it("a fronteira distingue disponibilidade de horario oferecivel", async () => {
    const r = await montar();

    const t = texto(r, "horarios-fronteira");
    expect(t).toContain("dentro do turno");
    expect(t).toContain("não é o mesmo");
  });
});
