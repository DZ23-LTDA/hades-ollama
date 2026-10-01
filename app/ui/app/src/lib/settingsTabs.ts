export type SettingsTabId = "general" | "providers" | "endpoint" | "connectors" | "apps";

export const SETTINGS_TABS: Array<{
  id: SettingsTabId;
  label: string;
  href: string;
}> = [
  { id: "general", label: "Geral", href: "/settings" },
  { id: "providers", label: "Provedores de IA", href: "/providers" },
  { id: "endpoint", label: "Computadores & API", href: "/endpoint" },
  { id: "connectors", label: "Conectores e Plugins", href: "/connectors" },
  { id: "apps", label: "Harnesses & Codex", href: "/connect" },
];

// Sidebar sections that belong to Configurações keep that item highlighted.
export const SETTINGS_SECTIONS = new Set([
  "settings",
  "providers",
  "endpoint",
  "connectors",
  "apps",
]);
