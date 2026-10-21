import { type FormEvent, useCallback, useEffect, useId, useState } from "react";
import {
  CheckCircleIcon,
  KeyIcon,
  PlusIcon,
} from "@heroicons/react/24/outline";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { SettingsTabs } from "@/components/SettingsTabs";
import { useT } from "@/i18n";
import { humanizeApiError } from "@/lib/userFacingError";
import {
  createProvider,
  listProviderModels,
  listProviders,
  mergeProviderPresets,
  parseModelIds,
  PROVIDER_PRESETS,
  removeProviderKey,
  saveProviderKey,
  saveProviderModels,
  sortProviders,
  staleModels,
  type ProviderModels,
  type ProviderPreset,
  type ProviderStatus,
} from "@/lib/providers";

function ModelPicker({
  name,
  onSaved,
  onClose,
}: {
  name: string;
  onSaved: () => void;
  onClose: () => void;
}) {
  const [models, setModels] = useState<ProviderModels | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { t } = useT();

  useEffect(() => {
    listProviderModels(name)
      .then((result) => {
        setModels(result);
        const available = new Set(result.available);
        // Start from configured models that still exist upstream.
        setSelected(
          new Set(
            result.configured.filter(
              (id) => result.available.length === 0 || available.has(id),
            ),
          ),
        );
      })
      .catch((err) =>
        setError(
          humanizeApiError(
            err,
            t("providers.models.queryError"),
          ).message,
        ),
      );
  }, [name, t]);

  if (!models && !error) {
    return (
      <p className="mt-3 text-xs text-neutral-500">
        {t("providers.models.loading")}
      </p>
    );
  }

  const stale = models ? staleModels(models) : [];
  const options = models
    ? Array.from(new Set([...models.available, ...models.configured])).filter(
        (id) => id.toLowerCase().includes(filter.trim().toLowerCase()),
      )
    : [];

  return (
    <div className="mt-3 rounded-xl bg-neutral-50 p-3 dark:bg-neutral-900">
      {models?.error && (
        <p role="alert" className="text-xs text-amber-700 dark:text-amber-300">
          {t("providers.models.listError", { message: models.error })}
        </p>
      )}
      {stale.length > 0 && (
        <p className="text-xs text-amber-700 dark:text-amber-300">
          {t("providers.models.stale", { models: stale.join(", ") })}
        </p>
      )}
      {models && models.available.length > 0 && (
        <input
          type="search"
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
          placeholder={t("providers.models.filterPlaceholder", {
            count: models.available.length,
          })}
          aria-label={t("providers.models.filterAria")}
          className="mt-2 w-full rounded-lg border border-neutral-300 bg-transparent px-3 py-1.5 text-xs outline-none dark:border-neutral-700"
        />
      )}
      <ul className="mt-2 max-h-60 space-y-1 overflow-y-auto">
        {options.map((id) => (
          <li key={id}>
            <label className="flex items-center gap-2 text-xs text-neutral-700 dark:text-neutral-200">
              <input
                type="checkbox"
                checked={selected.has(id)}
                onChange={(event) => {
                  const next = new Set(selected);
                  if (event.target.checked) next.add(id);
                  else next.delete(id);
                  setSelected(next);
                }}
              />
              <span className="truncate">{id}</span>
            </label>
          </li>
        ))}
      </ul>
      {error && (
        <p role="alert" className="mt-2 text-xs text-red-600 dark:text-red-400">
          {error}
        </p>
      )}
      <div className="mt-3 flex gap-2">
        <button
          type="button"
          disabled={busy || selected.size === 0}
          onClick={async () => {
            setBusy(true);
            setError(null);
            try {
              await saveProviderModels(name, [...selected]);
              onSaved();
            } catch (err) {
              setError(
                humanizeApiError(
                  err,
                  t("providers.models.saveError"),
                ).message,
              );
            } finally {
              setBusy(false);
            }
          }}
          className="rounded-lg bg-neutral-900 px-3 py-1.5 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900"
        >
          {busy
            ? t("providers.common.saving")
            : t("providers.models.use", { count: selected.size })}
        </button>
        <button
          type="button"
          onClick={onClose}
          className="rounded-lg px-3 py-1.5 text-xs text-neutral-500"
        >
          {t("providers.common.close")}
        </button>
      </div>
    </div>
  );
}

