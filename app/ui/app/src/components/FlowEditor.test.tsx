import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { describe, expect, it, vi } from "vitest";

const createScheduleMock = vi.fn(() => Promise.resolve({}));
vi.mock("@/lib/agenticClient", () => ({
  createSchedule: (...args: unknown[]) => createScheduleMock(...args),
}));

import { FlowEditor } from "./FlowEditor";

type TestNode = ReturnType<ReactTestRenderer["root"]["findAll"]>[number];

const fakeEvent = { stopPropagation() {}, preventDefault() {} };

function textOf(node: TestNode): string {
  const parts: string[] = [];
  const walk = (children: unknown): void => {
    if (typeof children === "string") {
      parts.push(children);
    } else if (Array.isArray(children)) {
      children.forEach(walk);
    } else if (children && typeof children === "object" && "children" in (children as TestNode).props) {
      walk((children as TestNode).props.children);
    }
  };
  walk(node.props.children);
  return parts.join(" ");
}

function clickButtonWithText(renderer: ReactTestRenderer, text: string) {
  const btn = renderer.root
    .findAll((n) => n.type === "button" && textOf(n).includes(text))
    .at(0);
  if (!btn) throw new Error(`button with text "${text}" not found`);
  act(() => {
    btn.props.onClick(fakeEvent);
  });
}

function clickByAriaLabel(renderer: ReactTestRenderer, label: string) {
  const el = renderer.root
    .findAll((n) => n.props?.["aria-label"] === label)
    .at(0);
  if (!el) throw new Error(`element with aria-label "${label}" not found`);
  act(() => {
    el.props.onClick(fakeEvent);
  });
}

function clickTestId(renderer: ReactTestRenderer, testid: string) {
  const el = renderer.root
    .findAll((n) => n.props?.["data-testid"] === testid)
    .at(0);
  if (!el) throw new Error(`element with data-testid "${testid}" not found`);
  act(() => {
    el.props.onClick(fakeEvent);
  });
}

function setInputs(renderer: ReactTestRenderer, values: string[]) {
  const inputs = renderer.root.findAll((n) => n.type === "input");
  values.forEach((value, i) => {
    if (inputs[i]) {
      act(() => inputs[i].props.onChange({ target: { value } }));
    }
  });
}

function nodeTestIds(renderer: ReactTestRenderer): string[] {
  return renderer.root
    .findAll((n) => typeof n.props?.["data-testid"] === "string" && n.props["data-testid"].startsWith("node-"))
    .map((n) => n.props["data-testid"] as string);
}

function hasTestId(renderer: ReactTestRenderer, testid: string): boolean {
  return renderer.root.findAll((n) => n.props?.["data-testid"] === testid).length > 0;
}

describe("FlowEditor", () => {
  it("adds nodes from the palette", () => {
    let r!: ReactTestRenderer;
    act(() => {
      r = create(<FlowEditor storageKey="test.flow.1" />);
    });
    clickButtonWithText(r, "Webhook");
    clickButtonWithText(r, "Rodar missão");
    expect(nodeTestIds(r)).toHaveLength(2);
    act(() => r.unmount());
  });

  it("shows a valid flow for a lone trigger and flags an orphan action", () => {
    let r!: ReactTestRenderer;
    act(() => {
      r = create(<FlowEditor storageKey="test.flow.2" />);
    });
    clickButtonWithText(r, "Webhook");
    expect(hasTestId(r, "flow-valid")).toBe(true);
    clickButtonWithText(r, "Rodar missão");
    // The action node has no incoming edge yet → orphan problem.
    expect(hasTestId(r, "flow-problems")).toBe(true);
    act(() => r.unmount());
  });

  it("connects a trigger to an action via click-to-connect", () => {
    let r!: ReactTestRenderer;
    act(() => {
      r = create(<FlowEditor storageKey="test.flow.3" />);
    });
    clickButtonWithText(r, "Webhook"); // n1_trigger
    clickButtonWithText(r, "Rodar missão"); // n2_action
    clickByAriaLabel(r, "Conectar saída de Webhook");
    clickTestId(r, "node-n2_action");
    // Edge created → flow becomes valid and a "Conexões" remove button exists.
    expect(hasTestId(r, "flow-valid")).toBe(true);
    const removeButtons = r.root.findAll((n) => n.props?.["aria-label"] === "Remover conexão");
    expect(removeButtons.length).toBe(1);
    act(() => r.unmount());
  });

  it("publishes a valid interval+mission flow as a schedule", async () => {
    let r!: ReactTestRenderer;
    act(() => {
      r = create(<FlowEditor storageKey="test.flow.pub" />);
    });
    clickButtonWithText(r, "Intervalo"); // n1_trigger
    clickButtonWithText(r, "Rodar missão"); // n2_action
    clickByAriaLabel(r, "Conectar saída de Intervalo");
    clickTestId(r, "node-n2_action"); // completes connection
    clickTestId(r, "node-n2_action"); // selects the mission node
    // Inspector inputs: [0] = Rótulo, [1] = objective config field.
    setInputs(r, ["Rodar missão", "Rodar testes do repositório"]);
    clickButtonWithText(r, "Publicar fluxo");
    await act(async () => {
      await Promise.resolve();
    });
    expect(createScheduleMock).toHaveBeenCalledWith({
      objective: "Rodar testes do repositório",
      interval_seconds: 3600,
    });
    act(() => r.unmount());
  });

  it("refuses to publish when the mission objective is empty", async () => {
    createScheduleMock.mockClear();
    let r!: ReactTestRenderer;
    act(() => {
      r = create(<FlowEditor storageKey="test.flow.pub2" />);
    });
    clickButtonWithText(r, "Intervalo");
    clickButtonWithText(r, "Rodar missão");
    clickByAriaLabel(r, "Conectar saída de Intervalo");
    clickTestId(r, "node-n2_action");
    clickButtonWithText(r, "Publicar fluxo");
    await act(async () => {
      await Promise.resolve();
    });
    expect(createScheduleMock).not.toHaveBeenCalled();
    expect(hasTestId(r, "flow-publish-msg")).toBe(true);
    act(() => r.unmount());
  });

  it("removes a node and its edges via the inspector", () => {
    let r!: ReactTestRenderer;
    act(() => {
      r = create(<FlowEditor storageKey="test.flow.4" />);
    });
    clickButtonWithText(r, "Webhook"); // n1_trigger
    clickButtonWithText(r, "Rodar missão"); // n2_action
    clickByAriaLabel(r, "Conectar saída de Webhook");
    clickTestId(r, "node-n2_action"); // select n2 after connect? connect clears selection
    // Select the action node, then delete it.
    clickTestId(r, "node-n2_action");
    clickByAriaLabel(r, "Excluir nó");
    expect(nodeTestIds(r)).toEqual(["node-n1_trigger"]);
    act(() => r.unmount());
  });
});
