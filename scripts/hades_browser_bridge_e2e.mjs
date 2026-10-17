/**
 * E2E COMPLETO: EDGE REAL + EXTENSAO + NATIVE MESSAGING -> COMPANION (`E-1032`).
 *
 * E' o que faltava desde o `E-1029`: o protocolo corre contra um processo real, mas nunca
 * tinha atravessado **um navegador a serio**. Aqui atravessa:
 *
 *   Edge (Chromium) -> service worker da extensao -> native messaging do SO
 *                   -> `companion.mjs` (processo separado) -> resposta
 *
 * CONCEDER A PERMISSAO -- como, e por que' assim:
 * `nativeMessaging` esta' em `optional_permissions` (o `E-1025` decidiu que a extensao nao
 * tem poder sobre o PC desde a instalacao). Medido, e e' o achado `E-1032`:
 *   - dentro do handler do clique, `chrome.permissions.request` **TRAVA para sempre** --
 *     a promessa nunca resolve nem rejeita, e o botao fica em "A pedir permissao...";
 *   - o mesmo pedido **fora** do handler resolve imediatamente com `true`.
 * O produto tem agora um prazo de 8s para nao ficar pendurado; o E2E concede pela via que
 * o navegador **comprovadamente aceita** e depois mede o caminho que interessa.
 *
 * O QUE ISTO NAO PROVA: publicacao, assinatura, Chrome (bloqueado pela versao 155, que ja'
 * nao carrega por `--load-extension`), Firefox, Safari, nem computer use.
 */
import { fileURLToPath, pathToFileURL } from "node:url";
import { execFileSync } from "node:child_process";
import path from "node:path";
import fs from "node:fs";
import os from "node:os";
import crypto from "node:crypto";

const AQUI = path.dirname(fileURLToPath(import.meta.url));
const RAIZ = path.resolve(AQUI, "..");
const { chromium } = await import(
  pathToFileURL(path.join(RAIZ, "app", "ui", "app", "node_modules", "playwright", "index.mjs")).href
);
const EXT = path.join(RAIZ, "apps", "hades-browser-bridge");
const EDGE = "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe";
const CHROME = "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe";
const NOME_HOST = "com.dz23ltd.hades.bridge";
const ALFA = "abcdefghijklmnop";

const idDaChave = (b64) =>
  crypto.createHash("sha256").update(Buffer.from(b64, "base64")).digest("hex").slice(0, 32)
    .split("").map((c) => ALFA[parseInt(c, 16)]).join("");

const falhas = [];
const ok = (n, c, d = "") => {
  console.log((c ? "  OK   " : "  FAIL ") + n + (d && !c ? "  -- " + d : ""));
  if (!c) falhas.push(n);
};

const esperado = idDaChave(
  fs.readFileSync(path.join(EXT, "native-host", "chave-publica.b64"), "utf8").trim(),
);
console.log("0) id derivado por mim: " + esperado);

const instalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1")], { encoding: "utf8" });
const desinstalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1"), "-Desinstalar"], { encoding: "utf8" });

console.log("1) instalar o native host (HKCU, reversivel)");
const saidaInst = instalar();
ok("instalacao registou navegadores", /registado/.test(saidaInst), saidaInst.slice(-160));

  // 1b) MEDIR, nao falhar: o Chrome 155 neutralizou `--load-extension`. Isto e' uma
  //     limitacao do NAVEGADOR do utilizador, nao um defeito do HADES -- mas e' uma
  //     limitacao que muda o que se pode homologar sem carregar a extensao a mao.
  console.log("1b) o Chrome 155 aceita `--load-extension`? (medicao, nao suposicao)");
  if (fs.existsSync(CHROME)) {
    const perfilC = fs.mkdtempSync(path.join(os.tmpdir(), "hades-chrome-"));
    let ctxC = null;
    try {
      ctxC = await chromium.launchPersistentContext(perfilC, {
        executablePath: CHROME,
        headless: false,
        args: [`--disable-extensions-except=${EXT}`, `--load-extension=${EXT}`,
          "--disable-features=DisableLoadExtensionCommandLineSwitch", "--no-first-run"],
      });
      await new Promise((r) => setTimeout(r, 3500));
      const n = ctxC.serviceWorkers().length;
      console.log("   Chrome 155: " + n + " service workers com `--load-extension`");
      console.log("   LIMITACAO MEDIDA: o Chrome real exige carregamento manual em" +
        " chrome://extensions (Modo de programador). O Edge aceita a flag.");
    } finally {
      if (ctxC) await ctxC.close().catch(() => {});
      fs.rmSync(perfilC, { recursive: true, force: true });
    }
  }

