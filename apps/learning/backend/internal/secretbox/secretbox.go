// Package secretbox seals short secrets for storage at rest with AES-256-GCM.
// The key is derived from the app secret with HKDF, so the same secret can keep
// signing sessions without the two uses sharing a key, and the additional data
// binds a ciphertext to the row it was stored in.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

const (
	keyInfo         = "learning_app secretbox v1"
	fingerprintInfo = "learning_app fingerprint v1"
)

var ErrOpen = errors.New("secretbox: cannot open")

type Box struct {
	aead cipher.AEAD
	mac  []byte
}

// New cannot fail for a 32-byte key, so the errors below are programming bugs.
func New(secret string) Box {
	key, err := hkdf.Key(sha256.New, []byte(secret), nil, keyInfo, 32)
	if err != nil {
		panic(err)
	}

	mac, err := hkdf.Key(sha256.New, []byte(secret), nil, fingerprintInfo, 32)
	if err != nil {
		panic(err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	return Box{aead: aead, mac: mac}
}

// Fingerprint names a secret without revealing it: the same secret always gives
// the same name, and the name tells nothing about the secret. It lets the app
// recognise one API key used by several accounts without keeping the key (the
// stored copy is sealed per account, so two copies never compare equal).
// Changing the app secret renames every fingerprint.
func (b Box) Fingerprint(secret []byte) string {
	h := hmac.New(sha256.New, b.mac)
	h.Write(secret)

	return hex.EncodeToString(h.Sum(nil)[:16])
}

func (b Box) Seal(plaintext, aad []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return b.aead.Seal(nonce, nonce, plaintext, aad), nil
}

func (b Box) Open(sealed, aad []byte) ([]byte, error) {
	n := b.aead.NonceSize()
	if len(sealed) < n+b.aead.Overhead() {
		return nil, ErrOpen
	}

	plaintext, err := b.aead.Open(nil, sealed[:n], sealed[n:], aad)
	if err != nil {
		return nil, ErrOpen
	}

	return plaintext, nil
}
