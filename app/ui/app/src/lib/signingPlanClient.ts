/**
 * FASE 28.7: cliente da rota `sign.plan`.
 *
 * PORQUE ESTE FICHEIRO EXISTE SEPARADO DE `signingClient.ts`
 *
 * A primeira versao deste cliente foi escrita **por cima** de `signingClient.ts`, que JA' EXISTIA e
 * serve o `Signing Identity Center` (FASE 28.5). Partiu `SigningIdentitiesPage.tsx` com 8 erros de
 * compilacao.
 *
 *	*Um nome que parece livre nao prova que o ficheiro nao existe: a pasta nao foi lida.*
 *
 * Dois clientes, dois ficheiros, dois nomes. O tipo de identidade aqui chama-se
 * `IdentidadeParaAvaliar` para nao haver dois `IdentidadeDeAssinatura` com campos diferentes.
 *
 * O QUE ESTE CLIENTE NAO TEM, POR DESENHO
 *
 * **Nao tem campo para a chave privada.** `caminho_do_certificado` e' o certificado PUBLICO:
 *
 *	*Nao leia o segredo para provar que ele existe: o tamanho e a data ja' provam que o ficheiro esta'
 *	la', e a leitura e' o risco.*
 *
 * **Nao ha' `assinar()`.** A rota avalia; assinar e' outra accao.
 *
 *	*Um botao morto e' pior do que um botao ausente: o ausente nao mente.*
 */

import { agentFetch } from "@/lib/agenticClient";

/** Tipo de assinatura, como o backend o classifica em `sign.plan`. */
export type TipoDeAssinatura =
  | "AUTHENTICODE"
  | "APPLE_DEVELOPER_ID"
  | "ANDROID_APK"
  | "OPENPGP";

/** Aptidao de uma identidade para um tipo. */
export type AptidaoDaIdentidade =
  | "APTA"
  | "SEM_EKU_DE_ASSINATURA"
  | "EXPIRADA"
  | "AINDA_NAO_VALIDA"
  | "EMISSOR_INESPERADO"
  | "FORMATO_DESCONHECIDO"
  | "ILEGIVEL";

/**
 * Identidade a avaliar.
 *
 * **Sem campo para segredo.** Nao ha' `chave_privada`, `senha` nem `pfx`.
 */
export interface IdentidadeParaAvaliar {
  caminho_do_certificado: string;
  rotulo: string;
  ambiente: string;
}

/** Avaliacao devolvida pelo provider. */
export interface AvaliacaoDaIdentidade {
  tipo: TipoDeAssinatura;
  aptidao: AptidaoDaIdentidade;
  apta: boolean;
  sujeito: string;
  emissor: string;
  valido_de: string;
  valido_ate: string;
  ekus: string[] | null;
  tem_eku_necessario: boolean;
  motivo: string;
}

/** Resposta de `sign.plan`. */
export interface PlanoDeAssinatura {
  somente_leitura: boolean;
  nada_assinado: boolean;
  avaliacao: AvaliacaoDaIdentidade;
  alvos_suportados: TipoDeAssinatura[] | null;
  aviso_de_fronteira: string;
}

/** Ambientes que a rota aceita. */
export function ambientesDeAssinatura(): string[] {
  return ["desenvolvimento", "preview", "producao"];
}

/** Traduz o tipo para uma etiqueta legivel. */
export function descreverTipoDeAssinatura(tipo: TipoDeAssinatura): string {
  switch (tipo) {
    case "AUTHENTICODE":
      return "Authenticode (Windows)";
    case "APPLE_DEVELOPER_ID":
      return "Apple Developer ID (macOS)";
    case "ANDROID_APK":
      return "Android APK";
    case "OPENPGP":
      return "OpenPGP";
    default: {
      const inesperado: never = tipo;
      throw new Error(`tipo de assinatura desconhecido: ${String(inesperado)}`);
    }
  }
}

/**
 * Traduz a aptidao para uma frase.
 *
 * Cada valor do backend tem de ter uma frase, e um valor desconhecido **falha alto** em vez de cair
 * num "desconhecido" silencioso.
 *
 *	*Um estado que so' o programador pode nomear invalido nao e' um estado: e' um campo de texto.*
 */
export function descreverAptidao(aptidao: AptidaoDaIdentidade): string {
  switch (aptidao) {
    case "APTA":
      return "a identidade serve para este tipo";
    case "SEM_EKU_DE_ASSINATURA":
      return "falta o EKU de assinatura de codigo (1.3.6.1.5.5.7.3.3)";
    case "EXPIRADA":
      return "a identidade esta' expirada";
    case "AINDA_NAO_VALIDA":
      return "a identidade ainda nao e' valida";
    case "EMISSOR_INESPERADO":
      return "o emissor nao e' o esperado para este tipo";
    case "FORMATO_DESCONHECIDO":
      return "o ficheiro nao e' um certificado num formato conhecido";
    case "ILEGIVEL":
      return "nao foi possivel ler o certificado";
    default: {
      const inesperado: never = aptidao;
      throw new Error(`aptidao desconhecida: ${String(inesperado)}`);
    }
  }
}

/** Pede o plano de assinatura. */
export async function planearAssinatura(
  tipo: TipoDeAssinatura,
  identidade: IdentidadeParaAvaliar,
): Promise<PlanoDeAssinatura> {
  return agentFetch<PlanoDeAssinatura>("/api/agent/v1/sign/plan", {
    method: "POST",
    body: JSON.stringify({ tipo, identidade }),
  });
}

/**
 * Diz o que uma avaliacao prova -- e o que nao prova.
 *
 *	*Aprovado significa "passou os gates que consegui aplicar", nao "todos os gates existiram".*
 */
export function provaDoPlanoDeAssinatura(plano: PlanoDeAssinatura): string {
  if (plano.nada_assinado !== true) {
    // Nunca deveria acontecer. Mas se acontecer, o painel tem de o dizer.
    return "ATENCAO: o servidor nao declarou que nada foi assinado.";
  }
  if (!plano.avaliacao.apta) {
    return `nao assinavel: ${descreverAptidao(plano.avaliacao.aptidao)}`;
  }
  return "avaliavel: os gates que este provider aplica passaram; NAO assinado. " +
    "A cadeia criptografica e a revogacao NAO sao validadas aqui.";
}
