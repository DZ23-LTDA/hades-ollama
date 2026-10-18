// FASE 28.8 do adendo V3: cliente do plano de release do projecto.
//
// Consome `GET /api/agent/v1/build/plan`, criada na E-1247. O contrato espelha o do backend de
// proposito: *dois contratos para a mesma pergunta divergem no primeiro dia em que alguem muda um
// deles.*
//
// Nada aqui instala, constroi ou assina. A rota e' de LEITURA e o tipo declara isso.
//
//	*Um painel que mostra "pronto" sem poder construir e' um painel que promete o que nao faz --
//	a menos que diga que so' le'.*

import { agentFetch } from "@/lib/agenticClient";

/** Um requisito de ferramenta, como o backend o descreve. */
export interface RequisitoDeFerramenta {
  Nome: string;
  Comando: string;
  Obrigatorio: boolean;
  Motivo: string;
  VersaoMinima?: string;
}

/** Uma sonda de ferramenta, com o estado observado. */
export interface SondaDeFerramenta {
  Requisito: RequisitoDeFerramenta;
  Estado: string;
  Caminho?: string;
  Versao?: string;
  Detalhe?: string;
}

/** O plano de um alvo. */
export interface PlanoDeAlvo {
  Alvo: string;
  Probes: SondaDeFerramenta[];
  Blockers: string[];
  TrabalhoIndependente: string[];
  Alvo_?: string;
}

/** O plano por alvo, como vem do backend. */
export interface AlvoPlanado {
  alvo: string;
  plano: PlanoDeAlvo;
  erro_de_plano?: string;
}

/** O projecto detectado. */
export interface ProjetoDetectado {
  raiz: string;
  stacks: string[];
  gestores: string[];
  alvos: string[];
  ficheiros_nativos?: string[];
  manifests: string[];
  avisos?: string[];
}

/** O plano do projecto inteiro. */
export interface PlanoDoProjeto {
  projeto: ProjetoDetectado;
  alvos: AlvoPlanado[];
  alvos_prontos: string[];
  alvos_bloqueados: string[];
  sem_alvos: boolean;
  trabalho_independente?: string[];
  avisos?: string[];
  explicacao: string;
}

/** Resposta de GET /build/plan. */
export interface PlanoDeRelease {
  somente_leitura: boolean;
  nada_instalado: boolean;
  nada_construido: boolean;
  repo: string;
  profundidade: number;
  plano: PlanoDoProjeto;
  blockers: string[];
  motivos: string[];
  aviso_de_fronteira: string;
}

/** Erro legivel de uma chamada ao plano. */
export class ErroDePlanoDeRelease extends Error {
  // O campo e' declarado explicitamente: `erasableSyntaxOnly` proibe parameter properties, e o
  // projecto tem essa opcao ligada. *Uma opcao do compilador que se contorna com sintaxe proibida
  // nao e' uma opcao: e' uma sugestao.*
  readonly estado: number;

  constructor(mensagem: string, estado: number) {
    super(mensagem);
    this.name = "ErroDePlanoDeRelease";
    this.estado = estado;
  }
}

/**
 * Pede o plano de release de um projecto.
 *
 * O caminho e' resolvido pelo SERVIDOR. Enviar um caminho relativo seria ambiguo: relativo a que
 * directorio? O painel pede o absoluto e o servidor recusa o que nao existir.
 *
 *	*Dois relativos diferentes com o mesmo texto sao dois caminhos diferentes.*
 *
 * `agentFetch<T>` ja' devolve o corpo descodificado e ja' lanca em erro HTTP; nao ha' aqui um
 * `Response` para inspeccionar. Duplicar a verificacao de `ok` seria tratar o transporte como se
 * ele nao fizesse o seu trabalho -- e *uma segunda verificacao que nunca dispara e' codigo morto
 * com boa reputacao.*
 */
export async function obterPlanoDeRelease(
  repo: string,
  profundidade?: number,
  sinal?: AbortSignal,
): Promise<PlanoDeRelease> {
  const params = new URLSearchParams({ repo });
  if (typeof profundidade === "number") {
    params.set("profundidade", String(profundidade));
  }
  try {
    return await agentFetch<PlanoDeRelease>(`/api/agent/v1/build/plan?${params.toString()}`, {
      method: "GET",
      signal: sinal,
    });
  } catch (causa) {
    if (causa instanceof ErroDePlanoDeRelease) throw causa;
    throw new ErroDePlanoDeRelease(
      causa instanceof Error ? causa.message : String(causa),
      0,
    );
  }
}

/**
 * Traduz o estado de uma sonda para texto humano, sem jargao.
 *
 * Existe porque o painel mostraria `ToolAbsent` a um utilizador que nao sabe o que isso e', e
 * *um estado tecnico apresentado sem traducao e' um estado que o utilizador nao tem.*
 */
export function descreverEstadoDaSonda(estado: string): string {
  switch (estado) {
    case "saudavel":
    case "healthy":
      return "pronta";
    case "ausente":
    case "absent":
      return "não instalada";
    case "versao_antiga":
    case "outdated":
      return "versão antiga";
    case "quebrada":
    case "broken":
      return "instalada mas não funciona";
    case "desconhecido":
    case "unknown":
      return "não foi possível determinar";
    default:
      return estado;
  }
}

/** Total de requisitos por estado, para o resumo do painel. */
export interface ResumoDeSondas {
  total: number;
  prontas: number;
  emFalta: number;
}

/**
 * Conta as sondas de todos os alvos.
 *
 * Um requisito que aparece em dois alvos conta UMA vez por alvo: sao planos diferentes e o
 * utilizador precisa de saber que falta nos dois.
 */
export function resumirSondas(plano: PlanoDoProjeto): ResumoDeSondas {
  let total = 0;
  let prontas = 0;
  for (const alvo of plano.alvos) {
    for (const sonda of alvo.plano.Probes ?? []) {
      total += 1;
      if (sonda.Estado === "saudavel" || sonda.Estado === "healthy") {
        prontas += 1;
      }
    }
  }
  return { total, prontas, emFalta: total - prontas };
}

/**
 * Diz se o plano permite construir alguma coisa AGORA.
 *
 * Existe para o painel nao ter de reinterpretar `alvos_prontos` em cada sitio, e para haver UM
 * lugar que muda se a regra mudar.
 */
export function podeConstruirAlgo(plano: PlanoDeRelease | null): boolean {
  if (!plano) return false;
  if (plano.plano.sem_alvos) return false;
  return plano.plano.alvos_prontos.length > 0;
}
