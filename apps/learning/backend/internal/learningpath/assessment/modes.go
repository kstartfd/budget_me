package assessment

import "fmt"

// AssessedSkill is independent of the legacy placement.Skill, which only knows
// grammar, vocabulary and reading. A skill is exposed here only when its whole
// assessment workflow can run; a format name alone does not enable that skill.
type AssessedSkill string

const (
	GrammarSkill       AssessedSkill = "grammar"
	VocabularySkill    AssessedSkill = "vocabulary"
	ReadingSkill       AssessedSkill = "reading"
	ListeningSkill     AssessedSkill = "listening"
	WritingSkill       AssessedSkill = "writing"
	SpeakingSkill      AssessedSkill = "speaking"
	PronunciationSkill AssessedSkill = "pronunciation"
)

type Evaluator string

const (
	KeyedEvaluator       Evaluator = "keyed"
	RubricEvaluator      Evaluator = "rubric"
	AudioRubricEvaluator Evaluator = "audio_rubric"
)

// SectionBlueprint describes a skill-specific measurement contract. Its policy
// values come from the snapshot stored with the assessment for durable resume.
type SectionBlueprint struct {
	Skill        AssessedSkill `json:"skill"`
	Formats      []Format      `json:"formats"`
	Evaluator    Evaluator     `json:"evaluator"`
	MinQuestions int           `json:"minQuestions"`
	MaxQuestions int           `json:"maxQuestions"`
}

// ModeBlueprint composes one or more skill sections. Only implemented, verified
// slices belong in RuntimeMode; target-mode plans live in ASSESSMENT_MODES.md.
type ModeBlueprint struct {
	Type     string             `json:"type"`
	Version  string             `json:"version"`
	Sections []SectionBlueprint `json:"sections"`
}

// Capability is deliberately narrower than the product roadmap: a client may
// offer only returned modes and skills. Missing modes are unavailable.
type Capability struct {
	Type             string   `json:"type"`
	BlueprintVersion string   `json:"blueprintVersion"`
	Skills           []string `json:"skills"`
}

func AvailableModes(language string, policy Policy) []Capability {
	available := []Capability{}
	mode, ok := RuntimeMode(QuickScreening, policy)
	if !ok {
		return available
	}
	if _, ok := Bank(language, language); !ok {
		return available
	}
	skills := make([]string, 0, len(mode.Sections))
	for _, section := range mode.Sections {
		skills = append(skills, string(section.Skill))
	}
	return append(available, Capability{Type: mode.Type, BlueprintVersion: mode.Version, Skills: skills})
}

func (b ModeBlueprint) Validate() error {
	if b.Type == "" || b.Version == "" || len(b.Sections) == 0 {
		return fmt.Errorf("assessment mode needs a type, version and sections")
	}
	seen := make(map[AssessedSkill]bool, len(b.Sections))
	for _, section := range b.Sections {
		if seen[section.Skill] || len(section.Formats) == 0 || section.MinQuestions <= 0 ||
			section.MaxQuestions < section.MinQuestions {
			return fmt.Errorf("invalid or duplicate assessment section %q", section.Skill)
		}
		seen[section.Skill] = true
		allowed, ok := allowedFormats[section.Skill]
		if !ok {
			return fmt.Errorf("unsupported assessment skill %q", section.Skill)
		}
		formats := make(map[Format]bool, len(section.Formats))
		for _, format := range section.Formats {
			if formats[format] || !allowed[format] {
				return fmt.Errorf("invalid format %q for %s", format, section.Skill)
			}
			formats[format] = true
		}
		if section.Evaluator != evaluatorFor[section.Skill] {
			return fmt.Errorf("invalid evaluator %q for %s", section.Evaluator, section.Skill)
		}
	}
	return nil
}

var allowedFormats = map[AssessedSkill]map[Format]bool{
	GrammarSkill: {
		MultipleChoice: true, OpenGapFill: true, ErrorCorrection: true,
		SentenceTransformation: true, VocabularyInContext: true, ReadingComprehension: true,
	},
	VocabularySkill:    {MultipleChoice: true, VocabularyInContext: true, OpenGapFill: true},
	ReadingSkill:       {ReadingComprehension: true},
	ListeningSkill:     {Listening: true},
	WritingSkill:       {Writing: true},
	SpeakingSkill:      {Speaking: true},
	PronunciationSkill: {Pronunciation: true},
}

var evaluatorFor = map[AssessedSkill]Evaluator{
	GrammarSkill: KeyedEvaluator, VocabularySkill: KeyedEvaluator, ReadingSkill: KeyedEvaluator,
	ListeningSkill: KeyedEvaluator, WritingSkill: RubricEvaluator,
	SpeakingSkill: AudioRubricEvaluator, PronunciationSkill: AudioRubricEvaluator,
}

// RuntimeMode advertises only a workflow that the backend can currently run.
// The target Quick mode has three skills, but vocabulary/reading are not yet
// implemented as independent sections, so the current version contains grammar.
func RuntimeMode(kind string, policy Policy) (ModeBlueprint, bool) {
	if kind != QuickScreening || policy.Validate() != nil {
		return ModeBlueprint{}, false
	}
	b := ModeBlueprint{
		Type: kind, Version: policy.Version,
		Sections: []SectionBlueprint{{
			Skill: GrammarSkill, Evaluator: KeyedEvaluator,
			Formats: []Format{MultipleChoice, OpenGapFill, ErrorCorrection, SentenceTransformation,
				VocabularyInContext, ReadingComprehension},
			MinQuestions: policy.MinQuestions, MaxQuestions: policy.MaxQuestions,
		}},
	}
	return b, b.Validate() == nil
}
