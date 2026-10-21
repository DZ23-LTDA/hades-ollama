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

  /* -------------------------------------------------- TAKEOVER HUMANO (E-1034) */
  //
  // O scope `browser:takeover` estava nos 3 canonicos e **nao tinha uma linha de
  // implementacao**. Pior: o content script nao observava a interaccao do utilizador,
  // logo o agente podia escrever por cima de quem estava a digitar -- exactamente o que
  // o contrato proibe.
  //
  // Aqui a pagina observa-se a si propria: qualquer gesto HUMANO (input, keydown,
  // pointerdown) marca takeover. Enquanto durar, as accoes de ESCRITA sao recusadas.
  // Nunca lemos o valor: so' o instante e o tipo de gesto.

  /** Quanto tempo o takeover humano dura depois do ultimo gesto. */
  const TAKEOVER_MS = 30 * 1000;

  const takeover = {
    ativo: false,
    desde: 0,
    autor: "system",
    motivo: "sem_interaccao",
  };

  /**
   * *A ESCRITA DO AGENTE NAO E' GESTO HUMANO.*
   *
   * Medido no `E-1034`: o `preencher` do agente faz `el.value = ...` e despacha `input` --
   * e o listener de takeover contava esse `input` como interaccao HUMANA. Resultado: o
   * agente escrevia uma vez e **perdia o controlo por sua propria accao**, e a accao
   * seguinte era recusada. *A automacao accionava o travao que existe para a travar.*
   *
   * Esta flag marca a janela em que quem escreve e' o agente. O listener respeita-a.
   */
  let escrevendoPeloAgente = false;

  function marcarTakeover(tipoDeGesto) {
    if (escrevendoPeloAgente) return; // e' a nossa propria escrita
    const agora = Date.now();
    const jaEstava = takeover.ativo && (agora - takeover.desde) < TAKEOVER_MS;
    takeover.ativo = true;
    takeover.desde = agora;
    takeover.autor = "human";
    takeover.motivo = tipoDeGesto;
    // Avisar o service worker UMA vez por episodio -- nao a cada tecla. *Uma ponte que
    // grita a cada frame e' pior do que uma que se cala.*
    if (!jaEstava) {
      try {
        chrome.runtime.sendMessage({ tipo: "takeover:humano", motivo: tipoDeGesto }).catch(() => {});
      } catch { /* o service worker pode estar a dormir; nao e' erro */ }
    }
  }

  function takeoverVivo() {
    return takeover.ativo && (Date.now() - takeover.desde) < TAKEOVER_MS;
  }

  // `capture: true` para apanhar ANTES de a pagina poder parar a propagacao.
  for (const evento of ["input", "keydown", "pointerdown"]) {
    window.addEventListener(evento, () => marcarTakeover(evento), { capture: true, passive: true });
  }

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
   *
   * `E-1039`: ATRAVESSA shadow roots ABERTOS e iframes de MESMA ORIGEM, e **DECLARA** os
   * limites. Medido antes: o `observar` devolvia `{url, titulo, nos, total}` e o `total` era
   * o total do **documento principal** -- parecia o total da pagina.
   *
   * *Um observador que devolve uma lista parcial como se fosse completa faz o modelo agir
   * sobre uma pagina que nao e' a que existe.* O defeito nao era nao atravessar frames: era
   * **nao dizer** que nao atravessou.
   *
   * Limites reais, declarados e nao escondidos:
   *   - shadow root **ABERTO** entra; **FECHADO** e' inacessivel por design (contado, nao lido);
   *   - **so' frames de mesma origem** (cross-origin lanca `SecurityError` -- *nunca contornar
   *     same-origin por exploracao*);
   *   - profundidade limitada, para nao explodir numa pagina hostil.
   */
  const SELETOR = "a[href], button, input, select, textarea, [role], h1, h2, h3, label";
  const PROFUNDIDADE_SHADOW = 5;
  const PROFUNDIDADE_FRAME = 3;

  /** Monta a entrada saneada de um elemento. O VALOR nunca sai. */
  function entradaDe(el) {
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
    return entrada;
  }

  /** Percorre uma raiz (documento, shadow root) somando nos e descendo nos shadow roots. */
  function coletarDe(raiz, nos, contas, limiteNos, profundidade) {
    for (const el of raiz.querySelectorAll(SELETOR)) {
      if (nos.length >= limiteNos) { contas.truncado = true; return; }
      // nada de invisiveis, nada de dentro de <script>/<style>
      if (!el.offsetParent && el.tagName !== "BODY" && getComputedStyle(el).position !== "fixed") continue;
      nos.push(entradaDe(el));
    }
    if (profundidade >= PROFUNDIDADE_SHADOW) return;
    // Shadow DOM: so' os ABERTOS. Os fechados sao contados mas NUNCA lidos.
    for (const el of raiz.querySelectorAll("*")) {
      if (!el.shadowRoot) continue;
      contas.shadowAbertos += 1;
      try { coletarDe(el.shadowRoot, nos, contas, limiteNos, profundidade + 1); } catch { /* hostil */ }
    }
  }

  /** Percorre iframes de MESMA ORIGEM. Cross-origin lanca -- e ficamos por fora. */
  function coletarFrames(doc, nos, contas, limiteNos, profundidade) {
    let quadros = [];
    try { quadros = Array.from(doc.querySelectorAll("iframe, frame")); } catch { return; }
    contas.framesTotal += quadros.length;
    if (profundidade >= PROFUNDIDADE_FRAME) return;
    for (const q of quadros) {
      let interno = null;
      try { interno = q.contentDocument; } catch { interno = null; }
      if (!interno) { contas.framesCrossOrigin += 1; continue; } // mesma-origem negada
      contas.framesMesmaOrigem += 1;
      contas.framesIncluidos += 1;
      try {
        coletarDe(interno, nos, contas, limiteNos, 0);
        coletarFrames(interno, nos, contas, limiteNos, profundidade + 1);
      } catch { /* hostil */ }
    }
  }

  function observar(limiteNos = 400) {
    const nos = [];
    const contas = {
      shadowAbertos: 0, truncado: false,
      framesTotal: 0, framesMesmaOrigem: 0, framesCrossOrigin: 0, framesIncluidos: 0,
    };
    const todos = document.querySelectorAll(
      "a[href], button, input, select, textarea, [role], h1, h2, h3, label",
    );
    const totalPrincipal = todos.length;
    coletarDe(document, nos, contas, limiteNos, 0);
    // Shadow FECHADO: **nao ha' API para enumerar**. A primeira versao procurava custom
    // elements com filhos -- *um palpite* -- e devolvia `fechadosNaoLidos: 0` para uma pagina
    // que TINHA um shadow fechado. *Um palpite que falha e' pior do que uma declaracao de
    // ignorancia: da' uma contagem que parece medida.*
    //
    // O que se pode afirmar com verdade: se existirem shadow roots fechados, esta observacao
    // **nao os ve e nao os pode ver**. Isso e' uma limitacao ESTRUTURAL, nao uma perda
    // concreta -- e por isso vai em campo proprio, para nao transformar `parcial` em ruido
    // permanente nem esconder o que e' accionavel.
    coletarFrames(document, nos, contas, limiteNos, 0);

    const parcial = contas.truncado || contas.framesCrossOrigin > 0
      || contas.framesTotal > contas.framesIncluidos;

    return {
      url: location.origin + location.pathname,
      titulo: limpar(document.title, 160),
      nos,
      total: nos.length,
      // A VERDADE INTEIRA: o que se viu, o que existe, e o que ficou de fora.
      limites: {
        parcial,
        totalNoDocumentoPrincipal: totalPrincipal,
        shadowRoots: {
          abertosIncluidos: contas.shadowAbertos,
          fechados: "nao_detectaveis_por_design",
          profundidadeMaxima: PROFUNDIDADE_SHADOW,
        },
        frames: {
          existentes: contas.framesTotal,
          mesmaOrigemIncluidos: contas.framesIncluidos,
          crossOriginNaoLidos: contas.framesCrossOrigin,
          profundidadeMaxima: PROFUNDIDADE_FRAME,
        },
        truncadoPorLimiteDeNos: contas.truncado,
        // Sempre presente: o consumidor SABE que a observacao nao pode ser completa.
        limitacaoEstrutural: "shadow roots fechados sao invisiveis a content scripts; "
          + "se existirem, nao estao nesta lista e nao ha' forma de os contar daqui",
      },
    };
  }

  /** Acao de escrita: recusa campos sensiveis -- o utilizador preenche-os. */
  function preencher(seletor, valor) {
    // O TAKEOVER vem primeiro: enquanto o humano esta' a interagir, o agente NAO escreve
    // por cima. *Nao basta recusar credenciais -- e' preciso nao disputar o campo.*
    if (takeoverVivo()) {
      return { ok: false, motivo: "takeover_humano_ativo", desde: takeover.desde };
    }
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
    // A escrita do agente fica MARCADA: o listener de takeover nao a conta como humana.
    escrevendoPeloAgente = true;
    try {
      el.value = String(valor ?? "");
      el.dispatchEvent(new Event("input", { bubbles: true }));
      el.dispatchEvent(new Event("change", { bubbles: true }));
    } finally {
      escrevendoPeloAgente = false;
    }
    return { ok: true };
  }

  /** Clique: so' em elemento visivel e nao sensivel. */
  function clicar(seletor) {
    // *Um clique tambem disputa o controlo.* Enquanto o humano interage, o agente para.
    if (takeoverVivo()) {
      return { ok: false, motivo: "takeover_humano_ativo", desde: takeover.desde };
    }
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
      case "takeover:estado":
        responder({
          ok: true,
          ativo: takeoverVivo(),
          autor: takeoverVivo() ? takeover.autor : "agent",
          desde: takeover.desde,
          motivo: takeover.motivo,
          ttlMs: TAKEOVER_MS,
        });
        return false;
      case "takeover:humano":
        // O utilizador assumiu pelo POPUP: marca localmente para o agente parar ja'.
        marcarTakeover(String(mensagem.motivo ?? "assumido_no_popup"));
        responder({ ok: true, ativo: takeoverVivo(), autor: "human" });
        return false;
      case "takeover:devolver":
        // O UTILIZADOR devolve a sessao ao agente. So' ele pode fazer isto -- e a
        // mensagem so' chega aqui vinda da propria extensao.
        takeover.ativo = false;
        takeover.autor = "agent";
        takeover.motivo = "devolvido_pelo_utilizador";
        responder({ ok: true, ativo: false });
        return false;
      default:
        responder({ ok: false, motivo: "tipo_desconhecido" });
        return false;
    }
  });
})();
