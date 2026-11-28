package commands

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningpath/assessment"
	"learning_app/backend/internal/learningpath/placement"
)

type QuestionMetadata struct {
	Type      assessment.Format `json:"type"`
	ItemKey   string            `json:"knowledgeItemKey"`
	Objective string            `json:"learningObjective"`
	Category  string            `json:"category"`
	Version   string            `json:"version"`
}

type AssessmentProgress struct {
	Type             string                      `json:"type"`
	Status           string                      `json:"status"`
	LearnerProfileID string                      `json:"learnerProfileId"`
	Batch            int32                       `json:"batch"`
	Answered         int                         `json:"answered"`
	MaxQuestions     int                         `json:"maxQuestions"`
	StopReason       string                      `json:"stopReason,omitempty"`
	Result           *assessment.Result          `json:"result,omitempty"`
	Sections         []AssessmentSectionProgress `json:"sections"`
	Questions        map[string]QuestionMetadata `json:"-"`
}

type AssessmentSectionProgress struct {
	ID           string                    `json:"id"`
	Skill        string                    `json:"skill"`
	Position     int32                     `json:"position"`
	Status       string                    `json:"status"`
	CurrentBand  string                    `json:"currentBand"`
	Batch        int32                     `json:"batch"`
	Answered     int32                     `json:"answered"`
	MaxQuestions int                       `json:"maxQuestions"`
	StopReason   string                    `json:"stopReason,omitempty"`
	Result       *assessment.SectionResult `json:"result,omitempty"`
}

func (h TestHandler) StartAssessment(ctx context.Context, cmd StartTest, kind, startPoint string) (db.LearningPathTest, error) {
	policy := h.policy
	mode, available := assessment.RuntimeMode(kind, policy)
	if !available {
		return db.LearningPathTest{}, apperr.NewInvalid("unsupported assessment type")
	}
	if !validLanguage(cmd.Language) || !validLanguage(cmd.TranslationLanguage) {
		return db.LearningPathTest{}, apperr.NewInvalid("invalid language code")
	}
	band := placement.Level(strings.ToUpper(strings.TrimSpace(startPoint)))
	if band == "" {
		band = placement.Level(policy.DefaultBand)
	}
	if !assessment.ValidBand(band) {
		return db.LearningPathTest{}, apperr.NewInvalid("screening starting point must be A1, A2, B1, or B2")
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return db.LearningPathTest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := h.q.WithTx(tx)
	user := ids.ToPg(cmd.UserID)
	now := db.Time(h.clock.Now())

	// The existing teacher upsert serializes competing starts for this profile.
	profile, err := tq.InsertLearningTeacher(ctx, db.InsertLearningTeacherParams{
		ID: ids.ToPg(ids.New()), UserID: user, Language: cmd.Language,
		TranslationLanguage: cmd.TranslationLanguage, CreatedAt: now,
	})
	if err != nil {
		return db.LearningPathTest{}, err
	}
	test, err := tq.GetOpenAssessment(ctx, db.GetOpenAssessmentParams{
		UserID: user, AssessmentProfileID: profile.ID, AssessmentType: assessmentText(kind),
	})
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return db.LearningPathTest{}, err
		}
		return h.prepareAssessment(ctx, test)
	}
	if !db.IsNotFound(err) {
		return db.LearningPathTest{}, err
	}
	if _, err := tq.GetActiveLearningPath(ctx, db.GetActiveLearningPathParams{UserID: user, Language: cmd.Language}); err == nil {
		return db.LearningPathTest{}, apperr.NewConflict("a program for this language is already running: finish it or reset it first")
	} else if !db.IsNotFound(err) {
		return db.LearningPathTest{}, err
	}
	explain := ExplainLanguage(profile)
	if _, ok := assessment.Bank(cmd.Language, explain); !ok {
		explain = cmd.Language
	}
	if _, ok := assessment.Bank(cmd.Language, explain); !ok {
		return db.LearningPathTest{}, apperr.NewInvalid("quick screening is unavailable for this language")
	}
	// Keep a full snapshot alongside the old policy fields. Old binaries ignore
	// the additional JSON property, so rollback retains this new information.
	encoded, err := json.Marshal(struct {
		assessment.Policy
		SectionBlueprint assessment.ModeBlueprint `json:"sectionBlueprint"`
	}{Policy: policy, SectionBlueprint: mode})
	if err != nil {
		return db.LearningPathTest{}, err
	}
	test, err = tq.InsertAssessment(ctx, db.InsertAssessmentParams{
		ID: ids.ToPg(ids.New()), UserID: user, Language: cmd.Language, TranslationLanguage: profile.TranslationLanguage,
		Level: string(band), CreatedAt: now, AssessmentProfileID: profile.ID,
		AssessmentType: assessmentText(kind), AssessmentPolicy: encoded, AssessmentExplanationLanguage: assessmentText(explain),
	})
	if err != nil {
		return db.LearningPathTest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return db.LearningPathTest{}, err
	}
	return h.prepareAssessment(ctx, test)
}

