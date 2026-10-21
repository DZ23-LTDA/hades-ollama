// Persistência da escolha de idioma da interface.
//
// Porque este módulo existe, e porque é separado de `./index.ts`: o módulo i18n
// é a fonte da verdade do idioma ACTIVO (memória + subscritores), mas não sabia
// guardar nem reaplicar a escolha do utilizador. `setLanguage()` só era chamado
// dentro de testes, portanto não havia controlo nenhum na UI e o idioma voltava
// sempre a pt-BR em cada arranque. Este ficheiro acrescenta a camada de
// preferência (localStorage + bootstrap) SEM tocar em `./index.ts`.

import { getLanguage, setLanguage, type Language } from "./index";

/** Chave usada no localStorage para recordar o idioma escolhido. */
export const LANGUAGE_STORAGE_KEY = "hades_language";

/** Idiomas que a UI sabe mostrar; a ordem é a ordem dos botões. */
export const SUPPORTED_LANGUAGES: readonly Language[] = ["pt-BR", "en"];

/**
 * Subconjunto de `Storage` que a persistência precisa.
 * Injectável para que os testes possam correr sem `window` (o projecto usa
 * `environment: "node"` no vitest) e para que um storage restrito não quebre a UI.
 */
export type LanguageStorage = Pick<Storage, "getItem" | "setItem">;

function defaultStorage(): LanguageStorage | undefined {
  try {
    return typeof window === "undefined" ? undefined : window.localStorage;
  } catch {
    return undefined;
  }
}

/** Restringe um valor arbitrário do storage a um idioma suportado (ou `null`). */
export function toLanguage(value: string | null | undefined): Language | null {
  return value === "pt-BR" || value === "en" ? value : null;
}

/** Devolve o idioma guardado, ou `null` quando não há preferência válida. */
export function getStoredLanguage(
  storage: LanguageStorage | undefined = defaultStorage(),
): Language | null {
  try {
    return toLanguage(storage?.getItem(LANGUAGE_STORAGE_KEY));
  } catch {
    // Modo privado / webview restrita: sem preferência guardada.
    return null;
  }
}

/** Guarda a escolha; nunca lança (a troca em memória já aconteceu). */
export function persistLanguage(
  language: Language,
  storage: LanguageStorage | undefined = defaultStorage(),
): void {
  try {
    storage?.setItem(LANGUAGE_STORAGE_KEY, language);
  } catch {
    // Falhar a persistência não pode falhar a troca de idioma.
  }
}

/** Mantém o atributo `lang` do documento coerente com o idioma visível. */
function applyDocumentLanguage(language: Language): void {
  try {
    if (typeof document !== "undefined") {
      document.documentElement.lang = language;
    }
  } catch {
    // Sem DOM (testes/node) não há nada a sincronizar.
  }
}

/**
 * Troca o idioma activo e persiste a escolha.
 * É esta a função que o controlo de Settings chama.
 */
export function changeLanguage(
  language: Language,
  storage: LanguageStorage | undefined = defaultStorage(),
): void {
  setLanguage(language);
  persistLanguage(language, storage);
  applyDocumentLanguage(language);
}

/**
 * Lê a preferência guardada e aplica-a ANTES do primeiro render.
 * Devolve o idioma efectivamente activo (útil para diagnóstico e testes).
 */
export function bootstrapLanguage(
  storage: LanguageStorage | undefined = defaultStorage(),
): Language {
  const stored = getStoredLanguage(storage);
  if (stored) setLanguage(stored);
  const active = getLanguage();
  applyDocumentLanguage(active);
  return active;
}
