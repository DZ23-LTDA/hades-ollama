export type SettingsTabId = "general" | "providers" | "endpoint" | "connectors" | "apps";

// `label` fica como texto canonico pt-BR (fallback e leitura humana);
// `labelKey` e' o que a UI usa, resolvido no idioma activo no momento do render.
export const SETTINGS_TABS: Array<{
  id: SettingsTabId;
  label: string;
  href: string;
  labelKey: string;
}> = [
  { id: "general", label: "Geral", href: "/settings", labelKey: "settings.tab.general" },
  { id: "providers", label: "Provedores de IA", href: "/providers", labelKey: "search.page.providers" },
  { id: "endpoint", label: "Computadores & API", href: "/endpoint", labelKey: "settings.tab.endpoint" },
  { id: "connectors", label: "Conectores e Plugins", href: "/connectors", labelKey: "settings.tab.connectors" },
  { id: "apps", label: "Harnesses & Codex", href: "/connect", labelKey: "settings.tab.apps" },
];
// Sidebar sections that belong to Configurações keep that item highlighted.
export const SETTINGS_SECTIONS = new Set([
  "settings",
  "providers",
  "endpoint",
  "connectors",
  "apps",
]);
