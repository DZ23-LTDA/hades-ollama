#!/usr/bin/env node

/**
 * Frontend <-> Backend Contract Matrix Verifier
 * 
 * Scans all frontend files in app/ui/app/src for API calls and validates that
 * each requested endpoint matches a registered backend route in Go server.
 * Fails with non-zero exit code if any unregistered endpoint is called.
 */

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, "..");

// 1. Collect Backend Routes
const backendFiles = [
  "server/agent_routes.go",
  "server/routes.go",
  "server/desktop_local_routes.go",
  "server/plugin_routes.go",
  "app/ui/ui.go",
];

const backendRoutePatterns = [];

for (const rel of backendFiles) {
  const full = path.join(repoRoot, rel);
  if (!fs.existsSync(full)) continue;
  const content = fs.readFileSync(full, "utf-8");
  const lines = content.split("\n");

  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.startsWith("//")) continue;

    // group.<METHOD>("...", ...) in agent_routes / plugin_routes
    const groupMatch = trimmed.match(/group\.(GET|POST|PUT|DELETE|PATCH)\(["']([^"']+)["']/);
    if (groupMatch) {
      const p = groupMatch[2];
      const fullPath = p.startsWith("/api/agent/v1")
        ? p
        : "/api/agent/v1" + (p.startsWith("/") ? p : "/" + p);
      backendRoutePatterns.push({
        method: groupMatch[1],
        path: fullPath,
        file: rel,
      });
    }

    // r.<METHOD>("...", ...) in routes / desktop_local_routes
    const rMatch = trimmed.match(/r\.(GET|POST|PUT|DELETE|PATCH)\(["']([^"']+)["']/);
    if (rMatch) {
      backendRoutePatterns.push({
        method: rMatch[1],
        path: rMatch[2],
        file: rel,
      });
    }

    // mux.Handle("<METHOD> <PATH>", ...) in app/ui/ui.go
    const muxMatch = trimmed.match(/mux\.Handle\(["'](?:(GET|POST|PUT|DELETE|PATCH)\s+)?([^"']+)["']/);
    if (muxMatch) {
      let p = muxMatch[2];
      // Convert {param} to :param
      p = p.replace(/\{([^}]+)\}/g, ":$1");
      backendRoutePatterns.push({
        method: muxMatch[1] || "ALL",
        path: p,
        file: rel,
      });
    }
  }
}

// 2. Helper to check if a frontend route matches a backend route pattern
function routeMatches(frontendPath, backendPattern) {
  // Normalize both by stripping query params and trailing slashes
  const cleanFront = frontendPath.split("?")[0].replace(/\/+$/, "");
  const cleanBack = backendPattern.split("?")[0].replace(/\/+$/, "");

  const fParts = cleanFront.split("/").filter(Boolean);
  const bParts = cleanBack.split("/").filter(Boolean);

  if (fParts.length !== bParts.length) {
    // Check wildcard match at the end (e.g. /*path)
    if (bParts.length > 0 && bParts[bParts.length - 1].startsWith("*")) {
      const prefix = bParts.slice(0, -1);
      if (fParts.length >= prefix.length) {
        for (let i = 0; i < prefix.length; i++) {
          if (prefix[i] !== fParts[i] && !prefix[i].startsWith(":")) {
            return false;
          }
        }
        return true;
      }
    }
    return false;
  }

  for (let i = 0; i < fParts.length; i++) {
    const fp = fParts[i];
    const bp = bParts[i];
    if (bp.startsWith(":") || bp.startsWith("*") || fp.startsWith(":")) {
      continue; // wildcard/param matches
    }
    if (fp !== bp) {
      return false;
    }
  }
  return true;
}

// 3. Scan Frontend Files in app/ui/app/src
const frontendDir = path.join(repoRoot, "app/ui/app/src");

function getAllFiles(dir, exts) {
  let files = [];
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files = files.concat(getAllFiles(full, exts));
    } else if (exts.includes(path.extname(entry.name))) {
      files.push(full);
    }
  }
  return files;
}

