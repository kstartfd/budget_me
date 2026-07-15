package commands

import "testing"

func TestPasswordHashRoundtrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	tests := []struct {
		name     string
		encoded  string
		password string
		want     bool
	}{
		{"correct password", hash, "correct horse battery", true},
		{"wrong password", hash, "wrong", false},
		{"garbage encoded", "not-a-hash", "anything", false},
		{"wrong scheme", "bcrypt$10$x$y", "anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyPassword(tt.encoded, tt.password); got != tt.want {
				t.Fatalf("VerifyPassword = %v, want %v", got, tt.want)
			}
		})
	}

	// Same password must hash differently (random salt).
	again, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if again == hash {
		t.Fatal("two hashes of the same password are identical — salt is not random")
	}
}
