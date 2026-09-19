package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                   string
	DBDriver               string
	DBDSN                  string
	DBAutoMigrate          bool
	JWTSecret              string
	JWTTTL                 time.Duration
	CORSOrigins            []string
	TimelineStepMS         int64
	MaxCuesPerRun          int
	RateLimitPerMinute     int
	LogLevel               string
	GracefulShutdownPeriod time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:                   env("PORT", "8080"),
		DBDriver:               env("DB_DRIVER", "sqlite"),
		DBDSN:                  env("DB_DSN", "file:rigging-local?mode=memory&cache=shared"),
		DBAutoMigrate:          envBool("DB_AUTO_MIGRATE", true),
		JWTSecret:              env("JWT_SECRET", "local-stage-rigging-secret-at-least-32-bytes"),
		JWTTTL:                 time.Duration(envInt("JWT_TTL_MINUTES", 480)) * time.Minute,
		CORSOrigins:            split(env("CORS_ORIGINS", "http://localhost:18528")),
		TimelineStepMS:         int64(envInt("TIMELINE_STEP_MS", 100)),
		MaxCuesPerRun:          envInt("MAX_CUES_PER_RUN", 40),
		RateLimitPerMinute:     envInt("RATE_LIMIT_PER_MINUTE", 240),
		LogLevel:               env("LOG_LEVEL", "info"),
		GracefulShutdownPeriod: time.Duration(envInt("SHUTDOWN_TIMEOUT_SECONDS", 10)) * time.Second,
	}
	if cfg.DBDriver != "postgres" && cfg.DBDriver != "sqlite" {
		return Config{}, fmt.Errorf("DB_DRIVER must be postgres or sqlite, got %q", cfg.DBDriver)
	}
	if strings.TrimSpace(cfg.DBDSN) == "" {
		return Config{}, fmt.Errorf("DB_DSN is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	if cfg.TimelineStepMS < 20 || cfg.TimelineStepMS > 5000 {
		return Config{}, fmt.Errorf("TIMELINE_STEP_MS must be between 20 and 5000")
	}
	if cfg.MaxCuesPerRun < 1 || cfg.MaxCuesPerRun > 200 {
		return Config{}, fmt.Errorf("MAX_CUES_PER_RUN must be between 1 and 200")
	}
	if cfg.RateLimitPerMinute < 10 {
		return Config{}, fmt.Errorf("RATE_LIMIT_PER_MINUTE must be at least 10")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw := env(key, strconv.Itoa(fallback))
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	raw := env(key, strconv.FormatBool(fallback))
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func split(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
