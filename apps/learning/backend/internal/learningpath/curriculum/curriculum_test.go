package curriculum_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"learning_app/backend/internal/learningpath/curriculum"
	"learning_app/backend/internal/learningpath/placement"
)

// TestLibraryIsWellFormed guards the data every program is built from: a topic
// with a wrong id, level or an empty rule would reach every learner who gets it.
func TestLibraryIsWellFormed(t *testing.T) {
	if !curriculum.Has("en", "ru") || !curriculum.Has("en-US", "ru-RU") {
		t.Fatal("the English library for Russian speakers is gone")
	}

	top, ok := curriculum.MaxLevel("en", "ru")
	if !ok || top != placement.B2 {
		t.Fatalf("the library must reach B2, got %q", top)
	}

	seen := map[string]bool{}
	perLevel := map[string]map[string]int{}

	for _, topic := range curriculum.UpTo("en", "ru", placement.C2) {
		if seen[topic.ID] {
			t.Fatalf("duplicate id %q", topic.ID)
		}

		seen[topic.ID] = true

		if want := strings.ToLower(topic.Level) + "-"; !strings.HasPrefix(topic.ID, want) {
			t.Errorf("%s: the id must start with %q, the file says level %s", topic.ID, want, topic.Level)
		}

		if !curriculum.Skills[topic.Skill] {
			t.Errorf("%s: unknown skill %q", topic.ID, topic.Skill)
		}

		for field, value := range map[string]string{"title": topic.Title, "goal": topic.Goal, "unit": topic.Unit} {
			if strings.TrimSpace(value) == "" {
				t.Errorf("%s: empty %s", topic.ID, field)
			}
		}

		if n := utf8.RuneCountInString(topic.Title); n > 120 {
			t.Errorf("%s: the title is %d characters, a program topic allows 120", topic.ID, n)
		}

		if n := utf8.RuneCountInString(topic.Goal); n > 400 {
			t.Errorf("%s: the goal is %d characters, a program topic allows 400", topic.ID, n)
		}

		if n := utf8.RuneCountInString(topic.Unit); n > 80 {
			t.Errorf("%s: the unit is %d characters, a program topic allows 80", topic.ID, n)
		}

		if n := utf8.RuneCountInString(topic.Theory); n < 150 || n > 1200 {
			t.Errorf("%s: the rule is %d characters, want 150 to 1200", topic.ID, n)
		}

		if len(topic.Tags) == 0 {
			t.Errorf("%s: no tags, so the planner cannot match mistakes to it", topic.ID)
		}

		// Themes and reading need their words: that is what they teach.
		if (topic.Skill == "vocabulary" || topic.Skill == "reading" || topic.Skill == "listening" || topic.Skill == "speaking") && len(topic.Vocabulary) < 6 {
			t.Errorf("%s: %s topics need at least 6 words, has %d", topic.ID, topic.Skill, len(topic.Vocabulary))
		}

		if len(topic.Vocabulary) > 10 {
			t.Errorf("%s: a program topic keeps at most 10 words, has %d", topic.ID, len(topic.Vocabulary))
		}

		for _, w := range topic.Vocabulary {
			if strings.TrimSpace(w.Word) == "" || strings.TrimSpace(w.Translation) == "" {
				t.Errorf("%s: a word without a translation: %+v", topic.ID, w)
			}
		}

		if perLevel[topic.Level] == nil {
			perLevel[topic.Level] = map[string]int{}
		}

		perLevel[topic.Level][topic.Skill]++
	}

	// A learner at any level needs grammar, words, reading, writing, listening and
	// speaking to draw from.
	for _, level := range []placement.Level{placement.A1, placement.A2, placement.B1, placement.B2} {
		counts := perLevel[string(level)]

		if counts["listening"] < 2 || counts["speaking"] < 2 {
			t.Errorf("%s has %d listening and %d speaking topics, want at least 2 of each", level, counts["listening"], counts["speaking"])
		}

		if counts["grammar"] < 8 {
			t.Errorf("%s has %d grammar topics, want at least 8", level, counts["grammar"])
		}

		for _, skill := range []string{"vocabulary", "reading", "writing", "listening", "speaking"} {
			if counts[skill] < 1 {
				t.Errorf("%s has no %s topic", level, skill)
			}
		}
	}
}

