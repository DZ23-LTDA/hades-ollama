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
