// Testes da fase de oportunidade.
//
// Discriminantes: fase terminal NAO reabre (recusa local + recusa do servidor, ambas como REGRA);
// fase inventada e' recusada; o id vai no caminho e e' codificado; o corpo so' leva a fase.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { CrmStagePanel } from "@/components/CrmStagePanel";
import {
  descreverMudanca,
  descreverRegraTerminal,
  faseTerminal,
  FASES,
  validarFase,
  validarOportunidade,
} from "@/lib/crmStageClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<CrmStagePanel />);
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

async function preencher(renderer: ReturnType<typeof create>, campos: Record<string, string>) {
  for (const [id, valor] of Object.entries(campos)) {
    await act(async () => {
      porTestId(renderer, `crm-fase-${id}`)[0].props.onChange({ target: { value: valor } });
    });
  }
}

async function clicar(renderer: ReturnType<typeof create>) {
  await act(async () => {
    porTestId(renderer, "crm-fase-mover")[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes da fase", () => {
  it("sao cinco fases e duas sao terminais", () => {
    expect([...FASES]).toEqual(["new", "qualified", "proposal", "won", "lost"]);
    expect(faseTerminal("won")).toBe(true);
    expect(faseTerminal("lost")).toBe(true);
    expect(faseTerminal("proposal")).toBe(false);
    expect(faseTerminal("")).toBe(false);
  });

  it("fase inventada e' recusada, e a mensagem lista as validas", () => {
    const t = validarFase("negociacao");
    expect(t).toContain("não existe");
    expect(t).toContain("qualified");
  });

  it("fase e' normalizada para minusculas", () => {
    expect(validarFase("WON")).toBeNull();
  });

  it("id vazio e' recusado com o nome do campo", () => {
    expect(validarOportunidade("   ")).toContain("id da oportunidade");
  });

  it("a regra terminal e' explicada como regra, nao como avaria", () => {
    const t = descreverRegraTerminal();
    expect(t).toContain("reabrir apagaria o resultado");
    expect(t).toContain("fechada");
    // E diz que o painel recusa ANTES: a mesma regra, nao uma segunda.
    expect(t).toContain("recusa antes de chamar");
  });

  it("o resultado diz a oportunidade e a fase", () => {
    const t = descreverMudanca({ id: "op-1", stage: "proposal" } as never);
    expect(t).toContain("op-1");
    expect(t).toContain("proposal");
  });
});

describe("CrmStagePanel", () => {
  it("nao move ao montar", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "crm-fase-pronto")).toContain("Nenhuma oportunidade");
  });

  it("id vazio e' recusado antes de chamar", async () => {
    const r = await montar();
    await clicar(r);

    expect(texto(r, "crm-fase-recusada")).toContain("id da oportunidade");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("fase terminal e' RECUSADA sem chamar o servidor", async () => {
    const r = await montar();
    await preencher(r, { id: "op-1", actual: "won" });
    await clicar(r);

    const t = texto(r, "crm-fase-terminal");
    expect(t).toContain("won");
    expect(t).toContain("reabrir apagaria o resultado");
    // A recusa e' local: a mesma regra, dita antes da rede.
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("vai para /crm/opportunities/:id/stage e o corpo so' leva a fase", async () => {
    mockAgentFetch.mockResolvedValueOnce({ id: "op-1", stage: "proposal" } as never);
    const r = await montar();
    await preencher(r, { id: "op-1" });
    await clicar(r);

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/crm/opportunities/op-1/stage");
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(init.method).toBe("POST");
    const corpo = JSON.parse(init.body ?? "{}");
    expect(corpo).toEqual({ stage: "qualified" });
    // O actor NAO vai no corpo: vem do token.
    expect(corpo).not.toHaveProperty("actor");
    expect(corpo).not.toHaveProperty("actor_id");
  });

  it("o id e' codificado no caminho", async () => {
    mockAgentFetch.mockResolvedValueOnce({ id: "a/b", stage: "qualified" } as never);
    const r = await montar();
    await preencher(r, { id: "a/b" });
    await clicar(r);

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toContain("a%2Fb");
    expect(caminho).not.toContain("a/b");
  });

  it("a recusa do servidor por fase terminal e' mostrada como REGRA", async () => {
    mockAgentFetch.mockRejectedValueOnce(
      new Error("crm opportunity is already closed; reopening would erase the outcome"),
    );
    const r = await montar();
    await preencher(r, { id: "op-1" });
    await clicar(r);

    const t = texto(r, "crm-fase-terminal");
    expect(t).toContain("já está fechada");
    expect(t).toContain("apagaria o resultado");
    expect(porTestId(r, "crm-fase-erro")).toHaveLength(0);
  });

  it("a regra terminal aparece no painel", async () => {
    const r = await montar();

    expect(texto(r, "crm-fase-regra")).toContain("reabrir apagaria o resultado");
  });
});
