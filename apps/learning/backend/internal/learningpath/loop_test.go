package learningpath_test

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/testdb"
	"learning_app/backend/internal/token"
)

// The first learning loop, through the real handlers and a real database:
// an answer becomes evidence, the evidence moves the item's state, the state
// schedules a review, and the review is asked and answered.
// Design: docs/learning/VERTICAL_SLICE_V1.md.

const (
	loopPath   = "/api/v1/learning/path"
	loopLang   = "?language=en"
	oneDay     = 24 * time.Hour
	timeSlack  = 2 * time.Minute
	reviewWord = "rode"
)

type evidenceRow struct {
	Key, Skill, Source, Mode, Outcome string
	Weight                            float64
	Assisted                          bool
	AttemptNo                         int
}

func evidenceOf(t *testing.T, e env) []evidenceRow {
	t.Helper()

	rows, err := e.pool.Query(context.Background(), `
		SELECT i.key, ev.skill, ev.source, ev.mode, ev.outcome, ev.weight, ev.assisted, ev.attempt_no
		FROM learning_evidence ev JOIN kb_items i ON i.id = ev.item_id
		WHERE ev.user_id = $1 ORDER BY ev.occurred_at, ev.id`, e.user)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var out []evidenceRow

	for rows.Next() {
		var r evidenceRow
		if err := rows.Scan(&r.Key, &r.Skill, &r.Source, &r.Mode, &r.Outcome, &r.Weight, &r.Assisted, &r.AttemptNo); err != nil {
			t.Fatal(err)
		}

		out = append(out, r)
	}

	return out
}

type stateRow struct {
	State                      string
	Stage, Count, Days, Lapses int
	Due                        *time.Time
}

func stateOf(t *testing.T, e env) (stateRow, bool) {
	t.Helper()

	var s stateRow

	err := e.pool.QueryRow(context.Background(), `
		SELECT state, stage, evidence_count, days, lapses, due_at FROM learner_item_state
		WHERE user_id = $1 AND evidence_count > 0`, e.user).Scan(&s.State, &s.Stage, &s.Count, &s.Days, &s.Lapses, &s.Due)
	if err != nil {
		return stateRow{}, false
	}

	return s, true
}

func mustState(t *testing.T, e env) stateRow {
	t.Helper()

	s, ok := stateOf(t, e)
	if !ok {
		t.Fatal("the item has no state")
	}

	return s
}

func near(t *testing.T, what string, got *time.Time, want time.Time) {
	t.Helper()

	if got == nil {
		t.Fatalf("%s: no time, want about %v", what, want)
	}

	if d := got.Sub(want); d < -timeSlack || d > timeSlack {
		t.Fatalf("%s: %v, want about %v", what, *got, want)
	}
}

// loopProgram opens a program and the exercise set of its first topic: a choice
// ("goes"), a fill ("rode") and a translation judged by the agent.
func loopProgram(t *testing.T) (env, string, map[string]map[string]any) {
	t.Helper()

	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	topicID := path["topics"].([]any)[0].(map[string]any)["id"].(string)

	e.provider.enqueue(exercisesReply)

	byKind := map[string]map[string]any{}
	for _, x := range generateExercises(t, e, topicID) {
		x := x.(map[string]any)
		byKind[x["kind"].(string)] = x
	}

	return e, topicID, byKind
}

// answeredProgram answers the three exercises right and completes the topic, so
// the item has evidence and a review scheduled and the topic is long finished.
func answeredProgram(t *testing.T) (env, string, map[string]map[string]any) {
	t.Helper()

	e, topicID, ex := loopProgram(t)
	answerAndComplete(t, e, topicID, ex)

	return e, topicID, ex
}

// answerAndComplete answers the three exercises right and completes the topic.
func answerAndComplete(t *testing.T, e env, topicID string, ex map[string]map[string]any) {
	t.Helper()

	attempt(t, e, ex["choice"]["id"].(string), "goes", true, "")
	attempt(t, e, ex["fill"]["id"].(string), reviewWord, true, "")
	e.provider.enqueue(`{"correct":true,"feedback":""}`)
	attempt(t, e, ex["translate"]["id"].(string), "I live in Moscow.", true, "")

	code, body := request(t, e.e, "POST", loopPath+"/topics/"+topicID+"/complete", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("complete: %d %v", code, body)
	}
}

