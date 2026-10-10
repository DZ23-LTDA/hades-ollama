import { API_BASE } from "@/lib/config";

export type AgentProject = {
	id: string;
	name: string;
	root?: string;
	organization_id?: string;
	created_at: string;
	updated_at: string;
};

export type ProjectImportResult = {
  project: AgentProject;
  source: "github" | "zip" | "attachments";
	repository_url?: string;
	ref?: string;
	worktree_path: string;
	branch: string;
	indexed_files: number;
	indexed_memories: number;
	ignored_files: number;
	files: ProjectImportFile[];
	archive_sha256: string;
	state: string;
	notice: string;
};

export type ProjectImportFile = {
	path: string;
	indexed: boolean;
	reason?: string;
	size_bytes: number;
};

/** One persisted step of a schedule flow, as the backend stores it. */
export type AgentScheduleStep = {
  id: string;
  kind: "trigger.interval" | "trigger.webhook" | "action.mission" | "condition.if";
  depends_on?: string[];
  objective?: string;
  expect?: "succeeded" | "failed";
};

export type AgentSchedule = {
  id: string;
  objective: string;
  model?: string;
  workspace?: string;
  project_id?: string;
  organization_id?: string;
  interval_seconds: number;
  enabled: boolean;
  webhook_secret_env?: string;
  /** Persisted flow graph; absent on legacy single-objective schedules. */
  steps?: AgentScheduleStep[];
  next_run_at: string;
  last_run_at?: string;
};

export type AgentArtifact = { id: string; name: string; sha256: string; size: number; media_type?: string };
export type AgentMission = {
  id: string;
  version: number;
  objective: string;
  provider?: string;
  model?: string;
  workspace?: string;
  project_id?: string;
  organization_id?: string;
  capabilities?: string[];
  workspace_isolated?: boolean;
  workspace_snapshot_id?: string;
  workspace_snapshot_sha256?: string;
  state: string;
  plan?: Array<{
    id: string;
    title: string;
    kind: string;
    state: string;
    requires_approval: boolean;
  }>;
  approvals?: Array<{
    id: string;
    step_id: string;
    status: string;
    policy?: string;
    nonce?: string;
    reason?: string;
  }>;
  artifacts?: AgentArtifact[];
  last_error?: string;
  created_at: string;
  updated_at: string;
};
export type AgentEvent = {
  id: string;
  type: string;
  step_id?: string;
  created_at: string;
  payload?: unknown;
};
export type AgentConnector = {
  organization_id?: string;
  id: string;
  provider: string;
  base_url: string;
  token_env?: string;
  oauth_provider?: string;
  allowed_origins?: string[];
  operations?: Array<{
    name: string;
    methods: string[];
    path_prefixes: string[];
  }>;
  disabled?: boolean;
  credential_configured?: boolean;
};
export type AgentConnectorCatalogEntry = {
  id: string;
  name: string;
  category: string;
  kind: string;
  description: string;
  auth: string;
  source: string;
  status: string;
  scopes?: string[];
  api_base_url?: string;
  api_auth_header?: string;
  api_auth_scheme?: string;
  api_self_hosted?: boolean;
  quick_connect?: boolean;
};
export type AgentMCPServer = {
  id: string;
  organization_id?: string;
  command?: string;
  url?: string;
  token_env?: string;
  headers_env?: Record<string, string>;
  transport?: string;
  args?: string[];
  allowed_methods?: string[];
  environment_vars?: string[];
  timeout_seconds?: number;
  disabled?: boolean;
};
export type AgentSkill = {
  id: string;
  organization_id?: string;
  version: string;
  description: string;
  scopes?: string[];
  tools?: string[];
  trusted: boolean;
  enabled: boolean;
};
export type AgentCLIStatus = {
  id: string;
  name: string;
  section: string;
  visibility: string;
  mode: string;
  executables?: string[];
  installed: boolean;
  executable?: string;
};

