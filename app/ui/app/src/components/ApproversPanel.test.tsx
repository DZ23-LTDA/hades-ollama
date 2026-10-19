// Testes dos aprovadores de aprovação.
//
// Discriminantes: lista vazia = portão ABERTO (não é «sem aprovadores»); 403 é o portão de papel a
// funcionar; a trilha pode falhar e a resposta DECLARA isso; o actor é obrigatório; o método é PUT.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { ApproversPanel } from "@/components/ApproversPanel";
import {
  descreverDuasIdentidades,
  descreverPortao,
  descreverResultado,
  validarAtor,
} from "@/lib/approversClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<ApproversPanel />);
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
    porTestId(renderer, `aprovadores-${id}`)[0].props.onChange({ target: { value: valor } });
  });
}

async function clicar(renderer: ReturnType<typeof create>) {
  await act(async () => {
    porTestId(renderer, "aprovadores-gravar")[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes dos aprovadores", () => {
  it("actor vazio e' recusado e o erro diz PORQUE o actor e' exigido", () => {
    const t = validarAtor("   ");
    expect(t).toContain("actor_id");
    expect(t).toContain("ator identificado");
  });

  it("lista vazia NAO e' 'sem aprovadores': e' sem checagem de papel", () => {
    const t = descreverPortao([]);
    expect(t).toContain("NÃO passará por checagem de papel");
    expect(t).toContain("portão aberto");
    // E nao usa a expressao neutra.
    expect(t).not.toContain("sem aprovadores");
  });

  it("lista com itens diz que passara' por checagem de papel", () => {
    const t = descreverPortao(["a", "b"]);
    expect(t).toContain("2 aprovador");
    expect(t).toContain("passará por checagem de papel");
  });

  it("o resultado distingue trilha gravada de trilha FALHADA", () => {
    const ok = descreverResultado({ approvers: ["a"], count: 1, role_checked: true, changed_by: "u1", audit_recorded: true } as never);
    expect(ok).toContain("trilha gravada");
    expect(ok).not.toContain("NÃO GRAVADA");

    const falha = descreverResultado({ approvers: ["a"], count: 1, role_checked: true, changed_by: "u1", audit_recorded: false, audit_reason: "disco" } as never);
    expect(falha).toContain("TRILHA NÃO GRAVADA");
    expect(falha).toContain("disco");
  });

  it("o resultado mostra SEM portao quando role_checked e' falso", () => {
    const t = descreverResultado({ approvers: [], count: 0, role_checked: false, changed_by: "u1", audit_recorded: true } as never);
    expect(t).toContain("SEM portão de papel");
  });

  it("as duas identidades sao explicadas: corpo vs sessao", () => {
    const t = descreverDuasIdentidades();
    expect(t).toContain("changed_by");
    expect(t).toContain("corpo");
    expect(t).toContain("SESSÃO");
    expect(t).toContain("trilha");
  });
});

describe("ApproversPanel", () => {
  it("nao grava ao montar", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "aprovadores-pronto")).toContain("não foi lida");
  });

  it("actor vazio e' recusado antes de chamar", async () => {
    const r = await montar();
    await escrever(r, "lista", "u1\n");
    await clicar(r);

    expect(texto(r, "aprovadores-recusado")).toContain("ator identificado");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("usa PUT e vai para /service-agents/approval-approvers", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      approvers: ["u1", "u2"], count: 2, role_checked: true, changed_by: "boss", audit_recorded: true,
    } as never);
    const r = await montar();
    await escrever(r, "ator", "boss");
    await escrever(r, "lista", "u1\n u2 \n\n");
    await clicar(r);

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/service-agents/approval-approvers");
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    // PUT: e' substituicao de lista, nao criacao.
    expect(init.method).toBe("PUT");
    const corpo = JSON.parse(init.body ?? "{}");
    expect(corpo.actor_id).toBe("boss");
    // A lista e' limpa: sem vazios nem espacos.
    expect(corpo.approvers).toEqual(["u1", "u2"]);
  });

  it("lista vazia e' enviada como pedido legitimo e o painel DIZ que o portao abriu", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      approvers: [], count: 0, role_checked: false, changed_by: "boss", audit_recorded: true,
    } as never);
    const r = await montar();
    await escrever(r, "ator", "boss");
    await clicar(r);

    const corpo = JSON.parse((mockAgentFetch.mock.calls[0][1] as { body?: string }).body ?? "{}");
    expect(corpo.approvers).toEqual([]);

    // O painel nao pode mostrar isto como sucesso neutro.
    const t = texto(r, "aprovadores-portao-aberto");
    expect(t).toContain("NÃO passará por checagem de papel");
    expect(t).toContain("portão aberto");
  });

  it("403 e' o PORTAO DE PAPEL a funcionar, nao uma falha do pedido", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("403: approval requires organization owner or admin"));
    const r = await montar();
    await escrever(r, "ator", "viewer");
    await escrever(r, "lista", "eu-mesmo");
    await clicar(r);

    const t = texto(r, "aprovadores-proibido");
    expect(t).toContain("owner ou admin");
    expect(t).toContain("portão de papel");
    expect(porTestId(r, "aprovadores-erro")).toHaveLength(0);
  });

  it("trilha falhada e' DECLARADA, nao escondida atras de um 200", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      approvers: ["u1"], count: 1, role_checked: true, changed_by: "boss",
      audit_recorded: false, audit_reason: "trilha indisponivel",
    } as never);
    const r = await montar();
    await escrever(r, "ator", "boss");
    await escrever(r, "lista", "u1");
    await clicar(r);

    const t = texto(r, "aprovadores-trilha-falhou");
    expect(t).toContain("TRILHA NÃO REGISTOU");
    expect(t).toContain("trilha indisponivel");
    expect(t).toContain("invisível para quem audita");
  });

  it("as duas identidades aparecem no painel", async () => {
    const r = await montar();

    const t = texto(r, "aprovadores-duas-identidades");
    expect(t).toContain("changed_by");
    expect(t).toContain("SESSÃO");
  });
});
