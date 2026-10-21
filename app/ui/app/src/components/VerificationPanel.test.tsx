/**
 * FASE 28.8: testes do painel de verificacao.
 *
 * O que estes testes travam:
 *
 *  1. o painel diz que a rota **so' compara checksums** antes de haver resultado;
 *  2. **nao existe botao de assinar nem de construir**;
 *  3. **`sem_assinatura` e `nao_verificadas` aparecem SEPARADOS** -- nunca somados;
 *  4. um artefacto **adulterado reprova** e o painel di-lo;
 *  5. `aprovada: true` **nao** e' apresentado como prova de assinatura;
 *  6. os caminhos de demonstracao sao declarados como tal;
 *  7. os estados de assinatura vem do backend (`AUSENTE`/`PRESENTE`/`INVALIDA`/`NAO_VERIFICADA`).
 *
 * Mocka-se **apenas o `agentFetch`**.
 */

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockAgentFetch = vi.fn();

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: (...args: unknown[]) => mockAgentFetch(...args),
}));

import { VerificationPanel } from "@/components/VerificationPanel";

function resultado(sobrescrever: Record<string, unknown> = {}) {
  return {
    caminho: "/artefactos/hades-windows-amd64.exe",
    tamanho_bytes: 1024,
    integro: true,
    adulterado: false,
    assinatura: "AUSENTE" as const,
    aprovado: true,
    duracao_ms: 3,
    explicacao: "checksum confere; sem assinatura",
    ...sobrescrever,
  };
}

/** Relatorio: tudo integro, tudo inspeccionado, nada assinado. */
function relatorioAprovado() {
  return {
    somente_leitura: true,
    nada_assinado: true,
    verificacao: {
      resultados: [resultado(), resultado({ caminho: "/artefactos/hades-linux-amd64" })],
      total: 2,
      integros: 2,
      adulterados: 0,
      sem_assinatura: 2,
      nao_verificadas: 0,
      aprovada: true,
      explicacao: "todos os checksums conferem",
    },
    aviso_de_fronteira:
      "esta rota apenas COMPARA cada artefacto com o SHA-256 que o cliente enviou; nao constroi, " +
      "nao assina, nao instala e nao procura o build por ID. `aprovada` significa que todos os " +
      "ficheiros continuam iguais ao que o build mediu -- nao significa que exista assinatura.",
  };
}

/** Relatorio: um adulterado. */
function relatorioAdulterado() {
  const base = relatorioAprovado();
  return {
    ...base,
    verificacao: {
      ...base.verificacao,
      resultados: [
        resultado(),
        resultado({
          caminho: "/artefactos/hades-linux-amd64",
          integro: false,
          adulterado: true,
          aprovado: false,
          explicacao: "checksum NAO confere",
        }),
      ],
      integros: 1,
      adulterados: 1,
      sem_assinatura: 1,
      aprovada: false,
      explicacao: "1 artefacto adulterado",
    },
  };
}

/** Relatorio: integro mas sem adaptador de assinatura. */
function relatorioNaoInspeccionado() {
  const base = relatorioAprovado();
  return {
    ...base,
    nada_assinado: false,
    verificacao: {
      ...base.verificacao,
      resultados: [
        resultado({ assinatura: "PRESENTE" as const }),
        resultado({
          caminho: "/artefactos/hades-linux-amd64",
          assinatura: "NAO_VERIFICADA" as const,
        }),
      ],
      sem_assinatura: 1,
      nao_verificadas: 1,
      aprovada: true,
    },
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
    arvore = create(<VerificationPanel />);
  });
  return arvore;
}

function botao(arvore: ReactTestRenderer) {
  return arvore.root.find(
    (no) => no.type === "button" && String(no.props.children).includes("Verificar artefactos"),
  );
}