function ProviderCard({
  provider,
  onChanged,
}: {
  provider: ProviderStatus;
  onChanged: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [pickingModels, setPickingModels] = useState(false);
  const [key, setKey] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { t } = useT();

  const run = async (action: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      setKey("");
      setEditing(false);
      onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <li className="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="text-sm font-semibold text-neutral-900 dark:text-white">
            {provider.name}
          </h3>
          <p className="mt-0.5 truncate text-xs text-neutral-500">
            {provider.base_url}
          </p>
          <p className="mt-1 text-[11px] text-neutral-400">
            {provider.models}{" "}
            {t(
              provider.models === 1
                ? "providers.card.model.one"
                : "providers.card.model.other",
            )}{" "}
            · {provider.type}
            {provider.api_key_env ? ` · ${provider.api_key_env}` : ""}
          </p>
        </div>
        {provider.configured ? (
          <span className="flex shrink-0 items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
            <CheckCircleIcon className="h-4 w-4" />
            {provider.needs_key
              ? t("providers.card.keyConfigured")
              : t("providers.card.noKeyNeeded")}
          </span>
        ) : (
          <span className="shrink-0 rounded-full bg-amber-100 px-2 py-0.5 text-[11px] text-amber-800 dark:bg-amber-950/40 dark:text-amber-300">
            {t("providers.card.noKey")}
          </span>
        )}
      </div>

      {provider.needs_key && !editing && (
        <div className="mt-3 flex gap-2">
          <button
            type="button"
            onClick={() => setEditing(true)}
            className="flex items-center gap-1.5 rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
          >
            <KeyIcon className="h-4 w-4" />
            {provider.configured
              ? t("providers.card.replaceKey")
              : t("providers.card.addKey")}
          </button>
          {provider.configured && (
            <button
              type="button"
              onClick={() => setPickingModels((value) => !value)}
              className="rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              {t("providers.card.models", { count: provider.models })}
            </button>
          )}
          {provider.configured && (
            <button
              type="button"
              disabled={busy}
              onClick={() => void run(() => removeProviderKey(provider.name))}
              className="rounded-lg px-3 py-1.5 text-xs text-red-600 hover:bg-red-50 disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-950/30"
            >
              {t("providers.card.removeKey")}
            </button>
          )}
        </div>
      )}

      {editing && (
        <form
          className="mt-3 flex flex-wrap gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            void run(() => saveProviderKey(provider.name, key));
          }}
        >
          <input
            type="password"
            autoComplete="off"
            spellCheck={false}
            value={key}
            onChange={(event) => setKey(event.target.value)}
            placeholder={t("providers.card.keyPlaceholder", {
              provider: provider.name,
            })}
            aria-label={t("providers.card.keyAria", { provider: provider.name })}
            className="min-w-0 flex-1 rounded-lg border border-neutral-300 bg-transparent px-3 py-1.5 text-xs text-neutral-900 outline-none dark:border-neutral-700 dark:text-neutral-100"
          />
          <button
            type="submit"
            disabled={busy || !key.trim()}
            className="rounded-lg bg-neutral-900 px-3 py-1.5 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900"
          >
            {busy ? t("providers.common.saving") : t("providers.card.save")}
          </button>
          <button
            type="button"
            onClick={() => {
              setEditing(false);
              setKey("");
            }}
            className="rounded-lg px-3 py-1.5 text-xs text-neutral-500"
          >
            {t("providers.common.cancel")}
          </button>
        </form>
      )}
      {pickingModels && (
        <ModelPicker
          name={provider.name}
          onClose={() => setPickingModels(false)}
          onSaved={() => {
            setPickingModels(false);
            onChanged();
          }}
        />
      )}
      {error && (
        <p role="alert" className="mt-2 text-xs text-red-600 dark:text-red-400">
          {error}
        </p>
      )}
    </li>
  );
}

