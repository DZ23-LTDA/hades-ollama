import { chromium } from "playwright";
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdir, readFile, stat, writeFile } from "node:fs/promises";
import { dirname, isAbsolute, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "../../../..");
const defaultOutputDir = resolve(repoRoot, "docs/images/screens");
const baseURL = process.env.UI_BASE_URL ?? "http://127.0.0.1:4173";
const requestedOutputDir = process.env.SCREEN_OUTPUT ?? defaultOutputDir;
const outputDir = isAbsolute(requestedOutputDir) ? requestedOutputDir : resolve(repoRoot, requestedOutputDir);
const explicitOutput = Boolean(process.env.SCREEN_OUTPUT);
const viewport = { width: 1440, height: 900 };
const routes = ["/", "/projects", "/library", "/scheduled", "/skills", "/plugins", "/tasks", "/agentic", "/settings", "/company"];

function isWithin(parent, child) {
  const path = relative(parent, child);
  return path === "" || (path !== ".." && !path.startsWith(`..${process.platform === "win32" ? "\\" : "/"}`) && !isAbsolute(path));
}

function currentGitSHA() {
  if (process.env.BUILD_SHA?.trim()) return process.env.BUILD_SHA.trim();
  try {
    return execFileSync("git", ["rev-parse", "HEAD"], { cwd: repoRoot, encoding: "utf8" }).trim();
  } catch {
    return "unknown";
  }
}

function validateBaseURL() {
  let parsed;
  try {
    parsed = new URL(baseURL);
  } catch {
    throw new Error(`UI_BASE_URL is invalid: ${baseURL}`);
  }
  if (!parsed.hostname || !["http:", "https:"].includes(parsed.protocol)) {
    throw new Error("UI_BASE_URL must be an HTTP(S) URL with a hostname");
  }
}

async function visible(locator) {
  try {
    return (await locator.count()) > 0 && (await locator.first().isVisible());
  } catch {
    return false;
  }
}

async function interactSafely(page, route) {
  const fields = [page.locator("main input"), page.locator("main textarea"), page.locator("main select"), page.locator("input"), page.locator("textarea"), page.locator("select")];
  for (const field of fields) {
    const count = await field.count();
    for (let index = 0; index < count; index += 1) {
      const element = field.nth(index);
      if (!(await element.isVisible().catch(() => false))) continue;
      const type = (await element.getAttribute("type").catch(() => ""))?.toLowerCase() ?? "";
      if (["checkbox", "radio", "file", "hidden", "button", "submit"].includes(type)) continue;
      if (await element.isEditable().catch(() => false)) {
        await element.fill("captura observável");
        await element.fill("");
        return "field-fill-clear";
      }
      await element.click();
      await page.keyboard.press("Escape").catch(() => {});
      return "field-focus";
    }
  }

  const safeButtons = [
    /nova tarefa/i,
    /novo projeto/i,
    /atualizar/i,
    /refresh/i,
    /mais/i,
    /abrir/i,
    /fechar/i,
    /selecionar um modelo/i,
    /select a model/i,
    /criar slides/i,
    /criar site/i,
    /design/i,
    /criar jogos/i,
  ];
  for (const pattern of safeButtons) {
    const button = page.getByRole("button", { name: pattern }).first();
    if (!(await visible(button))) continue;
    await button.click();
    await page.keyboard.press("Escape").catch(() => {});
    return `button:${pattern}`;
  }

  const tabs = page.getByRole("tab").first();
  if (await visible(tabs)) {
    await tabs.click();
    return "tab-click";
  }

  await page.keyboard.press("Tab");
  const focused = await page.evaluate(() => {
    const element = document.activeElement;
    return Boolean(element && element !== document.body && element !== document.documentElement);
  });
  if (focused) return "keyboard-tab-focus";

  throw new Error(`no safe interaction found for ${route}`);
}

async function waitForStablePage(page, diagnostics, route) {
  await page.locator("body").waitFor({ state: "visible", timeout: 15000 });
  await page.waitForFunction(() => {
    const bodyText = document.body?.innerText?.trim() ?? "";
    const mainText = document.querySelector("main")?.textContent?.trim() ?? "";
    return bodyText.length >= 80 && mainText.length >= 40;
  }, undefined, { timeout: 15000 });
  await page.evaluate(async () => {
    if (document.fonts?.ready) await document.fonts.ready;
  });
  await page.waitForFunction(() => {
    const busy = document.querySelectorAll('[aria-busy="true"], [data-loading="true"], [data-skeleton="true"]').length;
    const visibleOverlay = [...document.querySelectorAll('[role="dialog"], [data-overlay="true"]')].some((node) => {
      const style = getComputedStyle(node);
      return style.display !== "none" && style.visibility !== "hidden" && Number(style.opacity || 1) > 0;
    });
    return busy === 0 && !visibleOverlay;
  }, undefined, { timeout: 15000 });
  if (diagnostics.pageErrors.length || diagnostics.consoleErrors.length || diagnostics.requestFailures.length || diagnostics.unexpectedHTTPFailures.length) {
    throw new Error(`unexpected diagnostics on ${route}: ${JSON.stringify(diagnostics)}`);
  }
}

function expectedHTTPFailure(url, status) {
  const pathname = new URL(url).pathname;
  // 404 = endpoint ausente neste build. 501 = endpoint ausente POR DECISAO: o servidor
  // local responde `501 Not Implemented` no `POST /api/v1/settings` enquanto nao houver
  // persistencia, e ha' um ratchet a garantir isso (server/desktop_local_settings_ratchet_test.go,
  // E-ratchet de 2026-10-15). A ferramenta de captura tem de conhecer as duas grafias da
  // mesma ausencia -- recusar 501 aqui obrigaria a "implementar" um endpoint so' para tirar
  // uma fotografia.
  const expected = {
    "/api/me": new Set([401, 403]),
    "/api/v1/settings": new Set([404, 501]),
    "/api/v1/chats": new Set([404]),
    "/api/v1/cloud": new Set([404]),
    "/api/v1/inference-compute": new Set([404]),
    "/api/v1/model/upstream": new Set([404, 405, 501]),
  }[pathname];
  return expected?.has(status) ?? false;
}

