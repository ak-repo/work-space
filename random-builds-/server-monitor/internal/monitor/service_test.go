package monitor

import (
	"context"
	"testing"
	"time"
)

func TestDatabaseNotConfigured(t *testing.T) {
	m := NewDatabaseMonitor(nil, false, time.Second)
	got := m.Snapshot(context.Background())
	if got.Configured {
		t.Fatal("Configured = true, want false")
	}
	if got.Status != "not_configured" {
		t.Fatalf("Status = %q, want not_configured", got.Status)
	}
}
