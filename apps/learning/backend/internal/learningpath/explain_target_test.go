package learningpath_test

import (
	"net/http"
	"strings"
	"testing"
)

const teachersPath = "/api/v1/learning/path/teachers"

// teacherID makes the English teacher with Russian rules and returns its id.
func teacherID(t *testing.T, e env) string {
	t.Helper()

	code, body := request(t, e.e, "POST", teachersPath, e.session, map[string]any{"language": "en", "translationLanguage": "ru"})
	if code >= 300 {
		t.Fatalf("teacher: %d %v", code, body)
	}

	d := data(t, body)
	if d["explainInTarget"] != false || d["explainLanguage"] != "ru" {
		t.Fatalf("a new teacher explains in the learner's own language: %v", d)
	}

	return d["id"].(string)
}

// explainIn switches how a teacher explains and returns its new state.
func explainIn(t *testing.T, e env, id string, target bool) map[string]any {
	t.Helper()

	code, body := request(t, e.e, "PATCH", teachersPath+"/"+id, e.session, map[string]any{"explainInTarget": target})
	if code != http.StatusOK {
		t.Fatalf("switch: %d %v", code, body)
	}

	return data(t, body)
}

func ruleOfPair(t *testing.T, e env) map[string]any {
	t.Helper()

	code, body := request(t, e.e, "GET", rulePath+"?language=en", e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("rule: %d %v", code, body)
	}

	return data(t, body)
}

// unescaped is a request the stub remembered, with the quotes of the JSON the
// agent was given turned back, so a test can look for `"key":"value"`.
func unescaped(request string) string { return strings.ReplaceAll(request, `\"`, `"`) }

// TestARuleIsExplainedInTheLanguageTheTeacherIsSetTo: the same rule is served in
// Russian until the learner asks for English, then in English with its example
// sentences still translated into Russian, and back.
func TestARuleIsExplainedInTheLanguageTheTeacherIsSetTo(t *testing.T) {
	e := setup(t)
	id := teacherID(t, e)

	rule := ruleOfPair(t, e)
	if !strings.Contains(rule["title"].(string), "что было до прошлого") {
		t.Fatalf("by default the rule is in the learner's own language: %v", rule["title"])
	}

	state := explainIn(t, e, id, true)
	if state["explainInTarget"] != true || state["explainLanguage"] != "en" || state["translationLanguage"] != "ru" {
		t.Fatalf("the learner's own language is kept while English is the language of explanations: %v", state)
	}

	rule = ruleOfPair(t, e)
	if rule["title"] != "Past Perfect: what happened before the past" || !strings.Contains(rule["theory"].(string), "Past Perfect (had + V3) shows") {
		t.Fatalf("the rule must be in English: %v", rule)
	}

	for _, ex := range rule["examples"].([]any) {
		if ex.(map[string]any)["translation"] == "" {
			t.Fatalf("an example sentence lost its translation into the learner's language: %v", ex)
		}
	}

	if related := rule["related"].([]any); len(related) == 0 || strings.ContainsAny(related[0].(map[string]any)["title"].(string), "абвгдеёжзийклмнопрстуфхцчшщъыьэюя") {
		t.Fatalf("the rules it refers to are in English too: %v", related)
	}

	if code, body := request(t, e.e, "GET", rulePath[:strings.LastIndex(rulePath, "/")]+"?language=en", e.session, nil); code != http.StatusOK || len(listPayload(t, body)) < 40 {
		t.Fatalf("the English rule book lists every rule: %d %v", code, body)
	}

	if state := explainIn(t, e, id, false); state["explainLanguage"] != "ru" {
		t.Fatalf("switching back: %v", state)
	}

	if rule = ruleOfPair(t, e); !strings.Contains(rule["title"].(string), "что было до прошлого") {
		t.Fatalf("back in the learner's own language: %v", rule["title"])
	}
}

func TestOnlyTheLearnersOwnTeacherCanBeSwitched(t *testing.T) {
	e := setup(t)

	if code, _ := request(t, e.e, "PATCH", teachersPath+"/"+"00000000-0000-4000-8000-000000000000", e.session, map[string]any{"explainInTarget": true}); code != http.StatusNotFound {
		t.Fatalf("a teacher that is not there: %d", code)
	}

	if code, _ := request(t, e.e, "PATCH", teachersPath+"/nope", e.session, map[string]any{"explainInTarget": true}); code != http.StatusBadRequest && code != http.StatusNotFound {
		t.Fatalf("a malformed id: %d", code)
	}
}

