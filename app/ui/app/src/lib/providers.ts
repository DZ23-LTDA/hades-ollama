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
