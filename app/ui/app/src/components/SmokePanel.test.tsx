/**
 * FASE 28.8: testes do painel de SMOKE.
 *
 * O que estes testes travam:
 *
 *  1. o painel diz que **executa** antes de haver resultado;
 *  2. o botao esta' desactivado sem caminho -- nao se executa o que nao foi indicado;
 *  3. um resultado aprovado mostra exit code, duracao e saida;
 *  4. **o aviso de sandbox do backend e' mostrado por inteiro**;
 *  5. **o painel NAO diz "funciona"** -- diz que arrancou e respondeu;
 *  6. um timeout e' mostrado como *"um processo que nao termina nao e' um processo que passou"*;
 *  7. uma falha aparece em `role="alert"`;
 *  8. os argumentos sao separados por espacos com suporte a aspas, **sem imitar o shell**.
 *
 * Mocka-se **apenas o `agentFetch`**, nao o cliente:
 *
 *	*Um teste que substitui o cliente testa o mock, nao o produto.*
 */

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockAgentFetch = vi.fn();

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: (...args: unknown[]) => mockAgentFetch(...args),
}));

import { SmokePanel } from "@/components/SmokePanel";
import { separarArgumentos } from "@/lib/smokeClient";

/** Resultado de exemplo. */
function resultadoDeExemplo(sobrescrever: Record<string, unknown> = {}) {
  return {
    executou: true,
    aprovado: true,
    codigo_de_saida: 0,
    duracao_ms: 42,
    saida: "ollama version is 0.40.1",
    saida_truncada: false,
    timeout: false,
    explicacao: "o artefacto arrancou, terminou com exit code 0 em 42 ms",
    sem_sandbox: true,
    aviso_de_sandbox:
      "este smoke executou o artefacto com os privilegios deste processo. NAO ha' sandbox.",
    raiz_autorizada: true,
    ...sobrescrever,
  };
}

/** Texto de toda a arvore. */
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
    arvore = create(<SmokePanel />);
  });
  return arvore;
}

async function escrever(arvore: ReactTestRenderer, id: string, valor: string): Promise<void> {
  await act(async () => {
    arvore.root.findByProps({ id }).props.onChange({ target: { value: valor } });
  });
}

function botao(arvore: ReactTestRenderer) {
  return arvore.root.find(
    (no) => no.type === "button" && String(no.props.children).includes("Executar smoke"),
  );
}

beforeEach(() => {
  mockAgentFetch.mockReset();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("SmokePanel", () => {
  it("diz que EXECUTA antes de haver resultado", async () => {
    const arvore = await montar();
    const texto = textoDosNos(arvore);
    expect(texto).toContain("executa");
    expect(texto).toContain("nao escolhe a raiz");
    arvore.unmount();
  });

  it("mantem o botao desactivado sem caminho", async () => {
    const arvore = await montar();
    expect(botao(arvore).props.disabled).toBe(true);
    await escrever(arvore, "smoke-caminho", "C:/artefactos/hades.exe");
    expect(botao(arvore).props.disabled).toBe(false);
    arvore.unmount();
  });

  it("mostra exit code, duracao e saida de um resultado aprovado", async () => {
    mockAgentFetch.mockResolvedValue(resultadoDeExemplo());
    const arvore = await montar();
    await escrever(arvore, "smoke-caminho", "C:/artefactos/hades.exe");
    await escrever(arvore, "smoke-argumentos", "--version");
    await act(async () => {
      botao(arvore).props.onClick();
    });

    expect(mockAgentFetch).toHaveBeenCalledWith(
      "/api/agent/v1/build/smoke",
      expect.objectContaining({ method: "POST" }),
    );
    const texto = textoDosNos(arvore);
    expect(texto).toContain("0.40.1");
    expect(texto).toContain("42 ms");
    expect(arvore.root.findByProps({ "data-testid": "smoke-codigo" }).props.children).toBe(0);
    arvore.unmount();
  });

  // O aviso de sandbox vem do BACKEND e nao e' reescrito.
  it("mostra o aviso de sandbox do backend por inteiro", async () => {
    mockAgentFetch.mockResolvedValue(resultadoDeExemplo());
    const arvore = await montar();
    await escrever(arvore, "smoke-caminho", "C:/artefactos/hades.exe");
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("NAO ha' sandbox");
    expect(texto).toContain("privilegios deste processo");
    arvore.unmount();
  });

  // O PAINEL NAO PODE DIZER QUE FUNCIONA.
  it("nao afirma que o artefacto funciona", async () => {
    mockAgentFetch.mockResolvedValue(resultadoDeExemplo());
    const arvore = await montar();
    await escrever(arvore, "smoke-caminho", "C:/artefactos/hades.exe");
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("NAO provado que faz o que promete");
    // E nao pode conter uma afirmacao de seguranca que ninguem mediu.
    expect(texto.toLowerCase()).not.toContain("binario seguro");
    arvore.unmount();
  });

  it("mostra um timeout como processo que nao terminou", async () => {
    mockAgentFetch.mockResolvedValue(
      resultadoDeExemplo({
        aprovado: false,
        timeout: true,
        duracao_ms: 60000,
        explicacao: "o artefacto nao terminou em 60s",
      }),
    );
    const arvore = await montar();
    await escrever(arvore, "smoke-caminho", "C:/artefactos/hades.exe");
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("NAO terminou");
    expect(texto).toContain("nao termina nao");
    arvore.unmount();
  });

  it("mostra uma falha em role=alert", async () => {
    mockAgentFetch.mockRejectedValue(new Error("403 caminho fora da raiz autorizada"));
    const arvore = await montar();
    await escrever(arvore, "smoke-caminho", "C:/fora/hades.exe");
    await act(async () => {
      botao(arvore).props.onClick();
    });
    expect(arvore.root.findByProps({ role: "alert" })).toBeDefined();
    expect(textoDosNos(arvore)).toContain("fora da raiz autorizada");
    arvore.unmount();
  });
});

describe("separarArgumentos", () => {
  it("separa por espacos", () => {
    expect(separarArgumentos("--version")).toEqual(["--version"]);
    expect(separarArgumentos("run --fast x")).toEqual(["run", "--fast", "x"]);
  });

  it("respeita aspas duplas", () => {
    expect(separarArgumentos('"a b" c')).toEqual(["a b", "c"]);
  });

  // Nao imita o shell: um `$` fica literal.
  it("nao expande variaveis de shell", () => {
    expect(separarArgumentos("$HOME/x")).toEqual(["$HOME/x"]);
  });

  it("ignora espacos a mais", () => {
    expect(separarArgumentos("  a   b  ")).toEqual(["a", "b"]);
    expect(separarArgumentos("")).toEqual([]);
  });
});
