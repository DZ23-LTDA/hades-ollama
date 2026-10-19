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
  catalog_version?: string;
  catalog_count?: number;
}

export interface RelatorioDeDrift {
  organization_id?: string;
  items?: ItemDeDrift[];
  summary?: Record<string, number>;
}

/** Uma persona do catalogo embutido. `evaluation` vazio significa NAO AVALIADO. */
export interface PersonaDoCatalogo {
  id: string;
  name: string;
  version: string;
  description?: string;
  division?: string;
  category?: string;
  source?: string;
  license?: string;
  capabilities?: string[];
  scopes?: string[];
  requires_approval?: string[];
  forbidden?: string[];
  /** Vazio = nao avaliado. Nunca leia como "aprovado". */
  evaluation?: string;
}

export interface Catalogo {
  version?: string;
  count?: number;
  personas?: PersonaDoCatalogo[];
  evaluations_available?: boolean;
  note?: string;
}

/** Uma suite de avaliacao: o que a persona sera' medida a fazer. */
export interface SuiteDeAvaliacao {
  version: string;
  cases: { id: string; kind: string; prompt: string; expect: string; forbid?: string[] }[];
}

export interface EstadoDeAvaliacao {
  state: string;
  description: string;
  can_go_to: string[];
  terminal: boolean;
}

export interface Avaliacao {
  suites?: Record<string, SuiteDeAvaliacao>;
  divisions?: string[];
  states?: EstadoDeAvaliacao[];
  inference_available?: boolean;
  note?: string;
}

/** Um papel dentro de um preset de equipa. */
export interface PapelDeEquipa {
  persona_id: string;
  role: string;
  responsibility: string;
  deliverable: string;
  depends_on?: string[];
  why: string;
  missing_tools?: string[];
  risk?: string;
  requires_approval?: string[];
}

export interface PresetDeEquipa {
  id: string;
  name: string;
  version: string;
  description?: string;
  division: string;
  source: string;
  license: string;
  roles: PapelDeEquipa[];
  max_agents: number;
  max_parallel: number;
  stop_condition: string;
  /** Comeca `false`. So' uma missao representativa o promove. */
  functional: boolean;
  evidence_of_functionality?: string;
}

export interface ValidacaoDeEquipa {
  ok: boolean;
  problems?: string[];
  missing_personas?: string[];
  missing_tools?: string[];
  cycles?: string[][];
}

export interface ItemDeEquipa {
  team: PresetDeEquipa;
  validation: ValidacaoDeEquipa;
  covered_divisions: string[];
  missing_divisions: string[];
  summary: string;
}

export interface Equipas {
  version?: string;
  count?: number;
  teams?: ItemDeEquipa[];
  functional_count?: number;
  note?: string;
}

export interface AchadoDeQuarentena {
  field: string;
  pattern: string;
  severity: "BLOCK" | "WARN";
  excerpt: string;
  why: string;
}

export interface RelatorioDeQuarentena {
  ok: boolean;
  blocked: boolean;
  needs_review: boolean;
  findings?: AchadoDeQuarentena[];
}

export interface ResultadoDeImportacao {
  state?: string;
  report?: RelatorioDeQuarentena;
  installed?: boolean;
  note?: string;
  persona?: { id?: string; name?: string; version?: string; source?: string; license?: string };
}

export interface EntradaDeQuarentena {
  id: string;
  scope: string;
  persona_id: string;
  state: string;
  report?: RelatorioDeQuarentena;
  created_at: string;
  reviewed_by?: string;
  reviewed_at?: string;
}

export interface ListaDeQuarentena {
  count?: number;
  entries?: EntradaDeQuarentena[];
}

export interface BackupDePersonas {
  id: string;
  scope: string;
  created_at: string;
  entries: EntradaDeAgente[];
  files: Record<string, string>;
}

