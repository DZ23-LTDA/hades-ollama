// Comparacao objectiva de solucoes -- `POST /solutions/compare`.
//
// A ROTA E' POST E NAO MUTA NADA -- e diz isso no corpo.
//
//	*"O metodo e' POST, mas o efeito e' nenhum; o cliente nao deve ter de inferir isso do metodo."*
//
// `somente_leitura: true` vem do servidor. O painel nao infere do metodo: **le' o campo.**
//
// CINCO COISAS QUE ESTE CONTRATO ENSINA
//
//  1. **`integra_automaticamente` e' SEMPRE falso.** O comentario do tipo:
//     *"O sistema pode recomendar um vencedor; nao pode escolher por quem assina o merge."*
//  2. **`empate: true` significa que ha' algo a decidir, nao que falhou.** `vencedor` fica vazio
//     e `exige_aprovacao_humana` fica true.
//  3. **`nao_medidos` sao criterios que NINGUEM mediu** -- nao separaram os candidatos porque
//     nao havia dados. Nao e' uma lista de criterios que falharam.
//  4. **`medidos_parcialmente` e' o campo que mais se esquece**: criterios medidos em ALGUNS
//     candidatos e nao noutros. O comentario:
//     *"Um criterio medido num candidato e nao noutro nao compara: informa."*
//     Sem este campo, um criterio medido em metade dos candidatos **desaparecia** e quem
//     auditava nao sabia que a comparacao foi feita sem ele.
//  5. **`vencedor` vazio nao e' erro** -- e' empate ou ninguem passou nos testes.
//
// E AS MEDICOES SAO ENTRADAS, NAO SAIDA:
//   *"A rota nao inventa o que nao mediu -- quem mede e' o chamador, e o campo `medido`
//   obriga-o a dizer se mediu."*
import { agentFetch } from "@/lib/agenticClient";

export interface MedicaoDeCandidato {
  criterio: string;
  /** Se o chamador mediu, ou so' estimou. */
  medido?: boolean;
  valor?: number;
  unidade?: string;
  detalhe?: string;
}

export interface CandidatoDaSolucao {
  id: string;
  worktree?: string;
  branch?: string;
  commit?: string;
  agente?: string;
  testes_passaram: number;
  testes_falharam: number;
  medicoes?: MedicaoDeCandidato[] | null;
  evidencias?: string[] | null;
}

export interface MotivoDeRejeicao {
  id: string;
  motivo?: string;
  reason?: string;
}

export interface CriterioDeComparacao {
  criterio?: string;
  nome?: string;
  motivo?: string;
}

export interface ResultadoDaComparacao {
  candidatos?: CandidatoDaSolucao[] | null;
  /** IDs dos candidatos ACEITES, do melhor para o pior. */
  ranking?: string[] | null;
  /** Vazio em empate ou quando ninguem foi aceite. */
  vencedor?: string;
  empate: boolean;
  rejeitados?: MotivoDeRejeicao[] | null;
  nao_medidos?: CriterioDeComparacao[] | null;
  medidos_parcialmente?: CriterioDeComparacao[] | null;
  /** SEMPRE falso. */
  integra_automaticamente: boolean;
  exige_aprovacao_humana: boolean;
  explicacao?: string;
}

export interface RespostaDaComparacao extends ResultadoDaComparacao {
  missao?: string;
  somente_leitura: boolean;
  avisos?: string[] | null;
}

/** Nao foi possivel comparar (pedido invalido, demasiados candidatos, etc.). */
export class ComparacaoInvalidaError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "ComparacaoInvalidaError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/candidat|inv[aá]lid|obrigat|demasiad|400|413|422/i.test(mensagem)) {
    throw new ComparacaoInvalidaError(mensagem);
  }
  throw causa;
}

/** O vencedor, dito com o que ele significa. */
export function descreverVencedor(r: ResultadoDaComparacao): string {
  if (r.empate) return "EMPATE — nenhum vencedor; a decisão é humana";
  const v = (r.vencedor ?? "").trim();
  if (v !== "") return `vencedor recomendado: ${v}`;
  const rejeitados = (r.rejeitados ?? []).length;
  if (rejeitados > 0) return "nenhum candidato passou nos testes — não há recomendação";
  return "sem vencedor declarado";
}

/**
 * `integra_automaticamente` e' sempre falso. Este helper existe para o painel **nunca** mostrar
 * outra coisa, mesmo que o servidor um dia mude.
 */
export function descreverIntegracao(r: ResultadoDaComparacao): string {
  if (r.integra_automaticamente) {
    // Se isto acontecer, o contrato mudou -- e o painel diz que nao sabe o que fazer.
    return "o servidor declara integração automática — este painel NÃO a executa";
  }
  return "NÃO integra nada — aprovar e integrar é uma decisão humana separada";
}

/** Criterios que ninguem mediu: nao separaram, nao falharam. */
export function resumirNaoMedidos(r: ResultadoDaComparacao): string | null {
  const lista = r.nao_medidos ?? [];
  if (lista.length === 0) return null;
  return `${lista.length} criterio(s) sem medição em ninguém — não separaram os candidatos`;
}

/** Criterios medidos em alguns e nao noutros: informam, nao comparam. */
export function resumirMedidosParcialmente(r: ResultadoDaComparacao): string | null {
  const lista = r.medidos_parcialmente ?? [];
  if (lista.length === 0) return null;
  return `${lista.length} criterio(s) medido(s) só em alguns — informa, não compara`;
}

/** Nome de um criterio, sem inventar. */
export function nomeDoCriterio(c: CriterioDeComparacao): string {
  const n = (c.criterio ?? c.nome ?? "").trim();
  if (n !== "") return n;
  const m = (c.motivo ?? "").trim();
  return m === "" ? "critério sem nome" : m;
}

/** Motivo de rejeicao, sem inventar. */
export function motivoDaRejeicao(m: MotivoDeRejeicao): string {
  const t = (m.motivo ?? m.reason ?? "").trim();
  return t === "" ? "motivo não declarado" : t;
}

/** Contagem de testes de um candidato, crua. */
export function descreverTestes(c: CandidatoDaSolucao): string {
  const p = Math.max(0, c.testes_passaram ?? 0);
  const f = Math.max(0, c.testes_falharam ?? 0);
  if (f > 0) return `${p} passaram · ${f} FALHARAM`;
  return `${p} passaram`;
}

/** O ranking, na ordem. */
export function descreverRanking(r: ResultadoDaComparacao): string {
  const ranking = r.ranking ?? [];
  if (ranking.length === 0) return "nenhum candidato no ranking";
  return ranking.map((id, i) => `${i + 1}. ${id}`).join(" · ");
}

export async function compararSolucoes(
  candidatos: CandidatoDaSolucao[],
  missao = "",
): Promise<RespostaDaComparacao> {
  if (!Array.isArray(candidatos) || candidatos.length === 0) {
    throw new ComparacaoInvalidaError("o campo 'candidatos' e' obrigatorio e nao pode ser vazio");
  }
  try {
    return await agentFetch<RespostaDaComparacao>("/api/agent/v1/solutions/compare", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ missao, candidatos }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
