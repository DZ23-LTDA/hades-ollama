import { afterEach, describe, expect, it, vi } from "vitest";
import {
  listProviders,
  saveProviderKey,
  sortProviders,
  staleModels,
  type ProviderStatus,
} from "./providers";

const provider = (name: string, configured: boolean): ProviderStatus => ({
  name,
  type: "openai-compatible",
  base_url: "https://x",
  models: 1,
  enabled: true,
  configured,
  needs_key: true,
});

describe("providers client", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("sorts ready providers first, then by name", () => {
    const sorted = sortProviders([
      provider("groq", false),
      provider("openai", true),
      provider("anthropic", false),
    ]);
    expect(sorted.map((p) => p.name)).toEqual(["openai", "anthropic", "groq"]);
  });

  it("tolerates a null provider list", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response(JSON.stringify({ providers: null }), { status: 200 }),
        ),
    );
    await expect(listProviders()).resolves.toEqual({
      configPath: "",
      providers: [],
    });
  });

  it("sends the trimmed key and surfaces server errors", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ error: "invalid credential" }), {
          status: 400,
        }),
      );
    vi.stubGlobal("fetch", fetchMock);

    await saveProviderKey("openai", "  sk-abc  ");
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe("PUT");
    expect(JSON.parse(init.body as string)).toEqual({ key: "sk-abc" });

    await expect(saveProviderKey("openai", "sk-abc")).rejects.toThrow(
      "invalid credential",
    );
    await expect(saveProviderKey("openai", "   ")).rejects.toThrow(
      "Cole a chave",
    );
  });
});

describe("staleModels", () => {
  it("flags configured models the provider no longer offers", () => {
    expect(
      staleModels({ configured: ["a", "gone"], available: ["a", "b"] }),
    ).toEqual(["gone"]);
  });

  it("does not guess when the listing failed or is empty", () => {
    expect(
      staleModels({ configured: ["a"], available: [], error: "401" }),
    ).toEqual([]);
    expect(staleModels({ configured: ["a"], available: [] })).toEqual([]);
  });
});
