# Knowledge inventory (Task 01)

Snapshot: 2026-11-26 working tree, including uncommitted assessment code. This is an inventory of **runtime content**, not the target P1 database design. [Documentation index](../learning/README.md) · [KB design](../learning/KNOWLEDGE_BASE.md) · [status](../learning/IMPLEMENTATION_STATUS.md).

**Later naming update:** The identity table described below is physically
`learning_knowledge_items` after [migration 0038](../../apps/learning/backend/migrations/0038_learning_knowledge_items_name.sql);
`kb_items` remains an updatable view for older application versions. Content
and item counts are unchanged.

## Source, identity and coverage

[The `curriculum` loader](../../apps/learning/backend/internal/learningpath/curriculum/curriculum.go) embeds `data/*.json`. Four level files per pair (`A1`, `A2`, `B1`, `B2`) contain lesson topics. A fifth `<learned>_<explanation>_rules.json` maps grammar topic IDs to Rule Book extras. The loader attaches an `Extra` only when an ID matches a topic; [Rule Book reads](../../apps/learning/backend/internal/learningpath/routes.go) use these entries. Identical IDs in explanation-language variants represent the same learned-language topic, so pair counts below must **not** be summed as distinct knowledge concepts.

| Pair (learned → explanation) | All topics | Grammar / rules | Vocabulary | Reading | Writing | Listening | Speaking |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| English → English (`en_en`) | 85 | 45 / 45 | 12 | 8 | 4 | 8 | 8 |
| English → Russian (`en_ru`) | 85 | 45 / 45 | 12 | 8 | 4 | 8 | 8 |
| Spanish → Spanish (`es_es`) | 91 | 51 / 51 | 12 | 8 | 4 | 8 | 8 |
| Spanish → English (`es_en`) | 91 | 51 / 51 | 12 | 8 | 4 | 8 | 8 |
| Spanish → Russian (`es_ru`) | 91 | 51 / 51 | 12 | 8 | 4 | 8 | 8 |

Counts were computed from the 20 level JSON files and five rules maps in [embedded data](../../apps/learning/backend/internal/learningpath/curriculum/data). English has 85 distinct topic IDs and Spanish 91. Each language's two or three explanation variants have matching counts; no rule-map ID is orphaned. Other language pairs can use AI-authored legacy programs, but have **no canonical embedded library** or Quick Screening bank.

`Topic.ID` is a stable text key within the learned language, usually prefixed with the level (for example `b1-present-perfect`). The [lazy `kb_items` row](../../apps/learning/backend/migrations/0036_learning_loop.sql) gives `(language, key)` a unique database UUID; it stores `kind`, `skill`, `cefr_level`, and `unit`, **not** the rule body or a content version. A program topic stores its library ID in `learning_path_topics.source_id` as text; it has no `kb_item_id` foreign key yet ([Analyze.Handle](../../apps/learning/backend/internal/learningpath/commands/analyze.go), [P1 backfill plan](../learning/IMPLEMENTATION_PLAN.md)).

## English grammar categories and IDs

These are the complete 45 `skill=grammar` topics in `en_en`; `en_ru` has the same identifiers and category counts. `unit` is the current category label, rather than a first-class taxonomy. The table is generated directly from [English level files](../../apps/learning/backend/internal/learningpath/curriculum/data) and checked against both English Rule Book maps.

| CEFR | Category (`unit`) | Knowledge item ID | Topic title |
| --- | --- | --- | --- |
| A1 | Sentence basics | `a1-to-be` | The verb to be: am, is, are |
| A1 | Sentence basics | `a1-articles` | Articles: a, an, the |
| A1 | Sentence basics | `a1-plurals` | Plural nouns |
| A1 | Sentence basics | `a1-demonstratives-possessives` | This / that / these / those and my, your, his |
| A1 | Present tenses | `a1-present-simple` | Present Simple: statements |
| A1 | Present tenses | `a1-present-simple-questions` | Present Simple: negatives and questions |
| A1 | Describing places | `a1-there-is` | There is / there are |
| A1 | Modal verbs | `a1-can` | Can and can't: ability and permission |
| A1 | Sentence basics | `a1-have-got` | Have got / has got: what you have |
| A1 | Describing places | `a1-prepositions` | Prepositions in, on, at |
| A2 | The past | `a2-past-simple-regular` | Past Simple: regular verbs |
| A2 | The past | `a2-past-simple-irregular` | Past Simple: irregular verbs and to be |
| A2 | The past | `a2-past-simple-questions` | Past Simple: negatives and questions with did |
| A2 | Present tenses | `a2-present-continuous` | Present Continuous: happening now |
| A2 | Present tenses | `a2-present-simple-vs-continuous` | Present Simple or Present Continuous |
| A2 | The future | `a2-going-to` | Be going to: plans and intentions |
| A2 | The future | `a2-will` | Will: decisions, predictions, promises |
| A2 | Describing and comparing | `a2-comparatives` | Comparatives and superlatives |
| A2 | Nouns | `a2-quantifiers` | Countable and uncountable: some, any, much, many |
| A2 | Modal verbs | `a2-have-to-should` | Have to, must, should: duty and advice |
| A2 | Describing and comparing | `a2-adverbs` | Adverbs of manner and frequency |
| B1 | Perfect tenses | `b1-present-perfect` | Present Perfect: experience and result |
| B1 | Perfect tenses | `b1-present-perfect-vs-past` | Present Perfect or Past Simple |
| B1 | The past | `b1-past-continuous` | Past Continuous and Past Simple in a story |
| B1 | Conditionals | `b1-first-conditional` | First conditional |
| B1 | Conditionals | `b1-second-conditional` | Second conditional |
| B1 | The passive | `b1-passive` | The passive: Present and Past Simple |
| B1 | The past | `b1-used-to` | Used to and would: past habits |
| B1 | Complex sentences | `b1-relative-clauses` | Relative clauses: who, which, that |
| B1 | Complex sentences | `b1-reported-speech` | Reported speech: telling what was said |
| B1 | Modal verbs | `b1-modals-deduction` | Must, might, can't: making guesses |
| B1 | Verb forms | `b1-gerund-infinitive` | Gerund and infinitive after verbs |
| B1 | Complex sentences | `b1-linking-words` | Linking words: although, however, because, so |
| B2 | Perfect tenses | `b2-present-perfect-continuous` | Present Perfect Continuous |
| B2 | Perfect tenses | `b2-past-perfect` | Past Perfect: what happened before the past |
| B2 | The future | `b2-future-perfect-continuous` | Future Continuous and Future Perfect |
| B2 | Conditionals | `b2-third-conditional` | Third and mixed conditionals |
| B2 | Conditionals | `b2-wish` | Wish and if only: regret and wanting change |
| B2 | The passive | `b2-passive-advanced` | The passive in all tenses and impersonal structures |
| B2 | Complex sentences | `b2-reported-advanced` | Reported speech: questions, requests and reporting verbs |
| B2 | Complex sentences | `b2-relative-advanced` | Non-defining relative clauses and participle clauses |
| B2 | Modal verbs | `b2-modals-past` | Modal verbs in the past: should have, could have |
| B2 | Verb forms | `b2-gerund-infinitive-meaning` | Gerund and infinitive: a change in meaning |
| B2 | Words and phrases | `b2-phrasal-verbs` | Phrasal verbs |
| B2 | Words and phrases | `b2-discourse` | Linking words: moreover, nevertheless, whereas |

