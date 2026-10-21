/**
 * FASE 28.8: testes do painel de ASSINATURA.
 *
 * O que estes testes travam:
 *
 *  1. o painel diz que **só avalia**, antes de haver resultado;
 *  2. **não existe campo para chave privada nem para senha** — o painel só aceita um caminho de
 *     certificado público;
 *  3. **não existe botão de assinar** — só "Avaliar identidade";
 *  4. o botão está **desactivado sem caminho**;
 *  5. uma identidade **sem EKU** mostra a frase do EKU, não uma genérica;
 *  6. uma identidade **expirada** mostra "expirada" — não "falta o EKU";
 *  7. `nada_assinado: false` **grita** em vez de passar;
 *  8. os alvos suportados que o backend declara são mostrados.
 *
 * Mocka-se **apenas o `agentFetch`**, não o cliente.
 */

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockAgentFetch = vi.fn();

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: (...args: unknown[]) => mockAgentFetch(...args),
}));

import { SigningPanel } from "@/components/SigningPanel";

/** Avaliação de exemplo. */
function avaliacao(sobrescrever: Record<string, unknown> = {}) {
  return {
    tipo: "AUTHENTICODE",
    aptidao: "APTA",
    apta: true,
    sujeito: "CN=DZ23 LTDA, O=DZ23 LTDA",
    emissor: "CN=Test CA",
    valido_de: "2026-01-01T00:00:00Z",
    valido_ate: "2027-10-17T00:00:00Z",
    ekus: ["1.3.6.1.5.5.7.3.3"],
    tem_eku_necessario: true,
    motivo: "",
    ...sobrescrever,
  };
}

/** Plano de exemplo. */
function planoDeExemplo(sobrescrever: Record<string, unknown> = {}) {
  return {
    somente_leitura: true,
    nada_assinado: true,
    avaliacao: avaliacao(),
    alvos_suportados: ["AUTHENTICODE", "APPLE_DEVELOPER_ID", "ANDROID_APK", "OPENPGP"],
    aviso_de_fronteira:
      "esta rota apenas AVALIA se a identidade serve para o tipo pedido; nao le' a chave privada, " +
      "nao assina e nao escreve. A cadeia criptografica e a revogacao NAO sao validadas aqui.",
    ...sobrescrever,
  };
}

/** Texto de toda a arvore. */
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
    arvore = create(<SigningPanel />);
  });
  return arvore;
}

function botao(arvore: ReactTestRenderer) {
  return arvore.root.find(
    (no) => no.type === "button" && String(no.props.children).includes("Avaliar identidade"),
  );
}

/** Escreve o caminho do certificado. */
async function escreverCaminho(arvore: ReactTestRenderer): Promise<void> {
  await act(async () => {
    arvore.root.findByProps({ id: "assinatura-certificado" }).props.onChange({
      target: { value: "C:/certificados/distribution.cer" },
    });
  });
}

