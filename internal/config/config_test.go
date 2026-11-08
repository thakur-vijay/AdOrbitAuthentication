package config

import (
	"strings"
	"testing"
	"time"
)

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"APP_PORT", "DATABASE_URL", "JWT_PRIVATE_KEY", "JWT_PUBLIC_KEY", "RABBITMQ_URL", "IDEMPOTENCY_SCOPE_KEY",
		"IDEMPOTENCY_ENCRYPTION_KEY", "OTP_HMAC_SECRET", "SWAGGER_HOST",
	} {
		t.Setenv(key, "")
	}
}

func TestIdempotencyConfigUsesConfiguredKeyringsAndDurations(t *testing.T) {
	cfg := &Config{
		IdempotencyScopeKey:      []byte(strings.Repeat("s", 32)),
		IdempotencyEncryptionKey: []byte(strings.Repeat("e", 32)),
	}
	got := cfg.IdempotencyConfig()
	if got.TTL != 24*time.Hour || got.ProcessingTTL != 2*time.Minute {
		t.Fatalf("idempotency durations = (%v, %v), want (24h, 2m)", got.TTL, got.ProcessingTTL)
	}
	if got.ActiveScopeHashKeyID != "scope-v1" || len(got.ScopeHashKeys) != 1 || string(got.ScopeHashKeys[0].Key) != string(cfg.IdempotencyScopeKey) {
		t.Fatalf("scope keyring = %#v", got.ScopeHashKeys)
	}
	if got.ActiveResponseKeyID != "response-v1" || len(got.ResponseEncryptionKeys) != 1 || string(got.ResponseEncryptionKeys[0].Key) != string(cfg.IdempotencyEncryptionKey) {
		t.Fatalf("response keyring = %#v", got.ResponseEncryptionKeys)
	}
}

func TestLoadValidatesRequiredKeysAndConfiguration(t *testing.T) {
	t.Run("missing scope key", func(t *testing.T) {
		clearConfigEnv(t)
		_, err := Load()
		if err == nil || !strings.Contains(err.Error(), "IDEMPOTENCY_SCOPE_KEY") {
			t.Fatalf("Load() error = %v, want missing scope key", err)
		}
	})

	t.Run("missing required service configuration", func(t *testing.T) {
		clearConfigEnv(t)
		setValidKeys(t)
		_, err := Load()
		if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
			t.Fatalf("Load() error = %v, want missing database URL", err)
		}
	})

	t.Run("loads defaults and decodes multiline keys", func(t *testing.T) {
		clearConfigEnv(t)
		setValidKeys(t)
		t.Setenv("DATABASE_URL", "postgres://test/db")
		t.Setenv("JWT_PRIVATE_KEY", "line-one\\nline-two")
		t.Setenv("JWT_PUBLIC_KEY", "public-key")
		t.Setenv("RABBITMQ_URL", "amqp://test")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Port != ":8080" {
			t.Fatalf("Load() defaults = port %q", cfg.Port)
		}
		if cfg.JWTPrivateKey != "line-one\nline-two" {
			t.Fatalf("JWT private key was not expanded: %q", cfg.JWTPrivateKey)
		}
		if len(cfg.IdempotencyScopeKey) != 32 || len(cfg.IdempotencyEncryptionKey) != 32 || len(cfg.OTP_HMAC_SECRET) != 32 {
			t.Fatalf("decoded key lengths = %d, %d, %d; want 32 bytes each", len(cfg.IdempotencyScopeKey), len(cfg.IdempotencyEncryptionKey), len(cfg.OTP_HMAC_SECRET))
		}
	})

	t.Run("normalizes port", func(t *testing.T) {
		clearConfigEnv(t)
		setValidKeys(t)
		t.Setenv("DATABASE_URL", "postgres://test/db")
		t.Setenv("JWT_PRIVATE_KEY", "private")
		t.Setenv("JWT_PUBLIC_KEY", "public")
		t.Setenv("RABBITMQ_URL", "amqp://test")
		t.Setenv("APP_PORT", "8443")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Port != ":8443" {
			t.Fatalf("Port = %q, want :8443", cfg.Port)
		}
	})
}

func setValidKeys(t *testing.T) {
	t.Helper()
	key := strings.Repeat("ab", 32)
	t.Setenv("IDEMPOTENCY_SCOPE_KEY", key)
	t.Setenv("IDEMPOTENCY_ENCRYPTION_KEY", key)
	t.Setenv("OTP_HMAC_SECRET", key)
}
