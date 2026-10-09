// Opens an external URL (OAuth connect/authorize pages, docs, ollama.com…).
//
// In the desktop app's webview, window.open(…) silently does nothing — it
// returns null and opens no window anywhere (confirmed via CDP). So the app
// exposes a native `window.openExternal` binding that opens the system browser.
// In a normal browser (dev server / web), that binding is absent, so we fall
// back to window.open. Centralizing this keeps every "open link" working in
// both environments.
export function openExternal(url: string): void {
  if (typeof window === "undefined") return;
  if (typeof window.openExternal === "function") {
    void window.openExternal(url);
    return;
  }
  window.open(url, "_blank", "noopener,noreferrer");
}
