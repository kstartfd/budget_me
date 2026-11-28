package assessment

import (
	"slices"
	"testing"

	"learning_app/backend/internal/learningpath/curriculum"
	"learning_app/backend/internal/learningpath/placement"
)

func TestScreeningBankUsesCanonicalRules(t *testing.T) {
	for _, language := range []string{"en", "es"} {
		for _, explanation := range []string{"en", "es", "ru"} {
			if language == "en" && explanation == "es" {
				continue // no localized English-for-Spanish library
			}
			t.Run(language+"_"+explanation, func(t *testing.T) {
				questions, ok := Bank(language, explanation)
				if !ok || len(questions) != 48 {
					t.Fatalf("want 48 canonical items, got %d", len(questions))
				}
				seen := map[string]bool{}
				for _, q := range questions {
					item, found := curriculum.Find(language, explanation, q.ItemKey)
					if !found || item.Extra == nil || item.Goal != q.Objective || item.Level != string(q.Band) {
						t.Fatalf("invalid canonical reference: %s", q.ItemKey)
					}
					if seen[q.Fingerprint()] {
						t.Fatalf("duplicate prompt: %s", q.Prompt)
					}
					seen[q.Fingerprint()] = true
					for _, key := range Prerequisites(language, q.ItemKey) {
						if _, found := curriculum.Find(language, explanation, key); !found {
							t.Fatalf("invalid prerequisite %s", key)
						}
					}
				}
			})
		}
	}
}

func TestRuntimeModeMatchesIssuedQuestionBank(t *testing.T) {
	mode, ok := RuntimeMode(QuickScreening, QuickPolicy())
	if !ok || mode.Validate() != nil || mode.Version != QuickPolicy().Version || len(mode.Sections) != 1 {
		t.Fatalf("Quick Screening has no valid runtime blueprint: %+v", mode)
	}
	section := mode.Sections[0]
	if section.Skill != GrammarSkill || section.MinQuestions != QuickPolicy().MinQuestions ||
		section.MaxQuestions != QuickPolicy().MaxQuestions {
		t.Fatalf("runtime section does not match the measured grammar policy: %+v", section)
	}
	for _, language := range []string{"en", "es"} {
		questions, ok := Bank(language, language)
		if !ok {
			t.Fatalf("missing %s bank", language)
		}
		for _, question := range questions {
			if string(question.Skill) != string(section.Skill) || !slices.Contains(section.Formats, question.Format) {
				t.Fatalf("%s bank issued a question outside the runtime blueprint: %+v", language, question.Target)
			}
		}
	}
	for _, kind := range []string{FullAssessment, SkillAssessment, KnowledgeDiagnostic, PeriodicReassessment} {
		if _, ok := RuntimeMode(kind, QuickPolicy()); ok {
			t.Fatalf("unimplemented mode %s advertised as available", kind)
		}
	}
	custom := QuickPolicy()
	custom.MinQuestions = 18
	custom.MaxQuestions = 36
	configured, ok := RuntimeMode(QuickScreening, custom)
	if !ok || configured.Sections[0].MinQuestions != 18 || configured.Sections[0].MaxQuestions != 36 {
		t.Fatalf("runtime blueprint lost the configured policy snapshot: %+v", configured)
	}
}

func TestModeBlueprintRejectsSkillEvaluatorMismatch(t *testing.T) {
	mode, _ := RuntimeMode(QuickScreening, QuickPolicy())
	mode.Sections[0].Skill = PronunciationSkill
	if err := mode.Validate(); err == nil {
		t.Fatal("transcript/grammar formats must not activate pronunciation assessment")
	}
	mode, _ = RuntimeMode(QuickScreening, QuickPolicy())
	mode.Sections[0].Evaluator = AudioRubricEvaluator
	if err := mode.Validate(); err == nil {
		t.Fatal("a section cannot use an evaluator from a different skill")
	}
}

func TestCapabilitiesExposeOnlyBankedImplementedSkills(t *testing.T) {
	for _, language := range []string{"en", "es"} {
		available := AvailableModes(language, QuickPolicy())
		if len(available) != 1 || available[0].Type != QuickScreening ||
			available[0].BlueprintVersion != BlueprintVersion ||
			!slices.Equal(available[0].Skills, []string{"grammar"}) {
			t.Fatalf("%s advertised unsupported assessment coverage: %+v", language, available)
		}
	}
	if available := AvailableModes("fr", QuickPolicy()); len(available) != 0 {
		t.Fatalf("unsupported language advertised assessment: %+v", available)
	}
}