const frontendFiles = getAllFiles(frontendDir, [".ts", ".tsx", ".js", ".jsx"]);
const frontendCalls = [];

for (const file of frontendFiles) {
  const content = fs.readFileSync(file, "utf-8");
  const relPath = path.relative(repoRoot, file);

  // Pattern 1: fetch(`${API_BASE}/api/...`) or fetch("/api/...")
  const fetchRegex = /fetch\(\s*(?:`(?:\$\{API_BASE\}|\$\{DEV_API_URL\}|\$\{BASE_URL\})?(\/api\/[^`]+)`|["'](\/api\/[^"']+)["'])/g;
  let m;
  while ((m = fetchRegex.exec(content)) !== null) {
    const raw = m[1] || m[2];
    frontendCalls.push({ raw, file: relPath });
  }

  // Pattern 2: agentFetch<...>(`...`) or agentFetch(...)
  const agentFetchRegex = /agentFetch(?:<[^>]*>)?\(\s*(?:`([^`]+)`|["']([^"']+)["'])/g;
  while ((m = agentFetchRegex.exec(content)) !== null) {
    const raw = m[1] || m[2];
    frontendCalls.push({ raw, file: relPath, isAgentFetch: true });
  }

  // Pattern 3: EventSource or custom helpers with /api/
  const eventSourceRegex = /new\s+EventSource\(\s*(?:`(?:\$\{API_BASE\})?(\/api\/[^`]+)`|["'](\/api\/[^"']+)["'])/g;
  while ((m = eventSourceRegex.exec(content)) !== null) {
    const raw = m[1] || m[2];
    frontendCalls.push({ raw, file: relPath });
  }
}

// 4. Normalize Frontend Route Patterns
const normalizedCalls = [];
for (const call of frontendCalls) {
  let p = call.raw.trim();
  // Skip external URLs or non-API strings
  if (p.startsWith("http://") || p.startsWith("https://")) continue;

  if (call.isAgentFetch && !p.startsWith("/api/")) {
    p = "/api/agent/v1" + (p.startsWith("/") ? p : "/" + p);
  }

  // Normalize JS template literals to :param
  // e.g. ${encodeURIComponent(id)} -> :id, ${missionId} -> :id, ${id} -> :id
  p = p.replace(/\$\{[^}]+\}/g, ":id");
  // Clean query params
  p = p.split("?")[0].replace(/\/+$/, "");

  // Avoid duplicates per file
  const key = `${p} (${call.file})`;
  if (!normalizedCalls.some(c => c.key === key)) {
    normalizedCalls.push({
      path: p,
      file: call.file,
      raw: call.raw,
      key,
    });
  }
}

// 5. Validate Every Frontend Endpoint Against Backend Patterns
console.log("==========================================================");
console.log("    FRONTEND <-> BACKEND CONTRACT MATRIX VERIFICATION     ");
console.log("==========================================================");
console.log(`Backend routes discovered: ${backendRoutePatterns.length}`);
console.log(`Frontend API calls scanned: ${normalizedCalls.length}`);
console.log("----------------------------------------------------------");

let hasErrors = false;
const checked = new Set();

for (const call of normalizedCalls) {
  if (checked.has(call.path)) continue;
  checked.add(call.path);

  const matched = backendRoutePatterns.find(b => routeMatches(call.path, b.path));
  if (matched) {
    console.log(`[PASS] ${call.path.padEnd(45)} -> ${matched.path} (${matched.file})`);
  } else {
    console.error(`[FAIL] ${call.path.padEnd(45)} -> NO MATCH IN BACKEND ROUTES! (Called in: ${call.file})`);
    hasErrors = true;
  }
}

console.log("==========================================================");
if (hasErrors) {
  console.error("CONTRACT VERIFICATION FAILED: One or more frontend endpoints do not exist in the backend!");
  process.exit(1);
} else {
  console.log("CONTRACT VERIFICATION PASSED: All frontend endpoints exist in the backend!");
  process.exit(0);
}
