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

package system

import (
	"context"
	// NOTE: The CGO dependency for sdjournal can be problematic in minimal build environments.
	// We are using a dummy implementation to avoid this.
	// "github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// LogCollector defines the interface for components that collect system-level logs
// from sources like journald or syslog.
type LogCollector interface {
	// Collect retrieves a specified number of recent log entries from the system's log
	// manager. It can be filtered to a specific service or unit (e.g., a systemd unit).
	//
	// Parameters:
	//   ctx (context.Context): The context for the operation.
	//   lines (int): The maximum number of log lines to retrieve.
	//   unit (string): The name of the systemd unit to filter logs for. If empty, logs are collected system-wide.
	//
	// Returns:
	//   *models.LogData: A struct containing the collected log entries.
	//   error: An error if log collection fails.
	Collect(ctx context.Context, lines int, unit string) (*models.LogData, error)
}

// dummyLogCollector is a placeholder implementation that does nothing.
// This is used to avoid CGO dependencies on systems without systemd development headers.
type dummyLogCollector struct {
	log logger.Logger
}

// Collect is the dummy implementation of the LogCollector interface. It logs a
// warning that system log collection is disabled and returns an empty LogData struct.
// This approach avoids runtime errors in environments where systemd is not available
// or when CGO is disabled during the build.
func (c *dummyLogCollector) Collect(ctx context.Context, lines int, unit string) (*models.LogData, error) {
	c.log.Warn("System log collection is disabled because this build of KubeStack-AI was compiled without CGO or systemd headers.")
	c.log.Warn("Returning empty log data.")
	return &models.LogData{Entries: []string{}}, nil
}


// NewLogCollector creates a new system log collector. In the default build configuration,
// this function returns a `dummyLogCollector` to avoid introducing a CGO dependency
// for the systemd library. This ensures the application can be built and run easily
// on a wide range of systems. A full implementation for journald is available in the
// source code and can be enabled if required.
//
// Returns:
//   LogCollector: An instance of a log collector (dummy by default).
//   error: An error if the collector cannot be initialized (nil in the dummy implementation).
func NewLogCollector() (LogCollector, error) {
	return &dummyLogCollector{
		log: logger.NewLogger("dummy-log-collector"),
	}, nil
}

// The original journaldCollector implementation is preserved below for reference.
// To enable it, you would need to install systemd development headers on your build machine
// (e.g., `apt-get install libsystemd-dev` on Debian/Ubuntu) and uncomment the code.
/*
import (
	"fmt"
	"io"
	"time"
	"github.com/coreos/go-systemd/v22/sdjournal"
)
type journaldCollector struct {
	log logger.Logger
}
func newJournaldCollector() (LogCollector, error) {
	return &journaldCollector{
		log: logger.NewLogger("journald-collector"),
	}, nil
}
func (c *journaldCollector) Collect(ctx context.Context, lines int, unit string) (*models.LogData, error) {
	c.log.Infof("Collecting last %d journald logs for unit: '%s'", lines, unit)
	journal, err := sdjournal.NewJournal()
	if err != nil {
		return nil, fmt.Errorf("failed to open systemd journal (is systemd running?): %w", err)
	}
	defer journal.Close()
	if unit != "" {
		match := sdjournal.Match{
			Field: sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT,
			Value: unit,
		}
		if err := journal.AddMatch(match.String()); err != nil {
			return nil, fmt.Errorf("failed to add journal match for unit '%s': %w", err)
		}
	}
	if err := journal.SeekTail(); err != nil {
		return nil, fmt.Errorf("failed to seek to tail of journal: %w", err)
	}
	if _, err = journal.Previous(); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to step back from tail of journal: %w", err)
	}
	logEntries := make([]string, 0, lines)
	for i := 0; i < lines; i++ {
		select {
		case <-ctx.Done():
			c.log.Warn("Context cancelled during log collection.")
			return &models.LogData{Entries: logEntries}, ctx.Err()
		default:
			c, err := journal.Previous()
			if err != nil || c == 0 {
				break
			}
			entry, err := journal.GetEntry()
			if err != nil {
				c.log.Warnf("Failed to get journal entry: %v", err)
				continue
			}
			logLine := fmt.Sprintf("%s: %s", entry.Timestamp.Format(time.RFC3339), entry.Message)
			logEntries = append(logEntries, logLine)
		}
	}
	for i, j := 0, len(logEntries)-1; i < j; i, j = i+1, j-1 {
		logEntries[i], logEntries[j] = logEntries[j], logEntries[i]
	}
	return &models.LogData{Entries: logEntries}, nil
}
*/

//Personal.AI order the ending
