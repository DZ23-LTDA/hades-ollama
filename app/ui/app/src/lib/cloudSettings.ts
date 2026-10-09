import { API_BASE } from "@/lib/config";

export async function enableCloudModelsOptIn(fetcher: typeof fetch = fetch): Promise<void> {
  const response = await fetcher(`${API_BASE}/api/v1/cloud`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled: true }),
  });

  if (!response.ok) {
    throw new Error(`O servidor não salvou a preferência de modelos de nuvem (HTTP ${response.status}).`);
  }

  const status = (await response.json()) as { disabled?: unknown };
  if (status.disabled !== false) {
    throw new Error("O servidor não confirmou a ativação dos modelos de nuvem.");
  }
}
