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

// Plugin defines the core interface that all plugins must implement
type Plugin interface {
	// Info returns plugin metadata
	Info() EnhancedPluginInfo
	
	// Init initializes the plugin with configuration
	Init(ctx context.Context, config PluginConfig) error
	
	// Start starts the plugin
	Start(ctx context.Context) error
	
	// Stop stops the plugin gracefully
	Stop(ctx context.Context) error
	
	// HealthCheck performs a health check on the plugin
	HealthCheck(ctx context.Context) error
}

// EnhancedPluginInfo contains metadata about a plugin
type EnhancedPluginInfo struct {
	ID           string            `json:"id" yaml:"id"`
	Name         string            `json:"name" yaml:"name"`
	Version      string            `json:"version" yaml:"version"`
	Type         PluginType        `json:"type" yaml:"type"`
	Description  string            `json:"description" yaml:"description"`
	Author       string            `json:"author" yaml:"author"`
	Homepage     string            `json:"homepage" yaml:"homepage"`
	License      string            `json:"license" yaml:"license"`
	Requires     []string          `json:"requires" yaml:"requires"`
	Capabilities []string          `json:"capabilities" yaml:"capabilities"`
	ConfigSchema *JSONSchema       `json:"config_schema,omitempty" yaml:"config_schema,omitempty"`
}

// DiagnosticPlugin defines the interface implemented by plugins like ElasticsearchPlugin
type DiagnosticPlugin interface {
	Name() string
	SupportedTypes() []string
	Version() string
	Init(config map[string]interface{}) error
	Diagnose(ctx context.Context, req *models.DiagnosisRequest) (*models.DiagnosisResult, error)
	Shutdown() error
}

// MiddlewarePlugin defines the interface for all middleware plugins (Legacy/Full).
//
// DESIGN NOTE: This interface represents the operation-oriented plugin interface
// that is currently implemented by existing plugins. The design-aligned contract
// interface is defined in internal/core/contracts.MiddlewarePlugin, which provides
// a diagnosis-focused API surface.
//
// The adapter layer in internal/core/contracts/adapter bridges between this interface
// and the contracts interface, allowing existing plugins to work with the new contract
// without requiring changes to plugin implementations.
//
// For new plugin development, consider implementing the contracts.MiddlewarePlugin interface
// directly, or implement this interface and use the adapter for compatibility.
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

// EnhancedMiddlewarePlugin extends the Plugin interface for middleware diagnostics
type EnhancedMiddlewarePlugin interface {
	Plugin
	
	// MiddlewareType returns the type of middleware
	MiddlewareType() string
	
	// SupportedVersions returns the list of supported middleware versions
	SupportedVersions() []string
	
	// Connect establishes a connection to the middleware
	Connect(ctx context.Context, target MiddlewareTarget) error
	
	// Disconnect closes the connection
	Disconnect(ctx context.Context) error
	
	// Diagnose performs diagnostic checks
	Diagnose(ctx context.Context, opts DiagnoseOptions) (*DiagnosticResult, error)
	
	// GetMetrics retrieves current metrics
	GetMetrics(ctx context.Context) (map[string]interface{}, error)
	
	// Execute performs an action on the middleware
	Execute(ctx context.Context, action string, params map[string]interface{}) (interface{}, error)
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

// PluginType defines the type of plugin
type PluginType string

const (
	PluginTypeMiddleware  PluginType = "middleware"
	PluginTypeDiagnostic  PluginType = "diagnostic"
	PluginTypeAction      PluginType = "action"
	PluginTypeIntegration PluginType = "integration"
)

// JSONSchema represents a simplified JSON schema for config validation
type JSONSchema struct {
	Type       string                 `json:"type" yaml:"type"`
	Properties map[string]JSONSchema  `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required   []string               `json:"required,omitempty" yaml:"required,omitempty"`
	Default    interface{}            `json:"default,omitempty" yaml:"default,omitempty"`
}

// MiddlewareTarget defines a target middleware instance to diagnose
type MiddlewareTarget struct {
	Type      string            `json:"type" yaml:"type"`
	Name      string            `json:"name" yaml:"name"`
	Endpoints []string          `json:"endpoints" yaml:"endpoints"`
	Auth      *AuthConfig       `json:"auth,omitempty" yaml:"auth,omitempty"`
	TLS       *TLSConfig        `json:"tls,omitempty" yaml:"tls,omitempty"`
	Options   map[string]string `json:"options,omitempty" yaml:"options,omitempty"`
}

// AuthConfig contains authentication credentials
type AuthConfig struct {
	Username  string `json:"username,omitempty" yaml:"username,omitempty"`
	Password  string `json:"password,omitempty" yaml:"password,omitempty"`
	Token     string `json:"token,omitempty" yaml:"token,omitempty"`
	SecretRef string `json:"secret_ref,omitempty" yaml:"secret_ref,omitempty"`
}

// DiagnosticResult contains the result of a diagnostic operation
type DiagnosticResult struct {
	PluginID    string              `json:"plugin_id" yaml:"plugin_id"`
	TargetName  string              `json:"target_name" yaml:"target_name"`
	Status      DiagnosticStatus    `json:"status" yaml:"status"`
	Findings    []Finding           `json:"findings" yaml:"findings"`
	Metrics     map[string]interface{} `json:"metrics" yaml:"metrics"`
	Suggestions []string            `json:"suggestions" yaml:"suggestions"`
	Timestamp   time.Time           `json:"timestamp" yaml:"timestamp"`
	Duration    time.Duration       `json:"duration" yaml:"duration"`
}

// DiagnosticStatus represents the overall health status
type DiagnosticStatus string

const (
	DiagnosticStatusHealthy  DiagnosticStatus = "healthy"
	DiagnosticStatusWarning  DiagnosticStatus = "warning"
	DiagnosticStatusCritical DiagnosticStatus = "critical"
	DiagnosticStatusUnknown  DiagnosticStatus = "unknown"
)

// Finding represents a single diagnostic finding
type Finding struct {
	Severity    Severity               `json:"severity" yaml:"severity"`
	Category    string                 `json:"category" yaml:"category"`
	Title       string                 `json:"title" yaml:"title"`
	Description string                 `json:"description" yaml:"description"`
	Evidence    map[string]interface{} `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	Remediation string                 `json:"remediation,omitempty" yaml:"remediation,omitempty"`
}

// DiagnoseOptions contains options for diagnostic operations
type DiagnoseOptions struct {
	Categories []string      `json:"categories,omitempty" yaml:"categories,omitempty"`
	Depth      string        `json:"depth" yaml:"depth"`
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
}

// PluginFactory is a function that creates a new plugin instance (Enhanced version)
type PluginFactoryFunc func() Plugin

// PluginHooks defines callbacks for plugin lifecycle events
type PluginHooks interface {
	OnLoad(plugin Plugin) error
	OnUnload(plugin Plugin) error
	OnError(plugin Plugin, err error)
}

// Target struct for legacy compatibility if needed
type Target struct {
	Type    string
	Address string
}
