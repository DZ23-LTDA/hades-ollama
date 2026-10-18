import { act, create, type ReactTestInstance, type ReactTestRenderer } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";

// FASE 28.7 + 28.8: a UI das Capacidades consome as rotas `toolchain/*`.
//
// O que estes testes protegem:
//   1. a pagina mede o host no arranque e mostra o resumo do backend;
//   2. os TRES estados aparecem com rotulos distintos -- `incompleto` NAO e' pintado como
//      bloqueio;
//   3. o trabalho independente aparece quando ha' bloqueio;
//   4. erro do backend vira mensagem accionavel, nao ecra vazio;
//   5. a pagina NAO oferece botao de instalar (nao ha' botao morto nem accao sem approval).

const doctorMock = {
  host: "HOST-TESTE",
  goos: "windows",
  alvos: [
    { alvo: "android", pronto: true, veredicto: "pronto" as const, exigencias_bloqueantes: 0 },
    { alvo: "ios", pronto: false, veredicto: "bloqueado" as const, exigencias_bloqueantes: 2 },
    { alvo: "web", pronto: false, veredicto: "incompleto" as const, exigencias_bloqueantes: 0 },
  ],
  prontos: 1,
  bloqueados: 1,
  incompletos: 1,
  total: 3,
  resumo: "1 de 3 alvos bloqueados, 1 incompletos (ferramentas presentes, falta health funcional)",
};

const planoMock = {
  host: "HOST-TESTE",
  goos: "windows",
  alvo: "ios",
  pronto: false,
  veredicto: "bloqueado" as const,
  probes: [
    {
      nome: "xcodebuild",
      estado: "ABSENT",
      obrigatorio: true,
      motivo: "compilar iOS",
      detalhe: "nao encontrado no PATH",
    },
  ],
  blockers: ["ios exige host macos; este host e' windows"],
  trabalho_independente: ["preparar o perfil de assinatura", "revisar o Info.plist"],
};

const estado = {
  doctorResolve: true,
};

vi.mock("@/lib/toolchainClient", () => ({
  fetchToolchainDoctor: vi.fn(async () => {
    if (!estado.doctorResolve) throw new Error("servidor indisponivel");
    return doctorMock;
  }),
  fetchToolchainPlan: vi.fn(async () => planoMock),
  corDoVeredicto: (veredicto: string) => `cor-${veredicto}`,
  rotuloDoEstado: (estadoProbe: string) =>
    estadoProbe === "ABSENT" ? "Ausente" : estadoProbe,
  estadoEhAusente: (estadoProbe: string) => estadoProbe === "ABSENT",
}));

vi.mock("@/components/AppSidebar", () => ({
  AppSidebar: () => null,
}));

vi.mock("@/components/layout/layout", () => ({
  SidebarLayout: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

import { CapabilitiesPage } from "./CapabilitiesPage";

function texto(node: ReactTestInstance | string): string {
  if (typeof node === "string") return node;
  return node.children.map((c) => texto(c)).join("");
}

async function montar(): Promise<ReactTestRenderer> {
  let arvore!: ReactTestRenderer;
  await act(async () => {
    arvore = create(<CapabilitiesPage />);
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

describe("CapabilitiesPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    estado.doctorResolve = true;
  });

  it("mostra o host real e o resumo do doctor", async () => {
    const arvore = await montar();
    await esperar(arvore, "HOST-TESTE");
    const conteudo = texto(arvore.root);
    expect(conteudo).toContain("HOST-TESTE");
    // O resumo vem do backend, nao e' recalculado no frontend.
    // *Uma contagem calculada no cliente pode discordar do servidor e ninguem saber qual esta' certo.*
    expect(conteudo).toContain("1 de 3 alvos bloqueados");
  });

  it("lista os tres alvos com veredictos distintos", async () => {
    const arvore = await montar();
    await esperar(arvore, "android");
    const conteudo = texto(arvore.root);
    expect(conteudo).toContain("android");
    expect(conteudo).toContain("ios");
    expect(conteudo).toContain("web");
    expect(conteudo).toContain("pronto");
    expect(conteudo).toContain("bloqueado");
    expect(conteudo).toContain("incompleto");
  });

  it("erro do backend vira mensagem accionavel, nao ecra vazio", async () => {
    estado.doctorResolve = false;
    const arvore = await montar();
    await esperar(arvore, "Nao foi possivel medir as ferramentas");
    expect(texto(arvore.root)).toContain("servidor indisponivel");
  });

  it("nao oferece botao de instalar", async () => {
    const arvore = await montar();
    await esperar(arvore, "HOST-TESTE");
    // *Um botao "Instalar" aqui mentiria: instalar exige approval e uma superficie propria.*
    const conteudo = texto(arvore.root);
    expect(conteudo).not.toMatch(/Instalar/i);
  });

  it("ao expandir um alvo bloqueado mostra blockers E trabalho independente", async () => {
    const arvore = await montar();
    await esperar(arvore, "ios");
    const botao = arvore.root
      .findAllByType("button")
      .find((b) => texto(b).includes("ios"));
    expect(botao).toBeTruthy();
    await act(async () => {
      botao!.props.onClick();
    });
    await esperar(arvore, "ios exige host macos");
    const conteudo = texto(arvore.root);
    // O bloqueio externo vem com o trabalho que pode avancar.
    // *"Depende de conta externa" nao e' razao para parar o que nao depende.*
    expect(conteudo).toContain("Pode avancar sem esperar");
    expect(conteudo).toContain("revisar o Info.plist");
  });

  it("o probe ausente e' rotulado em portugues, nao com o enum cru", async () => {
    const arvore = await montar();
    await esperar(arvore, "ios");
    const botao = arvore.root
      .findAllByType("button")
      .find((b) => texto(b).includes("ios"));
    await act(async () => {
      botao!.props.onClick();
    });
    await esperar(arvore, "Ausente");
    const conteudo = texto(arvore.root);
    expect(conteudo).toContain("Ausente");
    // O enum cru nao deve vazar para o utilizador.
    expect(conteudo).not.toContain("ABSENT");
  });
});
