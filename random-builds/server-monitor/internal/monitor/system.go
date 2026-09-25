package monitor

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

type SystemCollector struct {
	interval time.Duration

	mu       sync.RWMutex
	latest   SystemSnapshot
	started  bool
	stopOnce sync.Once
}

func NewSystemCollector(interval time.Duration) *SystemCollector {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &SystemCollector{interval: interval}
}

func (c *SystemCollector) Start(ctx context.Context) {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return
	}
	c.started = true
	c.mu.Unlock()

	c.collect(ctx)

	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.collect(ctx)
			}
		}
	}()
}

func (c *SystemCollector) Latest() SystemSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latest
}

func (c *SystemCollector) collect(ctx context.Context) {
	snapshot := SystemSnapshot{
		CollectedAt: time.Now(),
		Status:      "healthy",
	}

	hostInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		snapshot.Status = "degraded"
		snapshot.Error = err.Error()
	} else {
		snapshot.Hostname = hostInfo.Hostname
		snapshot.OS = hostInfo.OS
		snapshot.Platform = hostInfo.Platform
		snapshot.Kernel = hostInfo.KernelVersion
		snapshot.UptimeSec = hostInfo.Uptime
	}

	if values, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(values) > 0 {
		snapshot.CPU.UsagePercent = values[0]
	} else if err != nil && snapshot.Error == "" {
		snapshot.Error = err.Error()
	}
	if cores, err := cpu.CountsWithContext(ctx, true); err == nil {
		snapshot.CPU.LogicalCores = cores
	}

	if memory, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		snapshot.Memory = MemoryMetrics{
			TotalBytes:   memory.Total,
			UsedBytes:    memory.Used,
			Available:    memory.Available,
			UsagePercent: memory.UsedPercent,
		}
	}

	if usage, err := disk.UsageWithContext(ctx, "/"); err == nil {
		snapshot.Disk = DiskMetrics{
			Path:         "/",
			TotalBytes:   usage.Total,
			UsedBytes:    usage.Used,
			FreeBytes:    usage.Free,
			UsagePercent: usage.UsedPercent,
		}
	}

	if average, err := load.AvgWithContext(ctx); err == nil {
		snapshot.Load = LoadMetrics{Load1: average.Load1, Load5: average.Load5, Load15: average.Load15}
	}

	if counters, err := net.IOCountersWithContext(ctx, false); err == nil && len(counters) > 0 {
		snapshot.Network = NetworkMetrics{BytesRecv: counters[0].BytesRecv, BytesSent: counters[0].BytesSent}
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	var pause uint64
	if memStats.NumGC > 0 {
		pause = memStats.PauseNs[(memStats.NumGC+255)%256]
	}
	snapshot.Go = GoMetrics{
		Version:       runtime.Version(),
		Goroutines:    runtime.NumGoroutine(),
		HeapAlloc:     memStats.HeapAlloc,
		HeapInUse:     memStats.HeapInuse,
		Sys:           memStats.Sys,
		GCCount:       memStats.NumGC,
		LastGCPauseNS: pause,
	}

	if snapshot.CPU.UsagePercent >= 90 || snapshot.Memory.UsagePercent >= 90 || snapshot.Disk.UsagePercent >= 95 {
		snapshot.Status = "warning"
	}

	c.mu.Lock()
	c.latest = snapshot
	c.mu.Unlock()
}
