import { chromium } from "playwright";
import path from "path";

const EVIDENCE_DIR = "/home/ubuntu/ollama-classe-a-plus/docs/evidencias";

async function main() {
  const browser = await chromium.launch({ headless: true, args: ["--no-sandbox", "--disable-setuid-sandbox"] });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

  await page.goto("http://127.0.0.1:5173/agentic", { waitUntil: "networkidle" });
  await page.waitForTimeout(2000);

  // Aba artefatos
  const artTab = page.locator("button:has-text('Artefatos')").first();
  await artTab.click();
  await page.waitForTimeout(1000);

  // Clicar em Exportar
  const exportBtn = page.locator("button:has-text('Exportar')").first();
  if (await exportBtn.isVisible()) {
    await exportBtn.click();
    await page.waitForTimeout(1000);
  }

  await page.screenshot({ path: path.join(EVIDENCE_DIR, "agentic-artifacts-download-desktop.png") });
  console.log("-> Screenshot capturado: agentic-artifacts-download-desktop.png");

  // Mobile
  const mobilePage = await browser.newPage({ viewport: { width: 390, height: 844 }, isMobile: true });
  await mobilePage.goto("http://127.0.0.1:5173/agentic", { waitUntil: "networkidle" });
  await mobilePage.waitForTimeout(2000);
  const mArtTab = mobilePage.locator("button:has-text('Artefatos')").first();
  await mArtTab.click();
  await mobilePage.waitForTimeout(1000);
  const mExportBtn = mobilePage.locator("button:has-text('Exportar')").first();
  if (await mExportBtn.isVisible()) {
    await mExportBtn.click();
    await mobilePage.waitForTimeout(1000);
  }
  await mobilePage.screenshot({ path: path.join(EVIDENCE_DIR, "agentic-artifacts-download-mobile.png") });
  console.log("-> Screenshot capturado: agentic-artifacts-download-mobile.png");

  await browser.close();
}
main().catch(console.error);
