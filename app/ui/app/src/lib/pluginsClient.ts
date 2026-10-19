// Plugins -- catalogo, estado e o formato que o runtime aceita.
//
// `plugins` tinha 9 rotas. TRES sao leitura:
//
//   GET /plugins/format -> {schema, format_version, kinds, runtimes}     ESTATICO, sem org
//   GET /plugins        -> {organization_id, plugins: PluginInstallation[]}
//   GET /plugins/status -> {organization_id, plugins: PluginStatus[]}
//
// As outras 6 sao escritas: execute, format/install, install, promote, rollback, bundle.
// Todas mudam estado ou correm codigo. **O painel nao as chama.**
//
// DUAS COISAS QUE `PluginStatus` NAO DIZ, E QUE O PAINEL NAO PODE INVENTAR:
//
//  1. **NAO HA' `granted_scopes`.** `PluginInstallation` tem os dois --
//     `requested_scopes` E `granted_scopes` -- mas `PluginStatus` so' traz os PEDIDOS.
//
//     *Um ecra que mostre `requested_scopes` debaixo do rotulo "permissoes" esta' a
//     dizer que o plugin TEM o que ele PEDIU. Nao e' a mesma lista.*
//
//     O painel mostra os pedidos como pedidos, e diz que o estado nao informa o que foi
//     concedido.
//
//  2. **`ready_to_use` E' O VEREDITO, E `reasons` E' A EXPLICACAO.** Um plugin com
//     `ready_to_use: false` sem mostrar as `reasons` e' uma linha vermelha sem causa.
//
// E O FORMATO NAO E' INVENTADO:
//
//   GET /plugins/format devolve os 5 `kind` que o validador aceita e os 5 `runtime` que o
//   executor suporta. O proprio handler diz porque:
//
//       // O runtime de execucao suportado. *Declarado aqui para que o autor do plugin
//       // saiba o que o runtime faz antes de o escrever.*
//
//   O painel mostra esta lista como o CONTRATO ACEITO, nao como sugestao.
import { agentFetch } from "@/lib/agenticClient";

export interface FormatoDePlugin {
  schema: string;
  format_version: string;
  kinds: string[];
  runtimes: string[];
}

export interface ExecucaoDePlugin {
  at?: string;
  ok?: boolean;
  duration_ms?: number;
  error?: string;
}

export interface EstadoDePlugin {
  plugin_id: string;
  version: string;
  kind: string;
  state: string;
  enabled: boolean;
  trusted: boolean;
  content_verified: boolean;
  capabilities: string[] | null;
  /** O que o plugin PEDIU. O estado NAO diz o que lhe foi CONCEDIDO. */
  requested_scopes?: string[];
  reasons?: string[];
  last_execution?: ExecucaoDePlugin;
  ready_to_use: boolean;
}

export interface ManifestoDePlugin {
  id: string;
  version: string;
  name: string;
  kind: string;
  description?: string;
  scopes?: string[];
  dependencies?: string[];
  license?: string;
  homepage?: string;
  content_sha256?: string;
}

export interface InstalacaoDePlugin {
  organization_id: string;
  manifest: ManifestoDePlugin;
  digest: string;
  trusted: boolean;
  enabled: boolean;
  requested_scopes?: string[];
  granted_scopes?: string[];
  installed_at: string;
  updated_at: string;
  content_sha256?: string;
  content_verified: boolean;
}

export interface RespostaDeFormato {
  schema: string;
  format_version: string;
  kinds: string[] | null;
  runtimes: string[] | null;
}

export interface RespostaDePlugins {
  organization_id: string;
  plugins: InstalacaoDePlugin[] | null;
}

export interface RespostaDeEstados {
  organization_id: string;
  plugins: EstadoDePlugin[] | null;
}

/** O registro de plugins nao esta' ligado neste runtime. */
export class PluginsAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "PluginsAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/registro de plugins|n[aã]o est[aá] ligado|n[aã]o est[aá] configurad|503/i.test(mensagem)) {
    throw new PluginsAusenteError(mensagem);
  }
  throw causa;
}

