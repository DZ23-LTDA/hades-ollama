import type { Model } from "@/gotypes";

export function modelGroup(model: Model): string {
  if (model.available === false || (model.status && model.status !== "PASS")) return "Indisponíveis";
  if (model.kind === "cli_subscription") return "Assinatura (CLI)";
  if (model.kind === "router") return "Roteamento inteligente";
  if (model.kind === "remote") return model.provider || "APIs externas";
  if (model.isCloud?.()) return "Ollama Cloud";
  return "Modelos locais";
}

// chatGroupRank orders the picker so the user's own installed local models come
// first, then downloadable local models, then Ollama Cloud, then the advanced
// provider/router/CLI groups. This keeps "Modelos locais" (e.g. qwen) at the
// top instead of being buried under cloud entries.
function chatGroupRank(model: Model): number {
  if (model.kind === "cli_subscription") return 5;
  if (model.kind === "router") return 6;
  if (model.kind === "remote") return 7;
  if (model.isCloud?.()) return 4;
  if (typeof model.digest === "string" && model.digest !== "") return 0; // installed local
  return 1; // downloadable local
}

export function isModelSelectable(model: Model): boolean {
  return model.available !== false && (model.status === undefined || model.status === "PASS");
}

export function getModelCostTag(model: Model): string | undefined {
  if (model.kind === "cli_subscription" && isModelSelectable(model)) {
    return "0-assinatura";
  }
  return model.cost_tag;
}

export function sortModelsClean(models: Model[]): Model[] {
  const usable = models.filter((m) => isModelSelectable(m));
  const unavailable = models.filter((m) => !isModelSelectable(m));
  // Stable sort keeps the upstream order inside each group while lifting the
  // user's installed local models to the top.
  const ordered = usable
    .map((model, index) => ({ model, index }))
    .sort((a, b) => chatGroupRank(a.model) - chatGroupRank(b.model) || a.index - b.index)
    .map((entry) => entry.model);
  return [...ordered, ...unavailable];
}
