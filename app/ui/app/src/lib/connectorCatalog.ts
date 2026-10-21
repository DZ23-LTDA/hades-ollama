import { t } from "@/i18n";
import type { AgentConnectorCatalogEntry } from "./agenticClient";

// Catalogo de conectores e fontes de dados exibido no marketplace de Plugins.
// Espelha a amplitude observavel em assistentes agentic (Gmail, Google Workspace,
// Notion, redes sociais, ads, fontes de dados) para que a superficie ja nasca
// rica mesmo quando o backend ainda nao homologou os providers. Nenhuma entrada
// afirma conexao pronta: `status: "available"` significa apenas "disponivel para
// configurar" — o backend continua exigindo OAuth/API key/homologacao real.

export type CatalogGroup = "connectors" | "data_sources";

export type BuiltinCatalogEntry = AgentConnectorCatalogEntry & {
  group: CatalogGroup;
};

const entry = (
  group: CatalogGroup,
  id: string,
  name: string,
  category: string,
  kind: string,
  auth: string,
  description: string,
): BuiltinCatalogEntry => ({
  id,
  name,
  category,
  kind,
  auth,
  description,
  group,
  source: "builtin",
  status: "available",
});

// `category` NAO e' traduzido de proposito: e' identificador de contrato vindo do
// backend (`ConnectorsPage` casa por igualdade exacta em `CATEGORY_GROUPS`) e serve
// de chave em `CATEGORY_ACCENT`. Traduzi-lo quebraria filtro e cor do tile.
export const BUILTIN_CONNECTOR_CATALOG: BuiltinCatalogEntry[] = [
  // Comunicação
  entry("connectors", "gmail", "Gmail", "Comunicação", "oauth", "OAuth", t("connectorCatalog.gmail.description")),
  entry("connectors", "outlook-mail", "Outlook Mail", "Comunicação", "oauth", "OAuth", t("connectorCatalog.outlookMail.description")),
  entry("connectors", "slack", "Slack", "Comunicação", "oauth", "OAuth", t("connectorCatalog.slack.description")),
  entry("connectors", "whatsapp", "WhatsApp Business", "Comunicação", "custom_api", "API key", t("connectorCatalog.whatsapp.description")),
  entry("connectors", "discord", "Discord", "Comunicação", "oauth", "OAuth", t("connectorCatalog.discord.description")),
  // Produtividade
  entry("connectors", "google-workspace", "Google Workspace", "Produtividade", "oauth", "OAuth", t("connectorCatalog.googleWorkspace.description")),
  entry("connectors", "google-calendar", "Google Agenda", "Produtividade", "oauth", "OAuth", t("connectorCatalog.googleCalendar.description")),
  entry("connectors", "notion", "Notion", "Produtividade", "oauth", "OAuth", t("connectorCatalog.notion.description")),
  entry("connectors", "microsoft-365", "Microsoft 365", "Produtividade", "oauth", "OAuth", t("connectorCatalog.microsoft365.description")),
  entry("connectors", "jira", "Jira", "Produtividade", "oauth", "OAuth", t("connectorCatalog.jira.description")),
  entry("connectors", "linear", "Linear", "Produtividade", "oauth", "OAuth", t("connectorCatalog.linear.description")),
  // Desenvolvimento
  entry("connectors", "github", "GitHub", "Desenvolvimento", "oauth", "OAuth", t("connectorCatalog.github.description")),
  entry("connectors", "gitlab", "GitLab", "Desenvolvimento", "oauth", "OAuth", t("connectorCatalog.gitlab.description")),
  entry("connectors", "supabase", "Supabase", "Desenvolvimento", "custom_api", "API key", t("connectorCatalog.supabase.description")),
  // Social e Ads
  entry("connectors", "instagram", "Instagram", "Social e Ads", "oauth", "OAuth", t("connectorCatalog.instagram.description")),
  entry("connectors", "meta-ads", "Meta Ads Manager", "Social e Ads", "oauth", "OAuth", t("connectorCatalog.metaAds.description")),
  entry("connectors", "tiktok", "TikTok", "Social e Ads", "oauth", "OAuth", t("connectorCatalog.tiktok.description")),
  entry("connectors", "linkedin", "LinkedIn", "Social e Ads", "oauth", "OAuth", t("connectorCatalog.linkedin.description")),
  // Mídia
  entry("connectors", "video-editor", "Video Editor", "Mídia", "mcp", "MCP", t("connectorCatalog.videoEditor.description")),
  entry("connectors", "higgsfield", "Higgsfield", "Mídia", "custom_api", "API key", t("connectorCatalog.higgsfield.description")),
  entry("connectors", "canva", "Canva", "Mídia", "oauth", "OAuth", t("connectorCatalog.canva.description")),

  // Fontes de dados
  entry("data_sources", "similarweb", "Similarweb", "Fontes de dados", "custom_api", "API key", t("connectorCatalog.similarweb.description")),
  entry("data_sources", "world-bank", "World Bank DataBank", "Fontes de dados", "custom_api", "API key", t("connectorCatalog.worldBank.description")),
  entry("data_sources", "bigquery", "Google BigQuery", "Fontes de dados", "oauth", "OAuth", t("connectorCatalog.bigquery.description")),
  entry("data_sources", "stripe", "Stripe", "Fontes de dados", "custom_api", "API key", t("connectorCatalog.stripe.description")),
  entry("data_sources", "google-analytics", "Google Analytics", "Fontes de dados", "oauth", "OAuth", t("connectorCatalog.googleAnalytics.description")),
];