// TestSwitchingEndsThePracticeSittingOfThatLanguage: what the agent wrote for a
// rule is in the old language, and a sitting is not a record, so it goes; asking
// for what the teacher already does leaves it alone.
func TestSwitchingEndsThePracticeSittingOfThatLanguage(t *testing.T) {
	e := setup(t)
	withTeacher(t, e)

	code, body := request(t, e.e, "GET", teachersPath, e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("teachers: %d %v", code, body)
	}

	id := listPayload(t, body)[0]["id"].(string)

	e.provider.enqueue(examplesReply("She had left.", "They had eaten.", "He had won.", "We had waited.", "I had slept."))

	if code, body := request(t, e.e, "POST", practicePath("/examples"), e.session, nil); code != http.StatusOK {
		t.Fatalf("examples: %d %v", code, body)
	}

	explainIn(t, e, id, false)

	if practiceRows(t, e) != 1 {
		t.Fatal("a switch to what the teacher already does must not end the sitting")
	}

	explainIn(t, e, id, true)

	if practiceRows(t, e) != 0 {
		t.Fatal("the sitting written in the old language stayed")
	}
}

// TestPracticeIsWrittenInTheLanguageTheTeacherIsSetTo: the writer of examples is
// told to explain in English and to translate the sentences into Russian, and
// the rule it is given is the English one.
func TestPracticeIsWrittenInTheLanguageTheTeacherIsSetTo(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	explainIn(t, e, teacherID(t, e), true)

	e.provider.enqueue(examplesReply("She had left.", "They had eaten.", "He had won.", "We had waited.", "I had slept."))

	if code, body := request(t, e.e, "POST", practicePath("/examples"), e.session, nil); code != http.StatusOK {
		t.Fatalf("examples: %d %v", code, body)
	}

	system := unescaped(e.provider.requests()[0])
	for _, want := range []string{`"explanationLanguage":"English"`, `"nativeLanguage":"Russian"`, "what happened before the past", "translation is the same sentence in the learner's own language"} {
		if !strings.Contains(system, want) {
			t.Errorf("the writer was not told %q:\n%s", want, system)
		}
	}

	e.provider.enqueue(blocksReply())

	if code, body := request(t, e.e, "POST", practicePath("/exercises"), e.session, nil); code != http.StatusOK {
		t.Fatalf("exercises: %d %v", code, body)
	}

	if drills := unescaped(e.provider.requests()[1]); !strings.Contains(drills, `"explanationLanguage":"English"`) || !strings.Contains(drills, "short, plain sentences") || strings.Contains(drills, `"nativeLanguage":"English"`) {
		t.Fatalf("the drills must be explained in English, to a Russian speaker:\n%s", drills)
	}
}

// TestAProgramIsBuiltInTheLanguageTheTeacherIsSetTo: the planner is told to write
// in English and chooses from the English library; the topics carry the English
// rule, while the key words keep their Russian translation.
func TestAProgramIsBuiltInTheLanguageTheTeacherIsSetTo(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	explainIn(t, e, teacherID(t, e), true)
	e.provider.setDefault(strings.NewReplacer("Вы путаете am, is и are.", "You mix up am, is and are.").Replace(programReply))

	path := analyze(t, e, climb(t, e, "A1"))
	if path["explainLanguage"] != "en" {
		t.Fatalf("the program says what it is explained in: %v", path["explainLanguage"])
	}

	topics := path["topics"].([]any)
	first := topics[0].(map[string]any)

	if first["title"] != "The verb to be: am, is, are" || !strings.Contains(first["theory"].(string), "We use to be in almost every sentence") {
		t.Fatalf("the topic must carry the English rule: %v", first)
	}

	if !strings.Contains(first["goal"].(string), "You mix up am, is and are.") {
		t.Fatalf("the reason is in the goal: %v", first["goal"])
	}

	food := topics[4].(map[string]any)
	words := food["vocabulary"].([]any)

	if len(words) < 6 || words[0].(map[string]any)["translation"] != "хлеб" {
		t.Fatalf("key words keep the translation into the learner's own language: %v", words)
	}

	planner := unescaped(e.provider.requests()[0])
	for _, want := range []string{`"explanationLanguage":"English"`, "The verb to be", "short, plain sentences"} {
		if !strings.Contains(planner, want) {
			t.Errorf("the planner was not told %q:\n%s", want, planner)
		}
	}

	if strings.Contains(planner, "Глагол") {
		t.Errorf("the planner was shown the Russian library:\n%s", planner)
	}
}

