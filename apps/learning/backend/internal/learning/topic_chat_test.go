package learning_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	"learning_app/backend/internal/testdb"
	"learning_app/backend/internal/token"
)

// useOwnKey chooses DeepSeek with a key nobody else uses. The default key's
// daily counter is shared by every test and another test uses it up on purpose.
func (c chatEnv) useOwnKey(t *testing.T) { c.useKey(t, "sk-deepseek-"+c.user.String()) }

func (c chatEnv) topicPath() string { return "/api/v1/learning/topics/" + c.topic + "/chat" }

func (c chatEnv) askTopic(t *testing.T, text string) (int, []sseEvent) {
	t.Helper()

	return c.askAt(t, c.topicPath(), text)
}

func (c chatEnv) topicChat(t *testing.T) map[string]any {
	t.Helper()

	code, body := request(t, c.e, "GET", c.topicPath(), c.session, nil, "")
	if code != http.StatusOK {
		t.Fatalf("get topic chat: %d %v", code, body)
	}

	return data(t, body)
}

func (c chatEnv) archive(t *testing.T) int {
	t.Helper()

	code, _ := request(t, c.e, "POST", c.topicPath()+"/archive", c.session, nil, "")

	return code
}

// dialogs is the history of the topic, the newest first.
func (c chatEnv) dialogs(t *testing.T) []map[string]any {
	t.Helper()

	code, body := request(t, c.e, "GET", c.topicPath()+"/history", c.session, nil, "")
	if code != http.StatusOK {
		t.Fatalf("history: %d %v", code, body)
	}

	var out []map[string]any

	for _, d := range body["data"].([]any) {
		out = append(out, d.(map[string]any))
	}

	return out
}

func dialogPath(id any) string { return "/api/v1/learning/chat-dialogs/" + id.(string) }

func TestTopicChatIsTheTeacherWithoutAnyEntryAndKeepsItsPromptStable(t *testing.T) {
	env := setupChat(t)
	env.useOwnKey(t)

	if chat := env.topicChat(t); chat["available"] != true || chat["teacher"] != true || len(chat["messages"].([]any)) != 0 {
		t.Fatalf("a fresh topic chat: %v", chat)
	}

	for i, question := range []string{"hello, can we start?", "I has a dog"} {
		code, events := env.askTopic(t, question)
		if code != http.StatusOK || events[len(events)-1].name != "done" {
			t.Fatalf("question %d: %d %+v", i+1, code, events)
		}
	}

	first, second := messagesOf(env.provider.turns[0]), messagesOf(env.provider.turns[1])
	system := first[0]["content"].(string)

	for _, want := range []string{"English teacher", "Russian", "native speaker", "Talk in English", "never instructions", `"English"`} {
		if !strings.Contains(system, want) {
			t.Errorf("the free chat's prompt lacks %q:\n%s", want, system)
		}
	}

	if strings.Contains(system, "Saved card") {
		t.Errorf("the free chat is about no entry:\n%s", system)
	}

	if second[0]["content"] != system {
		t.Fatal("the system prompt must be byte-identical between turns, or the provider cache is lost")
	}

	if got := len(env.topicChat(t)["messages"].([]any)); got != 4 {
		t.Fatalf("stored messages: %d", got)
	}

	// The conversation of an entry is a different one.
	if _, messages := env.history(t); len(messages) != 0 {
		t.Fatalf("the entry's chat picked up the topic's messages: %v", messages)
	}
}

