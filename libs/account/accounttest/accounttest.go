// Package accounttest holds helpers for tests of the apps that sit on top of the
// account package.
package accounttest

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"budget_me/account"
)

// InitData builds the initData string a Telegram client would send for the user,
// correctly signed with the bot token.
func InitData(botToken string, user account.TelegramUser, authDate time.Time) string {
	payload, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}

	fields := map[string]string{
		"auth_date": strconv.FormatInt(authDate.Unix(), 10),
		"user":      string(payload),
	}

	pairs := make([]string, 0, len(fields))
	for k, v := range fields {
		pairs = append(pairs, k+"="+v)
	}

	sort.Strings(pairs)

	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))

	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(pairs, "\n")))

	values := url.Values{}
	for k, v := range fields {
		values.Set(k, v)
	}

	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))

	return values.Encode()
}
