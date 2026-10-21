// English catalog for the UI.
// Must stay a superset of the pt-BR keys (see `i18n.test.ts`).
export const enMessages: Record<string, string> = {
  // Common
  "common.delete": "Delete",
  "common.remove": "Remove",
  "common.enable": "Enable",
  "common.disable": "Disable",
  "common.save": "Save",
  "common.edit": "Edit",
  "common.create": "Create",
  "common.creating": "Creating…",
  "common.retry": "Try again",
  "common.id": "ID",
  "common.provider": "Provider",
  "common.version": "Version",
  "common.description": "Description",

  // Page header by kind
  "productWorkspace.projects.title": "Projects",
  "productWorkspace.projects.eyebrow": "Persistent context",
  "productWorkspace.projects.description":
    "Organize memory, sources, tasks, members and builders in isolated workspaces.",
  "productWorkspace.projects.action": "New project",
  "productWorkspace.skills.title": "Skills",
  "productWorkspace.skills.eyebrow": "Skills and MCP",
  "productWorkspace.skills.description":
    "See manifests, versions, scopes and trust of the skills loaded by the runtime.",
  "productWorkspace.skills.action": "Open settings",
  "productWorkspace.plugins.title": "Advanced registry",
  "productWorkspace.plugins.eyebrow": "Connectors, MCP and skills",
  "productWorkspace.plugins.description":
    "Inspect GitHub, Google Workspace, Claude, Codex, OmniRoute, MCP and other services without exposing secrets.",
  "productWorkspace.plugins.action": "Open settings",
  "productWorkspace.tasks.title": "Tasks",
  "productWorkspace.tasks.eyebrow": "Mission inbox",
  "productWorkspace.tasks.description":
    "See missions running, awaiting approval, completed or recovering.",
  "productWorkspace.tasks.action": "New task",

  // Generic states
  "productWorkspace.resource.delete": "Delete resource",
  "productWorkspace.loading": "Querying agentic contracts…",
  "productWorkspace.loadError": "Could not load persisted data.",

  // Projects
  "productWorkspace.projects.updatedAt": "· updated",
  "productWorkspace.projects.localWorkspace": "managed local workspace",
  "productWorkspace.projects.empty":
    "Create a project to keep instructions, files and context across missions.",
  "productWorkspace.projects.emptyAction": "Create project",
  "productWorkspace.projects.created":
    "Project created and persisted in ContextStore.",
  "productWorkspace.projects.createFailed": "Failed to create project.",
  "productWorkspace.projects.updated": "Project updated.",
  "productWorkspace.projects.updateFailed": "Failed to update project.",
  "productWorkspace.projects.deleted": "Project deleted.",
  "productWorkspace.projects.deleteFailed": "Failed to delete project.",
  "productWorkspace.projects.deleteConfirm":
    "Delete project {name}? Associated local memories will also be removed.",
  "productWorkspace.projects.nameLabel": "New project name",
  "productWorkspace.projects.namePlaceholder": "Project name",
  "productWorkspace.projects.workspaceLabel": "New project workspace",
  "productWorkspace.projects.workspacePlaceholder": "Workspace (optional)",
  "productWorkspace.projects.importTitle": "Import entire project",
  "productWorkspace.projects.importDescription":
    "GitHub and ZIP are downloaded, isolated in a worktree and indexed. Imported code is never executed automatically.",
  "productWorkspace.projects.importAction": "Import project",
  "productWorkspace.projects.askTitle": "Ask the documents",
  "productWorkspace.projects.askDescription":
    "Answers come only from the project's indexed documents, with sources cited.",
  "productWorkspace.projects.askSelect": "Project to query",

  // Tasks
  "productWorkspace.tasks.created": "Task created",
  "productWorkspace.tasks.state": "· state",
  "productWorkspace.tasks.missingTitle":
    "The created task did not appear in the list",
  "productWorkspace.tasks.missingHint":
    "Refresh the list or check the active organization.",
  "productWorkspace.tasks.refresh": "Refresh list",
  "productWorkspace.tasks.empty":
    "No task created yet. Create the first one to follow its plan, approvals and artifacts here.",
  "productWorkspace.tasks.emptyAction": "New task",
  "productWorkspace.tasks.listLabel": "Tasks",
  "productWorkspace.tasks.meta":
    "{id} · {steps} steps · {artifacts} artifacts · {model}",
  "productWorkspace.tasks.defaultModel": "default model",
  "productWorkspace.tasks.cancelBeforeDelete":
    "Cancel the task before deleting",
  "productWorkspace.tasks.deleteLabel": "Delete task",
  "productWorkspace.tasks.deleted": "Task deleted.",
  "productWorkspace.tasks.deleteFailed": "Failed to delete the task.",
  "productWorkspace.tasks.deleteConfirm":
    'Delete task "{objective}"? This action cannot be undone.',

  // Skills
  "productWorkspace.skills.catalogTitle": "Catalog by context",
  "productWorkspace.skills.catalogDescription":
    "Activate skills by project, agent or model and see what the server selected for the given budget. Order and exclusions come from the server; the screen does not decide on its own.",
  "productWorkspace.skills.meta": "{count} tools · {trust}",
  "productWorkspace.skills.trusted": "trusted",
  "productWorkspace.skills.needsReview": "needs review",
  "productWorkspace.skills.empty":
    "No skill was loaded by the runtime. Install a reviewed manifest in the server's skills directory.",
  "productWorkspace.skills.emptyAction": "Open settings",
  "productWorkspace.skills.resolveFailed":
    "Could not resolve the skills catalog.",
  "productWorkspace.skills.activateFailed": "Could not activate the skill.",
  "productWorkspace.skills.deactivateFailed": "Could not deactivate the skill.",

  // Plugins / connectors / MCP
  "productWorkspace.plugins.statusError": "Could not read the plugin state.",
  "productWorkspace.plugins.installedTitle": "Installed plugins — real state",
  "productWorkspace.plugins.installedDescription":
    "The server reports each plugin's state from verifiable data. “Ready to use” only appears when a real execution is proven; a catalog is not an integration.",
  "productWorkspace.plugins.connectorTitle": "Register a secure connector",
  "productWorkspace.plugins.connectorDescription":
    "Register endpoint, operations and credential references. The token value is never accepted on this screen; use a server environment variable or the identifier of an already authorized OAuth.",
  "productWorkspace.plugins.fieldBaseUrl": "Base URL HTTPS",
  "productWorkspace.plugins.fieldTokenEnv":
    "Token environment variable name (optional)",
  "productWorkspace.plugins.fieldOAuthProvider": "OAuth provider ID (optional)",
  "productWorkspace.plugins.fieldMethods": "Allowed methods",
  "productWorkspace.plugins.fieldPathPrefix": "Path prefix",
  "productWorkspace.plugins.registerTenant": "Register in current tenant",
  "productWorkspace.plugins.connectorRegistered":
    "Connector registered in the current tenant. The runtime keeps only credential references; the value must exist in the server environment or in the authorized OAuth.",
  "productWorkspace.plugins.connectorRegisterFailed":
    "Failed to register the connector.",
  "productWorkspace.plugins.mcpTitle": "Register MCP and skills",
  "productWorkspace.plugins.mcpDescription":
    "Registration stores manifests only. Local MCP uses a validated absolute executable; Remote MCP uses HTTPS/SSRF/DNS pinning; skills start untrusted. Do not paste tokens on this screen.",
  "productWorkspace.plugins.mcpStdio": "MCP stdio",
  "productWorkspace.plugins.registerLocalMcp": "Register local MCP",
  "productWorkspace.plugins.remoteMcp": "Remote MCP",
  "productWorkspace.plugins.tokenEnvOptional": "TOKEN_ENV (optional)",
  "productWorkspace.plugins.registerRemoteMcp": "Register Remote MCP",
  "productWorkspace.plugins.skillManifest": "Skill manifest",
  "productWorkspace.plugins.skillScopes": "Comma-separated scopes",
  "productWorkspace.plugins.skillTools": "Comma-separated tools",
  "productWorkspace.plugins.registerSkill": "Register skill",
  "productWorkspace.plugins.mcpRegistered":
    "MCP stdio registered. The executable is validated and runs only on the host/workspace allowed by the server.",
  "productWorkspace.plugins.mcpRegisterFailed": "Failed to register MCP stdio.",
  "productWorkspace.plugins.remoteMcpRegistered":
    "Remote MCP registered. The endpoint must pass HTTPS/SSRF/DNS pinning; tokens stay in the server environment.",
  "productWorkspace.plugins.remoteMcpRegisterFailed":
    "Failed to register Remote MCP.",
  "productWorkspace.plugins.skillRegistered":
    "Skill manifest registered as untrusted and enabled by the server for review/approval.",
  "productWorkspace.plugins.skillRegisterFailed":
    "Failed to register the skill.",
  "productWorkspace.plugins.toggleFailed":
    "Failed to change the plugin lifecycle.",
  "productWorkspace.plugins.removeFailed": "Failed to remove the plugin.",
  "productWorkspace.plugins.toggleNotice": "{id} {status}.",
  "productWorkspace.plugins.enabled": "enabled",
  "productWorkspace.plugins.disabled": "disabled",
  "productWorkspace.plugins.removeNotice":
    "{id} removed from the local manifest.",
  "productWorkspace.plugins.removeConfirm":
    "Remove {id} from the durable manifest? This action does not revoke upstream credentials.",
  "productWorkspace.plugins.appsTitle": "Apps and connectors",
  "productWorkspace.plugins.appsDescription":
    "A broad catalog of apps, APIs and MCPs. The configuration button never invents a connection: each provider requires OAuth, an API key or its own approval.",
  "productWorkspace.plugins.catalogCount": "{count} available in the catalog",
  "productWorkspace.plugins.search": "Search connectors",
  "productWorkspace.plugins.categories": "Connector categories",
  "productWorkspace.plugins.allCategories": "All",
  "productWorkspace.plugins.auth": "· auth:",
  "productWorkspace.plugins.statusAvailable": "Available for configuration",
  "productWorkspace.plugins.statusProviderSelection": "Choose the tax provider",
  "productWorkspace.plugins.statusOperatorConfig":
    "Operator configuration required",
  "productWorkspace.plugins.statusConfiguredDisabled": "Configured · disabled",
  "productWorkspace.plugins.statusConfiguredCredential":
    "Configured · enabled · credential present",
  "productWorkspace.plugins.statusConfiguredNoCredential":
    "Configured · enabled · credential missing",
  "productWorkspace.plugins.viewState": "View state",
  "productWorkspace.plugins.configureProviders": "Configure in Providers",
  "productWorkspace.plugins.configuredNotice":
    "{name}: see the state in the configured connectors block below. The runtime does not call the integration without a credential and approval.",
  "productWorkspace.plugins.noConnectorMatch":
    "No connector matches the search or selected category.",
  "productWorkspace.plugins.configuredConnectors":
    "Configured connectors ({count})",
  "productWorkspace.plugins.connectorOperations":
    "{count} operations · secrets kept on the server",
  "productWorkspace.plugins.connectorEnabledCredential":
    "enabled · credential present",
  "productWorkspace.plugins.connectorEnabledNoCredential":
    "enabled · credential missing",
  "productWorkspace.plugins.noConnectors":
    "No connector configured. The catalog above shows what can be connected without pretending authorization.",
  "productWorkspace.plugins.mcpServers": "Local and remote MCP ({count})",
  "productWorkspace.plugins.remoteEndpoint": "remote endpoint",
  "productWorkspace.plugins.mcpMethods":
    "{count} allowlisted methods · credential only on the server",
  "productWorkspace.plugins.noMcpServers":
    "No MCP server loaded. The command or endpoint must be allowlisted on the server.",
  "productWorkspace.plugins.cliNote":
    "{count} integration CLI(s) detected. Installation and registration remain protected in the operator environment.",

  // Indicators and footer
  "productWorkspace.stats.status": "Status",
  "productWorkspace.stats.security": "Security",
  "productWorkspace.stats.persistence": "Persistence",
  "productWorkspace.stats.running": "{count} running",
  "productWorkspace.stats.projects": "{count} projects",
  "productWorkspace.stats.realCatalog": "Real catalog",
  "productWorkspace.stats.activeApprovals": "Active approvals",
  "productWorkspace.stats.localFirst": "Local-first",
  "productWorkspace.footer.persisted": "Persisted data",
  "productWorkspace.footer.persistedDescription":
    "This surface queries the agentic contract, not visual fixtures.",
  "productWorkspace.footer.policy": "Active policy",
  "productWorkspace.footer.secrets": "Secrets are not shown in the interface.",
  "productWorkspace.footer.approvals": "External actions require approval.",
  "productWorkspace.footer.crossTenant":
    "Cross-tenant is rejected on the server.",
  "productWorkspace.footer.nextSteps": "Next steps",
  "productWorkspace.footer.nextStepsDescription":
    "To create a complex mission, use the console with provider, project, timeline and approvals selection.",
  "productWorkspace.footer.openConsole": "Open Agentic Console",
};
