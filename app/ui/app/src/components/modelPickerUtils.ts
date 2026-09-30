import type { Model } from "@/gotypes";

export function modelGroup(model: Model): string {
  if (model.available === false) return "Indisponíveis";
  if (model.kind === "router") return "Roteamento inteligente";
  if (model.kind === "remote") return model.provider || "APIs externas";
  return "Modelos locais";
}

export function isModelSelectable(model: Model): boolean {
  return model.available !== false && (model.status === undefined || model.status === "PASS");
}

export function sortModelsClean(models: Model[]): Model[] {
  const usable = models.filter((m) => isModelSelectable(m));
  const unavailable = models.filter((m) => !isModelSelectable(m));
  return [...usable, ...unavailable];
}
