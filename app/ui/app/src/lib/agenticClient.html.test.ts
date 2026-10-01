import { afterEach, describe, expect, it, vi } from "vitest";
import {
  AGENT_LOGIN_REQUIRED_MESSAGE,
  agentFetch,
  listProjects,
} from "./agenticClient";

describe("agentFetch response validation", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("rejects the SPA HTML shell instead of returning it as data", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response("<!doctype html><html></html>", {
          status: 200,
          headers: { "Content-Type": "text/html; charset=utf-8" },
        }),
      ),
    );

    await expect(listProjects()).rejects.toThrow("Agent API indisponível");
  });

  it("still returns JSON bodies and tolerates empty 204 responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ projects: [] }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        )
        .mockResolvedValueOnce(new Response(null, { status: 204 })),
    );

    await expect(listProjects()).resolves.toEqual({ projects: [] });
    await expect(
      agentFetch("/api/agent/v1/projects/p1", { method: "DELETE" }),
    ).resolves.toEqual({});
  });

  it("explains how to proceed when network exposure requires login", async () => {
    vi.stubGlobal("window", { dispatchEvent: vi.fn() });
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "bearer token is required" }), {
          status: 401,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );

    await expect(listProjects()).rejects.toThrow(AGENT_LOGIN_REQUIRED_MESSAGE);
  });
});
