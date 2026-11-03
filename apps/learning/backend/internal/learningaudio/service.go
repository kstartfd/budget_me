package learningaudio

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"learning_app/backend/internal/apperr"
	"learning_app/backend/internal/clock"
	"learning_app/backend/internal/db"
	"learning_app/backend/internal/ids"
)

type Provider interface {
	Speak(ctx context.Context, text, language string) ([]byte, string, error)
}

type Service struct {
	q        *db.Queries
	provider Provider
	root     string
	clock    clock.Clock
}

func NewService(q *db.Queries, provider Provider, root string, clk clock.Clock) Service {
	if root == "" {
		root = filepath.Join(os.TempDir(), "learning-audio")
	}

	return Service{q: q, provider: provider, root: root, clock: clk}
}

func (s Service) Ensure(ctx context.Context, language, text string) (db.LearningAudioCache, error) {
	language = NormalizeLanguage(language)
	text = strings.Join(strings.Fields(text), " ")

	if language == "" || text == "" || utf8.RuneCountInString(text) > 1000 {
		return db.LearningAudioCache{}, apperr.NewInvalid("audio text and language are required")
	}

	textHash := TextHash(text)

	row, err := s.q.GetLearningAudioCache(ctx, db.GetLearningAudioCacheParams{Language: language, TextHash: textHash})
	if err == nil {
		return row, nil
	}
	if !db.IsNotFound(err) {
		return row, err
	}

	audio, contentType, err := s.provider.Speak(ctx, text, language)
	if err != nil {
		var providerErr *ProviderError
		if errors.As(err, &providerErr) && (providerErr.StatusCode == 400 || providerErr.StatusCode == 404 || providerErr.StatusCode == 422) {
			return row, apperr.NewInvalid("audio is not available for this language")
		}

		return row, apperr.NewInternal("audio synthesis failed")
	}
	if len(audio) == 0 {
		return row, apperr.NewInternal("audio synthesis returned an empty response")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	contentHash := sha256.Sum256(audio)
	id := ids.New()
	storagePath := id.String() + ".mp3"

	if err := s.write(storagePath, audio); err != nil {
		return row, err
	}

	now := db.Time(s.clock.Now())
	row, err = s.q.InsertLearningAudioCache(ctx, db.InsertLearningAudioCacheParams{
		ID:          ids.ToPg(id),
		Language:    language,
		Text:        text,
		TextHash:    textHash,
		ContentHash: contentHash[:],
		StoragePath: storagePath,
		ContentType: contentType,
		ByteSize:    int64(len(audio)),
		Provider:    "google_translate",
		CreatedAt:   now,
	})
	if err != nil {
		return row, err
	}
	if row.StoragePath != storagePath {
		_ = os.Remove(filepath.Join(s.root, storagePath))
	}

	return row, nil
}

func (s Service) Open(_ context.Context, row db.LearningAudioCache) (*os.File, error) {
	if filepath.Base(row.StoragePath) != row.StoragePath {
		return nil, apperr.NewInternal("invalid audio storage path")
	}

	return os.Open(filepath.Join(s.root, row.StoragePath))
}

func (s Service) write(name string, data []byte) error {
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(s.root, ".audio-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, filepath.Join(s.root, name))
}

func TextHash(text string) []byte {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.Join(strings.Fields(text), " "))))

	return sum[:]
}
