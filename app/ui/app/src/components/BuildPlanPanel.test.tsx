// FASE 28.8: testes do painel Build & Release.
//
// O que estes testes travam:
//
//  1. o painel comeca a pedir o caminho (nao inventa um);
//  2. caminho vazio NAO chama a rota;
//  3. o plano aparece com stacks, alvos e explicacao;
//  4. "pronto" e "bloqueado" vem do BACKEND, nao de calculo local;
//  5. projeto sem alvos tem mensagem propria, nao "erro";
//  6. erro da rota aparece em `role="alert"`;
//  7. os blockers vem no topo, sem abrir cada alvo;
//  8. os motivos explicam cada blocker;
//  9. a fronteira ("so' detecta e sonda") esta no corpo;
// 10. os estados das sondas sao traduzidos para texto humano.

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { BuildPlanPanel } from "@/components/BuildPlanPanel";
import type { PlanoDeRelease } from "@/lib/buildPlanClient";

// O TRANSPORTE e' substituido, nao o cliente. Assim o painel e o `buildPlanClient` reais correm, e
// so' a fronteira de rede e' falsa.
//
//	*Um teste que substitui o cliente testa o mock, nao o produto.*
vi.mock("@/lib/agenticClient", () => ({
  agentFetch: vi.fn(),
}));

import { agentFetch } from "@/lib/agenticClient";

const fetchMock = vi.mocked(agentFetch);

/**
 * Faz o transporte devolver um corpo.
 *
 * `agentFetch<T>` devolve o objecto TIPADO, nao um `Response` -- por isso o mock devolve o corpo
 * directamente. Devolver um `Response` aqui testaria um transporte que o produto nao usa.
 *
 *	*Um mock que imita uma API que nao existe obriga o produto a mentir para o teste passar.*
 */
function responderCom(corpo: unknown): void {
  fetchMock.mockResolvedValue(corpo);
}

function plano(sobrescrever: Partial<PlanoDeRelease> = {}): PlanoDeRelease {
  return {
    somente_leitura: true,
    nada_instalado: true,
    nada_construido: true,
    repo: "C:\\projeto",
    profundidade: 4,
    plano: {
      projeto: {
        raiz: "C:\\projeto",
        stacks: ["go", "node"],
        gestores: ["go-modules"],
        alvos: ["linux", "web"],
        manifests: ["go.mod", "package.json"],
      },
      alvos: [
        {
          alvo: "linux",
          plano: {
            Alvo: "linux",
            Probes: [
              {
                Requisito: { Nome: "go", Comando: "go", Obrigatorio: true, Motivo: "compilar" },
                Estado: "PRESENT_COMPATIBLE_HEALTHY",
              },
              {
                Requisito: {
                  Nome: "docker",
                  Comando: "docker",
                  Obrigatorio: true,
                  Motivo: "empacotar a imagem",
                },
                Estado: "ABSENT",
                Detalhe: "nao encontrado no PATH",
              },
            ],
            Blockers: ["docker"],
            TrabalhoIndependente: ["go install"],
          },
        },
      ],
      alvos_prontos: [],
      alvos_bloqueados: ["linux"],
      sem_alvos: false,
      trabalho_independente: ["go install"],
      avisos: ["projecto Expo com pasta android/ presente"],
      explicacao: "nenhum dos 1 alvo(s) do projecto esta' pronto; 1 item(ns) pode avancar",
    },
    blockers: ["docker"],
    motivos: ["linux: docker -- empacotar a imagem (nao encontrado no PATH)"],
    aviso_de_fronteira:
      "esta rota apenas DETECTA o projecto e SONDA ferramentas; nao instala nada, nao constroi nada",
    ...sobrescrever,
  };
}

