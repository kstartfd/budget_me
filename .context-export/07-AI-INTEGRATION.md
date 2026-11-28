# 07 — AI integration и gateway

**Новый Assessment contract (план):** общий generator принимает backend-selected
slots с навыком, band, objective, format и версией blueprint. Grammar prompt уже
работает; vocabulary, reading и audio форматы требуют отдельных валидаторов.
Writing/speaking нуждаются в версии рубрики, ограничении AI score и калибровке.
Транскрипт речи не является доказательством качества произношения. См.
`docs/learning/ASSESSMENT_MODES.md` §2, `AI_CONTRACTS.md` C-1/C-2.

## Реальная схема

BYOK: пользователь настраивает provider/model/effort/key через `GET|PUT|DELETE /learning/ai/settings`. Ключ шифруется в `internal/secretbox` (AES-256-GCM, привязка к user id) и хранится как ciphertext в `learning_ai_settings`; наружу отдаётся только безопасная подсказка. Модель/effort проверяются каталогом `internal/llm/catalog.go`. `internal/llm` реализует Anthropic (SDK) и OpenAI/DeepSeek/Kimi (HTTP); completion abstraction: `Client.Complete(ctx, Request) -> text`. Streaming chat профили в `internal/llm/chat.go` есть только для DeepSeek. Учёт каждого вызова по user/provider/model/purpose/tokens — `internal/learningusage` и `learning_agent_usage`. Реальный контракт: `docs/learning/AI_CONTRACTS.md`.

`internal/learningpath/commands/agent.go:Agent.completeJSON` читает настройки, открывает ключ, вызывает completion с константным system prompt и JSON user prompt, извлекает JSON object, unmarshals и запускает semantic validator. При ошибке формата/валидации добавляет объяснение к user prompt и повторяет **один** раз, не меняя system prompt. Возвращает явную ошибку после второго провала; command handler решает, что хранить. Backend, а не AI, присваивает ids, состояние, score/mastery/stop; исключение старого placement examiner документировано ниже.

## Промты и structured output, которые существуют

| Назначение | Prompt/validator | Выход и authority |
| --- | --- | --- |
| Legacy test questions | `commands/testgen.go:{testQuestionsSystem,testQuestionsPrompt,validateBatch}` | `questions[{skill,text,prompt,options[4],answer}]`; грамматика/словарь/reading, ключ сохраняется на сервере |
| Legacy examiner | `commands/testgen.go:{testJudgeSystem,testJudgePrompt,validateVerdict}` | `{verdict:climb|stop,level}`; AI **ещё решает** placement внутри guardrails |
| Quick Screening fallback | `commands/assessment_items.go:{screeningItemPrompt,screeningQuestions}` + `assessment.Validate` | `questions[{text,prompt,options,answers}]`; backend задаёт slots/уровень/item/format; неизвестные поля запрещены; AI не выдаёт verdict |
| Library plan | `commands/plan.go:{planPrompt,validateCatalogPlan}` | summary + `topics[{id,why}]`; ids должны входить в присланную библиотеку |
| No-library program | `commands/analyze.go:validateProgram` | AI пишет целые темы и теорию; сохраняется как отдельный source, риск качества |
| Topic exercises | `commands/exercise.go`, `commands/exercise_kinds.go` | набор `kind/text/prompt/options/answer/rule`; 9 видов, структурная проверка, версии в БД |
| Answer review | `commands/attempt.go:{ask,review}` | `{correct,mistakes,feedback}`; для свободного текста AI ещё определяет correctness, затем backend ограничивает результат |
| Rule Book practice | `commands/practice_prompt.go:{examplesSystem,practiceSystemFor,validateBlocks,cleanExamples}` | примеры и `form/choose/correct/make` drills; временный sitting |
| Notebook/chat | `internal/learning/commands/{fill,enrich,pack}.go`, `internal/learningchat/commands/{send,prompt}.go` | enrichment/pack или поток текста; chat summary/budgets |

Go structs и validators служат текущими structured schemas. Единого runtime JSON Schema registry, schema-versioned contracts для всех целей и golden fixture suite пока нет — это **проект** `learningaicontract` в `docs/learning/AI_CONTRACTS.md` §4–8. Quick Screening сохраняет opaque `assessment_version`, policy snapshot и `origin`; не распространять это на старые артефакты в описании.

## Ошибки, запись, ограничения

Нет настроенного agent → явная ошибка и переход к Settings; rejected key → повторный ввод; temporary outage/rate limit → 503/429; unreadable after retry → 502; зашифрованный ключ, который нельзя открыть → conflict. `internal/learningpath/controller.go:agentError`, `internal/llm` redact provider errors; AI-spending endpoints ограничены token bucket, чаты также бюджетами. Quick Screening при исчерпании банка и недоступном AI сохраняет partial/failed, не выдумывает ответ. Provider output не выполняется как SQL и не получает tools/DB access.

Сохранение: тестовые вопросы/ответы в `learning_path_test_*`; Quick Screening result в `assessment_result jsonb` и `learning_evidence`; программа в `learning_paths/topics`; упражнения в `learning_path_exercises` с generation history; Rule Book practice в истекающем `learning_rule_practice`; чат в `learning_chats/messages`; usage в `learning_agent_usage`. AI вызов делается до транзакции критического state transition, где применимо.

Границы: DeepSeek-only streaming chat, no-library AI-authored grammar, legacy AI examiner, AI correctness для free-text упражнений, отсутствие доказанного human agreement, отсутствие полноценного schema registry и per-contract golden tests. Целевой tutor evidence extractor P9 существует только как design (`docs/learning/AI_CONTRACTS.md` C-6), не как текущий код. Не включать секреты/ключи/пользовательские prompt payloads в экспорт.