// TestAProgramFollowsTheTeacherAfterItIsSwitched: a program is saved with the
// text it was built with, but what the learner reads follows the teacher now.
func TestAProgramFollowsTheTeacherAfterItIsSwitched(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	id := teacherID(t, e)
	analyze(t, e, climb(t, e, "A1"))

	read := func() map[string]any {
		code, body := request(t, e.e, "GET", "/api/v1/learning/path?language=en", e.session, nil)
		if code != http.StatusOK {
			t.Fatalf("path: %d %v", code, body)
		}

		return data(t, body)
	}

	if title := read()["topics"].([]any)[0].(map[string]any)["title"]; title != "Глагол to be: am, is, are" {
		t.Fatalf("built in Russian: %v", title)
	}

	explainIn(t, e, id, true)

	path := read()
	first := path["topics"].([]any)[0].(map[string]any)

	if first["title"] != "The verb to be: am, is, are" || !strings.Contains(first["goal"].(string), "Say who you are") {
		t.Fatalf("the topic must be shown in English once the teacher explains in it: %v", first)
	}

	if path["explainLanguage"] != "en" || path["translationLanguage"] != "ru" {
		t.Fatalf("languages: %v %v", path["explainLanguage"], path["translationLanguage"])
	}

	// A topic opened on its own shows its rule and its text in the same language.
	code, body := request(t, e.e, "GET", "/api/v1/learning/path/topics/"+first["id"].(string), e.session, nil)
	if code != http.StatusOK {
		t.Fatalf("topic: %d %v", code, body)
	}

	detail := data(t, body)
	if detail["topic"].(map[string]any)["title"] != "The verb to be: am, is, are" || detail["rule"].(map[string]any)["title"] != "The verb to be: am, is, are" {
		t.Fatalf("the topic and its rule disagree: %v", detail)
	}

	explainIn(t, e, id, false)

	if title := read()["topics"].([]any)[0].(map[string]any)["title"]; title != "Глагол to be: am, is, are" {
		t.Fatalf("back in Russian: %v", title)
	}
}

// TestExercisesAndMistakesAreExplainedInTheLanguageTheTeacherIsSetTo: the set of
// exercises and the explanation of a wrong answer are both asked for in English.
func TestExercisesAndMistakesAreExplainedInTheLanguageTheTeacherIsSetTo(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	id := teacherID(t, e)
	path := analyze(t, e, climb(t, e, "A1"))
	topicID := path["topics"].([]any)[0].(map[string]any)["id"].(string)

	explainIn(t, e, id, true)

	e.provider.enqueue(exercisesReply)

	exercises := generateExercises(t, e, topicID)
	asked := e.provider.requests()
	writer := unescaped(asked[len(asked)-1])

	for _, want := range []string{`"explanationLanguage":"English"`, "explanation language", "short, plain sentences"} {
		if !strings.Contains(writer, want) {
			t.Errorf("the exercise writer was not told %q:\n%s", want, writer)
		}
	}

	exercise := exercises[0].(map[string]any)

	e.provider.enqueue(`{"correct":false,"feedback":"With she we add -s."}`)

	code, body := request(t, e.e, "POST", "/api/v1/learning/path/exercises/"+exercise["id"].(string)+"/attempt", e.session, map[string]any{"answer": "go"})
	if code != http.StatusOK {
		t.Fatalf("attempt: %d %v", code, body)
	}

	asked = e.provider.requests()
	if review := unescaped(asked[len(asked)-1]); !strings.Contains(review, `"explanationLanguage":"English"`) {
		t.Errorf("the mistake was not to be explained in English:\n%s", review)
	}
}

// TestSwitchingDropsOnlyTheExerciseSetsNobodyStarted: the instructions of an
// exercise are written in the explanation language, so a set the learner has
// not touched is written again in the new one when its topic is opened. A set
// with an answer in it, and a completed topic's, stay as they were.
func TestSwitchingDropsOnlyTheExerciseSetsNobodyStarted(t *testing.T) {
	e := setup(t)
	chooseAgent(t, e)
	e.provider.setDefault(programReply)

	id := teacherID(t, e)
	topics := analyze(t, e, climb(t, e, "A1"))["topics"].([]any)
	untouched := topics[0].(map[string]any)["id"].(string)
	answered := topics[1].(map[string]any)["id"].(string)
	completed := topics[2].(map[string]any)["id"].(string)

	e.provider.enqueue(exercisesReply)
	generateExercises(t, e, untouched)

	e.provider.enqueue(exercisesReply)
	set := generateExercises(t, e, answered)
	attempt(t, e, set[0].(map[string]any)["id"].(string), "goes", true, "")

	e.provider.enqueue(exercisesReply)
	generateExercises(t, e, completed)

	if code, body := request(t, e.e, "POST", "/api/v1/learning/path/topics/"+completed+"/complete", e.session, nil); code != http.StatusOK {
		t.Fatalf("complete: %d %v", code, body)
	}

	count := func(topicID string) int { return len(topicDetail(t, e, topicID)["exercises"].([]any)) }

	// Switching to what the teacher already does is no switch.
	explainIn(t, e, id, false)

	if count(untouched) != 3 {
		t.Fatal("a switch to the same language dropped a set")
	}

	explainIn(t, e, id, true)

	if n := count(untouched); n != 0 {
		t.Fatalf("the untouched set, written in the old language, stayed: %d exercises", n)
	}

	if count(answered) != 3 || count(completed) != 3 {
		t.Fatal("a set with an answer in it, or a completed topic's, was dropped")
	}

	// Opened again, the topic is written in the new language.
	e.provider.enqueue(exercisesReply)
	generateExercises(t, e, untouched)

	asked := e.provider.requests()
	if writer := unescaped(asked[len(asked)-1]); !strings.Contains(writer, `"explanationLanguage":"English"`) {
		t.Errorf("the set was not written in English:\n%s", writer)
	}
}
