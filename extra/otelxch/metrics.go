package otelxch

import (
	"context"
	"errors"
	"fmt"

	"github.com/mkbeh/xch"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const instrumentationName = "github.com/mkbeh/xch/extra/otelxch"

const (
	connectionOpenMetricName    = "xch.client.connection.open"
	connectionIdleMetricName    = "xch.client.connection.idle"
	connectionMaxOpenMetricName = "xch.client.connection.max_open"
	connectionMaxIdleMetricName = "xch.client.connection.max_idle"
)

const poolNameAttribute = "db.client.connection.pool.name"

// Metrics exports xch client connection pool statistics through OpenTelemetry.
//
// Metrics is safe for concurrent use and reuse across multiple pools.
type Metrics struct {
	meter       metric.Meter
	instruments poolMetricInstruments
}

type poolMetricInstruments struct {
	connectionOpen    metric.Int64ObservableGauge
	connectionIdle    metric.Int64ObservableGauge
	connectionMaxOpen metric.Int64ObservableGauge
	connectionMaxIdle metric.Int64ObservableGauge
}

type registration struct {
	registration metric.Registration
}

func (r *registration) Close() {
	if r == nil || r.registration == nil {
		return
	}

	if err := r.registration.Unregister(); err != nil {
		otel.Handle(
			fmt.Errorf("otelxch: unregister pool metrics: %w", err),
		)
	}
}

// Register registers OpenTelemetry metrics for one metrics source.
//
// The source must have a stable name. Source labels are exported as metric
// attributes and must remain stable and low-cardinality.
func (m *Metrics) Register(source xch.MetricsSource) (xch.MetricsRegistration, error) {
	if m == nil {
		return nil, errors.New("otelxch: metrics is nil")
	}
	if source == nil {
		return nil, errors.New("otelxch: metrics source is nil")
	}
	if source.Name() == "" {
		return nil, errors.New("otelxch: pool name must not be empty")
	}

	attributes := newPoolMetricAttributes(
		source.Name(),
		source.Labels(),
	)
	option := metric.WithAttributeSet(attributes)

	registered, err := m.meter.RegisterCallback(
		func(_ context.Context, observer metric.Observer) error {
			m.instruments.observe(observer, source.Stats(), option)

			return nil
		},
		m.instruments.connectionOpen,
		m.instruments.connectionIdle,
		m.instruments.connectionMaxOpen,
		m.instruments.connectionMaxIdle,
	)
	if err != nil {
		return nil, fmt.Errorf("otelxch: register pool metrics callback: %w", err)
	}

	return &registration{
		registration: registered,
	}, nil
}

func newPoolMetricInstruments(meter metric.Meter) (poolMetricInstruments, error) {
	var instruments poolMetricInstruments

	var err error

	instruments.connectionOpen, err = meter.Int64ObservableGauge(
		connectionOpenMetricName,
		metric.WithDescription(
			"The number of currently open connections.",
		),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return poolMetricInstruments{}, fmt.Errorf(
			"otelxch: create %s: %w",
			connectionOpenMetricName,
			err,
		)
	}

	instruments.connectionIdle, err = meter.Int64ObservableGauge(
		connectionIdleMetricName,
		metric.WithDescription(
			"The number of currently idle connections.",
		),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return poolMetricInstruments{}, fmt.Errorf(
			"otelxch: create %s: %w",
			connectionIdleMetricName,
			err,
		)
	}

	instruments.connectionMaxOpen, err = meter.Int64ObservableGauge(
		connectionMaxOpenMetricName,
		metric.WithDescription(
			"The maximum number of open connections allowed by the pool.",
		),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return poolMetricInstruments{}, fmt.Errorf(
			"otelxch: create %s: %w",
			connectionMaxOpenMetricName,
			err,
		)
	}

	instruments.connectionMaxIdle, err = meter.Int64ObservableGauge(
		connectionMaxIdleMetricName,
		metric.WithDescription(
			"The maximum number of idle connections retained by the pool.",
		),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return poolMetricInstruments{}, fmt.Errorf(
			"otelxch: create %s: %w",
			connectionMaxIdleMetricName,
			err,
		)
	}

	return instruments, nil
}

func newPoolMetricAttributes(name string, labels map[string]string) attribute.Set {
	var attributes []attribute.KeyValue

	for key, value := range labels {
		if key == poolNameAttribute {
			continue
		}

		attributes = append(
			attributes,
			attribute.String(key, value),
		)
	}

	attributes = append(
		attributes,
		attribute.String(poolNameAttribute, name),
	)

	return attribute.NewSet(attributes...)
}

func (instruments poolMetricInstruments) observe(
	observer metric.Observer,
	stats xch.PoolStats,
	option metric.ObserveOption,
) {
	observer.ObserveInt64(
		instruments.connectionOpen,
		int64(stats.OpenConns),
		option,
	)
	observer.ObserveInt64(
		instruments.connectionIdle,
		int64(stats.IdleConns),
		option,
	)
	observer.ObserveInt64(
		instruments.connectionMaxOpen,
		int64(stats.MaxOpenConns),
		option,
	)
	observer.ObserveInt64(
		instruments.connectionMaxIdle,
		int64(stats.MaxIdleConns),
		option,
	)
}
