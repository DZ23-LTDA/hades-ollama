// FASE 28.5 do adendo V3: cliente do Signing Identity Center.
//
// Espelha o contrato de `server/agent_signing_routes.go`. Nada aqui devolve nem manipula valor
// privado: a rota so' devolve metadados, e este cliente nao tem para onde guardar mais nada.
//
// *Um cliente que nao tem campo para o segredo nao pode guarda-lo por descuido.*

import { agentFetch } from "@/lib/agenticClient";

/** Ambientes declarados. Uma identidade de producao exige aprovacao; preview nao. */
export type AmbienteDeAssinatura = "desenvolvimento" | "preview" | "producao";

/** Plataformas suportadas. Uma por identidade: nao existe assinador universal. */
export type PlataformaDeAssinatura =
  | "android"
  | "ios"
  | "macos"
  | "windows"
  | "linux"
  | "web"
  | "oci";

/** Estado calculado no servidor, no momento da leitura. */
export type EstadoDaIdentidade =
  | "ativa"
  | "expira_breve"
  | "expirada"
  | "revogada"
  | "desconhecida";

export interface IdentidadeDeAssinatura {
  nome: string;
  plataforma: PlataformaDeAssinatura;
  ambiente: AmbienteDeAssinatura;
  provedor: string;
  /** Ultimos 4 bytes do fingerprint publico. Truncado no servidor de proposito. */
  fingerprint_publico_fim: string;
  expira_em: string;
  estado: EstadoDaIdentidade;
  utilizavel_agora: boolean;
  exige_aprovacao: boolean;
  runners_autorizados: number;
  pacote_ou_bundle?: string;
}

export interface ProblemaDeIdentidade {
  nome: string;
  severidade: string;
  mensagem: string;
}

export interface ResumoDoRegisto {
  total: number;
  por_estado: Record<string, number>;
  por_ambiente: Record<string, number>;
  problemas: number;
}

export interface EstadoDoRegisto {
  configurado: boolean;
  carregado: boolean;
  caminho?: string;
  mensagem: string;
  total: number;
}

export interface SigningIdentitiesResposta {
  estado: EstadoDoRegisto;
  identidades: IdentidadeDeAssinatura[];
  problemas: ProblemaDeIdentidade[];
  resumo: ResumoDoRegisto;
  verificado_em: string;
  aviso: string;
}

export interface SigningSummaryResposta {
  configurado: boolean;
  carregado: boolean;
  total: number;
  mensagem: string;
  resumo: ResumoDoRegisto;
  problemas: number;
}

/** Registo completo com os metadados de cada identidade. */
export function fetchSigningIdentities(): Promise<SigningIdentitiesResposta> {
  return agentFetch<SigningIdentitiesResposta>("/api/agent/v1/signing/identities");
}

/** So' o resumo, para o cartao do Mission Control. */
export function fetchSigningSummary(): Promise<SigningSummaryResposta> {
  return agentFetch<SigningSummaryResposta>("/api/agent/v1/signing/summary");
}

/**
 * Cor do estado.
 *
 * `desconhecida` usa ambar, nao vermelho: *nao conseguir determinar nao e' o mesmo que estar
 * mal, e pintar os dois de vermelho faria o utilizador procurar um problema onde falta
 * informacao.*
 */
export function corDoEstadoDaIdentidade(estado: EstadoDaIdentidade): string {
  switch (estado) {
    case "ativa":
      return "text-emerald-400";
    case "expira_breve":
      return "text-amber-400";
    case "desconhecida":
      return "text-amber-400";
    case "expirada":
      return "text-red-400";
    case "revogada":
      return "text-red-400";
  }
}

/** Etiqueta legivel do estado, em portugues. O enum cru nao chega ao utilizador. */
export function rotuloDoEstadoDaIdentidade(estado: EstadoDaIdentidade): string {
  switch (estado) {
    case "ativa":
      return "Ativa";
    case "expira_breve":
      return "Expira em breve";
    case "expirada":
      return "Expirada";
    case "revogada":
      return "Revogada";
    case "desconhecida":
      return "Validade desconhecida";
  }
}

/** Etiqueta legivel do ambiente. */
export function rotuloDoAmbiente(ambiente: AmbienteDeAssinatura): string {
  switch (ambiente) {
    case "desenvolvimento":
      return "Desenvolvimento";
    case "preview":
      return "Preview";
    case "producao":
      return "Produção";
  }
}

/**
 * Formata a validade como data curta, ou avisa quando e' invalida.
 *
 * *Uma data invalida mostrada como "Invalid Date" e' pior do que nenhuma data: parece um
 * defeito do produto em vez de um dado em falta.*
 */
export function validadeLegivel(iso: string): string {
  if (!iso) return "sem validade declarada";
  const data = new Date(iso);
  if (Number.isNaN(data.getTime())) return "validade inválida";
  return data.toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit", year: "numeric" });
}

/**
 * Uma identidade de producao nunca se apresenta como "pronta para usar" sem aprovacao.
 *
 * A regra vive aqui, e nao so' no servidor: *uma UI que mostra verde numa identidade que ainda
 * precisa de aprovacao ensina o utilizador a carregar no botao antes de ler.*
 */
export function podeAssinarAgora(identidade: IdentidadeDeAssinatura): boolean {
  if (identidade.exige_aprovacao) return false;
  return identidade.utilizavel_agora;
}