export type CompanyDepartment = { id: string; name: string; mandate: string; autonomy: string; approval_required?: string[] };
export type CompanyRoadmapItem = { id: string; title: string; description?: string; owner_department?: string; priority: number; status: string; due_at?: string };
export type CompanyGoal = { id: string; title: string; metric: string; target: number; current: number; unit?: string; period: string; owner_department?: string; status: string };
export type CompanyBacklogItem = { id: string; title: string; description?: string; owner_department?: string; priority: number; status: string; source?: string; estimated_hours?: number };
export type CompanyCycle = { id: string; name: string; objective: string; frequency: string; interval_seconds: number; schedule_id?: string; enabled: boolean; next_run_at: string; paused_reason?: string };
export type CompanyAgent = { id: string; department_id: string; name: string; objective: string; budget_cents: number; spent_cents: number; allowed_tools?: string[]; memory_scope: string; metrics?: Record<string, number>; sla?: string; supervisor_id?: string; autonomy: string; pause_condition?: string; approval_required?: string[]; status: string; paused_reason?: string };
export type CompanySocialAccount = { id: string; provider: string; name: string; status: string; oauth_required: boolean };
export type CompanySocialDraft = { id: string; provider: string; account_id?: string; title: string; body: string; status: string; approval_required: boolean; approved: boolean; mode: string; published_at?: string };
export type CompanyApproval = { id: string; company_id: string; organization_id: string; resource_type: string; resource_id: string; policy: string; category?: string; amount_cents?: number; nonce: string; actor_id?: string; status: string; reason?: string; expires_at?: string };
export type TelAgentExchange = { id: string; channel: string; actor_id: string; organization_id: string; message: string; operation: string; status: string; reply: string; created_resource_id?: string; approval_required: boolean; created_at: string };
export type TelAgentResult = { company: AgentCompany; exchange: TelAgentExchange; report?: AgentCompanyReport | null };
export type CompanySocialReport = { company: AgentCompany; connected_accounts: number; pending_oauth: number; drafts: number; approved_drafts: number; published_sandbox: number; impressions: number; clicks: number; conversions: number };
export type AgentCompany = {
  id: string; version: number; organization_id: string; name: string; mission?: string; positioning?: string; business_model?: string;
  target_audience?: string; offer?: string; website?: string; currency: string; status: string;
  departments: CompanyDepartment[]; agents?: CompanyAgent[]; roadmap?: CompanyRoadmapItem[]; goals?: CompanyGoal[]; backlog?: CompanyBacklogItem[]; cycles?: CompanyCycle[]; campaigns?: CompanyCampaign[]; affiliate_programs?: CompanyAffiliateProgram[]; affiliate_links?: CompanyAffiliateLink[]; products?: CompanyProduct[]; orders?: CompanyOrder[]; social_accounts?: CompanySocialAccount[]; social_drafts?: CompanySocialDraft[]; social_metrics?: Array<{ id: string; draft_id: string; provider: string; impressions: number; clicks: number; conversions: number }>;
  tel_agent_history?: TelAgentExchange[]; budget: { currency: string; monthly_limit_cents: number; spent_cents: number; approval_threshold_cents: number; require_approval_for_ads: boolean; require_approval_for_sales: boolean };
  risk: { paused: boolean; pause_reason?: string; anomaly_count: number; last_anomaly?: string }; approvals?: CompanyApproval[];
  created_at: string; updated_at: string;
};
export type AgentCompanyReport = { company: AgentCompany; open_backlog: number; completed_backlog: number; goals_on_track: number; goals_at_risk: number; enabled_cycles: number; budget_utilization_pct: number };
export type CompanyCampaign = { id: string; name: string; channel: string; objective: string; status: string; daily_budget_cents: number; approval_required: boolean; approved: boolean; conversions: number; spend_cents: number };
export type CompanyAffiliateProgram = { id: string; name: string; network: string; status: string; commission_bps: number; approval_required: boolean; approved: boolean };
export type CompanyAffiliateLink = { id: string; program_id: string; product_id?: string; destination: string; conversions: number; revenue_cents: number };
export type CompanyProduct = { id: string; sku: string; name: string; supplier: string; cost_cents: number; price_cents: number; inventory: number; status: string };
export type CompanyOrder = { id: string; product_id: string; customer_ref: string; quantity: number; total_cents: number; status: string; approved: boolean; tracking_code?: string };
export type CompanyGrowthReport = { company: AgentCompany; campaigns_total: number; campaigns_active: number; affiliate_programs: number; affiliate_conversions: number; products: number; pending_orders: number; fulfilled_orders: number; revenue_cents: number };
export type GrokStatus = { provider: string; model: string; state: string; authenticated: boolean; healthy: boolean; last_latency_ms?: number; last_error?: string; checked_at: string };
export type AgentHealth = {
  status: "ok" | "degraded";
  runtime: string;
  checked_at: string;
  subsystems: Record<string, { status: string; detail: string }>;
};

type AgentSession = { token: string; organization?: string };
let agentSession: AgentSession | null = null;

export function setAgentSession(token: string, organization?: string): void {
  const normalizedToken = token.trim();
  if (!normalizedToken) throw new Error("agent session token is required");
  agentSession = { token: normalizedToken, ...(organization?.trim() ? { organization: organization.trim() } : {}) };
}

export function clearAgentSession(): void {
  agentSession = null;
  if (typeof window !== "undefined" && typeof window.dispatchEvent === "function") {
    window.dispatchEvent(new CustomEvent("ollama-agent-session-cleared"));
  }
}

