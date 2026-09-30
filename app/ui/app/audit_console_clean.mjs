import { chromium } from "playwright";
import fs from "fs";
import path from "path";

const EVIDENCE_DIR = "/home/ubuntu/ollama-classe-a-plus/docs/evidencias";

async function audit() {
  const browser = await chromium.launch({ headless: true, args: ["--no-sandbox", "--disable-setuid-sandbox"] });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

  const errors = [];
  page.on("response", (res) => {
    if (res.status() >= 400 && !res.url().includes("favicon")) {
      errors.push({ status: res.status(), url: res.url() });
    }
  });

  await page.goto("http://127.0.0.1:5173/agentic", { waitUntil: "networkidle" });
  await page.waitForTimeout(2000);

  fs.writeFileSync(
    path.join(EVIDENCE_DIR, "browser-console-audit.json"),
    JSON.stringify({ timestamp: new Date().toISOString(), httpErrors: errors }, null, 2)
  );

  console.log("Erros HTTP capturados:", errors);
  await browser.close();
}

audit().catch(console.error);
