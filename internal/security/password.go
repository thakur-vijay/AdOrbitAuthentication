package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 2
	argonKeyLen  uint32 = 32
	saltLength          = 16
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)

	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonTime,
		argonThreads,
		encodedSalt,
		encodedHash,
	), nil
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	var version, memory, time uint32
	var threads uint8
	var saltB64, hashB64 string

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("invalid password hash format")
	}

	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("invalid Argon2 version: %w", err)
	}

	if version != argon2.Version {
		return false, fmt.Errorf("unsupported Argon2 version: %d", version)
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, fmt.Errorf("invalid Argon2 parameters: %w", err)
	}

	if memory < 8*1024 || memory > 256*1024 ||
		time < 1 || time > 10 ||
		threads < 1 || threads > 16 {
		return false, fmt.Errorf("Argon2 parameters out of bounds")
	}

	saltB64 = parts[4]
	hashB64 = parts[5]

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil || len(salt) < 16 || len(salt) > 64 {
		return false, fmt.Errorf("invalid password salt")
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil || len(expectedHash) < 16 || len(expectedHash) > 64 {
		return false, fmt.Errorf("invalid password hash")
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		time,
		memory,
		threads,
		uint32(len(expectedHash)),
	)

	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1, nil
}
