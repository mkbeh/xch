# Observability

This example demonstrates OpenTelemetry metrics for an `xch` connection pool:

* Configure an OpenTelemetry metrics exporter
* Attach metrics to a ClickHouse connection pool
* Generate concurrent pool activity
* Export the resulting pool metrics to the console

## Local setup

From the `examples` directory, start ClickHouse:

```bash
docker compose up -d
```

ClickHouse is available at:

```text
Native: localhost:9000
HTTP:   localhost:8123
Web UI: http://localhost:8123/play
```

## Configuration

The example connects to the following DSN by default:

```text
clickhouse://clickhouse:clickhouse@localhost:9000/default?dial_timeout=5s
```

Set `XCH_DATABASE_URL` to use a different ClickHouse instance:

```bash
export XCH_DATABASE_URL='clickhouse://user:password@localhost:9000/default'
```

## Run

From this directory:

```shell
go run .
```

Or from the repository root:

```shell
go run ./examples/observability
```

## Expected output

The example exports an OpenTelemetry JSON snapshot containing the client pool metrics:

```text
xch.client.connection.open
xch.client.connection.idle
xch.client.connection.max_open
xch.client.connection.max_idle
```

For the default two-connection pool, the final snapshot reports two open and two idle connections after the workload
completes.

## Cleanup

From the `examples` directory, stop ClickHouse:

```bash
docker compose down
```

To also remove the ClickHouse data volume:

```bash
docker compose down -v
```
