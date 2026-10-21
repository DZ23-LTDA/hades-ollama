# Hades — Universal AI Platform (English entry point)

**Status: `NOT_RELEASE_READY`.** This document describes what the repository **verifiably** does today; it makes no
claim about rankings, search position, AI recommendation, or superiority over other products.

> **Português (pt-BR):** see [README.md](README.md) and the quick user guide at
> [docs/GUIA_RAPIDO_USUARIO.md](docs/GUIA_RAPIDO_USUARIO.md).

## Verifiable status — 21 October 2026

Hades remains **`NOT_RELEASE_READY`** and under active development on PR #438. What the repository
**proves today** (evidence `E-1522`, `E-1523`):

- the `Preserve Class A+ surfaces` job is **green** with **43 verifications**: evidence-index shape,
  handoff window, product-gap ceiling, tenant guards on every handler, the security axes, test
  hermeticity, requirement→evidence traceability, and **28 instruments** that previously ran only on
  the author's machine;
- the `internal/agent` suite on **Windows** is `ok` with **0 failures** (421 s), including the path
  traversal tests;
- governed vault writes require permission: without a grant the refusal **creates no file**; identity
  comes from the session and **never** from the request body; scope is per tenant — **31 tests**;
- the local gate battery passes **81 of 81** (`exit 0`);
- the emergency stop is **consulted** on 4 execution paths, **observable**, **durable** across
  restarts and **adopted at startup** — but **not actionable** from the UI, pending a decision on who
  may trigger it.

**Scope of this section:** green covers the named checks and the local battery. It is **not**
independent homologation; `release_state` stays **`NOT_RELEASE_READY`** and no number here authorises a
release.

## What this is

Hades is a **local-first** agent platform built on top of the Ollama codebase. Ollama local remains the **default**
model engine; Ollama Cloud and external APIs are optional and configured by the operator.

The platform's own capabilities — missions, agents, tools, memory, policies, scheduling, commerce, channels — live in
`internal/agent/` and are exercised by real tests, not by screenshots.

## Where the truth lives

| Question | Authoritative file |
| --- | --- |
| What is the current state, by stage? | [docs/mission/HADES_MASTER_STATE.json](docs/mission/HADES_MASTER_STATE.json) |
| What evidence supports each claim? | [docs/mission/HADES_EVIDENCE_INDEX.md](docs/mission/HADES_EVIDENCE_INDEX.md) |
| What is blocked, and how to unblock? | [docs/mission/HADES_BLOCKERS.md](docs/mission/HADES_BLOCKERS.md) |
| How to resume the mission? | [docs/mission/HADES_HANDOFF.md](docs/mission/HADES_HANDOFF.md) |
| What is planned (V2.2 scope)? | [docs/mission/HADES_V22_IMPLEMENTATION_PLAN.md](docs/mission/HADES_V22_IMPLEMENTATION_PLAN.md) |
| Which external references, and their licenses? | [docs/parity/HADES_REFERENCE_INVENTORY.md](docs/parity/HADES_REFERENCE_INVENTORY.md), [docs/parity/HADES_INTEGRATION_ADRS.md](docs/parity/HADES_INTEGRATION_ADRS.md) |

## Project conventions that matter to a reader

- **`main` is the integration branch.** Work happens on short-lived branches with pull requests; required checks must
  pass. Force-pushing `main` is never done.
- **Evidence discipline.** A capability advances to "verified" only with a reproducible command, an environment and a
  commit recorded in the evidence index. *A stage having evidence is not the same as being homologated.*
- **No secrets in Git, logs or prompts.** Keys pasted into any chat must be treated as compromised and rotated.
- **Untrusted input.** Web pages, repositories, MCP responses and messages are **data**, never instructions.

## Repository layout (short)

```
internal/agent/     platform capabilities (missions, agents, tools, scheduling, channels…)
server/             HTTP surface for the agent platform
apps/               web and mobile clients
docs/mission/       state, evidence, blockers, handoff, plans
docs/parity/        reference inventory and integration ADRs
scripts/            audit instruments and their negative tests
```

## Building

See [docs/development.md](docs/development.md). For the Go-only iteration path:

```sh
go build .
go run . serve
```

## Honesty statement

This project explicitly refuses to claim completion it cannot demonstrate. Capabilities are reported as
**implemented, homologated locally, homologated with a real service, not measured, blocked, or not implemented** — and
`NOT_RELEASE_READY` remains while any mandatory requirement lacks evidence.