// AssessmentAction uses the same row lock as answer submission. An interrupted
// provider request can be resumed from the durable evaluating state.
func (h TestHandler) AssessmentAction(ctx context.Context, userID, testID uuid.UUID, action string) (TestProgress, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return TestProgress{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := h.q.WithTx(tx)
	test, err := lockAssessment(ctx, tq, ids.ToPg(userID), ids.ToPg(testID))
	if err != nil {
		return TestProgress{}, err
	}

	switch action {
	case "resume", "pause", "complete", "cancel":
	default:
		return TestProgress{}, apperr.NewInvalid("invalid assessment action")
	}
	if test.Status != db.LearningPathTestStatusDone {
		switch action {
		case "pause":
			test, err = updateAssessment(ctx, tq, test, assessment.Paused, placement.Level(test.Level), test.AssessmentBatch)
		case "resume":
			if test.AssessmentStatus.String == assessment.Paused {
				status := assessment.InProgress
				rows, err := tq.ListLearningPathTestQuestions(ctx, test.ID)
				if err != nil {
					return TestProgress{}, err
				}
				if !hasAssessmentBatch(rows, test.AssessmentBatch) {
					status = assessment.Evaluating
				}
				if _, err := updateAssessment(ctx, tq, test, status, placement.Level(test.Level), test.AssessmentBatch); err != nil {
					return TestProgress{}, err
				}
				test.AssessmentStatus = assessmentText(status)
			}
		case "complete", "cancel":
			engine, err := assessmentEngine(ctx, tq, test)
			if err != nil {
				return TestProgress{}, err
			}
			batches, err := assessmentBatches(ctx, tq, test.ID)
			if err != nil {
				return TestProgress{}, err
			}
			decision := engine.Decide(placement.Level(test.Level), batches)
			decision.StopReason = "learner_stopped"
			if action == "cancel" {
				decision.StopReason = "cancelled"
			}
			test, err = h.finishAssessment(ctx, tq, test, engine, batches, decision)
			if err != nil {
				return TestProgress{}, err
			}
		}
	}
	if err != nil {
		return TestProgress{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return TestProgress{}, err
	}
	if action == "resume" {
		if _, err := h.prepareAssessment(ctx, test); err != nil {
			return TestProgress{}, err
		}
	}
	return h.Progress(ctx, userID, testID)
}

func assessmentBlueprint(ctx context.Context, q *db.Queries, test db.LearningPathTest) (assessment.Policy, []assessment.Question, assessment.ModeBlueprint, error) {
	section, err := q.GetAssessmentSection(ctx, db.GetAssessmentSectionParams{TestID: test.ID, UserID: test.UserID, Skill: string(assessment.GrammarSkill)})
	if err != nil {
		return assessment.Policy{}, nil, assessment.ModeBlueprint{}, err
	}
	var policy assessment.Policy
	err = json.Unmarshal(section.PolicySnapshot, &policy)
	mode, available := assessment.RuntimeMode(test.AssessmentType.String, policy)
	bank, ok := assessment.Bank(test.Language, test.AssessmentExplanationLanguage.String)
	if err != nil || !available || policy.Version != mode.Version || !ok || policy.Validate() != nil {
		return policy, nil, mode, errors.New("invalid stored assessment blueprint")
	}
	if len(section.BlueprintSnapshot) > 0 && string(section.BlueprintSnapshot) != "null" {
		if json.Unmarshal(section.BlueprintSnapshot, &mode) != nil || mode.Validate() != nil ||
			mode.Type != test.AssessmentType.String || mode.Version != policy.Version || len(mode.Sections) != 1 ||
			mode.Sections[0].Skill != assessment.GrammarSkill || mode.Sections[0].Evaluator != assessment.KeyedEvaluator ||
			mode.Sections[0].MinQuestions != policy.MinQuestions || mode.Sections[0].MaxQuestions != policy.MaxQuestions {
			return policy, nil, mode, errors.New("invalid stored assessment section snapshot")
		}
		formats := make(map[assessment.Format]bool, len(mode.Sections[0].Formats))
		for _, format := range mode.Sections[0].Formats {
			formats[format] = true
		}
		filtered := make([]assessment.Question, 0, len(bank))
		for _, question := range bank {
			if formats[question.Format] {
				filtered = append(filtered, question)
			}
		}
		bank = filtered
	}
	return policy, bank, mode, nil
}

func assessmentEngine(ctx context.Context, q *db.Queries, test db.LearningPathTest) (assessment.SectionEngine, error) {
	policy, bank, mode, err := assessmentBlueprint(ctx, q, test)
	if err != nil {
		return assessment.SectionEngine{}, err
	}
	return assessment.NewSectionEngine(policy, mode, bank)
}

func lockAssessment(ctx context.Context, q *db.Queries, user, id pgtype.UUID) (db.LearningPathTest, error) {
	t, err := q.LockAssessment(ctx, db.LockAssessmentParams{ID: id, UserID: user})
	if db.IsNotFound(err) {
		return t, apperr.NewNotFound("assessment not found")
	}
	return t, err
}

func updateAssessment(ctx context.Context, q *db.Queries, t db.LearningPathTest, status string, band placement.Level, batch int32) (db.LearningPathTest, error) {
	if err := assessment.TransitionSection(t.AssessmentStatus.String, status); err != nil {
		return t, err
	}
	return q.UpdateAssessmentState(ctx, db.UpdateAssessmentStateParams{
		ID: t.ID, UserID: t.UserID, AssessmentStatus: assessmentText(status), Level: string(band), AssessmentBatch: batch,
	})
}

func hasAssessmentBatch(rows []db.LearningPathTestQuestion, batch int32) bool {
	for _, r := range rows {
		if r.AssessmentBatch.Valid && r.AssessmentBatch.Int32 == batch {
			return true
		}
	}
	return false
}

func (h TestHandler) finishAssessment(ctx context.Context, q *db.Queries, t db.LearningPathTest, engine assessment.SectionEngine, batches []assessment.Batch, d assessment.Decision) (db.LearningPathTest, error) {
	result, status, err := engine.Result(t.Language, batches, d)
	if err != nil {
		return t, err
	}
	result.AssessmentID = ids.FromPg(t.ID).String()
	result.LearnerProfileID = ids.FromPg(t.AssessmentProfileID).String()
	result.AssessmentType = t.AssessmentType.String
	if err := assessment.TransitionSection(t.AssessmentStatus.String, status); err != nil {
		return t, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return t, err
	}

	// The path table requires a band label. Use a conservative planning band;
	// the assessment result has a nullable range and its own evidence status.
	level := string(placement.A1)
	if d.Range != nil {
		level = string(d.Range.Lower)
	}
	return q.FinishAssessment(ctx, db.FinishAssessmentParams{
		ID: t.ID, UserID: t.UserID, CompletedAt: db.Time(h.clock.Now()), LevelReached: level,
		AssessmentStatus: assessmentText(status), AssessmentStopReason: assessmentText(d.StopReason), AssessmentResult: encoded,
	})
}

func assessmentBatches(ctx context.Context, q *db.Queries, id pgtype.UUID) ([]assessment.Batch, error) {
	answers, err := q.ListLearningPathTestAnswers(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListLearningPathTestQuestions(ctx, id)
	if err != nil {
		return nil, err
	}
	return batchesOf(answers, rows), nil
}

func batchesOf(answers []db.LearningPathTestAnswer, rows []db.LearningPathTestQuestion) []assessment.Batch {
	byKey := make(map[string]db.LearningPathTestAnswer, len(answers))
	for _, a := range answers {
		byKey[a.QuestionKey] = a
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.AssessmentBatch.Int32 != b.AssessmentBatch.Int32 {
			return a.AssessmentBatch.Int32 < b.AssessmentBatch.Int32
		}
		return a.Position < b.Position
	})
	issued := make(map[int32]int)
	for _, r := range rows {
		issued[r.AssessmentBatch.Int32]++
	}
	var batches []assessment.Batch
	var prev int32
	for _, r := range rows {
		a, ok := byKey[r.QuestionKey]
		if !ok || !r.AssessmentBatch.Valid {
			continue
		}
		batch := r.AssessmentBatch.Int32
		if batch != prev {
			batches = append(batches, assessment.Batch{Band: placement.Level(r.Level), Issued: issued[batch]})
			prev = batch
		}
		i := len(batches) - 1
		batches[i].Observations = append(batches[i].Observations, observationOf(r, a))
	}
	return batches
}

func targetOf(r db.LearningPathTestQuestion) assessment.Target {
	return assessment.Target{
		ItemKey: r.AssessmentItemKey.String, Category: r.AssessmentCategory.String, Objective: r.AssessmentObjective.String,
		Skill: placement.Skill(r.Skill), Band: placement.Level(r.Level), Format: assessment.Format(r.AssessmentFormat.String),
	}
}

func observationOf(r db.LearningPathTestQuestion, a db.LearningPathTestAnswer) assessment.Observation {
	return assessment.Observation{Target: targetOf(r), Outcome: assessment.Outcome(a.AssessmentOutcome.String),
		EvidenceID: ids.FromPg(a.AssessmentAttemptID).String(), Batch: int(r.AssessmentBatch.Int32),
		Assisted: a.AssessmentAssisted, Repeated: r.AssessmentRepeated, Origin: r.AssessmentOrigin.String}
}
