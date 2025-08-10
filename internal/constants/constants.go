package constants

// MiddlewareType 中间件类型枚举。MiddlewareType enum for middleware types.
const (
	MiddlewareMySQL         = "mysql"
	MiddlewareRedis         = "redis"
	MiddlewareKafka         = "kafka"
	MiddlewareElasticsearch = "elasticsearch"
	MiddlewareRabbitMQ      = "rabbitmq"
	MiddlewarePostgreSQL    = "postgresql"
	MiddlewareMongoDB       = "mongodb"
	MiddlewareClickHouse    = "clickhouse"
	MiddlewareEtcd          = "etcd"
	MiddlewarePrometheus    = "prometheus"
	MiddlewareMinIO         = "minio"
	// 更多。More.
)

// StatusLevel 状态级别。StatusLevel for diagnosis status.
const (
	StatusHealthy  = "healthy"
	StatusWarning  = "warning"
	StatusCritical = "critical"
)

// SeverityLevel 严重性级别。SeverityLevel for findings.
const (
	SeverityLow    = "low"
	SeverityMedium = "medium"
	SeverityHigh   = "high"
)

//Personal.AI order the ending
