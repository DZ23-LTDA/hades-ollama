/**
 * FASE 28.7/28.8: testes da seccao "Verificar um artefacto" do `BuildPlanPanel`.
 *
 * O que estes testes travam:
 *
 *  1. a seccao existe no painel, com o aviso de que so' MEDE e COMPARA;
 *  2. "Medir" chama a rota `/artifact/checksum` e mostra o SHA-256 devolvido;
 *  3. "Verificar" chama `/artifact/verify` com o caminho E a referencia;
 *  4. um artefacto ADULTERADO aparece como ADULTERADO -- nao como "íntegro";
 *  5. uma assinatura `NAO_VERIFICADA` e' mostrada como tal, nao como "não assinada";
 *  6. sem referencia, "Verificar" esta' desactivado -- *sem referencia isto seria uma medicao*;
 *  7. o aviso de fronteira vem do BACKEND e e' mostrado;
 *  8. uma falha do backend aparece como `role="alert"`.
 *
 * MOCKA-SE APENAS O `agentFetch`, e deixam-se os DOIS clientes reais correrem.
 *
 *	*Um teste que substitui o cliente testa o mock, nao o produto.*
 *
 * A E-1248 tentou `vi.importActual` + spread e o bundler transpilou para `__spreadValues`, que nao
 * existe no ambiente de teste -- a suite nem arrancava. Mockar a fronteira HTTP e' mais estreito E
 * mais fiel: o URL que o `artifactClient` constroi e' verificado a serio.
 */

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";

/** URLs pedidos, na ordem. */
const pedidos: string[] = [];

/** Resposta a devolver para um URL. Configurada por cada teste. */
let responder: (url: string) => unknown = () => {
  throw new Error("sem resposta configurada");
};

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: async (url: string) => {
    pedidos.push(url);
    // O plano de build nao interessa a estes testes: `null` deixa o painel no estado inicial.
    if (url.includes("/build/plan")) return null;
    return responder(url);
  },
}));

import { BuildPlanPanel } from "@/components/BuildPlanPanel";

/** SHA-256 de exemplo, sintaticamente valido. */
const SHA = "fcef9d9d4303a75b0b9c658f155c9ae6a7ccb84912b4face2268dc51f8161df6";

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
    arvore = create(<BuildPlanPanel />);
  });
  return arvore;
}

async function escrever(arvore: ReactTestRenderer, id: string, valor: string): Promise<void> {
  await act(async () => {
    arvore.root.findByProps({ id }).props.onChange({ target: { value: valor } });
  });
}

function botao(arvore: ReactTestRenderer, texto: string) {
  return arvore.root.find(
    (no) => no.type === "button" && String(no.props.children).includes(texto),
  );
}

function medicao(sha: string) {
  return {
    somente_leitura: true,
    nada_assinado: true,
    caminho: "C:\\artefacto.exe",
    tamanho_bytes: 41111716,
    sha256: sha,
    aviso_de_fronteira: "apenas MEDE e COMPARA; nao constroi",
  };
}

function verificacao(opcoes: {
  medido: string;
  esperado: string;
  assinatura: string;
  aprovado: boolean;
  adulterado: boolean;
}) {
  return {
    somente_leitura: true,
    nada_assinado: false,
    aviso_de_fronteira: "estas rotas apenas MEDEM e COMPARAM",
    resultado: {
      caminho: "C:\\artefacto.exe",
      tamanho_bytes: 41111716,
      sha256_medido: opcoes.medido,
      sha256_esperado: opcoes.esperado,
      integro: !opcoes.adulterado,
      adulterado: opcoes.adulterado,
      assinatura: opcoes.assinatura,
      aprovado: opcoes.aprovado,
      duracao_ms: 12,
      explicacao: opcoes.adulterado ? "ADULTERADO" : "integro",
    },
  };
}

