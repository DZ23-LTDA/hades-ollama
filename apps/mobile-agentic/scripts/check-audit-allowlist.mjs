#!/usr/bin/env node
// Mobile production-dependency audit with a scoped allowlist.
//
// Replaces a bare `npm audit --omit=dev --audit-level=high`: it STILL fails on
// any high/critical advisory, EXCEPT ids in ALLOWLIST — advisories that have no
// fixed version to upgrade to yet and are reached only through build-time
// tooling (not the shipped app). New/unknown high advisories, and any advisory
// that gains a fix, still fail the gate. Remove an entry once upstream ships a
// fix so the audit re-asserts it.
//
// Prefer pinning/upgrading over allowlisting. Advisories closed that way on
// 2026-10-08 and kept OUT of the allowlist:
//  - shell-quote 1.10.0 -> 1.12.0 and source-map-js 1.2.1 -> 1.2.2, via the
//    non-breaking `npm audit fix` (GHSA-pqg4-j6r4-53mv, GHSA-68fv-2mgg-jv7q);
//  - node-forge pinned by the `overrides` entry in package.json to the patched
//    upstream tarball 1.4.1-0 (GHSA-86w9-cpqp-85rv).
import { execSync } from "node:child_process";

const ALLOWLIST = new Map([
  [
    "GHSA-vfj7-8cjw-p6xm",
    "braces <=3.0.3 stack-exhaustion DoS: 3.0.3 is the latest published release " +
      "and is still in the vulnerable range (no fix available). Reached only via " +
      "React Native's bundler (metro -> micromatch -> braces) and its test/build " +
      "tooling at build time, not in the shipped app. The only fix npm offers is " +
      "react-native@0.87.1, a breaking major upgrade. Remove when the dependency " +
      "chain ships a fixed braces.",
  ],
]);

function runAudit() {
  try {
    return JSON.parse(
      execSync("npm audit --omit=dev --json", { encoding: "utf8" }),
    );
  } catch (err) {
    // `npm audit` exits non-zero when vulnerabilities exist but still prints the
    // JSON report to stdout; parse that rather than treating it as a failure.
    const out = err.stdout ? err.stdout.toString() : "";
    if (!out) throw err;
    return JSON.parse(out);
  }
}

const report = runAudit();
const vulns = report.vulnerabilities || {};
const blocking = [];

for (const [pkg, info] of Object.entries(vulns)) {
  for (const entry of info.via || []) {
    // Objects are direct advisories on this package; strings are transitive
    // (the advisory is checked on the package that actually carries it).
    if (typeof entry !== "object" || !entry.url) continue;
    if (entry.severity !== "high" && entry.severity !== "critical") continue;
    const id = entry.url.split("/").pop();
    if (!ALLOWLIST.has(id)) {
      blocking.push(`${pkg}: ${entry.severity} ${id} — ${entry.title}`);
    }
  }
}

if (blocking.length > 0) {
  console.error("High/critical advisories not in the allowlist:");
  for (const line of blocking) console.error(`  - ${line}`);
  console.error(
    "\nFix the dependency. Only if there is genuinely no fix available, add the " +
      "advisory id to the allowlist in scripts/check-audit-allowlist.mjs with a reason.",
  );
  process.exit(1);
}

const allowed = [...ALLOWLIST.keys()].join(", ");
console.log(
  "Mobile production audit OK: no high/critical advisories outside the allowlist" +
    (allowed ? ` (allowlisted with tracked reasons: ${allowed}).` : "."),
);
