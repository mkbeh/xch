# Geographic sharding

This example demonstrates region-based routing across ClickHouse shards:

* Define shard regions using metadata
* Build a custom resolver from shard labels
* Route reads and writes by tenant region
* Handle regions without a configured shard

The topology contains two shards:

```text
eu -> shard-eu
us -> shard-us
```

## Local setup

From this directory, start both ClickHouse shards:

```shell
docker compose up -d
```

Apply the schema to both shards:

```shell
docker compose exec -T clickhouse-shard-a \
  clickhouse-client --user clickhouse --password clickhouse --multiquery \
  < sql/schema.sql

docker compose exec -T clickhouse-shard-b \
  clickhouse-client --user clickhouse --password clickhouse --multiquery \
  < sql/schema.sql
```

The shards are available at:

```text
EU shard: localhost:57431
US shard: localhost:57432
```

## Configuration

Set `XCH_SHARD_EU_DATABASE_URL` or `XCH_SHARD_US_DATABASE_URL` to use different endpoints.

## Run

From this directory:

```shell
go run .
```

Or from the repository root:

```shell
go run ./examples/shard_geo
```

## Expected output

```text
geo routing:
- tenant=tenant-42 region=eu shard=shard-eu name=Alice
- tenant=tenant-77 region=us shard=shard-us name=Bob
unsupported region: no shard
```

## Cleanup

Stop ClickHouse:

```shell
docker compose down
```

To also remove both data volumes:

```shell
docker compose down -v
```
