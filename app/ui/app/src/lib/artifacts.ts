import { API_BASE } from "@/lib/config";

export function artifactDownloadPath(missionId: string, artifactId: string): string {
  return `${API_BASE}/api/agent/v1/missions/${encodeURIComponent(missionId)}/artifacts/${encodeURIComponent(artifactId)}`;
}
