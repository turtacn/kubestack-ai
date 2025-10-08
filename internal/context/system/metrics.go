// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package system provides components for collecting context from the underlying operating system.
package system

import (
	"context"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// MetricsCollector defines the interface for components that collect OS-level metrics,
// such as CPU, memory, disk, and network usage.
type MetricsCollector interface {
	// Collect gathers a snapshot of various OS-level metrics.
	//
	// Parameters:
	//   ctx (context.Context): The context for the collection operation.
	//
	// Returns:
	//   map[string]interface{}: A map of metric names to their collected values.
	//   error: An error if a critical part of the collection process fails.
	Collect(ctx context.Context) (map[string]interface{}, error)
}

// gopsutilCollector is a concrete implementation of MetricsCollector that uses
// the popular and cross-platform gopsutil library.
type gopsutilCollector struct {
	log logger.Logger
}

// NewMetricsCollector creates a new system metrics collector that uses the gopsutil
// library as its backend.
//
// Returns:
//   MetricsCollector: A new instance of a metrics collector.
//   error: An error if initialization fails (nil in this implementation).
func NewMetricsCollector() (MetricsCollector, error) {
	return &gopsutilCollector{
		log: logger.NewLogger("system-metrics"),
	}, nil
}

// Collect implements the MetricsCollector interface using the gopsutil library.
// It gathers a wide range of OS metrics, including CPU usage, load average,
// memory statistics, disk usage for the root partition, and network I/O counters.
// It is designed to be resilient, logging warnings for individual metric collection
// failures while still returning successfully with the metrics that were collected.
//
// Parameters:
//   ctx (context.Context): The context for the gopsutil library calls.
//
// Returns:
//   map[string]interface{}: A map containing the collected metrics.
//   error: An error is not expected in the current implementation, so it returns nil.
func (c *gopsutilCollector) Collect(ctx context.Context) (map[string]interface{}, error) {
	c.log.Info("Collecting system-level OS metrics.")
	metrics := make(map[string]interface{})

	// CPU Usage (as a percentage of total).
	// The first reading is a snapshot; subsequent readings over a duration are more accurate.
	cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		metrics["cpu_usage_percent"] = cpuPercent[0]
	} else {
		c.log.Warnf("Failed to collect CPU usage: %v", err)
	}

	// CPU Load Average.
	loadAvg, err := load.AvgWithContext(ctx)
	if err == nil {
		metrics["load_1m"] = loadAvg.Load1
		metrics["load_5m"] = loadAvg.Load5
		metrics["load_15m"] = loadAvg.Load15
	} else {
		c.log.Warnf("Failed to collect load average: %v", err)
	}

	// Memory Usage.
	vmStat, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		metrics["memory_total_bytes"] = vmStat.Total
		metrics["memory_available_bytes"] = vmStat.Available
		metrics["memory_used_bytes"] = vmStat.Used
		metrics["memory_used_percent"] = vmStat.UsedPercent
	} else {
		c.log.Warnf("Failed to collect virtual memory stats: %v", err)
	}

	// Disk Usage (for the root filesystem).
	// A more advanced collector would iterate over all mount points.
	diskUsage, err := disk.UsageWithContext(ctx, "/")
	if err == nil {
		metrics["disk_total_bytes"] = diskUsage.Total
		metrics["disk_free_bytes"] = diskUsage.Free
		metrics["disk_used_percent"] = diskUsage.UsedPercent
	} else {
		c.log.Warnf("Failed to collect disk usage for '/': %v", err)
	}

	// Network I/O (for all interfaces combined).
	netIO, err := net.IOCountersWithContext(ctx, false)
	if err == nil && len(netIO) > 0 {
		metrics["net_bytes_sent_total"] = netIO[0].BytesSent
		metrics["net_bytes_recv_total"] = netIO[0].BytesRecv
	} else {
		c.log.Warnf("Failed to collect network I/O stats: %v", err)
	}

	// TODO: Collect process lists (`process.Processes`), open files, listening ports (`net.Connections`), etc.

	return metrics, nil
}

//Personal.AI order the ending
