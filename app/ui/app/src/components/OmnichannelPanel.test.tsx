// Testes do painel de omnicanal.
//
// Dois discriminantes centrais:
//   1. o segredo aparece pelo NOME da variavel -- nunca pelo valor;
//   2. `persisted=false` e' dito: a fila em memoria nao se apresenta como duravel.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { OmnichannelPanel } from "@/components/OmnichannelPanel";
import {
  descreverDurabilidade,
  descreverFalha,
  descreverLimite,
  descreverSegredo,
} from "@/lib/omnichannelClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<OmnichannelPanel />);
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

const CONECTOR = {
  id: "cn1", organization_id: "o1", channel: "whatsapp", name: "WhatsApp principal",
  secret_env: "OMNI_WHATSAPP_TOKEN", enabled: true,
  created_at: "2026-10-19T00:00:00Z",
};

const EVENTO = {
  id: "ev1", organization_id: "o1", channel_id: "whatsapp", connector_id: "cn1",
  external_message_id: "ext-1", contact_id: "c1", message_type: "text",
  timestamp: "2026-10-19T00:00:00Z", delivery_state: "received",
};

const LETRA = {
  id: "dl1", organization_id: "o1", channel: "whatsapp", event: EVENTO,
  reason: "timeout do fornecedor", attempts: 5,
  created_at: "2026-10-19T00:00:00Z", last_attempt_at: "2026-10-19T01:00:00Z",
};

const DLQ_OK = {
  organization_id: "o1", persisted: true, storage: "/var/hades/dlq",
  entries: 1, per_organization: 100,
};

function responder(persisted = true, error = "") {
  mockAgentFetch
    .mockResolvedValueOnce({ connectors: [CONECTOR], count: 1 })
    .mockResolvedValueOnce({ events: [EVENTO], count: 1 })
    .mockResolvedValueOnce({
      dead_letters: [LETRA], count: 1,
      storage: { ...DLQ_OK, persisted, error },
    });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de omnicanal", () => {
  it("o segredo e' o NOME da variavel, nunca o valor", () => {
    expect(descreverSegredo("OMNI_WHATSAPP_TOKEN")).toContain("OMNI_WHATSAPP_TOKEN");
    expect(descreverSegredo("OMNI_WHATSAPP_TOKEN")).toContain("nome da variavel");
    expect(descreverSegredo("")).toContain("sem variavel");
  });

  it("fila em memoria nao se apresenta como duravel", () => {
    expect(descreverDurabilidade({ ...DLQ_OK, persisted: false })).toContain("nao sobrevive a reinicio");
    expect(descreverDurabilidade({ ...DLQ_OK, persisted: true })).toContain("persistida em disco");
  });

  it("erro de persistencia e' mostrado", () => {
    expect(
      descreverDurabilidade({ ...DLQ_OK, persisted: true, error: "disco cheio" }),
    ).toContain("disco cheio");
  });

  it("limite por organizacao e' legivel", () => {
    expect(descreverLimite({ ...DLQ_OK, entries: 3 })).toBe("3 de 100 entrada(s)");
    expect(descreverLimite({ ...DLQ_OK, per_organization: 0 })).toContain("sem teto");
  });

  it("falha diz motivo e tentativas", () => {
    expect(descreverFalha("timeout", 5)).toBe("timeout (5 tentativa(s))");
    expect(descreverFalha("", 0)).toContain("motivo nao declarado");
  });
});

describe("OmnichannelPanel", () => {
  it("o conector mostra o NOME da variavel de segredo, nunca um valor", async () => {
    responder();

    const renderer = await montar();

    const segredo = texto(renderer, "omnicanal-segredo-cn1");
    expect(segredo).toContain("OMNI_WHATSAPP_TOKEN");
    expect(segredo).toContain("nome da variavel");
  });

  it("fila persistida diz que esta' em disco", async () => {
    responder(true);

    const renderer = await montar();

    expect(texto(renderer, "omnicanal-durabilidade")).toContain("persistida em disco");
  });

  it("fila NAO persistida e' declarada: nao sobrevive a reinicio", async () => {
    responder(false);

    const renderer = await montar();

    expect(texto(renderer, "omnicanal-durabilidade")).toContain("nao sobrevive a reinicio");
  });

  it("erro de persistencia aparece com a fila de pe'", async () => {
    responder(true, "disco cheio");

    const renderer = await montar();

    const durabilidade = texto(renderer, "omnicanal-durabilidade");
    expect(durabilidade).toContain("disco cheio");
    // A fila continua visivel.
    expect(texto(renderer, "omnicanal-letras")).toContain("whatsapp");
  });

  it("mostra o limite por organizacao", async () => {
    responder();

    const renderer = await montar();

    expect(texto(renderer, "omnicanal-limite")).toContain("1 de 100");
  });

  it("a letra morta mostra motivo e tentativas", async () => {
    responder();

    const renderer = await montar();

    const letra = texto(renderer, "omnicanal-letra-dl1");
    expect(letra).toContain("timeout do fornecedor");
    expect(letra).toContain("5 tentativa(s)");
  });

  it("mostra os eventos com estado de entrega", async () => {
    responder();

    const renderer = await montar();

    expect(texto(renderer, "omnicanal-eventos")).toContain("received");
  });

  it("fila vazia diz 'a fila existe e esta vazia', nao que nao ha fila", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ connectors: [], count: 0 })
      .mockResolvedValueOnce({ events: [], count: 0 })
      .mockResolvedValueOnce({
        dead_letters: [], count: 0, storage: { ...DLQ_OK, entries: 0 },
      });

    const renderer = await montar();

    expect(texto(renderer, "omnicanal-sem-letras")).toContain("a fila existe e está vazia");
  });

  it("o 503 aparece como omnicanal AUSENTE, nao como lista vazia", async () => {
    const erro = new Error("omnichannel nao esta configurado");
    mockAgentFetch.mockRejectedValueOnce(erro).mockRejectedValueOnce(erro).mockRejectedValueOnce(erro);

    const renderer = await montar();

    const sem = texto(renderer, "omnicanal-sem-omnicanal");
    expect(sem).toContain("não está configurado");
    expect(sem).toContain("não é «não há conectores»");
    expect(porTestId(renderer, "omnicanal-erro")).toHaveLength(0);
  });

  it("as tres chamadas usam o prefixo explicito e sao GET", async () => {
    responder();

    await montar();

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string).sort();
    expect(urls).toEqual([
      "/api/agent/v1/omnichannel/connectors",
      "/api/agent/v1/omnichannel/dlq",
      "/api/agent/v1/omnichannel/events",
    ]);
    for (const chamada of mockAgentFetch.mock.calls) {
      const init = chamada[1] as RequestInit | undefined;
      expect(init?.method ?? "GET").toBe("GET");
    }
  });
});
