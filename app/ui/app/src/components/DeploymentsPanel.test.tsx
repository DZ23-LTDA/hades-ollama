// Testes do painel de providers de deploy.
//
// O caso mais importante deste ficheiro é o da URL COM CREDENCIAIS: o painel tem
// de mostrar a URL sem `user:senha@` e avisar que removeu. Um teste que só
// verificasse "renderiza a URL" passaria com o defeito presente.
//
//	*Um teste que não falharia com o defeito não é um teste: é uma cerimónia.*
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: vi.fn(),
}));

import { agentFetch } from "@/lib/agenticClient";
import { DeploymentsPanel } from "@/components/DeploymentsPanel";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<DeploymentsPanel />);
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
    if (Array.isArray(nodo)) {
      nodo.forEach(visitar);
      return;
    }
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

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("DeploymentsPanel", () => {
  it("lista vazia não é erro: diz que não há provider", async () => {
    mockAgentFetch.mockResolvedValueOnce({ providers: [] });

    const renderer = await montar();

    expect(texto(renderer, "deployments-vazio")).toContain("Nenhum provider");
    expect(porTestId(renderer, "deployments-erro")).toHaveLength(0);
  });

  it("trata providers null como vazio", async () => {
    mockAgentFetch.mockResolvedValueOnce({ providers: null });

    const renderer = await montar();

    expect(porTestId(renderer, "deployments-vazio")).toHaveLength(1);
  });

  it("mostra o NOME da variável de token e diz que é o nome, não o valor", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      providers: [
        {
          id: "p1",
          provider: "vercel",
          base_url: "https://api.vercel.com",
          token_env: "VERCEL_TOKEN",
        },
      ],
    });

    const renderer = await montar();

    const token = texto(renderer, "deployments-token-p1");
    expect(token).toContain("VERCEL_TOKEN");
    expect(token).toContain("nome da variável, não o valor");
  });

  it("REDIGE credenciais da URL e avisa que as removeu", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      providers: [
        {
          id: "p2",
          provider: "netlify",
          base_url: "https://utilizador:senha-secreta@api.netlify.com",
        },
      ],
    });

    const renderer = await montar();

    const url = texto(renderer, "deployments-url-p2");
    // A credencial NAO pode aparecer...
    expect(url).not.toContain("senha-secreta");
    expect(url).not.toContain("utilizador");
    // ...e o painel tem de DIZER que a removeu.
    expect(url).toContain("credenciais removidas");
    expect(texto(renderer, "deployments-aviso-redigido")).toContain("credenciais");
  });

  it("não avisa de redação quando não havia nada a redigir", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      providers: [
        { id: "p3", provider: "cloudflare", base_url: "https://api.cloudflare.com" },
      ],
    });

    const renderer = await montar();

    expect(porTestId(renderer, "deployments-aviso-redigido")).toHaveLength(0);
    expect(porTestId(renderer, "deployments-redigido-p3")).toHaveLength(0);
  });

  it("diz o timeout na unidade certa, e a omissão quando ausente", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      providers: [
        { id: "t1", provider: "a", base_url: "https://a.test", timeout_seconds: 30 },
        { id: "t2", provider: "b", base_url: "https://b.test", timeout_seconds: 120 },
        { id: "t3", provider: "c", base_url: "https://c.test" },
      ],
    });

    const renderer = await montar();

    expect(texto(renderer, "deployments-tabela")).toContain("30 s");
    expect(texto(renderer, "deployments-tabela")).toContain("2 min");
    expect(texto(renderer, "deployments-tabela")).toContain("timeout por omissão");
  });

  it("diz que não há token quando token_env está ausente", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      providers: [{ id: "s1", provider: "x", base_url: "https://x.test" }],
    });

    const renderer = await montar();

    expect(texto(renderer, "deployments-token-s1")).toContain("sem token configurado");
  });

  it("um provider configurado não é uma publicação feita", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      providers: [{ id: "f1", provider: "x", base_url: "https://x.test" }],
    });

    const renderer = await montar();

    const fronteira = texto(renderer, "deployments-fronteira");
    expect(fronteira).toContain("só lê");
    expect(fronteira).toContain("aprovação humana específica");
    expect(fronteira).toContain("não é uma publicação feita");
  });

  it("mostra o erro do servidor em vez de o engolir", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("unauthorized"));

    const renderer = await montar();

    expect(texto(renderer, "deployments-erro")).toContain("unauthorized");
  });

  it("só lê: nenhuma chamada de escrita", async () => {
    mockAgentFetch.mockResolvedValueOnce({ providers: [] });

    await montar();

    for (const chamada of mockAgentFetch.mock.calls) {
      const init = chamada[1] as RequestInit | undefined;
      expect(init?.method ?? "GET").toBe("GET");
    }
  });

  it("usa o path com prefixo explícito", async () => {
    mockAgentFetch.mockResolvedValueOnce({ providers: [] });

    await montar();

    expect(mockAgentFetch.mock.calls[0][0]).toBe("/api/agent/v1/deployments");
  });
});
