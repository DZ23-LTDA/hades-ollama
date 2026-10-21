/**
 * FASE 28.8: testes do painel de CREDENCIAIS.
 *
 * O que estes testes travam:
 *
 *  1. um kit configurado mostra o resumo e a tabela;
 *  2. **"nao configurado" nao e' erro** -- aparece uma mensagem propria, nao um alerta;
 *  3. **um segredo mostra "conteudo NAO lido" com essas palavras** -- e' informacao, nao falha;
 *  4. um certificado mostra o sujeito e a validade que o servidor mediu;
 *  5. o aviso de fronteira vem do BACKEND e e' mostrado por inteiro;
 *  6. uma falha da rota aparece em `role="alert"`;
 *  7. kit vazio tem mensagem propria (nao e' o mesmo que "nao configurado").
 *
 * O QUE ESTES TESTES NAO PODEM PROVAR
 *
 * Nao provam que o backend responde o que o painel espera -- isso e' o E2E. Aqui mocka-se **apenas o
 * `agentFetch`**, nao o cliente inteiro:
 *
 *	*Um teste que substitui o cliente testa o mock, nao o produto.*
 */

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockAgentFetch = vi.fn();

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: (...args: unknown[]) => mockAgentFetch(...args),
}));

import { CredentialsPanel } from "@/components/CredentialsPanel";

/** Inventario de exemplo com um certificado publico e um segredo. */
function inventarioDeExemplo() {
  return {
    somente_leitura: true,
    configurado: true,
    credenciais: [
      {
        caminho_relativo: "apple/distribution.cer",
        tipo: "CERTIFICADO_X509" as const,
        tamanho_bytes: 1485,
        modificado_em: "2026-10-17T12:00:00Z",
        conteudo_lido: true,
        fingerprint_sha256: "abc123",
        sujeito: "Apple Distribution: leandro prado (K8G36CUP3S)",
        emissor: "Apple Worldwide Developer Relations CA",
        valido_ate: "2027-10-17T00:00:00Z",
        expirado: false,
      },
      {
        caminho_relativo: "apple/hades-apple-distribution.p12",
        tipo: "CHAVE_PRIVADA" as const,
        tamanho_bytes: 3512,
        modificado_em: "2026-10-17T12:00:00Z",
        conteudo_lido: false,
        expirado: false,
        nota: "conteudo NAO lido: apenas tamanho e data",
      },
    ],
    total: 2,
    lidos: 1,
    nao_lidos: 1,
    certificados: 1,
    expirados: 0,
    aviso: "inventario de dependencias declaradas",
    aviso_de_fronteira:
      "esta rota apenas INVENTARIA: os ficheiros de segredo NAO sao abertos. " +
      "O caminho vem da variavel de ambiente, nunca do pedido.",
  };
}

/** Texto de toda a arvore, para assercoes de conteudo. */
function textoDosNos(no: ReactTestRenderer): string {
  const partes: string[] = [];
  const visitar = (valor: unknown): void => {
    if (valor === null || valor === undefined) return;
    if (typeof valor === "string" || typeof valor === "number") {
      partes.push(String(valor));
      return;
    }
    if (Array.isArray(valor)) {
      valor.forEach(visitar);
      return;
    }
    if (typeof valor === "object" && "children" in valor) {
      visitar((valor as { children: unknown }).children);
    }
  };
  visitar(no.toJSON());
  return partes.join(" ");
}

async function montar(): Promise<ReactTestRenderer> {
  let arvore!: ReactTestRenderer;
  await act(async () => {
    arvore = create(<CredentialsPanel />);
  });
  return arvore;
}

beforeEach(() => {
  mockAgentFetch.mockReset();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("CredentialsPanel", () => {
  it("mostra o resumo e a tabela quando o kit esta' configurado", async () => {
    mockAgentFetch.mockResolvedValue(inventarioDeExemplo());
    const arvore = await montar();
    const texto = textoDosNos(arvore);

    expect(mockAgentFetch).toHaveBeenCalledWith("/api/agent/v1/credentials/status");
    expect(texto).toContain("2 ficheiro(s)");
    expect(texto).toContain("apple/distribution.cer");
    expect(texto).toContain("apple/hades-apple-distribution.p12");
    arvore.unmount();
  });

  // "Nao configurado" e' um estado do produto, nao um erro.
  it("distingue 'nao configurado' de erro", async () => {
    mockAgentFetch.mockResolvedValue({
      somente_leitura: true,
      configurado: false,
      credenciais: [],
      total: 0,
      lidos: 0,
      nao_lidos: 0,
      certificados: 0,
      expirados: 0,
      aviso: "o kit de credenciais nao esta' configurado neste servidor",
      aviso_de_fronteira: "x",
    });
    const arvore = await montar();

    expect(arvore.root.findByProps({ "data-testid": "credenciais-nao-configurado" })).toBeDefined();
    expect(() => arvore.root.findByProps({ "data-testid": "credenciais-erro" })).toThrow();
    expect(textoDosNos(arvore)).toContain("nao esta' configurado");
    arvore.unmount();
  });

  // Um segredo nao lido e' informacao, nao falha.
  it("diz 'conteudo NAO lido' para um segredo", async () => {
    mockAgentFetch.mockResolvedValue(inventarioDeExemplo());
    const arvore = await montar();
    const texto = textoDosNos(arvore);

    expect(texto).toContain("NAO lido");
    expect(texto).toContain("Apenas tamanho e data foram medidos");
    // E o painel NAO pode afirmar que o segredo foi verificado.
    expect(texto).not.toContain("verificado com sucesso");
    arvore.unmount();
  });

  it("mostra sujeito e validade do certificado publico", async () => {
    mockAgentFetch.mockResolvedValue(inventarioDeExemplo());
    const arvore = await montar();
    const texto = textoDosNos(arvore);

    expect(texto).toContain("Apple Distribution: leandro prado (K8G36CUP3S)");
    expect(texto).toContain("2027-10-17T00:00:00Z");
    expect(texto).toContain("Certificado");
    arvore.unmount();
  });

  // O aviso de fronteira vem do backend e nao e' reescrito pelo painel.
  it("mostra o aviso de fronteira do backend por inteiro", async () => {
    mockAgentFetch.mockResolvedValue(inventarioDeExemplo());
    const arvore = await montar();
    const aviso = arvore.root.findByProps({ "data-testid": "credenciais-aviso-fronteira" });

    const textoDoAviso = textoDosNos({
      toJSON: () => aviso.props.children,
    } as unknown as ReactTestRenderer);
    expect(textoDoAviso).toContain("os ficheiros de segredo NAO sao abertos");
    expect(textoDoAviso).toContain("nunca do pedido");
    arvore.unmount();
  });

  it("mostra uma falha em role=alert", async () => {
    mockAgentFetch.mockRejectedValue(new Error("Bad Request"));
    const arvore = await montar();

    const alerta = arvore.root.findByProps({ role: "alert" });
    expect(alerta).toBeDefined();
    expect(textoDosNos(arvore)).toContain("Bad Request");
    arvore.unmount();
  });

  it("distingue kit vazio de kit nao configurado", async () => {
    mockAgentFetch.mockResolvedValue({
      ...inventarioDeExemplo(),
      credenciais: [],
      total: 0,
      lidos: 0,
      nao_lidos: 0,
      certificados: 0,
    });
    const arvore = await montar();

    expect(arvore.root.findByProps({ "data-testid": "credenciais-vazio" })).toBeDefined();
    expect(() =>
      arvore.root.findByProps({ "data-testid": "credenciais-nao-configurado" }),
    ).toThrow();
    arvore.unmount();
  });
});
