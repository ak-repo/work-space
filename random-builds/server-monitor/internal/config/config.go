package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr                 string
	DatabaseURL          string
	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxLifetime    time.Duration
	DBPingTimeout        time.Duration
	SystemMetricInterval time.Duration
}

func Load() Config {
	return Config{
		Addr:                 envString("APP_ADDR", ":8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		DBMaxOpenConns:       envInt("DB_MAX_OPEN_CONNS", 20),
		DBMaxIdleConns:       envInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime:    envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		DBPingTimeout:        envDuration("DB_PING_TIMEOUT", 2*time.Second),
		SystemMetricInterval: envDuration("SYSTEM_METRIC_INTERVAL", 2*time.Second),
	}
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
