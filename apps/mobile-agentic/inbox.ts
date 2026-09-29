export type InboxMission = { id: string; objective: string; state: string; version?: number };
export type InboxEvent = { id: string; type: string; step_id?: string; created_at: string };
export type InboxStorage = { getItem(key: string): Promise<string | null>; setItem(key: string, value: string): Promise<unknown> };
export type InboxRequest = <T>(path: string) => Promise<T>;
type Detail = { mission: InboxMission; events: InboxEvent[] };
type DetailCache = { version: 2; order: string[]; entries: Record<string, Detail> };
const maxCachedDetails = 20;

function parseDetailCache(raw: string | null): DetailCache {
  const empty: DetailCache = { version: 2, order: [], entries: {} };
  if (!raw) return empty;
  try {
    const value = JSON.parse(raw) as DetailCache | Detail;
    if ("version" in value && value.version === 2 && Array.isArray(value.order) && value.entries && typeof value.entries === "object") return value;
    if ("mission" in value && value.mission?.id && Array.isArray(value.events)) {
      return { version: 2, order: [value.mission.id], entries: { [value.mission.id]: value } };
    }
  } catch { /* Corrupt cache is not shown. */ }
  return empty;
}

export async function readInboxDetailCache(storage: InboxStorage, key: string, id: string): Promise<Detail | null> {
  const cache = parseDetailCache(await storage.getItem(key));
  const detail = cache.entries[id];
  return detail?.mission?.id === id && Array.isArray(detail.events) ? detail : null;
}

export async function cacheInboxDetail(storage: InboxStorage, key: string, detail: Detail): Promise<void> {
  const cache = parseDetailCache(await storage.getItem(key));
  const order = [...cache.order.filter((id) => id !== detail.mission.id), detail.mission.id].slice(-maxCachedDetails);
  const entries: Record<string, Detail> = Object.create(null) as Record<string, Detail>;
  for (const id of order) if (id === detail.mission.id || cache.entries[id]?.mission?.id === id) entries[id] = id === detail.mission.id ? detail : cache.entries[id];
  await storage.setItem(key, JSON.stringify({ version: 2, order, entries } satisfies DetailCache));
}

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
    await cacheInboxDetail(storage, key, detail);
    return { ...detail, source: "network" as const, error: "" };
  } catch (cause) {
    if (typeof cause === "object" && cause !== null && "status" in cause) return { mission: null, events: [] as InboxEvent[], source: "error" as const, error: cause instanceof Error ? cause.message : "Erro do servidor." };
    const detail = await readInboxDetailCache(storage, key, id);
    if (detail) return { ...detail, source: "cache" as const, error: "" };
    return { mission: null, events: [] as InboxEvent[], source: "error" as const, error: cause instanceof Error ? cause.message : "Não foi possível abrir a missão." };
  }
}
