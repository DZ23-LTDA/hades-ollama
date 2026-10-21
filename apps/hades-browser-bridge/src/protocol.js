// HADES Browser Bridge -- protocolo de accoes e observacoes.
//
// Este modulo REUTILIZA o contrato que ja' existe no runtime do HADES, medido em
// internal/agent/browser.go e internal/agent/browser_helper.py:
//
//   browser.operator  Scopes: browser:navigate, browser:files, browser:takeover
//                     Risk: RiskExternalSideEffect, RequiresApproval: true
//                     request["session_id"] = toolContext.MissionID
//   browser_helper.py validate_url, validate_cdp_url, guard_browser_route
//                     allow_loopback = OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE == "1"
//
// Nao cria protocolo paralelo, nao cria aprovacao paralela e nao tem eval.

/** Os 3 scopes canonicos, copiados de internal/agent/browser.go. */
export const SCOPES = Object.freeze([
  "browser:navigate",
  "browser:files",
  "browser:takeover",
]);

/** A convencao do HADES e' "<ferramenta>:<capacidade>" (ver desktop.companion). */
export const TOOL = "browser.operator";

/** Esquemas que a extensao aceita para NAVEGAR. O resto e' recusado antes de qualquer accao. */
export const SCHEMES_PERMITIDOS = Object.freeze(["http:", "https:"]);

/**
 * Esquemas aceites apenas no canal CDP, e apenas em loopback.
 * O protocolo DevTools usa WebSocket (ws/wss) sobre http/https, por isso estes
 * dois esquemas sao legitimos AQUI e continuam proibidos em validarUrl.
 */
export const SCHEMES_CDP = Object.freeze(["http:", "https:", "ws:", "wss:"]);

/** Limites duros. Uma mensagem maior do que isto e' recusada, nao truncada. */
export const LIMITES = Object.freeze({
  url: 2048,
  texto: 64 * 1024,
  seletor: 512,
  snapshotBytes: 1 << 20,
  accoesPorMinuto: 120,
});

/** Prefixos que nunca podem ser alvo, mesmo com loopback autorizado. */
const PREFIXOS_PROIBIDOS = Object.freeze([
  "127.", "10.", "192.168.", "169.254.", "0.", "::1", "fe80:", "fc", "fd",
]);

/**
 * O opt-in de loopback, com o MESMO nome da variavel do helper Python.
 * Sem opt-in explicito, rede privada e' recusada.
 */
export function loopbackAutorizado(armazenamento) {
  try {
    return armazenamento !== null && armazenamento !== undefined
      && armazenamento.OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE === "1";
  } catch {
    return false;
  }
}

function hostPrivado(host) {
  const h = String(host || "").toLowerCase();
  if (h === "localhost" || h.endsWith(".localhost")) return true;
  return PREFIXOS_PROIBIDOS.some((p) => h.startsWith(p));
}

/**
 * Espelha validate_url do browser_helper.py: devolve {ok, motivo, url}.
 * Nao lanca: um URL mau e' um resultado, nao uma excepcao que alguem possa ignorar.
 */
export function validarUrl(bruto, opcoes = {}) {
  const permitirPrivado = opcoes.permitirPrivado === true;
  if (typeof bruto !== "string" || bruto.trim() === "") {
    return { ok: false, motivo: "url_ausente" };
  }
  if (bruto.length > LIMITES.url) {
    return { ok: false, motivo: "url_demasiado_longa" };
  }
  let u;
  try {
    u = new URL(bruto);
  } catch {
    return { ok: false, motivo: "url_invalido" };
  }
  if (!SCHEMES_PERMITIDOS.includes(u.protocol)) {
    return { ok: false, motivo: "esquema_nao_permitido" };
  }
  if (u.username !== "" || u.password !== "") {
    return { ok: false, motivo: "credenciais_no_url" };
  }
  if (!permitirPrivado && hostPrivado(u.hostname)) {
    return { ok: false, motivo: "rede_privada_sem_optin" };
  }
  return { ok: true, url: u.toString() };
}

