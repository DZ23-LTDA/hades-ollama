import { describe, expect, it } from "vitest";
import { describeImport } from "@/lib/importSummary";
import type { ProjectImportResult } from "@/lib/agenticClient";

function result(overrides: Partial<ProjectImportResult>): ProjectImportResult {
  return {
    project: { id: "proj_1", name: "Proj", root: "/tmp/proj" } as ProjectImportResult["project"],
    source: "zip",
    worktree_path: "/tmp/proj",
    branch: "main",
    indexed_files: 0,
    indexed_memories: 0,
    ignored_files: 0,
    files: [],
    archive_sha256: "",
    state: "IMPORTED_INDEXED",
    notice: "",
    ...overrides,
  };
}

describe("describeImport", () => {
  it("summarizes indexed counts without mentioning ignored files when none were skipped", () => {
    const text = describeImport(result({ indexed_files: 3, indexed_memories: 12, ignored_files: 0 }));
    expect(text).toContain("3 arquivos");
    expect(text).toContain("12 trechos");
    expect(text).not.toContain("ignorado");
  });

  it("surfaces skipped files so the user is not left guessing", () => {
    const text = describeImport(result({ indexed_files: 2, indexed_memories: 5, ignored_files: 4 }));
    expect(text).toContain("4 arquivos ignorados");
  });

  it("uses the singular form for a single skipped file", () => {
    const text = describeImport(result({ indexed_files: 1, indexed_memories: 1, ignored_files: 1 }));
    expect(text).toContain("1 arquivo ignorado");
    expect(text).not.toContain("arquivos ignorados");
  });
});
