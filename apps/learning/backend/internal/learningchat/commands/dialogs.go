package commands

import (
	"context"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

// Archive closes the open dialog of a topic and opens a fresh one: the
// learner's "clear chat". The closed dialog stays readable in the history. A
// dialog nobody wrote in is left alone, so clearing twice makes no empty
// dialogs.
func (h SendHandler) Archive(ctx context.Context, userID, topicID uuid.UUID) error {
	chat, err := h.q.GetOpenTopicChat(ctx, db.GetOpenTopicChatParams{TopicID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return h.topicExists(ctx, userID, topicID)
	}

	if err != nil {
		return err
	}

	now := h.clock.Now()

	chat, err = h.lock(ctx, chat, now)
	if err != nil {
		return err
	}

	defer func() { _ = h.q.UnlockLearningChat(context.WithoutCancel(ctx), chat.ID) }()

	stored, err := h.q.ListLearningChatMessages(ctx, chat.ID)
	if err != nil || len(stored) == 0 {
		return err
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	q := h.q.WithTx(tx)

	if err := q.ArchiveLearningChat(ctx, db.ArchiveLearningChatParams{ID: chat.ID, ArchivedAt: db.Time(now)}); err != nil {
		return err
	}

	err = q.OpenNextTopicChat(ctx, db.OpenNextTopicChatParams{
		ID:          ids.ToPg(ids.New()),
		UserID:      chat.UserID,
		TopicID:     chat.TopicID,
		SpentTokens: chat.SpentTokens,
		SpentSince:  chat.SpentSince,
		CreatedAt:   db.Time(now),
	})
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteDialog removes a closed dialog for good. The open one is never deleted
// here: it is cleared, which keeps it.
func (h SendHandler) DeleteDialog(ctx context.Context, userID, chatID uuid.UUID) error {
	n, err := h.q.DeleteArchivedTopicChat(ctx, db.DeleteArchivedTopicChatParams{ID: ids.ToPg(chatID), UserID: ids.ToPg(userID)})
	if err != nil {
		return err
	}

	if n == 0 {
		return apperr.NewNotFound("dialog not found")
	}

	return nil
}

// topicExists tells a missing topic from a topic nobody has talked in yet.
func (h SendHandler) topicExists(ctx context.Context, userID, topicID uuid.UUID) error {
	_, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: ids.ToPg(topicID), UserID: ids.ToPg(userID)})
	if db.IsNotFound(err) {
		return apperr.NewNotFound("topic not found")
	}

	return err
}
