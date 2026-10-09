import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowTopRightOnSquareIcon,
  CheckCircleIcon,
  CircleStackIcon,
  ExclamationTriangleIcon,
  KeyIcon,
  LockClosedIcon,
  WrenchScrewdriverIcon,
} from "@heroicons/react/24/outline";
import { agentFetch, listCLIStatus, listConnectors, listMCPServers, listSkills } from "@/lib/agenticClient";
import { getModels } from "@/api";

type SafeConfig = {
  runtime?: string;
  store?: string;
  queue?: string;
  auth_required?: boolean;
  approval_gated_tools?: boolean;
  workspace_isolation?: boolean;
  planner_model_configured?: boolean;
  embedding_configured?: boolean;
  connectors_configured?: boolean;
  mcp_configured?: boolean;
  media_configured?: boolean;
  deployments_configured?: boolean;
  otlp_configured?: boolean;
  push_configured?: boolean;
};

function StatusPill({ ok, children }: { ok: boolean; children: React.ReactNode }) {
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-[10px] font-medium ${ok ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300" : "bg-neutral-100 text-neutral-500 dark:bg-neutral-800 dark:text-neutral-400"}`}>
      {ok ? <CheckCircleIcon className="h-3.5 w-3.5" /> : <ExclamationTriangleIcon className="h-3.5 w-3.5" />}
      {children}
    </span>
  );
}

function ResourceList({ title, count, children }: { title: string; count: number; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-neutral-200/80 p-4 dark:border-neutral-700">
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-xs font-semibold text-neutral-800 dark:text-neutral-200">{title}</h3>
        <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] text-neutral-500 dark:bg-neutral-900 dark:text-neutral-400">{count}</span>
      </div>
      <div className="mt-3 space-y-2">{children}</div>
    </div>
  );
}

