module observability

go 1.27

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.48.0
	github.com/mkbeh/xch v0.2.0
	github.com/mkbeh/xch/extra/otelxch v0.1.0
	go.opentelemetry.io/otel/sdk/metric v1.46.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.46.0
)