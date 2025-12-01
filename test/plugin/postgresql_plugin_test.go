package plugin_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kubestack-ai/kubestack-ai/internal/common/logger"
	"github.com/kubestack-ai/kubestack-ai/internal/plugins/builtin/postgresql"
	"github.com/stretchr/testify/assert"
)

func TestPostgreSQL_Collector_Analyzer(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	log := logger.NewLogger("test")

	// 1. Mock Queries
	// pg_stat_activity
	// We use robust regex matching to handle whitespace differences.
	mock.ExpectQuery(`SELECT\s+count\(\*\)\s+as\s+total_connections.*FROM\s+pg_stat_activity`).
		WillReturnRows(sqlmock.NewRows([]string{"total", "active", "idle", "long_idle"}).AddRow(100, 10, 5, 2)) // 2 long idle

	// pg_stat_database
	// blks_hit=50, blks_read=100 -> ratio = 50/150 = 0.33
	mock.ExpectQuery(`SELECT\s+sum\(blks_hit\)\s+as\s+blks_hit.*FROM\s+pg_stat_database`).
		WillReturnRows(sqlmock.NewRows([]string{"hit", "read", "commit", "rollback"}).AddRow(50, 100, 1000, 10))

	// max_connections
	mock.ExpectQuery(`SHOW\s+max_connections`).
		WillReturnRows(sqlmock.NewRows([]string{"max_connections"}).AddRow("200"))

	// 2. Collect
	collector := postgresql.NewCollector(db, log)
	metricsData, err := collector.CollectMetrics(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, metricsData)

	// 3. Analyze
	analyzer := postgresql.NewAnalyzer(log)
	issues := analyzer.Analyze(metricsData.Data)

	// 4. Assertions
	assert.NotEmpty(t, issues)

	// Check Cache Hit Ratio Issue
	foundCacheIssue := false
	for _, issue := range issues {
		if issue.Title == "Low Cache Hit Ratio" {
			foundCacheIssue = true
			// 50 / 150 = 0.3333... -> 33.33%
			assert.Contains(t, issue.Evidence, "33.33%")
		}
	}
	assert.True(t, foundCacheIssue, "Should detect low cache hit ratio")

	// Check Long Idle Tx Issue
	foundIdleIssue := false
	for _, issue := range issues {
		if issue.Title == "Long Idle Transactions Detected" {
			foundIdleIssue = true
			assert.Contains(t, issue.Evidence, "2 transactions")
		}
	}
	assert.True(t, foundIdleIssue, "Should detect long idle transactions")

	// Check Connections Issue (100 / 200 = 50% < 85%) -> Should NOT find issue
	foundConnIssue := false
	for _, issue := range issues {
		if issue.Title == "High Connection Usage" {
			foundConnIssue = true
		}
	}
	assert.False(t, foundConnIssue, "Should NOT detect high connection usage (50%)")
}

func TestPostgreSQL_Analyzer_HighConnections(t *testing.T) {
	log := logger.NewLogger("test")
	analyzer := postgresql.NewAnalyzer(log)

	// Mock metrics for high connection usage
	metrics := map[string]interface{}{
		"connection_usage_percent": 90.0,
		"max_connections":          100.0,
	}

	issues := analyzer.Analyze(metrics)

	foundConnIssue := false
	for _, issue := range issues {
		if issue.Title == "High Connection Usage" {
			foundConnIssue = true
			assert.Contains(t, issue.Evidence, "90.00%")
		}
	}
	assert.True(t, foundConnIssue, "Should detect high connection usage")
}
