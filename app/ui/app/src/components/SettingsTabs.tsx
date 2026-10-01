import { SETTINGS_TABS, type SettingsTabId } from "@/lib/settingsTabs";

// SettingsTabs groups every configuration screen under Configurações so they
// are found in one place instead of scattered across the sidebar.
export function SettingsTabs({ current }: { current: SettingsTabId }) {
  return (
    <nav
      aria-label="Seções de configurações"
      className="border-b border-neutral-200 bg-white px-6 dark:border-neutral-800 dark:bg-neutral-900 lg:px-12"
    >
      <ul className="mx-auto flex max-w-6xl gap-1 overflow-x-auto">
        {SETTINGS_TABS.map((tab) => {
          const active = tab.id === current;
          return (
            <li key={tab.id}>
              <a
                href={tab.href}
                aria-current={active ? "page" : undefined}
                className={`block whitespace-nowrap border-b-2 px-3 py-3 text-sm transition-colors ${
                  active
                    ? "border-neutral-900 font-medium text-neutral-900 dark:border-white dark:text-white"
                    : "border-transparent text-neutral-500 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-white"
                }`}
              >
                {tab.label}
              </a>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
