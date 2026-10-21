import { useCallback, useEffect, useMemo, useState } from "react";
import {
  CheckIcon,
  DocumentTextIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  SparklesIcon,
} from "@heroicons/react/24/outline";
import { useT } from "@/i18n";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { SettingsTabs } from "@/components/SettingsTabs";
import { connectorIcon } from "@/lib/connectorIcons";
import { ConnectorsManagePanel } from "@/components/ConnectorsManagePanel";
import { ConnectorQuickConnect } from "@/components/ConnectorQuickConnect";
import { WhatsAppGatewayPanel } from "@/components/WhatsAppGatewayPanel";
import { API_BASE } from "@/lib/config";
import {
  type AgentConnector,
  type AgentConnectorCatalogEntry,
  type AgentMCPServer,
} from "@/lib/agenticClient";
import {
  connectorState,
} from "@/lib/connectors";

type CategoryFilter =
  | "all"
  | "productivity"
  | "creativity"
  | "business"
  | "development"
  | "infra"
  | "finance"
  | "data"
  | "connected"
  | "whatsapp";

const CATEGORIES: Array<{ id: CategoryFilter; labelKey: string }> = [
  { id: "all", labelKey: "connectors.categories.all" },
  { id: "productivity", labelKey: "connectors.categories.productivity" },
  { id: "creativity", labelKey: "connectors.categories.creativity" },
  { id: "business", labelKey: "connectors.categories.business" },
  { id: "development", labelKey: "connectors.categories.development" },
  { id: "infra", labelKey: "connectors.categories.infra" },
  { id: "finance", labelKey: "connectors.categories.finance" },
  { id: "data", labelKey: "connectors.categories.data" },
  { id: "connected", labelKey: "connectors.categories.connected" },
  { id: "whatsapp", labelKey: "connectors.categories.whatsapp" },
];

