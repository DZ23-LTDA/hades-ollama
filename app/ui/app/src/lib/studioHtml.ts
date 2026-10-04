import type { BuilderProject, VisualComponent } from "@/lib/agenticClient";

// generateStudioHTML turns a Studio builder project into a single, standalone
// HTML document the user can preview instantly or download — no server build
// required. It is a pure function (easy to test) and escapes every piece of
// user content so a component's text can never inject markup.

function escapeHTML(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function prop(comp: VisualComponent, key: string, fallback: string): string {
  const raw = comp.props?.[key];
  return escapeHTML(raw !== undefined && raw !== "" ? raw : fallback);
}

function renderComponent(comp: VisualComponent): string {
  switch (comp.type) {
    case "heading":
      return `<h2 class="c-heading">${prop(comp, "text", "Título")}</h2>`;
    case "paragraph":
      return `<p class="c-paragraph">${prop(comp, "text", "Texto do parágrafo")}</p>`;
    case "button":
      return `<button type="button" class="c-button">${prop(comp, "label", "Botão")}</button>`;
    case "card":
      return `<div class="c-card"><h3>${prop(comp, "title", "Título do Card")}</h3><p>${prop(comp, "body", "Corpo do card explicativo.")}</p></div>`;
    case "metric":
      return `<div class="c-metric"><div class="c-metric-label">${prop(comp, "label", "Métrica")}</div><div class="c-metric-value">${prop(comp, "value", "0")}</div></div>`;
    case "navbar": {
      const brand = prop(comp, "brand", "Brand");
      const links = (comp.props?.links || "Link 1, Link 2")
        .split(",")
        .map((l) => `<span>${escapeHTML(l.trim())}</span>`)
        .join("");
      return `<nav class="c-navbar"><div class="c-brand">${brand}</div><div class="c-links">${links}</div></nav>`;
    }
    case "image": {
      const src = comp.props?.src || "";
      const alt = prop(comp, "alt", "");
      // Only allow http(s)/relative image sources; block script-bearing schemes.
      const safeSrc = /^(https?:)?\/\//i.test(src) || src.startsWith("/") ? escapeHTML(src) : "";
      return safeSrc ? `<img class="c-image" src="${safeSrc}" alt="${alt}" />` : "";
    }
    case "hero":
      return `<section class="c-hero"><h1>${prop(comp, "title", "Título do Hero")}</h1><p>${prop(comp, "subtitle", "Subtítulo explicativo")}</p><button type="button" class="c-button">${prop(comp, "cta", "Ação")}</button></section>`;
    case "input":
      return `<label class="c-field"><span>${prop(comp, "label", "Rótulo")}</span><input type="text" placeholder="${prop(comp, "placeholder", "")}" /></label>`;
    case "link": {
      const href = comp.props?.href || "#";
      // Only http(s)/relative/anchor hrefs; block javascript:/data: schemes.
      const safeHref = /^(https?:)?\/\//i.test(href) || href.startsWith("/") || href.startsWith("#") ? escapeHTML(href) : "#";
      return `<a class="c-link" href="${safeHref}">${prop(comp, "text", "Link")}</a>`;
    }
    case "list": {
      const items = (comp.props?.items || "Item 1, Item 2")
        .split(",")
        .map((it) => `<li>${escapeHTML(it.trim())}</li>`)
        .join("");
      return `<ul class="c-list">${items}</ul>`;
    }
    case "divider":
      return `<hr class="c-divider" />`;
    case "pricing": {
      const features = (comp.props?.features || "Recurso 1, Recurso 2, Recurso 3")
        .split(",")
        .map((f) => `<li>${escapeHTML(f.trim())}</li>`)
        .join("");
      return `<div class="c-pricing"><div class="c-pricing-name">${prop(comp, "plan", "Plano Pro")}</div><div class="c-pricing-price">${prop(comp, "price", "R$ 49/mês")}</div><ul class="c-pricing-features">${features}</ul><button type="button" class="c-button">${prop(comp, "cta", "Assinar")}</button></div>`;
    }
    case "testimonial":
      return `<figure class="c-testimonial"><blockquote>${prop(comp, "quote", "Mudou completamente o nosso fluxo de trabalho.")}</blockquote><figcaption>${prop(comp, "author", "Cliente satisfeito")}</figcaption></figure>`;
    case "faq":
      return `<details class="c-faq"><summary>${prop(comp, "question", "Pergunta frequente?")}</summary><p>${prop(comp, "answer", "Resposta clara e objetiva para a dúvida.")}</p></details>`;
    case "footer": {
      const links = (comp.props?.links || "Sobre, Contato, Privacidade")
        .split(",")
        .map((l) => `<span>${escapeHTML(l.trim())}</span>`)
        .join("");
      return `<footer class="c-footer"><span>${prop(comp, "text", "© 2026 Minha Empresa")}</span><div class="c-footer-links">${links}</div></footer>`;
    }
    default:
      // Unknown component: render its text prop if any, never raw markup.
      return comp.props?.text ? `<div class="c-unknown">${escapeHTML(comp.props.text)}</div>` : "";
  }
}

const BASE_CSS = `
  :root { color-scheme: light dark; }
  * { box-sizing: border-box; }
  body { margin: 0; font-family: system-ui, -apple-system, Segoe UI, Roboto, sans-serif; color: #111827; background: #ffffff; }
  .page { max-width: 960px; margin: 0 auto; padding: 24px; display: flex; flex-direction: column; gap: 16px; }
  .c-heading { font-size: 1.5rem; font-weight: 700; margin: 0; }
  .c-paragraph { font-size: .95rem; color: #4b5563; margin: 0; }
  .c-button { display: inline-flex; width: fit-content; border: 0; border-radius: 8px; background: #111827; color: #fff; padding: 10px 16px; font-weight: 600; font-size: .85rem; cursor: pointer; }
  .c-card { border: 1px solid #e5e7eb; border-radius: 10px; background: #f9fafb; padding: 14px; }
  .c-card h3 { margin: 0; font-size: 1rem; }
  .c-card p { margin: 6px 0 0; font-size: .8rem; color: #6b7280; }
  .c-metric { border-radius: 10px; background: #f3f4f6; padding: 14px; }
  .c-metric-label { font-size: .7rem; text-transform: uppercase; letter-spacing: .05em; color: #9ca3af; }
  .c-metric-value { margin-top: 4px; font-size: 1.6rem; font-weight: 700; }
  .c-navbar { display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid #e5e7eb; padding-bottom: 10px; }
  .c-brand { font-weight: 700; }
  .c-links { display: flex; gap: 14px; font-size: .8rem; color: #6b7280; }
  .c-image { max-width: 100%; border-radius: 8px; }
  .c-hero { border-radius: 14px; padding: 40px 24px; text-align: center; background: linear-gradient(135deg, #111827, #374151); color: #fff; }
  .c-hero h1 { margin: 0; font-size: 2rem; }
  .c-hero p { margin: 10px 0 20px; opacity: .85; }
  .c-hero .c-button { background: #fff; color: #111827; }
  .c-field { display: block; font-size: .85rem; }
  .c-field span { display: block; margin-bottom: 4px; color: #374151; }
  .c-field input { width: 100%; border: 1px solid #d1d5db; border-radius: 8px; padding: 8px 12px; font-size: .9rem; }
  .c-link { color: #2563eb; font-weight: 600; text-decoration: underline; }
  .c-list { padding-left: 20px; color: #374151; display: flex; flex-direction: column; gap: 4px; }
  .c-divider { border: 0; border-top: 1px solid #e5e7eb; margin: 8px 0; }
  .c-pricing { border: 1px solid #e5e7eb; border-radius: 12px; padding: 20px; text-align: center; background: #f9fafb; max-width: 280px; }
  .c-pricing-name { font-weight: 700; font-size: 1.05rem; }
  .c-pricing-price { font-size: 1.8rem; font-weight: 800; margin: 6px 0 12px; }
  .c-pricing-features { list-style: none; padding: 0; margin: 0 0 14px; display: flex; flex-direction: column; gap: 6px; font-size: .85rem; color: #4b5563; }
  .c-testimonial { margin: 0; border-left: 3px solid #111827; padding: 6px 0 6px 16px; }
  .c-testimonial blockquote { margin: 0; font-size: 1rem; font-style: italic; }
  .c-testimonial figcaption { margin-top: 8px; font-size: .8rem; color: #6b7280; font-weight: 600; }
  .c-faq { border: 1px solid #e5e7eb; border-radius: 8px; padding: 10px 14px; }
  .c-faq summary { cursor: pointer; font-weight: 600; font-size: .9rem; }
  .c-faq p { margin: 8px 0 0; font-size: .85rem; color: #4b5563; }
  .c-footer { display: flex; align-items: center; justify-content: space-between; border-top: 1px solid #e5e7eb; padding-top: 14px; margin-top: 8px; font-size: .8rem; color: #6b7280; }
  .c-footer-links { display: flex; gap: 14px; }
  @media (prefers-color-scheme: dark) { body { background: #0a0a0a; color: #f3f4f6; } .c-paragraph { color: #d1d5db; } .c-card { background: #171717; border-color: #262626; } .c-metric { background: #171717; } .c-pricing { background: #171717; border-color: #262626; } .c-testimonial { border-left-color: #f3f4f6; } .c-faq { border-color: #262626; } }
`;

export function generateStudioHTML(project: BuilderProject): string {
  const components = project.components ?? [];
  const title = escapeHTML(project.name || "Meu app");
  const body = components.map(renderComponent).join("\n      ");
  const content = body.trim() || `<p class="c-paragraph">Adicione componentes no Studio para ver seu app aqui.</p>`;
  return `<!doctype html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${title}</title>
  <style>${BASE_CSS}</style>
</head>
<body>
  <main class="page">
      ${content}
  </main>
</body>
</html>`;
}
