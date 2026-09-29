export type SettingsTabId = "general" | "providers" | "endpoint" | "apps";

export const SETTINGS_TABS: Array<{
  id: SettingsTabId;
  label: string;
  href: string;
}> = [
  { id: "general", label: "Geral", href: "/settings" },
  { id: "providers", label: "Provedores de IA", href: "/providers" },
  { id: "endpoint", label: "Endpoint da API", href: "/endpoint" },
  { id: "apps", label: "Apps e integrações", href: "/connect" },
];

// Sidebar sections that belong to Configurações keep that item highlighted.
export const SETTINGS_SECTIONS = new Set([
  "settings",
  "providers",
  "endpoint",
  "apps",
]);
