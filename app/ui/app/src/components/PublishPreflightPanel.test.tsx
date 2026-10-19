/**
 * E-1298: testes do painel de `publish.preflight`.
 *
 * O que estes testes travam:
 *
 *  1. o painel diz que **nao publica** e que o provider nao tem `Publicar()`;
 *  2. **nao tem campo para o artefacto nem para a credencial** -- vem do ambiente do servidor;
 *  3. **nao existe botao de publicar**;
 *  4. `LOCAL_ONLY` e' apresentado como **ensaio local**, nunca como publicado;
 *  5. canal publico avisa que exige confirmacao humana;
 *  6. as duas garantias (`somente_leitura`, `nada_publicado`) aparecem **separadas** do veredicto;
 *  7. `publicado: true` seria dito como **inconsistente**, nao como sucesso;
 *  8. condicao pendente e' mostrada como PENDENTE.
 *
 * Mocka-se **apenas o `agentFetch`**.
 */

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockAgentFetch = vi.fn();

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: (...args: unknown[]) => mockAgentFetch(...args),
}));

import { PublishPreflightPanel } from "@/components/PublishPreflightPanel";

/** Resposta de exemplo com os valores REAIS do backend. */
function respostaLocal(sobrescrever: Record<string, unknown> = {}) {
  return {
    somente_leitura: true,
    nada_publicado: true,
    preflight: {
      canal: "LOCAL_ONLY",
      publico: false,
      pronto: true,
      condicoes: [
        { nome: "artefacto_existe", satisfeita: true, detalhe: "dist/hades.exe" },
        { nome: "credencial_presente", satisfeita: true, detalhe: "nao necessaria para local" },
      ],
      requer_aprovacao: false,
      publicado: false,
      explicacao: "canal local nao envia para fora",
      aviso: "",
      ...sobrescrever,
    },
    aviso_de_fronteira:
      "esta rota apenas LE' o artefacto e a credencial do ambiente do servidor; nao publica. " +
      "O pedido escolhe o CANAL, nunca o caminho nem o segredo.",
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
    if (typeof valor === "object" && "props" in valor) {
      const props = (valor as { props: { children?: unknown } }).props;
      if (props && "children" in props) visitar(props.children);
    }
  };
  visitar(no.toJSON());
  return partes.join(" ");
}

