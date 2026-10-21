// Testes do painel do Inbox.
//
// O caso que mais importa e' o `reply_blocked`: NAO e' um erro. A conversa continua
// visivel e legivel. Um teste que so' verificasse "mostra a conversa" passaria mesmo
// que o painel escondesse a conversa inteira ao ver o bloqueio.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { InboxPanel } from "@/components/InboxPanel";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<InboxPanel />);
  });
  return renderer;
}

async function agir(accao: () => unknown) {
  await act(async () => {
    await accao();
  });
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

const CONVERSAS = [
  {
    id: "cv1", organization_id: "o1", channel: "whatsapp", contact_id: "c1",
    contact_name: "Ana Silva", status: "open", message_count: 3,
    last_message: "Bom dia, preciso de ajuda",
    created_at: "2026-10-19T00:00:00Z", updated_at: "2026-10-19T00:00:00Z",
  },
  {
    id: "cv2", organization_id: "o1", channel: "email", contact_id: "c2",
    status: "pending", message_count: 1,
    created_at: "2026-10-19T00:00:00Z", updated_at: "2026-10-19T00:00:00Z",
  },
];

const METRICAS = {
  organization_id: "o1",
  total_conversations: 10,
  open_conversations: 4,
  pending_human: 2,
  closed_conversations: 6,
  sla_breached: 1,
  awaiting_first_reply: 2,
  first_response_avg_seconds: 125,
  resolution_sla_breached: 0,
  resolution_overdue: 1,
  resolution_avg_seconds: 45,
  messages_inbound: 30,
  messages_outbound: 25,
};

function responder() {
  mockAgentFetch
    .mockResolvedValueOnce({ conversations: CONVERSAS, count: CONVERSAS.length })
    .mockResolvedValueOnce(METRICAS);
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("InboxPanel", () => {
  it("mostra as metricas com os segundos convertidos, nao crus", async () => {
    responder();

    const renderer = await montar();

    const tempos = texto(renderer, "inbox-tempos");
    // 125 s -> "2 min 5 s", 45 s -> "45 s".
    expect(tempos).toContain("2 min 5 s");
    expect(tempos).toContain("45 s");
    // Os valores crus NAO aparecem.
    expect(tempos).not.toContain("125");
    expect(tempos).not.toContain("resolution_avg");
  });

  it("mostra o SLA violado e as contagens", async () => {
    responder();

    const renderer = await montar();

    const metricas = texto(renderer, "inbox-metricas");
    expect(metricas).toContain("4 aberta(s)");
    expect(metricas).toContain("2 à espera de humano");
    expect(metricas).toContain("1 SLA de resposta violado(s)");
    expect(metricas).toContain("1 resolução em atraso");
  });

  it("a lista diz que last_message e' uma previa", async () => {
    responder();

    const renderer = await montar();

    const linha = texto(renderer, "inbox-conversa-cv1");
    expect(linha).toContain("Ana Silva");
    expect(linha).toContain("Bom dia, preciso de ajuda");
    expect(linha).toContain("(previa)");
  });

  it("conversa sem last_message mostra 'sem previa', nao string vazia", async () => {
    responder();

    const renderer = await montar();

    expect(texto(renderer, "inbox-conversa-cv2")).toContain("sem previa");
  });

  it("reply_blocked NAO e' erro: a conversa continua visivel e legivel", async () => {
    responder();
    mockAgentFetch.mockResolvedValueOnce({
      conversation: CONVERSAS[0],
      messages: [
        { id: "m1", conversation_id: "cv1", organization_id: "o1", direction: "inbound",
          author_type: "contact", body: "Bom dia, preciso de ajuda",
          created_at: "2026-10-19T00:00:00Z" },
      ],
      reply_blocked: true,
      reply_blocked_reason: "consentimento revogado para whatsapp",
    });

    const renderer = await montar();
    await agir(() => porTestId(renderer, "inbox-abrir-cv1")[0].props.onClick());

    // O bloqueio aparece com o motivo.
    const bloqueio = texto(renderer, "inbox-bloqueio");
    expect(bloqueio).toContain("resposta bloqueada");
    expect(bloqueio).toContain("consentimento revogado para whatsapp");
    // MAS a conversa e as mensagens continuam la'.
    expect(texto(renderer, "inbox-mensagens")).toContain("Bom dia, preciso de ajuda");
    // E o painel explica que ler nao esta' bloqueado.
    expect(texto(renderer, "inbox-bloqueio-nota")).toContain("continua legível");
    expect(porTestId(renderer, "inbox-detalhe-erro")).toHaveLength(0);
  });

  it("reply_blocked=false diz 'pode responder'", async () => {
    responder();
    mockAgentFetch.mockResolvedValueOnce({
      conversation: CONVERSAS[0],
      messages: [],
      reply_blocked: false,
    });

    const renderer = await montar();
    await agir(() => porTestId(renderer, "inbox-abrir-cv1")[0].props.onClick());

    expect(texto(renderer, "inbox-bloqueio")).toContain("pode responder");
    expect(porTestId(renderer, "inbox-bloqueio-nota")).toHaveLength(0);
    expect(texto(renderer, "inbox-sem-mensagens")).toContain("Não há mensagens");
  });

  it("mensagem deduplicada e' marcada e nao passa por nova", async () => {
    responder();
    mockAgentFetch.mockResolvedValueOnce({
      conversation: CONVERSAS[0],
      messages: [
        { id: "m1", conversation_id: "cv1", organization_id: "o1", direction: "inbound",
          author_type: "contact", body: "Ola", external_id: "ext-1",
          created_at: "2026-10-19T00:00:00Z", deduplicated: true },
      ],
      reply_blocked: false,
    });

    const renderer = await montar();
    await agir(() => porTestId(renderer, "inbox-abrir-cv1")[0].props.onClick());

    expect(texto(renderer, "inbox-dedup-m1")).toContain("não é mensagem nova");
  });

  it("o 503 aparece como inbox AUSENTE, nao como lista vazia", async () => {
    const erro = new Error("conversa nao existe");
    mockAgentFetch.mockRejectedValueOnce(erro).mockRejectedValueOnce(erro);

    const renderer = await montar();

    const semInbox = texto(renderer, "inbox-sem-inbox");
    expect(semInbox).toContain("não está ligado");
    expect(semInbox).toContain("não é «não há conversas»");
    expect(porTestId(renderer, "inbox-erro")).toHaveLength(0);
  });

  it("inbox ligado com zero conversas e' diferente de ausente", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ conversations: [], count: 0 })
      .mockResolvedValueOnce({ ...METRICAS, open_conversations: 0, pending_human: 0, sla_breached: 0, resolution_overdue: 0 });

    const renderer = await montar();

    expect(texto(renderer, "inbox-sem-conversas")).toContain("não há conversas");
    expect(porTestId(renderer, "inbox-sem-inbox")).toHaveLength(0);
  });

  it("o filtro de estado vai na query", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ conversations: [], count: 0 })
      .mockResolvedValueOnce(METRICAS)
      .mockResolvedValueOnce({ conversations: [], count: 0 })
      .mockResolvedValueOnce(METRICAS);

    const renderer = await montar();
    await agir(() =>
      porTestId(renderer, "inbox-filtro")[0].props.onChange({ target: { value: "pending" } }),
    );
    await agir(() => porTestId(renderer, "inbox-filtrar")[0].props.onClick());

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string);
    expect(urls).toContain("/api/agent/v1/inbox/conversations?status=pending");
  });

  it("as tres chamadas usam o prefixo explicito e sao GET", async () => {
    responder();

    await montar();

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string).sort();
    expect(urls).toEqual([
      "/api/agent/v1/inbox/conversations",
      "/api/agent/v1/inbox/metrics",
    ]);
    for (const chamada of mockAgentFetch.mock.calls) {
      const init = chamada[1] as RequestInit | undefined;
      expect(init?.method ?? "GET").toBe("GET");
    }
  });
});
