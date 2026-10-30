// Package ocr reads the text off a picture with the tesseract program, on this
// machine: a picture never leaves it, and reading costs no tokens.
package ocr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/language"
)

const (
	// readTimeout: a photo takes a second or two, so this is a hang.
	readTimeout = 30 * time.Second
	// maxParallel keeps a few learners with a photo each from taking the whole machine.
	maxParallel = 2
)

var (
	// ErrUnavailable: tesseract is not installed where the server runs.
	ErrUnavailable = errors.New("ocr: tesseract is not installed")
	// ErrFailed: tesseract started and could not read this picture.
	ErrFailed = errors.New("ocr: the picture could not be read")
)

// Word is one word tesseract saw, with where it sits: words of a line share
// Block, Paragraph and Line, and Confidence is its own 0..100 estimate.
type Word struct {
	Block      int
	Paragraph  int
	Line       int
	Text       string
	Confidence float64
}

// Layout tells tesseract how the text is laid out on the picture; the values are
// its page segmentation modes.
type Layout int

const (
	// Page is a page of text in blocks: a book, a document, a screen.
	Page Layout = 3
	// Sparse is words scattered over the picture: a sign, a label, a menu board.
	Sparse Layout = 11
)

// Language is a language tesseract has data for in the server's image.
type Language struct {
	// Tesseract is its name for the language, the one its data file carries.
	Tesseract string
	// Script is the alphabet it is written in.
	Script *unicode.RangeTable
}

// languages must match OCR_LANGS in infra/learning/docker/backend.Dockerfile,
// which is what installs the data; a test keeps the two in step.
var languages = map[string]Language{
	"be": {"bel", unicode.Cyrillic},
	"cs": {"ces", unicode.Latin},
	"da": {"dan", unicode.Latin},
	"de": {"deu", unicode.Latin},
	"en": {"eng", unicode.Latin},
	"es": {"spa", unicode.Latin},
	"fi": {"fin", unicode.Latin},
	"fr": {"fra", unicode.Latin},
	"it": {"ita", unicode.Latin},
	"nl": {"nld", unicode.Latin},
	"no": {"nor", unicode.Latin},
	"pl": {"pol", unicode.Latin},
	"pt": {"por", unicode.Latin},
	"ru": {"rus", unicode.Cyrillic},
	"sv": {"swe", unicode.Latin},
	"tr": {"tur", unicode.Latin},
	"uk": {"ukr", unicode.Cyrillic},
}

// LanguageOf finds a topic's language (a tag such as "en-US") among the ones
// that can be read. False when there is no data for it.
func LanguageOf(tag string) (Language, bool) {
	parsed, err := language.Parse(tag)
	if err != nil {
		return Language{}, false
	}

	base, _ := parsed.Base()

	code := base.String()
	if code == "nb" || code == "nn" {
		code = "no"
	}

	found, ok := languages[code]

	return found, ok
}

// Codes lists the tesseract names of every language that can be read.
func Codes() []string {
	out := make([]string, 0, len(languages))
	for _, l := range languages {
		out = append(out, l.Tesseract)
	}

	return out
}

type Reader struct {
	bin   string
	slots chan struct{}
}

// New makes a reader that runs the tesseract found at bin; blank means the one
// on PATH. Tests point it at a stand-in.
func New(bin string) *Reader {
	if bin == "" {
		bin = "tesseract"
	}

	return &Reader{bin: bin, slots: make(chan struct{}, maxParallel)}
}

// Read recognizes the words of a JPEG, PNG or GIF file.
func (r *Reader) Read(ctx context.Context, image []byte, lang Language, layout Layout) ([]Word, error) {
	select {
	case r.slots <- struct{}{}:
		defer func() { <-r.slots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	runCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, r.bin, "stdin", "stdout", "-l", lang.Tesseract, "--psm", strconv.Itoa(int(layout)), "tsv")
	cmd.Stdin = bytes.NewReader(image)
	// Its own threads fight over the cores when several run at once.
	cmd.Env = append(os.Environ(), "OMP_THREAD_LIMIT=1")

	var out, stderr bytes.Buffer

	cmd.Stdout = &out
	cmd.Stderr = &stderr
	// A killed process must not hold the read open through a child of its own.
	cmd.WaitDelay = 2 * time.Second

	if err := cmd.Run(); err != nil {
		switch {
		case errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist):
			return nil, ErrUnavailable
		case ctx.Err() != nil:
			return nil, ctx.Err()
		case runCtx.Err() != nil:
			return nil, fmt.Errorf("%w: no answer in %s", ErrFailed, readTimeout)
		}

		return nil, fmt.Errorf("%w: %v: %s", ErrFailed, err, strings.TrimSpace(tail(stderr.String(), 400)))
	}

	return parseTSV(out.Bytes()), nil
}

// parseTSV picks the words (level 5) out of tesseract's table; the columns are
// level, page, block, paragraph, line, word, left, top, width, height, conf, text.
func parseTSV(out []byte) []Word {
	var words []Word

	for _, row := range strings.Split(string(out), "\n") {
		cols := strings.Split(strings.TrimRight(row, "\r"), "\t")
		if len(cols) < 12 || cols[0] != "5" {
			continue
		}

		text := strings.TrimSpace(cols[11])
		if text == "" {
			continue
		}

		block, blockErr := strconv.Atoi(cols[2])
		paragraph, paragraphErr := strconv.Atoi(cols[3])
		line, lineErr := strconv.Atoi(cols[4])
		confidence, confidenceErr := strconv.ParseFloat(cols[10], 64)

		if err := errors.Join(blockErr, paragraphErr, lineErr, confidenceErr); err != nil {
			continue
		}

		words = append(words, Word{Block: block, Paragraph: paragraph, Line: line, Text: text, Confidence: confidence})
	}

	return words
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}

	return s[len(s)-n:]
}
