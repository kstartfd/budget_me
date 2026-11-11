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

// questionResponse is a test question as the test-taker sees it: never the
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
	StartLevel          string             `json:"startLevel"`
	LevelReached        string             `json:"levelReached"`
	Passed              bool               `json:"passed"`
	Status              string             `json:"status"`
	Questions           []questionResponse `json:"questions"`
	Asked               int                `json:"asked"`
	Correct             int                `json:"correct"`
	Needed              int                `json:"needed"`
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
		StartLevel:          p.StartLevel,
		LevelReached:        p.Test.LevelReached,
		Passed:              p.Test.Passed,
		Status:              string(p.Test.Status),
		Questions:           questionsDTO(p.Questions),
		Asked:               p.Asked,
		Correct:             p.Correct,
		Needed:              p.Needed,
		CreatedAt:           p.Test.CreatedAt.Time,
	}

	if p.Test.CompletedAt.Valid {
		t := p.Test.CompletedAt.Time
		out.CompletedAt = &t
	}

	return out
}

// teacherResponse is a language the learner studies, with a line about its
// running program.
type teacherResponse struct {
	ID                  string               `json:"id"`
	Language            string               `json:"language"`
	TranslationLanguage string               `json:"translationLanguage"`
	CreatedAt           time.Time            `json:"createdAt"`
	Path                *teacherPathResponse `json:"path"`
}

type teacherPathResponse struct {
	ID         string `json:"id"`
	Level      string `json:"level"`
	TopicsDone int    `json:"topicsDone"`
	TopicsAll  int    `json:"topicsTotal"`
}

func teacherDTO(v queries.TeacherView) teacherResponse {
	out := teacherResponse{
		ID:                  ids.FromPg(v.Teacher.ID).String(),
		Language:            v.Teacher.Language,
		TranslationLanguage: v.Teacher.TranslationLanguage,
		CreatedAt:           v.Teacher.CreatedAt.Time,
	}

	if v.Path != nil {
		out.Path = &teacherPathResponse{ID: v.Path.ID.String(), Level: v.Path.Level, TopicsDone: v.Path.Done, TopicsAll: v.Path.Total}
	}

	return out
}

// topicResponse is one step of the program.
type topicResponse struct {
	ID          string         `json:"id"`
	Position    int            `json:"position"`
	Skill       string         `json:"skill"`
	Title       string         `json:"title"`
	Goal        string         `json:"goal"`
	Unit        string         `json:"unit"`
	Theory      string         `json:"theory"`
	Vocabulary  []wordResponse `json:"vocabulary"`
	Status      string         `json:"status"`
	CompletedAt *time.Time     `json:"completedAt"`
}

// wordResponse is one key word of a topic.
type wordResponse struct {
	Word        string `json:"word"`
	Translation string `json:"translation"`
}

func topicDTO(t db.LearningPathTopic) topicResponse {
	out := topicResponse{
		ID:       ids.FromPg(t.ID).String(),
		Position: int(t.Position),
		Skill:    t.Skill,
		Title:    t.Title,
		Goal:     t.Goal,
		Unit:     t.Unit,
		Theory:   t.Theory,
		Status:   string(t.Status),
	}

	out.Vocabulary = []wordResponse{}
	if len(t.Vocabulary) > 0 {
		_ = json.Unmarshal(t.Vocabulary, &out.Vocabulary)
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
	Preferences         map[string]any  `json:"preferences"`
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
		Preferences:         map[string]any{},
	}

	if len(p.Path.Preferences) > 0 {
		_ = json.Unmarshal(p.Path.Preferences, &out.Preferences)
	}

	if p.Path.ArchivedAt.Valid {
		at := p.Path.ArchivedAt.Time
		out.ArchivedAt = &at
	}

	return out
}

// exerciseResponse is one exercise with its answer and the rule that explains it.
type exerciseResponse struct {
	ID       string   `json:"id"`
	Position int      `json:"position"`
	Kind     string   `json:"kind"`
	Text     string   `json:"text"`
	Prompt   string   `json:"prompt"`
	Options  []string `json:"options"`
	Answer   string   `json:"answer"`
	Rule     string   `json:"rule"`
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
		Answer:   e.Answer,
		Rule:     e.Rule,
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

// lessonResponse is a timed lesson. The clock is the server's: the client
// counts down to endsAt, so a reload or another device shows the same time.
type lessonResponse struct {
	ID            string     `json:"id"`
	TopicID       string     `json:"topicId"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"startedAt"`
	EndsAt        time.Time  `json:"endsAt"`
	FinishedAt    *time.Time `json:"finishedAt"`
	PausedAt      *time.Time `json:"pausedAt"`
	PausedSeconds int        `json:"pausedSeconds"`
	PauseCount    int        `json:"pauseCount"`
	Attempts      int        `json:"attempts"`
	Correct       int        `json:"correct"`
}

func lessonDTO(l db.LearningPathLesson, s commands.Stats) lessonResponse {
	out := lessonResponse{
		ID:            ids.FromPg(l.ID).String(),
		TopicID:       ids.FromPg(l.TopicID).String(),
		Status:        string(l.Status),
		StartedAt:     l.StartedAt.Time,
		EndsAt:        l.EndsAt.Time,
		PausedSeconds: int(l.PausedSeconds),
		PauseCount:    int(l.PauseCount),
		Attempts:      s.Attempts,
		Correct:       s.Correct,
	}

	if l.FinishedAt.Valid {
		at := l.FinishedAt.Time
		out.FinishedAt = &at
	}

	if l.PausedAt.Valid {
		at := l.PausedAt.Time
		out.PausedAt = &at
	}

	return out
}
