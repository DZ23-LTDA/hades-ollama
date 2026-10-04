import type { VisualComponent } from "@/lib/agenticClient";

// AI-assisted Studio generation. The model is asked for a strict JSON list of
// components; parseGeneratedComponents turns that (possibly messy) text into a
// safe, known-only component array. Kept pure so the parsing contract — which
// must never trust raw model output — is fully unit-tested.

// The component types Studio can render (palette + export engine). Anything the
// model invents outside this set is dropped rather than rendered blindly.
export const KNOWN_STUDIO_TYPES = [
  "heading",
  "paragraph",
  "button",
  "card",
  "metric",
  "navbar",
  "image",
  "hero",
  "input",
  "link",
  "list",
  "divider",
  "pricing",
  "testimonial",
  "faq",
  "footer",
] as const;

const KNOWN = new Set<string>(KNOWN_STUDIO_TYPES);
const MAX_COMPONENTS = 40;
const MAX_PROP_LEN = 2000;

// The main editable text prop per component type, used by the Studio canvas for
// double-click inline editing. A subset of KNOWN_STUDIO_TYPES (divider has no
// text to edit inline). Co-located here so it stays consistent with the type
// list and is unit-testable.
export const PRIMARY_TEXT_PROP: Record<string, string> = {
  heading: "text",
  paragraph: "text",
  button: "label",
  card: "title",
  metric: "value",
  navbar: "brand",
  hero: "title",
  image: "alt",
  input: "label",
  link: "text",
  list: "items",
  pricing: "plan",
  testimonial: "quote",
  faq: "question",
  footer: "text",
};

// pickLocalFirstModel chooses which model to use for Studio AI generation,
// preferring a model that runs locally (not a ":cloud" one) to honor Hades's
// local-first positioning, falling back to the first available model.
export function pickLocalFirstModel(names: string[]): string | undefined {
  return names.find((n) => typeof n === "string" && n !== "" && !n.endsWith(":cloud")) ?? names.find((n) => typeof n === "string" && n !== "");
}

export function buildGenerationPrompt(description: string): string {
  return [
    "Você é um gerador de layout. A partir da descrição do usuário, responda APENAS",
    "com um JSON (sem texto fora do JSON) no formato:",
    '{"components": [{"type": "<tipo>", "props": {"<chave>": "<valor>"}}]}',
    "",
    `Tipos permitidos: ${KNOWN_STUDIO_TYPES.join(", ")}.`,
    "Props comuns por tipo: heading{text}; paragraph{text}; button{label};",
    "card{title,body}; metric{label,value}; navbar{brand,links}; hero{title,subtitle,cta};",
    "image{src,alt}; input{label,placeholder}; link{text,href}; list{items};",
    "pricing{plan,price,features,cta}; testimonial{quote,author}; faq{question,answer};",
    "footer{text,links}. Listas (links, items, features) são texto separado por vírgula.",
    "Gere uma página coerente e completa (ex.: navbar, hero, cards, pricing, footer).",
    "",
    `Descrição do usuário: ${description}`,
  ].join("\n");
}

function coerceProps(raw: unknown): Record<string, string> {
  const out: Record<string, string> = {};
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return out;
  for (const [key, value] of Object.entries(raw as Record<string, unknown>)) {
    if (typeof key !== "string" || !key) continue;
    let text: string;
    if (typeof value === "string") text = value;
    else if (typeof value === "number" || typeof value === "boolean") text = String(value);
    else if (Array.isArray(value)) text = value.map((v) => String(v)).join(", ");
    else continue;
    out[key] = text.slice(0, MAX_PROP_LEN);
  }
  return out;
}

/**
 * Parses a model response into a safe list of known Studio components. Tolerates
 * markdown code fences and both `[...]` and `{"components": [...]}` shapes.
 * Returns an empty array when nothing usable is found — never throws.
 */
export function parseGeneratedComponents(text: string, idPrefix = "ai"): VisualComponent[] {
  if (typeof text !== "string" || text.trim() === "") return [];
  let body = text.trim();

  // Strip a ```json ... ``` (or plain ```) fence if present.
  const fence = body.match(/```(?:json)?\s*([\s\S]*?)```/i);
  if (fence) body = fence[1].trim();

  // Fall back to the first {...} or [...] span if there is surrounding prose.
  if (!(body.startsWith("{") || body.startsWith("["))) {
    const span = body.match(/[[{][\s\S]*[\]}]/);
    if (span) body = span[0];
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(body);
  } catch {
    return [];
  }

  let list: unknown[];
  if (Array.isArray(parsed)) {
    list = parsed;
  } else if (parsed && typeof parsed === "object" && Array.isArray((parsed as { components?: unknown[] }).components)) {
    list = (parsed as { components: unknown[] }).components;
  } else {
    return [];
  }

  const components: VisualComponent[] = [];
  for (const item of list) {
    if (components.length >= MAX_COMPONENTS) break;
    if (!item || typeof item !== "object") continue;
    const type = (item as { type?: unknown }).type;
    if (typeof type !== "string" || !KNOWN.has(type)) continue;
    components.push({
      id: `${idPrefix}_${components.length + 1}_${type}`,
      type,
      props: coerceProps((item as { props?: unknown }).props),
    });
  }
  return components;
}
