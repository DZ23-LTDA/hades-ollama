// Testes do painel de acoes de negocio.
//
// Discriminante central: BLOQUEADA_ORCAMENTO e AGUARDANDO_APROVACAO sao ambas
// "pendentes" e NAO sao a mesma coisa -- uma precisa de dinheiro, a outra de uma pessoa.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { BusinessActionsPanel } from "@/components/BusinessActionsPanel";
import {
  aguardaHumano,
  bloqueadaPorOrcamento,
  descreverCusto,
  descreverEstadoDaAcao,
  descreverRisco,
  resumirPendentes,
} from "@/lib/businessClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<BusinessActionsPanel />);
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
  const arvore = renderer.toJSON();
  const encontrados: string[] = [];
  const visitar = (nodo: unknown) => {
    if (Array.isArray(nodo)) { nodo.forEach(visitar); return; }
    if (nodo === null || typeof nodo !== "object") return;
    const props = (nodo as { props?: Record<string, unknown> }).props;
    if (props && props["data-testid"] === id) encontrados.push(juntarTexto(nodo));
    visitar((nodo as { children?: unknown }).children);
  };
  visitar(arvore);
  return encontrados.join(" ");
}

function porTestId(renderer: ReturnType<typeof create>, id: string) {
  return renderer.root.findAll((nodo) => nodo.props["data-testid"] === id);
}

const ESPERA = {
  id: "a1", organization_id: "o1", kind: "repor_estoque", subject: "SKU-77",
  reason: "estoque abaixo do minimo", risk: "medio", cost_cents: 125000,
  status: "AGUARDANDO_APROVACAO", created_at: "2026-10-19T00:00:00Z",
};

const ORCAMENTO = {
  ...ESPERA, id: "a2", kind: "estornar_pagamento", subject: "PED-9",
  status: "BLOQUEADA_ORCAMENTO", cost_cents: 50000,
};

const AUTONOMA = {
  ...ESPERA, id: "a3", kind: "atribuir_fila_humana", subject: "fila-3",
  status: "AUTONOMA", cost_cents: 0,
};

function responder(acoes = [ESPERA]) {
  mockAgentFetch.mockResolvedValueOnce({
    pending: acoes, count: acoes.length, policy: {},
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de negocio", () => {
  it("orcamento e aprovacao NAO sao o mesmo estado", () => {
    expect(bloqueadaPorOrcamento(ORCAMENTO as never)).toBe(true);
    expect(aguardaHumano(ORCAMENTO as never)).toBe(false);
    expect(bloqueadaPorOrcamento(ESPERA as never)).toBe(false);
    expect(aguardaHumano(ESPERA as never)).toBe(true);
  });

  it("o estado de orcamento diz que aprovar nao resolve", () => {
    expect(descreverEstadoDaAcao("BLOQUEADA_ORCAMENTO")).toContain("aprovar não resolve");
    expect(descreverEstadoDaAcao("AGUARDANDO_APROVACAO")).toContain("aguardando aprovação");
    expect(descreverEstadoDaAcao("AUTONOMA")).toContain("sem humano");
  });

  it("custo zero e' um numero, nao 'sem custo'", () => {
    expect(descreverCusto(0)).toBe("sem custo declarado");
    expect(descreverCusto(125000)).toContain("1.250,00");
  });

  it("risco ausente e' dito como nao declarado", () => {
    expect(descreverRisco({ risk: "" } as never)).toContain("não declarado");
  });

  it("o resumo separa as duas pendencias", () => {
    const t = resumirPendentes([ESPERA, ORCAMENTO, AUTONOMA] as never);
    expect(t).toContain("1 à espera de uma pessoa");
    expect(t).toContain("1 à espera de orçamento");
    expect(t).toContain("1 noutro estado");
  });
});

describe("BusinessActionsPanel", () => {
  it("a acao bloqueada por orcamento e' mostrada como tal", async () => {
    responder([ORCAMENTO] as never);

    const renderer = await montar();

    const t = texto(renderer, "negocio-estado-a2");
    expect(t).toContain("BLOQUEADA POR ORÇAMENTO");
    expect(t).toContain("aprovar não resolve");
  });

  it("o custo aparece ao lado da acao", async () => {
    responder();

    const renderer = await montar();

    expect(texto(renderer, "negocio-custo-a1")).toContain("1.250,00");
    expect(texto(renderer, "negocio-custo-a1")).toContain("risco: medio");
  });

  it("o resumo mostra as duas pendencias separadas", async () => {
    responder([ESPERA, ORCAMENTO] as never);

    const renderer = await montar();

    const t = texto(renderer, "negocio-resumo");
    expect(t).toContain("1 à espera de uma pessoa");
    expect(t).toContain("1 à espera de orçamento");
  });

  it("lista vazia com supervisor ligado diz isso", async () => {
    responder([]);

    const renderer = await montar();

    expect(texto(renderer, "negocio-sem-acoes")).toContain("não há ações à espera");
  });

  it("o 503 aparece como supervisor AUSENTE, nao como zero acoes", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("business supervisor nao esta' ligado"));

    const renderer = await montar();

    const sem = texto(renderer, "negocio-sem-supervisor");
    expect(sem).toContain("não está ligado");
    expect(sem).toContain("não é «não há ações pendentes»");
    expect(porTestId(renderer, "negocio-erro")).toHaveLength(0);
  });

  it("a chamada usa o prefixo explicito e e' GET", async () => {
    responder();

    await montar();

    expect(mockAgentFetch.mock.calls).toHaveLength(1);
    expect(mockAgentFetch.mock.calls[0][0]).toBe("/api/agent/v1/business/actions");
    const init = mockAgentFetch.mock.calls[0][1] as RequestInit | undefined;
    expect(init?.method ?? "GET").toBe("GET");
  });
});
