# Learning platform — project documentation

Persistent documentation for evolving the **Learnings** app (`apps/learning/`) into a
structured, adaptive, AI-assisted language learning platform. It is written so that a
coding agent (or a person) can continue the work **without the original conversation**.

> Location: this tree lives in `docs/learning/`, not in a bare `docs/`, because the
> repository is a monorepo (budget + learning) and a root `docs/` would be ambiguous.
> It is outside `apps/learning/**`, so a docs-only push does not match the CI path
> filter and does not trigger a production deploy. See `DECISIONS.md` D-001.

## Read in this order

| # | File | What it answers |
| --- | --- | --- |
| 1 | [`CURRENT_STATE.md`](CURRENT_STATE.md) | What exists today (audit of 2026-11-25), baseline test results, tech debt, risks |
| 2 | [`PRODUCT_VISION.md`](PRODUCT_VISION.md) | What we are building and the principles that decide trade-offs |
| 3 | [`IMPLEMENTATION_PLAN.md`](IMPLEMENTATION_PLAN.md) | Phases 0-10, gap analysis, prioritized backlog |
| 4 | [`IMPLEMENTATION_STATUS.md`](IMPLEMENTATION_STATUS.md) | Live status of every phase and task, session log — **update after every phase** |
| 4a | [`BACKEND_BRIEF_FIT.md`](BACKEND_BRIEF_FIT.md) | How the owner's second ("Backend Implementation") brief fits the code and this plan; additions A-1..A-11 and the proposed loop-first order |
| 4b | [`VERTICAL_SLICE_V1.md`](VERTICAL_SLICE_V1.md) | The first end-to-end learning loop (D-024): design, rules, API, tests, limits |
| 5 | [`DECISIONS.md`](DECISIONS.md) | Decision log; section "Awaiting approval" lists what blocks which phase |
| 6 | [`ARCHITECTURE.md`](ARCHITECTURE.md) | Real code conventions today + target domain boundaries |
| 7 | [`DATA_MODEL.md`](DATA_MODEL.md) | Current schema (migrations 0001-0035) and the proposed target schema |
| 8 | [`API_CONTRACTS.md`](API_CONTRACTS.md) | API conventions, endpoint index, compatibility rules, planned endpoints |
| 9 | [`AI_CONTRACTS.md`](AI_CONTRACTS.md) | Every AI call today, its validation, and the target contract rules |
| 10 | [`KNOWLEDGE_BASE.md`](KNOWLEDGE_BASE.md) | Current library, target knowledge-item model, authoring and versioning |
| 11 | [`LEARNING_METHODOLOGY.md`](LEARNING_METHODOLOGY.md) | How each pedagogical approach becomes a concrete system rule |
| 12 | [`ASSESSMENT_ENGINE.md`](ASSESSMENT_ENGINE.md) | Placement/final tests today, evidence-based assessment target |
| 13 | [`LEARNING_ENGINE.md`](LEARNING_ENGINE.md) | Path generation, adaptation, review scheduling, sessions |
| 14 | [`PROGRESS_MODEL.md`](PROGRESS_MODEL.md) | Mastery states, evidence, skill estimates, analytics definitions |
| 15 | [`TESTING_STRATEGY.md`](TESTING_STRATEGY.md) | How to run the gates, what exists, what each phase must add |

## Existing documentation (kept, not duplicated)

These stay the source of truth for what they cover; the files above link to them instead of
copying them:

| File | Covers |
| --- | --- |
| [`apps/learning/backend/docs/learning-api.md`](../../apps/learning/backend/docs/learning-api.md) | **Canonical API reference** of every existing endpoint (notebook, capture queue, AI agent, chats, scan, teachers/program/exercises, rule book) |
| [`apps/learning/README.md`](../../apps/learning/README.md) | Local setup, production pointers |
| [`infra/learning/README.md`](../../infra/learning/README.md) | Compose, edge Caddy, secrets, backups, rebuild |
| [`infra/budget/CI_CD.md`](../../infra/budget/CI_CD.md) | CI/CD and blue-green deploy (shared by both apps) |
| [`libs/account/README.md`](../../libs/account/README.md) | Shared account (login, profile, deletion) |

`apps/learning/backend/CODING_STYLE.md` is a **stale generic template** (hexagonal layout,
`gin`, `wire`, MySQL) that does not describe this code. `ARCHITECTURE.md` documents the real
conventions; see `DECISIONS.md` D-004 for what to do with the stale file.

## Conventions for these documents

- **Status vocabulary** (phases and tasks): `NOT_STARTED`, `IN_PROGRESS`, `BLOCKED`,
  `IMPLEMENTED`, `VERIFIED`. `VERIFIED` means the relevant tests were run and passed;
  the run (command, date, result) is recorded in `IMPLEMENTATION_STATUS.md`. Never set
  it without that record.
- **Provenance tags** in `CURRENT_STATE.md` / `DATA_MODEL.md`: statements are facts read from
  code at commit `4662ad7` unless marked `(unverified)`.
- **IDs**: phases `P0`..`P10`; tasks `P<phase>-NN`; decisions `D-NNN`; risks `R-NN`;
  gaps `G-NN`. Reference them from commits and from `IMPLEMENTATION_STATUS.md`.
- **Language**: documents are English (like the other repo docs). The product UI is
  ru / en / es.
- When behaviour changes, update the matching document in the **same commit**. If two
  documents disagree, the code wins, then `learning-api.md`; fix the document.
- No secrets, no personal data, no production hostnames beyond those already in the
  repository's README files. The repository is publicly readable.

## Agent instruction file

The repository has **no** `AGENTS.md` / `CLAUDE.md` (the local `.claude/` directory is
git-ignored). Nothing was overwritten. Whoever adds one should reference this README and
`ARCHITECTURE.md` section "Conventions" (see `DECISIONS.md` D-005).