// Cada aba mapeia para as categorias REAIS do catálogo (em português, como
// vêm do backend). O casamento é por igualdade exata — não por palavra em
// inglês — e cobre todas as categorias do catálogo, de forma que nenhuma aba
// fique falsamente vazia e nenhuma categoria do catálogo fique órfã.
const CATEGORY_GROUPS: Partial<Record<CategoryFilter, readonly string[]>> = {
  productivity: ["Produtividade", "Comunicação", "E-mail", "Notificações", "Suporte"],
  creativity: ["Design", "Mídia", "CMS"],
  business: ["CRM e Vendas", "Marketing", "Comércio"],
  development: [
    "Desenvolvimento",
    "Automação",
    "Backend",
    "Banco de dados",
    "IA e vetores",
    "Autenticação",
    "Busca",
    "Tempo real",
    "Mobile",
  ],
  infra: ["Deploy", "Observabilidade"],
  finance: ["Pagamentos", "Fiscal", "Dados financeiros"],
  data: ["Dados e métricas"],
};

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
  const { t } = useT();
  const [catalog, setCatalog] = useState<AgentConnectorCatalogEntry[]>([]);
  const [connectors, setConnectors] = useState<AgentConnector[]>([]);
  const [mcpServers, setMcpServers] = useState<AgentMCPServer[]>([]);
  const [category, setCategory] = useState<CategoryFilter>("all");
  const [query, setQuery] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Modal para criar conector customizado
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [customName, setCustomName] = useState("");
  const [customBaseUrl, setCustomBaseUrl] = useState("");
  const [customTokenEnv, setCustomTokenEnv] = useState("");
  const [customPath, setCustomPath] = useState("/");
  const [customAllowWrite, setCustomAllowWrite] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [catalogResult, connectorResult, mcpResult] = await Promise.all([
        fetch(`${API_BASE}/api/agent/v1/connector-catalog`).then((r) => (r.ok ? r.json() : { connectors: [] })),
        fetch(`${API_BASE}/api/agent/v1/connectors`).then((r) => (r.ok ? r.json() : { connectors: [] })),
        fetch(`${API_BASE}/api/agent/v1/mcp`).then((r) => (r.ok ? r.json() : { servers: [] })),
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

  const handleCreateCustom = async () => {
    if (!customName.trim() || !customBaseUrl.trim()) return;
    setCreating(true);
    setCreateError(null);
    try {
      // Padrão seguro: somente leitura (GET) num caminho específico. A escrita
      // (POST/PUT/PATCH/DELETE) só é liberada quando o operador marca
      // explicitamente a opção, evitando conectores full-access silenciosos.
      const pathPrefix = customPath.trim() || "/";
      const operations = customAllowWrite
        ? [
            {
              name: "read-write",
              methods: ["GET", "POST", "PUT", "PATCH", "DELETE"],
              path_prefixes: [pathPrefix],
            },
          ]
        : [
            {
              name: "read",
              methods: ["GET"],
              path_prefixes: [pathPrefix],
            },
          ];
      const res = await fetch(`${API_BASE}/api/agent/v1/connectors`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          id: customName.toLowerCase().replace(/\s+/g, "-"),
          provider: customName.trim(),
          base_url: customBaseUrl.trim(),
          token_env: customTokenEnv.trim() || undefined,
          operations,
        }),
      });
      if (!res.ok) throw new Error(t("connectors.create.failedBackend"));
      setShowCreateModal(false);
      setCustomName("");
      setCustomBaseUrl("");
      setCustomTokenEnv("");
      setCustomPath("/");
      setCustomAllowWrite(false);
      load();
    } catch (e) {
      setCreateError(e instanceof Error ? e.message : t("connectors.create.failed"));
    } finally {
      setCreating(false);
    }
  };

  const visible = useMemo(() => {
    return catalog.filter((entry) => {
      // Filtro de texto
      const matchesQuery =
        !query ||
        entry.name.toLowerCase().includes(query.toLowerCase()) ||
        entry.description.toLowerCase().includes(query.toLowerCase());

      if (!matchesQuery) return false;

      // Filtro de categoria
      if (category === "all") return true;
      if (category === "connected") {
        const state = connectorState(entry, connectors, mcpServers);
        return state === "connected";
      }

      // Correspondência exata contra as categorias reais do catálogo.
      const group = CATEGORY_GROUPS[category];
      return group ? group.includes(entry.category) : false;
    });
  }, [catalog, category, query, connectors, mcpServers]);

  return (
    <SidebarLayout
      title={t("connectors.page.title")}
      sidebar={<AppSidebar current="connectors" />}
    >
      <SettingsTabs current="connectors" />
      <div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-900">
        <div className="mx-auto w-full max-w-6xl space-y-8 px-6 pb-16 pt-8 lg:px-10">
          {/* Cabeçalho */}
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h1 className="page-title font-bold sm:text-3xl">
                {t("connectors.page.heading")}
              </h1>
              <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
                {t("connectors.page.description")}
              </p>
            </div>
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => setCategory("connected")}
                className="rounded-xl border border-neutral-200 bg-white px-3.5 py-2 text-xs font-semibold text-neutral-800 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
              >
                {t("connectors.page.manage", {
                  count: connectors.length + mcpServers.length,
                })}
              </button>
              <button
                type="button"
                onClick={() => setShowCreateModal(true)}
                className="inline-flex items-center gap-1.5 rounded-xl bg-neutral-900 px-3.5 py-2 text-xs font-semibold text-white hover:opacity-90 dark:bg-white dark:text-neutral-900"
              >
                <PlusIcon className="h-4 w-4" />
                {t("connectors.page.create")}
              </button>
            </div>
          </div>

          {/* Visualização de WhatsApp Gateway quando selecionado */}
          {category === "whatsapp" && (
            <div className="pt-2">
              <WhatsAppGatewayPanel />
            </div>
          )}

          {category !== "whatsapp" && (
            <>
          {/* Barra de Busca e Categorias */}
          <div className="space-y-4">
            <label className="flex items-center gap-3 rounded-2xl border border-neutral-300 bg-white px-4 py-2.5 dark:border-neutral-700 dark:bg-neutral-950">
              <MagnifyingGlassIcon className="h-5 w-5 text-neutral-400" />
              <input
                type="search"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder={t("connectors.search.placeholder")}
                aria-label={t("connectors.search.aria")}
                className="w-full bg-transparent text-sm text-neutral-900 outline-none placeholder:text-neutral-400 dark:text-neutral-100"
              />
            </label>

            {/* Abas de Categorias Fiel ao Manus */}
            <div className="flex space-x-2 overflow-x-auto pb-1">
              {CATEGORIES.map((cat) => {
                const active = category === cat.id;
                return (
                  <button
                    key={cat.id}
                    type="button"
                    onClick={() => setCategory(cat.id)}
                    className={`whitespace-nowrap rounded-xl px-3.5 py-1.5 text-xs font-medium transition-colors ${
                      active
                        ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900"
                        : "bg-white text-neutral-600 hover:bg-neutral-100 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-700"
                    }`}
                  >
                    {t(cat.labelKey)}
                  </button>
                );
              })}
            </div>
          </div>
            </>
          )}

          {/* Modal de Criar Conector Customizado */}
          {showCreateModal && (
            <div className="rounded-2xl border border-neutral-200 bg-white p-6 shadow-md dark:border-neutral-800 dark:bg-neutral-950">
              <h3 className="text-base font-bold text-neutral-900 dark:text-white">
                {t("connectors.create.title")}
              </h3>
              <p className="mt-1 text-xs text-neutral-500">
                {t("connectors.create.description")}
              </p>
              <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                  <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                    {t("connectors.create.nameLabel")}
                  </label>
                  <input
                    type="text"
                    value={customName}
                    onChange={(e) => setCustomName(e.target.value)}
                    placeholder={t("connectors.create.namePlaceholder")}
                    className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                    {t("connectors.create.baseUrlLabel")}
                  </label>
                  <input
                    type="text"
                    value={customBaseUrl}
                    onChange={(e) => setCustomBaseUrl(e.target.value)}
                    placeholder="https://api.empresa.com/v1"
                    className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                  />
                </div>
                <div className="sm:col-span-2">
                  <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                    {t("connectors.create.tokenEnvLabel")}
                  </label>
                  <input
                    type="text"
                    value={customTokenEnv}
                    onChange={(e) => setCustomTokenEnv(e.target.value)}
                    placeholder={t("connectors.create.tokenEnvPlaceholder")}
                    className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                  />
                </div>
                <div className="sm:col-span-2">
                  <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                    {t("connectors.create.pathLabel")}
                  </label>
                  <input
                    type="text"
                    value={customPath}
                    onChange={(e) => setCustomPath(e.target.value)}
                    placeholder={t("connectors.create.pathPlaceholder")}
                    className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                  />
                  <p className="mt-1 text-[11px] text-neutral-500">
                    {t("connectors.create.pathHint")}
                  </p>
                </div>
                <div className="sm:col-span-2">
                  <label className="flex items-start gap-2 text-xs font-medium text-neutral-700 dark:text-neutral-300">
                    <input
                      type="checkbox"
                      checked={customAllowWrite}
                      onChange={(e) => setCustomAllowWrite(e.target.checked)}
                      className="mt-0.5 h-4 w-4 rounded border-neutral-300 dark:border-neutral-600"
                    />
                    <span>
                      {t("connectors.create.allowWriteLabel")}
                      <span className="mt-0.5 block font-normal text-[11px] text-neutral-500">
                        {t("connectors.create.allowWriteHint")}
                      </span>
                    </span>
                  </label>
                </div>
              </div>
              {createError && (
                <div
                  role="alert"
                  aria-live="assertive"
                  className="mt-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-xs text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300"
                >
                  {createError}
                </div>
              )}
              <div className="mt-5 flex justify-end gap-3">
                <button
                  type="button"
                  onClick={() => {
                    setShowCreateModal(false);
                    setCreateError(null);
                  }}
                  className="rounded-xl border border-neutral-200 px-4 py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-300"
                >
                  {t("connectors.create.cancel")}
                </button>
                <button
                  type="button"
                  disabled={creating || !customName.trim() || !customBaseUrl.trim()}
                  onClick={handleCreateCustom}
                  className="rounded-xl bg-neutral-900 px-4 py-2 text-xs font-semibold text-white hover:opacity-90 disabled:opacity-50 dark:bg-white dark:text-neutral-900"
                >
                  {creating
                    ? t("connectors.create.saving")
                    : t("connectors.create.submit")}
                </button>
              </div>
            </div>
          )}

          {/* Painel de Gerenciamento ou Grade de Plugins */}
          {category === "connected" ? (
            <ConnectorsManagePanel />
          ) : (
            <div>
              {loading && (
                <p className="py-12 text-center text-sm text-neutral-500">
                  {t("connectors.list.loading")}
                </p>
              )}

              {error && (
                <div className="rounded-2xl border border-red-900/40 bg-red-950/10 p-5 text-sm text-red-700 dark:text-red-300">
                  <p>{error}</p>
                  <button
                    type="button"
                    onClick={() => void load()}
                    className="mt-2 text-xs underline"
                  >
                    {t("common.retry")}
                  </button>
                </div>
              )}

              {!loading && !error && visible.length === 0 && (
                <div className="rounded-2xl border border-dashed border-neutral-300 p-12 text-center dark:border-neutral-700">
                  <SparklesIcon className="mx-auto h-8 w-8 text-neutral-400" />
                  <p className="mt-3 text-sm text-neutral-500">
                    {t("connectors.list.empty")}
                  </p>
                </div>
              )}

              <ul className="grid gap-4 md:grid-cols-2">
                {visible.map((entry) => {
                  const state = connectorState(entry, connectors, mcpServers);
                  const open = expanded === entry.id;
                  // Conectores exclusivamente OAuth não conectam com chave
                  // colada: exigem um app OAuth registrado pela DZ23. Em vez de
                  // prometer "Conectar" e cair num aviso, deixamos isso claro.
                  const oauthOnly = entry.auth === "oauth" && !entry.quick_connect;
                  return (
                    <li
                      key={entry.id}
                      className="rounded-2xl border border-neutral-200 bg-white p-5 shadow-sm transition-all hover:shadow-md dark:border-neutral-800 dark:bg-neutral-950"
                    >
                      <div className="flex items-start gap-4">
                        <ConnectorLogo id={entry.id} />
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-2">
                            <h3 className="text-sm font-bold text-neutral-900 dark:text-white">
                              {entry.name}
                            </h3>
                            <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] font-medium text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400">
                              {entry.category}
                            </span>
                          </div>
                          <p className="mt-1 text-xs leading-relaxed text-neutral-500 dark:text-neutral-400">
                            {entry.description}
                          </p>
                        </div>
                        {state === "connected" ? (
                          <span className="flex items-center gap-1 rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-semibold text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300">
                            <CheckIcon className="h-3.5 w-3.5" />
                            {t("connectors.card.active")}
                          </span>
                        ) : oauthOnly ? (
                          <button
                            type="button"
                            aria-expanded={open}
                            onClick={() => setExpanded(open ? null : entry.id)}
                            title={t("connectors.card.oauthTitle")}
                            className="inline-flex shrink-0 items-center gap-1 rounded-xl border border-amber-300 bg-amber-50 px-3 py-1.5 text-xs font-semibold text-amber-700 hover:bg-amber-100 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-300"
                          >
                            {t("connectors.card.oauthRequires")}
                          </button>
                        ) : (
                          <button
                            type="button"
                            aria-expanded={open}
                            onClick={() => setExpanded(open ? null : entry.id)}
                            className="inline-flex shrink-0 items-center gap-1 rounded-xl border border-neutral-300 bg-white px-3 py-1.5 text-xs font-semibold text-neutral-800 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
                          >
                            <PlusIcon className="h-3.5 w-3.5" />
                            {state === "disabled"
                              ? t("connectors.card.disabled")
                              : t("connectors.card.connect")}
                          </button>
                        )}
                      </div>
                      {open && (
                        <div className="mt-4 border-t border-neutral-100 pt-4 dark:border-neutral-800">
                          <ConnectorQuickConnect
                            entry={entry}
                            connected={state === "connected"}
                            onChanged={() => void load()}
                          />
                        </div>
                      )}
                    </li>
                  );
                })}
              </ul>
            </div>
          )}
        </div>
      </div>
    </SidebarLayout>
  );
}
