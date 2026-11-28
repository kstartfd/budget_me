# 04 — Действующая модель данных

**Целевое дополнение, ещё не в БД:** D-037 требует `assessment_sections` с
независимым статусом, снимком policy/blueprint и результатом по навыку; вопросы
получат section reference, исторические Quick grammar попытки — backfill в одну
секцию. Миграции 0036/0037 остаются текущей схемой. Детали и ограничения
совместимости: `docs/learning/DATA_MODEL.md` A.8 и `ASSESSMENT_MODES.md` §5.

Источник истины: `apps/learning/backend/migrations/0001_*.sql` … `0037_adaptive_assessment.sql`, SQL в `apps/learning/backend/queries/`, generated Go types в `internal/db/models.go`. Миграция 0037 ещё в рабочем дереве. `docs/learning/DATA_MODEL.md` раздел A описывает baseline и добавления; раздел B — **проект**, не текущая схема.

## Сущности и ключевые поля (типы PostgreSQL)

| Сущность | Поля/типы и связь |
| --- | --- |
| `users` | `id uuid` PK; локальный user, связанный с account DB через `account_id uuid` (без межбазового FK); `telegram_id`, `email`, password hash. Данные/значения не экспортируются. |
| `learning_teachers` | `id uuid`, `user_id uuid` FK users, `language text`, `translation_language text`, `explain_in_target boolean`, `created_at timestamptz`; UNIQUE `(user_id,language)` и `(id,user_id)` (0036). Профиль языка, не преподаватель-человек. |
| `learning_path_tests` | `id/user_id uuid`, `path_id uuid?`, `kind enum(placement,final)`, `status enum(in_progress,done)`, language pair `text`, `level/level_reached text`, `passed bool`, timestamps. 0037 добавляет `assessment_profile_id uuid`, `assessment_type/status/start_band text`, `assessment_policy/result jsonb`, `assessment_batch int`, explanation language/stop reason. |
| `learning_path_test_questions` | FK test, `question_key text`, `level text`, position, skill, `text/prompt/answer text`, `options jsonb`; 0037 добавляет item UUID/key, category, objective, format, version, origin, fingerprint, batch, private alternative keys (`jsonb`), repeated flag. |
| `learning_path_test_answers` | PK `(test_id,question_key)`, `answer text`, `correct bool`; 0037: `assessment_attempt_id uuid UNIQUE`, `assessment_outcome text`, `assessment_assisted bool`. |
| `learning_path_jobs` | background build status `queued/running/done/failed`, test reference, preferences, output path/error. |
| `learning_paths` | `id/user_id uuid`, language pair/level/summary/source `text`, `status enum(active,finished,abandoned)`, `preferences jsonb`, placement test ID **без FK**, timestamps; partial UNIQUE one active `(user_id,language)`. |
| `learning_path_topics` | `id/path_id uuid`, `position int`, skill/unit/title/goal/theory/source_id `text`, vocabulary `jsonb`, status enum, timestamps. `source_id` — свободный текстовый ключ embedded library, не FK. |
| `learning_path_exercises` | `id/topic_id uuid`, position, `kind enum` (9 видов), text/prompt/answer/rule `text`, `options jsonb`, timestamp; `learning_path_exercise_versions` хранит `generation int` и позволяет сохранить прошлые attempts. |
| `learning_path_attempts` | `id/exercise_id/user_id uuid`, `answer text`, `correct bool`, `mistakes int`, `feedback text`, optional `lesson_id uuid`, timestamp. |
| `learning_path_lessons` | learner-owned 30-минутная сессия-таймер с active/finished, pause/resume и временем; не хранит план активностей. |
| `learning_rule_practice` | UNIQUE learner+rule, examples/exercises `jsonb`, `expires_at`; временная free practice без evidence. |
| `kb_items` (0036) | `id uuid`, `(language,key text)` UNIQUE, `kind text CHECK(grammar,topic)`, `skill/cefr_level/unit text`, timestamp. Только identity: текст по-прежнему в JSON. |
| `learning_evidence` (0036/0037) | `id/user_id/teacher_id/item_id/ref_id uuid`, `skill/source/practice_mode/ref_type/mode/outcome text CHECK`, `score/weight double precision [0,1]`, `assisted bool`, `attempt_no/hints_used smallint`, `occurred_at`; UNIQUE `(ref_type,ref_id)`; composite FK `(teacher_id,user_id)`; 0037 расширяет `ref_type` для assessment. |
| `learner_item_state` (0036) | PK `(teacher_id,item_id)`, `user_id uuid`, state `text CHECK`, weighted sums/score `double precision`, counts/stage/lapses `int/smallint`, UTC streak day `date`, last timestamps и `due_at timestamptz`. Derived from evidence; review state вычисляется при чтении. |
| `learning_activities` (0036) | `id/user_id/teacher_id uuid`, type/status/reason `text CHECK`, goal `text`, optional `item_id/topic_id uuid`, `params jsonb`, timestamps; partial UNIQUE one open review per item, one open practice per topic. |
| Notebook: `learning_topics`, `learning_entries`, `learning_topic_plans`, `learning_captures` | отдельные пользовательские темы/карточки; `learning_entries` содержит stage/due date для notebook SRS; capture queue использует idempotency key, lease/retry. |
| AI/chat: `learning_ai_settings`, `learning_agent_usage`, `learning_chats`, `learning_chat_messages`, `learning_chat_usage` | настройки провайдера и зашифрованный ключ; почасовой usage; диалоги/сообщения и дневные лимиты. Значения ключей/сообщений не экспортируются. |

## Прогресс и учебный контент

Прогресс программы: статус темы и попытки; доказательство знаний — append-only `learning_evidence`; производное состояние знаний и срок повтора — `learner_item_state`; конкретное действие — `learning_activities`. Assessment сохраняет исходные question/answer rows, metadata/result JSON и evidence с `ref_type=assessment_attempt`; **не** передвигает item mastery напрямую. Учебный контент: авторские `curriculum/data/<learned>_<explained>_<level>.json` и `_rules.json` внутри бинаря, `kb_items` только стабилизирует идентичность. `learning_path_topics` — snapshot выбранного материала; генерации exercises versioned.

Удаление: `POST /path/reset` архивирует active path; `DELETE /path/history` удаляет закрытые paths и descendants, но не связанное без FK evidence; удаление teacher/account каскадирует evidence/state. Следовательно существующий reset **не** возвращает всё состояние пользователя к нулю. Подробности: `docs/learning/DATA_MODEL.md` §A.5, решения D-011/D-025/D-026.
