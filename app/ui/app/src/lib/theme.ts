import { useCallback, useEffect, useState } from "react";

// Controle de tema estilo Manus: Claro / Escuro / Automático (segue o SO).
// "auto" não adiciona classe e deixa o `prefers-color-scheme` decidir; "light"
// e "dark" forçam via classe no <html> (ver o @custom-variant dark em index.css).

export type ThemePreference = "auto" | "light" | "dark";

const STORAGE_KEY = "ollama-full-theme";

export function getStoredTheme(): ThemePreference {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    if (value === "light" || value === "dark" || value === "auto") return value;
  } catch {
    // localStorage pode falhar (modo privado); cai no padrão.
  }
  return "auto";
}

/** Aplica a preferência ao documento (classe no <html>). */
export function applyTheme(pref: ThemePreference): void {
  if (typeof document === "undefined") return;
  const root = document.documentElement;
  root.classList.remove("theme-light", "theme-dark");
  if (pref === "light") root.classList.add("theme-light");
  else if (pref === "dark") root.classList.add("theme-dark");
  // "auto": sem classe → segue o sistema.
}

export function persistTheme(pref: ThemePreference): void {
  try {
    localStorage.setItem(STORAGE_KEY, pref);
  } catch {
    // Ignora falha de persistência; a aplicação em memória ainda vale.
  }
  applyTheme(pref);
}

/** Aplica o tema salvo o quanto antes (chamar no bootstrap, antes do render). */
export function bootstrapTheme(): void {
  applyTheme(getStoredTheme());
}

/** Hook React para ler e trocar o tema. */
export function useTheme(): {
  theme: ThemePreference;
  setTheme: (pref: ThemePreference) => void;
} {
  const [theme, setThemeState] = useState<ThemePreference>(() => getStoredTheme());

  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  const setTheme = useCallback((pref: ThemePreference) => {
    persistTheme(pref);
    setThemeState(pref);
  }, []);

  return { theme, setTheme };
}