function expectedRequestFailure(url, errorText) {
  return new URL(url).pathname === "/api/me" && errorText === "net::ERR_ABORTED";
}

function attachDiagnostics(page) {
  const diagnostics = { pageErrors: [], expectedPageErrors: [], consoleErrors: [], requestFailures: [], expectedRequestFailures: [], httpFailures: [], expectedHTTPFailures: [], unexpectedHTTPFailures: [] };
  // A UI lanca (e o browser reporta como pageerror) o corpo do 501 deliberado de
  // `POST /api/v1/settings` -- ver `expectedHTTPFailure` acima e o ratchet em
  // server/desktop_local_settings_ratchet_test.go. Separá-lo evita que a ferramenta de
  // captura confunda "servidor local nao persiste configuracao" com "pagina avariada".
  const DELIBERATE_LOCAL_SETTINGS_ERROR = /atendido pelo aplicativo desktop/i;
  page.on("pageerror", (error) => {
    const text = String(error);
    if (DELIBERATE_LOCAL_SETTINGS_ERROR.test(text)) diagnostics.expectedPageErrors.push(text);
    else diagnostics.pageErrors.push(text);
  });
  page.on("console", (message) => {
    if (message.type() === "error" && !message.text().startsWith("Failed to load resource:")) diagnostics.consoleErrors.push(message.text());
  });
  page.on("requestfailed", (request) => {
    const errorText = request.failure()?.errorText ?? "failed";
    const failure = `${request.method()} ${request.url()} ${errorText}`;
    if (expectedRequestFailure(request.url(), errorText)) diagnostics.expectedRequestFailures.push(failure);
    else diagnostics.requestFailures.push(failure);
  });
  page.on("response", (response) => {
    if (response.status() < 400) return;
    const failure = { status: response.status(), url: response.url() };
    diagnostics.httpFailures.push(failure);
    if (expectedHTTPFailure(response.url(), response.status())) diagnostics.expectedHTTPFailures.push(failure);
    else diagnostics.unexpectedHTTPFailures.push(failure);
  });
  return diagnostics;
}

validateBaseURL();
if (!explicitOutput && !isWithin(repoRoot, outputDir)) {
  throw new Error(`SCREEN_OUTPUT must stay inside the current checkout: ${outputDir}`);
}
await mkdir(outputDir, { recursive: true });

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport, deviceScaleFactor: 1 });
const captures = [];
try {
  for (const route of routes) {
    const diagnostics = attachDiagnostics(page);
    const startedAt = new Date().toISOString();
    await page.goto(new URL(route, baseURL).toString(), { waitUntil: "domcontentloaded", timeout: 30000 });
    await waitForStablePage(page, diagnostics, route);
    const interaction = await interactSafely(page, route);
    await waitForStablePage(page, diagnostics, route);
    const screenshotPath = resolve(outputDir, `class-a-plus-${route.slice(1) || "home"}.png`);
    await page.screenshot({ path: screenshotPath, fullPage: true });
    const screenshotInfo = await stat(screenshotPath);
    if (screenshotInfo.size < 16 * 1024) {
      throw new Error(`screenshot is unexpectedly small for ${route}: ${screenshotInfo.size} bytes`);
    }
    const screenshotSHA256 = createHash("sha256").update(await readFile(screenshotPath)).digest("hex");
    captures.push({
      route,
      screenshot: screenshotPath,
      bytes: screenshotInfo.size,
      sha256: screenshotSHA256,
      interaction,
      title: await page.title(),
      h1: await page.locator("h1").first().textContent().catch(() => ""),
      capturedAtUTC: startedAt,
      diagnostics,
    });
    console.log(`${route} interaction=${interaction} title=${JSON.stringify(captures.at(-1).title)}`);
  }
} finally {
  await browser.close();
}

const manifest = {
  schema: "class-a-plus.capture.v1",
  capturedAtUTC: new Date().toISOString(),
  source: {
    baseURL,
    repoRoot,
    gitSHA: currentGitSHA(),
    buildID: process.env.BUILD_ID ?? "unknown",
    version: process.env.APP_VERSION ?? "unknown",
  },
  viewport,
  expectedHTTPFailurePolicy: "Only local no-account / known base-UI bridge gaps and aborted auth probes are classified as expected; assets, JavaScript, other request failures and agentic API failures remain fatal.",
  outputDir,
  // `files` e' o contrato lido por `scripts/validate-capture-manifest.mjs` (e o que o
  // manifesto commitado sempre teve): caminho RELATIVO a esta pasta + bytes + sha256.
  // `captures` fica como detalhe por rota; o validador exige o primeiro.
  files: captures.map((c) => ({ path: c.screenshot.split(/[\\/]/).pop(), bytes: c.bytes, sha256: c.sha256 })),
  captures,
  limitations: [
    "A capture proves only the observed local UI state and does not validate external providers, accounts, devices, deployment, or store review.",
    "The chosen README Home image supplied by the maintainer is not replaced by this script unless explicitly recaptured and reviewed.",
  ],
};
await writeFile(resolve(outputDir, "class-a-plus-capture-manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`, "utf8");
console.log(`captured=${captures.length} manifest=${resolve(outputDir, "class-a-plus-capture-manifest.json")}`);
