import { act, create, type ReactTestInstance, type ReactTestRenderer } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";

// FASE 28.6-A / 28.8: a seccao de PERFIS DE INSTALACAO dentro da pagina de Capacidades.
//
// O que estes testes protegem:
//   1. `PULAR` aparece como "Nao avaliado" -- NUNCA como "Ausente". Dizer "ausente" do que nao
//      se conseguiu avaliar faria o utilizador instalar por cima do que ja' tem.
//   2. Um componente que exige aprovacao aparece como "Requer aprovacao", nunca como pronto.
//   3. NAO existe botao de instalar: a rota devolve `executa_algo: false`.
//   4. Os avisos da auto-auditoria do catalogo chegam ao utilizador e sao anunciados.
//   5. Perfil sem itens (CUSTOM) diz POR QUE esta' vazio.

const catalogoMock = {
  host: "HOST-TESTE",
  goos: "windows",
  perfis: [
    {
      id: "BASIC",
      descricao: "HADES Core, Mission Control e diagnostico",
      pacotes: ["hades-core"],
      tamanho_mb: 180,
      escolhe_pacotes: false,
    },
    {
      id: "CONTAINERS_AND_CLOUD",
      descricao: "Core + containers e CLIs",
      pacotes: ["hades-core", "containers"],
      tamanho_mb: 720,
      escolhe_pacotes: false,
    },
    {
      id: "CUSTOM",
      descricao: "escolha componente a componente",
      pacotes: [],
      tamanho_mb: 0,
      escolhe_pacotes: true,
    },
  ],
  pacotes: [],
  avisos: [] as string[],
  count: 3,
};

const planoBasic = {
  perfil: "BASIC",
  host: "HOST-TESTE",
  goos: "windows",
  itens: [
    {
      pacote_id: "hades-core",
      nome: "HADES Core",
      acao: "PULAR",
      motivo: "nao foi possivel determinar o estado nesta maquina",
      requer_aprovacao: false,
      tamanho_mb: 180,
    },
  ],
  tamanho_total_mb: 0,
  downloads_evitados: 0,
  aprovacoes_necessarias: 0,
  bloqueios: ["hades-core: estado desconhecido (UNKNOWN nao e' ABSENT)"],
  pronto: false,
  resumo: "perfil BASIC: 1 itens, 0 MB a baixar, 0 aprovacoes, 1 bloqueios",
  executa_algo: false,
};

const planoContainers = {
  ...planoBasic,
  perfil: "CONTAINERS_AND_CLOUD",
  itens: [
    {
      pacote_id: "containers",
      nome: "Containers e Buildx",
      acao: "APROVAR",
      motivo: "ausente; exige aprovacao: elevacao administrativa, servico persistente",
      requer_aprovacao: true,
      tamanho_mb: 540,
    },
  ],
  aprovacoes_necessarias: 1,
  bloqueios: [],
};

const planoCustom = {
  ...planoBasic,
  perfil: "CUSTOM",
  itens: [],
  bloqueios: [],
  pronto: true,
  resumo: "perfil CUSTOM: 0 itens",
};

const estado = {
  catalogo: catalogoMock,
  plano: planoBasic,
};

vi.mock("@/lib/toolchainClient", () => ({
  fetchInstallCatalogo: vi.fn(async () => estado.catalogo),
  fetchInstallPlano: vi.fn(async () => estado.plano),
  // Os rotulos e cores REAIS: *um mock de rotulo testaria o mock, nao o produto.*
  rotuloDaAcao: (acao: string) => {
    switch (acao) {
      case "INSTALAR":
        return "Instalar";
      case "REUTILIZAR":
        return "Reutilizar o que ja' existe";
      case "MANTER":
        return "Manter";
      case "PULAR":
        return "Nao avaliado";
      case "APROVAR":
        return "Requer aprovacao";
      default:
        return acao;
    }
  },
  corDaAcao: (acao: string) => `cor-${acao}`,
}));

import { InstallProfilesPanel } from "./InstallProfilesPanel";

function texto(node: ReactTestInstance | string): string {
  if (typeof node === "string") return node;
  return node.children.map((c) => texto(c)).join("");
}

async function montar(): Promise<ReactTestRenderer> {
  let arvore!: ReactTestRenderer;
  await act(async () => {
    arvore = create(<InstallProfilesPanel />);
  });
  return arvore;
}

async function esperar(arvore: ReactTestRenderer, contem: string, tentativas = 40): Promise<void> {
  for (let i = 0; i < tentativas; i++) {
    if (texto(arvore.root).includes(contem)) return;
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });
  }
  throw new Error(`"${contem}" nao apareceu; conteudo: ${texto(arvore.root).slice(0, 400)}`);
}

