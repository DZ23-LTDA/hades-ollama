import { act, create, type ReactTestInstance } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AgenticSplitShell, type MissionData } from "./AgenticSplitShell";

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: vi.fn().mockResolvedValue({ events: [] }),
  agentFetchBlob: vi.fn().mockResolvedValue(new Blob(["artifact content"], { type: "text/plain" })),
}));

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

describe("AgenticSplitShell component", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    // Mock EventSource for testing
    vi.stubGlobal("EventSource", class {
      addEventListener = vi.fn();
      removeEventListener = vi.fn();
      close = vi.fn();
    });
  });

  const sampleMission: MissionData = {
    id: "mis_test_split_1",
    objective: "Desenvolver protótipo interativo e navegar no browser",
    state: "AWAITING_APPROVAL",
    plan: [
      {
        id: "step_1",
        title: "Navegar no site e inspecionar",
        kind: "browser.operator",
        state: "COMPLETED",
        requires_approval: false,
      },
      {
        id: "step_2",
        title: "Gravar modificação e gerar artefato",
        kind: "workspace.write",
        state: "AWAITING_APPROVAL",
        requires_approval: true,
      },
    ],
    approvals: [
      {
        id: "app_1",
        step_id: "step_2",
        status: "PENDING",
        policy: "workspace:write",
        nonce: "nonce_xyz",
      },
    ],
    artifacts: [
      {
        id: "art_1",
        name: "relatorio.md",
        size: 1024,
        sha256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      },
    ],
  };

  it("renders split-screen layout with objective, status and tabs", async () => {
    let renderer!: ReturnType<typeof create>;
    await act(async () => {
      renderer = create(<AgenticSplitShell mission={sampleMission} />);
      await Promise.resolve();
    });

    const rootText = textContent(renderer.root);
    expect(rootText).toContain("mis_test_split_1");
    expect(rootText).toContain("Desenvolver protótipo interativo");
    // Mission state is now rendered through the pt-BR label map, not the raw enum.
    expect(rootText).toContain("Aguardando aprovação");
    expect(rootText).not.toContain("AWAITING_APPROVAL");
    expect(rootText).toContain("Navegador ao Vivo");
    expect(rootText).toContain("Artefatos");
    // The former "Terminal / Canvas" tab is now "Eventos".
    expect(rootText).toContain("Eventos");
    expect(rootText).toContain("Esta etapa requer aprovação humana");
  });

  it("switches to artifacts tab and displays artifact details", async () => {
    let renderer!: ReturnType<typeof create>;
    await act(async () => {
      renderer = create(<AgenticSplitShell mission={sampleMission} />);
      await Promise.resolve();
    });

    const artifactsTab = renderer.root
      .findAllByType("button")
      .find((b) => textContent(b).includes("Artefatos (1)"));
    expect(artifactsTab).toBeDefined();

    await act(async () => {
      artifactsTab!.props.onClick();
      await Promise.resolve();
    });

    const rootText = textContent(renderer.root);
    expect(rootText).toContain("relatorio.md");
  });
});
