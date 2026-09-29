import { useCallback, useEffect, useMemo, useState } from "react";
import {
  CheckIcon,
  DocumentTextIcon,
  MagnifyingGlassIcon,
} from "@heroicons/react/24/outline";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { connectorIcon } from "@/lib/connectorIcons";
import { ConnectorsManagePanel } from "@/components/ConnectorsManagePanel";
import { ConnectorQuickConnect } from "@/components/ConnectorQuickConnect";
import {
  listConnectorCatalog,
  listConnectors,
  listMCPServers,
  type AgentConnector,
  type AgentConnectorCatalogEntry,
  type AgentMCPServer,
} from "@/lib/agenticClient";
import {
  connectorState,
  filterConnectors,
  type ConnectorTab,
} from "@/lib/connectors";

const TABS: Array<{ id: ConnectorTab; label: string }> = [
  { id: "app", label: "Aplicativos" },
  { id: "custom_api", label: "API personalizada" },
  { id: "mcp", label: "MCP personalizado" },
  { id: "connected", label: "Gerenciar" },
];

function ConnectorLogo({ id }: { id: string }) {
  const icon = connectorIcon(id);
  return (
    <div
      aria-hidden="true"
      className="flex h-11 w-11 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
    >
      {icon.kind === "svg" && (
        <svg
          viewBox="0 0 24 24"
          className="h-6 w-6"
          fill={icon.color}
          role="img"
        >
          <path d={icon.path} />
        </svg>
      )}
      {icon.kind === "image" && (
        <img
          src={icon.src}
          alt=""
          className="h-7 w-7 rounded-md object-contain"
          loading="lazy"
        />
      )}
      {icon.kind === "generic" && <DocumentTextIcon className="h-6 w-6" />}
    </div>
  );
}

export function ConnectorsPage() {
  const [catalog, setCatalog] = useState<AgentConnectorCatalogEntry[]>([]);
  const [connectors, setConnectors] = useState<AgentConnector[]>([]);
  const [mcpServers, setMcpServers] = useState<AgentMCPServer[]>([]);
  const [tab, setTab] = useState<ConnectorTab>("app");
  const [query, setQuery] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [catalogResult, connectorResult, mcpResult] = await Promise.all([
        listConnectorCatalog(),
        listConnectors(),
        listMCPServers(),
      ]);
      setCatalog(catalogResult.connectors ?? []);
      setConnectors(connectorResult.connectors ?? []);
      setMcpServers(mcpResult.servers ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const visible = useMemo(
    () => filterConnectors(catalog, tab, query, connectors, mcpServers),
    [catalog, tab, query, connectors, mcpServers],
  );

  return (
    <SidebarLayout
      title="Conectores"
      sidebar={<AppSidebar current="connectors" />}
    >
      <div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-900">
        <div className="mx-auto w-full max-w-6xl px-6 pb-14 pt-10 lg:px-12">
          <h2 className="font-rounded text-3xl font-semibold tracking-tight text-neutral-950 dark:text-white">
            Conectores
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-neutral-500 dark:text-neutral-400">
            Aplicativos, APIs e servidores MCP que o agente pode usar. Um
            conector só fica ativo depois de registrado com a credencial do
            operador; segredos nunca aparecem aqui.
          </p>

          <label className="mt-6 flex items-center gap-3 rounded-2xl border border-neutral-300 bg-white px-4 py-3 dark:border-neutral-700 dark:bg-neutral-950">
            <MagnifyingGlassIcon className="h-5 w-5 text-neutral-400" />
            <input
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Pesquisar conectores"
              aria-label="Pesquisar conectores"
              className="w-full bg-transparent text-sm text-neutral-900 outline-none placeholder:text-neutral-400 dark:text-neutral-100"
            />
          </label>

          <div
            role="tablist"
            aria-label="Tipo de conector"
            className="mt-5 flex flex-wrap gap-2"
          >
            {TABS.map((item) => (
              <button
                key={item.id}
                type="button"
                role="tab"
                aria-selected={tab === item.id}
                onClick={() => setTab(item.id)}
                className={`rounded-xl px-4 py-2 text-sm transition-colors ${
                  tab === item.id
                    ? "bg-neutral-900 text-white dark:bg-neutral-100 dark:text-neutral-900"
                    : "text-neutral-500 hover:bg-neutral-200 hover:text-neutral-900 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-white"
                }`}
              >
                {item.label}
              </button>
            ))}
          </div>

          {error && (
            <div
              role="alert"
              className="mt-6 rounded-2xl border border-red-900/40 bg-red-950/10 p-5 text-sm text-red-700 dark:text-red-300"
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

          {loading && !error && (
            <p className="mt-8 text-sm text-neutral-500" role="status">
              Carregando conectores…
            </p>
          )}

          {tab === "connected" && !error && <ConnectorsManagePanel />}

          {tab !== "connected" &&
            !loading &&
            !error &&
            visible.length === 0 && (
              <p className="mt-8 text-sm text-neutral-500">
                Nenhum conector encontrado.
              </p>
            )}

          <ul className="mt-6 grid gap-3 md:grid-cols-2">
            {(tab === "connected" ? [] : visible).map((entry) => {
              const state = connectorState(entry, connectors, mcpServers);
              const open = expanded === entry.id;
              return (
                <li
                  key={entry.id}
                  className="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950"
                >
                  <div className="flex items-start gap-4">
                    <ConnectorLogo id={entry.id} />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <h3 className="text-sm font-semibold text-neutral-900 dark:text-white">
                          {entry.name}
                        </h3>
                        <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400">
                          {entry.category}
                        </span>
                      </div>
                      <p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">
                        {entry.description}
                      </p>
                    </div>
                    {state === "connected" ? (
                      <span className="flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
                        <CheckIcon className="h-4 w-4" />
                        Conectado
                      </span>
                    ) : (
                      <button
                        type="button"
                        aria-expanded={open}
                        onClick={() => setExpanded(open ? null : entry.id)}
                        className="shrink-0 rounded-lg border border-neutral-300 px-3 py-1.5 text-xs text-neutral-700 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
                      >
                        {state === "disabled" ? "Desativado" : "Conectar"}
                      </button>
                    )}
                  </div>
                  {open && (
                    <ConnectorQuickConnect
                      entry={entry}
                      connected={state === "connected"}
                      onChanged={() => void load()}
                    />
                  )}
                </li>
              );
            })}
          </ul>
        </div>
      </div>
    </SidebarLayout>
  );
}
