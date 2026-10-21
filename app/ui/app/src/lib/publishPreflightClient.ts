/**
 * Cliente do `publish.preflight` do Release Hub (E-1298).
 *
 * A ROTA NAO PUBLICA NADA
 *
 * O contrato do provider tem UM metodo: `Preflight`. **Nao ha' `Publicar()`** -- nem para devolver
 * "nao autorizado". A resposta traz `publicado: false` sempre, e `nada_publicado: true` no envelope.
 *
 *     *Um cliente que so' sabe perguntar "posso?" nao pode publicar por engano. A ausencia da
 *     capacidade e' a garantia mais forte que existe.*
 *
 * PORQUE NAO HA' CAMPO PARA O ARTEFACTO NEM PARA A CREDENCIAL
 *
 * O pedido escolhe **o canal, e mais nada**. O caminho do artefacto vem de
 * `HADES_ARTEFACTO_A_PUBLICAR` e a credencial de `HADES_CREDENCIAL_DE_PUBLICACAO` -- variaveis **do
 * servidor**. O cliente nao as escolhe, nao as envia e nao as ve.
 *
 *     *Um painel que tem um campo para o segredo acaba por o receber.*
 *
 * Se este cliente tivesse um campo "caminho do artefacto", o utilizador poderia apontar o preflight
 * a um ficheiro que o servidor nao autoriza, e a fronteira mudava de lado. Nao tem.
 *
 * `LOCAL_ONLY` NAO E' DEPLOY PUBLICO
 *
 * O canal `LOCAL_ONLY` existe para provar o pipeline sem sair da maquina. Ele tem `publico: false`,
 * e o painel mostra-o como **"ensaio local"**, nunca como "publicado". A distincao e' o ponto todo:
 * publicar localmente e publicar publicamente nao sao a mesma coisa, e um painel que os mostra iguais
 * ensina o utilizador a confundir os dois.
 */

import { agentFetch } from "@/lib/agenticClient";

/** Os seis canais do `PublishingProvider`. Espelham `CanalDePublicacao` no Go. */
export type CanalDePublicacao =
  | "LOCAL_ONLY"
  | "GOOGLE_PLAY"
  | "APP_STORE"
  | "EAS_UPDATE"
  | "GITHUB_RELEASE"
  | "OCI_REGISTRY";

/** Uma condicao do preflight: o nome, se esta' satisfeita, e o detalhe humano. */
export interface CondicaoDoPreflight {
  nome: string;
  satisfeita: boolean;
  detalhe: string;
}

/** O veredicto sobre UM canal. Espelha `PreflightDePublicacao` no Go. */
export interface PreflightDePublicacao {
  canal: CanalDePublicacao;
  /** `false` para `LOCAL_ONLY`: o canal nao sai da maquina. */
  publico: boolean;
  pronto: boolean;
  condicoes: CondicaoDoPreflight[];
  /** Sempre verdadeiro para canal publico: publicar exige autorizacao humana. */
  requer_aprovacao: boolean;
  /** **Sempre `false`.** Nao ha' `submit` neste provider. */
  publicado: boolean;
  explicacao: string;
  aviso: string;
}

/** O envelope da resposta: as duas garantias vem separadas do veredicto. */
export interface RespostaDoPreflightDePublicacao {
  somente_leitura: boolean;
  nada_publicado: boolean;
  preflight: PreflightDePublicacao;
  aviso_de_fronteira: string;
}

/** Os seis canais, na ordem em que o painel os mostra. */
export function canaisDePublicacao(): CanalDePublicacao[] {
  return ["LOCAL_ONLY", "GOOGLE_PLAY", "APP_STORE", "EAS_UPDATE", "GITHUB_RELEASE", "OCI_REGISTRY"];
}

/** `true` quando o canal entrega a alguem que nao e' esta maquina. */
export function canalEhPublico(canal: CanalDePublicacao): boolean {
  return canal !== "LOCAL_ONLY";
}

/** Rotulo humano do canal. Nao inventa nomes: usa os do contrato. */
export function descreverCanal(canal: CanalDePublicacao): string {
  switch (canal) {
    case "LOCAL_ONLY":
      return "Somente local (ensaio, sem destino externo)";
    case "GOOGLE_PLAY":
      return "Google Play";
    case "APP_STORE":
      return "App Store";
    case "EAS_UPDATE":
      return "EAS Update (OTA)";
    case "GITHUB_RELEASE":
      return "GitHub Release";
    case "OCI_REGISTRY":
      return "Registro OCI";
  }
}

/** Pergunta ao servidor se o canal esta' pronto. **Nao publica nada.** */
export async function fetchPreflightDePublicacao(
  canal: CanalDePublicacao,
): Promise<RespostaDoPreflightDePublicacao> {
  return agentFetch<RespostaDoPreflightDePublicacao>(
    `/api/agent/v1/publish/preflight/${encodeURIComponent(canal)}`,
    { method: "POST" },
  );
}

/**
 * Texto que diz, em linguagem simples, o que o preflight decidiu.
 *
 * A ordem importa: `publicado` primeiro (e' sempre falso, e dize-lo e' o ponto), depois `pronto`,
 * e so' depois a explicacao do servidor.
 */
export function resumirPreflight(preflight: PreflightDePublicacao): string {
  if (preflight.publicado) {
    // Inalcancavel por contrato. Se acontecer, e' uma regressao grave e o texto di-lo.
    return "INCONSISTENTE: o provider devolveu `publicado: true`, mas o contrato nao tem `Publicar()`.";
  }
  if (!preflight.pronto) {
    const pendentes = preflight.condicoes.filter((c) => !c.satisfeita).map((c) => c.nome);
    const lista = pendentes.length > 0 ? ` Pendentes: ${pendentes.join(", ")}.` : "";
    return `Nao pronto.${lista}`;
  }
  if (!preflight.publico) {
    return "Pronto para ensaio local. Nada sera' enviado para fora desta maquina.";
  }
  return `Pronto para pedir autorizacao humana. Nada foi publicado. ${preflight.explicacao}`.trim();
}

/** Quantas condicoes estao satisfeitas, para o resumo do painel. */
export function contarCondicoes(preflight: PreflightDePublicacao): {
  satisfeitas: number;
  total: number;
} {
  return {
    satisfeitas: preflight.condicoes.filter((c) => c.satisfeita).length,
    total: preflight.condicoes.length,
  };
}
