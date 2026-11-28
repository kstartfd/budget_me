// Package assessment defines the Quick Screening blueprint and its measurement policy.
package assessment

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"learning_app/backend/internal/learningpath/curriculum"
	"learning_app/backend/internal/learningpath/placement"
)

const (
	QuickScreening       = "quick_screening"
	FullAssessment       = "full_assessment"
	SkillAssessment      = "skill_assessment"
	KnowledgeDiagnostic  = "knowledge_diagnostic"
	PeriodicReassessment = "periodic_reassessment"
	BlueprintVersion     = "grammar-quick-v1"
)

const (
	Created    = "created"
	InProgress = "in_progress"
	Paused     = "paused"
	Evaluating = "evaluating"
	Completed  = "completed"
	Partial    = "partial"
	Failed     = "failed"
	Cancelled  = "cancelled"
)

type Format string

const (
	MultipleChoice         Format = "multiple_choice"
	OpenGapFill            Format = "open_gap_fill"
	ErrorCorrection        Format = "error_correction"
	SentenceTransformation Format = "sentence_transformation"
	VocabularyInContext    Format = "vocabulary_in_context"
	ReadingComprehension   Format = "reading_comprehension"
	Writing                Format = "writing"
	Listening              Format = "listening"
	Speaking               Format = "speaking"
	Pronunciation          Format = "pronunciation"
)

type Target struct {
	ItemKey   string          `json:"knowledgeItemKey"`
	Category  string          `json:"category"`
	Objective string          `json:"learningObjective"`
	Skill     placement.Skill `json:"skill"`
	Band      placement.Level `json:"difficultyBand"`
	Format    Format          `json:"questionType"`
}

