import { act, create, type ReactTestInstance } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../lib/agenticClient", () => ({
  getSkillVersions: vi.fn(),
  restoreSkillVersion: vi.fn(),
}));

import { SkillVersionsCard } from "./SkillVersionsCard";
import { getSkillVersions, restoreSkillVersion } from "../lib/agenticClient";

function texto(node: ReactTestInstance): string {
  return node.children.map((child) => (typeof child === "string" ? child : texto(child))).join("");
}

async function renderizar(skillId = "skill-x"): Promise<ReturnType<typeof create>> {
  let arvore!: ReturnType<typeof create>;
  await act(async () => {
    arvore = create(<SkillVersionsCard skillId={skillId} />);
  });
  await act(async () => {});
  return arvore;
}

describe("SkillVersionsCard", () => {
  beforeEach(() => {
    vi.mocked(getSkillVersions).mockReset();
    vi.mocked(restoreSkillVersion).mockReset();
  });

  it("lista as versões arquivadas", async () => {
    vi.mocked(getSkillVersions).mockResolvedValue({ versions: ["1.0", "1.5"] });
    const arvore = await renderizar();
    const conteudo = texto(arvore.root);
    expect(conteudo).toContain("Histórico de versões");
    expect(conteudo).toContain("1.0");
    expect(conteudo).toContain("1.5");
  });

  it("lista vazia diz que não há versões anteriores — e NÃO afirma que a skill não existe", async () => {
    vi.mocked(getSkillVersions).mockResolvedValue({ versions: [] });
    const conteudo = texto((await renderizar()).root);
    expect(conteudo).toContain("Nenhuma versão anterior arquivada");
    // O servidor responde vazio também para skill de OUTRA organização, de propósito: a
    // superfície não pode transformar esse vazio numa afirmação de inexistência.
    expect(conteudo).not.toContain("não existe");
    expect(conteudo).not.toContain("inexistente");
  });

  it("***depois de restaurar, mostra a confiança COMO O SERVIDOR A DEVOLVEU***", async () => {
    vi.mocked(getSkillVersions).mockResolvedValue({ versions: ["1.0"] });
    vi.mocked(restoreSkillVersion).mockResolvedValue({
      skill: { id: "skill-x", version: "1.0", description: "v1", trusted: false },
    });
    const arvore = await renderizar();
    const botao = arvore.root.findAllByType("button")[0];
    await act(async () => {
      botao.props.onClick();
    });
    const conteudo = texto(arvore.root);
    expect(conteudo).toContain("Versão activa agora");
    expect(conteudo).toContain("1.0");
    // (1) E A CONFIANÇA NÃO É ARREDONDADA: restaurar não promove.
    expect(conteudo).toContain("não confiável");
    expect(conteudo).not.toContain("confiança: confiável");
  });

  it("ao falhar a restauração, diz o motivo e não afirma sucesso", async () => {
    vi.mocked(getSkillVersions).mockResolvedValue({ versions: ["1.0"] });
    vi.mocked(restoreSkillVersion).mockRejectedValue(new Error("versão arquivada não corresponde ao pedido"));
    const arvore = await renderizar();
    await act(async () => {
      arvore.root.findAllByType("button")[0].props.onClick();
    });
    const conteudo = texto(arvore.root);
    expect(conteudo).toContain("não corresponde ao pedido");
    expect(conteudo).not.toContain("Versão activa agora");
  });
});
