package commands

import (
	"reflect"
	"strings"
	"testing"
	"unicode"

	"learning_app/backend/internal/ocr"
)

// row is a line of words that tesseract was sure of.
func row(par, line int, text string) []ocr.Word {
	var out []ocr.Word

	for _, t := range strings.Fields(text) {
		out = append(out, ocr.Word{Block: 1, Paragraph: par, Line: line, Text: t, Confidence: 90})
	}

	return out
}

func lines(rows ...[]ocr.Word) []ocr.Word {
	var out []ocr.Word
	for _, r := range rows {
		out = append(out, r...)
	}

	return out
}

func kinds(found []Found) []string {
	out := make([]string, len(found))
	for i, f := range found {
		out[i] = string(f.Kind) + ":" + f.Text
	}

	return out
}

func TestPhrasesKeepEveryLineOfAListAsItsOwnItem(t *testing.T) {
	words := lines(
		row(1, 1, "Pizza margherita"),
		row(1, 2, "Spaghetti carbonara"),
		row(1, 3, "Hello"),
		row(2, 1, "12.50 EUR"),
		row(3, 1, "Good morning, my friend!"),
	)

	got := kinds(phrases(words, unicode.Latin))
	want := []string{"phrase:Pizza margherita", "phrase:Spaghetti carbonara", "word:Hello", "phrase:Good morning, my friend"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPhrasesJoinTheLinesOfAWrappedParagraphIntoSentences(t *testing.T) {
	words := lines(
		row(1, 1, "The quick brown fox jumps over"),
		row(1, 2, "the lazy dog and runs away fast."),
		row(1, 3, "It was a sunny day. We went to"),
		row(1, 4, "the beach and swam a lot."),
	)

	got := kinds(phrases(words, unicode.Latin))
	want := []string{
		"phrase:The quick brown fox jumps over the lazy dog and runs away fast",
		"phrase:It was a sunny day",
		"phrase:We went to the beach and swam a lot",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPhrasesMendAWordHyphenatedAtTheEndOfALine(t *testing.T) {
	words := lines(row(1, 1, "A good exam-"), row(1, 2, "ple of this."))

	if got := kinds(phrases(words, unicode.Latin)); !reflect.DeepEqual(got, []string{"phrase:A good example of this"}) {
		t.Fatalf("got %q", got)
	}
}

func TestPhrasesBreakWhereAWordCouldNotBeRead(t *testing.T) {
	words := lines(row(1, 1, "good morning"), []ocr.Word{{Block: 1, Paragraph: 1, Line: 1, Text: "x%q", Confidence: 20}}, row(1, 1, "friend"))

	got := kinds(phrases(words, unicode.Latin))
	want := []string{"phrase:good morning", "word:friend"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("a smudge was glued into a phrase: got %q, want %q", got, want)
	}
}

func TestPhrasesKeepOnlyTheLanguagesOwnAlphabet(t *testing.T) {
	words := lines(row(1, 1, "Доброе утро"), row(1, 2, "Hello"), row(1, 3, "Спасибо"))

	if got := kinds(phrases(words, unicode.Cyrillic)); !reflect.DeepEqual(got, []string{"phrase:Доброе утро", "word:Спасибо"}) {
		t.Errorf("Russian topic: %q", got)
	}

	if got := kinds(phrases(words, unicode.Latin)); !reflect.DeepEqual(got, []string{"word:Hello"}) {
		t.Errorf("English topic: %q", got)
	}
}

func TestPhrasesLeaveOutWhatIsNotWords(t *testing.T) {
	words := lines(row(1, 1, "12.50"), row(2, 1, "a"), row(3, 1, "||"), row(4, 1, "A4-7731"), row(5, 1, "(tea)"))

	if got := kinds(phrases(words, unicode.Latin)); !reflect.DeepEqual(got, []string{"word:tea"}) {
		t.Fatalf("got %q", got)
	}
}

func TestPhrasesGiveEachTextOnceWhateverItsCase(t *testing.T) {
	words := lines(row(1, 1, "Hello"), row(2, 1, "hello!"), row(3, 1, "HELLO"), row(4, 1, "Hello  there"), row(5, 1, "hello there"))

	if got := kinds(phrases(words, unicode.Latin)); !reflect.DeepEqual(got, []string{"word:Hello", "phrase:Hello there"}) {
		t.Fatalf("got %q", got)
	}
}

func TestPhrasesCutALongSentenceAtItsCommasAndDropWhatIsStillTooLong(t *testing.T) {
	short := "when it rains we stay home,"
	endless := strings.TrimSpace(strings.Repeat("and then it went on ", 6)) // 30 words, no pause

	words := lines(row(1, 1, "Well, "+short+" "+short+" "+short+" "+short), row(2, 1, endless+"."))

	got := kinds(phrases(words, unicode.Latin))
	want := []string{"word:Well", "phrase:when it rains we stay home"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
