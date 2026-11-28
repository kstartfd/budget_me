package learningpath_test

import (
	"net/http"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"learning_app/backend/internal/config"
	"learning_app/backend/internal/learningpath/assessment"
	"learning_app/backend/internal/server"
	"learning_app/backend/internal/testdb"
	"learning_app/backend/internal/token"
)

const testsURL = "/api/v1/learning/path/tests"

func TestAssessmentCapabilitiesAreScopedAndAccurate(t *testing.T) {
	e := setup(t)
	url := "/api/v1/learning/path/assessments/capabilities?language="
	for _, language := range []string{"en", "es"} {
		code, body := request(t, e.e, "GET", url+language, e.session, nil)
		if code != http.StatusOK {
			t.Fatalf("%s capabilities: %d %v", language, code, body)
		}
		response := data(t, body)
		available := response["available"].([]any)
		if response["language"] != language || len(available) != 1 {
			t.Fatalf("%s capabilities overstate coverage: %v", language, response)
		}
		quick := available[0].(map[string]any)
		if quick["type"] != "quick_screening" || quick["blueprintVersion"] != "grammar-quick-v1" ||
			!reflect.DeepEqual(quick["skills"], []any{"grammar"}) {
			t.Fatalf("%s has an unsupported advertised mode or skill: %v", language, quick)
		}
	}
	code, body := request(t, e.e, "GET", url+"fr", e.session, nil)
	if code != http.StatusOK || len(data(t, body)["available"].([]any)) != 0 {
		t.Fatalf("unsupported language capabilities: %d %v", code, body)
	}
	code, _ = request(t, e.e, "GET", url+"en", "", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("capabilities should require a session: %d", code)
	}
}

func screening(t *testing.T, e env, band string) (string, map[string]any) {
	t.Helper()
	code, body := request(t, e.e, "POST", testsURL, e.session, map[string]any{
		"language": "en", "translationLanguage": "ru", "purpose": "quick_screening", "startPoint": band,
	})
	if code != http.StatusCreated {
		t.Fatalf("start screening: %d %v", code, body)
	}
	state := data(t, body)
	return testsURL + "/" + state["id"].(string), state
}

func screeningAction(t *testing.T, e env, url, action string) map[string]any {
	t.Helper()
	code, body := request(t, e.e, "POST", url+"/assessment/"+action, e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("%s: %d %v", action, code, body)
	}
	return data(t, body)
}

func screeningAnswers(t *testing.T, e env, state map[string]any, correct int, skip bool) map[string]any {
	t.Helper()
	answers := []map[string]any{}
	for i, raw := range state["questions"].([]any) {
		q := raw.(map[string]any)
		if q["answer"] != nil || q["answers"] != nil || q["partialAnswers"] != nil || q["version"] == nil || q["knowledgeItemKey"] == nil {
			t.Fatalf("invalid public assessment question: %v", q)
		}
		key := q["key"].(string)
		if skip {
			answers = append(answers, map[string]any{"key": key, "skip": true})
			continue
		}
		var answer string
		if err := e.pool.QueryRow(t.Context(), `SELECT answer FROM learning_path_test_questions
			WHERE test_id = $1 AND question_key = $2`, state["id"], key).Scan(&answer); err != nil {
			t.Fatal(err)
		}
		if i >= correct {
			pick := q["options"].([]any)
			right := answer
			answer = "incorrect response"
			for _, option := range pick {
				if option != right {
					answer = option.(string)
					break
				}
			}
		}
		answers = append(answers, map[string]any{"key": key, "answer": answer})
	}
	return map[string]any{"answers": answers}
}

func submitScreening(t *testing.T, e env, url string, answers map[string]any) map[string]any {
	t.Helper()
	code, body := request(t, e.e, "POST", url+"/answers", e.session, answers)
	if code != http.StatusOK {
		t.Fatalf("submit screening: %d %v", code, body)
	}
	return data(t, body)
}

func progress(t *testing.T, state map[string]any) map[string]any {
	t.Helper()
	p, ok := state["assessment"].(map[string]any)
	if !ok {
		t.Fatalf("missing screening progress: %v", state)
	}
	return p
}