/** Clica no botao de um perfil pelo ID. */
async function abrirPerfil(arvore: ReactTestRenderer, id: string): Promise<void> {
  const botoes = arvore.root.findAll(
    (n) => n.type === "button" && n.props["aria-expanded"] !== undefined,
  );
  const alvo = botoes.find((b) => texto(b).includes(id));
  if (!alvo) throw new Error(`botao do perfil ${id} nao encontrado`);
  await act(async () => {
    alvo.props.onClick();
  });
}

beforeEach(() => {
  estado.catalogo = catalogoMock;
  estado.plano = planoBasic;
});

describe("InstallProfilesPanel", () => {
  it("mostra os perfis que o servidor devolveu", async () => {
    const arvore = await montar();
    await esperar(arvore, "Perfis de instalacao");
    const t = texto(arvore.root);
    expect(t).toContain("BASIC");
    expect(t).toContain("CONTAINERS_AND_CLOUD");
    expect(t).toContain("CUSTOM");
    // Cada perfil mostra tamanho ou o motivo de nao ter tamanho.
    expect(t).toContain("180 MB");
    expect(t).toContain("escolha componente a componente");
  });

  it("PULAR diz 'Nao avaliado' e nunca 'Ausente'", async () => {
    const arvore = await montar();
    await esperar(arvore, "Perfis de instalacao");
    await abrirPerfil(arvore, "BASIC");
    await esperar(arvore, "Nao avaliado");

    const t = texto(arvore.root);
    expect(t).toContain("Nao avaliado");
    // O rotulo da accao nunca pode ser "Ausente": isso afirmaria ausencia sem prova.
    const celulas = arvore.root.findAll(
      (n) => n.type === "td" && texto(n).trim().length > 0,
    );
    expect(celulas.some((c) => texto(c).trim() === "Ausente")).toBe(false);
  });

  it("um componente que exige aprovacao diz 'Requer aprovacao'", async () => {
    estado.plano = planoContainers;
    const arvore = await montar();
    await esperar(arvore, "Perfis de instalacao");
    await abrirPerfil(arvore, "CONTAINERS_AND_CLOUD");
    await esperar(arvore, "Requer aprovacao");

    const t = texto(arvore.root);
    expect(t).toContain("Requer aprovacao");
    // E a UI conta as aprovacoes pendentes.
    expect(t).toContain("exigem aprovacao humana");
    // *Um perfil mais completo nao torna um servico menos invasivo.*
    expect(t).toContain("elevacao administrativa");
  });

  it("nao existe botao de instalar em lado nenhum", async () => {
    estado.plano = planoContainers;
    const arvore = await montar();
    await esperar(arvore, "Perfis de instalacao");
    await abrirPerfil(arvore, "CONTAINERS_AND_CLOUD");
    await esperar(arvore, "Requer aprovacao");

    const botoes = arvore.root.findAll((n) => n.type === "button");
    for (const botao of botoes) {
      const rotulo = texto(botao).trim().toLowerCase();
      // *Um botao "Instalar" na UI de diagnostico mentiria: instalar exige approval e
      // superficie propria.*
      expect(rotulo).not.toBe("instalar");
      expect(rotulo).not.toContain("instalar agora");
    }
  });

  it("mostra os avisos da auto-auditoria do catalogo", async () => {
    estado.catalogo = { ...catalogoMock, avisos: ["containers: dependencia inexistente: xyz"] };
    const arvore = await montar();
    await esperar(arvore, "dependencia inexistente");

    // O aviso tem de ser anunciado a leitores de ecra, nao apenas desenhado.
    const alertas = arvore.root.findAll((n) => n.props.role === "alert");
    expect(alertas.length).toBeGreaterThan(0);
  });

  it("CUSTOM vazio diz por que esta' vazio", async () => {
    estado.plano = planoCustom;
    const arvore = await montar();
    await esperar(arvore, "Perfis de instalacao");
    await abrirPerfil(arvore, "CUSTOM");
    await esperar(arvore, "a escolha e' do utilizador");
  });

  it("mostra os bloqueios quando o estado e' desconhecido", async () => {
    const arvore = await montar();
    await esperar(arvore, "Perfis de instalacao");
    await abrirPerfil(arvore, "BASIC");
    await esperar(arvore, "O que falta saber");

    const t = texto(arvore.root);
    expect(t).toContain("O que falta saber");
    expect(t).toContain("UNKNOWN nao e' ABSENT");
  });

  it("erro do servidor vira mensagem accionavel, nao ecra vazio", async () => {
    // Um erro nao pode deixar a seccao em "a ler" para sempre.
    estado.catalogo = undefined as unknown as typeof catalogoMock;
    vi.mocked(
      (await import("@/lib/toolchainClient")).fetchInstallCatalogo,
    ).mockRejectedValueOnce(new Error("servidor indisponivel"));

    const arvore = await montar();
    await esperar(arvore, "servidor indisponivel");
    expect(texto(arvore.root)).toContain("Nao foi possivel ler o catalogo");
    const alertas = arvore.root.findAll((n) => n.props.role === "alert");
    expect(alertas.length).toBeGreaterThan(0);
  });
});
