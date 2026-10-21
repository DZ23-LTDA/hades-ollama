// Testes do registo de push.
//
// Discriminantes: o token NAO volta (garantia do tipo `json:"-"`); 501 ≠ 503; plataforma inventada
// e' recusada antes de chamar; o token sai do estado depois de enviado.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { NotificationsPanel } from "@/components/NotificationsPanel";
import {
  descreverAusenciaDoToken,
  descreverSubscricao,
  plataformasConhecidas,
  validarRegisto,
} from "@/lib/pushClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<NotificationsPanel />);
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

async function escreverToken(renderer: ReturnType<typeof create>, valor: string) {
  await act(async () => {
    porTestId(renderer, "notificacoes-token")[0].props.onChange({ target: { value: valor } });
  });
}

async function clicar(renderer: ReturnType<typeof create>) {
  await act(async () => {
    porTestId(renderer, "notificacoes-registar")[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes do push", () => {
  it("as plataformas conhecidas sao as tres do servidor", () => {
    expect([...plataformasConhecidas()]).toEqual(["ios", "android", "web"]);
  });

  it("token vazio e' recusado com o nome do campo", () => {
    expect(validarRegisto("   ", "web")).toContain("token");
  });

  it("plataforma inventada e' recusada, e a mensagem lista as validas", () => {
    const t = validarRegisto("tok", "symbian");
    expect(t).toContain("não reconhecida");
    expect(t).toContain("ios");
  });

  it("plataforma e' normalizada para minusculas", () => {
    expect(validarRegisto("tok", "ANDROID")).toBeNull();
  });

  it("a ausencia do token e' declarada como GARANTIA, nao como omissao", () => {
    const t = descreverAusenciaDoToken();
    expect(t).toContain("não o pode devolver");
    expect(t).toContain("garantia");
    expect(t).toContain("push.go");
    // A tag tem de aparecer: e' ela que prova a garantia.
    expect(t).toContain('json:"-"');
  });

  it("o resultado diz a subscricao, nao 'correu bem'", () => {
    const t = descreverSubscricao({ id: "sub-1", platform: "ios" } as never);
    expect(t).toContain("sub-1");
    expect(t).toContain("ios");
  });
});

describe("NotificationsPanel", () => {
  it("nao regista ao montar", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "notificacoes-pronto")).toContain("Nenhum dispositivo");
  });

  it("token vazio e' recusado antes de chamar", async () => {
    const r = await montar();
    await clicar(r);

    expect(texto(r, "notificacoes-recusado")).toContain("token");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("vai para /notifications/register com token e platform", async () => {
    mockAgentFetch.mockResolvedValueOnce({ id: "sub-1", platform: "web" } as never);
    const r = await montar();
    await escreverToken(r, "tok-abc");
    await clicar(r);

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(caminho).toBe("/api/agent/v1/notifications/register");
    expect(init.method).toBe("POST");
    const corpo = JSON.parse(init.body ?? "{}");
    expect(corpo.token).toBe("tok-abc");
    expect(corpo.platform).toBe("web");
  });

  it("o token e' LIMPO do estado depois de enviado", async () => {
    mockAgentFetch.mockResolvedValueOnce({ id: "sub-1", platform: "web" } as never);
    const r = await montar();
    await escreverToken(r, "tok-secreto");
    await clicar(r);

    // O input ficou vazio: o token nao continua em estado do React.
    const input = porTestId(r, "notificacoes-token")[0];
    expect(input.props.value).toBe("");
    // E o token nao aparece em NENHUM texto renderizado.
    expect(JSON.stringify(r.toJSON())).not.toContain("tok-secreto");
  });

  it("o resultado diz 201 e a subscricao -- e nao mostra o token", async () => {
    mockAgentFetch.mockResolvedValueOnce({ id: "sub-1", platform: "android" } as never);
    const r = await montar();
    await escreverToken(r, "tok-1");
    await clicar(r);

    const t = texto(r, "notificacoes-registado");
    expect(t).toContain("201");
    expect(t).toContain("sub-1");
    expect(t).not.toContain("tok-1");
  });

  it("501 e' NAO IMPLEMENTADO, nao falha temporaria", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("501: push service is not configured"));
    const r = await montar();
    await escreverToken(r, "tok-1");
    await clicar(r);

    const t = texto(r, "notificacoes-nao-implementado");
    expect(t).toContain("501");
    expect(t).toContain("configuração em falta");
    expect(porTestId(r, "notificacoes-erro")).toHaveLength(0);
  });

  it("a garantia do token aparece no painel", async () => {
    const r = await montar();

    const t = texto(r, "notificacoes-token-ausente");
    expect(t).toContain("garantia");
    expect(t).toContain('json:"-"');
  });
});
