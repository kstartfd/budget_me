# 05 — Assessment Engine: реализация на 2026-11-26

**Новый целевой scope:** Quick = grammar + vocabulary + reading; Full = независимые
секционные grammar/vocabulary/reading/listening/writing/speaking, pronunciation
только после подтверждённой проверки аудио; Skill = выбранный навык с повторными
запусками и историей. Все три режима используют общий lifecycle, blueprint,
источники вопросов, валидатор, адаптацию, оценку, evidence, анализ пробелов и
агрегацию. Действующий Quick по-прежнему проверяет **только grammar**. В рабочем
дереве добавлены `assessment/modes.go`, pure multi-section aggregator
`assessment/orchestration.go`, `result.sections[]` для одной grammar
секции и capability endpoint, возвращающий лишь runnable en/es grammar Quick;
полные режимы этим не реализованы. Точная новая спецификация:
`docs/learning/ASSESSMENT_MODES.md`; решения D-035..D-037, gate V-007.

**Важно:** два действующих маршрута определения уровня сосуществуют. Legacy placement/final уже опубликованы. Новый `grammar-quick-v1` (Quick Screening) находится в незакоммиченном рабочем дереве, локально проверен по V-004 (`docs/learning/IMPLEMENTATION_STATUS.md`), production deploy для него не подтверждён. Код является источником истины, методология — `docs/learning/QUICK_SCREENING.md`.

## Точки входа после onboarding

Пятишаговый wizard (`apps/learning/frontend/src/features/home/components/wizard.tsx`, `finish`) создаёт teacher через `POST /learning/path/teachers`, сохраняет часть предпочтений в браузерном localStorage (`features/home/profile.ts`), затем выбирает сценарий:

1. `start=assess`, en/es: переход `/learning/assessment`; `AssessmentFlow` вызывает `api.startAssessment` → `POST /path/tests` с `purpose:"quick_screening"`. **Wizard сам не стартует Quick Screening.** Он готовит профиль и перенаправляет на отдельный assessment screen.
2. `start=assess`, другой язык: legacy `api.startPlacement` → `POST /path/tests` без purpose; требует ключ и доступный AI.
3. `start=manual`: `POST /path/levels` с `selfReported:true`, выбранный A1–C1. `start=scratch`: тот же endpoint с `A1`. Это claim, не измеренный уровень. Затем `/tests/:id/analyze` строит программу.

`features/home/setup.ts:useSetupStage`, `features/assessment/screen.tsx:AssessmentFlow` и `features/path/api.ts` обеспечивают возобновление через серверное состояние + browser memory. В Overview сохранился компонент `TestRunner` для legacy теста (`features/home/components/overview-screen.tsx:Runner`), в новой `/learning/assessment` — отдельный assessment flow. Активная программа блокирует новый placement/screening до завершения или reset.

## Quick Screening: blueprint и выбор вопросов

`internal/learningpath/assessment/blueprint.go` определяет `QuickScreening`, `BlueprintVersion=grammar-quick-v1`, `Target`, `Content`, `GradingKey`, `Question`, форматы и `Bank/Select/Validate/Grade`. `bank.txt` содержит **48 авторских вопросов en и 48 es**: по 6 rule targets на A1/A2/B1/B2, по 2 разных задачи на target. Формат строки: `language|band|Rule Book key|category|format|text|prompt|answers|options|partial answers`. Bank привязан к реальным `curriculum.Topic` с `Extra` (Rule Book), `Goal` становится learning objective. `Bank` проверяет item, skill, level и локализованную цель; `assessment/assessment_test.go` проверяет ссылочную целостность.

Backend задаёт band, category, item key, objective и format; `Select` предпочитает непоказанные вопросы, разные категории и менее использованные items. Fingerprint нормализованного `text+prompt` предотвращает повтор как в текущем тесте, так и в прошлом assessment history того же профиля (`commands/assessment_items.go:prepareAssessment`, SQL `queries/learning_assessment.sql:PriorAssessmentPrompts`). Choice options перемешиваются при выдаче. Version и origin (`curated|ai`) сохраняются с вопросом. Банк — grammar only. Поддержанные форматы для оценки: `multiple_choice`, `open_gap_fill`, `error_correction`, `sentence_transformation`, `vocabulary_in_context`, `reading_comprehension`; `writing/listening/speaking/pronunciation` перечислены в коде как будущие, но `Validate` их сейчас отклоняет.

