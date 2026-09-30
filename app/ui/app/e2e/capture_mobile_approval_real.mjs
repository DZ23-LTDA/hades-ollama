import { chromium } from "playwright";
import path from "path";
import fs from "fs";

const EVIDENCE_DIR = "/home/ubuntu/ollama-classe-a-plus/docs/evidencias";

async function main() {
  console.log("=== CAPTURANDO APROVAÇÃO REAL NO MOBILE (390x844) ===");
  const browser = await chromium.launch({
    headless: true,
    args: ["--no-sandbox", "--disable-setuid-sandbox"],
  });

  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    isMobile: true,
    hasTouch: true,
  });

  const page = await context.newPage();

  // Acessar Home no mobile
  console.log("1. Acessando Home no mobile...");
  await page.goto("http://127.0.0.1:5173/", { waitUntil: "networkidle" });
  await page.waitForTimeout(1000);

  // Preencher objetivo que aciona approval
  console.log("2. Enviando missão no mobile...");
  const input = page.locator("textarea#home-objective-input, textarea").first();
  await input.fill("Gravar relatório confidencial e navegar");
  await page.waitForTimeout(500);

  const startBtn = page.locator("button:has-text('Criar missão'), button:has-text('Iniciar'), button[type='submit']").first();
  if (await startBtn.isVisible()) {
    await startBtn.click();
  } else {
    await input.press("Enter");
  }

  // Aguardar transição para /agentic
  console.log("3. Aguardando console agentic no mobile...");
  await page.waitForURL(/.*agentic.*/, { timeout: 15000 });
  await page.waitForTimeout(2000);

  // Aguardar card de aprovação aparecer
  console.log("4. Aguardando card de aprovação in-line (AWAITING_APPROVAL) no mobile...");
  const approvalCard = page.locator("textarea[placeholder*='justificativa'], textarea[id^='approval-reason']").first();
  await approvalCard.waitFor({ state: "visible", timeout: 20000 });
  await page.waitForTimeout(1500);

  // Capturar screenshot mobile do estado AWAITING_APPROVAL com o card e botões visíveis
  const targetPath = path.join(EVIDENCE_DIR, "agentic-approval-mobile.png");
  await page.screenshot({ path: targetPath });
  console.log("-> Screenshot capturado com sucesso:", targetPath);

  await context.close();
  await browser.close();
  console.log("=== CAPTURA MOBILE FINALIZADA ===");
}

main().catch((err) => {
  console.error("ERRO:", err);
  process.exit(1);
});
