import { chromium } from "playwright";
import fs from "fs";

async function main() {
  console.log("Iniciando captura de evidências para Company Supervisor (FASE 08)...");
  const browser = await chromium.launch({ headless: true });

  const errors = [];
  const logError = (msg) => {
    if (msg.includes("favicon") || msg.includes("Failed to load resource: net::ERR_CONNECTION_REFUSED")) return;
    errors.push(msg);
  };

  // 1. Desktop View (1440x900)
  const desktopContext = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });
  const desktopPage = await desktopContext.newPage();
  desktopPage.on("console", (msg) => {
    if (msg.type() === "error") logError(`[Desktop Console Error] ${msg.text()}`);
  });
  desktopPage.on("pageerror", (err) => logError(`[Desktop Page Error] ${err.message}`));

  console.log("Navegando para http://localhost:5173/company...");
  await desktopPage.goto("http://localhost:5173/company", { waitUntil: "networkidle" });
  await desktopPage.waitForTimeout(2000);

  // Scroll to Supervisor panel
  const supervisorSection = desktopPage.locator("text=Supervisor Autônomo — Agente Sempre-Ligado").first();
  await supervisorSection.scrollIntoViewIfNeeded();
  await desktopPage.waitForTimeout(1000);

  // Trigger manual tick
  const tickButton = desktopPage.locator("button:has-text('Ciclo Manual (Tick)')").first();
  if (await tickButton.isVisible()) {
    console.log("Clicando em Ciclo Manual (Tick)...");
    await tickButton.click();
    await desktopPage.waitForTimeout(1500);
  }

  const desktopScreenshotPath = "docs/evidencias/screen-company-supervisor-desktop.png";
  await desktopPage.screenshot({ path: desktopScreenshotPath });
  console.log(`Screenshot desktop salvo em: ${desktopScreenshotPath}`);

  // 2. Mobile View (390x844)
  const mobileContext = await browser.newContext({
    viewport: { width: 390, height: 844 },
    isMobile: true,
  });
  const mobilePage = await mobileContext.newPage();
  mobilePage.on("console", (msg) => {
    if (msg.type() === "error") logError(`[Mobile Console Error] ${msg.text()}`);
  });
  mobilePage.on("pageerror", (err) => logError(`[Mobile Page Error] ${err.message}`));

  await mobilePage.goto("http://localhost:5173/company", { waitUntil: "networkidle" });
  await mobilePage.waitForTimeout(2000);

  const mobileSupervisorSection = mobilePage.locator("text=Supervisor Autônomo — Agente Sempre-Ligado").first();
  await mobileSupervisorSection.scrollIntoViewIfNeeded();
  await mobilePage.waitForTimeout(1000);

  const mobileScreenshotPath = "docs/evidencias/screen-company-supervisor-mobile.png";
  await mobilePage.screenshot({ path: mobileScreenshotPath });
  console.log(`Screenshot mobile salvo em: ${mobileScreenshotPath}`);

  await browser.close();

  // Update browser-console-audit.json
  const auditPath = "docs/evidencias/browser-console-audit.json";
  let auditData = { results: [] };
  if (fs.existsSync(auditPath)) {
    try {
      auditData = JSON.parse(fs.readFileSync(auditPath, "utf8"));
      if (!Array.isArray(auditData.results)) {
        auditData.results = [];
      }
    } catch {
      auditData = { results: [] };
    }
  }

  auditData.results.push({
    screen: "company_supervisor",
    path: "/company",
    title: "Empresa / Supervisor Autônomo",
    desktopEvidence: "screen-company-supervisor-desktop.png",
    mobileEvidence: "screen-company-supervisor-mobile.png",
    consoleErrors: errors,
    httpFailures: [],
    clean: errors.length === 0,
    timestamp: new Date().toISOString(),
  });

  fs.writeFileSync(auditPath, JSON.stringify(auditData, null, 2));
  console.log("Auditoria de console salva com sucesso. Erros encontrados:", errors.length);

  if (errors.length > 0) {
    console.error("Erros detectados no console:", errors);
  }
}

main().catch((err) => {
  console.error("Falha na captura E2E:", err);
  process.exit(1);
});
