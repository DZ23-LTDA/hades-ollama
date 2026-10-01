import { useCallback, useEffect, useState } from "react";
import {
  listConnectors,
  listMCPServers,
  listSkills,
  removeConnector,
  removeMCP,
  removeSkill,
  setConnectorEnabled,
  setMCPEnabled,
  setSkillEnabled,
} from "@/lib/agenticClient";
import { buildManagedItems, type ManagedItem } from "@/lib/connectors";

const KIND_LABEL: Record<ManagedItem["kind"], string> = {
  connector: "Conectores registrados",
  mcp: "Servidores MCP",
  skill: "Skills",
};

// ConnectorsManagePanel lists everything registered in the agent runtime
// (connectors, MCP servers and skills) and lets the user enable, disable or
// remove each one. It replaces the separate "Plugins" management page.
export function ConnectorsManagePanel() {
  const [items, setItems] = useState<ManagedItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [connectors, mcp, skills] = await Promise.all([
        listConnectors(),
        listMCPServers(),
        listSkills(),
      ]);
      setItems(
        buildManagedItems(connectors.connectors, mcp.servers, skills.skills),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const act = async (item: ManagedItem, action: "toggle" | "remove") => {
    if (action === "remove" && !window.confirm(`Remover "${item.id}"?`)) return;
    setBusy(`${item.kind}:${item.id}`);
    setError(null);
    try {
      if (item.kind === "connector") {
        if (action === "remove") await removeConnector(item.id);
        else await setConnectorEnabled(item.id, !item.enabled);
      } else if (item.kind === "mcp") {
        if (action === "remove") await removeMCP(item.id, item.remote);
        else await setMCPEnabled(item.id, !item.enabled, item.remote);
      } else if (action === "remove") {
        await removeSkill(item.id);
      } else {
        await setSkillEnabled(item.id, !item.enabled);
      }
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(null);
    }
  };

  if (loading) {
    return (
      <p className="mt-8 text-sm text-neutral-500" role="status">
        Carregando itens registrados…
      </p>
    );
  }

  return (
    <div className="mt-6 space-y-6">
      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-900/40 p-4 text-sm text-red-700 dark:text-red-300"
        >
          {error}
        </p>
      )}
      {(["connector", "mcp", "skill"] as const).map((kind) => {
        const group = items.filter((item) => item.kind === kind);
        return (
          <section key={kind}>
            <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-neutral-400">
              {KIND_LABEL[kind]} ({group.length})
            </h3>
            {group.length === 0 ? (
              <p className="text-sm text-neutral-500">Nada registrado.</p>
            ) : (
              <ul className="space-y-2">
                {group.map((item) => {
                  const key = `${item.kind}:${item.id}`;
                  return (
                    <li
                      key={key}
                      className="flex items-center justify-between gap-3 rounded-xl border border-neutral-200 bg-white px-4 py-3 dark:border-neutral-800 dark:bg-neutral-950"
                    >
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-neutral-900 dark:text-white">
                          {item.id}
                        </p>
                        <p className="truncate text-xs text-neutral-500">
                          {item.detail}
                        </p>
                      </div>
                      <div className="flex shrink-0 items-center gap-2">
                        <span
                          className={`rounded-full px-2 py-0.5 text-[11px] ${
                            item.enabled
                              ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300"
                              : "bg-neutral-200 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400"
                          }`}
                        >
                          {item.enabled ? "Ativo" : "Desativado"}
                        </span>
                        <button
                          type="button"
                          disabled={busy === key}
                          onClick={() => void act(item, "toggle")}
                          className="rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-700 hover:bg-neutral-100 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
                        >
                          {item.enabled ? "Desativar" : "Ativar"}
                        </button>
                        <button
                          type="button"
                          disabled={busy === key}
                          onClick={() => void act(item, "remove")}
                          className="rounded-lg px-3 py-1.5 text-xs text-red-600 hover:bg-red-50 disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-950/30"
                        >
                          Remover
                        </button>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
          </section>
        );
      })}
      <p className="text-xs text-neutral-500">
        Para registrar um conector, servidor MCP ou skill manualmente, use o{" "}
        <a
          href="/plugins"
          className="font-medium text-violet-600 hover:underline dark:text-violet-300"
        >
          registro avançado
        </a>
        .
      </p>
    </div>
  );
}