AI вызывается только когда для выбранных backend slots кончились свежие авторские вопросы. `commands/assessment_items.go:screeningQuestions` передаёт `llm.PurposeTestQuestions`, константный system prompt `screeningItemPrompt` и JSON user prompt:

```json
{"language":"en","slots":[{"knowledgeItemKey":"...","category":"...","learningObjective":"...","skill":"grammar","difficultyBand":"A2","questionType":"open_gap_fill"}],"alreadyAsked":["..."]}
```

System prompt требует оригинальные constrained tasks в изучаемом языке, проще лексикой, строго по присланному objective/band/format; для choice — 4 разных варианта/один точный ключ, для gap/transformation — один `___`, для error correction — один отмеченный фрагмент. Ответ **только** `{"questions":[{"text":"...","prompt":"...","options":[],"answers":["..."]}]}` в порядке slots. AI не возвращает ID, category, level, verdict, evidence или другую authority. `diagnosticContent.UnmarshalJSON` запрещает неизвестные поля; `assessment.Validate` проверяет длины, формат, ключи, варианты и дубликаты; неправильный output получает одну попытку исправления через `Agent.completeJSON`. AI вопросы относятся только к конкретному тесту (D-017), вес evidence понижен до 70%.

`commands/assessment.go:StartAssessment` валидирует A1–B2, переиспользует открытый assessment данного teacher, сохраняет policy snapshot в `assessment_policy jsonb` и готовит первую batch. Настройки `ASSESSMENT_BATCH_SIZE` (3–6), `ASSESSMENT_MIN_QUESTIONS` (≥ две партии) и `ASSESSMENT_MAX_QUESTIONS` (до 60) считываются при старте; defaults 6/12/30 (`internal/config/config.go`, `assessment/policy.go:QuickPolicy`). Начальный band по умолчанию A2, может быть указан `startPoint` A1–B2. Объяснение берётся из teacher/Rule Book с fallback к языку обучения; `StartAssessment` не требует AI key, пока хватает bank.

## Проверка ответа и evidence

`commands/assessment_answers.go:answerAssessment` принимает 1–6 ответов на активную партию, блокирует test row, повтор точно такого же ответа идемпотентен, изменение записанного ответа — 409. `recordScreeningAnswer` восстанавливает private grading key из `assessment_alternatives`, сравнивает выбор с options и нормализует Unicode NFC, регистр, пробелы и апострофы для открытых ответов (`assessment/blueprint.go:Grade`). Результат: `correct`, `partial`, `incorrect`, `skipped`, `evaluation_unavailable`; partial = score 0.5. Ключ, correctness и feedback не уходят клиенту во время теста. Answer row и `learning_evidence` с `source=assessment`, `practice_mode=assessment`, `ref_type=assessment_attempt` пишутся в одной транзакции; item привязан к `kb_items`. Assisted/repeated/skipped/unavailable дают нулевой evidence weight; recognition = 0.4, recall/production = 1.0, AI-origin умножается на 0.7 (`assessment/policy.go:Observation.ScoreWeight`). **Assessment не двигает `learner_item_state` и review ladder непосредственно.**

## Difficulty, завершение, итоговый уровень

`assessment/policy.go:batchSignal` смотрит только полную batch (не меньше 3), минимум 2 категории и 2 решающих curated ответа. Для 6 вопросов ≥5 correct поднимает band; ≥4 incorrect опускает; иначе hold. Обобщённо: `ceil(5n/6)` correct или `ceil(4n/6)` incorrect для batch `n≥3`. Skips/partial/unavailable/assisted не дают решающего голоса. `Decide` ищет положительную band и отрицательную соседнюю band, либо два одинаково направленных сигнала на A1 floor/B2 ceiling. При конфликте не выдаёт достаточную границу. Для достаточного результата нужны ≥12 независимых usable ответов, ≥3 категории и подтверждённая/bracketed граница; иначе тест продолжается до лимита 30, исчерпания валидных вопросов или действия пользователя. `complete`/`cancel`/provider failure могут сохранить partial; начальный failure без попыток — `failed`. Pause/resume сохраняют состояние; подготавливаемая batch создаётся после commit ответов, поэтому обрыв не теряет принятые ответы. Методы: `commands/assessment.go:{AssessmentAction,finishAssessment}`, `assessment/policy.go:{Decide,Summarize}`.