function porTestId(no: ReactTestRenderer, id: string): string | null {
  const encontrados = no.root.findAll(
    (elemento) => elemento.props["data-testid"] === id,
  );
  if (encontrados.length === 0) return null;
  const textos: string[] = [];
  const visitar = (valor: unknown): void => {
    if (valor === null || valor === undefined) return;
    if (typeof valor === "string" || typeof valor === "number") {
      textos.push(String(valor));
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
  visitar(encontrados[0].props.children);
  return textos.join(" ");
}

describe("PublishPreflightPanel", () => {
  let arvore: ReactTestRenderer;

  beforeEach(() => {
    mockAgentFetch.mockReset();
  });

  afterEach(() => {
    if (arvore) {
      act(() => {
        arvore.unmount();
      });
    }
  });

  async function montar() {
    await act(async () => {
      arvore = create(<PublishPreflightPanel />);
    });
    return arvore;
  }

  async function verificar() {
    const botao = arvore.root.findAll(
      (elemento) =>
        elemento.props["data-testid"] === "preflight-verificar" &&
        typeof elemento.props.onClick === "function",
    );
    await act(async () => {
      botao[0].props.onClick();
    });
  }

  it("diz que nao publica e que o provider nao tem Publicar()", async () => {
    await montar();
    const texto = textoDosNos(arvore);
    expect(texto).toContain("Nao publica");
    expect(texto).toContain("Publicar()");
  });

  it("nao tem campo para o artefacto nem para a credencial", async () => {
    await montar();
    const campos = arvore.root.findAll((elemento) => elemento.type === "input");
    // O painel nao tem NENHUM input de texto: so' o select do canal.
    expect(campos).toHaveLength(0);
    const texto = textoDosNos(arvore);
    expect(texto).toContain("HADES_ARTEFACTO_A_PUBLICAR");
    expect(texto).toContain("HADES_CREDENCIAL_DE_PUBLICACAO");
  });

  it("nao existe botao de publicar", async () => {
    await montar();
    const botoes = arvore.root.findAll((elemento) => elemento.type === "button");
    const rotulos = botoes.map((botao) => String(botao.props.children ?? ""));
    expect(rotulos.join(" ")).not.toMatch(/publicar/i);
    expect(rotulos.join(" ")).toContain("Verificar canal");
  });

  it("LOCAL_ONLY e' apresentado como ensaio, nunca como publicado", async () => {
    await montar();
    const texto = textoDosNos(arvore);
    expect(texto).toContain("ensaio");
    expect(texto).toContain("Nada sai desta maquina");
    expect(texto).toContain("nao");
    expect(porTestId(arvore, "preflight-canal-local")).not.toBeNull();
    expect(porTestId(arvore, "preflight-canal-publico")).toBeNull();
  });

  it("chama a rota com o canal escolhido", async () => {
    mockAgentFetch.mockResolvedValue(respostaLocal());
    await montar();
    await verificar();
    expect(mockAgentFetch).toHaveBeenCalledWith(
      "/api/agent/v1/publish/preflight/LOCAL_ONLY",
      { method: "POST" },
    );
  });

  it("mostra as duas garantias separadas do veredicto", async () => {
    mockAgentFetch.mockResolvedValue(respostaLocal());
    await montar();
    await verificar();
    expect(porTestId(arvore, "preflight-somente-leitura")).toContain("sim");
    expect(porTestId(arvore, "preflight-nada-publicado")).toContain("sim");
    expect(porTestId(arvore, "preflight-publicado")).toContain("nao");
  });

  it("mostra condicao pendente como PENDENTE", async () => {
    mockAgentFetch.mockResolvedValue(
      respostaLocal({
        pronto: false,
        condicoes: [
          { nome: "artefacto_existe", satisfeita: false, detalhe: "nao encontrado" },
          { nome: "credencial_presente", satisfeita: true, detalhe: "ok" },
        ],
      }),
    );
    await montar();
    await verificar();
    expect(porTestId(arvore, "preflight-condicao-artefacto_existe")).toContain("PENDENTE");
    expect(porTestId(arvore, "preflight-condicao-credencial_presente")).toContain("OK");
    // O JSX separa os numeros em nos de texto; o helper junta com espaco. Normalizar antes de comparar.
    const contagem = (porTestId(arvore, "preflight-condicoes-contagem") ?? "").replace(/\s+/g, "");
    expect(contagem).toBe("1/2");
  });

  it("publicado:true e' dito INCONSISTENTE, nunca como sucesso", async () => {
    mockAgentFetch.mockResolvedValue(respostaLocal({ publicado: true }));
    await montar();
    await verificar();
    expect(porTestId(arvore, "preflight-resumo")).toContain("INCONSISTENTE");
  });

  it("falha do servidor vira erro legivel, nao sucesso", async () => {
    mockAgentFetch.mockRejectedValue(new Error("412: artefacto ausente"));
    await montar();
    await verificar();
    expect(porTestId(arvore, "preflight-erro")).toContain("412");
    expect(porTestId(arvore, "preflight-resultado")).toBeNull();
  });

  it("canal publico avisa que exige confirmacao humana", async () => {
    await montar();
    const select = arvore.root.findAll(
      (elemento) =>
        elemento.props["data-testid"] === "preflight-select-canal" &&
        typeof elemento.props.onChange === "function",
    );
    await act(async () => {
      select[0].props.onChange({ target: { value: "GOOGLE_PLAY" } });
    });
    expect(porTestId(arvore, "preflight-canal-publico")).toContain("confirmacao humana");
    expect(porTestId(arvore, "preflight-canal-local")).toBeNull();
  });
});
