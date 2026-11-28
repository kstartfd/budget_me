# 01 — Контекст проекта Learnings

**Обновление 2026-11-26:** целевая архитектура Assessment расширена до Quick,
Full и Skill через один движок (D-035..D-037). Первый кодовый шаг — проверяемый
реестр runtime blueprint и результат грамматической секции. Подробности и граница
между кодом и планом: `docs/learning/ASSESSMENT_MODES.md` и файл 05 этого пакета.

Срез: 2026-11-26, `main` / `6440236` плюс незакоммиченные изменения. Пути ниже указаны от корня репозитория `budget_me`. Коммитный V1 проверен в production; Quick Screening, новый onboarding и Overview находятся в рабочем дереве. Полный навигатор документации: `docs/learning/README.md`; краткая карта для агентов: `docs/PROJECT_CONTEXT.md`.

## Стек

| Слой | Реальная реализация |
| --- | --- |
| Backend | Go 1.25, Echo v4, pgx v5, sqlc, goose; `apps/learning/backend/go.mod`, `internal/server/server.go` |
| Данные | PostgreSQL 16: отдельная БД Learnings и БД `account_app` для общей учётной записи; `apps/learning/backend/migrations/`, `queries/` |
| Frontend | Next.js 16 App Router, React 19, TypeScript 5.9, TanStack Query, zod, next-intl, Tailwind 4, shadcn/ui/Radix, Playwright; `apps/learning/frontend/package.json` |
| AI | Собственный клиент `internal/llm`, ключ провайдера принадлежит пользователю; Anthropic/OpenAI/DeepSeek/Kimi; `internal/learningpath/commands/agent.go` |
| Дополнительно | Telegram Mini App, браузер, расширение Chrome, OCR через Tesseract, TTS с файловым кэшем, Telegram reminders |

## Архитектура и границы

Модульный монолит: один Go API и один Next.js frontend. Backend организован по функциям: `controller.go` и `routes.go` принимают HTTP, `commands/` изменяют состояние, `queries/` читают, `internal/db` — сгенерированный sqlc репозиторий. Контроллеры вызывают handlers напрямую. `internal/server/server.go` собирает зависимости. Транзакция охватывает попытку, evidence и состояние item; AI вызывается до открытия такой транзакции. Пакеты и реальные соглашения: `docs/learning/ARCHITECTURE.md` §2.

| Граница | Где сейчас | Что хранит/решает |
| --- | --- | --- |
| Identity | `libs/account`, `internal/auth`, `internal/accountlink` | Telegram/email login, локальный `users`, удаление аккаунта; account DB отдельно |
| Notebook | `internal/learning` | личные карточки, capture queue, отдельный Leitner SRS; не тождественен Learning Path |
| Teacher + path | `internal/learningpath` | язык, тесты, задания, программа, упражнения, Rule Book, V1 loop |
| Quick Screening | `internal/learningpath/assessment`, `commands/assessment*.go` | deterministic grammar screening; рабочее дерево |
| Evidence/mastery | `internal/learningprofile`, `commands/loop.go`, `queries/loop.go` | append-only evidence, item state, due review |
| Chats/AI | `internal/learningchat`, `internal/learningai`, `internal/llm` | чат, настройки провайдера, completion/streaming, учёт токенов |
| Frontend | `src/app/learning`, `src/features/{home,assessment,path,learning,chat,ai,...}` | страницы, сценарии, fetchers, zod, Query cache |

Не путать `learning_topics` (личный словарь) с `learning_path_topics` (раздел персональной программы). Их связывает только код языка и отдельные пользовательские действия; решение об объединении открыто (D-010). Подробнее: `docs/learning/CURRENT_STATE.md` §4.

## Репозиторий и инфраструктура

`apps/learning/backend` содержит `cmd/http_server`, `cmd/migrate`, `cmd/remind`, `internal`, `migrations`, `queries`, API reference. `apps/learning/frontend` содержит `src/app` (маршруты), `src/features` (функции), `messages/{ru,en,es}.json`, `e2e`. `apps/budget` — другое приложение. `libs/account` общий. `infra/learning`, `infra/edge` и `.github/workflows` определяют Compose/Caddy/GitHub Actions. CI собирает изменённое приложение, хранит образ в GHCR и развёртывает blue-green; push в `main` по затронутым путям запускает production deploy. Перед миграцией создаётся backup; старый и новый API временно работают параллельно. Подробности: `infra/learning/README.md`, `infra/budget/CI_CD.md`, `docs/learning/ARCHITECTURE.md` §7.

## Правила для продолжения

Источник истины о фактической реализации — код и миграции; для payload — `apps/learning/backend/docs/learning-api.md`; для планов — `docs/learning/IMPLEMENTATION_PLAN.md`; для статуса — `docs/learning/IMPLEMENTATION_STATUS.md`. Открытые проектные решения не считать реализованными. `apps/learning/backend/CODING_STYLE.md` — устаревший шаблон. Корневой `AGENTS.md` создан в этой сессии и содержит краткие рабочие правила.
