// FASE 17: política de retenção de memória por organização.
//
// ESTA ROTA EXISTIA NO BACKEND SEM NENHUMA SUPERFÍCIE NO PRODUTO.
//
// A auditoria do E-1319 mediu 93 rotas de negócio com backend pronto e nenhuma
// referência no frontend. `/retention` era uma delas -- e era a mais irónica,
// porque as três evidências anteriores (E-1316, E-1317, E-1318) foram todas
// sobre retenção que existia e não acontecia.
//
//	*Uma rota que existe e não é consumida é uma promessa do backend, não uma
//	funcionalidade do produto.*
//
// ARMADILHA DO CONTRATO: o `agentFetch` aceita `/retention` e acrescenta o prefixo em
// runtime -- mas o `scripts/verify-contracts.mjs` compara o path CRU. Um cliente sem o
// prefixo funciona no produto e reprova no gate. Todos os outros clientes o passam
// explicito; este passou a fazer o mesmo.
//
//	*Um cliente que funciona em runtime e reprova no contrato nao esta' errado no produto:
//	esta' fora da convencao que o gate mede -- e a convencao e' o que mantem as chamadas
//	verificaveis.*
import { agentFetch } from "@/lib/agenticClient";

/** Limites do contrato do servidor. Validar aqui evita um 400 previsível. */
export const IDADE_MINIMA_DIAS = 1;
export const IDADE_MAXIMA_DIAS = 3650;
export const MEMORIAS_MAXIMAS_POR_PROJECTO = 10000;

export interface PoliticaDeRetencao {
  organization_id: string;
  max_age_days: number;
  max_memories_per_project: number;
  updated_at?: string;
}

export interface RespostaDeRetencao {
  organization_id: string;
  configured: boolean;
  policy?: PoliticaDeRetencao;
}

/**
 * `max_memories_per_project = 0` significa SEM TECTO de contagem.
 *
 * O servidor aceita 0 como valor legítimo (o intervalo é 0..10000). Mostrar "0"
 * como se fosse um limite de zero memórias seria dizer ao utilizador que a
 * política apaga tudo -- o oposto do que ela faz.
 */
export function descreverTectoDeMemorias(valor: number): string {
  if (valor === 0) return "sem tecto";
  if (valor === 1) return "1 memória por projecto";
  return `${valor.toLocaleString("pt-BR")} memórias por projecto`;
}

export function descreverPolitica(politica: PoliticaDeRetencao): string {
  const idade =
    politica.max_age_days === 1
      ? "1 dia"
      : `${politica.max_age_days.toLocaleString("pt-BR")} dias`;
  return `Mantém memórias até ${idade}; ${descreverTectoDeMemorias(
    politica.max_memories_per_project,
  )}.`;
}

/** Valida antes de enviar. Devolve a mensagem do problema, ou null se está bem. */
export function validarPolitica(
  idadeEmDias: number,
  memoriasPorProjecto: number,
): string | null {
  if (!Number.isInteger(idadeEmDias)) return "A idade tem de ser um número inteiro de dias.";
  if (idadeEmDias < IDADE_MINIMA_DIAS || idadeEmDias > IDADE_MAXIMA_DIAS) {
    return `A idade tem de estar entre ${IDADE_MINIMA_DIAS} e ${IDADE_MAXIMA_DIAS} dias.`;
  }
  if (!Number.isInteger(memoriasPorProjecto)) {
    return "O tecto de memórias tem de ser um número inteiro.";
  }
  if (memoriasPorProjecto < 0 || memoriasPorProjecto > MEMORIAS_MAXIMAS_POR_PROJECTO) {
    return `O tecto de memórias tem de estar entre 0 e ${MEMORIAS_MAXIMAS_POR_PROJECTO.toLocaleString("pt-BR")}.`;
  }
  return null;
}

export async function obterPoliticaDeRetencao(
  init?: RequestInit,
): Promise<RespostaDeRetencao> {
  return agentFetch<RespostaDeRetencao>("/api/agent/v1/retention", init);
}

export async function definirPoliticaDeRetencao(
  idadeEmDias: number,
  memoriasPorProjecto: number,
  init?: RequestInit,
): Promise<PoliticaDeRetencao> {
  return agentFetch<PoliticaDeRetencao>("/api/agent/v1/retention", {
    method: "PUT",
    body: JSON.stringify({
      max_age_days: idadeEmDias,
      max_memories_per_project: memoriasPorProjecto,
    }),
    ...init,
  });
}

export interface ResultadoDeAplicacao {
  organization_id: string;
  projects_scanned: number;
  removed_by_age: number;
  removed_by_count: number;
  applied_at?: string;
}

export async function aplicarPoliticaDeRetencao(
  init?: RequestInit,
): Promise<ResultadoDeAplicacao> {
  return agentFetch<ResultadoDeAplicacao>("/api/agent/v1/retention/apply", {
    method: "POST",
    ...init,
  });
}

/**
 * Uma aplicação que não removeu nada diz-se em palavras, não com "0".
 *
 *	*Zero removido é um resultado, e um resultado que se mostra igual ao
 *	nenhum resultado não informa ninguém.*
 */
export function descreverResultadoDeAplicacao(
  resultado: ResultadoDeAplicacao,
): string {
  const total = resultado.removed_by_age + resultado.removed_by_count;
  const projectos =
    resultado.projects_scanned === 1
      ? "1 projecto"
      : `${resultado.projects_scanned} projectos`;
  if (total === 0) {
    return `Nada a remover em ${projectos}. A política está a ser cumprida.`;
  }
  const partes: string[] = [];
  if (resultado.removed_by_age > 0) {
    partes.push(`${resultado.removed_by_age} por idade`);
  }
  if (resultado.removed_by_count > 0) {
    partes.push(`${resultado.removed_by_count} por excesso de contagem`);
  }
  return `Removidas ${total} memórias em ${projectos}: ${partes.join(" e ")}.`;
}
