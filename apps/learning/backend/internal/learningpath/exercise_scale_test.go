package learningpath_test

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// activePath reads the learner's running program as the program screen does.
func activePath(t *testing.T, e env, language string) map[string]any {
	t.Helper()

	code, body := request(t, e.e, "GET", "/api/v1/learning/path?language="+language, e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("active path: %d %v", code, body)
	}

	return data(t, body)
}

// exerciseCounts is {total, done} of every topic of the running program, by id.
func exerciseCounts(t *testing.T, e env, language string) (order []string, counts map[string][2]int) {
	t.Helper()

	counts = map[string][2]int{}

	for _, item := range activePath(t, e, language)["topics"].([]any) {
		topic := item.(map[string]any)

		id := topic["id"].(string)
		order = append(order, id)

		got, ok := topic["exercises"].(map[string]any)
		if !ok {
			t.Fatalf("a topic of the program carries no exercise count: %v", topic)
		}

		counts[id] = [2]int{int(got["total"].(float64)), int(got["done"].(float64))}
	}

	return order, counts
}

// waitPrepared waits until at least n topics of the program have their set.
func waitPrepared(t *testing.T, e env, language string, n int) (order []string, counts map[string][2]int) {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)

	for {
		order, counts = exerciseCounts(t, e, language)

		ready := 0
		for _, c := range counts {
			if c[0] > 0 {
				ready++
			}
		}

		if ready >= n {
			return order, counts
		}

		if time.Now().After(deadline) {
			t.Fatalf("only %d of %d topics were prepared: %v", ready, n, counts)
		}

		time.Sleep(20 * time.Millisecond)
	}
}

// TestBuildPreparesEveryTopic: the whole program gets its exercises as soon as
// it is saved, each topic's count shows on the program, and preparing a set
// starts nobody's topic.
func TestBuildPreparesEveryTopic(t *testing.T) {
	e := setupPrefetch(t, "", 32)
	chooseAgent(t, e)

	testID := climb(t, e, "A1")
	e.provider.enqueue(programReply)
	e.provider.setDefault(exercisesReply)

	path := analyze(t, e, testID)
	language := path["language"].(string)

	order, counts := waitPrepared(t, e, language, 8)
	if len(order) != 8 {
		t.Fatalf("topics: %v", order)
	}

	for _, id := range order {
		if counts[id] != [2]int{3, 0} {
			t.Fatalf("a prepared topic holds its 3 exercises and none is answered: %v", counts)
		}
	}

	for _, item := range activePath(t, e, language)["topics"].([]any) {
		if status := item.(map[string]any)["status"]; status != "not_started" {
			t.Fatalf("preparing the sets must not start a topic: %v", item)
		}
	}

	// One call for the program, one for each of its eight topics.
	if calls := len(e.provider.requests()); calls != 9 {
		t.Fatalf("the agent was called %d times, want 9", calls)
	}
}

// articlesTitle is the title of the program's fourth topic, a1-articles, as the
// exercise writer is told it.
const articlesTitle = "Артикли a, an, the"

// TestAFailedSetDoesNotStopTheRest: a topic whose set cannot be written is left
// for when the learner opens it; the others are prepared, and the job stays done.
func TestAFailedSetDoesNotStopTheRest(t *testing.T) {
	e := setupPrefetch(t, "", 32)
	chooseAgent(t, e)

	testID := climb(t, e, "A1")
	e.provider.enqueue(programReply)

	var broken atomic.Bool
	broken.Store(true)

	e.provider.mu.Lock()
	e.provider.defaultText = exercisesReply
	e.provider.answer = func(prompt string) (string, bool) {
		// The planner's prompt lists every title of the catalogue, so only the
		// exercise writer's (it asks for how to apply the rule) is broken.
		if broken.Load() && strings.Contains(prompt, "how to apply it") && strings.Contains(prompt, articlesTitle) {
			return "I cannot write that.", true
		}

		return "", false
	}
	e.provider.mu.Unlock()

	path := analyze(t, e, testID)
	language := path["language"].(string)

	var articles string

	for _, item := range path["topics"].([]any) {
		if topic := item.(map[string]any); topic["title"] == articlesTitle {
			articles = topic["id"].(string)
		}
	}

	if articles == "" {
		t.Fatalf("the program has no %q topic: %v", articlesTitle, path["topics"])
	}

	_, counts := waitPrepared(t, e, language, 7)

	// The seven others are ready; give the broken one's retry time to finish.
	deadline := time.Now().Add(10 * time.Second)
	for {
		broke := 0
		for _, prompt := range e.provider.requests() {
			if strings.Contains(prompt, "how to apply it") && strings.Contains(prompt, articlesTitle) {
				broke++
			}
		}

		if broke >= 2 {
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("the broken topic was never tried")
		}

		time.Sleep(20 * time.Millisecond)
	}

	_, counts = exerciseCounts(t, e, language)
	if counts[articles] != [2]int{0, 0} {
		t.Fatalf("a set that could not be written must leave the topic empty: %v", counts[articles])
	}

	ready := 0
	for _, c := range counts {
		if c[0] > 0 {
			ready++
		}
	}

	if ready != 7 {
		t.Fatalf("the other topics must be prepared: %v", counts)
	}

	// The learner opens the topic later and the agent is well again.
	broken.Store(false)
	generateExercises(t, e, articles)

	if _, counts = exerciseCounts(t, e, language); counts[articles][0] != 3 {
		t.Fatalf("opening the topic must write its set: %v", counts[articles])
	}
}

