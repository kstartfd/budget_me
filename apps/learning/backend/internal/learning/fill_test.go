package learning_test

import (
	"context"
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

const filledReply = `{"transcription":"/ˈlɑːfɪŋ/","translation":"смех","explanation":"laugh + ing","nativeUsage":"обычное слово","example":"She was laughing.","rule":"глагол + ing"}`

func fillPath(entryID string) string { return "/api/v1/learning/entries/" + entryID + "/fill" }

func storedEntry(t *testing.T, env aiEnv, id string) db.LearningEntry {
	t.Helper()

	row, err := env.q.GetLearningEntry(context.Background(), db.GetLearningEntryParams{ID: ids.ToPg(uuid.MustParse(id)), UserID: ids.ToPg(env.user)})
	if err != nil {
		t.Fatal(err)
	}

	return row
}

// TestFillWritesTheCardWithTheLearnersAgent: the button on an empty card has the
// agent write every field the card lacks, and nothing the learner wrote is lost.
func TestFillWritesTheCardWithTheLearnersAgent(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)
	chooseAgent(t, env)

	entryID := createEntry(t, env.e, env.session, topicID, "laughing")
	env.provider.reply(filledReply)

	before := len(env.provider.requests())

	code, body := request(t, env.e, "POST", fillPath(entryID), env.session, nil, "")
	if code != http.StatusOK {
		t.Fatalf("fill: %d %v", code, body)
	}

	got := data(t, body)
	if got["translation"] != "смех" || got["example"] != "She was laughing." || got["rule"] != "глагол + ing" || got["transcription"] != "/ˈlɑːfɪŋ/" {
		t.Fatalf("the card was not written: %v", got)
	}

	if stored := storedEntry(t, env, entryID); stored.Translation != "смех" || stored.Explanation != "laugh + ing" || stored.NativeUsage != "обычное слово" {
		t.Fatalf("what the agent wrote was not kept: %+v", stored)
	}

	calls := env.provider.requests()[before:]
	if len(calls) != 1 || !strings.Contains(calls[0].Body, "laughing") || calls[0].Key != "sk-ant-test-key-1234" {
		t.Fatalf("one call with the text and the learner's own key: %+v", calls)
	}

	if strings.Contains(string(mustJSON(t, body)), "sk-ant") {
		t.Fatalf("the response carries the key: %v", body)
	}

	// A complete card is not sent anywhere again.
	code, _ = request(t, env.e, "POST", fillPath(entryID), env.session, nil, "")
	if code != http.StatusOK || len(env.provider.requests()) != before+1 {
		t.Fatalf("a complete card went to the agent again: %d, %d calls", code, len(env.provider.requests())-before)
	}
}

// TestFillKeepsWhatTheLearnerWrote: a field already on the card is neither asked for nor replaced.
func TestFillKeepsWhatTheLearnerWrote(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)
	chooseAgent(t, env)

	code, body := request(t, env.e, "POST", "/api/v1/learning/entries", env.session, map[string]any{"topicId": topicID, "kind": "word", "text": "posture", "translation": "осанка"}, "")
	if code != http.StatusCreated {
		t.Fatalf("entry: %d %v", code, body)
	}

	entryID := data(t, body)["id"].(string)

	env.provider.reply(`{"translation":"агент","explanation":"агент","example":"агент"}`)

	before := len(env.provider.requests())

	code, body = request(t, env.e, "POST", fillPath(entryID), env.session, nil, "")
	if code != http.StatusOK || data(t, body)["translation"] != "осанка" || data(t, body)["explanation"] != "агент" {
		t.Fatalf("fill: %d %v", code, body)
	}

	if prompt := env.provider.requests()[before].Body; strings.Contains(prompt, `\"translation\"`) {
		t.Fatalf("asked for a field that was already there: %s", prompt)
	}
}

// TestFillNeedsAnAgentAndTheSession: no agent is a message to act on, an extension
// token and another account get nowhere, and none of them spends a call.
func TestFillNeedsAnAgentAndTheSession(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)
	_, extension := apiToken(t, env.e, env.session, "read_write", topicID)
	entryID := createEntry(t, env.e, env.session, topicID, "laughing")

	code, body := request(t, env.e, "POST", fillPath(entryID), env.session, nil, "")
	if code != http.StatusBadRequest || !strings.Contains(body["message"].(string), "AI agent") {
		t.Fatalf("without an agent: %d %v", code, body)
	}

	chooseAgent(t, env)
	env.provider.reply(filledReply)

	if code, _ := request(t, env.e, "POST", fillPath(entryID), extension, nil, ""); code != http.StatusUnauthorized {
		t.Fatalf("an extension token spent the learner's key: %d", code)
	}

	stranger := testdb.NewUser(t, env.pool, env.q, time.Now().UTC(), 2)
	session := token.NewCodec(authSecret).Issue(stranger, time.Now().UTC(), time.Hour)

	if code, _ := request(t, env.e, "POST", fillPath(entryID), session, nil, ""); code != http.StatusNotFound {
		t.Fatalf("another account filled this card: %d", code)
	}

	if code, _ := request(t, env.e, "POST", fillPath(uuid.NewString()), env.session, nil, ""); code != http.StatusNotFound {
		t.Fatalf("an entry that is not there: %d", code)
	}

	if calls := env.provider.requests(); len(calls) != 0 {
		t.Fatalf("refused requests spent %d calls", len(calls))
	}

	if stored := storedEntry(t, env, entryID); stored.Translation != "" {
		t.Fatalf("a refused fill wrote the card: %+v", stored)
	}
}

// TestFillSaysWhatWentWrongAndLeavesTheCard: a refused key, an outage and an answer
// that holds nothing to write are told apart, and the card stays as it was.
func TestFillSaysWhatWentWrongAndLeavesTheCard(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)
	chooseAgent(t, env)
	entryID := createEntry(t, env.e, env.session, topicID, "laughing")

	for _, tc := range []struct {
		name   string
		status int32
		reply  string
		want   int
	}{
		{"key refused", http.StatusUnauthorized, "x", http.StatusBadRequest},
		{"outage", http.StatusServiceUnavailable, "x", http.StatusBadGateway},
		{"rejected request", http.StatusBadRequest, "x", http.StatusBadGateway},
		{"unreadable answer", http.StatusOK, "Sure! Here is the translation.", http.StatusBadGateway},
		{"answer of the wrong shape", http.StatusOK, `{"translation":["not","a","string"]}`, http.StatusBadGateway},
	} {
		env.keyStatus.Store(tc.status)
		env.provider.reply(tc.reply)

		code, body := request(t, env.e, "POST", fillPath(entryID), env.session, nil, "")
		if code != tc.want {
			t.Errorf("%s: %d %v, want %d", tc.name, code, body, tc.want)
		}

		if stored := storedEntry(t, env, entryID); stored.Translation != "" || stored.Example != "" {
			t.Errorf("%s: a failed fill wrote the card: %+v", tc.name, stored)
		}
	}
}
