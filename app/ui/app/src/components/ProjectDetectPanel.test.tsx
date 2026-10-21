/**
 * FASE 28.8: testes do painel de deteccao de projecto.
 *
 * O que estes testes travam:
 *
 *  1. o painel diz que a rota **so' le'** e que **os alvos sao sugestao**;
 *  2. **nao tem campo para a raiz** -- a raiz e' do servidor;
 *  3. **nao existe botao de instalar, construir ou assinar**;
 *  4. os valores de stack e gestor vem do BACKEND (`go`, `node`, `go-modules`, `npm`), nao inventados;
 *  5. **`provaDaDeteccao` nao promete toolchain** -- diz que `toolchain.doctor` e' que decide;
 *  6. um alvo sem gestor e' dito, nao escondido;
 *  7. os avisos do detector sao mostrados quando existem.
 *
 * Mocka-se **apenas o `agentFetch`**.
 */

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockAgentFetch = vi.fn();

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: (...args: unknown[]) => mockAgentFetch(...args),
}));

import { ProjectDetectPanel } from "@/components/ProjectDetectPanel";

/** Relatorio de exemplo com os valores REAIS do backend. */
function relatorioDeExemplo(sobrescrever: Record<string, unknown> = {}) {
  return {
    somente_leitura: true,
    detectado: {
      raiz: "/projecto-hades",
      stacks: ["go", "node"],
      gestores: ["go-modules", "npm"],
      alvos: ["windows", "linux"],
      manifests: ["go.mod", "package.json"],
      avisos: [],
      ...sobrescrever,
    },
    aviso_de_fronteira:
      "esta rota apenas LE' nomes de ficheiros e o conteudo dos manifests para classificar a stack " +
      "e listar alvos; nao escreve, nao instala, nao constroi e nao assina. A raiz vem da variavel " +
      "de ambiente `HADES_RAIZ_PROJECTO`, nunca do pedido. Os alvos listados sao uma SUGESTAO " +
      "derivada dos ficheiros presentes -- nao significam que a toolchain esteja instalada: " +
      "consulte `toolchain.doctor` antes de planear um build.",
  };
}

/** Relatorio de projecto Expo-only, sem gestores. */
function relatorioSemGestores() {
  return relatorioDeExemplo({ gestores: [], stacks: ["expo"], alvos: ["android", "ios"] });
}

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
    arvore = create(<ProjectDetectPanel />);
  });
  return arvore;
}

function botao(arvore: ReactTestRenderer) {
  return arvore.root.find(
    (no) => no.type === "button" && String(no.props.children).includes("Detectar projecto"),
  );
}

