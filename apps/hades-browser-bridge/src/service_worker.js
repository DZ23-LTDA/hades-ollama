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
// `E-1040`: o scope `browser:files` estava nos 3 canonicos e nao tinha implementacao. Aqui
// entra a DECISAO sobre ficheiros -- *nao um motor de download*.
import {
  PROTOCOLO_ARQUIVOS, decidir, sha256De, resumoDaDecisao, fingerprintDe,
  AUTORIDADE_DA_EXTENSAO, avisoDeAutoridade,
} from "./arquivos.js";

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

/**
 * LE' o estado, e **NAO PERDE O QUE NAO CONHECE** (`E-1040`).
 *
 * A versao anterior reconstruia o objecto com uma **lista fixa** de campos:
 *
 *     return { concessoes: ..., accoes: ..., missao: ..., host: ..., aprovado: ..., takeover: ... };
 *
 * => *todo o campo que nao estivesse na lista era APAGADO a cada leitura.* O `takeover` foi
 * perdido assim (`E-1037`); e quando fui acrescentar o `arquivos`, ia ser perdido da mesma
 * forma -- **o mesmo defeito, pela terceira vez.**
 *
 * *Curar o campo que falhou nao cura a lista.* Agora a lista e' de campos **com tratamento
 * proprio** (validados um a um) e **tudo o resto passa intacto** -- sem validacao, mas sem
 * perda. *Um campo desconhecido e' um campo que ainda nao tem regra, nao um campo a apagar.*
 */
const CAMPOS_TIPADOS = Object.freeze({
  concessoes: (v) => (v && typeof v === "object" ? v : {}),
  accoes: (v) => (v && typeof v === "object" ? v : {}),
  missao: (v) => v ?? null,
  host: (v) => v ?? null,
  aprovado: (v) => v === true,
  takeover: (v) => (v && typeof v === "object" ? v : {}),
});

async function lerEstado() {
  const bruto = await chrome.storage.session.get(CHAVE);
  const estado = bruto?.[CHAVE];
  if (!estado || typeof estado !== "object") {
    return {
      concessoes: {}, accoes: {}, missao: null, host: null, aprovado: false, takeover: {},
      arquivos: [],
    };
  }
  // 1. o que tem regra, normalizado
  const saida = {};
  for (const [campo, normalizar] of Object.entries(CAMPOS_TIPADOS)) {
    saida[campo] = normalizar(estado[campo]);
  }
  // 2. e TUDO O RESTO, preservado -- *um campo desconhecido e' um campo sem regra ainda.*
  for (const [campo, valor] of Object.entries(estado)) {
    if (campo in CAMPOS_TIPADOS) continue;
    saida[campo] = valor;
  }
  return saida;
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
    await marcarBadge(tabId, c.origem);
  }
}

/**
 * O INDICADOR DE ABA CONCEDIDA, num unico sitio (`E-1037`).
 *
 * *Antes estava inline em dois locais* -- e por isso a reposicao depois de um reload nao
 * tinha onde se apoiar. Agora ha' **uma** funcao que diz "esta aba esta' concedida", e os
 * dois locais usam-na.
 *
 * O contrato exige *badge/indicador persistente e inequivoco enquanto uma aba estiver sendo
 * observada ou controlada*. Medido: o Chrome **limpa o badge da aba ao navegar** -- sem
 * reposicao, uma concessao viva ficava **invisivel**, e *um indicador que nao reflecte o
 * estado e' pior do que indicador nenhum: da' confianca a quem nao a devia ter.*
 */
