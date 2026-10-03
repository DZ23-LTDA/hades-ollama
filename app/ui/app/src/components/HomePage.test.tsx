import type { ReactNode } from "react";
import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { HomePage } from "./HomePage";
import { importMissionAttachments } from "@/lib/agenticClient";

vi.mock("@/lib/agenticClient", () => ({ importMissionAttachments: vi.fn() }));
vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, to }: { children: ReactNode; to: string }) => <a href={to}>{children}</a>,
}));
vi.mock("@/components/Logo", () => ({ default: () => <span>Logo</span> }));
vi.mock("@/components/FirstModelCard", () => ({ FirstModelCard: () => <div /> }));
vi.mock("@/components/ModelPicker", () => ({ ModelPicker: () => <div /> }));
vi.mock("@/components/SlashCommandMenu", () => ({ SlashCommandMenu: () => null }));
vi.mock("@/components/ImportProjectDialog", () => ({ ImportProjectDialog: () => null }));
vi.mock("@/components/FileUpload", () => ({
  FileUpload: ({ children, onFilesAdded }: {
    children: ReactNode;
    onFilesAdded: (files: Array<{ filename: string; data: Uint8Array; type?: string }>, errors: Array<{ filename: string; error: string }>) => void;
  }) => (
    <div>
      {children}
      <button
        type="button"
        id="mock-add-file"
        onClick={() => onFilesAdded([{ filename: "private-notes.txt", data: new Uint8Array([65, 66, 67]), type: "text/plain" }], [])}
      >
        Add test file
      </button>
    </div>
  ),
}));

const mockImport = vi.mocked(importMissionAttachments);

async function renderHome(): Promise<ReactTestRenderer> {
  let renderer!: ReactTestRenderer;
  await act(async () => {
    renderer = create(<HomePage />);
    await Promise.resolve();
  });
  return renderer;
}

describe("HomePage mission start", () => {
  beforeEach(() => {
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    vi.stubGlobal("window", { location: { assign: vi.fn() } });
    mockImport.mockResolvedValue({ project: { id: "project-1" }, source: "attachments" } as Awaited<ReturnType<typeof importMissionAttachments>>);
  });

  afterEach(() => vi.unstubAllGlobals());

  it("uploads attachment bytes before navigation and passes only the project ID", async () => {
    const renderer = await renderHome();
    await act(async () => {
      renderer.root.findByProps({ id: "mock-add-file" }).props.onClick();
    });
    await act(async () => {
      renderer.root.findByProps({ "aria-label": "Objetivo da nova tarefa" }).props.onChange({ target: { value: "Resuma os documentos" } });
    });
    await act(async () => {
      renderer.root.findByProps({ "aria-label": "Iniciar no Console agentic" }).props.onClick();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mockImport).toHaveBeenCalledOnce();
    expect(mockImport.mock.calls[0][0][0].filename).toBe("private-notes.txt");
    const assign = window.location.assign as ReturnType<typeof vi.fn>;
    const destination = String(assign.mock.calls[0][0]);
    expect(destination).toContain("project_id=project-1");
    expect(destination).toContain("autorun=true");
    expect(destination).not.toContain("private-notes");
    expect(destination).not.toContain("ABC");
  });

  it("does not autorun a mission when workspace write is explicitly allowed", async () => {
    const renderer = await renderHome();
    await act(async () => {
      renderer.root.findByProps({ "aria-label": "Objetivo da nova tarefa" }).props.onChange({ target: { value: "Edite o projeto" } });
    });
    const checkbox = renderer.root.findAllByType("input").find((node) => node.props.type === "checkbox");
    expect(checkbox).toBeTruthy();
    await act(async () => {
      checkbox!.props.onChange({ target: { checked: true } });
    });
    await act(async () => {
      renderer.root.findByProps({ "aria-label": "Iniciar no Console agentic" }).props.onClick();
      await Promise.resolve();
    });

    const assign = window.location.assign as ReturnType<typeof vi.fn>;
    const destination = String(assign.mock.calls[0][0]);
    expect(destination).toContain("allow_write=true");
    expect(destination).not.toContain("autorun=true");
  });
});
