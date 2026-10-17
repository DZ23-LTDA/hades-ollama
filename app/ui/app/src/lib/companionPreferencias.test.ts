import { describe, expect, it } from "vitest";
import {
  CHAVE_DAS_PREFERENCIAS,
  PREFERENCIAS_PADRAO,
  carregarPreferencias,
  guardarPreferencias,
  podeDesenharCompanion,
  razaoParaNaoAnimar,
  restaurarPreferencias,
  sistemaPedeMenosMovimento,
  validarPreferencias,
} from "./companionPreferencias";

/**
 * `E-1084` — o teste das preferências do Companion.
 *
 * ***O QUE ESTE TESTE TEM DE DISCRIMINAR.*** Cada caso abaixo parte de um valor **que o produto
 * poderia receber** e exige o comportamento certo. *Um teste que só verificasse «a função existe»
 * passaria com o ficheiro vazio.*
 */

/** Um armazenamento em memória — *e o teste injecta-o, em vez de tocar no `localStorage` real.* */
function armazemFalso(inicial: Record<string, string> = {}) {
  const dados = { ...inicial };
  return {
    dados,
    getItem: (k: string) => dados[k] ?? null,
    setItem: (k: string, v: string) => {
      dados[k] = v;
    },
  };
}

describe("validarPreferencias", () => {
  it("um valor que nao e' objecto cai no default", () => {
    expect(validarPreferencias(null)).toEqual(PREFERENCIAS_PADRAO);
    expect(validarPreferencias("lixo")).toEqual(PREFERENCIAS_PADRAO);
    expect(validarPreferencias(42)).toEqual(PREFERENCIAS_PADRAO);
  });

  it("um campo com o tipo errado cai no default DESSE campo, e nao derruba os outros", () => {
    const r = validarPreferencias({ visivel: "sim", silenciado: true, movimentoReduzido: "x" });
    expect(r.visivel).toBe(PREFERENCIAS_PADRAO.visivel);
    expect(r.silenciado).toBe(true);
    expect(r.movimentoReduzido).toBe(PREFERENCIAS_PADRAO.movimentoReduzido);
  });

  it("nao inventa campos que nao existem", () => {
    const r = validarPreferencias({ visivel: false, invadido: true });
    expect(Object.keys(r).sort()).toEqual(["movimentoReduzido", "silenciado", "visivel"]);
  });
});

describe("carregarPreferencias", () => {
  it("sem nada guardado devolve o default", () => {
    expect(carregarPreferencias(armazemFalso())).toEqual(PREFERENCIAS_PADRAO);
  });

  it("sem armazenamento nenhum devolve o default (e nao rebenta)", () => {
    expect(carregarPreferencias(null)).toEqual(PREFERENCIAS_PADRAO);
  });

  it("JSON invalido devolve o default, em vez de propagar o erro", () => {
    const a = armazemFalso({ [CHAVE_DAS_PREFERENCIAS]: "{isto nao e' json" });
    expect(carregarPreferencias(a)).toEqual(PREFERENCIAS_PADRAO);
  });

  it("le' o que foi guardado", () => {
    const a = armazemFalso({
      [CHAVE_DAS_PREFERENCIAS]: JSON.stringify({ visivel: false, silenciado: true, movimentoReduzido: true }),
    });
    expect(carregarPreferencias(a)).toEqual({ visivel: false, silenciado: true, movimentoReduzido: true });
  });
});

describe("guardarPreferencias", () => {
  it("guarda e devolve true", () => {
    const a = armazemFalso();
    const ok = guardarPreferencias({ visivel: false, silenciado: true, movimentoReduzido: false }, a);
    expect(ok).toBe(true);
    expect(JSON.parse(a.dados[CHAVE_DAS_PREFERENCIAS])).toEqual({
      visivel: false,
      silenciado: true,
      movimentoReduzido: false,
    });
  });

  it("sem armazenamento devolve FALSE -- a UI tem de poder dizer que nao persistiu", () => {
    expect(guardarPreferencias({ ...PREFERENCIAS_PADRAO }, null)).toBe(false);
  });

  it("um armazenamento cheio devolve FALSE, em vez de fingir que guardou", () => {
    const cheio = {
      getItem: () => null,
      setItem: () => {
        throw new Error("QuotaExceededError");
      },
    };
    expect(guardarPreferencias({ ...PREFERENCIAS_PADRAO }, cheio)).toBe(false);
  });
});

describe("restaurarPreferencias", () => {
  it("restaura o default por cima do que estava", () => {
    const a = armazemFalso({ [CHAVE_DAS_PREFERENCIAS]: JSON.stringify({ visivel: false, silenciado: true, movimentoReduzido: true }) });
    expect(restaurarPreferencias(a)).toBe(true);
    expect(carregarPreferencias(a)).toEqual(PREFERENCIAS_PADRAO);
  });
});

describe("razaoParaNaoAnimar -- a PRECEDENCIA", () => {
  it("o PEDIDO do utilizador vence o SISTEMA", () => {
    expect(razaoParaNaoAnimar({ ...PREFERENCIAS_PADRAO, movimentoReduzido: true }, true)).toBe("pedido");
  });

  it("o SISTEMA e' a segunda razao", () => {
    expect(razaoParaNaoAnimar(PREFERENCIAS_PADRAO, true)).toBe("sistema");
  });

  it("nenhum dos dois => anima", () => {
    expect(razaoParaNaoAnimar(PREFERENCIAS_PADRAO, false)).toBe("nenhum");
  });
});

describe("sistemaPedeMenosMovimento", () => {
  it("le' o matchMedia", () => {
    expect(sistemaPedeMenosMovimento(() => ({ matches: true }))).toBe(true);
    expect(sistemaPedeMenosMovimento(() => ({ matches: false }))).toBe(false);
  });

  it("sem matchMedia devolve false -- a ausencia de informacao nao desliga nada a quem nao pediu", () => {
    expect(sistemaPedeMenosMovimento(null)).toBe(false);
  });

  it("matchMedia que rebenta devolve false, em vez de derrubar o render", () => {
    expect(
      sistemaPedeMenosMovimento(() => {
        throw new Error("nao suportado");
      }),
    ).toBe(false);
  });
});

describe("podeDesenharCompanion -- a guarda final", () => {
  it("visivel + estado conhecido => desenha", () => {
    expect(podeDesenharCompanion(PREFERENCIAS_PADRAO, "a_trabalhar")).toBe(true);
  });

  it("invisivel => NAO desenha, mesmo com estado legitimo", () => {
    expect(podeDesenharCompanion({ ...PREFERENCIAS_PADRAO, visivel: false }, "a_trabalhar")).toBe(false);
  });

  it("estado DESCONHECIDO => NAO desenha, mesmo com o pet visivel", () => {
    expect(podeDesenharCompanion(PREFERENCIAS_PADRAO, "a_dancar")).toBe(false);
  });

  it("estado vazio => NAO desenha", () => {
    expect(podeDesenharCompanion(PREFERENCIAS_PADRAO, "")).toBe(false);
  });
});