export function AgenticControlCenter() {
  const { data: rawModels, isLoading: modelsLoading } = useQuery({ queryKey: ["agent-models"], queryFn: () => getModels(""), retry: false });
  const models = useMemo(
    () => (Array.isArray(rawModels) ? rawModels : []),
    [rawModels],
  );
  const { data: safeConfig, isLoading: configLoading } = useQuery<SafeConfig>({
    queryKey: ["agent-safe-config"],
    queryFn: () => agentFetch<SafeConfig>("/api/agent/v1/config/safe"),
    retry: false,
  });
  const { data: connectorData, isLoading: connectorsLoading } = useQuery({ queryKey: ["agent-connectors"], queryFn: listConnectors, retry: false });
  const { data: mcpData, isLoading: mcpLoading } = useQuery({ queryKey: ["agent-mcp"], queryFn: listMCPServers, retry: false });
  const { data: skillData, isLoading: skillsLoading } = useQuery({ queryKey: ["agent-skills"], queryFn: listSkills, retry: false });
  const { data: cliData, isLoading: cliLoading } = useQuery({ queryKey: ["agent-cli-catalog"], queryFn: listCLIStatus, retry: false });
  const connectors = Array.isArray(connectorData?.connectors) ? connectorData.connectors : [];
  const mcpServers = Array.isArray(mcpData?.servers) ? mcpData.servers : [];
  const skills = Array.isArray(skillData?.skills) ? skillData.skills : [];

  const providers = useMemo(() => {
    const values = new Set<string>();
    for (const model of models) values.add(model.provider || (model.kind === "router" ? "DZ23 Router" : "Ollama"));
    return [...values].sort();
  }, [models]);

  const installedCLIs = Array.isArray(cliData?.tools) ? cliData.tools.filter((item) => item.installed) : [];
  const runtimeReady = Boolean(safeConfig?.runtime);
  const statusRows = [
    { label: "Planner", enabled: safeConfig?.planner_model_configured ?? false },
    { label: "Connectors", enabled: safeConfig?.connectors_configured ?? false },
    { label: "MCP / skills", enabled: safeConfig?.mcp_configured ?? false },
    { label: "Media provider", enabled: safeConfig?.media_configured ?? false },
    { label: "Deploy providers", enabled: safeConfig?.deployments_configured ?? false },
    { label: "OTLP traces", enabled: safeConfig?.otlp_configured ?? false },
  ];

  return (
    <section id="agentic" className="overflow-hidden rounded-xl border border-neutral-200/80 bg-white dark:border-neutral-800 dark:bg-neutral-800">
      <div className="border-b border-neutral-200/80 bg-gradient-to-br from-neutral-50 to-violet-50/40 p-5 dark:border-neutral-700 dark:from-neutral-900 dark:to-violet-950/20">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <div className="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><WrenchScrewdriverIcon className="h-5 w-5 text-violet-500" />Agentic Control Center</div>
            <p className="mt-1 max-w-2xl text-xs leading-5 text-neutral-500 dark:text-neutral-400">Catálogo e estado reais do runtime: providers, CLIs, connectors, MCP, skills, approvals e integrações. Valores de secrets nunca são retornados.</p>
          </div>
          <div className="flex items-center gap-2"><StatusPill ok={runtimeReady}>{configLoading ? "Consultando" : runtimeReady ? "Runtime ativo" : "Modo local"}</StatusPill><a href="/agentic" className="inline-flex items-center gap-1 text-xs font-medium text-violet-600 hover:text-violet-500 dark:text-violet-300">Abrir console <ArrowTopRightOnSquareIcon className="h-3.5 w-3.5" /></a></div>
        </div>
      </div>

      <div className="grid gap-4 p-5 lg:grid-cols-[1.1fr_0.9fr]">
        <div>
          <div className="mb-3 flex items-center gap-2 text-xs font-semibold text-neutral-800 dark:text-neutral-200"><CircleStackIcon className="h-4 w-4 text-neutral-400" />Providers visíveis no catálogo</div>
          <div className="flex flex-wrap gap-2">{modelsLoading ? <span className="text-xs text-neutral-400">Carregando catálogo…</span> : providers.length > 0 ? providers.map((provider) => <span key={provider} className="rounded-lg border border-neutral-200 px-2.5 py-1.5 text-xs text-neutral-600 dark:border-neutral-700 dark:text-neutral-300">{provider}</span>) : <span className="text-xs text-neutral-400">Nenhum provider remoto configurado; modelos locais continuam disponíveis.</span>}</div>
          <div className="mt-4 flex flex-wrap gap-2"><StatusPill ok={Boolean(safeConfig?.approval_gated_tools)}>Aprovações no servidor</StatusPill><StatusPill ok={Boolean(safeConfig?.workspace_isolation)}>Workspace isolado</StatusPill><StatusPill ok={safeConfig?.auth_required ?? false}>{safeConfig?.auth_required ? "Auth obrigatória" : "Auth local"}</StatusPill></div>

          <div className="mt-5 grid gap-3 sm:grid-cols-2">
            <ResourceList title="Conectores na allowlist" count={connectors.length}>{connectorsLoading ? <span className="text-xs text-neutral-400">Carregando…</span> : connectors.length ? connectors.map((connector) => <div key={connector.id} className="rounded-lg bg-neutral-50 p-2.5 text-xs dark:bg-neutral-900"><div className="font-medium text-neutral-800 dark:text-neutral-200">{connector.provider} <span className="text-neutral-400">· {connector.id}</span></div><div className="mt-1 truncate text-[10px] text-neutral-500">{connector.base_url}</div><div className="mt-1 text-[10px] text-neutral-400">{connector.operations?.length ?? 0} operações · token por ambiente{connector.oauth_provider ? ` · OAuth ${connector.oauth_provider}` : ""}</div></div>) : <span className="text-xs text-neutral-400">Nenhum connector carregado.</span>}</ResourceList>
            <ResourceList title="MCP stdio" count={mcpServers.length}>{mcpLoading ? <span className="text-xs text-neutral-400">Carregando…</span> : mcpServers.length ? mcpServers.map((server) => <div key={server.id} className="rounded-lg bg-neutral-50 p-2.5 text-xs dark:bg-neutral-900"><div className="font-medium text-neutral-800 dark:text-neutral-200">{server.id}</div><div className="mt-1 truncate font-mono text-[10px] text-neutral-500">{server.command} {(server.args ?? []).join(" ")}</div><div className="mt-1 text-[10px] text-neutral-400">{server.allowed_methods?.length ?? 0} métodos allowlisted · {server.environment_vars?.length ?? 0} nomes de ambiente</div></div>) : <span className="text-xs text-neutral-400">Nenhum servidor MCP carregado.</span>}</ResourceList>
            <ResourceList title="Skills" count={skills.length}>{skillsLoading ? <span className="text-xs text-neutral-400">Carregando…</span> : skills.length ? skills.map((skill) => <div key={skill.id} className="rounded-lg bg-neutral-50 p-2.5 text-xs dark:bg-neutral-900"><div className="flex items-center justify-between gap-2 font-medium text-neutral-800 dark:text-neutral-200"><span>{skill.id}</span><StatusPill ok={skill.trusted}>{skill.trusted ? "confiável" : "revisar"}</StatusPill></div><div className="mt-1 line-clamp-2 text-[10px] text-neutral-500">{skill.description}</div><div className="mt-1 text-[10px] text-neutral-400">v{skill.version} · {skill.tools?.length ?? 0} tools</div></div>) : <span className="text-xs text-neutral-400">Nenhuma skill carregada.</span>}</ResourceList>
            <ResourceList title="CLIs detectadas" count={installedCLIs.length}>{cliLoading ? <span className="text-xs text-neutral-400">Detectando…</span> : installedCLIs.length ? installedCLIs.map((cli) => <div key={cli.id} className="flex items-center justify-between rounded-lg bg-neutral-50 p-2.5 text-xs dark:bg-neutral-900"><span className="font-medium text-neutral-800 dark:text-neutral-200">{cli.name}</span><StatusPill ok>{cli.executable ?? "instalado"}</StatusPill></div>) : <span className="text-xs text-neutral-400">Nenhum CLI compatível detectado no ambiente.</span>}</ResourceList>
          </div>
        </div>

        <div className="rounded-xl border border-neutral-200/80 p-4 dark:border-neutral-700">
          <div className="flex items-center gap-2 text-xs font-semibold text-neutral-800 dark:text-neutral-200"><KeyIcon className="h-4 w-4 text-neutral-400" />Capacidades configuradas</div>
          <div className="mt-3 grid grid-cols-2 gap-2">{statusRows.map((row) => <div key={row.label} className="flex items-center gap-2 text-[11px] text-neutral-500 dark:text-neutral-400"><span className={`h-1.5 w-1.5 rounded-full ${row.enabled ? "bg-emerald-500" : "bg-neutral-300 dark:bg-neutral-600"}`} />{row.label}</div>)}</div>
          <div className="mt-4 flex items-start gap-2 rounded-lg bg-neutral-50 p-3 text-[10px] leading-4 text-neutral-500 dark:bg-neutral-900 dark:text-neutral-400"><LockClosedIcon className="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-500" />A edição de credenciais, instalação de CLIs e registro de MCP/connectors exigem configuração segura no servidor. Esta tela mostra o estado real e não simula conexão.</div>
        </div>
      </div>
    </section>
  );
}
