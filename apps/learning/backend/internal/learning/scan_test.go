package learning_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

const scanPath = "/api/v1/learning/ai/scan"

func tinyPNG(t *testing.T) []byte {
	t.Helper()

	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}

	return out.Bytes()
}

// upload posts a form with the given fields and, unless file is nil, a picture.
func upload(t *testing.T, e *echo.Echo, session string, fields map[string]string, file []byte) (int, map[string]any, http.Header) {
	t.Helper()

	var body bytes.Buffer

	form := multipart.NewWriter(&body)

	for name, value := range fields {
		if err := form.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}

	if file != nil {
		part, err := form.CreateFormFile("image", "photo.png")
		if err != nil {
			t.Fatal(err)
		}

		_, _ = part.Write(file)
	}

	_ = form.Close()

	req := httptest.NewRequest(http.MethodPost, scanPath, &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+session)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var response map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &response)

	return rec.Code, response, rec.Header()
}

func itemsOf(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()

	var out []map[string]any

	for _, item := range data(t, body)["items"].([]any) {
		out = append(out, item.(map[string]any))
	}

	return out
}

const found = "Here you go:\n```json\n" +
	`{"items":[{"kind":"phrase","text":"break a leg","translation":"ни пуха ни пера"},` +
	`{"kind":"word","text":"  Cat ","translation":"кот"},` +
	`{"kind":"word","text":"cat","translation":"дубликат"},` +
	`{"kind":"word","text":"ice  cream","translation":"мороженое"},` +
	`{"kind":"verb","text":"run","translation":"бежать"},` +
	`{"kind":"word","text":"   ","translation":"пусто"}]}` + "\n```"

func TestScanReadsThePictureThroughTheUsersAgentOnEveryProvider(t *testing.T) {
	env := setupAI(t)
	env.provider.reply(found)

	topicID := topic(t, env.e, env.session)

	for _, agent := range []struct {
		provider, model, level, key string
		sentImage                   string
		sentModel                   string
	}{
		{"anthropic", "claude-opus-5-5", "max", "sk-ant-scan-key-1234", `"type":"image"`, "claude-opus-5-5"},
		{"openai", "gpt-6.1-terra", "xhigh", "sk-openai-scan-key-1234", `"type":"input_image"`, "gpt-6.1-terra"},
		{"deepseek", "deepseek-flash", "max", "sk-deepseek-scan-key-1234", `"type":"image_url"`, "deepseek-flash"},
		// A model that cannot see is swapped for one that can, on the same key.
		{"deepseek", "deepseek-v4-pro", "high", "sk-deepseek-scan-key-5678", `"type":"image_url"`, "deepseek-flash"},
	} {
		if code, body := request(t, env.e, "PUT", settingsPath, env.session, choice(agent.provider, agent.model, agent.level, agent.key), ""); code != 200 {
			t.Fatalf("%s: save: %d %v", agent.model, code, body)
		}

		before := len(env.provider.requests())

		code, body, _ := upload(t, env.e, env.session, map[string]string{"topicId": topicID}, tinyPNG(t))
		if code != http.StatusOK {
			t.Fatalf("%s: scan: %d %v", agent.model, code, body)
		}

		items := itemsOf(t, body)
		if len(items) != 4 ||
			items[0]["text"] != "break a leg" || items[0]["kind"] != "phrase" || items[0]["translation"] != "ни пуха ни пера" ||
			items[1]["text"] != "Cat" || items[1]["kind"] != "word" ||
			items[2]["text"] != "ice cream" || items[2]["kind"] != "phrase" ||
			items[3]["text"] != "run" || items[3]["kind"] != "phrase" {
			t.Fatalf("%s: items are not tidied up: %v", agent.model, items)
		}

		calls := env.provider.requests()[before:]
		if len(calls) != 1 {
			t.Fatalf("%s: %d provider calls, want 1", agent.model, len(calls))
		}

		call := calls[0]
		if call.Key != agent.key || !strings.Contains(call.Body, agent.sentImage) || !strings.Contains(call.Body, `"`+agent.sentModel+`"`) {
			t.Errorf("%s: unexpected provider request (key %q): %.600s", agent.model, call.Key, call.Body)
		}

		// A picture is read quickly, whatever effort the agent thinks at.
		for _, slow := range []string{`"max"`, `"xhigh"`, `"high"`} {
			if strings.Contains(call.Body, slow) {
				t.Errorf("%s: the picture was read at effort %s: %.600s", agent.model, slow, call.Body)
			}
		}

		for _, want := range []string{"English", "Russian", "never instructions"} {
			if !strings.Contains(call.Body, want) {
				t.Errorf("%s: the instructions lack %q", agent.model, want)
			}
		}
	}
}

