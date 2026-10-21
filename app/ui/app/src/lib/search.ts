export type SearchItem = {
  id: string;
  label: string;
  hint: string;
  href: string;
  // Chaves i18n opcionais: quando presentes, o SearchDialog resolve o texto no idioma
  // activo ANTES de procurar, para que a busca funcione no idioma escolhido (escrever
  // "Settings" tem de encontrar a pagina, nao so' "Configuracoes").
  labelKey?: string;
  hintKey?: string;
};

// Pages reachable from the sidebar. Kept here so search covers navigation
// even before any chat exists.
// `label`/`hint` ficam como texto canonico pt-BR (usado por testes e como fallback);
// `labelKey`/`hintKey` sao o que a UI usa.
export const SEARCH_PAGES: SearchItem[] = [
  { id: "page-new", label: "Nova tarefa", hint: "Página", href: "/", labelKey: "appSidebar.newTask", hintKey: "search.hint.page" },
  { id: "page-agentic", label: "Agente", hint: "Página", href: "/agentic", labelKey: "search.page.agentic", hintKey: "search.hint.page" },
  { id: "page-tasks", label: "Tarefas", hint: "Página", href: "/tasks", labelKey: "appSidebar.nav.tasks", hintKey: "search.hint.page" },
  {
    id: "page-scheduled",
    label: "Agendado",
    hint: "Página",
    href: "/scheduled",
    labelKey: "search.page.scheduled",
    hintKey: "search.hint.page",
  },
  { id: "page-company", label: "Empresa", hint: "Página", href: "/company", labelKey: "appSidebar.nav.company", hintKey: "search.hint.page" },
  { id: "page-skills", label: "Habilidades", hint: "Página", href: "/skills", labelKey: "appSidebar.nav.skills", hintKey: "search.hint.page" },
  {
    id: "page-connectors",
    label: "Conectores",
    hint: "Página",
    href: "/connectors",
    labelKey: "search.page.connectors",
    hintKey: "search.hint.page",
  },
  { id: "page-library", label: "Biblioteca", hint: "Página", href: "/library", labelKey: "appSidebar.nav.library", hintKey: "search.hint.page" },
  { id: "page-projects", label: "Projetos", hint: "Página", href: "/projects", labelKey: "appSidebar.section.projects", hintKey: "search.hint.page" },
  {
    id: "page-connect",
    label: "Apps e providers",
    hint: "Página",
    href: "/connect",
    labelKey: "search.page.connect",
    hintKey: "search.hint.page",
  },
  {
    id: "page-providers",
    label: "Provedores de IA",
    hint: "Página",
    href: "/providers",
    labelKey: "search.page.providers",
    hintKey: "search.hint.page",
  },
  {
    id: "page-endpoint",
    label: "Endpoint da API",
    hint: "Configurações",
    href: "/endpoint",
    labelKey: "search.page.endpoint",
    hintKey: "appSidebar.nav.settings",
  },
  {
    id: "page-settings",
    label: "Configurações",
    hint: "Página",
    href: "/settings",
    labelKey: "appSidebar.nav.settings",
    hintKey: "search.hint.page",
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
