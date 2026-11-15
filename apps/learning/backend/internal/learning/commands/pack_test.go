package commands

import (
	"strings"
	"testing"
)

func TestPackSystemIsConstantAndLanguageAware(t *testing.T) {
	es := packSystem(PackWords, "es", "Spanish", "Russian", 10)

	if es != packSystem(PackWords, "es", "Spanish", "Russian", 10) {
		t.Fatal("the pack prompt changes between calls: the provider cache would be lost")
	}

	for _, want := range []string{"article", "infinitive", "remember the word", "exactly 10"} {
		if !strings.Contains(es, want) {
			t.Errorf("the Spanish word pack lacks %q", want)
		}
	}

	en := packSystem(PackPhrases, "en", "English", "Russian", 20)
	if strings.Contains(en, "article") && strings.Contains(en, "el libro") || !strings.Contains(en, "exactly 20 phrases") || !strings.Contains(en, "base form") {
		t.Errorf("the English phrase pack is shaped wrong:\n%s", en)
	}
}

func TestFreshEntriesSkipsWhatTheTopicHasAndWhatRepeats(t *testing.T) {
	known := map[string]bool{string(TextHash("Hello")): true}

	got := freshEntries([]packEntry{
		{Text: "hello", Translation: "привет"},           // already saved, whatever the case
		{Text: "How are you?", Translation: "как дела"},  // new
		{Text: "how  are you?", Translation: "как дела"}, // repeats the one above
		{Text: "", Translation: "x"},                     // empty
		{Text: "No translation"},                         // unusable
		{Text: "See you", Translation: "увидимся"},       // new, but over the size
	}, known, 1)

	if len(got) != 1 || got[0].Text != "How are you?" {
		t.Fatalf("got %+v", got)
	}
}

func TestParsePackReadsAFencedReply(t *testing.T) {
	got := parsePack("Here you go:\n```json\n{\"entries\": [{\"text\": \"a\", \"translation\": \"b\"}]}\n```")
	if len(got) != 1 || got[0].Text != "a" {
		t.Fatalf("got %+v", got)
	}

	if parsePack("nothing") != nil {
		t.Fatal("garbage must give no entries")
	}
}
