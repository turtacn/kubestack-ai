package builtin

import (
	"github.com/kubestack-ai/kubestack-ai/internal/plugin"
	"github.com/kubestack-ai/kubestack-ai/plugins/elasticsearch"
	"github.com/kubestack-ai/kubestack-ai/plugins/kafka"
	"github.com/kubestack-ai/kubestack-ai/plugins/mysql"
	"github.com/kubestack-ai/kubestack-ai/plugins/postgresql"
	"github.com/kubestack-ai/kubestack-ai/plugins/redis"
)

// RegisterAll registers all built-in plugins with the given manager
func RegisterAll(manager *plugin.Manager) {
	// Register Redis plugin
	manager.RegisterBuiltinPlugin("redis-diagnostics", func() plugin.Plugin {
		return redis.NewRedisPlugin()
	})

	// Register Kafka plugin
	manager.RegisterBuiltinPlugin("kafka-diagnostics", func() plugin.Plugin {
		return kafka.NewKafkaPlugin()
	})

	// Register MySQL plugin
	manager.RegisterBuiltinPlugin("mysql-diagnostics", func() plugin.Plugin {
		return mysql.NewMySQLPlugin()
	})

	// Register PostgreSQL plugin
	manager.RegisterBuiltinPlugin("postgresql-diagnostics", func() plugin.Plugin {
		return postgresql.NewPostgreSQLPlugin()
	})

	// Register Elasticsearch plugin
	manager.RegisterBuiltinPlugin("elasticsearch-diagnostics", func() plugin.Plugin {
		return elasticsearch.NewElasticsearchPlugin()
	})
}
