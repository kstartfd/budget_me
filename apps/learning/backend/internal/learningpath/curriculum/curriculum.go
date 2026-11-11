// Package curriculum is the library of study topics a program is built from.
// The rules, the key words and the goals are written and checked by people,
// not made up by an agent on the fly: an agent that invents a grammar rule can
// invent a wrong one and the learner will trust it. The agent only chooses
// which topics a learner needs, in which order, and writes exercises for them.
//
// A catalog is one JSON file per language pair and CEFR level, named
// <language>_<native>_<level>.json under data/ (for example en_ru_b1.json):
// the learned language, the language the rules are explained in, and the
// level. A pair with no files has no catalog, and programs for it are still
// planned by the agent alone.
package curriculum

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"learning_app/backend/internal/learningpath/placement"
)

//go:embed data/*.json
var files embed.FS

// Word is one key word of a topic with its translation.
type Word struct {
	Word        string `json:"word"`
	Translation string `json:"translation"`
}

// Example is a sentence that shows a rule at work, with its translation.
type Example struct {
	Text        string `json:"text"`
	Translation string `json:"translation"`
}

// Mistake is a typical error against a rule and its correction.
type Mistake struct {
	Wrong string `json:"wrong"`
	Right string `json:"right"`
	Note  string `json:"note"`
}

// Extra is what turns a grammar topic into an entry of the rule book: how the
// rule is built, sentences that show it, the mistakes made against it and the
// other rules it is read together with. It lives in <language>_<native>_rules.json,
// keyed by topic id.
type Extra struct {
	Forms    []string  `json:"forms"`
	Examples []Example `json:"examples"`
	Mistakes []Mistake `json:"mistakes"`
	Related  []string  `json:"related"`
}

// Topic is one lesson of the library: a rule or a theme, what the learner can
// do once it is learnt, and the words that go with it.
type Topic struct {
	ID         string   `json:"id"`
	Level      string   `json:"-"` // from the file name
	Skill      string   `json:"skill"`
	Unit       string   `json:"unit"`
	Title      string   `json:"title"`
	Goal       string   `json:"goal"`
	Theory     string   `json:"theory"`
	Vocabulary []Word   `json:"vocabulary"`
	Tags       []string `json:"tags"`
	// Extra is set for the grammar topics that are also rules of the rule book.
	Extra *Extra `json:"-"`
}

// Skills a topic can teach, the same set a program topic carries.
var Skills = map[string]bool{"grammar": true, "vocabulary": true, "reading": true, "writing": true, "listening": true, "speaking": true}

type catalog struct {
	topics []Topic
	byID   map[string]Topic
}

// catalogs is keyed by "<language>_<native>".
var catalogs = map[string]*catalog{}

func init() {
	entries, err := fs.ReadDir(files, "data")
	if err != nil {
		panic(err)
	}

	extras := map[string]map[string]Extra{}

	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".json")

		parts := strings.Split(name, "_")
		if len(parts) != 3 {
			panic(fmt.Sprintf("curriculum: bad file name %q, want <language>_<native>_<level>.json", e.Name()))
		}

		raw, err := files.ReadFile(path.Join("data", e.Name()))
		if err != nil {
			panic(err)
		}

		key := parts[0] + "_" + parts[1]

		// The rule book of a pair: extras by topic id, attached below once
		// every level is loaded.
		if parts[2] == "rules" {
			var rules map[string]Extra
			if err := json.Unmarshal(raw, &rules); err != nil {
				panic(fmt.Sprintf("curriculum: %s: %v", e.Name(), err))
			}

			extras[key] = rules

			continue
		}

		var list []Topic
		if err := json.Unmarshal(raw, &list); err != nil {
			panic(fmt.Sprintf("curriculum: %s: %v", e.Name(), err))
		}

		if catalogs[key] == nil {
			catalogs[key] = &catalog{byID: map[string]Topic{}}
		}

		c := catalogs[key]

		for _, t := range list {
			t.Level = strings.ToUpper(parts[2])
			c.topics = append(c.topics, t)
			c.byID[t.ID] = t
		}
	}

	for key, rules := range extras {
		c := catalogs[key]
		if c == nil {
			panic(fmt.Sprintf("curriculum: rules for %s have no topics", key))
		}

		for i := range c.topics {
			if extra, ok := rules[c.topics[i].ID]; ok {
				e := extra
				c.topics[i].Extra = &e
				c.byID[c.topics[i].ID] = c.topics[i]
			}
		}
	}

	for _, c := range catalogs {
		sort.SliceStable(c.topics, func(i, j int) bool {
			return placement.Index(placement.Level(c.topics[i].Level)) < placement.Index(placement.Level(c.topics[j].Level))
		})
	}
}

