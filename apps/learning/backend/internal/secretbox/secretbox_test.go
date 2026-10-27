package secretbox_test

import (
	"bytes"
	"errors"
	"testing"

	"learning_app/backend/internal/secretbox"
)

func TestSealOpenRoundTrip(t *testing.T) {
	box := secretbox.New("a-long-enough-test-secret")
	aad := []byte("token-id")

	for _, plaintext := range []string{"", "bml_x", "bml_RyrQ7R8JSCp3k0e9k0a1yH4w4cC7bQ3sXz8b0aQe1Zs"} {
		sealed, err := box.Seal([]byte(plaintext), aad)
		if err != nil {
			t.Fatalf("seal %q: %v", plaintext, err)
		}

		if plaintext != "" && bytes.Contains(sealed, []byte(plaintext)) {
			t.Fatalf("ciphertext contains the plaintext %q", plaintext)
		}

		got, err := box.Open(sealed, aad)
		if err != nil || string(got) != plaintext {
			t.Fatalf("open %q: got %q, %v", plaintext, got, err)
		}
	}
}

func TestSealIsRandomized(t *testing.T) {
	box := secretbox.New("a-long-enough-test-secret")

	first, err := box.Seal([]byte("same"), nil)
	if err != nil {
		t.Fatal(err)
	}

	second, err := box.Seal([]byte("same"), nil)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(first, second) {
		t.Fatal("two seals of the same plaintext are identical: nonce is reused")
	}
}

func TestOpenRejectsForeignInput(t *testing.T) {
	box := secretbox.New("a-long-enough-test-secret")
	aad := []byte("token-id")

	sealed, err := box.Seal([]byte("bml_secret"), aad)
	if err != nil {
		t.Fatal(err)
	}

	tampered := bytes.Clone(sealed)
	tampered[len(tampered)-1] ^= 1

	cases := map[string]struct {
		box    secretbox.Box
		sealed []byte
		aad    []byte
	}{
		"another secret":  {secretbox.New("some-other-test-secret"), sealed, aad},
		"another row":     {box, sealed, []byte("other-token-id")},
		"no aad":          {box, sealed, nil},
		"tampered":        {box, tampered, aad},
		"truncated":       {box, sealed[:10], aad},
		"empty":           {box, nil, aad},
		"nonce only":      {box, sealed[:12], aad},
		"plaintext as is": {box, []byte("bml_secret"), aad},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := c.box.Open(c.sealed, c.aad)
			if !errors.Is(err, secretbox.ErrOpen) || got != nil {
				t.Fatalf("opened foreign input: %q, %v", got, err)
			}
		})
	}
}

func TestFingerprintNamesASecretWithoutRevealingIt(t *testing.T) {
	box := secretbox.New("a-long-enough-test-secret")
	key := []byte("sk-deepseek-1234567890")

	fp := box.Fingerprint(key)
	if fp != box.Fingerprint(key) || len(fp) != 32 {
		t.Fatalf("the same secret must always have the same 32-character name: %q", fp)
	}

	if bytes.Contains([]byte(fp), key) || fp == box.Fingerprint([]byte("sk-deepseek-1234567891")) {
		t.Fatalf("a fingerprint must differ per secret and not contain it: %q", fp)
	}

	if fp == secretbox.New("another-long-enough-secret").Fingerprint(key) {
		t.Fatal("a fingerprint must depend on the app secret, or it could be computed from a guessed key")
	}
}
