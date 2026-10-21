// FASE 28.7 + 28.8 do adendo V3: cliente do preflight de toolchain.
//
// Consome as rotas `/api/agent/v1/toolchain/*` criadas no servidor. O contrato espelha o do
// backend de proposito: *dois contratos para a mesma pergunta divergem no primeiro dia em que
// alguem muda um deles.*
//
// Nada aqui instala, altera ou executa trabalho de utilizador. Todas as chamadas sao GET.
// *Um preflight que muda o sistema nao e' preflight: e' instalacao disfarcada.*

import { agentFetch } from "@/lib/agenticClient";

/** Os tres estados do plano. `incompleto` e' um estado de primeira classe, nao um erro. */
export type VeredictoToolchain = "pronto" | "bloqueado" | "incompleto";

/** Um probe individual: uma ferramenta exigida e o que o host respondeu sobre ela. */
export interface ToolchainProbe {
  nome: string;
  estado: string;
  obrigatorio: boolean;
  caminho?: string;
  versao?: string;
  motivo: string;
  detalhe: string;
}

/** O plano de um alvo de build. */
export interface ToolchainPlano {
  host: string;
  goos: string;
  alvo: string;
  pronto: boolean;
  veredicto: VeredictoToolchain;
  probes: ToolchainProbe[];
  blockers: string[];
  trabalho_independente: string[];
}

/** Um alvo no catalogo. */
export interface ToolchainAlvo {
  nome: string;
  descricao: string;
  host_necessario?: string;
  exigencias: string[];
  trabalho_independente?: string[];
}

export interface ToolchainCatalogo {
  host: string;
  goos: string;
  alvos: ToolchainAlvo[];
  count: number;
}

/** O resumo de um alvo no doctor. */
export interface ToolchainResumoAlvo {
  alvo: string;
  pronto: boolean;
  veredicto: VeredictoToolchain | "erro";
  exigencias_bloqueantes: number;
}

export interface ToolchainDoctor {
  host: string;
  goos: string;
  alvos: ToolchainResumoAlvo[];
  prontos: number;
  bloqueados: number;
  incompletos: number;
  total: number;
  resumo: string;
}

/** Catalogo de alvos que este HADES sabe construir. */
export function fetchToolchainTargets(): Promise<ToolchainCatalogo> {
  return agentFetch<ToolchainCatalogo>("/api/agent/v1/toolchain/targets");
}

/** Plano de UM alvo. Alvo desconhecido faz o servidor devolver 404 -- e' o esperado. */
export function fetchToolchainPlan(alvo: string): Promise<ToolchainPlano> {
  return agentFetch<ToolchainPlano>(
    `/api/agent/v1/toolchain/plan/${encodeURIComponent(alvo)}`,
  );
}

/** Plano de TODOS os alvos, com resumo em texto pronto a mostrar. */
export function fetchToolchainDoctor(): Promise<ToolchainDoctor> {
  return agentFetch<ToolchainDoctor>("/api/agent/v1/toolchain/doctor");
}

/**
 * Cor que representa o veredicto.
 *
 * `incompleto` NAO usa a cor de erro: ferramenta presente sem health funcional nao e' o mesmo
 * problema que ferramenta ausente, e pintar os dois de vermelho faria o utilizador procurar
 * o que ja' tem. *Uma tela que grita por tudo deixa de ser ouvida quando grita por algo.*
 */
export function corDoVeredicto(veredicto: VeredictoToolchain | "erro"): string {
  switch (veredicto) {
    case "pronto":
      return "text-emerald-400";
    case "incompleto":
      return "text-amber-400";
    case "erro":
      return "text-red-400";
    case "bloqueado":
      return "text-red-400";
  }
}

/** Etiqueta legivel do estado de um probe, para o utilizador final. */
export function rotuloDoEstado(estado: string): string {
  switch (estado) {
    case "PRESENT_COMPATIBLE_HEALTHY":
      return "Pronta e verificada";
    case "PRESENT_COMPATIBLE_UNVERIFIED":
      return "Presente, falta verificar";
    case "PRESENT_OUTDATED":
      return "Versao abaixo do minimo";
    case "PRESENT_BROKEN":
      return "Presente mas nao funciona";
    case "PRESENT_CONFLICTING":
      return "Presente, em conflito";
    case "PRESENT_UNTRUSTED_ORIGIN":
      return "Origem nao confiavel";
    case "PRESENT_EXTERNAL_OWNERSHIP":
      return "Pertence a outro produto";
    case "ABSENT":
      return "Ausente";
    case "UNKNOWN":
      return "Desconhecido";
    default:
      return estado;
  }
}