func TestModeAggregationKeepsSectionsIndependent(t *testing.T) {
	mode := ModeBlueprint{Type: FullAssessment, Version: "test-only", Sections: []SectionBlueprint{
		{Skill: GrammarSkill, Formats: []Format{MultipleChoice}, Evaluator: KeyedEvaluator, MinQuestions: 2, MaxQuestions: 5},
		{Skill: ReadingSkill, Formats: []Format{ReadingComprehension}, Evaluator: KeyedEvaluator, MinQuestions: 2, MaxQuestions: 5},
	}}
	if err := mode.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		sections []SectionState
		status   string
		next     AssessedSkill
	}{
		{"new", nil, Created, GrammarSkill},
		{"first created", []SectionState{{GrammarSkill, Created}}, Created, GrammarSkill},
		{"first finished, second pending", []SectionState{{GrammarSkill, Completed}}, InProgress, ReadingSkill},
		{"second created after first finished", []SectionState{{GrammarSkill, Completed}, {ReadingSkill, Created}}, InProgress, ReadingSkill},
		{"second paused", []SectionState{{GrammarSkill, Completed}, {ReadingSkill, Paused}}, Paused, ReadingSkill},
		{"both finished", []SectionState{{GrammarSkill, Completed}, {ReadingSkill, Completed}}, Completed, ""},
		{"partial coverage", []SectionState{{GrammarSkill, Completed}, {ReadingSkill, Partial}}, Partial, ""},
		{"all evaluators failed", []SectionState{{GrammarSkill, Failed}, {ReadingSkill, Failed}}, Failed, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := AggregateMode(mode, tc.sections)
			if err != nil || state.Status != tc.status || state.NextSkill != tc.next {
				t.Fatalf("aggregate %+v: %+v, %v", tc.sections, state, err)
			}
		})
	}
	for _, sections := range [][]SectionState{
		{{GrammarSkill, InProgress}, {ReadingSkill, Paused}},
		{{GrammarSkill, Completed}, {GrammarSkill, Completed}},
		{{ListeningSkill, Completed}},
		{{GrammarSkill, "certified"}},
	} {
		if _, err := AggregateMode(mode, sections); err == nil {
			t.Fatalf("invalid section combination accepted: %+v", sections)
		}
	}
}

func TestSectionLifecyclePreservesFinishedEvidence(t *testing.T) {
	for _, transition := range []struct {
		from, to string
		valid    bool
	}{
		{Created, InProgress, true},
		{Created, Paused, true},
		{InProgress, Evaluating, true},
		{Evaluating, InProgress, true},
		{Evaluating, Paused, true},
		{Paused, InProgress, true},
		{Paused, Evaluating, true},
		{InProgress, Completed, true},
		{Created, Failed, true},
		{Paused, Partial, true},
		{Paused, Paused, true},
		{Completed, InProgress, false},
		{Partial, Evaluating, false},
		{Failed, Paused, false},
		{Cancelled, Completed, false},
		{"", InProgress, false},
		{Created, "unknown", false},
	} {
		err := TransitionSection(transition.from, transition.to)
		if (err == nil) != transition.valid {
			t.Errorf("transition %q -> %q: valid=%t, error=%v", transition.from, transition.to, transition.valid, err)
		}
	}
}

func TestGateRejectsInvalidContent(t *testing.T) {
	choice := Content{Prompt: "Choose a form.", Options: []string{"a", "b", "c", "d"}, GradingKey: GradingKey{Answers: []string{"a"}}}
	open := Content{Prompt: "She ___ yesterday.", GradingKey: GradingKey{Answers: []string{"walked"}}}
	cases := []struct {
		name   string
		format Format
		body   Content
	}{
		{"duplicate options", MultipleChoice, Content{Prompt: choice.Prompt, Options: []string{"a", "A", "c", "d"}, GradingKey: choice.GradingKey}},
		{"missing key choice", MultipleChoice, Content{Prompt: choice.Prompt, Options: choice.Options, GradingKey: GradingKey{Answers: []string{"z"}}}},
		{"open choices", OpenGapFill, Content{Prompt: open.Prompt, Options: choice.Options, GradingKey: open.GradingKey}},
		{"two gaps", OpenGapFill, Content{Prompt: "She ___ ___ yesterday.", GradingKey: open.GradingKey}},
		{"unmarked error", ErrorCorrection, Content{Text: "She walk to work.", Prompt: "run", GradingKey: open.GradingKey}},
		{"no transformation source", SentenceTransformation, open},
		{"no passage", ReadingComprehension, choice},
		{"unsupported audio", Listening, choice},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate(Target{Format: tc.format}, tc.body); err == nil {
				t.Fatal("malformed question accepted")
			}
		})
	}
}

