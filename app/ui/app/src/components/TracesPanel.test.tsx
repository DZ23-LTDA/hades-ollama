// Testes do painel de traces.
//
// O que cada caso trava, e porque falharia com o defeito:
//
//   - `duration_ns` tem de ser dito em ms/s. Mostrar "4820000000" é o número
//     certo na unidade errada.
//   - um span sem `end_at` tem de aparecer como ABERTO, não como concluído.
//   - um span com `error` tem de aparecer como FALHADO mesmo que o status diga
//     "ok" -- o erro é mais forte que o rótulo.
//   - lista vazia NÃO é erro: uma organização sem traces é um estado legítimo.
//   - a contagem tem de distinguir os três estados, não somar tudo.
//   - o painel não pode apagar nem escrever nada (verificado pelo número de
//     chamadas: só leituras).
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: vi.fn(),
}));

import { agentFetch } from "@/lib/agenticClient";
import { TracesPanel } from "@/components/TracesPanel";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<TracesPanel />);
  });
  return renderer;
}

async function agir(accao: () => unknown) {
  await act(async () => {
    await accao();
  });
}

function porTestId(renderer: ReturnType<typeof create>, id: string) {
  return renderer.root.findAll((nodo) => nodo.props["data-testid"] === id);
}

/** Todo o texto de uma subarvore do `toJSON()`, incluindo elementos aninhados. */
function juntarTexto(nodo: unknown): string {
  if (typeof nodo === "string") return nodo;
  if (typeof nodo === "number") return String(nodo);
  if (Array.isArray(nodo)) return nodo.map(juntarTexto).join("");
  if (nodo === null || typeof nodo !== "object") return "";
  const filhos = (nodo as { children?: unknown }).children;
  return juntarTexto(filhos);
}

/** Procura o elemento com `data-testid` na arvore serializada e devolve o seu texto. */
function texto(renderer: ReturnType<typeof create>, id: string): string {
  const arvore = renderer.toJSON();
  const encontrados: string[] = [];
  const visitar = (nodo: unknown) => {
    if (Array.isArray(nodo)) {
      nodo.forEach(visitar);
      return;
    }
    if (nodo === null || typeof nodo !== "object") return;
    const props = (nodo as { props?: Record<string, unknown> }).props;
    if (props && props["data-testid"] === id) {
      encontrados.push(juntarTexto(nodo));
    }
    visitar((nodo as { children?: unknown }).children);
  };
  visitar(arvore);
  return encontrados.join(" ");
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("TracesPanel", () => {
  it("diz que não há traces em vez de mostrar erro", async () => {
    mockAgentFetch.mockResolvedValueOnce({ spans: [] });

    const renderer = await montar();

    expect(porTestId(renderer, "traces-vazio")).toHaveLength(1);
    expect(porTestId(renderer, "traces-erro")).toHaveLength(0);
  });

  it("trata spans null como vazio, não como falha", async () => {
    mockAgentFetch.mockResolvedValueOnce({ spans: null });

    const renderer = await montar();

    expect(porTestId(renderer, "traces-vazio")).toHaveLength(1);
  });

  it("formata duration_ns na unidade que se lê", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      spans: [
        {
          trace_id: "t1",
          span_id: "s1",
          name: "build.plan",
          status: "ok",
          start_at: "2026-10-19T00:00:00Z",
          end_at: "2026-10-19T00:00:04Z",
          duration_ns: 4_820_000_000,
        },
        {
          trace_id: "t1",
          span_id: "s2",
          name: "toolcall",
          status: "ok",
          start_at: "2026-10-19T00:00:00Z",
          end_at: "2026-10-19T00:00:00Z",
          duration_ns: 12_500_000,
        },
      ],
    });

    const renderer = await montar();

    const tabela = texto(renderer, "traces-tabela");
    expect(tabela).toContain("4.82 s");
    expect(tabela).toContain("12.5 ms");
    expect(tabela).not.toContain("4820000000");
  });

  it("mostra um span sem end_at como ABERTO, não como concluído", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      spans: [
        {
          trace_id: "t1",
          span_id: "s-aberto",
          name: "agent.run",
          status: "running",
          start_at: "2026-10-19T00:00:00Z",
        },
      ],
    });

    const renderer = await montar();

    expect(texto(renderer, "traces-estado-s-aberto")).toContain("aberto");
    expect(texto(renderer, "traces-contagem")).toContain("1 aberto(s)");
  });

  it("um erro vence o rótulo: span com error é FALHADO mesmo com status ok", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      spans: [
        {
          trace_id: "t1",
          span_id: "s-falha",
          name: "build.sign",
          status: "ok",
          error: "certificado sem chave privada",
          start_at: "2026-10-19T00:00:00Z",
          end_at: "2026-10-19T00:00:01Z",
          duration_ns: 1_000_000,
        },
      ],
    });

    const renderer = await montar();

    expect(texto(renderer, "traces-estado-s-falha")).toContain("falhou");
    expect(texto(renderer, "traces-estado-s-falha")).toContain("certificado sem chave privada");
    expect(texto(renderer, "traces-contagem")).toContain("1 falhado(s)");
  });

  it("a contagem distingue os três estados em vez de somar tudo", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      spans: [
        { trace_id: "t", span_id: "a", name: "x", status: "ok", start_at: "2026-10-19T00:00:00Z", end_at: "2026-10-19T00:00:01Z", duration_ns: 1 },
        { trace_id: "t", span_id: "b", name: "y", status: "running", start_at: "2026-10-19T00:00:00Z" },
        { trace_id: "t", span_id: "c", name: "z", status: "error", start_at: "2026-10-19T00:00:00Z", end_at: "2026-10-19T00:00:01Z" },
      ],
    });

    const renderer = await montar();

    const contagem = texto(renderer, "traces-contagem");
    expect(contagem).toContain("3 span");
    expect(contagem).toContain("1 concluído(s)");
    expect(contagem).toContain("1 aberto(s)");
    expect(contagem).toContain("1 falhado(s)");
  });

  it("filtra por trace_id na URL, com o prefixo explícito", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ spans: [] })
      .mockResolvedValueOnce({ spans: [] });

    const renderer = await montar();
    const campo = porTestId(renderer, "traces-filtro")[0];
    await agir(() => campo.props.onChange({ target: { value: "tr_abc" } }));
    await agir(() => porTestId(renderer, "traces-filtrar")[0].props.onClick());

    const segunda = mockAgentFetch.mock.calls[1];
    expect(segunda[0]).toBe("/api/agent/v1/traces?trace_id=tr_abc");
  });

  it("só lê: nenhuma chamada de escrita", async () => {
    mockAgentFetch.mockResolvedValueOnce({ spans: [] });

    await montar();

    for (const chamada of mockAgentFetch.mock.calls) {
      const init = chamada[1] as RequestInit | undefined;
      expect(init?.method ?? "GET").toBe("GET");
    }
  });

  it("mostra o erro do servidor em vez de o engolir", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("forbidden"));

    const renderer = await montar();

    expect(texto(renderer, "traces-erro")).toContain("forbidden");
  });
});
