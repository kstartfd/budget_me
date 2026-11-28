package assessment

import (
	"cmp"
	"fmt"
	"slices"

	"learning_app/backend/internal/learningpath/placement"
)

type Policy struct {
	Version       string `json:"version"`
	DefaultBand   string `json:"defaultBand"`
	BatchSize     int    `json:"batchSize"`
	MinQuestions  int    `json:"minQuestions"`
	MaxQuestions  int    `json:"maxQuestions"`
	MinCategories int    `json:"minCategories"`
}

func QuickPolicy() Policy {
	return Policy{Version: BlueprintVersion, DefaultBand: "A2", BatchSize: 6, MinQuestions: 12, MaxQuestions: 30, MinCategories: 3}
}

func (p Policy) Validate() error {
	if p.Version != BlueprintVersion || !ValidBand(placement.Level(p.DefaultBand)) ||
		p.BatchSize < 3 || p.BatchSize > 6 || p.MinQuestions < 2*p.BatchSize ||
		p.MaxQuestions < p.MinQuestions || p.MaxQuestions > 60 || p.MinCategories < 2 || p.MinCategories > 4 {
		return fmt.Errorf("invalid Quick Screening policy")
	}
	return nil
}

func ValidBand(level placement.Level) bool {
	return placement.Index(level) >= 0 && placement.Index(level) <= placement.Index(placement.B2)
}

type Outcome string

const (
	Correct               Outcome = "correct"
	PartialCredit         Outcome = "partial"
	Incorrect             Outcome = "incorrect"
	Skipped               Outcome = "skipped"
	EvaluationUnavailable Outcome = "evaluation_unavailable"
)

type Observation struct {
	Target
	Outcome    Outcome
	EvidenceID string
	Batch      int
	Assisted   bool
	Repeated   bool
	Origin     string
}

func (o Observation) Usable() bool {
	return !o.Assisted && !o.Repeated && (o.Outcome == Correct || o.Outcome == Incorrect || o.Outcome == PartialCredit)
}

func (o Observation) EvidenceMode() string {
	switch o.Format {
	case MultipleChoice, VocabularyInContext, ReadingComprehension:
		return "recognition"
	case SentenceTransformation, ErrorCorrection:
		return "production"
	default:
		return "recall"
	}
}

func (o Observation) ScoreWeight() (float64, float64) {
	score := 0.0
	switch o.Outcome {
	case Correct:
		score = 1
	case PartialCredit:
		score = 0.5
	}
	if !o.Usable() {
		return score, 0
	}
	weight := 1.0
	if o.EvidenceMode() == "recognition" {
		weight = 0.4
	}
	if o.Origin == "ai" {
		weight *= 0.7
	}
	return score, weight
}

type Batch struct {
	Issued       int
	Band         placement.Level
	Observations []Observation
}

type DifficultyRange struct {
	Lower    placement.Level `json:"lower"`
	Upper    placement.Level `json:"upper"`
	Boundary string          `json:"boundary"`
}

type Decision struct {
	Next       placement.Level
	StopReason string
	Range      *DifficultyRange
	Enough     bool
	Conflict   bool
}

type signal int

const (
	inconclusive signal = iota
	positive
	negative
)

func batchSignal(batch Batch) signal {
	cats := map[string]bool{}
	correct, incorrect, curated := 0, 0, 0
	for _, o := range batch.Observations {
		if !o.Usable() {
			continue
		}
		switch o.Outcome {
		case Correct:
			correct++
		case Incorrect:
			incorrect++
		default:
			continue
		}
		cats[o.Category] = true
		if o.Origin == "curated" {
			curated++
		}
	}
	n := len(batch.Observations)
	if n < 3 || n != batch.Issued || len(cats) < 2 || curated < 2 {
		return inconclusive
	}
	// Ceiling of 5/6 correct or 4/6 incorrect over the issued batch. A skip,
	// partial credit, invalid key or assisted answer cannot supply a vote.
	switch {
	case 6*correct >= 5*n:
		return positive
	case 6*incorrect >= 4*n:
		return negative
	default:
		return inconclusive
	}
}

