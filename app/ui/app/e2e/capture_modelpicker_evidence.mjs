import { chromium } from "playwright";
import fs from "fs";
import path from "path";

const EVIDENCE_DIR = "/home/ubuntu/ollama-classe-a-plus/docs/evidencias";
const AUDIT_JSON = path.join(EVIDENCE_DIR, "browser-console-audit.json");

async function main() {
  const browser = await chromium.launch({
    headless: true,
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });

  const errors = [];

  // 1. Desktop capture (1440x900)
  console.log("-> Capturing Desktop ModelPicker at /c/new (1440x900)...");
  const desktopContext = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });
  const page = await desktopContext.newPage();

  page.on("console", (msg) => {
    const text = msg.text();
    if (msg.type() === "error" && !text.includes("favicon.ico")) {
      errors.push(text);
    }
  });

  page.on("pageerror", (err) => {
    errors.push(err.message);
  });

  await page.goto("http://127.0.0.1:5173/c/new", { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(1500);

  // Click on ModelPicker dropdown button
  const modelPickerButton = page.locator('button[title="Select model"]').first();
  await modelPickerButton.waitFor({ state: "visible", timeout: 8000 });
  await modelPickerButton.click();
  await page.waitForTimeout(800);

  await page.screenshot({
    path: path.join(EVIDENCE_DIR, "screen-modelpicker-desktop.png"),
    fullPage: false,
  });

  await desktopContext.close();

  // 2. Mobile capture (390x844)
  console.log("-> Capturing Mobile ModelPicker at /c/new (390x844)...");
  const mobileContext = await browser.newContext({
    viewport: { width: 390, height: 844 },
    isMobile: true,
  });
  const mobilePage = await mobileContext.newPage();

  mobilePage.on("console", (msg) => {
    const text = msg.text();
    if (msg.type() === "error" && !text.includes("favicon.ico")) {
      errors.push(`[mobile] ${text}`);
    }
  });

  await mobilePage.goto("http://127.0.0.1:5173/c/new", { waitUntil: "domcontentloaded" });
  await mobilePage.waitForTimeout(1500);

  const mobilePickerBtn = mobilePage.locator('button[title="Select model"]').first();
  await mobilePickerBtn.waitFor({ state: "visible", timeout: 8000 });
  await mobilePickerBtn.click();
  await mobilePage.waitForTimeout(800);

  await mobilePage.screenshot({
    path: path.join(EVIDENCE_DIR, "screen-modelpicker-mobile.png"),
    fullPage: false,
  });

  await mobileContext.close();
  await browser.close();

  // Update browser-console-audit.json
  let existingAudit = { screens: {} };
  if (fs.existsSync(AUDIT_JSON)) {
    try {
      existingAudit = JSON.parse(fs.readFileSync(AUDIT_JSON, "utf8"));
    } catch {
      existingAudit = { screens: {} };
    }
  }

  existingAudit.screens = existingAudit.screens || {};
  existingAudit.screens["modelpicker"] = {
    status: errors.length === 0 ? 200 : 500,
    clean_console: errors.length === 0,
    errors: errors,
    desktop_png: "docs/evidencias/screen-modelpicker-desktop.png",
    mobile_png: "docs/evidencias/screen-modelpicker-mobile.png",
    captured_at: new Date().toISOString(),
  };

  fs.writeFileSync(AUDIT_JSON, JSON.stringify(existingAudit, null, 2));
  console.log("-> ModelPicker screenshots and console audit updated successfully.");
}

main().catch((err) => {
  console.error("Error capturing evidence:", err);
  process.exit(1);
});
