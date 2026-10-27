# Learning API

Base URL: `/api/v1/learning`. All responses use the existing envelope:

```json
{"code":0,"message":"success","data":{}}
```

Create a language topic in **Learning**, then create an extension token in
**Learning → Extension access** and select the topic it can access.
The database keeps a SHA-256 hash of every token to authenticate requests and an
AES-256-GCM encrypted copy (key derived from `AUTH_SECRET`) so the app can show the
token again. Revocation blocks subsequent requests; captures already accepted
remain queued.

Send `Authorization: Bearer <token>` on every request. Two token scopes:

- `capture` (default): list topics, enqueue captures, read capture status.
- `read_write`: the above, plus list/get/update/delete entries.

Extension tokens cannot access account endpoints, token management, or
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

`text` is required. `topicId` may be omitted for a topic-scoped extension token;
if supplied, it must match the token's topic. `kind` defaults to `phrase`, `status` defaults
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
Request bodies are limited to 64 KiB. The extension may prepare the translation and
explanations before queueing the capture; the API persists the supplied fields. If the
user has chosen an AI agent (see below), the queue writes the fields the capture left
empty, and never overwrites a supplied one.

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

## AI agent (session only)

The user picks which agent writes the card fields a capture leaves empty. Extension
tokens get **401** on these endpoints.

- `GET /ai/settings` — `{"provider":"anthropic","model":"claude-opus-5-5","level":"high","keyHint":"WXYZ"}`,
  or `null` while nothing is chosen. The key is never returned, only its last four characters.
- `PUT /ai/settings` — `provider`, `model`, `level`, `apiKey`. Omit `apiKey` (or send `""`) to keep
  the stored key, which only works while `provider` stays the same: a key belongs to one provider.
  Responds with the same shape as `GET`. **400** for an unknown provider/model, a level the model
  does not take, a key outside 8..500 characters, no stored key to keep, or a key the provider
  rejects (401 from its model list; an outage or a restricted key does not block saving).
  Responses use `Cache-Control: no-store`.
- `DELETE /ai/settings` — forgets the agent and deletes the stored key. Safe to repeat.

