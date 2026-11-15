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

// planResponse is a topic's daily-set settings: how many new cards a day and
// which side is shown first ("translation" or "text").
type planResponse struct {
	DailyNew int    `json:"dailyNew"`
	Front    string `json:"front"`
}

// periodResponse is a weekly or monthly review: its calendar dates, how many cards
// belong to it and how many are still to go through.
type periodResponse struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Total   int64  `json:"total"`
	Pending int64  `json:"pending"`
}

type setsResponse struct {
	Plan planResponse `json:"plan"`
	// Available is how many saved entries are still waiting to join a set.
	Available int64 `json:"available"`
	// Fresh is today's new cards, Weak the earlier ones the learner did not know.
	Fresh int64           `json:"fresh"`
	Weak  int64           `json:"weak"`
	Today []entryResponse `json:"today"`
	Week  periodResponse  `json:"week"`
	Month periodResponse  `json:"month"`
}

type gradeSetRequest struct {
	Period   string `json:"period"`
	Known    bool   `json:"known"`
	Timezone string `json:"tz"`
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
