package agent

import (
	"sort"
	"strings"
)

type ConnectorCatalogEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Kind        string   `json:"kind"`
	Description string   `json:"description"`
	Auth        string   `json:"auth"`
	Source      string   `json:"source"`
	Status      string   `json:"status"`
	Scopes      []string `json:"scopes,omitempty"`
	// Quick connect: services that accept an API key in a header can be
	// connected from the desktop app with a pasted key. APISelfHosted means
	// the user supplies the base URL of their own instance.
	APIBaseURL    string `json:"api_base_url,omitempty"`
	APIAuthHeader string `json:"api_auth_header,omitempty"`
	APIAuthScheme string `json:"api_auth_scheme,omitempty"`
	APISelfHosted bool   `json:"api_self_hosted,omitempty"`
	QuickConnect  bool   `json:"quick_connect,omitempty"`
}

type quickConnect struct {
	base       string
	header     string // default Authorization
	scheme     string // default Bearer for Authorization, bare token otherwise; "raw" forces bare
	selfHosted bool
}

// quickConnects lists services whose API key can be pasted to connect them.
var quickConnects = map[string]quickConnect{
	"airtable":        {base: "https://api.airtable.com/v0"},
	"algolia":         {selfHosted: true, header: "X-Algolia-API-Key"},
	"appwrite":        {base: "https://cloud.appwrite.io/v1", header: "X-Appwrite-Key"},
	"asaas":           {base: "https://api.asaas.com/v3", header: "access_token"},
	"asana":           {base: "https://app.asana.com/api/1.0"},
	"auth0":           {selfHosted: true},
	"betterstack":     {base: "https://uptime.betterstack.com/api/v2"},
	"bitbucket":       {base: "https://api.bitbucket.org/2.0"},
	"brevo":           {base: "https://api.brevo.com/v3", header: "api-key"},
	"calendly":        {base: "https://api.calendly.com"},
	"circleci":        {base: "https://circleci.com/api/v2", header: "Circle-Token"},
	"clerk":           {base: "https://api.clerk.com/v1"},
	"cloudflare":      {base: "https://api.cloudflare.com/client/v4"},
	"contentful":      {base: "https://api.contentful.com"},
	"coolify":         {selfHosted: true},
	"datadog":         {base: "https://api.datadoghq.com/api", header: "DD-API-KEY"},
	"deepgram":        {base: "https://api.deepgram.com/v1", scheme: "Token"},
	"digitalocean":    {base: "https://api.digitalocean.com/v2"},
	"dropbox":         {base: "https://api.dropboxapi.com/2"},
	"elevenlabs":      {base: "https://api.elevenlabs.io/v1", header: "xi-api-key"},
	"expo":            {base: "https://api.expo.dev"},
	"figma":           {base: "https://api.figma.com/v1", header: "X-Figma-Token"},
	"flyio":           {base: "https://api.machines.dev/v1"},
	"github":          {base: "https://api.github.com"},
	"gitlab":          {base: "https://gitlab.com/api/v4"},
	"grafana":         {selfHosted: true},
	"hetzner":         {base: "https://api.hetzner.cloud/v1"},
	"hubspot":         {base: "https://api.hubapi.com"},
	"huggingface-hub": {base: "https://huggingface.co/api"},
	"intercom":        {base: "https://api.intercom.io"},
	"lemon-squeezy":   {base: "https://api.lemonsqueezy.com/v1"},
	"linear":          {base: "https://api.linear.app", scheme: "raw"},
	"meilisearch":     {selfHosted: true},
	"mercado-pago":    {base: "https://api.mercadopago.com"},
	"neon":            {base: "https://console.neon.tech/api/v2"},
	"netlify":         {base: "https://api.netlify.com/api/v1"},
	"novu":            {base: "https://api.novu.co/v1", scheme: "ApiKey"},
	"paddle":          {base: "https://api.paddle.com"},
	"pagseguro":       {base: "https://api.pagseguro.com"},
	"pinecone":        {base: "https://api.pinecone.io", header: "Api-Key"},
	"pocketbase":      {selfHosted: true},
	"posthog":         {base: "https://us.posthog.com/api"},
	"postmark":        {base: "https://api.postmarkapp.com", header: "X-Postmark-Server-Token"},
	"qdrant":          {selfHosted: true, header: "api-key"},
	"railway":         {base: "https://backboard.railway.app/graphql/v2"},
	"render":          {base: "https://api.render.com/v1"},
	"replicate":       {base: "https://api.replicate.com/v1"},
	"resend":          {base: "https://api.resend.com"},
	"sanity":          {base: "https://api.sanity.io"},
	"sendgrid":        {base: "https://api.sendgrid.com"},
	"sentry":          {base: "https://sentry.io/api/0"},
	"shopify":         {selfHosted: true, header: "X-Shopify-Access-Token"},
	"strapi":          {selfHosted: true},
	"stripe":          {base: "https://api.stripe.com"},
	"supabase":        {base: "https://api.supabase.com/v1"},
	"typeform":        {base: "https://api.typeform.com"},
	"vercel":          {base: "https://api.vercel.com"},
	"whatsapp":        {base: "https://graph.facebook.com/v21.0"},
	"openai":          {base: "https://api.openai.com/v1"},
	"openrouter":      {base: "https://openrouter.ai/api/v1"},
	"groq":            {base: "https://api.groq.com/openai/v1"},
	"together":        {base: "https://api.together.xyz/v1"},
	"mistral":         {base: "https://api.mistral.ai/v1"},
	"cohere":          {base: "https://api.cohere.com/v1"},
	"clickup":         {base: "https://api.clickup.com/api/v2", scheme: "raw"},
	"vultr":           {base: "https://api.vultr.com/v2"},
	"linode":          {base: "https://api.linode.com/v4"},
}

