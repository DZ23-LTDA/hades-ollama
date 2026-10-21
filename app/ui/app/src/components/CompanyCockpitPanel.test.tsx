// Testes do cockpit da empresa.
//
// Discriminantes: a fábrica NÃO abre a empresa (entrega artefatos); 422 traz o RELATÓRIO junto e o painel
// mostra os dois; a prova de isolamento usa uma SEGUNDA organização; 201 na fábrica, 200/422 na jornada.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { CompanyCockpitPanel } from "@/components/CompanyCockpitPanel";
import {
  descreverArtefactos,
  descreverFronteiraDaFabrica,
  descreverIsolamento,
  resumirJornada,
  validarEmpresa,
} from "@/lib/companyCockpitClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<CompanyCockpitPanel />);
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
  const out: string[] = [];
  const visitar = (nodo: unknown) => {
    if (Array.isArray(nodo)) { nodo.forEach(visitar); return; }
    if (nodo === null || typeof nodo !== "object") return;
    const props = (nodo as { props?: Record<string, unknown> }).props;
    if (props && props["data-testid"] === id) out.push(juntarTexto(nodo));
    visitar((nodo as { children?: unknown }).children);
  };
  visitar(renderer.toJSON());
  return out.join(" ");
}

function porTestId(renderer: ReturnType<typeof create>, id: string) {
  return renderer.root.findAll((n) => n.props["data-testid"] === id);
}

async function escrever(renderer: ReturnType<typeof create>, id: string, valor: string) {
  await act(async () => {
    porTestId(renderer, `empresa-${id}`)[0].props.onChange({ target: { value: valor } });
  });
}

async function clicar(renderer: ReturnType<typeof create>, botao: string) {
  await act(async () => {
    porTestId(renderer, `empresa-${botao}`)[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes do cockpit", () => {
  it("nome vazio e' recusado com o nome do campo", () => {
    expect(validarEmpresa("  ", "BRL")).toContain("nome da empresa");
  });

  it("moeda tem de ter 3 letras", () => {
    expect(validarEmpresa("ACME", "REAL")).toContain("3 letras");
    expect(validarEmpresa("ACME", "BRL")).toBeNull();
  });

  it("moeda e' normalizada (minusculas aceites)", () => {
    expect(validarEmpresa("ACME", "brl")).toBeNull();
  });

  it("a fronteira da fabrica diz que ela NAO cria cliente nem emite registro legal", () => {
    const t = descreverFronteiraDaFabrica();
    expect(t).toContain("NÃO cria cliente");
    expect(t).toContain("nem emite registro legal");
    expect(t).toContain("materializa");
    // E diz que o botao «abrir empresa» entrega o prospecto.
    expect(t).toContain("prospecto");
  });

  it("a prova de isolamento precisa de uma SEGUNDA organizacao", () => {
    const t = descreverIsolamento();
    expect(t).toContain("SEGUNDA organização");
    expect(t).toContain("não prova isolamento");
    expect(t).toContain("prova identidade");
  });

  it("o resumo da jornada conta passos, isolamento e dependencias externas", () => {
    const t = resumirJornada({
      steps: [{ name: "a", ok: true }, { name: "b", ok: true }, { name: "c", ok: false }],
      isolation_proven: true,
      external_dependencies: ["stripe"],
    } as never);
    expect(t).toContain("3 etapa");
    expect(t).toContain("2 ok");
    expect(t).toContain("1 falhou");
    expect(t).toContain("isolamento PROVADO");
    expect(t).toContain("1 dependência");
  });

  it("isolamento nao provado e' dito, nao omitido", () => {
    const t = resumirJornada({ steps: [], isolation_proven: false } as never);
    expect(t).toContain("isolamento NÃO provado");
  });

  it("os artefactos da fabrica sao contados, nao inventados", () => {
    const t = descreverArtefactos({
      company_id: "emp-1",
      artifacts: [{ kind: "brand" }, { kind: "site" }],
      checklist: ["a", "b", "c"],
    } as never);
    expect(t).toContain("emp-1");
    expect(t).toContain("2 artefato");
    expect(t).toContain("3 item");
  });
});

describe("CompanyCockpitPanel", () => {
  it("nao corre ao montar", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "empresa-pronto")).toContain("Nada executado");
  });

  it("nome vazio e' recusado antes de chamar", async () => {
    const r = await montar();
    await clicar(r, "lancar");

    expect(texto(r, "empresa-recusada")).toContain("nome da empresa");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("lancar vai para /company/factory/launch com company.name e currency", async () => {
    mockAgentFetch.mockResolvedValueOnce({ company_id: "emp-1", artifacts: [{ kind: "brand" }], checklist: [] } as never);
    const r = await montar();
    await escrever(r, "nome", "ACME");
    await escrever(r, "moeda", "brl");
    await clicar(r, "lancar");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/company/factory/launch");
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(init.method).toBe("POST");
    const corpo = JSON.parse(init.body ?? "{}");
    expect(corpo.company.name).toBe("ACME");
    // Moeda normalizada para maiusculas.
    expect(corpo.company.currency).toBe("BRL");
  });

  it("o resultado diz 201 e quantos artefactos -- e a fronteira", async () => {
    mockAgentFetch.mockResolvedValueOnce({ company_id: "emp-1", artifacts: [{ kind: "brand" }], checklist: ["a"] } as never);
    const r = await montar();
    await escrever(r, "nome", "ACME");
    await clicar(r, "lancar");

    const t = texto(r, "empresa-lancada");
    expect(t).toContain("201");
    expect(t).toContain("1 artefato");
    // A fronteira aparece: a fabrica nao abriu a empresa.
    expect(texto(r, "empresa-fronteira")).toContain("NÃO cria cliente");
  });

  it("certificar vai para /business/certify com a SEGUNDA organizacao quando dada", async () => {
    mockAgentFetch.mockResolvedValueOnce({ steps: [{ ok: true }], isolation_proven: true, external_dependencies: [] } as never);
    const r = await montar();
    await escrever(r, "outra-org", "org-2");
    await clicar(r, "certificar");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/business/certify");
    const corpo = JSON.parse((mockAgentFetch.mock.calls[0][1] as { body?: string }).body ?? "{}");
    expect(corpo.other_organization_id).toBe("org-2");
  });

  it("422 e' JORNADA QUEBRADA, nao erro do pedido -- e mostra o relatorio", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("422: journey step 3 failed"));
    const r = await montar();
    await clicar(r, "certificar");

    const t = texto(r, "empresa-quebrada");
    expect(t).toContain("422");
    expect(t).toContain("jornada quebrou");
    // O relatorio veio junto: o painel diz isso.
    expect(texto(r, "empresa-quebrada-relatorio")).toContain("veio COM a falha");
    expect(porTestId(r, "empresa-erro")).toHaveLength(0);
  });

  it("o isolamento e' explicado no resultado da certificacao", async () => {
    mockAgentFetch.mockResolvedValueOnce({ steps: [{ ok: true }], isolation_proven: true } as never);
    const r = await montar();
    await clicar(r, "certificar");

    expect(texto(r, "empresa-isolamento")).toContain("SEGUNDA organização");
  });

  it("503 aparece como runtime da empresa ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("503: company runtime missing"));
    const r = await montar();
    await escrever(r, "nome", "ACME");
    await clicar(r, "lancar");

    expect(texto(r, "empresa-ausente")).toContain("não está ligado");
  });
});