| `provider` | `model` | `level` |
| --- | --- | --- |
| `anthropic` | `claude-opus-5-5`, `claude-sonnet-5-5`, `claude-haiku-5-5`, `claude-fable-5-1` | `low` `medium` `high` `xhigh` `max` (the model's effort) |
| `openai` | `gpt-6-astra`, `gpt-6.1-sol` | `low` … `max` |
| `openai` | `gpt-6.1-terra`, `gpt-6-luna` | `none` `low` … `max` |
| `deepseek` | `deepseek-v4-pro`, `deepseek-flash` | `off` (thinking disabled), `low`, `high`, `max` |

The key is stored sealed with AES-256-GCM (key derived from `AUTH_SECRET`, bound to the user's
row) and cannot be read back through the API. Rotating `AUTH_SECRET` makes stored keys unreadable:
captures are then saved as sent, and the user enters the key again.

## Chat about an entry (session only)

One conversation per entry, with the user's own agent. Extension tokens get **401**.
Only `deepseek` has a chat so far; the other providers answer **400** until they get a
profile in `llm/chat.go`.

- `GET /entries/<uuid>/chat` — `{"available":true,"messages":[{"id":1,"role":"user","content":"…","createdAt":"…"}]}`.
  `available` is `false` while no agent is chosen or its provider has no chat; the messages
  already exchanged stay readable. **404** for a missing or another user's entry.
- `POST /entries/<uuid>/chat` — body `{"text":"…"}` (1..2000 characters). A refused question is an
  ordinary error response: **400** (no agent, unsupported provider, bad text, unreadable key),
  **404**, **409** (the previous answer or a compaction of this chat is still running), **502**
  (code `50201`, the provider is down, timing out or rate-limiting).
  Otherwise the response is a `text/event-stream`:
  `event: delta` `{"text":"…"}` for each piece of the answer, then `event: done`
  `{"question":{…},"answer":{…}}` with the stored messages, or `event: error`
  `{"code":…,"message":"…"}` if it fails once started. Nothing is stored unless the answer
  arrived whole; the question can simply be sent again. The provider call keeps going if
  the client disconnects, so the answer is there when the chat is opened again.
- `DELETE /entries/<uuid>/chat` — forgets the conversation and its summary. **409** while busy.

Limits (both answer **429**, told apart by the business `code`):

- **Rate, `42901`:** per user, a token bucket of 5 questions back to back, then one every 4 seconds,
  and at most 2 answers being written at once. `Retry-After` says how many seconds to wait.
  It is kept in the process's memory, so it only ever gets looser by the number of running backends.
- **Budget, `42902`:** per chat, 120 000 tokens (prompt and reply of every turn, so a long history
  weighs what it is paid for) within a 24-hour window that opens with the first turn after the
  previous window ended. The turn that crosses the limit still completes, the next question is
  refused with `Retry-After` and `details: ["<RFC 3339 time the window ends>"]`. Clearing a chat
  does **not** give the budget back. `GET` returns `budget: {limit, used, resetsAt}` (null while
  `available` is false) and `done` carries it with the turn counted. The numbers are the provider
  profile's (`BudgetTokens`, `BudgetWindow` in `llm/chat.go`); the cost of the cheap summarizing
  call is not counted.

- **Daily, `42903`:** two counters per UTC day in `learning_chat_usage`, both in tokens across all chats.
  *Per user* (400 000): keyed by the user, so changing the API key or starting new chats buys nothing.
  *Per API key* (800 000, looser so people sharing one key are not the ones to hit it): keyed by an
  HMAC fingerprint of the key (never the key), so several accounts on one key share it. Refused with
  `Retry-After` and `details: ["<next UTC midnight>"]`. `GET` returns `daily` (the user's counter) and
  `done` carries it. Counters older than a week are swept as turns are recorded. Like the chat's budget
  these are soft by the turns in flight (at most 2 per user). Rotating `AUTH_SECRET` renames every
  fingerprint, which resets the key counters.

What the model sees each turn: a system prompt (tutor rules, both languages, the entry's card,
and the summary once there is one), then the unsummarized messages word for word, then the new
question. It is built append-only so the provider's prefix cache keeps hitting. When the
provider reports the context past the profile's mark (DeepSeek: 7000 tokens), everything but the
newest 8 messages is folded into `learning_chats.summary` by the cheap model
(`deepseek-flash`, thinking off), after the answer has been delivered. The full history stays in
`learning_chat_messages`. Chat replies use no thinking and at most 1500 tokens.

## Session-only management

- `POST /topics`, `PUT /topics/<uuid>` — `name`, `language`, `translationLanguage`.
- `DELETE /topics/<uuid>` — cascades to that topic's entries.
- `GET /tokens` — metadata only, no secrets. `revealable` tells whether the token can
  still be shown with `/secret`.
- `POST /tokens` — `name`, `topicId`, `scope` (`capture` or `read_write`); includes the raw
  token in this response. Token responses use `Cache-Control: no-store`.
- `GET /tokens/<uuid>/secret` — the raw token of an active token, `{"token":"bml_…"}`
  (`Cache-Control: no-store`). **404** when the token is revoked, belongs to someone else, or
  was issued before encrypted copies existed (`revealable: false`); those keep working but
  cannot be shown. Rotating `AUTH_SECRET` makes stored copies unreadable (**500**) the same way.
- `DELETE /tokens/<uuid>` — revoke.

Missing/invalid/revoked token: **401**; insufficient scope: **403**;
missing or another user's resource: **404**; invalid payload: **400**.

## Queue operation

Migration `0001_learning.sql` creates a durable Postgres queue. The HTTP server
runs a consumer automatically; no broker or cron setup is required.
Consumers claim jobs with `FOR UPDATE SKIP LOCKED`; a one-minute lease
allows recovery after a crash. Each attempt has a 55-second processing timeout, so the
lease outlives it.
Entry creation and receipt completion commit in one transaction. The capture ID
is also the entry ID, so retries cannot create additional entries.
Temporary failures retry with exponential delays, up to five attempts; permanent
errors (such as a deleted topic) finish as `failed`.

Enrichment happens in the same attempt, before the entry is written, and only for users with
an agent chosen and a capture with empty card fields. One call to the user's provider (40-second
limit) asks for exactly the missing fields; the prompt carries the saved text as data, never as
instructions, and the reply is parsed as a JSON object of strings. An agent problem never fails
a capture: a permanent one (rejected key, refusal, unreadable reply) saves the entry as sent; a
temporary one (timeout, rate limit, provider outage) is retried on attempts 1 and 2 and saved as
sent on attempt 3. Entries added from the app form (`POST /entries`) are not enriched.

## Chrome integration

Keep the token in extension storage and send requests from the extension's
background/service worker with host permission for the API origin. A content
script should pass the selected text to the background worker.
If your client sends an Origin requiring a CORS preflight, include its exact
origin in `CORS_ORIGINS`; `Authorization`, `Content-Type`, and `Idempotency-Key`
are allowed request headers. Never embed an issued token in distributed code.

## Verification

Start the Learning development stack with its own Postgres volume, then run
the backend and frontend checks:

```bash
cp apps/learning/.env.example apps/learning/.env
docker compose --env-file apps/learning/.env -f infra/learning/compose/dev.yml up -d --build postgres
docker compose --env-file apps/learning/.env -f infra/learning/compose/dev.yml --profile tools run --rm migrate
docker compose --env-file apps/learning/.env -f infra/learning/compose/dev.yml up -d --build backend frontend
cd apps/learning/backend && go test ./...
cd ../frontend && pnpm lint && pnpm typecheck
```

The browser suite can use `E2E_BACKEND_URL` and `E2E_FRONTEND_PORT` to select
the Learning test backend and frontend ports. `E2E_CHROMIUM_PATH` optionally
selects an installed Chrome/Chromium binary.
