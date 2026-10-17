/**
 * E2E do **pareamento real** com o HADES Local Companion -- §ESTÁGIO 14 / §FASE-3-B (`E-1118`).
 *
 * ***O QUE FALTAVA.*** *O `test/companion.test.js` já cobre o `pairing` e o `handshake` como **unit test**
 * (`node --test`). Mas o §FASE-3-B exige, como **produto**:*
 *
 *   - *«pairing iniciado pelo utilizador com codigo curto ou QR, chave aleatoria por dispositivo,
 *     **expiração**, **rotação** e **revogação**»*;
 *   - *«instalar/configurar o Native Host/Companion de modo **reversível**»*.
 *
 * => *e o que faltava **medir ao vivo** era a **revogação** e o **estado honesto** do host.*
 *
 * ***O QUE ESTE SCRIPT PROVA, contra o PROCESSO REAL (`spawn` + pipe do SO):***
 *   1. o `host:ola` responde com o protocolo e as capacidades -- **e declara `instalado: false` e
 *      `assinado: false`**, que é a verdade (não há assinatura nem registo no SO);
 *   2. o `pairing:iniciar` devolve um **código de 6 dígitos** com **expiração**;
 *   3. o `pairing:confirmar` **só aceita com a chave de 32 bytes** que a extensão deriva -- *o host não a
 *      inventa nem a deriva sozinho*;
 *   4. o `handshake:desafio` + `handshake:aceitar` provam a **posse da chave** (HMAC-SHA256 do desafio);
 *   5. o **replay** do mesmo nonce é recusado -- *e uma prova errada também*;
 *   6. ***e a REVOGAÇÃO***: `revogar` e depois o `handshake` recusa com `dispositivo_revogado`;
 *   7. e o `aprovacao:consultar` responde **honestamente** `verificada: false` -- *uma rota que responde
 *      «não posso» é melhor do que uma rota que não existe: quem pergunta fica a saber que perguntou ao sítio
 *      certo.*
 *
 * ***COMO CORRER.*** `node apps/hades-browser-bridge/test/e2e-companion-real.mjs`
 */
import { spawn } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

globalThis.crypto ??= (await import("node:crypto")).webcrypto;

const AQUI = path.dirname(fileURLToPath(import.meta.url));
const COMPANION = path.join(AQUI, "..", "native-host", "companion.mjs");
const C = await import("../src/canal_local.js");
const T = await import("../src/transporte.js");

const resultados = [];
function T_(nome, ok, detalhe = "") {
  resultados.push({ nome, ok, detalhe });
  console.log(`${ok ? "  OK   " : "  FALHA"} ${nome}${detalhe ? "  -- " + detalhe : ""}`);
}

/** Arranca o **processo real** e fala com ele pelo pipe do SO. */
function arrancar() {
  const p = spawn(process.execPath, [COMPANION], { stdio: ["pipe", "pipe", "pipe"] });
  let pendente = new Uint8Array(0);
  const esperando = new Map();
  let stderr = "";
  let id = 0;
  p.stdout.on("data", (b) => {
    const junto = new Uint8Array(pendente.length + b.length);
    junto.set(pendente, 0);
    junto.set(new Uint8Array(b), pendente.length);
    let r;
    try {
      r = T.desenquadrar(junto);
    } catch (e) {
      for (const [, rej] of esperando) rej(e);
      esperando.clear();
      return;
    }
    pendente = r.resto;
    for (const m of r.mensagens) {
      const w = esperando.get(m.id);
      if (w) { esperando.delete(m.id); w(m); }
    }
  });
  p.stderr.on("data", (b) => { stderr += b.toString("utf8"); });
  return {
    pedir(cmd, extra = {}, timeoutMs = 8000) {
      const meu = ++id;
      const m = { id: meu, cmd, ...extra };
      return new Promise((res, rej) => {
        const t = setTimeout(() => { esperando.delete(meu); rej(new Error("timeout no " + cmd)); }, timeoutMs);
        esperando.set(meu, (r) => { clearTimeout(t); res(r); });
        p.stdin.write(Buffer.from(T.enquadrar(m)));
      });
    },
    stderr: () => stderr,
    fechar() { try { p.stdin.end(); } catch {} },
    matar() { try { p.kill(); } catch {} },
  };
}