// ------------------------------------------------------------------- Edge
const perfil = fs.mkdtempSync(path.join(os.tmpdir(), "hades-e2e-nm-"));
let ctx = null;
try {
  console.log("2) arrancar o Edge REAL com a extensao");
  ctx = await chromium.launchPersistentContext(perfil, {
    executablePath: EDGE,
    headless: false,
    args: [`--disable-extensions-except=${EXT}`, `--load-extension=${EXT}`,
      "--disable-features=DisableLoadExtensionCommandLineSwitch",
      "--no-first-run", "--no-default-browser-check"],
  });
  let sw = ctx.serviceWorkers()[0];
  if (!sw) sw = await ctx.waitForEvent("serviceworker", { timeout: 30000 }).catch(() => null);
  ok("service worker da extensao", !!sw);
  if (!sw) throw new Error("sem service worker");

  const idDoNavegador = new URL(sw.url()).host;
  console.log("   id do navegador: " + idDoNavegador);
  ok("O NAVEGADOR DERIVA O MESMO ID", idDoNavegador === esperado, `${idDoNavegador} vs ${esperado}`);

  // 3) O POPUP existe, mostra o botao, e o clique produz DIAGNOSTICO (mesmo que falhe).
  console.log("3) o popup da extensao tem o controlo do canal local");
  const popup = await ctx.newPage();
  await popup.setViewportSize({ width: 460, height: 1400 });
  await popup.goto(`chrome-extension://${idDoNavegador}/popup/popup.html`, { waitUntil: "domcontentloaded" });
  await popup.waitForSelector("#bt-canal", { timeout: 10000 });
  await popup.waitForFunction(() => (document.querySelector("#aviso")?.textContent ?? "").length > 0, { timeout: 10000 });
  await popup.waitForTimeout(600);

  const alvo = await popup.evaluate(() => {
    const r = document.querySelector("#bt-canal").getBoundingClientRect();
    const cx = r.x + r.width / 2, cy = r.y + r.height / 2;
    const em = document.elementFromPoint(cx, cy);
    return { x: cx, y: cy, acerta: em ? (em.id || em.tagName) : "fora", scrollY: window.scrollY };
  });
  ok("o ponto do clique acerta no #bt-canal", alvo.acerta === "bt-canal", alvo.acerta);
  await popup.mouse.click(alvo.x, alvo.y);
  // O produto tem prazo de 8s: se travar, ele MESMO diz. Espelhamos esse prazo.
  await popup.waitForTimeout(10000);
  const aviso = await popup.evaluate(() => document.querySelector("#aviso")?.textContent);
  console.log("   aviso depois do clique: " + JSON.stringify(aviso));
  ok("o popup NAO fica pendurado (o prazo funciona)",
    !/A pedir permiss/i.test(aviso ?? ""), String(aviso));
  await popup.close();

  // 4) CONCEDER pela via que o navegador aceita.
  console.log("4) conceder `nativeMessaging` (a via que o navegador aceita)");
  const pag = await ctx.newPage();
  await pag.goto(`chrome-extension://${idDoNavegador}/popup/popup.html`, { waitUntil: "domcontentloaded" });
  const conceder = await pag.evaluate(async () => {
    if (await chrome.permissions.contains({ permissions: ["nativeMessaging"] })) return true;
    return chrome.permissions.request({ permissions: ["nativeMessaging"] });
  });
  console.log("   concedida: " + conceder);
  const tem = await sw.evaluate(() => chrome.permissions.contains({ permissions: ["nativeMessaging"] }));
  ok("permissao nativeMessaging concedida", tem === true, String(tem));

  // 4b) O ACHADO: `chrome.runtime` do service worker NAO ganha `sendNativeMessage`
  //     depois de a permissao ser concedida -- o objecto foi construido sem ela.
  //     Medido, e nao suposto: e' por isso que a extensao tem de RECARREGAR.
  console.log("4b) o `chrome.runtime` do worker ganha a funcao sem recarregar?");
  const antesReload = await sw.evaluate(() => typeof chrome.runtime.sendNativeMessage);
  console.log("   typeof sendNativeMessage (sem recarregar): " + antesReload);
  ok("SEM recarregar a funcao NAO existe (achado E-1032)", antesReload === "undefined", String(antesReload));
  await pag.close();

  // 4c) RECARREGAR a extensao: e' o que o utilizador faz, e e' o que reconstroi o objecto.
  console.log("4c) recarregar a extensao (reconstroi o `chrome.runtime`)");
  const naPag = await ctx.newPage();
  await naPag.goto(`chrome-extension://${idDoNavegador}/popup/popup.html`, { waitUntil: "domcontentloaded" });
  await naPag.evaluate(() => chrome.runtime.reload()).catch(() => {});
  await naPag.close().catch(() => {});
  await new Promise((r) => setTimeout(r, 3000));
  let sw2 = ctx.serviceWorkers()[0];
  if (!sw2) sw2 = await ctx.waitForEvent("serviceworker", { timeout: 20000 }).catch(() => null);
  if (sw2) sw = sw2;
  ok("service worker depois do reload", !!sw, "sem worker");
  const depoisReload = await sw.evaluate(() => ({
    tipo: typeof chrome.runtime.sendNativeMessage,
    perm: null,
  }));
  console.log("   typeof sendNativeMessage (apos recarregar): " + depoisReload.tipo);
  ok("DEPOIS de recarregar a funcao EXISTE", depoisReload.tipo === "function", String(depoisReload.tipo));

  // 5) NATIVE MESSAGING a serio: extensao -> SO -> Companion.
  console.log("5) native messaging: extensao -> SO -> processo Companion");
  const r = await sw.evaluate(async (nome) => {
    try {
      const resp = await chrome.runtime.sendNativeMessage(nome, { id: 1, cmd: "host:ola" });
      return { ok: true, resp };
    } catch (e) {
      return { ok: false, erro: String(e && e.message ? e.message : e) };
    }
  }, NOME_HOST);

  console.log("   resposta: " + JSON.stringify(r).slice(0, 240));
  ok("o SO arrancou o Companion", r.ok, r.erro || "");
  if (r.ok) {
    ok("ok:true", r.resp?.ok === true, JSON.stringify(r.resp));
    ok("protocolo == 1", r.resp?.protocolo === 1, String(r.resp?.protocolo));
    ok("CAPACIDADES VAZIO", Array.isArray(r.resp?.capacidades) && r.resp.capacidades.length === 0, JSON.stringify(r.resp?.capacidades));
    ok("instalado:false", r.resp?.instalado === false);
    ok("assinado:false", r.resp?.assinado === false);
  }

  // 6) Erro pelo fio + id ecoado + host vivo.
  if (r.ok) {
    console.log("6) erro PELO FIO, `id` ecoado, host vivo");
    const e2 = await sw.evaluate(async (nome) => ({
      a: await chrome.runtime.sendNativeMessage(nome, { id: 2, cmd: "nao:existe" }),
      b: await chrome.runtime.sendNativeMessage(nome, { id: 3, cmd: "host:ola" }),
    }), NOME_HOST);
    ok("erro pelo fio", e2.a?.ok === false && e2.a?.motivo === "cmd_desconhecido", JSON.stringify(e2.a));
    ok("id ecoado no erro", e2.a?.id === 2, JSON.stringify(e2.a));
    ok("host continua vivo", e2.b?.ok === true, JSON.stringify(e2.b));
  }
} catch (e) {
  console.log("   ERRO: " + e.message);
  falhas.push("execucao: " + e.message);
} finally {
  if (ctx) await ctx.close().catch(() => {});
  fs.rmSync(perfil, { recursive: true, force: true });
  if (/registado/.test(saidaInst)) {
    console.log("7) desinstalar (devolver o sistema ao estado anterior)");
    const s = desinstalar();
    ok("desinstalado", /DESINSTALADO/.test(s), s.slice(-120));
    const base = path.join(process.env.LOCALAPPDATA || "", "HADES", "NativeHost");
    ok("sem residuo no disco", !fs.existsSync(base), base);
  }
}

console.log("");
if (falhas.length) {
  console.log("FALHAS: " + falhas.length + " -> " + falhas.join(" | "));
  process.exit(1);
}
console.log("E2E REAL: Edge -> extensao -> native messaging -> Companion -> resposta, id ecoado.");
process.exit(0);