func listActivities(t *testing.T, e env) []map[string]any {
	t.Helper()

	code, body := request(t, e.e, "GET", loopPath+"/activities"+loopLang, e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("activities: %d %v", code, body)
	}

	return listPayload(t, body)
}

func dashboardOf(t *testing.T, e env) map[string]any {
	t.Helper()

	code, body := request(t, e.e, "GET", loopPath+"/dashboard"+loopLang, e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("dashboard: %d %v", code, body)
	}

	return data(t, body)
}

func number64(t *testing.T, m map[string]any, keys ...string) int {
	t.Helper()

	var cur any = m
	for _, k := range keys {
		cur = cur.(map[string]any)[k]
	}

	f, ok := cur.(float64)
	if !ok {
		t.Fatalf("%v is not a number: %v", keys, cur)
	}

	return int(f)
}

func reviewOf(list []map[string]any) map[string]any {
	for _, a := range list {
		if a["type"] == "review" {
			return a
		}
	}

	return nil
}

// due makes the item's review due now by moving its due time into the past.
func due(t *testing.T, e env) {
	t.Helper()

	if _, err := e.pool.Exec(context.Background(), `UPDATE learner_item_state SET due_at = now() - interval '1 hour' WHERE user_id = $1`, e.user); err != nil {
		t.Fatal(err)
	}
}

func reviewAnswer(t *testing.T, e env, exerciseID, activityID, answer string) (int, map[string]any) {
	t.Helper()

	return request(t, e.e, "POST", loopPath+"/exercises/"+exerciseID+"/attempt", e.session, map[string]any{"answer": answer, "activityId": activityID})
}

