# Testing strategy

## 1. How to run the gates (verified 2026-11-25)

Backend DB tests need **two** Postgres databases (the app DB and the shared account DB).
They skip silently when `DATABASE_URL` is unset (`testdb.Pool`), so an unconfigured run
looks green while testing nothing — always set both variables.

```sh
# 1. disposable databases (any free ports)
docker run -d --rm --name lp-app     -e POSTGRES_USER=ci -e POSTGRES_PASSWORD=ci -e POSTGRES_DB=app     -p 127.0.0.1:55441:5432 postgres:16-alpine
docker run -d --rm --name lp-account -e POSTGRES_USER=ci -e POSTGRES_PASSWORD=ci -e POSTGRES_DB=account -p 127.0.0.1:55442:5432 postgres:16-alpine

# 2. migrate + vet + test (this is exactly what CI runs)
cd apps/learning/backend
export DATABASE_URL='postgres://ci:ci@127.0.0.1:55441/app?sslmode=disable'
export ACCOUNT_DATABASE_URL='postgres://ci:ci@127.0.0.1:55442/account?sslmode=disable'
go run ./cmd/migrate up
go vet ./...
go test -count=1 ./...

# 3. frontend (pnpm lint can fail with ERR_PNPM_IGNORED_BUILDS locally; call the tools directly)
cd ../frontend
./node_modules/.bin/eslint && ./node_modules/.bin/tsc --noEmit
```

**Before every re-run on the same database:** `TRUNCATE learning_chat_usage` (or use fresh
containers). Otherwise 8 chat tests fail with 429/`42903` — a known hazard (TD-03), not a
regression. Phase 0 task **P0-05** removes it by cleaning the counters in the tests.

`tesseract` (+ `tesseract-ocr-eng`) is needed for 2 OCR tests; without it they `SKIP`.

Browser e2e (Playwright, 6 tests in `frontend/e2e`, not in CI): point `E2E_BACKEND_URL` at a
backend with a migrated **test** database, `E2E_FRONTEND_PORT` at `next dev`, optionally
`E2E_CHROMIUM_PATH` at a system Chrome (`playwright.config.ts`); sessions come from the
backend's `.learning-fixture` command (`ENV=test`, 1-hour token). The backend's
`LLMURLs` are never read from the environment (`config.go`), so real-provider calls cannot
happen by accident; a browser run that needs the agent requires a temporary fake-provider
server. Never run against production.

## 2. Baseline (2026-11-25, commit `4662ad7`)

262 passed, 0 failed, 2 skipped (+13 subtests) on fresh databases; `go vet`, `eslint`, `tsc`
clean. Per-package counts in `CURRENT_STATE.md` section 9. **Every phase must end with a run
at least this green plus its own new tests**, and records the numbers in
`IMPLEMENTATION_STATUS.md`.

## 3. Existing conventions to follow

- Table-driven unit tests for pure logic (`placement`, `NextReview`, `curriculum`, validators, grading helpers).
- HTTP-level integration tests through `server.New` with a **fake provider** (httptest server whose URL is passed in `LLMURLs`) — real handlers, real Postgres, no real AI. Reference: `learningpath/path_test.go` (65 tests), `learning/*_test.go` (75).
- Fixed clock via `testdb.Clock` / `clock.Clock` injection; never `time.Now()` in logic under test.
- `testdb.NewUser` registers cleanup deleting the user (cascade).
- Write backend tests **in the same change as the feature or fix** (owner's standing rule).

## 4. What is missing today

| Gap | Consequence | Phase |
| --- | --- | --- |
| Tests are not re-runnable the same day on one DB | local noise, hides real failures | P0-05 |
| No frontend unit tests, e2e not in CI | UI regressions unseen; zod/enum contract breaks (R-07) undetected | P0-06 (contract tests), later e2e in CI (D-019) |
| No migration up/down/up test | a bad migration is found in prod | P0-07 |
| No data-backfill test harness | backfills unverified | P1 |
| No simulation tests for adaptive logic | thresholds unvalidated | P3, P6 |
| No golden tests for AI contracts | prompt/validator drift | P3+ (`AI_CONTRACTS.md` section 8) |
| No authorization matrix test | ownership bugs possible per new endpoint | every phase |
| No coverage measurement | unknown blind spots | P0-08 (report only, no gate) |

## 5. Test layers per phase (required additions)

| Layer | What it covers | Tooling |
| --- | --- | --- |
| **Pure unit** | formulas: posterior update, item selection, stop rules, mastery transitions, `NextReview`, priority, `ValidatePath`, session planning | table-driven + property tests (`testing/quick` or hand-rolled generators); no I/O |
| **Contract (AI)** | each contract: golden valid, one fixture per invalid class, retry paths | fixtures in `testdata/ai/<contract>/`; fake `Completer` |
| **DB / repository** | new queries, constraints, partial unique indexes, composite FKs, cascade behaviour | `testdb.Pool`; real Postgres |
| **Migration** | up -> down -> up on a DB seeded with representative legacy rows; backfill idempotency (run twice, same counts); old and new code against the migrated schema | a `migration_test.go` per migration batch using goose programmatically |
| **HTTP integration** | request/response, status/error codes, **authorization matrix** (own / other user / extension token / no token), AI failure matrix | `server.New` + fake provider |
| **Simulation** | synthetic learners (known level per skill; known forgetting curve) through real handlers/functions; assert statistical acceptance (`ASSESSMENT_ENGINE.md` section 8, `LEARNING_METHODOLOGY.md` section 7) | seeded RNG, deterministic; thresholds in tests |
| **Library / KB** | import counts equal JSON; prerequisite graph acyclic and level-monotone; mirror parity across explanation languages; Rule Book DB-vs-embedded parity | extends `curriculum_test.go` |
| **Frontend contract** | every zod schema parses the backend's golden JSON; unknown enum behaviour; no `any` in schemas | a Node test runner added in P0-06 (vitest — requires approval, D-019) or tsc-checked fixtures |
| **E2E** | the critical path: sign in -> placement -> program -> exercise -> review | Playwright; manual/local until D-019 |

## 6. Authorization test matrix (template)

For every new endpoint, in one table-driven test:

| Caller | Expected |
| --- | --- |
| owner session | 2xx |
| other user's session on the owner's id | 404 |
| extension token (`capture`, `read_write`) | 401 |
| no token / expired token | 401 |
| owner, other language's profile id | 404 |

## 7. Definition of "VERIFIED"

A task is `VERIFIED` only when **all** hold, recorded in `IMPLEMENTATION_STATUS.md` with
date, commit, command and result:

1. `go run ./cmd/migrate up` from a database at the **previous** release's schema succeeds.
2. `go vet ./...` and `go test -count=1 ./...` pass on fresh databases (counts reported).
3. `eslint` and `tsc --noEmit` pass if the frontend changed.
4. The task's own acceptance tests exist and are in that run.
5. The authorization matrix test exists for new endpoints.
6. Docs updated in the same commit (`README.md` "Conventions").

`IMPLEMENTED` = code merged locally and its tests written, but the full run above has not
been done or recorded.
