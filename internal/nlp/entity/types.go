package entity

// EntityType defines the type of entity.
type EntityType string

const (
	EntityMiddlewareType EntityType = "middleware_type" // redis, mysql, etc.
	EntityInstanceID     EntityType = "instance_id"     // redis-cluster-01
	EntityMetricName     EntityType = "metric_name"     // memory_usage, connections
	EntityTimeRange      EntityType = "time_range"      // last 1h, yesterday
	EntityThreshold      EntityType = "threshold"       // 80%, 1000
	EntityCommand        EntityType = "command"         // FLUSHALL, KILL
	EntityConfigKey      EntityType = "config_key"      // maxmemory
	EntityConfigValue    EntityType = "config_value"    // 2gb
)

// Entity represents an extracted entity.
type Entity struct {
	Type       EntityType `json:"type"`
	Value      string     `json:"value"`      // Original extracted value
	NormValue  string     `json:"norm_value"` // Normalized value
	StartPos   int        `json:"start_pos"`
	EndPos     int        `json:"end_pos"`
	Confidence float64    `json:"confidence"`
}

// EntityExtractionResult holds all extracted entities from a text.
type EntityExtractionResult struct {
	Entities []Entity `json:"entities"`
	Text     string   `json:"text"`
}
