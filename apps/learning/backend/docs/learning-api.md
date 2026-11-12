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

A capture with the same normalized text (case-insensitive, whitespace folded) in the
same topic replaces the existing card instead of adding a duplicate. The new card's
fields, including `source` and `status`, replace the old ones even when the source is
different; the existing entry UUID and `createdAt` stay unchanged. This also applies
to `POST /entries` from the app form.

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

- `GET /ai/settings` — `{"provider":"anthropic","model":"claude-opus-5-5","level":"high","teacher":false,"keyHint":"WXYZ"}`,
  or `null` while nothing is chosen. The key is never returned, only its last four characters.
- `PUT /ai/settings` — `provider`, `model`, `level`, `teacher`, `apiKey`. `teacher` (default `false`)
  turns the chat into a language lesson, see below; it is saved with every `PUT`, so send the
  current value when only changing something else. Omit `apiKey` (or send `""`) to keep
  the stored key, which only works while `provider` stays the same: a key belongs to one provider.
  Responds with the same shape as `GET`. **400** for an unknown provider/model, a level the model
  does not take, a key outside 8..500 characters, no stored key to keep, or a key the provider
  rejects (401 from its model list; an outage or a restricted key does not block saving).
  Responses use `Cache-Control: no-store`.
- `DELETE /ai/settings` — forgets the agent and deletes the stored key. Safe to repeat.
- `POST /ai/complete` — runs one prompt through the chosen agent and returns its text, so a client
  can use the agent without ever holding the key. See below.
- `POST /entries/:id/fill` — has the agent write the fields a saved entry still lacks (transcription,
  translation, explanation, native usage, example, rule) and returns the entry. Fields already on the
  card are neither asked for nor replaced; a complete card comes back untouched, without a call. The
  account's session only (an extension token gets `401`: it spends the user's own key). **400** when no
  agent is chosen or the provider rejected the key, **404** for an entry that is not the user's, **502**
  when the provider is down or its answer held nothing to write from; the card is left as it was.

