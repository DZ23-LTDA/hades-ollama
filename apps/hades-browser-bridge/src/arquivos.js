// HADES Browser Bridge -- politica de transferencia de ficheiros (`E-1040`).
//
// O scope `browser:files` estava nos 3 canonicos e **nao tinha uma linha de implementacao** --
// tal como o `browser:takeover` antes do `E-1034`. O unico vestigio era o `preencher` a
// recusar `type=file` com a mensagem `upload_exige_aprovacao_e_caminho_autorizado`:
// **a mensagem prometia um caminho autorizado e um manifesto com hash que nao existiam.**
//
// O contrato exige:
//   * "uploads/downloads usam caminhos autorizados e manifestos com tamanho/hash";
//   * "download executavel ou arquivo sensivel exige aprovacao";
//   * "clipboard, notificacoes e downloads ficam fora do escopo ate permissao especifica".
//
// Este modulo NAO descarrega nem carrega nada. Decide. *Uma extensao que descarrega por conta
// propria seria mais um actor com escrita no disco do utilizador; o que falta ao HADES nao e'
// um motor de download -- e' uma decisao auditavel sobre ficheiros.*
//
// Sem `eval`, sem codigo remoto, sem I/O.

/** O protocolo deste modulo. Muda quando a forma da decisao mudar. */
export const PROTOCOLO_ARQUIVOS = 1;

/** Tetos. *Um limite escrito e' um limite aplicado.* */
export const LIMITES_ARQUIVOS = Object.freeze({
  /** Nome de ficheiro: o que o Chrome aceita com seguranca. */
  nomeBytes: 255,
  /** Um ficheiro que o HADES aceita avaliar (o hash tem de caber na memoria). */
  ficheiroBytes: 256 * 1024 * 1024,
  /** Manifesto serializado. */
  manifestoBytes: 8 * 1024,
});

/**
 * Extensoes que EXECUTAM codigo ou alteram o sistema quando abertas.
 * *Nao e' uma lista de "coisas más": e' a lista das que precisam de uma decisao humana
 * explicita antes de tocar no disco.*
 */
export const EXECUTAVEIS = Object.freeze([
  "exe", "msi", "msp", "bat", "cmd", "com", "scr", "pif", "cpl", "jar",
  "ps1", "psm1", "vbs", "vbe", "js", "jse", "wsf", "wsh", "hta", "reg",
  "dll", "sys", "drv", "apk", "app", "dmg", "pkg", "deb", "rpm", "appimage",
  "sh", "run", "bin", "so", "dylib", "lnk", "url", "iso", "img", "vhd",
]);

/** Extensoes que costumam conter credenciais, chaves ou dados pessoais. */
export const SENSIVEIS = Object.freeze([
  "pem", "key", "pfx", "p12", "jks", "keystore", "kdbx", "ovpn",
  "env", "sqlite", "db", "sql", "bak", "kdb", "1pif", "asc", "gpg",
]);

/** Extensoes que sao conteudo inerte e podem ser lidas sem decisao adicional. */
export const INERTES = Object.freeze([
  "txt", "md", "csv", "tsv", "json", "xml", "yaml", "yml", "log", "ini", "cfg",
  "html", "htm", "css", "png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "ico",
  "pdf", "mp3", "wav", "ogg", "mp4", "webm", "woff", "woff2", "ttf", "otf",
]);

/** Os dois sentidos. Nao ha' um terceiro. */
export const DIRECOES = Object.freeze(["download", "upload"]);

/** Os tres veredictos. */
export const VEREDICTOS = Object.freeze(["permitido", "exige_aprovacao", "proibido"]);

/** A extensao, em minusculas, sem caminho e sem pontos a mais. */
export function extensaoDe(nome) {
  const base = String(nome ?? "").split(/[\\/]/).pop() ?? "";
  const i = base.lastIndexOf(".");
  if (i <= 0 || i === base.length - 1) return "";
  return base.slice(i + 1).toLowerCase().replace(/[^a-z0-9]/g, "");
}

/**
 * O nome tem de ser um NOME, nao um caminho.
 *
 * *Um caminho absoluto num manifesto de extensao e' um pedido para escrever fora do sitio
 * combinado* -- e o `E-1009` ja' mostrou que **um caminho nao partilha a base de quem o
 * declara**. Por isso aqui o nome e' recusado, nao normalizado.
 */
