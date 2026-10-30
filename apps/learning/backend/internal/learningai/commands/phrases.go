package commands

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"learning_app/backend/internal/db"
	learningcommands "learning_app/backend/internal/learning/commands"
	"learning_app/backend/internal/ocr"
)

const (
	// minConfidence is the least tesseract's own estimate a word may have to be
	// trusted; below it the word is more likely a smudge than a word.
	minConfidence = 55
	// maxPhraseWords is the longest phrase kept; a sentence longer than this is
	// cut at its commas, and what is still longer is left out.
	maxPhraseWords = 20
	// maxFoundRunes keeps a found text a word or a short phrase, well inside the
	// 1000 an entry accepts.
	maxFoundRunes = 200
	// maxCandidates bounds what one picture can yield before the known ones are
	// taken out.
	maxCandidates = 300
)

// Found is one word or phrase read off a picture.
type Found struct {
	Kind db.LearningEntryKind `json:"kind"`
	Text string               `json:"text"`
}

// phrases turns the words tesseract saw into the words and phrases worth
// studying, in the order they stand on the picture, each once. A sentence is a
// phrase and a lone word is a word; lines of a wrapped paragraph are joined into
// their sentences, lines of a list or a menu stay apart.
func phrases(words []ocr.Word, script *unicode.RangeTable) []Found {
	x := extraction{script: script, seen: map[string]bool{}}

	var (
		tokens           []string
		block, paragraph = -1, -1
		line             = -1
	)

	for _, w := range words {
		if len(x.found) >= maxCandidates {
			break
		}

		if w.Block != block || w.Paragraph != paragraph {
			x.emit(tokens)

			tokens, block, paragraph, line = nil, w.Block, w.Paragraph, -1
		}

		if w.Confidence < minConfidence {
			x.emit(tokens)

			tokens, line = nil, w.Line

			continue
		}

		text := strings.TrimSpace(w.Text)

		if line != -1 && w.Line != line && len(tokens) > 0 {
			last := tokens[len(tokens)-1]

			switch {
			case strings.HasSuffix(last, "-") && len(last) > 1 && startsLower(text):
				tokens[len(tokens)-1] = strings.TrimSuffix(last, "-") + text
				line = w.Line

				continue
			case endsClause(last) || !startsLower(text):
				x.emit(tokens)

				tokens = nil
			}
		}

		line = w.Line
		tokens = append(tokens, text)

		if endsSentence(text) {
			x.emit(tokens)

			tokens = nil
		}
	}

	x.emit(tokens)

	return x.found
}

type extraction struct {
	script *unicode.RangeTable
	seen   map[string]bool
	found  []Found
}

// emit takes one sentence-sized run of words. A run too long to be a phrase is
// cut where the writer paused.
func (x *extraction) emit(tokens []string) {
	if len(tokens) <= maxPhraseWords {
		x.add(tokens)

		return
	}

	var chunk []string

	for _, t := range tokens {
		chunk = append(chunk, t)

		if endsClause(t) {
			x.add(chunk)

			chunk = nil
		}
	}

	x.add(chunk)
}

func (x *extraction) add(tokens []string) {
	if len(tokens) == 0 || len(tokens) > maxPhraseWords || len(x.found) >= maxCandidates {
		return
	}

	text := strings.TrimFunc(strings.Join(tokens, " "), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if !studyable(text, x.script) {
		return
	}

	hash := string(learningcommands.TextHash(text))
	if x.seen[hash] {
		return
	}

	x.seen[hash] = true

	kind := db.LearningEntryKindPhrase
	if !strings.ContainsRune(text, ' ') {
		kind = db.LearningEntryKindWord
	}

	x.found = append(x.found, Found{Kind: kind, Text: text})
}

// studyable keeps text that is made of letters of the language's alphabet:
// prices, codes, page numbers and a line of another language fall out.
func studyable(text string, script *unicode.RangeTable) bool {
	if text == "" || utf8.RuneCountInString(text) > maxFoundRunes {
		return false
	}

	var letters, inScript, visible int

	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}

		visible++

		if unicode.IsLetter(r) {
			letters++

			if unicode.Is(script, r) {
				inScript++
			}
		}
	}

	return letters >= 2 && letters*10 >= visible*7 && inScript*10 >= letters*8
}

func startsLower(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)

	return unicode.IsLower(r)
}

func endsSentence(s string) bool {
	return strings.HasSuffix(s, ".") || strings.HasSuffix(s, "!") || strings.HasSuffix(s, "?") || strings.HasSuffix(s, "…")
}

func endsClause(s string) bool {
	for _, mark := range []string{",", ";", ":", "—", "–"} {
		if strings.HasSuffix(s, mark) {
			return true
		}
	}

	return false
}
