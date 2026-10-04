import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  PlusIcon,
  TrashIcon,
  BoltIcon,
  CursorArrowRaysIcon,
  ArrowsRightLeftIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import {
  emptyFlow,
  addNode,
  removeNode,
  moveNode,
  updateNodeConfig,
  connect,
  disconnect,
  validateFlow,
  type FlowGraph,
  type FlowNode,
  type FlowNodeKind,
} from "@/lib/flowGraph";

// Catalog of node types offered in the palette. Each produces a concrete node
// with a sensible default label and config. Kept data-driven so new node types
// (more triggers, more connectors) are one entry, not new UI.
interface PaletteItem {
  kind: FlowNodeKind;
  type: string;
  label: string;
  defaults: Record<string, unknown>;
}

const PALETTE: { group: string; items: PaletteItem[] }[] = [
  {
    group: "Gatilhos",
    items: [
      { kind: "trigger", type: "trigger.webhook", label: "Webhook", defaults: { secretEnv: "" } },
      { kind: "trigger", type: "trigger.interval", label: "Intervalo", defaults: { seconds: 3600 } },
    ],
  },
  {
    group: "Ações",
    items: [
      { kind: "action", type: "action.mission", label: "Rodar missão", defaults: { objective: "" } },
      { kind: "action", type: "action.connector", label: "Chamar conector", defaults: { connectorId: "", operation: "" } },
      { kind: "action", type: "action.http", label: "Requisição HTTP", defaults: { method: "POST", url: "" } },
    ],
  },
  {
    group: "Controle",
    items: [
      { kind: "condition", type: "condition.if", label: "Condição (se/senão)", defaults: { expression: "" } },
    ],
  },
];

const NODE_W = 180;
const NODE_H = 72;

const KIND_STYLE: Record<FlowNodeKind, string> = {
  trigger: "border-emerald-400 bg-emerald-50 dark:border-emerald-500/60 dark:bg-emerald-900/30",
  action: "border-blue-400 bg-blue-50 dark:border-blue-500/60 dark:bg-blue-900/30",
  condition: "border-amber-400 bg-amber-50 dark:border-amber-500/60 dark:bg-amber-900/30",
};

interface FlowEditorProps {
  storageKey?: string;
}

function loadDraft(storageKey: string): FlowGraph | null {
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as FlowGraph;
    if (parsed && Array.isArray(parsed.nodes) && Array.isArray(parsed.edges)) {
      return parsed;
    }
  } catch {
    // Ignore corrupt or unavailable storage; start from an empty flow.
  }
  return null;
}

