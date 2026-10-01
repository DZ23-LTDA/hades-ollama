import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("Studio preview isolation", () => {
  it("uses a restricted iframe sandbox without allow-same-origin", () => {
    const studio = readFileSync(resolve(__dirname, "StudioCanvasPage.tsx"), "utf8");
    const artifacts = readFileSync(resolve(__dirname, "ArtifactsViewer.tsx"), "utf8");

    expect(studio).toContain('sandbox="allow-scripts"');
    expect(artifacts).toContain('sandbox="allow-scripts"');
    expect(studio).not.toContain("allow-same-origin");
    expect(artifacts).not.toContain("allow-same-origin");
  });

  it("downloads preview/export bytes through the authenticated client", () => {
    const studio = readFileSync(resolve(__dirname, "StudioCanvasPage.tsx"), "utf8");
    const client = readFileSync(resolve(__dirname, "../lib/agenticClient.ts"), "utf8");

    expect(studio).toContain("agentFetchBlob");
    expect(client).toContain("export async function agentFetchBlob");
    expect(client).toContain("...agentHeaders()");
  });
});
