package otelxch

import (
	"testing"

	"github.com/mkbeh/xch"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type testMetricsSource struct {
	name   string
	labels map[string]string
	stats  xch.PoolStats
}

func (source *testMetricsSource) Name() string {
	return source.name
}

func (source *testMetricsSource) Labels() map[string]string {
	return source.labels
}

func (source *testMetricsSource) Stats() xch.PoolStats {
	return source.stats
}

func TestMetricsRegister(t *testing.T) {
	reader, metrics := newTestMetrics(t)

	source := &testMetricsSource{
		name: "analytics",
		labels: map[string]string{
			"environment":     "production",
			poolNameAttribute: "ignored",
		},
		stats: xch.PoolStats{
			OpenConns:    7,
			IdleConns:    3,
			MaxOpenConns: 12,
			MaxIdleConns: 5,
		},
	}

	registration, err := metrics.Register(source)
	if err != nil {
		t.Fatalf("register metrics: %v", err)
	}
	t.Cleanup(registration.Close)

	collected := collectMetrics(t, reader)

	tests := []struct {
		name        string
		want        int64
		description string
	}{
		{
			name:        connectionOpenMetricName,
			want:        7,
			description: "The number of currently open connections.",
		},
		{
			name:        connectionIdleMetricName,
			want:        3,
			description: "The number of currently idle connections.",
		},
		{
			name:        connectionMaxOpenMetricName,
			want:        12,
			description: "The maximum number of open connections allowed by the pool.",
		},
		{
			name:        connectionMaxIdleMetricName,
			want:        5,
			description: "The maximum number of idle connections retained by the pool.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := findMetric(t, collected, test.name)

			if current.Description != test.description {
				t.Fatalf(
					"metric %q description = %q, want %q",
					test.name,
					current.Description,
					test.description,
				)
			}
			if current.Unit != "{connection}" {
				t.Fatalf(
					"metric %q unit = %q, want %q",
					test.name,
					current.Unit,
					"{connection}",
				)
			}

			point := findDataPoint(t, current, source.name)

			if point.Value != test.want {
				t.Fatalf(
					"metric %q value = %d, want %d",
					test.name,
					point.Value,
					test.want,
				)
			}

			assertAttribute(
				t,
				point.Attributes,
				poolNameAttribute,
				source.name,
			)
			assertAttribute(
				t,
				point.Attributes,
				"environment",
				"production",
			)
		})
	}

	source.stats.OpenConns = 9
	source.stats.IdleConns = 1

	collected = collectMetrics(t, reader)

	if got := findDataPoint(
		t,
		findMetric(t, collected, connectionOpenMetricName),
		source.name,
	).Value; got != 9 {
		t.Fatalf("open connections = %d, want 9", got)
	}

	if got := findDataPoint(
		t,
		findMetric(t, collected, connectionIdleMetricName),
		source.name,
	).Value; got != 1 {
		t.Fatalf("idle connections = %d, want 1", got)
	}
}

func TestMetricsReuseAcrossSources(t *testing.T) {
	reader, metrics := newTestMetrics(t)

	primary := &testMetricsSource{
		name: "primary",
		labels: map[string]string{
			"region": "eu",
		},
		stats: xch.PoolStats{
			MaxOpenConns: 8,
		},
	}
	analytics := &testMetricsSource{
		name: "analytics",
		labels: map[string]string{
			"region": "us",
		},
		stats: xch.PoolStats{
			MaxOpenConns: 16,
		},
	}

	primaryRegistration, err := metrics.Register(primary)
	if err != nil {
		t.Fatalf("register primary metrics: %v", err)
	}

	analyticsRegistration, err := metrics.Register(analytics)
	if err != nil {
		t.Fatalf("register analytics metrics: %v", err)
	}
	t.Cleanup(analyticsRegistration.Close)

	collected := collectMetrics(t, reader)
	current := findMetric(
		t,
		collected,
		connectionMaxOpenMetricName,
	)

	primaryPoint := findDataPoint(t, current, primary.name)
	if primaryPoint.Value != 8 {
		t.Fatalf(
			"primary max open connections = %d, want 8",
			primaryPoint.Value,
		)
	}
	assertAttribute(t, primaryPoint.Attributes, "region", "eu")

	analyticsPoint := findDataPoint(t, current, analytics.name)
	if analyticsPoint.Value != 16 {
		t.Fatalf(
			"analytics max open connections = %d, want 16",
			analyticsPoint.Value,
		)
	}
	assertAttribute(t, analyticsPoint.Attributes, "region", "us")

	primaryRegistration.Close()

	collected = collectMetrics(t, reader)
	current = findMetric(
		t,
		collected,
		connectionMaxOpenMetricName,
	)

	assertDataPointAbsent(t, current, primary.name)

	analyticsPoint = findDataPoint(t, current, analytics.name)
	if analyticsPoint.Value != 16 {
		t.Fatalf(
			"analytics max open connections after unregister = %d, want 16",
			analyticsPoint.Value,
		)
	}
}

