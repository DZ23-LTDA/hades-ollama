import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, describe, expect, it } from "vitest";
import { builtinConnectorCatalog } from "../lib/connectorCatalog";
import { getLanguage, setLanguage, t, useT } from "./index";

/**
 * Prova COMPORTAMENTAL da troca de idioma em tempo de execucao.
 *
 * Porque este ficheiro existe: a correcao do `lib/connectorCatalog.ts` (constante de modulo ->
 * funcao) foi justificada por desenho, mas o documento de evidencia declarava em aberto que
 * «nao existe teste de UI que alterne o idioma em tempo de execucao e verifique o repintar».
 * Estes testes fecham essa lacuna: nao provam que a correcao e' elegante, provam que o texto
 * muda de facto quando o idioma muda, sem recarregar nada.
 */

afterEach(() => {
  act(() => setLanguage("pt-BR"));
});

function Saudacao() {
  const { t: traduzir, language } = useT();
  return (
    <span>
      {language}:{traduzir("productWorkspace.projects.title")}
    </span>
  );
}

describe("troca de idioma em tempo de execucao", () => {
  it("t() devolve texto diferente quando o idioma activo muda", () => {
    setLanguage("pt-BR");
    const emPortugues = t("productWorkspace.projects.title");
    act(() => setLanguage("en"));
    const emIngles = t("productWorkspace.projects.title");
    expect(getLanguage()).toBe("en");
    expect(emPortugues.length).toBeGreaterThan(0);
    expect(emIngles.length).toBeGreaterThan(0);
    expect(emPortugues).not.toBe(emIngles);
  });

  it("o catalogo de conectores NAO congela as descricoes no arranque", () => {
    // Se `builtinConnectorCatalog` voltasse a ser uma constante de modulo, as duas leituras
    // devolveriam o mesmo texto: e' exactamente esse defeito que este teste apanha.
    setLanguage("pt-BR");
    const descricaoPt = builtinConnectorCatalog()[0]?.description;
    act(() => setLanguage("en"));
    const descricaoEn = builtinConnectorCatalog()[0]?.description;
    expect(typeof descricaoPt).toBe("string");
    expect(typeof descricaoEn).toBe("string");
    expect(descricaoEn).not.toBe(descricaoPt);
  });

  it("um componente com useT() repinta quando o idioma muda", () => {
    let arvore: ReactTestRenderer;
    act(() => {
      arvore = create(<Saudacao />);
    });
    const antes = JSON.stringify(arvore!.toJSON());
    expect(antes).toContain("pt-BR");

    act(() => setLanguage("en"));

    const depois = JSON.stringify(arvore!.toJSON());
    expect(depois).toContain("en");
    expect(depois).not.toBe(antes);
  });
});
