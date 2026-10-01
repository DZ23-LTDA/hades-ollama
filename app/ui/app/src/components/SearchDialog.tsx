import { useEffect, useMemo, useRef, useState } from "react";
import { MagnifyingGlassIcon } from "@heroicons/react/24/outline";
import { getChats } from "@/api";
import { SEARCH_PAGES, searchItems, type SearchItem } from "@/lib/search";

export function SearchDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const [query, setQuery] = useState("");
  const [chats, setChats] = useState<SearchItem[]>([]);
  const [active, setActive] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!open) return;
    setQuery("");
    setActive(0);
    inputRef.current?.focus();
    getChats()
      .then((response) =>
        setChats(
          (response.chatInfos ?? []).map((chat) => ({
            id: `chat-${chat.id}`,
            label: chat.title || chat.userExcerpt || "Conversa sem título",
            hint: chat.userExcerpt || "Conversa",
            href: `/c/${chat.id}`,
          })),
        ),
      )
      .catch(() => setChats([]));
  }, [open]);

  const results = useMemo(
    () => searchItems([...SEARCH_PAGES, ...chats], query),
    [chats, query],
  );

  if (!open) return null;

  const go = (item: SearchItem | undefined) => {
    if (!item) return;
    onClose();
    window.location.assign(item.href);
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/40 px-4 pt-[12vh]"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Pesquisar"
        className="w-full max-w-xl overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-2xl dark:border-neutral-700 dark:bg-neutral-900"
      >
        <label className="flex items-center gap-3 border-b border-neutral-200 px-4 py-3 dark:border-neutral-800">
          <MagnifyingGlassIcon className="h-5 w-5 text-neutral-400" />
          <input
            ref={inputRef}
            value={query}
            onChange={(event) => {
              setQuery(event.target.value);
              setActive(0);
            }}
            onKeyDown={(event) => {
              if (event.key === "Escape") onClose();
              else if (event.key === "ArrowDown") {
                event.preventDefault();
                setActive((index) => Math.min(index + 1, results.length - 1));
              } else if (event.key === "ArrowUp") {
                event.preventDefault();
                setActive((index) => Math.max(index - 1, 0));
              } else if (event.key === "Enter") {
                event.preventDefault();
                go(results[active]);
              }
            }}
            placeholder="Pesquisar conversas e páginas"
            aria-label="Pesquisar conversas e páginas"
            className="w-full bg-transparent text-sm text-neutral-900 outline-none placeholder:text-neutral-400 dark:text-neutral-100"
          />
        </label>
        <ul
          role="listbox"
          aria-label="Resultados"
          className="max-h-[50vh] overflow-y-auto p-2"
        >
          {results.length === 0 && (
            <li className="px-3 py-6 text-center text-sm text-neutral-500">
              Nada encontrado.
            </li>
          )}
          {results.map((item, index) => (
            <li key={item.id} role="option" aria-selected={index === active}>
              <button
                type="button"
                onMouseEnter={() => setActive(index)}
                onClick={() => go(item)}
                className={`flex w-full items-center justify-between gap-3 rounded-lg px-3 py-2 text-left text-sm ${
                  index === active
                    ? "bg-neutral-100 text-neutral-900 dark:bg-neutral-800 dark:text-white"
                    : "text-neutral-700 dark:text-neutral-300"
                }`}
              >
                <span className="min-w-0 truncate">{item.label}</span>
                <span className="shrink-0 truncate text-[11px] text-neutral-400">
                  {item.hint}
                </span>
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