export function FlowEditor({ storageKey = "hades.flow.draft" }: FlowEditorProps) {
  const [graph, setGraph] = useState<FlowGraph>(
    () => loadDraft(storageKey) ?? emptyFlow("flow_1", "Novo fluxo"),
  );
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [connectFrom, setConnectFrom] = useState<{ node: string; port?: string } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const canvasRef = useRef<HTMLDivElement | null>(null);
  const dragState = useRef<{ id: string; dx: number; dy: number } | null>(null);
  const nextId = useRef(1);

  useEffect(() => {
    try {
      localStorage.setItem(storageKey, JSON.stringify(graph));
    } catch {
      // Per-viewer draft only; safe to skip when storage is unavailable.
    }
  }, [graph, storageKey]);

  const problems = useMemo(() => validateFlow(graph), [graph]);
  const selected = graph.nodes.find((n) => n.id === selectedId) ?? null;

  const handleAdd = useCallback((item: PaletteItem) => {
    setGraph((g) => {
      const id = `n${nextId.current++}_${item.kind}`;
      const count = g.nodes.length;
      const node: FlowNode = {
        id,
        kind: item.kind,
        type: item.type,
        label: item.label,
        x: 40 + (count % 4) * (NODE_W + 40),
        y: 40 + Math.floor(count / 4) * (NODE_H + 60),
        config: { ...item.defaults },
      };
      return addNode(g, node);
    });
  }, []);

  const handleNodePointerDown = useCallback(
    (e: React.PointerEvent, node: FlowNode) => {
      setSelectedId(node.id);
      const rect = canvasRef.current?.getBoundingClientRect();
      if (!rect) return;
      dragState.current = {
        id: node.id,
        dx: e.clientX - rect.left - node.x,
        dy: e.clientY - rect.top - node.y,
      };
      (e.target as Element).setPointerCapture?.(e.pointerId);
    },
    [],
  );

  const handlePointerMove = useCallback((e: React.PointerEvent) => {
    const drag = dragState.current;
    const rect = canvasRef.current?.getBoundingClientRect();
    if (!drag || !rect) return;
    const x = Math.max(0, e.clientX - rect.left - drag.dx);
    const y = Math.max(0, e.clientY - rect.top - drag.dy);
    setGraph((g) => moveNode(g, drag.id, x, y));
  }, []);

  const handlePointerUp = useCallback(() => {
    dragState.current = null;
  }, []);

  // Click-to-connect: click an output port to start, then click a target node.
  const startConnect = useCallback((nodeId: string, port?: string) => {
    setError(null);
    setConnectFrom({ node: nodeId, port });
  }, []);

  const completeConnect = useCallback(
    (targetId: string) => {
      if (!connectFrom) return;
      setGraph((g) => {
        const result = connect(g, connectFrom.node, targetId, connectFrom.port);
        if (result.error) {
          setError(result.error);
          return g;
        }
        return result.graph;
      });
      setConnectFrom(null);
    },
    [connectFrom],
  );

  const handleNodeClick = useCallback(
    (node: FlowNode) => {
      if (connectFrom && connectFrom.node !== node.id) {
        completeConnect(node.id);
      } else {
        setSelectedId(node.id);
      }
    },
    [connectFrom, completeConnect],
  );

  const handleDelete = useCallback((id: string) => {
    setGraph((g) => removeNode(g, id));
    setSelectedId((cur) => (cur === id ? null : cur));
  }, []);

  const portCenter = (node: FlowNode, side: "out" | "in") => ({
    x: node.x + (side === "out" ? NODE_W : 0),
    y: node.y + NODE_H / 2,
  });

  return (
    <div className="flex h-[70vh] gap-3" data-testid="flow-editor">
      {/* Palette */}
      <aside className="w-44 shrink-0 space-y-3 overflow-y-auto">
        {PALETTE.map((group) => (
          <div key={group.group}>
            <h4 className="mb-1 text-xs font-semibold uppercase tracking-wide text-neutral-500">
              {group.group}
            </h4>
            <div className="space-y-1">
              {group.items.map((item) => (
                <button
                  key={item.type}
                  type="button"
                  onClick={() => handleAdd(item)}
                  className="flex w-full items-center gap-1.5 rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-left text-xs text-neutral-800 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100 dark:hover:bg-neutral-700"
                >
                  <PlusIcon className="h-3.5 w-3.5 shrink-0 text-neutral-400" />
                  {item.label}
                </button>
              ))}
            </div>
          </div>
        ))}
      </aside>

      {/* Canvas */}
      <div className="relative flex-1 overflow-hidden rounded-xl border border-neutral-200 bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-900/40">
        {connectFrom && (
          <div className="absolute left-2 top-2 z-20 flex items-center gap-2 rounded-lg bg-blue-600 px-3 py-1.5 text-xs text-white shadow">
            <ArrowsRightLeftIcon className="h-4 w-4" />
            Clique no nó de destino para conectar
            <button type="button" onClick={() => setConnectFrom(null)} aria-label="Cancelar conexão">
              <XMarkIcon className="h-4 w-4" />
            </button>
          </div>
        )}
        {error && (
          <div
            role="alert"
            className="absolute right-2 top-2 z-20 rounded-lg bg-red-600 px-3 py-1.5 text-xs text-white shadow"
          >
            {error}
          </div>
        )}
        <div
          ref={canvasRef}
          className="absolute inset-0"
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
        >
          {/* Edges */}
          <svg className="pointer-events-none absolute inset-0 h-full w-full">
            {graph.edges.map((edge) => {
              const from = graph.nodes.find((n) => n.id === edge.from);
              const to = graph.nodes.find((n) => n.id === edge.to);
              if (!from || !to) return null;
              const a = portCenter(from, "out");
              const b = portCenter(to, "in");
              const midX = (a.x + b.x) / 2;
              return (
                <path
                  key={edge.id}
                  d={`M ${a.x} ${a.y} C ${midX} ${a.y}, ${midX} ${b.y}, ${b.x} ${b.y}`}
                  className="fill-none stroke-neutral-400 dark:stroke-neutral-500"
                  strokeWidth={2}
                />
              );
            })}
          </svg>

          {/* Nodes */}
          {graph.nodes.map((node) => (
            <div
              key={node.id}
              data-testid={`node-${node.id}`}
              onPointerDown={(e) => handleNodePointerDown(e, node)}
              onClick={() => handleNodeClick(node)}
              style={{ left: node.x, top: node.y, width: NODE_W, height: NODE_H }}
              className={`absolute cursor-grab select-none rounded-xl border-2 p-2 text-xs shadow-sm ${KIND_STYLE[node.kind]} ${
                selectedId === node.id ? "ring-2 ring-blue-500" : ""
              }`}
            >
              <div className="flex items-center gap-1 font-semibold text-neutral-800 dark:text-neutral-100">
                <BoltIcon className="h-3.5 w-3.5" />
                {node.label}
              </div>
              <div className="mt-0.5 truncate text-[10px] text-neutral-500">{node.type}</div>
              {/* Output port(s): condition exposes true/false */}
              {node.kind === "condition" ? (
                <div className="absolute -right-2 top-1 flex flex-col gap-1">
                  <button
                    type="button"
                    aria-label={`Conectar saída verdadeiro de ${node.label}`}
                    onClick={(e) => { e.stopPropagation(); startConnect(node.id, "true"); }}
                    className="h-3 w-3 rounded-full bg-emerald-500"
                    title="verdadeiro"
                  />
                  <button
                    type="button"
                    aria-label={`Conectar saída falso de ${node.label}`}
                    onClick={(e) => { e.stopPropagation(); startConnect(node.id, "false"); }}
                    className="h-3 w-3 rounded-full bg-red-500"
                    title="falso"
                  />
                </div>
              ) : (
                <button
                  type="button"
                  aria-label={`Conectar saída de ${node.label}`}
                  onClick={(e) => { e.stopPropagation(); startConnect(node.id); }}
                  className="absolute -right-2 top-1/2 h-3 w-3 -translate-y-1/2 rounded-full bg-blue-500"
                  title="saída"
                />
              )}
            </div>
          ))}

          {graph.nodes.length === 0 && (
            <div className="flex h-full items-center justify-center text-sm text-neutral-400">
              <CursorArrowRaysIcon className="mr-2 h-5 w-5" />
              Adicione um gatilho da paleta para começar o fluxo
            </div>
          )}
        </div>
      </div>

      {/* Inspector + validation */}
      <aside className="w-56 shrink-0 space-y-3 overflow-y-auto">
        {selected ? (
          <div className="rounded-xl border border-neutral-200 bg-white p-3 dark:border-neutral-700 dark:bg-neutral-800">
            <div className="mb-2 flex items-center justify-between">
              <h4 className="text-xs font-semibold uppercase tracking-wide text-neutral-500">Nó</h4>
              <button
                type="button"
                onClick={() => handleDelete(selected.id)}
                aria-label="Excluir nó"
                className="text-red-500 hover:text-red-600"
              >
                <TrashIcon className="h-4 w-4" />
              </button>
            </div>
            <label className="block text-[11px] font-medium text-neutral-600 dark:text-neutral-300">
              Rótulo
            </label>
            <input
              value={selected.label}
              onChange={(e) =>
                setGraph((g) => ({
                  ...g,
                  nodes: g.nodes.map((n) =>
                    n.id === selected.id ? { ...n, label: e.target.value } : n,
                  ),
                }))
              }
              className="mt-1 w-full rounded-lg border border-neutral-300 bg-white px-2 py-1 text-xs dark:border-neutral-600 dark:bg-neutral-700"
            />
            {Object.keys(selected.config).map((key) => (
              <div key={key} className="mt-2">
                <label className="block text-[11px] font-medium text-neutral-600 dark:text-neutral-300">
                  {key}
                </label>
                <input
                  value={String(selected.config[key] ?? "")}
                  onChange={(e) => setGraph((g) => updateNodeConfig(g, selected.id, { [key]: e.target.value }))}
                  className="mt-1 w-full rounded-lg border border-neutral-300 bg-white px-2 py-1 text-xs dark:border-neutral-600 dark:bg-neutral-700"
                />
              </div>
            ))}
          </div>
        ) : (
          <div className="rounded-xl border border-dashed border-neutral-300 p-3 text-xs text-neutral-400 dark:border-neutral-600">
            Selecione um nó para editar seus campos.
          </div>
        )}

        <div className="rounded-xl border border-neutral-200 bg-white p-3 dark:border-neutral-700 dark:bg-neutral-800">
          <h4 className="mb-1 text-xs font-semibold uppercase tracking-wide text-neutral-500">
            Validação
          </h4>
          {problems.length === 0 ? (
            <p className="text-xs text-emerald-600 dark:text-emerald-400" data-testid="flow-valid">
              Fluxo válido e pronto para rodar.
            </p>
          ) : (
            <ul className="list-disc space-y-1 pl-4 text-xs text-amber-700 dark:text-amber-400" data-testid="flow-problems">
              {problems.map((p, i) => (
                <li key={i}>{p}</li>
              ))}
            </ul>
          )}
        </div>

        {graph.edges.length > 0 && (
          <div className="rounded-xl border border-neutral-200 bg-white p-3 dark:border-neutral-700 dark:bg-neutral-800">
            <h4 className="mb-1 text-xs font-semibold uppercase tracking-wide text-neutral-500">
              Conexões
            </h4>
            <ul className="space-y-1 text-[11px]">
              {graph.edges.map((edge) => (
                <li key={edge.id} className="flex items-center justify-between gap-1">
                  <span className="truncate">
                    {graph.nodes.find((n) => n.id === edge.from)?.label}
                    {edge.fromPort ? ` (${edge.fromPort})` : ""} → {graph.nodes.find((n) => n.id === edge.to)?.label}
                  </span>
                  <button
                    type="button"
                    onClick={() => setGraph((g) => disconnect(g, edge.id))}
                    aria-label="Remover conexão"
                    className="text-red-400 hover:text-red-600"
                  >
                    <XMarkIcon className="h-3 w-3" />
                  </button>
                </li>
              ))}
            </ul>
          </div>
        )}
      </aside>
    </div>
  );
}
