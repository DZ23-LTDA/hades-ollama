import { useCallback, useEffect, useState } from "react";
import {
  ShieldCheckIcon,
  PhoneIcon,
  TrashIcon,
  PlusIcon,
  ArrowPathIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
} from "@heroicons/react/24/outline";
import {
  getWhatsAppStatus,
  getWhatsAppAllowlist,
  setWhatsAppContactPolicy,
  removeWhatsAppContactPolicy,
  setWhatsAppActiveBackend,
  configureWhatsAppEvolution,
  type WhatsAppStatusSummary,
  type WhatsAppContactPolicy,
} from "@/lib/agenticClient";

// translateStatus converte os enums crus do backend (ex.: NOT_CONFIGURED)
// para rótulos em português do Brasil, evitando mostrar siglas internas na UI.
function translateStatus(raw?: string): string {
  switch ((raw ?? "").toUpperCase()) {
    case "PASS":
    case "OK":
    case "CONNECTED":
      return "Conectado";
    case "NOT_CONFIGURED":
    case "":
      return "Não configurado";
    case "FAIL":
    case "ERROR":
      return "Falha";
    case "PENDING":
    case "CONNECTING":
      return "Conectando";
    default:
      return raw ?? "Não configurado";
  }
}

export function WhatsAppGatewayPanel() {
  const [status, setStatus] = useState<WhatsAppStatusSummary | null>(null);
  const [allowlist, setAllowlist] = useState<WhatsAppContactPolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [newPhone, setNewPhone] = useState("");
  const [newName, setNewName] = useState("");
  const [newRole, setNewRole] = useState<"owner" | "operator" | "viewer">("owner");
  const [saving, setSaving] = useState(false);
  // Credenciais do Evolution API (auto-hospedado). O servidor guarda com 0600 e
  // nunca devolve os segredos; por isso começam vazios e só são enviados ao salvar.
  const [evoBaseURL, setEvoBaseURL] = useState("");
  const [evoApiKey, setEvoApiKey] = useState("");
  const [evoInstance, setEvoInstance] = useState("");
  const [evoWebhookSecret, setEvoWebhookSecret] = useState("");
  const [evoSaving, setEvoSaving] = useState(false);
  const [evoSaved, setEvoSaved] = useState(false);

  const handleSaveEvolution = async () => {
    setEvoSaving(true);
    setEvoSaved(false);
    setError(null);
    try {
      const updated = await configureWhatsAppEvolution({
        base_url: evoBaseURL.trim(),
        api_key: evoApiKey.trim(),
        instance: evoInstance.trim(),
        webhook_secret: evoWebhookSecret.trim(),
      });
      setStatus(updated);
      setEvoSaved(true);
      // Limpa os campos de segredo da memória da UI depois de salvar.
      setEvoApiKey("");
      setEvoWebhookSecret("");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setEvoSaving(false);
    }
  };

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [st, al] = await Promise.all([
        getWhatsAppStatus().catch(() => null),
        getWhatsAppAllowlist().catch(() => ({ count: 0, items: [] })),
      ]);
      if (st) setStatus(st);
      if (al && al.items) setAllowlist(al.items);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleSwitchBackend = async (backend: "evolution_api" | "cloud_api") => {
    try {
      const updated = await setWhatsAppActiveBackend(backend);
      setStatus(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const handleAddContact = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPhone.trim() || !newName.trim()) return;
    setSaving(true);
    try {
      await setWhatsAppContactPolicy({
        phone_number: newPhone.trim(),
        name: newName.trim(),
        role: newRole,
        allowed: true,
      });
      setNewPhone("");
      setNewName("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const handleRemoveContact = async (phone: string) => {
    try {
      await removeWhatsAppContactPolicy(phone);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const isConfigured = status?.gate_status === "PASS";

  return (
    <div className="space-y-6">
      {/* Header Card */}
      <div className="rounded-2xl border border-neutral-200 bg-white p-6 shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-emerald-50 text-emerald-600 dark:bg-emerald-950/50">
              <PhoneIcon className="h-6 w-6" />
            </div>
            <div>
              <h2 className="text-base font-bold text-neutral-900 dark:text-white">
                WhatsApp Gateway — Controle Remoto do Agente
              </h2>
              <p className="text-xs text-neutral-500">
                Dispare missões, consulte progresso e aprove ações sensíveis pelo celular.
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span
              className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-semibold ${
                isConfigured
                  ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300"
                  : "bg-amber-50 text-amber-700 dark:bg-amber-950/50 dark:text-amber-300"
              }`}
            >
              {isConfigured ? (
                <>
                  <CheckCircleIcon className="h-4 w-4" />
                  Conectado (PASS)
                </>
              ) : (
                <>
                  <ExclamationCircleIcon className="h-4 w-4" />
                  {translateStatus(status?.gate_status)}
                </>
              )}
            </span>
            <button
              type="button"
              onClick={() => void load()}
              disabled={loading}
              className="rounded-xl border border-neutral-200 p-2 text-neutral-600 hover:bg-neutral-50 dark:border-neutral-800 dark:text-neutral-400 dark:hover:bg-neutral-900"
              aria-label="Atualizar status"
            >
              <ArrowPathIcon className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            </button>
          </div>
        </div>

        {error && (
          <div className="mt-4 rounded-xl border border-red-200 bg-red-50 p-3 text-xs text-red-700 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-300">
            {error}
          </div>
        )}

        {/* Backends Selector */}
        <div className="mt-6 grid gap-4 sm:grid-cols-2">
          <button
            type="button"
            aria-pressed={status?.active_backend === "evolution_api"}
            onClick={() => void handleSwitchBackend("evolution_api")}
            className={`cursor-pointer rounded-xl border p-4 text-left transition ${
              status?.active_backend === "evolution_api"
                ? "border-emerald-500 bg-emerald-50/30 dark:border-emerald-500/50 dark:bg-emerald-950/20"
                : "border-neutral-200 bg-white hover:border-neutral-300 dark:border-neutral-800 dark:bg-neutral-900"
            }`}
          >
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-bold text-neutral-900 dark:text-white">
                Evolution API (Self-Hosted)
              </h3>
              {status?.active_backend === "evolution_api" && (
                <span className="text-[10px] font-semibold text-emerald-600">ATIVO</span>
              )}
            </div>
            <p className="mt-1 text-[11px] text-neutral-500">
              Gateway aberto de alta velocidade para instâncias próprias.
            </p>
            <div className="mt-3 text-[10px] text-neutral-400">
              Status: {translateStatus(status?.adapters?.evolution_api?.status)}
            </div>
          </button>

          <button
            type="button"
            aria-pressed={status?.active_backend === "cloud_api"}
            onClick={() => void handleSwitchBackend("cloud_api")}
            className={`cursor-pointer rounded-xl border p-4 text-left transition ${
              status?.active_backend === "cloud_api"
                ? "border-emerald-500 bg-emerald-50/30 dark:border-emerald-500/50 dark:bg-emerald-950/20"
                : "border-neutral-200 bg-white hover:border-neutral-300 dark:border-neutral-800 dark:bg-neutral-900"
            }`}
          >
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-bold text-neutral-900 dark:text-white">
                WhatsApp Business Cloud API (Meta)
              </h3>
              {status?.active_backend === "cloud_api" && (
                <span className="text-[10px] font-semibold text-emerald-600">ATIVO</span>
              )}
            </div>
            <p className="mt-1 text-[11px] text-neutral-500">
              Endpoint oficial Meta Graph API para contas corporativas.
            </p>
            <div className="mt-3 text-[10px] text-neutral-400">
              Status: {translateStatus(status?.adapters?.cloud_api?.status)}
            </div>
          </button>
        </div>

        {/* Configuração do Evolution API (credenciais) */}
        <div className="mt-6 rounded-xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
          <h3 className="text-xs font-bold text-neutral-900 dark:text-white">
            Conectar Evolution API
          </h3>
          <p className="mt-1 text-[11px] text-neutral-500">
            Informe o endereço e a chave do seu servidor Evolution API. As credenciais são guardadas com segurança no servidor local (0600) e nunca são exibidas de volta.
          </p>
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-1 text-[11px] text-neutral-500">
              URL do servidor
              <input type="url" value={evoBaseURL} onChange={(e) => setEvoBaseURL(e.target.value)} placeholder="https://evolution.seudominio.com" className="h-9 rounded-lg border border-neutral-300 bg-transparent px-2.5 text-sm outline-none dark:border-neutral-700" />
            </label>
            <label className="flex flex-col gap-1 text-[11px] text-neutral-500">
              Instância
              <input type="text" value={evoInstance} onChange={(e) => setEvoInstance(e.target.value)} placeholder="nome-da-instancia" className="h-9 rounded-lg border border-neutral-300 bg-transparent px-2.5 text-sm outline-none dark:border-neutral-700" />
            </label>
            <label className="flex flex-col gap-1 text-[11px] text-neutral-500">
              API Key
              <input type="password" value={evoApiKey} onChange={(e) => setEvoApiKey(e.target.value)} autoComplete="off" placeholder="chave da API" className="h-9 rounded-lg border border-neutral-300 bg-transparent px-2.5 text-sm outline-none dark:border-neutral-700" />
            </label>
            <label className="flex flex-col gap-1 text-[11px] text-neutral-500">
              Segredo do webhook
              <input type="password" value={evoWebhookSecret} onChange={(e) => setEvoWebhookSecret(e.target.value)} autoComplete="off" placeholder="segredo para validar o webhook" className="h-9 rounded-lg border border-neutral-300 bg-transparent px-2.5 text-sm outline-none dark:border-neutral-700" />
            </label>
          </div>
          <div className="mt-3 flex items-center gap-3">
            <button type="button" onClick={() => void handleSaveEvolution()} disabled={evoSaving || !evoBaseURL.trim() || !evoApiKey.trim() || !evoInstance.trim()} className="h-9 rounded-lg bg-emerald-600 px-4 text-xs font-medium text-white disabled:opacity-40 hover:bg-emerald-700">
              {evoSaving ? "Salvando…" : "Salvar e conectar"}
            </button>
            {evoSaved && <span className="text-[11px] text-emerald-600">Credenciais salvas. Verificando status…</span>}
          </div>
          <p className="mt-2 text-[10px] text-neutral-400">
            Dica: configure o webhook do seu Evolution API para apontar para o endpoint do Hades (/api/agent/v1/whatsapp/webhook) usando o mesmo segredo.
          </p>
        </div>

        {/* Command Bridge & Security Badges — só quando o gateway está
            configurado; caso contrário seria um selo falso de recursos ativos. */}
        {isConfigured ? (
          <div className="mt-6 grid gap-3 sm:grid-cols-3">
            <div className="rounded-xl border border-neutral-100 bg-neutral-50 p-3 dark:border-neutral-800/80 dark:bg-neutral-900/50">
              <span className="text-[10px] font-bold uppercase tracking-wider text-neutral-400">
                Command Bridge
              </span>
              <p className="mt-1 text-xs font-medium text-neutral-800 dark:text-neutral-200">
                /goal, missao, health, projetos
              </p>
              <p className="text-[10px] text-neutral-500">Execução real no runtime local</p>
            </div>
            <div className="rounded-xl border border-neutral-100 bg-neutral-50 p-3 dark:border-neutral-800/80 dark:bg-neutral-900/50">
              <span className="text-[10px] font-bold uppercase tracking-wider text-neutral-400">
                Segurança e aprovação humana
              </span>
              <p className="mt-1 text-xs font-medium text-neutral-800 dark:text-neutral-200">
                Aprovação via celular (APROVAR/REJEITAR)
              </p>
              <p className="text-[10px] text-neutral-500">Ações sensíveis bloqueadas até autorização</p>
            </div>
            <div className="rounded-xl border border-neutral-100 bg-neutral-50 p-3 dark:border-neutral-800/80 dark:bg-neutral-900/50">
              <span className="text-[10px] font-bold uppercase tracking-wider text-neutral-400">
                Idempotência & DLQ
              </span>
              <p className="mt-1 text-xs font-medium text-neutral-800 dark:text-neutral-200">
                Filtro fromMe + Dedupe TTL 24h
              </p>
              <p className="text-[10px] text-neutral-500">Zero loops de resposta e DLQ ativa</p>
            </div>
          </div>
        ) : (
          <p className="mt-6 rounded-xl border border-amber-200 bg-amber-50 p-3 text-[11px] text-amber-700 dark:border-amber-900/50 dark:bg-amber-950/20 dark:text-amber-300">
            Configure um backend acima (Evolution API ou Cloud API da Meta) para
            ativar o Command Bridge, a aprovação humana e a proteção contra loops
            (DLQ). Esses recursos ficam disponíveis apenas após a conexão.
          </p>
        )}
      </div>

      {/* Allowlist Section */}
      <div className="rounded-2xl border border-neutral-200 bg-white p-6 shadow-sm dark:border-neutral-800 dark:bg-neutral-950">
        <div className="flex items-center gap-2">
          <ShieldCheckIcon className="h-5 w-5 text-violet-600" />
          <h3 className="text-sm font-bold text-neutral-900 dark:text-white">
            Allowlist de Donos e Contatos Autorizados
          </h3>
        </div>
        <p className="mt-1 text-xs text-neutral-500">
          Apenas os números cadastrados abaixo têm permissão para interagir com o agente. Contatos fora da lista são bloqueados imediatamente.
        </p>

        {/* Contact List */}
        <div className="mt-4 divide-y divide-neutral-100 overflow-hidden rounded-xl border border-neutral-200 dark:divide-neutral-800 dark:border-neutral-800">
          {allowlist.length === 0 ? (
            <div className="p-4 text-center text-xs text-neutral-500">
              Nenhum contato cadastrado na allowlist. Adicione o seu número abaixo para começar.
            </div>
          ) : (
            allowlist.map((contact) => (
              <div
                key={contact.phone_number}
                className="flex items-center justify-between p-3.5 hover:bg-neutral-50/50 dark:hover:bg-neutral-900/50"
              >
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-semibold text-neutral-900 dark:text-white">
                      {contact.name}
                    </span>
                    <span
                      className={`rounded-full px-2 py-0.5 text-[10px] font-bold uppercase ${
                        contact.role === "owner"
                          ? "bg-purple-100 text-purple-700 dark:bg-purple-950/60 dark:text-purple-300"
                          : "bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400"
                      }`}
                    >
                      {contact.role}
                    </span>
                  </div>
                  <span className="text-[11px] text-neutral-500">
                    +{contact.phone_number}
                  </span>
                </div>
                <button
                  type="button"
                  onClick={() => void handleRemoveContact(contact.phone_number)}
                  className="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-red-600 dark:hover:bg-neutral-800"
                  aria-label="Remover contato"
                >
                  <TrashIcon className="h-4 w-4" />
                </button>
              </div>
            ))
          )}
        </div>

        {/* Add Contact Form */}
        <form onSubmit={handleAddContact} className="mt-4 flex flex-wrap items-center gap-2">
          <input
            type="text"
            placeholder="Telefone (ex: 5511999999999)"
            value={newPhone}
            onChange={(e) => setNewPhone(e.target.value)}
            className="flex-1 min-w-[180px] rounded-xl border border-neutral-300 bg-transparent px-3 py-2 text-xs text-neutral-900 outline-none focus:border-neutral-500 dark:border-neutral-700 dark:text-white"
          />
          <input
            type="text"
            placeholder="Nome do contato"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            className="flex-1 min-w-[140px] rounded-xl border border-neutral-300 bg-transparent px-3 py-2 text-xs text-neutral-900 outline-none focus:border-neutral-500 dark:border-neutral-700 dark:text-white"
          />
          <select
            value={newRole}
            onChange={(e) => setNewRole(e.target.value as "owner" | "operator" | "viewer")}
            className="rounded-xl border border-neutral-300 bg-white px-3 py-2 text-xs text-neutral-900 outline-none dark:border-neutral-700 dark:bg-neutral-900 dark:text-white"
          >
            <option value="owner">Dono (Owner - Acesso Total)</option>
            <option value="operator">Operador (Executa missões)</option>
            <option value="viewer">Visualizador (Somente leitura)</option>
          </select>
          <button
            type="submit"
            disabled={saving || !newPhone.trim() || !newName.trim()}
            className="inline-flex items-center gap-1 rounded-xl bg-neutral-900 px-4 py-2 text-xs font-semibold text-white transition hover:bg-neutral-800 disabled:opacity-40 dark:bg-white dark:text-neutral-900"
          >
            <PlusIcon className="h-4 w-4" />
            Adicionar
          </button>
        </form>
      </div>
    </div>
  );
}
