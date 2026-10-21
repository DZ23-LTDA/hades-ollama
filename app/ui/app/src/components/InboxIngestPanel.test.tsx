// Testes da ingestão de inbox.
//
// Discriminantes: 201 nova e 200 reentrega são ESTADOS DISTINTOS; sem external_id não há deduplicação e
// o painel diz isso; o corpo só leva external_id quando existe.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { InboxIngestPanel } from "@/components/InboxIngestPanel";
import {
  descreverDeduplicacao,
  descreverEntrega,
  validarMensagem,
} from "@/lib/inboxIngestClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<InboxIngestPanel />);
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
  const out: string[] = [];
  const visitar = (nodo: unknown) => {
    if (Array.isArray(nodo)) { nodo.forEach(visitar); return; }
    if (nodo === null || typeof nodo !== "object") return;
    const props = (nodo as { props?: Record<string, unknown> }).props;
    if (props && props["data-testid"] === id) out.push(juntarTexto(nodo));
    visitar((nodo as { children?: unknown }).children);
  };
  visitar(renderer.toJSON());
  return out.join(" ");
}

function porTestId(renderer: ReturnType<typeof create>, id: string) {
  return renderer.root.findAll((n) => n.props["data-testid"] === id);
}

async function escrever(renderer: ReturnType<typeof create>, id: string, valor: string) {
  await act(async () => {
    porTestId(renderer, `inbox-ingest-${id}`)[0].props.onChange({ target: { value: valor } });
  });
}

async function clicar(renderer: ReturnType<typeof create>) {
  await act(async () => {
    porTestId(renderer, "inbox-ingest-receber")[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes do inbox", () => {
  it("canal vazio e' recusado com o nome do campo", () => {
    expect(validarMensagem("", "oi")).toContain("channel");
  });

  it("corpo vazio e' recusado com o nome do campo", () => {
    expect(validarMensagem("whatsapp", "   ")).toContain("body");
  });

  it("sem external_id NAO ha' deduplicacao, e o texto di-lo", () => {
    const t = descreverDeduplicacao("");
    expect(t).toContain("não há deduplicação");
    expect(t).toContain("não inventa a chave");
    expect(t).toContain("segunda mensagem");
  });

  it("com external_id a deduplicacao e' activa e nomeia a chave", () => {
    const t = descreverDeduplicacao("wamid-123");
    expect(t).toContain("wamid-123");
    expect(t).toContain("não cria segunda mensagem");
  });

  it("201 e 200 sao DESCRITOS de forma diferente", () => {
    const nova = descreverEntrega({ message: { id: "m1" }, deduplicated: false } as never);
    expect(nova).toContain("201");
    expect(nova).toContain("nova");
    expect(nova).toContain("criada");

    const reentrega = descreverEntrega({ message: { id: "m1" }, deduplicated: true } as never);
    expect(reentrega).toContain("200");
    expect(reentrega).toContain("reentrega");
    expect(reentrega).toContain("já existia");
    // E nao se anuncia como "nova": o marcador do 201 nao aparece.
    // (O texto DIZ "nao foi criada outra" -- isso e' a negacao, e e' o que se quer.)
    expect(reentrega).not.toContain("· nova ·");
  });
});

describe("InboxIngestPanel", () => {
  it("nao recebe ao montar", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "inbox-ingest-pronto")).toContain("Nada recebido");
  });

  it("corpo vazio e' recusado antes de chamar", async () => {
    const r = await montar();
    await clicar(r);

    expect(texto(r, "inbox-ingest-recusada")).toContain("body");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("vai para /inbox/messages com channel e body", async () => {
    mockAgentFetch.mockResolvedValueOnce({ message: { id: "m1" }, deduplicated: false } as never);
    const r = await montar();
    await escrever(r, "corpo", "olá");
    await clicar(r);

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/inbox/messages");
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(init.method).toBe("POST");
    const corpo = JSON.parse(init.body ?? "{}");
    expect(corpo.channel).toBe("whatsapp");
    expect(corpo.body).toBe("olá");
    // Sem external_id informado, o campo NAO vai.
    expect(corpo).not.toHaveProperty("external_id");
  });

  it("nova e' 201 e mostra o estado 'nova'", async () => {
    mockAgentFetch.mockResolvedValueOnce({ message: { id: "m1" }, deduplicated: false } as never);
    const r = await montar();
    await escrever(r, "corpo", "olá");
    await clicar(r);

    const t = texto(r, "inbox-ingest-nova");
    expect(t).toContain("201");
    expect(t).toContain("m1");
    // E nao aparece como reentrega.
    expect(porTestId(r, "inbox-ingest-reentrega")).toHaveLength(0);
  });

  it("reentrega e' 200 e NAO e' mostrada como nova", async () => {
    mockAgentFetch.mockResolvedValueOnce({ message: { id: "m1" }, deduplicated: true } as never);
    const r = await montar();
    await escrever(r, "corpo", "olá");
    await escrever(r, "externo", "wamid-1");
    await clicar(r);

    const t = texto(r, "inbox-ingest-reentrega");
    expect(t).toContain("200");
    expect(t).toContain("já existia");
    expect(t).toContain("não foi criada uma segunda conversa");
    // E nao aparece como nova.
    expect(porTestId(r, "inbox-ingest-nova")).toHaveLength(0);
  });

  it("external_id vai no corpo quando informado", async () => {
    mockAgentFetch.mockResolvedValueOnce({ message: { id: "m1" }, deduplicated: false } as never);
    const r = await montar();
    await escrever(r, "corpo", "olá");
    await escrever(r, "externo", "wamid-9");
    await clicar(r);

    const corpo = JSON.parse((mockAgentFetch.mock.calls[0][1] as { body?: string }).body ?? "{}");
    expect(corpo.external_id).toBe("wamid-9");
  });

  it("503 aparece como armazem ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("503: inbox store unavailable"));
    const r = await montar();
    await escrever(r, "corpo", "olá");
    await clicar(r);

    expect(texto(r, "inbox-ingest-ausente")).toContain("não está ligado");
  });

  it("a deduplicacao e' explicada conforme o external_id", async () => {
    const r = await montar();
    expect(texto(r, "inbox-ingest-dedup")).toContain("não há deduplicação");

    await escrever(r, "externo", "wamid-1");
    expect(texto(r, "inbox-ingest-dedup")).toContain("wamid-1");
  });
});