/** Espelha validate_cdp_url: o CDP so' pode ser loopback, nunca LAN nem publico. */
export function validarCdpUrl(bruto) {
  // O CDP usa WebSocket (ws/wss) alem de http/https: validamos o esquema aqui,
  // porque validarUrl (para navegacao) so' aceita http/https de proposito.
  if (typeof bruto !== "string" || bruto.trim() === "") {
    return { ok: false, motivo: "url_ausente" };
  }
  if (bruto.length > LIMITES.url) {
    return { ok: false, motivo: "url_demasiado_longa" };
  }
  let u;
  try {
    u = new URL(bruto);
  } catch {
    return { ok: false, motivo: "url_invalido" };
  }
  if (!SCHEMES_CDP.includes(u.protocol)) {
    return { ok: false, motivo: "esquema_nao_permitido" };
  }
  if (u.username !== "" || u.password !== "") {
    return { ok: false, motivo: "credenciais_no_url" };
  }
  // Guardar a mesma regra de validarUrl: navegacao do CDP tambem nao vai a rede privada
  // sem opt-in -- aqui o opt-in e' o proprio CDP, que ja' exige loopback abaixo.
  const h = u.hostname.toLowerCase();
  const loopback = h === "localhost" || h === "127.0.0.1" || h === "::1";
  if (!loopback) return { ok: false, motivo: "cdp_fora_de_loopback" };
  return { ok: true, url: u.toString() };
}

/**
 * Espelha guard_browser_route: decide se uma accao pode tocar neste alvo.
 * `concedidos` e' o conjunto de origens que o UTILIZADOR autorizou para esta aba.
 */
export function guardarRota(alvo, concedidos, opcoes = {}) {
  const v = validarUrl(alvo, { permitirPrivado: opcoes.permitirPrivado === true });
  if (!v.ok) return v;
  const origem = new URL(v.url).origin;
  const lista = Array.isArray(concedidos) ? concedidos : [];
  if (!lista.includes(origem)) {
    return { ok: false, motivo: "origem_nao_concedida", origem };
  }
  return { ok: true, url: v.url, origem };
}

/**
 * O pedido canonico levado ao runtime. `session_id` e' o ID da MISSAO -- igual
 * ao que internal/agent/browser.go faz com toolContext.MissionID.
 */
export function pedidoCanonico(missaoId, acao, parametros = {}) {
  if (typeof missaoId !== "string" || missaoId.trim() === "") {
    return { ok: false, motivo: "missao_ausente" };
  }
  if (!SCOPES.includes(acao.scope)) {
    return { ok: false, motivo: "scope_nao_canonico" };
  }
  if (typeof acao.nome !== "string" || acao.nome.trim() === "") {
    return { ok: false, motivo: "accao_ausente" };
  }
  return {
    ok: true,
    pedido: {
      tool: TOOL,
      session_id: missaoId,
      scope: acao.scope,
      action: acao.nome,
      params: parametros,
      // A aprovacao NAO e' decidida aqui: este campo so' declara a exigencia.
      requires_approval: true,
    },
  };
}

/** Campos que nunca sao lidos nem devolvidos, seja qual for o scope concedido. */
export const CAMPOS_SENSIVEIS = Object.freeze([
  "password", "passwd", "pwd", "senha", "credit-card", "cc-number", "cvc", "cvv",
  "one-time-code", "otp", "mfa", "totp", "iban", "account-number", "ssn",
]);

/** Um campo e' sensivel pelo tipo, pelo autocomplete ou pelo nome -- todos os tres. */
export function campoSensivel(descricao) {
  const d = descricao || {};
  const tipo = String(d.type || "").toLowerCase();
  if (tipo === "password") return true;
  const auto = String(d.autocomplete || "").toLowerCase();
  if (auto.includes("cc-") || auto.includes("one-time-code") || auto.includes("current-password")
    || auto.includes("new-password")) return true;
  const nome = `${d.name || ""} ${d.id || ""} ${d.label || ""}`.toLowerCase();
  return CAMPOS_SENSIVEIS.some((s) => nome.includes(s));
}
