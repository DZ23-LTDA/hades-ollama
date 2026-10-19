// Agentes de servico e APROVACAO HUMANA -- a fila que a FASE 4 exige.
//
// Decima nona, vigesima e vigesima primeira rotas do E-1319 a chegar ao produto:
// `/service-agents`, `/service-agents/approvals` e `/service-agents/audit`.
//
// `service-agents` tinha 6 rotas e `service_agent_approvals.go` **45 KB**: quorum, politica,
// fingerprint, tenant, expiracao, notificacao. Nao havia componente `*ervice*`/`*gent*`.
//
// OS CONTRATOS:
//
//	GET /service-agents                  -> 200 {agents, count}  (catalogo estatico, sempre 200)
//	GET /service-agents/approvals?status= -> 200 {approvals, count, queue, notification}
//	                                         503 store ausente / 400 status invalido
//	GET /service-agents/audit            -> 200 AuditStatus   503 sem trilha duravel
//
// CINCO DECISOES QUE O PAINEL NAO PODE ERRAR:
//
//  1. **`role_checked=false` E' UMA DECLARACAO, NAO UM DETALHE.** O proprio tipo diz:
//
//         // RoleChecked diz se a decisao passou por LISTA DE APROVADORES. `false` e' declarado:
//         // sem lista configurada o sistema exige ator identificado, mas NAO verifica se ele
//         // pode aprovar -- e isso precisa aparecer no registro, em vez de virar confianca
//         // implicita.
//
//     O painel mostra **«sem lista de aprovadores»** quando for falso. Uma aprovacao que nao
//     verificou QUEM aprovou nao e' a mesma aprovacao.
//
//  2. **QUORUM E' PARCIAL, NAO BINARIO.** `required_approvals` conta decisoes favoraveis de
//     atores DISTINTOS e fica gravado no pedido. O painel mostra **«2 de 3»** -- nunca
//     «pendente» a seco, que esconde que ja' houve uma aprovacao.
//
//  3. **`overdue` SEPARA PENDENTE DE ESQUECIDO.** O tipo diz: *«Overdue diz que o pedido
//     passou do limiar de atencao -- sem isto, "pendente" e "esquecido" ficam
//     indistinguiveis para quem olha a fila.»*
//
//  4. **A NOTIFICACAO PODE TER FALHADO COM A FILA DE PE'.** Se o webhook estiver fora do ar,
//     a fila existe e ninguem foi avisado. O painel mostra os tres campos.
//
//  5. **O FINGERPRINT E' O QUE AMARRA A APROVACAO AO PEDIDO.** O tipo: *«Fingerprint amarra a
//     aprovacao a PARAMETROS e ESCOPO: aprovar um valor nao autoriza outro.»* O painel
//     mostra-o **truncado**: e' identificador, nao conteudo.
import { agentFetch } from "@/lib/agenticClient";

export type EstadoDaAprovacao = "pending" | "approved" | "denied" | "consumed" | "expired";

export interface DecisaoDeAprovacao {
  actor_id?: string;
  decision?: string;
  reason?: string;
  decided_at?: string;
}

export interface AprovacaoDeServico {
  id: string;
  organization_id: string;
  agent_id?: string;
  conversation_id?: string;
  tool_name: string;
  fingerprint: string;
  status: string;
  requested_at: string;
  requested_by?: string;
  decided_at?: string;
  decided_by?: string;
  reason?: string;
  consumed_at?: string;
  expires_at?: string;
  role_checked: boolean;
  value_cents?: number;
  required_approvals: number;
  decisions?: DecisaoDeAprovacao[] | null;
}

export interface ItemDaFila {
  id: string;
  tool_name: string;
  conversation_id?: string;
  approvals_given: number;
  approvals_needed: number;
  age_seconds: number;
  overdue: boolean;
}

export interface FilaDeAprovacoes {
  organization_id: string;
  pending: number;
  overdue: number;
  oldest_seconds: number;
  pending_ttl_seconds: number;
  expired: number;
  items: ItemDaFila[] | null;
}

export interface EstadoDaNotificacao {
  configurado: boolean;
  entregues: number;
  ultimo_erro: string;
}

export interface RespostaDeAprovacoes {
  approvals: AprovacaoDeServico[] | null;
  count: number;
  queue: FilaDeAprovacoes;
  notification: EstadoDaNotificacao;
}

export interface AgenteDeServico {
  id?: string;
  name?: string;
  description?: string;
  tools?: string[];
  [chave: string]: unknown;
}

export interface RespostaDeAgentes {
  agents: AgenteDeServico[] | null;
  count: number;
}