export function nomeSeguro(nome) {
  const s = String(nome ?? "");
  if (s.trim() === "") return { ok: false, motivo: "nome_vazio" };
  if (s.length > LIMITES_ARQUIVOS.nomeBytes) return { ok: false, motivo: "nome_longo" };
  if (/[\\/]/.test(s)) return { ok: false, motivo: "nome_tem_caminho" };
  if (/^[a-zA-Z]:/.test(s)) return { ok: false, motivo: "nome_tem_unidade" };
  if (s === "." || s === ".." || s.startsWith("..")) return { ok: false, motivo: "nome_travessia" };
  if (/[\x00-\x1f<>:"|?*]/.test(s)) return { ok: false, motivo: "nome_caracteres_invalidos" };
  if (/^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\.|$)/i.test(s)) return { ok: false, motivo: "nome_reservado_windows" };
  return { ok: true, nome: s };
}

/**
 * Normaliza bytes venham de onde vierem (`E-1040`).
 *
 * **Medido:** o `chrome.runtime.sendMessage` **serializa em JSON** -- e um `Uint8Array`
 * atravessa como um objecto de chaves numericas, **sem `byteLength`**. A primeira versao do
 * E2E recebeu `{ok:false, motivo:"bytes_ausentes"}` e o defeito era do canal, nao do pedido.
 *
 * Isso tem uma consequencia de seguranca que vale a pena dizer em voz alta: ***o canal NAO
 * transporta binario.*** O que atravessa sao metadados e uma representacao -- e por isso a
 * MEDICAO tem de acontecer onde os bytes existem de verdade.
 *
 * Aceita: `ArrayBuffer`, `TypedArray` e a representacao JSON (valores inteiros 0..255).
 * Recusa: qualquer coisa que nao seja uma sequencia de octetos *valida*.
 */
export function normalizarBytes(bruto) {
  if (bruto === null || bruto === undefined) return { ok: false, motivo: "bytes_ausentes" };
  if (bruto instanceof ArrayBuffer) return { ok: true, bytes: new Uint8Array(bruto) };
  if (ArrayBuffer.isView(bruto)) {
    return { ok: true, bytes: new Uint8Array(bruto.buffer, bruto.byteOffset, bruto.byteLength) };
  }
  if (Array.isArray(bruto)) {
    if (bruto.some((v) => !Number.isInteger(v) || v < 0 || v > 255)) {
      return { ok: false, motivo: "bytes_invalidos" };
    }
    return { ok: true, bytes: Uint8Array.from(bruto) };
  }
  if (typeof bruto === "object") {
    // a forma que o JSON do canal produz: chaves "0","1","2",... e nada mais
    const chaves = Object.keys(bruto);
    if (chaves.length === 0) return { ok: false, motivo: "bytes_ausentes" };
    if (chaves.some((k) => !/^\d+$/.test(k))) return { ok: false, motivo: "bytes_invalidos" };
    const valores = chaves.map((k) => bruto[k]);
    if (valores.some((v) => !Number.isInteger(v) || v < 0 || v > 255)) {
      return { ok: false, motivo: "bytes_invalidos" };
    }
    return { ok: true, bytes: Uint8Array.from(valores) };
  }
  return { ok: false, motivo: "bytes_ausentes" };
}

/** SHA-256 sobre os BYTES, em hex. *O hash e' do conteudo: um nome nao prova nada.* */
export async function sha256De(bruto) {
  const n = normalizarBytes(bruto);
  if (!n.ok) return n;
  const bytes = n.bytes;
  if (bytes.byteLength > LIMITES_ARQUIVOS.ficheiroBytes) {
    return { ok: false, motivo: "ficheiro_demasiado_grande", bytes: bytes.byteLength };
  }
  const digest = await crypto.subtle.digest("SHA-256", bytes);
  const hex = Array.from(new Uint8Array(digest)).map((b) => b.toString(16).padStart(2, "0")).join("");
  return { ok: true, sha256: hex, bytes: bytes.byteLength };
}

/**
 * Classifica um ficheiro pelo nome e pelo tamanho.
 * *A classificacao nao decide: informa a decisao.*
 */
export function classificar(nome, bytesOuTamanho) {
  const n = nomeSeguro(nome);
  if (!n.ok) return n;
  const ext = extensaoDe(n.nome);
  let tamanho = null;
  if (typeof bytesOuTamanho === "number") tamanho = bytesOuTamanho;
  else if (bytesOuTamanho instanceof ArrayBuffer) tamanho = bytesOuTamanho.byteLength;
  else if (ArrayBuffer.isView(bytesOuTamanho)) tamanho = bytesOuTamanho.byteLength;
  else if (Array.isArray(bytesOuTamanho)) tamanho = bytesOuTamanho.length;
  else if (bytesOuTamanho && typeof bytesOuTamanho === "object") {
    const nb = normalizarBytes(bytesOuTamanho);
    tamanho = nb.ok ? nb.bytes.byteLength : null;
  }
  if (tamanho === null || tamanho < 0) return { ok: false, motivo: "tamanho_ausente" };
  if (tamanho > LIMITES_ARQUIVOS.ficheiroBytes) {
    return { ok: false, motivo: "ficheiro_demasiado_grande", tamanho };
  }
  const risco = EXECUTAVEIS.includes(ext) ? "executavel"
    : SENSIVEIS.includes(ext) ? "sensivel"
    : INERTES.includes(ext) ? "inerte"
    : "desconhecido";
  return { ok: true, nome: n.nome, extensao: ext, tamanho, risco };
}

/**
 * A DECISAO. Recebe o manifesto e se ja' houve aprovacao para ESTA transferencia.
 *
 * Regras, na ordem em que sao aplicadas -- *a ordem e' a politica*:
 *   1. nome invalido               -> proibido;
 *   2. direccao invalida           -> proibido;
 *   3. sem sha256 quando exigido   -> proibido (nao se aprova o que nao se mediu);
 *   4. risco executavel            -> exige_aprovacao (mesmo com aprovacao previa: o
 *      executavel nunca e' "permitido" -- no maximo "permitido_com_aprovacao");
 *   5. risco sensivel              -> exige_aprovacao;
 *   6. upload de qualquer ficheiro -> exige_aprovacao (sair do disco e' sempre um acto);
 *   7. risco desconhecido          -> exige_aprovacao (*nao ha' lista completa de perigo*);
 *   8. inerte + download           -> permitido.
 */
export function decidir(manifesto, opcoes = {}) {
  const aprovado = opcoes.aprovado === true;
  const m = manifesto ?? {};

  const direcao = String(m.direcao ?? "");
  if (!DIRECOES.includes(direcao)) return { veredicto: "proibido", motivo: "direccao_invalida" };

  const c = classificar(m.nome, m.tamanho);
  if (!c.ok) return { veredicto: "proibido", motivo: c.motivo };

  if (typeof m.sha256 !== "string" || !/^[a-f0-9]{64}$/.test(m.sha256)) {
    return { veredicto: "proibido", motivo: "sha256_ausente_ou_invalido" };
  }

  const base = {
    nome: c.nome, extensao: c.extensao, tamanho: c.tamanho, risco: c.risco,
    direcao, sha256: m.sha256,
  };

  if (c.risco === "executavel") {
    // *Um executavel nunca fica "permitido" sozinho.* Com aprovacao, o veredicto continua a
    // dizer de onde veio a permissao.
    return aprovado
      ? { veredicto: "exige_aprovacao", motivo: "executavel_aprovado_nesta_transferencia", aprovado: true, ...base }
      : { veredicto: "exige_aprovacao", motivo: "executavel_exige_aprovacao", aprovado: false, ...base };
  }
  if (c.risco === "sensivel") {
    return { veredicto: "exige_aprovacao", motivo: "ficheiro_sensivel_exige_aprovacao", aprovado, ...base };
  }
  if (direcao === "upload") {
    return { veredicto: "exige_aprovacao", motivo: "upload_exige_aprovacao", aprovado, ...base };
  }
  if (c.risco === "desconhecido") {
    return { veredicto: "exige_aprovacao", motivo: "extensao_desconhecida_exige_aprovacao", aprovado, ...base };
  }
  return { veredicto: "permitido", motivo: "conteudo_inerte_em_download", aprovado, ...base };
}

/**
 * Monta o manifesto canonico. O `sha256` tem de vir de `sha256De(bytes)` -- nunca de um
 * campo declarado por quem pede. *Quem pede nao certifica o que pede.*
 */
export function manifesto({ nome, direcao, bytes, sha256, missao, origem }) {
  const nb = normalizarBytes(bytes);
  if (!nb.ok) return { ok: false, motivo: nb.motivo };
  const c = classificar(nome, nb.bytes.byteLength);
  if (!c.ok) return { ok: false, motivo: c.motivo };
  const m = {
    protocolo: PROTOCOLO_ARQUIVOS,
    nome: c.nome,
    extensao: c.extensao,
    direcao,
    tamanho: c.tamanho,
    sha256: sha256 ?? null,
    missao: missao ?? null,
    origem: origem ?? null,
  };
  const json = JSON.stringify(m);
  if (json.length > LIMITES_ARQUIVOS.manifestoBytes) return { ok: false, motivo: "manifesto_longo" };
  return { ok: true, manifesto: m };
}

/** Resumo curto e sem segredo, para o audit log e para o popup. */
export function resumoDaDecisao(d) {
  if (!d || typeof d !== "object") return "sem decisao";
  const partes = [d.veredicto, d.risco, d.direcao, d.extensao || "sem-extensao"].filter(Boolean);
  return partes.join(" / ");
}
