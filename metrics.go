package xch

// MetricsSource provides pool metadata and statistics required by metrics
// integrations.
type MetricsSource interface {
	Name() string
	Labels() map[string]string
	Stats() PoolStats
}

// Metrics registers metrics for a Pool.
//
// Implementations must be safe to reuse across multiple pools.
type Metrics interface {
	Register(source MetricsSource) (MetricsRegistration, error)
}

// MetricsRegistration represents a metrics registration for one Pool.
//
// Close is called once before the underlying clickhouse-go connection pool is
// closed.
type MetricsRegistration interface {
	Close()
}
