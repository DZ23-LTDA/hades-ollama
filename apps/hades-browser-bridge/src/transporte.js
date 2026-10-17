/**
 * HADES Browser Bridge -- TRANSPORTE (`E-1027`).
 *
 * O QUE FALTAVA: o `E-1025` provou **quem** fala, mas nao havia **por onde**. O
 * contrato do canal, o handshake e os envelopes existiam sem transporte nenhum.
 * Este modulo e' o fio: o enquadramento do native messaging (`stdio`) e um canal
 * em memoria que faz o protocolo COMPLETO passar pelas mesmas funcoes.
 *
 * ENQUADRAMENTO (native messaging, Chrome/Edge/Firefox):
 *   4 bytes em LITTLE-ENDIAN = comprimento do JSON em UTF-8, depois o JSON.
 *   O limite do lado do browser e' 1 MB para mensagens do host; um limite menor
 *   aqui e' uma decisao, nao um espelho: 256 KiB e' o mesmo `LIMITES.mensagemBytes`
 *   do contrato, para nao haver dois numeros a divergir.
 *
 * ERROS TIPADOS, e nao `throw` de string: uma mensagem curta no meio de um fluxo e'
 * um ERRO (o fluxo morreu), nao um "ainda nao chegou". Confundir os dois e' a
 * diferenca entre esperar e travar para sempre.
 */

/** Cabecalho: 4 bytes little-endian com o comprimento do corpo. */
export const CABECALHO_BYTES = 4;

/** Teto do corpo de uma mensagem. Igual ao contrato, para nao divergir. */
export const MAX_CORPO_BYTES = 256 * 1024;

/** Códigos de erro. Um numero estavel, para o outro lado poder decidir. */
export const ERROS = {
  CURTO: "fluxo_curto",
  LONGO: "fluxo_longo",
  VAZIO: "corpo_vazio",
  JSON: "json_invalido",
  FORMA: "forma_invalida",
};

export class ErroDeTransporte extends Error {
  constructor(codigo, detalhe) {
    super(codigo + (detalhe ? ": " + detalhe : ""));
    this.name = "ErroDeTransporte";
    this.codigo = codigo;
    this.detalhe = detalhe ?? null;
  }
}

/**
 * Enquadra um objeto: 4 bytes de comprimento + JSON UTF-8.
 * `Uint8Array` de saida, pronto para `process.stdout`/`port.postMessage`.
 */
export function enquadrar(mensagem) {
  if (mensagem === null || typeof mensagem !== "object" || Array.isArray(mensagem)) {
    throw new ErroDeTransporte(ERROS.FORMA, "esperado objeto, veio " + (Array.isArray(mensagem) ? "array" : typeof mensagem));
  }
  let corpo;
  try {
    corpo = new TextEncoder().encode(JSON.stringify(mensagem));
  } catch (e) {
    throw new ErroDeTransporte(ERROS.FORMA, "nao serializavel");
  }
  if (corpo.length === 0) throw new ErroDeTransporte(ERROS.VAZIO);
  if (corpo.length > MAX_CORPO_BYTES) {
    throw new ErroDeTransporte(ERROS.LONGO, corpo.length + " > " + MAX_CORPO_BYTES);
  }
  const saida = new Uint8Array(CABECALHO_BYTES + corpo.length);
  // LITTLE-ENDIAN explicito: `DataView` com `true`. Uma ordem implicita da' um
  // numero gigante do outro lado, e o erro aparece como "mensagem longa" -- longe
  // da causa.
  new DataView(saida.buffer).setUint32(0, corpo.length, true);
  saida.set(corpo, CABECALHO_BYTES);
  return saida;
}

/**
 * Desenquadra o que ja' chegou. Devolve as mensagens COMPLETAS e o resto.
 * Um corpo parcial NAO e' erro: e' o normal de um fluxo -- por isso o resto sai
 * em vez de se lancar. So' o que ja' se sabe ser invalido lanca.
 */
export function desenquadrar(buffer) {
  if (!(buffer instanceof Uint8Array)) {
    throw new ErroDeTransporte(ERROS.FORMA, "esperado Uint8Array");
  }
  const mensagens = [];
  let p = 0;
  while (buffer.length - p >= CABECALHO_BYTES) {
    const n = new DataView(buffer.buffer, buffer.byteOffset + p, CABECALHO_BYTES).getUint32(0, true);
    if (n === 0) throw new ErroDeTransporte(ERROS.VAZIO);
    if (n > MAX_CORPO_BYTES) throw new ErroDeTransporte(ERROS.LONGO, n + " > " + MAX_CORPO_BYTES);
    if (buffer.length - p - CABECALHO_BYTES < n) break; // corpo parcial: espera
    const corpo = buffer.subarray(p + CABECALHO_BYTES, p + CABECALHO_BYTES + n);
    let obj;
    try {
      obj = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(corpo));
    } catch (e) {
      // UTF-8 invalido OU JSON invalido: nos dois casos o fluxo esta' dessincronizado
      // e nao ha como recuperar sem adivinhar onde acaba a mensagem.
      throw new ErroDeTransporte(ERROS.JSON, String(e.message ?? e).slice(0, 80));
    }
    if (obj === null || typeof obj !== "object" || Array.isArray(obj)) {
      throw new ErroDeTransporte(ERROS.FORMA, "corpo nao e' objeto");
    }
    mensagens.push(obj);
    p += CABECALHO_BYTES + n;
  }
  return { mensagens, resto: buffer.slice(p) };
}

/**
 * CANAL EM MEMORIA. Nao ha socket nem processo: o "fio" e' um `Uint8Array` que
 * passa pelas MESMAS funcoes de enquadramento. Serve para exercitar o protocolo
 * inteiro -- pairing, handshake, envelope, replay -- **fim a fim**, e para medir
 * o que acontece quando o fluxo chega partido em pedacos de 1 byte.
 *
 * LIMITE HONESTO: e' memoria, nao `stdio`. Nao prova permissao, nao prova
 * `allowed_origins`, nao prova nada do SO. Prova o protocolo.
 */
export function canalEmMemoria() {
  let pendente = new Uint8Array(0);
  const recebidas = [];
  return {
    /** Escreve bytes como um `stdout` faria. */
    escrever(bytes) {
      const juntos = new Uint8Array(pendente.length + bytes.length);
      juntos.set(pendente, 0);
      juntos.set(bytes, pendente.length);
      const r = desenquadrar(juntos);
      pendente = r.resto;
      for (const m of r.mensagens) recebidas.push(m);
      return r.mensagens.length;
    },
    /** Envia um objeto (enquadra + escreve). */
    enviar(mensagem) {
      return this.escrever(enquadrar(mensagem));
    },
    /** O que chegou inteiro ate' agora. */
    recebidas() {
      return recebidas.slice();
    },
    /** Quantos bytes ficaram a meio -- tem de ser 0 no fim de um fluxo bem formado. */
    pendentes() {
      return pendente.length;
    },
    /** Limpa o estado (para o teste do fluxo partido). */
    limpar() {
      pendente = new Uint8Array(0);
      recebidas.length = 0;
    },
  };
}
