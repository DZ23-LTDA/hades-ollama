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

/**
 * HKDF-SHA256 para DERIVAR a chave do canal a partir do codigo curto do pairing.
 *
 * POR QUE ISTO EXISTE (`E-1024`): o contrato do `E-1023` usava HMAC **simetrico** e
 * apenas **assumia** que a chave tinha sido partilhada com seguranca -- o que, em
 * seguranca, e' o mesmo que nao ter contrato nenhum. Aqui a chave **deriva-se**.
 *
 * O `salt` e o `info` NAO sao decoracao: sem eles, o mesmo codigo curto produziria a
 * MESMA chave em instalacoes diferentes, e um segredo de 6 digitos reutilizado entre
 * dispositivos e' um segredo de 6 digitos para todos eles.
 *
 * LIMITE HONESTO: HKDF **nao adiciona entropia**. Um codigo de 6 digitos tem ~20 bits;
 * derivar nao o torna forte. O que protege e' o codigo ser **de uso unico**, ter
 * **prazo de 3 minutos**, **teto de 5 tentativas** e nunca atravessar um canal publico.
 */
export const HKDF = "HKDF";
export const HKDF_BITS = 256;

/** Sem contexto, a chave derivada e' so' um codigo curto. O contexto separa-os. */
export function contextoDoPairing(pedido, instalacao, missao = null) {
  return new TextEncoder().encode(
    ["hades-canal", "v" + PROTOCOLO, pedido, instalacao, missao ?? "-"].join("\u0000"),
  );
}

/* -------------------------------------------- prova de posse e trafego (E-1025) */

/**
 * O DESAFIO. Um nonce que o outro lado tem de ASSINAR antes de o host arrancar.
 *
 * POR QUE ISTO EXISTE (`E-1025`): o `E-1024` derivou a chave, mas **nada provava que
 * o portador a tinha**. Um host que aceita a primeira conexao e depois valida o MAC
 * esta' a fazer o trabalho caro antes de saber com quem fala -- e num canal `loopback`
 * sem TLS o codigo curto de 6 digitos viaja EM CLARO no `localhost`. O desafio resolve
 * a segunda metade: quem nao tem a chave nao passa, mesmo que tenha ouvido o codigo.
 *
 * O nonce tem 32 bytes. E' **de uso unico** e **expira**: um desafio reutilizado e' um
 * replay, e um desafio eterno e' uma assinatura que nunca caduca.
 */
export const DESAFIO_BYTES = 32;
export const TTL_DESAFIO_MS = 30 * 1000;

/** Limites de taxa. Sem isto, um copo de 10^6 codigos e' questao de tempo. */
export const LIMITES_DE_TAXA = {
  tentativasPorPedido: 5,
  tentativasPorInstalacao: 20,
  janelaMs: 60 * 1000,
  conexoesSimultaneas: 1,
};

/** Um desafio novo. Nonce aleatorio, uso unico, com prazo. */
export function abrirDesafio(instalacao, agora = Date.now()) {
  if (typeof instalacao !== "string" || !instalacao) {
    return { ok: false, motivo: "instalacao_ausente" };
  }
  return {
    ok: true,
    desafio: {
      nonce: Array.from(aleatorio(DESAFIO_BYTES), (b) => b.toString(16).padStart(2, "0")).join(""),
      instalacao,
      criadoEm: agora,
      expiraEm: agora + TTL_DESAFIO_MS,
      usado: false,
    },
  };
}

/**
 * A PROVA DE POSSE: assina o desafio com a chave derivada.
 * Isto e' o que separa "ouviu o codigo" de "tem a chave".
 *
 * `agora` e' parametro explicito porque TODO o resto do contrato o recebe: usar
 * `Date.now()` escondido aqui tornava o prazo do desafio impossivel de testar sem
 * esperar 30 segundos reais, e uma expiracao que so' se testa esperando e' uma
 * expiracao que ninguem testa.
 *
 * `canonicoDoDesafio` e' PROPOSITAL e nao usa `canonico` (`E-1026`): o `canonico`
 * le' campos de um ENVELOPE (`tipoCanal`, `sequencia`, `instante`, `corpo`). Assinar
 * uma string com ele produziria o MAC de `v1\0undefined\0undefined\0...` -- **uma
 * constante igual para todos os desafios**, e o handshake aceitaria qualquer prova.
 * Um objeto desafio tem a sua propria forma, logo tem a sua propria canonizacao.
 */
export function canonicoDoDesafio(nonce, instalacao) {
  return new TextEncoder().encode(
    ["hades-desafio", "v" + PROTOCOLO, instalacao, nonce].join("\u0000"),
  );
}

/** Assina o desafio. Dois argumentos, ambos usados -- ver `E-1026`. */
export async function assinarDesafio(chave, nonce, instalacao) {
  const mac = await crypto.subtle.sign(ALG, chave, canonicoDoDesafio(nonce, instalacao));
  return Array.from(new Uint8Array(mac), (x) => x.toString(16).padStart(2, "0")).join("");
}

/** Verifica a prova. Dois argumentos, ambos usados -- ver `E-1026`. */
export async function verificarDesafio(chave, nonce, instalacao, prova) {
  if (typeof prova !== "string" || !prova) return false;
  const esperado = await assinarDesafio(chave, nonce, instalacao);
  if (esperado.length !== prova.length) return false;
  let dif = 0;
  for (let i = 0; i < esperado.length; i++) {
    dif |= esperado.charCodeAt(i) ^ prova.charCodeAt(i);
  }
  return dif === 0;
}

