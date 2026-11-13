package learningpath

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningpath/commands"
	"learning_app/backend/internal/learningpath/curriculum"
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

// jobResponse is the building of a program: queued, running, done (with the
// program) or failed (with why).
type jobResponse struct {
	ID     string        `json:"id"`
	TestID string        `json:"testId"`
	Status string        `json:"status"`
	Error  string        `json:"error"`
	Path   *pathResponse `json:"path"`
}

// teacherResponse is a language the learner studies, with a line about its
// running program.
type teacherResponse struct {
	ID                  string `json:"id"`
	Language            string `json:"language"`
	TranslationLanguage string `json:"translationLanguage"`
	// ExplainInTarget is the learner's choice to be explained to in the learned
	// language; ExplainLanguage is the language that gives, either way.
	ExplainInTarget bool                 `json:"explainInTarget"`
	ExplainLanguage string               `json:"explainLanguage"`
	CreatedAt       time.Time            `json:"createdAt"`
	Path            *teacherPathResponse `json:"path"`
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
		ExplainInTarget:     v.Teacher.ExplainInTarget,
		ExplainLanguage:     commands.ExplainLanguage(v.Teacher),
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
	// Exercises is what the topic's current exercise set holds and how much of
	// it is answered right; left out where it was not asked for (a single topic).
	Exercises *exerciseCountResponse `json:"exercises,omitempty"`
}

// exerciseCountResponse is the progress of a topic by its exercises. A topic
// whose set is not written yet has a total of 0.
type exerciseCountResponse struct {
	Total int `json:"total"`
	Done  int `json:"done"`
}

// wordResponse is one key word of a topic.
type wordResponse struct {
	Word        string `json:"word"`
	Translation string `json:"translation"`
}

// localizedTopic puts a program topic that came from the library into the
// language the teacher explains in now. A program keeps the text it was built
// with, so one built before the learner switched (or built in the other
// language) would otherwise stay in it; the library has the same topic in both.
// The reason the planner added to the goal is not kept: it was written in the
// old language. explain is empty where there is no teacher to follow.
func localizedTopic(t db.LearningPathTopic, language, explain string) db.LearningPathTopic {
	if explain == "" || t.SourceID == "" {
		return t
	}

	lib, ok := curriculum.Find(language, explain, t.SourceID)
	if !ok || lib.Title == t.Title {
		return t
	}

	t.Title, t.Unit, t.Goal, t.Theory = lib.Title, lib.Unit, lib.Goal, lib.Theory

	return t
}

