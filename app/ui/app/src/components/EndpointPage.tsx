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
import { endpointHealthLabel, type EndpointHealthStatus } from "@/lib/endpoint";
import { Link } from "@tanstack/react-router";

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
  const [osName, setOsName] = useState("Local-First");
  const [serverVersion, setServerVersion] = useState("");
  const [hostStatus, setHostStatus] = useState<EndpointHealthStatus>("checking");

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
              disabled
              title="Ainda não configurado: nenhum provedor de computador em nuvem foi configurado"
              className="inline-flex shrink-0 cursor-not-allowed items-center gap-2 rounded-xl bg-neutral-200 px-4 py-2 text-sm font-medium text-neutral-500 dark:bg-neutral-800 dark:text-neutral-500"
            >
              <PlusIcon className="h-4 w-4" />
              Criar computador na nuvem (não configurado)
            </button>
          </div>

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
                      <h3 className="text-base font-bold text-neutral-900 dark:text-white">
                        {hostname}
                      </h3>
                      <div className="flex items-center gap-2">
                        <span
                          className={`flex items-center gap-1.5 text-xs font-medium ${hostStatus === "online" ? "text-emerald-600 dark:text-emerald-400" : hostStatus === "offline" ? "text-red-600 dark:text-red-400" : "text-neutral-500 dark:text-neutral-400"}`}
                          role="status"
                        >
                          <span className={`h-2 w-2 rounded-full ${hostStatus === "online" ? "bg-emerald-500 animate-pulse" : hostStatus === "offline" ? "bg-red-500" : "bg-neutral-400 animate-pulse"}`} />
                          {endpointHealthLabel(hostStatus)}
                        </span>
                        <span className="text-xs text-neutral-400">• {osName} / Local-First {serverVersion ? `(v${serverVersion})` : ""}</span>
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
                    <span className="font-mono font-medium">11434</span>
                  </div>
                  <div className="flex items-center justify-between py-1 border-b border-neutral-200/50 dark:border-neutral-800">
                    <span className="text-neutral-400">Isolamento:</span>
                    <span className="font-medium text-emerald-600 dark:text-emerald-400">Local Sandbox & RLS</span>
                  </div>
                  <div className="flex items-center justify-between py-1">
                    <span className="text-neutral-400">Permissão de Agente:</span>
                    <span className="font-medium">Total com aprovação humana</span>
                  </div>
                </div>
              </div>

              <div className="mt-6 flex flex-col gap-3">
                <button
                  type="button"
                  disabled
                  title="Ainda não configurado: acesso remoto exige um endpoint e aprovação configurados"
                  className="w-full cursor-not-allowed rounded-xl bg-neutral-200 py-2.5 text-sm font-semibold text-neutral-500 dark:bg-neutral-800 dark:text-neutral-500"
                >
                  Acesso remoto (não configurado)
                </button>
                <p className="text-center text-xs text-neutral-500 dark:text-neutral-400" role="status">
                  Ainda não configurado — este host local não autoriza acesso remoto automaticamente.
                </p>
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
                <h3 className="mt-4 text-base font-semibold text-neutral-900 dark:text-white">
                  Conectar outro dispositivo
                </h3>
                <p className="mx-auto mt-2 max-w-xs text-xs text-neutral-500 dark:text-neutral-400">
                  Adicione nós remotos, instâncias na nuvem ou conecte seu smartphone para controlar tarefas em andamento.
                </p>
              </div>

              <div className="mt-6 flex flex-col gap-2.5">
                <button
                  type="button"
                  disabled
                  title="Ainda não configurado: pareamento de outro computador ainda não está disponível"
                  className="w-full cursor-not-allowed rounded-xl border border-neutral-200 bg-neutral-100 py-2 text-xs font-medium text-neutral-500 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-500"
                >
                  Conectar meu computador (não configurado)
                </button>
                <button
                  type="button"
                  disabled
                  title="Ainda não configurado: controle por telefone exige pareamento explícito"
                  className="inline-flex w-full cursor-not-allowed items-center justify-center gap-1.5 rounded-xl border border-neutral-200 bg-neutral-100 py-2 text-xs font-medium text-neutral-500 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-500"
                >
                  <DevicePhoneMobileIcon className="h-4 w-4" />
                  Controlar pelo seu telefone (não configurado)
                </button>
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
                    OpenAI-Compatible
                  </p>
                  <code className="mt-0.5 block truncate font-mono text-sm text-neutral-900 dark:text-neutral-100">
                    {`${API_BASE}/v1`}
                  </code>
                  <p className="mt-0.5 text-[11px] text-neutral-400">chat/completions, responses, models</p>
                </div>
                <CopyButton text={`${API_BASE}/v1`} label="OpenAI-Compatible" />
              </li>
              <li className="flex items-center gap-3 rounded-xl bg-neutral-50 p-3.5 dark:bg-neutral-900">
                <div className="min-w-0 flex-1">
                  <p className="text-[11px] font-semibold uppercase tracking-wider text-neutral-500">
                    Anthropic-Compatible (Claude Code)
                  </p>
                  <code className="mt-0.5 block truncate font-mono text-sm text-neutral-900 dark:text-neutral-100">
                    {API_BASE}
                  </code>
                  <p className="mt-0.5 text-[11px] text-neutral-400">/v1/messages (Claude Code, SDK Anthropic)</p>
                </div>
                <CopyButton text={API_BASE} label="Anthropic-Compatible" />
              </li>
            </ul>
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
                <pre className="mt-3 overflow-x-auto rounded-xl bg-neutral-900 p-3.5 font-mono text-xs leading-5 text-neutral-100 dark:bg-black">
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
                <pre className="mt-3 overflow-x-auto rounded-xl bg-neutral-900 p-3.5 font-mono text-xs leading-5 text-neutral-100 dark:bg-black">
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
