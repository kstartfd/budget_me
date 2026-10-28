package queries

import (
	"context"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningchat/commands"
	"learning_app/backend/internal/llm"
)

type ChatHandler struct {
	q     *db.Queries
	clock clock.Clock
}

func NewChatHandler(q *db.Queries, clk clock.Clock) ChatHandler {
	return ChatHandler{q: q, clock: clk}
}

type Chat struct {
	// Available is false when there is no agent to talk to: none chosen, or a
	// provider chat is not built for. The conversation already held stays
	// readable either way.
	Available bool
	// Teacher is the user's choice to have the agent teach the language rather
	// than just answer; it only matters while the chat is available.
	Teacher  bool
	Messages []db.LearningChatMessage
	// Budget is what the chat may still spend; nil while it is not available.
	Budget *commands.Budget
	// Daily is what the user may still spend today across all chats; nil while
	// the chat is not available.
	Daily *commands.Usage
}

// Get returns the conversation of an entry, empty until the first question.
func (h ChatHandler) Get(ctx context.Context, userID, entryID uuid.UUID) (Chat, error) {
	_, err := h.q.GetLearningEntry(ctx, db.GetLearningEntryParams{ID: ids.ToPg(entryID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return Chat{}, apperr.NewNotFound("entry not found")
	}

	if err != nil {
		return Chat{}, err
	}

	out := Chat{Messages: []db.LearningChatMessage{}}

	var profile llm.ChatProfile

	settings, err := h.q.GetLearningAISettings(ctx, ids.ToPg(userID))
	if err != nil && !db.IsNotFound(err) {
		return Chat{}, err
	}

	if err == nil {
		profile, out.Available = llm.ChatProfileOf(llm.Provider(settings.Provider))
		out.Teacher = out.Available && settings.Teacher
	}

	chat, err := h.q.GetLearningChatByEntry(ctx, db.GetLearningChatByEntryParams{EntryID: ids.ToPg(entryID), UserID: ids.ToPg(userID)})
	if err != nil && !db.IsNotFound(err) {
		return Chat{}, err
	}

	if out.Available {
		// A chat not started yet has the whole budget; its zero row says so.
		budget := commands.BudgetOf(chat, profile, h.clock.Now())
		out.Budget = &budget

		// Only the user's own count is shown: the key's would need the key opened.
		daily, err := commands.LoadDaily(ctx, h.q, userID, "", profile, h.clock.Now())
		if err != nil {
			return Chat{}, err
		}

		out.Daily = &daily.User
	}

	if db.IsNotFound(err) {
		return out, nil
	}

	out.Messages, err = h.q.ListLearningChatMessages(ctx, chat.ID)

	return out, err
}