| `provider` | `model` | `level` |
| --- | --- | --- |
| `anthropic` | `claude-opus-5-5`, `claude-sonnet-5-5`, `claude-haiku-5-5`, `claude-fable-5-1` | `low` `medium` `high` `xhigh` `max` (the model's effort) |
| `openai` | `gpt-6-astra`, `gpt-6.1-sol` | `low` … `max` |
| `openai` | `gpt-6.1-terra`, `gpt-6-luna` | `none` `low` … `max` |
| `deepseek` | `deepseek-v4-pro`, `deepseek-flash` | `off` (thinking disabled), `low`, `high`, `max` |

The key is stored sealed with AES-256-GCM (key derived from `AUTH_SECRET`, bound to the user's
row) and cannot be read back through the API. Rotating `AUTH_SECRET` makes stored keys unreadable:
captures are then saved as sent, and the user enters the key again.

### Run a prompt through the agent

```http
POST /api/v1/learning/ai/complete
Authorization: Bearer <session token>
Content-Type: application/json

{"system":"You explain phrases from subtitles.","prompt":"Explain: hit the road"}
```

Returns `{"code":0,"message":"success","data":{"text":"…"}}` with `Cache-Control: no-store`.
The server opens the sealed key, makes one call to the user's provider with the stored model and
level, and returns the text. The key and the prompt are never logged or returned. The answer arrives
whole (no streaming), after up to 150 seconds. `prompt` is required (up to 60,000 characters);
`system` is optional (up to 10,000); the body is limited to 192 KiB. There is no conversation
memory: a client that wants follow-ups includes the earlier turns in `prompt`.

| Status | Code | Meaning |
| --- | --- | --- |
| 400 | 10001 | Missing, blank or too long `prompt`/`system` |
| 401 | 20001 | Missing or expired session, or an extension token |
| 404 | 10002 | No agent chosen yet |
| 409 | 40001 | The stored key can no longer be read; save the key again |
| 413 | 10001 | Body over the size limit |
| 422 | 30001 | The provider rejected the key |
| 503 | 30002 | Provider outage, rate limit or timeout; try again later |
| 502 | 30003 | The agent refused or returned nothing usable |

## Chat about an entry (session only)

One conversation per entry, with the user's own agent. Extension tokens get **401**.
Only `deepseek` has a chat so far; the other providers answer **400** until they get a
profile in `llm/chat.go`.

- `GET /entries/<uuid>/chat` — `{"available":true,"teacher":false,"messages":[{"id":1,"role":"user","content":"…","createdAt":"…"}]}`.
  `available` is `false` while no agent is chosen or its provider has no chat; the messages
  already exchanged stay readable. `teacher` is the `teacher` choice of the agent settings
  (always `false` while the chat is not available). **404** for a missing or another user's entry.
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

Two modes, by the `teacher` choice of the agent settings. Off, the agent is a tutor that answers
questions about the entry in the topic's translation language. On, it is a teacher of the
topic's language: it talks in that language (short replies that end with a question), and when
the learner writes in it, it adds a `✏️` section after the reply that explains each mistake in the
translation language: what is wrong and why, the correct version, and how a native speaker would
say it. A message in the translation language is treated as the learner being stuck: the agent
answers in it, gives the wording in the learned language and asks to try it. Switching changes the
system prompt (so the provider cache is lost once) and keeps the history.

What the model sees each turn: a system prompt (tutor or teacher rules, both languages, the entry's card,
and the summary once there is one), then the unsummarized messages word for word, then the new
question. It is built append-only so the provider's prefix cache keeps hitting. When the
provider reports the context past the profile's mark (DeepSeek: 7000 tokens), everything but the
newest 8 messages is folded into `learning_chats.summary` by the cheap model
(`deepseek-flash`, thinking off), after the answer has been delivered. The full history stays in
`learning_chat_messages`. Chat replies use no thinking and at most 1500 tokens.

## Free chat with the teacher (session only)

A conversation with the user's agent that is about no entry: a lesson in the topic's language,
opened from the app's chat tab. It is **always the teacher** (the `teacher` setting does not
matter), speaks the topic's language, finds out the learner's level and leads the lesson, and
corrects mistakes with the same `✏️` section. Same provider, rate, budget and daily limits as the
chat of an entry (they are the same rows in `learning_chats`, `topic_id` set instead of `entry_id`).
Extension tokens get **401**.

A topic has one **open** dialog. Clearing it **closes** it (`archived_at`) and the next message
opens a new one, so nothing said is lost; closed dialogs are read-only and can be deleted.
The new dialog starts without the old one's history and summary but **inherits what the old one
spent** of its budget window, so clearing never buys a new budget.

- `GET /topics/<uuid>/chat` — same body as the chat of an entry, for the open dialog
  (`teacher` is `true` whenever `available` is). **404** for a missing or another user's topic.
- `POST /topics/<uuid>/chat` — body and event stream exactly as `POST /entries/<uuid>/chat`.
- `POST /topics/<uuid>/chat/archive` — "clear chat": closes the open dialog. A dialog nobody wrote in
  is left alone. **409** while an answer is being written.
- `GET /topics/<uuid>/chat/history` — closed dialogs, newest first (up to 100):
  `[{"id":"…","title":"how the dialog began (160 chars)","messageCount":4,"startedAt":"…","archivedAt":"…"}]`.
- `GET /chat-dialogs/<uuid>` — one closed dialog with its `messages`. **404** for the open one.
- `DELETE /chat-dialogs/<uuid>` — deletes a closed dialog for good. **404** for the open one.

Deleting a topic deletes its dialogs.

## Words and phrases from a picture (session only)

`POST /ai/scan` — `multipart/form-data` with `topicId` and `image` (JPEG, PNG or GIF up to 6 MB and
16 megapixels, recognized by its bytes). The picture is read **on the server** with tesseract, in the
language of the topic (`en`, `ru`, `uk`, … see `ocr.languages`; **400** for any other); it is never
sent to an AI agent, so reading needs no agent and costs no tokens. The response is

```json
{"items":[{"kind":"word"|"phrase","text":"…"}], "known": 2, "seen": false}
```

