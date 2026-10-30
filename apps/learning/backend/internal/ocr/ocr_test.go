package ocr

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"
)

const tsv = "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
	"1\t1\t0\t0\t0\t0\t0\t0\t900\t300\t-1\t\n" +
	"4\t1\t1\t1\t1\t0\t30\t30\t400\t50\t-1\t\n" +
	"5\t1\t1\t1\t1\t1\t30\t30\t200\t50\t96.5\tPizza\n" +
	"5\t1\t1\t1\t1\t2\t240\t30\t300\t50\t91\tmargherita\r\n" +
	"5\t1\t1\t2\t3\t1\t30\t90\t100\t50\t12.25\t  Hello \n" +
	"5\t1\t1\t2\t3\t2\t30\t90\t100\t50\t90\t\n" +
	"5\tnope\n"

func TestParseTSVTakesTheWordsWithWhereTheyStand(t *testing.T) {
	want := []Word{
		{Block: 1, Paragraph: 1, Line: 1, Text: "Pizza", Confidence: 96.5},
		{Block: 1, Paragraph: 1, Line: 1, Text: "margherita", Confidence: 91},
		{Block: 1, Paragraph: 2, Line: 3, Text: "Hello", Confidence: 12.25},
	}

	if got := parseTSV([]byte(tsv)); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLanguageOfKnowsTheTagsATopicCarries(t *testing.T) {
	for tag, want := range map[string]string{"en": "eng", "en-US": "eng", "ru": "rus", "uk": "ukr", "nb": "nor", "no": "nor", "pt-BR": "por"} {
		if got, ok := LanguageOf(tag); !ok || got.Tesseract != want {
			t.Errorf("%s: %v %v, want %s", tag, got, ok, want)
		}
	}

	for _, tag := range []string{"zh", "ja", "ar", "el", "xx", "", "not a tag"} {
		if got, ok := LanguageOf(tag); ok {
			t.Errorf("%q has no data in the image but reads as %v", tag, got)
		}
	}

	ru, _ := LanguageOf("ru")
	en, _ := LanguageOf("en")

	if !unicode.Is(ru.Script, 'я') || unicode.Is(ru.Script, 'z') || !unicode.Is(en.Script, 'z') {
		t.Error("the scripts of Russian and English are mixed up")
	}
}

// The image installs the data of OCR_LANGS and the code reads the ones in
// `languages`; a language in one and not the other fails only in production.
func TestLanguagesAreTheOnesTheImageInstalls(t *testing.T) {
	dockerfile, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "infra", "learning", "docker", "backend.Dockerfile"))
	if err != nil {
		t.Skipf("the Dockerfile is not here: %v", err)
	}

	match := regexp.MustCompile(`ARG OCR_LANGS="([^"]*)"`).FindSubmatch(dockerfile)
	if match == nil {
		t.Fatal("the Dockerfile has no OCR_LANGS")
	}

	installed := strings.Fields(string(match[1]))
	slices.Sort(installed)

	known := Codes()
	slices.Sort(known)

	if !slices.Equal(installed, known) {
		t.Fatalf("the image installs %v, the code reads %v", installed, known)
	}
}

// stand-in writes a program in place of tesseract: it keeps what it was run with
// and what it was fed, and runs body.
func standIn(t *testing.T, body string) (bin, args, input string) {
	t.Helper()

	dir := t.TempDir()
	bin, args, input = filepath.Join(dir, "tesseract"), filepath.Join(dir, "args"), filepath.Join(dir, "input")

	script := "#!/bin/sh\necho \"$@\" > " + args + "\ncat > " + input + "\n" + body + "\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	return bin, args, input
}

func TestReadFeedsThePictureAndAsksForTheLanguageAndLayout(t *testing.T) {
	bin, args, input := standIn(t, "printf '5\\t1\\t1\\t1\\t1\\t1\\t0\\t0\\t0\\t0\\t96\\tHello\\n'")
	english, _ := LanguageOf("en")

	words, err := New(bin).Read(context.Background(), []byte("picture bytes"), english, Sparse)
	if err != nil {
		t.Fatal(err)
	}

	if len(words) != 1 || words[0].Text != "Hello" {
		t.Errorf("words: %+v", words)
	}

	if got, _ := os.ReadFile(args); strings.TrimSpace(string(got)) != "stdin stdout -l eng --psm 11 tsv" {
		t.Errorf("run with %q", got)
	}

	if got, _ := os.ReadFile(input); string(got) != "picture bytes" {
		t.Errorf("fed %q", got)
	}
}

func TestReadTellsAMissingProgramFromAPictureItCouldNotRead(t *testing.T) {
	english, _ := LanguageOf("en")

	if _, err := New(filepath.Join(t.TempDir(), "absent")).Read(context.Background(), nil, english, Page); !errors.Is(err, ErrUnavailable) {
		t.Errorf("no program: %v", err)
	}

	bin, _, _ := standIn(t, "echo 'Error in pixReadStream: Unsupported image type' >&2; exit 1")

	_, err := New(bin).Read(context.Background(), nil, english, Page)
	if !errors.Is(err, ErrFailed) || !strings.Contains(err.Error(), "Unsupported image type") {
		t.Errorf("unreadable picture: %v", err)
	}
}

func TestReadStopsWhenTheCallerGivesUp(t *testing.T) {
	bin, _, _ := standIn(t, "exec sleep 30")
	english, _ := LanguageOf("en")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()

	if _, err := New(bin).Read(ctx, nil, english, Page); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v", err)
	}

	if time.Since(start) > 5*time.Second {
		t.Errorf("waited %s for a program that was told to stop", time.Since(start))
	}
}

func TestReadRunsNoMoreProgramsThanItIsAllowed(t *testing.T) {
	bin, _, _ := standIn(t, "exec sleep 30")
	english, _ := LanguageOf("en")
	reader := New(bin)

	for i := 0; i < maxParallel; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() { _, _ = reader.Read(ctx, nil, english, Page) }()
	}

	deadline := time.Now().Add(5 * time.Second)
	for len(reader.slots) < maxParallel && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	if _, err := reader.Read(ctx, nil, english, Page); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a third program started: %v", err)
	}
}

func TestReadsARealPicture(t *testing.T) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract is not installed")
	}

	picture, err := os.ReadFile(filepath.Join("testdata", "menu.png"))
	if err != nil {
		t.Fatal(err)
	}

	english, _ := LanguageOf("en")

	words, err := New("").Read(context.Background(), picture, english, Page)
	if err != nil {
		t.Fatal(err)
	}

	var texts []string

	for _, w := range words {
		if w.Confidence < 60 {
			t.Errorf("%q read with only %.0f%% sure", w.Text, w.Confidence)
		}

		texts = append(texts, w.Text)
	}

	for _, want := range []string{"Pizza", "margherita", "Spaghetti", "carbonara", "Hello", "friend!"} {
		if !slices.Contains(texts, want) {
			t.Errorf("%q is not among %q", want, texts)
		}
	}
}
