// Package placement holds the in-code question bank the level tests are built
// from. The bank is data, not prompts: every question is multiple choice with
// exactly one right option, so grading a batch is deterministic and costs
// nothing. A fresh test is still a fresh test, because each batch is a random
// draw the already-asked questions are left out of.
//
// The questions are the project's own, written against the CEFR level
// descriptors; no certification material is copied. A language joins the
// ladder by adding its questions to the bank.
package placement

import (
	"math/rand/v2"
	"strings"
)

// Skill is what a question probes. Reading questions carry a short Text; the
// other skills ask straight from the Prompt.
type Skill string

const (
	SkillGrammar    Skill = "grammar"
	SkillVocabulary Skill = "vocabulary"
	SkillReading    Skill = "reading"
)

// Level is one rung of the ladder.
type Level string

const (
	A1 Level = "A1"
	A2 Level = "A2"
	B1 Level = "B1"
	B2 Level = "B2"
	C1 Level = "C1"
	C2 Level = "C2"
)

// Levels is the ladder from the first rung up.
var Levels = []Level{A1, A2, B1, B2, C1, C2}

// How a test runs: a batch this big is drawn per level, and passing needs at
// least PassCorrect right answers in it. The final test is one bigger batch
// and a stricter bar.
const (
	BatchSize        = 8
	PassCorrect      = 6
	FinalBatchSize   = 10
	FinalPassCorrect = 8
)

// Question is one bank item. Options are the choices the test-taker sees;
// Answer is the correct one, exactly as it appears in Options.
type Question struct {
	Key     string
	Skill   Skill
	Text    string
	Prompt  string
	Options []string
	Answer  string
}

type keyedQuestion struct {
	question Question
	level    Level
}

// bank is questions per language, per level. The flat index lets a question be
// found by key alone, which is how an answer row refers to it.
var (
	bank  = map[string]map[Level][]Question{}
	index = map[string]map[string]keyedQuestion{}
)

func add(language string, level Level, questions []Question) {
	if bank[language] == nil {
		bank[language] = map[Level][]Question{}
		index[language] = map[string]keyedQuestion{}
	}

	bank[language][level] = questions

	for _, q := range questions {
		index[language][q.Key] = keyedQuestion{question: q, level: level}
	}
}

// Has tells whether the ladder exists for a language at all.
func Has(language string) bool {
	_, ok := bank[language]
	return ok
}

// Questions returns the bank of one rung, for tests and for drawing batches.
func Questions(language string, level Level) []Question {
	return bank[language][level]
}

// Find locates a question by its key, wherever on the ladder it sits.
func Find(language, key string) (Question, Level, bool) {
	found, ok := index[language][key]
	if !ok {
		return Question{}, "", false
	}

	return found.question, found.level, true
}

// Next is the rung above, or false when the ladder is finished.
func Next(level Level) (Level, bool) {
	for i, l := range Levels {
		if l == level {
			if i+1 < len(Levels) {
				return Levels[i+1], true
			}

			return "", false
		}
	}

	return "", false
}

// Below is the rung under, clamped at the first: failing the very first batch
// still places the learner at the start of the ladder.
func Below(level Level) Level {
	for i, l := range Levels {
		if l == level {
			if i > 0 {
				return Levels[i-1]
			}

			return Levels[0]
		}
	}

	return Levels[0]
}

// Batch draws up to n random questions of one level, leaving out every key in
// exclude (what this test has already asked).
func Batch(language string, level Level, exclude map[string]bool, n int) []Question {
	var candidates []Question

	for _, q := range bank[language][level] {
		if !exclude[q.Key] {
			candidates = append(candidates, q)
		}
	}

	rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })

	if len(candidates) > n {
		candidates = candidates[:n]
	}

	return candidates
}

// Grade compares an answer against the key, forgiving only case and edges:
// what is graded is whether the right option was chosen, not typing.
func Grade(q Question, answer string) bool {
	return norm(answer) == norm(q.Answer)
}

func norm(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

// PassNeed is how many right answers of a batch pass it, by test kind.
func PassNeed(kind string) int {
	if kind == "final" {
		return FinalPassCorrect
	}

	return PassCorrect
}

// BatchNeed is how many questions a level batch carries, by test kind.
func BatchNeed(kind string) int {
	if kind == "final" {
		return FinalBatchSize
	}

	return BatchSize
}