beforeEach(() => {
  mockAgentFetch.mockReset();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("ProjectDetectPanel", () => {
  it("diz que a rota so' le' e que os alvos sao sugestao", async () => {
    const arvore = await montar();
    const texto = textoDosNos(arvore);
    expect(texto).toContain("le' os nomes dos ficheiros");
    expect(texto).toContain("Nao escreve, nao instala e nao constroi");
    expect(texto).toContain("sugestao");
    expect(texto).toContain("Nao significam que a toolchain esteja instalada");
    arvore.unmount();
  });

  // NAO TEM CAMPO PARA A RAIZ.
  it("nao tem campo para a raiz do projecto", async () => {
    const arvore = await montar();
    expect(arvore.root.findAll((no) => no.type === "input")).toHaveLength(0);
    const texto = textoDosNos(arvore);
    expect(texto).toContain("nunca deste painel");
    arvore.unmount();
  });

  // NAO EXISTE BOTAO DE INSTALAR, CONSTRUIR OU ASSINAR.
  it("nao oferece accoes de instalar, construir ou assinar", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const botoes = arvore.root.findAll((no) => no.type === "button");
    expect(botoes).toHaveLength(1);
    const rotulo = String(botoes[0].props.children).toLowerCase();
    for (const proibido of ["instalar", "construir", "assinar", "publicar", "build"]) {
      expect(rotulo).not.toContain(proibido);
    }
    expect(rotulo).toContain("detectar");
    arvore.unmount();
  });

  // OS VALORES SAO OS DO BACKEND.
  it("mostra as stacks e gestores com os valores REAIS do backend", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    // `go` -> "Go", `node` -> "Node"
    expect(texto).toContain("Go");
    expect(texto).toContain("Node");
    // `go-modules` -> "Go modules", `npm` -> "npm"
    expect(texto).toContain("Go modules");
    expect(texto).toContain("npm");
    expect(arvore.root.findByProps({ "data-testid": "deteccao-stacks" }).props.children).toBe(2);
    expect(arvore.root.findByProps({ "data-testid": "deteccao-gestores" }).props.children).toBe(2);
    arvore.unmount();
  });

  it("traduz a stack Expo e o gestor pub", async () => {
    mockAgentFetch.mockResolvedValue(
      relatorioDeExemplo({ stacks: ["expo", "flutter"], gestores: ["pub"] }),
    );
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("Expo / React Native");
    expect(texto).toContain("Flutter / Dart");
    expect(texto).toContain("pub");
    arvore.unmount();
  });

  // A PROVA NAO PROMETE TOOLCHAIN.
  it("a prova diz que toolchain.doctor e' que decide", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const prova = arvore.root.findByProps({ "data-testid": "deteccao-prova" });
    const textoProva = String(prova.props.children);
    expect(textoProva).toContain("NAO prova que a toolchain esteja instalada");
    expect(textoProva).toContain("toolchain.doctor");
    arvore.unmount();
  });

  // UM ALVO SEM GESTOR E' DITO.
  it("avisa quando ha' alvos mas nenhum gestor de pacotes", async () => {
    mockAgentFetch.mockResolvedValue(relatorioSemGestores());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const prova = arvore.root.findByProps({ "data-testid": "deteccao-prova" });
    const textoProva = String(prova.props.children);
    expect(textoProva).toContain("NENHUM gestor de pacotes reconhecido");
    // E o resumo conta os alvos, nao promete build.
    expect(textoDosNos(arvore)).toContain("2 alvos");
    arvore.unmount();
  });

  it("mostra os avisos do detector quando existem", async () => {
    mockAgentFetch.mockResolvedValue(
      relatorioDeExemplo({ avisos: ["monorepo: varios go.mod encontrados"] }),
    );
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const avisos = arvore.root.findByProps({ "data-testid": "deteccao-avisos" });
    expect(avisos).toBeDefined();
    expect(textoDosNos(arvore)).toContain("varios go.mod encontrados");
    arvore.unmount();
  });

  it("mostra os manifests lidos", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("go.mod");
    expect(texto).toContain("package.json");
    arvore.unmount();
  });

  it("mostra o aviso de fronteira do backend por inteiro", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const aviso = arvore.root.findByProps({ "data-testid": "deteccao-aviso-fronteira" });
    const textoAviso = String(aviso.props.children);
    expect(textoAviso).toContain("apenas LE'");
    expect(textoAviso).toContain("SUGESTAO");
    expect(textoAviso).toContain("toolchain.doctor");
    expect(textoAviso).toContain("nunca do pedido");
    arvore.unmount();
  });

  it("envia um corpo vazio -- sem raiz", async () => {
    mockAgentFetch.mockResolvedValue(relatorioDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const [caminho, init] = mockAgentFetch.mock.calls[0] as [string, { body: string }];
    expect(caminho).toBe("/api/agent/v1/project/detect");
    expect(JSON.parse(init.body)).toEqual({});
    arvore.unmount();
  });

  it("mostra uma falha em role=alert", async () => {
    mockAgentFetch.mockRejectedValue(
      new Error("409 falha a detectar o projecto: nenhuma stack reconhecida"),
    );
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    expect(arvore.root.findByProps({ role: "alert" })).toBeDefined();
    expect(textoDosNos(arvore)).toContain("nenhuma stack reconhecida");
    arvore.unmount();
  });
});
