// Testes do painel de aprovacoes humanas.
//
// Os tres discriminantes:
//   1. `role_checked=false` aparece como «sem lista de aprovadores» -- nao fica em silencio;
//   2. quorum aparece como «2 de 3» -- nao como «pendente» a seco;
//   3. a notificacao sem canal configurado aparece -- a fila existe e ninguem foi avisado.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { ApprovalsPanel } from "@/components/ApprovalsPanel";
import {
  descreverIdade,
  descreverNotificacao,
  descreverQuorum,
  descreverVerificacaoDePapel,
  truncarFingerprint,
} from "@/lib/approvalsClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<ApprovalsPanel />);
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

const APROVACAO = {
  id: "ap1", organization_id: "o1", tool_name: "finance.transfer",
  fingerprint: "sha256:abcdef1234567890abcdef", status: "pending",
  requested_at: "2026-10-19T00:00:00Z", role_checked: false,
  required_approvals: 3, value_cents: 500000,
  decisions: [
    { actor_id: "u1", decision: "approved", decided_at: "2026-10-19T01:00:00Z" },
    { actor_id: "u2", decision: "approved", decided_at: "2026-10-19T02:00:00Z" },
  ],
};

const APROVACAO_VERIFICADA = {
  ...APROVACAO,
  id: "ap2",
  status: "approved",
  role_checked: true,
  required_approvals: 1,
  decisions: [{ actor_id: "u1", decision: "approved" }],
};

const FILA = {
  organization_id: "o1",
  pending: 2,
  overdue: 1,
  oldest_seconds: 7200,
  pending_ttl_seconds: 86400,
  expired: 1,
  items: [
    { id: "ap1", tool_name: "finance.transfer", approvals_given: 2, approvals_needed: 3,
      age_seconds: 7200, overdue: true },
  ],
};

const NOTIFICACAO_OK = { configurado: true, entregues: 5, ultimo_erro: "" };

function responder(aprovacoes = [APROVACAO], notificacao = NOTIFICACAO_OK) {
  mockAgentFetch
    .mockResolvedValueOnce({ approvals: aprovacoes, count: aprovacoes.length, queue: FILA, notification: notificacao })
    .mockResolvedValueOnce({ persisted: true, entries: 10 });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes de aprovacao", () => {
  it("role_checked=false e' declarado, nao escondido", () => {
    expect(descreverVerificacaoDePapel(false)).toContain("sem lista de aprovadores");
    expect(descreverVerificacaoDePapel(true)).toContain("lista de aprovadores");
    expect(descreverVerificacaoDePapel(undefined)).toContain("sem informacao");
  });

  it("quorum e' parcial: 2 de 3", () => {
    expect(descreverQuorum(2, 3)).toBe("2 de 3 aprovacoes");
    expect(descreverQuorum(1, 1)).toBe("decisao unica");
  });

  it("idade em segundos vira texto legivel", () => {
    expect(descreverIdade(45)).toBe("45 s");
    expect(descreverIdade(7200)).toBe("2 h");
    expect(descreverIdade(90)).toBe("1 min");
  });

  it("notificacao sem canal diz que a fila nao avisa ninguem", () => {
    expect(
      descreverNotificacao({ configurado: false, entregues: 0, ultimo_erro: "" }),
    ).toContain("nao avisa ninguem");
    expect(
      descreverNotificacao({ configurado: true, entregues: 5, ultimo_erro: "timeout" }),
    ).toContain("ultimo aviso falhou");
  });

  it("o fingerprint e' truncado -- identifica, nao expoe", () => {
    const truncado = truncarFingerprint("sha256:abcdef1234567890abcdef");
    expect(truncado).toContain("…");
    expect(truncado.length).toBeLessThan("sha256:abcdef1234567890abcdef".length);
  });
});

