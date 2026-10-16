// HADES Browser Bridge -- service worker (MV3).
//
// REUTILIZA o contrato do runtime do HADES (internal/agent/browser.go e
// browser_helper.py) e NAO cria protocolo, aprovacao ou missao paralelos:
//   - os 3 scopes canonicos vivem em src/protocol.js
//   - session_id = id da MISSAO, igual a toolContext.MissionID
//   - requires_approval e' declarado aqui e DECIDIDO no HADES, nunca na extensao
//   - a extensao nunca executa shell nem toca no sistema operativo
//
// Regras que este ficheiro faz cumprir:
//   * uma aba so e' observada/controlada depois de o UTILIZADOR a conceder;
//   * a concessao e' por (aba, janela, origem) e tem prazo;
//   * badge persistente enquanto houver concedao ativa;
//   * desconectar/revogar apaga tudo, sem tocar em projetos do utilizador;
//   * incognito nunca e', sequer, alcancavel (manifest: incognito not_allowed);
//   * nada de eval, nada de codigo remoto.

import {
  SCOPES, TOOL, LIMITES, validarUrl, guardarRota, campoSensivel, loopbackAutorizado,
} from "./protocol.js";

/** Chave unica de armazenamento. Uma so' fonte de verdade. */
const CHAVE = "hades.bridge.v1";

/**
 * O OPT-IN de rede privada (`E-1033`).
 *
 * O protocolo ja' tinha `loopbackAutorizado()` -- com o MESMO nome da variavel do
 * `browser_helper.py` -- mas **ninguem o chamava**: o service worker validava o URL sem
 * opcoes, logo `permitirPrivado` era sempre `false` e o opt-in era **codigo morto**.
 *
 * E havia uma contradicao pior: o `content_scripts.matches` do manifest cobre **so'**
 * `127.0.0.1` e `localhost` -- exactamente as origens que o protocolo **sempre** recusa.
 * A unica origem onde o content script e' injectado era a unica que o HADES nao podia
 * usar. Duas metades coerentes, juntas impossiveis.
 *
 * Agora o opt-in e' do UTILIZADOR, explicito, persistente e revogavel -- em
 * `chrome.storage.local`, com o nome canonico, e nunca ligado por omissao.
 */
const CHAVE_OPTIN = "OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE";

async function lerOptIn() {
  try {
    const bruto = await chrome.storage.local.get(CHAVE_OPTIN);
    return loopbackAutorizado(bruto);
  } catch {
    return false; // fail-closed: sem conseguir ler, a rede privada continua recusada
  }
}

async function definirOptIn(ligar) {
  // Grava o valor CANONICO ("1"), o que `loopbackAutorizado` espera.
  await chrome.storage.local.set({ [CHAVE_OPTIN]: ligar ? "1" : "0" });
  return { ok: true, ligado: ligar === true };
}

/**
 * O nome do host de native messaging. Tem de ser o MESMO que o `name` do manifesto
 * (`native-host/com.dz23ltd.hades.bridge.json`) e que o nome registado no SO.
 */
const NOME_DO_HOST = "com.dz23ltd.hades.bridge";

/** Uma concessao expira sozinha. O utilizador nao fica exposto por esquecimento. */
const TTL_PADRAO_MS = 30 * 60 * 1000;

/** Teto de accoes por minuto, por concessao (backpressure). */
const JANELA_MS = 60 * 1000;

/* ------------------------------------------------------------------ estado */

async function lerEstado() {
  const bruto = await chrome.storage.session.get(CHAVE);
  const estado = bruto?.[CHAVE];
  if (!estado || typeof estado !== "object") {
    return { concessoes: {}, accoes: {}, missao: null, host: null, aprovado: false, takeover: {} };
  }
  return {
    concessoes: estado.concessoes ?? {},
    accoes: estado.accoes ?? {},
    missao: estado.missao ?? null,
    host: estado.host ?? null,
    aprovado: estado.aprovado === true,
    // *Um `lerEstado` que reconstroi o objecto com campos fixos APAGA todo o campo que
    // nao esteja na lista* -- o `takeover` do `E-1034` era perdido a cada leitura.
    takeover: estado.takeover ?? {},
  };
}

