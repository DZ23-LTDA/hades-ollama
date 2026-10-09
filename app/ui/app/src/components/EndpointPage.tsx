import { useEffect, useState } from "react";
import {
  CheckIcon,
  ClipboardIcon,
  ComputerDesktopIcon,
  CloudArrowUpIcon,
  DevicePhoneMobileIcon,
  PlusIcon,
  Cog6ToothIcon,
  ArrowPathIcon,
  ShieldCheckIcon,
} from "@heroicons/react/24/outline";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { SettingsTabs } from "@/components/SettingsTabs";
import { API_BASE } from "@/lib/config";
import { agentFetch } from "@/lib/agenticClient";
import {
  endpointHealthLabel,
  fetchGatewayConnection,
  gatewayRotationLabel,
  gatewaySetups,
  rotateGatewayKey,
  type EndpointHealthStatus,
  type GatewayConnection,
} from "@/lib/endpoint";
import { Link } from "@tanstack/react-router";

// apiPort deriva a porta real a partir do endereço efetivo da API (API_BASE,
// ou a mesma origem quando ele é relativo), em vez de assumir um valor fixo.
function apiPort(): string {
  try {
    const href = typeof window !== "undefined" ? window.location.href : "http://localhost";
    const base = API_BASE || (typeof window !== "undefined" ? window.location.origin : "http://localhost:11434");
    const url = new URL(base, href);
    if (url.port) return url.port;
    return url.protocol === "https:" ? "443" : "80";
  } catch {
    return "11434";
  }
}

function CopyButton({ text, label }: { text: string; label: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      aria-label={`Copiar ${label}`}
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        } catch {
          setCopied(false);
        }
      }}
      className="shrink-0 rounded-lg p-2 text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900 dark:hover:bg-neutral-800 dark:hover:text-white"
    >
      {copied ? (
        <CheckIcon className="h-4 w-4 text-emerald-500" />
      ) : (
        <ClipboardIcon className="h-4 w-4" />
      )}
    </button>
  );
}

