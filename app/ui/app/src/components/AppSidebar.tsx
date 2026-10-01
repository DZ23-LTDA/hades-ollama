import { Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { ThemeSwitcher } from "@/components/ThemeSwitcher";
import HadesEmblem from "@/components/HadesEmblem";
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
  SparklesIcon,
  UserIcon,
  ArrowRightOnRectangleIcon,
  CommandLineIcon,
  BellIcon,
  PencilSquareIcon,
} from "@heroicons/react/24/outline";
import { ChatIcon } from "@/components/ChatIcon";
import { SearchDialog } from "@/components/SearchDialog";
import { HelpDialog } from "@/components/HelpDialog";
import { newTaskShortcut } from "@/lib/help";
import { SETTINGS_SECTIONS } from "@/lib/settingsTabs";
import { isTypingTarget } from "@/lib/search";
import { disconnectUser, fetchUser } from "@/api";
import { clearAgentSession } from "@/lib/agenticClient";

export type AppSection =
  | "apps"
  | "chat"
  | "agentic"
  | "settings"
  | "library"
  | "creations"
  | "studio"
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

export async function performLogout(): Promise<void> {
  await disconnectUser();
  clearAgentSession();
}

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
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [shortcutsOpen, setShortcutsOpen] = useState(false);
  const [signOutOpen, setSignOutOpen] = useState(false);
  const [signOutPending, setSignOutPending] = useState(false);
  const [signOutError, setSignOutError] = useState<string | null>(null);
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [userProfile, setUserProfile] = useState<{
    name: string;
    username: string;
    email: string;
    plan: string;
  }>({
    name: "Operador local",
    username: "local",
    email: "local@localhost",
    plan: "Local-first",
  });

  useEffect(() => {
    fetchUser()
      .then((data) => {
        if (data && data.name) {
          setUserProfile({
            name: data.name || "Operador local",
            username: data.name || "local",
            email: data.email || "local@localhost",
            plan: data.plan || "Local-first",
          });
        }
      })
      .catch(() => {});
  }, []);

  const handleSignOut = async () => {
    setSignOutPending(true);
    setSignOutError(null);
    try {
      await performLogout();
      setSignOutOpen(false);
      window.location.reload();
    } catch (error) {
      setSignOutError(error instanceof Error ? error.message : "Não foi possível encerrar a sessão.");
    } finally {
      setSignOutPending(false);
    }
  };
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
        <HadesEmblem
          size={22}
          className="shrink-0 text-neutral-900 dark:text-white"
        />
        <div className="min-w-0">
          <div className="truncate text-[12px] font-semibold text-neutral-900 dark:text-white">
            Hades
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
      <TargetLink
        href="/endpoint"
        label="Computadores"
        current={current}
        section="endpoint"
        icon={ComputerDesktopIcon}
      />
      <Link
        to="/agentic"
        className={itemClass(current === "agentic")}
        draggable={false}
      >
        <BoltIcon className={iconClass} />
        <span className="min-w-0 flex-1 truncate">Agents</span>
        <span className="rounded bg-neutral-200 px-1 py-0.5 text-[9px] font-semibold text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300">Cue!</span>
        <span
          className="h-1.5 w-1.5 rounded-full bg-emerald-500"
          title="Runtime local"
        />
      </Link>
      <TargetLink
        href="/library"
        label="Biblioteca"
        current={current}
        section="library"
        icon={BookOpenIcon}
      />
      <TargetLink
        href="/creations"
        label="Criações"
        current={current}
        section="creations"
        icon={SparklesIcon}
      />
      <TargetLink
        href="/studio"
        label="Studio"
        current={current}
        section="studio"
        icon={PencilSquareIcon}
      />
      <TargetLink
        href="/scheduled"
        label="Automações"
        current={current}
        section="scheduled"
        icon={ClockIcon}
      />
      <TargetLink
        href="/connectors"
        label="Plugins"
        current={current}
        section="connectors"
        icon={LinkIcon}
      />

      <NavLabel>Ferramentas</NavLabel>
      <TargetLink
        href="/skills"
        label="Habilidades"
        current={current}
        section="skills"
        icon={BoltIcon}
      />
      <TargetLink
        href="/company"
        label="Empresa"
        current={current}
        section="company"
        icon={BuildingOffice2Icon}
        badge="Novo"
      />
      <TargetLink
        href="/tasks"
        label="Tarefas"
        current={current}
        section="tasks"
        icon={ArrowPathIcon}
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
        {/* Popover do Perfil (Abre para cima) */}
        <div className="relative">
          {userMenuOpen && (
            <div className="absolute bottom-full left-0 mb-2 w-64 rounded-2xl border border-neutral-200 bg-white p-3 shadow-xl dark:border-neutral-800 dark:bg-neutral-900 z-50">
              <div className="flex items-center justify-between border-b border-neutral-100 pb-3 dark:border-neutral-800">
                <div className="flex items-center gap-2.5">
                  <div className="flex h-9 w-9 items-center justify-center rounded-full bg-emerald-600 font-bold text-white text-sm">
                    {userProfile.name.charAt(0).toUpperCase()}
                  </div>
                  <div>
                    <div className="text-xs font-bold text-neutral-900 dark:text-white">{userProfile.name}</div>
                    <div className="text-[10px] text-neutral-400">Pessoal</div>
                  </div>
                </div>
                <span className="rounded bg-neutral-100 px-1.5 py-0.5 text-[10px] font-semibold text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300">
                  {userProfile.plan}
                </span>
              </div>

              <div className="py-2 border-b border-neutral-100 dark:border-neutral-800 flex items-center justify-between text-xs">
                <span className="text-neutral-500">Créditos:</span>
                <span className="font-semibold text-emerald-600 dark:text-emerald-400">Ilimitado (Local)</span>
              </div>

              <div className="py-2 space-y-1">
                <a
                  href="/settings#account"
                  onClick={() => setUserMenuOpen(false)}
                  className="flex items-center gap-2 rounded-lg px-2 py-1.5 text-xs text-neutral-700 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <UserIcon className="h-4 w-4 text-neutral-400" />
                  Conta
                </a>
                <a
                  href="/settings#personalization"
                  onClick={() => setUserMenuOpen(false)}
                  className="flex items-center gap-2 rounded-lg px-2 py-1.5 text-xs text-neutral-700 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <SparklesIcon className="h-4 w-4 text-neutral-400" />
                  Personalização
                </a>
                <a
                  href="/settings"
                  onClick={() => setUserMenuOpen(false)}
                  className="flex items-center gap-2 rounded-lg px-2 py-1.5 text-xs text-neutral-700 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <Cog6ToothIcon className="h-4 w-4 text-neutral-400" />
                  Configurações
                </a>
                <button
                  type="button"
                  onClick={() => {
                    setUserMenuOpen(false);
                    setShortcutsOpen(true);
                  }}
                  className="w-full flex items-center gap-2 rounded-lg px-2 py-1.5 text-xs text-neutral-700 hover:bg-neutral-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
                >
                  <CommandLineIcon className="h-4 w-4 text-neutral-400" />
                  Atalhos de teclado
                </button>
              </div>

              <div className="pt-2 border-t border-neutral-100 dark:border-neutral-800">
                <button
                  type="button"
                  onClick={() => {
                    setUserMenuOpen(false);
                    setSignOutOpen(true);
                  }}
                  className="w-full flex items-center gap-2 rounded-lg px-2 py-1.5 text-xs text-red-600 hover:bg-red-50 dark:hover:bg-red-950/30"
                >
                  <ArrowRightOnRectangleIcon className="h-4 w-4" />
                  Sair
                </button>
              </div>
            </div>
          )}

          {/* Barra de Perfil no Rodapé */}
          <div className="flex items-center justify-between">
            <button
              type="button"
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="flex items-center gap-2 rounded-xl p-1 text-left transition-colors hover:bg-neutral-200/60 dark:hover:bg-neutral-800"
            >
              <div className="relative flex h-7 w-7 items-center justify-center rounded-full bg-emerald-600 font-bold text-white text-xs">
                {userProfile.name.charAt(0).toUpperCase()}
                <span className="absolute bottom-0 right-0 h-2 w-2 rounded-full bg-emerald-400 ring-2 ring-white dark:ring-neutral-900" />
              </div>
              <span className="truncate text-xs font-semibold text-neutral-800 dark:text-neutral-200">
                {userProfile.name}
              </span>
            </button>
            <div className="flex items-center gap-1 text-neutral-400">
              <button
                type="button"
                onClick={() => setNotificationsOpen((open) => !open)}
                aria-expanded={notificationsOpen}
                className="rounded-lg p-1 hover:text-neutral-900 dark:hover:text-white"
                title="Notificações"
              >
                <BellIcon className="h-4 w-4" />
              </button>
              <ThemeSwitcher />
            </div>
          </div>
        </div>
      </div>

      {notificationsOpen && (
        <div role="status" aria-live="polite" className="fixed bottom-16 right-4 z-50 w-72 rounded-2xl border border-neutral-200 bg-white p-4 shadow-xl dark:border-neutral-800 dark:bg-neutral-900">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-neutral-900 dark:text-white">Notificações</h3>
            <button type="button" aria-label="Fechar notificações" onClick={() => setNotificationsOpen(false)} className="rounded p-1 text-neutral-400 hover:bg-neutral-100 dark:hover:bg-neutral-800">×</button>
          </div>
          <p className="mt-3 text-xs leading-5 text-neutral-500 dark:text-neutral-400">Nenhuma notificação persistida para este workspace.</p>
          <p className="mt-2 text-[10px] text-neutral-400">Eventos de missões e approvals aparecem aqui quando o backend registrar uma notificação.</p>
        </div>
      )}
      {/* Modal de Atalhos de Teclado */}
      {shortcutsOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
          <div className="w-full max-w-md rounded-2xl border border-neutral-200 bg-white p-6 shadow-2xl dark:border-neutral-800 dark:bg-neutral-900">
            <div className="flex items-center justify-between pb-3 border-b border-neutral-100 dark:border-neutral-800">
              <h3 className="text-base font-bold text-neutral-900 dark:text-white">Atalhos de Teclado</h3>
              <button onClick={() => setShortcutsOpen(false)} className="text-neutral-400 hover:text-neutral-600">✕</button>
            </div>
            <div className="mt-4 space-y-3 text-xs">
              <div className="flex items-center justify-between">
                <span className="text-neutral-600 dark:text-neutral-300">Nova tarefa</span>
                <kbd className="rounded bg-neutral-100 px-2 py-1 font-mono dark:bg-neutral-800">Ctrl + ⇧ + O</kbd>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-neutral-600 dark:text-neutral-300">Pesquisar</span>
                <kbd className="rounded bg-neutral-100 px-2 py-1 font-mono dark:bg-neutral-800">/</kbd>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-neutral-600 dark:text-neutral-300">Novo Chat rápido</span>
                <kbd className="rounded bg-neutral-100 px-2 py-1 font-mono dark:bg-neutral-800">Ctrl + K</kbd>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-neutral-600 dark:text-neutral-300">Enviar e iniciar missão</span>
                <kbd className="rounded bg-neutral-100 px-2 py-1 font-mono dark:bg-neutral-800">Ctrl + Enter</kbd>
              </div>
            </div>
            <div className="mt-6 flex justify-end">
              <button onClick={() => setShortcutsOpen(false)} className="rounded-xl bg-neutral-900 px-4 py-2 text-xs font-semibold text-white dark:bg-white dark:text-neutral-900">
                Fechar
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Modal de Confirmação de Saída */}
      {signOutOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
          <div className="w-full max-w-sm rounded-2xl border border-neutral-200 bg-white p-6 shadow-2xl dark:border-neutral-800 dark:bg-neutral-900 text-center">
            <h3 className="text-base font-bold text-neutral-900 dark:text-white">Tem certeza de que deseja sair?</h3>
            <p className="mt-2 text-xs text-neutral-500">Sair do Hades como {userProfile.email}?</p>
            {signOutError && <p role="alert" className="mt-3 text-xs text-red-600">{signOutError}</p>}
            <div className="mt-6 flex items-center justify-center gap-3">
              <button type="button" disabled={signOutPending} onClick={() => setSignOutOpen(false)} className="flex-1 rounded-xl border border-neutral-200 py-2 text-xs font-semibold text-neutral-700 hover:bg-neutral-50 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200">
                Manter-se conectado
              </button>
              <button type="button" disabled={signOutPending} onClick={() => void handleSignOut()} className="flex-1 rounded-xl bg-red-600 py-2 text-xs font-semibold text-white hover:bg-red-700 disabled:opacity-50">
                {signOutPending ? "Saindo…" : "Sair"}
              </button>
            </div>
          </div>
        </div>
      )}
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