func TestGradeAlternativesPartialAndUnknown(t *testing.T) {
	bank, _ := Bank("es", "en")
	foundAlternative := false
	for _, q := range bank {
		if !slices.Contains(q.Answers, "hubiésemos reservado") {
			continue
		}
		foundAlternative = true
		for _, a := range []string{"hubiéramos reservado", "  HUBIE\u0301SEMOS  RESERVADO  "} {
			if got, err := Grade(q, a, false); err != nil || got != Correct {
				t.Fatalf("valid alternative %q got %v, %v", a, got, err)
			}
		}
		if got, _ := Grade(q, "", true); got != Skipped {
			t.Fatalf("unknown answer got %s", got)
		}
		if got, _ := Grade(q, "habíamos reservado", false); got != Incorrect {
			t.Fatalf("incorrect answer got %s", got)
		}
	}
	if !foundAlternative {
		t.Fatal("no curated alternative key")
	}
	en, _ := Bank("en", "en")
	found := false
	for _, q := range en {
		if len(q.PartialAnswers) > 0 {
			found = true
			if got, _ := Grade(q, q.PartialAnswers[0], false); got != PartialCredit {
				t.Fatalf("curated partial key got %s", got)
			}
		}
	}
	if !found {
		t.Fatal("no curated partial key")
	}
}

func sample(band placement.Level, correct, size int) Batch {
	batch := Batch{Band: band, Issued: size}
	for i := range size {
		outcome := Incorrect
		if i < correct {
			outcome = Correct
		}
		batch.Observations = append(batch.Observations, Observation{
			Target: Target{Category: []string{"verb_forms", "noun_phrase", "sentence_structure"}[i%3], Format: OpenGapFill},
			Origin: "curated", Outcome: outcome,
		})
	}
	return batch
}

func TestAdaptiveBoundariesAndWorkload(t *testing.T) {
	p := QuickPolicy()
	positive := sample(placement.A2, 5, 6)
	negative := sample(placement.B1, 2, 6)
	if d := Decide(p, placement.A2, []Batch{positive}); d.Next != placement.B1 || d.Enough || d.StopReason != "" {
		t.Fatalf("single good batch should move up without a result: %+v", d)
	}
	if d := Decide(p, placement.B1, []Batch{negative}); d.Next != placement.A2 || d.Enough {
		t.Fatalf("negative batch should move down: %+v", d)
	}
	if d := Decide(p, placement.B1, []Batch{positive, negative}); !d.Enough || d.Range.Lower != placement.A2 || d.Range.Upper != placement.B1 {
		t.Fatalf("adjacent usable batches should establish a boundary: %+v", d)
	}
	conflict := []Batch{positive, sample(placement.A2, 1, 6)}
	if d := Decide(p, placement.A2, conflict); !d.Conflict || d.Enough || d.Next != placement.A2 || d.Range != nil {
		t.Fatalf("conflict should hold the band: %+v", d)
	}
	for len(conflict)*6 < p.MaxQuestions {
		conflict = append(conflict, sample(placement.A2, 3, 6))
	}
	if d := Decide(p, placement.A2, conflict); !d.Conflict || d.Enough || d.StopReason != "max_questions" {
		t.Fatalf("conflict must stop at the cap: %+v", d)
	}
	for _, tc := range []struct {
		band     placement.Level
		right    int
		boundary string
	}{{placement.A1, 1, "floor"}, {placement.B2, 6, "ceiling"}} {
		one := sample(tc.band, tc.right, 6)
		d := Decide(p, tc.band, []Batch{one, one})
		if !d.Enough || d.Range.Boundary != tc.boundary {
			t.Fatalf("replicated edge: %+v", d)
		}
	}
}

