import { useEffect, useState } from "react";
import { CheckIcon, ClipboardIcon } from "@heroicons/react/24/outline";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { SettingsTabs } from "@/components/SettingsTabs";
import { API_BASE } from "@/lib/config";
import { endpoints, toolSetups } from "@/lib/endpoint";

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
        <CheckIcon className="h-4 w-4" />
      ) : (
        <ClipboardIcon className="h-4 w-4" />
      )}
    </button>
  );
}

export function EndpointPage() {
  const [models, setModels] = useState<string[]>([]);
  const [model, setModel] = useState("");

  useEffect(() => {
    fetch(`${API_BASE}/api/tags`)
      .then((response) => (response.ok ? response.json() : { models: [] }))
      .then((body: { models?: Array<{ name?: string; model?: string }> }) => {
        const names = (body.models ?? [])
          .map((item) => item.name ?? item.model ?? "")
          .filter(Boolean);
        setModels(names);
        setModel((current) => current || names[0] || "");
      })
      .catch(() => setModels([]));
  }, []);

  return (
    <SidebarLayout
      title="Configurações"
      sidebar={<AppSidebar current="endpoint" />}
    >
      <SettingsTabs current="endpoint" />
      <div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-900">
        <div className="mx-auto w-full max-w-4xl px-6 pb-14 pt-10 lg:px-12">
          <h2 className="font-rounded text-3xl font-semibold tracking-tight text-neutral-950 dark:text-white">
            Endpoint da API
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-neutral-500 dark:text-neutral-400">
            Use os seus modelos locais, do Ollama Cloud e dos provedores
            configurados em qualquer ferramenta: Claude Code, Codex, SDKs e apps
            compatíveis com OpenAI ou Anthropic.
          </p>

          <section className="mt-6 rounded-2xl border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-950">
            <h3 className="text-sm font-semibold text-neutral-900 dark:text-white">
              Endereços
            </h3>
            <ul className="mt-3 space-y-2">
              {endpoints().map((item) => (
                <li
                  key={item.label}
                  className="flex items-center gap-3 rounded-xl bg-neutral-50 px-3 py-2 dark:bg-neutral-900"
                >
                  <div className="min-w-0 flex-1">
                    <p className="text-[11px] text-neutral-500">{item.label}</p>
                    <code className="block truncate text-sm text-neutral-900 dark:text-neutral-100">
                      {item.url}
                    </code>
                    <p className="text-[11px] text-neutral-400">{item.hint}</p>
                  </div>
                  <CopyButton text={item.url} label={item.label} />
                </li>
              ))}
            </ul>
            <p className="mt-3 text-xs leading-5 text-neutral-500">
              Chave de API: no próprio computador não é exigida; use qualquer
              valor, como <code>ollama</code>. Com &quot;Expose Ollama to the
              network&quot; ligado, outros dispositivos acessam por{" "}
              <code>http://IP-deste-PC:11434</code>.
            </p>
          </section>

          <section className="mt-6">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h3 className="text-sm font-semibold text-neutral-900 dark:text-white">
                Configurar ferramentas
              </h3>
              {models.length > 0 && (
                <label className="flex items-center gap-2 text-xs text-neutral-500">
                  Modelo
                  <select
                    value={model}
                    onChange={(event) => setModel(event.target.value)}
                    className="rounded-lg border border-neutral-300 bg-transparent px-2 py-1 text-xs text-neutral-900 dark:border-neutral-700 dark:text-neutral-100"
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
            <ul className="mt-3 space-y-3">
              {toolSetups(model).map((setup) => (
                <li
                  key={setup.id}
                  className="rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-950"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <h4 className="text-sm font-semibold text-neutral-900 dark:text-white">
                        {setup.title}
                      </h4>
                      <p className="mt-0.5 text-xs text-neutral-500">
                        {setup.description}
                      </p>
                    </div>
                    <CopyButton text={setup.code} label={setup.title} />
                  </div>
                  <pre className="mt-3 overflow-x-auto rounded-xl bg-neutral-900 p-3 text-xs leading-5 text-neutral-100 dark:bg-black">
                    {setup.code}
                  </pre>
                </li>
              ))}
            </ul>
            <p className="mt-3 text-xs text-neutral-500">
              Os comandos <code>ollama launch</code> rodam no terminal
              (PowerShell ou Prompt de Comando). Veja todas as integrações em{" "}
              <a
                href="/connect"
                className="font-medium text-violet-600 hover:underline dark:text-violet-300"
              >
                Apps e providers
              </a>
              .
            </p>
          </section>
        </div>
      </div>
    </SidebarLayout>
  );
}
