// URL-scheme guards for values that come from the backend, a model/mission, a
// connector, or any other untrusted source before they are placed in an href,
// window.open, or an <img> src. They block script-executing schemes
// (javascript:, data:text/html, vbscript:, …) while allowing ordinary links.

// safeHttpUrl returns url only when it is a same-origin relative path or an
// explicit http(s) URL; otherwise it returns a harmless fallback. Use it for
// anchor href / window.open targets so a `javascript:`/`data:` value cannot run.
export function safeHttpUrl(
  url: string | undefined | null,
  fallback = "#",
): string {
  if (!url) return fallback;
  const trimmed = url.trim();
  if (trimmed === "") return fallback;
  // Same-origin relative path (but not a protocol-relative "//host").
  if (trimmed.startsWith("/") && !trimmed.startsWith("//")) return trimmed;
  if (/^https?:\/\//i.test(trimmed)) return trimmed;
  return fallback;
}

// safeImageSrc returns url only when it is a data:image/* URL, a same-origin
// relative path, or an http(s) URL; otherwise a harmless fallback. Use it for
// <img src> fed by untrusted data (e.g. a mission browser-frame screenshot).
export function safeImageSrc(
  url: string | undefined | null,
  fallback = "",
): string {
  if (!url) return fallback;
  const trimmed = url.trim();
  if (trimmed === "") return fallback;
  if (/^data:image\//i.test(trimmed)) return trimmed;
  if (trimmed.startsWith("/") && !trimmed.startsWith("//")) return trimmed;
  if (/^https?:\/\//i.test(trimmed)) return trimmed;
  return fallback;
}