`Summarize` записывает `assessment_result jsonb`: assessment/profile IDs, blueprint version, stop reason, optional difficulty range, sufficiency (`insufficient|preliminary|sufficient_for_screening`), conflict, grammar estimate/usable count, явно unassessed skills, per-item outcomes, strengths, gaps, recommended checks/objectives и evidence IDs. Per-item `demonstrated` требует 2 correct, включая recall/production и curated confirmation; strong gap — 2 независимых wrong с curated confirmation; один wrong — suspected gap. Это **предварительная сложность грамматических материалов**, не CEFR сертификат и не оценка остальных навыков. Legacy `levelReached` для планировщика = нижняя граница range, либо A1 при её отсутствии (`finishAssessment`); D-022 для полного P3 остаётся открытым.

## Как результат меняет Learning Path

`POST /tests/:id/analyze` создаёт build job. `commands/analyze.go` выбирает `analyzeScreening` для нового assessment, `commands/assessment_plan.go` берёт gaps/checks, не добавляет demonstrated rules, вставляет небольшой explicit prerequisite map (`assessment/blueprint.go:Prerequisites`), добирает упражнения для неоценённых навыков и учитывает goal/interests/hours. Ограничение плана ~6–14 topics. Затем `commands/build.go` сохраняет path/topics, prefetch exercises. Путь пока статичен; полноценные graph validation, revisions и `AdjustPath` — P4. Rule Book `related` links — навигационные, не prerequisite graph. Сам список explicit dependencies мал (10 en + 10 es rule entries), и это не завершённый KB graph.

## Legacy placement/final — совместимость

Без `purpose` `commands/test.go:Start/Answer` стартует с A1 или с разрешённого после final уровня. `commands/testgen.go:testQuestionsPrompt` даёт AI JSON `{learnedLanguage,nativeLanguage,kind,level,count,alreadyAsked}`; system prompt требует ровно 8 multiple-choice вопросов на placement (10 на final), 4 options, смесь grammar/vocabulary/reading, passage только для reading и gap `___` для grammar. AI отвечает `{"questions":[{"skill":"grammar|vocabulary|reading","text":"...","prompt":"...","options":["..."],"answer":"..."}]}`; `validateBatch` проверяет набор, длины, уникальность, ровно один совпадающий ключ. Сервер `placement.Grade` сравнивает answer. После полной партии `testJudgePrompt` отправляет AI все вопросы, ключи, ответы и correct flags; ожидает `{"verdict":"climb|stop","level":"A1..C2"}`. `validateVerdict`: climb только при ≥50% correct и том же уровне; stop не допускается при 100% correct и может поставить текущую или предыдущую ступень. Здесь AI всё ещё выбирает итог под guardrails. Финальный тест подтверждает уровень программы. Legacy вопросы не обязаны иметь Rule Book item ID; прошлые ответы нельзя надёжно приписать правилу. Вариант `POST /path/levels` с `selfReported:true` сохраняет claimed level без вопросов — это не assessment.

## Ограничения для проектирования

Quick Screening покрывает только grammar A1–B2 en/es; шесть основных форматов, конечные ключи, нет calibrated IRT/posterior, item analysis/quarantine, cross-learner bank или независимых estimates reading/listening/speaking. AI fallback не определяет уровень. Legacy AI examiner остаётся для совместимости. Исторические тесты без item reference не стали evidence автоматически. Достоверность продукта требует будущих simulation и human agreement measurements (`docs/learning/ASSESSMENT_ENGINE.md`, `IMPLEMENTATION_PLAN.md` P3).
