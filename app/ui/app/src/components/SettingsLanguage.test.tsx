import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, describe, expect, it } from "vitest";
import { getLanguage, setLanguage, useT } from "@/i18n";
import {
  bootstrapLanguage,
  getStoredLanguage,
  LANGUAGE_STORAGE_KEY,
  type LanguageStorage,
} from "@/i18n/persistencia";
import { SettingsLanguageControl } from "./SettingsLanguage";

/**
 * Prova que o controlo de idioma de Settings é UTILIZÁVEL, não só que a camada
 * i18n repinta. O teste que já existia (`src/i18n/troca-idioma.test.tsx`) chama
 * `setLanguage()` diretamente; estes testes clicam num botão renderizado e
 * verificam o texto de um componente irmão — o caminho que o utilizador faz.
 */

function fakeStorage(initial: Record<string, string> = {}) {
  const data = new Map(Object.entries(initial));
  return {
    getItem: (key: string) => data.get(key) ?? null,
    setItem: (key: string, value: string) => {
      data.set(key, value);
    },
  } satisfies LanguageStorage;
}

/** Consumidor independente do controlo: prova o repintar fora do próprio botão. */
function TituloTraduzido() {
  const { t } = useT();
  return <p id="amostra">{t("settings.language.title")}</p>;
}

function Preferencias({ storage }: { storage: LanguageStorage }) {
  return (
    <div>
      <SettingsLanguageControl storage={storage} />
      <TituloTraduzido />
    </div>
  );
}

function render(storage: LanguageStorage): ReactTestRenderer {
  let arvore!: ReactTestRenderer;
  act(() => {
    arvore = create(<Preferencias storage={storage} />);
  });
  return arvore;
}

function clicarIdioma(arvore: ReactTestRenderer, lang: "pt-BR" | "en") {
  const botoes = arvore.root.findAll(
    (node) => node.type === "button" && node.props.lang === lang,
  );
  expect(botoes).toHaveLength(1);
  act(() => {
    botoes[0].props.onClick();
  });
}

afterEach(() => {
  act(() => setLanguage("pt-BR"));
});

describe("controlo de idioma em Settings", () => {
  it("muda o texto de um componente renderizado ao clicar em English", () => {
    act(() => setLanguage("pt-BR"));
    const arvore = render(fakeStorage());
    expect(JSON.stringify(arvore.toJSON())).toContain("Idioma da interface");

    clicarIdioma(arvore, "en");

    const depois = JSON.stringify(arvore.toJSON());
    // O texto mudou no controlo E no componente irmão (que só usa useT()).
    expect(depois).toContain("Interface language");
    expect(depois).not.toContain("Idioma da interface");
    expect(getLanguage()).toBe("en");
  });

  it("persiste a escolha no localStorage", () => {
    const storage = fakeStorage();
    const arvore = render(storage);

    clicarIdioma(arvore, "en");

    expect(storage.getItem(LANGUAGE_STORAGE_KEY)).toBe("en");
    expect(getStoredLanguage(storage)).toBe("en");
  });

  it("aplica a escolha guardada antes do primeiro render (sem flash)", () => {
    const storage = fakeStorage({ [LANGUAGE_STORAGE_KEY]: "en" });
    act(() => setLanguage("pt-BR"));

    expect(bootstrapLanguage(storage)).toBe("en");

    const primeiroRender = JSON.stringify(render(storage).toJSON());
    expect(primeiroRender).toContain("Interface language");
    expect(primeiroRender).not.toContain("Idioma da interface");
  });

  it("ignora um valor guardado fora dos idiomas suportados", () => {
    const storage = fakeStorage({ [LANGUAGE_STORAGE_KEY]: "fr" });
    act(() => setLanguage("pt-BR"));

    expect(bootstrapLanguage(storage)).toBe("pt-BR");
    expect(getStoredLanguage(storage)).toBeNull();
  });
});
