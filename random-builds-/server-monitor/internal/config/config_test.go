package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	t.Setenv("DB_CONN_MAX_LIFETIME", "")
	t.Setenv("DB_PING_TIMEOUT", "")
	t.Setenv("SYSTEM_METRIC_INTERVAL", "")

	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.Addr)
	}
	if cfg.DBMaxOpenConns != 20 {
		t.Fatalf("DBMaxOpenConns = %d, want 20", cfg.DBMaxOpenConns)
	}
	if cfg.SystemMetricInterval != 2*time.Second {
		t.Fatalf("SystemMetricInterval = %s, want 2s", cfg.SystemMetricInterval)
	}
}

func TestLoadCustomValues(t *testing.T) {
	t.Setenv("APP_ADDR", ":9090")
	t.Setenv("DB_MAX_OPEN_CONNS", "40")
	t.Setenv("SYSTEM_METRIC_INTERVAL", "5s")

	cfg := Load()
	if cfg.Addr != ":9090" {
		t.Fatalf("Addr = %q, want :9090", cfg.Addr)
	}
	if cfg.DBMaxOpenConns != 40 {
		t.Fatalf("DBMaxOpenConns = %d, want 40", cfg.DBMaxOpenConns)
	}
	if cfg.SystemMetricInterval != 5*time.Second {
		t.Fatalf("SystemMetricInterval = %s, want 5s", cfg.SystemMetricInterval)
	}
}