`items` are the words and phrases written in the topic's language, in the order they stand on the
picture, at most 30: words tesseract was not sure of, prices, codes and text in another alphabet are
left out, the lines of a wrapped paragraph are joined into sentences (a sentence of up to 20 words is
a phrase, a longer one is cut at its commas), a text with a space is a phrase. No translation is given:
the app sends the pick to `POST /captures` as `source: "photo"`, and the queue fills the card as usual
with the learner's agent, if there is one. A page of text is tried first, scattered words (a sign) if
the page holds none; an empty list means the picture holds nothing to learn.

**Duplicates.** Each entry carries `text_hash`, the SHA-256 of its text in lower case with whitespace
folded (`learning/commands.TextHash`). A found text whose hash an entry of the same topic has is not
offered and is counted in `known`; the same text twice in a picture is offered once. The picture
itself is remembered by the SHA-256 of its file (table `learning_scans`, per topic): `seen` is true
when this very file was read for this topic before. It is a hint, not a refusal: what the picture
holds that the topic does not have is offered again, so a learner who kept part of it can come back
for the rest.

Errors: **400** (no/invalid/too large picture, bad topic id, a language that cannot be read, a picture
the reader could not make sense of), **404** (topic), **500** (tesseract is not installed on the
server), **429** (`42901`, 4 pictures back to back, then one every 15 seconds, one at a time; only
requests that get to the reader count). At most 2 pictures are read at a time on the server.
Extension tokens get **401**. The image installs the language data named in `OCR_LANGS` of
`infra/learning/docker/backend.Dockerfile`; `ocr.languages` must list the same ones.

## Teachers and the study program (session only)

A teacher is a language the learner studies, with the language its rules are
explained in; there is one per language. Everything here spends the learner's
own agent and is never reachable with a topic-scoped extension token. Paths are
under `/api/v1/learning/path`.

| Method and path | What it does |
| --- | --- |
| `GET /teachers` | The learner's teachers, each with a line about its running program (`level`, `topicsDone`, `topicsTotal`). |
| `POST /teachers` | `{language, translationLanguage}`. Adds a teacher; a language that has one hands it back with its own pair. |
| `DELETE /teachers/:id` | Removes a teacher. Its running program is closed into the history, not thrown away. |
| `POST /tests` | Opens a placement test (`{language, translationLanguage, level?}`; `level` starts above A1 and is accepted only as the rung after a confirmed final test). The first batch is written by the agent before the test exists, so without a reachable agent it fails and leaves nothing open. |
| `POST /final-tests` | Opens the final test of the running program's level. |
| `POST /levels` | `{language, translationLanguage, level}`. Picks a level the learner has already reached instead of taking the test again: it answers `201` with a finished placement at that level (no questions, no agent call), which is built into a program with `POST /tests/:id/analyze` like any other, the program's `source` is then `chosen`. The levels that can be picked are `A1` up to the highest level of the language's closed programs (the history is the proof), so `400` with no history or above it, `409` while a program is running. |
| `GET /tests/:id`, `POST /tests/:id/answers` | The state of a test, and one batch of answers; a full batch goes to the examiner (the agent), which climbs or places the learner. |
| `POST /tests/:id/analyze` | Queues building the program from a finished placement test and answers `202` with a job; asking again hands the same job back. |
| `GET /jobs/:id` | `queued`, `running`, `done` (with the program) or `failed` (with why). |
| `DELETE /history?language=` | Clears the learner's history of a language: its closed programs, with their topics, exercises, attempts and lessons (`{deleted: n}`). The running program stays. The newest finished program is also what lets a test start above A1, so after clearing the next test starts from A1. |
| `GET /?language=`, `GET /history`, `POST /reset` | The running program of a language, the closed ones, and closing the running one. Each topic of a program carries `exercises: {total, done}`: how many exercises its current set holds (0 while it is not written) and how many of them the learner has answered right. A regenerated set starts from `done: 0`. |
| `GET /topics/:id`, `POST /topics/:id/exercises`, `POST /topics/:id/complete` | A topic with its rule and words, its exercise set (`?regenerate=1` writes a new one), and completing it. |
| `POST /exercises/:id/attempt` | Checks one answer. Kinds: `choice`, `fill`, `listen` are graded here; `speak` is compared with the phrase; `translate` and `write` are judged by the agent. |
| `GET /exercises/:id/audio`, `GET /topics/:id/words/:index/audio` | The voice of a listening sentence or a phrase to repeat, and of a key word of a topic. |

