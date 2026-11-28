# 02 — Текущее состояние

**Assessment update 2026-11-26:** по V-006 прошли migration 0037, `go vet ./...`,
`go test -count=1 ./...`, ESLint и TypeScript на двух одноразовых PostgreSQL.
Код получил общий runtime blueprint (`assessment/modes.go`), pure aggregator
(`assessment/orchestration.go`), `result.sections[]` для grammar и read-only
capability endpoint. Полный backend gate V-007 прошёл после подключения aggregator.
Это первый срез P3-14; vocabulary, reading как самостоятельные оценки, Full и
Skill Assessment по-прежнему не реализованы. Целевой scope — D-035..D-037.

Срез: 2026-11-26, код `6440236` + незакоммиченные изменения. Статусы ниже различают **production verified**, **локально проверено в рабочем дереве** и **план**. Детальный журнал: `docs/learning/IMPLEMENTATION_STATUS.md`.

## Реализовано

- **Teacher / профиль языка:** один `learning_teachers` на `(user_id, language)`, язык объяснения и флаг объяснения на изучаемом языке. Управление через `/learning/path/teachers`. Полный серверный Learner Profile ещё не построен.
- **Legacy placement/final:** AI пишет партии вопросов, сервер сохраняет вопросы/ответы и проверяет вариант, AI examiner с guardrails решает дальнейший подъём или итоговый уровень. Финальный тест закрывает программу. `internal/learningpath/commands/{test.go,testgen.go}`.
- **Learning Path:** background job, выбор 6–14 проверенных тем из авторской JSON-библиотеки для поддерживаемых пар; fallback AI-authored для остальных. Версии упражнений, попытки, 30-минутные lesson timers, Rule Book. `internal/learningpath/commands/{analyze.go,build.go,exercise.go,attempt.go}`.
- **V1 loop (production verified):** `kb_items`, `learning_evidence`, `learner_item_state`, `learning_activities`; попытка → evidence → mastery state → review due; минимальный dashboard/activities API. `migrations/0036_learning_loop.sql`, `internal/learningprofile`, `internal/learningpath/commands/loop.go`.
- **Notebook и чат:** карточки с собственным Leitner SRS, capture queue, AI enrich/pack, OCR/TTS, чат и BYOK AI. Это отдельная область относительно path. `internal/learning`, `internal/learningchat`, `internal/llm`.

## Частично реализовано в незакоммиченном дереве

- **Quick Screening (P3):** миграция `0037_adaptive_assessment.sql`, grammar bank по 48 вопросов для en/es, deterministic bounded policy, private keys, durable pause/resume/cancel/partial, per-answer evidence, результат с gaps/checks и детерминированный стартовый план. Локальная проверка V-004 описана в status; этот срез не является подтверждением production deploy. Источники: `internal/learningpath/assessment/`, `commands/assessment*.go`, `docs/learning/QUICK_SCREENING.md`.
- **Onboarding/Overview/Assessment frontend:** новые страницы `/learning/start`, `/learning/languages`, `/learning/overview`, `/learning/assessment`, wizard, dashboard, отдельный assessment flow, локализация и e2e. `src/features/home`, `src/features/assessment`, `src/components/app-shell.tsx`. Wizard prefs в localStorage, а серверу передаются только данные существующих teacher/test/build контрактов.
- **KB:** `kb_items` пока даёт стабильную идентичность на первом использовании; текст остаётся в embedded JSON. Частичное описание P1, не полноценная DB Knowledge Base.
- **Adaptive/mastery:** только V1 ladder и state rules; полноценные P2/P6/P8 не сделаны.

## Планируется

P0 hygiene (повторяемость DB тестов, frontend enum tolerance), P1 DB Knowledge Base и импорт, P2 серверный Learner Profile и backfill evidence, остаток P3 calibrated multi-skill assessment/shadow/item analysis, P4 graph-based Path и revisions, P5 item-aware Exercise Engine/hidden keys/rubrics, P6 Adaptive Learning, P7 sessions, P8 analytics, P9 context-aware AI Tutor, P10 advanced skills. План: `docs/learning/IMPLEMENTATION_PLAN.md` §§4–5.

## Известные проблемы и расхождения

| Факт | Последствие / источник |
| --- | --- |
| `CURRENT_STATE.md` и начало `DATA_MODEL.md` описывают исходный аудит `4662ad7`/миграции 0001–0035; позже в тех же документах добавлены V1/0037 | Читать их как исторический baseline + дополнения, сверять с миграциями 0036/0037 |
| `ARCHITECTURE.md` §3 и некоторые формулировки `CURRENT_STATE.md` всё ещё говорят «evidence none» или «dashboard planned» | Это устаревшее описание до V1; V1 уже в production |
| `IMPLEMENTATION_STATUS.md` помечает P1/P2/P4–P10 `NOT_STARTED`, хотя V1 реализовал узкие срезы | Статус относится к полным фазам, а не к отсутствию всех сущностей |
| Исходный аудит зафиксировал отсутствие `AGENTS.md` | Корневой `AGENTS.md` создан в этой сессии; live index, architecture и D-005 синхронизированы, исторический `CURRENT_STATE.md` остаётся снимком прежней даты |
| Программные упражнения отдают ключ до ответа | Положительный ответ является assisted evidence; сам по себе не даёт mastery |
| Wizard goals/session/prefs — localStorage | Нет cross-device серверного профиля; удаление истории/teacher не является полным «с нуля» |
| DB тесты повторно на одной БД могут упасть на дневном лимите чата (TD-03); e2e вне CI (TD-04) | Для проверки брать свежие БД; `docs/learning/TESTING_STRATEGY.md` |
| Нет полной prerequisite graph, C1/C2 контента, calibrated estimates; production data shape не измерена (R-01), закрытые frontend enums риск rollout (R-07) | Ограничивает P1–P4 и безопасный deploy |

Текущий продуктовый запрос: привести боковую навигацию к исходному Overview design, определить роли вкладок и спроектировать управляемый reset/start-from-zero. Код и пользовательские данные в ходе подготовки этого пакета не менялись.
