# 03 — Архитектурные решения

**Новые принятые решения 2026-11-26:** D-035 задаёт три режима через общий
Assessment Engine, D-036 требует проверять каждый навык подходящими заданиями и
не выдавать `unassessed` за уровень, D-037 требует независимо сохранять секции
Full и историю повторных Skill Assessment. D-034 остаётся решением о первом
грамматическом Quick milestone, не о постоянном ограничении продукта. Контракт:
`docs/learning/ASSESSMENT_MODES.md`; первичный журнал: `docs/learning/DECISIONS.md`.

Реестр решений: `docs/learning/DECISIONS.md` (это проектный журнал ADR, отдельной директории ADR нет). Здесь — актуальные последствия для внешнего архитектора; формулировки и статусы проверять в журнале.

## Принятые и действующие

| ID | Действующее правило |
| --- | --- |
| D-100 | Сохранить Go/Echo/sqlc/goose + Next.js/shadcn/ui + Postgres; замену стека отдельно согласовывать. |
| D-101 | Модульный монолит; без микросервисов и брокера. |
| D-102 | CEFR — справочная шкала, не структура доменной модели. |
| D-103 | Backend владеет состоянием, AI помогает и валидируется перед записью. Legacy examiner — текущий переходный случай. |
| D-104 | BYOK для каждого learner; provider-independent client. |
| D-105 | Правила/слова/цели библиотеки написаны и проверены людьми; AI подбирает, упорядочивает и пишет упражнения. |
| D-106 | Feature packages, прямые вызовы command/query handlers, sqlc как repository, `apperr`. |
| D-108 | Push в `main` запускает production deploy затронутого приложения; миграции идут при deploy. |
| D-109 | Версии инструментов закреплены; не использовать `:latest`/`@latest`. |
| D-110 | Backend tests пишутся вместе с feature/fix. |
| D-001 | Документация Learnings живёт в `docs/learning/`. |
| D-002 | Владелец делегировал агенту согласованные backend фазы; destructive/data deletion/stack/большие изменения требований требуют отдельного согласования. |
| D-003 | Этапная поставка: локальные gates, additive миграции, затем push/deploy и наблюдение. Решение обратимо. |
| **D-024** | Сначала тонкий сквозной цикл: rule → exercise → answer → check → attempt → evidence → state → review → dashboard. V1 реализован и проверен в production. Горизонтальные P1–P10 остаются. |
| **D-025** | Источник упражнения и режим практики независимы. `free` не хранит ответы/evidence и не меняет mastery/path/review; `tracked` хранит попытки и может влиять на них; `assessment` — формальная проверка. V1 хранит tracked path/review; Quick Screening добавляет assessment evidence. Tracked Rule Book ещё нет. |
| **D-026** | Отдельные сущности: Knowledge Item, Path Topic, Activity, Exercise, Study Session, Attempt, Evidence. Typed Activity (`learn`, `practice`, `review`, `apply`, `assess`) может принадлежать теме и завершается независимо от сессии. Сейчас материализуются `practice`/`review`. |
| **D-107** | Rule Book free practice — временная попытка без постоянной истории; через 10 мин после ухода sitting истекает. D-025 сохраняет это правило именно для `FREE_PRACTICE`, добавляя другие режимы. |
| D-027 | Wire enum values lower-case snake, несмотря на UPPER_CASE в продуктовом brief. |
| D-028 | Mastery vocabulary (`unknown` без row; `learning/practiced/mastered`, `relearning` как `learning` + `lapsed`); review state вычисляется из `due_at`. |
| D-029 | V1 item-review ladder 1/3/7/14/30 дней, не меняющий notebook SRS. Default применён, подтверждение владельца ещё ожидается. |
| D-030 | Evidence хранит независимые `source`, `practice_mode`, `ref_type/ref_id`, outcome, weight и assisted; `free` не создаёт row. |
| D-031 | `learning_activities` хранит тип/status/цель/item/topic/params; practice и review материализуются лениво. |
| D-032 | Для `mastered` нужно независимое доказательство: program practice с заранее доступным ключом максимум достигает `practiced`. |
| D-033 | Границы V1: без полного P1–P9, без replay старых attempts, без изменения старых API shapes. |
| D-034 | Quick Screening: versioned deterministic grammar heuristics, AI только для недостающего content; legacy placement/manual/final сохраняются. |

## Заменены/уточнены

- Прежнее «backend пишет только владелец» заменено D-002 для согласованных фаз.
- Прежний горизонтальный порядок плана дополнен D-024: V1 первым.
- Предложение исключить любую Rule Book practice из evidence заменено D-025; исключена только `free`.
- Предложение держать типы активностей только в session plan заменено D-026: activity хранится отдельно.
- D-016 (форма scheduler) уточнено D-029 (текущие default интервалы). Notebook ladder остаётся своей.
- D-107 не отменено; оно переименовано в правило `FREE_PRACTICE`.

## Открытые вопросы

`D-007` teacher как постоянный корень профиля; `D-008` JSON как источник авторского KB; `D-009` `text+CHECK`; `D-010` связь notebook/path; `D-011` семантика удаления истории и evidence; `D-012` роли/teacher viewer; `D-013` авторство prerequisite graph; `D-014` rollout flags; `D-015` лицензии; `D-016` долгосрочная модель scheduler; `D-017` общий bank AI вопросов; `D-018` заморозка extension API; `D-019` frontend тестовая инфраструктура; `D-020` production data shape; `D-021` масштаб языков; `D-022` legacy level для программы; `D-023` chat profiles иных провайдеров. `D-004` устаревший `CODING_STYLE.md`; `D-005` принято владельцем, корневой `AGENTS.md` создан; `D-006` package extraction. См. статусы и блокировки в оригинальном журнале, не считать рекомендуемые варианты одобренными.

## Практические ограничения

Миграции и API должны работать при blue-green с двумя версиями приложения; новые enum values могут сломать закрытые `z.enum` на frontend. Не удалять историю по умолчанию, не смешивать notebook с KB, не выдавать AI право на уровень/mastery, не передавать ключ ответа в независимую оценку. Все user-owned reads ограничиваются `user_id`; чужой ID отвечает 404. Источники: `docs/learning/ARCHITECTURE.md`, `API_CONTRACTS.md`, `VERTICAL_SLICE_V1.md`.
