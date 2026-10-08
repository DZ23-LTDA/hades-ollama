// Pure, immutable data model and operations for the visual automation flow
// editor (n8n/dify-style node graph). Kept free of React/DOM so every rule —
// connection validity, cycle prevention, execution order — is unit-testable.
//
// A flow is a directed graph: one or more trigger nodes feed action/condition
// nodes through edges. Condition nodes branch on named ports ("true"/"false").
// The graph is plain JSON and serializes directly (no hidden state).

export type FlowNodeKind = "trigger" | "action" | "condition";

export interface FlowNode {
  id: string;
  kind: FlowNodeKind;
  /** Concrete type, e.g. "trigger.webhook", "action.mission", "action.connector", "condition.if". */
  type: string;
  label: string;
  x: number;
  y: number;
  config: Record<string, unknown>;
}

export interface FlowEdge {
  id: string;
  /** Source node id. */
  from: string;
  /** Target node id. */
  to: string;
  /** Output port on the source node (condition nodes use "true"/"false"). */
  fromPort?: string;
}

export interface FlowGraph {
  id: string;
  name: string;
  nodes: FlowNode[];
  edges: FlowEdge[];
}

export interface ConnectResult {
  graph: FlowGraph;
  /** Human-readable pt-BR reason when the connection was rejected; null on success. */
  error: string | null;
}

export function emptyFlow(id: string, name: string): FlowGraph {
  return { id, name, nodes: [], edges: [] };
}

export function addNode(graph: FlowGraph, node: FlowNode): FlowGraph {
  if (graph.nodes.some((n) => n.id === node.id)) {
    return graph;
  }
  return { ...graph, nodes: [...graph.nodes, node] };
}

export function removeNode(graph: FlowGraph, nodeId: string): FlowGraph {
  if (!graph.nodes.some((n) => n.id === nodeId)) {
    return graph;
  }
  return {
    ...graph,
    nodes: graph.nodes.filter((n) => n.id !== nodeId),
    edges: graph.edges.filter((e) => e.from !== nodeId && e.to !== nodeId),
  };
}

export function moveNode(graph: FlowGraph, nodeId: string, x: number, y: number): FlowGraph {
  return {
    ...graph,
    nodes: graph.nodes.map((n) => (n.id === nodeId ? { ...n, x, y } : n)),
  };
}

export function updateNodeConfig(
  graph: FlowGraph,
  nodeId: string,
  patch: Record<string, unknown>,
): FlowGraph {
  return {
    ...graph,
    nodes: graph.nodes.map((n) =>
      n.id === nodeId ? { ...n, config: { ...n.config, ...patch } } : n,
    ),
  };
}

/**
 * Connects two nodes, enforcing every structural rule fail-closed. Returns the
 * (possibly unchanged) graph plus a pt-BR reason when the edge was rejected, so
 * the UI never silently drops a user action nor creates an invalid flow.
 */
export function connect(
  graph: FlowGraph,
  from: string,
  to: string,
  fromPort?: string,
): ConnectResult {
  const source = graph.nodes.find((n) => n.id === from);
  const target = graph.nodes.find((n) => n.id === to);
  if (!source || !target) {
    return { graph, error: "Nó de origem ou destino não existe." };
  }
  if (from === to) {
    return { graph, error: "Um nó não pode se conectar a si mesmo." };
  }
  if (target.kind === "trigger") {
    return { graph, error: "Um gatilho não pode receber conexões de entrada." };
  }
  if (graph.edges.some((e) => e.from === from && e.to === to && e.fromPort === fromPort)) {
    return { graph, error: "Essa conexão já existe." };
  }
  const candidate: FlowGraph = {
    ...graph,
    edges: [
      ...graph.edges,
      { id: `edge_${from}_${to}_${fromPort ?? "out"}`, from, to, fromPort },
    ],
  };
  if (hasCycle(candidate)) {
    return { graph, error: "Essa conexão criaria um ciclo no fluxo." };
  }
  return { graph: candidate, error: null };
}

export function disconnect(graph: FlowGraph, edgeId: string): FlowGraph {
  return { ...graph, edges: graph.edges.filter((e) => e.id !== edgeId) };
}