function AddProviderForm({
  presets,
  onCreated,
  onCancel,
}: {
  presets: ProviderPreset[];
  onCreated: (name: string) => void;
  onCancel: () => void;
}) {
  const options = presets.length > 0 ? presets : PROVIDER_PRESETS;
  const initial =
    options.find((preset) => preset.id === "openai") ?? options[0];
  const [presetId, setPresetId] = useState(initial?.id ?? "custom");
  const [name, setName] = useState(initial?.name ?? "openai");
  const [baseUrl, setBaseUrl] = useState(initial?.base_url ?? "");
  const [type, setType] = useState<string | undefined>(initial?.type);
  const [modelsText, setModelsText] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { t } = useT();

  const ids = useId();

  const selected = options.find((preset) => preset.id === presetId);

  const applyPreset = (id: string) => {
    setPresetId(id);
    setError(null);
    const preset = options.find((p) => p.id === id);
    if (!preset) return;
    setName(preset.name);
    setBaseUrl(preset.base_url);
    setType(preset.type);
  };

  const models = parseModelIds(modelsText);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError(null);
    if (!name.trim()) {
      setError(t("providers.form.nameRequired"));
      return;
    }
    if (!baseUrl.trim()) {
      setError(t("providers.form.urlRequired"));
      return;
    }
    if (models.length === 0) {
      setError(t("providers.form.modelsRequired"));
      return;
    }
    setBusy(true);
    try {
      await createProvider({
        name: name.trim(),
        base_url: baseUrl.trim(),
        type,
        models,
        api_key: apiKey.trim() || undefined,
      });
      onCreated(name.trim());
    } catch (err) {
      setError(
        humanizeApiError(err, t("providers.form.createError")).message,
      );
    } finally {
      setBusy(false);
    }
  };

  const field =
    "mt-1 w-full rounded-lg border border-neutral-300 bg-transparent px-3 py-2 text-sm text-neutral-900 outline-none focus:border-neutral-500 dark:border-neutral-700 dark:text-neutral-100";
  const labelClass =
    "block text-xs font-medium text-neutral-700 dark:text-neutral-300";

  return (
    <form
      onSubmit={submit}
      className="mt-6 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-950"
      aria-label={t("providers.form.aria")}
    >
      <h3 className="text-sm font-semibold text-neutral-900 dark:text-white">
        {t("providers.form.title")}
      </h3>
      <p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
        {t("providers.form.intro")}
      </p>

      <div className="mt-4 grid gap-4 sm:grid-cols-2">
        <div>
          <label htmlFor={`${ids}-preset`} className={labelClass}>
            {t("providers.form.service")}
          </label>
          <select
            id={`${ids}-preset`}
            value={presetId}
            onChange={(event) => applyPreset(event.target.value)}
            className={field}
          >
            {options.map((preset) => (
              <option key={preset.id} value={preset.id}>
                {preset.label}
              </option>
            ))}
          </select>
          {selected?.hint ? (
            <p className="mt-1 text-[11px] text-neutral-400">{selected.hint}</p>
          ) : null}
        </div>
        <div>
          <label htmlFor={`${ids}-name`} className={labelClass}>
            {t("providers.form.name")}
          </label>
          <input
            id={`${ids}-name`}
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder={t("providers.form.namePlaceholder")}
            className={field}
            autoComplete="off"
            spellCheck={false}
          />
        </div>
      </div>

      <div className="mt-4">
        <label htmlFor={`${ids}-url`} className={labelClass}>
          {t("providers.form.endpoint")}
        </label>
        <input
          id={`${ids}-url`}
          value={baseUrl}
          onChange={(event) => setBaseUrl(event.target.value)}
          placeholder={t("providers.form.urlPlaceholder")}
          className={field}
          autoComplete="off"
          spellCheck={false}
          inputMode="url"
        />
        <p className="mt-1 text-[11px] text-neutral-400">
          {t("providers.form.urlHint")}
        </p>
      </div>

      <div className="mt-4">
        <label htmlFor={`${ids}-models`} className={labelClass}>
          {t("providers.form.models")}
        </label>
        <textarea
          id={`${ids}-models`}
          value={modelsText}
          onChange={(event) => setModelsText(event.target.value)}
          placeholder={t("providers.form.modelsPlaceholder")}
          rows={3}
          className={field}
          spellCheck={false}
        />
        <p className="mt-1 text-[11px] text-neutral-400">
          {t("providers.form.modelsCount", { count: models.length })}
        </p>
      </div>

      <div className="mt-4">
        <label htmlFor={`${ids}-key`} className={labelClass}>
          {t("providers.form.key")}
        </label>
        <input
          id={`${ids}-key`}
          type="password"
          value={apiKey}
          onChange={(event) => setApiKey(event.target.value)}
          placeholder={t("providers.form.keyPlaceholder")}
          className={field}
          autoComplete="off"
          spellCheck={false}
        />
        <p className="mt-1 text-[11px] text-neutral-400">
          {t("providers.form.keyHint")}
        </p>
      </div>

      {error && (
        <p role="alert" className="mt-4 text-xs text-red-600 dark:text-red-400">
          {error}
        </p>
      )}

      <div className="mt-5 flex gap-2">
        <button
          type="submit"
          disabled={busy}
          className="rounded-lg bg-neutral-900 px-4 py-2 text-xs font-medium text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900"
        >
          {busy ? t("providers.form.submitting") : t("providers.form.submit")}
        </button>
        <button
          type="button"
          onClick={onCancel}
          disabled={busy}
          className="rounded-lg px-4 py-2 text-xs text-neutral-500 disabled:opacity-40"
        >
          {t("providers.common.cancel")}
        </button>
      </div>
    </form>
  );
}