func TestQuickScreeningAdaptsAndBuildsFromEvidence(t *testing.T) {
	e := setup(t)
	url, start := screening(t, e, "")
	if start["level"] != "A2" || len(start["questions"].([]any)) != 6 {
		t.Fatalf("default screening should start at A2: %v", start)
	}
	up := submitScreening(t, e, url, screeningAnswers(t, e, start, 6, false))
	if up["level"] != "B1" || progress(t, up)["status"] != "in_progress" {
		t.Fatalf("the first batch should move up without completing: %v", up)
	}
	done := submitScreening(t, e, url, screeningAnswers(t, e, up, 0, false))
	if progress(t, done)["status"] != "completed" {
		t.Fatalf("difficulty boundary should complete: %v", done)
	}
	code, body := request(t, e.e, "GET", url+"/results", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("results: %d %v", code, body)
	}
	result := data(t, body)
	if result["assessmentId"] != start["id"] || result["learnerProfileId"] != progress(t, start)["learnerProfileId"] ||
		result["evidenceSufficiency"] != "sufficient_for_screening" || len(result["evidenceIds"].([]any)) != 12 ||
		!slices.Contains(result["unassessedSkills"].([]any), any("speaking")) {
		t.Fatalf("incomplete or overstated result: %v", result)
	}
	sections := result["sections"].([]any)
	if len(sections) != 1 || sections[0].(map[string]any)["skill"] != "grammar" ||
		sections[0].(map[string]any)["status"] != "completed" ||
		sections[0].(map[string]any)["usableEvidence"] != float64(12) {
		t.Fatalf("quick screening lost its section result: %v", sections)
	}
	bracket := result["difficultyRange"].(map[string]any)
	if bracket["lower"] != "A2" || bracket["upper"] != "B1" {
		t.Fatalf("wrong range: %v", bracket)
	}
	path := analyze(t, e, start["id"].(string))
	if path["source"] != "assessment" {
		t.Fatalf("legacy plan used: %v", path)
	}
	var gaps, unassessed int
	for _, gap := range result["knowledgeGaps"].([]any) {
		item := gap.(map[string]any)["knowledgeItemKey"]
		var n int
		if err := e.pool.QueryRow(t.Context(), `SELECT count(*) FROM learning_path_topics WHERE path_id=$1 AND source_id=$2`, path["id"], item).Scan(&n); err != nil {
			t.Fatal(err)
		}
		gaps += n
	}
	if err := e.pool.QueryRow(t.Context(), `SELECT count(*) FROM learning_path_topics WHERE path_id=$1 AND skill<>'grammar'`, path["id"]).Scan(&unassessed); err != nil {
		t.Fatal(err)
	}
	if gaps == 0 || unassessed < 2 || len(e.provider.requests()) != 0 || len(e.provider.testCalls()) != 0 {
		t.Fatalf("screening path omitted recommendations or called the legacy agent: %d gaps, %d unknown-skill topics", gaps, unassessed)
	}

	code, body = request(t, e.e, "POST", "/api/v1/learning/path/reset", e.session, map[string]any{"language": "en"})
	if code != http.StatusOK {
		t.Fatalf("archive screening plan: %d %v", code, body)
	}
	code, body = request(t, e.e, "POST", "/api/v1/learning/path/levels", e.session, map[string]any{
		"language": "en", "translationLanguage": "ru", "level": "A2",
	})
	if code != http.StatusBadRequest {
		t.Fatalf("a screening plan is not confirmation of a reached level: %d %v", code, body)
	}

}

