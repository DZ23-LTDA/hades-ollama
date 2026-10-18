// FASE 2 do adendo V3: cliente da superficie de worktrees.
//
// Consome as rotas `/api/agent/v1/worktrees/*` criadas na E-1232. O contrato espelha o do backend
// de proposito: *dois contratos para a mesma pergunta divergem no primeiro dia em que alguem muda
// um deles.*
//
// Nada aqui remove, suspende ou cria worktrees. Todas as chamadas sao GET.
//
//	*Remover um worktree exige uma decisao sobre trabalho nao integrado; nao pode ser efeito
//	lateral de um clique num painel.*

import { agentFetch } from "@/lib/agenticClient";

/** Uma linha do inventario de worktrees do repositorio. */
export interface WorktreeDescoberto {
  caminho_resolvido: string;
  branch?: string;
  commit?: string;
  detached: boolean;
  criado_pelo_hades: boolean;
  principal: boolean;
  bloqueado: boolean;
}

/** Resposta de GET /worktrees. */
export interface WorktreeInventario {
  repo: string;
  resumo: string;
  total: number;
  do_hades: number;
  externo: number;
  lista: WorktreeDescoberto[];
  avisos: string[];
  somente_leitura: boolean;
  medido_em: string;
}

/** Resposta de GET /worktrees/diagnose. */
export interface WorktreeDiagnostico {
  mission_id: string;
  estado: string;
  estado_valido: boolean;
  estado_traduzido: boolean;
  estado_observado: string;
  proximos: string[];
  alvo_seguro: boolean;
  commits_nao_integrados: number;
  ficheiros_modificados: number;
  tem_trabalho_nao_integrado: boolean;
  detalhe?: string;
  inconsistente?: string;
  pode_remover: boolean;
  motivo_remover?: string;
}

/** Inventario de worktrees de um repositorio. */
export function fetchWorktreeInventario(repo: string): Promise<WorktreeInventario> {
  return agentFetch<WorktreeInventario>(
    `/api/agent/v1/worktrees?repo=${encodeURIComponent(repo)}`,
  );
}

/** Diagnostico de ciclo de vida de uma missao. */
export function fetchWorktreeDiagnostico(
  repo: string,
  missao: string,
): Promise<WorktreeDiagnostico> {
  return agentFetch<WorktreeDiagnostico>(
    `/api/agent/v1/worktrees/diagnose?repo=${encodeURIComponent(repo)}&missao=${encodeURIComponent(missao)}`,
  );
}

/**
 * Rotulo legivel de um estado do ciclo de vida.
 *
 * O estado vem do backend numa lista fechada de 9 valores. Um valor desconhecido NAO e' traduzido
 * para um estado plausivel: mostra-se como desconhecido.
 *
 *	*Um painel que inventa um rotulo para um estado que nao conhece ensina o utilizador a
 *	confiar no que nao foi verificado.*
 */
export function rotuloDoEstadoDoWorktree(estado: string): string {
  switch (estado) {
    case "criado":
      return "Criado";
    case "preparado":
      return "Preparado";
    case "em_uso":
      return "Em uso";
    case "suspenso":
      return "Suspenso";
    case "integrado":
      return "Integrado";
    case "arquivado":
      return "Arquivado";
    case "removido":
      return "Removido";
    case "falhado":
      return "Falhado";
    case "orfao":
      return "Orfao";
    default:
      return "Estado desconhecido";
  }
}

/** Classe de cor do estado. Vermelho so' quando ha' trabalho em risco ou inconsistencia. */
export function corDoEstadoDoWorktree(estado: string): string {
  switch (estado) {
    case "em_uso":
      return "text-emerald-300";
    case "preparado":
    case "criado":
      return "text-sky-300";
    case "suspenso":
    case "arquivado":
      return "text-amber-300";
    case "integrado":
      return "text-emerald-200";
    case "falhado":
    case "orfao":
      return "text-rose-300";
    default:
      return "text-neutral-400";
  }
}

/** Nome curto de um caminho, para caber numa linha da tabela. */
export function nomeCurtoDoCaminho(caminho: string): string {
  const partes = caminho.split(/[\\/]/).filter(Boolean);
  return partes.length === 0 ? caminho : partes[partes.length - 1];
}