func Decide(p Policy, current placement.Level, batches []Batch) Decision {
	d := Decision{Next: current}
	if len(batches) == 0 {
		return d
	}
	var positives, negatives [4]int
	cats := map[string]bool{}
	total, usable := 0, 0
	for _, batch := range batches {
		total += len(batch.Observations)
		for _, o := range batch.Observations {
			if o.Usable() {
				cats[o.Category] = true
				usable++
			}
		}
		i := placement.Index(batch.Band)
		if i < 0 || i >= len(positives) {
			continue
		}
		switch batchSignal(batch) {
		case positive:
			positives[i]++
		case negative:
			negatives[i]++
		}
	}
	highPositive, lowNegative := -1, len(negatives)
	for i := range positives {
		switch {
		case positives[i] > negatives[i]:
			highPositive = i
		case negatives[i] > positives[i]:
			lowNegative = min(lowNegative, i)
		case positives[i] > 0:
			d.Conflict = true
		}
	}
	d.Conflict = d.Conflict || highPositive >= lowNegative

	if !d.Conflict {
		last := len(positives) - 1
		switch {
		case highPositive == last && positives[last]-negatives[last] >= 2:
			d.Range = &DifficultyRange{placement.B2, placement.B2, "ceiling"}
		case lowNegative == 0 && negatives[0]-positives[0] >= 2:
			d.Range = &DifficultyRange{placement.A1, placement.A1, "floor"}
		case highPositive >= 0 && lowNegative < len(negatives) && lowNegative-highPositive == 1:
			d.Range = &DifficultyRange{placement.Levels[highPositive], placement.Levels[lowNegative], "bracketed"}
		}
	}
	d.Enough = d.Range != nil && usable >= p.MinQuestions && len(cats) >= p.MinCategories
	switch {
	case d.Enough:
		d.StopReason = "difficulty_boundary"
	case total >= p.MaxQuestions:
		d.StopReason = "max_questions"
	}
	if d.StopReason != "" || d.Conflict || d.Range != nil {
		return d
	}

	i := placement.Index(current)
	switch batchSignal(batches[len(batches)-1]) {
	case positive:
		d.Next = placement.Levels[min(i+1, len(positives)-1)]
	case negative:
		d.Next = placement.Levels[max(i-1, 0)]
	}
	return d
}

const (
	Demonstrated      = "demonstrated"
	NeedsPractice     = "needs_practice"
	NeedsMoreEvidence = "needs_more_evidence"
	NotAssessed       = "not_assessed"
)

type KnowledgeOutcome struct {
	ItemKey     string   `json:"knowledgeItemKey"`
	Category    string   `json:"category"`
	Objective   string   `json:"learningObjective"`
	Status      string   `json:"status"`
	GapSupport  string   `json:"gapSupport,omitempty"`
	EvidenceIDs []string `json:"evidenceIds"`
}

type SkillEstimate struct {
	Skill           string           `json:"skill"`
	DifficultyRange *DifficultyRange `json:"difficultyRange"`
	UsableEvidence  int              `json:"usableEvidence"`
}

// SectionResult is the stable result shape for one measured skill. Quick
// Screening currently has only a grammar section; future modes can aggregate
// independent sections without conflating missing skills with low scores.
type SectionResult struct {
	Skill               string           `json:"skill"`
	Status              string           `json:"status"`
	StopReason          string           `json:"stopReason"`
	EvidenceSufficiency string           `json:"evidenceSufficiency"`
	DifficultyRange     *DifficultyRange `json:"difficultyRange"`
	UsableEvidence      int              `json:"usableEvidence"`
	EvidenceIDs         []string         `json:"evidenceIds"`
}

func CompletionStatus(d Decision, evidenceCount int) string {
	switch {
	case d.StopReason == "cancelled":
		return Cancelled
	case d.StopReason == "question_generation_unavailable" && evidenceCount == 0:
		return Failed
	case d.Enough:
		return Completed
	default:
		return Partial
	}
}

type Result struct {
	AssessmentID                  string             `json:"assessmentId"`
	LearnerProfileID              string             `json:"learnerProfileId"`
	AssessmentType                string             `json:"assessmentType"`
	BlueprintVersion              string             `json:"blueprintVersion"`
	StopReason                    string             `json:"stopReason"`
	DifficultyRange               *DifficultyRange   `json:"difficultyRange"`
	EvidenceSufficiency           string             `json:"evidenceSufficiency"`
	ConflictingEvidence           bool               `json:"conflictingEvidence"`
	SkillEstimates                []SkillEstimate    `json:"skillEstimates"`
	Sections                      []SectionResult    `json:"sections"`
	UnassessedSkills              []string           `json:"unassessedSkills"`
	KnowledgeOutcomes             []KnowledgeOutcome `json:"knowledgeOutcomes"`
	DemonstratedStrengths         []KnowledgeOutcome `json:"demonstratedStrengths"`
	KnowledgeGaps                 []KnowledgeOutcome `json:"knowledgeGaps"`
	RecommendedKnowledgeChecks    []KnowledgeOutcome `json:"recommendedKnowledgeChecks"`
	RecommendedLearningObjectives []KnowledgeOutcome `json:"recommendedLearningObjectives"`
	EvidenceIDs                   []string           `json:"evidenceIds"`
}