func TestQuickScreeningResumeAndDuplicateEvidence(t *testing.T) {
	e := setup(t)
	url, start := screening(t, e, "B1")
	paused := screeningAction(t, e, url, "pause")
	if progress(t, paused)["status"] != "paused" || len(paused["questions"].([]any)) != 0 {
		t.Fatalf("paused state exposed active tasks: %v", paused)
	}
	againURL, _ := screening(t, e, "A1")
	resumed := screeningAction(t, e, url, "resume")
	if againURL != url || !reflect.DeepEqual(resumed["questions"], start["questions"]) {
		t.Fatal("resume issued different questions")
	}

	answers := screeningAnswers(t, e, start, 0, false)
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	for range 2 {
		wg.Go(func() {
			code, _ := request(t, e.e, "POST", url+"/answers", e.session, answers)
			codes <- code
		})
	}
	wg.Wait()
	close(codes)
	for code := range codes {
		if code != http.StatusOK {
			t.Fatalf("identical concurrent submission returned %d", code)
		}
	}
	code, body := request(t, e.e, "GET", url, e.session, nil)
	state := data(t, body)
	if code != http.StatusOK || state["level"] != "A2" || progress(t, state)["answered"] != float64(6) {
		t.Fatalf("weak batch should move down exactly once: %v", state)
	}
	got := screeningAction(t, e, url, "complete")
	if progress(t, got)["status"] != "partial" {
		t.Fatalf("early completion must be partial: %v", got)
	}

	var evidence, states int
	if err := e.pool.QueryRow(t.Context(), `SELECT count(*)
		FROM learning_evidence ev
		JOIN learning_path_test_answers a ON a.assessment_attempt_id = ev.ref_id
		JOIN learning_path_test_questions q USING (test_id, question_key)
		JOIN learning_path_tests test ON test.id = a.test_id
		WHERE a.test_id = $1 AND ev.source = 'assessment' AND ev.practice_mode = 'assessment'
		    AND ev.id = ev.ref_id AND q.assessment_item_id = ev.item_id AND ev.teacher_id = test.assessment_profile_id
		    AND q.assessment_version IS NOT NULL AND q.assessment_objective IS NOT NULL`, start["id"]).Scan(&evidence); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(t.Context(), `SELECT count(*) FROM learner_item_state WHERE user_id = $1`, e.user).Scan(&states); err != nil {
		t.Fatal(err)
	}
	if evidence != 6 || states != 0 {
		t.Fatalf("evidence should be written once without mastery effects: %d evidence, %d state changes", evidence, states)
	}
	var linked, sectionAnswered int
	var sectionStatus, sectionSkill string
	if err := e.pool.QueryRow(t.Context(), `SELECT s.status,s.skill,s.answered,
		(SELECT count(*) FROM learning_assessment_section_evidence e WHERE e.section_id=s.id)
		FROM learning_assessment_sections s WHERE s.test_id=$1`, start["id"]).Scan(&sectionStatus, &sectionSkill, &sectionAnswered, &linked); err != nil {
		t.Fatal(err)
	}
	if sectionStatus != "partial" || sectionSkill != "grammar" || sectionAnswered != 6 || linked != 6 {
		t.Fatalf("section does not match accepted answers and evidence: %s %s %d %d", sectionStatus, sectionSkill, sectionAnswered, linked)
	}

	now := time.Now().UTC()
	other := token.NewCodec(authSecret).Issue(testdb.NewUser(t, e.pool, e.q, now, 2), now, time.Hour)
	for _, endpoint := range []struct{ method, suffix string }{
		{"GET", ""}, {"GET", "/results"}, {"POST", "/answers"}, {"POST", "/assessment/resume"}, {"POST", "/analyze"},
	} {
		if code, _ := request(t, e.e, endpoint.method, url+endpoint.suffix, other, answers); code != http.StatusNotFound {
			t.Fatalf("foreign %s %s returned %d", endpoint.method, endpoint.suffix, code)
		}
	}
}

func TestAssessmentSectionResumesWithStoredPolicy(t *testing.T) {
	e := setup(t)
	url, start := screening(t, e, "A2")
	sections := progress(t, start)["sections"].([]any)
	section := sections[0].(map[string]any)
	if len(sections) != 1 || section["id"] != start["id"] || section["skill"] != "grammar" || section["answered"] != float64(0) {
		t.Fatalf("missing persisted section: %v", sections)
	}
	var hasSnapshot bool
	if err := e.pool.QueryRow(t.Context(), `SELECT blueprint_snapshot IS NOT NULL FROM learning_assessment_sections WHERE test_id=$1`, start["id"]).Scan(&hasSnapshot); err != nil || !hasSnapshot {
		t.Fatalf("new attempt did not save its section blueprint: %v %v", hasSnapshot, err)
	}
	screeningAction(t, e, url, "pause")
	policy := assessment.QuickPolicy()
	policy.BatchSize, policy.MinQuestions, policy.MaxQuestions = 3, 6, 12
	// Restart with different defaults, as a new release or configuration would.
	newServer := server.New(config.Config{AuthSecret: authSecret, AssessmentPolicy: policy}, e.pool, testdb.AccountPool(t))
	code, body := request(t, newServer, "POST", url+"/assessment/resume", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("resume: %d %v", code, body)
	}
	resumed := data(t, body)
	if len(resumed["questions"].([]any)) != 6 || progress(t, resumed)["maxQuestions"] != float64(30) {
		t.Fatalf("resume replaced the saved policy: %v", resumed)
	}
	e.e = newServer
	next := submitScreening(t, e, url, screeningAnswers(t, e, resumed, 6, false))
	if len(next["questions"].([]any)) != 6 {
		t.Fatalf("next batch ignored stored policy: %v", next)
	}
	finished := screeningAction(t, e, url, "complete")
	section = progress(t, finished)["sections"].([]any)[0].(map[string]any)
	if section["status"] != "partial" || section["stopReason"] != "learner_stopped" || section["answered"] != float64(6) || section["result"] == nil {
		t.Fatalf("section result not persisted: %v", section)
	}
}