async function gravarEstado(estado) {
  await chrome.storage.session.set({ [CHAVE]: estado });
  await refletirBadges(estado);
}

/** A chave de uma concessao e' a aba: uma aba = uma concessao. */
const chaveDaAba = (tabId) => `t${tabId}`;

function expirou(concessao, agora = Date.now()) {
  return !concessao || typeof concessao.expiraEm !== "number" || concessao.expiraEm <= agora;
}

/**
 * Devolve a concessao viva de uma aba, ou null. Uma concessao expirada e'
 * removida -- nao ha' "reconexao que restaura permissao expirada".
 */
async function concessaoViva(tabId) {
  const estado = await lerEstado();
  const c = estado.concessoes[chaveDaAba(tabId)];
  if (!c) return null;
  if (expirou(c)) {
    delete estado.concessoes[chaveDaAba(tabId)];
    delete estado.accoes[chaveDaAba(tabId)];
    await gravarEstado(estado);
    return null;
  }
  return c;
}

/** Badge persistente e inequivoco enquanto a aba estiver concedida. */
async function refletirBadges(estado) {
  const agora = Date.now();
  for (const [chave, c] of Object.entries(estado.concessoes)) {
    const tabId = Number(chave.slice(1));
    if (!Number.isFinite(tabId)) continue;
    if (expirou(c, agora)) continue;
    try {
      await chrome.action.setBadgeText({ tabId, text: "ON" });
      await chrome.action.setBadgeBackgroundColor({ tabId, color: "#171717" });
      await chrome.action.setTitle({ tabId, title: `HADES: aba conectada (${c.origem})` });
    } catch {
      /* a aba pode ter fechado; nao e' erro */
    }
  }
}

async function limparBadge(tabId) {
  try {
    await chrome.action.setBadgeText({ tabId, text: "" });
    await chrome.action.setTitle({ tabId, title: "HADES Browser Bridge" });
  } catch {
    /* aba fechada */
  }
}

/* --------------------------------------------------------------- concessao */

/**
 * Concede a aba ATIVA ao HADES, a pedido explicito do utilizador.
 * So' aceita origem http/https validada; guarda aba, janela, origem e prazo.
 *
 * Aceita `tabId` explicito (`E-1033`): sem isso, so' a aba activa podia ser concedida --
 * e uma superficie que escolhe a aba numa lista (ou o proprio E2E, onde o popup e' uma
 * aba e passa a ser ele a aba activa) nao tinha como conceder a aba certa. O caminho
 * por omissao continua a ser a aba activa.
 */
async function conectarAbaAtiva(tabIdPedido) {
  let aba;
  if (Number.isFinite(Number(tabIdPedido))) {
    try {
      aba = await chrome.tabs.get(Number(tabIdPedido));
    } catch {
      return { ok: false, motivo: "aba_inexistente" };
    }
  } else {
    [aba] = await chrome.tabs.query({ active: true, currentWindow: true });
  }
  if (!aba || typeof aba.id !== "number") {
    return { ok: false, motivo: "sem_aba_ativa" };
  }
  // incognito: recusa, mesmo que o manifest ja' o bloqueie
  if (aba.incognito) return { ok: false, motivo: "incognito_recusado" };

  // `E-1033`: o opt-in de rede privada tem de ser LIDO e passado, senao `permitirPrivado`
  // era sempre `false` e o `loopbackAutorizado` do protocolo nunca servia para nada.
  const v = validarUrl(aba.url ?? "", { permitirPrivado: await lerOptIn() });
  if (!v.ok) return { ok: false, motivo: v.motivo };

  const dono = new URL(v.url).origin;
  const estado = await lerEstado();
  estado.concessoes[chaveDaAba(aba.id)] = {
    tabId: aba.id,
    windowId: aba.windowId,
    origem: dono,
    concedidaEm: Date.now(),
    expiraEm: Date.now() + TTL_PADRAO_MS,
    scopes: [...SCOPES],
  };
  estado.accoes[chaveDaAba(aba.id)] = [];
  await gravarEstado(estado);
  await chrome.action.setBadgeText({ tabId: aba.id, text: "ON" });
  return { ok: true, tabId: aba.id, origem: dono, expiraEm: estado.concessoes[chaveDaAba(aba.id)].expiraEm };
}

