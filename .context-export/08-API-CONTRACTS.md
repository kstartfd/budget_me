# 08 — Действующие API контракты

**Изменение 2026-11-26:** действующий `GET /learning/path/tests/:id/results`
дополнен `sections[]` с одной grammar секцией (status, stopReason,
evidenceSufficiency, difficultyRange, usableEvidence, evidenceIds). Full/Skill
routes, section actions и history/comparison только запланированы;
не считайте их существующими endpoints. См. канонический
`apps/learning/backend/docs/learning-api.md` и `docs/learning/API_CONTRACTS.md`.

Новый session-only `GET /learning/path/assessments/capabilities?language=` возвращает
`{language,available:[{type,blueprintVersion,skills[]}]}`. Сейчас в en/es
`available` содержит только `quick_screening` с `grammar`; для иных языков пуст.

Основной индекс и ограничения: `docs/learning/API_CONTRACTS.md`. **Каноническая спецификация всех действующих endpoint/payload:** `apps/learning/backend/docs/learning-api.md`. Фактическая регистрация: `apps/learning/backend/internal/{server/server.go,learningpath/routes.go,learning/routes.go,learningai/routes.go,learningchat/routes.go}`. Quick Screening маршруты относятся к незакоммиченному дереву.

## Общие правила

Base `/api/v1`; Learnings `/api/v1/learning`. Ответ успеха `{"code":0,"message":"success","data":...}`, ошибка `{code,message,data:null,details?}`. JSON поля camelCase, UUID строками, время RFC 3339 UTC. `Authorization: Bearer <session>` для teacher/path/assessment/AI/chat. Notebook/captures допускают отдельные extension tokens по scope; **новые path endpoints session-only**. SQL ограничивает owner `user_id`; чужой id → 404. `Accept-Language` локализует ошибки. User-specific GET — `no-store`. Дополнительный request field должен быть optional, существующие поля не менять; закрытые zod enums frontend требуют двухэтапного rollout нового значения.

## Teacher и onboarding

| Метод | Request → response data | Факт |
| --- | --- | --- |
| `GET /learning/path/teachers` | без body → `Teacher[]`: `{id,language,translationLanguage,explainInTarget,explainLanguage,createdAt,path:{id,level,topicsDone,topicsTotal}|null}` | `learningpath/dto.go:teacherResponse` |
| `POST /learning/path/teachers` | `{language,translationLanguage}` → `Teacher` | one teacher/language, существующий возвращается |
| `PATCH /learning/path/teachers/:id` | `{explainInTarget:boolean}` → `Teacher` | изменяет язык объяснения, при необходимости будущие exercise sets |
| `DELETE /learning/path/teachers/:id` | path id → success | закрывает active program в историю; это **не** wipe всех данных языка |
| `POST /learning/path/levels` | `{language,translationLanguage,level,selfReported?:boolean}` → завершённый placement `PathTest` без вопросов | manual/scratch onboarding использует `selfReported:true`; A1–C1 |

Отдельного `/onboarding` или `/learner-profile` backend API **нет**. Wizard state/goals/skills/session/prefs находятся в localStorage (`frontend/src/features/home/profile.ts`). Программа получает subset через `POST .../analyze` body `{interests?,variant?,goal?,hoursPerWeek?}`.

## Assessment

| Метод | Request → response data |
| --- | --- |
| `POST /learning/path/tests` | legacy `{language,translationLanguage,level?}` или Quick Screening `{language,translationLanguage,purpose:"quick_screening",startPoint?:"A1"|"A2"|"B1"|"B2"}` → `PathTest` |
| `POST /learning/path/final-tests` | `{language,translationLanguage}` → `PathTest` для active program |
| `GET /learning/path/tests/:id` | `PathTest`: `{id,kind,language,translationLanguage,level,startLevel,levelReached,passed,status,questions,asked,correct,needed,createdAt,completedAt,assessment?}`; `questions[]` содержат `{key,skill,text,prompt,options}` без key ответа; в Quick Screening добавляются `type,knowledgeItemKey,learningObjective,category,version` |
| `POST /learning/path/tests/:id/answers` | `{answers:[{key,answer,skip?,assisted?}]}` → обновлённый `PathTest`; в Quick Screening 1–6 ответов, одинаковый повтор идемпотентен, изменение записанного → 409 |
| `POST /learning/path/tests/:id/assessment/:action` | `action=pause|resume|complete|cancel`, без body → `PathTest` |
| `GET /learning/path/tests/:id/results` | Quick Screening result после завершения; 409 пока не готов; `difficultyRange?`, `evidenceSufficiency`, `skillEstimates[]`, `unassessedSkills[]`, `knowledgeOutcomes[]`, gaps/strengths/checks/objectives, `evidenceIds[]` |