func TestUpToAndFind(t *testing.T) {
	a2 := curriculum.UpTo("en", "ru", placement.A2)
	if len(a2) == 0 {
		t.Fatal("nothing up to A2")
	}

	for _, topic := range a2 {
		if placement.Index(placement.Level(topic.Level)) > placement.Index(placement.A2) {
			t.Fatalf("%s is above A2", topic.ID)
		}
	}

	found, ok := curriculum.Find("en", "ru", "b1-present-perfect")
	if !ok || found.Level != "B1" {
		t.Fatalf("find: %+v %t", found, ok)
	}

	if _, ok := curriculum.Find("en", "ru", "nope"); ok {
		t.Fatal("found a topic that does not exist")
	}

	if curriculum.Has("fr", "ru") || curriculum.UpTo("fr", "ru", placement.B2) != nil {
		t.Fatal("French has no library yet")
	}
}

// TestRuleBookIsWellFormed: every grammar topic is a rule with its form,
// examples, a mistake and references, and every reference leads to a rule.
func TestRuleBookIsWellFormed(t *testing.T) {
	rules := curriculum.Rules("en", "ru")
	have := map[string]bool{}

	for _, r := range rules {
		have[r.ID] = true
	}

	for _, topic := range curriculum.UpTo("en", "ru", placement.C2) {
		if topic.Skill == "grammar" && !have[topic.ID] {
			t.Errorf("%s is grammar but has no entry in the rule book", topic.ID)
		}
	}

	for _, r := range rules {
		if r.Skill != "grammar" {
			t.Errorf("%s: only grammar topics are rules", r.ID)
		}

		if len(r.Extra.Forms) < 2 || len(r.Extra.Examples) < 3 || len(r.Extra.Mistakes) < 1 || len(r.Extra.Related) < 2 {
			t.Errorf("%s: a rule needs 2 forms, 3 examples, a mistake and 2 references, has %d/%d/%d/%d",
				r.ID, len(r.Extra.Forms), len(r.Extra.Examples), len(r.Extra.Mistakes), len(r.Extra.Related))
		}

		for _, ex := range r.Extra.Examples {
			if strings.TrimSpace(ex.Text) == "" || strings.TrimSpace(ex.Translation) == "" {
				t.Errorf("%s: an example without a translation: %+v", r.ID, ex)
			}
		}

		for _, m := range r.Extra.Mistakes {
			if m.Wrong == "" || m.Right == "" || m.Wrong == m.Right {
				t.Errorf("%s: a mistake must differ from its correction: %+v", r.ID, m)
			}
		}

		for _, id := range r.Extra.Related {
			if id == r.ID {
				t.Errorf("%s refers to itself", r.ID)
			}

			if !have[id] {
				t.Errorf("%s refers to %q, which is not a rule", r.ID, id)
			}
		}
	}
}

// TestMatchGivesAgentWrittenTopicsTheirRule: a topic an agent wrote on its own
// is recognised by its title and gets the library's rule.
func TestMatchGivesAgentWrittenTopicsTheirRule(t *testing.T) {
	cases := []struct {
		title string
		want  string
	}{
		{"Past Perfect: действия до момента в прошлом", "b2-past-perfect"},
		{"Past Perfect в связках by the time, when, before, after", "b2-past-perfect"},
		{"Present Perfect Continuous: как долго и результат", "b2-present-perfect-continuous"},
		{"Present Perfect: опыт", "b1-present-perfect"},
		{"Present Simple или Present Continuous", "a2-present-simple-vs-continuous"},
	}

	for _, c := range cases {
		got, ok := curriculum.Match("en", "ru", c.title, placement.C2)
		if !ok || got.ID != c.want {
			t.Errorf("Match(%q) = %q %t, want %q", c.title, got.ID, ok, c.want)
		}
	}

	if _, ok := curriculum.Match("en", "ru", "Past Perfect: действия до момента в прошлом", placement.B1); ok {
		t.Error("a B2 rule must not match a B1 learner's topic")
	}

	if _, ok := curriculum.Match("en", "ru", "Чтение коротких текстов", placement.C2); ok {
		t.Error("a topic that is not about a rule must not match one")
	}

	if _, ok := curriculum.Match("fr", "ru", "Past Perfect", placement.C2); ok {
		t.Error("French has no rule book")
	}
}

