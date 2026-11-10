// Package learningmarks keeps, per user, the dictionary of trigger words the
// chat marks in a teacher's reply and how it marks each of them.
package learningmarks

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

const (
	maxTriggers = 60
	maxWordRune = 40
	maxIDLength = 64
)

var (
	styles = map[string]bool{"frame": true, "bar": true, "marker": true, "chip": true, "bold": true, "underline": true}
	colors = map[string]bool{"neutral": true, "blue": true, "green": true, "amber": true, "red": true}
)

// Trigger is one word of the dictionary and how it is marked.
type Trigger struct {
	ID    string `json:"id"`
	Word  string `json:"word"`
	Style string `json:"style"`
	Color string `json:"color"`
}

// Marks is everything the user chose about marking the chat.
type Marks struct {
	AnyLabel bool      `json:"anyLabel"`
	Dialogue bool      `json:"dialogue"`
	Triggers []Trigger `json:"triggers"`
}

type Store struct {
	q     *db.Queries
	clock clock.Clock
}

func NewStore(q *db.Queries, clk clock.Clock) Store { return Store{q: q, clock: clk} }

// Get returns nil until the user has changed anything: the app then shows its
// own defaults.
func (s Store) Get(ctx context.Context, userID uuid.UUID) (*Marks, error) {
	row, err := s.q.GetLearningChatMarks(ctx, ids.ToPg(userID))
	if db.IsNotFound(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return decode(row)
}

// Save replaces the user's dictionary. It is checked here, not trusted: the
// words reach every reply the user reads.
func (s Store) Save(ctx context.Context, userID uuid.UUID, in Marks) (*Marks, error) {
	clean, err := validate(in)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(clean.Triggers)
	if err != nil {
		return nil, err
	}

	row, err := s.q.UpsertLearningChatMarks(ctx, db.UpsertLearningChatMarksParams{
		UserID:    ids.ToPg(userID),
		AnyLabel:  clean.AnyLabel,
		Dialogue:  clean.Dialogue,
		Triggers:  raw,
		UpdatedAt: db.Time(s.clock.Now()),
	})
	if err != nil {
		return nil, err
	}

	return decode(row)
}

// Delete returns the user to the defaults. Nothing to forget is not an error.
func (s Store) Delete(ctx context.Context, userID uuid.UUID) error {
	_, err := s.q.DeleteLearningChatMarks(ctx, ids.ToPg(userID))

	return err
}

func decode(row db.LearningChatMark) (*Marks, error) {
	marks := Marks{AnyLabel: row.AnyLabel, Dialogue: row.Dialogue, Triggers: []Trigger{}}
	if err := json.Unmarshal(row.Triggers, &marks.Triggers); err != nil {
		return nil, err
	}

	return &marks, nil
}

func validate(in Marks) (Marks, error) {
	if len(in.Triggers) > maxTriggers {
		return Marks{}, apperr.NewInvalid("too many trigger words")
	}

	out := Marks{AnyLabel: in.AnyLabel, Dialogue: in.Dialogue, Triggers: make([]Trigger, 0, len(in.Triggers))}
	seen := map[string]bool{}

	for _, t := range in.Triggers {
		word := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(t.Word), ":："))
		key := strings.ToLower(word)

		switch {
		case word == "" || utf8.RuneCountInString(word) > maxWordRune || strings.ContainsAny(word, "\r\n"):
			return Marks{}, apperr.NewInvalid("a trigger word must be 1 to 40 characters on one line")
		case !styles[t.Style] || !colors[t.Color]:
			return Marks{}, apperr.NewInvalid("unknown style or colour")
		case t.ID == "" || len(t.ID) > maxIDLength:
			return Marks{}, apperr.NewInvalid("a trigger needs an id")
		case seen[key]:
			return Marks{}, apperr.NewInvalid("a trigger word is listed twice")
		}

		seen[key] = true
		out.Triggers = append(out.Triggers, Trigger{ID: t.ID, Word: word, Style: t.Style, Color: t.Color})
	}

	return out, nil
}