func TestMetricsRegisterValidation(t *testing.T) {
	_, metrics := newTestMetrics(t)

	source := &testMetricsSource{
		name: "analytics",
	}

	var nilMetrics *Metrics

	if _, err := nilMetrics.Register(source); err == nil ||
		err.Error() != "otelxch: metrics is nil" {
		t.Fatalf("nil metrics error = %v", err)
	}

	if _, err := metrics.Register(nil); err == nil ||
		err.Error() != "otelxch: metrics source is nil" {
		t.Fatalf("nil source error = %v", err)
	}

	source.name = ""

	if _, err := metrics.Register(source); err == nil ||
		err.Error() != "otelxch: pool name must not be empty" {
		t.Fatalf("empty name error = %v", err)
	}
}

func newTestMetrics(
	t *testing.T,
) (*sdkmetric.ManualReader, *Metrics) {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)

	t.Cleanup(func() {
		if err := provider.Shutdown(t.Context()); err != nil {
			t.Fatalf("shutdown meter provider: %v", err)
		}
	})

	metrics, err := New(
		WithMeterProvider(provider),
	)
	if err != nil {
		t.Fatalf("create metrics: %v", err)
	}

	return reader, metrics
}

func collectMetrics(
	t *testing.T,
	reader *sdkmetric.ManualReader,
) metricdata.ResourceMetrics {
	t.Helper()

	var collected metricdata.ResourceMetrics

	if err := reader.Collect(t.Context(), &collected); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	return collected
}

func findMetric(
	t *testing.T,
	collected metricdata.ResourceMetrics,
	name string,
) metricdata.Metrics {
	t.Helper()

	for _, scope := range collected.ScopeMetrics {
		for _, current := range scope.Metrics {
			if current.Name == name {
				return current
			}
		}
	}

	t.Fatalf("metric %q not found", name)

	return metricdata.Metrics{}
}

func findDataPoint(
	t *testing.T,
	current metricdata.Metrics,
	poolName string,
) metricdata.DataPoint[int64] {
	t.Helper()

	gauge, ok := current.Data.(metricdata.Gauge[int64])
	if !ok {
		t.Fatalf(
			"metric %q data type = %T, want metricdata.Gauge[int64]",
			current.Name,
			current.Data,
		)
	}

	for _, point := range gauge.DataPoints {
		value, ok := point.Attributes.Value(
			attribute.Key(poolNameAttribute),
		)
		if ok && value.AsString() == poolName {
			return point
		}
	}

	t.Fatalf(
		"metric %q has no data point for pool %q",
		current.Name,
		poolName,
	)

	return metricdata.DataPoint[int64]{}
}

func assertDataPointAbsent(
	t *testing.T,
	current metricdata.Metrics,
	poolName string,
) {
	t.Helper()

	gauge, ok := current.Data.(metricdata.Gauge[int64])
	if !ok {
		t.Fatalf(
			"metric %q data type = %T, want metricdata.Gauge[int64]",
			current.Name,
			current.Data,
		)
	}

	for _, point := range gauge.DataPoints {
		value, ok := point.Attributes.Value(
			attribute.Key(poolNameAttribute),
		)
		if ok && value.AsString() == poolName {
			t.Fatalf(
				"metric %q still contains pool %q",
				current.Name,
				poolName,
			)
		}
	}
}

func assertAttribute(
	t *testing.T,
	attributes attribute.Set,
	key string,
	want string,
) {
	t.Helper()

	value, ok := attributes.Value(attribute.Key(key))
	if !ok {
		t.Fatalf("attribute %q not found", key)
	}

	if got := value.AsString(); got != want {
		t.Fatalf(
			"attribute %q = %q, want %q",
			key,
			got,
			want,
		)
	}
}