export async function provarPosse(chave, desafio, agora = Date.now()) {
  if (!desafio || typeof desafio.nonce !== "string" || !desafio.nonce) {
    throw new Error("sem desafio: nao ha o que provar");
  }
  if (desafio.usado === true) throw new Error("desafio ja usado");
  if (agora >= desafio.expiraEm) throw new Error("desafio expirado");
  return assinarDesafio(chave, desafio.nonce, desafio.instalacao);
}

/**
 * A FRONTEIRA DO HANDSHAKE. O host so' abre o canal depois disto devolver `ok`.
 *
 * Recusa por ordem, do mais barato ao mais caro: estrutura -> instalacao -> desafio
 * -> expiracao -> uso unico -> taxa -> assinatura. E marca o desafio como USADO
 * DEPOIS de a assinatura conferir: um desafio consumido por uma prova errada deixaria
 * qualquer um a queimar o nonce do vizinho.
 */
export async function aceitarHandshake(recebido, estado, agora = Date.now()) {
  if (!recebido || typeof recebido !== "object") return { ok: false, motivo: "handshake_malformado" };
  const { instalacao, nonce, prova } = recebido;
  if (typeof instalacao !== "string" || !instalacao) return { ok: false, motivo: "instalacao_ausente" };
  if (typeof nonce !== "string" || !nonce) return { ok: false, motivo: "nonce_ausente" };
  if (typeof prova !== "string" || !prova) return { ok: false, motivo: "prova_ausente" };

  const registro = estado.dispositivos[instalacao];
  if (!registro) return { ok: false, motivo: "instalacao_desconhecida" };
  // O CAMPO E' `revogada` e nao `revogadaEm` (`E-1026`): eu inventei o nome, o
  // `undefined` era sempre falsy, e a guarda de revogacao estava MORTA -- o erro
  // so' apareceu porque `importarChave(new Uint8Array(0))` explodiu a seguir.
  if (registro.revogada === true) return { ok: false, motivo: "dispositivo_revogado" };
  if (agora >= registro.expiraEm) return { ok: false, motivo: "chave_expirada" };

  const d = estado.desafios[nonce];
  if (!d) return { ok: false, motivo: "desafio_desconhecido" };
  if (d.instalacao !== instalacao) return { ok: false, motivo: "desafio_de_outra_instalacao" };
  if (d.usado === true) return { ok: false, motivo: "desafio_ja_usado" };
  // `>=`: um desafio NO instante de expiracao esta' expirado. Fail-closed.
  if (agora >= d.expiraEm) return { ok: false, motivo: "desafio_expirado" };

  // Taxa: barata de medir, e ANTES da verificacao criptografica.
  const taxa = estado.taxa[instalacao] ?? (estado.taxa[instalacao] = []);
  const recentes = taxa.filter((t) => agora - t < LIMITES_DE_TAXA.janelaMs);
  if (recentes.length >= LIMITES_DE_TAXA.tentativasPorInstalacao) {
    estado.taxa[instalacao] = recentes;
    return { ok: false, motivo: "taxa_excedida" };
  }
  recentes.push(agora);
  estado.taxa[instalacao] = recentes;

  const chave = await importarChave(registro.chave);
  if (!(await verificarDesafio(chave, d.nonce, instalacao, prova))) {
    return { ok: false, motivo: "prova_invalida" };
  }
  // So' DEPOIS de a prova conferir e' que o nonce morre.
  d.usado = true;
  return { ok: true, instalacao, missao: registro.missao ?? null };
}

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
 * DERIVA a chave do canal a partir do codigo curto + contexto.
 *
 * O codigo curto NUNCA e' usado directamente como chave: passa por HKDF-SHA256 com
 * `salt` e `info` derivados do pedido, da instalacao e da missao. Assim:
 *   - o mesmo codigo em instalacoes diferentes da' chaves DIFERENTES;
 *   - a chave nao pode ser testada offline contra um dicionario de 10^6 codigos
 *     sem conhecer o pedido e a instalacao (que sao aleatorios por pairing);
 *   - a chave derivada tem 256 bits, mesmo que o codigo tenha ~20.
 */
export async function derivarChave(codigoCurto, pedido, instalacao, missao = null) {
  if (typeof codigoCurto !== "string" || !codigoCurto) {
    throw new Error("codigo curto ausente: nao derive chave de nada");
  }
  const base = await crypto.subtle.importKey(
    "raw", new TextEncoder().encode(codigoCurto), HKDF, false, ["deriveBits"],
  );
  const bits = await crypto.subtle.deriveBits(
    {
      name: HKDF,
      hash: HASH,
      salt: contextoDoPairing(pedido, instalacao, missao),
      info: new TextEncoder().encode("hades-canal-chave"),
    },
    base,
    HKDF_BITS,
  );
  return importarChave(new Uint8Array(bits));
}

/**
 * O codigo curto morre DEPOIS de ser usado. Um codigo de 6 digitos que continua
 * valido depois do pairing e' uma porta aberta com prazo de 3 minutos.
 */
export function consumirCodigo(pedido) {
  if (!pedido || typeof pedido !== "object") return { ok: false, motivo: "pedido_ausente" };
  if (pedido.consumido === true) return { ok: false, motivo: "codigo_ja_usado" };
  pedido.consumido = true;
  pedido.codigo = null;
  return { ok: true };
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
  // Um codigo de USO UNICO: depois de consumido nao volta a valer, nem com o
  // codigo certo. Sem isto, um codigo de 6 digitos reutilizado e' uma porta aberta.
  if (pedido.consumido === true) return { ok: false, motivo: "codigo_ja_usado" };
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
  return { dispositivos: {}, desafios: {}, taxa: {}, protocolo: PROTOCOLO };
}
