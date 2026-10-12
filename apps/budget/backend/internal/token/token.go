// Package token issues and parses the stateless bearer tokens the API uses
// for sessions: HMAC-SHA256 over "v1.<user id>.<expiry unix>", standard
// library only. No server-side session state; revocation (if ever needed)
// means introducing a session table and is deliberately out of scope now.
package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidToken is returned for a malformed, tampered or expired token.
var ErrInvalidToken = errors.New("invalid token")

// Codec signs and verifies bearer tokens with a shared secret.
type Codec struct {
	secret []byte
}

// NewCodec builds a token codec signing with the given shared secret.
func NewCodec(secret string) *Codec {
	return &Codec{secret: []byte(secret)}
}

// Issue creates a token for the user that expires at now+ttl.
func (c *Codec) Issue(userID uuid.UUID, now time.Time, ttl time.Duration) string {
	payload := fmt.Sprintf("v1.%s.%d", userID, now.Add(ttl).Unix())
	return payload + "." + c.sign(payload)
}

// Parse verifies the signature and expiry and returns the user id.
func (c *Codec) Parse(tok string, now time.Time) (uuid.UUID, error) {
	parts := strings.Split(tok, ".")
	if len(parts) != 4 || parts[0] != "v1" {
		return uuid.Nil, ErrInvalidToken
	}

	payload := strings.Join(parts[:3], ".")
	if !hmac.Equal([]byte(c.sign(payload)), []byte(parts[3])) {
		return uuid.Nil, ErrInvalidToken
	}

	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || now.Unix() > exp {
		return uuid.Nil, ErrInvalidToken
	}

	id, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	return id, nil
}

func (c *Codec) sign(payload string) string {
	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(payload))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