// base reduces a language tag to its language: en-US is en.
func base(tag string) string {
	tag = strings.ToLower(tag)
	if i := strings.IndexAny(tag, "-_"); i >= 0 {
		return tag[:i]
	}

	return tag
}

func lookup(language, native string) *catalog {
	return catalogs[base(language)+"_"+base(native)]
}

// Has tells whether a library exists for the language pair.
func Has(language, native string) bool {
	return lookup(language, native) != nil
}

// MaxLevel is the highest rung the library covers for the pair, or false when
// there is no library.
func MaxLevel(language, native string) (placement.Level, bool) {
	c := lookup(language, native)
	if c == nil {
		return "", false
	}

	top := -1
	for _, t := range c.topics {
		top = max(top, placement.Index(placement.Level(t.Level)))
	}

	if top < 0 {
		return "", false
	}

	return placement.Levels[top], true
}

// UpTo returns the topics of a pair from the first rung up to and including
// level, ordered by level and then as written.
func UpTo(language, native string, level placement.Level) []Topic {
	c := lookup(language, native)
	if c == nil {
		return nil
	}

	limit := placement.Index(level)

	var out []Topic

	for _, t := range c.topics {
		if placement.Index(placement.Level(t.Level)) <= limit {
			out = append(out, t)
		}
	}

	return out
}

// Rules returns the rule book of a pair: the topics that carry a rule, from
// the first rung up, or nil when the pair has none.
func Rules(language, native string) []Topic {
	c := lookup(language, native)
	if c == nil {
		return nil
	}

	var out []Topic

	for _, t := range c.topics {
		if t.Extra != nil {
			out = append(out, t)
		}
	}

	return out
}

// ruleKey is what a topic title is recognised by: the part before a colon, or
// the whole title, lowered ("Past Perfect: что было до прошлого" is "past perfect").
func ruleKey(title string) string {
	if i := strings.Index(title, ":"); i > 0 {
		title = title[:i]
	}

	return strings.ToLower(strings.TrimSpace(title))
}

// Match finds the rule a topic written without the library is about: the rule
// whose key is the longest one the title contains, among the rules up to level.
// An agent writes "Past Perfect: действия до момента в прошлом", and this gives
// that topic the library's rule for Past Perfect.
func Match(language, native, title string, upTo placement.Level) (Topic, bool) {
	c := lookup(language, native)
	if c == nil {
		return Topic{}, false
	}

	title = strings.ToLower(title)
	limit := placement.Index(upTo)

	var best Topic

	bestLen := 0

	for _, t := range c.topics {
		if t.Extra == nil || placement.Index(placement.Level(t.Level)) > limit {
			continue
		}

		key := ruleKey(t.Title)
		if len(key) < 4 || !strings.Contains(title, key) {
			continue
		}

		if len(key) > bestLen {
			best, bestLen = t, len(key)
		}
	}

	return best, bestLen > 0
}

// Find returns one topic by its id.
func Find(language, native, id string) (Topic, bool) {
	c := lookup(language, native)
	if c == nil {
		return Topic{}, false
	}

	t, ok := c.byID[id]

	return t, ok
}
