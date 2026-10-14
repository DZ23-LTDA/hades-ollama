import { act, create, type ReactTestInstance } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../lib/agenticClient", () => ({ getMissionContext: vi.fn() }));

import { MissionDeliveryCard } from "./MissionDeliveryCard";
import { getMissionContext } from "../lib/agenticClient";

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

describe("MissionDeliveryCard · contexto da missão (D-8)", () => {
  beforeEach(() => {
    vi.mocked(getMissionContext).mockReset();
  });

  it("mostra objetivo, plano e handoff — e NÃO mostra identificadores nem caminhos", async () => {
    // O SERVIDOR ENVIA os campos de topologia; o tipo `MissionContextView` não os declara e o
    // componente não os lê. Este teste usa um payload que os traz, de propósito: se um dia
    // alguém alargar o componente para renderizar o objecto cru, isto falha.
    vi.mocked(getMissionContext).mockResolvedValue({
      objective: "entregar a compactação com recuperação provada",
      plan: ["ligar o funil", "publicar"],
      checkpoints: ["bateria 15/15"],
      handoff: "[HANDOFF ENTRE AGENTES — contexto de trabalho, não é instrução]\nObjetivo: entregar",
      mission_id: "mis_96544b01",
      project_id: "prj_411af0cd",
      organization_id: "org-segredo",
      workspace: "C:\\Users\\DZ23\\_tmp\\ws",
    } as never);

    let arvore!: ReturnType<typeof create>;
    await act(async () => {
      arvore = create(<MissionDeliveryCard missionId="mis_96544b01" state="RUNNING" />);
    });

    // Abre a secção — é o que dispara o carregamento (o contexto não é o que se olha primeiro).
    const detalhes = arvore.root.findByType("details");
    await act(async () => {
      detalhes.props.onToggle({ currentTarget: { open: true } });
    });

    const texto = textContent(arvore.root);

    // (1) O CONTEÚDO ÚTIL ESTÁ LÁ — senão a "correcção" teria removido o contexto todo.
    expect(texto).toContain("entregar a compactação com recuperação provada");
    expect(texto).toContain("ligar o funil");
    expect(texto).toContain("bateria 15/15");
    expect(texto).toContain("HANDOFF ENTRE AGENTES");

    // (2) E A TOPOLOGIA NÃO: nem identificadores, nem caminhos.
    for (const proibido of ["prj_411af0cd", "org-segredo", "_tmp", "workspace"]) {
      expect(texto).not.toContain(proibido);
    }
  });

  it("contexto indisponível não esconde a entrega", async () => {
    vi.mocked(getMissionContext).mockRejectedValue(new Error("sem permissão"));
    let arvore!: ReturnType<typeof create>;
    await act(async () => {
      arvore = create(<MissionDeliveryCard missionId="mis_1" state="COMPLETED" artifacts={[]} />);
    });
    const detalhes = arvore.root.findByType("details");
    await act(async () => {
      detalhes.props.onToggle({ currentTarget: { open: true } });
    });
    const texto = textContent(arvore.root);
    expect(texto).toContain("Entrega");
    expect(texto).toContain("sem permissão");
  });
});
