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
 * A IMPRESSAO DIGITAL DO PEDIDO -- o `fingerprint` do Mission Control (`E-1041`).
 *
 * O backend tem `FindApproved(organizationID, fingerprint, now)`: procura uma aprovacao
 * **aprovada, com quorum e nao consumida** para um pedido identificado por impressao. E o
 * `Consume` garante **uso unico**.
 *
 * A extensao nao tem acesso ao store -- mas pode exigir a **mesma forma de prova**. *Um
 * booleano `aprovado: true` que quem pede controla nao e' uma aprovacao: e' uma afirmacao.*
 *
 * A impressao tem de cobrir o que a decisao decide: direccao, nome, tamanho e hash. *Uma
 * aprovacao para `a.csv` de 12 bytes nao pode servir para `b.exe` de 2 MB.*
 */
export function impressaoDoPedido(pedido) {
  const p = pedido ?? {};
  const partes = [
    "v1",
    String(p.direcao ?? ""),
    String(p.nome ?? ""),
    String(p.tamanho ?? ""),
    String(p.sha256 ?? ""),
  ];
  return partes.join("|");
}

/** SHA-256 da impressao, em hex -- e' isto que casa com o `fingerprint` do Mission Control. */
export async function fingerprintDe(pedido) {
  const texto = impressaoDoPedido(pedido);
  const bytes = new TextEncoder().encode(texto);
  const digest = await crypto.subtle.digest("SHA-256", bytes);
  const hex = Array.from(new Uint8Array(digest)).map((b) => b.toString(16).padStart(2, "0")).join("");
  return { ok: true, fingerprint: hex, impressao: texto };
}

/** A forma minima de uma prova de aprovacao. *Nada de booleanos soltos.* */
export function provaValida(prova) {
  if (!prova || typeof prova !== "object") return { ok: false, motivo: "aprovacao_ausente" };
  const { aprovacaoId, fingerprint, aprovadoPor } = prova;
  if (typeof aprovacaoId !== "string" || aprovacaoId.trim() === "") {
    return { ok: false, motivo: "aprovacao_sem_id" };
  }
  if (typeof fingerprint !== "string" || !/^[a-f0-9]{64}$/.test(fingerprint)) {
    return { ok: false, motivo: "aprovacao_sem_fingerprint" };
  }
  if (typeof aprovadoPor !== "string" || aprovadoPor.trim() === "") {
    return { ok: false, motivo: "aprovacao_sem_actor" };
  }
  return { ok: true, prova: { aprovacaoId, fingerprint, aprovadoPor } };
}

/**
 * A AUTORIDADE DA APROVACAO (`E-1042`).
 *
 * Medido: existe uma rota autenticada por **sessao** (`serviceAgentApprovals`), e ela usa
 * `store.ListApprovals(organizationID, ...)`. Mas o **Companion e' um processo local SEM
 * sessao HTTP** -- e dar-lhe a sessao do utilizador seria pior: *um processo local com a
 * sessao do utilizador e' exactamente o que nao queremos.*
 *
 * Portanto a extensao **nao pode verificar** a prova contra o store. E o contrato tem de o
 * DIZER, em vez de o esconder.
 *
 *   `forma_apenas`  -- a prova tem a forma certa; **ninguem confirmou que existe**. E' o
 *                      unico valor que a extensao pode produzir hoje.
 *   `verificada`    -- alguem consultou o store e o `FindApproved` devolveu esta aprovacao.
 *                      **Sera' o Companion a produzi-lo, quando a rota existir la'.**
 *
 * *Um campo que diz `verificada` sem ter sido verificado e' pior do que nao ter campo.*
 */
export const AUTORIDADES = Object.freeze(["forma_apenas", "verificada"]);

/** A extensao NAO tem autoridade de aprovacao. Este e' o valor honesto. */
export const AUTORIDADE_DA_EXTENSAO = "forma_apenas";

/**
 * A DECISAO. Recebe o manifesto e a PROVA de aprovacao (nao um booleano).
 *
 * `E-1041`: a versao anterior aceitava `opcoes.aprovado === true`. ***Um campo que quem pede
 * controla nao e' uma aprovacao.*** Agora e' preciso:
 *   * `aprovacaoId` -- a aprovacao concreta, para poder ser consumida uma vez;
 *   * `fingerprint` -- e tem de ser **o fingerprint DESTE pedido**;
 *   * `aprovadoPor` -- *uma aprovacao sem autor nao tem responsavel.*
 *
 * Regras, na ordem em que sao aplicadas -- *a ordem e' a politica*:
 *   1. nome invalido               -> proibido;
 *   2. direccao invalida           -> proibido;
 *   3. sem sha256 quando exigido   -> proibido (nao se aprova o que nao se mediu);
 *   4. risco executavel            -> exige_aprovacao;
 *   5. risco sensivel              -> exige_aprovacao;
 *   6. upload de qualquer ficheiro -> exige_aprovacao;
 *   7. risco desconhecido          -> exige_aprovacao;
 *   8. inerte + download           -> permitido.
 *
 * Para as regras 4..7, o `exige_aprovacao` so' se torna `autorizado` com uma prova cujo
 * `fingerprint` **casa com este pedido**.
 *
 * E TODO o veredicto `autorizado` traz `autoridade` -- que hoje e' sempre `forma_apenas`.
 */
