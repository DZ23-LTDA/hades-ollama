// Testes da comparacao de solucoes.
//
// Discriminantes: integra_automaticamente SEMPRE falso; empate NAO e' falha; nao_medidos
// e' "ninguem mediu" e nao "falhou"; medidos_parcialmente INFORMA e nao compara.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { SolutionsPanel } from "@/components/SolutionsPanel";
import {
  descreverIntegracao,
  descreverRanking,
  descreverTestes,
  descreverVencedor,
  nomeDoCriterio,
  resumirMedidosParcialmente,
  resumirNaoMedidos,
} from "@/lib/solutionsClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<SolutionsPanel />);
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
  candidatos: [
    { id: "wt-a", branch: "solucao/a", testes_passaram: 42, testes_falharam: 0 },
    { id: "wt-b", branch: "solucao/b", testes_passaram: 41, testes_falharam: 1 },
  ],
  ranking: ["wt-a", "wt-b"],
  vencedor: "wt-a",
  empate: false,
  rejeitados: [],
  nao_medidos: [],
  medidos_parcialmente: [],
  integra_automaticamente: false,
  exige_aprovacao_humana: true,
  explicacao: "wt-a ganhou por testes.",
  somente_leitura: true,
  avisos: [],
};

function responder(r: Record<string, unknown> = BASE) {
  mockAgentFetch.mockResolvedValueOnce(r as never);
}

async function comparar(renderer: ReturnType<typeof create>, json?: string) {
  if (json !== undefined) {
    const areas = porTestId(renderer, "solucoes-candidatos");
    await act(async () => {
      areas[0].props.onChange({ target: { value: json } });
    });
  }
  const botoes = porTestId(renderer, "solucoes-comparar");
  await act(async () => {
    botoes[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de comparacao", () => {
  it("empate NAO e' falha: e' decisao humana", () => {
    expect(descreverVencedor({ empate: true } as never)).toContain("decisão é humana");
  });

  it("vencedor vazio com rejeitados diz que ninguem passou", () => {
    const t = descreverVencedor({
      empate: false, vencedor: "", rejeitados: [{ id: "x" }],
    } as never);
    expect(t).toContain("nenhum candidato passou nos testes");
  });

  it("integra_automaticamente=false e' dito como 'NAO integra'", () => {
    expect(descreverIntegracao({ integra_automaticamente: false } as never)).toContain(
      "NÃO integra nada",
    );
  });

  it("se o servidor disser que integra, o painel NAO a executa", () => {
    const t = descreverIntegracao({ integra_automaticamente: true } as never);
    expect(t).toContain("NÃO a executa");
  });

  it("nao_medidos e' 'ninguem mediu', nao 'falhou'", () => {
    const t = resumirNaoMedidos({ nao_medidos: [{ criterio: "custo" }] } as never);
    expect(t).toContain("sem medição em ninguém");
    expect(t).toContain("não separaram");
  });

  it("medidos_parcialmente diz 'informa, nao compara'", () => {
    const t = resumirMedidosParcialmente({ medidos_parcialmente: [{ criterio: "latencia" }] } as never);
    expect(t).toContain("informa, não compara");
  });

  it("listas vazias devolvem null, nao texto a fingir", () => {
    expect(resumirNaoMedidos({ nao_medidos: [] } as never)).toBeNull();
    expect(resumirMedidosParcialmente({} as never)).toBeNull();
  });

  it("o ranking e' numerado na ordem, nao alfabetico", () => {
    expect(descreverRanking({ ranking: ["b", "a"] } as never)).toBe("1. b · 2. a");
  });

  it("testes falhados aparecem em maiusculas", () => {
    expect(descreverTestes({ testes_passaram: 1, testes_falharam: 2 } as never)).toContain("FALHARAM");
    expect(descreverTestes({ testes_passaram: 3, testes_falharam: 0 } as never)).toBe("3 passaram");
  });

  it("criterio sem nome e' dito, nao inventado", () => {
    expect(nomeDoCriterio({})).toBe("critério sem nome");
  });
});

describe("SolutionsPanel", () => {
  it("o vencedor recomendado aparece", async () => {
    responder();
    const r = await montar();
    await comparar(r);

    expect(texto(r, "solucoes-vencedor")).toContain("vencedor recomendado: wt-a");
  });

  it("somente_leitura e' lido do campo, nao inferido do POST", async () => {
    responder();
    const r = await montar();
    await comparar(r);

    expect(texto(r, "solucoes-somente-leitura")).toContain("nada foi alterado");
  });

  it("a integracao e' dita como NAO automatica", async () => {
    responder();
    const r = await montar();
    await comparar(r);

    expect(texto(r, "solucoes-integracao")).toContain("NÃO integra nada");
  });

  it("empate e' mostrado como decisao humana, nao como erro", async () => {
    responder({ ...BASE, empate: true, vencedor: "" });
    const r = await montar();
    await comparar(r);

    expect(texto(r, "solucoes-vencedor")).toContain("EMPATE");
    expect(porTestId(r, "solucoes-erro")).toHaveLength(0);
  });

  it("nao_medidos e medidos_parcialmente aparecem separados", async () => {
    responder({
      ...BASE,
      nao_medidos: [{ criterio: "energia" }],
      medidos_parcialmente: [{ criterio: "latencia" }],
    });
    const r = await montar();
    await comparar(r);

    expect(texto(r, "solucoes-nao-medidos")).toContain("sem medição em ninguém");
    expect(texto(r, "solucoes-parciais")).toContain("informa, não compara");
    expect(porTestId(r, "solucoes-tudo-medido")).toHaveLength(0);
  });

  it("sem criterios por medir, o painel diz que todos foram medidos", async () => {
    responder();
    const r = await montar();
    await comparar(r);

    expect(texto(r, "solucoes-tudo-medido")).toContain("Todos os critérios");
  });

  it("os rejeitados aparecem com motivo", async () => {
    responder({ ...BASE, rejeitados: [{ id: "wt-c", motivo: "testes falharam" }] });
    const r = await montar();
    await comparar(r);

    expect(texto(r, "solucoes-rejeitados")).toContain("wt-c: testes falharam");
  });

  it("JSON invalido e' recusado ANTES de chamar o servidor", async () => {
    const r = await montar();
    await comparar(r, "{ nao e json");

    expect(texto(r, "solucoes-invalida")).toContain("JSON");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("lista vazia e' recusada sem chamar o servidor", async () => {
    const r = await montar();
    await comparar(r, "[]");

    expect(texto(r, "solucoes-invalida")).toBeTruthy();
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("a chamada e' POST com o prefixo explicito e corpo JSON", async () => {
    responder();
    const r = await montar();
    await comparar(r);

    expect(mockAgentFetch.mock.calls[0][0]).toBe("/api/agent/v1/solutions/compare");
    const init = mockAgentFetch.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe("POST");
    expect(String(init.body)).toContain("candidatos");
  });

  it("a fronteira diz que recomenda e nao integra", async () => {
    const r = await montar();

    const t = texto(r, "solucoes-fronteira");
    expect(t).toContain("não altera nada");
    expect(t).toContain("nunca integra");
  });
});