export interface EstadoDaAuditoria {
  persisted: boolean;
  organization_id?: string;
  entries?: number;
  [chave: string]: unknown;
}

/** O store de aprovacao humana nao esta' configurado neste runtime. */
export class AprovacoesAusentesError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "AprovacoesAusentesError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/n[aã]o est[aá] configurad|n[aã]o est[aá] dispon[íi]vel|503|trilha de auditoria/i.test(mensagem)) {
    throw new AprovacoesAusentesError(mensagem);
  }
  throw causa;
}

/**
 * Diz se a decisao passou por lista de aprovadores.
 *
 * `role_checked=false` NAO e' «aprovado por qualquer um»: e' «o sistema exigiu ator
 * identificado, mas nao verificou se ele podia aprovar». Sao coisas diferentes.
 */
export function descreverVerificacaoDePapel(roleChecked: boolean | undefined): string {
  if (roleChecked === true) return "verificado contra lista de aprovadores";
  if (roleChecked === false) return "sem lista de aprovadores — apenas ator identificado";
  return "sem informacao de verificacao de papel";
}

/** Quorum em texto: «2 de 3». Nunca esconde que ja' houve aprovacoes parciais. */
export function descreverQuorum(aprovacoes?: number, necessarias?: number): string {
  const dadas = Math.max(0, aprovacoes ?? 0);
  const pedidas = Math.max(1, necessarias ?? 1);
  if (pedidas <= 1) return "decisao unica";
  return `${dadas} de ${pedidas} aprovacoes`;
}

/** Idade em segundos, legivel. Segundos nao se mostram crus. */
export function descreverIdade(segundos: number | undefined): string {
  const total = Math.max(0, Math.round(segundos ?? 0));
  if (total < 60) return `${total} s`;
  const minutos = Math.floor(total / 60);
  if (minutos < 60) return `${minutos} min`;
  const horas = Math.floor(minutos / 60);
  if (horas < 24) return `${horas} h`;
  return `${Math.floor(horas / 24)} d`;
}

/**
 * Descreve o estado da notificacao: a fila pode existir sem ninguem ter sido avisado.
 *
 * O servidor manda os tres campos por uma razao declarada: *«se o destino do webhook
 * estiver fora do ar, quem opera precisa saber que a fila existe mas ninguem foi avisado»*.
 */
export function descreverNotificacao(estado: EstadoDaNotificacao | undefined): string {
  if (!estado) return "sem informacao de notificacao";
  if (!estado.configurado) return "nenhum canal de aviso configurado — a fila nao avisa ninguem";
  const erro = (estado.ultimo_erro ?? "").trim();
  if (erro !== "") return `canal configurado, mas o ultimo aviso falhou: ${erro}`;
  return `canal configurado, ${estado.entregues} aviso(s) entregue(s)`;
}

/** O fingerprint truncado: e' identificador, nao conteudo. */
export function truncarFingerprint(fingerprint: string | undefined): string {
  const limpo = (fingerprint ?? "").trim();
  if (limpo === "") return "sem fingerprint";
  if (limpo.length <= 12) return limpo;
  return `${limpo.slice(0, 8)}…${limpo.slice(-4)}`;
}

/** Estado da aprovacao em texto legivel, sem inventar. */
export function descreverEstadoDaAprovacao(estado: string | undefined): string {
  const bruto = (estado ?? "").trim();
  if (bruto === "") return "sem estado";
  const mapa: Record<string, string> = {
    pending: "pendente",
    approved: "aprovada",
    denied: "negada",
    consumed: "consumida",
    expired: "expirada",
  };
  return mapa[bruto] ?? `estado desconhecido (${bruto})`;
}

export async function listarAgentesDeServico(init?: RequestInit): Promise<RespostaDeAgentes> {
  return agentFetch<RespostaDeAgentes>("/api/agent/v1/service-agents", init);
}

export async function listarAprovacoes(
  estado?: string,
  init?: RequestInit,
): Promise<RespostaDeAprovacoes> {
  const limpo = (estado ?? "").trim();
  const sufixo = limpo !== "" ? `?status=${encodeURIComponent(limpo)}` : "";
  try {
    return await agentFetch<RespostaDeAprovacoes>(
      `/api/agent/v1/service-agents/approvals${sufixo}`,
      init,
    );
  } catch (causa) {
    traduzir(causa);
  }
}

export async function estadoDaAuditoria(init?: RequestInit): Promise<EstadoDaAuditoria> {
  try {
    return await agentFetch<EstadoDaAuditoria>("/api/agent/v1/service-agents/audit", init);
  } catch (causa) {
    traduzir(causa);
  }
}