export function decidir(manifesto, opcoes = {}) {
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
    // A autoridade acompanha SEMPRE a decisao -- nao so' o caso autorizado.
    autoridade: opcoes.autoridade === "verificada" ? "verificada" : AUTORIDADE_DA_EXTENSAO,
  };

  // A prova e' avaliada UMA vez, e so' se for preciso.
  let prova = null;
  let motivoDaProva = null;
  if (opcoes.prova !== undefined && opcoes.prova !== null) {
    const pv = provaValida(opcoes.prova);
    if (!pv.ok) motivoDaProva = pv.motivo;
    else prova = pv.prova;
  } else if (opcoes.aprovado === true) {
    // *Compatibilidade explicita e' um erro, nao um atalho.* Um booleano solto diz que
    // alguem afirmou; nao diz quem, nem para que pedido.
    motivoDaProva = "booleano_solto_nao_e_aprovacao";
  }

  /** Decide se a prova autoriza ESTE pedido. */
  const autoriza = () => {
    if (!prova) return { autorizado: false, motivo: motivoDaProva ?? "aprovacao_ausente" };
    if (prova.fingerprint !== m.fingerprint) {
      // *Uma aprovacao para outro pedido nao serve.* E o fingerprint tem de vir medido.
      return {
        autorizado: false,
        motivo: m.fingerprint ? "fingerprint_nao_corresponde" : "fingerprint_do_pedido_ausente",
      };
    }
    return { autorizado: true, motivo: null };
  };

  const comAprovacao = (motivoSemProva) => {
    const a = autoriza();
    if (a.autorizado) {
      return {
        veredicto: "autorizado", motivo: motivoSemProva + "_com_aprovacao",
        aprovacaoId: prova.aprovacaoId, aprovadoPor: prova.aprovadoPor,
        fingerprint: prova.fingerprint,
        // *A extensao NUNCA afirma `verificada`: ela nao consultou o store.*
        autoridade: opcoes.autoridade === "verificada" ? "verificada" : AUTORIDADE_DA_EXTENSAO,
        ...base,
      };
    }
    return { veredicto: "exige_aprovacao", motivo: motivoSemProva, aprovacao: a.motivo, ...base };
  };

  if (c.risco === "executavel") return comAprovacao("executavel_exige_aprovacao");
  if (c.risco === "sensivel") return comAprovacao("ficheiro_sensivel_exige_aprovacao");
  if (direcao === "upload") return comAprovacao("upload_exige_aprovacao");
  if (c.risco === "desconhecido") return comAprovacao("extensao_desconhecida_exige_aprovacao");
  return { veredicto: "permitido", motivo: "conteudo_inerte_em_download", ...base };
}

/**
 * O TEXTO QUE DIZ O LIMITE (`E-1042`). *Um limite que nao se le' nao e' um limite.*
 * Usado no popup e no registo, para que quem decide saiba o que a decisao vale.
 */
export function avisoDeAutoridade(autoridade) {
  if (autoridade === "verificada") {
    return "Aprovacao verificada no store do Mission Control.";
  }
  return "Aprovacao com FORMA valida, mas NAO verificada: a extensao nao tem acesso ao store "
    + "de aprovacoes. Uma prova bem formada mas inexistente passaria aqui.";
}

/**
 * Monta o manifesto canonico, **com o fingerprint** (`E-1041`).
 *
 * O `sha256` e o `fingerprint` tem de vir MEDIDOS. *Quem pede nao certifica o que pede* -- e
 * por isso o fingerprint e' derivado do proprio pedido normalizado, nao aceite de fora.
 */
export async function manifesto({ nome, direcao, bytes, sha256, missao, origem }) {
  const nb = normalizarBytes(bytes);
  if (!nb.ok) return { ok: false, motivo: nb.motivo };
  const c = classificar(nome, nb.bytes.byteLength);
  if (!c.ok) return { ok: false, motivo: c.motivo };
  if (typeof sha256 !== "string" || !/^[a-f0-9]{64}$/.test(sha256)) {
    // *Um manifesto sem hash medido nao e' um manifesto: e' uma descricao.*
    return { ok: false, motivo: "sha256_ausente_ou_invalido" };
  }
  const fp = await fingerprintDe({
    direcao, nome: c.nome, tamanho: c.tamanho, sha256,
  });
  const m = {
    protocolo: PROTOCOLO_ARQUIVOS,
    nome: c.nome,
    extensao: c.extensao,
    direcao,
    tamanho: c.tamanho,
    sha256,
    fingerprint: fp.fingerprint,
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
