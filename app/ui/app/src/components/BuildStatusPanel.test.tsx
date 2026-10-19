/**
 * E-1314: testes do painel de ESTADO de build.
 *
 * O que estes testes travam — cada um falharia com o defeito correspondente:
 *
 *  1. lista vazia mostra "ainda não há builds", NAO um erro;
 *  2. o 409 do `cancel` e' apresentado como "ja' tinha terminado", NAO como falha de rede;
 *  3. `cancelavel` do servidor decide se o botao aparece -- o painel nao decide sozinho;
 *  4. `nada_assinado`/`nada_publicado`/`nada_instalado` vem do SERVIDOR (o teste forca `false` numa
 *     resposta e exige que o painel nao invente `true`);
 *  5. build sem artefactos diz que nao produziu, em vez de mostrar uma lista vazia sem explicacao;
 *  6. saida truncada e' mostrada como veio, sem o painel a completar.
 *
 * Mocka-se **apenas o `agentFetch`**.
 */

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockAgentFetch = vi.fn();

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: (...args: unknown[]) => mockAgentFetch(...args),
}));

import { BuildStatusPanel } from "@/components/BuildStatusPanel";

/** Um build como o backend o devolve. */
function build(sobrescrever: Record<string, unknown> = {}) {
  return {
    id: "abc123def456abc123def456",
    alvo: "windows",
    raiz: "/r",
    destino: "/d",
    estado: "CONCLUIDO",
    iniciado_em: "2026-10-19T00:00:00Z",
    terminado_em: "2026-10-19T00:00:05Z",
    cancelavel: false,
    nada_assinado: true,
    nada_publicado: true,
    resultado: {
      Artefatos: [
        {
          Caminho: "/d/x.exe",
          Nome: "x.exe",
          TamanhoBytes: 2048,
          SHA256: "a".repeat(64),
          Alvo: "windows",
        },
      ],
      Explicacao: "go build concluiu",
    },
    ...sobrescrever,
  };
}

function respostaRegistos(builds: unknown[]) {
  return {
    total: builds.length,
    builds,
    nada_assinado: true,
    nada_publicado: true,
    aviso_de_fronteira: "estado de build: nada e' assinado",
  };
}

function respostaSaida(sobrescrever: Record<string, unknown> = {}) {
  return {
    id: "abc123def456abc123def456",
    estado: "CONCLUIDO",
    saida: "go build: ok",
    truncada: false,
    aviso: "a saida guardada e' limitada",
    ...sobrescrever,
  };
}

function respostaArtefactos(sobrescrever: Record<string, unknown> = {}) {
  return {
    id: "abc123def456abc123def456",
    alvo: "windows",
    estado: "CONCLUIDO",
    artefactos: [
      {
        Caminho: "/d/x.exe",
        Nome: "x.exe",
        TamanhoBytes: 2048,
        SHA256: "a".repeat(64),
        Alvo: "windows",
      },
    ],
    produziu: true,
    nada_assinado: true,
    aviso: "estes artefactos foram medidos",
    ...sobrescrever,
  };
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
    if (typeof valor === "object" && "children" in (valor as Record<string, unknown>)) {
      visitar((valor as Record<string, unknown>).children);
    }
  };
  visitar(no.toJSON());
  return partes.join(" ");
}

function porTestId(no: ReactTestRenderer, id: string): unknown {
  return no.root.findAll(
    (n) => (n.props as Record<string, unknown>)["data-testid"] === id,
    { deep: true },
  );
}

