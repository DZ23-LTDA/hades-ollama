// HADES Browser Bridge -- popup.
//
// Fala SO' com o service worker da propria extensao. Nao usa eval, nao carrega
// nada remoto, e mostra sempre o estado real (nunca um sucesso inventado).

const $ = (id) => document.getElementById(id);

const ROTULOS = {
  sem_aba_ativa: "Não há aba ativa para conectar.",
  incognito_recusado: "Modo anónimo não é suportado por decisão explícita.",
  url_invalido: "Esta aba não tem um endereço http/https utilizável.",
  esquema_nao_permitido: "Só http e https podem ser conectados.",
  rede_privada_sem_optin: "Rede privada é recusada sem opt-in explícito.",
  credenciais_no_url: "O endereço tem credenciais embutidas; remova-as.",
  url_demasiado_longa: "O endereço é demasiado longo.",
  url_ausente: "Esta aba ainda não tem endereço.",
  limpeza: "",
};

function dizer(motivoOuTexto, erro = false) {
  const texto = ROTULOS[motivoOuTexto] ?? motivoOuTexto ?? "";
  const el = $("aviso");
  el.textContent = texto;
  el.classList.toggle("erro", erro && texto !== "");
}

function formatarPrazo(expiraEm) {
  if (typeof expiraEm !== "number") return "—";
  const restam = Math.max(0, Math.round((expiraEm - Date.now()) / 60000));
  return restam <= 0 ? "expirada" : `expira em ${restam} min`;
}

async function pedir(mensagem) {
  try {
    return await chrome.runtime.sendMessage(mensagem);
  } catch (erro) {
    return { ok: false, motivo: "service_worker_indisponivel", detalhe: String(erro?.message ?? erro) };
  }
}

async function atualizar() {
  const [aba] = await chrome.tabs.query({ active: true, currentWindow: true });
  const est = await pedir({ tipo: "estado" });

  if (!est?.ok) {
    dizer(est?.motivo ?? "estado_indisponivel", true);
    return;
  }

  $("missao").textContent = est.missao ?? "— (nenhuma missão ligada)";
  $("host").textContent = est.host ?? "— (native host não instalado)";
  $("aprovacao").textContent = est.aprovado ? "ligada" : "desligada";
  $("abas").textContent = String(est.concessoes.length);
  $("escopos").replaceChildren(
    ...est.escopos.map((s) => {
      const li = document.createElement("li");
      li.textContent = s;
      return li;
    }),
  );

  const minha = est.concessoes.find((c) => c.tabId === aba?.id);
  if (aba?.url) {
    try {
      const u = new URL(aba.url);
      $("origem-aba").textContent = u.protocol === "http:" || u.protocol === "https:"
        ? u.origin
        : `${u.protocol} (não conectável)`;
    } catch {
      $("origem-aba").textContent = "—";
    }
  }
  $("bt-conectar").disabled = Boolean(minha) || !aba;
  $("bt-desconectar").disabled = !minha;
  $("bt-desconectar").textContent = minha ? `Desconectar (${formatarPrazo(minha.expiraEm)})` : "Desconectar";
}

async function agir(mensagem, sucesso) {
  const r = await pedir(mensagem);
  dizer(r?.ok ? sucesso : (r?.motivo ?? "falhou"), !r?.ok);
  await atualizar();
}

$("bt-conectar").addEventListener("click", () => agir({ tipo: "conectar" }, "Aba conectada."));
$("bt-desconectar").addEventListener("click", async () => {
  const [aba] = await chrome.tabs.query({ active: true, currentWindow: true });
  await agir({ tipo: "desconectar", tabId: aba?.id }, "Aba desconectada.");
});
$("bt-revogar").addEventListener("click", () => agir({ tipo: "desconectar" }, "Todas as abas revogadas."));

/**
 * O CANAL LOCAL (`E-1032`).
 *
 * `nativeMessaging` esta' em `optional_permissions`: enquanto nao for CONCEDIDA,
 * `chrome.runtime.sendNativeMessage` **nao existe** -- nao da' erro de permissao, nao e'
 * uma funcao.
 *
 * E `chrome.permissions.request` **nao pode ser pedido a partir do service worker**:
 * o `sendMessage` e' assincrono, e quando o pedido chega la' o GESTO DO UTILIZADOR JA'
 * EXPIROU. Medido: o Chrome recusa e o motivo e' `permissao_nao_pedivel` (antes disso,
 * o catch devolvia `erro_interno` e escondia a causa).
 *
 * Logo o pedido vive AQUI, no handler do clique -- que e' onde o gesto existe.
 */