`assessment` в `PathTest` — optional `{type:"quick_screening",status,learnerProfileId,batch,answered,maxQuestions,stopReason?,result?}`. Top-level `kind`/`status` остаются legacy `placement|final` и `in_progress|done` для совместимости. Значения assessment lifecycle: `created/in_progress/paused/evaluating/completed/partial/failed/cancelled`. Schemas: `learningpath/dto.go:{questionResponse,testResponse}`, `learningpath/commands/assessment.go:AssessmentProgress`, `frontend/src/features/{path,assessment}/schemas.ts`. Методика: `05-ASSESSMENT-ENGINE.md`.

## Learning Path, exercise, activities и Overview

| Метод | Request → response data |
| --- | --- |
| `POST /learning/path/tests/:id/analyze` | `{interests?,variant?,goal?,hoursPerWeek?}` → `202` job `{id,testId,status,error,path?}`; повтор возвращает тот же job |
| `GET /learning/path/jobs/:id` | job `queued|running|done|failed`, при done содержит path |
| `GET /learning/path?language=` | active path или 404: `id,language,translationLanguage,level,summary,status,source,preferences,topics[]` с exercises counters; точный shape в `learning-api.md` и `frontend/src/features/path/schemas.ts` |
| `GET /learning/path/history`, `DELETE /learning/path/history?language=` | закрытые программы / `{deleted:n}`; DELETE удаляет path descendants, **не** evidence |
| `POST /learning/path/reset` | `{language}` → архивирует активную программу; не очищает assessment/evidence/mastery |
| `PUT /learning/path/goal` | `{language,hoursPerWeek}` → обновление недельной цели |
| `GET /learning/path/topics/:id` | topic с rule/words/exercises; `POST .../:id/exercises` генерирует или регенерирует набор; `POST .../:id/complete` закрывает тему |
| `POST /learning/path/exercises/:id/attempt` | `{answer,activityId?}` → attempt `{id,answer,correct,mistakes,feedback,...}`; `activityId` обозначает due review |
| `GET /learning/path/activities?language=` | `Activity[]` (`review` due, максимум 5; `practice` текущей темы), для review exercise **без** ключа/правила; response shape в `learning-api.md` |
| `GET /learning/path/dashboard?language=` | `{language,teacherId,items:{tracked,learning,practiced,mastered,lapsed},reviews:{due,overdue,upcoming,nextDueAt},evidence:{total,last7Days},activities:[]}` |
| `POST/GET /learning/path/lessons`, `GET /lessons/active`, `GET /lessons/totals`, `POST /lessons/:id/pause|resume|finish` | 30-минутный timer, история/итоги; не Study Session planner |

Полного `/learning/dashboard`, `/learning/profiles`, `/learning/kb/items` или `/review-queue` **нет**: они в `docs/learning/API_CONTRACTS.md` §5 помечены `PROPOSED`. Overview frontend агрегирует существующие `/path`, `/path/dashboard`, `/lessons` и `/teachers` (`frontend/src/features/home/components/dashboard.tsx`).

## Knowledge Base/Rule Book, notebook, AI/chat

- `GET /learning/path/rules?language=`, `GET /learning/path/rules/:id`: авторский Rule Book; `GET .../:id/practice`, `POST .../:id/practice/{examples,exercises,check,key,leave}` — временная free practice. Реальные формы — `learningpath/controller.go`, `commands/practice.go`, `learning-api.md` §Rule book.
- Notebook: `GET|POST /learning/topics`, `PUT|DELETE /learning/topics/:id`, `GET|POST /learning/entries`, `GET|PUT|DELETE /learning/entries/:id`, review/sets/pack/captures/audio/scan routes. Extension permissions различаются, подробности в canonical reference §Find a topic–Queue operation.
- AI: `GET|PUT|DELETE /learning/ai/settings`, `POST /learning/ai/complete`, `POST /learning/ai/scan`; chat endpoints для entries/topics/path topics/rule examples и dialogs. Токены расширения управляются `/learning/tokens`. У chat есть SSE `delta/done/error`.

Коды ошибок и auth matrix — `docs/learning/API_CONTRACTS.md` §§1,4. Для будущей синхронизации API использовать `learning-api.md` как детальную спецификацию, а миграции/DTO как проверку фактического shape.
