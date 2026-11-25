# Knowledge base

Phase 1. Section 1 describes what exists; sections 2-8 are `PROPOSED`. Tables are in
`DATA_MODEL.md` B.1; the API in `API_CONTRACTS.md` section 6.

## 1. Current library (facts)

- **Format.** `apps/learning/backend/internal/learningpath/curriculum/data/<learned>_<explained>_<level>.json` (topics) and `<learned>_<explained>_rules.json` (rule-book extras keyed by topic id), embedded in the Go binary (`//go:embed`) and parsed in `init()`; a malformed file panics at start-up. A language tag is reduced to its base (`en-US` -> `en`).
- **Content.** 25 files: 5 pairs (`en_ru`, `en_en`, `es_ru`, `es_en`, `es_es`) x A1..B2 + rules. 85 (English) / 91 (Spanish) topics per pair, 45 / 51 rules per pair, ~60 key words per level where the explanation language differs from the studied one (counts in `CURRENT_STATE.md` section 6).
- **Topic shape.** `id` (e.g. `a1-to-be`), `skill` (grammar|vocabulary|reading|writing|listening|speaking), `unit`, `title`, `goal` (a can-do sentence), `theory` (markdown-ish text with `` `examples` ``), `vocabulary[{word, translation}]`, `tags[]` (the mistakes it fixes — used by the AI planner). Rule extras: `forms[]`, `examples[{text, translation}]`, `mistakes[{wrong, right, note}]`, `related[]` (topic ids).
- **API surface.** Only through the program (topics copied into `learning_path_topics.theory/vocabulary` at build time) and the rule book (`GET /path/rules`, `/path/rules/:id`, practice). No endpoint lists the library.
- **Guarantees tested** (`curriculum_test.go`, 9 tests). The deepest field-level checks (`TestLibraryIsWellFormed`) run on the **`en_ru`** pair only; the other pairs are covered by mirroring tests against it and by their own, lighter well-formedness tests (`TestTheSpanishLibraryIsWellFormed`). Phase 1 must generalise the field checks to every pair. What `en_ru` is checked for: the library reaches B2; unique ids that start with the file's level; valid skill; non-empty required fields and length limits (title 120, goal 400, unit 80, theory 150-1200 characters); every topic has tags; vocabulary topics have 6-10 words, others at most 10, each with a translation; every level has >= 2 listening and >= 2 speaking and >= 8 grammar topics; example markup in pairs; the English and Spanish libraries mirror their other explanation-language versions id-for-id; Spanish keeps accents.
- **Principle already adopted:** rules, words and goals are *written and checked by people*; the agent only chooses and orders them.

### 1.1 What is missing relative to the plan

| Need | Today |
| --- | --- |
| Stable canonical item id | id is unique only inside a file; `source_id` references are untyped text |
| Language-level identity | the same `a1-to-be` exists as separate rows in `en_ru` and `en_en`; nothing says they are the same item |
| Prerequisites / dependencies | none (only `related` between rules and `tags`) |
| Taxonomy | `unit` free-text label inside a level |
| Sources / provenance | none |
| Versioning | a content edit changes behaviour for everyone at the next deploy; no history |
| Kinds beyond topic/rule | no pronunciation, functional language, communication tasks, objectives |
| Above B2, other pairs | not covered (the AI writes the program) |

## 2. Target model

An **item** is a concept a learner can know, with a stable identity per *studied language*;
its **content** is per *explanation language* and versioned. This matches the data: the
English items are shared by `en_ru` and `en_en`, the Spanish by `es_ru`, `es_en`, `es_es`,
so **prerequisites are authored once per studied language** (two graphs of ~85 and ~91
nodes), not once per pair.

### 2.1 Kinds

| `kind` | Examples | Notes |
| --- | --- | --- |
| `grammar` | to be, Past Simple, conditionals | today's grammar topics and rules |
| `lexeme` | a word sense with level and lemma | child of a vocabulary topic (`part_of`); enables word-level mastery |
| `phrase` | collocations, idioms, chunks | later mirror of notebook cards (D-010) |
| `pronunciation` | /θ/ vs /s/, word stress, intonation of questions | new content |
| `function` | greeting, requesting, apologising, giving opinion | new content (CEFR functional/can-do) |
| `communication_task` | "order a meal", "ask for directions" with outcome and rubric | tutor scenarios exist; this makes them items |
| `reading_text`, `listening_task`, `speaking_task`, `writing_task` | graded materials | today these are *topics* of the library with `skill` set |

### 2.2 Identity and stability rules (important)

- Natural key `(language, key)`; `key` is the existing library id for migrated items (`a1-to-be`). **Keys are immutable.** Renaming = new item + `superseded_by` on the old one; deleting = `status = deprecated` (kept, never reused).
- Today's keys **embed the level** (`a1-to-be`; `TestLibraryIsWellFormed` requires the id to start with the file's level). After migration the key is just an opaque slug: if an item's level is corrected the key stays and only `cefr_level` changes. The "id starts with level" assertion applies to legacy keys only and must be relaxed for new ones.
- `kb_items.id` (uuid) is generated on first import and is what every foreign key uses.
- Evidence references **items**, never versions; versions describe how the item was taught at a time.
- A program topic stores a snapshot of the text shown (as today) **and** `kb_item_id` + the `version` it was built from, so history stays readable after edits.

### 2.3 Taxonomy

Two orthogonal axes plus free categories:

1. **Skill** (`grammar, vocabulary, reading, writing, listening, speaking, pronunciation`).
2. **CEFR level** per item (`A1..C2`) — a property of the item, not a gate.
3. **Categories** (`kb_categories`, a tree per language): e.g. grammar -> verbs/tenses, nouns & articles, modality, clauses, questions & negation; vocabulary -> everyday, travel, work, health; function -> social, transactional, opinion. The existing `unit` strings seed the first tree; goal domains (`travel`, `work`, `exam`) are categories on tasks and vocabulary.

### 2.4 Relations

| `relation` | Meaning | Use |
| --- | --- | --- |
| `prerequisite` (`strength` 1 hard / 2 soft) | A should be learned before B | path validation, availability, remedial insertion |
| `related` | studied together | rule-book "see also" (today's `related[]`), tutor retrieval |
| `contrasts_with` | commonly confused | interleaving, targeted drills (Past Simple vs Present Perfect) |
| `part_of` | lexeme in a vocabulary set, example in a unit | grouping |

Constraints enforced by import and tests, not by SQL: the prerequisite graph is **acyclic**;
a hard prerequisite has `cefr_level <=` its dependent's; no dangling ids; every `kind` has
the required `body` fields.

### 2.5 Content body per kind (`kb_item_versions.body`, `jsonb`)

Versioned JSON Schema files in the repo (`kb/schemas/*.json`), validated on import.
`grammar` keeps today's shape exactly (`theory`, `forms`, `examples`, `mistakes`) so the
existing rule-book and program text come out byte-identical. Language-specific extras are
optional fields, e.g. Spanish `gender`, `accent_notes`; English `article_notes`.

### 2.6 Sources

`kb_sources` records where knowledge comes from (a CEFR descriptor, a textbook section, a
corpus, "authored by the project"). Each item links to >= 1 source with a `locator`.
**Licensing rule:** a source is a *citation*; copyrighted lists or text are never copied
into the repo. Reference candidates to evaluate (not yet used): the CEFR and its Companion
Volume descriptors; Cambridge's English Grammar/Vocabulary Profile; Instituto Cervantes'
Plan curricular for Spanish. Each needs a licence check before any import (D-015).

## 3. Versioning

- `version` increments **when `content_hash` changes** during import (hash of the canonical JSON body + title + goal). Old versions stay (`kb_item_versions` rows are never deleted).
- `published_at` NULL = draft; the importer publishes only when the full validation suite passes.
- Existing learners see an edited item only on newly built topics; a program topic keeps its snapshot. A "content updated" marker is a possible later UI.
- A change that alters the *meaning* of the item (new prerequisite, different level) bumps a `major` flag in `change_note` and requires a path re-validation job (`AdjustPath` for affected profiles).

## 4. Authoring and import pipeline

```
repo JSON (human-reviewed PRs)  ──►  kb-import (idempotent)  ──►  kb_* tables
   curriculum/data/*.json              validates: schemas, ids, graph, parity across explanation languages
   kb/relations/<lang>.json            upserts items/versions by (language,key) and content_hash
   kb/sources.json                     writes an import report (added/changed/unchanged/deprecated)
```

- **Phase 1 keeps the JSON in the repo as the authoring source of truth** (git review, tests, no admin UI). The database is the *runtime* store; the embedded copy stays until the DB read path is proven by a parity test, then is removed. A content-admin UI is out of scope (D-008).
- Import runs as an operator command in the deploy (same pattern as `cmd/migrate backfill-accounts`) and at server start when the table is empty; it is transactional per item and resumable.
- **Parity test** (acceptance for the Rule Book): for every pair, every rule id and every topic, `GET /path/rules*` served from the DB equals the embedded-library response byte for byte.
- CI test: import into a fresh DB, assert counts equal the JSON (`85/91` topics, `45/51` rules x pairs), the prerequisite graph is acyclic, hard prerequisite levels are monotone.

## 5. Prerequisite authoring (the main content effort)

No prerequisites exist today. They must be authored for English (85 items) and Spanish (91).
Proposed method (needs owner approval, D-013): an **offline** drafting pass — a model proposes
edges from titles/theory/`related`/`tags`, a human reviews every edge in a PR — never at
runtime. Seed rules: same-unit grammar items follow the library order; a tense's negative/question
forms depend on its affirmative; vocabulary sets depend on no grammar; `contrasts_with` pairs
come from `Extra.mistakes` and `related`.

## 6. Question bank and exercise seeds

For assessment (`ASSESSMENT_ENGINE.md` section 5) and exercises, the KB also holds optional
**curated question items** linked to knowledge items and objectives, with a verified key.
They are the preferred source; AI-written items are the fallback. A curated item is just
`kb_items` kind `assessment_item` with `body = {format, prompt, options, answer, rationale}`
(add the kind in Phase 3). Seed from today's `forms/examples/mistakes` of each rule, which
already contain verified right/wrong pairs.

## 7. Language-specific structure

- Items are keyed per **studied** language; nothing in code may assume English or Spanish.
- A new studied language = new items + explanation-language versions for the languages it
  is offered in; a new explanation language = new `kb_item_versions` for existing items.
- Writing-system/phonology differences live in `pronunciation` items and optional `body`
  fields (accents, gender, script), not in code branches.
- Until a language has KB items, the existing AI-generated fallback applies (`LEARNING_ENGINE.md` section 2.2).

## 8. Open points

| # | Question | Where decided |
| --- | --- | --- |
| K-1 | Are lexemes items from the start or a Phase-6 addition? (cost: ~240 words x pair; benefit: word-level SRS) | D-010 |
| K-2 | Who owns content creation for C1/C2 and new pairs? | D-013 |
| K-3 | May the owner's notebook (`learning_entries`) feed `phrase` items? | D-010 |