/** Revoga uma aba (ou todas). Nao toca em nada do utilizador. */
async function desconectar(tabId) {
  const estado = await lerEstado();
  if (typeof tabId === "number") {
    delete estado.concessoes[chaveDaAba(tabId)];
    delete estado.accoes[chaveDaAba(tabId)];
    await gravarEstado(estado);
    await limparBadge(tabId);
    return { ok: true, revogadas: 1 };
  }
  const ids = Object.keys(estado.concessoes).map((k) => Number(k.slice(1)));
  estado.concessoes = {};
  estado.accoes = {};
  await gravarEstado(estado);
  for (const id of ids) await limparBadge(id);
  return { ok: true, revogadas: ids.length };
}

/* ------------------------------------------------------------------ accoes */

/** Backpressure simples: no maximo LIMITES.accoesPorMinuto por concessao. */
function dentroDoLimite(estado, chave) {
  const agora = Date.now();
  const marcas = (estado.accoes[chave] ?? []).filter((t) => agora - t < JANELA_MS);
  if (marcas.length >= LIMITES.accoesPorMinuto) {
    estado.accoes[chave] = marcas;
    return false;
  }
  marcas.push(agora);
  estado.accoes[chave] = marcas;
  return true;
}

/**
 * Ponto unico por onde passam as accoes vindas do popup/content script.
 * Nao executa nada por si: valida escopo, concessao e limite, e devolve o
 * pedido canonico para o HADES decidir (aprovacao incluida).
 */
async function tratarAccao(mensagem) {
  if (!mensagem || typeof mensagem !== "object") return { ok: false, motivo: "mensagem_invalida" };
  const { tipo } = mensagem;

  if (tipo === "conectar") return conectarAbaAtiva(mensagem.tabId);
  if (tipo === "desconectar") return desconectar(mensagem.tabId);
  if (tipo === "estado") {
    const estado = await lerEstado();
    return {
      ok: true,
      missao: estado.missao,
      host: estado.host,
      aprovado: estado.aprovado,
      concessoes: Object.values(estado.concessoes).filter((c) => !expirou(c)),
      escopos: [...SCOPES],
      ferramenta: TOOL,
    };
  }

  if (tipo !== "accao") return { ok: false, motivo: "tipo_desconhecido" };

  const tabId = Number(mensagem.tabId);
  if (!Number.isFinite(tabId)) return { ok: false, motivo: "aba_ausente" };
  const concessao = await concessaoViva(tabId);
  if (!concessao) return { ok: false, motivo: "aba_nao_concedida" };

  // a accao tem de estar dentro dos 3 scopes canonicos
  if (!SCOPES.includes(mensagem.scope)) return { ok: false, motivo: "scope_nao_canonico" };

  // e a origem do alvo tem de continuar a ser a concedida
  if (mensagem.url) {
    // `E-1035`: o `guardarRota` valida o URL com `validarUrl` -- e era o **TERCEIRO**
    // local a faze-lo **sem o opt-in**, depois de o `conectarAbaAtiva` e o `tabs.onUpdated`
    // ja' terem sido curados. *Curar dois de tres sitios deixa o terceiro a recusar o que
    // os outros aceitam* -- e o sintoma, medido, foi uma concessao viva a receber
    // `rede_privada_sem_optin`.
    const rota = guardarRota(mensagem.url, [concessao.origem], { permitirPrivado: await lerOptIn() });
    if (!rota.ok) return rota;
  }

  const estado = await lerEstado();
  if (!dentroDoLimite(estado, chaveDaAba(tabId))) {
    await gravarEstado(estado);
    return { ok: false, motivo: "limite_de_accoes_excedido" };
  }
  await gravarEstado(estado);

  // A extensao NAO decide aprovacao: declara a exigencia e devolve o pedido.
  return {
    ok: true,
    pedido: {
      tool: TOOL,
      scope: mensagem.scope,
      action: mensagem.action,
      params: mensagem.params ?? {},
      session_id: estado.missao,
      requires_approval: true,
      origem: concessao.origem,
    },
  };
}

/* ------------------------------------------------- o canal local (E-1032) */

