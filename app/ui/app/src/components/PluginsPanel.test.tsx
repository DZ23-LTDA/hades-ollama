// Testes do painel de plugins.
//
// Discriminantes centrais:
//   1. `requested_scopes` do ESTADO nunca sao apresentados como concedidos;
//   2. `ready_to_use: false` mostra as razoes, nao so' um vermelho;
//   3. `trusted` e `content_verified` sao declarados separadamente;
//   4. o formato e' o contrato aceito, nao uma sugestao.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { PluginsPanel } from "@/components/PluginsPanel";
import {
  descreverFormato,
  descreverIntegridade,
  descreverProntidao,
  descreverScopes,
  descreverScopesPedidos,
} from "@/lib/pluginsClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<PluginsPanel />);
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

const FORMATO = {
  schema: "hades-plugin/v1",
  format_version: "1.0",
  kinds: ["connector", "mcp", "remote-mcp", "skill", "bundle"],
  runtimes: ["python", "python3", "node", "javascript", "js"],
};

const MANIFESTO = {
  id: "p1", version: "1.2.0", name: "Leitor de PDF", kind: "skill",
  content_sha256: "abc",
};

const INSTALACAO = {
  organization_id: "o1",
  manifest: MANIFESTO,
  digest: "sha256:abc",
  trusted: true,
  enabled: true,
  requested_scopes: ["chat.read", "fs.read", "net.out"],
  granted_scopes: ["chat.read", "fs.read"],
  installed_at: "2026-10-01T00:00:00Z",
  updated_at: "2026-10-01T00:00:00Z",
  content_verified: true,
};

const ESTADO_OK = {
  plugin_id: "p1", version: "1.2.0", kind: "skill", state: "installed",
  enabled: true, trusted: true, content_verified: true,
  capabilities: ["skills"],
  requested_scopes: ["chat.read", "fs.read", "net.out"],
  ready_to_use: true,
};

const ESTADO_BLOQUEADO = {
  ...ESTADO_OK,
  plugin_id: "p2",
  trusted: false,
  content_verified: false,
  ready_to_use: false,
  reasons: ["assinatura ausente", "scope net.out nao concedido"],
};

function responder(estados = [ESTADO_OK]) {
  mockAgentFetch
    .mockResolvedValueOnce(FORMATO)
    .mockResolvedValueOnce({ organization_id: "o1", plugins: [INSTALACAO] })
    .mockResolvedValueOnce({ organization_id: "o1", plugins: estados });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de plugins", () => {
  it("`requested_scopes` do estado NAO sao apresentados como concedidos", () => {
    const t = descreverScopesPedidos(["chat.read", "net.out"]);
    expect(t).toContain("pediu");
    expect(t).toContain("não informa o que foi concedido");
    expect(t).not.toContain("concedido:");
  });

  it("pedido vs concedido nomeia o que FALTA", () => {
    const t = descreverScopes(["chat.read", "net.out"], ["chat.read"]);
    expect(t).toContain("2 pedido(s)");
    expect(t).toContain("1 concedido(s)");
    expect(t).toContain("NÃO concedido(s): net.out");
  });

  it("tudo concedido nao inventa uma falta", () => {
    expect(descreverScopes(["a"], ["a"])).toContain("pedido(s) e concedido(s)");
  });

  it("`ready_to_use: false` mostra as razoes", () => {
    const t = descreverProntidao(ESTADO_BLOQUEADO as never);
    expect(t).toContain("NÃO está pronto");
    expect(t).toContain("assinatura ausente");
    expect(t).toContain("net.out nao concedido");
  });

  it("sem razoes, diz que o servidor nao as declarou", () => {
    const t = descreverProntidao({ ...ESTADO_BLOQUEADO, reasons: [] } as never);
    expect(t).toContain("motivo não declarado");
  });

  it("`trusted` e `content_verified` sao coisas diferentes", () => {
    expect(descreverIntegridade({ trusted: true, content_verified: false })).toContain(
      "conteúdo NÃO foi verificado",
    );
    expect(descreverIntegridade({ trusted: false, content_verified: true })).toContain(
      "NÃO é confiável",
    );
    expect(descreverIntegridade({ trusted: false, content_verified: false })).toContain(
      "NÃO confiável e conteúdo NÃO verificado",
    );
  });

  it("o formato declara kinds e runtimes", () => {
    const t = descreverFormato(FORMATO);
    expect(t).toContain("hades-plugin/v1");
    expect(t).toContain("connector");
    expect(t).toContain("remote-mcp");
    expect(t).toContain("python3");
  });
});

describe("PluginsPanel", () => {
  it("mostra o formato aceito", async () => {
    responder();

    const renderer = await montar();

    expect(texto(renderer, "plugins-formato")).toContain("hades-plugin/v1");
    expect(texto(renderer, "plugins-formato")).toContain("skill");
  });

  it("a instalacao mostra o que foi PEDIDO e NAO concedido", async () => {
    responder();

    const renderer = await montar();

    const t = texto(renderer, "plugins-scopes-p1");
    expect(t).toContain("NÃO concedido(s): net.out");
  });

  it("o estado mostra os pedidos sem os chamar de concedidos", async () => {
    responder();

    const renderer = await montar();

    const t = texto(renderer, "plugins-pedidos-p1");
    expect(t).toContain("pediu");
    expect(t).toContain("não informa o que foi concedido");
  });

  it("plugin bloqueado mostra a prontidao E as razoes", async () => {
    responder([ESTADO_BLOQUEADO] as never);

    const renderer = await montar();

    const t = texto(renderer, "plugins-prontidao-p2");
    expect(t).toContain("NÃO está pronto");
    expect(t).toContain("assinatura ausente");
  });

  it("o 503 aparece como registro AUSENTE, nao como zero plugins", async () => {
    const erro = new Error("registro de plugins nao esta' ligado neste runtime");
    mockAgentFetch.mockRejectedValueOnce(erro).mockRejectedValueOnce(erro).mockRejectedValueOnce(erro);

    const renderer = await montar();

    const sem = texto(renderer, "plugins-sem-registro");
    expect(sem).toContain("não está ligado");
    expect(sem).toContain("não é «não há plugins instalados»");
    expect(porTestId(renderer, "plugins-erro")).toHaveLength(0);
  });

  it("as tres chamadas usam o prefixo explicito e sao GET", async () => {
    responder();

    await montar();

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string).sort();
    expect(urls).toEqual([
      "/api/agent/v1/plugins",
      "/api/agent/v1/plugins/format",
      "/api/agent/v1/plugins/status",
    ]);
    for (const chamada of mockAgentFetch.mock.calls) {
      const init = chamada[1] as RequestInit | undefined;
      expect(init?.method ?? "GET").toBe("GET");
    }
  });
});
