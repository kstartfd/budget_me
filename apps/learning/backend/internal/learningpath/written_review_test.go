package learningpath_test

import (
	"strings"
	"testing"
)

// writeExercise opens a topic and returns its exercise of the learner's own text.
func writeExercise(t *testing.T, e env) map[string]any {
	t.Helper()

	e.provider.setDefault(programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	topicID := path["topics"].([]any)[0].(map[string]any)["id"].(string)

	e.provider.enqueue(variedExercisesReply)

	return generateExercises(t, e, topicID)[2].(map[string]any)
}

// TestAWrittenTextIsReadLikeANativeSpeakerWould: the agent is told to count what
// a native speaker would find wrong, wording that is merely stiff included, and
// what the learner wrote and their level go with the request.
func TestAWrittenTextIsReadLikeANativeSpeakerWould(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	write := writeExercise(t, e)

	e.provider.enqueue(`{"correct":true,"mistakes":0,"feedback":""}`)
	attempt(t, e, write["id"].(string), "Yesterday while I was working my dogs were playing, during this time the phone rang", true, "")

	prompts := e.provider.requests()
	judge := prompts[len(prompts)-1]

	for _, want := range []string{
		"the way a native speaker would read it", "bookish", "even when it is grammatical", "Typing slips are not mistakes",
		"the answer is simply accepted", `\"level\":\"A1\"`, "my dogs were playing, during this time the phone rang",
	} {
		if !strings.Contains(judge, want) {
			t.Errorf("the agent was not told %q:\n%s", want, judge)
		}
	}
}

// TestACleanTextIsJustAccepted: no mistakes, no explanation, however much the
// agent says, and the count is stored as 0.
func TestACleanTextIsJustAccepted(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	write := writeExercise(t, e)

	e.provider.enqueue(`{"correct":true,"mistakes":0,"feedback":"Nice one!"}`)

	got := attempt(t, e, write["id"].(string), "I get up at seven. I work in an office.", true, "")
	if got["mistakes"].(float64) != 0 || got["feedback"] != "" {
		t.Fatalf("an accepted text carries no count and nothing to explain: %v", got)
	}
}

// TestAWrittenTextWithMistakesIsCountedAndExplained: the count and the
// explanation are kept with the attempt, and a text with a mistake is not
// correct even when the agent said it was.
func TestAWrittenTextWithMistakesIsCountedAndExplained(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	write := writeExercise(t, e)

	e.provider.enqueue(`{"correct":true,"mistakes":2,"feedback":"\"during this time\" sounds bookish: say \"and then\". Add a comma after \"working\"."}`)

	got := attempt(t, e, write["id"].(string), "Yesterday while I was working my dogs were playing, during this time the phone rang", false, "sounds bookish")
	if got["mistakes"].(float64) != 2 {
		t.Fatalf("the learner is told how many mistakes there are: %v", got)
	}

	// What the topic screen reads back later is the same.
	attempts := topicDetail(t, e, writeExerciseTopic(t, e, write))["attempts"].([]any)
	last := attempts[len(attempts)-1].(map[string]any)

	if last["mistakes"].(float64) != 2 || last["correct"] != false {
		t.Fatalf("the stored attempt lost its count: %v", last)
	}
}

func writeExerciseTopic(t *testing.T, e env, exercise map[string]any) string {
	t.Helper()

	var topicID string
	if err := e.pool.QueryRow(t.Context(), "SELECT topic_id::text FROM learning_path_exercises WHERE id = $1", exercise["id"]).Scan(&topicID); err != nil {
		t.Fatal(err)
	}

	return topicID
}

// TestAnUnexplainedRefusalIsAskedAgain: a count with nothing said about it, or a
// refusal with no reason, never reaches the learner; the agent is asked once more
// and told what was missing.
func TestAnUnexplainedRefusalIsAskedAgain(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	write := writeExercise(t, e)

	e.provider.enqueue(`{"correct":false,"mistakes":2,"feedback":""}`, `{"correct":false,"mistakes":1,"feedback":"Use \"went\", not \"goed\"."}`)

	got := attempt(t, e, write["id"].(string), "I goed home.", false, "went")
	if got["mistakes"].(float64) != 1 {
		t.Fatalf("the second answer is the one that counts: %v", got)
	}

	prompts := e.provider.requests()
	if retry := prompts[len(prompts)-1]; !strings.Contains(retry, "you counted 2 mistakes but gave no feedback") {
		t.Fatalf("the retry did not say what was missing:\n%s", retry)
	}

	e.provider.enqueue(`{"correct":false,"mistakes":0,"feedback":""}`, `{"correct":false,"mistakes":0,"feedback":"The task asks for two sentences about your day."}`)

	got = attempt(t, e, write["id"].(string), "Hello.", false, "two sentences")
	if got["mistakes"].(float64) != 0 {
		t.Fatalf("a text off the task has no language mistakes to count: %v", got)
	}
}

// TestOnlyAWrittenTextHasAMistakeCount: a translation is right or wrong as it
// was, whatever count the agent adds.
func TestOnlyAWrittenTextHasAMistakeCount(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	path := analyze(t, e, climb(t, e, "A1"))
	topicID := path["topics"].([]any)[0].(map[string]any)["id"].(string)

	e.provider.enqueue(exercisesReply)

	translate := generateExercises(t, e, topicID)[2].(map[string]any)

	e.provider.enqueue(`{"correct":false,"mistakes":3,"feedback":"Say \"I live in Moscow.\""}`)

	got := attempt(t, e, translate["id"].(string), "I living in Moscow.", false, "I live in Moscow")
	if got["mistakes"].(float64) != 0 {
		t.Fatalf("a translation is graded right or wrong: %v", got)
	}
}
