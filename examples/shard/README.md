# Sharding

This example demonstrates direct routing across ClickHouse shards:

* Route user IDs to logical shards
* Execute writes on the resolved shard

The topology contains two shards:

```text
[0, 100)   -> shard-a
[100, 200) -> shard-b
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
Shard A: localhost:56431
Shard B: localhost:56432
```

## Configuration

The example connects to the following DSNs by default:

```text
Shard A: clickhouse://clickhouse:clickhouse@localhost:56431/default?dial_timeout=5s
Shard B: clickhouse://clickhouse:clickhouse@localhost:56432/default?dial_timeout=5s
```

Set `XCH_SHARD_A_DATABASE_URL` or `XCH_SHARD_B_DATABASE_URL` to use different endpoints.

## Run

From this directory:

```shell
go run .
```

Or from the repository root:

```shell
go run ./examples/shard
```

## Expected output

```text
range routing:
- user_id=42 shard=shard-a pool=shard-a
- user_id=142 shard=shard-b pool=shard-b
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
