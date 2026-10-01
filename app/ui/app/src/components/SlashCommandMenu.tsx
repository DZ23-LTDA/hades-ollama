import { CommandLineIcon } from "@heroicons/react/24/outline";
import { filterSlashCommands, type SlashCommand } from "@/lib/slashCommands";

type SlashCommandMenuProps = {
  query: string | null;
  activeIndex: number;
  onActiveIndexChange: (index: number) => void;
  onSelect: (command: SlashCommand) => void;
};

export function SlashCommandMenu({
  query,
  activeIndex,
  onActiveIndexChange,
  onSelect,
}: SlashCommandMenuProps) {
  if (query === null) return null;
  const commands = filterSlashCommands(query);
  if (commands.length === 0) {
    return (
      <div role="status" className="rounded-xl border border-neutral-200 bg-white p-3 text-xs text-neutral-500 shadow-lg dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-400">
        Nenhum comando disponível para esta busca.
      </div>
    );
  }

  return (
    <div role="listbox" aria-label="Comandos disponíveis" className="overflow-hidden rounded-xl border border-neutral-200 bg-white p-1 shadow-lg dark:border-neutral-700 dark:bg-neutral-900">
      {commands.map((command, index) => {
        const active = index === activeIndex;
        return (
          <button
            key={command.id}
            type="button"
            role="option"
            aria-selected={active}
            onMouseEnter={() => onActiveIndexChange(index)}
            onClick={() => onSelect(command)}
            className={`flex w-full items-start gap-3 rounded-lg px-3 py-2 text-left transition ${active ? "bg-neutral-100 dark:bg-neutral-800" : "hover:bg-neutral-50 dark:hover:bg-neutral-800/60"}`}
          >
            <CommandLineIcon className="mt-0.5 h-4 w-4 shrink-0 text-violet-600 dark:text-violet-300" />
            <span className="min-w-0">
              <span className="block text-xs font-semibold text-neutral-900 dark:text-white">{command.label}</span>
              <span className="block text-[11px] leading-4 text-neutral-500 dark:text-neutral-400">{command.description}</span>
            </span>
          </button>
        );
      })}
    </div>
  );
}