func topicDTO(t db.LearningPathTopic, language, explain string) topicResponse {
	t = localizedTopic(t, language, explain)

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

func topicsDTO(list []db.LearningPathTopic, counts map[uuid.UUID]queries.ExerciseCount, language, explain string) []topicResponse {
	out := make([]topicResponse, len(list))
	for i, t := range list {
		out[i] = topicDTO(t, language, explain)

		count := counts[ids.FromPg(t.ID)]
		out[i].Exercises = &exerciseCountResponse{Total: count.Total, Done: count.Done}
	}

	return out
}

// pathResponse is a program with its steps.
type pathResponse struct {
	ID                  string `json:"id"`
	Language            string `json:"language"`
	TranslationLanguage string `json:"translationLanguage"`
	// ExplainLanguage is what the program's teacher explains in now.
	ExplainLanguage string          `json:"explainLanguage"`
	Level           string          `json:"level"`
	Summary         string          `json:"summary"`
	Status          string          `json:"status"`
	Source          string          `json:"source"`
	Preferences     map[string]any  `json:"preferences"`
	Topics          []topicResponse `json:"topics"`
	CreatedAt       time.Time       `json:"createdAt"`
	ArchivedAt      *time.Time      `json:"archivedAt"`
}

// pathDTO renders a program. explain is the language its teacher explains in
// now, empty for a program with no teacher (one in the history).
func pathDTO(p queries.PathWithTopics, explain string) pathResponse {
	shown := explain
	if shown == "" {
		shown = p.Path.TranslationLanguage
	}

	out := pathResponse{
		ID:                  ids.FromPg(p.Path.ID).String(),
		Language:            p.Path.Language,
		TranslationLanguage: p.Path.TranslationLanguage,
		ExplainLanguage:     shown,
		Level:               p.Path.Level,
		Summary:             p.Path.Summary,
		Status:              string(p.Path.Status),
		Source:              p.Path.Source,
		Topics:              topicsDTO(p.Topics, p.Exercises, p.Path.Language, explain),
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
	ID         string `json:"id"`
	ExerciseID string `json:"exerciseId"`
	Answer     string `json:"answer"`
	Correct    bool   `json:"correct"`
	// Mistakes is what a native speaker would find wrong in a text the learner
	// wrote on their own; 0 for every other kind of answer.
	Mistakes  int       `json:"mistakes"`
	Feedback  string    `json:"feedback"`
	CreatedAt time.Time `json:"createdAt"`
}

func attemptDTO(a db.LearningPathAttempt) attemptResponse {
	return attemptResponse{
		ID:         ids.FromPg(a.ID).String(),
		ExerciseID: ids.FromPg(a.ExerciseID).String(),
		Answer:     a.Answer,
		Correct:    a.Correct,
		Mistakes:   int(a.Mistakes),
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

// ruleRef is a rule in a list or a reference: enough to show and to link to.
type ruleRef struct {
	ID    string `json:"id"`
	Level string `json:"level"`
	Title string `json:"title"`
	Unit  string `json:"unit"`
}

// ruleResponse is one rule of the rule book with its references resolved.
type ruleResponse struct {
	ID       string            `json:"id"`
	Level    string            `json:"level"`
	Title    string            `json:"title"`
	Unit     string            `json:"unit"`
	Goal     string            `json:"goal"`
	Theory   string            `json:"theory"`
	Forms    []string          `json:"forms"`
	Examples []exampleResponse `json:"examples"`
	Mistakes []mistakeResponse `json:"mistakes"`
	Related  []ruleRef         `json:"related"`
}

type exampleResponse struct {
	Text        string `json:"text"`
	Translation string `json:"translation"`
}

type mistakeResponse struct {
	Wrong string `json:"wrong"`
	Right string `json:"right"`
	Note  string `json:"note"`
}

func ruleRefOf(t curriculum.Topic) ruleRef {
	return ruleRef{ID: t.ID, Level: t.Level, Title: t.Title, Unit: t.Unit}
}

// ruleDTO renders a topic of the rule book; its references are looked up in the
// same book, and one that does not lead to a rule is left out.
func ruleDTO(t curriculum.Topic, language, native, explain string) ruleResponse {
	out := ruleResponse{
		ID: t.ID, Level: t.Level, Title: t.Title, Unit: t.Unit, Goal: t.Goal, Theory: t.Theory,
		Forms: []string{}, Examples: []exampleResponse{}, Mistakes: []mistakeResponse{}, Related: []ruleRef{},
	}

	if t.Extra == nil {
		return out
	}

	out.Forms = append(out.Forms, t.Extra.Forms...)

	// A rule explained in the learned language still gives the sentences their
	// translation into the learner's own, from the rule book they would read
	// otherwise. The two books keep their examples in the same order.
	var own []curriculum.Example

	if explain != native {
		if book, ok := curriculum.Find(language, native, t.ID); ok && book.Extra != nil && len(book.Extra.Examples) == len(t.Extra.Examples) {
			own = book.Extra.Examples
		}
	}

	for i, e := range t.Extra.Examples {
		translation := e.Translation
		if translation == "" && own != nil {
			translation = own[i].Translation
		}

		out.Examples = append(out.Examples, exampleResponse{Text: e.Text, Translation: translation})
	}

	for _, m := range t.Extra.Mistakes {
		out.Mistakes = append(out.Mistakes, mistakeResponse{Wrong: m.Wrong, Right: m.Right, Note: m.Note})
	}

	for _, id := range t.Extra.Related {
		if other, ok := curriculum.Find(language, explain, id); ok && other.Extra != nil {
			out.Related = append(out.Related, ruleRefOf(other))
		}
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
	// Rule is the rule of the topic from the rule book, when there is one.
	Rule *ruleResponse `json:"rule"`
}

func topicDetailDTO(d queries.TopicDetail, rule *ruleResponse, explain string) topicDetailResponse {
	return topicDetailResponse{
		Path:      pathDTO(queries.PathWithTopics{Path: d.Path}, explain),
		Topic:     topicDTO(d.Topic, d.Path.Language, explain),
		Exercises: exercisesDTO(d.Exercises),
		Attempts:  attemptsDTO(d.Attempts),
		Rule:      rule,
	}
}

// lessonResponse is a timed lesson of the learner. The clock is the server's: the client
// counts down to endsAt, so a reload or another device shows the same time.
type lessonResponse struct {
	ID            string     `json:"id"`
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
