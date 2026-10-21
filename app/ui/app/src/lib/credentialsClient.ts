/**
 * FASE 28.7/28.8: cliente da rota `credentials.status`.
 *
 * O QUE ESTE FICHEIRO LIGA
 *
 * A E-1270 criou a rota no servidor e provou-a com 7 testes. Mas **nenhum painel a chamava** -- e o
 * defeito recorrente desta missao ja' se reabriu nove vezes assim:
 *
 *	*Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
 *	com o outro.*
 *
 * Este modulo e' o unico lugar do frontend que sabe o formato da rota. Se o backend mudar um nome de
 * campo, o TypeScript falha aqui e nao no ecra.
 *
 * O QUE ESTE CLIENTE NAO PODE FAZER, POR DESENHO
 *
 * A rota **nao aceita caminho**. O caminho vem de `HADES_KIT_CREDENCIAIS` no ambiente do servidor.
 * Este cliente nao tem -- e nao pode ganhar -- um parametro para o escolher:
 *
 *	*Um cliente que escolhe o caminho do cofre e' um cliente que le' o sistema de ficheiros.*
 *
 * E os segredos continuam nao lidos: o que chega aqui tem `conteudo_lido: false` para eles, e o tipo
 * `CredencialPublica` nao tem campo para o valor -- porque nao ha' valor para ter.
 */

import { agentFetch } from "@/lib/agenticClient";

/** Tipo de credencial, como o backend o classifica. */
export type TipoDeCredencial =
  | "CERTIFICADO_X509"
  | "PERFIL_DE_PROVISIONAMENTO"
  | "CHAVE_PRIVADA"
  | "KEYSTORE"
  | "SENHA_PROTEGIDA"
  | "CHECKSUMS"
  | "DOCUMENTACAO"
  | "DESCONHECIDA";

/**
 * Uma credencial, como a rota a devolve.
 *
 * Nao existe campo para o conteudo de um segredo. `conteudo_lido: false` significa que o servidor so'
 * fez `os.Stat` -- e o tipo nao permite representar outra coisa.
 */
export interface CredencialPublica {
  caminho_relativo: string;
  tipo: TipoDeCredencial;
  tamanho_bytes: number;
  modificado_em: string;
  conteudo_lido: boolean;
  fingerprint_sha256?: string;
  sujeito?: string;
  emissor?: string;
  valido_ate?: string;
  expirado: boolean;
  nota?: string;
}

/** Resposta de `credentials.status`. */
export interface InventarioDeCredenciais {
  somente_leitura: boolean;
  configurado: boolean;
  credenciais: CredencialPublica[];
  total: number;
  lidos: number;
  nao_lidos: number;
  certificados: number;
  expirados: number;
  aviso: string;
  aviso_de_fronteira: string;
}

/** Le' o inventario do kit configurado no servidor. */
export async function obterInventarioDeCredenciais(): Promise<InventarioDeCredenciais> {
  return agentFetch<InventarioDeCredenciais>("/api/agent/v1/credentials/status");
}

/**
 * Traduz o tipo para uma etiqueta legivel em pt-BR.
 *
 * Um `switch` exaustivo com `never`: se o backend acrescentar um tipo, o TypeScript falha aqui em vez
 * de mostrar uma string vazia ao utilizador.
 */
export function descreverTipoDeCredencial(tipo: TipoDeCredencial): string {
  switch (tipo) {
    case "CERTIFICADO_X509":
      return "Certificado";
    case "PERFIL_DE_PROVISIONAMENTO":
      return "Perfil de provisionamento";
    case "CHAVE_PRIVADA":
      return "Chave privada";
    case "KEYSTORE":
      return "Keystore";
    case "SENHA_PROTEGIDA":
      return "Senha protegida";
    case "CHECKSUMS":
      return "Checksums";
    case "DOCUMENTACAO":
      return "Documentacao";
    case "DESCONHECIDA":
      return "Desconhecida";
    default: {
      const inesperado: never = tipo;
      throw new Error(`tipo de credencial desconhecido: ${String(inesperado)}`);
    }
  }
}

/**
 * Diz se uma credencial e' um segredo.
 *
 * A lista e' positiva (os tipos que NAO sao segredo) e nao negativa: um tipo novo cai em "e' segredo"
 * ate' alguem decidir o contrario.
 *
 *	*Na duvida, nao se abre -- e na duvida, tambem nao se mostra.*
 */
export function eSegredo(tipo: TipoDeCredencial): boolean {
  return !(
    tipo === "CERTIFICADO_X509" ||
    tipo === "CHECKSUMS" ||
    tipo === "DOCUMENTACAO"
  );
}

/** Formata o tamanho sem inventar precisao que o ficheiro nao tem. */
export function tamanhoLegivelDeCredencial(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) {
    return "tamanho desconhecido";
  }
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

/**
 * Resume o inventario numa frase.
 *
 * Nunca diz "seguro", "aprovado" ou "pronto": conta o que foi medido.
 *
 *	*Um resumo que tranquiliza sem ter medido e' pior do que nenhum resumo.*
 */
export function resumirInventario(inventario: InventarioDeCredenciais): string {
  if (!inventario.configurado) {
    return "Kit de credenciais nao configurado neste servidor.";
  }
  if (inventario.total === 0) {
    return "Kit configurado, mas nenhuma credencial encontrada.";
  }
  const partes = [
    `${inventario.total} ficheiro(s)`,
    `${inventario.lidos} lido(s)`,
    `${inventario.nao_lidos} nao lido(s)`,
  ];
  if (inventario.certificados > 0) {
    partes.push(`${inventario.certificados} certificado(s)`);
  }
  if (inventario.expirados > 0) {
    partes.push(`${inventario.expirados} expirado(s)`);
  }
  return `${partes.join(", ")}.`;
}
