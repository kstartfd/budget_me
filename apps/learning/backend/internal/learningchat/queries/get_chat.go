package queries

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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
	// than just answer; it only matters while the chat is available. The free
	// chat of a topic is always the teacher.
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

	chat, err := h.q.GetLearningChatByEntry(ctx, db.GetLearningChatByEntryParams{EntryID: ids.ToPg(entryID), UserID: ids.ToPg(userID)})
	if err != nil && !db.IsNotFound(err) {
		return Chat{}, err
	}

	return h.state(ctx, userID, chat, err == nil, false)
}

// GetTopic returns the open dialog of a topic, empty until the first question.
func (h ChatHandler) GetTopic(ctx context.Context, userID, topicID uuid.UUID) (Chat, error) {
	if err := h.topicExists(ctx, userID, topicID); err != nil {
		return Chat{}, err
	}

	chat, err := h.q.GetOpenTopicChat(ctx, db.GetOpenTopicChatParams{TopicID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if err != nil && !db.IsNotFound(err) {
		return Chat{}, err
	}

	return h.state(ctx, userID, chat, err == nil, true)
}

// GetPathTopic returns the conversation of a program topic, empty until the
// first question. It is always the teacher.
func (h ChatHandler) GetPathTopic(ctx context.Context, userID, topicID uuid.UUID) (Chat, error) {
	_, err := h.q.GetLearningPathTopic(ctx, db.GetLearningPathTopicParams{ID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return Chat{}, apperr.NewNotFound("topic not found")
	}

	if err != nil {
		return Chat{}, err
	}

	chat, err := h.q.GetLearningChatByPathTopic(ctx, db.GetLearningChatByPathTopicParams{PathTopicID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if err != nil && !db.IsNotFound(err) {
		return Chat{}, err
	}

	return h.state(ctx, userID, chat, err == nil, true)
}

// state is a chat as the learner sees it. `found` is false for a chat not
// started yet, whose zero row stands for the whole budget.
func (h ChatHandler) state(ctx context.Context, userID uuid.UUID, chat db.LearningChat, found, teaches bool) (Chat, error) {
	out := Chat{Messages: []db.LearningChatMessage{}}

	var profile llm.ChatProfile

	settings, err := h.q.GetLearningAISettings(ctx, ids.ToPg(userID))
	if err != nil && !db.IsNotFound(err) {
		return Chat{}, err
	}

	if err == nil {
		profile, out.Available = llm.ChatProfileOf(llm.Provider(settings.Provider))
		out.Teacher = out.Available && (teaches || settings.Teacher)
	}

	if out.Available {
		budget := commands.BudgetOf(chat, profile, h.clock.Now())
		out.Budget = &budget

		// Only the user's own count is shown: the key's would need the key opened.
		daily, err := commands.LoadDaily(ctx, h.q, userID, "", profile, h.clock.Now())
		if err != nil {
			return Chat{}, err
		}

		out.Daily = &daily.User
	}

	if !found {
		return out, nil
	}

	out.Messages, err = h.q.ListLearningChatMessages(ctx, chat.ID)

	return out, err
}

// Dialog is a closed conversation of a topic, as listed in its history.
type Dialog struct {
	ID           uuid.UUID
	StartedAt    time.Time
	ArchivedAt   time.Time
	FirstMessage string
	MessageCount int64
}

// History lists the closed dialogs of a topic, the newest first.
func (h ChatHandler) History(ctx context.Context, userID, topicID uuid.UUID) ([]Dialog, error) {
	if err := h.topicExists(ctx, userID, topicID); err != nil {
		return nil, err
	}

	rows, err := h.q.ListArchivedTopicChats(ctx, db.ListArchivedTopicChatsParams{TopicID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if err != nil {
		return nil, err
	}

	out := make([]Dialog, len(rows))
	for i, row := range rows {
		out[i] = Dialog{ID: ids.FromPg(row.ID), StartedAt: row.CreatedAt.Time, ArchivedAt: row.ArchivedAt.Time, FirstMessage: row.FirstMessage, MessageCount: row.MessageCount}
	}

	return out, nil
}

// Transcript is a closed dialog with everything said in it.
type Transcript struct {
	Dialog   Dialog
	Messages []db.LearningChatMessage
}

// Transcript reads one closed dialog.
func (h ChatHandler) Transcript(ctx context.Context, userID, chatID uuid.UUID) (Transcript, error) {
	chat, err := h.q.GetArchivedTopicChat(ctx, db.GetArchivedTopicChatParams{ID: ids.ToPg(chatID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return Transcript{}, apperr.NewNotFound("dialog not found")
	}

	if err != nil {
		return Transcript{}, err
	}

	messages, err := h.q.ListLearningChatMessages(ctx, chat.ID)
	if err != nil {
		return Transcript{}, err
	}

	return Transcript{
		Dialog:   Dialog{ID: chatID, StartedAt: chat.CreatedAt.Time, ArchivedAt: chat.ArchivedAt.Time, MessageCount: int64(len(messages))},
		Messages: messages,
	}, nil
}

func (h ChatHandler) topicExists(ctx context.Context, userID, topicID uuid.UUID) error {
	_, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return apperr.NewNotFound("topic not found")
	}

	return err
}

// sentenceRef is the part of a stored sentence that names it.
type sentenceRef struct {
	ID string `json:"id"`
}

// GetExample returns the conversation about one sentence of the learner's
// sitting on a rule, empty until the first question. It is always the teacher.
func (h ChatHandler) GetExample(ctx context.Context, userID uuid.UUID, ruleID, exampleID string) (Chat, error) {
	row, err := h.q.GetLearningRulePractice(ctx, db.GetLearningRulePracticeParams{UserID: ids.ToPg(userID), RuleID: ruleID})
	if err == nil && !row.ExpiresAt.Time.After(h.clock.Now()) {
		err = pgx.ErrNoRows
	}

	if db.IsNotFound(err) {
		return Chat{}, apperr.NewNotFound("these sentences are gone: write new ones")
	}

	if err != nil {
		return Chat{}, err
	}

	var sentences []sentenceRef
	if err := json.Unmarshal(row.Examples, &sentences); err != nil {
		return Chat{}, err
	}

	if !slices.ContainsFunc(sentences, func(s sentenceRef) bool { return s.ID == exampleID }) {
		return Chat{}, apperr.NewNotFound("sentence not found")
	}

	chat, err := h.q.GetLearningChatByExample(ctx, db.GetLearningChatByExampleParams{
		PracticeID: row.ID, ExampleID: pgtype.Text{String: exampleID, Valid: true}, UserID: ids.ToPg(userID),
	})
	if err != nil && !db.IsNotFound(err) {
		return Chat{}, err
	}

	return h.state(ctx, userID, chat, err == nil, true)
}
