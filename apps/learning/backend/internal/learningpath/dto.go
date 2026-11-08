package learningpath

import (
	"encoding/json"
	"time"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningpath/commands"
	"learning_app/backend/internal/learningpath/placement"
	"learning_app/backend/internal/learningpath/queries"
)

// questionResponse is a bank question as the test-taker sees it: never the
// answer.
type questionResponse struct {
	Key     string   `json:"key"`
	Skill   string   `json:"skill"`
	Text    string   `json:"text"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
}

func questionDTO(q placement.Question) questionResponse {
	return questionResponse{Key: q.Key, Skill: string(q.Skill), Text: q.Text, Prompt: q.Prompt, Options: q.Options}
}

func questionsDTO(list []placement.Question) []questionResponse {
	out := make([]questionResponse, len(list))
	for i, q := range list {
		out[i] = questionDTO(q)
	}

	return out
}

// testResponse is a test and where it stands.
type testResponse struct {
	ID                  string             `json:"id"`
	Kind                string             `json:"kind"`
	Language            string             `json:"language"`
	TranslationLanguage string             `json:"translationLanguage"`
	Level               string             `json:"level"`
	LevelReached        string             `json:"levelReached"`
	Passed              bool               `json:"passed"`
	Status              string             `json:"status"`
	Questions           []questionResponse `json:"questions"`
	Asked               int                `json:"asked"`
	Correct             int                `json:"correct"`
	Needed              int                `json:"needed"`
	PassNeed            int                `json:"passNeed"`
	CreatedAt           time.Time          `json:"createdAt"`
	CompletedAt         *time.Time         `json:"completedAt"`
}

func testDTO(p commands.TestProgress) testResponse {
	out := testResponse{
		ID:                  ids.FromPg(p.Test.ID).String(),
		Kind:                string(p.Test.Kind),
		Language:            p.Test.Language,
		TranslationLanguage: p.Test.TranslationLanguage,
		Level:               p.Test.Level,
		LevelReached:        p.Test.LevelReached,
		Passed:              p.Test.Passed,
		Status:              string(p.Test.Status),
		Questions:           questionsDTO(p.Questions),
		Asked:               p.Asked,
		Correct:             p.Correct,
		Needed:              p.Needed,
		PassNeed:            p.PassNeed,
		CreatedAt:           p.Test.CreatedAt.Time,
	}

	if p.Test.CompletedAt.Valid {
		t := p.Test.CompletedAt.Time
		out.CompletedAt = &t
	}

	return out
}

// topicResponse is one step of the program.
type topicResponse struct {
	ID          string     `json:"id"`
	Position    int        `json:"position"`
	Skill       string     `json:"skill"`
	Title       string     `json:"title"`
	Goal        string     `json:"goal"`
	Status      string     `json:"status"`
	CompletedAt *time.Time `json:"completedAt"`
}

func topicDTO(t db.LearningPathTopic) topicResponse {
	out := topicResponse{
		ID:       ids.FromPg(t.ID).String(),
		Position: int(t.Position),
		Skill:    t.Skill,
		Title:    t.Title,
		Goal:     t.Goal,
		Status:   string(t.Status),
	}

	if t.CompletedAt.Valid {
		at := t.CompletedAt.Time
		out.CompletedAt = &at
	}

	return out
}

func topicsDTO(list []db.LearningPathTopic) []topicResponse {
	out := make([]topicResponse, len(list))
	for i, t := range list {
		out[i] = topicDTO(t)
	}

	return out
}

// pathResponse is a program with its steps.
type pathResponse struct {
	ID                  string          `json:"id"`
	Language            string          `json:"language"`
	TranslationLanguage string          `json:"translationLanguage"`
	Level               string          `json:"level"`
	Summary             string          `json:"summary"`
	Status              string          `json:"status"`
	Source              string          `json:"source"`
	Topics              []topicResponse `json:"topics"`
	CreatedAt           time.Time       `json:"createdAt"`
	ArchivedAt          *time.Time      `json:"archivedAt"`
}

func pathDTO(p queries.PathWithTopics) pathResponse {
	out := pathResponse{
		ID:                  ids.FromPg(p.Path.ID).String(),
		Language:            p.Path.Language,
		TranslationLanguage: p.Path.TranslationLanguage,
		Level:               p.Path.Level,
		Summary:             p.Path.Summary,
		Status:              string(p.Path.Status),
		Source:              p.Path.Source,
		Topics:              topicsDTO(p.Topics),
		CreatedAt:           p.Path.CreatedAt.Time,
	}

	if p.Path.ArchivedAt.Valid {
		at := p.Path.ArchivedAt.Time
		out.ArchivedAt = &at
	}

	return out
}

// exerciseResponse is one exercise as the learner sees it: the answer never
// leaves the server.
type exerciseResponse struct {
	ID       string   `json:"id"`
	Position int      `json:"position"`
	Kind     string   `json:"kind"`
	Text     string   `json:"text"`
	Prompt   string   `json:"prompt"`
	Options  []string `json:"options"`
}

func exerciseDTO(e db.LearningPathExercise) exerciseResponse {
	var options []string
	if len(e.Options) > 0 {
		_ = json.Unmarshal(e.Options, &options)
	}

	return exerciseResponse{
		ID:       ids.FromPg(e.ID).String(),
		Position: int(e.Position),
		Kind:     string(e.Kind),
		Text:     e.Text,
		Prompt:   e.Prompt,
		Options:  options,
	}
}

func exercisesDTO(list []db.LearningPathExercise) []exerciseResponse {
	out := make([]exerciseResponse, len(list))
	for i, e := range list {
		out[i] = exerciseDTO(e)
	}

	return out
}

// attemptResponse is a stored attempt.
type attemptResponse struct {
	ID         string    `json:"id"`
	ExerciseID string    `json:"exerciseId"`
	Answer     string    `json:"answer"`
	Correct    bool      `json:"correct"`
	Feedback   string    `json:"feedback"`
	CreatedAt  time.Time `json:"createdAt"`
}

func attemptDTO(a db.LearningPathAttempt) attemptResponse {
	return attemptResponse{
		ID:         ids.FromPg(a.ID).String(),
		ExerciseID: ids.FromPg(a.ExerciseID).String(),
		Answer:     a.Answer,
		Correct:    a.Correct,
		Feedback:   a.Feedback,
		CreatedAt:  a.CreatedAt.Time,
	}
}

func attemptsDTO(list []db.LearningPathAttempt) []attemptResponse {
	out := make([]attemptResponse, len(list))
	for i, a := range list {
		out[i] = attemptDTO(a)
	}

	return out
}

// topicDetailResponse is the topic screen: the step, its exercises and the
// attempts made on them.
type topicDetailResponse struct {
	Path      pathResponse       `json:"path"`
	Topic     topicResponse      `json:"topic"`
	Exercises []exerciseResponse `json:"exercises"`
	Attempts  []attemptResponse  `json:"attempts"`
}

func topicDetailDTO(d queries.TopicDetail) topicDetailResponse {
	return topicDetailResponse{
		Path:      pathDTO(queries.PathWithTopics{Path: d.Path}),
		Topic:     topicDTO(d.Topic),
		Exercises: exercisesDTO(d.Exercises),
		Attempts:  attemptsDTO(d.Attempts),
	}
}
