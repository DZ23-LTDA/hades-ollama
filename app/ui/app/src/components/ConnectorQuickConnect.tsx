import { useEffect, useState, type FormEvent } from "react";
import { openExternal } from "@/lib/openExternal";
import {
  type AgentConnectorCatalogEntry,
  listOAuthClients,
  removeConnector,
  saveOAuthClient,
  startConnectorOAuth,
} from "@/lib/agenticClient";
import {
  connectHint,
  connectWithKey,
  disconnectConnector,
  keyHelpURL,
} from "@/lib/connectorConnect";
import { generateCodeVerifier } from "@/lib/pkce";
import { safeHttpUrl } from "@/lib/safeUrl";

// ConnectorQuickConnect is shown inside a catalog card. Services that accept
// an API key connect right here; OAuth-only services say plainly what is
// missing instead of sending the user through other pages.
export function ConnectorQuickConnect({
  entry,
  connected,
  onChanged,
}: {
  entry: AgentConnectorCatalogEntry;
  connected: boolean;
  onChanged: () => void;
}) {
  const [key, setKey] = useState("");
  const [baseURL, setBaseURL] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = async (action: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      setKey("");
      onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  if (!entry.quick_connect) {
    if (entry.auth === "oauth") {
      return (
        <OAuthConnect
          entry={entry}
          connected={connected}
          onChanged={onChanged}
        />
      );
    }
    return (
      <div className="mt-4 rounded-xl bg-neutral-50 p-3 text-xs leading-5 text-neutral-600 dark:bg-neutral-900 dark:text-neutral-300">
        <p>{connectHint(entry.name, entry.auth)}</p>
        <p className="mt-2 text-[11px] text-neutral-500 dark:text-neutral-400">
          Para registrá-lo, use o botão “Criar Conector” no topo desta página,
          informando o endereço e a variável de ambiente com a credencial.
        </p>
      </div>
    );
  }

  const help = keyHelpURL(entry.id);
  return (
    <div className="mt-4 rounded-xl bg-neutral-50 p-3 text-xs leading-5 text-neutral-600 dark:bg-neutral-900 dark:text-neutral-300">
      {connected ? (
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span>
            Conectado em modo somente leitura, restrito às rotas GET permitidas
            para este serviço. Para escrita, use o registro avançado com uma
            política explícita.
          </span>
          <button
            type="button"
            disabled={busy}
            onClick={() => void run(() => disconnectConnector(entry.id))}
            className="rounded-lg px-3 py-1.5 text-red-600 hover:bg-red-50 disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-950/30"
          >
            Desconectar
          </button>
        </div>
      ) : (
        <form
          className="flex flex-wrap items-center gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            void run(() =>
              connectWithKey(
                entry.id,
                key,
                entry.api_self_hosted ? baseURL : undefined,
              ),
            );
          }}
        >
          {entry.api_self_hosted && (
            <input
              type="url"
              value={baseURL}
              onChange={(event) => setBaseURL(event.target.value)}
              placeholder={`URL da sua instância ${entry.name} (https://...)`}
              aria-label={`URL da instância ${entry.name}`}
              className="w-full rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 outline-none dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
            />
          )}
          <input
            type="password"
            autoComplete="off"
            spellCheck={false}
            value={key}
            onChange={(event) => setKey(event.target.value)}
            placeholder={`Chave de API do ${entry.name}`}
            aria-label={`Chave de API do ${entry.name}`}
            className="min-w-0 flex-1 rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 outline-none dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
          />
          <button
            type="submit"
            disabled={
              busy ||
              !key.trim() ||
              (!!entry.api_self_hosted && !baseURL.trim())
            }
            className="rounded-lg bg-neutral-900 px-3 py-1.5 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900"
          >
            {busy ? "Conectando…" : "Conectar"}
          </button>
          {help && (
            <a
              href={help}
              target="_blank"
              rel="noreferrer"
              className="w-full text-violet-600 hover:underline dark:text-violet-300"
            >
              Onde pego a chave do {entry.name}?
            </a>
          )}
        </form>
      )}
      {error && (
        <p role="alert" className="mt-2 text-red-600 dark:text-red-400">
          {error}
        </p>
      )}
    </div>
  );
}

// connectorOAuthProviderName mirrors the server mapping: the Google services
// share one Google OAuth app; every other connector uses a provider named
// after itself.
function connectorOAuthProviderName(id: string): string {
  switch (id) {
    case "gmail":
    case "google-drive":
    case "google-calendar":
    case "google-analytics":
    case "google-ads":
    case "youtube":
    case "google-workspace":
      return "google";
    default:
      return id;
  }
}