Once a program is saved the job is done and the learner can start; the agent then
writes the exercise set of every topic in the background (a few at a time, in the
program's order), so each topic opens with its exercises ready and the program's
scale can be counted by them. A set that fails to write is left for when the
learner opens the topic, and the learner can regenerate any set: the counts follow
the current set.

For a language pair that has a library (English for Russian speakers, A1 to B2)
the program is planned from it: the agent only picks topics, and each topic
carries the library's own rule, words and goal.

## Rule book and practice on a rule (session only)

The rule book is the library's grammar rules of the pair a learner has a teacher
for (`403` without one), written by people and never by an agent. Practice is
what the learner's own agent adds to a rule: more example sentences, drills in
the manner of English Grammar in Use, and a chat about each sentence. It is a
*sitting*, not a record: there is one per learner and rule, and it goes away
when the learner leaves the rule's page. Paths are under `/api/v1/learning/path`.

| Method and path | What it does |
| --- | --- |
| `GET /rules`, `GET /rules/:id` | The rules by level, and one rule with its forms, examples, mistakes and the rules it refers to. `?language=` says whose book is asked for: the learner's teacher of that language, and `403` when that pair has no book (a learner who studies English and Polish gets no English rules on the Polish teacher's screen). Without it, the first teacher whose book has the rule. |
| `GET /rules/:id/practice` | The sitting: `{examples: [{id, text, translation}], blocks: [{id, kind, instruction, items: [{id, text, hint, options}]}]}`, empty when there is none. Opening it lifts the expiry that leaving set. The keys never come with it. |
| `POST /rules/:id/practice/examples` | The agent writes five more sentences for the rule, never repeating one already shown (up to 20 per sitting; `409` after that). |
| `POST /rules/:id/practice/exercises` | The agent writes four drills of five lines, one of each kind, replacing the sitting's drills. Kinds: `form` (put the word in brackets into the right form), `choose`, `correct` (the sentence has a mistake), `make` (turn it into a question, a negative…). |
| `POST /rules/:id/practice/check` | `{block, item, answer}`. Grades one line by its key (case, spacing and punctuation are ignored) and answers `{correct, key, note}`. It asks no agent. |
| `POST /rules/:id/practice/key` | `{block, item}`. The key and the reason for a learner who gives up. |
| `POST /rules/:id/practice/leave` | The learner has left the page: the sitting is kept for ten minutes, then goes with its chats. Coming back within that time (`GET`) keeps it for six hours again, and a sitting nobody leaves ends after six hours. |
| `GET`, `POST /rules/:id/examples/:example/chat` | The chat about one sentence: always the teacher, in the learner's language, with the usual `✏️` corrections. It streams like the other chats and shares their budget and limits; it ends with the sitting (`404` after). |

A sitting is swept lazily, when its learner next asks for anything of it, so no
background job is needed. Writing examples or drills spends the learner's key and
is paced like the rest of the program's calls.

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
Entry creation or replacement and receipt completion commit in one transaction.
A new entry initially uses the capture ID; a replacement keeps the existing entry ID
and returns it as `entryId`. Reusing an idempotency key cannot create additional jobs.
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

An extension has two ways to authenticate. A pasted extension token (above) is scoped to one topic
and cannot reach `/ai/*`. Alternatively the user can sign in with the email and password of the
account: `POST /api/v1/auth/login` with `{"email":"…","password":"…"}` returns
`{"code":0,"data":{"token":"v1.…","user":{…}}}`. That session token is a full-access Bearer token valid
for 30 days: it works on every `/learning` route (all topics, captures, entries) and on `/ai/*`,
including `/ai/complete`, so the extension can use the user's agent without ever seeing its key.
Session tokens are stateless and cannot be revoked before they expire; the extension drops its copy
on sign-out. Send `Accept-Language: ru` to get Russian error messages.


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