func TestAssessmentSectionLegacyPolicyWithoutBlueprint(t *testing.T) {
	e := setup(t)
	url, start := screening(t, e, "A2")
	// A pre-P3-15 writer saved policy only. It must remain resumable without
	// claiming that its historical full blueprint was recorded.
	if _, err := e.pool.Exec(t.Context(), `UPDATE learning_path_tests SET assessment_policy=assessment_policy - 'sectionBlueprint' WHERE id=$1`, start["id"]); err != nil {
		t.Fatal(err)
	}
	screeningAction(t, e, url, "pause")
	resumed := screeningAction(t, e, url, "resume")
	if !reflect.DeepEqual(resumed["questions"], start["questions"]) {
		t.Fatal("legacy resume lost its questions")
	}
	var absent bool
	if err := e.pool.QueryRow(t.Context(), `SELECT blueprint_snapshot IS NULL FROM learning_assessment_sections WHERE test_id=$1`, start["id"]).Scan(&absent); err != nil || !absent {
		t.Fatalf("legacy blueprint was fabricated: %v %v", absent, err)
	}
	finished := screeningAction(t, e, url, "complete")
	if progress(t, finished)["status"] != "partial" {
		t.Fatalf("legacy partial result: %v", finished)
	}
}

func TestQuickScreeningPreservesSkipsWhenProviderFails(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.mu.Lock()
	e.provider.diagnosticStatus = http.StatusServiceUnavailable
	e.provider.mu.Unlock()

	url, start := screening(t, e, "A1")
	next := submitScreening(t, e, url, screeningAnswers(t, e, start, 0, true))
	done := submitScreening(t, e, url, screeningAnswers(t, e, next, 0, true))
	p := progress(t, done)
	if p["status"] != "partial" || p["stopReason"] != "question_generation_unavailable" || p["answered"] != float64(12) {
		t.Fatalf("provider failure lost answers or fabricated placement: %v", p)
	}
	result := p["result"].(map[string]any)
	if result["difficultyRange"] != nil || result["evidenceSufficiency"] != "insufficient" ||
		!slices.Contains(result["unassessedSkills"].([]any), any("grammar")) {
		t.Fatalf("skips should leave grammar unknown: %v", result)
	}
	var skips int
	if err := e.pool.QueryRow(t.Context(), `SELECT count(*) FROM learning_evidence WHERE user_id = $1 AND outcome='skipped' AND weight=0`, e.user).Scan(&skips); err != nil {
		t.Fatal(err)
	}
	if skips != 12 {
		t.Fatalf("saved %d skips, want 12", skips)
	}
}

func TestQuickScreeningAcceptsIndividualAnswersThroughTheCeiling(t *testing.T) {
	e := setup(t)
	url, state := screening(t, e, "A2")
	for batch := 0; batch < 4; batch++ {
		if progress(t, state)["status"] != "in_progress" {
			t.Fatalf("screening stopped before %d: %v", batch, state)
		}
		submission := screeningAnswers(t, e, state, 6, false)
		for _, answer := range submission["answers"].([]map[string]any) {
			state = submitScreening(t, e, url, map[string]any{"answers": []map[string]any{answer}})
		}
	}
	if p := progress(t, state); p["status"] != "completed" || p["answered"] != float64(24) {
		t.Fatalf("screening should reach the confirmed ceiling: %v", state)
	}
}

func TestQuickScreeningPreservesAnAnswerWhenEvaluationIsUnavailable(t *testing.T) {
	e := setup(t)
	url, start := screening(t, e, "A2")
	first := start["questions"].([]any)[0].(map[string]any)["key"]
	if _, err := e.pool.Exec(t.Context(), `UPDATE learning_path_test_questions SET assessment_alternatives = '{}'
		WHERE test_id = $1 AND question_key = $2`, start["id"], first); err != nil {
		t.Fatal(err)
	}
	given := "walked"
	state := submitScreening(t, e, url, map[string]any{"answers": []map[string]any{
		{"key": first, "answer": given, "assisted": true},
	}})
	if progress(t, state)["answered"] != float64(1) {
		t.Fatalf("attempt was lost: %v", state)
	}
	var answer, outcome string
	var weight float64
	var assisted bool
	if err := e.pool.QueryRow(t.Context(), `SELECT a.answer, ev.outcome, ev.weight, ev.assisted FROM learning_evidence ev
		JOIN learning_path_test_answers a ON a.assessment_attempt_id = ev.ref_id
		WHERE a.test_id = $1 AND a.question_key = $2`, start["id"], first).Scan(&answer, &outcome, &weight, &assisted); err != nil {
		t.Fatal(err)
	}
	if answer != given || outcome != "evaluation_unavailable" || weight != 0 || !assisted {
		t.Fatalf("unavailable attempt incorrectly represented: %q %q %v %v", answer, outcome, weight, assisted)
	}
}
