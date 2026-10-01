export type SearchItem = {
  id: string;
  label: string;
  hint: string;
  href: string;
};

// Pages reachable from the sidebar. Kept here so search covers navigation
// even before any chat exists.
export const SEARCH_PAGES: SearchItem[] = [
  { id: "page-new", label: "Nova tarefa", hint: "Página", href: "/" },
  { id: "page-agentic", label: "Agente", hint: "Página", href: "/agentic" },
  { id: "page-tasks", label: "Tarefas", hint: "Página", href: "/tasks" },
  {
    id: "page-scheduled",
    label: "Agendado",
    hint: "Página",
    href: "/scheduled",
  },
  { id: "page-company", label: "Empresa", hint: "Página", href: "/company" },
  { id: "page-skills", label: "Habilidades", hint: "Página", href: "/skills" },
  {
    id: "page-connectors",
    label: "Conectores",
    hint: "Página",
    href: "/connectors",
  },
  { id: "page-library", label: "Biblioteca", hint: "Página", href: "/library" },
  { id: "page-projects", label: "Projetos", hint: "Página", href: "/projects" },
  {
    id: "page-connect",
    label: "Apps e providers",
    hint: "Página",
    href: "/connect",
  },
  {
    id: "page-providers",
    label: "Provedores de IA",
    hint: "Página",
    href: "/providers",
  },
  {
    id: "page-endpoint",
    label: "Endpoint da API",
    hint: "Configurações",
    href: "/endpoint",
  },
  {
    id: "page-settings",
    label: "Configurações",
    hint: "Página",
    href: "/settings",
  },
];

function normalize(value: string) {
  return value.normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase();
}

// searchItems ranks label prefix matches first, then any substring match in
// the label or hint text. Accents and case are ignored.
export function searchItems(
  items: SearchItem[],
  query: string,
  limit = 30,
): SearchItem[] {
  const needle = normalize(query.trim());
  if (!needle) return items.slice(0, limit);
  const scored: Array<{ item: SearchItem; score: number }> = [];
  for (const item of items) {
    const label = normalize(item.label);
    const hint = normalize(item.hint);
    let score = -1;
    if (label.startsWith(needle)) score = 0;
    else if (label.includes(needle)) score = 1;
    else if (hint.includes(needle)) score = 2;
    if (score >= 0) scored.push({ item, score });
  }
  scored.sort((a, b) => a.score - b.score);
  return scored.slice(0, limit).map((entry) => entry.item);
}

// isTypingTarget avoids hijacking "/" while the user types in a field.
export function isTypingTarget(target: EventTarget | null): boolean {
  if (typeof HTMLElement === "undefined" || !(target instanceof HTMLElement))
    return false;
  const tag = target.tagName;
  return (
    tag === "INPUT" ||
    tag === "TEXTAREA" ||
    tag === "SELECT" ||
    target.isContentEditable
  );
}
