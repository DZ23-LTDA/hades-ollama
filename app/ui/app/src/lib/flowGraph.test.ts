import { describe, it, expect } from "vitest";
import {
  emptyFlow,
  addNode,
  removeNode,
  moveNode,
  updateNodeConfig,
  connect,
  disconnect,
  hasCycle,
  topologicalOrder,
  validateFlow,
  compileFlow,
  type FlowGraph,
  type FlowNode,
} from "./flowGraph";

const node = (id: string, kind: FlowNode["kind"], x = 0, y = 0): FlowNode => ({
  id,
  kind,
  type: `${kind}.test`,
  label: id,
  x,
  y,
  config: {},
});

function sample(): FlowGraph {
  let g = emptyFlow("f1", "Fluxo");
  g = addNode(g, node("t", "trigger"));
  g = addNode(g, node("a", "action"));
  g = addNode(g, node("b", "action"));
  return g;
}

describe("flowGraph nodes", () => {
  it("adds nodes and ignores duplicates", () => {
    let g = emptyFlow("f1", "Fluxo");
    g = addNode(g, node("t", "trigger"));
    g = addNode(g, node("t", "trigger"));
    expect(g.nodes).toHaveLength(1);
  });

  it("removes a node and its connected edges", () => {
    let g = sample();
    g = connect(g, "t", "a").graph;
    g = connect(g, "a", "b").graph;
    g = removeNode(g, "a");
    expect(g.nodes.map((n) => n.id)).toEqual(["t", "b"]);
    expect(g.edges).toHaveLength(0);
  });

  it("moves a node without mutating the input", () => {
    const g = sample();
    const moved = moveNode(g, "a", 120, 40);
    expect(moved.nodes.find((n) => n.id === "a")).toMatchObject({ x: 120, y: 40 });
    expect(g.nodes.find((n) => n.id === "a")).toMatchObject({ x: 0, y: 0 });
  });

  it("merges node config immutably", () => {
    let g = sample();
    g = updateNodeConfig(g, "a", { url: "https://x" });
    g = updateNodeConfig(g, "a", { method: "POST" });
    expect(g.nodes.find((n) => n.id === "a")?.config).toEqual({
      url: "https://x",
      method: "POST",
    });
  });
});

describe("flowGraph connect rules", () => {
  it("connects valid nodes", () => {
    const { graph, error } = connect(sample(), "t", "a");
    expect(error).toBeNull();
    expect(graph.edges).toHaveLength(1);
  });

  it("rejects self-connection", () => {
    const { error } = connect(sample(), "a", "a");
    expect(error).toMatch(/si mesmo/);
  });

  it("rejects an incoming edge into a trigger", () => {
    const { error } = connect(sample(), "a", "t");
    expect(error).toMatch(/gatilho/i);
  });

  it("rejects an unknown node", () => {
    const { error } = connect(sample(), "t", "zzz");
    expect(error).toMatch(/não existe/);
  });

  it("rejects a duplicate edge", () => {
    let g = sample();
    g = connect(g, "t", "a").graph;
    const { error } = connect(g, "t", "a");
    expect(error).toMatch(/já existe/);
  });

  it("rejects an edge that would create a cycle", () => {
    let g = sample();
    g = connect(g, "t", "a").graph;
    g = connect(g, "a", "b").graph;
    const { error, graph } = connect(g, "b", "a");
    expect(error).toMatch(/ciclo/);
    expect(graph.edges).toHaveLength(2);
  });

  it("allows parallel edges through distinct condition ports", () => {
    let g = emptyFlow("f1", "Fluxo");
    g = addNode(g, node("t", "trigger"));
    g = addNode(g, node("c", "condition"));
    g = addNode(g, node("a", "action"));
    g = connect(g, "t", "c").graph;
    const first = connect(g, "c", "a", "true");
    expect(first.error).toBeNull();
    const second = connect(first.graph, "c", "a", "false");
    expect(second.error).toBeNull();
    expect(second.graph.edges.filter((e) => e.from === "c")).toHaveLength(2);
  });
});

describe("flowGraph disconnect / cycles / order", () => {
  it("disconnects by edge id", () => {
    let g = sample();
    g = connect(g, "t", "a").graph;
    const edgeId = g.edges[0].id;
    g = disconnect(g, edgeId);
    expect(g.edges).toHaveLength(0);
  });

  it("detects cycles directly", () => {
    const g: FlowGraph = {
      id: "f",
      name: "c",
      nodes: [node("a", "action"), node("b", "action")],
      edges: [
        { id: "e1", from: "a", to: "b" },
        { id: "e2", from: "b", to: "a" },
      ],
    };
    expect(hasCycle(g)).toBe(true);
  });

  it("returns a topological order with triggers first", () => {
    let g = sample();
    g = connect(g, "t", "a").graph;
    g = connect(g, "a", "b").graph;
    const order = topologicalOrder(g);
    expect(order?.map((n) => n.id)).toEqual(["t", "a", "b"]);
  });

  it("returns null topological order for a cyclic graph", () => {
    const g: FlowGraph = {
      id: "f",
      name: "c",
      nodes: [node("a", "action"), node("b", "action")],
      edges: [
        { id: "e1", from: "a", to: "b" },
        { id: "e2", from: "b", to: "a" },
      ],
    };
    expect(topologicalOrder(g)).toBeNull();
  });
});