/** Detects a directed cycle via depth-first search over the edge list. */
export function hasCycle(graph: FlowGraph): boolean {
  const adjacency = new Map<string, string[]>();
  for (const node of graph.nodes) {
    adjacency.set(node.id, []);
  }
  for (const edge of graph.edges) {
    adjacency.get(edge.from)?.push(edge.to);
  }
  const VISITING = 1;
  const DONE = 2;
  const state = new Map<string, number>();

  const walk = (nodeId: string): boolean => {
    state.set(nodeId, VISITING);
    for (const next of adjacency.get(nodeId) ?? []) {
      const seen = state.get(next);
      if (seen === VISITING) {
        return true;
      }
      if (seen !== DONE && walk(next)) {
        return true;
      }
    }
    state.set(nodeId, DONE);
    return false;
  };

  for (const node of graph.nodes) {
    if (!state.has(node.id) && walk(node.id)) {
      return true;
    }
  }
  return false;
}

/**
 * Returns the nodes in execution order (triggers first), or null when the graph
 * cannot be linearized (a cycle). Uses Kahn's algorithm on in-degrees.
 */
export function topologicalOrder(graph: FlowGraph): FlowNode[] | null {
  const indegree = new Map<string, number>();
  const byId = new Map<string, FlowNode>();
  for (const node of graph.nodes) {
    indegree.set(node.id, 0);
    byId.set(node.id, node);
  }
  for (const edge of graph.edges) {
    if (indegree.has(edge.to)) {
      indegree.set(edge.to, (indegree.get(edge.to) ?? 0) + 1);
    }
  }
  // Seed with zero in-degree nodes, triggers before everything else so the
  // order reads naturally from the entry points.
  const queue = graph.nodes
    .filter((n) => (indegree.get(n.id) ?? 0) === 0)
    .sort((a, b) => (a.kind === "trigger" ? -1 : 0) - (b.kind === "trigger" ? -1 : 0))
    .map((n) => n.id);

  const ordered: FlowNode[] = [];
  const adjacency = new Map<string, string[]>();
  for (const edge of graph.edges) {
    const list = adjacency.get(edge.from) ?? [];
    list.push(edge.to);
    adjacency.set(edge.from, list);
  }
  while (queue.length > 0) {
    const id = queue.shift() as string;
    const node = byId.get(id);
    if (node) {
      ordered.push(node);
    }
    for (const next of adjacency.get(id) ?? []) {
      const remaining = (indegree.get(next) ?? 0) - 1;
      indegree.set(next, remaining);
      if (remaining === 0) {
        queue.push(next);
      }
    }
  }
  return ordered.length === graph.nodes.length ? ordered : null;
}

/**
 * Validates a flow for execution. Returns a list of pt-BR problems; an empty
 * list means the flow is runnable. Never throws.
 */
export function validateFlow(graph: FlowGraph): string[] {
  const problems: string[] = [];
  const triggers = graph.nodes.filter((n) => n.kind === "trigger");
  if (graph.nodes.length === 0) {
    problems.push("O fluxo está vazio: adicione um gatilho para começar.");
    return problems;
  }
  if (triggers.length === 0) {
    problems.push("O fluxo precisa de pelo menos um gatilho de entrada.");
  }
  if (hasCycle(graph)) {
    problems.push("O fluxo tem um ciclo: remova a conexão que volta para um nó anterior.");
  }
  // Non-trigger nodes with no incoming edge are unreachable.
  const hasIncoming = new Set(graph.edges.map((e) => e.to));
  for (const node of graph.nodes) {
    if (node.kind !== "trigger" && !hasIncoming.has(node.id)) {
      problems.push(`O nó "${node.label}" está solto (sem conexão de entrada).`);
    }
  }
  return problems;
}

export interface CompiledSchedule {
  objective: string;
  interval_seconds?: number;
  webhook_secret_env?: string;
}

export interface CompileResult {
  /** The schedule to create, or null when the flow cannot run as-is. */
  schedule: CompiledSchedule | null;
  /** Blocking problems in pt-BR (empty when schedule is non-null). */
  errors: string[];
  /** Node types not recognized by the compiler (normally empty). */
  unsupported: string[];
}

interface StepResult {
  text?: string;
  error?: string;
}

