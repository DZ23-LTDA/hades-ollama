import { useSyncExternalStore } from "react";
import { enMessages } from "./messages.en";
import { ptBRMessages } from "./messages.pt-BR";

export type Language = "pt-BR" | "en";
export type TranslateParams = Record<string, string | number>;
export type MessageCatalog = Record<string, string>;
export type Translator = (key: string, params?: TranslateParams) => string;

const PARAM_PATTERN = /\{(\w+)\}/g;

/** Substitui `{nome}` por `params.nome`; placeholders sem parâmetro ficam intactos. */
function interpolate(template: string, params?: TranslateParams): string {
  if (!params) return template;
  return template.replace(PARAM_PATTERN, (placeholder, name: string) =>
    Object.prototype.hasOwnProperty.call(params, name)
      ? String(params[name])
      : placeholder,
  );
}

/**
 * Cria um tradutor isolado a partir de dois catálogos.
 * A ordem de fallback é sempre: idioma pedido → pt-BR → a própria chave.
 * Exportado para que os testes possam provar o fallback com catálogos próprios.
 */
export function createTranslator(catalogs: Record<Language, MessageCatalog>) {
  return function translate(
    language: Language,
    key: string,
    params?: TranslateParams,
  ): string {
    const template = catalogs[language][key] ?? catalogs["pt-BR"][key] ?? key;
    return interpolate(template, params);
  };
}

export const translate = createTranslator({
  en: enMessages,
  "pt-BR": ptBRMessages,
});

let activeLanguage: Language = "pt-BR";
const listeners = new Set<() => void>();

/** Idioma activo no momento (pt-BR por omissão). */
export function getLanguage(): Language {
  return activeLanguage;
}

/** Troca o idioma activo e notifica os componentes subscritos. */
export function setLanguage(language: Language): void {
  if (language === activeLanguage) return;
  activeLanguage = language;
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/**
 * Traduz com o idioma activo no momento da chamada.
 * A identidade da função é estável, por isso pode ser usada em callbacks.
 */
export const t: Translator = (key, params) =>
  translate(activeLanguage, key, params);

/**
 * Hook de tradução: subscreve às mudanças de idioma para forçar re-render
 * e devolve o tradutor estável. Fora do React, use `t` directamente.
 */
export function useT(): { t: Translator; language: Language } {
  const language = useSyncExternalStore(subscribe, getLanguage, getLanguage);
  return { t, language };
}
