// Contexto da missão NA FORMA QUE A SUPERFÍCIE MÓVEL PODE MOSTRAR (D-9 / E-610).
//
// LISTA BRANCA, E É AQUI QUE A GARANTIA VIVE. O servidor devolve o `ContextBootstrap` inteiro,
// com `mission_id`, `project_id`, `organization_id`, `workspace` e referências de memória.
// Nenhum desses campos é conteúdo de trabalho: são topologia do servidor — e um caminho local
// revela a estrutura da máquina. A projecção abaixo **não os copia**, e é uma FUNÇÃO PURA para
// que a garantia seja testável no harness desta app (que não tem vitest nem react-test-renderer).
//
// A mesma disciplina existe em três sítios e por isso os três se podem comparar: o
// `ParaPreambulo` do servidor (E-597), o `MissionContextView` do desktop (E-608) e esta
// projecção. ***Uma garantia que se reescreve em cada cliente tem de ser uma garantia que se
// TESTA em cada cliente*** — pôr a lista branca no JSX deixa-a dependente de disciplina.

export type MissionContextView = {
  objective?: string;
  plan?: string[];
  checkpoints?: string[];
  /** Já vem REDIGIDO do servidor (E-597): sem identificadores, sem caminhos. */
  handoff?: string;
};

function textoOuNada(valor: unknown): string | undefined {
  return typeof valor === "string" && valor.trim() !== "" ? valor : undefined;
}

function listaDeTextos(valor: unknown): string[] | undefined {
  if (!Array.isArray(valor)) return undefined;
  const itens = valor.filter((item): item is string => typeof item === "string" && item.trim() !== "");
  return itens.length > 0 ? itens : undefined;
}

/**
 * projectMissionContext copia do payload do servidor APENAS os campos de trabalho.
 *
 * Aceita `unknown` de propósito: o payload chega da rede, e um tipo que prometesse a forma
 * certa estaria a confiar na rede. Entrada estranha devolve um objecto vazio em vez de explodir:
 * contexto indisponível não pode derrubar a superfície.
 */
export function projectMissionContext(payload: unknown): MissionContextView {
  if (payload === null || typeof payload !== "object") return {};
  const fonte = payload as Record<string, unknown>;
  const vista: MissionContextView = {};
  const objective = textoOuNada(fonte.objective);
  const plan = listaDeTextos(fonte.plan);
  const checkpoints = listaDeTextos(fonte.checkpoints);
  const handoff = textoOuNada(fonte.handoff);
  if (objective !== undefined) vista.objective = objective;
  if (plan !== undefined) vista.plan = plan;
  if (checkpoints !== undefined) vista.checkpoints = checkpoints;
  if (handoff !== undefined) vista.handoff = handoff;
  return vista;
}
