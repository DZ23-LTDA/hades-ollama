// Testes do painel de agendamento.
//
// O caso que mais importa é o 503: o servidor distingue «motor não ligado» de
// «não há serviços» e diz isso textualmente no código. Um painel que tratasse os
// dois como lista vazia passaria num teste ingénuo e mentiria ao utilizador.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { SchedulingPanel } from "@/components/SchedulingPanel";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<SchedulingPanel />);
  });
  return renderer;
}

function juntarTexto(nodo: unknown): string {
  if (typeof nodo === "string") return nodo;
  if (typeof nodo === "number") return String(nodo);
  if (Array.isArray(nodo)) return nodo.map(juntarTexto).join("");
  if (nodo === null || typeof nodo !== "object") return "";
  return juntarTexto((nodo as { children?: unknown }).children);
}

function texto(renderer: ReturnType<typeof create>, id: string): string {
  const arvore = renderer.toJSON();
  const encontrados: string[] = [];
  const visitar = (nodo: unknown) => {
    if (Array.isArray(nodo)) { nodo.forEach(visitar); return; }
    if (nodo === null || typeof nodo !== "object") return;
    const props = (nodo as { props?: Record<string, unknown> }).props;
    if (props && props["data-testid"] === id) encontrados.push(juntarTexto(nodo));
    visitar((nodo as { children?: unknown }).children);
  };
  visitar(arvore);
  return encontrados.join(" ");
}

function porTestId(renderer: ReturnType<typeof create>, id: string) {
  return renderer.root.findAll((nodo) => nodo.props["data-testid"] === id);
}

const SERVICOS = [
  { id: "s1", organization_id: "o1", name: "Consulta", duration_minutes: 30, buffer_minutes: 10 },
];
const AGENDAMENTOS = [
  {
    id: "a1", organization_id: "o1", resource_id: "r1", service_id: "s1",
    starts_at: "2026-10-20T13:00:00Z", ends_at: "2026-10-20T13:30:00Z",
    busy_until: "2026-10-20T13:40:00Z", status: "booked", created_at: "2026-10-19T00:00:00Z",
  },
  {
    id: "a2", organization_id: "o1", resource_id: "r1", service_id: "s1",
    starts_at: "2026-10-20T14:00:00Z", ends_at: "2026-10-20T14:30:00Z",
    busy_until: "2026-10-20T14:30:00Z", status: "cancelled", created_at: "2026-10-19T00:00:00Z",
  },
  {
    id: "a3", organization_id: "o1", resource_id: "r1", service_id: "s1",
    starts_at: "2026-10-20T15:00:00Z", ends_at: "2026-10-20T15:30:00Z",
    busy_until: "2026-10-20T15:30:00Z", status: "no_show", created_at: "2026-10-19T00:00:00Z",
  },
  {
    id: "a4", organization_id: "o1", resource_id: "r1", service_id: "s1",
    starts_at: "2026-10-20T16:00:00Z", ends_at: "2026-10-20T16:30:00Z",
    busy_until: "2026-10-20T16:30:00Z", status: "estado-novo", created_at: "2026-10-19T00:00:00Z",
  },
];

function responder(servicos: unknown, agendamentos: unknown) {
  mockAgentFetch
    .mockResolvedValueOnce({ services: servicos })
    .mockResolvedValueOnce({ appointments: agendamentos, count: Array.isArray(agendamentos) ? agendamentos.length : 0 });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("SchedulingPanel", () => {
  it("o 503 aparece como MOTOR AUSENTE, não como lista vazia", async () => {
    mockAgentFetch
      .mockRejectedValueOnce(new Error("motor de agendamento não está ligado neste runtime"))
      .mockRejectedValueOnce(new Error("motor de agendamento não está ligado neste runtime"));

    const renderer = await montar();

    const semMotor = texto(renderer, "scheduling-sem-motor");
    expect(semMotor).toContain("não está ligado");
    expect(semMotor).toContain("não é «não há serviços»");
    expect(porTestId(renderer, "scheduling-erro")).toHaveLength(0);
    expect(porTestId(renderer, "scheduling-sem-servicos")).toHaveLength(0);
  });

  it("motor ligado e zero serviços diz isso, e é diferente de motor ausente", async () => {
    responder([], []);

    const renderer = await montar();

    expect(texto(renderer, "scheduling-sem-servicos")).toContain("motor está ligado");
    expect(porTestId(renderer, "scheduling-sem-motor")).toHaveLength(0);
  });

  it("lista os serviços com duração e buffer", async () => {
    responder(SERVICOS, AGENDAMENTOS);

    const renderer = await montar();

    const tabela = texto(renderer, "scheduling-servicos");
    expect(tabela).toContain("Consulta");
    expect(tabela).toContain("30 min");
    expect(tabela).toContain("10 min");
  });

  it("MOSTRA O BUFFER: busy_until depois de ends_at vira '+N min de buffer'", async () => {
    responder(SERVICOS, AGENDAMENTOS);

    const renderer = await montar();

    // a1 tem ends_at 13:30 e busy_until 13:40 -> 10 min de buffer.
    expect(texto(renderer, "scheduling-buffer-a1")).toContain("+10 min de buffer");
    // a2 não tem diferença: nenhum selo de buffer.
    expect(porTestId(renderer, "scheduling-buffer-a2")).toHaveLength(0);
  });

  it("conta os três estados e separa os desconhecidos", async () => {
    responder(SERVICOS, AGENDAMENTOS);

    const renderer = await montar();

    const contagem = texto(renderer, "scheduling-contagem");
    expect(contagem).toContain("1 marcado(s)");
    expect(contagem).toContain("1 cancelado(s)");
    expect(contagem).toContain("1 falta(s)");
    expect(contagem).toContain("1 desconhecido(s)");
  });

  it("estado fora da lista fechada NÃO ganha rótulo inventado", async () => {
    responder(SERVICOS, AGENDAMENTOS);

    const renderer = await montar();

    const tabela = texto(renderer, "scheduling-agendamentos");
    expect(tabela).toContain("estado desconhecido (estado-novo)");
    expect(tabela).toContain("falta");
  });

  it("um erro que não é 503 continua a ser erro", async () => {
    mockAgentFetch
      .mockRejectedValueOnce(new Error("organizacao invalida"))
      .mockRejectedValueOnce(new Error("organizacao invalida"));

    const renderer = await montar();

    expect(texto(renderer, "scheduling-erro")).toContain("organizacao invalida");
    expect(porTestId(renderer, "scheduling-sem-motor")).toHaveLength(0);
  });

  it("as duas chamadas usam o prefixo explícito e são GET", async () => {
    responder(SERVICOS, AGENDAMENTOS);

    await montar();

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string).sort();
    expect(urls).toEqual([
      "/api/agent/v1/scheduling/appointments",
      "/api/agent/v1/scheduling/services",
    ]);
    for (const chamada of mockAgentFetch.mock.calls) {
      const init = chamada[1] as RequestInit | undefined;
      expect(init?.method ?? "GET").toBe("GET");
    }
  });

  it("a fronteira diz que só lê", async () => {
    responder(SERVICOS, AGENDAMENTOS);

    const renderer = await montar();

    const fronteira = texto(renderer, "scheduling-fronteira");
    expect(fronteira).toContain("só lê");
    expect(fronteira).toContain("intervalo real");
  });
});