async function atualizarCanal() {
  const r = await pedir({ tipo: "canal:diagnostico" });
  const bt = $("bt-canal");
  if (!bt) return;
  if (r?.ligado) {
    bt.textContent = "Desligar do HADES";
    dizer(`Ligado ao host ${r.nomeDoHost} (protocolo ${r.host?.protocolo}).`, false);
  } else if (r?.permissaoConcedida) {
    bt.textContent = "Ligar ao HADES";
    dizer("Permissao concedida; o host ainda nao respondeu.", false);
  } else {
    bt.textContent = "Ligar ao HADES";
    dizer("O canal local esta' desligado. Nada e' enviado ao PC.", false);
  }
}

/**
 * `chrome.permissions.request` **TRAVA em vez de falhar.** Medido (`E-1032`): quando o
 * contexto nao consegue apresentar o dialogo nativo, a promessa **nunca resolve nem
 * rejeita** -- o handler fica congelado e o utilizador ve "A pedir permissao..." para
 * sempre, sem erro e sem timeout. *Uma promessa pendurada e' pior do que uma excepcao.*
 *
 * Por isso o pedido tem prazo. Sem isto, o unico sintoma e' um botao que nao faz nada.
 */
function comPrazo(promessa, ms, motivo) {
  return Promise.race([
    promessa,
    new Promise((_, rej) => setTimeout(() => rej(new Error(motivo)), ms)),
  ]);
}

const btCanal = $("bt-canal");
if (btCanal) {
  btCanal.addEventListener("click", async () => {
    dizer("A pedir permissão…", false);
    let concedida = false;
    try {
      concedida = await chrome.permissions.contains({ permissions: ["nativeMessaging"] });
    } catch (erro) {
      dizer("contains falhou: " + String(erro?.message ?? erro), true);
      return;
    }
    if (!concedida) {
      try {
        concedida = await comPrazo(
          chrome.permissions.request({ permissions: ["nativeMessaging"] }),
          8000,
          "o navegador não respondeu ao pedido de permissão",
        );
      } catch (erro) {
        dizer("Pedido de permissão: " + String(erro?.message ?? erro), true);
        return;
      }
    }
    if (!concedida) {
      dizer("Permissão recusada: a extensão continua sem falar com o Companion.", true);
      return;
    }
    const r = await pedir({ tipo: "canal:ligar" });
    if (r?.ok) {
      dizer(`Ligado ao Companion (protocolo ${r.resposta?.protocolo ?? "?"}).`, false);
    } else {
      dizer("canal:ligar -> " + (r?.motivo ?? r?.detalhe ?? "falhou"), true);
    }
    await atualizarCanal();
    if (!r?.ok) dizer("canal:ligar -> " + (r?.motivo ?? r?.detalhe ?? "falhou"), true);
  });
}

/**
 * O OPT-IN DE REDE PRIVADA (`E-1033`).
 *
 * O protocolo ja' tinha `loopbackAutorizado()` -- **mas ninguem o chamava**, logo a rede
 * privada era sempre recusada e o opt-in era codigo morto. E o `content_scripts.matches`
 * cobre *so'* `127.0.0.1` e `localhost`: a unica origem onde o content script e'
 * injectado era a unica que o HADES nao podia usar.
 *
 * Aqui o utilizador LIGA-O, com o risco escrito ao lado. Por omissao continua DESLIGADO.
 */
async function atualizarOptIn() {
  const r = await pedir({ tipo: "optin:estado" });
  const bt = $("bt-optin");
  if (!bt) return;
  bt.textContent = r?.ligado ? "Desligar rede privada" : "Ligar rede privada";
  const alvo = $("optin-estado");
  if (alvo) {
    alvo.textContent = r?.ligado
      ? "Ligado: o HADES pode abrir localhost e 127.0.0.1."
      : "Desligado: localhost e 127.0.0.1 são recusados.";
  }
}

const btOptIn = $("bt-optin");
if (btOptIn) {
  btOptIn.addEventListener("click", async () => {
    const atual = await pedir({ tipo: "optin:estado" });
    const r = await pedir({ tipo: "optin:definir", ligado: !atual?.ligado });
    dizer(r?.ligado ? "Rede privada ligada por decisão sua." : "Rede privada desligada.", false);
    await atualizarOptIn();
  });
}

atualizar();
atualizarCanal();
atualizarOptIn();