export function hasAgentSession(): boolean {
  return agentSession !== null;
}

export async function logoutAgentSession(): Promise<void> {
  try {
    await agentFetch("/api/agent/v1/auth/logout", { method: "POST", body: "{}" });
  } finally {
    clearAgentSession();
  }
}

export const refreshOAuthCredential = (provider: string, credentialID: string) => agentFetch<{ credential_id: string; provider: string; expires_at?: string; updated_at: string; revoked_at?: string | null }>(`/api/agent/v1/auth/oauth/${encodeURIComponent(provider)}/refresh`, { method: "POST", body: JSON.stringify({ credential_id: credentialID }) });
export const revokeOAuthCredential = (provider: string, credentialID: string) => agentFetch<void>(`/api/agent/v1/auth/oauth/${encodeURIComponent(provider)}/revoke`, { method: "POST", body: JSON.stringify({ credential_id: credentialID }) });

export type OAuthClientStatus = { provider: string; configured: boolean; built_in: boolean };
// saveOAuthClient stores the OAuth app client_id/secret for a provider (admin).
export const saveOAuthClient = (provider: string, clientID: string, clientSecret: string) =>
  agentFetch<{ status: string; provider: string; configured: boolean }>(
    `/api/agent/v1/oauth-clients/${encodeURIComponent(provider)}`,
    { method: "POST", body: JSON.stringify({ client_id: clientID, client_secret: clientSecret }) },
  );
// listOAuthClients lists which OAuth providers have an app credential (no secrets).
export const listOAuthClients = async () => {
  const result = await agentFetch<{ providers?: OAuthClientStatus[] | null }>("/api/agent/v1/oauth-clients");
  return { providers: Array.isArray(result.providers) ? result.providers : [] };
};
// startConnectorOAuth begins an OAuth connect for a catalog connector and
// returns the provider authorization URL to open in the browser.
export const startConnectorOAuth = (connectorID: string, redirectURI: string, codeVerifier: string) =>
  agentFetch<{ connector: string; provider: string; authorization_url: string; state: string; expires_in: number }>(
    `/api/agent/v1/connectors/${encodeURIComponent(connectorID)}/oauth/start`,
    { method: "POST", body: JSON.stringify({ redirect_uri: redirectURI, code_verifier: codeVerifier }) },
  );
export const fetchAgentHealth = () => agentFetch<AgentHealth>("/api/agent/v1/health");

function agentHeaders(): Record<string, string> {
  if (!agentSession) return {};
  return { Authorization: `Bearer ${agentSession.token}`, ...(agentSession.organization ? { "X-Ollama-Organization": agentSession.organization } : {}) };
}
export function agentSessionHeaders(): Record<string, string> {
  return agentHeaders();
}
export const AGENT_LOGIN_REQUIRED_MESSAGE =
  'O Ollama está exposto na rede, então os recursos de agente exigem login. Entre em Configurações → Workspace ou desative "Expose Ollama to the network" em Configurações.';
export async function agentFetch<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const multipart = typeof FormData !== "undefined" && init.body instanceof FormData;
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      ...(multipart ? {} : { "Content-Type": "application/json" }),
      ...agentHeaders(),
      ...(init.headers ?? {}),
    },
  });
  const contentType = response.headers.get("content-type") ?? "";
  if (
    response.ok &&
    response.status !== 204 &&
    contentType.includes("text/html")
  ) {
    // An API route falling through to the SPA must never be parsed as empty JSON.
    throw new Error("Agent API indisponível: resposta inesperada do servidor");
  }
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    if (response.status === 401) clearAgentSession();
    if (response.status === 401 && body?.error === "bearer token is required") {
      throw new Error(AGENT_LOGIN_REQUIRED_MESSAGE);
    }
    throw new Error(typeof body?.error === "string" ? body.error : response.statusText || "Agent API request failed");
  }
  return body as T;
}