func TestAnAnswerFormsEvidenceMovesTheStateAndSchedulesAReview(t *testing.T) {
	e, topicID, ex := loopProgram(t)

	// A right pick: evidence of recognition, assisted (the key was on screen).
	result := attempt(t, e, ex["choice"]["id"].(string), "goes", true, "")

	// Nothing about the attempt's response changed: the screens read the same keys.
	var keys []string
	for k := range result {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	if got := strings.Join(keys, ","); got != "answer,correct,createdAt,exerciseId,feedback,id,mistakes" {
		t.Fatalf("the attempt response changed: %s", got)
	}

	rows := evidenceOf(t, e)
	if len(rows) != 1 {
		t.Fatalf("one answer, %d evidence rows", len(rows))
	}

	if r := rows[0]; r.Key == "" || r.Skill != "grammar" || r.Source != "learning_path" || r.Mode != "recognition" || r.Outcome != "correct" || r.Weight != 0.4 || !r.Assisted || r.AttemptNo != 1 {
		t.Fatalf("evidence: %+v", r)
	}

	state := mustState(t, e)
	if state.State != "learning" || state.Stage != 1 || state.Count != 1 || state.Days != 1 {
		t.Fatalf("state after the first answer: %+v", state)
	}

	near(t, "first review", state.Due, time.Now().Add(oneDay))

	// A wrong fill is evidence of a failed recall: the ladder starts again.
	e.provider.enqueue(`{"correct":false,"feedback":"'Ride' in the past is 'rode'."}`)
	attempt(t, e, ex["fill"]["id"].(string), "rided", false, "rode")

	rows = evidenceOf(t, e)
	if r := rows[1]; r.Mode != "recall" || r.Outcome != "incorrect" || r.Weight != 1.0 {
		t.Fatalf("a wrong fill: %+v", r)
	}

	if state = mustState(t, e); state.Stage != 0 || state.Count != 2 {
		t.Fatalf("state after a wrong answer: %+v", state)
	}

	// The retry counts half, and practising before the review is due moves nothing.
	attempt(t, e, ex["fill"]["id"].(string), reviewWord, true, "")

	rows = evidenceOf(t, e)
	if r := rows[2]; r.AttemptNo != 2 || r.Weight != 0.5 {
		t.Fatalf("a retry: %+v", r)
	}

	if state = mustState(t, e); state.Stage != 0 || state.State != "learning" {
		t.Fatalf("early practice moved the item: %+v", state)
	}

	// The minimal dashboard shows exactly that, and offers the topic being worked on.
	d := dashboardOf(t, e)
	if number64(t, d, "items", "tracked") != 1 || number64(t, d, "items", "learning") != 1 || number64(t, d, "items", "mastered") != 0 {
		t.Fatalf("dashboard items: %v", d["items"])
	}

	if number64(t, d, "evidence", "total") != 3 || number64(t, d, "reviews", "due") != 0 || number64(t, d, "reviews", "upcoming") != 1 {
		t.Fatalf("dashboard evidence and reviews: %v %v", d["evidence"], d["reviews"])
	}

	acts := d["activities"].([]any)
	if len(acts) != 1 {
		t.Fatalf("activities: %v", acts)
	}

	practice := acts[0].(map[string]any)
	if practice["type"] != "practice" || practice["status"] != "open" || practice["topic"].(map[string]any)["id"] != topicID || practice["item"] == nil {
		t.Fatalf("the practice activity: %v", practice)
	}
}

func TestSixRightAnswersInOneSittingAdvanceTheLadderOnce(t *testing.T) {
	e, _, ex := loopProgram(t)

	for range 6 {
		attempt(t, e, ex["fill"]["id"].(string), reviewWord, true, "")
	}

	state := mustState(t, e)
	if state.Stage != 1 || state.Count != 6 || state.Days != 1 || state.State != "learning" {
		t.Fatalf("six answers in a day: %+v", state)
	}

	near(t, "due", state.Due, time.Now().Add(oneDay))
}

func TestATopicWithoutALibraryEntryLeavesNoEvidence(t *testing.T) {
	e, topicID, ex := loopProgram(t)

	if _, err := e.pool.Exec(context.Background(), `UPDATE learning_path_topics SET source_id = '' WHERE id = $1`, topicID); err != nil {
		t.Fatal(err)
	}

	attempt(t, e, ex["choice"]["id"].(string), "goes", true, "")

	if n := len(evidenceOf(t, e)); n != 0 {
		t.Fatalf("%d evidence rows for a topic the library does not know", n)
	}

	if _, ok := stateOf(t, e); ok {
		t.Fatal("a state for a topic the library does not know")
	}

	// It is still something to practise: the activity just has no item.
	acts := listActivities(t, e)
	if len(acts) != 1 || acts[0]["type"] != "practice" || acts[0]["item"] != nil {
		t.Fatalf("activities: %v", acts)
	}
}

// The attempt and what it proves are one unit: when the evidence cannot be
// written, the attempt is not kept either.
func TestAnAttemptAndItsEvidenceAreStoredTogether(t *testing.T) {
	e, _, ex := loopProgram(t)
	ctx := context.Background()
	fn := "loop_test_fail_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")

	for _, stmt := range []string{
		fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'evidence refused by the test'; END $$`, fn),
		fmt.Sprintf(`CREATE TRIGGER %s BEFORE INSERT ON learning_evidence FOR EACH ROW WHEN (NEW.user_id = '%s') EXECUTE FUNCTION %s()`, fn, e.user, fn),
	} {
		if _, err := e.pool.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}

	t.Cleanup(func() {
		_, _ = e.pool.Exec(ctx, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON learning_evidence", fn))
		_, _ = e.pool.Exec(ctx, fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", fn))
	})

	code, _ := request(t, e.e, "POST", loopPath+"/exercises/"+ex["choice"]["id"].(string)+"/attempt", e.session, map[string]any{"answer": "goes"})
	if code < 500 {
		t.Fatalf("a failed evidence write answered %d", code)
	}

	var attempts int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM learning_path_attempts WHERE user_id = $1`, e.user).Scan(&attempts); err != nil || attempts != 0 {
		t.Fatalf("the attempt stayed without its evidence: %d, %v", attempts, err)
	}

	if _, ok := stateOf(t, e); ok {
		t.Fatal("a state without evidence")
	}
}

