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

  // Company — page / header
  "companyWorkspace.page.title": "Company",
  "companyWorkspace.eyebrow": "Company System",
  "companyWorkspace.title": "Create and run your company",
  "companyWorkspace.description":
    "Strategy, product, engineering, marketing, sales, support and operations in one tenant with autonomous cycles, metrics, approvals and safe pause.",
  "companyWorkspace.refresh": "Refresh",
  "companyWorkspace.loadError": "Could not load the Company System.",
  "companyWorkspace.actionFailed": "The operation failed.",
  "companyWorkspace.loading": "Loading the Company System…",
  "companyWorkspace.created": "Company created as a local tenant.",
  "companyWorkspace.tenant": "tenant {id}",

  // Company — creation
  "companyWorkspace.start.title": "Start with a company",
  "companyWorkspace.start.namePlaceholder": "Company name",
  "companyWorkspace.start.missionPlaceholder":
    "Mission and outcome the company delivers",
  "companyWorkspace.start.positioningPlaceholder":
    "Positioning and differentiator",
  "companyWorkspace.start.audiencePlaceholder": "Target audience",
  "companyWorkspace.start.offerPlaceholder": "Main offer",
  "companyWorkspace.start.offerLabel": "Offer",
  "companyWorkspace.start.budgetPlaceholder": "Monthly limit in BRL",
  "companyWorkspace.start.budgetLabel": "Monthly limit",
  "companyWorkspace.start.create": "Create company and tenant",
  "companyWorkspace.field.mission": "Mission",
  "companyWorkspace.field.positioning": "Positioning",
  "companyWorkspace.field.businessModel": "Business model",
  "companyWorkspace.field.name": "Name",

  // Company — Tel-Agent
  "companyWorkspace.telAgent.title": "Tel-Agent — operational text channel",
  "companyWorkspace.telAgent.description":
    "Command this company by text and receive the response persisted in the tenant. Operations are allowlisted and local; telephony, WhatsApp and external messages stay unconfigured until an approved connection exists.",
  "companyWorkspace.telAgent.operationLabel": "Tel-Agent operation",
  "companyWorkspace.telAgent.operationReport": "Read operational report",
  "companyWorkspace.telAgent.operationBacklog": "Create backlog item",
  "companyWorkspace.telAgent.operationCampaign": "Create campaign draft",
  "companyWorkspace.telAgent.operationTitlePlaceholder": "Operation title",
  "companyWorkspace.telAgent.priorityPlaceholder": "Priority",
  "companyWorkspace.telAgent.priorityLabel": "Tel-Agent priority",
  "companyWorkspace.telAgent.descriptionPlaceholder":
    "Draft objective (does not publish)",
  "companyWorkspace.telAgent.descriptionLabel": "Campaign objective",
  "companyWorkspace.telAgent.messagePlaceholder":
    "E.g. create a task to review onboarding",
  "companyWorkspace.telAgent.messageLabel": "Tel-Agent message",
  "companyWorkspace.telAgent.history":
    "History limited to the last 100 exchanges; entries are redacted before persistence.",
  "companyWorkspace.telAgent.run": "Run operation",
  "companyWorkspace.telAgent.approvalPending":
    "Tel-Agent created a local draft; approval is still required.",
  "companyWorkspace.telAgent.completed":
    "Tel-Agent completed the local operation.",
  "companyWorkspace.telAgent.failed":
    "Tel-Agent could not complete the operation.",

  // Company — indicators
  "companyWorkspace.stats.status": "Status",
  "companyWorkspace.stats.backlog": "Open backlog",
  "companyWorkspace.stats.goalsOnTrack": "Goals on track",
  "companyWorkspace.stats.activeCycles": "Active cycles",
  "companyWorkspace.stats.budgetUsed": "Budget used",
  "companyWorkspace.stats.anomalies": "Anomalies",
  "companyWorkspace.status.paused": "paused",
  "companyWorkspace.status.active": "active",
  "companyWorkspace.status.none": "no company",
  "companyWorkspace.cycleState.enabled": "active",
  "companyWorkspace.cycleState.paused": "paused",

  // Company — identity
  "companyWorkspace.identity.title": "Identity, positioning and model",
  "companyWorkspace.identity.nameUpdated": "Name updated.",
  "companyWorkspace.identity.missionUpdated": "Mission updated.",
  "companyWorkspace.identity.positioningUpdated": "Positioning updated.",
  "companyWorkspace.identity.businessModelUpdated": "Business model updated.",
  "companyWorkspace.departments.title": "Virtual departments",

  // Company — roadmap
  "companyWorkspace.roadmap.title": "Strategic roadmap",
  "companyWorkspace.roadmap.placeholder": "Next strategic outcome",
  "companyWorkspace.roadmap.itemLabel": "Roadmap item",
  "companyWorkspace.roadmap.added": "Item added to the roadmap.",
  "companyWorkspace.roadmap.empty": "No milestone defined.",
  "companyWorkspace.add": "Add",

  // Company — goals
  "companyWorkspace.goals.title": "Goals and KPIs",
  "companyWorkspace.goals.titlePlaceholder": "Goal",
  "companyWorkspace.goals.titleLabel": "Goal title",
  "companyWorkspace.goals.metricPlaceholder": "KPI: leads, revenue, customers",
  "companyWorkspace.goals.metricLabel": "Metric",
  "companyWorkspace.goals.targetLabel": "Target",
  "companyWorkspace.goals.created": "Goal created.",
  "companyWorkspace.goals.empty": "The company KPIs will appear here.",
  "companyWorkspace.create": "Create",

  // Company — backlog
  "companyWorkspace.backlog.title": "Prioritized backlog",
  "companyWorkspace.backlog.placeholder": "Task or experiment",
  "companyWorkspace.backlog.itemLabel": "Backlog item",
  "companyWorkspace.backlog.priorityLabel": "Priority",
  "companyWorkspace.backlog.added": "Item added and prioritized.",
  "companyWorkspace.backlog.empty": "Backlog is empty.",

  // Company — cycles
  "companyWorkspace.cycles.title": "Autonomous cycles",
  "companyWorkspace.cycles.nameLabel": "Cycle name",
  "companyWorkspace.cycles.objectiveLabel": "Cycle objective",
  "companyWorkspace.cycles.frequencyLabel": "Frequency",
  "companyWorkspace.cycles.frequencyDaily": "Daily",
  "companyWorkspace.cycles.frequencyWeekly": "Weekly",
  "companyWorkspace.cycles.create": "Create cycle",
  "companyWorkspace.cycles.created": "Cycle created and linked to the scheduler.",
  "companyWorkspace.cycles.defaultName": "Daily cycle",
  "companyWorkspace.cycles.defaultObjective":
    "Review metrics, roadmap and run the next prioritized task",

  // Company — security and spend
  "companyWorkspace.security.title": "Security, budget and pause",
  "companyWorkspace.security.approvalsNotice":
    "Spend, ad, contract and message actions require approval.",
  "companyWorkspace.security.amountPlaceholder": "Amount in cents",
  "companyWorkspace.security.amountLabel": "Spend amount",
  "companyWorkspace.security.categoryLabel": "Category",
  "companyWorkspace.security.categoryAds": "Ads",
  "companyWorkspace.security.categoryContract": "Contract",
  "companyWorkspace.security.categoryInfrastructure": "Infrastructure",
  "companyWorkspace.security.categorySales": "Sales",
  "companyWorkspace.security.recordSpend": "Record spend",
  "companyWorkspace.security.spendRequested":
    "Spend requested; decision pending before updating the budget.",
  "companyWorkspace.security.resumed": "Company resumed.",
  "companyWorkspace.security.paused": "Company paused safely.",
  "companyWorkspace.security.resume": "Resume",
  "companyWorkspace.security.pause": "Pause",
  "companyWorkspace.signal.title": "Operational signal",
  "companyWorkspace.signal.monthlyBudget": "Monthly budget",
  "companyWorkspace.signal.consumed": "Consumed",
  "companyWorkspace.signal.persistence":
    "Local persistence, tenant and server-side audit.",

  // Endpoint — page and header
  "endpoint.title": "Computers & Endpoint",
  "endpoint.heading": "Computers",
  "endpoint.description":
    "Manage your local and cloud computers for secure mission execution and Computer Use.",
  "endpoint.cloud.create": "Create cloud computer",
  "endpoint.cloud.prerequisiteStrong": "Prerequisite:",
  "endpoint.cloud.prerequisiteText":
    "cloud computers require a configured infrastructure provider (credentials and region). This local-first build does not include a cloud provider, so creation is not available yet. Meanwhile, use your local computer, which already appears connected below.",

  // Endpoint — local host
  "endpoint.host.thisComputer": "This computer",
  "endpoint.host.osLocal": "Local",
  "endpoint.host.statusOnline": "Host online",
  "endpoint.host.statusOffline": "Host offline",
  "endpoint.host.statusChecking": "Checking host",
  "endpoint.host.osLine": "• {os} / Local {version}",
  "endpoint.host.portLabel": "API port:",
  "endpoint.host.osLabel": "System:",
  "endpoint.host.versionLabel": "Server version:",
  "endpoint.remote.button": "Remote access",
  "endpoint.remote.before":
    "By default this host responds only locally. For remote access you must enable",
  "endpoint.remote.strong": "“Expose Hades on the network”",
  "endpoint.remote.afterStart": "in Settings and protect the port",
  "endpoint.remote.afterPort":
    "with TLS/authentication (for example, a reverse proxy). Until that is done, remote access stays off for safety.",
  "endpoint.settings": "Settings",
  "endpoint.tasks": "Tasks",

  // Endpoint — connect a device
  "endpoint.connect.title": "Connect another device",
  "endpoint.connect.description":
    "Add remote nodes, cloud instances or connect your smartphone to control running tasks.",
  "endpoint.connect.computer": "Connect my computer",
  "endpoint.connect.computerHintBefore":
    "To connect another computer as a remote node it must run Hades and be reachable over the network (host and port",
  "endpoint.connect.computerHintAfter":
    "exposed with TLS). This local-first build has no provider that does that provisioning automatically.",
  "endpoint.pairing.generating": "Generating code…",
  "endpoint.pairing.button": "Pair your phone",
  "endpoint.pairing.startFailed": "Could not start pairing now.",
  "endpoint.pairing.codeLabel": "Pairing code:",
  "endpoint.pairing.instructions":
    "Open the Hades companion app on your phone and enter this code to finish pairing.",
  "endpoint.pairing.expiresAt": "The code expires at {time}.",
  "endpoint.pairing.onlyPhone":
    "Pairing only completes from the app on the phone; nothing is sent outside this computer.",

  // Endpoint — API addresses
  "endpoint.api.title": "API addresses & connection",
  "endpoint.api.description":
    "Use the endpoints below to connect Claude Code, Codex, Cursor and local SDKs.",
  "endpoint.api.secure": "Secure",
  "endpoint.api.native": "Native Ollama",
  "endpoint.api.openai": "OpenAI compatible",
  "endpoint.api.openaiLabel": "OpenAI compatible",
  "endpoint.api.anthropic": "Anthropic compatible (Claude Code)",
  "endpoint.api.anthropicLabel": "Anthropic compatible",
  "endpoint.copy.aria": "Copy {label}",

  // Endpoint — gateway
  "endpoint.gateway.title": "Configure third-party inference",
  "endpoint.gateway.description":
    "A single address for local models and third-party APIs. If a provider rejects the call, the gateway tries the next candidate without changing the Claude or Codex configuration.",
  "endpoint.gateway.rotationDefault": "Default rotation · 3 attempts",
  "endpoint.gateway.baseUrl": "Gateway base URL",
  "endpoint.gateway.apiKey": "Gateway API key",
  "endpoint.gateway.keyStored":
    "Stored outside the configuration file, in the variable",
  "endpoint.gateway.copyKey": "gateway key",
  "endpoint.gateway.hide": "Hide",
  "endpoint.gateway.show": "Show",
  "endpoint.gateway.noKey":
    "No active key. Without a key, the gateway only accepts requests from this computer — that is what keeps local inference private.",
  "endpoint.gateway.generating": "Generating…",
  "endpoint.gateway.rotate": "Generate new key",
  "endpoint.gateway.generate": "Generate gateway key",
  "endpoint.gateway.rotateFailed": "Could not generate the gateway key now.",
  "endpoint.gateway.loopbackBefore": "The gateway is in",
  "endpoint.gateway.loopbackStrong": "this computer only",
  "endpoint.gateway.loopbackAfter":
    ": remote clients receive 401 until a key exists. Generate the key above before pointing another device to this address.",
  "endpoint.gateway.aliases": "Aliases accepted in the model field:",
  "endpoint.gateway.providersOne": "{count} provider declared in",
  "endpoint.gateway.providersMany": "{count} providers declared in",

  // Endpoint — quick integration
  "endpoint.integration.heading": "Quick Integration Commands",
  "endpoint.integration.activeModel": "Active model:",
  "endpoint.integration.commandAria": "Integration command {title}",
  "endpoint.claude.description":
    "Run with local models through the compatible route.",
  "endpoint.claude.aria": "Claude integration command",
  "endpoint.codex.description":
    "Ollama configures and opens Codex with the chosen model.",
  "endpoint.codex.aria": "Codex integration command",

  // Onboarding — common
  "onboarding.signIn": "Sign in",
  "onboarding.continue": "Continue",
  "onboarding.retry": "Try again",

  // Onboarding — intro
  "onboarding.intro.alt": "Hades waving hello",
  "onboarding.intro.title": "Welcome to Hades!",
  "onboarding.intro.tagline":
    "Run open models with your coding agents to spend less while keeping your data private.",
  "onboarding.features.connectApps.title": "Connect your apps",
  "onboarding.features.connectApps.description":
    "Power up your coding apps with open models",
  "onboarding.features.switchModel.title": "Switch models easily",
  "onboarding.features.switchModel.description":
    "Switch between frontier models in one click.",
  "onboarding.features.yourData.title": "Your data stays yours",
  "onboarding.features.yourData.description":
    "Local models process your requests on this computer. Cloud models only receive requests when you choose to use them.",

  // Onboarding — account
  "onboarding.welcome.title": "Create an account",
  "onboarding.welcome.subtitle":
    "Create your account to access larger and faster open models.",
  "onboarding.welcome.privacy":
    "Local models process your requests on this computer. If you enable cloud models, requests are sent to the chosen provider when you use them.",
  "onboarding.welcome.openingApps": "Opening the apps…",
  "onboarding.welcome.finishInBrowser": "Finish in your browser…",
  "onboarding.welcome.createAccount": "Create account",
  "onboarding.welcome.cloudOptIn":
    "Enable free cloud models from my account",
  "onboarding.welcome.useLocal": "No thanks — I will use local models",
  "onboarding.welcome.saveCloudFailed":
    "Could not save your cloud models choice. The activation was not confirmed; try again.",

  // Onboarding — first model
  "onboarding.run.title": "Download your first model",
  "onboarding.run.description":
    "Choose a local model to get started. Hades downloads and runs the model on your computer — no terminal needed.",
  "onboarding.run.modelLabel": "Local model",
  "onboarding.run.modelPlaceholder": "e.g. {model}",
  "onboarding.run.preparing": "Preparing download…",
  "onboarding.run.downloading": "Downloading…",
  "onboarding.run.download": "Download",
  "onboarding.run.fileNote":
    "The file can be large; keep Ollama running during the process.",
  "onboarding.run.modelReady": "Model {model} ready for local use.",
  "onboarding.run.downloadFailed":
    "Could not download the model on this computer.",

  // Onboarding — Claude connected
  "onboarding.claudeIntro.alt":
    "Example of mapping Claude models in the Hades settings",
  "onboarding.claudeIntro.title": "Access Hades models in your Claude",
  "onboarding.claudeIntro.description":
    "Hades routes Claude models automatically. Open the Hades settings to choose which models Claude Desktop uses.",

  // Onboarding — apps
  "onboarding.apps.recommended": "Recommended",
  "onboarding.apps.titleApps": "Apps",
  "onboarding.apps.titleOthers": "Other apps",
  "onboarding.apps.empty": "No apps found.",
  "onboarding.apps.loadFailed": "Could not load integrations.",
  "onboarding.apps.checking": "Checking integrations…",
  "onboarding.apps.copyAria": "Copy {name} command",
  "onboarding.apps.copiedAria": "{name} command copied",
  "onboarding.apps.copyFailedTitle": "Could not copy the {name} command",
  "onboarding.apps.copiedTitle": "Launch command copied.",
  "onboarding.apps.pasteHint": "Paste it in your terminal",
  "onboarding.apps.selectHint":
    "Select and copy the command below and paste it in your terminal.",
  "onboarding.chatgpt.description": "Use Hades models in ChatGPT",

  // Onboarding — Claude connection
  "onboarding.claude.readStatusFailed":
    "Hades could not read the Claude connection status.",
  "onboarding.claude.openFailed":
    "Hades connected Claude but could not open the app.",
  "onboarding.claude.connectFailed": "Hades could not connect to Claude.",
  "onboarding.claude.disconnectFailed":
    "Hades could not disconnect from Claude.",
  "onboarding.claude.timeoutOpen":
    "Claude is taking too long to open. Check Claude and try again.",
  "onboarding.claude.timeoutConnect":
    "Claude is taking too long to connect. Check Claude and try again.",
  "onboarding.claude.timeoutAction":
    "Claude is taking too long to {action}. Try again.",
  "onboarding.claude.actionConnect": "connect",
  "onboarding.claude.actionDisconnect": "disconnect",
  "onboarding.claude.completeFailed":
    "Hades could not complete the Claude connection.",
  "onboarding.claude.installNotDetected":
    "The Claude installation was not detected. Try again.",
  "onboarding.claude.installedRestart":
    "Claude is installed. Turn on Connect to restart it with Hades.",
  "onboarding.claude.installerFailed":
    "Hades could not open the Claude installer.",
  "onboarding.claude.macosOnly":
    "The Claude connection is available in the Hades app for macOS.",
  "onboarding.claude.restartToUse":
    "Restart Claude Desktop to use Ollama? Any running task will be interrupted.",
  "onboarding.claude.restartToRemove":
    "Restart Claude Desktop to remove Ollama? Any running task will be interrupted.",
  "onboarding.claude.restartLabel": "Restart",
  "onboarding.claude.connected": "Connected to Hades · {count}",
  "onboarding.claude.installing": "Hades is downloading the Claude installer…",
  "onboarding.claude.waitingInstall":
    "Finish the Claude installation. Hades will connect it automatically.",
  "onboarding.claude.connectingDetail": "Connecting Claude to Hades…",
  "onboarding.claude.launchingDetail": "Opening Claude…",
  "onboarding.claude.disconnectingDetail":
    "Restoring Claude's normal connection…",
  "onboarding.claude.notInstalled":
    "We will download Claude and connect it to Hades.",
  "onboarding.claude.ready": "Use Hades models in your Claude Code.",
  "onboarding.claude.disconnect": "Disconnect Claude",
  "onboarding.claude.connecting": "Connecting Claude",
  "onboarding.claude.connect": "Connect Claude",
  "onboarding.claude.disconnectTitle": "Disconnect",
  "onboarding.claude.connectTitle": "Connect",
  "onboarding.claude.statusDownloading": "Downloading…",
  "onboarding.claude.statusFinishingInstall": "Finish the installation…",
  "onboarding.claude.statusConnecting": "Connecting…",
  "onboarding.claude.statusLaunching": "Opening…",
  "onboarding.claude.statusDisconnecting": "Disconnecting…",

  // Agent Center (AgentCenterPanel)
  "agentCenter.title": "Agents and Teams",
  "agentCenter.subtitle": "deterministic ledger · drift by hash",
  "agentCenter.actions.list": "List agents",
  "agentCenter.actions.catalog": "View catalog",
  "agentCenter.actions.evaluation": "View evaluation criteria",
  "agentCenter.actions.teams": "View teams",
  "agentCenter.actions.runEvaluation": "Run real evaluation",
  "agentCenter.actions.quarantine": "Hold in quarantine",
  "agentCenter.actions.triage": "Triage import",
  "agentCenter.actions.backup": "Backup and restore",
  "agentCenter.actions.drift": "Check integrity",
  "agentCenter.actions.remove": "Remove installation",
  "agentCenter.actions.update": "Update (with backup)",
  "agentCenter.actions.install": "Record installation",
  "agentCenter.target.aria": "Destination format",
  "agentCenter.destination.placeholder": "relative destination",
  "agentCenter.destination.aria": "Relative destination",
  "agentCenter.idle": "Nothing queried in this session.",
  "agentCenter.loading": "Querying...",
  "agentCenter.list.countOne": "{count} installation recorded",
  "agentCenter.list.countMany": "{count} installations recorded",
  "agentCenter.catalog.version":
    "version {version} · {total} persona(s) · {evaluated} evaluated",
  "agentCenter.catalog.noneEvaluated":
    "No catalog persona has been evaluated. This is not «approved»: it is «not measured».",
  "agentCenter.catalog.unrated": "NOT EVALUATED",
  "agentCenter.evaluation.noNote": "no note from the server",
  "agentCenter.evaluation.inferenceOn": "execution bound to an engine",
  "agentCenter.evaluation.inferenceOff":
    "execution against a model NOT bound in this runtime",
  "agentCenter.evaluation.suiteLine":
    "{division} · v{version} · {cases} case(s) · {refusals} for refusal/injection",
  "agentCenter.evaluation.states":
    "{count} declared states · {terminal} terminal(s)",
  "agentCenter.teams.summary":
    "{total} team(s) · {functional} functional · {missing} with missing tools · {invalid} invalid",
  "agentCenter.teams.noneProven":
    "no team has been proven in a real mission",
  "agentCenter.teams.rolesOne": "{count} role",
  "agentCenter.teams.rolesMany": "{count} roles",
  "agentCenter.teams.valid": "valid",
  "agentCenter.teams.invalid": "INVALID ({count})",
  "agentCenter.teams.missingTools": "missing: {tools}",
  "agentCenter.teams.noMissingTools": "no missing tools declared",
  "agentCenter.triage.placeholder":
    "Paste the persona manifest (JSON) to triage",
  "agentCenter.triage.status":
    "state: {state} · {blocks} blocker(s) · {warnings} warning(s) · {install}",
  "agentCenter.triage.installed": "INSTALLED",
  "agentCenter.triage.notInstalled": "nothing installed",
  "agentCenter.triage.note":
    "triage reads and refuses; it writes nothing and activates nothing",
  "agentCenter.realEval.status":
    "state: {state} · {cases} case(s) · {failed} failure(s) · engine: {engine}",
  "agentCenter.realEval.note":
    "runs the suite against the local engine; it does not approve for production",
  "agentCenter.realEval.passed": "passed",
  "agentCenter.realEval.failed": "failed",
  "agentCenter.quarantine.countOne": "{count} persona held",
  "agentCenter.quarantine.countMany": "{count} personas held",
  "agentCenter.quarantine.release": "release",
  "agentCenter.quarantine.empty":
    "No persona held. Holding one creates the first entry.",
  "agentCenter.quarantine.reviewerPlaceholder":
    "reviewer (required to release)",
  "agentCenter.quarantine.releaseNote":
    "Releasing moves to SECURITY_REVIEWED and requires a reviewer. It does not approve for production.",
  "agentCenter.backups.countOne": "{count} backup stored",
  "agentCenter.backups.countMany": "{count} backups stored",
  "agentCenter.backups.restore": "restore",
  "agentCenter.backups.restored": "Restored from backup",
  "agentCenter.backups.empty":
    "No backup yet. Storing one creates the first.",
  "agentCenter.backups.restoreNote":
    "Restoring requires choosing an id: there is no «restore the latest».",
  "agentCenter.install.created":
    "201 created · {id} · {bytes} bytes rendered",
  "agentCenter.install.existing":
    "200 already recorded · {id} · {bytes} bytes rendered",
  "agentCenter.install.noId": "no id",
  "agentCenter.update.done": "Updated with backup",
  "agentCenter.update.bytes": "bytes",
  "agentCenter.remove.line": "Removed from the ledger: {id} · {state}",
  "agentCenter.remove.fileKept": "the file at the destination was preserved",
  "agentCenter.remove.noFile": "no file to preserve",
  "agentCenter.error.centerMissing":
    "The Agent Center is not connected in this runtime (503): this is not «no agents», it is «no storage».",
  "agentCenter.error.conflict":
    "A record with this key and different hashes already exists: resolve the conflict, do not reinstall over it.",
  "agentCenter.error.generic": "Could not run.",
  "agentCenter.footer":
    "Recording an installation does not write to the destination: it computes the hashes and saves them to the ledger. The destination must be relative and the server refuses `..` — a destination the client writes is a destination the client chooses.",
  // Settings (Settings.tsx)
  "settings.interfaceMode.title": "Interface mode",
  "settings.interfaceMode.description":
    "Simple shows only the essentials. Advanced also shows MCP, diagnostics and technical details.",
  "settings.interfaceMode.simple": "Simple",
  "settings.interfaceMode.advanced": "Advanced",
  // Interface language (control in SettingsLanguage.tsx)
  "settings.language.title": "Interface language",
  "settings.language.description":
    "Choose the language of the interface text. The choice is saved on this device.",
  "settings.language.pt-BR": "Português (Brasil)",
  "settings.language.en": "English",
  "settings.discardDrafts.message":
    "Discard the model changes for apps that have not been applied yet?",
  "settings.discardDrafts.confirm": "Discard",
  "settings.reset.failed":
    "Could not restore all settings. Check the options above and try again.",
  "settings.reset.restoring": "Restoring…",
  "settings.reset.action": "Restore defaults",
  "settings.native.loading": "Querying native configuration…",
  "settings.native.unavailable": "Native configuration unavailable right now",
  "settings.native.unavailableHint":
    "The agentic panel above remains available for sanitized diagnostics; start the Ollama server to edit the native preferences.",
  "settings.account.upgrade": "Upgrade",
  "settings.account.manage": "Manage",
  "settings.account.disconnect": "Disconnect",
  "settings.account.connect": "Sign in",
  "settings.account.label": "Ollama account",
  "settings.account.notConnected":
    "Not connected · sign in to use cloud models and web search",
  "settings.account.title": "Account & Operator ({name})",
  "settings.account.storageLabel": "Local storage:",
  "settings.account.storageFallback": "Local workspace active",
  "settings.account.databaseLabel": "Database:",
  "settings.account.databaseValue": "SQLite (local storage)",
  "settings.account.modeLabel": "Mode:",
  "settings.account.modeValue": "Local-first (no cloud dependency)",
  "settings.cloud.label": "Cloud",
  "settings.cloud.forcedOff":
    "The OLLAMA_NO_CLOUD environment variable is forcing the cloud off.",
  "settings.cloud.description": "Enable cloud models and web search.",
  "settings.appsMenu.label": "Show apps in the menu",
  "settings.appsMenu.description":
    "Show connected apps at the top of the Ollama menu.",
  "settings.autoUpdate.label": "Download updates automatically",
  "settings.autoUpdate.on": "Download updates automatically when available.",
  "settings.autoUpdate.off": "Updates will not be downloaded automatically.",
  "settings.expose.label": "Expose Hades on the network",
  "settings.expose.description":
    "Allow other devices or services to access Hades.",
  "settings.modelsDir.label": "Models location",
  "settings.modelsDir.description": "Where the models are stored.",
  "settings.modelsDir.browse": "Browse",
  "settings.context.label": "Context size",
  "settings.context.description":
    "The context size defines how much of the conversation the model can remember and use to generate answers. Larger values keep more history but consume more memory.",
  "settings.apps.heading": "Apps",
  "settings.agentMode.label": "Enable agent mode",
  "settings.agentMode.description":
    "Use tools in multiple steps to fulfil requests",
  "settings.toolsMode.label": "Enable tools mode",
  "settings.toolsMode.description":
    "Use tools in a single step to fulfil requests",
  "settings.personalization.title": "Personalisation",
  "settings.personalization.description":
    "Custom instructions and long-term memory to guide the agent's behaviour.",
  "settings.personalization.languageLabel": "Language:",
  "settings.personalization.languageBefore": "Hades always answers in",
  "settings.personalization.languageName": "Brazilian Portuguese",
  "settings.personalization.languageAfter":
    "by default (unless you ask for another language in the conversation). This is already active.",
  "settings.personalization.instructionsLabel": "Custom instructions",
  "settings.personalization.instructionsBefore":
    "Write down style/response preferences. These instructions are saved on this computer and",
  "settings.personalization.instructionsStrong":
    "sent to the model in every conversation",
  "settings.personalization.instructionsAfter":
    ", together with Hades' default behaviour.",
  "settings.personalization.instructionsPlaceholder":
    "E.g. short, direct answers; always with examples.",
  "settings.personalization.memoryTitle": "Long-Term Memory",
  "settings.personalization.memoryDescription":
    "Cross-session context lookup is not configured in this environment yet.",
  "settings.personalization.memoryUnavailable": "Unavailable",
  "settings.saved": "Saved",
  "settings.operator.defaultName": "Local operator",
  "settings.operator.defaultPlan": "Local-first mode",
  "settings.error.connectUrl": "Failed to get connect URL",
  "settings.error.connectFailed": "Failed to connect to Ollama account",
  "settings.error.disconnectFailed": "Failed to disconnect Ollama account",
  // Company campaign scheduling (CompanyCampaignSchedulingPanel)
  "companyCampaign.load.panelFailed": "Campaign panel unavailable.",
  "companyCampaign.load.channelsFailed": "Marketplace catalog unavailable.",
  "companyCampaign.action.failed": "The campaign operation failed.",
  "companyCampaign.sync.done":
    "Synchronisation ran against the sandbox double.",
  "companyCampaign.sync.failed": "Synchronisation failed.",
  "companyCampaign.eyebrow": "Campaigns and marketplaces",
  "companyCampaign.title":
    "Scheduling, measured metrics and sandbox synchronisation",
  "companyCampaign.intro.before":
    "Scheduling requires an approved campaign and a start time in the future; metrics require a campaign that has already run and only add up. Marketplace synchronisation sends",
  "companyCampaign.intro.items": "the items you build",
  "companyCampaign.intro.after": "against a local double:",
  "companyCampaign.intro.never": "nothing is published to an external platform",
  "companyCampaign.refresh": "Refresh panel",
  "companyCampaign.external.label": "External publication:",
  "companyCampaign.schedule.heading": "Schedule and measure a campaign",
  "companyCampaign.fields.hours": "Starts in hours",
  "companyCampaign.fields.impressions": "Measured impressions",
  "companyCampaign.fields.clicks": "Measured clicks",
  "companyCampaign.campaigns.empty":
    "No campaign yet: create one in the Growth OS panel above.",
  "companyCampaign.scheduledFor": "Scheduled for {date}",
  "companyCampaign.measured":
    "Measured: {impressions} impressions · {clicks} clicks · {conversions} conversions",
  "companyCampaign.action.schedule": "Schedule",
  "companyCampaign.action.metrics": "Record metrics",
  "companyCampaign.success.scheduled": "Campaign scheduled in the sandbox.",
  "companyCampaign.success.metrics":
    "Measured metrics recorded (added to the total).",
  "companyCampaign.total":
    "Total measured: {impressions} impressions · {clicks} clicks · CTR {ctr} · conversion {conversion} · spend {spend}",
  "companyCampaign.ingestions.prefix":
    "Accepted deliveries (idempotency): {count} — redelivery with the same key",
  "companyCampaign.ingestions.noSum": "does not add up",
  "companyCampaign.manual.prefix":
    "No automatic delivery recorded: the numbers above come from",
  "companyCampaign.manual.badge": "manual entry",
  "companyCampaign.marketplaces.heading": "Marketplaces",
  "companyCampaign.marketplaces.descriptionBefore":
    "The catalog reports the channel's",
  "companyCampaign.marketplaces.descriptionStrong": "real",
  "companyCampaign.marketplaces.descriptionAfter":
    "state. Without a configured credential the channel is not ready — and the panel says so instead of showing a green badge.",
  "companyCampaign.channels.empty": "No channel in the catalog.",
  "companyCampaign.liveTested": "Live tested: {value}",
  "companyCampaign.yes": "yes",
  "companyCampaign.no": "no",
  "companyCampaign.items.heading": "Items to synchronise",
  "companyCampaign.items.noProducts":
    "No product in the catalog: register a product in Growth OS to build the item.",
  "companyCampaign.items.stock": "Stock",
  "companyCampaign.items.add": "Add item",
  "companyCampaign.items.line":
    "{entity} · {sku} · version {version} · stock {quantity}",
  "companyCampaign.items.syncOne": "Synchronise {count} item (sandbox)",
  "companyCampaign.items.syncMany": "Synchronise {count} items (sandbox)",
  "companyCampaign.items.needOne":
    "Add at least one item: synchronising without an item is refused by the server.",
  "companyCampaign.sync.summary":
    "{applied} applied · {skipped} skipped · {conflicts} conflicts · {used}/{limit} operations · {homologation}",

  // Home (HomePage.tsx)
  "home.actions.chat.description": "Local-first chat with Ollama models and configured providers.",
  "home.actions.chat.eyebrow": "Chat and Models",
  "home.actions.chat.title": "Conversation with a model",
  "home.actions.compile.description": "Plan, run in split-screen, approve steps and generate code with artifacts.",
  "home.actions.compile.eyebrow": "Compile >",
  "home.actions.compile.title": "Sites, apps and games",
  "home.actions.create.description": "Executive documents, presentations and media generated by missions.",
  "home.actions.create.eyebrow": "Create >",
  "home.actions.create.title": "Slides, images and videos",
  "home.actions.local.description": "Open and orchestrate local files and projects with persistent context.",
  "home.actions.local.eyebrow": "Local workspace",
  "home.actions.local.title": "Start from a local file",
  "home.attachments.aria": "Attached files",
  "home.attachments.attachAria": "Attach file",
  "home.attachments.chooseAria": "Choose files to attach",
  "home.attachments.limitCount": "{filename}: limit of {count} files per task",
  "home.attachments.limitTotal": "{filename}: total limit of {megabytes} MB per task",
  "home.attachments.note": "Attachments are sent to Hades for local indexing (up to 10 files, 10 MB each and 32 MB in total). Images are not indexed yet. Automatic mode is read-only; writing requires consent and review.",
  "home.attachments.projectTitle": "Attachments: {objective}",
  "home.attachments.remove": "Remove {filename}",
  "home.commands.hint": "Real commands: /goal delegates, /plan only plans, /test runs tests and /review reviews. Ctrl+Enter to start.",
  "home.commands.reviewObjective": "Review the current code or artifact, point out risks and actionable recommendations. Context: {objective}",
  "home.commands.testObjective": "Run the current project's test runner and record the result. Context: {objective}",
  "home.consent.remoteSend": "I understand that the selected remote provider may receive my request and excerpts of the attachments to generate the answer.",
  "home.consent.workspaceWrite": "Allow changes to the project files. If checked, the task will not run automatically: you will review and start the mission in the console.",
  "home.error.attachmentsFailed": "Failed to upload or index the attachments.",
  "home.error.projectNotConfirmed": "The server did not confirm the creation of the attachments project.",
  "home.error.remoteConsentRequired": "Confirm that you understand the selected remote provider may receive the request and the context of the attachments.",
  "home.heading.description": "A local-first runtime: chat, compile, search, automate and track missions with approvals and data in your own environment.",
  "home.model.auto": "Automatic · free-first",
  "home.model.choose": "Choose model",
  "home.model.free": "Free",
  "home.model.manual": "Manual",
  "home.model.modeTitle": "Task AI",
  "home.model.paid": "Paid",
  "home.objective.aria": "New task objective",
  "home.objective.label": "Describe a task",
  "home.objective.placeholder": "Assign a task or type / for more options...",
  "home.quickActions.code.label": "Analyze code",
  "home.quickActions.code.prompt": "Inspect the repository, audit security and list recommendations",
  "home.quickActions.research.label": "Deep research",
  "home.quickActions.research.prompt": "Perform in-depth market research with synthesis and cited sources",
  "home.quickActions.site.label": "Create site",
  "home.quickActions.site.prompt": "Create a modern responsive landing page with Tailwind and React",
  "home.quickActions.slides.label": "Create slides",
  "home.quickActions.slides.prompt": "Create a professional slide deck about AI innovation",
  "home.quickActions.title": "Quick actions:",
  "home.startOptions.aria": "Ways to start",
  "home.startOptions.attach": "Attach file",
  "home.startOptions.import": "Import project",
  "home.startOptions.remove": "Remove",
  "home.startOptions.removeSelected": "Remove selected project",
  "home.startOptions.selected": "Selected project: {name}",
  "home.startOptions.title": "Start with:",
  "home.submit.aria": "Start in the agentic console",
  "home.submit.review": "Continue to review",
  "home.submit.sending": "Sending…",
  "home.submit.start": "Start",

  // Console Codex Desktop (CodexDesktopRow.tsx)
  "codexDesktop.action.addModels": "Add Hades models to ChatGPT",
  "codexDesktop.action.addModelsTitle": "Add Hades models",
  "codexDesktop.action.connecting": "Connecting ChatGPT",
  "codexDesktop.action.finishConnection": "Finish the ChatGPT connection",
  "codexDesktop.action.installTitle": "Install ChatGPT and add the Hades models",
  "codexDesktop.action.removeModels": "Remove the Hades models from ChatGPT",
  "codexDesktop.action.removeModelsTitle": "Remove Hades models",
  "codexDesktop.action.retry": "Try again",
  "codexDesktop.action.retryAria": "Try saving the progress again",
  "codexDesktop.confirm.restartLabel": "Restart",
  "codexDesktop.confirm.restartToAdd": "Restart ChatGPT to add Ollama models? Any running task will be interrupted.",
  "codexDesktop.confirm.restartToRemove": "Restart ChatGPT to remove Ollama models? Any running task will be interrupted.",
  "codexDesktop.description.connectedMany": "Connected to Hades · {count} requests in this session",
  "codexDesktop.description.connectedOne": "Connected to Hades · {count} request in this session",
  "codexDesktop.description.downloadIntro": "Let's download ChatGPT and connect it to Hades.",
  "codexDesktop.description.useCodexModels": "Use the Hades models in ChatGPT's Codex mode.",
  "codexDesktop.error.addModels": "Hades could not add your models to ChatGPT.",
  "codexDesktop.error.completeConnection": "Hades could not complete the connection to ChatGPT.",
  "codexDesktop.error.installFailed": "Hades could not install ChatGPT.",
  "codexDesktop.error.installNotDetected": "The ChatGPT installation was not detected. Try again.",
  "codexDesktop.error.integrationUnavailable": "The ChatGPT integration is unavailable.",
  "codexDesktop.error.readStatus": "Hades could not read the connection status with ChatGPT.",
  "codexDesktop.error.removeModels": "Hades could not remove your models from ChatGPT.",
  "codexDesktop.error.restartToConnect": "ChatGPT is installed. Click Connect to restart it with the Hades models.",
  "codexDesktop.error.saveProgress": "Hades could not save your progress. Try again.",
  "codexDesktop.notice.modelsAdded": "Hades models added alongside the Codex models",
  "codexDesktop.notice.modelsRemoved": "Hades models removed · the Codex models remain available",
  "codexDesktop.progress.connecting.description": "Connecting ChatGPT to Hades…",
  "codexDesktop.progress.connecting.label": "Connecting…",
  "codexDesktop.progress.disconnecting.description": "Restoring ChatGPT's normal connection…",
  "codexDesktop.progress.disconnecting.label": "Disconnecting…",
  "codexDesktop.progress.installing.description": "Hades is downloading the ChatGPT installer…",
  "codexDesktop.progress.installing.label": "Downloading…",
  "codexDesktop.progress.saving.description": "Saving your progress…",
  "codexDesktop.progress.saving.label": "Saving…",
  "codexDesktop.progress.waitingForInstall.description": "Complete the ChatGPT installation. Hades will connect it automatically.",
  "codexDesktop.progress.waitingForInstall.label": "Complete the installation…",

  // Catalogo de conectores (lib/connectorCatalog.ts)
  // `category` is left untranslated: it is a backend contract identifier, not copy.
  "connectorCatalog.gmail.description": "Write replies, search the inbox and summarise email conversations.",
  "connectorCatalog.outlookMail.description": "Write, search and manage Outlook emails without leaving the flow.",
  "connectorCatalog.slack.description": "Read channels, send messages and follow team conversations.",
  "connectorCatalog.whatsapp.description": "Send and receive messages in approved support flows.",
  "connectorCatalog.discord.description": "Follow servers and channels and publish updates with approval.",
  "connectorCatalog.googleWorkspace.description": "Access Drive, Docs and Sheets files and let the agent help with context.",
  "connectorCatalog.googleCalendar.description": "Understand your schedule, manage events and make the most of your time.",
  "connectorCatalog.notion.description": "Search the workspace, update notes and automate workflows.",
  "connectorCatalog.microsoft365.description": "Word, Excel, Outlook and Teams in a single productivity connector.",
  "connectorCatalog.jira.description": "Browse issues, sprints and boards and create tasks with approval.",
  "connectorCatalog.linear.description": "Follow issues and engineering cycles and propose updates.",
  "connectorCatalog.github.description": "Repositories, issues and pull requests with code context.",
  "connectorCatalog.gitlab.description": "Repositories, merge requests and pipelines in one place.",
  "connectorCatalog.supabase.description": "Query tables, functions and logs from your managed Postgres project.",
  "connectorCatalog.instagram.description": "Generate and publish posts, stories and reels with approval.",
  "connectorCatalog.metaAds.description": "Ad insights and optimisation to save hours and maximise return.",
  "connectorCatalog.tiktok.description": "Publish content and track reach and engagement metrics.",
  "connectorCatalog.linkedin.description": "Publish updates and search connections and pages with approval.",
  "connectorCatalog.videoEditor.description": "Create and refine videos on an editable timeline.",
  "connectorCatalog.higgsfield.description": "Generate cinematic images and videos with Higgsfield.",
  "connectorCatalog.canva.description": "Create and edit designs and export assets for your missions.",
  "connectorCatalog.similarweb.description": "Analyse traffic and SEO data for any domain or URL.",
  "connectorCatalog.worldBank.description": "Official World Bank statistics for any country, region or income bracket.",
  "connectorCatalog.bigquery.description": "Query large datasets and cross-reference them with mission context.",
  "connectorCatalog.stripe.description": "Read payments, subscriptions and financial metrics (read-only).",
  "connectorCatalog.googleAnalytics.description": "Audience, acquisition and conversion metrics for your sites.",

  // AgenticConsole
  "agenticConsole.approval.send":
    "It can send or publish content outside Hades. Review the recipient, text and scope before deciding.",
  "agenticConsole.approval.spend":
    "It can incur spend, a financial commitment or an ad. Confirm the amount, limit and recipient before deciding.",
  "agenticConsole.approval.write":
    "It can change files, publish or modify an external service. Check the summary and the affected project.",
  "agenticConsole.approval.blocked":
    "The server pre-emptively blocked this action until an explicit decision. Review the context before deciding.",
  "agenticConsole.provider.ollamaLocal": "Local Ollama",
  "agenticConsole.provider.local": "local",
  "agenticConsole.provider.configured": "configured",
  "agenticConsole.provider.noKey": "no key",
  "agenticConsole.error.load": "Could not load the agentic runtime.",
  "agenticConsole.error.autoMission": "Failed to start the automatic mission.",
  "agenticConsole.error.createMission": "Failed to create the mission.",
  "agenticConsole.error.reasonRequired": "Provide a reason before deciding the approval.",
  "agenticConsole.error.decision": "Failed to record the decision.",
  "agenticConsole.error.run": "Failed to start the mission.",
  "agenticConsole.error.orchestration": "Failed to start the orchestration.",
  "agenticConsole.error.research": "Deep research failed.",
  "agenticConsole.aria.console": "Agentic console",
  "agenticConsole.header.eyebrow": "DZ23 Agent Runtime",
  "agenticConsole.header.title": "Mission Console",
  "agenticConsole.header.description":
    "Plan, orchestrate, research, approve, run and observe operations with the same persistent trail used by the local API.",
  "agenticConsole.objective.label": "New task",
  "agenticConsole.objective.placeholder":
    "Describe what you want to build, research, review or automate",
  "agenticConsole.engine.label": "Engine",
  "agenticConsole.model.label": "Model",
  "agenticConsole.project.label": "Project",
  "agenticConsole.project.none": "No project",
  "agenticConsole.isolate.label": "Run in an isolated copy",
  "agenticConsole.allowWrite.label": "Allow writes",
  "agenticConsole.create": "Create mission",
  "agenticConsole.help.providerLabel": "Selected engine:",
  "agenticConsole.help.providerDetail":
    "Remote providers only appear when the catalog publishes them; the mission fails closed without an adapter/credential.",
  "agenticConsole.help.write":
    "The mission starts read-only; enable “Allow writes” when the task needs to change files.",
  "agenticConsole.help.isolate":
    "“Run in an isolated copy” creates a Git snapshot per mission and leaves the origin untouched; it requires a project linked to the organisation and excludes sensitive paths/files.",
  "agenticConsole.help.isolateAvailable": "The option is only available with a project selected.",
  "agenticConsole.help.scopes":
    "Elevated scopes (terminal, browser, desktop, CLIs, MCP, connectors) only appear under «Mission permissions», derived from the runtime tool catalog.",
  "agenticConsole.help.server": "The server validates grants and requires approval for write effects.",
  "agenticConsole.help.isolateHint":
    "The isolated copy is opt-in; select a project. Secrets, ignored files and symlinks are not copied.",
  "agenticConsole.metrics.created": "Created",
  "agenticConsole.metrics.completed": "Completed",
  "agenticConsole.metrics.retries": "Retries",
  "agenticConsole.metrics.tools": "Tools",
  "agenticConsole.orchestration.eyebrow": "Multi-agent",
  "agenticConsole.orchestration.title": "Orchestrate specialists",
  "agenticConsole.orchestration.placeholder":
    "E.g.: research competitors, review security and propose an implementation",
  "agenticConsole.orchestration.start": "Start orchestration",
  "agenticConsole.orchestration.summary": "View synthesis",
  "agenticConsole.research.eyebrow": "Deep research",
  "agenticConsole.research.title": "Sources and citations",
  "agenticConsole.research.queryPlaceholder": "Research question",
  "agenticConsole.research.urlsPlaceholder": "One public HTTPS URL per line",
  "agenticConsole.research.submit": "Search",
  "agenticConsole.view.split": "Split view",
  "agenticConsole.view.classic": "Classic view",
  "agenticConsole.plan.title": "Plan",
  "agenticConsole.plan.step": "Step {current} of {total}",
  "agenticConsole.plan.approval": "approval",
  "agenticConsole.run": "Run",
  "agenticConsole.approvals.title": "Approvals",
  "agenticConsole.approvals.empty": "No pending approvals.",
  "agenticConsole.approvals.note":
    "Nothing runs until the server validates the decision. The mission cannot approve on its own.",
  "agenticConsole.approvals.reasonLabel": "Decision reason",
  "agenticConsole.approvals.reasonAria": "Decision reason for {step}",
  "agenticConsole.approvals.reasonPlaceholder":
    "Explain why this action should be approved or rejected",
  "agenticConsole.approvals.approve": "Approve",
  "agenticConsole.approvals.reject": "Reject",

  // BuildPlanPanel
  "buildPlan.title": "Build & Release",
  "buildPlan.start.accepted": "Build for «{target}» accepted{suffix} — follow it in «Build status».",
  "buildPlan.start.incompletePlan": "The plan for «{target}» is incomplete: {message}",
  "buildPlan.intro.before": "This panel only",
  "buildPlan.intro.detect": "detects",
  "buildPlan.intro.middle": "the project and",
  "buildPlan.intro.probe": "probes",
  "buildPlan.intro.after":
    "the installed tools. It installs nothing, builds nothing and signs nothing.",
  "buildPlan.repo.label": "Project path",
  "buildPlan.repo.placeholder": "e.g.: C:\\path\\to\\project",
  "buildPlan.loading": "Analyzing the project…",
  "buildPlan.error.analyze": "Could not analyze the project:",
  "buildPlan.field.project": "Project",
  "buildPlan.field.stacks": "Stacks",
  "buildPlan.stacks.none": "none recognized",
  "buildPlan.field.packageManagers": "Package managers",
  "buildPlan.packageManagers.none": "none detected",
  "buildPlan.field.targets": "Targets",
  "buildPlan.targets.none": "none",
  "buildPlan.field.tools": "Tools",
  "buildPlan.tools.ready": "{ready} of {total} ready",
  "buildPlan.tools.missing": ", {count} missing",
  "buildPlan.noTargets":
    "The project declares no target that HADES knows. There is no build plan to construct — this is not an error, it is a result.",
  "buildPlan.blockers.title": "What is missing",
  "buildPlan.reasons.title": "Why",
  "buildPlan.independent.title": "What can move forward without unblocking anything",
  "buildPlan.plan.caption": "Plan by target",
  "buildPlan.col.target": "Target",
  "buildPlan.col.status": "Status",
  "buildPlan.col.actions": "Actions",
  "buildPlan.col.tool": "Tool",
  "buildPlan.col.detail": "Detail",
  "buildPlan.status.planFailed": "could not be planned",
  "buildPlan.status.ready": "ready",
  "buildPlan.status.blocked": "blocked",
  "buildPlan.blockers.none": "nothing",
  "buildPlan.start.starting": "Starting…",
  "buildPlan.start.action": "Start build",
  "buildPlan.probes.details": "Details of the probed tools",
  "buildPlan.probes.caption": "Tools by target",
  "buildPlan.warnings.title": "Detection warnings",
  "buildPlan.artifact.title": "Verify an artifact",
  "buildPlan.artifact.intro.before": "Measures the SHA-256 of a file that",
  "buildPlan.artifact.intro.exists": "already exists",
  "buildPlan.artifact.intro.after":
    "and, given the reference, compares it. It does not build, sign or install. The path must be",
  "buildPlan.artifact.intro.absolute": "absolute",
  "buildPlan.artifact.path.label": "Artifact path",
  "buildPlan.artifact.path.placeholder": "e.g.: C:\\path\\hades.exe",
  "buildPlan.artifact.sha.label": "Expected SHA-256 (optional for measuring)",
  "buildPlan.artifact.sha.placeholder": "64 hexadecimal characters",
  "buildPlan.artifact.measure": "Measure",
  "buildPlan.artifact.verify": "Verify",
  "buildPlan.artifact.measuring": "Measuring the artifact…",
  "buildPlan.artifact.checksumAria": "Measured checksum",
  "buildPlan.artifact.file": "File",
  "buildPlan.artifact.size": "Size",
  "buildPlan.artifact.resultAria": "Verification result",
  "buildPlan.artifact.verdict": "Verdict",
  "buildPlan.artifact.shaMeasured": "Measured SHA-256",
  "buildPlan.artifact.shaExpected": "Expected SHA-256",
  "buildPlan.artifact.signature": "Signature",
  "buildPlan.artifact.measureFailed": "could not measure the artifact",

  // CompanyGrowthPanel
  "companyGrowth.error.report": "Growth OS unavailable.",
  "companyGrowth.error.action": "The Growth OS operation failed.",
  "companyGrowth.title": "Controlled marketing, affiliates and dropshipping",
  "companyGrowth.intro":
    "Everything starts in a local sandbox. Publishing, ads, partners and fulfillment require approval and an authorized connector.",
  "companyGrowth.refresh": "Refresh growth",
  "companyGrowth.stats.campaigns": "Campaigns",
  "companyGrowth.stats.active": "Active",
  "companyGrowth.stats.programs": "Programs",
  "companyGrowth.stats.conversions": "Conversions",
  "companyGrowth.stats.products": "Products",
  "companyGrowth.stats.ordersDone": "Completed orders",
  "companyGrowth.campaign.title": "Campaign",
  "companyGrowth.campaign.name": "Campaign name",
  "companyGrowth.campaign.objectivePlaceholder": "Objective: generate leads",
  "companyGrowth.campaign.objective": "Campaign objective",
  "companyGrowth.campaign.channel": "Campaign channel",
  "companyGrowth.campaign.channelSocial": "Social networks",
  "companyGrowth.campaign.channelEmail": "Email",
  "companyGrowth.campaign.channelAffiliate": "Affiliates",
  "companyGrowth.campaign.channelContent": "Content",
  "companyGrowth.campaign.create": "Create campaign",
  "companyGrowth.campaign.created": "Campaign created as a draft; approval pending.",
  "companyGrowth.campaign.approve": "Approve",
  "companyGrowth.campaign.approved": "Campaign approved in the sandbox.",
  "companyGrowth.campaign.launch": "Start",
  "companyGrowth.campaign.launched": "Campaign started in the sandbox.",
  "companyGrowth.program.title": "Affiliate program",
  "companyGrowth.program.name": "Program name",
  "companyGrowth.program.networkPlaceholder": "Network or partner",
  "companyGrowth.program.network": "Affiliate network",
  "companyGrowth.program.create": "Create program",
  "companyGrowth.program.created": "Program created; approval pending.",
  "companyGrowth.program.approve": "Approve program",
  "companyGrowth.program.approved": "Affiliate program approved in the sandbox.",
  "companyGrowth.product.title": "Product and order",
  "companyGrowth.product.name": "Product name",
  "companyGrowth.product.supplierPlaceholder": "Sandbox supplier",
  "companyGrowth.product.supplier": "Supplier",
  "companyGrowth.product.priceAria": "Price in cents",
  "companyGrowth.product.pricePlaceholder": "Price in cents",
  "companyGrowth.product.inventory": "Inventory",
  "companyGrowth.product.add": "Add product",
  "companyGrowth.product.added": "Product created in the sandbox catalog.",
  "companyGrowth.order.customer": "Customer reference",
  "companyGrowth.order.quantity": "Quantity",
  "companyGrowth.order.created": "Order created; approval pending.",
  "companyGrowth.order.create": "Create order",
  "companyGrowth.order.approve": "Approve order",
  "companyGrowth.order.approved": "Order approved in the sandbox.",
  "companyGrowth.order.fulfill": "Fulfill sandbox",
  "companyGrowth.order.fulfilled": "Order fulfilled in the sandbox.",

  // ProvidersPage
  "providers.common.cancel": "Cancel",
  "providers.common.close": "Close",
  "providers.common.saving": "Saving…",
  "providers.models.queryError": "Could not query this provider's models.",
  "providers.models.loading": "Querying the provider's models…",
  "providers.models.listError": "Could not list the models: {message}",
  "providers.models.stale":
    "No longer exist at the provider and were unselected: {models}",
  "providers.models.filterPlaceholder": "Filter {count} models",
  "providers.models.filterAria": "Filter models",
  "providers.models.saveError": "Could not save the selected models.",
  "providers.models.use": "Use {count} model(s)",
  "providers.card.model.one": "model",
  "providers.card.model.other": "models",
  "providers.card.keyConfigured": "Key configured",
  "providers.card.noKeyNeeded": "No key required",
  "providers.card.noKey": "No key",
  "providers.card.replaceKey": "Replace key",
  "providers.card.addKey": "Add key",
  "providers.card.models": "Models ({count})",
  "providers.card.removeKey": "Remove key",
  "providers.card.keyPlaceholder": "Paste the {provider} key",
  "providers.card.keyAria": "{provider} API key",
  "providers.card.save": "Save",
  "providers.form.nameRequired": "Give the provider a name.",
  "providers.form.urlRequired": "Enter the provider's endpoint (base URL).",
  "providers.form.modelsRequired": "Enter at least one model ID.",
  "providers.form.createError": "Could not register the provider.",
  "providers.form.aria": "Add AI provider",
  "providers.form.title": "Add provider",
  "providers.form.intro":
    "Pick a compatible service preset or use \"Custom\". Pricing and quota always depend on the chosen provider; registering it here does not make the service free.",
  "providers.form.service": "Service",
  "providers.form.name": "Name",
  "providers.form.namePlaceholder": "e.g. openai",
  "providers.form.endpoint": "Endpoint (base URL)",
  "providers.form.urlPlaceholder":
    "https://api.example.com/v1 or http://localhost:1234/v1",
  "providers.form.urlHint":
    "External services require HTTPS. HTTP is accepted only for local services (localhost / 127.0.0.1), such as LM Studio or llama.cpp.",
  "providers.form.models": "Model IDs",
  "providers.form.modelsPlaceholder":
    "One per line or comma-separated. E.g. gpt-4o, gpt-4o-mini",
  "providers.form.modelsCount": "{count} model(s) recognized.",
  "providers.form.key": "API key (optional)",
  "providers.form.keyPlaceholder": "Paste the provider key here, if required",
  "providers.form.keyHint":
    "The key is stored protected in your system user and never comes back to this screen. Local services usually need no key.",
  "providers.form.submitting": "Registering…",
  "providers.form.submit": "Register provider",
  "providers.page.configTitle": "Settings",
  "providers.page.title": "AI providers",
  "providers.page.intro":
    "Models from other providers used by the router, alongside local models and Ollama Cloud. Keys stay protected in your system user and never come back to this screen.",
  "providers.page.add": "Add provider",
  "providers.page.created":
    "{name}: provider registered. The runtime will apply the configuration on the next startup.",
  "providers.page.ready": "{ready} of {total} providers ready",
  "providers.page.retry": "Try again",
  "providers.page.loading": "Loading providers…",
  "providers.page.empty.intro":
    "You have not registered any external provider yet. Use the \"Add provider\" button above to register a compatible service (such as OpenAI, Groq or OpenRouter) by pasting the API key — no file editing needed.",
  "providers.page.empty.local":
    "Local models and Ollama Cloud (with login) keep working normally, with no provider needed here.",
  "providers.page.changed":
    "{name}: change saved. The changes were saved; the runtime will apply the configuration on the next startup.",

  // WhatsAppGatewayPanel
  "whatsapp.status.connected": "Connected",
  "whatsapp.status.notConfigured": "Not configured",
  "whatsapp.status.failed": "Failed",
  "whatsapp.status.connecting": "Connecting",
  "whatsapp.header.title": "WhatsApp Gateway — Agent Remote Control",
  "whatsapp.header.subtitle":
    "Launch missions, check progress and approve sensitive actions from your phone.",
  "whatsapp.header.connected": "Connected (PASS)",
  "whatsapp.header.refreshAria": "Refresh status",
  "whatsapp.backend.active": "ACTIVE",
  "whatsapp.backend.status": "Status: {status}",
  "whatsapp.backend.evolution.description":
    "Fast open gateway for self-hosted instances.",
  "whatsapp.backend.cloud.description":
    "Official Meta Graph API endpoint for business accounts.",
  "whatsapp.evolution.title": "Connect Evolution API",
  "whatsapp.evolution.intro":
    "Enter your Evolution API server address and key. The credentials are stored securely on the local server (0600) and never displayed again.",
  "whatsapp.evolution.baseUrl": "Server URL",
  "whatsapp.evolution.baseUrlPlaceholder": "https://evolution.yourdomain.com",
  "whatsapp.evolution.instance": "Instance",
  "whatsapp.evolution.instancePlaceholder": "instance-name",
  "whatsapp.evolution.apiKey": "API Key",
  "whatsapp.evolution.apiKeyPlaceholder": "API key",
  "whatsapp.evolution.webhookSecret": "Webhook secret",
  "whatsapp.evolution.webhookSecretPlaceholder":
    "secret to validate the webhook",
  "whatsapp.evolution.save": "Save and connect",
  "whatsapp.evolution.saving": "Saving…",
  "whatsapp.evolution.saved": "Credentials saved. Checking status…",
  "whatsapp.evolution.hint":
    "Tip: point your Evolution API webhook to the Hades endpoint (/api/agent/v1/whatsapp/webhook) using the same secret.",
  "whatsapp.bridge.execution": "Real execution on the local runtime",
  "whatsapp.bridge.off":
    "Configure a backend above (Evolution API or Meta Cloud API) to enable the Command Bridge, human approval and loop protection (DLQ). These features become available only after connecting.",
  "whatsapp.card.security.title": "Security and human approval",
  "whatsapp.card.security.value": "Approval from the phone (APROVAR/REJEITAR)",
  "whatsapp.card.security.note":
    "Sensitive actions blocked until authorized",
  "whatsapp.card.idempotency.title": "Idempotency & DLQ",
  "whatsapp.card.idempotency.value": "fromMe filter + 24h TTL dedupe",
  "whatsapp.card.idempotency.note": "Zero reply loops and active DLQ",
  "whatsapp.allowlist.title": "Allowlist of Owners and Authorized Contacts",
  "whatsapp.allowlist.intro":
    "Only the numbers registered below may interact with the agent. Contacts outside the list are blocked immediately.",
  "whatsapp.allowlist.empty":
    "No contact registered in the allowlist yet. Add your number below to get started.",
  "whatsapp.allowlist.removeAria": "Remove contact",
  "whatsapp.allowlist.phonePlaceholder": "Phone (e.g. 5511999999999)",
  "whatsapp.allowlist.namePlaceholder": "Contact name",
  "whatsapp.allowlist.role.owner": "Owner (full access)",
  "whatsapp.allowlist.role.operator": "Operator (runs missions)",
  "whatsapp.allowlist.role.viewer": "Viewer (read only)",
  "whatsapp.allowlist.add": "Add",

  // AppSidebar
  "appSidebar.header.workspace": "Local workspace",
  "appSidebar.newTask": "New task",
  "appSidebar.search": "Search",
  "appSidebar.section.content": "Content",
  "appSidebar.nav.library": "Library",
  "appSidebar.nav.creations": "Creations",
  "appSidebar.nav.capabilities": "Capabilities",
  "appSidebar.nav.signing": "Signing",
  "appSidebar.section.agents": "Agents",
  "appSidebar.nav.computers": "Computers",
  "appSidebar.nav.agents": "Agents",
  "appSidebar.badge.new": "New",
  "appSidebar.nav.localRuntimeTitle": "Local runtime",
  "appSidebar.nav.automations": "Automations",
  "appSidebar.nav.plugins": "Plugins",
  "appSidebar.section.tools": "Tools",
  "appSidebar.nav.skills": "Skills",
  "appSidebar.nav.company": "Company",
  "appSidebar.nav.tasks": "Tasks",
  "appSidebar.nav.compareModels": "Compare models",
  "appSidebar.section.projects": "Projects",
  "appSidebar.projects.new": "New project",
  "appSidebar.nav.allProjects": "All projects",
  "appSidebar.section.system": "System",
  "appSidebar.nav.settings": "Settings",
  "appSidebar.nav.help": "Help & about",
  "appSidebar.profile.personal": "Personal",
  "appSidebar.profile.credits": "Credits:",
  "appSidebar.profile.creditsUnlimited": "Unlimited (Local)",
  "appSidebar.login.opening": "Opening…",
  "appSidebar.login.action": "Sign in with Ollama account",
  "appSidebar.login.hint":
    "Optional — only for cloud models and web search. Local models work without an account.",
  "appSidebar.profile.account": "Account",
  "appSidebar.profile.personalization": "Personalization",
  "appSidebar.profile.shortcuts": "Keyboard shortcuts",
  "appSidebar.notifications.title": "Notifications",
  "appSidebar.notifications.close": "Close notifications",
  "appSidebar.notifications.loading": "Loading events…",
  "appSidebar.notifications.empty": "No mission events recorded yet.",
  "appSidebar.notifications.listAria": "Recent mission events",
  "appSidebar.notifications.loadFailed": "Could not load notifications.",
  "appSidebar.notifications.invalidStream":
    "The notifications stream returned invalid data.",
  "appSidebar.notifications.streamUnavailable":
    "Live updates unavailable; already loaded events remain visible.",
  "appSidebar.shortcuts.title": "Keyboard shortcuts",
  "appSidebar.shortcuts.newChat": "New quick chat",
  "appSidebar.shortcuts.sendMission": "Send and start mission",
  "appSidebar.shortcuts.close": "Close",
  "appSidebar.signOut.confirm": "Are you sure you want to sign out?",
  "appSidebar.signOut.asEmail": "Sign out of Hades as {email}?",
  "appSidebar.signOut.stay": "Stay signed in",
  "appSidebar.signOut.pending": "Signing out…",
  "appSidebar.signOut.action": "Sign out",
  "appSidebar.signOut.failed": "Could not end the session.",
  "appSidebar.nav.aria": "Main navigation",
  "appSidebar.chatShortcut": "Chat",

  // Backend enum labels (lib/labels.ts)
  "labels.missionState.pending": "Pending",
  "labels.missionState.queued": "Queued",
  "labels.missionState.planning": "Planning",
  "labels.missionState.running": "Running",
  "labels.missionState.observing": "Observing",
  "labels.missionState.recovering": "Recovering",
  "labels.missionState.awaitingApproval": "Awaiting approval",
  "labels.missionState.paused": "Paused",
  "labels.missionState.blocked": "Blocked",
  "labels.missionState.blockedExternal": "Blocked (external dependency)",
  "labels.missionState.completed": "Completed",
  "labels.missionState.failed": "Failed",
  "labels.missionState.cancelled": "Cancelled",
  "labels.missionState.notExecuted": "Not run",
  "labels.missionState.pass": "OK",
  "labels.eventType.missionCreated": "Mission created",
  "labels.eventType.missionPlanned": "Plan generated",
  "labels.eventType.missionRun": "Mission started",
  "labels.eventType.missionRunning": "Running",
  "labels.eventType.missionCompleted": "Mission completed",
  "labels.eventType.missionFailed": "Mission failed",
  "labels.eventType.missionCancelled": "Mission cancelled",
  "labels.eventType.missionRepairAttempt": "Attempting to fix",
  "labels.eventType.missionRepairSucceeded": "Fix applied",
  "labels.eventType.missionRepairFailedTest": "Test still failing",
  "labels.eventType.missionRepairError": "Error while fixing",
  "labels.eventType.missionRepairExhausted": "Fix attempts exhausted",
  "labels.eventType.missionQueueFailed": "Failed to enqueue",
  "labels.eventType.stepStarted": "Step started",
  "labels.eventType.stepSucceeded": "Step completed",
  "labels.eventType.stepFailed": "Step failed",
  "labels.eventType.stepAwaitingApproval": "Step awaiting approval",
  "labels.eventType.stepRetryScheduled": "Retry scheduled",
  "labels.eventType.browserOperator": "Browser",
  "labels.eventType.browserFrame": "Browser frame",
  "labels.eventType.browserApproval": "Browser approval",
  "labels.eventType.gitMergeSucceeded": "Merge completed",
  "labels.eventType.gitMergeRejected": "Merge rejected",
  "labels.stepKind.workspaceRead": "Read file",
  "labels.stepKind.workspaceWrite": "Write file",
  "labels.stepKind.workspacePatch": "Patch file",
  "labels.stepKind.workspaceList": "List files",
  "labels.stepKind.browserOperator": "Operate browser",
  "labels.stepKind.projectTestRun": "Run tests",

  // Connectors page
  "connectors.page.title": "Plugins & Connectors",
  "connectors.page.heading": "Plugins",
  "connectors.page.description":
    "Connect external services, OAuth accounts and MCP servers to expand the agent's capabilities.",
  "connectors.page.manage": "Manage connectors ({count})",
  "connectors.page.create": "Create connector",
  "connectors.categories.all": "All",
  "connectors.categories.productivity": "Productivity",
  "connectors.categories.creativity": "Creativity and media",
  "connectors.categories.business": "Business",
  "connectors.categories.development": "Development",
  "connectors.categories.infra": "Infrastructure",
  "connectors.categories.finance": "Finance",
  "connectors.categories.data": "Data and metrics",
  "connectors.categories.connected": "Connected",
  "connectors.categories.whatsapp": "WhatsApp Gateway",
  "connectors.search.placeholder": "Search connectors and plugins...",
  "connectors.search.aria": "Search connectors",
  "connectors.create.title": "Register connector / custom API",
  "connectors.create.description":
    "Add local or remote REST endpoints or MCP servers for the agent to invoke safely.",
  "connectors.create.nameLabel": "Service name / provider",
  "connectors.create.namePlaceholder": "E.g. My Internal API",
  "connectors.create.baseUrlLabel": "API base URL",
  "connectors.create.tokenEnvLabel":
    "Environment variable with the key (optional)",
  "connectors.create.tokenEnvPlaceholder":
    "E.g. MY_API_KEY (the secret is never exposed)",
  "connectors.create.pathLabel": "Allowed path (prefix)",
  "connectors.create.pathPlaceholder": "E.g. /v1/clients",
  "connectors.create.pathHint":
    "The agent can only call routes under this prefix. Keep the scope as small as possible.",
  "connectors.create.allowWriteLabel":
    "Allow writes (POST/PUT/PATCH/DELETE) on this path.",
  "connectors.create.allowWriteHint":
    "By default the connector is read-only (GET only). Enable writes only if this address really needs to perform changes.",
  "connectors.create.cancel": "Cancel",
  "connectors.create.saving": "Saving...",
  "connectors.create.submit": "Register connector",
  "connectors.create.failedBackend":
    "Could not register the connector with the backend",
  "connectors.create.failed": "Could not register the connector",
  "connectors.list.loading": "Loading plugin catalog...",
  "connectors.list.empty": "No connector found in this category.",
  "connectors.card.active": "Active",
  "connectors.card.oauthTitle":
    "Requires an OAuth app registered by DZ23",
  "connectors.card.oauthRequires": "Requires OAuth configuration",
  "connectors.card.disabled": "Disabled",
  "connectors.card.connect": "Connect",

  // BI / business status panel
  "businessStatus.errors.panel": "Business panel unavailable.",
  "businessStatus.errors.trend": "Trend series unavailable.",
  "businessStatus.errors.record": "Could not record the sample.",
  "businessStatus.errors.export":
    "Could not generate the export right now. Try again.",
  "businessStatus.record.created": "Sample recorded now.",
  "businessStatus.record.existing":
    "There was already a sample in this hour: the series uses one per window and does not duplicate the same instant.",
  "businessStatus.indicators.companies": "Companies",
  "businessStatus.indicators.orders": "Orders",
  "businessStatus.indicators.paid": "Paid",
  "businessStatus.indicators.revenue": "Revenue",
  "businessStatus.indicators.financeIncome": "Finance income",
  "businessStatus.indicators.financeExpense": "Finance expense",
  "businessStatus.indicators.net": "Net",
  "businessStatus.indicators.reconciled": "Reconciled",
  "businessStatus.indicators.campaigns": "Campaigns",
  "businessStatus.indicators.scheduled": "Scheduled",
  "businessStatus.indicators.impressions": "Impressions",
  "businessStatus.indicators.clicks": "Clicks",
  "businessStatus.indicators.proposals": "Proposals",
  "businessStatus.indicators.followUpsOverdue": "Overdue follow-ups",
  "businessStatus.indicators.conversations": "Conversations",
  "businessStatus.indicators.slaViolations": "SLA violations",
  "businessStatus.indicators.incompleteChannels": "Incomplete channels",
  "businessStatus.header.eyebrow": "BI and business status",
  "businessStatus.header.title":
    "Real indicators, with origin and declared gaps",
  "businessStatus.header.leadBefore":
    "Each number comes from an internal module and says where it came from. What this panel",
  "businessStatus.header.leadStrong": "does not measure",
  "businessStatus.header.leadAfter":
    "appears just below, with the reason: zero without measurement is not an answer.",
  "businessStatus.actions.export": "Export CSV",
  "businessStatus.actions.refreshing": "Refreshing…",
  "businessStatus.actions.refresh": "Refresh panel",
  "businessStatus.exportNote.before": "The export is a CSV with a single schema",
  "businessStatus.exportNote.afterSchema": "and carries",
  "businessStatus.exportNote.gaps": "the gaps",
  "businessStatus.exportNote.conjunction": "and the",
  "businessStatus.exportNote.origin": "origin of each number",
  "businessStatus.exportNote.after":
    ": outside the product the screen does not come along.",
  "businessStatus.export.done":
    "{file} generated: the file includes what is NOT measured and the origin of each number.",
  "businessStatus.empty.text":
    "No data for this company yet: the indicators appear once an order, entry, campaign, proposal or conversation is recorded.",
  "businessStatus.trend.title": "Trend series",
  "businessStatus.trend.record": "Record sample now",
  "businessStatus.trend.unavailable": "Trend series unavailable.",
  "businessStatus.trend.samplesOne": "{count} sample · retention cap {max}",
  "businessStatus.trend.samplesMany": "{count} samples · retention cap {max}",
  "businessStatus.trend.discardedOne": "{count} outside the window",
  "businessStatus.trend.discardedMany": "{count} outside the window",
  "businessStatus.trend.noBase": "No trend: {reason}",
  "businessStatus.trend.interval": "Compared interval: {interval}",
  "businessStatus.chart.title": "Series chart",
  "businessStatus.chart.metricLabel": "Chart metric",
  "businessStatus.naoMedido.title": "What this panel does NOT measure",
  "businessStatus.sources.title": "Origin of each number",
  "businessStatus.sources.external": "External publication:",

  // Codex Desktop models (CodexDesktopModelsSettings)
  "codexModels.status.cloudOff": "Cloud models disabled",
  "codexModels.status.signIn": "Sign-in required",
  "codexModels.status.upgradePlan": "{plan} plan required",
  "codexModels.status.upgradeRequired": "Plan upgrade required",
  "codexModels.status.unavailable": "Access unavailable",
  "codexModels.status.notInstalled": "Not installed",
  "codexModels.search.placeholder": "Search models…",
  "codexModels.search.aria": "Search ChatGPT models",
  "codexModels.search.empty": "No models found",
  "codexModels.error.version": "ChatGPT model settings are not available in this version of Ollama.",
  "codexModels.error.load": "Could not load the ChatGPT model settings.",
  "codexModels.error.macosOnly": "ChatGPT model settings are only available on macOS for now.",
  "codexModels.error.selectOne": "Choose at least one model for ChatGPT.",
  "codexModels.confirm.restartToUpdate": "Restart ChatGPT to update the Ollama models? Any running task will be interrupted.",
  "codexModels.error.openTimeout": "ChatGPT is taking longer than expected to open. Try again.",
  "codexModels.error.apply": "Could not apply the ChatGPT models.",
  "codexModels.error.reset": "Could not restore the ChatGPT models.",
  "codexModels.pick.upToOne": "Choose up to {count} Ollama model to use in ChatGPT.",
  "codexModels.pick.upToMany": "Choose up to {count} Ollama models to use in ChatGPT.",
  "codexModels.action.saveAndRestart": "Save and restart ChatGPT",
  "codexModels.action.restart": "Restart ChatGPT",
  "codexModels.action.saveAndStart": "Save and start ChatGPT",
  "codexModels.action.start": "Start ChatGPT",
  "codexModels.action.restarting": "Restarting…",
  "codexModels.action.starting": "Starting…",
  "codexModels.notice.macosOnly": "Only available on macOS for now. ChatGPT model configuration depends on the Ollama app for macOS.",
  "codexModels.picker.addAria": "Add a ChatGPT model",
  "codexModels.picker.chooseAria": "Choose ChatGPT models",
  "codexModels.picker.removeAria": "Remove {model}",
  "codexModels.picker.loading": "Loading models…",
  "codexModels.picker.select": "Select models",

  // Claude Desktop models (ClaudeDesktopModelsSettings)
  "claudeModels.picker.placeholder": "Select a model",
  "claudeModels.picker.routeAria": "Ollama model for {route}",
  "claudeModels.search.placeholder": "Search models…",
  "claudeModels.search.aria": "Search models for {route}",
  "claudeModels.search.empty": "No models found",
  "claudeModels.error.readStatus": "Could not read the Claude connection status.",
  "claudeModels.error.loadModels": "Could not load your models.",
  "claudeModels.confirm.restartDesktop": "Restart Claude Desktop? Any running task will be interrupted.",
  "claudeModels.confirm.restartAutoMode": "Restart Claude to change automatic mode? Any running task will be interrupted.",
  "claudeModels.error.macosRouting": "Claude model routing is only available on macOS for now.",
  "claudeModels.error.selectOne": "Choose at least one Ollama model for Claude.",
  "claudeModels.error.invalidMapping": "Choose models available to your account and device.",
  "claudeModels.error.apply": "Could not apply the Claude models.",
  "claudeModels.error.autoModeUnavailable": "Automatic mode is only available on macOS for now.",
  "claudeModels.error.autoMode": "Could not update Claude's automatic mode.",
  "claudeModels.error.reset": "Could not restore the Claude models.",
  "claudeModels.autoMode.draftHint": "Start or restart Claude to apply the changes before switching automatic mode.",
  "claudeModels.autoMode.description": "Let Claude decide when to ask for confirmation before making changes.",
  "claudeModels.autoMode.selectCloud": "Select an Ollama.com cloud model to use automatic mode.",
  "claudeModels.autoMode.selectOne": "Select one of these models to use automatic mode: {models}.",
  "claudeModels.autoMode.needCloud": "Automatic mode needs a cloud model available in your Ollama.com account.",
  "claudeModels.autoMode.toggle": "Enable automatic mode",
  "claudeModels.guidance.restart": "Restarting Claude will interrupt any running task.",
  "claudeModels.section.aria": "Claude settings",
  "claudeModels.section.heading": "Apps",
  "claudeModels.description": "Choose which Ollama model Claude uses for each option.",
  "claudeModels.action.restoring": "Restoring…",
  "claudeModels.action.restarting": "Restarting…",
  "claudeModels.action.starting": "Starting…",
  "claudeModels.action.restart": "Restart Claude",
  "claudeModels.action.start": "Start Claude",
  "claudeModels.notice.macosOnly": "Only available on macOS for now. Claude model routing depends on the Ollama app for macOS.",
  "claudeModels.list.two": "{first} or {last}",
  "claudeModels.list.many": "{items} or {last}",

  // Automation flow (lib/flowGraph)
  "flowGraph.connect.missingNodes": "Source or target node does not exist.",
  "flowGraph.connect.self": "A node cannot connect to itself.",
  "flowGraph.connect.triggerTarget": "A trigger cannot receive incoming connections.",
  "flowGraph.connect.duplicate": "That connection already exists.",
  "flowGraph.connect.cycle": "That connection would create a cycle in the flow.",
  "flowGraph.validate.empty": "The flow is empty: add a trigger to get started.",
  "flowGraph.validate.noTrigger": "The flow needs at least one input trigger.",
  "flowGraph.validate.cycle": "The flow has a cycle: remove the connection that loops back to an earlier node.",
  "flowGraph.validate.danglingNode": "Node \"{label}\" is dangling (no incoming connection).",
  "flowGraph.steps.empty": "The flow is empty: nothing to persist as a step graph.",
  "flowGraph.steps.singleTrigger": "The step graph requires exactly one trigger.",
  "flowGraph.steps.unsupportedNode": "Node \"{label}\" ({type}) cannot run as a persisted step yet; the flow runs as a textual plan.",
  "flowGraph.steps.tooMany": "The step graph accepts at most {max} steps.",
  "flowGraph.steps.cycle": "The flow has a cycle and cannot be persisted.",
  "flowGraph.steps.triggerDepends": "Trigger \"{label}\" cannot depend on another step.",
  "flowGraph.steps.actionNoInput": "Action \"{label}\" needs an incoming connection.",
  "flowGraph.steps.missingObjective": "Fill in the objective of the \"{label}\" action.",
  "flowGraph.describe.missionMissingObjective": "Fill in the objective of the \"Run mission\" action.",
  "flowGraph.describe.httpMissingUrl": "Fill in the URL of the \"HTTP request\" action.",
  "flowGraph.describe.connectorMissing": "Choose the connector of the \"Call connector\" action.",
  "flowGraph.compile.singleTrigger": "Scheduled execution accepts exactly one trigger per flow.",
  "flowGraph.compile.needsMission": "The flow needs a \"Run mission\" action to run on a schedule.",
  "flowGraph.compile.webhookSecret": "Provide the environment variable holding the webhook trigger secret.",

  // Common (shared confirmation label)
  "common.restart": "Restart",
  // System health (HealthCenter) + home route title
  "health.title": "System health",
  "health.loading": "Checking services…",
  "health.error": "Could not read the backend diagnostics.",
  "health.retry": "Try again",
  "health.checkedAt": "Diagnostics received from the backend at {at}.",
  "health.now": "now",
  "health.details": "Details",
  "health.detailSuffix": ". The corrective action depends on the displayed configuration and is not run automatically.",
  "health.status.ok": "Operational",
  "health.status.attention": "Attention",
  "health.status.unavailable": "Unavailable",
  "health.subsystem.agent": "Mission engine",
  "health.subsystem.store": "Persistence",
  "health.subsystem.queue": "Execution queue",
  "health.subsystem.sandbox": "Isolation",
  "route.home.title": "Home",
  "backend.offline.message": "The local backend has not responded yet. It may be starting — wait a few seconds and try again.",

  // New task (NewTaskPage)
  "newTask.title": "New task",
  "newTask.back": "Back to Tasks",
  "newTask.eyebrow": "Agentic mission",
  "newTask.description":
    "The task is created in the local runtime with workspace read access and no automatic execution. You follow the plan, approvals and artifacts under Tasks.",
  "newTask.objective.label": "Task objective",
  "newTask.objective.placeholder":
    "E.g.: Review the README and list what is missing to run the project locally",
  "newTask.objective.hint": "{count}/{max} characters · Ctrl+Enter to create",
  "newTask.validation.empty": "Describe the task objective before creating it.",
  "newTask.error.title": "The task was not created.",
  "newTask.error.create": "Could not create the task.",
  "newTask.submit": "Create task",
  "newTask.submit.pending": "Creating task…",
  "common.cancel": "Cancel",
};
