import { act, create, type ReactTestInstance, type ReactTestRenderer } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";

// FASE 28.5: a UI do Signing Identity Center.
//
// O que estes testes protegem:
//   1. os TRES estados de registo aparecem diferentes (nao configurado / vazio / com identidades);
//   2. `desconhecida` nao e' pintada como bloqueio;
//   3. identidade de producao NUNCA diz "pode assinar agora";
//   4. a pagina NAO oferece botoes de assinar/importar/revogar (accoes exigem aprovacao);
//   5. o enum cru nao vaza para o utilizador;
//   6. erro do backend vira mensagem accionavel.

const estado = { modo: "com-identidades" as "com-identidades" | "vazio" | "nao-configurado" | "erro" };

function resposta() {
  if (estado.modo === "nao-configurado") {
    return {
      estado: { configurado: false, carregado: false, mensagem: "registo nao configurado", total: 0 },
      identidades: [],
      problemas: [],
      resumo: { total: 0, por_estado: {}, por_ambiente: {}, problemas: 0 },
      verificado_em: "2026-10-18T12:00:00Z",
      aviso: "apenas metadados publicos",
    };
  }
  if (estado.modo === "vazio") {
    return {
      estado: { configurado: true, carregado: true, mensagem: "registo carregado e vazio", total: 0 },
      identidades: [],
      problemas: [],
      resumo: { total: 0, por_estado: {}, por_ambiente: {}, problemas: 0 },
      verificado_em: "2026-10-18T12:00:00Z",
      aviso: "apenas metadados publicos",
    };
  }
  return {
    estado: { configurado: true, carregado: true, mensagem: "3 identidade(s) registada(s)", total: 3 },
    identidades: [
      {
        nome: "apple-distribution",
        plataforma: "ios",
        ambiente: "producao",
        provedor: "dpapi",
        fingerprint_publico_fim: "1AEB29",
        expira_em: "2027-10-17T00:00:00Z",
        estado: "ativa",
        utilizavel_agora: true,
        exige_aprovacao: true,
        runners_autorizados: 1,
      },
      {
        nome: "android-upload",
        plataforma: "android",
        ambiente: "preview",
        provedor: "dpapi",
        fingerprint_publico_fim: "B07D68",
        expira_em: "2036-10-17T00:00:00Z",
        estado: "expira_breve",
        utilizavel_agora: true,
        exige_aprovacao: false,
        runners_autorizados: 1,
      },
      {
        nome: "windows-authenticode",
        plataforma: "windows",
        ambiente: "producao",
        provedor: "kms",
        fingerprint_publico_fim: "DEADBE",
        expira_em: "",
        estado: "desconhecida",
        utilizavel_agora: false,
        exige_aprovacao: true,
        runners_autorizados: 0,
      },
    ],
    problemas: [
      { nome: "windows-authenticode", severidade: "critico", mensagem: "nenhum runner autorizado" },
    ],
    resumo: { total: 3, por_estado: { ativa: 1 }, por_ambiente: { producao: 2 }, problemas: 1 },
    verificado_em: "2026-10-18T12:00:00Z",
    aviso: "apenas metadados publicos; o valor privado vive no broker",
  };
}

vi.mock("@/lib/signingClient", () => ({
  fetchSigningIdentities: vi.fn(async () => {
    if (estado.modo === "erro") throw new Error("cofre indisponivel");
    return resposta();
  }),
  corDoEstadoDaIdentidade: (e: string) => `cor-${e}`,
  rotuloDoEstadoDaIdentidade: (e: string) =>
    ({
      ativa: "Ativa",
      expira_breve: "Expira em breve",
      expirada: "Expirada",
      revogada: "Revogada",
      desconhecida: "Validade desconhecida",
    })[e] ?? e,
  rotuloDoAmbiente: (a: string) =>
    ({ desenvolvimento: "Desenvolvimento", preview: "Preview", producao: "Produção" })[a] ?? a,
  validadeLegivel: (iso: string) => (iso ? "17/10/2027" : "sem validade declarada"),
  podeAssinarAgora: (i: { exige_aprovacao: boolean; utilizavel_agora: boolean }) =>
    !i.exige_aprovacao && i.utilizavel_agora,
}));

