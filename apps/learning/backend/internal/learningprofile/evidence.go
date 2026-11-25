// Package learningprofile holds the rules that turn what a learner showed into
// what they know and when to ask again: how much an answer proves, how that
// moves an item through its states, and the ladder of reviews.
//
// It does no I/O and reads no clock, so every rule is a table-driven test. The
// numbers are PROPOSED defaults (docs/learning/LEARNING_METHODOLOGY.md) kept in
// one Config, so tuning them later never touches the code that uses them. They
// are not calibrated probabilities and nothing here claims to be one.
package learningprofile

// Mode is what an answer asked the learner to do.
type Mode string

const (
	// Recognition: pick the right one of a few. Proves little: a guess can be right.
	Recognition Mode = "recognition"
	// Recall: produce a known form (fill a gap).
	Recall Mode = "recall"
	// Production: build or write language, not just choose or complete it.
	Production Mode = "production"
)

// Source says where the exercise came from; it is independent of the practice
// mode (decision D-025). V1 writes LearningPath and Review.
type Source string

const (
	SourceRulebook     Source = "rulebook"
	SourceLearningPath Source = "learning_path"
	SourceStudySession Source = "study_session"
	SourceReview       Source = "review"
	SourceAITutor      Source = "ai_tutor"
	SourceAssessment   Source = "assessment"
)

// PracticeMode is how it was practised. Free practice stores nothing, so it is
// deliberately not a value here: there is no evidence row to give it.
type PracticeMode string

const (
	Tracked    PracticeMode = "tracked"
	Assessment PracticeMode = "assessment"
)

// Skills an exercise can prove, the closed list of the database.
const (
	SkillGrammar    = "grammar"
	SkillVocabulary = "vocabulary"
	SkillReading    = "reading"
	SkillWriting    = "writing"
	SkillListening  = "listening"
	SkillSpeaking   = "speaking"
)

// kindRule is what one kind of exercise proves.
type kindRule struct {
	mode   Mode
	weight float64
	skill  string // fixed skill of the kind; empty means the topic's own
}

// kinds maps the exercise kinds of a program to their evidence. The weight is how
// reliable the verdict is: the system checks a key (1.0) or the learner's agent
// judges (capped at 0.7 until its agreement with people is measured), and a pick
// among a few proves less than producing the form.
var kinds = map[string]kindRule{
	"choice":    {Recognition, 0.4, ""},
	"listen":    {Recognition, 0.4, SkillListening},
	"fill":      {Recall, 1.0, ""},
	"cloze":     {Production, 1.0, ""},
	"order":     {Production, 1.0, ""},
	"speak":     {Production, 1.0, SkillSpeaking},
	"translate": {Production, 0.7, ""},
	"write":     {Production, 0.7, SkillWriting},
	"read":      {Recall, 0.7, SkillReading},
}

// Kind is what an answer to an exercise of the given kind proves. ok is false
// for a kind that is not known, which stores no evidence rather than guessing.
func Kind(kind string) (mode Mode, weight float64, ok bool) {
	r, ok := kinds[kind]

	return r.mode, r.weight, ok
}

// SkillOf is the skill an answer proves: the kind's own (hearing, speaking,
// reading, writing) or, for the kinds that can serve any topic, the topic's.
func SkillOf(kind, topicSkill string) string {
	if r, ok := kinds[kind]; ok && r.skill != "" {
		return r.skill
	}

	switch topicSkill {
	case SkillGrammar, SkillVocabulary, SkillReading, SkillWriting, SkillListening, SkillSpeaking, "pronunciation":
		return topicSkill
	}

	return SkillGrammar
}
