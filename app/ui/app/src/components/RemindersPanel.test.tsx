// Testes dos lembretes devidos.
//
// Discriminantes: `now` NAO e' enviado; `kind` vazio nao e' "todos"; esta rota e' SO' a metade
// do "quem deve ser avisado" -- nao o registo do que ja' foi enviado.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { RemindersPanel } from "@/components/RemindersPanel";
import {
  descreverJanela,
  descreverLembrete,
  descreverMetade,
  validarAntecedencia,
  validarTipo,
} from "@/lib/remindersClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<RemindersPanel />);
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

async function consultar(renderer: ReturnType<typeof create>, tipo: string, lead = "") {
  const ks = porTestId(renderer, "lembretes-kind");
  await act(async () => {
    ks[0].props.onChange({ target: { value: tipo } });
  });
  const ls = porTestId(renderer, "lembretes-lead");
  await act(async () => {
    ls[0].props.onChange({ target: { value: lead } });
  });
  const bs = porTestId(renderer, "lembretes-consultar");
  await act(async () => {
    bs[0].props.onClick();
  });
}

const RESPOSTA = {
  appointments: [{ appointment_id: "ag-1", kind: "24h", starts_at: "2026-10-20T10:00:00Z" }],
  count: 1,
  kind: "24h",
  lead_minutes: 1440,
};

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de lembretes", () => {
  it("kind vazio NAO e' 'todos' -- e' um pedido recusado", () => {
    expect(validarTipo("")).toContain("obrigatório");
    expect(validarTipo("24h")).toBeNull();
  });

  it("lead_minutes fora do intervalo e' recusado", () => {
    expect(validarAntecedencia("0")).toContain("entre 1 e 100000");
    expect(validarAntecedencia("100001")).toContain("entre 1 e 100000");
    expect(validarAntecedencia("1440")).toBeNull();
    expect(validarAntecedencia("")).toBeNull();
  });

  it("a metade do 'quem deve ser avisado' e' dita como metade", () => {
    const t = descreverMetade(RESPOSTA as never);
    expect(t).toContain("precisam de lembrete agora");
    expect(t).toContain("noutra rota");
  });

  it("lista vazia NAO diz que os anteriores foram enviados", () => {
    const t = descreverMetade({ appointments: [], count: 0 } as never);
    expect(t).toContain("NÃO diz que os anteriores foram enviados");
  });

  it("a janela diz tipo e antecedencia", () => {
    const t = descreverJanela(RESPOSTA as never);
    expect(t).toContain("tipo: 24h");
    expect(t).toContain("1440 min");
  });

  it("sem janela declarada, nao inventa", () => {
    expect(descreverJanela({} as never)).toContain("não declarou");
  });

  it("um lembrete mostra o agendamento e quando comeca", () => {
    const t = descreverLembrete(RESPOSTA.appointments[0] as never);
    expect(t).toContain("ag-1");
    expect(t).toContain("começa");
  });
});

describe("RemindersPanel", () => {
  it("mostra a metade pendente, nao o registo", async () => {
    mockAgentFetch.mockResolvedValueOnce(RESPOSTA as never);
    const r = await montar();
    await consultar(r, "24h");

    expect(texto(r, "lembretes-metade")).toContain("noutra rota");
  });

  it("a lista mostra os agendamentos devidos", async () => {
    mockAgentFetch.mockResolvedValueOnce(RESPOSTA as never);
    const r = await montar();
    await consultar(r, "24h");

    expect(texto(r, "lembrete-ag-1")).toContain("ag-1");
  });

  it("kind vazio e' recusado ANTES de chamar o servidor", async () => {
    const r = await montar();
    await consultar(r, "");

    expect(texto(r, "lembretes-invalido")).toContain("obrigatório");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("lead invalido e' recusado antes de chamar", async () => {
    const r = await montar();
    await consultar(r, "24h", "0");

    expect(texto(r, "lembretes-invalido")).toContain("entre 1 e 100000");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("o pedido NAO envia 'now'", async () => {
    mockAgentFetch.mockResolvedValueOnce(RESPOSTA as never);
    const r = await montar();
    await consultar(r, "24h");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toContain("/api/agent/v1/scheduling/reminders/due");
    expect(caminho).toContain("kind=24h");
    expect(caminho).not.toContain("now");
  });

  it("o 503 aparece como motor ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("motor de agendamento nao esta' ligado"));
    const r = await montar();
    await consultar(r, "24h");

    expect(texto(r, "lembretes-ausente")).toContain("não está ligado");
    expect(porTestId(r, "lembretes-erro")).toHaveLength(0);
  });

  it("a fronteira diz que e' so' uma metade", async () => {
    const r = await montar();

    const t = texto(r, "lembretes-fronteira");
    expect(t).toContain("não é o registo");
    expect(t).toContain("não o escolhe");
  });
});