/**
 * A permissao `nativeMessaging` esta' em `optional_permissions` -- de proposito: *o
 * `E-1025` decidiu que a extensao NAO deve ter poder sobre o PC desde a instalacao.*
 *
 * MAS ISSO TEM UMA CONSEQUENCIA QUE FALTAVA: enquanto a permissao nao for CONCEDIDA,
 * `chrome.runtime.sendNativeMessage` **nao existe no objecto**. Nao devolve erro de
 * permissao -- nao e' uma funcao. E nenhum ficheiro desta extensao a pedia.
 *
 * Este e' o unico sitio onde o canal local e' ligado, e ele exige o gesto do utilizador
 * (`chrome.permissions.request` so' funciona em contexto de clique).
 */
async function estadoDoCanal() {
  const tem = await chrome.permissions.contains({ permissions: ["nativeMessaging"] });
  return { ok: true, permissaoConcedida: tem, host: NOME_DO_HOST };
}

async function ligarCanal() {
  const tem = await chrome.permissions.contains({ permissions: ["nativeMessaging"] });
  if (!tem) return { ok: false, motivo: "permissao_nao_concedida" };
  try {
    // Pergunta minima: so' o necessario. Nada de scopes do SO.
    const resposta = await chrome.runtime.sendNativeMessage(NOME_DO_HOST, { id: 1, cmd: "host:ola" });
    const estado = await lerEstado();
    if (resposta?.ok === true) {
      estado.host = { protocolo: resposta.protocolo, instalado: resposta.instalado === true, em: Date.now() };
      await gravarEstado(estado);
    }
    return { ok: resposta?.ok === true, resposta };
  } catch (e) {
    return { ok: false, motivo: "host_nao_responde", detalhe: String(e?.message ?? e) };
  }
}

/**
 * ATENCAO -- **isto NAO serve para PEDIR a permissao.** Medido (`E-1032`): o Chrome
 * recusa com «This function must be called during a user gesture» porque o
 * `sendMessage` e' assincrono e o gesto ja' expirou quando o pedido chega aqui.
 *
 * O pedido tem de viver no POPUP, no handler do clique. Esta funcao existe apenas para
 * documentar o beco sem saida -- o `popup.js` **nao** a chama.
 */
async function pedirCanal() {
  return {
    ok: false,
    motivo: "gesto_expirado",
    dica: "peça a permissão no popup: o gesto não sobrevive ao sendMessage assíncrono",
  };
}

async function desligarCanal() {
  const estado = await lerEstado();
  estado.host = null;
  await gravarEstado(estado);
  return { ok: true };
}

/** Diagnostico honesto: o que esta' ligado e o que nao esta'. */
async function diagnosticoDoCanal() {
  const base = await estadoDoCanal();
  const estado = await lerEstado();
  return {
    ...base,
    ligado: estado.host !== null,
    host: estado.host,
    nomeDoHost: NOME_DO_HOST,
    extensaoId: chrome.runtime.id,
  };
}

/* ------------------------------------------- takeover humano (E-1034) */

/**
 * O scope `browser:takeover` estava nos 3 canonicos e **nao tinha implementacao**. E o
 * contrato exige: *enquanto o utilizador estiver a interagir, o agente nao disputa
 * mouse/teclado nem sobrescreve campos.*
 *
 * O content script observa os gestos humanos e avisa; aqui registamos QUEM tem o
 * controlo, com autor (`agent`/`human`/`system`), e permitimos que o utilizador devolva
 * a sessao. Nunca guardamos o que foi escrito -- so' o instante e o tipo de gesto.
 */
async function registarTakeover(tabId, motivo) {
  const estado = await lerEstado();
  estado.takeover = estado.takeover ?? {};
  const chave = chaveDaAba(tabId);
  if (!estado.takeover[chave]) {
    estado.takeover[chave] = {
      tabId, autor: "human", desde: Date.now(), motivo, eventos: 0,
    };
  }
  estado.takeover[chave].eventos += 1;
  estado.takeover[chave].desde = Date.now();
  await gravarEstado(estado);
  return { ok: true, autor: "human" };
}

async function estadoDoTakeover(tabId) {
  const estado = await lerEstado();
  const t = (estado.takeover ?? {})[chaveDaAba(tabId)] ?? null;
  return { ok: true, ativo: t !== null, takeover: t, autor: t ? "human" : "agent" };
}

