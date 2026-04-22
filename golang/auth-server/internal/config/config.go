package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
// All values are read from environment variables.
// See .env.example for the full list.
type Config struct {
	// Server
	Port    string
	BaseURL string // e.g. https://auth.yourapp.com (no trailing slash)

	// Frontend (for redirects after social login, email links)
	FrontendURL string // e.g. https://yourapp.com

	// JWT
	// Tip: swap HS256 for RS256 in pkg/jwtutil by loading RSA keys here.
	JWTSecret        string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration

	// Email / Password
	BCryptCost int // 10–14; default 12

	// Social – Google
	GoogleClientID     string
	GoogleClientSecret string

	// Social – GitHub
	GitHubClientID     string
	GitHubClientSecret string

	// Email sending
	// Implement the email.Sender interface with your provider (SMTP, SendGrid, etc.)
	// The console sender is used by default (prints to stdout).
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	EmailFrom    string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		BaseURL:     getEnv("BASE_URL", "http://localhost:8080"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),

		JWTSecret:       requireEnv("JWT_SECRET"),
		AccessTokenTTL:  getDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: getDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),

		BCryptCost: getInt("BCRYPT_COST", 12),

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),

		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		EmailFrom:    getEnv("EMAIL_FROM", "no-reply@yourapp.com"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required environment variable not set: " + key)
	}
	return v
}

func getInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
