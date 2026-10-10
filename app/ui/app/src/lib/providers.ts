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
  presets: ServerProviderPreset[];
}> {
  const response = await fetch(`${API_BASE}/api/v1/providers`);
  if (!response.ok)
    throw await failure(response, "Falha ao carregar provedores");
  const body = (await response.json()) as {
    config_path?: string;
    providers?: ProviderStatus[] | null;
    presets?: ServerProviderPreset[] | null;
  };
  return {
    configPath: body.config_path ?? "",
    providers: Array.isArray(body.providers) ? body.providers : [],
    presets: Array.isArray(body.presets) ? body.presets : [],
  };
}

// ServerProviderPreset mirrors the canonical guided presets served by
// GET /api/v1/providers (stage 4). It carries the credential variable NAME,
// never a credential value.
export type ServerProviderPreset = {
  id: string;
  name: string;
  type?: string;
  base_url: string;
  api_key_env?: string;
  auth_style?: string;
  paths?: string[];
  local?: boolean;
  requires_key?: boolean;
  docs_url?: string;
  notes?: string;
};

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
  /** Human-readable hint about credential/endpoint expectations. */
  hint?: string;
  /** Whether the service needs a credential; undefined when unknown. */
  requires_key?: boolean;
  /** Whether the service is a local server reached over loopback. */
  local?: boolean;
};

// PROVIDER_PRESETS pre-fill the name and endpoint for well-known
// OpenAI-compatible services. They do NOT imply the service is free: price and
// quota always depend on the chosen provider. "custom" clears the fields.
//
// This list is the OFFLINE fallback: when the server answers with its canonical
// guided presets, `mergeProviderPresets` prefers them (they also cover local
// servers such as Ollama, vLLM, llama.cpp and LM Studio).
export const PROVIDER_PRESETS: ProviderPreset[] = [
  {
    id: "openai",
    label: "OpenAI",
    name: "openai",
    base_url: "https://api.openai.com/v1",
  },
  {
    id: "anthropic",
    label: "Anthropic (Claude)",
    name: "anthropic",
    base_url: "https://api.anthropic.com",
    type: "anthropic",
  },
  {
    id: "groq",
    label: "Groq",
    name: "groq",
    base_url: "https://api.groq.com/openai/v1",
  },
  {
    id: "openrouter",
    label: "OpenRouter",
    name: "openrouter",
    base_url: "https://openrouter.ai/api/v1",
  },
  {
    id: "deepseek",
    label: "DeepSeek",
    name: "deepseek",
    base_url: "https://api.deepseek.com",
  },
  {
    id: "mistral",
    label: "Mistral",
    name: "mistral",
    base_url: "https://api.mistral.ai/v1",
  },
  {
    id: "gemini",
    label: "Google Gemini",
    name: "gemini",
    base_url: "https://generativelanguage.googleapis.com/v1beta/openai",
  },
  { id: "custom", label: "Personalizado", name: "", base_url: "" },
];

// providerPresetHint explains, in pt-BR, what the preset expects from the
// operator: a local server reached over loopback, a credential variable, or
// neither. It never includes a credential value.
export function providerPresetHint(
  preset: ServerProviderPreset,
): string | undefined {
  const parts: string[] = [];
  if (preset.local) {
    parts.push("Servidor local (HTTP em loopback)");
  } else if (preset.requires_key) {
    parts.push("Exige credencial");
  }
  if (preset.api_key_env) parts.push(`variável ${preset.api_key_env}`);
  if (preset.notes) parts.push(preset.notes);
  return parts.length > 0 ? parts.join(" · ") : undefined;
}

// mergeProviderPresets turns the canonical server presets into the options the
// form renders. Order: the server list; when it is empty (older server or
// offline) the bundled fallback is used instead. "Personalizado" is always
// kept, because it is what lets the operator type an arbitrary endpoint.
export function mergeProviderPresets(
  server: ServerProviderPreset[] | null | undefined,
  fallback: ProviderPreset[] = PROVIDER_PRESETS,
): ProviderPreset[] {
  const out: ProviderPreset[] = [];
  const seen = new Set<string>();
  for (const preset of server ?? []) {
    const id = (preset?.id ?? "").trim();
    if (!id || seen.has(id)) continue;
    seen.add(id);
    out.push({
      id,
      label: (preset.name ?? "").trim() || id,
      name: id,
      base_url: preset.base_url ?? "",
      type: preset.type,
      hint: providerPresetHint(preset),
      requires_key: preset.requires_key,
      local: preset.local,
    });
  }
  const source =
    out.length > 0 ? fallback.filter((p) => p.id === "custom") : fallback;
  for (const preset of source) {
    if (seen.has(preset.id)) continue;
    seen.add(preset.id);
    out.push(preset);
  }
  return out;
}

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
