package learning_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"learning_app/backend/internal/config"
	"learning_app/backend/internal/learning/commands"
)

const scanPath = "/api/v1/learning/ai/scan"

// picture is a small PNG; pictures of different shades are different files.
func picture(t *testing.T, shade uint8) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for x := 0; x < 2; x++ {
		img.Set(x, 0, color.RGBA{shade, shade, shade, 255})
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}

	return out.Bytes()
}

// hugePicture is a PNG whose header says it is 5000 x 5000: small on the wire,
// 25 million pixels once unpacked.
func hugePicture(t *testing.T) []byte {
	t.Helper()

	data := picture(t, 0)
	binary.BigEndian.PutUint32(data[16:], 5000)
	binary.BigEndian.PutUint32(data[20:], 5000)
	// The header chunk is checked by its checksum, over the type and the data.
	binary.BigEndian.PutUint32(data[29:], crc32.ChecksumIEEE(data[12:29]))

	return data
}

// table is what tesseract prints for lines of text it is sure of; every line
// stands in a paragraph of its own, like the entries of a menu.
func table(lines ...string) string {
	var out strings.Builder

	out.WriteString("level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n")

	for i, line := range lines {
		for j, word := range strings.Fields(line) {
			fmt.Fprintf(&out, "5\t1\t1\t%d\t1\t%d\t0\t0\t0\t0\t90.0\t%s\n", i+1, j+1, word)
		}
	}

	return out.String()
}

// standIn is a program in place of tesseract: it answers with `page` when asked
// to read a page and with `sparse` when asked to read scattered words, and
// writes down how it was run.
type standIn struct{ bin, log string }

