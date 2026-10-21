import { useT, type Language } from "@/i18n";
import { changeLanguage, type LanguageStorage } from "@/i18n/persistencia";

// Controlo de idioma da interface dentro de Settings.
//
// Fica ao lado do "Modo da interface" porque é o mesmo tipo de preferência:
// local ao browser (localStorage), sem backend pelo meio e visível mesmo quando
// a configuração nativa está indisponível. Segue o padrão visual/ARIA desse
// controlo (secção com heading + grupo de botões `aria-pressed`).

const OPTIONS: ReadonlyArray<{ id: Language; labelKey: string }> = [
  { id: "pt-BR", labelKey: "settings.language.pt-BR" },
  { id: "en", labelKey: "settings.language.en" },
];

export function SettingsLanguageControl({
  storage,
}: {
  storage?: LanguageStorage;
}) {
  const { t, language } = useT();
  return (
    <section
      aria-labelledby="settings-language-heading"
      className="rounded-xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-900/60 dark:bg-blue-950/20"
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2
            id="settings-language-heading"
            className="text-sm font-semibold text-blue-950 dark:text-blue-100"
          >
            {t("settings.language.title")}
          </h2>
          <p className="mt-1 text-xs leading-5 text-blue-800 dark:text-blue-200">
            {t("settings.language.description")}
          </p>
        </div>
        <div
          className="inline-flex rounded-lg border border-blue-200 bg-white p-1 dark:border-blue-800 dark:bg-neutral-900"
          role="group"
          aria-label={t("settings.language.title")}
        >
          {OPTIONS.map((option) => (
            <button
              key={option.id}
              type="button"
              lang={option.id}
              aria-pressed={language === option.id}
              onClick={() => changeLanguage(option.id, storage)}
              className={`min-h-9 rounded-md px-3 text-xs font-medium ${
                language === option.id
                  ? "bg-blue-700 text-white"
                  : "text-blue-800 dark:text-blue-200"
              }`}
            >
              {t(option.labelKey)}
            </button>
          ))}
        </div>
      </div>
    </section>
  );
}
