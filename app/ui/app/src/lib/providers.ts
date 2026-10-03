import { API_BASE } from "@/lib/config";

export type ProviderStatus = {
  name: string;
  type: string;
  base_url: string;
  api_key_env?: string;
  models: number;
  enabled: boolean;
  configured: boolean;
  needs_key: boolean;
};

async function failure(response: Response, fallback: string): Promise<Error> {
  const text = (await response.text().catch(() => "")).trim();
  try {
    const parsed = JSON.parse(text) as { error?: string };
    if (parsed.error) return new Error(parsed.error);
  } catch {
    // Not JSON; fall through to the raw text.
  }
  return new Error(text || fallback);
}

export async function listProviders(): Promise<{
  configPath: string;
  providers: ProviderStatus[];
}> {
  const response = await fetch(`${API_BASE}/api/v1/providers`);
  if (!response.ok)
    throw await failure(response, "Falha ao carregar provedores");
  const body = (await response.json()) as {
    config_path?: string;
    providers?: ProviderStatus[] | null;
  };
  return {
    configPath: body.config_path ?? "",
    providers: Array.isArray(body.providers) ? body.providers : [],
  };
}

export async function saveProviderKey(
  name: string,
  key: string,
): Promise<void> {
  const trimmed = key.trim();
  if (!trimmed) throw new Error("Cole a chave de API");
  const response = await fetch(
    `${API_BASE}/api/v1/providers/${encodeURIComponent(name)}/key`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: trimmed }),
    },
  );
  if (!response.ok) throw await failure(response, "Falha ao salvar a chave");
}

export async function removeProviderKey(name: string): Promise<void> {
  const response = await fetch(
    `${API_BASE}/api/v1/providers/${encodeURIComponent(name)}/key`,
    { method: "DELETE" },
  );
  if (!response.ok) throw await failure(response, "Falha ao remover a chave");
}

export type CreateProviderInput = {
  name: string;
  base_url: string;
  type?: string;
  api_key_env?: string;
  models: string[];
  api_key?: string;
};

// createProvider registers a brand-new provider from the UI. The optional API
// key travels only in this request body; the backend stores it in the OS
// credential vault and never returns it.
export async function createProvider(
  input: CreateProviderInput,
): Promise<ProviderStatus> {
  const response = await fetch(`${API_BASE}/api/v1/providers`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!response.ok)
    throw await failure(response, "Falha ao cadastrar o provedor");
  return (await response.json()) as ProviderStatus;
}

// parseModelIds splits a free-text field (commas or line breaks) into a unique,
// trimmed, non-empty list of model ids, preserving the order the user typed.
export function parseModelIds(raw: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const piece of raw.split(/[\n,]+/)) {
    const id = piece.trim();
    if (id && !seen.has(id)) {
      seen.add(id);
      out.push(id);
    }
  }
  return out;
}

export type ProviderPreset = {
  id: string;
  label: string;
  name: string;
  base_url: string;
  type?: string;
};

// PROVIDER_PRESETS pre-fill the name and endpoint for well-known
// OpenAI-compatible services. They do NOT imply the service is free: price and
// quota always depend on the chosen provider. "custom" clears the fields.
export const PROVIDER_PRESETS: ProviderPreset[] = [
  { id: "openai", label: "OpenAI", name: "openai", base_url: "https://api.openai.com/v1" },
  {
    id: "anthropic",
    label: "Anthropic (Claude)",
    name: "anthropic",
    base_url: "https://api.anthropic.com",
    type: "anthropic",
  },
  { id: "groq", label: "Groq", name: "groq", base_url: "https://api.groq.com/openai/v1" },
  { id: "openrouter", label: "OpenRouter", name: "openrouter", base_url: "https://openrouter.ai/api/v1" },
  { id: "deepseek", label: "DeepSeek", name: "deepseek", base_url: "https://api.deepseek.com" },
  { id: "mistral", label: "Mistral", name: "mistral", base_url: "https://api.mistral.ai/v1" },
  {
    id: "gemini",
    label: "Google Gemini",
    name: "gemini",
    base_url: "https://generativelanguage.googleapis.com/v1beta/openai",
  },
  { id: "custom", label: "Personalizado", name: "", base_url: "" },
];

// sortProviders lists ready providers first, then the rest by name.
export function sortProviders(providers: ProviderStatus[]): ProviderStatus[] {
  return [...providers].sort(
    (a, b) =>
      Number(b.configured) - Number(a.configured) ||
      a.name.localeCompare(b.name),
  );
}

export type ProviderModels = {
  configured: string[];
  available: string[];
  error?: string;
};

export async function listProviderModels(
  name: string,
): Promise<ProviderModels> {
  const response = await fetch(
    `${API_BASE}/api/v1/providers/${encodeURIComponent(name)}/models`,
  );
  if (!response.ok) throw await failure(response, "Falha ao consultar modelos");
  const body = (await response.json()) as Partial<ProviderModels>;
  return {
    configured: Array.isArray(body.configured) ? body.configured : [],
    available: Array.isArray(body.available) ? body.available : [],
    error: body.error,
  };
}

export async function saveProviderModels(
  name: string,
  models: string[],
): Promise<void> {
  if (models.length === 0) throw new Error("Selecione ao menos um modelo");
  const response = await fetch(
    `${API_BASE}/api/v1/providers/${encodeURIComponent(name)}/models`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ models }),
    },
  );
  if (!response.ok) throw await failure(response, "Falha ao salvar modelos");
}

// staleModels lists configured ids the provider no longer offers, which is
// what makes chats fail with 404 "model does not exist".
export function staleModels(models: ProviderModels): string[] {
  if (models.error || models.available.length === 0) return [];
  const available = new Set(models.available);
  return models.configured.filter((id) => !available.has(id));
}