describe("ApprovalsPanel", () => {
  it("mostra a fila com pendentes, em atraso e a idade", async () => {
    responder();

    const renderer = await montar();

    const fila = texto(renderer, "aprovacoes-fila");
    expect(fila).toContain("2 pendente(s)");
    expect(fila).toContain("1 em atraso");
    expect(fila).toContain("2 h");
  });

  it("role_checked=false aparece como 'sem lista de aprovadores'", async () => {
    responder();

    const renderer = await montar();

    const papel = texto(renderer, "aprovacao-papel-ap1");
    expect(papel).toContain("sem lista de aprovadores");
    expect(papel).toContain("apenas ator identificado");
  });

  it("role_checked=true diz que foi verificado", async () => {
    responder([APROVACAO_VERIFICADA]);

    const renderer = await montar();

    expect(texto(renderer, "aprovacao-papel-ap2")).toContain("verificado contra lista");
  });

  it("quorum aparece como '2 de 3', nao como 'pendente' a seco", async () => {
    responder();

    const renderer = await montar();

    const linha = texto(renderer, "aprovacao-ap1");
    expect(linha).toContain("2 de 3 aprovacoes");
    expect(linha).toContain("finance.transfer");
    // O fingerprint inteiro NAO aparece.
    expect(linha).not.toContain("abcdef1234567890abcdef");
  });

  it("a fila marca o item que passou do limiar", async () => {
    responder();

    const renderer = await montar();

    const itens = texto(renderer, "aprovacoes-fila-itens");
    expect(itens).toContain("PASSOU DO LIMIAR");
    expect(itens).toContain("2 de 3");
  });

  it("notificacao sem canal e' avisada -- a fila existe e ninguem foi avisado", async () => {
    responder([APROVACAO], { configurado: false, entregues: 0, ultimo_erro: "" });

    const renderer = await montar();

    expect(texto(renderer, "aprovacoes-notificacao")).toContain("nao avisa ninguem");
  });

  it("o 503 aparece como store AUSENTE, nao como fila vazia", async () => {
    const erro = new Error("store de aprovação humana não está configurado");
    mockAgentFetch.mockRejectedValueOnce(erro);

    const renderer = await montar();

    const semStore = texto(renderer, "aprovacoes-sem-store");
    expect(semStore).toContain("não está configurado");
    expect(semStore).toContain("não é «não há aprovações pendentes»");
    expect(porTestId(renderer, "aprovacoes-erro")).toHaveLength(0);
  });

  it("store ligado sem aprovacoes e' diferente de ausente", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({
        approvals: [], count: 0,
        queue: { ...FILA, pending: 0, overdue: 0, expired: 0, oldest_seconds: 0, items: [] },
        notification: NOTIFICACAO_OK,
      })
      .mockResolvedValueOnce({ persisted: true });

    const renderer = await montar();

    // O filtro inicial e' `pending`: a lista vazia nomeia o filtro, nao afirma
    // que nao ha aprovacoes nenhumas. As duas coisas sao diferentes.
    expect(texto(renderer, "aprovacoes-lista-vazia")).toContain("pendente");
    expect(porTestId(renderer, "aprovacoes-lista")).toHaveLength(0);
    expect(porTestId(renderer, "aprovacoes-sem-store")).toHaveLength(0);
  });

  it("a auditoria sem trilha duravel e' avisada, e a sua falha nao apaga a fila", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({
        approvals: [APROVACAO], count: 1, queue: FILA, notification: NOTIFICACAO_OK,
      })
      .mockRejectedValueOnce(new Error("trilha de auditoria durável não está configurada"));

    const renderer = await montar();

    // A fila continua visivel.
    expect(texto(renderer, "aprovacoes-fila")).toContain("pendente");
    // E a auditoria diz que falhou.
    expect(texto(renderer, "aprovacoes-auditoria-erro")).toContain("Trilha de auditoria");
  });

  it("auditoria nao persistida diz que nada e' guardado", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({
        approvals: [], count: 0,
        queue: { ...FILA, pending: 0, overdue: 0, expired: 0, oldest_seconds: 0, items: [] },
        notification: NOTIFICACAO_OK,
      })
      .mockResolvedValueOnce({ persisted: false });

    const renderer = await montar();

    expect(texto(renderer, "aprovacoes-auditoria")).toContain("nada é guardado");
  });

  it("o filtro de estado vai na query", async () => {
    responder();
    responder();

    const renderer = await montar();
    await agir(() =>
      porTestId(renderer, "aprovacoes-filtro")[0].props.onChange({ target: { value: "denied" } }),
    );
    await agir(() => porTestId(renderer, "aprovacoes-filtrar")[0].props.onClick());

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string);
    expect(urls).toContain("/api/agent/v1/service-agents/approvals?status=denied");
  });

  it("as rotas usam o prefixo explicito e sao GET", async () => {
    responder();

    await montar();

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string).sort();
    expect(urls).toEqual([
      "/api/agent/v1/service-agents/approvals?status=pending",
      "/api/agent/v1/service-agents/audit",
    ]);
    for (const chamada of mockAgentFetch.mock.calls) {
      const init = chamada[1] as RequestInit | undefined;
      expect(init?.method ?? "GET").toBe("GET");
    }
  });
});
