package commands

import (
	"context"
	"time"

	"github.com/google/uuid"

	"budget_me/backend/internal/ai"
	"budget_me/backend/internal/clock"
	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
	"budget_me/backend/internal/insights/queries"
)

// systemPrompt fences the model in: it may only rephrase the numbers it is
// given. Everything it could get wrong — amounts, comparisons, limits — is
// already decided in Go.
const systemPrompt = `Ты — помощник в приложении личного бюджета.
Тебе дают ГОТОВЫЕ цифры за месяц. Твоя задача — коротко и по-дружески сказать,
что с ними не так и что сделать.

Правила:
- 2-3 предложения, обычный человеческий язык, обращение на «ты».
- Используй ТОЛЬКО те цифры, что даны. Не придумывай новых, не считай проценты сам.
- Про лимит категории говори РОВНО то, что стоит в скобках рядом с ней:
  «в пределах лимита», «БЛИЗКО К ЛИМИТУ» или «ЛИМИТ ПРЕВЫШЕН». Никогда не решай
  сам, близко ли трата к лимиту, и не называй лимит достигнутым, если такой
  пометки нет.
- Категории называй ровно так, как они названы в данных, ничего не переводя.
- Никаких вступлений, приветствий и списков — просто текст.
- Если всё в порядке — так и скажи, коротко.`

type GenerateAdvice struct {
	UserID uuid.UUID
	Facts  queries.Facts
}

type GenerateAdviceHandler struct {
	q     *db.Queries
	ai    *ai.Client
	clock clock.Clock
}

func NewGenerateAdviceHandler(q *db.Queries, client *ai.Client, clk clock.Clock) GenerateAdviceHandler {
	return GenerateAdviceHandler{q: q, ai: client, clock: clk}
}

// Handle writes the advice for a month and caches it. Callers run this in the
// BACKGROUND — the model needs seconds, and no screen waits on it.
func (h GenerateAdviceHandler) Handle(ctx context.Context, cmd GenerateAdvice) error {
	text, err := h.ai.Write(ctx, systemPrompt, cmd.Facts.Prompt())
	if err != nil {
		return err
	}

	return h.q.UpsertAdvice(ctx, db.UpsertAdviceParams{
		UserID:    ids.ToPg(cmd.UserID),
		Month:     cmd.Facts.Month,
		FactsHash: cmd.Facts.Hash(),
		Text:      text,
		Now:       db.Time(h.clock.Now()),
	})
}

// Stale reports whether the cached advice no longer describes these facts (or
// there is none yet) — the one condition worth paying the model for.
func Stale(cached db.AiAdvice, facts queries.Facts) bool {
	return cached.FactsHash != facts.Hash()
}

// BackgroundTimeout caps a background generation; it is not tied to the request
// that triggered it, which is long gone by then.
const BackgroundTimeout = 45 * time.Second
