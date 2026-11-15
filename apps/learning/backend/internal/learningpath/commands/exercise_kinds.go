package commands

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"learning_app/backend/internal/learningpath/placement"
)

// Three kinds of exercise beyond a single sentence, all kept in the columns an
// exercise already has:
//
//	order  options hold the words of a sentence in the wrong order, answer holds
//	       the right order (several natural orders are separated by " | ").
//	cloze  text is a passage with numbered gaps [1], [2]…, answer holds what goes
//	       into each gap, in order, separated by " | " (variants of one gap by "/").
//	read   text is a passage, options are the questions about it (a question that
//	       ends in "…" is a sentence to finish), answer holds a model answer for
//	       each question, separated by " | ".
//
// The learner's answer to every one of them is written the same way as the key:
// the words in the order they chose, or the gaps and the replies in order, with
// " | " between them.
const (
	answerSep  = " | "
	variantSep = "/"

	minOrderWords = 3
	maxOrderWords = 14
	minGaps       = 3
	maxGaps       = 10
	minReadQs     = 2
	maxReadQs     = 5
	minPassage    = 100
)

var gapMark = regexp.MustCompile(`\[(\d{1,2})\]`)

// splitAnswers cuts an answer into its parts, dropping the blanks around them.
func splitAnswers(answer string) []string {
	parts := strings.Split(answer, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	return parts
}

// words lowers a sentence into its words, which is all that the order of an
// "order" exercise is made of; the capital letter and the full stop do not count.
func words(sentence string) []string {
	out := strings.Fields(strings.ToLower(sentence))

	return out
}

func sameWords(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	x, y := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)

	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}

	return true
}

// gapNumbers lists the numbers of the gaps of a passage, in order of appearance.
func gapNumbers(text string) []string {
	found := gapMark.FindAllStringSubmatch(text, -1)
	out := make([]string, len(found))

	for i, m := range found {
		out[i] = m[1]
	}

	return out
}

func validOrder(e exercise) error {
	if len(e.Options) < minOrderWords || len(e.Options) > maxOrderWords {
		return fmt.Errorf("order needs %d to %d words in options", minOrderWords, maxOrderWords)
	}

	tokens := make([]string, len(e.Options))
	for i, o := range e.Options {
		if strings.TrimSpace(o) == "" || len(strings.Fields(o)) != 1 {
			return fmt.Errorf("each option of order is one word, got %q", o)
		}

		tokens[i] = strings.ToLower(strings.TrimSpace(o))
	}

	orders := splitAnswers(e.Answer)
	for _, o := range orders {
		if !sameWords(words(o), tokens) {
			return fmt.Errorf("the answer %q does not use exactly the words in options", o)
		}
	}

	// A sentence given already in order is no exercise.
	if placement.Grade(placement.Question{Answer: strings.Join(e.Options, " ")}, orders[0]) {
		return fmt.Errorf("the words in options are already in the right order: shuffle them")
	}

	// The ready sentence must not be handed over in the task itself: not in the
	// clue, not in the instruction. The clue is a translation, never the answer.
	for _, o := range orders {
		if spellsOut(e.Text, o) || spellsOut(e.Prompt, o) {
			return fmt.Errorf("text and prompt must not contain the sentence %q: the learner puts it together", o)
		}
	}

	return nil
}

// bareWords lowers a text to its bare words: no punctuation, no capitals.
func bareWords(text string) string {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}

		return ' '
	}, text)

	return strings.Join(strings.Fields(cleaned), " ")
}

// spellsOut tells whether text holds the sentence, whatever its punctuation.
func spellsOut(text, sentence string) bool {
	want := bareWords(sentence)

	return want != "" && strings.Contains(" "+bareWords(text)+" ", " "+want+" ")
}

func validCloze(e exercise) error {
	if len(e.Options) != 0 {
		return fmt.Errorf("cloze takes no options")
	}

	if utf8.RuneCountInString(strings.TrimSpace(e.Text)) < minPassage {
		return fmt.Errorf("cloze needs a passage of at least %d characters in text", minPassage)
	}

	numbers := gapNumbers(e.Text)
	if len(numbers) < minGaps || len(numbers) > maxGaps {
		return fmt.Errorf("cloze needs %d to %d gaps marked [1], [2]… in text", minGaps, maxGaps)
	}

	for i, n := range numbers {
		if n != fmt.Sprint(i+1) {
			return fmt.Errorf("the gaps must be numbered [1], [2]… in order, found [%s] at place %d", n, i+1)
		}
	}

	answers := splitAnswers(e.Answer)
	if len(answers) != len(numbers) {
		return fmt.Errorf("%d gaps but %d answers: answer lists what goes into each gap, separated by %q", len(numbers), len(answers), answerSep)
	}

	for _, a := range answers {
		if a == "" {
			return fmt.Errorf("an empty answer for a gap")
		}
	}

	return nil
}

func validRead(e exercise) error {
	if utf8.RuneCountInString(strings.TrimSpace(e.Text)) < minPassage {
		return fmt.Errorf("read needs a passage of at least %d characters in text", minPassage)
	}

	if len(e.Options) < minReadQs || len(e.Options) > maxReadQs {
		return fmt.Errorf("read needs %d to %d questions in options", minReadQs, maxReadQs)
	}

	for _, q := range e.Options {
		if strings.TrimSpace(q) == "" {
			return fmt.Errorf("an empty question")
		}
	}

	answers := splitAnswers(e.Answer)
	if len(answers) != len(e.Options) {
		return fmt.Errorf("%d questions but %d answers: answer lists a model answer for each, separated by %q", len(e.Options), len(answers), answerSep)
	}

	for _, a := range answers {
		if a == "" {
			return fmt.Errorf("an empty model answer")
		}
	}

	return nil
}

// gradeOrder tells whether the words are in one of the orders the key allows.
func gradeOrder(key, learner string) bool {
	for _, o := range splitAnswers(key) {
		if placement.Grade(placement.Question{Answer: o}, learner) {
			return true
		}
	}

	return false
}

// gradeGaps counts the gaps of a cloze the learner got wrong. A gap with
// variants in the key ("don't/do not") takes any of them.
func gradeGaps(key, learner string) int {
	right, given := splitAnswers(key), splitAnswers(learner)
	wrong := 0

	for i, want := range right {
		if i >= len(given) {
			wrong++

			continue
		}

		ok := false

		for _, variant := range strings.Split(want, variantSep) {
			if placement.Grade(placement.Question{Answer: strings.TrimSpace(variant)}, given[i]) {
				ok = true

				break
			}
		}

		if !ok {
			wrong++
		}
	}

	return wrong
}
