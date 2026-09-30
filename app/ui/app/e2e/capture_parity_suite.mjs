import { chromium } from "playwright";
import path from "path";

const EVIDENCE_DIR = "/home/ubuntu/ollama-classe-a-plus/docs/evidencias";
const BASE_URL = "http://127.0.0.1:5173";

async function run() {
  const browser = await chromium.launch({ headless: true });

  // 1. Desktop context
  const desktopContext = await browser.newContext({
    viewport: { width: 1280, height: 800 },
  });
  const page = await desktopContext.newPage();

  console.log("-> Capturando /library (Desktop)");
  await page.goto(`${BASE_URL}/library`, { waitUntil: "networkidle" });
  await page.screenshot({ path: path.join(EVIDENCE_DIR, "screen-library-desktop.png"), fullPage: false });

  console.log("-> Capturando /creations (Desktop)");
  await page.goto(`${BASE_URL}/creations`, { waitUntil: "networkidle" });
  await page.screenshot({ path: path.join(EVIDENCE_DIR, "screen-creations-desktop.png"), fullPage: false });

  console.log("-> Capturando /endpoint (Computadores) (Desktop)");
  await page.goto(`${BASE_URL}/endpoint`, { waitUntil: "networkidle" });
  await page.screenshot({ path: path.join(EVIDENCE_DIR, "screen-computers-desktop.png"), fullPage: false });

  console.log("-> Capturando /scheduled (Automações) (Desktop)");
  await page.goto(`${BASE_URL}/scheduled`, { waitUntil: "networkidle" });
  await page.screenshot({ path: path.join(EVIDENCE_DIR, "screen-automations-desktop.png"), fullPage: false });

  console.log("-> Capturando /connectors (Plugins) (Desktop)");
  await page.goto(`${BASE_URL}/connectors`, { waitUntil: "networkidle" });
  await page.screenshot({ path: path.join(EVIDENCE_DIR, "screen-plugins-desktop.png"), fullPage: false });

  console.log("-> Capturando Popover do Usuário (Desktop)");
  await page.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
  // Clicar no botão do perfil no rodapé
  const profileBtn = await page.$("button:has-text('Operador')");
  if (profileBtn) {
    await profileBtn.click();
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "screen-user-popover-desktop.png"), fullPage: false });
  }

  await desktopContext.close();

  // 2. Mobile context (390x844)
  const mobileContext = await browser.newContext({
    viewport: { width: 390, height: 844 },
    isMobile: true,
  });
  const mobilePage = await mobileContext.newPage();

  console.log("-> Capturando /library (Mobile)");
  await mobilePage.goto(`${BASE_URL}/library`, { waitUntil: "networkidle" });
  await mobilePage.screenshot({ path: path.join(EVIDENCE_DIR, "screen-library-mobile.png") });

  console.log("-> Capturando /creations (Mobile)");
  await mobilePage.goto(`${BASE_URL}/creations`, { waitUntil: "networkidle" });
  await mobilePage.screenshot({ path: path.join(EVIDENCE_DIR, "screen-creations-mobile.png") });

  console.log("-> Capturando /endpoint (Mobile)");
  await mobilePage.goto(`${BASE_URL}/endpoint`, { waitUntil: "networkidle" });
  await mobilePage.screenshot({ path: path.join(EVIDENCE_DIR, "screen-computers-mobile.png") });

  console.log("-> Capturando /scheduled (Mobile)");
  await mobilePage.goto(`${BASE_URL}/scheduled`, { waitUntil: "networkidle" });
  await mobilePage.screenshot({ path: path.join(EVIDENCE_DIR, "screen-automations-mobile.png") });

  await mobileContext.close();
  await browser.close();

  console.log("✅ Todas as evidências capturadas com sucesso!");
}

run().catch((err) => {
  console.error("Erro no teste E2E:", err);
  process.exit(1);
});
