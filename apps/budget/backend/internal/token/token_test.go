package token

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCodecRoundtrip(t *testing.T) {
	codec := NewCodec("test-secret")
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	userID := uuid.MustParse("019807b4-0000-7000-8000-000000000042")

	tok := codec.Issue(userID, now, time.Hour)

	tests := []struct {
		name    string
		token   string
		at      time.Time
		codec   *Codec
		wantErr bool
	}{
		{"valid", tok, now.Add(30 * time.Minute), codec, false},
		{"expired", tok, now.Add(2 * time.Hour), codec, true},
		{"wrong secret", tok, now, NewCodec("other-secret"), true},
		{"tampered payload", strings.Replace(tok, "v1.", "v1x.", 1), now, codec, true},
		{"garbage", "abc", now, codec, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.codec.Parse(tt.token, tt.at)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got id %s", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != userID {
				t.Fatalf("user id = %s, want %s", got, userID)
			}
		})
	}
}