async function main() {
  if (!COMPANION) {
    console.log("  BLOCKED: o companion nao existe");
    process.exit(2);
  }
  const c = arrancar();
  const INST = "inst-e2e-1";
  try {
    // ---------- (1) o host responde e e' HONESTO sobre o que e' ----------
    const ola = await c.pedir("host:ola");
    T_("(1) o processo real arranca e responde ao `host:ola` pelo pipe do SO", ola?.ok === true,
      `protocolo=${ola?.protocolo}`);
    T_("(1b) e declara **`instalado: false`** e **`assinado: false`** (a verdade, nao uma promessa)",
      ola?.instalado === false && ola?.assinado === false,
      `instalado=${ola?.instalado} assinado=${ola?.assinado}`);
    // ***VAZIO E' HONESTO.*** *O `CAPACIDADES = []` do companion tem o comentario «Vazio e' honesto: este host
    // nao controla o PC». => *o teste **nao** pode exigir capacidade: tem de exigir que o campo seja um ARRAY e
    // que, se estiver vazio, isso seja a verdade declarada.*
    T_("(1c) e o campo `capacidades` e' um **array** (vazio e' honesto: o host nao controla o PC)",
      Array.isArray(ola?.capacidades),
      `capacidades=[${(ola?.capacidades || []).join(",")}]`);

    // ---------- (2) pairing: codigo curto + expiracao ----------
    const ini = await c.pedir("pairing:iniciar");
    T_("(2) `pairing:iniciar` devolve um codigo de **6 digitos**", /^\d{6}$/.test(String(ini?.codigo ?? "")),
      `codigo=******`);
    T_("(2b) e uma **expiracao**", typeof ini?.expiraEm === "number" && ini.expiraEm > Date.now(),
      `expiraEm em ${Math.round((ini.expiraEm - Date.now()) / 1000)}s`);

    // ---------- (3) a chave vem da EXTENSAO, nao do host ----------
    const semChave = await c.pedir("pairing:confirmar", { codigo: ini.codigo, instalacao: INST });
    T_("(3) `pairing:confirmar` **sem chave** e' recusado", semChave?.ok === false,
      semChave?.motivo);

    const chave = await C.derivarChave(ini.codigo, ini.pedido, INST, "m1");
    const bytes = Array.from(await C.exportarChave(chave));
    T_("(3b) a extensao deriva uma chave de **32 bytes**", bytes.length === 32, `${bytes.length} bytes`);

    const conf = await c.pedir("pairing:confirmar", { codigo: ini.codigo, chave: bytes, instalacao: INST });
    T_("(3c) e com a chave, o **pairing** completa", conf?.ok === true, JSON.stringify(conf).slice(0, 60));

    // ---------- (4) handshake: prova de posse ----------
    const d = await c.pedir("handshake:desafio", { instalacao: INST });
    T_("(4) `handshake:desafio` devolve um **nonce** de 64 hex", /^[0-9a-f]{64}$/.test(String(d?.nonce ?? "")),
      `nonce=${String(d?.nonce ?? "").slice(0, 12)}...`);

    const prova = await C.assinarDesafio(chave, d.nonce, INST);
    const h = await c.pedir("handshake:aceitar", { instalacao: INST, nonce: d.nonce, prova });
    T_("(4b) e o **handshake** aceita a prova de posse (HMAC-SHA256)", h?.ok === true,
      JSON.stringify(h).slice(0, 60));

    // ---------- (5) replay e prova errada ----------
    const h2 = await c.pedir("handshake:aceitar", { instalacao: INST, nonce: d.nonce, prova });
    T_("(5) o **replay** do mesmo nonce e' recusado", h2?.ok === false, h2?.motivo);

    const d3 = await c.pedir("handshake:desafio", { instalacao: INST });
    const ruim = await c.pedir("handshake:aceitar", { instalacao: INST, nonce: d3.nonce, prova: "00" });
    T_("(5b) e uma **prova errada** tambem", ruim?.ok === false, ruim?.motivo);

    // ---------- (6) a rota de aprovacao responde com a VERDADE ----------
    const ap = await c.pedir("aprovacao:consultar", { fingerprint: "a".repeat(64), instalacao: INST });
    T_("(6) `aprovacao:consultar` responde -- e responde **`verificada: false`**",
      ap?.ok === true && ap?.verificada === false,
      `verificada=${ap?.verificada} motivo=${String(ap?.motivo || "").slice(0, 40)}`);

    // ---------- (7) A REVOGACAO ----------
    const rev = await c.pedir("revogar", { instalacao: INST });
    T_("(7) `revogar` completa", rev?.ok === true, JSON.stringify(rev).slice(0, 50));

    // ***O TRAVA REAL E' O `aceitar`, NAO O `desafio`.*** *O `abrirDesafio` so' gera um nonce -- nao da'
    // acesso a nada. Quem recusa um dispositivo revogado e' o `aceitarHandshake` (`dispositivo_revogado`) e o
    // `aprovacao:consultar`.* => *o teste tem de exercitar o **trava**, nao a porta que so' emite um numero.*
    const d4 = await c.pedir("handshake:desafio", { instalacao: INST });
    const prova4 = d4?.ok ? await C.assinarDesafio(chave, d4.nonce, INST) : "00";
    const h4 = await c.pedir("handshake:aceitar", { instalacao: INST, nonce: d4?.nonce, prova: prova4 });
    T_("(7b) e depois de revogar o **handshake e' RECUSADO** (`dispositivo_revogado`)",
      h4?.ok === false && h4?.motivo === "dispositivo_revogado", h4?.motivo);

    const ap2 = await c.pedir("aprovacao:consultar", { fingerprint: "a".repeat(64), instalacao: INST });
    T_("(7c) e um dispositivo revogado **nao tem direito a perguntar** (`dispositivo_revogado`)",
      ap2?.ok === false && ap2?.motivo === "dispositivo_revogado", ap2?.motivo);

    // ---------- (8) e o host continua a responder (nao morreu) ----------
    const ola2 = await c.pedir("host:ola");
    T_("(8) e o host **continua vivo** depois da revogacao", ola2?.ok === true);
  } finally {
    c.fechar();
    c.matar();
  }

  const falhas = resultados.filter((r) => !r.ok);
  console.log("");
  if (falhas.length) {
    console.log(`  ==== ${falhas.length} FALHA(S) ====`);
    for (const f of falhas) console.log(`    - ${f.nome}${f.detalhe ? "  (" + f.detalhe + ")" : ""}`);
    process.exit(1);
  }
  console.log(`  ==== O E2E DO PAREAMENTO REAL PASSOU (${resultados.length} verificacoes) ====`);
  console.log("  ***O QUE FALTA***: *o `nativeMessaging` do Chrome (que exige o gesto do utilizador em");
  console.log("  `chrome.permissions.request`) e a instalacao do native host no `%LOCALAPPDATA%`.*");
}

main().catch((e) => {
  console.error("  ERRO: " + (e && e.message ? e.message : e));
  process.exit(1);
});