// quickConnectOperations is the least-privilege capability registry for API keys.
// Quick-connect is read-only; mutations require explicit advanced connector setup.
var quickConnectOperations = map[string][]ConnectorOperation{
	"airtable":        readOnlyConnectorPaths("/meta/bases"),
	"algolia":         readOnlyConnectorPaths("/1/indexes"),
	"appwrite":        readOnlyConnectorPaths("/account"),
	"asaas":           readOnlyConnectorPaths("/customers"),
	"asana":           readOnlyConnectorPaths("/users/me"),
	"betterstack":     readOnlyConnectorPaths("/monitors"),
	"bitbucket":       readOnlyConnectorPaths("/user"),
	"brevo":           readOnlyConnectorPaths("/account"),
	"calendly":        readOnlyConnectorPaths("/users/me"),
	"circleci":        readOnlyConnectorPaths("/me"),
	"clerk":           readOnlyConnectorPaths("/users"),
	"cloudflare":      readOnlyConnectorPaths("/user/tokens/verify"),
	"contentful":      readOnlyConnectorPaths("/spaces"),
	"datadog":         readOnlyConnectorPaths("/v1/validate"),
	"deepgram":        readOnlyConnectorPaths("/projects"),
	"digitalocean":    readOnlyConnectorPaths("/account"),
	"elevenlabs":      readOnlyConnectorPaths("/user"),
	"figma":           readOnlyConnectorPaths("/me"),
	"flyio":           readOnlyConnectorPaths("/apps"),
	"github":          readOnlyConnectorPaths("/user", "/repos"),
	"gitlab":          readOnlyConnectorPaths("/user", "/projects"),
	"grafana":         readOnlyConnectorPaths("/api/org"),
	"hetzner":         readOnlyConnectorPaths("/servers"),
	"hubspot":         readOnlyConnectorPaths("/account-info/v3/details"),
	"huggingface-hub": readOnlyConnectorPaths("/whoami-v2"),
	"intercom":        readOnlyConnectorPaths("/me"),
	"lemon-squeezy":   readOnlyConnectorPaths("/users/me"),
	"meilisearch":     readOnlyConnectorPaths("/indexes"),
	"mercado-pago":    readOnlyConnectorPaths("/users/me"),
	"neon":            readOnlyConnectorPaths("/projects"),
	"netlify":         readOnlyConnectorPaths("/sites"),
	"paddle":          readOnlyConnectorPaths("/products"),
	"pinecone":        readOnlyConnectorPaths("/indexes"),
	"posthog":         readOnlyConnectorPaths("/projects"),
	"postmark":        readOnlyConnectorPaths("/server"),
	"qdrant":          readOnlyConnectorPaths("/collections"),
	"render":          readOnlyConnectorPaths("/services"),
	"replicate":       readOnlyConnectorPaths("/account"),
	"resend":          readOnlyConnectorPaths("/domains"),
	"sanity":          readOnlyConnectorPaths("/v1/projects"),
	"sendgrid":        readOnlyConnectorPaths("/v3/user/profile"),
	"sentry":          readOnlyConnectorPaths("/organizations"),
	"shopify":         readOnlyConnectorPaths("/admin/api"),
	"stripe":          readOnlyConnectorPaths("/v1/account"),
	"supabase":        readOnlyConnectorPaths("/projects"),
	"typeform":        readOnlyConnectorPaths("/forms"),
	"vercel":          readOnlyConnectorPaths("/v2/user"),
	"openai":          readOnlyConnectorPaths("/models"),
	"openrouter":      readOnlyConnectorPaths("/models"),
	"groq":            readOnlyConnectorPaths("/models"),
	"together":        readOnlyConnectorPaths("/models"),
	"mistral":         readOnlyConnectorPaths("/models"),
	"cohere":          readOnlyConnectorPaths("/models"),
	"clickup":         readOnlyConnectorPaths("/user"),
	"vultr":           readOnlyConnectorPaths("/account"),
	"linode":          readOnlyConnectorPaths("/account"),
}

