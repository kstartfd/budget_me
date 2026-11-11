// Package placement holds the ladder the level tests climb: the CEFR rungs,
// the shape of a question and how an answer is graded. It holds no questions:
// every test is written for its learner by the learner's own agent.
package placement

import "strings"

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

// BatchSize and FinalBatchSize are how many questions one level of a test asks.
// Whether a batch is good enough to climb is the agent's call, not a threshold.
const (
	BatchSize      = 8
	FinalBatchSize = 10
)

// Question is one question of a test. Options are the choices the test-taker sees;
// Answer is the correct one, exactly as it appears in Options.
type Question struct {
	Key     string
	Skill   Skill
	Text    string
	Prompt  string
	Options []string
	Answer  string
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

// Grade compares an answer against the key, forgiving only case and edges:
// what is graded is whether the right option was chosen, not typing.
func Grade(q Question, answer string) bool {
	return norm(answer) == norm(q.Answer)
}

func norm(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

// Index is the position of a level on the ladder, or -1 when it is not a rung.
func Index(level Level) int {
	for i, l := range Levels {
		if l == level {
			return i
		}
	}

	return -1
}

// BatchNeed is how many questions a level batch carries, by test kind.
func BatchNeed(kind string) int {
	if kind == "final" {
		return FinalBatchSize
	}

	return BatchSize
}
