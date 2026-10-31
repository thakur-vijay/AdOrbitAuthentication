package config

import (
	"errors"
	"os"
	"strings"

	"github.com/thakur-vijay/adorbit-go/idempotency"
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTPrivateKey    string
	JWTPublicKey     string
	UsersGRPCAddress string
	RabbitMQURL      string

	IdempotencyScopeKey      []byte
	IdempotencyEncryptionKey []byte
}

func Load() (*Config, error) {
	port := normalizePort(getEnv("APP_PORT", "8080"))

	cfg := &Config{
		Port:             port,
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		JWTPrivateKey:    strings.ReplaceAll(os.Getenv("JWT_PRIVATE_KEY"), `\n`, "\n"),
		JWTPublicKey:     strings.ReplaceAll(os.Getenv("JWT_PUBLIC_KEY"), `\n`, "\n"),
		UsersGRPCAddress: getEnv("USERS_GRPC_ADDRESS", "localhost:9090"),
		RabbitMQURL:      getEnv("RABBITMQ_URL", ""),

		IdempotencyScopeKey:      []byte(os.Getenv("IDEMPOTENCY_SCOPE_KEY")),
		IdempotencyEncryptionKey: []byte(os.Getenv("IDEMPOTENCY_ENCRYPTION_KEY")),
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}

	if cfg.JWTPrivateKey == "" {
		return nil, errors.New("JWT_PRIVATE_KEY is not set")
	}

	if cfg.JWTPublicKey == "" {
		return nil, errors.New("JWT_PUBLIC_KEY is not set")
	}

	if cfg.RabbitMQURL == "" {
		return nil, errors.New("RABBITMQ_URL is not set")
	}

	if len(cfg.IdempotencyScopeKey) != 32 {
		return nil, errors.New("IDEMPOTENCY_SCOPE_KEY must be exactly 32 bytes")
	}

	if len(cfg.IdempotencyEncryptionKey) != 32 {
		return nil, errors.New("IDEMPOTENCY_ENCRYPTION_KEY must be exactly 32 bytes")
	}

	return cfg, nil
}

func (c *Config) IdempotencyConfig() idempotency.Config {
	return idempotency.Config{
		TTL:           24 * 60 * 60 * 1e9,
		ProcessingTTL: 2 * 60 * 1e9,

		ScopeHashKeys: []idempotency.KeyMaterial{
			{
				ID:  "scope-v1",
				Key: c.IdempotencyScopeKey,
			},
		},
		ActiveScopeHashKeyID: "scope-v1",

		ResponseEncryptionKeys: []idempotency.KeyMaterial{
			{
				ID:  "response-v1",
				Key: c.IdempotencyEncryptionKey,
			},
		},
		ActiveResponseKeyID: "response-v1",
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func normalizePort(port string) string {
	if port == "" {
		return ":8080"
	}

	if strings.Contains(port, ":") {
		return port
	}

	return ":" + port
}
