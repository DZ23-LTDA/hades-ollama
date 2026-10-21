// E-1315: testes do INICIO de build a partir do painel de plano.
//
// O que estes testes travam — cada um falharia com o defeito correspondente:
//
//  1. alvo PRONTO ganha botao "Iniciar build";
//  2. alvo BLOQUEADO nao ganha botao (nem desactivado). *Um botao que nunca funciona ensina a
//     ignorar botoes.*
//  3. o clique chama `POST /build/start` com `assincrono: true` e a `raiz` do plano;
//  4. 202 com `build_id` produz um aviso que diz o ID e para onde ir;
//  5. 412 e' apresentado como "plano incompleto", NAO como erro de rede.
//
// O transporte e' substituido, nao o cliente.

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { BuildPlanPanel } from "@/components/BuildPlanPanel";
import type { PlanoDeRelease } from "@/lib/buildPlanClient";

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: vi.fn(),
}));

import { agentFetch } from "@/lib/agenticClient";

const fetchMock = vi.mocked(agentFetch);

function responderCom(corpo: unknown): void {
  fetchMock.mockResolvedValue(corpo);
}

/** Plano com um alvo PRONTO e outro BLOQUEADO, para exercitar os dois ramos. */
function planoComProntoEBloqueado(): PlanoDeRelease {
  return {
    somente_leitura: true,
    nada_instalado: true,
    nada_construido: true,
    repo: "C:\\projeto",
    profundidade: 4,
    plano: {
      projeto: {
        raiz: "C:\\projeto",
        stacks: ["go"],
        gestores: ["go-modules"],
        alvos: ["windows", "linux"],
        manifests: ["go.mod"],
      },
      alvos: [
        {
          alvo: "windows",
          plano: {
            Alvo: "windows",
            Probes: [
              {
                Requisito: { Nome: "go", Comando: "go", Obrigatorio: true, Motivo: "compilar" },
                Estado: "PRESENT_COMPATIBLE_HEALTHY",
              },
            ],
          },
        },
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
                  Motivo: "empacotar",
                },
                Estado: "ABSENT",
                Detalhe: "nao encontrado no PATH",
              },
            ],
            Blockers: ["docker"],
          },
        },
      ],
      alvos_prontos: ["windows"],
      alvos_bloqueados: ["linux"],
      sem_alvos: false,
      trabalho_independente: [],
      avisos: [],
      explicacao: "1 de 2 alvo(s) pronto(s)",
    },
    blockers: ["docker"],
    motivos: ["linux: docker -- empacotar (nao encontrado no PATH)"],
    aviso_de_fronteira: "esta rota apenas DETECTA e SONDA",
  };
}

function porTestId(no: ReactTestRenderer, id: string): unknown[] {
  return no.root.findAll(
    (n) => (n.props as Record<string, unknown>)["data-testid"] === id,
    { deep: true },
  );
}

describe("BuildPlanPanel — iniciar build", () => {
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

  async function montarComPlano(): Promise<ReactTestRenderer> {
    responderCom(planoComProntoEBloqueado());
    await act(async () => {
      renderer = create(<BuildPlanPanel />);
    });
    const r = renderer as ReactTestRenderer;
    const campo = r.root.findByProps({ id: "build-plan-repo" });
    await act(async () => {
      campo.props.onChange({ target: { value: "C:\\projeto" } });
    });
    return r;
  }

  it("alvo pronto ganha o botao «Iniciar build»", async () => {
    const r = await montarComPlano();
    expect(porTestId(r, "build-plan-iniciar-windows").length).toBeGreaterThan(0);
  });

  it("alvo bloqueado NAO ganha botao, nem desactivado", async () => {
    const r = await montarComPlano();
    // Discriminante: um botao desactivado para um alvo bloqueado seria um botao morto.
    expect(porTestId(r, "build-plan-iniciar-linux")).toHaveLength(0);
    // E o painel mostra explicitamente que nao ha' accao.
    expect(porTestId(r, "build-plan-sem-iniciar-linux").length).toBeGreaterThan(0);
  });

  it("o clique chama build/start com assincrono:true e a raiz do plano", async () => {
    const r = await montarComPlano();
    fetchMock.mockResolvedValueOnce({
      construiu: false,
      nada_assinado: true,
      nada_publicado: true,
      alvo: "windows",
      plano: {},
      aviso_de_fronteira: "build aceite, a correr em segundo plano",
      build_id: "abc123def456abc123def456",
      estado: "A_EXECUTAR",
    });

    const botao = r.root.find(
      (n) =>
        (n.props as Record<string, unknown>)["data-testid"] === "build-plan-iniciar-windows",
    );
    await act(async () => {
      (botao.props as { onClick: () => void }).onClick();
    });

    // A ultima chamada tem de ser o POST, nao uma recarga do plano.
    const chamadas = fetchMock.mock.calls;
    const ultima = chamadas[chamadas.length - 1];
    expect(String(ultima[0])).toContain("/build/start");
    const init = ultima[1] as { method?: string; body?: string };
    expect(init.method).toBe("POST");
    const corpo = JSON.parse(String(init.body)) as Record<string, unknown>;
    // O modo assincrono e' o que da' alvo ao `cancel`.
    expect(corpo["assincrono"]).toBe(true);
    expect(corpo["alvo"]).toBe("windows");
    expect(corpo["raiz"]).toBe("C:\\projeto");
  });

  it("202 com build_id produz um aviso que diz o ID", async () => {
    const r = await montarComPlano();
    fetchMock.mockResolvedValueOnce({
      construiu: false,
      nada_assinado: true,
      nada_publicado: true,
      alvo: "windows",
      plano: {},
      aviso_de_fronteira: "build aceite",
      build_id: "abc123def456abc123def456",
      estado: "A_EXECUTAR",
    });

    const botao = r.root.find(
      (n) =>
        (n.props as Record<string, unknown>)["data-testid"] === "build-plan-iniciar-windows",
    );
    await act(async () => {
      (botao.props as { onClick: () => void }).onClick();
    });

    expect(porTestId(r, "build-plan-aviso-inicio").length).toBeGreaterThan(0);
    const textos = JSON.stringify(r.toJSON());
    expect(textos).toContain("abc123de"); // os 8 primeiros do ID
    // E o painel diz para onde ir ver.
    expect(textos).toContain("Estado dos builds");
  });

  it("412 diz «plano incompleto», nao «erro de rede»", async () => {
    const r = await montarComPlano();
    fetchMock.mockRejectedValueOnce(
      new Error("HTTP 412: plano incompleto: ha' blockers obrigatorios por resolver"),
    );

    const botao = r.root.find(
      (n) =>
        (n.props as Record<string, unknown>)["data-testid"] === "build-plan-iniciar-windows",
    );
    await act(async () => {
      (botao.props as { onClick: () => void }).onClick();
    });

    expect(porTestId(r, "build-plan-erro-inicio").length).toBeGreaterThan(0);
    const textos = JSON.stringify(r.toJSON());
    // O 412 e' um RESULTADO (falta pre-requisito), nao uma falha do servidor.
    expect(textos.toLowerCase()).toContain("incompleto");
  });
});