/**
 * Descreve os scopes PEDIDOS, sem os apresentar como concedidos.
 *
 * `PluginStatus` nao traz `granted_scopes`; so' `PluginInstallation` traz.
 */
export function descreverScopesPedidos(pedidos: string[] | undefined): string {
  const lista = (pedidos ?? []).filter((s) => (s ?? "").trim() !== "");
  if (lista.length === 0) return "nenhum scope pedido";
  return `pediu: ${lista.join(", ")} — o estado não informa o que foi concedido`;
}

/**
 * Compara pedidos com concedidos, e nomeia o que foi pedido e NAO concedido.
 *
 * So' funciona com `PluginInstallation`; com `PluginStatus` nao ha' como saber.
 */
export function descreverScopes(
  pedidos: string[] | undefined,
  concedidos: string[] | undefined,
): string {
  const p = (pedidos ?? []).filter((s) => (s ?? "").trim() !== "");
  const c = (concedidos ?? []).filter((s) => (s ?? "").trim() !== "");
  if (p.length === 0 && c.length === 0) return "sem scopes";
  const faltam = p.filter((s) => !c.includes(s));
  if (faltam.length === 0) {
    return `${c.length} scope(s) pedido(s) e concedido(s)`;
  }
  return `${p.length} pedido(s), ${c.length} concedido(s) — NÃO concedido(s): ${faltam.join(", ")}`;
}

/**
 * Diz se o plugin esta' pronto a usar, e porque nao quando nao esta'.
 *
 * `ready_to_use: false` sem as razoes e' uma linha vermelha sem causa.
 */
export function descreverProntidao(estado: EstadoDePlugin): string {
  if (estado.ready_to_use) return "pronto a usar";
  const razoes = (estado.reasons ?? []).filter((r) => (r ?? "").trim() !== "");
  if (razoes.length === 0) {
    return `NÃO está pronto a usar — motivo não declarado pelo servidor (estado «${estado.state}»)`;
  }
  return `NÃO está pronto a usar — ${razoes.join("; ")}`;
}

/**
 * Descreve confianca e verificacao de conteudo -- que sao duas coisas diferentes.
 *
 * `trusted` e' o que o registo ACEITA; `content_verified` e' o que o conteudo PROVA.
 */
export function descreverIntegridade(estado: {
  trusted: boolean;
  content_verified: boolean;
}): string {
  if (estado.trusted && estado.content_verified) {
    return "confiável e conteúdo verificado";
  }
  if (estado.trusted && !estado.content_verified) {
    return "confiável, mas o conteúdo NÃO foi verificado";
  }
  if (!estado.trusted && estado.content_verified) {
    return "conteúdo verificado, mas o plugin NÃO é confiável";
  }
  return "NÃO confiável e conteúdo NÃO verificado";
}

/** Descreve o formato aceito, sem o apresentar como sugestao. */
export function descreverFormato(formato: RespostaDeFormato | undefined): string {
  if (!formato) return "sem informação de formato";
  const kinds = (formato.kinds ?? []).join(", ");
  const runtimes = (formato.runtimes ?? []).join(", ");
  return (
    `schema ${formato.schema} v${formato.format_version} — ` +
    `kinds aceitos: ${kinds}; runtimes suportados: ${runtimes}`
  );
}

export async function obterFormato(): Promise<RespostaDeFormato> {
  try {
    return await agentFetch<RespostaDeFormato>("/api/agent/v1/plugins/format");
  } catch (causa) {
    traduzir(causa);
  }
}

export async function listarPlugins(): Promise<RespostaDePlugins> {
  try {
    return await agentFetch<RespostaDePlugins>("/api/agent/v1/plugins");
  } catch (causa) {
    traduzir(causa);
  }
}

export async function listarEstados(): Promise<RespostaDeEstados> {
  try {
    return await agentFetch<RespostaDeEstados>("/api/agent/v1/plugins/status");
  } catch (causa) {
    traduzir(causa);
  }
}
