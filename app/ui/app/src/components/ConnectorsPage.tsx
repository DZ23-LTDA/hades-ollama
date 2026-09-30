import { useCallback, useEffect, useMemo, useState } from "react";
import {
  CheckIcon,
  DocumentTextIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  SparklesIcon,
  EnvelopeIcon,
  ChatBubbleLeftRightIcon,
  BookOpenIcon,
  CodeBracketIcon,
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
  registerConnector,
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
  | "finance"
  | "travel"
  | "health"
  | "connected";

const CATEGORIES: Array<{ id: CategoryFilter; label: string }> = [
  { id: "all", label: "Todos" },
  { id: "productivity", label: "Produtividade" },
  { id: "creativity", label: "Criatividade" },
  { id: "business", label: "Negócios" },
  { id: "development", label: "Desenvolvimento" },
  { id: "finance", label: "Finanças" },
  { id: "travel", label: "Viagem" },
  { id: "health", label: "Saúde" },
  { id: "connected", label: "Conectados" },
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
  const [creating, setCreating] = useState(false);

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

  const handleCreateCustom = async () => {
    if (!customName.trim() || !customBaseUrl.trim()) return;
    setCreating(true);
    try {
      await registerConnector({
        id: customName.toLowerCase().replace(/\s+/g, "-"),
        provider: customName.trim(),
        base_url: customBaseUrl.trim(),
        token_env: customTokenEnv.trim() || undefined,
        operations: [
          {
            name: "default",
            methods: ["GET", "POST", "PUT", "DELETE"],
            path_prefixes: ["/"],
          },
        ],
      });
      setShowCreateModal(false);
      setCustomName("");
      setCustomBaseUrl("");
      setCustomTokenEnv("");
      load();
    } catch (e) {
      alert(e instanceof Error ? e.message : "Falha ao registrar conector");
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

      // Mapeamento aproximado de categoria
      const catLower = entry.category.toLowerCase();
      if (category === "productivity" && (catLower.includes("prod") || catLower.includes("doc") || catLower.includes("mail"))) return true;
      if (category === "creativity" && (catLower.includes("media") || catLower.includes("video") || catLower.includes("art") || catLower.includes("audio"))) return true;
      if (category === "business" && (catLower.includes("crm") || catLower.includes("sales") || catLower.includes("negócios"))) return true;
      if (category === "development" && (catLower.includes("dev") || catLower.includes("code") || catLower.includes("git") || catLower.includes("api"))) return true;
      if (category === "finance" && (catLower.includes("fin") || catLower.includes("pay") || catLower.includes("bank"))) return true;
      if (category === "travel" && (catLower.includes("trip") || catLower.includes("viagem") || catLower.includes("flight"))) return true;
      if (category === "health" && (catLower.includes("health") || catLower.includes("saúde"))) return true;

      return catLower.includes(category);
    });
  }, [catalog, category, query, connectors, mcpServers]);

  return (
    <SidebarLayout
      title="Plugins & Conectores"
      sidebar={<AppSidebar current="connectors" />}
    >
      <div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-900">
        <div className="mx-auto w-full max-w-6xl space-y-8 px-6 pb-16 pt-8 lg:px-10">
          {/* Cabeçalho */}
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h1 className="font-rounded text-2xl font-bold tracking-tight text-neutral-950 dark:text-white sm:text-3xl">
                Plugins
              </h1>
              <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
                Conecte serviços externos, contas OAuth e servidores MCP para expandir as capacidades do agente.
              </p>
            </div>
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => setCategory("connected")}
                className="rounded-xl border border-neutral-200 bg-white px-3.5 py-2 text-xs font-semibold text-neutral-800 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
              >
                Gerenciar Conectores ({connectors.length + mcpServers.length})
              </button>
              <button
                type="button"
                onClick={() => setShowCreateModal(true)}
                className="inline-flex items-center gap-1.5 rounded-xl bg-neutral-900 px-3.5 py-2 text-xs font-semibold text-white hover:opacity-90 dark:bg-white dark:text-neutral-900"
              >
                <PlusIcon className="h-4 w-4" />
                Criar Conector
              </button>
            </div>
          </div>

          {/* Banners de Destaque Oficiais do Manus */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-blue-50 text-blue-600 dark:bg-blue-950/50">
                  <EnvelopeIcon className="h-5 w-5" />
                </div>
                <h4 className="text-xs font-bold text-neutral-900 dark:text-white">Workspace & Docs</h4>
              </div>
              <p className="mt-3 text-[11px] leading-relaxed text-neutral-500">
                Reúna e-mail, documentos e calendário em um só lugar.
              </p>
            </div>
            <div className="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-purple-50 text-purple-600 dark:bg-purple-950/50">
                  <ChatBubbleLeftRightIcon className="h-5 w-5" />
                </div>
                <h4 className="text-xs font-bold text-neutral-900 dark:text-white">Comunicação</h4>
              </div>
              <p className="mt-3 text-[11px] leading-relaxed text-neutral-500">
                Acompanhe discussões e atualizações da equipe em tempo real.
              </p>
            </div>
            <div className="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-amber-50 text-amber-600 dark:bg-amber-950/50">
                  <BookOpenIcon className="h-5 w-5" />
                </div>
                <h4 className="text-xs font-bold text-neutral-900 dark:text-white">Conhecimento</h4>
              </div>
              <p className="mt-3 text-[11px] leading-relaxed text-neutral-500">
                Encontre respostas no seu workspace do Notion e Obsidian.
              </p>
            </div>
            <div className="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-50 text-emerald-600 dark:bg-emerald-950/50">
                  <CodeBracketIcon className="h-5 w-5" />
                </div>
                <h4 className="text-xs font-bold text-neutral-900 dark:text-white">Código & Git</h4>
              </div>
              <p className="mt-3 text-[11px] leading-relaxed text-neutral-500">
                Automatize tarefas, commits e auditoria de repositórios.
              </p>
            </div>
          </div>

          {/* Barra de Busca e Categorias */}
          <div className="space-y-4">
            <label className="flex items-center gap-3 rounded-2xl border border-neutral-300 bg-white px-4 py-2.5 dark:border-neutral-700 dark:bg-neutral-950">
              <MagnifyingGlassIcon className="h-5 w-5 text-neutral-400" />
              <input
                type="search"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Buscar conectores e plugins..."
                aria-label="Buscar conectores"
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
                    {cat.label}
                  </button>
                );
              })}
            </div>
          </div>

          {/* Modal de Criar Conector Customizado */}
          {showCreateModal && (
            <div className="rounded-2xl border border-neutral-200 bg-white p-6 shadow-md dark:border-neutral-800 dark:bg-neutral-950">
              <h3 className="text-base font-bold text-neutral-900 dark:text-white">
                Cadastrar Conector / API Personalizada
              </h3>
              <p className="mt-1 text-xs text-neutral-500">
                Adicione endpoints REST ou servidores MCP locais/remotos para o agente invocar com segurança.
              </p>
              <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                  <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                    Nome do Serviço / Provedor
                  </label>
                  <input
                    type="text"
                    value={customName}
                    onChange={(e) => setCustomName(e.target.value)}
                    placeholder="Ex: Minha API Interna"
                    className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                    URL Base da API
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
                    Variável de Ambiente com a Chave (opcional)
                  </label>
                  <input
                    type="text"
                    value={customTokenEnv}
                    onChange={(e) => setCustomTokenEnv(e.target.value)}
                    placeholder="Ex: MINHA_API_KEY (o segredo nunca é exposto)"
                    className="mt-1 w-full rounded-xl border border-neutral-300 bg-white px-3 py-2 text-sm text-neutral-900 focus:outline-none dark:border-neutral-700 dark:bg-neutral-800 dark:text-white"
                  />
                </div>
              </div>
              <div className="mt-5 flex justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="rounded-xl border border-neutral-200 px-4 py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-300"
                >
                  Cancelar
                </button>
                <button
                  type="button"
                  disabled={creating || !customName.trim() || !customBaseUrl.trim()}
                  onClick={handleCreateCustom}
                  className="rounded-xl bg-neutral-900 px-4 py-2 text-xs font-semibold text-white hover:opacity-90 disabled:opacity-50 dark:bg-white dark:text-neutral-900"
                >
                  {creating ? "Salvando..." : "Registrar conector"}
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
                  Carregando catálogo de plugins...
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
                    Tentar novamente
                  </button>
                </div>
              )}

              {!loading && !error && visible.length === 0 && (
                <div className="rounded-2xl border border-dashed border-neutral-300 p-12 text-center dark:border-neutral-700">
                  <SparklesIcon className="mx-auto h-8 w-8 text-neutral-400" />
                  <p className="mt-3 text-sm text-neutral-500">
                    Nenhum conector encontrado nesta categoria.
                  </p>
                </div>
              )}

              <ul className="grid gap-4 md:grid-cols-2">
                {visible.map((entry) => {
                  const state = connectorState(entry, connectors, mcpServers);
                  const open = expanded === entry.id;
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
                            Ativo
                          </span>
                        ) : (
                          <button
                            type="button"
                            aria-expanded={open}
                            onClick={() => setExpanded(open ? null : entry.id)}
                            className="inline-flex shrink-0 items-center gap-1 rounded-xl border border-neutral-300 bg-white px-3 py-1.5 text-xs font-semibold text-neutral-800 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
                          >
                            <PlusIcon className="h-3.5 w-3.5" />
                            {state === "disabled" ? "Desativado" : "Conectar"}
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
