# QWEN.md — HelixQA LLMProvider

| Field | Value |
|---|---|
| Revision | 1 |
| Created | 2026-05-23 |
| Last modified | 2026-05-23 |
| Status | active |
| Status summary | Created per Phase 39.IT (User mandate 2026-05-23) — propagation of QWEN.md across the consumer fleet, mirroring CLAUDE.md + AGENTS.md per §11.4.35 canonical-root inheritance. |
| Issues | none |
| Continuation | — |

## INHERITED FROM constitution/QWEN.md

All rules in `constitution/QWEN.md` (and the `constitution/Constitution.md`
it references) apply unconditionally. This module's rules below extend them —
they do NOT weaken any universal clause. When this file disagrees with the
constitution submodule, the constitution wins. Locate the constitution
submodule from any arbitrary nested depth using its `find_constitution.sh`
helper.

The universal anti-bluff covenant (§11.4), no-guessing mandate (§11.4.6),
credentials-handling mandate (§11.4.10), host-session safety (§12 + §12.6 +
§12.10), and data safety (§9) all live in `constitution/Constitution.md`.
Read it before working on any non-trivial change.

@constitution/QWEN.md

Canonical reference: <https://github.com/HelixDevelopment/HelixConstitution>

## How this file relates to CLAUDE.md + AGENTS.md

Per §11.4.35 canonical-root inheritance clarity:

- `constitution/QWEN.md` is the universal canonical root for the Qwen Code CLI.
- This file is the consumer-side extension for this submodule, carrying only
  the inheritance pointer + the §11.4 covenant anchors + a brief module summary.
- The full module ruleset lives in this submodule's sibling `CLAUDE.md`.
  Qwen Code agents MUST read CLAUDE.md before performing any work; this
  QWEN.md is the Qwen-specific entry point that ensures Qwen reads the
  inheritance pointer + anti-bluff covenant on every session.

## Module summary

LLM provider abstraction layer for HelixQA. Wraps OpenAI, Anthropic, Gemini, and local-model backends behind a uniform API consumed by LLMOrchestrator.

For full module context (build steps, integration points, host-session safety,
submodule-specific commit/push discipline) read this directory's `CLAUDE.md`
and `AGENTS.md`.
## Companion documents

| File | Role |
|---|---|
| `CLAUDE.md` | Full module ruleset (Claude Code primary context) |
| `AGENTS.md` | Cross-agent mirror (OpenCode, Cursor, Aider, generic AI tooling) |
| `QWEN.md` (this file) | Qwen Code CLI entry point |
| `../../../../constitution/Constitution.md` | Universal canonical rules |
| `../../../../constitution/QWEN.md` | Universal Qwen entry point |
