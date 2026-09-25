package monitor

import (
	"context"
	"database/sql"
	"time"
)

type DatabaseMonitor struct {
	db          *sql.DB
	configured  bool
	pingTimeout time.Duration
}

func NewDatabaseMonitor(db *sql.DB, configured bool, pingTimeout time.Duration) *DatabaseMonitor {
	if pingTimeout <= 0 {
		pingTimeout = 2 * time.Second
	}
	return &DatabaseMonitor{db: db, configured: configured, pingTimeout: pingTimeout}
}

func (m *DatabaseMonitor) Snapshot(ctx context.Context) DatabaseSnapshot {
	now := time.Now()
	if !m.configured || m.db == nil {
		return DatabaseSnapshot{
			Configured: false,
			Status:     "not_configured",
			CheckedAt:  now,
		}
	}

	stats := m.db.Stats()
	result := DatabaseSnapshot{
		Configured:         true,
		Status:             "healthy",
		Driver:             "pgx/postgresql",
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDurationMS:     float64(stats.WaitDuration.Microseconds()) / 1000,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
		CheckedAt:          now,
	}

	pingCtx, cancel := context.WithTimeout(ctx, m.pingTimeout)
	defer cancel()

	start := time.Now()
	if err := m.db.PingContext(pingCtx); err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
		result.LatencyMS = float64(time.Since(start).Microseconds()) / 1000
		return result
	}

	result.LatencyMS = float64(time.Since(start).Microseconds()) / 1000
	return result
}
