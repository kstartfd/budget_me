# Repository guidance for coding agents

This repository contains two applications. For Learnings work, start with
[`docs/PROJECT_CONTEXT.md`](docs/PROJECT_CONTEXT.md) and the document index in
[`docs/learning/README.md`](docs/learning/README.md). Use
[`docs/learning/ARCHITECTURE.md`](docs/learning/ARCHITECTURE.md) section 2 for the
actual code conventions. `apps/learning/backend/CODING_STYLE.md` is an outdated
template and is not authoritative.

- A push to `main` can deploy the changed application to production. Check the
  affected CI path filters and run the relevant gates before pushing.
- Database changes must be additive and compatible with both application versions
  during a blue-green deploy. Preserve existing user data; use a reversible
  migration and a staged expand/backfill/switch/contract sequence when needed.
- Name every new Learning-owned database table with a descriptive `learning_`
  prefix (for example, `learning_knowledge_items`). Do not copy the provisional
  `kb_*`, `learner_*`, or `assessment_*` table names from planning documents into
  new migrations. Migration 0038 keeps `kb_items` as a compatibility view over
  `learning_knowledge_items`; do not remove that view while older application
  versions may run. `learner_item_state` remains a shipped naming exception.
  The shared account schema and existing `users` table are outside this rule.
- Never commit secrets, API keys, tokens, personal data, or production dumps.
- Keep `apps/budget/` outside the scope of Learnings work unless the task explicitly
  requires a shared change.
- When behavior changes, update the canonical API reference and the relevant
  learning documents in the same change. `docs/learning/IMPLEMENTATION_STATUS.md`
  is the status ledger; only mark work `VERIFIED` with a recorded gate result.

The user's current instructions take precedence over this file.
