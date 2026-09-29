// newTaskShortcut labels the Ctrl/Cmd+K shortcut for the current platform.
export function newTaskShortcut(): string {
  const platform =
    typeof navigator === "undefined" ? "" : navigator.platform.toLowerCase();
  return platform.includes("mac") ? "⌘K" : "Ctrl K";
}

const REPO = "https://github.com/DZ23-LTDA/ollama-classe-a-plus";

export const HELP_LINKS = [
  {
    label: "Documentação",
    hint: "Guias do projeto no GitHub",
    href: `${REPO}/tree/main/docs`,
  },
  {
    label: "Reportar um problema",
    hint: "Abrir uma issue no GitHub",
    href: `${REPO}/issues/new`,
  },
  {
    label: "Novidades",
    hint: "Histórico de mudanças",
    href: `${REPO}/blob/main/CHANGELOG.md`,
  },
  {
    label: "Código-fonte",
    hint: "Repositório do Ollama Classe A+",
    href: REPO,
  },
];

export const KEYBOARD_SHORTCUTS = [
  { action: "Pesquisar conversas e páginas", keys: "/" },
  { action: "Nova tarefa", keys: newTaskShortcut() },
  { action: "Fechar janela", keys: "Esc" },
];