func newStandIn(t *testing.T, page, sparse string) standIn {
	t.Helper()

	dir := t.TempDir()
	s := standIn{bin: filepath.Join(dir, "tesseract"), log: filepath.Join(dir, "runs")}

	for name, content := range map[string]string{"page.tsv": page, "sparse.tsv": sparse} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	script := "#!/bin/sh\necho \"$@\" >> " + s.log + "\ncat > /dev/null\n" +
		"case \"$*\" in\n*\"--psm 11\"*) cat " + filepath.Join(dir, "sparse.tsv") + " ;;\n*) cat " + filepath.Join(dir, "page.tsv") + " ;;\nesac\n"
	if err := os.WriteFile(s.bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	return s
}

func (s standIn) runs() []string {
	raw, _ := os.ReadFile(s.log)

	return strings.Fields(string(raw))
}

// scanEnv is a server that reads pictures with the stand-in.
func scanEnv(t *testing.T, page, sparse string) (aiEnv, standIn) {
	t.Helper()

	reader := newStandIn(t, page, sparse)

	return setupAIWith(t, func(cfg *config.Config) { cfg.OCRBin = reader.bin }), reader
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

type scanned struct {
	Items []string
	Known int
	Seen  bool
}

// scan sends a picture for a topic and expects it to be read; the items come
// back as "kind:text".
func scan(t *testing.T, env aiEnv, topicID string, file []byte) scanned {
	t.Helper()

	code, body, _ := upload(t, env.e, env.session, map[string]string{"topicId": topicID}, file)
	if code != http.StatusOK {
		t.Fatalf("scan: %d %v", code, body)
	}

	d := data(t, body)
	out := scanned{Items: []string{}, Known: int(d["known"].(float64)), Seen: d["seen"].(bool)}

	items, ok := d["items"].([]any)
	if !ok {
		t.Fatalf("items must be a list, even an empty one: %v", d)
	}

	for _, item := range items {
		m := item.(map[string]any)
		out.Items = append(out.Items, m["kind"].(string)+":"+m["text"].(string))
	}

	return out
}

var menu = table("Pizza margherita", "Spaghetti carbonara", "Hello", "12.50 EUR", "Good morning, my friend!")

func TestScanReadsThePictureOnTheServerAndNeverSendsItToTheAgent(t *testing.T) {
	env, reader := scanEnv(t, menu, "")
	topicID := topic(t, env.e, env.session)

	// The learner has an agent, and it must not be asked: the picture is read here.
	if code, _ := request(t, env.e, "PUT", settingsPath, env.session, choice("deepseek", "deepseek-flash", "off", "sk-deepseek-scan-key-1234"), ""); code != 200 {
		t.Fatal("save")
	}

	got := scan(t, env, topicID, picture(t, 10))
	want := []string{"phrase:Pizza margherita", "phrase:Spaghetti carbonara", "word:Hello", "phrase:Good morning, my friend"}

	if !reflect.DeepEqual(got.Items, want) || got.Known != 0 || got.Seen {
		t.Fatalf("got %+v, want %q", got, want)
	}

	if calls := env.provider.requests(); len(calls) != 0 {
		t.Fatalf("the picture reached the agent's provider: %d calls, first %.300s", len(calls), calls[0].Body)
	}

	if calls := env.agent.requests(); len(calls) != 0 {
		t.Fatalf("the agent was asked %d times", len(calls))
	}

	if runs := reader.runs(); !reflect.DeepEqual(runs, strings.Fields("stdin stdout -l eng --psm 3 tsv")) {
		t.Errorf("the reader ran as %q", runs)
	}
}

func TestScanNeedsNoAgentAtAll(t *testing.T) {
	env, _ := scanEnv(t, menu, "")
	topicID := topic(t, env.e, env.session)

	if got := scan(t, env, topicID, picture(t, 10)); len(got.Items) != 4 {
		t.Fatalf("without an agent: %+v", got)
	}
}

func TestScanOffersOnlyWhatTheTopicDoesNotHaveYet(t *testing.T) {
	env, _ := scanEnv(t, menu, "")
	topicID := topic(t, env.e, env.session)

	// Saved by hand, saved by the queue, with other case and spacing than the picture has.
	if code, body := request(t, env.e, "POST", "/api/v1/learning/entries", env.session, map[string]any{"topicId": topicID, "kind": "word", "text": "  hello ", "status": "inbox"}, ""); code != http.StatusCreated {
		t.Fatalf("entry: %d %v", code, body)
	}

	capture(t, env, topicID, map[string]any{"kind": "phrase", "text": "SPAGHETTI   carbonara"})

	got := scan(t, env, topicID, picture(t, 10))
	if want := []string{"phrase:Pizza margherita", "phrase:Good morning, my friend"}; !reflect.DeepEqual(got.Items, want) || got.Known != 2 {
		t.Fatalf("got %+v, want %q and 2 known", got, want)
	}

	// Another topic's entries do not count.
	other := topic(t, env.e, env.session)
	if got := scan(t, env, other, picture(t, 10)); len(got.Items) != 4 || got.Known != 0 {
		t.Errorf("another topic: %+v", got)
	}
}

func TestScanFollowsAnEntryWhoseTextWasEdited(t *testing.T) {
	env, _ := scanEnv(t, menu, "")
	topicID := topic(t, env.e, env.session)

	code, body := request(t, env.e, "POST", "/api/v1/learning/entries", env.session, map[string]any{"topicId": topicID, "kind": "word", "text": "Hello", "status": "inbox"}, "")
	if code != http.StatusCreated {
		t.Fatalf("entry: %d %v", code, body)
	}

	id := data(t, body)["id"].(string)

	if got := scan(t, env, topicID, picture(t, 10)); got.Known != 1 {
		t.Fatalf("before the edit: %+v", got)
	}

	if code, body := request(t, env.e, "PUT", "/api/v1/learning/entries/"+id, env.session, map[string]any{"topicId": topicID, "kind": "word", "text": "Goodbye", "status": "inbox"}, ""); code != http.StatusOK {
		t.Fatalf("edit: %d %v", code, body)
	}

	if got := scan(t, env, topicID, picture(t, 11)); got.Known != 0 || len(got.Items) != 4 {
		t.Errorf("the old text still counts as had: %+v", got)
	}
}

func TestScanRecognizesTheSamePictureSentAgain(t *testing.T) {
	env, _ := scanEnv(t, menu, "")
	topicID := topic(t, env.e, env.session)
	other := topic(t, env.e, env.session)

	if got := scan(t, env, topicID, picture(t, 10)); got.Seen {
		t.Fatalf("a new picture is called seen: %+v", got)
	}

	// The same file again is recognized; what it holds is still offered, since
	// the learner may not have kept all of it.
	again := scan(t, env, topicID, picture(t, 10))
	if !again.Seen || len(again.Items) != 4 {
		t.Errorf("the same picture: %+v", again)
	}

	if got := scan(t, env, topicID, picture(t, 11)); got.Seen {
		t.Errorf("another picture is called seen: %+v", got)
	}

	if got := scan(t, env, other, picture(t, 10)); got.Seen {
		t.Errorf("the picture was read for another topic: %+v", got)
	}
}

func TestScanRepeatsNothingWithinAPicture(t *testing.T) {
	env, _ := scanEnv(t, table("Hello", "hello!", "HELLO", "Hello there", "hello   there"), "")
	topicID := topic(t, env.e, env.session)

	if got := scan(t, env, topicID, picture(t, 10)); !reflect.DeepEqual(got.Items, []string{"word:Hello", "phrase:Hello there"}) {
		t.Fatalf("got %+v", got)
	}
}

func TestScanTriesScatteredWordsWhenThePageHoldsNone(t *testing.T) {
	env, reader := scanEnv(t, table("12.50"), table("STOP"))
	topicID := topic(t, env.e, env.session)

	if got := scan(t, env, topicID, picture(t, 10)); !reflect.DeepEqual(got.Items, []string{"word:STOP"}) {
		t.Fatalf("got %+v", got)
	}

	if want := strings.Fields("stdin stdout -l eng --psm 3 tsv stdin stdout -l eng --psm 11 tsv"); !reflect.DeepEqual(reader.runs(), want) {
		t.Errorf("the reader ran as %q", reader.runs())
	}
}

func TestScanSaysSoWhenThePictureHoldsNothing(t *testing.T) {
	env, _ := scanEnv(t, table("12.50"), table("|||"))
	topicID := topic(t, env.e, env.session)

	if got := scan(t, env, topicID, picture(t, 10)); len(got.Items) != 0 || got.Known != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestScanReadsInTheLanguageOfTheTopic(t *testing.T) {
	env, reader := scanEnv(t, table("Доброе утро", "Hello"), "")

	code, body := request(t, env.e, "POST", "/api/v1/learning/topics", env.session, map[string]any{"name": "Русский", "language": "ru", "translationLanguage": "en"}, "")
	if code != http.StatusCreated {
		t.Fatalf("topic: %d %v", code, body)
	}

	if got := scan(t, env, data(t, body)["id"].(string), picture(t, 10)); !reflect.DeepEqual(got.Items, []string{"phrase:Доброе утро"}) {
		t.Fatalf("got %+v", got)
	}

	if runs := reader.runs(); len(runs) < 4 || runs[3] != "rus" {
		t.Errorf("the reader ran as %q", runs)
	}
}

func TestScanSaysWhenTheLanguageCannotBeRead(t *testing.T) {
	env, reader := scanEnv(t, menu, "")

	code, body := request(t, env.e, "POST", "/api/v1/learning/topics", env.session, map[string]any{"name": "中文", "language": "zh", "translationLanguage": "ru"}, "")
	if code != http.StatusCreated {
		t.Fatalf("topic: %d %v", code, body)
	}

	if code, body, _ := upload(t, env.e, env.session, map[string]string{"topicId": data(t, body)["id"].(string)}, picture(t, 10)); code != http.StatusBadRequest {
		t.Errorf("no data for Chinese: %d %v", code, body)
	}

	if runs := reader.runs(); len(runs) != 0 {
		t.Errorf("the reader ran as %q", runs)
	}
}

func TestScanChecksTheRequestBeforeReading(t *testing.T) {
	env, reader := scanEnv(t, menu, "")
	topicID := topic(t, env.e, env.session)

	huge := append(picture(t, 10), make([]byte, 6<<20)...)

	truncated := picture(t, 10)[:20]

	for name, tc := range map[string]struct {
		fields map[string]string
		file   []byte
		want   int
	}{
		"no picture":                  {map[string]string{"topicId": topicID}, nil, http.StatusBadRequest},
		"no topic":                    {map[string]string{}, picture(t, 10), http.StatusBadRequest},
		"bad topic":                   {map[string]string{"topicId": "nope"}, picture(t, 10), http.StatusBadRequest},
		"someone else's topic":        {map[string]string{"topicId": "0198a7b2-0000-7000-8000-000000000000"}, picture(t, 10), http.StatusNotFound},
		"text called a picture":       {map[string]string{"topicId": topicID}, []byte("<html>not an image</html>"), http.StatusBadRequest},
		"a picture cut short":         {map[string]string{"topicId": topicID}, truncated, http.StatusBadRequest},
		"picture over the limit":      {map[string]string{"topicId": topicID}, huge, http.StatusBadRequest},
		"small file, enormous canvas": {map[string]string{"topicId": topicID}, hugePicture(t), http.StatusBadRequest},
	} {
		if code, body, _ := upload(t, env.e, env.session, tc.fields, tc.file); code != tc.want {
			t.Errorf("%s: %d, want %d: %v", name, code, tc.want, body)
		}
	}

	if runs := reader.runs(); len(runs) != 0 {
		t.Fatalf("the reader ran for refused requests: %q", runs)
	}
}

func TestScanTellsAPictureItCannotReadFromAServerThatCannotRead(t *testing.T) {
	env := setupAIWith(t, func(cfg *config.Config) { cfg.OCRBin = filepath.Join(t.TempDir(), "absent") })
	topicID := topic(t, env.e, env.session)

	if code, body, _ := upload(t, env.e, env.session, map[string]string{"topicId": topicID}, picture(t, 10)); code != http.StatusInternalServerError {
		t.Errorf("no reader installed: %d %v", code, body)
	}

	failing := filepath.Join(t.TempDir(), "tesseract")
	if err := os.WriteFile(failing, []byte("#!/bin/sh\ncat > /dev/null\necho 'Error in pixReadStream' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	env = setupAIWith(t, func(cfg *config.Config) { cfg.OCRBin = failing })
	topicID = topic(t, env.e, env.session)

	if code, body, _ := upload(t, env.e, env.session, map[string]string{"topicId": topicID}, picture(t, 10)); code != http.StatusBadRequest {
		t.Errorf("a picture the reader choked on: %d %v", code, body)
	}

	// A picture that could not be read is not a picture that was seen.
	env, _ = scanEnv(t, menu, "")
	topicID = topic(t, env.e, env.session)

	if got := scan(t, env, topicID, picture(t, 10)); got.Seen {
		t.Errorf("seen at once: %+v", got)
	}
}

func TestScanBelongsToTheSessionAndTheTopicsOwner(t *testing.T) {
	env, reader := scanEnv(t, menu, "")
	topicID := topic(t, env.e, env.session)
	_, extension := apiToken(t, env.e, env.session, "read_write", topicID)

	if code, _, _ := upload(t, env.e, extension, map[string]string{"topicId": topicID}, picture(t, 10)); code != http.StatusUnauthorized {
		t.Errorf("an extension token used the reader: %d", code)
	}

	if code, _, _ := upload(t, env.e, "", map[string]string{"topicId": topicID}, picture(t, 10)); code != http.StatusUnauthorized {
		t.Errorf("no credentials at all: %d", code)
	}

	if runs := reader.runs(); len(runs) != 0 {
		t.Fatalf("the reader ran for refused requests: %q", runs)
	}
}

func TestScanIsRateLimited(t *testing.T) {
	env, _ := scanEnv(t, menu, "")
	topicID := topic(t, env.e, env.session)

	var refused http.Header

	code := http.StatusOK

	for i := 0; i < 6 && code == http.StatusOK; i++ {
		code, _, refused = upload(t, env.e, env.session, map[string]string{"topicId": topicID}, picture(t, 10))
	}

	if code != http.StatusTooManyRequests || refused.Get("Retry-After") == "" {
		t.Fatalf("a flood of pictures: %d retry-after %q", code, refused.Get("Retry-After"))
	}
}

// The recipe in migration 0011 (copied here) hashes the old entries in SQL; the
// application hashes the new ones in Go. They must agree, or an old entry is not found.
func TestTextHashAgreesWithTheMigration(t *testing.T) {
	env := setupAI(t)

	for _, text := range []string{"Hello", "  Hello   World ", "Доброе УТРО", "Straße", "ÉCOLE", "tab\tseparated\nlines", "ice  cream cone"} {
		var sql []byte

		err := env.pool.QueryRow(context.Background(), `SELECT sha256(convert_to(lower(btrim(regexp_replace($1::text, '[\t\n\v\f\r \u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]+', ' ', 'g'))), 'UTF8'))`, text).Scan(&sql)
		if err != nil {
			t.Fatal(err)
		}

		if got := commands.TextHash(text); !bytes.Equal(got, sql) {
			t.Errorf("%q: Go %x, SQL %x", text, got, sql)
		}
	}
}

func TestScanReadsARealPictureWhenTesseractIsInstalled(t *testing.T) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract is not installed")
	}

	file, err := os.ReadFile(filepath.Join("..", "ocr", "testdata", "menu.png"))
	if err != nil {
		t.Fatal(err)
	}

	env := setupAI(t)
	topicID := topic(t, env.e, env.session)

	got := scan(t, env, topicID, file)
	want := []string{"phrase:Pizza margherita", "phrase:Spaghetti carbonara", "word:Hello", "phrase:Good morning, my friend"}

	if !reflect.DeepEqual(got.Items, want) {
		t.Fatalf("got %q, want %q", got.Items, want)
	}
}