func TestPlanningIsSafeToRepeat(t *testing.T) {
	e, _, _ := answeredProgram(t)
	due(t, e)

	first := listActivities(t, e)
	second := listActivities(t, e)

	if len(first) != len(second) {
		t.Fatalf("the plan changed between two calls: %d vs %d", len(first), len(second))
	}

	for i := range first {
		if first[i]["id"] != second[i]["id"] {
			t.Fatalf("activity %d got a new id", i)
		}
	}

	var open int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM learning_activities WHERE user_id = $1 AND status = 'open'`, e.user).Scan(&open); err != nil || open != len(first) {
		t.Fatalf("open activities in the database: %d, listed %d, %v", open, len(first), err)
	}
}

func TestADueItemIsReviewedOnAFinishedTopicAndRescheduled(t *testing.T) {
	e, _, _ := answeredProgram(t)

	// Nothing is due yet: no review is offered.
	if acts := listActivities(t, e); reviewOf(acts) != nil {
		t.Fatalf("a review before it is due: %v", acts)
	}

	due(t, e)

	review := reviewOf(listActivities(t, e))
	if review == nil {
		t.Fatal("a due item was not offered for review")
	}

	if review["reason"] != "due" || review["status"] != "open" || review["item"] == nil || review["state"] != "learning" {
		t.Fatalf("the review activity: %v", review)
	}

	// The exercise comes without its key and without the rule that explains it.
	exercise := review["exercise"].(map[string]any)
	for _, leak := range []string{"answer", "rule"} {
		if _, ok := exercise[leak]; ok {
			t.Fatalf("the review exercise carries %q", leak)
		}
	}

	if exercise["kind"] != "fill" {
		t.Fatalf("a due review asks for a produced answer, the one answered longest ago: %v", exercise["kind"])
	}

	d := dashboardOf(t, e)
	if number64(t, d, "reviews", "due") != 1 {
		t.Fatalf("dashboard reviews: %v", d["reviews"])
	}

	// Answering it works although the topic is completed, and the answer is independent evidence.
	code, body := reviewAnswer(t, e, exercise["id"].(string), review["id"].(string), reviewWord)
	if code != http.StatusOK || data(t, body)["correct"] != true {
		t.Fatalf("review answer: %d %v", code, body)
	}

	rows := evidenceOf(t, e)
	if last := rows[len(rows)-1]; last.Source != "review" || last.Assisted || last.Mode != "recall" || last.Outcome != "correct" || last.AttemptNo != 1 || last.Weight != 1.0 {
		t.Fatalf("review evidence: %+v", last)
	}

	state := mustState(t, e)
	if state.Stage != 2 {
		t.Fatalf("a right answer to a due review moves one rung: %+v", state)
	}

	near(t, "next review", state.Due, time.Now().Add(3*oneDay))

	if reviewOf(listActivities(t, e)) != nil {
		t.Fatal("a done review is offered again")
	}

	// The same review cannot be answered twice.
	code, _ = reviewAnswer(t, e, exercise["id"].(string), review["id"].(string), reviewWord)
	if code != http.StatusConflict {
		t.Fatalf("a second answer to a done review: %d", code)
	}

	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM learning_path_attempts WHERE user_id = $1`, e.user).Scan(&n); err != nil || n != 4 {
		t.Fatalf("attempts: %d, %v (three of the program and one review)", n, err)
	}
}

func TestAReviewOfAnItemThatIsNotDueIsRefused(t *testing.T) {
	e, _, _ := answeredProgram(t)
	due(t, e)

	review := reviewOf(listActivities(t, e))
	exercise := review["exercise"].(map[string]any)

	// The item is answered another way and moves into the future before the review is answered.
	if _, err := e.pool.Exec(context.Background(), `UPDATE learner_item_state SET due_at = now() + interval '2 days' WHERE user_id = $1`, e.user); err != nil {
		t.Fatal(err)
	}

	code, _ := reviewAnswer(t, e, exercise["id"].(string), review["id"].(string), reviewWord)
	if code != http.StatusConflict {
		t.Fatalf("a review that is not due: %d", code)
	}

	// Looking again drops the stale review instead of offering it.
	if reviewOf(listActivities(t, e)) != nil {
		t.Fatal("a stale review is still offered")
	}

	var status string
	if err := e.pool.QueryRow(context.Background(), `SELECT status FROM learning_activities WHERE id = $1`, review["id"]).Scan(&status); err != nil || status != "skipped" {
		t.Fatalf("the stale review: %q, %v", status, err)
	}
}