Category totals: Complex sentences 5, Conditionals 4, Describing and comparing 2, Describing places 2, Modal verbs 4, Nouns 1, Perfect tenses 4, Present tenses 4, Sentence basics 5, The future 3, The passive 2, The past 5, Verb forms 2, Words and phrases 2.

## Content shape and relationships

- Each level-file topic has `id`, `skill`, `unit`, `title`, `goal`, `theory`, optional `vocabulary` (word/translation) and `tags`; `Level` is inferred from the filename, not stored in the topic JSON ([Topic](../../apps/learning/backend/internal/learningpath/curriculum/curriculum.go)). `goal` is a human-authored learning objective. The Rule Book `Extra` has `forms`, translated `examples`, common `mistakes`, and `related` IDs. There are 91 English and 102 Spanish `related` link occurrences **per pair**; they are navigational, not prerequisite edges.
- There is **no canonical prerequisite graph** in the content. Quick Screening has a separate, hard-coded map of ten prerequisite targets for English and ten for Spanish, used by its gap recommendations and path planner ([Prerequisites](../../apps/learning/backend/internal/learningpath/assessment/blueprint.go)). It does not cover the full library or carry authorship/version metadata.
- CEFR metadata is one A1–B2 band per file/topic and copied into lazy `kb_items.cefr_level`; Rule Book examples do not have separate CEFR or difficulty metadata. No C1/C2 authored catalog is embedded.
- Rule explanations, objectives and example/mistake lists are static, human-curated JSON. [Free Rule Book practice](../../apps/learning/backend/internal/learningpath/commands/practice.go) can ask AI for additional examples and drills; [program exercises](../../apps/learning/backend/internal/learningpath/commands/exercise.go) are AI-generated and persisted against a program topic. Generated text does not replace the canonical rule body.
- The current JSON has no source citation, licence field, editorial revision, published version, effective date, or content migration history. Git revisions are the only version history of Rule Book content. Individual assessment **questions** receive an `assessment_version` UUID when issued; this is not a Rule Book content version ([migration 0037](../../apps/learning/backend/migrations/0037_adaptive_assessment.sql)). Source ownership and versioning remain open at [D-008, D-013 and D-015](../learning/DECISIONS.md).
- The [Quick bank](../../apps/learning/backend/internal/learningpath/assessment/bank.txt) has 48 prewritten questions per language, mapped to 24 grammar rule IDs each. It covers only a subset of the 45/51 rules; generated fallback questions inherit the backend-selected canonical ID. A `reading_comprehension` or `vocabulary_in_context` **format** in this bank still measures grammar, not an independent reading or vocabulary skill ([Bank/Select](../../apps/learning/backend/internal/learningpath/assessment/blueprint.go)).

## Connection to programs, exercises and progress

[Legacy catalog planning](../../apps/learning/backend/internal/learningpath/commands/plan.go) lets AI choose existing topic IDs and validates that choice before copying content into a path. [Quick planning](../../apps/learning/backend/internal/learningpath/commands/assessment_plan.go) deterministically chooses gap/check IDs and the small prerequisite map, then adds unassessed-skill topics. [Exercise generation](../../apps/learning/backend/internal/learningpath/commands/exercise.go) uses the copied topic; [attempt evidence](../../apps/learning/backend/internal/learningpath/commands/loop.go) resolves `source_id` back to the library and lazily upserts `kb_items`. Rule Book free practice is transient and writes no evidence ([D-107/D-025](../learning/DECISIONS.md)).

The user preference to store **item-generation specifications/prompts** instead of relying on reusable fixed question wording is a proposed next-stage product decision, not the present implementation. Such specifications should retain canonical item ID, skill, CEFR target, task format, evaluation key policy, and content provenance. Newly generated variants would need validation, exposure tracking, and calibration before they can replace the current bank's measurement role. See [gap analysis](ASSESSMENT_GAP_ANALYSIS.md).
