/**
 * FASE 28.7: cliente da rota `project.detect`.
 *
 * O QUE ESTA ROTA E' -- E O QUE NAO E'
 *
 * Ela **le' nomes de ficheiros e o conteudo dos manifests** para classificar a stack e listar alvos.
 * Nao escreve, nao instala, nao constroi e nao assina.
 *
 * A raiz vem de `HADES_RAIZ_PROJECTO` no servidor. O cliente **nao a escolhe**:
 *
 *	*Um cliente que escolhe a raiz dos binarios e' um cliente que decide o que corre.* (E-1273)
 *
 * OS ALVOS SAO SUGESTAO
 *
 *	*Um alvo derivado de um ficheiro nao prova que a toolchain exista.*
 *
 * Um `go.mod` diz que o projecto e' Go; nao diz que o Go esta' instalado. Quem responde a isso e'
 * `toolchain.doctor`.
 *
 * OS VALORES SAO OS DO BACKEND
 *
 * `go`, `node`, `expo`, `python`, `rust`, `dotnet`, `flutter` para stack; `go-modules`, `npm`, `pnpm`,
 * `yarn`, `bun`, `poetry`, `uv`, `cargo`, `nuget`, `pub` para gestor. **Nao inventar**: um teste que
 * inventa o valor que o backend devolve nunca falha por causa do backend.
 */

import { agentFetch } from "@/lib/agenticClient";

/** Stack reconhecida pelo detector. Os valores sao os do backend. */
export type StackDeProjecto =
  | "go"
  | "node"
  | "expo"
  | "python"
  | "rust"
  | "dotnet"
  | "flutter";

/** Gestor de pacotes reconhecido. Os valores sao os do backend. */
export type GestorDePacotes =
  | "go-modules"
  | "npm"
  | "pnpm"
  | "yarn"
  | "bun"
  | "poetry"
  | "uv"
  | "cargo"
  | "nuget"
  | "pub";

/** O que o detector encontrou. */
export interface ProjetoDetectado {
  raiz: string;
  stacks: StackDeProjecto[] | null;
  gestores: GestorDePacotes[] | null;
  alvos: string[] | null;
  ficheiros_nativos?: string[] | null;
  manifests: string[] | null;
  /** `omitempty` no contrato: uma lista vazia desaparece do JSON. */
  avisos?: string[] | null;
}

/** Resposta de `project.detect`. */
export interface RelatorioDeDeteccao {
  somente_leitura: boolean;
  detectado: ProjetoDetectado;
  aviso_de_fronteira: string;
}

/** Pede a deteccao. Nao ha' parametros: a raiz e' do servidor. */
export async function detectarProjecto(): Promise<RelatorioDeDeteccao> {
  return agentFetch<RelatorioDeDeteccao>("/api/agent/v1/project/detect", {
    method: "POST",
    body: JSON.stringify({}),
  });
}

/**
 * Traduz a stack para portugues.
 *
 * Um valor desconhecido **falha alto** -- nao cai num "outro" silencioso.
 */
export function descreverStack(stack: StackDeProjecto): string {
  switch (stack) {
    case "go":
      return "Go";
    case "node":
      return "Node";
    case "expo":
      return "Expo / React Native";
    case "python":
      return "Python";
    case "rust":
      return "Rust";
    case "dotnet":
      return ".NET";
    case "flutter":
      return "Flutter / Dart";
    default: {
      const inesperado: never = stack;
      throw new Error(`stack desconhecida: ${String(inesperado)}`);
    }
  }
}

/**
 * Traduz o gestor de pacotes.
 *
 * Um valor desconhecido **falha alto**.
 */
export function descreverGestor(gestor: GestorDePacotes): string {
  switch (gestor) {
    case "go-modules":
      return "Go modules";
    case "npm":
      return "npm";
    case "pnpm":
      return "pnpm";
    case "yarn":
      return "yarn";
    case "bun":
      return "bun";
    case "poetry":
      return "Poetry";
    case "uv":
      return "uv";
    case "cargo":
      return "Cargo";
    case "nuget":
      return "NuGet";
    case "pub":
      return "pub";
    default: {
      const inesperado: never = gestor;
      throw new Error(`gestor de pacotes desconhecido: ${String(inesperado)}`);
    }
  }
}

/**
 * Resume o que foi detectado.
 *
 * **Nunca diz "pronto para construir".** Diz o que encontrou.
 */
export function resumirDeteccao(relatorio: RelatorioDeDeteccao): string {
  const detectado = relatorio.detectado;
  const stacks = detectado.stacks ?? [];
  const alvos = detectado.alvos ?? [];

  if (stacks.length === 0) {
    // Nao deveria acontecer: o backend recusa com 409 quando nao reconhece stack. Mas se acontecer,
    // o painel nao pode inventar uma stack.
    return "Nenhuma stack reconhecida.";
  }

  const nomes = stacks.map(descreverStack).join(", ");
  const contagemAlvos = alvos.length === 1 ? "1 alvo" : `${alvos.length} alvos`;
  return `${nomes} -- ${contagemAlvos}.`;
}

/**
 * Diz o que a deteccao prova -- e o que nao prova.
 *
 *	*Um alvo derivado de um ficheiro nao prova que a toolchain exista.*
 */
export function provaDaDeteccao(relatorio: RelatorioDeDeteccao): string {
  const detectado = relatorio.detectado;
  const alvos = detectado.alvos ?? [];
  const gestores = detectado.gestores ?? [];

  if (alvos.length === 0) {
    return "Nenhum alvo derivado: nao ha' como planear um build a partir desta deteccao.";
  }
  if (gestores.length === 0) {
    return "Alvos derivados dos ficheiros presentes, mas NENHUM gestor de pacotes reconhecido -- " +
      "as dependencias nao podem ser resolvidas.";
  }
  return "Alvos derivados dos ficheiros presentes. Isto NAO prova que a toolchain esteja " +
    "instalada: consulte `toolchain.doctor` antes de planear um build.";
}
