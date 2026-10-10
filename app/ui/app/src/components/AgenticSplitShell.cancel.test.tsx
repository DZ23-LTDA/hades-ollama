import { act, create, type ReactTestInstance } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AgenticSplitShell, type MissionData } from "./AgenticSplitShell";
import { agentFetch } from "@/lib/agenticClient";

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: vi.fn().mockResolvedValue({ events: [] }),
  agentFetchBlob: vi
    .fn()
    .mockResolvedValue(new Blob(["artifact"], { type: "text/plain" })),
}));

/**
 * Regressao da capacidade de cancelamento da jornada de primeira execucao:
 * com a missao em execucao, "Interromper" precisa chamar
 * `POST /api/agent/v1/missions/<id>/cancel` e recarregar a missao.
 */

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

const runningMission: MissionData = {
  id: "mis_jornada_cancel",
  objective: "Executar a jornada de primeira execucao",
  state: "RUNNING",
  plan: [
    {
      id: "step_1",
      title: "Preparar o workspace",
      kind: "workspace.read",
      state: "COMPLETED",
      requires_approval: false,
    },
  ],
  approvals: [],
  artifacts: [],
};

describe("AgenticSplitShell cancelamento", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    vi.stubGlobal(
      "EventSource",
      class {
        addEventListener = vi.fn();
        removeEventListener = vi.fn();
        close = vi.fn();
      },
    );
  });

  it("envia POST de cancelamento ao interromper uma missao em execucao", async () => {
    const onRefreshMission = vi.fn().mockResolvedValue(undefined);
    let renderer!: ReturnType<typeof create>;

    await act(async () => {
      renderer = create(
        <AgenticSplitShell
          mission={runningMission}
          onRefreshMission={onRefreshMission}
        />,
      );
      await Promise.resolve();
    });

    const stop = renderer.root
      .findAllByType("button")
      .find((button) => textContent(button).includes("Interromper"));
    expect(stop).toBeDefined();

    await act(async () => {
      stop!.props.onClick();
      await Promise.resolve();
    });

    const calls = vi.mocked(agentFetch).mock.calls;
    const cancelCall = calls.find(
      ([path]) =>
        path === `/api/agent/v1/missions/${runningMission.id}/cancel`,
    );
    expect(cancelCall).toBeDefined();
    expect(cancelCall?.[1]?.method).toBe("POST");
    expect(onRefreshMission).toHaveBeenCalled();
  });

  it("nao oferece cancelamento quando a missao nao esta em execucao", async () => {
    let renderer!: ReturnType<typeof create>;

    await act(async () => {
      renderer = create(
        <AgenticSplitShell mission={{ ...runningMission, state: "CANCELLED" }} />,
      );
      await Promise.resolve();
    });

    const stop = renderer.root
      .findAllByType("button")
      .find((button) => textContent(button).includes("Interromper"));
    expect(stop).toBeUndefined();
  });
});
