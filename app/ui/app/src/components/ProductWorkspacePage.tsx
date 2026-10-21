import { Link, useNavigate } from "@tanstack/react-router";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
	  createProject,
	  deleteProject,
	deleteMission,
	listCLIStatus,
	listConnectorCatalog,
	listConnectors,
  listMCPServers,
  listMissions,
  listPluginStatuses,
  listActiveSkills,
  activateSkill,
  deactivateSkill,
  listProjects,
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
import type { AgentConnector, AgentConnectorCatalogEntry, AgentMCPServer, AgentMission, AgentProject, AgentSkill } from "@/lib/agenticClient";
import type { AgentPluginStatus } from "@/lib/pluginStatus";
import {
  emptySkillContext,
  parseTokenBudget,
  type AgentSkillActivation,
  type AgentSkillSelection,
  type SkillContextInput,
} from "@/lib/skillCatalog";
import { PluginStatusList } from "@/components/PluginStatusList";
import { SkillCatalogPanel } from "@/components/SkillCatalogPanel";
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
import { missionStateLabel } from "@/lib/labels";
import { confirmDialog } from "@/lib/confirmDialog";
import { AppSidebar, type AppSection } from "@/components/AppSidebar";
import { ImportProjectDialog } from "@/components/ImportProjectDialog";
import { AskDocumentsPanel } from "@/components/AskDocumentsPanel";
import { SidebarLayout } from "@/components/layout/layout";
import { useT } from "@/i18n";

type ProductPageKind = "projects" | "skills" | "plugins" | "tasks";

const pageCopyKeys: Record<ProductPageKind, { title: string; eyebrow: string; description: string; action: string }> = {
  projects: { title: "productWorkspace.projects.title", eyebrow: "productWorkspace.projects.eyebrow", description: "productWorkspace.projects.description", action: "productWorkspace.projects.action" },
  skills: { title: "productWorkspace.skills.title", eyebrow: "productWorkspace.skills.eyebrow", description: "productWorkspace.skills.description", action: "productWorkspace.skills.action" },
  plugins: { title: "productWorkspace.plugins.title", eyebrow: "productWorkspace.plugins.eyebrow", description: "productWorkspace.plugins.description", action: "productWorkspace.plugins.action" },
  tasks: { title: "productWorkspace.tasks.title", eyebrow: "productWorkspace.tasks.eyebrow", description: "productWorkspace.tasks.description", action: "productWorkspace.tasks.action" },
};

function StatCard({ label, value, tone = "neutral" }: { label: string; value: string; tone?: "neutral" | "green" | "violet" }) {
  const toneClass = tone === "green" ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300" : tone === "violet" ? "bg-violet-50 text-violet-700 dark:bg-violet-950/30 dark:text-violet-300" : "bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300";
  return <div className="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><div className="text-[11px] uppercase tracking-[0.12em] text-neutral-400">{label}</div><div className={`mt-3 inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${toneClass}`}>{value}</div></div>;
}

function EmptyState({ message, action, onAction }: { message: string; action?: string; onAction?: () => void }) {
  return <div className="rounded-xl border border-dashed border-neutral-300 px-6 py-10 text-center dark:border-neutral-700"><FolderOpenIcon className="mx-auto h-8 w-8 text-neutral-300 dark:text-neutral-600" /><p className="mx-auto mt-3 max-w-md text-sm text-neutral-500 dark:text-neutral-400">{message}</p>{action && onAction && <button onClick={onAction} className="mt-5 inline-flex items-center gap-2 rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800">{action}<ArrowRightIcon className="h-3.5 w-3.5" /></button>}</div>;
}

function ResourceRow({ children, onDelete, deleteLabel, deleteDisabled = false }: { children: React.ReactNode; onDelete?: () => void; deleteLabel?: string; deleteDisabled?: boolean }) {
	  const { t } = useT();
	  const resolvedDeleteLabel = deleteLabel ?? t("productWorkspace.resource.delete");
	  return <div className="flex items-center justify-between gap-3 rounded-xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><div className="min-w-0 flex-1">{children}</div>{onDelete && <button type="button" onClick={onDelete} disabled={deleteDisabled} className="rounded-lg p-2 text-neutral-400 hover:bg-red-50 hover:text-red-600 disabled:cursor-wait disabled:opacity-40 dark:hover:bg-red-950/30" aria-label={resolvedDeleteLabel} title={resolvedDeleteLabel}><TrashIcon className="h-4 w-4" /></button>}</div>;
}

export function skillActivationsFromSelection(selection: AgentSkillSelection | null): AgentSkillActivation[] {
  // O painel é por contexto: as duas listas do servidor (selecionadas e
  // excluídas) são exatamente as skills ATIVADas para aquele contexto — as
  // excluídas saíram por orçamento/estado, não por falta de ativação.
  if (!selection) return [];
  // `selected` e `skipped` chegam do servidor e podem vir null quando o contexto ainda nao
  // tem skills associadas. Sem as duas guardas, a pagina /skills rebentava com
  // "Cannot read properties of null (reading 'map')" e caia no ecra de erro.
  const selecionadas = selection.selected ?? [];
  const ignoradas = selection.skipped ?? [];
  const ids = [...selecionadas.map((skill) => skill.id), ...ignoradas.map((skip) => skip.id)];
  return [...new Set(ids)].map((id) => ({ skill_id: id }));
}