// describeFlowStep turns an action/condition node into one pt-BR instruction the
// agent executes with its real tools (HTTP, connectors), or an error when the
// node is underspecified. This is how non-mission nodes actually run: they
// become explicit, ordered steps of the mission objective — not a fake label.
function describeFlowStep(node: FlowNode): StepResult {
  switch (node.type) {
    case "action.mission": {
      const objective = String(node.config.objective ?? "").trim();
      if (!objective) return { error: 'Preencha o objetivo da ação "Rodar missão".' };
      return { text: `Rodar missão: ${objective}` };
    }
    case "action.http": {
      const url = String(node.config.url ?? "").trim();
      if (!url) return { error: 'Preencha a URL da ação "Requisição HTTP".' };
      const method = String(node.config.method ?? "GET").trim().toUpperCase() || "GET";
      return { text: `Fazer uma requisição ${method} para ${url}` };
    }
    case "action.connector": {
      const connectorId = String(node.config.connectorId ?? "").trim();
      if (!connectorId) return { error: 'Escolha o conector da ação "Chamar conector".' };
      const operation = String(node.config.operation ?? "").trim();
      return {
        text: operation
          ? `Chamar o conector "${connectorId}", operação "${operation}"`
          : `Chamar o conector "${connectorId}"`,
      };
    }
    case "condition.if": {
      const expression = String(node.config.expression ?? "").trim();
      return {
        text: expression
          ? `Avaliar a condição "${expression}": se verdadeira, prosseguir; caso contrário, encerrar o fluxo`
          : "Avaliar a condição configurada: se verdadeira, prosseguir; caso contrário, encerrar",
      };
    }
    default:
      return { text: node.label };
  }
}

/**
 * Compiles a flow into the schedule the backend runs: one trigger (interval or
 * webhook) driving a mission objective. A single mission compiles to its plain
 * objective; a multi-node flow compiles to an ordered, numbered plan the agent
 * executes step by step with its real tools (HTTP, connectors). Nothing is
 * faked: every valid node becomes a real instruction.
 */
export function compileFlow(graph: FlowGraph): CompileResult {
  const errors = validateFlow(graph);
  if (errors.length > 0) {
    return { schedule: null, errors, unsupported: [] };
  }

  const triggers = graph.nodes.filter((n) => n.kind === "trigger");
  if (triggers.length !== 1) {
    return {
      schedule: null,
      errors: ["A execução por agendamento aceita exatamente um gatilho por fluxo."],
      unsupported: [],
    };
  }
  const trigger = triggers[0];

  const order = topologicalOrder(graph) ?? graph.nodes;
  const actionNodes = order.filter((n) => n.kind !== "trigger");
  if (!actionNodes.some((n) => n.type === "action.mission")) {
    return {
      schedule: null,
      errors: ['O fluxo precisa de uma ação "Rodar missão" para ser executado por agendamento.'],
      unsupported: [],
    };
  }

  const steps: string[] = [];
  for (const node of actionNodes) {
    const step = describeFlowStep(node);
    if (step.error) {
      return { schedule: null, errors: [step.error], unsupported: [] };
    }
    if (step.text) steps.push(step.text);
  }

  // One lone mission keeps its plain objective (simplest case); any richer flow
  // becomes an ordered plan executed in sequence by the agent.
  let objective: string;
  if (actionNodes.length === 1 && actionNodes[0].type === "action.mission") {
    objective = String(actionNodes[0].config.objective ?? "").trim();
  } else {
    objective =
      "Execute este fluxo de automação, seguindo os passos na ordem:\n" +
      steps.map((s, i) => `${i + 1}. ${s}`).join("\n");
  }

  const unsupported: string[] = [];

  const schedule: CompiledSchedule = { objective };
  if (trigger.type === "trigger.interval") {
    const seconds = Number(trigger.config.seconds);
    schedule.interval_seconds = Number.isFinite(seconds) && seconds > 0 ? Math.floor(seconds) : 3600;
  } else if (trigger.type === "trigger.webhook") {
    const secretEnv = String(trigger.config.secretEnv ?? "").trim();
    if (!secretEnv) {
      return {
        schedule: null,
        errors: ["Informe a variável de ambiente com o segredo do gatilho de webhook."],
        unsupported,
      };
    }
    schedule.webhook_secret_env = secretEnv;
    schedule.interval_seconds = 0;
  }
  return { schedule, errors: [], unsupported };
}
