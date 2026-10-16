// Package secretbox seals short secrets for storage at rest with AES-256-GCM.
// The key is derived from the app secret with HKDF, so the same secret can keep
// signing sessions without the two uses sharing a key, and the additional data
// binds a ciphertext to the row it was stored in.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

const keyInfo = "learning_app secretbox v1"

var ErrOpen = errors.New("secretbox: cannot open")

type Box struct {
	aead cipher.AEAD
}

// New cannot fail for a 32-byte key, so the errors below are programming bugs.
func New(secret string) Box {
	key, err := hkdf.Key(sha256.New, []byte(secret), nil, keyInfo, 32)
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

	return Box{aead: aead}
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