describe("BuildPlanPanel", () => {
  let renderer: ReactTestRenderer | null = null;

  beforeEach(() => {
    fetchMock.mockReset();
  });

  afterEach(() => {
    if (renderer) {
      act(() => renderer?.unmount());
      renderer = null;
    }
  });

  async function montar(): Promise<ReactTestRenderer> {
    await act(async () => {
      renderer = create(<BuildPlanPanel />);
    });
    return renderer as ReactTestRenderer;
  }

  it("nao chama a rota quando o caminho esta' vazio", async () => {
    await montar();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("mostra o plano com stacks, alvos e explicacao", async () => {
    responderCom(plano());
    const r = await montar();

    const campo = r.root.findByType("input");
    await act(async () => {
      campo.props.onChange({ target: { value: "C:\\projeto" } });
    });

    const textos = JSON.stringify(r.toJSON());
    expect(textos).toContain("go, node");
    expect(textos).toContain("linux");
    expect(textos).toContain("nenhum dos 1 alvo(s)");
  });

  it("declara a fronteira no corpo: so' detecta e sonda", async () => {
    responderCom(plano());
    const r = await montar();

    await act(async () => {
      r.root.findByType("input").props.onChange({ target: { value: "C:\\projeto" } });
    });

    const textos = JSON.stringify(r.toJSON());
    expect(textos).toContain("apenas");
    expect(textos).toContain("DETECTA");
    expect(textos).toContain("nao instala nada");
  });

  it("mostra os blockers no topo sem abrir cada alvo", async () => {
    responderCom(plano());
    const r = await montar();

    await act(async () => {
      r.root.findByType("input").props.onChange({ target: { value: "C:\\projeto" } });
    });

    const textos = JSON.stringify(r.toJSON());
    expect(textos).toContain("docker");
    // A seccao "O que falta" existe e o item esta' la'.
    const secoes = r.root
      .findAllByType("h3" as never)
      .map((h) => (Array.isArray(h.props.children) ? h.props.children.join("") : h.props.children));
    expect(secoes).toContain("O que falta");
  });

  it("mostra os motivos que explicam cada blocker", async () => {
    responderCom(plano());
    const r = await montar();

    await act(async () => {
      r.root.findByType("input").props.onChange({ target: { value: "C:\\projeto" } });
    });

    const textos = JSON.stringify(r.toJSON());
    expect(textos).toContain("empacotar a imagem");
    expect(textos).toContain("nao encontrado no PATH");
  });

  it("projecto sem alvos tem mensagem propria, nao erro", async () => {
    responderCom(
      plano({
        plano: {
          ...plano().plano,
          sem_alvos: true,
          alvos: [],
          alvos_prontos: [],
          alvos_bloqueados: [],
          explicacao: "o projecto nao declara nenhum alvo",
        },
        blockers: [],
        motivos: [],
      }),
    );
    const r = await montar();

    await act(async () => {
      r.root.findByType("input").props.onChange({ target: { value: "C:\\projeto" } });
    });

    const textos = JSON.stringify(r.toJSON());
    expect(textos).toContain("não é um erro, é um resultado");
    // Nao deve haver alerta: nao e' erro.
    expect(r.root.findAllByProps({ role: "alert" })).toHaveLength(0);
  });

  it("erro da rota aparece em role=alert", async () => {
    fetchMock.mockRejectedValue(new Error("projecto nao reconhecido"));
    const r = await montar();

    await act(async () => {
      r.root.findByType("input").props.onChange({ target: { value: "C:\\projeto" } });
    });

    const alertas = r.root.findAllByProps({ role: "alert" });
    expect(alertas.length).toBeGreaterThan(0);
    expect(JSON.stringify(alertas[0].props.children)).toContain("projecto nao reconhecido");
  });

  it("traduz o estado da sonda para texto humano", async () => {
    responderCom(plano());
    const r = await montar();

    await act(async () => {
      r.root.findByType("input").props.onChange({ target: { value: "C:\\projeto" } });
    });

    const textos = JSON.stringify(r.toJSON());
    // `PRESENT_COMPATIBLE_HEALTHY` -> "pronta"; `ABSENT` -> "não instalada". Os valores crus nao\n    // aparecem. Os valores do mock sao os REAIS de `ToolState`, nao nomes inventados.
    expect(textos).toContain("pronta");
    expect(textos).toContain("não instalada");
    expect(textos).not.toContain('"PRESENT_COMPATIBLE_HEALTHY"');
    expect(textos).not.toContain('"ABSENT"');
  });

  it("mostra o trabalho que pode avancar sem desbloquear nada", async () => {
    responderCom(plano());
    const r = await montar();

    await act(async () => {
      r.root.findByType("input").props.onChange({ target: { value: "C:\\projeto" } });
    });

    expect(JSON.stringify(r.toJSON())).toContain("go install");
  });

  it("mostra os avisos da deteccao", async () => {
    responderCom(plano());
    const r = await montar();

    await act(async () => {
      r.root.findByType("input").props.onChange({ target: { value: "C:\\projeto" } });
    });

    expect(JSON.stringify(r.toJSON())).toContain("android/");
  });
});
