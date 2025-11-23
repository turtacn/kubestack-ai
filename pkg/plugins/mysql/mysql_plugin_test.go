package mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/stretchr/testify/assert"
)

func TestMySQLHealthChecker_Check(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	p := &MySQLPlugin{db: db}
	checker := &MySQLHealthChecker{plugin: p}
	ctx := context.Background()

	// 1. PING
	mock.ExpectPing()

	// 2. Connections
	// SHOW GLOBAL STATUS LIKE 'Threads_connected'
	mock.ExpectQuery("SHOW GLOBAL STATUS LIKE 'Threads_connected'").
		WillReturnRows(sqlmock.NewRows([]string{"Variable_name", "Value"}).AddRow("Threads_connected", "10"))

	// SHOW VARIABLES LIKE 'max_connections'
	mock.ExpectQuery("SHOW VARIABLES LIKE 'max_connections'").
		WillReturnRows(sqlmock.NewRows([]string{"Variable_name", "Value"}).AddRow("max_connections", "100"))

	// 3. Replication
	// SHOW SLAVE STATUS -> empty (Master)
	mock.ExpectQuery("SHOW SLAVE STATUS").
		WillReturnRows(sqlmock.NewRows([]string{"Slave_IO_Running"}))

	status, err := checker.Check(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, plugin.HealthyLevel, status.Overall)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMySQLHealthChecker_ReplicationBroken(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	p := &MySQLPlugin{db: db}
	checker := &MySQLHealthChecker{plugin: p}

	mock.ExpectPing()
	mock.ExpectQuery("SHOW GLOBAL STATUS LIKE 'Threads_connected'").
		WillReturnRows(sqlmock.NewRows([]string{"k", "v"}).AddRow("Threads_connected", "10"))
	mock.ExpectQuery("SHOW VARIABLES LIKE 'max_connections'").
		WillReturnRows(sqlmock.NewRows([]string{"k", "v"}).AddRow("max_connections", "100"))

	// SHOW SLAVE STATUS -> Broken
	mock.ExpectQuery("SHOW SLAVE STATUS").
		WillReturnRows(sqlmock.NewRows([]string{"Slave_IO_Running", "Slave_SQL_Running", "Seconds_Behind_Master"}).
			AddRow("No", "Yes", "0"))

	status, err := checker.Check(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, plugin.UnhealthyLevel, status.Overall)
	// Find replication item
	for _, item := range status.Items {
		if item.Name == "replication" {
			assert.Contains(t, item.Message, "Replication broken")
		}
	}
}

func TestMySQLMetricParser_Parse(t *testing.T) {
	p := &MySQLPlugin{}
	parser := &MySQLMetricParser{plugin: p}

	data := &plugin.CollectedData{
		RawData: map[string]interface{}{
			"status": map[string]string{
				"Queries":           "1000",
				"Uptime":            "100",
				"Threads_connected": "50",
			},
			"variables": map[string]string{
				"max_connections": "100",
			},
		},
	}

	metrics, err := parser.Parse(context.Background(), data)
	assert.NoError(t, err)

	assert.Equal(t, 10.0, metrics.Metrics["qps"].Value)
	assert.Equal(t, int64(50), metrics.Metrics["threads_connected"].Value)
	assert.Equal(t, 0.5, metrics.Metrics["connection_usage"].Value)
}
