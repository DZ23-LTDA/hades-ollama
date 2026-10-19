/**
 * FASE 28.7 do adendo V3: o cliente do ESTADO de build.
 *
 * O QUE ESTE FICHEIRO FECHA
 *
 * O E-1313 entregou cinco rotas no servidor — `status`, `logs`, `artifacts`, `registos` e `cancel` —
 * e **nenhum painel as consumia**. Um rota que so' o `curl` conhece e' uma rota provada no servidor e
 * invisivel no produto.
 *
 *	*Uma rota que existe e nao e' consumida e' uma promessa do backend, nao uma funcionalidade do
 *	produto.*
 *
 * AS REGRAS QUE ESTE CLIENTE RESPEITA
 *
 *  1. **`agentFetch<T>` devolve o corpo descodificado e lanca em erro HTTP.** Nao ha' aqui um
 *     `Response` para inspecionar: o erro chega como excepcao, e o 409 do `cancel` e' um erro
 *     LEGITIMO que o painel tem de distinguir de uma falha.
 *  2. **O cliente NAO calcula estado.** `EstadoDeBuild` vem do servidor como texto. Um painel que
 *     mapeia codigos para o seu proprio enum cria uma segunda verdade.
 *  3. **O `nada_assinado`/`nada_publicado` vem do servidor e e' repassado.** Nao ha' aqui constantes
 *     `true` locais: se o servidor um dia devolver `false`, o painel tem de o mostrar.
 *  4. **`cancelavel` vem do servidor.** O botao de cancelar nao decide sozinho se pode agir.
 */

import { agentFetch } from "@/lib/agenticClient";

/** Estados que o servidor devolve. Os valores sao os do backend, nao um enum paralelo. */
export type EstadoDeBuild =
  | "NA_FILA"
  | "A_EXECUTAR"
  | "CONCLUIDO"
  | "FALHOU"
  | "CANCELADO";

/** Um artefacto medido pelo provider. Ausente nao e' o mesmo que vazio. */
export interface ArtefactoDeBuild {
  Caminho: string;
  Nome: string;
  TamanhoBytes: number;
  SHA256: string;
  Alvo: string;
  PidDoProcesso?: number;
  DuracaoMS?: number;
  SaidaDoComando?: string;
}

/** O resultado de um build, tal como o provider o mediu. */
export interface ResultadoDeBuild {
  Artefatos?: ArtefactoDeBuild[];
  NaoConstruidos?: string[];
  SemArtefacto?: boolean;
  Explicacao?: string;
}

/** O registo de um build. */
export interface RegistoDeBuild {
  id: string;
  alvo: string;
  raiz: string;
  destino: string;
  estado: EstadoDeBuild;
  iniciado_em: string;
  terminado_em?: string;
  resultado?: ResultadoDeBuild;
  erro?: string;
  saida_do_comando?: string;
  cancelavel: boolean;
  nada_assinado: boolean;
  nada_publicado: boolean;
}

/** Resposta de `GET /build/status/:id`. */
export interface RespostaDeEstadoDeBuild {
  build: RegistoDeBuild;
  nada_assinado: boolean;
  nada_publicado: boolean;
  nada_instalado: boolean;
  aviso_de_fronteira: string;
}

/** Resposta de `GET /build/registos`. */
export interface RespostaDeRegistosDeBuild {
  total: number;
  builds: RegistoDeBuild[];
  nada_assinado: boolean;
  nada_publicado: boolean;
  aviso_de_fronteira: string;
}

/** Resposta de `GET /build/logs/:id`. */
export interface RespostaDeSaidaDeBuild {
  id: string;
  estado: EstadoDeBuild;
  saida: string;
  erro?: string;
  truncada: boolean;
  aviso: string;
}

/** Resposta de `GET /build/artifacts/:id`. */
export interface RespostaDeArtefactosDeBuild {
  id: string;
  alvo: string;
  estado: EstadoDeBuild;
  artefactos: ArtefactoDeBuild[];
  produziu: boolean;
  explicacao?: string;
  nada_assinado: boolean;
  aviso: string;
}

/** Pedido de `POST /build/start`. */
export interface PedidoDeInicioDeBuild {
  raiz: string;
  alvo: string;
  destino?: string;
  dry_run?: boolean;
  timeout_segundos?: number;
  /**
   * `true` devolve 202 com o `build_id` e corre em segundo plano.
   *
   * E' o UNICO modo em que `cancel` tem um build por cancelar: se a rota so' respondesse no fim, o
   * botao de cancelar so' encontraria builds ja' terminados.
   */
  assincrono?: boolean;
}

