package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	Port          string
	DatabaseURL   string
	JWTPrivateKey string
	JWTPublicKey  string
}

func Load() (*Config, error) {
	port := normalizePort(getEnv("APP_PORT", "8080"))

	cfg := &Config{
		Port:          port,
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		JWTPrivateKey: strings.ReplaceAll(os.Getenv("JWT_PRIVATE_KEY"), `\n`, "\n"),
		JWTPublicKey:  strings.ReplaceAll(os.Getenv("JWT_PUBLIC_KEY"), `\n`, "\n"),
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

	return cfg, nil
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
