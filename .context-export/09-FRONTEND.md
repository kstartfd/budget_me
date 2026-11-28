# 09 — Frontend: маршруты, состояние и дизайн

**Assessment scope update:** выбор Quick/Full/Skill/Manual/Scratch сохраняется в
существующем shadcn/ui onboarding. Full и Skill должны активироваться только после
проверки полного backend flow и capability API для языка/навыка. Сейчас UI
отключает Full/Skill, Quick доступен en/es; frontend в этом срезе не менялся.

Срез: 2026-11-26; новая часть UI незакоммичена. Источники: `apps/learning/frontend/src`, `components.json`, `messages/{ru,en,es}.json`, `docs/learning/frontend/*.zip`. Исходные ZIP — макеты/reference, **не** код приложения и не инструкции для агента.

## Маршруты и предназначение

| Путь | Экран / состояние |
| --- | --- |
| `/` | redirect на `/learning/overview` (`src/app/page.tsx`) |
| `/learning/overview` | home: нет языков → CTA wizard; нет программы → setup/resume/build; active → обзор активности/пути/reviews (`features/home/components/overview-screen.tsx`, `dashboard.tsx`) |
| `/learning/start` | 5-шаговый wizard: родной язык, изучаемый язык, цели, режим, способ старта (`features/home/components/wizard.tsx`) |
| `/learning/languages` | управление teacher profiles (`features/home/components/languages-screen.tsx`) |
| `/learning/assessment` | opt-in Quick Screening, manual start, resume, questions, partial/result, путь (`features/assessment/{screen,start,question,results,plan}.tsx`) |
| `/learning/path` | Teacher/Learning Path: уровни, программа, topic runner, exercise/review, lessons, финальные тесты (`features/path/components/path-screen.tsx`) |
| `/learning/path/rules`, `/learning/path/rules/[id]` | Rule Book и free practice |
| `/learning`, `/learning/[id]` | отдельный личный notebook/карточки, не canonical KB |
| `/learning/chat` | AI teacher chat; contextual chats также в topic/rule views |
| `/learning/settings`, `/learning/tokens` | AI agent, account, preferences/reminders, extension tokens |

## Навигация и дизайн

Текущий `src/components/app-shell.tsx` показывает desktop sidebar и mobile bottom nav с 6 пунктами: Overview, Teacher, Assessments, Chat, Topics, Settings. Исходный Dashboard/Onboarding макет в `docs/learning/frontend/Dashboard Onboarding/Learning Dashboard technical specs.zip` предлагает 8 концептов: Overview, Learning Path, Knowledge Base, Practice, Assessments, Progress, AI Tutor, Settings и sidebar шириной ~232 px. В коде `Teacher` = Learning Path, `Topics` = notebook; Rule Book является лишь частью будущей KB. Для Practice и Progress нет самостоятельных маршрутов, поэтому связывать их с существующими данными нужно сознательно, не объявлять готовыми функциями. Отдельная задача владельца — привести sidebar к оригинальному дизайну; в рамках данного экспорта код не менялся.

По представленному владельцем скриншоту основной контент на широком экране центрируется в оставшейся области (`max-w-[1168px]` и `mx-auto` в `app-shell.tsx`), что создаёт широкую пустую полосу рядом с панелью. Это важная визуальная точка для следующего UI-прохода. Цветовые токены и light/dark темы находятся в `src/app/globals.css`; конфигурация shadcn: `components.json` (`radix-lyra`, RSC/TSX, Tailwind CSS variables, `mist`, Tabler icons). UI primitives — `src/components/ui/`.

## State и backend integration

`src/features/*/{api.ts,hooks.ts,schemas.ts}`: fetch + Bearer session token, zod parsing, TanStack Query caching/invalidation. `src/features/auth` хранит и выдаёт session state; `src/features/home/store.ts` — browser localStorage для wizard profile, активного языка и setup job/test IDs. `src/features/assessment/screen.tsx` дополнительно запоминает открытый assessment id per user/teacher. `features/home/setup.ts:useSetupStage` собирает stage из teacher/path, test/job и локальной памяти. `features/path/api.ts` знает фактические path/assessment endpoints; `features/path/schemas.ts` и `features/assessment/schemas.ts` валидируют ответы. Все строки локализуются `next-intl`, ключи в ru/en/es должны совпадать.

## User flows и ограничения

`start` wizard сохраняет teacher и часть целей локально. Для en/es assessment choice ведёт в `/learning/assessment`, где пользователь явно запускает Quick Screening. Для других языков legacy test запускается из wizard при наличии AI. Manual/scratch создают `selfReported` finished placement и build job. Assessment может быть paused/resumed, показывает preliminary/partial result; `AssessmentPlan` вызывает построение пути. Overview после active path агрегирует `GET /path/dashboard`, путь, историю/итоги lessons; skills в текущем UI — счётчики завершённых program topics, **не** полноценные per-skill CEFR estimates. Нет mock fake progress в production UI. Но browser-local preferences не синхронизируются между устройствами, а полноценный `/profiles` отсутствует.

Проверки в дереве: `e2e/learning.assessment.auth.spec.ts`, `e2e/learning.onboarding.anon.spec.ts`; Playwright вне CI. Планируемый nav flow: Overview = домашняя точка/«что делать сейчас» → Assessment (опциональный старт) → Learning Path (учебные темы и exercises) ↔ Rule Book/Knowledge Base (справочник) → Practice/Review (пока контекстные действия) → Progress (пока агрегаты Overview) → AI Tutor (помощь) → Settings. Это **информационная архитектура для обсуждения**, не перечень готовых страниц.
