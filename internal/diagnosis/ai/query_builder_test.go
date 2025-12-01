package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSearchQuery_RedisLog(t *testing.T) {
	qb := NewQueryBuilder()
	logs := `2023-10-27 10:00:00 INFO starting redis
2023-10-27 10:00:01 ERROR Can't save in background: fork: Cannot allocate memory
2023-10-27 10:00:02 INFO ready`
	metrics := ""
	mwType := "Redis"

	query := qb.BuildSearchQuery(logs, metrics, mwType)

	assert.Contains(t, query, "Redis")
	assert.Contains(t, query, "Can't save in background")
	assert.Contains(t, query, "Cannot allocate memory")
}

func TestBuildSearchQuery_MysqlMetric(t *testing.T) {
	qb := NewQueryBuilder()
	logs := ""
	metrics := "Threads_connected > max_connections (1000 > 500)"
	mwType := "MySQL"

	query := qb.BuildSearchQuery(logs, metrics, mwType)

	assert.Contains(t, query, "MySQL")
	assert.Contains(t, query, "Threads_connected")
}

func TestBuildSearchQuery_Complex(t *testing.T) {
	qb := NewQueryBuilder()
	logs := `[ERROR] replication link down: master_link_status: down`
	metrics := "replication_lag: 3600s (High)"
	mwType := "Redis"

	query := qb.BuildSearchQuery(logs, metrics, mwType)

	assert.Contains(t, query, "Redis")
	assert.Contains(t, query, "replication link down")
	assert.Contains(t, query, "replication_lag")
}