export function ProvidersPage() {
  const { t } = useT();
  const [providers, setProviders] = useState<ProviderStatus[]>([]);
  const [configPath, setConfigPath] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  // Canonical guided presets come from the server (stage 4); PROVIDER_PRESETS
  // stays as the offline fallback for an older server.
  const [presets, setPresets] = useState<ProviderPreset[]>(PROVIDER_PRESETS);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await listProviders();
      setProviders(sortProviders(result.providers));
      setConfigPath(result.configPath);
      setPresets(mergeProviderPresets(result.presets));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const ready = providers.filter((provider) => provider.configured).length;

  return (
    <SidebarLayout
      title={t("providers.page.configTitle")}
      sidebar={<AppSidebar current="providers" />}
    >
      <SettingsTabs current="providers" />
      <div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-900">
        <div className="mx-auto w-full max-w-6xl px-6 pb-14 pt-10 lg:px-12">
          <h2 className="font-rounded text-3xl font-semibold tracking-tight text-neutral-950 dark:text-white">
            {t("providers.page.title")}
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-neutral-500 dark:text-neutral-400">
            {t("providers.page.intro")}
          </p>
          {!loading && !error && !adding && (
            <button
              type="button"
              onClick={() => {
                setNotice(null);
                setAdding(true);
              }}
              className="mt-4 inline-flex items-center gap-1.5 rounded-lg bg-neutral-900 px-3 py-2 text-xs font-medium text-white hover:bg-neutral-800 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200"
            >
              <PlusIcon className="h-4 w-4" />
              {t("providers.page.add")}
            </button>
          )}
          {adding && (
            <AddProviderForm
              presets={presets}
              onCancel={() => setAdding(false)}
              onCreated={(name) => {
                setAdding(false);
                setNotice(t("providers.page.created", { name }));
                void load();
              }}
            />
          )}
          {!loading && !error && (
            <p className="mt-3 text-xs text-neutral-500">
              {t("providers.page.ready", {
                ready,
                total: providers.length,
              })}
              {configPath ? ` · ${configPath}` : ""}
            </p>
          )}
          {notice && (
            <p
              role="status"
              className="mt-4 rounded-xl bg-emerald-50 px-4 py-2 text-xs text-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-300"
            >
              {notice}
            </p>
          )}
          {error && (
            <div
              role="alert"
              className="mt-6 rounded-2xl border border-red-900/40 p-5 text-sm text-red-700 dark:text-red-300"
            >
              <p>{error}</p>
              <button
                type="button"
                onClick={() => void load()}
                className="mt-3 rounded-lg border border-current px-3 py-1.5 text-xs"
              >
                {t("providers.page.retry")}
              </button>
            </div>
          )}
          {loading && (
            <p className="mt-8 text-sm text-neutral-500">
              {t("providers.page.loading")}
            </p>
          )}
          {!loading && !error && !adding && providers.length === 0 && (
            <div className="mt-8 max-w-2xl rounded-2xl border border-dashed border-neutral-300 p-6 text-sm leading-6 text-neutral-600 dark:border-neutral-700 dark:text-neutral-300">
              <p>{t("providers.page.empty.intro")}</p>
              <p className="mt-2 text-neutral-500 dark:text-neutral-400">
                {t("providers.page.empty.local")}
              </p>
            </div>
          )}
          <ul className="mt-6 grid gap-3 md:grid-cols-2">
            {providers.map((provider) => (
              <ProviderCard
                key={provider.name}
                provider={provider}
                onChanged={() => {
                  setNotice(t("providers.page.changed", { name: provider.name }));
                  void load();
                }}
              />
            ))}
          </ul>
        </div>
      </div>
    </SidebarLayout>
  );
}