// OAuthConnect configures the provider's OAuth app (client_id/secret) and then
// runs the one-click browser connect for a catalog connector.
function OAuthConnect({
  entry,
  connected,
  onChanged,
}: {
  entry: AgentConnectorCatalogEntry;
  connected: boolean;
  onChanged: () => void;
}) {
  const provider = connectorOAuthProviderName(entry.id);
  const [clientID, setClientID] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [configured, setConfigured] = useState(false);
  const [busy, setBusy] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    listOAuthClients()
      .then(({ providers }) => {
        if (!active) return;
        setConfigured(
          providers.some((item) => item.provider === provider && item.configured),
        );
      })
      .catch(() => {
        // Leave as not-configured; the user can still try to save the app.
      });
    return () => {
      active = false;
    };
  }, [provider]);

  const fail = (err: unknown) =>
    setError(err instanceof Error ? err.message : String(err));

  const saveClient = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await saveOAuthClient(provider, clientID.trim(), clientSecret.trim());
      setConfigured(true);
      setClientSecret("");
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  };

  const connect = async () => {
    setBusy(true);
    setError(null);
    try {
      const verifier = generateCodeVerifier();
      try {
        sessionStorage.setItem(
          `connector-oauth-verifier:${entry.id}`,
          verifier,
        );
      } catch {
        // Private mode may block sessionStorage; the flow still works because
        // the server keeps the verifier bound to the OAuth state.
      }
      const redirectURI = `${window.location.origin}/api/agent/v1/connectors/${encodeURIComponent(
        entry.id,
      )}/oauth/callback`;
      const { authorization_url } = await startConnectorOAuth(
        entry.id,
        redirectURI,
        verifier,
      );
      const target = safeHttpUrl(authorization_url);
      if (target === "#") {
        throw new Error("URL de autorização inválida recebida do servidor");
      }
      openExternal(target);
      setWaiting(true);
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  };

  const disconnect = async () => {
    setBusy(true);
    setError(null);
    try {
      await removeConnector(entry.id);
      setWaiting(false);
      onChanged();
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  };

  const refresh = () => {
    setWaiting(false);
    onChanged();
  };

  return (
    <div className="mt-4 rounded-xl bg-neutral-50 p-3 text-xs leading-5 text-neutral-600 dark:bg-neutral-900 dark:text-neutral-300">
      {connected ? (
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span>
            Conectado via OAuth, restrito às rotas GET permitidas para este
            serviço. Para escrita, use o registro avançado com uma política
            explícita.
          </span>
          <button
            type="button"
            disabled={busy}
            onClick={() => void disconnect()}
            className="rounded-lg px-3 py-1.5 text-red-600 hover:bg-red-50 disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-950/30"
          >
            Desconectar
          </button>
        </div>
      ) : (
        <>
          <p className="mb-2 font-semibold text-neutral-700 dark:text-neutral-200">
            Configurar app OAuth ({provider})
          </p>
          <p className="mb-2 text-[11px] text-neutral-500 dark:text-neutral-400">
            Registre um app OAuth no provedor com o redirect{" "}
            <code className="break-all">{`${window.location.origin}/api/agent/v1/connectors/${entry.id}/oauth/callback`}</code>{" "}
            e cole o ID do cliente e a chave secreta abaixo.
          </p>
          <form className="flex flex-col gap-2" onSubmit={(event) => void saveClient(event)}>
            <input
              type="text"
              autoComplete="off"
              spellCheck={false}
              value={clientID}
              onChange={(event) => setClientID(event.target.value)}
              placeholder="ID do cliente (client_id)"
              aria-label={`ID do cliente OAuth do ${provider}`}
              className="w-full rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 outline-none dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
            />
            <input
              type="password"
              autoComplete="off"
              spellCheck={false}
              value={clientSecret}
              onChange={(event) => setClientSecret(event.target.value)}
              placeholder="Chave secreta (client_secret)"
              aria-label={`Chave secreta OAuth do ${provider}`}
              className="w-full rounded-lg border border-neutral-300 bg-white px-3 py-1.5 text-xs text-neutral-900 outline-none dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100"
            />
            <div className="flex flex-wrap items-center gap-2">
              <button
                type="submit"
                disabled={busy || !clientID.trim() || !clientSecret.trim()}
                className="rounded-lg bg-neutral-900 px-3 py-1.5 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900"
              >
                {busy ? "Salvando…" : configured ? "Atualizar credenciais" : "Salvar"}
              </button>
              {configured && (
                <span className="text-[11px] font-semibold text-green-600 dark:text-green-400">
                  app configurado
                </span>
              )}
            </div>
          </form>
          {configured && (
            <div className="mt-3 border-t border-neutral-200 pt-3 dark:border-neutral-800">
              {waiting ? (
                <div className="flex flex-wrap items-center gap-2">
                  <span>
                    Aguardando aprovação no navegador… quando terminar, clique em
                    Atualizar.
                  </span>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={refresh}
                    className="rounded-lg bg-neutral-900 px-3 py-1.5 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900"
                  >
                    Atualizar
                  </button>
                </div>
              ) : (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => void connect()}
                  className="rounded-lg bg-neutral-900 px-3 py-1.5 text-xs text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900"
                >
                  {busy ? "Abrindo…" : `Conectar com ${entry.name}`}
                </button>
              )}
            </div>
          )}
        </>
      )}
      {error && (
        <p role="alert" className="mt-2 text-red-600 dark:text-red-400">
          {error}
        </p>
      )}
    </div>
  );
}