async function devolverSessao(tabId) {
  const estado = await lerEstado();
  const chave = chaveDaAba(tabId);
  const tinha = (estado.takeover ?? {})[chave] ?? null;
  if (estado.takeover) delete estado.takeover[chave];
  await gravarEstado(estado);
  return { ok: true, devolvida: tinha !== null, autor: "agent" };
}

/* -------------------------------------------------- mensagens e ciclo de vida */

chrome.runtime.onMessage.addListener((mensagem, remetente, responder) => {
  // Uma pagina NUNCA fala com o service worker: so' o popup e o content script
  // da propria extensao. Uma mensagem de origem externa e' recusada de imediato.
  if (remetente?.id !== chrome.runtime.id) {
    responder({ ok: false, motivo: "remetente_nao_autorizado" });
    return false;
  }
  const t = mensagem?.tipo;
  // O canal local tem os seus proprios handlers: `pedir` PRECISA do gesto do
  // utilizador, logo nao pode passar pelo funil generico sem se perder a origem.
  const doCanal = t === "canal:estado" ? estadoDoCanal()
    : t === "canal:ligar" ? ligarCanal()
    : t === "canal:desligar" ? desligarCanal()
    : t === "canal:diagnostico" ? diagnosticoDoCanal()
    : t === "optin:estado" ? lerOptIn().then((ligado) => ({ ok: true, ligado }))
    : t === "optin:definir" ? definirOptIn(mensagem.ligado === true)
    // E-1034: o takeover humano. O content script avisa (sem `tabId`: ele nao o conhece --
    // vem do `remetente`); o popup consulta e devolve.
    : t === "takeover:humano"
      ? registarTakeover(Number(remetente?.tab?.id ?? mensagem.tabId), String(mensagem.motivo ?? "interaccao"))
    : t === "takeover:estado" ? estadoDoTakeover(Number(mensagem.tabId))
    : t === "takeover:devolver" ? devolverSessao(Number(mensagem.tabId))
    : null;
  if (doCanal) {
    doCanal.then(responder).catch((erro) =>
      responder({ ok: false, motivo: "erro_interno", detalhe: String(erro?.message ?? erro) }));
    return true;
  }
  tratarAccao(mensagem)
    .then(responder)
    .catch((erro) => responder({ ok: false, motivo: "erro_interno", detalhe: String(erro?.message ?? erro) }));
  return true; // resposta assincrona
});

// fechar a aba revoga a concessao
chrome.tabs.onRemoved.addListener(async (tabId) => {
  const estado = await lerEstado();
  if (estado.concessoes[chaveDaAba(tabId)]) {
    delete estado.concessoes[chaveDaAba(tabId)];
    delete estado.accoes[chaveDaAba(tabId)];
    await gravarEstado(estado);
  }
});

// navegar para outra origem dentro da mesma aba exige nova concessao
chrome.tabs.onUpdated.addListener(async (tabId, info) => {
  if (!info.url && !info.status) return;
  const estado = await lerEstado();
  const c = estado.concessoes[chaveDaAba(tabId)];
  if (!c) return;
  if (expirou(c)) {
    delete estado.concessoes[chaveDaAba(tabId)];
    await gravarEstado(estado);
    await limparBadge(tabId);
    return;
  }
  const v = validarUrl(info.url ?? "", { permitirPrivado: await lerOptIn() });
  if (!v.ok || new URL(v.url).origin !== c.origem) {
    // nova origem: revoga. O utilizador reconecta se quiser.
    delete estado.concessoes[chaveDaAba(tabId)];
    delete estado.accoes[chaveDaAba(tabId)];
    await gravarEstado(estado);
    await limparBadge(tabId);
  }
});

// limpeza na instalacao/arranque: nada de estado orfao de uma versao anterior
chrome.runtime.onInstalled.addListener(async () => {
  const estado = await lerEstado();
  const agora = Date.now();
  for (const [k, c] of Object.entries(estado.concessoes)) {
    if (expirou(c, agora)) {
      delete estado.concessoes[k];
      delete estado.accoes[k];
    }
  }
  await gravarEstado(estado);
});
