// Testes do aditamento ao CompanySupervisorPanel: incidentes e ciclos.
//
// Discriminantes centrais:
//   1. `acknowledged_by` e' dito como "reconhecer NAO e' resolver";
//   2. lista vazia diz que NAO distingue "sem incidentes" de "sem supervisor";
//   3. `blocked_external_actions` e `risk_paused_companies` aparecem nos ciclos.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({
  getSupervisorStatus: vi.fn(),
  setSupervisorConfig: vi.fn(),
  triggerSupervisorTick: vi.fn(),
  getSupervisorIncidents: vi.fn(),
  getSupervisorTicks: vi.fn(),
}));

import {
  getSupervisorIncidents,
  getSupervisorStatus,
  getSupervisorTicks,
} from "@/lib/agenticClient";
import { CompanySupervisorPanel } from "@/components/CompanySupervisorPanel";

const mockStatus = vi.mocked(getSupervisorStatus);
const mockIncidentes = vi.mocked(getSupervisorIncidents);
const mockTicks = vi.mocked(getSupervisorTicks);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<CompanySupervisorPanel />);
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

const STATUS = {
  enabled: true, running: true, last_tick_at: "2026-10-19T00:00:00Z",
  blocked_external_actions: 3, pending_approvals_count: 1,
};

const INCIDENTE = {
  id: "i1", kind: "risk_pause", subject: "Empresa A pausada",
  state: "open", opened_at: "2026-10-19T00:00:00Z", updated_at: "2026-10-19T00:00:00Z",
  recovery_attempts: 2, acknowledged_by: "ana@exemplo.com",
};

const INCIDENTE_RESOLVIDO = {
  ...INCIDENTE, id: "i2", state: "resolved",
  resolution: "reiniciado", resolved_at: "2026-10-19T01:00:00Z",
};

const TICK = {
  tick_number: 7, executed_at: "2026-10-19T00:00:00Z",
  missions_resumed: 2, schedules_triggered: 1, cycles_advanced: 3,
  approvals_created: 1, blocked_external_actions: 4,
  risk_paused_companies: 1, error_count: 0,
};

function responder(incidentes = [INCIDENTE], ticks = [TICK]) {
  mockStatus.mockResolvedValueOnce(STATUS as never);
  mockIncidentes.mockResolvedValueOnce({ incidents: incidentes as never, count: incidentes.length });
  mockTicks.mockResolvedValueOnce({ ticks: ticks as never, count: ticks.length });
}

afterEach(() => {
  mockStatus.mockReset();
  mockIncidentes.mockReset();
  mockTicks.mockReset();
});

describe("CompanySupervisorPanel: incidentes e ciclos", () => {
  it("reconhecido NAO e' resolvido", async () => {
    responder();

    const renderer = await montar();

    const ciencia = texto(renderer, "supervisor-ciencia-i1");
    expect(ciencia).toContain("reconhecido por ana@exemplo.com");
    expect(ciencia).toContain("reconhecer NÃO é resolver");
  });

  it("o incidente mostra tentativas e ausencia de resolucao", async () => {
    responder();

    const renderer = await montar();

    const t = texto(renderer, "supervisor-incidente-i1");
    expect(t).toContain("2 tentativa(s)");
    expect(t).toContain("sem resolução registada");
  });

  it("um incidente resolvido mostra a resolucao", async () => {
    responder([INCIDENTE_RESOLVIDO], []);

    const renderer = await montar();

    const t = texto(renderer, "supervisor-incidente-i2");
    expect(t).toContain("resolução: reiniciado");
  });

  it("lista vazia NAO e' declarada como 'tudo bem'", async () => {
    responder([], []);

    const renderer = await montar();

    const t = texto(renderer, "supervisor-sem-incidentes");
    expect(t).toContain("não distingue");
    expect(t).toContain("não há supervisor");
    expect(t).not.toContain("tudo bem");
  });

  it("os ciclos mostram bloqueios externos e pausas por risco", async () => {
    responder();

    const renderer = await montar();

    const t = texto(renderer, "supervisor-ticks");
    expect(t).toContain("4 ação(ões) externa(s) bloqueada(s)");
    expect(t).toContain("1 empresa(s) pausada(s) por risco");
    expect(t).toContain("#7");
  });

  it("sem ciclos, diz que o diario esta' vazio", async () => {
    responder([], []);

    const renderer = await montar();

    expect(texto(renderer, "supervisor-sem-ticks")).toContain("Nenhum ciclo");
  });

  it("as tres chamadas usam o prefixo explicito", async () => {
    responder();

    await montar();

    expect(mockStatus.mock.calls[0][0]).toBeUndefined();
    expect(mockIncidentes).toHaveBeenCalledTimes(1);
    expect(mockTicks).toHaveBeenCalledTimes(1);
  });
});
