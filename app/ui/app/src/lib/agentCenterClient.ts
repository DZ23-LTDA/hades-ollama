// Agent Center -- client da superficie HTTP.
//
// O CATALOGO NAO ESTA' LIGADO, E A RESPOSTA DIZ ISSO
//
// A rota devolve `catalog_available: false` e uma nota. O painel NAO pode apresentar as entradas do ledger
// como se fossem o catalogo:
//
//   *Uma lista que so' mostra o que ja' se instalou nao e' um catalogo: e' um historico.*
//
// O DRIFT VEM CLASSIFICADO DO SERVIDOR
//
// O painel nao recalcula o estado: mostra o que o servidor devolveu, e mostra a ACCAO que o servidor
// sugere. *Um painel que calcula o seu proprio estado mostra o que o programador achou que ia acontecer,
// nao o que aconteceu.*
//
// O DESTINO E' RELATIVO, E O SERVIDOR RECUSA O RESTO
//
// O painel manda o destino como o utilizador o escreveu; o servidor recusa absoluto e `..`. A fronteira nao
// e' repetida aqui.
import { agentFetch } from "@/lib/agenticClient";

export interface EntradaDeAgente {
  persona_id: string;
  version: string;
  target: string;
  destination: string;
  scope: string;
  source_hash: string;
  rendered_hash: string;
}

export interface ItemDeDrift {
  entry: EntradaDeAgente;
  state: string;
  action: string;
}

export interface ListaDeAgentes {
  organization_id?: string;
  agents?: EntradaDeAgente[];
  count?: number;
  catalog_available?: boolean;
  catalog_note?: string;
}

export interface RelatorioDeDrift {
  organization_id?: string;
  items?: ItemDeDrift[];
  summary?: Record<string, number>;
}

export interface ResultadoDeInstalacao {
  organization_id?: string;
  entry?: EntradaDeAgente;
  created?: boolean;
  bytes?: number;
}

/** Sem ledger configurado o servidor responde 503 -- e nao e' "nenhum agente". */
export class CentroAusenteError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "CentroAusenteError";
  }
}

export class InstalacaoRecusadaError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "InstalacaoRecusadaError";
  }
}

export class ConflitoDeInstalacaoError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "ConflitoDeInstalacaoError";
  }
}

function traduzir(causa: unknown): never {
  const m = causa instanceof Error ? causa.message : String(causa);
  if (/503|not configured|indispon[ií]vel/i.test(m)) throw new CentroAusenteError(m);
  if (/409|conflict/i.test(m)) throw new ConflitoDeInstalacaoError(m);
  throw new InstalacaoRecusadaError(m);
}

/** O destino tem de ser relativo e sem `..`. O servidor recusa; aqui so' se avisa antes. */
export function validarDestino(destino: string): string | null {
  const d = (destino ?? "").trim();
  if (d === "") return "falta o destino";
  if (d.startsWith("/") || /^[A-Za-z]:[\\/]/.test(d)) return "o destino tem de ser relativo ao workspace";
  if (d.split(/[\\/]/).includes("..")) return "o destino não pode sair do workspace (`..`)";
  return null;
}

/** A nota do servidor sobre o catalogo, dita sem inventar. */
export function descreverCatalogo(lista: ListaDeAgentes | null): string {
  if (!lista) return "sem resposta";
  if (lista.catalog_available === true) return "catálogo ligado";
  return lista.catalog_note ?? "o catálogo não está ligado: estas são as instalações registadas";
}

/** O estado e a accao, como o SERVIDOR os deu. */
export function descreverDrift(item: ItemDeDrift): string {
  const estado = (item.state ?? "").trim();
  const accao = (item.action ?? "").trim();
  if (estado === "") return "estado não declarado pelo servidor";
  return `${estado} · ${accao}`;
}

/** Um estado por linha, e os zeros nao desaparecem. */
export function resumoLegivel(summary: Record<string, number> | undefined): string {
  if (!summary) return "sem resumo";
  const estados = ["CURRENT", "OUTDATED", "LOCALLY_MODIFIED", "REMOVED", "FOREIGN_UNMANAGED", "CONFLICTED", "BROKEN_DEPENDENCY"];
  return estados.map((e) => `${e} ${summary[e] ?? 0}`).join(" · ");
}

export async function listarAgentes(): Promise<ListaDeAgentes> {
  try {
    return await agentFetch<ListaDeAgentes>("/api/agent/v1/agents");
  } catch (causa) {
    traduzir(causa);
  }
}

export async function lerDrift(): Promise<RelatorioDeDrift> {
  try {
    return await agentFetch<RelatorioDeDrift>("/api/agent/v1/agents/drift");
  } catch (causa) {
    traduzir(causa);
  }
}

export async function instalarAgente(
  persona: Record<string, unknown>,
  target: string,
  destination: string,
): Promise<ResultadoDeInstalacao> {
  const problema = validarDestino(destination);
  if (problema) throw new InstalacaoRecusadaError(problema);
  try {
    return await agentFetch<ResultadoDeInstalacao>("/api/agent/v1/agents/install", {
      method: "POST",
      body: JSON.stringify({ persona, target, destination }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
