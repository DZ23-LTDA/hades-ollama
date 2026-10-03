import { act, create, type ReactTestRenderer, type ReactTestInstance } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { askProjectDocuments } from "@/lib/agenticClient";
import { AskDocumentsPanel, ASK_NO_SOURCES_MESSAGE } from "./AskDocumentsPanel";

vi.mock("@/lib/agenticClient", () => ({ askProjectDocuments: vi.fn() }));

const mockAsk = vi.mocked(askProjectDocuments);

function textContent(node: ReactTestInstance): string {
  return node.children.map((child) => (typeof child === "string" ? child : textContent(child))).join("");
}

async function ask(query: string): Promise<ReactTestRenderer> {
  let renderer!: ReactTestRenderer;
  await act(async () => {
    renderer = create(<AskDocumentsPanel projectId="p1" />);
    await Promise.resolve();
  });
  const input = renderer.root.findByProps({ id: "ask-query" });
  await act(async () => {
    input.props.onChange({ target: { value: query } });
    await Promise.resolve();
  });
  const form = renderer.root.findByType("form");
  await act(async () => {
    form.props.onSubmit({ preventDefault() {} });
    await Promise.resolve();
    await Promise.resolve();
  });
  return renderer;
}

describe("AskDocumentsPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("asks the project and lists the cited sources when grounded", async () => {
    mockAsk.mockResolvedValue({
      project_id: "p1",
      query: "férias",
      citations: [{ index: 1, source: "rh.pdf#1", score: 0.98, snippet: "A política prevê 30 dias de férias." }],
      grounded: true,
    });

    const renderer = await ask("férias");

    expect(mockAsk).toHaveBeenCalledWith("p1", "férias");
    const text = textContent(renderer.root);
    expect(text).toContain("rh.pdf#1");
    expect(text).toContain("A política prevê 30 dias de férias.");
    expect(text).toContain("não gera uma resposta de IA");
    expect(text).not.toContain(ASK_NO_SOURCES_MESSAGE);
  });

  it("shows an honest message and no citations when not grounded", async () => {
    mockAsk.mockResolvedValue({ project_id: "p1", query: "x", citations: [], grounded: false });

    const renderer = await ask("x");

    expect(textContent(renderer.root)).toContain(ASK_NO_SOURCES_MESSAGE);
  });
});
