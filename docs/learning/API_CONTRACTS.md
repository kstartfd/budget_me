# API contracts

Sections 1-4 describe the API **as it is** (facts, verified in `routes.go` files and
`httpx`). The full request/response reference of every existing endpoint stays in
[`apps/learning/backend/docs/learning-api.md`](../../apps/learning/backend/docs/learning-api.md)
— **that file is canonical for existing endpoints; do not copy it here**, update it when
behaviour changes. Sections 5-7 are the compatibility rules and the **planned** additions
per phase (`PROPOSED`; shapes are drafts that each phase confirms before coding).

## 1. Conventions

| Topic | Rule |
| --- | --- |
| Base | `/api/v1`; learning routes under `/api/v1/learning` |
| Envelope | `{"code":0,"message":"success","data":...}`; errors `{"code":N,"message":"...","data":null,"details":[...]}` |
| JSON | camelCase fields; ids are UUID strings; timestamps RFC 3339 UTC |
| Auth | `Authorization: Bearer <token>`. **Session token** (`v1.…`, 30 days) or, on notebook routes only, an **extension token** (`bml_…`, scope `capture` / `read_write`, optionally bound to one notebook topic) |
| Ownership | a resource of another user is **404**, never 403 |
| Language | `Accept-Language: ru|en|es` localises error messages |
| Caching | user-specific reads send `Cache-Control: no-store`; secrets/tokens too |
| Idempotency | `Idempotency-Key` on `POST /captures`; heavy POSTs are idempotent by state (one open test per kind/language, one job per test) |
| Limits | per-route body limits (16K/64K/192K/…); list endpoints `limit` 1-200 default 100 + `offset` |
| Streaming | chat answers are `text/event-stream` (`delta`, `done`, `error`) |

### 1.1 Error codes (`httpx/errors.go`, `learningai/errors.go`)

| HTTP | Code | Meaning |
| --- | --- | --- |
| 400 | 10001 | invalid input (also: no AI agent set up where one is required) |
| 404 | 10002 | not found / not yours |
| 401 | 20001 | missing/expired session, or extension token on a session-only route |
| 403 | 20002 | extension token with insufficient scope |
| 409 | 40001 | conflict (program already running, busy chat, unreadable stored key, test already done) |
| 422 | 30001 | the provider rejected the learner's API key |
| 503 | 30002 | AI provider outage / rate limit / timeout |
| 502 | 30003 | the agent refused or returned nothing usable |
| 502 | 50201 | chat provider down |
| 429 | 42901 | rate limit (`Retry-After`) |
| 429 | 42902 | per-chat token budget (`details: [window end]`) |
| 429 | 42903 | daily token limit (`details: [next UTC midnight]`) |
| 500 | 10000 | server error |

New business errors take a new code in the matching range; **codes are never reused or renumbered**.

## 2. Endpoint index (existing)

`S` = session token only; `S/E` = session or extension token (scope rules in `learning-api.md`).

### 2.1 Notebook, capture queue, review (S/E for the first block)

`GET|POST /learning/topics`, `PUT|DELETE /learning/topics/:id` · `GET|POST /learning/entries`,
`GET|PUT|DELETE /learning/entries/:id`, `GET /learning/entries/:id/audio` · `POST /learning/captures`,
`GET /learning/captures/:id`.
**S:** `POST /learning/entries/:id/fill` · `POST /learning/topics/:id/pack` ·
`GET /learning/topics/:id/review` · `POST /learning/entries/:id/review` ·
`GET /learning/topics/:id/sets`, `GET /learning/topics/:id/sets/:period`,
`PUT /learning/topics/:id/plan`, `POST /learning/entries/:id/set-grade` ·
`GET|PUT /learning/reminder`.

### 2.2 AI, tokens, marks (S)

`GET|PUT|DELETE /learning/ai/settings` · `POST /learning/ai/complete` · `POST /learning/ai/scan` ·
`GET|POST /learning/tokens`, `GET /learning/tokens/:id/secret`, `DELETE /learning/tokens/:id` ·
`GET|PUT|DELETE /learning/chat/marks`.

### 2.3 Chats (S)

`GET|POST|DELETE /learning/entries/:id/chat` · `GET|POST /learning/topics/:id/chat`,
`POST /learning/topics/:id/chat/archive`, `GET /learning/topics/:id/chat/history` ·
`GET|POST|DELETE /learning/path/topics/:id/chat` ·
`GET|POST /learning/path/rules/:id/examples/:example/chat` ·
`GET|DELETE /learning/chat-dialogs/:id`.