export async function agentFetchBlob(path: string, init: RequestInit = {}): Promise<Blob> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      ...agentHeaders(),
      ...(init.headers ?? {}),
    },
  });
  if (!response.ok) {
    if (response.status === 401) clearAgentSession();
    const body = await response.json().catch(() => ({}));
    if (response.status === 401 && body?.error === "bearer token is required") {
      throw new Error(AGENT_LOGIN_REQUIRED_MESSAGE);
    }
    throw new Error(typeof body?.error === "string" ? body.error : response.statusText || "Agent API request failed");
  }
  return response.blob();
}
export const listProjects = () => agentFetch<{ projects: AgentProject[] }>("/api/agent/v1/projects");
export const createProject = (name: string, root = "") => agentFetch<AgentProject>("/api/agent/v1/projects", { method: "POST", body: JSON.stringify({ name, root }) });
export const importGitHubProject = (payload: { url: string; ref?: string; name?: string }) => agentFetch<ProjectImportResult>("/api/agent/v1/projects/import/github", { method: "POST", body: JSON.stringify(payload) });
export const importZIPProject = (payload: { project_id: string; upload_id: string; name?: string }) => agentFetch<ProjectImportResult>("/api/agent/v1/projects/import/zip", { method: "POST", body: JSON.stringify(payload) });
export async function importMissionAttachments(files: Array<{ filename: string; data: Uint8Array; type?: string }>, name: string) {
  const body = new FormData();
  body.append("name", name);
  for (const file of files) {
    const bytes = file.data.slice();
    const blob = new Blob([bytes.buffer as ArrayBuffer], { type: file.type || "application/octet-stream" });
    body.append("files", blob, file.filename);
  }
  return agentFetch<ProjectImportResult>("/api/agent/v1/projects/import/attachments", { method: "POST", body });
}
export const startProjectUpload = (payload: { project_id: string; filename: string; total_size: number; chunk_size: number; sha256?: string }) => agentFetch<{ id: string } & Record<string, unknown>>("/api/agent/v1/uploads", { method: "POST", body: JSON.stringify(payload) });
export const uploadProjectChunk = (uploadID: string, offset: number, data: ArrayBuffer) => agentFetch<Record<string, unknown>>(`/api/agent/v1/uploads/${encodeURIComponent(uploadID)}/chunk?offset=${offset}`, { method: "PUT", headers: { "Content-Type": "application/zip" }, body: data });
export const finalizeProjectUpload = (uploadID: string) => agentFetch<Record<string, unknown>>(`/api/agent/v1/uploads/${encodeURIComponent(uploadID)}/finalize`, { method: "POST", body: "{}" });
export const getProjectUpload = (uploadID: string) => agentFetch<{ id: string; received_bytes: number; total_size: number; state: string } & Record<string, unknown>>(`/api/agent/v1/uploads/${encodeURIComponent(uploadID)}`);
export const updateProject = (id: string, name: string, root = "") => agentFetch<AgentProject>(`/api/agent/v1/projects/${encodeURIComponent(id)}`, { method: "PATCH", body: JSON.stringify({ name, root }) });
export const deleteProject = (id: string) => agentFetch<void>(`/api/agent/v1/projects/${encodeURIComponent(id)}`, { method: "DELETE" });

export interface AgentCitation {
  index: number;
  source: string;
  score: number;
  snippet?: string;
}

export interface AskDocumentsResult {
  project_id: string;
  query: string;
  citations: AgentCitation[];
  grounded: boolean;
}

// askProjectDocuments retrieves relevant, cited excerpts. It does not invoke a
// language model, so callers must not display the prompt context as an answer.
export const askProjectDocuments = (projectID: string, query: string) =>
  agentFetch<AskDocumentsResult>(`/api/agent/v1/projects/${encodeURIComponent(projectID)}/ask?q=${encodeURIComponent(query)}`);

// fetchDiagnostics returns a secret-safe support snapshot (A7) and
// downloadBackupArchive streams a local-first backup of the data root (A6).
export const fetchDiagnostics = () => agentFetch<Record<string, unknown>>("/api/agent/v1/diagnostics");
export const downloadBackupArchive = () => agentFetchBlob("/api/agent/v1/backup");
export const listMissions = () => agentFetch<{ missions: AgentMission[] }>("/api/agent/v1/missions");
export const getMission = (id: string) => agentFetch<AgentMission>(`/api/agent/v1/missions/${encodeURIComponent(id)}`);
export const listMissionEvents = (id: string) => agentFetch<{ events: AgentEvent[] }>(`/api/agent/v1/missions/${encodeURIComponent(id)}/events`);
export const createMission = (payload: { objective: string; provider?: string; model?: string; project_id?: string; workspace?: string; isolate_workspace?: boolean; capabilities?: string[]; auto_run?: boolean }) => agentFetch<AgentMission>("/api/agent/v1/missions", { method: "POST", body: JSON.stringify(payload) });
export const runMission = (id: string) => agentFetch<AgentMission>(`/api/agent/v1/missions/${encodeURIComponent(id)}/run`, { method: "POST", body: "{}" });
export const deleteMission = (id: string) => agentFetch<{ status: string; id: string }>(`/api/agent/v1/missions/${encodeURIComponent(id)}`, { method: "DELETE" });
export const decideMissionApproval = (missionID: string, approvalID: string, approved: boolean, nonce?: string) => agentFetch<AgentMission>(`/api/agent/v1/missions/${encodeURIComponent(missionID)}/approvals/${encodeURIComponent(approvalID)}`, { method: "POST", body: JSON.stringify({ decision: approved ? "approve" : "reject", nonce, reason: approved ? "Aprovado no Agentic Console" : "Rejeitado no Agentic Console" }) });
export type BrowserEnvironmentStatus = {
  python_path?: string;
  python_ok: boolean;
  playwright_ok: boolean;
  playwright_version?: string;
  chromium_ok: boolean;
  ready: boolean;
  guidance?: string[] | null;
  can_auto_setup: boolean;
};
export type BrowserSetupResult = {
  steps?: { description: string; ok: boolean; output?: string }[] | null;
  status: BrowserEnvironmentStatus;
};
export const getBrowserEnvironment = () => agentFetch<BrowserEnvironmentStatus>("/api/agent/v1/browser/environment");
export const setupBrowserEnvironment = () => agentFetch<BrowserSetupResult>("/api/agent/v1/browser/environment/setup", { method: "POST", body: "{}" });

