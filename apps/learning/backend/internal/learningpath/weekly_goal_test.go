package learningpath_test

import (
	"net/http"
	"strings"
	"testing"
)

const goalPath = "/api/v1/learning/path/goal"

// weeklyGoal reads the goal of the running English program, in minutes.
func weeklyGoal(t *testing.T, e env) float64 {
	t.Helper()

	code, body := request(t, e.e, "GET", "/api/v1/learning/path?language=en", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("path: %d %v", code, body)
	}

	return data(t, body)["weeklyGoalMinutes"].(float64)
}

func setGoal(t *testing.T, e env, hours int) (int, map[string]any) {
	t.Helper()

	return request(t, e.e, "PUT", goalPath, e.session, map[string]any{"language": "en", "hoursPerWeek": hours})
}

// TestTheWeeklyGoalIsAskedForWhenTheProgramIsBuiltAndCanBeChanged: the hours a
// week from the survey become the program's goal in minutes, the learner can
// change it or take it away, and the one the survey used to ask for (minutes a
// day times days a week) still counts on programs made before.
func TestTheWeeklyGoalIsAskedForWhenTheProgramIsBuiltAndCanBeChanged(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	testID, questions := startPlacement(t, e)
	state := answerBatch(t, e, testID, questions, true)
	answerBatch(t, e, testID, state["questions"].([]any), false)

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/tests/"+testID+"/analyze", e.session, map[string]any{"hoursPerWeek": 5})
	if code != http.StatusAccepted {
		t.Fatalf("analyze: %d %v", code, body)
	}

	path := waitJob(t, e, data(t, body)["id"].(string))["path"].(map[string]any)
	if path["weeklyGoalMinutes"] != float64(300) || path["preferences"].(map[string]any)["hoursPerWeek"] != float64(5) {
		t.Fatalf("the survey's hours a week must be the goal: %v", path)
	}

	asked := e.provider.requests()
	if survey := unescaped(asked[len(asked)-1]); !strings.Contains(survey, `"hoursPerWeek":5`) {
		t.Errorf("the planner was not told the hours a week:\n%s", survey)
	}

	if code, body := setGoal(t, e, 7); code != http.StatusOK || weeklyGoal(t, e) != 420 {
		t.Fatalf("change the goal: %d %v", code, body)
	}

	if code, _ := setGoal(t, e, 41); code != http.StatusBadRequest {
		t.Fatalf("a goal above 40 hours a week: %d", code)
	}

	if code, _ := setGoal(t, e, -1); code != http.StatusBadRequest {
		t.Fatalf("a negative goal: %d", code)
	}

	if weeklyGoal(t, e) != 420 {
		t.Fatal("a refused goal changed the stored one")
	}

	if code, body := setGoal(t, e, 0); code != http.StatusOK || weeklyGoal(t, e) != 0 {
		t.Fatalf("take the goal away: %d %v", code, body)
	}

	// A program made before: minutes a day times days a week is its goal, until
	// the learner sets one, which replaces both answers.
	if _, err := e.pool.Exec(t.Context(), `UPDATE learning_paths SET preferences = '{"minutesPerDay":30,"daysPerWeek":5,"goal":"work"}'::jsonb WHERE user_id = $1 AND status = 'active'`, e.user); err != nil {
		t.Fatal(err)
	}

	if weeklyGoal(t, e) != 150 {
		t.Fatalf("an old program's goal: %v", weeklyGoal(t, e))
	}

	if code, body := setGoal(t, e, 3); code != http.StatusOK || weeklyGoal(t, e) != 180 {
		t.Fatalf("set a goal on an old program: %d %v", code, body)
	}

	code, body = request(t, e.e, "GET", "/api/v1/learning/path?language=en", e.session, nil)
	prefs := data(t, body)["preferences"].(map[string]any)
	if _, has := prefs["minutesPerDay"]; has || prefs["goal"] != "work" || prefs["hoursPerWeek"] != float64(3) {
		t.Fatalf("the goal must replace the old answers and keep the rest: %v", prefs)
	}

	// No program for that language: nothing to set it on.
	if code, _ := request(t, e.e, "PUT", goalPath, e.session, map[string]any{"language": "de", "hoursPerWeek": 3}); code != http.StatusNotFound {
		t.Fatalf("no program: %d", code)
	}
}
