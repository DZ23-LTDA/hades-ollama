import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createProvider,
  listProviders,
  mergeProviderPresets,
  parseModelIds,
  providerPresetHint,
  PROVIDER_PRESETS,
  saveProviderKey,
  sortProviders,
  staleModels,
  type ProviderStatus,
  type ServerProviderPreset,
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
      presets: [],
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

describe("guided presets", () => {
  const serverPreset = (
    overrides: Partial<ServerProviderPreset>,
  ): ServerProviderPreset => ({
    id: "openai",
    name: "OpenAI",
    type: "openai-compatible",
    base_url: "https://api.openai.com/v1",
    paths: ["/v1/chat/completions"],
    requires_key: true,
    ...overrides,
  });

  it("prefers the canonical server presets and keeps Personalizado", () => {
    const options = mergeProviderPresets([
      serverPreset({
        id: "ollama",
        name: "Ollama (local)",
        base_url: "http://127.0.0.1:11434",
        local: true,
        requires_key: false,
      }),
      serverPreset({
        id: "vllm",
        name: "vLLM (local)",
        base_url: "http://127.0.0.1:8000/v1",
        local: true,
        requires_key: false,
      }),
    ]);
    expect(options.map((preset) => preset.id)).toEqual([
      "ollama",
      "vllm",
      "custom",
    ]);
    expect(options[0]).toMatchObject({
      label: "Ollama (local)",
      name: "ollama",
      base_url: "http://127.0.0.1:11434",
      local: true,
      requires_key: false,
    });
    expect(options[0].hint).toContain("Servidor local");
    expect(options.at(-1)?.id).toBe("custom");
  });

  it("falls back to the bundled list when the server exposes nothing", () => {
    for (const empty of [undefined, null, []]) {
      const options = mergeProviderPresets(empty);
      expect(options).toEqual(PROVIDER_PRESETS);
    }
  });

  it("drops duplicates and nameless entries from the server list", () => {
    const options = mergeProviderPresets([
      serverPreset({ id: "openai" }),
      serverPreset({ id: "openai", name: "Duplicado" }),
      serverPreset({ id: "  " }),
    ]);
    expect(options.map((preset) => preset.id)).toEqual(["openai", "custom"]);
  });

  it("explains credential and local expectations without exposing a key", () => {
    expect(
      providerPresetHint(serverPreset({ api_key_env: "OPENAI_API_KEY" })),
    ).toBe("Exige credencial · variável OPENAI_API_KEY");
    expect(
      providerPresetHint(
        serverPreset({
          id: "lmstudio",
          local: true,
          requires_key: false,
          api_key_env: undefined,
          notes: "Porta padrão.",
        }),
      ),
    ).toBe("Servidor local (HTTP em loopback) · Porta padrão.");
    expect(
      providerPresetHint(serverPreset({ requires_key: false })),
    ).toBeUndefined();
  });
});

describe("createProvider", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("posts the provider and returns the created status", async () => {
    const created: ProviderStatus = {
      name: "openai",
      type: "openai-compatible",
      base_url: "https://api.openai.com/v1",
      api_key_env: "OPENAI_API_KEY",
      models: 2,
      enabled: true,
      configured: true,
      needs_key: true,
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify(created), { status: 201 }),
      );
    vi.stubGlobal("fetch", fetchMock);

    const result = await createProvider({
      name: "openai",
      base_url: "https://api.openai.com/v1",
      models: ["gpt-4o", "gpt-4o-mini"],
      api_key: "sk-secret",
    });
    expect(result).toEqual(created);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/api/v1/providers");
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toMatchObject({
      name: "openai",
      base_url: "https://api.openai.com/v1",
      models: ["gpt-4o", "gpt-4o-mini"],
      api_key: "sk-secret",
    });
  });

  it("surfaces the server error message", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response(
            JSON.stringify({ error: 'já existe um provedor chamado "openai"' }),
            { status: 409 },
          ),
        ),
    );
    await expect(
      createProvider({
        name: "openai",
        base_url: "https://api.openai.com/v1",
        models: ["gpt-4o"],
      }),
    ).rejects.toThrow("já existe um provedor");
  });
});

describe("parseModelIds", () => {
  it("splits on commas and newlines, trimming and de-duplicating", () => {
    expect(parseModelIds("gpt-4o, gpt-4o-mini\n gpt-4o \n\n")).toEqual([
      "gpt-4o",
      "gpt-4o-mini",
    ]);
    expect(parseModelIds("   ")).toEqual([]);
  });
});

describe("PROVIDER_PRESETS", () => {
  it("includes known services and marks Anthropic's type", () => {
    const byId = new Map(PROVIDER_PRESETS.map((p) => [p.id, p]));
    expect(byId.get("openai")?.base_url).toBe("https://api.openai.com/v1");
    expect(byId.get("anthropic")?.type).toBe("anthropic");
    expect(byId.get("custom")?.base_url).toBe("");
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
