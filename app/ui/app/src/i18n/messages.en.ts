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

  // Studio — component palette and project templates
  "studio.palette.ai.title": "Generate with AI",
  "studio.palette.ai.placeholder": "Describe the app/site you want…",
  "studio.palette.ai.replace": "Generate from scratch (replace the canvas)",
  "studio.palette.ai.generating": "Generating with the local model…",
  "studio.palette.ai.generate": "Generate components",
  "studio.palette.components": "Components",
  "studio.palette.addHint": "Click to add",
  "studio.palette.templates": "Project Templates",
  "studio.palette.mobileAdd": "Add:",
  "studio.palette.template.site": "Site / Landing Page",
  "studio.palette.template.dashboard": "Analytics Dashboard",
  "studio.palette.template.slides": "Slides / Pitch Deck",
  "studio.palette.template.game": "Interactive 2D Game",
  "studio.palette.template.app": "Web Application",

  // Studio — component templates (palette labels)
  "studio.template.heading.label": "Heading",
  "studio.template.paragraph.label": "Paragraph",
  "studio.template.button.label": "Call-to-Action Button",
  "studio.template.card.label": "Card / Feature",
  "studio.template.metric.label": "Metric / KPI",
  "studio.template.navbar.label": "Navigation Bar",
  "studio.template.hero.label": "Hero / Highlight",
  "studio.template.image.label": "Image",
  "studio.template.input.label": "Form Field",
  "studio.template.link.label": "Link",
  "studio.template.list.label": "List",
  "studio.template.divider.label": "Divider",
  "studio.template.pricing.label": "Plan / Price",
  "studio.template.testimonial.label": "Testimonial",
  "studio.template.faq.label": "FAQ (question)",
  "studio.template.footer.label": "Footer",

  // Studio — initial content of the templates
  "studio.template.heading.text": "New High-Impact Heading",
  "studio.template.paragraph.text":
    "Describe your idea, solution or product with clear, objective details.",
  "studio.template.button.text": "Click Here",
  "studio.template.card.title": "Product Highlight",
  "studio.template.card.body":
    "Explanation of the key benefits of the solution you built.",
  "studio.template.metric.sampleLabel": "Success Rate",
  "studio.template.navbar.links": "Home, Features, Pricing, Contact",
  "studio.template.hero.title": "Your idea, ready in minutes",
  "studio.template.hero.subtitle": "Build, publish and evolve — all local.",
  "studio.template.hero.cta": "Start now",
  "studio.template.image.alt": "Image description",
  "studio.template.input.sampleLabel": "Your e-mail",
  "studio.template.input.placeholder": "you@example.com",
  "studio.template.link.text": "Learn more",
  "studio.template.list.items": "First item, Second item, Third item",
  "studio.template.pricing.plan": "Pro Plan",
  "studio.template.pricing.price": "$49/month",
  "studio.template.pricing.features":
    "Everything in Basic, Priority support, No limits",
  "studio.template.pricing.cta": "Subscribe",
  "studio.template.testimonial.quote": "It completely changed our workflow.",
  "studio.template.testimonial.author": "Satisfied customer",
  "studio.template.faq.question": "Frequently asked question?",
  "studio.template.faq.answer": "A clear, objective answer to the question.",
  "studio.template.footer.text": "© 2026 My Company",
  "studio.template.footer.links": "About, Contact, Privacy",

  // Studio — canvas render fallbacks
  "studio.render.paragraphFallback": "Paragraph text",
  "studio.render.buttonFallback": "Button",
  "studio.render.cardTitleFallback": "Card title",
  "studio.render.cardBodyFallback": "Explanatory card body.",
  "studio.render.metricFallback": "Metric",
  "studio.render.navbarBrandFallback": "Brand",
  "studio.render.navbarLinksFallback": "Link 1, Link 2",
  "studio.render.heroTitleFallback": "Hero title",
  "studio.render.heroSubtitleFallback": "Explanatory subtitle",
  "studio.render.heroCtaFallback": "Action",
  "studio.render.imageEmpty": "Image (set the URL in the properties)",
  "studio.render.inputLabelFallback": "Label",
  "studio.render.listFallback": "Item 1, Item 2",
  "studio.render.pricingFeaturesFallback": "Feature 1, Feature 2",
  "studio.render.footerLinksFallback": "About, Contact",

  // Studio — header
  "studio.header.title": "Studio Editor",
  "studio.header.subtitle":
    "Build a site or app by adding components with one click and reordering the list; preview it and export as ZIP.",
  "studio.header.undo": "Undo",
  "studio.header.redo": "Redo",
  "studio.header.tabEditor": "Editor",
  "studio.header.tabPreview": "Preview",
  "studio.header.export": "Export ZIP",
  "studio.header.exportTitle": "Export ZIP with SHA-256 checksum",
  "studio.header.downloadHtml": "Download HTML",
  "studio.header.downloadHtmlTitle":
    "Download the app as a standalone HTML file",
  "studio.header.instantPreview": "View now",
  "studio.header.instantPreviewTitle":
    "View the app now in a new tab (instant preview, no server)",
  "studio.header.publish": "Publish",

  // Studio — canvas
  "studio.canvas.area": "Canvas area ({count} components)",
  "studio.canvas.hint": "Click a component to edit its properties",
  "studio.canvas.loading": "Loading the Studio canvas...",
  "studio.canvas.empty": "The canvas is empty",
  "studio.canvas.emptyHint":
    "Click a component in the left sidebar to start building.",
  "studio.canvas.dragHint": "Drag to reorder",
  "studio.canvas.moveUp": "Move up",
  "studio.canvas.moveDown": "Move down",
  "studio.canvas.delete": "Delete",

  // Studio — properties inspector
  "studio.inspector.title": "Properties",
  "studio.inspector.type": "Type",
  "studio.inspector.text": "Text",
  "studio.inspector.titleField": "Title",
  "studio.inspector.body": "Body",
  "studio.inspector.label": "Label",
  "studio.inspector.value": "Value",
  "studio.inspector.brand": "Brand",
  "studio.inspector.links": "Links (comma-separated)",
  "studio.inspector.change": "Change",
  "studio.inspector.action": "Action",

  // Studio — preview
  "studio.preview.authNote":
    "Authenticated preview: loaded with your session; the internal URL is protected and not shareable.",
  "studio.preview.expand": "Expand preview",
  "studio.preview.frameTitle": "Live preview",
  "studio.preview.dialogLabel": "App preview",
  "studio.preview.dialogTitle": "Preview — {name}",
  "studio.preview.close": "Close ✕",
  "studio.preview.closeLabel": "Close preview",
  "studio.preview.expandedDialogLabel": "Expanded preview",
  "studio.preview.expandedDialogTitle": "Authenticated preview — {name}",
  "studio.preview.expandedFrameTitle": "Expanded authenticated preview",

  // Studio — publishing
  "studio.deploy.title": "Publish project",
  "studio.deploy.description":
    "Select the cloud provider to host your project. Without active credentials, the system keeps an honest status.",
  "studio.deploy.requiresCredentials": "Requires credentials",
  "studio.deploy.close": "Close",
  "studio.deploy.provider.vercel": "Vercel",
  "studio.deploy.provider.cloudflare": "Cloudflare Pages",
  "studio.deploy.provider.netlify": "Netlify",
  "studio.deploy.provider.ssh": "SSH / Self-Hosted Server",
  "studio.deploy.credential.vercel": "VERCEL_TOKEN",
  "studio.deploy.credential.cloudflare":
    "CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID",
  "studio.deploy.credential.netlify": "NETLIFY_AUTH_TOKEN",
  "studio.deploy.credential.ssh":
    "SSH key and host (SSH_PRIVATE_KEY, SSH_HOST, SSH_USER)",
  "studio.deploy.publishing": "Publishing…",
  "studio.deploy.sending":
    "Sending the project to {provider} through the Deploy Adapter…",
  "studio.deploy.published": "Published",
  "studio.deploy.publishedWithUrl": "Published on {provider}. URL: {url}",
  "studio.deploy.processing": "Processing",
  "studio.deploy.accepted":
    "{provider}: publish request accepted by the runtime. Track the completion and the final URL.",
  "studio.deploy.acceptedWithStatus":
    "{provider}: publish request accepted by the runtime (state: {status}). Track the completion and the final URL.",
  "studio.deploy.blocked": "Blocked (external dependency)",
  "studio.deploy.failed":
    "Could not publish to {provider}: {reason}. Configure the credential in the server environment: {credential}. The system does not fake the publication.",

  // Studio — notices
  "studio.notice.loadError": "Error loading project: {error}",
  "studio.notice.previewAuthError":
    "Error loading authenticated preview: {error}",
  "studio.notice.canvasUpdated": "Canvas updated (v{version})",
  "studio.notice.canvasSaveError": "Error saving canvas: {error}",
  "studio.notice.generating": "Generating components with the local model…",
  "studio.notice.noModel":
    "No model available. Download a model in Settings first.",
  "studio.notice.invalidComponents":
    "The AI did not return valid components. Try describing it differently.",
  "studio.notice.generatedFresh":
    "{count} components generated from scratch by the local model ({model}).",
  "studio.notice.generatedAdded":
    "{count} components added by the local model ({model}).",
  "studio.notice.generateError": "Failed to generate with AI: {error}",
  "studio.notice.previewError": "Could not generate the preview: {error}",
  "studio.notice.downloaded": "App HTML downloaded.",
  "studio.notice.downloadError": "Could not download the HTML: {error}",
  "studio.notice.undo": "Undo executed (v{version})",
  "studio.notice.undoError": "Nothing to undo: {error}",
  "studio.notice.redo": "Redo executed (v{version})",
  "studio.notice.redoError": "Nothing to redo: {error}",
  "studio.notice.previewGenerated":
    "Live preview generated on the backend successfully!",
  "studio.notice.previewGenerateError": "Error generating preview: {error}",
  "studio.notice.exported": "Project exported successfully (SHA-256 verified)!",
  "studio.notice.exportError": "Error exporting: {error}",

  // Studio — creating from a template
  "studio.create.confirm":
    'Create "{name}"? The project currently open in the editor will be replaced by a new blank project.',
  "studio.create.confirmLabel": "Create",
  "studio.create.success": 'New project "{name}" created.',
  "studio.create.error": "Error creating project from template: {error}",
  "studio.templateName.site": "New Website",
  "studio.templateName.dashboard": "New Dashboard",
  "studio.templateName.slides": "New Presentation",
  "studio.templateName.game": "New Web Game",
  "studio.templateName.app": "New Mobile App",

  // Studio — starter project
  "studio.seed.hero": "Welcome to the Hades Studio",
  "studio.seed.lead":
    "End-to-end interactive visual editor with live preview, undo/redo and traceable export.",
  "studio.seed.cta": "Try It Now",

  // Automations — header and view switch
  "automations.title": "Automations",
  "automations.description":
    "Automate recurring flows, configure triggers and let the agent work in the background.",
  "automations.new": "New automation",
  "automations.view.list": "List",
  "automations.view.flow": "Visual editor (beta)",

  // Automations — visual flow editor
  "automations.flow.hint":
    "Build the flow by dragging nodes from the palette, connect the steps and validate before publishing. The draft is saved in this browser.",
  "automations.flow.published":
    "Flow published as an automation. See it under Active Routines.",

  // Automations — follow the created mission
  "automations.followMission": "Follow in the Mission Console →",

  // Automations — webhook instructions
  "automations.webhook.title": "Webhook trigger created — how to trigger it",
  "automations.webhook.close": "Close",
  "automations.webhook.intro":
    "This trigger runs only when the endpoint below receives an authenticated request (it does not run on a timer). Send a",
  "automations.webhook.introTail":
    "with the secret header and a unique idempotency key per event.",
  "automations.webhook.endpoint": "Endpoint (POST)",
  "automations.webhook.secretHeader": "Secret header",
  "automations.webhook.idempotency":
    "Idempotency key (required, unique per event)",
  "automations.webhook.secretValue":
    "X-Ollama-Agent-Secret: <value of the environment variable {env}>",
  "automations.webhook.idempotencyExample":
    "Idempotency-Key: <unique event id>",
  "automations.webhook.secretExplain": "The value of",
  "automations.webhook.secretExplainMid":
    "must equal the content of the environment variable",
  "automations.webhook.secretExplainTail":
    "configured on the server. Requests without the idempotency key or with an invalid secret are refused.",

  // Automations — Card: schedule
  "automations.card.schedule.title": "Schedule",
  "automations.card.schedule.description":
    "Automate recurring jobs by choosing when and how often Hades should run.",
  "automations.card.schedule.presetProgress": "⚡ Daily progress report (24h)",
  "automations.card.schedule.presetSecurity": "⚡ Weekly security scan (7d)",
  "automations.card.schedule.action": "Create schedule",
  "automations.card.schedule.objectiveProgress":
    "Audit files and generate a progress report",
  "automations.card.schedule.objectiveSecurity":
    "Check for dependency and security updates",

  // Automations — Card: trigger
  "automations.card.trigger.title": "Trigger",
  "automations.card.trigger.description":
    "Configure an authenticated webhook; the endpoint requires a secret in an environment variable and an idempotency key.",
  "automations.card.trigger.presetTests":
    "⚡ Test the repository when commits are detected",
  "automations.card.trigger.presetInvoices":
    "⚡ Extract data from incoming invoices",
  "automations.card.trigger.action": "Create trigger",
  "automations.card.trigger.objectiveTests":
    "Run tests and validate the build when a new commit is detected",
  "automations.card.trigger.objectiveInvoices":
    "Process files and invoices added to the local folder",

  // Automations — Card: advanced automation
  "automations.card.advanced.title": "Advanced automation",
  "automations.card.advanced.description":
    "Describe in your own words what Hades should do and choose the frequency below. Natural-language time interpretation is not available yet: for now the automation runs at a fixed interval.",
  "automations.card.advanced.placeholder":
    "E.g.: Monitor my repository and generate a summary...",
  "automations.card.advanced.frequency": "Frequency",
  "automations.card.advanced.start": "Start",
  "automations.card.advanced.missingPrompt":
    "Describe what Hades should do before starting.",

  // Automations — intervals
  "automations.interval.hour1": "Every 1 hour",
  "automations.interval.hour6": "Every 6 hours",
  "automations.interval.hour12": "Every 12 hours",
  "automations.interval.day24": "Daily (24 hours)",
  "automations.interval.week7": "Weekly (7 days)",
  "automations.interval.day24Long": "Daily (every 24 hours)",
  "automations.interval.week7Long": "Weekly (every 7 days)",
  "automations.interval.everyHours": "Every {hours} hours",

  // Automations — creation modal
  "automations.modal.title": "Configure New Automation Routine",
  "automations.modal.objective": "Objective / Instruction for the Agent",
  "automations.modal.objectivePlaceholder":
    "E.g.: Compare competitor data and update the report",
  "automations.modal.triggerType": "Trigger type",
  "automations.modal.triggerInterval": "Recurring interval",
  "automations.modal.triggerWebhook": "Webhook/event",
  "automations.modal.secretEnv": "Webhook secret variable",
  "automations.modal.secretEnvPlaceholder": "VARIABLE_NAME",
  "automations.modal.webhookNote":
    "Webhook triggers do not run on a timer: they are fired only by the authenticated endpoint, whose URL and instructions appear after saving.",
  "automations.modal.frequency": "Execution frequency",
  "automations.modal.cancel": "Cancel",
  "automations.modal.saving": "Saving...",
  "automations.modal.save": "Save automation",

  // Automations — routines list
  "automations.list.title": "Active Routines ({count})",
  "automations.list.openConsole": "Open Mission Console →",
  "automations.list.loading": "Loading automations...",
  "automations.list.empty": "No routine configured",
  "automations.list.emptyHint":
    "Use the cards above to create your first schedule or autonomous trigger.",

  // Automations — list item
  "automations.item.webhookOnly":
    "Triggered by webhook (does not run on a timer)",
  "automations.item.nextRun": "{interval} • Next run: {when}",
  "automations.item.runNow": "Run now",
  "automations.item.runNowTitle": "Run immediately",
  "automations.item.reprocess": "Reprocess pending",
  "automations.item.reprocessTitle":
    "Reprocess only the skipped or failed steps",
  "automations.item.approve": "Approve {step}",
  "automations.item.approveTitle": 'Approve "{step}" for this run',
  "automations.item.reject": "Reject",
  "automations.item.rejectTitle": 'Reject "{step}" for this run',
  "automations.item.pause": "Pause",
  "automations.item.resume": "Resume",
  "automations.item.deleteTitle": "Delete routine",

  // Automations — errors
  "automations.error.load": "Could not load the automations.",
  "automations.error.missingObjective":
    "Provide the automation objective/instruction.",
  "automations.error.missingSecret":
    "Provide the environment variable with the webhook secret.",
  "automations.error.create": "Could not create the automation.",
  "automations.error.approval": "Could not record the decision.",
  "automations.error.reprocess": "Could not reprocess the automation.",
  "automations.error.run": "Could not run the automation.",
  "automations.error.toggle": "Could not change the automation state.",
  "automations.error.delete": "Could not delete the automation.",

  // Automations — success
  "automations.success.created": "Automation created successfully!",
  "automations.success.approved":
    'Approval recorded for "{step}". Use "Reprocess pending" to resume the flow.',
  "automations.success.rejected":
    'Rejection recorded for "{step}". The flow will not continue down that path.',

  // Automations — reprocessing and execution
  "automations.reprocess.failed": "Reprocessing failed: {error}",
  "automations.reprocess.errorUnknown": "error not provided",
  "automations.reprocess.clean":
    "Reprocessing finished with no pending items: {summary}",
  "automations.reprocess.pending":
    "Reprocessing finished, still with {count} pending step(s): {summary}",
  "automations.run.partial": "Routine partially executed: {summary}",
  "automations.run.done": "Routine executed: {summary}",
  "automations.run.failed": "Routine execution failed ({verdict}): {error}",

  // Automations — deletion
  "automations.delete.confirm":
    "Are you sure you want to delete this automation?",
  "automations.delete.confirmLabel": "Delete",
};