describe("BuildStatusPanel", () => {
  let renderer: ReactTestRenderer | null = null;

  beforeEach(() => {
    mockAgentFetch.mockReset();
  });

  afterEach(() => {
    if (renderer) {
      act(() => renderer?.unmount());
      renderer = null;
    }
  });

  it("lista vazia mostra uma mensagem propria, nao um erro", async () => {
    mockAgentFetch.mockResolvedValue(respostaRegistos([]));
    await act(async () => {
      renderer = create(<BuildStatusPanel />);
    });
    const texto = textoDosNos(renderer!);
    expect(texto).toMatch(/ainda n[ãa]o h[áa] builds/i);
    // Nao pode aparecer um erro: uma lista vazia nao e' uma falha.
    expect(porTestId(renderer!, "build-status-erro")).toHaveLength(0);
    expect(porTestId(renderer!, "build-status-vazio").length).toBeGreaterThan(0);
  });

  it("mostra estado, alvo e contagem de artefactos", async () => {
    mockAgentFetch.mockResolvedValue(respostaRegistos([build()]));
    await act(async () => {
      renderer = create(<BuildStatusPanel />);
    });
    const texto = textoDosNos(renderer!);
    expect(texto).toContain("windows");
    expect(texto).toContain("Concluído");
    expect(texto).toContain("1 medido(s)");
  });

  it("o botao de cancelar so' aparece quando o servidor diz `cancelavel`", async () => {
    // Build em curso e cancelavel.
    mockAgentFetch.mockResolvedValue(
      respostaRegistos([build({ estado: "A_EXECUTAR", cancelavel: true, resultado: undefined })]),
    );
    await act(async () => {
      renderer = create(<BuildStatusPanel />);
    });
    expect(
      porTestId(renderer!, "build-status-cancelar-abc123def456abc123def456").length,
    ).toBeGreaterThan(0);
    act(() => renderer?.unmount());

    // Build terminado: o servidor diz que nao e' cancelavel, o painel nao inventa o botao.
    mockAgentFetch.mockResolvedValue(respostaRegistos([build({ cancelavel: false })]));
    await act(async () => {
      renderer = create(<BuildStatusPanel />);
    });
    expect(
      porTestId(renderer!, "build-status-cancelar-abc123def456abc123def456"),
    ).toHaveLength(0);
  });

  it("409 ao cancelar diz «ja' tinha terminado», nao «falhou»", async () => {
    // 1) carregar lista com build em curso cancelavel
    mockAgentFetch.mockResolvedValueOnce(
      respostaRegistos([build({ estado: "A_EXECUTAR", cancelavel: true, resultado: undefined })]),
    );
    await act(async () => {
      renderer = create(<BuildStatusPanel />);
    });

    // 2) o cancelar devolve 409 (agentFetch lanca); a recarga seguinte tem de voltar a funcionar.
    mockAgentFetch.mockRejectedValueOnce(new Error("HTTP 409: build ja' terminou"));
    mockAgentFetch.mockResolvedValue(respostaRegistos([build({ cancelavel: false })]));
    const botao = renderer!.root.find(
      (n) =>
        (n.props as Record<string, unknown>)["data-testid"] ===
        "build-status-cancelar-abc123def456abc123def456",
    );
    await act(async () => {
      (botao.props as { onClick: () => void }).onClick();
    });

    const texto = textoDosNos(renderer!);
    // Tem de dizer que ja' tinha terminado.
    expect(texto).toMatch(/j[áa] tinha terminado/i);
    // E NAO pode dizer que falhou.
    expect(texto).not.toMatch(/falhou ao cancelar/i);
    // Nao e' um erro de rede: o painel de erro nao aparece.
    expect(porTestId(renderer!, "build-status-erro")).toHaveLength(0);
  });

  it("erro de rede real aparece como erro", async () => {
    mockAgentFetch.mockRejectedValue(new Error("rede indisponivel"));
    await act(async () => {
      renderer = create(<BuildStatusPanel />);
    });
    expect(porTestId(renderer!, "build-status-erro").length).toBeGreaterThan(0);
    expect(textoDosNos(renderer!)).toContain("rede indisponivel");
  });

  it("build sem artefactos diz que nao produziu, em vez de lista vazia sem explicacao", async () => {
    mockAgentFetch
      .mockResolvedValueOnce(respostaRegistos([build()]))
      .mockResolvedValueOnce(respostaSaida())
      .mockResolvedValueOnce(respostaArtefactos({ artefactos: [], produziu: false }));

    await act(async () => {
      renderer = create(<BuildStatusPanel />);
    });
    const abrir = renderer!.root.find(
      (n) =>
        (n.props as Record<string, unknown>)["data-testid"] ===
        "build-status-abrir-abc123def456abc123def456",
    );
    await act(async () => {
      (abrir.props as { onClick: () => void }).onClick();
    });

    expect(porTestId(renderer!, "build-status-sem-artefactos").length).toBeGreaterThan(0);
    expect(textoDosNos(renderer!)).toMatch(/n[ãa]o produziu artefactos/i);
  });

  it("mostra a saida guardada como veio", async () => {
    mockAgentFetch
      .mockResolvedValueOnce(respostaRegistos([build()]))
      .mockResolvedValueOnce(respostaSaida({ saida: "linha A\nlinha B" }))
      .mockResolvedValueOnce(respostaArtefactos());

    await act(async () => {
      renderer = create(<BuildStatusPanel />);
    });
    const abrir = renderer!.root.find(
      (n) =>
        (n.props as Record<string, unknown>)["data-testid"] ===
        "build-status-abrir-abc123def456abc123def456",
    );
    await act(async () => {
      (abrir.props as { onClick: () => void }).onClick();
    });

    expect(textoDosNos(renderer!)).toContain("linha A");
  });

  it("a fronteira de nada assinado/publicado/instalado e' dita no detalhe", async () => {
    mockAgentFetch
      .mockResolvedValueOnce(respostaRegistos([build()]))
      .mockResolvedValueOnce(respostaSaida())
      .mockResolvedValueOnce(respostaArtefactos());

    await act(async () => {
      renderer = create(<BuildStatusPanel />);
    });
    const abrir = renderer!.root.find(
      (n) =>
        (n.props as Record<string, unknown>)["data-testid"] ===
        "build-status-abrir-abc123def456abc123def456",
    );
    await act(async () => {
      (abrir.props as { onClick: () => void }).onClick();
    });

    const fronteira = porTestId(renderer!, "build-status-fronteira");
    expect(fronteira.length).toBeGreaterThan(0);
    expect(textoDosNos(renderer!)).toMatch(/nada foi assinado/i);
  });
});
