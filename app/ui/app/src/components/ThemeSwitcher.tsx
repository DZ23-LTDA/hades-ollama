import {
  ComputerDesktopIcon,
  MoonIcon,
  SunIcon,
} from "@heroicons/react/24/outline";
import { useTheme, type ThemePreference } from "@/lib/theme";

const OPTIONS: {
  value: ThemePreference;
  label: string;
  Icon: typeof SunIcon;
}[] = [
  { value: "light", label: "Claro", Icon: SunIcon },
  { value: "dark", label: "Escuro", Icon: MoonIcon },
  { value: "auto", label: "Automático", Icon: ComputerDesktopIcon },
];

/**
 * Controle de tema Claro/Escuro/Automático (paridade com o Manus, identidade
 * Ollama). `full` = cartões grandes (Configurações › Aparência); `compact` =
 * segmentado só-ícone (rodapé da sidebar).
 */
export function ThemeSwitcher({
  variant = "compact",
}: {
  variant?: "compact" | "full";
}) {
  const { theme, setTheme } = useTheme();

  if (variant === "full") {
    return (
      <div role="radiogroup" aria-label="Tema" className="grid grid-cols-3 gap-2">
        {OPTIONS.map(({ value, label, Icon }) => {
          const active = theme === value;
          return (
            <button
              key={value}
              type="button"
              role="radio"
              aria-checked={active}
              onClick={() => setTheme(value)}
              className={`flex flex-col items-center gap-2 rounded-xl border px-3 py-4 text-xs font-medium transition ${active ? "border-neutral-900 bg-neutral-50 text-neutral-900 dark:border-white dark:bg-neutral-800 dark:text-white" : "border-neutral-200 text-neutral-500 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-400 dark:hover:bg-neutral-800/50"}`}
            >
              <Icon className="h-5 w-5" />
              {label}
            </button>
          );
        })}
      </div>
    );
  }

  return (
    <div
      role="radiogroup"
      aria-label="Tema"
      className="inline-flex rounded-lg border border-neutral-200 p-0.5 dark:border-neutral-700"
    >
      {OPTIONS.map(({ value, label, Icon }) => {
        const active = theme === value;
        return (
          <button
            key={value}
            type="button"
            role="radio"
            aria-checked={active}
            title={label}
            aria-label={label}
            onClick={() => setTheme(value)}
            className={`flex h-7 w-7 items-center justify-center rounded-md transition ${active ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "text-neutral-400 hover:text-neutral-700 dark:hover:text-neutral-200"}`}
          >
            <Icon className="h-4 w-4" />
          </button>
        );
      })}
    </div>
  );
}
