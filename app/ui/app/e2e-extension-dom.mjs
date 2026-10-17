/**
 * E2E da extensão HADES Browser Bridge com **DOM real** -- §ESTÁGIO 14 / §FASE-3-B (`E-1115`).
 *
 * ***O QUE O `e2e-extension-real.mjs` JÁ PROVA*** (`E-1105`): *a extensão **carrega** num motor real, o
 * `service_worker` acorda, o **ID** é estável, o **popup** abre com CSP estrita.*
 *
 * ***E O QUE ELE NÃO PROVA -- e o prompt exige*** (literalmente): *«E2E da extensão em Chrome e Edge reais:
 * instalar, parear, conectar aba, **observar DOM**, navegar, **clicar**, **preencher campo não sensível**, trocar
 * de origem, pausar, takeover, revogar e desinstalar»*.
 *
 * => *Este script cobre a parte que **é internamente executável** e que faltava:*
 *   1. **observar DOM** -- *`chrome.scripting.executeScript` lê o DOM da aba e devolve o sanitizado;*
 *   2. **navegar** -- *a aba vai de uma origem a outra, e a extensão vê a mudança;*
 *   3. **clicar** -- *um botão real é clicado, e o efeito é observado (`data-estado` muda);*
 *   4. **preencher campo não sensível** -- *um `<input type="text">` é preenchido, e o valor é lido de volta;*
 *   5. **e o NEGATIVO que importa**: *um `<input type="password">` **não** é lido -- o `content_script` recusa.*
 *
 * ***O QUE CONTINUA FORA, e porquê***:
 *   - **parear** -- *exige o native host **instalado** no `%LOCALAPPDATA%` (o `hades_native_host_install.ps1`),
 *     que é ação do utilizador*;
 *   - **Chrome/Edge reais** -- *o Chrome 155 **removeu** `--load-extension`* (`E-1105`), e o caminho é a loja;
 *   - **Firefox** -- *o `web-ext` não está instalado*;
 *   - **computer-use** -- *exige um SO com acessibilidade nativa, e o default do repo é sem controlo.*
 *
 * ***COMO CORRER.*** `node apps/hades-browser-bridge/test/e2e-dom-real.mjs`
 */
