import { afterEach, describe, expect, it, vi } from "vitest";

import { askProjectDocuments, fetchDiagnostics, importMissionAttachments } from "./agenticClient";

describe("askProjectDocuments", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("calls the project ask endpoint with the encoded query and parses citations", async () => {
    const payload = {
      project_id: "p1",
      query: "férias",
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
    const payload = { project_id: "p1", query: "x", citations: [], grounded: false };
    vi.stubGlobal("window", { dispatchEvent: vi.fn() });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify(payload), { status: 200, headers: { "Content-Type": "application/json" } })));

    const result = await askProjectDocuments("p1", "x");
    expect(result.grounded).toBe(false);
    expect(result.citations).toHaveLength(0);
  });

  it("uploads file bytes as multipart without putting names or content in the URL", async () => {
    const payload = { project: { id: "project-1" }, source: "attachments" };
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(payload), { status: 201, headers: { "Content-Type": "application/json" } }));
    vi.stubGlobal("window", { dispatchEvent: vi.fn() });
    vi.stubGlobal("fetch", fetchMock);

    const result = await importMissionAttachments(
      [{ filename: "private-notes.txt", data: new Uint8Array([65, 66, 67]), type: "text/plain" }],
      "Anexos da missão",
    );

    expect(result.project.id).toBe("project-1");
    const [calledURL, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(calledURL).toContain("/api/agent/v1/projects/import/attachments");
    expect(calledURL).not.toContain("private-notes");
    expect(init.headers).not.toHaveProperty("Content-Type");
    expect(init.body).toBeInstanceOf(FormData);
    const body = init.body as FormData;
    expect(body.get("name")).toBe("Anexos da missão");
    expect((body.get("files") as Blob).size).toBe(3);
    expect(new TextDecoder().decode(await (body.get("files") as Blob).arrayBuffer())).toBe("ABC");
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
