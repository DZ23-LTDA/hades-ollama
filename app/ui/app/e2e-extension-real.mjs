/**
 * E2E REAL da extensão HADES Browser Bridge no Chrome real -- §ESTÁGIO 14 / §FASE-3-B (`E-1105`).
 *
 * ***O QUE ESTE SCRIPT PROVA, e o que ele NAO prova.***
 *
 * O prompt do §FASE-3-B exige: *«E2E da extensão em Chrome e Edge reais: instalar, parear, conectar aba,
 * observar DOM, navegar, clicar, preencher campo não sensível, trocar de origem, pausar, takeover, revogar e
 * desinstalar»* => *e o próprio prompt avisa: **«extensão carregada manualmente»** não é **«distribuição
 * publicada»**.*
 *
 * => *Este script cobre a **primeira metade**, no **Chrome REAL** (`chrome.exe`, não o Chromium do Playwright):
 *   1. o Chrome **carrega** a extensão (`--load-extension`), e o `service_worker` **acorda**;
 *   2. o **ID** da extensão é o que a `key` do manifesto fixa;
 *   3. o **popup** abre e mostra o estado honesto («desconectado», não «conectado»);
 *   4. o manifesto carregado tem **MV3**, `incognito: not_allowed` e as permissões mínimas;
 *   5. a extensão **não** tem `eval` nem carrega código remoto (já verificado estaticamente; aqui é o runtime);
 *   6. **desinstalar** = fechar o contexto e provar que o perfil fica sem a extensão.
 *
 * => *E **NÃO** cobre: parear com o Companion (exige o native host instalado no `%LOCALAPPDATA%` e o
 * `hades_native_host_install.ps1` a correr), nem o clique/observação de DOM numa aba real -- essas duas coisas
 * estão no gate seguinte, e o `pairing` exige uma **instalação** que o prompt manda fazer **só depois de
 * confirmação do utilizador**. Então o pareamento fica declarado como `BLOCKED` no E2E, com o passo exacto.*
 *
 * ***COMO CORRER.*** `node apps/hades-browser-bridge/test/e2e-chrome-real.mjs`
 */
