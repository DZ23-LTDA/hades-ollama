import { API_BASE } from "@/lib/config";
import { agentSessionHeaders } from "@/lib/agenticClient";

async function failure(response: Response, fallback: string): Promise<Error> {
  const text = (await response.text().catch(() => "")).trim();
  try {
    const parsed = JSON.parse(text) as { error?: string };
    if (parsed.error) return new Error(parsed.error);
  } catch {
    // Not JSON; use the raw text.
  }
  return new Error(text || fallback);
}

export async function connectWithKey(
  id: string,
  key: string,
  baseURL?: string,
): Promise<void> {
  const trimmed = key.trim();
  if (!trimmed) throw new Error("Cole a chave de API");
  const response = await fetch(
    `${API_BASE}/api/v1/connectors/${encodeURIComponent(id)}/key`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json", ...agentSessionHeaders() },
      body: JSON.stringify(
        baseURL ? { key: trimmed, base_url: baseURL.trim() } : { key: trimmed },
      ),
    },
  );
  if (!response.ok) throw await failure(response, "Falha ao conectar");
}

export async function disconnectConnector(id: string): Promise<void> {
  const response = await fetch(
    `${API_BASE}/api/v1/connectors/${encodeURIComponent(id)}/key`,
    {
      method: "DELETE",
      headers: agentSessionHeaders(),
    },
  );
  if (!response.ok) throw await failure(response, "Falha ao desconectar");
}

// Where each service issues API keys, so the user does not have to search.
const KEY_HELP: Record<string, string> = {
  appwrite: "https://cloud.appwrite.io/console",
  asaas: "https://www.asaas.com/customerApiAccessToken/index",
  asana: "https://app.asana.com/0/my-apps",
  bitbucket: "https://bitbucket.org/account/settings/app-passwords/",
  brevo: "https://app.brevo.com/settings/keys/api",
  calendly: "https://calendly.com/integrations/api_webhooks",
  circleci: "https://app.circleci.com/settings/user/tokens",
  contentful: "https://app.contentful.com/account/profile/cma_tokens",
  datadog: "https://app.datadoghq.com/organization-settings/api-keys",
  deepgram: "https://console.deepgram.com/",
  dropbox: "https://www.dropbox.com/developers/apps",
  elevenlabs: "https://elevenlabs.io/app/settings/api-keys",
  expo: "https://expo.dev/settings/access-tokens",
  figma: "https://www.figma.com/developers/api#access-tokens",
  flyio: "https://fly.io/user/personal_access_tokens",
  gitlab: "https://gitlab.com/-/user_settings/personal_access_tokens",
  hetzner: "https://console.hetzner.cloud/",
  "huggingface-hub": "https://huggingface.co/settings/tokens",
  intercom: "https://app.intercom.com/a/apps/_/developer-hub",
  "lemon-squeezy": "https://app.lemonsqueezy.com/settings/api",
  linear: "https://linear.app/settings/account/security",
  novu: "https://dashboard.novu.co/api-keys",
  paddle: "https://vendors.paddle.com/authentication-v2",
  pinecone: "https://app.pinecone.io/",
  postmark: "https://account.postmarkapp.com/servers",
  railway: "https://railway.com/account/tokens",
  replicate: "https://replicate.com/account/api-tokens",
  sanity: "https://www.sanity.io/manage",
  typeform: "https://admin.typeform.com/user/tokens",
  whatsapp: "https://developers.facebook.com/apps/",
  airtable: "https://airtable.com/create/tokens",
  betterstack: "https://uptime.betterstack.com/team/api-tokens",
  clerk: "https://dashboard.clerk.com/last-active?path=api-keys",
  cloudflare: "https://dash.cloudflare.com/profile/api-tokens",
  digitalocean: "https://cloud.digitalocean.com/account/api/tokens",
  github: "https://github.com/settings/personal-access-tokens",
  hubspot: "https://app.hubspot.com/private-apps",
  "mercado-pago": "https://www.mercadopago.com.br/developers/panel/app",
  neon: "https://console.neon.tech/app/settings/api-keys",
  netlify: "https://app.netlify.com/user/applications#personal-access-tokens",
  posthog: "https://us.posthog.com/settings/user-api-keys",
  render: "https://dashboard.render.com/u/settings#api-keys",
  resend: "https://resend.com/api-keys",
  sendgrid: "https://app.sendgrid.com/settings/api_keys",
  sentry: "https://sentry.io/settings/account/api/auth-tokens/",
  stripe: "https://dashboard.stripe.com/apikeys",
  supabase: "https://supabase.com/dashboard/account/tokens",
  vercel: "https://vercel.com/account/settings/tokens",
};

export function keyHelpURL(id: string): string | undefined {
  return KEY_HELP[id];
}

const AUTH_DESCRIPTIONS: Record<string, string> = {
  oauth: "login OAuth",
  connection_url: "uma URL de conexão (banco de dados)",
  bot_token: "um token de bot",
  bot_or_oauth: "um token de bot ou OAuth",
  url_or_api_key: "URL e chave da sua instância",
};

// connectHint explains, for services that cannot be connected with a pasted
// key, what they need instead.
export function connectHint(name: string, auth: string): string {
  if (auth === "oauth") {
    return `${name} só aceita login OAuth, que precisa de um app registrado no ${name} pela equipe DZ23. Esse login de um clique ainda não está disponível.`;
  }
  const kind = AUTH_DESCRIPTIONS[auth] ?? "credenciais específicas";
  return `${name} usa ${kind}. Conecte pelo registro avançado, informando o endereço e a credencial.`;
}
