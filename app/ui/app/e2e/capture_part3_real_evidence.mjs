import { chromium } from "playwright";
import fs from "fs";
import path from "path";
import crypto from "crypto";

const EVIDENCE_DIR = "/home/ubuntu/ollama-classe-a-plus/docs/evidencias";
const BASE_URL = "http://127.0.0.1:5173";

fs.mkdirSync(EVIDENCE_DIR, { recursive: true });

const screens = [
  { name: "library", path: "/library", title: "Biblioteca" },
  { name: "creations", path: "/creations", title: "Criações" },
  { name: "automations", path: "/scheduled", title: "Automações" },
  { name: "connectors", path: "/connectors", title: "Conectores" },
  { name: "endpoint", path: "/endpoint", title: "Computadores & Endpoint" },
];

const consoleAudit = {
  timestamp: new Date().toISOString(),
  environment: {
    baseUrl: BASE_URL,
    backendUrl: "http://127.0.0.1:11434",
  },
  results: [],
  hasErrors: false,
};

async function main() {
  console.log("=== INICIANDO CAPTURA DE EVIDÊNCIAS REAIS - PARTE 3 ===");
  const browser = await chromium.launch({ headless: true });

  for (const screen of screens) {
    console.log(`\n--- Testando tela: ${screen.title} (${screen.path}) ---`);
    const pageErrors = [];
    const httpFailures = [];

    // 1. DESKTOP (1440x900)
    const desktopContext = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });
    const desktopPage = await desktopContext.newPage();

    desktopPage.on("console", (msg) => {
      const type = msg.type();
      const text = msg.text();
      if (type === "error" && !text.includes("favicon")) {
        pageErrors.push(`[Console Error] ${text}`);
      }
    });

    desktopPage.on("response", (res) => {
      const status = res.status();
      const url = res.url();
      if (status >= 400 && !url.includes("favicon")) {
        httpFailures.push(`[HTTP ${status}] ${url}`);
      }
    });

    await desktopPage.goto(`${BASE_URL}${screen.path}`, { waitUntil: "networkidle" });
    await desktopPage.waitForTimeout(1000); // aguardar renderização dos dados do fetch

    const desktopFile = path.join(EVIDENCE_DIR, `screen-${screen.name}-desktop.png`);
    await desktopPage.screenshot({ path: desktopFile, fullPage: false });
    console.log(`✓ Salvo Desktop: ${desktopFile}`);
    await desktopContext.close();

    // 2. MOBILE (390x844)
    const mobileContext = await browser.newContext({
      viewport: { width: 390, height: 844 },
      isMobile: true,
      hasTouch: true,
    });
    const mobilePage = await mobileContext.newPage();

    mobilePage.on("console", (msg) => {
      const type = msg.type();
      const text = msg.text();
      if (type === "error" && !text.includes("favicon")) {
        pageErrors.push(`[Mobile Console Error] ${text}`);
      }
    });

    mobilePage.on("response", (res) => {
      const status = res.status();
      const url = res.url();
      if (status >= 400 && !url.includes("favicon")) {
        httpFailures.push(`[Mobile HTTP ${status}] ${url}`);
      }
    });

    await mobilePage.goto(`${BASE_URL}${screen.path}`, { waitUntil: "networkidle" });
    await mobilePage.waitForTimeout(1000);

    const mobileFile = path.join(EVIDENCE_DIR, `screen-${screen.name}-mobile.png`);
    await mobilePage.screenshot({ path: mobileFile, fullPage: false });
    console.log(`✓ Salvo Mobile: ${mobileFile}`);
    await mobileContext.close();

    const screenAudit = {
      screen: screen.name,
      path: screen.path,
      title: screen.title,
      desktopEvidence: `screen-${screen.name}-desktop.png`,
      mobileEvidence: `screen-${screen.name}-mobile.png`,
      consoleErrors: pageErrors,
      httpFailures: httpFailures,
      clean: pageErrors.length === 0 && httpFailures.length === 0,
    };

    if (!screenAudit.clean) {
      consoleAudit.hasErrors = true;
    }

    consoleAudit.results.push(screenAudit);
  }

  await browser.close();

  // Salvar auditoria
  const auditPath = path.join(EVIDENCE_DIR, "browser-console-audit.json");
  fs.writeFileSync(auditPath, JSON.stringify(consoleAudit, null, 2), "utf8");
  console.log(`\n✓ Relatório de auditoria salvo em: ${auditPath}`);
  console.log(`Total de telas auditadas: ${screens.length}`);
  console.log(`Audit Clean: ${!consoleAudit.hasErrors}`);
}

main().catch((err) => {
  console.error("Erro fatal na suite E2E:", err);
  process.exit(1);
});
