import { act, create, type ReactTestInstance } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../lib/agenticClient", () => ({ getCapabilities: vi.fn() }));

import { CapabilitiesCard } from "./CapabilitiesCard";
import { getCapabilities } from "../lib/agenticClient";

function texto(node: ReactTestInstance): string {
  return node.children.map((child) => (typeof child === "string" ? child : texto(child))).join("");
}

async function renderizar(model?: string): Promise<string> {
  let arvore!: ReturnType<typeof create>;
  await act(async () => {
    arvore = create(<CapabilitiesCard model={model} />);
  });
  await act(async () => {});
  return texto(arvore.root);
}

const LISTA = {
  models: [
    { model: "qwen2.5:7b", tool_calling: true, vision: false, structured_output: true, embeddings: false, source: "medido", note: "tool calls nativos funcionam (medido)." },
    { model: "modelo-de-doc:1b", tool_calling: false, vision: false, structured_output: false, embeddings: false, source: "declarado", note: "documentação do provedor" },
  ],
  policy: "Modelo ausente desta lista esta' NAO_MEDIDO: o runtime cai em modo COMPATIVEL.",
  scope: "fatos medidos/declarados neste ambiente",
};

describe("CapabilitiesCard", () => {
  beforeEach(() => {
    vi.mocked(getCapabilities).mockReset();
  });

  it("lista os modelos e MOSTRA A ORIGEM de cada um", async () => {
    vi.mocked(getCapabilities).mockResolvedValue(LISTA as never);
    const conteudo = await renderizar();
    expect(conteudo).toContain("Capacidade dos modelos");
    expect(conteudo).toContain("qwen2.5:7b");
    // ***CAPACIDADE SEM ORIGEM NÃO É CAPACIDADE*** — as duas origens têm de aparecer.
    expect(conteudo).toContain("medido");
    expect(conteudo).toContain("declarado");
    expect(conteudo).toContain("tool calls nativos");
    expect(conteudo).toContain("sem tool call nativo");
    // E o escopo/política: um dado sem escopo é lido como global.
    expect(conteudo).toContain("NAO_MEDIDO");
  });

  it("***modelo NÃO MEDIDO: diz que não foi medido e NÃO afirma nem nega suporte***", async () => {
    vi.mocked(getCapabilities).mockResolvedValue({
      model: "ninguem-sondou:99b",
      known: false,
      tool_mode: "unknown",
      reason: "capacidade NAO MEDIDA para este modelo: o runtime usa modo compativel e DECLARA isso",
    } as never);
    const conteudo = await renderizar("ninguem-sondou:99b");
    expect(conteudo).toContain("NÃO MEDIDO");
    expect(conteudo).toContain("modo compatível");
    // Não pode aparecer como suportado nem como recusado.
    expect(conteudo).not.toContain("tool calls nativos");
    expect(conteudo).not.toContain("não suporta");
  });

  it("modelo conhecido mostra o modo e a origem", async () => {
    vi.mocked(getCapabilities).mockResolvedValue({
      model: "qwen2.5:7b",
      known: true,
      tool_mode: "native",
      reason: "capacidade medida neste ambiente",
      capabilities: { model: "qwen2.5:7b", tool_calling: true, vision: false, structured_output: true, embeddings: false, source: "medido" },
    } as never);
    const conteudo = await renderizar("qwen2.5:7b");
    expect(conteudo).toContain("medido/declarado");
    expect(conteudo).toContain("medido");
  });

  it("payload inesperado NÃO derruba a página (a lição do E-651)", async () => {
    vi.mocked(getCapabilities).mockResolvedValue({} as never);
    const conteudo = await renderizar();
    expect(conteudo).toContain("não devolveu o relatório de capacidade");
  });

  it("ao falhar, diz o motivo e não afirma capacidade nenhuma", async () => {
    vi.mocked(getCapabilities).mockRejectedValue(new Error("sem permissão"));
    const conteudo = await renderizar();
    expect(conteudo).toContain("sem permissão");
    expect(conteudo).not.toContain("tool calls nativos");
  });
});
