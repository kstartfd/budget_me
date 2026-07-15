package commands

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Telegram Mini App initData verification, per
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app:
// secret = HMAC_SHA256(key="WebAppData", data=botToken); the received `hash`
// must equal hex(HMAC_SHA256(secret, data_check_string)) where the check
// string is all other fields sorted as "key=value" and joined with "\n".

// ErrInvalidInitData covers every verification failure — the caller never
// learns which part failed (no oracle for forgers).
var ErrInvalidInitData = errors.New("invalid init data")

// initDataMaxAge bounds replay: Telegram stamps auth_date at open time.
const initDataMaxAge = 24 * time.Hour

// TelegramUser is the `user` payload inside initData.
type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

// VerifyInitData checks the signature and freshness of a raw initData query
// string and returns the authenticated Telegram user.
func VerifyInitData(initData, botToken string, now time.Time) (TelegramUser, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return TelegramUser{}, ErrInvalidInitData
	}

	gotHash := values.Get("hash")
	if gotHash == "" {
		return TelegramUser{}, ErrInvalidInitData
	}

	values.Del("hash")

	pairs := make([]string, 0, len(values))
	for key := range values {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, values.Get(key)))
	}

	sort.Strings(pairs)
	checkString := strings.Join(pairs, "\n")

	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))

	want := hex.EncodeToString(hmacSHA256(secret, []byte(checkString)))
	if !hmac.Equal([]byte(want), []byte(gotHash)) {
		return TelegramUser{}, ErrInvalidInitData
	}

	authDate, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil || now.Sub(time.Unix(authDate, 0)) > initDataMaxAge {
		return TelegramUser{}, ErrInvalidInitData
	}

	var user TelegramUser
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID == 0 {
		return TelegramUser{}, ErrInvalidInitData
	}

	return user, nil
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)

	return mac.Sum(nil)
}
