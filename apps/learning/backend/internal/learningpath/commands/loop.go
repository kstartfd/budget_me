package commands

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/learningpath/curriculum"
	"learning_app/backend/internal/learningprofile"
)

// answerFacts is everything the learning loop needs to know about one stored
// attempt (docs/learning/VERTICAL_SLICE_V1.md).
type answerFacts struct {
	Path     db.LearningPath
	Topic    db.LearningPathTopic
	Exercise db.LearningPathExercise
	Attempt  db.LearningPathAttempt
	// Review: the answer was given to a due review, whose exercise carried no key.
	Review bool
	At     time.Time
}

// recordEvidence turns a stored attempt into evidence, updates what is known
// about the item and schedules its next review. It runs in the attempt's own
// transaction, so an attempt and what it proves are stored together or not at all.
//
// It records nothing, and is not an error, for an answer that cannot be tied to
// a piece of knowledge: a topic the agent wrote with no library entry, an
// exercise kind it does not know, or a teacher who has been removed.
func recordEvidence(ctx context.Context, tq *db.Queries, cfg learningprofile.Config, f answerFacts) error {
	if f.Topic.SourceID == "" {
		return nil
	}

	mode, base, ok := learningprofile.Kind(string(f.Exercise.Kind))
	if !ok {
		return nil
	}

	teacher, err := tq.GetLearningTeacherByLanguage(ctx, db.GetLearningTeacherByLanguageParams{UserID: f.Path.UserID, Language: f.Path.Language})
	if db.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	item, lib, ok, err := ensureItem(ctx, tq, teacher, f.Path.Language, f.Topic.SourceID, f.At)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	attemptNo, err := tq.CountLearningPathAttemptsOfExercise(ctx, db.CountLearningPathAttemptsOfExerciseParams{ExerciseID: f.Exercise.ID, UserID: f.Path.UserID})
	if err != nil {
		return err
	}

	attemptNo = max(attemptNo, 1)
	if f.Review {
		// A review is a fresh try after time has passed, not a retry of one just made.
		attemptNo = 1
	}

	weight := cfg.EvidenceWeight(base, int(attemptNo))
	source := learningprofile.SourceLearningPath

	if f.Review {
		source = learningprofile.SourceReview
	}

	score, outcome := 0.0, "incorrect"
	if f.Attempt.Correct {
		score, outcome = 1.0, "correct"
	}

	written, err := tq.InsertLearningEvidence(ctx, db.InsertLearningEvidenceParams{
		ID: ids.ToPg(ids.New()), UserID: f.Path.UserID, TeacherID: teacher.ID, ItemID: item.ID,
		Skill:  learningprofile.SkillOf(string(f.Exercise.Kind), lib.Skill),
		Source: string(source), PracticeMode: string(learningprofile.Tracked),
		RefType: "attempt", RefID: f.Attempt.ID,
		Mode: string(mode), Outcome: outcome, Score: score, Weight: weight,
		Assisted: !f.Review, AttemptNo: int16(min(attemptNo, 32000)), OccurredAt: db.Time(f.At),
	})
	if err != nil {
		return err
	}

	// The same attempt was recorded already: its effect is in the state.
	if written == 0 {
		return nil
	}

	return applyToState(ctx, tq, cfg, teacher.ID, f.Path.UserID, item.ID, learningprofile.Evidence{
		Mode: mode, Correct: f.Attempt.Correct, Weight: base, AttemptNo: int(attemptNo),
		Assisted: !f.Review, Review: f.Review, At: f.At,
	})
}

