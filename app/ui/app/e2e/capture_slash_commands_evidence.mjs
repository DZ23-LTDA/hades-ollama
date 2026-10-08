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
    if (msg.type() === "error" && !msg.text().includes("favicon.ico")) errors.push(`[${label}] console: ${msg.text()}`);
  });
  page.on("pageerror", (error) => errors.push(`[${label}] pageerror: ${error.message}`));
  page.on("response", (response) => {
    if (response.status() >= 500) errors.push(`[${label}] HTTP ${response.status()} ${response.url()}`);
  });
}

async function capture(viewport, suffix) {
  const browser = await chromium.launch({ headless: true, args: ["--no-sandbox", "--disable-dev-shm-usage"] });
  const context = await browser.newContext({ viewport, isMobile: suffix === "mobile" });
  const page = await context.newPage();
  observe(page, suffix);
  await page.goto(`${BASE}/`, { waitUntil: "domcontentloaded" });
  const composer = page.locator("#home-objective");
  await composer.waitFor({ state: "visible", timeout: 10000 });
  await composer.fill("/");
  await page.waitForTimeout(300);
  await page.screenshot({ path: path.join(EVIDENCE_DIR, `screen-slash-menu-${suffix}.png`) });
  if (!(await page.getByRole("listbox", { name: "Comandos disponíveis" }).isVisible())) throw new Error(`${suffix}: slash menu did not open`);
  await composer.fill("/goal preparar evidência local");
  await composer.press("Enter");
  await page.waitForURL(/\/agentic\?/, { timeout: 10000 });
  await page.waitForTimeout(1200);
  await page.screenshot({ path: path.join(EVIDENCE_DIR, `screen-slash-goal-${suffix}.png`) });
  const body = await page.locator("body").innerText();
  if (!body.includes("Console de Missões")) throw new Error(`${suffix}: Console de Missões not visible after /goal`);
  await context.close();
  await browser.close();
}

async function main() {
  fs.mkdirSync(EVIDENCE_DIR, { recursive: true });
  await capture({ width: 1440, height: 900 }, "desktop");
  await capture({ width: 390, height: 844 }, "mobile");
  let audit = { screens: {} };
  if (fs.existsSync(AUDIT_PATH)) audit = JSON.parse(fs.readFileSync(AUDIT_PATH, "utf8"));
  audit.screens = audit.screens || {};
  audit.screens.slash_commands = {
    status: errors.length ? 500 : 200,
    clean_console: errors.length === 0,
    errors,
    desktop_menu_png: "docs/evidencias/screen-slash-menu-desktop.png",
    mobile_menu_png: "docs/evidencias/screen-slash-menu-mobile.png",
    desktop_goal_png: "docs/evidencias/screen-slash-goal-desktop.png",
    mobile_goal_png: "docs/evidencias/screen-slash-goal-mobile.png",
    captured_at: new Date().toISOString(),
  };
  fs.writeFileSync(AUDIT_PATH, JSON.stringify(audit, null, 2) + "\n");
  console.log(JSON.stringify(audit.screens.slash_commands, null, 2));
  if (errors.length) process.exitCode = 1;
}

main().catch((error) => { console.error(error); process.exit(1); });