export function ProductWorkspacePage({ kind, createdMissionId }: { kind: ProductPageKind; createdMissionId?: string }) {
  const { t } = useT();
  const copyKeys = pageCopyKeys[kind];
  const copy = { title: t(copyKeys.title), eyebrow: t(copyKeys.eyebrow), description: t(copyKeys.description), action: t(copyKeys.action) };
  const current = kind as AppSection;
  const navigate = useNavigate();
  const [notice, setNotice] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [pendingActions, setPendingActions] = useState<Record<string, boolean>>({});
  const [projects, setProjects] = useState<AgentProject[]>([]);
  const [missions, setMissions] = useState<AgentMission[]>([]);
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
  const [skillContext, setSkillContext] = useState<SkillContextInput>(emptySkillContext);
  const [skillSelection, setSkillSelection] = useState<AgentSkillSelection | null>(null);
  const [skillPanelError, setSkillPanelError] = useState<string | null>(null);
  const [pluginStatuses, setPluginStatuses] = useState<AgentPluginStatus[]>([]);
  const [pluginStatusError, setPluginStatusError] = useState<string | null>(null);
  const [cliCount, setCliCount] = useState(0);
  const [projectName, setProjectName] = useState("");
  const [projectRoot, setProjectRoot] = useState("");
	  const [editingProject, setEditingProject] = useState<string | null>(null);
	  const [editingProjectName, setEditingProjectName] = useState("");
	  const [importOpen, setImportOpen] = useState(false);
	  const [askProjectId, setAskProjectId] = useState("");

  const refresh = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      if (kind === "projects") setProjects((await listProjects()).projects ?? []);
      if (kind === "tasks") setMissions((await listMissions()).missions ?? []);
		if (kind === "skills") setSkills((await listSkills()).skills ?? []);
		if (kind === "plugins") {
			const [connectorResult, catalogResult, mcpResult, cliResult] = await Promise.all([listConnectors(), listConnectorCatalog(), listMCPServers(), listCLIStatus()]);
			setConnectors(connectorResult.connectors ?? []);
			setConnectorCatalog(catalogResult.connectors ?? []);
			setMcpServers(mcpResult.servers ?? []);
        setCliCount(cliResult.tools.filter((item) => item.installed).length);
        // Estado real dos plugins instalados (§11.3). Falha aqui não derruba a
        // página: mostramos o motivo e seguimos sem inventar prontidão.
        try {
          const statusResult = await listPluginStatuses();
          setPluginStatuses(statusResult.statuses);
          setPluginStatusError(null);
        } catch (statusCause) {
          setPluginStatuses([]);
          setPluginStatusError(statusCause instanceof Error ? statusCause.message : t("productWorkspace.plugins.statusError"));
        }
      }
      if (kind === "skills") {
        // Resolve o catálogo para o contexto inicial (vazio = só curingas).
        // Falha aqui aparece no painel; nunca é mascarada como "nenhuma skill".
        try {
          const result = await listActiveSkills({ projectId: "", agentId: "", model: "", tokenBudget: null });
          setSkillSelection(result);
          setSkillPanelError(null);
        } catch (cause) {
          setSkillSelection(null);
          setSkillPanelError(cause instanceof Error ? cause.message : t("productWorkspace.skills.resolveFailed"));
        }
      }
    } catch (cause) {
      const message = cause instanceof Error ? cause.message : t("productWorkspace.loadError");
      setLoadError(message);
      setNotice(message);
    } finally {
      setLoading(false);
    }
  }, [kind, t]);

  useEffect(() => { void refresh(); }, [refresh]);

  // Resolve o catálogo de skills para o contexto informado na tela.
  const resolveSkillCatalog = useCallback(async () => {
    const parsed = parseTokenBudget(skillContext.tokenBudget);
    if (parsed.error) {
      setSkillPanelError(parsed.error);
      return;
    }
    setPendingActions((current) => ({ ...current, "skill-catalog": true }));
    try {
      const result = await listActiveSkills({
        projectId: skillContext.projectId,
        agentId: skillContext.agentId,
        model: skillContext.model,
        tokenBudget: parsed.value,
      });
      setSkillSelection(result);
      setSkillPanelError(null);
    } catch (cause) {
      setSkillSelection(null);
      setSkillPanelError(cause instanceof Error ? cause.message : t("productWorkspace.skills.resolveFailed"));
    } finally {
      setPendingActions((current) => {
        const next = { ...current };
        delete next["skill-catalog"];
        return next;
      });
    }
  }, [skillContext, t]);

  const activateSkillForContext = useCallback(
    async (id: string, context: SkillContextInput) => {
      try {
        await activateSkill(id, { projectId: context.projectId, agentId: context.agentId, model: context.model });
        await resolveSkillCatalog();
      } catch (cause) {
        // Recusa do servidor (skill não confiável, por exemplo) é mostrada como
        // está — a interface não promove a skill por conta própria.
        setSkillPanelError(cause instanceof Error ? cause.message : t("productWorkspace.skills.activateFailed"));
      }
    },
    [resolveSkillCatalog, t],
  );

  const deactivateSkillForContext = useCallback(
    async (id: string, context: SkillContextInput) => {
      try {
        await deactivateSkill(id, { projectId: context.projectId, agentId: context.agentId, model: context.model });
        await resolveSkillCatalog();
      } catch (cause) {
        setSkillPanelError(cause instanceof Error ? cause.message : t("productWorkspace.skills.deactivateFailed"));
      }
    },
    [resolveSkillCatalog, t],
  );

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
    if (kind === "tasks") void navigate({ to: "/agentic" });
    else if (kind === "projects") document.getElementById("new-project-name")?.focus();
    else void navigate({ to: "/settings", hash: "agentic" });
  };

	  const createNewProject = async () => {
	    if (!projectName.trim()) return;
	    await runPending("create-project", async () => {
	      try {
	        await createProject(projectName, projectRoot);
	        setProjectName(""); setProjectRoot(""); setNotice(t("productWorkspace.projects.created")); await refresh();
	      } catch (cause) { setNotice(cause instanceof Error ? cause.message : t("productWorkspace.projects.createFailed")); }
	    });
	  };

  const saveProject = async (project: AgentProject) => {
    if (!editingProjectName.trim()) return;
    await runPending(`project-save:${project.id}`, async () => {
      try { await updateProject(project.id, editingProjectName, project.root ?? ""); setEditingProject(null); setNotice(t("productWorkspace.projects.updated")); await refresh(); }
      catch (cause) { setNotice(cause instanceof Error ? cause.message : t("productWorkspace.projects.updateFailed")); }
    });
  };

  const removeProject = async (project: AgentProject) => {
    if (!(await confirmDialog(t("productWorkspace.projects.deleteConfirm", { name: project.name }), { danger: true, confirmLabel: t("common.delete") }))) return;
    await runPending(`project-delete:${project.id}`, async () => {
      try { await deleteProject(project.id); setNotice(t("productWorkspace.projects.deleted")); await refresh(); }
      catch (cause) { setNotice(cause instanceof Error ? cause.message : t("productWorkspace.projects.deleteFailed")); }
    });
  };

  const removeMission = async (mission: AgentMission) => {
    if (!(await confirmDialog(t("productWorkspace.tasks.deleteConfirm", { objective: mission.objective }), { danger: true, confirmLabel: t("common.delete") }))) return;
    await runPending(`mission-delete:${mission.id}`, async () => {
      try { await deleteMission(mission.id); setNotice(t("productWorkspace.tasks.deleted")); await refresh(); }
      catch (cause) { setNotice(cause instanceof Error ? cause.message : t("productWorkspace.tasks.deleteFailed")); }
    });
  };


		const togglePlugin = async (type: "connector" | "mcp" | "remote-mcp" | "skill", id: string, enabled: boolean) => {
			await runPending(`plugin-toggle:${type}:${id}`, async () => {
				try {
					if (type === "connector") await setConnectorEnabled(id, enabled);
					else if (type === "skill") await setSkillEnabled(id, enabled);
					else await setMCPEnabled(id, enabled, type === "remote-mcp");
					setNotice(t("productWorkspace.plugins.toggleNotice", { id, status: enabled ? t("productWorkspace.plugins.enabled") : t("productWorkspace.plugins.disabled") }));
					await refresh();
				} catch (cause) { setNotice(cause instanceof Error ? cause.message : t("productWorkspace.plugins.toggleFailed")); }
			});
		};

			const removePlugin = async (type: "connector" | "mcp" | "remote-mcp" | "skill", id: string) => {
				if (!(await confirmDialog(t("productWorkspace.plugins.removeConfirm", { id }), { danger: true, confirmLabel: t("common.remove") }))) return;
				await runPending(`plugin-delete:${type}:${id}`, async () => {
					try {
						if (type === "connector") await removeConnector(id);
						else if (type === "skill") await removeSkill(id);
						else await removeMCP(id, type === "remote-mcp");
						setNotice(t("productWorkspace.plugins.removeNotice", { id }));
						await refresh();
					} catch (cause) { setNotice(cause instanceof Error ? cause.message : t("productWorkspace.plugins.removeFailed")); }
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
				setNotice(t("productWorkspace.plugins.connectorRegistered"));
				await refresh();
				} catch (cause) { setNotice(cause instanceof Error ? cause.message : t("productWorkspace.plugins.connectorRegisterFailed")); }
			};

			const createNewMCP = async () => {
				if (!newMCPID.trim() || !newMCPCommand.trim()) return;
				try {
					await registerMCP({ id: newMCPID.trim(), command: newMCPCommand.trim(), allowed_methods: newMCPMethods.split(",").map((method) => method.trim()).filter(Boolean) });
					setNewMCPID(""); setNewMCPCommand("");
					setNotice(t("productWorkspace.plugins.mcpRegistered"));
					await refresh();
				} catch (cause) { setNotice(cause instanceof Error ? cause.message : t("productWorkspace.plugins.mcpRegisterFailed")); }
			};

			const createNewRemoteMCP = async () => {
				if (!newRemoteMCPID.trim() || !newRemoteMCPURL.trim()) return;
				try {
					await registerRemoteMCP({ id: newRemoteMCPID.trim(), url: newRemoteMCPURL.trim(), ...(newRemoteMCPTokenEnv.trim() ? { token_env: newRemoteMCPTokenEnv.trim() } : {}), allowed_methods: newRemoteMCPMethods.split(",").map((method) => method.trim()).filter(Boolean) });
					setNewRemoteMCPID(""); setNewRemoteMCPURL(""); setNewRemoteMCPTokenEnv("");
					setNotice(t("productWorkspace.plugins.remoteMcpRegistered"));
					await refresh();
				} catch (cause) { setNotice(cause instanceof Error ? cause.message : t("productWorkspace.plugins.remoteMcpRegisterFailed")); }
			};

			const createNewSkill = async () => {
				if (!newSkillID.trim() || !newSkillVersion.trim()) return;
				try {
					await registerSkill({ id: newSkillID.trim(), version: newSkillVersion.trim(), description: newSkillDescription.trim(), scopes: newSkillScopes.split(",").map((scope) => scope.trim()).filter(Boolean), tools: newSkillTools.split(",").map((tool) => tool.trim()).filter(Boolean) });
					setNewSkillID(""); setNewSkillDescription(""); setNewSkillScopes(""); setNewSkillTools("");
					setNotice(t("productWorkspace.plugins.skillRegistered"));
					await refresh();
				} catch (cause) { setNotice(cause instanceof Error ? cause.message : t("productWorkspace.plugins.skillRegisterFailed")); }
			};

			// E-400 (port da paridade de UI, medido pelos 4 casos de TasksJourney.test.tsx):
			// *sucesso só se confirma quando a missão criada volta da LISTAGEM PERSISTIDA; um registro ausente é
			// reportado em vez de presumido.* O banner recebe foco para quem usa leitor de tela não perder o aviso.
			const createdBannerRef = useRef<HTMLDivElement>(null);
			const createdMission = useMemo(
				() => (kind === "tasks" && createdMissionId ? missions.find((mission) => mission.id === createdMissionId) : undefined),
				[kind, createdMissionId, missions],
			);
			const showCreatedBanner = kind === "tasks" && Boolean(createdMissionId) && !loading && !loadError;
			useEffect(() => {
				if (showCreatedBanner) createdBannerRef.current?.focus();
			}, [showCreatedBanner]);
			const openNewTask = () => void navigate({ to: "/tasks/new" });

			const renderContent = () => {
	    if (loading) return <div role="status" aria-live="polite" aria-busy="true" className="rounded-xl border border-dashed border-neutral-300 px-6 py-12 text-center text-sm text-neutral-500 dark:border-neutral-700">{t("productWorkspace.loading")}</div>;
	    if (loadError) return <div role="alert" aria-live="assertive" className="rounded-xl border border-red-200 bg-red-50 px-6 py-8 text-center text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300"><p>{loadError}</p><button type="button" onClick={() => void refresh()} className="mt-4 rounded-lg border border-red-300 px-3 py-2 text-xs font-medium dark:border-red-800">{t("common.retry")}</button></div>;
    if (kind === "projects") return <div className="space-y-3">{projects.length ? projects.map((project) => <ResourceRow key={project.id} onDelete={() => void removeProject(project)}><div className="flex flex-wrap items-center justify-between gap-2"><div><p className="font-medium text-neutral-900 dark:text-white">{editingProject === project.id ? <input autoFocus value={editingProjectName} onChange={(event) => setEditingProjectName(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") void saveProject(project); if (event.key === "Escape") setEditingProject(null); }} className="h-8 rounded-lg border border-neutral-300 bg-transparent px-2 text-sm dark:border-neutral-700" /> : project.name}</p><p className="mt-1 text-xs text-neutral-500">{project.root || t("productWorkspace.projects.localWorkspace")} {t("productWorkspace.projects.updatedAt")} {new Date(project.updated_at).toLocaleString()}</p></div>{editingProject === project.id ? <button onClick={() => void saveProject(project)} className="rounded-lg bg-neutral-900 px-3 py-2 text-xs text-white dark:bg-white dark:text-neutral-900">{t("common.save")}</button> : <button onClick={() => { setEditingProject(project.id); setEditingProjectName(project.name); }} className="rounded-lg border border-neutral-200 px-3 py-2 text-xs dark:border-neutral-700">{t("common.edit")}</button>}</div></ResourceRow>) : <EmptyState message={t("productWorkspace.projects.empty")} action={t("productWorkspace.projects.emptyAction")} onAction={() => document.getElementById("new-project-name")?.focus()} />}</div>;
    if (kind === "tasks") {
      const createdBanner = showCreatedBanner && (createdMission
        ? <div ref={createdBannerRef} tabIndex={-1} role="status" aria-live="polite" className="mb-4 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/30 dark:text-emerald-200"><p className="font-medium">{t("productWorkspace.tasks.created")}</p><p className="mt-1 break-words text-xs">{createdMission.objective} {t("productWorkspace.tasks.state")} {createdMission.state}</p></div>
        : <div ref={createdBannerRef} tabIndex={-1} role="alert" aria-live="assertive" className="mb-4 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200"><p className="font-medium">{t("productWorkspace.tasks.missingTitle")}</p><p className="mt-1 break-all text-xs">{t("common.id")} {createdMissionId}. {t("productWorkspace.tasks.missingHint")}</p><button type="button" onClick={() => void refresh()} className="mt-3 rounded-lg border border-amber-300 px-3 py-1.5 text-xs font-medium dark:border-amber-800">{t("productWorkspace.tasks.refresh")}</button></div>);
      if (!missions.length) return <>{createdBanner}<EmptyState message={t("productWorkspace.tasks.empty")} action={t("productWorkspace.tasks.emptyAction")} onAction={openNewTask} /></>;
      // MEDIDO (E-403): o teste procura `node.type === "li" && node.props["aria-current"] === "true"`.
      // A minha primeira versão pôs o `aria-current` no `<Link>` (`<a>`) e o `ResourceRow` renderiza um `<div>` ⇒
      // `findAll` devolvia `[]`. ***A asserção não pede "algo que indique a missão criada": ela pede um item de LISTA
      // marcado — e a estrutura da lista faz parte do contrato.***
      return <>{createdBanner}<ul aria-label={t("productWorkspace.tasks.listLabel")} className="space-y-3">{missions.map((mission) => { const missionActive = ["RUNNING", "OBSERVING", "RECOVERING", "PLANNING", "PENDING"].includes(mission.state); const isCreated = mission.id === createdMissionId; return (<li key={mission.id} aria-current={isCreated ? "true" : undefined} className={isCreated ? "rounded-xl ring-2 ring-emerald-400" : undefined}><ResourceRow onDelete={() => void removeMission(mission)} deleteDisabled={missionActive} deleteLabel={missionActive ? t("productWorkspace.tasks.cancelBeforeDelete") : t("productWorkspace.tasks.deleteLabel")}><Link to="/agentic" className="block"><div className="flex flex-wrap items-center justify-between gap-2"><p className="font-medium text-neutral-900 dark:text-white">{mission.objective}</p><span className="rounded-full bg-neutral-100 px-2 py-1 text-[10px] font-medium dark:bg-neutral-800">{missionStateLabel(mission.state)}</span></div><p className="mt-1 text-xs text-neutral-500">{t("productWorkspace.tasks.meta", { id: mission.id, steps: mission.plan?.length ?? 0, artifacts: mission.artifacts?.length ?? 0, model: mission.model || t("productWorkspace.tasks.defaultModel") })}</p></Link></ResourceRow></li>); })}</ul></>;
    }
			if (kind === "skills") return <div className="space-y-5"><section className="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
				<div><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">{t("productWorkspace.skills.catalogTitle")}</h3><p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">{t("productWorkspace.skills.catalogDescription")}</p></div>
				<div className="mt-4"><SkillCatalogPanel
					skills={skills}
					activations={skillActivationsFromSelection(skillSelection)}
					selection={skillSelection}
					context={skillContext}
					onContextChange={setSkillContext}
					onResolve={() => void resolveSkillCatalog()}
					onActivate={(id, context) => void activateSkillForContext(id, context)}
					onDeactivate={(id, context) => void deactivateSkillForContext(id, context)}
					pending={Boolean(pendingActions["skill-catalog"])}
					error={skillPanelError}
				/></div>
			</section>{skills.length ? <div className="space-y-3">{skills.map((skill) => <ResourceRow key={skill.id} onDelete={() => void removePlugin("skill", skill.id)}><div className="flex items-start justify-between gap-3"><div><p className="font-medium text-neutral-900 dark:text-white">{skill.id} <span className="text-xs text-neutral-400">v{skill.version}</span></p><p className="mt-1 text-xs text-neutral-500">{skill.description}</p><p className="mt-1 text-[10px] text-neutral-400">{t("productWorkspace.skills.meta", { count: skill.tools?.length ?? 0, trust: skill.trusted ? t("productWorkspace.skills.trusted") : t("productWorkspace.skills.needsReview") })}</p></div><button type="button" onClick={() => void togglePlugin("skill", skill.id, !skill.enabled)} className="rounded-lg border border-neutral-200 px-2 py-1 text-[10px] dark:border-neutral-700">{skill.enabled ? t("common.disable") : t("common.enable")}</button></div></ResourceRow>)}</div> : <EmptyState message={t("productWorkspace.skills.empty")} action={t("productWorkspace.skills.emptyAction")} onAction={() => void navigate({ to: "/settings", hash: "agentic" })} />}</div>;
			return <div className="space-y-5"><section className="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
				<div><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">{t("productWorkspace.plugins.installedTitle")}</h3><p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">{t("productWorkspace.plugins.installedDescription")}</p></div>
				<div className="mt-4"><PluginStatusList statuses={pluginStatuses} error={pluginStatusError} /></div>
			</section><section className="rounded-2xl border border-violet-200 bg-violet-50/60 p-4 dark:border-violet-900/60 dark:bg-violet-950/20">
				<div className="flex flex-wrap items-start justify-between gap-3"><div><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">{t("productWorkspace.plugins.connectorTitle")}</h3><p className="mt-1 max-w-2xl text-xs leading-5 text-neutral-600 dark:text-neutral-300">{t("productWorkspace.plugins.connectorDescription")}</p></div><span className="rounded-full bg-white px-2.5 py-1 text-[10px] text-violet-700 dark:bg-neutral-900 dark:text-violet-300">owner/admin</span></div>
				<div className="mt-4 grid gap-3 md:grid-cols-2"><label className="text-xs text-neutral-600 dark:text-neutral-300">{t("common.id")}<input value={newConnectorID} onChange={(event) => setNewConnectorID(event.target.value)} placeholder="github-prod" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300">{t("common.provider")}<input value={newConnectorProvider} onChange={(event) => setNewConnectorProvider(event.target.value)} placeholder="GitHub" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300 md:col-span-2">{t("productWorkspace.plugins.fieldBaseUrl")}<input value={newConnectorBaseURL} onChange={(event) => setNewConnectorBaseURL(event.target.value)} placeholder="https://api.example.com" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300">{t("productWorkspace.plugins.fieldTokenEnv")}<input value={newConnectorTokenEnv} onChange={(event) => setNewConnectorTokenEnv(event.target.value)} placeholder="OLLAMA_GITHUB_TOKEN" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300">{t("productWorkspace.plugins.fieldOAuthProvider")}<input value={newConnectorOAuthProvider} onChange={(event) => setNewConnectorOAuthProvider(event.target.value)} placeholder="github" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300">{t("productWorkspace.plugins.fieldMethods")}<input value={newConnectorMethods} onChange={(event) => setNewConnectorMethods(event.target.value)} placeholder="GET,POST" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label><label className="text-xs text-neutral-600 dark:text-neutral-300">{t("productWorkspace.plugins.fieldPathPrefix")}<input value={newConnectorPathPrefix} onChange={(event) => setNewConnectorPathPrefix(event.target.value)} placeholder="/v1" className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /></label></div>
				<button type="button" onClick={() => void createNewConnector()} disabled={!newConnectorID.trim() || !newConnectorProvider.trim() || !newConnectorBaseURL.trim()} className="mt-4 rounded-xl bg-violet-700 px-4 py-2.5 text-xs font-medium text-white disabled:cursor-not-allowed disabled:opacity-40">{t("productWorkspace.plugins.registerTenant")}</button>
				</section><section className="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><div><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">{t("productWorkspace.plugins.mcpTitle")}</h3><p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">{t("productWorkspace.plugins.mcpDescription")}</p></div><div className="mt-4 grid gap-4 lg:grid-cols-3"><div className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-700"><p className="text-xs font-medium text-neutral-700 dark:text-neutral-200">{t("productWorkspace.plugins.mcpStdio")}</p><input value={newMCPID} onChange={(event) => setNewMCPID(event.target.value)} placeholder={t("common.id")} className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newMCPCommand} onChange={(event) => setNewMCPCommand(event.target.value)} placeholder="/absolute/path/executable" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newMCPMethods} onChange={(event) => setNewMCPMethods(event.target.value)} placeholder="tools/list,tools/call" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><button type="button" onClick={() => void createNewMCP()} disabled={!newMCPID.trim() || !newMCPCommand.trim()} className="mt-3 rounded-lg bg-neutral-900 px-3 py-2 text-[10px] text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900">{t("productWorkspace.plugins.registerLocalMcp")}</button></div><div className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-700"><p className="text-xs font-medium text-neutral-700 dark:text-neutral-200">{t("productWorkspace.plugins.remoteMcp")}</p><input value={newRemoteMCPID} onChange={(event) => setNewRemoteMCPID(event.target.value)} placeholder={t("common.id")} className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newRemoteMCPURL} onChange={(event) => setNewRemoteMCPURL(event.target.value)} placeholder="https://mcp.example.com" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newRemoteMCPTokenEnv} onChange={(event) => setNewRemoteMCPTokenEnv(event.target.value)} placeholder={t("productWorkspace.plugins.tokenEnvOptional")} className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newRemoteMCPMethods} onChange={(event) => setNewRemoteMCPMethods(event.target.value)} placeholder="tools/list" className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><button type="button" onClick={() => void createNewRemoteMCP()} disabled={!newRemoteMCPID.trim() || !newRemoteMCPURL.trim()} className="mt-3 rounded-lg bg-neutral-900 px-3 py-2 text-[10px] text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900">{t("productWorkspace.plugins.registerRemoteMcp")}</button></div><div className="rounded-xl border border-neutral-200 p-3 dark:border-neutral-700"><p className="text-xs font-medium text-neutral-700 dark:text-neutral-200">{t("productWorkspace.plugins.skillManifest")}</p><input value={newSkillID} onChange={(event) => setNewSkillID(event.target.value)} placeholder={t("common.id")} className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newSkillVersion} onChange={(event) => setNewSkillVersion(event.target.value)} placeholder={t("common.version")} className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newSkillDescription} onChange={(event) => setNewSkillDescription(event.target.value)} placeholder={t("common.description")} className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newSkillScopes} onChange={(event) => setNewSkillScopes(event.target.value)} placeholder={t("productWorkspace.plugins.skillScopes")} className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><input value={newSkillTools} onChange={(event) => setNewSkillTools(event.target.value)} placeholder={t("productWorkspace.plugins.skillTools")} className="mt-2 h-9 w-full rounded-lg border border-neutral-300 bg-transparent px-2 text-xs dark:border-neutral-700" /><button type="button" onClick={() => void createNewSkill()} disabled={!newSkillID.trim() || !newSkillVersion.trim()} className="mt-3 rounded-lg bg-neutral-900 px-3 py-2 text-[10px] text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900">{t("productWorkspace.plugins.registerSkill")}</button></div></div></section><section>
			<div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
				<div><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">{t("productWorkspace.plugins.appsTitle")}</h3><p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">{t("productWorkspace.plugins.appsDescription")}</p></div><span className="rounded-full bg-neutral-100 px-2.5 py-1 text-[10px] text-neutral-500 dark:bg-neutral-800 dark:text-neutral-300">{t("productWorkspace.plugins.catalogCount", { count: connectorCatalog.length })}</span>
			</div>
			<div className="mt-4 flex flex-col gap-3 lg:flex-row"><input value={connectorSearch} onChange={(event) => setConnectorSearch(event.target.value)} aria-label={t("productWorkspace.plugins.search")} placeholder={t("productWorkspace.plugins.search")} className="h-10 min-w-0 flex-1 rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900" /><div role="tablist" aria-label={t("productWorkspace.plugins.categories")} className="flex gap-2 overflow-x-auto pb-1">{connectorCategories.map((category) => <button key={category} type="button" role="tab" aria-selected={connectorCategory === category} onClick={() => setConnectorCategory(category)} className={`whitespace-nowrap rounded-xl px-3 py-2 text-xs ${connectorCategory === category ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "border border-neutral-200 text-neutral-600 dark:border-neutral-700 dark:text-neutral-300"}`}>{category === "Todos" ? t("productWorkspace.plugins.allCategories") : category}</button>)}</div></div>
				<div className="mt-4 grid gap-3 md:grid-cols-2">{filteredConnectorCatalog.map((entry) => { const configuredConnector = connectors.find((connector) => connector.id === entry.id || connector.provider.toLocaleLowerCase() === entry.name.toLocaleLowerCase()); const configured = Boolean(configuredConnector); const statusLabel = !configured ? (entry.status === "available" ? t("productWorkspace.plugins.statusAvailable") : entry.status === "provider_selection_required" ? t("productWorkspace.plugins.statusProviderSelection") : t("productWorkspace.plugins.statusOperatorConfig")) : configuredConnector?.disabled ? t("productWorkspace.plugins.statusConfiguredDisabled") : configuredConnector?.credential_configured ? t("productWorkspace.plugins.statusConfiguredCredential") : t("productWorkspace.plugins.statusConfiguredNoCredential"); const actionLabel = configured ? t("productWorkspace.plugins.viewState") : t("productWorkspace.plugins.configureProviders"); return <article key={entry.id} className="rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><div className="flex items-start gap-3"><div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-neutral-100 text-sm font-semibold text-neutral-700 dark:bg-neutral-800 dark:text-neutral-200">{entry.name.slice(0, 2).toUpperCase()}</div><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><h4 className="font-medium text-neutral-900 dark:text-white">{entry.name}</h4><span className="rounded-full bg-violet-50 px-2 py-1 text-[10px] text-violet-700 dark:bg-violet-950/30 dark:text-violet-300">{entry.category}</span></div><p className="mt-1 text-xs leading-5 text-neutral-500 dark:text-neutral-400">{entry.description}</p><p className="mt-2 text-[10px] text-neutral-400">{statusLabel} {t("productWorkspace.plugins.auth")} {entry.auth}</p></div><button type="button" onClick={() => { if (configured) { setNotice(t("productWorkspace.plugins.configuredNotice", { name: entry.name })); } else { void navigate({ to: "/providers" }); } }} className="shrink-0 rounded-xl border border-neutral-200 px-3 py-2 text-[10px] font-medium text-neutral-700 dark:border-neutral-700 dark:text-neutral-200"><PlusIcon className="mr-1 inline h-3.5 w-3.5" />{actionLabel}</button></div></article>; })}</div>
					{!filteredConnectorCatalog.length && <EmptyState message={t("productWorkspace.plugins.noConnectorMatch")} />}</section><section><h3 className="mb-3 text-xs font-semibold uppercase tracking-[0.12em] text-neutral-500">{t("productWorkspace.plugins.configuredConnectors", { count: connectors.length })}</h3><div className="space-y-3">{connectors.length ? connectors.map((connector) => <ResourceRow key={connector.id} onDelete={() => void removePlugin("connector", connector.id)}><div className="flex items-start justify-between gap-3"><div><div className="flex flex-wrap items-center gap-2"><p className="font-medium text-neutral-900 dark:text-white">{connector.provider} <span className="text-xs text-neutral-400">· {connector.id}</span></p><span className={`rounded-full px-2 py-1 text-[10px] ${connector.disabled ? "bg-neutral-100 text-neutral-500" : connector.credential_configured ? "bg-emerald-50 text-emerald-700" : "bg-amber-50 text-amber-700"}`}>{connector.disabled ? t("productWorkspace.plugins.disabled") : connector.credential_configured ? t("productWorkspace.plugins.connectorEnabledCredential") : t("productWorkspace.plugins.connectorEnabledNoCredential")}</span></div><p className="mt-1 truncate text-xs text-neutral-500">{connector.base_url}</p><p className="mt-1 text-[10px] text-neutral-400">{t("productWorkspace.plugins.connectorOperations", { count: connector.operations?.length ?? 0 })}</p></div><button type="button" onClick={() => void togglePlugin("connector", connector.id, Boolean(connector.disabled))} className="rounded-lg border border-neutral-200 px-2 py-1 text-[10px] dark:border-neutral-700">{connector.disabled ? t("common.enable") : t("common.disable")}</button></div></ResourceRow>) : <EmptyState message={t("productWorkspace.plugins.noConnectors")} />}</div></section><section><h3 className="mb-3 text-xs font-semibold uppercase tracking-[0.12em] text-neutral-500">{t("productWorkspace.plugins.mcpServers", { count: mcpServers.length })}</h3><div className="space-y-3">{mcpServers.length ? mcpServers.map((server) => <ResourceRow key={`${server.transport ?? "stdio"}-${server.id}`} onDelete={() => void removePlugin(server.transport === "streamable-http" ? "remote-mcp" : "mcp", server.id)}><div className="flex items-start justify-between gap-3"><div><p className="font-medium text-neutral-900 dark:text-white">{server.id} <span className="ml-2 rounded-full bg-violet-50 px-2 py-1 text-[10px] text-violet-700 dark:bg-violet-950/30 dark:text-violet-300">{server.transport ?? "stdio"}</span></p><p className="mt-1 truncate font-mono text-xs text-neutral-500">{server.command ?? server.url ?? t("productWorkspace.plugins.remoteEndpoint")} {server.command ? (server.args ?? []).join(" ") : ""}</p><p className="mt-1 text-[10px] text-neutral-400">{t("productWorkspace.plugins.mcpMethods", { count: server.allowed_methods?.length ?? 0 })}</p></div><button type="button" onClick={() => void togglePlugin(server.transport === "streamable-http" ? "remote-mcp" : "mcp", server.id, Boolean(server.disabled))} className="rounded-lg border border-neutral-200 px-2 py-1 text-[10px] dark:border-neutral-700">{server.disabled ? t("common.enable") : t("common.disable")}</button></div></ResourceRow>) : <EmptyState message={t("productWorkspace.plugins.noMcpServers")} />}</div></section><p className="text-xs text-neutral-500">{t("productWorkspace.plugins.cliNote", { count: cliCount })}</p></div>;
  };

  return <SidebarLayout title={copy.title} sidebar={<AppSidebar current={current} />}><div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-900"><div className="mx-auto w-full max-w-6xl px-6 pb-14 pt-10 lg:px-12"><div className="flex flex-col gap-5 border-b border-neutral-200 pb-8 dark:border-neutral-800 md:flex-row md:items-end md:justify-between"><div className="max-w-2xl"><div className="mb-3 flex items-center gap-2 text-xs font-medium text-violet-600 dark:text-violet-300"><SparklesIcon className="h-4 w-4" />{copy.eyebrow}</div><h2 className="font-rounded text-3xl font-semibold tracking-tight text-neutral-950 dark:text-white">{copy.title}</h2><p className="mt-3 text-sm leading-6 text-neutral-500 dark:text-neutral-400">{copy.description}</p></div><button onClick={handleAction} className="inline-flex items-center justify-center gap-2 rounded-xl bg-neutral-950 px-4 py-2.5 text-sm font-medium text-white shadow-sm dark:bg-white dark:text-neutral-950"><PlusIcon className="h-4 w-4" />{copy.action}</button></div>{notice && <div role="status" aria-live="polite" className="mt-5 rounded-xl border border-violet-200 bg-violet-50 px-4 py-3 text-xs leading-5 text-violet-800 dark:border-violet-900/60 dark:bg-violet-950/20 dark:text-violet-200">{notice}</div>}

{kind === "projects" && <section className="mt-6 grid gap-3 rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900 md:grid-cols-[1fr_1fr_auto]"><input id="new-project-name" aria-label={t("productWorkspace.projects.nameLabel")} value={projectName} onChange={(event) => setProjectName(event.target.value)} placeholder={t("productWorkspace.projects.namePlaceholder")} className="h-10 rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700" /><input aria-label={t("productWorkspace.projects.workspaceLabel")} value={projectRoot} onChange={(event) => setProjectRoot(event.target.value)} placeholder={t("productWorkspace.projects.workspacePlaceholder")} className="h-10 rounded-xl border border-neutral-300 bg-transparent px-3 text-sm dark:border-neutral-700" /><button type="button" onClick={() => void createNewProject()} disabled={!projectName.trim() || Boolean(pendingActions["create-project"])} aria-busy={Boolean(pendingActions["create-project"])} className="h-10 rounded-xl bg-neutral-900 px-4 text-sm text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900">{pendingActions["create-project"] ? t("common.creating") : t("common.create")}</button></section>}


{kind === "projects" && <section className="mt-4 rounded-2xl border border-violet-200 bg-violet-50/60 p-4 dark:border-violet-900/60 dark:bg-violet-950/20"><div className="flex flex-wrap items-center justify-between gap-3"><div><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">{t("productWorkspace.projects.importTitle")}</h3><p className="mt-1 text-xs text-neutral-600 dark:text-neutral-300">{t("productWorkspace.projects.importDescription")}</p></div><button type="button" onClick={() => { setImportOpen(true);  }} className="rounded-xl bg-violet-700 px-4 py-2.5 text-xs font-medium text-white">{t("productWorkspace.projects.importAction")}</button></div></section>}

{kind === "projects" && projects.length > 0 && <section className="mt-4 rounded-2xl border border-neutral-200/80 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">{t("productWorkspace.projects.askTitle")}</h3><p className="mt-1 text-xs text-neutral-600 dark:text-neutral-300">{t("productWorkspace.projects.askDescription")}</p><div className="mt-3 flex flex-col gap-3">{projects.length > 1 && <select aria-label={t("productWorkspace.projects.askSelect")} value={askProjectId || projects[0].id} onChange={(event) => setAskProjectId(event.target.value)} className="min-h-10 rounded-lg border border-neutral-300 bg-white px-3 text-sm text-neutral-900 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100">{projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}</select>}<AskDocumentsPanel projectId={askProjectId || projects[0].id} /></div></section>}<ImportProjectDialog open={importOpen && kind === "projects"} onClose={() => setImportOpen(false)} onImported={() => void refresh()} /><div className="mt-7 grid gap-3 sm:grid-cols-3"><StatCard label={t("productWorkspace.stats.status")} value={kind === "tasks" ? t("productWorkspace.stats.running", { count: runningMissions }) : kind === "projects" ? t("productWorkspace.stats.projects", { count: projects.length }) : t("productWorkspace.stats.realCatalog")} tone="green" /><StatCard label={t("productWorkspace.stats.security")} value={t("productWorkspace.stats.activeApprovals")} tone="violet" /><StatCard label={t("productWorkspace.stats.persistence")} value={t("productWorkspace.stats.localFirst")} /></div><div className="mt-7 grid gap-5 lg:grid-cols-[1.45fr_0.8fr]"><section className="rounded-2xl border border-neutral-200/80 bg-neutral-50/50 p-6 dark:border-neutral-800 dark:bg-neutral-950/30"><div className="flex items-center justify-between"><div><h3 className="text-sm font-semibold text-neutral-900 dark:text-white">{t("productWorkspace.footer.persisted")}</h3><p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">{t("productWorkspace.footer.persistedDescription")}</p></div><CheckCircleIcon className="h-5 w-5 text-emerald-500" /></div><div className="mt-5">{renderContent()}</div></section><aside className="space-y-5"><section className="rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900"><div className="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><LockClosedIcon className="h-4 w-4 text-emerald-500" />{t("productWorkspace.footer.policy")}</div><ul className="mt-4 space-y-3 text-xs leading-5 text-neutral-500 dark:text-neutral-400"><li className="flex gap-2"><CheckCircleIcon className="mt-0.5 h-4 w-4 shrink-0 text-emerald-500" />{t("productWorkspace.footer.secrets")}</li><li className="flex gap-2"><CheckCircleIcon className="mt-0.5 h-4 w-4 shrink-0 text-emerald-500" />{t("productWorkspace.footer.approvals")}</li><li className="flex gap-2"><CheckCircleIcon className="mt-0.5 h-4 w-4 shrink-0 text-emerald-500" />{t("productWorkspace.footer.crossTenant")}</li></ul></section><section className="rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900"><div className="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white"><ClockIcon className="h-4 w-4 text-violet-500" />{t("productWorkspace.footer.nextSteps")}</div><p className="mt-3 text-xs leading-5 text-neutral-500 dark:text-neutral-400">{t("productWorkspace.footer.nextStepsDescription")}</p><Link to="/agentic" className="mt-4 inline-flex items-center gap-2 text-xs font-medium text-violet-600 dark:text-violet-300">{t("productWorkspace.footer.openConsole")} <ArrowRightIcon className="h-3.5 w-3.5" /></Link></section></aside></div></div></div></SidebarLayout>;
}
