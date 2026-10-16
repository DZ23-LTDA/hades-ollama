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

atualizar();