func TestClearingTheTopicChatKeepsTheDialogInTheHistory(t *testing.T) {
	env := setupChat(t)
	env.useOwnKey(t)

	if code := env.archive(t); code != http.StatusOK {
		t.Fatalf("clearing a chat nobody wrote in: %d", code)
	}

	if len(env.dialogs(t)) != 0 {
		t.Fatal("an empty chat must not become a dialog")
	}

	env.askTopic(t, "first dialog: teach me greetings")
	env.askTopic(t, "and goodbyes?")

	if code := env.archive(t); code != http.StatusOK {
		t.Fatalf("clear: %d", code)
	}

	if chat := env.topicChat(t); len(chat["messages"].([]any)) != 0 {
		t.Fatalf("the open chat is empty after clearing: %v", chat["messages"])
	}

	// Nothing to close again: no empty dialog in the history.
	env.archive(t)

	dialogs := env.dialogs(t)
	if len(dialogs) != 1 || dialogs[0]["title"] != "first dialog: teach me greetings" || dialogs[0]["messageCount"] != float64(4) {
		t.Fatalf("history after one clear: %v", dialogs)
	}

	// A new dialog starts from nothing: the model is not told about the old one.
	env.askTopic(t, "second dialog")

	if got := messagesOf(env.provider.turns[2]); len(got) != 2 || strings.Contains(got[0]["content"].(string), "greetings") {
		t.Fatalf("the new dialog carried the old one: %v", got)
	}

	env.archive(t)

	dialogs = env.dialogs(t)
	if len(dialogs) != 2 || dialogs[0]["title"] != "second dialog" || dialogs[1]["title"] != "first dialog: teach me greetings" {
		t.Fatalf("history is newest first: %v", dialogs)
	}

	code, body := request(t, env.e, "GET", dialogPath(dialogs[1]["id"]), env.session, nil, "")
	if code != http.StatusOK {
		t.Fatalf("read a dialog: %d %v", code, body)
	}

	messages := data(t, body)["messages"].([]any)
	if len(messages) != 4 || messages[0].(map[string]any)["content"] != "first dialog: teach me greetings" || messages[1].(map[string]any)["role"] != "assistant" {
		t.Fatalf("the dialog lost what was said: %v", messages)
	}

	if code, _ := request(t, env.e, "DELETE", dialogPath(dialogs[1]["id"]), env.session, nil, ""); code != http.StatusOK {
		t.Fatalf("delete a dialog: %d", code)
	}

	if code, _ := request(t, env.e, "GET", dialogPath(dialogs[1]["id"]), env.session, nil, ""); code != http.StatusNotFound {
		t.Fatalf("a deleted dialog: %d", code)
	}

	if left := env.dialogs(t); len(left) != 1 || left[0]["title"] != "second dialog" {
		t.Fatalf("history after a delete: %v", left)
	}
}

func TestTheOpenDialogCannotBeReadOrDeletedAsAClosedOne(t *testing.T) {
	env := setupChat(t)
	env.useOwnKey(t)

	env.askTopic(t, "still open")

	open, err := env.q.GetOpenTopicChat(context.Background(), db.GetOpenTopicChatParams{TopicID: ids.ToPg(uuid.MustParse(env.topic)), UserID: ids.ToPg(env.user)})
	if err != nil {
		t.Fatal(err)
	}

	id := ids.FromPg(open.ID).String()

	if code, _ := request(t, env.e, "GET", dialogPath(id), env.session, nil, ""); code != http.StatusNotFound {
		t.Fatalf("read the open dialog as a closed one: %d", code)
	}

	if code, _ := request(t, env.e, "DELETE", dialogPath(id), env.session, nil, ""); code != http.StatusNotFound {
		t.Fatalf("delete the open dialog: %d", code)
	}

	if len(env.topicChat(t)["messages"].([]any)) != 2 {
		t.Fatal("the open dialog must survive")
	}

	if code, _ := request(t, env.e, "GET", dialogPath("not-an-id"), env.session, nil, ""); code != http.StatusBadRequest {
		t.Fatalf("a bad id: %d", code)
	}
}