type Content struct {
	Text    string   `json:"text"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
	GradingKey
}

type GradingKey struct {
	Answers        []string `json:"answers"`
	PartialAnswers []string `json:"partialAnswers,omitempty"`
}

type Question struct {
	Target
	Content
	Origin string
}

func (q Question) Fingerprint() string {
	return digest([]byte(normalize(q.Text) + "\n" + normalize(q.Prompt)))
}

func digest(body []byte) string { return fmt.Sprintf("%x", sha256.Sum256(body)) }

//go:embed bank.txt
var bankSource string

type bankEntry struct {
	language string
	question Question
}

var bank = loadBank()

func loadBank() []bankEntry {
	var out []bankEntry
	for i, line := range strings.Split(bankSource, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 8 || len(fields) > 10 {
			panic(fmt.Sprintf("assessment bank: invalid line %d", i+1))
		}
		q := Question{
			Target: Target{ItemKey: fields[2], Category: fields[3], Skill: placement.SkillGrammar,
				Band: placement.Level(fields[1]), Format: Format(fields[4])},
			Content: Content{Text: fields[5], Prompt: fields[6], Options: []string{}, GradingKey: GradingKey{Answers: strings.Split(fields[7], ";")}},
			Origin:  "curated",
		}
		if len(fields) >= 9 && fields[8] != "" {
			q.Options = strings.Split(fields[8], ";")
		}
		if len(fields) == 10 {
			q.PartialAnswers = strings.Split(fields[9], ";")
		}
		if err := Validate(q.Target, q.Content); err != nil {
			panic(fmt.Sprintf("assessment bank: line %d: %v", i+1, err))
		}
		out = append(out, bankEntry{language: fields[0], question: q})
	}
	return out
}

// Bank resolves every task to a grammar rule and a localized objective in the
// learner's Rule Book. A pair without the required canonical rules is unavailable.
func Bank(language, explanation string) ([]Question, bool) {
	var questions []Question
	for _, entry := range bank {
		if entry.language != language {
			continue
		}
		q := entry.question
		item, ok := curriculum.Find(language, explanation, q.ItemKey)
		if !ok || item.Extra == nil || item.Skill != string(q.Skill) || item.Level != string(q.Band) || strings.TrimSpace(item.Goal) == "" {
			return nil, false
		}
		q.Objective = item.Goal
		questions = append(questions, q)
	}
	return questions, len(questions) != 0
}

// Slot is the exact target the backend chose and an optional unseen bank item.
type Slot struct {
	Target
	Bank *Question
}

// Select balances item exposure first and category coverage within each batch
// second. Old prompts are never offered again. Generation fills gaps in the pool.
func Select(bank []Question, band placement.Level, count int, seen map[string]bool, itemCounts map[string]int) []Slot {

	categoryUse := make(map[string]int)
	used := make(map[string]bool)
	var out []Slot
	for range count {
		best := -1
		for i, q := range bank {
			if q.Band != band || used[q.ItemKey] {
				continue
			}
			if best == -1 || prefer(q, bank[best], seen, itemCounts, categoryUse) {
				best = i
			}
		}
		if best == -1 {
			break
		}
		q := bank[best]
		slot := Slot{Target: q.Target}
		if !seen[q.Fingerprint()] {
			slot.Bank = &q
		}
		out = append(out, slot)
		used[q.ItemKey] = true
		categoryUse[q.Category]++
	}
	return out
}

func prefer(a, b Question, seen map[string]bool, counts, categoryUse map[string]int) bool {
	if categoryUse[a.Category] != categoryUse[b.Category] {
		return categoryUse[a.Category] < categoryUse[b.Category]
	}
	aSeen, bSeen := seen[a.Fingerprint()], seen[b.Fingerprint()]
	if aSeen != bSeen {
		return !aSeen
	}
	return counts[a.ItemKey] < counts[b.ItemKey]
}

func Validate(target Target, body Content) error {
	if len(strings.TrimSpace(body.Prompt)) == 0 || utf8.RuneCountInString(body.Prompt) > 600 || utf8.RuneCountInString(body.Text) > 1600 {
		return fmt.Errorf("invalid prompt or context length")
	}
	if len(body.Answers) == 0 || len(body.Answers) > 12 || len(body.PartialAnswers) > 4 ||
		!distinct(slices.Concat(body.Answers, body.PartialAnswers), 200) {
		return fmt.Errorf("invalid accepted answers")
	}
	switch target.Format {
	case MultipleChoice, VocabularyInContext, ReadingComprehension:
		if len(body.PartialAnswers) != 0 || len(body.Options) != 4 || !distinct(body.Options, 200) || len(body.Answers) != 1 || !slices.Contains(body.Options, body.Answers[0]) {
			return fmt.Errorf("expected four distinct options and one exact answer")
		}
		if target.Format != MultipleChoice && strings.TrimSpace(body.Text) == "" {
			return fmt.Errorf("context is required")
		}
	case OpenGapFill, ErrorCorrection, SentenceTransformation:
		if len(body.Options) != 0 {
			return fmt.Errorf("open questions must not offer choices")
		}
		if target.Format == OpenGapFill || target.Format == SentenceTransformation {
			if strings.Count(body.Prompt, "___") != 1 {
				return fmt.Errorf("expected exactly one gap")
			}
		}
		if target.Format == ErrorCorrection && strings.Count(body.Text, body.Prompt) != 1 {
			return fmt.Errorf("marked error must appear exactly once in the sentence")
		}
		if target.Format != OpenGapFill && strings.TrimSpace(body.Text) == "" {
			return fmt.Errorf("source sentence is required")
		}
	default:
		return fmt.Errorf("unsupported question format")
	}
	return nil
}

func distinct(values []string, maxRunes int) bool {
	seen := make(map[string]bool, len(values))
	for _, s := range values {
		key := normalize(s)
		if key == "" || seen[key] || utf8.RuneCountInString(s) > maxRunes {
			return false
		}
		seen[key] = true
	}
	return true
}

func normalize(s string) string {
	s = strings.NewReplacer("’", "'", "‘", "'").Replace(s)
	return norm.NFC.String(strings.Join(strings.Fields(strings.ToLower(s)), " "))
}

func Grade(q Question, answer string, skip bool) (Outcome, error) {
	if skip || strings.TrimSpace(answer) == "" {
		return Skipped, nil
	}
	if len(q.Options) > 0 && !slices.Contains(q.Options, answer) {
		return "", fmt.Errorf("answer must be one of the options")
	}
	for _, key := range q.Answers {
		if normalize(key) == normalize(answer) {
			return Correct, nil
		}
	}
	for _, key := range q.PartialAnswers {
		if normalize(key) == normalize(answer) {
			return PartialCredit, nil
		}
	}
	return Incorrect, nil
}

// Prerequisites records the explicit links the screening path can use today.
// Rule Book "related" links are navigational and do not imply prerequisites.
func Prerequisites(language, item string) []string {
	return prerequisites[language][item]
}

var prerequisites = map[string]map[string][]string{
	"en": {
		"a1-present-simple-questions":   {"a1-present-simple"},
		"a1-there-is":                   {"a1-to-be", "a1-plurals"},
		"a2-past-simple-questions":      {"a2-past-simple-irregular"},
		"b1-present-perfect":            {"a2-past-simple-irregular"},
		"b1-passive":                    {"a1-to-be", "a2-past-simple-irregular"},
		"b2-present-perfect-continuous": {"b1-present-perfect", "a2-present-continuous"},
		"b2-past-perfect":               {"b1-present-perfect"},
		"b2-third-conditional":          {"b1-second-conditional", "b2-past-perfect"},
		"b2-passive-advanced":           {"b1-passive"},
		"b2-reported-advanced":          {"a2-past-simple-questions"},
	},
	"es": {
		"a1-gustar":                {"a1-presente-regular"},
		"a2-adjetivos":             {"a1-plural"},
		"b1-pluscuamperfecto":      {"a2-preterito-imperfecto"},
		"b1-pronombres-combinados": {"a2-pronombres-complemento"},
		"b2-subjuntivo-imperfecto": {"b1-subjuntivo-presente"},
		"b2-condicionales-si":      {"b1-condicional", "b2-subjuntivo-imperfecto"},
		"b2-temporales-subjuntivo": {"b1-subjuntivo-presente"},
		"b2-tiempos-compuestos":    {"b1-pluscuamperfecto"},
		"b2-pasiva":                {"a1-ser", "a2-adjetivos"},
		"b2-estilo-indirecto":      {"b1-pluscuamperfecto"},
	},
}
