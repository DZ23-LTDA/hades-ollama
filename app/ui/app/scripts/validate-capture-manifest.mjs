import { createHash } from "node:crypto";
import { readFile, stat } from "node:fs/promises";
import { dirname, resolve, relative, isAbsolute } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "../../../..");
const manifestPath = resolve(repoRoot, "docs/images/screens/class-a-plus-capture-manifest.json");
const screenshotRoot = resolve(repoRoot, "docs/images/screens");
const manifest = JSON.parse(await readFile(manifestPath, "utf8"));

if (manifest.schema !== "class-a-plus.capture.v1" || !Array.isArray(manifest.files) || manifest.files.length === 0) {
  throw new Error("invalid Class A+ screenshot manifest");
}

for (const entry of manifest.files) {
  if (!entry || typeof entry.path !== "string" || entry.path.includes("..") || entry.path.includes("\\")) {
    throw new Error(`invalid screenshot path in manifest: ${JSON.stringify(entry)}`);
  }
  const filePath = resolve(screenshotRoot, entry.path);
  const rel = relative(screenshotRoot, filePath);
  if (rel === "" || rel.startsWith("..") || isAbsolute(rel)) {
    throw new Error(`screenshot path escaped root: ${entry.path}`);
  }
  const info = await stat(filePath);
  const digest = createHash("sha256").update(await readFile(filePath)).digest("hex");
  if (info.size !== entry.bytes || digest !== entry.sha256) {
    throw new Error(`screenshot provenance mismatch for ${entry.path}: expected ${entry.bytes}/${entry.sha256}, got ${info.size}/${digest}`);
  }
}

console.log(`capture manifest: PASS (${manifest.files.length} files)`);