func Summarize(p Policy, language string, bank []Question, batches []Batch, d Decision) Result {
	out := Result{
		BlueprintVersion: p.Version, StopReason: d.StopReason, DifficultyRange: d.Range, ConflictingEvidence: d.Conflict,
		EvidenceSufficiency: "insufficient", SkillEstimates: []SkillEstimate{},
		UnassessedSkills:  []string{"vocabulary", "reading", "writing", "listening", "speaking", "pronunciation"},
		KnowledgeOutcomes: []KnowledgeOutcome{}, DemonstratedStrengths: []KnowledgeOutcome{}, KnowledgeGaps: []KnowledgeOutcome{},
		RecommendedKnowledgeChecks: []KnowledgeOutcome{}, RecommendedLearningObjectives: []KnowledgeOutcome{}, EvidenceIDs: []string{},
	}
	items := map[string][]Observation{}
	usable := 0
	for _, b := range batches {
		for _, o := range b.Observations {
			items[o.ItemKey] = append(items[o.ItemKey], o)
			out.EvidenceIDs = append(out.EvidenceIDs, o.EvidenceID)
			if o.Usable() {
				usable++
			}
		}
	}
	switch {
	case d.Enough:
		out.EvidenceSufficiency = "sufficient_for_screening"
	case usable > 0:
		out.EvidenceSufficiency = "preliminary"
	}
	if usable > 0 {
		out.SkillEstimates = append(out.SkillEstimates, SkillEstimate{"grammar", d.Range, usable})
	} else {
		out.UnassessedSkills = append([]string{"grammar"}, out.UnassessedSkills...)
	}
	out.Sections = []SectionResult{{
		Skill: string(GrammarSkill), Status: CompletionStatus(d, len(out.EvidenceIDs)),
		StopReason: d.StopReason, EvidenceSufficiency: out.EvidenceSufficiency,
		DifficultyRange: d.Range, UsableEvidence: usable, EvidenceIDs: append([]string(nil), out.EvidenceIDs...),
	}}

	seen := map[string]bool{}
	for _, q := range bank {
		if seen[q.ItemKey] {
			continue
		}
		seen[q.ItemKey] = true
		facts := items[q.ItemKey]
		item := summarizeItem(q.Target, facts)
		out.KnowledgeOutcomes = append(out.KnowledgeOutcomes, item)
		switch {
		case item.Status == Demonstrated:
			out.DemonstratedStrengths = append(out.DemonstratedStrengths, item)
		case item.Status == NeedsPractice:
			out.KnowledgeGaps = append(out.KnowledgeGaps, item)
			out.RecommendedLearningObjectives = append(out.RecommendedLearningObjectives, item)
		}
		if item.Status == NeedsMoreEvidence || item.GapSupport == "suspected" {
			out.RecommendedKnowledgeChecks = append(out.RecommendedKnowledgeChecks, item)
		}
	}
	checks := make(map[string]bool)
	for _, check := range out.RecommendedKnowledgeChecks {
		checks[check.ItemKey] = true
	}
	byKey := make(map[string]KnowledgeOutcome, len(out.KnowledgeOutcomes))
	for _, item := range out.KnowledgeOutcomes {
		byKey[item.ItemKey] = item
	}
	for _, gap := range out.KnowledgeGaps {
		for _, key := range Prerequisites(language, gap.ItemKey) {
			item, ok := byKey[key]
			if ok && item.Status != Demonstrated && !checks[key] {
				out.RecommendedKnowledgeChecks = append(out.RecommendedKnowledgeChecks, item)
				checks[key] = true
			}
		}
	}
	slices.SortStableFunc(out.RecommendedLearningObjectives, func(a, b KnowledgeOutcome) int {
		return cmp.Compare(a.GapSupport, b.GapSupport) // strong before suspected
	})
	return out
}

func summarizeItem(t Target, facts []Observation) KnowledgeOutcome {
	item := KnowledgeOutcome{ItemKey: t.ItemKey, Category: t.Category, Objective: t.Objective,
		Status: NotAssessed, EvidenceIDs: []string{}}
	correct, incorrect, partial := 0, 0, 0
	recall, curatedCorrect, curatedIncorrect := false, false, false
	for _, o := range facts {
		item.EvidenceIDs = append(item.EvidenceIDs, o.EvidenceID)
		if !o.Usable() {
			continue
		}
		switch o.Outcome {
		case Correct:
			correct++
			curatedCorrect = curatedCorrect || o.Origin == "curated"
			recall = recall || o.EvidenceMode() != "recognition"
		case Incorrect:
			incorrect++
			curatedIncorrect = curatedIncorrect || o.Origin == "curated"
		case PartialCredit:
			partial++
		}
	}
	switch {
	case len(facts) == 0:
	case correct >= 2 && recall && curatedCorrect && incorrect == 0 && partial == 0:
		item.Status = Demonstrated
	case incorrect >= 2 && curatedIncorrect && correct == 0 && partial == 0:
		item.Status, item.GapSupport = NeedsPractice, "strong"
	case incorrect >= 1 && correct == 0 && partial == 0:
		item.Status, item.GapSupport = NeedsPractice, "suspected"
	default:
		item.Status = NeedsMoreEvidence
	}
	return item
}
