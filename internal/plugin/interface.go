package plugin

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/kubestack-ai/kubestack-ai/internal/core/models"
)

// MiddlewareType defines the type of middleware
type MiddlewareType string

const (
	MiddlewareRedis         MiddlewareType = "redis"
	MiddlewareMySQL         MiddlewareType = "mysql"
	MiddlewareKafka         MiddlewareType = "kafka"
	MiddlewareElasticsearch MiddlewareType = "elasticsearch"
	MiddlewarePostgreSQL    MiddlewareType = "postgresql"
	MiddlewareMongoDB       MiddlewareType = "mongodb"
)

// Plugin is an alias for DiagnosticPlugin
type Plugin = DiagnosticPlugin

// DiagnosticPlugin defines the interface implemented by plugins like ElasticsearchPlugin
type DiagnosticPlugin interface {
	Name() string
	SupportedTypes() []string
	Version() string
	Init(config map[string]interface{}) error
	Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error)
	Shutdown() error
}

// MiddlewarePlugin defines the interface for all middleware plugins (Legacy/Full)
type MiddlewarePlugin interface {
	// === Basic Information ===

	// Name returns the plugin name
	Name() string

	// Type returns the middleware type
	Type() MiddlewareType

	// Version returns the plugin version
	Version() string

	// === Connection Management ===

	// Connect establishes a connection
	Connect(ctx context.Context, config *ConnectionConfig) error

	// Disconnect closes the connection
	Disconnect(ctx context.Context) error

	// Ping checks if the connection is alive
	Ping(ctx context.Context) error

	// IsConnected returns the current connection status
	IsConnected() bool

	// === Metrics Collection ===

	// CollectMetrics collects all metrics
	CollectMetrics(ctx context.Context) (*MetricsSnapshot, error)

	// CollectSpecificMetric collects a specific metric group or value
	CollectSpecificMetric(ctx context.Context, metricName string) (interface{}, error)

	// === Command Execution ===

	// Execute executes a command
	Execute(ctx context.Context, cmd *Command) (*CommandResult, error)

	// SupportedCommands returns the list of supported commands
	SupportedCommands() []CommandSpec

	// === Diagnosis Support ===

	// GetDiagnosticData collects all data needed for diagnosis
	GetDiagnosticData(ctx context.Context) (*DiagnosticData, error)

	// GetBuiltinRules returns built-in diagnosis rules
	GetBuiltinRules() []DiagnosisRule
}

// TLSConfig defines TLS configuration
type TLSConfig struct {
	InsecureSkipVerify bool
	CertFile           string
	KeyFile            string
	CAFile             string
}

// ToTLSConfig converts to standard tls.Config
func (c *TLSConfig) ToTLSConfig() *tls.Config {
	if c == nil {
		return nil
	}
	// Note: In a real implementation, we would load certs here.
	// For now, we just return a basic config or nil if empty.
	return &tls.Config{
		InsecureSkipVerify: c.InsecureSkipVerify,
	}
}

// ConnectionConfig defines connection parameters
type ConnectionConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string // MySQL/PostgreSQL
	TLS      *TLSConfig
	Timeout  time.Duration
	PoolSize int
	Extra    map[string]string // Middleware specific config
}

// MetricsSnapshot represents a snapshot of metrics
type MetricsSnapshot struct {
	Timestamp time.Time
	Metrics   map[string]MetricValue
	RawData   map[string]interface{} // Raw data
}

// MetricValue represents a single metric value
type MetricValue struct {
	Name      string
	Value     float64
	Unit      string
	Labels    map[string]string
	Timestamp time.Time
}

// Command represents a command to be executed
type Command struct {
	Name    string
	Args    []interface{} // Changed to interface{} to support diverse args
	Timeout time.Duration
	DryRun  bool // Preview only
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success      bool
	Output       string
	Error        string
	Duration     time.Duration
	AffectedRows int64 // For databases
}

// CommandSpec defines the specification of a command
type CommandSpec struct {
	Name        string
	Description string
	Syntax      string
	RiskLevel   int // 1-5
	Examples    []string
}

// DiagnosticData contains all data collected for diagnosis
type DiagnosticData struct {
	Metrics     *MetricsSnapshot
	Config      map[string]interface{}
	SlowLogs    []SlowLogEntry
	Connections []ConnectionInfo
	Replication *ReplicationInfo
	Extra       map[string]interface{}
}

// SlowLogEntry represents a slow query log entry
type SlowLogEntry struct {
	ID        string
	Time      time.Time
	Duration  time.Duration
	Command   string
	Query     string
	ClientIP  string
	User      string
	Database  string
	RowsSent  int64
	RowsExam  int64
}

// ConnectionInfo represents client connection info
type ConnectionInfo struct {
	ID       string
	User     string
	ClientIP string
	Database string
	Command  string
	Time     int64 // seconds
	State    string
	Info     string
}

// ReplicationInfo represents replication status
type ReplicationInfo struct {
	Role             string // master/slave
	MasterHost       string
	MasterPort       int
	MasterLinkStatus string
	SlaveLag         time.Duration
	ConnectedSlaves  int
	Details          map[string]interface{}
}

// DiagnosisRule represents a rule for diagnosis
type DiagnosisRule struct {
	ID          string
	Name        string
	Description string
	Severity    Severity
	Condition   string // Expression
	Message     string // Message template
	Suggestion  string // Suggestion template
	Tags        []string
	Enabled     bool
}

// Severity defines the severity level of an issue
type Severity int

const (
	SeverityInfo     Severity = 1
	SeverityWarning  Severity = 2
	SeverityError    Severity = 3
	SeverityCritical Severity = 4
)

// Target struct for legacy compatibility if needed
type Target struct {
	Type    string
	Address string
}
