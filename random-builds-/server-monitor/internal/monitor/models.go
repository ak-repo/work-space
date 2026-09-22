package monitor

import "time"

type SystemSnapshot struct {
	CollectedAt time.Time      `json:"collected_at"`
	Status      string         `json:"status"`
	Hostname    string         `json:"hostname"`
	OS          string         `json:"os"`
	Platform    string         `json:"platform"`
	Kernel      string         `json:"kernel"`
	UptimeSec   uint64         `json:"uptime_seconds"`
	Load        LoadMetrics    `json:"load"`
	CPU         CPUMetrics     `json:"cpu"`
	Memory      MemoryMetrics  `json:"memory"`
	Disk        DiskMetrics    `json:"disk"`
	Network     NetworkMetrics `json:"network"`
	Go          GoMetrics      `json:"go"`
	Error       string         `json:"error,omitempty"`
}

type LoadMetrics struct {
	Load1  float64 `json:"load_1"`
	Load5  float64 `json:"load_5"`
	Load15 float64 `json:"load_15"`
}

type CPUMetrics struct {
	UsagePercent float64 `json:"usage_percent"`
	LogicalCores int     `json:"logical_cores"`
}

type MemoryMetrics struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	Available    uint64  `json:"available_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type DiskMetrics struct {
	Path         string  `json:"path"`
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

type NetworkMetrics struct {
	BytesRecv uint64 `json:"bytes_recv"`
	BytesSent uint64 `json:"bytes_sent"`
}

type GoMetrics struct {
	Version       string `json:"version"`
	Goroutines    int    `json:"goroutines"`
	HeapAlloc     uint64 `json:"heap_alloc_bytes"`
	HeapInUse     uint64 `json:"heap_in_use_bytes"`
	Sys           uint64 `json:"sys_bytes"`
	GCCount       uint32 `json:"gc_count"`
	LastGCPauseNS uint64 `json:"last_gc_pause_ns"`
}

type DatabaseSnapshot struct {
	Configured         bool          `json:"configured"`
	Status             string        `json:"status"`
	LatencyMS          float64       `json:"latency_ms"`
	Driver             string        `json:"driver,omitempty"`
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUse              int           `json:"in_use"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"wait_count"`
	WaitDurationMS     float64       `json:"wait_duration_ms"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
	CheckedAt          time.Time     `json:"checked_at"`
	Error              string        `json:"error,omitempty"`
	PingTimeout        time.Duration `json:"-"`
}

type Snapshot struct {
	OverallStatus string           `json:"overall_status"`
	ServerTime    time.Time        `json:"server_time"`
	System        SystemSnapshot   `json:"system"`
	Database      DatabaseSnapshot `json:"database"`
}
