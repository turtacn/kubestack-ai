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

// EnvironmentType 环境类型。EnvironmentType for deployment environments.
const (
	EnvironmentKubernetes = "kubernetes"
	EnvironmentBareMetal  = "baremetal"
)

// KnowledgeSource 知识来源类型。KnowledgeSource for RAG content sources.
const (
	KnowledgeSourceOfficial  = "official"
	KnowledgeSourceCommunity = "community"
	KnowledgeSourceCaseStudy = "case_study"
)

//Personal.AI order the ending
