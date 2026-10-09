// API configuration
// In dev, default to same-origin ("") so requests flow through Vite's `/api`
// proxy (vite.config.ts → 127.0.0.1:11434). This avoids the old dead-port
// fallback (3001) that made `npm run dev` look offline against a running Hades,
// and sidesteps cross-origin CORS preflights. Set VITE_API_URL only to target a
// different backend explicitly.
const DEV_API_URL = import.meta.env.VITE_API_URL ?? "";

// Base URL for fetch API calls (can be relative in production)
export const API_BASE = import.meta.env.DEV ? DEV_API_URL : "";

// Full host URL for Ollama client (needs full origin in production)
export const OLLAMA_HOST = import.meta.env.DEV
  ? DEV_API_URL
  : window.location.origin;

export const OLLAMA_DOT_COM =
  import.meta.env.VITE_OLLAMA_DOT_COM_URL || "https://ollama.com";
