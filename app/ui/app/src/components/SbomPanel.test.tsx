/**
 * FASE 28.8: testes do painel do SBOM.
 *
 * O que estes testes travam:
 *
 *  1. o painel diz que o SBOM é **mínimo** antes de haver resultado;
 *  2. **não tem campo de raiz** — a raiz é do servidor;
 *  3. **não existe botão de assinar nem de atestar**;
 *  4. um inventário com componentes mostra-os e distingue **sem licença**;
 *  5. **`nao_produzido` é mostrado por extenso**;
 *  6. um inventário **de zero componentes não é erro** — é um resultado;
 *  7. `assinado`/`provenance` verdadeiros **gritam** em vez de passar;
 *  8. os tipos de componente vêm do backend (`MODULO_GO`/`PACOTE_NPM`/`BINARIO`), não inventados.
 *
 * Mocka-se **apenas o `agentFetch`**, não o cliente.
 */

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockAgentFetch = vi.fn();

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: (...args: unknown[]) => mockAgentFetch(...args),
}));

import { SbomPanel } from "@/components/SbomPanel";

/** Relatório de exemplo, com os tipos REAIS do backend. */
function relatorioDeExemplo(sobrescrever: Record<string, unknown> = {}) {
  return {
    gerado: true,
    assinado: false,
    provenance: false,
    inventario: {
      raiz: "/projecto",
      medido_em: "2026-10-18T22:00:00Z",
      componentes: [
        {
          nome: "github.com/gin-gonic/gin",
          versao: "v1.10.0",
          tipo: "MODULO_GO",
          direto: true,
          origem: "go.mod",
          declarado_em: "go.mod",
        },
        {
          nome: "left-pad",
          versao: "1.3.0",
          tipo: "PACOTE_NPM",
          licenca: "MIT",
          direto: false,
          origem: "package.json",
          declarado_em: "package.json",
        },
      ],
      total: 2,
      por_tipo: { MODULO_GO: 1, PACOTE_NPM: 1 },
      sem_licenca: 1,
      ficheiros_lidos: ["go.mod", "package.json"],
      nao_produzido: [
        "dependencias transitivas",
        "THIRD_PARTY_NOTICES",
        "provenance e attestation",
      ],
      aviso: "este inventario lista as dependencias DECLARADAS nos ficheiros de origem",
    },
    assinatura_do_inventario: ["total=2", "sem_licenca=1"],
    aviso_de_fronteira:
      "este e' um SBOM MINIMO: cobre `go.mod` e `package.json`. NAO inclui dependencias " +
      "transitivas, THIRD_PARTY_NOTICES, provenance, assinatura criptografica nem attestation. " +
      "`assinado` e `provenance` sao sempre false. A raiz vem da variavel de ambiente " +
      "`HADES_RAIZ_SBOM`, nunca do pedido.",
    ...sobrescrever,
  };
}

/** Relatório de inventário vazio. */
function relatorioVazio() {
  return relatorioDeExemplo({
    inventario: {
      raiz: "/projecto-vazio",
      medido_em: "2026-10-18T22:00:00Z",
      componentes: null,
      total: 0,
      por_tipo: {},
      sem_licenca: 0,
      ficheiros_lidos: [],
      nao_produzido: ["nenhuma fonte de dependencia encontrada (go.mod, package.json)"],
      aviso: "nenhuma dependencia declarada encontrada",
    },
    assinatura_do_inventario: [],
  });
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
    arvore = create(<SbomPanel />);
  });
  return arvore;
}

function botao(arvore: ReactTestRenderer) {
  return arvore.root.find(
    (no) => no.type === "button" && String(no.props.children).includes("Gerar inventario"),
  );
}

