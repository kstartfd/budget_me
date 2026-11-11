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

		var list []Topic
		if err := json.Unmarshal(raw, &list); err != nil {
			panic(fmt.Sprintf("curriculum: %s: %v", e.Name(), err))
		}

		key := parts[0] + "_" + parts[1]
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

// Find returns one topic by its id.
func Find(language, native, id string) (Topic, bool) {
	c := lookup(language, native)
	if c == nil {
		return Topic{}, false
	}

	t, ok := c.byID[id]

	return t, ok
}