async function marcarBadge(tabId, origem) {
  try {
    await chrome.action.setBadgeText({ tabId, text: "ON" });
    await chrome.action.setBadgeBackgroundColor({ tabId, color: "#171717" });
    await chrome.action.setTitle({
      tabId,
      title: origem ? `HADES: aba conectada (${origem})` : "HADES: aba conectada",
    });
  } catch {
    /* a aba pode ter fechado; nao e' erro */
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
  await marcarBadge(aba.id, dono);
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

/* ------------------------------------- ficheiros: a decisao (`E-1040`) */

/**
 * AVALIA um pedido de transferencia e devolve a decisao. **Nao transfere nada.**
 *
 * O que NAO faz, de proposito:
 *   * nao descarrega -- a extensao nao e' um cliente de download com escrita no disco;
 *   * nao le' o ficheiro do disco do utilizador -- o content script nao tem esse poder;
 *   * nao aprova nada sozinha -- a aprovacao vem do utilizador (`aprovado`), e um
 *     executavel **continua** a exigir aprovacao mesmo depois dela.
 *
 * O `sha256` tem de vir MEDIDO (`arquivo:medir`) ou de quem ja' mediu. *Quem pede nao
 * certifica o que pede* -- e por isso o campo declarado e' cruzado com o tamanho.
 */
async function avaliarArquivo(mensagem) {
  const pedido = mensagem?.pedido ?? {};
  // `E-1041`: a PROVA, nao um booleano. *Um campo que quem pede controla nao e' uma
  // aprovacao -- e' uma afirmacao.* O `fingerprint` da prova tem de casar com o do pedido.
  //
  // O `aprovado` continua a ser LIDO -- mas so' para ser RECUSADO com o nome certo. *Ignorar
  // um campo que quem pede envia deixa quem o envia sem saber porque nao funcionou.*
  const decisao = decidir(pedido, { prova: mensagem?.prova, aprovado: mensagem?.aprovado });
  // O resumo vai para o audit log; o hash NAO -- um hash e' um identificador, e identificadores
  // nao pertencem a um log que se mostra.
  const registo = {
    em: Date.now(),
    decisao: decisao.veredicto,
    motivo: decisao.motivo,
    risco: decisao.risco ?? null,
    direcao: decisao.direcao ?? null,
    extensao: decisao.extensao ?? null,
    tamanho: decisao.tamanho ?? null,
    resumo: resumoDaDecisao(decisao),
    // A autoria da aprovacao, quando houve uma. *Quem autorizou, e nao o quao convincente
    // parecia o pedido.* O fingerprint NAO entra: e' um identificador.
    aprovacaoId: decisao.aprovacaoId ?? null,
    aprovadoPor: decisao.aprovadoPor ?? null,
    // `E-1042`: a decisao DIZ o que vale. *Um limite que nao se le' nao e' um limite.*
    autoridade: decisao.autoridade ?? "forma_apenas",
  };
  const estado = await lerEstado();
  estado.arquivos = Array.isArray(estado.arquivos) ? estado.arquivos : [];
  estado.arquivos.push(registo);
  if (estado.arquivos.length > 200) estado.arquivos = estado.arquivos.slice(-200);
  await gravarEstado(estado);
  // *Um campo `ok` num caminho de PERMISSAO tem de significar "podes transferir agora".*
  // Medido no E2E: `ok: true` para um `.exe` **sem** aprovacao -- porque `ok` estava a dizer
  // "a avaliacao correu", e nao "esta autorizado". *A ambiguidade num campo de permissao e'
  // um defeito: quem le' `ok` age.*
  //
  //   permitido                       -> sim
  //   exige_aprovacao + aprovado      -> sim (a decisao registou que a aprovacao veio)
  //   exige_aprovacao sem aprovacao   -> nao
  //   proibido                        -> nao
  const podesTransferir = decisao.veredicto === "permitido" || decisao.veredicto === "autorizado";
  return {
    // a avaliacao correu -- nada rebentou
    ok: true,
    // e a resposta que interessa a quem vai transferir
    podesTransferir,
    // e o que essa autorizacao VALE -- sempre, nao so' no caso autorizado
    autoridade: decisao.autoridade ?? AUTORIDADE_DA_EXTENSAO,
    aviso: avisoDeAutoridade(decisao.autoridade ?? AUTORIDADE_DA_EXTENSAO),
    protocolo: PROTOCOLO_ARQUIVOS,
    decisao,
    registo,
  };
}

/**
 * MEDE os bytes de um ficheiro que o POPUP escolheu (`<input type=file>`).
 *
 * *O content script nao pode ler ficheiros; o popup pode, com um gesto.* Assim a medicao
 * acontece onde o utilizador a autorizou, e o que sai dali e' **o hash e o tamanho** -- nunca
 * o conteudo.
 */
async function medirBytes(mensagem) {
  const bytes = mensagem?.bytes;
  const h = await sha256De(bytes);
  if (!h.ok) return { ok: false, motivo: h.motivo };
  return {
    ok: true, protocolo: PROTOCOLO_ARQUIVOS,
    nome: String(mensagem?.nome ?? ""), tamanho: h.bytes, sha256: h.sha256,
  };
}

/* --------------------------------- consulta de aprovacao (`E-1043`) */

/**
 * CONSULTA O COMPANION PELA APROVACAO.
 *
 * O caminho existe ponta a ponta: extensao -> native messaging -> Companion. **E o Companion
 * responde que NAO verificou** -- porque a rota do runtime e' de sessao e ele nao tem sessao.
 *
 * *O que isto acrescenta nao e' a verificacao: e' a PERGUNTA.* Antes desta entrada, a
 * extensao decidia sozinha e nao havia a quem perguntar. Agora pergunta ao sitio certo,
 * recebe uma resposta honesta, e **nao deixa a decisao depender dela**:
 *
 *   * se o Companion disser `verificada: true`, a autoridade **sobe** para `verificada`;
 *   * se disser `false`, fica `forma_apenas` -- **e o motivo viaja**;
 *   * se o canal nem existir (sem `nativeMessaging`), fica `forma_apenas` com
 *     `motivo: "canal_indisponivel"`.
 *
 * *Nunca o contrario: um canal em baixo nao pode fazer uma decisao parecer melhor.*
 */
async function consultarAutoridade(fingerprint) {
  const base = {
    ok: true, fingerprint,
    verificada: false, autoridade: AUTORIDADE_DA_EXTENSAO,
    motivo: "canal_indisponivel", desbloqueio: null,
  };
  if (typeof chrome?.runtime?.sendNativeMessage !== "function") return base;
  try {
    const estado = await lerEstado();
    const r = await chrome.runtime.sendNativeMessage(NOME_DO_HOST, {
      cmd: "aprovacao:consultar", fingerprint,
      // A instalacao sobe quem pergunta. Sem par emparelhado, o Companion recusa.
      instalacao: estado.instalacao ?? null,
    });
    if (!r || r.ok !== true) {
      return { ...base, motivo: r?.motivo ?? "resposta_invalida" };
    }
    // A unica coisa que pode subir a autoridade: o Companion dizer que verificou.
    const verificada = r.verificada === true;
    return {
      ok: true,
      fingerprint: r.fingerprint ?? fingerprint,
      verificada,
      autoridade: verificada ? "verificada" : AUTORIDADE_DA_EXTENSAO,
      motivo: r.motivo ?? null,
      desbloqueio: r.desbloqueio ?? null,
    };
  } catch (e) {
    // *Um canal em baixo nao pode fazer uma decisao parecer melhor.*
    return { ...base, motivo: "erro_de_canal", detalhe: String(e?.message ?? e).slice(0, 120) };
  }
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
    // E-1040: o scope `browser:files`. A extensao NAO descarrega nem carrega nada por conta
    // propria: avalia um manifesto e devolve a decisao. *O que faltava nao era um motor de
    // download -- era uma decisao auditavel sobre ficheiros.*
    : t === "arquivo:avaliar" ? avaliarArquivo(mensagem)
    : t === "arquivo:medir" ? medirBytes(mensagem)
    // `E-1043`: pergunta ao Companion se ele consegue VERIFICAR a aprovacao. O caminho
    // existe; a resposta honesta e' que nao pode. *A extensao pergunta e nao depende.*
    : t === "arquivo:consultarAutoridade" ? consultarAutoridade(String(mensagem.fingerprint ?? ""))
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

  // `E-1037`: SE NAO HA' URL, NAO HA' MUDANCA DE ORIGEM A AVALIAR.
  //
  // Medido: o browser emite `onUpdated` com `changeInfo` que traz **so' `status`**
  // (`{status:"loading"}` / `{status:"complete"}`) e **sem `url`** -- a cada `pushState`,
  // `replaceState`, **reload** e ate' a cada mudanca de titulo. O codigo antigo fazia
  // `validarUrl(info.url ?? "", ...)`: validava a **string vazia**, que falha, e concluia
  // "nova origem" => **revogava a concessao**.
  //
  // *A navegacao mais comum que existe -- recarregar a pagina -- matava a sessao.*
  // O `E-1035` nao apanhou isto porque la' a mudanca de origem era **real** e trazia URL.
  //
  // Se o URL esta' AUSENTE, nao ha' nada a decidir: mantem-se a concessao. Decidir sobre
  // um vazio inventado e' o defeito; nao decidir e' a cura.
  if (!info.url) {
    // ...MAS O INDICADOR TEM DE CONTINUAR A DIZER A VERDADE. Medido: o Chrome **limpa o
    // badge da aba** ao navegar, e a concessao sobrevivia com o badge VAZIO -- o utilizador
    // deixava de ver que a aba estava concedida. *Um indicador que nao reflecte o estado
    // e' pior do que indicador nenhum: da' confianca a quem nao a devia ter.*
    // Repor aqui, quando a pagina termina de carregar, custa nada e mantem o badge honesto.
    if (info.status === "complete") await marcarBadge(tabId, c.origem);
    return;
  }

  const v = validarUrl(info.url, { permitirPrivado: await lerOptIn() });
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
