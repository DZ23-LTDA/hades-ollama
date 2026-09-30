import { chromium } from "playwright";
import fs from "fs";
import path from "path";

const ROOT = "/home/ubuntu/ollama-classe-a-plus";
const EVIDENCE_DIR = path.join(ROOT, "docs/evidencias");
const AUDIT_PATH = path.join(EVIDENCE_DIR, "browser-console-audit.json");
const BASE = "http://127.0.0.1:5173";
const errors = [];

function observe(page, label) {
  page.on("console", (msg) => {
    if (msg.type() === "error" && !msg.text().includes("favicon.ico")) {
      errors.push(`[${label}] console: ${msg.text()}`);
    }
  });
  page.on("pageerror", (error) => errors.push(`[${label}] pageerror: ${error.message}`));
  page.on("response", (response) => {
    if (response.status() >= 500) {
      errors.push(`[${label}] HTTP ${response.status()} ${response.url()}`);
    }
  });
}

async function capture(viewport, suffix) {
  const browser = await chromium.launch({
    headless: true,
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });
  const context = await browser.newContext({
    viewport,
    isMobile: suffix === "mobile",
  });
  const page = await context.newPage();
  observe(page, suffix);

  await page.goto(`${BASE}/connectors`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(800);

  // Click on "WhatsApp Gateway" category tab
  const waTab = page.locator("button", { hasText: "WhatsApp Gateway" });
  await waTab.waitFor({ state: "visible", timeout: 10000 });
  await waTab.click();
  await page.waitForTimeout(800);

  // Verify WhatsApp Gateway panel is rendered
  const title = page.locator("h2", { hasText: "WhatsApp Gateway" });
  await title.waitFor({ state: "visible", timeout: 10000 });

  // Take screenshot
  const screenshotPath = path.join(EVIDENCE_DIR, `screen-whatsapp-gateway-${suffix}.png`);
  await page.screenshot({ path: screenshotPath });

  const body = await page.locator("body").innerText();
  if (!body.includes("WhatsApp Gateway") || !body.includes("Allowlist")) {
    throw new Error(`${suffix}: WhatsApp Gateway Panel content not fully visible`);
  }

  await context.close();
  await browser.close();
}

async function main() {
  fs.mkdirSync(EVIDENCE_DIR, { recursive: true });

  // Pre-seed an allowlist contact via backend API
  try {
    await fetch("http://127.0.0.1:11434/api/agent/v1/whatsapp/allowlist", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        phone_number: "5511999999999",
        name: "Operador Dono",
        role: "owner",
        allowed: true,
      }),
    });
  } catch (err) {
    console.warn("Could not pre-seed allowlist:", err);
  }

  await capture({ width: 1440, height: 900 }, "desktop");
  await capture({ width: 390, height: 844 }, "mobile");

  let audit = { screens: {} };
  if (fs.existsSync(AUDIT_PATH)) {
    audit = JSON.parse(fs.readFileSync(AUDIT_PATH, "utf8"));
  }
  audit.screens = audit.screens || {};
  audit.screens.whatsapp_gateway = {
    status: errors.length ? 500 : 200,
    clean_console: errors.length === 0,
    errors,
    desktop_png: "docs/evidencias/screen-whatsapp-gateway-desktop.png",
    mobile_png: "docs/evidencias/screen-whatsapp-gateway-mobile.png",
    captured_at: new Date().toISOString(),
  };

  fs.writeFileSync(AUDIT_PATH, JSON.stringify(audit, null, 2) + "\n");
  console.log(JSON.stringify(audit.screens.whatsapp_gateway, null, 2));

  if (errors.length) process.exitCode = 1;
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