### 2.4 Teacher, tests, program, lessons (S) — all under `/learning/path`

| Group | Endpoints |
| --- | --- |
| Teachers | `GET /teachers`, `POST /teachers`, `PATCH /teachers/:id`, `DELETE /teachers/:id` |
| Tests | `POST /tests`, `POST /final-tests`, `POST /levels`, `GET /tests/:id`, `POST /tests/:id/answers`, `POST /tests/:id/analyze`, `GET /jobs/:id` |
| Program | `GET /?language=`, `GET /history`, `DELETE /history?language=`, `POST /reset`, `PUT /goal` |
| Topics / exercises | `GET /topics/:id`, `POST /topics/:id/exercises` (`?regenerate=1`), `POST /topics/:id/complete`, `POST /exercises/:id/attempt` (optional `activityId` since V1), `GET /exercises/:id/audio`, `GET /topics/:id/words/:index/audio` |
| Learning loop (V1) | `GET /activities?language=`, `GET /dashboard?language=` (both session-only, under `/learning/path`; shapes in `apps/learning/backend/docs/learning-api.md`) |
| Lessons | `POST /lessons`, `GET /lessons`, `GET /lessons/active`, `GET /lessons/totals`, `POST /lessons/:id/pause|resume|finish` |
| Rule book & practice | `GET /rules`, `GET /rules/:id`, `GET /rules/:id/practice`, `POST /rules/:id/practice/{examples,exercises,check,key,leave}` |

### 2.5 Auth / account

`POST /auth/telegram`, `POST /auth/login`, `GET /me`, `POST /auth/link/email`,
`POST /account/deletion`, `DELETE /account/deletion`, `GET /healthz`.

## 3. Compatibility rules (binding for every phase)

1. **Additive only within a release.** New fields on responses, new optional request fields, new endpoints. Never remove, rename, retype or change the meaning of a field, status code or error code that a shipped client reads.
2. **Two-release contraction.** To remove something: release N stops using it and documents it as deprecated here; release N+1 (after the frontend no longer reads it, verified by search) removes it. Production runs two app versions during a blue-green deploy, so N and N+1 are never the same deploy.
3. **Unknown fields are ignored** by our clients (no zod schema uses `.strict()`, verified); keep it that way. **But enums are closed:** the frontend parses statuses and kinds with `z.enum([...])` (`features/path/schemas.ts`: test `status` `in_progress|done`, topic `status` `not_started|in_progress|completed`, path `status`, job `status`, exercise `kind`, lesson `status`). **A new enum value emitted by the backend makes that response fail to parse and breaks the screen.** Therefore any new value (planned: test status `incomplete`, topic status `blocked|skipped`, new exercise kinds, new path `source`) needs a two-step ship: release N changes the frontend schema to tolerate it (`z.enum([...]).catch(...)` or a string with a known-values helper) — release N+1 lets the backend emit it. Add this to the phase's migration plan every time.
4. **Extension/capture API is frozen**: `/learning/captures`, `/learning/entries`, `/learning/topics` (S/E routes) keep their shapes — real tokens live in users' browsers. Any change goes through D-018.
5. **Session-only by default**: every new endpoint is `S` unless this file says otherwise.
6. **No version prefix.** `/api/v1` stays. A genuinely breaking redesign would need `/api/v2` and its own decision record.
7. **Documentation in the same commit**: a change to an existing endpoint updates `learning-api.md`; a new endpoint is added there and to the planned/implemented tables below.
8. **Server-owned meaning**: clients never send levels, mastery, scores, weights or evidence outcomes; they send answers and observations (hints used, elapsed time) that the server validates and clamps.

## 4. Authorization checklist (per new endpoint)

- [ ] resource lookup joins on `user_id` (or proves ownership through a parent)
- [ ] another user's id returns 404 (test with two users)
- [ ] extension token returns 401 (session-only) — test
- [ ] profile/teacher scoping: a profile id of language X cannot read language Y's data
- [ ] AI-spending endpoints are rate-limited like existing ones
- [ ] responses of user-specific data send `Cache-Control: no-store`

## 5. Planned additions by phase (`PROPOSED`, all `S`)

Status of each row lives in `IMPLEMENTATION_STATUS.md`.