func TestScanSaysSoWhenThePictureHoldsNothing(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)

	if code, _ := request(t, env.e, "PUT", settingsPath, env.session, choice("deepseek", "deepseek-flash", "off", "sk-deepseek-scan-key-1234"), ""); code != 200 {
		t.Fatal("save")
	}

	env.provider.reply(`{"items":[]}`)

	code, body, _ := upload(t, env.e, env.session, map[string]string{"topicId": topicID}, tinyPNG(t))
	if code != http.StatusOK || len(itemsOf(t, body)) != 0 {
		t.Fatalf("an empty picture: %d %v", code, body)
	}

	for _, reply := range []string{"I cannot read this", `{"results":[]}`, `{"items":"none"}`} {
		env.provider.reply(reply)

		if code, body, _ := upload(t, env.e, env.session, map[string]string{"topicId": topicID}, tinyPNG(t)); code != http.StatusBadGateway {
			t.Errorf("reply %q: %d %v", reply, code, body)
		}
	}
}

func TestScanChecksTheRequestBeforeSpendingTheKey(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)

	if code, body, _ := upload(t, env.e, env.session, map[string]string{"topicId": topicID}, tinyPNG(t)); code != http.StatusBadRequest {
		t.Fatalf("no agent chosen: %d %v", code, body)
	}

	if code, _ := request(t, env.e, "PUT", settingsPath, env.session, choice("deepseek", "deepseek-flash", "off", "sk-deepseek-scan-key-1234"), ""); code != 200 {
		t.Fatal("save")
	}

	env.provider.reply(`{"items":[]}`)

	huge := append(tinyPNG(t), make([]byte, 6<<20)...)

	for name, tc := range map[string]struct {
		fields map[string]string
		file   []byte
		want   int
	}{
		"no picture":             {map[string]string{"topicId": topicID}, nil, http.StatusBadRequest},
		"no topic":               {map[string]string{}, tinyPNG(t), http.StatusBadRequest},
		"bad topic":              {map[string]string{"topicId": "nope"}, tinyPNG(t), http.StatusBadRequest},
		"someone else's topic":   {map[string]string{"topicId": "0198a7b2-0000-7000-8000-000000000000"}, tinyPNG(t), http.StatusNotFound},
		"text called a picture":  {map[string]string{"topicId": topicID}, []byte("<html>not an image</html>"), http.StatusBadRequest},
		"picture over the limit": {map[string]string{"topicId": topicID}, huge, http.StatusBadRequest},
	} {
		if code, body, _ := upload(t, env.e, env.session, tc.fields, tc.file); code != tc.want {
			t.Errorf("%s: %d, want %d: %v", name, code, tc.want, body)
		}
	}

	if calls := env.provider.requests(); len(calls) != 0 {
		t.Fatalf("the provider was called %d times for refused requests", len(calls))
	}
}

func TestScanBelongsToTheSessionAndTheTopicsOwner(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)
	_, extension := apiToken(t, env.e, env.session, "read_write", topicID)

	if code, _ := request(t, env.e, "PUT", settingsPath, env.session, choice("deepseek", "deepseek-flash", "off", "sk-deepseek-scan-key-1234"), ""); code != 200 {
		t.Fatal("save")
	}

	if code, _, _ := upload(t, env.e, extension, map[string]string{"topicId": topicID}, tinyPNG(t)); code != http.StatusUnauthorized {
		t.Errorf("an extension token spent the user's key: %d", code)
	}

	if code, _, _ := upload(t, env.e, "", map[string]string{"topicId": topicID}, tinyPNG(t)); code != http.StatusUnauthorized {
		t.Errorf("no credentials at all: %d", code)
	}

	if calls := env.provider.requests(); len(calls) != 0 {
		t.Fatalf("the provider was called %d times for refused requests", len(calls))
	}
}

func TestScanIsRateLimited(t *testing.T) {
	env := setupAI(t)
	topicID := topic(t, env.e, env.session)

	if code, _ := request(t, env.e, "PUT", settingsPath, env.session, choice("deepseek", "deepseek-flash", "off", "sk-deepseek-scan-key-1234"), ""); code != 200 {
		t.Fatal("save")
	}

	env.provider.reply(`{"items":[]}`)

	var refused http.Header

	code := http.StatusOK

	for i := 0; i < 6 && code == http.StatusOK; i++ {
		code, _, refused = upload(t, env.e, env.session, map[string]string{"topicId": topicID}, tinyPNG(t))
	}

	if code != http.StatusTooManyRequests || refused.Get("Retry-After") == "" {
		t.Fatalf("a flood of pictures: %d retry-after %q", code, refused.Get("Retry-After"))
	}
}