export const listSchedules = () => agentFetch<{ schedules: AgentSchedule[] }>("/api/agent/v1/schedules");
export const createSchedule = (payload: Partial<AgentSchedule>) => agentFetch<AgentSchedule>("/api/agent/v1/schedules", { method: "POST", body: JSON.stringify(payload) });
export const updateSchedule = (id: string, payload: Partial<AgentSchedule>) => agentFetch<AgentSchedule>(`/api/agent/v1/schedules/${encodeURIComponent(id)}`, { method: "PATCH", body: JSON.stringify(payload) });
export const deleteSchedule = (id: string) => agentFetch<void>(`/api/agent/v1/schedules/${encodeURIComponent(id)}`, { method: "DELETE" });
export const listSkills = async () => { const result = await agentFetch<{ skills?: AgentSkill[] | null }>("/api/agent/v1/skills"); return { skills: Array.isArray(result.skills) ? result.skills : [] }; };
export const listConnectors = async () => { const result = await agentFetch<{ connectors?: AgentConnector[] | null }>("/api/agent/v1/connectors"); return { connectors: Array.isArray(result.connectors) ? result.connectors : [] }; };
export const listConnectorCatalog = async () => { const result = await agentFetch<{ connectors?: AgentConnectorCatalogEntry[] | null }>("/api/agent/v1/connector-catalog"); return { connectors: Array.isArray(result.connectors) ? result.connectors : [] }; };
export const listMCPServers = async () => { const result = await agentFetch<{ servers?: AgentMCPServer[] | null; remote_servers?: AgentMCPServer[] | null }>("/api/agent/v1/mcp"); return { servers: [...(Array.isArray(result.servers) ? result.servers : []), ...(Array.isArray(result.remote_servers) ? result.remote_servers.map((server) => ({ ...server, transport: "streamable-http" })) : [])] }; };
export const listCLIStatus = async () => { const result = await agentFetch<{ tools?: AgentCLIStatus[] | null }>("/api/dz23/cli-catalog"); return { tools: Array.isArray(result.tools) ? result.tools : [] }; };
export const registerConnector = (payload: { id: string; provider: string; base_url: string; token_env?: string; oauth_provider?: string; operations: Array<{ name: string; methods: string[]; path_prefixes: string[] }> }) => agentFetch<{ connectors: AgentConnector[]; mcp: AgentMCPServer[]; remote_mcp: AgentMCPServer[]; skills: AgentSkill[] }>("/api/agent/v1/connectors", { method: "POST", body: JSON.stringify(payload) });
export const registerMCP = (payload: { id: string; command: string; args?: string[]; working_directory?: string; allowed_methods: string[]; environment_vars?: string[]; timeout_seconds?: number }) => agentFetch<{ connectors: AgentConnector[]; mcp: AgentMCPServer[]; remote_mcp: AgentMCPServer[]; skills: AgentSkill[] }>("/api/agent/v1/mcp", { method: "POST", body: JSON.stringify(payload) });
export const registerRemoteMCP = (payload: { id: string; url: string; token_env?: string; headers_env?: Record<string, string>; allowed_methods: string[]; timeout_seconds?: number }) => agentFetch<{ connectors: AgentConnector[]; mcp: AgentMCPServer[]; remote_mcp: AgentMCPServer[]; skills: AgentSkill[] }>("/api/agent/v1/remote-mcp", { method: "POST", body: JSON.stringify(payload) });
export const registerSkill = (payload: { id: string; version: string; description?: string; scopes?: string[]; tools?: string[] }) => agentFetch<{ connectors: AgentConnector[]; mcp: AgentMCPServer[]; remote_mcp: AgentMCPServer[]; skills: AgentSkill[] }>("/api/agent/v1/skills", { method: "POST", body: JSON.stringify(payload) });
export const setConnectorEnabled = (id: string, enabled: boolean) => agentFetch(`/api/agent/v1/connectors/${encodeURIComponent(id)}/${enabled ? "enable" : "disable"}`, { method: "POST", body: "{}" });
export const removeConnector = (id: string) => agentFetch(`/api/agent/v1/connectors/${encodeURIComponent(id)}`, { method: "DELETE" });
export const setMCPEnabled = (id: string, enabled: boolean, remote = false) => agentFetch(`/api/agent/v1/${remote ? "remote-mcp" : "mcp"}/${encodeURIComponent(id)}/${enabled ? "enable" : "disable"}`, { method: "POST", body: "{}" });
export const removeMCP = (id: string, remote = false) => agentFetch(`/api/agent/v1/${remote ? "remote-mcp" : "mcp"}/${encodeURIComponent(id)}`, { method: "DELETE" });
export const setSkillEnabled = (id: string, enabled: boolean) => agentFetch(`/api/agent/v1/skills/${encodeURIComponent(id)}/${enabled ? "enable" : "disable"}`, { method: "POST", body: "{}" });
export const removeSkill = (id: string) => agentFetch(`/api/agent/v1/skills/${encodeURIComponent(id)}`, { method: "DELETE" });

