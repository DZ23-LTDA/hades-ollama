import type { ModelRecommendation } from "@/api";

// A sensible tiny default that runs even on modest hardware, used when the
// backend offers no local (non-cloud) recommendation.
export const FIRST_MODEL_FALLBACK = "gemma3:1b";

export const FIRST_MODEL_ERROR_MESSAGE =
  "Não foi possível baixar o modelo agora. Verifique sua conexão e tente novamente.";

// pickFirstModel chooses the friendliest first local model to download: the
// smallest (by VRAM) non-cloud recommendation from the backend, falling back to
// a small default when there is no usable recommendation. Pure for testing.
export function pickFirstModel(
  recommendations: ModelRecommendation[] | undefined,
  fallback: string = FIRST_MODEL_FALLBACK,
): string {
  const local = (recommendations || []).filter(
    (r) => r.model && !r.model.endsWith("cloud"),
  );
  if (local.length === 0) return fallback;
  const sorted = [...local].sort(
    (a, b) => (a.vram_bytes ?? Infinity) - (b.vram_bytes ?? Infinity),
  );
  return sorted[0].model;
}
