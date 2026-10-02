import { act, create, type ReactTestRenderer, type ReactTestInstance } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { downloadBackupArchive, fetchDiagnostics } from "@/lib/agenticClient";
import { BACKUP_ERROR_MESSAGE, DataToolsPanel } from "./DataToolsPanel";

vi.mock("@/lib/agenticClient", () => ({
  downloadBackupArchive: vi.fn(),
  fetchDiagnostics: vi.fn(),
}));

const mockBackup = vi.mocked(downloadBackupArchive);
const mockDiagnostics = vi.mocked(fetchDiagnostics);

function textContent(node: ReactTestInstance): string {
  return node.children.map((child) => (typeof child === "string" ? child : textContent(child))).join("");
}

let clickSpy: ReturnType<typeof vi.fn>;
let lastAnchor: { href: string; download: string; click: () => void; remove: () => void };

beforeEach(() => {
  vi.clearAllMocks();
  vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
  clickSpy = vi.fn();
  lastAnchor = { href: "", download: "", click: clickSpy, remove: vi.fn() };
  // The test environment is node (no DOM); stub the minimal document/URL that
  // triggerDownload touches. react-test-renderer renders to a tree, not the DOM,
  // so it does not use these.
  vi.stubGlobal("document", {
    createElement: vi.fn(() => lastAnchor),
    body: { appendChild: vi.fn() },
  });
  vi.stubGlobal("URL", { createObjectURL: vi.fn(() => "blob:mock"), revokeObjectURL: vi.fn() });
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

async function clickButton(label: string): Promise<ReactTestRenderer> {
  let renderer!: ReactTestRenderer;
  await act(async () => {
    renderer = create(<DataToolsPanel />);
    await Promise.resolve();
  });
  const button = renderer.root.findAllByType("button").find((candidate) => textContent(candidate).includes(label));
  if (!button) throw new Error(`button ${label} not found`);
  await act(async () => {
    button.props.onClick();
    await Promise.resolve();
    await Promise.resolve();
  });
  return renderer;
}

describe("DataToolsPanel", () => {
  it("downloads a backup archive with a dated filename", async () => {
    mockBackup.mockResolvedValue(new Blob(["backup"], { type: "application/gzip" }));
    await clickButton("Baixar backup");
    expect(mockBackup).toHaveBeenCalledTimes(1);
    expect(clickSpy).toHaveBeenCalledTimes(1);
    expect(lastAnchor.download).toContain("hades-backup");
    expect(lastAnchor.download).toContain(".tar.gz");
  });

  it("exports diagnostics as a JSON file", async () => {
    mockDiagnostics.mockResolvedValue({ product: "Hades" });
    await clickButton("Exportar diagnóstico");
    expect(mockDiagnostics).toHaveBeenCalledTimes(1);
    expect(clickSpy).toHaveBeenCalledTimes(1);
    expect(lastAnchor.download).toBe("hades-diagnostico.json");
  });

  it("shows an error message when the backup fails", async () => {
    mockBackup.mockRejectedValue(new Error("boom"));
    const renderer = await clickButton("Baixar backup");
    expect(textContent(renderer.root)).toContain(BACKUP_ERROR_MESSAGE);
  });
});
