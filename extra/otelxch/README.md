# OpenTelemetry Metrics for xch

`otelxch` provides optional OpenTelemetry metrics integration for `xch`.

The package is exporter-agnostic: applications own the OpenTelemetry SDK lifecycle and exporter configuration, while
`otelxch` uses the configured `MeterProvider` to expose client connection pool metrics.

## Installation

```bash
go get github.com/mkbeh/xch/extra/otelxch
```

## Usage

<!-- @formatter:off -->
```go
import (
	"context"

	"github.com/mkbeh/xch"
	"github.com/mkbeh/xch/extra/otelxch"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

meterProvider := sdkmetric.NewMeterProvider()
defer meterProvider.Shutdown(context.Background())

// Create a reusable OpenTelemetry metrics integration.
metrics, err := otelxch.New(
	otelxch.WithMeterProvider(meterProvider),
)
if err != nil {
	return err
}

// Attach metrics when creating the pool.
pool, err := xch.New(
	config,
	xch.WithName("example-pool"),
	xch.WithMetrics(metrics),
)
if err != nil {
	return err
}
defer pool.Close()
```
<!-- @formatter:on -->

For a complete runnable setup using OpenTelemetry and Prometheus, see the [example](../../examples/observability).
