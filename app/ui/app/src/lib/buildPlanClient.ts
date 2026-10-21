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
 * OS VALORES REAIS DO BACKEND SAO OS DE `ToolState` EM `internal/agent/toolchain_plan.go`:
 *
 *   PRESENT_COMPATIBLE_HEALTHY     ferramenta pronta
 *   PRESENT_COMPATIBLE_UNVERIFIED  existe, mas ninguem executou um smoke
 *   PRESENT_OUTDATED              existe, versao abaixo do minimo
 *   PRESENT_BROKEN                existe, mas nao funciona
 *   ABSENT                        nao encontrada
 *   UNKNOWN                       nao foi possivel determinar
 *
 * A primeira versao deste switch comparava `saudavel`/`ausente`/`outdated` -- **nomes que eu
 * inventei**. O E2E contra o backend REAL (`inno-setup` -> `ABSENT`) apanhou-o depois de o fixture
 * passar, porque o fixture tambem os tinha inventado.
 *
 *	*Um teste que inventa o valor que o backend devolve nunca falha por causa do backend.*
 *
 * A comparacao passou a ser sobre o valor normalizado E sobre o prefixo `PRESENT_`, para nao
 * depender de uma lista que fica desactualizada quando o backend ganha um estado novo.
 */
export function descreverEstadoDaSonda(estado: string): string {
  const normalizado = (estado ?? "").trim().toUpperCase();
  // `PRESENT_*` e' a familia de estados em que a ferramenta EXISTE. Distinguir dentro dela importa;
  // negar a familia inteira seria dizer que uma ferramenta quebrada nao esta' instalada -- e ela
  // esta': *a diferenca entre "nao tenho" e "tenho e nao funciona" decide o que o utilizador faz a
  // seguir.*
  switch (normalizado) {
    case "PRESENT_COMPATIBLE_HEALTHY":
      return "pronta";
    case "PRESENT_COMPATIBLE_UNVERIFIED":
      return "instalada, sem verificação de funcionamento";
    case "PRESENT_OUTDATED":
      return "versão antiga";
    case "PRESENT_BROKEN":
      return "instalada mas não funciona";
    case "ABSENT":
      return "não instalada";
    case "UNKNOWN":
      return "não foi possível determinar";
    default:
      return normalizado === "" ? "desconhecido" : estado;
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
      if (sonda.Estado === "PRESENT_COMPATIBLE_HEALTHY") {
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
