package config

import (
	"os"
	"time"
)

type Config struct {
	Port           string
	DatabaseURL    string
	ReservationTTL time.Duration
}

func Load() Config {
	return Config{
		Port:           getEnv("PORT", "8081"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/warehouse?sslmode=disable"),
		ReservationTTL: 5 * time.Minute,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
