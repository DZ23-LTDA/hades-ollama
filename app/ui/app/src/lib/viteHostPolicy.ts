const LOOPBACK_HOSTS = new Set(["127.0.0.1", "::1", "localhost"]);

export type ViteHostEnv = {
  HADES_UI_HOST?: string;
  HADES_UI_ALLOW_LAN?: string;
  HADES_UI_AUTH_REQUIRED?: string;
};

function defaultEnvironment(): ViteHostEnv {
  const runtime = globalThis as typeof globalThis & {
    process?: { env?: ViteHostEnv };
  };
  return runtime.process?.env ?? {};
}

/**
 * UI binds to loopback by default. LAN exposure is an explicit, authenticated opt-in.
 */
export function resolveViteHost(env: ViteHostEnv = defaultEnvironment()): string {
  const host = env.HADES_UI_HOST?.trim() || "127.0.0.1";
  if (LOOPBACK_HOSTS.has(host)) return host;

  if (env.HADES_UI_ALLOW_LAN !== "true") {
    throw new Error(
      `Refusing non-loopback UI host ${host}; set HADES_UI_ALLOW_LAN=true for an explicit LAN opt-in`,
    );
  }
  if (env.HADES_UI_AUTH_REQUIRED !== "true") {
    throw new Error(
      "Refusing LAN UI exposure without HADES_UI_AUTH_REQUIRED=true",
    );
  }
  return host;
}
