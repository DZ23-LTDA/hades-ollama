// HADES Browser Bridge -- contrato do CANAL LOCAL (pairing).
//
// Este ficheiro NAO abre socket, NAO fala com o Companion e NAO decide aprovacao.
// Ele define e FAZ CUMPRIR o contrato que um canal local (native messaging ou
// loopback autenticado) tem de satisfazer antes de a extensao confiar nele.
//
// POR QUE ISTO EXISTE
//   O `E-1022` fechou o pacote, mas o pairing continua a NAO existir: o popup mostra
//   "host: - (native host nao instalado)". Escrever o contrato AGORA, com teste, e' o
//   trabalho interno que falta -- e e' o que impede que o canal futuro nasça com
//   token estatico, replay ou ligacao publica.
//
// AS REGRAS QUE ESTE MODULO FAZ CUMPRIR
//   1. NUNCA ha token estatico no codigo nem no pacote. O segredo nasce em runtime.
//   2. O pairing e' iniciado pelo UTILIZADOR e confirmado por codigo curto.
//   3. Cada dispositivo tem chave ALEATORIA propria; ela expira e roda.
//   4. Toda mensagem tem schema, tamanho maximo, origem, instalacao e SEQUENCIA.
//   5. Mensagem repetida, fora de ordem ou expirada e' RECUSADA (anti-replay).
//   6. So' loopback. Nunca LAN, nunca rede publica.
//   7. Revogar e' instantaneo e nao depende do host.
//
// O canal REAL (native host) ainda nao existe: este contrato e' a fronteira. Quando
// ele existir, tem de passar por `aceitar()` sem alterar o modulo.

/** Versao do protocolo. Uma mudanca incompativel sobe isto. */
export const PROTOCOLO = 1;

/** Algoritmo de assinatura. HMAC-SHA256, disponivel em WebCrypto no service worker. */
export const ALG = "HMAC";
export const HASH = "SHA-256";

/** Origens de canal aceitas. `native` e' o preferido; loopback e' a alternativa. */
export const TIPOS_DE_CANAL = ["native", "loopback"];

/** Hosts de loopback permitidos. NAO ha entrada para LAN nem para rede publica. */
// Em Node, new URL("http://[::1]:9").hostname devolve "[::1]" COM brackets.
// Medido: com `::1` sem brackets, o loopback IPv6 era RECUSADO. Aceita as duas formas.
export const HOSTS_LOOPBACK = ["127.0.0.1", "::1", "[::1]", "localhost"];

/** Limites duros. Uma mensagem acima disto e' recusada antes de ser lida. */
export const LIMITES = {
  mensagemBytes: 256 * 1024,
  identificador: 128,
  codigoCurtoDigitos: 6,
  tentativasDePairing: 5,
  desvioDeRelogioMs: 2 * 60 * 1000,
};

/** Validade do codigo de pairing: curta, porque e' um segredo de 6 digitos. */
export const TTL_CODIGO_MS = 3 * 60 * 1000;

/** Validade da chave de dispositivo. Roda antes de expirar. */
export const TTL_CHAVE_MS = 24 * 60 * 60 * 1000;

/** Onde a sequencia comeca. A sequencia e' estritamente crescente. */
export const SEQUENCIA_INICIAL = 1;

/* ------------------------------------------------------------- aleatoriedade */

/**
 * Bytes aleatorios com WebCrypto. `Math.random` NAO serve para segredo: e'
 * previsivel. Se `crypto.getRandomValues` nao existir, isto FALHA em vez de
 * degradar para algo inseguro.
 */
export function aleatorio(bytes) {
  if (typeof crypto === "undefined" || typeof crypto.getRandomValues !== "function") {
    throw new Error("sem WebCrypto: nao gere segredo com Math.random");
  }
  const b = new Uint8Array(bytes);
  crypto.getRandomValues(b);
  return b;
}

/** Codigo curto de 6 digitos, derivado de aleatoriedade real. */
export function gerarCodigoCurto() {
  const b = aleatorio(4);
  const n = ((b[0] << 24) | (b[1] << 16) | (b[2] << 8) | b[3]) >>> 0;
  return String(n % 1000000).padStart(LIMITES.codigoCurtoDigitos, "0");
}

/** Identificador opaco, sem significado. */
export function gerarIdentificador(prefixo) {
  const b = aleatorio(16);
  return prefixo + Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("");
}

/* ------------------------------------------------------------------- segredo */

/** Uma chave de dispositivo. Nunca e' serializada como texto depois de criada. */
export async function gerarChave() {
  return crypto.subtle.generateKey({ name: ALG, hash: HASH }, true, ["sign", "verify"]);
}

/** Exporta a chave para bytes. So' o chamador decide onde guardar (nunca em claro fora do storage seguro). */
export async function exportarChave(chave) {
  return new Uint8Array(await crypto.subtle.exportKey("raw", chave));
}

