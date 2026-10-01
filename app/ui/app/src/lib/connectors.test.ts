import { describe, expect, it } from "vitest";
import type { AgentConnectorCatalogEntry } from "./agenticClient";
import {
  buildManagedItems,
  connectorState,
  filterConnectors,
} from "./connectors";

const entry = (
  id: string,
  kind: string,
  name = id,
): AgentConnectorCatalogEntry => ({
  id,
  name,
  kind,
  category: "Teste",
  description: `${name} descrição`,
  auth: "oauth",
  source: "built-in",
  status: "available",
});

const catalog = [
  entry("github", "app", "GitHub"),
  entry("jira", "custom_api", "Jira"),
  entry("composio", "mcp", "Composio"),
];

describe("connectors", () => {
  it("derives state from registered connectors and MCP servers", () => {
    expect(connectorState(catalog[0], [], [])).toBe("available");
    expect(
      connectorState(
        catalog[0],
        [{ id: "gh", provider: "github", base_url: "https://api.github.com" }],
        [],
      ),
    ).toBe("connected");
    expect(
      connectorState(
        catalog[0],
        [{ id: "github", provider: "github", base_url: "x", disabled: true }],
        [],
      ),
    ).toBe("disabled");
    expect(connectorState(catalog[2], [], [{ id: "composio" }])).toBe(
      "connected",
    );
  });

  it("filters by tab and search text", () => {
    expect(
      filterConnectors(catalog, "app", "", [], []).map((e) => e.id),
    ).toEqual(["github"]);
    expect(
      filterConnectors(catalog, "custom_api", "JIRA", [], []).map((e) => e.id),
    ).toEqual(["jira"]);
    expect(filterConnectors(catalog, "app", "jira", [], [])).toEqual([]);
    expect(
      filterConnectors(catalog, "connected", "", [], [{ id: "composio" }]).map(
        (e) => e.id,
      ),
    ).toEqual(["composio"]);
  });
});

describe("buildManagedItems", () => {
  it("flattens connectors, local and remote MCP servers and skills", () => {
    const items = buildManagedItems(
      [
        {
          id: "gh",
          provider: "github",
          base_url: "https://api.github.com",
          disabled: true,
        },
      ],
      [
        { id: "local", command: "C:/node.exe" },
        {
          id: "remote",
          url: "https://mcp.example.com",
          transport: "streamable-http",
        },
      ],
      [
        {
          id: "s1",
          version: "1.0.0",
          description: "Teste",
          trusted: false,
          enabled: true,
        },
      ],
    );
    expect(items.map((i) => [i.kind, i.id, i.enabled, i.remote])).toEqual([
      ["connector", "gh", false, false],
      ["mcp", "local", true, false],
      ["mcp", "remote", true, true],
      ["skill", "s1", true, false],
    ]);
  });
});