beforeEach(() => {
  mockAgentFetch.mockReset();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("SigningPanel", () => {
  it("diz que so' avalia antes de haver resultado", async () => {
    const arvore = await montar();
    const texto = textoDosNos(arvore);
    expect(texto).toContain("apenas");
    expect(texto).toContain("Nao assina");
    arvore.unmount();
  });

  // NAO EXISTE CAMPO PARA SEGREDO.
  it("nao tem campo para chave privada nem para senha", async () => {
    const arvore = await montar();
    const campos = arvore.root.findAll((no) => no.type === "input");
    const ids = campos.map((no) => no.props.id as string);
    expect(ids).toEqual([
      "assinatura-certificado",
      "assinatura-rotulo",
    ]);
    // E nenhum campo e' de password.
    expect(campos.every((no) => no.props.type !== "password")).toBe(true);
    const texto = textoDosNos(arvore).toLowerCase();
    expect(texto).not.toContain("chave privada:");
    expect(texto).not.toContain("senha");
    arvore.unmount();
  });

  // NAO EXISTE BOTAO DE ASSINAR.
  it("nao oferece nenhuma accao de assinar", async () => {
    mockAgentFetch.mockResolvedValue(planoDeExemplo());
    const arvore = await montar();
    await escreverCaminho(arvore);
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const botoes = arvore.root.findAll((no) => no.type === "button");
    expect(botoes).toHaveLength(1);
    const rotulo = String(botoes[0].props.children).toLowerCase();
    expect(rotulo).not.toContain("assinar");
    expect(rotulo).toContain("avaliar");
    arvore.unmount();
  });

  it("o botao esta' desactivado sem caminho", async () => {
    const arvore = await montar();
    expect(botao(arvore).props.disabled).toBe(true);
    await escreverCaminho(arvore);
    expect(botao(arvore).props.disabled).toBe(false);
    arvore.unmount();
  });

  // CADA NEGACAO TEM A SUA FRASE.
  it("identidade sem EKU mostra a frase do EKU", async () => {
    mockAgentFetch.mockResolvedValue(
      planoDeExemplo({
        avaliacao: avaliacao({
          aptidao: "SEM_EKU_DE_ASSINATURA",
          apta: false,
          tem_eku_necessario: false,
          motivo: "o certificado nao tem o EKU de assinatura de codigo",
        }),
      }),
    );
    const arvore = await montar();
    await escreverCaminho(arvore);
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("falta o EKU de assinatura de codigo");
    expect(texto).toContain("1.3.6.1.5.5.7.3.3");
    arvore.unmount();
  });

  // O CERTIFICADO EXPIRADO NAO PODE DIZER "FALTA O EKU".
  it("identidade expirada diz expirada, nao falta de EKU", async () => {
    mockAgentFetch.mockResolvedValue(
      planoDeExemplo({
        avaliacao: avaliacao({
          aptidao: "EXPIRADA",
          apta: false,
          tem_eku_necessario: true,
          valido_ate: "2026-01-01T00:00:00Z",
        }),
      }),
    );
    const arvore = await montar();
    await escreverCaminho(arvore);
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("expirada");
    // A frase de EKU so' aparece quando a aptidao E' essa.
    expect(texto).not.toContain("falta o EKU");
    arvore.unmount();
  });

  it("mostra os alvos suportados que o backend declara", async () => {
    mockAgentFetch.mockResolvedValue(planoDeExemplo());
    const arvore = await montar();
    await escreverCaminho(arvore);
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("Authenticode (Windows)");
    expect(texto).toContain("Apple Developer ID (macOS)");
    arvore.unmount();
  });

  it("mostra o aviso de fronteira do backend por inteiro", async () => {
    mockAgentFetch.mockResolvedValue(planoDeExemplo());
    const arvore = await montar();
    await escreverCaminho(arvore);
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("nao le' a chave privada");
    expect(texto).toContain("NAO sao validadas aqui");
    arvore.unmount();
  });

  // Se o servidor negar `nada_assinado`, o painel tem de gritar.
  it("grita se o servidor nao declarar nada_assinado", async () => {
    mockAgentFetch.mockResolvedValue(planoDeExemplo({ nada_assinado: false }));
    const arvore = await montar();
    await escreverCaminho(arvore);
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("ATENCAO");
    expect(texto).toContain("SIM (inesperado)");
    arvore.unmount();
  });

  it("envia o caminho do certificado e o tipo", async () => {
    mockAgentFetch.mockResolvedValue(planoDeExemplo());
    const arvore = await montar();
    await escreverCaminho(arvore);
    await act(async () => {
      botao(arvore).props.onClick();
    });
    expect(mockAgentFetch).toHaveBeenCalledTimes(1);
    const [caminhoRota, init] = mockAgentFetch.mock.calls[0] as [string, { body: string }];
    expect(caminhoRota).toBe("/api/agent/v1/sign/plan");
    const corpo = JSON.parse(init.body) as {
      tipo: string;
      identidade: Record<string, string>;
    };
    expect(corpo.tipo).toBe("AUTHENTICODE");
    expect(corpo.identidade.caminho_do_certificado).toBe("C:/certificados/distribution.cer");
    // O corpo NAO pode ter chave privada nem senha.
    expect(Object.keys(corpo.identidade).sort()).toEqual([
      "ambiente",
      "caminho_do_certificado",
      "rotulo",
    ]);
    arvore.unmount();
  });

  it("mostra uma falha em role=alert", async () => {
    mockAgentFetch.mockRejectedValue(new Error("400 certificado fora da raiz autorizada"));
    const arvore = await montar();
    await escreverCaminho(arvore);
    await act(async () => {
      botao(arvore).props.onClick();
    });
    expect(arvore.root.findByProps({ role: "alert" })).toBeDefined();
    expect(textoDosNos(arvore)).toContain("fora da raiz autorizada");
    arvore.unmount();
  });
});