beforeEach(() => {
  mockAgentFetch.mockReset();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("SbomPanel", () => {
  it("diz que o SBOM e' minimo antes de haver resultado", async () => {
    const arvore = await montar();
    const texto = textoDosNos(arvore);
    expect(texto).toContain("minimo");
    expect(texto).toContain("Nao assina");
    expect(texto).toContain("nunca deste painel");
    arvore.unmount();
  });

  // NAO TEM CAMPO DE RAIZ.
  it("nao tem campo para a raiz do projecto", async () => {
    const arvore = await montar();
    const campos = arvore.root.findAll((no) => no.type === "input");
    expect(campos).toHaveLength(0);
    arvore.unmount();
  });

  // NAO EXISTE BOTAO DE ASSINAR NEM DE ATESTAR.
  it("nao oferece nenhuma accao de assinar ou atestar", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const botoes = arvore.root.findAll((no) => no.type === "button");
    expect(botoes).toHaveLength(1);
    const rotulo = String(botoes[0].props.children).toLowerCase();
    expect(rotulo).not.toContain("assinar");
    expect(rotulo).not.toContain("atestar");
    expect(rotulo).toContain("gerar");
    arvore.unmount();
  });

  it("mostra os componentes e distingue sem licenca", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("github.com/gin-gonic/gin");
    expect(texto).toContain("v1.10.0");
    expect(texto).toContain("modulo Go");
    expect(texto).toContain("pacote npm");
    // O sem licenca e' dito, nao escondido.
    expect(texto).toContain("sem licenca declarada");
    expect(arvore.root.findByProps({ "data-testid": "sbom-sem-licenca" }).props.children).toBe(1);
    arvore.unmount();
  });

  // O QUE NAO FOI PRODUZIDO E' DITO.
  it("mostra nao_produzido por extenso", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("dependencias transitivas");
    expect(texto).toContain("THIRD_PARTY_NOTICES");
    expect(texto).toContain("provenance e attestation");
    arvore.unmount();
  });

  it("mostra o aviso de fronteira do backend por inteiro", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("SBOM MINIMO");
    expect(texto).toContain("sempre false");
    expect(texto).toContain("nunca do pedido");
    arvore.unmount();
  });

  // UM INVENTARIO VAZIO NAO E' ERRO.
  it("um inventario de zero componentes nao e' erro", async () => {
    mockAgentFetch.mockResolvedValue(relatorioVazio());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    // Sem role=alert.
    expect(arvore.root.findAll((no) => no.props.role === "alert")).toHaveLength(0);
    const texto = textoDosNos(arvore);
    expect(texto).toContain("Nenhum componente encontrado");
    expect(arvore.root.findByProps({ "data-testid": "sbom-total" }).props.children).toBe(0);
    // E diz o que faltou encontrar.
    expect(texto).toContain("nenhuma fonte de dependencia encontrada");
    arvore.unmount();
  });

  // Se o servidor disser que assinou, o painel tem de gritar.
  it("grita se o servidor declarar que assinou", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo({ assinado: true }));
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("ATENCAO");
    expect(texto).toContain("ASSINOU");
    expect(arvore.root.findByProps({ "data-testid": "sbom-assinado" }).props.children).toBe("SIM");
    arvore.unmount();
  });

  it("grita se o servidor declarar provenance", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo({ provenance: true }));
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    expect(textoDosNos(arvore)).toContain("PROVENANCE");
    arvore.unmount();
  });

  it("envia um corpo vazio -- sem raiz", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    expect(mockAgentFetch).toHaveBeenCalledTimes(1);
    const [caminho, init] = mockAgentFetch.mock.calls[0] as [string, { body: string }];
    expect(caminho).toBe("/api/agent/v1/sbom/generate");
    const corpo = JSON.parse(init.body) as Record<string, unknown>;
    expect(Object.keys(corpo)).toEqual([]);
    arvore.unmount();
  });

  it("mostra uma falha em role=alert", async () => {
    mockAgentFetch.mockRejectedValue(new Error("412 a raiz do projecto nao esta' configurada"));
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    expect(arvore.root.findByProps({ role: "alert" })).toBeDefined();
    expect(textoDosNos(arvore)).toContain("412");
    arvore.unmount();
  });
});
