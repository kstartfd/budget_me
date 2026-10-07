# Learning API

Base URL: `/api/v1/learning`. All responses use the existing envelope:

```json
{"code":0,"message":"success","data":{}}
```

Create a language topic in **Learning**, then create an extension token in
**Settings → Extension access** or **Learning → Extension access**.
Tokens are shown once. The database stores only their SHA-256 hashes. Revocation
blocks subsequent requests; captures already accepted remain queued.

Send `Authorization: Bearer <token>` on every request. Two token scopes:

- `capture` (default): list topics, enqueue captures, read capture status.
- `read_write`: the above, plus list/get/update/delete entries.

Extension tokens cannot access money, account settings, token management, or
create/update/delete topics. Those operations require the app's session token.

## Find a topic

`GET /topics` returns your topics:

```json
[{"id":"<topic-uuid>","name":"English","language":"en","translationLanguage":"ru"}]
```

Language values are BCP 47 tags, for example `en`, `ru`, `en-US`.

## Queue a word or phrase

```http
POST /api/v1/learning/captures
Authorization: Bearer <extension-token>
Content-Type: application/json
Idempotency-Key: <unique-id-for-this-selection>

{"topicId":"<topic-uuid>","kind":"phrase","text":"hit the road","translation":"отправиться в путь","explanation":"hit the road — отправиться; устойчивое выражение","rule":"Идиома: hit the road","nativeUsage":"Разговорное, обычно об отъезде","source":"Video 01:23"}
```

Returns **202 Accepted**, with a `Location` header for the receipt:

```json
{"code":0,"message":"success","data":{"id":"<capture-uuid>","status":"pending","entryId":null,"error":""}}
```

`topicId` and `text` are required. `kind` defaults to `phrase`, `status` defaults
to `inbox`. Optional fields default to empty strings:

| Field | Meaning |
| --- | --- |
| `kind` | `word` or `phrase` |
| `transcription` | Pronunciation |
| `translation` | Translation |
| `explanation` | Breakdown of words and expressions in the phrase |
| `nativeUsage` | Natural wording, nuance, native usage |
| `example` | Example sentence and usage situation |
| `rule` | Grammar or construction explanation |
| `source` | Origin of the phrase, such as a video title and timestamp |
| `status` | `inbox`, `learning`, `known` |

`text` is limited to 1,000 characters; each other text field to 4,000.
Request bodies are limited to 64 KiB. The extension prepares the translation and
explanations before queueing the capture; the API persists the supplied fields.

Generate a new idempotency key for each intentional selection and **reuse it on
network retries**. Keys are scoped to your account, not to one token.
The same key and normalized payload return the same receipt; a different payload
with that key returns **409**. Without a key, each request creates a new job.
Retries with the original payload return the existing receipt even after the
topic or entry is deleted; a new capture still requires an existing topic.
A 202 response means the capture is persisted, not that its entry is ready yet.

## Check delivery

`GET /captures/<capture-uuid>` works with either extension scope:

```json
{"id":"<capture-uuid>","status":"done","entryId":"<entry-uuid>","error":""}
```

Statuses: `pending`, `processing`, `done`, `failed`.
Poll with a delay (for example one second, then gradually increase it).
For `failed`, show the returned explanation. A receipt describes delivery;
it remains available even if the user later deletes the entry or its topic.

## Read, update, delete

Requires `read_write` or an app session:

- `GET /entries?topicId=<uuid>&limit=100&offset=0`
- `GET /entries/<uuid>`
- `PUT /entries/<uuid>` — complete replacement; send `topicId`, `kind`, `text`,
  all optional card fields and `status`. Omitted fields become empty.
- `DELETE /entries/<uuid>`

List supports `kind=word|phrase`, `status=inbox|learning|known`, and literal
case-insensitive `search` across original text, translation, breakdown, rule and source.
`limit` is 1–200 (default 100); pagination and filters run on the server.

`POST /entries` is the synchronous save used by the app session.
**Extension tokens always add entries through `/captures`**, so they cannot
bypass the queue.

## Session-only management

- `POST /topics`, `PUT /topics/<uuid>` — `name`, `language`, `translationLanguage`.
- `DELETE /topics/<uuid>` — cascades to that topic's entries.
- `GET /tokens` — metadata only, no secrets.
- `POST /tokens` — `name`, `scope` (`capture` or `read_write`); includes the raw
  token only in this response. Token responses use `Cache-Control: no-store`.
- `DELETE /tokens/<uuid>` — revoke.

Missing/invalid/revoked token: **401**; insufficient scope: **403**;
missing or another user's resource: **404**; invalid payload: **400**.

## Queue operation

Migration `0021_learning.sql` creates a durable Postgres queue. The HTTP server
runs a consumer automatically; no new broker or cron setup is required.
Additional consumers can run independently using the existing backend image:

```bash
docker compose -f infra/compose/prod.yml --profile tools run --rm worker learning-captures
```

This command runs continuously until SIGTERM/SIGINT, unlike the other worker
commands. Consumers claim jobs with `FOR UPDATE SKIP LOCKED`; a one-minute lease
allows recovery after a crash. Each attempt has a 20-second processing timeout.
Entry creation and receipt completion commit in one transaction. The capture ID
is also the entry ID, so retries cannot create additional entries.
Temporary failures retry with exponential delays, up to five attempts; permanent
errors (such as a deleted topic) finish as `failed`. Entries are not automatically
translated or enriched: optional fields stay empty until supplied or edited.

## Chrome integration

Keep the token in extension storage and send requests from the extension's
background/service worker with host permission for the API origin. A content
script should pass the selected text to the background worker.
If your client sends an Origin requiring a CORS preflight, include its exact
origin in `CORS_ORIGINS`; `Authorization`, `Content-Type`, and `Idempotency-Key`
are allowed request headers. Never embed an issued token in distributed code.

## Verification

Use the isolated test stack; it has its own Postgres volume and a test-only
session signing secret:

```bash
make test.up
docker compose -f infra/compose/test.yml exec backend go test ./...
docker compose -f infra/compose/test.yml exec -e E2E_SESSION_FILE=/app/tmp/learning-session.json backend go run ./.learning-fixture
cd frontend
E2E_SESSION_FILE=../backend/tmp/learning-session.json pnpm test:e2e --project=chromium-auth
```

The fixture requires `ENV=test`, `DATABASE_URL` and `AUTH_SECRET`, creates a test
account with sample entries, and writes a one-hour session token to the JSON
file with mode `0600`. Re-run it if the token expires. The Compose backend
already supplies these environment variables. `backend/tmp/` is ignored by Git.

The Learning browser suite creates and removes its own topics and extension
tokens. It checks desktop and mobile editing while a status filter is active,
capture delivery through the running queue consumer, and token revocation.
Without `E2E_SESSION_FILE`, these tests are skipped. Use
`E2E_BACKEND_URL` and `E2E_FRONTEND_PORT` for alternate ports;
`E2E_CHROMIUM_PATH` optionally selects an installed Chrome/Chromium binary.
The default browser requires `pnpm exec playwright install chromium`.