describe("flowGraph validation", () => {
  it("flags an empty flow", () => {
    expect(validateFlow(emptyFlow("f", "x"))).toEqual([
      "O fluxo está vazio: adicione um gatilho para começar.",
    ]);
  });

  it("flags a missing trigger", () => {
    let g = emptyFlow("f", "x");
    g = addNode(g, node("a", "action"));
    expect(validateFlow(g).some((p) => /gatilho de entrada/.test(p))).toBe(true);
  });

  it("flags an orphan (disconnected) action", () => {
    let g = sample();
    g = connect(g, "t", "a").graph;
    // "b" has no incoming edge
    expect(validateFlow(g).some((p) => p.includes('"b"'))).toBe(true);
  });

  it("passes a fully connected flow", () => {
    let g = sample();
    g = connect(g, "t", "a").graph;
    g = connect(g, "a", "b").graph;
    expect(validateFlow(g)).toEqual([]);
  });
});

describe("compileFlow", () => {
  const mission = (id: string, objective: string): FlowNode => ({
    id,
    kind: "action",
    type: "action.mission",
    label: "Rodar missão",
    x: 0,
    y: 0,
    config: { objective },
  });
  const interval = (id: string, seconds: number): FlowNode => ({
    id,
    kind: "trigger",
    type: "trigger.interval",
    label: "Intervalo",
    x: 0,
    y: 0,
    config: { seconds },
  });
  const webhook = (id: string, secretEnv: string): FlowNode => ({
    id,
    kind: "trigger",
    type: "trigger.webhook",
    label: "Webhook",
    x: 0,
    y: 0,
    config: { secretEnv },
  });

  it("compiles interval trigger + mission into a schedule", () => {
    let g = emptyFlow("f", "x");
    g = addNode(g, interval("t", 7200));
    g = addNode(g, mission("m", "Rodar testes"));
    g = connect(g, "t", "m").graph;
    const r = compileFlow(g);
    expect(r.errors).toEqual([]);
    expect(r.schedule).toEqual({ objective: "Rodar testes", interval_seconds: 7200 });
  });

  it("compiles webhook trigger + mission with secret env", () => {
    let g = emptyFlow("f", "x");
    g = addNode(g, webhook("t", "MY_SECRET"));
    g = addNode(g, mission("m", "Processar faturas"));
    g = connect(g, "t", "m").graph;
    const r = compileFlow(g);
    expect(r.schedule).toEqual({
      objective: "Processar faturas",
      webhook_secret_env: "MY_SECRET",
      interval_seconds: 0,
    });
  });

  it("refuses a webhook trigger without a secret env", () => {
    let g = emptyFlow("f", "x");
    g = addNode(g, webhook("t", ""));
    g = addNode(g, mission("m", "x"));
    g = connect(g, "t", "m").graph;
    const r = compileFlow(g);
    expect(r.schedule).toBeNull();
    expect(r.errors[0]).toMatch(/segredo do gatilho de webhook/);
  });

  it("refuses a flow without a mission action", () => {
    let g = emptyFlow("f", "x");
    g = addNode(g, interval("t", 3600));
    const r = compileFlow(g);
    expect(r.schedule).toBeNull();
    expect(r.errors[0]).toMatch(/Rodar missão/);
  });

  it("refuses when the mission objective is empty", () => {
    let g = emptyFlow("f", "x");
    g = addNode(g, interval("t", 3600));
    g = addNode(g, mission("m", "   "));
    g = connect(g, "t", "m").graph;
    const r = compileFlow(g);
    expect(r.schedule).toBeNull();
    expect(r.errors[0]).toMatch(/objetivo/);
  });

  it("compiles a multi-node flow into an ordered agent-executed plan", () => {
    let g = emptyFlow("f", "x");
    g = addNode(g, interval("t", 3600));
    g = addNode(g, mission("m", "Rodar testes"));
    g = addNode(g, {
      id: "c",
      kind: "action",
      type: "action.connector",
      label: "Chamar conector",
      x: 0,
      y: 0,
      config: { connectorId: "slack", operation: "postMessage" },
    });
    g = connect(g, "t", "m").graph;
    g = connect(g, "m", "c").graph;
    const r = compileFlow(g);
    expect(r.errors).toEqual([]);
    expect(r.unsupported).toEqual([]);
    expect(r.schedule?.objective).toContain("1. Rodar missão: Rodar testes");
    expect(r.schedule?.objective).toContain('2. Chamar o conector "slack", operação "postMessage"');
    expect(r.schedule?.interval_seconds).toBe(3600);
  });

  it("compiles an HTTP action step and requires its URL", () => {
    const base = () => {
      let g = emptyFlow("f", "x");
      g = addNode(g, interval("t", 3600));
      g = addNode(g, mission("m", "Preparar"));
      g = addNode(g, {
        id: "h",
        kind: "action",
        type: "action.http",
        label: "Requisição HTTP",
        x: 0,
        y: 0,
        config: { method: "post", url: "https://api.exemplo.com/hook" },
      });
      g = connect(g, "t", "m").graph;
      g = connect(g, "m", "h").graph;
      return g;
    };
    const ok = compileFlow(base());
    expect(ok.schedule?.objective).toContain("Fazer uma requisição POST para https://api.exemplo.com/hook");

    let missingUrl = base();
    missingUrl = {
      ...missingUrl,
      nodes: missingUrl.nodes.map((n) => (n.id === "h" ? { ...n, config: { method: "POST", url: "" } } : n)),
    };
    const bad = compileFlow(missingUrl);
    expect(bad.schedule).toBeNull();
    expect(bad.errors[0]).toMatch(/URL/);
  });

  it("propagates validation errors (empty flow)", () => {
    const r = compileFlow(emptyFlow("f", "x"));
    expect(r.schedule).toBeNull();
    expect(r.errors.length).toBeGreaterThan(0);
  });
});
