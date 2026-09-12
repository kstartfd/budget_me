package tasks

import (
	"time"

	"budget_me/backend/internal/db"
	"budget_me/backend/internal/ids"
)

type taskResponse struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Note       *string `json:"note"`
	PlannedFor string  `json:"plannedFor"`
	DoneOn     *string `json:"doneOn"`
	CreatedAt  string  `json:"createdAt"`
}

type createTaskRequest struct {
	Title      string  `json:"title"`
	Note       *string `json:"note"`
	PlannedFor string  `json:"plannedFor"`
}

type updateTaskRequest struct {
	Title      string  `json:"title"`
	Note       *string `json:"note"`
	PlannedFor string  `json:"plannedFor"`
}

type setTaskDoneRequest struct {
	DoneOn *string `json:"doneOn"`
}

func toTaskResponse(t db.Task) taskResponse {
	return taskResponse{
		ID:         ids.FromPg(t.ID).String(),
		Title:      t.Title,
		Note:       db.TextPtr(t.Note),
		PlannedFor: db.DateStr(t.PlannedFor),
		DoneOn:     db.DateStrPtr(t.DoneOn),
		CreatedAt:  t.CreatedAt.Time.UTC().Format(time.RFC3339),
	}
}
