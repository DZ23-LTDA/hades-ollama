import { useCallback, useEffect, useState } from "react";
import { CheckCircleIcon, KeyIcon } from "@heroicons/react/24/outline";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { SettingsTabs } from "@/components/SettingsTabs";
import {
  listProviderModels,
  listProviders,
  removeProviderKey,
  saveProviderKey,
  saveProviderModels,
  sortProviders,
  staleModels,
  type ProviderModels,
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
        setError(err instanceof Error ? err.message : String(err)),
      );
  }, [name]);

  if (!models && !error) {
    return (
      <p className="mt-3 text-xs text-neutral-500">
        Consultando modelos do provedor…
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
          Não foi possível listar os modelos: {models.error}
        </p>
      )}
      {stale.length > 0 && (
        <p className="text-xs text-amber-700 dark:text-amber-300">
          Não existem mais no provedor e foram desmarcados: {stale.join(", ")}
        </p>
      )}
      {models && models.available.length > 0 && (
        <input
          type="search"
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
          placeholder={`Filtrar ${models.available.length} modelos`}
          aria-label="Filtrar modelos"
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
              setError(err instanceof Error ? err.message : String(err));
            } finally {
              setBusy(false);
            }
          }}
          className="rounded-lg bg-neutral-900 px-3 py-1.5 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900"
        >
          {busy ? "Salvando…" : `Usar ${selected.size} modelo(s)`}
        </button>
        <button
          type="button"
          onClick={onClose}
          className="rounded-lg px-3 py-1.5 text-xs text-neutral-500"
        >
          Fechar
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
            {provider.models} {provider.models === 1 ? "modelo" : "modelos"} ·{" "}
            {provider.type}
            {provider.api_key_env ? ` · ${provider.api_key_env}` : ""}
          </p>
        </div>
        {provider.configured ? (
          <span className="flex shrink-0 items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
            <CheckCircleIcon className="h-4 w-4" />
            {provider.needs_key ? "Chave configurada" : "Sem chave necessária"}
          </span>
        ) : (
          <span className="shrink-0 rounded-full bg-amber-100 px-2 py-0.5 text-[11px] text-amber-800 dark:bg-amber-950/40 dark:text-amber-300">
            Sem chave
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
            {provider.configured ? "Trocar chave" : "Adicionar chave"}
          </button>
          {provider.configured && (
            <button
              type="button"
              onClick={() => setPickingModels((value) => !value)}
              className="rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              Modelos ({provider.models})
            </button>
          )}
          {provider.configured && (
            <button
              type="button"
              disabled={busy}
              onClick={() => void run(() => removeProviderKey(provider.name))}
              className="rounded-lg px-3 py-1.5 text-xs text-red-600 hover:bg-red-50 disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-950/30"
            >
              Remover chave
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
            placeholder={`Cole a chave de ${provider.name}`}
            aria-label={`Chave de API de ${provider.name}`}
            className="min-w-0 flex-1 rounded-lg border border-neutral-300 bg-transparent px-3 py-1.5 text-xs text-neutral-900 outline-none dark:border-neutral-700 dark:text-neutral-100"
          />
          <button
            type="submit"
            disabled={busy || !key.trim()}
            className="rounded-lg bg-neutral-900 px-3 py-1.5 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900"
          >
            {busy ? "Salvando…" : "Salvar"}
          </button>
          <button
            type="button"
            onClick={() => {
              setEditing(false);
              setKey("");
            }}
            className="rounded-lg px-3 py-1.5 text-xs text-neutral-500"
          >
            Cancelar
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

export function ProvidersPage() {
  const [providers, setProviders] = useState<ProviderStatus[]>([]);
  const [configPath, setConfigPath] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await listProviders();
      setProviders(sortProviders(result.providers));
      setConfigPath(result.configPath);
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
      title="Configurações"
      sidebar={<AppSidebar current="providers" />}
    >
      <SettingsTabs current="providers" />
      <div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-900">
        <div className="mx-auto w-full max-w-6xl px-6 pb-14 pt-10 lg:px-12">
          <h2 className="font-rounded text-3xl font-semibold tracking-tight text-neutral-950 dark:text-white">
            Provedores de IA
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-neutral-500 dark:text-neutral-400">
            Modelos de outros provedores usados pelo roteador, ao lado dos
            modelos locais e do Ollama Cloud. As chaves ficam protegidas no seu
            usuário do sistema e nunca voltam para esta tela.
          </p>
          {!loading && !error && (
            <p className="mt-3 text-xs text-neutral-500">
              {ready} de {providers.length} provedores prontos
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
                Tentar novamente
              </button>
            </div>
          )}
          {loading && (
            <p className="mt-8 text-sm text-neutral-500">
              Carregando provedores…
            </p>
          )}
          {!loading && !error && providers.length === 0 && (
            <p className="mt-8 text-sm text-neutral-500">
              Nenhum provedor configurado. Defina OLLAMA_DZ23_CONFIG com um
              arquivo de provedores.
            </p>
          )}
          <ul className="mt-6 grid gap-3 md:grid-cols-2">
            {providers.map((provider) => (
              <ProviderCard
                key={provider.name}
                provider={provider}
                onChanged={() => {
                  setNotice(
                    `${provider.name}: alteração salva. O motor foi reiniciado para aplicar.`,
                  );
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
