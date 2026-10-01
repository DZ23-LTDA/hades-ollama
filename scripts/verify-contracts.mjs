#!/usr/bin/env node
/** Frontend/backend route contract verifier. Fails on method or path mismatches. */
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const backendFiles = [
  "server/agent_routes.go", "server/routes.go", "server/desktop_local_routes.go",
  "server/plugin_routes.go", "server/company_routes.go", "server/company_growth_routes.go",
  "server/company_approval_routes.go", "server/whatsapp_routes.go", "app/ui/ui.go",
];
const backendRoutes = [];
const addRoute = (method, route, file, prefix = "") => {
  let p = route.replace(/\{([^}]+)\}/g, ":$1");
  if (prefix && !p.startsWith("/api/")) p = prefix + (p.startsWith("/") ? p : `/${p}`);
  backendRoutes.push({ method: method.toUpperCase(), path: p, file });
};
for (const rel of backendFiles) {
  const full = path.join(repoRoot, rel);
  if (!fs.existsSync(full)) continue;
  for (const line of fs.readFileSync(full, "utf8").split("\n")) {
    const s = line.trim();
    if (s.startsWith("//")) continue;
    let m = s.match(/(?:group|r)\.(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD|Any)\(["']([^"']+)["']/);
    if (m) addRoute(m[1] === "Any" ? "ALL" : m[1], m[2], rel, s.includes("group.") ? "/api/agent/v1" : "");
    m = s.match(/mux\.Handle\(["'](?:(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\s+)?([^"']+)["']/);
    if (m) addRoute(m[1] || "ALL", m[2], rel);
  }
}

function files(dir) {
  const out = [];
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) out.push(...files(p));
    else if ([".ts", ".tsx", ".js", ".jsx"].includes(path.extname(e.name))) out.push(p);
  }
  return out;
}
function methodNear(content, index, fallback = "GET") {
  const lineEnd = content.indexOf("\n", index);
  const firstLine = content.slice(index, lineEnd < 0 ? content.length : lineEnd);
  const closes = [content.indexOf("});", index), content.indexOf(");", index)]
    .filter((value) => value >= 0 && value - index < 500);
  const close = closes.length ? Math.min(...closes) : -1;
  const snippet = close >= 0 ? content.slice(index, close) : firstLine;
  const m = snippet.match(/method\s*:\s*["'](GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)["']/i);
  return (m?.[1] || fallback).toUpperCase();
}
function collectExpression(raw, file, method, kind, calls) {
  let p = raw.trim();
  if (p.startsWith("http://") || p.startsWith("https://")) return;
  if (kind === "agentFetch" && !p.startsWith("/api/")) p = `/api/agent/v1${p.startsWith("/") ? p : `/${p}`}`;
  calls.push({ raw, path: stripQuery(p).replace(/\/+$/, ""), method, file, kind });
}
function stripQuery(value) {
  let expressionDepth = 0;
  for (let i = 0; i < value.length; i += 1) {
    if (value[i] === "$" && value[i + 1] === "{") expressionDepth += 1;
    if (value[i] === "}" && expressionDepth > 0) expressionDepth -= 1;
    if (value[i] === "?" && expressionDepth === 0) return value.slice(0, i);
  }
  return value;
}
const frontendCalls = [];
for (const full of files(path.join(repoRoot, "app/ui/app/src"))) {
  const content = fs.readFileSync(full, "utf8");
  const file = path.relative(repoRoot, full);
  if (/\.(test|spec)\.[jt]sx?$/.test(file)) continue;
  let m;
  const fetchRe = /fetch\(\s*(?:`(?:\$\{[^}]+\})?(\/api\/[^`]+)`|["'](\/api\/[^"']+)["'])/g;
  while ((m = fetchRe.exec(content))) collectExpression(m[1] || m[2], file, methodNear(content, m.index), "fetch", frontendCalls);
  const agentRe = /agentFetch(?:<[^>]*>)?\(\s*(?:`([^`]+)`|["']([^"']+)["'])/g;
  while ((m = agentRe.exec(content))) collectExpression(m[1] || m[2], file, methodNear(content, m.index), "agentFetch", frontendCalls);
  const apiRe = /\bapi\(\s*(?:`([^`]+)`|["']([^"']+)["'])/g;
  while ((m = apiRe.exec(content))) collectExpression(m[1] || m[2], file, methodNear(content, m.index), "api", frontendCalls);
  const eventRe = /new\s+EventSource\(\s*(?:`(?:\$\{[^}]+\})?(\/api\/[^`]+)`|["'](\/api\/[^"']+)["'])/g;
  while ((m = eventRe.exec(content))) collectExpression(m[1] || m[2], file, "GET", "EventSource", frontendCalls);
  const hrefRe = /href\s*=\s*(?:`([^`]*\/api\/[^`]*)`|["']([^"']*\/api\/[^"']*)["'])/g;
  while ((m = hrefRe.exec(content))) collectExpression(m[1] || m[2], file, "GET", "href", frontendCalls);
}

function normalizedPath(raw) {
  // Keep dynamic template segments distinct from literals: backend :params may only
  // match a frontend expression, never an arbitrary literal in the same position.
  return raw.replace(/\$\{[^}]+\}/g, "__PARAM__").split("?")[0].replace(/\/+$/, "") || "/";
}
function routeMatches(frontRaw, backendRaw) {
  const f = stripQuery(frontRaw).replace(/\/+$/, "").split("/").filter(Boolean);
  const b = stripQuery(backendRaw).replace(/\/+$/, "").split("/").filter(Boolean);
  if (b.length && b.at(-1).startsWith("*")) {
    if (f.length < b.length - 1) return false;
  } else if (f.length !== b.length) return false;
  for (let i = 0; i < b.length; i++) {
    if (b[i]?.startsWith("*")) break;
    if (b[i]?.startsWith(":")) {
      if (!f[i]?.includes("${") && f[i] !== "__PARAM__") return false;
    } else if (f[i] !== b[i] && !(f[i]?.includes("${") && f[i].includes(b[i]))) return false;
  }
  return true;
}
function matches(call, route) {
  return (route.method === "ALL" || route.method === call.method) && routeMatches(call.raw, route.path);
}

const uniqueCalls = [...new Map(frontendCalls.map(c => [`${c.method} ${c.path} ${c.file}`, c])).values()];
console.log("==========================================================");
console.log("    FRONTEND <-> BACKEND CONTRACT MATRIX VERIFICATION     ");
console.log("==========================================================");
console.log(`Backend routes discovered: ${backendRoutes.length}`);
console.log(`Frontend API calls scanned: ${uniqueCalls.length}`);
let failed = false;
const matchedRoutes = new Set();
for (const call of uniqueCalls) {
  const route = backendRoutes.find(r => matches(call, r));
  if (!route) {
    console.error(`[FAIL] ${call.method} ${call.path} -> NO MATCH (${call.kind}, ${call.file})`);
    failed = true;
  } else {
    matchedRoutes.add(backendRoutes.indexOf(route));
    console.log(`[PASS] ${call.method} ${call.path} -> ${route.method} ${route.path} (${route.file})`);
  }
}
const orphanRoutes = backendRoutes.filter((_, i) => !matchedRoutes.has(i));
console.warn(`CONTRACT WARNING: ${orphanRoutes.length} backend routes have no static frontend caller.`);
for (const r of orphanRoutes.slice(0, 20)) console.warn(`[ORPHAN] ${r.method} ${r.path} (${r.file})`);
console.log("==========================================================");
if (failed) {
  console.error("CONTRACT VERIFICATION FAILED: frontend method/path does not match backend.");
  process.exit(1);
}
console.log("CONTRACT VERIFICATION PASSED: all frontend calls match method and route shape.");
