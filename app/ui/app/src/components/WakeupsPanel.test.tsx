// Testes do despertar de agentes.
//
// Discriminante central: `wakeups` (tudo o que acordou) e `enqueued` (o que vai correr) NAO
// sao a mesma lista -- e um despertar informativo nao vira trabalho.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { WakeupsPanel } from "@/components/WakeupsPanel";
import {
  descreverEscopo,
  eInformativo,
  idDoDespertar,
  motivoDoDespertar,
  resumirDespertares,
  resumirInformativo,
} from "@/lib/wakeupsClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<WakeupsPanel />);
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

const BASE = {
  wakeups: [
    { mission_id: "m1", reason: "missao pronta a executar" },
    { mission_id: "m2", reason: "aguarda aprovacao", informative: true },
  ],
  enqueued: ["m1"],
  failed: [],
  scope: "missoes visiveis para o chamador",
};

function responder(r: Record<string, unknown> = BASE) {
  mockAgentFetch.mockResolvedValueOnce(r as never);
}

async function despertar(renderer: ReturnType<typeof create>) {
  const botoes = porTestId(renderer, "wakeups-executar");
  await act(async () => {
    botoes[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de wakeups", () => {
  it("despertares e enfileirados sao contados separadamente", () => {
    const t = resumirDespertares(BASE as never);
    expect(t).toContain("2 desperto(s)");
    expect(t).toContain("1 enfileirado(s)");
  });

  it("falhas aparecem no resumo em maiusculas", () => {
    const t = resumirDespertares({ ...BASE, failed: ["m9"] } as never);
    expect(t).toContain("1 FALHA(S)");
  });

  it("sem nada, diz que ninguem precisou", () => {
    expect(resumirDespertares({} as never)).toContain("nenhuma missão precisou");
  });

  it("o informativo e' a DIFERENCA entre desperto e enfileirado", () => {
    const t = resumirInformativo(BASE as never);
    expect(t).toContain("1 despertar(es) informativo(s)");
    expect(t).toContain("NÃO foram enfileirados");
  });

  it("sem diferenca, nao inventa informativo", () => {
    expect(resumirInformativo({ wakeups: [{ id: "a" }], enqueued: ["a"] } as never)).toBeNull();
  });

  it("informativo e' lido de dois nomes possiveis", () => {
    expect(eInformativo({ informative: true })).toBe(true);
    expect(eInformativo({ informational: true })).toBe(true);
    expect(eInformativo({})).toBe(false);
  });

  it("id e motivo ausentes sao ditos, nao inventados", () => {
    expect(idDoDespertar({}, 0)).toBe("despertar-1");
    expect(motivoDoDespertar({})).toBe("motivo não declarado");
  });

  it("escopo ausente e' declarado como nao declarado", () => {
    expect(descreverEscopo({})).toContain("não declarou o escopo");
    expect(descreverEscopo({ scope: "org" })).toBe("escopo: org");
  });
});

describe("WakeupsPanel", () => {
  it("mostra as DUAS listas, nao so' uma", async () => {
    responder();
    const r = await montar();
    await despertar(r);

    expect(texto(r, "wakeups-lista")).toContain("m1");
    expect(texto(r, "wakeups-lista")).toContain("m2");
    expect(texto(r, "wakeups-enfileirados")).toContain("m1");
    // m2 NAO esta' nos enfileirados.
    expect(texto(r, "wakeups-enfileirados")).not.toContain("m2");
  });

  it("o despertar informativo e' marcado como tal", async () => {
    responder();
    const r = await montar();
    await despertar(r);

    expect(texto(r, "wakeups-item-m2")).toContain("informativo");
    expect(texto(r, "wakeups-item-m1")).not.toContain("informativo");
  });

  it("a diferenca informativo/enfileirado e' dita em texto", async () => {
    responder();
    const r = await montar();
    await despertar(r);

    const t = texto(r, "wakeups-informativo");
    expect(t).toContain("NÃO foram enfileirados");
    expect(t).toContain("decisão humana");
  });

  it("falha parcial NAO e' mostrada como sucesso", async () => {
    responder({ ...BASE, enqueued: [], failed: ["m1"] });
    const r = await montar();
    await despertar(r);

    // Com enqueued vazio e falhas, o painel mostra as falhas -- nao o resumo de sucesso.
    expect(texto(r, "wakeups-falhas")).toContain("m1");
  });

  it("o 500 e' nomeado como falha sem efeito, nao como erro de rede", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("500 internal server error"));
    const r = await montar();
    await despertar(r);

    expect(texto(r, "wakeups-sem-efeito")).toContain("não é sucesso parcial");
  });

  it("o escopo aparece, nao presumido global", async () => {
    responder();
    const r = await montar();
    await despertar(r);

    expect(texto(r, "wakeups-escopo")).toContain("missoes visiveis para o chamador");
  });

  it("sem nada, o painel diz que o servidor nao viu missoes", async () => {
    responder({ wakeups: [], enqueued: [], failed: [], scope: "org" });
    const r = await montar();
    await despertar(r);

    expect(texto(r, "wakeups-vazio")).toContain("nenhuma missão a despertar");
  });

  it("a chamada e' POST no prefixo explicito", async () => {
    responder();
    const r = await montar();
    await despertar(r);

    expect(mockAgentFetch.mock.calls[0][0]).toBe("/api/agent/v1/wakeups");
    const init = mockAgentFetch.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe("POST");
  });

  it("a fronteira declara a idempotencia e o informativo", async () => {
    const r = await montar();

    const t = texto(r, "wakeups-fronteira");
    expect(t).toContain("idempotente por missão");
    expect(t).toContain("informativo");
  });
});
