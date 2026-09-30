import { Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { ThemeSwitcher } from "@/components/ThemeSwitcher";
import {
  ArrowPathIcon,
  BookOpenIcon,
  BuildingOffice2Icon,
  BoltIcon,
  ClockIcon,
  ComputerDesktopIcon,
  Cog6ToothIcon,
  FolderIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  LinkIcon,
  Squares2X2Icon,
} from "@heroicons/react/24/outline";
import { ChatIcon } from "@/components/ChatIcon";
import { SearchDialog } from "@/components/SearchDialog";
import { HelpDialog } from "@/components/HelpDialog";
import { newTaskShortcut } from "@/lib/help";
import { SETTINGS_SECTIONS } from "@/lib/settingsTabs";
import { isTypingTarget } from "@/lib/search";

export type AppSection =
  | "apps"
  | "chat"
  | "agentic"
  | "settings"
  | "library"
  | "projects"
  | "scheduled"
  | "skills"
  | "plugins"
  | "connectors"
  | "providers"
  | "endpoint"
  | "tasks"
  | "company";

type Icon = React.ComponentType<{ className?: string }>;

const iconClass = "h-[18px] w-[18px] shrink-0 stroke-[1.7]";

function itemClass(active: boolean, prominent = false) {
  return `group flex w-full items-center gap-3 rounded-lg px-2.5 py-2 text-left text-[13px] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-400 ${
    prominent
      ? "bg-neutral-900 text-white shadow-sm hover:bg-neutral-700 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200"
      : active
        ? "bg-neutral-200/80 text-neutral-950 dark:bg-neutral-800 dark:text-white"
        : "text-neutral-600 hover:bg-neutral-200/60 hover:text-neutral-950 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-white"
  }`;
}

function NavLabel({ children }: { children: React.ReactNode }) {
  return (
    <div className="px-2.5 pb-1 pt-4 text-[10px] font-semibold uppercase tracking-[0.14em] text-neutral-400 dark:text-neutral-600">
      {children}
    </div>
  );
}

function TargetLink({
  href,
  label,
  current,
  section,
  icon: IconComponent,
  badge,
}: {
  href: string;
  label: string;
  current: AppSection;
  section: AppSection;
  icon: Icon;
  badge?: string;
}) {
  return (
    <a
      href={href}
      className={itemClass(current === section)}
      aria-current={current === section ? "page" : undefined}
    >
      <IconComponent className={iconClass} />
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {badge && (
        <span className="rounded-full bg-violet-100 px-1.5 py-0.5 text-[9px] font-semibold text-violet-700 dark:bg-violet-950/60 dark:text-violet-300">
          {badge}
        </span>
      )}
    </a>
  );
}

export function AppNavigation({ current }: { current: AppSection }) {
  const [searchOpen, setSearchOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        window.location.assign("/c/new");
        return;
      }
      if (
        event.key === "/" &&
        !event.metaKey &&
        !event.ctrlKey &&
        !event.altKey &&
        !isTypingTarget(event.target)
      ) {
        event.preventDefault();
        setSearchOpen(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  return (
    <div className="flex flex-col gap-0.5">
      <SearchDialog open={searchOpen} onClose={() => setSearchOpen(false)} />
      <HelpDialog open={helpOpen} onClose={() => setHelpOpen(false)} />
      <div className="mb-3 flex items-center gap-2 px-2.5 pt-1">
        <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-neutral-900 text-xs font-semibold text-white dark:bg-white dark:text-neutral-900">
          OF
        </div>
        <div className="min-w-0">
          <div className="truncate text-[12px] font-semibold text-neutral-900 dark:text-white">
            Ollama Full
          </div>
          <div className="truncate text-[10px] text-neutral-400">
            Local-first workspace
          </div>
        </div>
      </div>

      <Link
        to="/c/$chatId"
        params={{ chatId: "new" }}
        mask={{ to: "/" }}
        className={itemClass(false, true)}
        draggable={false}
      >
        <PlusIcon className={iconClass} />
        <span>Nova tarefa</span>
        <span className="ml-auto text-[10px] text-white/60 dark:text-neutral-500">
          {newTaskShortcut()}
        </span>
      </Link>

      <button
        type="button"
        onClick={() => setSearchOpen(true)}
        className={itemClass(false)}
      >
        <MagnifyingGlassIcon className={iconClass} />
        <span className="min-w-0 flex-1 truncate">Pesquisar</span>
        <span className="text-[10px] text-neutral-400">/</span>
      </button>

      <NavLabel>Agentes</NavLabel>
      <Link
        to="/agentic"
        className={itemClass(current === "agentic")}
        draggable={false}
      >
        <BoltIcon className={iconClass} />
        <span className="min-w-0 flex-1 truncate">Agente</span>
        <span
          className="h-1.5 w-1.5 rounded-full bg-emerald-500"
          title="Runtime local"
        />
      </Link>
      <TargetLink
        href="/endpoint"
        label="Computadores"
        current={current}
        section="endpoint"
        icon={ComputerDesktopIcon}
      />
      <TargetLink
        href="/tasks"
        label="Tarefas"
        current={current}
        section="tasks"
        icon={ArrowPathIcon}
      />
      <TargetLink
        href="/scheduled"
        label="Automações"
        current={current}
        section="scheduled"
        icon={ClockIcon}
      />
      <TargetLink
        href="/company"
        label="Empresa"
        current={current}
        section="company"
        icon={BuildingOffice2Icon}
        badge="Novo"
      />

      <NavLabel>Ferramentas</NavLabel>
      <TargetLink
        href="/connectors"
        label="Plugins / Conectores"
        current={current}
        section="connectors"
        icon={LinkIcon}
      />
      <TargetLink
        href="/skills"
        label="Habilidades"
        current={current}
        section="skills"
        icon={BoltIcon}
      />
      <TargetLink
        href="/library"
        label="Biblioteca"
        current={current}
        section="library"
        icon={BookOpenIcon}
      />

      <div className="mt-2 flex items-center justify-between px-2.5 pt-2">
        <NavLabel>Projetos</NavLabel>
        <a
          href="/projects"
          aria-label="Novo projeto"
          title="Novo projeto"
          className="rounded-md p-1 text-neutral-400 hover:bg-neutral-200 hover:text-neutral-900 dark:hover:bg-neutral-800 dark:hover:text-white"
        >
          <PlusIcon className="h-4 w-4" />
        </a>
      </div>
      <TargetLink
        href="/projects"
        label="Todos os projetos"
        current={current}
        section="projects"
        icon={FolderIcon}
      />

      <NavLabel>Sistema</NavLabel>
      <Link
        to="/settings"
        className={itemClass(SETTINGS_SECTIONS.has(current))}
        draggable={false}
      >
        <Cog6ToothIcon className={iconClass} />
        <span className="min-w-0 flex-1 truncate">Configurações</span>
      </Link>
      <button
        type="button"
        onClick={() => setHelpOpen(true)}
        className={itemClass(false)}
      >
        <Squares2X2Icon className={iconClass} />
        <span className="min-w-0 flex-1 truncate">Ajuda e sobre</span>
      </button>

      <div className="mt-auto border-t border-neutral-200/80 px-2.5 pt-3 dark:border-neutral-800">
        <div className="flex items-center justify-between gap-2">
          <div className="text-[10px] leading-4 text-neutral-400">
            <div className="font-medium text-neutral-500 dark:text-neutral-500">
              Modo local-first
            </div>
            <div>Approvals e secrets protegidos</div>
          </div>
          <ThemeSwitcher />
        </div>
      </div>
    </div>
  );
}

export function AppSidebar({ current }: { current: AppSection }) {
  return (
    <nav className="flex flex-1 flex-col overflow-y-auto px-3 pb-4 select-none">
      <AppNavigation current={current} />
    </nav>
  );
}

export function ChatNavigationShortcut() {
  return (
    <Link
      to="/c/$chatId"
      params={{ chatId: "new" }}
      mask={{ to: "/" }}
      className="flex items-center gap-2 rounded-md px-2 py-1.5 text-xs text-neutral-500 hover:bg-neutral-100 dark:hover:bg-neutral-800"
    >
      <ChatIcon />
      Chat
    </Link>
  );
}