/** Importa uma chave ja' existente. */
export async function importarChave(bytes) {
  return crypto.subtle.importKey("raw", bytes, { name: ALG, hash: HASH }, true, ["sign", "verify"]);
}

/**
 * Assinatura de uma mensagem canonica.
 * A mensagem canonica inclui TUDO o que identifica o contexto: protocolo, tipo de
 * canal, instalacao, missao, sequencia, instante e corpo. Se um destes mudar, a
 * assinatura deixa de valer -- e' isso que impede troca de contexto.
 */
export async function assinar(chave, envelope) {
  const mac = await crypto.subtle.sign(ALG, chave, canonico(envelope));
  return Array.from(new Uint8Array(mac), (x) => x.toString(16).padStart(2, "0")).join("");
}

/** Verificacao em tempo constante, para nao vazar o MAC por comparacao. */
export async function verificar(chave, envelope) {
  if (typeof envelope.mac !== "string") return false;
  const esperado = await assinar(chave, envelope);
  if (esperado.length !== envelope.mac.length) return false;
  let dif = 0;
  for (let i = 0; i < esperado.length; i++) {
    dif |= esperado.charCodeAt(i) ^ envelope.mac.charCodeAt(i);
  }
  return dif === 0;
}

/** Representacao canonica: ordem fixa, sem espacos, sem surpresa. */
export function canonico(e) {
  const texto = [
    "v" + PROTOCOLO,
    e.tipoCanal,
    e.instalacao,
    e.missao ?? "-",
    "s" + e.sequencia,
    "t" + e.instante,
    e.corpo,
  ].join("\u0000");
  return new TextEncoder().encode(texto);
}

/* --------------------------------------------------------------------- canal */

/** Um host de canal so' e' aceite se for native, ou loopback com host permitido. */
export function canalPermitido(url) {
  if (typeof url !== "string" || !url) return { ok: false, motivo: "canal_ausente" };
  if (url === "native") return { ok: true, tipo: "native" };
  let u;
  try {
    u = new URL(url);
  } catch {
    return { ok: false, motivo: "canal_invalido" };
  }
  if (u.protocol !== "http:" && u.protocol !== "https:") {
    return { ok: false, motivo: "canal_esquema_nao_permitido" };
  }
  if (!HOSTS_LOOPBACK.includes(u.hostname)) {
    // LAN e rede publica recusadas SEM opcao: nao ha flag que as habilite.
    return { ok: false, motivo: "canal_nao_e_loopback" };
  }
  if (u.username || u.password) return { ok: false, motivo: "canal_com_credenciais" };
  return { ok: true, tipo: "loopback" };
}

/* ------------------------------------------------------------------- pairing */

/**
 * Abre um pedido de pairing. Devolve o codigo curto e o id do pedido; guarda o
 * segredo esperado, o prazo e o contador de tentativas.
 */
export function abrirPairing(agora = Date.now()) {
  return {
    pedido: gerarIdentificador("pr-"),
    codigo: gerarCodigoCurto(),
    expiraEm: agora + TTL_CODIGO_MS,
    tentativas: 0,
  };
}

/**
 * Confirma o pairing. Recusa codigo errado, codigo expirado e forca bruta:
 * o contador de tentativas tem teto, e ao esgotar o pedido morre.
 */
export function confirmarPairing(pedido, codigo, agora = Date.now()) {
  if (!pedido || typeof pedido !== "object") return { ok: false, motivo: "pedido_ausente" };
  if (agora > pedido.expiraEm) return { ok: false, motivo: "codigo_expirado" };
  if (pedido.tentativas >= LIMITES.tentativasDePairing) {
    return { ok: false, motivo: "tentativas_esgotadas" };
  }
  pedido.tentativas += 1;
  if (typeof codigo !== "string" || codigo !== pedido.codigo) {
    return { ok: false, motivo: "codigo_incorreto", tentativas: pedido.tentativas };
  }
  return { ok: true, pedido: pedido.pedido };
}

/* ------------------------------------------------------------------ envelope */

/**
 * Cria o envelope de uma mensagem. `sequencia` tem de ser estritamente maior do
 * que a ultima aceita -- quem chama nao inventa o numero, o canal conta.
 */
export function envelope({ tipoCanal, instalacao, missao, sequencia, corpo, agora = Date.now() }) {
  return {
    v: PROTOCOLO,
    tipoCanal,
    instalacao,
    missao: missao ?? null,
    sequencia,
    instante: agora,
    corpo,
    mac: null,
  };
}

