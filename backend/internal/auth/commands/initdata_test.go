package commands

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"testing"
	"time"
)

const testBotToken = "12345:TEST-TOKEN"

// signInitData builds a correctly signed initData string the way a Telegram
// client would, so the verifier is tested against the real algorithm.
func signInitData(t *testing.T, botToken string, fields map[string]string) string {
	t.Helper()

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	// data_check_string wants sorted key=value lines
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	check := ""

	for i, k := range keys {
		if i > 0 {
			check += "\n"
		}

		check += fmt.Sprintf("%s=%s", k, fields[k])
	}

	secretMac := hmac.New(sha256.New, []byte("WebAppData"))
	secretMac.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secretMac.Sum(nil))
	mac.Write([]byte(check))
	hash := hex.EncodeToString(mac.Sum(nil))

	values := url.Values{}
	for k, v := range fields {
		values.Set(k, v)
	}

	values.Set("hash", hash)

	return values.Encode()
}

func TestVerifyInitData(t *testing.T) {
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	freshAuthDate := strconv.FormatInt(now.Add(-time.Hour).Unix(), 10)
	userJSON := `{"id":42,"first_name":"Artem","username":"artem","language_code":"ru"}`

	tests := []struct {
		name     string
		initData func() string
		wantErr  bool
		wantID   int64
	}{
		{
			name: "valid",
			initData: func() string {
				return signInitData(t, testBotToken, map[string]string{
					"auth_date": freshAuthDate,
					"query_id":  "AAF03QcE",
					"user":      userJSON,
				})
			},
			wantID: 42,
		},
		{
			name: "tampered user payload",
			initData: func() string {
				valid := signInitData(t, testBotToken, map[string]string{
					"auth_date": freshAuthDate,
					"user":      userJSON,
				})
				values, _ := url.ParseQuery(valid)
				values.Set("user", `{"id":999,"first_name":"Mallory"}`)

				return values.Encode()
			},
			wantErr: true,
		},
		{
			name: "signed with another bot token",
			initData: func() string {
				return signInitData(t, "999:OTHER", map[string]string{
					"auth_date": freshAuthDate,
					"user":      userJSON,
				})
			},
			wantErr: true,
		},
		{
			name: "stale auth_date",
			initData: func() string {
				return signInitData(t, testBotToken, map[string]string{
					"auth_date": strconv.FormatInt(now.Add(-25*time.Hour).Unix(), 10),
					"user":      userJSON,
				})
			},
			wantErr: true,
		},
		{
			name:     "missing hash",
			initData: func() string { return "auth_date=" + freshAuthDate },
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := VerifyInitData(tt.initData(), testBotToken, now)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got user %+v", user)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if user.ID != tt.wantID {
				t.Fatalf("user id = %d, want %d", user.ID, tt.wantID)
			}
		})
	}
}
