import { describe, expect, it } from "vitest";
import { estadoParaMostrar } from "./CompanionLigado";

/**
 * A decisao da ligacao, testada sem DOM (`vitest` sem jsdom, como o resto do repo):
 * *o que se mostra quando se SABE o estado, e o que se mostra quando NAO se sabe.*
 */
describe("estadoParaMostrar", () => {
  it("mostra o estado e a razao que vieram do servidor", () => {
    const mostrado = estadoParaMostrar({
      tipo: "ok",
      estado: {
        estado: "a_aprovar",
        razao: "ha' aprovacao pendente",
        scope: "missao do chamador",
      },
    });

    expect(mostrado.estado).toBe("a_aprovar");
    expect(mostrado.razao).toBe("ha' aprovacao pendente");
  });

  it("quando a ligacao falha, declara a AUSENCIA - nunca um estado inventado", () => {
    const mostrado = estadoParaMostrar({ tipo: "falhou", motivo: "500" });

    expect(mostrado.estado).toBe("ausente");
    expect(mostrado.razao).toContain("500");
    // O que NAO pode acontecer: aparecer um estado de actividade que o servidor nunca afirmou.
    expect(mostrado.estado).not.toBe("a_trabalhar");
    expect(mostrado.razao).not.toBe("");
  });
});
