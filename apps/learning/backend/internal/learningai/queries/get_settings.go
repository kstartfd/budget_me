package queries

import (
	"context"

	"github.com/google/uuid"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

type SettingsHandler struct{ q *db.Queries }

func NewSettingsHandler(q *db.Queries) SettingsHandler { return SettingsHandler{q: q} }

// Get returns nil until the user has chosen an agent.
func (h SettingsHandler) Get(ctx context.Context, userID uuid.UUID) (*db.LearningAiSetting, error) {
	row, err := h.q.GetLearningAISettings(ctx, ids.ToPg(userID))
	if db.IsNotFound(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &row, nil
}