func readOnlyConnectorPaths(paths ...string) []ConnectorOperation {
	return []ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: paths}}
}

// QuickConnectOperations returns a defensive copy of the catalog-owned policy.
func QuickConnectOperations(id string) []ConnectorOperation {
	operations := quickConnectOperations[strings.TrimSpace(id)]
	if len(operations) == 0 {
		return nil
	}
	result := make([]ConnectorOperation, len(operations))
	for i, operation := range operations {
		result[i] = ConnectorOperation{
			Name:         operation.Name,
			Methods:      append([]string(nil), operation.Methods...),
			PathPrefixes: append([]string(nil), operation.PathPrefixes...),
		}
	}
	return result
}

// ConnectorTokenEnv is the credential variable used for a quick-connected
// catalog entry.
func ConnectorTokenEnv(id string) string {
	return "OLLAMA_CONNECTOR_" + strings.ToUpper(strings.ReplaceAll(id, "-", "_")) + "_TOKEN"
}

// ConnectorCatalog is intentionally a capability catalog, not a claim that every
// provider is connected. Entries requiring an operator credential remain explicit.
func ConnectorCatalog() []ConnectorCatalogEntry {
	entries := []ConnectorCatalogEntry{
		{ID: "github", Name: "GitHub", Category: "Desenvolvimento", Kind: "app", Description: "Repositórios, branches, issues, pull requests, Actions e releases.", Auth: "oauth", Source: "built-in/example", Status: "available", Scopes: []string{"repositories", "issues", "pull_requests", "actions"}},
		{ID: "google-workspace", Name: "Google Workspace", Category: "Produtividade", Kind: "app", Description: "Gmail, Drive, Docs, Sheets, Calendar e contatos.", Auth: "oauth", Source: "built-in", Status: "available", Scopes: []string{"gmail", "drive", "docs", "sheets", "calendar"}},
		{ID: "composio", Name: "Composio", Category: "Automação", Kind: "mcp", Description: "Gateway MCP para aplicativos autorizados pelo operador, com descoberta de ferramentas.", Auth: "oauth_or_api_key", Source: "mcp", Status: "available", Scopes: []string{"tool_discovery", "oauth", "external_actions"}},
		{ID: "slack", Name: "Slack", Category: "Comunicação", Kind: "app", Description: "Mensagens, canais, threads e notificações com aprovação para efeitos externos.", Auth: "oauth", Source: "built-in/example", Status: "available", Scopes: []string{"messages", "channels"}},
		{ID: "discord", Name: "Discord", Category: "Comunicação", Kind: "custom_api", Description: "Bots, canais e mensagens via aplicação Discord ou Composio.", Auth: "bot_or_oauth", Source: "custom_api/mcp", Status: "operator_setup_required", Scopes: []string{"messages", "channels"}},
		{ID: "whatsapp", Name: "WhatsApp Business", Category: "Comunicação", Kind: "custom_api", Description: "Mensagens transacionais pela WhatsApp Cloud API/Meta Business.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"messages", "templates"}},
		{ID: "notion", Name: "Notion", Category: "Produtividade", Kind: "app", Description: "Páginas, databases, blocos e conhecimento operacional.", Auth: "oauth_or_api_key", Source: "built-in", Status: "available", Scopes: []string{"pages", "databases"}},
		{ID: "jira", Name: "Jira", Category: "Desenvolvimento", Kind: "custom_api", Description: "Projetos, issues, sprints e workflow de engenharia.", Auth: "oauth_or_api_key", Source: "custom_api/mcp", Status: "operator_setup_required", Scopes: []string{"issues", "projects"}},
		{ID: "hubspot", Name: "HubSpot", Category: "CRM e Vendas", Kind: "app", Description: "CRM, contatos, empresas, negócios e automações comerciais.", Auth: "oauth", Source: "built-in", Status: "available", Scopes: []string{"crm", "contacts"}},
		{ID: "stripe", Name: "Stripe", Category: "Pagamentos", Kind: "app", Description: "Clientes, produtos, invoices e webhooks; movimentações sensíveis permanecem approval-gated.", Auth: "oauth_or_api_key", Source: "built-in", Status: "available", Scopes: []string{"customers", "products", "invoices", "webhooks"}},
		{ID: "woovi-openpix", Name: "Woovi / OpenPix", Category: "Pagamentos", Kind: "custom_api", Description: "Cobranças Pix, QR Code, conciliação e webhooks de pagamento em tempo real.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"charges", "webhooks", "transactions"}},
		{ID: "fiscal-invoicing", Name: "Emissão fiscal NF-e/NFS-e", Category: "Fiscal", Kind: "custom_api", Description: "Contrato para emissão, consulta, cancelamento e armazenamento de XML/DANFE por provedor fiscal homologado.", Auth: "api_key_or_certificate", Source: "custom_api", Status: "provider_selection_required", Scopes: []string{"nfe", "nfse", "xml", "danfe", "webhooks"}},
		{ID: "fiscal-ai", Name: "Fiscal AI", Category: "Dados financeiros", Kind: "mcp", Description: "Dados financeiros e análises de empresas; não é emissor de NF-e/NFS-e.", Auth: "oauth_or_api_key", Source: "built-in", Status: "available", Scopes: []string{"financial_data", "filings"}},
		{ID: "instagram", Name: "Instagram", Category: "Marketing", Kind: "app", Description: "Contas profissionais, conteúdo e métricas via permissões da Meta.", Auth: "oauth", Source: "built-in", Status: "available", Scopes: []string{"content", "insights"}},
		{ID: "meta-ads", Name: "Meta Ads Manager", Category: "Marketing", Kind: "app", Description: "Campanhas, anúncios, orçamento e métricas com approval para publicação e gasto.", Auth: "oauth", Source: "built-in", Status: "available", Scopes: []string{"campaigns", "insights", "ads"}},
		{ID: "tiktok-business", Name: "TikTok for Business", Category: "Marketing", Kind: "app", Description: "Campanhas e métricas TikTok; TikTok Shop depende de elegibilidade e APIs da conta.", Auth: "oauth", Source: "built-in", Status: "available", Scopes: []string{"campaigns", "insights"}},
		{ID: "shopify", Name: "Shopify", Category: "Comércio", Kind: "app", Description: "Produtos, inventário, pedidos e fulfillment com aprovação antes de efeitos comerciais.", Auth: "oauth_or_api_key", Source: "built-in", Status: "available", Scopes: []string{"products", "inventory", "orders"}},
		{ID: "woocommerce", Name: "WooCommerce", Category: "Comércio", Kind: "custom_api", Description: "Catálogo, pedidos e estoque via REST API do operador.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"products", "orders", "inventory"}},
		{ID: "mercado-livre", Name: "Mercado Livre", Category: "Comércio", Kind: "custom_api", Description: "Catálogo, pedidos, anúncios e logística conforme permissões da conta.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"catalog", "orders", "shipping"}},
		{ID: "amazon-seller", Name: "Amazon Seller", Category: "Comércio", Kind: "custom_api", Description: "Produtos, pedidos e fulfillment pela Selling Partner API.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"catalog", "orders", "inventory"}},
		{ID: "vercel", Name: "Vercel", Category: "Deploy", Kind: "app", Description: "Projetos, deployments, domínios e logs de publicação.", Auth: "oauth_or_api_key", Source: "built-in", Status: "available", Scopes: []string{"projects", "deployments", "domains"}},
		{ID: "netlify", Name: "Netlify", Category: "Deploy", Kind: "app", Description: "Sites, deploys, domains e logs de publicação.", Auth: "oauth_or_api_key", Source: "built-in", Status: "available", Scopes: []string{"sites", "deploys", "domains"}},
		{ID: "cloudflare", Name: "Cloudflare", Category: "Deploy", Kind: "app", Description: "Workers, Pages, DNS e configurações de edge com escopo explícito.", Auth: "oauth_or_api_key", Source: "built-in", Status: "available", Scopes: []string{"pages", "workers", "dns"}},
		{ID: "zapier", Name: "Zapier", Category: "Automação", Kind: "mcp", Description: "Ações e workflows externos através de conexão autorizada.", Auth: "oauth_or_api_key", Source: "built-in", Status: "available", Scopes: []string{"actions", "triggers"}},
		{ID: "n8n", Name: "n8n", Category: "Automação", Kind: "mcp", Description: "Workflows self-hosted ou cloud para integrações operacionais.", Auth: "url_or_api_key", Source: "built-in", Status: "available", Scopes: []string{"workflows", "executions"}},
		{ID: "pinterest", Name: "Pinterest", Category: "Marketing", Kind: "custom_api", Description: "Pins, boards, catálogo e métricas de anúncios; publicação exige approval.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"pins", "boards", "ads"}},
		{ID: "youtube", Name: "YouTube", Category: "Marketing", Kind: "custom_api", Description: "Canais, vídeos, playlists, comentários e métricas; upload exige approval.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"videos", "analytics"}},
		{ID: "linkedin", Name: "LinkedIn", Category: "Marketing", Kind: "custom_api", Description: "Páginas de empresa, posts e métricas; publicação exige approval.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"pages", "posts"}},
		{ID: "x-twitter", Name: "X (Twitter)", Category: "Marketing", Kind: "custom_api", Description: "Posts, menções e métricas via API do X; publicação exige approval.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"posts", "mentions"}},
		{ID: "facebook-pages", Name: "Facebook Pages", Category: "Marketing", Kind: "custom_api", Description: "Páginas, posts, mensagens e insights via permissões da Meta.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"pages", "posts", "insights"}},
		{ID: "telegram", Name: "Telegram", Category: "Comunicação", Kind: "custom_api", Description: "Bot para receber tarefas e enviar resultados em chats autorizados.", Auth: "bot_token", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"messages"}},
		{ID: "gmail", Name: "Gmail", Category: "Comunicação", Kind: "custom_api", Description: "Ler, buscar e redigir e-mails; envio exige approval.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"read", "draft", "send"}},
		{ID: "google-drive", Name: "Google Drive", Category: "Produtividade", Kind: "custom_api", Description: "Arquivos, pastas e compartilhamento com escopo por pasta.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"files"}},
		{ID: "google-calendar", Name: "Google Agenda", Category: "Produtividade", Kind: "custom_api", Description: "Eventos, disponibilidade e convites; criar convite exige approval.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"events"}},
		{ID: "google-analytics", Name: "Google Analytics", Category: "Dados e métricas", Kind: "custom_api", Description: "Relatórios GA4 de tráfego, conversão e público.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"reports"}},
		{ID: "google-ads", Name: "Google Ads", Category: "Marketing", Kind: "custom_api", Description: "Campanhas, orçamento e métricas; gasto e publicação exigem approval.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"campaigns", "reports"}},
		{ID: "outlook", Name: "Outlook / Microsoft 365", Category: "Comunicação", Kind: "custom_api", Description: "E-mail, calendário e contatos via Microsoft Graph; envio exige approval.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"mail", "calendar"}},
		{ID: "trello", Name: "Trello", Category: "Produtividade", Kind: "custom_api", Description: "Quadros, listas e cartões.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"boards", "cards"}},
		{ID: "asana", Name: "Asana", Category: "Produtividade", Kind: "custom_api", Description: "Projetos, tarefas e responsáveis.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"projects", "tasks"}},
		{ID: "linear", Name: "Linear", Category: "Desenvolvimento", Kind: "custom_api", Description: "Issues, ciclos e projetos de engenharia.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"issues", "projects"}},
		{ID: "gitlab", Name: "GitLab", Category: "Desenvolvimento", Kind: "custom_api", Description: "Repositórios, merge requests, issues e pipelines.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"repos", "merge_requests", "pipelines"}},
		{ID: "airtable", Name: "Airtable", Category: "Dados e métricas", Kind: "custom_api", Description: "Bases, tabelas e registros.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"bases", "records"}},
		{ID: "dropbox", Name: "Dropbox", Category: "Produtividade", Kind: "custom_api", Description: "Arquivos e pastas com links de compartilhamento.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"files"}},
		{ID: "figma", Name: "Figma", Category: "Design", Kind: "custom_api", Description: "Arquivos, frames, componentes e comentários de design.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"files", "comments"}},
		{ID: "canva", Name: "Canva", Category: "Design", Kind: "custom_api", Description: "Designs, modelos e exportação de artes.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"designs"}},
		{ID: "mailchimp", Name: "Mailchimp", Category: "Marketing", Kind: "custom_api", Description: "Audiências, campanhas de e-mail e relatórios; envio exige approval.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"audiences", "campaigns"}},
		{ID: "salesforce", Name: "Salesforce", Category: "CRM e Vendas", Kind: "custom_api", Description: "Contas, contatos, oportunidades e relatórios.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"accounts", "opportunities"}},
		{ID: "zoom", Name: "Zoom", Category: "Comunicação", Kind: "custom_api", Description: "Reuniões, gravações e transcrições.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"meetings", "recordings"}},
		{ID: "paypal", Name: "PayPal", Category: "Pagamentos", Kind: "custom_api", Description: "Pedidos, pagamentos e reembolsos; movimentações exigem approval.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"orders", "payments"}},
		{ID: "mercado-pago", Name: "Mercado Pago", Category: "Pagamentos", Kind: "custom_api", Description: "Cobranças, Pix, links de pagamento e conciliação; movimentações exigem approval.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"payments", "pix"}},
		{ID: "supabase", Name: "Supabase", Category: "Desenvolvimento", Kind: "custom_api", Description: "Projetos, banco Postgres, storage e edge functions.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"projects", "database"}},
		{ID: "wordpress", Name: "WordPress", Category: "Marketing", Kind: "custom_api", Description: "Posts, páginas e mídia via REST API; publicação exige approval.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"posts", "media"}},
		{ID: "calendly", Name: "Calendly", Category: "Produtividade", Kind: "custom_api", Description: "Tipos de evento, agenda e convidados.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"events"}},
		{ID: "typeform", Name: "Typeform", Category: "Dados e métricas", Kind: "custom_api", Description: "Formulários e respostas.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"forms", "responses"}},
		{ID: "zendesk", Name: "Zendesk", Category: "Suporte", Kind: "custom_api", Description: "Tickets, usuários e macros de atendimento.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"tickets"}},
		{ID: "intercom", Name: "Intercom", Category: "Suporte", Kind: "custom_api", Description: "Conversas, contatos e artigos de ajuda.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"conversations"}},
		{ID: "twilio", Name: "Twilio", Category: "Comunicação", Kind: "custom_api", Description: "SMS, voz e WhatsApp via Twilio; envio exige approval.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"messages", "calls"}},
		{ID: "resend", Name: "Resend", Category: "E-mail", Kind: "custom_api", Description: "E-mail transacional por API (React Email, domínios, webhooks); envio exige approval.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"emails", "domains"}},
		{ID: "sendgrid", Name: "SendGrid", Category: "E-mail", Kind: "custom_api", Description: "E-mail transacional e marketing da Twilio; envio exige approval.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"mail", "templates"}},
		{ID: "postmark", Name: "Postmark", Category: "E-mail", Kind: "custom_api", Description: "E-mail transacional com alta entregabilidade e templates.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"email", "templates"}},
		{ID: "brevo", Name: "Brevo", Category: "E-mail", Kind: "custom_api", Description: "E-mail, SMS e automações de marketing (antigo Sendinblue).", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"email", "contacts"}},
		{ID: "firebase", Name: "Firebase", Category: "Backend", Kind: "custom_api", Description: "Auth, Firestore, Storage, Hosting e Functions do Google.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"projects", "firestore"}},
		{ID: "appwrite", Name: "Appwrite", Category: "Backend", Kind: "custom_api", Description: "Backend open source: auth, banco, storage e functions.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"databases", "storage"}},
		{ID: "pocketbase", Name: "PocketBase", Category: "Backend", Kind: "custom_api", Description: "Backend em um binário: banco SQLite, auth e realtime.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"collections"}},
		{ID: "postgresql", Name: "PostgreSQL", Category: "Banco de dados", Kind: "custom_api", Description: "Banco Postgres por URL de conexão; escrita exige approval.", Auth: "connection_url", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"query"}},
		{ID: "mongodb", Name: "MongoDB Atlas", Category: "Banco de dados", Kind: "custom_api", Description: "Clusters, coleções e consultas no MongoDB Atlas.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"clusters", "collections"}},
		{ID: "neon", Name: "Neon", Category: "Banco de dados", Kind: "custom_api", Description: "Postgres serverless com branches de banco.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"projects", "branches"}},
		{ID: "planetscale", Name: "PlanetScale", Category: "Banco de dados", Kind: "custom_api", Description: "MySQL e Postgres gerenciados com branches.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"databases", "branches"}},
		{ID: "redis", Name: "Redis", Category: "Banco de dados", Kind: "custom_api", Description: "Cache e filas Redis por URL de conexão.", Auth: "connection_url", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"keys"}},
		{ID: "upstash", Name: "Upstash", Category: "Banco de dados", Kind: "custom_api", Description: "Redis, Kafka e QStash serverless por HTTP.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"redis", "qstash"}},
		{ID: "prisma", Name: "Prisma", Category: "Banco de dados", Kind: "custom_api", Description: "Prisma Postgres, Accelerate e migrações do schema.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"databases"}},
		{ID: "qdrant", Name: "Qdrant", Category: "IA e vetores", Kind: "custom_api", Description: "Banco vetorial para RAG e busca semântica.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"collections"}},
		{ID: "pinecone", Name: "Pinecone", Category: "IA e vetores", Kind: "custom_api", Description: "Banco vetorial gerenciado para embeddings.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"indexes"}},
		{ID: "huggingface-hub", Name: "Hugging Face Hub", Category: "IA e vetores", Kind: "custom_api", Description: "Modelos, datasets, Spaces e Inference Endpoints.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"models", "datasets"}},
		{ID: "replicate", Name: "Replicate", Category: "IA e vetores", Kind: "custom_api", Description: "Rodar modelos de imagem, vídeo e áudio por API.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"predictions"}},
		{ID: "elevenlabs", Name: "ElevenLabs", Category: "IA e vetores", Kind: "custom_api", Description: "Síntese de voz, clonagem e dublagem.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"tts", "voices"}},
		{ID: "deepgram", Name: "Deepgram", Category: "IA e vetores", Kind: "custom_api", Description: "Transcrição de áudio e voz em tempo real.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"listen", "speak"}},
		{ID: "google-cloud", Name: "Google Cloud", Category: "Deploy", Kind: "custom_api", Description: "Cloud Run, Storage, BigQuery e IAM com escopo explícito.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"run", "storage"}},
		{ID: "aws", Name: "AWS", Category: "Deploy", Kind: "custom_api", Description: "S3, Lambda, EC2 e demais serviços via credencial IAM restrita.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"s3", "lambda"}},
		{ID: "azure", Name: "Microsoft Azure", Category: "Deploy", Kind: "custom_api", Description: "App Service, Functions, Storage e Azure OpenAI.", Auth: "oauth", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"resources"}},
		{ID: "digitalocean", Name: "DigitalOcean", Category: "Deploy", Kind: "custom_api", Description: "Droplets, App Platform, bancos e Spaces.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"apps", "droplets"}},
		{ID: "railway", Name: "Railway", Category: "Deploy", Kind: "custom_api", Description: "Serviços, bancos, variáveis e deploys.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"projects", "deployments"}},
		{ID: "render", Name: "Render", Category: "Deploy", Kind: "custom_api", Description: "Web services, workers, cron e bancos gerenciados.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"services", "deploys"}},
		{ID: "flyio", Name: "Fly.io", Category: "Deploy", Kind: "custom_api", Description: "Apps, máquinas e volumes perto do usuário.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"apps", "machines"}},
		{ID: "hetzner", Name: "Hetzner Cloud", Category: "Deploy", Kind: "custom_api", Description: "Servidores, volumes, firewalls e redes.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"servers"}},
		{ID: "hostinger", Name: "Hostinger", Category: "Deploy", Kind: "custom_api", Description: "VPS, hospedagem e domínios.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"vps", "domains"}},
		{ID: "coolify", Name: "Coolify", Category: "Deploy", Kind: "custom_api", Description: "PaaS self-hosted: apps, bancos e serviços.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"applications"}},
		{ID: "docker-hub", Name: "Docker Hub", Category: "Deploy", Kind: "custom_api", Description: "Repositórios e tags de imagens de contêiner.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"repositories"}},
		{ID: "bitbucket", Name: "Bitbucket", Category: "Desenvolvimento", Kind: "custom_api", Description: "Repositórios, pull requests e pipelines.", Auth: "oauth_or_api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"repos", "pipelines"}},
		{ID: "circleci", Name: "CircleCI", Category: "Desenvolvimento", Kind: "custom_api", Description: "Pipelines e jobs de CI/CD.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"pipelines"}},
		{ID: "sentry", Name: "Sentry", Category: "Observabilidade", Kind: "custom_api", Description: "Erros, performance e releases da aplicação.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"issues", "events"}},
		{ID: "datadog", Name: "Datadog", Category: "Observabilidade", Kind: "custom_api", Description: "Métricas, logs, traces e monitores.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"metrics", "logs"}},
		{ID: "grafana", Name: "Grafana", Category: "Observabilidade", Kind: "custom_api", Description: "Dashboards, alertas e fontes de dados.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"dashboards", "alerts"}},
		{ID: "betterstack", Name: "Better Stack", Category: "Observabilidade", Kind: "custom_api", Description: "Uptime, incidentes, status page e logs.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"monitors", "logs"}},
		{ID: "posthog", Name: "PostHog", Category: "Dados e métricas", Kind: "custom_api", Description: "Analytics de produto, feature flags e session replay.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"events", "flags"}},
		{ID: "mixpanel", Name: "Mixpanel", Category: "Dados e métricas", Kind: "custom_api", Description: "Eventos, funis e coortes de produto.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"events"}},
		{ID: "auth0", Name: "Auth0", Category: "Autenticação", Kind: "custom_api", Description: "Usuários, aplicações e regras de login.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"users", "applications"}},
		{ID: "clerk", Name: "Clerk", Category: "Autenticação", Kind: "custom_api", Description: "Usuários, organizações e sessões.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"users", "organizations"}},
		{ID: "algolia", Name: "Algolia", Category: "Busca", Kind: "custom_api", Description: "Índices e busca instantânea.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"indices"}},
		{ID: "meilisearch", Name: "Meilisearch", Category: "Busca", Kind: "custom_api", Description: "Busca open source rápida e tolerante a erros.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"indexes"}},
		{ID: "cloudinary", Name: "Cloudinary", Category: "Mídia", Kind: "custom_api", Description: "Upload, transformação e entrega de imagens e vídeos.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"assets"}},
		{ID: "contentful", Name: "Contentful", Category: "CMS", Kind: "custom_api", Description: "Conteúdo estruturado e publicação; publicar exige approval.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"entries", "assets"}},
		{ID: "sanity", Name: "Sanity", Category: "CMS", Kind: "custom_api", Description: "Conteúdo estruturado com GROQ e Studio.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"documents"}},
		{ID: "strapi", Name: "Strapi", Category: "CMS", Kind: "custom_api", Description: "CMS headless open source.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"content"}},
		{ID: "pusher", Name: "Pusher", Category: "Tempo real", Kind: "custom_api", Description: "Canais e eventos em tempo real.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"channels"}},
		{ID: "livekit", Name: "LiveKit", Category: "Tempo real", Kind: "custom_api", Description: "Salas de áudio e vídeo e agentes de voz.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"rooms"}},
		{ID: "novu", Name: "Novu", Category: "Notificações", Kind: "custom_api", Description: "Notificações multicanal: e-mail, SMS, push e in-app.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"workflows"}},
		{ID: "expo", Name: "Expo / EAS", Category: "Mobile", Kind: "custom_api", Description: "Builds, submissões e updates de apps React Native.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"builds", "updates"}},
		{ID: "lemon-squeezy", Name: "Lemon Squeezy", Category: "Pagamentos", Kind: "custom_api", Description: "Produtos digitais, assinaturas e licenças; movimentações exigem approval.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"products", "subscriptions"}},
		{ID: "paddle", Name: "Paddle", Category: "Pagamentos", Kind: "custom_api", Description: "Merchant of record para SaaS: assinaturas e impostos.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"subscriptions"}},
		{ID: "asaas", Name: "Asaas", Category: "Pagamentos", Kind: "custom_api", Description: "Cobranças por boleto, Pix e cartão no Brasil; movimentações exigem approval.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"payments"}},
		{ID: "pagseguro", Name: "PagBank / PagSeguro", Category: "Pagamentos", Kind: "custom_api", Description: "Checkout, Pix e cartão; movimentações exigem approval.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"orders"}},
		{ID: "pagarme", Name: "Pagar.me", Category: "Pagamentos", Kind: "custom_api", Description: "Pedidos, cobranças e recebedores; movimentações exigem approval.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"orders", "charges"}},
		{ID: "openai", Name: "OpenAI", Category: "Modelos de IA", Kind: "custom_api", Description: "Modelos GPT, embeddings e áudio; conecte colando a API key.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"models"}},
		{ID: "openrouter", Name: "OpenRouter", Category: "Modelos de IA", Kind: "custom_api", Description: "Gateway para centenas de modelos de múltiplos provedores.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"models"}},
		{ID: "groq", Name: "Groq", Category: "Modelos de IA", Kind: "custom_api", Description: "Inferência LLM de baixa latência (API compatível com OpenAI).", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"models"}},
		{ID: "together", Name: "Together AI", Category: "Modelos de IA", Kind: "custom_api", Description: "Centenas de modelos open-weights hospedados via API.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"models"}},
		{ID: "mistral", Name: "Mistral AI", Category: "Modelos de IA", Kind: "custom_api", Description: "Modelos Mistral/Codestral via API; conecte colando a API key.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"models"}},
		{ID: "cohere", Name: "Cohere", Category: "Modelos de IA", Kind: "custom_api", Description: "Modelos de geração, embeddings e rerank via API.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"models"}},
		{ID: "clickup", Name: "ClickUp", Category: "Produtividade", Kind: "custom_api", Description: "Tarefas, listas e espaços; leitura por chave, mutações exigem approval.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"tasks", "spaces"}},
		{ID: "vultr", Name: "Vultr", Category: "Infraestrutura", Kind: "custom_api", Description: "Instâncias, redes e billing da conta Vultr.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"instances", "account"}},
		{ID: "linode", Name: "Linode / Akamai", Category: "Infraestrutura", Kind: "custom_api", Description: "Linodes, volumes e billing via API da Akamai Connected Cloud.", Auth: "api_key", Source: "custom_api", Status: "operator_setup_required", Scopes: []string{"linodes", "account"}},
	}
	for i := range entries {
		if qc, ok := quickConnects[entries[i].ID]; ok && len(QuickConnectOperations(entries[i].ID)) > 0 {
			entries[i].QuickConnect = true
			entries[i].APIBaseURL = qc.base
			entries[i].APIAuthHeader = qc.header
			entries[i].APIAuthScheme = qc.scheme
			entries[i].APISelfHosted = qc.selfHosted
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries
}
