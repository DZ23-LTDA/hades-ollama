// Testes da fabrica de empresa.
//
// Discriminante central: `customers_created: 0` NAO e' "zero clientes" -- e' "esta fabrica
// nunca cria clientes". E o valor do estoque e' valor de TABELA, nao receita.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { CompanyFactoryPanel } from "@/components/CompanyFactoryPanel";
import {
  descreverClientesCriados,
  descreverValorDeCatalogo,
  dependeDeTerceiro,
  resumirChecklist,
  rotuloDoItem,
} from "@/lib/companyFactoryClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<CompanyFactoryPanel />);
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

const RELATORIO = {
  company_id: "c1", organization_id: "o1",
  launched_at: "2026-10-19T00:00:00Z",
  company: { id: "c1" },
  products: [{}, {}],
  artifacts: [{ id: "a1", kind: "plano", title: "Plano de negócio" }],
  checklist: [
    { id: "k1", label: "Registar CNPJ", status: "external" },
    { id: "k2", label: "Definir missão", status: "done" },
  ],
  catalog_value_cents: 250000,
  customers_created: 0,
};

function responder(r = RELATORIO) {
  mockAgentFetch.mockResolvedValueOnce(r as never);
}

/** Preenche o id e clica, em dois `act` separados. */
async function comId(renderer: ReturnType<typeof create>, valor: string) {
  const campos = porTestId(renderer, "fabrica-id");
  await act(async () => {
    campos[0].props.onChange({ target: { value: valor } });
  });
  const botoes = porTestId(renderer, "fabrica-carregar");
  await act(async () => {
    botoes[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes da fabrica", () => {
  it("clientes_created=0 diz o que o zero SIGNIFICA", () => {
    const t = descreverClientesCriados(0);
    expect(t).toContain("a fábrica não inventa clientes");
    expect(t).not.toBe("0");
  });

  it("sem valor declarado, diz que nao foi declarado", () => {
    expect(descreverClientesCriados(undefined)).toContain("não declarado");
  });

  it("valor de catalogo diz 'nao e receita'", () => {
    const t = descreverValorDeCatalogo(250000);
    expect(t).toContain("2.500,00");
    expect(t).toContain("não é receita");
    expect(t).toContain("valor de tabela");
  });

  it("valor zero nao e receita zero", () => {
    expect(descreverValorDeCatalogo(0)).toContain("sem valor de tabela declarado");
  });

  it("o checklist separa o que a fabrica fez do que depende de terceiro", () => {
    const t = resumirChecklist(RELATORIO.checklist as never);
    expect(t).toContain("1 feito(s) pela fábrica");
    expect(t).toContain("1 à espera de terceiro");
  });

  it("status 'external' conta como terceiro mesmo sem o booleano", () => {
    expect(dependeDeTerceiro({ status: "external" })).toBe(true);
    expect(dependeDeTerceiro({ status: "done" })).toBe(false);
    expect(dependeDeTerceiro({})).toBe(false);
  });

  it("item sem rotulo e' dito, nao inventado", () => {
    expect(rotuloDoItem({})).toBe("item sem rótulo");
    expect(rotuloDoItem({ title: "Definir missão" })).toBe("Definir missão");
  });
});

describe("CompanyFactoryPanel", () => {
  it("sem id, pede o id em vez de mostrar zero", async () => {
    const renderer = await montar();

    expect(texto(renderer, "fabrica-ausente")).toContain("Indique o id");
    expect(porTestId(renderer, "fabrica-clientes")).toHaveLength(0);
  });

  it("com id, mostra o zero como fronteira, nao como contagem", async () => {
    responder();
    const renderer = await montar();
    await comId(renderer, "c1");

    const t = texto(renderer, "fabrica-clientes");
    expect(t).toContain("a fábrica não inventa clientes");
  });

  it("o valor do estoque diz que nao e' receita", async () => {
    responder();
    const renderer = await montar();
    await comId(renderer, "c1");

    const t = texto(renderer, "fabrica-valor");
    expect(t).toContain("valor de tabela");
    expect(t).toContain("não é receita");
  });

  it("o item que depende de terceiro e' marcado", async () => {
    responder();
    const renderer = await montar();
    await comId(renderer, "c1");

    expect(texto(renderer, "fabrica-item-k1")).toContain("depende de terceiro");
    expect(texto(renderer, "fabrica-item-k2")).not.toContain("depende de terceiro");
  });

  it("a fronteira declara que a fabrica nao cria clientes nem emite registo", async () => {
    const renderer = await montar();

    const t = texto(renderer, "fabrica-fronteira");
    expect(t).toContain("Não cria clientes");
    expect(t).toContain("registro legal");
  });

  it("o caminho usa o id e o prefixo explicito", async () => {
    responder();
    const renderer = await montar();
    await comId(renderer, "c1");

    expect(mockAgentFetch.mock.calls[0][0]).toBe(
      "/api/agent/v1/company/factory/report/c1",
    );
  });

  it("o id e' codificado no caminho, nao concatenado cru", async () => {
    responder();
    const renderer = await montar();
    await comId(renderer, "a/b");

    expect(mockAgentFetch.mock.calls[0][0]).toBe(
      "/api/agent/v1/company/factory/report/a%2Fb",
    );
  });
});
