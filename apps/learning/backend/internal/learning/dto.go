package learning

import (
	"time"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learning/commands"
)

type topicRequest struct {
	Name                string `json:"name"`
	Language            string `json:"language"`
	TranslationLanguage string `json:"translationLanguage"`
}
type topicResponse struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Language            string `json:"language"`
	TranslationLanguage string `json:"translationLanguage"`
}

func topicDTO(row db.LearningTopic) topicResponse {
	return topicResponse{ID: ids.FromPg(row.ID).String(), Name: row.Name, Language: row.Language, TranslationLanguage: row.TranslationLanguage}
}

// packRequest names the pack to write: "phrases" or "words".
type packRequest struct {
	Kind string `json:"kind"`
}

// reviewRequest is how a review went: again, good or easy.
type reviewRequest struct {
	Grade string `json:"grade"`
}

// reviewResponse is what is due in a topic: Overdue entries that were seen
// before and are due again, Fresh ones never reviewed, and the next cards.
type reviewResponse struct {
	Overdue int64           `json:"overdue"`
	Fresh   int64           `json:"fresh"`
	Entries []entryResponse `json:"entries"`
}

// reminderRequest switches the daily review reminder; timezone is the learner's
// IANA zone, so "morning" is theirs.
type reminderRequest struct {
	Enabled  bool   `json:"enabled"`
	Timezone string `json:"timezone"`
}

type reminderResponse struct {
	Enabled  bool   `json:"enabled"`
	Timezone string `json:"timezone"`
}

type entryResponse struct {
	ID string `json:"id"`
	commands.EntryFields
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func entryDTO(row db.LearningEntry) entryResponse {
	return entryResponse{ID: ids.FromPg(row.ID).String(), EntryFields: commands.EntryFields{TopicID: ids.FromPg(row.TopicID), Kind: row.Kind, Text: row.Text, Transcription: row.Transcription, Translation: row.Translation, Explanation: row.Explanation, NativeUsage: row.NativeUsage, Example: row.Example, Rule: row.Rule, Source: row.Source, Status: row.Status}, CreatedAt: row.CreatedAt.Time.UTC().Format(time.RFC3339), UpdatedAt: row.UpdatedAt.Time.UTC().Format(time.RFC3339)}
}

type captureResponse struct {
	ID      string  `json:"id"`
	Status  string  `json:"status"`
	EntryID *string `json:"entryId"`
	Error   string  `json:"error"`
}

func captureDTO(row db.LearningCapture) captureResponse {
	out := captureResponse{ID: ids.FromPg(row.ID).String(), Status: string(row.Status), Error: row.Error}
	if row.EntryID.Valid {
		id := ids.FromPg(row.EntryID).String()
		out.EntryID = &id
	}

	return out
}
