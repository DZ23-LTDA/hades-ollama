/**
 * FASE 28.7/28.8: cliente das rotas `artifact.checksum` e `artifact.verify`.
 *
 * O QUE ESTE FICHEIRO LIGA
 *
 * A E-1258 criou as rotas no servidor e provou-as com 8 testes. Mas **nenhum painel as chamava** --
 * o defeito recorrente desta missao:
 *
 *	*Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
 *	com o outro.*
 *
 * Este modulo e' o unico lugar do frontend que sabe o formato das duas rotas. Se o backend mudar um
 * nome de campo, o TypeScript falha aqui e nao no ecra.
 *
 * OS DOIS NOMES QUE NAO SE CONFUNDEM
 *
 *	`obterChecksum`      -> mede; nao compara com nada
 *	`verificarArtefacto` -> compara com uma referencia
 *
 * *A medicao e a verificacao sao operacoes diferentes. Uma funcao que so' sabe verificar obriga quem
 * so' quer o hash a inventar uma referencia -- e uma referencia inventada "passa" sempre.*
 */

import { agentFetch } from "@/lib/agenticClient";

/** Resposta de `artifact.checksum`. */
export interface ChecksumDeArtefacto {
  somente_leitura: boolean;
  nada_assinado: boolean;
  caminho: string;
  tamanho_bytes: number;
  sha256: string;
  aviso_de_fronteira: string;
}

/** Estado de assinatura, como o backend o declara. Quatro valores, nao um booleano. */
export type EstadoDeAssinatura =
  | "AUSENTE"
  | "PRESENTE"
  | "INVALIDA"
  | "NAO_VERIFICADA";

/** Resultado de `artifact.verify`. */
export interface ResultadoDeVerificacao {
  caminho: string;
  tamanho_bytes: number;
  sha256_medido: string;
  sha256_esperado: string;
  integro: boolean;
  adulterado: boolean;
  assinatura: EstadoDeAssinatura;
  detalhe_da_assinatura?: string;
  aprovado: boolean;
  duracao_ms: number;
  explicacao: string;
}

/** Resposta de `artifact.verify`. */
export interface VerificacaoDeArtefacto {
  somente_leitura: boolean;
  nada_assinado: boolean;
  resultado: ResultadoDeVerificacao;
  aviso_de_fronteira: string;
}

/**
 * Mede o SHA-256 de um ficheiro. NAO compara com nada.
 *
 * O caminho tem de ser absoluto: o servidor recusa caminhos relativos porque *um caminho relativo e'
 * relativo ao directorio de quem pede, e o servidor nao sabe qual e'*.
 */
export async function obterChecksum(
  caminho: string,
  sinal?: AbortSignal,
): Promise<ChecksumDeArtefacto> {
  return agentFetch<ChecksumDeArtefacto>(
    `/api/agent/v1/build/artifact/checksum?caminho=${encodeURIComponent(caminho)}`,
    { signal: sinal },
  );
}

/**
 * Verifica um ficheiro contra um SHA-256 de referencia.
 *
 * Sem referencia o servidor recusa com 400: *sem referencia isto seria uma medicao, nao uma
 * verificacao*. Este cliente nao inventa uma.
 */
export async function verificarArtefacto(
  caminho: string,
  sha256: string,
  sinal?: AbortSignal,
): Promise<VerificacaoDeArtefacto> {
  return agentFetch<VerificacaoDeArtefacto>(
    `/api/agent/v1/build/artifact/verify?caminho=${encodeURIComponent(caminho)}&sha256=${encodeURIComponent(sha256)}`,
    { signal: sinal },
  );
}

/**
 * Traduz o estado de assinatura para texto humano.
 *
 * A primeira versao do painel de build comparava `saudavel`/`ausente` -- **nomes que eu inventei** --
 * e passava nos dois lados porque o fixture tambem os tinha inventado (E-1250). Aqui os valores sao
 * os do tipo `EstadoDeAssinatura`, que o TypeScript obriga a cobrir por inteiro.
 *
 *	*Um teste que inventa o valor que o backend devolve nunca falha por causa do backend.*
 */
export function descreverAssinatura(estado: EstadoDeAssinatura): string {
  switch (estado) {
    case "AUSENTE":
      return "não assinada (esperado nesta RC)";
    case "PRESENTE":
      return "assinada";
    case "INVALIDA":
      return "assinatura inválida";
    case "NAO_VERIFICADA":
      return "assinatura não verificada neste host";
    default: {
      // Exaustividade: se o backend ganhar um estado novo, o TypeScript falha AQUI e nao no ecra.
      const inesperado: never = estado;
      return String(inesperado);
    }
  }
}

/**
 * Diz se o resultado autoriza tratar o artefacto como integro.
 *
 * *Aprovado significa "passou os gates que consegui aplicar", nao "todos os gates existiram".* Quem
 * exige assinatura le' `resultado.assinatura` e decide -- este resumo nao esconde a diferenca.
 */
export function resumirVerificacao(resultado: ResultadoDeVerificacao): string {
  if (resultado.adulterado) {
    return `ADULTERADO — o SHA-256 medido (${resultado.sha256_medido.slice(0, 12)}…) não corresponde ao esperado`;
  }
  if (resultado.aprovado) {
    return `íntegro — ${descreverAssinatura(resultado.assinatura)}`;
  }
  return `não aprovado — ${resultado.explicacao}`;
}

/** Formata o tamanho para o ecra, sem inventar precisao que o numero nao tem. */
export function tamanhoLegivel(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
