export type InboxMission = { id: string; objective: string; state: string; version?: number };
export type InboxEvent = { id: string; type: string; step_id?: string; created_at: string };
export type InboxStorage = { getItem(key: string): Promise<string | null>; setItem(key: string, value: string): Promise<unknown> };
export type InboxRequest = <T>(path: string) => Promise<T>;

export function inboxKey(server: string, organizationID: string | null, authenticated: boolean): string | null {
  if (authenticated && !organizationID) return null;
  return `dz23.agent.cached.inbox.${encodeURIComponent(server)}.${encodeURIComponent(authenticated ? organizationID as string : "local")}`;
}

export async function loadInbox(storage: InboxStorage, key: string, request: InboxRequest) {
  try {
    const response = await request<{ missions: InboxMission[] }>("/api/agent/v1/missions");
    if (!Array.isArray(response.missions)) throw new Error("Resposta de missões inválida");
    await storage.setItem(key, JSON.stringify(response.missions));
    return { missions: response.missions, source: "network" as const, error: "" };
  } catch (cause) {
    if (typeof cause === "object" && cause !== null && "status" in cause) return { missions: [] as InboxMission[], source: "error" as const, error: cause instanceof Error ? cause.message : "Erro do servidor." };
    const cached = await storage.getItem(key);
    if (cached) {
      try {
        const missions: unknown = JSON.parse(cached);
        if (Array.isArray(missions)) return { missions: missions as InboxMission[], source: "cache" as const, error: "" };
      } catch { /* Corrupt cache is not shown. */ }
    }
    return { missions: [] as InboxMission[], source: "error" as const, error: cause instanceof Error ? cause.message : "Não foi possível carregar missões." };
  }
}

export async function loadInboxDetail(storage: InboxStorage, key: string, id: string, request: InboxRequest) {
  try {
    const [mission, eventResponse] = await Promise.all([
      request<InboxMission>(`/api/agent/v1/missions/${encodeURIComponent(id)}`),
      request<{ events: InboxEvent[] }>(`/api/agent/v1/missions/${encodeURIComponent(id)}/events`),
    ]);
    if (!mission || mission.id !== id || !Array.isArray(eventResponse.events)) throw new Error("Detalhe de missão inválido");
    const detail = { mission, events: eventResponse.events };
    await storage.setItem(key, JSON.stringify(detail));
    return { ...detail, source: "network" as const, error: "" };
  } catch (cause) {
    if (typeof cause === "object" && cause !== null && "status" in cause) return { mission: null, events: [] as InboxEvent[], source: "error" as const, error: cause instanceof Error ? cause.message : "Erro do servidor." };
    const cached = await storage.getItem(key);
    if (cached) {
      try {
        const detail = JSON.parse(cached) as { mission?: InboxMission; events?: InboxEvent[] };
        if (detail.mission?.id === id && Array.isArray(detail.events)) return { mission: detail.mission, events: detail.events, source: "cache" as const, error: "" };
      } catch { /* Corrupt cache is not shown. */ }
    }
    return { mission: null, events: [] as InboxEvent[], source: "error" as const, error: cause instanceof Error ? cause.message : "Não foi possível abrir a missão." };
  }
}