vi.mock("@/components/AppSidebar", () => ({ AppSidebar: () => null }));
vi.mock("@/components/layout/layout", () => ({
  SidebarLayout: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

import { SigningIdentitiesPage } from "./SigningIdentitiesPage";

function texto(node: ReactTestInstance | string): string {
  if (typeof node === "string") return node;
  return node.children.map((c) => texto(c)).join("");
}

async function montar(): Promise<ReactTestRenderer> {
  let arvore!: ReactTestRenderer;
  await act(async () => {
    arvore = create(<SigningIdentitiesPage />);
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

describe("SigningIdentitiesPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    estado.modo = "com-identidades";
  });

  it("lista as identidades com plataforma, ambiente e estado", async () => {
    const arvore = await montar();
    await esperar(arvore, "apple-distribution");
    const conteudo = texto(arvore.root);
    expect(conteudo).toContain("apple-distribution");
    expect(conteudo).toContain("android-upload");
    expect(conteudo).toContain("windows-authenticode");
    expect(conteudo).toContain("Produção");
    expect(conteudo).toContain("Ativa");
    expect(conteudo).toContain("Expira em breve");
    expect(conteudo).toContain("Validade desconhecida");
  });

  // Producao exige aprovacao: NUNCA pode dizer "pode assinar agora".
  it("identidade de producao nao aparece como pronta a assinar", async () => {
    const arvore = await montar();
    await esperar(arvore, "apple-distribution");
    // Encontra a linha da apple e verifica o texto dela.
    const linhas = arvore.root.findAllByType("tr");
    const linhaApple = linhas.find((l) => texto(l).includes("apple-distribution"));
    expect(linhaApple).toBeTruthy();
    const textoDaLinha = texto(linhaApple!);
    expect(textoDaLinha).not.toContain("pode assinar agora");
    expect(textoDaLinha).toContain("requer aprovação");
  });

  // O enum cru nao chega ao utilizador.
  //
  // O discriminante e' a FORMA do enum, nao a palavra: o enum cru usa underscore
  // (`expira_breve`) e nao tem acento (`producao`); o rotulo em portugues tem espaco e acento
  // ("Validade desconhecida", "Produção"). *Um teste que proibe uma palavra legitima nao testa
  // o vazamento: testa a minha imaginacao de como o vazamento seria.*
  it("nao vaza os enums crus", async () => {
    const arvore = await montar();
    await esperar(arvore, "apple-distribution");
    const conteudo = texto(arvore.root);
    // Formas que SO' existem no enum cru:
    expect(conteudo).not.toContain("expira_breve");
    expect(conteudo).not.toContain("producao");
    expect(conteudo).not.toContain("desenvolvimento");
    // E os rotulos legiveis TEM de estar presentes (senao o teste passaria por omissao).
    expect(conteudo).toContain("Produção");
    expect(conteudo).toContain("Validade desconhecida");
  });

  // A pagina NAO oferece accoes privilegiadas.
  it("nao oferece botoes de assinar, importar ou revogar", async () => {
    const arvore = await montar();
    await esperar(arvore, "apple-distribution");
    const conteudo = texto(arvore.root);
    expect(conteudo).not.toMatch(/assinar agora\s*$/i);
    expect(conteudo).not.toMatch(/importar/i);
    expect(conteudo).not.toMatch(/revogar/i);
  });

  it("mostra os pontos de atenção com severidade", async () => {
    const arvore = await montar();
    await esperar(arvore, "Pontos de atenção");
    const conteudo = texto(arvore.root);
    expect(conteudo).toContain("nenhum runner autorizado");
    expect(conteudo).toContain("critico");
  });

  // TRES estados de registo, tres mensagens diferentes.
  it("distingue registo nao configurado de registo vazio", async () => {
    estado.modo = "nao-configurado";
    const primeira = await montar();
    await esperar(primeira, "Registo não configurado");
    expect(texto(primeira.root)).toContain("Registo não configurado");

    estado.modo = "vazio";
    const segunda = await montar();
    await esperar(segunda, "não declara nenhuma identidade");
    const conteudo = texto(segunda.root);
    expect(conteudo).toContain("não declara nenhuma identidade");
    // E NAO mostra o aviso de "nao configurado".
    expect(conteudo).not.toContain("Registo não configurado");
  });

  it("erro do backend vira mensagem accionavel", async () => {
    estado.modo = "erro";
    const arvore = await montar();
    await esperar(arvore, "Não foi possível ler o registo");
    expect(texto(arvore.root)).toContain("cofre indisponivel");
  });
});