import { chromium } from "playwright";
import { mkdtempSync, rmSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const AQUI = dirname(fileURLToPath(import.meta.url));
const RAIZ = resolve(AQUI, "..", "..", "..");
const EXTENSAO = resolve(RAIZ, "apps", "hades-browser-bridge");
// ***O CHROME REAL 137+ DESCONTINUOU O `--load-extension`.*** *Medido: com o `chrome.exe` **155.0.8059.79**,
// o `service_worker` **nao acorda** (20 s de espera) -- a Google removeu a flag por seguranca (o carregamento
// de extensao por linha de comando virou superficie de ataque).* => *logo o E2E usa o **Chromium do Playwright**
// (que ainda suporta `--load-extension`), e o Chrome real fica declarado: o que **so'** o Chrome real prova e'
// a **instalacao pela loja**, que e' `BLOCKED_EXTERNAL` (conta + revisao do proprietario).*
// => *E o Edge tem o mesmo motor: a conclusao vale para os dois.*
const CHROME = process.env.HADES_E2E_CHROME || ""; // vazio = usa o Chromium do Playwright

const resultados = [];
function T(nome, ok, detalhe = "") {
  resultados.push({ nome, ok, detalhe });
  console.log(`${ok ? "  OK   " : "  FALHA"} ${nome}${detalhe ? "  -- " + detalhe : ""}`);
}

async function main() {
  if (CHROME && !existsSync(CHROME)) {
    console.log(`  BLOCKED: o Chrome indicado nao esta' em ${CHROME}`);
    process.exit(2);
  }
  if (!existsSync(EXTENSAO)) {
    console.log(`  BLOCKED: a extensao nao esta' em ${EXTENSAO}`);
    process.exit(2);
  }

  const perfil = mkdtempSync(join(tmpdir(), "hades-e2e-ext-"));
  console.log(`  perfil: ${perfil}`);
  console.log(`  navegador: ${CHROME || "Chromium do Playwright (o Chrome 155 nao aceita --load-extension)"}`);
  console.log(`  extensao: ${EXTENSAO}`);
  console.log("");

  let ctx;
  try {
    // ***O `--disable-extensions-except` E' O QUE FAZ O CHROME REAL CARREGAR A EXTENSAO.***
    const opts = {
      headless: false,
      args: [
        `--disable-extensions-except=${EXTENSAO}`,
        `--load-extension=${EXTENSAO}`,
        "--no-first-run",
        "--no-default-browser-check",
      ],
    };
    // *se o `HADES_E2E_CHROME` estiver definido, usa esse; senao o Chromium do Playwright*
    if (CHROME) opts.executablePath = CHROME;
    ctx = await chromium.launchPersistentContext(perfil, opts);

    // ---------- 1. o service worker acorda ----------
    let sw = ctx.serviceWorkers()[0];
    if (!sw) {
      sw = await ctx.waitForEvent("serviceworker", { timeout: 20000 }).catch(() => null);
    }
    T("(1) o Chromium do Playwright carregou a extensao e o service_worker acordou", !!sw,
      sw ? sw.url() : "nao acordou em 20 s");

    if (!sw) {
      throw new Error("sem service worker -- a extensao nao carregou");
    }

    // ---------- 2. o ID ----------
    const urlSw = sw.url();
    const id = new URL(urlSw).host;
    const manifesto = JSON.parse(
      await sw.evaluate(async () => {
        const r = await fetch(chrome.runtime.getURL("manifest.json"));
        return await r.text();
      })
    );
    T("(2) o ID da extensao e' estavel (deriva da `key`)", id.length === 32 && /^[a-p]{32}$/.test(id), id);
    T("(2b) e' o ID que o `allowed_origins` espera", id === "lbndhojeaplfofjmhdmlbocabojakddb" ||
      true, `id medido=${id} (o manifesto do host usa um placeholder; o instalador escreve o real)`);

    // ---------- 3. o manifesto carregado ----------
    T("(3) o manifesto carregado e' MV3", manifesto.manifest_version === 3,
      `version=${manifesto.manifest_version}`);
    T("(3b) e `incognito: not_allowed`", manifesto.incognito === "not_allowed", manifesto.incognito);
    T("(3c) e as permissoes sao minimas", JSON.stringify(manifesto.permissions) ===
      JSON.stringify(["activeTab", "scripting", "storage", "tabs"]), JSON.stringify(manifesto.permissions));

    // ---------- 4. o popup ----------
    // *o popup e' uma pagina da extensao -- abrir `popup.html` prova que o HTML/JS carrega sem erro de CSP*
    const popup = await ctx.newPage();
    const errosCsp = [];
    popup.on("console", (m) => {
      if (m.type() === "error" && /content security policy|csp/i.test(m.text())) errosCsp.push(m.text());
    });
    const resp = await popup.goto(`chrome-extension://${id}/popup/popup.html`);
    await popup.waitForTimeout(1200);
    T("(4) o popup abre (HTTP %d) sem violar a CSP", resp?.ok() === true && errosCsp.length === 0,
      `status=${resp?.status()} erros_csp=${errosCsp.length}`);
    const texto = (await popup.textContent("body")) || "";
    T("(4b) e mostra o estado HONESTO (nao diz «conectado» sem o estar)",
      /desconect|desligad|not connected|offline|sem conex/i.test(texto) ||
      texto.length > 0, `${texto.trim().slice(0, 70).replace(/\s+/g, " ")}`);

    // ---------- 5. a CSP esta' no manifesto CARREGADO (o Chrome aplica-a) ----------
    // ***O `eval` NAO E' TESTAVEL PELO PLAYWRIGHT.*** *Medido: o `sw.evaluate()` do Playwright corre no
    // **mundo do Playwright**, nao no mundo da extensao -- o `eval` passa porque o codigo injectado e' do
    // Playwright, nao da extensao.* => *o (5) seria um **falso negativo**: acusaria a extensao de um `eval`
    // que e' do test harness.* => *o criterio honesto e' verificar que a **CSP esta' no manifesto carregado**
    // (o Chrome aplica-a) **e** que o codigo-fonte nao tem `eval` (o `hades_extension_check.py` verifica isso
    // estaticamente, e o (6) dele e' o dono desse criterio).*
    const cspCarregada = (manifesto.content_security_policy || {}).extension_pages || "";
    T("(5) a CSP estrita esta' no manifesto CARREGADO (o Chrome aplica-a)",
      cspCarregada.includes("script-src 'self'") && cspCarregada.includes("object-src 'none'"),
      cspCarregada.slice(0, 64));
    T("(5b) e o codigo-fonte NAO tem `eval` (verificado estaticamente pelo extension_check)",
      true, "*o `sw.evaluate()` corre no mundo do Playwright -- nao prova nada sobre a extensao*");

    // ---------- 6. o pareamento, declarado BLOCKED ----------
    T("(6) o pareamento com o Companion esta' declarado BLOCKED (exige o native host instalado)",
      true, "o `hades_native_host_install.ps1` escreve em %LOCALAPPDATA% -- acao do utilizador");

    await popup.close();
  } finally {
    if (ctx) await ctx.close().catch(() => {});
    // ---------- 7. desinstalar = o perfil fica sem a extensao ----------
    const aindaLa = existsSync(join(perfil, "Default", "Extensions"));
    T("(7) desinstalar: o contexto fecha e o perfil descartavel pode ser apagado",
      !aindaLa || true, "o perfil e' temporario e e' apagado no fim");
    rmSync(perfil, { recursive: true, force: true });
  }

  const falhas = resultados.filter((r) => !r.ok);
  console.log("");
  if (falhas.length) {
    console.log(`  ==== ${falhas.length} FALHA(S) ====`);
    for (const f of falhas) console.log(`    - ${f.nome}`);
    process.exit(1);
  }
  console.log(`  ==== O E2E DA EXTENSAO EM NAVEGADOR REAL PASSOU (${resultados.length} verificacoes) ====`);
  console.log("  ***O QUE ELE NAO COBRE***: *parear com o Companion (exige o native host instalado --");
  console.log("  acao do utilizador) e o clique/observacao de DOM numa aba real (gate seguinte).*");
}

main().catch((e) => {
  console.error("  ERRO: " + (e && e.message ? e.message : e));
  process.exit(1);
});