// TestTheEnglishLibraryMirrorsTheRussianOne: a learner who asks for explanations
// in English gets the same topics and the same rules, only written in English.
// Everything the program and the rule book key on (ids, levels, skills, the
// order of the examples they translate by, the references between rules) must
// be identical, or a program built in one language would not open in the other.
func TestTheEnglishLibraryMirrorsTheRussianOne(t *testing.T) {
	if !curriculum.Has("en", "en") {
		t.Fatal("the English library explained in English is gone")
	}

	cyrillic := func(s string) bool {
		return strings.ContainsAny(strings.ToLower(s), "абвгдеёжзийклмнопрстуфхцчшщъыьэюя")
	}

	ru := curriculum.UpTo("en", "ru", placement.C2)
	en := curriculum.UpTo("en", "en", placement.C2)

	if len(ru) != len(en) {
		t.Fatalf("%d topics in Russian, %d in English", len(ru), len(en))
	}

	for i, r := range ru {
		e := en[i]

		if e.ID != r.ID || e.Level != r.Level || e.Skill != r.Skill {
			t.Fatalf("topic %d: %s/%s/%s in English, %s/%s/%s in Russian", i, e.ID, e.Level, e.Skill, r.ID, r.Level, r.Skill)
		}

		for field, value := range map[string]string{"title": e.Title, "goal": e.Goal, "unit": e.Unit, "theory": e.Theory} {
			if strings.TrimSpace(value) == "" || cyrillic(value) {
				t.Errorf("%s: the English %s is empty or not English: %q", e.ID, field, value)
			}
		}

		if n := utf8.RuneCountInString(e.Theory); n < 150 || n > 1200 {
			t.Errorf("%s: the English rule is %d characters, want 150 to 1200", e.ID, n)
		}

		if n := utf8.RuneCountInString(e.Title); n > 120 {
			t.Errorf("%s: the English title is %d characters, a program topic allows 120", e.ID, n)
		}

		if n := utf8.RuneCountInString(e.Goal); n > 400 {
			t.Errorf("%s: the English goal is %d characters, a program topic allows 400", e.ID, n)
		}

		if len(e.Tags) == 0 {
			t.Errorf("%s: no English tags, so the planner cannot match mistakes to it", e.ID)
		}

		// Key words stay translated for the learner's own language, in the Russian book.
		if len(e.Vocabulary) != 0 {
			t.Errorf("%s: the English library carries no words of its own", e.ID)
		}

		if (r.Extra == nil) != (e.Extra == nil) {
			t.Fatalf("%s: a rule in one language only", e.ID)
		}

		if r.Extra == nil {
			continue
		}

		if len(e.Extra.Forms) != len(r.Extra.Forms) || len(e.Extra.Examples) != len(r.Extra.Examples) || len(e.Extra.Mistakes) != len(r.Extra.Mistakes) {
			t.Errorf("%s: the rule is built differently: %d/%d/%d forms, examples, mistakes in English, %d/%d/%d in Russian",
				e.ID, len(e.Extra.Forms), len(e.Extra.Examples), len(e.Extra.Mistakes), len(r.Extra.Forms), len(r.Extra.Examples), len(r.Extra.Mistakes))

			continue
		}

		if strings.Join(e.Extra.Related, ",") != strings.Join(r.Extra.Related, ",") {
			t.Errorf("%s: it refers to %v in English and %v in Russian", e.ID, e.Extra.Related, r.Extra.Related)
		}

		for _, f := range e.Extra.Forms {
			if cyrillic(f) {
				t.Errorf("%s: a form is not in English: %q", e.ID, f)
			}
		}

		for _, ex := range e.Extra.Examples {
			if strings.TrimSpace(ex.Text) == "" || cyrillic(ex.Text) {
				t.Errorf("%s: an example sentence is empty or not English: %q", e.ID, ex.Text)
			}

			// The translation comes from the Russian book, by position.
			if ex.Translation != "" {
				t.Errorf("%s: the English book leaves the translating to the Russian one: %+v", e.ID, ex)
			}
		}

		for _, m := range e.Extra.Mistakes {
			if m.Wrong == "" || m.Right == "" || m.Wrong == m.Right || cyrillic(m.Note) {
				t.Errorf("%s: a mistake must differ from its correction and be explained in English: %+v", e.ID, m)
			}
		}
	}
}
