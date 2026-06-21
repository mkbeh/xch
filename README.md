<div align="center">

# xclick

**Lightweight ClickHouse wrapper for Go, built on top of [clickhouse-go](https://github.com/ClickHouse/clickhouse-go).**

![Go Version](https://img.shields.io/badge/go-1.26%2B-blue)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

</div>

`xclick` wraps the excellent [`clickhouse-go`](https://github.com/ClickHouse/clickhouse-go) client with a compact API
for common ClickHouse workflows: connection pool initialization, query building, embedded SQL migrations, normalized
errors, TLS and compression options, and Prometheus metrics.

## Features

* **Pool**: Simple ClickHouse connection pool initialization.
* **Queries**: ClickHouse-friendly query builder support.
* **Migrations**: Embedded SQL migrations via [golang-migrate](https://github.com/golang-migrate/migrate).
* **Observability**: Prometheus connection pool metrics out of the box.
* **Errors**: Normalized ClickHouse error codes for common failure cases.
* **Security**: TLS configuration and insecure-skip-verify support.
* **Configuration**: Configure via Go structs or environment variables.

## Installation

```bash
go get github.com/mkbeh/xclick
```

## Quick Start

The example below creates a ClickHouse connection pool and runs a simple query.

<!-- @formatter:off -->
```go
package main

import (
	"context"
	"fmt"
	"log"

	clickhouse "github.com/mkbeh/xclick"
)

func main() {
	ctx := context.Background()

	cfg := &clickhouse.Config{
		Hosts:    "127.0.0.1:9000",
		User:     "user",
		Password: "password",
		DB:       "mydb",
	}

	pool, err := clickhouse.NewPool(
		clickhouse.WithConfig(cfg),
		clickhouse.WithClientID("my-service"), // used in metric labels
	)
	if err != nil {
		log.Fatalf("failed to init ClickHouse pool: %v", err)
	}
	defer pool.Close()

	var result string
	err = pool.QueryRow(ctx, "SELECT 'Hello, world!'").Scan(&result)
	if err != nil {
		log.Fatalf("query failed: %v", err)
	}

	fmt.Println(result)
}
```
<!-- @formatter:on -->

More examples: [examples/](https://github.com/mkbeh/xclick/tree/main/examples)

## Query builder

Each pool includes a preconfigured [squirrel](https://github.com/Masterminds/squirrel) statement builder.

<!-- @formatter:off -->
```go
sql, args, err := pool.QueryBuilder().
	Select("event_type", "count() AS total").
	From("events").
	Where(squirrel.Eq{"project_id": projectID}).
	GroupBy("event_type").
	ToSql()
if err != nil {
	log.Fatalf("failed to build query: %v", err)
}

rows, err := pool.Query(ctx, sql, args...)
if err != nil {
	log.Fatalf("failed to execute query: %v", err)
}
defer rows.Close()
```
<!-- @formatter:on -->

## Migrations

`xclick` supports embedded SQL migrations through [golang-migrate](https://github.com/golang-migrate/migrate).

Create `embed.go` in your migrations package:

```go
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

Pass the embedded filesystem via `WithMigrations`:

<!-- @formatter:off -->
```go
pool, err := clickhouse.NewPool(
	clickhouse.WithConfig(&clickhouse.Config{
		Hosts:          "127.0.0.1:9000",
		User:           "user",
		Password:       "password",
		DB:             "mydb",
		MigrateEnabled: true,
	}),
	clickhouse.WithMigrations(migrations.FS),
)
if err != nil {
	log.Fatalf("failed to initialize pool and run migrations: %v", err)
}
defer pool.Close()
```
<!-- @formatter:on -->


Migrations run automatically on `NewPool` when `MigrateEnabled` is `true`.

Migration files follow the standard `golang-migrate` naming format:

```text
000001_create_events.up.sql
000001_create_events.down.sql
```

### Additional Migration Arguments

Use the `CLICKHOUSE_MIGRATE_ARGS` environment variable or the `MigrateArgs` config field to inject custom parameters
into the migration connection DSN.

**Example values:**

```ini
# Enable multiple statements per file and configure cluster settings
x-multi-statement=true
x-cluster-name=distributed_cluster
x-migrations-table-engine=ReplicatedMergeTree
```

## Observability

`xclick` exposes ClickHouse connection pool metrics through Prometheus.

<!-- @formatter:off -->
```go
pool, err := clickhouse.NewPool(
	clickhouse.WithConfig(cfg),
	clickhouse.WithClientID("analytics-service"),
	clickhouse.WithMetricsNamespace("analytics"),
)
if err != nil {
	log.Fatalf("failed to initialize observed ClickHouse pool: %v", err)
}
defer pool.Close()
```
<!-- @formatter:on -->


The following metric labels are added automatically:

| Label       | Description                                                        |
|-------------|--------------------------------------------------------------------|
| `client_id` | Generated client identifier or configured ID with a unique suffix. |
| `db`        | Database name from the config.                                     |
| `shard_id`  | Shard ID from the config.                                          |

## Error handling

`xclick` provides normalized ClickHouse error codes through `ConvertError`, so application code does not need to deal
with raw driver errors directly.

<!-- @formatter:off -->
```go
err := pool.QueryRow(ctx, "SELECT name FROM users WHERE id = ?", userID).Scan(&name)
if err != nil {
	chErr := clickhouse.ConvertError(err)

	if chErr.Code() == clickhouse.ErrNoRowsClickhouse {
		// handle missing row
		return nil
	}

	return chErr
}
```
<!-- @formatter:on -->


Supported error categories include connection errors, no rows, and unknown errors.

## Configuration

`Config` can be created directly as a Go struct.

It also includes `envconfig` tags, so you can populate it from environment variables using your preferred configuration
layer.

### Config Struct

<!-- @formatter:off -->
```go
cfg := &clickhouse.Config{
    Hosts:    "127.0.0.1:9000", // required
    User:     "user",           // required
    Password: "password",       // required
    DB:       "mydb",           // required

    MaxOpenConns:    32,
    MaxIdleConns:    8,
    ConnMaxLifetime: time.Hour,
    DialTimeout:     10 * time.Second,
    ReadTimeout:     10 * time.Second,

    MigrateEnabled: true,
}
```
<!-- @formatter:on -->

### Environment Variables

| Variable | Required | Default | Description |
| ---------------------------------------- | :------: | ---------- | ------------------------------------------------------------------- |
| `CLICKHOUSE_HOSTS` | ✓ | — | Comma-separated list of hosts, for example `host1:9000,host2:9000`. |
| `CLICKHOUSE_USER` | ✓ | — | Username. |
| `CLICKHOUSE_PASSWORD` | ✓ | — | Password. |
| `CLICKHOUSE_DB` | ✓ | — | Database name. |
| `CLICKHOUSE_SHARD_ID` | | `0` | Shard ID exposed in metrics. |
| `CLICKHOUSE_MAX_OPEN_CONNS` | | `32` | Maximum open connections. |
| `CLICKHOUSE_MAX_IDLE_CONNS` | | `8` | Maximum idle connections. |
| `CLICKHOUSE_CONN_MAX_LIFETIME` | | `1h` | Maximum connection lifetime. |
| `CLICKHOUSE_DIAL_TIMEOUT` | | `10s` | Connection dial timeout. |
| `CLICKHOUSE_READ_TIMEOUT` | | `10s` | Read timeout. |
| `CLICKHOUSE_CONN_OPEN_STRATEGY` | | `in_order` | Connection open strategy: `in_order`, `round_robin`, or `random`. |
| `CLICKHOUSE_BLOCK_BUFFER_SIZE` | | `2` | Block buffer size. |
| `CLICKHOUSE_MAX_COMPRESSION_BUFFER` | | `10 MiB` | Maximum compression buffer size. |
| `CLICKHOUSE_HTTP_HEADERS` | | — | Additional HTTP headers. |
| `CLICKHOUSE_HTTP_URL_PATH` | | — | Additional URL path for HTTP requests. |
| `CLICKHOUSE_SETTINGS` | | — | Additional ClickHouse settings. |
| `CLICKHOUSE_DEBUG` | | `false` | Enable debug logging. |
| `CLICKHOUSE_FREE_BUFFER_ON_CONN_RELEASE` | | `false` | Free memory buffer after each query. |
| `CLICKHOUSE_INSECURE_SKIP_VERIFY` | | `false` | Skip TLS certificate verification. |
| `CLICKHOUSE_MIGRATE_ENABLED` | | `false` | Run migrations on pool startup. |
| `CLICKHOUSE_MIGRATE_ARGS` | | — | Extra connection string args for migrations. |

## License

This project is licensed under the [MIT License](LICENSE).