// TestACountedAnswerFollowsTheCurrentSet: the topic's count grows with right
// answers only, and a regenerated set starts from nothing again.
func TestACountedAnswerFollowsTheCurrentSet(t *testing.T) {
	e := setupPrefetch(t, "", 32)
	chooseAgent(t, e)

	testID := climb(t, e, "A1")
	e.provider.enqueue(programReply)
	e.provider.setDefault(exercisesReply)

	path := analyze(t, e, testID)
	language := path["language"].(string)
	topicID := path["topics"].([]any)[0].(map[string]any)["id"].(string)

	waitPrepared(t, e, language, 8)

	exercises := generateExercises(t, e, topicID)
	choice := exercises[0].(map[string]any)["id"].(string)
	fill := exercises[1].(map[string]any)["id"].(string)

	// A wrong answer counts for nothing, a right one for one, and again for the same does not double.
	e.provider.setDefault(`{"correct":false,"feedback":"no"}`)
	attempt(t, e, choice, "go", false, "")

	if _, counts := exerciseCounts(t, e, language); counts[topicID] != [2]int{3, 0} {
		t.Fatalf("a wrong answer is not progress: %v", counts[topicID])
	}

	attempt(t, e, choice, "goes", true, "")
	attempt(t, e, choice, "goes", true, "")
	attempt(t, e, fill, "rode", true, "")

	if _, counts := exerciseCounts(t, e, language); counts[topicID] != [2]int{3, 2} {
		t.Fatalf("two of three exercises are answered right: %v", counts[topicID])
	}

	// The learner asks for a new set: the earlier answers belong to the old one.
	e.provider.setDefault(exercisesReply)

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/topics/"+topicID+"/exercises?regenerate=1", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("regenerate: %d %v", code, body)
	}

	if _, counts := exerciseCounts(t, e, language); counts[topicID] != [2]int{3, 0} {
		t.Fatalf("a new set starts from nothing: %v", counts[topicID])
	}

	// A topic nobody touched shows no progress.
	if got := activePath(t, e, language)["topics"].([]any)[1].(map[string]any)["exercises"].(map[string]any); got["done"].(float64) != 0 {
		t.Fatalf("an untouched topic shows progress: %v", got)
	}
}

// TestOpeningATopicWhileItsSetIsBeingWrittenKeepsOneSet: the agent takes its
// time, and a learner who opens the topic meanwhile must not end up with the
// background's set replacing the one they are answering.
func TestOpeningATopicWhileItsSetIsBeingWrittenKeepsOneSet(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)

	e.provider.setDefault(programReply)
	path := analyze(t, e, climb(t, e, "A1"))
	topicID := path["topics"].([]any)[0].(map[string]any)["id"].(string)

	e.provider.setDefault(exercisesReply)

	// Two openings at once, both of which find no set at the start.
	results := make(chan int, 2)

	for range 2 {
		go func() {
			code, _ := request(t, e.e, "POST", "/api/v1/learning/path/topics/"+topicID+"/exercises", e.session, nil)
			results <- code
		}()
	}

	for range 2 {
		if code := <-results; code != http.StatusOK {
			t.Fatalf("an opening failed: %d", code)
		}
	}

	var generations int
	if err := e.pool.QueryRow(t.Context(), "SELECT count(DISTINCT generation) FROM learning_path_exercise_versions WHERE topic_id = $1", topicID).Scan(&generations); err != nil {
		t.Fatal(err)
	}

	if generations != 1 {
		t.Fatalf("two openings wrote %d sets, want one", generations)
	}
}
