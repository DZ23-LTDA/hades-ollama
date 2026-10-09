import { gzipSync } from "node:zlib";
import { readdir, readFile } from "node:fs/promises";

const assetsDir = new URL("../dist/assets/", import.meta.url);
const rawBudget = Number(process.env.HADES_BUNDLE_MAX_JS_BYTES ?? 2_000_000);
const gzipBudget = Number(process.env.HADES_BUNDLE_MAX_JS_GZIP_BYTES ?? 600_000);

const files = (await readdir(assetsDir)).filter((name) => name.endsWith(".js"));
if (files.length === 0) {
  throw new Error("Bundle budget: nenhum asset JavaScript foi encontrado em dist/assets");
}

const entries = await Promise.all(
  files.map(async (name) => {
    // Resolve each asset against the directory URL so the path stays correct
    // on Windows too (URL.pathname yields a leading-slash "/D:/..." that join
    // turns into "D:\D:\...").
    const data = await readFile(new URL(name, assetsDir));
    return { name, raw: data.byteLength, gzip: gzipSync(data, { level: 9 }).byteLength };
  }),
);
const entryChunks = entries.filter(({ name }) => /^index-[^/]+\.js$/.test(name));
const measured = entryChunks.length > 0 ? entryChunks : entries;
const largest = measured.toSorted((a, b) => b.raw - a.raw)[0];
console.log(`Bundle entry: ${largest.name} raw=${largest.raw} gzip=${largest.gzip}`);
console.log(`Budgets: raw<=${rawBudget} gzip<=${gzipBudget}`);

if (largest.raw > rawBudget || largest.gzip > gzipBudget) {
  throw new Error(`Bundle budget exceeded by ${largest.name}`);
}
console.log("BUNDLE_BUDGET=PASS");
