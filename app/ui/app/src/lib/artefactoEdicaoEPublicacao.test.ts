import { afterEach, describe, expect, it, vi } from "vitest";
import { editarArtefacto, publicarArtefacto } from "./agenticClient";

// FATIA 1 / I1b: o contrato das duas chamadas que a UI faz.
// O caminho e o metodo sao o que o servidor exige; se mudarem aqui, o teste avisa antes do utilizador.
const fetchOriginal = globalThis.fetch;
let capturado: { url: string; init?: RequestInit } | null = null;

function mockFetch(corpo: unknown) {
  capturado = null;
  globalThis.fetch = vi.fn(async (url: unknown, init?: RequestInit) => {
    capturado = { url: String(url), init };
    return new Response(JSON.stringify(corpo), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  }) as unknown as typeof fetch;
}

afterEach(() => {
  globalThis.fetch = fetchOriginal;
  capturado = null;
});

describe("publicarArtefacto", () => {
  it("faz POST no caminho de publish e devolve o URL", async () => {
    mockFetch({ url: "/api/agent/v1/published/a1-v3/index.html", artifact_id: "a1", version: 3 });

    const r = await publicarArtefacto("m1", "a1");

    expect(r.url).toBe("/api/agent/v1/published/a1-v3/index.html");
    expect(capturado?.url).toContain("/api/agent/v1/missions/m1/artifacts/a1/publish");
    expect((capturado?.init?.method ?? "").toUpperCase()).toBe("POST");
  });

  it("NEGATIVO: escapa identificadores, para nao construir caminho com barra", async () => {
    mockFetch({ url: "/x", artifact_id: "a1", version: 1 });

    await publicarArtefacto("m/1", "a 1");

    expect(capturado?.url).toContain("m%2F1");
    expect(capturado?.url).toContain("a%201");
  });
});

describe("editarArtefacto", () => {
  it("faz PUT com content_base64 e expected_version, e SEM campo de caminho", async () => {
    mockFetch({ mission_id: "m1", artifact_id: "a1", version: 4, sha256: "abc", size: 3 });

    await editarArtefacto("m1", "a1", "<h1>x</h1>", 3);

    expect((capturado?.init?.method ?? "").toUpperCase()).toBe("PUT");
    const corpo = JSON.parse(String(capturado?.init?.body));
    expect(corpo.expected_version).toBe(3);
    expect(typeof corpo.content_base64).toBe("string");
    // O servidor NAO aceita caminho: o corpo nao pode ter essa chave.
    expect(Object.keys(corpo)).not.toContain("path");
  });
});