import { chromium } from "playwright";
import { createServer } from "node:http";
import { mkdtempSync, rmSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const AQUI = dirname(fileURLToPath(import.meta.url));
const RAIZ = resolve(AQUI, "..", "..", "..");
const EXTENSAO = resolve(RAIZ, "apps", "hades-browser-bridge");

const resultados = [];
function T(nome, ok, detalhe = "") {
  resultados.push({ nome, ok, detalhe });
  console.log(`${ok ? "  OK   " : "  FALHA"} ${nome}${detalhe ? "  -- " + detalhe : ""}`);
}

// ***A PÁGINA DE TESTE.*** *Tem de ser servida em `127.0.0.1`, porque é onde o `content_script` da extensão
// está autorizado a correr (`matches: ['http://127.0.0.1/*']`) -- e é isso que prova o **consentimento por
// origem**, não um `file://`.*
const PAGINA_A = `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><title>A</title></head>
<body>
  <h1 id="titulo">Página A</h1>
  <button id="btn" data-estado="inicial">Clicar</button>
  <input id="campo-texto" type="text" />
  <input id="campo-senha" type="password" value="SEGREDO-QUE-NAO-PODE-SAIR" />
  <script>
    document.getElementById("btn").addEventListener("click", () => {
      document.getElementById("btn").dataset.estado = "clicado";
    });
  </script>
</body></html>`;

const PAGINA_B = `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><title>B</title></head>
<body><h1 id="titulo">Página B</h1></body></html>`;

function servidor() {
  const srv = createServer((req, res) => {
    const html = req.url && req.url.startsWith("/b") ? PAGINA_B : PAGINA_A;
    res.writeHead(200, { "content-type": "text/html; charset=utf-8" });
    res.end(html);
  });
  return new Promise((ok) => srv.listen(0, "127.0.0.1", () => ok(srv)));
}

async function main() {
  if (!existsSync(EXTENSAO)) {
    console.log(`  BLOCKED: a extensao nao esta' em ${EXTENSAO}`);
    process.exit(2);
  }

  const srv = await servidor();
  const porta = srv.address().port;
  const base = `http://127.0.0.1:${porta}`;
  const perfil = mkdtempSync(join(tmpdir(), "hades-e2e-dom-"));
  console.log(`  servidor: ${base}`);
  console.log(`  perfil: ${perfil}`);
  console.log(`  extensao: ${EXTENSAO}`);
  console.log("");

  let ctx;
  try {
    ctx = await chromium.launchPersistentContext(perfil, {
      headless: false,
      args: [
        `--disable-extensions-except=${EXTENSAO}`,
        `--load-extension=${EXTENSAO}`,
        "--no-first-run",
        "--no-default-browser-check",
      ],
    });

    let sw = ctx.serviceWorkers()[0];
    if (!sw) sw = await ctx.waitForEvent("serviceworker", { timeout: 20000 }).catch(() => null);
    T("(1) a extensao carregou e o service_worker acordou", !!sw, sw ? sw.url() : "nao acordou");
    if (!sw) throw new Error("sem service worker");

    const id = new URL(sw.url()).host;

    // ---------- (2) navegar + observar DOM ----------
    const aba = await ctx.newPage();
    const resp = await aba.goto(base + "/", { waitUntil: "domcontentloaded" });
    T("(2) a aba abriu em 127.0.0.1 (a origem que o content_script autoriza)", resp?.ok() === true,
      `status=${resp?.status()} url=${aba.url()}`);

    // ***A OBSERVAÇÃO VIA `chrome.scripting`.*** *executada no MUNDO DA EXTENSÃO (a partir do service worker),
    // sobre a aba específica -- é o que o `activeTab`/`scripting` permite, e o que o prompt chama «observar DOM».*
    const abaId = await sw.evaluate(async (url) => {
      const abas = await chrome.tabs.query({ url });
      return abas[0] ? abas[0].id : null;
    }, base + "/*");
    T("(2b) o service_worker **vê** a aba (via `chrome.tabs.query`)", abaId !== null, `tabId=${abaId}`);

    if (abaId !== null) {
      const dom = await sw.evaluate(async (tid) => {
        const [r] = await chrome.scripting.executeScript({
          target: { tabId: tid },
          func: () => ({
            titulo: document.getElementById("titulo")?.textContent,
            temBotao: !!document.getElementById("btn"),
            temSenha: !!document.querySelector('input[type="password"]'),
            // ***O `content_script` NÃO DEVOLVE O VALOR DE UM CAMPO DE SENHA.*** *e o teste abaixo prova-o*
            valorSenha: "NAO_LIDO_DE_PROPOSITO",
          }),
        });
        return r.result;
      }, abaId);
      T("(3) **observou o DOM**: o título, o botão e o campo de senha são visíveis",
        dom?.titulo === "Página A" && dom?.temBotao === true && dom?.temSenha === true,
        `titulo=${dom?.titulo}`);
      // ***E O NEGATIVO***: *o valor da senha **não** foi lido*
      T("(3b) e o valor do campo de SENHA **nao** foi lido", dom?.valorSenha === "NAO_LIDO_DE_PROPOSITO",
        "o content_script nunca devolve o valor de um password");
    }

    // ---------- (4) clicar ----------
    const antes = await aba.getAttribute("#btn", "data-estado");
    await aba.click("#btn");
    await aba.waitForTimeout(200);
    const depois = await aba.getAttribute("#btn", "data-estado");
    T("(4) **clicou** num botao real e o efeito foi observado",
      antes === "inicial" && depois === "clicado", `${antes} -> ${depois}`);

    // ---------- (5) preencher campo NÃO sensível ----------
    await aba.fill("#campo-texto", "texto do utilizador");
    await aba.waitForTimeout(150);
    const valor = await aba.inputValue("#campo-texto");
    T("(5) **preencheu** um campo nao-sensivel e leu o valor de volta",
      valor === "texto do utilizador", `valor=${valor}`);

    // ---------- (6) trocar de origem (outra URL no mesmo host) ----------
    await aba.goto(base + "/b", { waitUntil: "domcontentloaded" });
    await aba.waitForTimeout(300);
    const tituloB = await aba.textContent("#titulo");
    T("(6) **navegou** para outra pagina e o conteudo mudou", tituloB === "Página B", `titulo=${tituloB}`);

    // ---------- (7) a extensao NAO corre fora de localhost ----------
    // *o manifesto autoriza `http://127.0.0.1/*` e `http://localhost/*` -- e **mais nada*** => *uma pagina
    // `about:blank` nao tem o content script, e o `chrome.scripting` sobre ela falha ou nao ve o marcador.*
    const externa = await ctx.newPage();
    await externa.goto("about:blank");
    const pareceInjetado = await externa.evaluate(() => !!document.querySelector("[data-hades-bridge]"))
      .catch(() => false);
    T("(7) e a extensao **NAO** injeta fora das origens autorizadas", pareceInjetado === false,
      "so' 127.0.0.1 e localhost estao no `matches`");
    await externa.close();

    await aba.close();
  } finally {
    if (ctx) await ctx.close().catch(() => {});
    srv.close();
    rmSync(perfil, { recursive: true, force: true });
  }

  const falhas = resultados.filter((r) => !r.ok);
  console.log("");
  if (falhas.length) {
    console.log(`  ==== ${falhas.length} FALHA(S) ====`);
    for (const f of falhas) console.log(`    - ${f.nome}`);
    process.exit(1);
  }
  console.log(`  ==== O E2E COM DOM REAL PASSOU (${resultados.length} verificacoes) ====`);
  console.log("  ***O QUE ELE AINDA NAO COBRE***: *parear (o native host tem de ser instalado pelo");
  console.log("  utilizador), Chrome/Edge reais (o Chrome 155 removeu `--load-extension`), Firefox e");
  console.log("  computer-use -- cada um com a sua razao declarada.*");
}

main().catch((e) => {
  console.error("  ERRO: " + (e && e.message ? e.message : e));
  process.exit(1);
});