const CATEGORY_ACCENT: Record<string, string> = {
  Comunicação: "bg-sky-100 text-sky-700 dark:bg-sky-950/40 dark:text-sky-300",
  Produtividade: "bg-violet-100 text-violet-700 dark:bg-violet-950/40 dark:text-violet-300",
  Desenvolvimento: "bg-slate-200 text-slate-700 dark:bg-slate-800 dark:text-slate-200",
  "Social e Ads": "bg-pink-100 text-pink-700 dark:bg-pink-950/40 dark:text-pink-300",
  Mídia: "bg-amber-100 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300",
  "Fontes de dados": "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300",
};

/** Classes do tile de ícone (monograma colorido) por categoria do conector. */
export function catalogAccentClasses(category: string): string {
  return (
    CATEGORY_ACCENT[category] ??
    "bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
  );
}

/**
 * Mescla o catálogo do backend com o catálogo interno, sem duplicar entradas
 * (o backend tem precedência por id ou nome). O resultado alimenta a mesma UI
 * de marketplace já existente.
 */
export function mergeConnectorCatalog(
  backend: AgentConnectorCatalogEntry[],
  builtin: BuiltinCatalogEntry[] = BUILTIN_CONNECTOR_CATALOG,
): AgentConnectorCatalogEntry[] {
  const seenIds = new Set(backend.map((item) => item.id.toLocaleLowerCase()));
  const seenNames = new Set(backend.map((item) => item.name.toLocaleLowerCase()));
  const extras = builtin.filter(
    (item) =>
      !seenIds.has(item.id.toLocaleLowerCase()) &&
      !seenNames.has(item.name.toLocaleLowerCase()),
  );
  return [...backend, ...extras];
}

// Domínio por conector, usado para montar o logo pela CDN do Brandfetch.
const CONNECTOR_DOMAINS: Record<string, string> = {
  gmail: "gmail.com",
  "outlook-mail": "outlook.com",
  slack: "slack.com",
  whatsapp: "whatsapp.com",
  discord: "discord.com",
  "google-workspace": "workspace.google.com",
  "google-calendar": "calendar.google.com",
  notion: "notion.so",
  "microsoft-365": "microsoft.com",
  jira: "atlassian.com",
  linear: "linear.app",
  github: "github.com",
  gitlab: "gitlab.com",
  supabase: "supabase.com",
  instagram: "instagram.com",
  "meta-ads": "meta.com",
  tiktok: "tiktok.com",
  linkedin: "linkedin.com",
  higgsfield: "higgsfield.ai",
  canva: "canva.com",
  similarweb: "similarweb.com",
  "world-bank": "worldbank.org",
  bigquery: "cloud.google.com",
  stripe: "stripe.com",
  "google-analytics": "analytics.google.com",
};

// clientId público de embed do Brandfetch, via variável de ambiente (NÃO
// versionado). Sem ele, o marketplace usa os monogramas coloridos por padrão —
// o repositório não embute nenhum logo de terceiros. Configure em .env:
//   VITE_BRANDFETCH_CLIENT_ID=<seu clientId público de embed>
const BRANDFETCH_CLIENT_ID =
  (import.meta.env.VITE_BRANDFETCH_CLIENT_ID as string | undefined) || "";

/**
 * URL do logo do conector via CDN do Brandfetch (somente para renderização em
 * `<img>`), ou `null` quando não há clientId/domínio — nesse caso a UI mostra o
 * monograma. As imagens vêm da conta Brandfetch do operador, não do repositório.
 */
export function connectorLogoUrl(id: string, size = 64): string | null {
  if (!BRANDFETCH_CLIENT_ID) return null;
  const domain = CONNECTOR_DOMAINS[id];
  if (!domain) return null;
  return `https://cdn.brandfetch.io/domain/${domain}/w/${size}/h/${size}/fallback/transparent?c=${BRANDFETCH_CLIENT_ID}`;
}