| Phase | Endpoint / change | Shape (draft) |
| --- | --- | --- |
| 1 KB | `GET /learning/kb/items?language&level&skill&kind&category&q&limit&offset` | `[{id,key,kind,skill,cefrLevel,unit,title}]` (title in the learner's explanation language) |
| 1 | `GET /learning/kb/items/:id?explainIn=` | item + current version content + `prerequisites[]`, `related[]`, `sources[]` |
| 1 | `GET /learning/kb/categories?language` | tree |
| 1 | Rule book endpoints | **unchanged shape**; served from DB after the parity test |
| 2 Profile | `GET /learning/profiles` ; `GET /learning/profiles/:language` | profile + goals + preferences + `skills:[{skill,level,confidence,basis,updatedAt}]` |
| 2 | `PUT /learning/profiles/:language` | `{goals:{interests,variant,goal,domain}, weeklyMinutes, sessionMinutes}` |
| 2 | teacher responses | gain `profile` summary; existing fields untouched |
| 3 Assessment | `POST /path/tests` | accepts optional `purpose`, `skills[]`, `startPoint`; response adds `progress.skills[{skill,level,confidence,answers}]`, `stopReason` when done |
| 3 | `POST /path/tests/:id/stop` | marks `incomplete`, returns the partial result |
| 3 | `GET /learning/profiles/:language/assessments`, `GET .../assessments/:id` | per-skill results, `gaps[]` ranked |
| 4 Path | `GET /path` | topics gain `reason`, `priority`, `kbItemId`, and **boolean** `blocked` / `skipped` (not new `status` values — closed frontend enum, R-07); path gains `revision` |
| 4 | `GET /path/revisions?language` | `[{revision,trigger,createdAt,changes:[{op,topicId,reason}]}]` |
| 4 | `POST /path/refresh` | runs `AdjustPath` on demand (idempotent) |
| 5 Exercises | exercise payload | adds `kbItemId`, `difficulty`, `mode`; **`answer` omitted when the client sends `X-Hide-Keys: 1`** (default unchanged until the frontend migrates, then flipped via rule 2) |
| 5 | `POST /exercises/:id/reveal` | records a hint/give-up, returns the key; affects evidence weight |
| 5 | `POST /exercises/:id/attempt` | request adds optional `hintsUsed`, `elapsedMs`; response adds `evidence:{outcome,score}`, `mistake:{category,itemId}|null` |
| 6 Adaptive | `GET /learning/profiles/:language/review-queue?limit` | due items with the exercise to run |
| 6 | `GET /learning/mistakes?resolved&limit` | the mistake journal; recurring flagged |
| 7 Sessions | `POST /path/lessons` | accepts `{language, minutes?}`; response gains `plan[]`; `GET /lessons/active` returns plan + cursor |
| 7 | `POST /path/lessons/:id/heartbeat`, `POST /path/lessons/:id/items/:itemId/skip` | active time and skipping |
| 8 Dashboard | `GET /learning/dashboard?language&tz` (V1 ships a minimal `GET /learning/path/dashboard?language=`; this row is the full version, it extends rather than replaces it) | `{nextActions[], reviewQueue{count,dueSoon}, week{activeSeconds,sessions,attempts}, skills[], mastery{perLevel}, insights[{text,evidence[]}], states{}}` — every numeric field nullable with `notEnoughData` reason |
| 8 | `GET /learning/stats/weekly?language&tz&weeks` | per-week series |
| 9 Tutor | chat `done` event | adds optional `evidence` summary (what was logged) — additive |
| 10 Advanced | speech/pronunciation upload, audio assessment | designed when started (multipart; size limits; storage decision) |

## 6. Shape rules for new payloads

- Percentages and scores are **fractions 0..1** with `null` when not enough data (never `0` as "unknown").
- Every aggregate that depends on a sample size returns the sample size beside it (`n`).
- Level fields are `A1..C2` strings; per-skill objects always carry `confidence` and `basis`.
- Lists are explicit-ordered and paginated; no unbounded arrays except program topics (<= 14 today).
- Timezone: `tz` (IANA) is passed for day/week bucketing, as the notebook sets API already does (`LocalDate`).

## 7. Contract tests (required per phase)

For each new/changed endpoint: a handler-level HTTP test (like the existing `server.New`-based
tests) covering success, validation errors, **ownership 404 with a second user**, extension-token
401, and — for AI-using endpoints — the failure matrix (no agent, rejected key, outage,
unreadable output). A golden-JSON test pins the response shape of every endpoint that the
frontend consumes so accidental breaking changes fail CI.
