# Basic usage

This example demonstrates the core `xch` API:

* Create and verify a ClickHouse connection pool
* Perform batched inserts
* Read single records into structs
* Handle missing records
* Load result sets into slices
* Distinguish ClickHouse exceptions from connection errors

## Local setup

From the `examples` directory, start ClickHouse:

```bash
docker compose up -d
```

Apply the schema:

```bash
docker compose exec -T clickhouse \
  clickhouse-client \
  --user clickhouse \
  --password clickhouse \
  --multiquery \
  < basic/sql/schema.sql
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
go run ./examples/basic
```

## Expected output

```text
missing user: not found
pool: basic-example
selected user: 1 Alice <alice@example.com> active=true
active users:
- 1 Alice <alice@example.com>
```

## Cleanup

From the `examples` directory, remove the example database:

```bash
docker compose exec -T clickhouse \
  clickhouse-client \
  --user clickhouse \
  --password clickhouse \
  --query 'DROP DATABASE IF EXISTS xch_basic_example'
```

Stop ClickHouse:

```bash
docker compose down
```

To also remove the ClickHouse data volume:

```bash
docker compose down -v
```
