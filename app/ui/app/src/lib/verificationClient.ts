/**
 * FASE 28.7: cliente da rota `build.verify`.
 *
 * O QUE ESTA ROTA E' -- E O QUE NAO E'
 *
 * Ela **compara cada artefacto com o SHA-256 que o cliente enviou**. Nao constroi, nao assina, nao
 * instala e nao procura o build por ID.
 *
 * `aprovada: true` significa que **todos os ficheiros continuam iguais ao que o build mediu** -- nao
 * significa que exista assinatura.
 *
 *	*Aprovado significa "passou os gates que consegui aplicar", nao "todos os gates existiram".*
 *
 * A DISTINCAO QUE O BACKEND FAZ E ESTE CLIENTE PRESERVA
 *
 * `sem_assinatura`: o adaptador olhou e nao havia assinatura.
 * `nao_verificadas`: nao houve adaptador para olhar.
 *
 *	"Olhei e nao ha'" e "nao consegui olhar" sao factos diferentes.
 *
 * *Um estado que mente e' pior do que nenhum.*
 */

import { agentFetch } from "@/lib/agenticClient";

/** Estado da assinatura de um artefacto. */
export type EstadoDeAssinatura = "AUSENTE" | "PRESENTE" | "INVALIDA" | "NAO_VERIFICADA";

/** Resultado da verificacao de um artefacto. */
export interface ResultadoDeVerificacao {
  caminho: string;
  tamanho_bytes: number;
  integro: boolean;
  adulterado: boolean;
  assinatura: EstadoDeAssinatura;
  detalhe_da_assinatura?: string;
  aprovado: boolean;
  duracao_ms: number;
  explicacao: string;
}

/** A verificacao da construcao inteira. */
export interface VerificacaoDaConstrucao {
  resultados: ResultadoDeVerificacao[] | null;
  total: number;
  integros: number;
  adulterados: number;
  /** Olhou e nao havia assinatura. */
  sem_assinatura: number;
  /** Nao houve adaptador para olhar. **Nunca somar com `sem_assinatura`.** */
  nao_verificadas: number;
  aprovada: boolean;
  explicacao: string;
}

/** Resposta de `build.verify`. */
export interface RelatorioDeVerificacao {
  somente_leitura: boolean;
  nada_assinado: boolean;
  verificacao: VerificacaoDaConstrucao;
  aviso_de_fronteira: string;
}

/** Um artefacto a verificar, com o SHA-256 que o build mediu. */
export interface ArtefactoParaVerificar {
  caminho: string;
  sha256: string;
}

/**
 * Verifica artefactos contra os checksums dados.
 *
 * O cliente **envia** os checksums; o servidor **mede** os ficheiros e compara. O cliente nao escolhe
 * a raiz -- os caminhos sao os que o build produziu.
 */
export async function verificarArtefactos(
  artefactos: ArtefactoParaVerificar[],
): Promise<RelatorioDeVerificacao> {
  return agentFetch<RelatorioDeVerificacao>("/api/agent/v1/build/verify", {
    method: "POST",
    body: JSON.stringify({ artefactos }),
  });
}

/**
 * Traduz o estado da assinatura.
 *
 * Um valor desconhecido **falha alto** -- nao cai num "outro" silencioso.
 */
export function descreverEstadoDeAssinatura(estado: EstadoDeAssinatura): string {
  switch (estado) {
    case "AUSENTE":
      return "sem assinatura (o adaptador olhou)";
    case "PRESENTE":
      return "assinatura presente";
    case "INVALIDA":
      return "assinatura INVALIDA";
    case "NAO_VERIFICADA":
      return "nao inspeccionada (sem adaptador)";
    default: {
      const inesperado: never = estado;
      throw new Error(`estado de assinatura desconhecido: ${String(inesperado)}`);
    }
  }
}

/**
 * Resume o que a verificacao provou -- e o que ficou por provar.
 *
 * **Nunca diz "build assinado".** Diz quantos ficaram por inspeccionar.
 */
export function resumirVerificacao(relatorio: RelatorioDeVerificacao): string {
  const v = relatorio.verificacao;
  if (v.total === 0) {
    return "Nenhum artefacto foi verificado.";
  }
  if (v.adulterados > 0) {
    return `${v.adulterados} de ${v.total} artefacto(s) ADULTERADO(S) -- a construcao nao pode ser ` +
      "distribuida.";
  }
  if (!v.aprovada) {
    return `${v.integros} de ${v.total} integro(s); a construcao nao foi aprovada.`;
  }
  const porInspecionar = v.nao_verificadas;
  if (porInspecionar > 0) {
    return `${v.total} artefacto(s) integro(s), mas ${porInspecionar} NAO foi/foram inspeccionado(s) ` +
      "quanto a assinatura.";
  }
  return `${v.total} artefacto(s) integro(s) e todos inspeccionados quanto a assinatura.`;
}

/**
 * Diz o que `aprovada` significa.
 *
 *	*Aprovado significa "passou os gates que consegui aplicar", nao "todos os gates existiram".*
 */
export function provaDaVerificacao(relatorio: RelatorioDeVerificacao): string {
  if (relatorio.verificacao.adulterados > 0) {
    return "REPROVADA: ha' pelo menos um artefacto cujo checksum nao corresponde ao que o build mediu.";
  }
  if (relatorio.nada_assinado) {
    return "nada foi assinado: esta rota apenas compara checksums. `aprovada` nao e' prova de " +
      "assinatura.";
  }
  if (!relatorio.verificacao.aprovada) {
    return "nao aprovada. Leia a explicacao do servidor.";
  }
  return "checksums conferem. Isto NAO prova assinatura, instalabilidade nem ausencia de malware.";
}
