import type { ProjectImportResult } from "@/lib/agenticClient";

// describeImport summarizes what an import indexed and, when files were skipped,
// makes that visible so the user knows an unindexed document (unsupported format
// or over the size limit) will not answer questions instead of silently failing.
export function describeImport(result: ProjectImportResult): string {
  const base = `${result.indexed_files} arquivos e ${result.indexed_memories} trechos`;
  if (result.ignored_files > 0) {
    const noun = result.ignored_files === 1 ? "arquivo ignorado" : "arquivos ignorados";
    return `${base} (${result.ignored_files} ${noun} — formato não indexável ou acima de 4 MB)`;
  }
  return base;
}