export interface ListaDeBackups {
  count?: number;
  backups?: string[];
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

/**
 * Reconhece o erro do transporte por FORMA, nao por identidade de classe.
 *
 * PORQUE `instanceof` NAO SERVE AQUI (E-1381)
 *
 * Tentei `causa instanceof ErroDaApi` e regrediu 2 testes. A causa: um teste do painel mocka
 * `@/lib/agenticClient` devolvendo so' `{ agentFetch }`. Com o mock, `ErroDaApi` e' `undefined` -- e
 * `causa instanceof undefined` LANCA em vez de devolver `false`. O painel caia no ramo generico, punha
 * `estado = "erro"`, e os elementos `agent-center-ausente`/`agent-center-conflito` nem existiam.
 *
 *	*Um painel que le uma chave que ninguem emite nao mostra o vazio: mostra o fallback.* -- e aqui o
 *	fallback era um erro generico onde devia estar o motivo.
 *
 * Uma verificacao estrutural funciona com ou sem mock, com ou sem bundler a duplicar a classe.
 */
function ehErroDaApi(
  causa: unknown,
): causa is { status: number; message: string; campo(c: string): string | undefined } {
  if (typeof causa !== "object" || causa === null) return false;
  const c = causa as { status?: unknown; campo?: unknown; message?: unknown };
  return (
    typeof c.status === "number" &&
    typeof c.campo === "function" &&
    typeof c.message === "string"
  );
}

function traduzir(causa: unknown): never {
  // O status e' a fonte de verdade; a mensagem e' o fallback.
  if (ehErroDaApi(causa)) {
    const motivo = causa.campo("reason") ?? causa.campo("error") ?? causa.message;
    if (causa.status === 503) throw new CentroAusenteError(motivo);
    if (causa.status === 409) throw new ConflitoDeInstalacaoError(motivo);
  }
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
  if (lista.catalog_available === true) {
    const n = lista.catalog_count ?? 0;
    return `catálogo ligado · versão ${lista.catalog_version ?? "?"} · ${n} persona(s) disponíveis para instalar`;
  }
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

/**
 * Quantas personas do catalogo estao AVALIADAS.
 *
 * Devolve `avaliadas` e `total` separados porque *um numero sem a sua qualidade ao lado e' um numero que se
 * cita mal*: "6 personas" e "6 personas, 0 avaliadas" sao a mesma contagem e duas afirmacoes diferentes.
 */
export function coberturaDeAvaliacao(catalogo: Catalogo | null): { avaliadas: number; total: number } {
  const personas = catalogo?.personas ?? [];
  const avaliadas = personas.filter((p) => (p.evaluation ?? "").trim() !== "").length;
  return { avaliadas, total: personas.length };
}

export async function listarAgentes(): Promise<ListaDeAgentes> {
  try {
    return await agentFetch<ListaDeAgentes>("/api/agent/v1/agents");
  } catch (causa) {
    traduzir(causa);
  }
}

/** Le' so' o catalogo embutido. Nao toca no ledger: catalogo e historico sao coisas diferentes. */
export async function listarCatalogo(): Promise<Catalogo> {
  try {
    return await agentFetch<Catalogo>("/api/agent/v1/agents/catalog");
  } catch (causa) {
    traduzir(causa);
  }
}

/** Quantos casos tem a suite, por divisao. Um numero sem o tipo ao lado engana. */
export function contarCasos(suite: SuiteDeAvaliacao | undefined): { total: number; negativos: number } {
  const casos = suite?.cases ?? [];
  const negativos = casos.filter((c) => c.kind === "REFUSAL" || c.kind === "INJECTION").length;
  return { total: casos.length, negativos };
}

/** Le' as suites e a maquina de estados. NAO corre avaliacao. */
export async function lerAvaliacao(): Promise<Avaliacao> {
  try {
    return await agentFetch<Avaliacao>("/api/agent/v1/agents/evaluation");
  } catch (causa) {
    traduzir(causa);
  }
}

/** Le' os presets JA' VALIDADOS pelo servidor. O painel nao revalida: mostra o veredicto recebido. */
export async function lerEquipas(): Promise<Equipas> {
  try {
    return await agentFetch<Equipas>("/api/agent/v1/agents/teams");
  } catch (causa) {
    traduzir(causa);
  }
}

/**
 * Conta as equipas que se dizem funcionais e as que declararam ferramentas em falta.
 *
 * *Um numero sem a sua qualidade ao lado e' um numero que se cita mal*: "4 equipas" e "4 equipas, 0
 * funcionais, 3 com ferramentas em falta" sao a mesma contagem e duas afirmacoes diferentes.
 */
export function resumoDeEquipas(e: Equipas | null): { total: number; funcionais: number; comFalta: number; invalidas: number } {
  const itens = e?.teams ?? [];
  return {
    total: itens.length,
    funcionais: e?.functional_count ?? itens.filter((i) => i.team.functional).length,
    comFalta: itens.filter((i) => (i.validation.missing_tools ?? []).length > 0).length,
    invalidas: itens.filter((i) => !i.validation.ok).length,
  };
}

/** Guarda ledger E ficheiros. Idempotente: o mesmo estado devolve o mesmo id. */
export async function guardarBackup(): Promise<{ backup?: BackupDePersonas; files?: number; entries?: number }> {
  try {
    return await agentFetch("/api/agent/v1/agents/backup", { method: "POST", body: "{}" });
  } catch (causa) {
    traduzir(causa);
  }
}

export async function listarBackups(): Promise<ListaDeBackups> {
  try {
    return await agentFetch<ListaDeBackups>("/api/agent/v1/agents/backups");
  } catch (causa) {
    traduzir(causa);
  }
}

/**
 * Repoe um backup. O id e' OBRIGATORIO: nao ha' "restaurar o mais recente" por omissao.
 *
 * *O ultimo a escrever nao ganha por ser o ultimo, e o mais recente tambem nao.*
 */
export async function restaurarBackup(backupId: string): Promise<{ restored?: string }> {
  const id = (backupId ?? "").trim();
  if (id === "") throw new InstalacaoRecusadaError("escolha um backup: não há «restaurar o mais recente»");
  try {
    return await agentFetch("/api/agent/v1/agents/restore", {
      method: "POST",
      body: JSON.stringify({ backup_id: id }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}

/**
 * Triagem de um manifesto. NAO instala nada -- escrever «importado» quando nada foi escrito seria o defeito.
 */
export async function triarImportacao(content: string): Promise<ResultadoDeImportacao> {
  const bruto = (content ?? "").trim();
  if (bruto === "") throw new InstalacaoRecusadaError("cole o manifesto antes de triar");
  try {
    return await agentFetch<ResultadoDeImportacao>("/api/agent/v1/agents/import", {
      method: "POST",
      body: JSON.stringify({ content: bruto }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}

/** Quantos achados de cada severidade, para o resumo nao esconder um BLOCK atras de WARNs. */
export function contarAchados(r: RelatorioDeQuarentena | null): { bloqueios: number; avisos: number } {
  const lista = r?.findings ?? [];
  return {
    bloqueios: lista.filter((f) => f.severity === "BLOCK").length,
    avisos: lista.filter((f) => f.severity !== "BLOCK").length,
  };
}

/** Valida, triagem e RETE. Escreve -- e o estado que faltava. */
export async function reterEmQuarentena(content: string): Promise<{ quarantined?: boolean; state?: string; record?: EntradaDeQuarentena; note?: string }> {
  const bruto = (content ?? "").trim();
  if (bruto === "") throw new InstalacaoRecusadaError("cole o manifesto antes de reter");
  try {
    return await agentFetch("/api/agent/v1/agents/quarantine", { method: "POST", body: JSON.stringify({ content: bruto }) });
  } catch (causa) {
    traduzir(causa);
  }
}

export async function listarQuarentena(): Promise<ListaDeQuarentena> {
  try {
    return await agentFetch<ListaDeQuarentena>("/api/agent/v1/agents/quarantine");
  } catch (causa) {
    traduzir(causa);
  }
}

/**
 * Liberta para `SECURITY_REVIEWED`. NAO promove a producao, e o revisor e obrigatorio.
 *
 * *Uma revisao de seguranca que aprova para producao nao e uma revisao: e um atalho.*
 */
export async function libertarDaQuarentena(id: string, reviewer: string): Promise<{ record?: EntradaDeQuarentena; note?: string }> {
  const alvo = (id ?? "").trim();
  const autor = (reviewer ?? "").trim();
  if (alvo === "") throw new InstalacaoRecusadaError("escolha uma entrada");
  if (autor === "") throw new InstalacaoRecusadaError("uma revisão precisa de revisor");
  try {
    return await agentFetch("/api/agent/v1/agents/quarantine/release", {
      method: "POST",
      body: JSON.stringify({ id: alvo, reviewer: autor }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}

export interface MotorLocal {
  available: boolean;
  endpoint: string;
  model?: string;
  models?: string[];
  reason?: string;
}

export interface ExecucaoDeAvaliacao {
  engine?: MotorLocal;
  report?: {
    state?: string;
    persona_id?: string;
    suite_version?: string;
    cases?: { id?: string; kind?: string; passed?: boolean; reason?: string; output?: string }[];
    total?: number;
    failed?: number;
  };
  state?: string;
  inference_ran?: boolean;
  note?: string;
}

/** CORRE a suite contra o motor local. POST porque gasta tempo e CPU. */
export async function correrAvaliacaoReal(personaId: string, division: string): Promise<ExecucaoDeAvaliacao> {
  const persona = (personaId ?? "").trim();
  const div = (division ?? "").trim();
  if (persona === "" || div === "") throw new InstalacaoRecusadaError("escolha persona e divisão");
  try {
    return await agentFetch<ExecucaoDeAvaliacao>("/api/agent/v1/agents/evaluation/run", {
      method: "POST",
      body: JSON.stringify({ persona_id: persona, division: div }),
    });
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

/**
 * Desinstala uma persona do ledger.
 *
 * O SERVIDOR PODE RECUSAR, E ESSA RECUSA E' O PONTO
 *
 * Quando o destino tem trabalho local nao integrado (`LOCALLY_MODIFIED`/`CONFLICTED`), a rota responde 409 e
 * nao remove nada. O `traduzir` transforma esse 409 num erro tipado para o painel poder dizer PORQUE recusou,
 * em vez de mostrar um generico. *Um 409 que o cliente achata num "falhou" apaga a unica informacao que
 * importava.*
 */
/**
 * Actualiza uma instalacao. O servidor faz BACKUP antes de escrever e recusa (409) se houver trabalho local.
 */
export async function actualizarAgente(
  persona: Record<string, unknown>,
  target: string,
  destination: string,
): Promise<{ updated?: boolean; backup_id?: string; bytes?: number; note?: string }> {
  const problema = validarDestino(destination);
  if (problema) throw new InstalacaoRecusadaError(problema);
  try {
    return await agentFetch("/api/agent/v1/agents/update", {
      method: "POST",
      body: JSON.stringify({ persona, target, destination }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}

export async function removerAgente(
  personaId: string,
  target: string,
  destination: string,
): Promise<{ removed?: string; file_kept?: boolean; note?: string }> {
  try {
    return await agentFetch("/api/agent/v1/agents/remove", {
      method: "POST",
      body: JSON.stringify({ persona_id: personaId, target, destination }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
