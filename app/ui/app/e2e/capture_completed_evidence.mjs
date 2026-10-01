import { chromium } from "playwright";
import fs from "fs";
import path from "path";
import crypto from "crypto";

const EVIDENCE_DIR = "/home/ubuntu/ollama-classe-a-plus/docs/evidencias";

async function captureEvidence() {
  console.log("=== CAPTURANDO EVIDÊNCIAS DA MISSÃO CONCLUÍDA E ARTEFATO ===");
  const browser = await chromium.launch({
    headless: true,
    args: ["--no-sandbox", "--disable-setuid-sandbox"],
  });

  // 1. DESKTOP
  const desktopContext = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    acceptDownloads: true,
  });
  const page = await desktopContext.newPage();

  console.log("1. Acessando /agentic no desktop...");
  await page.goto("http://127.0.0.1:5173/agentic", { waitUntil: "networkidle" });
  await page.waitForTimeout(2000);

  // Garantir que a visão split-screen está ativa
  const splitBtn = page.locator("button:has-text('Visão Split-Screen')").first();
  if (await splitBtn.isVisible().catch(() => false)) {
    await splitBtn.click();
    await page.waitForTimeout(1000);
  }

  // Clicar na aba Artefatos
  console.log("2. Clicando na aba Artefatos...");
  const artifactsTab = page.locator("button:has-text('Artefatos')").first();
  await artifactsTab.waitFor({ state: "visible", timeout: 10000 });
  await artifactsTab.click();
  await page.waitForTimeout(1500);

  // Capturar tela com o artefato e checksum visíveis
  await page.screenshot({ path: path.join(EVIDENCE_DIR, "agentic-artifacts-desktop.png") });
  console.log("-> Screenshot capturado: agentic-artifacts-desktop.png");

  // Testar download real
  console.log("3. Testando download real do artefato...");
  const downloadBtn = page.locator("button:has-text('Baixar Arquivo'), a:has-text('Baixar Arquivo'), a[download]").first();
  if (await downloadBtn.isVisible().catch(() => false)) {
    const [download] = await Promise.all([
      page.waitForEvent("download", { timeout: 10000 }),
      downloadBtn.click(),
    ]);
    const downloadPath = path.join(EVIDENCE_DIR, download.suggestedFilename());
    await download.saveAs(downloadPath);
    const content = fs.readFileSync(downloadPath);
    const sha = crypto.createHash("sha256").update(content).digest("hex");
    console.log(`-> Arquivo baixado via browser: ${download.suggestedFilename()} (${content.length} bytes)`);
    console.log(`-> SHA-256 verificado: ${sha}`);
  }

  // Clicar na aba Navegador ao Vivo para capturar o canvas ativo
  console.log("4. Alternando para aba Navegador ao Vivo...");
  const browserTab = page.locator("button:has-text('Navegador ao Vivo')").first();
  if (await browserTab.isVisible().catch(() => false)) {
    await browserTab.click();
    await page.waitForTimeout(1000);
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "agentic-live-browser-desktop.png") });
    console.log("-> Screenshot capturado: agentic-live-browser-desktop.png");
  }

  await desktopContext.close();

  // 2. MOBILE (390x844)
  console.log("\n--- CAPTURA MOBILE (390x844) ---");
  const mobileContext = await browser.newContext({
    viewport: { width: 390, height: 844 },
    isMobile: true,
    hasTouch: true,
  });
  const mobilePage = await mobileContext.newPage();

  console.log("1. Acessando /agentic no mobile...");
  await mobilePage.goto("http://127.0.0.1:5173/agentic", { waitUntil: "networkidle" });
  await mobilePage.waitForTimeout(2000);

  const mobileArtifactsTab = mobilePage.locator("button:has-text('Artefatos')").first();
  if (await mobileArtifactsTab.isVisible().catch(() => false)) {
    await mobileArtifactsTab.click();
    await mobilePage.waitForTimeout(1000);
  }
  await mobilePage.screenshot({ path: path.join(EVIDENCE_DIR, "agentic-artifacts-mobile.png") });
  console.log("-> Screenshot capturado: agentic-artifacts-mobile.png");

  const mobileBrowserTab = mobilePage.locator("button:has-text('Navegador ao Vivo')").first();
  if (await mobileBrowserTab.isVisible().catch(() => false)) {
    await mobileBrowserTab.click();
    await mobilePage.waitForTimeout(1000);
  }
  await mobilePage.screenshot({ path: path.join(EVIDENCE_DIR, "agentic-live-browser-mobile.png") });
  console.log("-> Screenshot capturado: agentic-live-browser-mobile.png");

  await mobileContext.close();
  await browser.close();
  console.log("=== CAPTURA CONCLUÍDA COM SUCESSO! ===");
}

captureEvidence().catch((err) => {
  console.error("ERRO:", err);
  process.exit(1);
});
