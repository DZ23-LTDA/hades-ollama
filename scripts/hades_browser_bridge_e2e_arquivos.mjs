/**
 * E2E DO SCOPE `browser:files` (`E-1040`).
 *
 * O scope estava nos 3 canonicos e **nao tinha uma linha de implementacao**. O unico vestigio
 * era o `preencher` a recusar `type=file` com a mensagem
 * `upload_exige_aprovacao_e_caminho_autorizado` -- **a mensagem prometia um caminho
 * autorizado e um manifesto com hash que nao existiam.**
 *
 * Este E2E corre a extensao REAL no Edge e fala com o service worker REAL. O que discrimina:
 *   1. `arquivo:medir` mede o hash dos BYTES (e dois conteudos diferentes dao hashes diferentes);
 *   2. download de `.csv` inerte -> **permitido**;
 *   3. download de `.exe` -> **exige_aprovacao**, e **continua** a exigir com `aprovado: true`
 *      (*um executavel nunca fica "permitido"*);
 *   4. `../a.txt` -> **proibido** (*nao normaliza: recusa*);
 *   5. sem `sha256` -> **proibido** (*nao se aprova o que nao se mediu*);
 *   6. upload de `.txt` inerte -> **exige_aprovacao** (*sair do disco e' sempre um acto*);
 *   7. o registo fica no estado -- E O ESTADO SOBREVIVE AS LEITURAS (`E-1040` curou a lista
 *      fixa do `lerEstado`, que apagava campos desconhecidos).
 */
import { fileURLToPath, pathToFileURL } from "node:url";
import { execFileSync } from "node:child_process";
import path from "node:path";
import fs from "node:fs";
import os from "node:os";

const AQUI = path.dirname(fileURLToPath(import.meta.url));
const RAIZ = path.resolve(AQUI, "..");
const { chromium } = await import(
  pathToFileURL(path.join(RAIZ, "app", "ui", "app", "node_modules", "playwright", "index.mjs")).href
);
const EXT = path.join(RAIZ, "apps", "hades-browser-bridge");
// **E-1083: o browser vem do ambiente, e nao do codigo.** *O prompt exige Chrome E Edge; com o
// caminho do Edge fixo, so' um deles era testavel -- e a constante `EDGE` afirmava o browser.*
const BROWSERS = {
  edge: "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe",
  chrome: "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
};
const QUAL = (process.env.HADES_E2E_BROWSER || "edge").trim().toLowerCase();
if (!(QUAL in BROWSERS)) {
  console.error(`  HADES_E2E_BROWSER="${QUAL}" nao suportado. Aceita: ${Object.keys(BROWSERS).join(", ")}.`);
  if (QUAL === "firefox" || QUAL === "webkit") {
    console.error("  Firefox/WebKit NAO sao suportados: a extensao e' Manifest V3, e o caminho de carga");
    console.error("  e' o de Chromium (--load-extension). Testa-la noutro motor exige outro manifesto.");
  }
  process.exit(2);
}
const EDGE = BROWSERS[QUAL];
const CHROMIUM = true; // *as duas entradas acima sao Chromium*

const falhas = [];
const ok = (n, c, d = "") => {
  console.log((c ? "  OK   " : "  FAIL ") + n + (d && !c ? "  -- " + d : ""));
  if (!c) falhas.push(n);
};

const instalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1")], { encoding: "utf8" });
const desinstalar = () => execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass",
  "-File", path.join(RAIZ, "scripts", "hades_native_host_install.ps1"), "-Desinstalar"], { encoding: "utf8" });

