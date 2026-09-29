import { afterEach, describe, expect, it, vi } from "vitest";
import { clearAgentSession, setAgentSession } from "./agenticClient";
import { connectHint, connectWithKey, keyHelpURL } from "./connectorConnect";

describe("connector quick connect client", () => {
  afterEach(() => {
    clearAgentSession();
    vi.unstubAllGlobals();
  });

  it("sends the trimmed key to the connector endpoint", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    await connectWithKey("resend", "  re_123  ");
    expect(fetchMock.mock.calls[0][0]).toContain(
      "/api/v1/connectors/resend/key",
    );
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe("PUT");
    expect(JSON.parse(init.body as string)).toEqual({ key: "re_123" });
  });

  it("forwards the current agent bearer and organization to the UI bridge", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    setAgentSession("session-token", "org-a");

    await connectWithKey("resend", "re_123");

    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.headers).toMatchObject({
      Authorization: "Bearer session-token",
      "X-Ollama-Organization": "org-a",
    });
    expect(JSON.parse(init.body as string)).toEqual({ key: "re_123" });
  });

  it("surfaces the server explanation, e.g. login required", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response(
            JSON.stringify({ error: "o Ollama está exposto na rede" }),
            { status: 403 },
          ),
        ),
    );
    await expect(connectWithKey("github", "x")).rejects.toThrow(
      "exposto na rede",
    );
    await expect(connectWithKey("github", "  ")).rejects.toThrow(
      "Cole a chave",
    );
  });

  it("links to where each service issues keys", () => {
    expect(keyHelpURL("stripe")).toMatch(/^https:\/\/dashboard\.stripe\.com/);
    expect(keyHelpURL("gmail")).toBeUndefined();
  });
});

describe("connectHint", () => {
  it("only blames OAuth for OAuth-only services", () => {
    expect(connectHint("Gmail", "oauth")).toContain("OAuth");
    expect(connectHint("PostgreSQL", "connection_url")).toContain(
      "URL de conexão",
    );
    expect(connectHint("PostgreSQL", "connection_url")).not.toContain("OAuth");
  });
});

describe("connectWithKey self-hosted", () => {
  afterEach(() => vi.unstubAllGlobals());
  it("sends the instance URL when given", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    await connectWithKey("coolify", "k", " https://c.example.com ");
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(JSON.parse(init.body as string)).toEqual({
      key: "k",
      base_url: "https://c.example.com",
    });
  });
});
