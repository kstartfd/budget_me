# 10 — Roadmap и следующий этап

**Новый приоритет Assessment (D-035..D-037):** 1) общий движок/секции; 2)
grammar; 3) vocabulary; 4) reading; 5) полный Quick; 6) individual Skill;
7) Full orchestration; 8) writing; 9) listening; 10) speaking; 11)
pronunciation после acoustic validation. P3-14 в процессе: runtime blueprint,
grammar section result, pure section aggregator, capability API и gate V-007 готовы; секционная БД,
generic evaluator и остальные режимы ещё нет. Подробные зависимости P3-14..P3-25 — в
`docs/learning/IMPLEMENTATION_PLAN.md`.

Авторитетные источники: `docs/learning/IMPLEMENTATION_PLAN.md` (объём и зависимости), `docs/learning/IMPLEMENTATION_STATUS.md` (живой статус/verification log), `docs/learning/DECISIONS.md` (решения). Срез на 2026-11-26 включает незакоммиченный Quick Screening UI/backend.

| Фаза | Реальный статус | Выполнено / осталось |
| --- | --- | --- |
| V1 thin loop (D-024) | **VERIFIED production smoke** | 0036 migration, evidence/state/review/activity/dashboard и тесты; V-002/V-003 |
| P0 audit/baseline | **IMPLEMENTED**, hygiene открыт | docs/baseline V-001; P0-05 DB test repeatability, P0-06 contract runner, P0-07 migration harness, P0-09 production shape, P0-11 enum tolerance не завершены |
| P1 Knowledge Base | **NOT_STARTED** как полная фаза | V1 `kb_items` identity не равна versioned KB; нужен импорт JSON, relations/sources/read switch/parity |
| P2 Learner Profile | **NOT_STARTED** как полная фаза | Teacher уже есть, V1 evidence есть; server goals/skill estimates/backfill/rebuild/profile endpoints ещё нет |
| P3 Assessment | **IN_PROGRESS** | Quick Screening локально VERIFIED V-004, grammar only; остаются simulation/calibration, shadow vs legacy, item analysis, другие skill/purposes, D-022 |
| P4 Learning Path | **NOT_STARTED** как полная фаза | Legacy path и узкий screening planner есть; полный prerequisite DAG, deterministic builder, revisions/AdjustPath нет |
| P5 Exercise Engine | **NOT_STARTED** как полная фаза | Legacy generation/attempt и V1 evidence есть; скрытие ключа, reveal, item-aware rubric, mistake taxonomy нет |
| P6 Adaptive Learning | **NOT_STARTED** как полная фаза | V1 review ladder есть; full queue/state machine/mistake journal/rebuild simulation нет |
| P7 Study Session | **NOT_STARTED** | Есть 30-минутный timer, нет typed activity session planner/resume |
| P8 Dashboard/Analytics | **NOT_STARTED** как полная фаза | V1 endpoint и новый Overview UI есть; нет full analytics/insights/per-skill estimates |
| P9 AI Tutor integration | **NOT_STARTED** | Чат есть, нет context retrieval и validated tutor evidence |
| P10 Advanced | **NOT_STARTED** | Listening/speaking/pronunciation measurement, writing rubric, lexeme SRS и расширение языков |

## Приоритеты и зависимости

Владелец поставил ближайшие продуктовые задачи: (1) привести sidebar/навигацию к исходному Overview design и связать вкладки понятными задачами; (2) определить «начать с нуля» для тестового пользователя и затем реализовать безопасный пользовательский reset. Нынешний `POST /path/reset` только архивирует active path, а `DELETE /path/history` затрагивает часть истории; evidence/assessment и browser localStorage требуют отдельной семантики. Не запускать удаление данных без указания конкретного тестового аккаунта, среды и объёма.

Технический критический путь по плану: P0-05 (повторяемые DB тесты), P0-11 (двухэтапное расширение frontend enums), затем P1 KB import, P2 profile/evidence backfill, остальной P3 и P4, далее P5/P6/P7/P8/P9. V1 изменил порядок поставки, не отменил фазы. Graph prerequisites требует D-013 и авторской работы; backfills требуют D-020 (production shape) и D-011 (deletion semantics); полное assessment compatibility требует D-022; rollout flags D-014; item bank sharing D-017. Открытые решения перечислены в `03-ARCHITECTURE-DECISIONS.md`.

## Что проверить перед следующим изменением

Рабочее дерево содержит десятки modified/untracked файлов Quick Screening и нового UI; `main` HEAD `6440236`. Не считать их опубликованными и не перезаписывать. Перед кодовой правкой сверить текущий diff и тестовый gate. Переход в `VERIFIED` оформлять только с датой, commit, командами и результатом в `IMPLEMENTATION_STATUS.md`. Полный gate требует две свежие PostgreSQL БД, migrate, `go vet`, `go test`, eslint/tsc для frontend; браузерный flow отдельный. Инструкции: `docs/learning/TESTING_STRATEGY.md`.
