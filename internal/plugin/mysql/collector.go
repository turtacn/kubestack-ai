package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
)

// MetricsCollector MySQL metrics collector
type MetricsCollector struct {
	db *sql.DB
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

func (c *MetricsCollector) SetDB(db *sql.DB) {
	c.db = db
}

func (c *MetricsCollector) Collect(ctx context.Context) (*plugin.MetricsSnapshot, error) {
	snapshot := &plugin.MetricsSnapshot{
		Timestamp: time.Now(),
		Metrics:   make(map[string]plugin.MetricValue),
		RawData:   make(map[string]interface{}),
	}

	rows, err := c.db.QueryContext(ctx, "SHOW GLOBAL STATUS")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statusMap := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err == nil {
			statusMap[name] = value
			if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				snapshot.Metrics[name] = plugin.MetricValue{
					Name:      name,
					Value:     floatVal,
					Timestamp: snapshot.Timestamp,
				}
			}
		}
	}
	snapshot.RawData["status"] = statusMap

	return snapshot, nil
}

func (c *MetricsCollector) CollectSpecific(ctx context.Context, metricName string) (interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *MetricsCollector) GetVariables(ctx context.Context) map[string]interface{} {
	vars := make(map[string]interface{})
	rows, err := c.db.QueryContext(ctx, "SHOW GLOBAL VARIABLES")
	if err != nil {
		return vars
	}
	defer rows.Close()

	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err == nil {
			// Try parse float for calculations
			if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				vars[name] = floatVal
			} else {
				vars[name] = value
			}
		}
	}
	return vars
}

func (c *MetricsCollector) GetProcessList(ctx context.Context) []plugin.ConnectionInfo {
	var conns []plugin.ConnectionInfo
	rows, err := c.db.QueryContext(ctx, "SHOW PROCESSLIST")
	if err != nil {
		return conns
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var user, host, db, command, state, info sql.NullString
		var timeVal int64
		if err := rows.Scan(&id, &user, &host, &db, &command, &timeVal, &state, &info); err == nil {
			conns = append(conns, plugin.ConnectionInfo{
				ID:       fmt.Sprintf("%d", id),
				User:     user.String,
				ClientIP: host.String,
				Database: db.String,
				Command:  command.String,
				Time:     timeVal,
				State:    state.String,
				Info:     info.String,
			})
		}
	}
	return conns
}
