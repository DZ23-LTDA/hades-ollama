import { chromium } from "playwright";
import path from "path";
import fs from "fs";

const EVIDENCE_DIR = "/home/ubuntu/ollama-classe-a-plus/docs/evidencias";

const SCREENS = [
  { path: "/", name: "home" },
  { path: "/agentic", name: "agentic" },
  { path: "/tasks", name: "tasks" },
  { path: "/scheduled", name: "scheduled" },
  { path: "/company", name: "company" },
  { path: "/connectors", name: "connectors" },
  { path: "/skills", name: "skills" },
  { path: "/library", name: "library" },
  { path: "/projects", name: "projects" },
  { path: "/settings", name: "settings" },
];

async function main() {
  console.log("=== INICIANDO AUDITORIA DE TODAS AS TELAS DO SHELL ===");
  const browser = await chromium.launch({
    headless: true,
    args: ["--no-sandbox", "--disable-setuid-sandbox"],
  });

  const httpErrors = [];
  const consoleErrors = [];

  // 1. DESKTOP (1440x900)
  console.log("\n--- TESTANDO TELAS NO DESKTOP ---");
  const desktopContext = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const desktopPage = await desktopContext.newPage();

  desktopPage.on("response", (res) => {
    if (res.status() >= 400 && !res.url().includes("favicon")) {
      httpErrors.push({ screen: "desktop", url: res.url(), status: res.status() });
      console.error(`[HTTP Error ${res.status()}]:`, res.url());
    }
  });

  desktopPage.on("console", (msg) => {
    if (msg.type() === "error" && !msg.text().includes("favicon")) {
      consoleErrors.push({ screen: "desktop", text: msg.text() });
      console.error("[Console Error]:", msg.text());
    }
  });

  for (const screen of SCREENS) {
    console.log(`-> Acessando ${screen.path} (${screen.name})...`);
    await desktopPage.goto(`http://127.0.0.1:5173${screen.path}`, { waitUntil: "networkidle" });
    await desktopPage.waitForTimeout(1000);
    const targetFile = path.join(EVIDENCE_DIR, `screen-${screen.name}-desktop.png`);
    await desktopPage.screenshot({ path: targetFile });
  }

  await desktopContext.close();

  // 2. MOBILE (390x844)
  console.log("\n--- TESTANDO TELAS NO MOBILE ---");
  const mobileContext = await browser.newContext({
    viewport: { width: 390, height: 844 },
    isMobile: true,
    hasTouch: true,
  });
  const mobilePage = await mobileContext.newPage();

  mobilePage.on("response", (res) => {
    if (res.status() >= 400 && !res.url().includes("favicon")) {
      httpErrors.push({ screen: "mobile", url: res.url(), status: res.status() });
      console.error(`[Mobile HTTP Error ${res.status()}]:`, res.url());
    }
  });

  for (const screen of SCREENS) {
    console.log(`-> [Mobile] Acessando ${screen.path} (${screen.name})...`);
    await mobilePage.goto(`http://127.0.0.1:5173${screen.path}`, { waitUntil: "networkidle" });
    await mobilePage.waitForTimeout(1000);
    const targetFile = path.join(EVIDENCE_DIR, `screen-${screen.name}-mobile.png`);
    await mobilePage.screenshot({ path: targetFile });
  }

  await mobileContext.close();
  await browser.close();

  // Relatório consolidado
  const auditReport = {
    timestamp: new Date().toISOString(),
    totalScreensTested: SCREENS.length,
    httpErrors,
    consoleErrors,
    status: httpErrors.length === 0 ? "PASSED_ZERO_ERRORS" : "FAILED_WITH_ERRORS",
  };

  fs.writeFileSync(
    path.join(EVIDENCE_DIR, "browser-console-audit.json"),
    JSON.stringify(auditReport, null, 2)
  );

  console.log("\n=== RESULTADO DA AUDITORIA DO SHELL ===");
  console.log("Total de telas testadas:", SCREENS.length);
  console.log("Erros HTTP:", httpErrors.length);
  console.log("Erros no Console:", consoleErrors.length);
  console.log("Status:", auditReport.status);

  if (httpErrors.length > 0) {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error("ERRO FATAL:", err);
  process.exit(1);
});