/** Resposta de `POST /build/start`. */
export interface RespostaDeInicioDeBuild {
  construiu: boolean;
  nada_assinado: boolean;
  nada_publicado: boolean;
  alvo: string;
  plano: unknown;
  resultado?: ResultadoDeBuild;
  aviso_de_fronteira: string;
  build_id?: string;
  estado?: EstadoDeBuild;
}

/** Resposta de `POST /build/cancel/:id`. */
export interface RespostaDeCancelamentoDeBuild {
  id: string;
  estado: EstadoDeBuild;
  cancelado: boolean;
  aviso_de_fronteira?: string;
}

/**
 * Traduz o estado do backend para texto humano.
 *
 * Existe para o painel nao espalhar `switch` pelo JSX e para haver UM lugar que muda se o backend
 * acrescentar um estado.
 */
export function descreverEstadoDeBuild(estado: string): string {
  switch (estado) {
    case "NA_FILA":
      return "Na fila";
    case "A_EXECUTAR":
      return "A executar";
    case "CONCLUIDO":
      return "Concluído";
    case "FALHOU":
      return "Falhou";
    case "CANCELADO":
      return "Cancelado";
    default:
      // Um estado desconhecido e' mostrado como veio, nao escondido atras de "desconhecido": um
      // painel que esconde o que nao conhece adia o problema para quem o le'.
      return estado;
  }
}

/** Diz se o estado e' terminal. E' a unica regra que o painel usa para decidir se mostra acoes. */
export function estadoDeBuildTerminou(estado: string): boolean {
  return estado === "CONCLUIDO" || estado === "FALHOU" || estado === "CANCELADO";
}

/** Cor semantica do estado. Devolve um nome de token, nao um valor de CSS. */
export function corDoEstadoDeBuild(estado: string): string {
  switch (estado) {
    case "CONCLUIDO":
      return "green";
    case "FALHOU":
      return "red";
    case "CANCELADO":
      return "amber";
    case "A_EXECUTAR":
      return "blue";
    default:
      return "gray";
  }
}

/** Formata bytes sem inventar precisao. */
export function formatarBytes(n: number | undefined): string {
  if (n === undefined || Number.isNaN(n)) return "—";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KiB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MiB`;
}

/** Busca um build pelo ID. */
export async function obterEstadoDeBuild(
  id: string,
  init?: RequestInit,
): Promise<RespostaDeEstadoDeBuild> {
  return await agentFetch<RespostaDeEstadoDeBuild>(
    `/api/agent/v1/build/status/${encodeURIComponent(id)}`,
    init,
  );
}

/** Lista os builds deste processo. */
export async function listarBuilds(
  init?: RequestInit,
): Promise<RespostaDeRegistosDeBuild> {
  return await agentFetch<RespostaDeRegistosDeBuild>("/api/agent/v1/build/registos", init);
}

/** Le a saida guardada de um build. */
export async function obterSaidaDeBuild(
  id: string,
  init?: RequestInit,
): Promise<RespostaDeSaidaDeBuild> {
  return await agentFetch<RespostaDeSaidaDeBuild>(
    `/api/agent/v1/build/logs/${encodeURIComponent(id)}`,
    init,
  );
}

/** Le os artefactos medidos de um build. */
export async function obterArtefactosDeBuild(
  id: string,
  init?: RequestInit,
): Promise<RespostaDeArtefactosDeBuild> {
  return await agentFetch<RespostaDeArtefactosDeBuild>(
    `/api/agent/v1/build/artifacts/${encodeURIComponent(id)}`,
    init,
  );
}

/**
 * Inicia um build.
 *
 * Por omissao pede o modo ASSINCRONO: o servidor devolve 202 e um `build_id`, e a build corre em
 * segundo plano. O `BuildStatusPanel` mostra-a a partir da lista.
 *
 *	*Um `build.start` sincrono e' um pedido que so' responde quando ja' nao ha' nada a cancelar.*
 */
export async function iniciarBuild(
  pedido: PedidoDeInicioDeBuild,
  init?: RequestInit,
): Promise<RespostaDeInicioDeBuild> {
  return await agentFetch<RespostaDeInicioDeBuild>("/api/agent/v1/build/start", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ assincrono: true, ...pedido }),
    ...init,
  });
}

/**
 * Pede o cancelamento de um build.
 *
 * O 409 do backend NAO e' tratado aqui como sucesso nem como falha generica: sobe como excepcao,
 * porque quem sabe distinguir "ja' tinha acabado" de "nao consegui cancelar" e' o painel, que tem o
 * estado anterior para comparar.
 */
export async function cancelarBuild(
  id: string,
  init?: RequestInit,
): Promise<RespostaDeCancelamentoDeBuild> {
  return await agentFetch<RespostaDeCancelamentoDeBuild>(
    `/api/agent/v1/build/cancel/${encodeURIComponent(id)}`,
    {
      method: "POST",
      ...init,
    },
  );
}
