import fs from "node:fs";
import path from "node:path";

const root = process.cwd();
const config = path.join(root, "app/ui/app/tailwind.config.js");
if (fs.existsSync(config)) {
  throw new Error("tailwind.config.js is obsolete; use Tailwind v4 CSS tokens");
}
const css = fs.readFileSync(path.join(root, "app/ui/app/src/index.css"), "utf8");
for (const token of ["--color-gray-350", "--spacing-3_5", "--spacing-4_5", ".page-title", ".page-description", ".section-title"]) {
  if (!css.includes(token)) throw new Error(`missing design token: ${token}`);
}
for (const file of ["HomePage.tsx", "ConnectorsPage.tsx", "EndpointPage.tsx"]) {
  const source = fs.readFileSync(path.join(root, "app/ui/app/src/components", file), "utf8");
  if (!source.includes("page-title")) throw new Error(`${file} does not use shared page-title token`);
}
console.log("DESIGN SYSTEM VERIFICATION PASSED: Tailwind v4 CSS tokens and shared page headings are active");
