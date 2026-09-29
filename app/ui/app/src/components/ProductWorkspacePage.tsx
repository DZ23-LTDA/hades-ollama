import { Link } from "@tanstack/react-router";
import { useCallback, useEffect, useMemo, useState } from "react";
import { API_BASE } from "@/lib/config";
import {
  createProject,
  createSchedule,
  deleteProject,
	deleteSchedule,
	listCLIStatus,
	listConnectorCatalog,
	listConnectors,
  listMCPServers,
  listMissions,
  listProjects,
	listSchedules,
	listSkills,
	registerConnector,
	registerMCP,
	registerRemoteMCP,
	registerSkill,
	removeConnector,
	removeMCP,
	removeSkill,
	setConnectorEnabled,
	setMCPEnabled,
	setSkillEnabled,
	updateProject,
} from "@/lib/agenticClient";
import type { AgentConnector, AgentConnectorCatalogEntry, AgentMCPServer, AgentMission, AgentProject, AgentSchedule, AgentSkill } from "@/lib/agenticClient";
import {
  ArrowRightIcon,
  CheckCircleIcon,
  ClockIcon,
  FolderOpenIcon,
  LockClosedIcon,
  PlusIcon,
  SparklesIcon,
  TrashIcon,
} from "@heroicons/react/24/outline";
import { AppSidebar, type AppSection } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";

type ProductPageKind = "library" | "projects" | "scheduled" | "skills" | "plugins" | "tasks";

const pageCopy: Record<ProductPageKind, { title: string; eyebrow: string; description: string; action: string }> = {
  library: { title: "Biblioteca", eyebrow: "Artifacts e arquivos", description: "Encontre documentos, sites, jogos, dashboards e outros artifacts produzidos pelas suas missões.", action: "Abrir Agentic Console" },
  projects: { title: "Projetos", eyebrow: "Contexto persistente", description: "Organize memória, fontes, tarefas, membros e builders em workspaces isolados.", action: "Novo projeto" },
  scheduled: { title: "Agendado", eyebrow: "Automação controlada", description: "Acompanhe tarefas recorrentes, webhooks, retries e execuções aguardando aprovação.", action: "Agendar tarefa" },
  skills: { title: "Habilidades", eyebrow: "Skills e MCP", description: "Veja manifestos, versões, escopos e confiança das habilidades carregadas pelo runtime.", action: "Abrir configuração" },
  plugins: { title: "Registro avançado", eyebrow: "Conectores, MCP e skills", description: "Inspecione GitHub, Google Workspace, Claude, Codex, OmniRoute, MCP e outros serviços sem expor segredos.", action: "Abrir configuração" },
  tasks: { title: "Tarefas", eyebrow: "Inbox de missões", description: "Veja missões em execução, aguardando approval, concluídas ou em recuperação.", action: "Nova tarefa" },
};

function StatCard({ label, value, tone = "neutral" }: { label: string; value: string; tone?: "neutral" | "green" | "violet" }) {
  const toneClass = tone === "green" ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300" : tone === "violet" ? "bg-violet-50 text-violet-700 dark:bg-violet-950/30 dark:text-violet-300" : "bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300";
  return <div className="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><div className="text-[11px] uppercase tracking-[0.12em] text-neutral-400">{label}</div><div className={`mt-3 inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${toneClass}`}>{value}</div></div>;
}

function EmptyState({ message, action, onAction }: { message: string; action?: string; onAction?: () => void }) {
  return <div className="rounded-xl border border-dashed border-neutral-300 px-6 py-10 text-center dark:border-neutral-700"><FolderOpenIcon className="mx-auto h-8 w-8 text-neutral-300 dark:text-neutral-600" /><p className="mx-auto mt-3 max-w-md text-sm text-neutral-500 dark:text-neutral-400">{message}</p>{action && onAction && <button onClick={onAction} className="mt-5 inline-flex items-center gap-2 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800">{action}<ArrowRightIcon className="h-3.5 w-3.5" /></button>}</div>;
}

function ResourceRow({ children, onDelete, deleteLabel = "Excluir recurso", deleteDisabled = false }: { children: React.ReactNode; onDelete?: () => void; deleteLabel?: string; deleteDisabled?: boolean }) {
	  return <div className="flex items-center justify-between gap-3 rounded-xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><div className="min-w-0 flex-1">{children}</div>{onDelete && <button type="button" onClick={onDelete} disabled={deleteDisabled} className="rounded-lg p-2 text-neutral-400 hover:bg-red-50 hover:text-red-600 disabled:cursor-wait disabled:opacity-40 dark:hover:bg-red-950/30" aria-label={deleteLabel} title={deleteLabel}><TrashIcon className="h-4 w-4" /></button>}</div>;
}