/**
 * A FRONTEIRA. Toda mensagem vinda do canal passa por aqui antes de ser tratada.
 *
 * Recusa, por ordem: envelope malformado -> tamanho -> origem do canal -> instalacao
 * desconhecida -> chave revogada -> relogio -> sequencia (replay/fora de ordem) ->
 * assinatura. A ordem importa: nada de caro (a assinatura) antes do barato.
 */
export async function aceitar(envelopeRecebido, estado, agora = Date.now()) {
  const e = envelopeRecebido;
  if (!e || typeof e !== "object") return { ok: false, motivo: "envelope_invalido" };
  if (e.v !== PROTOCOLO) return { ok: false, motivo: "protocolo_incompativel" };
  if (typeof e.tipoCanal !== "string" || !TIPOS_DE_CANAL.includes(e.tipoCanal)) {
    return { ok: false, motivo: "tipo_de_canal_invalido" };
  }
  if (typeof e.corpo !== "string") return { ok: false, motivo: "corpo_invalido" };
  if (new TextEncoder().encode(e.corpo).length > LIMITES.mensagemBytes) {
    return { ok: false, motivo: "mensagem_demasiado_grande" };
  }
  if (typeof e.instalacao !== "string" || !e.instalacao
      || e.instalacao.length > LIMITES.identificador) {
    return { ok: false, motivo: "instalacao_invalida" };
  }
  if (!Number.isInteger(e.sequencia) || e.sequencia < SEQUENCIA_INICIAL) {
    return { ok: false, motivo: "sequencia_invalida" };
  }
  if (!Number.isFinite(e.instante)) return { ok: false, motivo: "instante_invalido" };

  const par = estado?.dispositivos?.[e.instalacao];
  if (!par) return { ok: false, motivo: "instalacao_desconhecida" };
  if (par.revogada === true) return { ok: false, motivo: "instalacao_revogada" };
  if (typeof par.expiraEm === "number" && agora > par.expiraEm) {
    return { ok: false, motivo: "chave_expirada" };
  }
  if (Math.abs(agora - e.instante) > LIMITES.desvioDeRelogioMs) {
    return { ok: false, motivo: "relogio_fora_da_janela" };
  }
  // ANTI-REPLAY: a sequencia tem de ser ESTRITAMENTE maior. Igual = repetida.
  if (e.sequencia <= (par.ultimaSequencia ?? 0)) {
    return { ok: false, motivo: "replay_ou_fora_de_ordem", ultima: par.ultimaSequencia ?? 0 };
  }

  const chave = await importarChave(par.chave);
  if (!(await verificar(chave, e))) return { ok: false, motivo: "assinatura_invalida" };

  // So' DEPOIS de tudo valido a sequencia avanca. Um envelope recusado nao consome
  // numero: senao um atacante poderia queimar a sequencia com lixo.
  par.ultimaSequencia = e.sequencia;
  return { ok: true, instalacao: e.instalacao, missao: e.missao, corpo: e.corpo };
}

/* ---------------------------------------------------------------- ciclo de vida */

/** Registra um dispositivo pareado. A chave e' por dispositivo, nunca compartilhada. */
export function registrarDispositivo(estado, instalacao, chaveBytes, agora = Date.now()) {
  if (typeof instalacao !== "string" || !instalacao
      || instalacao.length > LIMITES.identificador) {
    return { ok: false, motivo: "instalacao_invalida" };
  }
  if (estado.dispositivos[instalacao]?.revogada === false) {
    return { ok: false, motivo: "instalacao_ja_registrada" };
  }
  estado.dispositivos[instalacao] = {
    chave: chaveBytes,
    registadaEm: agora,
    expiraEm: agora + TTL_CHAVE_MS,
    ultimaSequencia: 0,
    revogada: false,
  };
  return { ok: true };
}

/** Revoga. Instantaneo e local: nao depende do host responder. */
export function revogarDispositivo(estado, instalacao) {
  const d = estado.dispositivos[instalacao];
  if (!d) return { ok: false, motivo: "instalacao_desconhecida" };
  d.revogada = true;
  d.chave = new Uint8Array(0); // apaga o material da chave
  return { ok: true };
}

/** Roda a chave sem perder a identidade da instalacao. A sequencia reinicia. */
export async function rodarChave(estado, instalacao, agora = Date.now()) {
  const d = estado.dispositivos[instalacao];
  if (!d) return { ok: false, motivo: "instalacao_desconhecida" };
  if (d.revogada) return { ok: false, motivo: "instalacao_revogada" };
  const bytes = await exportarChave(await gerarChave());
  d.chave = bytes;
  d.expiraEm = agora + TTL_CHAVE_MS;
  d.ultimaSequencia = 0;
  return { ok: true, expiraEm: d.expiraEm };
}

/** Estado inicial vazio. Sem dispositivos, nada e' aceito -- o default e' fechado. */
export function estadoInicial() {
  return { dispositivos: {}, protocolo: PROTOCOLO };
}
