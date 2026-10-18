/**
 * FASE 28.7: cliente da rota `sbom.generate`.
 *
 * O QUE ESTE CLIENTE NAO TEM, POR DESENHO
 *
 * **Nao tem parametro para a raiz.** Vem de `HADES_RAIZ_SBOM` no servidor:
 *
 *	*`HADES_ARTEFACTOS` e' a fronteira dos binarios; `HADES_RAIZ_SBOM` e' a fronteira do codigo. Sao
 *	cofres diferentes, logo variaveis diferentes -- e nenhuma das duas e' escolhida pelo cliente.*
 *
 * **Nao tem `assinar()` nem `atestar()`.** A rota gera o inventario e devolve `assinado: false` e
 * `provenance: false` sempre.
 *
 *	*Um botao morto e' pior do que um botao ausente: o ausente nao mente.*
 */

import { agentFetch } from "@/lib/agenticClient";

/** Tipo de componente do inventario. */
// Os valores vem do backend (`internal/agent/compliance_provider.go`): MODULO_GO, PACOTE_NPM,
// BINARIO. **Nao inventar**: um teste que inventa o valor que o backend devolve nunca falha por
// causa do backend.
export type TipoDeComponente = "MODULO_GO" | "PACOTE_NPM" | "BINARIO";

/** Um componente do inventario. */
export interface ComponenteDoInventario {
  nome: string;
  versao: string;
  tipo: TipoDeComponente;
  licenca?: string;
  direto: boolean;
  origem: string;
  declarado_em: string;
}

/** O SBOM minimo. */
export interface InventarioDeSoftware {
  raiz: string;
  medido_em: string;
  componentes: ComponenteDoInventario[] | null;
  total: number;
  por_tipo: Record<string, number> | null;
  sem_licenca: number;
  ficheiros_lidos: string[] | null;
  nao_produzido: string[] | null;
  aviso: string;
}

/** Resposta de `sbom.generate`. */
export interface RelatorioDeSBOM {
  gerado: boolean;
  assinado: boolean;
  provenance: boolean;
  inventario: InventarioDeSoftware;
  assinatura_do_inventario: string[] | null;
  aviso_de_fronteira: string;
}

/** Pede o SBOM. Nao ha' parametros: a raiz e' do servidor. */
export async function gerarSBOM(): Promise<RelatorioDeSBOM> {
  return agentFetch<RelatorioDeSBOM>("/api/agent/v1/sbom/generate", {
    method: "POST",
    body: JSON.stringify({}),
  });
}

/**
 * Traduz o tipo de componente.
 *
 * Um valor desconhecido **falha alto** em vez de cair num "outro" silencioso.
 *
 *	*Um estado que so' o programador pode nomear invalido nao e' um estado: e' um campo de texto.*
 */
export function descreverTipoDeComponente(tipo: TipoDeComponente): string {
  switch (tipo) {
    case "MODULO_GO":
      return "modulo Go";
    case "PACOTE_NPM":
      return "pacote npm";
    case "BINARIO":
      return "binario vendorizado";
    default: {
      const inesperado: never = tipo;
      throw new Error(`tipo de componente desconhecido: ${String(inesperado)}`);
    }
  }
}

/**
 * Resume o inventario numa frase.
 *
 * **Nao diz "SBOM completo"** -- diz quantos e de onde.
 */
export function resumirInventario(relatorio: RelatorioDeSBOM): string {
  const inventario = relatorio.inventario;
  if (inventario.total === 0) {
    return "Nenhum componente encontrado nas fontes que este provider le.";
  }
  const tipos = inventario.por_tipo ?? {};
  const partes: string[] = [];
  for (const [tipo, quantidade] of Object.entries(tipos)) {
    partes.push(`${quantidade} ${descreverTipoDeComponente(tipo as TipoDeComponente)}`);
  }
  const detalhe = partes.length > 0 ? ` (${partes.join(", ")})` : "";
  return `${inventario.total} componente(s)${detalhe}; ` +
    `${inventario.sem_licenca} sem licenca declarada na origem.`;
}

/**
 * Diz o que o SBOM prova -- e o que nao prova.
 *
 *	*Um SBOM que inclui o que o programador supoe e' pior do que nenhum: quem le' acredita.*
 */
export function provaDoSBOM(relatorio: RelatorioDeSBOM): string {
  if (relatorio.assinado) {
    // Nunca deveria acontecer. Mas se acontecer, o painel tem de o dizer.
    return "ATENCAO: o servidor declarou que ASSINOU o inventario.";
  }
  if (relatorio.provenance) {
    return "ATENCAO: o servidor declarou PROVENANCE, que esta rota nao produz.";
  }
  if (!relatorio.gerado) {
    return "nao gerado";
  }
  const naoProduzido = relatorio.inventario.nao_produzido ?? [];
  const lacunas = naoProduzido.length > 0 ? ` Nao produzido: ${naoProduzido.join("; ")}.` : "";
  return "minimo: cobre go.mod e package.json; NAO inclui transitivas, notices, " +
    `provenance nem attestation.${lacunas}`;
}
