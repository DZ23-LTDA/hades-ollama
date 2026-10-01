import { chromium } from "playwright";
import fs from "fs";
import path from "path";

const ROOT = "/home/ubuntu/ollama-classe-a-plus";
const EVIDENCE_DIR = path.join(ROOT, "docs/evidencias");
const AUDIT_PATH = path.join(EVIDENCE_DIR, "browser-console-audit.json");
const BASE = process.env.E2E_BASE_URL || "http://127.0.0.1:5173";
const errors = [];

function observe(page, label) {
  page.on("console", (msg) => {
    if (msg.type() === "error" && !msg.text().includes("favicon.ico")) {
      errors.push(`[${label}] console: ${msg.text()}`);
    }
  });
  page.on("pageerror", (error) => errors.push(`[${label}] pageerror: ${error.message}`));
  page.on("response", (response) => {
    if (response.url().includes("/builders/") && response.url().endsWith("/visual")) {
      if (response.status() !== 200) {
        errors.push(`[${label}] Studio visual mutation HTTP ${response.status()} ${response.url()}`);
      }
    }
    if (response.status() >= 400 && !response.url().includes("/favicon.ico")) {
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

  // Navigate to /studio
  await page.goto(`${BASE}/studio`, { waitUntil: "domcontentloaded" });
  await page.waitForTimeout(1000);

  // Verify Studio title and components
  await page.waitForSelector("h1", { timeout: 10000 });
  await page.waitForTimeout(800);

  // If on desktop, interactively add a component, undo, redo, and export
  if (suffix === "desktop") {
    // 1. Click to add a card component from palette
    const cardButton = page.locator("aside button", { hasText: "Card / Recurso" }).first();
    if (await cardButton.isVisible()) {
      await cardButton.click();
      await page.waitForTimeout(600);
      if (errors.some((error) => error.includes("Studio visual mutation HTTP"))) {
        throw new Error(`${suffix}: visual edit did not return HTTP 200`);
      }
    }

    // 2. Click Undo
    const undoButton = page.locator("button[title*='Desfazer']");
    if (await undoButton.isVisible() && await undoButton.isEnabled()) {
      await undoButton.click();
      await page.waitForTimeout(600);
    }

    // 3. Click Redo
    const redoButton = page.locator("button[title*='Refazer']");
    if (await redoButton.isVisible() && await redoButton.isEnabled()) {
      await redoButton.click();
      await page.waitForTimeout(600);
    }

    // 4. Click Exportar ZIP
    const exportButton = page.locator("button", { hasText: "Exportar ZIP" });
    if (await exportButton.isVisible()) {
      await exportButton.click();
      await page.waitForTimeout(1000);
    }
  }

  // Take screenshot
  const screenshotPath = path.join(EVIDENCE_DIR, `screen-r2-studio-edit-${suffix}.png`);
  await page.screenshot({ path: screenshotPath });

  const body = await page.locator("body").innerText();
  const upper = body.toUpperCase();
  if (!upper.includes("STUDIO") || !upper.includes("COMPONENTES")) {
    throw new Error(`${suffix}: Studio builder page content not fully visible`);
  }

  await context.close();
  await browser.close();
}

async function main() {
  fs.mkdirSync(EVIDENCE_DIR, { recursive: true });

  console.log("Capturing Desktop 1440x900...");
  await capture({ width: 1440, height: 900 }, "desktop");

  console.log("Capturing Mobile 390x844...");
  await capture({ width: 390, height: 844 }, "mobile");

  // Update browser-console-audit.json
  let audit = { results: [] };
  try {
    if (fs.existsSync(AUDIT_PATH)) {
      audit = JSON.parse(fs.readFileSync(AUDIT_PATH, "utf-8"));
    }
  } catch (e) {
    audit = { results: [] };
  }

  const entry = {
    screen: "Studio Builders Interactive Real Backend",
    path: "/studio",
    title: "Studio",
    timestamp: new Date().toISOString(),
    desktopEvidence: "screen-r2-studio-edit-desktop.png",
    mobileEvidence: "screen-r2-studio-edit-mobile.png",
    consoleErrors: errors.filter((e) => e.includes("console:")),
    httpFailures: errors.filter((e) => e.includes("HTTP")),
    errorsCount: errors.length,
    clean: errors.length === 0,
  };

  if (Array.isArray(audit)) {
    audit.push(entry);
  } else if (Array.isArray(audit.results)) {
    audit.results.push(entry);
  } else {
    audit.results = [entry];
  }

  fs.writeFileSync(AUDIT_PATH, JSON.stringify(audit, null, 2));
  if (errors.length > 0) {
    throw new Error(`Studio evidence contains live console/network failures: ${JSON.stringify(errors)}`);
  }
  console.log("Studio evidence capture complete with 0 errors!");
}

main().catch((err) => {
  console.error("Studio capture failed:", err);
  process.exit(1);
});
