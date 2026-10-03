const STORAGE_PREFIX = "dz23.agent";

export function shouldQueueOffline(error: unknown): boolean {
  if (typeof error !== "object" || error === null) return false;
  if ("status" in error && typeof (error as { status?: unknown }).status === "number") {
    return false;
  }
  const candidate = error as { name?: unknown; message?: unknown };
  if (candidate.name !== "TypeError" || typeof candidate.message !== "string") {
    return false;
  }
  return /failed to fetch|fetch failed|network request failed|networkerror/i.test(candidate.message);
}

export function buildApprovalDecisionPayload(
  approved: boolean,
  nonce: string | undefined,
  reason: string,
): { decision: "approve" | "reject"; nonce?: string; reason: string } {
  const normalizedNonce = nonce?.trim();
  return {
    decision: approved ? "approve" : "reject",
    ...(normalizedNonce ? { nonce: normalizedNonce } : {}),
    reason: reason.trim(),
  };
}

function storageScope(
  organizationID: string | null,
  subjectID: string | null,
  authenticated: boolean,
): string | null {
  if (authenticated) {
    if (!organizationID?.trim() || !subjectID?.trim()) return null;
    return `${organizationID.trim()}.${subjectID.trim()}`;
  }
  if (organizationID !== "local" || subjectID !== "local") return null;
  return "local";
}

function storageKey(
  resource: string,
  server: string,
  organizationID: string | null,
  subjectID: string | null,
  authenticated: boolean,
): string | null {
  const normalizedServer = server.trim().replace(/\/+$/, "");
  const scope = storageScope(organizationID, subjectID, authenticated);
  if (!normalizedServer || !scope) return null;
  return `${STORAGE_PREFIX}.${resource}.${encodeURIComponent(normalizedServer)}.${encodeURIComponent(scope)}`;
}

export function queueStorageKey(
  server: string,
  organizationID: string | null,
  subjectID: string | null,
  authenticated: boolean,
): string | null {
  return storageKey("offline.queue", server, organizationID, subjectID, authenticated);
}

export function missionStorageKey(
  server: string,
  organizationID: string | null,
  subjectID: string | null,
  authenticated: boolean,
): string | null {
  return storageKey("cached.mission", server, organizationID, subjectID, authenticated);
}

export function pushStorageKey(
  server: string,
  organizationID: string | null,
  subjectID: string | null,
): string | null {
  return storageKey("push.registered", server, organizationID, subjectID, true);
}
