# xclick

Go client for ClickHouse with migrations, query builder, and observability.

Built on top of [clickhouse-go](https://github.com/ClickHouse/clickhouse-go).

## Features

- **Query builder** — [squirrel](https://github.com/Masterminds/squirrel) with `$`-placeholder support out of the box
- **Migrations** — via [golang-migrate](https://github.com/golang-migrate/migrate) using `embed.FS`
- **Observability** — connection pool metrics exposed via Prometheus

## Installation

```bash
go get github.com/mkbeh/xclick
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
)

func main() {
	cfg := &clickhouse.Config{
		Hosts:    "127.0.0.1:9000",
		User:     "user",
		Password: "password",
		DB:       "mydb",
	}

	pool, err := clickhouse.NewPool(
		clickhouse.WithConfig(cfg),
		clickhouse.WithClientID("my-service"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	var result string
	if err := pool.QueryRow(context.Background(), "SELECT 'hello world'").Scan(&result); err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
}

```

More examples in [examples/](https://github.com/mkbeh/xclick/tree/main/examples).

## Migrations

Create an `embed.go` file in your migrations directory:

```go
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

Pass `FS` via `WithMigrations`:

```go
pool, err := clickhouse.NewPool(
    clickhouse.WithConfig(cfg),
    clickhouse.WithMigrations(migrations.FS),
)
```

Migrations are applied automatically on pool startup when `MigrateEnabled: true` is set in the config.

### Additional migration connection args

Set via `CLICKHOUSE_MIGRATE_ARGS` or the `MigrateArgs` config field:

```
x-multi-statement=true
x-cluster-name=distributed_cluster
x-migrations-table-engine=ReplicatedMergeTree
```

## Configuration

All parameters can be set via environment variables or directly in the `Config` struct.

| ENV | Required | Default | Description |
|-----|:--------:|:-------:|-------------|
| `CLICKHOUSE_HOSTS` | ✓ | — | Comma-separated list of hosts: `host1:9000,host2:9000` |
| `CLICKHOUSE_USER` | ✓ | — | Username |
| `CLICKHOUSE_PASSWORD` | ✓ | — | Password |
| `CLICKHOUSE_DB` | ✓ | — | Database name |
| `CLICKHOUSE_SHARD_ID` | | `0` | Shard identifier |
| `CLICKHOUSE_MAX_OPEN_CONNS` | | `32` | Max open connections |
| `CLICKHOUSE_MAX_IDLE_CONNS` | | `8` | Max idle connections |
| `CLICKHOUSE_CONN_MAX_LIFETIME` | | `1h` | Max connection lifetime |
| `CLICKHOUSE_DIAL_TIMEOUT` | | `10s` | Connection dial timeout |
| `CLICKHOUSE_READ_TIMEOUT` | | `10s` | Read timeout |
| `CLICKHOUSE_CONN_OPEN_STRATEGY` | | `in_order` | Strategy: `in_order`, `round_robin`, `random` |
| `CLICKHOUSE_BLOCK_BUFFER_SIZE` | | `2` | Block buffer size |
| `CLICKHOUSE_MAX_COMPRESSION_BUFFER` | | `10 MiB` | Max compression buffer size |
| `CLICKHOUSE_HTTP_HEADERS` | | — | Additional HTTP headers |
| `CLICKHOUSE_HTTP_URL_PATH` | | — | Additional URL path for HTTP requests |
| `CLICKHOUSE_DEBUG` | | `false` | Enable debug logging |
| `CLICKHOUSE_FREE_BUFFER_ON_CONN_RELEASE` | | `false` | Free memory buffer after each query |
| `CLICKHOUSE_INSECURE_SKIP_VERIFY` | | `false` | Skip TLS certificate verification |
| `CLICKHOUSE_MIGRATE_ENABLED` | | `false` | Run migrations on startup |
| `CLICKHOUSE_MIGRATE_ARGS` | | — | Extra connection string args for migrations |

## Client options

```go
clickhouse.WithConfig(cfg)           // connection config
clickhouse.WithClientID("name")      // client identifier, used in metrics labels
clickhouse.WithMigrations(fs)        // embed.FS containing SQL migration files
clickhouse.WithLogger(logger)        // custom slog.Logger
clickhouse.WithTLS(tlsCfg)           // TLS configuration
clickhouse.WithCompression(c)        // compression settings
clickhouse.WithHTTPProxy(proxyURL)   // HTTP proxy
clickhouse.WithMetricsNamespace(ns)  // Prometheus metrics namespace
```

## License

[MIT](LICENSE)