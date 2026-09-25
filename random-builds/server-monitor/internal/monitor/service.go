package monitor

import (
	"context"
	"time"
)

type Service struct {
	system   *SystemCollector
	database *DatabaseMonitor
}

func NewService(system *SystemCollector, database *DatabaseMonitor) *Service {
	return &Service{system: system, database: database}
}

func (s *Service) Snapshot(ctx context.Context) Snapshot {
	system := s.system.Latest()
	database := s.database.Snapshot(ctx)

	overall := "healthy"
	if system.Status == "warning" || system.Status == "degraded" {
		overall = "degraded"
	}
	if database.Configured && database.Status == "unhealthy" {
		overall = "unhealthy"
	}

	return Snapshot{
		OverallStatus: overall,
		ServerTime:    time.Now(),
		System:        system,
		Database:      database,
	}
}
