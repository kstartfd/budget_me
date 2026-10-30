package commands

import (
	"context"
	"crypto/sha256"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
	learningcommands "learning_app/backend/internal/learning/commands"
	"learning_app/backend/internal/ocr"
)

const (
	// scanTimeout covers both ways of reading a picture.
	scanTimeout = 60 * time.Second
	maxFound    = 30
)

// Scan reads the words and phrases off a picture for a topic.
type Scan struct {
	UserID  uuid.UUID
	TopicID uuid.UUID
	Image   []byte
}

// ScanResult is what a picture held, for the learner to pick from.
type ScanResult struct {
	Items []Found
	// Known is how many of the texts found are already in the topic, and so
	// are not offered again.
	Known int
	// Seen: this very picture was read for this topic before.
	Seen bool
}

type ScanHandler struct {
	q      *db.Queries
	clock  clock.Clock
	reader *ocr.Reader
}

func NewScanHandler(q *db.Queries, clk clock.Clock, reader *ocr.Reader) ScanHandler {
	return ScanHandler{q: q, clock: clk, reader: reader}
}

// Handle reads the picture here, with no AI agent and no tokens, and offers the
// texts the topic does not have yet. Nothing but the picture's hash is stored:
// the learner picks what to keep, and the pick becomes ordinary captures.
func (h ScanHandler) Handle(ctx context.Context, cmd Scan) (ScanResult, error) {
	userID, topicID := ids.ToPg(cmd.UserID), ids.ToPg(cmd.TopicID)

	topic, err := h.q.GetLearningTopic(ctx, db.GetLearningTopicParams{ID: topicID, UserID: userID})
	if db.IsNotFound(err) {
		return ScanResult{}, apperr.NewNotFound("topic not found")
	}

	if err != nil {
		return ScanResult{}, err
	}

	lang, ok := ocr.LanguageOf(topic.Language)
	if !ok {
		return ScanResult{}, apperr.NewInvalid("reading a picture is not available for the language of this topic yet")
	}

	ctx, cancel := context.WithTimeout(ctx, scanTimeout)
	defer cancel()

	found, err := h.read(ctx, cmd.Image, lang)

	switch {
	case errors.Is(err, ocr.ErrUnavailable):
		slog.Error("learning picture reader is not installed", "err", err)

		return ScanResult{}, apperr.NewInternal("reading a picture is not available on this server")
	case errors.Is(err, ocr.ErrFailed):
		slog.Warn("learning picture could not be read", "user_id", cmd.UserID, "err", err)

		return ScanResult{}, apperr.NewInvalid("the picture could not be read: try another one")
	case err != nil:
		return ScanResult{}, err
	}

	items, err := h.unknown(ctx, cmd, found)
	if err != nil {
		return ScanResult{}, err
	}

	sum := sha256.Sum256(cmd.Image)

	fresh, err := h.q.RecordLearningScan(ctx, db.RecordLearningScanParams{UserID: userID, TopicID: topicID, ImageHash: sum[:], CreatedAt: db.Time(h.clock.Now())})
	if err != nil {
		return ScanResult{}, err
	}

	result := ScanResult{Items: items, Known: len(found) - len(items), Seen: fresh == 0}
	if len(result.Items) > maxFound {
		result.Items = result.Items[:maxFound]
	}

	return result, nil
}

// read tries the picture as a page of text first and, when that finds nothing,
// as words scattered over it, like a sign.
func (h ScanHandler) read(ctx context.Context, image []byte, lang ocr.Language) ([]Found, error) {
	for _, layout := range []ocr.Layout{ocr.Page, ocr.Sparse} {
		words, err := h.reader.Read(ctx, image, lang, layout)
		if err != nil {
			return nil, err
		}

		if found := phrases(words, lang.Script); len(found) > 0 {
			return found, nil
		}
	}

	return nil, nil
}

// unknown is what the topic does not have yet: a text counts as had when an
// entry of the topic has the same hash.
func (h ScanHandler) unknown(ctx context.Context, cmd Scan, found []Found) ([]Found, error) {
	if len(found) == 0 {
		return []Found{}, nil
	}

	hashes := make([][]byte, len(found))
	for i, f := range found {
		hashes[i] = learningcommands.TextHash(f.Text)
	}

	have, err := h.q.KnownLearningTextHashes(ctx, db.KnownLearningTextHashesParams{UserID: ids.ToPg(cmd.UserID), TopicID: ids.ToPg(cmd.TopicID), Hashes: hashes})
	if err != nil {
		return nil, err
	}

	known := make(map[string]bool, len(have))
	for _, hash := range have {
		known[string(hash)] = true
	}

	items := make([]Found, 0, len(found))

	for i, f := range found {
		if !known[string(hashes[i])] {
			items = append(items, f)
		}
	}

	return items, nil
}