export function ProductWorkspacePage({ kind }: { kind: ProductPageKind }) {
  const copy = pageCopy[kind];
  const current = kind as AppSection;
  const [notice, setNotice] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [pendingActions, setPendingActions] = useState<Record<string, boolean>>({});
  const [projects, setProjects] = useState<AgentProject[]>([]);
  const [missions, setMissions] = useState<AgentMission[]>([]);
  const [schedules, setSchedules] = useState<AgentSchedule[]>([]);
  const [skills, setSkills] = useState<AgentSkill[]>([]);
	const [connectors, setConnectors] = useState<AgentConnector[]>([]);
	const [connectorCatalog, setConnectorCatalog] = useState<AgentConnectorCatalogEntry[]>([]);
	const [connectorSearch, setConnectorSearch] = useState("");
	const [connectorCategory, setConnectorCategory] = useState("Todos");
	const [newConnectorID, setNewConnectorID] = useState("");
	const [newConnectorProvider, setNewConnectorProvider] = useState("");
	const [newConnectorBaseURL, setNewConnectorBaseURL] = useState("");
	const [newConnectorTokenEnv, setNewConnectorTokenEnv] = useState("");
	const [newConnectorOAuthProvider, setNewConnectorOAuthProvider] = useState("");
	const [newConnectorMethods, setNewConnectorMethods] = useState("GET");
	const [newConnectorPathPrefix, setNewConnectorPathPrefix] = useState("/");
	const [newMCPID, setNewMCPID] = useState("");
	const [newMCPCommand, setNewMCPCommand] = useState("");
	const [newMCPMethods, setNewMCPMethods] = useState("tools/list");
	const [newRemoteMCPID, setNewRemoteMCPID] = useState("");
	const [newRemoteMCPURL, setNewRemoteMCPURL] = useState("");
	const [newRemoteMCPTokenEnv, setNewRemoteMCPTokenEnv] = useState("");
	const [newRemoteMCPMethods, setNewRemoteMCPMethods] = useState("tools/list");
	const [newSkillID, setNewSkillID] = useState("");
	const [newSkillVersion, setNewSkillVersion] = useState("1.0.0");
	const [newSkillDescription, setNewSkillDescription] = useState("");
	const [newSkillScopes, setNewSkillScopes] = useState("");
	const [newSkillTools, setNewSkillTools] = useState("");
  const [mcpServers, setMcpServers] = useState<AgentMCPServer[]>([]);
  const [cliCount, setCliCount] = useState(0);
  const [projectName, setProjectName] = useState("");
  const [projectRoot, setProjectRoot] = useState("");
  const [scheduleObjective, setScheduleObjective] = useState("");
  const [scheduleInterval, setScheduleInterval] = useState("3600");
  const [scheduleProject, setScheduleProject] = useState("");
  const [editingProject, setEditingProject] = useState<string | null>(null);
  const [editingProjectName, setEditingProjectName] = useState("");

  const refresh = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      if (kind === "projects" || kind === "scheduled") setProjects((await listProjects()).projects ?? []);
      if (kind === "tasks" || kind === "library") setMissions((await listMissions()).missions ?? []);
      if (kind === "scheduled") setSchedules((await listSchedules()).schedules ?? []);
		if (kind === "skills") setSkills((await listSkills()).skills ?? []);
		if (kind === "plugins") {
			const [connectorResult, catalogResult, mcpResult, cliResult] = await Promise.all([listConnectors(), listConnectorCatalog(), listMCPServers(), listCLIStatus()]);
			setConnectors(connectorResult.connectors ?? []);
			setConnectorCatalog(catalogResult.connectors ?? []);
			setMcpServers(mcpResult.servers ?? []);
        setCliCount(cliResult.tools.filter((item) => item.installed).length);
      }
    } catch (cause) {
      const message = cause instanceof Error ? cause.message : "Não foi possível carregar os dados persistidos.";
      setLoadError(message);
      setNotice(message);
    } finally {
      setLoading(false);
    }
  }, [kind]);

  useEffect(() => { void refresh(); }, [refresh]);

  const setPending = (key: string, pending: boolean) => {
    setPendingActions((current) => {
      const next = { ...current };
      if (pending) next[key] = true;
      else delete next[key];
      return next;
    });
  };

  const runPending = async (key: string, action: () => Promise<void>) => {
    if (pendingActions[key]) return;
    setPending(key, true);
    try {
      await action();
    } finally {
      setPending(key, false);
    }
  };

	const artifacts = useMemo(() => missions.flatMap((mission) => (mission.artifacts ?? []).map((artifact) => ({ ...artifact, mission }))), [missions]);
	const runningMissions = missions.filter((mission) => ["RUNNING", "OBSERVING", "RECOVERING"].includes(mission.state)).length;
	const connectorCategories = useMemo(() => ["Todos", ...Array.from(new Set(connectorCatalog.map((entry) => entry.category))).sort()], [connectorCatalog]);
	const filteredConnectorCatalog = useMemo(() => {
		const query = connectorSearch.trim().toLocaleLowerCase();
		return connectorCatalog.filter((entry) => {
			const matchesCategory = connectorCategory === "Todos" || entry.category === connectorCategory;
			const matchesQuery = !query || `${entry.name} ${entry.description} ${entry.category}`.toLocaleLowerCase().includes(query);
			return matchesCategory && matchesQuery;
		});
	}, [connectorCatalog, connectorCategory, connectorSearch]);

  const handleAction = () => {
    if (kind === "tasks" || kind === "library") window.location.assign("/agentic");
    else if (kind === "projects") document.getElementById("new-project-name")?.focus();
    else if (kind === "scheduled") document.getElementById("new-schedule-objective")?.focus();
    else window.location.assign("/settings#agentic");
  };

  const createNewProject = async () => {
    if (!projectName.trim()) return;
    await runPending("create-project", async () => {
      try {
        await createProject(projectName, projectRoot);
        setProjectName(""); setProjectRoot(""); setNotice("Projeto criado e persistido no ContextStore."); await refresh();
      } catch (cause) { setNotice(cause instanceof Error ? cause.message : "Falha ao criar projeto."); }
    });
  };

  const saveProject = async (project: AgentProject) => {
    if (!editingProjectName.trim()) return;
    await runPending(`project-save:${project.id}`, async () => {
      try { await updateProject(project.id, editingProjectName, project.root ?? ""); setEditingProject(null); setNotice("Projeto atualizado."); await refresh(); }
      catch (cause) { setNotice(cause instanceof Error ? cause.message : "Falha ao atualizar projeto."); }
    });
  };

  const removeProject = async (project: AgentProject) => {
    if (!window.confirm(`Excluir o projeto ${project.name}? Memórias locais associadas também serão removidas.`)) return;
    await runPending(`project-delete:${project.id}`, async () => {
      try { await deleteProject(project.id); setNotice("Projeto excluído."); await refresh(); }
      catch (cause) { setNotice(cause instanceof Error ? cause.message : "Falha ao excluir projeto."); }
    });
  };

  const createNewSchedule = async () => {
    if (!scheduleObjective.trim()) return;
    await runPending("create-schedule", async () => {
      try { await createSchedule({ objective: scheduleObjective, interval_seconds: Number(scheduleInterval), project_id: scheduleProject || undefined, enabled: true }); setScheduleObjective(""); setNotice("Schedule criado; o worker usará o intervalo configurado."); await refresh(); }
      catch (cause) { setNotice(cause instanceof Error ? cause.message : "Falha ao criar schedule."); }
    });
  };

		const removeSchedule = async (schedule: AgentSchedule) => {
			if (!window.confirm("Excluir este schedule?")) return;
			await runPending(`schedule-delete:${schedule.id}`, async () => {
				try { await deleteSchedule(schedule.id); setNotice("Schedule excluído."); await refresh(); }
				catch (cause) { setNotice(cause instanceof Error ? cause.message : "Falha ao excluir schedule."); }
			});
		};

		const togglePlugin = async (type: "connector" | "mcp" | "remote-mcp" | "skill", id: string, enabled: boolean) => {
			await runPending(`plugin-toggle:${type}:${id}`, async () => {
				try {
					if (type === "connector") await setConnectorEnabled(id, enabled);
					else if (type === "skill") await setSkillEnabled(id, enabled);
					else await setMCPEnabled(id, enabled, type === "remote-mcp");
					setNotice(`${id} ${enabled ? "habilitado" : "desabilitado"}.`);
					await refresh();
				} catch (cause) { setNotice(cause instanceof Error ? cause.message : "Falha ao alterar lifecycle do plugin."); }
			});
		};

			const removePlugin = async (type: "connector" | "mcp" | "remote-mcp" | "skill", id: string) => {
				if (!window.confirm(`Remover ${id} do manifest durável? Essa ação não revoga credenciais upstream.`)) return;
				await runPending(`plugin-delete:${type}:${id}`, async () => {
					try {
						if (type === "connector") await removeConnector(id);
						else if (type === "skill") await removeSkill(id);
						else await removeMCP(id, type === "remote-mcp");
						setNotice(`${id} removido do manifest local.`);
						await refresh();
					} catch (cause) { setNotice(cause instanceof Error ? cause.message : "Falha ao remover o plugin."); }
				});
			};

			const createNewConnector = async () => {
			if (!newConnectorID.trim() || !newConnectorProvider.trim() || !newConnectorBaseURL.trim()) return;
			try {
				await registerConnector({
					id: newConnectorID.trim(),
					provider: newConnectorProvider.trim(),
					base_url: newConnectorBaseURL.trim(),
					...(newConnectorTokenEnv.trim() ? { token_env: newConnectorTokenEnv.trim() } : {}),
					...(newConnectorOAuthProvider.trim() ? { oauth_provider: newConnectorOAuthProvider.trim() } : {}),
					operations: [{
						name: "default",
						methods: newConnectorMethods.split(",").map((method) => method.trim().toUpperCase()).filter(Boolean),
						path_prefixes: [newConnectorPathPrefix.trim() || "/"],
					}],
				});
				setNewConnectorID(""); setNewConnectorProvider(""); setNewConnectorBaseURL(""); setNewConnectorTokenEnv(""); setNewConnectorOAuthProvider("");
				setNotice("Connector registrado no tenant atual. O runtime mantém apenas referências de credencial; o valor deve existir no ambiente do servidor ou no OAuth autorizado.");
				await refresh();
				} catch (cause) { setNotice(cause instanceof Error ? cause.message : "Falha ao registrar connector."); }
			};

			const createNewMCP = async () => {
				if (!newMCPID.trim() || !newMCPCommand.trim()) return;
				try {
					await registerMCP({ id: newMCPID.trim(), command: newMCPCommand.trim(), allowed_methods: newMCPMethods.split(",").map((method) => method.trim()).filter(Boolean) });
					setNewMCPID(""); setNewMCPCommand("");
					setNotice("MCP stdio registrado. O executável é validado e executado somente no host/workspace permitido pelo servidor.");
					await refresh();
				} catch (cause) { setNotice(cause instanceof Error ? cause.message : "Falha ao registrar MCP stdio."); }
			};

			const createNewRemoteMCP = async () => {
				if (!newRemoteMCPID.trim() || !newRemoteMCPURL.trim()) return;
				try {
					await registerRemoteMCP({ id: newRemoteMCPID.trim(), url: newRemoteMCPURL.trim(), ...(newRemoteMCPTokenEnv.trim() ? { token_env: newRemoteMCPTokenEnv.trim() } : {}), allowed_methods: newRemoteMCPMethods.split(",").map((method) => method.trim()).filter(Boolean) });
					setNewRemoteMCPID(""); setNewRemoteMCPURL(""); setNewRemoteMCPTokenEnv("");
					setNotice("Remote MCP registrado. O endpoint deve passar HTTPS/SSRF/DNS pinning; tokens permanecem no ambiente do servidor.");
					await refresh();
				} catch (cause) { setNotice(cause instanceof Error ? cause.message : "Falha ao registrar Remote MCP."); }
			};

			const createNewSkill = async () => {
				if (!newSkillID.trim() || !newSkillVersion.trim()) return;
				try {
					await registerSkill({ id: newSkillID.trim(), version: newSkillVersion.trim(), description: newSkillDescription.trim(), scopes: newSkillScopes.split(",").map((scope) => scope.trim()).filter(Boolean), tools: newSkillTools.split(",").map((tool) => tool.trim()).filter(Boolean) });
					setNewSkillID(""); setNewSkillDescription(""); setNewSkillScopes(""); setNewSkillTools("");
					setNotice("Manifesto de skill registrado como não confiável e habilitado pelo servidor para revisão/approval.");
					await refresh();
				} catch (cause) { setNotice(cause instanceof Error ? cause.message : "Falha ao registrar skill."); }
			};

			const renderContent = () => {
	    if (loading) return <div role="status" aria-live="polite" aria-busy="true" className="rounded-xl border border-dashed border-neutral-300 px-6 py-12 text-center text-sm text-neutral-500 dark:border-neutral-700">Consultando contratos agentic…</div>;
	    if (loadError) return <div role="alert" aria-live="assertive" className="rounded-xl border border-red-200 bg-red-50 px-6 py-8 text-center text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300"><p>{loadError}</p><button type="button" onClick={() => void refresh()} className="mt-4 rounded-lg border border-red-300 px-3 py-2 text-xs font-medium dark:border-red-800">Tentar novamente</button></div>;
    if (kind === "projects") return <div className="space-y-3">{projects.length ? projects.map((project) => <ResourceRow key={project.id} onDelete={() => void removeProject(project)}><div className="flex flex-wrap items-center justify-between gap-2"><div><p className="font-medium text-neutral-900 dark:text-white">{editingProject === project.id ? <input autoFocus value={editingProjectName} onChange={(event) => setEditingProjectName(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") void saveProject(project); if (event.key === "Escape") setEditingProject(null); }} className="h-8 rounded-lg border border-neutral-300 bg-transparent px-2 text-sm dark:border-neutral-700" /> : project.name}</p><p className="mt-1 text-xs text-neutral-500">{project.root || "workspace local gerenciado"} · atualizado {new Date(project.updated_at).toLocaleString()}</p></div>{editingProject === project.id ? <button onClick={() => void saveProject(project)} className="rounded-lg bg-neutral-900 px-3 py-2 text-xs text-white dark:bg-white dark:text-neutral-900">Salvar</button> : <button onClick={() => { setEditingProject(project.id); setEditingProjectName(project.name); }} className="rounded-lg border border-neutral-200 px-3 py-2 text-xs dark:border-neutral-700">Editar</button>}</div></ResourceRow>) : <EmptyState message="Crie um projeto para manter instruções, arquivos e contexto entre missões." action="Criar projeto" onAction={() => document.getElementById("new-project-name")?.focus()} />}</div>;
    if (kind === "tasks") return missions.length ? <div className="space-y-3">{missions.map((mission) => <ResourceRow key={mission.id}><Link to="/agentic" className="block"><div className="flex flex-wrap items-center justify-between gap-2"><p className="font-medium text-neutral-900 dark:text-white">{mission.objective}</p><span className="rounded-full bg-neutral-100 px-2 py-1 text-[10px] font-medium dark:bg-neutral-800">{mission.state}</span></div><p className="mt-1 text-xs text-neutral-500">{mission.id} · {mission.plan?.length ?? 0} passos · {mission.artifacts?.length ?? 0} artifacts · {mission.model || "modelo padrão"}</p></Link></ResourceRow>)}</div> : <EmptyState message="As missões criadas no Agentic Console aparecerão aqui com timeline e artifacts." action="Nova tarefa" onAction={() => window.location.assign("/agentic")} />;
    if (kind === "library") return artifacts.length ? <div className="grid gap-3 md:grid-cols-2">{artifacts.map(({ mission, ...artifact }) => <ResourceRow key={`${mission.id}-${artifact.id}`}><a href={`${API_BASE}/api/agent/v1/missions/${encodeURIComponent(mission.id)}/artifacts/${encodeURIComponent(artifact.id)}`} className="block"><p className="font-medium text-neutral-900 dark:text-white">{artifact.name}</p><p className="mt-1 text-xs text-neutral-500">{Math.round(artifact.size / 1024)} KiB · SHA-256 {artifact.sha256.slice(0, 16)}…</p><p className="mt-1 text-[10px] text-violet-600">Missão {mission.id}</p></a></ResourceRow>)}</div> : <EmptyState message="Os artifacts gerados aparecerão aqui com preview, hash, versão e download." action="Abrir Agentic Console" onAction={() => window.location.assign("/agentic")} />;
    if (kind === "scheduled") return <div className="space-y-3">{schedules.length ? schedules.map((schedule) => <ResourceRow key={schedule.id} onDelete={() => void removeSchedule(schedule)}><div className="flex flex-wrap items-center justify-between gap-2"><div><p className="font-medium text-neutral-900 dark:text-white">{schedule.objective}</p><p className="mt-1 text-xs text-neutral-500">a cada {schedule.interval_seconds}s · próxima {new Date(schedule.next_run_at).toLocaleString()}</p></div><span className={`rounded-full px-2 py-1 text-[10px] ${schedule.enabled ? "bg-emerald-50 text-emerald-700" : "bg-neutral-100 text-neutral-500"}`}>{schedule.enabled ? "ativo" : "pausado"}</span></div></ResourceRow>) : <EmptyState message="Nenhuma automação está agendada. Crie uma para ativar o worker persistente local." action="Agendar tarefa" onAction={() => document.getElementById("new-schedule-objective")?.focus()} />}</div>;
			if (kind === "skills") return skills.length ? <div className="space-y-3">{skills.map((skill) => <ResourceRow key={skill.id} onDelete={() => void removePlugin("skill", skill.id)}><div className="flex items-start justify-between gap-3"><div><p className="font-medium text-neutral-900 dark:text-white">{skill.id} <span className="text-xs text-neutral-400">v{skill.version}</span></p><p className="mt-1 text-xs text-neutral-500">{skill.description}</p><p className="mt-1 text-[10px] text-neutral-400">{skill.tools?.length ?? 0} tools · {skill.trusted ? "trusted" : "requer revisão"}</p></div><button type="button" onClick={() => void togglePlugin("skill", skill.id, !skill.enabled)} className="rounded-lg border border-neutral-200 px-2 py-1 text-[10px] dark:border-neutral-700">{skill.enabled ? "Desabilitar" : "Habilitar"}</button></div></ResourceRow>)}</div> : <EmptyState message="Nenhuma skill foi carregada pelo runtime. Instale um manifesto revisado no diretório de skills do servidor." action="Abrir configuração" onAction={() => window.location.assign("/settings#agentic")} />;
			return <div className="space-y-5"><section className="rounded-2xl border border-violet-200 bg-violet-50/60 p-4 dark:border-violet-900/60 dark:bg-violet-950/20">
				<div className="flex flex-wrap items-start justify-between gap-3"><div><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">Registrar connector seguro</h3><p className="mt-1 max-w-2xl text-xs leading-5 text-neutral-600 dark:text-neutral-300">Cadastre endpoint, operações e referências de credencial. O valor do token nunca é aceito nesta tela; use uma variável de ambiente do servidor ou o identificador de um OAuth já autorizado.</p></div><span className="rounded-full bg-white px-2.5 py-1 text-[10px] text-violet-700 dark:bg-neutral-900 dark:text-violet-300">owner/admin</span></div>
				<div className="mt-4 grid gap-3 md:grid-cols-2"><label className="text-xs text-neutral-600 dark:text-neutral-300">ID<input value={newConnectorID} onChange={(event) => setNewConnectorID(event.target.value)} placeholder="github-prod" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300">Provider<input value={newConnectorProvider} onChange={(event) => setNewConnectorProvider(event.target.value)} placeholder="GitHub" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300 md:col-span-2">Base URL HTTPS<input value={newConnectorBaseURL} onChange={(event) => setNewConnectorBaseURL(event.target.value)} placeholder="https://api.example.com" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300">Nome da variável de token (opcional)<input value={newConnectorTokenEnv} onChange={(event) => setNewConnectorTokenEnv(event.target.value)} placeholder="OLLAMA_GITHUB_TOKEN" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300">ID do provider OAuth (opcional)<input value={newConnectorOAuthProvider} onChange={(event) => setNewConnectorOAuthProvider(event.target.value)} placeholder="github" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300">Métodos permitidos<input value={newConnectorMethods} onChange={(event) => setNewConnectorMethods(event.target.value)} placeholder="GET,POST" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300">Prefixo de caminho<input value={newConnectorPathPrefix} onChange={(event) => setNewConnectorPathPrefix(event.target.value)} placeholder="/v1" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label></div>
				<button type="button" onClick={() => void createNewConnector()} disabled={!newConnectorID.trim() || !newConnectorProvider.trim() || !newConnectorBaseURL.trim()} className="mt-4 rounded-xl bg-violet-700 px-4 py-2.5 text-xs font-medium text-white disabled:cursor-not-allowed disabled:opacity-40">Registrar no tenant atual</button>
				</section><section className="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><div><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">Registrar MCP e skills</h3><p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">O cadastro grava somente manifestos. MCP local usa executável absoluto validado; Remote MCP usa HTTPS/SSRF/DNS pinning; skills começam não confiáveis. Não cole tokens nesta tela.</p></div><div className="mt-4 grid gap-4 lg:grid-cols-3"><div className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-700"><p className="text-xs font-medium text-neutral-700 dark:text-neutral-200">MCP stdio</p><input value={newMCPID} onChange={(event) => setNewMCPID(event.target.value)} placeholder="ID" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newMCPCommand} onChange={(event) => setNewMCPCommand(event.target.value)} placeholder="/absolute/path/executable" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newMCPMethods} onChange={(event) => setNewMCPMethods(event.target.value)} placeholder="tools/list,tools/call" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><button type="button" onClick={() => void createNewMCP()} disabled={!newMCPID.trim() || !newMCPCommand.trim()} className="mt-3 rounded-lg bg-neutral-900 px-3 py-2 text-[10px] text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900">Registrar MCP local</button></div><div className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-700"><p className="text-xs font-medium text-neutral-700 dark:text-neutral-200">Remote MCP</p><input value={newRemoteMCPID} onChange={(event) => setNewRemoteMCPID(event.target.value)} placeholder="ID" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newRemoteMCPURL} onChange={(event) => setNewRemoteMCPURL(event.target.value)} placeholder="https://mcp.example.com" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newRemoteMCPTokenEnv} onChange={(event) => setNewRemoteMCPTokenEnv(event.target.value)} placeholder="TOKEN_ENV (opcional)" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newRemoteMCPMethods} onChange={(event) => setNewRemoteMCPMethods(event.target.value)} placeholder="tools/list" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><button type="button" onClick={() => void createNewRemoteMCP()} disabled={!newRemoteMCPID.trim() || !newRemoteMCPURL.trim()} className="mt-3 rounded-lg bg-neutral-900 px-3 py-2 text-[10px] text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900">Registrar Remote MCP</button></div><div className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-700"><p className="text-xs font-medium text-neutral-700 dark:text-neutral-200">Skill manifest</p><input value={newSkillID} onChange={(event) => setNewSkillID(event.target.value)} placeholder="ID" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newSkillVersion} onChange={(event) => setNewSkillVersion(event.target.value)} placeholder="Versão" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newSkillDescription} onChange={(event) => setNewSkillDescription(event.target.value)} placeholder="Descrição" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newSkillScopes} onChange={(event) => setNewSkillScopes(event.target.value)} placeholder="Scopes separados por vírgula" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newSkillTools} onChange={(event) => setNewSkillTools(event.target.value)} placeholder="Tools separados por vírgula" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><button type="button" onClick={() => void createNewSkill()} disabled={!newSkillID.trim() || !newSkillVersion.trim()} className="mt-3 rounded-lg bg-neutral-900 px-3 py-2 text-[10px] text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900">Registrar skill</button></div></div></section><section>
			<div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
				<div><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">Aplicativos e conectores</h3><p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">Catálogo amplo de apps, APIs e MCPs. O botão de configuração nunca inventa uma conexão: cada provider exige OAuth, API key ou homologação própria.</p></div><span className="rounded-full bg-neutral-100 px-2.5 py-1 text-[10px] text-neutral-500 dark:bg-neutral-800 dark:text-neutral-300">{connectorCatalog.length} disponíveis no catálogo</span>
			</div>
			<div className="mt-4 flex flex-col gap-3 lg:flex-row"><input value={connectorSearch} onChange={(event) => setConnectorSearch(event.target.value)} aria-label="Pesquisar conectores" placeholder="Pesquisar conectores" className="h-10 min-w-0 flex-1 rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /><div role="tablist" aria-label="Categorias de conectores" className="flex gap-2 overflow-x-auto pb-1">{connectorCategories.map((category) => <button key={category} type="button" role="tab" aria-selected={connectorCategory === category} onClick={() => setConnectorCategory(category)} className={`whitespace-nowrap rounded-xl px-3 py-2 text-xs ${connectorCategory === category ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "border border-neutral-200 text-neutral-600 dark:border-neutral-700 dark:text-neutral-300"}`}>{category}</button>)}</div></div>
				<div className="mt-4 grid gap-3 md:grid-cols-2">{filteredConnectorCatalog.map((entry) => { const configuredConnector = connectors.find((connector) => connector.id === entry.id || connector.provider.toLocaleLowerCase() === entry.name.toLocaleLowerCase()); const configured = Boolean(configuredConnector); const statusLabel = !configured ? (entry.status === "available" ? "Disponível para configuração" : entry.status === "provider_selection_required" ? "Escolha o provedor fiscal" : "Configuração do operador necessária") : configuredConnector?.disabled ? "Configurado · desabilitado" : configuredConnector?.credential_configured ? "Configurado · habilitado · credencial presente" : "Configurado · habilitado · credencial ausente"; const actionLabel = configured ? "Ver estado" : entry.status === "provider_selection_required" ? "Escolher provedor" : entry.status === "available" ? "Configurar" : "Definir API"; return <article key={entry.id} className="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><div className="flex items-start gap-3"><div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-sm font-semibold text-neutral-700 dark:bg-neutral-800 dark:text-neutral-200">{entry.name.slice(0, 2).toUpperCase()}</div><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><h4 className="font-medium text-neutral-900 dark:text-white">{entry.name}</h4><span className="rounded-full bg-violet-50 px-2 py-1 text-[10px] text-violet-700 dark:bg-violet-950/30 dark:text-violet-300">{entry.category}</span></div><p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">{entry.description}</p><p className="mt-2 text-[10px] text-neutral-400">{statusLabel} · autenticação: {entry.auth}</p></div><button type="button" onClick={() => setNotice(configured ? `${entry.name}: veja o estado no bloco de connectors configurados abaixo. O runtime não chama a integração sem credencial e approval.` : `${entry.name}: abra Configurações para registrar ${entry.kind === "custom_api" ? "o endpoint e a credencial no servidor" : "o fluxo OAuth/MCP oficial"}. Nenhuma ação externa foi executada.`)} className="shrink-0 rounded-xl border border-neutral-200 px-3 py-2 text-[10px] font-medium text-neutral-700 dark:border-neutral-700 dark:text-neutral-200"><PlusIcon className="mr-1 inline h-3.5 w-3.5" />{actionLabel}</button></div></article>; })}</div>
					{!filteredConnectorCatalog.length && <EmptyState message="Nenhum conector corresponde à pesquisa ou categoria selecionada." />}</section><section><h3 className="mb-3 text-xs font-semibold uppercase tracking-[0.12em] text-neutral-500">Connectors configurados ({connectors.length})</h3><div className="space-y-3">{connectors.length ? connectors.map((connector) => <ResourceRow key={connector.id} onDelete={() => void removePlugin("connector", connector.id)}><div className="flex items-start justify-between gap-3"><div><div className="flex flex-wrap items-center gap-2"><p className="font-medium text-neutral-900 dark:text-white">{connector.provider} <span className="text-xs text-neutral-400">· {connector.id}</span></p><span className={`rounded-full px-2 py-1 text-[10px] ${connector.disabled ? "bg-neutral-100 text-neutral-500" : connector.credential_configured ? "bg-emerald-50 text-emerald-700" : "bg-amber-50 text-amber-700"}`}>{connector.disabled ? "desabilitado" : connector.credential_configured ? "habilitado · credencial presente" : "habilitado · credencial ausente"}</span></div><p className="mt-1 truncate text-xs text-neutral-500">{connector.base_url}</p><p className="mt-1 text-[10px] text-neutral-400">{connector.operations?.length ?? 0} operações · segredos mantidos no servidor</p></div><button type="button" onClick={() => void togglePlugin("connector", connector.id, Boolean(connector.disabled))} className="rounded-lg border border-neutral-200 px-2 py-1 text-[10px] dark:border-neutral-700">{connector.disabled ? "Habilitar" : "Desabilitar"}</button></div></ResourceRow>) : <EmptyState message="Nenhum connector configurado. O catálogo acima mostra o que pode ser conectado sem fingir autorização." />}</div></section><section><h3 className="mb-3 text-xs font-semibold uppercase tracking-[0.12em] text-neutral-500">MCP local e remoto ({mcpServers.length})</h3><div className="space-y-3">{mcpServers.length ? mcpServers.map((server) => <ResourceRow key={`${server.transport ?? "stdio"}-${server.id}`} onDelete={() => void removePlugin(server.transport === "streamable-http" ? "remote-mcp" : "mcp", server.id)}><div className="flex items-start justify-between gap-3"><div><p className="font-medium text-neutral-900 dark:text-white">{server.id} <span className="ml-2 rounded-full bg-violet-50 px-2 py-1 text-[10px] text-violet-700 dark:bg-violet-950/30 dark:text-violet-300">{server.transport ?? "stdio"}</span></p><p className="mt-1 truncate font-mono text-xs text-neutral-500">{server.command ?? server.url ?? "endpoint remoto"} {server.command ? (server.args ?? []).join(" ") : ""}</p><p className="mt-1 text-[10px] text-neutral-400">{server.allowed_methods?.length ?? 0} métodos allowlisted · credencial somente no servidor</p></div><button type="button" onClick={() => void togglePlugin(server.transport === "streamable-http" ? "remote-mcp" : "mcp", server.id, Boolean(server.disabled))} className="rounded-lg border border-neutral-200 px-2 py-1 text-[10px] dark:border-neutral-700">{server.disabled ? "Habilitar" : "Desabilitar"}</button></div></ResourceRow>) : <EmptyState message="Nenhum servidor MCP carregado. O comando ou endpoint precisa estar allowlisted no servidor." />}</div></section><p className="text-xs text-neutral-500">{cliCount} CLI(s) de integração detectado(s). A instalação e o registro continuam protegidos no ambiente do operador.</p></div>;
  };

  return <SidebarLayout title={copy.title} sidebar={<AppSidebar current={current} />}><div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-900"><div className="mx-auto w-full max-w-6xl px-6 pb-14 pt-10 lg:px-12"><div className="flex flex-col gap-5 border-b border-neutral-200 pb-8 dark:border-neutral-800 md:flex-row md:items-end md:justify-between"><div className="max-w-2xl"><div className="mb-3 flex items-center gap-2 text-xs font-medium text-violet-600 dark:text-violet-300"><SparklesIcon className="h-4 w-4" />{copy.eyebrow}</div><h2 className="font-rounded text-3xl font-semibold tracking-tight text-neutral-950 dark:text-white">{copy.title}</h2><p className="mt-3 text-sm leading-6 text-neutral-500 dark:text-neutral-400">{copy.description}</p></div><button onClick={handleAction} className="inline-flex items-center justify-center gap-2 rounded-xl bg-neutral-950 px-4 py-2.5 text-sm font-medium text-white shadow-sm dark:bg-white dark:text-neutral-950"><PlusIcon className="h-4 w-4" />{copy.action}</button></div>{notice && <div role="status" aria-live="polite" className="mt-5 rounded-xl border border-violet-200 bg-violet-50 px-4 py-3 text-xs leading-5 text-violet-800 dark:border-violet-900/60 dark:bg-violet-950/20 dark:text-violet-200">{notice}</div>}

{kind === "projects" && <section className="mt-6 grid gap-3 rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900 md:grid-cols-[1fr_1fr_auto]"><input id="new-project-name" aria-label="Nome do novo projeto" value={projectName} onChange={(event) => setProjectName(event.target.value)} placeholder="Nome do projeto" className="h-10 rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700" /><input aria-label="Workspace do novo projeto" value={projectRoot} onChange={(event) => setProjectRoot(event.target.value)} placeholder="Workspace (opcional)" className="h-10 rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700" /><button type="button" onClick={() => void createNewProject()} disabled={!projectName.trim() || Boolean(pendingActions["create-project"])} aria-busy={Boolean(pendingActions["create-project"])} className="h-10 rounded-xl bg-neutral-900 px-4 text-sm text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900">{pendingActions["create-project"] ? "Criando…" : "Criar"}</button></section>}
{kind === "scheduled" && <section className="mt-6 grid gap-3 rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900 md:grid-cols-[1.4fr_0.5fr_0.8fr_auto]"><input id="new-schedule-objective" aria-label="Objetivo do novo schedule" value={scheduleObjective} onChange={(event) => setScheduleObjective(event.target.value)} placeholder="Objetivo recorrente" className="h-10 rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700" /><input type="number" min="1" value={scheduleInterval} onChange={(event) => setScheduleInterval(event.target.value)} aria-label="Intervalo em segundos" className="h-10 rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700" /><select value={scheduleProject} onChange={(event) => setScheduleProject(event.target.value)} aria-label="Projeto do schedule" className="h-10 rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700"><option value="">Sem projeto</option>{projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}</select><button type="button" onClick={() => void createNewSchedule()} disabled={!scheduleObjective.trim() || Boolean(pendingActions["create-schedule"])} aria-busy={Boolean(pendingActions["create-schedule"])} className="h-10 rounded-xl bg-neutral-900 px-4 text-sm text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900">{pendingActions["create-schedule"] ? "Agendando…" : "Agendar"}</button></section>}

<div className="mt-7 grid gap-3 sm:grid-cols-3"><StatCard label="Status" value={kind === "tasks" ? `${runningMissions} em execução` : kind === "projects" ? `${projects.length} projetos` : kind === "scheduled" ? `${schedules.length} schedules` : kind === "library" ? `${artifacts.length} artifacts` : "Catálogo real"} tone="green" /><StatCard label="Segurança" value="Approvals ativos" tone="violet" /><StatCard label="Persistência" value="Local-first" /></div><div className="mt-7 grid gap-5 lg:grid-cols-[1.45fr_0.8fr]"><section className="rounded-2xl border border-neutral-200/80 bg-neutral-50/50 p-6 dark:border-neutral-800 dark:bg-neutral-950/30"><div className="flex items-center justify-between"><div><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">Dados persistidos</h3><p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">Esta superfície consulta o contrato agentic, não fixtures visuais.</p></div><CheckCircleIcon className="h-5 w-5 text-emerald-500" /></div><div className="mt-5">{renderContent()}</div></section><aside className="space-y-5"><section className="rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900"><div className="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><LockClosedIcon className="h-4 w-4 text-emerald-500" />Política ativa</div><ul className="mt-4 space-y-3 text-xs leading-5 text-neutral-500 dark:text-neutral-400"><li className="flex gap-2"><CheckCircleIcon className="mt-0.5 h-4 w-4 shrink-0 text-emerald-500" />Secrets não aparecem na interface.</li><li className="flex gap-2"><CheckCircleIcon className="mt-0.5 h-4 w-4 shrink-0 text-emerald-500" />Ações externas exigem approval.</li><li className="flex gap-2"><CheckCircleIcon className="mt-0.5 h-4 w-4 shrink-0 text-emerald-500" />Cross-tenant é rejeitado no servidor.</li></ul></section><section className="rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900"><div className="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><ClockIcon className="h-4 w-4 text-violet-500" />Próximos passos</div><p className="mt-3 text-xs leading-5 text-neutral-500 dark:text-neutral-400">Para criar uma missão complexa, use o console com seleção de provider, projeto, timeline e approvals.</p><Link to="/agentic" className="mt-4 inline-flex items-center gap-2 text-xs font-medium text-violet-600 dark:text-violet-300">Abrir Agentic Console <ArrowRightIcon className="h-3.5 w-3.5" /></Link></section></aside></div></div></div></SidebarLayout>;
}