export function EndpointPage() {
  const [models, setModels] = useState<string[]>([]);
  const [model, setModel] = useState("");
  const [hostname, setHostname] = useState("Este computador");
  const [osName, setOsName] = useState("Local");
  const [serverVersion, setServerVersion] = useState("");
  const [hostStatus, setHostStatus] = useState<EndpointHealthStatus>("checking");

  // Qual aviso informativo (pré-requisitos) está aberto; nenhum botão fica morto.
  const [openInfo, setOpenInfo] = useState<string | null>(null);
  const toggleInfo = (key: string) => setOpenInfo((current) => (current === key ? null : key));

  // Pareamento real de dispositivo (ex.: celular) via backend de companion devices.
  const [pairing, setPairing] = useState<{ code: string; expiresAt?: string } | null>(null);
  const [pairingError, setPairingError] = useState<string | null>(null);
  const [pairingBusy, setPairingBusy] = useState(false);

  const startPairing = async () => {
    setPairingBusy(true);
    setPairingError(null);
    setPairing(null);
    try {
      const result = await agentFetch<{ pairing_code: string; expires_at?: string }>(
        "/api/agent/v1/devices/pair/start",
        { method: "POST", body: "{}" },
      );
      setPairing({ code: result.pairing_code, expiresAt: result.expires_at });
    } catch (error) {
      setPairingError(
        error instanceof Error
          ? error.message
          : "Não foi possível iniciar o pareamento agora.",
      );
    } finally {
      setPairingBusy(false);
    }
  };

  // Gateway de terceiros: endereço base, chave e rotação automática.
  const [gateway, setGateway] = useState<GatewayConnection | null>(null);
  const [gatewayError, setGatewayError] = useState<string | null>(null);
  const [gatewayBusy, setGatewayBusy] = useState(false);
  const [showGatewayKey, setShowGatewayKey] = useState(false);

  const rotateGateway = async () => {
    setGatewayBusy(true);
    setGatewayError(null);
    try {
      const updated = await rotateGatewayKey();
      setGateway(updated);
      setShowGatewayKey(true);
    } catch (error) {
      setGatewayError(
        error instanceof Error
          ? error.message
          : "Não foi possível gerar a chave do gateway agora.",
      );
    } finally {
      setGatewayBusy(false);
    }
  };

  useEffect(() => {
    void fetchGatewayConnection().then(setGateway);
  }, []);

  useEffect(() => {
    fetch(`${API_BASE}/api/tags`)
      .then((response) => {
        if (!response.ok) throw new Error(`tags request failed: ${response.status}`);
        setHostStatus("online");
        return response.json();
      })
      .then((body: { models?: Array<{ name?: string; model?: string }> }) => {
        const names = (body.models ?? [])
          .map((item) => item.name ?? item.model ?? "")
          .filter(Boolean);
        setModels(names);
        setModel((current) => current || names[0] || "");
      })
      .catch(() => {
        setHostStatus("offline");
        setModels([]);
      });

    // Obter hostname local se disponível
    fetch(`${API_BASE}/api/v1/host`)
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data && data.hostname) {
          setHostname(data.hostname);
          if (data.os) setOsName(data.os === "windows" ? "Windows" : data.os === "darwin" ? "macOS" : "Linux");
        }
      })
      .catch(() => {});

    fetch(`${API_BASE}/api/version`)
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data && data.version) setServerVersion(data.version);
      })
      .catch(() => {});
  }, []);

  // Endereço base efetivo: o do gateway quando o endpoint responde, senão o
  // endereço real da API (mesma origem nos builds de produção).
  const gatewayBase = (
    gateway?.base_url ||
    API_BASE ||
    (typeof window !== "undefined" ? window.location.origin : "")
  ).replace(/\/+$/, "");
  const gatewayKey = gateway?.gateway_key ?? "";

  return (
    <SidebarLayout
      title="Computadores & Endpoint"
      sidebar={<AppSidebar current="endpoint" />}
    >
      <SettingsTabs current="endpoint" />
      <div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-900">
        <div className="mx-auto w-full max-w-5xl px-6 pb-16 pt-8 lg:px-10">
          {/* Cabeçalho da seção de Computadores */}
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h1 className="page-title font-bold sm:text-3xl">
                Computadores
              </h1>
              <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
                Gerencie seus computadores locais e na nuvem para execução segura de missões e Computer Use.
              </p>
            </div>
            <button
              type="button"
              onClick={() => toggleInfo("cloud")}
              aria-expanded={openInfo === "cloud"}
              className="inline-flex shrink-0 items-center gap-2 rounded-xl border border-neutral-300 bg-white px-4 py-2 text-sm font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              <PlusIcon className="h-4 w-4" />
              Criar computador na nuvem
            </button>
          </div>
          {openInfo === "cloud" && (
            <div className="mt-4 rounded-xl border border-amber-200 bg-amber-50 p-4 text-xs leading-5 text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/20 dark:text-amber-200" role="status">
              <strong className="font-semibold">Pré-requisito:</strong> computadores na nuvem exigem um provedor
              de infraestrutura configurado (credenciais e região). Este build local-first não inclui um provedor
              de nuvem, portanto a criação ainda não está disponível. Enquanto isso, use o seu computador local,
              que já aparece conectado abaixo.
            </div>
          )}

          {/* Cards de Computadores Conectados (Fiel ao Manus) */}
          <div className="mt-8 grid grid-cols-1 gap-6 md:grid-cols-2">
            {/* Card 1: Computador Conectado do Usuário */}
            <div className="flex flex-col justify-between rounded-2xl border border-neutral-200 bg-white p-6 shadow-sm transition-all dark:border-neutral-800 dark:bg-neutral-950">
              <div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-neutral-800">
                      <ComputerDesktopIcon className="h-6 w-6 text-neutral-900 dark:text-white" />
                    </div>
                    <div>
                      <h2 className="text-base font-bold text-neutral-900 dark:text-white">
                        {hostname}
                      </h2>
                      <div className="flex items-center gap-2">
                        <span
                          className={`flex items-center gap-1.5 text-xs font-medium ${hostStatus === "online" ? "text-emerald-600 dark:text-emerald-400" : hostStatus === "offline" ? "text-red-600 dark:text-red-400" : "text-neutral-500 dark:text-neutral-400"}`}
                          role="status"
                        >
                          <span className={`h-2 w-2 rounded-full ${hostStatus === "online" ? "bg-emerald-500 animate-pulse" : hostStatus === "offline" ? "bg-red-500" : "bg-neutral-400 animate-pulse"}`} />
                          {endpointHealthLabel(hostStatus)}
                        </span>
                        <span className="text-xs text-neutral-400">• {osName} / Local {serverVersion ? `(v${serverVersion})` : ""}</span>
                      </div>
                    </div>
                  </div>
                  <span className={`rounded-full px-2.5 py-1 text-[11px] font-semibold ${hostStatus === "online" ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300" : "bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300"}`}>
                    {hostStatus === "online" ? "Host Ativo" : hostStatus === "offline" ? "Host Offline" : "Verificando host"}
                  </span>
                </div>

                <div className="mt-6 rounded-xl bg-neutral-50 p-3.5 text-xs text-neutral-600 dark:bg-neutral-900 dark:text-neutral-300">
                  <div className="flex items-center justify-between py-1 border-b border-neutral-200/50 dark:border-neutral-800">
                    <span className="text-neutral-400">Porta da API:</span>
                    <span className="font-mono font-medium">{apiPort()}</span>
                  </div>
                  <div className="flex items-center justify-between py-1 border-b border-neutral-200/50 dark:border-neutral-800">
                    <span className="text-neutral-400">Sistema:</span>
                    <span className="font-medium">{osName}</span>
                  </div>
                  <div className="flex items-center justify-between py-1">
                    <span className="text-neutral-400">Versão do servidor:</span>
                    <span className="font-mono font-medium">{serverVersion ? `v${serverVersion}` : "—"}</span>
                  </div>
                </div>
              </div>

              <div className="mt-6 flex flex-col gap-3">
                <button
                  type="button"
                  onClick={() => toggleInfo("remote")}
                  aria-expanded={openInfo === "remote"}
                  className="w-full rounded-xl border border-neutral-300 bg-white py-2.5 text-sm font-semibold text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
                >
                  Acesso remoto
                </button>
                {openInfo === "remote" && (
                  <div className="rounded-xl border border-amber-200 bg-amber-50 p-3.5 text-xs leading-5 text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/20 dark:text-amber-200" role="status">
                    Por padrão este host responde apenas localmente. Para acesso remoto é preciso ativar
                    <strong> “Expor o Hades na rede”</strong> em Configurações e proteger a porta
                    {" "}<span className="font-mono">{apiPort()}</span> com TLS/autenticação (por exemplo, um proxy reverso).
                    Enquanto isso não for feito, o acesso remoto permanece desligado por segurança.
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <a
                    href="#endpoints-section"
                    className="flex-1 inline-flex items-center justify-center gap-1.5 rounded-xl border border-neutral-200 py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-900"
                  >
                    <Cog6ToothIcon className="h-4 w-4" />
                    Configurações
                  </a>
                  <Link
                    to="/tasks"
                    className="flex-1 inline-flex items-center justify-center gap-1.5 rounded-xl border border-neutral-200 py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-800 dark:text-neutral-300 dark:hover:bg-neutral-900"
                  >
                    <ArrowPathIcon className="h-4 w-4" />
                    Tarefas
                  </Link>
                </div>
              </div>
            </div>

            {/* Card 2: Conectar novo ou nuvem */}
            <div className="flex flex-col justify-between rounded-2xl border border-dashed border-neutral-300 bg-white/50 p-6 text-center dark:border-neutral-700 dark:bg-neutral-900/50">
              <div>
                <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-neutral-100 dark:bg-neutral-800">
                  <CloudArrowUpIcon className="h-6 w-6 text-neutral-400" />
                </div>
                <h2 className="mt-4 text-base font-semibold text-neutral-900 dark:text-white">
                  Conectar outro dispositivo
                </h2>
                <p className="mx-auto mt-2 max-w-xs text-xs text-neutral-500 dark:text-neutral-400">
                  Adicione nós remotos, instâncias na nuvem ou conecte seu smartphone para controlar tarefas em andamento.
                </p>
              </div>

              <div className="mt-6 flex flex-col gap-2.5">
                <button
                  type="button"
                  onClick={() => toggleInfo("computer")}
                  aria-expanded={openInfo === "computer"}
                  className="w-full rounded-xl border border-neutral-300 bg-white py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
                >
                  Conectar meu computador
                </button>
                {openInfo === "computer" && (
                  <div className="rounded-xl border border-amber-200 bg-amber-50 p-3 text-left text-[11px] leading-5 text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/20 dark:text-amber-200" role="status">
                    Para conectar outro computador como nó remoto é preciso que ele rode o Hades e seja alcançável
                    pela rede (host e porta <span className="font-mono">{apiPort()}</span> expostos com TLS). Não há,
                    neste build local-first, um provedor que faça esse provisionamento automaticamente.
                  </div>
                )}
                <button
                  type="button"
                  onClick={() => void startPairing()}
                  disabled={pairingBusy}
                  className="inline-flex w-full items-center justify-center gap-1.5 rounded-xl border border-neutral-300 bg-white py-2 text-xs font-medium text-neutral-700 hover:bg-neutral-50 disabled:opacity-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
                >
                  <DevicePhoneMobileIcon className="h-4 w-4" />
                  {pairingBusy ? "Gerando código…" : "Parear o seu telefone"}
                </button>
                {pairingError && (
                  <div className="rounded-xl border border-red-200 bg-red-50 p-3 text-left text-[11px] leading-5 text-red-800 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-200" role="alert">
                    {pairingError}
                  </div>
                )}
                {pairing && (
                  <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-3 text-left text-[11px] leading-5 text-emerald-900 dark:border-emerald-900/50 dark:bg-emerald-950/20 dark:text-emerald-200" role="status">
                    <p className="font-semibold">Código de pareamento:</p>
                    <p className="mt-1 select-all font-mono text-base tracking-widest text-emerald-700 dark:text-emerald-300">{pairing.code}</p>
                    <p className="mt-2">
                      Abra o app complementar do Hades no seu celular e informe este código para concluir o pareamento.
                      {pairing.expiresAt ? ` O código expira às ${new Date(pairing.expiresAt).toLocaleTimeString("pt-BR")}.` : ""}
                    </p>
                    <p className="mt-1 opacity-80">
                      O pareamento só se conclui a partir do app no telefone; nada é enviado para fora deste computador.
                    </p>
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Seção de Endereços da API */}
          <section id="endpoints-section" className="mt-10 rounded-2xl border border-neutral-200 bg-white p-6 dark:border-neutral-800 dark:bg-neutral-950">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-neutral-900 dark:text-white">
                  Endereços da API & Conexão
                </h3>
                <p className="mt-1 text-xs text-neutral-500">
                  Use os endpoints abaixo para conectar Claude Code, Codex, Cursor e SDKs locais.
                </p>
              </div>
              <span className="flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
                <ShieldCheckIcon className="h-4 w-4" />
                Seguro
              </span>
            </div>

            <ul className="mt-5 space-y-3">
              <li className="flex items-center gap-3 rounded-xl bg-neutral-50 p-3.5 dark:bg-neutral-900">
                <div className="min-w-0 flex-1">
                  <p className="text-[11px] font-semibold uppercase tracking-wider text-neutral-500">
                    Ollama Nativo
                  </p>
                  <code className="mt-0.5 block truncate font-mono text-sm text-neutral-900 dark:text-neutral-100">
                    {API_BASE}
                  </code>
                  <p className="mt-0.5 text-[11px] text-neutral-400">/api/chat, /api/generate, /api/tags, /api/agent</p>
                </div>
                <CopyButton text={API_BASE} label="Ollama Nativo" />
              </li>
              <li className="flex items-center gap-3 rounded-xl bg-neutral-50 p-3.5 dark:bg-neutral-900">
                <div className="min-w-0 flex-1">
                  <p className="text-[11px] font-semibold uppercase tracking-wider text-neutral-500">
                    Compatível com OpenAI
                  </p>
                  <code className="mt-0.5 block truncate font-mono text-sm text-neutral-900 dark:text-neutral-100">
                    {`${API_BASE}/v1`}
                  </code>
                  <p className="mt-0.5 text-[11px] text-neutral-400">chat/completions, responses, models</p>
                </div>
                <CopyButton text={`${API_BASE}/v1`} label="compatível com OpenAI" />
              </li>
              <li className="flex items-center gap-3 rounded-xl bg-neutral-50 p-3.5 dark:bg-neutral-900">
                <div className="min-w-0 flex-1">
                  <p className="text-[11px] font-semibold uppercase tracking-wider text-neutral-500">
                    Compatível com Anthropic (Claude Code)
                  </p>
                  <code className="mt-0.5 block truncate font-mono text-sm text-neutral-900 dark:text-neutral-100">
                    {API_BASE}
                  </code>
                  <p className="mt-0.5 text-[11px] text-neutral-400">/v1/messages (Claude Code, SDK Anthropic)</p>
                </div>
                <CopyButton text={API_BASE} label="compatível com Anthropic" />
              </li>
            </ul>
          </section>

          {/* Configurar inferência de terceiros */}
          <section
            id="gateway-section"
            className="mt-8 rounded-2xl border border-neutral-200 bg-white p-6 dark:border-neutral-800 dark:bg-neutral-950"
          >
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="text-base font-bold text-neutral-900 dark:text-white">
                  Configurar inferência de terceiros
                </h3>
                <p className="mt-1 text-xs text-neutral-500">
                  Um único endereço para os modelos locais e para as APIs de
                  terceiros. Se um provedor recusar a chamada, o gateway
                  tenta o próximo candidato sem trocar a configuração do
                  Claude ou do Codex.
                </p>
              </div>
              <span className="rounded-full bg-emerald-50 px-2.5 py-1 text-[11px] font-medium text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300">
                {gateway
                  ? gatewayRotationLabel(gateway.rotation)
                  : "Rotação padrão · 3 tentativas"}
              </span>
            </div>

            <div className="mt-5 grid gap-3 lg:grid-cols-2">
              <div className="rounded-xl bg-neutral-50 p-3.5 dark:bg-neutral-900">
                <p className="text-[11px] font-semibold uppercase tracking-wider text-neutral-500">
                  URL base do gateway
                </p>
                <code
                  aria-label="URL base do gateway"
                  className="mt-0.5 block break-all font-mono text-sm text-neutral-900 dark:text-neutral-100"
                >
                  {gatewayBase}
                </code>
                <p className="mt-0.5 text-[11px] text-neutral-400">
                  /v1/messages (Claude), /v1/responses (Codex),
                  /v1/chat/completions (SDKs)
                </p>
                <div className="mt-2 flex items-center">
                  <CopyButton text={gatewayBase} label="URL base do gateway" />
                </div>
              </div>

              <div className="rounded-xl bg-neutral-50 p-3.5 dark:bg-neutral-900">
                <p className="text-[11px] font-semibold uppercase tracking-wider text-neutral-500">
                  Chave de API do gateway
                </p>
                {gatewayKey ? (
                  <>
                    <code
                      aria-label="Chave de API do gateway"
                      className="mt-0.5 block break-all font-mono text-sm text-neutral-900 dark:text-neutral-100"
                    >
                      {showGatewayKey ? gatewayKey : "•".repeat(28)}
                    </code>
                    <p className="mt-0.5 text-[11px] text-neutral-400">
                      Guardada fora do arquivo de configuração, na variável{" "}
                      <span className="font-mono">
                        {gateway?.gateway_key_env}
                      </span>
                      .
                    </p>
                    <div className="mt-2 flex items-center gap-2">
                      <button
                        type="button"
                        onClick={() => setShowGatewayKey((current) => !current)}
                        className="rounded-lg border border-neutral-300 bg-white px-2.5 py-1 text-[11px] font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
                      >
                        {showGatewayKey ? "Ocultar" : "Mostrar"}
                      </button>
                      <CopyButton text={gatewayKey} label="chave do gateway" />
                    </div>
                  </>
                ) : (
                  <p className="mt-1 text-xs text-neutral-500">
                    Nenhuma chave ativa. Sem chave, o gateway aceita apenas
                    pedidos deste computador — é o que mantém a inferência
                    local privada.
                  </p>
                )}
                <button
                  type="button"
                  onClick={() => void rotateGateway()}
                  disabled={gatewayBusy}
                  className="mt-3 inline-flex w-full items-center justify-center gap-1.5 rounded-lg bg-neutral-900 px-3 py-2 text-xs font-medium text-white hover:bg-neutral-800 disabled:opacity-50 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200"
                >
                  <ArrowPathIcon className="h-3.5 w-3.5" />
                  {gatewayBusy
                    ? "Gerando…"
                    : gatewayKey
                      ? "Gerar nova chave"
                      : "Gerar chave do gateway"}
                </button>
                {gatewayError && (
                  <div
                    role="alert"
                    className="mt-2 rounded-lg border border-red-200 bg-red-50 p-2.5 text-left text-[11px] leading-5 text-red-800 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-200"
                  >
                    {gatewayError}
                  </div>
                )}
              </div>
            </div>

            {gateway?.loopback_only && (
              <div
                role="status"
                className="mt-3 rounded-xl border border-amber-200 bg-amber-50 p-3 text-left text-[11px] leading-5 text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/20 dark:text-amber-200"
              >
                O gateway está no modo <span className="font-semibold">somente este computador</span>:
                clientes remotos recebem 401 até existir uma chave. Gere a chave
                acima antes de apontar outro dispositivo para este endereço.
              </div>
            )}

            {gateway && gateway.protocols.length > 0 && (
              <ul className="mt-4 grid gap-2 sm:grid-cols-2">
                {gateway.protocols.map((protocol) => (
                  <li
                    key={protocol.path}
                    className="rounded-lg border border-neutral-200 px-3 py-2 text-[11px] dark:border-neutral-800"
                  >
                    <span className="font-mono text-neutral-900 dark:text-neutral-100">
                      {protocol.path}
                    </span>
                    <span className="ml-2 text-neutral-500">
                      {protocol.label}
                    </span>
                  </li>
                ))}
              </ul>
            )}

            {gateway && (
              <ul className="mt-4 space-y-3">
                {gatewaySetups(gateway, model).map((setup) => (
                  <li
                    key={setup.id}
                    className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-950"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <h4 className="text-sm font-bold text-neutral-900 dark:text-white">
                          {setup.title}
                        </h4>
                        <p className="mt-0.5 text-xs text-neutral-500">
                          {setup.description}
                        </p>
                      </div>
                      <CopyButton text={setup.code} label={setup.title} />
                    </div>
                    <pre
                      tabIndex={0}
                      aria-label={`Comando de integração ${setup.title}`}
                      className="mt-3 overflow-x-auto rounded-xl bg-neutral-900 p-3.5 font-mono text-xs leading-5 text-neutral-100 dark:bg-black"
                    >
                      {setup.code}
                    </pre>
                  </li>
                ))}
              </ul>
            )}

            {gateway && (
              <p className="mt-3 text-[11px] text-neutral-400">
                {gateway.providers} provedor
                {gateway.providers === 1 ? "" : "es"} declarado
                {gateway.providers === 1 ? "" : "s"} em{" "}
                <span className="font-mono">{gateway.config_path}</span>.
              </p>
            )}
          </section>

          {/* Configuração Rápida de Ferramentas */}
          <section className="mt-8">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h3 className="text-base font-bold text-neutral-900 dark:text-white">
                Comandos de Integração Rápida
              </h3>
              {models.length > 0 && (
                <label className="flex items-center gap-2 text-xs text-neutral-500">
                  Modelo ativo:
                  <select
                    value={model}
                    onChange={(event) => setModel(event.target.value)}
                    className="rounded-lg border border-neutral-300 bg-white px-2.5 py-1 text-xs text-neutral-900 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100"
                  >
                    {models.map((name) => (
                      <option key={name} value={name}>
                        {name}
                      </option>
                    ))}
                  </select>
                </label>
              )}
            </div>

            <ul className="mt-4 space-y-4">
              <li className="rounded-2xl border border-neutral-200 bg-white p-5 shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h4 className="text-sm font-bold text-neutral-900 dark:text-white">
                      Claude Code
                    </h4>
                    <p className="mt-0.5 text-xs text-neutral-500">
                      Execução com modelos locais através da rota compatível.
                    </p>
                  </div>
                  <CopyButton text={`ollama launch claude --model ${model || "qwen2.5-coder:7b"}`} label="Claude Code" />
                </div>
                <pre tabIndex={0} aria-label="Comando de integração Claude" className="mt-3 overflow-x-auto rounded-xl bg-neutral-900 p-3.5 font-mono text-xs leading-5 text-neutral-100 dark:bg-black">
                  {`ollama launch claude --model ${model || "qwen2.5-coder:7b"}`}
                </pre>
              </li>
              <li className="rounded-2xl border border-neutral-200 bg-white p-5 shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h4 className="text-sm font-bold text-neutral-900 dark:text-white">
                      Codex CLI
                    </h4>
                    <p className="mt-0.5 text-xs text-neutral-500">
                      Ollama configura e abre o Codex com o modelo escolhido.
                    </p>
                  </div>
                  <CopyButton text={`ollama launch codex --model ${model || "qwen2.5-coder:7b"}`} label="Codex CLI" />
                </div>
                <pre tabIndex={0} aria-label="Comando de integração Codex" className="mt-3 overflow-x-auto rounded-xl bg-neutral-900 p-3.5 font-mono text-xs leading-5 text-neutral-100 dark:bg-black">
                  {`ollama launch codex --model ${model || "qwen2.5-coder:7b"}`}
                </pre>
              </li>
            </ul>
          </section>
        </div>
      </div>
    </SidebarLayout>
  );
}