describe("BuildPlanPanel — verificar um artefacto", () => {
  beforeEach(() => {
    pedidos.length = 0;
    responder = () => {
      throw new Error("sem resposta configurada");
    };
  });

  it("declara no ecra que apenas mede e compara, antes de qualquer pedido", async () => {
    const arvore = await montar();
    const texto = textoDosNos(arvore);
    expect(texto).toContain("Verificar um artefacto");
    // *Um painel que mostra "pronto" sem poder construir e' um painel que promete o que nao faz --
    // a menos que diga que so' le'.*
    expect(texto).toContain("já existe");
    expect(texto).toContain("absoluto");
  });

  it("mede o artefacto e mostra o SHA-256 devolvido pelo backend", async () => {
    responder = () => medicao(SHA);
    const arvore = await montar();
    await escrever(arvore, "build-artefacto-caminho", "C:\\artefacto.exe");
    await act(async () => {
      botao(arvore, "Medir").props.onClick();
    });

    const doChecksum = pedidos.filter((u) => u.includes("/artifact/checksum"));
    expect(doChecksum).toHaveLength(1);
    expect(doChecksum[0]).toContain(encodeURIComponent("C:\\artefacto.exe"));

    const texto = textoDosNos(arvore);
    expect(texto).toContain(SHA);
    expect(texto).toContain("39.2 MB");
  });

  it("verifica com o caminho E a referencia, e mostra o veredito", async () => {
    responder = () =>
      verificacao({
        medido: SHA,
        esperado: SHA,
        assinatura: "AUSENTE",
        aprovado: true,
        adulterado: false,
      });
    const arvore = await montar();
    await escrever(arvore, "build-artefacto-caminho", "C:\\artefacto.exe");
    await escrever(arvore, "build-artefacto-sha256", SHA);
    await act(async () => {
      botao(arvore, "Verificar").props.onClick();
    });

    const daVerificacao = pedidos.filter((u) => u.includes("/artifact/verify"));
    expect(daVerificacao).toHaveLength(1);
    expect(daVerificacao[0]).toContain(`sha256=${SHA}`);

    const texto = textoDosNos(arvore);
    expect(texto).toContain("íntegro");
    expect(texto).toContain("não assinada (esperado nesta RC)");
    // O aviso de fronteira vem do BACKEND e e' mostrado tal e qual.
    expect(texto).toContain("estas rotas apenas MEDEM e COMPARAM");
  });

  it("mostra ADULTERADO quando o checksum nao corresponde", async () => {
    responder = () =>
      verificacao({
        medido: "0".repeat(64),
        esperado: SHA,
        assinatura: "AUSENTE",
        aprovado: false,
        adulterado: true,
      });
    const arvore = await montar();
    await escrever(arvore, "build-artefacto-caminho", "C:\\artefacto.exe");
    await escrever(arvore, "build-artefacto-sha256", SHA);
    await act(async () => {
      botao(arvore, "Verificar").props.onClick();
    });

    const texto = textoDosNos(arvore);
    expect(texto).toContain("ADULTERADO");
    // *Um estado que mente e' pior do que nenhum.*
    expect(texto).not.toContain("íntegro —");
  });

  it("distingue assinatura NAO_VERIFICADA de AUSENTE", async () => {
    responder = () =>
      verificacao({
        medido: SHA,
        esperado: SHA,
        assinatura: "NAO_VERIFICADA",
        aprovado: true,
        adulterado: false,
      });
    const arvore = await montar();
    await escrever(arvore, "build-artefacto-caminho", "C:\\artefacto.exe");
    await escrever(arvore, "build-artefacto-sha256", SHA);
    await act(async () => {
      botao(arvore, "Verificar").props.onClick();
    });

    const texto = textoDosNos(arvore);
    // "nao consegui olhar" e "olhei e nao ha'" sao factos diferentes: o ecra tem de os separar.
    expect(texto).toContain("assinatura não verificada neste host");
    expect(texto).not.toContain("não assinada (esperado nesta RC)");
  });

  it("desactiva Verificar sem referencia", async () => {
    const arvore = await montar();
    await escrever(arvore, "build-artefacto-caminho", "C:\\artefacto.exe");
    // Sem `sha256`, o botao tem de estar desactivado: *sem referencia isto seria uma medicao.*
    expect(botao(arvore, "Verificar").props.disabled).toBe(true);
    expect(botao(arvore, "Medir").props.disabled).toBe(false);
  });

  it("mostra a falha do backend como alerta", async () => {
    responder = () => {
      throw new Error("o parametro `caminho` tem de ser ABSOLUTO");
    };
    const arvore = await montar();
    await escrever(arvore, "build-artefacto-caminho", "relativo.exe");
    await act(async () => {
      botao(arvore, "Medir").props.onClick();
    });

    const alerta = arvore.root.findByProps({ role: "alert" });
    expect(String(alerta.props.children)).toContain("ABSOLUTO");
  });
});
