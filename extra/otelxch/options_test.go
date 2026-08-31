package otelxch

import (
	"testing"

	"go.opentelemetry.io/otel/metric/noop"
)

func TestNewIgnoresNilOptions(t *testing.T) {
	provider := noop.NewMeterProvider()

	metrics, err := New(
		nil,
		WithMeterProvider(provider),
		nil,
	)
	if err != nil {
		t.Fatalf("create metrics: %v", err)
	}
	if metrics == nil {
		t.Fatal("metrics is nil")
	}
}

func TestWithMeterProviderIgnoresNil(t *testing.T) {
	provider := noop.NewMeterProvider()
	settings := settings{
		meterProvider: provider,
	}

	WithMeterProvider(nil).apply(&settings)

	if settings.meterProvider != provider {
		t.Fatal("nil meter provider replaced the configured provider")
	}
}
