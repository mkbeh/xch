package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/mkbeh/xch"
	"github.com/mkbeh/xch/extra/otelxch"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const defaultDSN = "clickhouse://clickhouse:clickhouse@localhost:9000/default?dial_timeout=5s"

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	// Configure an OpenTelemetry SDK exporter and MeterProvider.
	exporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint(),
	)
	if err != nil {
		return fmt.Errorf("create metrics exporter: %w", err)
	}

	reader := sdkmetric.NewPeriodicReader(exporter)

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)
	defer func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			log.Printf("shutdown meter provider: %v", err)
		}
	}()

	// Create one reusable xch OpenTelemetry integration.
	poolMetrics, err := otelxch.New(
		otelxch.WithMeterProvider(meterProvider),
	)
	if err != nil {
		return fmt.Errorf("create pool metrics: %w", err)
	}

	config, err := clickhouse.ParseDSN(databaseDSN())
	if err != nil {
		return fmt.Errorf("parse ClickHouse config: %w", err)
	}

	config.MaxOpenConns = 2
	config.MaxIdleConns = 2

	pool, err := xch.New(
		config,
		xch.WithName("observability-example"),
		xch.WithLabel("deployment.environment.name", "local"),
		xch.WithMetrics(poolMetrics),
	)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping ClickHouse: %w", err)
	}

	// Generate representative pool activity.
	if err := runWorkload(ctx, pool); err != nil {
		return fmt.Errorf("run workload: %w", err)
	}

	// Force one collection because this short-lived example exits
	// immediately. Long-running applications normally export periodically.
	if err := meterProvider.ForceFlush(ctx); err != nil {
		return fmt.Errorf("flush metrics: %w", err)
	}

	return nil
}

func runWorkload(ctx context.Context, pool *xch.Pool) error {
	const workerCount = 6

	start := make(chan struct{})
	results := make(chan error, workerCount)

	for range workerCount {
		go func() {
			<-start

			var result uint8
			err := pool.QueryRow(
				ctx,
				"SELECT sleep(1)",
			).Scan(&result)

			results <- err
		}()
	}

	close(start)

	var workloadErr error
	for range workerCount {
		workloadErr = errors.Join(
			workloadErr,
			<-results,
		)
	}

	return workloadErr
}

func databaseDSN() string {
	if value := os.Getenv("XCH_DATABASE_URL"); value != "" {
		return value
	}

	return defaultDSN
}
