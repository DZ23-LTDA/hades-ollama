import { useEffect, useState } from "react";
import { XMarkIcon } from "@heroicons/react/24/outline";
import { API_BASE } from "@/lib/config";
import { HELP_LINKS, KEYBOARD_SHORTCUTS } from "@/lib/help";

// HelpDialog shows the running version, where to get help and the keyboard
// shortcuts. It replaces a link to a settings anchor that did not exist.
export function HelpDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const [version, setVersion] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    fetch(`${API_BASE}/api/version`)
      .then((response) => (response.ok ? response.json() : null))
      .then((body: { version?: string } | null) =>
        setVersion(body?.version ?? null),
      )
      .catch(() => setVersion(null));
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/40 px-4 pt-[10vh]"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Ajuda e sobre"
        className="w-full max-w-md rounded-2xl border border-neutral-200 bg-white p-6 shadow-2xl dark:border-neutral-700 dark:bg-neutral-900"
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-neutral-900 dark:text-white">
              Ollama Classe A+
            </h2>
            <p className="mt-1 text-xs text-neutral-500">
              Versão {version ?? "desconhecida"} · Local-first
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Fechar"
            className="rounded-lg p-1 text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"
          >
            <XMarkIcon className="h-5 w-5" />
          </button>
        </div>

        <h3 className="mt-5 text-xs font-semibold uppercase tracking-wider text-neutral-400">
          Ajuda
        </h3>
        <ul className="mt-2 space-y-1">
          {HELP_LINKS.map((link) => (
            <li key={link.href}>
              <a
                href={link.href}
                target="_blank"
                rel="noreferrer"
                className="block rounded-lg px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-100 dark:text-neutral-200 dark:hover:bg-neutral-800"
              >
                {link.label}
                <span className="block text-[11px] text-neutral-400">
                  {link.hint}
                </span>
              </a>
            </li>
          ))}
        </ul>

        <h3 className="mt-5 text-xs font-semibold uppercase tracking-wider text-neutral-400">
          Atalhos
        </h3>
        <dl className="mt-2 space-y-1 text-sm">
          {KEYBOARD_SHORTCUTS.map((shortcut) => (
            <div key={shortcut.keys} className="flex justify-between px-3 py-1">
              <dt className="text-neutral-600 dark:text-neutral-300">
                {shortcut.action}
              </dt>
              <dd>
                <kbd className="rounded border border-neutral-300 px-1.5 text-xs text-neutral-500 dark:border-neutral-700">
                  {shortcut.keys}
                </kbd>
              </dd>
            </div>
          ))}
        </dl>
      </div>
    </div>
  );
}
