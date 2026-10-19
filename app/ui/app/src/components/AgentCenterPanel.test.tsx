// Testes do Agent Center.
//
// Discriminantes: o ledger NAO e' apresentado como catalogo; o drift mostra o estado E a accao do SERVIDOR;
// o destino relativo e' validado; 201 e 200 sao estados distintos; 503 nao e' "nenhum agente".
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { AgentCenterPanel } from "@/components/AgentCenterPanel";
import {
  descreverCatalogo,
  descreverDrift,
  resumoLegivel,
  validarDestino,
} from "@/lib/agentCenterClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<AgentCenterPanel />);
  });
  return renderer;
}

function juntarTexto(nodo: unknown): string {
  if (typeof nodo === "string") return nodo;
  if (typeof nodo === "number") return String(nodo);
  if (Array.isArray(nodo)) return nodo.map(juntarTexto).join(" ");
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

async function clicar(renderer: ReturnType<typeof create>, botao: string) {
  await act(async () => {
    porTestId(renderer, `agent-center-${botao}`)[0].props.onClick();
  });
}

async function escreverDestino(renderer: ReturnType<typeof create>, valor: string) {
  await act(async () => {
    porTestId(renderer, "agent-center-destino")[0].props.onChange({ target: { value: valor } });
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes do Agent Center", () => {
  it("destino vazio e' recusado", () => {
    expect(validarDestino("")).toContain("falta o destino");
  });

  it("destino absoluto e' recusado", () => {
    expect(validarDestino("/etc/passwd")).toContain("relativo");
    expect(validarDestino("C:\\Windows\\x")).toContain("relativo");
  });

  it("destino com `..` e' recusado: nao sai do workspace", () => {
    expect(validarDestino("../../fora.md")).toContain("sair do workspace");
  });

  it("destino relativo simples passa", () => {
    expect(validarDestino(".hades/agents/a.md")).toBeNull();
  });

  // *Uma lista que so' mostra o que ja' se instalou nao e' um catalogo: e' um historico.*
  it("o catalogo NAO e' apresentado como ligado quando o servidor diz que nao", () => {
    expect(descreverCatalogo({ catalog_available: false, catalog_note: "só instalações" })).toBe("só instalações");
    expect(descreverCatalogo({ catalog_available: false })).toContain("não está ligado");
    expect(descreverCatalogo({ catalog_available: true })).toBe("catálogo ligado");
  });

  // *Um painel que calcula o seu proprio estado mostra o que o programador achou, nao o que aconteceu.*
  it("o drift mostra o estado E a accao, ambos do servidor", () => {
    const t = descreverDrift({
      entry: { persona_id: "x" } as never,
      state: "LOCALLY_MODIFIED",
      action: "Editado fora do HADES. Preserve a edicao.",
    });
    expect(t).toContain("LOCALLY_MODIFIED");
    expect(t).toContain("Preserve");
  });

  it("o resumo mostra as 7 chaves, incluindo os zeros", () => {
    const t = resumoLegivel({ CURRENT: 2 });
    expect(t).toContain("CURRENT 2");
    expect(t).toContain("REMOVED 0");
    expect(t).toContain("FOREIGN_UNMANAGED 0");
  });
});

describe("AgentCenterPanel", () => {
  it("nao consulta ao montar e repete a nota do servidor", async () => {
    const r = await montar();
    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "agent-center-pronto")).toContain("Nada consultado");
    // Sem resposta ainda, a nota nao afirma catalogo.
    expect(texto(r, "agent-center-catalogo")).not.toContain("ligado");
  });

  it("listar vai para /agents", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      agents: [{ persona_id: "rev", version: "1.0.0", target: "hades", destination: ".h/x.md" }],
      count: 1,
      catalog_available: false,
      catalog_note: "o catálogo ainda não está ligado",
    } as never);
    const r = await montar();
    await clicar(r, "listar");

    expect(mockAgentFetch.mock.calls[0][0]).toBe("/api/agent/v1/agents");
    expect(texto(r, "agent-center-contagem")).toContain("instalação");
    expect(texto(r, "agent-center-lista")).toContain("rev");
    // E a nota do servidor aparece, nao uma inventada.
    expect(texto(r, "agent-center-catalogo")).toContain("ainda não está ligado");
  });

  it("integridade vai para /agents/drift", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      items: [
        { entry: { persona_id: "rev", destination: ".h/x.md" }, state: "CURRENT", action: "Actual: nada a fazer." },
      ],
      summary: { CURRENT: 1 },
    } as never);
    const r = await montar();
    await clicar(r, "drift");

    expect(mockAgentFetch.mock.calls[0][0]).toBe("/api/agent/v1/agents/drift");
    expect(texto(r, "agent-center-resumo")).toContain("CURRENT 1");
    expect(texto(r, "agent-center-itens")).toContain("nada a fazer");
  });

  it("instalar 201 diz CRIADO", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      entry: { persona_id: "revisor-exemplo" },
      created: true,
      bytes: 123,
    } as never);
    const r = await montar();
    await clicar(r, "instalar");

    expect(mockAgentFetch.mock.calls[0][0]).toBe("/api/agent/v1/agents/install");
    const corpo = JSON.parse((mockAgentFetch.mock.calls[0][1] as { body?: string }).body ?? "{}");
    expect(corpo.target).toBe("hades");
    expect(corpo.destination).toBe(".hades/agents/revisor-exemplo.md");
    expect(corpo.persona.id).toBe("revisor-exemplo");
    expect(texto(r, "agent-center-instalado")).toContain("201 criado");
  });

  it("retry (created=false) mostra 200 JA' REGISTADO, nao 201", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      entry: { persona_id: "revisor-exemplo" },
      created: false,
      bytes: 123,
    } as never);
    const r = await montar();
    await clicar(r, "instalar");

    const t = texto(r, "agent-center-instalado");
    expect(t).toContain("200");
    expect(t).toContain("já registado");
    expect(t).not.toContain("201");
  });

  it("destino com `..` e' recusado antes de chamar", async () => {
    const r = await montar();
    await escreverDestino(r, "../fora.md");
    await clicar(r, "instalar");

    expect(texto(r, "agent-center-recusado")).toContain("sair do workspace");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  // *Uma lista vazia e um erro nao sao a mesma coisa.*
  it("503 aparece como centro ausente, nao como lista vazia", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("503: agent persona ledger is not configured"));
    const r = await montar();
    await clicar(r, "listar");

    const t = texto(r, "agent-center-ausente");
    expect(t).toContain("não é «nenhum agente»");
    expect(porTestId(r, "agent-center-contagem")).toHaveLength(0);
  });

  it("409 aparece como conflito, nao como recusa generica", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("409: already recorded with different hashes"));
    const r = await montar();
    await clicar(r, "instalar");

    expect(texto(r, "agent-center-conflito")).toContain("conflito");
    expect(porTestId(r, "agent-center-recusado")).toHaveLength(0);
  });
});
