// HADES Browser Bridge -- content script.
//
// Este script corre DENTRO da pagina. Por isso parte do principio de que a
// pagina e' HOSTIL: conteudo da pagina, DOM e accessibility tree sao DADOS NAO
// CONFIAVEIS, nunca instrucoes.
//
// O que este script faz:
//   * responde a pedidos de OBSERVACAO com um DOM saneado;
//   * nunca le nem devolve campos sensiveis (senha, cartao, MFA);
//   * recusa tudo se a aba nao estiver concedida -- e a decisao e' do service
//     worker, nao deste script;
//   * nao tem acesso a segredo nenhum do HADES nem do Companion.

(() => {
  "use strict";

  // Se o service worker ja' injetou este script, nao duplicar listeners.
  if (window.__HADES_BRIDGE_ATIVO__) return;
  window.__HADES_BRIDGE_ATIVO__ = true;

  const CAMPOS_SENSIVEIS = [
    "password", "passwd", "pwd", "senha", "credit-card", "cc-number", "cvc", "cvv",
    "one-time-code", "otp", "mfa", "totp", "iban", "account-number", "ssn",
  ];

  /** Mesma regra do protocolo: tipo, autocomplete E nome, os tres. */
  function campoSensivel(el) {
    if (!el) return false;
    const tipo = String(el.getAttribute?.("type") ?? "").toLowerCase();
    if (tipo === "password") return true;
    const auto = String(el.getAttribute?.("autocomplete") ?? "").toLowerCase();
    if (auto.includes("cc-") || auto.includes("one-time-code")
      || auto.includes("current-password") || auto.includes("new-password")) return true;
    const nome = `${el.getAttribute?.("name") ?? ""} ${el.id ?? ""} ${el.getAttribute?.("aria-label") ?? ""}`.toLowerCase();
    return CAMPOS_SENSIVEIS.some((s) => nome.includes(s));
  }

  /** Texto util e curto, sem lixo de script/style. */
  function limpar(texto, limite = 200) {
    const t = String(texto ?? "").replace(/\s+/g, " ").trim();
    return t.length > limite ? t.slice(0, limite) : t;
  }

  /**
   * DOM saneado + acessibilidade. Nunca inclui o VALOR de um campo sensivel --
   * nem o valor de nenhum input, so' o rotulo e o estado.
   */
  function observar(limiteNos = 400) {
    const nos = [];
    const todos = document.querySelectorAll(
      "a[href], button, input, select, textarea, [role], h1, h2, h3, label",
    );
    for (const el of todos) {
      if (nos.length >= limiteNos) break;
      // nada de invisiveis, nada de dentro de <script>/<style>
      if (!el.offsetParent && el.tagName !== "BODY" && getComputedStyle(el).position !== "fixed") continue;

      const sensivel = campoSensivel(el);
      const entrada = {
        tag: el.tagName.toLowerCase(),
        role: el.getAttribute?.("role") ?? null,
        rotulo: limpar(el.getAttribute?.("aria-label") ?? el.textContent ?? "", 120),
        // O VALOR nunca sai. Para campos sensiveis, nem o tipo exato sai.
        tipo: sensivel ? "campo_sensivel" : (el.getAttribute?.("type") ?? null),
        desabilitado: el.disabled === true,
        sensivel,
      };
      if (el.tagName === "A" && el.href) {
        try {
          // so' a origem + caminho: nunca query nem fragmento (podem ter tokens)
          const u = new URL(el.href);
          if (u.protocol === "http:" || u.protocol === "https:") {
            entrada.href = u.origin + u.pathname;
          }
        } catch { /* href invalido: ignora */ }
      }
      nos.push(entrada);
    }
    return {
      url: location.origin + location.pathname,
      titulo: limpar(document.title, 160),
      nos,
      total: todos.length,
    };
  }

  /** Acao de escrita: recusa campos sensiveis -- o utilizador preenche-os. */
  function preencher(seletor, valor) {
    let el;
    try {
      el = document.querySelector(seletor);
    } catch {
      return { ok: false, motivo: "seletor_invalido" };
    }
    if (!el) return { ok: false, motivo: "elemento_nao_encontrado" };
    if (campoSensivel(el)) {
      // takeover humano: o agente nao escreve credenciais
      return { ok: false, motivo: "campo_sensivel_takeover_humano" };
    }
    const tipo = String(el.getAttribute?.("type") ?? "").toLowerCase();
    if (tipo === "file") return { ok: false, motivo: "upload_exige_aprovacao_e_caminho_autorizado" };
    el.focus();
    el.value = String(valor ?? "");
    el.dispatchEvent(new Event("input", { bubbles: true }));
    el.dispatchEvent(new Event("change", { bubbles: true }));
    return { ok: true };
  }

  /** Clique: so' em elemento visivel e nao sensivel. */
  function clicar(seletor) {
    let el;
    try {
      el = document.querySelector(seletor);
    } catch {
      return { ok: false, motivo: "seletor_invalido" };
    }
    if (!el) return { ok: false, motivo: "elemento_nao_encontrado" };
    if (!el.offsetParent && getComputedStyle(el).position !== "fixed") {
      return { ok: false, motivo: "elemento_invisivel" };
    }
    el.click();
    return { ok: true };
  }

  chrome.runtime.onMessage.addListener((mensagem, remetente, responder) => {
    // So' a propria extensao fala com este script.
    if (remetente?.id !== chrome.runtime.id) {
      responder({ ok: false, motivo: "remetente_nao_autorizado" });
      return false;
    }
    if (!mensagem || typeof mensagem !== "object") {
      responder({ ok: false, motivo: "mensagem_invalida" });
      return false;
    }

    switch (mensagem.tipo) {
      case "observar":
        responder({ ok: true, observacao: observar() });
        return false;
      case "preencher":
        responder(preencher(mensagem.seletor, mensagem.valor));
        return false;
      case "clicar":
        responder(clicar(mensagem.seletor));
        return false;
      case "origem":
        responder({ ok: true, origem: location.origin });
        return false;
      default:
        responder({ ok: false, motivo: "tipo_desconhecido" });
        return false;
    }
  });
})();
