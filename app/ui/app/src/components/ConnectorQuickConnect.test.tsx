import type { ReactElement } from "react";
import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { describe, expect, it } from "vitest";
import type { AgentConnectorCatalogEntry } from "@/lib/agenticClient";
import { ConnectorQuickConnect } from "./ConnectorQuickConnect";

function entry(
  overrides: Partial<AgentConnectorCatalogEntry>,
): AgentConnectorCatalogEntry {
  return {
    id: "x",
    name: "X",
    category: "Comunicação",
    kind: "custom_api",
    description: "",
    auth: "api_key",
    source: "custom_api",
    status: "operator_setup_required",
    ...overrides,
  };
}

function render(node: ReactElement): ReactTestRenderer {
  let tree: ReactTestRenderer | undefined;
  act(() => {
    tree = create(node);
  });
  return tree as ReactTestRenderer;
}

function hasTestId(tree: ReactTestRenderer, id: string): boolean {
  return (
    tree.root.findAll((node) => node.props?.["data-testid"] === id).length > 0
  );
}

function textOf(tree: ReactTestRenderer): string {
  const parts: string[] = [];
  const walk = (children: unknown): void => {
    if (typeof children === "string") {
      parts.push(children);
      return;
    }
    if (Array.isArray(children)) {
      children.forEach(walk);
      return;
    }
    if (children && typeof children === "object" && "props" in children) {
      walk((children as { props?: { children?: unknown } }).props?.children);
    }
  };
  tree.root.findAll(() => true).forEach((node) => walk(node.props?.children));
  return parts.join(" ");
}

function card(overrides: Partial<AgentConnectorCatalogEntry>) {
  return render(
    <ConnectorQuickConnect
      entry={entry(overrides)}
      connected={false}
      onChanged={() => {}}
    />,
  );
}

describe("ConnectorQuickConnect channel guidance", () => {
  it("tells the operator how to connect Telegram before asking for a token", () => {
    const tree = card({ id: "telegram", name: "Telegram", auth: "bot_token" });
    expect(hasTestId(tree, "connector-guidance")).toBe(true);
    const text = textOf(tree);
    expect(text).toContain("BotFather");
    expect(text).toContain("Group Privacy Mode");
    // The card must keep explaining the registration path it had before.
    expect(text).toContain("registro avançado");
  });

  it("explains what Lark/Feishu needs and warns about the WhatsApp account", () => {
    expect(
      textOf(card({ id: "lark", name: "Lark / Feishu", auth: "bot_token" })),
    ).toContain("fluxo de trabalho");
    expect(
      textOf(
        card({
          id: "whatsapp",
          name: "WhatsApp Business",
          quick_connect: true,
        }),
      ),
    ).toContain("conta de teste exclusiva");
  });

  it("does not invent guidance for services without extra steps", () => {
    const tree = card({ id: "stripe", name: "Stripe", quick_connect: true });
    expect(hasTestId(tree, "connector-guidance")).toBe(false);
  });
});
