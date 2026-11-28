# 06 — Knowledge Base и Rule Book

**Зависимость нового Assessment scope:** формат `vocabulary_in_context` или
`reading_comprehension` внутри грамматического Quick bank не создаёт отдельный
vocabulary/reading skill result. Нужны лексические objectives и passage-linked
вопросы с ключами, покрытием и валидаторами до включения этих секций. См.
`docs/learning/ASSESSMENT_MODES.md` §2 и D-036.

**Состояние:** полноценная P1 DB Knowledge Base не реализована. Реальный источник учебного текста — JSON, встроенный в Go binary (`apps/learning/backend/internal/learningpath/curriculum/curriculum.go`, `data/*.json`). `kb_items` из миграции 0036 хранит пока только стабильную identity использованной темы. План развития: `docs/learning/KNOWLEDGE_BASE.md`, P1 в `IMPLEMENTATION_PLAN.md`.

## Состав и количество

| Пара изучаемый/язык объяснения | Тем A1–B2 | Grammar rules с `Extra` | A1/A2/B1/B2 grammar |
| --- | ---: | ---: | --- |
| `en_ru`, `en_en` | 85 на пару | 45 на пару | 10 / 11 / 12 / 12 |
| `es_ru`, `es_en`, `es_es` | 91 на пару | 51 на пару | 13 / 13 / 13 / 12 |

Это **локализации одного набора тем на язык**, не 45/51 разных правил в каждой локали. Для en/en и es/es не все словарные списки есть. Категория Rule Book в схеме не существует: темы имеют `unit` (локализованный раздел), `skill`, `tags`, CEFR из имени файла. В Quick Screening bank категории `verb_forms`, `noun_phrase`, `sentence_structure`, `clause_linking`; по 48 вопросов на язык, распределение соответственно **16/8/16/8** для en и es. Это категории проверки, **не** структурные категории всего Rule Book. Подсчёт выполнен по текущим `data/*_rules.json`, файлам level JSON и `assessment/bank.txt`.

## Формат и идентификаторы

`<learned>_<explained>_<level>.json` — массив `Topic {id,skill,unit,title,goal,theory,vocabulary[{word,translation}],tags}`; `Level` выводится из имени файла, не хранится в объекте. `<learned>_<explained>_rules.json` — словарь по тому же `topic.id` со `forms[]`, `examples[{text,translation}]`, `mistakes[{wrong,right,note}]`, `related[]`. `Extra != nil` превращает grammar topic в Rule Book entry. Ключи вида `a1-to-be` одинаковы между локализациями одного изучаемого языка; код `curriculum.Find/Rules/UpTo/Has` ищет по базовым BCP-47 кодам и проверяет пару. `curriculum_test.go` проверяет идентификаторы и зеркальность. `related[]` — навигационные ссылки, **не** педагогические prerequisites.

Источник текущего контента — авторские JSON в репозитории, проверяемые людьми (D-105); runtime AI не пишет каноническое правило для поддержанных пар. В prompt Rule Book practice упоминаются учебные пособия как стилистические ориентиры, это не означает, что текст JSON импортирован из них. Лицензирование внешних источников и их таблица остаются открытыми (D-015). Нет `kb_item_versions`, `kb_relations`, `kb_sources`, importer/DB read switch; полный target в `docs/learning/DATA_MODEL.md` §B.1.

## Использование

- **Rule Book:** `internal/learningpath/controller.go:{listRules,getRule}` и `queries.ReadHandler`; frontend `src/app/learning/path/rules`, `features/path/components/{rule-book,rule-card,rule-practice}.tsx`.
- **Примеры и drills:** `commands/practice.go` использует авторскую `RuleSubject` и AI prompts `commands/practice_prompt.go`. Примеры — пачка 5, максимум 20 на sitting; drills — 4 блока `form/choose/correct/make` по 5. Валидаторы проверяют длины, дубликаты и ключи; данные сидят в `learning_rule_practice` временно. Free practice по D-107 не пишет mastery/evidence.
- **Assessment:** `assessment/blueprint.go:Bank` связывает каждую авторскую задачу с `Topic`/`Extra`, берёт локализованный `Goal` как objective. AI fallback получает target, выбранный backend. В `kb_items` по `(language,key)` создаётся canonical identity, question хранит item UUID/key, snapshot objective/category/version; `learning_evidence` указывает item. Пока оценка касается 6 grammar targets на уровень, не всей KB.
- **Path:** поддержанные пары: `commands/plan.go:libraryFor/validateCatalogPlan` позволяют AI выбрать существующие ID; текст темы идёт из JSON. Новый screening plan использует gaps/checks и небольшой `assessment.Prerequisites` map. Неподдержанные пары/уровни выше B2 по legacy fallback получают полностью AI-authored program — известный риск качества.

Дальнейший P1: versioned authored KB в DB, schema validation, idempotent import, parity tests, связи, источники и read flag. P4 потребует полноценный, проверяемый DAG prerequisites; нынешние `related` и маленький screening map его не заменяют.
