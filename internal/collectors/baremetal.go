package collectors

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/turtacn/kubestack-ai/internal/errors"
	"github.com/turtacn/kubestack-ai/internal/logging"
	"github.com/turtacn/kubestack-ai/internal/models"
)

// bareCollector 裸机采集实现。bareCollector implements Collector for bare-metal.
type bareCollector struct {
	serviceName string
}

// NewBareCollector 创建裸机采集器。NewBareCollector creates bare-metal collector.
func NewBareCollector(serviceName string) Collector {
	return &bareCollector{
		serviceName: serviceName,
	}
}

// GetInstanceStatus 获取进程状态。Get process status.
func (c *bareCollector) GetInstanceStatus(ctx context.Context) (string, error) {
	logging.Logger.Debugf("Checking status of service: %s", c.serviceName)

	// 检查服务状态。Check service status.
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", c.serviceName)
	output, err := cmd.Output()
	if err != nil {
		// 尝试检查进程是否存在。Try checking if process exists.
		cmd := exec.CommandContext(ctx, "pgrep", c.serviceName)
		if pgrepErr := cmd.Run(); pgrepErr != nil {
			logging.Logger.Errorf("Service %s is not running: %v", c.serviceName, err)
			return "inactive", nil
		}
		return "running (not managed by systemd)", nil
	}

	return strings.TrimSpace(string(output)), nil
}

// GetResourceUsage 获取资源使用。Get resource usage.
func (c *bareCollector) GetResourceUsage(ctx context.Context) (models.Metrics, error) {
	logging.Logger.Debugf("Getting resource usage for: %s", c.serviceName)

	// 使用ps获取CPU和内存使用情况。Use ps to get CPU and memory usage.
	cmd := exec.CommandContext(ctx, "ps", "-C", c.serviceName, "-o", "%cpu,%mem,rss,vsize")
	output, err := cmd.Output()
	if err != nil {
		logging.Logger.Errorf("Failed to get resource usage: %v", err)
		return nil, errors.ErrDataCollectionFailed
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return models.Metrics{"cpu_usage": 0, "memory_usage": 0}, nil
	}

	// 解析ps输出。Parse ps output.
	parts := strings.Fields(lines[1])
	if len(parts) < 2 {
		return models.Metrics{"cpu_usage": 0, "memory_usage": 0}, nil
	}

	cpu, _ := strconv.ParseFloat(parts[0], 64)
	mem, _ := strconv.ParseFloat(parts[1], 64)

	return models.Metrics{
		"cpu_usage_percent":    cpu,
		"memory_usage_percent": mem,
		"memory_rss_kb":        parts[2],
		"memory_vsize_kb":      parts[3],
	}, nil
}

// GetLogs 获取日志。Get logs.
func (c *bareCollector) GetLogs(ctx context.Context, since time.Duration) (models.Logs, error) {
	logging.Logger.Debugf("Getting logs for: %s", c.serviceName)

	// 尝试从journalctl获取日志。Try getting logs from journalctl.
	cmd := exec.CommandContext(ctx, "journalctl", "--unit", c.serviceName,
		"--since", strconv.FormatInt(int64(since.Seconds()), 10)+"s")
	output, err := cmd.Output()
	if err != nil {
		// 尝试从/var/log获取日志。Try getting logs from /var/log.
		logFile := "/var/log/" + c.serviceName + ".log"
		cmd := exec.CommandContext(ctx, "tail", "-n", "100", logFile)
		output, err = cmd.Output()
		if err != nil {
			logging.Logger.Warnf("Failed to get logs: %v", err)
			return models.Logs{}, nil
		}
	}

	logs := models.Logs{}
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}
		logs = append(logs, models.LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   line,
		})
	}

	return logs, nil
}

// GetNetworkInfo 获取网络信息。Get network information.
func (c *bareCollector) GetNetworkInfo(ctx context.Context) (models.Metrics, error) {
	// 获取监听端口。Get listening ports.
	cmd := exec.CommandContext(ctx, "netstat", "-tulpn")
	output, err := cmd.Output()
	if err != nil {
		logging.Logger.Warnf("Failed to get network info: %v", err)
		return models.Metrics{}, nil
	}

	// 过滤与服务相关的端口。Filter ports related to our service.
	lines := strings.Split(string(output), "\n")
	ports := []string{}
	for _, line := range lines {
		if strings.Contains(line, c.serviceName) {
			ports = append(ports, line)
		}
	}

	return models.Metrics{
		"listening_ports": ports,
		"hostname":        getHostname(),
	}, nil
}

// GetEvents 获取事件。Get events.
func (c *bareCollector) GetEvents(ctx context.Context, since time.Duration) ([]string, error) {
	// 从journalctl获取服务事件。Get service events from journalctl.
	cmd := exec.CommandContext(ctx, "journalctl", "--unit", c.serviceName,
		"--since", strconv.FormatInt(int64(since.Seconds()), 10)+"s", "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		logging.Logger.Warnf("Failed to get events: %v", err)
		return []string{}, nil
	}

	return strings.Split(string(output), "\n"), nil
}

// GetEnvironmentInfo 获取环境信息。Get environment information.
func (c *bareCollector) GetEnvironmentInfo(ctx context.Context) (models.Config, error) {
	// 获取系统信息。Get system information.
	cmd := exec.CommandContext(ctx, "uname", "-a")
	output, err := cmd.Output()
	if err != nil {
		logging.Logger.Warnf("Failed to get system info: %v", err)
		return models.Config{}, nil
	}

	// 获取CPU信息。Get CPU info.
	cpuCmd := exec.CommandContext(ctx, "grep", "^processor", "/proc/cpuinfo", "|", "wc", "-l")
	cpuOutput, err := cpuCmd.Output()
	if err != nil {
		logging.Logger.Warnf("Failed to get CPU info: %v", err)
		return models.Config{}, nil
	}

	return models.Config{
		"environment":  "baremetal",
		"system_info":  strings.TrimSpace(string(output)),
		"cpu_cores":    strings.TrimSpace(string(cpuOutput)),
		"service_name": c.serviceName,
	}, nil
}

// getHostname 获取主机名。Get hostname.
func getHostname() string {
	cmd := exec.Command("hostname")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

//Personal.AI order the ending
