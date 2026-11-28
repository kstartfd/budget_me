package commands

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningpath/assessment"
	"learning_app/backend/internal/learningpath/placement"
)

func assessmentText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }

func (h TestHandler) answerAssessment(ctx context.Context, test db.LearningPathTest, given []Answer) (TestProgress, error) {
	if len(given) == 0 || len(given) > 6 {
		return TestProgress{}, apperr.NewInvalid("a screening submission needs 1 to 6 answers")
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return TestProgress{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tq := h.q.WithTx(tx)
	test, err = lockAssessment(ctx, tq, test.UserID, test.ID)
	if err != nil {
		return TestProgress{}, err
	}
	engine, err := assessmentEngine(ctx, tq, test)
	if err != nil {
		return TestProgress{}, err
	}
	rows, err := tq.ListLearningPathTestQuestions(ctx, test.ID)
	if err != nil {
		return TestProgress{}, err
	}
	answers, err := tq.ListLearningPathTestAnswers(ctx, test.ID)
	if err != nil {
		return TestProgress{}, err
	}
	questions := make(map[string]db.LearningPathTestQuestion, len(rows))
	answered := make(map[string]db.LearningPathTestAnswer, len(answers))
	for _, r := range rows {
		questions[r.QuestionKey] = r
	}
	for _, a := range answers {
		answered[a.QuestionKey] = a
	}
	submitted := make(map[string]bool, len(given))
	for _, a := range given {
		a.Answer = strings.TrimSpace(a.Answer)
		if utf8.RuneCountInString(a.Answer) > maxTestAnswerRunes || (a.Skip && a.Answer != "") || submitted[a.Key] {
			return TestProgress{}, apperr.NewInvalid("invalid or repeated assessment answer")
		}
		submitted[a.Key] = true
		r, ok := questions[a.Key]
		if !ok {
			return TestProgress{}, apperr.NewInvalid("unknown assessment question")
		}
		if old, ok := answered[a.Key]; ok {
			if old.Answer != a.Answer || old.AssessmentAssisted != a.Assisted {
				return TestProgress{}, apperr.NewConflict("this question was already answered")
			}
			continue
		}
		if test.AssessmentStatus.String != assessment.InProgress || test.AssessmentBatch != r.AssessmentBatch.Int32 {
			return TestProgress{}, apperr.NewConflict("this question is not active")
		}

		stored, err := h.recordScreeningAnswer(ctx, tq, test, r, a, engine)
		if err != nil {
			return TestProgress{}, err
		}
		answered[a.Key] = stored
		answers = append(answers, stored)
	}

	if test.AssessmentStatus.String == assessment.InProgress && currentBatchAnswered(test, rows, answered) {
		batches := batchesOf(answers, rows)
		decision := engine.Decide(placement.Level(test.Level), batches)
		if decision.StopReason != "" {
			test, err = h.finishAssessment(ctx, tq, test, engine, batches, decision)
		} else {
			test, err = updateAssessment(ctx, tq, test, assessment.Evaluating, decision.Next, test.AssessmentBatch+1)
		}
		if err != nil {
			return TestProgress{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return TestProgress{}, err
	}
	if _, err := h.prepareAssessment(ctx, test); err != nil {
		return TestProgress{}, err
	}
	return h.Progress(ctx, ids.FromPg(test.UserID), ids.FromPg(test.ID))
}

func currentBatchAnswered(t db.LearningPathTest, rows []db.LearningPathTestQuestion, answered map[string]db.LearningPathTestAnswer) bool {
	for _, r := range rows {
		if r.AssessmentBatch.Int32 == t.AssessmentBatch {
			if _, ok := answered[r.QuestionKey]; !ok {
				return false
			}
		}
	}
	return true
}

func (h TestHandler) recordScreeningAnswer(ctx context.Context, q *db.Queries, t db.LearningPathTest, r db.LearningPathTestQuestion, a Answer, engine assessment.SectionEngine) (db.LearningPathTestAnswer, error) {
	public := questionOf(r)
	qn := assessment.Question{Target: targetOf(r), Content: assessment.Content{
		Text: public.Text, Prompt: public.Prompt, Options: public.Options,
	}, Origin: r.AssessmentOrigin.String}
	outcome := assessment.EvaluationUnavailable
	if json.Unmarshal(r.AssessmentAlternatives, &qn.GradingKey) == nil && assessment.Validate(qn.Target, qn.Content) == nil {
		evaluated, err := engine.Evaluate(qn, a.Answer, a.Skip)
		if err != nil {
			return db.LearningPathTestAnswer{}, apperr.NewInvalid(err.Error())
		}
		outcome = evaluated
	} else if a.Answer == "" {
		outcome = assessment.Skipped
	}

	now := db.Time(h.clock.Now())
	attempt := ids.ToPg(ids.New())
	row := db.LearningPathTestAnswer{
		TestID: t.ID, QuestionKey: a.Key, Answer: a.Answer, Correct: outcome == assessment.Correct,
		AssessmentAttemptID: attempt, AssessmentOutcome: assessmentText(string(outcome)), AssessmentAssisted: a.Assisted,
	}
	inserted, err := q.InsertAssessmentAnswer(ctx, db.InsertAssessmentAnswerParams(row))
	if err != nil {
		return row, err
	}
	if inserted != 1 {
		return row, apperr.NewConflict("this question was already answered")
	}
	fact := observationOf(r, row)
	score, weight := fact.ScoreWeight()
	_, err = q.InsertLearningEvidence(ctx, db.InsertLearningEvidenceParams{
		ID: attempt, RefID: attempt, RefType: "assessment_attempt", UserID: t.UserID, TeacherID: t.AssessmentProfileID,
		ItemID: r.AssessmentItemID, Skill: r.Skill, Source: "assessment", PracticeMode: "assessment",
		Mode: fact.EvidenceMode(), Outcome: string(outcome), Score: score, Weight: weight, Assisted: a.Assisted, AttemptNo: 1,
		OccurredAt: now,
	})
	return row, err
}

func (h TestHandler) assessmentProgress(ctx context.Context, test db.LearningPathTest) (TestProgress, error) {
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return TestProgress{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := h.q.WithTx(tx)
	test, err = q.GetLearningPathTest(ctx, db.GetLearningPathTestParams{ID: test.ID, UserID: test.UserID})
	if db.IsNotFound(err) {
		return TestProgress{}, apperr.NewNotFound("assessment not found")
	}
	if err != nil {
		return TestProgress{}, err
	}
	answers, err := q.ListLearningPathTestAnswers(ctx, test.ID)
	if err != nil {
		return TestProgress{}, err
	}
	rows, err := q.ListLearningPathTestQuestions(ctx, test.ID)
	if err != nil {
		return TestProgress{}, err
	}
	section, err := q.GetAssessmentSection(ctx, db.GetAssessmentSectionParams{TestID: test.ID, UserID: test.UserID, Skill: string(assessment.GrammarSkill)})
	if err != nil {
		return TestProgress{}, err
	}
	var policy assessment.Policy
	if err := json.Unmarshal(section.PolicySnapshot, &policy); err != nil {
		return TestProgress{}, err
	}
	sectionProgress := AssessmentSectionProgress{
		ID: ids.FromPg(section.ID).String(), Skill: section.Skill, Position: section.Position,
		Status: section.Status, CurrentBand: section.CurrentBand, Batch: section.CurrentBatch,
		Answered: section.Answered, MaxQuestions: policy.MaxQuestions, StopReason: section.StopReason.String,
	}
	if len(section.ResultSnapshot) > 0 && string(section.ResultSnapshot) != "null" {
		if err := json.Unmarshal(section.ResultSnapshot, &sectionProgress.Result); err != nil {
			return TestProgress{}, err
		}
	}
	a := &AssessmentProgress{
		Type: test.AssessmentType.String, Status: test.AssessmentStatus.String,
		LearnerProfileID: ids.FromPg(test.AssessmentProfileID).String(), Batch: test.AssessmentBatch,
		Answered: len(answers), MaxQuestions: policy.MaxQuestions, StopReason: test.AssessmentStopReason.String,
		Sections: []AssessmentSectionProgress{sectionProgress}, Questions: make(map[string]QuestionMetadata),
	}
	if len(test.AssessmentResult) > 0 {
		if err := json.Unmarshal(test.AssessmentResult, &a.Result); err != nil {
			return TestProgress{}, err
		}
	}
	p := TestProgress{Test: test, StartLevel: test.StartBand.String, Assessment: a, Asked: len(answers)}
	answered := make(map[string]bool, len(answers))
	for _, row := range answers {
		answered[row.QuestionKey] = true
	}
	for _, row := range rows {
		if row.AssessmentBatch.Int32 != test.AssessmentBatch {
			continue
		}
		p.Needed++
		if answered[row.QuestionKey] || a.Status != assessment.InProgress {
			continue
		}
		p.Questions = append(p.Questions, questionOf(row))
		a.Questions[row.QuestionKey] = QuestionMetadata{
			Type: assessment.Format(row.AssessmentFormat.String), ItemKey: row.AssessmentItemKey.String,
			Objective: row.AssessmentObjective.String, Category: row.AssessmentCategory.String, Version: row.AssessmentVersion.String,
		}
	}
	return p, tx.Commit(ctx)
}
