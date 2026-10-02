import { afterEach, describe, expect, it, vi } from "vitest";
import { disconnectUser } from "@/api";
import { clearAgentSession, hasAgentSession, setAgentSession } from "@/lib/agenticClient";
import { artifactDownloadPath } from "@/lib/artifacts";

const originalFetch = globalThis.fetch;

afterEach(() => {
  globalThis.fetch = originalFetch;
  clearAgentSession();
  vi.restoreAllMocks();
});

describe("AUD-FIX-1 contracts", () => {
  it("logout calls the real POST signout endpoint and clears the agent session", async () => {
    setAgentSession("session-token", "local-org");
    globalThis.fetch = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));

    await disconnectUser();
    clearAgentSession();

    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/signout"),
      expect.objectContaining({ method: "POST" }),
    );
    expect(hasAgentSession()).toBe(false);
  });

  it("uses the backend artifact route without the obsolete content suffix", () => {
    expect(artifactDownloadPath("mission-7", "artifact-9")).toContain(
      "/api/agent/v1/missions/mission-7/artifacts/artifact-9",
    );
    expect(artifactDownloadPath("mission-7", "artifact-9")).not.toContain("/content");
  });
});