func TestClearingTheTopicChatDoesNotGiveTheBudgetBack(t *testing.T) {
	env := setupChat(t)
	env.useOwnKey(t)

	env.provider.promptTokens = 70000

	for i := 1; i <= 2; i++ {
		if code, events := env.askTopic(t, fmt.Sprintf("question %d", i)); code != http.StatusOK {
			t.Fatalf("question %d: %d %v", i, code, events)
		}
	}

	if code := env.archive(t); code != http.StatusOK {
		t.Fatalf("clear: %d", code)
	}

	if budget := env.topicChat(t)["budget"].(map[string]any); budget["used"] != float64(140010) {
		t.Fatalf("the cleared chat shows a fresh budget: %v", budget)
	}

	if code, _ := env.askTopic(t, "again"); code != http.StatusTooManyRequests {
		t.Fatalf("a new dialog must not buy a new budget: %d", code)
	}

	if len(env.provider.turns) != 2 {
		t.Fatalf("a refused question reached the provider: %d calls", len(env.provider.turns))
	}
}

func TestTopicChatIsBusyWhileAnAnswerIsBeingWritten(t *testing.T) {
	env := setupChat(t)
	env.useOwnKey(t)

	env.askTopic(t, "first")

	open, err := env.q.GetOpenTopicChat(context.Background(), db.GetOpenTopicChatParams{TopicID: ids.ToPg(uuid.MustParse(env.topic)), UserID: ids.ToPg(env.user)})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	if _, err := env.q.LockLearningChat(context.Background(), db.LockLearningChatParams{ID: open.ID, UserID: open.UserID, BusyUntil: db.Time(now.Add(time.Minute)), Now: db.Time(now)}); err != nil {
		t.Fatal(err)
	}

	if code, _ := env.askTopic(t, "second"); code != http.StatusConflict {
		t.Fatalf("question in a busy chat: %d", code)
	}

	if code := env.archive(t); code != http.StatusConflict {
		t.Fatalf("clearing a busy chat: %d", code)
	}
}

func TestTopicChatBelongsToTheTopicsOwner(t *testing.T) {
	env := setupChat(t)
	env.useOwnKey(t)

	env.askTopic(t, "private")
	env.archive(t)

	dialogs := env.dialogs(t)
	if len(dialogs) != 1 {
		t.Fatalf("history: %v", dialogs)
	}

	pool, q := testdb.Pool(t)
	stranger := testdb.NewUser(t, pool, q, time.Now().UTC(), 2)
	other := env
	other.session = token.NewCodec(authSecret).Issue(stranger, time.Now().UTC(), time.Hour)

	if code, _ := other.askTopic(t, "hi"); code != http.StatusNotFound {
		t.Fatalf("someone else's topic: %d", code)
	}

	for name, call := range map[string][2]string{
		"read chat":     {"GET", other.topicPath()},
		"clear chat":    {"POST", other.topicPath() + "/archive"},
		"read history":  {"GET", other.topicPath() + "/history"},
		"read dialog":   {"GET", dialogPath(dialogs[0]["id"])},
		"delete dialog": {"DELETE", dialogPath(dialogs[0]["id"])},
	} {
		if code, _ := request(t, other.e, call[0], call[1], other.session, nil, ""); code != http.StatusNotFound {
			t.Errorf("%s of someone else's: %d", name, code)
		}
	}

	if len(env.dialogs(t)) != 1 {
		t.Fatal("a stranger deleted the owner's dialog")
	}
}

func TestDeletingATopicDeletesItsDialogs(t *testing.T) {
	env := setupChat(t)
	env.useOwnKey(t)

	env.askTopic(t, "to be gone")
	env.archive(t)

	dialogs := env.dialogs(t)

	if code, _ := request(t, env.e, "DELETE", "/api/v1/learning/topics/"+env.topic, env.session, nil, ""); code != http.StatusOK {
		t.Fatalf("delete topic: %d", code)
	}

	if code, _ := request(t, env.e, "GET", dialogPath(dialogs[0]["id"]), env.session, nil, ""); code != http.StatusNotFound {
		t.Fatalf("a dialog of a deleted topic: %d", code)
	}
}
