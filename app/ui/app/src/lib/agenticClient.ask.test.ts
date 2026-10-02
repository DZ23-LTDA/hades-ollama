import { afterEach, describe, expect, it, vi } from "vitest";

import { askProjectDocuments, fetchDiagnostics } from "./agenticClient";

describe("askProjectDocuments", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("calls the project ask endpoint with the encoded query and parses citations", async () => {
    const payload = {
      project_id: "p1",
      query: "férias",
      context: "[1] (rh.pdf#1)\n30 dias\n\nPergunta: férias",
      citations: [{ index: 1, source: "rh.pdf#1", score: 0.98 }],
      grounded: true,
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify(payload), { status: 200, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("window", { dispatchEvent: vi.fn() });
    vi.stubGlobal("fetch", fetchMock);

    const result = await askProjectDocuments("p1", "férias");

    expect(result.grounded).toBe(true);
    expect(result.citations).toHaveLength(1);
    expect(result.citations[0].source).toBe("rh.pdf#1");

    const calledURL = String(fetchMock.mock.calls[0][0]);
    expect(calledURL).toContain("/api/agent/v1/projects/p1/ask");
    expect(calledURL).toContain("q=f%C3%A9rias");
  });

  it("surfaces grounded=false so the UI can say it found nothing", async () => {
    const payload = { project_id: "p1", query: "x", context: "Não há fontes relevantes...", citations: [], grounded: false };
    vi.stubGlobal("window", { dispatchEvent: vi.fn() });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify(payload), { status: 200, headers: { "Content-Type": "application/json" } })));

    const result = await askProjectDocuments("p1", "x");
    expect(result.grounded).toBe(false);
    expect(result.citations).toHaveLength(0);
  });
});

describe("fetchDiagnostics", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("fetches the diagnostics snapshot", async () => {
    const payload = { product: "Hades", integrations: { agent_database_url: true } };
    vi.stubGlobal("window", { dispatchEvent: vi.fn() });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify(payload), { status: 200, headers: { "Content-Type": "application/json" } })));
    const result = await fetchDiagnostics();
    expect(result.product).toBe("Hades");
  });
});
