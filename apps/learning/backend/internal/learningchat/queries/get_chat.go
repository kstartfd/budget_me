package queries

import (
	"context"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/llm"
)

type ChatHandler struct{ q *db.Queries }

func NewChatHandler(q *db.Queries) ChatHandler { return ChatHandler{q: q} }

type Chat struct {
	// Available is false when there is no agent to talk to: none chosen, or a
	// provider chat is not built for. The conversation already held stays
	// readable either way.
	Available bool
	Messages  []db.LearningChatMessage
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

	settings, err := h.q.GetLearningAISettings(ctx, ids.ToPg(userID))
	if err != nil && !db.IsNotFound(err) {
		return Chat{}, err
	}

	if err == nil {
		_, out.Available = llm.ChatProfileOf(llm.Provider(settings.Provider))
	}

	chat, err := h.q.GetLearningChatByEntry(ctx, db.GetLearningChatByEntryParams{EntryID: ids.ToPg(entryID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return out, nil
	}

	if err != nil {
		return Chat{}, err
	}

	out.Messages, err = h.q.ListLearningChatMessages(ctx, chat.ID)

	return out, err
}
