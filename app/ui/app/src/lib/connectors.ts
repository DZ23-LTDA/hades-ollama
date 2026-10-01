import type {
  AgentConnector,
  AgentConnectorCatalogEntry,
  AgentMCPServer,
  AgentSkill,
} from "@/lib/agenticClient";

export type ConnectorTab = "app" | "custom_api" | "mcp" | "connected";

export type ConnectorState = "connected" | "disabled" | "available";

// connectorState reports whether a catalog entry is registered in the runtime.
// Registration is what makes a connector usable; the catalog alone is not.
export function connectorState(
  entry: AgentConnectorCatalogEntry,
  connectors: AgentConnector[],
  mcpServers: AgentMCPServer[],
): ConnectorState {
  const registered =
    connectors.find(
      (item) => item.id === entry.id || item.provider === entry.id,
    ) ?? mcpServers.find((item) => item.id === entry.id);
  if (!registered) return "available";
  return registered.disabled ? "disabled" : "connected";
}

export function filterConnectors(
  catalog: AgentConnectorCatalogEntry[],
  tab: ConnectorTab,
  query: string,
  connectors: AgentConnector[],
  mcpServers: AgentMCPServer[],
): AgentConnectorCatalogEntry[] {
  const needle = query.trim().toLowerCase();
  return catalog.filter((entry) => {
    const inTab =
      tab === "connected"
        ? connectorState(entry, connectors, mcpServers) !== "available"
        : entry.kind === tab;
    if (!inTab) return false;
    if (!needle) return true;
    return [entry.name, entry.description, entry.category]
      .join(" ")
      .toLowerCase()
      .includes(needle);
  });
}

export type ManagedItem = {
  kind: "connector" | "mcp" | "skill";
  id: string;
  detail: string;
  enabled: boolean;
  remote: boolean;
};

// buildManagedItems flattens what is registered in the agent runtime into one
// list for the management tab. Remote MCP servers are marked so enable,
// disable and remove use the remote endpoints.
export function buildManagedItems(
  connectors: AgentConnector[],
  mcpServers: AgentMCPServer[],
  skills: AgentSkill[],
): ManagedItem[] {
  return [
    ...connectors.map((item) => ({
      kind: "connector" as const,
      id: item.id,
      detail: `${item.provider} · ${item.base_url}`,
      enabled: !item.disabled,
      remote: false,
    })),
    ...mcpServers.map((item) => {
      const remote = item.transport === "streamable-http" || !!item.url;
      return {
        kind: "mcp" as const,
        id: item.id,
        detail: remote
          ? `Remoto · ${item.url ?? ""}`
          : `Local · ${item.command ?? ""}`,
        enabled: !item.disabled,
        remote,
      };
    }),
    ...skills.map((item) => ({
      kind: "skill" as const,
      id: item.id,
      detail: `v${item.version}${item.description ? ` · ${item.description}` : ""}`,
      enabled: item.enabled,
      remote: false,
    })),
  ];
}