const saidaInst = instalar();
const perfil = fs.mkdtempSync(path.join(os.tmpdir(), "hades-arq-"));
let ctx = null;
try {
  ctx = await chromium.launchPersistentContext(perfil, {
    executablePath: EDGE, headless: false,
    args: [`--disable-extensions-except=${EXT}`, `--load-extension=${EXT}`,
      // *O Chrome 155 removeu `--load-extension`; o Edge ainda o aceita com este workaround.*
        // => *declarado, e nao silencioso: o leitor sabe que ha um switch a compensar.*
        "--disable-features=DisableLoadExtensionCommandLineSwitch", "--no-first-run", "--no-default-browser-check"],
  });
  let sw = ctx.serviceWorkers()[0];
  if (!sw) sw = await ctx.waitForEvent("serviceworker", { timeout: 30000 }).catch(() => null);
  ok("service worker da extensao", !!sw);
  if (!sw) throw new Error("sem service worker");
  const id = new URL(sw.url()).host;

  const popup = await ctx.newPage();
  await popup.setViewportSize({ width: 460, height: 1400 });
  await popup.goto(`chrome-extension://${id}/popup/popup.html`);
  await popup.waitForSelector("#bt-conectar");
  await popup.waitForFunction(() => (document.querySelector("#aviso")?.textContent ?? "").length > 0);
  await popup.waitForTimeout(600);
  const pedir = (m) => popup.evaluate(async (mm) => chrome.runtime.sendMessage(mm), m);

  const utf8 = (s) => Array.from(new TextEncoder().encode(s));

  console.log("\n1) `arquivo:medir` mede o HASH dos bytes");
  const b1 = utf8("conteudo-um");
  const m1 = await pedir({ tipo: "arquivo:medir", nome: "a.csv", bytes: b1 });
  console.log("   " + JSON.stringify(m1));
  ok("mede com sucesso", m1?.ok === true, JSON.stringify(m1));
  ok("o hash tem 64 hex", /^[a-f0-9]{64}$/.test(m1?.sha256 ?? ""), String(m1?.sha256));
  ok("o tamanho e' o dos bytes", m1?.tamanho === b1.length, String(m1?.tamanho));

  const m2 = await pedir({ tipo: "arquivo:medir", nome: "a.csv", bytes: utf8("conteudo-dois") });
  ok("conteudo diferente -> hash diferente", m1?.sha256 !== m2?.sha256);
  const m3 = await pedir({ tipo: "arquivo:medir", nome: "a.csv", bytes: b1 });
  ok("conteudo igual -> hash igual", m1?.sha256 === m3?.sha256);
  const H = m1.sha256;

  console.log("\n2) download inerte -> PERMITIDO");
  const r1 = await pedir({ tipo: "arquivo:avaliar", pedido: { nome: "relatorio.csv", direcao: "download", tamanho: 12, sha256: H } });
  console.log("   " + JSON.stringify(r1?.decisao));
  ok("ok=true", r1?.ok === true, JSON.stringify(r1));
  ok("veredicto `permitido`", r1?.decisao?.veredicto === "permitido", JSON.stringify(r1?.decisao));
  ok("risco `inerte`", r1?.decisao?.risco === "inerte");
  ok("e `podesTransferir` e' TRUE", r1?.podesTransferir === true, String(r1?.podesTransferir));

  console.log("\n3) download EXECUTAVEL -> exige aprovacao, E CONTINUA A EXIGIR");
  const r2 = await pedir({ tipo: "arquivo:avaliar", pedido: { nome: "setup.exe", direcao: "download", tamanho: 12, sha256: H } });
  console.log("   sem aprovacao: " + JSON.stringify(r2?.decisao));
  ok("exige aprovacao", r2?.decisao?.veredicto === "exige_aprovacao", JSON.stringify(r2?.decisao));
  ok("motivo `executavel_exige_aprovacao`", r2?.decisao?.motivo === "executavel_exige_aprovacao");
  // *`ok` significa "a avaliacao correu"; `podesTransferir` significa "podes transferir".*
  ok("a avaliacao correu", r2?.ok === true, String(r2?.ok));
  ok("mas NAO podes transferir sem aprovacao", r2?.podesTransferir === false, String(r2?.podesTransferir));

  // `E-1041`: um booleano solto JA' NAO AUTORIZA. *Um campo que quem pede controla nao e'
  // uma aprovacao -- e' uma afirmacao.*
  const r3s = await pedir({ tipo: "arquivo:avaliar", aprovado: true, pedido: { nome: "setup.exe", direcao: "download", tamanho: 12, sha256: H } });
  console.log("   booleano solto: " + JSON.stringify(r3s?.decisao));
  ok("booleano solto NAO autoriza", r3s?.decisao?.veredicto === "exige_aprovacao", JSON.stringify(r3s?.decisao));
  ok("e diz porque", r3s?.decisao?.aprovacao === "booleano_solto_nao_e_aprovacao", JSON.stringify(r3s?.decisao));

  // a PROVA estruturada: o fingerprint tem de casar com o pedido
  // *`import()` e' proibido no ServiceWorkerGlobalScope por especificacao.* O fingerprint
  // calcula-se AQUI, com o mesmo modulo, e vai medido para dentro.
  const { fingerprintDe } = await import(pathToFileURL(path.join(EXT, "src", "arquivos.js")).href);
  const fp = (await fingerprintDe({ direcao: "download", nome: "setup.exe", tamanho: 12, sha256: H })).fingerprint;
  const r3 = await pedir({
    tipo: "arquivo:avaliar",
    prova: { aprovacaoId: "AP-1", fingerprint: fp, aprovadoPor: "humano-do-teste" },
    pedido: { nome: "setup.exe", direcao: "download", tamanho: 12, sha256: H, fingerprint: fp },
  });
  console.log("   com prova: " + JSON.stringify(r3?.decisao));
  ok("COM a prova certa, autorizado", r3?.decisao?.veredicto === "autorizado", JSON.stringify(r3?.decisao));
  ok("e o id da aprovacao fica registado", r3?.decisao?.aprovacaoId === "AP-1");
  ok("e o autor tambem", r3?.decisao?.aprovadoPor === "humano-do-teste");

  // prova de OUTRO pedido
  const fpOutro = (await fingerprintDe({ direcao: "download", nome: "outro.csv", tamanho: 1, sha256: H })).fingerprint;
  const r3x = await pedir({
    tipo: "arquivo:avaliar",
    prova: { aprovacaoId: "AP-2", fingerprint: fpOutro, aprovadoPor: "humano" },
    pedido: { nome: "setup.exe", direcao: "download", tamanho: 12, sha256: H, fingerprint: fp },
  });
  console.log("   prova cruzada: " + JSON.stringify(r3x?.decisao));
  ok("prova de OUTRO pedido NAO serve", r3x?.decisao?.veredicto === "exige_aprovacao", JSON.stringify(r3x?.decisao));
  ok("e diz porque", r3x?.decisao?.aprovacao === "fingerprint_nao_corresponde");
  ok("COM a prova, podesTransferir e' TRUE", r3?.podesTransferir === true, String(r3?.podesTransferir));
  ok("e com o booleano solto era FALSE -- a prova muda a resposta",
    r3s?.podesTransferir === false && r3?.podesTransferir === true,
    JSON.stringify([r3s?.podesTransferir, r3?.podesTransferir]));

  console.log("\n4) nome com CAMINHO -> proibido");
  const r4 = await pedir({ tipo: "arquivo:avaliar", pedido: { nome: "../a.txt", direcao: "download", tamanho: 5, sha256: H } });
  console.log("   " + JSON.stringify(r4?.decisao));
  ok("proibido", r4?.decisao?.veredicto === "proibido", JSON.stringify(r4?.decisao));
  ok("motivo `nome_tem_caminho`", r4?.decisao?.motivo === "nome_tem_caminho");
  const r4b = await pedir({ tipo: "arquivo:avaliar", pedido: { nome: "C:\\\\x\\\\a.exe", direcao: "download", tamanho: 5, sha256: H } });
  ok("caminho absoluto tambem proibido", r4b?.decisao?.veredicto === "proibido", JSON.stringify(r4b?.decisao));

  console.log("\n5) SEM sha256 -> proibido -- *nao se aprova o que nao se mediu*");
  const r5 = await pedir({ tipo: "arquivo:avaliar", pedido: { nome: "a.csv", direcao: "download", tamanho: 12 } });
  console.log("   " + JSON.stringify(r5?.decisao));
  ok("proibido", r5?.decisao?.veredicto === "proibido", JSON.stringify(r5?.decisao));
  ok("motivo `sha256_ausente_ou_invalido`", r5?.decisao?.motivo === "sha256_ausente_ou_invalido");
  const r5b = await pedir({ tipo: "arquivo:avaliar", pedido: { nome: "a.csv", direcao: "download", tamanho: 12, sha256: "abc" } });
  ok("sha256 curto tambem proibido", r5b?.decisao?.veredicto === "proibido", JSON.stringify(r5b?.decisao));

  console.log("\n6) UPLOAD de ficheiro INERTE -> exige aprovacao -- *sair do disco e' sempre um acto*");
  const r6 = await pedir({ tipo: "arquivo:avaliar", pedido: { nome: "a.txt", direcao: "upload", tamanho: 12, sha256: H } });
  console.log("   " + JSON.stringify(r6?.decisao));
  ok("exige aprovacao", r6?.decisao?.veredicto === "exige_aprovacao", JSON.stringify(r6?.decisao));
  ok("motivo `upload_exige_aprovacao`", r6?.decisao?.motivo === "upload_exige_aprovacao");

  const r1b = await pedir({ tipo: "arquivo:avaliar", pedido: { nome: "a.txt", direcao: "download", tamanho: 12, sha256: H } });
  ok("o MESMO ficheiro em download e' permitido", r1b?.decisao?.veredicto === "permitido", JSON.stringify(r1b?.decisao));
  ok("...logo a direccao muda a decisao",
    r1b?.decisao?.veredicto !== r6?.decisao?.veredicto, "iguais!");

  console.log("\n7) o REGISTO fica no estado -- e o estado SOBREVIVE as leituras");
  const est = await sw.evaluate(async () => {
    const bruto = await chrome.storage.session.get("hades.bridge.v1");
    return bruto?.["hades.bridge.v1"] ?? null;
  });
  const quantos = (est?.arquivos ?? []).length;
  console.log("   registos: " + quantos + "  ultimo: " + JSON.stringify(est?.arquivos?.[est.arquivos.length - 1]));
  ok("ha registos de decisao", quantos >= 10, String(quantos));
  // *a autoria da aprovacao fica no registo -- quem autorizou.*
  const comAutor = (est?.arquivos ?? []).filter((a) => a.aprovadoPor);
  console.log("   registos com autor: " + JSON.stringify(comAutor));
  ok("o registo guarda QUEM autorizou", comAutor.length >= 1 && comAutor[0].aprovadoPor === "humano-do-teste", JSON.stringify(comAutor));
  ok("e guarda o id da aprovacao", comAutor[0]?.aprovacaoId === "AP-1", JSON.stringify(comAutor[0]));
  ok("o registo NAO leva o hash", !JSON.stringify(est?.arquivos ?? []).includes(H), "o hash esta no registo!");
  ok("o registo leva o resumo", typeof est?.arquivos?.[0]?.resumo === "string", JSON.stringify(est?.arquivos?.[0]));

  // *A CURA DO `lerEstado`: um campo DESCONHECIDO tem de sobreviver a uma leitura.*
  console.log("\n8) a CURA do `lerEstado`: um campo desconhecido nao e' apagado");
  await sw.evaluate(async () => {
    const bruto = await chrome.storage.session.get("hades.bridge.v1");
    const e = bruto?.["hades.bridge.v1"] ?? {};
    e.campoDesconhecidoDoFuturo = { marca: "nao-me-apagues" };
    await chrome.storage.session.set({ "hades.bridge.v1": e });
  });
  // Qualquer mensagem que leia o estado... e o `takeover:estado` le'. Depois gravamos.
  await pedir({ tipo: "arquivo:avaliar", pedido: { nome: "a.csv", direcao: "download", tamanho: 12, sha256: H } });
  const est2 = await sw.evaluate(async () => {
    const bruto = await chrome.storage.session.get("hades.bridge.v1");
    return bruto?.["hades.bridge.v1"] ?? null;
  });
  console.log("   campoDesconhecidoDoFuturo=" + JSON.stringify(est2?.campoDesconhecidoDoFuturo));
  ok("o campo DESCONHECIDO sobreviveu", est2?.campoDesconhecidoDoFuturo?.marca === "nao-me-apagues",
    "foi apagado pela lista fixa do lerEstado!");
  ok("e os registos continuam", (est2?.arquivos ?? []).length > quantos, String((est2?.arquivos ?? []).length));

  console.log("\n9) `E-1042`: a decisao DIZ o que vale -- a extensao nao pode verificar");
  const comProva = await pedir({
    tipo: "arquivo:avaliar",
    prova: { aprovacaoId: "AP-1", fingerprint: fp, aprovadoPor: "humano-do-teste" },
    pedido: { nome: "setup.exe", direcao: "download", tamanho: 12, sha256: H, fingerprint: fp },
  });
  console.log("   autoridade=" + JSON.stringify(comProva?.autoridade));
  console.log("   aviso=" + JSON.stringify(comProva?.aviso));
  ok("a resposta traz `autoridade`", typeof comProva?.autoridade === "string", JSON.stringify(comProva?.autoridade));
  ok("e a extensao diz `forma_apenas`, nunca `verificada`",
    comProva?.autoridade === "forma_apenas", String(comProva?.autoridade));
  ok("e traz o AVISO por palavras", typeof comProva?.aviso === "string" && comProva.aviso.includes("NAO verificada"),
    String(comProva?.aviso));
  ok("o aviso diz a consequencia", String(comProva?.aviso).includes("passaria aqui"), String(comProva?.aviso));
  // e o registo tambem guarda a autoridade
  const est3 = await sw.evaluate(async () => {
    const b = await chrome.storage.session.get("hades.bridge.v1");
    return b?.["hades.bridge.v1"] ?? null;
  });
  const comAut = (est3?.arquivos ?? []).filter((a) => a.autoridade === "forma_apenas");
  ok("o registo guarda a autoridade", comAut.length >= 1, JSON.stringify(comAut.slice(-1)));

  // e uma decisao PERMITIDA tambem declara a autoridade -- *nao so' o caso autorizado*
  const permissiva = await pedir({ tipo: "arquivo:avaliar", pedido: { nome: "a.csv", direcao: "download", tamanho: 12, sha256: H } });
  ok("um `permitido` tambem declara autoridade", permissiva?.autoridade === "forma_apenas",
    JSON.stringify(permissiva?.autoridade));

  console.log("\n10) `E-1043`: a extensao PERGUNTA ao Companion -- e o Companion responde que nao pode");
  // Primeiro conceder `nativeMessaging` e recarregar (o `E-1032` mostrou que sem reload o
  // `chrome.runtime` nao ganha a permissao).
  const perm = await sw.evaluate(async () => {
    try { return await chrome.permissions.contains({ permissions: ["nativeMessaging"] }); }
    catch (e) { return "erro:" + e.message; }
  });
  console.log("   nativeMessaging concedida? " + JSON.stringify(perm));
  // *`chrome.runtime.reload()` FECHA o popup* (medido: `Target page ... has been closed`).
  // E nao e' preciso: o que este passo prova e' que a CONSULTA responde e que ela **nunca diz
  // `verificada`** -- e isso mede-se sem a permissao. O caminho COM `nativeMessaging` ja' esta'
  // provado ponta a ponta no `E-1032` (extensao -> native messaging -> Companion).
  const cons = await pedir({ tipo: "arquivo:consultarAutoridade", fingerprint: fp });
  console.log("   resposta: " + JSON.stringify(cons));
  ok("a consulta responde", cons?.ok === true, JSON.stringify(cons));
  ok("e responde que NAO verificou", cons?.verificada === false, JSON.stringify(cons));
  ok("a autoridade fica `forma_apenas`", cons?.autoridade === "forma_apenas", String(cons?.autoridade));
  ok("com um motivo", typeof cons?.motivo === "string" && cons.motivo.length > 0, String(cons?.motivo));
  // O motivo e'' o do Companion quando o canal funciona; `canal_indisponivel` e'' o honesto
  // quando a permissao nao esta. Ambos sao verdade -- e nenhum diz `verificada`.
  ok("o motivo e' honesto",
    ["sem_rota_autenticavel_no_runtime", "canal_indisponivel", "sem_dispositivo_pareado",
     "instalacao_desconhecida", "erro_de_canal", "resposta_invalida"].includes(cons?.motivo),
    String(cons?.motivo));
  ok("NUNCA diz verificada", cons?.verificada !== true, JSON.stringify(cons));

  await popup.close();
} catch (e) {
  console.log("   ERRO: " + e.message);
  falhas.push("execucao: " + e.message);
} finally {
  if (ctx) await ctx.close().catch(() => {});
  fs.rmSync(perfil, { recursive: true, force: true });
  if (/registado/.test(saidaInst)) desinstalar();
}

console.log("");
if (falhas.length) {
  console.log("FALHAS: " + falhas.length + " -> " + falhas.join(" | "));
  process.exit(1);
}
console.log("E2E DE ARQUIVOS: o scope browser:files decide -- e um executavel nunca fica 'permitido'.");
process.exit(0);
