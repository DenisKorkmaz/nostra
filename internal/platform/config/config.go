package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	Env             string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    envOr("NOSTRA_HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("NOSTRA_DATABASE_URL"),
		Env:         envOr("NOSTRA_ENV", "dev"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("NOSTRA_DATABASE_URL is required")
	}

	timeout, err := time.ParseDuration(envOr("NOSTRA_SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse NOSTRA_SHUTDOWN_TIMEOUT: %w", err)
	}
	cfg.ShutdownTimeout = timeout

	return cfg, nil
}

func (c Config) IsProduction() bool {
	return c.Env == "prod"
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