func TestEvidenceQualityAndOtherBatchSizes(t *testing.T) {
	p := QuickPolicy()
	b := sample(placement.A2, 5, 6)
	b.Observations[0].Assisted = true
	b.Observations[1].Outcome = EvaluationUnavailable
	b.Observations[2].Repeated = true
	if d := Decide(p, placement.A2, []Batch{b}); d.Next != placement.A2 {
		t.Fatalf("low quality answers moved the band: %+v", d)
	}
	for _, b := range []Batch{sample(placement.A2, 4, 4), sample(placement.A2, 5, 5)} {
		if d := Decide(p, placement.A2, []Batch{b}); d.Next != placement.B1 {
			t.Fatalf("equivalent thresholds should move up: %+v", d)
		}
	}
	short := sample(placement.A2, 4, 4)
	short.Issued = 6
	if d := Decide(p, placement.A2, []Batch{short}); d.Next != placement.A2 {
		t.Fatalf("unfinished batch drove a band move: %+v", d)
	}
	if d := Decide(p, placement.A2, []Batch{sample(placement.A2, 3, 5)}); d.Next != placement.A2 {
		t.Fatalf("inconclusive evidence moved the band: %+v", d)
	}
}

func TestResultsKeepUntestedItemsAndSkillsUnknown(t *testing.T) {
	p := QuickPolicy()
	bank, _ := Bank("en", "en")
	probe := func(key string, outcome Outcome) Observation {
		for _, q := range bank {
			if q.ItemKey == key {
				target := q.Target
				target.Format = OpenGapFill
				return Observation{Target: target, Outcome: outcome, Origin: "curated", EvidenceID: key}
			}
		}
		t.Fatalf("missing target %s", key)
		return Observation{}
	}
	facts := []Observation{
		probe("a1-present-simple-questions", Incorrect),
		probe("a1-present-simple-questions", Incorrect),
		probe("b1-second-conditional", Correct),
		probe("b1-second-conditional", Correct),
		probe("a2-quantifiers", Incorrect),
		probe("a1-articles", Skipped),
	}
	result := Summarize(p, "en", bank, []Batch{{Band: placement.A1, Issued: 6, Observations: facts}}, Decision{StopReason: "learner_stopped"})
	if result.DifficultyRange != nil || result.EvidenceSufficiency != "preliminary" || len(result.UnassessedSkills) != 6 {
		t.Fatalf("unexpected preliminary result: %+v", result)
	}
	if len(result.Sections) != 1 || result.Sections[0].Skill != "grammar" ||
		result.Sections[0].Status != Partial || result.Sections[0].UsableEvidence != 5 ||
		len(result.Sections[0].EvidenceIDs) != len(facts) {
		t.Fatalf("grammar section lost its independent partial result: %+v", result.Sections)
	}
	statuses := map[string]string{}
	for _, item := range result.KnowledgeOutcomes {
		statuses[item.ItemKey] = item.Status
	}
	for key, want := range map[string]string{
		"a1-present-simple": NotAssessed, "a1-articles": NeedsMoreEvidence, "b1-second-conditional": Demonstrated,
	} {
		if statuses[key] != want {
			t.Fatalf("%s: got %s, want %s", key, statuses[key], want)
		}
	}
	if len(result.KnowledgeGaps) != 2 || result.RecommendedLearningObjectives[0].GapSupport != "strong" ||
		!slices.ContainsFunc(result.RecommendedKnowledgeChecks, func(k KnowledgeOutcome) bool { return k.ItemKey == "a1-present-simple" }) {
		t.Fatalf("gap strength or prerequisite recommendation was lost: %+v", result)
	}
}

func TestCompletionStatusKeepsFailureAndPartialDistinct(t *testing.T) {
	for _, tc := range []struct {
		decision Decision
		evidence int
		want     string
	}{
		{Decision{StopReason: "difficulty_boundary", Enough: true}, 12, Completed},
		{Decision{StopReason: "learner_stopped"}, 3, Partial},
		{Decision{StopReason: "question_generation_unavailable"}, 0, Failed},
		{Decision{StopReason: "question_generation_unavailable"}, 2, Partial},
		{Decision{StopReason: "cancelled", Enough: true}, 12, Cancelled},
	} {
		if got := CompletionStatus(tc.decision, tc.evidence); got != tc.want {
			t.Fatalf("%+v with %d evidence: %s, want %s", tc.decision, tc.evidence, got, tc.want)
		}
	}
}