export const listCompanies = () => agentFetch<{ companies: AgentCompany[] }>("/api/agent/v1/companies");
export const createCompany = (payload: Record<string, unknown>) => agentFetch<AgentCompany>("/api/agent/v1/companies", { method: "POST", body: JSON.stringify(payload) });
export const updateCompany = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}`, { method: "PATCH", body: JSON.stringify(payload) });
export const getCompanyReport = (id: string) => agentFetch<AgentCompanyReport>(`/api/agent/v1/companies/${encodeURIComponent(id)}/report`);
export const executeCompanyTelAgent = (id: string, payload: { message: string; operation: "report.read" | "backlog.create" | "campaign.draft"; title?: string; description?: string; priority?: number; daily_budget_cents?: number }, idempotencyKey?: string) => agentFetch<TelAgentResult>(`/api/agent/v1/companies/${encodeURIComponent(id)}/tel-agent`, { method: "POST", headers: idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {}, body: JSON.stringify(payload) });
export const getCompanyTelAgentHistory = (id: string) => agentFetch<{ history: TelAgentExchange[]; channel: string; telephony: string }>(`/api/agent/v1/companies/${encodeURIComponent(id)}/tel-agent/history`);
export const addCompanyRoadmap = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/roadmap`, { method: "POST", body: JSON.stringify(payload) });
export const addCompanyGoal = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/goals`, { method: "POST", body: JSON.stringify(payload) });
export const addCompanyBacklog = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/backlog`, { method: "POST", body: JSON.stringify(payload) });
export const addCompanyCycle = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/cycles`, { method: "POST", body: JSON.stringify(payload) });
export const pauseCompany = (id: string, reason: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/pause`, { method: "POST", body: JSON.stringify({ reason }) });
export const resumeCompany = (id: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/resume`, { method: "POST", body: "{}" });
export const recordCompanyAnomaly = (id: string, severity: string, reason: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/anomalies`, { method: "POST", body: JSON.stringify({ severity, reason }) });
export const recordCompanySpend = (id: string, category: string, amount_cents: number) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/spend`, { method: "POST", body: JSON.stringify({ category, amount_cents }) });
export const decideCompanyApproval = (id: string, approvalID: string, approved: boolean, nonce: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/approvals/${encodeURIComponent(approvalID)}/decide`, { method: "POST", body: JSON.stringify({ decision: approved ? "approve" : "reject", nonce, reason: approved ? "Aprovado no Company OS" : "Rejeitado no Company OS" }) });
export const getCompanyGrowthReport = (id: string) => agentFetch<CompanyGrowthReport>(`/api/agent/v1/companies/${encodeURIComponent(id)}/growth/report`);
export const getGrokStatus = () => agentFetch<GrokStatus>("/api/agent/v1/grok/status");
export const listCompanyAgents = (id: string) => agentFetch<{ agents: CompanyAgent[] }>(`/api/agent/v1/companies/${encodeURIComponent(id)}/agents`);
export const pauseCompanyAgent = (id: string, agentID: string, reason: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/agents/${encodeURIComponent(agentID)}/pause`, { method: "POST", body: JSON.stringify({ reason }) });
export const resumeCompanyAgent = (id: string, agentID: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/agents/${encodeURIComponent(agentID)}/resume`, { method: "POST", body: "{}" });
export const recordCompanyAgentSpend = (id: string, agentID: string, amount_cents: number) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/agents/${encodeURIComponent(agentID)}/spend`, { method: "POST", body: JSON.stringify({ amount_cents }) });
export const getCompanySocialReport = (id: string) => agentFetch<CompanySocialReport>(`/api/agent/v1/companies/${encodeURIComponent(id)}/social/report`);
export const addCompanySocialAccount = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/social/accounts`, { method: "POST", body: JSON.stringify(payload) });
export const createCompanySocialDraft = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/social/drafts`, { method: "POST", body: JSON.stringify(payload) });
export const approveCompanySocialDraft = (id: string, draftID: string, nonce: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/social/drafts/${encodeURIComponent(draftID)}/approve`, { method: "POST", body: JSON.stringify({ decision: "approve", nonce, reason: "Aprovado no Company OS" }) });
export const publishCompanySocialDraft = (id: string, draftID: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/social/drafts/${encodeURIComponent(draftID)}/publish`, { method: "POST", body: "{}" });
export const recordCompanySocialMetric = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/social/metrics`, { method: "POST", body: JSON.stringify(payload) });
export const addCompanyCampaign = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/campaigns`, { method: "POST", body: JSON.stringify(payload) });
export const approveCompanyCampaign = (id: string, campaignID: string, nonce: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/campaigns/${encodeURIComponent(campaignID)}/approve`, { method: "POST", body: JSON.stringify({ decision: "approve", nonce, reason: "Aprovado no Company OS" }) });
export const launchCompanyCampaign = (id: string, campaignID: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/campaigns/${encodeURIComponent(campaignID)}/launch`, { method: "POST", body: "{}" });
export const pauseCompanyCampaign = (id: string, campaignID: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/campaigns/${encodeURIComponent(campaignID)}/pause`, { method: "POST", body: "{}" });
export const addCompanyAffiliateProgram = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/affiliate-programs`, { method: "POST", body: JSON.stringify(payload) });
export const approveCompanyAffiliateProgram = (id: string, programID: string, nonce: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/affiliate-programs/${encodeURIComponent(programID)}/approve`, { method: "POST", body: JSON.stringify({ decision: "approve", nonce, reason: "Aprovado no Company OS" }) });
export const addCompanyAffiliateLink = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/affiliate-links`, { method: "POST", body: JSON.stringify(payload) });
export const recordCompanyAffiliateConversion = (id: string, linkID: string, revenue_cents: number) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/affiliate-links/${encodeURIComponent(linkID)}/conversion`, { method: "POST", body: JSON.stringify({ revenue_cents }) });
export const addCompanyProduct = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/products`, { method: "POST", body: JSON.stringify(payload) });
export const createCompanyOrder = (id: string, payload: Record<string, unknown>) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/orders`, { method: "POST", body: JSON.stringify(payload) });
export const approveCompanyOrder = (id: string, orderID: string, nonce: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/orders/${encodeURIComponent(orderID)}/approve`, { method: "POST", body: JSON.stringify({ decision: "approve", nonce, reason: "Aprovado no Company OS" }) });
export const fulfillCompanyOrder = (id: string, orderID: string, tracking_code: string) => agentFetch<AgentCompany>(`/api/agent/v1/companies/${encodeURIComponent(id)}/orders/${encodeURIComponent(orderID)}/fulfill`, { method: "POST", body: JSON.stringify({ tracking_code }) });

export type WhatsAppContactPolicy = {
  phone_number: string;
  name: string;
  role: "owner" | "operator" | "viewer";
  allowed: boolean;
};

export type WhatsAppStatusSummary = {
  active_backend: "evolution_api" | "cloud_api";
  gate_status: "PASS" | "FAIL" | "NOT_CONFIGURED" | "BLOCKED_EXTERNAL" | "UNKNOWN";
  adapters: Record<string, { status: string; details: string }>;
  allowlist_size: number;
  pending_hitl: number;
  dlq_size: number;
  capabilities: {
    stt: string;
    tts: string;
    vision: string;
  };
};

export const getWhatsAppStatus = () => agentFetch<WhatsAppStatusSummary>("/api/agent/v1/whatsapp/status");
export const getWhatsAppAllowlist = () => agentFetch<{ count: number; items: WhatsAppContactPolicy[] }>("/api/agent/v1/whatsapp/allowlist");
export const setWhatsAppContactPolicy = (policy: WhatsAppContactPolicy) => agentFetch<{ status: string; policy: WhatsAppContactPolicy }>("/api/agent/v1/whatsapp/allowlist", { method: "POST", body: JSON.stringify(policy) });
export const removeWhatsAppContactPolicy = (phone: string) => agentFetch<{ status: string; phone: string }>(`/api/agent/v1/whatsapp/allowlist/${encodeURIComponent(phone)}`, { method: "DELETE" });
export const setWhatsAppActiveBackend = (active_backend: "evolution_api" | "cloud_api") => agentFetch<WhatsAppStatusSummary>("/api/agent/v1/whatsapp/config", { method: "POST", body: JSON.stringify({ active_backend }) });
export const configureWhatsAppEvolution = (evolution: { base_url: string; api_key: string; instance: string; webhook_secret: string }, active_backend?: "evolution_api" | "cloud_api") => agentFetch<WhatsAppStatusSummary>("/api/agent/v1/whatsapp/config", { method: "POST", body: JSON.stringify({ evolution, active_backend: active_backend ?? "evolution_api" }) });
export const getWhatsAppDLQ = () => agentFetch<{ count: number; items: unknown[] }>("/api/agent/v1/whatsapp/dlq");

export type SupervisorStatus = {
  enabled: boolean;
  running: boolean;
  worker_id: string;
  last_tick_at?: string;
  tick_count: number;
  pending_missions_resumed: number;
  schedules_triggered: number;
  company_cycles_advanced: number;
  risk_paused_companies: string[];
  blocked_external_actions: number;
  pending_approvals_count: number;
  last_error?: string;
};

export const getSupervisorStatus = () => agentFetch<SupervisorStatus>("/api/agent/v1/supervisor/status");
export const setSupervisorConfig = (config: { enabled: boolean }) => agentFetch<SupervisorStatus>("/api/agent/v1/supervisor/config", { method: "POST", body: JSON.stringify(config) });
export const triggerSupervisorTick = () => agentFetch<{ result: unknown; status: SupervisorStatus }>("/api/agent/v1/supervisor/tick", { method: "POST", body: "{}" });

export type VisualComponent = {
  id: string;
  type: string;
  props?: Record<string, string>;
  style?: Record<string, string>;
  bindings?: Record<string, string>;
  events?: Record<string, string>;
  children?: VisualComponent[];
  x?: number;
  y?: number;
  width?: number;
  height?: number;
};

export type BuilderProject = {
  id: string;
  organization_id: string;
  name: string;
  kind: "website" | "app" | "game" | "slides" | "dashboard";
  entry: string;
  version: number;
  status: string;
  root: string;
  preview_path?: string;
  published_path?: string;
  export_checksum?: string;
  export_path?: string;
  created_at: string;
  updated_at: string;
  components?: VisualComponent[];
  undo_stack?: VisualComponent[][];
  redo_stack?: VisualComponent[][];
};

export type BuilderSpec = {
  name: string;
  kind: "website" | "app" | "game" | "slides" | "dashboard";
  entry?: string;
  files?: Record<string, string>;
  components?: VisualComponent[];
};

export const listBuilders = () => agentFetch<{ projects: BuilderProject[] }>("/api/agent/v1/builders");
export const getBuilder = (id: string) => agentFetch<BuilderProject>(`/api/agent/v1/builders/${encodeURIComponent(id)}`);
export const createBuilder = (spec: BuilderSpec) => agentFetch<BuilderProject>("/api/agent/v1/builders", { method: "POST", body: JSON.stringify(spec) });
export const updateBuilderVisual = (id: string, components: VisualComponent[], expectedVersion?: number) => agentFetch<BuilderProject>(`/api/agent/v1/builders/${encodeURIComponent(id)}/visual`, { method: "POST", body: JSON.stringify({ components, ...(expectedVersion === undefined ? {} : { expected_version: expectedVersion }) }) });
export const undoBuilder = (id: string, expectedVersion?: number) => agentFetch<BuilderProject>(`/api/agent/v1/builders/${encodeURIComponent(id)}/undo`, { method: "POST", body: JSON.stringify(expectedVersion === undefined ? {} : { expected_version: expectedVersion }) });
export const redoBuilder = (id: string, expectedVersion?: number) => agentFetch<BuilderProject>(`/api/agent/v1/builders/${encodeURIComponent(id)}/redo`, { method: "POST", body: JSON.stringify(expectedVersion === undefined ? {} : { expected_version: expectedVersion }) });
export const previewBuilder = (id: string) => agentFetch<{ project: BuilderProject; artifact: unknown }>(`/api/agent/v1/builders/${encodeURIComponent(id)}/preview`, { method: "POST", body: "{}" });
export const exportBuilder = (id: string) => agentFetch<{ project: BuilderProject; archive_path: string; checksum: string; sha256: string; download_url: string }>(`/api/agent/v1/builders/${encodeURIComponent(id)}/export`, { method: "POST", body: "{}" });
export const deployBuilder = (id: string, provider: string, payload: { target?: string; approval_id?: string; nonce?: string }) => agentFetch<{ status: string; url?: string }>(`/api/agent/v1/builders/${encodeURIComponent(id)}/deploy/${encodeURIComponent(provider)}`, { method: "POST", body: JSON.stringify(payload) });
