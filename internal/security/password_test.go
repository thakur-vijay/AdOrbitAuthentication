package security

import (
	"strings"
	"testing"
)

func TestHashPasswordAndVerifyPassword(t *testing.T) {
	password := "CorrectHorse9!"
	first, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	second, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() second error = %v", err)
	}
	if first == second {
		t.Fatal("HashPassword() produced identical hashes for independent salts")
	}
	if !strings.HasPrefix(first, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("hash does not encode the configured Argon2id parameters: %q", first)
	}

	for _, tc := range []struct {
		name     string
		password string
		want     bool
	}{
		{name: "correct password", password: password, want: true},
		{name: "incorrect password", password: "WrongHorse9!", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := VerifyPassword(tc.password, first)
			if err != nil {
				t.Fatalf("VerifyPassword() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("VerifyPassword() = %v, want %v", got, tc.want)
			}
		})
	}
	if strings.Contains(first, password) {
		t.Fatal("encoded hash contains the plaintext password")
	}
}

func TestVerifyPasswordRejectsMalformedAndOutOfRangeHashes(t *testing.T) {
	tests := []struct {
		name string
		hash string
	}{
		{name: "empty", hash: ""},
		{name: "wrong algorithm", hash: "$bcrypt$v=19$m=65536,t=3,p=2$c2FsdHNhbHQ$YWJjZGVmZ2hpamtsbW5vcA"},
		{name: "bad version", hash: "$argon2id$v=x$m=65536,t=3,p=2$c2FsdHNhbHQ$YWJjZGVmZ2hpamtsbW5vcA"},
		{name: "unsupported version", hash: "$argon2id$v=18$m=65536,t=3,p=2$c2FsdHNhbHQ$YWJjZGVmZ2hpamtsbW5vcA"},
		{name: "invalid parameters", hash: "$argon2id$v=19$m=not-a-number$c2FsdHNhbHQ$YWJjZGVmZ2hpamtsbW5vcA"},
		{name: "excessive memory", hash: "$argon2id$v=19$m=262145,t=3,p=2$c2FsdHNhbHQ$YWJjZGVmZ2hpamtsbW5vcA"},
		{name: "excessive time", hash: "$argon2id$v=19$m=8192,t=11,p=1$c2FsdHNhbHQ$YWJjZGVmZ2hpamtsbW5vcA"},
		{name: "empty salt", hash: "$argon2id$v=19$m=8192,t=1,p=1$$YWJjZGVmZ2hpamtsbW5vcA"},
		{name: "invalid salt encoding", hash: "$argon2id$v=19$m=8192,t=1,p=1$%%%$YWJjZGVmZ2hpamtsbW5vcA"},
		{name: "short hash", hash: "$argon2id$v=19$m=8192,t=1,p=1$c2FsdHNhbHQ$YWJjZGVmZ2hpamtsbW5vcA"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := VerifyPassword("password", tc.hash)
			if err == nil || ok {
				t.Fatalf("VerifyPassword() = (%v, %v), want (false, error)", ok, err)
			}
		})
	}
}

func BenchmarkHashPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := HashPassword("BenchmarkPass9!"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	hash, err := HashPassword("BenchmarkPass9!")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ok, err := VerifyPassword("BenchmarkPass9!", hash)
		if err != nil || !ok {
			b.Fatalf("VerifyPassword() = (%v, %v)", ok, err)
		}
	}
}
