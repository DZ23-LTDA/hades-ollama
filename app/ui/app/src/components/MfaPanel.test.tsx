// Testes do MFA.
//
// Discriminantes: o painel NÃO finge reautenticação (declara o achado); desligar/codigos exigem
// confirmação local que é só trava de clique; `disable` e `codes` vão SEM corpo; os códigos de
// recuperação são mostrados uma vez e o aviso vem do servidor.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { MfaPanel } from "@/components/MfaPanel";
import {
  descreverAchadoDeReautenticacao,
  descreverCodigos,
  descreverEstado,
  descreverLimiteDaConfirmacao,
  validarSegredo,
} from "@/lib/authMfaClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<MfaPanel />);
  });
  return renderer;
}

function juntarTexto(nodo: unknown): string {
  if (typeof nodo === "string") return nodo;
  if (typeof nodo === "number") return String(nodo);
  if (Array.isArray(nodo)) return nodo.map(juntarTexto).join(" ");
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

async function escreverSegredo(renderer: ReturnType<typeof create>, valor: string) {
  await act(async () => {
    porTestId(renderer, "mfa-segredo")[0].props.onChange({ target: { value: valor } });
  });
}

async function confirmar(renderer: ReturnType<typeof create>) {
  await act(async () => {
    porTestId(renderer, "mfa-confirmar")[0].props.onChange({ target: { checked: true } });
  });
}

async function clicar(renderer: ReturnType<typeof create>, botao: string) {
  await act(async () => {
    porTestId(renderer, `mfa-${botao}`)[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes do MFA", () => {
  it("segredo vazio e' recusado com o nome do campo", () => {
    expect(validarSegredo("")).toContain("secret");
  });

  it("segredo que nao e' base32 e' recusado", () => {
    expect(validarSegredo("nao-e-base32!!!")).toContain("base32");
  });

  it("segredo curto e' recusado: nao e' uma chave", () => {
    expect(validarSegredo("ABCDEF")).toContain("menos de 16");
  });

  it("segredo base32 valido passa", () => {
    expect(validarSegredo("JBSWY3DPEHPK3PXP")).toBeNull();
  });

  it("o ACHADO e' enunciado como achado, com as palavras duras", () => {
    const t = descreverAchadoDeReautenticacao();
    expect(t).toContain("ACHADO");
    expect(t).toContain("verificam apenas a sessão");
    expect(t).toContain("pedido vazio");
    // A frase central.
    expect(t).toContain("não é um segundo factor");
  });

  it("o limite da confirmacao e' declarado: trava de clique, nao barreira", () => {
    const t = descreverLimiteDaConfirmacao();
    expect(t).toContain("trava de clique");
    expect(t).toContain("não uma barreira de segurança");
    expect(t).toContain("não pede senha");
  });

  it("o estado distingue ligado, desligado e nao declarado", () => {
    expect(descreverEstado({ enabled: true, account: "a@b" } as never)).toContain("LIGADO");
    expect(descreverEstado({ enabled: false } as never)).toContain("DESLIGADO");
    expect(descreverEstado({} as never)).toContain("não declarou");
  });

  it("zero codigos e' declarado como falha da operacao, nao como sucesso vazio", () => {
    const t = descreverCodigos({ recovery_codes: [] } as never);
    expect(t).toContain("zero códigos");
    expect(t).toContain("não produziu o que prometeu");
  });
});

describe("MfaPanel", () => {
  it("nao corre ao montar e mostra o achado", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "mfa-pronto")).toContain("Nenhuma operação");
    expect(texto(r, "mfa-achado")).toContain("ACHADO");
  });

  it("desligar SEM confirmacao e' recusado antes de chamar", async () => {
    const r = await montar();
    await clicar(r, "desligar");

    expect(texto(r, "mfa-recusado")).toContain("Confirme antes");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("gerar codigos SEM confirmacao e' recusado antes de chamar", async () => {
    const r = await montar();
    await clicar(r, "codigos");

    expect(texto(r, "mfa-recusado")).toContain("Confirme antes");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("ligar com segredo invalido e' recusado antes de chamar", async () => {
    const r = await montar();
    await escreverSegredo(r, "curto");
    await clicar(r, "ligar");

    expect(texto(r, "mfa-recusado")).toContain("menos de 16");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("ligar vai para /auth/mfa/enable com secret", async () => {
    mockAgentFetch.mockResolvedValueOnce({ enabled: true, issuer: "Ollama DZ23 Agentic", account: "a@b" } as never);
    const r = await montar();
    await escreverSegredo(r, "JBSWY3DPEHPK3PXP");
    await clicar(r, "ligar");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/auth/mfa/enable");
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body ?? "{}").secret).toBe("JBSWY3DPEHPK3PXP");
  });

  it("desligar vai SEM CORPO -- e e' isso que torna o achado grave", async () => {
    mockAgentFetch.mockResolvedValueOnce({ enabled: false } as never);
    const r = await montar();
    await confirmar(r);
    await clicar(r, "desligar");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/auth/mfa/disable");
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(init.method).toBe("POST");
    // Sem corpo: nem senha, nem TOTP, nem nada.
    expect(init.body).toBeUndefined();

    expect(texto(r, "mfa-desligado")).toContain("DESLIGADO");
  });

  it("gerar codigos vai SEM CORPO e mostra os codigos UMA vez", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      recovery_codes: ["AAAA-1111", "BBBB-2222"],
      warning: "store these codes securely; they are shown only once",
    } as never);
    const r = await montar();
    await confirmar(r);
    await clicar(r, "codigos");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/auth/mfa/recovery/generate");
    const init = mockAgentFetch.mock.calls[0][1] as { body?: string };
    expect(init.body).toBeUndefined();

    const t = texto(r, "mfa-codigos-estado");
    expect(t).toContain("201");
    expect(t).toContain("2 código");
    expect(texto(r, "mfa-lista-codigos")).toContain("AAAA-1111");
    // O aviso vem do servidor, nao e' inventado pelo painel.
    expect(texto(r, "mfa-aviso")).toContain("shown only once");
  });

  it("401 aparece como sem sessao", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("401: authenticated user is required"));
    const r = await montar();
    await escreverSegredo(r, "JBSWY3DPEHPK3PXP");
    await clicar(r, "ligar");

    expect(texto(r, "mfa-sem-sessao")).toContain("Não há sessão autenticada");
  });

  it("o limite da confirmacao aparece no painel", async () => {
    const r = await montar();

    expect(texto(r, "mfa-limite")).toContain("trava de clique");
  });
});
