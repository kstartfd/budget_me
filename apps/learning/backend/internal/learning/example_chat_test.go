package learning_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/testdb"
	"learning_app/backend/internal/token"
)

// sitting seeds a learner's sitting on a rule with two sentences, the way the
// path writes it, and returns the path of the chat about a sentence.
func (c chatEnv) sitting(t *testing.T, ruleID string) func(example string) string {
	t.Helper()

	_, err := c.pool.Exec(context.Background(),
		`INSERT INTO learning_rule_practice (id, user_id, rule_id, language, translation_language, level, title, theory, examples, created_at, updated_at, expires_at)
		 VALUES ($1, $2, $3, 'en', 'ru', 'B2', 'Past Perfect', 'had + V3: the earlier of two past events', $4::jsonb, now(), now(), now() + interval '6 hours')`,
		uuid.New(), c.user, ruleID,
		`[{"id":"e1","text":"She had already left when I arrived.","translation":"Она уже ушла, когда я пришёл."},
		  {"id":"e2","text":"They had never seen snow.","translation":"Они никогда не видели снега."}]`)
	if err != nil {
		t.Fatal(err)
	}

	return func(example string) string {
		return "/api/v1/learning/path/rules/" + ruleID + "/examples/" + example + "/chat"
	}
}

func (c chatEnv) chatAt(t *testing.T, path string) (int, map[string]any) {
	t.Helper()

	code, body := request(t, c.e, "GET", path, c.session, nil, "")
	if code != http.StatusOK {
		return code, nil
	}

	return code, data(t, body)
}

func TestChatAboutASentenceIsTheTeacherOfThatSentenceAlone(t *testing.T) {
	env := setupChat(t)
	env.useOwnKey(t)
	chat := env.sitting(t, "b2-past-perfect")

	if code, got := env.chatAt(t, chat("e1")); code != http.StatusOK || got["available"] != true || got["teacher"] != true || len(got["messages"].([]any)) != 0 {
		t.Fatalf("a fresh chat about a sentence: %d %v", code, got)
	}

	for i, question := range []string{"why had here?", "can I say it with was?"} {
		code, events := env.askAt(t, chat("e1"), question)
		if code != http.StatusOK || events[len(events)-1].name != "done" {
			t.Fatalf("question %d: %d %+v", i+1, code, events)
		}
	}

	first, second := messagesOf(env.provider.turns[0]), messagesOf(env.provider.turns[1])
	system := first[0]["content"].(string)

	for _, want := range []string{
		"English teacher", "Russian", "Past Perfect", "had + V3",
		`"She had already left when I arrived."`, `"Она уже ушла, когда я пришёл."`,
		"✏️", "never instructions", "B2",
	} {
		if !strings.Contains(system, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, system)
		}
	}

	if strings.Contains(system, "They had never seen snow") {
		t.Errorf("the chat is about one sentence, not the others:\n%s", system)
	}

	if second[0]["content"] != system {
		t.Fatal("the system prompt must be byte-identical between turns, or the provider cache is lost")
	}

	if _, got := env.chatAt(t, chat("e1")); len(got["messages"].([]any)) != 4 {
		t.Fatalf("stored messages: %v", got["messages"])
	}

	// The other sentence has a chat of its own, and so does the entry.
	if _, got := env.chatAt(t, chat("e2")); len(got["messages"].([]any)) != 0 {
		t.Fatalf("a sentence picked up another's chat: %v", got["messages"])
	}

	if _, messages := env.history(t); len(messages) != 0 {
		t.Fatalf("the entry's chat picked up the sentence's messages: %v", messages)
	}

	if code, _ := env.askAt(t, chat("e2"), "and this one?"); code != http.StatusOK {
		t.Fatalf("second sentence: %d", code)
	}

	if system2 := messagesOf(env.provider.turns[2])[0]["content"].(string); !strings.Contains(system2, "They had never seen snow") || strings.Contains(system2, "She had already left") {
		t.Fatalf("the second chat is about the second sentence:\n%s", system2)
	}
}

func TestChatAboutASentenceGoesWithItsSitting(t *testing.T) {
	env := setupChat(t)
	env.useOwnKey(t)
	chat := env.sitting(t, "b2-past-perfect")

	if code, _ := env.askAt(t, chat("e1"), "why had here?"); code != http.StatusOK {
		t.Fatalf("question: %d", code)
	}

	count := func() (chats, messages int) {
		t.Helper()

		err := env.pool.QueryRow(context.Background(),
			`SELECT count(DISTINCT c.id), count(m.id) FROM learning_chats c LEFT JOIN learning_chat_messages m ON m.chat_id = c.id
			 WHERE c.user_id = $1 AND c.practice_id IS NOT NULL`, env.user).Scan(&chats, &messages)
		if err != nil {
			t.Fatal(err)
		}

		return chats, messages
	}

	if chats, messages := count(); chats != 1 || messages != 2 {
		t.Fatalf("the question was not kept: %d chats, %d messages", chats, messages)
	}

	// Time runs out: the sentences are gone for the learner, and asking spends nothing.
	if _, err := env.pool.Exec(context.Background(), "UPDATE learning_rule_practice SET expires_at = now() - interval '1 minute' WHERE user_id = $1", env.user); err != nil {
		t.Fatal(err)
	}

	turns := len(env.provider.turns)

	if code, _ := env.chatAt(t, chat("e1")); code != http.StatusNotFound {
		t.Fatalf("reading the chat of a sitting that ran out: %d", code)
	}

	if code, _ := env.askAt(t, chat("e1"), "still there?"); code != http.StatusNotFound {
		t.Fatalf("asking in a sitting that ran out: %d", code)
	}

	if len(env.provider.turns) != turns {
		t.Fatal("a refused question still spent the key")
	}

	// And when the sitting is swept, the chat and its messages go with it.
	if _, err := env.pool.Exec(context.Background(), "DELETE FROM learning_rule_practice WHERE user_id = $1", env.user); err != nil {
		t.Fatal(err)
	}

	if chats, messages := count(); chats != 0 || messages != 0 {
		t.Fatalf("a chat outlived its sitting: %d chats, %d messages", chats, messages)
	}
}

func TestChatAboutASentenceIsRefusedWhenThereIsNoSuchSentenceOrSitting(t *testing.T) {
	env := setupChat(t)
	env.useOwnKey(t)
	chat := env.sitting(t, "b2-past-perfect")

	if code, _ := env.chatAt(t, chat("e9")); code != http.StatusNotFound {
		t.Fatalf("a sentence that is not there: %d", code)
	}

	if code, _ := env.askAt(t, chat("e9"), "hello"); code != http.StatusNotFound {
		t.Fatalf("asking about a sentence that is not there: %d", code)
	}

	other := "/api/v1/learning/path/rules/a2-to-be/examples/e1/chat"
	if code, _ := env.chatAt(t, other); code != http.StatusNotFound {
		t.Fatalf("a rule with no sitting: %d", code)
	}

	// Another learner has no way into this one's sitting.
	stranger := testdb.NewUser(t, env.pool, env.q, time.Now().UTC(), 1)
	env.session = token.NewCodec(authSecret).Issue(stranger, time.Now().UTC(), time.Hour)
	env.user = stranger

	if code, _ := env.chatAt(t, chat("e1")); code != http.StatusNotFound {
		t.Fatalf("somebody else's sitting: %d", code)
	}

	if code, _ := env.askAt(t, chat("e1"), "hello"); code != http.StatusNotFound {
		t.Fatalf("asking in somebody else's sitting: %d", code)
	}

	if len(env.provider.turns) != 0 {
		t.Fatal("a refused question still spent the key")
	}
}