beforeEach(() => {
  mockAgentFetch.mockReset();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("VerificationPanel", () => {
  it("diz que a rota so' compara checksums, antes de haver resultado", async () => {
    const arvore = await montar();
    const texto = textoDosNos(arvore);
    expect(texto).toContain("apenas compara");
    expect(texto).toContain("Nao constroi, nao assina, nao instala");
    expect(texto).toContain("Aprovada nao significa assinada");
    arvore.unmount();
  });

  // OS CAMINHOS SAO DE DEMONSTRACAO, E O PAINEL DI-LO.
  it("declara que os caminhos sao de demonstracao", async () => {
    const arvore = await montar();
    const texto = textoDosNos(arvore);
    expect(texto).toContain("demonstracao");
    expect(texto).toContain("nao de um build real");
    arvore.unmount();
  });

  // NAO EXISTE BOTAO DE ASSINAR NEM DE CONSTRUIR.
  it("nao oferece accoes de assinar, construir ou instalar", async () => {
    mockAgentFetch.mockResolvedValue(relatorioAprovado());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const botoes = arvore.root.findAll((no) => no.type === "button");
    expect(botoes).toHaveLength(1);
    const rotulo = String(botoes[0].props.children).toLowerCase();
    for (const proibido of ["assinar", "construir", "instalar", "publicar"]) {
      expect(rotulo).not.toContain(proibido);
    }
    expect(rotulo).toContain("verificar");
    arvore.unmount();
  });

  // NAO TEM CAMPO PARA ESCOLHER CAMINHOS.
  it("nao tem campo para escolher caminhos ou checksums", async () => {
    const arvore = await montar();
    const campos = arvore.root.findAll((no) => no.type === "input");
    expect(campos).toHaveLength(0);
    arvore.unmount();
  });

  it("mostra os componentes e os estados de assinatura do backend", async () => {
    mockAgentFetch.mockResolvedValue(relatorioNaoInspeccionado());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("/artefactos/hades-windows-amd64.exe");
    // Os rotulos vem do backend, nao de valores inventados.
    expect(texto).toContain("assinatura presente");
    expect(texto).toContain("nao inspeccionada");
    arvore.unmount();
  });

  // A DISTINCAO QUE O PAINEL NAO PODE PERDER.
  it("separa sem_assinatura de nao_verificadas -- nunca soma", async () => {
    mockAgentFetch.mockResolvedValue(relatorioNaoInspeccionado());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    // Cada um no seu campo.
    expect(arvore.root.findByProps({ "data-testid": "verificacao-sem-assinatura" }).props.children).toBe(1);
    expect(
      arvore.root.findByProps({ "data-testid": "verificacao-nao-verificadas" }).props.children,
    ).toBe(1);
    // E o resumo diz que ha' por inspeccionar -- nao que "todos assinados".
    const resumo = textoDosNos(arvore);
    expect(resumo).toContain("NAO foi/foram inspeccionado");
    arvore.unmount();
  });

  // UM ADULTERADO REPROVA.
  it("um artefacto adulterado reprova e o painel di-lo", async () => {
    mockAgentFetch.mockResolvedValue(relatorioAdulterado());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("ADULTERADO");
    expect(texto).toContain("nao pode ser distribuida");
    expect(texto).toContain("REPROVADA");
    expect(arvore.root.findByProps({ "data-testid": "verificacao-adulterados" }).props.children).toBe(1);
    expect(arvore.root.findByProps({ "data-testid": "verificacao-aprovada" }).props.children).toBe("nao");
    arvore.unmount();
  });

  // APROVADA NAO E' PROVA DE ASSINATURA.
  it("aprovada nao e' apresentada como prova de assinatura", async () => {
    mockAgentFetch.mockResolvedValue(relatorioAprovado());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const prova = arvore.root.findByProps({ "data-testid": "verificacao-prova" });
    const textoProva = String(prova.props.children);
    expect(textoProva).toContain("nada foi assinado");
    expect(textoProva).toContain("nao e' prova de assinatura");
    // E o resumo diz que todos foram inspeccionados -- nao que estao assinados.
    expect(textoDosNos(arvore)).toContain("inspeccionados quanto a assinatura");
    arvore.unmount();
  });

  it("mostra o aviso de fronteira do backend por inteiro", async () => {
    mockAgentFetch.mockResolvedValue(relatorioAprovado());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("apenas COMPARA");
    expect(texto).toContain("nao significa que exista assinatura");
    arvore.unmount();
  });

  it("envia `artefactos` com caminho e sha256 de 64 hex", async () => {
    mockAgentFetch.mockResolvedValue(relatorioAprovado());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const [caminho, init] = mockAgentFetch.mock.calls[0] as [string, { body: string }];
    expect(caminho).toBe("/api/agent/v1/build/verify");
    const corpo = JSON.parse(init.body) as { artefactos: { caminho: string; sha256: string }[] };
    expect(Array.isArray(corpo.artefactos)).toBe(true);
    expect(corpo.artefactos.length).toBeGreaterThan(0);
    for (const artefacto of corpo.artefactos) {
      expect(typeof artefacto.caminho).toBe("string");
      expect(artefacto.sha256).toMatch(/^[a-f0-9]{64}$/);
    }
    // E nao ha' chave para raiz.
    expect(Object.keys(corpo)).toEqual(["artefactos"]);
    arvore.unmount();
  });

  it("um relatorio vazio nao e' erro", async () => {
    const vazio = relatorioAprovado();
    vazio.verificacao = {
      ...vazio.verificacao,
      resultados: [],
      total: 0,
      integros: 0,
      sem_assinatura: 0,
      nao_verificadas: 0,
      aprovada: true,
      explicacao: "nada a verificar",
    };
    mockAgentFetch.mockResolvedValue(vazio);
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    expect(arvore.root.findAll((no) => no.props.role === "alert")).toHaveLength(0);
    expect(textoDosNos(arvore)).toContain("Nenhum artefacto foi verificado");
    arvore.unmount();
  });

  it("mostra uma falha em role=alert", async () => {
    mockAgentFetch.mockRejectedValue(new Error("400 corpo invalido"));
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    expect(arvore.root.findByProps({ role: "alert" })).toBeDefined();
    expect(textoDosNos(arvore)).toContain("400");
    arvore.unmount();
  });
});