// ensureItem is the knowledge item of a library topic, made from the library the
// first time something is shown about it. ok is false when the library has no
// such topic (it was renamed, or the pair has no library): there is no item to
// attach anything to.
func ensureItem(ctx context.Context, tq *db.Queries, teacher db.LearningTeacher, language, sourceID string, at time.Time) (db.LearningKnowledgeItem, curriculum.Topic, bool, error) {
	lib, ok := curriculum.Find(language, ExplainLanguage(teacher), sourceID)
	if !ok {
		return db.LearningKnowledgeItem{}, curriculum.Topic{}, false, nil
	}

	// Most answers are about an item that exists: read it, and write only the first time.
	item, err := tq.GetKBItemByKey(ctx, db.GetKBItemByKeyParams{Language: language, Key: lib.ID})
	if err == nil {
		return item, lib, true, nil
	}

	if !db.IsNotFound(err) {
		return db.LearningKnowledgeItem{}, curriculum.Topic{}, false, err
	}

	kind := "topic"
	if lib.Skill == learningprofile.SkillGrammar {
		kind = "grammar"
	}

	item, err = tq.UpsertKBItem(ctx, db.UpsertKBItemParams{
		ID: ids.ToPg(ids.New()), Language: language, Key: lib.ID, Kind: kind,
		Skill: lib.Skill, CefrLevel: lib.Level, Unit: lib.Unit, CreatedAt: db.Time(at),
	})

	return item, lib, err == nil, err
}

// applyToState folds one answer into the item's row, which is made first and
// then locked so two answers at once cannot both start from nothing.
func applyToState(ctx context.Context, tq *db.Queries, cfg learningprofile.Config, teacher, user, item pgtype.UUID, ev learningprofile.Evidence) error {
	if err := tq.EnsureLearnerItemState(ctx, db.EnsureLearnerItemStateParams{TeacherID: teacher, UserID: user, ItemID: item, UpdatedAt: db.Time(ev.At)}); err != nil {
		return err
	}

	row, err := tq.LockLearnerItemState(ctx, db.LockLearnerItemStateParams{TeacherID: teacher, ItemID: item, UserID: user})
	if err != nil {
		return err
	}

	next := cfg.Apply(itemFromRow(row), ev)

	return tq.SaveLearnerItemState(ctx, db.SaveLearnerItemStateParams{
		TeacherID: teacher, ItemID: item, UserID: user,
		State: string(next.State), Score: next.Score(), WeightedSum: next.WeightedSum, WeightSum: next.WeightSum,
		EvidenceCount: int32(next.Count), Days: int32(next.Days), RecallSuccesses: int32(next.RecallSuccesses),
		Streak: int32(next.Streak), StreakDay: pgDate(next.StreakDay), Stage: int16(next.Stage), Lapses: int32(next.Lapses),
		LastEvidenceAt: pgTime(next.LastEvidenceAt), LastSuccessAt: pgTime(next.LastSuccessAt), DueAt: pgTime(next.DueAt),
		UpdatedAt: db.Time(ev.At),
	})
}

// ItemFromRow reads the stored state back into the rules' own type.
func itemFromRow(r db.LearnerItemState) learningprofile.Item {
	return learningprofile.Item{
		State: learningprofile.State(r.State), WeightedSum: r.WeightedSum, WeightSum: r.WeightSum,
		Count: int(r.EvidenceCount), Days: int(r.Days), RecallSuccesses: int(r.RecallSuccesses),
		Streak: int(r.Streak), StreakDay: dateOf(r.StreakDay), Stage: int(r.Stage), Lapses: int(r.Lapses),
		LastEvidenceAt: timeOf(r.LastEvidenceAt), LastSuccessAt: timeOf(r.LastSuccessAt), DueAt: timeOf(r.DueAt),
	}
}

// ItemOf is itemFromRow for the read side.
func ItemOf(r db.LearnerItemState) learningprofile.Item { return itemFromRow(r) }

func timeOf(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}

	return t.Time.UTC()
}

func dateOf(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}

	y, m, day := d.Time.Date()

	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}

func pgTime(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}

	return db.Time(t)
}

func pgDate(t time.Time) pgtype.Date {
	if t.IsZero() {
		return pgtype.Date{}
	}

	y, m, day := t.UTC().Date()

	return pgtype.Date{Time: time.Date(y, m, day, 0, 0, 0, 0, time.UTC), Valid: true}
}

// uuidOf is a pg uuid as a uuid, nil for NULL.
func uuidOf(u pgtype.UUID) uuid.UUID {
	if !u.Valid {
		return uuid.Nil
	}

	return ids.FromPg(u)
}