/**
 * `UNKNOWN` nunca e' `ABSENT`.
 *
 * A distincao e' o ponto: *"nao consegui determinar" nao e' "nao existe", e tratar os dois
 * como iguais faria o HADES instalar uma ferramenta que a maquina ja' tem.*
 */
export function estadoEhAusente(estado: string): boolean {
  return estado === "ABSENT";
}

// ---------------------------------------------------------------------------------------------
// FASE 28.6-A / 28.7: cliente do INSTALADOR MODULAR.
//
// Mesmo ficheiro, mesmo dominio: o instalador e o toolchain respondem a mesma pergunta --
// "o que falta nesta maquina para o HADES fazer o que prometi?". *Dois clientes para a mesma
// pergunta divergem no primeiro dia em que alguem muda um deles.*
//
// Nada aqui instala. A rota `/install/plan/:perfil` devolve `executa_algo: false` e este
// cliente nao tem funcao de instalar. *Um botao "Instalar" na UI de diagnostico mentiria:
// instalar exige approval e superficie propria.*
// ---------------------------------------------------------------------------------------------

/** Um perfil de instalacao, tal como o servidor o declara. */
export interface InstallPerfil {
  id: string;
  descricao: string;
  pacotes: string[];
  tamanho_mb: number;
  escolhe_pacotes: boolean;
}

/** Um componente do catalogo, sem decisao tomada. */
export interface InstallPacote {
  id: string;
  nome: string;
  dependencias: string[] | null;
  tamanho_mb: number;
  licenca: string;
  redistribuivel: boolean;
  exige_aprovacao: boolean;
  impacto: string[] | null;
}

/** Resposta de `install/profiles`. */
export interface InstallCatalogo {
  host: string;
  goos: string;
  perfis: InstallPerfil[];
  pacotes: InstallPacote[];
  /** Avisos da auto-auditoria do catalogo. Lista vazia e' o estado saudavel. */
  avisos: string[] | null;
  count: number;
}

/** Um pacote com a decisao ja' tomada para ESTA maquina. */
export interface InstallItem {
  pacote_id: string;
  nome: string;
  /** `INSTALAR`, `REUTILIZAR`, `MANTER`, `PULAR` ou `APROVAR`. */
  acao: string;
  motivo: string;
  requer_aprovacao: boolean;
  tamanho_mb: number;
}

/** Resposta de `install/plan/:perfil`. */
export interface InstallPlano {
  perfil: string;
  host: string;
  goos: string;
  itens: InstallItem[];
  tamanho_total_mb: number;
  downloads_evitados: number;
  aprovacoes_necessarias: number;
  /** Lista de impedimentos. O servidor garante lista, nunca `null`. */
  bloqueios: string[] | null;
  pronto: boolean;
  resumo: string;
  /** Sempre `false`: a rota planeia, nao executa. */
  executa_algo: boolean;
}

/** Catalogo de perfis e pacotes. */
export function fetchInstallCatalogo(): Promise<InstallCatalogo> {
  return agentFetch<InstallCatalogo>("/api/agent/v1/install/profiles");
}

/** Plano para UM perfil, com descoberta real do host. */
export function fetchInstallPlano(perfil: string): Promise<InstallPlano> {
  return agentFetch<InstallPlano>(`/api/agent/v1/install/plan/${encodeURIComponent(perfil)}`);
}

/**
 * Rotulo legivel de uma acao do plano.
 *
 * `PULAR` e' o rotulo de `UNKNOWN`: *dizer "ausente" do que nao se conseguiu avaliar faria o
 * utilizador instalar por cima do que ja' tem.*
 */
export function rotuloDaAcao(acao: string): string {
  switch (acao) {
    case "INSTALAR":
      return "Instalar";
    case "REUTILIZAR":
      return "Reutilizar o que ja' existe";
    case "MANTER":
      return "Manter";
    case "PULAR":
      return "Nao avaliado";
    case "APROVAR":
      return "Requer aprovacao";
    default:
      return acao;
  }
}

/**
 * Cor de uma acao.
 *
 * `PULAR` usa ambar, nao vermelho: nao saber nao e' falhar. *Uma tela que pinta a duvida de
 * vermelho ensina o utilizador a ignorar o vermelho quando ele for real.*
 */
export function corDaAcao(acao: string): string {
  switch (acao) {
    case "INSTALAR":
      return "text-sky-400";
    case "REUTILIZAR":
    case "MANTER":
      return "text-emerald-400";
    case "PULAR":
      return "text-amber-400";
    case "APROVAR":
      return "text-violet-400";
    default:
      return "text-neutral-400";
  }
}