// age sets the item up as one that was learnt a week ago: practiced, with enough
// evidence on enough days, due for a review now.
func age(t *testing.T, e env, state string, stage int) {
	t.Helper()

	_, err := e.pool.Exec(context.Background(), `
		UPDATE learner_item_state SET state = $2, stage = $3, score = 1, weighted_sum = 5, weight_sum = 5,
			evidence_count = 5, days = 3, recall_successes = 3, streak = 3, streak_day = (now() - interval '8 days')::date,
			last_evidence_at = now() - interval '8 days', last_success_at = now() - interval '8 days',
			due_at = now() - interval '1 hour'
		WHERE user_id = $1`, e.user, state, stage)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOnlyAnIndependentReviewMakesAnItemMastered(t *testing.T) {
	e, _, _ := answeredProgram(t)
	age(t, e, "practiced", 2)

	review := reviewOf(listActivities(t, e))
	exercise := review["exercise"].(map[string]any)

	code, body := reviewAnswer(t, e, exercise["id"].(string), review["id"].(string), reviewWord)
	if code != http.StatusOK {
		t.Fatalf("review: %d %v", code, body)
	}

	state := mustState(t, e)
	if state.State != "mastered" || state.Stage != 3 {
		t.Fatalf("after an independent review a week after the last success: %+v", state)
	}

	near(t, "next review", state.Due, time.Now().Add(7*oneDay))

	d := dashboardOf(t, e)
	if number64(t, d, "items", "mastered") != 1 || number64(t, d, "items", "learning") != 0 {
		t.Fatalf("dashboard items: %v", d["items"])
	}
}

func TestAnAssistedAnswerNeverMastersAnItem(t *testing.T) {
	e, topicID, _ := answeredProgram(t)
	age(t, e, "practiced", 2)

	// Back to the program: its exercises show their key, so this is assisted.
	if _, err := e.pool.Exec(context.Background(), `UPDATE learning_path_topics SET status = 'in_progress', completed_at = NULL WHERE id = $1`, topicID); err != nil {
		t.Fatal(err)
	}

	var exerciseID string
	if err := e.pool.QueryRow(context.Background(), `
		SELECT e.id FROM learning_path_exercises e WHERE e.topic_id = $1 AND e.kind = 'fill'`, topicID).Scan(&exerciseID); err != nil {
		t.Fatal(err)
	}

	attempt(t, e, exerciseID, reviewWord, true, "")

	if state := mustState(t, e); state.State != "practiced" {
		t.Fatalf("an assisted answer made the item %s", state.State)
	}
}

func TestAWrongReviewIsALapse(t *testing.T) {
	e, _, _ := answeredProgram(t)
	age(t, e, "mastered", 3)

	review := reviewOf(listActivities(t, e))
	exercise := review["exercise"].(map[string]any)

	e.provider.enqueue(`{"correct":false,"feedback":"Past of ride is rode."}`)

	code, body := reviewAnswer(t, e, exercise["id"].(string), review["id"].(string), "rided")
	if code != http.StatusOK || data(t, body)["correct"] != false {
		t.Fatalf("a wrong review: %d %v", code, body)
	}

	state := mustState(t, e)
	if state.State != "relearning" || state.Lapses != 1 || state.Stage != 0 {
		t.Fatalf("after a lapse: %+v", state)
	}

	near(t, "back tomorrow", state.Due, time.Now().Add(oneDay))

	d := dashboardOf(t, e)
	if number64(t, d, "items", "lapsed") != 1 || number64(t, d, "items", "learning") != 1 || number64(t, d, "items", "mastered") != 0 {
		t.Fatalf("dashboard after a lapse: %v", d["items"])
	}
}

func TestAReviewWithNothingToAskIsNotOffered(t *testing.T) {
	e, topicID, _ := answeredProgram(t)
	due(t, e)

	// The history was cleared: the item is due but nothing is left to ask.
	if _, err := e.pool.Exec(context.Background(), `DELETE FROM learning_path_exercises WHERE topic_id = $1`, topicID); err != nil {
		t.Fatal(err)
	}

	if reviewOf(listActivities(t, e)) != nil {
		t.Fatal("a review without an exercise was offered")
	}

	if d := dashboardOf(t, e); number64(t, d, "reviews", "due") != 1 {
		t.Fatalf("the due count is a count of the schedule, not of what can be asked: %v", d["reviews"])
	}
}

func TestPracticeIsClosedWithItsTopicAndMovesOn(t *testing.T) {
	e, topicID, ex := loopProgram(t)

	var first string
	for _, a := range listActivities(t, e) {
		if a["type"] == "practice" {
			first = a["id"].(string)
		}
	}

	if first == "" {
		t.Fatal("no practice for the topic being worked on")
	}

	answerAndComplete(t, e, topicID, ex)

	var status string

	var doneAt *time.Time

	var open []string
	for _, a := range listActivities(t, e) {
		if a["type"] == "practice" {
			open = append(open, a["topic"].(map[string]any)["id"].(string))
		}
	}

	if len(open) != 1 || open[0] == topicID {
		t.Fatalf("after finishing a topic the practice is the next one: %v (finished %s)", open, topicID)
	}

	if err := e.pool.QueryRow(context.Background(), `SELECT status, completed_at FROM learning_activities WHERE id = $1`, first).Scan(&status, &doneAt); err != nil || status != "completed" || doneAt == nil {
		t.Fatalf("the finished topic's practice: %q %v, %v", status, doneAt, err)
	}
}

func TestFreePracticeOfTheRuleBookLeavesNoEvidence(t *testing.T) {
	e := setup(t)
	withTeacher(t, e)
	e.provider.enqueue(blocksReply())

	if code, body := request(t, e.e, "POST", practicePath("/exercises"), e.session, nil); code != http.StatusOK {
		t.Fatalf("exercises: %d %v", code, body)
	}

	code, body := request(t, e.e, "POST", practicePath("/check"), e.session, map[string]any{"block": "b1", "item": "b1i1", "answer": "seen"})
	if code != http.StatusOK || data(t, body)["correct"] != true {
		t.Fatalf("check: %d %v", code, body)
	}

	if n := len(evidenceOf(t, e)); n != 0 {
		t.Fatalf("free practice wrote %d evidence rows", n)
	}

	var items int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM learner_item_state WHERE user_id = $1`, e.user).Scan(&items); err != nil || items != 0 {
		t.Fatalf("free practice made %d item states, %v", items, err)
	}
}

func TestTheLoopIsPrivateToTheLearner(t *testing.T) {
	e, _, ex := answeredProgram(t)
	due(t, e)

	review := reviewOf(listActivities(t, e))
	exercise := review["exercise"].(map[string]any)

	// A second learner, with a teacher of the same language and a review of their own.
	other := testdb.NewUser(t, e.pool, e.q, time.Now().UTC(), 1)
	otherSession := token.NewCodec(authSecret).Issue(other, time.Now().UTC(), time.Hour)
	teacher := ids.New()
	ctx := context.Background()

	if _, err := e.pool.Exec(ctx, `INSERT INTO learning_teachers (id, user_id, language, translation_language, created_at) VALUES ($1, $2, 'en', 'ru', now())`, teacher, other); err != nil {
		t.Fatal(err)
	}

	foreign := ids.New()
	if _, err := e.pool.Exec(ctx, `INSERT INTO learning_activities (id, user_id, teacher_id, type, status, reason, created_at) VALUES ($1, $2, $3, 'practice', 'open', 'path', now())`, foreign, other, teacher); err != nil {
		t.Fatal(err)
	}

	// Their activity id used on the first learner's exercise: not found.
	code, _ := reviewAnswer(t, e, exercise["id"].(string), foreign.String(), reviewWord)
	if code != http.StatusNotFound {
		t.Fatalf("another learner's activity: %d", code)
	}

	// The first learner's activity id used by the second learner: not found.
	code, _ = request(t, e.e, "POST", loopPath+"/exercises/"+exercise["id"].(string)+"/attempt", otherSession, map[string]any{"answer": reviewWord, "activityId": review["id"]})
	if code != http.StatusNotFound {
		t.Fatalf("another learner's exercise and activity: %d", code)
	}

	// The second learner sees their own activity and none of the first one's loop.
	code, body := request(t, e.e, "GET", loopPath+"/activities"+loopLang, otherSession, nil)
	mine := listPayload(t, body)
	if code != http.StatusOK || len(mine) != 1 || mine[0]["id"] != foreign.String() || mine[0]["type"] != "practice" {
		t.Fatalf("another learner's activities: %d %v", code, body)
	}

	code, body = request(t, e.e, "GET", loopPath+"/dashboard"+loopLang, otherSession, nil)
	if code != http.StatusOK || number64(t, data(t, body), "items", "tracked") != 0 || number64(t, data(t, body), "evidence", "total") != 0 {
		t.Fatalf("another learner's dashboard: %d %v", code, body)
	}

	// The first learner's exercise ids are theirs alone.
	_ = ex

	// A topic-scoped extension token (what the browser extension holds) is not a session.
	code, body = request(t, e.e, "POST", "/api/v1/learning/topics", e.session, map[string]any{"name": "English", "language": "en", "translationLanguage": "ru"})
	if code != http.StatusCreated {
		t.Fatalf("notebook topic: %d %v", code, body)
	}

	code, body = request(t, e.e, "POST", "/api/v1/learning/tokens", e.session, map[string]any{"name": "Chrome", "scope": "read_write", "topicId": data(t, body)["id"]})
	if code != http.StatusCreated {
		t.Fatalf("extension token: %d %v", code, body)
	}

	extension := data(t, body)["token"].(string)

	// Bad requests and no session.
	for _, tc := range []struct {
		name, method, path, session string
		want                        int
	}{
		{"no language", "GET", loopPath + "/activities", e.session, http.StatusBadRequest},
		{"no language, dashboard", "GET", loopPath + "/dashboard", e.session, http.StatusBadRequest},
		{"no teacher of that language", "GET", loopPath + "/activities?language=es", e.session, http.StatusNotFound},
		{"no session", "GET", loopPath + "/activities" + loopLang, "", http.StatusUnauthorized},
		{"no session, dashboard", "GET", loopPath + "/dashboard" + loopLang, "", http.StatusUnauthorized},
		{"extension token, activities", "GET", loopPath + "/activities" + loopLang, extension, http.StatusUnauthorized},
		{"extension token, dashboard", "GET", loopPath + "/dashboard" + loopLang, extension, http.StatusUnauthorized},
	} {
		if code, _ := request(t, e.e, tc.method, tc.path, tc.session, nil); code != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, code, tc.want)
		}
	}

	code, _ = request(t, e.e, "POST", loopPath+"/exercises/"+exercise["id"].(string)+"/attempt", extension, map[string]any{"answer": reviewWord, "activityId": review["id"]})
	if code != http.StatusUnauthorized {
		t.Fatalf("a review answered with an extension token: %d", code)
	}

	code, _ = request(t, e.e, "POST", loopPath+"/exercises/"+exercise["id"].(string)+"/attempt", e.session, map[string]any{"answer": reviewWord, "activityId": "not-a-uuid"})
	if code != http.StatusBadRequest {
		t.Fatalf("a malformed activity id: %d", code)
	}
}

func TestNoMoreThanFiveReviewsAreOpenAtOnce(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	topics := path["topics"].([]any)

	// Seven different items with something to show. The agent's rate limit allows only a few
	// sets in a burst; the others get theirs straight from the database.
	for i, topic := range topics[:7] {
		topicID := topic.(map[string]any)["id"].(string)

		var choice string

		if i < 2 {
			e.provider.enqueue(exercisesReply)

			for _, x := range generateExercises(t, e, topicID) {
				if x := x.(map[string]any); x["kind"] == "choice" {
					choice = x["id"].(string)
				}
			}
		} else {
			id, topic := ids.New(), uuid.MustParse(topicID)
			if _, err := e.q.InsertLearningPathExercise(context.Background(), db.InsertLearningPathExerciseParams{
				ID: ids.ToPg(id), TopicID: ids.ToPg(topic), Kind: db.LearningPathExerciseKindChoice, Prompt: "She ___ to school.",
				Options: []byte(`["go","goes"]`), Answer: "goes", CreatedAt: db.Time(time.Now()),
			}); err != nil {
				t.Fatal(err)
			}

			if err := e.q.InsertLearningPathExerciseVersion(context.Background(), db.InsertLearningPathExerciseVersionParams{ExerciseID: ids.ToPg(id), TopicID: ids.ToPg(topic), Generation: 1}); err != nil {
				t.Fatal(err)
			}

			choice = id.String()
		}

		attempt(t, e, choice, "goes", true, "")
	}

	var items int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM learner_item_state WHERE user_id = $1 AND evidence_count > 0`, e.user).Scan(&items); err != nil || items != 7 {
		t.Fatalf("items with evidence: %d, %v", items, err)
	}

	due(t, e)

	for range 3 {
		reviews := 0
		for _, a := range listActivities(t, e) {
			if a["type"] == "review" {
				reviews++
			}
		}

		if reviews != 5 {
			t.Fatalf("%d reviews offered of 7 due, want 5", reviews)
		}
	}

	if d := dashboardOf(t, e); number64(t, d, "reviews", "due") != 7 {
		t.Fatalf("the due count is the whole backlog: %v", d["reviews"])
	}
}
