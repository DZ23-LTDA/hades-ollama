// Testes do painel de sessao.
//
// Discriminantes centrais:
//   1. `mfa_enabled` AUSENTE le-se como `false`, nunca como «desconhecido»;
//   2. nenhum segredo de MFA aparece (a API nao os devolve);
//   3. `mode: "local"` nao e' sessao invalida.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { SessionPanel } from "@/components/SessionPanel";
import {
  descreverMFA,
  descreverModo,
  descreverPapel,
  mfaLigado,
  nomeDoUtilizador,
} from "@/lib/authClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<SessionPanel />);
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

const UTILIZADOR = {
  id: "u1", email: "ana@exemplo.com", name: "Ana Silva",
  created_at: "2026-10-01T00:00:00Z",
};

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de sessao", () => {
  it("`mfa_enabled` AUSENTE e' `false`, nunca desconhecido", () => {
    // A armadilha: `omitempty` faz `false` nao viajar.
    expect(mfaLigado({ ...UTILIZADOR })).toBe(false);
    expect(descreverMFA({ ...UTILIZADOR })).toContain("desligado");
    expect(descreverMFA({ ...UTILIZADOR })).not.toContain("desconhecido");
  });

  it("`mfa_enabled: true` e' ligado", () => {
    expect(mfaLigado({ ...UTILIZADOR, mfa_enabled: true })).toBe(true);
    expect(descreverMFA({ ...UTILIZADOR, mfa_enabled: true })).toContain("ligado");
  });

  it("`mode: local` nao e' sessao invalida", () => {
    const t = descreverModo({ authenticated: false, mode: "local" });
    expect(t).toContain("modo local");
    expect(t).toContain("sem autenticação exigida");
  });

  it("nome recorre a email e depois a id", () => {
    expect(nomeDoUtilizador(UTILIZADOR)).toBe("Ana Silva");
    expect(nomeDoUtilizador({ ...UTILIZADOR, name: "" })).toBe("ana@exemplo.com");
    expect(nomeDoUtilizador({ ...UTILIZADOR, name: "", email: "" })).toBe("u1");
  });

  it("papel ausente e' dito como nao declarado", () => {
    expect(descreverPapel(undefined)).toContain("não declarado");
    expect(descreverPapel({ role: "owner" })).toBe("papel: owner");
  });
});

describe("SessionPanel", () => {
  it("sessao autenticada mostra nome, email e MFA desligado", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      authenticated: true, user: UTILIZADOR, membership: { role: "owner" },
    });

    const renderer = await montar();

    expect(texto(renderer, "sessao-utilizador")).toContain("Ana Silva");
    expect(texto(renderer, "sessao-email")).toContain("ana@exemplo.com");
    expect(texto(renderer, "sessao-mfa")).toContain("desligado");
    expect(texto(renderer, "sessao-papel")).toContain("owner");
  });

  it("MFA ligado aparece como ligado", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      authenticated: true, user: { ...UTILIZADOR, mfa_enabled: true },
    });

    const renderer = await montar();

    expect(texto(renderer, "sessao-mfa")).toContain("ligado");
  });

  it("nenhum segredo de MFA aparece no ecra", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      authenticated: true,
      user: { ...UTILIZADOR, mfa_enabled: true },
    });

    const renderer = await montar();

    const bruto = JSON.stringify(renderer.toJSON());
    expect(bruto).not.toContain("mfa_secret");
    expect(bruto).not.toContain("recovery_codes");
    expect(bruto).not.toContain("Ciphertext");
  });

  it("`mode: local` nao e' tratado como erro", async () => {
    mockAgentFetch.mockResolvedValueOnce({ authenticated: false, mode: "local" });

    const renderer = await montar();

    expect(texto(renderer, "sessao-modo")).toContain("modo local");
    expect(texto(renderer, "sessao-sem-utilizador")).toContain("não é um erro");
    expect(porTestId(renderer, "sessao-erro")).toHaveLength(0);
  });

  it("a chamada usa o prefixo explicito e e' GET", async () => {
    mockAgentFetch.mockResolvedValueOnce({ authenticated: false, mode: "local" });

    await montar();

    expect(mockAgentFetch.mock.calls).toHaveLength(1);
    expect(mockAgentFetch.mock.calls[0][0]).toBe("/api/agent/v1/auth/session");
    const init = mockAgentFetch.mock.calls[0][1] as RequestInit | undefined;
    expect(init?.method ?? "GET").toBe("GET");
  });

  it("erro de rede aparece como erro, nao como sessao anonima", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("rede indisponivel"));

    const renderer = await montar();

    expect(texto(renderer, "sessao-erro")).toContain("rede indisponivel");
    expect(porTestId(renderer, "sessao-sem-utilizador")).toHaveLength(0);
  });
});
